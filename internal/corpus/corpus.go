// Package corpus provides correction pattern matching against user messages.
package corpus

import "regexp"

// Corpus holds a set of correction patterns and matches them against messages.
type Corpus struct {
	patterns []Pattern
}

// New creates a Corpus from the given patterns.
func New(patterns []Pattern) *Corpus {
	return &Corpus{patterns: patterns}
}

// Match returns the first pattern that matches message, or nil.
func (c *Corpus) Match(message string) *Match {
	for _, pattern := range c.patterns {
		if loc := pattern.Regex.FindStringIndex(message); loc != nil {
			return &Match{
				Pattern:    pattern,
				Text:       message[loc[0]:loc[1]],
				Confidence: pattern.Confidence,
			}
		}
	}

	return nil
}

// Match holds a matched pattern and the matched text.
type Match struct {
	Pattern    Pattern
	Text       string
	Confidence string // "A" for remember patterns, "B" for correction patterns
}

// Pattern pairs a compiled regex with a human-readable label and confidence tier.
type Pattern struct {
	Regex      *regexp.Regexp
	Label      string
	Confidence string // "A" for remember patterns, "B" for correction patterns
}

// DefaultPatterns returns built-in correction patterns for detecting user feedback signals.
//
//nolint:funlen // Pattern registry, not complexity: each entry is a correction signal definition.
func DefaultPatterns() []Pattern {
	raw := []struct {
		regex      string
		label      string
		confidence string
	}{
		// Direct negation and interruption
		{`(?i)^no[.,;!]`, "direct-negation", confidenceTierB},
		{`(?i)^wait`, "interruption", confidenceTierB},
		{`(?i)^hold on`, "pause", confidenceTierB},
		{`(?i)\bwrong\b`, "wrong", confidenceTierB},

		// Prohibition patterns
		{`(?i)\bdon't\s+\w+`, "dont", confidenceTierB},
		{`(?i)\bdo\s+not\s+\w+`, "do-not", confidenceTierB},
		{`(?i)\bshould\s+not\b`, "should-not", confidenceTierB},
		{`(?i)\bmust\s+not\b`, "must-not", confidenceTierB},

		// Retry and revert
		{`(?i)\bstop\s+\w+ing`, "stop", confidenceTierB},
		{`(?i)\btry again`, "retry", confidenceTierB},
		{`(?i)\bgo back`, "revert", confidenceTierB},
		{`(?i)\bthat's not`, "negation", confidenceTierB},
		{`(?i)^actually,`, "correction", confidenceTierB},

		// Standing instructions (tier A)
		{`(?i)\bremember[\s:,]`, "reminder", confidenceTierA},
		{`(?i)\bfrom\s+now\s+on\b`, "standing-instruction", confidenceTierA},

		// Capability negation
		{`(?i)\byou\s+can'?t\b`, "you-cant", confidenceTierB},
		{`(?i)\byou\s+cannot\b`, "you-cannot", confidenceTierB},
		{`(?i)\bthat\s+won'?t\b`, "that-wont", confidenceTierB},
		{`(?i)\bthat\s+will\s+not\b`, "that-will-not", confidenceTierB},

		// Restart and state
		{`(?i)\bstart over`, "restart", confidenceTierB},
		{`(?i)\bpre-?existing`, "preexisting", confidenceTierB},
		{`(?i)\byou're still`, "persistence", confidenceTierB},
		{`(?i)\bincorrect`, "incorrect", confidenceTierB},

		// Retrospective and omission
		{`(?i)\byou\s+should\s+have\b`, "retrospective", confidenceTierB},
		{`(?i)\byou\s+(?:forgot|overlooked)\s+to\b`, "omission", confidenceTierB},
		{`(?i)\byou\s+missed\b`, "missed", confidenceTierB},
		{`(?i)\bI\s+(?:told|already\s+told)\s+you\b`, "repeated-instruction", confidenceTierB},
		{`(?i)\bI\s+already\s+(?:said|asked|mentioned)\b`, "repeated-request", confidenceTierB},

		// Preference and contrast
		{`(?i)\brather\s+than\b`, "preference", confidenceTierB},
		{`(?i)\bnot\s+\w+,?\s+(?:but|instead)\b`, "contrast", confidenceTierB},
		{`(?i)\bthat's\s+not\s+what\s+I\b`, "rejection", confidenceTierB},
		{`(?i)\bnext\s+time\b`, "prospective", confidenceTierB},

		// Scope / Over-engineering (issue #24)
		{`(?i)\bjust\s+wanted\b`, "scope-complaint", confidenceTierB},
		{`(?i)\bover-?engineer`, "over-engineering", confidenceTierB},
		{`(?i)\bI\s+only\s+asked\b`, "scope-restriction", confidenceTierB},

		// Quality Complaints (issue #24)
		{`(?i)\bdoes(?:n't| not)\s+work\b`, "broken-output", confidenceTierB},
		{`(?i)\bit(?:'s| is)\s+broken\b`, "broken", confidenceTierB},
		{`(?i)\bnot\s+working\b`, "not-working", confidenceTierB},

		// Style / Convention (issue #24)
		{`(?i)\bwe\s+use\b`, "convention", confidenceTierB},
		{`(?i)\bthe\s+convention\b`, "convention-reference", confidenceTierB},
		{`(?i)\bin\s+this\s+(?:project|repo|codebase)\b`, "project-norm", confidenceTierB},

		// Permission Boundaries (issue #24)
		{`(?i)\bleave\s+\w+\s+alone\b`, "hands-off", confidenceTierB},
		{`(?i)\bhands\s+off\b`, "hands-off-explicit", confidenceTierB},
		{`(?i)\boff\s+limits\b`, "off-limits", confidenceTierB},

		// Confusion / Misunderstanding (issue #24)
		{`(?i)\byou\s+misunderstood\b`, "misunderstood", confidenceTierB},
		{`(?i)\bno,?\s+I\s+mean\b`, "clarification", confidenceTierB},
		{`(?i)\bmisinterpreted\b`, "misinterpreted", confidenceTierB},
	}

	patterns := make([]Pattern, 0, len(raw))

	for _, entry := range raw {
		patterns = append(patterns, Pattern{
			Regex:      regexp.MustCompile(entry.regex),
			Label:      entry.label,
			Confidence: entry.confidence,
		})
	}

	return patterns
}

// unexported constants.
const (
	confidenceTierA = "A"
	confidenceTierB = "B"
)
