// Package learn implements the Session Learning pipeline (ARCH-14).
// It extracts candidate learnings from a session transcript, deduplicates
// them against existing memories, and writes surviving candidates as memories
// with confidence tier C.
package learn

import (
	"context"
	"fmt"
	"time"

	"engram/internal/memory"
)

// Deduplicator filters candidates that are already represented in existing memories.
type Deduplicator interface {
	Filter(
		candidates []memory.CandidateLearning,
		existing []*memory.Stored,
	) []memory.CandidateLearning
}

// Learner orchestrates the four-stage Session Learning pipeline.
type Learner struct {
	extractor    TranscriptExtractor
	retriever    MemoryRetriever
	deduplicator Deduplicator
	writer       MemoryWriter
	dataDir      string
}

// New creates a Learner wired with all pipeline stages.
func New(
	extractor TranscriptExtractor,
	retriever MemoryRetriever,
	deduplicator Deduplicator,
	writer MemoryWriter,
	dataDir string,
) *Learner {
	return &Learner{
		extractor:    extractor,
		retriever:    retriever,
		deduplicator: deduplicator,
		writer:       writer,
		dataDir:      dataDir,
	}
}

// Run executes the learning pipeline for a single session transcript.
// Returns a Result with created file paths and skipped count.
func (l *Learner) Run(ctx context.Context, transcript string) (*Result, error) {
	candidates, err := l.extractor.Extract(ctx, transcript)
	if err != nil {
		return nil, fmt.Errorf("learn: extract: %w", err)
	}

	if len(candidates) == 0 {
		return &Result{}, nil
	}

	existing, err := l.retriever.ListMemories(ctx, l.dataDir)
	if err != nil {
		return nil, fmt.Errorf("learn: list memories: %w", err)
	}

	surviving := l.deduplicator.Filter(candidates, existing)
	skippedCount := len(candidates) - len(surviving)

	createdPaths := make([]string, 0, len(surviving))

	now := time.Now()

	for _, candidate := range surviving {
		enriched := &memory.Enriched{
			Title:           candidate.Title,
			Content:         candidate.Content,
			ObservationType: candidate.ObservationType,
			Concepts:        candidate.Concepts,
			Keywords:        candidate.Keywords,
			Principle:       candidate.Principle,
			AntiPattern:     candidate.AntiPattern,
			Rationale:       candidate.Rationale,
			FilenameSummary: candidate.FilenameSummary,
			Confidence:      "C",
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		filePath, err := l.writer.Write(enriched, l.dataDir)
		if err != nil {
			return nil, fmt.Errorf("learn: write: %w", err)
		}

		createdPaths = append(createdPaths, filePath)
	}

	return &Result{
		CreatedPaths: createdPaths,
		SkippedCount: skippedCount,
	}, nil
}

// MemoryRetriever lists existing memories from the data directory.
type MemoryRetriever interface {
	ListMemories(ctx context.Context, dataDir string) ([]*memory.Stored, error)
}

// MemoryWriter writes an enriched memory to persistent storage.
type MemoryWriter interface {
	Write(mem *memory.Enriched, dataDir string) (string, error)
}

// Result holds the output of a learning run for feedback rendering.
type Result struct {
	CreatedPaths []string // file paths of created memories
	SkippedCount int      // number of candidates filtered by dedup
}

// TranscriptExtractor extracts candidate learnings from a session transcript.
type TranscriptExtractor interface {
	Extract(ctx context.Context, transcript string) ([]memory.CandidateLearning, error)
}
