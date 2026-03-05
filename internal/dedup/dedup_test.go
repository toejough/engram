package dedup_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/dedup"
	"engram/internal/memory"
)

func TestFilter_EmptyKeywords_NeverFiltered(t *testing.T) {
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

func TestFilter_HighOverlap_FiltersCandidate(t *testing.T) {
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

func TestFilter_Idempotency_SecondRunDeduplicatesAgainstFirst(t *testing.T) {
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

func TestFilter_LowOverlap_KeepsCandidate(t *testing.T) {
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

func TestFilter_NoExistingMemories_AllCandidatesSurvive(t *testing.T) {
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
