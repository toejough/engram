package memory

import (
	"database/sql"
	"errors"
	"strings"
	"testing"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	. "github.com/onsi/gomega"
)

func TestApplyReembedAction_DeleteVecFails(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db := leechTestDB(t)
	memID, _ := insertMemoryWithEmbedding(t, db, "original content")

	// Drop the vec_embeddings table so the DELETE inside the transaction fails.
	_, err := db.Exec("DROP TABLE vec_embeddings")
	g.Expect(err).ToNot(HaveOccurred())

	diagnosis := LeechDiagnosis{
		MemoryID:         memID,
		ProposedAction:   "rewrite",
		SuggestedContent: "new content",
		Embedder: func(_ string) ([]float32, error) {
			newEmb := make([]float32, 384)
			return newEmb, nil
		},
	}

	err = ApplyLeechAction(db, diagnosis, RealFS{})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("delete old vec")))
}

func TestApplyReembedAction_EmbedderError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db := leechTestDB(t)
	memID, _ := insertMemoryWithEmbedding(t, db, "original content")

	diagnosis := LeechDiagnosis{
		MemoryID:         memID,
		ProposedAction:   "rewrite",
		SuggestedContent: "new content",
		Embedder: func(_ string) ([]float32, error) {
			return nil, errors.New("embedder unavailable")
		},
	}

	err := ApplyLeechAction(db, diagnosis, RealFS{})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("generate embedding")))
}

func TestApplyReembedAction_EmptySuggestedContent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db := leechTestDB(t)

	diagnosis := LeechDiagnosis{
		MemoryID:         1,
		ProposedAction:   "rewrite",
		SuggestedContent: "",
	}

	err := ApplyLeechAction(db, diagnosis, RealFS{})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("requires SuggestedContent")))
}

func TestApplyReembedAction_MemoryNotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db := leechTestDB(t)

	diagnosis := LeechDiagnosis{
		MemoryID:         99999,
		ProposedAction:   "rewrite",
		SuggestedContent: "improved content",
	}

	err := ApplyLeechAction(db, diagnosis, RealFS{})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("get embedding_id")))
}

// TestApplyReembedAction_NilEmbedderEntersONNXPath verifies that when Embedder is nil,
// applyReembedAction enters the ONNX initialization path.
// Covers the "embed == nil" block (lines 364-381) including homeDir, modelDir, and initONNX.
// NOTE: Not parallel — uses t.Setenv("HOME", ...) to redirect ONNX init to a temp dir.
func TestApplyReembedAction_NilEmbedderEntersONNXPath(t *testing.T) {
	g := NewWithT(t)

	db := leechTestDB(t)
	memID, _ := insertMemoryWithEmbedding(t, db, "original content")

	// Redirect HOME to a temp dir so ONNX runtime download fails at MkdirAll
	// (the temp dir exists but /dev/null/... path would be ENOTDIR; use a read-only-ish approach).
	// Actually use t.TempDir() as HOME so the path is valid but the model file won't exist.
	// Either initializeONNXRuntime fails (no network) or generateEmbeddingONNX fails (no model).
	t.Setenv("HOME", t.TempDir())

	// Nil Embedder → enters embed == nil block → covers homeDir, modelDir, ONNX init path.
	diagnosis := LeechDiagnosis{
		MemoryID:         memID,
		ProposedAction:   "rewrite",
		SuggestedContent: "improved content",
		// Embedder intentionally nil → triggers ONNX path
	}

	err := ApplyLeechAction(db, diagnosis, RealFS{})
	g.Expect(err).To(HaveOccurred())
}

// TestApplyReembedAction_NoEmbeddingRow verifies that when the memory has no embedding row
// (embedding_id is NULL), applyReembedAction returns an error.
// Covers the "!oldEmbeddingID.Valid || oldEmbeddingID.Int64 == 0" branch (line 358).
func TestApplyReembedAction_NoEmbeddingRow(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db := leechTestDB(t)

	// Insert memory without embedding_id (embedding_id = NULL).
	res, err := db.Exec("INSERT INTO embeddings (content, source) VALUES (?, ?)", "content without embedding", "test")
	g.Expect(err).ToNot(HaveOccurred())

	var memID int64

	if res != nil {
		memID, _ = res.LastInsertId()
	}

	diagnosis := LeechDiagnosis{
		MemoryID:         memID,
		ProposedAction:   "rewrite",
		SuggestedContent: "improved content without embedding",
	}

	err = ApplyLeechAction(db, diagnosis, RealFS{})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("has no embedding row")))
}

func TestApplyReembedAction_Success(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db := leechTestDB(t)
	memID, _ := insertMemoryWithEmbedding(t, db, "original content")

	diagnosis := LeechDiagnosis{
		MemoryID:         memID,
		ProposedAction:   "rewrite",
		SuggestedContent: "improved and rewritten content",
		Embedder: func(_ string) ([]float32, error) {
			newEmb := make([]float32, 384)
			newEmb[0] = 1.0

			return newEmb, nil
		},
	}

	err := ApplyLeechAction(db, diagnosis, RealFS{})
	g.Expect(err).ToNot(HaveOccurred())

	var updatedContent string

	err = db.QueryRow("SELECT content FROM embeddings WHERE id = ?", memID).Scan(&updatedContent)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(updatedContent).To(Equal("improved and rewritten content"))
}

func TestApplyReembedAction_UpdateFails(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db := leechTestDB(t)
	memID, _ := insertMemoryWithEmbedding(t, db, "original content")

	// Add a BEFORE UPDATE trigger that always rejects updates on embeddings.
	_, err := db.Exec(`CREATE TRIGGER reject_embed_update BEFORE UPDATE ON embeddings BEGIN SELECT RAISE(FAIL, 'update rejected'); END`)
	g.Expect(err).ToNot(HaveOccurred())

	diagnosis := LeechDiagnosis{
		MemoryID:         memID,
		ProposedAction:   "rewrite",
		SuggestedContent: "new content",
		Embedder: func(_ string) ([]float32, error) {
			newEmb := make([]float32, 384)
			return newEmb, nil
		},
	}

	err = ApplyLeechAction(db, diagnosis, RealFS{})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("update content")))
}

func TestApplyReembedAction_WrongDimensionEmbedding(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db := leechTestDB(t)
	memID, _ := insertMemoryWithEmbedding(t, db, "original content")

	// Return only 3 floats instead of 384 — vec0 INSERT will fail with dimension mismatch.
	diagnosis := LeechDiagnosis{
		MemoryID:         memID,
		ProposedAction:   "narrow_scope",
		SuggestedContent: "narrower content",
		Embedder: func(_ string) ([]float32, error) {
			return []float32{1.0, 2.0, 3.0}, nil
		},
	}

	err := ApplyLeechAction(db, diagnosis, RealFS{})
	g.Expect(err).To(HaveOccurred())
}

// ─── FormatLeechDiagnosis tests ───────────────────────────────────────────────

func TestFormatLeechDiagnosis_BasicFields(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	diag := &LeechDiagnosis{
		MemoryID:      42,
		Content:       "always run tests before committing",
		DiagnosisType: "insufficient_data",
		Signal:        "No surfacing events recorded",
	}

	result := FormatLeechDiagnosis(diag)

	g.Expect(result).To(ContainSubstring("Memory ID: 42"))
	g.Expect(result).To(ContainSubstring("Content: always run tests before committing"))
	g.Expect(result).To(ContainSubstring("Diagnosis: insufficient_data"))
	g.Expect(result).To(ContainSubstring("Signal: No surfacing events recorded"))
}

func TestFormatLeechDiagnosis_NoProposedAction(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	diag := &LeechDiagnosis{
		MemoryID:      5,
		Content:       "some content",
		DiagnosisType: "insufficient_data",
		Signal:        "Not enough history",
	}

	result := FormatLeechDiagnosis(diag)

	g.Expect(result).ToNot(ContainSubstring("Proposed Action"))
}

func TestFormatLeechDiagnosis_NoRecommendation(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	diag := &LeechDiagnosis{
		MemoryID:      7,
		Content:       "some content",
		DiagnosisType: "content_quality",
		Signal:        "low faithfulness",
	}

	result := FormatLeechDiagnosis(diag)

	g.Expect(result).ToNot(ContainSubstring("Recommendation"))
}

func TestFormatLeechDiagnosis_NoSuggestedContent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	diag := &LeechDiagnosis{
		MemoryID:       6,
		Content:        "some content",
		DiagnosisType:  "retrieval_mismatch",
		Signal:         "haiku rated irrelevant",
		ProposedAction: "narrow_scope",
	}

	result := FormatLeechDiagnosis(diag)

	g.Expect(result).ToNot(ContainSubstring("Suggested Content"))
}

func TestFormatLeechDiagnosis_ReturnsNewlineSeparatedFields(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	diag := &LeechDiagnosis{
		MemoryID:      10,
		Content:       "test",
		DiagnosisType: "insufficient_data",
		Signal:        "no data",
	}

	result := FormatLeechDiagnosis(diag)

	lines := strings.Split(result, "\n")
	g.Expect(len(lines)).To(BeNumerically(">=", 4), "result should have at least 4 newline-separated lines")
}

func TestFormatLeechDiagnosis_WithProposedAction(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	diag := &LeechDiagnosis{
		MemoryID:       1,
		Content:        "use go test -tags sqlite_fts5",
		DiagnosisType:  "wrong_tier",
		Signal:         "Surfaced 3 times with user corrections",
		ProposedAction: "promote_to_claude_md",
	}

	result := FormatLeechDiagnosis(diag)

	g.Expect(result).To(ContainSubstring("Proposed Action: promote_to_claude_md"))
}

func TestFormatLeechDiagnosis_WithRecommendation(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	diag := &LeechDiagnosis{
		MemoryID:       3,
		Content:        "some content",
		DiagnosisType:  "enforcement_gap",
		Signal:         "agent used it but user corrected",
		ProposedAction: "convert_to_hook",
		Recommendation: &Recommendation{
			Category:    "hook-conversion",
			Description: "Convert to a PreToolUse hook",
		},
	}

	result := FormatLeechDiagnosis(diag)

	g.Expect(result).To(ContainSubstring("Recommendation [hook-conversion]: Convert to a PreToolUse hook"))
}

func TestFormatLeechDiagnosis_WithSuggestedContent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	diag := &LeechDiagnosis{
		MemoryID:         2,
		Content:          "original content",
		DiagnosisType:    "content_quality",
		Signal:           "low faithfulness",
		ProposedAction:   "rewrite",
		SuggestedContent: "rewritten and cleaner content",
	}

	result := FormatLeechDiagnosis(diag)

	g.Expect(result).To(ContainSubstring("Suggested Content:"))
	g.Expect(result).To(ContainSubstring("rewritten and cleaner content"))
}

// ─── applyReembedAction tests (via ApplyLeechAction) ──────────────────────────

// insertMemoryWithEmbedding inserts a memory row with a real vec_embeddings row
// and returns (memID, embID).
func insertMemoryWithEmbedding(t *testing.T, db *sql.DB, content string) (memID, embID int64) {
	t.Helper()

	emb := make([]float32, 384)

	blob, err := sqlite_vec.SerializeFloat32(emb)
	if err != nil {
		t.Fatalf("SerializeFloat32: %v", err)
	}

	res, err := db.Exec("INSERT INTO vec_embeddings(embedding) VALUES (?)", blob)
	if err != nil {
		t.Fatalf("INSERT vec_embeddings: %v", err)
	}

	embID, _ = res.LastInsertId()

	res, err = db.Exec(
		"INSERT INTO embeddings (content, source, embedding_id) VALUES (?, ?, ?)",
		content, "test", embID,
	)
	if err != nil {
		t.Fatalf("INSERT embeddings: %v", err)
	}

	memID, _ = res.LastInsertId()

	return memID, embID
}
