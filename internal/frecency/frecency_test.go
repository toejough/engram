package frecency_test

import (
	"math"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/frecency"
)

func TestAlpha_DefaultIsZero(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	scorer := frecency.New(now, 0)

	g.Expect(scorer.Alpha()).To(BeNumerically("==", 0))
}

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
	quality := scorer.Quality(input)

	combined := scorer.CombinedScore(relevance, 0.5, 1.0, input)

	// (relevance*genFactor + alpha*spreading) * (1 + quality), genFactor=1.0
	// alpha defaults to 0 — spreading is disabled
	expected := (2.0*1.0 + 0*0.5) * (1.0 + quality)
	g.Expect(combined).To(BeNumerically("~", expected, 0.0001))
}

func TestCombinedScore_GenFactorDoesNotAffectSpreading(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	scorer := frecency.New(now, 0)
	input := frecency.Input{}

	a := scorer.CombinedScore(0.0, 1.0, 0.05, input)
	b := scorer.CombinedScore(0.0, 1.0, 1.0, input)

	g.Expect(a).To(Equal(b))
}

func TestCombinedScore_GenFactorReducesRelevance(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	scorer := frecency.New(now, 0)
	input := frecency.Input{}

	full := scorer.CombinedScore(1.0, 0.0, 1.0, input)
	halved := scorer.CombinedScore(1.0, 0.0, 0.5, input)

	g.Expect(halved).To(BeNumerically("<", full))
	g.Expect(halved).To(BeNumerically("~", full*0.5, 0.01))
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
	combined := scorer.CombinedScore(0.0, 0.8, 1.0, input)

	// alpha=0 — spreading is disabled, so spreading-only score is 0
	// (0*genFactor + 0*0.8) * (1 + quality) = 0
	expected := (0.0 + 0*0.8) * (1.0 + quality)
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

	combined := scorer.CombinedScore(0.0, 0.0, 1.0, input)
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

			// Use zero times so recency=0, tier="" so tierBoost=0, no evaluations so eff=0.5
			// quality = 0.3*0.5 + 0*0 + 1.0*freq + 0.3*0 = 0.15 + freq
			input := frecency.Input{
				SurfacedCount: testCase.surfacedCount,
				FilePath:      "mem/freq.toml",
			}

			quality := scorer.Quality(input)
			freq := quality - 0.15 // subtract eff component (wEff=0.3, defaultEff=0.5 → 0.15)

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
		Tier:           "A",
		FilePath:       "mem/alpha.toml",
	}

	quality := scorer.Quality(input)

	// eff = 4/5 = 0.8
	// recency = 1 / (1 + 3.5/7) = 1/1.5 = 0.6667 (wRec=0 so contributes 0)
	// freq = ln(101) / ln(1001)
	// tierBoost = 1.2 (tier A)
	expectedEff := 0.8
	expectedFreq := math.Log(101) / math.Log(1001)
	expectedTierBoost := 1.2
	expected := 0.3*expectedEff + 0*0 + 1.0*expectedFreq + 0.3*expectedTierBoost

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

	// eff defaults to 0.5 when no evaluations, wRec=0 so recency contributes 0, no tier
	expectedEff := 0.5
	expectedFreq := math.Log(11) / math.Log(101)
	expected := 0.3*expectedEff + 0*0 + 1.0*expectedFreq + 0.3*0

	g.Expect(quality).To(BeNumerically("~", expected, 0.0001))
}

func TestQuality_TierBoost_A(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	// maxSurfaced=0, no evaluations → eff=0.5, freq=0, wRec=0
	// quality = 0.3*0.5 + 0 + 0 + 0.3*tierBoost
	scorer := frecency.New(now, 0)

	input := frecency.Input{
		Tier:     "A",
		FilePath: "mem/tier-a.toml",
	}

	quality := scorer.Quality(input)

	// 0.3*0.5 + 0.3*1.2 = 0.15 + 0.36 = 0.51
	expected := 0.3*0.5 + 0.3*1.2
	g.Expect(quality).To(BeNumerically("~", expected, 0.0001))
}

func TestQuality_TierBoost_B(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	scorer := frecency.New(now, 0)

	input := frecency.Input{
		Tier:     "B",
		FilePath: "mem/tier-b.toml",
	}

	quality := scorer.Quality(input)

	// 0.3*0.5 + 0.3*0.2 = 0.15 + 0.06 = 0.21
	expected := 0.3*0.5 + 0.3*0.2
	g.Expect(quality).To(BeNumerically("~", expected, 0.0001))
}

func TestQuality_TierBoost_C(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	scorer := frecency.New(now, 0)

	input := frecency.Input{
		Tier:     "C",
		FilePath: "mem/tier-c.toml",
	}

	quality := scorer.Quality(input)

	// tier C contributes 0 boost: 0.3*0.5 + 0.3*0 = 0.15
	expected := 0.3 * 0.5
	g.Expect(quality).To(BeNumerically("~", expected, 0.0001))
}

func TestQuality_TierBoost_Empty(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	scorer := frecency.New(now, 0)

	input := frecency.Input{
		Tier:     "",
		FilePath: "mem/tier-empty.toml",
	}

	quality := scorer.Quality(input)

	// empty tier contributes 0 boost: 0.3*0.5 + 0.3*0 = 0.15
	expected := 0.3 * 0.5
	g.Expect(quality).To(BeNumerically("~", expected, 0.0001))
}

func TestRecency_Disabled(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	// maxSurfaced=0 and no evaluations → eff=0.5, freq=0, tier=""
	// wRec=0 so recency has no effect — quality is identical regardless of daysAgo
	scorer := frecency.New(now, 0)

	inputRecent := frecency.Input{
		LastSurfacedAt: now, // 0 days ago
		FilePath:       "mem/recency.toml",
	}

	inputStale := frecency.Input{
		LastSurfacedAt: now.Add(-7 * 24 * time.Hour), // 7 days ago
		FilePath:       "mem/recency.toml",
	}

	qualityRecent := scorer.Quality(inputRecent)
	qualityStale := scorer.Quality(inputStale)

	g.Expect(qualityRecent).To(BeNumerically("~", qualityStale, 0.0001),
		"recency signal is disabled (wRec=0): quality must not vary with staleness")
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

func TestWithAlpha_SetsCustomValue(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	scorer := frecency.New(now, 0, frecency.WithAlpha(1.5))

	g.Expect(scorer.Alpha()).To(BeNumerically("~", 1.5, 0.0001))
}
