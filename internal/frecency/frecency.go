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
	Tier              string // "A", "B", "C", or "" — memory confidence tier
}

// Option configures a Scorer.
type Option func(*Scorer)

// Scorer computes quality-weighted scores for memories.
type Scorer struct {
	maxSurfaced int
	wEff        float64
	wFreq       float64
	wTier       float64
	tierABoost  float64
	tierBBoost  float64
	alpha       float64
}

// New creates a Scorer. maxSurfaced is the corpus-wide max surfaced count.
func New(_ time.Time, maxSurfaced int, opts ...Option) *Scorer {
	s := &Scorer{
		maxSurfaced: maxSurfaced,
		wEff:        defaultWEff,
		wFreq:       defaultWFreq,
		wTier:       defaultWTier,
		tierABoost:  defaultTierABoost,
		tierBBoost:  defaultTierBBoost,
		alpha:       defaultAlpha,
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
		s.wFreq*s.frequency(input) +
		s.wTier*s.tierBoost(input)
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

func (s *Scorer) tierBoost(input Input) float64 {
	switch input.Tier {
	case "A":
		return s.tierABoost
	case "B":
		return s.tierBBoost
	default:
		return 0
	}
}

// WithAlpha sets the spreading activation weight (alpha).
func WithAlpha(alpha float64) Option {
	return func(s *Scorer) { s.alpha = alpha }
}

// unexported constants.
const (
	defaultAlpha         = 0
	defaultEffectiveness = 0.5
	defaultTierABoost    = 1.2
	defaultTierBBoost    = 0.2
	defaultWEff          = 0.3
	defaultWFreq         = 1.0
	defaultWTier         = 0.3
)
