//go:build integration

package memory_test

import (
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/projctl/internal/memory"
	"pgregory.net/rapid"
)

// Property: higher threshold always returns <= results than lower threshold
func TestHigherThresholdFewerResults(t *testing.T) {
	// Set up shared test data
	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	entries := []string{
		"Go concurrency patterns with goroutines",
		"Rust memory safety ownership borrowing",
		"Python data science numpy pandas",
		"JavaScript React frontend development",
		"Database SQL optimization indexing",
	}
	for _, msg := range entries {
		err := memory.Learn(memory.LearnOpts{
			Message:    msg,
			MemoryRoot: memoryDir,
		})
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}
	}

	rapid.Check(t, func(t *rapid.T) {
		g := NewWithT(t)

		lowThreshold := rapid.Float64Range(0.0, 0.5).Draw(t, "lowThreshold")
		highThreshold := rapid.Float64Range(lowThreshold, 1.0).Draw(t, "highThreshold")

		lowResults, err := memory.Query(memory.QueryOpts{
			Text:       "programming languages",
			Limit:      10,
			MinScore:   lowThreshold,
			MemoryRoot: memoryDir,
		})
		g.Expect(err).ToNot(HaveOccurred())

		highResults, err := memory.Query(memory.QueryOpts{
			Text:       "programming languages",
			Limit:      10,
			MinScore:   highThreshold,
			MemoryRoot: memoryDir,
		})
		g.Expect(err).ToNot(HaveOccurred())

		g.Expect(len(highResults.Results)).To(BeNumerically("<=", len(lowResults.Results)))
	})
}

// Test that MinScore only returns results meeting threshold
func TestMinScoreFiltersCorrectly(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	// Learn entries
	entries := []string{
		"Go concurrency with goroutines and channels for parallel processing",
		"Python decorators and metaclasses for advanced OOP",
		"JavaScript async await promises for asynchronous programming",
	}
	for _, msg := range entries {
		err := memory.Learn(memory.LearnOpts{
			Message:    msg,
			MemoryRoot: memoryDir,
		})
		g.Expect(err).ToNot(HaveOccurred())
	}

	// Query with moderate MinScore
	threshold := 0.01 // Very low but non-zero to validate filtering works
	results, err := memory.Query(memory.QueryOpts{
		Text:       "Go goroutines",
		Limit:      10,
		MinScore:   threshold,
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// All returned results should meet the threshold
	for _, r := range results.Results {
		g.Expect(r.Score).To(BeNumerically(">=", threshold))
	}
}

// Test that very high MinScore filters out poor matches
func TestMinScoreHighFiltersResults(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	// Learn an entry
	err := memory.Learn(memory.LearnOpts{
		Message:    "Go concurrency patterns with goroutines and channels",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Query with extremely high threshold - should filter everything
	results, err := memory.Query(memory.QueryOpts{
		Text:       "completely unrelated quantum physics topic",
		Limit:      10,
		MinScore:   0.99,
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results.Results).To(BeEmpty())
}

// Test that MinScore=0.0 (default) returns all results - backward compatible
func TestMinScoreZeroReturnsAll(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")

	// Learn entries
	for _, msg := range []string{
		"Go concurrency patterns with goroutines",
		"Python machine learning with tensorflow",
		"Rust ownership and borrowing rules",
	} {
		err := memory.Learn(memory.LearnOpts{
			Message:    msg,
			MemoryRoot: memoryDir,
		})
		g.Expect(err).ToNot(HaveOccurred())
	}

	// Query with MinScore=0 (default)
	results, err := memory.Query(memory.QueryOpts{
		Text:       "programming languages",
		Limit:      10,
		MinScore:   0.0,
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results.Results).ToNot(BeEmpty())
}
