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

// LLMExtractor defines the interface for LLM-based memory processing.
type LLMExtractor interface {
	Extract(content string) (*Observation, error)
	Synthesize(memories []string) (string, error)
	Curate(query string, candidates []QueryResult) ([]CuratedResult, error)
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
func (c *ClaudeCLIExtractor) Extract(content string) (*Observation, error) {
	prompt := fmt.Sprintf(`Analyze this memory entry and extract structured knowledge. Return ONLY valid JSON matching this schema:
{"type":"<correction|pattern|decision|discovery>","concepts":["<concept1>","<concept2>"],"principle":"<actionable rule in imperative form>","anti_pattern":"<what NOT to do>","rationale":"<why this matters>"}

Memory entry:
%s`, content)

	output, err := c.runClaude(prompt)
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
func (c *ClaudeCLIExtractor) Synthesize(memories []string) (string, error) {
	var sb strings.Builder
	for i, mem := range memories {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, mem))
	}

	prompt := fmt.Sprintf(`Given these related memories, produce a single actionable principle suitable for a developer guidelines document (like CLAUDE.md). Return ONLY the principle text, no JSON, no markdown formatting.

Related memories:
%s`, sb.String())

	output, err := c.runClaude(prompt)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// Curate selects the most relevant memory results for a query with relevance explanations.
func (c *ClaudeCLIExtractor) Curate(query string, candidates []QueryResult) ([]CuratedResult, error) {
	var sb strings.Builder
	for i, cand := range candidates {
		sb.WriteString(fmt.Sprintf("%d. [score=%.2f] %s\n", i+1, cand.Score, cand.Content))
	}

	prompt := fmt.Sprintf(`User's current request: %s
Here are %d memory candidates:
%s
Select the 5-7 most relevant results for the user's request. Return ONLY a JSON array of objects:
[{"content":"<exact content>","relevance":"<why this is relevant>","memory_type":"<type>"}]`, query, len(candidates), sb.String())

	output, err := c.runClaude(prompt)
	if err != nil {
		return nil, err
	}

	var results []CuratedResult
	if err := json.Unmarshal(output, &results); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response as CuratedResult array: %w", err)
	}

	return results, nil
}

// runClaude executes the claude CLI and returns the output.
func (c *ClaudeCLIExtractor) runClaude(prompt string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
	defer cancel()

	output, err := c.CommandRunner(ctx, "claude", "--print", "--model", c.Model, "-p", prompt)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrLLMUnavailable, err)
	}

	return output, nil
}
