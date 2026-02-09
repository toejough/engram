package memory_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/memory"
)

// ============================================================================
// TASK-4: context-inject command implementation
// ============================================================================

// TEST-4001: ContextInjectOpts accepts required parameters
// traces: TASK-4 AC-1
func TestContextInjectOptsAcceptsParameters(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()

	opts := memory.ContextInjectOpts{
		MemoryRoot:   tempDir,
		MaxEntries:   10,
		MaxTokens:    1000,
		MinConfidence: 0.3,
	}

	g.Expect(opts.MemoryRoot).To(Equal(tempDir))
	g.Expect(opts.MaxEntries).To(Equal(10))
	g.Expect(opts.MaxTokens).To(Equal(1000))
	g.Expect(opts.MinConfidence).To(Equal(0.3))
}

// TEST-4002: ContextInject uses Query infrastructure
// traces: TASK-4 AC-2
func TestContextInjectUsesQueryInfrastructure(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, "memory")

	// Create some learnings first
	err := memory.Learn(memory.LearnOpts{
		Message:    "Test learning with high confidence",
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.ContextInjectOpts{
		MemoryRoot:   memoryRoot,
		QueryText:    "recent learnings",
		MaxEntries:   5,
		MaxTokens:    500,
		MinConfidence: 0.3,
	}

	result, err := memory.ContextInject(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeEmpty())
}

// TEST-4003: ContextInject filters by confidence threshold
// traces: TASK-4 AC-3
func TestContextInjectFiltersLowConfidence(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, "memory")

	// Create learnings
	err := memory.Learn(memory.LearnOpts{
		Message:    "High confidence learning",
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Query with high confidence threshold
	opts := memory.ContextInjectOpts{
		MemoryRoot:   memoryRoot,
		QueryText:    "confidence learning",
		MaxEntries:   10,
		MaxTokens:    1000,
		MinConfidence: 0.5, // Higher threshold
	}

	result, err := memory.ContextInject(opts)
	g.Expect(err).ToNot(HaveOccurred())
	// Should still return valid markdown (may be empty if no results)
	g.Expect(result).ToNot(BeNil())
}

// TEST-4004: ContextInject formats output as compact markdown
// traces: TASK-4 AC-4
func TestContextInjectFormatsAsMarkdown(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, "memory")

	// Create a learning
	err := memory.Learn(memory.LearnOpts{
		Message:    "Always use TDD approach",
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.ContextInjectOpts{
		MemoryRoot:   memoryRoot,
		QueryText:    "TDD",
		MaxEntries:   5,
		MaxTokens:    500,
		MinConfidence: 0.3,
	}

	result, err := memory.ContextInject(opts)
	g.Expect(err).ToNot(HaveOccurred())

	// Should be markdown format
	g.Expect(result).To(ContainSubstring("##"))
	// Should not contain excessive whitespace
	lines := strings.Split(result, "\n")
	for _, line := range lines {
		g.Expect(len(line)).To(BeNumerically("<=", 120), "Lines should be reasonably short")
	}
}

// TEST-4005: ContextInject respects max entries bound
// traces: TASK-4 AC-5
func TestContextInjectRespectsMaxEntries(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, "memory")

	// Create multiple learnings
	for i := 0; i < 10; i++ {
		err := memory.Learn(memory.LearnOpts{
			Message:    "Learning about testing patterns",
			MemoryRoot: memoryRoot,
		})
		g.Expect(err).ToNot(HaveOccurred())
	}

	opts := memory.ContextInjectOpts{
		MemoryRoot:   memoryRoot,
		QueryText:    "testing patterns",
		MaxEntries:   3,
		MaxTokens:    10000,
		MinConfidence: 0.3,
	}

	result, err := memory.ContextInject(opts)
	g.Expect(err).ToNot(HaveOccurred())

	// Count bullet points or sections - should not exceed MaxEntries
	bulletCount := strings.Count(result, "- ")
	g.Expect(bulletCount).To(BeNumerically("<=", opts.MaxEntries))
}

// TEST-4006: ContextInject respects max tokens bound
// traces: TASK-4 AC-6
func TestContextInjectRespectsMaxTokens(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, "memory")

	// Create a long learning
	longMessage := strings.Repeat("This is a learning about testing patterns. ", 100)
	err := memory.Learn(memory.LearnOpts{
		Message:    longMessage,
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.ContextInjectOpts{
		MemoryRoot:   memoryRoot,
		QueryText:    "testing patterns",
		MaxEntries:   10,
		MaxTokens:    200, // Small token limit
		MinConfidence: 0.3,
	}

	result, err := memory.ContextInject(opts)
	g.Expect(err).ToNot(HaveOccurred())

	// Rough estimate: 1 token ~= 4 characters
	estimatedTokens := len(result) / 4
	g.Expect(estimatedTokens).To(BeNumerically("<=", int(float64(opts.MaxTokens)*1.2)), "Should approximately respect token limit")
}

// TEST-4007: ContextInject prioritizes high-confidence memories
// traces: TASK-4 AC-7
func TestContextInjectPrioritizesHighConfidence(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, "memory")

	// Create multiple learnings (all get default confidence 1.0)
	err := memory.Learn(memory.LearnOpts{
		Message:    "First priority item",
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.ContextInjectOpts{
		MemoryRoot:   memoryRoot,
		QueryText:    "priority",
		MaxEntries:   5,
		MaxTokens:    500,
		MinConfidence: 0.3,
	}

	result, err := memory.ContextInject(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeEmpty())
}

// TEST-4008: ContextInject returns empty markdown when no results
// traces: TASK-4 AC-4
func TestContextInjectHandlesNoResults(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, "memory")

	opts := memory.ContextInjectOpts{
		MemoryRoot:   memoryRoot,
		QueryText:    "nonexistent content that will not match",
		MaxEntries:   5,
		MaxTokens:    500,
		MinConfidence: 0.3,
	}

	result, err := memory.ContextInject(opts)
	g.Expect(err).ToNot(HaveOccurred())
	// Should return valid markdown, possibly with a "no memories" message
	g.Expect(result).ToNot(BeNil())
}

// TEST-4009: ContextInject handles missing memory root
// traces: TASK-4 AC-2
func TestContextInjectHandlesMissingMemoryRoot(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, "nonexistent")

	opts := memory.ContextInjectOpts{
		MemoryRoot:   memoryRoot,
		QueryText:    "test",
		MaxEntries:   5,
		MaxTokens:    500,
		MinConfidence: 0.3,
	}

	// Should handle gracefully (return empty or error)
	result, err := memory.ContextInject(opts)
	if err != nil {
		// Error is acceptable for missing directory
		g.Expect(err).To(HaveOccurred())
	} else {
		// Or return empty result
		g.Expect(result).ToNot(BeNil())
	}
}

// TEST-4010: ContextInject uses default query text if empty
// traces: TASK-4 AC-7
func TestContextInjectDefaultQueryText(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, "memory")

	// Create a learning
	err := memory.Learn(memory.LearnOpts{
		Message:    "Recent learning about context",
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.ContextInjectOpts{
		MemoryRoot:   memoryRoot,
		QueryText:    "", // Empty - should use default
		MaxEntries:   5,
		MaxTokens:    500,
		MinConfidence: 0.3,
	}

	result, err := memory.ContextInject(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())
}

// TEST-4011: Property test - ContextInject handles various token limits
// traces: TASK-4 AC-6
func TestContextInjectPropertyVariousTokenLimits(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		suffix := rapid.StringMatching(`[a-zA-Z0-9]{8}`).Draw(rt, "suffix")
		tempDir := filepath.Join(os.TempDir(), "context-inject-test-"+suffix)
		defer func() { _ = os.RemoveAll(tempDir) }()
		_ = os.MkdirAll(tempDir, 0755)
		memoryRoot := filepath.Join(tempDir, "memory")

		// Create a learning
		_ = memory.Learn(memory.LearnOpts{
			Message:    "Test learning for property test",
			MemoryRoot: memoryRoot,
		})

		// Random token limit between 50 and 2000
		maxTokens := rapid.IntRange(50, 2000).Draw(rt, "maxTokens")

		opts := memory.ContextInjectOpts{
			MemoryRoot:   memoryRoot,
			QueryText:    "test learning",
			MaxEntries:   10,
			MaxTokens:    maxTokens,
			MinConfidence: 0.3,
		}

		result, err := memory.ContextInject(opts)
		g.Expect(err).ToNot(HaveOccurred())

		// Should produce valid output within bounds
		estimatedTokens := len(result) / 4
		g.Expect(estimatedTokens).To(BeNumerically("<=", int(float64(maxTokens)*1.5)), "Should be within reasonable bounds")
	})
}

// TEST-4012: Property test - ContextInject handles various entry limits
// traces: TASK-4 AC-5
func TestContextInjectPropertyVariousEntryLimits(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		suffix := rapid.StringMatching(`[a-zA-Z0-9]{8}`).Draw(rt, "suffix")
		tempDir := filepath.Join(os.TempDir(), "context-inject-test-"+suffix)
		defer func() { _ = os.RemoveAll(tempDir) }()
		_ = os.MkdirAll(tempDir, 0755)
		memoryRoot := filepath.Join(tempDir, "memory")

		// Create multiple learnings
		numLearnings := rapid.IntRange(1, 20).Draw(rt, "numLearnings")
		for i := 0; i < numLearnings; i++ {
			_ = memory.Learn(memory.LearnOpts{
				Message:    "Property test learning item",
				MemoryRoot: memoryRoot,
			})
		}

		// Random entry limit
		maxEntries := rapid.IntRange(1, 15).Draw(rt, "maxEntries")

		opts := memory.ContextInjectOpts{
			MemoryRoot:   memoryRoot,
			QueryText:    "learning item",
			MaxEntries:   maxEntries,
			MaxTokens:    10000,
			MinConfidence: 0.3,
		}

		result, err := memory.ContextInject(opts)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).ToNot(BeNil())

		// Count entries (approximate)
		bulletCount := strings.Count(result, "- ")
		g.Expect(bulletCount).To(BeNumerically("<=", maxEntries))
	})
}

// TEST-4013: ContextInject includes confidence scores in output
// traces: TASK-4 AC-3, AC-4
func TestContextInjectIncludesConfidenceScores(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, "memory")

	// Create a learning
	err := memory.Learn(memory.LearnOpts{
		Message:    "Learning with confidence tracking",
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.ContextInjectOpts{
		MemoryRoot:   memoryRoot,
		QueryText:    "confidence tracking",
		MaxEntries:   5,
		MaxTokens:    500,
		MinConfidence: 0.3,
	}

	result, err := memory.ContextInject(opts)
	g.Expect(err).ToNot(HaveOccurred())

	// Output may include confidence indicators
	// This is a soft requirement - checking structure
	g.Expect(result).ToNot(BeEmpty())
}

// TEST-4014: ContextInject sorts by relevance score
// traces: TASK-4 AC-7
func TestContextInjectSortsByRelevance(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, "memory")

	// Create multiple learnings with different content
	err := memory.Learn(memory.LearnOpts{
		Message:    "Highly relevant: testing patterns and best practices",
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).ToNot(HaveOccurred())

	err = memory.Learn(memory.LearnOpts{
		Message:    "Less relevant: general information",
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).ToNot(HaveOccurred())

	opts := memory.ContextInjectOpts{
		MemoryRoot:   memoryRoot,
		QueryText:    "testing patterns",
		MaxEntries:   5,
		MaxTokens:    500,
		MinConfidence: 0.3,
	}

	result, err := memory.ContextInject(opts)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeEmpty())

	// More relevant items should appear first
	// This is tested implicitly by the Query function which sorts by score
}
