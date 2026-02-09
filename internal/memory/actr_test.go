package memory_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"

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
