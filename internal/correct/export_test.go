package correct

import "engram/internal/memory"

// ExportBuildRefinePrompt exposes buildRefinePrompt for testing.
func ExportBuildRefinePrompt(record *memory.MemoryRecord, transcript string) string {
	return buildRefinePrompt(record, transcript)
}

// ExportParseExtractionResponse exposes parseExtractionResponse for testing.
func ExportParseExtractionResponse(response string) (*ExtractionResult, error) {
	return parseExtractionResponse(response)
}
