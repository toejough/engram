package embed_test

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/embed"
)

func TestBodyText_ExcludesRelatedToSection(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	raw := []byte("---\ntype: fact\nluhmann: \"1\"\n---\n\n" +
		"Information learned: when in X, S P O.\n\n" +
		"Related to:\n- [[2.note]] — because.\n- [[3.note]] — also.\n")
	want := "Information learned: when in X, S P O.\n"
	g.Expect(string(embed.BodyText(raw))).To(Equal(want))
}

func TestBodyText_ExcludesSupersedesLines(t *testing.T) {
	t.Parallel()

	// Same machine-written category as the Vocab: line: `Supersedes:` body
	// lines are replace-whole channel content and must not feed the vector/hash.
	g := NewWithT(t)
	raw := []byte("---\ntype: fact\nluhmann: \"2\"\n---\n\n" +
		"Information learned: when in X, S P O.\n\n" +
		"Supersedes: [[9.old-note]] — updates: the old claim.\n" +
		"Supersedes: [[7.other]] — narrows: the scope.\n")
	want := "Information learned: when in X, S P O.\n"
	g.Expect(string(embed.BodyText(raw))).To(Equal(want))
}

func TestBodyText_InlineRelatedToProseIsNotStripped(t *testing.T) {
	t.Parallel()

	// "Related to:" appears as inline prose with no bullet block beneath it,
	// so the whole body — including that line — must survive.
	g := NewWithT(t)
	raw := []byte("---\ntype: fact\nluhmann: \"1\"\n---\n\n" +
		"The bug was Related to: a missing nil guard in the parser.\n")
	want := "The bug was Related to: a missing nil guard in the parser.\n"
	g.Expect(string(embed.BodyText(raw))).To(Equal(want))
}

func TestBodyText_MarkerFollowedByProseIsNotStripped(t *testing.T) {
	t.Parallel()

	// A "Related to:" marker line whose following non-blank line is prose (not
	// a "- [[" bullet) is not a relation block; nothing is stripped.
	g := NewWithT(t)
	raw := []byte("---\ntype: fact\nluhmann: \"1\"\n---\n\n" +
		"Body line.\n\nRelated to:\nsee the design doc for context.\n")
	want := "Body line.\n\nRelated to:\nsee the design doc for context.\n"
	g.Expect(string(embed.BodyText(raw))).To(Equal(want))
}

func TestBodyText_NoFrontmatterReturnsRaw(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	in := []byte("Just a plain note with no frontmatter.\n")
	g.Expect(string(embed.BodyText(in))).To(Equal("Just a plain note with no frontmatter.\n"))
}

func TestBodyText_NormalizesTrailingBlankLines(t *testing.T) {
	t.Parallel()

	// The learn renderers end bodies with "\n\n" (97/133 live notes), while
	// the channel writers trim trailing blanks before appending their line —
	// the original trailing-blank count is unrecoverable after a write. The
	// hash must therefore be insensitive to trailing blank lines on BOTH
	// sides: BodyText normalizes them to a single trailing newline.
	g := NewWithT(t)
	doubleNewline := []byte("---\ntype: fact\nluhmann: \"4\"\n---\n\n" +
		"Information learned: when in X, S P O.\n\n")
	single := []byte("---\ntype: fact\nluhmann: \"4\"\n---\n\n" +
		"Information learned: when in X, S P O.\n")
	want := "Information learned: when in X, S P O.\n"
	g.Expect(string(embed.BodyText(doubleNewline))).To(Equal(want))
	g.Expect(embed.ContentHash(doubleNewline)).To(Equal(embed.ContentHash(single)))
}

func TestBodyText_StripsFrontmatter(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	raw := []byte("---\ntype: fact\nsituation: x\n---\n\nthe body\n")
	g.Expect(string(embed.BodyText(raw))).To(Equal("the body\n"))
}

func TestBodyText_VocabLineIsOrdinaryBody(t *testing.T) {
	t.Parallel()

	// The Vocab: body line is ordinary user prose now (the writer no longer
	// strips it, migration-by-touch is retired) — BodyText must include it,
	// not treat it as a machine-written channel line.
	g := NewWithT(t)
	raw := []byte("---\ntype: fact\nluhmann: \"1\"\n---\n\n" +
		"Information learned: when in X, S P O.\n\n" +
		"Vocab: [[vocab.eval-methodology]]\n")
	want := "Information learned: when in X, S P O.\n\nVocab: [[vocab.eval-methodology]]\n"
	g.Expect(string(embed.BodyText(raw))).To(Equal(want))
}

func TestContentHash_ChangesWhenEitherSourceChanges(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	base := []byte("---\ntype: fact\nsituation: A\n---\n\nbody B\n")
	sitChanged := []byte("---\ntype: fact\nsituation: A2\n---\n\nbody B\n")
	bodyChanged := []byte("---\ntype: fact\nsituation: A\n---\n\nbody B2\n")

	g.Expect(embed.ContentHash(base)).NotTo(Equal(embed.ContentHash(sitChanged)))
	g.Expect(embed.ContentHash(base)).NotTo(Equal(embed.ContentHash(bodyChanged)))
}

func TestContentHash_EpisodeSituationChangeChangesHash(t *testing.T) {
	t.Parallel()

	// Episodes embed their situation:, so a situation edit must change the
	// staleness hash even when the body is byte-identical. Otherwise an
	// edited episode reads as fresh against its outdated vector.
	g := NewWithT(t)
	a := []byte("---\ntype: episode\nsituation: evaluating agent memory\n---\nshared body.\n")
	b := []byte("---\ntype: episode\nsituation: debugging a flaky test\n---\nshared body.\n")
	g.Expect(embed.ContentHash(a)).NotTo(Equal(embed.ContentHash(b)))
}

func TestContentHash_FactTracksBody(t *testing.T) {
	t.Parallel()

	// Facts embed their body, so the hash must change when the body changes.
	g := NewWithT(t)
	a := []byte("---\ntype: fact\nluhmann: \"1\"\n---\noriginal body.\n")
	b := []byte("---\ntype: fact\nluhmann: \"1\"\n---\nedited body.\n")
	g.Expect(embed.ContentHash(a)).NotTo(Equal(embed.ContentHash(b)))
}

func TestContentHash_FrontmatterChangeDoesNotChangeHash(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	a := []byte("---\ntype: fact\nluhmann: \"1\"\n---\nshared body.\n")
	b := []byte("---\ntype: fact\nluhmann: \"1\"\nextra: added\n---\nshared body.\n")
	g.Expect(embed.ContentHash(a)).To(Equal(embed.ContentHash(b)))
}

func TestContentHash_IgnoresRelatedToLinkEdits(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	noBlock := []byte("---\ntype: fact\nluhmann: \"1\"\n---\n\n" +
		"Information learned: when in X, S P O.\n")
	withBlock := []byte("---\ntype: fact\nluhmann: \"1\"\n---\n\n" +
		"Information learned: when in X, S P O.\n\n" +
		"Related to:\n- [[2.note]] — because.\n")
	diffLinks := []byte("---\ntype: fact\nluhmann: \"1\"\n---\n\n" +
		"Information learned: when in X, S P O.\n\n" +
		"Related to:\n- [[2.note]] — because.\n- [[9.other]] — added later.\n")

	g.Expect(embed.ContentHash(noBlock)).To(Equal(embed.ContentHash(withBlock)))
	g.Expect(embed.ContentHash(withBlock)).To(Equal(embed.ContentHash(diffLinks)))
}

func TestContentHash_IsSha256OfSituationAndBody(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	raw := []byte("---\ntype: fact\nsituation: when X\n---\n\nthe body.\n")

	hasher := sha256.New()
	hasher.Write([]byte("when X"))
	hasher.Write([]byte{0})
	hasher.Write([]byte("the body.\n"))

	g.Expect(embed.ContentHash(raw)).
		To(Equal("sha256:" + hex.EncodeToString(hasher.Sum(nil))))
}

func TestContributorsBodyMarker_IsExported(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	g.Expect(embed.ContributorsBodyMarker).To(Equal("Contributors:"))
	g.Expect(embed.AnsweredByBodyMarker).To(Equal("Answered by:"))
	g.Expect(embed.AnswersBodyMarker).To(Equal("Answers:"))
}

func TestExtractBody_NoFrontmatter(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	in := []byte("Just a body, no frontmatter.\n")
	g.Expect(string(embed.ExtractBody(in))).To(Equal("Just a body, no frontmatter.\n"))
}

func TestExtractBody_StripsFrontmatter(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	in := []byte("---\ntype: fact\nluhmann: \"5\"\n---\n\nThis is the body.\n")
	g.Expect(string(embed.ExtractBody(in))).To(Equal("This is the body.\n"))
}

func TestExtractBody_UnterminatedFrontmatterReturnedUnchanged(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	in := []byte("---\ntype: fact\n(no closing delimiter)\n")
	g.Expect(string(embed.ExtractBody(in))).To(Equal(string(in)))
}

func TestSituationText_ExtractsFieldForAnyType(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	fact := []byte("---\ntype: fact\nsituation: when wiring a Go CLI\n---\n\nbody here\n")
	g.Expect(string(embed.SituationText(fact))).To(Equal("when wiring a Go CLI"))

	noFM := []byte("just a body, no frontmatter\n")
	g.Expect(embed.SituationText(noFM)).To(BeEmpty())
}

func TestSituationText_WhitespaceIsTrimmed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	in := []byte("---\ntype: episode\nsituation:  spaced value  \nluhmann: \"7\"\n---\n\nBody.\n")
	g.Expect(string(embed.SituationText(in))).To(Equal("spaced value"))
}

func TestStripMachineLines_QAMarkersRemoved(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "contributors_line_stripped",
			in:   "Body line.\n\nContributors: [[100.note]], [[101.note]]\n",
			want: "Body line.\n",
		},
		{
			name: "answered_by_line_stripped",
			in:   "What is the question?\n\nAnswered by: [[qa.2026-07-03.slug.a]]\n",
			want: "What is the question?\n",
		},
		{
			name: "answers_line_stripped",
			in:   "The answer body.\n\nAnswers: [[qa.2026-07-03.slug.q]]\n",
			want: "The answer body.\n",
		},
		{
			name: "all_three_stripped_together",
			in: "Body.\n\nAnswered by: [[qa.2026-07-03.slug.a]]" +
				"\nAnswers: [[qa.2026-07-03.slug.q]]\nContributors: [[100.note]]\n",
			want: "Body.\n",
		},
		{
			name: "no_markers_unchanged",
			in:   "Body without machine lines.\n",
			want: "Body without machine lines.\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)
			// BodyText strips frontmatter first; wrap in frontmatter to exercise BodyText.
			raw := []byte("---\ntype: qa-answer\n---\n\n" + tc.in)
			got := string(embed.BodyText(raw))
			g.Expect(got).To(Equal(tc.want))
		})
	}
}
