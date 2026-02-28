package corpus_test

// Tests for correction pattern corpus (pure implementation, no I/O).
// T-21 moved here from correct_test.go.
// Won't compile yet — RED phase.

import (
	"testing"

	"engram/internal/corpus"
	"github.com/onsi/gomega"
)

// T-21: Each of the 15 initial patterns matches expected input.
func TestCorpus_AllFifteenPatternsMatch(t *testing.T) {
	g := gomega.NewWithT(t)

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

	for _, tc := range cases {
		t.Run(tc.pattern, func(t *testing.T) {
			c, err := corpus.NewRegex([]corpus.Pattern{{Regex: tc.pattern, Label: tc.pattern}})
			g.Expect(err).ToNot(gomega.HaveOccurred())

			m, err := c.Match(tc.input)
			g.Expect(err).ToNot(gomega.HaveOccurred())
			g.Expect(m).ToNot(gomega.BeNil(), "pattern %q did not match %q", tc.pattern, tc.input)
		})
	}
}
