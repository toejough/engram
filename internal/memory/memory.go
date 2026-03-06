// Package memory defines shared types for the engram memory pipeline.
package memory

import "time"

// CandidateLearning holds a learning extracted from a session transcript (ARCH-15).
// The Learner pipeline uses Tier as Confidence when writing.
type CandidateLearning struct {
	Tier            string // "A", "B", or "C" — classified by the LLM
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

// ClassifiedMemory holds the output of the unified classifier (ARCH-2).
// Combines classification (tier) and enrichment (structured fields) in one step.
type ClassifiedMemory struct {
	Tier            string // "A", "B", or "C"
	Title           string
	Content         string
	ObservationType string
	Concepts        []string
	Keywords        []string
	Principle       string
	AntiPattern     string // tier-gated: required for A, optional for B, empty for C
	Rationale       string
	FilenameSummary string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// ToEnriched converts a ClassifiedMemory to an Enriched for TOML writing compatibility.
func (cm *ClassifiedMemory) ToEnriched() *Enriched {
	return &Enriched{
		Title:           cm.Title,
		Content:         cm.Content,
		ObservationType: cm.ObservationType,
		Concepts:        cm.Concepts,
		Keywords:        cm.Keywords,
		Principle:       cm.Principle,
		AntiPattern:     cm.AntiPattern,
		Rationale:       cm.Rationale,
		FilenameSummary: cm.FilenameSummary,
		Confidence:      cm.Tier,
		CreatedAt:       cm.CreatedAt,
		UpdatedAt:       cm.UpdatedAt,
	}
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
	Title             string
	Content           string
	Concepts          []string
	Keywords          []string
	AntiPattern       string
	Principle         string
	UpdatedAt         time.Time
	FilePath          string
	SurfacedCount     int
	LastSurfaced      time.Time
	SurfacingContexts []string
}
