package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// LLMClient is the union interface for all LLM functionality.
type LLMClient interface {
	LLMExtractor
	SkillCompiler
	SpecificityDetector
}

type DirectAPIExtractor struct {
	token   string
	model   string
	baseURL string
	timeout time.Duration
	client  *http.Client
}

type DirectAPIOption func(*DirectAPIExtractor)

func WithBaseURL(url string) DirectAPIOption {
	return func(d *DirectAPIExtractor) { d.baseURL = url }
}

func WithModel(model string) DirectAPIOption {
	return func(d *DirectAPIExtractor) { d.model = model }
}

func WithTimeout(timeout time.Duration) DirectAPIOption {
	return func(d *DirectAPIExtractor) { d.timeout = timeout }
}

func NewDirectAPIExtractor(token string, opts ...DirectAPIOption) *DirectAPIExtractor {
	d := &DirectAPIExtractor{
		token:   token,
		model:   "claude-haiku-4-5-20251001",
		baseURL: "https://api.anthropic.com",
		timeout: 30 * time.Second,
		client:  &http.Client{},
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

// callAPI sends a prompt to the Anthropic API and returns the raw text response.
func (d *DirectAPIExtractor) callAPI(ctx context.Context, prompt string, maxTokens int) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	body := map[string]any{
		"model":      d.model,
		"max_tokens": maxTokens,
		"messages": []map[string]any{
			{"role": "user", "content": prompt},
		},
	}
	bodyBytes, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "POST", d.baseURL+"/v1/messages", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrLLMUnavailable, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+d.token)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("anthropic-beta", "oauth-2025-04-20")

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrLLMUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("%w: API returned %d", ErrLLMUnavailable, resp.StatusCode)
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("%w: failed to decode API response: %v", ErrLLMUnavailable, err)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("%w: API error: %s", ErrLLMUnavailable, result.Error.Message)
	}
	if len(result.Content) == 0 {
		return nil, fmt.Errorf("%w: empty response content", ErrLLMUnavailable)
	}

	return []byte(stripMarkdownFencing(result.Content[0].Text)), nil
}

// stripMarkdownFencing removes ```json ... ``` or ``` ... ``` wrapping
// that models sometimes add despite "Return ONLY valid JSON" prompts.
func stripMarkdownFencing(s string) string {
	trimmed := strings.TrimSpace(s)
	if !strings.HasPrefix(trimmed, "```") {
		return s
	}
	// Strip opening fence line (```json, ```, etc.)
	if idx := strings.Index(trimmed, "\n"); idx != -1 {
		trimmed = trimmed[idx+1:]
	}
	// Strip closing fence
	trimmed = strings.TrimSuffix(trimmed, "```")
	return strings.TrimSpace(trimmed)
}

// Extract analyzes a memory entry and returns structured knowledge.
func (d *DirectAPIExtractor) Extract(ctx context.Context, content string) (*Observation, error) {
	prompt := fmt.Sprintf(`Analyze this memory entry and extract structured knowledge. Return ONLY valid JSON matching this schema:
{"type":"<correction|pattern|decision|discovery>","concepts":["<concept1>","<concept2>"],"principle":"<actionable rule in imperative form>","anti_pattern":"<what NOT to do>","rationale":"<why this matters>"}

Memory entry:
%s`, content)

	output, err := d.callAPI(ctx, prompt, 256)
	if err != nil {
		return nil, err
	}

	var obs Observation
	if err := json.Unmarshal(output, &obs); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response as Observation: %w", err)
	}

	return &obs, nil
}

// Synthesize produces a single actionable principle from a cluster of related memories.
func (d *DirectAPIExtractor) Synthesize(ctx context.Context, memories []string) (string, error) {
	var sb strings.Builder
	for i, mem := range memories {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, mem))
	}

	prompt := fmt.Sprintf(`Given these related memories, produce a single actionable principle suitable for a developer guidelines document (like CLAUDE.md). Return ONLY the principle text, no JSON, no markdown formatting.

Related memories:
%s`, sb.String())

	output, err := d.callAPI(ctx, prompt, 512)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// Curate selects the most relevant memory results for a query with relevance explanations.
func (d *DirectAPIExtractor) Curate(ctx context.Context, query string, candidates []QueryResult) ([]CuratedResult, error) {
	var sb strings.Builder
	for i, cand := range candidates {
		sb.WriteString(fmt.Sprintf("%d. [score=%.2f] %s\n", i+1, cand.Score, cand.Content))
	}

	prompt := fmt.Sprintf(`User's current request: %s
Here are %d memory candidates:
%s
Select the 5-7 most relevant results for the user's request. Return ONLY a JSON array of objects:
[{"content":"<exact content>","relevance":"<why this is relevant>","memory_type":"<type>"}]`, query, len(candidates), sb.String())

	output, err := d.callAPI(ctx, prompt, 2048)
	if err != nil {
		return nil, err
	}

	var results []CuratedResult
	if err := json.Unmarshal(output, &results); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response as CuratedResult array: %w", err)
	}

	return results, nil
}

// Decide asks the LLM whether a new memory should be added, should update/replace
// an existing memory, makes an existing memory obsolete, or is a duplicate.
func (d *DirectAPIExtractor) Decide(ctx context.Context, newContent string, existing []ExistingMemory) (*IngestDecision, error) {
	var sb strings.Builder
	for i, mem := range existing {
		sb.WriteString(fmt.Sprintf("%d. [id=%d, similarity=%.2f] %q\n", i+1, mem.ID, mem.Similarity, mem.Content))
	}

	prompt := fmt.Sprintf(`A new memory is being stored:
%q

These similar memories already exist:
%s
Decide the best action. Return ONLY valid JSON:
{"action":"ADD|UPDATE|DELETE|NOOP","target_id":<id or 0>,"reason":"<why>"}

Rules:
- ADD: genuinely new knowledge, no close match
- UPDATE: new memory refines/corrects an existing one (target_id = which to update)
- DELETE: new memory makes an existing one obsolete (target_id = which to remove)
- NOOP: duplicate or less valuable than existing (target_id = which is better)`, newContent, sb.String())

	output, err := d.callAPI(ctx, prompt, 256)
	if err != nil {
		return nil, err
	}

	var decision IngestDecision
	if err := json.Unmarshal(output, &decision); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response as IngestDecision: %w", err)
	}

	return &decision, nil
}

// CompileSkill generates skill content from a theme and related memories.
func (d *DirectAPIExtractor) CompileSkill(ctx context.Context, theme string, memories []string) (string, error) {
	var sb strings.Builder
	for i, mem := range memories {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, mem))
	}

	prompt := fmt.Sprintf(`Generate a Claude Code skill document for the theme "%s" based on these memory patterns:

%s

Create a comprehensive SKILL.md content that:
- Explains the concept clearly
- Provides actionable guidance
- Includes specific examples or patterns from the memories
- Uses markdown formatting

Return ONLY the skill content (markdown), no JSON or wrappers.`, theme, sb.String())

	output, err := d.callAPI(ctx, prompt, 4096)
	if err != nil {
		return "", err
	}

	return string(output), nil
}

// IsNarrowLearning determines if a learning is narrow/context-specific or universal.
// Returns true if the learning is narrow (e.g., references specific file paths, project names),
// false if universal (e.g., general development principles).
func (d *DirectAPIExtractor) IsNarrowLearning(ctx context.Context, learning string) (isNarrow bool, reason string, err error) {
	prompt := fmt.Sprintf(`Analyze this learning and determine if it is narrow/context-specific or universal. Return ONLY valid JSON matching this schema:
{"is_narrow": <bool>, "reason": "<explanation>", "confidence": <float>}

A learning is NARROW if it:
- References specific file paths (e.g., "src/config.yaml", "internal/foo.go")
- Mentions specific project names or codebases
- Contains environment-specific details (e.g., "production server at X")
- Describes one-off fixes or project-specific workarounds

A learning is UNIVERSAL if it:
- Describes general development principles (e.g., "Always verify inputs")
- Provides broadly applicable patterns (e.g., "Use TDD for all changes")
- Contains best practices that work across contexts
- Teaches transferable skills or techniques

Learning to analyze:
%s`, learning)

	output, err := d.callAPI(ctx, prompt, 256)
	if err != nil {
		return false, "", err
	}

	var response struct {
		IsNarrow   bool    `json:"is_narrow"`
		Reason     string  `json:"reason"`
		Confidence float64 `json:"confidence"`
	}
	if err := json.Unmarshal(output, &response); err != nil {
		return false, "", fmt.Errorf("%w: failed to parse IsNarrowLearning response: %v", ErrLLMUnavailable, err)
	}

	return response.IsNarrow, response.Reason, nil
}

// Rewrite improves clarity and specificity of a memory entry.
func (d *DirectAPIExtractor) Rewrite(ctx context.Context, content string) (string, error) {
	prompt := fmt.Sprintf(`Rewrite this memory entry to improve clarity and specificity while preserving its meaning:

%s

Return ONLY the rewritten version, no JSON or markdown formatting.`, content)

	output, err := d.callAPI(ctx, prompt, 512)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// AddRationale adds an explanation of why a rule or principle matters.
func (d *DirectAPIExtractor) AddRationale(ctx context.Context, content string) (string, error) {
	prompt := fmt.Sprintf(`Add a brief explanation of WHY this rule/principle matters to the end of it. Format: "Rule - explanation of why it matters"

Rule: %s

Return ONLY the enriched version with rationale, no JSON or markdown formatting.`, content)

	output, err := d.callAPI(ctx, prompt, 512)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// LLMExtractorOption configures NewLLMExtractor.
type LLMExtractorOption func(*llmExtractorConfig)

type llmExtractorConfig struct {
	auth *KeychainAuth
}

// WithAuth configures a custom KeychainAuth for testing.
func WithAuth(auth *KeychainAuth) LLMExtractorOption {
	return func(c *llmExtractorConfig) { c.auth = auth }
}

// NewLLMExtractor creates the best available LLM client.
// Uses DirectAPIExtractor via Keychain token. Returns nil if auth is unavailable.
func NewLLMExtractor(opts ...LLMExtractorOption) LLMClient {
	cfg := &llmExtractorConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	auth := cfg.auth
	if auth == nil {
		auth = NewKeychainAuth()
	}

	token, err := auth.GetToken(context.Background())
	if err == nil {
		return NewDirectAPIExtractor(token)
	}

	return nil
}
