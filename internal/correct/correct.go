// Package correct implements the Remember & Correct pipeline (ARCH-1).
// It detects correction patterns, enriches them via LLM, writes TOML files,
// and renders system reminder feedback.
package correct

import (
	"context"
	"fmt"

	"engram/internal/memory"
)

// PatternMatcher detects correction patterns in user messages (ARCH-2).
type PatternMatcher interface {
	Match(message string) *memory.PatternMatch
}

// Enricher enriches a pattern match into a structured memory (ARCH-3).
type Enricher interface {
	Enrich(ctx context.Context, message string, match *memory.PatternMatch) (*memory.Enriched, error)
}

// MemoryWriter writes an enriched memory to persistent storage (ARCH-4).
type MemoryWriter interface {
	Write(mem *memory.Enriched, dataDir string) (string, error)
}

// Renderer formats an enriched memory as a system reminder string (ARCH-5).
type Renderer interface {
	Render(mem *memory.Enriched, filePath string) string
}

// Corrector orchestrates the four-stage Remember & Correct pipeline.
type Corrector struct {
	matcher  PatternMatcher
	enricher Enricher
	writer   MemoryWriter
	renderer Renderer
	dataDir  string
}

// New creates a Corrector wired with all four pipeline stages.
func New(
	matcher PatternMatcher,
	enricher Enricher,
	writer MemoryWriter,
	renderer Renderer,
	dataDir string,
) *Corrector {
	return &Corrector{
		matcher:  matcher,
		enricher: enricher,
		writer:   writer,
		renderer: renderer,
		dataDir:  dataDir,
	}
}

// Run executes the correction pipeline for a single message.
// Returns a system reminder string, or empty string if no pattern matched.
func (c *Corrector) Run(ctx context.Context, message string) (string, error) {
	match := c.matcher.Match(message)
	if match == nil {
		return "", nil
	}

	enriched, err := c.enricher.Enrich(ctx, message, match)
	if err != nil {
		return "", fmt.Errorf("correct: enrich: %w", err)
	}

	filePath, err := c.writer.Write(enriched, c.dataDir)
	if err != nil {
		return "", fmt.Errorf("correct: write: %w", err)
	}

	reminder := c.renderer.Render(enriched, filePath)

	return reminder, nil
}
