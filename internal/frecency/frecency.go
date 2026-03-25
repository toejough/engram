// Package frecency implements ACT-R activation scoring for memory ranking (ARCH-35).
package frecency

import (
	"math"
	"time"
)

// EffectivenessStat holds per-memory effectiveness data.
type EffectivenessStat struct {
	EffectivenessScore float64 // 0-100 range
}

// Input holds the data needed to compute activation for one memory.
type Input struct {
	SurfacedCount int
	LastSurfaced  time.Time
	UpdatedAt     time.Time // fallback for never-surfaced
	FilePath      string    // key for effectiveness lookup
}

// Scorer computes frecency activation scores for memories.
type Scorer struct {
	now           time.Time
	effectiveness map[string]EffectivenessStat // keyed by memory file path
}

// New creates a Scorer with the given time and effectiveness data.
func New(now time.Time, effectiveness map[string]EffectivenessStat) *Scorer {
	return &Scorer{now: now, effectiveness: effectiveness}
}

// Activation computes the frecency activation score for a memory.
// Formula: frequency x recency x effectiveness.
func (s *Scorer) Activation(input Input) float64 {
	freq := math.Log(1 + float64(input.SurfacedCount))

	// Recency: use LastSurfaced if set, else UpdatedAt.
	refTime := input.LastSurfaced
	if refTime.IsZero() {
		refTime = input.UpdatedAt
	}

	hoursSince := s.now.Sub(refTime).Hours()
	if hoursSince < 0 {
		hoursSince = 0
	}

	recency := 1.0 / (1.0 + hoursSince)

	// Effectiveness: default 0.5 when no data, floor 0.1.
	eff := defaultEffectiveness

	if s.effectiveness != nil {
		if stat, ok := s.effectiveness[input.FilePath]; ok {
			eff = math.Max(effectivenessFloor, stat.EffectivenessScore/effectivenessMaxScore)
		}
	}

	return freq * recency * eff
}

// CombinedScore computes BM25 x (1 + activation) for prompt/tool modes.
// BM25 of zero stays zero regardless of frecency.
func (s *Scorer) CombinedScore(bm25Score float64, input Input) float64 {
	return bm25Score * (1.0 + s.Activation(input))
}

// unexported constants.
const (
	defaultEffectiveness  = 0.5
	effectivenessFloor    = 0.1
	effectivenessMaxScore = 100.0
)
