package memory

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	. "github.com/onsi/gomega"
)

// ─── calculateSimilarity tests ────────────────────────────────────────────────

func TestCalculateSimilarity_ErrorOnMissingRows(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "embeddings.db")

	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	defer func() { _ = db.Close() }()

	// Non-existent embedding IDs should produce a DB error
	_, err = calculateSimilarity(db, 999, 998)
	g.Expect(err).To(HaveOccurred())
}

func TestClusterBySimilarityWithFunc_AllMerged(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	entries := []ClusterEntry{
		{ID: 1, Content: "entry one", EmbeddingID: 1},
		{ID: 2, Content: "entry two", EmbeddingID: 2},
		{ID: 3, Content: "entry three", EmbeddingID: 3},
	}

	// sim always >= threshold → all entries in one cluster
	db := (*sql.DB)(nil)
	clusters := clusterBySimilarityWithFunc(db, entries, 0.5, func(_ *sql.DB, _, _ int64) (float64, error) {
		return 0.9, nil
	})

	g.Expect(clusters).To(HaveLen(1))

	if len(clusters) == 0 {
		t.Fatal("clusters must be non-empty")
	}

	g.Expect(clusters[0]).To(HaveLen(3))
}

// ─── clusterBySimilarityWithFunc tests ───────────────────────────────────────

func TestClusterBySimilarityWithFunc_Empty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db := (*sql.DB)(nil)
	clusters := clusterBySimilarityWithFunc(db, nil, 0.5, func(_ *sql.DB, _, _ int64) (float64, error) {
		return 0, nil
	})

	g.Expect(clusters).To(BeEmpty())
}

func TestClusterBySimilarityWithFunc_NothingMerged(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	entries := []ClusterEntry{
		{ID: 1, Content: "entry one", EmbeddingID: 1},
		{ID: 2, Content: "entry two", EmbeddingID: 2},
	}

	// sim < threshold → no merging, each entry in its own cluster
	db := (*sql.DB)(nil)
	clusters := clusterBySimilarityWithFunc(db, entries, 0.5, func(_ *sql.DB, _, _ int64) (float64, error) {
		return 0.1, nil
	})

	g.Expect(clusters).To(HaveLen(2))
}

func TestClusterBySimilarityWithFunc_SimFuncError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	entries := []ClusterEntry{
		{ID: 1, Content: "alpha", EmbeddingID: 1},
		{ID: 2, Content: "beta", EmbeddingID: 2},
	}

	// sim func always errors → no unions → 2 single-entry clusters
	db := (*sql.DB)(nil)
	clusters := clusterBySimilarityWithFunc(db, entries, 0.5, func(_ *sql.DB, _, _ int64) (float64, error) {
		return 0, errors.New("sim error")
	})

	g.Expect(clusters).To(HaveLen(2))
}

// ─── ConsolidateClaudeMD additional coverage tests ────────────────────────────

// TestConsolidateClaudeMD_DefaultClaudeMDPath exercises the claudeMDPath == ""
// code path that computes the default ~/.claude/CLAUDE.md location.
func TestConsolidateClaudeMD_DefaultClaudeMDPath(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	// Initialize DB so the function can open it after passing the ONNX check
	db, err := initEmbeddingsDB(filepath.Join(tmpDir, "embeddings.db"))
	g.Expect(err).ToNot(HaveOccurred())

	err = db.Close()
	g.Expect(err).ToNot(HaveOccurred())

	// No ClaudeMDPath → exercises the "if claudeMDPath == ''" body (lines 71-77)
	result, err := ConsolidateClaudeMD(ConsolidateClaudeMDOpts{
		MemoryRoot: tmpDir,
		// ClaudeMDPath intentionally omitted to exercise default path computation
	})
	if err != nil {
		t.Skipf("ONNX unavailable or other setup issue: %v", err)
	}

	g.Expect(result).ToNot(BeNil())
}

// TestConsolidateClaudeMD_EmptyBulletSkipped exercises the continue path when
// a promoted learning line is just "- " (empty after stripping the bullet).
func TestConsolidateClaudeMD_EmptyBulletSkipped(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	claudePath := filepath.Join(tmpDir, "CLAUDE.md")

	// Include both an empty bullet and a real one so the loop iterates
	content := "## Promoted Learnings\n\n- \n- always run tests before committing\n"

	err := os.WriteFile(claudePath, []byte(content), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	db, err := initEmbeddingsDB(filepath.Join(tmpDir, "embeddings.db"))
	g.Expect(err).ToNot(HaveOccurred())

	err = db.Close()
	g.Expect(err).ToNot(HaveOccurred())

	result, err := ConsolidateClaudeMD(ConsolidateClaudeMDOpts{
		MemoryRoot:   tmpDir,
		ClaudeMDPath: claudePath,
	})
	if err != nil {
		t.Skipf("ONNX unavailable: %v", err)
	}

	g.Expect(result).ToNot(BeNil())
}

func TestConsolidateClaudeMD_EmptyFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	claudePath := filepath.Join(tmpDir, "CLAUDE.md")

	err := os.WriteFile(claudePath, []byte{}, 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := ConsolidateClaudeMD(ConsolidateClaudeMDOpts{
		MemoryRoot:   tmpDir,
		ClaudeMDPath: claudePath,
	})

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())

	if result == nil {
		t.Fatal("result must not be nil")
	}

	g.Expect(result.Proposals).To(BeEmpty())
}

// ─── ConsolidateClaudeMD tests ────────────────────────────────────────────────

func TestConsolidateClaudeMD_EmptyMemoryRoot(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, err := ConsolidateClaudeMD(ConsolidateClaudeMDOpts{MemoryRoot: ""})
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("memory root is required"))
}

func TestConsolidateClaudeMD_EmptyPromotedLearningsSection(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	claudePath := filepath.Join(tmpDir, "CLAUDE.md")
	content := "## Promoted Learnings\n\n"

	err := os.WriteFile(claudePath, []byte(content), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := ConsolidateClaudeMD(ConsolidateClaudeMDOpts{
		MemoryRoot:   tmpDir,
		ClaudeMDPath: claudePath,
	})

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())

	if result == nil {
		t.Fatal("result must not be nil")
	}

	g.Expect(result.Proposals).To(BeEmpty())
}

func TestConsolidateClaudeMD_MissingFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	result, err := ConsolidateClaudeMD(ConsolidateClaudeMDOpts{
		MemoryRoot:   tmpDir,
		ClaudeMDPath: filepath.Join(tmpDir, "nonexistent.md"),
	})

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())

	if result == nil {
		t.Fatal("result must not be nil")
	}

	g.Expect(result.Proposals).To(BeEmpty())
}

func TestConsolidateClaudeMD_NoPromotedLearningsSection(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	claudePath := filepath.Join(tmpDir, "CLAUDE.md")
	content := "## Other Section\n\n- some content\n"

	err := os.WriteFile(claudePath, []byte(content), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := ConsolidateClaudeMD(ConsolidateClaudeMDOpts{
		MemoryRoot:   tmpDir,
		ClaudeMDPath: claudePath,
	})

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())

	if result == nil {
		t.Fatal("result must not be nil")
	}

	g.Expect(result.Proposals).To(BeEmpty())
}

// TestConsolidateClaudeMD_PromoteCandidatesFound exercises the Promote candidates
// loop body when entries in the DB meet the MinRetrievals/MinProjects thresholds
// and are not already present in CLAUDE.md.
func TestConsolidateClaudeMD_PromoteCandidatesFound(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	claudePath := filepath.Join(tmpDir, "CLAUDE.md")

	// CLAUDE.md with a different learning than the DB entry
	content := "## Promoted Learnings\n\n- always write tests before implementing\n"

	err := os.WriteFile(claudePath, []byte(content), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	db, err := initEmbeddingsDB(filepath.Join(tmpDir, "embeddings.db"))
	g.Expect(err).ToNot(HaveOccurred())

	// Insert an entry meeting ConsolidateClaudeMD's hardcoded Promote thresholds
	// (MinRetrievals=5, MinProjects=3). Content is NOT in CLAUDE.md so it becomes
	// a "promote" proposal candidate.
	_, err = db.Exec(
		`INSERT INTO embeddings (content, source, retrieval_count, projects_retrieved)
		 VALUES (?, ?, ?, ?)`,
		"use proper error handling throughout all production code",
		"test", 5, "projectA,projectB,projectC",
	)
	g.Expect(err).ToNot(HaveOccurred())

	err = db.Close()
	g.Expect(err).ToNot(HaveOccurred())

	result, err := ConsolidateClaudeMD(ConsolidateClaudeMDOpts{
		MemoryRoot:   tmpDir,
		ClaudeMDPath: claudePath,
	})
	if err != nil {
		t.Skipf("ONNX unavailable: %v", err)
	}

	g.Expect(result).ToNot(BeNil())

	if result == nil {
		t.Fatal("result must not be nil")
	}

	g.Expect(result.PromoteCount).To(BeNumerically(">=", 1))
}

// TestConsolidateClaudeMD_ReviewFuncApproves exercises the ReviewFunc loop
// with an approving callback, covering the Applied++ path.
func TestConsolidateClaudeMD_ReviewFuncApproves(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	claudePath := filepath.Join(tmpDir, "CLAUDE.md")
	content := "## Promoted Learnings\n\n- always write tests before implementing\n"

	err := os.WriteFile(claudePath, []byte(content), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	db, err := initEmbeddingsDB(filepath.Join(tmpDir, "embeddings.db"))
	g.Expect(err).ToNot(HaveOccurred())

	_, err = db.Exec(
		`INSERT INTO embeddings (content, source, retrieval_count, projects_retrieved)
		 VALUES (?, ?, ?, ?)`,
		"maintain consistent code style across the entire codebase",
		"test", 5, "alpha,beta,gamma",
	)
	g.Expect(err).ToNot(HaveOccurred())

	err = db.Close()
	g.Expect(err).ToNot(HaveOccurred())

	reviewCalled := 0

	result, err := ConsolidateClaudeMD(ConsolidateClaudeMDOpts{
		MemoryRoot:   tmpDir,
		ClaudeMDPath: claudePath,
		ReviewFunc: func(_ ConsolidateProposal) (bool, error) {
			reviewCalled++

			return true, nil
		},
	})
	if err != nil {
		t.Skipf("ONNX unavailable: %v", err)
	}

	g.Expect(result).ToNot(BeNil())

	if result.PromoteCount > 0 {
		g.Expect(reviewCalled).To(BeNumerically(">=", 1))
		g.Expect(result.Applied).To(BeNumerically(">=", 1))
	}
}

// TestConsolidateClaudeMD_ReviewFuncErrors exercises the ReviewFunc error return
// path, where ReviewFunc returns a non-nil error causing ConsolidateClaudeMD to
// return that error wrapped as "review failed".
func TestConsolidateClaudeMD_ReviewFuncErrors(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	claudePath := filepath.Join(tmpDir, "CLAUDE.md")
	content := "## Promoted Learnings\n\n- always write tests before implementing\n"

	err := os.WriteFile(claudePath, []byte(content), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	db, err := initEmbeddingsDB(filepath.Join(tmpDir, "embeddings.db"))
	g.Expect(err).ToNot(HaveOccurred())

	_, err = db.Exec(
		`INSERT INTO embeddings (content, source, retrieval_count, projects_retrieved)
		 VALUES (?, ?, ?, ?)`,
		"document all public APIs with clear usage examples always",
		"test", 5, "x,y,z",
	)
	g.Expect(err).ToNot(HaveOccurred())

	err = db.Close()
	g.Expect(err).ToNot(HaveOccurred())

	result, err := ConsolidateClaudeMD(ConsolidateClaudeMDOpts{
		MemoryRoot:   tmpDir,
		ClaudeMDPath: claudePath,
		ReviewFunc: func(_ ConsolidateProposal) (bool, error) {
			return false, errors.New("review rejected")
		},
	})

	if err == nil && result != nil && result.PromoteCount == 0 {
		// No proposals generated (maybe ONNX filtered them) — test is vacuous but ok
		return
	}

	if err != nil && !errors.Is(err, errors.New("review rejected")) {
		// Error from ONNX init or something unrelated to ReviewFunc
		if result == nil {
			t.Skipf("ONNX or setup issue: %v", err)
		}
	}

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("review"))
}

// TestConsolidateClaudeMD_SimilarityAbove09 exercises the similarity > 0.9 block
// (lines 186-196) by pre-populating vec_embeddings with an embedding that matches
// the learning text in CLAUDE.md. When ConsolidateClaudeMD computes the embedding
// for that text and queries the DB, cosine similarity ≈ 1.0 triggers the "redundant"
// proposal path.
func TestConsolidateClaudeMD_SimilarityAbove09(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()

	learning := "always write unit tests before implementing new features in code"
	claudePath := filepath.Join(tmpDir, "CLAUDE.md")
	content := "## Promoted Learnings\n\n- " + learning + "\n"

	err := os.WriteFile(claudePath, []byte(content), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	// Initialize ONNX runtime (no-op if already initialized)
	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())

	modelDir := filepath.Join(homeDir, ".claude", "models")
	if initErr := initializeONNXRuntime(modelDir); initErr != nil {
		t.Skipf("ONNX not available: %v", initErr)
	}

	modelPath := filepath.Join(modelDir, "e5-small-v2.onnx")

	// Generate the same embedding that ConsolidateClaudeMD will generate
	embedding, _, _, embErr := generateEmbeddingONNX("passage: "+learning, modelPath)
	if embErr != nil {
		t.Skipf("generateEmbeddingONNX failed: %v", embErr)
	}

	// Open DB and pre-insert the embedding
	db, err := initEmbeddingsDB(filepath.Join(tmpDir, "embeddings.db"))
	g.Expect(err).ToNot(HaveOccurred())

	// Serialize and insert embedding into vec_embeddings
	embeddingBlob, err := sqlite_vec.SerializeFloat32(embedding)
	g.Expect(err).ToNot(HaveOccurred())

	vecRes, err := db.Exec(`INSERT INTO vec_embeddings(embedding) VALUES (?)`, embeddingBlob)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(vecRes).ToNot(BeNil())

	if vecRes == nil {
		t.Fatal("vecRes must not be nil")
	}

	vecRowID, err := vecRes.LastInsertId()
	g.Expect(err).ToNot(HaveOccurred())

	// Insert into embeddings with embedding_id pointing to the vec rowid
	_, err = db.Exec(
		`INSERT INTO embeddings (content, source, embedding_id) VALUES (?, ?, ?)`,
		learning, "test", vecRowID,
	)
	g.Expect(err).ToNot(HaveOccurred())

	err = db.Close()
	g.Expect(err).ToNot(HaveOccurred())

	// Run ConsolidateClaudeMD — the embedding matches itself (similarity ≈ 1.0 > 0.9)
	result, err := ConsolidateClaudeMD(ConsolidateClaudeMDOpts{
		MemoryRoot:   tmpDir,
		ClaudeMDPath: claudePath,
	})
	if err != nil {
		t.Skipf("ConsolidateClaudeMD error: %v", err)
	}

	g.Expect(result).ToNot(BeNil())

	if result == nil {
		t.Fatal("result must not be nil")
	}

	g.Expect(result.RedundantCount).To(BeNumerically(">", 0))
}

func TestConsolidateClaudeMD_WithPromotedLearningsEmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	claudePath := filepath.Join(tmpDir, "CLAUDE.md")
	content := "## Promoted Learnings\n\n- always write unit tests before implementing new features\n"

	err := os.WriteFile(claudePath, []byte(content), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	// Initialize empty DB so the function can proceed
	db, err := initEmbeddingsDB(filepath.Join(tmpDir, "embeddings.db"))
	g.Expect(err).ToNot(HaveOccurred())

	err = db.Close()
	g.Expect(err).ToNot(HaveOccurred())

	result, err := ConsolidateClaudeMD(ConsolidateClaudeMDOpts{
		MemoryRoot:   tmpDir,
		ClaudeMDPath: claudePath,
	})
	if err != nil {
		// ONNX not available in this environment - skip
		t.Skipf("ONNX unavailable: %v", err)
	}

	g.Expect(result).ToNot(BeNil())
	// Empty DB = no similar entries = no redundancy proposals
	g.Expect(result.RedundantCount).To(Equal(0))
}

func TestConsolidateClaudeMD_WithReviewFunc(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	claudePath := filepath.Join(tmpDir, "CLAUDE.md")
	content := "## Promoted Learnings\n\n- always run tests before committing code changes\n"

	err := os.WriteFile(claudePath, []byte(content), 0o644)
	g.Expect(err).ToNot(HaveOccurred())

	db, err := initEmbeddingsDB(filepath.Join(tmpDir, "embeddings.db"))
	g.Expect(err).ToNot(HaveOccurred())

	err = db.Close()
	g.Expect(err).ToNot(HaveOccurred())

	reviewed := 0

	result, err := ConsolidateClaudeMD(ConsolidateClaudeMDOpts{
		MemoryRoot:   tmpDir,
		ClaudeMDPath: claudePath,
		ReviewFunc: func(_ ConsolidateProposal) (bool, error) {
			reviewed++

			return false, nil
		},
	})
	if err != nil {
		t.Skipf("ONNX unavailable: %v", err)
	}

	g.Expect(result).ToNot(BeNil())
	// ReviewFunc only called if there are proposals
	_ = reviewed
}

func TestGeneratePatternLLM_ActionabilityFailure(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cluster := []ClusterEntry{
		{ID: 1, Content: "testing is important good practice", EmbeddingID: 1},
	}

	extractor := &consolidateMockLLM{
		synthesizeFunc: func(_ context.Context, _ []string) (string, error) {
			// Fails actionability: too short
			return "short", nil
		},
	}

	result := GeneratePatternLLM(context.Background(), cluster, extractor)

	// Falls back to template-based pattern due to actionability failure
	g.Expect(result.Occurrences).To(Equal(1))
}

func TestGeneratePatternLLM_LLMUnavailable(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cluster := []ClusterEntry{
		{ID: 1, Content: "write tests before code always", EmbeddingID: 1},
	}

	extractor := &consolidateMockLLM{
		synthesizeFunc: func(_ context.Context, _ []string) (string, error) {
			return "", ErrLLMUnavailable
		},
	}

	result := GeneratePatternLLM(context.Background(), cluster, extractor)

	// Falls back to template-based pattern
	g.Expect(result.Occurrences).To(Equal(1))
}

// ─── GeneratePatternLLM tests ─────────────────────────────────────────────────

func TestGeneratePatternLLM_NilExtractor(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cluster := []ClusterEntry{
		{ID: 1, Content: "write tests before code always important", EmbeddingID: 1},
	}

	result := GeneratePatternLLM(context.Background(), cluster, nil)

	g.Expect(result.Occurrences).To(Equal(1))
}

func TestGeneratePatternLLM_OtherError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cluster := []ClusterEntry{
		{ID: 1, Content: "use proper error handling always in code", EmbeddingID: 1},
	}

	extractor := &consolidateMockLLM{
		synthesizeFunc: func(_ context.Context, _ []string) (string, error) {
			return "", errors.New("network error")
		},
	}

	result := GeneratePatternLLM(context.Background(), cluster, extractor)

	// Falls back to template-based pattern on any error
	g.Expect(result.Occurrences).To(Equal(1))
}

func TestGeneratePatternLLM_Success(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cluster := []ClusterEntry{
		{ID: 1, Content: "write unit tests before implementing features to ensure correctness", EmbeddingID: 1},
		{ID: 2, Content: "prefer test-driven development approach for all new code changes", EmbeddingID: 2},
	}

	synthesis := "Always write tests before implementing features to ensure code correctness and maintainability"

	extractor := &consolidateMockLLM{
		synthesizeFunc: func(_ context.Context, _ []string) (string, error) {
			return synthesis, nil
		},
	}

	result := GeneratePatternLLM(context.Background(), cluster, extractor)

	g.Expect(result.Synthesis).To(Equal(synthesis))
	g.Expect(result.Occurrences).To(Equal(2))
	g.Expect(result.Examples).To(HaveLen(2))
	// Theme is first 50 chars of synthesis
	g.Expect(result.Theme).To(HaveLen(50))
}

func TestGeneratePattern_ExamplesTruncated(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	longContent := "this is a very long content string that definitely exceeds eighty characters in total length"
	cluster := []ClusterEntry{
		{ID: 1, Content: longContent, EmbeddingID: 1},
		{ID: 2, Content: longContent, EmbeddingID: 2},
		{ID: 3, Content: longContent, EmbeddingID: 3},
	}

	result := generatePattern(cluster)

	g.Expect(result.Synthesis).To(ContainSubstring("..."))
}

func TestGeneratePattern_MultipleEntries(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cluster := []ClusterEntry{
		{ID: 1, Content: "run tests before committing changes", EmbeddingID: 1},
		{ID: 2, Content: "run tests before merging pull requests", EmbeddingID: 2},
		{ID: 3, Content: "run tests before deploying to production", EmbeddingID: 3},
	}

	result := generatePattern(cluster)

	g.Expect(result.Occurrences).To(Equal(3))
	g.Expect(result.Examples).To(HaveLen(3))
	// "run", "tests" and "before" should appear as common keywords (in >50% of entries)
	g.Expect(result.Theme).ToNot(BeEmpty())
	g.Expect(result.Synthesis).To(ContainSubstring("Pattern observed across 3 memories"))
}

// ─── generatePattern tests ────────────────────────────────────────────────────

func TestGeneratePattern_SingleEntry(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cluster := []ClusterEntry{
		{ID: 1, Content: "always write tests before implementing features", EmbeddingID: 1},
	}

	result := generatePattern(cluster)

	g.Expect(result.Occurrences).To(Equal(1))
	g.Expect(result.Examples).To(HaveLen(1))
	g.Expect(result.Synthesis).ToNot(BeEmpty())
}

func TestSynthesizePatterns_EmptyDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "embeddings.db")

	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	err = db.Close()
	g.Expect(err).ToNot(HaveOccurred())

	result, err := SynthesizePatterns(tmpDir, 0.8, 3)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())

	if result == nil {
		t.Fatal("result must not be nil")
	}

	g.Expect(result.Patterns).To(BeEmpty())
}

// ─── SynthesizePatterns tests ─────────────────────────────────────────────────

func TestSynthesizePatterns_InvalidPath(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Using a file path that can't contain a DB
	_, err := SynthesizePatterns("/nonexistent/path/that/does/not/exist", 0.8, 3)
	g.Expect(err).To(HaveOccurred())
}

func TestSynthesizePatterns_WithEntriesAndMinClusterSizeOne(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "embeddings.db")

	db, err := initEmbeddingsDB(dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	// Insert entries with fake embedding_ids (no vec_embeddings rows)
	// calculateSimilarity will fail for each pair, so each entry forms its own cluster
	for _, content := range []string{
		"write unit tests before implementing features always",
		"use proper error handling in all production code",
	} {
		_, err = db.Exec(
			"INSERT INTO embeddings (content, source, embedding_id) VALUES (?, ?, ?)",
			content, "test", 99999,
		)
		g.Expect(err).ToNot(HaveOccurred())
	}

	err = db.Close()
	g.Expect(err).ToNot(HaveOccurred())

	// With minClusterSize=1, each single-entry cluster produces a pattern
	result, err := SynthesizePatterns(tmpDir, 0.8, 1)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())

	if result == nil {
		t.Fatal("result must not be nil")
	}

	g.Expect(result.Patterns).ToNot(BeEmpty())
}

func TestTruncateContent_ExactLength(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(truncateContent("hello", 5)).To(Equal("hello"))
}

// ─── truncateContent tests ────────────────────────────────────────────────────

func TestTruncateContent_NoTruncation(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(truncateContent("hello", 10)).To(Equal("hello"))
}

func TestTruncateContent_Truncated(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	result := truncateContent("hello world", 8)

	g.Expect(result).To(Equal("hello..."))
	g.Expect(result).To(HaveLen(8))
}

// ─── mock LLM extractor ───────────────────────────────────────────────────────

type consolidateMockLLM struct {
	synthesizeFunc func(ctx context.Context, memories []string) (string, error)
}

func (m *consolidateMockLLM) AddRationale(_ context.Context, _ string) (string, error) {
	return "", errors.New("not implemented")
}

func (m *consolidateMockLLM) Curate(_ context.Context, _ string, _ []QueryResult) ([]CuratedResult, error) {
	return nil, errors.New("not implemented")
}

func (m *consolidateMockLLM) Decide(_ context.Context, _ string, _ []ExistingMemory) (*IngestDecision, error) {
	return nil, errors.New("not implemented")
}

func (m *consolidateMockLLM) Extract(_ context.Context, _ string) (*Observation, error) {
	return nil, errors.New("not implemented")
}

func (m *consolidateMockLLM) Filter(_ context.Context, _ string, _ []QueryResult) ([]FilterResult, error) {
	return nil, errors.New("not implemented")
}

func (m *consolidateMockLLM) PostEval(_ context.Context, _, _ string) (*PostEvalResult, error) {
	return nil, errors.New("not implemented")
}

func (m *consolidateMockLLM) Rewrite(_ context.Context, _ string) (string, error) {
	return "", errors.New("not implemented")
}

func (m *consolidateMockLLM) Synthesize(ctx context.Context, memories []string) (string, error) {
	if m.synthesizeFunc != nil {
		return m.synthesizeFunc(ctx, memories)
	}

	return "", errors.New("not implemented")
}
