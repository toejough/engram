package embed_test

// Embedding readers read a CRLF note as its LF form (#789, design D8): the
// situation and body a note is embedded from, and the content hash that
// marks its sidecar stale, are the same for a note and its LF conversion.

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/embed"
)

func TestEmbedReaders_CRLFNoteReadsAsLF(t *testing.T) {
	t.Parallel()

	lf := "---\ntype: fact\nsituation: when converting line endings\nsubject: a\n---\n\n" +
		"Information learned: body.\n\nmore\n"
	crlf := strings.ReplaceAll(lf, "\n", "\r\n")

	t.Run("situation", func(t *testing.T) {
		t.Parallel()
		NewWithT(t).Expect(string(embed.SituationText([]byte(crlf)))).To(Equal("when converting line endings"))
	})

	t.Run("body", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)
		g.Expect(string(embed.ExtractBody([]byte(crlf)))).To(Equal(string(embed.ExtractBody([]byte(lf)))))
		g.Expect(string(embed.BodyText([]byte(crlf)))).To(Equal(string(embed.BodyText([]byte(lf)))))
	})

	t.Run("content hash", func(t *testing.T) {
		t.Parallel()
		NewWithT(t).Expect(embed.ContentHash([]byte(crlf))).To(Equal(embed.ContentHash([]byte(lf))))
	})
}
