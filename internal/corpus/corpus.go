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
func DefaultPatterns() []Pattern {
	raw := []struct {
		regex      string
		label      string
		confidence string
	}{
		{`(?i)^no,`, "direct-negation", confidenceTierB},
		{`(?i)^wait`, "interruption", confidenceTierB},
		{`(?i)^hold on`, "pause", confidenceTierB},
		{`(?i)\bwrong\b`, "wrong", confidenceTierB},
		{`(?i)\bdon't\s+\w+`, "dont", confidenceTierB},
		{`(?i)\bstop\s+\w+ing`, "stop", confidenceTierB},
		{`(?i)\btry again`, "retry", confidenceTierB},
		{`(?i)\bgo back`, "revert", confidenceTierB},
		{`(?i)\bthat's not`, "negation", confidenceTierB},
		{`(?i)^actually,`, "correction", confidenceTierB},
		{`(?i)\bremember\s+(that|to)`, "reminder", confidenceTierA},
		{`(?i)\bstart over`, "restart", confidenceTierB},
		{`(?i)\bpre-?existing`, "preexisting", confidenceTierB},
		{`(?i)\byou're still`, "persistence", confidenceTierB},
		{`(?i)\bincorrect`, "incorrect", confidenceTierB},
	}

	patterns := make([]Pattern, 0, len(raw))

	for _, rawPattern := range raw {
		patterns = append(patterns, Pattern{
			Regex:      regexp.MustCompile(rawPattern.regex),
			Label:      rawPattern.label,
			Confidence: rawPattern.confidence,
		})
	}

	return patterns
}

// unexported constants.
const (
	confidenceTierA = "A"
	confidenceTierB = "B"
)
