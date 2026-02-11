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

// ErrLLMUnavailable is returned when the LLM extractor cannot be reached
// (e.g., claude CLI not found, timeout). Callers should fall back to
// non-LLM behavior.
var ErrLLMUnavailable = errors.New("LLM extractor unavailable")

// Observation represents structured knowledge extracted from a memory entry
// by an LLM.
type Observation struct {
	Type        string   `json:"type"`         // "correction", "pattern", "decision", "discovery"
	Concepts    []string `json:"concepts"`
	Principle   string   `json:"principle"`     // actionable rule in imperative form
	AntiPattern string   `json:"anti_pattern"`  // what NOT to do
	Rationale   string   `json:"rationale"`     // why this matters
}

// CuratedResult represents a single memory result that has been evaluated
// by an LLM for relevance to a specific query.
type CuratedResult struct {
	Content    string `json:"content"`
	Relevance  string `json:"relevance"`
	MemoryType string `json:"memory_type"`
}

// IngestAction represents the action to take when ingesting a new memory.
type IngestAction string

const (
	IngestAdd    IngestAction = "ADD"
	IngestUpdate IngestAction = "UPDATE"
	IngestDelete IngestAction = "DELETE"
	IngestNoop   IngestAction = "NOOP"
)

// ExistingMemory represents a similar memory found during dedup checking.
type ExistingMemory struct {
	ID         int64   `json:"id"`
	Content    string  `json:"content"`
	Similarity float64 `json:"similarity"`
}

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
	Decide(ctx context.Context, newContent string, existing []ExistingMemory) (*IngestDecision, error)
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

// defaultCommandRunner executes the claude CLI with the given arguments.
func defaultCommandRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.Output()
}

// Extract analyzes a memory entry and returns structured knowledge.
func (c *ClaudeCLIExtractor) Extract(ctx context.Context, content string) (*Observation, error) {
	prompt := fmt.Sprintf(`Analyze this memory entry and extract structured knowledge. Return ONLY valid JSON matching this schema:
{"type":"<correction|pattern|decision|discovery>","concepts":["<concept1>","<concept2>"],"principle":"<actionable rule in imperative form>","anti_pattern":"<what NOT to do>","rationale":"<why this matters>"}

Memory entry:
%s`, content)

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

// Synthesize produces a single actionable principle from a cluster of related memories.
func (c *ClaudeCLIExtractor) Synthesize(ctx context.Context, memories []string) (string, error) {
	var sb strings.Builder
	for i, mem := range memories {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, mem))
	}

	prompt := fmt.Sprintf(`Given these related memories, produce a single actionable principle suitable for a developer guidelines document (like CLAUDE.md). Return ONLY the principle text, no JSON, no markdown formatting.

Related memories:
%s`, sb.String())

	output, err := c.runClaude(ctx, prompt)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
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

// CompileSkill generates skill content from a theme and related memories.
func (c *ClaudeCLIExtractor) CompileSkill(ctx context.Context, theme string, memories []string) (string, error) {
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

	output, err := c.runClaude(ctx, prompt)
	if err != nil {
		return "", err
	}

	return string(output), nil
}

// IsNarrowLearning determines if a learning is narrow/context-specific or universal.
// Returns true if the learning is narrow (e.g., references specific file paths, project names),
// false if universal (e.g., general development principles).
func (c *ClaudeCLIExtractor) IsNarrowLearning(ctx context.Context, learning string) (isNarrow bool, reason string, err error) {
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
		return false, "", fmt.Errorf("%w: failed to parse IsNarrowLearning response: %v", ErrLLMUnavailable, err)
	}

	return response.IsNarrow, response.Reason, nil
}

// runClaude executes the claude CLI and returns the output.
func (c *ClaudeCLIExtractor) runClaude(parentCtx context.Context, prompt string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(parentCtx, c.Timeout)
	defer cancel()

	output, err := c.CommandRunner(ctx, "claude", "--print", "--model", c.Model, "-p", prompt)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrLLMUnavailable, err)
	}

	return output, nil
}
