package correct

// ExportParseExtractionResponse exposes parseExtractionResponse for testing.
func ExportParseExtractionResponse(response string) (*ExtractionResult, error) {
	return parseExtractionResponse(response)
}
