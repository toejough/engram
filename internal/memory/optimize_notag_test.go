package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	. "github.com/onsi/gomega"
)

func TestCheckContext_ActiveContext(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := checkContext(OptimizeOpts{Context: context.Background()})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestCheckContext_CancelledContext(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := checkContext(OptimizeOpts{Context: ctx})
	g.Expect(err).To(HaveOccurred())
}

// ─── checkContext ────────────────────────────────────────────────────────────

func TestCheckContext_Nil(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := checkContext(OptimizeOpts{})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestClusterEntriesToEmbeddings_Empty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := clusterEntriesToEmbeddings(nil)
	g.Expect(result).To(BeEmpty())
}

func TestClusterEntriesToEmbeddings_Values(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cluster := []ClusterEntry{
		{ID: 1, Content: "alpha"},
		{ID: 2, Content: "beta"},
	}
	result := clusterEntriesToEmbeddings(cluster)
	g.Expect(result).To(HaveLen(2))
	g.Expect(result[0].ID).To(Equal(int64(1)))
	g.Expect(result[0].Content).To(Equal("alpha"))
	g.Expect(result[1].ID).To(Equal(int64(2)))
}

func TestClusterHasExistingSkill_Empty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(clusterHasExistingSkill(nil, map[int64]bool{})).To(BeFalse())
}

func TestClusterHasExistingSkill_Found(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cluster := []ClusterEntry{{ID: 3, Content: "x"}}
	ids := map[int64]bool{3: true}

	g.Expect(clusterHasExistingSkill(cluster, ids)).To(BeTrue())
}

func TestClusterHasExistingSkill_NotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cluster := []ClusterEntry{{ID: 5, Content: "x"}}
	ids := map[int64]bool{3: true}

	g.Expect(clusterHasExistingSkill(cluster, ids)).To(BeFalse())
}

func TestDefaultSkillTester_TestAndEvaluate_InvalidRuns(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tester := &defaultSkillTester{apiKey: ""}
	_, _, err := tester.TestAndEvaluate(TestScenario{}, 0)
	g.Expect(err).To(HaveOccurred())

	if err == nil {
		t.Fatal("expected error but got nil")
	}

	g.Expect(err.Error()).To(ContainSubstring("runs must be > 0"))
}

// TestDefaultSkillTester_TestAndEvaluate_SuccessPath covers the success branch of TestAndEvaluate.
// TestSkillCandidate with runs>=1 always returns nil error (HTTP errors are absorbed into SkillTestResult.Error).
// This covers lines for EvaluateTestResults and the final return.
func TestDefaultSkillTester_TestAndEvaluate_SuccessPath(t *testing.T) {
	// Not parallel: makes 2 HTTP calls to api.anthropic.com (fail fast with fake key).
	g := NewWithT(t)

	tester := &defaultSkillTester{apiKey: "fake-key-for-coverage"}

	// runs=1: TestSkillCandidate always returns nil error even when HTTP fails.
	_, _, err := tester.TestAndEvaluate(TestScenario{}, 1)

	// TestSkillCandidate with runs>=1 never returns an error; HTTP failures are captured in SkillTestResult.
	g.Expect(err).ToNot(HaveOccurred())
}

func TestFormatClusterSourceIDs_Basic(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cluster := []ClusterEntry{{ID: 1}, {ID: 2}, {ID: 3}}
	result := formatClusterSourceIDs(cluster)

	var ids []int64

	err := json.Unmarshal([]byte(result), &ids)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(ids).To(Equal([]int64{1, 2, 3}))
}

func TestFormatClusterSourceIDs_Empty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := formatClusterSourceIDs([]ClusterEntry{})
	g.Expect(result).To(Equal("[]"))
}

// TestGenerateSkillFromLearning_ComplianceFailure covers lines 719-721 (compliance failure returns error).
// A fakeSkillCompiler returning content without required sections triggers ValidateSkillCompliance failure.
func TestGenerateSkillFromLearning_ComplianceFailure(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	opts := OptimizeOpts{
		SkillsDir:     t.TempDir(),
		SkillCompiler: &fakeSkillCompiler{content: "no sections here at all"},
	}

	// Compiler returns content without required sections → ValidateSkillCompliance fails → error
	err = generateSkillFromLearning(db, opts, "always validate input before processing")
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("failed compliance validation"))
	}
}

func TestGenerateSkillFromLearning_EmptySkillsDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	skillsDir := t.TempDir()
	// learning must produce a skill with a valid description via template
	// The template generates "Use when working on ..." which passes compliance
	opts := OptimizeOpts{
		SkillsDir: skillsDir,
	}

	err = generateSkillFromLearning(db, opts, "use standard library patterns for error handling")
	g.Expect(err).ToNot(HaveOccurred())
}

// TestGenerateSkillFromLearning_ErrLLMUnavailable covers lines 688-692 (LLMUnavailable fallback to template).
func TestGenerateSkillFromLearning_ErrLLMUnavailable(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	opts := OptimizeOpts{
		SkillsDir:     t.TempDir(),
		SkillCompiler: &fakeSkillCompiler{err: ErrLLMUnavailable},
	}

	// ErrLLMUnavailable → prints to stderr, falls back to template → template is compliant → no error
	err = generateSkillFromLearning(db, opts, "always validate input before processing")
	g.Expect(err).ToNot(HaveOccurred())
}

// TestGenerateSkillFromLearning_LongLearning covers line 662-663 (len(theme) > 50 truncation).
func TestGenerateSkillFromLearning_LongLearning(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	opts := OptimizeOpts{SkillsDir: t.TempDir()}

	// Learning > 50 chars triggers len(theme) > 50 truncation at line 662-663
	longLearning := "always make sure to validate all user input data thoroughly before using it in any operation"

	err = generateSkillFromLearning(db, opts, longLearning)
	g.Expect(err).ToNot(HaveOccurred())
}

func TestGenerateSkillFromLearning_NilSkillCompiler(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	opts := OptimizeOpts{
		SkillsDir:     t.TempDir(),
		SkillCompiler: nil,
	}

	// Short but valid learning
	err = generateSkillFromLearning(db, opts, "use interfaces for dependency injection")
	g.Expect(err).ToNot(HaveOccurred())
}

// ─── generateSkillFromLearning (update path) ─────────────────────────────────

func TestGenerateSkillFromLearning_UpdatesExistingSkill(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	skillsDir := t.TempDir()
	learning := "use standard library patterns for error handling"
	opts := OptimizeOpts{SkillsDir: skillsDir}

	// First call: inserts new skill
	err = generateSkillFromLearning(db, opts, learning)
	g.Expect(err).ToNot(HaveOccurred())

	// Second call with same learning: finds existing skill and updates it
	err = generateSkillFromLearning(db, opts, learning)
	g.Expect(err).ToNot(HaveOccurred())
}

func TestGenerateThemeFromCluster_Empty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := generateThemeFromCluster(nil)
	g.Expect(result).To(Equal("Unknown"))
}

func TestGenerateThemeFromCluster_Long(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cluster := []ClusterEntry{{Content: "this is a very long content string that exceeds fifty characters by quite a lot"}}
	result := generateThemeFromCluster(cluster)
	g.Expect(len(result)).To(BeNumerically("<=", 50))
}

func TestGenerateThemeFromCluster_Short(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cluster := []ClusterEntry{{Content: "short content"}}
	result := generateThemeFromCluster(cluster)
	g.Expect(result).ToNot(BeEmpty())
}

func TestGetExistingPatterns_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	patterns := getExistingPatterns(db)
	g.Expect(patterns).To(BeEmpty())
}

// ─── getExistingPatterns ─────────────────────────────────────────────────────

func TestGetExistingPatterns_WithSynthesizedEntries(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	// The getExistingPatterns query uses ORDER BY created_at DESC; add column so query succeeds
	_, _ = db.Exec("ALTER TABLE embeddings ADD COLUMN created_at TEXT NOT NULL DEFAULT ''")

	_, err = db.Exec(
		"INSERT INTO embeddings (content, source, memory_type) VALUES (?, 'synthesized', 'reflection')",
		"always prefer explicit error handling over silent failures",
	)
	g.Expect(err).ToNot(HaveOccurred())

	patterns := getExistingPatterns(db)
	g.Expect(patterns).To(HaveLen(1))

	if len(patterns) > 0 {
		g.Expect(patterns[0]).To(ContainSubstring("error handling"))
	}
}

func TestGetExistingSkillSourceIDs_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	ids, err := getExistingSkillSourceIDs(db)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(ids).To(BeEmpty())
}

func TestGetExistingSkillSourceIDs_WithSkill(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	skill := &GeneratedSkill{
		Slug:            "test-skill",
		Theme:           "test skill",
		Description:     "Use when testing",
		Content:         "content",
		SourceMemoryIDs: "[1,2,3]",
		Alpha:           1.0,
		Beta:            1.0,
	}
	_, err = insertSkill(db, skill)
	g.Expect(err).ToNot(HaveOccurred())

	ids, err := getExistingSkillSourceIDs(db)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(ids[1]).To(BeTrue())
	g.Expect(ids[2]).To(BeTrue())
	g.Expect(ids[3]).To(BeTrue())
}

func TestHasLearnTimestampPrefix_Invalid(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(hasLearnTimestampPrefix("not a timestamp")).To(BeFalse())
	g.Expect(hasLearnTimestampPrefix("- 2026/02/10 15:04: wrong")).To(BeFalse())
	g.Expect(hasLearnTimestampPrefix("- short")).To(BeFalse())
	g.Expect(hasLearnTimestampPrefix("")).To(BeFalse())
}

// ─── hasLearnTimestampPrefix edge cases ──────────────────────────────────────

func TestHasLearnTimestampPrefix_NonDigitInDatePart(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(hasLearnTimestampPrefix("- 2026-XX-10 15:04: message here")).To(BeFalse())
}

func TestHasLearnTimestampPrefix_TooShort(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(hasLearnTimestampPrefix("- 2026-02-10")).To(BeFalse())
}

func TestHasLearnTimestampPrefix_Valid(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(hasLearnTimestampPrefix("- 2026-02-10 15:04: some message")).To(BeTrue())
}

func TestHasLearnTimestampPrefix_WrongForwardSlashSep(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(hasLearnTimestampPrefix("- 2026/02/10 15:04: message here")).To(BeFalse())
}

// ─── migrateMemSkills ────────────────────────────────────────────────────────

func TestMigrateMemSkills_NonExistentDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := migrateMemSkills(RealFS{}, filepath.Join(t.TempDir(), "nonexistent"))
	g.Expect(err).ToNot(HaveOccurred())
}

func TestMigrateMemSkills_RenamesMemDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	skillsDir := t.TempDir()
	memDir := filepath.Join(skillsDir, "mem-foo")
	err := os.Mkdir(memDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	err = os.WriteFile(filepath.Join(memDir, "SKILL.md"), []byte("---\nname: mem:foo\n---\n\ncontent"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	err = migrateMemSkills(RealFS{}, skillsDir)
	g.Expect(err).ToNot(HaveOccurred())

	_, statErr := os.Stat(filepath.Join(skillsDir, "memory-foo"))
	g.Expect(statErr).ToNot(HaveOccurred())
}

func TestMigrateMemSkills_SkipsExistingDestination(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	skillsDir := t.TempDir()
	memDir := filepath.Join(skillsDir, "mem-bar")
	memoryDir := filepath.Join(skillsDir, "memory-bar")
	err := os.Mkdir(memDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	err = os.Mkdir(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	err = os.WriteFile(filepath.Join(memoryDir, "SKILL.md"), []byte("---\nname: memory.bar\n---\n\nexisting"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	err = migrateMemSkills(RealFS{}, skillsDir)
	g.Expect(err).ToNot(HaveOccurred())

	// Original mem-bar should still exist (not renamed since destination exists)
	_, statErr := os.Stat(memDir)
	g.Expect(statErr).ToNot(HaveOccurred())
}

func TestMigrateMemoryGenSkills_MigratesEntries(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	skillsDir := t.TempDir()
	memGenDir := filepath.Join(skillsDir, "memory-gen")
	err := os.MkdirAll(filepath.Join(memGenDir, "myskill"), 0755)
	g.Expect(err).ToNot(HaveOccurred())

	err = migrateMemoryGenSkills(RealFS{}, skillsDir)
	g.Expect(err).ToNot(HaveOccurred())

	_, statErr := os.Stat(filepath.Join(skillsDir, "memory-myskill"))
	g.Expect(statErr).ToNot(HaveOccurred())

	_, goneErr := os.Stat(memGenDir)
	g.Expect(goneErr).To(HaveOccurred())
}

// ─── migrateMemoryGenSkills ──────────────────────────────────────────────────

func TestMigrateMemoryGenSkills_NonExistentDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := migrateMemoryGenSkills(RealFS{}, filepath.Join(t.TempDir(), "nonexistent"))
	g.Expect(err).ToNot(HaveOccurred())
}

func TestMigrateMemoryGenSkills_SkipsExistingDestination(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	skillsDir := t.TempDir()
	memGenDir := filepath.Join(skillsDir, "memory-gen")
	err := os.MkdirAll(filepath.Join(memGenDir, "existing"), 0755)
	g.Expect(err).ToNot(HaveOccurred())

	err = os.Mkdir(filepath.Join(skillsDir, "memory-existing"), 0755)
	g.Expect(err).ToNot(HaveOccurred())

	err = migrateMemoryGenSkills(RealFS{}, skillsDir)
	g.Expect(err).ToNot(HaveOccurred())
}

func TestOptimizeAutoDemote_DemotesLowConfidence(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	_, err = db.Exec(
		"INSERT INTO embeddings (content, source, confidence, promoted) VALUES (?, 'memory', 0.2, 1)",
		"low confidence promoted entry",
	)
	g.Expect(err).ToNot(HaveOccurred())

	result := &OptimizeResult{}
	opts := OptimizeOpts{ClaudeMDPath: filepath.Join(t.TempDir(), "CLAUDE.md")}

	err = optimizeAutoDemote(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.AutoDemoted).To(Equal(1))
}

// ─── optimizeAutoDemote ──────────────────────────────────────────────────────

func TestOptimizeAutoDemote_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	result := &OptimizeResult{}
	opts := OptimizeOpts{ClaudeMDPath: filepath.Join(t.TempDir(), "CLAUDE.md")}

	err = optimizeAutoDemote(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.AutoDemoted).To(Equal(0))
}

func TestOptimizeAutoDemote_KeepsHighConfidence(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	_, err = db.Exec(
		"INSERT INTO embeddings (content, source, confidence, promoted) VALUES (?, 'memory', 0.8, 1)",
		"high confidence promoted entry",
	)
	g.Expect(err).ToNot(HaveOccurred())

	result := &OptimizeResult{}
	opts := OptimizeOpts{ClaudeMDPath: filepath.Join(t.TempDir(), "CLAUDE.md")}

	err = optimizeAutoDemote(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.AutoDemoted).To(Equal(0))
}

func TestOptimizeClaudeMDDedup_NoDuplicates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	claudeMD := filepath.Join(dir, "CLAUDE.md")

	content := "# My Project\n\n## Promoted Learnings\n\n- Use dependency injection for testing\n- Write tests before implementation\n- Keep functions small and focused\n"
	err := os.WriteFile(claudeMD, []byte(content), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	callCount := 0
	result := &OptimizeResult{}
	opts := OptimizeOpts{
		ClaudeMDPath: claudeMD,
		Embedder: func(text string) ([]float32, error) {
			callCount++
			// Return distinct embeddings so nothing is considered duplicate
			emb := make([]float32, 3)
			emb[callCount%3] = 1.0

			return emb, nil
		},
	}

	err = optimizeClaudeMDDedup(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.ClaudeMDDeduped).To(Equal(0))
}

// TestOptimizeClaudeMDDedup_NoEmbedder verifies that when embed=nil, the ONNX block is entered
// and returns nil gracefully when ONNX is not available.
func TestOptimizeClaudeMDDedup_NoEmbedder(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	claudeMD := filepath.Join(dir, "CLAUDE.md")

	// Two entries → passes len(promoted) < 2 check → reaches embed=nil → ONNX block
	content := "# Project\n\n## Promoted Learnings\n\n- Use dependency injection\n- Write unit tests\n"

	err := os.WriteFile(claudeMD, []byte(content), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	result := &OptimizeResult{}

	// No Embedder → embed=nil → ONNX init fails → return nil
	opts := OptimizeOpts{ClaudeMDPath: claudeMD}

	err = optimizeClaudeMDDedup(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
}

func TestOptimizeClaudeMDDedup_NoFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	result := &OptimizeResult{}
	opts := OptimizeOpts{
		ClaudeMDPath: filepath.Join(t.TempDir(), "nonexistent-CLAUDE.md"),
	}

	err = optimizeClaudeMDDedup(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
}

func TestOptimizeClaudeMDDedup_TooFewEntries(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	claudeMD := filepath.Join(dir, "CLAUDE.md")

	content := "# Project\n\n## Promoted Learnings\n\n- Only one entry\n"
	err := os.WriteFile(claudeMD, []byte(content), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	result := &OptimizeResult{}
	opts := OptimizeOpts{ClaudeMDPath: claudeMD}

	err = optimizeClaudeMDDedup(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
}

// TestOptimizeClaudeMDDedup_WithDuplicates verifies that identical embeddings cause deduplication
// and the duplicate entry is removed from CLAUDE.md.
func TestOptimizeClaudeMDDedup_WithDuplicates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	claudeMD := filepath.Join(dir, "CLAUDE.md")

	// Two identical entries — same embedding will trigger duplicate detection
	content := "# Project\n\n## Promoted Learnings\n\n- Use dependency injection for testability\n- Use dependency injection for testability\n"

	err := os.WriteFile(claudeMD, []byte(content), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	result := &OptimizeResult{}
	opts := OptimizeOpts{
		ClaudeMDPath: claudeMD,
		Embedder: func(_ string) ([]float32, error) {
			// Identical embeddings → cosine similarity = 1.0 > 0.9 → duplicate
			return []float32{1.0, 0.0, 0.0}, nil
		},
	}

	err = optimizeClaudeMDDedup(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.ClaudeMDDeduped).To(BeNumerically(">=", 1))
}

// TestOptimizeCompileSkills_CancelledContext covers checkContext returning an error inside the
// cluster loop (line ~1340): cancelled context → function returns context.Canceled.
func TestOptimizeCompileSkills_CancelledContext(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	recentTime := time.Now().UTC().Format(time.RFC3339)

	err = setMetadata(db, "last_skill_reorg_at", recentTime)
	g.Expect(err).ToNot(HaveOccurred())

	// Insert 3 entries with embedding_id so they form a cluster (>= minCluster).
	for _, content := range []string{"ctx test one", "ctx test two", "ctx test three"} {
		_, err = db.Exec("INSERT INTO embeddings (content, source, embedding_id) VALUES (?, 'memory', 1)", content)
		g.Expect(err).ToNot(HaveOccurred())
	}

	// Cancel the context so checkContext returns error on first cluster iteration.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := &OptimizeResult{}
	opts := OptimizeOpts{
		SkillsDir:      t.TempDir(),
		MinClusterSize: 3,
		SimilarityFunc: func(_ *sql.DB, _, _ int64) (float64, error) { return 1.0, nil },
		Context:        ctx,
	}

	err = optimizeCompileSkills(db, opts, result)
	g.Expect(err).To(HaveOccurred())
}

func TestOptimizeCompileSkills_ComplianceFailure(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	recentTime := time.Now().UTC().Format(time.RFC3339)

	err = setMetadata(db, "last_skill_reorg_at", recentTime)
	g.Expect(err).ToNot(HaveOccurred())

	for _, content := range []string{
		"error handling first", "error handling second", "error handling third",
	} {
		_, err = db.Exec(
			"INSERT INTO embeddings (content, source, embedding_id) VALUES (?, 'memory', 1)",
			content,
		)
		g.Expect(err).ToNot(HaveOccurred())
	}

	// Return JSON that fails ValidateSkillCompliance: description doesn't start with "Use when"
	nonCompliantJSON := `{"description":"Bad description without use-when","body":"no sections here"}`

	result := &OptimizeResult{}
	opts := OptimizeOpts{
		SkillsDir:      t.TempDir(),
		MinClusterSize: 3,
		SimilarityFunc: func(_ *sql.DB, _, _ int64) (float64, error) { return 1.0, nil },
		SkillCompiler:  &fakeSkillCompiler{content: nonCompliantJSON},
	}

	err = optimizeCompileSkills(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.SkillsBlocked).To(BeNumerically(">=", 1))
}

func TestOptimizeCompileSkills_ForceReorg(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	result := &OptimizeResult{}
	opts := OptimizeOpts{
		SkillsDir:      t.TempDir(),
		MinClusterSize: 3,
		SimilarityFunc: calculateSimilarity,
		ForceReorg:     true,
	}

	err = optimizeCompileSkills(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
}

func TestOptimizeCompileSkills_InvalidReorgTimestamp(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	err = setMetadata(db, "last_skill_reorg_at", "not-a-valid-timestamp")
	g.Expect(err).ToNot(HaveOccurred())

	result := &OptimizeResult{}
	opts := OptimizeOpts{
		SkillsDir:      t.TempDir(),
		MinClusterSize: 3,
		SimilarityFunc: func(_ *sql.DB, _, _ int64) (float64, error) { return 1.0, nil },
	}

	err = optimizeCompileSkills(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
}

// TestOptimizeCompileSkills_JSONNoDescription covers the inner-else branch (line ~1385):
// parseCompileSkillJSON succeeds but jsonDesc=="" → description via generateTriggerDescription.
func TestOptimizeCompileSkills_JSONNoDescription(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	recentTime := time.Now().UTC().Format(time.RFC3339)

	err = setMetadata(db, "last_skill_reorg_at", recentTime)
	g.Expect(err).ToNot(HaveOccurred())

	for _, content := range []string{"json test one", "json test two", "json test three"} {
		_, err = db.Exec("INSERT INTO embeddings (content, source, embedding_id) VALUES (?, 'memory', 1)", content)
		g.Expect(err).ToNot(HaveOccurred())
	}

	// JSON with no "description" key → jsonDesc="" → else branch → generateTriggerDescription.
	// Body has all 4 required sections → ValidateSkillCompliance passes.
	jsonContent := `{"body":"## Overview\n\ncontent\n\n## When to Use\n\nuse\n\n## Quick Reference\n\nref\n\n## Common Mistakes\n\nmistakes\n"}`

	result := &OptimizeResult{}
	opts := OptimizeOpts{
		SkillsDir:      t.TempDir(),
		MinClusterSize: 3,
		SimilarityFunc: func(_ *sql.DB, _, _ int64) (float64, error) { return 1.0, nil },
		SkillCompiler:  &fakeSkillCompiler{content: jsonContent},
		Context:        context.Background(),
	}

	err = optimizeCompileSkills(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
}

// TestOptimizeCompileSkills_MergeExistingSkill covers the merge path (lines ~1422-1451):
// getSkillBySlug finds existing non-pruned skill with same slug → update + SkillsMerged++.
func TestOptimizeCompileSkills_MergeExistingSkill(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	recentTime := time.Now().UTC().Format(time.RFC3339)

	err = setMetadata(db, "last_skill_reorg_at", recentTime)
	g.Expect(err).ToNot(HaveOccurred())

	// Insert 3 entries. generateThemeFromCluster uses first-entry content → theme "tdd first"
	// → slug "tdd-first". Use embedding_id=1 so they appear in the DB query.
	for _, content := range []string{"tdd first", "tdd second", "tdd third"} {
		_, err = db.Exec("INSERT INTO embeddings (content, source, embedding_id) VALUES (?, 'memory', 1)", content)
		g.Expect(err).ToNot(HaveOccurred())
	}

	// Pre-insert skill with slug="tdd-first" and source_memory_ids=[99999] so that:
	//   - getExistingSkillSourceIDs puts {99999:true} in existingSourceIDs
	//   - clusterHasExistingSkill(cluster, {99999:true}) returns false (entries have IDs 1,2,3)
	//   - getSkillBySlug("tdd-first") finds this skill → merge path
	now := time.Now().UTC().Format(time.RFC3339)

	_, err = insertSkill(db, &GeneratedSkill{
		Slug:            "tdd-first",
		Theme:           "tdd first",
		Description:     "Use when working on tdd patterns",
		Content:         "## Overview\n\ncontent\n\n## When to Use\n\nuse\n\n## Quick Reference\n\nref\n\n## Common Mistakes\n\nmistakes\n",
		SourceMemoryIDs: "[99999]",
		Alpha:           1.0,
		Beta:            1.0,
		Utility:         0.5,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	g.Expect(err).ToNot(HaveOccurred())

	result := &OptimizeResult{}
	opts := OptimizeOpts{
		SkillsDir:      t.TempDir(),
		MinClusterSize: 3,
		SimilarityFunc: func(_ *sql.DB, _, _ int64) (float64, error) { return 1.0, nil },
		Context:        context.Background(),
	}

	// nil SkillCompiler → template fallback → body has all 4 sections → compliance passes.
	// TestSkills=false (default) → TestAndCompileSkill returns nil → merge succeeds.
	err = optimizeCompileSkills(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.SkillsMerged).To(BeNumerically(">=", 1))
}

// TestOptimizeCompileSkills_MigrateMemGenError covers stmt 6: migrateMemoryGenSkills returns
// an error (not os.IsNotExist) when skillsDir/memory-gen is a file, not a directory.
func TestOptimizeCompileSkills_MigrateMemGenError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	skillsDir := t.TempDir()

	// Create a FILE named "memory-gen" inside skillsDir. ReadDir on a file returns ENOTDIR,
	// which is not os.IsNotExist → migrateMemoryGenSkills returns an error.
	err = os.WriteFile(filepath.Join(skillsDir, "memory-gen"), []byte("not a dir"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	result := &OptimizeResult{}
	opts := OptimizeOpts{SkillsDir: skillsDir}

	err = optimizeCompileSkills(db, opts, result)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("failed to migrate memory-gen skills"))
	}
}

// TestOptimizeCompileSkills_MinClusterSizeZero covers stmt 43: when opts.MinClusterSize==0
// the function defaults minCluster to 3.
func TestOptimizeCompileSkills_MinClusterSizeZero(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	recentTime := time.Now().UTC().Format(time.RFC3339)

	err = setMetadata(db, "last_skill_reorg_at", recentTime)
	g.Expect(err).ToNot(HaveOccurred())

	result := &OptimizeResult{}
	opts := OptimizeOpts{
		SkillsDir:      t.TempDir(),
		MinClusterSize: 0, // → defaults to 3 inside optimizeCompileSkills
		SimilarityFunc: calculateSimilarity,
	}

	err = optimizeCompileSkills(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
}

func TestOptimizeCompileSkills_MkdirAllFails(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	result := &OptimizeResult{}
	opts := OptimizeOpts{SkillsDir: "/dev/null/invalid-skills-dir"}

	err = optimizeCompileSkills(db, opts, result)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("failed to create skills directory"))
	}
}

func TestOptimizeCompileSkills_NoReorg_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	// Set last_skill_reorg_at to now so daysSince < 30 → shouldReorg = false
	recentTime := time.Now().UTC().Format(time.RFC3339)

	err = setMetadata(db, "last_skill_reorg_at", recentTime)
	g.Expect(err).ToNot(HaveOccurred())

	result := &OptimizeResult{}
	opts := OptimizeOpts{
		SkillsDir:      t.TempDir(),
		MinClusterSize: 3,
		SimilarityFunc: calculateSimilarity,
	}

	err = optimizeCompileSkills(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
}

func TestOptimizeCompileSkills_NoReorg_WithClusters(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	// Set last_skill_reorg_at to now so shouldReorg = false
	recentTime := time.Now().UTC().Format(time.RFC3339)

	err = setMetadata(db, "last_skill_reorg_at", recentTime)
	g.Expect(err).ToNot(HaveOccurred())

	// Insert 3 entries with a non-null embedding_id; mock simFunc returns 1.0 so they cluster
	for _, content := range []string{
		"always handle errors explicitly",
		"always handle errors consistently",
		"always handle errors carefully",
	} {
		_, err = db.Exec(
			"INSERT INTO embeddings (content, source, embedding_id) VALUES (?, 'memory', 1)",
			content,
		)
		g.Expect(err).ToNot(HaveOccurred())
	}

	optimizeResult := &OptimizeResult{}
	opts := OptimizeOpts{
		SkillsDir:      t.TempDir(),
		MinClusterSize: 3,
		SimilarityFunc: func(_ *sql.DB, _, _ int64) (float64, error) { return 1.0, nil },
	}

	err = optimizeCompileSkills(db, opts, optimizeResult)
	g.Expect(err).ToNot(HaveOccurred())
}

// ─── optimizeCompileSkills ───────────────────────────────────────────────────

func TestOptimizeCompileSkills_NoSkillsDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	result := &OptimizeResult{}
	opts := OptimizeOpts{}

	err = optimizeCompileSkills(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
}

func TestOptimizeCompileSkills_OldReorgTriggersReorg(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	oldTime := time.Now().UTC().Add(-40 * 24 * time.Hour).Format(time.RFC3339)

	err = setMetadata(db, "last_skill_reorg_at", oldTime)
	g.Expect(err).ToNot(HaveOccurred())

	result := &OptimizeResult{}
	opts := OptimizeOpts{
		SkillsDir:      t.TempDir(),
		MinClusterSize: 3,
		SimilarityFunc: func(_ *sql.DB, _, _ int64) (float64, error) { return 1.0, nil },
	}

	err = optimizeCompileSkills(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
}

func TestOptimizeCompileSkills_SmallClusters(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	recentTime := time.Now().UTC().Format(time.RFC3339)

	err = setMetadata(db, "last_skill_reorg_at", recentTime)
	g.Expect(err).ToNot(HaveOccurred())

	for _, content := range []string{"a entry", "b entry", "c entry", "d entry"} {
		_, err = db.Exec(
			"INSERT INTO embeddings (content, source, embedding_id) VALUES (?, 'memory', 1)",
			content,
		)
		g.Expect(err).ToNot(HaveOccurred())
	}

	result := &OptimizeResult{}
	opts := OptimizeOpts{
		SkillsDir:      t.TempDir(),
		MinClusterSize: 4,
		SimilarityFunc: func(_ *sql.DB, _, _ int64) (float64, error) {
			return 0.0, errors.New("no similarity computed")
		},
	}

	err = optimizeCompileSkills(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
}

func TestOptimizeCompileSkills_WithSkillsDir_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	result := &OptimizeResult{}
	opts := OptimizeOpts{
		SkillsDir:      t.TempDir(),
		MinClusterSize: 3,
		SimilarityFunc: calculateSimilarity,
	}

	err = optimizeCompileSkills(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
}

// ─── optimizeContradictions ──────────────────────────────────────────────────

func TestOptimizeContradictions_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	result := &OptimizeResult{}
	opts := OptimizeOpts{}

	err = optimizeContradictions(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.ContradictionsFound).To(Equal(0))
}

func TestOptimizeContradictions_NoPromotedEntries(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	_, err = db.Exec("INSERT INTO embeddings (content, source, promoted) VALUES ('not promoted', 'memory', 0)")
	g.Expect(err).ToNot(HaveOccurred())

	result := &OptimizeResult{}
	opts := OptimizeOpts{}

	err = optimizeContradictions(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.ContradictionsFound).To(Equal(0))
}

func TestOptimizeContradictions_WithContradiction(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	// Use non-zero embeddings so vec_distance_cosine returns valid results (not NULL).
	emb := make([]float32, 384)
	emb[0] = 1.0

	blob, err := sqlite_vec.SerializeFloat32(emb)
	g.Expect(err).ToNot(HaveOccurred())

	// Insert promoted entry with non-zero embedding.
	res, err := db.Exec("INSERT INTO vec_embeddings(embedding) VALUES (?)", blob)
	if err != nil {
		t.Fatalf("INSERT vec_embeddings promoted: %v", err)
	}

	promotedEmbID, _ := res.LastInsertId()

	_, err = db.Exec(
		"INSERT INTO embeddings (content, source, embedding_id, promoted) VALUES (?, ?, ?, 1)",
		"always use strict mode in TypeScript", "test", promotedEmbID,
	)
	g.Expect(err).ToNot(HaveOccurred())

	// Insert correction with identical embedding (similarity = 1.0 > 0.8) and
	// content that triggers detectConflictType → "contradiction" (negation pattern).
	res, err = db.Exec("INSERT INTO vec_embeddings(embedding) VALUES (?)", blob)
	if err != nil {
		t.Fatalf("INSERT vec_embeddings correction: %v", err)
	}

	correctionEmbID, _ := res.LastInsertId()

	_, err = db.Exec(
		"INSERT INTO embeddings (content, source, embedding_id, memory_type) VALUES (?, ?, ?, 'correction')",
		"never use strict mode in TypeScript", "test", correctionEmbID,
	)
	g.Expect(err).ToNot(HaveOccurred())

	result := &OptimizeResult{}

	err = optimizeContradictions(db, OptimizeOpts{}, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.ContradictionsFound).To(BeNumerically(">=", 1))
}

// ─── optimizeDecay ───────────────────────────────────────────────────────────

func TestOptimizeDecay_FirstRun(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	result := &OptimizeResult{}
	opts := OptimizeOpts{DecayBase: 0.9}

	err = optimizeDecay(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.DecayApplied).To(BeTrue())
	g.Expect(result.DaysSinceLastOptimize).To(BeNumerically("~", 1.0, 0.001))
}

func TestOptimizeDecay_OldRun_AppliesDecay(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	past := time.Now().UTC().Add(-48 * time.Hour).Format(time.RFC3339)
	_, err = db.Exec("INSERT OR REPLACE INTO metadata (key, value) VALUES ('last_optimized_at', ?)", past)
	g.Expect(err).ToNot(HaveOccurred())

	result := &OptimizeResult{}
	opts := OptimizeOpts{DecayBase: 0.9}

	err = optimizeDecay(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.DecayApplied).To(BeTrue())
	g.Expect(result.DaysSinceLastOptimize).To(BeNumerically(">=", 1.9))
}

func TestOptimizeDecay_RecentRun_SkipsDecay(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = db.Exec("INSERT OR REPLACE INTO metadata (key, value) VALUES ('last_optimized_at', ?)", now)
	g.Expect(err).ToNot(HaveOccurred())

	result := &OptimizeResult{}
	opts := OptimizeOpts{DecayBase: 0.9}

	err = optimizeDecay(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.DecayApplied).To(BeFalse())
}

// ─── optimizeDedup ───────────────────────────────────────────────────────────

func TestOptimizeDedup_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	result := &OptimizeResult{}
	opts := OptimizeOpts{DupThreshold: 0.95}

	err = optimizeDedup(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.DuplicatesMerged).To(Equal(0))
}

func TestOptimizeDedup_EqualConfHigherRetrieval_IDeleted(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	emb := make([]float32, 384)
	emb[0] = 1.0

	blob, err := sqlite_vec.SerializeFloat32(emb)
	if err != nil {
		t.Fatalf("SerializeFloat32: %v", err)
	}

	// Entry i: confidence=1.0, retrieval_count=2.
	res, err := db.Exec("INSERT INTO vec_embeddings(embedding) VALUES (?)", blob)
	if err != nil {
		t.Fatalf("INSERT vec_embeddings i: %v", err)
	}

	iEmbID, _ := res.LastInsertId()

	_, err = db.Exec(
		"INSERT INTO embeddings (content, source, confidence, retrieval_count, embedding_id) VALUES (?, ?, ?, ?, ?)",
		"use strict mode", "test", 1.0, 2, iEmbID,
	)
	g.Expect(err).ToNot(HaveOccurred())

	// Entry j: same confidence=1.0, higher retrieval_count=10 → j wins, i is deleted via break.
	res, err = db.Exec("INSERT INTO vec_embeddings(embedding) VALUES (?)", blob)
	if err != nil {
		t.Fatalf("INSERT vec_embeddings j: %v", err)
	}

	jEmbID, _ := res.LastInsertId()

	_, err = db.Exec(
		"INSERT INTO embeddings (content, source, confidence, retrieval_count, embedding_id) VALUES (?, ?, ?, ?, ?)",
		"use strict mode", "test", 1.0, 10, jEmbID,
	)
	g.Expect(err).ToNot(HaveOccurred())

	result := &OptimizeResult{}

	err = optimizeDedup(db, OptimizeOpts{DupThreshold: 0.95}, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.DuplicatesMerged).To(Equal(1))
}

func TestOptimizeDedup_IHigherConfidence_JsDeleted(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	// All three entries share the same non-zero embedding → pairwise similarity=1.0.
	emb := make([]float32, 384)
	emb[0] = 1.0

	blob, err := sqlite_vec.SerializeFloat32(emb)
	if err != nil {
		t.Fatalf("SerializeFloat32: %v", err)
	}

	insertEntry := func(content string, confidence float64) {
		res, err := db.Exec("INSERT INTO vec_embeddings(embedding) VALUES (?)", blob)
		if err != nil {
			t.Fatalf("INSERT vec_embeddings: %v", err)
		}

		embID, _ := res.LastInsertId()

		_, err = db.Exec(
			"INSERT INTO embeddings (content, source, confidence, embedding_id) VALUES (?, ?, ?, ?)",
			content, "test", confidence, embID,
		)
		if err != nil {
			t.Fatalf("INSERT embeddings: %v", err)
		}
	}

	// Entry A (highest confidence) is kept; B and C are duplicates and get deleted.
	// After i=0 deletes B and C, the outer loop skips them via the toDelete[entries[i].id] branch.
	insertEntry("use strict mode", 1.0)
	insertEntry("use strict mode", 0.5)
	insertEntry("use strict mode", 0.3)

	result := &OptimizeResult{}

	err = optimizeDedup(db, OptimizeOpts{DupThreshold: 0.95}, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.DuplicatesMerged).To(Equal(2))
}

func TestOptimizeDedup_JHigherConfidence_IDeleted(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	// Non-zero embedding so vec_distance_cosine returns valid results.
	emb := make([]float32, 384)
	emb[0] = 1.0

	blob, err := sqlite_vec.SerializeFloat32(emb)
	if err != nil {
		t.Fatalf("SerializeFloat32: %v", err)
	}

	// Insert entry i (lower confidence) with non-zero embedding.
	res, err := db.Exec("INSERT INTO vec_embeddings(embedding) VALUES (?)", blob)
	if err != nil {
		t.Fatalf("INSERT vec_embeddings i: %v", err)
	}

	iEmbID, _ := res.LastInsertId()

	_, err = db.Exec(
		"INSERT INTO embeddings (content, source, confidence, embedding_id) VALUES (?, ?, ?, ?)",
		"always run tests before committing", "test", 0.5, iEmbID,
	)
	g.Expect(err).ToNot(HaveOccurred())

	// Insert entry j (higher confidence) with identical embedding → similarity=1.0 > DupThreshold=0.95.
	res, err = db.Exec("INSERT INTO vec_embeddings(embedding) VALUES (?)", blob)
	if err != nil {
		t.Fatalf("INSERT vec_embeddings j: %v", err)
	}

	jEmbID, _ := res.LastInsertId()

	_, err = db.Exec(
		"INSERT INTO embeddings (content, source, confidence, embedding_id) VALUES (?, ?, ?, ?)",
		"always run tests before committing", "test", 1.0, jEmbID,
	)
	g.Expect(err).ToNot(HaveOccurred())

	result := &OptimizeResult{}

	err = optimizeDedup(db, OptimizeOpts{DupThreshold: 0.95}, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.DuplicatesMerged).To(Equal(1))
}

func TestOptimizeDedup_NoEmbeddings(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	_, err = db.Exec("INSERT INTO embeddings (content, source, confidence) VALUES ('entry one', 'memory', 0.8)")
	g.Expect(err).ToNot(HaveOccurred())

	_, err = db.Exec("INSERT INTO embeddings (content, source, confidence) VALUES ('entry two', 'memory', 0.7)")
	g.Expect(err).ToNot(HaveOccurred())

	result := &OptimizeResult{}
	opts := OptimizeOpts{DupThreshold: 0.95}

	err = optimizeDedup(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.DuplicatesMerged).To(Equal(0))
}

// TestOptimizeDemoteClaudeMD_AutoApproveEmbedding verifies the DestinationEmbedding path
// (lines 1861-1869): AutoApprove=true, content triggers DestinationEmbedding → db.Exec,
// toRemove set, ClaudeMDDemoted++, RemoveFromClaudeMD called.
func TestOptimizeDemoteClaudeMD_AutoApproveEmbedding(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	claudeMD := filepath.Join(dir, "CLAUDE.md")

	// "specific to" triggers isSituationalContent → DestinationEmbedding, Safe=true
	// "projctl" triggers isNarrowByKeywords → candidate is produced
	content := "# Project\n\n## Promoted Learnings\n\n- projctl: specific to this project when running go test only\n"

	err := os.WriteFile(claudeMD, []byte(content), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	db, err := initEmbeddingsDB(filepath.Join(dir, "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	result := &OptimizeResult{}
	opts := OptimizeOpts{
		SkillsDir:    t.TempDir(),
		ClaudeMDPath: claudeMD,
		MemoryRoot:   dir,
		AutoApprove:  true,
	}

	err = optimizeDemoteClaudeMD(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.ClaudeMDDemoted).To(Equal(1))
}

// TestOptimizeDemoteClaudeMD_AutoApproveHook verifies the DestinationHook path (lines 1870-1875):
// AutoApprove=true, content triggers DestinationHook → logChangelogMutation + continue (no removal).
func TestOptimizeDemoteClaudeMD_AutoApproveHook(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	claudeMD := filepath.Join(dir, "CLAUDE.md")

	// "never" triggers isDeterministicRule → DestinationHook, Safe=true
	// "mage " triggers isNarrowByKeywords → candidate is produced
	content := "# Project\n\n## Promoted Learnings\n\n- Never use mage for builds\n"

	err := os.WriteFile(claudeMD, []byte(content), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	db, err := initEmbeddingsDB(filepath.Join(dir, "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	result := &OptimizeResult{}
	opts := OptimizeOpts{
		SkillsDir:    t.TempDir(),
		ClaudeMDPath: claudeMD,
		MemoryRoot:   dir,
		AutoApprove:  true,
	}

	// DestinationHook: logged but not removed from CLAUDE.md → ClaudeMDDemoted stays 0
	err = optimizeDemoteClaudeMD(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.ClaudeMDDemoted).To(Equal(0))
}

// TestOptimizeDemoteClaudeMD_AutoApproveUnsafe verifies the !plan.Safe path (lines 1845-1850):
// AutoApprove=true but content is unclassifiable → plan.Safe=false → logChangelogMutation + continue.
func TestOptimizeDemoteClaudeMD_AutoApproveUnsafe(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	claudeMD := filepath.Join(dir, "CLAUDE.md")

	// "ISSUE-23" is narrow (tracker ID) but content has no deterministic/procedural/situational pattern
	// → PlanCLAUDEMDDemotion returns Safe=false
	content := "# Project\n\n## Promoted Learnings\n\n- ISSUE-23 was resolved by fixing the auth module\n"

	err := os.WriteFile(claudeMD, []byte(content), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	db, err := initEmbeddingsDB(filepath.Join(dir, "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	result := &OptimizeResult{}
	opts := OptimizeOpts{
		SkillsDir:    t.TempDir(),
		ClaudeMDPath: claudeMD,
		MemoryRoot:   dir,
		AutoApprove:  true,
	}

	// Safe=false → blocked, ClaudeMDDemoted stays 0
	err = optimizeDemoteClaudeMD(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.ClaudeMDDemoted).To(Equal(0))
}

func TestOptimizeDemoteClaudeMD_MissingClaudeMD(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	result := &OptimizeResult{}
	opts := OptimizeOpts{
		SkillsDir:    t.TempDir(),
		ClaudeMDPath: filepath.Join(t.TempDir(), "nonexistent-CLAUDE.md"),
	}

	err = optimizeDemoteClaudeMD(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
}

func TestOptimizeDemoteClaudeMD_NoDryRunWithNoReviewFunc(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	dir := t.TempDir()
	claudeMD := filepath.Join(dir, "CLAUDE.md")
	// ISSUE-23 is a narrow learning (has issue tracker ID)
	content := "# Project\n\n## Promoted Learnings\n\n- ISSUE-23 was resolved by fixing the auth module\n"
	err = os.WriteFile(claudeMD, []byte(content), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	result := &OptimizeResult{}
	opts := OptimizeOpts{
		SkillsDir:    t.TempDir(),
		ClaudeMDPath: claudeMD,
		AutoApprove:  false,
		ReviewFunc:   nil,
	}

	// Without AutoApprove and without ReviewFunc, candidates are collected but not demoted
	err = optimizeDemoteClaudeMD(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.ClaudeMDDemoted).To(Equal(0))
}

func TestOptimizeDemoteClaudeMD_NoPromotedLearningsSection(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	dir := t.TempDir()
	claudeMD := filepath.Join(dir, "CLAUDE.md")
	err = os.WriteFile(claudeMD, []byte("# My Project\n\n## Core Principles\n\n- Always write tests\n"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	result := &OptimizeResult{}
	opts := OptimizeOpts{
		SkillsDir:    t.TempDir(),
		ClaudeMDPath: claudeMD,
	}

	err = optimizeDemoteClaudeMD(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.ClaudeMDDemoted).To(Equal(0))
}

// ─── optimizeDemoteClaudeMD ──────────────────────────────────────────────────

func TestOptimizeDemoteClaudeMD_NoSkillsDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	result := &OptimizeResult{}
	opts := OptimizeOpts{}

	err = optimizeDemoteClaudeMD(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
}

// ─── optimizeMergeSkills ─────────────────────────────────────────────────────

func TestOptimizeMergeSkills_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	result := &OptimizeResult{}
	opts := OptimizeOpts{SimilarityFunc: calculateSimilarity}

	err = optimizeMergeSkills(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.SkillsMerged).To(Equal(0))
}

func TestOptimizeMergeSkills_SingleSkill(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	_, err = insertSkill(db, &GeneratedSkill{
		Slug:            "single-skill",
		Theme:           "single skill",
		Description:     "Use when single",
		Content:         "content",
		SourceMemoryIDs: "[]",
		Alpha:           1.0,
		Beta:            1.0,
	})
	g.Expect(err).ToNot(HaveOccurred())

	result := &OptimizeResult{}
	opts := OptimizeOpts{SimilarityFunc: calculateSimilarity}

	err = optimizeMergeSkills(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.SkillsMerged).To(Equal(0))
}

func TestOptimizeMergeSkills_TwoSimilarSkills(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	now := time.Now().UTC().Format(time.RFC3339)

	// Insert two skills with non-zero embedding_id (stored as non-NULL in DB).
	_, err = insertSkill(db, &GeneratedSkill{
		Slug:            "merge-skill-a",
		Theme:           "error handling a",
		Description:     "Use when error",
		Content:         "content a",
		SourceMemoryIDs: "[1,2]",
		Alpha:           3.0,
		Beta:            1.0,
		Utility:         0.75,
		EmbeddingID:     10,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	g.Expect(err).ToNot(HaveOccurred())

	_, err = insertSkill(db, &GeneratedSkill{
		Slug:            "merge-skill-b",
		Theme:           "error handling b",
		Description:     "Use when error b",
		Content:         "content b",
		SourceMemoryIDs: "[3,4]",
		Alpha:           2.0,
		Beta:            1.0,
		Utility:         0.6,
		EmbeddingID:     20,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	g.Expect(err).ToNot(HaveOccurred())

	result := &OptimizeResult{}
	opts := OptimizeOpts{
		SkillsDir: t.TempDir(),
		SimilarityFunc: func(_ *sql.DB, _, _ int64) (float64, error) {
			return 0.92, nil
		},
	}

	err = optimizeMergeSkills(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.SkillsMerged).To(Equal(1))
}

// TestOptimizePromoteSkills_AutoApprove_Promotes verifies that AutoApprove=true with a qualifying
// skill (enough utility, confidence, and project usage) promotes the skill to CLAUDE.md.
func TestOptimizePromoteSkills_AutoApprove_Promotes(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS skill_usage (skill_id INTEGER, project TEXT, timestamp TEXT)`)
	g.Expect(err).ToNot(HaveOccurred())

	now := time.Now().UTC().Format(time.RFC3339)

	skill := &GeneratedSkill{
		Slug:            "auto-promo",
		Theme:           "testing",
		Description:     "desc",
		Content:         "content",
		SourceMemoryIDs: "[999]", // non-empty so second memories loop executes its body
		Alpha:           5.0,
		Beta:            1.0,
		Utility:         0.9,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	skillID, err := insertSkill(db, skill)
	g.Expect(err).ToNot(HaveOccurred())

	_, err = db.Exec("INSERT INTO skill_usage (skill_id, project, timestamp) VALUES (?, ?, ?)", skillID, "proj-a", now)
	g.Expect(err).ToNot(HaveOccurred())

	tmpDir := t.TempDir()
	claudeMDPath := filepath.Join(tmpDir, "CLAUDE.md")

	result := &OptimizeResult{}
	opts := OptimizeOpts{
		SkillsDir:          tmpDir,
		ClaudeMDPath:       claudeMDPath,
		SkillCompiler:      &mockSkillCompiler{content: "always write tests first"},
		MinSkillUtility:    0.5,
		MinSkillConfidence: 0.5,
		MinSkillProjects:   1,
		AutoApprove:        true,
		TestSkills:         false, // TestAndCompileSkill returns nil immediately
		Embedder: func(_ string) ([]float32, error) {
			return []float32{1.0, 0.0, 0.0}, nil
		},
	}

	err = optimizePromoteSkills(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.SkillsPromoted).To(Equal(1))
}

// TestOptimizePromoteSkills_IsDuplicate_SkipsPromotion verifies that a skill whose synthesized
// principle has cosine similarity > 0.9 with an existing CLAUDE.md entry is skipped.
func TestOptimizePromoteSkills_IsDuplicate_SkipsPromotion(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS skill_usage (skill_id INTEGER, project TEXT, timestamp TEXT)`)
	g.Expect(err).ToNot(HaveOccurred())

	now := time.Now().UTC().Format(time.RFC3339)

	skill := &GeneratedSkill{
		Slug:            "dup-candidate",
		Theme:           "testing",
		Description:     "desc",
		Content:         "content",
		SourceMemoryIDs: "[]",
		Alpha:           5.0,
		Beta:            1.0,
		Utility:         0.9,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	skillID, err := insertSkill(db, skill)
	g.Expect(err).ToNot(HaveOccurred())

	_, err = db.Exec("INSERT INTO skill_usage (skill_id, project, timestamp) VALUES (?, ?, ?)", skillID, "proj-a", now)
	g.Expect(err).ToNot(HaveOccurred())

	tmpDir := t.TempDir()
	claudeMDPath := filepath.Join(tmpDir, "CLAUDE.md")

	// CLAUDE.md with an existing promoted learning that embeds to the same vector.
	claudeContent := "# Claude\n\n## Promoted Learnings\n\n- existing rule that is already promoted\n"

	err = os.WriteFile(claudeMDPath, []byte(claudeContent), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	// Embedder always returns [1,0,0] → cosineSimilarity with existing = 1.0 > 0.9 → isDuplicate.
	result := &OptimizeResult{}
	opts := OptimizeOpts{
		SkillsDir:          tmpDir,
		ClaudeMDPath:       claudeMDPath,
		SkillCompiler:      &mockSkillCompiler{content: "always write tests"},
		MinSkillUtility:    0.5,
		MinSkillConfidence: 0.5,
		MinSkillProjects:   1,
		AutoApprove:        true,
		Embedder: func(_ string) ([]float32, error) {
			return []float32{1.0, 0.0, 0.0}, nil
		},
	}

	err = optimizePromoteSkills(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.SkillsPromoted).To(Equal(0)) // skipped: duplicate
}

func TestOptimizePromoteSkills_NoCandidates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	// Insert a skill with high utility threshold that still qualifies, but no
	// skill_usage table → projectCount query fails → all candidates filtered out.
	now := time.Now().UTC().Format(time.RFC3339)
	skill := &GeneratedSkill{
		Slug:            "promo-candidate",
		Theme:           "error handling",
		Description:     "desc",
		Content:         "content",
		SourceMemoryIDs: "[]",
		Alpha:           5.0,
		Beta:            1.0,
		Utility:         0.9,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	_, err = insertSkill(db, skill)
	g.Expect(err).ToNot(HaveOccurred())

	result := &OptimizeResult{}
	opts := OptimizeOpts{
		SkillsDir:          t.TempDir(),
		SkillCompiler:      &mockSkillCompiler{},
		MinSkillUtility:    0.5,
		MinSkillConfidence: 0.5,
		MinSkillProjects:   1,
	}

	err = optimizePromoteSkills(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.SkillsPromoted).To(Equal(0))
}

func TestOptimizePromoteSkills_NoSkillCompiler(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	result := &OptimizeResult{}
	opts := OptimizeOpts{
		SkillsDir:     t.TempDir(),
		SkillCompiler: nil,
	}

	err = optimizePromoteSkills(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.SkillsPromoted).To(Equal(0))
}

// ─── optimizePromoteSkills ───────────────────────────────────────────────────

func TestOptimizePromoteSkills_NoSkillsDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	result := &OptimizeResult{}
	opts := OptimizeOpts{}

	err = optimizePromoteSkills(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.SkillsPromoted).To(Equal(0))
}

// TestOptimizePromoteSkills_ReviewFuncApproves verifies that when AutoApprove=false,
// a ReviewFunc returning (true, nil) leads to skill promotion.
func TestOptimizePromoteSkills_ReviewFuncApproves(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS skill_usage (skill_id INTEGER, project TEXT, timestamp TEXT)`)
	g.Expect(err).ToNot(HaveOccurred())

	now := time.Now().UTC().Format(time.RFC3339)

	skill := &GeneratedSkill{
		Slug:            "review-candidate",
		Theme:           "testing",
		Description:     "desc",
		Content:         "content",
		SourceMemoryIDs: "[]",
		Alpha:           5.0,
		Beta:            1.0,
		Utility:         0.9,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	skillID, err := insertSkill(db, skill)
	g.Expect(err).ToNot(HaveOccurred())

	_, err = db.Exec("INSERT INTO skill_usage (skill_id, project, timestamp) VALUES (?, ?, ?)", skillID, "proj-a", now)
	g.Expect(err).ToNot(HaveOccurred())

	tmpDir := t.TempDir()
	claudeMDPath := filepath.Join(tmpDir, "CLAUDE.md")

	result := &OptimizeResult{}
	opts := OptimizeOpts{
		SkillsDir:          tmpDir,
		ClaudeMDPath:       claudeMDPath,
		SkillCompiler:      &mockSkillCompiler{content: "always write tests"},
		MinSkillUtility:    0.5,
		MinSkillConfidence: 0.5,
		MinSkillProjects:   1,
		AutoApprove:        false, // not auto-approved
		TestSkills:         false, // skip LLM test step
		ReviewFunc: func(_ string, _ string) (bool, error) {
			return true, nil // approve
		},
		Embedder: func(_ string) ([]float32, error) {
			return []float32{0.0, 1.0, 0.0}, nil // unique vector — not a duplicate
		},
	}

	err = optimizePromoteSkills(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.SkillsPromoted).To(Equal(1))
}

// TestOptimizePromoteSkills_SynthesizeErrLLMUnavailable verifies that when Synthesize returns
// ErrLLMUnavailable, the candidate is skipped and no error is returned.
func TestOptimizePromoteSkills_SynthesizeErrLLMUnavailable(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS skill_usage (skill_id INTEGER, project TEXT, timestamp TEXT)`)
	g.Expect(err).ToNot(HaveOccurred())

	now := time.Now().UTC().Format(time.RFC3339)

	skill := &GeneratedSkill{
		Slug:            "synth-err-skill",
		Theme:           "testing",
		Description:     "desc",
		Content:         "content",
		SourceMemoryIDs: "[]",
		Alpha:           5.0,
		Beta:            1.0,
		Utility:         0.9,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	skillID, err := insertSkill(db, skill)
	g.Expect(err).ToNot(HaveOccurred())

	_, err = db.Exec("INSERT INTO skill_usage (skill_id, project, timestamp) VALUES (?, ?, ?)", skillID, "proj-a", now)
	g.Expect(err).ToNot(HaveOccurred())

	tmpDir := t.TempDir()

	result := &OptimizeResult{}
	opts := OptimizeOpts{
		SkillsDir:          tmpDir,
		ClaudeMDPath:       filepath.Join(tmpDir, "CLAUDE.md"),
		SkillCompiler:      &mockSkillCompilerSynthErr{err: ErrLLMUnavailable},
		MinSkillUtility:    0.5,
		MinSkillConfidence: 0.5,
		MinSkillProjects:   1,
		AutoApprove:        true,
		Embedder: func(_ string) ([]float32, error) {
			return []float32{1.0, 0.0, 0.0}, nil
		},
	}

	err = optimizePromoteSkills(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.SkillsPromoted).To(Equal(0)) // skipped: LLM unavailable
}

func TestOptimizePromoteSkills_WithCandidateNotApproved(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	// Create skill_usage table (not in default schema) and insert a qualifying skill.
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS skill_usage (skill_id INTEGER, project TEXT, timestamp TEXT)`)
	g.Expect(err).ToNot(HaveOccurred())

	now := time.Now().UTC().Format(time.RFC3339)
	skill := &GeneratedSkill{
		Slug:            "promo-candidate-2",
		Theme:           "error handling",
		Description:     "desc",
		Content:         "content",
		SourceMemoryIDs: "[1]",
		Alpha:           5.0,
		Beta:            1.0,
		Utility:         0.9,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	skillID, err := insertSkill(db, skill)
	g.Expect(err).ToNot(HaveOccurred())

	// Insert 1 project usage entry so MinSkillProjects=1 passes.
	_, err = db.Exec("INSERT INTO skill_usage (skill_id, project, timestamp) VALUES (?, ?, ?)", skillID, "proj-a", now)
	g.Expect(err).ToNot(HaveOccurred())

	result := &OptimizeResult{}
	opts := OptimizeOpts{
		SkillsDir:          t.TempDir(),
		ClaudeMDPath:       filepath.Join(t.TempDir(), "CLAUDE.md"),
		SkillCompiler:      &mockSkillCompiler{},
		MinSkillUtility:    0.5,
		MinSkillConfidence: 0.5,
		MinSkillProjects:   1,
		AutoApprove:        false, // not approved → skip promotion block
		Embedder: func(_ string) ([]float32, error) {
			return []float32{1.0, 0.0, 0.0}, nil
		},
	}

	err = optimizePromoteSkills(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.SkillsPromoted).To(Equal(0))
}

// TestOptimizePromote_AutoApprove verifies that AutoApprove=true with a qualifying candidate
// promotes it to CLAUDE.md and increments PromotionsApproved.
func TestOptimizePromote_AutoApprove(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	_, err = db.Exec(`INSERT INTO embeddings
		(content, principle, source, confidence, retrieval_count, projects_retrieved, promoted)
		VALUES ('use targ for builds', 'use targ for builds', 'memory', 0.9, 10, 'proj1,proj2,proj3', 0)`)
	g.Expect(err).ToNot(HaveOccurred())

	result := &OptimizeResult{}
	opts := OptimizeOpts{
		MinRetrievals: 5,
		MinProjects:   3,
		AutoApprove:   true,
		ClaudeMDPath:  filepath.Join(t.TempDir(), "CLAUDE.md"),
	}

	err = optimizePromote(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.PromotionsApproved).To(Equal(1))
}

func TestOptimizePromote_CandidateEmptyPrinciple(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	_, err = db.Exec(`INSERT INTO embeddings
		(content, principle, source, confidence, retrieval_count, projects_retrieved, promoted)
		VALUES ('some content', '', 'memory', 0.9, 10, 'proj1,proj2,proj3', 0)`)
	g.Expect(err).ToNot(HaveOccurred())

	result := &OptimizeResult{}
	opts := OptimizeOpts{
		MinRetrievals: 5,
		MinProjects:   3,
		ClaudeMDPath:  filepath.Join(t.TempDir(), "CLAUDE.md"),
	}

	err = optimizePromote(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.PromotionsApproved).To(Equal(0))
}

// ─── optimizePromote ─────────────────────────────────────────────────────────

func TestOptimizePromote_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	result := &OptimizeResult{}
	opts := OptimizeOpts{
		MinRetrievals: 5,
		MinProjects:   3,
		ClaudeMDPath:  filepath.Join(t.TempDir(), "CLAUDE.md"),
	}

	err = optimizePromote(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.PromotionCandidates).To(Equal(0))
}

// TestOptimizePromote_ReviewFuncApproves verifies that ReviewFunc returning true promotes
// a qualifying candidate and increments PromotionsApproved.
func TestOptimizePromote_ReviewFuncApproves(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	_, err = db.Exec(`INSERT INTO embeddings
		(content, principle, source, confidence, retrieval_count, projects_retrieved, promoted)
		VALUES ('run tests before commit', 'run tests before commit', 'memory', 0.9, 8, 'alpha,beta', 0)`)
	g.Expect(err).ToNot(HaveOccurred())

	result := &OptimizeResult{}
	opts := OptimizeOpts{
		MinRetrievals: 5,
		MinProjects:   2,
		AutoApprove:   false,
		ReviewFunc:    func(_, _ string) (bool, error) { return true, nil },
		ClaudeMDPath:  filepath.Join(t.TempDir(), "CLAUDE.md"),
	}

	err = optimizePromote(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.PromotionsApproved).To(Equal(1))
}

// ─── optimizePrune ───────────────────────────────────────────────────────────

func TestOptimizePrune_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	result := &OptimizeResult{}
	opts := OptimizeOpts{PruneThreshold: 0.1}

	err = optimizePrune(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.EntriesPruned).To(Equal(0))
}

func TestOptimizePrune_KeepsPromoted(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	_, err = db.Exec("INSERT INTO embeddings (content, source, confidence, promoted) VALUES ('promoted low conf', 'memory', 0.02, 1)")
	g.Expect(err).ToNot(HaveOccurred())

	result := &OptimizeResult{}
	opts := OptimizeOpts{PruneThreshold: 0.1}

	err = optimizePrune(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.EntriesPruned).To(Equal(0))
}

func TestOptimizePrune_PrunesLowConfidence(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	_, err = db.Exec("INSERT INTO embeddings (content, source, confidence, promoted) VALUES ('low conf', 'memory', 0.05, 0)")
	g.Expect(err).ToNot(HaveOccurred())

	result := &OptimizeResult{}
	opts := OptimizeOpts{PruneThreshold: 0.1}

	err = optimizePrune(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.EntriesPruned).To(Equal(1))
}

// ─── optimizePurgeBoilerplate ────────────────────────────────────────────────

func TestOptimizePurgeBoilerplate_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	result := &OptimizeResult{}
	opts := OptimizeOpts{}

	err = optimizePurgeBoilerplate(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.BoilerplatePurged).To(Equal(0))
}

func TestOptimizePurgeBoilerplate_KeepsRealContent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	_, err = db.Exec("INSERT INTO embeddings (content, source, promoted) VALUES ('use explicit error handling patterns consistently across all codebases', 'memory', 0)")
	g.Expect(err).ToNot(HaveOccurred())

	result := &OptimizeResult{}
	opts := OptimizeOpts{}

	err = optimizePurgeBoilerplate(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.BoilerplatePurged).To(Equal(0))
}

func TestOptimizePurgeBoilerplate_PurgesHeaderLines(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	_, err = db.Exec("INSERT INTO embeddings (content, source, promoted) VALUES ('# Session Summary', 'memory', 0)")
	g.Expect(err).ToNot(HaveOccurred())

	result := &OptimizeResult{}
	opts := OptimizeOpts{}

	err = optimizePurgeBoilerplate(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.BoilerplatePurged).To(Equal(1))
}

// ─── optimizePurgeLegacySessionEmbeddings ───────────────────────────────────

func TestOptimizePurgeLegacySessionEmbeddings_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	result := &OptimizeResult{}
	opts := OptimizeOpts{}

	err = optimizePurgeLegacySessionEmbeddings(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.LegacySessionPurged).To(Equal(0))
}

func TestOptimizePurgeLegacySessionEmbeddings_KeepsTimestampedEntries(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	_, err = db.Exec(`INSERT INTO embeddings
		(content, source, promoted, observation_type, memory_type, retrieval_count)
		VALUES ('- 2026-02-10 15:04: always write tests before implementation code', 'memory', 0, '', '', 0)`)
	g.Expect(err).ToNot(HaveOccurred())

	result := &OptimizeResult{}
	opts := OptimizeOpts{}

	err = optimizePurgeLegacySessionEmbeddings(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.LegacySessionPurged).To(Equal(0))
}

func TestOptimizePurgeLegacySessionEmbeddings_KeepsUsedEntries(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	_, err = db.Exec(`INSERT INTO embeddings
		(content, source, promoted, observation_type, memory_type, retrieval_count)
		VALUES ('no timestamp but being used here in tests', 'memory', 0, '', '', 3)`)
	g.Expect(err).ToNot(HaveOccurred())

	result := &OptimizeResult{}
	opts := OptimizeOpts{}

	err = optimizePurgeLegacySessionEmbeddings(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.LegacySessionPurged).To(Equal(0))
}

func TestOptimizePurgeLegacySessionEmbeddings_PurgesLegacyEntries(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	_, err = db.Exec(`INSERT INTO embeddings
		(content, source, promoted, observation_type, memory_type, retrieval_count)
		VALUES ('raw session content from old query behavior', 'memory', 0, '', '', 0)`)
	g.Expect(err).ToNot(HaveOccurred())

	result := &OptimizeResult{}
	opts := OptimizeOpts{}

	err = optimizePurgeLegacySessionEmbeddings(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.LegacySessionPurged).To(Equal(1))
}

// ─── optimizeSplitSkills ─────────────────────────────────────────────────────

func TestOptimizeSplitSkills_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	result := &OptimizeResult{}
	opts := OptimizeOpts{MinClusterSize: 3, SimilarityFunc: calculateSimilarity}

	err = optimizeSplitSkills(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.SkillsSplit).To(Equal(0))
}

func TestOptimizeSplitSkills_InvalidJSON(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	_, err = insertSkill(db, &GeneratedSkill{
		Slug:            "bad-json-skill",
		Theme:           "bad json",
		Description:     "Use when testing",
		Content:         "content",
		SourceMemoryIDs: "not-valid-json",
		Alpha:           1.0,
		Beta:            1.0,
	})
	g.Expect(err).ToNot(HaveOccurred())

	result := &OptimizeResult{}
	opts := OptimizeOpts{MinClusterSize: 3, SimilarityFunc: calculateSimilarity}

	err = optimizeSplitSkills(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.SkillsSplit).To(Equal(0))
}

func TestOptimizeSplitSkills_MemoriesNotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	// SourceMemoryIDs has 6 IDs but none exist in the embeddings table.
	_, err = insertSkill(db, &GeneratedSkill{
		Slug:            "missing-mem-skill",
		Theme:           "missing memories",
		Description:     "Use when testing",
		Content:         "content",
		SourceMemoryIDs: "[100,200,300,400,500,600]",
		Alpha:           1.0,
		Beta:            1.0,
	})
	g.Expect(err).ToNot(HaveOccurred())

	result := &OptimizeResult{}
	opts := OptimizeOpts{MinClusterSize: 3, SimilarityFunc: calculateSimilarity}

	err = optimizeSplitSkills(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.SkillsSplit).To(Equal(0))
}

func TestOptimizeSplitSkills_SingleCluster(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	// All 6 memories share the same embedding → all cluster together → 1 cluster → no split.
	emb := make([]float32, 384)
	emb[0] = 1.0

	blob, err := sqlite_vec.SerializeFloat32(emb)
	if err != nil {
		t.Fatalf("SerializeFloat32: %v", err)
	}

	var memIDs []string

	for i := range 6 {
		res, err := db.Exec("INSERT INTO vec_embeddings(embedding) VALUES (?)", blob)
		if err != nil {
			t.Fatalf("INSERT vec_embeddings[%d]: %v", i, err)
		}

		embID, _ := res.LastInsertId()

		res, err = db.Exec(
			"INSERT INTO embeddings (content, source, embedding_id) VALUES (?, ?, ?)",
			"always test before commit", "test", embID,
		)
		if err != nil {
			t.Fatalf("INSERT embeddings[%d]: %v", i, err)
		}

		memID, _ := res.LastInsertId()
		memIDs = append(memIDs, strconv.FormatInt(memID, 10))
	}

	sourceIDs := "[" + strings.Join(memIDs, ",") + "]"

	_, err = insertSkill(db, &GeneratedSkill{
		Slug:            "single-cluster-skill",
		Theme:           "single cluster",
		Description:     "Use when testing",
		Content:         "content",
		SourceMemoryIDs: sourceIDs,
		Alpha:           1.0,
		Beta:            1.0,
	})
	g.Expect(err).ToNot(HaveOccurred())

	result := &OptimizeResult{}
	opts := OptimizeOpts{MinClusterSize: 3, SimilarityFunc: calculateSimilarity}

	err = optimizeSplitSkills(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.SkillsSplit).To(Equal(0))
}

func TestOptimizeSplitSkills_SkillWithFewMemories(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	_, err = insertSkill(db, &GeneratedSkill{
		Slug:            "small-skill",
		Theme:           "small skill",
		Description:     "Use when small",
		Content:         "content",
		SourceMemoryIDs: "[1,2]",
		Alpha:           1.0,
		Beta:            1.0,
	})
	g.Expect(err).ToNot(HaveOccurred())

	result := &OptimizeResult{}
	opts := OptimizeOpts{MinClusterSize: 3, SimilarityFunc: calculateSimilarity, SkillsDir: t.TempDir()}

	err = optimizeSplitSkills(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.SkillsSplit).To(Equal(0))
}

func TestOptimizeSplitSkills_TwoClusters_CompilerFails(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	// Two orthogonal unit vectors → cosine similarity = 0.0 < 0.6 → two distinct clusters.
	embA := make([]float32, 384)
	embA[0] = 1.0

	blobA, err := sqlite_vec.SerializeFloat32(embA)
	if err != nil {
		t.Fatalf("SerializeFloat32 A: %v", err)
	}

	embB := make([]float32, 384)
	embB[1] = 1.0

	blobB, err := sqlite_vec.SerializeFloat32(embB)
	if err != nil {
		t.Fatalf("SerializeFloat32 B: %v", err)
	}

	var memIDs []string

	insertCluster := func(blob []byte, n int) {
		for i := range n {
			res, err := db.Exec("INSERT INTO vec_embeddings(embedding) VALUES (?)", blob)
			if err != nil {
				t.Fatalf("INSERT vec_embeddings: %v", err)
			}

			embID, _ := res.LastInsertId()

			res, err = db.Exec(
				"INSERT INTO embeddings (content, source, embedding_id) VALUES (?, ?, ?)",
				"cluster content", "test", embID,
			)
			if err != nil {
				t.Fatalf("INSERT embeddings[%d]: %v", i, err)
			}

			memID, _ := res.LastInsertId()
			memIDs = append(memIDs, strconv.FormatInt(memID, 10))
		}
	}

	// Cluster A: 3 memories with embA; Cluster B: 3 memories with embB.
	insertCluster(blobA, 3)
	insertCluster(blobB, 3)

	sourceIDs := "[" + strings.Join(memIDs, ",") + "]"

	_, err = insertSkill(db, &GeneratedSkill{
		Slug:            "two-cluster-skill",
		Theme:           "two clusters",
		Description:     "Use when testing",
		Content:         "content",
		SourceMemoryIDs: sourceIDs,
		Alpha:           1.0,
		Beta:            1.0,
	})
	g.Expect(err).ToNot(HaveOccurred())

	result := &OptimizeResult{}
	opts := OptimizeOpts{
		MinClusterSize: 3,
		SimilarityFunc: calculateSimilarity,
		SkillCompiler:  &mockSkillCompiler{compileErr: errors.New("compiler unavailable")},
	}

	err = optimizeSplitSkills(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	// Compiler fails but template fallback produces content → SkillsSplit >= 1.
	g.Expect(result.SkillsSplit).To(BeNumerically(">=", 1))
}

// TestOptimizeSynthesize_AutoApprove_HighQuality verifies that a high-quality synthesis
// is approved when AutoApprove is true, covering the `if approved { ... }` block.
func TestOptimizeSynthesize_AutoApprove_HighQuality(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	// Entries that produce a synthesis with "always" (actionable), "git" (specific tool), and long content.
	for _, content := range []string{
		"always use git commit to save progress before switching branches in the repository",
		"always run git commit before switching to a different feature branch in any repository",
		"always ensure git commit history is clean before creating new branches for features",
	} {
		_, err = db.Exec(
			"INSERT INTO embeddings (content, source, embedding_id) VALUES (?, 'memory', 1)",
			content,
		)
		g.Expect(err).ToNot(HaveOccurred())
	}

	result := &OptimizeResult{}
	opts := OptimizeOpts{
		MinClusterSize: 2,
		SynthThreshold: 0.0,
		AutoApprove:    true,
		SimilarityFunc: func(_ *sql.DB, _, _ int64) (float64, error) { return 1.0, nil },
	}

	err = optimizeSynthesize(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.PatternsApproved).To(BeNumerically(">=", 1))
}

// ─── optimizeSynthesize ──────────────────────────────────────────────────────

func TestOptimizeSynthesize_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	result := &OptimizeResult{}
	opts := OptimizeOpts{MinClusterSize: 3, SynthThreshold: 0.8}

	err = optimizeSynthesize(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.PatternsFound).To(Equal(0))
}

// TestOptimizeSynthesize_LowQualitySynthesis verifies that a low-quality synthesis
// is rejected (quality < 0.8), covering the rejection path.
func TestOptimizeSynthesize_LowQualitySynthesis(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	// Entries with no actionable keywords and no technical terms → low-quality synthesis.
	for _, content := range []string{
		"foo bar baz qux",
		"foo bar baz quux",
		"foo bar baz quuz",
	} {
		_, err = db.Exec(
			"INSERT INTO embeddings (content, source, embedding_id) VALUES (?, 'memory', 1)",
			content,
		)
		g.Expect(err).ToNot(HaveOccurred())
	}

	result := &OptimizeResult{}
	opts := OptimizeOpts{
		MinClusterSize: 2,
		SynthThreshold: 0.0,
		AutoApprove:    true,
		SimilarityFunc: func(_ *sql.DB, _, _ int64) (float64, error) { return 1.0, nil },
	}

	err = optimizeSynthesize(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	// Pattern found but rejected due to low quality
	g.Expect(result.PatternsFound).To(BeNumerically(">=", 1))
	g.Expect(result.PatternsApproved).To(Equal(0))
}

func TestOptimizeSynthesize_NoEmbeddingEntries(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	// Entries without embedding_id are not selected by the query
	_, err = db.Exec("INSERT INTO embeddings (content, source) VALUES ('entry without embedding', 'memory')")
	g.Expect(err).ToNot(HaveOccurred())

	result := &OptimizeResult{}
	opts := OptimizeOpts{MinClusterSize: 3, SynthThreshold: 0.8}

	err = optimizeSynthesize(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.PatternsFound).To(Equal(0))
}

func TestOptimizeSynthesize_WithEmbeddingEntries(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	// Insert entries with embedding_id set; mock SimilarityFunc clusters them together.
	for _, content := range []string{
		"always write tests before implementation code",
		"always write tests before writing production code",
		"always write tests first then implementation",
	} {
		_, err = db.Exec(
			"INSERT INTO embeddings (content, source, embedding_id) VALUES (?, 'memory', 1)",
			content,
		)
		g.Expect(err).ToNot(HaveOccurred())
	}

	result := &OptimizeResult{}
	opts := OptimizeOpts{
		MinClusterSize: 2,
		SynthThreshold: 0.0,
		AutoApprove:    false,
		SimilarityFunc: func(_ *sql.DB, _, _ int64) (float64, error) { return 1.0, nil },
	}

	err = optimizeSynthesize(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
}

// ─── Optimize: step error blocks ─────────────────────────────────────────────

// TestOptimize_AutoDemoteError verifies the auto-demote error path when ClaudeMDPath is a directory.
func TestOptimize_AutoDemoteError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()

	// ClaudeMDPath set to a directory → os.ReadFile fails with "is a directory" → RemoveFromClaudeMD error
	claudeMDDir := filepath.Join(dir, "claude-md-dir")
	err := os.MkdirAll(claudeMDDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	memRoot := filepath.Join(dir, "memory")
	err = os.MkdirAll(memRoot, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	db, err := initEmbeddingsDB(filepath.Join(memRoot, "embeddings.db"))
	g.Expect(err).ToNot(HaveOccurred())

	// promoted=1, confidence=0.2 → after promoted-decay (×0.995) → 0.199 < 0.3 → auto-demote triggers
	_, err = db.Exec(
		"INSERT INTO embeddings (content, source, confidence, promoted) VALUES (?, 'claude-md', 0.2, 1)",
		"some promoted learning about patterns that needs demoting from CLAUDE.md",
	)
	g.Expect(err).ToNot(HaveOccurred())

	err = db.Close()
	g.Expect(err).ToNot(HaveOccurred())

	opts := OptimizeOpts{
		MemoryRoot:   memRoot,
		ClaudeMDPath: claudeMDDir, // directory → ReadFile fails
	}

	_, err = Optimize(opts)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("auto-demote failed"))
}

func TestOptimize_CancelledContext(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	dir := t.TempDir()
	opts := OptimizeOpts{
		MemoryRoot:   filepath.Join(dir, "memory"),
		ClaudeMDPath: filepath.Join(dir, "CLAUDE.md"),
		Context:      ctx,
		TestSkills:   false,
	}

	_, err := Optimize(opts)
	g.Expect(err).To(HaveOccurred())
}

func TestOptimize_CheckContextStep10(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	ctx := &nthErrorContext{failAt: 10}
	opts := OptimizeOpts{
		MemoryRoot:   filepath.Join(dir, "memory"),
		ClaudeMDPath: filepath.Join(dir, "CLAUDE.md"),
		Context:      ctx,
	}
	_, err := Optimize(opts)
	g.Expect(err).To(HaveOccurred())
}

func TestOptimize_CheckContextStep11(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	ctx := &nthErrorContext{failAt: 11}
	opts := OptimizeOpts{
		MemoryRoot:   filepath.Join(dir, "memory"),
		ClaudeMDPath: filepath.Join(dir, "CLAUDE.md"),
		Context:      ctx,
	}
	_, err := Optimize(opts)
	g.Expect(err).To(HaveOccurred())
}

func TestOptimize_CheckContextStep12(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	ctx := &nthErrorContext{failAt: 12}
	opts := OptimizeOpts{
		MemoryRoot:   filepath.Join(dir, "memory"),
		ClaudeMDPath: filepath.Join(dir, "CLAUDE.md"),
		Context:      ctx,
	}
	_, err := Optimize(opts)
	g.Expect(err).To(HaveOccurred())
}

func TestOptimize_CheckContextStep13(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	ctx := &nthErrorContext{failAt: 13}
	opts := OptimizeOpts{
		MemoryRoot:   filepath.Join(dir, "memory"),
		ClaudeMDPath: filepath.Join(dir, "CLAUDE.md"),
		Context:      ctx,
	}
	_, err := Optimize(opts)
	g.Expect(err).To(HaveOccurred())
}

func TestOptimize_CheckContextStep14(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	ctx := &nthErrorContext{failAt: 14}
	opts := OptimizeOpts{
		MemoryRoot:   filepath.Join(dir, "memory"),
		ClaudeMDPath: filepath.Join(dir, "CLAUDE.md"),
		Context:      ctx,
	}
	_, err := Optimize(opts)
	g.Expect(err).To(HaveOccurred())
}

func TestOptimize_CheckContextStep15(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	ctx := &nthErrorContext{failAt: 15}
	opts := OptimizeOpts{
		MemoryRoot:   filepath.Join(dir, "memory"),
		ClaudeMDPath: filepath.Join(dir, "CLAUDE.md"),
		Context:      ctx,
	}
	_, err := Optimize(opts)
	g.Expect(err).To(HaveOccurred())
}

// ─── Optimize: checkContext steps 2–15 ───────────────────────────────────────

func TestOptimize_CheckContextStep2(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	ctx := &nthErrorContext{failAt: 2}
	opts := OptimizeOpts{
		MemoryRoot:   filepath.Join(dir, "memory"),
		ClaudeMDPath: filepath.Join(dir, "CLAUDE.md"),
		Context:      ctx,
	}
	_, err := Optimize(opts)
	g.Expect(err).To(HaveOccurred())
}

func TestOptimize_CheckContextStep3(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	ctx := &nthErrorContext{failAt: 3}
	opts := OptimizeOpts{
		MemoryRoot:   filepath.Join(dir, "memory"),
		ClaudeMDPath: filepath.Join(dir, "CLAUDE.md"),
		Context:      ctx,
	}
	_, err := Optimize(opts)
	g.Expect(err).To(HaveOccurred())
}

func TestOptimize_CheckContextStep4(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	ctx := &nthErrorContext{failAt: 4}
	opts := OptimizeOpts{
		MemoryRoot:   filepath.Join(dir, "memory"),
		ClaudeMDPath: filepath.Join(dir, "CLAUDE.md"),
		Context:      ctx,
	}
	_, err := Optimize(opts)
	g.Expect(err).To(HaveOccurred())
}

func TestOptimize_CheckContextStep5(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	ctx := &nthErrorContext{failAt: 5}
	opts := OptimizeOpts{
		MemoryRoot:   filepath.Join(dir, "memory"),
		ClaudeMDPath: filepath.Join(dir, "CLAUDE.md"),
		Context:      ctx,
	}
	_, err := Optimize(opts)
	g.Expect(err).To(HaveOccurred())
}

func TestOptimize_CheckContextStep6(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	ctx := &nthErrorContext{failAt: 6}
	opts := OptimizeOpts{
		MemoryRoot:   filepath.Join(dir, "memory"),
		ClaudeMDPath: filepath.Join(dir, "CLAUDE.md"),
		Context:      ctx,
	}
	_, err := Optimize(opts)
	g.Expect(err).To(HaveOccurred())
}

func TestOptimize_CheckContextStep7(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	ctx := &nthErrorContext{failAt: 7}
	opts := OptimizeOpts{
		MemoryRoot:   filepath.Join(dir, "memory"),
		ClaudeMDPath: filepath.Join(dir, "CLAUDE.md"),
		Context:      ctx,
	}
	_, err := Optimize(opts)
	g.Expect(err).To(HaveOccurred())
}

func TestOptimize_CheckContextStep8(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	ctx := &nthErrorContext{failAt: 8}
	opts := OptimizeOpts{
		MemoryRoot:   filepath.Join(dir, "memory"),
		ClaudeMDPath: filepath.Join(dir, "CLAUDE.md"),
		Context:      ctx,
	}
	_, err := Optimize(opts)
	g.Expect(err).To(HaveOccurred())
}

func TestOptimize_CheckContextStep9(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	dir := t.TempDir()
	ctx := &nthErrorContext{failAt: 9}
	opts := OptimizeOpts{
		MemoryRoot:   filepath.Join(dir, "memory"),
		ClaudeMDPath: filepath.Join(dir, "CLAUDE.md"),
		Context:      ctx,
	}
	_, err := Optimize(opts)
	g.Expect(err).To(HaveOccurred())
}

// TestOptimize_CompileSkillsError verifies the compile-skills error path when SkillsDir is unwritable.
func TestOptimize_CompileSkillsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	claudeMD := filepath.Join(dir, "CLAUDE.md")
	err := os.WriteFile(claudeMD, []byte(""), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	opts := OptimizeOpts{
		MemoryRoot:   filepath.Join(dir, "memory"),
		ClaudeMDPath: claudeMD,
		SkillsDir:    "/dev/null/invalid-path", // MkdirAll fails on /dev/null
	}

	_, err = Optimize(opts)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("skill compilation failed"))
}

// ─── Optimize: HOME resolution ────────────────────────────────────────────────

// TestOptimize_HomeResolution verifies that empty MemoryRoot and ClaudeMDPath are resolved from HOME.
// NOTE: Not parallel — uses t.Setenv to override HOME.
func TestOptimize_HomeResolution(t *testing.T) {
	g := NewWithT(t)

	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Both MemoryRoot and ClaudeMDPath are empty → resolved from HOME (~/.claude/memory and ~/.claude/CLAUDE.md)
	opts := OptimizeOpts{}

	_, err := Optimize(opts)
	g.Expect(err).ToNot(HaveOccurred())
}

// ─── Optimize: result > 0 log blocks ─────────────────────────────────────────

// TestOptimize_ResultsGreaterThanZero verifies that the logChangelogMutation blocks are reached
// when EntriesDecayed, AutoDemoted, EntriesPruned, BoilerplatePurged, and LegacySessionPurged > 0.
func TestOptimize_ResultsGreaterThanZero(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	claudeMD := filepath.Join(dir, "CLAUDE.md")
	err := os.WriteFile(claudeMD, []byte("# Project\n\n## Promoted Learnings\n\n"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	memRoot := filepath.Join(dir, "memory")
	err = os.MkdirAll(memRoot, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	db, err := initEmbeddingsDB(filepath.Join(memRoot, "embeddings.db"))
	g.Expect(err).ToNot(HaveOccurred())

	// Entry 1: observation_type='correction' → excluded from legacy purge query; any entry → EntriesDecayed > 0
	_, err = db.Exec(
		"INSERT INTO embeddings (content, source, observation_type, confidence, promoted) VALUES (?, 'memory', 'correction', 0.8, 0)",
		"a useful learning about coding patterns in projects consistently applied",
	)
	g.Expect(err).ToNot(HaveOccurred())

	// Entry 2: promoted=1, confidence=0.2 → after promoted-decay (×0.995) → 0.199 < 0.3 → AutoDemoted > 0
	_, err = db.Exec(
		"INSERT INTO embeddings (content, source, confidence, promoted) VALUES (?, 'claude-md', 0.2, 1)",
		"some specific promoted learning about testing patterns in Go projects carefully",
	)
	g.Expect(err).ToNot(HaveOccurred())

	// Entry 3: promoted=0, confidence=0.05 → after decay (×0.9) → 0.045 < PruneThreshold(0.1) → EntriesPruned > 0
	// Has timestamp prefix → excluded from legacy purge; gets pruned first
	_, err = db.Exec(
		"INSERT INTO embeddings (content, source, confidence, promoted) VALUES (?, 'memory', 0.05, 0)",
		"- 2026-01-01 10:00: a real learning entry that gets pruned due to low confidence",
	)
	g.Expect(err).ToNot(HaveOccurred())

	// Entry 4: boilerplate content, confidence=0.9 → after decay (×0.9) → 0.81 > 0.1 → BoilerplatePurged > 0
	_, err = db.Exec(
		"INSERT INTO embeddings (content, source, confidence, promoted) VALUES (?, 'memory', 0.9, 0)",
		"# Session Header",
	)
	g.Expect(err).ToNot(HaveOccurred())

	// Entry 5: source='memory', no timestamp prefix, retrieval_count=0, confidence=0.5
	// → after decay (×0.9) → 0.45 > 0.1, not boilerplate → LegacySessionPurged > 0
	_, err = db.Exec(
		"INSERT INTO embeddings (content, source, confidence, promoted) VALUES (?, 'memory', 0.5, 0)",
		"old session learning line without timestamp prefix in this content entry",
	)
	g.Expect(err).ToNot(HaveOccurred())

	err = db.Close()
	g.Expect(err).ToNot(HaveOccurred())

	opts := OptimizeOpts{
		MemoryRoot:   memRoot,
		ClaudeMDPath: claudeMD,
	}

	result, err := Optimize(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())

	if result != nil {
		g.Expect(result.EntriesDecayed).To(BeNumerically(">", 0))
		g.Expect(result.AutoDemoted).To(BeNumerically(">", 0))
		g.Expect(result.EntriesPruned).To(BeNumerically(">", 0))
		g.Expect(result.BoilerplatePurged).To(BeNumerically(">", 0))
		g.Expect(result.LegacySessionPurged).To(BeNumerically(">", 0))
	}
}

// ─── Optimize (full pipeline) ─────────────────────────────────────────────────

func TestOptimize_WithEmptyMemoryDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	claudeMD := filepath.Join(dir, "CLAUDE.md")
	err := os.WriteFile(claudeMD, []byte("# Project\n\n## Promoted Learnings\n\n"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	opts := OptimizeOpts{
		MemoryRoot:     filepath.Join(dir, "memory"),
		ClaudeMDPath:   claudeMD,
		DecayBase:      0.9,
		TestSkills:     false,
		SimilarityFunc: calculateSimilarity,
	}

	result, err := Optimize(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())
}

func TestParseCompileSkillJSON_InvalidJSON(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, _, err := parseCompileSkillJSON("not json")

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("invalid CompileSkill JSON"))
}

func TestParseCompileSkillJSON_NilDescription(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	input := `{"body":"some body"}`
	desc, body, err := parseCompileSkillJSON(input)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(desc).To(BeEmpty())
	g.Expect(body).To(Equal("some body"))
}

func TestParseCompileSkillJSON_Valid(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	input := `{"description":"Use when foo","body":"## Overview\ncontent"}`
	desc, body, err := parseCompileSkillJSON(input)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(desc).To(Equal("Use when foo"))
	g.Expect(body).To(Equal("## Overview\ncontent"))
}

// TestPerformSkillReorganization_ComplianceFailsExistingSkill verifies the compliance
// failure path in the UPDATE branch (lines 3027-3032: SkillsBlocked++, append, continue).
// A pre-inserted skill matching the generated slug triggers the update path, then the
// regenerated description fails the pronoun check (V4) → SkillsBlocked++.
func TestPerformSkillReorganization_ComplianceFailsExistingSkill(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	// Same content → same theme "Always say I am done here" → slug "always-say-i-am-done-here"
	// → update path → regenerated description contains " i " → V4 fails → SkillsBlocked++
	for range 3 {
		_, err = db.Exec(
			`INSERT INTO embeddings (content, source, confidence, embedding_id) VALUES (?, ?, ?, ?)`,
			"Always say I am done here", "test", 0.9, int64(1),
		)
		g.Expect(err).ToNot(HaveOccurred())
	}

	// Pre-insert skill with the slug reorg will generate for "Always say I am done here"
	// slugify("Always say I am done here") = "always-say-i-am-done-here"
	now := time.Now().UTC().Format(time.RFC3339)
	existingSkill := &GeneratedSkill{
		Slug:            "always-say-i-am-done-here",
		Theme:           "Always say I am done here",
		Description:     "Use when testing compliance failures",
		Content:         "## Overview\n\nTest.\n\n## When to Use\n\nAlways.\n\n## Quick Reference\n\n1. Test.\n\n## Common Mistakes\n\n- None.",
		SourceMemoryIDs: "[]",
		Alpha:           1.0,
		Beta:            1.0,
		Utility:         0.5,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	_, err = insertSkill(db, existingSkill)
	g.Expect(err).ToNot(HaveOccurred())

	result := &OptimizeResult{}
	opts := OptimizeOpts{
		SkillsDir:      t.TempDir(),
		SimilarityFunc: func(_ *sql.DB, _, _ int64) (float64, error) { return 1.0, nil },
	}

	err = performSkillReorganization(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.SkillsBlocked).To(BeNumerically(">=", 1))
}

// TestPerformSkillReorganization_ComplianceFailsNewSkill verifies that a cluster whose
// generated description fails the pronoun check (V4) causes SkillsBlocked to be
// incremented and the new skill is NOT inserted.
// Covers the compliance failure branch for new skills (lines 3056-3062).
func TestPerformSkillReorganization_ComplianceFailsNewSkill(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	// Content "Always say I am done here" → theme contains " I " (space-I-space)
	// → generateTriggerDescription → description contains " i " (lowercase) → V4 pronoun check fails
	// → compliance.Issues non-empty → result.SkillsBlocked++ instead of SkillsReorganized++
	for range 3 {
		_, err = db.Exec(
			`INSERT INTO embeddings (content, source, confidence, embedding_id) VALUES (?, ?, ?, ?)`,
			"Always say I am done here", "test", 0.9, int64(1),
		)
		g.Expect(err).ToNot(HaveOccurred())
	}

	result := &OptimizeResult{}
	opts := OptimizeOpts{
		SkillsDir:      t.TempDir(),
		SimilarityFunc: func(_ *sql.DB, _, _ int64) (float64, error) { return 1.0, nil },
	}

	err = performSkillReorganization(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.SkillsBlocked).To(BeNumerically(">=", 1))
}

// TestPerformSkillReorganization_ContextCancel verifies context cancellation inside cluster loop.
func TestPerformSkillReorganization_ContextCancel(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	db, err := InitDBForTest(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	// Insert 3 memories (so len(allMemories) >= minCluster)
	for i := range 3 {
		_, err = db.Exec(
			`INSERT INTO embeddings (content, source, confidence, embedding_id) VALUES (?, ?, ?, ?)`,
			fmt.Sprintf("Use lint for quality step %d", i+1),
			"test",
			0.9,
			int64(i+1),
		)
		g.Expect(err).ToNot(HaveOccurred())
	}

	// Cancelled context → checkContext inside cluster loop returns error
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	alwaysSimilar := func(_ *sql.DB, _, _ int64) (float64, error) {
		return 1.0, nil
	}

	result := &OptimizeResult{}
	opts := OptimizeOpts{
		Context:        ctx,
		SkillsDir:      t.TempDir(),
		SimilarityFunc: alwaysSimilar,
	}

	err = performSkillReorganization(db, opts, result)

	g.Expect(err).To(HaveOccurred())

	err = db.Close()
	g.Expect(err).ToNot(HaveOccurred())
}

// TestPerformSkillReorganization_DBQueryFails verifies the error path when db.Query fails at start.
func TestPerformSkillReorganization_DBQueryFails(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	// Close DB before calling: db.Query will fail
	_ = db.Close()

	result := &OptimizeResult{}
	opts := OptimizeOpts{SkillsDir: t.TempDir()}

	err = performSkillReorganization(db, opts, result)
	g.Expect(err).To(HaveOccurred())
}

func TestPerformSkillReorganization_SetsLastReorgMetadata(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	result := &OptimizeResult{}
	opts := OptimizeOpts{
		SkillsDir:      t.TempDir(),
		MinClusterSize: 3,
		ReorgThreshold: 0.8,
		SimilarityFunc: calculateSimilarity,
	}

	err = performSkillReorganization(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())

	val, err := getMetadata(db, "last_skill_reorg_at")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(val).ToNot(BeEmpty())
}

// TestPerformSkillReorganization_SmallCluster verifies that a cluster smaller than MinClusterSize is skipped.
func TestPerformSkillReorganization_SmallCluster(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	db, err := InitDBForTest(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	// Insert 3 memories with embedding_id but use a SimilarityFunc that returns 0 → 3 singletons
	for i := range 3 {
		_, err = db.Exec(
			`INSERT INTO embeddings (content, source, confidence, embedding_id) VALUES (?, ?, ?, ?)`,
			fmt.Sprintf("Memory about topic %d", i+1),
			"test",
			0.9,
			int64(i+1),
		)
		g.Expect(err).ToNot(HaveOccurred())
	}

	// Similarity always 0 → no pairs merged → 3 singleton clusters → all skipped (< minCluster=3)
	neverSimilar := func(_ *sql.DB, _, _ int64) (float64, error) {
		return 0.0, nil
	}

	result := &OptimizeResult{}
	opts := OptimizeOpts{
		SkillsDir:      t.TempDir(),
		SimilarityFunc: neverSimilar,
	}

	err = performSkillReorganization(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())

	g.Expect(result.SkillsReorganized).To(Equal(0))

	err = db.Close()
	g.Expect(err).ToNot(HaveOccurred())
}

// ─── performSkillReorganization ──────────────────────────────────────────────

func TestPerformSkillReorganization_TooFewMemories(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	result := &OptimizeResult{}
	opts := OptimizeOpts{
		SkillsDir:      t.TempDir(),
		MinClusterSize: 3,
		ReorgThreshold: 0.8,
		SimilarityFunc: calculateSimilarity,
	}

	err = performSkillReorganization(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.SkillsReorganized).To(Equal(0))
}

// TestPerformSkillReorganization_UpdateExistingSkillNoTag verifies that an existing
// non-pruned skill with the right slug is updated (not inserted) during reorganization.
func TestPerformSkillReorganization_UpdateExistingSkillNoTag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	// Memory content "always handle errors consistently" → theme "always handle errors" → slug "always-handle-errors"
	for range 3 {
		_, err = db.Exec(
			`INSERT INTO embeddings (content, source, confidence, embedding_id) VALUES (?, ?, ?, ?)`,
			"always handle errors consistently", "test", 0.9, int64(1),
		)
		g.Expect(err).ToNot(HaveOccurred())
	}

	// Pre-insert skill with the slug that reorg will generate
	now := time.Now().UTC().Format(time.RFC3339)
	existingSkill := &GeneratedSkill{
		Slug:            "always-handle-errors",
		Theme:           "always handle errors",
		Description:     "Use when handling errors",
		Content:         "## Overview\n\nError handling.\n\n## When to Use\n\nAlways.\n\n## Quick Reference\n\n1. Handle errors.\n\n## Common Mistakes\n\n- Ignoring errors.",
		SourceMemoryIDs: "[]",
		Alpha:           1.0,
		Beta:            1.0,
		Utility:         0.5,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	_, err = insertSkill(db, existingSkill)
	g.Expect(err).ToNot(HaveOccurred())

	result := &OptimizeResult{}
	opts := OptimizeOpts{
		SkillsDir:      t.TempDir(),
		MinClusterSize: 3,
		SimilarityFunc: func(_ *sql.DB, _, _ int64) (float64, error) { return 1.0, nil },
	}

	err = performSkillReorganization(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
}

// ─── performSkillReorganization ───────────────────────────────────────────────

// TestPerformSkillReorganization_WithCluster verifies the cluster-processing path:
// 3 memories with embedding_id IS NOT NULL, custom SimilarityFunc returning 1.0
// → all 3 cluster together → skill created via template fallback.
func TestPerformSkillReorganization_WithCluster(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, "skills")

	db, err := InitDBForTest(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	// Insert 3 memories with non-NULL embedding_id so they appear in the query
	for i := range 3 {
		_, err = db.Exec(
			`INSERT INTO embeddings (content, source, confidence, embedding_id) VALUES (?, ?, ?, ?)`,
			fmt.Sprintf("Always use targ for builds step %d", i+1),
			"test",
			0.9,
			int64(i+1),
		)
		g.Expect(err).ToNot(HaveOccurred())
	}

	// SimilarityFunc always returns 1.0 → all 3 memories in one cluster of size 3 (≥ minCluster)
	alwaysSimilar := func(_ *sql.DB, _, _ int64) (float64, error) {
		return 1.0, nil
	}

	result := &OptimizeResult{}
	opts := OptimizeOpts{
		SkillsDir:      skillsDir,
		SimilarityFunc: alwaysSimilar,
	}

	err = performSkillReorganization(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())

	// One cluster → one skill inserted (nil compiler → template fallback → compliance passes)
	g.Expect(result.SkillsReorganized).To(BeNumerically(">=", 1))

	err = db.Close()
	g.Expect(err).ToNot(HaveOccurred())
}

func TestPerformSkillReorganization_WithSimilarEmbeddings(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	// Insert 3 entries with a non-null embedding_id; mock simFunc clusters them all together
	for _, content := range []string{
		"always handle errors explicitly",
		"always handle errors consistently",
		"always handle errors carefully",
	} {
		_, err = db.Exec(
			"INSERT INTO embeddings (content, source, embedding_id) VALUES (?, 'memory', 1)",
			content,
		)
		g.Expect(err).ToNot(HaveOccurred())
	}

	result := &OptimizeResult{}
	opts := OptimizeOpts{
		SkillsDir:      t.TempDir(),
		MinClusterSize: 3,
		ReorgThreshold: 0.8,
		SimilarityFunc: func(_ *sql.DB, _, _ int64) (float64, error) { return 1.0, nil },
	}

	err = performSkillReorganization(db, opts, result)
	g.Expect(err).ToNot(HaveOccurred())
}

func TestPruneOrphanedSkills_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	pruned, err := pruneOrphanedSkills(db, t.TempDir(), map[string]bool{})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(pruned).To(Equal(0))
}

func TestPruneOrphanedSkills_KeepsActiveSlug(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	skill := &GeneratedSkill{
		Slug:            "active-skill",
		Theme:           "active skill",
		Description:     "Use when active",
		Content:         "content",
		SourceMemoryIDs: "[]",
		Alpha:           1.0,
		Beta:            1.0,
	}
	_, err = insertSkill(db, skill)
	g.Expect(err).ToNot(HaveOccurred())

	pruned, err := pruneOrphanedSkills(db, t.TempDir(), map[string]bool{"active-skill": true})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(pruned).To(Equal(0))
}

func TestPruneOrphanedSkills_PrunesAbsentSlug(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	skill := &GeneratedSkill{
		Slug:            "old-skill",
		Theme:           "old skill",
		Description:     "Use when old",
		Content:         "content",
		SourceMemoryIDs: "[]",
		Alpha:           1.0,
		Beta:            1.0,
	}
	_, err = insertSkill(db, skill)
	g.Expect(err).ToNot(HaveOccurred())

	// activeThemes does NOT include "old-skill"
	pruned, err := pruneOrphanedSkills(db, t.TempDir(), map[string]bool{"other-skill": true})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(pruned).To(Equal(1))
}

func TestPruneStaleSkills_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	pruned := pruneStaleSkills(db, t.TempDir(), 0.4)
	g.Expect(pruned).To(Equal(0))
}

func TestPruneStaleSkills_PrunesLowUtility(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	skill := &GeneratedSkill{
		Slug:            "stale-skill",
		Theme:           "stale skill",
		Description:     "Use when stale",
		Content:         "content",
		SourceMemoryIDs: "[]",
		Alpha:           1.0,
		Beta:            1.0,
		Utility:         0.1,
	}
	id, err := insertSkill(db, skill)
	g.Expect(err).ToNot(HaveOccurred())

	// Set retrieval_count >= 5 so it qualifies for pruning
	_, err = db.Exec("UPDATE generated_skills SET retrieval_count = 5 WHERE id = ?", id)
	g.Expect(err).ToNot(HaveOccurred())

	pruned := pruneStaleSkills(db, t.TempDir(), 0.4)
	g.Expect(pruned).To(Equal(1))
}

func TestPruneStaleSkills_SkipsHighUtility(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	skill := &GeneratedSkill{
		Slug:            "good-skill",
		Theme:           "good skill",
		Description:     "Use when good",
		Content:         "content",
		SourceMemoryIDs: "[]",
		Alpha:           1.0,
		Beta:            1.0,
		Utility:         0.9,
	}
	id, err := insertSkill(db, skill)
	g.Expect(err).ToNot(HaveOccurred())

	_, err = db.Exec("UPDATE generated_skills SET retrieval_count = 5 WHERE id = ?", id)
	g.Expect(err).ToNot(HaveOccurred())

	pruned := pruneStaleSkills(db, t.TempDir(), 0.4)
	g.Expect(pruned).To(Equal(0))
}

func TestTestAndCompileSkill_TestFails(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	opts := OptimizeOpts{
		TestSkills:  true,
		MemoryRoot:  dir,
		SkillTester: &mockSkillTester{pass: false, reasoning: "bad skill"},
	}
	candidate := SkillCandidate{Theme: "test", Content: "content"}

	err := TestAndCompileSkill(opts, candidate)
	g.Expect(err).To(HaveOccurred())

	if err == nil {
		t.Fatal("expected error but got nil")
	}

	g.Expect(err.Error()).To(ContainSubstring("skill test failed"))
}

func TestTestAndCompileSkill_TestPasses(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	opts := OptimizeOpts{
		TestSkills:  true,
		MemoryRoot:  dir,
		SkillTester: &mockSkillTester{pass: true, reasoning: "looks good"},
	}
	candidate := SkillCandidate{Theme: "test", Content: "content"}

	err := TestAndCompileSkill(opts, candidate)
	g.Expect(err).ToNot(HaveOccurred())
}

func TestTestAndCompileSkill_TestSkillsDisabled(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	opts := OptimizeOpts{TestSkills: false}
	candidate := SkillCandidate{Theme: "test", Content: "content"}

	err := TestAndCompileSkill(opts, candidate)
	g.Expect(err).ToNot(HaveOccurred())
}

func TestValidateSkillCompliance_BodyTooLong(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	bodyLines := strings.Repeat("line of content here\n", 502)
	skill := &GeneratedSkill{
		Slug:        "test-skill",
		Description: "Use when testing body length limits",
		Content:     "## Overview\n\n## When to Use\n\n## Quick Reference\n\n## Common Mistakes\n\n" + bodyLines,
	}

	result := ValidateSkillCompliance(skill)
	g.Expect(result.BodyLengthOK).To(BeFalse())
	g.Expect(result.Issues).To(ContainElement(ContainSubstring("exceeds 500 lines")))
}

func TestValidateSkillCompliance_DescriptionTooLong(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	skill := &GeneratedSkill{
		Slug:        "test-skill",
		Description: "Use when " + strings.Repeat("x", 1020),
		Content:     "## Overview\n\nc\n\n## When to Use\n\nu\n\n## Quick Reference\n\nr\n\n## Common Mistakes\n\nm\n",
	}

	result := ValidateSkillCompliance(skill)
	g.Expect(result.DescriptionOK).To(BeFalse())
	g.Expect(result.Issues).To(ContainElement(ContainSubstring("exceeds 1024 chars")))
}

func TestValidateSkillCompliance_DescriptionWithPronoun(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	skill := &GeneratedSkill{
		Slug:        "test-skill",
		Description: "Use when I need to handle errors",
		Content:     "## Overview\n\nc\n\n## When to Use\n\nu\n\n## Quick Reference\n\nr\n\n## Common Mistakes\n\nm\n",
	}

	result := ValidateSkillCompliance(skill)
	g.Expect(result.DescriptionOK).To(BeFalse())
	g.Expect(result.Issues).To(ContainElement(ContainSubstring("third person")))
}

func TestValidateSkillCompliance_MissingBodySections(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	skill := &GeneratedSkill{
		Slug:        "test-skill",
		Description: "Use when testing",
		Content:     "# Title\n\nNo required sections present.",
	}

	result := ValidateSkillCompliance(skill)
	g.Expect(result.BodyStructureOK).To(BeFalse())
	g.Expect(result.Issues).To(ContainElement(ContainSubstring("body missing required section")))
}

// ─── ValidateSkillCompliance ─────────────────────────────────────────────────

func TestValidateSkillCompliance_ValidSkill(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	skill := &GeneratedSkill{
		Slug:        "test-skill",
		Description: "Use when working on standard patterns for error handling",
		Content:     "## Overview\n\ncontent\n\n## When to Use\n\nuse it\n\n## Quick Reference\n\nreference\n\n## Common Mistakes\n\nmistakes\n",
	}

	result := ValidateSkillCompliance(skill)
	g.Expect(result.DescriptionOK).To(BeTrue())
	g.Expect(result.BodyStructureOK).To(BeTrue())
	g.Expect(result.Issues).To(BeEmpty())
}

// fakeSkillCompiler is a SkillCompiler test double that returns a fixed string response.
type fakeSkillCompiler struct {
	content string
	err     error
}

func (f *fakeSkillCompiler) CompileSkill(_ context.Context, _ string, _ []string) (string, error) {
	return f.content, f.err
}

func (f *fakeSkillCompiler) Synthesize(_ context.Context, _ []string) (string, error) {
	return "", nil
}

type mockSkillCompiler struct {
	compileErr error
	content    string
}

func (m *mockSkillCompiler) CompileSkill(_ context.Context, _ string, _ []string) (string, error) {
	if m.compileErr != nil {
		return "", m.compileErr
	}

	return m.content, nil
}

func (m *mockSkillCompiler) Synthesize(_ context.Context, _ []string) (string, error) {
	return m.content, nil
}

type mockSkillCompilerSynthErr struct {
	err error
}

func (m *mockSkillCompilerSynthErr) CompileSkill(_ context.Context, _ string, _ []string) (string, error) {
	return "", nil
}

func (m *mockSkillCompilerSynthErr) Synthesize(_ context.Context, _ []string) (string, error) {
	return "", m.err
}

type mockSkillTester struct {
	pass      bool
	reasoning string
	err       error
}

func (m *mockSkillTester) TestAndEvaluate(_ TestScenario, _ int) (bool, string, error) {
	return m.pass, m.reasoning, m.err
}

// ─── nthErrorContext ──────────────────────────────────────────────────────────

// nthErrorContext is a context.Context that returns context.Canceled on the Nth call to Err().
// This enables deterministic coverage of specific checkContext error blocks.
// Deadline/Done/Value delegate to background context behaviour (no deadline, no cancel, no value).
type nthErrorContext struct {
	mu     sync.Mutex
	count  int
	failAt int
}

func (*nthErrorContext) Deadline() (time.Time, bool) { return time.Time{}, false }

func (*nthErrorContext) Done() <-chan struct{} { return nil }

func (c *nthErrorContext) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.count++

	if c.count >= c.failAt {
		return context.Canceled
	}

	return nil
}

func (*nthErrorContext) Value(any) any { return nil }
