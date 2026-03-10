// Package registry defines the unified instruction registry for engram (UC-23).
package registry

import "time"

// Exported constants.
const (
	SourceTypeMemory = "memory"
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
	ID            string             `json:"id"`
	SourceType    string             `json:"source_type"`
	SourcePath    string             `json:"source_path"`
	Title         string             `json:"title"`
	ContentHash   string             `json:"content_hash"`
	RegisteredAt  time.Time          `json:"registered_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
	SurfacedCount int                `json:"surfaced_count"`
	LastSurfaced  *time.Time         `json:"last_surfaced,omitempty"`
	Evaluations   EvaluationCounters `json:"evaluations"`
	Absorbed      []AbsorbedRecord   `json:"absorbed,omitempty"`
}
