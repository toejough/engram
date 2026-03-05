package corpus_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"engram/internal/corpus"
)

// TestDefaultPatterns_Entries verifies every entry returned by DefaultPatterns has a compiled
// regex, a non-empty label, and a valid confidence tier. This exercises the full loop body.
func TestDefaultPatterns_Entries(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	patterns := corpus.DefaultPatterns()

	g.Expect(patterns).NotTo(BeEmpty())

	for _, pattern := range patterns {
		t.Run(pattern.Label, func(t *testing.T) {
			t.Parallel()

			g := NewGomegaWithT(t)
			g.Expect(pattern.Regex).NotTo(BeNil())
			g.Expect(pattern.Label).NotTo(BeEmpty())
			g.Expect(pattern.Confidence).To(Or(Equal("A"), Equal("B")))
		})
	}
}

func TestMatch_NoMatch(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	patterns := corpus.New(corpus.DefaultPatterns())
	g.Expect(patterns.Match("perfectly normal message")).To(BeNil())
}

// T-1: All 40 correction patterns match when embedded in messages with arbitrary context.
// Property: adding digit-only prefix/suffix to known-matching text does not prevent a match.
func TestT1_CorrectionPatternMatchesWithContext(t *testing.T) {
	t.Parallel()

	type patternCase struct {
		matchText string
		label     string
		anchored  bool // true if pattern starts with ^ (must appear at start of string)
	}

	cases := []patternCase{
		{"no. do it this way", "direct-negation", true},
		{"wait, that is wrong", "interruption", true},
		{"hold on a moment", "pause", true},
		{"that is wrong here", "wrong", false},
		{"don't do that", "dont", false},
		{"do not use that approach", "do-not", false},
		{"you should not do that", "should-not", false},
		{"you must not delete it", "must-not", false},
		{"stop deleting files", "stop", false},
		{"try again please", "retry", false},
		{"go back to start", "revert", false},
		{"that's not correct", "negation", false},
		{"actually, use bun instead", "correction", true},
		{"remember to run tests", "reminder", false},
		{"remember: always use DI", "reminder", false},
		{"you can't do that here", "you-cant", false},
		{"you cannot use that", "you-cannot", false},
		{"that won't work here", "that-wont", false},
		{"that will not compile", "that-will-not", false},
		{"start over from scratch", "restart", false},
		{"that is a pre-existing issue", "preexisting", false},
		{"you're still wrong", "persistence", false},
		{"that is incorrect behavior", "incorrect", false},
		{"from now on, always use targ", "standing-instruction", false},
		{"you should have checked first", "retrospective", false},
		{"you forgot to run the tests", "omission", false},
		{"you missed the edge case", "missed", false},
		{"I told you to use targ", "repeated-instruction", false},
		{"I already asked for tests", "repeated-request", false},
		{"rather than guessing, read the docs", "preference", false},
		{"not this, but that", "contrast", false},
		{"that's not what I asked for", "rejection", false},
		{"next time, check the tests first", "prospective", false},
		{"I just wanted a simple fix", "scope-complaint", false},
		{"this is over-engineered", "over-engineering", false},
		{"I only asked for a bug fix", "scope-restriction", false},
		{"this doesn't work at all", "broken-output", false},
		{"it's broken now", "broken", false},
		{"the build is not working", "not-working", false},
		{"we use targ for builds", "convention", false},
		{"the convention is snake_case", "convention-reference", false},
		{"in this project we use DI", "project-norm", false},
		{"leave it alone", "hands-off", false},
		{"hands off the config", "hands-off-explicit", false},
		{"that module is off limits", "off-limits", false},
		{"you misunderstood the requirement", "misunderstood", false},
		{"no, I mean the other function", "clarification", false},
		{"you misinterpreted my request", "misinterpreted", false},
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
					prefix := rapid.StringOf(rapid.RuneFrom([]rune("0123456789"))).
						Draw(rt, "prefix")
					message = prefix + " " + tc.matchText + " " + suffix
				}

				result := matcher.Match(message)
				g.Expect(result).NotTo(BeNil(), "pattern %q should match in %q", tc.label, message)
			})
		})
	}
}

func TestT21_All40PatternsMatchExpectedInput(t *testing.T) {
	t.Parallel()

	// Given each pattern from the corpus and its expected matching string
	cases := []struct {
		pattern string
		input   string
	}{
		{`^no[.,;!]`, "no. do it this way"},
		{`^wait`, "wait, that's wrong"},
		{`^hold on`, "hold on, let me check"},
		{`\bwrong\b`, "that's wrong"},
		{`\bdon't\s+\w+`, "don't use that"},
		{`\bdo\s+not\s+\w+`, "do not use that approach"},
		{`\bshould\s+not\b`, "you should not do that"},
		{`\bmust\s+not\b`, "you must not delete it"},
		{`\bstop\s+\w+ing`, "stop deleting files"},
		{`\btry again`, "try again with the right path"},
		{`\bgo back`, "go back to the previous version"},
		{`\bthat's not`, "that's not what I meant"},
		{`^actually,`, "actually, use bun instead"},
		{`\bremember[\s:,]`, "remember to run tests"},
		{`\byou\s+can'?t\b`, "you can't do that here"},
		{`\byou\s+cannot\b`, "you cannot use that"},
		{`\bthat\s+won'?t\b`, "that won't work here"},
		{`\bthat\s+will\s+not\b`, "that will not compile"},
		{`\bstart over`, "start over from scratch"},
		{`\bpre-?existing`, "that's a pre-existing issue"},
		{`\byou're still`, "you're still making that mistake"},
		{`\bincorrect`, "that's incorrect"},
		{`\bfrom\s+now\s+on\b`, "from now on, always use targ"},
		{`\byou\s+should\s+have\b`, "you should have checked first"},
		{`\byou\s+(?:forgot|overlooked)\s+to\b`, "you forgot to run the tests"},
		{`\byou\s+missed\b`, "you missed the edge case"},
		{`\bI\s+(?:told|already\s+told)\s+you\b`, "I told you to use targ"},
		{`\bI\s+already\s+(?:said|asked|mentioned)\b`, "I already asked for tests"},
		{`\brather\s+than\b`, "rather than guessing, read the docs"},
		{`\bnot\s+\w+,?\s+(?:but|instead)\b`, "not this, but that"},
		{`\bthat's\s+not\s+what\s+I\b`, "that's not what I asked for"},
		{`\bnext\s+time\b`, "next time, check the tests first"},
		{`\bjust\s+wanted\b`, "I just wanted a simple fix"},
		{`\bover-?engineer`, "this is over-engineered"},
		{`\bI\s+only\s+asked\b`, "I only asked for a bug fix"},
		{`\bdoes(?:n't| not)\s+work\b`, "this doesn't work at all"},
		{`\bit(?:'s| is)\s+broken\b`, "it's broken now"},
		{`\bnot\s+working\b`, "the build is not working"},
		{`\bwe\s+use\b`, "we use targ for builds"},
		{`\bthe\s+convention\b`, "the convention is snake_case"},
		{`\bin\s+this\s+(?:project|repo|codebase)\b`, "in this project we use DI"},
		{`\bleave\s+\w+\s+alone\b`, "leave it alone"},
		{`\bhands\s+off\b`, "hands off the config"},
		{`\boff\s+limits\b`, "that module is off limits"},
		{`\byou\s+misunderstood\b`, "you misunderstood the requirement"},
		{`\bno,?\s+I\s+mean\b`, "no, I mean the other function"},
		{`\bmisinterpreted\b`, "you misinterpreted my request"},
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

// T-2: Messages containing no correction patterns return nil.
// Property: digit-only strings never match any pattern.
func TestT2_NonMatchingMessageReturnsNil(t *testing.T) {
	t.Parallel()

	matcher := corpus.New(corpus.DefaultPatterns())

	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(t)

		// Digit-only strings cannot match any of the 40 word-based patterns.
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
		"remember: always use DI",
		"remember, no I/O in internal",
		"from now on, always use targ",
	}

	for _, message := range cases {
		result := matcher.Match(message)
		g.Expect(result).NotTo(BeNil(), "expected match for %q", message)

		if result == nil {
			return
		}

		g.Expect(result.Confidence).
			To(Equal("A"), "remember pattern should produce Confidence A for %q", message)
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

	// All patterns except "remember" (label "reminder") and "from now on" (standing-instruction, tier A)
	cases := []patternCase{
		{"no. do it this way", "direct-negation", true},
		{"wait, that is wrong", "interruption", true},
		{"hold on a moment", "pause", true},
		{"that is wrong here", "wrong", false},
		{"don't do that", "dont", false},
		{"do not use that approach", "do-not", false},
		{"you should not do that", "should-not", false},
		{"you must not delete it", "must-not", false},
		{"stop deleting files", "stop", false},
		{"try again please", "retry", false},
		{"go back to start", "revert", false},
		{"that's not correct", "negation", false},
		{"actually, use bun instead", "correction", true},
		{"start over from scratch", "restart", false},
		{"that is a pre-existing issue", "preexisting", false},
		{"you're still wrong", "persistence", false},
		{"that is incorrect behavior", "incorrect", false},
		{"you should have checked first", "retrospective", false},
		{"you forgot to run the tests", "omission", false},
		{"you missed the edge case", "missed", false},
		{"I told you to use targ", "repeated-instruction", false},
		{"I already asked for tests", "repeated-request", false},
		{"rather than guessing, read the docs", "preference", false},
		{"not this, but that", "contrast", false},
		{"that's not what I asked for", "rejection", false},
		{"next time, check the tests first", "prospective", false},
		{"I just wanted a simple fix", "scope-complaint", false},
		{"this is over-engineered", "over-engineering", false},
		{"I only asked for a bug fix", "scope-restriction", false},
		{"this doesn't work at all", "broken-output", false},
		{"it's broken now", "broken", false},
		{"the build is not working", "not-working", false},
		{"we use targ for builds", "convention", false},
		{"the convention is snake_case", "convention-reference", false},
		{"in this project we use DI", "project-norm", false},
		{"leave it alone", "hands-off", false},
		{"hands off the config", "hands-off-explicit", false},
		{"that module is off limits", "off-limits", false},
		{"you misunderstood the requirement", "misunderstood", false},
		{"no, I mean the other function", "clarification", false},
		{"you misinterpreted my request", "misinterpreted", false},
		{"you can't do that here", "you-cant", false},
		{"you cannot use that", "you-cannot", false},
		{"that won't work here", "that-wont", false},
		{"that will not compile", "that-will-not", false},
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
					prefix := rapid.StringOf(rapid.RuneFrom([]rune("0123456789"))).
						Draw(rt, "prefix")
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
