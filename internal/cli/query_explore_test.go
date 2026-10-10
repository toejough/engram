package cli_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
)

// Test792b24c13a1_Property_SoftmaxAllocate_BudgetConservation locks task 1.1's second
// property: allocations always sum exactly to the explore budget, across an
// arbitrary term set and temperature.
func Test792b24c13a1_Property_SoftmaxAllocate_BudgetConservation(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		const (
			minTerms  = 1
			maxTerms  = 12
			minBudget = 0
			maxBudget = 40
			minTemp   = 0.01
			maxTemp   = 2.0
			minSim    = -1.0
			maxSim    = 1.0
		)

		termCount := rapid.IntRange(minTerms, maxTerms).Draw(rt, "termCount")
		budget := rapid.IntRange(minBudget, maxBudget).Draw(rt, "budget")
		temperature := float32(rapid.Float64Range(minTemp, maxTemp).Draw(rt, "temperature"))

		similarities := make(map[string]float32, termCount)
		for i := range termCount {
			term := rapid.StringMatching(`term[0-9]`).Draw(rt, "term")
			sim := rapid.Float64Range(minSim, maxSim).Draw(rt, "sim")
			similarities[term+string(rune('a'+i))] = float32(sim)
		}

		allocated := cli.ExportSoftmaxAllocate(similarities, temperature, budget)

		total := 0
		for _, count := range allocated {
			total += count
		}

		g.Expect(total).To(Equal(budget))
	})
}

// Test792b24c13a1_SoftmaxAllocate_BudgetConservation locks task 1.1: allocations across
// terms must sum exactly to the explore budget via largest-remainder
// apportionment.
func Test792b24c13a1_SoftmaxAllocate_BudgetConservation(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	similarities := map[string]float32{
		"alpha": 0.9,
		"beta":  0.5,
		"gamma": 0.1,
	}

	const budget = 7

	allocated := cli.ExportSoftmaxAllocate(similarities, cli.ExportExploreTemperatureDefault, budget)

	total := 0
	for _, count := range allocated {
		total += count
	}

	g.Expect(total).To(Equal(budget))
}

// Test792b24c13a2_SoftmaxAllocate_HigherSimilarityGetsMoreOrEqual is an explicit example
// of the monotonicity property backing Test792b24c13a3_Property_SoftmaxAllocate_Monotonic.
func Test792b24c13a2_SoftmaxAllocate_HigherSimilarityGetsMoreOrEqual(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	similarities := map[string]float32{
		"near": 0.95,
		"far":  0.05,
	}

	allocated := cli.ExportSoftmaxAllocate(similarities, cli.ExportExploreTemperatureDefault, 10)

	g.Expect(allocated["near"]).To(BeNumerically(">=", allocated["far"]))
}

// Test792b24c13a3_Property_SoftmaxAllocate_Monotonic locks the monotonicity property from
// tasks.md 1.1: sim_A >= sim_B => allocation(A) >= allocation(B) for
// un-boosted terms. Similarities are drawn strictly ordered to avoid the
// undefined case of two terms tying exactly (a tie forces the
// largest-remainder tiebreak to favor exactly one, which does not
// contradict >= but is easiest reasoned about with distinct draws).
func Test792b24c13a3_Property_SoftmaxAllocate_Monotonic(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		const (
			minBudget = 0
			maxBudget = 50
			minSim    = -1.0
			maxSim    = 1.0
			simGap    = 0.001
		)

		simB := rapid.Float64Range(minSim, maxSim-simGap).Draw(rt, "simB")
		simA := rapid.Float64Range(simB+simGap, maxSim).Draw(rt, "simA")
		budget := rapid.IntRange(minBudget, maxBudget).Draw(rt, "budget")

		similarities := map[string]float32{
			"termA": float32(simA),
			"termB": float32(simB),
		}

		allocated := cli.ExportSoftmaxAllocate(similarities, cli.ExportExploreTemperatureDefault, budget)

		g.Expect(allocated["termA"]).To(BeNumerically(">=", allocated["termB"]))
	})
}

// Test792b24c13a4_SoftmaxAllocate_NoTermsYieldsEmpty covers a vault with no centroids at
// all (missing/unreadable vocab.centroids.json degrades to exploit-only).
func Test792b24c13a4_SoftmaxAllocate_NoTermsYieldsEmpty(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	allocated := cli.ExportSoftmaxAllocate(map[string]float32{}, cli.ExportExploreTemperatureDefault, 5)

	g.Expect(allocated).To(BeEmpty())
}

// Test792b24c13a4_SoftmaxAllocate_ZeroBudgetYieldsEmpty locks the B=0 skip case from
// design.md Decision 1: no notes matched -> explore half skipped entirely.
func Test792b24c13a4_SoftmaxAllocate_ZeroBudgetYieldsEmpty(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	similarities := map[string]float32{"alpha": 0.9}

	allocated := cli.ExportSoftmaxAllocate(similarities, cli.ExportExploreTemperatureDefault, 0)

	g.Expect(allocated).To(BeEmpty())
}

// Test792b24c13a5_ApplyMatchEvidenceBonus_FlatNonStacking locks task 1.2: a term with
// >=1 member in the exploit half gets the fixed delta bonus once, never
// stacked per member.
func Test792b24c13a5_ApplyMatchEvidenceBonus_FlatNonStacking(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	similarities := map[string]float32{
		"evidenced":   0.2,
		"unevidenced": 0.2,
	}
	evidenced := map[string]bool{"evidenced": true}

	const bonus = float32(0.05)

	boosted := cli.ExportApplyMatchEvidenceBonus(similarities, evidenced, bonus)

	g.Expect(boosted["evidenced"]).To(BeNumerically("~", 0.25, 1e-6))
	g.Expect(boosted["unevidenced"]).To(BeNumerically("~", 0.2, 1e-6))
}

// Test792b24c13a6_ApplyMatchEvidenceBonus_LiftsDistantClusterAboveUnevidenced covers the
// design.md 3.1 calibration bar: the bonus must be able to lift a
// vault-median-sim evidenced cluster above an un-evidenced cluster at
// sim_med + delta/2.
func Test792b24c13a6_ApplyMatchEvidenceBonus_LiftsDistantClusterAboveUnevidenced(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		simMed = float32(0.30)
		bonus  = float32(0.05)
	)

	similarities := map[string]float32{
		"evidenced":   simMed,
		"unevidenced": simMed + bonus/2,
	}
	evidenced := map[string]bool{"evidenced": true}

	boosted := cli.ExportApplyMatchEvidenceBonus(similarities, evidenced, bonus)

	g.Expect(boosted["evidenced"]).To(BeNumerically(">", boosted["unevidenced"]))
}

// Test792b24c13a7_ApplyMatchEvidenceBonus_NeverReordersTwoUnevidencedTerms locks the
// second calibration bar: the additive bonus never reorders two unevidenced
// clusters relative to each other.
func Test792b24c13a7_ApplyMatchEvidenceBonus_NeverReordersTwoUnevidencedTerms(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		const minSim, maxSim = -1.0, 1.0

		simA := rapid.Float64Range(minSim, maxSim).Draw(rt, "simA")
		simB := rapid.Float64Range(minSim, maxSim).Draw(rt, "simB")
		bonus := float32(rapid.Float64Range(0, 0.5).Draw(rt, "bonus"))

		similarities := map[string]float32{
			"termA": float32(simA),
			"termB": float32(simB),
		}

		boosted := cli.ExportApplyMatchEvidenceBonus(similarities, map[string]bool{}, bonus)

		if simA >= simB {
			g.Expect(boosted["termA"]).To(BeNumerically(">=", boosted["termB"]))
		} else {
			g.Expect(boosted["termB"]).To(BeNumerically(">=", boosted["termA"]))
		}
	})
}

// Test792b24c13a8_TermsWithExploitEvidence_MembersInExploitHalf locks the helper that
// derives the evidenced-term set applyMatchEvidenceBonus needs: a term
// qualifies when at least one of its member notes' paths appears in the
// exploit-half path set.
func Test792b24c13a8_TermsWithExploitEvidence_MembersInExploitHalf(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	members := map[string][]cli.ExportExploreCandidate{
		"alpha": {{Path: "a.md"}, {Path: "b.md"}},
		"beta":  {{Path: "c.md"}},
	}
	exploitPaths := map[string]struct{}{"b.md": {}}

	evidenced := cli.ExportTermsWithExploitEvidence(members, exploitPaths)

	g.Expect(evidenced).To(HaveKeyWithValue("alpha", true))
	g.Expect(evidenced).NotTo(HaveKey("beta"))
}

// Test792b24c14a1_SelectWithinCluster_DescendingCentroidCosine locks task 1.3: within a
// term's members, the top-k selected are those with highest cosine to the
// term centroid, descending.
func Test792b24c14a1_SelectWithinCluster_DescendingCentroidCosine(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	centroid := []float32{1, 0}
	members := []cli.ExportExploreCandidate{
		{Path: "far.md", Vector: []float32{0, 1}},
		{Path: "near.md", Vector: []float32{1, 0}},
		{Path: "mid.md", Vector: []float32{1, 1}},
	}

	picks := cli.ExportSelectWithinCluster("shape", centroid, members, 2)

	g.Expect(picks).To(HaveLen(2))
	g.Expect(picks[0].Path).To(Equal("near.md"))
	g.Expect(picks[1].Path).To(Equal("mid.md"))
	g.Expect(picks[0].SourceTerm).To(Equal("shape"))
	g.Expect(picks[0].Cosine).To(BeNumerically(">", picks[1].Cosine))
}

// Test792b24c14a2_SelectWithinCluster_KGreaterThanMembersReturnsAll covers the
// candidates-exhausted edge used by dedupe+backfill (task 1.4).
func Test792b24c14a2_SelectWithinCluster_KGreaterThanMembersReturnsAll(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	centroid := []float32{1, 0}
	members := []cli.ExportExploreCandidate{{Path: "only.md", Vector: []float32{1, 0}}}

	picks := cli.ExportSelectWithinCluster("shape", centroid, members, 5)

	g.Expect(picks).To(HaveLen(1))
}

// Test792b24c14a3_SelectWithinCluster_ZeroKReturnsEmpty covers a term with zero
// allocation — no candidates should be selected.
func Test792b24c14a3_SelectWithinCluster_ZeroKReturnsEmpty(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	picks := cli.ExportSelectWithinCluster("shape", []float32{1, 0}, nil, 0)

	g.Expect(picks).To(BeEmpty())
}

// Test792b24c15a1_DedupeAndBackfill_DropsCrossClusterDuplicate locks the cross-cluster
// half of task 1.4: the same note tagged under two terms is only selected
// once; the second cluster's slot backfills from its own next member.
func Test792b24c15a1_DedupeAndBackfill_DropsCrossClusterDuplicate(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	centroid := []float32{1, 0}
	shared := cli.ExportExploreCandidate{Path: "shared.md", Vector: []float32{1, 0}}
	other := cli.ExportExploreCandidate{Path: "other.md", Vector: []float32{1, 0}}

	rankedByTerm := map[string][]cli.ExportExplorePick{
		"termA": cli.ExportSelectWithinCluster("termA", centroid, []cli.ExportExploreCandidate{shared}, 1),
		"termB": cli.ExportSelectWithinCluster("termB", centroid, []cli.ExportExploreCandidate{shared, other}, 2),
	}

	allocated := map[string]int{"termA": 1, "termB": 1}
	order := []string{"termA", "termB"}

	picks := cli.ExportDedupeAndBackfill(allocated, order, rankedByTerm, map[string]struct{}{})

	paths := make([]string, 0, len(picks))
	for _, p := range picks {
		paths = append(paths, p.Path)
	}

	g.Expect(paths).To(ConsistOf("shared.md", "other.md"))
}

// Test792b24c15a2_DedupeAndBackfill_DropsExploitHalfDuplicate locks task 1.4: a pick
// duplicating an exploit-half note is dropped and the freed slot refills
// from the next-highest-weight term with remaining candidates.
func Test792b24c15a2_DedupeAndBackfill_DropsExploitHalfDuplicate(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	centroid := []float32{1, 0}
	rankedByTerm := map[string][]cli.ExportExplorePick{
		"near": cli.ExportSelectWithinCluster("near", centroid, []cli.ExportExploreCandidate{
			{Path: "exploit.md", Vector: []float32{1, 0}},
		}, 1),
		"far": cli.ExportSelectWithinCluster("far", centroid, []cli.ExportExploreCandidate{
			{Path: "backfill.md", Vector: []float32{1, 0}},
		}, 1),
	}

	allocated := map[string]int{"near": 1, "far": 0}
	order := []string{"near", "far"}
	exploitPaths := map[string]struct{}{"exploit.md": {}}

	picks := cli.ExportDedupeAndBackfill(allocated, order, rankedByTerm, exploitPaths)

	g.Expect(picks).To(HaveLen(1))
	g.Expect(picks[0].Path).To(Equal("backfill.md"))
	g.Expect(picks[0].SourceTerm).To(Equal("far"))
}

// Test792b24c15a3_DedupeAndBackfill_ExhaustedCandidatesLeaveBudgetUnmet covers the
// candidates-exhausted terminal case: when the vault lacks enough distinct
// notes, the delivered explore half is smaller than the budget rather than
// erroring (spec: "MAY be smaller ... shortfall SHALL be visible").
func Test792b24c15a3_DedupeAndBackfill_ExhaustedCandidatesLeaveBudgetUnmet(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	centroid := []float32{1, 0}
	rankedByTerm := map[string][]cli.ExportExplorePick{
		"only": cli.ExportSelectWithinCluster("only", centroid, []cli.ExportExploreCandidate{
			{Path: "sole.md", Vector: []float32{1, 0}},
		}, 1),
	}

	allocated := map[string]int{"only": 3}
	order := []string{"only"}

	picks := cli.ExportDedupeAndBackfill(allocated, order, rankedByTerm, map[string]struct{}{})

	g.Expect(picks).To(HaveLen(1))
}
