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

func TestBodyText_StripsFrontmatter(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	raw := []byte("---\ntype: fact\nsituation: x\n---\n\nthe body\n")
	g.Expect(string(embed.BodyText(raw))).To(Equal("the body\n"))
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
