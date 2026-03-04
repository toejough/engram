// Package memory defines shared types for the engram memory pipeline.
package memory

import "time"

// PatternMatch holds the result of pattern matching against a user message.
type PatternMatch struct {
	Pattern    string
	Label      string
	Confidence string // "A" for remember patterns, "B" for correction patterns
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
	Degraded        bool // true when LLM enrichment was unavailable
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
