package memory

import (
	"database/sql"
	"testing"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	. "github.com/onsi/gomega"
)

// TestSpreadingActivation_BoostsSecondaryResult verifies that memories
// semantically similar to top-K results are included with a boosted score.
// Memory B (similar to A) should be activated; Memory C (orthogonal) should not.
func TestSpreadingActivation_BoostsSecondaryResult(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := InitEmbeddingsDBForTest(t.TempDir() + "/spreading.db")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// vec_A: query-like [1, 0, 0, ..., 0]
	vecA := make([]float32, embeddingDim)
	vecA[0] = 1.0

	// vec_B: similar to A (cosine sim = 0.8 > threshold 0.7)
	// |vecB| = sqrt(0.8² + 0.6²) = sqrt(0.64 + 0.36) = 1.0 (already normalized)
	vecB := make([]float32, embeddingDim)
	vecB[0] = 0.8
	vecB[1] = 0.6

	// vec_C: orthogonal to A (cosine sim = 0.0 < threshold 0.7)
	vecC := make([]float32, embeddingDim)
	vecC[embeddingDim-1] = 1.0

	idA := insertSpreadTestVec(t, db, "Memory A: database query patterns", vecA)
	idB := insertSpreadTestVec(t, db, "Memory B: SQL optimization techniques", vecB)
	_ = insertSpreadTestVec(t, db, "Memory C: cooking recipes", vecC)

	// Simulate top-K from a prior query search (only A directly matched query)
	topK := []QueryResult{
		{ID: idA, Content: "Memory A: database query patterns", Score: 0.9, MatchType: "vector"},
	}

	results, err := applySpreadingActivation(db, topK, 0.7)
	g.Expect(err).ToNot(HaveOccurred())

	// B should appear — similarity to A ≈ 0.8 > threshold 0.7
	var (
		foundB bool
		bScore float64
	)

	for _, r := range results {
		if r.ID == idB {
			foundB = true
			bScore = r.Score
		}
	}

	g.Expect(foundB).To(BeTrue(), "spreading activation should boost B (cos sim to A > threshold)")

	// B's score must be strictly less than A's (secondary, not primary)
	g.Expect(bScore).To(BeNumerically(">", 0))
	g.Expect(bScore).To(BeNumerically("<", topK[0].Score))

	// A remains the top result
	if len(results) < 1 {
		t.Fatal("expected at least 1 result from applySpreadingActivation")
	}

	g.Expect(results[0].ID).To(Equal(idA))

	// C must not appear — it is orthogonal to A (cos sim = 0.0)
	for _, r := range results {
		g.Expect(r.Content).ToNot(ContainSubstring("cooking"))
	}
}

// TestSpreadingActivation_EmptyTopK returns empty for nil/empty input.
func TestSpreadingActivation_EmptyTopK(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := InitEmbeddingsDBForTest(t.TempDir() + "/empty.db")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	results, err := applySpreadingActivation(db, nil, 0.7)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).To(BeEmpty())
}

// TestSpreadingActivation_NoDuplicatesFromTopK verifies that primary results
// are not duplicated in the secondary activation pass.
func TestSpreadingActivation_NoDuplicatesFromTopK(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := InitEmbeddingsDBForTest(t.TempDir() + "/dedup.db")
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	vecA := make([]float32, embeddingDim)
	vecA[0] = 1.0
	idA := insertSpreadTestVec(t, db, "Memory A only", vecA)

	topK := []QueryResult{
		{ID: idA, Content: "Memory A only", Score: 0.9, MatchType: "vector"},
	}

	results, err := applySpreadingActivation(db, topK, 0.7)
	g.Expect(err).ToNot(HaveOccurred())

	// Only one result (A); the self-similar search must not duplicate it
	count := 0

	for _, r := range results {
		if r.ID == idA {
			count++
		}
	}

	g.Expect(count).To(Equal(1))
}

// TestSpreadingActivation_OptionInQueryOpts verifies SpreadingActivation field exists.
func TestSpreadingActivation_OptionInQueryOpts(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	opts := QueryOpts{SpreadingActivation: true}
	g.Expect(opts.SpreadingActivation).To(BeTrue())
}

// insertSpreadTestVec inserts a memory with a pre-computed embedding vector into the test DB.
func insertSpreadTestVec(t *testing.T, db *sql.DB, content string, vec []float32) int64 {
	t.Helper()

	blob, err := sqlite_vec.SerializeFloat32(vec)
	if err != nil {
		t.Fatalf("insertSpreadTestVec: serialize: %v", err)
	}

	vecResult, err := db.Exec("INSERT INTO vec_embeddings(embedding) VALUES (?)", blob)
	if err != nil {
		t.Fatalf("insertSpreadTestVec: insert vec: %v", err)
	}

	vecID, _ := vecResult.LastInsertId()

	metaResult, err := db.Exec(
		`INSERT INTO embeddings(content, source, source_type, confidence, embedding_id, model_version)
		 VALUES (?, 'memory', 'internal', 1.0, ?, 'e5-small-v2')`,
		content, vecID,
	)
	if err != nil {
		t.Fatalf("insertSpreadTestVec: insert metadata: %v", err)
	}

	id, _ := metaResult.LastInsertId()

	return id
}
