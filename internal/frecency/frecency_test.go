package frecency_test

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/frecency"
)

// T-165: All components present — verify activation > 0.
func TestT165_AllComponentsPresentActivationPositive(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	effectiveness := map[string]frecency.EffectivenessStat{
		"mem/alpha.toml": {EffectivenessScore: 80.0},
	}

	scorer := frecency.New(now, effectiveness)

	input := frecency.Input{
		SurfacedCount:     10,
		LastSurfaced:      now.Add(-2 * time.Hour),
		UpdatedAt:         now.Add(-24 * time.Hour),
		SurfacingContexts: []string{"session-start", "prompt", "tool"},
		FilePath:          "mem/alpha.toml",
	}

	activation := scorer.Activation(input)
	g.Expect(activation).To(BeNumerically(">", 0.0))
}

// T-166: Never-surfaced memory — frequency=0 dominates, activation = 0.
func TestT166_NeverSurfacedActivationZero(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)

	scorer := frecency.New(now, nil)

	input := frecency.Input{
		SurfacedCount:     0,
		LastSurfaced:      time.Time{}, // zero value
		UpdatedAt:         now.Add(-24 * time.Hour),
		SurfacingContexts: nil,
		FilePath:          "mem/beta.toml",
	}

	activation := scorer.Activation(input)
	g.Expect(activation).To(BeNumerically("==", 0.0))
}

// T-167: No effectiveness data — uses default 0.5.
func TestT167_NoEffectivenessDataUsesDefault(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	// Effectiveness map exists but has no entry for this file path.
	effectiveness := map[string]frecency.EffectivenessStat{
		"mem/other.toml": {EffectivenessScore: 90.0},
	}

	scorer := frecency.New(now, effectiveness)

	input := frecency.Input{
		SurfacedCount:     5,
		LastSurfaced:      now.Add(-1 * time.Hour),
		UpdatedAt:         now.Add(-48 * time.Hour),
		SurfacingContexts: []string{"prompt"},
		FilePath:          "mem/gamma.toml",
	}

	// Compute activation with default effectiveness (0.5).
	activationDefault := scorer.Activation(input)

	// Now compute with explicit 50% effectiveness — should be the same.
	effectivenessExplicit := map[string]frecency.EffectivenessStat{
		"mem/gamma.toml": {EffectivenessScore: 50.0},
	}
	scorerExplicit := frecency.New(now, effectivenessExplicit)
	activationExplicit := scorerExplicit.Activation(input)

	g.Expect(activationDefault).To(BeNumerically("~", activationExplicit, 0.0001))
}

// T-168: CombinedScore with BM25=0 — verify result is 0.0.
func TestT168_CombinedScoreBM25ZeroReturnsZero(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	effectiveness := map[string]frecency.EffectivenessStat{
		"mem/alpha.toml": {EffectivenessScore: 80.0},
	}

	scorer := frecency.New(now, effectiveness)

	input := frecency.Input{
		SurfacedCount:     10,
		LastSurfaced:      now.Add(-1 * time.Hour),
		UpdatedAt:         now.Add(-24 * time.Hour),
		SurfacingContexts: []string{"prompt", "tool"},
		FilePath:          "mem/alpha.toml",
	}

	combined := scorer.CombinedScore(0.0, input)
	g.Expect(combined).To(BeNumerically("==", 0.0))
}

// T-170: CombinedScore re-ranking — verify ordering changes after combined scoring.
func TestT170_CombinedScoreReRanking(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	effectiveness := map[string]frecency.EffectivenessStat{
		"mem/low-bm25-high-frecency.toml": {EffectivenessScore: 95.0},
		"mem/high-bm25-low-frecency.toml": {EffectivenessScore: 10.0},
		"mem/mid-bm25-mid-frecency.toml":  {EffectivenessScore: 50.0},
	}

	scorer := frecency.New(now, effectiveness)

	// Memory A: low BM25 but high frecency (recently surfaced, many times, many contexts).
	inputA := frecency.Input{
		SurfacedCount:     20,
		LastSurfaced:      now.Add(-30 * time.Minute),
		UpdatedAt:         now.Add(-1 * time.Hour),
		SurfacingContexts: []string{"session-start", "prompt", "tool", "extra"},
		FilePath:          "mem/low-bm25-high-frecency.toml",
	}

	// Memory B: high BM25 but low frecency (rarely surfaced, long ago).
	inputB := frecency.Input{
		SurfacedCount:     1,
		LastSurfaced:      now.Add(-720 * time.Hour), // 30 days ago
		UpdatedAt:         now.Add(-720 * time.Hour),
		SurfacingContexts: []string{"session-start"},
		FilePath:          "mem/high-bm25-low-frecency.toml",
	}

	// Memory C: mid BM25, mid frecency.
	inputC := frecency.Input{
		SurfacedCount:     5,
		LastSurfaced:      now.Add(-12 * time.Hour),
		UpdatedAt:         now.Add(-24 * time.Hour),
		SurfacingContexts: []string{"prompt", "tool"},
		FilePath:          "mem/mid-bm25-mid-frecency.toml",
	}

	bm25A := 6.0  // lowest BM25
	bm25B := 10.0 // highest BM25
	bm25C := 8.0  // middle BM25

	// Before combined scoring: B > C > A by BM25 alone.
	g.Expect(bm25B).To(BeNumerically(">", bm25C))
	g.Expect(bm25C).To(BeNumerically(">", bm25A))

	combinedA := scorer.CombinedScore(bm25A, inputA)
	combinedB := scorer.CombinedScore(bm25B, inputB)
	combinedC := scorer.CombinedScore(bm25C, inputC)

	// After combined scoring: A should rank higher than B due to frecency boost.
	g.Expect(combinedA).To(BeNumerically(">", combinedB),
		"high-frecency memory should overtake high-BM25 memory after combined scoring")
	// Verify ordering actually changed from pure BM25.
	g.Expect(combinedA).To(BeNumerically(">", combinedC),
		"high-frecency memory should also beat mid-BM25 memory")
}
