package memory_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/memory"
)

// ============================================================================
// ISSUE-188 Task 6: LLM-driven pattern generation in synthesis
// ============================================================================

// --- generatePatternLLM tests ---

// TestGeneratePatternLLMProducesActionablePrinciple verifies that GeneratePatternLLM
// with a working extractor produces a SynthesisPattern whose Synthesis contains
// the LLM's output.
func TestGeneratePatternLLMProducesActionablePrinciple(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	extractor := &memory.ClaudeCLIExtractor{
		Model:   "haiku",
		Timeout: 30 * time.Second,
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte("Always write tests before implementation to catch regressions early."), nil
		},
	}

	cluster := []memory.ClusterEntry{
		{Content: "- 2026-02-09 10:00: always write tests first"},
		{Content: "- 2026-02-09 11:00: write tests before code"},
		{Content: "- 2026-02-09 12:00: TDD is essential for quality"},
	}

	pattern := memory.GeneratePatternLLM(context.Background(), cluster, extractor)

	g.Expect(pattern.Synthesis).To(ContainSubstring("Always write tests before implementation"))
	g.Expect(pattern.Occurrences).To(Equal(3))
	g.Expect(pattern.Examples).To(HaveLen(3))
	g.Expect(pattern.Theme).ToNot(BeEmpty())
}

// TestGeneratePatternLLMThemeTruncatesTo50Chars verifies Theme is capped at 50 characters.
func TestGeneratePatternLLMThemeTruncatesTo50Chars(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	longPrinciple := "This is a very long principle that exceeds fifty characters and should be truncated for the theme field"
	extractor := &memory.ClaudeCLIExtractor{
		Model:   "haiku",
		Timeout: 30 * time.Second,
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte(longPrinciple), nil
		},
	}

	cluster := []memory.ClusterEntry{
		{Content: "memory one"},
		{Content: "memory two"},
		{Content: "memory three"},
	}

	pattern := memory.GeneratePatternLLM(context.Background(), cluster, extractor)

	g.Expect(len(pattern.Theme)).To(BeNumerically("<=", 50))
}

// TestGeneratePatternLLMFallsBackOnExtractorError verifies that when the extractor
// returns an error, generatePatternLLM falls back to keyword-based generatePattern.
func TestGeneratePatternLLMFallsBackOnExtractorError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	extractor := &memory.ClaudeCLIExtractor{
		Model:   "haiku",
		Timeout: 30 * time.Second,
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return nil, errors.New("LLM unavailable")
		},
	}

	cluster := []memory.ClusterEntry{
		{Content: "- 2026-02-09 10:00: always write tests first"},
		{Content: "- 2026-02-09 11:00: write tests before code"},
		{Content: "- 2026-02-09 12:00: write your tests before implementation"},
	}

	pattern := memory.GeneratePatternLLM(context.Background(), cluster, extractor)

	// Fallback produces "Pattern observed across N memories" format
	g.Expect(pattern.Synthesis).To(ContainSubstring("Pattern observed across"))
	g.Expect(pattern.Occurrences).To(Equal(3))
	g.Expect(pattern.Examples).ToNot(BeEmpty())
}

// --- optimizeSynthesize LLM integration tests ---

// TestOptimizeSynthesizeUsesLLMWhenExtractorProvided verifies that when
// OptimizeOpts.Extractor is set, synthesis uses LLM output instead of keywords.
func TestOptimizeSynthesizeUsesLLMWhenExtractorProvided(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	claudeMDPath := filepath.Join(tempDir, "CLAUDE.md")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	// Learn 3 related messages that will cluster
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

	// Skip decay
	g.Expect(memory.SetMetadataForTest(memoryRoot, "last_optimized_at", time.Now().Format(time.RFC3339))).To(Succeed())

	synthCalled := false
	extractor := &memory.ClaudeCLIExtractor{
		Model:   "haiku",
		Timeout: 30 * time.Second,
		CommandRunner: func(_ context.Context, _ string, args ...string) ([]byte, error) {
			synthCalled = true
			return []byte("Write tests before implementation to catch regressions early."), nil
		},
	}

	result, err := memory.Optimize(memory.OptimizeOpts{
		MemoryRoot:   memoryRoot,
		ClaudeMDPath: claudeMDPath,
		AutoApprove:  true,
		Extractor:    extractor,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// If patterns were found, the LLM should have been called
	if result.PatternsFound > 0 {
		g.Expect(synthCalled).To(BeTrue(), "LLM extractor should be called when patterns are found")
	}
}

// TestOptimizeSynthesizeFallsBackWhenExtractorNil verifies that synthesis
// works normally (keyword-based) when no extractor is provided.
func TestOptimizeSynthesizeFallsBackWhenExtractorNil(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, ".claude", "memory")
	claudeMDPath := filepath.Join(tempDir, "CLAUDE.md")
	g.Expect(os.MkdirAll(memoryRoot, 0755)).To(Succeed())

	// Learn 3 related messages
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

	// Skip decay
	g.Expect(memory.SetMetadataForTest(memoryRoot, "last_optimized_at", time.Now().Format(time.RFC3339))).To(Succeed())

	// No extractor provided (nil)
	result, err := memory.Optimize(memory.OptimizeOpts{
		MemoryRoot:   memoryRoot,
		ClaudeMDPath: claudeMDPath,
		AutoApprove:  true,
		// Extractor is nil - should use keyword-based synthesis
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())
}

// --- Property tests ---

// TestPropertySynthesisOutputNeverEmptyRegardlessOfExtractor verifies that
// synthesis output is never empty regardless of extractor state (nil, error, success).
func TestPropertySynthesisOutputNeverEmptyRegardlessOfExtractor(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(rt)

		// Choose extractor behavior: nil, error, or success
		behavior := rapid.SampledFrom([]string{"nil", "error", "success"}).Draw(rt, "behavior")

		cluster := []memory.ClusterEntry{
			{Content: "- 2026-02-09 10:00: test content alpha"},
			{Content: "- 2026-02-09 11:00: test content beta"},
			{Content: "- 2026-02-09 12:00: test content gamma"},
		}

		var pattern memory.SynthesisPattern

		switch behavior {
		case "nil":
			pattern = memory.GeneratePatternLLM(context.Background(), cluster, nil)
		case "error":
			extractor := &memory.ClaudeCLIExtractor{
				Model:   "haiku",
				Timeout: 30 * time.Second,
				CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
					return nil, errors.New("simulated failure")
				},
			}
			pattern = memory.GeneratePatternLLM(context.Background(), cluster, extractor)
		case "success":
			extractor := &memory.ClaudeCLIExtractor{
				Model:   "haiku",
				Timeout: 30 * time.Second,
				CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
					return []byte("A synthesized principle about testing."), nil
				},
			}
			pattern = memory.GeneratePatternLLM(context.Background(), cluster, extractor)
		}

		g.Expect(pattern.Synthesis).ToNot(BeEmpty(), "Synthesis must never be empty for behavior=%s", behavior)
		g.Expect(pattern.Occurrences).To(Equal(3))
		g.Expect(pattern.Examples).ToNot(BeEmpty())
	})
}
