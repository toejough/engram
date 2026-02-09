package memory

import (
	"fmt"
	"sort"
	"strings"
)

// FormatMarkdownOpts holds options for formatting query results as markdown.
type FormatMarkdownOpts struct {
	// Results are pre-fetched query results to format
	Results []QueryResult

	// MinConfidence is the minimum confidence score to include
	MinConfidence float64

	// MaxEntries is the maximum number of entries to include
	MaxEntries int

	// MaxTokens is the approximate maximum token count
	MaxTokens int

	// Primacy enables primacy ordering (corrections first)
	Primacy bool
}

// FormatMarkdown takes pre-fetched query results and applies confidence filtering,
// entry cap, optional primacy sort, and markdown formatting with token budget.
func FormatMarkdown(opts FormatMarkdownOpts) string {
	maxEntries := opts.MaxEntries
	if maxEntries == 0 {
		maxEntries = 10
	}

	maxTokens := opts.MaxTokens
	if maxTokens == 0 {
		maxTokens = 2000
	}

	// Filter by confidence threshold
	var filtered []QueryResult
	for _, r := range opts.Results {
		if r.Confidence >= opts.MinConfidence {
			filtered = append(filtered, r)
		}
	}

	// Limit to MaxEntries
	if len(filtered) > maxEntries {
		filtered = filtered[:maxEntries]
	}

	// Apply primacy ordering if requested
	if opts.Primacy {
		filtered = SortByPrimacy(filtered)
	}

	return formatAsMarkdown(filtered, maxTokens)
}

// formatAsMarkdown formats query results as compact markdown suitable for system prompts.
func formatAsMarkdown(results []QueryResult, maxTokens int) string {
	if len(results) == 0 {
		return formatEmptyResult()
	}

	var sb strings.Builder

	sb.WriteString("## Recent Context from Memory\n\n")

	currentTokens := 0
	estimatedHeaderTokens := 10

	currentTokens += estimatedHeaderTokens

	for i, result := range results {
		// Estimate tokens: ~4 chars per token
		entryTokens := len(result.Content) / 4

		// Check if adding this entry would exceed token limit
		if currentTokens+entryTokens > maxTokens {
			// Add truncation notice
			remaining := len(results) - i
			if remaining > 0 {
				sb.WriteString(fmt.Sprintf("\n_(... and %d more memories truncated due to token limit)_\n", remaining))
			}
			break
		}

		// Add the entry (strip leading "- " from content if present)
		content := result.Content
		content = strings.TrimPrefix(content, "- ")
		sb.WriteString(fmt.Sprintf("- %s\n", truncateLine(content, 120)))

		currentTokens += entryTokens
	}

	return sb.String()
}

// formatEmptyResult returns markdown for empty results.
func formatEmptyResult() string {
	return "## Recent Context from Memory\n\n_(No relevant memories found)_\n"
}

// SortByPrimacy reorders results so corrections appear first (primacy position),
// then by score descending within each group. Uses stable sort for deterministic ordering.
func SortByPrimacy(results []QueryResult) []QueryResult {
	if len(results) == 0 {
		return results
	}

	// Make a copy to avoid mutating the input
	sorted := make([]QueryResult, len(results))
	copy(sorted, results)

	sort.SliceStable(sorted, func(i, j int) bool {
		iIsCorrection := sorted[i].MemoryType == "correction"
		jIsCorrection := sorted[j].MemoryType == "correction"

		// Corrections come first
		if iIsCorrection != jIsCorrection {
			return iIsCorrection
		}

		// Within same group, higher score first
		return sorted[i].Score > sorted[j].Score
	})

	return sorted
}

// truncateLine truncates a line to a maximum length, preserving word boundaries.
func truncateLine(line string, maxLen int) string {
	// Remove leading/trailing whitespace and collapse internal whitespace
	line = strings.TrimSpace(line)
	line = strings.Join(strings.Fields(line), " ")

	if len(line) <= maxLen {
		return line
	}

	// Truncate at word boundary
	truncated := line[:maxLen]
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > maxLen/2 {
		truncated = truncated[:lastSpace]
	}

	return truncated + "..."
}
