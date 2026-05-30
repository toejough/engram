package transcript

import "time"

// Exported variables.
var (
	MustMarshalJSON = mustMarshalJSON
)

// SegmentsFromStrippedForTest exposes segmentsFromStripped for unit testing.
func SegmentsFromStrippedForTest(stripped []string, times []time.Time) []Segment {
	return segmentsFromStripped(stripped, times)
}
