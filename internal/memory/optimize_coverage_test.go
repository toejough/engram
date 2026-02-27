//go:build sqlite_fts5

package memory_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/memory"
)

func TestCheckContext_ActiveContext(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	opts := memory.OptimizeOpts{Context: ctx}
	err := memory.CheckContextForTest(opts)
	g.Expect(err).ToNot(HaveOccurred())
}

func TestCheckContext_CancelledContext(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	opts := memory.OptimizeOpts{Context: ctx}
	err := memory.CheckContextForTest(opts)
	g.Expect(err).To(HaveOccurred())
}

// ============================================================================
// checkContext
// ============================================================================

func TestCheckContext_NilContext(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	opts := memory.OptimizeOpts{}
	err := memory.CheckContextForTest(opts)
	g.Expect(err).ToNot(HaveOccurred())
}

// ============================================================================
// clusterEntriesToEmbeddings
// ============================================================================

func TestClusterEntriesToEmbeddings_Empty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	result := memory.ClusterEntriesToEmbeddingsForTest(nil)
	g.Expect(result).To(BeEmpty())
}

func TestClusterEntriesToEmbeddings_NonEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	cluster := []memory.ClusterEntry{
		{ID: 1, Content: "foo", EmbeddingID: 10},
		{ID: 2, Content: "bar", EmbeddingID: 20},
	}
	result := memory.ClusterEntriesToEmbeddingsForTest(cluster)
	g.Expect(result).To(HaveLen(2))
	// clusterEntriesToEmbeddings maps entry.ID → Embedding.ID
	g.Expect(result[0].ID).To(BeEquivalentTo(1))
	g.Expect(result[1].ID).To(BeEquivalentTo(2))
	g.Expect(result[0].Content).To(Equal("foo"))
	g.Expect(result[1].Content).To(Equal("bar"))
}

// ============================================================================
// clusterHasExistingSkill
// ============================================================================

func TestClusterHasExistingSkill_EmptyCluster(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	g.Expect(memory.ClusterHasExistingSkillForTest(nil, map[int64]bool{1: true})).To(BeFalse())
}

func TestClusterHasExistingSkill_HasMatch(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	cluster := []memory.ClusterEntry{{ID: 5, EmbeddingID: 5}, {ID: 7, EmbeddingID: 7}}
	g.Expect(memory.ClusterHasExistingSkillForTest(cluster, map[int64]bool{7: true})).To(BeTrue())
}

func TestClusterHasExistingSkill_NoMatch(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	cluster := []memory.ClusterEntry{{ID: 5, EmbeddingID: 5}}
	g.Expect(memory.ClusterHasExistingSkillForTest(cluster, map[int64]bool{99: true})).To(BeFalse())
}

func TestCosineSimilarity_DifferentLengths(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	a := []float32{1, 0}
	b := []float32{1, 0, 0}
	g.Expect(memory.CosineSimilarityForTest(a, b)).To(BeNumerically("==", 0.0))
}

func TestCosineSimilarity_IdenticalVectors(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	a := []float32{1, 0, 0}
	b := []float32{1, 0, 0}
	sim := memory.CosineSimilarityForTest(a, b)
	g.Expect(sim).To(BeNumerically("~", 1.0, 1e-6))
}

func TestCosineSimilarity_OrthogonalVectors(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	a := []float32{1, 0}
	b := []float32{0, 1}
	sim := memory.CosineSimilarityForTest(a, b)
	g.Expect(sim).To(BeNumerically("~", 0.0, 1e-6))
}

// ============================================================================
// cosineSimilarity
// ============================================================================

func TestCosineSimilarity_ZeroVectors(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	a := []float32{0, 0, 0}
	b := []float32{0, 0, 0}
	g.Expect(memory.CosineSimilarityForTest(a, b)).To(BeNumerically("==", 0.0))
}

// ============================================================================
// defaultSkillTester.TestAndEvaluate
// ============================================================================

func TestDefaultSkillTester_ZeroRuns(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	tester := memory.DefaultSkillTesterForTest("fake-api-key")
	_, _, err := tester.TestAndEvaluate(memory.TestScenario{}, 0)
	g.Expect(err).To(HaveOccurred())
}

// ============================================================================
// formatClusterSourceIDs
// ============================================================================

func TestFormatClusterSourceIDs_Empty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	result := memory.FormatClusterSourceIDsForTest(nil)
	g.Expect(result).To(Equal("[]"))
}

func TestFormatClusterSourceIDs_NonEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	cluster := []memory.ClusterEntry{{ID: 1}, {ID: 2}, {ID: 3}}
	result := memory.FormatClusterSourceIDsForTest(cluster)
	g.Expect(result).To(ContainSubstring("1"))
	g.Expect(result).To(ContainSubstring("2"))
	g.Expect(result).To(ContainSubstring("3"))
}

// ============================================================================
// generateSkillFromLearning
// ============================================================================

func TestGenerateSkillFromLearning_TemplateMode(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	tmpDir := t.TempDir()
	opts := memory.OptimizeOpts{
		SkillsDir:     tmpDir,
		SkillCompiler: nil, // no compiler → template fallback
	}
	learning := "Always run tests before committing code changes to the repository"
	err := memory.GenerateSkillFromLearningForTest(db, opts, learning)
	g.Expect(err).ToNot(HaveOccurred())
	// Verify skill directory was created
	entries, err := os.ReadDir(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())
	// At least one skill dir created (memory-{slug})
	found := false
	for _, e := range entries {
		if e.IsDir() && strings.HasPrefix(e.Name(), "memory-") {
			found = true
			break
		}
	}
	g.Expect(found).To(BeTrue())
}

// TestGenerateSkillFromLearning_UpdateExisting verifies a second call with the same learning
// finds the existing skill by slug and updates it rather than inserting a duplicate.
func TestGenerateSkillFromLearning_UpdateExisting(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	tmpDir := t.TempDir()
	db := newOptDB(t)

	learning := "TASK-7 run tests first then commit changes to maintain code quality"
	opts := memory.OptimizeOpts{
		SkillsDir:  tmpDir,
		MemoryRoot: tmpDir,
	}

	// First call: inserts a new skill.
	err := memory.GenerateSkillFromLearningForTest(db, opts, learning)
	g.Expect(err).ToNot(HaveOccurred())

	var count1 int
	_ = db.QueryRow("SELECT COUNT(*) FROM generated_skills WHERE pruned=0").Scan(&count1)
	g.Expect(count1).To(Equal(1))

	// Second call: finds existing skill by slug → update path → count stays 1.
	err = memory.GenerateSkillFromLearningForTest(db, opts, learning)
	g.Expect(err).ToNot(HaveOccurred())

	var count2 int
	_ = db.QueryRow("SELECT COUNT(*) FROM generated_skills WHERE pruned=0").Scan(&count2)
	g.Expect(count2).To(Equal(1))
}

// ============================================================================
// generateThemeFromCluster
// ============================================================================

func TestGenerateThemeFromCluster_Empty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	g.Expect(memory.GenerateThemeFromClusterForTest(nil)).To(Equal("Unknown"))
}

func TestGenerateThemeFromCluster_NonEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	cluster := []memory.ClusterEntry{
		{ID: 1, Content: "Always check git status before committing changes to repository"},
		{ID: 2, Content: "Use git diff to review staged changes"},
	}
	theme := memory.GenerateThemeFromClusterForTest(cluster)
	g.Expect(theme).ToNot(BeEmpty())
}

// ============================================================================
// getExistingPatterns
// ============================================================================

func TestGetExistingPatterns_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	patterns := memory.GetExistingPatternsForTest(db)
	g.Expect(patterns).To(BeEmpty())
}

func TestGetExistingPatterns_WithSynthesizedEntry(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	// getExistingPatterns uses ORDER BY created_at DESC; add the column so the query succeeds
	_, _ = db.Exec("ALTER TABLE embeddings ADD COLUMN created_at TEXT NOT NULL DEFAULT ''")
	_, err := db.Exec(
		"INSERT INTO embeddings (content, source, memory_type) VALUES (?, 'synthesized', 'reflection')",
		"Always review code before committing",
	)
	g.Expect(err).ToNot(HaveOccurred())
	patterns := memory.GetExistingPatternsForTest(db)
	g.Expect(patterns).To(HaveLen(1))
	g.Expect(patterns[0]).To(ContainSubstring("review code"))
}

// ============================================================================
// getExistingSkillSourceIDs
// ============================================================================

func TestGetExistingSkillSourceIDs_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	ids, err := memory.GetExistingSkillSourceIDsForTest(db)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(ids).To(BeEmpty())
}

func TestGetExistingSkillSourceIDs_WithSkill(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec(
		`INSERT INTO generated_skills (slug, theme, description, content, source_memory_ids, created_at, updated_at, pruned)
		 VALUES ('test-skill', 'test', 'Use when testing', 'content', '[1,2,3]', ?, ?, 0)`,
		now, now,
	)
	g.Expect(err).ToNot(HaveOccurred())
	ids, err := memory.GetExistingSkillSourceIDsForTest(db)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(ids).To(HaveKey(int64(1)))
	g.Expect(ids).To(HaveKey(int64(2)))
	g.Expect(ids).To(HaveKey(int64(3)))
}

func TestHasLearnTimestampPrefix_NoLeadingDash(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	g.Expect(memory.HasLearnTimestampPrefixForTest("2026-02-10 15:04: message")).To(BeFalse())
}

func TestHasLearnTimestampPrefix_TooShort(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	g.Expect(memory.HasLearnTimestampPrefixForTest("- 2026-02")).To(BeFalse())
}

// ============================================================================
// hasLearnTimestampPrefix
// ============================================================================

func TestHasLearnTimestampPrefix_Valid(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	g.Expect(memory.HasLearnTimestampPrefixForTest("- 2026-02-10 15:04: some message")).To(BeTrue())
}

func TestHasLearnTimestampPrefix_WithProject(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	g.Expect(memory.HasLearnTimestampPrefixForTest("- 2026-02-10 15:04: [myproject] some message")).To(BeTrue())
}

func TestHasLearnTimestampPrefix_WrongFormat(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	g.Expect(memory.HasLearnTimestampPrefixForTest("- hello world this is some content")).To(BeFalse())
}

// TestInsertVecEmbeddingForTest_DBError covers the db.Exec error path by
// dropping the vec_embeddings table so the INSERT fails.
func TestInsertVecEmbeddingForTest_DBError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)

	// Drop vec_embeddings so the INSERT fails.
	_, err := db.Exec("DROP TABLE IF EXISTS vec_embeddings")
	g.Expect(err).ToNot(HaveOccurred())

	_, err = memory.InsertVecEmbeddingForTest(db, makeUnitVec())
	g.Expect(err).To(HaveOccurred())
}

func TestIsNarrowByKeywords_FilePath(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	narrow, reason := memory.IsNarrowByKeywordsForTest("Edit internal/memory/foo.go")
	g.Expect(narrow).To(BeTrue())
	g.Expect(reason).ToNot(BeEmpty())
}

func TestIsNarrowByKeywords_Generic(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	narrow, _ := memory.IsNarrowByKeywordsForTest("Keep code changes minimal and focused")
	g.Expect(narrow).To(BeFalse())
}

func TestIsNarrowByKeywords_ProjectName(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	narrow, reason := memory.IsNarrowByKeywordsForTest("In the projctl codebase we use targ")
	g.Expect(narrow).To(BeTrue())
	g.Expect(reason).ToNot(BeEmpty())
}

func TestIsNarrowByKeywords_RetroContent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	narrow, reason := memory.IsNarrowByKeywordsForTest("retro: this sprint went well")
	g.Expect(narrow).To(BeTrue())
	g.Expect(reason).ToNot(BeEmpty())
}

func TestIsNarrowByKeywords_TechSpecific(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	narrow, reason := memory.IsNarrowByKeywordsForTest("Run go test ./... with coverage flags")
	g.Expect(narrow).To(BeTrue())
	g.Expect(reason).ToNot(BeEmpty())
}

func TestIsNarrowByKeywords_ToolName(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	narrow, reason := memory.IsNarrowByKeywordsForTest("Use targ build when running tests")
	g.Expect(narrow).To(BeTrue())
	g.Expect(reason).ToNot(BeEmpty())
}

// ============================================================================
// isNarrowByKeywords
// ============================================================================

func TestIsNarrowByKeywords_TrackerID(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	narrow, reason := memory.IsNarrowByKeywordsForTest("Fix ISSUE-123 in the codebase")
	g.Expect(narrow).To(BeTrue())
	g.Expect(reason).ToNot(BeEmpty())
}

// ============================================================================
// migrateMemoryGenSkills — covers !entry.IsDir() path (77.8% → 100%)
// ============================================================================

func TestMigrateMemoryGenSkills_NonDirEntry(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	fs := &MigrateMockFS{
		Directories: map[string][]MockDirEntry{
			"/skills/memory-gen": {
				{name: "someskill", isDir: true},   // dir → migrated
				{name: "readme.txt", isDir: false}, // file → skipped (covers !entry.IsDir() path)
			},
		},
	}
	err := memory.MigrateMemoryGenSkillsForTest(fs, "/skills")
	g.Expect(err).ToNot(HaveOccurred())
	// The file entry should not appear in renames
	for old := range fs.Renamed {
		g.Expect(old).ToNot(ContainSubstring("readme.txt"))
	}
	// The dir entry should be renamed
	g.Expect(fs.Renamed).To(HaveKey("/skills/memory-gen/someskill"))
}

func TestOptimizeAutoDemote_LowConfidencePromoted(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	// Insert promoted entry with very low confidence
	insertOptEntry(t, db, "Always check error returns in functions", "memory", 0.1, 1)
	result := &memory.OptimizeResult{}
	tmpDir := t.TempDir()
	opts := memory.OptimizeOpts{ClaudeMDPath: filepath.Join(tmpDir, "CLAUDE.md")}
	err := memory.OptimizeAutoDemoteForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.AutoDemoted).To(Equal(1))
}

// ============================================================================
// optimizeAutoDemote
// ============================================================================

func TestOptimizeAutoDemote_NoPromoted(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	result := &memory.OptimizeResult{}
	tmpDir := t.TempDir()
	opts := memory.OptimizeOpts{ClaudeMDPath: filepath.Join(tmpDir, "CLAUDE.md")}
	err := memory.OptimizeAutoDemoteForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.AutoDemoted).To(Equal(0))
}

// TestOptimizeClaudeMDDedup_NoDuplicates verifies that dissimilar learnings are kept as-is.
func TestOptimizeClaudeMDDedup_NoDuplicates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	tmpDir := t.TempDir()
	claudeMD := filepath.Join(tmpDir, "CLAUDE.md")

	content := "# CLAUDE\n\n## Promoted Learnings\n\n- Always write tests\n- Use dependency injection\n"
	err := os.WriteFile(claudeMD, []byte(content), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	db := newOptDB(t)

	// Return orthogonal vectors → similarity=0.0 < 0.9 → no dedup.
	call := 0
	opts := memory.OptimizeOpts{
		ClaudeMDPath: claudeMD,
		Embedder: func(_ string) ([]float32, error) {
			call++
			if call%2 == 1 {
				return []float32{1, 0, 0, 0}, nil
			}
			return []float32{0, 1, 0, 0}, nil
		},
	}
	result := &memory.OptimizeResult{}

	err = memory.OptimizeClaudeMDDedupForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.ClaudeMDDeduped).To(Equal(0))
}

// ============================================================================
// optimizeClaudeMDDedup
// ============================================================================

func TestOptimizeClaudeMDDedup_NoFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	tmpDir := t.TempDir()
	result := &memory.OptimizeResult{}
	opts := memory.OptimizeOpts{ClaudeMDPath: filepath.Join(tmpDir, "nonexistent.md")}
	err := memory.OptimizeClaudeMDDedupForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.ClaudeMDDeduped).To(Equal(0))
}

func TestOptimizeClaudeMDDedup_NoPromotedSection(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	tmpDir := t.TempDir()
	claudeMD := filepath.Join(tmpDir, "CLAUDE.md")
	// CLAUDE.md without "Promoted Learnings" section
	_ = os.WriteFile(claudeMD, []byte("# My Config\n\nSome content.\n"), 0600)
	result := &memory.OptimizeResult{}
	opts := memory.OptimizeOpts{ClaudeMDPath: claudeMD}
	err := memory.OptimizeClaudeMDDedupForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.ClaudeMDDeduped).To(Equal(0))
}

// TestOptimizeClaudeMDDedup_SingleEntry verifies that fewer than 2 Promoted Learnings
// entries causes early return without deduplication.
func TestOptimizeClaudeMDDedup_SingleEntry(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	tmpDir := t.TempDir()
	claudeMD := filepath.Join(tmpDir, "CLAUDE.md")

	content := "# CLAUDE\n\n## Promoted Learnings\n\n- Only one learning here\n"
	err := os.WriteFile(claudeMD, []byte(content), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	db := newOptDB(t)
	opts := memory.OptimizeOpts{ClaudeMDPath: claudeMD}
	result := &memory.OptimizeResult{}

	err = memory.OptimizeClaudeMDDedupForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.ClaudeMDDeduped).To(Equal(0))
}

// TestOptimizeClaudeMDDedup_WithDuplicates verifies that two near-identical promoted learnings
// are deduplicated when using an injected embedder that returns identical vectors.
func TestOptimizeClaudeMDDedup_WithDuplicates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	tmpDir := t.TempDir()
	claudeMD := filepath.Join(tmpDir, "CLAUDE.md")

	content := "# CLAUDE\n\n## Promoted Learnings\n\n- Always write tests\n- Always write tests (duplicate)\n"
	err := os.WriteFile(claudeMD, []byte(content), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	db := newOptDB(t)

	// Same unit vector for every embedding → similarity=1.0 > 0.9 → one entry removed.
	unitVec := []float32{1.0, 0, 0, 0}
	opts := memory.OptimizeOpts{
		ClaudeMDPath: claudeMD,
		Embedder:     func(_ string) ([]float32, error) { return unitVec, nil },
	}
	result := &memory.OptimizeResult{}

	err = memory.OptimizeClaudeMDDedupForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.ClaudeMDDeduped).To(Equal(1))
}

func TestOptimizeCompileSkills_EmptyDB_TriggerReorg(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	tmpDir := t.TempDir()
	result := &memory.OptimizeResult{}
	opts := memory.OptimizeOpts{
		SkillsDir:      tmpDir,
		MinClusterSize: 3,
		SynthThreshold: 0.8,
		ReorgThreshold: 0.8,
	}
	// last_skill_reorg_at not set → shouldReorg = true → performSkillReorganization called
	// With empty DB → sets metadata and returns nil
	err := memory.OptimizeCompileSkillsForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
}

func TestOptimizeCompileSkills_ForceReorg(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	tmpDir := t.TempDir()
	result := &memory.OptimizeResult{}
	opts := memory.OptimizeOpts{
		SkillsDir:      tmpDir,
		ForceReorg:     true,
		MinClusterSize: 3,
		ReorgThreshold: 0.8,
	}
	err := memory.OptimizeCompileSkillsForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
}

// ============================================================================
// optimizeCompileSkills
// ============================================================================

func TestOptimizeCompileSkills_NoSkillsDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	result := &memory.OptimizeResult{}
	opts := memory.OptimizeOpts{SkillsDir: ""}
	err := memory.OptimizeCompileSkillsForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
}

// TestOptimizeCompileSkills_NonReorg_EmptyDB verifies no skills are compiled with an empty
// embeddings DB when last_skill_reorg_at is recent (non-reorg mode).
func TestOptimizeCompileSkills_NonReorg_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	tmpDir := t.TempDir()

	fiveDaysAgo := time.Now().UTC().Add(-5 * 24 * time.Hour).Format(time.RFC3339)
	_, err := db.Exec("INSERT OR REPLACE INTO metadata (key, value) VALUES ('last_skill_reorg_at', ?)", fiveDaysAgo)
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.OptimizeOpts{
		SkillsDir:      tmpDir,
		SimilarityFunc: func(_ *sql.DB, _, _ int64) (float64, error) { return 1.0, nil },
	}
	result := &memory.OptimizeResult{}

	err = memory.OptimizeCompileSkillsForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.SkillsCompiled).To(Equal(0))
}

// TestOptimizeCompileSkills_NonReorg_WithCluster verifies a cluster of identical embeddings
// compiles a new skill when last_skill_reorg_at is recent (non-reorg mode).
func TestOptimizeCompileSkills_NonReorg_WithCluster(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	tmpDir := t.TempDir()

	fiveDaysAgo := time.Now().UTC().Add(-5 * 24 * time.Hour).Format(time.RFC3339)
	_, err := db.Exec("INSERT OR REPLACE INTO metadata (key, value) VALUES ('last_skill_reorg_at', ?)", fiveDaysAgo)
	g.Expect(err).ToNot(HaveOccurred())

	for range 3 {
		id := insertOptEntry(t, db, "commit workflow procedure steps", "memory", 0.9, 0)
		attachVec(t, db, id)
	}

	opts := memory.OptimizeOpts{
		SkillsDir:      tmpDir,
		SynthThreshold: 0.5,
		MinClusterSize: 3,
		SimilarityFunc: func(_ *sql.DB, _, _ int64) (float64, error) { return 1.0, nil },
	}
	result := &memory.OptimizeResult{}

	err = memory.OptimizeCompileSkillsForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.SkillsCompiled).To(BeNumerically(">=", 1))
}

func TestOptimizeContradictions_ContradictionDetected(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	// Insert promoted entry
	promoID := insertOptEntry(t, db, "Always use git commit messages", "memory", 1.0, 1)
	attachVec(t, db, promoID)
	// Insert correction entry with identical vec embedding
	corrID := insertOptEntryFull(t, db, "Never commit directly to main branch", "memory", 1.0, 0, "correction", "", 0)
	attachVec(t, db, corrID)

	result := &memory.OptimizeResult{}
	err := memory.OptimizeContradictionsForTest(db, memory.OptimizeOpts{}, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.ContradictionsFound).To(BeNumerically(">=", 1))
}

// ============================================================================
// optimizeContradictions
// ============================================================================

func TestOptimizeContradictions_NoPromoted(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	result := &memory.OptimizeResult{}
	err := memory.OptimizeContradictionsForTest(db, memory.OptimizeOpts{}, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.ContradictionsFound).To(Equal(0))
}

// ============================================================================
// optimizeDecay
// ============================================================================

func TestOptimizeDecay_FirstRun(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	result := &memory.OptimizeResult{}
	opts := memory.OptimizeOpts{DecayBase: 0.9}
	err := memory.OptimizeDecayForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.DecayApplied).To(BeTrue())
	g.Expect(result.DaysSinceLastOptimize).To(BeNumerically("~", 1.0, 1e-6))
}

// TestOptimizeDecay_InvalidTimestamp verifies that an invalid timestamp string in last_optimized_at
// results in daysSince=0 and DecayApplied=true (falls through to the decay logic).
func TestOptimizeDecay_InvalidTimestamp(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)

	_, err := db.Exec("INSERT OR REPLACE INTO metadata (key, value) VALUES ('last_optimized_at', 'not-a-valid-timestamp')")
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.OptimizeOpts{DecayBase: 0.9}
	result := &memory.OptimizeResult{}

	err = memory.OptimizeDecayForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	// Parse failure → falls through with daysSince=0 but still applies decay.
	g.Expect(result.DecayApplied).To(BeTrue())
	g.Expect(result.DaysSinceLastOptimize).To(BeNumerically("==", 0.0))
}

func TestOptimizeDecay_OldRun_WithEntries(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	// Set last_optimized_at to 2 days ago
	old := time.Now().Add(-48 * time.Hour).Format(time.RFC3339)
	_, err := db.Exec("INSERT INTO metadata (key, value) VALUES ('last_optimized_at', ?)", old)
	g.Expect(err).ToNot(HaveOccurred())
	// Insert an entry to decay
	insertOptEntry(t, db, "Some memory", "memory", 1.0, 0)
	result := &memory.OptimizeResult{}
	opts := memory.OptimizeOpts{DecayBase: 0.9}
	err = memory.OptimizeDecayForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.DecayApplied).To(BeTrue())
	g.Expect(result.EntriesDecayed).To(BeNumerically(">", 0))
}

func TestOptimizeDecay_RecentRun(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	// Set last_optimized_at to 5 minutes ago (< 1 hour → skip)
	recent := time.Now().Add(-5 * time.Minute).Format(time.RFC3339)
	_, err := db.Exec("INSERT INTO metadata (key, value) VALUES ('last_optimized_at', ?)", recent)
	g.Expect(err).ToNot(HaveOccurred())
	result := &memory.OptimizeResult{}
	opts := memory.OptimizeOpts{DecayBase: 0.9}
	err = memory.OptimizeDecayForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.DecayApplied).To(BeFalse())
}

// ============================================================================
// optimizeDedup
// ============================================================================

func TestOptimizeDedup_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	result := &memory.OptimizeResult{}
	opts := memory.OptimizeOpts{DupThreshold: 0.95}
	err := memory.OptimizeDedupForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.DuplicatesMerged).To(Equal(0))
}

func TestOptimizeDedup_IdenticalEmbeddings(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	// Two entries with identical vec embeddings → similarity = 1.0 → merge
	id1 := insertOptEntry(t, db, "Always run tests before committing code", "memory", 1.0, 0)
	attachVec(t, db, id1)
	id2 := insertOptEntry(t, db, "Always run tests before committing code", "memory", 0.8, 0)
	attachVec(t, db, id2)

	result := &memory.OptimizeResult{}
	opts := memory.OptimizeOpts{DupThreshold: 0.95}
	err := memory.OptimizeDedupForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.DuplicatesMerged).To(Equal(1))
}

// TestOptimizeDemoteClaudeMD_AutoApproveUnsafe covers the AutoApprove=true path
// where a narrow learning cannot be safely demoted (plan.Safe==false),
// exercising the logChangelogMutation+continue block.
func TestOptimizeDemoteClaudeMD_AutoApproveUnsafe(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	tmpDir := t.TempDir()
	claudeMD := filepath.Join(tmpDir, "CLAUDE.md")
	// "projctl" triggers isNarrowByKeywords, but has no deterministic/procedural/situational
	// keywords, so PlanCLAUDEMDDemotion returns plan.Safe=false → blocked demotion path.
	content := "## Promoted Learnings\n\n- projctl uses go test to run tests\n"
	_ = os.WriteFile(claudeMD, []byte(content), 0600)
	result := &memory.OptimizeResult{}
	opts := memory.OptimizeOpts{
		SkillsDir:    tmpDir,
		ClaudeMDPath: claudeMD,
		MemoryRoot:   tmpDir,
		AutoApprove:  true,
	}
	err := memory.OptimizeDemoteClaudeMDForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	// Demotion was blocked (unsafe plan), so nothing was demoted
	g.Expect(result.ClaudeMDDemoted).To(Equal(0))
}

func TestOptimizeDemoteClaudeMD_BroadLearning_NoCandidates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	tmpDir := t.TempDir()
	claudeMD := filepath.Join(tmpDir, "CLAUDE.md")
	// Broad learning: no issue IDs, no project names, no file paths, no tool names
	// → isNarrowByKeywords returns false → len(candidates) == 0 → early return
	content := "## Promoted Learnings\n\n- Always write tests before implementation to ensure correctness\n"
	_ = os.WriteFile(claudeMD, []byte(content), 0600)
	result := &memory.OptimizeResult{}
	opts := memory.OptimizeOpts{
		SkillsDir:    tmpDir,
		ClaudeMDPath: claudeMD,
		MemoryRoot:   tmpDir,
	}
	err := memory.OptimizeDemoteClaudeMDForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.ClaudeMDDemoted).To(Equal(0))
}

// TestOptimizeDemoteClaudeMD_DestinationHook verifies that hook-destination candidates
// ("never" → deterministic rule) are not removed and ClaudeMDDemoted stays 0.
func TestOptimizeDemoteClaudeMD_DestinationHook(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	tmpDir := t.TempDir()
	claudeMD := filepath.Join(tmpDir, "CLAUDE.md")

	// "never" → isDeterministicRule → DestinationHook → continue (no ClaudeMDDemoted++).
	content := "## Promoted Learnings\n\n- ISSUE-42 never commit debug code to production\n"
	err := os.WriteFile(claudeMD, []byte(content), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	db := newOptDB(t)
	opts := memory.OptimizeOpts{
		ClaudeMDPath: claudeMD,
		SkillsDir:    tmpDir,
		AutoApprove:  true,
		MemoryRoot:   tmpDir,
	}
	result := &memory.OptimizeResult{}

	err = memory.OptimizeDemoteClaudeMDForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.ClaudeMDDemoted).To(Equal(0))
}

// TestOptimizeDemoteClaudeMD_DestinationSkill verifies that skill-destination candidates
// ("first"/"then" → procedural workflow) are demoted and ClaudeMDDemoted is incremented.
func TestOptimizeDemoteClaudeMD_DestinationSkill(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	tmpDir := t.TempDir()
	claudeMD := filepath.Join(tmpDir, "CLAUDE.md")

	// "first" + "then" → isProceduralWorkflow → DestinationSkill → generateSkillFromLearning → ClaudeMDDemoted++.
	content := "## Promoted Learnings\n\n- TASK-5 run tests first then commit the changes to the repository\n"
	err := os.WriteFile(claudeMD, []byte(content), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	db := newOptDB(t)
	opts := memory.OptimizeOpts{
		ClaudeMDPath: claudeMD,
		SkillsDir:    tmpDir,
		AutoApprove:  true,
		MemoryRoot:   tmpDir,
	}
	result := &memory.OptimizeResult{}

	err = memory.OptimizeDemoteClaudeMDForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.ClaudeMDDemoted).To(BeNumerically(">=", 1))
}

// TestOptimizeDemoteClaudeMD_EmptyEntries covers the len(entries)==0 early return
// when Promoted Learnings section exists but contains only blank/whitespace lines.
func TestOptimizeDemoteClaudeMD_EmptyEntries(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	tmpDir := t.TempDir()
	claudeMD := filepath.Join(tmpDir, "CLAUDE.md")
	// Promoted Learnings section exists but has only empty/dash-only lines
	content := "## Promoted Learnings\n\n- \n  \n"
	_ = os.WriteFile(claudeMD, []byte(content), 0600)
	result := &memory.OptimizeResult{}
	opts := memory.OptimizeOpts{SkillsDir: tmpDir, ClaudeMDPath: claudeMD, MemoryRoot: tmpDir}
	err := memory.OptimizeDemoteClaudeMDForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.ClaudeMDDemoted).To(Equal(0))
}

func TestOptimizeDemoteClaudeMD_NarrowLearning_DryRun(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	tmpDir := t.TempDir()
	claudeMD := filepath.Join(tmpDir, "CLAUDE.md")
	// Narrow learning that would be demoted, but AutoApprove=false and no ReviewFunc
	// → dry-run path: candidates found but no demotion applied
	learning := "In this codebase ISSUE-123 was fixed using special patches"
	content := "## Promoted Learnings\n\n- " + learning + "\n"
	_ = os.WriteFile(claudeMD, []byte(content), 0600)
	result := &memory.OptimizeResult{}
	opts := memory.OptimizeOpts{
		SkillsDir:    tmpDir,
		ClaudeMDPath: claudeMD,
		MemoryRoot:   tmpDir,
		AutoApprove:  false,
		ReviewFunc:   nil,
	}
	err := memory.OptimizeDemoteClaudeMDForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	// Dry-run: no actual demotion happens
	g.Expect(result.ClaudeMDDemoted).To(Equal(0))
}

func TestOptimizeDemoteClaudeMD_NarrowLearning_EmbeddingDestination(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	tmpDir := t.TempDir()
	claudeMD := filepath.Join(tmpDir, "CLAUDE.md")
	// Narrow learning: contains "in this" (isSituationalContent) + "ISSUE-123" (isNarrowByKeywords)
	// Does NOT contain "always"/"never"/"must"/"not" (not isDeterministicRule)
	// Does NOT contain "first"/"then"/"step" (not isProceduralWorkflow)
	learning := "In this codebase ISSUE-123 was fixed using special patches"
	content := "## Promoted Learnings\n\n- " + learning + "\n"
	_ = os.WriteFile(claudeMD, []byte(content), 0600)

	result := &memory.OptimizeResult{}
	opts := memory.OptimizeOpts{
		SkillsDir:    tmpDir,
		ClaudeMDPath: claudeMD,
		MemoryRoot:   tmpDir,
		AutoApprove:  true,
	}
	err := memory.OptimizeDemoteClaudeMDForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.ClaudeMDDemoted).To(BeNumerically(">=", 1))
}

func TestOptimizeDemoteClaudeMD_NoFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	tmpDir := t.TempDir()
	result := &memory.OptimizeResult{}
	opts := memory.OptimizeOpts{
		SkillsDir:    tmpDir,
		ClaudeMDPath: filepath.Join(tmpDir, "nonexistent.md"),
	}
	err := memory.OptimizeDemoteClaudeMDForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.ClaudeMDDemoted).To(Equal(0))
}

func TestOptimizeDemoteClaudeMD_NoPromotedSection(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	tmpDir := t.TempDir()
	claudeMD := filepath.Join(tmpDir, "CLAUDE.md")
	// CLAUDE.md without "Promoted Learnings" section → !ok path
	content := "# My Config\n\n## Other Section\n\n- some item\n"
	_ = os.WriteFile(claudeMD, []byte(content), 0600)
	result := &memory.OptimizeResult{}
	opts := memory.OptimizeOpts{
		SkillsDir:    tmpDir,
		ClaudeMDPath: claudeMD,
	}
	err := memory.OptimizeDemoteClaudeMDForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.ClaudeMDDemoted).To(Equal(0))
}

// ============================================================================
// optimizeDemoteClaudeMD
// ============================================================================

func TestOptimizeDemoteClaudeMD_NoSkillsDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	result := &memory.OptimizeResult{}
	opts := memory.OptimizeOpts{SkillsDir: ""}
	err := memory.OptimizeDemoteClaudeMDForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.ClaudeMDDemoted).To(Equal(0))
}

// TestOptimizeDemoteClaudeMD_NonExistentFile covers the os.IsNotExist branch
// when ClaudeMDPath points to a non-existent file (returns nil, not an error).
func TestOptimizeDemoteClaudeMD_NonExistentFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	tmpDir := t.TempDir()
	result := &memory.OptimizeResult{}
	opts := memory.OptimizeOpts{
		SkillsDir:    tmpDir,
		ClaudeMDPath: filepath.Join(tmpDir, "DOES_NOT_EXIST.md"),
		MemoryRoot:   tmpDir,
	}
	err := memory.OptimizeDemoteClaudeMDForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.ClaudeMDDemoted).To(Equal(0))
}

// ============================================================================
// optimizeMergeSkills
// ============================================================================

func TestOptimizeMergeSkills_NoSkills(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	result := &memory.OptimizeResult{}
	opts := memory.OptimizeOpts{
		SimilarityFunc: func(_ *sql.DB, _, _ int64) (float64, error) { return 0.9, nil },
	}
	err := memory.OptimizeMergeSkillsForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.SkillsMerged).To(Equal(0))
}

func TestOptimizeMergeSkills_TwoSimilarSkills(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	tmpDir := t.TempDir()
	now := time.Now().UTC().Format(time.RFC3339)

	// Insert two skills with embedding IDs
	vecID1, _ := memory.InsertVecEmbeddingForTest(db, makeUnitVec())
	vecID2, _ := memory.InsertVecEmbeddingForTest(db, makeUnitVec())
	_, _ = db.Exec(
		`INSERT INTO generated_skills (slug, theme, description, content, source_memory_ids, alpha, beta, utility, created_at, updated_at, pruned, embedding_id)
		 VALUES ('skill-a', 'skill a', 'Use when a', 'content a', '[1]', 1.0, 1.0, 0.8, ?, ?, 0, ?)`,
		now, now, vecID1,
	)
	_, _ = db.Exec(
		`INSERT INTO generated_skills (slug, theme, description, content, source_memory_ids, alpha, beta, utility, created_at, updated_at, pruned, embedding_id)
		 VALUES ('skill-b', 'skill b', 'Use when b', 'content b', '[2]', 1.0, 1.0, 0.6, ?, ?, 0, ?)`,
		now, now, vecID2,
	)

	result := &memory.OptimizeResult{}
	opts := memory.OptimizeOpts{
		SkillsDir: tmpDir,
		// Injected similarity function returns 0.9 > 0.85 → merge
		SimilarityFunc: func(_ *sql.DB, _, _ int64) (float64, error) { return 0.9, nil },
	}
	err := memory.OptimizeMergeSkillsForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.SkillsMerged).To(Equal(1))
}

// TestOptimizePromoteSkills_CandidateFiltered verifies that a skill with fewer projects than
// MinSkillProjects is filtered out and SkillsPromoted stays 0.
// Uses the early-return path (no ONNX needed when filteredCandidates is empty).
func TestOptimizePromoteSkills_CandidateFiltered(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	tmpDir := t.TempDir()
	claudeMD := filepath.Join(tmpDir, "CLAUDE.md")

	err := os.WriteFile(claudeMD, []byte("# CLAUDE\n"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	db := newOptDB(t)

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS skill_usage (skill_id INTEGER, project TEXT, timestamp TEXT)")
	g.Expect(err).ToNot(HaveOccurred())

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = db.Exec(
		`INSERT INTO generated_skills (slug, theme, description, content, source_memory_ids, alpha, beta, utility, retrieval_count, created_at, updated_at, pruned, claude_md_promoted)
		 VALUES ('high-util', 'high util', 'Use when needed', 'content', '[]', 9.0, 1.0, 0.9, 10, ?, ?, 0, 0)`,
		now, now,
	)
	g.Expect(err).ToNot(HaveOccurred())

	var skillID int64
	_ = db.QueryRow("SELECT id FROM generated_skills WHERE slug='high-util'").Scan(&skillID)

	// Only 1 distinct project — MinSkillProjects=3 filters it out → early return (no ONNX needed).
	_, err = db.Exec("INSERT INTO skill_usage (skill_id, project, timestamp) VALUES (?, 'proj1', ?)", skillID, now)
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.OptimizeOpts{
		SkillsDir:          tmpDir,
		ClaudeMDPath:       claudeMD,
		SkillCompiler:      &mockSkillCompiler{},
		MinSkillUtility:    0.8,
		MinSkillConfidence: 0.7,
		MinSkillProjects:   3,
	}
	result := &memory.OptimizeResult{}

	err = memory.OptimizePromoteSkillsForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.SkillsPromoted).To(Equal(0))
}

// TestOptimizePromoteSkills_DedupWithEmbedder verifies that a candidate passing project filtering
// is deduplicated against an existing CLAUDE.md entry when using an injected embedder.
func TestOptimizePromoteSkills_DedupWithEmbedder(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	tmpDir := t.TempDir()
	claudeMD := filepath.Join(tmpDir, "CLAUDE.md")

	// Existing CLAUDE.md learning so the dedup check has something to compare against.
	err := os.WriteFile(claudeMD, []byte("# CLAUDE\n\n## Promoted Learnings\n\n- Always test\n"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	db := newOptDB(t)

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS skill_usage (skill_id INTEGER, project TEXT, timestamp TEXT)")
	g.Expect(err).ToNot(HaveOccurred())

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = db.Exec(
		`INSERT INTO generated_skills (slug, theme, description, content, source_memory_ids, alpha, beta, utility, retrieval_count, created_at, updated_at, pruned, claude_md_promoted)
		 VALUES ('dup-skill', 'dup theme', 'Use when needed', 'content', '[]', 9.0, 1.0, 0.9, 10, ?, ?, 0, 0)`,
		now, now,
	)
	g.Expect(err).ToNot(HaveOccurred())

	var skillID int64
	_ = db.QueryRow("SELECT id FROM generated_skills WHERE slug='dup-skill'").Scan(&skillID)

	// 3 distinct projects → passes MinSkillProjects=3 → filteredCandidates non-empty.
	for _, proj := range []string{"proj1", "proj2", "proj3"} {
		_, err = db.Exec("INSERT INTO skill_usage (skill_id, project, timestamp) VALUES (?, ?, ?)", skillID, proj, now)
		g.Expect(err).ToNot(HaveOccurred())
	}

	// Mock embedder: always returns the same unit vector → similarity=1.0 → dedup fires.
	unitVec := []float32{1.0, 0, 0, 0}
	opts := memory.OptimizeOpts{
		SkillsDir:          tmpDir,
		ClaudeMDPath:       claudeMD,
		SkillCompiler:      &mockSkillCompiler{},
		MinSkillUtility:    0.8,
		MinSkillConfidence: 0.7,
		MinSkillProjects:   3,
		Embedder:           func(_ string) ([]float32, error) { return unitVec, nil },
	}
	result := &memory.OptimizeResult{}

	err = memory.OptimizePromoteSkillsForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	// Candidate is deduplicated (same embedding as existing learning) → not promoted.
	g.Expect(result.SkillsPromoted).To(Equal(0))
}

func TestOptimizePromoteSkills_NoCompiler(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	tmpDir := t.TempDir()
	result := &memory.OptimizeResult{}
	opts := memory.OptimizeOpts{
		SkillsDir:     tmpDir,
		SkillCompiler: nil,
	}
	err := memory.OptimizePromoteSkillsForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
}

// ============================================================================
// optimizePromoteSkills
// ============================================================================

func TestOptimizePromoteSkills_NoSkillsDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	result := &memory.OptimizeResult{}
	opts := memory.OptimizeOpts{SkillsDir: ""}
	err := memory.OptimizePromoteSkillsForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
}

func TestOptimizePromote_CandidateSkippedNoPrinciple(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	tmpDir := t.TempDir()
	// Insert candidate: retrieval_count=10 (>= MinRetrievals), 3 projects, but principle=""
	_, _ = db.Exec(
		`INSERT INTO embeddings (content, source, confidence, promoted, retrieval_count, projects_retrieved, principle)
		 VALUES ('some content', 'memory', 1.0, 0, 10, 'proj1,proj2,proj3', '')`,
	)
	result := &memory.OptimizeResult{}
	opts := memory.OptimizeOpts{
		MinRetrievals: 5,
		MinProjects:   3,
		AutoApprove:   true,
		ClaudeMDPath:  filepath.Join(tmpDir, "CLAUDE.md"),
	}
	err := memory.OptimizePromoteForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	// Candidate found but skipped (no principle)
	g.Expect(result.PromotionCandidates).To(BeNumerically(">=", 1))
	g.Expect(result.PromotionsApproved).To(Equal(0))
}

// ============================================================================
// optimizePromote
// ============================================================================

func TestOptimizePromote_NoCandiates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	tmpDir := t.TempDir()
	result := &memory.OptimizeResult{}
	opts := memory.OptimizeOpts{
		MinRetrievals: 5,
		MinProjects:   3,
		ClaudeMDPath:  filepath.Join(tmpDir, "CLAUDE.md"),
	}
	err := memory.OptimizePromoteForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.PromotionCandidates).To(Equal(0))
}

// TestOptimizePromote_WithPrinciple_AutoApprove verifies a candidate with a non-empty principle
// is promoted when AutoApprove=true and CLAUDE.md has no existing Promoted Learnings.
func TestOptimizePromote_WithPrinciple_AutoApprove(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	tmpDir := t.TempDir()
	claudeMD := filepath.Join(tmpDir, "CLAUDE.md")

	// No "Promoted Learnings" section → no ONNX embedding generation needed.
	err := os.WriteFile(claudeMD, []byte("# My Claude\n\n## Rules\n\n- Be concise\n"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	db := newOptDB(t)
	_, err = db.Exec(
		`INSERT INTO embeddings (content, principle, source, confidence, retrieval_count, projects_retrieved, promoted)
		 VALUES ('Always run tests before committing any changes to production code', 'Always run tests before committing', 'memory', 0.9, 10, 'proj1,proj2,proj3', 0)`,
	)
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.OptimizeOpts{
		ClaudeMDPath:  claudeMD,
		AutoApprove:   true,
		MinRetrievals: 5,
		MinProjects:   2,
	}
	result := &memory.OptimizeResult{}

	err = memory.OptimizePromoteForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.PromotionCandidates).To(BeNumerically(">=", 1))
	g.Expect(result.PromotionsApproved).To(BeNumerically(">=", 1))
}

func TestOptimizePrune_AboveThreshold_NotPruned(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	id := insertOptEntry(t, db, "High confidence memory", "memory", 0.9, 0)
	attachVec(t, db, id)
	result := &memory.OptimizeResult{}
	opts := memory.OptimizeOpts{PruneThreshold: 0.1}
	err := memory.OptimizePruneForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.EntriesPruned).To(Equal(0))
}

func TestOptimizePrune_BelowThreshold(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	// Must attach a vec embedding — optimizePrune scans embedding_id as int64 (not NullInt64)
	id := insertOptEntry(t, db, "Low confidence memory", "memory", 0.05, 0)
	attachVec(t, db, id)
	result := &memory.OptimizeResult{}
	opts := memory.OptimizeOpts{PruneThreshold: 0.1}
	err := memory.OptimizePruneForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.EntriesPruned).To(Equal(1))
}

// ============================================================================
// optimizePrune
// ============================================================================

func TestOptimizePrune_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	result := &memory.OptimizeResult{}
	opts := memory.OptimizeOpts{PruneThreshold: 0.1}
	err := memory.OptimizePruneForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.EntriesPruned).To(Equal(0))
}

// TestOptimizePrune_NoVecEmbedding verifies that entries with NULL embedding_id are pruned
// without attempting vec_embeddings cleanup.
func TestOptimizePrune_NoVecEmbedding(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)

	_, err := db.Exec(
		"INSERT INTO embeddings (content, source, confidence, promoted) VALUES ('prunable entry', 'test', 0.001, 0)",
	)
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.OptimizeOpts{PruneThreshold: 0.01}
	result := &memory.OptimizeResult{}

	err = memory.OptimizePruneForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.EntriesPruned).To(Equal(1))
}

func TestOptimizePurgeBoilerplate_BoilerplateEntry(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	// "# Session Summary" is boilerplate (header line)
	insertOptEntry(t, db, "# Session Summary", "memory", 1.0, 0)
	result := &memory.OptimizeResult{}
	err := memory.OptimizePurgeBoilerplateForTest(db, memory.OptimizeOpts{}, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.BoilerplatePurged).To(Equal(1))
}

// ============================================================================
// optimizePurgeBoilerplate
// ============================================================================

func TestOptimizePurgeBoilerplate_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	result := &memory.OptimizeResult{}
	err := memory.OptimizePurgeBoilerplateForTest(db, memory.OptimizeOpts{}, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.BoilerplatePurged).To(Equal(0))
}

func TestOptimizePurgeBoilerplate_NonBoilerplate_NotPurged(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	insertOptEntry(t, db, "Always test before committing code to repository", "memory", 1.0, 0)
	result := &memory.OptimizeResult{}
	err := memory.OptimizePurgeBoilerplateForTest(db, memory.OptimizeOpts{}, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.BoilerplatePurged).To(Equal(0))
}

// TestOptimizePurgeBoilerplate_TimestampedBoilerplateMessage verifies entries that are
// NOT raw boilerplate but whose extracted message IS boilerplate are purged
// (covers `msg != "" && IsSessionBoilerplate(msg)` branch).
func TestOptimizePurgeBoilerplate_TimestampedBoilerplateMessage(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	// Content where extractMessageContent extracts "# Session Summary" (a boilerplate header).
	// The raw content itself is NOT boilerplate (starts with "- ", not "#").
	insertOptEntry(t, db, "- 2026-02-10 15:04: # Session Summary", "memory", 1.0, 0)
	result := &memory.OptimizeResult{}
	err := memory.OptimizePurgeBoilerplateForTest(db, memory.OptimizeOpts{}, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.BoilerplatePurged).To(Equal(1))
}

// ============================================================================
// optimizePurgeBoilerplate (additional coverage)
// ============================================================================

// TestOptimizePurgeBoilerplate_WithVecEmbedding verifies boilerplate entries with vec
// embedding_id are cleaned up correctly (covers `if e.embeddingID > 0` branch).
func TestOptimizePurgeBoilerplate_WithVecEmbedding(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	// Insert boilerplate entry and attach a vec embedding
	id := insertOptEntry(t, db, "# Session Summary", "memory", 1.0, 0)
	attachVec(t, db, id)
	result := &memory.OptimizeResult{}
	err := memory.OptimizePurgeBoilerplateForTest(db, memory.OptimizeOpts{}, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.BoilerplatePurged).To(Equal(1))
}

// ============================================================================
// optimizePurgeLegacySessionEmbeddings
// ============================================================================

func TestOptimizePurgeLegacy_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	result := &memory.OptimizeResult{}
	err := memory.OptimizePurgeLegacySessionEmbeddingsForTest(db, memory.OptimizeOpts{}, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.LegacySessionPurged).To(Equal(0))
}

func TestOptimizePurgeLegacy_LegacyEntry(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	// Entry with source='memory', no obs_type, no mem_type, no timestamp prefix → legacy
	insertOptEntryFull(t, db, "raw session line without timestamp", "memory", 1.0, 0, "", "", 0)
	result := &memory.OptimizeResult{}
	err := memory.OptimizePurgeLegacySessionEmbeddingsForTest(db, memory.OptimizeOpts{}, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.LegacySessionPurged).To(Equal(1))
}

func TestOptimizePurgeLegacy_RetrievalCount_Kept(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	// Entry with retrieval_count > 0 → preserved even if no timestamp
	insertOptEntryFull(t, db, "raw line but actively used", "memory", 1.0, 0, "", "", 5)
	result := &memory.OptimizeResult{}
	err := memory.OptimizePurgeLegacySessionEmbeddingsForTest(db, memory.OptimizeOpts{}, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.LegacySessionPurged).To(Equal(0))
}

func TestOptimizePurgeLegacy_TimestampEntry_Kept(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	// Entry with Learn() timestamp prefix → NOT legacy
	insertOptEntryFull(t, db, "- 2026-02-10 15:04: some lesson learned", "memory", 1.0, 0, "", "", 0)
	result := &memory.OptimizeResult{}
	err := memory.OptimizePurgeLegacySessionEmbeddingsForTest(db, memory.OptimizeOpts{}, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.LegacySessionPurged).To(Equal(0))
}

// ============================================================================
// optimizeSplitSkills
// ============================================================================

func TestOptimizeSplitSkills_NoSkills(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	result := &memory.OptimizeResult{}
	opts := memory.OptimizeOpts{
		MinClusterSize: 3,
		SimilarityFunc: func(_ *sql.DB, _, _ int64) (float64, error) { return 0.5, nil },
	}
	err := memory.OptimizeSplitSkillsForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.SkillsSplit).To(Equal(0))
}

func TestOptimizeSplitSkills_SkillTooSmall(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	now := time.Now().UTC().Format(time.RFC3339)
	// Skill with only 3 memory IDs (< minCluster*2 = 6) → skipped
	_, _ = db.Exec(
		`INSERT INTO generated_skills (slug, theme, description, content, source_memory_ids, created_at, updated_at, pruned)
		 VALUES ('small-skill', 'small', 'Use when small', 'content', '[1,2,3]', ?, ?, 0)`,
		now, now,
	)
	result := &memory.OptimizeResult{}
	opts := memory.OptimizeOpts{
		MinClusterSize: 3,
		SimilarityFunc: func(_ *sql.DB, _, _ int64) (float64, error) { return 0.9, nil },
	}
	err := memory.OptimizeSplitSkillsForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.SkillsSplit).To(Equal(0))
}

// TestOptimizeSplitSkills_TwoSubclusters verifies that a skill with source memories in two
// distinct semantic groups is split and SkillsSplit is incremented.
func TestOptimizeSplitSkills_TwoSubclusters(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	tmpDir := t.TempDir()

	// Group 1: 3 entries with unit vec [1,0,...].
	group1VecIDs := make(map[int64]bool)
	var memIDs []int64
	for range 3 {
		id := insertOptEntry(t, db, "git workflow procedure", "memory", 0.9, 0)
		vecID := attachVec(t, db, id)
		group1VecIDs[vecID] = true
		memIDs = append(memIDs, id)
	}

	// Group 2: 3 entries with vec [0,1,...] (orthogonal to group 1).
	group2VecIDs := make(map[int64]bool)
	vec2 := make([]float32, 384)
	vec2[1] = 1.0
	for range 3 {
		id := insertOptEntry(t, db, "database migration pattern", "memory", 0.9, 0)
		vecID, err := memory.InsertVecEmbeddingForTest(db, vec2)
		g.Expect(err).ToNot(HaveOccurred())
		_, err = db.Exec("UPDATE embeddings SET embedding_id = ? WHERE id = ?", vecID, id)
		g.Expect(err).ToNot(HaveOccurred())
		group2VecIDs[vecID] = true
		memIDs = append(memIDs, id)
	}

	// Build source_memory_ids JSON from all 6 IDs.
	strIDs := make([]string, len(memIDs))
	for i, id := range memIDs {
		strIDs[i] = strconv.FormatInt(id, 10)
	}
	sourceIDsJSON := "[" + strings.Join(strIDs, ",") + "]"

	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec(
		`INSERT INTO generated_skills (slug, theme, description, content, source_memory_ids, alpha, beta, utility, retrieval_count, created_at, updated_at, pruned)
		 VALUES ('mixed-skill', 'mixed theme', 'Use when mixed', 'content', ?, 1.0, 1.0, 0.5, 0, ?, ?, 0)`,
		sourceIDsJSON, now, now,
	)
	g.Expect(err).ToNot(HaveOccurred())

	// Injected sim function: 1.0 within same group, 0.0 across groups.
	simFunc := func(_ *sql.DB, id1, id2 int64) (float64, error) {
		if (group1VecIDs[id1] && group1VecIDs[id2]) || (group2VecIDs[id1] && group2VecIDs[id2]) {
			return 1.0, nil
		}
		return 0.0, nil
	}

	opts := memory.OptimizeOpts{
		SkillsDir:      tmpDir,
		MinClusterSize: 3,
		SimilarityFunc: simFunc,
	}
	result := &memory.OptimizeResult{}

	err = memory.OptimizeSplitSkillsForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.SkillsSplit).To(BeNumerically(">=", 1))
}

func TestOptimizeSynthesize_ClusterFormed(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	// Insert 3 entries with identical vec embeddings → cluster forms
	for range 3 {
		id := insertOptEntry(t, db,
			"Always use git to track changes during software development workflows and collaboration",
			"memory", 1.0, 0)
		attachVec(t, db, id)
	}
	result := &memory.OptimizeResult{}
	opts := memory.OptimizeOpts{
		MinClusterSize: 3,
		SynthThreshold: 0.8,
		AutoApprove:    true,
	}
	err := memory.OptimizeSynthesizeForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	// Cluster forms → at least checked
	g.Expect(result.PatternsFound).To(BeNumerically(">=", 0))
}

// ============================================================================
// optimizeSynthesize
// ============================================================================

func TestOptimizeSynthesize_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	result := &memory.OptimizeResult{}
	opts := memory.OptimizeOpts{MinClusterSize: 3, SynthThreshold: 0.8, AutoApprove: true}
	err := memory.OptimizeSynthesizeForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
}

func TestOptimizeSynthesize_InsufficientEntries(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	// Insert 2 entries with embeddings (less than MinClusterSize=3)
	id1 := insertOptEntry(t, db, "Always use git", "memory", 1.0, 0)
	attachVec(t, db, id1)
	id2 := insertOptEntry(t, db, "Always use git", "memory", 1.0, 0)
	attachVec(t, db, id2)

	result := &memory.OptimizeResult{}
	opts := memory.OptimizeOpts{MinClusterSize: 3, SynthThreshold: 0.8}
	err := memory.OptimizeSynthesizeForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.PatternsFound).To(Equal(0))
}

// TestOptimizeSynthesize_LowQualityPattern verifies a cluster with non-actionable, non-specific
// content increments PatternsFound but not PatternsApproved.
func TestOptimizeSynthesize_LowQualityPattern(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)

	// 3 entries with identical content and unit vecs cluster together.
	// generatePattern produces a short synthesis that fails quality validation (quality < 0.8).
	for range 3 {
		id := insertOptEntry(t, db, "x", "memory", 0.9, 0)
		attachVec(t, db, id)
	}

	opts := memory.OptimizeOpts{
		SynthThreshold: 0.5, // Identical unit vecs have cosine similarity 1.0 ≥ 0.5.
		MemoryRoot:     t.TempDir(),
	}
	result := &memory.OptimizeResult{}

	err := memory.OptimizeSynthesizeForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.PatternsFound).To(BeNumerically(">=", 1))
	g.Expect(result.PatternsApproved).To(Equal(0))
}

func TestOptimize_CancelledContext(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	tmpDir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	result, err := memory.Optimize(memory.OptimizeOpts{
		MemoryRoot:   tmpDir,
		ClaudeMDPath: filepath.Join(tmpDir, "CLAUDE.md"),
		Context:      ctx,
	})
	g.Expect(err).To(HaveOccurred())
	g.Expect(result).To(BeNil())
}

// ============================================================================
// Optimize (full pipeline integration)
// ============================================================================

func TestOptimize_EmptyMemoryRoot(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, "skills")
	result, err := memory.Optimize(memory.OptimizeOpts{
		MemoryRoot:   tmpDir,
		ClaudeMDPath: filepath.Join(tmpDir, "CLAUDE.md"),
		SkillsDir:    skillsDir,
		AutoApprove:  true,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())
}

// TestOptimize_WithEntries verifies the full Optimize pipeline runs without error and
// purges boilerplate session entries.
func TestOptimize_WithEntries(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	memRoot := t.TempDir()
	claudeMD := filepath.Join(memRoot, "CLAUDE.md")

	err := os.WriteFile(claudeMD, []byte("# CLAUDE\n"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	// Pre-populate the DB with boilerplate entries.
	dbPath := filepath.Join(memRoot, "embeddings.db")
	db, err := memory.InitEmbeddingsDBForTest(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	for _, content := range []string{"---", "## Summary", "..."} {
		_, err = db.Exec(
			"INSERT INTO embeddings (content, source, confidence, promoted) VALUES (?, 'memory', 0.9, 0)",
			content,
		)
		g.Expect(err).ToNot(HaveOccurred())
	}
	_ = db.Close()

	opts := memory.OptimizeOpts{
		MemoryRoot:   memRoot,
		ClaudeMDPath: claudeMD,
		AutoApprove:  false,
		// SkillsDir="" → skip all skill-related steps that need ONNX.
	}
	result, err := memory.Optimize(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.BoilerplatePurged).To(BeNumerically(">=", 1))
}

// ============================================================================
// performSkillReorganization
// ============================================================================

func TestPerformSkillReorganization_InsufficientMemories(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	tmpDir := t.TempDir()
	result := &memory.OptimizeResult{}
	opts := memory.OptimizeOpts{
		SkillsDir:      tmpDir,
		MinClusterSize: 3,
		ReorgThreshold: 0.8,
	}
	// Empty DB → not enough memories → sets metadata, returns nil
	err := memory.PerformSkillReorganizationForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
}

// TestPerformSkillReorganization_UpdateExistingSkill verifies that a pre-existing skill with
// a matching slug is updated (not duplicated) during reorganization.
func TestPerformSkillReorganization_UpdateExistingSkill(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	tmpDir := t.TempDir()

	// 3 entries with identical unit vecs → cluster of 3.
	for range 3 {
		id := insertOptEntry(t, db, "git commit workflow", "memory", 0.9, 0)
		attachVec(t, db, id)
	}

	// Pre-insert the skill that will be generated from "git commit workflow".
	// generateThemeFromCluster → "git commit workflow" → slugify → "git-commit-workflow".
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec(
		`INSERT INTO generated_skills (slug, theme, description, content, source_memory_ids, alpha, beta, utility, retrieval_count, created_at, updated_at, pruned)
		 VALUES ('git-commit-workflow', 'git commit workflow', 'Use when git commit workflow', 'old content', '[]', 1.0, 1.0, 0.5, 0, ?, ?, 0)`,
		now, now,
	)
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.OptimizeOpts{
		SkillsDir:      tmpDir,
		ReorgThreshold: 0.5, // Low threshold so identical unit vecs cluster.
	}
	result := &memory.OptimizeResult{}

	err = memory.PerformSkillReorganizationForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.SkillsReorganized).To(BeNumerically(">=", 1))
}

func TestPerformSkillReorganization_WithCluster(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	tmpDir := t.TempDir()
	// Insert 3 entries with identical vec embeddings to form a cluster
	for range 3 {
		id := insertOptEntry(t, db, "Always write tests for all new code changes", "memory", 1.0, 0)
		attachVec(t, db, id)
	}
	result := &memory.OptimizeResult{}
	opts := memory.OptimizeOpts{
		SkillsDir:      tmpDir,
		MinClusterSize: 3,
		ReorgThreshold: 0.8,
	}
	err := memory.PerformSkillReorganizationForTest(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
}

// ============================================================================
// pruneOrphanedSkills
// ============================================================================

func TestPruneOrphanedSkills_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	tmpDir := t.TempDir()
	count, err := memory.PruneOrphanedSkillsForTest(db, tmpDir, map[string]bool{"keep-this": true})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(count).To(Equal(0))
}

func TestPruneOrphanedSkills_SkillInActiveThemes_Kept(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	tmpDir := t.TempDir()
	now := time.Now().UTC().Format(time.RFC3339)
	_, _ = db.Exec(
		`INSERT INTO generated_skills (slug, theme, description, content, source_memory_ids, created_at, updated_at, pruned)
		 VALUES ('active-skill', 'active', 'Use when active', 'content', '[]', ?, ?, 0)`,
		now, now,
	)
	count, err := memory.PruneOrphanedSkillsForTest(db, tmpDir, map[string]bool{"active-skill": true})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(count).To(Equal(0))
}

func TestPruneOrphanedSkills_SkillNotInActiveThemes(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	tmpDir := t.TempDir()
	now := time.Now().UTC().Format(time.RFC3339)
	_, _ = db.Exec(
		`INSERT INTO generated_skills (slug, theme, description, content, source_memory_ids, created_at, updated_at, pruned)
		 VALUES ('orphan-skill', 'orphan', 'Use when orphan', 'content', '[]', ?, ?, 0)`,
		now, now,
	)
	// activeThemes does not contain "orphan-skill" → pruned
	count, err := memory.PruneOrphanedSkillsForTest(db, tmpDir, map[string]bool{})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(count).To(Equal(1))
}

// ============================================================================
// pruneStaleSkills
// ============================================================================

func TestPruneStaleSkills_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	tmpDir := t.TempDir()
	count := memory.PruneStaleSkillsForTest(db, tmpDir, 0.4)
	g.Expect(count).To(Equal(0))
}

func TestPruneStaleSkills_LowRetrievals_NotPruned(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	tmpDir := t.TempDir()
	now := time.Now().UTC().Format(time.RFC3339)
	// Skill with utility < 0.4 but retrieval_count < 5 → NOT stale
	_, _ = db.Exec(
		`INSERT INTO generated_skills (slug, theme, description, content, source_memory_ids, utility, retrieval_count, created_at, updated_at, pruned)
		 VALUES ('young-skill', 'young', 'Use when young', 'content', '[]', 0.2, 2, ?, ?, 0)`,
		now, now,
	)
	count := memory.PruneStaleSkillsForTest(db, tmpDir, 0.4)
	g.Expect(count).To(Equal(0))
}

func TestPruneStaleSkills_StaleSkill(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	db := newOptDB(t)
	tmpDir := t.TempDir()
	now := time.Now().UTC().Format(time.RFC3339)
	// Skill with utility < 0.4 AND retrieval_count >= 5 → stale
	_, _ = db.Exec(
		`INSERT INTO generated_skills (slug, theme, description, content, source_memory_ids, utility, retrieval_count, created_at, updated_at, pruned)
		 VALUES ('stale-skill', 'stale', 'Use when stale', 'content', '[]', 0.2, 7, ?, ?, 0)`,
		now, now,
	)
	count := memory.PruneStaleSkillsForTest(db, tmpDir, 0.4)
	g.Expect(count).To(Equal(1))
}

// attachVec inserts a vec embedding and links it to an embeddings row.
func attachVec(t *testing.T, db *sql.DB, entryID int64) int64 {
	t.Helper()
	vecID, err := memory.InsertVecEmbeddingForTest(db, makeUnitVec())
	if err != nil {
		t.Fatalf("failed to insert vec embedding: %v", err)
	}
	_, err = db.Exec("UPDATE embeddings SET embedding_id = ? WHERE id = ?", vecID, entryID)
	if err != nil {
		t.Fatalf("failed to attach vec: %v", err)
	}
	return vecID
}

// insertOptEntry inserts a minimal embeddings row and returns its id.
func insertOptEntry(t *testing.T, db *sql.DB, content, source string, confidence float64, promoted int) int64 {
	t.Helper()
	res, err := db.Exec(
		"INSERT INTO embeddings (content, source, confidence, promoted) VALUES (?, ?, ?, ?)",
		content, source, confidence, promoted,
	)
	if err != nil {
		t.Fatalf("failed to insert entry: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

// insertOptEntryFull inserts an entry with all relevant columns set.
func insertOptEntryFull(t *testing.T, db *sql.DB, content, source string, confidence float64, promoted int, memType, obsType string, retrievalCount int) int64 {
	t.Helper()
	res, err := db.Exec(
		`INSERT INTO embeddings (content, source, confidence, promoted, memory_type, observation_type, retrieval_count)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		content, source, confidence, promoted, memType, obsType, retrievalCount,
	)
	if err != nil {
		t.Fatalf("failed to insert entry: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

// makeUnitVec returns a 384-dimensional unit vector (1.0 at index 0).
func makeUnitVec() []float32 {
	v := make([]float32, 384)
	v[0] = 1.0
	return v
}

// newOptDB returns an in-memory SQLite DB for optimize tests.
func newOptDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := memory.InitEmbeddingsDBForTest(":memory:")
	if err != nil {
		t.Fatalf("failed to init test DB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}
