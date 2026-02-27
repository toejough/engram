package memory

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func TestMatchesFilter_ActionFilterMatch(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	entry := ChangelogEntry{
		Action:    "promote",
		Timestamp: time.Now(),
	}

	filter := ChangelogFilter{Action: "promote"}

	g.Expect(matchesFilter(entry, filter)).To(BeTrue())
}

func TestMatchesFilter_ActionFilterNoMatch(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	entry := ChangelogEntry{
		Action:    "prune",
		Timestamp: time.Now(),
	}

	filter := ChangelogFilter{Action: "promote"}

	g.Expect(matchesFilter(entry, filter)).To(BeFalse())
}

func TestMatchesFilter_AllFiltersMatch(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	entry := ChangelogEntry{
		Action:          "promote",
		SourceTier:      "embeddings",
		DestinationTier: "claude-md",
		Timestamp:       time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	filter := ChangelogFilter{
		Since:           time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Action:          "promote",
		SourceTier:      "embeddings",
		DestinationTier: "claude-md",
	}

	g.Expect(matchesFilter(entry, filter)).To(BeTrue())
}

func TestMatchesFilter_DestinationTierMatch(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	entry := ChangelogEntry{
		Action:          "promote",
		DestinationTier: "claude-md",
		Timestamp:       time.Now(),
	}

	filter := ChangelogFilter{DestinationTier: "claude-md"}

	g.Expect(matchesFilter(entry, filter)).To(BeTrue())
}

func TestMatchesFilter_DestinationTierNoMatch(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	entry := ChangelogEntry{
		Action:          "promote",
		DestinationTier: "skills",
		Timestamp:       time.Now(),
	}

	filter := ChangelogFilter{DestinationTier: "claude-md"}

	g.Expect(matchesFilter(entry, filter)).To(BeFalse())
}

// ─── matchesFilter tests ───────────────────────────────────────────────────────

func TestMatchesFilter_NoFilters(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	entry := ChangelogEntry{
		Action:          "store",
		SourceTier:      "embeddings",
		DestinationTier: "claude-md",
		Timestamp:       time.Now(),
	}

	g.Expect(matchesFilter(entry, ChangelogFilter{})).To(BeTrue())
}

func TestMatchesFilter_SinceFilterAfter(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	entry := ChangelogEntry{
		Action:    "store",
		Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	filter := ChangelogFilter{
		Since: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	g.Expect(matchesFilter(entry, filter)).To(BeTrue(), "entry after Since should be included")
}

func TestMatchesFilter_SinceFilterBefore(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	entry := ChangelogEntry{
		Action:    "store",
		Timestamp: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	filter := ChangelogFilter{
		Since: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	g.Expect(matchesFilter(entry, filter)).To(BeFalse(), "entry before Since should be excluded")
}

func TestMatchesFilter_SourceTierMatch(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	entry := ChangelogEntry{
		Action:     "decay",
		SourceTier: "embeddings",
		Timestamp:  time.Now(),
	}

	filter := ChangelogFilter{SourceTier: "embeddings"}

	g.Expect(matchesFilter(entry, filter)).To(BeTrue())
}

func TestMatchesFilter_SourceTierNoMatch(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	entry := ChangelogEntry{
		Action:     "decay",
		SourceTier: "skills",
		Timestamp:  time.Now(),
	}

	filter := ChangelogFilter{SourceTier: "embeddings"}

	g.Expect(matchesFilter(entry, filter)).To(BeFalse())
}
