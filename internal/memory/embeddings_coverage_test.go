package memory

import (
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	. "github.com/onsi/gomega"
)

// ─── boostExistingConfidence ──────────────────────────────────────────────────

func TestBoostExistingConfidence_CapsAtOne(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	res, err := db.Exec(`INSERT INTO embeddings(content, source, confidence) VALUES('cap test', 'test', 0.98)`)
	g.Expect(err).ToNot(HaveOccurred())

	if res == nil {
		t.Fatal("db.Exec returned nil result")
	}

	id, _ := res.LastInsertId()

	err = boostExistingConfidence(db, id)
	g.Expect(err).ToNot(HaveOccurred())

	var conf float64

	err = db.QueryRow("SELECT confidence FROM embeddings WHERE id = ?", id).Scan(&conf)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(conf).To(BeNumerically("<=", 1.0))
}

func TestBoostExistingConfidence_Increases(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	res, err := db.Exec(`INSERT INTO embeddings(content, source, confidence) VALUES('boost test', 'test', 0.5)`)
	g.Expect(err).ToNot(HaveOccurred())

	if res == nil {
		t.Fatal("db.Exec returned nil result")
	}

	id, _ := res.LastInsertId()

	err = boostExistingConfidence(db, id)
	g.Expect(err).ToNot(HaveOccurred())

	var conf float64

	err = db.QueryRow("SELECT confidence FROM embeddings WHERE id = ?", id).Scan(&conf)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(conf).To(BeNumerically("~", 0.55, 0.01))
}

// ─── deleteFTS5 ───────────────────────────────────────────────────────────────

func TestDeleteFTS5_NoFTS5Table(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Drop the FTS5 table so hasFTS5 returns false
	_, _ = db.Exec("DROP TABLE IF EXISTS embeddings_fts")

	// Should be a no-op and not panic
	deleteFTS5(db, 999)

	// Verify FTS5 table is gone
	g.Expect(hasFTS5(db)).To(BeFalse())
}

func TestDeleteFTS5_WithFTS5Table(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	if !hasFTS5(db) {
		t.Skip("FTS5 not available")
	}

	_, err = db.Exec(`INSERT INTO embeddings_fts(rowid, content) VALUES(42, 'delete me')`)
	g.Expect(err).ToNot(HaveOccurred())

	deleteFTS5(db, 42)

	var count int

	err = db.QueryRow("SELECT COUNT(*) FROM embeddings_fts WHERE rowid = 42").Scan(&count)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(count).To(Equal(0))
}

// ─── downloadModel ────────────────────────────────────────────────────────────

func TestDownloadModel_FileAlreadyExists(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	modelPath := filepath.Join(tempDir, "model.onnx")

	err := os.WriteFile(modelPath, []byte("fake model data"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	// Fast path: file exists → should return nil without network access
	err = downloadModel(modelPath, http.DefaultClient)
	g.Expect(err).ToNot(HaveOccurred())
}

// ─── downloadONNXRuntime ─────────────────────────────────────────────────────

func TestDownloadONNXRuntime_FileAlreadyExists(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()

	var libName string

	switch runtime.GOOS {
	case "darwin":
		libName = "libonnxruntime.dylib"
	case "linux":
		libName = "libonnxruntime.so"
	case "windows":
		libName = "onnxruntime.dll"
	default:
		t.Skip("unsupported platform for ONNX runtime test")
	}

	libPath := filepath.Join(tempDir, libName)

	err := os.WriteFile(libPath, []byte("fake onnx runtime"), 0755)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := downloadONNXRuntime(tempDir, http.DefaultClient)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(Equal(libPath))
}

// ─── ensureCorrectModel ───────────────────────────────────────────────────────

func TestEnsureCorrectModel_ModelDoesNotExist(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	needsDownload, err := ensureCorrectModel(db, "/nonexistent/path/model.onnx")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(needsDownload).To(BeTrue())
}

func TestEnsureCorrectModel_ModelExistsEmptyMetadata(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	modelPath := filepath.Join(tempDir, "model.onnx")

	err := os.WriteFile(modelPath, []byte("fake model"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// No model_url metadata → should set it and return (false, nil)
	needsDownload, err := ensureCorrectModel(db, modelPath)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(needsDownload).To(BeFalse())
}

func TestEnsureCorrectModel_ModelExistsMatchingURL(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	modelPath := filepath.Join(tempDir, "model.onnx")

	err := os.WriteFile(modelPath, []byte("fake model"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Set matching URL
	err = setMetadata(db, "model_url", e5SmallModelURL)
	g.Expect(err).ToNot(HaveOccurred())

	needsDownload, err := ensureCorrectModel(db, modelPath)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(needsDownload).To(BeFalse())
}

func TestEnsureCorrectModel_StaleModel(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	modelPath := filepath.Join(tempDir, "model.onnx")

	err := os.WriteFile(modelPath, []byte("old model"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Set a different URL to simulate stale model
	err = setMetadata(db, "model_url", "https://old-url.example.com/model.onnx")
	g.Expect(err).ToNot(HaveOccurred())

	needsDownload, err := ensureCorrectModel(db, modelPath)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(needsDownload).To(BeTrue())

	// Model file should have been deleted
	_, statErr := os.Stat(modelPath)
	g.Expect(os.IsNotExist(statErr)).To(BeTrue())
}

// ─── updateEmbeddingContent ───────────────────────────────────────────────────

// ─── ensureCorrectModel (stale model cannot remove) ──────────────────────────

func TestEnsureCorrectModel_StaleModelCannotRemove(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()

	// Create a directory at the model path so os.Remove fails (can't remove non-empty dir
	// is more reliable, but even removing an empty dir works with os.Remove on some platforms;
	// use a non-empty directory to force the failure)
	modelPath := filepath.Join(tempDir, "model.onnx")

	err := os.MkdirAll(modelPath, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Create a file inside so the dir is non-empty (makes Remove fail on all platforms)
	err = os.WriteFile(filepath.Join(modelPath, "child.txt"), []byte("data"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Set a stale URL to trigger the removal path
	err = setMetadata(db, "model_url", "https://old-url.example.com/stale-model.onnx")
	g.Expect(err).ToNot(HaveOccurred())

	// os.Remove on a non-empty directory returns an error
	_, err = ensureCorrectModel(db, modelPath)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("failed to delete stale model"))
}

// ─── GetMetadataForTest ───────────────────────────────────────────────────────

func TestGetMetadataForTest_MissingKey(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()

	val, err := GetMetadataForTest(tempDir, "nonexistent_key")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(val).To(Equal(""))
}

func TestGetMetadataForTest_ReturnsValue(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()

	err := SetMetadataForTest(tempDir, "test_key", "hello_world")
	g.Expect(err).ToNot(HaveOccurred())

	val, err := GetMetadataForTest(tempDir, "test_key")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(val).To(Equal("hello_world"))
}

// ─── GetSessionInitCount ─────────────────────────────────────────────────────

func TestGetSessionInitCount_ReturnsNonNegative(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	count := GetSessionInitCount()
	g.Expect(count).To(BeNumerically(">=", 0))
}

// ─── insertFTS5 ───────────────────────────────────────────────────────────────

func TestInsertFTS5_NoFTS5Table(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Drop FTS5 table so hasFTS5 returns false
	_, _ = db.Exec("DROP TABLE IF EXISTS embeddings_fts")

	// Should be a no-op and not panic
	insertFTS5(db, 1, "content that should not be inserted")

	g.Expect(hasFTS5(db)).To(BeFalse())
}

func TestInsertFTS5_WithFTS5Table(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	if !hasFTS5(db) {
		t.Skip("FTS5 not available")
	}

	insertFTS5(db, 99, "fts5 inserted content")

	var count int

	err = db.QueryRow("SELECT COUNT(*) FROM embeddings_fts WHERE rowid = 99").Scan(&count)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(count).To(Equal(1))
}

// TestIsEmbeddingsEmpty_ClosedDB verifies isEmbeddingsEmpty returns error when DB is closed.
func TestIsEmbeddingsEmpty_ClosedDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	err = db.Close()
	g.Expect(err).ToNot(HaveOccurred())

	_, err = isEmbeddingsEmpty(db)
	g.Expect(err).To(HaveOccurred())
}

// ─── isEmbeddingsEmpty ────────────────────────────────────────────────────────

// TestIsEmbeddingsEmpty_EmptyDB verifies isEmbeddingsEmpty returns true for empty DB.
func TestIsEmbeddingsEmpty_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	empty, err := isEmbeddingsEmpty(db)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(empty).To(BeTrue())
}

// TestIsEmbeddingsEmpty_NonEmpty verifies isEmbeddingsEmpty returns false when rows exist.
func TestIsEmbeddingsEmpty_NonEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	_, err = db.Exec("INSERT INTO embeddings (content, source) VALUES ('test', 'test')")
	g.Expect(err).ToNot(HaveOccurred())

	empty, err := isEmbeddingsEmpty(db)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(empty).To(BeFalse())
}

// ─── migrateModelVersion ─────────────────────────────────────────────────────

func TestMigrateModelVersion_AlreadyMigrated(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	err = setMetadata(db, "model_version_migrated", "e5-small-v2")
	g.Expect(err).ToNot(HaveOccurred())

	// Should return nil immediately (already migrated)
	err = migrateModelVersion(db, &testEmbedder{})
	g.Expect(err).ToNot(HaveOccurred())
}

func TestMigrateModelVersion_EmptyDatabase(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Empty DB: no entries need migration, so loop is skipped and metadata is set
	err = migrateModelVersion(db, &testEmbedder{})
	g.Expect(err).ToNot(HaveOccurred())

	migrated, err := getMetadata(db, "model_version_migrated")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(migrated).To(Equal("e5-small-v2"))
}

// ─── rawResultsToExistingMemories ────────────────────────────────────────────

func TestRawResultsToExistingMemories_Empty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := rawResultsToExistingMemories(nil)
	g.Expect(result).To(BeEmpty())
}

func TestRawResultsToExistingMemories_WithData(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	input := []rawSimilarResult{
		{id: 1, content: "first content", similarity: 0.95},
		{id: 2, content: "second content", similarity: 0.82},
	}

	result := rawResultsToExistingMemories(input)
	g.Expect(result).To(HaveLen(2))
	g.Expect(result[0].ID).To(Equal(int64(1)))
	g.Expect(result[0].Content).To(Equal("first content"))
	g.Expect(result[0].Similarity).To(BeNumerically("~", 0.95, 0.001))
	g.Expect(result[1].ID).To(Equal(int64(2)))
}

// ─── searchBM25 ───────────────────────────────────────────────────────────────

func TestSearchBM25_EmptyQuery(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	results, err := searchBM25(db, "", 10)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).To(BeEmpty())
}

func TestSearchBM25_OnlyOperators(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Query consisting only of FTS5 operators → sanitized becomes empty string
	results, err := searchBM25(db, "AND OR NOT NEAR", 10)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).To(BeEmpty())
}

func TestSearchBM25_WithData(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	if !hasFTS5(db) {
		t.Skip("FTS5 not available")
	}

	res, err := db.Exec(`INSERT INTO embeddings(content, source) VALUES('golang unit testing patterns', 'test')`)
	g.Expect(err).ToNot(HaveOccurred())

	if res == nil {
		t.Fatal("db.Exec returned nil for embeddings insert")
	}

	id, _ := res.LastInsertId()

	_, err = db.Exec(`INSERT INTO embeddings_fts(rowid, content) VALUES(?, 'golang unit testing patterns')`, id)
	g.Expect(err).ToNot(HaveOccurred())

	results, err := searchBM25(db, "golang testing", 10)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).ToNot(BeEmpty())
	g.Expect(results).ToNot(BeNil())

	if len(results) == 0 {
		t.Fatal("results must be non-empty")
	}

	g.Expect(results[0].MatchType).To(Equal("bm25"))
}

// ─── searchSimilar ────────────────────────────────────────────────────────────

func TestSearchSimilar_EmptyDatabase(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	queryEmbedding := make([]float32, 384)

	results, err := searchSimilar(db, queryEmbedding, 10)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).To(BeEmpty())
}

// ─── searchSimilar (score clamping) ──────────────────────────────────────────

func TestSearchSimilar_HighConfidenceScore(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// confidence > 1.0 means score = (1 - 0) * 1.5 = 1.5 → clamped to 1.0
	embedding := make([]float32, 384)
	embedding[0] = 1.0

	blob, err := sqlite_vec.SerializeFloat32(embedding)
	g.Expect(err).ToNot(HaveOccurred())

	vecRes, err := db.Exec("INSERT INTO vec_embeddings(embedding) VALUES(?)", blob)
	g.Expect(err).ToNot(HaveOccurred())

	if vecRes == nil {
		t.Fatal("db.Exec returned nil for vec insert")
	}

	embID, _ := vecRes.LastInsertId()

	_, err = db.Exec(`INSERT INTO embeddings(content, source, confidence, embedding_id)
		VALUES('high confidence entry', 'test', 1.5, ?)`, embID)
	g.Expect(err).ToNot(HaveOccurred())

	// Query with identical vector → cos_dist=0, score = 1.5 → clamped to 1.0
	results, err := searchSimilar(db, embedding, 10)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).ToNot(BeEmpty())
	g.Expect(results).ToNot(BeNil())

	if len(results) == 0 {
		t.Fatal("results must be non-empty")
	}

	g.Expect(results[0].Score).To(BeNumerically("<=", 1.0))
}

func TestSearchSimilar_NegativeScore(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Store a vector pointing in opposite direction to query
	storedEmbedding := make([]float32, 384)
	storedEmbedding[0] = -1.0 // opposite of query

	blob, err := sqlite_vec.SerializeFloat32(storedEmbedding)
	g.Expect(err).ToNot(HaveOccurred())

	vecRes, err := db.Exec("INSERT INTO vec_embeddings(embedding) VALUES(?)", blob)
	g.Expect(err).ToNot(HaveOccurred())

	if vecRes == nil {
		t.Fatal("db.Exec returned nil for vec insert")
	}

	embID, _ := vecRes.LastInsertId()

	_, err = db.Exec(`INSERT INTO embeddings(content, source, confidence, embedding_id)
		VALUES('opposite direction entry', 'test', 1.0, ?)`, embID)
	g.Expect(err).ToNot(HaveOccurred())

	// Query with +1 vector vs stored -1 vector → cos_dist≈2, score = (1-2)*1.0 = -1 → clamped to 0.0
	queryEmbedding := make([]float32, 384)
	queryEmbedding[0] = 1.0

	results, err := searchSimilar(db, queryEmbedding, 10)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).ToNot(BeEmpty())
	g.Expect(results).ToNot(BeNil())

	if len(results) == 0 {
		t.Fatal("results must be non-empty")
	}

	g.Expect(results[0].Score).To(BeNumerically(">=", 0.0))
}

func TestSearchSimilar_WithData(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Create a unit-normalized test embedding
	embedding := make([]float32, 384)
	embedding[0] = 1.0

	blob, err := sqlite_vec.SerializeFloat32(embedding)
	g.Expect(err).ToNot(HaveOccurred())

	vecRes, err := db.Exec("INSERT INTO vec_embeddings(embedding) VALUES(?)", blob)
	g.Expect(err).ToNot(HaveOccurred())

	if vecRes == nil {
		t.Fatal("db.Exec returned nil for vec insert")
	}

	embID, _ := vecRes.LastInsertId()

	_, err = db.Exec(`INSERT INTO embeddings(content, source, confidence, embedding_id) VALUES('similar test content', 'test', 1.0, ?)`, embID)
	g.Expect(err).ToNot(HaveOccurred())

	results, err := searchSimilar(db, embedding, 10)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).ToNot(BeEmpty())
	g.Expect(results).ToNot(BeNil())

	if len(results) == 0 {
		t.Fatal("results must be non-empty")
	}

	g.Expect(results[0].Content).To(Equal("similar test content"))
	g.Expect(results[0].MatchType).To(Equal("vector"))
}

// ─── SetMetadataForTest ───────────────────────────────────────────────────────

func TestSetMetadataForTest_Success(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()

	err := SetMetadataForTest(tempDir, "my_key", "my_value")
	g.Expect(err).ToNot(HaveOccurred())

	val, err := GetMetadataForTest(tempDir, "my_key")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(val).To(Equal("my_value"))
}

func TestSetMetadataForTest_Upsert(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()

	err := SetMetadataForTest(tempDir, "upsert_key", "original")
	g.Expect(err).ToNot(HaveOccurred())

	err = SetMetadataForTest(tempDir, "upsert_key", "updated")
	g.Expect(err).ToNot(HaveOccurred())

	val, err := GetMetadataForTest(tempDir, "upsert_key")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(val).To(Equal("updated"))
}

// ─── setMetadata (error path) ─────────────────────────────────────────────────

func TestSetMetadata_ClosedDB_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	_ = db.Close()

	// Should return an error since DB is closed
	err = setMetadata(db, "key", "value")
	g.Expect(err).To(HaveOccurred())
}

// ─── supersedeEmbedding ───────────────────────────────────────────────────────

func TestSupersedeEmbedding_SetsConfidenceToZero(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	res, err := db.Exec(`INSERT INTO embeddings(content, source, confidence) VALUES('supersede me', 'test', 0.8)`)
	g.Expect(err).ToNot(HaveOccurred())

	if res == nil {
		t.Fatal("db.Exec returned nil result")
	}

	id, _ := res.LastInsertId()

	err = supersedeEmbedding(db, id)
	g.Expect(err).ToNot(HaveOccurred())

	var conf float64

	err = db.QueryRow("SELECT confidence FROM embeddings WHERE id = ?", id).Scan(&conf)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(conf).To(BeNumerically("~", 0.0, 0.001))
}

// TestUpdateEmbeddingContent_DeleteVecError verifies the "failed to delete old vec row"
// error path when vec_embeddings table has been dropped (covers line 1547-1548).
func TestUpdateEmbeddingContent_DeleteVecError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Insert an embedding row referencing a (non-existent) vec row
	res, err := db.Exec(`INSERT INTO embeddings(content, source, embedding_id) VALUES('test', 'test', 42)`)
	g.Expect(err).ToNot(HaveOccurred())

	if res == nil {
		t.Fatal("db.Exec returned nil")
	}

	embID, _ := res.LastInsertId()

	// Drop vec_embeddings so the DELETE fails (QueryRow SELECT succeeds, DELETE fails)
	_, err = db.Exec("DROP TABLE IF EXISTS vec_embeddings")
	g.Expect(err).ToNot(HaveOccurred())

	blob := make([]byte, 384*4) // raw bytes for new embedding
	err = updateEmbeddingContent(db, embID, "new content", blob,
		"", "", "", "", "", "")
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("failed to delete old vec row"))
	}
}

// ─── updateEmbeddingContent (ID not found) ───────────────────────────────────

func TestUpdateEmbeddingContent_IDNotFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	newEmbedding := make([]float32, 384)
	newEmbedding[0] = 1.0

	blob, err := sqlite_vec.SerializeFloat32(newEmbedding)
	g.Expect(err).ToNot(HaveOccurred())

	// ID 99999 doesn't exist → QueryRow.Scan returns sql.ErrNoRows
	err = updateEmbeddingContent(db, 99999, "new content", blob,
		"pattern", "testing", "Use TDD", "", "ensures correctness", "enriched")

	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("failed to get old embedding_id"))
	}
}

func TestUpdateEmbeddingContent_Success(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(":memory:")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Create initial embedding blob
	embedding := make([]float32, 384)
	embedding[0] = 1.0

	blob, err := sqlite_vec.SerializeFloat32(embedding)
	g.Expect(err).ToNot(HaveOccurred())

	vecRes, err := db.Exec("INSERT INTO vec_embeddings(embedding) VALUES(?)", blob)
	g.Expect(err).ToNot(HaveOccurred())

	if vecRes == nil {
		t.Fatal("db.Exec returned nil for vec insert")
	}

	embID, _ := vecRes.LastInsertId()

	metaRes, err := db.Exec(`INSERT INTO embeddings(content, source, embedding_id) VALUES('old content', 'test', ?)`, embID)
	g.Expect(err).ToNot(HaveOccurred())

	if metaRes == nil {
		t.Fatal("db.Exec returned nil for meta insert")
	}

	id, _ := metaRes.LastInsertId()

	// New embedding blob
	newEmbedding := make([]float32, 384)
	newEmbedding[1] = 1.0

	newBlob, err := sqlite_vec.SerializeFloat32(newEmbedding)
	g.Expect(err).ToNot(HaveOccurred())

	err = updateEmbeddingContent(db, id, "new content", newBlob,
		"pattern", "testing,go", "Use TDD always", "", "ensures correctness", "enriched for TDD")
	g.Expect(err).ToNot(HaveOccurred())

	var content string

	err = db.QueryRow("SELECT content FROM embeddings WHERE id = ?", id).Scan(&content)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(content).To(Equal("new content"))
}
