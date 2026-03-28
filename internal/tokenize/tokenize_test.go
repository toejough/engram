package tokenize_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/tokenize"
)

func TestTokenize_BasicSplit(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	tokens := tokenize.Tokenize("Hello World")
	g.Expect(tokens).To(Equal([]string{"hello", "world"}))
}

func TestTokenize_PunctuationStripped(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	tokens := tokenize.Tokenize("configuration-management, testing!")
	g.Expect(tokens).To(Equal([]string{"configuration", "management", "testing"}))
}

func TestTokenize_DigitsIncluded(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	tokens := tokenize.Tokenize("http2 h264codec")
	g.Expect(tokens).To(Equal([]string{"http2", "h264codec"}))
}

func TestTokenize_EmptyString(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	tokens := tokenize.Tokenize("")
	g.Expect(tokens).To(BeEmpty())
}

func TestTokenize_OnlyPunctuation(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	tokens := tokenize.Tokenize("---!!!")
	g.Expect(tokens).To(BeEmpty())
}

func TestFrequencies_CountsDuplicates(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	freqs := tokenize.Frequencies("the cat sat on the mat")
	g.Expect(freqs).To(Equal(map[string]int{
		"the": 2, "cat": 1, "sat": 1, "on": 1, "mat": 1,
	}))
}

func TestFrequencies_EmptyString(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	freqs := tokenize.Frequencies("")
	g.Expect(freqs).To(BeEmpty())
}
