package cli //nolint:testpackage // exercises unexported graphBridgeBasenames

import (
	"strconv"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/vaultgraph"
)

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
