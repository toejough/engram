package cli_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

func TestNextLuhmannID_ContinuationRejectsInvalidTarget(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	_, err := cli.ExportNextLuhmannID(nil, "abc", "continuation")
	g.Expect(err).To(HaveOccurred())
}

func TestNextLuhmannID_ContinuationRequiresTarget(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	_, err := cli.ExportNextLuhmannID(nil, "", "continuation")
	g.Expect(err).To(HaveOccurred())
}

// TestNextLuhmannID_DoubleLetterRollover guards against the regression where
// maxLetterSeg used pure lexical comparison: with children {a..z, aa, ab},
// lexical "z" > "aa" so the allocator returned "z"→"aa" (collision) instead of
// the correct Luhmann order (a..z, aa..az, ba..) which puts "ab" as max.
func TestNextLuhmannID_DoubleLetterRollover(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	existing := buildLetterChildren("1", 'a', 'z')
	existing = append(existing, "1aa", "1ab")

	got, err := cli.ExportNextLuhmannID(existing, "1", "continuation")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).To(Equal("1ac"))
}

func TestNextLuhmannID_FirstChild(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	existing := []string{"1", "2"}
	got, err := cli.ExportNextLuhmannID(existing, "1", "continuation")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).To(Equal("1a"))
}

func TestNextLuhmannID_FirstChildOfDeepGrandchild(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	existing := []string{"1", "1a", "1a1", "1a2"}
	got, err := cli.ExportNextLuhmannID(existing, "1a", "continuation")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).To(Equal("1a3"))
}

func TestNextLuhmannID_FirstGrandchild(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	existing := []string{"1", "1a"}
	got, err := cli.ExportNextLuhmannID(existing, "1a", "continuation")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).To(Equal("1a1"))
}

func TestNextLuhmannID_LetterRollover(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	existing := buildLetterChildren("1", 'a', 'z')
	got, err := cli.ExportNextLuhmannID(existing, "1", "continuation")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).To(Equal("1aa"))
}

func TestNextLuhmannID_NewTopLevel(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	existing := []string{"1", "1a", "2", "2a"}
	got, err := cli.ExportNextLuhmannID(existing, "", "top")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).To(Equal("3"))
}

func TestNextLuhmannID_NewTopLevel_Empty(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	got, err := cli.ExportNextLuhmannID(nil, "", "top")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).To(Equal("1"))
}

func TestNextLuhmannID_NextChild(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	existing := []string{"1", "1a", "1b"}
	got, err := cli.ExportNextLuhmannID(existing, "1", "continuation")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).To(Equal("1c"))
}

func TestNextLuhmannID_NextChildIgnoresUnrelated(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// "2a" is not a child of "1"; "1a3" is a grandchild (depth+2), should be ignored.
	existing := []string{"1", "2", "2a", "1a3"}
	got, err := cli.ExportNextLuhmannID(existing, "1", "continuation")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).To(Equal("1a"))
}

func TestNextLuhmannID_RejectsUnknownRelation(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	_, err := cli.ExportNextLuhmannID(nil, "", "bogus")
	g.Expect(err).To(HaveOccurred())
}

func TestNextLuhmannID_Sibling(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	existing := []string{"1", "1a"}
	got, err := cli.ExportNextLuhmannID(existing, "1a", "sibling")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got).To(Equal("1b"))
}

func TestNextLuhmannID_SiblingOfTopLevel_RejectsTopLevelSibling(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	existing := []string{"1", "2"}
	_, err := cli.ExportNextLuhmannID(existing, "1", "sibling")
	g.Expect(err).To(HaveOccurred())
}

func TestNextLuhmannID_SiblingRejectsInvalidTarget(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	_, err := cli.ExportNextLuhmannID(nil, "abc", "sibling")
	g.Expect(err).To(HaveOccurred())
}

func TestNextLuhmannID_SiblingRequiresTarget(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	_, err := cli.ExportNextLuhmannID(nil, "", "sibling")
	g.Expect(err).To(HaveOccurred())
}

func buildLetterChildren(parent string, from, last rune) []string {
	const extraSlots = 2 // parent + inclusive endpoint

	out := make([]string, 0, int(last-from)+extraSlots)
	out = append(out, parent)

	for r := from; r <= last; r++ {
		out = append(out, parent+string(r))
	}

	return out
}
