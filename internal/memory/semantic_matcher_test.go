package memory_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/memory"
)

// ============================================================================
// ISSUE-194: SemanticMatcher implementation tests
// ============================================================================

// Compile-time interface check: MemoryStoreSemanticMatcher implements SemanticMatcher
var _ memory.SemanticMatcher = (*memory.MemoryStoreSemanticMatcher)(nil)

// TEST-194-01: FindSimilarMemories returns results for seeded memories
// traces: ISSUE-194 AC-1
func TestMemoryStoreSemanticMatcherFindsMemories(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()

	// Seed the memory store with entries so there's something to find
	for _, msg := range []string{
		"use TDD for all code changes",
		"always write tests before implementation",
		"run mage check before declaring done",
	} {
		err := memory.Learn(memory.LearnOpts{
			Message:    msg,
			MemoryRoot: tempDir,
		})
		g.Expect(err).ToNot(HaveOccurred())
	}

	matcher := memory.NewMemoryStoreSemanticMatcher(tempDir)

	// Threshold must be low because stored entries include timestamp prefix
	// (e.g. "- 2026-02-10 12:10: use TDD for all code changes") which dilutes
	// the embedding similarity score even for exact text matches.
	results, err := matcher.FindSimilarMemories("use TDD for all code changes", 0.01, 10)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).ToNot(BeNil())
	g.Expect(results).To(ContainElement(ContainSubstring("use TDD for all code changes")))
}

// TEST-194-02: FindSimilarMemories returns nil, nil when no memories match
// traces: ISSUE-194 AC-2
func TestMemoryStoreSemanticMatcherEmptyReturnsNilNil(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Empty memory root — no memories stored
	tempDir := t.TempDir()
	matcher := memory.NewMemoryStoreSemanticMatcher(tempDir)

	results, err := matcher.FindSimilarMemories("completely unique query with no matches", 0.9, 10)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).To(BeNil())
}

// TEST-194-03: FindSimilarMemories respects threshold filtering
// traces: ISSUE-194 AC-3
func TestMemoryStoreSemanticMatcherThresholdFiltering(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	matcher := memory.NewMemoryStoreSemanticMatcher(tempDir)

	// High threshold should return fewer or no results
	highResults, err := matcher.FindSimilarMemories("test query", 0.99, 100)
	g.Expect(err).ToNot(HaveOccurred())

	// Low threshold should return equal or more results
	lowResults, err := matcher.FindSimilarMemories("test query", 0.01, 100)
	g.Expect(err).ToNot(HaveOccurred())

	g.Expect(len(highResults)).To(BeNumerically("<=", len(lowResults)))
}

// TEST-194-04: Property test — FindSimilarMemories never returns more than limit results
// traces: ISSUE-194 AC-4
func TestMemoryStoreSemanticMatcherLimitCapping(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		tempDir := t.TempDir()
		matcher := memory.NewMemoryStoreSemanticMatcher(tempDir)

		limit := rapid.IntRange(1, 50).Draw(rt, "limit")
		threshold := rapid.Float64Range(0.0, 1.0).Draw(rt, "threshold")
		query := rapid.StringMatching(`[a-zA-Z ]{5,30}`).Draw(rt, "query")

		results, err := matcher.FindSimilarMemories(query, threshold, limit)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(len(results)).To(BeNumerically("<=", limit))
	})
}
