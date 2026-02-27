package memory

import (
	"os"
	"strconv"
	"testing"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	. "github.com/onsi/gomega"
)

func TestApplyConsolidate_InvalidDeleteID(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// "1,abc" → deleteID parse fails
	proposal := MaintenanceProposal{
		Tier:   "embeddings",
		Action: "consolidate",
		Target: "1,abc",
		Reason: "test",
	}

	err = applyConsolidate(db, proposal)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("invalid delete ID"))
	}
}

func TestApplyConsolidate_InvalidKeepID(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// "abc,123" → keepID parse fails
	proposal := MaintenanceProposal{
		Tier:   "embeddings",
		Action: "consolidate",
		Target: "abc,123",
		Reason: "test",
	}

	err = applyConsolidate(db, proposal)

	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("invalid keep ID"))
	}
}

// ─── applyConsolidate ─────────────────────────────────────────────────────────

func TestApplyConsolidate_InvalidTarget(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Single item in target (missing second ID)
	proposal := MaintenanceProposal{
		Tier:   "embeddings",
		Action: "consolidate",
		Target: "only-one-id",
		Reason: "test",
	}
	err = applyConsolidate(db, proposal)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("invalid consolidate target format"))
	}
}

func TestApplyConsolidate_Success(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Insert keep entry
	keepRes, err := db.Exec(`INSERT INTO embeddings(content, source, confidence, retrieval_count) VALUES('keep this entry', 'test', 0.9, 5)`)
	g.Expect(err).ToNot(HaveOccurred())

	if keepRes == nil {
		t.Fatal("db.Exec returned nil for keep insert")
	}

	keepID, _ := keepRes.LastInsertId()

	// Insert delete entry
	delRes, err := db.Exec(`INSERT INTO embeddings(content, source, confidence, retrieval_count) VALUES('delete this entry', 'test', 0.7, 3)`)
	g.Expect(err).ToNot(HaveOccurred())

	if delRes == nil {
		t.Fatal("db.Exec returned nil for delete insert")
	}

	delID, _ := delRes.LastInsertId()

	proposal := MaintenanceProposal{
		Tier:    "embeddings",
		Action:  "consolidate",
		Target:  strconv.FormatInt(keepID, 10) + "," + strconv.FormatInt(delID, 10),
		Reason:  "Redundant (similarity 0.95)",
		Preview: "Keep: keep this entry\nDelete: delete this entry",
	}

	err = applyConsolidate(db, proposal)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify delete entry is gone
	var count int

	err = db.QueryRow("SELECT COUNT(*) FROM embeddings WHERE id = ?", delID).Scan(&count)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(count).To(Equal(0))

	// Verify keep entry still exists
	err = db.QueryRow("SELECT COUNT(*) FROM embeddings WHERE id = ?", keepID).Scan(&count)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(count).To(Equal(1))
}

func TestApplyConsolidate_WithVecRow(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Insert vec row for the delete entry
	embedding := make([]float32, 384)
	embedding[0] = 1.0

	blob, serErr := sqlite_vec.SerializeFloat32(embedding)
	g.Expect(serErr).ToNot(HaveOccurred())

	vecRes, err := db.Exec("INSERT INTO vec_embeddings(embedding) VALUES(?)", blob)
	g.Expect(err).ToNot(HaveOccurred())

	if vecRes == nil {
		t.Fatal("db.Exec returned nil for vec insert")
	}

	embID, _ := vecRes.LastInsertId()

	keepRes, err := db.Exec(`INSERT INTO embeddings(content, source, confidence, retrieval_count) VALUES('keep', 'test', 0.9, 5)`)
	g.Expect(err).ToNot(HaveOccurred())

	if keepRes == nil {
		t.Fatal("db.Exec returned nil for keep insert")
	}

	keepID, _ := keepRes.LastInsertId()

	// Delete entry has a non-NULL embedding_id pointing at the vec row
	delRes, err := db.Exec(`INSERT INTO embeddings(content, source, confidence, retrieval_count, embedding_id) VALUES('delete-with-vec', 'test', 0.7, 3, ?)`, embID)
	g.Expect(err).ToNot(HaveOccurred())

	if delRes == nil {
		t.Fatal("db.Exec returned nil for delete insert")
	}

	delID, _ := delRes.LastInsertId()

	proposal := MaintenanceProposal{
		Tier:    "embeddings",
		Action:  "consolidate",
		Target:  strconv.FormatInt(keepID, 10) + "," + strconv.FormatInt(delID, 10),
		Reason:  "redundant",
		Preview: "test",
	}

	err = applyConsolidate(db, proposal)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify vec row was cleaned up
	var vecCount int

	err = db.QueryRow("SELECT COUNT(*) FROM vec_embeddings WHERE rowid = ?", embID).Scan(&vecCount)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(vecCount).To(Equal(0))
}

// ─── applyDecay ───────────────────────────────────────────────────────────────

func TestApplyDecay_InvalidTarget(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	proposal := MaintenanceProposal{
		Tier:   "embeddings",
		Action: "decay",
		Target: "not-a-number",
		Reason: "test",
	}

	err = applyDecay(db, proposal)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("invalid target ID"))
	}
}

// ─── applyEmbeddingsAddRationale ─────────────────────────────────────────────

func TestApplyEmbeddingsAddRationale_InvalidTarget(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	proposal := MaintenanceProposal{
		Tier:    "embeddings",
		Action:  "add-rationale",
		Target:  "bad-id",
		Preview: "principle - rationale text",
	}

	err = applyEmbeddingsAddRationale(db, proposal)
	g.Expect(err).To(HaveOccurred())
}

func TestApplyEmbeddingsAddRationale_Success(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	res, err := db.Exec(`INSERT INTO embeddings(content, source) VALUES('principle content', 'test')`)
	g.Expect(err).ToNot(HaveOccurred())

	if res == nil {
		t.Fatal("db.Exec returned nil result")
	}

	id, _ := res.LastInsertId()

	proposal := MaintenanceProposal{
		Tier:    "embeddings",
		Action:  "add-rationale",
		Target:  strconv.FormatInt(id, 10),
		Preview: "Use TDD - ensures code correctness from the start",
	}

	err = applyEmbeddingsAddRationale(db, proposal)
	g.Expect(err).ToNot(HaveOccurred())

	var rationale string

	err = db.QueryRow("SELECT rationale FROM embeddings WHERE id = ?", id).Scan(&rationale)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(rationale).To(ContainSubstring("ensures code correctness"))
}

// ─── applyEmbeddingsProposal ─────────────────────────────────────────────────

func TestApplyEmbeddingsProposal_AddRationale(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	res, err := db.Exec(`INSERT INTO embeddings(content, source) VALUES('some content', 'test')`)
	g.Expect(err).ToNot(HaveOccurred())

	if res == nil {
		t.Fatal("db.Exec returned nil result")
	}

	id, _ := res.LastInsertId()

	proposal := MaintenanceProposal{
		Tier:    "embeddings",
		Action:  "add-rationale",
		Target:  strconv.FormatInt(id, 10),
		Preview: "principle because important reasoning here",
	}

	err = applyEmbeddingsProposal(db, "", "", proposal)
	g.Expect(err).ToNot(HaveOccurred())
}

func TestApplyEmbeddingsProposal_Rewrite(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	res, err := db.Exec(`INSERT INTO embeddings(content, source, flagged_for_rewrite) VALUES('original content', 'test', 1)`)
	g.Expect(err).ToNot(HaveOccurred())

	if res == nil {
		t.Fatal("db.Exec returned nil result")
	}

	id, _ := res.LastInsertId()

	proposal := MaintenanceProposal{
		Tier:    "embeddings",
		Action:  "rewrite",
		Target:  strconv.FormatInt(id, 10),
		Reason:  "Content needs clarification",
		Preview: "rewritten clearer content",
	}

	err = applyEmbeddingsProposal(db, "", "", proposal)
	g.Expect(err).ToNot(HaveOccurred())

	var content string

	err = db.QueryRow("SELECT content FROM embeddings WHERE id = ?", id).Scan(&content)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(content).To(Equal("rewritten clearer content"))
}

func TestApplyEmbeddingsProposal_Split(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	proposal := MaintenanceProposal{
		Tier:   "embeddings",
		Action: "split",
		Target: "1",
		Reason: "Multi-topic",
	}
	err = applyEmbeddingsProposal(db, "", "", proposal)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("not yet implemented"))
	}
}

func TestApplyEmbeddingsProposal_UnknownAction(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	proposal := MaintenanceProposal{
		Tier:   "embeddings",
		Action: "unknown-action",
		Target: "1",
		Reason: "test",
	}
	err = applyEmbeddingsProposal(db, "", "", proposal)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("unknown action"))
	}
}

// ─── applyEmbeddingsRewrite ───────────────────────────────────────────────────

func TestApplyEmbeddingsRewrite_InvalidTarget(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	proposal := MaintenanceProposal{
		Tier:    "embeddings",
		Action:  "rewrite",
		Target:  "not-a-number",
		Preview: "rewritten",
	}

	err = applyEmbeddingsRewrite(db, proposal)
	g.Expect(err).To(HaveOccurred())
}

func TestApplyEmbeddingsRewrite_Success(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	res, err := db.Exec(`INSERT INTO embeddings(content, source) VALUES('before rewrite', 'test')`)
	g.Expect(err).ToNot(HaveOccurred())

	if res == nil {
		t.Fatal("db.Exec returned nil result")
	}

	id, _ := res.LastInsertId()

	proposal := MaintenanceProposal{
		Tier:    "embeddings",
		Action:  "rewrite",
		Target:  strconv.FormatInt(id, 10),
		Preview: "after rewrite content",
	}

	err = applyEmbeddingsRewrite(db, proposal)
	g.Expect(err).ToNot(HaveOccurred())

	var content string

	err = db.QueryRow("SELECT content FROM embeddings WHERE id = ?", id).Scan(&content)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(content).To(Equal("after rewrite content"))
}

func TestApplyPromote_IDNotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	proposal := MaintenanceProposal{
		Tier:   "embeddings",
		Action: "promote",
		Target: "99999",
		Reason: "High value",
	}

	err = applyPromote(db, t.TempDir(), proposal)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("failed to fetch embedding"))
	}
}

func TestApplyPromote_InvalidID(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	proposal := MaintenanceProposal{
		Tier:   "embeddings",
		Action: "promote",
		Target: "not-a-number",

		Reason: "High value",
	}

	err = applyPromote(db, t.TempDir(), proposal)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("invalid target ID"))
	}
}

// ─── applyPromote ────────────────────────────────────────────────────────────

func TestApplyPromote_NoSkillsDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	proposal := MaintenanceProposal{
		Tier:   "embeddings",
		Action: "promote",
		Target: "1",

		Reason: "High value",
	}

	// Empty skillsDir should return error
	err = applyPromote(db, "", proposal)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("skills directory not configured"))
	}
}

func TestApplyPromote_Success(t *testing.T) {
	// NOTE: mutates global DB state via insertSkill; not parallel
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Insert embedding with all required fields
	res, err := db.Exec(`INSERT INTO embeddings(content, principle, source, confidence, retrieval_count)
		VALUES('Always use TDD for better code design', 'use-tdd-principle', 'test', 0.8, 5)`)
	g.Expect(err).ToNot(HaveOccurred())

	if res == nil {
		t.Fatal("db.Exec returned nil result")
	}

	id, _ := res.LastInsertId()

	skillsDir := t.TempDir()

	proposal := MaintenanceProposal{
		Tier:   "embeddings",
		Action: "promote",
		Target: strconv.FormatInt(id, 10),
		Reason: "High retrieval count",
	}

	err = applyPromote(db, skillsDir, proposal)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify a skill file was written
	entries, readErr := os.ReadDir(skillsDir)
	g.Expect(readErr).ToNot(HaveOccurred())
	g.Expect(entries).ToNot(BeEmpty())
}

func TestApplyPromote_UpdatesExistingSkill(t *testing.T) {
	// NOT parallel — calls insertSkill which interacts with global schema
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	skillsDir := t.TempDir()

	res, err := db.Exec(`INSERT INTO embeddings(content, principle, source, confidence, retrieval_count)
		VALUES('Always write failing tests first', 'tdd-principle', 'test', 0.9, 7)`)
	g.Expect(err).ToNot(HaveOccurred())

	if res == nil {
		t.Fatal("db.Exec returned nil result")
	}

	id, _ := res.LastInsertId()

	proposal := MaintenanceProposal{
		Tier:   "embeddings",
		Action: "promote",
		Target: strconv.FormatInt(id, 10),
		Reason: "High value",
	}

	// First call inserts a new skill
	err = applyPromote(db, skillsDir, proposal)
	g.Expect(err).ToNot(HaveOccurred())

	// Second call finds the existing skill and takes the update path
	err = applyPromote(db, skillsDir, proposal)
	g.Expect(err).ToNot(HaveOccurred())
}

// ─── applyPrune ───────────────────────────────────────────────────────────────

func TestApplyPrune_EmbeddingNotFound(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Valid integer target but no such row → QueryRow.Scan returns sql.ErrNoRows.
	err = applyPrune(db, MaintenanceProposal{Target: "999", Reason: "test"})
	g.Expect(err).To(HaveOccurred())

	if err == nil {
		t.Fatal("expected error but got nil")
	}

	g.Expect(err.Error()).To(ContainSubstring("failed to get embedding_id"))
}

func TestApplyPrune_InvalidTarget(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	proposal := MaintenanceProposal{
		Tier:   "embeddings",
		Action: "prune",
		Target: "not-a-number",
		Reason: "test",
	}

	err = applyPrune(db, proposal)
	g.Expect(err).To(HaveOccurred())
}

func TestApplyPrune_ValidEmbedding(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Insert an embedding row.
	res, err := db.Exec(
		"INSERT INTO embeddings (content, source, memory_type) VALUES (?, ?, ?)",
		"test content", "memory", "fact",
	)
	g.Expect(err).ToNot(HaveOccurred())

	if err != nil {
		t.Fatalf("db.Exec: %v", err)
	}

	id, err := res.LastInsertId()
	g.Expect(err).ToNot(HaveOccurred())

	target := strconv.FormatInt(id, 10)

	err = applyPrune(db, MaintenanceProposal{Target: target, Reason: "test prune"})
	g.Expect(err).ToNot(HaveOccurred())

	// Row should be deleted.
	var count int

	err = db.QueryRow("SELECT COUNT(*) FROM embeddings WHERE id = ?", id).Scan(&count)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(count).To(Equal(0))
}

// ─── applySplit ───────────────────────────────────────────────────────────────

func TestApplySplit_NotImplemented(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	proposal := MaintenanceProposal{
		Tier:   "embeddings",
		Action: "split",
		Target: "1",
		Reason: "Multi-topic content",
	}

	err = applySplit(db, proposal)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("not yet implemented"))
}

// ─── extractRationaleFromEnriched ────────────────────────────────────────────

func TestExtractRationaleFromEnriched_BecauseSeparator(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := extractRationaleFromEnriched("Use mocks because they isolate dependencies")
	g.Expect(result).To(Equal("they isolate dependencies"))
}

func TestExtractRationaleFromEnriched_DashSeparator(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := extractRationaleFromEnriched("Always use TDD - ensures code correctness")
	g.Expect(result).To(Equal("ensures code correctness"))
}

func TestExtractRationaleFromEnriched_NoSeparator(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := extractRationaleFromEnriched("plain rationale text without separator")
	g.Expect(result).To(Equal("plain rationale text without separator"))
}

// ─── scanRedundantEmbeddings ─────────────────────────────────────────────────

func TestScanRedundantEmbeddings_EmptyDatabase(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	proposals, err := scanRedundantEmbeddings(db)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(proposals).To(BeEmpty())
}

func TestScanRedundantEmbeddings_IdenticalEmbeddings(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Create identical embeddings (similarity = 1.0 > 0.92 threshold)
	embedding := make([]float32, 384)
	embedding[0] = 1.0

	blob, err := sqlite_vec.SerializeFloat32(embedding)
	g.Expect(err).ToNot(HaveOccurred())

	// Insert first vec row
	vecRes1, err := db.Exec("INSERT INTO vec_embeddings(embedding) VALUES(?)", blob)
	g.Expect(err).ToNot(HaveOccurred())

	if vecRes1 == nil {
		t.Fatal("db.Exec returned nil for first vec insert")
	}

	embID1, _ := vecRes1.LastInsertId()

	// Insert second vec row (identical)
	vecRes2, err := db.Exec("INSERT INTO vec_embeddings(embedding) VALUES(?)", blob)
	g.Expect(err).ToNot(HaveOccurred())

	if vecRes2 == nil {
		t.Fatal("db.Exec returned nil for second vec insert")
	}

	embID2, _ := vecRes2.LastInsertId()

	// Insert two embeddings rows
	res1, err := db.Exec(`INSERT INTO embeddings(content, source, confidence, retrieval_count, embedding_id) VALUES('first duplicate', 'test', 0.8, 5, ?)`, embID1)
	g.Expect(err).ToNot(HaveOccurred())

	if res1 == nil {
		t.Fatal("db.Exec returned nil for first embedding")
	}

	res2, err := db.Exec(`INSERT INTO embeddings(content, source, confidence, retrieval_count, embedding_id) VALUES('second duplicate', 'test', 0.7, 3, ?)`, embID2)
	g.Expect(err).ToNot(HaveOccurred())

	if res2 == nil {
		t.Fatal("db.Exec returned nil for second embedding")
	}

	proposals, err := scanRedundantEmbeddings(db)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(proposals).ToNot(BeEmpty())

	if len(proposals) < 1 {
		t.Fatal("expected at least one redundant proposal")
	}

	g.Expect(proposals[0].Action).To(Equal("consolidate"))
}
