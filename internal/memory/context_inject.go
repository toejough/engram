package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ContextInjectOpts holds options for context injection.
type ContextInjectOpts struct {
	// MemoryRoot is the root directory for memory storage
	MemoryRoot string

	// QueryText is the text to search for (defaults to "recent important learnings")
	QueryText string

	// MaxEntries is the maximum number of entries to include
	MaxEntries int

	// MaxTokens is the approximate maximum token count
	MaxTokens int

	// MinConfidence is the minimum confidence score to include (default: 0.3)
	MinConfidence float64
}

// ContextInject queries high-confidence recent memories and formats them as compact markdown
// suitable for Claude's system prompt.
func ContextInject(opts ContextInjectOpts) (string, error) {
	// Set defaults
	queryText := opts.QueryText
	if queryText == "" {
		queryText = "recent important learnings"
	}

	maxEntries := opts.MaxEntries
	if maxEntries == 0 {
		maxEntries = 10
	}

	maxTokens := opts.MaxTokens
	if maxTokens == 0 {
		maxTokens = 2000
	}

	minConfidence := opts.MinConfidence
	if minConfidence == 0 {
		minConfidence = 0.3
	}

	// Determine model directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	modelDir := filepath.Join(homeDir, ".claude", "models")

	// Query memories using existing infrastructure
	queryOpts := QueryOpts{
		Text:       queryText,
		Limit:      maxEntries * 2, // Query more than needed for filtering
		MemoryRoot: opts.MemoryRoot,
		ModelDir:   modelDir,
	}

	results, err := Query(queryOpts)
	if err != nil {
		// If query fails, return empty markdown (graceful degradation)
		return formatEmptyResult(), nil
	}

	// Filter by confidence threshold
	var filtered []QueryResult
	for _, result := range results.Results {
		if result.Confidence >= minConfidence {
			filtered = append(filtered, result)
		}
	}

	// Limit to MaxEntries
	if len(filtered) > maxEntries {
		filtered = filtered[:maxEntries]
	}

	// Apply primacy ordering: corrections first, then by score (ISSUE-178)
	filtered = SortByPrimacy(filtered)

	// Format as compact markdown
	output := formatAsMarkdown(filtered, maxTokens)

	return output, nil
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
