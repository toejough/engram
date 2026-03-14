package registry_test

import (
	"math"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/registry"
)

// --- Entry / EvaluationCounters ---

func TestT240_EvaluationCountersTotal(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	counters := registry.EvaluationCounters{
		Followed: 5, Contradicted: 2, Ignored: 1,
	}
	g.Expect(counters.Total()).To(Equal(8))
}

func TestT241_EvaluationCountersTotalZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	counters := registry.EvaluationCounters{}
	g.Expect(counters.Total()).To(Equal(0))
}

// --- Signals: Effectiveness ---

func TestT242_EffectivenessWithSufficientData(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	entry := &registry.InstructionEntry{
		Evaluations: registry.EvaluationCounters{
			Followed: 7, Contradicted: 2, Ignored: 1,
		},
	}
	result := registry.Effectiveness(entry)
	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(*result).To(BeNumerically("~", 70.0, 0.01))
}

func TestT243_EffectivenessNilBelowMinEvaluations(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	entry := &registry.InstructionEntry{
		Evaluations: registry.EvaluationCounters{
			Followed: 1, Contradicted: 1, Ignored: 0,
		},
	}
	result := registry.Effectiveness(entry)
	g.Expect(result).To(BeNil())
}

func TestT244_EffectivenessExactlyAtMinEvaluations(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	entry := &registry.InstructionEntry{
		Evaluations: registry.EvaluationCounters{
			Followed: 3, Contradicted: 0, Ignored: 0,
		},
	}
	result := registry.Effectiveness(entry)
	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(*result).To(BeNumerically("~", 100.0, 0.01))
}

func TestT245_EffectivenessZeroFollowed(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	entry := &registry.InstructionEntry{
		Evaluations: registry.EvaluationCounters{
			Followed: 0, Contradicted: 2, Ignored: 1,
		},
	}
	result := registry.Effectiveness(entry)
	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(*result).To(BeNumerically("~", 0.0, 0.01))
}

// --- Signals: Frecency ---

func TestT246_FrecencyDecaysWithTime(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	now := time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC)
	lastSurfaced := now.Add(-7 * 24 * time.Hour) // 7 days ago

	const halfLifeDays = 7.0

	entry := &registry.InstructionEntry{
		SurfacedCount: 10,
		LastSurfaced:  &lastSurfaced,
		UpdatedAt:     now.Add(-30 * 24 * time.Hour),
	}

	score := registry.Frecency(entry, now, halfLifeDays)
	// 10 * exp(-7/7) = 10 * exp(-1) ≈ 3.679
	g.Expect(score).To(BeNumerically("~", 10*math.Exp(-1), 0.01))
}

func TestT247_FrecencyUsesUpdatedAtWhenNoLastSurfaced(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	now := time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC)

	const halfLifeDays = 7.0

	entry := &registry.InstructionEntry{
		SurfacedCount: 5,
		UpdatedAt:     now.Add(-14 * 24 * time.Hour), // 14 days ago
	}

	score := registry.Frecency(entry, now, halfLifeDays)
	// 5 * exp(-14/7) = 5 * exp(-2) ≈ 0.677
	g.Expect(score).To(BeNumerically("~", 5*math.Exp(-2), 0.01))
}

func TestT248_FrecencyZeroSurfacedCountReturnsZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	now := time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC)
	entry := &registry.InstructionEntry{
		SurfacedCount: 0,
		UpdatedAt:     now,
	}

	score := registry.Frecency(entry, now, 7.0)
	g.Expect(score).To(BeNumerically("~", 0.0, 0.001))
}

func TestT249_FrecencyFutureLastSurfacedClampsToZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	now := time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC)
	future := now.Add(24 * time.Hour)

	entry := &registry.InstructionEntry{
		SurfacedCount: 10,
		LastSurfaced:  &future,
		UpdatedAt:     now,
	}

	score := registry.Frecency(entry, now, 7.0)
	// daysSince clamped to 0 → exp(0) = 1 → 10 * 1 = 10
	g.Expect(score).To(BeNumerically("~", 10.0, 0.01))
}

// --- Classify ---

func TestT250_ClassifyWorkingHighSurfacingHighEffectiveness(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	entry := &registry.InstructionEntry{
		SourceType:    "memory",
		SurfacedCount: 10,
		Evaluations: registry.EvaluationCounters{
			Followed: 8, Contradicted: 1, Ignored: 1,
		},
	}
	quadrant := registry.Classify(entry, 3, 50.0)
	g.Expect(quadrant).To(Equal(registry.Working))
}

func TestT251_ClassifyLeechHighSurfacingLowEffectiveness(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	entry := &registry.InstructionEntry{
		SourceType:    "memory",
		SurfacedCount: 10,
		Evaluations: registry.EvaluationCounters{
			Followed: 1, Contradicted: 5, Ignored: 4,
		},
	}
	quadrant := registry.Classify(entry, 3, 50.0)
	g.Expect(quadrant).To(Equal(registry.Leech))
}

func TestT252_ClassifyHiddenGemLowSurfacingHighEffectiveness(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	entry := &registry.InstructionEntry{
		SourceType:    "memory",
		SurfacedCount: 1,
		Evaluations: registry.EvaluationCounters{
			Followed: 3, Contradicted: 0, Ignored: 0,
		},
	}
	quadrant := registry.Classify(entry, 3, 50.0)
	g.Expect(quadrant).To(Equal(registry.HiddenGem))
}

func TestT253_ClassifyNoiseLowSurfacingLowEffectiveness(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	entry := &registry.InstructionEntry{
		SourceType:    "memory",
		SurfacedCount: 1,
		Evaluations: registry.EvaluationCounters{
			Followed: 0, Contradicted: 2, Ignored: 1,
		},
	}
	quadrant := registry.Classify(entry, 3, 50.0)
	g.Expect(quadrant).To(Equal(registry.Noise))
}

func TestT254_ClassifyInsufficientDataBelowMinEvals(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	entry := &registry.InstructionEntry{
		SourceType:    "memory",
		SurfacedCount: 10,
		Evaluations: registry.EvaluationCounters{
			Followed: 1, Contradicted: 0, Ignored: 0,
		},
	}
	quadrant := registry.Classify(entry, 3, 50.0)
	g.Expect(quadrant).To(Equal(registry.Insufficient))
}
