package memory

import (
	"errors"
	"path/filepath"
	"testing"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	. "github.com/onsi/gomega"
)

func TestFindSimilarMemoriesBatch_BadMemoryRoot(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	matcher := &MemoryStoreSemanticMatcher{
		MemoryRoot: "/dev/null/nonexistent",
		ModelDir:   t.TempDir(),
	}

	_, err := matcher.FindSimilarMemoriesBatch([]string{"test"}, 0.5, 5)
	g.Expect(err).To(HaveOccurred())
}

// TestFindSimilarMemoriesBatch_DefaultModelDir verifies FindSimilarMemoriesBatch
// uses home dir when ModelDir is empty. Uses injected embedder so the test is fast.
func TestFindSimilarMemoriesBatch_DefaultModelDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	matcher := &MemoryStoreSemanticMatcher{
		MemoryRoot:     t.TempDir(),
		VectorEmbedder: &testEmbedder{},
		// ModelDir intentionally left empty
	}

	results, err := matcher.FindSimilarMemoriesBatch([]string{"hello"}, 0.5, 5)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).To(HaveLen(1))
}

func TestFindSimilarMemoriesBatch_Empty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	matcher := &MemoryStoreSemanticMatcher{MemoryRoot: t.TempDir()}

	results, err := matcher.FindSimilarMemoriesBatch(nil, 0.5, 5)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).To(BeNil())
}

func TestFindSimilarMemoriesBatch_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	matcher := &MemoryStoreSemanticMatcher{
		MemoryRoot:     t.TempDir(),
		VectorEmbedder: &testEmbedder{},
	}

	results, err := matcher.FindSimilarMemoriesBatch([]string{"test query"}, 0.5, 5)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).To(HaveLen(1))
}

// TestFindSimilarMemoriesBatch_MultipleTexts verifies batch processing with multiple texts.
func TestFindSimilarMemoriesBatch_MultipleTexts(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	matcher := &MemoryStoreSemanticMatcher{
		MemoryRoot:     t.TempDir(),
		VectorEmbedder: &testEmbedder{},
	}

	results, err := matcher.FindSimilarMemoriesBatch([]string{"hello", "world", "test"}, 0.5, 5)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).To(HaveLen(3))
}

// TestQuerySingleWithDB_HybridSearchFails covers the hybridSearch error path (lines 142-145).
// Closing the DB before calling querySingleWithDB makes db.Query fail inside searchSimilar.
func TestQuerySingleWithDB_HybridSearchFails(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal("failed to init db:", err)
	}

	// Close DB so hybridSearch → searchSimilar → db.Query fails
	_ = db.Close()

	_, err = querySingleWithDB(db, "test text", &testEmbedder{}, 0.5, 5)
	g.Expect(err).To(HaveOccurred())
}

func TestQuerySingleWithDB_NoModel(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal("failed to init db:", err)
	}

	defer func() { _ = db.Close() }()

	// Embedder that always returns an error (simulates ONNX failure)
	failEmbed := &testEmbedder{err: errors.New("embed failed")}

	_, err = querySingleWithDB(db, "test text", failEmbed, 0.5, 5)
	if err == nil {
		t.Fatal("expected error for failing embedder")
	}

	g.Expect(err).To(HaveOccurred())
}

// TestQuerySingleWithDB_ReturnsMatches covers the success path (return matches, nil).
// Inserts an embedding and queries with the same vector to get a match above threshold.
func TestQuerySingleWithDB_ReturnsMatches(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal("failed to init db:", err)
	}

	defer func() { _ = db.Close() }()

	// Build a non-zero vector: 1.0 at index 0
	emb := make([]float32, embeddingDim)
	emb[0] = 1.0

	blob, err := sqlite_vec.SerializeFloat32(emb)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := db.Exec(`INSERT INTO vec_embeddings(embedding) VALUES (?)`, blob)
	if err != nil {
		t.Fatal("failed to insert vec embedding:", err)
	}

	vecID, _ := result.LastInsertId()

	_, err = db.Exec(`INSERT INTO embeddings(content, source, embedding_id) VALUES (?, ?, ?)`,
		"test match content", "test", vecID)
	g.Expect(err).ToNot(HaveOccurred())

	// Query with same vector → cosine distance=0, score=1.0*1.0=1.0 ≥ threshold 0.5
	matches, err := querySingleWithDB(db, "test text", &fixedEmbedder{vec: emb}, 0.5, 5)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(matches).ToNot(BeNil())
	g.Expect(matches).To(ContainElement("test match content"))
}

// TestQuerySingleWithDB_SuccessEmptyDB verifies querySingleWithDB returns nil on empty DB.
func TestQuerySingleWithDB_SuccessEmptyDB(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal("failed to init db:", err)
	}

	defer func() { _ = db.Close() }()

	// testEmbedder returns zero vector → hybridSearch runs → empty DB → nil results
	results, err := querySingleWithDB(db, "test text", &testEmbedder{}, 0.5, 5)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).To(BeNil())
}

// TestFindSimilarMemoriesBatch_EmbedError covers the continue-on-error path in the inner loop.
func TestFindSimilarMemoriesBatch_EmbedError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	matcher := &MemoryStoreSemanticMatcher{
		MemoryRoot:     t.TempDir(),
		VectorEmbedder: &testEmbedder{err: errors.New("embed fail")},
	}

	results, err := matcher.FindSimilarMemoriesBatch([]string{"a", "b"}, 0.5, 5)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).To(HaveLen(2))
	g.Expect(results[0]).To(BeNil())
	g.Expect(results[1]).To(BeNil())
}

// TestFindSimilarMemoriesBatch_ONNXPath exercises the full ONNX runtime path
// (VectorEmbedder nil, ModelDir empty) including the model-checks block at lines 119-148.
func TestFindSimilarMemoriesBatch_ONNXPath(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	matcher := &MemoryStoreSemanticMatcher{
		MemoryRoot: t.TempDir(),
		// VectorEmbedder nil, ModelDir empty → resolves from home dir
	}

	results, err := matcher.FindSimilarMemoriesBatch([]string{"test query"}, 0.5, 5)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).To(HaveLen(1))
}

// fixedEmbedder returns a specific pre-set vector.
type fixedEmbedder struct {
	vec []float32
}

func (f *fixedEmbedder) Embed(_ string) ([]float32, error) {
	return f.vec, nil
}
