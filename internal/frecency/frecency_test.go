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
	quality := scorer.Quality(input)

	combined := scorer.CombinedScore(relevance, 1.0, input)

	// relevance * genFactor * (1 + quality), genFactor=1.0
	expected := 2.0 * 1.0 * (1.0 + quality)
	g.Expect(combined).To(BeNumerically("~", expected, 0.0001))
}

func TestCombinedScore_GenFactorReducesRelevance(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	scorer := frecency.New(now, 0)
	input := frecency.Input{}

	full := scorer.CombinedScore(1.0, 1.0, input)
	halved := scorer.CombinedScore(1.0, 0.5, input)

	g.Expect(halved).To(BeNumerically("<", full))
	g.Expect(halved).To(BeNumerically("~", full*0.5, 0.01))
}

func TestCombinedScore_ZeroRelevance(t *testing.T) {
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

	combined := scorer.CombinedScore(0.0, 1.0, input)
	g.Expect(combined).To(BeNumerically("==", 0.0))
}

func TestEffectiveness_IncludesIrrelevantCount(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	// maxSurfaced=0 so freq=0, no tier so tierBoost=0
	// eff = 5/(5+5) = 0.5, quality = 0.3*0.5 + 0 + 0 = 0.15
	scorer := frecency.New(now, 0)

	input := frecency.Input{
		FollowedCount:   5,
		IrrelevantCount: 5,
		FilePath:        "mem/irrelevant-test.toml",
	}

	quality := scorer.Quality(input)

	expected := 0.3 * 0.5 // wEff * (5/(5+5))
	g.Expect(quality).To(BeNumerically("~", expected, 0.0001))
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

func TestNew_WithTierBoostOverrides(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	now := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)

	const maxSurfaced = 100

	// WithTierABoost, WithTierBBoost, WithWTier all produce a scorer that applies overrides.
	scorer := frecency.New(now, maxSurfaced,
		frecency.WithTierABoost(2.0),
		frecency.WithTierBBoost(1.5),
		frecency.WithWTier(0.5),
	)

	inputTierA := frecency.Input{
		FollowedCount: 5,
		IgnoredCount:  5,
		SurfacedCount: 10,
		Tier:          "A",
	}

	inputTierB := frecency.Input{
		FollowedCount: 5,
		IgnoredCount:  5,
		SurfacedCount: 10,
		Tier:          "B",
	}

	qualityA := scorer.Quality(inputTierA)
	qualityB := scorer.Quality(inputTierB)

	// Tier A boost (2.0) > Tier B boost (1.5), so tier A quality should be higher.
	g.Expect(qualityA).To(BeNumerically(">", qualityB))
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
	// freq = ln(101) / ln(1001)
	// tierBoost = 1.2 (tier A)
	expectedEff := 0.8
	expectedFreq := math.Log(101) / math.Log(1001)
	expectedTierBoost := 1.2
	expected := 0.3*expectedEff + 1.0*expectedFreq + 0.3*expectedTierBoost

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

	// eff defaults to 0.5 when no evaluations, no tier boost
	expectedEff := 0.5
	expectedFreq := math.Log(11) / math.Log(101)
	expected := 0.3*expectedEff + 1.0*expectedFreq

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

func TestQuality_WithWeightOverrides(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	now := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)

	const maxSurfaced = 100

	scorer := frecency.New(now, maxSurfaced,
		frecency.WithWEff(0.5),
		frecency.WithWFreq(0.7),
	)

	input := frecency.Input{
		FollowedCount: 8,
		IgnoredCount:  2,
		SurfacedCount: 50,
	}

	quality := scorer.Quality(input)
	// effectiveness = 8/10 = 0.8
	// frequency = log(51)/log(101) ≈ 0.851
	// tierBoost = 0
	// quality = 0.5*0.8 + 0.7*0.851 + 0.3*0 = 0.4 + 0.596 = 0.996
	g.Expect(quality).To(BeNumerically("~", 0.996, 0.02))
}
