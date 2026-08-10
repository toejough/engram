package vaultgraph_test

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/vaultgraph"
)

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

// TestParseWikilinks_MdNormalizationProperty verifies for any basename that
// linking it bare and linking it .md-suffixed parse to identical target lists.
//
// The generator excludes basenames whose final dot-segment is literally "md"
// (e.g. "0.md"): for that one case the property provably cannot hold, not
// because of a parser defect. The raw wikilink text for the bare form,
// "[[0.md]]", is byte-identical whether the author meant a legacy
// .md-suffixed link to a note named "0" or a literal link to a note whose
// own basename is "0.md" — there is no information in the raw text to
// disambiguate, and Obsidian has the same limitation. ParseWikilinks
// resolves that ambiguity by always treating a trailing ".md" as the
// legacy-suffixed form (see TestParseWikilinks_MdSuffixLiteralBasenameEdgeCase),
// which is correct per vault-wikilink-resolution's normalization spec but
// necessarily breaks the bare/suffixed equivalence for this input. Real
// vault basenames (`<id>.<date>.<slug>.md` — kebab-case slugs) don't produce
// this case, so excluding it from the generator does not narrow real
// coverage. Do not remove this exclusion without also changing how
// ParseWikilinks resolves the ambiguity.
func TestParseWikilinks_MdNormalizationProperty(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		name := rapid.StringMatching(`[0-9]+[a-z0-9]*\.[a-z0-9-]+`).
			Filter(func(s string) bool {
				segments := strings.Split(s, ".")

				return segments[len(segments)-1] != "md"
			}).
			Draw(rt, "name")

		bare := vaultgraph.ParseWikilinks([]byte("see [[" + name + "]] here"))
		suffixed := vaultgraph.ParseWikilinks([]byte("see [[" + name + ".md]] here"))

		g.Expect(suffixed).To(Equal(bare), "the two link forms must parse identically")
	})
}

// TestParseWikilinks_MdSuffixDedupesWithBareForm verifies normalization runs
// before dedup: the same target linked bare and .md-suffixed yields one entry.
func TestParseWikilinks_MdSuffixDedupesWithBareForm(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	body := []byte("see [[x]] and [[x.md]] and [[y.md]] and [[y]].")

	g.Expect(vaultgraph.ParseWikilinks(body)).To(Equal([]string{"x", "y"}))
}

// TestParseWikilinks_MdSuffixLiteralBasenameEdgeCase documents the accepted,
// intended behavior for the one input the property test above excludes: a
// bare link whose final dot-segment is literally "md" (e.g. "[[0.md]]").
// ParseWikilinks always resolves this as the legacy .md-suffixed form,
// yielding the extension-less basename "0" — never a literal target named
// "0.md". This matches vault-wikilink-resolution's normalization
// requirement ("Parser normalizes .md-suffixed wikilink targets"), which
// gives .md-suffix stripping priority with no exception. If this test ever
// needs to change to accept a literal "0.md" target, the property test's
// generator exclusion above must be revisited too.
func TestParseWikilinks_MdSuffixLiteralBasenameEdgeCase(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	got := vaultgraph.ParseWikilinks([]byte("see [[0.md]] here"))

	g.Expect(got).To(Equal([]string{"0"}))
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

// TestParseWikilinks_NormalizesMdSuffixedTargets verifies a wikilink written
// with the .md extension resolves to the extension-less canonical basename.
func TestParseWikilinks_NormalizesMdSuffixedTargets(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	body := []byte("Supersedes: [[210.2026-07-10.old-definition.md]] — updates: retired")

	g.Expect(vaultgraph.ParseWikilinks(body)).To(Equal([]string{"210.2026-07-10.old-definition"}))
}

func TestParseWikilinks_SingleLink(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	body := []byte("see [[9o.2026-05-09.holistic-final-review]] for context.")

	g.Expect(vaultgraph.ParseWikilinks(body)).
		To(Equal([]string{"9o.2026-05-09.holistic-final-review"}))
}

func TestParseWikilinks_SkipsFencedBlock(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	body := []byte(strings.Join([]string{
		"outside [[y]]",
		"```",
		"inside [[x]]",
		"```",
		"after [[z]]",
	}, "\n"))

	// `[[x]]` lives inside a fenced code block — Obsidian does not resolve it, so neither do we.
	g.Expect(vaultgraph.ParseWikilinks(body)).To(Equal([]string{"y", "z"}))
}

func TestParseWikilinks_UnclosedFenceRunsToEnd(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// An opened-but-never-closed fence consumes the rest of the note (CommonMark behavior),
	// so links after the opener are dropped.
	body := []byte(strings.Join([]string{
		"outside [[y]]",
		"```",
		"inside [[x]]",
	}, "\n"))

	g.Expect(vaultgraph.ParseWikilinks(body)).To(Equal([]string{"y"}))
}

func TestParseWikilinks_VariableLengthFence(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// A 4-backtick fence wraps a transcript that itself contains a 3-backtick line.
	// The inner 3-backtick line must NOT close the 4-backtick fence (3 < 4), so every
	// wikilink between the outer fences is dropped.
	body := []byte(strings.Join([]string{
		"outside [[y]]",
		"````",
		"a [[drop1]]",
		"```",
		"b [[drop2]]",
		"````",
		"after [[z]]",
	}, "\n"))

	g.Expect(vaultgraph.ParseWikilinks(body)).To(Equal([]string{"y", "z"}))
}
