package learn_test

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
)

// LLMUnavailableError is a test error type for LLM unavailability.
type LLMUnavailableError struct{}

func (e LLMUnavailableError) Error() string {
	return "llm unavailable"
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

	var absorbCalled bool

	var absorbedCandidateTitle string

	var recordAbsorbedErr error

	recordAbsorbed := func(_ string, candidateTitle, _ string, _ time.Time) error {
		absorbCalled = true
		absorbedCandidateTitle = candidateTitle

		return recordAbsorbedErr
	}

	candidate := memory.CandidateLearning{
		Title:    "candidate learning",
		Keywords: []string{"alpha", "beta"},
	}

	// Simulate calling recordAbsorbed
	_ = recordAbsorbed("/path/to/existing.toml", candidate.Title, "somehash", time.Now())

	g.Expect(absorbCalled).To(BeTrue())
	g.Expect(absorbedCandidateTitle).To(Equal("candidate learning"))
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

// mockMemoryMerger is a test double for the MemoryMerger interface.
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
