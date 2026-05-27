// Package transcript finds and reads Claude Code session transcripts.
// It provides SessionFinder (locates transcript files sorted by recency)
// and JSONLReader (reads and strips transcript noise with a byte budget).
package transcript

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	sessionctx "github.com/toejough/engram/internal/context"
)

// DirLister lists .jsonl files in a directory with their modification times.
type DirLister interface {
	ListJSONL(dir string) ([]FileEntry, error)
}

// FileEntry represents a transcript file with its path and modification time.
type FileEntry struct {
	Path   string
	Mtime  time.Time
	Source string
}

// Finder finds session transcript files.
type Finder interface {
	Find(dirs ...string) ([]FileEntry, error)
}

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

// SessionFinder finds Claude Code session transcript files for a project.
type SessionFinder struct {
	lister DirLister
}

// NewSessionFinder creates a SessionFinder with the given directory lister.
func NewSessionFinder(lister DirLister) *SessionFinder {
	return &SessionFinder{lister: lister}
}

// Find returns transcript entries from all directories, merged and sorted by
// mtime descending (newest first). Missing directories are silently skipped.
func (f *SessionFinder) Find(dirs ...string) ([]FileEntry, error) {
	seen := make(map[string]struct{})
	all := make([]FileEntry, 0)

	for _, dir := range dirs {
		entries, err := f.lister.ListJSONL(dir)
		if err != nil {
			return nil, fmt.Errorf("listing sessions in %s: %w", dir, err)
		}

		for _, e := range entries {
			if _, ok := seen[e.Path]; !ok {
				seen[e.Path] = struct{}{}
				e.Source = sourceClaude
				all = append(all, e)
			}
		}
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].Mtime.After(all[j].Mtime)
	})

	return all, nil
}

// unexported constants.
const (
	sourceClaude   = "claude"
	sourceOpencode = "opencode"
)

// accumulateWithinBudget walks lines chronologically and emits up to
// budgetBytes, with a first-row progress guarantee. Returns the
// emitted content plus the last row's timestamp.
func accumulateWithinBudget(
	lines []string, times []time.Time, budgetBytes int,
) ReadResult {
	var builder strings.Builder

	bytesUsed := 0
	lastTs := time.Time{}
	partial := false

	for i, line := range lines {
		lineLen := len(line) + 1
		if bytesUsed > 0 && bytesUsed+lineLen > budgetBytes {
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
