// Package correct implements the Remember & Correct pipeline (ARCH-1).
// Three-stage pipeline: Classifier → Writer → Renderer.
package correct

import (
	"context"
	"fmt"

	"engram/internal/memory"
)

// ConsolidationActionType describes the consolidator's decision.
type ConsolidationActionType int

// ConsolidationActionType values.
const (
	// StoreAsIs means no cluster was found; store the memory normally.
	StoreAsIs ConsolidationActionType = iota
	// Consolidated means a cluster was found and merged into a generalized memory.
	Consolidated
)

// Classifier classifies a message and returns a ClassifiedMemory or nil (ARCH-2).
type Classifier interface {
	Classify(
		ctx context.Context,
		message, transcriptContext string,
	) (*memory.ClassifiedMemory, error)
}

// ConsolidationAction is the result of a consolidation check before storing a memory.
type ConsolidationAction struct {
	Type            ConsolidationActionType
	ConsolidatedMem *memory.MemoryRecord
}

// Corrector orchestrates the three-stage Remember & Correct pipeline.
type Corrector struct {
	classifier   Classifier
	writer       MemoryWriter
	renderer     Renderer
	dataDir      string
	projectSlug  string
	consolidator interface {
		BeforeStore(ctx context.Context, candidate *memory.MemoryRecord) (ConsolidationAction, error)
	}
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

	if c.consolidator != nil {
		record := enrichedToMemoryRecord(enriched)

		action, consErr := c.consolidator.BeforeStore(ctx, record)
		if consErr == nil && action.Type == Consolidated {
			consolidatedEnriched := memoryRecordToEnriched(action.ConsolidatedMem)
			consolidatedEnriched.ProjectSlug = c.projectSlug

			filePath, writeErr := c.writer.Write(consolidatedEnriched, c.dataDir)
			if writeErr != nil {
				return "", fmt.Errorf("correct: write consolidated: %w", writeErr)
			}

			return c.renderer.Render(classified, filePath), nil
		}
	}

	filePath, err := c.writer.Write(enriched, c.dataDir)
	if err != nil {
		return "", fmt.Errorf("correct: write: %w", err)
	}

	reminder := c.renderer.Render(classified, filePath)

	return reminder, nil
}

// SetConsolidator sets an optional consolidator for cluster-based merging before storage.
func (c *Corrector) SetConsolidator(cons interface {
	BeforeStore(ctx context.Context, candidate *memory.MemoryRecord) (ConsolidationAction, error)
}) {
	c.consolidator = cons
}

// SetProjectSlug sets the originating project slug for new memories.
func (c *Corrector) SetProjectSlug(slug string) {
	c.projectSlug = slug
}

// MemoryWriter writes an enriched memory to persistent storage (ARCH-4).
type MemoryWriter interface {
	Write(mem *memory.Enriched, dataDir string) (string, error)
}

// Renderer formats a classified memory as a system reminder string (ARCH-5).
type Renderer interface {
	Render(mem *memory.ClassifiedMemory, filePath string) string
}

// enrichedToMemoryRecord maps Enriched fields to MemoryRecord for consolidation.
func enrichedToMemoryRecord(enriched *memory.Enriched) *memory.MemoryRecord {
	return &memory.MemoryRecord{
		Title:            enriched.Title,
		Content:          enriched.Content,
		Keywords:         enriched.Keywords,
		Concepts:         enriched.Concepts,
		Principle:        enriched.Principle,
		AntiPattern:      enriched.AntiPattern,
		Generalizability: enriched.Generalizability,
		Confidence:       enriched.Confidence,
		ProjectSlug:      enriched.ProjectSlug,
	}
}

// memoryRecordToEnriched maps MemoryRecord back to Enriched after consolidation.
func memoryRecordToEnriched(record *memory.MemoryRecord) *memory.Enriched {
	return &memory.Enriched{
		Title:            record.Title,
		Content:          record.Content,
		Keywords:         record.Keywords,
		Concepts:         record.Concepts,
		Principle:        record.Principle,
		AntiPattern:      record.AntiPattern,
		Generalizability: record.Generalizability,
		Confidence:       record.Confidence,
	}
}
