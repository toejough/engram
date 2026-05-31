package transcript

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	// Pure-Go SQLite driver. Registered via init() side effect.
	_ "modernc.org/sqlite"

	sessionctx "github.com/toejough/engram/internal/context"
)

// Exported variables.
var (
	ErrEmptySessionID = errors.New("empty opencode session ID")
	ErrNoReader       = errors.New("no reader could handle path")
)

// CompositeSessionFinder combines multiple finders into one.
type CompositeSessionFinder struct {
	finders []Finder
}

// NewCompositeSessionFinder creates a finder that aggregates multiple finders.
func NewCompositeSessionFinder(finders ...Finder) *CompositeSessionFinder {
	return &CompositeSessionFinder{finders: finders}
}

// Find returns transcript entries from all finders, merged and sorted by
// mtime descending (newest first). Duplicate paths are deduplicated.
func (f *CompositeSessionFinder) Find(dirs ...string) ([]FileEntry, error) {
	seen := make(map[string]struct{})
	all := make([]FileEntry, 0)

	for _, finder := range f.finders {
		entries, err := finder.Find(dirs...)
		if err != nil {
			continue
		}

		for _, e := range entries {
			if _, ok := seen[e.Path]; !ok {
				seen[e.Path] = struct{}{}
				all = append(all, e)
			}
		}
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].Mtime.After(all[j].Mtime)
	})

	return all, nil
}

// CompositeTranscriptReader combines multiple readers into one.
type CompositeTranscriptReader struct {
	readers []Reader
}

// NewCompositeTranscriptReader creates a reader that tries multiple readers.
func NewCompositeTranscriptReader(readers ...Reader) *CompositeTranscriptReader {
	return &CompositeTranscriptReader{readers: readers}
}

// ReadFrom dispatches to the first reader whose ReadFrom call succeeds.
func (r *CompositeTranscriptReader) ReadFrom(
	path string,
	fromTime time.Time,
	budgetBytes int,
) (ReadResult, error) {
	for _, reader := range r.readers {
		result, err := reader.ReadFrom(path, fromTime, budgetBytes)
		if err == nil {
			return result, nil
		}
	}

	return ReadResult{}, fmt.Errorf("%w: %s", ErrNoReader, path)
}

// SegmentsFrom dispatches to the first underlying reader that implements
// SegmentsReader and succeeds. Readers that do not implement SegmentsReader
// are skipped. When no reader can handle path, returns an empty slice.
func (r *CompositeTranscriptReader) SegmentsFrom(
	path string,
	fromTime time.Time,
	budgetBytes int,
) ([]Segment, error) {
	for _, reader := range r.readers {
		sr, ok := reader.(SegmentsReader)
		if !ok {
			continue
		}

		segs, err := sr.SegmentsFrom(path, fromTime, budgetBytes)
		if err == nil {
			return segs, nil
		}
	}

	return []Segment{}, nil
}

// OpencodeSessionFinder finds OpenCode session transcripts from a SQLite database.
// When cwd is non-empty, sessions are filtered to those whose stored directory
// equals cwd or is a subdirectory of cwd (matched against the session table's
// `directory` column with a trailing-slash prefix). Empty cwd returns all
// sessions globally — same behavior as before this filter existed.
type OpencodeSessionFinder struct {
	dbPath string
	cwd    string
}

// NewOpencodeSessionFinder creates a finder that queries the OpenCode SQLite
// database, optionally scoped to sessions opened in cwd or a subdirectory of cwd.
// Pass an empty cwd to disable directory filtering.
func NewOpencodeSessionFinder(dbPath, cwd string) *OpencodeSessionFinder {
	return &OpencodeSessionFinder{dbPath: dbPath, cwd: cwd}
}

// Find returns transcript entries for sessions in the OpenCode database,
// optionally scoped to the finder's cwd, sorted by time_updated descending
// (newest first).
func (f *OpencodeSessionFinder) Find(_ ...string) ([]FileEntry, error) {
	db, openErr := sql.Open("sqlite", f.dbPath)
	if openErr != nil {
		return nil, fmt.Errorf("opening opencode database: %w", openErr)
	}

	defer db.Close() //nolint:errcheck

	ctx := context.Background()

	rows, queryErr := f.querySessions(ctx, db)
	if queryErr != nil {
		return nil, fmt.Errorf("querying sessions: %w", queryErr)
	}

	defer rows.Close() //nolint:errcheck

	entries := make([]FileEntry, 0)

	for rows.Next() {
		var id string

		var rowUpdatedAt int64

		scanErr := rows.Scan(&id, &rowUpdatedAt)
		if scanErr != nil {
			return nil, fmt.Errorf("scanning session row: %w", scanErr)
		}

		entries = append(entries, FileEntry{
			Path:   opencodeURIPrefix + id,
			Mtime:  time.UnixMilli(rowUpdatedAt),
			Source: sourceOpencode,
		})
	}

	rowErr := rows.Err()
	if rowErr != nil {
		return nil, fmt.Errorf("iterating session rows: %w", rowErr)
	}

	return entries, nil
}

// querySessions runs the session-list query, optionally filtered by f.cwd.
func (f *OpencodeSessionFinder) querySessions(ctx context.Context, db *sql.DB) (*sql.Rows, error) {
	if f.cwd == "" {
		//nolint:wrapcheck // sql driver error returned to caller verbatim
		return db.QueryContext(
			ctx,
			"SELECT id, time_updated FROM session ORDER BY time_updated DESC",
		)
	}

	prefix := f.cwd
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	return db.QueryContext( //nolint:wrapcheck
		ctx,
		"SELECT id, time_updated FROM session "+
			"WHERE directory = ? OR directory LIKE ? "+
			"ORDER BY time_updated DESC",
		f.cwd, prefix+"%",
	)
}

// OpencodeTranscriptReader reads OpenCode session transcripts from SQLite.
type OpencodeTranscriptReader struct {
	dbPath string
}

// NewOpencodeTranscriptReader creates a reader for OpenCode session transcripts.
func NewOpencodeTranscriptReader(dbPath string) *OpencodeTranscriptReader {
	return &OpencodeTranscriptReader{dbPath: dbPath}
}

// ReadFrom reads an OpenCode session transcript, filters to parts whose
// time_created is strictly after fromTime, strips noise, and returns a
// chronological ReadResult capped at budgetBytes. The path must be an
// "opencode://<session_id>" URI. Each emitted line embeds a "timestamp"
// field so the caller (and shared accumulator) can recover the per-row
// time of the last-included row.
func (r *OpencodeTranscriptReader) ReadFrom(
	path string,
	fromTime time.Time,
	budgetBytes int,
) (ReadResult, error) {
	sessionID := strings.TrimPrefix(path, opencodeURIPrefix)
	if sessionID == "" {
		return ReadResult{}, ErrEmptySessionID
	}

	jsonlLines, queryErr := r.queryPartsAfter(sessionID, fromTime)
	if queryErr != nil {
		return ReadResult{}, queryErr
	}

	inputTimes := extractRowTimestamps(jsonlLines)

	cfg := sessionctx.StripConfig{ToolSummaryMode: true}
	stripped, sourceIdx := sessionctx.StripWithConfigIndexed(jsonlLines, cfg)
	strippedTimes := mapTimestampsByIndex(sourceIdx, inputTimes)

	return accumulateWithinBudget(stripped, strippedTimes, budgetBytes), nil
}

// ReadRange returns the noise-stripped OpenCode transcript for parts whose
// time_created falls within [start, end] inclusive. The path must be an
// "opencode://<session_id>" URI. It satisfies the RangeReader interface,
// mirroring JSONLRangeReader for the SQLite-backed source so that
// `engram learn episode --from-transcript-range` can inline an OpenCode window.
func (r *OpencodeTranscriptReader) ReadRange(path string, start, end time.Time) (string, error) {
	sessionID := strings.TrimPrefix(path, opencodeURIPrefix)
	if sessionID == "" || sessionID == path {
		return "", ErrEmptySessionID
	}

	jsonlLines, queryErr := r.queryPartsBetween(sessionID, start, end)
	if queryErr != nil {
		return "", queryErr
	}

	cfg := sessionctx.StripConfig{ToolSummaryMode: true}
	stripped := sessionctx.StripWithConfig(jsonlLines, cfg)

	var builder strings.Builder

	for _, line := range stripped {
		builder.WriteString(line)
		builder.WriteByte('\n')
	}

	return builder.String(), nil
}

// SegmentsFrom returns user-turn segments for an OpenCode session window.
// Applies the same strip config and byte budget as ReadFrom so the segment
// list covers exactly the content window a non-segment read emits.
func (r *OpencodeTranscriptReader) SegmentsFrom(
	path string,
	fromTime time.Time,
	budgetBytes int,
) ([]Segment, error) {
	sessionID := strings.TrimPrefix(path, opencodeURIPrefix)
	if sessionID == "" {
		return nil, ErrEmptySessionID
	}

	jsonlLines, queryErr := r.queryPartsAfter(sessionID, fromTime)
	if queryErr != nil {
		return nil, queryErr
	}

	inputTimes := extractRowTimestamps(jsonlLines)

	cfg := sessionctx.StripConfig{ToolSummaryMode: true}
	stripped, sourceIdx := sessionctx.StripWithConfigIndexed(jsonlLines, cfg)
	strippedTimes := mapTimestampsByIndex(sourceIdx, inputTimes)

	// Apply the same byte-budget window as ReadFrom.
	budget := budgetBytes
	budgetedStripped := make([]string, 0, len(stripped))
	budgetedTimes := make([]time.Time, 0, len(stripped))
	total := 0

	for i, line := range stripped {
		lineLen := len(line) + 1
		if total > 0 && total+lineLen > budget {
			break
		}

		budgetedStripped = append(budgetedStripped, line)
		budgetedTimes = append(budgetedTimes, strippedTimes[i])
		total += lineLen
	}

	return segmentsFromStripped(budgetedStripped, budgetedTimes), nil
}

// queryPartsAfter runs the parts query filtered to time_created strictly
// after fromTime (or all parts when fromTime is the zero value). Each
// emitted JSONL line carries an embedded "timestamp" field encoding the
// row's time_created so downstream consumers can recover per-row times
// from the stripped output.
func (r *OpencodeTranscriptReader) queryPartsAfter(
	sessionID string,
	fromTime time.Time,
) ([]string, error) {
	db, openErr := sql.Open("sqlite", r.dbPath)
	if openErr != nil {
		return nil, fmt.Errorf("opening opencode database: %w", openErr)
	}

	defer db.Close() //nolint:errcheck

	fromMillis := int64(0)
	if !fromTime.IsZero() {
		fromMillis = fromTime.UnixMilli()
	}

	rows, queryErr := db.QueryContext(
		context.Background(),
		partSelectSQL+" AND p.time_created > ? ORDER BY p.time_created",
		sessionID,
		fromMillis,
	)
	if queryErr != nil {
		return nil, fmt.Errorf("querying parts: %w", queryErr)
	}

	defer rows.Close() //nolint:errcheck

	return scanPartRows(rows)
}

// queryPartsBetween runs the parts query bounded to time_created within
// [start, end] inclusive — the window a bounded ReadRange inlines into an
// episode note.
func (r *OpencodeTranscriptReader) queryPartsBetween(
	sessionID string,
	start, end time.Time,
) ([]string, error) {
	db, openErr := sql.Open("sqlite", r.dbPath)
	if openErr != nil {
		return nil, fmt.Errorf("opening opencode database: %w", openErr)
	}

	defer db.Close() //nolint:errcheck

	rows, queryErr := db.QueryContext(
		context.Background(),
		partSelectSQL+" AND p.time_created >= ? AND p.time_created <= ? ORDER BY p.time_created",
		sessionID,
		start.UnixMilli(),
		end.UnixMilli(),
	)
	if queryErr != nil {
		return nil, fmt.Errorf("querying parts: %w", queryErr)
	}

	defer rows.Close() //nolint:errcheck

	return scanPartRows(rows)
}

// DefaultOpencodeDBPath returns the standard path to the OpenCode SQLite database.
func DefaultOpencodeDBPath() string {
	return defaultOpencodeDBPath()
}

// unexported constants.
const (
	opencodeURIPrefix = "opencode://"
	// partSelectSQL is the shared column list + join + session filter for the
	// parts query. queryPartsAfter / queryPartsBetween append their own
	// time bound and ORDER BY.
	partSelectSQL = "SELECT p.time_created, json_extract(p.data, '$.type'), " +
		"json_extract(p.data, '$.text'), json_extract(p.data, '$.tool'), " +
		"json_extract(p.data, '$.state'), json_extract(m.data, '$.role') " +
		"FROM part p LEFT JOIN message m ON p.message_id = m.id " +
		"WHERE p.session_id = ?"
	userRole = "user"
)

func buildJSONLLine(
	rowTime time.Time,
	partType, text, toolName, state, role sql.NullString,
) string {
	if !partType.Valid {
		return ""
	}

	switch partType.String {
	case "text":
		return buildTextJSONL(rowTime, text, role)
	case "tool":
		if !toolName.Valid || !state.Valid {
			return ""
		}

		return buildToolJSONL(rowTime, toolName.String, state.String)
	default:
		return ""
	}
}

func buildTextJSONL(rowTime time.Time, text, role sql.NullString) string {
	if !text.Valid || text.String == "" {
		return ""
	}

	msgRole := "assistant"
	if role.Valid && role.String == userRole {
		msgRole = userRole
	}

	return `{"type":"` + msgRole + `","timestamp":"` + rowTime.UTC().Format(time.RFC3339Nano) +
		`","message":{"role":"` + msgRole + `","content":[{"type":"text","text":` +
		mustMarshalJSON(text.String) + `}]}}`
}

func buildToolJSONL(rowTime time.Time, toolName, stateJSON string) string {
	var toolState map[string]any

	unmarshalErr := json.Unmarshal([]byte(stateJSON), &toolState)
	if unmarshalErr != nil {
		return ""
	}

	status, _ := toolState["status"].(string)
	if status == "" {
		return ""
	}

	role := "assistant"
	if status == "completed" {
		role = userRole
	}

	inputBytes, marshalErr := json.Marshal(toolState["input"])
	if marshalErr != nil {
		return ""
	}

	outputBytes, marshalErr := json.Marshal(toolState["output"])
	if marshalErr != nil {
		return ""
	}

	content := fmt.Sprintf(
		`[{"type":"tool_use","name":%s,"input":%s},{"type":"tool_result","output":%s}]`,
		mustMarshalJSON(toolName), inputBytes, outputBytes,
	)

	return `{"type":"` + role + `","timestamp":"` + rowTime.UTC().Format(time.RFC3339Nano) +
		`","message":{"role":"` + role + `","content":` + content + `}}`
}

func defaultOpencodeDBPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(home, ".local", "share", "opencode", "opencode.db")
}

func mustMarshalJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return `""`
	}

	return string(b)
}

// scanPartRows converts part-query rows into JSONL lines via buildJSONLLine,
// dropping rows that render empty. Shared by queryPartsAfter / queryPartsBetween.
func scanPartRows(rows *sql.Rows) ([]string, error) {
	jsonlLines := make([]string, 0)

	for rows.Next() {
		var timeCreated int64

		var partType, text, toolName, state, role sql.NullString

		scanErr := rows.Scan(&timeCreated, &partType, &text, &toolName, &state, &role)
		if scanErr != nil {
			return nil, fmt.Errorf("scanning part row: %w", scanErr)
		}

		rowTime := time.UnixMilli(timeCreated).UTC()

		line := buildJSONLLine(rowTime, partType, text, toolName, state, role)
		if line != "" {
			jsonlLines = append(jsonlLines, line)
		}
	}

	rowErr := rows.Err()
	if rowErr != nil {
		return nil, fmt.Errorf("iterating part rows: %w", rowErr)
	}

	return jsonlLines, nil
}
