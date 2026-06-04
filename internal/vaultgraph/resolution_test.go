package vaultgraph_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/vaultgraph"
)

// G0 invariant: every authored wikilink resolves to a note. UnresolvedTargets
// surfaces the links BuildGraph silently drops — non-empty means G0 is broken.

func TestUnresolvedTargets_FlagsBareIDLinks(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	notes := []vaultgraph.Note{
		// Bare-id link — does NOT match the basename, so it never resolves (the G0 bug).
		{Basename: "105.2026-05-30.foo", Outgoing: []string{"105"}},
		// Full-basename link — resolves.
		{Basename: "1a.2026-05-30.bar", Outgoing: []string{"105.2026-05-30.foo"}},
	}

	got := vaultgraph.UnresolvedTargets(notes)

	g.Expect(got).To(HaveLen(1))
	g.Expect(got[0].Source).To(Equal("105.2026-05-30.foo"))
	g.Expect(got[0].Target).To(Equal("105"))
}

func TestUnresolvedTargets_IgnoresSelfLinksAndResolved(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	notes := []vaultgraph.Note{
		{Basename: "A", Outgoing: []string{"A", "B"}}, // self-link + resolved link
		{Basename: "B"},
	}

	g.Expect(vaultgraph.UnresolvedTargets(notes)).To(BeEmpty())
}
