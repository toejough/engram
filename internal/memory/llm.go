package memory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Exported constants.
const (
	IngestAdd    IngestAction = "ADD"
	IngestDelete IngestAction = "DELETE"
	IngestNoop   IngestAction = "NOOP"
	IngestUpdate IngestAction = "UPDATE"
)

// Exported variables.
var (
	ErrLLMUnavailable = errors.New("LLM extractor unavailable")
)

// ClaudeCLIExtractor implements LLMExtractor using the claude CLI tool.
type ClaudeCLIExtractor struct {
	Model         string
	Timeout       time.Duration
	CommandRunner func(ctx context.Context, name string, args ...string) ([]byte, error)
}

// NewClaudeCLIExtractor creates a ClaudeCLIExtractor with sensible defaults.
func NewClaudeCLIExtractor() *ClaudeCLIExtractor {
	return &ClaudeCLIExtractor{
		Model:         "haiku",
		Timeout:       30 * time.Second,
		CommandRunner: defaultCommandRunner,
	}
}

// AddRationale adds an explanation of why a rule or principle matters.
func (c *ClaudeCLIExtractor) AddRationale(ctx context.Context, content string) (string, error) {
	prompt := fmt.Sprintf(`Add a brief explanation of WHY this rule/principle matters to the end of it. Format: "Rule - explanation of why it matters"

Rule: %s

Return ONLY the enriched version with rationale, no JSON or markdown formatting.`, content)

	output, err := c.runClaude(ctx, prompt)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// CompileSkill generates skill content from a theme and related memories.
func (c *ClaudeCLIExtractor) CompileSkill(ctx context.Context, theme string, memories []string) (string, error) {
	var sb strings.Builder
	for i, mem := range memories {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, mem))
	}

	prompt := fmt.Sprintf(`Generate a Claude Code skill document for the theme "%s" based on these memory patterns:

%s

Create a comprehensive SKILL.md with these sections: Overview, When to Use, Quick Reference, Common Mistakes.

Return ONLY valid JSON with this exact structure:
{"description": "Use when <triggering conditions>", "body": "## Overview\n..."}

The "description" must:
- Start with "Use when"
- Be under 1024 characters
- Describe when to invoke this skill (third person, no "I"/"you"/"we")

The "body" must use markdown with the 4 section headers.`, theme, sb.String())

	output, err := c.runClaude(ctx, prompt)
	if err != nil {
		return "", err
	}

	return string(output), nil
}

// Curate selects the most relevant memory results for a query with relevance explanations.
func (c *ClaudeCLIExtractor) Curate(ctx context.Context, query string, candidates []QueryResult) ([]CuratedResult, error) {
	var sb strings.Builder
	for i, cand := range candidates {
		sb.WriteString(fmt.Sprintf("%d. [score=%.2f] %s\n", i+1, cand.Score, cand.Content))
	}

	prompt := fmt.Sprintf(`User's current request: %s
Here are %d memory candidates:
%s
Select the 5-7 most relevant results for the user's request. Return ONLY a JSON array of objects:
[{"content":"<exact content>","relevance":"<why this is relevant>","memory_type":"<type>"}]`, query, len(candidates), sb.String())

	output, err := c.runClaude(ctx, prompt)
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
func (c *ClaudeCLIExtractor) Decide(ctx context.Context, newContent string, existing []ExistingMemory) (*IngestDecision, error) {
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

	output, err := c.runClaude(ctx, prompt)
	if err != nil {
		return nil, err
	}

	var decision IngestDecision
	if err := json.Unmarshal(output, &decision); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response as IngestDecision: %w", err)
	}

	return &decision, nil
}

// Extract analyzes a memory entry and returns structured knowledge.
func (c *ClaudeCLIExtractor) Extract(ctx context.Context, content string) (*Observation, error) {
	prompt := "Analyze this memory entry and extract structured knowledge. Return ONLY valid JSON matching this schema:\n{\"type\":\"<correction|pattern|decision|discovery>\",\"concepts\":[\"<concept1>\",\"<concept2>\"],\"principle\":\"<actionable rule in imperative form>\",\"anti_pattern\":\"<what NOT to do>\",\"rationale\":\"<why this matters>\"}\n\nMemory entry:\n" + content

	output, err := c.runClaude(ctx, prompt)
	if err != nil {
		return nil, err
	}

	var obs Observation
	if err := json.Unmarshal(output, &obs); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response as Observation: %w", err)
	}

	return &obs, nil
}

// Filter evaluates candidates for relevance to a query (stub: degrades gracefully without claude CLI).
func (c *ClaudeCLIExtractor) Filter(_ context.Context, _ string, candidates []QueryResult) ([]FilterResult, error) {
	results := make([]FilterResult, 0, len(candidates))
	for _, cand := range candidates {
		results = append(results, FilterResult{
			MemoryID:       cand.ID,
			Content:        cand.Content,
			Relevant:       true,
			RelevanceScore: -1.0,
			MemoryType:     cand.MemoryType,
		})
	}

	return results, nil
}

// IsNarrowLearning determines if a learning is narrow/context-specific or universal.
// Returns true if the learning is narrow (e.g., references specific file paths, project names),
// false if universal (e.g., general development principles).
func (c *ClaudeCLIExtractor) IsNarrowLearning(ctx context.Context, learning string) (isNarrow bool, reason string, err error) {
	prompt := "Analyze this learning and determine if it is narrow/context-specific or universal. Return ONLY valid JSON matching this schema:\n{\"is_narrow\": <bool>, \"reason\": \"<explanation>\", \"confidence\": <float>}\n\nA learning is NARROW if it:\n- References specific file paths (e.g., \"src/config.yaml\", \"internal/foo.go\")\n- Mentions specific project names or codebases\n- Contains environment-specific details (e.g., \"production server at X\")\n- Describes one-off fixes or project-specific workarounds\n\nA learning is UNIVERSAL if it:\n- Describes general development principles (e.g., \"Always verify inputs\")\n- Provides broadly applicable patterns (e.g., \"Use TDD for all changes\")\n- Contains best practices that work across contexts\n- Teaches transferable skills or techniques\n\nLearning to analyze:\n" + learning

	output, err := c.runClaude(ctx, prompt)
	if err != nil {
		return false, "", err
	}

	var response struct {
		IsNarrow   bool    `json:"is_narrow"`
		Reason     string  `json:"reason"`
		Confidence float64 `json:"confidence"`
	}
	if err := json.Unmarshal(output, &response); err != nil {
		return false, "", fmt.Errorf("%w: failed to parse IsNarrowLearning response: %w", ErrLLMUnavailable, err)
	}

	return response.IsNarrow, response.Reason, nil
}

// PostEval scores how faithfully the agent followed memory guidance (FR-004).
func (c *ClaudeCLIExtractor) PostEval(ctx context.Context, memoryContent, queryText string) (*PostEvalResult, error) {
	prompt := fmt.Sprintf(`Given the surfaced memory and the query context:

Memory: %q
Query: %q

Did the agent's response align with the guidance in the surfaced memory?
Score 0.0 (completely ignored/contradicted) to 1.0 (fully followed).

Return ONLY valid JSON: {"faithfulness": <float>, "signal": "<positive|negative>"}`, memoryContent, queryText)

	output, err := c.runClaude(ctx, prompt)
	if err != nil {
		return nil, err
	}

	var result PostEvalResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("PostEval: parse response: %w", err)
	}

	return &result, nil
}

// Rewrite improves clarity and specificity of a memory entry.
func (c *ClaudeCLIExtractor) Rewrite(ctx context.Context, content string) (string, error) {
	prompt := fmt.Sprintf(`Rewrite this memory entry to improve clarity and specificity while preserving its meaning:

%s

Return ONLY the rewritten version, no JSON or markdown formatting.`, content)

	output, err := c.runClaude(ctx, prompt)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// Synthesize produces a single actionable principle from a cluster of related memories.
func (c *ClaudeCLIExtractor) Synthesize(ctx context.Context, memories []string) (string, error) {
	var sb strings.Builder
	for i, mem := range memories {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, mem))
	}

	prompt := "Given these related memories, produce a single actionable principle suitable for a developer guidelines document (like CLAUDE.md). Return ONLY the principle text, no JSON, no markdown formatting.\n\nRelated memories:\n" + sb.String()

	output, err := c.runClaude(ctx, prompt)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// runClaude executes the claude CLI and returns the output.
func (c *ClaudeCLIExtractor) runClaude(parentCtx context.Context, prompt string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(parentCtx, c.Timeout)
	defer cancel()

	output, err := c.CommandRunner(ctx, "claude", "--print", "--model", c.Model, "-p", prompt)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrLLMUnavailable, err)
	}

	return output, nil
}

// CuratedResult represents a single memory result that has been evaluated
// by an LLM for relevance to a specific query.
type CuratedResult struct {
	Content    string `json:"content"`
	Relevance  string `json:"relevance"`
	MemoryType string `json:"memory_type"`
}

// ExistingMemory represents a similar memory found during dedup checking.
type ExistingMemory struct {
	ID         int64   `json:"id"`
	Content    string  `json:"content"`
	Similarity float64 `json:"similarity"`
}

// IngestAction represents the action to take when ingesting a new memory.
type IngestAction string

// IngestDecision is the LLM's recommendation for how to handle a new memory
// relative to existing similar memories.
type IngestDecision struct {
	Action   IngestAction `json:"action"`
	TargetID int64        `json:"target_id"`
	Reason   string       `json:"reason"`
}

// LLMExtractor defines the interface for LLM-based memory processing.
type LLMExtractor interface {
	Extract(ctx context.Context, content string) (*Observation, error)
	Synthesize(ctx context.Context, memories []string) (string, error)
	Curate(ctx context.Context, query string, candidates []QueryResult) ([]CuratedResult, error)
	Filter(ctx context.Context, query string, candidates []QueryResult) ([]FilterResult, error)
	Decide(ctx context.Context, newContent string, existing []ExistingMemory) (*IngestDecision, error)
	Rewrite(ctx context.Context, content string) (string, error)                            // ISSUE-218: Improve clarity/specificity
	AddRationale(ctx context.Context, content string) (string, error)                       // ISSUE-218: Add explanation of why
	PostEval(ctx context.Context, memoryContent, queryText string) (*PostEvalResult, error) // FR-004: Post-interaction faithfulness scoring
}

// Observation represents structured knowledge extracted from a memory entry
// by an LLM.
type Observation struct {
	Type        string   `json:"type"` // "correction", "pattern", "decision", "discovery"
	Concepts    []string `json:"concepts"`
	Principle   string   `json:"principle"`    // actionable rule in imperative form
	AntiPattern string   `json:"anti_pattern"` // what NOT to do
	Rationale   string   `json:"rationale"`    // why this matters
}

// PostEvalResult is the LLM's assessment of how faithfully an agent followed memory guidance.
type PostEvalResult struct {
	Faithfulness float64 `json:"faithfulness"`
	Signal       string  `json:"signal"` // "positive" or "negative"
}

// SkillCompiler defines the interface for compiling memory clusters into skill content.
type SkillCompiler interface {
	CompileSkill(ctx context.Context, theme string, memories []string) (string, error)
	Synthesize(ctx context.Context, memories []string) (string, error) // TASK-3: Synthesize principle from memories
}

// SpecificityDetector defines the interface for detecting narrow/context-specific learnings.
type SpecificityDetector interface {
	IsNarrowLearning(ctx context.Context, learning string) (isNarrow bool, reason string, err error)
}

// defaultCommandRunner executes the claude CLI with the given arguments.
func defaultCommandRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.Output()
}
