package luhmann_test

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/luhmann"
)

func TestLess_AntiSymmetricProperty(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		first := genValidID(rt)
		second := genValidID(rt)

		if first == second {
			g.Expect(luhmann.Less(first, second)).To(BeFalse())
			g.Expect(luhmann.Less(second, first)).To(BeFalse())

			return
		}

		ab := luhmann.Less(first, second)
		ba := luhmann.Less(second, first)
		g.Expect(ab).NotTo(Equal(ba))
	})
}

// TestLess_LuhmannLetterOrder pins the contract that Less follows Luhmann
// letter ordering (length-first, then lex), matching nextLetter's z→aa
// rollover convention in internal/cli/luhmann.go.
func TestLess_LuhmannLetterOrder(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(luhmann.Less("1z", "1aa")).To(BeTrue(), "z should sort before aa")
	g.Expect(luhmann.Less("1aa", "1ab")).To(BeTrue(), "aa should sort before ab")
	g.Expect(luhmann.Less("1az", "1ba")).To(BeTrue(), "az should sort before ba")
	g.Expect(luhmann.Less("1zz", "1aaa")).To(BeTrue(), "zz should sort before aaa")
}

// TestLetterLess_LuhmannOrder pins the contract of the shared comparator
// used by Less and (in internal/cli) maxLetterSeg. Length-first, then lex
// within equal length. The cardinal cases match nextLetter's rollover.
func TestLetterLess_LuhmannOrder(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(luhmann.LetterLess("z", "aa")).To(BeTrue())
	g.Expect(luhmann.LetterLess("aa", "ab")).To(BeTrue())
	g.Expect(luhmann.LetterLess("az", "ba")).To(BeTrue())
	g.Expect(luhmann.LetterLess("zz", "aaa")).To(BeTrue())
	g.Expect(luhmann.LetterLess("aa", "aa")).To(BeFalse())
	g.Expect(luhmann.LetterLess("aa", "z")).To(BeFalse())
}

func TestParseID_AlternatingSegments(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	got, err := luhmann.ParseID("1a3b")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).To(Equal([]string{"1", "a", "3", "b"}))
}

func TestParseID_MultiCharSegments(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	got, err := luhmann.ParseID("12ab3")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).To(Equal([]string{"12", "ab", "3"}))
}

func TestParseID_RejectsEmpty(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	_, err := luhmann.ParseID("")
	g.Expect(err).To(MatchError(luhmann.ErrEmpty))
}

func TestParseID_RejectsLeadingLetter(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	_, err := luhmann.ParseID("a1")
	g.Expect(err).To(MatchError(luhmann.ErrLeadingLetter))
}

func TestParseID_TopLevelDigit(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	got, err := luhmann.ParseID("1")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).To(Equal([]string{"1"}))
}

// TestSortIDs_DoubleLetterAfterSingle guards against the regression where
// Less used pure lexical comparison for letter segments. In Luhmann letter
// ordering, "z" < "aa" — single letters come before double letters. Lexical
// comparison reverses this ("aa" < "z"), producing the wrong sort.
func TestSortIDs_DoubleLetterAfterSingle(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	in := []string{"1z", "1aa", "1ab", "1a"}
	luhmann.SortIDs(in)
	g.Expect(in).To(Equal([]string{"1a", "1z", "1aa", "1ab"}))
}

func TestSortIDs_IdempotentProperty(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		const maxLen = 20

		n := rapid.IntRange(0, maxLen).Draw(rt, "n")
		ids := make([]string, n)

		for idx := range ids {
			ids[idx] = genValidID(rt)
		}

		once := append([]string(nil), ids...)
		luhmann.SortIDs(once)

		twice := append([]string(nil), once...)
		luhmann.SortIDs(twice)

		g.Expect(twice).To(Equal(once))
	})
}

func TestSortIDs_NumericNotLexical(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	in := []string{"10", "2", "1"}
	luhmann.SortIDs(in)
	g.Expect(in).To(Equal([]string{"1", "2", "10"}))
}

func TestSortIDs_TreeOrder(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	in := []string{"2", "1b", "1a1", "1", "1a", "10", "1a2"}
	luhmann.SortIDs(in)
	g.Expect(in).To(Equal([]string{"1", "1a", "1a1", "1a2", "1b", "2", "10"}))
}

// genValidID generates a Luhmann ID string that ParseID will accept:
// starts with a digit segment, then alternates letter/digit segments.
func genValidID(rt *rapid.T) string {
	const maxDepth = 6

	depth := rapid.IntRange(1, maxDepth).Draw(rt, "depth")

	var builder strings.Builder

	for level := range depth {
		if level%2 == 0 {
			builder.WriteString(rapid.StringMatching(`[1-9][0-9]{0,2}`).Draw(rt, "digit"))
		} else {
			builder.WriteString(rapid.StringMatching(`[a-z]{1,3}`).Draw(rt, "letter"))
		}
	}

	return builder.String()
}
