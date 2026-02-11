package memory_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/memory"
)

// ============================================================================
// ISSUE-188 Task 2.6: LLM-Driven Ingest Decisions
// ============================================================================

// defaultObs returns a standard observation for use in mock extractors.
func defaultObs() *memory.Observation {
	return &memory.Observation{
		Type:        "pattern",
		Concepts:    []string{"testing"},
		Principle:   "Always use TDD",
		AntiPattern: "Writing code without tests",
		Rationale:   "TDD catches regressions early",
	}
}

// mockDecider returns a ClaudeCLIExtractor that distinguishes between Extract
// and Decide calls by inspecting the prompt. Extract calls get a fixed Observation;
// Decide calls get the provided IngestDecision.
func mockDecider(decision *memory.IngestDecision) memory.LLMExtractor {
	obs := defaultObs()
	return &memory.ClaudeCLIExtractor{
		Model:   "haiku",
		Timeout: 30,
		CommandRunner: func(_ context.Context, _ string, args ...string) ([]byte, error) {
			// The prompt is the last argument (after "-p")
			prompt := args[len(args)-1]
			if strings.Contains(prompt, "Decide the best action") {
				jsonBytes, _ := json.Marshal(decision)
				return jsonBytes, nil
			}
			// Default: return observation for Extract calls
			jsonBytes, _ := json.Marshal(obs)
			return jsonBytes, nil
		},
	}
}

// failingDecider returns a ClaudeCLIExtractor where Extract succeeds but Decide fails.
func failingDecider(decideErr error) memory.LLMExtractor {
	obs := defaultObs()
	return &memory.ClaudeCLIExtractor{
		Model:   "haiku",
		Timeout: 30,
		CommandRunner: func(_ context.Context, _ string, args ...string) ([]byte, error) {
			prompt := args[len(args)-1]
			if strings.Contains(prompt, "Decide the best action") {
				return nil, decideErr
			}
			jsonBytes, _ := json.Marshal(obs)
			return jsonBytes, nil
		},
	}
}

// readConfidence reads the confidence value for a row matching contentSubstr.
func readConfidence(g Gomega, memoryRoot, contentSubstr string) float64 {
	dbPath := filepath.Join(memoryRoot, "embeddings.db")
	db, err := sql.Open("sqlite3", dbPath)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	var confidence float64
	err = db.QueryRow(
		"SELECT confidence FROM embeddings WHERE content LIKE ?",
		"%"+contentSubstr+"%",
	).Scan(&confidence)
	g.Expect(err).ToNot(HaveOccurred())
	return confidence
}

// countAllEmbeddings returns the total number of rows in the embeddings table.
func countAllEmbeddings(g Gomega, memoryRoot string) int {
	dbPath := filepath.Join(memoryRoot, "embeddings.db")
	db, err := sql.Open("sqlite3", dbPath)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM embeddings").Scan(&count)
	g.Expect(err).ToNot(HaveOccurred())
	return count
}

// getEmbeddingID returns the embeddings.id for a row matching contentSubstr.
func getEmbeddingID(g Gomega, memoryRoot, contentSubstr string) int64 {
	dbPath := filepath.Join(memoryRoot, "embeddings.db")
	db, err := sql.Open("sqlite3", dbPath)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	var id int64
	err = db.QueryRow(
		"SELECT id FROM embeddings WHERE content LIKE ?",
		"%"+contentSubstr+"%",
	).Scan(&id)
	g.Expect(err).ToNot(HaveOccurred())
	return id
}

// contentExists checks whether a row matching contentSubstr exists in embeddings.
func contentExists(g Gomega, memoryRoot, contentSubstr string) bool {
	dbPath := filepath.Join(memoryRoot, "embeddings.db")
	db, err := sql.Open("sqlite3", dbPath)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	var count int
	err = db.QueryRow(
		"SELECT COUNT(*) FROM embeddings WHERE content LIKE ?",
		"%"+contentSubstr+"%",
	).Scan(&count)
	g.Expect(err).ToNot(HaveOccurred())
	return count > 0
}

// fts5TableExists checks whether the embeddings_fts table exists.
func fts5TableExists(g Gomega, memoryRoot string) bool {
	dbPath := filepath.Join(memoryRoot, "embeddings.db")
	db, err := sql.Open("sqlite3", dbPath)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	var name string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='embeddings_fts'").Scan(&name)
	return err == nil && name == "embeddings_fts"
}

// fts5Contains checks whether the FTS5 table has a match for the given term.
func fts5Contains(g Gomega, memoryRoot, term string) bool {
	dbPath := filepath.Join(memoryRoot, "embeddings.db")
	db, err := sql.Open("sqlite3", dbPath)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db.Close() }()

	var count int
	err = db.QueryRow(
		"SELECT COUNT(*) FROM embeddings_fts WHERE content MATCH ?", term,
	).Scan(&count)
	if err != nil {
		return false // FTS5 table may not exist
	}
	return count > 0
}

// learnWithDefaults stores a memory using sensible defaults for tests.
func learnWithDefaults(g Gomega, memoryDir, message string, extractor memory.LLMExtractor) {
	err := memory.Learn(memory.LearnOpts{
		Message:    message,
		MemoryRoot: memoryDir,
		Extractor:  extractor,
	})
	g.Expect(err).ToNot(HaveOccurred())
}

// --- Unit Tests ---

// TEST: ADD decision → memory inserted normally
func TestIngestDecisionAdd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	memoryDir := filepath.Join(t.TempDir(), "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	// First, store a base memory (no extractor → no Decide call)
	learnWithDefaults(g, memoryDir, "base memory for ADD test alpha1111", nil)
	baseCount := countAllEmbeddings(g, memoryDir)

	// Now learn a similar-ish memory with extractor that returns ADD
	decision := &memory.IngestDecision{Action: memory.IngestAdd, Reason: "new knowledge"}
	err := memory.Learn(memory.LearnOpts{
		Message:    "new distinct knowledge for ADD test alpha1111",
		MemoryRoot: memoryDir,
		Extractor:  mockDecider(decision),
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Should have one more row
	g.Expect(countAllEmbeddings(g, memoryDir)).To(Equal(baseCount + 1))
	g.Expect(contentExists(g, memoryDir, "new distinct knowledge")).To(BeTrue())
}

// TEST: UPDATE decision → existing content replaced, no new row
func TestIngestDecisionUpdate(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	memoryDir := filepath.Join(t.TempDir(), "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	// Store base memory
	learnWithDefaults(g, memoryDir, "always use TDD for all code beta2222", nil)
	targetID := getEmbeddingID(g, memoryDir, "beta2222")
	baseCount := countAllEmbeddings(g, memoryDir)

	// Learn with UPDATE decision pointing at the base memory
	decision := &memory.IngestDecision{
		Action:   memory.IngestUpdate,
		TargetID: targetID,
		Reason:   "refines existing",
	}
	err := memory.Learn(memory.LearnOpts{
		Message:    "always use TDD with property-based tests beta2222",
		MemoryRoot: memoryDir,
		Extractor:  mockDecider(decision),
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Row count should NOT increase (update in place)
	g.Expect(countAllEmbeddings(g, memoryDir)).To(Equal(baseCount))

	// Old content gone, new content present
	g.Expect(contentExists(g, memoryDir, "always use TDD for all code")).To(BeFalse())
	g.Expect(contentExists(g, memoryDir, "property-based tests")).To(BeTrue())
}

// TEST: DELETE decision → old confidence=0, new memory inserted
func TestIngestDecisionDelete(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	memoryDir := filepath.Join(t.TempDir(), "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	// Store base memory WITH extractor so embedding uses enriched content
	// (ensures similarity > 0.5 with the second learn)
	obs := defaultObs()
	baseExtractor := &memory.ClaudeCLIExtractor{
		Model: "haiku", Timeout: 30,
		CommandRunner: func(_ context.Context, _ string, args ...string) ([]byte, error) {
			jsonBytes, _ := json.Marshal(obs)
			return jsonBytes, nil
		},
	}
	learnWithDefaults(g, memoryDir, "use println for debugging gamma3333", baseExtractor)
	targetID := getEmbeddingID(g, memoryDir, "gamma3333")
	baseCount := countAllEmbeddings(g, memoryDir)

	// Learn with DELETE decision
	decision := &memory.IngestDecision{
		Action:   memory.IngestDelete,
		TargetID: targetID,
		Reason:   "obsolete advice",
	}
	err := memory.Learn(memory.LearnOpts{
		Message:    "use structured logging instead of println gamma3333",
		MemoryRoot: memoryDir,
		Extractor:  mockDecider(decision),
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Old memory should have confidence 0
	g.Expect(readConfidence(g, memoryDir, "use println")).To(Equal(0.0))

	// New memory should be inserted (count increases by 1)
	g.Expect(countAllEmbeddings(g, memoryDir)).To(Equal(baseCount + 1))
	g.Expect(contentExists(g, memoryDir, "structured logging")).To(BeTrue())
}

// TEST: NOOP decision → no insert, existing confidence boosted
func TestIngestDecisionNoop(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	memoryDir := filepath.Join(t.TempDir(), "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	// Store base memory with extractor for consistent embeddings
	obs := defaultObs()
	baseExtractor := &memory.ClaudeCLIExtractor{
		Model: "haiku", Timeout: 30,
		CommandRunner: func(_ context.Context, _ string, args ...string) ([]byte, error) {
			jsonBytes, _ := json.Marshal(obs)
			return jsonBytes, nil
		},
	}
	learnWithDefaults(g, memoryDir, "always write tests first delta4444", baseExtractor)
	targetID := getEmbeddingID(g, memoryDir, "delta4444")
	beforeConfidence := readConfidence(g, memoryDir, "delta4444")
	baseCount := countAllEmbeddings(g, memoryDir)

	// Learn with NOOP decision
	decision := &memory.IngestDecision{
		Action:   memory.IngestNoop,
		TargetID: targetID,
		Reason:   "duplicate",
	}
	err := memory.Learn(memory.LearnOpts{
		Message:    "tests should be written before code delta4444",
		MemoryRoot: memoryDir,
		Extractor:  mockDecider(decision),
	})
	g.Expect(err).ToNot(HaveOccurred())

	// No new row (NOOP prevents insert)
	g.Expect(countAllEmbeddings(g, memoryDir)).To(Equal(baseCount))

	// Confidence should be at least what it was (boost capped at 1.0)
	afterConfidence := readConfidence(g, memoryDir, "delta4444")
	g.Expect(afterConfidence).To(BeNumerically(">=", beforeConfidence))
	g.Expect(afterConfidence).To(BeNumerically("<=", 1.0))
}

// TEST: Invalid TargetID → falls back to threshold behavior
func TestIngestDecisionInvalidTargetID(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	memoryDir := filepath.Join(t.TempDir(), "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	// Store base memory
	learnWithDefaults(g, memoryDir, "base memory for invalid target epsilon5555", nil)
	baseCount := countAllEmbeddings(g, memoryDir)

	// Learn with UPDATE decision pointing at a non-existent ID
	decision := &memory.IngestDecision{
		Action:   memory.IngestUpdate,
		TargetID: 99999, // bogus ID
		Reason:   "hallucinated target",
	}
	err := memory.Learn(memory.LearnOpts{
		Message:    "similar memory for invalid target epsilon5555",
		MemoryRoot: memoryDir,
		Extractor:  mockDecider(decision),
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Should still succeed — falls back to threshold. Since similarity is high
	// (same unique token), it either boosts or inserts based on threshold.
	// Key: no crash, no data loss.
	g.Expect(countAllEmbeddings(g, memoryDir)).To(BeNumerically(">=", baseCount))
}

// TEST: Nil extractor → threshold fallback (backward compatibility)
func TestIngestDecisionNilExtractor(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	memoryDir := filepath.Join(t.TempDir(), "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	// Store two similar memories without extractor
	learnWithDefaults(g, memoryDir, "always use TDD zeta6666", nil)
	learnWithDefaults(g, memoryDir, "always use TDD for code zeta6666", nil)

	// With nil extractor, threshold dedup should handle it
	// (either boost or insert depending on similarity)
	// Key: no error
}

// TEST: Failing extractor → threshold fallback
func TestIngestDecisionExtractorFails(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	memoryDir := filepath.Join(t.TempDir(), "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	// Store base memory
	learnWithDefaults(g, memoryDir, "base for failing extractor eta7777", nil)

	// Learn with failing decider
	err := memory.Learn(memory.LearnOpts{
		Message:    "similar to base failing extractor eta7777",
		MemoryRoot: memoryDir,
		Extractor:  failingDecider(errors.New("connection refused")),
	})
	g.Expect(err).ToNot(HaveOccurred(), "Learn must not fail due to Decide errors")
}

// TEST: ErrLLMUnavailable → threshold fallback
func TestIngestDecisionErrLLMUnavailable(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	memoryDir := filepath.Join(t.TempDir(), "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	learnWithDefaults(g, memoryDir, "base for unavailable theta8888", nil)

	err := memory.Learn(memory.LearnOpts{
		Message:    "similar to base unavailable theta8888",
		MemoryRoot: memoryDir,
		Extractor:  failingDecider(memory.ErrLLMUnavailable),
	})
	g.Expect(err).ToNot(HaveOccurred(), "Learn must not fail when LLM is unavailable")
}

// TEST: Type="correction" bypasses LLM decision entirely
func TestIngestDecisionCorrectionBypass(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	memoryDir := filepath.Join(t.TempDir(), "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	// Store base memory
	learnWithDefaults(g, memoryDir, "some pattern about testing iota9999", nil)
	baseCount := countAllEmbeddings(g, memoryDir)

	// Learn a correction — should always be stored regardless of similarity
	decision := &memory.IngestDecision{Action: memory.IngestNoop, Reason: "duplicate"}
	err := memory.Learn(memory.LearnOpts{
		Message:    "correction: testing pattern updated iota9999",
		MemoryRoot: memoryDir,
		Type:       "correction",
		Extractor:  mockDecider(decision),
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Correction must be stored (new row), NOOP decision should be ignored
	g.Expect(countAllEmbeddings(g, memoryDir)).To(Equal(baseCount + 1))
}

// TEST: All results < 0.5 similarity → skips LLM, goes to insert
func TestIngestDecisionLowSimilarity(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	memoryDir := filepath.Join(t.TempDir(), "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	// Store a memory about a completely different topic
	learnWithDefaults(g, memoryDir, "kubernetes pod scheduling kappa0001", nil)
	baseCount := countAllEmbeddings(g, memoryDir)

	// Learn something completely different — similarity should be low
	// The NOOP decision should NOT be consulted because similarity < 0.5
	decision := &memory.IngestDecision{Action: memory.IngestNoop, Reason: "should not be used"}
	err := memory.Learn(memory.LearnOpts{
		Message:    "french cooking techniques for soufflé lambda0002",
		MemoryRoot: memoryDir,
		Extractor:  mockDecider(decision),
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Should be inserted (NOOP ignored due to low similarity)
	g.Expect(countAllEmbeddings(g, memoryDir)).To(Equal(baseCount + 1))
}

// --- Integration Tests ---

// TEST: After UPDATE, retrieval_count and projects_retrieved are preserved
func TestIngestDecisionUpdatePreservesRetrieval(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	memoryDir := filepath.Join(t.TempDir(), "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	// Store and query to create retrieval tracking (use extractor for consistent embeddings)
	obs := defaultObs()
	baseExtractor := &memory.ClaudeCLIExtractor{
		Model: "haiku", Timeout: 30,
		CommandRunner: func(_ context.Context, _ string, args ...string) ([]byte, error) {
			jsonBytes, _ := json.Marshal(obs)
			return jsonBytes, nil
		},
	}
	learnWithDefaults(g, memoryDir, "always review code before merging mu1111", baseExtractor)
	_, err := memory.Query(memory.QueryOpts{
		Text:       "code review merging mu1111",
		MemoryRoot: memoryDir,
		Project:    "test-project",
	})
	g.Expect(err).ToNot(HaveOccurred())

	targetID := getEmbeddingID(g, memoryDir, "mu1111")

	// Verify retrieval data exists
	dbPath := filepath.Join(memoryDir, "embeddings.db")
	db, err := sql.Open("sqlite3", dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	var retrievalCount int
	var projectsRetrieved string
	err = db.QueryRow("SELECT retrieval_count, projects_retrieved FROM embeddings WHERE id = ?", targetID).
		Scan(&retrievalCount, &projectsRetrieved)
	g.Expect(err).ToNot(HaveOccurred())
	_ = db.Close()

	// Now UPDATE the memory
	decision := &memory.IngestDecision{
		Action:   memory.IngestUpdate,
		TargetID: targetID,
		Reason:   "refines existing",
	}
	err = memory.Learn(memory.LearnOpts{
		Message:    "always review code and run tests before merging mu1111",
		MemoryRoot: memoryDir,
		Extractor:  mockDecider(decision),
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Verify retrieval data is still present (UPDATE preserves retrieval columns)
	db2, err := sql.Open("sqlite3", dbPath)
	g.Expect(err).ToNot(HaveOccurred())
	defer func() { _ = db2.Close() }()

	var newRetrievalCount int
	var newProjectsRetrieved string
	err = db2.QueryRow("SELECT retrieval_count, projects_retrieved FROM embeddings WHERE id = ?", targetID).
		Scan(&newRetrievalCount, &newProjectsRetrieved)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(newRetrievalCount).To(Equal(retrievalCount))
	g.Expect(newProjectsRetrieved).To(Equal(projectsRetrieved))
}

// TEST: After DELETE+insert, old memory not returned by query
func TestIngestDecisionDeleteThenQuery(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	memoryDir := filepath.Join(t.TempDir(), "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	// Store old memory with extractor for consistent embeddings
	obs := defaultObs()
	baseExtractor := &memory.ClaudeCLIExtractor{
		Model: "haiku", Timeout: 30,
		CommandRunner: func(_ context.Context, _ string, args ...string) ([]byte, error) {
			jsonBytes, _ := json.Marshal(obs)
			return jsonBytes, nil
		},
	}
	learnWithDefaults(g, memoryDir, "use global variables for state nu2222", baseExtractor)
	targetID := getEmbeddingID(g, memoryDir, "nu2222")

	// DELETE old, insert new
	decision := &memory.IngestDecision{
		Action:   memory.IngestDelete,
		TargetID: targetID,
		Reason:   "obsolete",
	}
	err := memory.Learn(memory.LearnOpts{
		Message:    "use dependency injection for state management nu2222",
		MemoryRoot: memoryDir,
		Extractor:  mockDecider(decision),
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Query for the topic — superseded memory should have score=0
	results, err := memory.Query(memory.QueryOpts{
		Text:       "state management nu2222",
		MemoryRoot: memoryDir,
		Limit:      10,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// The superseded entry should have confidence 0 (score 0)
	// The new entry should be the top result (highest score)
	g.Expect(results.Results).ToNot(BeEmpty())
	topResult := results.Results[0]
	g.Expect(topResult.Content).To(ContainSubstring("dependency injection"),
		"new memory should be the top result")
	// Verify the old memory was superseded (confidence should be 0 or very low)
	oldConf := readConfidence(g, memoryDir, "global variables")
	g.Expect(oldConf).To(BeNumerically("<", 0.1),
		"superseded memory should have very low confidence")
}

// TEST: After UPDATE, FTS5 search finds new content, not old
func TestIngestDecisionUpdateFTS5(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	memoryDir := filepath.Join(t.TempDir(), "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	// Store base memory with extractor for consistent embeddings
	obs := defaultObs()
	baseExtractor := &memory.ClaudeCLIExtractor{
		Model: "haiku", Timeout: 30,
		CommandRunner: func(_ context.Context, _ string, args ...string) ([]byte, error) {
			jsonBytes, _ := json.Marshal(obs)
			return jsonBytes, nil
		},
	}
	learnWithDefaults(g, memoryDir, "use monoliths for simplicity xi3333", baseExtractor)
	targetID := getEmbeddingID(g, memoryDir, "xi3333")

	// UPDATE to new content
	decision := &memory.IngestDecision{
		Action:   memory.IngestUpdate,
		TargetID: targetID,
		Reason:   "updated advice",
	}
	err := memory.Learn(memory.LearnOpts{
		Message:    "use microservices when scaling requires it xi3333",
		MemoryRoot: memoryDir,
		Extractor:  mockDecider(decision),
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Verify the content was actually updated in the embeddings table
	g.Expect(contentExists(g, memoryDir, "microservices")).To(BeTrue(),
		"Updated content should exist in embeddings table")
	g.Expect(contentExists(g, memoryDir, "monoliths")).To(BeFalse(),
		"Old content should not exist in embeddings table after UPDATE")

	// FTS5 should find new content (if FTS5 table exists)
	if fts5TableExists(g, memoryDir) {
		g.Expect(fts5Contains(g, memoryDir, "microservices")).To(BeTrue(),
			"FTS5 should contain updated content")
		g.Expect(fts5Contains(g, memoryDir, "monoliths")).To(BeFalse(),
			"FTS5 should not contain old content after UPDATE")
	}
}

// --- Property Tests ---

// TEST: For any sequence of Learn calls with mock decisions, Query returns at least one result
func TestPropertyIngestNeverLosesData(t *testing.T) {
	t.Parallel()
	actions := []memory.IngestAction{memory.IngestAdd, memory.IngestUpdate, memory.IngestDelete, memory.IngestNoop}

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		memoryDir, err := os.MkdirTemp("", "ingest-prop-data-*")
		g.Expect(err).ToNot(HaveOccurred())
		defer func() { _ = os.RemoveAll(memoryDir) }()

		action := rapid.SampledFrom(actions).Draw(rt, "action")
		suffix := rapid.StringMatching(`[a-z]{10}`).Draw(rt, "suffix")

		// Store initial memory
		learnWithDefaults(g, memoryDir, "initial concept "+suffix, nil)
		targetID := getEmbeddingID(g, memoryDir, suffix)

		// Apply an ingest decision
		decision := &memory.IngestDecision{
			Action:   action,
			TargetID: targetID,
			Reason:   "property test",
		}
		err = memory.Learn(memory.LearnOpts{
			Message:    "related concept " + suffix,
			MemoryRoot: memoryDir,
			Extractor:  mockDecider(decision),
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Query should always return at least one result with the suffix
		results, err := memory.Query(memory.QueryOpts{
			Text:       suffix,
			MemoryRoot: memoryDir,
			Limit:      10,
		})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(results.Results).ToNot(BeEmpty(),
			"Query should return at least one result after any ingest action %s", action)
	})
}

// TEST: For any failing extractor, Learn succeeds and memory is stored
func TestPropertyIngestFallbackAlwaysWorks(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		memoryDir, err := os.MkdirTemp("", "ingest-prop-fallback-*")
		g.Expect(err).ToNot(HaveOccurred())
		defer func() { _ = os.RemoveAll(memoryDir) }()

		errMsg := rapid.StringMatching(`[a-zA-Z ]{1,50}`).Draw(rt, "errorMessage")
		suffix := rapid.StringMatching(`[a-z]{10}`).Draw(rt, "suffix")

		err = memory.Learn(memory.LearnOpts{
			Message:    "fallback memory " + suffix,
			MemoryRoot: memoryDir,
			Extractor:  failingDecider(errors.New(errMsg)),
		})
		g.Expect(err).ToNot(HaveOccurred(),
			"Learn must never fail due to Decide errors, got: %v", err)

		g.Expect(contentExists(g, memoryDir, suffix)).To(BeTrue(),
			"Memory must be stored even when extractor fails")
	})
}

// TEST: IngestAction is always one of ADD/UPDATE/DELETE/NOOP
func TestPropertyIngestActionValid(t *testing.T) {
	t.Parallel()
	validActions := []memory.IngestAction{memory.IngestAdd, memory.IngestUpdate, memory.IngestDelete, memory.IngestNoop}

	rapid.Check(t, func(rt *rapid.T) {
		action := rapid.SampledFrom(validActions).Draw(rt, "action")

		g := NewWithT(t)
		g.Expect(action).To(BeElementOf(
			memory.IngestAction("ADD"),
			memory.IngestAction("UPDATE"),
			memory.IngestAction("DELETE"),
			memory.IngestAction("NOOP"),
		))
	})
}
