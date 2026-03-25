// Package frecency implements quality-weighted scoring for memory ranking (ARCH-35).
package frecency

import (
	"math"
	"time"
)

// Input holds the data needed to score one memory.
type Input struct {
	SurfacedCount     int
	LastSurfacedAt    time.Time
	UpdatedAt         time.Time
	FollowedCount     int
	ContradictedCount int
	IgnoredCount      int
	FilePath          string
}

// Option configures a Scorer.
type Option func(*Scorer)

// Scorer computes quality-weighted scores for memories.
type Scorer struct {
	now          time.Time
	maxSurfaced  int
	halfLifeDays float64
	wEff         float64
	wRec         float64
	wFreq        float64
	alpha        float64
}

// New creates a Scorer. maxSurfaced is the corpus-wide max surfaced count.
func New(now time.Time, maxSurfaced int, opts ...Option) *Scorer {
	s := &Scorer{
		now:          now,
		maxSurfaced:  maxSurfaced,
		halfLifeDays: defaultHalfLifeDays,
		wEff:         defaultWEff,
		wRec:         defaultWRec,
		wFreq:        defaultWFreq,
		alpha:        defaultAlpha,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Alpha returns the spreading activation weight.
func (s *Scorer) Alpha() float64 {
	return s.alpha
}

// CombinedScore computes (relevance*genFactor + alpha*spreading) * (1 + quality).
func (s *Scorer) CombinedScore(relevance, spreading, genFactor float64, input Input) float64 {
	return (relevance*genFactor + s.alpha*spreading) * (1.0 + s.Quality(input))
}

// Quality computes the quality multiplier for a memory.
func (s *Scorer) Quality(input Input) float64 {
	return s.wEff*s.effectiveness(input) +
		s.wRec*s.recency(input) +
		s.wFreq*s.frequency(input)
}

func (s *Scorer) effectiveness(input Input) float64 {
	total := input.FollowedCount + input.ContradictedCount + input.IgnoredCount
	if total == 0 {
		return defaultEffectiveness
	}

	return float64(input.FollowedCount) / float64(total)
}

func (s *Scorer) frequency(input Input) float64 {
	if s.maxSurfaced <= 0 {
		return 0
	}

	return math.Log(1+float64(input.SurfacedCount)) /
		math.Log(1+float64(s.maxSurfaced))
}

func (s *Scorer) recency(input Input) float64 {
	ref := input.LastSurfacedAt
	if ref.IsZero() {
		ref = input.UpdatedAt
	}

	if ref.IsZero() {
		return 0
	}

	daysSince := s.now.Sub(ref).Hours() / hoursPerDay
	if daysSince < 0 {
		daysSince = 0
	}

	return 1.0 / (1.0 + daysSince/s.halfLifeDays)
}

// unexported constants.
const (
	defaultAlpha         = 1.0
	defaultEffectiveness = 0.5
	defaultHalfLifeDays  = 7.0
	defaultWEff          = 1.5
	defaultWFreq         = 1.0
	defaultWRec          = 0.5
	hoursPerDay          = 24.0
)
