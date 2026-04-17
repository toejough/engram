package recall

import (
	"context"
	"strings"

	"engram/internal/externalsources"
)

// ExtractFromClaudeMd runs the CLAUDE.md+rules phase of the recall pipeline.
// It concatenates every discovered CLAUDE.md and rules file, then makes a
// single Haiku extract call against the result. CLAUDE.md files are designed
// to be small (<200 lines each), so no index step is needed.
func ExtractFromClaudeMd(
	ctx context.Context,
	files []externalsources.ExternalFile,
	query string,
	cache *externalsources.FileCache,
	summarizer SummarizerI,
	buffer *strings.Builder,
	bytesUsed, bytesCap int,
) int {
	if summarizer == nil || cache == nil || bytesUsed >= bytesCap {
		return 0
	}

	combined := concatRulesAndClaudeMd(files, cache)
	if combined == "" {
		return 0
	}

	snippet, err := summarizer.ExtractRelevant(ctx, combined, query)
	if err != nil || snippet == "" {
		return 0
	}

	buffer.WriteString(snippet)

	return len(snippet)
}

// concatRulesAndClaudeMd reads and concatenates all CLAUDE.md and rules files
// from the discovered list. Files that fail to read are silently skipped.
func concatRulesAndClaudeMd(
	files []externalsources.ExternalFile,
	cache *externalsources.FileCache,
) string {
	var builder strings.Builder

	for _, file := range files {
		if file.Kind != externalsources.KindClaudeMd && file.Kind != externalsources.KindRules {
			continue
		}

		body, err := cache.Read(file.Path)
		if err != nil {
			continue
		}

		builder.Write(body)
	}

	return builder.String()
}
