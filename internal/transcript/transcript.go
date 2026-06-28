// Package transcript reads Claude Code session transcripts.
// JSONLReader reads and strips transcript noise with a byte budget.
package transcript

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	sessionctx "github.com/toejough/engram/internal/context"
)

// JSONLReader reads session transcripts and strips noise.
type JSONLReader struct {
	reader sessionctx.FileReader
}

// NewJSONLReader creates a JSONLReader with the given file reader.
func NewJSONLReader(reader sessionctx.FileReader) *JSONLReader {
	return &JSONLReader{reader: reader}
}

// ReadFrom reads the JSONL transcript at path, filters to rows whose
// per-row timestamp is strictly after fromTime, strips noise via
// context.Strip, and emits chronologically until budgetBytes is hit.
// The first surviving row is always emitted regardless of size (progress
// guarantee). Rows with null/missing timestamps inherit the previous
// row's timestamp; rows preceding any timestamped row inherit zero time
// (so they pass any zero-time fromTime filter but get excluded under a
// non-zero fromTime).
func (r *JSONLReader) ReadFrom(
	path string,
	fromTime time.Time,
	budgetBytes int,
) (ReadResult, error) {
	raw, err := r.reader.Read(path)
	if err != nil {
		return ReadResult{}, fmt.Errorf("reading transcript: %w", err)
	}

	lines := splitNonEmptyLines(string(raw))

	keptLines, keptTimes := filterRowsAfter(lines, extractRowTimestamps(lines), fromTime)

	cfg := sessionctx.StripConfig{ToolSummaryMode: true}
	stripped, sourceIdx := sessionctx.StripWithConfigIndexed(keptLines, cfg)
	strippedTimes := mapTimestampsByIndex(sourceIdx, keptTimes)

	return accumulateWithinBudget(stripped, strippedTimes, budgetBytes), nil
}

// SegmentsFrom reads the JSONL transcript at path over the same resolved
// window as ReadFrom (fromTime, budgetBytes) and returns one Segment per
// genuine user turn in chronological order. Injected skill/command turns are
// absent because the same strip config as ReadFrom is used; the budget cap is
// respected so the segment list covers exactly the content window that a
// non-segment read would emit (continuation warnings still apply at the CLI
// level).
func (r *JSONLReader) SegmentsFrom(
	path string,
	fromTime time.Time,
	budgetBytes int,
) (SegmentsResult, error) {
	raw, err := r.reader.Read(path)
	if err != nil {
		return SegmentsResult{}, fmt.Errorf("reading transcript: %w", err)
	}

	lines := splitNonEmptyLines(string(raw))

	keptLines, keptTimes := filterRowsAfter(lines, extractRowTimestamps(lines), fromTime)

	cfg := sessionctx.StripConfig{ToolSummaryMode: true}
	stripped, sourceIdx := sessionctx.StripWithConfigIndexed(keptLines, cfg)
	strippedTimes := mapTimestampsByIndex(sourceIdx, keptTimes)

	budgetedStripped, budgetedTimes, partial := budgetSegmentLines(
		stripped,
		strippedTimes,
		budgetBytes,
	)

	return SegmentsResult{
		Segments: segmentsFromStripped(budgetedStripped, budgetedTimes),
		Partial:  partial,
	}, nil
}

// ReadResult bundles a partial-or-full session read's output with the
// information emitTranscripts needs to advance the per-source marker.
// LastTimestamp is the per-row timestamp of the last row included in
// Content (zero when Content is empty or no row had a non-null
// timestamp). Partial reports whether budgetBytes halted the read
// before the file was exhausted.
type ReadResult struct {
	Content       string
	BytesUsed     int
	LastTimestamp time.Time
	Partial       bool
}

// Reader reads and strips a transcript file forward from a marker
// timestamp, with a byte budget. Returns a ReadResult bundling content,
// bytes consumed, the per-row LastTimestamp of the last row included,
// and Partial reporting whether the budget halted the read mid-file.
type Reader interface {
	ReadFrom(path string, fromTime time.Time, budgetBytes int) (ReadResult, error)
}

// Segment is a single genuine user turn distilled to a timestamp + preview.
// It is produced by SegmentsFrom and used by the --segments transcript flag
// so agents can identify arc boundaries without reading the full transcript.
type Segment struct {
	// Timestamp is the per-row RFC3339 timestamp of the user turn.
	Timestamp time.Time
	// Preview is the first segmentPreviewLen characters of the cleaned user
	// text, with internal newlines collapsed to spaces.
	Preview string
}

// SegmentsReader returns user-turn segments for a transcript window.
// SegmentsFrom applies the same strip config and byte budget as ReadFrom so
// the segment list covers exactly the content window a non-segment read emits.
// The returned SegmentsResult.Partial reports whether the byte budget truncated
// the scan, mirroring Reader.ReadFrom's ReadResult.Partial.
type SegmentsReader interface {
	SegmentsFrom(path string, fromTime time.Time, budgetBytes int) (SegmentsResult, error)
}

// SegmentsResult bundles a segments read's output with the truncation signal
// emitSegments needs to gate the per-source marker advance. Segments holds the
// user-turn segments within the byte window (chronological); Partial reports
// whether budgetBytes halted the scan before the file was exhausted (mirrors
// ReadResult.Partial). On Partial the caller must hold the marker at the last
// fully-included segment rather than advancing to the entry's Mtime, or a
// budget-truncated scan over-advances past unread rows (M2 data loss).
type SegmentsResult struct {
	Segments []Segment
	Partial  bool
}

// unexported constants.
const (
	segmentPreviewLen = 100
)

// accumulateWithinBudget walks lines chronologically and emits up to
// budgetBytes (budgetBytes <= 0 means no cap — emit everything), with a
// first-row progress guarantee. Returns the emitted content plus the
// last row's timestamp.
func accumulateWithinBudget(
	lines []string, times []time.Time, budgetBytes int,
) ReadResult {
	var builder strings.Builder

	bytesUsed := 0
	lastTs := time.Time{}
	partial := false

	for i, line := range lines {
		lineLen := len(line) + 1
		if budgetBytes > 0 && bytesUsed > 0 && bytesUsed+lineLen > budgetBytes {
			partial = true

			break
		}

		builder.WriteString(line)
		builder.WriteByte('\n')

		bytesUsed += lineLen

		if !times[i].IsZero() {
			lastTs = times[i]
		}
	}

	return ReadResult{
		Content:       builder.String(),
		BytesUsed:     bytesUsed,
		LastTimestamp: lastTs,
		Partial:       partial,
	}
}

// budgetSegmentLines walks (stripped, times) chronologically and returns the
// prefix that fits within budgetBytes — budgetBytes <= 0 means no cap
// (counting each line's length + newline,
// consistent with accumulateWithinBudget), the parallel timestamps, and whether
// the budget truncated the scan before the input was exhausted. The first line
// is always included (progress guarantee, mirroring accumulateWithinBudget) so
// a non-empty input never reports Partial unless a later line was actually
// dropped.
func budgetSegmentLines(
	stripped []string, times []time.Time, budgetBytes int,
) ([]string, []time.Time, bool) {
	budgetedStripped := make([]string, 0, len(stripped))
	budgetedTimes := make([]time.Time, 0, len(stripped))
	total := 0
	partial := false

	for i, line := range stripped {
		lineLen := len(line) + 1
		if budgetBytes > 0 && total > 0 && total+lineLen > budgetBytes {
			partial = true

			break
		}

		budgetedStripped = append(budgetedStripped, line)
		budgetedTimes = append(budgetedTimes, times[i])
		total += lineLen
	}

	return budgetedStripped, budgetedTimes, partial
}

// extractRowTimestamps parses each JSONL line for a top-level
// "timestamp" field. Rows with null/missing/unparseable timestamps
// inherit the previous row's timestamp; the first row inherits zero.
func extractRowTimestamps(lines []string) []time.Time {
	out := make([]time.Time, len(lines))

	carry := time.Time{}

	for i, line := range lines {
		if ts := parseRowTimestamp(line); !ts.IsZero() {
			carry = ts
		}

		out[i] = carry
	}

	return out
}

// filterRowsAfter returns the subset of (lines, times) whose timestamp
// is strictly after fromTime. When fromTime is the zero time, returns
// all rows (no filtering).
func filterRowsAfter(
	lines []string, times []time.Time, fromTime time.Time,
) ([]string, []time.Time) {
	if fromTime.IsZero() {
		return lines, times
	}

	keptLines := make([]string, 0, len(lines))
	keptTimes := make([]time.Time, 0, len(lines))

	for i, line := range lines {
		if !times[i].After(fromTime) {
			continue
		}

		keptLines = append(keptLines, line)
		keptTimes = append(keptTimes, times[i])
	}

	return keptLines, keptTimes
}

// mapTimestampsByIndex returns the per-output-line timestamps by indexing
// inputTimes through the source-index slice that strip returned alongside
// the stripped lines. Out-of-range indices (shouldn't happen, defensive
// only) map to the zero time.
func mapTimestampsByIndex(sourceIdx []int, inputTimes []time.Time) []time.Time {
	out := make([]time.Time, len(sourceIdx))

	for i, idx := range sourceIdx {
		if idx < 0 || idx >= len(inputTimes) {
			continue
		}

		out[i] = inputTimes[idx]
	}

	return out
}

// parseRowTimestamp extracts a top-level "timestamp" field from a JSON
// line. Returns the zero time when the line is not JSON, has no
// "timestamp", or the timestamp is null/unparseable.
func parseRowTimestamp(line string) time.Time {
	var probe struct {
		Timestamp string `json:"timestamp"`
	}

	unmarshalErr := json.Unmarshal([]byte(line), &probe)
	if unmarshalErr != nil {
		return time.Time{}
	}

	if probe.Timestamp == "" {
		return time.Time{}
	}

	t, err := time.Parse(time.RFC3339Nano, probe.Timestamp)
	if err != nil {
		return time.Time{}
	}

	return t
}

// segmentsFromStripped converts a parallel (stripped-lines, times) slice
// into Segment values for each genuine user turn. Stripped lines that start
// with sessionctx.UserPrefix and have a non-zero timestamp yield a segment;
// all others are skipped. Preview text is truncated to segmentPreviewLen
// runes and newlines are replaced with spaces so each segment fits one line.
func segmentsFromStripped(stripped []string, times []time.Time) []Segment {
	out := make([]Segment, 0)

	for i, line := range stripped {
		if !strings.HasPrefix(line, sessionctx.UserPrefix) {
			continue
		}

		// Defensive bounds guard: stripped and times are parallel, but guarding
		// keeps the index access provably safe (and satisfies nilaway across the
		// budgetSegmentLines call boundary).
		if i >= len(times) {
			continue
		}

		if times[i].IsZero() {
			continue
		}

		text := strings.ReplaceAll(line, "\n", " ")

		runes := []rune(text)
		if len(runes) > segmentPreviewLen {
			runes = runes[:segmentPreviewLen]
			text = string(runes)
		}

		out = append(out, Segment{
			Timestamp: times[i],
			Preview:   text,
		})
	}

	return out
}

// splitNonEmptyLines splits content on newlines, dropping empty entries
// (notably the trailing empty line from a final newline).
func splitNonEmptyLines(content string) []string {
	raw := strings.Split(content, "\n")

	out := make([]string, 0, len(raw))

	for _, line := range raw {
		if line != "" {
			out = append(out, line)
		}
	}

	return out
}
