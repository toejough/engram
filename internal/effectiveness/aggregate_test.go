package effectiveness_test

import (
	"testing"

	"github.com/onsi/gomega"

	"engram/internal/effectiveness"
	"engram/internal/memory"
)

// FromMemories computes effectiveness from embedded TOML counters.
func TestFromMemories_ComputesEffectiveness(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	memories := []*memory.Stored{
		{
			FilePath:          "/data/memories/mem-a.toml",
			FollowedCount:     3,
			ContradictedCount: 1,
			IgnoredCount:      1,
		},
	}

	stats := effectiveness.FromMemories(memories)
	g.Expect(stats).To(gomega.HaveKey("/data/memories/mem-a.toml"))

	stat := stats["/data/memories/mem-a.toml"]
	g.Expect(stat.FollowedCount).To(gomega.Equal(3))
	g.Expect(stat.ContradictedCount).To(gomega.Equal(1))
	g.Expect(stat.IgnoredCount).To(gomega.Equal(1))
	g.Expect(stat.EffectivenessScore).To(gomega.BeNumerically("~", 60.0, 0.001))
}

// FromMemories returns empty map for nil input.
func TestFromMemories_EmptyInput(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	stats := effectiveness.FromMemories(nil)
	g.Expect(stats).To(gomega.BeEmpty())
}

// FromMemories: IrrelevantCount is included in the denominator.
func TestFromMemories_IncludesIrrelevantInDenominator(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	memories := []*memory.Stored{
		{
			FilePath:        "/data/memories/mem-c.toml",
			FollowedCount:   5,
			IrrelevantCount: 5,
		},
	}

	stats := effectiveness.FromMemories(memories)
	g.Expect(stats).To(gomega.HaveKey("/data/memories/mem-c.toml"))

	stat := stats["/data/memories/mem-c.toml"]
	g.Expect(stat.EffectivenessScore).To(gomega.BeNumerically("~", 50.0, 0.001))
}

// FromMemories: multiple memories produce separate stats entries.
func TestFromMemories_MultipleMemories(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	memories := []*memory.Stored{
		{FilePath: "/data/memories/a.toml", FollowedCount: 5},
		{FilePath: "/data/memories/b.toml", IgnoredCount: 3},
	}

	stats := effectiveness.FromMemories(memories)
	g.Expect(stats).To(gomega.HaveLen(2))
	g.Expect(stats["/data/memories/a.toml"].EffectivenessScore).To(gomega.BeNumerically("~", 100.0, 0.001))
	g.Expect(stats["/data/memories/b.toml"].EffectivenessScore).To(gomega.BeNumerically("~", 0.0, 0.001))
}

// FromMemories: zero evaluations produce zero effectiveness score.
func TestFromMemories_ZeroEvaluations(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	memories := []*memory.Stored{
		{
			FilePath:          "/data/memories/mem-b.toml",
			FollowedCount:     0,
			ContradictedCount: 0,
			IgnoredCount:      0,
		},
	}

	stats := effectiveness.FromMemories(memories)
	g.Expect(stats).To(gomega.HaveKey("/data/memories/mem-b.toml"))
	g.Expect(stats["/data/memories/mem-b.toml"].EffectivenessScore).To(gomega.Equal(0.0))
}
