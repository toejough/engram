package memory

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// Exported constants.
const (
	// TierCompact is hook output: type prefix + content, token-budgeted. Default tier.
	TierCompact OutputTier = "compact"
	// TierCurated is LLM-selected results with relevance annotations.
	TierCurated OutputTier = "curated"
	// TierFiltered is hook output when Filter() was used: only relevant memories, optional synthesis.
	TierFiltered OutputTier = "filtered"
	// TierFull is CLI --rich: metadata + full content, no truncation.
	TierFull OutputTier = "full"
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

	// Tier controls output detail level (default: TierCompact)
	Tier OutputTier

	// Query is the original user query (used by TierCurated for LLM context)
	Query string

	// Extractor is an optional LLM extractor for TierCurated
	Extractor LLMExtractor
}

// OutputTier controls the level of detail in formatted output.
type OutputTier string

// FormatFiltered formats filter results for hook output.
// Only relevant results are shown. If synthesizedText is non-empty,
// it replaces individual memories with a synthesized guidance section.
// Returns empty string if no relevant results exist.
func FormatFiltered(results []FilterResult, synthesizedText string) string {
	const maxTokens = 2000

	// Collect relevant results
	var relevant []FilterResult

	for _, r := range results {
		if r.Relevant {
			relevant = append(relevant, r)
		}
	}

	if len(relevant) == 0 {
		return ""
	}

	// If synthesis is available, use it instead of individual memories
	if synthesizedText != "" {
		return "## Recent Context from Memory\n\n" + synthesizedText + "\n"
	}

	// Format relevant results as numbered list with type prefixes
	var sb strings.Builder
	sb.WriteString("## Recent Context from Memory\n\n")

	currentTokens := 10 // estimated header tokens

	for i, r := range relevant {
		content := extractMessageContent(r.Content)
		if content == "" {
			content = strings.TrimPrefix(r.Content, "- ")
		}

		prefix := typePrefix(r.MemoryType)
		entry := prefix + content

		entryTokens := len(entry) / 4
		if currentTokens+entryTokens > maxTokens {
			remaining := len(relevant) - i
			if remaining > 0 {
				sb.WriteString(fmt.Sprintf("\n_(... and %d more memories truncated due to token limit)_\n", remaining))
			}

			break
		}

		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, entry))

		currentTokens += entryTokens
	}

	return sb.String()
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

	tier := opts.Tier
	if tier == "" {
		tier = TierCompact
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

	var memoriesSection string

	switch tier {
	case TierFull:
		memoriesSection = formatAsMarkdownFull(filtered)
	case TierCurated:
		memoriesSection = formatAsMarkdownCurated(filtered, opts.Query, opts.Extractor)
	case TierCompact, TierFiltered:
		memoriesSection = formatAsMarkdownCompact(filtered, maxTokens)
	}

	return memoriesSection
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

// formatAsMarkdownCompact formats results with type prefixes and no hard truncation.
// Token budget still controls total output length.
func formatAsMarkdownCompact(results []QueryResult, maxTokens int) string {
	if len(results) == 0 {
		return formatEmptyResult()
	}

	var sb strings.Builder

	sb.WriteString("## Recent Context from Memory\n\n")

	currentTokens := 0
	estimatedHeaderTokens := 10

	currentTokens += estimatedHeaderTokens

	for i, result := range results {
		// Extract clean content (strip timestamp/project prefix)
		content := extractMessageContent(result.Content)
		if content == "" {
			content = strings.TrimPrefix(result.Content, "- ")
		}

		// Add type prefix
		prefix := typePrefix(result.MemoryType)

		entry := prefix + content

		// Estimate tokens: ~4 chars per token
		entryTokens := len(entry) / 4

		// Check if adding this entry would exceed token limit
		if currentTokens+entryTokens > maxTokens {
			remaining := len(results) - i
			if remaining > 0 {
				sb.WriteString(fmt.Sprintf("\n_(... and %d more memories truncated due to token limit)_\n", remaining))
			}

			break
		}

		// ISSUE-214: Use numbered output format
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, entry))

		currentTokens += entryTokens
	}

	return sb.String()
}

// formatAsMarkdownCurated formats LLM-curated results with relevance annotations.
func formatAsMarkdownCurated(results []QueryResult, query string, extractor LLMExtractor) string {
	if len(results) == 0 {
		return formatEmptyResult()
	}

	// If extractor available, try LLM curation
	if extractor != nil && query != "" {
		curated, err := extractor.Curate(context.Background(), query, results)
		if err == nil && len(curated) > 0 {
			var sb strings.Builder
			sb.WriteString("## Recent Context from Memory\n\n")

			for _, c := range curated {
				sb.WriteString(fmt.Sprintf("- %s\n", c.Content))

				if c.Relevance != "" {
					sb.WriteString(fmt.Sprintf("  _(relevant: %s)_\n", c.Relevance))
				}
			}

			return sb.String()
		}
	}

	// Fall back to compact
	return formatAsMarkdownCompact(results, 2000)
}

// formatAsMarkdownFull formats results with full metadata, no truncation.
func formatAsMarkdownFull(results []QueryResult) string {
	if len(results) == 0 {
		return formatEmptyResult()
	}

	var sb strings.Builder

	sb.WriteString("## Recent Context from Memory\n\n")

	for i, result := range results {
		// Extract clean content
		content := extractMessageContent(result.Content)
		if content == "" {
			content = strings.TrimPrefix(result.Content, "- ")
		}

		// ISSUE-214: Use numbered output format
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, content))

		// Metadata line
		var meta []string

		// ISSUE-214: Show ID in rich mode
		if result.ID > 0 {
			meta = append(meta, fmt.Sprintf("id=%d", result.ID))
		}

		meta = append(meta, fmt.Sprintf("%d%% confidence", int(result.Confidence*100)))

		if result.MemoryType != "" {
			meta = append(meta, result.MemoryType)
		}

		if result.RetrievalCount > 0 {
			meta = append(meta, fmt.Sprintf("%d retrievals", result.RetrievalCount))
		}

		if len(result.ProjectsRetrieved) > 0 {
			meta = append(meta, fmt.Sprintf("%d projects", len(result.ProjectsRetrieved)))
		}

		if result.MatchType != "" {
			meta = append(meta, result.MatchType)
		}

		sb.WriteString(fmt.Sprintf("  _(%s)_\n", strings.Join(meta, " | ")))
	}

	return sb.String()
}

// formatEmptyResult returns markdown for empty results.
func formatEmptyResult() string {
	return "## Recent Context from Memory\n\n_(No relevant memories found)_\n"
}

// typePrefix returns a short type indicator for compact output.
func typePrefix(memoryType string) string {
	switch memoryType {
	case "correction":
		return "[C] "
	case "reflection":
		return "[R] "
	default:
		return ""
	}
}
