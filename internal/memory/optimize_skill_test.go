//go:build sqlite_fts5

package memory_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/memory"
)

// ============================================================================
// Unit tests for Compile Pipeline Integration (TASK-3)
// ============================================================================

// TestOptimizeOptsHasSkillFields verifies that OptimizeOpts has SkillsDir
// and SkillCompiler fields.
func TestOptimizeOptsHasSkillFields(t *testing.T) {
	g := NewWithT(t)

	opts := memory.OptimizeOpts{
		SkillsDir:     "/tmp/skills",
		SkillCompiler: &mockSkillCompiler{},
	}

	g.Expect(opts.SkillsDir).To(Equal("/tmp/skills"))
	g.Expect(opts.SkillCompiler).ToNot(BeNil())
}

// TestOptimizeResultHasSkillFields verifies that OptimizeResult has
// SkillsCompiled, SkillsMerged, and SkillsPruned fields.
func TestOptimizeResultHasSkillFields(t *testing.T) {
	g := NewWithT(t)

	result := memory.OptimizeResult{
		SkillsCompiled: 3,
		SkillsMerged:   1,
		SkillsPruned:   2,
	}

	g.Expect(result.SkillsCompiled).To(Equal(3))
	g.Expect(result.SkillsMerged).To(Equal(1))
	g.Expect(result.SkillsPruned).To(Equal(2))
}

// TestOptimizeCompileSkillsCreatesSkill verifies that the compile step
// creates a skill from a qualifying cluster.
func TestOptimizeCompileSkillsCreatesSkill(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	skillsDir := filepath.Join(tempDir, ".claude", "skills", "memory-gen")
	claudeMDPath := filepath.Join(tempDir, ".claude", "CLAUDE.md")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())
	g.Expect(os.MkdirAll(filepath.Dir(claudeMDPath), 0755)).To(Succeed())
	g.Expect(os.WriteFile(claudeMDPath, []byte("# CLAUDE.md\n\n## Promoted Learnings\n\n"), 0644)).To(Succeed())

	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	// Seed metadata so Optimize takes the normal compile path (not the reorg path)
	now := time.Now().UTC().Format(time.RFC3339)
	g.Expect(memory.SetMetadataForTest(memoryRoot, "last_skill_reorg_at", now)).To(Succeed())
	g.Expect(memory.SetMetadataForTest(memoryRoot, "last_optimized_at", now)).To(Succeed())

	// Insert 3+ similar memories with embeddings (to form a cluster)
	baseEmb := make([]float32, 384)
	for i := range baseEmb {
		baseEmb[i] = 0.5
	}

	for i := 0; i < 4; i++ {
		emb := make([]float32, 384)
		copy(emb, baseEmb)
		// Tiny variation to keep similarity high
		emb[0] = 0.5 + float32(i)*0.001

		blob, err := sqlite_vec.SerializeFloat32(emb)
		g.Expect(err).ToNot(HaveOccurred())

		result, err := db.Exec("INSERT INTO vec_embeddings(embedding) VALUES (?)", blob)
		g.Expect(err).ToNot(HaveOccurred())
		embID, err := result.LastInsertId()
		g.Expect(err).ToNot(HaveOccurred())

		_, err = db.Exec(`INSERT INTO embeddings (content, source, source_type, confidence, retrieval_count, last_retrieved, embedding_id, memory_type)
			VALUES (?, 'test', 'internal', 0.9, 10, ?, ?, 'pattern')`,
			"TDD pattern: always write tests first "+string(rune('A'+i)), now, embID)
		g.Expect(err).ToNot(HaveOccurred())
	}

	compiler := &mockSkillCompiler{
		compileFunc: func(theme string, memories []string) (string, error) {
			return "# " + theme + "\n\nGenerated skill content.", nil
		},
	}

	// Reject synthesis/promotion so memories survive to compile step.
	// Set DupThreshold high to prevent dedup from consuming similar entries.
	result, err := memory.Optimize(memory.OptimizeOpts{
		MemoryRoot:     memoryRoot,
		ClaudeMDPath:   claudeMDPath,
		SkillsDir:      skillsDir,
		SkillCompiler:  compiler,
		SynthThreshold: 0.8,
		DupThreshold:   1.01, // Effectively disable dedup
		MinClusterSize: 3,
		AutoApprove:    false,
		ReviewFunc: func(action, description string) (bool, error) {
			return false, nil // Reject synthesis/promotion so memories stay for compile
		},
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.SkillsCompiled).To(BeNumerically(">=", 1))

	// Verify skill file was created
	dirEntries, err := os.ReadDir(skillsDir)
	if err == nil && len(dirEntries) > 0 {
		skillFile := filepath.Join(skillsDir, dirEntries[0].Name(), "SKILL.md")
		_, err := os.Stat(skillFile)
		g.Expect(err).ToNot(HaveOccurred())
	}
}

// TestOptimizeCompileSkillsSkipsExistingSkillMembers verifies that clusters
// whose members already belong to an existing non-pruned skill are skipped.
func TestOptimizeCompileSkillsSkipsExistingSkillMembers(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	skillsDir := filepath.Join(tempDir, ".claude", "skills", "memory-gen")
	claudeMDPath := filepath.Join(tempDir, ".claude", "CLAUDE.md")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())
	g.Expect(os.MkdirAll(filepath.Dir(claudeMDPath), 0755)).To(Succeed())
	g.Expect(os.WriteFile(claudeMDPath, []byte("# CLAUDE.md\n\n## Promoted Learnings\n\n"), 0644)).To(Succeed())

	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	now := time.Now().UTC().Format(time.RFC3339)

	// Insert 4 similar memories
	var memIDs []int64
	baseEmb := make([]float32, 384)
	for i := range baseEmb {
		baseEmb[i] = 0.5
	}

	for i := 0; i < 4; i++ {
		emb := make([]float32, 384)
		copy(emb, baseEmb)
		emb[0] = 0.5 + float32(i)*0.001

		blob, err := sqlite_vec.SerializeFloat32(emb)
		g.Expect(err).ToNot(HaveOccurred())

		result, err := db.Exec("INSERT INTO vec_embeddings(embedding) VALUES (?)", blob)
		g.Expect(err).ToNot(HaveOccurred())
		embID, err := result.LastInsertId()
		g.Expect(err).ToNot(HaveOccurred())

		res, err := db.Exec(`INSERT INTO embeddings (content, source, source_type, confidence, retrieval_count, last_retrieved, embedding_id, memory_type)
			VALUES (?, 'test', 'internal', 0.9, 10, ?, ?, 'pattern')`,
			"TDD pattern: always write tests first "+string(rune('A'+i)), now, embID)
		g.Expect(err).ToNot(HaveOccurred())
		memID, _ := res.LastInsertId()
		memIDs = append(memIDs, memID)
	}

	// Create an existing skill that references these memory IDs
	sourceIDs, _ := json.Marshal(memIDs)
	skill := &memory.GeneratedSkill{
		Slug:            "existing-skill",
		Theme:           "TDD Patterns",
		Description:     "Existing skill for TDD",
		Content:         "# TDD\n\nContent",
		SourceMemoryIDs: string(sourceIDs),
		Alpha:           5.0,
		Beta:            1.0,
		Utility:         0.8,
		CreatedAt:       now,
		UpdatedAt:       now,
		Pruned:          false,
	}
	_, err = memory.InsertSkillForTest(db, skill)
	g.Expect(err).ToNot(HaveOccurred())

	compiler := &mockSkillCompiler{
		compileFunc: func(theme string, memories []string) (string, error) {
			return "# " + theme + "\n\nGenerated skill content.", nil
		},
	}

	result, err := memory.Optimize(memory.OptimizeOpts{
		MemoryRoot:     memoryRoot,
		ClaudeMDPath:   claudeMDPath,
		SkillsDir:      skillsDir,
		SkillCompiler:  compiler,
		SynthThreshold: 0.8,
		DupThreshold:   1.01,
		MinClusterSize: 3,
		AutoApprove:    false,
		ReviewFunc: func(action, description string) (bool, error) {
			return false, nil
		},
	})
	g.Expect(err).ToNot(HaveOccurred())
	// Should not create new skills since members belong to existing skill
	g.Expect(result.SkillsCompiled).To(Equal(0))
}

// TestPruneStaleSkillsSoftDeletes verifies that pruneStaleSkills soft-deletes
// skills with utility < 0.4 and retrieval_count >= 5.
func TestPruneStaleSkillsSoftDeletes(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	skillsDir := filepath.Join(tempDir, ".claude", "skills", "memory-gen")
	claudeMDPath := filepath.Join(tempDir, ".claude", "CLAUDE.md")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())
	g.Expect(os.MkdirAll(filepath.Dir(claudeMDPath), 0755)).To(Succeed())
	g.Expect(os.WriteFile(claudeMDPath, []byte("# CLAUDE.md\n\n## Promoted Learnings\n\n"), 0644)).To(Succeed())

	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	// Seed metadata so Optimize takes the normal compile path (not the reorg path)
	now := time.Now().UTC().Format(time.RFC3339)
	g.Expect(memory.SetMetadataForTest(memoryRoot, "last_skill_reorg_at", now)).To(Succeed())
	g.Expect(memory.SetMetadataForTest(memoryRoot, "last_optimized_at", now)).To(Succeed())

	// Create a stale skill: low utility, enough retrievals
	staleSkill := &memory.GeneratedSkill{
		Slug:            "stale-skill",
		Theme:           "Stale Topic",
		Description:     "This skill is stale",
		Content:         "# Stale\n\nContent",
		SourceMemoryIDs: "[1,2,3]",
		Alpha:           1.0,
		Beta:            4.0, // confidence = 0.2
		Utility:         0.3, // < 0.4
		RetrievalCount:  10,  // >= 5
		CreatedAt:       now,
		UpdatedAt:       now,
		Pruned:          false,
	}

	_, err = memory.InsertSkillForTest(db, staleSkill)
	g.Expect(err).ToNot(HaveOccurred())

	// Create skill file on disk
	g.Expect(os.MkdirAll(filepath.Join(skillsDir, "stale-skill"), 0755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(skillsDir, "stale-skill", "SKILL.md"), []byte("stale"), 0644)).To(Succeed())

	// Create a healthy skill for contrast
	healthySkill := &memory.GeneratedSkill{
		Slug:            "healthy-skill",
		Theme:           "Healthy Topic",
		Description:     "This skill is healthy",
		Content:         "# Healthy\n\nContent",
		SourceMemoryIDs: "[4,5,6]",
		Alpha:           8.0,
		Beta:            2.0,
		Utility:         0.85,
		RetrievalCount:  20,
		CreatedAt:       now,
		UpdatedAt:       now,
		Pruned:          false,
	}

	_, err = memory.InsertSkillForTest(db, healthySkill)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := memory.Optimize(memory.OptimizeOpts{
		MemoryRoot:   memoryRoot,
		ClaudeMDPath: claudeMDPath,
		SkillsDir:    skillsDir,
		AutoApprove:  true,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.SkillsPruned).To(Equal(1))

	// Verify stale skill is pruned in DB
	stale, err := memory.GetSkillBySlugForTest(db, "stale-skill")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(stale.Pruned).To(BeTrue())

	// Verify stale skill file was removed
	_, err = os.Stat(filepath.Join(skillsDir, "stale-skill", "SKILL.md"))
	g.Expect(os.IsNotExist(err)).To(BeTrue())

	// Verify healthy skill is untouched
	healthy, err := memory.GetSkillBySlugForTest(db, "healthy-skill")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(healthy.Pruned).To(BeFalse())
}

// TestOptimizeCompileSkillsNoCompiler verifies that skill compilation
// is skipped when no SkillsDir is provided.
func TestOptimizeCompileSkillsNoCompiler(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	claudeMDPath := filepath.Join(tempDir, ".claude", "CLAUDE.md")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())
	g.Expect(os.MkdirAll(filepath.Dir(claudeMDPath), 0755)).To(Succeed())
	g.Expect(os.WriteFile(claudeMDPath, []byte("# CLAUDE.md\n\n## Promoted Learnings\n\n"), 0644)).To(Succeed())

	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	// No SkillsDir → compile step should be a no-op
	result, err := memory.Optimize(memory.OptimizeOpts{
		MemoryRoot:   memoryRoot,
		ClaudeMDPath: claudeMDPath,
		AutoApprove:  true,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.SkillsCompiled).To(Equal(0))
	g.Expect(result.SkillsMerged).To(Equal(0))
	g.Expect(result.SkillsPruned).To(Equal(0))
}
