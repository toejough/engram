package cli_test

import (
	"slices"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
)

// TestMatchTriggers_HitsIffNormalizedSubstring: a note is a hit iff some
// trigger, lowercased with whitespace runs collapsed, is a substring of the
// text normalized the same way.
func TestMatchTriggers_HitsIffNormalizedSubstring(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		words := rapid.SliceOfN(rapid.StringMatching(`[A-Za-z/-]{1,6}`), 0, 8).Draw(rt, "words")
		text := strings.Join(words, rapid.SampledFrom([]string{" ", "  ", "\t", "\n"}).Draw(rt, "sep"))

		index := map[string][]string{}

		for _, name := range []string{"a", "b", "c"} {
			index[name] = rapid.SliceOfN(rapid.StringMatching(`[A-Za-z/-]{3,6}( [A-Za-z/-]{3,6})?`), 0, 3).Draw(rt, name)
		}

		norm := strings.ToLower(strings.Join(strings.Fields(text), " "))

		want := make([]string, 0, len(index))

		for name, triggers := range index {
			for _, trigger := range triggers {
				if strings.Contains(norm, strings.ToLower(strings.Join(strings.Fields(trigger), " "))) {
					want = append(want, name)

					break
				}
			}
		}

		slices.Sort(want)

		got := cli.ExportMatchTriggers(text, index)
		g.Expect(got).To(Equal(orEmpty(want)))
	})
}

// TestResolvedItemLess_TriggerAlwaysFirst: for arbitrary provenance sets and
// scores, an item with "trigger" provenance precedes every item without it.
func TestResolvedItemLess_TriggerAlwaysFirst(t *testing.T) {
	t.Parallel()

	roles := []string{"direct", "cluster_rep", "explore", "recent", "ride_along"}

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		other := rapid.SliceOfNDistinct(rapid.SampledFrom(roles), 1, len(roles), rapid.ID[string]).Draw(rt, "other")
		extra := rapid.SliceOfNDistinct(rapid.SampledFrom(roles), 0, len(roles), rapid.ID[string]).Draw(rt, "extra")
		trig := append([]string{"trigger"}, extra...)
		scoreT := rapid.Float32Range(0, 1).Draw(rt, "scoreT")
		scoreO := rapid.Float32Range(0, 1).Draw(rt, "scoreO")

		g.Expect(cli.ExportResolvedItemLessByProvenance(trig, scoreT, other, scoreO)).To(BeTrue())
		g.Expect(cli.ExportResolvedItemLessByProvenance(other, scoreO, trig, scoreT)).To(BeFalse())
	})
}

func orEmpty(in []string) []string {
	if in == nil {
		return []string{}
	}

	return in
}
