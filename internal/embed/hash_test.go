package embed_test

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/embed"
)

func TestExtractBody_StripsFrontmatter(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	in := []byte("---\ntype: fact\nluhmann: \"5\"\n---\n\nThis is the body.\n")
	g.Expect(string(embed.ExtractBody(in))).To(Equal("This is the body.\n"))
}

func TestExtractBody_NoFrontmatter(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	in := []byte("Just a body, no frontmatter.\n")
	g.Expect(string(embed.ExtractBody(in))).To(Equal("Just a body, no frontmatter.\n"))
}

func TestExtractBody_UnterminatedFrontmatterReturnedUnchanged(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	in := []byte("---\ntype: fact\n(no closing delimiter)\n")
	g.Expect(string(embed.ExtractBody(in))).To(Equal(string(in)))
}

func TestContentHash_IsSha256OfBody(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	body := []byte("the body.\n")
	want := sha256.Sum256(body)
	g.Expect(embed.ContentHash([]byte("---\ntype: x\n---\nthe body.\n"))).
		To(Equal("sha256:" + hex.EncodeToString(want[:])))
}

func TestContentHash_FrontmatterChangeDoesNotChangeHash(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	a := []byte("---\ntype: fact\nluhmann: \"1\"\n---\nshared body.\n")
	b := []byte("---\ntype: fact\nluhmann: \"1\"\nextra: added\n---\nshared body.\n")
	g.Expect(embed.ContentHash(a)).To(Equal(embed.ContentHash(b)))
}
