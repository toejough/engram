package frecency_test

import (
	"math"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/frecency"
)

func TestCombinedScore_Basic(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	scorer := frecency.New(now, 100)

	input := frecency.Input{
		SurfacedCount:  10,
		LastSurfacedAt: now.Add(-24 * time.Hour),
		FollowedCount:  3,
		IgnoredCount:   1,
		FilePath:       "mem/gamma.toml",
	}

	relevance := 2.0
	spreading := 0.5
	quality := scorer.Quality(input)

	combined := scorer.CombinedScore(relevance, spreading, input)

	// (relevance + alpha*spreading) * (1 + quality)
	// alpha defaults to 1.0
	expected := (2.0 + 1.0*0.5) * (1.0 + quality)
	g.Expect(combined).To(BeNumerically("~", expected, 0.0001))
}

func TestCombinedScore_SpreadingOnly(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	scorer := frecency.New(now, 100)

	input := frecency.Input{
		SurfacedCount:  5,
		LastSurfacedAt: now.Add(-48 * time.Hour),
		FollowedCount:  2,
		IgnoredCount:   1,
		FilePath:       "mem/epsilon.toml",
	}

	quality := scorer.Quality(input)
	combined := scorer.CombinedScore(0.0, 0.8, input)

	// (0 + 1.0*0.8) * (1 + quality)
	expected := (0.0 + 1.0*0.8) * (1.0 + quality)
	g.Expect(combined).To(BeNumerically("~", expected, 0.0001))
}

func TestCombinedScore_ZeroRelevanceZeroSpreading(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	scorer := frecency.New(now, 100)

	input := frecency.Input{
		SurfacedCount:  10,
		LastSurfacedAt: now.Add(-24 * time.Hour),
		FollowedCount:  5,
		FilePath:       "mem/delta.toml",
	}

	combined := scorer.CombinedScore(0.0, 0.0, input)
	g.Expect(combined).To(BeNumerically("==", 0.0))
}

func TestFrequency_Normalized(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)

	const maxSurfaced = 50

	scorer := frecency.New(now, maxSurfaced)

	tests := []struct {
		name          string
		surfacedCount int
		expected      float64
	}{
		{
			name:          "max_surfaced_equals_one",
			surfacedCount: maxSurfaced,
			expected:      1.0,
		},
		{
			name:          "zero_surfaced_equals_zero",
			surfacedCount: 0,
			expected:      0.0,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			g := NewGomegaWithT(t)

			// Use zero times so recency=0, no evaluations so eff=0.5
			// quality = 1.5*0.5 + 0.5*0 + 1.0*freq = 0.75 + freq
			input := frecency.Input{
				SurfacedCount: testCase.surfacedCount,
				FilePath:      "mem/freq.toml",
			}

			quality := scorer.Quality(input)
			freq := quality - 0.75 // subtract eff component

			g.Expect(freq).To(
				BeNumerically("~", testCase.expected, 0.0001),
			)
		})
	}
}

func TestQuality_AllSignals(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)

	const maxSurfaced = 1000

	scorer := frecency.New(now, maxSurfaced)

	input := frecency.Input{
		SurfacedCount:  100,
		LastSurfacedAt: now.Add(-84 * time.Hour), // 3.5 days ago
		UpdatedAt:      now.Add(-168 * time.Hour),
		FollowedCount:  4,
		IgnoredCount:   1,
		FilePath:       "mem/alpha.toml",
	}

	quality := scorer.Quality(input)

	// eff = 4/5 = 0.8
	// recency = 1 / (1 + 3.5/7) = 1/1.5 = 0.6667
	// freq = ln(101) / ln(1001)
	expectedEff := 0.8
	expectedRecency := 1.0 / (1.0 + 3.5/7.0)
	expectedFreq := math.Log(101) / math.Log(1001)
	expected := 1.5*expectedEff + 0.5*expectedRecency + 1.0*expectedFreq

	g.Expect(quality).To(BeNumerically("~", expected, 0.0001))
}

func TestQuality_NoEvaluations(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	scorer := frecency.New(now, 100)

	input := frecency.Input{
		SurfacedCount:  10,
		LastSurfacedAt: now.Add(-24 * time.Hour),
		FilePath:       "mem/beta.toml",
	}

	quality := scorer.Quality(input)

	// eff defaults to 0.5 when no evaluations
	expectedEff := 0.5
	expectedRecency := 1.0 / (1.0 + 1.0/7.0)
	expectedFreq := math.Log(11) / math.Log(101)
	expected := 1.5*expectedEff + 0.5*expectedRecency + 1.0*expectedFreq

	g.Expect(quality).To(BeNumerically("~", expected, 0.0001))
}

func TestRecency_FallbackToUpdatedAt(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	scorer := frecency.New(now, 0)

	// LastSurfacedAt is zero → should use UpdatedAt (7 days ago → recency 0.5)
	inputFallback := frecency.Input{
		UpdatedAt: now.Add(-168 * time.Hour), // 7 days
		FilePath:  "mem/fallback.toml",
	}

	// Explicit LastSurfacedAt at the same time
	inputExplicit := frecency.Input{
		LastSurfacedAt: now.Add(-168 * time.Hour),
		FilePath:       "mem/explicit.toml",
	}

	qualityFallback := scorer.Quality(inputFallback)
	qualityExplicit := scorer.Quality(inputExplicit)

	g.Expect(qualityFallback).To(
		BeNumerically("~", qualityExplicit, 0.0001),
	)
}

func TestRecency_HalfLife(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	// maxSurfaced=0 and no evaluations → eff=0.5, freq=0
	// quality = 1.5*0.5 + 0.5*recency + 1.0*0
	// We can extract recency by: (quality - 0.75) / 0.5
	scorer := frecency.New(now, 0)

	tests := []struct {
		name     string
		daysAgo  float64
		expected float64
	}{
		{name: "zero_days", daysAgo: 0, expected: 1.0},
		{name: "seven_days", daysAgo: 7, expected: 0.5},
		{name: "fourteen_days", daysAgo: 14, expected: 1.0 / 3.0},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			g := NewGomegaWithT(t)

			input := frecency.Input{
				LastSurfacedAt: now.Add(
					-time.Duration(testCase.daysAgo*24) * time.Hour,
				),
				FilePath: "mem/recency.toml",
			}

			quality := scorer.Quality(input)
			// quality = 1.5*0.5 + 0.5*recency + 1.0*0 = 0.75 + 0.5*recency
			recency := (quality - 0.75) / 0.5

			g.Expect(recency).To(
				BeNumerically("~", testCase.expected, 0.0001),
			)
		})
	}
}
