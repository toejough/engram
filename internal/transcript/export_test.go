package transcript

import "time"

// SegmentsFromStrippedForTest exposes segmentsFromStripped for unit testing.
func SegmentsFromStrippedForTest(stripped []string, times []time.Time) []Segment {
	return segmentsFromStripped(stripped, times)
}
