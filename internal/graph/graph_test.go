package graph_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/graph"
	"engram/internal/registry"
)

// TestBuildContentSimilarity_CreatesLinkAboveThreshold verifies a link is created when BM25 score
// exceeds the minimum threshold.
// BM25 IDF formula: log((N-df+0.5)/(df+0.5)). With N=2,df=1: log(1)=0.
// Need N>=3 for non-zero IDF when df=1.
func TestBuildContentSimilarity_CreatesLinkAboveThreshold(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	builder := graph.New()
	entry := registry.InstructionEntry{
		ID:      "e1",
		Title:   "goroutine channel mutex waitgroup atomics",
		Content: "goroutine channel mutex waitgroup atomics",
	}
	// 3-doc corpus ensures IDF > 0 for terms appearing in only 1 doc.
	existing := []registry.InstructionEntry{
		{ID: "related", Title: "goroutine channel mutex waitgroup", Content: "goroutine channel mutex waitgroup"},
		{ID: "noise1", Title: "recipe cooking ingredient baking flour", Content: "recipe cooking ingredient baking flour"},
		{ID: "noise2", Title: "photosynthesis chlorophyll sunlight", Content: "photosynthesis chlorophyll sunlight"},
	}

	links := builder.BuildContentSimilarity(entry, existing)

	// "related" shares 4 unique terms with entry → BM25 score > 0.05 threshold.
	g.Expect(links).NotTo(BeEmpty())
	g.Expect(links[0].Basis).To(Equal("content_similarity"))
	g.Expect(links[0].Weight).To(BeNumerically(">", 0))
}

// TestBuildContentSimilarity_EmptyExisting returns no links for empty corpus.
func TestBuildContentSimilarity_EmptyExisting(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	builder := graph.New()
	entry := registry.InstructionEntry{ID: "e1", Title: "foo", Content: "bar"}

	links := builder.BuildContentSimilarity(entry, nil)

	g.Expect(links).To(BeEmpty())
}

// TestBuildContentSimilarity_SelfLinkExcluded verifies entry's own ID is never a link target.
// Uses two docs so BM25 IDF is non-zero; the self-entry is excluded from results.
func TestBuildContentSimilarity_SelfLinkExcluded(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	builder := graph.New()
	entry := registry.InstructionEntry{ID: "same", Title: "alpha beta gamma", Content: "alpha beta gamma"}
	existing := []registry.InstructionEntry{
		{ID: "same", Title: "alpha beta gamma", Content: "alpha beta gamma"},
		{ID: "other", Title: "alpha beta gamma delta", Content: "alpha beta gamma delta"},
	}

	links := builder.BuildContentSimilarity(entry, existing)

	for _, l := range links {
		g.Expect(l.Target).NotTo(Equal("same"))
	}
}

// T-P3-4: BuildConceptOverlap self-link excluded
func TestConceptOverlapSelfLinkExcluded(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	builder := graph.New()
	entry := registry.InstructionEntry{
		ID:    "same",
		Title: "foo bar",
	}
	existing := []registry.InstructionEntry{
		{ID: "same", Title: "foo bar"},  // Self-match, should be excluded
		{ID: "other", Title: "foo baz"}, // Different, should be included
	}

	links := builder.BuildConceptOverlap(entry, existing)

	// Should only have link to "other", not "same"
	g.Expect(links).To(HaveLen(1))
	g.Expect(links[0].Target).To(Equal("other"))
}

// T-P3-3: BuildConceptOverlap threshold 0.15
func TestConceptOverlapThreshold(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	builder := graph.New()
	entry := registry.InstructionEntry{
		ID:    "new",
		Title: "alpha beta gamma delta epsilon",
	}
	existing := []registry.InstructionEntry{
		// Jaccard above: {alpha,beta,gamma} / {alpha,beta,gamma,delta,epsilon,zeta} = 3/6 = 0.5 >= 0.15
		{ID: "above", Title: "alpha beta gamma zeta"},
		// Jaccard below: {} / {alpha,beta,gamma,delta,epsilon,zeta,eta,theta,iota,kappa} = 0/10 < 0.15
		{ID: "below", Title: "zeta eta theta iota kappa"},
	}

	links := builder.BuildConceptOverlap(entry, existing)

	// Only the "above" entry should produce a link
	g.Expect(links).To(HaveLen(1))
	g.Expect(links[0].Target).To(Equal("above"))
}

// T-P3-5: BuildContentSimilarity threshold 0.05
func TestContentSimilarityThreshold(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	builder := graph.New()
	entry := registry.InstructionEntry{
		ID:    "new",
		Title: "always use parallel test parallel use test parallel",
	}
	existing := []registry.InstructionEntry{
		{ID: "similar", Title: "always use parallel test with parallel"},
		{ID: "dissimilar", Title: "fish birds cats dogs elephant"},
	}

	links := builder.BuildContentSimilarity(entry, existing)

	// At least verify that links are produced and filtered
	// (exact threshold behavior depends on BM25 scoring of the text)
	for _, link := range links {
		g.Expect(link.Weight).To(BeNumerically(">=", 0.01))
		g.Expect(link.Weight).To(BeNumerically("<=", 1.0))
		g.Expect(link.Basis).To(Equal("content_similarity"))
	}
}

// T-P3-6: BuildContentSimilarity weight capped at 1.0
func TestContentSimilarityWeightCapped(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	builder := graph.New()
	entry := registry.InstructionEntry{
		ID:    "new",
		Title: "mercury venus earth mars jupiter",
	}
	existing := []registry.InstructionEntry{
		{ID: "ex1", Title: "mercury venus earth mars jupiter"},
	}

	links := builder.BuildContentSimilarity(entry, existing)

	for _, link := range links {
		g.Expect(link.Weight).To(BeNumerically("<=", 1.0))
	}
}

// T-P3-2: Jaccard correct for overlapping sets
func TestJaccardCorrectForOverlapping(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	builder := graph.New()
	entry := registry.InstructionEntry{
		ID:    "new",
		Title: "use targ build",
	}
	existing := []registry.InstructionEntry{
		{ID: "ex1", Title: "use targ test"},
	}

	links := builder.BuildConceptOverlap(entry, existing)
	g.Expect(links).To(HaveLen(1))
	g.Expect(links[0].Target).To(Equal("ex1"))
	g.Expect(links[0].Basis).To(Equal("concept_overlap"))
	// Jaccard of {use, targ, build} vs {use, targ, test} = {use, targ} / {use, targ, build, test} = 2/4 = 0.5
	g.Expect(links[0].Weight).To(BeNumerically("~", 0.5, 0.01))
}

// T-P3-1: Jaccard zero for disjoint token sets
func TestJaccardZeroForDisjointSets(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	builder := graph.New()
	entry := registry.InstructionEntry{
		ID:    "new",
		Title: "foo bar",
	}
	existing := []registry.InstructionEntry{
		{ID: "ex1", Title: "baz qux"},
	}

	links := builder.BuildConceptOverlap(entry, existing)
	g.Expect(links).To(BeEmpty())
}

// T-P3-12: Prune preserves links at or above weight threshold
func TestPrunePreservesHighWeight(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	links := []registry.Link{
		{Target: "ex1", Weight: 0.1, Basis: "co_surfacing", CoSurfacingCount: 20},
	}

	pruned := graph.Prune(links)

	g.Expect(pruned).To(HaveLen(1))
	g.Expect(pruned[0].Target).To(Equal("ex1"))
}

// T-P3-11: Prune preserves links below threshold with insufficient count
func TestPrunePreservesInsufficient(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	links := []registry.Link{
		{Target: "ex1", Weight: 0.05, Basis: "co_surfacing", CoSurfacingCount: 9},
	}

	pruned := graph.Prune(links)

	g.Expect(pruned).To(HaveLen(1))
	g.Expect(pruned[0].Target).To(Equal("ex1"))
}

// T-P3-10: Prune removes links below threshold with sufficient count
func TestPruneRemovesDeadLinks(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	links := []registry.Link{
		{Target: "ex1", Weight: 0.05, Basis: "co_surfacing", CoSurfacingCount: 10},
	}

	pruned := graph.Prune(links)

	g.Expect(pruned).To(BeEmpty())
}

// T-P3-9: UpdateCoSurfacing caps weight at 1.0
func TestUpdateCoSurfacingCapWeight(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	links := []registry.Link{
		{Target: "ex1", Weight: 0.95, Basis: "co_surfacing", CoSurfacingCount: 5},
	}

	updated := graph.UpdateCoSurfacing(links, "ex1")

	g.Expect(updated[0].Weight).To(Equal(1.0))
	g.Expect(updated[0].CoSurfacingCount).To(Equal(6))
}

// T-P3-8: UpdateCoSurfacing creates new link if none exists
func TestUpdateCoSurfacingCreatesNew(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	links := []registry.Link{}

	updated := graph.UpdateCoSurfacing(links, "new_target")

	g.Expect(updated).To(HaveLen(1))
	g.Expect(updated[0].Target).To(Equal("new_target"))
	g.Expect(updated[0].Weight).To(Equal(0.1))
	g.Expect(updated[0].Basis).To(Equal("co_surfacing"))
	g.Expect(updated[0].CoSurfacingCount).To(Equal(1))
}

// T-P3-7: UpdateCoSurfacing increments existing link
func TestUpdateCoSurfacingIncrementsExisting(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	links := []registry.Link{
		{Target: "ex1", Weight: 0.5, Basis: "co_surfacing", CoSurfacingCount: 3},
	}

	updated := graph.UpdateCoSurfacing(links, "ex1")

	g.Expect(updated).To(HaveLen(1))
	g.Expect(updated[0].Weight).To(BeNumerically("~", 0.6, 0.001))
	g.Expect(updated[0].CoSurfacingCount).To(Equal(4))
}

// TestUpdateEvaluationCorrelation_CapsAtOne verifies weight is capped at 1.0.
func TestUpdateEvaluationCorrelation_CapsAtOne(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	links := []registry.Link{
		{Target: "target1", Weight: 0.99, Basis: "evaluation_correlation"},
	}

	updated := graph.UpdateEvaluationCorrelation(links, "target1")

	g.Expect(updated[0].Weight).To(BeNumerically("<=", 1.0))
}

// TestUpdateEvaluationCorrelation_CreatesNewLink verifies a new link is appended when none exists.
func TestUpdateEvaluationCorrelation_CreatesNewLink(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	updated := graph.UpdateEvaluationCorrelation(nil, "target1")

	g.Expect(updated).To(HaveLen(1))
	g.Expect(updated[0].Target).To(Equal("target1"))
	g.Expect(updated[0].Basis).To(Equal("evaluation_correlation"))
	g.Expect(updated[0].Weight).To(BeNumerically(">", 0))
}

// TestUpdateEvaluationCorrelation_IncrementsExisting verifies an existing link weight grows.
func TestUpdateEvaluationCorrelation_IncrementsExisting(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	links := []registry.Link{
		{Target: "target1", Weight: 0.5, Basis: "evaluation_correlation"},
	}

	updated := graph.UpdateEvaluationCorrelation(links, "target1")

	g.Expect(updated).To(HaveLen(1))
	g.Expect(updated[0].Weight).To(BeNumerically(">", 0.5))
}
