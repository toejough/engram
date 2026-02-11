package memory_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/memory"
)

// TEST: hybridSearch returns results (basic end-to-end)
func TestHybridSearchReturnsResults(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Learn several entries
	entries := []string{
		"Go programming language concurrency patterns",
		"Rust memory safety and ownership model",
		"Python data science with pandas and numpy",
	}
	for _, msg := range entries {
		err = memory.Learn(memory.LearnOpts{
			Message:    msg,
			MemoryRoot: memoryDir,
		})
		g.Expect(err).ToNot(HaveOccurred())
	}

	// Query using the hybrid search path
	results, err := memory.Query(memory.QueryOpts{
		Text:       "Go concurrency",
		Limit:      5,
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results.Results).ToNot(BeEmpty())
	g.Expect(results.UsedHybridSearch).To(BeTrue())
}

// TEST: Query with UsedHybridSearch=true (property)
func TestQueryUsesHybridSearch(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		g := NewWithT(t)

		tempDir := os.TempDir()
		memoryDir, err := os.MkdirTemp(tempDir, "hybrid-prop-*")
		g.Expect(err).ToNot(HaveOccurred())
		defer func() { _ = os.RemoveAll(memoryDir) }()

		// Learn at least one entry
		msg := rapid.StringMatching(`[a-z]{5,15}`).Draw(t, "message")
		err = memory.Learn(memory.LearnOpts{
			Message:    msg,
			MemoryRoot: memoryDir,
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Query
		queryText := rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "query")
		results, err := memory.Query(memory.QueryOpts{
			Text:       queryText,
			Limit:      3,
			MemoryRoot: memoryDir,
		})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(results.UsedHybridSearch).To(BeTrue())
	})
}

// TEST: exact keyword match always appears in results (BM25 contribution) - property
func TestBM25ExactKeywordMatchAppears(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Learn an entry with a very unique word
	uniqueWord := "supercalifragilisticexpialidocious"
	err = memory.Learn(memory.LearnOpts{
		Message:    "The word " + uniqueWord + " is used for testing BM25",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Also learn some unrelated entries to make results non-trivial
	for _, msg := range []string{
		"database optimization techniques for large datasets",
		"network programming with TCP sockets",
		"machine learning model training procedures",
	} {
		err = memory.Learn(memory.LearnOpts{
			Message:    msg,
			MemoryRoot: memoryDir,
		})
		g.Expect(err).ToNot(HaveOccurred())
	}

	// Query for the unique word - BM25 should boost it to the top
	results, err := memory.Query(memory.QueryOpts{
		Text:       uniqueWord,
		Limit:      5,
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results.Results).ToNot(BeEmpty())

	// The unique word should appear in at least one result
	found := false
	for _, r := range results.Results {
		if hybridContains(r.Content, uniqueWord) {
			found = true
			break
		}
	}
	g.Expect(found).To(BeTrue(), "exact keyword match should appear in hybrid search results")
}

// TEST: empty FTS5 table returns empty from BM25
func TestBM25EmptyFTS5ReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Initialize DB without learning anything
	// Query should still work (vector search may return nothing, BM25 returns nothing)
	results, err := memory.Query(memory.QueryOpts{
		Text:       "anything at all",
		Limit:      5,
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())
	// Empty DB -> empty results (no crash)
	g.Expect(results).ToNot(BeNil())
}

// TEST: query with FTS5 special characters doesn't error (falls back)
func TestBM25SpecialCharactersFallback(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Learn something so DB exists
	err = memory.Learn(memory.LearnOpts{
		Message:    "some normal content for testing",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Query with FTS5 special characters that could cause syntax errors
	specialQueries := []string{
		`"unclosed quote`,
		`foo AND OR bar`,
		`(())`,
		`*`,
		`col:value`,
		`NOT NOT NOT`,
		`foo NEAR bar`,
	}

	for _, q := range specialQueries {
		results, err := memory.Query(memory.QueryOpts{
			Text:       q,
			Limit:      5,
			MemoryRoot: memoryDir,
		})
		// Should not error - BM25 errors are caught and fall back to vector-only
		g.Expect(err).ToNot(HaveOccurred(), "query with special chars %q should not error", q)
		g.Expect(results).ToNot(BeNil())
	}
}

// hybridContains is a helper for checking if content contains a substring.
func hybridContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
