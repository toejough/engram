package cli_test

// Scenario tests for the Memory Query product capability, part 2 (861c2zm), second half: the explore half
// (792a7i to 792a7m). The clusters, the candidate notes and the superseding notes (792a7g, 792a7h, 792a7n) and the
// helpers both halves share (the mq2 prefix) are in memory_query_part2_scenarios_test.go. Spec:
// docs/specs/product/memory-query.md.
//
// 792a7i and 792a7m run cli.RunQuery over the in-memory world of part 1 with a vocab.centroids.json in the vault.
// 792a7j, 792a7k and 792a7l are about how the explore budget is divided and how the notes are picked, so they run
// the explore step of the query (cli.ExportComputeExploreHalf, the glue RunQuery calls) with the term centroids,
// their member notes and the exploit half given directly. Term centroids are unit axes (mq2Axis), so the cosine of
// a member to its centroid is the number the scenario names (mq2TermMember), and the query vector has the cosine to
// each centroid that the scenario names (mq2Query).

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"math"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// Test792a7i1a_RunQuery_TheHalvesAreEqualOnAHealthyVault verifies 792a7i1a (The halves are equal on a healthy
// vault): a phrase that matches two notes of a vault that has vocabulary centroids and further notes under a term
// gives those two notes and two explore notes, the two nearest to the term's centroid.
func Test792a7i1a_RunQuery_TheHalvesAreEqualOnAHealthyVault(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a bootstrapped vault with enough notes.
	world := mq2NewExploreWorld(2, false, []mq2Spec{
		{name: "7.go-core-a", cosine: 0.95},
		{name: "5.go-core-b", cosine: 0.90},
		{name: "6.go-fringe-a", cosine: 0.80},
		{name: "8.go-fringe-b", cosine: 0.70},
	})
	mq2WriteCentroids(t, world)

	// When the user runs engram query with a phrase that matches two notes.
	result := mq2MustRun(t, world)

	// Then the payload holds those two notes and two explore notes.
	g.Expect(result.payload.directPaths()).To(ConsistOf("1.baking-0.md", "2.baking-1.md"))
	g.Expect(result.payload.explorePaths()).To(Equal([]string{"7.go-core-a.md", "5.go-core-b.md"}))
	g.Expect(result.payload.Items).To(HaveLen(4))
}

// Test792a7i1b_RunQuery_MissingCentroidsLeaveOnlyTheMatches verifies 792a7i1b (Missing centroids leave only the
// matches): when vocab.centroids.json is absent, or present but not readable, no item has the provenance explore
// and the budget's explore_allocated is {} (present, not omitted).
func Test792a7i1b_RunQuery_MissingCentroidsLeaveOnlyTheMatches(t *testing.T) {
	t.Parallel()

	cases := map[string]func(world *mq1World){
		"absent": func(*mq1World) {},
		"unreadable": func(world *mq1World) {
			world.files.files[mq2CentroidsPath] = []byte("{ this is not json")
		},
	}

	for name, damage := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			// Given a vault whose vocab.centroids.json is absent or unreadable.
			world := mq2NewExploreWorld(2, false, []mq2Spec{{name: "3.go-core", cosine: 0.95}})
			damage(world)

			// When the user runs engram query with a phrase that matches notes.
			result := mq2MustRun(t, world)

			// Then no item has provenance explore and the budget's explore_allocated is {}.
			g.Expect(result.payload.directPaths()).To(ConsistOf("1.baking-0.md", "2.baking-1.md"))
			g.Expect(result.payload.explorePaths()).To(BeEmpty())
			g.Expect(result.payload.Budget.ExploreAllocated).NotTo(BeNil())
			g.Expect(result.payload.Budget.ExploreAllocated).To(BeEmpty())
			g.Expect(result.stdout).To(ContainSubstring("explore_allocated: {}"))
		})
	}
}

// Test792a7i1c_RunQuery_NoMatchedNoteMeansNoExploreHalf verifies 792a7i1c (No matched note means no explore
// half): a vault whose only match is a chunk, with centroids and notes under a term that the explore half would
// sample from, gives no explore item and an explore_allocated of {}.
func Test792a7i1c_RunQuery_NoMatchedNoteMeansNoExploreHalf(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a vault whose only matches are chunks.
	world := mq2NewExploreWorld(0, false, []mq2Spec{
		{name: "3.go-core-a", cosine: 0.95},
		{name: "4.go-core-b", cosine: 0.90},
	})
	mq2WriteCentroids(t, world)
	world.chunks = append(world.chunks, world.chunk(mq2AnchorOne, map[int]float64{0: mq2Main}))

	// When the user runs engram query with a phrase.
	result := mq2MustRun(t, world)

	// Then no item has provenance explore and explore_allocated is {}.
	g.Expect(result.payload.directPaths()).To(Equal([]string{mq1ChunkPath(mq2AnchorOne)}), "only the chunk matched")
	g.Expect(result.payload.explorePaths()).To(BeEmpty())
	g.Expect(result.payload.Budget.ExploreAllocated).To(BeEmpty())
	g.Expect(result.stdout).To(ContainSubstring("explore_allocated: {}"))
}

// Test792a7i1d_RunQuery_ASmallVaultShortensTheExploreHalf verifies 792a7i1d (A small vault shortens the explore
// half): with three matched notes, the explore half is empty when every note of the term is among the matches and
// one note long when one more note holds the term; either way it is smaller than the exploit half, and
// explore_allocated sums to the number of explore notes delivered.
func Test792a7i1d_RunQuery_ASmallVaultShortensTheExploreHalf(t *testing.T) {
	t.Parallel()

	const exploitNotes = 3

	cases := map[string]struct {
		members       []mq2Spec
		wantDelivered int
	}{
		"every note of the vault is a match": {members: nil, wantDelivered: 0},
		"one note is not a match": {
			members:       []mq2Spec{{name: "4.go-core", cosine: 0.95}},
			wantDelivered: 1,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			// Given a vault whose notes are all, or all but one, in the exploit half.
			world := mq2NewExploreWorld(exploitNotes, true, tc.members)
			mq2WriteCentroids(t, world)

			// When the user runs engram query with a phrase.
			result := mq2MustRun(t, world)

			// Then the explore half is smaller than the exploit half and explore_allocated sums to the number
			// delivered.
			g.Expect(result.payload.directPaths()).To(HaveLen(exploitNotes))
			g.Expect(result.payload.explorePaths()).To(HaveLen(tc.wantDelivered))
			g.Expect(len(result.payload.explorePaths())).To(BeNumerically("<", exploitNotes))
			g.Expect(mq2Sum(result.payload.Budget.ExploreAllocated)).To(Equal(tc.wantDelivered))
		})
	}
}

// Test792a7j1a_ComputeExploreHalf_NearerTermsSampleMore verifies 792a7j1a (Nearer terms sample more): of two terms
// whose centroids have cosines 0.5 and 0.3 to the query, the nearer is allocated at least as much as the other
// and together they get the whole budget; the instance is chosen so the softmax at temperature 0.106 gives 7 and
// 1 of 8, and the nearer term's name sorts after the other's, so the name order cannot produce the result.
func Test792a7j1a_ComputeExploreHalf_NearerTermsSampleMore(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const budget = 8

	// Given two terms, one with a centroid nearer the query.
	terms := []mq2ExploreTerm{
		{name: "zulu-near", axis: 1, members: mq2TermMembers("near", 1, mq2Cosines(10))},
		{name: "alpha-far", axis: 2, members: mq2TermMembers("far", 2, mq2Cosines(10))},
	}

	// When the explore budget is divided.
	_, delivered := mq2ComputeExplore(t, terms, mq2Query(0.5, 0.3), nil, budget)

	// Then the nearer term's allocation is at least the other's, and together they equal the budget.
	g.Expect(delivered["zulu-near"]).To(BeNumerically(">=", delivered["alpha-far"]))
	g.Expect(mq2Sum(delivered)).To(Equal(budget))
	g.Expect(delivered).To(Equal(map[string]int{"zulu-near": 7, "alpha-far": 1}))
}

// Test792a7j1b_ComputeExploreHalf_EvidenceLiftsADistantTerm verifies 792a7j1b (Evidence lifts a distant term): of
// two terms equally distant from the query (cosine 0.3), the one with a member in the exploit half is allocated
// more (6 of 10, against 4); the term with the member sorts after the other by name, so a tie would favour the
// other.
func Test792a7j1b_ComputeExploreHalf_EvidenceLiftsADistantTerm(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const budget = 10

	// Given a distant term with one member in the exploit half and an equally distant term with none.
	terms := []mq2ExploreTerm{
		{name: "zulu-evidenced", axis: 1, members: mq2TermMembers("evidenced", 1, mq2Cosines(11))},
		{name: "alpha-bare", axis: 2, members: mq2TermMembers("bare", 2, mq2Cosines(10))},
	}
	exploit := []string{terms[0].members[0].NotePath}

	// When the explore budget is divided.
	_, delivered := mq2ComputeExplore(t, terms, mq2Query(0.3, 0.3), exploit, budget)

	// Then the term with a member is allocated more.
	g.Expect(delivered["zulu-evidenced"]).To(BeNumerically(">", delivered["alpha-bare"]))
	g.Expect(delivered).To(Equal(map[string]int{"zulu-evidenced": 6, "alpha-bare": 4}))
}

// Test792a7j1c_ComputeExploreHalf_TheBonusDoesNotStack verifies 792a7j1c (The bonus does not stack): of two terms
// of equal distance from the query, one with one member in the exploit half and one with three, the two are
// allocated the same (4 and 4 of 8).
func Test792a7j1c_ComputeExploreHalf_TheBonusDoesNotStack(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		budget       = 8
		threeMembers = 3
	)

	// Given two terms of equal distance, one with one member in the exploit half and one with three.
	terms := []mq2ExploreTerm{
		{name: "zulu-three", axis: 1, members: mq2TermMembers("three", 1, mq2Cosines(budget+threeMembers))},
		{name: "alpha-one", axis: 2, members: mq2TermMembers("one", 2, mq2Cosines(budget+1))},
	}

	exploit := make([]string, 0, threeMembers+1)
	for _, member := range terms[0].members[:threeMembers] {
		exploit = append(exploit, member.NotePath)
	}

	exploit = append(exploit, terms[1].members[0].NotePath)

	// When the explore budget is divided.
	_, delivered := mq2ComputeExplore(t, terms, mq2Query(0.3, 0.3), exploit, budget)

	// Then the two are allocated the same.
	g.Expect(delivered["zulu-three"]).To(Equal(delivered["alpha-one"]))
	g.Expect(delivered).To(Equal(map[string]int{"zulu-three": 4, "alpha-one": 4}))
}

// Test792a7k1a_ComputeExploreHalf_TheCoreOutranksTheFringe verifies 792a7k1a (The core outranks the fringe): from
// a term with five member notes and an allocation of two, the notes picked are the two members closest to the
// term's centroid, nearest first; their paths are not in that order.
func Test792a7k1a_ComputeExploreHalf_TheCoreOutranksTheFringe(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const budget = 2

	// Given a term with five member notes and an allocation of two.
	terms := []mq2ExploreTerm{{name: mq2Term, axis: 1, members: []cli.ExportVaultTermMember{
		mq2TermMember("1.k-fringe", 1, 0.60),
		mq2TermMember("2.k-core-b", 1, 0.95),
		mq2TermMember("3.k-mid", 1, 0.80),
		mq2TermMember("4.k-core-a", 1, 0.98),
		mq2TermMember("5.k-edge", 1, 0.55),
	}}}

	// When the explore notes are picked.
	picks, delivered := mq2ComputeExplore(t, terms, mq2Query(0.7), nil, budget)

	// Then they are the two members closest to the term's centroid.
	g.Expect(mq2PickPaths(picks)).To(Equal([]string{"4.k-core-a", "2.k-core-b"}))
	g.Expect(delivered).To(Equal(map[string]int{mq2Term: budget}))

	if len(picks) != budget {
		return
	}

	g.Expect(float64(picks[0].Cosine)).To(BeNumerically("~", 0.98, mq2Tolerance))
	g.Expect(float64(picks[1].Cosine)).To(BeNumerically("~", 0.95, mq2Tolerance))
}

// Test792a7k1b_ComputeExploreHalf_DefinitionNotesAreNeverPicked verifies 792a7k1b (Definition notes are never
// picked): a term whose definition note, in the shape the vocabulary gives it, is the closest note to its
// centroid (cosine 1) contributes the next two members instead.
func Test792a7k1b_ComputeExploreHalf_DefinitionNotesAreNeverPicked(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const budget = 2

	// Given a term whose definition note is the closest note to its centroid.
	definition := mq2TermMember("1.definition", 1, 1)
	definition.Content = mq1DefinitionContent(mq2Term)

	terms := []mq2ExploreTerm{{name: mq2Term, axis: 1, members: []cli.ExportVaultTermMember{
		definition,
		mq2TermMember("2.k-a", 1, 0.90),
		mq2TermMember("3.k-b", 1, 0.80),
		mq2TermMember("4.k-c", 1, 0.70),
	}}}

	// When the explore notes are picked.
	picks, delivered := mq2ComputeExplore(t, terms, mq2Query(0.7), nil, budget)

	// Then the definition note is not among them.
	g.Expect(mq2PickPaths(picks)).To(Equal([]string{"2.k-a", "3.k-b"}))
	g.Expect(delivered).To(Equal(map[string]int{mq2Term: budget}))
}

// Test792a7l1a_ComputeExploreHalf_ADuplicateIsReplaced verifies 792a7l1a (A duplicate is replaced): when the one
// pick of a term is already in the exploit half, it is dropped and the freed slot is filled from the next term in
// allocation order (the nearer term, which sorts last by name), which gives its second member.
func Test792a7l1a_ComputeExploreHalf_ADuplicateIsReplaced(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const budget = 3

	// Given an explore pick that is already in the exploit half, and two other terms with unpicked members, the
	// nearer of which sorts after the other by name (so the allocation order and the name order disagree).
	terms := []mq2ExploreTerm{
		{name: "alpha-dup", axis: 1, members: []cli.ExportVaultTermMember{mq2TermMember("x1", 1, 0.99)}},
		{name: "bravo-after", axis: 3, members: []cli.ExportVaultTermMember{
			mq2TermMember("b1", 3, 0.99), mq2TermMember("b2", 3, 0.98),
		}},
		{name: "zulu-next", axis: 2, members: []cli.ExportVaultTermMember{
			mq2TermMember("z1", 2, 0.99), mq2TermMember("z2", 2, 0.98), mq2TermMember("z3", 2, 0.97),
		}},
	}

	// When the explore notes are picked (the cosines to alpha, zulu and bravo are 0.40, 0.41 and 0.40, so the
	// allocation is one note each and the evidence bonus puts alpha first, then zulu, then bravo).
	picks, delivered := mq2ComputeExplore(t, terms, mq2Query(0.40, 0.41, 0.40), []string{"x1"}, budget)

	// Then the duplicate is dropped and its slot is filled from the next term in allocation order (zulu, which
	// is nearer, although bravo sorts first by name).
	g.Expect(mq2PickPaths(picks)).To(ConsistOf("z1", "b1", "z2"))
	g.Expect(delivered).To(Equal(map[string]int{"zulu-next": 2, "bravo-after": 1}))
}

// Test792a7l1b_ComputeExploreHalf_ANoteUnderTwoTermsIsPickedOnce verifies 792a7l1b (A note under two terms is
// picked once): a note that is the nearest member of two allocated terms is picked once, and the other term takes
// its next member so the budget is still met.
func Test792a7l1b_ComputeExploreHalf_ANoteUnderTwoTermsIsPickedOnce(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const budget = 2

	// Given a note that is the nearest member of two allocated terms.
	shared := cli.ExportVaultTermMember{
		NotePath: "shared", Content: mq1FactContent("shared"), Vector: mq2Vec(0, math.Sqrt2/2, math.Sqrt2/2, 0),
	}
	terms := []mq2ExploreTerm{
		{name: "alpha-first", axis: 1, members: []cli.ExportVaultTermMember{shared, mq2TermMember("a2", 1, 0.60)}},
		{name: "bravo-second", axis: 2, members: []cli.ExportVaultTermMember{shared, mq2TermMember("b2", 2, 0.50)}},
	}

	// When the explore notes are picked.
	picks, delivered := mq2ComputeExplore(t, terms, mq2Query(0.4, 0.4), nil, budget)

	// Then it appears once.
	g.Expect(mq2PickPaths(picks)).To(ConsistOf("shared", "b2"))
	g.Expect(mq2Sum(delivered)).To(Equal(budget))
}

// Test792a7l1c_ComputeExploreHalf_ExhaustedCandidatesLeaveTheBudgetUnmet verifies 792a7l1c (Exhausted candidates
// leave the budget unmet): with a budget of 5 and only three notes eligible (a fourth is among the matches and a
// fifth is the term's definition note), every eligible note is picked and the half is smaller than its budget.
func Test792a7l1c_ComputeExploreHalf_ExhaustedCandidatesLeaveTheBudgetUnmet(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const (
		budget   = 5
		eligible = 3
	)

	// Given fewer eligible notes than the explore budget.
	definition := mq2TermMember("5.definition", 1, 1)
	definition.Content = mq1DefinitionContent(mq2Term)

	terms := []mq2ExploreTerm{{name: mq2Term, axis: 1, members: []cli.ExportVaultTermMember{
		mq2TermMember("1.e-a", 1, 0.90),
		mq2TermMember("2.e-b", 1, 0.80),
		mq2TermMember("3.e-c", 1, 0.70),
		mq2TermMember("4.matched", 1, 0.95),
		definition,
	}}}

	// When the explore notes are picked.
	picks, delivered := mq2ComputeExplore(t, terms, mq2Query(0.7), []string{"4.matched"}, budget)

	// Then every eligible note is picked and the half is smaller than its budget.
	g.Expect(mq2PickPaths(picks)).To(Equal([]string{"1.e-a", "2.e-b", "3.e-c"}))
	g.Expect(len(picks)).To(BeNumerically("<", budget))
	g.Expect(delivered).To(Equal(map[string]int{mq2Term: eligible}))
}

// Test792a7m1a_RunQuery_AnExploreNoteNamesItsTerm verifies 792a7m1a (An explore note names its term): a phrase
// about cooking, over a vault with the term go-tooling, gives explore items with the provenance explore, the
// source_term go-tooling and a score equal to the cosine of their vector to the term's centroid.
func Test792a7m1a_RunQuery_AnExploreNoteNamesItsTerm(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a bootstrapped vault with a term go-tooling.
	members := []mq2Spec{
		{name: "3.go-core-a", cosine: 0.80},
		{name: "4.go-core-b", cosine: 0.60},
		{name: "5.go-fringe", cosine: 0.50},
	}
	world := mq2NewExploreWorld(2, false, members)
	mq2WriteCentroids(t, world)

	// When the user runs engram query with a phrase about cooking.
	result := mq2MustRun(t, world)

	// Then an item has provenance explore, source_term go-tooling and a score equal to its cosine to that term's
	// centroid.
	explored := result.payload.exploreItems()
	g.Expect(explored).To(HaveLen(2))

	for _, item := range explored {
		g.Expect(item.Provenances).To(Equal([]string{"explore"}))
		g.Expect(item.SourceTerm).To(Equal(mq2Term))
	}

	for index, spec := range members[:len(explored)] {
		g.Expect(explored[index].Path).To(Equal(spec.name + ".md"))

		score := float64(explored[index].Score)
		g.Expect(score).To(BeNumerically("~", mq2Cosine(mq2Centroid, spec.vec()), mq2Tolerance))
		g.Expect(score).To(BeNumerically("~", spec.cosine, mq2Tolerance))
	}
}

// Test792a7m1b_RunQuery_ExploreNotesAreOutsideTheClusters verifies 792a7m1b (Explore notes are outside the
// clusters): in a payload that has an explore note and two clusters, the explore note is not a member of any
// cluster and is in no cluster's candidate_l2s.
func Test792a7m1b_RunQuery_ExploreNotesAreOutsideTheClusters(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Given a payload with an explore note.
	world := newMq1World(mq2Dims, mq2GoPhrase)

	for index, deviation := range []float32{0, 0.02, -0.04, 0.06} {
		mq2PlantNote(world, fmt.Sprintf("%d.match-%d", index+1, index), mq2Vec(mq2Main, 0, deviation, 0), "")
	}

	mq2PlantOtherTopic(world, mq2OtherTopicSize)
	mq2PlantNote(world, "9.explore", mq2Spec{cosine: 0.8}.vec(), mq2TagLine())
	mq2WriteCentroids(t, world)

	result := mq2MustRun(t, world)

	g.Expect(result.payload.explorePaths()).To(Equal([]string{"9.explore.md"}))
	g.Expect(result.payload.Clusters).To(HaveLen(2))

	// When its clusters are read.
	var (
		members    []string
		candidates []string
	)

	for _, group := range result.payload.Clusters {
		members = append(members, group.memberPaths()...)
		candidates = append(candidates, group.candidatePaths()...)
	}

	// Then the explore note is not a member of any cluster and not in any candidate_l2s.
	g.Expect(candidates).NotTo(BeEmpty(), "the clusters do list candidates")
	g.Expect(members).NotTo(ContainElement("9.explore.md"))
	g.Expect(candidates).NotTo(ContainElement("9.explore.md"))
}

// Test792a7m1c_RunQuery_TheBudgetCountsWhatWasDelivered verifies 792a7m1c (The budget counts what was delivered):
// a payload whose explore half holds two notes of one term maps that term to 2 in explore_allocated, and when the
// term has one eligible note for the two slots it maps the term to 1, the number delivered.
func Test792a7m1c_RunQuery_TheBudgetCountsWhatWasDelivered(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		members []mq2Spec
		want    int
	}{
		"two notes delivered": {
			members: []mq2Spec{
				{name: "3.go-core-a", cosine: 0.90},
				{name: "4.go-core-b", cosine: 0.80},
				{name: "5.go-fringe", cosine: 0.50},
			},
			want: 2,
		},
		"one note delivered of two slots": {
			members: []mq2Spec{{name: "3.go-core-a", cosine: 0.90}},
			want:    1,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			// Given a payload whose explore half holds the notes of one term.
			world := mq2NewExploreWorld(2, false, tc.members)
			mq2WriteCentroids(t, world)

			result := mq2MustRun(t, world)

			// When its budget is read.
			allocated := result.payload.Budget.ExploreAllocated

			// Then explore_allocated maps that term to the number of its notes delivered.
			g.Expect(result.payload.explorePaths()).To(HaveLen(tc.want))
			g.Expect(allocated).To(Equal(map[string]int{mq2Term: tc.want}))
		})
	}
}

// unexported constants.
const (
	mq2CentroidsPath = mq1Vault + "/vocab.centroids.json"
	mq2Term          = "go-tooling"
)

// unexported variables.
var (
	// mq2Centroid is the centroid of the term mq2Term in the end-to-end scenarios: the unit vector of axis 1.
	mq2Centroid = mq2Vec(0, 1, 0, 0)
)

// mq2ExploreTerm is a vocabulary term of the direct explore scenarios: its centroid is the unit vector of the axis.
type mq2ExploreTerm struct {
	name    string
	axis    int
	members []cli.ExportVaultTermMember
}

// mq2Spec is a note of the term mq2Term whose vector has the cosine to the term's centroid.
type mq2Spec struct {
	name   string
	cosine float64
}

// vec returns the vector of the note: the cosine on axis 1 (the centroid axis) and the rest of the unit length
// on axis 2, so its cosine to the first phrase's axis is 0 and the note is never a match.
func (s mq2Spec) vec() []float32 {
	return mq2Vec(0, float32(s.cosine), float32(math.Sqrt(1-s.cosine*s.cosine)), 0)
}

// mq2Axis returns the unit vector of the axis.
func mq2Axis(axis int) []float32 {
	vec := make([]float32, mq2Dims)
	vec[axis] = 1

	return vec
}

// mq2ComputeExplore runs the explore step of the query over the terms: the vault holds a vocab.centroids.json
// with their centroids, the query vector is the given vector, the exploit half is the given paths and the budget
// is the number of notes to pick.
func mq2ComputeExplore(
	t *testing.T,
	terms []mq2ExploreTerm,
	query []float32,
	exploit []string,
	budget int,
) ([]cli.ExportExplorePick, map[string]int) {
	t.Helper()

	const vault = "/vault"

	entries := make(map[string]cli.ExportVocabCentroidEntry, len(terms))
	index := make(map[string][]cli.ExportVaultTermMember, len(terms))

	for _, term := range terms {
		entries[term.name] = cli.ExportVocabCentroidEntry{Vector: mq2Axis(term.axis), MemberCount: len(term.members)}
		index[term.name] = term.members
	}

	data, err := json.Marshal(cli.ExportVocabCentroidsDoc{SchemaVersion: 1, Dims: mq2Dims, Terms: entries})
	if err != nil {
		t.Fatalf("encoding the centroids: %v", err)
	}

	readFile := func(path string) ([]byte, error) {
		if path == filepath.Join(vault, "vocab.centroids.json") {
			return data, nil
		}

		return nil, fs.ErrNotExist
	}

	exploitPaths := make(map[string]struct{}, len(exploit))
	for _, path := range exploit {
		exploitPaths[path] = struct{}{}
	}

	meta := cli.ExportNewVaultNotesMetaWithTerms(index)

	return cli.ExportComputeExploreHalf(vault, readFile, query, exploitPaths, budget, meta)
}

// mq2Cosines returns count cosines, descending from 0.99 by 0.01.
func mq2Cosines(count int) []float64 {
	cosines := make([]float64, count)

	for index := range cosines {
		cosines[index] = 0.99 - 0.01*float64(index)
	}

	return cosines
}

// mq2NewExploreWorld returns a world whose one phrase (axis 0) matches count notes, tagged with the term when
// tagged is true, and which holds the given further notes of the term mq2Term. No vocab.centroids.json is written.
func mq2NewExploreWorld(count int, tagged bool, members []mq2Spec) *mq1World {
	world := newMq1World(mq2Dims, mq2BakingPhrase)

	tag := ""
	if tagged {
		tag = mq2TagLine()
	}

	for index := range count {
		vec := world.cosines(map[int]float64{0: mq2Main - 0.01*float64(index)})
		mq2PlantNote(world, fmt.Sprintf("%d.baking-%d", index+1, index), vec, tag)
	}

	for _, member := range members {
		mq2PlantNote(world, member.name, member.vec(), mq2TagLine())
	}

	return world
}

// mq2PickPaths returns the paths of the picks, in order.
func mq2PickPaths(picks []cli.ExportExplorePick) []string {
	paths := make([]string, 0, len(picks))

	for _, pick := range picks {
		paths = append(paths, pick.Path)
	}

	return paths
}

// mq2Query returns a unit query vector whose cosine to the unit vector of axis k (k from 1) is the k-th cosine;
// the rest of the unit length is on axis 0.
func mq2Query(cosines ...float64) []float32 {
	vec := make([]float32, mq2Dims)

	var squares float64

	for index, cosine := range cosines {
		vec[index+1] = float32(cosine)
		squares += cosine * cosine
	}

	vec[0] = float32(math.Sqrt(1 - squares))

	return vec
}

// mq2Sum adds the values of the map.
func mq2Sum(counts map[string]int) int {
	total := 0

	for _, count := range counts {
		total += count
	}

	return total
}

// mq2TagLine returns the frontmatter line that tags a note with the term mq2Term.
func mq2TagLine() string {
	return "tags:\n    - vocab/" + mq2Term + "\n"
}

// mq2TermMember returns a member note of a term whose centroid is the unit vector of the axis, with the given cosine to
// that centroid (the rest of the unit length is on axis 0, which no centroid uses).
func mq2TermMember(path string, axis int, cosine float64) cli.ExportVaultTermMember {
	vec := make([]float32, mq2Dims)
	vec[axis] = float32(cosine)
	vec[0] = float32(math.Sqrt(1 - cosine*cosine))

	return cli.ExportVaultTermMember{NotePath: path, Content: mq1FactContent(path), Vector: vec}
}

// mq2TermMembers returns one member per cosine, named <prefix>-<index>.
func mq2TermMembers(prefix string, axis int, cosines []float64) []cli.ExportVaultTermMember {
	members := make([]cli.ExportVaultTermMember, 0, len(cosines))

	for index, cosine := range cosines {
		members = append(members, mq2TermMember(fmt.Sprintf("%s-%02d", prefix, index), axis, cosine))
	}

	return members
}

// mq2WriteCentroids writes the vocab.centroids.json of the world: the term mq2Term with the centroid mq2Centroid.
func mq2WriteCentroids(t *testing.T, world *mq1World) {
	t.Helper()

	data, err := json.Marshal(cli.ExportVocabCentroidsDoc{
		SchemaVersion:    1,
		EmbeddingModelID: mq1ModelID,
		Dims:             mq2Dims,
		Terms:            map[string]cli.ExportVocabCentroidEntry{mq2Term: {Vector: mq2Centroid, MemberCount: 1}},
	})
	if err != nil {
		t.Fatalf("encoding the centroids: %v", err)
	}

	world.files.files[mq2CentroidsPath] = data
}
