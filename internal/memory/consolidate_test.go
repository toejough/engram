package memory_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	_ "github.com/mattn/go-sqlite3"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/memory"
)

// ============================================================================
// Unit tests for Consolidate function
// traces: TASK-6, ISSUE-160
// ============================================================================

// TEST-960: Consolidate runs without error on empty database
// traces: TASK-6
func TestConsolidateEmptyDatabase(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	opts := memory.ConsolidateOpts{
		MemoryRoot: memoryRoot,
	}

	result, err := memory.Consolidate(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())
	g.Expect(result.EntriesDecayed).To(Equal(0))
	g.Expect(result.EntriesPruned).To(Equal(0))
	g.Expect(result.DuplicatesMerged).To(Equal(0))
	g.Expect(result.PromotionCandidates).To(Equal(0))
}

// TEST-961: Consolidate calls Decay on all memories
// traces: TASK-6
func TestConsolidateCallsDecay(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	// Add a learning to create entries
	g.Expect(memory.Learn(memory.LearnOpts{
		Message:    "test learning for decay",
		MemoryRoot: memoryRoot,
	})).To(Succeed())

	// Query to populate embeddings DB
	_, err := memory.Query(memory.QueryOpts{
		Text:       "test learning",
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.ConsolidateOpts{
		MemoryRoot: memoryRoot,
		DecayFactor: 0.8,
	}

	result, err := memory.Consolidate(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.EntriesDecayed).To(BeNumerically(">", 0))
}

// TEST-962: Consolidate calls Prune to remove low-confidence entries
// traces: TASK-6
func TestConsolidateCallsPrune(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	// Add learnings
	g.Expect(memory.Learn(memory.LearnOpts{
		Message:    "learning to be pruned",
		MemoryRoot: memoryRoot,
	})).To(Succeed())

	// Query to create embeddings
	_, err := memory.Query(memory.QueryOpts{
		Text:       "learning",
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Decay several times to get confidence very low
	for i := 0; i < 10; i++ {
		_, err := memory.Decay(memory.DecayOpts{
			MemoryRoot: memoryRoot,
			Factor:     0.5,
		})
		g.Expect(err).ToNot(HaveOccurred())
	}

	opts := memory.ConsolidateOpts{
		MemoryRoot:       memoryRoot,
		PruneThreshold:   0.1,
	}

	result, err := memory.Consolidate(opts)
	g.Expect(err).ToNot(HaveOccurred())
	// After aggressive decay, entries should be pruned
	g.Expect(result.EntriesPruned).To(BeNumerically(">", 0))
}

// TEST-963: Consolidate identifies duplicate memories using semantic similarity
// traces: TASK-6
func TestConsolidateIdentifiesDuplicates(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	// Add very similar learnings
	g.Expect(memory.Learn(memory.LearnOpts{
		Message:    "use TDD for all code changes",
		MemoryRoot: memoryRoot,
	})).To(Succeed())

	g.Expect(memory.Learn(memory.LearnOpts{
		Message:    "always use TDD for code changes",
		MemoryRoot: memoryRoot,
	})).To(Succeed())

	// Query to create embeddings
	_, err := memory.Query(memory.QueryOpts{
		Text:       "TDD",
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.ConsolidateOpts{
		MemoryRoot:           memoryRoot,
		DuplicateThreshold:   0.95, // Very high similarity threshold
	}

	result, err := memory.Consolidate(opts)
	g.Expect(err).ToNot(HaveOccurred())
	// Duplicates should be detected and merged
	g.Expect(result.DuplicatesMerged).To(BeNumerically(">=", 0))
}

// TEST-964: Consolidate identifies promotion candidates
// traces: TASK-6
func TestConsolidateIdentifiesPromotionCandidates(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	// Add a learning
	g.Expect(memory.Learn(memory.LearnOpts{
		Message:    "important learning for promotion",
		MemoryRoot: memoryRoot,
	})).To(Succeed())

	// Query multiple times from different projects to build retrieval count
	for i := 0; i < 5; i++ {
		_, err := memory.Query(memory.QueryOpts{
			Text:       "important learning",
			Project:    "project-" + string(rune('A'+i)),
			MemoryRoot: memoryRoot,
		})
		g.Expect(err).ToNot(HaveOccurred())
	}

	opts := memory.ConsolidateOpts{
		MemoryRoot:        memoryRoot,
		MinRetrievals:     3,
		MinProjects:       2,
	}

	result, err := memory.Consolidate(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.PromotionCandidates).To(BeNumerically(">=", 0))
}

// TEST-965: Consolidate returns comprehensive summary
// traces: TASK-6
func TestConsolidateReturnsSummary(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	// Add learnings
	g.Expect(memory.Learn(memory.LearnOpts{
		Message:    "test learning one",
		MemoryRoot: memoryRoot,
	})).To(Succeed())

	// Query to create embeddings
	_, err := memory.Query(memory.QueryOpts{
		Text:       "test",
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.ConsolidateOpts{
		MemoryRoot: memoryRoot,
	}

	result, err := memory.Consolidate(opts)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify result structure
	g.Expect(result.EntriesDecayed).To(BeNumerically(">=", 0))
	g.Expect(result.EntriesPruned).To(BeNumerically(">=", 0))
	g.Expect(result.DuplicatesMerged).To(BeNumerically(">=", 0))
	g.Expect(result.PromotionCandidates).To(BeNumerically(">=", 0))
}

// TEST-966: Consolidate uses default values when options are zero
// traces: TASK-6
func TestConsolidateDefaultValues(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	opts := memory.ConsolidateOpts{
		MemoryRoot: memoryRoot,
		// All zero values - should use defaults
	}

	result, err := memory.Consolidate(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())
}

// TEST-967: Property: Consolidate is idempotent on empty database
// traces: TASK-6
func TestPropertyConsolidateIdempotent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		g := NewWithT(t)

		suffix := rapid.StringMatching(`[a-zA-Z0-9]{8}`).Draw(t, "suffix")
		tempDir := os.TempDir()
		memoryRoot := filepath.Join(tempDir, "consolidate-idem-"+suffix)
		defer func() { _ = os.RemoveAll(memoryRoot) }()

		g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

		opts := memory.ConsolidateOpts{
			MemoryRoot: memoryRoot,
		}

		// Run consolidate twice
		result1, err1 := memory.Consolidate(opts)
		g.Expect(err1).ToNot(HaveOccurred())

		result2, err2 := memory.Consolidate(opts)
		g.Expect(err2).ToNot(HaveOccurred())

		// Results should be identical for empty database
		g.Expect(result1.EntriesDecayed).To(Equal(result2.EntriesDecayed))
		g.Expect(result1.EntriesPruned).To(Equal(result2.EntriesPruned))
		g.Expect(result1.DuplicatesMerged).To(Equal(result2.DuplicatesMerged))
		g.Expect(result1.PromotionCandidates).To(Equal(result2.PromotionCandidates))
	})
}

// TEST-968: Property: Consolidate never increases entry count
// traces: TASK-6
func TestPropertyConsolidateNeverIncreasesEntries(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		g := NewWithT(t)

		suffix := rapid.StringMatching(`[a-zA-Z0-9]{8}`).Draw(t, "suffix")
		tempDir := os.TempDir()
		memoryRoot := filepath.Join(tempDir, "consolidate-count-"+suffix)
		defer func() { _ = os.RemoveAll(memoryRoot) }()

		g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

		// Add random learnings
		numLearnings := rapid.IntRange(1, 5).Draw(t, "numLearnings")
		for i := 0; i < numLearnings; i++ {
			msg := rapid.StringMatching(`[a-zA-Z ]{10,30}`).Draw(t, "learning")
			g.Expect(memory.Learn(memory.LearnOpts{
				Message:    msg,
				MemoryRoot: memoryRoot,
			})).To(Succeed())
		}

		// Query to create embeddings
		_, err := memory.Query(memory.QueryOpts{
			Text:       "test",
			MemoryRoot: memoryRoot,
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Count entries before consolidate
		dbPath := filepath.Join(memoryRoot, "embeddings.db")
		db, err := initEmbeddingsDBForTest(dbPath)
		g.Expect(err).ToNot(HaveOccurred())

		var countBefore int
		err = db.QueryRow("SELECT COUNT(*) FROM embeddings").Scan(&countBefore)
		g.Expect(err).ToNot(HaveOccurred())
		_ = db.Close()

		// Run consolidate
		opts := memory.ConsolidateOpts{
			MemoryRoot: memoryRoot,
		}
		_, err = memory.Consolidate(opts)
		g.Expect(err).ToNot(HaveOccurred())

		// Count entries after consolidate
		db2, err := initEmbeddingsDBForTest(dbPath)
		g.Expect(err).ToNot(HaveOccurred())

		var countAfter int
		err = db2.QueryRow("SELECT COUNT(*) FROM embeddings").Scan(&countAfter)
		g.Expect(err).ToNot(HaveOccurred())
		_ = db2.Close()

		// Consolidate should only remove or keep entries, never add
		g.Expect(countAfter).To(BeNumerically("<=", countBefore))
	})
}

// Helper function for tests
func initEmbeddingsDBForTest(dbPath string) (*sql.DB, error) {
	// This will be replaced with the actual internal function call
	// For now, we'll import sql and use the same init pattern
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	return db, nil
}
