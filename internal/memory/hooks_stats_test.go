package memory_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/memory"
)

// TestGetHookStats verifies aggregation of hook statistics
func TestGetHookStats(t *testing.T) {
	g := NewWithT(t)

	// Create in-memory DB
	db, err := memory.InitEmbeddingsDBForTest(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	// Record multiple events
	err = memory.RecordHookEvent(db, "hook-a", 0, 100)
	g.Expect(err).ToNot(HaveOccurred())
	err = memory.RecordHookEvent(db, "hook-a", 1, 200)
	g.Expect(err).ToNot(HaveOccurred())
	err = memory.RecordHookEvent(db, "hook-b", 0, 150)
	g.Expect(err).ToNot(HaveOccurred())

	// Retrieve stats
	stats, err := memory.GetHookStats(db)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(stats).To(HaveLen(2))

	// Find hook-a stats
	var hookA *memory.HookStat

	for i := range stats {
		if stats[i].HookName == "hook-a" {
			hookA = &stats[i]
			break
		}
	}

	g.Expect(hookA).ToNot(BeNil())

	if hookA == nil {
		t.Fatal("hookA is nil")
	}

	g.Expect(hookA.FireCount).To(Equal(2))
	g.Expect(hookA.SuccessRate).To(Equal(0.5)) // 1 success out of 2
	g.Expect(hookA.AvgDuration).To(Equal(150)) // (100 + 200) / 2
}

// TestRecordHookEvent verifies hook event recording and retrieval
func TestRecordHookEvent(t *testing.T) {
	g := NewWithT(t)

	// Create in-memory DB
	db, err := memory.InitEmbeddingsDBForTest(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	// Record a hook event
	err = memory.RecordHookEvent(db, "test-hook", 0, 100)
	g.Expect(err).ToNot(HaveOccurred())

	// Retrieve stats
	stats, err := memory.GetHookStats(db)
	g.Expect(err).ToNot(HaveOccurred())

	if len(stats) < 1 {
		t.Fatal("expected at least 1 stat from GetHookStats")
	}

	g.Expect(stats).To(HaveLen(1))
	g.Expect(stats[0].HookName).To(Equal("test-hook"))
	g.Expect(stats[0].FireCount).To(Equal(1))
	g.Expect(stats[0].SuccessRate).To(Equal(1.0))
	g.Expect(stats[0].AvgDuration).To(Equal(100))
}

// TestRecordHookEvent_Retention verifies pruning of old events
func TestRecordHookEvent_Retention(t *testing.T) {
	g := NewWithT(t)

	// Create in-memory DB
	db, err := memory.InitEmbeddingsDBForTest(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	// Record 1100 events (should keep only last 1000)
	for range 1100 {
		err = memory.RecordHookEvent(db, "test-hook", 0, 100)
		g.Expect(err).ToNot(HaveOccurred())
	}

	// Verify only 1000 events remain
	stats, err := memory.GetHookStats(db)
	g.Expect(err).ToNot(HaveOccurred())

	if len(stats) < 1 {
		t.Fatal("expected at least 1 stat from GetHookStats")
	}

	g.Expect(stats).To(HaveLen(1))
	g.Expect(stats[0].FireCount).To(Equal(1000))
}
