//go:build sqlite_fts5

package memory

import (
	"database/sql"
	"encoding/json"
	"slices"
	"testing"
	"time"
)

// TestApplyConsolidate_KeepSkillNotFound tests fallback when keep-skill not found.
func TestApplyConsolidate_KeepSkillNotFound(t *testing.T) {
	db := setupApplyTestDB(t)
	now := time.Now().UTC().Format(time.RFC3339)

	// Insert only the merge-skill (keep-skill doesn't exist)
	mergeSkill := &GeneratedSkill{
		Slug:            "orphan-skill",
		Theme:           "Orphan",
		Description:     "no keep target",
		Content:         "orphan content",
		SourceMemoryIDs: `[1,2]`,
		Alpha:           3.0,
		Beta:            1.0,
		Utility:         0.6,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	mergeID, err := insertSkill(db, mergeSkill)
	if err != nil {
		t.Fatalf("failed to insert merge skill: %v", err)
	}

	// Create applier
	applier := &SkillsApplier{db: db, opts: SkillsApplierOpts{}}

	// Apply consolidate proposal - keep-skill doesn't exist
	proposal := MaintenanceProposal{
		Tier:   "skills",
		Action: "consolidate",
		Target: "orphan-skill",
		Reason: `Redundant with "nonexistent-skill" (similarity 0.90)`,
	}

	err = applier.applyConsolidate(proposal)
	if err != nil {
		t.Fatalf("applyConsolidate failed: %v", err)
	}

	// Verify orphan-skill is pruned (fallback behavior)
	var pruned int

	err = db.QueryRow("SELECT pruned FROM generated_skills WHERE id = ?", mergeID).Scan(&pruned)
	if err != nil {
		t.Fatalf("failed to query pruned status: %v", err)
	}

	if pruned != 1 {
		t.Errorf("expected orphan-skill to be pruned (fallback), got pruned=%d", pruned)
	}
}

// TestApplyConsolidate_MergesSourceIDs tests consolidation merges source memory IDs.
func TestApplyConsolidate_MergesSourceIDs(t *testing.T) {
	db := setupApplyTestDB(t)
	now := time.Now().UTC().Format(time.RFC3339)

	// Insert two skills with different source memories
	keepSkill := &GeneratedSkill{
		Slug:            "keep-skill",
		Theme:           "Keep",
		Description:     "high utility",
		Content:         "keep content",
		SourceMemoryIDs: `[1,2]`,
		Alpha:           5.0,
		Beta:            1.0,
		Utility:         0.8,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	keepID, err := insertSkill(db, keepSkill)
	if err != nil {
		t.Fatalf("failed to insert keep skill: %v", err)
	}

	mergeSkill := &GeneratedSkill{
		Slug:            "merge-skill",
		Theme:           "Merge",
		Description:     "lower utility",
		Content:         "merge content",
		SourceMemoryIDs: `[3,4]`,
		Alpha:           3.0,
		Beta:            1.0,
		Utility:         0.7,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	mergeID, err := insertSkill(db, mergeSkill)
	if err != nil {
		t.Fatalf("failed to insert merge skill: %v", err)
	}

	// Create applier
	applier := &SkillsApplier{db: db, opts: SkillsApplierOpts{}}

	// Apply consolidate proposal - reason contains keep-skill slug
	proposal := MaintenanceProposal{
		Tier:   "skills",
		Action: "consolidate",
		Target: "merge-skill",
		Reason: `Redundant with "keep-skill" (similarity 0.90)`,
	}

	err = applier.applyConsolidate(proposal)
	if err != nil {
		t.Fatalf("applyConsolidate failed: %v", err)
	}

	// Verify merge-skill is pruned
	var mergedPruned int

	err = db.QueryRow("SELECT pruned FROM generated_skills WHERE id = ?", mergeID).Scan(&mergedPruned)
	if err != nil {
		t.Fatalf("failed to query merge skill pruned status: %v", err)
	}

	if mergedPruned != 1 {
		t.Errorf("expected merge-skill to be pruned, got pruned=%d", mergedPruned)
	}

	// Verify keep-skill has combined source_memory_ids
	var (
		sourceMemIDs   string
		mergeSourceIDs string
	)

	err = db.QueryRow("SELECT source_memory_ids, COALESCE(merge_source_ids, '') FROM generated_skills WHERE id = ?",
		keepID).Scan(&sourceMemIDs, &mergeSourceIDs)
	if err != nil {
		t.Fatalf("failed to query keep skill: %v", err)
	}

	// Parse source_memory_ids
	var memIDs []int64
	if err := json.Unmarshal([]byte(sourceMemIDs), &memIDs); err != nil {
		t.Fatalf("failed to unmarshal source_memory_ids: %v", err)
	}

	// Should contain all 4 IDs (1,2,3,4)
	if len(memIDs) != 4 {
		t.Errorf("expected 4 source memory IDs, got %d: %v", len(memIDs), memIDs)
	}

	// Verify merge_source_ids contains the merged skill ID
	var mergeIDs []int64
	if mergeSourceIDs != "" && mergeSourceIDs != "[]" {
		if err := json.Unmarshal([]byte(mergeSourceIDs), &mergeIDs); err != nil {
			t.Fatalf("failed to unmarshal merge_source_ids: %v", err)
		}
	}

	foundMergeID := slices.Contains(mergeIDs, mergeID)

	if !foundMergeID {
		t.Errorf("expected merge_source_ids to contain %d, got %v", mergeID, mergeIDs)
	}
}

// TestApplyDemote_SourcesExist tests demotion when source embeddings exist.
func TestApplyDemote_SourcesExist(t *testing.T) {
	db := setupApplyTestDB(t)
	now := time.Now().UTC().Format(time.RFC3339)

	// Insert embeddings (source memories)
	for i := 1; i <= 3; i++ {
		_, err := db.Exec(`INSERT INTO embeddings (id, content, source, embedding_id) VALUES (?, ?, ?, ?)`,
			i, "memory content", "test", i)
		if err != nil {
			t.Fatalf("failed to insert embedding: %v", err)
		}
	}

	// Insert skill
	skill := &GeneratedSkill{
		Slug:            "demote-skill",
		Theme:           "Demote Test",
		Description:     "test",
		Content:         "test",
		SourceMemoryIDs: `[1,2,3]`,
		Alpha:           1.0,
		Beta:            1.0,
		Utility:         0.5,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	skillID, err := insertSkill(db, skill)
	if err != nil {
		t.Fatalf("failed to insert skill: %v", err)
	}

	// Create applier
	applier := &SkillsApplier{db: db, opts: SkillsApplierOpts{}}

	// Apply demote proposal
	proposal := MaintenanceProposal{
		Tier:   "skills",
		Action: "demote",
		Target: "demote-skill",
		Reason: "single project",
	}

	err = applier.applyDemote(proposal)
	if err != nil {
		t.Fatalf("applyDemote failed: %v", err)
	}

	// Verify skill is pruned
	var pruned int

	err = db.QueryRow("SELECT pruned FROM generated_skills WHERE id = ?", skillID).Scan(&pruned)
	if err != nil {
		t.Fatalf("failed to query pruned status: %v", err)
	}

	if pruned != 1 {
		t.Errorf("expected skill to be pruned, got pruned=%d", pruned)
	}
}

// TestApplyDemote_SourcesMissing tests demotion still works when source embeddings are gone.
func TestApplyDemote_SourcesMissing(t *testing.T) {
	db := setupApplyTestDB(t)
	now := time.Now().UTC().Format(time.RFC3339)

	// Insert skill with source_memory_ids that don't exist
	skill := &GeneratedSkill{
		Slug:            "missing-sources-skill",
		Theme:           "Missing Sources",
		Description:     "test",
		Content:         "test",
		SourceMemoryIDs: `[99,100,101]`,
		Alpha:           1.0,
		Beta:            1.0,
		Utility:         0.5,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	skillID, err := insertSkill(db, skill)
	if err != nil {
		t.Fatalf("failed to insert skill: %v", err)
	}

	// Create applier
	applier := &SkillsApplier{db: db, opts: SkillsApplierOpts{}}

	// Apply demote proposal - should still prune even though sources missing
	proposal := MaintenanceProposal{
		Tier:   "skills",
		Action: "demote",
		Target: "missing-sources-skill",
		Reason: "single project",
	}

	err = applier.applyDemote(proposal)
	if err != nil {
		t.Fatalf("applyDemote failed: %v", err)
	}

	// Verify skill is pruned
	var pruned int

	err = db.QueryRow("SELECT pruned FROM generated_skills WHERE id = ?", skillID).Scan(&pruned)
	if err != nil {
		t.Fatalf("failed to query pruned status: %v", err)
	}

	if pruned != 1 {
		t.Errorf("expected skill to be pruned, got pruned=%d", pruned)
	}
}

// TestApplySplit_Success tests splitting a skill with 4 source memories into 2 new skills.
func TestApplySplit_Success(t *testing.T) {
	db := setupApplyTestDB(t)
	now := time.Now().UTC().Format(time.RFC3339)

	// Insert embeddings (source memories)
	for i := 1; i <= 4; i++ {
		_, err := db.Exec(`INSERT INTO embeddings (id, content, source, embedding_id) VALUES (?, ?, ?, ?)`,
			i, "memory content", "test", i)
		if err != nil {
			t.Fatalf("failed to insert embedding: %v", err)
		}
	}

	// Insert skill with 4 source memories
	skill := &GeneratedSkill{
		Slug:            "test-skill",
		Theme:           "Test Theme",
		Description:     "test description",
		Content:         "test content",
		SourceMemoryIDs: `[1,2,3,4]`,
		Alpha:           5.0,
		Beta:            1.0,
		Utility:         0.8,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	skillID, err := insertSkill(db, skill)
	if err != nil {
		t.Fatalf("failed to insert skill: %v", err)
	}

	// Create applier
	applier := &SkillsApplier{db: db, opts: SkillsApplierOpts{}}

	// Apply split proposal
	proposal := MaintenanceProposal{
		Tier:   "skills",
		Action: "split",
		Target: "test-skill",
		Reason: "skill too broad",
	}

	err = applier.applySplit(proposal)
	if err != nil {
		t.Fatalf("applySplit failed: %v", err)
	}

	// Verify original skill is pruned
	var pruned int

	err = db.QueryRow("SELECT pruned FROM generated_skills WHERE id = ?", skillID).Scan(&pruned)
	if err != nil {
		t.Fatalf("failed to query pruned status: %v", err)
	}

	if pruned != 1 {
		t.Errorf("expected original skill to be pruned, got pruned=%d", pruned)
	}

	// Verify 2 new skills created
	var count int

	err = db.QueryRow("SELECT COUNT(*) FROM generated_skills WHERE split_from_id = ? AND pruned = 0", skillID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to count split skills: %v", err)
	}

	if count != 2 {
		t.Errorf("expected 2 split skills, got %d", count)
	}

	// Verify new skills have correct slugs
	rows, err := db.Query("SELECT slug FROM generated_skills WHERE split_from_id = ? AND pruned = 0 ORDER BY slug", skillID)
	if err != nil {
		t.Fatalf("failed to query split skills: %v", err)
	}
	defer rows.Close()

	expectedSlugs := []string{"test-skill-a", "test-skill-b"}
	i := 0

	for rows.Next() {
		var slug string
		if err := rows.Scan(&slug); err != nil {
			t.Fatalf("failed to scan slug: %v", err)
		}

		if i >= len(expectedSlugs) {
			t.Errorf("too many split skills")
			break
		}

		if slug != expectedSlugs[i] {
			t.Errorf("expected slug %q, got %q", expectedSlugs[i], slug)
		}

		i++
	}
}

// TestApplySplit_TooFewMemories tests that splitting fails with fewer than 2 source memories.
func TestApplySplit_TooFewMemories(t *testing.T) {
	db := setupApplyTestDB(t)
	now := time.Now().UTC().Format(time.RFC3339)

	// Insert embedding
	_, err := db.Exec(`INSERT INTO embeddings (id, content, source, embedding_id) VALUES (?, ?, ?, ?)`,
		1, "memory content", "test", 1)
	if err != nil {
		t.Fatalf("failed to insert embedding: %v", err)
	}

	// Insert skill with only 1 source memory
	skill := &GeneratedSkill{
		Slug:            "single-memory-skill",
		Theme:           "Single Memory",
		Description:     "test",
		Content:         "test",
		SourceMemoryIDs: `[1]`,
		Alpha:           1.0,
		Beta:            1.0,
		Utility:         0.5,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	_, err = insertSkill(db, skill)
	if err != nil {
		t.Fatalf("failed to insert skill: %v", err)
	}

	// Create applier
	applier := &SkillsApplier{db: db, opts: SkillsApplierOpts{}}

	// Apply split proposal - should fail
	proposal := MaintenanceProposal{
		Tier:   "skills",
		Action: "split",
		Target: "single-memory-skill",
		Reason: "test",
	}

	err = applier.applySplit(proposal)
	if err == nil {
		t.Fatal("expected error for skill with only 1 source memory, got nil")
	}

	if !contains(err.Error(), "fewer than 2 source memories") {
		t.Errorf("expected error about too few memories, got: %v", err)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && indexOfSubstring(s, substr) >= 0))
}

func indexOfSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}

	return -1
}

// setupApplyTestDB creates a temporary database for testing.
func setupApplyTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()

	db, err := InitDBForTest(dir)
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}

	t.Cleanup(func() { _ = db.Close() })

	return db
}
