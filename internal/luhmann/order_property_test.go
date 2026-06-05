package luhmann_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/luhmann"
)

// TestLess_StrictTotalOrderProperty asserts that Less is a strict total order
// over a generated set of valid Luhmann IDs: total + antisymmetric (per pair,
// exactly one of Less(a,b)/Less(b,a) holds unless a==b, in which case neither
// does) and transitive (Less(a,b) && Less(b,c) implies Less(a,c)). Triples are
// drawn from the generated set so the transitivity antecedent fires non-vacuously.
func TestLess_StrictTotalOrderProperty(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		const maxLen = 8

		n := rapid.IntRange(1, maxLen).Draw(rt, "n")
		ids := make([]string, n)

		for idx := range ids {
			ids[idx] = genValidID(rt)
		}

		expectTotalAndAntisymmetric(g, ids)
		expectTransitive(g, ids)
	})
}

// TestSortIDs_SortedIsNoOpProperty asserts that sorting an already-sorted slice
// of valid Luhmann IDs leaves it unchanged: SortIDs is idempotent (and thus
// stable on its own output). This complements the antisymmetry/transitivity
// property by pinning the comparator-to-sort contract.
func TestSortIDs_SortedIsNoOpProperty(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		const maxLen = 20

		n := rapid.IntRange(0, maxLen).Draw(rt, "n")
		ids := make([]string, n)

		for idx := range ids {
			ids[idx] = genValidID(rt)
		}

		luhmann.SortIDs(ids)

		// Snapshot the already-sorted slice, then sort the snapshot again. Both
		// once and twice are built the same way so the comparison is unaffected
		// by nil-vs-empty-slice distinctions when n == 0.
		once := append([]string(nil), ids...)
		twice := append([]string(nil), once...)
		luhmann.SortIDs(twice)

		g.Expect(twice).To(Equal(once))
	})
}

// expectTotalAndAntisymmetric checks, over every ordered pair drawn from ids,
// that equal IDs are never Less in either direction, and distinct IDs are Less
// in exactly one direction. The exactly-one check folds together antisymmetry
// (not both) and connexity (at least one) — the "total" in strict total order.
func expectTotalAndAntisymmetric(g *WithT, ids []string) {
	for i := range ids {
		for j := range ids {
			forward := luhmann.Less(ids[i], ids[j])
			backward := luhmann.Less(ids[j], ids[i])

			if ids[i] == ids[j] {
				g.Expect(forward).To(BeFalse(), "Less(%q, %q)", ids[i], ids[j])
				g.Expect(backward).To(BeFalse(), "Less(%q, %q)", ids[j], ids[i])

				continue
			}

			g.Expect(forward).
				ToNot(Equal(backward), "exactly one of Less must hold for distinct %q and %q", ids[i], ids[j])
		}
	}
}

// expectTransitive checks, over every triple drawn from ids, that
// Less(a,b) && Less(b,c) implies Less(a,c).
func expectTransitive(g *WithT, ids []string) {
	for i := range ids {
		for j := range ids {
			for k := range ids {
				if luhmann.Less(ids[i], ids[j]) && luhmann.Less(ids[j], ids[k]) {
					g.Expect(luhmann.Less(ids[i], ids[k])).
						To(BeTrue(), "transitivity for %q < %q < %q", ids[i], ids[j], ids[k])
				}
			}
		}
	}
}
