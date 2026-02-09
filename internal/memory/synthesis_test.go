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
// Unit tests for SynthesizePatterns (ISSUE-179)
// ============================================================================

// TestSynthesizePatternsClustersSimilarMemories verifies that 3+ related
// messages about the same topic form a pattern cluster.
func TestSynthesizePatternsClustersSimilarMemories(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	// Learn 3 related messages about TDD - different enough to avoid learn-time dedup (raw sim <0.9)
	// but similar enough to cluster at 0.8 threshold (pairwise sim 0.83-0.88)
	g.Expect(memory.Learn(memory.LearnOpts{
		Message:    "always write tests before writing implementation code",
		MemoryRoot: memoryRoot,
	})).To(Succeed())
	g.Expect(memory.Learn(memory.LearnOpts{
		Message:    "write tests first then write the implementation code",
		MemoryRoot: memoryRoot,
	})).To(Succeed())
	g.Expect(memory.Learn(memory.LearnOpts{
		Message:    "write your tests before you write the implementation",
		MemoryRoot: memoryRoot,
	})).To(Succeed())

	result, err := memory.SynthesizePatterns(memoryRoot, 0.8, 3)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())
	g.Expect(len(result.Patterns)).To(BeNumerically(">=", 1))
	g.Expect(result.Patterns[0].Occurrences).To(BeNumerically(">=", 3))
}

// TestSynthesizePatternsSingletonsDontForm verifies that 3 completely different
// messages do not form any patterns.
func TestSynthesizePatternsSingletonsDontForm(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	// Learn 3 completely different messages
	g.Expect(memory.Learn(memory.LearnOpts{
		Message:    "the weather forecast predicts rain tomorrow",
		MemoryRoot: memoryRoot,
	})).To(Succeed())
	g.Expect(memory.Learn(memory.LearnOpts{
		Message:    "quantum computing uses qubits for parallel calculation",
		MemoryRoot: memoryRoot,
	})).To(Succeed())
	g.Expect(memory.Learn(memory.LearnOpts{
		Message:    "the roman empire fell in four seventy six AD",
		MemoryRoot: memoryRoot,
	})).To(Succeed())

	result, err := memory.SynthesizePatterns(memoryRoot, 0.8, 3)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())
	g.Expect(result.Patterns).To(BeEmpty())
}

// TestSynthesizePatternsMinClusterSize verifies that a pair of similar messages
// does not form a pattern when minClusterSize=3.
func TestSynthesizePatternsMinClusterSize(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	// Learn 2 similar + 1 different
	g.Expect(memory.Learn(memory.LearnOpts{
		Message:    "always use dependency injection in Go code",
		MemoryRoot: memoryRoot,
	})).To(Succeed())
	g.Expect(memory.Learn(memory.LearnOpts{
		Message:    "use dependency injection for all Go functions",
		MemoryRoot: memoryRoot,
	})).To(Succeed())
	g.Expect(memory.Learn(memory.LearnOpts{
		Message:    "the capital of France is Paris",
		MemoryRoot: memoryRoot,
	})).To(Succeed())

	result, err := memory.SynthesizePatterns(memoryRoot, 0.7, 3)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())
	// The pair of 2 similar doesn't meet minClusterSize=3
	g.Expect(result.Patterns).To(BeEmpty())
}

// TestSynthesizePatternsGeneratesTheme verifies that the Theme field is
// non-empty for found patterns.
func TestSynthesizePatternsGeneratesTheme(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	// Learn 3 related messages - different enough to avoid dedup but close enough to cluster
	g.Expect(memory.Learn(memory.LearnOpts{
		Message:    "always write tests before writing implementation code",
		MemoryRoot: memoryRoot,
	})).To(Succeed())
	g.Expect(memory.Learn(memory.LearnOpts{
		Message:    "write tests first then write the implementation code",
		MemoryRoot: memoryRoot,
	})).To(Succeed())
	g.Expect(memory.Learn(memory.LearnOpts{
		Message:    "write your tests before you write the implementation",
		MemoryRoot: memoryRoot,
	})).To(Succeed())

	result, err := memory.SynthesizePatterns(memoryRoot, 0.8, 3)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Patterns).ToNot(BeEmpty())
	g.Expect(result.Patterns[0].Theme).ToNot(BeEmpty())
}

// TestSynthesizePatternsGeneratesSynthesis verifies that the Synthesis field
// describes the pattern.
func TestSynthesizePatternsGeneratesSynthesis(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	// Learn 3 related messages - different enough to avoid dedup but close enough to cluster
	g.Expect(memory.Learn(memory.LearnOpts{
		Message:    "always write tests before writing implementation code",
		MemoryRoot: memoryRoot,
	})).To(Succeed())
	g.Expect(memory.Learn(memory.LearnOpts{
		Message:    "write tests first then write the implementation code",
		MemoryRoot: memoryRoot,
	})).To(Succeed())
	g.Expect(memory.Learn(memory.LearnOpts{
		Message:    "write your tests before you write the implementation",
		MemoryRoot: memoryRoot,
	})).To(Succeed())

	result, err := memory.SynthesizePatterns(memoryRoot, 0.8, 3)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Patterns).ToNot(BeEmpty())
	g.Expect(result.Patterns[0].Synthesis).To(ContainSubstring("Pattern observed across"))
	g.Expect(result.Patterns[0].Synthesis).To(ContainSubstring("memories"))
}

// TestPropertySynthesisClusterSizeAlwaysGteMinClusterSize verifies via property
// test that every cluster in results has size >= minClusterSize.
func TestPropertySynthesisClusterSizeAlwaysGteMinClusterSize(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		g := NewWithT(t)

		suffix := rapid.StringMatching(`[a-zA-Z0-9]{8}`).Draw(t, "suffix")
		tempDir := os.TempDir()
		memoryRoot := filepath.Join(tempDir, "synthesis-prop-"+suffix)
		defer func() { _ = os.RemoveAll(memoryRoot) }()
		g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

		minCluster := rapid.IntRange(2, 5).Draw(t, "minCluster")

		// Add related but distinct learnings to avoid learn-time dedup
		messages := []string{
			"always write unit tests before writing the implementation code for any feature",
			"TDD red green refactor cycle requires failing tests first then minimal implementation",
			"test driven development means writing test cases before production code changes",
			"ensure every code change has corresponding unit test coverage written beforehand",
		}
		for _, msg := range messages {
			g.Expect(memory.Learn(memory.LearnOpts{
				Message:    msg,
				MemoryRoot: memoryRoot,
			})).To(Succeed())
		}

		result, err := memory.SynthesizePatterns(memoryRoot, 0.7, minCluster)
		g.Expect(err).ToNot(HaveOccurred())

		for _, p := range result.Patterns {
			g.Expect(p.Occurrences).To(BeNumerically(">=", minCluster))
		}
	})
}

// TestSynthesizePatternsEmptyDatabase verifies no patterns from empty DB.
func TestSynthesizePatternsEmptyDatabase(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	result, err := memory.SynthesizePatterns(memoryRoot, 0.8, 3)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())
	g.Expect(result.Patterns).To(BeEmpty())
}

// TestSynthesizePatternsAllRelated verifies that all related memories about
// the same topic form one cluster.
func TestSynthesizePatternsAllRelated(t *testing.T) {
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	// Learn 5 related but distinct messages about Go testing assertions
	g.Expect(memory.Learn(memory.LearnOpts{
		Message:    "use gomega for readable test assertions in Go projects",
		MemoryRoot: memoryRoot,
	})).To(Succeed())
	g.Expect(memory.Learn(memory.LearnOpts{
		Message:    "Go test assertions should use gomega matchers like Expect and Equal",
		MemoryRoot: memoryRoot,
	})).To(Succeed())
	g.Expect(memory.Learn(memory.LearnOpts{
		Message:    "gomega provides human readable assertion library for Go unit testing",
		MemoryRoot: memoryRoot,
	})).To(Succeed())
	g.Expect(memory.Learn(memory.LearnOpts{
		Message:    "prefer gomega over testify for assertion matchers in Go test suites",
		MemoryRoot: memoryRoot,
	})).To(Succeed())
	g.Expect(memory.Learn(memory.LearnOpts{
		Message:    "Go testing best practice: use gomega BDD assertions for clearer failures",
		MemoryRoot: memoryRoot,
	})).To(Succeed())

	result, err := memory.SynthesizePatterns(memoryRoot, 0.7, 3)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Patterns).To(HaveLen(1))
	g.Expect(result.Patterns[0].Occurrences).To(BeNumerically(">=", 3))
}

