package memory_test

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/memory"
)

// TestFormatDigest_EmptyDigest verifies empty digest still renders header and footer.
func TestFormatDigest_EmptyDigest(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	digest := memory.Digest{
		Since:       24 * time.Hour,
		GeneratedAt: time.Now(),
	}

	output := memory.FormatDigest(digest)

	g.Expect(output).To(ContainSubstring("Learning Digest"))
	g.Expect(output).ToNot(ContainSubstring("Recent Learnings"))
	g.Expect(output).ToNot(ContainSubstring("Skill Changes"))
	g.Expect(output).ToNot(ContainSubstring("CLAUDE.md Changes"))
}

// TestFormatDigest_RecentLearningsWithTiers verifies tier display in recent learnings.
func TestFormatDigest_RecentLearningsWithTiers(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	now := time.Now()
	digest := memory.Digest{
		Since:       24 * time.Hour,
		GeneratedAt: now,
		RecentLearnings: []memory.ChangelogEntry{
			{
				Action:          "promote",
				ContentSummary:  "TDD workflow",
				SourceTier:      "embeddings",
				DestinationTier: "skills",
			},
			{
				Action:         "store",
				ContentSummary: "Just stored",
				// No source/dest tier - only action
			},
		},
	}

	output := memory.FormatDigest(digest)

	g.Expect(output).To(ContainSubstring("embeddings"))
	g.Expect(output).To(ContainSubstring("skills"))
	g.Expect(output).To(ContainSubstring("TDD workflow"))
	g.Expect(output).To(ContainSubstring("Just stored"))
}

// TestFormatDigest_WithClaudeMDChanges verifies CLAUDE.md changes section appears.
func TestFormatDigest_WithClaudeMDChanges(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	now := time.Now()
	digest := memory.Digest{
		Since:       24 * time.Hour,
		GeneratedAt: now,
		ClaudeMDChanges: []memory.ChangelogEntry{
			{
				Timestamp:      now.Add(-1 * time.Hour),
				Action:         "promote",
				ContentSummary: "Use TDD always",
			},
		},
	}

	output := memory.FormatDigest(digest)

	g.Expect(output).To(ContainSubstring("CLAUDE.md Changes"))
	g.Expect(output).To(ContainSubstring("Use TDD always"))
}

// TestFormatDigest_WithMetrics verifies metrics section appears.
func TestFormatDigest_WithMetrics(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	now := time.Now()
	metrics := memory.MetricsSnapshot{
		CorrectionRecurrenceRate: 0.12,
		RetrievalPrecision:       0.85,
		EmbeddingCount:           42,
		SkillCount:               5,
		ClaudeMDLines:            80,
	}
	digest := memory.Digest{
		Since:         24 * time.Hour,
		GeneratedAt:   now,
		LatestMetrics: &metrics,
	}

	output := memory.FormatDigest(digest)

	g.Expect(output).To(ContainSubstring("Latest Metrics"))
	g.Expect(output).To(ContainSubstring("42"))   // embedding count
	g.Expect(output).To(ContainSubstring("0.85")) // retrieval precision
}
