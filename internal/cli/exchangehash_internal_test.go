package cli

// Exchange-hash pins for the rename's CRLF→LF conversion (design D5
// "Exchange hash"; fix-show-amend-reparent-frontmatter task 4.2). The first
// three pin existing hashing behavior and passed on first run; they record
// why converting a CRLF note is harmless to the exchange.

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

// TestDecodeExchangeFrontmatter_CRLFFrontmatterReadsAsLF pins that a note
// whose frontmatter is CRLF has its exchange fields read from its LF form
// (#789 design D4), replacing fix-show-amend-reparent-frontmatter's pin
// that such a note had no readable xid.
func TestDecodeExchangeFrontmatter_CRLFFrontmatterReadsAsLF(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	crlf, err := decodeExchangeFrontmatter([]byte(crlfExchangeNoteLF(true, true)))
	g.Expect(err).NotTo(HaveOccurred())

	lf, lfErr := decodeExchangeFrontmatter(toLF([]byte(crlfExchangeNoteLF(true, true))))
	g.Expect(lfErr).NotTo(HaveOccurred())
	g.Expect(crlf).To(Equal(lf))
	g.Expect(crlf.XID).NotTo(BeEmpty())
}

// TestExchangeHash_CRLFBodyUnchangedByConversion pins that converting a note
// with LF frontmatter and a CRLF body to LF leaves its exchange hash
// unchanged. With an LF blank line after the closing delimiter this held
// already (canonicalExchangeBody normalizes the body); with that blank line
// itself CRLF — the whole body CRLF — the hash used to keep a leading blank
// line, so conversion moved it.
func TestExchangeHash_CRLFBodyUnchangedByConversion(t *testing.T) {
	t.Parallel()

	for name, pre := range map[string]string{
		"LF separator line":   strings.Replace(crlfExchangeNoteLF(false, true), "---\n\r\n", "---\n\n", 1),
		"CRLF separator line": crlfExchangeNoteLF(false, true),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			before, beforeErr := exchangeHash([]byte(pre))
			g.Expect(beforeErr).NotTo(HaveOccurred())

			after, afterErr := exchangeHash(toLF([]byte(pre)))
			g.Expect(afterErr).NotTo(HaveOccurred())

			g.Expect(after).To(Equal(before))
		})
	}
}

// TestExchangeHash_ConvertedCRLFFrontmatterMatchesLFAuthored pins that a
// note converted from CRLF frontmatter hashes the same as the same note
// authored with LF line endings, so a hash recorded before the file became
// CRLF matches again.
func TestExchangeHash_ConvertedCRLFFrontmatterMatchesLFAuthored(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	converted, convertedErr := exchangeHash(toLF([]byte(crlfExchangeNoteLF(true, true))))
	g.Expect(convertedErr).NotTo(HaveOccurred())

	authored, authoredErr := exchangeHash([]byte(crlfExchangeNoteLF(false, false)))
	g.Expect(authoredErr).NotTo(HaveOccurred())

	g.Expect(converted).To(Equal(authored))
}

// TestToLF converts every CRLF pair to LF and leaves a lone CR alone.
func TestToLF(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(string(toLF([]byte("a\r\nb\rc\r\n\r\n")))).To(Equal("a\nb\rc\n\n"))
	g.Expect(toLF([]byte("plain\n"))).To(Equal([]byte("plain\n")))
}

// crlfExchangeNoteLF renders a fact note carrying exchange fields, with its
// frontmatter and body lines ending in CRLF when the matching flag is set.
func crlfExchangeNoteLF(crlfFrontmatter, crlfBody bool) string {
	frontmatter := "---\ntype: fact\nsituation: s\nsubject: a\npredicate: b\nobject: c\n" +
		"luhmann: \"12\"\nxid: 7f3c0a9e1b2d4c5f8a6e9d0c1b2a3f4e\n---\n"
	body := "\nInformation learned: when in s, a b c.\n\nmore text\n"

	if crlfFrontmatter {
		frontmatter = strings.ReplaceAll(frontmatter, "\n", "\r\n")
	}

	if crlfBody {
		body = strings.ReplaceAll(body, "\n", "\r\n")
	}

	return frontmatter + body
}
