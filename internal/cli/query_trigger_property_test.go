package cli_test

import (
	"regexp"
	"slices"
	"strings"
	"testing"
	"unicode"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
)

// TestMatchTriggers_HitsIffNormalizedWholeWord: a note is a hit iff some
// trigger, lowercased with whitespace runs collapsed, occurs in the text
// normalized the same way with a non-letter/digit (or text edge) on each side
// where the trigger's own edge rune is a letter or digit. The oracle is a
// regexp, independent of the implementation's rune scan.
func TestMatchTriggers_HitsIffNormalizedWholeWord(t *testing.T) {
	t.Parallel()

	const alphabet = `[A-Za-z0-9é_/.-]`

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		words := rapid.SliceOfN(rapid.StringMatching(alphabet+`{1,6}`), 0, 8).Draw(rt, "words")
		text := strings.Join(words, rapid.SampledFrom([]string{" ", "  ", "\t", "\n", "-", ""}).Draw(rt, "sep"))

		index := map[string][]string{}

		for _, name := range []string{"a", "b", "c"} {
			index[name] = rapid.SliceOfN(
				rapid.StringMatching(alphabet+`{3,6}( `+alphabet+`{3,6})?`), 0, 3,
			).Draw(rt, name)
		}

		norm := strings.ToLower(strings.Join(strings.Fields(text), " "))

		want := make([]string, 0, len(index))

		for name, triggers := range index {
			for _, trigger := range triggers {
				if wholeWordOracle(norm, strings.ToLower(strings.Join(strings.Fields(trigger), " "))) {
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

// wholeWordOracle reports whether needle occurs in norm bounded, on each side
// whose needle edge rune is a letter or digit, by a non-letter/digit or the
// text edge. Built as a regexp so it shares no logic with the matcher.
func wholeWordOracle(norm, needle string) bool {
	const (
		before = `(^|[^\p{L}\p{N}])`
		after  = `([^\p{L}\p{N}]|$)`
	)

	runes := []rune(needle)
	pattern := ""

	if unicode.IsLetter(runes[0]) || unicode.IsDigit(runes[0]) {
		pattern += before
	}

	pattern += regexp.QuoteMeta(needle)

	if last := runes[len(runes)-1]; unicode.IsLetter(last) || unicode.IsDigit(last) {
		pattern += after
	}

	return regexp.MustCompile(pattern).MatchString(norm)
}
