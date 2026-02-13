package memory_test

import (
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/memory"
)

// TestQueryResult_IDField verifies that QueryResult has an ID field and it's populated
func TestQueryResult_IDField(t *testing.T) {
	g := NewWithT(t)

	memoryRoot := t.TempDir()

	// Learn a memory entry
	err := memory.Learn(memory.LearnOpts{
		Message:    "Test memory for ID field",
		Project:    "testproject",
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).To(BeNil())

	// Query it back
	result, err := memory.Query(memory.QueryOpts{
		Text:       "Test memory for ID field",
		Limit:      1,
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).To(BeNil())
	g.Expect(result.Results).To(HaveLen(1))

	// Verify ID field is populated and > 0
	g.Expect(result.Results[0].ID).To(BeNumerically(">", 0))
}

// TestFormatMarkdown_NumberedOutput verifies that numbered output format is used
func TestFormatMarkdown_NumberedOutput(t *testing.T) {
	g := NewWithT(t)

	memoryRoot := t.TempDir()

	// Learn two memories
	err := memory.Learn(memory.LearnOpts{
		Message:    "First memory",
		Project:    "testproject",
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).To(BeNil())

	err = memory.Learn(memory.LearnOpts{
		Message:    "Second memory",
		Project:    "testproject",
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).To(BeNil())

	// Query both back
	result, err := memory.Query(memory.QueryOpts{
		Text:       "memory",
		Limit:      2,
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).To(BeNil())
	g.Expect(result.Results).To(HaveLen(2))

	// Format as markdown
	output := memory.FormatMarkdown(memory.FormatMarkdownOpts{
		Results:    result.Results,
		MaxEntries: 2,
	})

	// Verify numbered format: "1. " instead of "- "
	g.Expect(output).To(ContainSubstring("1. "))
	g.Expect(output).To(ContainSubstring("2. "))
}

// TestFormatMarkdown_RichModeShowsID verifies that rich mode shows (id=N)
func TestFormatMarkdown_RichModeShowsID(t *testing.T) {
	g := NewWithT(t)

	memoryRoot := t.TempDir()

	// Learn a memory
	err := memory.Learn(memory.LearnOpts{
		Message:    "Test memory",
		Project:    "testproject",
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).To(BeNil())

	// Query it back
	result, err := memory.Query(memory.QueryOpts{
		Text:       "Test memory",
		Limit:      1,
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).To(BeNil())
	g.Expect(result.Results).To(HaveLen(1))

	// Format as markdown with rich mode
	output := memory.FormatMarkdown(memory.FormatMarkdownOpts{
		Results:    result.Results,
		MaxEntries: 1,
		Tier:       memory.TierFull,
	})

	// Verify ID is shown in rich mode
	g.Expect(output).To(MatchRegexp(`id=\d+`))
}

// TestRecordFeedback_Helpful verifies helpful feedback increases confidence
func TestRecordFeedback_Helpful(t *testing.T) {
	g := NewWithT(t)

	memoryRoot := t.TempDir()

	// Learn an external memory (starts at 0.7 confidence instead of 1.0)
	err := memory.Learn(memory.LearnOpts{
		Message:    "Test memory",
		Project:    "testproject",
		Source:     "external",
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).To(BeNil())

	// Query to get ID
	result, err := memory.Query(memory.QueryOpts{
		Text:       "Test memory",
		Limit:      1,
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).To(BeNil())
	g.Expect(result.Results).To(HaveLen(1))

	embeddingID := result.Results[0].ID
	initialConfidence := result.Results[0].Confidence

	// Record helpful feedback
	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).To(BeNil())
	defer db.Close()

	err = memory.RecordFeedback(db, embeddingID, memory.FeedbackHelpful)
	g.Expect(err).To(BeNil())

	// Query again with completely different text to avoid re-ask detection
	result2, err := memory.Query(memory.QueryOpts{
		Text:       "goroutine concurrency patterns in Go programming language",
		Limit:      10, // Query more to ensure we get our result
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).To(BeNil())
	g.Expect(len(result2.Results)).To(BeNumerically(">=", 1))

	// Find our original result by ID and verify confidence increased
	var foundResult *memory.QueryResult
	for i := range result2.Results {
		if result2.Results[i].ID == embeddingID {
			foundResult = &result2.Results[i]
			break
		}
	}
	g.Expect(foundResult).ToNot(BeNil(), "should find the original result")

	// Confidence should increase by 0.05
	g.Expect(foundResult.Confidence).To(BeNumerically(">", initialConfidence))
}

// TestRecordFeedback_Wrong verifies wrong feedback decreases confidence and flags for review
func TestRecordFeedback_Wrong(t *testing.T) {
	g := NewWithT(t)

	memoryRoot := t.TempDir()

	// Learn a memory
	err := memory.Learn(memory.LearnOpts{
		Message:    "Test memory",
		Project:    "testproject",
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).To(BeNil())

	// Query to get ID
	result, err := memory.Query(memory.QueryOpts{
		Text:       "Test memory",
		Limit:      1,
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).To(BeNil())
	g.Expect(result.Results).To(HaveLen(1))

	embeddingID := result.Results[0].ID
	initialConfidence := result.Results[0].Confidence

	// Record wrong feedback
	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).To(BeNil())
	defer db.Close()

	err = memory.RecordFeedback(db, embeddingID, memory.FeedbackWrong)
	g.Expect(err).To(BeNil())

	// Query again to verify confidence decreased
	result2, err := memory.Query(memory.QueryOpts{
		Text:       "Test memory",
		Limit:      1,
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).To(BeNil())
	g.Expect(result2.Results).To(HaveLen(1))

	// Confidence should decrease by 0.1
	g.Expect(result2.Results[0].Confidence).To(BeNumerically("<", initialConfidence))

	// Verify flagged for review
	flagged, err := memory.ListFlaggedForReview(db)
	g.Expect(err).To(BeNil())
	g.Expect(flagged).To(HaveLen(1))
	g.Expect(flagged[0].ID).To(Equal(embeddingID))
}

// TestRecordFeedback_Unclear verifies unclear feedback flags for rewrite
func TestRecordFeedback_Unclear(t *testing.T) {
	g := NewWithT(t)

	memoryRoot := t.TempDir()

	// Learn a memory
	err := memory.Learn(memory.LearnOpts{
		Message:    "Test memory",
		Project:    "testproject",
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).To(BeNil())

	// Query to get ID
	result, err := memory.Query(memory.QueryOpts{
		Text:       "Test memory",
		Limit:      1,
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).To(BeNil())
	g.Expect(result.Results).To(HaveLen(1))

	embeddingID := result.Results[0].ID

	// Record unclear feedback
	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).To(BeNil())
	defer db.Close()

	err = memory.RecordFeedback(db, embeddingID, memory.FeedbackUnclear)
	g.Expect(err).To(BeNil())

	// Verify flagged for rewrite
	flagged, err := memory.ListFlaggedForRewrite(db)
	g.Expect(err).To(BeNil())
	g.Expect(flagged).To(HaveLen(1))
	g.Expect(flagged[0].ID).To(Equal(embeddingID))
}

// TestGetFeedbackStats verifies feedback statistics retrieval
func TestGetFeedbackStats(t *testing.T) {
	g := NewWithT(t)

	memoryRoot := t.TempDir()

	// Learn a memory
	err := memory.Learn(memory.LearnOpts{
		Message:    "Test memory",
		Project:    "testproject",
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).To(BeNil())

	// Query to get ID
	result, err := memory.Query(memory.QueryOpts{
		Text:       "Test memory",
		Limit:      1,
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).To(BeNil())
	g.Expect(result.Results).To(HaveLen(1))

	embeddingID := result.Results[0].ID

	// Record multiple feedback entries
	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).To(BeNil())
	defer db.Close()

	err = memory.RecordFeedback(db, embeddingID, memory.FeedbackHelpful)
	g.Expect(err).To(BeNil())

	err = memory.RecordFeedback(db, embeddingID, memory.FeedbackHelpful)
	g.Expect(err).To(BeNil())

	err = memory.RecordFeedback(db, embeddingID, memory.FeedbackWrong)
	g.Expect(err).To(BeNil())

	// Get feedback stats
	stats, err := memory.GetFeedbackStats(db, embeddingID)
	g.Expect(err).To(BeNil())
	g.Expect(stats.HelpfulCount).To(Equal(2))
	g.Expect(stats.WrongCount).To(Equal(1))
	g.Expect(stats.UnclearCount).To(Equal(0))
}

// TestSaveLoadLastQueryResults verifies last query caching
func TestSaveLoadLastQueryResults(t *testing.T) {
	g := NewWithT(t)

	memoryRoot := t.TempDir()

	// Create test results
	results := []memory.QueryResult{
		{
			ID:      42,
			Content: "Test memory 1",
			Score:   0.95,
		},
		{
			ID:      43,
			Content: "Test memory 2",
			Score:   0.85,
		},
	}

	// Save results
	err := memory.SaveLastQueryResults(results, "test query", memoryRoot)
	g.Expect(err).To(BeNil())

	// Load results back
	loaded, query, err := memory.LoadLastQueryResults(memoryRoot)
	g.Expect(err).To(BeNil())
	g.Expect(query).To(Equal("test query"))
	g.Expect(loaded).To(HaveLen(2))
	g.Expect(loaded[0].ID).To(Equal(int64(42)))
	g.Expect(loaded[0].Content).To(Equal("Test memory 1"))
	g.Expect(loaded[1].ID).To(Equal(int64(43)))
}

// TestImplicitReAskDetection verifies auto-feedback on repeated similar queries
func TestImplicitReAskDetection(t *testing.T) {
	g := NewWithT(t)

	memoryRoot := t.TempDir()

	// Learn memories
	err := memory.Learn(memory.LearnOpts{
		Message:    "Go concurrency patterns",
		Project:    "testproject",
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).To(BeNil())

	// First query
	result1, err := memory.Query(memory.QueryOpts{
		Text:       "how to use goroutines",
		Limit:      1,
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).To(BeNil())
	g.Expect(result1.Results).To(HaveLen(1))

	// Second query (similar, should trigger re-ask detection)
	result2, err := memory.Query(memory.QueryOpts{
		Text:       "how to use channels and goroutines",
		Limit:      1,
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).To(BeNil())
	g.Expect(result2.Results).To(HaveLen(1))

	// Verify feedback was automatically recorded for first query results
	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).To(BeNil())
	defer db.Close()

	// First result should have wrong feedback recorded
	stats, err := memory.GetFeedbackStats(db, result1.Results[0].ID)
	g.Expect(err).To(BeNil())
	g.Expect(stats.WrongCount).To(BeNumerically(">", 0))
}

// TestPropagateEmbeddingFeedbackToSkills_NoFlaggedSources verifies no propagation when sources are not flagged
func TestPropagateEmbeddingFeedbackToSkills_NoFlaggedSources(t *testing.T) {
	g := NewWithT(t)

	memoryRoot := t.TempDir()

	// Learn some memories
	err := memory.Learn(memory.LearnOpts{
		Message:    "Test memory 1",
		Project:    "testproject",
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).To(BeNil())

	err = memory.Learn(memory.LearnOpts{
		Message:    "Test memory 2",
		Project:    "testproject",
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).To(BeNil())

	// Get DB
	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).To(BeNil())
	defer db.Close()

	// Query to get embedding IDs
	result, err := memory.Query(memory.QueryOpts{
		Text:       "Test memory",
		Limit:      10,
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).To(BeNil())
	g.Expect(len(result.Results)).To(BeNumerically(">=", 2))

	// Create a fake skill referencing these embeddings
	sourceIDs := []int64{result.Results[0].ID, result.Results[1].ID}
	sourceIDsJSON := `[` + fmt.Sprintf("%d,%d", sourceIDs[0], sourceIDs[1]) + `]`

	_, err = db.Exec(`
		INSERT INTO generated_skills (slug, theme, description, content, source_memory_ids, alpha, beta, utility, created_at, updated_at, pruned)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "test-skill", "Test Theme", "Test description", "Test content", sourceIDsJSON, 1.0, 1.0, 0.5, "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z", 0)
	g.Expect(err).To(BeNil())

	// Call PropagateEmbeddingFeedbackToSkills (no flagged sources)
	affected, err := memory.PropagateEmbeddingFeedbackToSkills(db)
	g.Expect(err).To(BeNil())
	g.Expect(affected).To(Equal(0), "should not propagate when no sources are flagged")
}

// TestPropagateEmbeddingFeedbackToSkills_FlaggedSource verifies propagation when source is flagged
func TestPropagateEmbeddingFeedbackToSkills_FlaggedSource(t *testing.T) {
	g := NewWithT(t)

	memoryRoot := t.TempDir()

	// Learn some memories
	err := memory.Learn(memory.LearnOpts{
		Message:    "Test memory 1",
		Project:    "testproject",
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).To(BeNil())

	err = memory.Learn(memory.LearnOpts{
		Message:    "Test memory 2",
		Project:    "testproject",
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).To(BeNil())

	// Get DB
	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).To(BeNil())
	defer db.Close()

	// Query to get embedding IDs
	result, err := memory.Query(memory.QueryOpts{
		Text:       "Test memory",
		Limit:      10,
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).To(BeNil())
	g.Expect(len(result.Results)).To(BeNumerically(">=", 2))

	embID1 := result.Results[0].ID
	embID2 := result.Results[1].ID

	// Flag first embedding for review
	err = memory.RecordFeedback(db, embID1, memory.FeedbackWrong)
	g.Expect(err).To(BeNil())

	// Create a skill referencing these embeddings
	sourceIDsJSON := fmt.Sprintf(`[%d,%d]`, embID1, embID2)

	_, err = db.Exec(`
		INSERT INTO generated_skills (slug, theme, description, content, source_memory_ids, alpha, beta, utility, created_at, updated_at, pruned)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "test-skill", "Test Theme", "Test description", "Test content", sourceIDsJSON, 1.0, 1.0, 0.5, "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z", 0)
	g.Expect(err).To(BeNil())

	// Call PropagateEmbeddingFeedbackToSkills
	affected, err := memory.PropagateEmbeddingFeedbackToSkills(db)
	g.Expect(err).To(BeNil())
	g.Expect(affected).To(Equal(1), "should propagate to 1 skill")

	// Verify skill beta was incremented and utility recomputed
	var beta, utility float64
	var propagatedAt string
	err = db.QueryRow(`SELECT beta, utility, feedback_propagated_at FROM generated_skills WHERE slug = ?`, "test-skill").Scan(&beta, &utility, &propagatedAt)
	g.Expect(err).To(BeNil())
	g.Expect(beta).To(Equal(2.0), "beta should be incremented by 1.0")
	g.Expect(propagatedAt).ToNot(BeEmpty(), "feedback_propagated_at should be set")

	// Utility should be recomputed (with beta=2, utility should be lower than initial 0.5)
	g.Expect(utility).To(BeNumerically("<", 0.5), "utility should decrease when beta increases")
}

// TestPropagateEmbeddingFeedbackToSkills_AlreadyPropagated verifies no re-propagation when already propagated
func TestPropagateEmbeddingFeedbackToSkills_AlreadyPropagated(t *testing.T) {
	g := NewWithT(t)

	memoryRoot := t.TempDir()

	// Learn some memories
	err := memory.Learn(memory.LearnOpts{
		Message:    "Test memory 1",
		Project:    "testproject",
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).To(BeNil())

	// Get DB
	db, err := memory.InitDBForTest(memoryRoot)
	g.Expect(err).To(BeNil())
	defer db.Close()

	// Query to get embedding ID
	result, err := memory.Query(memory.QueryOpts{
		Text:       "Test memory",
		Limit:      1,
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).To(BeNil())
	g.Expect(result.Results).To(HaveLen(1))

	embID := result.Results[0].ID

	// Flag embedding for review
	err = memory.RecordFeedback(db, embID, memory.FeedbackWrong)
	g.Expect(err).To(BeNil())

	// Create a skill with feedback already propagated
	sourceIDsJSON := fmt.Sprintf(`[%d]`, embID)
	propagatedTimestamp := "2024-01-01T12:00:00Z"

	_, err = db.Exec(`
		INSERT INTO generated_skills (slug, theme, description, content, source_memory_ids, alpha, beta, utility, created_at, updated_at, pruned, feedback_propagated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "test-skill", "Test Theme", "Test description", "Test content", sourceIDsJSON, 1.0, 2.0, 0.3, "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z", 0, propagatedTimestamp)
	g.Expect(err).To(BeNil())

	// Call PropagateEmbeddingFeedbackToSkills
	affected, err := memory.PropagateEmbeddingFeedbackToSkills(db)
	g.Expect(err).To(BeNil())
	g.Expect(affected).To(Equal(0), "should not re-propagate when already propagated")

	// Verify beta hasn't changed
	var beta float64
	var propagatedAt string
	err = db.QueryRow(`SELECT beta, feedback_propagated_at FROM generated_skills WHERE slug = ?`, "test-skill").Scan(&beta, &propagatedAt)
	g.Expect(err).To(BeNil())
	g.Expect(beta).To(Equal(2.0), "beta should remain unchanged")
	g.Expect(propagatedAt).To(Equal(propagatedTimestamp), "propagation timestamp should remain unchanged")
}
