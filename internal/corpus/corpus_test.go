package corpus_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

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

			g := NewGomegaWithT(t)
			// When test calls corpus.Match with the input string
			m := patterns.Match(tc.input)
			// Then corpus.Match returns a non-nil match
			g.Expect(m).NotTo(BeNil(), "pattern %q should match %q", tc.pattern, tc.input)
		})
	}
}

// T-1: All 15 correction patterns match when embedded in messages with arbitrary context.
// Property: adding digit-only prefix/suffix to known-matching text does not prevent a match.
func TestT1_CorrectionPatternMatchesWithContext(t *testing.T) {
	t.Parallel()

	type patternCase struct {
		matchText string
		label     string
		anchored  bool // true if pattern starts with ^ (must appear at start of string)
	}

	cases := []patternCase{
		{"no, use specific files", "direct-negation", true},
		{"wait, that is wrong", "interruption", true},
		{"hold on a moment", "pause", true},
		{"that is wrong here", "wrong", false},
		{"don't do that", "dont", false},
		{"stop deleting files", "stop", false},
		{"try again please", "retry", false},
		{"go back to start", "revert", false},
		{"that's not correct", "negation", false},
		{"actually, use bun instead", "correction", true},
		{"remember to run tests", "reminder", false},
		{"start over from scratch", "restart", false},
		{"that is a pre-existing issue", "preexisting", false},
		{"you're still wrong", "persistence", false},
		{"that is incorrect behavior", "incorrect", false},
	}

	matcher := corpus.New(corpus.DefaultPatterns())

	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			t.Parallel()

			rapid.Check(t, func(rt *rapid.T) {
				g := NewGomegaWithT(t)

				// Use digit-only strings as context — no pattern can match in digits alone.
				suffix := rapid.StringOf(rapid.RuneFrom([]rune("0123456789"))).Draw(rt, "suffix")

				var message string
				if tc.anchored {
					// Anchored patterns must appear at the start of the string.
					message = tc.matchText + " " + suffix
				} else {
					prefix := rapid.StringOf(rapid.RuneFrom([]rune("0123456789"))).Draw(rt, "prefix")
					message = prefix + " " + tc.matchText + " " + suffix
				}

				result := matcher.Match(message)
				g.Expect(result).NotTo(BeNil(), "pattern %q should match in %q", tc.label, message)
			})
		})
	}
}

// T-2: Messages containing no correction patterns return nil.
// Property: digit-only strings never match any pattern.
func TestT2_NonMatchingMessageReturnsNil(t *testing.T) {
	t.Parallel()

	matcher := corpus.New(corpus.DefaultPatterns())

	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(t)

		// Digit-only strings cannot match any of the 15 word-based patterns.
		message := rapid.StringOf(rapid.RuneFrom([]rune("0123456789"))).Draw(rt, "message")

		result := matcher.Match(message)
		g.Expect(result).To(BeNil(), "digit string %q should not match any pattern", message)
	})
}

// T-3: The remember pattern produces Confidence "A".
func TestT3_RememberPatternProducesConfidenceA(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	matcher := corpus.New(corpus.DefaultPatterns())

	cases := []string{
		"remember to run tests",
		"remember that targ is the build tool",
		"please remember to commit with the right trailer",
	}

	for _, message := range cases {
		result := matcher.Match(message)
		g.Expect(result).NotTo(BeNil(), "expected match for %q", message)

		if result == nil {
			return
		}

		g.Expect(result.Confidence).To(Equal("A"), "remember pattern should produce Confidence A for %q", message)
	}
}

// T-4: All non-remember correction patterns produce Confidence "B".
// Property: arbitrary context around non-remember pattern text produces Confidence "B".
func TestT4_CorrectionPatternsProduceConfidenceB(t *testing.T) {
	t.Parallel()

	type patternCase struct {
		matchText string
		label     string
		anchored  bool
	}

	// All patterns except "remember" (label "reminder")
	cases := []patternCase{
		{"no, use specific files", "direct-negation", true},
		{"wait, that is wrong", "interruption", true},
		{"hold on a moment", "pause", true},
		{"that is wrong here", "wrong", false},
		{"don't do that", "dont", false},
		{"stop deleting files", "stop", false},
		{"try again please", "retry", false},
		{"go back to start", "revert", false},
		{"that's not correct", "negation", false},
		{"actually, use bun instead", "correction", true},
		{"start over from scratch", "restart", false},
		{"that is a pre-existing issue", "preexisting", false},
		{"you're still wrong", "persistence", false},
		{"that is incorrect behavior", "incorrect", false},
	}

	matcher := corpus.New(corpus.DefaultPatterns())

	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			t.Parallel()

			rapid.Check(t, func(rt *rapid.T) {
				g := NewGomegaWithT(t)

				suffix := rapid.StringOf(rapid.RuneFrom([]rune("0123456789"))).Draw(rt, "suffix")

				var message string
				if tc.anchored {
					message = tc.matchText + " " + suffix
				} else {
					prefix := rapid.StringOf(rapid.RuneFrom([]rune("0123456789"))).Draw(rt, "prefix")
					message = prefix + " " + tc.matchText + " " + suffix
				}

				result := matcher.Match(message)
				g.Expect(result).NotTo(BeNil(), "pattern %q should match in %q", tc.label, message)

				if result == nil {
					return
				}

				g.Expect(result.Confidence).To(Equal("B"),
					"non-remember pattern %q should produce Confidence B", tc.label)
			})
		})
	}
}
