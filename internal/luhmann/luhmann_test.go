package luhmann_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/luhmann"
)

// TestFromBasename_ExtractsLeadingID covers the happy path: a basename of
// the form `<luhmann-id>.<date>.<slug>` yields the leading ID.
func TestFromBasename_ExtractsLeadingID(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	cases := map[string]string{
		"9o1.2026-05-10.slug":           "9o1",
		"14.2026-05-10.drop-by-example": "14",
		"1a3b.2026-05-09.foo":           "1a3b",
		"5.2026-05-09.rationalization":  "5",
	}

	for input, want := range cases {
		got, ok := luhmann.FromBasename(input)
		g.Expect(ok).To(BeTrue(), "input: %s", input)
		g.Expect(got).To(Equal(want), "input: %s", input)
	}
}

// TestFromBasename_RejectsLeadingDot guards against accepting hidden files.
func TestFromBasename_RejectsLeadingDot(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	_, ok := luhmann.FromBasename(".2026-05-10.slug")
	g.Expect(ok).To(BeFalse())
}

// TestFromBasename_RejectsNoDot guards against the no-separator case.
func TestFromBasename_RejectsNoDot(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	_, ok := luhmann.FromBasename("9o1nodate")
	g.Expect(ok).To(BeFalse())
}

// TestFromBasename_RejectsNoLeadingDigit guards against the no-digit-prefix case.
func TestFromBasename_RejectsNoLeadingDigit(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	_, ok := luhmann.FromBasename("notes-only.2026-05-10.something")
	g.Expect(ok).To(BeFalse())
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
