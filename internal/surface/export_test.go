package surface

import (
	"io"

	"engram/internal/memory"
)

// PromptMatch is the exported view of promptMatch for testing.
type PromptMatch struct {
	Mem       *memory.Stored
	BM25Score float64
}

// ExportBuildGateUserPrompt exposes buildGateUserPrompt for testing.
func ExportBuildGateUserPrompt(candidates []*memory.Stored, userMessage string) string {
	return buildGateUserPrompt(candidates, userMessage)
}

// ExportFilenameSlug exposes memory.NameFromPath for testing.
func ExportFilenameSlug(path string) string {
	return memory.NameFromPath(path)
}

// ExportFilterBySlug exposes filterBySlug for testing.
func ExportFilterBySlug(candidates []*memory.Stored, slugs []string) []*memory.Stored {
	return filterBySlug(candidates, slugs)
}

// ExportMatchPromptMemories exposes matchPromptMemories for testing.
// Returns (mem.FilePath, bm25Score) pairs sorted in original scored order.
func ExportMatchPromptMemories(message string, mems []*memory.Stored, halfLife int) []PromptMatch {
	matches := matchPromptMemories(message, mems, halfLife)
	result := make([]PromptMatch, len(matches))

	for i, m := range matches {
		result[i] = PromptMatch{Mem: m.mem, BM25Score: m.bm25Score}
	}

	return result
}

// ExportParseGateResponse exposes parseGateResponse for testing.
func ExportParseGateResponse(response string) ([]string, error) {
	return parseGateResponse(response)
}

// ExportSortPromptMatchesByScore sorts PromptMatches by score with project penalty applied.
func ExportSortPromptMatchesByScore(matches []PromptMatch, currentProjectSlug string) {
	inner := make([]promptMatch, len(matches))

	for i, m := range matches {
		inner[i] = promptMatch{mem: m.Mem, bm25Score: m.BM25Score}
	}

	sortPromptMatchesByScore(inner, currentProjectSlug)

	for i, m := range inner {
		matches[i] = PromptMatch{Mem: m.mem, BM25Score: m.bm25Score}
	}
}

// ExportSuppressByTranscript exposes suppressByTranscript for testing.
func ExportSuppressByTranscript(
	candidates []*memory.Stored,
	transcriptWindow string,
) ([]*memory.Stored, []SuppressionEvent) {
	return suppressByTranscript(candidates, transcriptWindow)
}

// ExportWriteResult exposes writeResult for testing.
func ExportWriteResult(s *Surfacer, w io.Writer, result Result, format string) error {
	return s.writeResult(w, result, format)
}
