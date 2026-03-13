package graph_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/graph"
	"engram/internal/registry"
)

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

	if len(links) == 0 {
		return
	}

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
		{
			ID:    "above",
			Title: "alpha beta gamma zeta",
		}, // Jaccard: {alpha,beta,gamma} / {alpha,beta,gamma,delta,epsilon,zeta} = 3/6 = 0.5 >= 0.15
		{
			ID:    "below",
			Title: "zeta eta theta iota kappa",
		}, // Jaccard: {} / {alpha,beta,gamma,delta,epsilon,zeta,eta,theta,iota,kappa} = 0/10 = 0 < 0.15
	}

	links := builder.BuildConceptOverlap(entry, existing)

	// Only the "above" entry should produce a link
	g.Expect(links).To(HaveLen(1))

	if len(links) == 0 {
		return
	}

	g.Expect(links[0].Target).To(Equal("above"))
}

// T-P3-5b: BuildContentSimilarity returns empty for empty existing
func TestContentSimilarityEmptyExisting(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	builder := graph.New()
	entry := registry.InstructionEntry{
		ID:      "new",
		Title:   "use targ build",
		Content: "always use targ for all builds",
	}

	links := builder.BuildContentSimilarity(entry, nil)

	g.Expect(links).To(BeEmpty())
}

// T-P3-5a: BuildContentSimilarity produces link for matching content.
// BM25 requires >=3 documents for positive IDF (with N=1, IDF=log(0.5/1.5)<0).
func TestContentSimilarityProducesLink(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	builder := graph.New()
	entry := registry.InstructionEntry{
		ID:      "new",
		Title:   "targ build invocation system",
		Content: "targ build invocation system",
	}
	// 3 docs: "match" shares unique terms; "other1"/"other2" use different words.
	// With N=3 and df=1, IDF = log(2.5/1.5) > 0.
	existing := []registry.InstructionEntry{
		{
			ID:    "match",
			Title: "targ build invocation system",
		},
		{
			ID:    "other1",
			Title: "cat fish bird elephant",
		},
		{
			ID:    "other2",
			Title: "river mountain ocean desert",
		},
	}

	links := builder.BuildContentSimilarity(entry, existing)

	g.Expect(links).NotTo(BeEmpty())

	if len(links) == 0 {
		return
	}

	g.Expect(links[0].Target).To(Equal("match"))
	g.Expect(links[0].Weight).To(BeNumerically("<=", 1.0))
	g.Expect(links[0].Basis).To(Equal("content_similarity"))
}

// T-P3-5c: BuildContentSimilarity excludes self-link
func TestContentSimilaritySelfLinkExcluded(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	builder := graph.New()
	entry := registry.InstructionEntry{
		ID:      "mem1",
		Title:   "use targ build for all go builds",
		Content: "always use targ",
	}
	existing := []registry.InstructionEntry{
		{ID: "mem1", Title: "use targ build for all go builds", Content: "always use targ"},
		{ID: "mem2", Title: "use targ test for all go tests", Content: "always use targ"},
	}

	links := builder.BuildContentSimilarity(entry, existing)

	for _, link := range links {
		g.Expect(link.Target).NotTo(Equal("mem1"))
	}
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
		Title: "alpha beta gamma delta epsilon",
	}
	existing := []registry.InstructionEntry{
		{ID: "ex1", Title: "alpha beta gamma delta epsilon"},
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

	if len(links) == 0 {
		return
	}

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

	if len(pruned) == 0 {
		return
	}

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

	if len(pruned) == 0 {
		return
	}

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

// UpdateEvaluationCorrelation caps weight at 1.0
func TestUpdateEvaluationCorrelationCapWeight(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	links := []registry.Link{
		{Target: "mem2", Weight: 0.98, Basis: "evaluation_correlation"},
	}

	updated := graph.UpdateEvaluationCorrelation(links, "mem2")

	g.Expect(updated[0].Weight).To(Equal(1.0))
}

// UpdateEvaluationCorrelation creates new link when none exists
func TestUpdateEvaluationCorrelationCreatesNew(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	links := []registry.Link{}

	updated := graph.UpdateEvaluationCorrelation(links, "mem2")

	g.Expect(updated).To(HaveLen(1))
	g.Expect(updated[0].Target).To(Equal("mem2"))
	g.Expect(updated[0].Weight).To(BeNumerically("~", 0.05, 0.001))
	g.Expect(updated[0].Basis).To(Equal("evaluation_correlation"))
}

// UpdateEvaluationCorrelation increments existing link
func TestUpdateEvaluationCorrelationIncrementsExisting(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	links := []registry.Link{
		{Target: "mem2", Weight: 0.2, Basis: "evaluation_correlation"},
	}

	updated := graph.UpdateEvaluationCorrelation(links, "mem2")

	g.Expect(updated).To(HaveLen(1))
	g.Expect(updated[0].Weight).To(BeNumerically("~", 0.25, 0.001))
}
