package vaultgraph_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/vaultgraph"
)

func TestFollow_DedupesAcrossOutgoingAndBacklinks(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// b links to a AND a links to b — only one b should appear.
	notes := []vaultgraph.Note{
		{Basename: "a", Outgoing: []string{"b"}},
		{Basename: "b", Outgoing: []string{"a"}},
	}
	graph := vaultgraph.BuildGraph(notes)

	got := vaultgraph.Follow(graph, []string{"a"}, nil)

	g.Expect(got).To(Equal([]string{"b"}))
}

func TestFollow_OutputIsDeterministicallySorted(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	notes := []vaultgraph.Note{
		{Basename: "a", Outgoing: []string{"z", "m", "b"}},
		{Basename: "b"},
		{Basename: "m"},
		{Basename: "z"},
	}
	graph := vaultgraph.BuildGraph(notes)

	got := vaultgraph.Follow(graph, []string{"a"}, nil)

	g.Expect(got).To(Equal([]string{"b", "m", "z"}))
}

func TestFollow_ReturnsBacklinks(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	notes := []vaultgraph.Note{
		{Basename: "a"},
		{Basename: "b", Outgoing: []string{"a"}},
		{Basename: "c", Outgoing: []string{"a"}},
	}
	graph := vaultgraph.BuildGraph(notes)

	got := vaultgraph.Follow(graph, []string{"a"}, nil)

	g.Expect(got).To(ConsistOf("b", "c"))
}

func TestFollow_ReturnsOutgoingLinks(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	notes := []vaultgraph.Note{
		{Basename: "a", Outgoing: []string{"b", "c"}},
		{Basename: "b"},
		{Basename: "c"},
	}
	graph := vaultgraph.BuildGraph(notes)

	got := vaultgraph.Follow(graph, []string{"a"}, nil)

	g.Expect(got).To(ConsistOf("b", "c"))
}

func TestFollow_SubtractsAlreadyRead(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	notes := []vaultgraph.Note{
		{Basename: "a", Outgoing: []string{"b", "c"}},
		{Basename: "b"},
		{Basename: "c"},
	}
	graph := vaultgraph.BuildGraph(notes)

	got := vaultgraph.Follow(graph, []string{"a"}, []string{"b"})

	g.Expect(got).To(Equal([]string{"c"}))
}

func TestFollow_SubtractsFollowSetItself(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Following 'a' should never re-emit 'a' even if it appears in a self-link
	// or a partner's outgoing list.
	notes := []vaultgraph.Note{
		{Basename: "a", Outgoing: []string{"b"}},
		{Basename: "b", Outgoing: []string{"a"}},
	}
	graph := vaultgraph.BuildGraph(notes)

	got := vaultgraph.Follow(graph, []string{"a", "b"}, nil)

	g.Expect(got).To(BeEmpty())
}
