package vaultgraph_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/vaultgraph"
)

// Test792b13c3a10_ParseWikilinks_DisplayPartStaysInTarget verifies 792b13c3a10: a `|display` part is kept as
// part of the listed target.
func Test792b13c3a10_ParseWikilinks_DisplayPartStaysInTarget(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	got := vaultgraph.ParseWikilinks([]byte("see [[4.gamma|the gamma note]]"))

	g.Expect(got).To(Equal([]string{"4.gamma|the gamma note"}))
}

// Test792b13c5a2_Graph_UnknownBasenameInDegreeIsZero verifies 792b13c5a2: the in-degree of a basename that is
// not a node is 0, including a basename a note links to but no scanned note has (the broken link is dropped).
func Test792b13c5a2_Graph_UnknownBasenameInDegreeIsZero(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	graph := vaultgraph.BuildGraph([]vaultgraph.Note{
		{Basename: "A", Outgoing: []string{"B", "ghost"}},
		{Basename: "B"},
	})

	g.Expect(graph.InDegree("B")).To(Equal(1))
	g.Expect(graph.InDegree("ghost")).To(BeZero())
	g.Expect(graph.InDegree("never-linked")).To(BeZero())
}
