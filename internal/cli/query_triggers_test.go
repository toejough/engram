package cli_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

func TestMatchTriggers_WholeWord(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		text    string
		trigger string
		want    bool
	}{
		{"bare word mid sentence", "please curate the offers", "curate", true},
		{"bare word at start", "curate the offers", "curate", true},
		{"bare word at end", "time to curate", "curate", true},
		{"bare word is whole text", "curate", "curate", true},
		{"trailing punctuation", "Curate!", "curate", true},
		{"parentheses", "(curate)", "curate", true},
		{"slash before is a boundary", "/curate", "curate", true},
		{"hyphen is a boundary", "re-curate-now", "curate", true},
		{"underscore is a boundary", "do_curate_it", "curate", true},
		{"inside longer word (prefix)", "that is accurate", "curate", false},
		{"inside longer word (in-)", "inaccurate", "curate", false},
		{"suffix letters", "curated offers", "curate", false},
		{"plural suffix", "curates", "curate", false},
		{"pleased is not please", "I am pleased", "please", false},
		{"displease is not please", "do not displease", "please", false},
		{"first occurrence fails, later matches", "accurate, curate now", "curate", true},
		{"all occurrences fail", "accurate and curated", "curate", false},
		{"digit after blocks", "curate2 now", "curate", false},
		{"digit before blocks", "2curate now", "curate", false},
		{"unicode letter before blocks", "écurate", "curate", false},
		{"unicode letter after blocks", "curateé", "curate", false},
		{"unicode punctuation neighbor allows", "«curate»", "curate", true},
		{"case insensitive", "CURATE the offers", "Curate", true},
		{"slash trigger at start", "/please fix it", "/please", true},
		{"slash trigger at end", "do /please", "/please", true},
		{"slash trigger needs no left boundary", "x/please fix", "/please", true},
		{"slash trigger still needs right boundary", "/pleased", "/please", false},
		{"phrase after prefix word", "Please take this end-to-end: X", "take this end-to-end", true},
		{"phrase left boundary enforced", "retake this end-to-end", "take this end-to-end", false},
		{"trigger ending in punctuation needs no right boundary", "ok e.g.x", "e.g.", true},
		{"whitespace collapse", "PLEASE   take\tthis\nend-to-end", "take this end-to-end", true},
		{"empty text", "", "curate", false},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)
			index := map[string][]string{"note": {testCase.trigger}}

			got := cli.ExportMatchTriggers(testCase.text, index)
			if testCase.want {
				g.Expect(got).To(Equal([]string{"note"}))
			} else {
				g.Expect(got).To(BeEmpty())
			}
		})
	}
}
