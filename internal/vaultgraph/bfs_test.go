package vaultgraph_test

import (
	"sort"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/vaultgraph"
)

func TestBFSWithCap_CapStopsExpansion(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Build a chain of 50 notes; cap at 10.
	const chainLen = 50

	notes := make([]vaultgraph.Note, chainLen)
	for i := range chainLen - 1 {
		notes[i] = vaultgraph.Note{
			Basename: nodeName(i),
			Outgoing: []string{nodeName(i + 1)},
		}
	}

	notes[chainLen-1] = vaultgraph.Note{Basename: nodeName(chainLen - 1)}

	graph := vaultgraph.BuildGraph(notes)

	const maxDepth = 100

	const capacity = 10

	result := vaultgraph.BFSWithCap(graph, []string{nodeName(0)}, maxDepth, capacity)
	g.Expect(len(result.Visited)).To(BeNumerically("<=", capacity))
	g.Expect(result.Capped).To(BeTrue())
}

// TestBFSWithCap_DuplicateSeedsDeduped exercises the "seed already visited"
// branch of admitSeeds.
func TestBFSWithCap_DuplicateSeedsDeduped(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	notes := []vaultgraph.Note{{Basename: "A"}, {Basename: "B"}}
	graph := vaultgraph.BuildGraph(notes)

	result := vaultgraph.BFSWithCap(graph, []string{"A", "A", "B"}, 3, 100)
	g.Expect(result.Visited).To(HaveLen(2))
}

func TestBFSWithCap_EmptyStartsReturnsEmpty(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	notes := []vaultgraph.Note{{Basename: "A"}}
	graph := vaultgraph.BuildGraph(notes)

	result := vaultgraph.BFSWithCap(graph, nil, 3, 100)
	g.Expect(result.Visited).To(BeEmpty())
	g.Expect(result.HopsReached).To(BeZero())
}

// TestBFSWithCap_SeedAdmissionHitsCap exercises the cap-during-seeding branch.
func TestBFSWithCap_SeedAdmissionHitsCap(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	notes := []vaultgraph.Note{
		{Basename: "A"}, {Basename: "B"}, {Basename: "C"},
	}
	graph := vaultgraph.BuildGraph(notes)

	// Capacity of 2 with 3 seeds: third must trigger cap.
	result := vaultgraph.BFSWithCap(graph, []string{"A", "B", "C"}, 3, 2)
	g.Expect(result.Visited).To(HaveLen(2))
	g.Expect(result.Capped).To(BeTrue())
}

func TestBFSWithCap_StartIncludesSeeds(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Seeds always count as depth 0 in the visited set.
	notes := []vaultgraph.Note{
		{Basename: "A"},
		{Basename: "B"},
	}
	graph := vaultgraph.BuildGraph(notes)

	result := vaultgraph.BFSWithCap(graph, []string{"A"}, 3, 100)
	g.Expect(result.Visited).To(HaveKey("A"))
	g.Expect(result.Visited).NotTo(HaveKey("B"))
}

func TestBFSWithCap_TraversesBackwardEdges(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// A → B (only directed forward). Starting at B should still reach A
	// because expansion is undirected.
	notes := []vaultgraph.Note{
		{Basename: "A", Outgoing: []string{"B"}},
		{Basename: "B"},
	}
	graph := vaultgraph.BuildGraph(notes)

	const maxDepth = 3

	const capacity = 100

	result := vaultgraph.BFSWithCap(graph, []string{"B"}, maxDepth, capacity)
	g.Expect(result.Visited).To(HaveKey("A"))
}

func TestBFSWithCap_TraversesUndirectedNeighborsAtDepth(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// A → B → C → D (chain). Start at A, depth 2 hops → reaches A, B, C; not D.
	notes := []vaultgraph.Note{
		{Basename: "A", Outgoing: []string{"B"}},
		{Basename: "B", Outgoing: []string{"C"}},
		{Basename: "C", Outgoing: []string{"D"}},
		{Basename: "D"},
	}
	graph := vaultgraph.BuildGraph(notes)

	const maxDepth = 2

	const capacity = 100

	result := vaultgraph.BFSWithCap(graph, []string{"A"}, maxDepth, capacity)

	got := make([]string, 0, len(result.Visited))
	for name := range result.Visited {
		got = append(got, name)
	}

	sort.Strings(got)
	g.Expect(got).To(Equal([]string{"A", "B", "C"}))
	g.Expect(result.Capped).To(BeFalse())
	g.Expect(result.HopsReached).To(Equal(2))
}

// TestBFSWithCap_UnknownSeedsSkipped exercises the "seed not in graph"
// branch of admitSeeds.
func TestBFSWithCap_UnknownSeedsSkipped(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	notes := []vaultgraph.Note{{Basename: "A"}}
	graph := vaultgraph.BuildGraph(notes)

	result := vaultgraph.BFSWithCap(graph, []string{"missing-1", "A", "missing-2"}, 3, 100)
	g.Expect(result.Visited).To(HaveLen(1))
	g.Expect(result.Visited).To(HaveKey("A"))
}

func TestBFSWithCap_VisitedSetHandlesCycles(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// A → B → A cycle.
	notes := []vaultgraph.Note{
		{Basename: "A", Outgoing: []string{"B"}},
		{Basename: "B", Outgoing: []string{"A"}},
	}
	graph := vaultgraph.BuildGraph(notes)

	const maxDepth = 100

	const capacity = 100

	result := vaultgraph.BFSWithCap(graph, []string{"A"}, maxDepth, capacity)
	g.Expect(result.Visited).To(HaveLen(2))
	g.Expect(result.Capped).To(BeFalse())
}

func nodeName(idx int) string {
	const offsetA = 'A'

	const wrap = 26

	first := byte(offsetA + (idx/wrap)%wrap)
	second := byte(offsetA + idx%wrap)

	return string([]byte{first, second})
}
