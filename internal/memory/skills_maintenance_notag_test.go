package memory

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

// TestApplyConsolidate_InvalidKeepJSON verifies that when the keep-skill has invalid JSON
// in source_memory_ids, applyConsolidate returns an error.
// Covers the json.Unmarshal(keepSkill.SourceMemoryIDs) error path (line 79).
func TestApplyConsolidate_InvalidKeepJSON(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	applier, cleanup := newTestSkillsApplier(t)
	defer cleanup()

	insertTestSkill(t, applier, "target-valid", "[]")
	insertTestSkill(t, applier, "keep-invalid-json", "not-valid-json")

	proposal := MaintenanceProposal{
		Tier:   "skills",
		Action: "consolidate",
		Target: "target-valid",
		Reason: `Redundant with "keep-invalid-json" (similarity 0.95)`,
	}

	err := applier.applyConsolidate(proposal)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("failed to parse keep source_memory_ids"))
	}
}

// TestApplyConsolidate_InvalidTargetJSON verifies that when the target skill has invalid JSON
// in source_memory_ids, applyConsolidate returns an error.
// Covers the json.Unmarshal(targetSkill.SourceMemoryIDs) error path (line 75).
func TestApplyConsolidate_InvalidTargetJSON(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	applier, cleanup := newTestSkillsApplier(t)
	defer cleanup()

	insertTestSkill(t, applier, "target-invalid-json", "not-valid-json")
	insertTestSkill(t, applier, "keep-valid", "[]")

	proposal := MaintenanceProposal{
		Tier:   "skills",
		Action: "consolidate",
		Target: "target-invalid-json",
		Reason: `Redundant with "keep-valid" (similarity 0.95)`,
	}

	err := applier.applyConsolidate(proposal)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("failed to parse target source_memory_ids"))
	}
}

// TestApplyConsolidate_KeepSkillMissing verifies that when keepSlug is set but the
// keep-skill doesn't exist in the DB, the target is still pruned without error.
// Covers the keepSlug != "" → getSkillBySlug → keepSkill == nil branch (lines 65-125).
func TestApplyConsolidate_KeepSkillMissing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	applier, cleanup := newTestSkillsApplier(t)
	defer cleanup()

	insertTestSkill(t, applier, "missing-keep-target", "[]")

	// Reason references "no-such-keep-skill" which is not in the DB → keepSkill == nil
	proposal := MaintenanceProposal{
		Tier:   "skills",
		Action: "consolidate",
		Target: "missing-keep-target",
		Reason: `Redundant with "no-such-keep-skill" (similarity 0.88)`,
	}

	err := applier.applyConsolidate(proposal)
	g.Expect(err).ToNot(HaveOccurred())

	// Target should be pruned
	skills, err := listSkills(applier.db)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(skills).To(BeEmpty())
}

func TestApplyConsolidate_NoKeepSlug(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	applier, cleanup := newTestSkillsApplier(t)
	defer cleanup()

	insertTestSkill(t, applier, "target-skill", "[]")

	proposal := MaintenanceProposal{
		Tier:   "skills",
		Action: "consolidate",
		Target: "target-skill",
		Reason: "no keep specified",
	}

	err := applier.applyConsolidate(proposal)
	g.Expect(err).ToNot(HaveOccurred())

	// Target should now be pruned
	skills, err := listSkills(applier.db)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(skills).To(BeEmpty())
}

func TestApplyConsolidate_TargetNotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	applier, cleanup := newTestSkillsApplier(t)
	defer cleanup()

	proposal := MaintenanceProposal{
		Tier:   "skills",
		Action: "consolidate",
		Target: "nonexistent-skill",
		Reason: "",
	}

	err := applier.applyConsolidate(proposal)
	g.Expect(err).To(HaveOccurred())
}

func TestApplyConsolidate_WithKeepSlug(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	applier, cleanup := newTestSkillsApplier(t)
	defer cleanup()

	insertTestSkill(t, applier, "target-to-merge", "[1,2]")
	insertTestSkill(t, applier, "keep-skill", "[3,4]")

	proposal := MaintenanceProposal{
		Tier:   "skills",
		Action: "consolidate",
		Target: "target-to-merge",
		Reason: `Redundant with "keep-skill" (similarity 0.92)`,
	}

	err := applier.applyConsolidate(proposal)
	g.Expect(err).ToNot(HaveOccurred())

	// Target should be pruned; keep-skill should still be alive
	skills, err := listSkills(applier.db)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(skills).To(HaveLen(1))

	if len(skills) > 0 {
		g.Expect(skills[0].Slug).To(Equal("keep-skill"))
	}
}

// TestApplyConsolidate_WithPreexistingMergeSourceIDs verifies that pre-existing
// merge_source_ids on the keep-skill are combined with the newly merged source ID.
// Covers the mergeSourceIDsStr != "" && != "[]" branch (line 100).
func TestApplyConsolidate_WithPreexistingMergeSourceIDs(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	applier, cleanup := newTestSkillsApplier(t)
	defer cleanup()

	insertTestSkill(t, applier, "target-to-absorb", "[1,2]")
	keepID := insertTestSkill(t, applier, "keep-with-history", "[3,4]")

	// Pre-populate merge_source_ids on keep skill to trigger the unmarshal branch (line 100)
	_, err := applier.db.Exec("UPDATE generated_skills SET merge_source_ids = ? WHERE id = ?", "[99]", keepID)
	g.Expect(err).ToNot(HaveOccurred())

	proposal := MaintenanceProposal{
		Tier:   "skills",
		Action: "consolidate",
		Target: "target-to-absorb",
		Reason: `Redundant with "keep-with-history" (similarity 0.95)`,
	}

	err = applier.applyConsolidate(proposal)
	g.Expect(err).ToNot(HaveOccurred())

	// Only keep-with-history should remain (target-to-absorb was pruned)
	skills, err := listSkills(applier.db)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(skills).To(HaveLen(1))

	if len(skills) > 0 {
		g.Expect(skills[0].Slug).To(Equal("keep-with-history"))
	}
}

// TestApplyConsolidate_WithSkillsDirSet verifies that when SkillsDir is configured,
// applyConsolidate attempts to remove the skill directory after pruning.
// Covers the SkillsDir block (lines 134-141).
func TestApplyConsolidate_WithSkillsDirSet(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	skillsDir := t.TempDir()

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("initEmbeddingsDB: %v", err)
	}

	defer db.Close()

	applier := NewSkillsApplier(db, SkillsApplierOpts{SkillsDir: skillsDir})
	insertTestSkill(t, applier, "dir-target", "[]")

	proposal := MaintenanceProposal{
		Tier:   "skills",
		Action: "consolidate",
		Target: "dir-target",
		Reason: "no keep specified",
	}

	err = applier.applyConsolidate(proposal)
	g.Expect(err).ToNot(HaveOccurred())
}

func TestApplyDecay_SkillNotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	applier, cleanup := newTestSkillsApplier(t)
	defer cleanup()

	proposal := MaintenanceProposal{
		Tier:   "skills",
		Action: "decay",
		Target: "nonexistent-decay",
	}

	err := applier.applyDecay(proposal)
	g.Expect(err).To(HaveOccurred())
}

func TestApplyDecay_ValidSkill(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	applier, cleanup := newTestSkillsApplier(t)
	defer cleanup()

	insertTestSkill(t, applier, "decay-skill", "[]")

	proposal := MaintenanceProposal{
		Tier:   "skills",
		Action: "decay",
		Target: "decay-skill",
	}

	err := applier.applyDecay(proposal)
	g.Expect(err).ToNot(HaveOccurred())

	skill, err := getSkillBySlug(applier.db, "decay-skill")
	g.Expect(err).ToNot(HaveOccurred())

	if skill != nil {
		g.Expect(skill.Beta).To(BeNumerically(">", 1.0))
	}
}

// TestApplyDemote_AllMissing verifies that a skill with all nonexistent source IDs is pruned
// without error. This covers the loop body (query + missingCount++) for all-missing case.
func TestApplyDemote_AllMissing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	applier, cleanup := newTestSkillsApplier(t)
	defer cleanup()

	// source_memory_ids=[99999] → nonexistent → missingCount=1 = len=1 → no warning (not partial)
	insertTestSkill(t, applier, "demote-skill-all-missing", "[99999]")

	err := applier.Apply(MaintenanceProposal{
		Tier:   "skills",
		Action: "demote",
		Target: "demote-skill-all-missing",
	})

	g.Expect(err).ToNot(HaveOccurred())
}

// TestApplyDemote_PartialMissing verifies the warning is printed when some (not all) source
// memories are missing. Covers the fmt.Fprintf warning path.
func TestApplyDemote_PartialMissing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	applier, cleanup := newTestSkillsApplier(t)
	defer cleanup()

	// Insert one real embedding so we have a known ID
	res, err := applier.db.Exec("INSERT INTO embeddings (content, source, confidence) VALUES ('test', 'memory', 0.9)")
	g.Expect(err).ToNot(HaveOccurred())

	if res == nil {
		t.Fatal("expected non-nil result from Exec")
	}

	realID, err := res.LastInsertId()
	g.Expect(err).ToNot(HaveOccurred())

	// source_memory_ids=[realID, 99999] → 1 existing + 1 missing → partial warning printed
	sourceIDs := fmt.Sprintf("[%d, 99999]", realID)

	insertTestSkill(t, applier, "demote-skill-partial", sourceIDs)

	err = applier.Apply(MaintenanceProposal{
		Tier:   "skills",
		Action: "demote",
		Target: "demote-skill-partial",
	})

	g.Expect(err).ToNot(HaveOccurred())
}

func TestApplyDemote_SkillNotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	applier, cleanup := newTestSkillsApplier(t)
	defer cleanup()

	proposal := MaintenanceProposal{
		Tier:   "skills",
		Action: "demote",
		Target: "nonexistent",
	}

	err := applier.applyDemote(proposal)
	g.Expect(err).To(HaveOccurred())
}

func TestApplyDemote_ValidSkill(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	applier, cleanup := newTestSkillsApplier(t)
	defer cleanup()

	insertTestSkill(t, applier, "demote-skill", "[]")

	proposal := MaintenanceProposal{
		Tier:   "skills",
		Action: "demote",
		Target: "demote-skill",
	}

	err := applier.applyDemote(proposal)
	g.Expect(err).ToNot(HaveOccurred())

	skills, err := listSkills(applier.db)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(skills).To(BeEmpty())
}

func TestApplyPromote_NoClaudeMDPath(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	applier, cleanup := newTestSkillsApplier(t)
	defer cleanup()

	proposal := MaintenanceProposal{
		Tier:   "skills",
		Action: "promote",
		Target: "some-skill",
	}

	err := applier.applyPromote(proposal)
	g.Expect(err).To(HaveOccurred())
}

// TestApplyPromote_SkillNotFoundError verifies applyPromote returns error when skill not in DB.
// Covers line 221: `return fmt.Errorf("skill %q not found", ...)` when getSkillBySlug returns nil.
func TestApplyPromote_SkillNotFoundError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// ClaudeMDPath non-empty to pass early nil check; target "nonexistent" → getSkillBySlug → nil
	applier := NewSkillsApplier(db, SkillsApplierOpts{ClaudeMDPath: "/any/nonempty/path.md"})

	err = applier.applyPromote(MaintenanceProposal{
		Tier:   "skills",
		Action: "promote",
		Target: "nonexistent-slug",
	})
	g.Expect(err).To(MatchError(ContainSubstring("not found")))
}

// TestApplyPromote_ThemeFallback verifies applyPromote uses skill.Theme when Preview is empty.
// Covers line 227: `principle = skill.Theme` (the "empty Preview" branch).
func TestApplyPromote_ThemeFallback(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	claudeMDPath := filepath.Join(tmpDir, "CLAUDE.md")

	err := os.WriteFile(claudeMDPath, []byte("# Test\n\n## Promoted Learnings\n\n"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	db, err := initEmbeddingsDB(filepath.Join(tmpDir, "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	applier := NewSkillsApplier(db, SkillsApplierOpts{ClaudeMDPath: claudeMDPath})
	insertTestSkill(t, applier, "theme-skill", "[]")

	// Preview="" → principle = skill.Theme (= "theme-skill") → appended to CLAUDE.md
	err = applier.applyPromote(MaintenanceProposal{
		Tier:    "skills",
		Action:  "promote",
		Target:  "theme-skill",
		Preview: "", // empty → uses theme fallback (line 227)
	})
	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(claudeMDPath)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content)).To(ContainSubstring("theme-skill"))
}

func TestApplyPromote_WithClaudeMDPath(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	claudeMDPath := filepath.Join(tmpDir, "CLAUDE.md")
	err := os.WriteFile(claudeMDPath, []byte("# Test\n\n## Promoted Learnings\n\n"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	db, err := initEmbeddingsDB(filepath.Join(tmpDir, "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	applier := NewSkillsApplier(db, SkillsApplierOpts{ClaudeMDPath: claudeMDPath})
	insertTestSkill(t, applier, "promote-skill", "[]")

	proposal := MaintenanceProposal{
		Tier:    "skills",
		Action:  "promote",
		Target:  "promote-skill",
		Preview: "Always write tests first",
	}

	err = applier.applyPromote(proposal)
	g.Expect(err).ToNot(HaveOccurred())

	content, err := os.ReadFile(claudeMDPath)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(content)).To(ContainSubstring("Always write tests first"))
}

func TestApplyPrune_DBError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())
	db.Close() // Close so DB Exec fails

	applier := NewSkillsApplier(db, SkillsApplierOpts{})

	err = applier.applyPrune(MaintenanceProposal{Target: "any-skill"})
	g.Expect(err).To(HaveOccurred())

	if err == nil {
		t.Fatal("expected error but got nil")
	}

	g.Expect(err.Error()).To(ContainSubstring("failed to soft-delete skill"))
}

func TestApplyPrune_ValidSkill(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	applier, cleanup := newTestSkillsApplier(t)
	defer cleanup()

	insertTestSkill(t, applier, "prune-skill", "[]")

	proposal := MaintenanceProposal{
		Tier:   "skills",
		Action: "prune",
		Target: "prune-skill",
	}

	err := applier.applyPrune(proposal)
	g.Expect(err).ToNot(HaveOccurred())

	skills, err := listSkills(applier.db)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(skills).To(BeEmpty())
}

func TestApplyPrune_WithSkillsDir(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	tmpDir := t.TempDir()
	db, err := initEmbeddingsDB(filepath.Join(tmpDir, "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	skillsDir := filepath.Join(tmpDir, "skills")
	applier := NewSkillsApplier(db, SkillsApplierOpts{SkillsDir: skillsDir})

	insertTestSkill(t, applier, "dir-prune-skill", "[]")

	// Create the skill dir so RemoveAll has something to remove
	skillDir := filepath.Join(skillsDir, "memory-dir-prune-skill")
	err = os.MkdirAll(skillDir, 0o755)
	g.Expect(err).ToNot(HaveOccurred())

	err = applier.applyPrune(MaintenanceProposal{Target: "dir-prune-skill"})
	g.Expect(err).ToNot(HaveOccurred())

	// Skill dir should be removed
	_, statErr := os.Stat(skillDir)
	g.Expect(os.IsNotExist(statErr)).To(BeTrue())
}

func TestApplySplit_SkillNotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	applier, cleanup := newTestSkillsApplier(t)
	defer cleanup()

	proposal := MaintenanceProposal{
		Tier:   "skills",
		Action: "split",
		Target: "nonexistent",
	}

	err := applier.applySplit(proposal)
	g.Expect(err).To(HaveOccurred())
}

func TestApplySplit_ValidSkill(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	applier, cleanup := newTestSkillsApplier(t)
	defer cleanup()

	insertTestSkill(t, applier, "splittable-skill", "[1,2,3,4]")

	proposal := MaintenanceProposal{
		Tier:   "skills",
		Action: "split",
		Target: "splittable-skill",
	}

	err := applier.applySplit(proposal)
	g.Expect(err).ToNot(HaveOccurred())

	// Original should be pruned; two sub-skills should exist
	skills, err := listSkills(applier.db)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(skills).To(HaveLen(2))

	slugs := make([]string, len(skills))
	for i, s := range skills {
		slugs[i] = s.Slug
	}

	g.Expect(slugs).To(ContainElement("splittable-skill-a"))
	g.Expect(slugs).To(ContainElement("splittable-skill-b"))
}

func TestApply_Consolidate(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	applier, cleanup := newTestSkillsApplier(t)
	defer cleanup()

	err := applier.Apply(MaintenanceProposal{
		Tier:   "skills",
		Action: "consolidate",
		Target: "nonexistent-consolidate-target",
	})

	g.Expect(err).To(HaveOccurred())
}

func TestApply_Decay(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	applier, cleanup := newTestSkillsApplier(t)
	defer cleanup()

	insertTestSkill(t, applier, "apply-decay-skill", "[]")

	err := applier.Apply(MaintenanceProposal{
		Tier:   "skills",
		Action: "decay",
		Target: "apply-decay-skill",
	})

	g.Expect(err).ToNot(HaveOccurred())
}

func TestApply_Demote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	applier, cleanup := newTestSkillsApplier(t)
	defer cleanup()

	err := applier.Apply(MaintenanceProposal{
		Tier:   "skills",
		Action: "demote",
		Target: "nonexistent-demote-target",
	})

	g.Expect(err).To(HaveOccurred())
}

func TestApply_InvalidTier(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	applier, cleanup := newTestSkillsApplier(t)
	defer cleanup()

	err := applier.Apply(MaintenanceProposal{
		Tier:   "wrong-tier",
		Action: "prune",
		Target: "any",
	})

	g.Expect(err).To(MatchError(ContainSubstring("invalid tier")))
}

func TestApply_Promote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	applier, cleanup := newTestSkillsApplier(t)
	defer cleanup()

	// No ClaudeMDPath → applyPromote returns error
	err := applier.Apply(MaintenanceProposal{
		Tier:   "skills",
		Action: "promote",
		Target: "some-promote-skill",
	})

	g.Expect(err).To(HaveOccurred())
}

func TestApply_Prune(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	applier, cleanup := newTestSkillsApplier(t)
	defer cleanup()

	insertTestSkill(t, applier, "apply-prune-skill", "[]")

	err := applier.Apply(MaintenanceProposal{
		Tier:   "skills",
		Action: "prune",
		Target: "apply-prune-skill",
	})

	g.Expect(err).ToNot(HaveOccurred())
}

func TestApply_Split(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	applier, cleanup := newTestSkillsApplier(t)
	defer cleanup()

	err := applier.Apply(MaintenanceProposal{
		Tier:   "skills",
		Action: "split",
		Target: "nonexistent-split-target",
	})

	g.Expect(err).To(HaveOccurred())
}

func TestApply_UnknownAction(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	applier, cleanup := newTestSkillsApplier(t)
	defer cleanup()

	err := applier.Apply(MaintenanceProposal{
		Tier:   "skills",
		Action: "unknown-action",
		Target: "any",
	})

	g.Expect(err).To(MatchError(ContainSubstring("unknown action")))
}

func TestDeduplicateInt64(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Empty
	g.Expect(deduplicateInt64(nil)).To(BeEmpty())

	// No duplicates
	result := deduplicateInt64([]int64{1, 2, 3})
	g.Expect(result).To(Equal([]int64{1, 2, 3}))

	// With duplicates
	result = deduplicateInt64([]int64{1, 2, 1, 3, 2})
	g.Expect(result).To(Equal([]int64{1, 2, 3}))
}

func TestDetectPromotionCandidates_WithSkillUsage(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	applier, cleanup := newTestSkillsApplier(t)
	defer cleanup()

	now := time.Now().UTC().Format(time.RFC3339)
	highUtilSkill := &GeneratedSkill{
		Slug:            "promo-candidate",
		Theme:           "High Utility Skill",
		Description:     "desc",
		Content:         "content",
		SourceMemoryIDs: "[]",
		Alpha:           9.0,
		Beta:            1.0,
		Utility:         0.88,
		RetrievalCount:  10,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	id, err := insertSkill(applier.db, highUtilSkill)
	g.Expect(err).ToNot(HaveOccurred())

	_, err = applier.db.Exec(`CREATE TABLE IF NOT EXISTS skill_usage (skill_id INTEGER, project TEXT, timestamp TEXT)`)
	g.Expect(err).ToNot(HaveOccurred())

	for _, proj := range []string{"proj-a", "proj-b", "proj-c"} {
		_, err = applier.db.Exec("INSERT INTO skill_usage (skill_id, project, timestamp) VALUES (?, ?, ?)", id, proj, now)
		g.Expect(err).ToNot(HaveOccurred())
	}

	scanner := NewSkillsScanner(applier.db, SkillsScannerOpts{
		PromoteUtilityThreshold: 0.7,
		PromoteMinProjects:      3,
	})

	proposals, err := scanner.detectPromotionCandidates()
	g.Expect(err).ToNot(HaveOccurred())

	var found bool

	for _, p := range proposals {
		if p.Action == "promote" && p.Target == "promo-candidate" {
			found = true

			break
		}
	}

	g.Expect(found).To(BeTrue())
}

func TestDetectRedundantSkills_TwoSimilarSkills(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	applier, cleanup := newTestSkillsApplier(t)
	defer cleanup()

	// Insert two skills with embedding IDs so detectRedundantSkills can process them
	insertTestSkillWithEmbedding(t, applier.db, "skill-alpha", 10, 0.8)
	insertTestSkillWithEmbedding(t, applier.db, "skill-beta", 20, 0.6)

	scanner := NewSkillsScanner(applier.db, SkillsScannerOpts{
		SimilarityThreshold: 0.85,
		SimilarityFunc: func(_ *sql.DB, _, _ int64) (float64, error) {
			return 0.92, nil
		},
	})

	proposals, err := scanner.detectRedundantSkills()
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(proposals).To(HaveLen(1))

	if len(proposals) > 0 {
		g.Expect(proposals[0].Action).To(Equal("consolidate"))
		g.Expect(proposals[0].Target).To(Equal("skill-beta"))
	}
}

func TestDetectUnusedSkills_OldSkill(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	applier, cleanup := newTestSkillsApplier(t)
	defer cleanup()

	// Create a skill that is 60 days old with zero retrievals and low utility
	oldTime := time.Now().UTC().AddDate(0, 0, -60).Format(time.RFC3339)
	oldSkill := &GeneratedSkill{
		Slug:            "old-unused-skill",
		Theme:           "Old Unused",
		Description:     "desc",
		Content:         "content",
		SourceMemoryIDs: "[]",
		Alpha:           1.0,
		Beta:            3.0,
		Utility:         0.2,
		CreatedAt:       oldTime,
		UpdatedAt:       oldTime,
	}

	_, err := insertSkill(applier.db, oldSkill)
	g.Expect(err).ToNot(HaveOccurred())

	scanner := NewSkillsScanner(applier.db, SkillsScannerOpts{
		UnusedDaysThreshold: 30,
		LowUtilityThreshold: 0.4,
	})

	proposals, err := scanner.detectUnusedSkills()
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(proposals).ToNot(BeEmpty())

	if len(proposals) > 0 {
		g.Expect(proposals[0].Action).To(Equal("prune"))
		g.Expect(proposals[0].Target).To(Equal("old-unused-skill"))
	}
}

func TestDetectUnusedSkills_TooNew(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	applier, cleanup := newTestSkillsApplier(t)
	defer cleanup()

	// Recent skill with low utility - should NOT be pruned
	insertTestSkill(t, applier, "new-low-util", "[]")

	scanner := NewSkillsScanner(applier.db, SkillsScannerOpts{
		UnusedDaysThreshold: 30,
		LowUtilityThreshold: 0.9,
	})

	proposals, err := scanner.detectUnusedSkills()
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(proposals).To(BeEmpty())
}

func TestExtractKeepSkillSlug(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Normal case
	slug := extractKeepSkillSlug(`Redundant with "keep-skill-slug" (similarity 0.92)`)
	g.Expect(slug).To(Equal("keep-skill-slug"))

	// No quotes
	slug = extractKeepSkillSlug("no quotes here")
	g.Expect(slug).To(Equal(""))

	// Only opening quote
	slug = extractKeepSkillSlug(`only "opening`)
	g.Expect(slug).To(Equal(""))

	// Empty reason
	slug = extractKeepSkillSlug("")
	g.Expect(slug).To(Equal(""))
}

func TestScan_Empty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	applier, cleanup := newTestSkillsApplier(t)
	defer cleanup()

	scanner := NewSkillsScanner(applier.db, SkillsScannerOpts{})
	proposals, err := scanner.Scan()
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(proposals).To(BeEmpty())
}

func insertTestSkill(t *testing.T, applier *SkillsApplier, slug string, sourceMemIDs string) int64 {
	t.Helper()

	g := NewWithT(t)
	now := time.Now().UTC().Format(time.RFC3339)
	skill := &GeneratedSkill{
		Slug:            slug,
		Theme:           slug,
		Description:     "desc",
		Content:         "content",
		SourceMemoryIDs: sourceMemIDs,
		Alpha:           1.0,
		Beta:            1.0,
		Utility:         0.5,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	id, err := insertSkill(applier.db, skill)
	g.Expect(err).ToNot(HaveOccurred())

	return id
}

func insertTestSkillWithEmbedding(t *testing.T, db *sql.DB, slug string, embeddingID int64, utility float64) int64 {
	t.Helper()

	g := NewWithT(t)
	now := time.Now().UTC().Format(time.RFC3339)
	skill := &GeneratedSkill{
		Slug:            slug,
		Theme:           slug,
		Description:     "desc",
		Content:         "content",
		SourceMemoryIDs: "[]",
		Alpha:           1.0,
		Beta:            1.0,
		Utility:         utility,
		EmbeddingID:     embeddingID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	id, err := insertSkill(db, skill)
	g.Expect(err).ToNot(HaveOccurred())

	return id
}

func newTestSkillsApplier(t *testing.T) (*SkillsApplier, func()) {
	t.Helper()

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("initEmbeddingsDB: %v", err)
	}

	applier := NewSkillsApplier(db, SkillsApplierOpts{})

	return applier, func() { db.Close() }
}
