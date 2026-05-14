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

func TestSelectStartingPoints_MOCWinsOverHighInDegreeNonMOC(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// B and C link to A (non-MOC), giving A in-degree 2. But MOC M is in the component too.
	notes := []vaultgraph.Note{
		{Basename: "M", LuhmannID: "1", IsMOC: true, Outgoing: []string{"A"}},
		{Basename: "A", LuhmannID: "2", Outgoing: []string{"M"}},
		{Basename: "B", LuhmannID: "3", Outgoing: []string{"A"}},
		{Basename: "C", LuhmannID: "4", Outgoing: []string{"A"}},
	}
	graph := vaultgraph.BuildGraph(notes)
	got := vaultgraph.SelectStartingPoints([]string{"M", "A", "B", "C"}, graph)
	g.Expect(got).To(Equal([]string{"M"}))
}

func TestSelectStartingPoints_MultipleMOCsSortedAlphabetically(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	notes := []vaultgraph.Note{
		{Basename: "12.zk-craft", LuhmannID: "12", IsMOC: true, Outgoing: []string{"7.zk-mem"}},
		{Basename: "7.zk-mem", LuhmannID: "7", IsMOC: true},
	}
	graph := vaultgraph.BuildGraph(notes)

	got := vaultgraph.SelectStartingPoints([]string{"12.zk-craft", "7.zk-mem"}, graph)
	g.Expect(got).To(Equal([]string{"12.zk-craft", "7.zk-mem"}))
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

func TestSelectStartingPoints_SingleMOC(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	notes := []vaultgraph.Note{
		{Basename: "7.2026-05-09.zk", LuhmannID: "7", IsMOC: true, Outgoing: []string{"4.x.y"}},
		{Basename: "4.x.y", LuhmannID: "4"},
	}
	graph := vaultgraph.BuildGraph(notes)

	got := vaultgraph.SelectStartingPoints([]string{"7.2026-05-09.zk", "4.x.y"}, graph)
	g.Expect(got).To(Equal([]string{"7.2026-05-09.zk"}))
}

func TestSelectStartingPoints_SingletonIsolatedNode(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	notes := []vaultgraph.Note{{Basename: "lonely", LuhmannID: "1"}}
	graph := vaultgraph.BuildGraph(notes)
	got := vaultgraph.SelectStartingPoints([]string{"lonely"}, graph)
	g.Expect(got).To(Equal([]string{"lonely"}))
}
