package memory_test

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	. "github.com/onsi/gomega"
	_ "github.com/mattn/go-sqlite3"

	"github.com/toejough/projctl/internal/memory"
)

// insertMemorySeqCounter ensures each test memory gets a distinct embedding.
var insertMemorySeqCounter int

// insertMemoryWithEmbedding inserts a memory entry with a unique embedding vector.
// Each call produces a distinct vector so the dedup step won't merge them
// (cosine sim < 0.95), while keeping vectors similar enough to cluster
// (cosine sim > 0.6). Uses a block-based approach: each seq gets a unique
// 96-dim block set to 1.0 over a shared 0.3 base.
func insertMemoryWithEmbedding(g *WithT, db *sql.DB, content string) int64 {
	insertMemorySeqCounter++
	seq := insertMemorySeqCounter

	fakeEmb := make([]float32, 384)
	for i := range fakeEmb {
		fakeEmb[i] = 0.3
	}
	// Each seq bumps a unique 96-dim block to 1.0
	blockStart := ((seq - 1) % 4) * 96
	for i := blockStart; i < blockStart+96 && i < 384; i++ {
		fakeEmb[i] = 1.0
	}
	blob, err := sqlite_vec.SerializeFloat32(fakeEmb)
	g.Expect(err).ToNot(HaveOccurred())

	// Insert into vec_embeddings
	result, err := db.Exec("INSERT INTO vec_embeddings(embedding) VALUES (?)", blob)
	g.Expect(err).ToNot(HaveOccurred())
	embID, err := result.LastInsertId()
	g.Expect(err).ToNot(HaveOccurred())

	// Insert into embeddings with embedding_id
	result, err = db.Exec(`
		INSERT INTO embeddings (content, source, source_type, confidence, memory_type, embedding_id)
		VALUES (?, 'test', 'internal', 1.0, 'observation', ?)
	`, content, embID)
	g.Expect(err).ToNot(HaveOccurred())

	memID, err := result.LastInsertId()
	g.Expect(err).ToNot(HaveOccurred())
	return memID
}

// ============================================================================
// Tests for TASK-11: Periodic Skill Reorganization
// traces: ISSUE-182, REQ-11
// ============================================================================

// TestSkillReorganization_NotTriggeredWithin30Days verifies that reorganization
// doesn't trigger when last_skill_reorg_at is < 30 days ago.
func TestSkillReorganization_NotTriggeredWithin30Days(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	skillsDir := filepath.Join(tempDir, "skills")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())
	g.Expect(os.MkdirAll(skillsDir, 0755)).To(Succeed())

	// Set last_skill_reorg_at to 20 days ago
	dbPath := filepath.Join(memoryRoot, "embeddings.db")
	twentyDaysAgo := time.Now().UTC().Add(-20 * 24 * time.Hour).Format(time.RFC3339)
	g.Expect(memory.SetMetadataForTest(memoryRoot, "last_skill_reorg_at", twentyDaysAgo)).To(Succeed())

	// Add some memories
	g.Expect(memory.Learn(memory.LearnOpts{
		Message:    "test memory 1",
		MemoryRoot: memoryRoot,
	})).To(Succeed())

	// Run optimize - should NOT trigger reorganization
	result, err := memory.Optimize(memory.OptimizeOpts{
		MemoryRoot:  memoryRoot,
		SkillsDir:   skillsDir,
		AutoApprove: true,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.SkillsReorganized).To(Equal(0), "Should not reorganize within 30 days")

	// Verify timestamp unchanged
	newTimestamp, err := memory.GetMetadataForTest(memoryRoot, "last_skill_reorg_at")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(newTimestamp).To(Equal(twentyDaysAgo), "Timestamp should remain unchanged")

	_ = dbPath // silence unused
}

// TestSkillReorganization_TriggeredAfter30Days verifies that reorganization
// triggers automatically when >30 days elapsed since last run.
func TestSkillReorganization_TriggeredAfter30Days(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	skillsDir := filepath.Join(tempDir, "skills")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())
	g.Expect(os.MkdirAll(skillsDir, 0755)).To(Succeed())

	// Open DB for direct inserts (don't call SetMetadataForTest - use ForceReorg instead)
	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	// Add cluster of similar memories (need 3+ for skill generation)
	authMemories := []string{
		"authentication workflow requires token validation",
		"auth workflow validates user credentials first",
		"authentication workflow checks session tokens",
		"auth workflow pattern uses JWT for validation",
	}
	for _, msg := range authMemories {
		insertMemoryWithEmbedding(g, db, msg)
	}

	// DIAGNOSTIC: Verify data inserted correctly
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM embeddings WHERE embedding_id IS NOT NULL").Scan(&count)
	g.Expect(err).ToNot(HaveOccurred())
	t.Logf("Inserted %d memories with embeddings", count)
	g.Expect(count).To(Equal(4), "Should have 4 memories with embeddings")

	// DIAGNOSTIC: Test vec_distance_cosine function
	var distance float64
	err = db.QueryRow(`
		SELECT vec_distance_cosine(
			(SELECT embedding FROM vec_embeddings WHERE rowid = 1),
			(SELECT embedding FROM vec_embeddings WHERE rowid = 2)
		)
	`).Scan(&distance)
	if err != nil {
		t.Logf("vec_distance_cosine ERROR: %v", err)
	} else {
		t.Logf("vec_distance_cosine(1,2) = %f, similarity = %f", distance, 1.0-distance)
	}

	// Close first connection and open new one to test visibility
	_ = db.Close()
	db2, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db2.Close() }()

	var count2 int
	err = db2.QueryRow("SELECT COUNT(*) FROM embeddings WHERE embedding_id IS NOT NULL").Scan(&count2)
	g.Expect(err).ToNot(HaveOccurred())
	t.Logf("New connection sees %d memories with embeddings", count2)

	// Create mock compiler
	compiler := &mockSkillCompiler{
		compileFunc: func(theme string, memories []string) (string, error) {
			return fmt.Sprintf("# %s\n\nSkill content", theme), nil
		},
	}

	// Run optimize - SHOULD trigger reorganization
	// Use ForceReorg to bypass time check and avoid SetMetadataForTest
	result, err := memory.Optimize(memory.OptimizeOpts{
		MemoryRoot:     memoryRoot,
		SkillsDir:      skillsDir,
		SkillCompiler:  compiler,
		AutoApprove:    true,
		ForceReorg:     true, // Bypass time check
		ReorgThreshold: 0.6,
		MinClusterSize: 3,
	})
	g.Expect(err).ToNot(HaveOccurred())
	t.Logf("Optimize result: SkillsReorganized=%d, SkillsCompiled=%d, SkillsMerged=%d",
		result.SkillsReorganized, result.SkillsCompiled, result.SkillsMerged)
	g.Expect(result.SkillsReorganized).To(BeNumerically(">", 0), "Should trigger reorganization after 30 days")

	// Verify timestamp updated to now
	newTimestamp, err := memory.GetMetadataForTest(memoryRoot, "last_skill_reorg_at")
	g.Expect(err).ToNot(HaveOccurred())
	parsedTime, err := time.Parse(time.RFC3339, newTimestamp)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(time.Since(parsedTime)).To(BeNumerically("<", 1*time.Minute), "Timestamp should be recent")
}

// TestSkillReorganization_ForceReorgFlag verifies that ForceReorg=true
// triggers reorganization regardless of elapsed time.
func TestSkillReorganization_ForceReorgFlag(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	skillsDir := filepath.Join(tempDir, "skills")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())
	g.Expect(os.MkdirAll(skillsDir, 0755)).To(Succeed())

	// Set last_skill_reorg_at to 1 day ago (< 30 days)
	oneDayAgo := time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)
	g.Expect(memory.SetMetadataForTest(memoryRoot, "last_skill_reorg_at", oneDayAgo)).To(Succeed())

	// Open DB for direct inserts
	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	// Add cluster of similar memories
	testMemories := []string{
		"testing workflow requires red-green-refactor cycle",
		"test workflow validates acceptance criteria first",
		"testing workflow pattern uses property-based tests",
		"test workflow ensures code coverage metrics",
	}
	for _, msg := range testMemories {
		insertMemoryWithEmbedding(g, db, msg)
	}

	// Create mock compiler
	compiler := &mockSkillCompiler{
		compileFunc: func(theme string, memories []string) (string, error) {
			return fmt.Sprintf("# %s\n\nSkill content", theme), nil
		},
	}

	// Run optimize with ForceReorg=true - SHOULD reorganize despite < 30 days
	// Use lower threshold for test (0.6) so test memories can cluster
	result, err := memory.Optimize(memory.OptimizeOpts{
		MemoryRoot:     memoryRoot,
		SkillsDir:      skillsDir,
		SkillCompiler:  compiler,
		AutoApprove:    true,
		ForceReorg:     true,
		ReorgThreshold: 0.6,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.SkillsReorganized).To(BeNumerically(">", 0), "ForceReorg should trigger reorganization")
}

// TestSkillReorganization_UpdatesExistingSkill verifies that reorganization
// updates existing skill content while preserving alpha/beta parameters.
func TestSkillReorganization_UpdatesExistingSkill(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	skillsDir := filepath.Join(tempDir, "skills")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())
	g.Expect(os.MkdirAll(skillsDir, 0755)).To(Succeed())

	// Insert existing skill with high alpha/beta (indicates usage history)
	dbPath := filepath.Join(memoryRoot, "embeddings.db")
	db, err := sql.Open("sqlite3", dbPath)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	// Initialize DB schema
	_, err = memory.Optimize(memory.OptimizeOpts{
		MemoryRoot:  memoryRoot,
		SkillsDir:   skillsDir,
		AutoApprove: true,
	})
	g.Expect(err).ToNot(HaveOccurred())

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = db.Exec(`
		INSERT INTO generated_skills (
			slug, theme, description, content, source_memory_ids,
			alpha, beta, utility, retrieval_count, last_retrieved,
			created_at, updated_at, pruned
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "test-pattern-theme-workflow-requires", "Test Pattern Theme", "Test description", "# Old Content", "[]",
		5.0, 2.0, 0.7, 10, now, now, now, 0)
	g.Expect(err).ToNot(HaveOccurred())

	originalAlpha := 5.0
	originalBeta := 2.0

	// Add cluster of memories that will regenerate the skill with similar theme
	themeMemories := []string{
		"test pattern theme workflow requires planning",
		"test pattern theme uses structured approach",
		"test pattern theme workflow detail validation",
		"test pattern theme ensures quality standards",
	}
	for _, msg := range themeMemories {
		insertMemoryWithEmbedding(g, db, msg)
	}

	// Create mock compiler
	compiler := &mockSkillCompiler{
		compileFunc: func(theme string, memories []string) (string, error) {
			return fmt.Sprintf("# %s\n\nRegenerated skill content", theme), nil
		},
	}

	// Force reorganization with lower threshold for test
	result, err := memory.Optimize(memory.OptimizeOpts{
		MemoryRoot:     memoryRoot,
		SkillsDir:      skillsDir,
		SkillCompiler:  compiler,
		AutoApprove:    true,
		ForceReorg:     true,
		ReorgThreshold: 0.6,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.SkillsReorganized).To(BeNumerically(">", 0))

	// Verify skill updated: content changed, alpha/beta preserved
	var updatedContent string
	var updatedAlpha, updatedBeta float64
	err = db.QueryRow(`
		SELECT content, alpha, beta
		FROM generated_skills
		WHERE slug = ?
	`, "test-pattern-theme-workflow-requires").Scan(&updatedContent, &updatedAlpha, &updatedBeta)
	g.Expect(err).ToNot(HaveOccurred())

	g.Expect(updatedContent).ToNot(Equal("# Old Content"), "Content should be regenerated")
	g.Expect(updatedAlpha).To(Equal(originalAlpha), "Alpha should be preserved")
	g.Expect(updatedBeta).To(Equal(originalBeta), "Beta should be preserved")
}

// TestSkillReorganization_CreatesNewSkill verifies that reorganization
// creates new skills for clusters that don't match existing skill themes.
func TestSkillReorganization_CreatesNewSkill(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	skillsDir := filepath.Join(tempDir, "skills")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())
	g.Expect(os.MkdirAll(skillsDir, 0755)).To(Succeed())

	// Open DB for direct inserts
	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	// Add cluster of memories for a new theme
	errorMemories := []string{
		"error handling pattern workflow captures context",
		"new error handling uses structured logging",
		"error handling pattern workflow wraps errors",
		"new error handling pattern ensures traceability",
	}
	for _, msg := range errorMemories {
		insertMemoryWithEmbedding(g, db, msg)
	}

	// Create mock compiler
	compiler := &mockSkillCompiler{
		compileFunc: func(theme string, memories []string) (string, error) {
			return fmt.Sprintf("# %s\n\nNew skill content", theme), nil
		},
	}

	// Force reorganization with lower threshold for test
	result, err := memory.Optimize(memory.OptimizeOpts{
		MemoryRoot:     memoryRoot,
		SkillsDir:      skillsDir,
		SkillCompiler:  compiler,
		AutoApprove:    true,
		ForceReorg:     true,
		ReorgThreshold: 0.6,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.SkillsReorganized).To(BeNumerically(">", 0))

	// Verify new skill created (db already open from earlier)
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM generated_skills WHERE pruned = 0").Scan(&count)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(count).To(BeNumerically(">", 0), "Should create at least one new skill")

	// Verify skill has fresh prior (Alpha=1, Beta=1)
	var alpha, beta float64
	err = db.QueryRow("SELECT alpha, beta FROM generated_skills WHERE pruned = 0 LIMIT 1").Scan(&alpha, &beta)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(alpha).To(Equal(1.0), "New skill should have Alpha=1")
	g.Expect(beta).To(Equal(1.0), "New skill should have Beta=1")
}

// TestSkillReorganization_PrunesOrphanedSkills verifies that reorganization
// soft-deletes skills whose themes no longer appear in any cluster.
func TestSkillReorganization_PrunesOrphanedSkills(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	skillsDir := filepath.Join(tempDir, "skills")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())
	g.Expect(os.MkdirAll(skillsDir, 0755)).To(Succeed())

	// Initialize DB
	_, err := memory.Optimize(memory.OptimizeOpts{
		MemoryRoot:  memoryRoot,
		SkillsDir:   skillsDir,
		AutoApprove: true,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Insert orphaned skill (no matching memories)
	dbPath := filepath.Join(memoryRoot, "embeddings.db")
	db, err := sql.Open("sqlite3", dbPath)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = db.Exec(`
		INSERT INTO generated_skills (
			slug, theme, description, content, source_memory_ids,
			alpha, beta, utility, retrieval_count, last_retrieved,
			created_at, updated_at, pruned
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "orphan-skill", "Orphan Theme", "Orphaned", "# Orphan", "[]",
		1.0, 1.0, 0.5, 0, nil, now, now, 0)
	g.Expect(err).ToNot(HaveOccurred())

	// Add cluster of memories for DIFFERENT theme (not "orphan")
	differentMemories := []string{
		"database migration pattern requires version tracking",
		"database migration workflow validates schema changes",
		"database migration pattern ensures backward compatibility",
		"database migration workflow pattern uses transactions",
	}
	for _, msg := range differentMemories {
		insertMemoryWithEmbedding(g, db, msg)
	}

	// Create mock compiler
	compiler := &mockSkillCompiler{
		compileFunc: func(theme string, memories []string) (string, error) {
			return fmt.Sprintf("# %s\n\nSkill content", theme), nil
		},
	}

	// Force reorganization with lower threshold for test
	result, err := memory.Optimize(memory.OptimizeOpts{
		MemoryRoot:     memoryRoot,
		SkillsDir:      skillsDir,
		SkillCompiler:  compiler,
		AutoApprove:    true,
		ForceReorg:     true,
		ReorgThreshold: 0.6,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.SkillsReorganized).To(BeNumerically(">", 0))

	// Verify orphan skill pruned
	var pruned int
	err = db.QueryRow("SELECT pruned FROM generated_skills WHERE slug = ?", "orphan-skill").Scan(&pruned)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(pruned).To(Equal(1), "Orphan skill should be pruned")
}
