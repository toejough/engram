package transcript

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	sessionctx "github.com/toejough/engram/internal/context"
)

// JSONLRangeReader reads a Claude Code session JSONL file and returns the
// noise-stripped transcript lines whose `timestamp` field falls within
// [start, end] inclusive. Lines without a parseable timestamp are dropped.
type JSONLRangeReader struct {
	reader sessionctx.FileReader
}

// NewJSONLRangeReader wires a JSONLRangeReader against the given file reader.
func NewJSONLRangeReader(reader sessionctx.FileReader) *JSONLRangeReader {
	return &JSONLRangeReader{reader: reader}
}

// ReadRange returns the filtered transcript slice for [start, end]. Lines
// without a parseable RFC3339 `timestamp` are dropped. The same noise
// filter as `JSONLReader.Read` is applied (tool-summary mode). Lines are
// returned in their original chronological order.
func (r *JSONLRangeReader) ReadRange(path string, start, end time.Time) (string, error) {
	content, err := r.reader.Read(path)
	if err != nil {
		return "", fmt.Errorf("reading transcript: %w", err)
	}

	rawLines := strings.Split(string(content), "\n")

	inRange := make([]string, 0, len(rawLines))

	for _, line := range rawLines {
		if line == "" {
			continue
		}

		ts, ok := extractTimestamp(line)
		if !ok {
			continue
		}

		if ts.Before(start) || ts.After(end) {
			continue
		}

		inRange = append(inRange, line)
	}

	cfg := sessionctx.StripConfig{ToolSummaryMode: true}
	stripped := sessionctx.StripWithConfig(inRange, cfg)

	var builder strings.Builder

	for _, line := range stripped {
		builder.WriteString(line)
		builder.WriteByte('\n')
	}

	return builder.String(), nil
}

// RangeReader reads a filtered transcript chunk bounded by [start, end]
// (inclusive on both ends). Used to inline filtered transcript content
// into episode notes via `engram learn episode --from-transcript-range`.
// Path identifies the session source; for Claude Code this is the
// per-session JSONL file path. start/end are RFC3339 timestamps.
type RangeReader interface {
	ReadRange(path string, start, end time.Time) (string, error)
}

// extractTimestamp parses the `timestamp` field from a JSONL line. Returns
// (time, true) on success; (zero, false) when the field is missing or
// unparseable. Used to filter transcript lines by RFC3339 range.
func extractTimestamp(line string) (time.Time, bool) {
	var probe struct {
		Timestamp string `json:"timestamp"`
	}

	err := json.Unmarshal([]byte(line), &probe)
	if err != nil || probe.Timestamp == "" {
		return time.Time{}, false
	}

	ts, parseErr := time.Parse(time.RFC3339, probe.Timestamp)
	if parseErr != nil {
		return time.Time{}, false
	}

	return ts, true
}
