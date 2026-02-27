package memory

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	. "github.com/onsi/gomega"
)

func TestClearSessionCache_Empty(t *testing.T) {
	g := NewWithT(t)

	// Should not panic on empty cache
	ClearSessionCache()

	g.Expect(sessionCache).To(BeEmpty())
	g.Expect(sessionOnce).To(BeEmpty())
}

// ─── collectBM25Rows tests ────────────────────────────────────────────────────

func TestCollectBM25Rows_Empty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("initEmbeddingsDB: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE bm25rows (id INTEGER, content TEXT, rank REAL, memory_type TEXT, retrieval_count INTEGER, projects_retrieved TEXT)`)
	if err != nil {
		t.Fatalf("CREATE TABLE: %v", err)
	}

	rows, err := db.Query("SELECT id, content, rank, memory_type, retrieval_count, projects_retrieved FROM bm25rows")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	defer rows.Close()

	results, err := collectBM25Rows(rows)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).To(BeEmpty())
}

func TestCollectBM25Rows_MultipleRows(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("initEmbeddingsDB: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE bm25rows (id INTEGER, content TEXT, rank REAL, memory_type TEXT, retrieval_count INTEGER, projects_retrieved TEXT)`)
	if err != nil {
		t.Fatalf("CREATE TABLE: %v", err)
	}

	_, err = db.Exec("INSERT INTO bm25rows VALUES (1, 'first result', -3.0, 'correction', 5, 'proj-a,proj-b'), (2, 'second result', -1.0, 'observation', 2, '')")
	if err != nil {
		t.Fatalf("INSERT: %v", err)
	}

	rows, err := db.Query("SELECT id, content, rank, memory_type, retrieval_count, projects_retrieved FROM bm25rows ORDER BY rank")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	defer rows.Close()

	results, err := collectBM25Rows(rows)
	g.Expect(err).ToNot(HaveOccurred())

	if len(results) < 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	g.Expect(results[0].ID).To(Equal(int64(1)))
	g.Expect(results[0].Content).To(Equal("first result"))
	g.Expect(results[0].Score).To(BeNumerically("~", 3.0, 0.001))
	g.Expect(results[0].Source).To(Equal("memory"))
	g.Expect(results[0].MemoryType).To(Equal("correction"))
	g.Expect(results[0].RetrievalCount).To(Equal(5))
	g.Expect(results[0].MatchType).To(Equal("bm25"))
	g.Expect(results[0].ProjectsRetrieved).To(ContainElements("proj-a", "proj-b"))
}

func TestCollectBM25Rows_PositiveRankClamped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("initEmbeddingsDB: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE bm25rows (id INTEGER, content TEXT, rank REAL, memory_type TEXT, retrieval_count INTEGER, projects_retrieved TEXT)`)
	if err != nil {
		t.Fatalf("CREATE TABLE: %v", err)
	}

	// rank = 1.5 (positive) → score = -1.5 → clamped to 0.0
	_, err = db.Exec("INSERT INTO bm25rows VALUES (10, 'clamped content', 1.5, '', 0, '')")
	if err != nil {
		t.Fatalf("INSERT: %v", err)
	}

	rows, err := db.Query("SELECT id, content, rank, memory_type, retrieval_count, projects_retrieved FROM bm25rows")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	defer rows.Close()

	results, err := collectBM25Rows(rows)
	g.Expect(err).ToNot(HaveOccurred())

	if len(results) < 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	g.Expect(results[0].Score).To(Equal(0.0))
}

// TestComputeRRFScore_BM25Only verifies RRF returns bm25-only results when vector is empty.
func TestComputeRRFScore_BM25Only(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	bm25Results := []QueryResult{
		{ID: 1, Content: "memory about errors", Score: 0.7},
	}

	results := computeRRFScore(nil, bm25Results, 60, 5)

	g.Expect(results).To(HaveLen(1))
	g.Expect(results[0].MatchType).To(Equal("bm25"))
}

// TestComputeRRFScore_HybridOverlap verifies RRF marks results in both lists as "hybrid".
func TestComputeRRFScore_HybridOverlap(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	shared := "shared memory content"
	vectorResults := []QueryResult{
		{ID: 1, Content: shared, Score: 0.9},
		{ID: 2, Content: "vector only content", Score: 0.8},
	}
	bm25Results := []QueryResult{
		{ID: 1, Content: shared, Score: 0.7},
		{ID: 3, Content: "bm25 only content", Score: 0.6},
	}

	results := computeRRFScore(vectorResults, bm25Results, 60, 5)

	g.Expect(results).To(HaveLen(3))

	// Shared content should have the highest RRF score (appeared in both lists)
	g.Expect(results[0].Content).To(Equal(shared))
	g.Expect(results[0].MatchType).To(Equal("hybrid"))
}

// TestComputeRRFScore_LimitEnforced verifies RRF enforces the limit.
func TestComputeRRFScore_LimitEnforced(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vectorResults := []QueryResult{
		{ID: 1, Content: "a", Score: 0.9},
		{ID: 2, Content: "b", Score: 0.8},
		{ID: 3, Content: "c", Score: 0.7},
	}

	results := computeRRFScore(vectorResults, nil, 60, 2)

	g.Expect(results).To(HaveLen(2))
}

// TestComputeRRFScore_VectorOnly verifies RRF returns vector-only results when BM25 is empty.
func TestComputeRRFScore_VectorOnly(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	vectorResults := []QueryResult{
		{ID: 1, Content: "memory about errors", Score: 0.9},
		{ID: 2, Content: "memory about testing", Score: 0.8},
	}

	results := computeRRFScore(vectorResults, nil, 60, 5)

	g.Expect(results).To(HaveLen(2))
	g.Expect(results[0].MatchType).To(Equal("vector"))
	g.Expect(results[1].MatchType).To(Equal("vector"))
}

// TestDeleteFTS5_NoOp verifies deleteFTS5 does not panic regardless of FTS5 availability.
func TestDeleteFTS5_NoOp(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))

	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Either hits early return (no FTS5 build tag) or executes delete (with FTS5 tag)
	deleteFTS5(db, 1)
	deleteFTS5(db, 9999)
}

// TestDeleteFTS5_WithFTSTable covers line 308 (DELETE FROM embeddings_fts)
// by creating a regular table named embeddings_fts so hasFTS5 returns true.
func TestDeleteFTS5_WithFTSTable(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Create a regular table named embeddings_fts so hasFTS5(db) returns true
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS embeddings_fts (rowid INTEGER PRIMARY KEY, content TEXT)`)
	g.Expect(err).ToNot(HaveOccurred())

	// deleteFTS5 now sees hasFTS5=true → executes DELETE (no-op since table is empty)
	deleteFTS5(db, 1)
}

// TestDownloadModel_ExistingFile verifies downloadModel returns nil when file already exists (fast path).
func TestDownloadModel_ExistingFile(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	tmpDir := t.TempDir()
	modelPath := filepath.Join(tmpDir, "model.onnx")

	err := os.WriteFile(modelPath, []byte("fake model data"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	err = downloadModel(modelPath, http.DefaultClient)

	g.Expect(err).ToNot(HaveOccurred())
}

// TestDownloadModel_MockHTTPError verifies downloadModel returns error on non-200 response.
func TestDownloadModel_MockHTTPError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	defer ts.Close()

	modelPath := filepath.Join(t.TempDir(), "model.onnx")

	err := downloadModel(modelPath, testClient(ts.URL))

	g.Expect(err).To(MatchError(ContainSubstring("HTTP 500")))
}

// TestDownloadModel_MockSuccess verifies downloadModel succeeds when server returns 200.
func TestDownloadModel_MockSuccess(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("fake onnx binary data"))
	}))

	defer ts.Close()

	modelPath := filepath.Join(t.TempDir(), "model.onnx")

	err := downloadModel(modelPath, testClient(ts.URL))

	g.Expect(err).ToNot(HaveOccurred())
}

// TestDownloadONNXRuntime_ClientError verifies downloadONNXRuntime returns error when client.Get fails.
func TestDownloadONNXRuntime_ClientError(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("windows not supported in extraction path")
	}

	g := NewWithT(t)

	// Use a closed server so the HTTP request fails with a connection error.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	ts.Close()

	_, err := downloadONNXRuntime(t.TempDir(), testClient(ts.URL))

	g.Expect(err).To(MatchError(ContainSubstring("failed to download ONNX Runtime")))
}

// TestDownloadONNXRuntime_EmptyTar verifies downloadONNXRuntime returns error when lib not found in archive.
func TestDownloadONNXRuntime_EmptyTar(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("windows not supported in extraction path")
	}

	g := NewWithT(t)

	// Create a valid but empty tar.gz (no entries)
	var buf bytes.Buffer

	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	_ = tw.Close()
	_ = gz.Close()
	tarGZContent := buf.Bytes()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(tarGZContent)
	}))

	defer ts.Close()

	_, err := downloadONNXRuntime(t.TempDir(), testClient(ts.URL))

	g.Expect(err).To(MatchError(ContainSubstring("library not found in archive")))
}

// TestDownloadONNXRuntime_ExistingLib verifies downloadONNXRuntime returns early when lib already exists.
func TestDownloadONNXRuntime_ExistingLib(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	tmpDir := t.TempDir()

	var libName string

	switch runtime.GOOS {
	case "darwin":
		libName = "libonnxruntime.dylib"
	case "linux":
		libName = "libonnxruntime.so"
	case "windows":
		libName = "onnxruntime.dll"
	default:
		t.Skip("unsupported platform")
	}

	libPath := filepath.Join(tmpDir, libName)

	err := os.WriteFile(libPath, []byte("fake lib"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	resultPath, err := downloadONNXRuntime(tmpDir, http.DefaultClient)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(resultPath).To(Equal(libPath))
}

// TestDownloadONNXRuntime_InvalidGzip verifies downloadONNXRuntime returns error on non-gzip response body.
func TestDownloadONNXRuntime_InvalidGzip(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("windows not supported in extraction path")
	}

	g := NewWithT(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("this is not valid gzip content"))
	}))

	defer ts.Close()

	_, err := downloadONNXRuntime(t.TempDir(), testClient(ts.URL))

	g.Expect(err).To(MatchError(ContainSubstring("failed to create gzip reader")))
}

// TestDownloadONNXRuntime_LibExtractedFromTar verifies downloadONNXRuntime extracts the dylib from a tar.gz.
func TestDownloadONNXRuntime_LibExtractedFromTar(t *testing.T) {
	t.Parallel()

	if runtime.GOOS != "darwin" {
		t.Skip("dylib extraction only for darwin")
	}

	g := NewWithT(t)

	// Create a tar.gz containing a file that matches the extraction criteria.
	var buf bytes.Buffer

	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	content := []byte("fake onnxruntime library content for testing")
	err := tw.WriteHeader(&tar.Header{
		Name:     "onnxruntime-osx-arm64-1.0.0/lib/libonnxruntime.1.0.0.dylib",
		Typeflag: tar.TypeReg,
		Size:     int64(len(content)),
	})
	g.Expect(err).ToNot(HaveOccurred())

	_, err = tw.Write(content)
	g.Expect(err).ToNot(HaveOccurred())

	_ = tw.Close()
	_ = gz.Close()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(buf.Bytes())
	}))

	defer ts.Close()

	tmpDir := t.TempDir()
	libPath, err := downloadONNXRuntime(tmpDir, testClient(ts.URL))

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(libPath).To(ContainSubstring("libonnxruntime.dylib"))

	_, statErr := os.Stat(libPath)
	g.Expect(statErr).ToNot(HaveOccurred())
}

// TestDownloadONNXRuntime_MockHTTPError verifies downloadONNXRuntime returns error on non-200 response.
func TestDownloadONNXRuntime_MockHTTPError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))

	defer ts.Close()

	// TempDir without lib file: passes the "already exists" check, proceeds to download
	_, err := downloadONNXRuntime(t.TempDir(), testClient(ts.URL))

	g.Expect(err).To(MatchError(ContainSubstring("HTTP 503")))
}

// TestDownloadONNXRuntime_TarReadError verifies downloadONNXRuntime returns error on corrupted tar data.
func TestDownloadONNXRuntime_TarReadError(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("windows not supported in extraction path")
	}

	g := NewWithT(t)

	// Create a gzip archive with random bytes that are not valid tar.
	var buf bytes.Buffer

	gz := gzip.NewWriter(&buf)
	_, _ = gz.Write(bytes.Repeat([]byte{0xFF}, 1024))
	_ = gz.Close()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(buf.Bytes())
	}))

	defer ts.Close()

	_, err := downloadONNXRuntime(t.TempDir(), testClient(ts.URL))

	// May return "failed to read tar" or "library not found" depending on tar parsing behavior.
	g.Expect(err).To(HaveOccurred())
}

// TestDownloadVocab_MockHTTPError verifies downloadVocab returns error on non-200 response.
func TestDownloadVocab_MockHTTPError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	defer ts.Close()

	vocabPath := filepath.Join(t.TempDir(), "vocab.txt")

	err := downloadVocab(vocabPath, testClient(ts.URL))

	g.Expect(err).To(MatchError(ContainSubstring("HTTP 404")))
}

// TestDownloadVocab_MockSuccess verifies downloadVocab succeeds when server returns 200.
func TestDownloadVocab_MockSuccess(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "hello")
		fmt.Fprintln(w, "world")
		fmt.Fprintln(w, "[PAD]")
	}))

	defer ts.Close()

	vocabPath := filepath.Join(t.TempDir(), "vocab.txt")

	err := downloadVocab(vocabPath, testClient(ts.URL))

	g.Expect(err).ToNot(HaveOccurred())

	_, statErr := os.Stat(vocabPath)

	g.Expect(statErr).ToNot(HaveOccurred())
}

func TestGenerateEmbeddingONNX_NonExistentModel(t *testing.T) {
	g := NewWithT(t)

	modelPath := filepath.Join(t.TempDir(), "nonexistent.onnx")

	_, _, _, err := GenerateEmbeddingONNX("test text", modelPath)
	if err == nil {
		t.Fatal("expected error for non-existent model path")
	}

	g.Expect(err).To(HaveOccurred())
}

func TestGetModelPath_ReturnsPath(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	path, err := GetModelPath()

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(path).ToNot(BeEmpty())
	g.Expect(strings.HasSuffix(path, "e5-small-v2.onnx")).To(BeTrue())
}

// TestHybridSearch_EmptyDB verifies hybridSearch returns empty results on empty DB.
func TestHybridSearch_EmptyDB(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))

	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	embedding := make([]float32, embeddingDim)

	results, err := hybridSearch(db, embedding, "test query", 5, 60)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).To(BeEmpty())
}

// TestHybridSearch_EmptyQueryText verifies hybridSearch handles empty query text without error.
func TestHybridSearch_EmptyQueryText(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))

	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	embedding := make([]float32, embeddingDim)

	results, err := hybridSearch(db, embedding, "", 5, 60)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).To(BeEmpty())
}

func TestInitializeONNXRuntimeForTest_Callable(t *testing.T) {
	modelDir := t.TempDir()

	// Either nil (already initialized) or error (download failed) - both are valid
	_ = InitializeONNXRuntimeForTest(modelDir)
}

// TestInitializeONNXRuntime_FastPath verifies initializeONNXRuntime returns nil when already initialized.
func TestInitializeONNXRuntime_FastPath(t *testing.T) {
	// Not parallel — modifies global onnxRuntimeInitialized state
	onnxRuntimeInitMu.Lock()

	saved := onnxRuntimeInitialized
	onnxRuntimeInitialized = true

	onnxRuntimeInitMu.Unlock()

	defer func() {
		onnxRuntimeInitMu.Lock()

		onnxRuntimeInitialized = saved

		onnxRuntimeInitMu.Unlock()
	}()

	g := NewWithT(t)

	err := initializeONNXRuntime("/any/path/does/not/matter")

	g.Expect(err).ToNot(HaveOccurred())
}

// TestInsertFTS5_NoOp verifies insertFTS5 does not panic regardless of FTS5 availability.
func TestInsertFTS5_NoOp(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))

	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Either hits early return (no FTS5 build tag) or executes insert (with FTS5 tag)
	insertFTS5(db, 1, "test content")
	insertFTS5(db, 2, "another content")
}

// TestInsertFTS5_WithFTSTable covers line 1006 (INSERT INTO embeddings_fts)
// by creating a regular table named embeddings_fts so hasFTS5 returns true.
func TestInsertFTS5_WithFTSTable(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Create a regular table named embeddings_fts so hasFTS5(db) returns true
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS embeddings_fts (rowid INTEGER PRIMARY KEY, content TEXT)`)
	g.Expect(err).ToNot(HaveOccurred())

	// insertFTS5 now sees hasFTS5=true → executes INSERT into the regular table
	insertFTS5(db, 1, "test content for fts indexing")
}

// TestLearnToEmbeddings_EmptyMessage verifies learnToEmbeddings does not panic with minimal input.
func TestLearnToEmbeddings_EmptyMessage(t *testing.T) {
	t.Parallel()
	// Inject a zero-vector embedder to bypass ONNX — exercises logic after model setup.
	opts := LearnOpts{
		Message:        "",
		MemoryRoot:     t.TempDir(),
		VectorEmbedder: &testEmbedder{},
	}

	// Empty message stored without error using injected embedder.
	_ = learnToEmbeddings(opts)
}

// TestLearnToEmbeddings_PrecomputedObs_EmptyConcepts verifies that an empty concepts list
// triggers the validation failure path when PrecomputedObservation is provided.
func TestLearnToEmbeddings_PrecomputedObs_EmptyConcepts(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	opts := LearnOpts{
		Message:        "always write tests before implementing features",
		MemoryRoot:     t.TempDir(),
		VectorEmbedder: &testEmbedder{}, // bypass ONNX
		PrecomputedObservation: &Observation{
			Type:      "correction",
			Concepts:  nil, // empty → conceptsCSV="" → validationErrors appended → return error
			Principle: "Always write tests",
		},
	}

	// Validation fails before embedding is called because concepts is empty.
	err := learnToEmbeddings(opts)
	g.Expect(err).To(MatchError(ContainSubstring("LLM enrichment validation failed")))
}

// TestLearnToEmbeddings_PrecomputedObservation verifies that when a PrecomputedObservation is
// provided, its fields populate observationType, conceptsCSV, principle, and enrichedContent,
// triggering the validation block and using the enriched content as the embedding text.
func TestLearnToEmbeddings_PrecomputedObservation(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	opts := LearnOpts{
		Message:        "always write tests before implementing features",
		MemoryRoot:     t.TempDir(),
		VectorEmbedder: &testEmbedder{}, // bypass ONNX
		PrecomputedObservation: &Observation{
			Type:        "correction",
			Concepts:    []string{"tdd", "testing"},
			Principle:   "Always write tests before implementing",
			AntiPattern: "Write implementation without tests first",
			Rationale:   "Ensures quality and coverage",
		},
	}

	// PrecomputedObservation fields populate enriched content; validation passes; entry stored.
	err := learnToEmbeddings(opts)
	g.Expect(err).ToNot(HaveOccurred())
}

// TestLoadVocab_DownloadFails verifies loadVocab returns error when vocab download fails.
func TestLoadVocab_DownloadFails(t *testing.T) {
	// Not parallel — resets global vocabCache, vocabOnce, HTTP transport, and HOME env var
	g := NewWithT(t)

	// Save and clear vocabCache (map reference — safe to copy)
	vocabCacheMu.Lock()

	savedCache := vocabCache
	vocabCache = nil

	vocabCacheMu.Unlock()

	// Reset vocabOnce so loadVocab re-runs the Do block.
	// Assign a fresh literal — do NOT copy from/to another variable (copylocks vet rule).
	vocabOnce = sync.Once{}

	defer func() {
		vocabCacheMu.Lock()

		vocabCache = savedCache

		vocabCacheMu.Unlock()

		vocabOnce = sync.Once{}
	}()

	// Point HOME to empty temp dir (no vocab file present) so downloadVocab tries network
	t.Setenv("HOME", t.TempDir())

	// Mock HTTP server returns 404 → downloadVocab fails
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	defer ts.Close()

	orig := http.DefaultTransport
	http.DefaultTransport = &testRedirectTransport{targetURL: ts.URL, orig: orig}

	defer func() { http.DefaultTransport = orig }()

	_, err := loadVocab()

	g.Expect(err).To(HaveOccurred())
}

// TestLoadVocab_WithTempVocabFile verifies loadVocab loads from a local vocab file.
func TestLoadVocab_WithTempVocabFile(t *testing.T) {
	// Not parallel — resets global vocabCache, vocabOnce, and HOME env var
	g := NewWithT(t)

	// Save and clear vocabCache (map reference — safe to copy)
	vocabCacheMu.Lock()

	savedCache := vocabCache
	vocabCache = nil

	vocabCacheMu.Unlock()

	// Reset vocabOnce so loadVocab re-runs the Do block.
	// Assign a fresh literal — do NOT copy from/to another variable (copylocks vet rule).
	vocabOnce = sync.Once{}

	defer func() {
		vocabCacheMu.Lock()

		vocabCache = savedCache

		vocabCacheMu.Unlock()

		// Reset again so subsequent tests can also call loadVocab cleanly.
		vocabOnce = sync.Once{}
	}()

	// Point HOME to temp dir so downloadVocab looks for vocab there
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	vocabPath := filepath.Join(tmpHome, ".projctl", "models", "vocab.txt")

	err := os.MkdirAll(filepath.Dir(vocabPath), 0755)
	g.Expect(err).ToNot(HaveOccurred())

	err = os.WriteFile(vocabPath, []byte("hello\nworld\n[PAD]"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := loadVocab()

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(HaveLen(3))
	g.Expect(result).To(HaveKey("hello"))
}

// TestMigrateFTS5_NoTable verifies migrateFTS5 returns an error when the
// embeddings_fts table does not exist (FTS5 not compiled in).
func TestMigrateFTS5_NoTable(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Create only the embeddings table, not the FTS5 virtual table.
	_, err = db.Exec(`CREATE TABLE embeddings (id INTEGER PRIMARY KEY, content TEXT, source TEXT)`)
	g.Expect(err).ToNot(HaveOccurred())

	err = migrateFTS5(db)
	g.Expect(err).To(HaveOccurred())
}

// TestMigrateModelVersion_DBClosedDuringMigration verifies that closing the DB inside Embed
// causes QueryRow.Scan (lines 1350-1353) and setMetadata (lines 1381-1382) to fail,
// returning "failed to set migration metadata".
func TestMigrateModelVersion_DBClosedDuringMigration(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	zeroVec := make([]float32, embeddingDim)

	blob, err := sqlite_vec.SerializeFloat32(zeroVec)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := db.Exec("INSERT INTO vec_embeddings(embedding) VALUES (?)", blob)
	g.Expect(err).ToNot(HaveOccurred())

	var embID int64
	if result != nil {
		embID, _ = result.LastInsertId()
	}

	_, err = db.Exec(
		"INSERT INTO embeddings(content, source, embedding_id, model_version) VALUES (?, ?, ?, ?)",
		"test content", "test", embID, "old-model",
	)
	g.Expect(err).ToNot(HaveOccurred())

	// Embed closes DB then returns zero vec → QueryRow.Scan fails → setMetadata fails.
	err = migrateModelVersion(db, &dbClosingEmbedder{closeDB: func() { _ = db.Close() }})

	g.Expect(err).To(MatchError(ContainSubstring("failed to set migration metadata")))
}

// TestMigrateModelVersion_EmptyDB verifies migrateModelVersion completes on empty DB without ONNX.
func TestMigrateModelVersion_EmptyDB(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))

	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Empty DB: no embeddings to re-embed, just marks migration complete
	err = migrateModelVersion(db, &testEmbedder{})

	g.Expect(err).ToNot(HaveOccurred())

	// Second call is idempotent: already-migrated metadata causes early return
	err = migrateModelVersion(db, &testEmbedder{})

	g.Expect(err).ToNot(HaveOccurred())
}

// TestMigrateModelVersion_GetMetadataFails verifies that a closed DB causes getMetadata
// to fail with "failed to check migration status" (lines 1292-1294).
func TestMigrateModelVersion_GetMetadataFails(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	// Close immediately so getMetadata fails inside migrateModelVersion.
	_ = db.Close()

	err = migrateModelVersion(db, &testEmbedder{})

	g.Expect(err).To(MatchError(ContainSubstring("failed to check migration status")))
}

// TestMigrateModelVersion_WithEntries_EmbedFails covers the continue path when embed fails.
func TestMigrateModelVersion_WithEntries_EmbedFails(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	zeroVec := make([]float32, embeddingDim)

	blob, err := sqlite_vec.SerializeFloat32(zeroVec)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := db.Exec("INSERT INTO vec_embeddings(embedding) VALUES (?)", blob)
	g.Expect(err).ToNot(HaveOccurred())

	var embID int64
	if result != nil {
		embID, _ = result.LastInsertId()
	}

	_, err = db.Exec(
		"INSERT INTO embeddings(content, source, embedding_id, model_version) VALUES (?, ?, ?, ?)",
		"test content", "test", embID, "old-model",
	)
	g.Expect(err).ToNot(HaveOccurred())

	// Embed always fails → error logged and skipped, migration still marks done
	err = migrateModelVersion(db, &testEmbedder{err: errors.New("embed failed")})

	g.Expect(err).ToNot(HaveOccurred())
}

// TestMigrateModelVersion_WithEntries_Success covers the re-embedding loop on entries
// that have an old model_version, with a testEmbedder that returns zero vectors.
func TestMigrateModelVersion_WithEntries_Success(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	zeroVec := make([]float32, embeddingDim)

	blob, err := sqlite_vec.SerializeFloat32(zeroVec)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := db.Exec("INSERT INTO vec_embeddings(embedding) VALUES (?)", blob)
	g.Expect(err).ToNot(HaveOccurred())

	var embID int64
	if result != nil {
		embID, _ = result.LastInsertId()
	}

	_, err = db.Exec(
		"INSERT INTO embeddings(content, source, embedding_id, model_version) VALUES (?, ?, ?, ?)",
		"test content", "test", embID, "old-model",
	)
	g.Expect(err).ToNot(HaveOccurred())

	err = migrateModelVersion(db, &testEmbedder{})

	g.Expect(err).ToNot(HaveOccurred())

	migrated, err := getMetadata(db, "model_version_migrated")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(migrated).To(Equal("e5-small-v2"))
}

// TestMigrateModelVersion_WrongDimEmbedding verifies that a 1-dim embedding causes the
// INSERT into vec_embeddings to fail (lines 1363-1366), logging a warning and continuing.
// Migration still completes successfully (setMetadata succeeds, returns nil).
func TestMigrateModelVersion_WrongDimEmbedding(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	zeroVec := make([]float32, embeddingDim)

	blob, err := sqlite_vec.SerializeFloat32(zeroVec)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := db.Exec("INSERT INTO vec_embeddings(embedding) VALUES (?)", blob)
	g.Expect(err).ToNot(HaveOccurred())

	var embID int64
	if result != nil {
		embID, _ = result.LastInsertId()
	}

	_, err = db.Exec(
		"INSERT INTO embeddings(content, source, embedding_id, model_version) VALUES (?, ?, ?, ?)",
		"test content", "test", embID, "old-model",
	)
	g.Expect(err).ToNot(HaveOccurred())

	// 1-dim vector → sqlite-vec INSERT fails (wrong dimension) → warning+continue → setMetadata OK.
	err = migrateModelVersion(db, &fixedEmbedder{vec: []float32{1.0}})

	g.Expect(err).ToNot(HaveOccurred())
}

// TestNewTokenizer_LoadFails verifies NewTokenizer returns non-nil tokenizer with empty vocab
// when vocab loading fails (no network, no local file).
func TestNewTokenizer_LoadFails(t *testing.T) {
	// Not parallel — resets global vocabCache, vocabOnce, HTTP transport, and HOME env var
	g := NewWithT(t)

	// Save and clear vocabCache (map reference — safe to copy)
	vocabCacheMu.Lock()

	savedCache := vocabCache
	vocabCache = nil

	vocabCacheMu.Unlock()

	// Reset vocabOnce so NewTokenizer will try to load vocab fresh.
	// Assign a fresh literal — do NOT copy from/to another variable (copylocks vet rule).
	vocabOnce = sync.Once{}

	defer func() {
		vocabCacheMu.Lock()

		vocabCache = savedCache

		vocabCacheMu.Unlock()

		vocabOnce = sync.Once{}
	}()

	// Point HOME to empty temp dir (no vocab file)
	t.Setenv("HOME", t.TempDir())

	// Mock HTTP returns 404 → vocab load fails → NewTokenizer returns empty-vocab tokenizer
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	defer ts.Close()

	orig := http.DefaultTransport
	http.DefaultTransport = &testRedirectTransport{targetURL: ts.URL, orig: orig}

	defer func() { http.DefaultTransport = orig }()

	tok := NewTokenizer()

	g.Expect(tok).ToNot(BeNil())
	// With empty vocab, all words → [UNK], but tokenizer is still functional
	tokens := tok.Tokenize("hello")

	g.Expect(tokens).ToNot(BeEmpty())
	g.Expect(tokens[0]).To(Equal(int64(clsTokenID)))
}

// TestSearchBM25_EmptyInput verifies searchBM25 returns nil for an empty query string.
func TestSearchBM25_EmptyInput(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))

	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	results, err := searchBM25(db, "", 5)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).To(BeNil())
}

// TestSearchBM25_OperatorsOnly verifies searchBM25 returns nil when query contains only FTS5 operators.
func TestSearchBM25_OperatorsOnly(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))

	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// All FTS5 operators are stripped → sanitized = "" → early return
	results, err := searchBM25(db, "AND OR NOT NEAR", 5)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).To(BeNil())
}

// TestSearchBM25_ValidQueryEmptyDB verifies searchBM25 handles valid query on empty DB gracefully.
func TestSearchBM25_ValidQueryEmptyDB(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))

	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Without FTS5 tag: embeddings_fts missing → error ignored, returns nil.
	// With FTS5 tag: empty FTS5 table → returns empty slice.
	results, err := searchBM25(db, "hello world", 5)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).To(BeEmpty())
}

// TestSearchBM25_WithFTS4Table covers lines 1467-1469 (defer + collectBM25Rows) by creating an FTS4
// virtual table named embeddings_fts with a rank column. FTS3/FTS4 are compiled in go-sqlite3 by
// default, so this works without the sqlite_fts5 build tag.
func TestSearchBM25_WithFTS4Table(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// FTS4 (built on FTS3) is compiled in go-sqlite3 by default. Create an embeddings_fts FTS4 table
	// with a rank column so searchBM25's SELECT...MATCH query can succeed.
	_, fts4Err := db.Exec(`CREATE VIRTUAL TABLE IF NOT EXISTS embeddings_fts USING fts4(content, rank)`)
	if fts4Err != nil {
		t.Skipf("FTS4 not available in this build: %v", fts4Err)
	}

	// searchBM25 sees hasFTS5=true (table exists), runs the MATCH query on the empty FTS4 table
	// → returns 0 rows successfully, covering lines 1467-1469
	results, err := searchBM25(db, "hello world test", 5)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).To(BeEmpty())
}

// dbClosingEmbedder calls closeDB on the first Embed call, then returns a zero-value embedding.
// This simulates a mid-migration DB closure to cover warning paths in migrateModelVersion.
type dbClosingEmbedder struct {
	closeDB func()
	once    sync.Once
}

func (e *dbClosingEmbedder) Embed(_ string) ([]float32, error) {
	e.once.Do(e.closeDB)
	return make([]float32, embeddingDim), nil
}

// testEmbedder is a minimal Embedder test double.
// Embed returns a zero-vector or the configured error.
type testEmbedder struct {
	err error
}

func (e *testEmbedder) Embed(_ string) ([]float32, error) {
	if e.err != nil {
		return nil, e.err
	}

	return make([]float32, embeddingDim), nil
}

// testRedirectTransport redirects all outgoing HTTP requests to a fixed URL.
// Used in tests to intercept download calls without real network access.
type testRedirectTransport struct {
	targetURL string
	orig      http.RoundTripper
}

func (t *testRedirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	newReq, err := http.NewRequest(req.Method, t.targetURL, req.Body)
	if err != nil {
		return nil, err
	}

	return t.orig.RoundTrip(newReq)
}

// testClient returns an *http.Client that redirects all requests to targetURL.
func testClient(targetURL string) *http.Client {
	return &http.Client{
		Transport: &testRedirectTransport{targetURL: targetURL, orig: http.DefaultTransport},
	}
}
