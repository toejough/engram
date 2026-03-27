package review_test

import (
	"strings"
	"testing"

	"github.com/onsi/gomega"

	"engram/internal/effectiveness"
	"engram/internal/review"
)

// T-127: Empty input produces empty output.
func TestClassify_EmptyInput(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	result := review.Classify(
		map[string]effectiveness.Stat{},
		map[string]review.TrackingData{},
	)

	g.Expect(result).To(gomega.BeEmpty())
}

// T-124: Memory with 5+ evaluations and effectiveness < 40% is flagged.
func TestClassify_FlaggedWhenLowEffectiveness(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	stats := map[string]effectiveness.Stat{
		"low-perf": {FollowedCount: 2, IgnoredCount: 4, EffectivenessScore: 33.3},
	}
	tracking := map[string]review.TrackingData{
		"low-perf": {SurfacedCount: 3},
	}

	result := review.Classify(stats, tracking)

	g.Expect(result).To(gomega.HaveLen(1))
	g.Expect(result[0].Flagged).To(gomega.BeTrue())
}

// T-126: Memory with effectiveness exactly 50% classified as high follow-through quadrant.
func TestClassify_HighFollowThroughAt50Percent(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	// 5 followed, 5 other → total=10, score=50%.
	stats := map[string]effectiveness.Stat{
		"borderline": {
			FollowedCount:      5,
			ContradictedCount:  3,
			IgnoredCount:       2,
			EffectivenessScore: 50.0,
		},
	}
	tracking := map[string]review.TrackingData{
		"borderline": {SurfacedCount: 5},
	}

	result := review.Classify(stats, tracking)

	g.Expect(result).To(gomega.HaveLen(1))

	highFollowThrough := gomega.Or(
		gomega.Equal(review.Working),
		gomega.Equal(review.HiddenGem),
	)
	g.Expect(result[0].Quadrant).To(highFollowThrough)
}

// T-123: Memories with fewer than 5 evaluations classified as InsufficientData.
func TestClassify_InsufficientData(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	stats := map[string]effectiveness.Stat{
		"sparse": {FollowedCount: 2, ContradictedCount: 1, EffectivenessScore: 66.6},
	}
	tracking := map[string]review.TrackingData{
		"sparse": {SurfacedCount: 5},
	}

	result := review.Classify(stats, tracking)

	g.Expect(result).To(gomega.HaveLen(1))
	g.Expect(result[0].Quadrant).To(gomega.Equal(review.InsufficientData))
	g.Expect(result[0].Flagged).To(gomega.BeFalse())
}

// T-125: Memory with effectiveness exactly 40% is not flagged (boundary: strictly < 40%).
func TestClassify_NotFlaggedAtExactly40Percent(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	// 2 followed, 3 contradicted → total=5, score=2/5*100=40%.
	stats := map[string]effectiveness.Stat{
		"boundary": {FollowedCount: 2, ContradictedCount: 3, EffectivenessScore: 40.0},
	}
	tracking := map[string]review.TrackingData{
		"boundary": {SurfacedCount: 1},
	}

	result := review.Classify(stats, tracking)

	g.Expect(result).To(gomega.HaveLen(1))
	g.Expect(result[0].Flagged).To(gomega.BeFalse())
}

// T-122: Correct quadrant assignment based on median + effectiveness threshold.
// Uses 4 memories (1 per quadrant) with surfaced counts [2, 2, 10, 10] → median=6.
// > 6: working (10, 80%) → Working; leech (10, 30%) → Leech.
// <= 6: hidden-gem (2, 80%) → HiddenGem; noise (2, 33%) → Noise.
func TestClassify_QuadrantAssignment(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	stats := map[string]effectiveness.Stat{
		"working":    {FollowedCount: 4, IgnoredCount: 1, EffectivenessScore: 80.0},
		"hidden-gem": {FollowedCount: 4, IgnoredCount: 1, EffectivenessScore: 80.0},
		"leech":      {FollowedCount: 3, ContradictedCount: 7, EffectivenessScore: 30.0},
		"noise":      {FollowedCount: 2, ContradictedCount: 4, EffectivenessScore: 33.3},
	}
	tracking := map[string]review.TrackingData{
		"working":    {SurfacedCount: 10},
		"hidden-gem": {SurfacedCount: 2},
		"leech":      {SurfacedCount: 10},
		"noise":      {SurfacedCount: 2},
	}

	result := review.Classify(stats, tracking)

	g.Expect(result).To(gomega.HaveLen(4))

	byName := indexByName(result)
	g.Expect(byName["working"].Quadrant).To(gomega.Equal(review.Working))
	g.Expect(byName["hidden-gem"].Quadrant).To(gomega.Equal(review.HiddenGem))
	g.Expect(byName["leech"].Quadrant).To(gomega.Equal(review.Leech))
	g.Expect(byName["noise"].Quadrant).To(gomega.Equal(review.Noise))
}

// T-128: Memories with tracking data but no evaluations classified as InsufficientData.
func TestClassify_TrackingOnlyNoEvals(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	tracking := map[string]review.TrackingData{
		"track-a": {SurfacedCount: 3},
		"track-b": {SurfacedCount: 7},
		"track-c": {SurfacedCount: 1},
	}

	result := review.Classify(map[string]effectiveness.Stat{}, tracking)

	g.Expect(result).To(gomega.HaveLen(3))

	for _, mem := range result {
		g.Expect(mem.Quadrant).
			To(gomega.Equal(review.InsufficientData), "expected InsufficientData for %s", mem.Name)
		g.Expect(mem.Flagged).To(gomega.BeFalse(), "expected not flagged for %s", mem.Name)
	}
}

// TestClassify_WithEffectivenessThreshold verifies that a custom effectiveness threshold
// changes quadrant assignment from HiddenGem to Noise when the score is below the custom boundary.
func TestClassify_WithEffectivenessThreshold(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	// Score of 60% is above the default 50% threshold (Hidden Gem) but below 70%.
	stats := map[string]effectiveness.Stat{
		"borderline": {FollowedCount: 3, ContradictedCount: 2, EffectivenessScore: 60.0},
	}
	tracking := map[string]review.TrackingData{
		"borderline": {SurfacedCount: 1},
	}

	// With default threshold (50%), score 60% → high follow-through (HiddenGem).
	defaultResult := review.Classify(stats, tracking)

	g.Expect(defaultResult).To(gomega.HaveLen(1))
	g.Expect(defaultResult[0].Quadrant).To(gomega.Equal(review.HiddenGem))

	// With custom threshold (70%), score 60% → low follow-through (Noise).
	customResult := review.Classify(stats, tracking, review.WithEffectivenessThreshold(70.0))

	g.Expect(customResult).To(gomega.HaveLen(1))
	g.Expect(customResult[0].Quadrant).To(gomega.Equal(review.Noise))
}

// TestClassify_WithFlagThreshold verifies that a custom flag threshold changes flagging behavior.
func TestClassify_WithFlagThreshold(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	// Score of 45% is above the default 40% flag threshold (not flagged) but below 50%.
	stats := map[string]effectiveness.Stat{
		"borderline": {FollowedCount: 9, ContradictedCount: 11, EffectivenessScore: 45.0},
	}
	tracking := map[string]review.TrackingData{
		"borderline": {SurfacedCount: 1},
	}

	// With default threshold (40%), score 45% → not flagged.
	defaultResult := review.Classify(stats, tracking)

	g.Expect(defaultResult).To(gomega.HaveLen(1))
	g.Expect(defaultResult[0].Flagged).To(gomega.BeFalse())

	// With custom flag threshold (50%), score 45% → flagged.
	customResult := review.Classify(stats, tracking, review.WithFlagThreshold(50.0))

	g.Expect(customResult).To(gomega.HaveLen(1))
	g.Expect(customResult[0].Flagged).To(gomega.BeTrue())
}

// TestClassify_WithMinEvaluations verifies that a custom minEvaluations threshold changes
// the InsufficientData boundary.
func TestClassify_WithMinEvaluations(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	// Total of 3 evaluations: below default (5) but above custom minimum (2).
	stats := map[string]effectiveness.Stat{
		"sparse": {FollowedCount: 2, ContradictedCount: 1, EffectivenessScore: 66.6},
	}
	tracking := map[string]review.TrackingData{
		"sparse": {SurfacedCount: 1},
	}

	// With default minimum (5), 3 evals → InsufficientData.
	defaultResult := review.Classify(stats, tracking)

	g.Expect(defaultResult).To(gomega.HaveLen(1))
	g.Expect(defaultResult[0].Quadrant).To(gomega.Equal(review.InsufficientData))

	// With custom minimum (2), 3 evals → assigned a real quadrant.
	customResult := review.Classify(stats, tracking, review.WithMinEvaluations(2))

	g.Expect(customResult).To(gomega.HaveLen(1))
	g.Expect(customResult[0].Quadrant).NotTo(gomega.Equal(review.InsufficientData))
}

// TestRender_NoFlaggedSection omits the flagged section when no memories are flagged.
func TestRender_NoFlaggedSection(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	classified := []review.ClassifiedMemory{
		{
			Name:               "gem",
			Quadrant:           review.HiddenGem,
			SurfacedCount:      1,
			EffectivenessScore: 75.0,
			EvaluationCount:    8,
		},
	}

	var buf strings.Builder
	review.Render(classified, &buf)
	output := buf.String()

	g.Expect(output).NotTo(gomega.ContainSubstring("Flagged for action"))
	g.Expect(output).NotTo(gomega.ContainSubstring("Insufficient data"))
}

// TestRender_NoInsufficientSection omits the insufficient-data section when all have 5+ evals.
func TestRender_NoInsufficientSection(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	classified := []review.ClassifiedMemory{
		{
			Name:               "noisy",
			Quadrant:           review.Noise,
			SurfacedCount:      1,
			EffectivenessScore: 25.0,
			EvaluationCount:    8,
			Flagged:            true,
		},
	}

	var buf strings.Builder
	review.Render(classified, &buf)
	output := buf.String()

	g.Expect(output).To(gomega.ContainSubstring("Flagged for action"))
	g.Expect(output).NotTo(gomega.ContainSubstring("Insufficient data"))
}

// TestRender_OutputFormat checks the human-readable output format per DES-16.
func TestRender_OutputFormat(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	classified := []review.ClassifiedMemory{
		{
			Name:               "good-memory",
			Quadrant:           review.Working,
			SurfacedCount:      10,
			EffectivenessScore: 80.0,
			EvaluationCount:    5,
		},
		{
			Name:               "low-perf",
			Quadrant:           review.Noise,
			SurfacedCount:      2,
			EffectivenessScore: 30.0,
			EvaluationCount:    6,
			Flagged:            true,
		},
		{
			Name:               "new-one",
			Quadrant:           review.InsufficientData,
			SurfacedCount:      1,
			EffectivenessScore: 0,
			EvaluationCount:    2,
		},
	}

	var buf strings.Builder
	review.Render(classified, &buf)
	output := buf.String()

	g.Expect(output).To(gomega.ContainSubstring("[engram] Memory Effectiveness Review"))
	g.Expect(output).
		To(gomega.ContainSubstring("Total: 3 memories, 2 with sufficient data, 1 flagged"))
	g.Expect(output).To(gomega.ContainSubstring("Quadrant Summary:"))
	g.Expect(output).To(gomega.ContainSubstring("Working:    1"))
	g.Expect(output).To(gomega.ContainSubstring("Flagged for action"))
	g.Expect(output).To(gomega.ContainSubstring("low-perf"))
	g.Expect(output).To(gomega.ContainSubstring("Insufficient data"))
	g.Expect(output).To(gomega.ContainSubstring("new-one"))
}

// indexByName converts a slice of ClassifiedMemory to a map keyed by Name.
func indexByName(memories []review.ClassifiedMemory) map[string]review.ClassifiedMemory {
	result := make(map[string]review.ClassifiedMemory, len(memories))
	for _, mem := range memories {
		result[mem.Name] = mem
	}

	return result
}
