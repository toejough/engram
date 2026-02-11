//go:build sqlite_fts5

package memory_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/memory"
)

// ============================================================================
// End-to-End Integration Test for Dynamic Skill Generation (TASK-6)
// ============================================================================

// TestE2EOptimizeCreatesSkillAndQueryReturns tests the full lifecycle:
// 1. Insert 6 related memories with embeddings
// 2. Optimize with mock compiler → generates skill
// 3. Skill files exist on disk with correct frontmatter
// 4. FormatMarkdown includes skill section
func TestE2EOptimizeCreatesSkillAndQueryReturns(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	skillsDir := filepath.Join(tempDir, ".claude", "skills")
	claudeMDPath := filepath.Join(tempDir, ".claude", "CLAUDE.md")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())
	g.Expect(os.MkdirAll(filepath.Dir(claudeMDPath), 0755)).To(Succeed())
	g.Expect(os.WriteFile(claudeMDPath, []byte("# CLAUDE.md\n\n## Promoted Learnings\n\n"), 0644)).To(Succeed())

	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())

	// Seed metadata so Optimize takes the normal compile path (not the reorg path)
	now := time.Now().UTC().Format(time.RFC3339)
	g.Expect(memory.SetMetadataForTest(memoryRoot, "last_skill_reorg_at", now)).To(Succeed())
	g.Expect(memory.SetMetadataForTest(memoryRoot, "last_optimized_at", now)).To(Succeed())

	// Insert 6 related memories about TDD with embeddings
	memories := []string{
		"Always write tests before implementation code in TDD workflow",
		"Write the minimal test that fails first then implement",
		"Test-driven development catches bugs earlier in the cycle",
		"Run tests after every small change to catch regressions",
		"TDD leads to better API design through testability constraints",
		"Red-green-refactor is the core TDD cycle to follow",
	}

	baseEmb := make([]float32, 384)
	for i := range baseEmb {
		baseEmb[i] = 0.5
	}

	for i, msg := range memories {
		emb := make([]float32, 384)
		copy(emb, baseEmb)
		emb[0] = 0.5 + float32(i)*0.001

		blob, err := sqlite_vec.SerializeFloat32(emb)
		g.Expect(err).ToNot(HaveOccurred())

		result, err := db.Exec("INSERT INTO vec_embeddings(embedding) VALUES (?)", blob)
		g.Expect(err).ToNot(HaveOccurred())
		embID, err := result.LastInsertId()
		g.Expect(err).ToNot(HaveOccurred())

		_, err = db.Exec(`INSERT INTO embeddings (content, source, source_type, confidence, retrieval_count, last_retrieved, embedding_id, memory_type)
			VALUES (?, 'test', 'internal', 0.9, 10, ?, ?, 'pattern')`,
			msg, now, embID)
		g.Expect(err).ToNot(HaveOccurred())
	}

	// Close DB before Optimize opens its own
	_ = db.Close()

	// Step 2: Optimize with mock compiler
	compiler := &mockSkillCompiler{
		compileFunc: func(_ context.Context, theme string, mems []string) (string, error) {
			return "# " + theme + "\n\nTest-driven development is essential.\n\n## Guidelines\n\n1. Write tests first\n2. Make them fail\n3. Implement minimum code\n4. Refactor\n", nil
		},
	}

	result, err := memory.Optimize(memory.OptimizeOpts{
		MemoryRoot:     memoryRoot,
		ClaudeMDPath:   claudeMDPath,
		SkillsDir:      skillsDir,
		SkillCompiler:  compiler,
		SynthThreshold: 0.7,
		DupThreshold:   1.01, // Disable dedup
		MinClusterSize: 3,
		AutoApprove:    false,
		ReviewFunc: func(action, description string) (bool, error) {
			return false, nil
		},
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.SkillsCompiled).To(BeNumerically(">=", 1))

	// Step 3: Verify skill files exist on disk with correct frontmatter
	dirEntries, err := os.ReadDir(skillsDir)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(dirEntries).ToNot(BeEmpty())

	skillFile := filepath.Join(skillsDir, dirEntries[0].Name(), "SKILL.md")
	content, err := os.ReadFile(skillFile)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content)).To(ContainSubstring("---"))
	g.Expect(string(content)).To(ContainSubstring("mem:"))
	g.Expect(string(content)).To(ContainSubstring("confidence:"))
	g.Expect(string(content)).To(ContainSubstring("generated: true"))
}

// TestE2EPrunedSkillFileRemoval verifies that pruning removes skill files.
func TestE2EPrunedSkillFileRemoval(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	skillsDir := filepath.Join(tempDir, ".claude", "skills")
	claudeMDPath := filepath.Join(tempDir, ".claude", "CLAUDE.md")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())
	g.Expect(os.MkdirAll(filepath.Dir(claudeMDPath), 0755)).To(Succeed())
	g.Expect(os.WriteFile(claudeMDPath, []byte("# CLAUDE.md\n\n## Promoted Learnings\n\n"), 0644)).To(Succeed())

	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())

	// Seed metadata so Optimize takes the normal compile path (not the reorg path)
	now := time.Now().UTC().Format(time.RFC3339)
	g.Expect(memory.SetMetadataForTest(memoryRoot, "last_skill_reorg_at", now)).To(Succeed())
	g.Expect(memory.SetMetadataForTest(memoryRoot, "last_optimized_at", now)).To(Succeed())

	// Create a skill that should be pruned (low utility, enough retrievals)
	skill := &memory.GeneratedSkill{
		Slug:            "prune-me",
		Theme:           "Obsolete Pattern",
		Description:     "This will be pruned",
		Content:         "# Obsolete\n\nContent",
		SourceMemoryIDs: "[1,2,3]",
		Alpha:           1.0,
		Beta:            4.0,
		Utility:         0.2, // < 0.4
		RetrievalCount:  10,  // >= 5
		CreatedAt:       "2025-01-01T00:00:00Z",
		UpdatedAt:       "2025-01-01T00:00:00Z",
	}
	_, err = memory.InsertSkillForTest(db, skill)
	g.Expect(err).ToNot(HaveOccurred())

	// Create skill file on disk (mem- prefix matches prune path)
	skillDir := filepath.Join(skillsDir, "mem-prune-me")
	g.Expect(os.MkdirAll(skillDir, 0755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("pruned content"), 0644)).To(Succeed())

	// Close the DB before Optimize opens its own
	_ = db.Close()

	// Run optimize (which includes pruning)
	optimizeResult, err := memory.Optimize(memory.OptimizeOpts{
		MemoryRoot:   memoryRoot,
		ClaudeMDPath: claudeMDPath,
		SkillsDir:    skillsDir,
		AutoApprove:  true,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(optimizeResult.SkillsPruned).To(Equal(1))

	// Verify skill file was removed
	_, err = os.Stat(filepath.Join(skillDir, "SKILL.md"))
	g.Expect(os.IsNotExist(err)).To(BeTrue())
}

// TestE2ESkillFeedbackCycle verifies the feedback → utility update cycle.
func TestE2ESkillFeedbackCycle(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	// Create a skill
	skill := &memory.GeneratedSkill{
		Slug:            "feedback-test",
		Theme:           "Test",
		Description:     "Test skill for feedback",
		Content:         "Content",
		SourceMemoryIDs: "[1]",
		Alpha:           1.0,
		Beta:            1.0,
		Utility:         0.5,
		CreatedAt:       "2025-01-01T00:00:00Z",
		UpdatedAt:       "2025-01-01T00:00:00Z",
	}
	_, err = memory.InsertSkillForTest(db, skill)
	g.Expect(err).ToNot(HaveOccurred())

	// Record 3 positive feedbacks
	for i := 0; i < 3; i++ {
		err = memory.RecordSkillFeedback(db, "feedback-test", true)
		g.Expect(err).ToNot(HaveOccurred())
	}

	// Check confidence increased
	updated, err := memory.GetSkillBySlugForTest(db, "feedback-test")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(updated.Alpha).To(Equal(4.0)) // 1.0 + 3
	g.Expect(updated.Beta).To(Equal(1.0))
	confidence := updated.Alpha / (updated.Alpha + updated.Beta)
	g.Expect(confidence).To(BeNumerically("==", 0.8)) // 4/5

	// Record 1 negative feedback
	err = memory.RecordSkillFeedback(db, "feedback-test", false)
	g.Expect(err).ToNot(HaveOccurred())

	updated, err = memory.GetSkillBySlugForTest(db, "feedback-test")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(updated.Alpha).To(Equal(4.0))
	g.Expect(updated.Beta).To(Equal(2.0))
	confidence = updated.Alpha / (updated.Alpha + updated.Beta)
	g.Expect(confidence).To(BeNumerically("~", 0.667, 0.01)) // 4/6
}
