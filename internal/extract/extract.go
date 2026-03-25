// Package extract extracts candidate learnings from session transcripts via the Anthropic API.
package extract

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"engram/internal/memory"
)

// Exported variables.
var (
	// ErrNilResponse is returned when the HTTP client returns a nil response without error.
	ErrNilResponse = errors.New("extract: calling Anthropic API: nil response")
	// ErrNoToken is returned when no API token is configured.
	ErrNoToken = errors.New("extract: no API token configured")
)

// HTTPDoer is the interface for making HTTP requests. Wire http.DefaultClient in production.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// LLMExtractor uses the Anthropic API to extract candidate learnings from session transcripts.
type LLMExtractor struct {
	token  string
	client HTTPDoer
}

// New creates an LLMExtractor. Pass http.DefaultClient as client in production.
func New(token string, client HTTPDoer) *LLMExtractor {
	return &LLMExtractor{
		token:  token,
		client: client,
	}
}

// Extract extracts candidate learnings from a session transcript via the Anthropic API.
// Returns ErrNoToken if no API token is configured.
func (e *LLMExtractor) Extract(
	ctx context.Context,
	transcript string,
) ([]memory.CandidateLearning, error) {
	if e.token == "" {
		return nil, ErrNoToken
	}

	learnings, err := e.callLLM(ctx, transcript)
	if err != nil {
		return nil, fmt.Errorf("extraction: %w", err)
	}

	return learnings, nil
}

func (e *LLMExtractor) callLLM(
	ctx context.Context,
	transcript string,
) ([]memory.CandidateLearning, error) {
	resp, err := e.sendRequest(ctx, transcript)
	if err != nil {
		return nil, err
	}

	if resp == nil {
		return nil, ErrNilResponse
	}

	defer func() { _ = resp.Body.Close() }()

	return parseLLMResponse(resp)
}

func (e *LLMExtractor) sendRequest(ctx context.Context, transcript string) (*http.Response, error) {
	reqBody := anthropicRequest{
		Model:     anthropicModel,
		MaxTokens: maxResponseTokens,
		System:    systemPrompt(),
		Messages: []anthropicMessage{
			{
				Role:    "user",
				Content: transcript,
			},
		},
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling request body: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		anthropicAPIURL,
		bytes.NewReader(reqBytes),
	)
	if err != nil {
		return nil, fmt.Errorf("creating HTTP request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+e.token)
	req.Header.Set("Anthropic-Version", anthropicVersion)
	req.Header.Set("Anthropic-Beta", "oauth-2025-04-20")
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling Anthropic API: %w", err)
	}

	return resp, nil
}

// unexported constants.
const (
	anthropicAPIURL   = "https://api.anthropic.com/v1/messages"
	anthropicModel    = "claude-haiku-4-5-20251001"
	anthropicVersion  = "2023-06-01"
	maxResponseTokens = 2048
)

// unexported variables.
var (
	errEmptyAPIResponse = errors.New("API response contained no content blocks")
)

// anthropicContentBlock is a content block in an Anthropic API response.
type anthropicContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// anthropicMessage is a single message in the Anthropic messages API.
type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// anthropicRequest is the request body for the Anthropic messages API.
//
//nolint:tagliatelle // Anthropic API requires snake_case JSON field names.
type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system"`
	Messages  []anthropicMessage `json:"messages"`
}

// anthropicResponse is the response body from the Anthropic messages API.
type anthropicResponse struct {
	Content []anthropicContentBlock `json:"content"`
}

// llmCandidateLearningJSON is the JSON structure the LLM is instructed to return per item.
//
//nolint:tagliatelle // LLM prompt specifies snake_case JSON field names.
type llmCandidateLearningJSON struct {
	Tier             string   `json:"tier"`
	Title            string   `json:"title"`
	Content          string   `json:"content"`
	ObservationType  string   `json:"observation_type"`
	Concepts         []string `json:"concepts"`
	Keywords         []string `json:"keywords"`
	Principle        string   `json:"principle"`
	AntiPattern      string   `json:"anti_pattern"`
	Rationale        string   `json:"rationale"`
	FilenameSummary  string   `json:"filename_summary"`
	Generalizability int      `json:"generalizability"`
}

func parseLLMResponse(resp *http.Response) ([]memory.CandidateLearning, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	var apiResp anthropicResponse

	err = json.Unmarshal(body, &apiResp)
	if err != nil {
		return nil, fmt.Errorf("parsing API response JSON: %w", err)
	}

	if len(apiResp.Content) == 0 {
		return nil, errEmptyAPIResponse
	}

	llmText := stripMarkdownFence(apiResp.Content[0].Text)

	var llmItems []llmCandidateLearningJSON

	err = json.Unmarshal([]byte(llmText), &llmItems)
	if err != nil {
		return nil, fmt.Errorf("parsing LLM JSON output: %w", err)
	}

	learnings := make([]memory.CandidateLearning, 0, len(llmItems))

	for _, item := range llmItems {
		learnings = append(learnings, memory.CandidateLearning{
			Tier:             item.Tier,
			Title:            item.Title,
			Content:          item.Content,
			ObservationType:  item.ObservationType,
			Concepts:         item.Concepts,
			Keywords:         item.Keywords,
			Principle:        item.Principle,
			AntiPattern:      item.AntiPattern,
			Rationale:        item.Rationale,
			FilenameSummary:  item.FilenameSummary,
			Generalizability: item.Generalizability,
		})
	}

	return learnings, nil
}

// stripMarkdownFence removes markdown code fences (```json ... ```) that LLMs
// sometimes wrap around JSON output despite being told not to.
func stripMarkdownFence(text string) string {
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "```") {
		return text
	}

	// Remove opening fence (```json or ```)
	firstNewline := strings.Index(trimmed, "\n")
	if firstNewline < 0 {
		return text
	}

	trimmed = trimmed[firstNewline+1:]

	// Remove closing fence
	if idx := strings.LastIndex(trimmed, "```"); idx >= 0 {
		trimmed = trimmed[:idx]
	}

	return strings.TrimSpace(trimmed)
}

// systemPrompt returns the system prompt instructing the LLM to extract candidate learnings
// from session transcripts with a quality gate.
func systemPrompt() string {
	return strings.TrimSpace(`
You are a learning extraction assistant. Given a session transcript between a user and an AI assistant,
extract high-value learnings and return ONLY a JSON array — no markdown, no explanation.

QUALITY GATE — reject the following:
- mechanical patterns (e.g., "always add t.Parallel()")
- vague generalizations (e.g., "use good practices")
- overly narrow observations tied to a single insignificant detail
- ephemeral context: task/validation status updates (e.g., "S6 is validated," "step 3 is complete"),
  debugging observations about specific data or state (e.g., "pipeline produced flat faces,"
  "normals are inverted on mesh B"), project-specific variable/file names without a generalizable
  principle. Litmus test: would a developer on a different task in a different project, weeks from
  now, benefit from knowing this? If probably not, reject it or score it low.
- one-time tasks or completed actions (e.g., "remove the --data-dir flag," "file an issue about X,"
  "clean up the hooks"). If the user said "do X" and X has a completion state, it is a task, not a
  reusable principle. Do not extract it.
- common knowledge any competent developer already knows (e.g., "test both branches of a boolean,"
  "handle errors," "use descriptive names"). If the principle would appear in an introductory
  course or tutorial, the model already knows it — skip it.

EXTRACT only high-signal learnings such as:
- missed corrections the AI should have caught
- architectural decisions and their rationale
- discovered constraints that affect design choices
- working solutions to previously unsolved problems
- implicit preferences the user expressed through their corrections

TIER CLASSIFICATION — classify each learning into exactly one tier:
- A = explicit instruction: the user directly told the AI to do or not do something
  (e.g., "always use targ", "never run go test directly")
- B = teachable correction: the user corrected the AI in a way that generalizes
  (e.g., fixing an approach the AI should learn from)
- C = contextual fact: a discovered constraint, architectural decision, or
  environmental fact (e.g., "this project uses SQLite")

ANTI-PATTERN GATING — populate the anti_pattern field based on tier:
- Tier A: ALWAYS generate anti_pattern (the inverse of the explicit instruction)
- Tier B: generate anti_pattern ONLY when the correction is generalizable (use your judgment)
- Tier C: ALWAYS leave anti_pattern as empty string ""

Return a JSON array of objects, each with these exact fields:
[
  {
    "tier": "A, B, or C",
    "title": "Short title (5-10 words) summarizing the learning",
    "content": "The full learning verbatim or paraphrased from transcript",
    "observation_type": "One of: correction, architectural, constraint, solution, preference",
    "concepts": ["key", "concepts"],
    "keywords": ["searchable", "keywords"],
    "principle": "The positive rule or principle to follow",
    "anti_pattern": "The negative pattern or mistake to avoid (tier-gated, see rules above)",
    "rationale": "Why this principle matters",
    "filename_summary": "three to five words",
    "generalizability": "Integer 1-5: 1=only this session, 2=this project/narrow,
      3=across this project, 4=across similar projects, 5=universal"
  }
]

If no high-value learnings are found, return an empty JSON array: []`)
}
