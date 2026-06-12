package vaultgraph_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/vaultgraph"
)

func TestSelectStartingPoints_EmptyComponent(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	got := vaultgraph.SelectStartingPoints(nil, vaultgraph.Graph{})
	g.Expect(got).To(BeEmpty())
}

func TestSelectStartingPoints_NoMOC_ClearInDegreeWinner(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	notes := []vaultgraph.Note{
		{Basename: "A", LuhmannID: "1", Outgoing: []string{"C"}},
		{Basename: "B", LuhmannID: "2", Outgoing: []string{"C"}},
		{Basename: "C", LuhmannID: "3"},
	}
	graph := vaultgraph.BuildGraph(notes)
	got := vaultgraph.SelectStartingPoints([]string{"A", "B", "C"}, graph)
	g.Expect(got).To(Equal([]string{"C"}))
}

func TestSelectStartingPoints_NoMOC_IDBeatsIDLessOnTie(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	notes := []vaultgraph.Note{
		{Basename: "fleeting", LuhmannID: ""},
		{Basename: "9a.x.slug", LuhmannID: "9a"},
		{Basename: "linker", LuhmannID: "1", Outgoing: []string{"fleeting", "9a.x.slug"}},
	}
	graph := vaultgraph.BuildGraph(notes)
	got := vaultgraph.SelectStartingPoints([]string{"fleeting", "9a.x.slug", "linker"}, graph)
	g.Expect(got).To(Equal([]string{"9a.x.slug"}))
}

func TestSelectStartingPoints_NoMOC_LuhmannTieBreak(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// X and Y both have in-degree 1; tie-break by Luhmann ("1" < "2").
	notes := []vaultgraph.Note{
		{Basename: "X", LuhmannID: "1", Outgoing: []string{}},
		{Basename: "Y", LuhmannID: "2", Outgoing: []string{}},
		{Basename: "Z", LuhmannID: "3", Outgoing: []string{"X", "Y"}},
	}
	graph := vaultgraph.BuildGraph(notes)
	got := vaultgraph.SelectStartingPoints([]string{"X", "Y", "Z"}, graph)
	g.Expect(got).To(Equal([]string{"X"}))
}

func TestSelectStartingPoints_NoMOC_UnbreakableTieAllIDLess(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Two fleetings without Luhmann IDs, same in-degree 1 each. Tie cannot be broken; both win.
	notes := []vaultgraph.Note{
		{Basename: "fleeting-a", LuhmannID: "", Outgoing: []string{}},
		{Basename: "fleeting-b", LuhmannID: "", Outgoing: []string{}},
		{Basename: "linker", LuhmannID: "1", Outgoing: []string{"fleeting-a", "fleeting-b"}},
	}
	graph := vaultgraph.BuildGraph(notes)
	got := vaultgraph.SelectStartingPoints([]string{"fleeting-a", "fleeting-b", "linker"}, graph)
	g.Expect(got).To(Equal([]string{"fleeting-a", "fleeting-b"}))
}

func TestSelectStartingPoints_SingletonIsolatedNode(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	notes := []vaultgraph.Note{{Basename: "lonely", LuhmannID: "1"}}
	graph := vaultgraph.BuildGraph(notes)
	got := vaultgraph.SelectStartingPoints([]string{"lonely"}, graph)
	g.Expect(got).To(Equal([]string{"lonely"}))
}
