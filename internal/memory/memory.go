// Package memory defines shared types for the engram memory pipeline.
package memory

import "time"

// CandidateLearning holds a learning extracted from a session transcript (ARCH-15).
// The Learner pipeline sets Confidence and timestamps before writing.
type CandidateLearning struct {
	Title           string
	Content         string
	ObservationType string
	Concepts        []string
	Keywords        []string
	Principle       string
	AntiPattern     string
	Rationale       string
	FilenameSummary string
}

// Enriched holds all structured fields of an enriched memory.
type Enriched struct {
	Title           string
	Content         string
	ObservationType string
	Concepts        []string
	Keywords        []string
	Principle       string
	AntiPattern     string
	Rationale       string
	FilenameSummary string
	Confidence      string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// PatternMatch holds the result of pattern matching against a user message.
type PatternMatch struct {
	Pattern    string
	Label      string
	Confidence string // "A" for remember patterns, "B" for correction patterns
}

// Stored represents a memory read back from a TOML file on disk (ARCH-9).
type Stored struct {
	Title       string
	Content     string
	Concepts    []string
	Keywords    []string
	AntiPattern string
	Principle   string
	UpdatedAt   time.Time
	FilePath    string
}
