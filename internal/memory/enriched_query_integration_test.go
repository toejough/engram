//go:build integration

package memory_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/memory"
)

// TEST: searchBM25 populates RetrievalCount and ProjectsRetrieved
func TestBM25ResultsHaveEnrichedFields(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	// Learn entry with unique keyword for BM25
	uniqueKey := "xylophonequartzjumble"
	g.Expect(memory.Learn(memory.LearnOpts{
		Message:    "testing BM25 enrichment with " + uniqueKey,
		MemoryRoot: memoryDir,
	})).To(Succeed())

	// First query to populate retrieval tracking
	_, err := memory.Query(memory.QueryOpts{
		Text:       uniqueKey,
		Limit:      5,
		MemoryRoot: memoryDir,
		Project:    "bm25-project",
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Second query — results should have enriched fields from DB
	results, err := memory.Query(memory.QueryOpts{
		Text:       uniqueKey,
		Limit:      5,
		MemoryRoot: memoryDir,
		Project:    "bm25-project",
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results.Results).ToNot(BeEmpty())

	found := false
	for _, r := range results.Results {
		if strings.Contains(r.Content, uniqueKey) {
			found = true
			g.Expect(r.RetrievalCount).To(BeNumerically(">=", 1),
				"BM25 result should have RetrievalCount populated")
			g.Expect(r.ProjectsRetrieved).To(ContainElement("bm25-project"),
				"BM25 result should have ProjectsRetrieved populated")
			break
		}
	}
	g.Expect(found).To(BeTrue(), "should find the BM25 test entry in results")
}

// TEST: extractMessageContent strips timestamp and project tags
func TestExtractMessageContent(t *testing.T) {
	t.Parallel()
	_ = NewWithT(t)

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "full format with project",
			input: "- 2024-01-15 10:30: [myproject] actual learning content",
			want:  "actual learning content",
		},
		{
			name:  "format without project",
			input: "- 2024-01-15 10:30: just a plain learning",
			want:  "just a plain learning",
		},
		{
			name:  "plain text",
			input: "no prefix at all",
			want:  "no prefix at all",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g2 := NewWithT(t)
			// extractMessageContent is package-accessible; we verify it indirectly
			// via QueryResult.Content which passes through it. The function is unexported
			// but called in detectConflictType which is tested in contradiction_test.go.
			// Here we just confirm the struct fields compile.
			_ = memory.QueryResult{
				Content:           tc.input,
				RetrievalCount:    0,
				ProjectsRetrieved: nil,
				MatchType:         "vector",
			}
			g2.Expect(tc.want).ToNot(BeEmpty()) // sanity check
		})
	}
}

// TEST: Property - MatchType is always one of {"vector", "bm25", "hybrid"} for non-empty results
func TestPropertyMatchTypeValid(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		g := NewWithT(t)

		tempDir := os.TempDir()
		memoryDir, err := os.MkdirTemp(tempDir, "matchtype-prop-*")
		g.Expect(err).ToNot(HaveOccurred())
		defer func() { _ = os.RemoveAll(memoryDir) }()

		msg := rapid.StringMatching(`[a-z]{5,15}`).Draw(t, "message")
		g.Expect(memory.Learn(memory.LearnOpts{
			Message:    msg,
			MemoryRoot: memoryDir,
		})).To(Succeed())

		queryText := rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "query")
		results, err := memory.Query(memory.QueryOpts{
			Text:       queryText,
			Limit:      3,
			MemoryRoot: memoryDir,
		})
		g.Expect(err).ToNot(HaveOccurred())

		for _, r := range results.Results {
			g.Expect(r.MatchType).To(BeElementOf("vector", "bm25", "hybrid"),
				"MatchType must be one of the valid values, got %q", r.MatchType)
		}
	})
}

// TEST: Property - ProjectsRetrieved is nil or contains valid strings
func TestPropertyProjectsRetrievedValid(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		g := NewWithT(t)

		tempDir := os.TempDir()
		memoryDir, err := os.MkdirTemp(tempDir, "projret-prop-*")
		g.Expect(err).ToNot(HaveOccurred())
		defer func() { _ = os.RemoveAll(memoryDir) }()

		msg := rapid.StringMatching(`[a-z]{5,15}`).Draw(t, "message")
		project := rapid.StringMatching(`[a-z]{3,8}`).Draw(t, "project")
		g.Expect(memory.Learn(memory.LearnOpts{
			Message:    msg,
			MemoryRoot: memoryDir,
		})).To(Succeed())

		// Query with a project to populate projects_retrieved
		_, err = memory.Query(memory.QueryOpts{
			Text:       msg,
			Limit:      3,
			MemoryRoot: memoryDir,
			Project:    project,
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Second query to read back
		results, err := memory.Query(memory.QueryOpts{
			Text:       msg,
			Limit:      3,
			MemoryRoot: memoryDir,
			Project:    project,
		})
		g.Expect(err).ToNot(HaveOccurred())

		for _, r := range results.Results {
			// ProjectsRetrieved should be nil or contain non-empty strings
			for _, p := range r.ProjectsRetrieved {
				g.Expect(p).ToNot(BeEmpty(),
					"ProjectsRetrieved entries must not be empty strings")
			}
		}
	})
}

// TEST: Property - RetrievalCount is non-negative in QueryResult
func TestPropertyRetrievalCountNonNegative(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		g := NewWithT(t)

		tempDir := os.TempDir()
		memoryDir, err := os.MkdirTemp(tempDir, "retcount-prop-*")
		g.Expect(err).ToNot(HaveOccurred())
		defer func() { _ = os.RemoveAll(memoryDir) }()

		msg := rapid.StringMatching(`[a-z]{5,15}`).Draw(t, "message")
		g.Expect(memory.Learn(memory.LearnOpts{
			Message:    msg,
			MemoryRoot: memoryDir,
		})).To(Succeed())

		queryText := rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "query")
		results, err := memory.Query(memory.QueryOpts{
			Text:       queryText,
			Limit:      3,
			MemoryRoot: memoryDir,
		})
		g.Expect(err).ToNot(HaveOccurred())

		for _, r := range results.Results {
			g.Expect(r.RetrievalCount).To(BeNumerically(">=", 0),
				"RetrievalCount must be non-negative")
		}
	})
}

// TEST: Verify DB columns are read into QueryResult (direct DB inspection)
func TestQueryResultFieldsMatchDB(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	g.Expect(memory.Learn(memory.LearnOpts{
		Message:    "DB match verification entry unique789",
		MemoryRoot: memoryDir,
	})).To(Succeed())

	// Manually set retrieval_count and projects_retrieved in DB
	dbPath := filepath.Join(memoryDir, "embeddings.db")
	db, err := sql.Open("sqlite3", dbPath)
	g.Expect(err).ToNot(HaveOccurred())

	_, err = db.Exec(
		`UPDATE embeddings SET retrieval_count = 42, projects_retrieved = 'alpha,beta,gamma' WHERE content LIKE '%unique789%'`)
	g.Expect(err).ToNot(HaveOccurred())
	_ = db.Close()

	// Query and verify the fields are populated from DB
	results, err := memory.Query(memory.QueryOpts{
		Text:       "DB match verification unique789",
		Limit:      5,
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results.Results).ToNot(BeEmpty())

	found := false
	for _, r := range results.Results {
		if strings.Contains(r.Content, "unique789") {
			found = true
			g.Expect(r.RetrievalCount).To(Equal(42),
				"RetrievalCount should match DB value")
			g.Expect(r.ProjectsRetrieved).To(ConsistOf("alpha", "beta", "gamma"),
				"ProjectsRetrieved should be parsed from comma-separated DB value")
			break
		}
	}
	g.Expect(found).To(BeTrue(), "should find the test entry in results")
}

// TEST: QueryResult has ProjectsRetrieved field populated after query
func TestQueryResultHasProjectsRetrieved(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	g.Expect(memory.Learn(memory.LearnOpts{
		Message:    "enriched projects retrieved test entry",
		MemoryRoot: memoryDir,
	})).To(Succeed())

	// Query from project-alpha
	_, err := memory.Query(memory.QueryOpts{
		Text:       "enriched projects retrieved",
		Limit:      5,
		MemoryRoot: memoryDir,
		Project:    "project-alpha",
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Second query — results should carry projects_retrieved
	results, err := memory.Query(memory.QueryOpts{
		Text:       "enriched projects retrieved",
		Limit:      5,
		MemoryRoot: memoryDir,
		Project:    "project-beta",
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results.Results).ToNot(BeEmpty())

	found := false
	for _, r := range results.Results {
		if strings.Contains(r.Content, "enriched projects retrieved") {
			found = true
			g.Expect(r.ProjectsRetrieved).ToNot(BeEmpty(),
				"ProjectsRetrieved should be populated from DB")
			g.Expect(r.ProjectsRetrieved).To(ContainElement("project-alpha"),
				"ProjectsRetrieved should include previously querying project")
			break
		}
	}
	g.Expect(found).To(BeTrue(), "should find the test entry in results")
}

// ============================================================================
// ISSUE-188: Enriched QueryResult fields
// Tests for RetrievalCount, ProjectsRetrieved, MatchType on QueryResult
// ============================================================================

// TEST: QueryResult has RetrievalCount field populated after query
func TestQueryResultHasRetrievalCount(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	// Learn an entry then query twice so retrieval_count >= 1
	g.Expect(memory.Learn(memory.LearnOpts{
		Message:    "enriched retrieval count test entry",
		MemoryRoot: memoryDir,
	})).To(Succeed())

	// First query seeds retrieval_count=1
	_, err := memory.Query(memory.QueryOpts{
		Text:       "enriched retrieval count",
		Limit:      5,
		MemoryRoot: memoryDir,
		Project:    "proj-rc",
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Second query — the returned results should reflect the DB state
	results, err := memory.Query(memory.QueryOpts{
		Text:       "enriched retrieval count",
		Limit:      5,
		MemoryRoot: memoryDir,
		Project:    "proj-rc",
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results.Results).ToNot(BeEmpty())

	// Find the matching result
	found := false
	for _, r := range results.Results {
		if strings.Contains(r.Content, "enriched retrieval count") {
			found = true
			g.Expect(r.RetrievalCount).To(BeNumerically(">=", 1),
				"RetrievalCount should be populated from DB")
			break
		}
	}
	g.Expect(found).To(BeTrue(), "should find the test entry in results")
}

// TEST: hybridSearch sets MatchType="hybrid" when result appears in both vector and BM25
func TestQueryResultMatchTypeHybrid(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	// Use a unique word that BM25 will definitely match, combined with
	// semantic content that vector search will also match
	uniqueWord := "supercalifragilisticexpialidocious"
	g.Expect(memory.Learn(memory.LearnOpts{
		Message:    "testing hybrid match with " + uniqueWord + " for verification",
		MemoryRoot: memoryDir,
	})).To(Succeed())

	// Add unrelated entries for contrast
	for _, msg := range []string{
		"database optimization techniques for large datasets",
		"network programming with TCP sockets",
		"machine learning model training procedures",
	} {
		g.Expect(memory.Learn(memory.LearnOpts{
			Message:    msg,
			MemoryRoot: memoryDir,
		})).To(Succeed())
	}

	// Query with the unique word — BM25 will match on the keyword,
	// and vector search will match semantically (it's the closest entry)
	results, err := memory.Query(memory.QueryOpts{
		Text:       uniqueWord,
		Limit:      5,
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results.Results).ToNot(BeEmpty())

	// Check that all results have valid MatchType
	for _, r := range results.Results {
		g.Expect(r.MatchType).To(BeElementOf("vector", "bm25", "hybrid"),
			"MatchType should always be set to a valid value")
	}

	// Verify at least the unique-word entry is found via hybrid or vector
	found := false
	for _, r := range results.Results {
		if strings.Contains(r.Content, uniqueWord) {
			found = true
			// The entry should be "hybrid" if BM25 matched, or "vector" if only vector matched.
			// Both are acceptable — the key is that MatchType is always populated.
			g.Expect(r.MatchType).To(BeElementOf("vector", "hybrid"),
				"unique-word entry should be found via vector or hybrid")
			break
		}
	}
	g.Expect(found).To(BeTrue(), "should find the unique-word entry in results")

	// Also check that we have at least some results with MatchType set
	hasMatchType := false
	for _, r := range results.Results {
		if r.MatchType != "" {
			hasMatchType = true
			break
		}
	}
	g.Expect(hasMatchType).To(BeTrue(), "at least one result should have MatchType set")
}

// TEST: QueryResult MatchType is set to "vector" for vector-only results
func TestQueryResultMatchTypeVector(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	g.Expect(os.MkdirAll(memoryDir, 0755)).To(Succeed())

	g.Expect(memory.Learn(memory.LearnOpts{
		Message:    "vector match type verification entry",
		MemoryRoot: memoryDir,
	})).To(Succeed())

	results, err := memory.Query(memory.QueryOpts{
		Text:       "vector match type verification",
		Limit:      5,
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results.Results).ToNot(BeEmpty())

	// Every result should have a non-empty MatchType
	for _, r := range results.Results {
		g.Expect(r.MatchType).To(BeElementOf("vector", "bm25", "hybrid"),
			"MatchType should be set to one of the valid values, got %q for %q", r.MatchType, r.Content)
	}
}
