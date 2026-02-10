package memory_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/memory"
)

// ============================================================================
// TASK-8: Contradiction detection tests
// ============================================================================

// TEST-1100: Detect contradictions via negation patterns
// traces: TASK-8
func TestDetectContradictionViaNegation(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Store initial advice
	err = memory.Learn(memory.LearnOpts{
		Message:    "Always use dependency injection for testability",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Store contradictory advice with negation (high semantic overlap)
	result, err := memory.LearnWithConflictCheck(memory.LearnOpts{
		Message:    "Never use dependency injection for testability",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())
	g.Expect(result.HasConflict).To(BeTrue())
	g.Expect(result.ConflictType).To(Equal("contradiction"))
	g.Expect(result.Similarity).To(BeNumerically(">", 0.85))
}

// TEST-1101: Detect contradictions via opposing advice
// traces: TASK-8
func TestDetectContradictionViaOpposingAdvice(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Store advice
	err = memory.Learn(memory.LearnOpts{
		Message:    "Use PostgreSQL for complex queries",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Store contradictory advice (avoid vs use)
	result, err := memory.LearnWithConflictCheck(memory.LearnOpts{
		Message:    "Avoid PostgreSQL for complex queries, use MySQL instead",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())
	g.Expect(result.HasConflict).To(BeTrue())
	g.Expect(result.ConflictType).To(Equal("contradiction"))
}

// TEST-1102: High similarity without contradiction = duplicate
// traces: TASK-8
func TestHighSimilarityWithoutContradictionIsDuplicate(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Store initial advice
	err = memory.Learn(memory.LearnOpts{
		Message:    "Use dependency injection for testability",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Store nearly identical advice (no negation/opposition)
	result, err := memory.LearnWithConflictCheck(memory.LearnOpts{
		Message:    "Use dependency injection for better testability in code",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())
	g.Expect(result.HasConflict).To(BeTrue())
	g.Expect(result.ConflictType).To(Equal("duplicate"))
	g.Expect(result.Similarity).To(BeNumerically(">", 0.85))
}

// TEST-1103: Low similarity = no conflict
// traces: TASK-8
func TestLowSimilarityNoConflict(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	// Store initial advice
	err = memory.Learn(memory.LearnOpts{
		Message:    "PostgreSQL is good for complex queries",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Store unrelated advice
	result, err := memory.LearnWithConflictCheck(memory.LearnOpts{
		Message:    "CSS grid layout is useful for responsive design",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())
	g.Expect(result.HasConflict).To(BeFalse())
	g.Expect(result.ConflictType).To(BeEmpty())
}

// TEST-1104: Conflict result returns existing entry text
// traces: TASK-8
func TestConflictReturnsExistingEntryText(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	originalMsg := "Always validate input data before processing"
	err = memory.Learn(memory.LearnOpts{
		Message:    originalMsg,
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Store contradictory advice (high semantic overlap)
	result, err := memory.LearnWithConflictCheck(memory.LearnOpts{
		Message:    "Never validate input data before processing",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.HasConflict).To(BeTrue())
	g.Expect(result.ConflictEntry).To(ContainSubstring(originalMsg))
}

// TEST-1105: Conflict result includes similarity score
// traces: TASK-8
func TestConflictResultIncludesSimilarityScore(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	err = memory.Learn(memory.LearnOpts{
		Message:    "Use caching for performance optimization",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Store contradictory advice
	result, err := memory.LearnWithConflictCheck(memory.LearnOpts{
		Message:    "Avoid caching for performance optimization",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.HasConflict).To(BeTrue())
	g.Expect(result.Similarity).To(BeNumerically(">", 0.85))
	g.Expect(result.Similarity).To(BeNumerically("<=", 1.0))
}

// TEST-1106: Contradiction still stores entry (surfaces to caller)
// traces: TASK-8
func TestContradictionStillStoresEntry(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryDir := filepath.Join(tempDir, "memory")
	err := os.MkdirAll(memoryDir, 0755)
	g.Expect(err).ToNot(HaveOccurred())

	err = memory.Learn(memory.LearnOpts{
		Message:    "Use tabs for indentation in code",
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Store contradictory advice (high semantic overlap)
	contradictoryMsg := "Never use tabs for indentation in code"
	result, err := memory.LearnWithConflictCheck(memory.LearnOpts{
		Message:    contradictoryMsg,
		MemoryRoot: memoryDir,
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.HasConflict).To(BeTrue())
	g.Expect(result.Stored).To(BeTrue())

	// Verify DB was created and has content
	dbPath := filepath.Join(memoryDir, "embeddings.db")
	_, err = os.Stat(dbPath)
	g.Expect(err).ToNot(HaveOccurred(), "embeddings.db should exist")
}

// TEST-1107: Property-based: negation patterns always detected when present with high similarity
// traces: TASK-8
func TestPropertyNegationPatternsAlwaysDetected(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		g := NewWithT(t)

		suffix := rapid.StringMatching(`[a-zA-Z0-9]{8}`).Draw(t, "suffix")
		tempDir := os.TempDir()
		memoryDir := filepath.Join(tempDir, "contradiction-prop-"+suffix)
		defer func() { _ = os.RemoveAll(memoryDir) }()

		err := os.MkdirAll(memoryDir, 0755)
		g.Expect(err).ToNot(HaveOccurred())

		// Base message with an action verb
		baseMsg := "Use dependency injection for testing"
		err = memory.Learn(memory.LearnOpts{
			Message:    baseMsg,
			MemoryRoot: memoryDir,
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Pick a negation pattern
		negations := []string{"Never", "Don't", "Avoid", "Do not"}
		negation := negations[rapid.IntRange(0, len(negations)-1).Draw(t, "negation")]

		// Create contradictory message
		contradictoryMsg := negation + " use dependency injection for testing"

		result, err := memory.LearnWithConflictCheck(memory.LearnOpts{
			Message:    contradictoryMsg,
			MemoryRoot: memoryDir,
		})
		g.Expect(err).ToNot(HaveOccurred())

		// Property: when high similarity + negation pattern, must be contradiction
		if result.Similarity > 0.85 {
			g.Expect(result.HasConflict).To(BeTrue())
			g.Expect(result.ConflictType).To(Equal("contradiction"))
		}
	})
}
