package surface

import "engram/internal/memory"

// ExportSuppressByTranscript exposes suppressByTranscript for testing.
func ExportSuppressByTranscript(
	candidates []*memory.Stored,
	transcriptWindow string,
) ([]*memory.Stored, []SuppressionEvent) {
	return suppressByTranscript(candidates, transcriptWindow)
}
