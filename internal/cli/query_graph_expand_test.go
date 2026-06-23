package cli //nolint:testpackage // exercises unexported graphBridgeBasenames

import (
	"strconv"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/embed"
	"github.com/toejough/engram/internal/vaultgraph"
)

func TestAppendGraphBridges_AppendsBridgesAndHonoursDisable(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	notes := []vaultgraph.Note{
		{Basename: "seed", Outgoing: []string{"bridge"}},
		{Basename: "bridge"},
	}
	hits := []compatibleSidecar{
		{note: notes[0], sidecar: embed.Sidecar{SituationVector: []float32{1, 0}, BodyVector: []float32{1, 0}}},
		{note: notes[1], sidecar: embed.Sidecar{SituationVector: []float32{0, 1}, BodyVector: []float32{0, 1}}},
	}
	noteUnion := []scoredCandidate{{basename: "seed"}}

	disabled := matchedSet{members: []matchedMember{{basename: "seed"}}}
	appendGraphBridges(&disabled, notes, hits, noteUnion, -1)
	g.Expect(disabled.members).To(HaveLen(1)) // negative hops disables expansion

	enabled := matchedSet{members: []matchedMember{{basename: "seed"}}}
	appendGraphBridges(&enabled, notes, hits, noteUnion, 1)
	g.Expect(enabled.members).To(HaveLen(2)) // bridge appended at 1 hop
	g.Expect(enabled.members[1].basename).To(Equal("bridge"))
	g.Expect(enabled.members[1].graphExpanded).To(BeTrue())
}

func TestBuildBridgeMembers_IncludesSidecaredSkipsRest(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	hitByBasename := map[string]compatibleSidecar{
		"has-vec": {
			note:    vaultgraph.Note{Basename: "has-vec"},
			sidecar: embed.Sidecar{SituationVector: []float32{0, 1}, BodyVector: []float32{1, 0}},
		},
	}

	members := buildBridgeMembers([]string{"has-vec", "no-vec"}, hitByBasename)

	g.Expect(members).To(HaveLen(1)) // sidecar-less bridge skipped
	g.Expect(members[0].basename).To(Equal("has-vec"))
	g.Expect(members[0].vector).To(Equal([]float32{0, 1})) // cluster coord = situation axis
	g.Expect(members[0].bodyVec).To(Equal([]float32{1, 0}))
	g.Expect(members[0].score).To(BeNumerically("==", 0)) // no query cosine
	g.Expect(members[0].graphExpanded).To(BeTrue())
}

func TestGraphBridgeBasenames_RespectsCapacity(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		n := rapid.IntRange(2, 30).Draw(rt, "n")
		capacity := rapid.IntRange(1, 10).Draw(rt, "cap")

		outgoing := make([]string, 0, n-1)
		for i := 1; i < n; i++ {
			outgoing = append(outgoing, "note"+strconv.Itoa(i))
		}

		notes := []vaultgraph.Note{{Basename: "note0", Outgoing: outgoing}}
		for i := 1; i < n; i++ {
			notes = append(notes, vaultgraph.Note{Basename: "note" + strconv.Itoa(i)})
		}

		bridges := graphBridgeBasenames(notes, []string{"note0"}, 2, capacity)
		if got := len(bridges) + 1; got > capacity { // visited = seed + bridges
			rt.Fatalf("visited %d exceeds cap %d", got, capacity)
		}
	})
}

func TestGraphBridgeBasenames_SurfacesUnmatchedBridge(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	// Chain A -> B -> C via body wikilinks (Outgoing). Seed = A; one hop surfaces B.
	notes := []vaultgraph.Note{
		{Basename: "a-wants-cake", Outgoing: []string{"b-cake-needs-sweetness"}},
		{Basename: "b-cake-needs-sweetness", Outgoing: []string{"c-sugar-provides-sweetness"}},
		{Basename: "c-sugar-provides-sweetness"},
	}

	g.Expect(graphBridgeBasenames(notes, []string{"a-wants-cake"}, 1, 300)).
		To(ConsistOf("b-cake-needs-sweetness"))
	g.Expect(graphBridgeBasenames(notes, []string{"a-wants-cake"}, 2, 300)).
		To(ConsistOf("b-cake-needs-sweetness", "c-sugar-provides-sweetness"))
	g.Expect(graphBridgeBasenames(notes, []string{"a-wants-cake"}, 0, 300)).
		To(BeEmpty())
}
