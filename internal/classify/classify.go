// Package classify implements the unified classifier (ARCH-2).
// Two-stage detection: deterministic fast-path keywords, then LLM classifier.
package classify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"engram/internal/memory"
)

// Exported variables.
var (
	// ErrNoToken is returned when no API token is configured.
	ErrNoToken = errors.New("no API token configured")
)

// HTTPDoer is the interface for making HTTP requests.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// LLMClassifier uses fast-path keyword detection and the Anthropic API
// for unified classification and enrichment.
type LLMClassifier struct {
	token  string
	client HTTPDoer
}

// New creates an LLMClassifier.
func New(token string, client HTTPDoer) *LLMClassifier {
	return &LLMClassifier{
		token:  token,
		client: client,
	}
}

// Classify classifies a message, returning a ClassifiedMemory or nil if no signal.
func (c *LLMClassifier) Classify(
	ctx context.Context,
	message, transcriptContext string,
) (*memory.ClassifiedMemory, error) {
	isFastPath := containsFastPathKeyword(message)

	if c.token == "" {
		if isFastPath {
			return nil, ErrNoToken
		}
		// No token and no fast-path keyword: degrade gracefully
		return nil, nil //nolint:nilnil // no token + no keyword = no signal
	}

	result, err := c.callLLM(ctx, message, transcriptContext, isFastPath)
	if err != nil {
		return nil, fmt.Errorf("classify: %w", err)
	}

	return result, nil
}

func (c *LLMClassifier) callLLM(
	ctx context.Context,
	message, transcriptContext string,
	isFastPath bool,
) (*memory.ClassifiedMemory, error) {
	userContent := buildUserContent(message, transcriptContext, isFastPath)

	reqBody := anthropicRequest{
		Model:     anthropicModel,
		MaxTokens: maxResponseTokens,
		System:    systemPrompt(isFastPath),
		Messages: []anthropicMessage{
			{Role: "user", Content: userContent},
		},
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		anthropicAPIURL,
		bytes.NewReader(reqBytes),
	)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Anthropic-Version", anthropicVersion)
	req.Header.Set("Anthropic-Beta", "oauth-2025-04-20")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling Anthropic API: %w", err)
	}

	if resp == nil {
		return nil, errNilResponse
	}

	defer func() { _ = resp.Body.Close() }()

	return parseClassifyResponse(resp, isFastPath)
}

// unexported constants.
const (
	anthropicAPIURL   = "https://api.anthropic.com/v1/messages"
	anthropicModel    = "claude-haiku-4-5-20251001"
	anthropicVersion  = "2023-06-01"
	maxResponseTokens = 1024
)

// unexported variables.
var (
	errEmptyResponse = errors.New("API response contained no content blocks")
	errNilResponse   = errors.New("calling Anthropic API: nil response")
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

// llmClassifyJSON is the JSON structure the LLM returns.
//
//nolint:tagliatelle // LLM prompt specifies snake_case JSON field names.
type llmClassifyJSON struct {
	Tier            *string  `json:"tier"` // pointer to distinguish null from empty
	Title           string   `json:"title"`
	Content         string   `json:"content"`
	ObservationType string   `json:"observation_type"`
	Concepts        []string `json:"concepts"`
	Keywords        []string `json:"keywords"`
	Principle       string   `json:"principle"`
	AntiPattern     string   `json:"anti_pattern"`
	Rationale       string   `json:"rationale"`
	FilenameSummary string   `json:"filename_summary"`
}

func buildUserContent(message, transcriptContext string, isFastPath bool) string {
	var sb strings.Builder

	sb.WriteString("Message: ")
	sb.WriteString(message)

	if transcriptContext != "" {
		sb.WriteString("\n\nRecent transcript context:\n")
		sb.WriteString(transcriptContext)
	}

	if isFastPath {
		sb.WriteString(
			"\n\nNote: This message contains a fast-path keyword " +
				"(remember/always/never). " +
				"Classify as tier A and provide all structured fields.",
		)
	}

	return sb.String()
}

// containsFastPathKeyword checks for case-insensitive whole-word matches
// of "remember", "always", or "never" in the message.
func containsFastPathKeyword(message string) bool {
	lower := strings.ToLower(message)
	for _, kw := range []string{"remember", "always", "never"} {
		if containsWholeWord(lower, kw) {
			return true
		}
	}

	return false
}

// containsWholeWord checks if word appears as a whole word in text.
func containsWholeWord(text, word string) bool {
	idx := 0
	for {
		pos := strings.Index(text[idx:], word)
		if pos < 0 {
			return false
		}

		absPos := idx + pos
		start := absPos
		end := absPos + len(word)

		startOK := start == 0 || !isWordChar(text[start-1])
		endOK := end >= len(text) || !isWordChar(text[end])

		if startOK && endOK {
			return true
		}

		idx = absPos + 1
	}
}

func isWordChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}

func parseClassifyResponse(
	resp *http.Response,
	isFastPath bool,
) (*memory.ClassifiedMemory, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var apiResp anthropicResponse

	err = json.Unmarshal(body, &apiResp)
	if err != nil {
		return nil, fmt.Errorf("parsing API response: %w", err)
	}

	if len(apiResp.Content) == 0 {
		return nil, errEmptyResponse
	}

	llmText := stripMarkdownFence(apiResp.Content[0].Text)

	var llmData llmClassifyJSON

	err = json.Unmarshal([]byte(llmText), &llmData)
	if err != nil {
		return nil, fmt.Errorf("parsing LLM JSON: %w", err)
	}

	// null tier → no signal (nil result is intentional API — means no learning detected)
	if llmData.Tier == nil || *llmData.Tier == "" {
		return nil, nil //nolint:nilnil // nil result = no signal per ARCH-2
	}

	tier := *llmData.Tier

	// Override tier to A for fast-path messages
	if isFastPath {
		tier = "A"
	}

	now := time.Now()

	return &memory.ClassifiedMemory{
		Tier:            tier,
		Title:           llmData.Title,
		Content:         llmData.Content,
		ObservationType: llmData.ObservationType,
		Concepts:        llmData.Concepts,
		Keywords:        llmData.Keywords,
		Principle:       llmData.Principle,
		AntiPattern:     llmData.AntiPattern,
		Rationale:       llmData.Rationale,
		FilenameSummary: llmData.FilenameSummary,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

// stripMarkdownFence removes markdown code fences that LLMs sometimes wrap around JSON.
func stripMarkdownFence(text string) string {
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "```") {
		return text
	}

	firstNewline := strings.Index(trimmed, "\n")
	if firstNewline < 0 {
		return text
	}

	trimmed = trimmed[firstNewline+1:]

	if idx := strings.LastIndex(trimmed, "```"); idx >= 0 {
		trimmed = trimmed[:idx]
	}

	return strings.TrimSpace(trimmed)
}

// systemPrompt returns the system prompt for classification.
func systemPrompt(isFastPath bool) string {
	base := strings.TrimSpace(`
You are a memory classification and extraction assistant. Given a user message (and optional
transcript context), determine if the message contains a learning signal worth remembering.

Classify the message into one of these tiers:
- Tier A (explicit instruction): Direct commands like "remember X", "always do Y", "never do Z",
  or explicit standing instructions. Anti-pattern is REQUIRED.
- Tier B (teachable correction): Corrections, complaints, or feedback that imply a generalizable
  rule. Anti-pattern is included when the correction is generalizable.
- Tier C (contextual fact): Facts about the project, environment, or preferences that provide
  useful context. Anti-pattern is EMPTY.
- null: No learning signal detected. The message is casual conversation, a simple command,
  a question with no implicit preference, or ephemeral context that does not generalize
  (e.g., current task status, one-off session state, transient observations that apply only
  to this specific moment and would not be useful in a future session).

Return ONLY a JSON object — no markdown, no explanation:
{
  "tier": "A" | "B" | "C" | null,
  "title": "Short title (5-10 words)",
  "content": "The full original message verbatim",
  "observation_type": "category label",
  "concepts": ["key", "concepts"],
  "keywords": ["searchable", "keywords"],
  "principle": "The positive rule or principle",
  "anti_pattern": "The negative pattern to avoid (tier-gated)",
  "rationale": "Why this matters",
  "filename_summary": "three to five words"
}`)

	if isFastPath {
		base += "\n\nIMPORTANT: This message contains a fast-path keyword. Always classify as tier A."
	}

	return base
}
