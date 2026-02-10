package memory_test

import (
	"fmt"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/memory"
)

// ============================================================================
// TASK-4: FormatMarkdown (refactored from ContextInject)
// ============================================================================

// TEST-4001: FormatMarkdownOpts accepts required parameters
// traces: TASK-4 AC-1
func TestFormatMarkdownOptsAcceptsParameters(t *testing.T) {
	g := NewWithT(t)

	results := []memory.QueryResult{
		{Content: "test", Score: 0.8, Confidence: 0.9},
	}

	opts := memory.FormatMarkdownOpts{
		Results:       results,
		MinConfidence: 0.3,
		MaxEntries:    10,
		MaxTokens:     1000,
		Primacy:       false,
	}

	g.Expect(opts.Results).To(HaveLen(1))
	g.Expect(opts.MaxEntries).To(Equal(10))
	g.Expect(opts.MaxTokens).To(Equal(1000))
	g.Expect(opts.MinConfidence).To(Equal(0.3))
	g.Expect(opts.Primacy).To(BeFalse())
}

// TEST-4002: FormatMarkdown produces output from pre-built results
// traces: TASK-4 AC-2
func TestFormatMarkdownProducesOutput(t *testing.T) {
	g := NewWithT(t)

	results := []memory.QueryResult{
		{Content: "Test learning with high confidence", Score: 0.8, Confidence: 0.9},
	}

	output := memory.FormatMarkdown(memory.FormatMarkdownOpts{
		Results:       results,
		MinConfidence: 0.3,
		MaxEntries:    5,
		MaxTokens:     500,
	})

	g.Expect(output).ToNot(BeEmpty())
	g.Expect(output).To(ContainSubstring("Test learning"))
}

// TEST-4003: FormatMarkdown filters by confidence threshold
// traces: TASK-4 AC-3
func TestFormatMarkdownFiltersLowConfidence(t *testing.T) {
	g := NewWithT(t)

	results := []memory.QueryResult{
		{Content: "High confidence learning", Score: 0.8, Confidence: 0.9},
		{Content: "Low confidence learning", Score: 0.7, Confidence: 0.2},
	}

	output := memory.FormatMarkdown(memory.FormatMarkdownOpts{
		Results:       results,
		MinConfidence: 0.5,
		MaxEntries:    10,
		MaxTokens:     1000,
	})

	g.Expect(output).To(ContainSubstring("High confidence"))
	g.Expect(output).ToNot(ContainSubstring("Low confidence"))
}

// TEST-4004: FormatMarkdown formats output as compact markdown
// traces: TASK-4 AC-4
func TestFormatMarkdownFormatsAsMarkdown(t *testing.T) {
	g := NewWithT(t)

	results := []memory.QueryResult{
		{Content: "Always use TDD approach", Score: 0.8, Confidence: 0.9},
	}

	output := memory.FormatMarkdown(memory.FormatMarkdownOpts{
		Results:       results,
		MinConfidence: 0.3,
		MaxEntries:    5,
		MaxTokens:     500,
	})

	// Should be markdown format
	g.Expect(output).To(ContainSubstring("##"))
	// Should not contain excessive whitespace
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		g.Expect(len(line)).To(BeNumerically("<=", 120), "Lines should be reasonably short")
	}
}

// TEST-4005: FormatMarkdown respects max entries bound
// traces: TASK-4 AC-5
func TestFormatMarkdownRespectsMaxEntries(t *testing.T) {
	g := NewWithT(t)

	results := make([]memory.QueryResult, 10)
	for i := range results {
		results[i] = memory.QueryResult{
			Content:    "Learning about testing patterns",
			Score:      0.8,
			Confidence: 0.9,
		}
	}

	output := memory.FormatMarkdown(memory.FormatMarkdownOpts{
		Results:       results,
		MinConfidence: 0.3,
		MaxEntries:    3,
		MaxTokens:     10000,
	})

	// Count bullet points - should not exceed MaxEntries
	bulletCount := strings.Count(output, "- ")
	g.Expect(bulletCount).To(BeNumerically("<=", 3))
}

// TEST-4006: FormatMarkdown respects max tokens bound
// traces: TASK-4 AC-6
func TestFormatMarkdownRespectsMaxTokens(t *testing.T) {
	g := NewWithT(t)

	results := []memory.QueryResult{
		{
			Content:    strings.Repeat("This is a learning about testing patterns. ", 100),
			Score:      0.8,
			Confidence: 0.9,
		},
	}

	output := memory.FormatMarkdown(memory.FormatMarkdownOpts{
		Results:       results,
		MinConfidence: 0.3,
		MaxEntries:    10,
		MaxTokens:     200,
	})

	// Rough estimate: 1 token ~= 4 characters
	estimatedTokens := len(output) / 4
	g.Expect(estimatedTokens).To(BeNumerically("<=", int(float64(200)*1.2)), "Should approximately respect token limit")
}

// TEST-4007: FormatMarkdown preserves input order without primacy
// traces: TASK-4 AC-7
func TestFormatMarkdownPreservesInputOrder(t *testing.T) {
	g := NewWithT(t)

	results := []memory.QueryResult{
		{Content: "First item", Score: 0.5, Confidence: 0.9},
		{Content: "Second item", Score: 0.9, Confidence: 0.9},
	}

	output := memory.FormatMarkdown(memory.FormatMarkdownOpts{
		Results:       results,
		MinConfidence: 0.3,
		MaxEntries:    5,
		MaxTokens:     500,
		Primacy:       false,
	})

	g.Expect(output).ToNot(BeEmpty())
	// Without primacy, first item should appear before second
	firstIdx := strings.Index(output, "First item")
	secondIdx := strings.Index(output, "Second item")
	g.Expect(firstIdx).To(BeNumerically("<", secondIdx))
}

// TEST-4008: FormatMarkdown returns empty markdown when no results
// traces: TASK-4 AC-4
func TestFormatMarkdownHandlesNoResults(t *testing.T) {
	g := NewWithT(t)

	output := memory.FormatMarkdown(memory.FormatMarkdownOpts{
		Results:       []memory.QueryResult{},
		MinConfidence: 0.3,
		MaxEntries:    5,
		MaxTokens:     500,
	})

	g.Expect(output).ToNot(BeNil())
	g.Expect(output).To(ContainSubstring("No relevant memories found"))
}

// TEST-4009: FormatMarkdown handles nil results slice
// traces: TASK-4 AC-2
func TestFormatMarkdownHandlesNilResults(t *testing.T) {
	g := NewWithT(t)

	output := memory.FormatMarkdown(memory.FormatMarkdownOpts{
		Results:       nil,
		MinConfidence: 0.3,
		MaxEntries:    5,
		MaxTokens:     500,
	})

	g.Expect(output).ToNot(BeNil())
	g.Expect(output).To(ContainSubstring("No relevant memories found"))
}

// TEST-4010: FormatMarkdown with zero-value opts uses defaults
// traces: TASK-4 AC-7
func TestFormatMarkdownZeroValueOptsUsesDefaults(t *testing.T) {
	g := NewWithT(t)

	results := []memory.QueryResult{
		{Content: "Test with defaults", Score: 0.8, Confidence: 0.9},
	}

	output := memory.FormatMarkdown(memory.FormatMarkdownOpts{
		Results: results,
	})

	// Should still produce valid output with defaults
	g.Expect(output).ToNot(BeEmpty())
	g.Expect(output).To(ContainSubstring("Test with defaults"))
}

// TEST-4011: Property test - FormatMarkdown handles various token limits
// traces: TASK-4 AC-6
func TestFormatMarkdownPropertyVariousTokenLimits(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		results := []memory.QueryResult{
			{Content: "Test learning for property test", Score: 0.8, Confidence: 0.9},
		}

		// Random token limit between 50 and 2000
		maxTokens := rapid.IntRange(50, 2000).Draw(rt, "maxTokens")

		output := memory.FormatMarkdown(memory.FormatMarkdownOpts{
			Results:       results,
			MinConfidence: 0.3,
			MaxEntries:    10,
			MaxTokens:     maxTokens,
		})

		// Should produce valid output within bounds
		estimatedTokens := len(output) / 4
		g.Expect(estimatedTokens).To(BeNumerically("<=", int(float64(maxTokens)*1.5)), "Should be within reasonable bounds")
	})
}

// TEST-4012: Property test - FormatMarkdown handles various entry limits
// traces: TASK-4 AC-5
func TestFormatMarkdownPropertyVariousEntryLimits(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		numResults := rapid.IntRange(1, 20).Draw(rt, "numResults")
		results := make([]memory.QueryResult, numResults)
		for i := range results {
			results[i] = memory.QueryResult{
				Content:    "Property test learning item",
				Score:      0.8,
				Confidence: 0.9,
			}
		}

		// Random entry limit
		maxEntries := rapid.IntRange(1, 15).Draw(rt, "maxEntries")

		output := memory.FormatMarkdown(memory.FormatMarkdownOpts{
			Results:       results,
			MinConfidence: 0.3,
			MaxEntries:    maxEntries,
			MaxTokens:     10000,
		})

		g.Expect(output).ToNot(BeNil())

		// Count entries (approximate)
		bulletCount := strings.Count(output, "- ")
		g.Expect(bulletCount).To(BeNumerically("<=", maxEntries))
	})
}

// TEST-4013: FormatMarkdown with results produces non-empty output
// traces: TASK-4 AC-3, AC-4
func TestFormatMarkdownWithResultsProducesOutput(t *testing.T) {
	g := NewWithT(t)

	results := []memory.QueryResult{
		{Content: "Learning with confidence tracking", Score: 0.8, Confidence: 0.9},
	}

	output := memory.FormatMarkdown(memory.FormatMarkdownOpts{
		Results:       results,
		MinConfidence: 0.3,
		MaxEntries:    5,
		MaxTokens:     500,
	})

	g.Expect(output).ToNot(BeEmpty())
}

// TEST-4014: FormatMarkdown with Primacy=true sorts corrections first
// traces: TASK-4 AC-7
func TestFormatMarkdownPrimacySortsCorrectionsFirst(t *testing.T) {
	g := NewWithT(t)

	results := []memory.QueryResult{
		{Content: "regular learning about patterns", Score: 0.9, Confidence: 0.9, MemoryType: ""},
		{Content: "correction: never use force push", Score: 0.7, Confidence: 0.9, MemoryType: "correction"},
	}

	output := memory.FormatMarkdown(memory.FormatMarkdownOpts{
		Results:       results,
		MinConfidence: 0.3,
		MaxEntries:    5,
		MaxTokens:     500,
		Primacy:       true,
	})

	// With primacy enabled, correction should appear before regular
	correctionIdx := strings.Index(output, "correction")
	regularIdx := strings.Index(output, "regular learning")
	g.Expect(correctionIdx).To(BeNumerically("<", regularIdx),
		"Correction should appear before regular learning when Primacy=true")
}

// ============================================================================
// ISSUE-185: Project-aware content in FormatMarkdown
// ============================================================================

// TestFormatMarkdownProjectTaggedContent tests that project-tagged content appears in output
func TestFormatMarkdownProjectTaggedContent(t *testing.T) {
	g := NewWithT(t)

	results := []memory.QueryResult{
		{Content: "projctl uses TDD for all changes", Score: 0.9, Confidence: 0.9},
		{Content: "general testing advice", Score: 0.7, Confidence: 0.8},
	}

	output := memory.FormatMarkdown(memory.FormatMarkdownOpts{
		Results:       results,
		MinConfidence: 0.3,
		MaxEntries:    5,
		MaxTokens:     500,
	})

	g.Expect(output).ToNot(BeEmpty())
	g.Expect(output).To(ContainSubstring("projctl"))
}

// TestFormatMarkdownGenericContent tests output without project tagging
func TestFormatMarkdownGenericContent(t *testing.T) {
	g := NewWithT(t)

	results := []memory.QueryResult{
		{Content: "Always run tests before committing", Score: 0.8, Confidence: 0.9},
	}

	output := memory.FormatMarkdown(memory.FormatMarkdownOpts{
		Results:       results,
		MinConfidence: 0.3,
		MaxEntries:    5,
		MaxTokens:     500,
	})

	g.Expect(output).ToNot(BeEmpty())
	g.Expect(output).To(ContainSubstring("Always run tests"))
}

// ============================================================================
// ISSUE-188: Output tiers (compact, full, curated)
// ============================================================================

// TestFormatMarkdownCompactNoTruncation verifies TierCompact removes the 120-char truncation.
func TestFormatMarkdownCompactNoTruncation(t *testing.T) {
	g := NewWithT(t)

	longContent := strings.Repeat("word ", 30) // 150 chars
	results := []memory.QueryResult{
		{Content: longContent, Score: 0.8, Confidence: 0.9},
	}

	output := memory.FormatMarkdown(memory.FormatMarkdownOpts{
		Results:    results,
		MaxEntries: 5,
		MaxTokens:  2000,
		Tier:       memory.TierCompact,
	})

	// Should contain full content, not truncated with "..."
	g.Expect(output).ToNot(ContainSubstring("..."))
	g.Expect(output).To(ContainSubstring(strings.TrimSpace(longContent)))
}

// TestFormatMarkdownCompactTypePrefixCorrection verifies [C] prefix for corrections.
func TestFormatMarkdownCompactTypePrefixCorrection(t *testing.T) {
	g := NewWithT(t)

	results := []memory.QueryResult{
		{Content: "never use force push", Score: 0.8, Confidence: 0.9, MemoryType: "correction"},
	}

	output := memory.FormatMarkdown(memory.FormatMarkdownOpts{
		Results:    results,
		MaxEntries: 5,
		MaxTokens:  500,
		Tier:       memory.TierCompact,
	})

	g.Expect(output).To(ContainSubstring("[C]"))
}

// TestFormatMarkdownCompactTypePrefixReflection verifies [R] prefix for reflections.
func TestFormatMarkdownCompactTypePrefixReflection(t *testing.T) {
	g := NewWithT(t)

	results := []memory.QueryResult{
		{Content: "TDD is effective for reducing rework", Score: 0.8, Confidence: 0.9, MemoryType: "reflection"},
	}

	output := memory.FormatMarkdown(memory.FormatMarkdownOpts{
		Results:    results,
		MaxEntries: 5,
		MaxTokens:  500,
		Tier:       memory.TierCompact,
	})

	g.Expect(output).To(ContainSubstring("[R]"))
}

// TestFormatMarkdownCompactStripsTimestamp verifies timestamp/project tags are stripped.
func TestFormatMarkdownCompactStripsTimestamp(t *testing.T) {
	g := NewWithT(t)

	results := []memory.QueryResult{
		{Content: "- 2026-02-09 13:00: [projctl] always use TDD", Score: 0.8, Confidence: 0.9},
	}

	output := memory.FormatMarkdown(memory.FormatMarkdownOpts{
		Results:    results,
		MaxEntries: 5,
		MaxTokens:  500,
		Tier:       memory.TierCompact,
	})

	g.Expect(output).ToNot(ContainSubstring("2026-02-09"))
	g.Expect(output).ToNot(ContainSubstring("[projctl]"))
	g.Expect(output).To(ContainSubstring("always use TDD"))
}

// TestFormatMarkdownFullShowsConfidence verifies TierFull includes confidence percentage.
func TestFormatMarkdownFullShowsConfidence(t *testing.T) {
	g := NewWithT(t)

	results := []memory.QueryResult{
		{Content: "always use TDD", Score: 0.8, Confidence: 0.85, MemoryType: "correction"},
	}

	output := memory.FormatMarkdown(memory.FormatMarkdownOpts{
		Results:    results,
		MaxEntries: 5,
		MaxTokens:  2000,
		Tier:       memory.TierFull,
	})

	g.Expect(output).To(ContainSubstring("85%"))
}

// TestFormatMarkdownFullShowsMatchType verifies TierFull includes match type.
func TestFormatMarkdownFullShowsMatchType(t *testing.T) {
	g := NewWithT(t)

	results := []memory.QueryResult{
		{Content: "always use TDD", Score: 0.8, Confidence: 0.85, MatchType: "hybrid"},
	}

	output := memory.FormatMarkdown(memory.FormatMarkdownOpts{
		Results:    results,
		MaxEntries: 5,
		MaxTokens:  2000,
		Tier:       memory.TierFull,
	})

	g.Expect(output).To(ContainSubstring("hybrid"))
}

// TestFormatMarkdownFullShowsRetrievalCount verifies TierFull includes retrieval count.
func TestFormatMarkdownFullShowsRetrievalCount(t *testing.T) {
	g := NewWithT(t)

	results := []memory.QueryResult{
		{Content: "always use TDD", Score: 0.8, Confidence: 0.85, RetrievalCount: 7},
	}

	output := memory.FormatMarkdown(memory.FormatMarkdownOpts{
		Results:    results,
		MaxEntries: 5,
		MaxTokens:  2000,
		Tier:       memory.TierFull,
	})

	g.Expect(output).To(ContainSubstring("7"))
}

// TestFormatMarkdownFullShowsMemoryType verifies TierFull includes memory type.
func TestFormatMarkdownFullShowsMemoryType(t *testing.T) {
	g := NewWithT(t)

	results := []memory.QueryResult{
		{Content: "always use TDD", Score: 0.8, Confidence: 0.85, MemoryType: "correction"},
	}

	output := memory.FormatMarkdown(memory.FormatMarkdownOpts{
		Results:    results,
		MaxEntries: 5,
		MaxTokens:  2000,
		Tier:       memory.TierFull,
	})

	g.Expect(output).To(ContainSubstring("correction"))
}

// TestFormatMarkdownFullNoTruncation verifies TierFull does not truncate content.
func TestFormatMarkdownFullNoTruncation(t *testing.T) {
	g := NewWithT(t)

	longContent := strings.Repeat("detailed memory content ", 20) // ~480 chars
	results := []memory.QueryResult{
		{Content: longContent, Score: 0.8, Confidence: 0.85},
	}

	output := memory.FormatMarkdown(memory.FormatMarkdownOpts{
		Results:    results,
		MaxEntries: 5,
		MaxTokens:  5000,
		Tier:       memory.TierFull,
	})

	// Full tier should not truncate with "..."
	g.Expect(output).ToNot(ContainSubstring("..."))
	g.Expect(output).To(ContainSubstring(strings.TrimSpace(longContent)))
}

// TestFormatMarkdownFullShowsProjectBreadth verifies TierFull shows project breadth.
func TestFormatMarkdownFullShowsProjectBreadth(t *testing.T) {
	g := NewWithT(t)

	results := []memory.QueryResult{
		{
			Content:           "always use TDD",
			Score:             0.8,
			Confidence:        0.85,
			ProjectsRetrieved: []string{"projctl", "otherproject", "thirdproject"},
		},
	}

	output := memory.FormatMarkdown(memory.FormatMarkdownOpts{
		Results:    results,
		MaxEntries: 5,
		MaxTokens:  2000,
		Tier:       memory.TierFull,
	})

	g.Expect(output).To(ContainSubstring("3 projects"))
}

// TestFormatMarkdownDefaultTierIsCompact verifies that zero-value tier behaves as compact.
func TestFormatMarkdownDefaultTierIsCompact(t *testing.T) {
	g := NewWithT(t)

	results := []memory.QueryResult{
		{Content: "- 2026-02-09 13:00: [projctl] test content", Score: 0.8, Confidence: 0.9, MemoryType: "correction"},
	}

	// Zero-value tier (empty string) should behave as TierCompact
	output := memory.FormatMarkdown(memory.FormatMarkdownOpts{
		Results:    results,
		MaxEntries: 5,
		MaxTokens:  500,
	})

	// Should strip timestamp and add type prefix (compact behavior)
	g.Expect(output).To(ContainSubstring("[C]"))
	g.Expect(output).ToNot(ContainSubstring("2026-02-09"))
}

// TestPropertyFormatMarkdownTierFullContainsConfidence verifies that for any QueryResult
// with confidence X, TierFull output contains X as a percentage.
func TestPropertyFormatMarkdownTierFullContainsConfidence(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		conf := rapid.Float64Range(0.01, 0.99).Draw(rt, "confidence")
		results := []memory.QueryResult{
			{Content: "test memory content", Score: 0.8, Confidence: conf},
		}

		output := memory.FormatMarkdown(memory.FormatMarkdownOpts{
			Results:    results,
			MaxEntries: 5,
			MaxTokens:  2000,
			Tier:       memory.TierFull,
		})

		// Confidence should appear as percentage
		pctStr := fmt.Sprintf("%d%%", int(conf*100))
		g.Expect(output).To(ContainSubstring(pctStr),
			"TierFull should contain confidence as percentage: "+pctStr)
	})
}

// TestFormatMarkdownCompactBackwardCompatible verifies TierCompact still produces
// the same header and bullet format.
func TestFormatMarkdownCompactBackwardCompatible(t *testing.T) {
	g := NewWithT(t)

	results := []memory.QueryResult{
		{Content: "simple learning", Score: 0.8, Confidence: 0.9},
	}

	output := memory.FormatMarkdown(memory.FormatMarkdownOpts{
		Results:    results,
		MaxEntries: 5,
		MaxTokens:  500,
		Tier:       memory.TierCompact,
	})

	g.Expect(output).To(ContainSubstring("## Recent Context from Memory"))
	g.Expect(output).To(ContainSubstring("- "))
}
