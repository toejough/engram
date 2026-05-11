package vaultgraph_test

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"engram/internal/vaultgraph"
)

func TestLuhmannFromBasename_ExtractsLeadingID(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	cases := map[string]string{
		"9o1.2026-05-10.slug":           "9o1",
		"14.2026-05-10.drop-by-example": "14",
		"1a3b.2026-05-09.foo":           "1a3b",
		"5.2026-05-09.rationalization":  "5",
	}

	for input, want := range cases {
		got, ok := vaultgraph.LuhmannFromBasename(input)
		g.Expect(ok).To(BeTrue(), "input: %s", input)
		g.Expect(got).To(Equal(want), "input: %s", input)
	}
}

func TestLuhmannFromBasename_RejectsLeadingDot(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	_, ok := vaultgraph.LuhmannFromBasename(".2026-05-10.slug")
	g.Expect(ok).To(BeFalse())
}

func TestLuhmannFromBasename_RejectsNoDot(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	_, ok := vaultgraph.LuhmannFromBasename("9o1nodate")
	g.Expect(ok).To(BeFalse())
}

func TestLuhmannFromBasename_RejectsNoLeadingDigit(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	_, ok := vaultgraph.LuhmannFromBasename("notes-only.2026-05-10.something")
	g.Expect(ok).To(BeFalse())
}

func TestParseBasename_RejectsNonMd(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	_, ok := vaultgraph.ParseBasename("README")
	g.Expect(ok).To(BeFalse())

	_, ok = vaultgraph.ParseBasename("notes.txt")
	g.Expect(ok).To(BeFalse())
}

func TestParseBasename_StripsMdExt(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	got, ok := vaultgraph.ParseBasename("9o1.2026-05-10.slug.md")
	g.Expect(ok).To(BeTrue())
	g.Expect(got).To(Equal("9o1.2026-05-10.slug"))
}

func TestParseWikilinks_AcrossLines(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	body := []byte("first [[X]]\nsecond [[Y]]\nthird [[X]]")

	g.Expect(vaultgraph.ParseWikilinks(body)).To(Equal([]string{"X", "Y"}))
}

func TestParseWikilinks_DedupedSubsetProperty(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		const maxLinks = 10

		// Build a body of N wikilinks (possibly duplicated) plus arbitrary surrounding prose.
		n := rapid.IntRange(0, maxLinks).Draw(rt, "n")

		var builder strings.Builder

		all := make([]string, 0, n)

		for range n {
			target := rapid.StringMatching(`[a-zA-Z0-9.-]+`).Draw(rt, "target")
			filler := rapid.StringMatching(`[ a-z]{0,5}`).Draw(rt, "filler")

			builder.WriteString(filler)
			builder.WriteString("[[")
			builder.WriteString(target)
			builder.WriteString("]] ")

			all = append(all, target)
		}

		got := vaultgraph.ParseWikilinks([]byte(builder.String()))

		// Result is deduped.
		seen := make(map[string]struct{}, len(got))

		for _, target := range got {
			_, dup := seen[target]
			g.Expect(dup).To(BeFalse())

			seen[target] = struct{}{}
		}

		// Every result was one of the inputs.
		inputSet := make(map[string]struct{}, len(all))
		for _, target := range all {
			inputSet[target] = struct{}{}
		}

		for _, target := range got {
			_, ok := inputSet[target]
			g.Expect(ok).To(BeTrue())
		}

		// Every non-empty input appears in the result (round-trip).
		for _, target := range all {
			if target == "" {
				continue
			}

			_, found := seen[target]
			g.Expect(found).To(BeTrue())
		}
	})
}

func TestParseWikilinks_Dedupes(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	body := []byte("[[A]] and again [[A]] and once more [[A]] plus [[B]].")

	g.Expect(vaultgraph.ParseWikilinks(body)).To(Equal([]string{"A", "B"}))
}

func TestParseWikilinks_DoesNotSpanNewlines(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	body := []byte("partial [[start\nend]] not a link")

	g.Expect(vaultgraph.ParseWikilinks(body)).To(BeEmpty())
}

func TestParseWikilinks_Empty(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	g.Expect(vaultgraph.ParseWikilinks(nil)).To(BeEmpty())
	g.Expect(vaultgraph.ParseWikilinks([]byte("no links here, just prose."))).To(BeEmpty())
}

func TestParseWikilinks_IgnoresEmpty(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	body := []byte("empty [[]] valid [[Real]].")

	g.Expect(vaultgraph.ParseWikilinks(body)).To(Equal([]string{"Real"}))
}

func TestParseWikilinks_MultipleLinksFirstAppearanceOrder(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	body := []byte("ref [[B]] then [[A]] then [[C]].")

	g.Expect(vaultgraph.ParseWikilinks(body)).To(Equal([]string{"B", "A", "C"}))
}

func TestParseWikilinks_NoNesting(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// `]` terminates the match — `[[outer[[inner]]` parses as `[[outer[[inner]]`
	// where the body is `outer[[inner` (no `]` until the inner pair closes).
	body := []byte("[[outer[[inner]]")

	result := vaultgraph.ParseWikilinks(body)
	g.Expect(result).To(HaveLen(1))

	if len(result) < 1 {
		return
	}

	g.Expect(result[0]).To(Equal("outer[[inner"))
}

func TestParseWikilinks_SingleLink(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	body := []byte("see [[9o.2026-05-09.holistic-final-review]] for context.")

	g.Expect(vaultgraph.ParseWikilinks(body)).To(Equal([]string{"9o.2026-05-09.holistic-final-review"}))
}
