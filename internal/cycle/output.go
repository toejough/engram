// Package cycle implements the engram cycle command — extracts learnings and
// proposes recall queries from a session transcript via two LLM calls.
package cycle

import "engram/internal/memory"

// LearnedMemory is a memory record that was actually persisted by this cycle.
// MemoryRecord is embedded so its fields appear flattened in the JSON output;
// Name is added at the top level.
type LearnedMemory struct {
	memory.MemoryRecord

	Name string `json:"name"`
}

// Output is the JSON shape returned by `engram cycle`.
type Output struct {
	Learned  []LearnedMemory  `json:"learned"`
	Recalled []RecalledReport `json:"recalled"`
}

// NewOutput returns an Output with non-nil empty slices so JSON serializes
// "learned":[] / "recalled":[] rather than null.
func NewOutput() *Output {
	return &Output{
		Learned:  []LearnedMemory{},
		Recalled: []RecalledReport{},
	}
}

// RecalledReport is a query and the synthesized prose report it produced.
type RecalledReport struct {
	Query  string `json:"query"`
	Report string `json:"report"`
}
