package embed_test

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/embed"
)

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

func TestContentHash_IsSha256OfBody(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	body := []byte("the body.\n")
	want := sha256.Sum256(body)
	g.Expect(embed.ContentHash([]byte("---\ntype: x\n---\nthe body.\n"))).
		To(Equal("sha256:" + hex.EncodeToString(want[:])))
}

func TestEmbedText_EpisodeMissingSituationFallsBackToBody(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	in := []byte("---\ntype: episode\nluhmann: \"5\"\n---\n\nEpisode body without situation.\n")
	g.Expect(string(embed.Text(in))).To(Equal("Episode body without situation.\n"))
}

func TestEmbedText_EpisodeReturnsSituationField(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	in := []byte(
		"---\ntype: episode\nsituation: evaluating agent memory\nluhmann: \"251\"\n---\n\nLong transcript body...\n",
	)
	g.Expect(string(embed.Text(in))).To(Equal("evaluating agent memory"))
}

func TestEmbedText_EpisodeSituationWithWhitespaceIsTrimmed(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	in := []byte("---\ntype: episode\nsituation:  spaced value  \nluhmann: \"7\"\n---\n\nBody.\n")
	g.Expect(string(embed.Text(in))).To(Equal("spaced value"))
}

func TestEmbedText_FactReturnsBody(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	in := []byte("---\ntype: fact\nluhmann: \"1\"\n---\n\nThe fact body.\n")
	g.Expect(string(embed.Text(in))).To(Equal("The fact body.\n"))
}

func TestEmbedText_FeedbackReturnsBody(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	// Feedback notes also have a situation: field — must NOT use it; embed the body.
	in := []byte("---\ntype: feedback\nsituation: When debugging a plugin\n---\n\nFeedback body.\n")
	g.Expect(string(embed.Text(in))).To(Equal("Feedback body.\n"))
}

func TestEmbedText_NoFrontmatterReturnsRaw(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	in := []byte("Just a plain note with no frontmatter.\n")
	g.Expect(string(embed.Text(in))).To(Equal("Just a plain note with no frontmatter.\n"))
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
