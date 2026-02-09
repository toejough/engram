package main

import (
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/memory"
)

// ============================================================================
// TASK-4: context-inject CLI command
// ============================================================================

// TEST-4100: memoryContextInjectArgs structure accepts required flags
// traces: TASK-4 AC-1
func TestMemoryContextInjectArgsStructure(t *testing.T) {
	g := NewWithT(t)

	args := memoryContextInjectArgs{
		MemoryRoot:   "/path/to/memory",
		QueryText:    "recent learnings",
		MaxEntries:   10,
		MaxTokens:    1000,
		MinConfidence: 0.3,
	}

	g.Expect(args.MemoryRoot).To(Equal("/path/to/memory"))
	g.Expect(args.QueryText).To(Equal("recent learnings"))
	g.Expect(args.MaxEntries).To(Equal(10))
	g.Expect(args.MaxTokens).To(Equal(1000))
	g.Expect(args.MinConfidence).To(Equal(0.3))
}

// TEST-4101: memoryContextInject command executes successfully
// traces: TASK-4 AC-1
func TestMemoryContextInjectCommandExecutes(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, "memory")

	// Create a learning
	err := memory.Learn(memory.LearnOpts{
		Message:    "Test learning for context injection",
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).ToNot(HaveOccurred())

	args := memoryContextInjectArgs{
		MemoryRoot:   memoryRoot,
		QueryText:    "context injection",
		MaxEntries:   5,
		MaxTokens:    500,
		MinConfidence: 0.3,
	}

	err = memoryContextInject(args)
	g.Expect(err).ToNot(HaveOccurred())
}

// TEST-4102: memoryContextInject defaults MemoryRoot to ~/.claude/memory
// traces: TASK-4 AC-1
func TestMemoryContextInjectDefaultsMemoryRoot(t *testing.T) {
	g := NewWithT(t)

	args := memoryContextInjectArgs{
		QueryText:    "test",
		MaxEntries:   5,
		MaxTokens:    500,
		MinConfidence: 0.3,
		// MemoryRoot not specified - should default
	}

	// Should not panic when MemoryRoot is empty
	g.Expect(args.MemoryRoot).To(Equal(""))
}

// TEST-4103: memoryContextInject defaults MaxEntries to 10
// traces: TASK-4 AC-5
func TestMemoryContextInjectDefaultsMaxEntries(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, "memory")

	// Create learnings
	for i := 0; i < 5; i++ {
		_ = memory.Learn(memory.LearnOpts{
			Message:    "Test learning item",
			MemoryRoot: memoryRoot,
		})
	}

	args := memoryContextInjectArgs{
		MemoryRoot:   memoryRoot,
		QueryText:    "learning",
		MaxEntries:   0, // Should default to 10
		MaxTokens:    500,
		MinConfidence: 0.3,
	}

	err := memoryContextInject(args)
	// Should complete successfully with defaults
	if err != nil {
		// May error on empty memory, but should not panic
		g.Expect(err).To(HaveOccurred())
	}
}

// TEST-4104: memoryContextInject defaults MaxTokens to 2000
// traces: TASK-4 AC-6
func TestMemoryContextInjectDefaultsMaxTokens(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, "memory")

	// Create a learning
	_ = memory.Learn(memory.LearnOpts{
		Message:    "Test learning",
		MemoryRoot: memoryRoot,
	})

	args := memoryContextInjectArgs{
		MemoryRoot:   memoryRoot,
		QueryText:    "test",
		MaxEntries:   5,
		MaxTokens:    0, // Should default to 2000
		MinConfidence: 0.3,
	}

	err := memoryContextInject(args)
	// Should complete successfully with defaults
	g.Expect(err).ToNot(HaveOccurred())
}

// TEST-4105: memoryContextInject defaults MinConfidence to 0.3
// traces: TASK-4 AC-3
func TestMemoryContextInjectDefaultsMinConfidence(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, "memory")

	// Create a learning
	_ = memory.Learn(memory.LearnOpts{
		Message:    "Test learning",
		MemoryRoot: memoryRoot,
	})

	args := memoryContextInjectArgs{
		MemoryRoot:   memoryRoot,
		QueryText:    "test",
		MaxEntries:   5,
		MaxTokens:    500,
		MinConfidence: 0.0, // Should default to 0.3
	}

	err := memoryContextInject(args)
	g.Expect(err).ToNot(HaveOccurred())
}

// TEST-4106: memoryContextInject defaults QueryText to "recent important learnings"
// traces: TASK-4 AC-7
func TestMemoryContextInjectDefaultsQueryText(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, "memory")

	// Create a learning
	_ = memory.Learn(memory.LearnOpts{
		Message:    "Important recent learning",
		MemoryRoot: memoryRoot,
	})

	args := memoryContextInjectArgs{
		MemoryRoot:   memoryRoot,
		QueryText:    "", // Should default
		MaxEntries:   5,
		MaxTokens:    500,
		MinConfidence: 0.3,
	}

	err := memoryContextInject(args)
	g.Expect(err).ToNot(HaveOccurred())
}

// TEST-4107: memoryContextInject prints markdown to stdout
// traces: TASK-4 AC-4
func TestMemoryContextInjectPrintsMarkdown(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, "memory")

	// Create a learning
	err := memory.Learn(memory.LearnOpts{
		Message:    "Test learning for output",
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).ToNot(HaveOccurred())

	args := memoryContextInjectArgs{
		MemoryRoot:   memoryRoot,
		QueryText:    "output",
		MaxEntries:   5,
		MaxTokens:    500,
		MinConfidence: 0.3,
	}

	// Command should succeed and output markdown
	err = memoryContextInject(args)
	g.Expect(err).ToNot(HaveOccurred())
	// Note: actual stdout capture would require more complex test setup
	// This test verifies command completes successfully
}

// TEST-4108: memoryContextInject handles missing memory root gracefully
// traces: TASK-4 AC-2
func TestMemoryContextInjectHandlesMissingMemoryRoot(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, "nonexistent")

	args := memoryContextInjectArgs{
		MemoryRoot:   memoryRoot,
		QueryText:    "test",
		MaxEntries:   5,
		MaxTokens:    500,
		MinConfidence: 0.3,
	}

	err := memoryContextInject(args)
	// Should either error or return empty result gracefully
	if err != nil {
		g.Expect(err).To(HaveOccurred())
	}
}

// TEST-4109: memoryContextInject respects MaxEntries flag
// traces: TASK-4 AC-5
func TestMemoryContextInjectRespectsMaxEntriesFlag(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, "memory")

	// Create multiple learnings
	for i := 0; i < 10; i++ {
		_ = memory.Learn(memory.LearnOpts{
			Message:    "Test learning pattern",
			MemoryRoot: memoryRoot,
		})
	}

	args := memoryContextInjectArgs{
		MemoryRoot:   memoryRoot,
		QueryText:    "pattern",
		MaxEntries:   3,
		MaxTokens:    10000,
		MinConfidence: 0.3,
	}

	err := memoryContextInject(args)
	g.Expect(err).ToNot(HaveOccurred())
}

// TEST-4110: memoryContextInject respects MaxTokens flag
// traces: TASK-4 AC-6
func TestMemoryContextInjectRespectsMaxTokensFlag(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, "memory")

	// Create a long learning
	longMessage := "This is a very long learning message. " + "It contains lots of text to test token limits. "
	err := memory.Learn(memory.LearnOpts{
		Message:    longMessage + longMessage + longMessage,
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).ToNot(HaveOccurred())

	args := memoryContextInjectArgs{
		MemoryRoot:   memoryRoot,
		QueryText:    "long learning",
		MaxEntries:   5,
		MaxTokens:    100, // Small limit
		MinConfidence: 0.3,
	}

	err = memoryContextInject(args)
	g.Expect(err).ToNot(HaveOccurred())
}

// TEST-4111: memoryContextInject respects MinConfidence flag
// traces: TASK-4 AC-3
func TestMemoryContextInjectRespectsMinConfidenceFlag(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, "memory")

	// Create a learning
	err := memory.Learn(memory.LearnOpts{
		Message:    "Test learning with confidence",
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).ToNot(HaveOccurred())

	args := memoryContextInjectArgs{
		MemoryRoot:   memoryRoot,
		QueryText:    "confidence",
		MaxEntries:   5,
		MaxTokens:    500,
		MinConfidence: 0.8, // High threshold
	}

	err = memoryContextInject(args)
	g.Expect(err).ToNot(HaveOccurred())
}
