package corpus_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/corpus"
)

func TestMatch_NoMatch(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	patterns := corpus.New(corpus.DefaultPatterns())
	g.Expect(patterns.Match("perfectly normal message")).To(BeNil())
}

func TestT21_All15InitialPatternsMatchExpectedInput(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// Given each pattern from the initial corpus and its expected matching string
	cases := []struct {
		pattern string
		input   string
	}{
		{`^no,`, "no, use specific files"},
		{`^wait`, "wait, that's wrong"},
		{`^hold on`, "hold on, let me check"},
		{`\bwrong\b`, "that's wrong"},
		{`\bdon't\s+\w+`, "don't use that"},
		{`\bstop\s+\w+ing`, "stop deleting files"},
		{`\btry again`, "try again with the right path"},
		{`\bgo back`, "go back to the previous version"},
		{`\bthat's not`, "that's not what I meant"},
		{`^actually,`, "actually, use bun instead"},
		{`\bremember\s+(that|to)`, "remember to run tests"},
		{`\bstart over`, "start over from scratch"},
		{`\bpre-?existing`, "that's a pre-existing issue"},
		{`\byou're still`, "you're still making that mistake"},
		{`\bincorrect`, "that's incorrect"},
	}

	patterns := corpus.New(corpus.DefaultPatterns())

	for _, tc := range cases {
		t.Run(tc.pattern, func(t *testing.T) {
			t.Parallel()

			g = NewGomegaWithT(t)
			// When test calls corpus.Match with the input string
			m := patterns.Match(tc.input)
			// Then corpus.Match returns a non-nil match
			g.Expect(m).NotTo(BeNil(), "pattern %q should match %q", tc.pattern, tc.input)
		})
	}
}
