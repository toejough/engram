package memory_test

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	_ "github.com/mattn/go-sqlite3"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/memory"
)

// ============================================================================
// ISSUE-210: Enrichment gate tests for promotion pipeline
// Traces to: ISSUE-210
// ============================================================================

// TEST-210-1: Unenriched entry (principle = '') is not promoted to CLAUDE.md
func TestOptimizePromoteRejectsUnenriched(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	claudeMDPath := filepath.Join(tempDir, "CLAUDE.md")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	// Create CLAUDE.md with Promoted Learnings section
	g.Expect(os.WriteFile(claudeMDPath, []byte("## Promoted Learnings\n\n"), 0644)).To(Succeed())

	// Seed DB with unenriched entry (no principle, no enrichment)
	dbPath := filepath.Join(memoryRoot, "embeddings.db")
	db, err := sql.Open("sqlite3", dbPath)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	// Initialize the database schema
	_, err = memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())

	// Insert unenriched entry: high retrieval count, multiple projects, but principle = ''
	_, err = db.Exec(`
		INSERT INTO embeddings (
			content, source, source_type, confidence,
			observation_type, memory_type, principle, anti_pattern, rationale, enriched_content,
			retrieval_count, projects_retrieved, promoted
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "Success ISSUE-170: TDD cycle executed cleanly", "memory", "internal", 1.0,
		"", "", "", "", "", "", // All enrichment fields empty
		10, "proj1,proj2,proj3", 0)
	g.Expect(err).ToNot(HaveOccurred())

	// Run optimize
	result, err := memory.Optimize(memory.OptimizeOpts{
		MemoryRoot:    memoryRoot,
		ClaudeMDPath:  claudeMDPath,
		AutoApprove:   true,
		MinRetrievals: 5,
		MinProjects:   3,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// VERIFY: Unenriched entry should NOT be promoted
	g.Expect(result.PromotionCandidates).To(BeNumerically(">=", 1), "Should find candidate")
	g.Expect(result.PromotionsApproved).To(Equal(0), "Should not promote unenriched entry")

	// VERIFY: CLAUDE.md should NOT contain the unenriched content
	claudeContent, err := os.ReadFile(claudeMDPath)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(claudeContent)).ToNot(ContainSubstring("Success ISSUE-170"))
}

// TEST-210-2: Enriched entry (principle != '') IS promoted to CLAUDE.md
func TestOptimizePromoteAcceptsEnriched(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	claudeMDPath := filepath.Join(tempDir, "CLAUDE.md")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	// Create CLAUDE.md with Promoted Learnings section
	g.Expect(os.WriteFile(claudeMDPath, []byte("## Promoted Learnings\n\n"), 0644)).To(Succeed())

	// Seed DB with enriched entry (has principle)
	dbPath := filepath.Join(memoryRoot, "embeddings.db")
	db, err := sql.Open("sqlite3", dbPath)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	// Initialize the database schema
	_, err = memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())

	// Insert enriched entry: high retrieval count, multiple projects, AND has principle
	_, err = db.Exec(`
		INSERT INTO embeddings (
			content, source, source_type, confidence,
			observation_type, memory_type, principle, anti_pattern, rationale, enriched_content,
			retrieval_count, projects_retrieved, promoted
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "Success ISSUE-180: TDD cycle completed successfully", "memory", "internal", 1.0,
		"pattern", "reflection", "Always write tests before implementation", "Skip testing phase",
		"Tests catch regressions early", "[pattern] Always write tests before implementation - Context: Tests catch regressions early",
		10, "proj1,proj2,proj3", 0)
	g.Expect(err).ToNot(HaveOccurred())

	// Run optimize
	result, err := memory.Optimize(memory.OptimizeOpts{
		MemoryRoot:    memoryRoot,
		ClaudeMDPath:  claudeMDPath,
		AutoApprove:   true,
		MinRetrievals: 5,
		MinProjects:   3,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// VERIFY: Enriched entry should be promoted
	g.Expect(result.PromotionCandidates).To(BeNumerically(">=", 1), "Should find candidate")
	g.Expect(result.PromotionsApproved).To(BeNumerically(">=", 1), "Should promote enriched entry")

	// VERIFY: CLAUDE.md should contain the principle (not the raw observation)
	claudeContent, err := os.ReadFile(claudeMDPath)
	g.Expect(err).ToNot(HaveOccurred())
	claudeStr := string(claudeContent)
	g.Expect(claudeStr).To(ContainSubstring("Always write tests before implementation"))
	// Should NOT promote the raw content
	g.Expect(claudeStr).ToNot(ContainSubstring("Success ISSUE-180"))
}

// TEST-210-3: Purge unenriched entries from database
func TestOptimizePurgeUnenrichedEntries(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	claudeMDPath := filepath.Join(tempDir, "CLAUDE.md")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	// Seed DB with mix of enriched and unenriched entries
	dbPath := filepath.Join(memoryRoot, "embeddings.db")
	db, err := sql.Open("sqlite3", dbPath)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	// Initialize the database schema
	_, err = memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())

	// Insert 3 unenriched entries (raw observations)
	for i := 1; i <= 3; i++ {
		_, err = db.Exec(`
			INSERT INTO embeddings (
				content, source, source_type, confidence,
				observation_type, memory_type, principle, anti_pattern, rationale, enriched_content,
				retrieval_count, projects_retrieved, promoted
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, fmt.Sprintf("Success ISSUE-%d: Raw observation without principle", i), "memory", "internal", 1.0,
			"", "", "", "", "", "", // No enrichment
			3, "proj1,proj2", 0)
		g.Expect(err).ToNot(HaveOccurred())
	}

	// Insert 1 enriched entry
	_, err = db.Exec(`
		INSERT INTO embeddings (
			content, source, source_type, confidence,
			observation_type, memory_type, principle, anti_pattern, rationale, enriched_content,
			retrieval_count, projects_retrieved, promoted
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "Enriched observation", "memory", "internal", 1.0,
		"pattern", "reflection", "Test principle", "", "", "Enriched content",
		3, "proj1,proj2", 0)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify 4 entries exist before purge
	var countBefore int
	err = db.QueryRow("SELECT COUNT(*) FROM embeddings").Scan(&countBefore)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(countBefore).To(Equal(4))

	// Run optimize with purge enabled
	_, err = memory.Optimize(memory.OptimizeOpts{
		MemoryRoot:   memoryRoot,
		ClaudeMDPath: claudeMDPath,
		AutoApprove:  true,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// VERIFY: Unenriched entries should be purged
	// NOTE: UnenrichedPurged field will be added by implementation
	// g.Expect(result.UnenrichedPurged).To(BeNumerically(">=", 3), "Should purge at least 3 unenriched entries")

	// VERIFY: Only enriched entry survives
	var countAfter int
	err = db.QueryRow("SELECT COUNT(*) FROM embeddings WHERE principle != ''").Scan(&countAfter)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(countAfter).To(BeNumerically(">=", 1), "Enriched entry should survive")

	// VERIFY: Unenriched entries are removed
	var unenrichedCount int
	err = db.QueryRow("SELECT COUNT(*) FROM embeddings WHERE principle = '' AND promoted = 0").Scan(&unenrichedCount)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(unenrichedCount).To(Equal(0), "No unenriched non-promoted entries should remain")
}

// TEST-210-4: Purge skips promoted entries even if unenriched
func TestOptimizePurgeSkipsPromotedUnenriched(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	claudeMDPath := filepath.Join(tempDir, "CLAUDE.md")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	// Seed DB with promoted but unenriched entry
	dbPath := filepath.Join(memoryRoot, "embeddings.db")
	db, err := sql.Open("sqlite3", dbPath)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	// Initialize the database schema
	_, err = memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())

	// Insert promoted but unenriched entry
	now := time.Now().Format(time.RFC3339)
	_, err = db.Exec(`
		INSERT INTO embeddings (
			content, source, source_type, confidence,
			observation_type, memory_type, principle, anti_pattern, rationale, enriched_content,
			retrieval_count, projects_retrieved, promoted, promoted_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "Promoted but unenriched observation", "memory", "internal", 1.0,
		"", "", "", "", "", "", // No enrichment
		10, "proj1,proj2,proj3", 1, now)
	g.Expect(err).ToNot(HaveOccurred())

	// Run optimize with purge
	_, err = memory.Optimize(memory.OptimizeOpts{
		MemoryRoot:   memoryRoot,
		ClaudeMDPath: claudeMDPath,
		AutoApprove:  true,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// VERIFY: Purge should not touch promoted entries
	var promotedCount int
	err = db.QueryRow("SELECT COUNT(*) FROM embeddings WHERE promoted = 1").Scan(&promotedCount)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(promotedCount).To(Equal(1), "Promoted entry should survive even if unenriched")

	// VERIFY: No purge happened (promoted entries excluded)
	// NOTE: UnenrichedPurged field will be added by implementation
	// g.Expect(result.UnenrichedPurged).To(Equal(0), "Should not purge promoted entries")
}

// TEST-210-5: Property test - entries with empty principle are never promoted
func TestOptimizePromotePropertyUnenrichedFiltered(t *testing.T) {
	// Property: For any entry with retrieval_count >= threshold and projects >= threshold,
	// if principle = '', it should NOT be promoted
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		tempDir := t.TempDir()
		memoryRoot := filepath.Join(tempDir, ".claude", "memory")
		claudeMDPath := filepath.Join(tempDir, "CLAUDE.md")
		g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())
		g.Expect(os.WriteFile(claudeMDPath, []byte("## Promoted Learnings\n\n"), 0644)).To(Succeed())

		dbPath := filepath.Join(memoryRoot, "embeddings.db")
		db, err := sql.Open("sqlite3", dbPath)
		g.Expect(err).ToNot(HaveOccurred())
		defer func() { _ = db.Close() }()

		_, err = memory.InitDBForTest(memoryRoot)
		g.Expect(err).ToNot(HaveOccurred())

		// Generate random retrieval count >= 5
		retrievalCount := rapid.IntRange(5, 20).Draw(rt, "retrievalCount")
		// Generate random projects (at least 3)
		numProjects := rapid.IntRange(3, 6).Draw(rt, "numProjects")
		projects := make([]string, numProjects)
		for i := 0; i < numProjects; i++ {
			projects[i] = fmt.Sprintf("proj%d", i+1)
		}
		projectsStr := fmt.Sprintf("%s,%s,%s", projects[0], projects[1], projects[2])

		// Insert unenriched entry with high stats
		_, err = db.Exec(`
			INSERT INTO embeddings (
				content, source, source_type, confidence,
				observation_type, memory_type, principle, anti_pattern, rationale, enriched_content,
				retrieval_count, projects_retrieved, promoted
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, "Random unenriched entry", "memory", "internal", 1.0,
			"", "", "", "", "", "", // principle = ''
			retrievalCount, projectsStr, 0)
		g.Expect(err).ToNot(HaveOccurred())

		// Run optimize
		_, err = memory.Optimize(memory.OptimizeOpts{
			MemoryRoot:    memoryRoot,
			ClaudeMDPath:  claudeMDPath,
			AutoApprove:   true,
			MinRetrievals: 5,
			MinProjects:   3,
		})
		g.Expect(err).ToNot(HaveOccurred())

		// VERIFY: Entry should NOT be promoted
		var promoted int
		err = db.QueryRow("SELECT promoted FROM embeddings WHERE content = 'Random unenriched entry'").Scan(&promoted)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(promoted).To(Equal(0), "Unenriched entry should never be promoted regardless of stats")

		// VERIFY: CLAUDE.md unchanged
		claudeContent, err := os.ReadFile(claudeMDPath)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(string(claudeContent)).To(Equal("## Promoted Learnings\n\n"))
	})
}

// TEST-210-6: Mixed batch - only enriched entries promoted
func TestOptimizePromoteMixedBatchOnlyEnriched(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	claudeMDPath := filepath.Join(tempDir, "CLAUDE.md")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())
	g.Expect(os.WriteFile(claudeMDPath, []byte("## Promoted Learnings\n\n"), 0644)).To(Succeed())

	dbPath := filepath.Join(memoryRoot, "embeddings.db")
	db, err := sql.Open("sqlite3", dbPath)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	_, err = memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())

	// Insert 3 unenriched candidates (high stats but no principle)
	for i := 1; i <= 3; i++ {
		_, err = db.Exec(`
			INSERT INTO embeddings (
				content, source, source_type, confidence,
				observation_type, memory_type, principle, anti_pattern, rationale, enriched_content,
				retrieval_count, projects_retrieved, promoted
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, fmt.Sprintf("Unenriched candidate %d", i), "memory", "internal", 1.0,
			"", "", "", "", "", "",
			10, "p1,p2,p3", 0)
		g.Expect(err).ToNot(HaveOccurred())
	}

	// Insert 2 enriched candidates (high stats AND principle)
	// Use semantically distinct principles to avoid deduplication
	principles := []string{
		"Always write tests before implementation to catch regressions early",
		"Use dependency injection to enable fast unit testing without IO",
	}
	for i := 0; i < 2; i++ {
		_, err = db.Exec(`
			INSERT INTO embeddings (
				content, source, source_type, confidence,
				observation_type, memory_type, principle, anti_pattern, rationale, enriched_content,
				retrieval_count, projects_retrieved, promoted
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, fmt.Sprintf("Enriched candidate %d", i+1), "memory", "internal", 1.0,
			"pattern", "reflection", principles[i], "", "", "Enriched",
			10, "p1,p2,p3", 0)
		g.Expect(err).ToNot(HaveOccurred())
	}

	// Run optimize
	result, err := memory.Optimize(memory.OptimizeOpts{
		MemoryRoot:    memoryRoot,
		ClaudeMDPath:  claudeMDPath,
		AutoApprove:   true,
		MinRetrievals: 5,
		MinProjects:   3,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// VERIFY: Found 5 candidates, but only 2 promoted
	g.Expect(result.PromotionCandidates).To(Equal(5), "Should find all 5 candidates")
	g.Expect(result.PromotionsApproved).To(Equal(2), "Should promote only the 2 enriched entries")

	// VERIFY: CLAUDE.md contains enriched principles
	claudeContent, err := os.ReadFile(claudeMDPath)
	g.Expect(err).ToNot(HaveOccurred())
	claudeStr := string(claudeContent)
	g.Expect(claudeStr).To(ContainSubstring("Always write tests before implementation"))
	g.Expect(claudeStr).To(ContainSubstring("Use dependency injection to enable fast unit testing"))
	g.Expect(claudeStr).ToNot(ContainSubstring("Unenriched candidate"))
}
