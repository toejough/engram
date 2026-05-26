package vaultgraph_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/vaultgraph"
)

func TestInDegreeIn_RestrictsToSubset(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// A → C, B → C, X → C. Subset = {A, B, C} — D's wikilink to C
	// counts in the global graph but not in the subset.
	notes := []vaultgraph.Note{
		{Basename: "A", Outgoing: []string{"C"}},
		{Basename: "B", Outgoing: []string{"C"}},
		{Basename: "C"},
		{Basename: "X", Outgoing: []string{"C"}},
	}
	graph := vaultgraph.BuildGraph(notes)

	subset := map[string]struct{}{
		"A": {},
		"B": {},
		"C": {},
	}
	// Global in-degree of C is 3; subset in-degree is 2.
	g.Expect(graph.InDegree("C")).To(Equal(3))
	g.Expect(graph.InDegreeIn("C", subset)).To(Equal(2))
}

func TestInDegreeIn_UnknownNodeReturnsZero(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	notes := []vaultgraph.Note{{Basename: "A"}}
	graph := vaultgraph.BuildGraph(notes)

	subset := map[string]struct{}{"A": {}}

	g.Expect(graph.InDegreeIn("missing", subset)).To(BeZero())
}
