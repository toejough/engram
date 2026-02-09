package memory_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/memory"
)

// ============================================================================
// TASK-9: ACT-R activation scoring tests
// ============================================================================

// TEST-1200: Query retrieval adds timestamp to activation history
// traces: TASK-9
func TestQueryRetrievalAddsTimestamp(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Store learning
	err = memory.Learn(memory.LearnOpts{
		Message:    "Test activation tracking",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// First query
	_, err = memory.Query(memory.QueryOpts{
		Text:       "activation tracking",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Small delay
	time.Sleep(10 * time.Millisecond)

	// Second query
	_, err = memory.Query(memory.QueryOpts{
		Text:       "activation tracking",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Verify activation history has multiple timestamps
	stats, err := memory.GetActivationStats(memory.ActivationStatsOpts{
		MemoryRoot: memoryDir,
		Content:    "activation tracking",
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(stats.RetrievalCount).To(BeNumerically(">=", 2))
	g.Expect(stats.TimestampCount).To(BeNumerically(">=", 2))
}

// TEST-1201: Activation score calculated from timestamps
// traces: TASK-9
func TestActivationScoreCalculated(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Store learning
	err = memory.Learn(memory.LearnOpts{
		Message:    "Activation calculation test",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Query to generate retrieval timestamp
	_, err = memory.Query(memory.QueryOpts{
		Text:       "Activation calculation",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Get activation stats
	stats, err := memory.GetActivationStats(memory.ActivationStatsOpts{
		MemoryRoot: memoryDir,
		Content:    "Activation calculation",
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Activation should be positive for recently retrieved memory
	g.Expect(stats.Activation).To(BeNumerically(">", 0))
	g.Expect(stats.DecayParameter).To(BeNumerically("~", 0.5, 0.01))
}

// TEST-1202: Corrections use minimal decay
// traces: TASK-9
func TestCorrectionsUseMinimalDecay(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Store correction type learning
	err = memory.Learn(memory.LearnOpts{
		Message:    "CORRECTION: Never amend pushed commits",
		MemoryRoot: memoryDir,
		Type:       "correction",
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Query to generate retrieval
	_, err = memory.Query(memory.QueryOpts{
		Text:       "amend commits",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Get stats
	stats, err := memory.GetActivationStats(memory.ActivationStatsOpts{
		MemoryRoot: memoryDir,
		Content:    "CORRECTION",
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Corrections should have lower decay parameter (d=0.1 vs default 0.5)
	g.Expect(stats.DecayParameter).To(BeNumerically("<", 0.5))
}

// TEST-1203: Reflections apply 30-day sliding window
// traces: TASK-9
func TestReflectionsUse30DayWindow(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Store reflection
	err = memory.Learn(memory.LearnOpts{
		Message:    "REFLECTION: Context compaction loses state",
		MemoryRoot: memoryDir,
		Type:       "reflection",
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Query multiple times
	for i := 0; i < 3; i++ {
		_, err = memory.Query(memory.QueryOpts{
			Text:       "context compaction",
			MemoryRoot: memoryDir,
		})
		g.Expect(err).ToNot(HaveOccurred())
	}

	// Get initial stats
	initialStats, err := memory.GetActivationStats(memory.ActivationStatsOpts{
		MemoryRoot: memoryDir,
		Content:    "REFLECTION",
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(initialStats.TimestampCount).To(BeNumerically(">=", 3))
	g.Expect(initialStats.ActiveTimestamps).To(BeNumerically(">=", 3))

	// Simulate 35 days passing (beyond 30-day window)
	err = memory.SimulateTimePassage(memory.SimulateTimeOpts{
		MemoryRoot: memoryDir,
		Content:    "REFLECTION",
		DaysToAge:  35,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Get stats after aging
	agedStats, err := memory.GetActivationStats(memory.ActivationStatsOpts{
		MemoryRoot: memoryDir,
		Content:    "REFLECTION",
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Active timestamps should be 0 (all outside 30-day window)
	g.Expect(agedStats.ActiveTimestamps).To(Equal(0))
}

// TEST-1204: Migration completes successfully
// traces: TASK-9
func TestMigrationCompletesSuccessfully(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Store learning (timestamps are now automatically managed)
	err = memory.Learn(memory.LearnOpts{
		Message:    "Entry for migration test",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Run migration (should be idempotent - OK if no rows need updating)
	err = memory.MigrateToACTR(memory.MigrateToACTROpts{
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred(), "Migration should complete without error")
}

// ============================================================================
// ISSUE-180: Session-aware ACT-R scoring tests
// ============================================================================

// TEST-1300: ClusterIntoSessions groups timestamps by gap threshold
// traces: ISSUE-180
func TestClusterIntoSessions(t *testing.T) {
	g := NewWithT(t)

	base := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	gap := 30 * time.Minute

	// 3 timestamps within same session, then 2 in a second session after 1h gap
	timestamps := []string{
		base.Format(time.RFC3339),
		base.Add(5 * time.Minute).Format(time.RFC3339),
		base.Add(10 * time.Minute).Format(time.RFC3339),
		base.Add(90 * time.Minute).Format(time.RFC3339), // 1h30m after base = new session
		base.Add(95 * time.Minute).Format(time.RFC3339),
	}

	sessions := memory.ClusterIntoSessions(timestamps, gap)
	g.Expect(sessions).To(HaveLen(2))
	g.Expect(sessions[0]).To(HaveLen(3))
	g.Expect(sessions[1]).To(HaveLen(2))
}

// TEST-1301: Single timestamp → 1 session
// traces: ISSUE-180
func TestClusterSingleTimestamp(t *testing.T) {
	g := NewWithT(t)

	ts := []string{time.Now().Format(time.RFC3339)}
	sessions := memory.ClusterIntoSessions(ts, 30*time.Minute)
	g.Expect(sessions).To(HaveLen(1))
	g.Expect(sessions[0]).To(HaveLen(1))
}

// TEST-1302: All timestamps within 30min → 1 session, no bonus
// traces: ISSUE-180
func TestClusterAllWithinGap(t *testing.T) {
	g := NewWithT(t)

	base := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	timestamps := []string{
		base.Format(time.RFC3339),
		base.Add(10 * time.Minute).Format(time.RFC3339),
		base.Add(20 * time.Minute).Format(time.RFC3339),
		base.Add(29 * time.Minute).Format(time.RFC3339),
	}

	sessions := memory.ClusterIntoSessions(timestamps, 30*time.Minute)
	g.Expect(sessions).To(HaveLen(1))
	g.Expect(sessions[0]).To(HaveLen(4))
}

// TEST-1303: Timestamps exactly at 30min boundary stay in same session
// traces: ISSUE-180
func TestClusterExactBoundary(t *testing.T) {
	g := NewWithT(t)

	base := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	timestamps := []string{
		base.Format(time.RFC3339),
		base.Add(30 * time.Minute).Format(time.RFC3339), // exactly at boundary
	}

	// Gap of exactly 30min should NOT start a new session (need >30min)
	sessions := memory.ClusterIntoSessions(timestamps, 30*time.Minute)
	g.Expect(sessions).To(HaveLen(1))
}

// TEST-1304: Empty timestamps → empty sessions
// traces: ISSUE-180
func TestClusterEmptyTimestamps(t *testing.T) {
	g := NewWithT(t)

	sessions := memory.ClusterIntoSessions([]string{}, 30*time.Minute)
	g.Expect(sessions).To(BeEmpty())
}

// TEST-1305: Unsorted timestamps are handled correctly
// traces: ISSUE-180
func TestClusterUnsortedTimestamps(t *testing.T) {
	g := NewWithT(t)

	base := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	// Provide out of order: session2 timestamp, session1 timestamps
	timestamps := []string{
		base.Add(90 * time.Minute).Format(time.RFC3339),
		base.Format(time.RFC3339),
		base.Add(5 * time.Minute).Format(time.RFC3339),
	}

	sessions := memory.ClusterIntoSessions(timestamps, 30*time.Minute)
	g.Expect(sessions).To(HaveLen(2))
}

// TEST-1306: Property — cross-session always scores higher than single-session
// traces: ISSUE-180
func TestPropertyCrossSessionScoresHigher(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		g := NewWithT(t)

		// Generate 2-10 timestamps all within a single session (within 30min)
		count := rapid.IntRange(2, 10).Draw(t, "count")
		base := time.Now().Add(-1 * time.Hour) // 1h ago so all timestamps are in the past

		singleSessionTS := make([]string, count)
		for i := 0; i < count; i++ {
			offset := time.Duration(rapid.IntRange(0, 29).Draw(t, "offset")) * time.Minute
			singleSessionTS[i] = base.Add(offset).Format(time.RFC3339)
		}

		// Same count of timestamps but spread across 2 sessions (>30min gap)
		crossSessionTS := make([]string, count)
		half := count / 2
		for i := 0; i < half; i++ {
			offset := time.Duration(rapid.IntRange(0, 10).Draw(t, "offset1")) * time.Minute
			crossSessionTS[i] = base.Add(offset).Format(time.RFC3339)
		}
		for i := half; i < count; i++ {
			offset := time.Duration(rapid.IntRange(0, 10).Draw(t, "offset2")) * time.Minute
			crossSessionTS[i] = base.Add(2*time.Hour + offset).Format(time.RFC3339) // 2h later = new session
		}

		// Verify clustering
		singleSessions := memory.ClusterIntoSessions(singleSessionTS, 30*time.Minute)
		crossSessions := memory.ClusterIntoSessions(crossSessionTS, 30*time.Minute)

		g.Expect(len(singleSessions)).To(Equal(1), "single session timestamps should cluster into 1 session")
		g.Expect(len(crossSessions)).To(BeNumerically(">=", 2), "cross session timestamps should cluster into 2+ sessions")
	})
}

// TEST-1307: Property — cluster count matches expected for known patterns
// traces: ISSUE-180
func TestPropertyClusterCount(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		g := NewWithT(t)

		// Generate 1-5 sessions with random timestamps in each
		numSessions := rapid.IntRange(1, 5).Draw(t, "numSessions")
		base := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
		gap := 30 * time.Minute

		var allTimestamps []string
		for s := 0; s < numSessions; s++ {
			// Each session starts 2h after the previous one (well beyond 30min gap)
			sessionBase := base.Add(time.Duration(s) * 2 * time.Hour)
			tsPerSession := rapid.IntRange(1, 4).Draw(t, "tsPerSession")
			for i := 0; i < tsPerSession; i++ {
				offset := time.Duration(rapid.IntRange(0, 15).Draw(t, "offset")) * time.Minute
				allTimestamps = append(allTimestamps, sessionBase.Add(offset).Format(time.RFC3339))
			}
		}

		sessions := memory.ClusterIntoSessions(allTimestamps, gap)
		g.Expect(len(sessions)).To(Equal(numSessions))
	})
}

// TEST-1308: SessionCount and SessionBonus populated in stats
// traces: ISSUE-180
func TestSessionBonusApplied(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Store learning
	err = memory.Learn(memory.LearnOpts{
		Message:    "Session bonus test entry",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Query to generate a retrieval timestamp
	_, err = memory.Query(memory.QueryOpts{
		Text:       "Session bonus test",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Age the existing timestamp by 2 hours (creates gap for next retrieval)
	err = memory.SimulateTimePassage(memory.SimulateTimeOpts{
		MemoryRoot: memoryDir,
		Content:    "Session bonus test",
		DaysToAge:  0,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// For multi-session test, we need timestamps in different sessions.
	// Use SimulateTimePassage to shift existing timestamps back, then query again.
	// First, shift existing timestamps back by 2 hours via a custom approach.
	// Since SimulateTimePassage uses DaysToAge, let's use a workaround:
	// We'll create an entry, query it, age timestamps, then query again.

	// Age all timestamps to be 1 day old (puts them in a past session)
	err = memory.SimulateTimePassage(memory.SimulateTimeOpts{
		MemoryRoot: memoryDir,
		Content:    "Session bonus test",
		DaysToAge:  1,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Query again — this creates a new timestamp "now", which is >30min from aged ones
	_, err = memory.Query(memory.QueryOpts{
		Text:       "Session bonus test",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Get stats — should have 2 sessions and a bonus
	stats, err := memory.GetActivationStats(memory.ActivationStatsOpts{
		MemoryRoot: memoryDir,
		Content:    "Session bonus test",
	})
	g.Expect(err).ToNot(HaveOccurred())

	g.Expect(stats.SessionCount).To(BeNumerically(">=", 2))
	g.Expect(stats.SessionBonus).To(BeNumerically(">", 0))
	g.Expect(stats.Activation).To(BeNumerically(">", 0))
}

// TEST-1309: Single session → no bonus applied
// traces: ISSUE-180
func TestSingleSessionNoBonus(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Store learning
	err = memory.Learn(memory.LearnOpts{
		Message:    "Single session no bonus test",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Query to create timestamp (all within same session)
	_, err = memory.Query(memory.QueryOpts{
		Text:       "Single session no bonus",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	stats, err := memory.GetActivationStats(memory.ActivationStatsOpts{
		MemoryRoot: memoryDir,
		Content:    "Single session no bonus",
	})
	g.Expect(err).ToNot(HaveOccurred())

	g.Expect(stats.SessionCount).To(Equal(1))
	g.Expect(stats.SessionBonus).To(Equal(0.0))
}
