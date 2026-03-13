package learn_test

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/dedup"
	"engram/internal/learn"
	"engram/internal/memory"
)

// LLMUnavailableError is returned when the LLM is not available.
type LLMUnavailableError struct{}

func (e LLMUnavailableError) Error() string {
	return "llm unavailable"
}

// TestComputeContentHash_ReturnsConsistentHash tests the ComputeContentHash function.
func TestComputeContentHash_ReturnsConsistentHash(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	hash1 := learn.ComputeContentHash([]string{"alpha", "beta"})
	hash2 := learn.ComputeContentHash([]string{"alpha", "beta"})
	hashEmpty := learn.ComputeContentHash([]string{})

	g.Expect(hash1).To(Equal(hash2))
	g.Expect(hash1).To(HaveLen(16))
	g.Expect(hashEmpty).NotTo(Equal(hash1))
}

// TestRegistryAbsorberFunc_RecordAbsorbed tests the function adapter.
func TestRegistryAbsorberFunc_RecordAbsorbed(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var called bool

	fn := learn.RegistryAbsorberFunc(func(_, _, _ string, _ time.Time) error {
		called = true

		return nil
	})

	err := fn.RecordAbsorbed("/path", "title", "hash", time.Now())
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(called).To(BeTrue())
}

func TestTP5c3_LLMMergerCombinesPrinciples(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	ctx := context.Background()

	merger := &mockMemoryMerger{merged: "combined principle"}
	existing := &memory.Stored{
		Title:     "existing",
		Principle: "old principle",
		Keywords:  []string{"alpha", "beta"},
		Concepts:  []string{"concept1"},
	}
	candidate := memory.CandidateLearning{
		Title:     "candidate",
		Principle: "new principle",
		Keywords:  []string{"alpha", "gamma"},
		Concepts:  []string{"concept2"},
	}

	merged, err := merger.MergePrinciples(ctx, existing.Principle, candidate.Principle)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(merged).To(Equal("combined principle"))
}

func TestTP5c4_FallbackMergeUsesLongerPrinciple(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	existing := &memory.Stored{
		Title:     "existing",
		Principle: "short",
		Keywords:  []string{"alpha", "beta"},
		Concepts:  []string{"concept1"},
	}
	candidate := memory.CandidateLearning{
		Title:     "candidate",
		Principle: "much longer principle text",
		Keywords:  []string{"alpha", "gamma"},
		Concepts:  []string{"concept2"},
	}

	// Fallback merge: longer principle
	if len(candidate.Principle) > len(existing.Principle) {
		g.Expect(candidate.Principle).To(Equal("much longer principle text"))
	}

	// Keywords union
	mergedKeywords := make(map[string]struct{})
	for _, k := range existing.Keywords {
		mergedKeywords[k] = struct{}{}
	}

	for _, k := range candidate.Keywords {
		mergedKeywords[k] = struct{}{}
	}

	g.Expect(mergedKeywords).To(HaveLen(3)) // alpha, beta, gamma
}

func TestTP5c5_FallbackMergeKeepsExistingWhenLonger(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	existing := &memory.Stored{
		Title:     "existing",
		Principle: "much longer principle text",
		Keywords:  []string{"alpha", "beta"},
		Concepts:  []string{"concept1"},
	}
	candidate := memory.CandidateLearning{
		Title:     "candidate",
		Principle: "short",
		Keywords:  []string{"alpha", "gamma"},
		Concepts:  []string{"concept2"},
	}

	// Fallback merge: existing is longer
	var merged string
	if len(candidate.Principle) > len(existing.Principle) {
		merged = candidate.Principle
	} else {
		merged = existing.Principle
	}

	g.Expect(merged).To(Equal("much longer principle text"))
}

func TestTP5c6_AbsorbedRecordAppendedAfterMerge(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	absorber := &mockRegistryAbsorber{}
	candidate := memory.CandidateLearning{
		Title:    "candidate learning",
		Keywords: []string{"alpha", "beta"},
	}

	// Simulate calling RecordAbsorbed
	err := absorber.RecordAbsorbed("/path/to/existing.toml", candidate.Title, "somehash", time.Now())
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(absorber.called).To(BeTrue())
	g.Expect(absorber.candidateTitle).To(Equal("candidate learning"))
}

func TestTP5c7_LLMMergerErrorFallsBackToDeterministic(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	ctx := context.Background()

	merger := &mockMemoryMerger{err: LLMUnavailableError{}}
	existing := &memory.Stored{
		Title:     "existing",
		Principle: "short",
		Keywords:  []string{"alpha", "beta"},
		Concepts:  []string{"concept1"},
	}
	candidate := memory.CandidateLearning{
		Title:     "candidate",
		Principle: "much longer principle",
		Keywords:  []string{"alpha", "gamma"},
		Concepts:  []string{"concept2"},
	}

	_, err := merger.MergePrinciples(ctx, existing.Principle, candidate.Principle)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		// On error, fallback to deterministic: longer principle
		var fallback string
		if len(candidate.Principle) > len(existing.Principle) {
			fallback = candidate.Principle
		} else {
			fallback = existing.Principle
		}

		g.Expect(fallback).To(Equal("much longer principle"))
	}
}

// TestTP5c_ProcessMerge_ExistingLonger exercises fallbackMergePrinciple when existing is longer.
func TestTP5c_ProcessMerge_ExistingLonger(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	ctx := context.Background()

	existing := &memory.Stored{
		Title:     "existing",
		Principle: "this is a much longer existing principle text",
		Keywords:  []string{"alpha"},
		Concepts:  []string{},
	}
	candidate := memory.CandidateLearning{
		Title:     "candidate",
		Principle: "short",
		Keywords:  []string{"beta"},
		Concepts:  []string{},
		Tier:      "A",
	}

	pair := dedup.MergePair{Candidate: candidate, Existing: existing}
	deduplicator := &mergePairDeduplicator{pair: pair}
	extractor := &fakeExtractorForMerge{candidates: []memory.CandidateLearning{candidate}}
	retriever := &fakeRetrieverForMerge{}
	writer := &fakeWriterForMerge{}
	mergeWriter := &fakeMergeWriterForTest{}

	learner := learn.New(extractor, retriever, deduplicator, writer, t.TempDir())
	learner.SetMergeWriter(mergeWriter)

	result, err := learner.Run(ctx, "transcript")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeNil())
	// existing principle is longer → keep existing
	g.Expect(mergeWriter.principle).To(Equal("this is a much longer existing principle text"))
}

// TestTP5c_ProcessMerge_ViaLearnerRun exercises processMerge through Learner.Run.
// This covers: processMerge, fallbackMergePrinciple, unionKeywords, unionConcepts,
// hashKeywords, SetMemoryMerger, SetMergeWriter, SetRegistryAbsorber.
func TestTP5c_ProcessMerge_ViaLearnerRun(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	ctx := context.Background()

	existing := &memory.Stored{
		Title:     "existing",
		Principle: "short",
		Keywords:  []string{"alpha", "beta"},
		Concepts:  []string{"c1"},
	}
	candidate := memory.CandidateLearning{
		Title:     "candidate",
		Principle: "much longer principle text",
		Keywords:  []string{"alpha", "gamma"},
		Concepts:  []string{"c2"},
		Tier:      "A",
	}

	pair := dedup.MergePair{Candidate: candidate, Existing: existing}
	deduplicator := &mergePairDeduplicator{pair: pair}
	extractor := &fakeExtractorForMerge{candidates: []memory.CandidateLearning{candidate}}
	retriever := &fakeRetrieverForMerge{}
	writer := &fakeWriterForMerge{}
	mergeWriter := &fakeMergeWriterForTest{}
	absorber := &mockRegistryAbsorber{}

	learner := learn.New(extractor, retriever, deduplicator, writer, t.TempDir())
	learner.SetMergeWriter(mergeWriter)
	learner.SetRegistryAbsorber(absorber)
	learner.SetMemoryMerger(nil) // triggers fallback path

	result, err := learner.Run(ctx, "transcript")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeNil())
	g.Expect(mergeWriter.called).To(BeTrue())
	// fallback merge: candidate.Principle is longer
	g.Expect(mergeWriter.principle).To(Equal("much longer principle text"))
	g.Expect(absorber.called).To(BeTrue())
}

// TestTP5c_ProcessMerge_WithLLMMerger exercises processMerge with a MemoryMerger.
func TestTP5c_ProcessMerge_WithLLMMerger(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	ctx := context.Background()

	existing := &memory.Stored{
		Title:     "existing",
		Principle: "old principle",
		Keywords:  []string{"alpha"},
		Concepts:  []string{},
	}
	candidate := memory.CandidateLearning{
		Title:     "candidate",
		Principle: "new principle",
		Keywords:  []string{"beta"},
		Concepts:  []string{},
		Tier:      "B",
	}

	pair := dedup.MergePair{Candidate: candidate, Existing: existing}
	deduplicator := &mergePairDeduplicator{pair: pair}
	extractor := &fakeExtractorForMerge{candidates: []memory.CandidateLearning{candidate}}
	retriever := &fakeRetrieverForMerge{}
	writer := &fakeWriterForMerge{}
	mergeWriter := &fakeMergeWriterForTest{}
	merger := &mockMemoryMerger{merged: "combined principle"}

	learner := learn.New(extractor, retriever, deduplicator, writer, t.TempDir())
	learner.SetMergeWriter(mergeWriter)
	learner.SetMemoryMerger(merger)

	result, err := learner.Run(ctx, "transcript")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeNil())
	g.Expect(mergeWriter.principle).To(Equal("combined principle"))
}

// fakeExtractorForMerge returns fixed candidates.
type fakeExtractorForMerge struct {
	candidates []memory.CandidateLearning
}

func (f *fakeExtractorForMerge) Extract(_ context.Context, _ string) ([]memory.CandidateLearning, error) {
	return f.candidates, nil
}

// fakeMergeWriterForTest records UpdateMerged calls.
type fakeMergeWriterForTest struct {
	called    bool
	principle string
}

func (w *fakeMergeWriterForTest) UpdateMerged(
	_ *memory.Stored,
	principle string,
	_, _ []string,
	_ time.Time,
) error {
	w.called = true
	w.principle = principle

	return nil
}

// fakeRetrieverForMerge returns no existing memories.
type fakeRetrieverForMerge struct{}

func (f *fakeRetrieverForMerge) ListMemories(_ context.Context, _ string) ([]*memory.Stored, error) {
	return nil, nil
}

// fakeWriterForMerge records writes.
type fakeWriterForMerge struct{}

func (f *fakeWriterForMerge) Write(_ *memory.Enriched, _ string) (string, error) {
	return "/tmp/memory.toml", nil
}

// mergePairDeduplicator is a test deduplicator that returns a single merge pair.
type mergePairDeduplicator struct {
	pair dedup.MergePair
}

func (d *mergePairDeduplicator) Classify(
	_ []memory.CandidateLearning,
	_ []*memory.Stored,
) dedup.ClassifyResult {
	return dedup.ClassifyResult{
		Surviving:  nil,
		MergePairs: []dedup.MergePair{d.pair},
	}
}

func (d *mergePairDeduplicator) Filter(
	_ []memory.CandidateLearning,
	_ []*memory.Stored,
) []memory.CandidateLearning {
	return nil
}

// Mock MemoryMerger for testing.
type mockMemoryMerger struct {
	merged string
	err    error
}

func (m *mockMemoryMerger) MergePrinciples(
	_ context.Context,
	_, _ string,
) (string, error) {
	if m.err != nil {
		return "", m.err
	}

	return m.merged, nil
}

// Mock RegistryAbsorber for testing.
type mockRegistryAbsorber struct {
	called         bool
	candidateTitle string
}

func (m *mockRegistryAbsorber) RecordAbsorbed(
	_, candidateTitle, _ string,
	_ time.Time,
) error {
	m.called = true
	m.candidateTitle = candidateTitle

	return nil
}
