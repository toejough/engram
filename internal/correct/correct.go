// Package correct implements the Remember & Correct pipeline (ARCH-1).
// Three-stage pipeline: Classifier → Writer → Renderer.
package correct

import (
	"context"
	"fmt"

	"engram/internal/memory"
)

// Classifier classifies a message and returns a ClassifiedMemory or nil (ARCH-2).
type Classifier interface {
	Classify(
		ctx context.Context,
		message, transcriptContext string,
	) (*memory.ClassifiedMemory, error)
}

// Corrector orchestrates the three-stage Remember & Correct pipeline.
type Corrector struct {
	classifier  Classifier
	writer      MemoryWriter
	renderer    Renderer
	dataDir     string
	projectSlug string
}

// New creates a Corrector wired with all three pipeline stages.
func New(
	classifier Classifier,
	writer MemoryWriter,
	renderer Renderer,
	dataDir string,
) *Corrector {
	return &Corrector{
		classifier: classifier,
		writer:     writer,
		renderer:   renderer,
		dataDir:    dataDir,
	}
}

// SetProjectSlug sets the originating project slug for new memories.
func (c *Corrector) SetProjectSlug(slug string) {
	c.projectSlug = slug
}

// Run executes the correction pipeline for a single message.
// Returns a system reminder string, or empty string if no signal detected.
func (c *Corrector) Run(
	ctx context.Context,
	message, transcriptContext string,
) (string, error) {
	classified, err := c.classifier.Classify(ctx, message, transcriptContext)
	if err != nil {
		return "", fmt.Errorf("correct: classify: %w", err)
	}

	if classified == nil {
		return "", nil
	}

	const minGeneralizability = 2

	if classified.Generalizability > 0 && classified.Generalizability < minGeneralizability {
		return "", nil
	}

	enriched := classified.ToEnriched()
	enriched.ProjectSlug = c.projectSlug

	filePath, err := c.writer.Write(enriched, c.dataDir)
	if err != nil {
		return "", fmt.Errorf("correct: write: %w", err)
	}

	reminder := c.renderer.Render(classified, filePath)

	return reminder, nil
}

// MemoryWriter writes an enriched memory to persistent storage (ARCH-4).
type MemoryWriter interface {
	Write(mem *memory.Enriched, dataDir string) (string, error)
}

// Renderer formats a classified memory as a system reminder string (ARCH-5).
type Renderer interface {
	Render(mem *memory.ClassifiedMemory, filePath string) string
}
