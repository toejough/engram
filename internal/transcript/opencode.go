package transcript

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
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

// Read reads a transcript using the first reader that recognizes the path.
func (r *CompositeTranscriptReader) Read(path string, budgetBytes int) (string, int, error) {
	for _, reader := range r.readers {
		content, size, err := reader.Read(path, budgetBytes)
		if err == nil {
			return content, size, nil
		}
	}

	return "", 0, fmt.Errorf("%w: %s", ErrNoReader, path)
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
			Path:   "opencode://" + id,
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

// Read reads an OpenCode session transcript, strips noise, and returns the
// content. The path must be an "opencode://<session_id>" URI.
func (r *OpencodeTranscriptReader) Read(path string, budgetBytes int) (string, int, error) {
	sessionID := strings.TrimPrefix(path, "opencode://")
	if sessionID == "" {
		return "", 0, ErrEmptySessionID
	}

	jsonlLines, queryErr := r.queryParts(sessionID)
	if queryErr != nil {
		return "", 0, queryErr
	}

	content, bytesUsed := r.stripAndBudget(jsonlLines, budgetBytes)

	return content, bytesUsed, nil
}

func (r *OpencodeTranscriptReader) queryParts(sessionID string) ([]string, error) {
	db, openErr := sql.Open("sqlite", r.dbPath)
	if openErr != nil {
		return nil, fmt.Errorf("opening opencode database: %w", openErr)
	}

	defer db.Close() //nolint:errcheck

	ctx := context.Background()

	rows, queryErr := db.QueryContext(
		ctx,
		"SELECT json_extract(p.data, '$.type'), json_extract(p.data, '$.text'), "+
			"json_extract(p.data, '$.tool'), json_extract(p.data, '$.state'), "+
			"json_extract(m.data, '$.role') "+
			"FROM part p LEFT JOIN message m ON p.message_id = m.id "+
			"WHERE p.session_id = ? "+
			"ORDER BY p.time_created",
		sessionID,
	)
	if queryErr != nil {
		return nil, fmt.Errorf("querying parts: %w", queryErr)
	}

	defer rows.Close() //nolint:errcheck

	jsonlLines := make([]string, 0)

	for rows.Next() {
		var partType, text, toolName, state, role sql.NullString

		scanErr := rows.Scan(&partType, &text, &toolName, &state, &role)
		if scanErr != nil {
			return nil, fmt.Errorf("scanning part row: %w", scanErr)
		}

		line := buildJSONLLine(partType, text, toolName, state, role)
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

func (r *OpencodeTranscriptReader) stripAndBudget(lines []string, budgetBytes int) (string, int) {
	cfg := sessionctx.StripConfig{ToolSummaryMode: true}
	stripped := sessionctx.StripWithConfig(lines, cfg)

	bytesUsed := 0
	startIdx := len(stripped)

	for i, v := range slices.Backward(stripped) {
		lineLen := len(v) + 1
		if bytesUsed+lineLen > budgetBytes && bytesUsed > 0 {
			break
		}

		startIdx = i
		bytesUsed += lineLen
	}

	var builder strings.Builder

	for _, line := range stripped[startIdx:] {
		builder.WriteString(line)
		builder.WriteByte('\n')
	}

	return builder.String(), bytesUsed
}

// DefaultOpencodeDBPath returns the standard path to the OpenCode SQLite database.
func DefaultOpencodeDBPath() string {
	return defaultOpencodeDBPath()
}

// unexported constants.
const (
	userRole = "user"
)

func buildJSONLLine(partType, text, toolName, state, role sql.NullString) string {
	if !partType.Valid {
		return ""
	}

	switch partType.String {
	case "text":
		return buildTextJSONL(text, role)
	case "tool":
		if !toolName.Valid || !state.Valid {
			return ""
		}

		return buildToolJSONL(toolName.String, state.String)
	default:
		return ""
	}
}

func buildTextJSONL(text, role sql.NullString) string {
	if !text.Valid || text.String == "" {
		return ""
	}

	msgRole := "assistant"
	if role.Valid && role.String == userRole {
		msgRole = userRole
	}

	return `{"type":"` + msgRole + `","message":{"role":"` + msgRole + `","content":[{"type":"text","text":` +
		mustMarshalJSON(
			text.String,
		) + `}]}}`
}

func buildToolJSONL(toolName, stateJSON string) string {
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

	return `{"type":"` + role + `","message":{"role":"` + role + `","content":` + content + `}}`
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
