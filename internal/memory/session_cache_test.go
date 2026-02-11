package memory_test

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/memory"
)

// TEST-850: ONNX session is cached across multiple Query calls
// traces: ISSUE-48
func TestONNXSessionCachedAcrossQueries(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Clear session cache to ensure test isolation
	memory.ClearSessionCache()

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Create test content via Learn
	for _, msg := range []string{"First memory entry", "Second memory entry", "Third memory entry"} {
		err = memory.Learn(memory.LearnOpts{Message: msg, MemoryRoot: memoryDir})
		g.Expect(err).ToNot(HaveOccurred())
	}

	// Clear session cache after Learn (which also uses ONNX) to test Query caching
	memory.ClearSessionCache()

	// First query (uses default ModelDir with pre-downloaded model)
	opts1 := memory.QueryOpts{
		Text:       "first",
		MemoryRoot: memoryDir,
	}
	result1, err := memory.Query(opts1)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result1.SessionCreatedNew).To(BeTrue(), "first query should create new session")

	// Second query - should reuse session
	opts2 := memory.QueryOpts{
		Text:       "second",
		MemoryRoot: memoryDir,
	}
	result2, err := memory.Query(opts2)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result2.SessionCreatedNew).To(BeFalse(), "second query should reuse existing session")
	g.Expect(result2.SessionReused).To(BeTrue(), "second query should indicate session was reused")
}

// TEST-851: ONNX session initialization counter increments only once
// traces: ISSUE-48
func TestONNXSessionInitializedOnce(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Clear session cache to ensure test isolation
	memory.ClearSessionCache()

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	err = memory.Learn(memory.LearnOpts{Message: "Memory content", MemoryRoot: memoryDir})
	g.Expect(err).ToNot(HaveOccurred())

	// Clear session cache after Learn to isolate Query session counting
	memory.ClearSessionCache()

	// Get initial session init count
	initialCount := memory.GetSessionInitCount()

	// Run multiple queries (uses default ModelDir with pre-downloaded model)
	for i := 0; i < 5; i++ {
		opts := memory.QueryOpts{
			Text:       "test query",
			MemoryRoot: memoryDir,
		}
		_, err := memory.Query(opts)
		g.Expect(err).ToNot(HaveOccurred())
	}

	// Session should have been initialized exactly once
	finalCount := memory.GetSessionInitCount()
	g.Expect(finalCount-initialCount).To(Equal(1), "session should be initialized exactly once")
}

// TEST-852: ONNX session caching is thread-safe
// traces: ISSUE-48
func TestONNXSessionCachingThreadSafe(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Clear session cache to ensure test isolation
	memory.ClearSessionCache()

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	err = memory.Learn(memory.LearnOpts{Message: "Test content for concurrency", MemoryRoot: memoryDir})
	g.Expect(err).ToNot(HaveOccurred())

	// Clear session cache after Learn to isolate Query session counting
	memory.ClearSessionCache()

	// Run queries concurrently (uses default ModelDir with pre-downloaded model)
	const numGoroutines = 10
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)
	sessionCreatedCount := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			opts := memory.QueryOpts{
				Text:       "concurrent test",
				MemoryRoot: memoryDir,
			}
			result, err := memory.Query(opts)
			if err != nil {
				errors <- err
				return
			}
			sessionCreatedCount <- result.SessionCreatedNew
		}(i)
	}

	wg.Wait()
	close(errors)
	close(sessionCreatedCount)

	// Check for errors
	for err := range errors {
		g.Expect(err).ToNot(HaveOccurred())
	}

	// Count how many times session was created
	createdCount := 0
	for created := range sessionCreatedCount {
		if created {
			createdCount++
		}
	}

	// Session should be created exactly once despite concurrent access
	g.Expect(createdCount).To(Equal(1), "session should be created exactly once even with concurrent queries")
}

// TEST-853: Session cache survives across different memory roots
// traces: ISSUE-48
func TestONNXSessionCacheSurvivesAcrossMemoryRoots(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Clear session cache to ensure test isolation
	memory.ClearSessionCache()

	tempDir := t.TempDir()
	memoryDir1 := filepath.Join(tempDir, "memory1")
	memoryDir2 := filepath.Join(tempDir, "memory2")

	for _, dir := range []string{memoryDir1, memoryDir2} {
		err := os.MkdirAll(dir, 0755)
		g.Expect(err).ToNot(HaveOccurred())
	}

	// Create content in both memory directories via Learn
	for _, memDir := range []string{memoryDir1, memoryDir2} {
		err := memory.Learn(memory.LearnOpts{Message: "Test memory", MemoryRoot: memDir})
		g.Expect(err).ToNot(HaveOccurred())
	}

	// Clear session cache after Learn to isolate Query session counting
	memory.ClearSessionCache()

	// Query first memory root (uses default ModelDir with pre-downloaded model)
	opts1 := memory.QueryOpts{
		Text:       "test",
		MemoryRoot: memoryDir1,
	}
	result1, err := memory.Query(opts1)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result1.SessionCreatedNew).To(BeTrue())

	// Query second memory root - session should be reused
	opts2 := memory.QueryOpts{
		Text:       "test",
		MemoryRoot: memoryDir2,
	}
	result2, err := memory.Query(opts2)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result2.SessionReused).To(BeTrue(), "session should be reused across different memory roots")
}

// TEST-854: Session cache can be cleared for testing
// traces: ISSUE-48
func TestONNXSessionCacheCanBeCleared(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Clear session cache to ensure test isolation
	memory.ClearSessionCache()

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	err = memory.Learn(memory.LearnOpts{Message: "Cache test content", MemoryRoot: memoryDir})
	g.Expect(err).ToNot(HaveOccurred())

	// Clear session cache after Learn to isolate Query session counting
	memory.ClearSessionCache()

	// First query creates session (uses default ModelDir with pre-downloaded model)
	opts := memory.QueryOpts{
		Text:       "test",
		MemoryRoot: memoryDir,
	}
	result1, err := memory.Query(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result1.SessionCreatedNew).To(BeTrue())

	// Clear the session cache
	memory.ClearSessionCache()

	// Next query should create new session
	result2, err := memory.Query(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result2.SessionCreatedNew).To(BeTrue(), "query after cache clear should create new session")
}

// TEST-855: Session cache reduces query time significantly
// traces: ISSUE-48
func TestONNXSessionCacheReducesQueryTime(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping timing test in short mode")
	}

	g := NewWithT(t)

	// Clear session cache to ensure test isolation
	memory.ClearSessionCache()

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	err = memory.Learn(memory.LearnOpts{Message: "Timing test content", MemoryRoot: memoryDir})
	g.Expect(err).ToNot(HaveOccurred())

	// Clear session cache after Learn to isolate Query timing
	memory.ClearSessionCache()

	// Uses default ModelDir with pre-downloaded model
	opts := memory.QueryOpts{
		Text:       "timing",
		MemoryRoot: memoryDir,
	}

	// First query (cold start)
	result1, err := memory.Query(opts)
	g.Expect(err).ToNot(HaveOccurred())
	firstQueryTime := result1.QueryDuration

	// Second query (warm cache)
	result2, err := memory.Query(opts)
	g.Expect(err).ToNot(HaveOccurred())
	secondQueryTime := result2.QueryDuration

	// Second query should be significantly faster (at least 2x)
	g.Expect(secondQueryTime).To(BeNumerically("<", firstQueryTime/2),
		"cached query should be at least 2x faster than cold start")
}

// TEST-856: Property-based test for session reuse consistency
// traces: ISSUE-48
func TestONNXSessionReusePropertyBased(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(t *rapid.T) {
		g := NewWithT(t)

		// Clear session cache for each property test iteration
		memory.ClearSessionCache()

		suffix := rapid.StringMatching(`[a-zA-Z0-9]{8}`).Draw(t, "suffix")
		tempDir := os.TempDir()
		memoryDir := filepath.Join(tempDir, "session-cache-test-"+suffix)
		defer func() {
			_ = os.RemoveAll(memoryDir)
		}()

		err := os.MkdirAll(memoryDir, 0755)
		g.Expect(err).ToNot(HaveOccurred())

		err = memory.Learn(memory.LearnOpts{Message: "Property test content", MemoryRoot: memoryDir})
		g.Expect(err).ToNot(HaveOccurred())

		// Clear session cache after Learn to isolate Query session counting
		memory.ClearSessionCache()

		// Generate random number of queries (uses default ModelDir with pre-downloaded model)
		numQueries := rapid.IntRange(2, 10).Draw(t, "numQueries")

		var results []*memory.QueryResults
		for i := 0; i < numQueries; i++ {
			opts := memory.QueryOpts{
				Text:       rapid.StringMatching(`[a-zA-Z]{5,10}`).Draw(t, "query"),
				MemoryRoot: memoryDir,
			}
			result, err := memory.Query(opts)
			g.Expect(err).ToNot(HaveOccurred())
			results = append(results, result)
		}

		// Property: First query creates session, all others reuse it
		g.Expect(results[0].SessionCreatedNew).To(BeTrue())
		for i := 1; i < len(results); i++ {
			g.Expect(results[i].SessionReused).To(BeTrue(),
				"query %d should reuse session", i)
		}
	})
}

// TEST-857: Session cache handles model path changes
// traces: ISSUE-48
func TestONNXSessionCacheHandlesModelPathChanges(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Clear session cache to ensure test isolation
	memory.ClearSessionCache()

	// Get default model path (pre-downloaded)
	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())
	defaultModelPath := filepath.Join(homeDir, ".claude", "models", "e5-small-v2.onnx")

	// Skip if model not available
	if _, err := os.Stat(defaultModelPath); os.IsNotExist(err) {
		t.Skip("ONNX model not available at default location")
	}

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	modelDir1 := filepath.Join(tempDir, "models1")
	modelDir2 := filepath.Join(tempDir, "models2")

	for _, dir := range []string{memoryDir, modelDir1, modelDir2} {
		err := os.MkdirAll(dir, 0755)
		g.Expect(err).ToNot(HaveOccurred())
	}

	// Copy model to both test directories to avoid network download
	modelData, err := os.ReadFile(defaultModelPath)
	g.Expect(err).ToNot(HaveOccurred())
	err = os.WriteFile(filepath.Join(modelDir1, "e5-small-v2.onnx"), modelData, 0644)
	g.Expect(err).ToNot(HaveOccurred())
	err = os.WriteFile(filepath.Join(modelDir2, "e5-small-v2.onnx"), modelData, 0644)
	g.Expect(err).ToNot(HaveOccurred())

	err = memory.Learn(memory.LearnOpts{Message: "Model path test", MemoryRoot: memoryDir})
	g.Expect(err).ToNot(HaveOccurred())

	// Clear session cache after Learn to isolate Query session counting
	memory.ClearSessionCache()

	// Query with first model directory
	opts1 := memory.QueryOpts{
		Text:       "test",
		MemoryRoot: memoryDir,
		ModelDir:   modelDir1,
	}
	result1, err := memory.Query(opts1)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result1.SessionCreatedNew).To(BeTrue())

	// Query with different model directory - should create new session
	opts2 := memory.QueryOpts{
		Text:       "test",
		MemoryRoot: memoryDir,
		ModelDir:   modelDir2,
	}
	result2, err := memory.Query(opts2)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result2.SessionCreatedNew).To(BeTrue(), "changing model path should create new session")

	// Query back to first model directory - should reuse first session
	result3, err := memory.Query(opts1)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result3.SessionReused).To(BeTrue(), "should reuse session for same model path")
}
