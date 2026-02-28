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
	for _, p := range c.patterns {
		if loc := p.Regex.FindStringIndex(message); loc != nil {
			return &Match{Pattern: p, Text: message[loc[0]:loc[1]]}
		}
	}

	return nil
}

// Match holds a matched pattern and the matched text.
type Match struct {
	Pattern Pattern
	Text    string
}

// Pattern pairs a compiled regex with a human-readable label.
type Pattern struct {
	Regex *regexp.Regexp
	Label string
}

// DefaultPatterns returns built-in correction patterns for detecting user feedback signals.
func DefaultPatterns() []Pattern {
	raw := []struct {
		regex string
		label string
	}{
		{`(?i)^no,`, "direct-negation"},
		{`(?i)^wait`, "interruption"},
		{`(?i)^hold on`, "pause"},
		{`(?i)\bwrong\b`, "wrong"},
		{`(?i)\bdon't\s+\w+`, "dont"},
		{`(?i)\bstop\s+\w+ing`, "stop"},
		{`(?i)\btry again`, "retry"},
		{`(?i)\bgo back`, "revert"},
		{`(?i)\bthat's not`, "negation"},
		{`(?i)^actually,`, "correction"},
		{`(?i)\bremember\s+(that|to)`, "reminder"},
		{`(?i)\bstart over`, "restart"},
		{`(?i)\bpre-?existing`, "preexisting"},
		{`(?i)\byou're still`, "persistence"},
		{`(?i)\bincorrect`, "incorrect"},
	}
	patterns := make([]Pattern, 0, len(raw))

	for _, r := range raw {
		patterns = append(patterns, Pattern{
			Regex: regexp.MustCompile(r.regex),
			Label: r.label,
		})
	}

	return patterns
}
