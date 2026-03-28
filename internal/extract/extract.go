// Package extract extracts candidate learnings from session transcripts via the Anthropic API.
package extract

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"engram/internal/anthropic"
	"engram/internal/memory"
)

// Exported variables.
var (
	// ErrNoToken is returned when no API token is configured.
	ErrNoToken = errors.New("extract: no API token configured")
)

// ExtractionGuidance is a learned directive to inject into the extraction prompt.
type ExtractionGuidance struct {
	Directive string
	Rationale string
}

// LLMExtractor uses the Anthropic API to extract candidate learnings from session transcripts.
type LLMExtractor struct {
	client   *anthropic.Client
	guidance []ExtractionGuidance
}

// New creates an LLMExtractor. Pass http.DefaultClient as client in production.
func New(token string, client anthropic.HTTPDoer, opts ...Option) *LLMExtractor {
	e := &LLMExtractor{
		client: anthropic.NewClient(token, client),
	}
	for _, opt := range opts {
		opt(e)
	}

	return e
}

// Extract extracts candidate learnings from a session transcript via the Anthropic API.
// Returns ErrNoToken if no API token is configured.
func (e *LLMExtractor) Extract(
	ctx context.Context,
	transcript string,
) ([]memory.CandidateLearning, error) {
	learnings, err := e.callLLM(ctx, transcript)
	if err != nil {
		if errors.Is(err, anthropic.ErrNoToken) {
			return nil, ErrNoToken
		}

		return nil, fmt.Errorf("extraction: %w", err)
	}

	return learnings, nil
}

func (e *LLMExtractor) callLLM(
	ctx context.Context,
	transcript string,
) ([]memory.CandidateLearning, error) {
	text, err := e.client.Call(
		ctx, anthropic.HaikuModel,
		SystemPromptWithGuidance(e.guidance),
		transcript, maxResponseTokens,
	)
	if err != nil {
		return nil, err
	}

	return parseLLMText(text)
}

// Option configures an LLMExtractor.
type Option func(*LLMExtractor)

// SystemPromptWithGuidance returns the extraction system prompt, optionally extended
// with learned extraction guidance directives derived from memory feedback.
func SystemPromptWithGuidance(guidance []ExtractionGuidance) string {
	base := strings.TrimSpace(extractionSystemPrompt)
	if len(guidance) == 0 {
		return base
	}

	var sb strings.Builder
	sb.WriteString(base)
	sb.WriteString("\n\n## Learned Extraction Guidance\n\n")
	sb.WriteString("Based on feedback from this user's memory corpus:\n\n")

	for _, item := range guidance {
		sb.WriteString("- ")
		sb.WriteString(item.Directive)
		sb.WriteString(" (")
		sb.WriteString(item.Rationale)
		sb.WriteString(")\n")
	}

	return sb.String()
}

// WithGuidance sets learned extraction guidance policies.
func WithGuidance(guidance []ExtractionGuidance) Option {
	return func(e *LLMExtractor) {
		e.guidance = guidance
	}
}

// unexported constants.
const (
	extractionSystemPrompt = `
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

GENERALIZE — before storing, restate each learning at its most transferable level:
- Strip project-specific details (file names, variable names, tool names) unless they ARE the point.
- Ask: "What is the underlying principle that makes this correct?" State that, not the specific instance.
- Example: "persist surfacing queries in irrelevant_queries field" → "capture diagnostic context at
  the point of observation for later analysis, not after the fact."
- If the generalized form is identical to an existing well-known principle, score generalizability
  lower or reject entirely.

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

KEYWORD QUALITY — keywords should match the SITUATION where this principle is needed, not just
the subject area. Bad: "git log", "boolean", "testing", "UI" — domain terms that match too broadly.
Good: "post-migration verification", "parallel-agent id collision", "algorithm-exposed controls" —
activity-level terms describing when someone would need this memory. Ask: "What would someone be
doing when they need this memory?" Use those activity-level terms.

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

If no high-value learnings are found, return an empty JSON array: []`
	maxResponseTokens = 2048
)

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

func parseLLMText(text string) ([]memory.CandidateLearning, error) {
	llmText := stripMarkdownFence(text)

	var llmItems []llmCandidateLearningJSON

	if err := json.Unmarshal([]byte(llmText), &llmItems); err != nil {
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

// systemPrompt returns the base extraction system prompt without learned guidance.
func systemPrompt() string {
	return SystemPromptWithGuidance(nil)
}
