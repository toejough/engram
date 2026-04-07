package memory_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
)

func TestEffectiveness_AllFollowed(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	mem := &memory.Stored{
		FollowedCount:    10,
		NotFollowedCount: 0,
		IrrelevantCount:  0,
	}
	g.Expect(mem.Effectiveness()).To(Equal(1.0))
}

func TestEffectiveness_ExactlyFiftyPercent(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	mem := &memory.Stored{
		FollowedCount:    5,
		NotFollowedCount: 3,
		IrrelevantCount:  2,
	}
	g.Expect(mem.Effectiveness()).To(Equal(0.5))
}

func TestEffectiveness_MixedCounts(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	mem := &memory.Stored{
		FollowedCount:    3,
		NotFollowedCount: 4,
		IrrelevantCount:  3,
	}
	g.Expect(mem.Effectiveness()).To(BeNumerically("~", 0.3, 0.001))
}

func TestEffectiveness_NoneFollowed(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	mem := &memory.Stored{
		FollowedCount:    0,
		NotFollowedCount: 5,
		IrrelevantCount:  3,
	}
	g.Expect(mem.Effectiveness()).To(Equal(0.0))
}

func TestEffectiveness_ZeroEvaluations(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	mem := &memory.Stored{}
	g.Expect(mem.Effectiveness()).To(Equal(0.0))
}

func TestQuadrant_BelowFiftyPercent_AtMedian_IsLeech(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	mem := &memory.Stored{
		FollowedCount:    2,
		NotFollowedCount: 6,
		IrrelevantCount:  2,
		SurfacedCount:    10,
	}
	// 20% effective, surfaced exactly at median -> leech
	g.Expect(mem.Quadrant(10)).To(Equal(memory.QuadrantLeech))
}

func TestQuadrant_ExactlyAtMedian_IsAboveMedian(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	mem := &memory.Stored{
		FollowedCount:    8,
		NotFollowedCount: 2,
		SurfacedCount:    10,
	}
	// 80% effective, surfaced exactly at median of 10 -> working (>= median)
	g.Expect(mem.Quadrant(10)).To(Equal(memory.QuadrantWorking))
}

func TestQuadrant_ExactlyFiftyPercent_IsEffective(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	mem := &memory.Stored{
		FollowedCount:    5,
		NotFollowedCount: 3,
		IrrelevantCount:  2,
		SurfacedCount:    15,
	}
	// 50% effective (exactly at threshold), surfaced above median -> working
	g.Expect(mem.Quadrant(10)).To(Equal(memory.QuadrantWorking))
}

func TestQuadrant_ExactlyFiveEvals_NotInsufficientData(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	mem := &memory.Stored{
		FollowedCount:    3,
		NotFollowedCount: 2,
		SurfacedCount:    10,
	}
	// 3/5 = 60% effective, surfaced at median -> working
	g.Expect(mem.Quadrant(10)).To(Equal(memory.QuadrantWorking))
}

func TestQuadrant_HiddenGem(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	mem := &memory.Stored{
		FollowedCount:    8,
		NotFollowedCount: 2,
		SurfacedCount:    5,
	}
	// 80% effective, surfaced below median of 10 -> hidden-gem
	g.Expect(mem.Quadrant(10)).To(Equal(memory.QuadrantHiddenGem))
}

func TestQuadrant_InsufficientData_FourEvals(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	mem := &memory.Stored{
		FollowedCount:    3,
		NotFollowedCount: 1,
	}
	g.Expect(mem.Quadrant(10)).To(Equal(memory.QuadrantInsufficientData))
}

func TestQuadrant_InsufficientData_ZeroEvals(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	mem := &memory.Stored{}
	g.Expect(mem.Quadrant(10)).To(Equal(memory.QuadrantInsufficientData))
}

func TestQuadrant_Leech(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	mem := &memory.Stored{
		FollowedCount:    2,
		NotFollowedCount: 6,
		IrrelevantCount:  2,
		SurfacedCount:    15,
	}
	// 20% effective, surfaced above median of 10 -> leech
	g.Expect(mem.Quadrant(10)).To(Equal(memory.QuadrantLeech))
}

func TestQuadrant_Noise(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	mem := &memory.Stored{
		FollowedCount:    2,
		NotFollowedCount: 6,
		IrrelevantCount:  2,
		SurfacedCount:    5,
	}
	// 20% effective, surfaced below median of 10 -> noise
	g.Expect(mem.Quadrant(10)).To(Equal(memory.QuadrantNoise))
}

func TestQuadrant_Working(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	mem := &memory.Stored{
		FollowedCount:    8,
		NotFollowedCount: 2,
		SurfacedCount:    15,
	}
	// 80% effective, surfaced above median of 10 -> working
	g.Expect(mem.Quadrant(10)).To(Equal(memory.QuadrantWorking))
}
