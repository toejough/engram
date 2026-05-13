package vaultgraph_test

import (
	"sort"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/vaultgraph"
)

func TestComponents_AddingEdgeCanOnlyMergeProperty(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		gExpect := NewWithT(rt)

		notes := genNotesProperty(rt)
		before := len(vaultgraph.Components(vaultgraph.BuildGraph(notes)))

		if len(notes) < 2 {
			return
		}

		// Add one extra edge from notes[0] to notes[1] (might be redundant).
		notes[0].Outgoing = append(notes[0].Outgoing, notes[1].Basename)
		after := len(vaultgraph.Components(vaultgraph.BuildGraph(notes)))

		gExpect.Expect(after).To(BeNumerically("<=", before))
	})
}

func TestComponents_AllIsolated(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	notes := []vaultgraph.Note{{Basename: "A"}, {Basename: "B"}, {Basename: "C"}}
	got := sortedComponents(vaultgraph.Components(vaultgraph.BuildGraph(notes)))

	g.Expect(got).To(Equal([][]string{{"A"}, {"B"}, {"C"}}))
}

func TestComponents_DirectionAgnostic(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// A → B, C → B. Even though A and C never link to each other directly,
	// they're in the same undirected component via B.
	notes := []vaultgraph.Note{
		{Basename: "A", Outgoing: []string{"B"}},
		{Basename: "B"},
		{Basename: "C", Outgoing: []string{"B"}},
	}
	got := sortedComponents(vaultgraph.Components(vaultgraph.BuildGraph(notes)))

	g.Expect(got).To(Equal([][]string{{"A", "B", "C"}}))
}

func TestComponents_EmptyGraph(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(vaultgraph.Components(vaultgraph.BuildGraph(nil))).To(BeEmpty())
}

func TestComponents_PartitionCoversAllNodesProperty(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		gExpect := NewWithT(rt)

		notes := genNotesProperty(rt)
		graph := vaultgraph.BuildGraph(notes)
		comps := vaultgraph.Components(graph)

		seen := make(map[string]int)

		for _, members := range comps {
			for _, name := range members {
				seen[name]++
			}
		}

		// Each node appears in exactly one component.
		for name := range graph.Notes {
			gExpect.Expect(seen[name]).
				To(Equal(1), "node %s appears in %d components", name, seen[name])
		}

		gExpect.Expect(seen).To(HaveLen(len(graph.Notes)))
	})
}

func TestComponents_TwoNodesOneEdge(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	notes := []vaultgraph.Note{
		{Basename: "A", Outgoing: []string{"B"}},
		{Basename: "B"},
	}
	got := sortedComponents(vaultgraph.Components(vaultgraph.BuildGraph(notes)))

	g.Expect(got).To(Equal([][]string{{"A", "B"}}))
}

func TestComponents_TwoSeparateComponents(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	notes := []vaultgraph.Note{
		{Basename: "A", Outgoing: []string{"B"}},
		{Basename: "B"},
		{Basename: "C", Outgoing: []string{"D"}},
		{Basename: "D"},
	}
	got := sortedComponents(vaultgraph.Components(vaultgraph.BuildGraph(notes)))

	g.Expect(got).To(Equal([][]string{{"A", "B"}, {"C", "D"}}))
}

func sortedComponents(comps [][]string) [][]string {
	out := make([][]string, len(comps))
	for idx, members := range comps {
		copied := append([]string(nil), members...)
		sort.Strings(copied)
		out[idx] = copied
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i][0] < out[j][0]
	})

	return out
}
