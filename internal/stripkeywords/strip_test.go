package stripkeywords_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"engram/internal/stripkeywords"
)

func TestStripKeywordsSuffix_NoSuffix(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	g.Expect(stripkeywords.StripKeywordsSuffix("when running tests")).
		To(Equal("when running tests"))
}

func TestStripKeywordsSuffix_StripsSuffix(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	g.Expect(stripkeywords.StripKeywordsSuffix("when running tests\nKeywords: go, test, targ")).
		To(Equal("when running tests"))
}

func TestStripKeywordsSuffix_MultipleNewlines(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	g.Expect(stripkeywords.StripKeywordsSuffix("line one\nline two\nKeywords: a, b")).
		To(Equal("line one\nline two"))
}

func TestStripKeywordsSuffix_KeywordsWithNoLeadingNewline(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	// "Keywords: ..." without a preceding \n must NOT be stripped (not the suffix pattern)
	g.Expect(stripkeywords.StripKeywordsSuffix("Keywords: foo, bar")).
		To(Equal("Keywords: foo, bar"))
}

func TestStripKeywordsSuffix_Empty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	g.Expect(stripkeywords.StripKeywordsSuffix("")).To(Equal(""))
}

func TestStripKeywordsSuffix_Idempotent(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		base := rapid.StringMatching(`[A-Za-z0-9 .,!?;:'\-]{1,80}`).Draw(rt, "base")

		withSuffix := base + "\nKeywords: " +
			rapid.StringMatching(`[A-Za-z0-9 ,]{1,40}`).Draw(rt, "keywords")

		once := stripkeywords.StripKeywordsSuffix(withSuffix)
		twice := stripkeywords.StripKeywordsSuffix(once)

		g.Expect(twice).To(Equal(once))
		g.Expect(once).NotTo(ContainSubstring("\nKeywords:"))
	})
}

func TestStripKeywordsSuffix_NeverCorruptsContent(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		base := rapid.StringMatching(`[A-Za-z0-9 .,!?;'\-]{1,80}`).Draw(rt, "base")
		input := base + "\nKeywords: " +
			rapid.StringMatching(`[A-Za-z0-9 ,]{1,40}`).Draw(rt, "kw")

		result := stripkeywords.StripKeywordsSuffix(input)

		g.Expect(result).To(Equal(base))
	})
}
