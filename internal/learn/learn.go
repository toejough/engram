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
	"engram/internal/dedup"
	"engram/internal/memory"
)

// CreationLogger records memory creation events for deferred visibility.
type CreationLogger interface {
	Append(entry creationlog.LogEntry, dataDir string) error
}

// Deduplicator filters and classifies candidates for dedup and merge (UC-33).
type Deduplicator interface {
	Filter(
		candidates []memory.CandidateLearning,
		existing []*memory.Stored,
	) []memory.CandidateLearning
	Classify(
		candidates []memory.CandidateLearning,
		existing []*memory.Stored,
	) dedup.ClassifyResult
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
	merger         MemoryMerger     // optional: merge candidates with existing memories (UC-33)
	mergeWriter    MergeWriter      // optional: write merged memories to disk (UC-33)
	absorber       RegistryAbsorber // optional: record merges in registry (UC-33)
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

	// Classify candidates into survivors (new memories) and merge pairs (UC-33)
	classified := l.deduplicator.Classify(candidates, existing)
	surviving := classified.Surviving
	mergePairs := classified.MergePairs
	skippedCount := len(candidates) - len(surviving) - len(mergePairs)

	createdPaths := make([]string, 0, len(surviving))
	tierCounts := make(map[string]int)
	now := time.Now()

	// Write surviving candidates as new memories
	for _, candidate := range surviving {
		filePath, err := l.writeCandidate(candidate, now)
		if err != nil {
			return nil, err
		}

		createdPaths = append(createdPaths, filePath)
		tierCounts[candidate.Tier]++
	}

	// Process merge pairs (UC-33)
	for _, pair := range mergePairs {
		err := l.processMerge(ctx, pair.Candidate, pair.Existing, now)
		if err != nil {
			return nil, fmt.Errorf("learn: merge: %w", err)
		}
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

// SetMemoryMerger attaches an optional MemoryMerger to the Learner (UC-33).
func (l *Learner) SetMemoryMerger(merger MemoryMerger) {
	l.merger = merger
}

// SetMergeWriter attaches an optional MergeWriter to the Learner (UC-33).
func (l *Learner) SetMergeWriter(writer MergeWriter) {
	l.mergeWriter = writer
}

// SetRegistryAbsorber attaches an optional RegistryAbsorber to the Learner (UC-33).
func (l *Learner) SetRegistryAbsorber(absorber RegistryAbsorber) {
	l.absorber = absorber
}

// SetRegistryRegistrar attaches an optional RegistryRegistrar to the Learner (UC-23).
func (l *Learner) SetRegistryRegistrar(registrar RegistryRegistrar) {
	l.registrar = registrar
}

// fallbackMergePrinciple uses the longer principle text (UC-33).
func (l *Learner) fallbackMergePrinciple(existing, candidate string) string {
	if len(candidate) > len(existing) {
		return candidate
	}

	return existing
}

// hashKeywords returns a hash of the keywords (for the Absorbed record).
func (l *Learner) hashKeywords(keywords []string) string {
	return ComputeContentHash(keywords)
}

// processMerge handles the merge of a candidate with an existing memory (UC-33).
func (l *Learner) processMerge(
	ctx context.Context,
	candidate memory.CandidateLearning,
	existing *memory.Stored,
	now time.Time,
) error {
	// Determine the merged principle
	var mergedPrinciple string

	if l.merger != nil {
		// Try LLM-assisted merge
		merged, err := l.merger.MergePrinciples(ctx, existing.Principle, candidate.Principle)
		if err == nil && merged != "" {
			mergedPrinciple = merged
		} else {
			// Fall back to deterministic merge on error or empty result
			mergedPrinciple = l.fallbackMergePrinciple(existing.Principle, candidate.Principle)
		}
	} else {
		// No merger configured, use deterministic merge
		mergedPrinciple = l.fallbackMergePrinciple(existing.Principle, candidate.Principle)
	}

	// Union keywords and concepts
	mergedKeywords := l.unionKeywords(existing.Keywords, candidate.Keywords)
	mergedConcepts := l.unionConcepts(existing.Concepts, candidate.Concepts)

	// Write merged memory to disk
	if l.mergeWriter != nil {
		err := l.mergeWriter.UpdateMerged(existing, mergedPrinciple, mergedKeywords, mergedConcepts, now)
		if err != nil {
			return fmt.Errorf("merge writer: %w", err)
		}
	}

	// Record merge in registry
	if l.absorber != nil {
		contentHash := l.hashKeywords(candidate.Keywords)

		err := l.absorber.RecordAbsorbed(existing.FilePath, candidate.Title, contentHash, now)
		if err != nil {
			_, _ = fmt.Fprintf(l.stderr, "learn: absorber: %v\n", err)
		}
	}

	return nil
}

// unionConcepts returns the union of two concept slices.
func (l *Learner) unionConcepts(a, b []string) []string {
	set := make(map[string]struct{})

	for _, c := range a {
		set[c] = struct{}{}
	}

	for _, c := range b {
		set[c] = struct{}{}
	}

	result := make([]string, 0, len(set))
	for c := range set {
		result = append(result, c)
	}

	return result
}

// unionKeywords returns the union of two keyword slices.
func (l *Learner) unionKeywords(a, b []string) []string {
	set := make(map[string]struct{})

	for _, k := range a {
		set[k] = struct{}{}
	}

	for _, k := range b {
		set[k] = struct{}{}
	}

	result := make([]string, 0, len(set))
	for k := range set {
		result = append(result, k)
	}

	return result
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

// MemoryMerger combines principles during merge (UC-33).
type MemoryMerger interface {
	MergePrinciples(ctx context.Context, existing, candidate string) (string, error)
}

// MemoryRetriever lists existing memories from the data directory.
type MemoryRetriever interface {
	ListMemories(ctx context.Context, dataDir string) ([]*memory.Stored, error)
}

// MemoryWriter writes an enriched memory to persistent storage.
type MemoryWriter interface {
	Write(mem *memory.Enriched, dataDir string) (string, error)
}

// MergeWriter updates an existing memory with merged fields (UC-33).
type MergeWriter interface {
	UpdateMerged(existing *memory.Stored, principle string, keywords, concepts []string, now time.Time) error
}

// RegistryAbsorber records a merge in the registry (UC-33).
type RegistryAbsorber interface {
	RecordAbsorbed(existingPath, candidateTitle, contentHash string, now time.Time) error
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
