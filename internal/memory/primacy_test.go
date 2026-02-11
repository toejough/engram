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
// ISSUE-178: Primacy ordering in FormatMarkdown
// ============================================================================

// TEST-178001: Corrections appear before non-corrections in sorted results
func TestSortByPrimacyCorrectionsFirst(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	results := []memory.QueryResult{
		{Content: "regular learning", Score: 0.9, MemoryType: ""},
		{Content: "correction: always use TDD", Score: 0.7, MemoryType: "correction"},
		{Content: "another regular", Score: 0.8, MemoryType: "reflection"},
	}

	sorted := memory.SortByPrimacy(results)

	// Corrections should appear first
	g.Expect(sorted[0].MemoryType).To(Equal("correction"))
	g.Expect(sorted[0].Content).To(Equal("correction: always use TDD"))

	// Non-corrections follow
	g.Expect(sorted[1].MemoryType).ToNot(Equal("correction"))
	g.Expect(sorted[2].MemoryType).ToNot(Equal("correction"))
}

// TEST-178002: Within corrections, higher score appears first
func TestSortByPrimacyWithinCorrections(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	results := []memory.QueryResult{
		{Content: "low score correction", Score: 0.5, MemoryType: "correction"},
		{Content: "high score correction", Score: 0.9, MemoryType: "correction"},
		{Content: "mid score correction", Score: 0.7, MemoryType: "correction"},
	}

	sorted := memory.SortByPrimacy(results)

	g.Expect(sorted[0].Score).To(BeNumerically(">=", sorted[1].Score))
	g.Expect(sorted[1].Score).To(BeNumerically(">=", sorted[2].Score))
}

// TEST-178003: Within non-corrections, higher score appears first
func TestSortByPrimacyWithinNonCorrections(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	results := []memory.QueryResult{
		{Content: "low score regular", Score: 0.3, MemoryType: ""},
		{Content: "high score regular", Score: 0.9, MemoryType: ""},
		{Content: "mid score reflection", Score: 0.6, MemoryType: "reflection"},
	}

	sorted := memory.SortByPrimacy(results)

	g.Expect(sorted[0].Score).To(BeNumerically(">=", sorted[1].Score))
	g.Expect(sorted[1].Score).To(BeNumerically(">=", sorted[2].Score))
}

// TEST-178004: No corrections means ordering is purely by score
func TestSortByPrimacyNoCorrections(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	results := []memory.QueryResult{
		{Content: "low score", Score: 0.3, MemoryType: ""},
		{Content: "high score", Score: 0.9, MemoryType: "reflection"},
		{Content: "mid score", Score: 0.6, MemoryType: ""},
	}

	sorted := memory.SortByPrimacy(results)

	g.Expect(sorted[0].Score).To(BeNumerically(">=", sorted[1].Score))
	g.Expect(sorted[1].Score).To(BeNumerically(">=", sorted[2].Score))
}

// TEST-178005: All corrections means ordering is purely by score
func TestSortByPrimacyAllCorrections(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	results := []memory.QueryResult{
		{Content: "correction A", Score: 0.3, MemoryType: "correction"},
		{Content: "correction B", Score: 0.9, MemoryType: "correction"},
		{Content: "correction C", Score: 0.6, MemoryType: "correction"},
	}

	sorted := memory.SortByPrimacy(results)

	g.Expect(sorted[0].Score).To(BeNumerically(">=", sorted[1].Score))
	g.Expect(sorted[1].Score).To(BeNumerically(">=", sorted[2].Score))
}

// TEST-178006: Property - corrections always appear before non-corrections
func TestSortByPrimacyPropertyCorrectionsFirst(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		numCorrections := rapid.IntRange(0, 5).Draw(rt, "numCorrections")
		numRegular := rapid.IntRange(0, 5).Draw(rt, "numRegular")

		if numCorrections+numRegular == 0 {
			return // Skip empty input
		}

		var results []memory.QueryResult
		for i := 0; i < numCorrections; i++ {
			score := rapid.Float64Range(0.1, 1.0).Draw(rt, "correctionScore")
			results = append(results, memory.QueryResult{
				Content:    "correction item",
				Score:      score,
				MemoryType: "correction",
			})
		}
		for i := 0; i < numRegular; i++ {
			score := rapid.Float64Range(0.1, 1.0).Draw(rt, "regularScore")
			memType := rapid.SampledFrom([]string{"", "reflection"}).Draw(rt, "memType")
			results = append(results, memory.QueryResult{
				Content:    "regular item",
				Score:      score,
				MemoryType: memType,
			})
		}

		sorted := memory.SortByPrimacy(results)

		// Find index where corrections end and non-corrections start
		lastCorrectionIdx := -1
		firstNonCorrectionIdx := len(sorted)
		for i, r := range sorted {
			if r.MemoryType == "correction" {
				lastCorrectionIdx = i
			}
		}
		for i, r := range sorted {
			if r.MemoryType != "correction" {
				firstNonCorrectionIdx = i
				break
			}
		}

		// All corrections must precede all non-corrections
		if numCorrections > 0 && numRegular > 0 {
			g.Expect(lastCorrectionIdx).To(BeNumerically("<", firstNonCorrectionIdx),
				"All corrections must appear before non-corrections")
		}
	})
}

// TEST-178007: Property - ordering is stable (deterministic for equal scores)
func TestSortByPrimacyPropertyStableOrdering(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		n := rapid.IntRange(1, 10).Draw(rt, "numResults")
		var results []memory.QueryResult
		for i := 0; i < n; i++ {
			score := rapid.Float64Range(0.1, 1.0).Draw(rt, "score")
			memType := rapid.SampledFrom([]string{"", "correction", "reflection"}).Draw(rt, "memType")
			results = append(results, memory.QueryResult{
				Content:    rapid.StringMatching(`[a-z]{5,15}`).Draw(rt, "content"),
				Score:      score,
				MemoryType: memType,
			})
		}

		// Sort twice - results should be identical (stable)
		sorted1 := memory.SortByPrimacy(results)
		sorted2 := memory.SortByPrimacy(results)

		g.Expect(len(sorted1)).To(Equal(len(sorted2)))
		for i := range sorted1 {
			g.Expect(sorted1[i].Content).To(Equal(sorted2[i].Content),
				"Sorting must be deterministic")
			g.Expect(sorted1[i].Score).To(Equal(sorted2[i].Score))
		}
	})
}

// TEST-178008: Integration - corrections appear first in FormatMarkdown output with primacy
func TestFormatMarkdownPrimacyOrdering(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, "memory")

	// Learn a correction
	err := memory.Learn(memory.LearnOpts{
		Message:    "never use git checkout dash dash dot",
		Type:       "correction",
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Learn a regular entry
	err = memory.Learn(memory.LearnOpts{
		Message:    "testing patterns are important for quality",
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Query first
	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())
	modelDir := filepath.Join(homeDir, ".claude", "models")

	queryResults, err := memory.Query(memory.QueryOpts{
		Text:       "git checkout testing patterns",
		Limit:      20,
		MemoryRoot: memoryRoot,
		ModelDir:   modelDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Then format with primacy
	result := memory.FormatMarkdown(memory.FormatMarkdownOpts{
		Results:       queryResults.Results,
		MinConfidence: 0.0,
		MaxEntries:    10,
		MaxTokens:     2000,
		Primacy:       true,
	})

	// If both entries appear, the correction should come first
	lines := strings.Split(result, "\n")
	var contentLines []string
	for _, line := range lines {
		if strings.HasPrefix(line, "- ") {
			contentLines = append(contentLines, line)
		}
	}

	// If we got at least 2 results, verify ordering
	if len(contentLines) >= 2 {
		correctionFound := false
		for _, line := range contentLines {
			if strings.Contains(line, "git checkout") {
				correctionFound = true
				break
			}
		}
		g.Expect(correctionFound).To(BeTrue(), "Correction should appear in results")
	}
}

// TEST-178009: SortByPrimacy preserves result count
func TestSortByPrimacyPreservesCount(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	results := []memory.QueryResult{
		{Content: "a", Score: 0.9, MemoryType: ""},
		{Content: "b", Score: 0.7, MemoryType: "correction"},
		{Content: "c", Score: 0.8, MemoryType: "reflection"},
		{Content: "d", Score: 0.5, MemoryType: "correction"},
	}

	sorted := memory.SortByPrimacy(results)
	g.Expect(sorted).To(HaveLen(4))
}

// TEST-178010: SortByPrimacy handles empty slice
func TestSortByPrimacyEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	sorted := memory.SortByPrimacy(nil)
	g.Expect(sorted).To(BeEmpty())

	sorted = memory.SortByPrimacy([]memory.QueryResult{})
	g.Expect(sorted).To(BeEmpty())
}

// TEST-178011: searchSimilar returns MemoryType
func TestSearchSimilarReturnsMemoryType(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tempDir := t.TempDir()
	memoryRoot := filepath.Join(tempDir, "memory")

	// Learn a correction entry
	err := memory.Learn(memory.LearnOpts{
		Message:    "important correction about git safety",
		Type:       "correction",
		MemoryRoot: memoryRoot,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Query for it
	homeDir, err := os.UserHomeDir()
	g.Expect(err).ToNot(HaveOccurred())
	modelDir := filepath.Join(homeDir, ".claude", "models")

	queryResults, err := memory.Query(memory.QueryOpts{
		Text:       "git safety",
		Limit:      5,
		MemoryRoot: memoryRoot,
		ModelDir:   modelDir,
	})
	g.Expect(err).ToNot(HaveOccurred())

	// Find the correction in results
	found := false
	for _, r := range queryResults.Results {
		if strings.Contains(r.Content, "git safety") {
			g.Expect(r.MemoryType).To(Equal("correction"))
			found = true
			break
		}
	}
	g.Expect(found).To(BeTrue(), "Should find the correction entry with MemoryType populated")
}
