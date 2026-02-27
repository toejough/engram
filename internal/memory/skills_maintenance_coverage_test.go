//go:build sqlite_fts5

package memory_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/memory"
)

// TestApplyConsolidate_DBClosed verifies consolidate returns error when DB is closed.
func TestApplyConsolidate_DBClosed(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	db, err := memory.OpenSkillDB(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	g.Expect(db.Close()).To(Succeed())

	applier := memory.NewSkillsApplier(db, memory.SkillsApplierOpts{})
	err = applier.Apply(memory.MaintenanceProposal{
		Tier:   "skills",
		Action: "consolidate",
		Target: "any-skill",
	})

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("failed to get target skill"))
}

// TestApplyConsolidate_ExistingMergeSourceIDs verifies consolidate merges when keepSkill has prior merge_source_ids.
func TestApplyConsolidate_ExistingMergeSourceIDs(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db := openTestSkillDB(t)

	now := time.Now().UTC().Format(time.RFC3339)
	_, err := memory.InsertSkillForTest(db, &memory.GeneratedSkill{
		Slug:            "cons-target-merge",
		Theme:           "Target to Merge",
		SourceMemoryIDs: "[10]",
		Alpha:           1.0,
		Beta:            1.0,
		Utility:         0.5,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	g.Expect(err).ToNot(HaveOccurred())

	_, err = memory.InsertSkillForTest(db, &memory.GeneratedSkill{
		Slug:            "cons-keep-merge",
		Theme:           "Keep Skill With History",
		SourceMemoryIDs: "[20]",
		Alpha:           5.0,
		Beta:            1.0,
		Utility:         0.8,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Set pre-existing merge_source_ids on keep skill (triggers json.Unmarshal path)
	_, err = db.Exec("UPDATE generated_skills SET merge_source_ids = ? WHERE slug = ?", "[5]", "cons-keep-merge")
	g.Expect(err).ToNot(HaveOccurred())

	applier := memory.NewSkillsApplier(db, memory.SkillsApplierOpts{})
	err = applier.Apply(memory.MaintenanceProposal{
		Tier:   "skills",
		Action: "consolidate",
		Target: "cons-target-merge",
		Reason: `Redundant with "cons-keep-merge" (similarity 0.92)`,
	})

	g.Expect(err).ToNot(HaveOccurred())

	target, err := memory.GetSkillBySlugForTest(db, "cons-target-merge")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(target.Pruned).To(BeTrue())
}

// TestApplyConsolidate_TargetNotFound verifies consolidate returns error for missing target.
func TestApplyConsolidate_TargetNotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db := openTestSkillDB(t)

	applier := memory.NewSkillsApplier(db, memory.SkillsApplierOpts{})
	err := applier.Apply(memory.MaintenanceProposal{
		Tier:   "skills",
		Action: "consolidate",
		Target: "nonexistent-slug",
	})

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("not found"))
}

// TestApplyConsolidate_WithSkillsDir verifies consolidate removes skill directory when SkillsDir set.
func TestApplyConsolidate_WithSkillsDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, "skills")
	db, err := memory.OpenSkillDB(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = memory.InsertSkillForTest(db, &memory.GeneratedSkill{
		Slug:            "cons-with-dir",
		Theme:           "Consolidate Dir Test",
		SourceMemoryIDs: "[1]",
		Alpha:           1.0,
		Beta:            1.0,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Create skill directory
	skillDir := filepath.Join(skillsDir, "memory-cons-with-dir")
	g.Expect(os.MkdirAll(skillDir, 0755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# Skill"), 0644)).To(Succeed())

	applier := memory.NewSkillsApplier(db, memory.SkillsApplierOpts{SkillsDir: skillsDir})
	err = applier.Apply(memory.MaintenanceProposal{
		Tier:   "skills",
		Action: "consolidate",
		Target: "cons-with-dir",
		Reason: "no quotes here so keepSlug is empty",
	})

	g.Expect(err).ToNot(HaveOccurred())

	_, err = os.Stat(skillDir)
	g.Expect(os.IsNotExist(err)).To(BeTrue())
}

// TestApplyDecay_SkillNotFound verifies decay returns error for missing skill.
func TestApplyDecay_SkillNotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db := openTestSkillDB(t)

	applier := memory.NewSkillsApplier(db, memory.SkillsApplierOpts{})
	err := applier.Apply(memory.MaintenanceProposal{
		Tier:   "skills",
		Action: "decay",
		Target: "nonexistent-skill",
	})

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("not found"))
}

// TestApplyDemote_PartialSources verifies demote succeeds with warning when some source memories are missing.
func TestApplyDemote_PartialSources(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db := openTestSkillDB(t)

	// Insert a real embedding so one source ID exists (fresh DB → id=1)
	_, err := db.Exec(`INSERT INTO embeddings (content, source) VALUES (?, ?)`, "test memory", "test-src")
	g.Expect(err).ToNot(HaveOccurred())

	now := time.Now().UTC().Format(time.RFC3339)
	// Source IDs: [1, 99999] — id=1 exists, 99999 does not → partial missing → warning path
	_, err = memory.InsertSkillForTest(db, &memory.GeneratedSkill{
		Slug:            "partial-sources-skill",
		Theme:           "Partial Sources",
		SourceMemoryIDs: "[1,99999]",
		Alpha:           1.0,
		Beta:            1.0,
		Utility:         0.5,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	g.Expect(err).ToNot(HaveOccurred())

	applier := memory.NewSkillsApplier(db, memory.SkillsApplierOpts{})
	err = applier.Apply(memory.MaintenanceProposal{
		Tier:   "skills",
		Action: "demote",
		Target: "partial-sources-skill",
	})

	// Warning goes to stderr; apply should succeed and prune the skill
	g.Expect(err).ToNot(HaveOccurred())

	skill, err := memory.GetSkillBySlugForTest(db, "partial-sources-skill")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(skill.Pruned).To(BeTrue())
}

// TestApplyDemote_TargetNotFound verifies demote returns error for missing skill.
func TestApplyDemote_TargetNotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db := openTestSkillDB(t)

	applier := memory.NewSkillsApplier(db, memory.SkillsApplierOpts{})
	err := applier.Apply(memory.MaintenanceProposal{
		Tier:   "skills",
		Action: "demote",
		Target: "nonexistent-demote-skill",
	})

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("not found"))
}

// TestApplyPromote_NoClaudeMDPath verifies promote errors when ClaudeMDPath not configured.
func TestApplyPromote_NoClaudeMDPath(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db := openTestSkillDB(t)

	applier := memory.NewSkillsApplier(db, memory.SkillsApplierOpts{})
	err := applier.Apply(memory.MaintenanceProposal{
		Tier:   "skills",
		Action: "promote",
		Target: "some-skill",
	})

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("ClaudeMDPath not configured"))
}

// TestApplyPromote_SkillNotFound verifies promote errors when skill missing from DB.
func TestApplyPromote_SkillNotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	claudeMDPath := filepath.Join(tmpDir, "CLAUDE.md")
	db, err := memory.OpenSkillDB(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	g.Expect(os.WriteFile(claudeMDPath, []byte("# Test\n\n## Promoted Learnings\n\n"), 0644)).To(Succeed())

	applier := memory.NewSkillsApplier(db, memory.SkillsApplierOpts{ClaudeMDPath: claudeMDPath})
	err = applier.Apply(memory.MaintenanceProposal{
		Tier:   "skills",
		Action: "promote",
		Target: "nonexistent-skill",
	})

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("not found"))
}

// TestApplyPromote_Success verifies promote appends principle to CLAUDE.md and marks DB.
func TestApplyPromote_Success(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	claudeMDPath := filepath.Join(tmpDir, "CLAUDE.md")
	db, err := memory.OpenSkillDB(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	initialContent := "# Working With Joe\n\n## Promoted Learnings\n\n- Existing learning\n"
	g.Expect(os.WriteFile(claudeMDPath, []byte(initialContent), 0644)).To(Succeed())

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = memory.InsertSkillForTest(db, &memory.GeneratedSkill{
		Slug:            "promote-skill",
		Theme:           "Always test first",
		SourceMemoryIDs: "[]",
		Alpha:           9.0,
		Beta:            1.0,
		Utility:         0.88,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	g.Expect(err).ToNot(HaveOccurred())

	applier := memory.NewSkillsApplier(db, memory.SkillsApplierOpts{ClaudeMDPath: claudeMDPath})
	err = applier.Apply(memory.MaintenanceProposal{
		Tier:    "skills",
		Action:  "promote",
		Target:  "promote-skill",
		Preview: "Always write tests before implementation",
	})

	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(claudeMDPath)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content)).To(ContainSubstring("Always write tests before implementation"))
}

// TestApplyPromote_ThemeFallback verifies promote uses Theme when Preview is empty.
func TestApplyPromote_ThemeFallback(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	claudeMDPath := filepath.Join(tmpDir, "CLAUDE.md")
	db, err := memory.OpenSkillDB(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	initialContent := "# Working With Joe\n\n## Promoted Learnings\n\n"
	g.Expect(os.WriteFile(claudeMDPath, []byte(initialContent), 0644)).To(Succeed())

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = memory.InsertSkillForTest(db, &memory.GeneratedSkill{
		Slug:            "theme-fallback-skill",
		Theme:           "TDD Pattern",
		SourceMemoryIDs: "[]",
		Alpha:           9.0,
		Beta:            1.0,
		Utility:         0.88,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	g.Expect(err).ToNot(HaveOccurred())

	applier := memory.NewSkillsApplier(db, memory.SkillsApplierOpts{ClaudeMDPath: claudeMDPath})
	err = applier.Apply(memory.MaintenanceProposal{
		Tier:    "skills",
		Action:  "promote",
		Target:  "theme-fallback-skill",
		Preview: "", // empty → should use Theme
	})

	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(claudeMDPath)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content)).To(ContainSubstring("TDD Pattern"))
}

// TestApplyPrune_DBClosed verifies prune returns error when DB is closed.
func TestApplyPrune_DBClosed(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	db, err := memory.OpenSkillDB(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	g.Expect(db.Close()).To(Succeed())

	applier := memory.NewSkillsApplier(db, memory.SkillsApplierOpts{})
	err = applier.Apply(memory.MaintenanceProposal{
		Tier:   "skills",
		Action: "prune",
		Target: "any-skill",
	})

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("failed to soft-delete skill"))
}

// TestApplySplit_InsertAFails verifies split returns error when inserting skill A fails due to duplicate slug.
func TestApplySplit_InsertAFails(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db := openTestSkillDB(t)

	now := time.Now().UTC().Format(time.RFC3339)
	_, err := memory.InsertSkillForTest(db, &memory.GeneratedSkill{
		Slug:            "split-fail",
		Theme:           "Split Fail",
		SourceMemoryIDs: "[1,2,3,4]",
		Alpha:           5.0,
		Beta:            1.0,
		Utility:         0.7,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Pre-insert the "split-fail-a" slug to cause UNIQUE constraint failure
	_, err = memory.InsertSkillForTest(db, &memory.GeneratedSkill{
		Slug:            "split-fail-a",
		Theme:           "Blocker Skill A",
		SourceMemoryIDs: "[]",
		Alpha:           1.0,
		Beta:            1.0,
		Utility:         0.5,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	g.Expect(err).ToNot(HaveOccurred())

	applier := memory.NewSkillsApplier(db, memory.SkillsApplierOpts{})
	err = applier.Apply(memory.MaintenanceProposal{
		Tier:   "skills",
		Action: "split",
		Target: "split-fail",
	})

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("failed to insert skill A"))
}

// TestDetectDemotionCandidates_WithUsageData verifies demotion is proposed for single-project skills.
func TestDetectDemotionCandidates_WithUsageData(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	db, err := memory.OpenSkillDB(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	now := time.Now().UTC().Format(time.RFC3339)
	result, err := db.Exec(`
		INSERT INTO generated_skills (slug, theme, description, content, source_memory_ids,
			alpha, beta, utility, retrieval_count, created_at, updated_at, pruned)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "narrow-skill", "Narrow Skill", "Desc", "Content", "[]",
		5.0, 1.0, 0.75, 5, now, now, 0)
	g.Expect(err).ToNot(HaveOccurred())

	skillID, _ := result.LastInsertId()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS skill_usage (skill_id INTEGER, project TEXT, timestamp TEXT)`)
	g.Expect(err).ToNot(HaveOccurred())

	_, err = db.Exec("INSERT INTO skill_usage (skill_id, project, timestamp) VALUES (?, ?, ?)",
		skillID, "proj-a", now)
	g.Expect(err).ToNot(HaveOccurred())

	scanner := memory.NewSkillsScanner(db, memory.SkillsScannerOpts{DemoteMaxProjects: 1})
	proposals, err := scanner.Scan()

	g.Expect(err).ToNot(HaveOccurred())

	var found bool

	for _, p := range proposals {
		if p.Action == "demote" && p.Target == "narrow-skill" {
			found = true

			break
		}
	}

	g.Expect(found).To(BeTrue())
}

// TestDetectPromotionCandidates_NoSkillUsageTable verifies no promote proposals without skill_usage table.
func TestDetectPromotionCandidates_NoSkillUsageTable(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db := openTestSkillDB(t)

	now := time.Now().UTC().Format(time.RFC3339)
	_, err := memory.InsertSkillForTest(db, &memory.GeneratedSkill{
		Slug:            "high-util-skill",
		Theme:           "High Utility",
		SourceMemoryIDs: "[]",
		Alpha:           9.0,
		Beta:            1.0,
		Utility:         0.88,
		RetrievalCount:  15,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	g.Expect(err).ToNot(HaveOccurred())

	scanner := memory.NewSkillsScanner(db, memory.SkillsScannerOpts{
		PromoteUtilityThreshold: 0.7,
		PromoteMinProjects:      3,
	})
	proposals, err := scanner.Scan()

	g.Expect(err).ToNot(HaveOccurred())

	for _, p := range proposals {
		g.Expect(p.Action).ToNot(Equal("promote"))
	}
}

// TestDetectRedundantSkills_LessThanTwo verifies no consolidation proposed with fewer than 2 skills.
func TestDetectRedundantSkills_LessThanTwo(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db := openTestSkillDB(t)

	now := time.Now().UTC().Format(time.RFC3339)
	_, err := memory.InsertSkillForTest(db, &memory.GeneratedSkill{
		Slug:            "only-skill",
		Theme:           "Only Skill",
		SourceMemoryIDs: "[]",
		Alpha:           5.0,
		Beta:            1.0,
		Utility:         0.8,
		EmbeddingID:     1,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	g.Expect(err).ToNot(HaveOccurred())

	scanner := memory.NewSkillsScanner(db, memory.SkillsScannerOpts{
		SimilarityThreshold: 0.85,
		SimilarityFunc: func(_ *sql.DB, _, _ int64) (float64, error) {
			return 0.92, nil
		},
	})
	proposals, err := scanner.Scan()

	g.Expect(err).ToNot(HaveOccurred())

	for _, p := range proposals {
		g.Expect(p.Action).ToNot(Equal("consolidate"))
	}
}

// TestDetectRedundantSkills_ThreeSkills_InnerContinue verifies inner checked[] skip when a skill is already processed.
func TestDetectRedundantSkills_ThreeSkills_InnerContinue(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db := openTestSkillDB(t)

	now := time.Now().UTC().Format(time.RFC3339)
	// Insert 3 skills with EmbeddingID set so they appear in query (ORDER BY utility DESC)
	// SkillA utility=0.80 EmbeddingID=10, SkillB utility=0.75 EmbeddingID=20, SkillC utility=0.70 EmbeddingID=30
	_, err := memory.InsertSkillForTest(db, &memory.GeneratedSkill{
		Slug:            "skill-a-inner",
		Theme:           "Skill A",
		SourceMemoryIDs: "[]",
		Alpha:           5.0,
		Beta:            1.0,
		Utility:         0.80,
		EmbeddingID:     10,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	g.Expect(err).ToNot(HaveOccurred())

	_, err = memory.InsertSkillForTest(db, &memory.GeneratedSkill{
		Slug:            "skill-b-inner",
		Theme:           "Skill B",
		SourceMemoryIDs: "[]",
		Alpha:           4.0,
		Beta:            1.0,
		Utility:         0.75,
		EmbeddingID:     20,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	g.Expect(err).ToNot(HaveOccurred())

	_, err = memory.InsertSkillForTest(db, &memory.GeneratedSkill{
		Slug:            "skill-c-inner",
		Theme:           "Skill C",
		SourceMemoryIDs: "[]",
		Alpha:           3.0,
		Beta:            1.0,
		Utility:         0.70,
		EmbeddingID:     30,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Similarity: high for (A,C) pair to trigger consolidation, low for (A,B) and (B,C)
	// This causes C to be marked checked, so when i=1(B), j=2(C) hits inner continue
	scanner := memory.NewSkillsScanner(db, memory.SkillsScannerOpts{
		SimilarityThreshold: 0.85,
		SimilarityFunc: func(_ *sql.DB, emb1, emb2 int64) (float64, error) {
			if (emb1 == 10 && emb2 == 30) || (emb1 == 30 && emb2 == 10) {
				return 0.92, nil
			}

			return 0.50, nil
		},
	})
	proposals, err := scanner.Scan()

	g.Expect(err).ToNot(HaveOccurred())

	var found bool

	for _, p := range proposals {
		if p.Action == "consolidate" && p.Target == "skill-c-inner" {
			found = true

			break
		}
	}

	g.Expect(found).To(BeTrue())
}

// TestDetectRedundantSkills_TwoSkills_HighSimilarity verifies consolidation proposed for two similar skills.
func TestDetectRedundantSkills_TwoSkills_HighSimilarity(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db := openTestSkillDB(t)

	now := time.Now().UTC().Format(time.RFC3339)
	// SkillX utility=0.80 (keep), SkillY utility=0.60 (merge)
	_, err := memory.InsertSkillForTest(db, &memory.GeneratedSkill{
		Slug:            "skill-x-high",
		Theme:           "Skill X",
		SourceMemoryIDs: "[]",
		Alpha:           5.0,
		Beta:            1.0,
		Utility:         0.80,
		EmbeddingID:     100,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	g.Expect(err).ToNot(HaveOccurred())

	_, err = memory.InsertSkillForTest(db, &memory.GeneratedSkill{
		Slug:            "skill-y-high",
		Theme:           "Skill Y",
		SourceMemoryIDs: "[]",
		Alpha:           2.0,
		Beta:            1.0,
		Utility:         0.60,
		EmbeddingID:     200,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	g.Expect(err).ToNot(HaveOccurred())

	scanner := memory.NewSkillsScanner(db, memory.SkillsScannerOpts{
		SimilarityThreshold: 0.85,
		SimilarityFunc: func(_ *sql.DB, _, _ int64) (float64, error) {
			return 0.92, nil
		},
	})
	proposals, err := scanner.Scan()

	g.Expect(err).ToNot(HaveOccurred())

	var found bool

	for _, p := range proposals {
		if p.Action == "consolidate" && p.Target == "skill-y-high" {
			found = true

			break
		}
	}

	g.Expect(found).To(BeTrue())
}

// TestDetectUnusedSkills_TooNew verifies recently-created skill is not proposed for pruning.
func TestDetectUnusedSkills_TooNew(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db := openTestSkillDB(t)

	now := time.Now().UTC().Format(time.RFC3339)
	_, err := memory.InsertSkillForTest(db, &memory.GeneratedSkill{
		Slug:            "new-skill",
		Theme:           "New Skill",
		SourceMemoryIDs: "[]",
		Alpha:           1.0,
		Beta:            1.0,
		Utility:         0.25,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	g.Expect(err).ToNot(HaveOccurred())

	scanner := memory.NewSkillsScanner(db, memory.SkillsScannerOpts{
		UnusedDaysThreshold: 30,
		LowUtilityThreshold: 0.4,
	})
	proposals, err := scanner.Scan()

	g.Expect(err).ToNot(HaveOccurred())

	for _, p := range proposals {
		g.Expect(p.Target).ToNot(Equal("new-skill"))
	}
}

// TestScan_DBClosed verifies Scan returns error when DB is closed.
func TestScan_DBClosed(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	db, err := memory.OpenSkillDB(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	g.Expect(db.Close()).To(Succeed())

	scanner := memory.NewSkillsScanner(db, memory.SkillsScannerOpts{})
	_, err = scanner.Scan()

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("failed to detect unused skills"))
}

// TestSkillsApplierApply_Consolidate tests Apply dispatches to consolidate action.
func TestSkillsApplierApply_Consolidate(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db := openTestSkillDB(t)
	insertTestSkill(t, db, "cons-dispatch-skill")

	// Override SourceMemoryIDs for consolidate to work (json.Unmarshal needed)
	_, err := db.Exec("UPDATE generated_skills SET source_memory_ids = '[1]' WHERE slug = ?", "cons-dispatch-skill")
	g.Expect(err).ToNot(HaveOccurred())

	applier := memory.NewSkillsApplier(db, memory.SkillsApplierOpts{})
	err = applier.Apply(memory.MaintenanceProposal{
		Tier:   "skills",
		Action: "consolidate",
		Target: "cons-dispatch-skill",
		Reason: "no quotes - keepSlug empty",
	})

	g.Expect(err).ToNot(HaveOccurred())

	skill, err := memory.GetSkillBySlugForTest(db, "cons-dispatch-skill")

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(skill.Pruned).To(BeTrue())
}

// TestSkillsApplierApply_Decay tests Apply dispatches to decay action.
func TestSkillsApplierApply_Decay(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db := openTestSkillDB(t)
	insertTestSkill(t, db, "decay-dispatch-skill")

	applier := memory.NewSkillsApplier(db, memory.SkillsApplierOpts{})
	err := applier.Apply(memory.MaintenanceProposal{
		Tier:   "skills",
		Action: "decay",
		Target: "decay-dispatch-skill",
	})

	g.Expect(err).ToNot(HaveOccurred())

	skill, err := memory.GetSkillBySlugForTest(db, "decay-dispatch-skill")

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(skill.Beta).To(BeNumerically(">", 1.0))
}

// TestSkillsApplierApply_Demote tests Apply dispatches to demote action.
func TestSkillsApplierApply_Demote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db := openTestSkillDB(t)
	insertTestSkill(t, db, "demote-dispatch-skill")

	applier := memory.NewSkillsApplier(db, memory.SkillsApplierOpts{})
	err := applier.Apply(memory.MaintenanceProposal{
		Tier:   "skills",
		Action: "demote",
		Target: "demote-dispatch-skill",
	})

	g.Expect(err).ToNot(HaveOccurred())

	skill, err := memory.GetSkillBySlugForTest(db, "demote-dispatch-skill")

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(skill.Pruned).To(BeTrue())
}

// TestSkillsApplierApply_InvalidTier tests Apply rejects wrong tier.
func TestSkillsApplierApply_InvalidTier(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db := openTestSkillDB(t)

	applier := memory.NewSkillsApplier(db, memory.SkillsApplierOpts{})
	err := applier.Apply(memory.MaintenanceProposal{
		Tier:   "wrong-tier",
		Action: "prune",
		Target: "some-skill",
	})

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("invalid tier"))
}

// TestSkillsApplierApply_Prune tests Apply dispatches to prune action and removes skill directory.
func TestSkillsApplierApply_Prune(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	skillsDir := filepath.Join(tmpDir, "skills")
	db, err := memory.OpenSkillDB(tmpDir)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = memory.InsertSkillForTest(db, &memory.GeneratedSkill{
		Slug:            "prune-dispatch-skill",
		Theme:           "Prune Dispatch",
		SourceMemoryIDs: "[]",
		Alpha:           1.0,
		Beta:            1.0,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	g.Expect(err).ToNot(HaveOccurred())

	skillDir := filepath.Join(skillsDir, "memory-prune-dispatch-skill")
	g.Expect(os.MkdirAll(skillDir, 0755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# Skill"), 0644)).To(Succeed())

	applier := memory.NewSkillsApplier(db, memory.SkillsApplierOpts{SkillsDir: skillsDir})
	err = applier.Apply(memory.MaintenanceProposal{
		Tier:   "skills",
		Action: "prune",
		Target: "prune-dispatch-skill",
	})

	g.Expect(err).ToNot(HaveOccurred())

	skill, err := memory.GetSkillBySlugForTest(db, "prune-dispatch-skill")

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(skill.Pruned).To(BeTrue())

	_, err = os.Stat(skillDir)
	g.Expect(os.IsNotExist(err)).To(BeTrue())
}

// TestSkillsApplierApply_Split tests Apply dispatches to split action.
func TestSkillsApplierApply_Split(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db := openTestSkillDB(t)

	now := time.Now().UTC().Format(time.RFC3339)
	_, err := memory.InsertSkillForTest(db, &memory.GeneratedSkill{
		Slug:            "split-dispatch-skill",
		Theme:           "Split Dispatch",
		SourceMemoryIDs: "[1,2,3,4]",
		Alpha:           5.0,
		Beta:            1.0,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	g.Expect(err).ToNot(HaveOccurred())

	applier := memory.NewSkillsApplier(db, memory.SkillsApplierOpts{})
	err = applier.Apply(memory.MaintenanceProposal{
		Tier:   "skills",
		Action: "split",
		Target: "split-dispatch-skill",
	})

	g.Expect(err).ToNot(HaveOccurred())

	skill, err := memory.GetSkillBySlugForTest(db, "split-dispatch-skill")

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(skill.Pruned).To(BeTrue())
}

// TestSkillsApplierApply_UnknownAction tests Apply returns error for unknown action.
func TestSkillsApplierApply_UnknownAction(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db := openTestSkillDB(t)

	applier := memory.NewSkillsApplier(db, memory.SkillsApplierOpts{})
	err := applier.Apply(memory.MaintenanceProposal{
		Tier:   "skills",
		Action: "unknown-action",
		Target: "some-skill",
	})

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("unknown action"))
}

// insertTestSkill inserts a minimal skill for testing.
func insertTestSkill(t *testing.T, db *sql.DB, slug string) int64 {
	t.Helper()
	g := NewWithT(t)

	now := time.Now().UTC().Format(time.RFC3339)
	id, err := memory.InsertSkillForTest(db, &memory.GeneratedSkill{
		Slug:            slug,
		Theme:           slug + " theme",
		SourceMemoryIDs: "[]",
		Alpha:           1.0,
		Beta:            1.0,
		Utility:         0.5,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	g.Expect(err).ToNot(HaveOccurred())

	return id
}

// openTestSkillDB opens a temp-dir skills DB for testing.
func openTestSkillDB(t *testing.T) *sql.DB {
	t.Helper()
	g := NewWithT(t)

	db, err := memory.OpenSkillDB(t.TempDir())
	g.Expect(err).ToNot(HaveOccurred())

	t.Cleanup(func() { _ = db.Close() })

	return db
}
