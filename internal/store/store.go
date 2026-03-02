// Package store manages memory persistence and retrieval with full-text search.
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Exported variables.
var (
	ErrNotFound = errors.New("store: not found")
)

// DB abstracts database operations for dependency injection.
type DB interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Memory represents a stored memory record.
type Memory struct {
	ID              string
	Title           string
	Content         string
	ObservationType string
	Concepts        []string
	Principle       string
	AntiPattern     string
	Rationale       string
	EnrichedContent string
	Keywords        []string
	Confidence      string
	EnrichmentCount int
	ImpactScore     float64
	CreatedAt       time.Time
	UpdatedAt       time.Time
	LastSurfacedAt  *time.Time
	SurfacingCount  int
}

// SQLiteStore implements memory persistence using SQLite.
type SQLiteStore struct {
	db DB
}

// New initializes the schema and returns a new SQLiteStore.
func New(db DB) (*SQLiteStore, error) {
	_, err := db.ExecContext(context.Background(), schema)
	if err != nil {
		return nil, fmt.Errorf("store: create schema: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

// ClearSessionSurfacings removes all entries from the session surfacing log.
func (s *SQLiteStore) ClearSessionSurfacings(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM session_surfacings`)
	if err != nil {
		return fmt.Errorf("store: clear session surfacings: %w", err)
	}

	return nil
}

// Create inserts a new memory into the store.
func (s *SQLiteStore) Create(ctx context.Context, m *Memory) error {
	if m.Confidence == "" {
		return errEmptyConfidence
	}

	concepts, keywords := marshalFields(m)

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO memories (
			id, title, content, observation_type, concepts, principle,
			anti_pattern, rationale, enriched_content, keywords, confidence,
			enrichment_count, impact_score, created_at, updated_at,
			last_surfaced_at, surfacing_count
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.ID, m.Title, m.Content, m.ObservationType, string(concepts),
		m.Principle, m.AntiPattern, m.Rationale, m.EnrichedContent,
		string(keywords), m.Confidence, m.EnrichmentCount, m.ImpactScore,
		m.CreatedAt.Format(time.RFC3339), m.UpdatedAt.Format(time.RFC3339),
		FormatNullableTime(m.LastSurfacedAt), m.SurfacingCount,
	)
	if err != nil {
		return fmt.Errorf("store: create: %w", err)
	}
	// Sync FTS index
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO memories_fts (rowid, title, content, keywords, enriched_content)
		SELECT rowid, title, content, keywords, enriched_content FROM memories WHERE id = ?`,
		m.ID,
	)
	if err != nil {
		return fmt.Errorf("store: fts insert: %w", err)
	}

	return nil
}

// DecreaseImpact applies multiplicative decay to a memory's impact score.
// The new score is impact_score * factor, floored at 0.0.
// Returns ErrNotFound if the memory ID does not exist.
func (s *SQLiteStore) DecreaseImpact(ctx context.Context, memoryID string, factor float64) error {
	result, err := s.db.ExecContext(ctx,
		`UPDATE memories SET impact_score = MAX(0.0, impact_score * ?) WHERE id = ?`,
		factor, memoryID,
	)
	if err != nil {
		return fmt.Errorf("store: decrease impact: %w", err)
	}

	if result == nil {
		return errNilResult
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: decrease impact rows: %w", err)
	}

	if affected == 0 {
		return fmt.Errorf("%w: %s", ErrNotFound, memoryID)
	}

	return nil
}

// FindSimilar searches for memories matching the query and returns them sorted by relevance.
func (s *SQLiteStore) FindSimilar(
	ctx context.Context,
	query string,
	limit int,
) ([]ScoredMemory, error) {
	ftsQuery := ToFTS5Query(query)

	rows, err := s.db.QueryContext(ctx, `
		SELECT m.id, m.title, m.content, m.observation_type, m.concepts,
		       m.principle, m.anti_pattern, m.rationale, m.enriched_content,
		       m.keywords, m.confidence, m.enrichment_count, m.impact_score,
		       m.created_at, m.updated_at, m.last_surfaced_at, m.surfacing_count,
		       rank
		FROM memories_fts f
		JOIN memories m ON m.rowid = f.rowid
		WHERE memories_fts MATCH ?
		ORDER BY rank
		LIMIT ?`,
		ftsQuery, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("store: find similar: %w", err)
	}

	defer func() { _ = rows.Close() }()

	var results []ScoredMemory

	for rows.Next() {
		scored, err := scanScoredMemory(rows)
		if err != nil {
			return nil, fmt.Errorf("store: scan: %w", err)
		}
		// FTS5 rank is negative (lower = better match), negate for descending score
		scored.Score = -scored.Score
		results = append(results, scored)
	}

	return results, rows.Err() //nolint:wrapcheck // rows.Err wrapping adds uncoverable branch
}

// Get retrieves a memory by ID.
func (s *SQLiteStore) Get(ctx context.Context, id string) (*Memory, error) {
	var m Memory

	var conceptsJSON, keywordsJSON, createdStr, updatedStr string

	var lastSurfacedStr sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT id, title, content, observation_type, concepts, principle,
		       anti_pattern, rationale, enriched_content, keywords, confidence,
		       enrichment_count, impact_score, created_at, updated_at,
		       last_surfaced_at, surfacing_count
		FROM memories WHERE id = ?`, id).Scan(
		&m.ID, &m.Title, &m.Content, &m.ObservationType, &conceptsJSON,
		&m.Principle, &m.AntiPattern, &m.Rationale, &m.EnrichedContent,
		&keywordsJSON, &m.Confidence, &m.EnrichmentCount, &m.ImpactScore,
		&createdStr, &updatedStr, &lastSurfacedStr, &m.SurfacingCount,
	)
	if err != nil {
		return nil, fmt.Errorf("store: get: %w", err)
	}

	err = UnmarshalMemoryFields(
		&m, conceptsJSON, keywordsJSON, createdStr, updatedStr, lastSurfacedStr,
	)
	if err != nil {
		return nil, err
	}

	return &m, nil
}

// GetSessionSurfacings returns unique memory IDs from the session surfacing log.
func (s *SQLiteStore) GetSessionSurfacings(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT DISTINCT memory_id FROM session_surfacings`)
	if err != nil {
		return nil, fmt.Errorf("store: get session surfacings: %w", err)
	}

	defer func() { _ = rows.Close() }()

	var ids []string

	for rows.Next() {
		var id string

		err = rows.Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("store: scan surfacing: %w", err)
		}

		ids = append(ids, id)
	}

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("store: surfacing rows: %w", err)
	}

	if ids == nil {
		ids = make([]string, 0)
	}

	return ids, nil
}

// IncrementSurfacing increments the surfacing count and updates last surfaced time for memories by ID.
func (s *SQLiteStore) IncrementSurfacing(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	query, args := BuildIncrementQuery(ids)

	_, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("store: increment surfacing: %w", err)
	}

	return nil
}

// RecordSurfacing records memory IDs in the session surfacing log.
func (s *SQLiteStore) RecordSurfacing(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	now := time.Now().UTC().Format(time.RFC3339)

	for _, id := range ids {
		_, err := s.db.ExecContext(ctx,
			`INSERT INTO session_surfacings (memory_id, surfaced_at) VALUES (?, ?)`,
			id, now,
		)
		if err != nil {
			return fmt.Errorf("store: record surfacing: %w", err)
		}
	}

	return nil
}

// Surface returns memories ranked by frecency (recency and frequency weighted by impact score).
func (s *SQLiteStore) Surface(
	ctx context.Context,
	query string,
	limit int,
) ([]ScoredMemory, error) {
	ftsQuery := ToFTS5Query(query)

	rows, err := s.db.QueryContext(ctx, `
		SELECT m.id, m.title, m.content, m.observation_type, m.concepts,
		       m.principle, m.anti_pattern, m.rationale, m.enriched_content,
		       m.keywords, m.confidence, m.enrichment_count, m.impact_score,
		       m.created_at, m.updated_at, m.last_surfaced_at, m.surfacing_count,
		       2.0 * (1.0 / (1.0 + julianday('now') - julianday(
		           COALESCE(m.last_surfaced_at, m.updated_at, m.created_at)
		       ))) * m.impact_score / (
		           (1.0 / (1.0 + julianday('now') - julianday(
		               COALESCE(m.last_surfaced_at, m.updated_at, m.created_at)
		           ))) + m.impact_score
		       ) AS frecency
		FROM memories_fts f
		JOIN memories m ON m.rowid = f.rowid
		WHERE memories_fts MATCH ?
		ORDER BY frecency DESC,
		    CASE m.confidence WHEN 'A' THEN 3 WHEN 'B' THEN 2 ELSE 1 END DESC
		LIMIT ?`,
		ftsQuery, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("store: surface: %w", err)
	}

	defer func() { _ = rows.Close() }()

	var results []ScoredMemory

	for rows.Next() {
		scored, err := scanScoredMemory(rows)
		if err != nil {
			return nil, fmt.Errorf("store: scan surface: %w", err)
		}

		results = append(results, scored)
	}

	return results, rows.Err() //nolint:wrapcheck // rows.Err wrapping adds uncoverable branch
}

// Update modifies an existing memory in the store.
func (s *SQLiteStore) Update(ctx context.Context, m *Memory) error {
	concepts, keywords := marshalFields(m)
	// Delete old FTS tokens BEFORE updating the main table.
	// FTS5 external content tables look up current content from the
	// source table during DELETE, so the tokens must still match.
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM memories_fts WHERE rowid = (SELECT rowid FROM memories WHERE id = ?)`, m.ID)
	if err != nil {
		return fmt.Errorf("store: fts delete: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE memories SET
			title = ?, content = ?, observation_type = ?, concepts = ?,
			principle = ?, anti_pattern = ?, rationale = ?, enriched_content = ?,
			keywords = ?, confidence = ?, enrichment_count = ?, impact_score = ?,
			updated_at = ?, last_surfaced_at = ?, surfacing_count = ?
		WHERE id = ?`,
		m.Title, m.Content, m.ObservationType, string(concepts),
		m.Principle, m.AntiPattern, m.Rationale, m.EnrichedContent,
		string(keywords), m.Confidence, m.EnrichmentCount, m.ImpactScore,
		m.UpdatedAt.Format(time.RFC3339), FormatNullableTime(m.LastSurfacedAt),
		m.SurfacingCount, m.ID,
	)
	if err != nil {
		return fmt.Errorf("store: update: %w", err)
	}
	// Reinsert updated content into FTS index.
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO memories_fts (rowid, title, content, keywords, enriched_content)
		SELECT rowid, title, content, keywords, enriched_content FROM memories WHERE id = ?`,
		m.ID,
	)
	if err != nil {
		return fmt.Errorf("store: fts reinsert: %w", err)
	}

	return nil
}

// ScoredMemory pairs a memory with a relevance score.
type ScoredMemory struct {
	Memory Memory
	Score  float64
}

// BuildIncrementQuery constructs the SQL and args for incrementing surfacing counts.
func BuildIncrementQuery(ids []string) (string, []any) {
	now := time.Now().UTC().Format(time.RFC3339)
	placeholders := make([]string, len(ids))
	args := make([]any, 0, len(ids)+1)
	args = append(args, now)

	for i, id := range ids {
		placeholders[i] = "?"

		args = append(args, id)
	}

	query := fmt.Sprintf(`
		UPDATE memories SET
			surfacing_count = surfacing_count + 1,
			last_surfaced_at = ?
		WHERE id IN (%s)`, strings.Join(placeholders, ","))

	return query, args
}

// FormatNullableTime formats a *time.Time for SQL storage.
func FormatNullableTime(t *time.Time) any {
	if t == nil {
		return nil
	}

	return t.Format(time.RFC3339)
}

// ToFTS5Query converts a search string into an FTS5 OR query.
func ToFTS5Query(query string) string {
	words := strings.Fields(query)
	if len(words) == 0 {
		return `""`
	}

	quoted := make([]string, len(words))
	for i, w := range words {
		quoted[i] = `"` + strings.ReplaceAll(w, `"`, `""`) + `"`
	}

	return strings.Join(quoted, " OR ")
}

// UnmarshalMemoryFields populates a Memory from raw DB column values.
func UnmarshalMemoryFields(
	m *Memory,
	conceptsJSON, keywordsJSON, createdStr, updatedStr string,
	lastSurfacedStr sql.NullString,
) error {
	err := json.Unmarshal([]byte(conceptsJSON), &m.Concepts)
	if err != nil {
		return fmt.Errorf("store: unmarshal concepts: %w", err)
	}

	err = json.Unmarshal([]byte(keywordsJSON), &m.Keywords)
	if err != nil {
		return fmt.Errorf("store: unmarshal keywords: %w", err)
	}

	m.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	m.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)

	if lastSurfacedStr.Valid {
		t, _ := time.Parse(time.RFC3339, lastSurfacedStr.String)
		m.LastSurfacedAt = &t
	}

	return nil
}

// unexported constants.
const (
	schema = `
CREATE TABLE IF NOT EXISTS memories (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    observation_type TEXT,
    concepts TEXT,
    principle TEXT,
    anti_pattern TEXT,
    rationale TEXT,
    enriched_content TEXT,
    keywords TEXT,
    confidence TEXT NOT NULL,
    enrichment_count INTEGER DEFAULT 0,
    impact_score REAL DEFAULT 0.5,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    last_surfaced_at TEXT,
    surfacing_count INTEGER DEFAULT 0
);

CREATE VIRTUAL TABLE IF NOT EXISTS memories_fts USING fts5(
    title, content, keywords, enriched_content,
    content='memories', content_rowid='rowid'
);

CREATE TABLE IF NOT EXISTS session_surfacings (
    memory_id TEXT NOT NULL,
    surfaced_at TEXT NOT NULL
);
`
)

// unexported variables.
var (
	errEmptyConfidence = errors.New("store: confidence must not be empty")
	errNilResult       = errors.New("store: decrease impact: nil result")
)

// marshalFields serializes Concepts and Keywords to JSON.
// []string is always JSON-serializable, so errors are impossible.
func marshalFields(m *Memory) ([]byte, []byte) {
	concepts, _ := json.Marshal(m.Concepts) //nolint:errchkjson // []string never fails
	keywords, _ := json.Marshal(m.Keywords) //nolint:errchkjson // []string never fails

	return concepts, keywords
}

func scanScoredMemory(rows *sql.Rows) (ScoredMemory, error) {
	var scored ScoredMemory

	var conceptsJSON, keywordsJSON, createdStr, updatedStr string

	var lastSurfacedStr sql.NullString

	err := rows.Scan(
		&scored.Memory.ID, &scored.Memory.Title, &scored.Memory.Content,
		&scored.Memory.ObservationType, &conceptsJSON,
		&scored.Memory.Principle, &scored.Memory.AntiPattern,
		&scored.Memory.Rationale, &scored.Memory.EnrichedContent,
		&keywordsJSON, &scored.Memory.Confidence,
		&scored.Memory.EnrichmentCount, &scored.Memory.ImpactScore,
		&createdStr, &updatedStr, &lastSurfacedStr,
		&scored.Memory.SurfacingCount, &scored.Score,
	)
	if err != nil {
		return ScoredMemory{}, fmt.Errorf("scan: %w", err)
	}

	err = UnmarshalMemoryFields(
		&scored.Memory, conceptsJSON, keywordsJSON, createdStr, updatedStr, lastSurfacedStr,
	)
	if err != nil {
		return ScoredMemory{}, err
	}

	return scored, nil
}
