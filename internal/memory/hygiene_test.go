package memory_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/memory"
)

// ============================================================================
// Decay tests (TASK-43)
// ============================================================================

// TEST-980: Decay reduces confidence of all entries by default factor 0.9
// traces: ARCH-062, REQ-015, REQ-016
func TestDecayReducesConfidenceByDefaultFactor(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Store a learning first (should have confidence 1.0 by default)
	learnOpts := memory.LearnOpts{
		Message:    "Test learning for decay",
		MemoryRoot: memoryDir,
	}
	err = memory.Learn(learnOpts)
	g.Expect(err).ToNot(HaveOccurred())

	// Run decay with default factor (0.9)
	decayOpts := memory.DecayOpts{
		MemoryRoot: memoryDir,
	}
	result, err := memory.Decay(decayOpts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())
	g.Expect(result.EntriesAffected).To(BeNumerically(">", 0))
	g.Expect(result.Factor).To(BeNumerically("~", 0.9, 0.001))
}

// TEST-981: Decay uses custom factor when provided
// traces: ARCH-062, REQ-015, REQ-016
func TestDecayUsesCustomFactor(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	learnOpts := memory.LearnOpts{
		Message:    "Test learning for custom decay",
		MemoryRoot: memoryDir,
	}
	err = memory.Learn(learnOpts)
	g.Expect(err).ToNot(HaveOccurred())

	decayOpts := memory.DecayOpts{
		MemoryRoot: memoryDir,
		Factor:     0.5,
	}
	result, err := memory.Decay(decayOpts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Factor).To(BeNumerically("~", 0.5, 0.001))
}

// TEST-982: Decay applied twice compounds (confidence *= factor each time)
// traces: ARCH-062, REQ-015, REQ-016
func TestDecayCompoundsOverMultipleApplications(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	learnOpts := memory.LearnOpts{
		Message:    "Compound decay test",
		MemoryRoot: memoryDir,
	}
	err = memory.Learn(learnOpts)
	g.Expect(err).ToNot(HaveOccurred())

	decayOpts := memory.DecayOpts{
		MemoryRoot: memoryDir,
		Factor:     0.5,
	}

	// First decay: 1.0 * 0.5 = 0.5
	result1, err := memory.Decay(decayOpts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result1.EntriesAffected).To(BeNumerically(">", 0))

	// Second decay: 0.5 * 0.5 = 0.25
	result2, err := memory.Decay(decayOpts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result2.EntriesAffected).To(BeNumerically(">", 0))

	// Verify the confidence has been compounded: should be 0.25 now
	g.Expect(result2.MinConfidence).To(BeNumerically("~", 0.25, 0.01))
}

// TEST-983: Decay returns count of entries affected
// traces: ARCH-062, REQ-015, REQ-016
func TestDecayReturnsEntriesAffectedCount(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Store multiple learnings
	for _, msg := range []string{"learning one", "learning two", "learning three"} {
		err = memory.Learn(memory.LearnOpts{
			Message:    msg,
			MemoryRoot: memoryDir,
		})
		g.Expect(err).ToNot(HaveOccurred())
	}

	decayOpts := memory.DecayOpts{
		MemoryRoot: memoryDir,
	}
	result, err := memory.Decay(decayOpts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.EntriesAffected).To(BeNumerically(">=", 3))
}

// TEST-984: Property-based: decay always reduces confidence (never increases)
// traces: ARCH-062, REQ-015, REQ-016
func TestDecayPropertyAlwaysReduces(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		g := NewWithT(t)

		suffix := rapid.StringMatching(`[a-zA-Z0-9]{8}`).Draw(t, "suffix")
		tempDir := os.TempDir()
		memoryDir := filepath.Join(tempDir, "decay-prop-"+suffix)
		defer func() { _ = os.RemoveAll(memoryDir) }()

		err := os.MkdirAll(memoryDir, 0755)
		g.Expect(err).ToNot(HaveOccurred())

		msg := rapid.StringMatching(`[a-zA-Z0-9 ]{10,50}`).Draw(t, "message")
		err = memory.Learn(memory.LearnOpts{
			Message:    msg,
			MemoryRoot: memoryDir,
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Factor between 0.01 and 0.99 -- always a reduction
		factor := rapid.Float64Range(0.01, 0.99).Draw(t, "factor")

		result, err := memory.Decay(memory.DecayOpts{
			MemoryRoot: memoryDir,
			Factor:     factor,
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Property: decay should always affect at least one entry
		g.Expect(result.EntriesAffected).To(BeNumerically(">", 0))

		// Property: max confidence after decay should be less than 1.0
		g.Expect(result.MaxConfidence).To(BeNumerically("<", 1.0))
	})
}

// ============================================================================
// Prune tests (TASK-43)
// ============================================================================

// TEST-985: Prune removes entries below default threshold (0.1)
// traces: ARCH-062, REQ-015, REQ-016
func TestPruneRemovesBelowDefaultThreshold(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Store learning and decay it heavily to get below 0.1
	err = memory.Learn(memory.LearnOpts{
		Message:    "Soon to be pruned",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Decay with very small factor to push confidence below 0.1
	_, err = memory.Decay(memory.DecayOpts{
		MemoryRoot: memoryDir,
		Factor:     0.05, // 1.0 * 0.05 = 0.05, below 0.1 threshold
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Prune with default threshold (0.1)
	pruneOpts := memory.PruneOpts{
		MemoryRoot: memoryDir,
	}
	result, err := memory.Prune(pruneOpts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())
	g.Expect(result.EntriesRemoved).To(BeNumerically(">", 0))
}

// TEST-986: Prune uses custom threshold when provided
// traces: ARCH-062, REQ-015, REQ-016
func TestPruneUsesCustomThreshold(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	err = memory.Learn(memory.LearnOpts{
		Message:    "Learning to prune with custom threshold",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Decay to 0.5
	_, err = memory.Decay(memory.DecayOpts{
		MemoryRoot: memoryDir,
		Factor:     0.5,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Prune with high threshold (0.6) -- entry at 0.5 should be removed
	pruneOpts := memory.PruneOpts{
		MemoryRoot: memoryDir,
		Threshold:  0.6,
	}
	result, err := memory.Prune(pruneOpts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.EntriesRemoved).To(BeNumerically(">", 0))
	g.Expect(result.Threshold).To(BeNumerically("~", 0.6, 0.001))
}

// TEST-987: Prune does not remove entries above threshold
// traces: ARCH-062, REQ-015, REQ-016
func TestPruneKeepsEntriesAboveThreshold(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Store learning (confidence 1.0)
	err = memory.Learn(memory.LearnOpts{
		Message:    "High confidence learning",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Prune with default threshold (0.1) -- entry at 1.0 should NOT be removed
	pruneOpts := memory.PruneOpts{
		MemoryRoot: memoryDir,
	}
	result, err := memory.Prune(pruneOpts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.EntriesRemoved).To(Equal(0))
	g.Expect(result.EntriesRetained).To(BeNumerically(">", 0))
}

// TEST-988: Prune returns count of removed and retained entries
// traces: ARCH-062, REQ-015, REQ-016
func TestPruneReturnsCounts(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Store one high-confidence and one that will be decayed
	err = memory.Learn(memory.LearnOpts{
		Message:    "Will be decayed",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Decay below threshold
	_, err = memory.Decay(memory.DecayOpts{
		MemoryRoot: memoryDir,
		Factor:     0.05,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Store another learning AFTER decay (this one stays at 1.0)
	err = memory.Learn(memory.LearnOpts{
		Message:    "Fresh and confident",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	pruneOpts := memory.PruneOpts{
		MemoryRoot: memoryDir,
	}
	result, err := memory.Prune(pruneOpts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.EntriesRemoved).To(BeNumerically(">", 0))
	g.Expect(result.EntriesRetained).To(BeNumerically(">", 0))
}

// TEST-989: Property-based: prune never removes entries above threshold
// traces: ARCH-062, REQ-015, REQ-016
func TestPrunePropertyNeverRemovesAboveThreshold(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		g := NewWithT(t)

		suffix := rapid.StringMatching(`[a-zA-Z0-9]{8}`).Draw(t, "suffix")
		tempDir := os.TempDir()
		memoryDir := filepath.Join(tempDir, "prune-prop-"+suffix)
		defer func() { _ = os.RemoveAll(memoryDir) }()

		err := os.MkdirAll(memoryDir, 0755)
		g.Expect(err).ToNot(HaveOccurred())

		msg := rapid.StringMatching(`[a-zA-Z0-9 ]{10,50}`).Draw(t, "message")
		err = memory.Learn(memory.LearnOpts{
			Message:    msg,
			MemoryRoot: memoryDir,
		})
		g.Expect(err).ToNot(HaveOccurred())

		// With no decay, confidence is 1.0, so any threshold < 1.0 should keep it
		threshold := rapid.Float64Range(0.01, 0.99).Draw(t, "threshold")
		result, err := memory.Prune(memory.PruneOpts{
			MemoryRoot: memoryDir,
			Threshold:  threshold,
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Property: fresh entries (confidence 1.0) are never pruned
		g.Expect(result.EntriesRemoved).To(Equal(0))
	})
}

// ============================================================================
// Conflict detection tests (TASK-43)
// ============================================================================

// TEST-1000: Learn detects high-similarity existing entry (>0.85) and returns conflict
// traces: ARCH-062, REQ-015, REQ-016
func TestLearnDetectsConflictWithHighSimilarity(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Store an initial learning
	err = memory.Learn(memory.LearnOpts{
		Message:    "PostgreSQL is the best database for complex queries",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Store a nearly identical learning -- should detect conflict
	result, err := memory.LearnWithConflictCheck(memory.LearnOpts{
		Message:    "PostgreSQL is the best database for complex queries and joins",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())
	g.Expect(result.HasConflict).To(BeTrue())
	g.Expect(result.ConflictEntry).ToNot(BeEmpty())
	g.Expect(result.Similarity).To(BeNumerically(">", 0.85))
}

// TEST-1001: Learn does not flag conflict when entries are dissimilar
// traces: ARCH-062, REQ-015, REQ-016
func TestLearnNoConflictWhenDissimilar(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Store an initial learning
	err = memory.Learn(memory.LearnOpts{
		Message:    "PostgreSQL is the best database for complex queries",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Store a completely different learning -- no conflict
	result, err := memory.LearnWithConflictCheck(memory.LearnOpts{
		Message:    "CSS grid layout is useful for responsive web design",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())
	g.Expect(result.HasConflict).To(BeFalse())
}

// TEST-1002: Conflict detection returns the conflicting entry content
// traces: ARCH-062, REQ-015, REQ-016
func TestConflictDetectionReturnsConflictingContent(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	err = memory.Learn(memory.LearnOpts{
		Message:    "Always use dependency injection for testability",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Near-duplicate
	result, err := memory.LearnWithConflictCheck(memory.LearnOpts{
		Message:    "Use dependency injection for better testability",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.HasConflict).To(BeTrue())
	g.Expect(result.ConflictEntry).To(ContainSubstring("dependency injection"))
}

// TEST-1003: Learn still stores entry even when conflict is detected
// traces: ARCH-062, REQ-015, REQ-016
func TestLearnStoresEntryEvenWithConflict(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	err = memory.Learn(memory.LearnOpts{
		Message:    "PostgreSQL is the best database for complex queries",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Conflict-checked learn should still store the new entry
	newMsg := "PostgreSQL is the best database for complex queries and joins"
	result, err := memory.LearnWithConflictCheck(memory.LearnOpts{
		Message:    newMsg,
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Stored).To(BeTrue())

	// Verify the entry exists in index.md
	content, err := os.ReadFile(filepath.Join(memoryDir, "index.md"))
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content)).To(ContainSubstring(newMsg))
}

// ============================================================================
// Confidence-weighted search tests (TASK-43)
// ============================================================================

// TEST-1004: searchSimilar ranks by (cosine_similarity * confidence)
// traces: ARCH-062, REQ-015, REQ-016
func TestConfidenceWeightedSearchRanking(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Store two semantically similar entries
	err = memory.Learn(memory.LearnOpts{
		Message:    "Database query optimization techniques",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	err = memory.Learn(memory.LearnOpts{
		Message:    "Database indexing for faster queries",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Decay everything
	_, err = memory.Decay(memory.DecayOpts{
		MemoryRoot: memoryDir,
		Factor:     0.1, // Very low confidence
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Add a fresh entry about databases -- this one has confidence 1.0
	err = memory.Learn(memory.LearnOpts{
		Message:    "SQL database performance tuning guide",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Query for database-related content
	opts := memory.QueryOpts{
		Text:       "database performance",
		Limit:      3,
		MemoryRoot: memoryDir,
	}

	results, err := memory.Query(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results.Results).ToNot(BeEmpty())

	// The fresh entry (confidence 1.0) should rank higher than
	// decayed entries (confidence 0.1) even if raw similarity is similar
	topResult := results.Results[0]
	g.Expect(topResult.Content).To(ContainSubstring("SQL database performance tuning"))
}

// TEST-1005: QueryResult includes confidence score
// traces: ARCH-062, REQ-015, REQ-016
func TestQueryResultIncludesConfidence(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	err = memory.Learn(memory.LearnOpts{
		Message:    "Test confidence in query results",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.QueryOpts{
		Text:       "confidence",
		MemoryRoot: memoryDir,
	}

	results, err := memory.Query(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results.Results).ToNot(BeEmpty())

	// Each result should have a Confidence field
	for _, r := range results.Results {
		g.Expect(r.Confidence).To(BeNumerically(">", 0))
		g.Expect(r.Confidence).To(BeNumerically("<=", 1.0))
	}
}

// TEST-1006: Decayed entries have lower effective score than fresh ones
// traces: ARCH-062, REQ-015, REQ-016
func TestDecayedEntriesHaveLowerEffectiveScore(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Store an entry and decay it
	err = memory.Learn(memory.LearnOpts{
		Message:    "Use indexes on frequently queried columns to speed up SQL database performance",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	_, err = memory.Decay(memory.DecayOpts{
		MemoryRoot: memoryDir,
		Factor:     0.1,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Add fresh entry on same topic but different enough to avoid learn-time dedup
	err = memory.Learn(memory.LearnOpts{
		Message:    "The EXPLAIN ANALYZE command helps identify slow SQL queries that need optimization",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.QueryOpts{
		Text:       "SQL optimization",
		Limit:      2,
		MemoryRoot: memoryDir,
	}

	results, err := memory.Query(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results.Results).To(HaveLen(2))

	// The score incorporates confidence, so fresh entry should score higher
	g.Expect(results.Results[0].Score).To(BeNumerically(">", results.Results[1].Score))
	g.Expect(results.Results[0].Content).To(ContainSubstring("EXPLAIN"))
}

// TEST-1007: Property-based: score is always in [0, 1] for confidence-weighted search
// traces: ARCH-062, REQ-015, REQ-016
func TestConfidenceWeightedScorePropertyBounded(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		g := NewWithT(t)

		suffix := rapid.StringMatching(`[a-zA-Z0-9]{8}`).Draw(t, "suffix")
		tempDir := os.TempDir()
		memoryDir := filepath.Join(tempDir, "cscore-prop-"+suffix)
		defer func() { _ = os.RemoveAll(memoryDir) }()

		err := os.MkdirAll(memoryDir, 0755)
		g.Expect(err).ToNot(HaveOccurred())

		msg := rapid.StringMatching(`[a-zA-Z0-9 ]{10,50}`).Draw(t, "message")
		err = memory.Learn(memory.LearnOpts{
			Message:    msg,
			MemoryRoot: memoryDir,
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Optionally decay
		if rapid.Bool().Draw(t, "shouldDecay") {
			factor := rapid.Float64Range(0.1, 0.99).Draw(t, "factor")
			_, err = memory.Decay(memory.DecayOpts{
				MemoryRoot: memoryDir,
				Factor:     factor,
			})
			g.Expect(err).ToNot(HaveOccurred())
		}

		queryText := rapid.StringMatching(`[a-zA-Z0-9 ]{5,20}`).Draw(t, "query")
		results, err := memory.Query(memory.QueryOpts{
			Text:       queryText,
			MemoryRoot: memoryDir,
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Property: all scores must be in [0, 1]
		for _, r := range results.Results {
			g.Expect(r.Score).To(BeNumerically(">=", 0))
			g.Expect(r.Score).To(BeNumerically("<=", 1.0))
		}
	})
}
