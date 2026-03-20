package keyword_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/keyword"
)

func TestFilterByDocFrequency_AllCommon_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// "test" in 4/10 = 40%, "build" in 5/10 = 50% — both exceed 30%
	existing := [][]string{
		{"test", "build"},
		{"test", "build"},
		{"test", "build"},
		{"test", "build"},
		{"build"},
		{"zeta"}, {"eta"}, {"theta"}, {"iota"}, {"kappa"},
	}

	candidates := []string{"test", "build"}

	result := keyword.FilterByDocFrequency(candidates, existing, 0.3)

	g.Expect(result).To(BeEmpty())
}

func TestFilterByDocFrequency_EmptyCandidates_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	existing := [][]string{
		{"test", "alpha"},
		{"test", "beta"},
	}

	result := keyword.FilterByDocFrequency(nil, existing, 0.3)

	g.Expect(result).To(BeEmpty())
}

func TestFilterByDocFrequency_EmptyExisting_KeepsAll(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	candidates := []string{"alpha", "beta", "gamma"}

	result := keyword.FilterByDocFrequency(candidates, nil, 0.3)

	g.Expect(result).To(ConsistOf("alpha", "beta", "gamma"))
}

func TestFilterByDocFrequency_NewKeyword_AlwaysPasses(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	existing := [][]string{
		{"test", "alpha"},
		{"test", "beta"},
		{"test", "gamma"},
	}

	// "brand-new" not in any existing memory → 0% < 30% → passes
	candidates := []string{"brand-new"}

	result := keyword.FilterByDocFrequency(candidates, existing, 0.3)

	g.Expect(result).To(ConsistOf("brand-new"))
}

func TestFilterByDocFrequency_RemovesCommonKeywords(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// 10 existing memories. "test" appears in 4 (40% > 30% threshold).
	// "gomega" appears in 1 (10% < 30% threshold).
	existing := [][]string{
		{"test", "alpha"},
		{"test", "beta"},
		{"test", "gamma"},
		{"test", "delta"},
		{"gomega", "epsilon"},
		{"zeta"}, {"eta"}, {"theta"}, {"iota"}, {"kappa"},
	}

	candidates := []string{"test", "gomega", "targ-check-full"}

	result := keyword.FilterByDocFrequency(candidates, existing, 0.3)

	g.Expect(result).To(ConsistOf("gomega", "targ-check-full"))
	g.Expect(result).NotTo(ContainElement("test"))
}
