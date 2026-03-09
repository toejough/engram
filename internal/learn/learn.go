// Package learn implements the Session Learning pipeline (ARCH-14).
// It extracts candidate learnings from a session transcript, deduplicates
// them against existing memories, and writes surviving candidates as memories
// using the tier classified by the LLM extractor.
package learn

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"engram/internal/creationlog"
	"engram/internal/memory"
)

// CreationLogger records memory creation events for deferred visibility.
type CreationLogger interface {
	Append(entry creationlog.LogEntry, dataDir string) error
}

// Deduplicator filters candidates that are already represented in existing memories.
type Deduplicator interface {
	Filter(
		candidates []memory.CandidateLearning,
		existing []*memory.Stored,
	) []memory.CandidateLearning
}

// Learner orchestrates the four-stage Session Learning pipeline.
type Learner struct {
	extractor      TranscriptExtractor
	retriever      MemoryRetriever
	deduplicator   Deduplicator
	writer         MemoryWriter
	dataDir        string
	creationLogger CreationLogger // optional: log creation events for deferred visibility
	registrar      RegistryRegistrar
	stderr         io.Writer
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
		stderr:       os.Stderr,
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
	tierCounts := make(map[string]int)
	now := time.Now()

	for _, candidate := range surviving {
		filePath, err := l.writeCandidate(candidate, now)
		if err != nil {
			return nil, err
		}

		createdPaths = append(createdPaths, filePath)
		tierCounts[candidate.Tier]++
	}

	return &Result{
		CreatedPaths: createdPaths,
		SkippedCount: skippedCount,
		TierCounts:   tierCounts,
	}, nil
}

// SetCreationLogger attaches an optional CreationLogger to the Learner.
func (l *Learner) SetCreationLogger(logger CreationLogger) {
	l.creationLogger = logger
}

// SetRegistryRegistrar attaches an optional RegistryRegistrar to the Learner (UC-23).
func (l *Learner) SetRegistryRegistrar(registrar RegistryRegistrar) {
	l.registrar = registrar
}

// writeCandidate enriches and writes a single candidate, then logs its creation.
func (l *Learner) writeCandidate(
	candidate memory.CandidateLearning,
	now time.Time,
) (string, error) {
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
		Confidence:      candidate.Tier,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	filePath, err := l.writer.Write(enriched, l.dataDir)
	if err != nil {
		return "", fmt.Errorf("learn: write: %w", err)
	}

	if l.creationLogger != nil {
		entry := creationlog.LogEntry{
			Title:    candidate.Title,
			Tier:     candidate.Tier,
			Filename: filepath.Base(filePath),
		}

		logErr := l.creationLogger.Append(entry, l.dataDir)
		if logErr != nil {
			_, _ = fmt.Fprintf(l.stderr, "learn: creation log: %v\n", logErr)
		}
	}

	if l.registrar != nil {
		regErr := l.registrar.RegisterMemory(
			filePath, candidate.Title, candidate.Content, now,
		)
		if regErr != nil {
			_, _ = fmt.Fprintf(l.stderr, "learn: registry: %v\n", regErr)
		}
	}

	return filePath, nil
}

// MemoryRetriever lists existing memories from the data directory.
type MemoryRetriever interface {
	ListMemories(ctx context.Context, dataDir string) ([]*memory.Stored, error)
}

// MemoryWriter writes an enriched memory to persistent storage.
type MemoryWriter interface {
	Write(mem *memory.Enriched, dataDir string) (string, error)
}

// RegistryRegistrar registers new memories in the instruction registry (UC-23).
type RegistryRegistrar interface {
	RegisterMemory(filePath, title, content string, now time.Time) error
}

// Result holds the output of a learning run for feedback rendering.
type Result struct {
	CreatedPaths []string       // file paths of created memories
	SkippedCount int            // number of candidates filtered by dedup
	TierCounts   map[string]int // count of created memories per tier (A/B/C)
}

// TranscriptExtractor extracts candidate learnings from a session transcript.
type TranscriptExtractor interface {
	Extract(ctx context.Context, transcript string) ([]memory.CandidateLearning, error)
}
