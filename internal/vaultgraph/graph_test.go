package vaultgraph_test

import (
	"sort"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/vaultgraph"
)

func TestBuildGraph_DropsBrokenLinks(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	notes := []vaultgraph.Note{
		{Basename: "A", Outgoing: []string{"B", "MISSING"}},
		{Basename: "B"},
	}

	got := vaultgraph.BuildGraph(notes)
	g.Expect(got.Outgoing["A"]).To(HaveKey("B"))
	g.Expect(got.Outgoing["A"]).NotTo(HaveKey("MISSING"))
	g.Expect(got.InDegree("MISSING")).To(BeZero())
}

func TestBuildGraph_DropsSelfLinks(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	notes := []vaultgraph.Note{
		{Basename: "A", Outgoing: []string{"A", "B"}},
		{Basename: "B"},
	}

	got := vaultgraph.BuildGraph(notes)
	g.Expect(got.Outgoing["A"]).NotTo(HaveKey("A"))
	g.Expect(got.Outgoing["A"]).To(HaveKey("B"))
}

func TestBuildGraph_EdgeCountBoundsProperty(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		gExpect := NewWithT(rt)

		notes := genNotesProperty(rt)
		got := vaultgraph.BuildGraph(notes)

		// Σ outgoing edge counts ≤ Σ raw outgoing lengths (post-filter ≤ pre-filter).
		rawTotal := 0
		for _, note := range notes {
			rawTotal += len(note.Outgoing)
		}

		gotTotal := 0
		for _, dsts := range got.Outgoing {
			gotTotal += len(dsts)
		}

		gExpect.Expect(gotTotal).To(BeNumerically("<=", rawTotal))

		// Σ in-degree = Σ out-degree (every edge counted both ways).
		inSum := 0
		for name := range got.Notes {
			inSum += got.InDegree(name)
		}

		gExpect.Expect(inSum).To(Equal(gotTotal))
	})
}

func TestBuildGraph_EmptyInput(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	got := vaultgraph.BuildGraph(nil)
	g.Expect(got.Notes).To(BeEmpty())
	g.Expect(got.Outgoing).To(BeEmpty())
	g.Expect(got.Incoming).To(BeEmpty())
}

func TestBuildGraph_InDegree(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	notes := []vaultgraph.Note{
		{Basename: "A", Outgoing: []string{"C"}},
		{Basename: "B", Outgoing: []string{"C"}},
		{Basename: "C", Outgoing: []string{"A"}},
		{Basename: "D"},
	}

	got := vaultgraph.BuildGraph(notes)
	g.Expect(got.InDegree("C")).To(Equal(2))
	g.Expect(got.InDegree("A")).To(Equal(1))
	g.Expect(got.InDegree("B")).To(Equal(0))
	g.Expect(got.InDegree("D")).To(Equal(0))
}

func TestBuildGraph_NoSelfLinksProperty(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		gExpect := NewWithT(rt)

		notes := genNotesProperty(rt)
		got := vaultgraph.BuildGraph(notes)

		for src, dsts := range got.Outgoing {
			_, selfLoop := dsts[src]
			gExpect.Expect(selfLoop).To(BeFalse())
		}
	})
}

func TestBuildGraph_UndirectedNeighbors(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	notes := []vaultgraph.Note{
		{Basename: "A", Outgoing: []string{"B"}},
		{Basename: "B", Outgoing: []string{"C"}},
		{Basename: "C", Outgoing: []string{"A"}},
	}

	got := vaultgraph.BuildGraph(notes)

	neighbors := got.UndirectedNeighbors("B")
	sort.Strings(neighbors)
	g.Expect(neighbors).To(Equal([]string{"A", "C"}))
}

// genNotesProperty generates a random set of notes whose outgoing links only
// reference basenames in the set. The generated graph is therefore broken-link-free.
func genNotesProperty(rt *rapid.T) []vaultgraph.Note {
	const maxNotes = 8

	n := rapid.IntRange(0, maxNotes).Draw(rt, "n")

	names := make([]string, n)
	for idx := range names {
		names[idx] = rapid.StringMatching(`[A-Z][0-9]`).Draw(rt, "name")
	}

	notes := make([]vaultgraph.Note, 0, len(names))

	for _, name := range names {
		linkCount := rapid.IntRange(0, len(names)).Draw(rt, "links")

		outgoing := make([]string, 0, linkCount)

		for range linkCount {
			if len(names) == 0 {
				break
			}

			outgoing = append(outgoing, rapid.SampledFrom(names).Draw(rt, "target"))
		}

		notes = append(notes, vaultgraph.Note{Basename: name, Outgoing: outgoing})
	}

	return notes
}
