// Package registry defines the unified instruction registry for engram (UC-23).
package registry

import "time"

// Exported constants.
const (
	EnforcementAdvisory           EnforcementLevel = "advisory"
	EnforcementEmphasizedAdvisory EnforcementLevel = "emphasized_advisory"
	EnforcementGraduated          EnforcementLevel = "graduated"
	EnforcementReminder           EnforcementLevel = "reminder"
	SourceTypeMemory                               = "memory"
)

// AbsorbedRecord preserves counters from a merged source instruction.
//
//nolint:tagliatelle // spec requires snake_case JSON field names.
type AbsorbedRecord struct {
	From          string             `json:"from"`
	SurfacedCount int                `json:"surfaced_count"`
	Evaluations   EvaluationCounters `json:"evaluations"`
	ContentHash   string             `json:"content_hash"`
	MergedAt      time.Time          `json:"merged_at"`
}

// EnforcementLevel represents the delivery salience of an instruction.
// Levels are ordinal: advisory < emphasized_advisory < reminder < graduated.
type EnforcementLevel string

// EnforcementTransition records a change in enforcement level for an instruction.
type EnforcementTransition struct {
	From   EnforcementLevel `json:"from"`
	To     EnforcementLevel `json:"to"`
	At     time.Time        `json:"at"`
	Reason string           `json:"reason"`
}

// EvaluationCounters holds follow/contradict/ignore tallies.
type EvaluationCounters struct {
	Followed     int `json:"followed"`
	Contradicted int `json:"contradicted"`
	Ignored      int `json:"ignored"`
}

// Total returns the sum of all evaluation outcomes.
func (e EvaluationCounters) Total() int {
	return e.Followed + e.Contradicted + e.Ignored
}

// InstructionEntry represents one registered instruction.
//
//nolint:tagliatelle // spec requires snake_case JSON field names.
type InstructionEntry struct {
	ID               string                  `json:"id"`
	SourceType       string                  `json:"source_type"`
	SourcePath       string                  `json:"source_path"`
	Title            string                  `json:"title"`
	Content          string                  `json:"content,omitempty"`
	ContentHash      string                  `json:"content_hash"`
	RegisteredAt     time.Time               `json:"registered_at"`
	UpdatedAt        time.Time               `json:"updated_at"`
	SurfacedCount    int                     `json:"surfaced_count"`
	LastSurfaced     *time.Time              `json:"last_surfaced,omitempty"`
	Evaluations      EvaluationCounters      `json:"evaluations"`
	Absorbed         []AbsorbedRecord        `json:"absorbed,omitempty"`
	EnforcementLevel EnforcementLevel        `json:"enforcement_level,omitempty"`
	Transitions      []EnforcementTransition `json:"transitions,omitempty"`
	Links            []Link                  `json:"links,omitempty"`
}

// Link represents a typed, weighted relationship between two instruction entries.
//
//nolint:tagliatelle // spec requires snake_case JSON field names.
type Link struct {
	Target           string  `json:"target"`
	Weight           float64 `json:"weight"`
	Basis            string  `json:"basis"`
	CoSurfacingCount int     `json:"co_surfacing_count,omitempty"`
}
