package memory

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestDefaultSimilarityThreshold(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(DefaultSimilarityThreshold).To(Equal(0.7))
}

func TestFilterByMinScore_ZeroReturnsAll(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	results := []QueryResult{
		{Content: "a", Score: 0.1},
		{Content: "b", Score: 0.5},
		{Content: "c", Score: 0.9},
	}

	filtered := FilterByMinScore(results, 0.0)
	g.Expect(filtered).To(HaveLen(3))
}

func TestFilterByMinScore_FiltersBelow(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	results := []QueryResult{
		{Content: "low", Score: 0.3},
		{Content: "mid", Score: 0.6},
		{Content: "high", Score: 0.9},
	}

	filtered := FilterByMinScore(results, 0.5)
	g.Expect(filtered).To(HaveLen(2))
	g.Expect(filtered[0].Content).To(Equal("mid"))
	g.Expect(filtered[1].Content).To(Equal("high"))
}

func TestFilterByMinScore_AllBelowReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	results := []QueryResult{
		{Content: "low", Score: 0.1},
		{Content: "also low", Score: 0.2},
	}

	filtered := FilterByMinScore(results, 0.7)
	g.Expect(filtered).To(BeEmpty())
}

func TestFilterByMinScore_ExactThresholdIncluded(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	results := []QueryResult{
		{Content: "exact", Score: 0.7},
		{Content: "below", Score: 0.69},
	}

	filtered := FilterByMinScore(results, 0.7)
	g.Expect(filtered).To(HaveLen(1))
	g.Expect(filtered[0].Content).To(Equal("exact"))
}

func TestFilterByMinScore_EmptyInputReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	filtered := FilterByMinScore(nil, 0.5)
	g.Expect(filtered).To(BeEmpty())
}
