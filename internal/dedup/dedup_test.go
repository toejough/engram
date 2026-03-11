package dedup_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/dedup"
	"engram/internal/memory"
)

func TestT52_HighOverlapFiltersCandidate(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	d := dedup.New()

	// 3 of 4 candidate keywords match → 75% overlap → filtered
	candidates := []memory.CandidateLearning{
		{Keywords: []string{"alpha", "beta", "gamma", "delta"}},
	}
	existing := []*memory.Stored{
		{Keywords: []string{"alpha", "beta", "gamma", "unrelated"}},
	}

	result := d.Filter(candidates, existing)

	g.Expect(result).To(BeEmpty())
}

func TestT53_LowOverlapKeepsCandidate(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	d := dedup.New()

	// 2 of 4 candidate keywords match → exactly 50% overlap → kept (threshold is >50%)
	candidates := []memory.CandidateLearning{
		{Keywords: []string{"alpha", "beta", "gamma", "delta"}},
	}
	existing := []*memory.Stored{
		{Keywords: []string{"alpha", "beta", "unrelated1", "unrelated2"}},
	}

	result := d.Filter(candidates, existing)

	g.Expect(result).To(HaveLen(1))
}

func TestT54_NoExistingMemoriesAllCandidatesSurvive(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	d := dedup.New()

	candidates := []memory.CandidateLearning{
		{Keywords: []string{"foo", "bar"}},
		{Keywords: []string{"baz", "qux"}},
	}

	result := d.Filter(candidates, nil)

	g.Expect(result).To(HaveLen(2))
}

func TestT55_EmptyKeywordsNeverFiltered(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	d := dedup.New()

	// Candidate has no keywords — 0% overlap regardless of existing memories
	candidates := []memory.CandidateLearning{
		{Keywords: []string{}},
	}
	existing := []*memory.Stored{
		{Keywords: []string{"alpha", "beta", "gamma"}},
	}

	result := d.Filter(candidates, existing)

	g.Expect(result).To(HaveLen(1))
}

func TestT56_IdempotencySecondRunDeduplicatesAgainstFirst(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	d := dedup.New()

	// First pass: no existing memories, candidate survives
	candidates := []memory.CandidateLearning{
		{Keywords: []string{"alpha", "beta", "gamma"}},
	}
	firstPass := d.Filter(candidates, nil)
	g.Expect(firstPass).To(HaveLen(1))

	// Second pass: the first result now acts as existing memory
	// Same candidate again → >50% overlap → filtered
	existingFromFirstPass := []*memory.Stored{
		{Keywords: firstPass[0].Keywords},
	}
	secondPass := d.Filter(candidates, existingFromFirstPass)

	g.Expect(secondPass).To(BeEmpty())
}

// Tests for UC-33: Merge-on-Write

func TestTP5c1_HighOverlapReturnsAsMergePair(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	d := dedup.New()

	// 3 of 4 candidate keywords match → 75% overlap → merge pair
	candidates := []memory.CandidateLearning{
		{Title: "candidate", Keywords: []string{"alpha", "beta", "gamma", "delta"}},
	}
	existing := []*memory.Stored{
		{Title: "existing", Keywords: []string{"alpha", "beta", "gamma", "unrelated"}},
	}

	result := d.Classify(candidates, existing)

	g.Expect(result.Surviving).To(BeEmpty())
	g.Expect(result.MergePairs).To(HaveLen(1))
	g.Expect(result.MergePairs[0].Candidate.Title).To(Equal("candidate"))
	g.Expect(result.MergePairs[0].Existing.Title).To(Equal("existing"))
}

func TestTP5c2_LowOverlapPassesThroughAsSurviving(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	d := dedup.New()

	// 2 of 4 candidate keywords match → exactly 50% overlap → pass-through (threshold is >50%)
	candidates := []memory.CandidateLearning{
		{Title: "candidate", Keywords: []string{"alpha", "beta", "gamma", "delta"}},
	}
	existing := []*memory.Stored{
		{Title: "existing", Keywords: []string{"alpha", "beta", "unrelated1", "unrelated2"}},
	}

	result := d.Classify(candidates, existing)

	g.Expect(result.Surviving).To(HaveLen(1))
	g.Expect(result.Surviving[0].Title).To(Equal("candidate"))
	g.Expect(result.MergePairs).To(BeEmpty())
}

func TestTP5c8_EmptyKeywordsPassThroughAsSurviving(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	d := dedup.New()

	// Candidate has no keywords → pass-through, not merge
	candidates := []memory.CandidateLearning{
		{Title: "candidate", Keywords: []string{}},
	}
	existing := []*memory.Stored{
		{Title: "existing", Keywords: []string{"alpha", "beta"}},
	}

	result := d.Classify(candidates, existing)

	g.Expect(result.Surviving).To(HaveLen(1))
	g.Expect(result.MergePairs).To(BeEmpty())
}
