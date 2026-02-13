package memory

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
	_ "github.com/mattn/go-sqlite3"
	ort "github.com/yalue/onnxruntime_go"
)

// embeddingDim is the dimension of the e5-small-v2 embeddings
const embeddingDim = 384

// e5SmallModelURL is the HuggingFace URL for the e5-small-v2 ONNX model
const e5SmallModelURL = "https://huggingface.co/intfloat/e5-small-v2/resolve/main/model.onnx"

// onnxRuntimeVersion is the version of ONNX Runtime to download
const onnxRuntimeVersion = "1.23.2"

var onnxRuntimeInitialized bool

// Session cache for ISSUE-48: Cache ONNX sessions across test functions
var (
	sessionCache     map[string]*ort.DynamicAdvancedSession
	sessionCacheMu   sync.RWMutex
	sessionInitCount int
	sessionInitMu    sync.Mutex
	sessionOnce      map[string]*sync.Once
	sessionOnceMu    sync.Mutex
	modelDownloadMu  sync.Mutex
)

func init() {
	// Auto-register sqlite-vec extension
	sqlite_vec.Auto()
	// ONNX Runtime will be initialized on first use

	// Initialize session cache
	sessionCache = make(map[string]*ort.DynamicAdvancedSession)
	sessionOnce = make(map[string]*sync.Once)
}

// GetSessionInitCount returns the number of times sessions have been initialized.
// This is a test-only function for verifying caching behavior.
func GetSessionInitCount() int {
	sessionInitMu.Lock()
	defer sessionInitMu.Unlock()
	return sessionInitCount
}

// ClearSessionCache clears the session cache and destroys all cached sessions.
// This is a test-only function for test isolation.
func ClearSessionCache() {
	sessionCacheMu.Lock()
	defer sessionCacheMu.Unlock()

	// Destroy all cached sessions
	for _, session := range sessionCache {
		_ = session.Destroy()
	}

	// Clear the caches
	sessionCache = make(map[string]*ort.DynamicAdvancedSession)

	sessionOnceMu.Lock()
	sessionOnce = make(map[string]*sync.Once)
	sessionOnceMu.Unlock()
}

// initEmbeddingsDB initializes the embeddings database with sqlite-vec.
func initEmbeddingsDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for better concurrent write handling
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Set busy timeout for concurrent writes (5 seconds)
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to set busy timeout: %w", err)
	}

	// Create embeddings table with vec0 virtual table
	createTable := `
		CREATE TABLE IF NOT EXISTS embeddings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			content TEXT NOT NULL,
			source TEXT NOT NULL,
			embedding_id INTEGER
		);

		CREATE VIRTUAL TABLE IF NOT EXISTS vec_embeddings USING vec0(
			embedding FLOAT[384]
		);

		CREATE TABLE IF NOT EXISTS generated_skills (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			slug TEXT NOT NULL UNIQUE,
			theme TEXT NOT NULL,
			description TEXT NOT NULL,
			content TEXT NOT NULL,
			source_memory_ids TEXT NOT NULL DEFAULT '',
			alpha REAL NOT NULL DEFAULT 1.0,
			beta REAL NOT NULL DEFAULT 1.0,
			utility REAL NOT NULL DEFAULT 0.5,
			retrieval_count INTEGER NOT NULL DEFAULT 0,
			last_retrieved TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			pruned INTEGER NOT NULL DEFAULT 0,
			embedding_id INTEGER
		);
	`

	if _, err := db.Exec(createTable); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	// Create FTS5 virtual table for full-text search (ISSUE-181)
	// FTS5 availability depends on SQLite compilation flags; degrade gracefully
	_, ftsErr := db.Exec(`CREATE VIRTUAL TABLE IF NOT EXISTS embeddings_fts USING fts5(content)`)
	if ftsErr == nil {
		// Migrate existing embeddings into FTS5 (idempotent)
		if err := migrateFTS5(db); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("failed to migrate FTS5: %w", err)
		}
	}

	// Add new columns via ALTER TABLE (ignore "duplicate column" errors)
	alterStatements := []string{
		"ALTER TABLE embeddings ADD COLUMN retrieval_count INTEGER NOT NULL DEFAULT 0",
		"ALTER TABLE embeddings ADD COLUMN last_retrieved TEXT",
		"ALTER TABLE embeddings ADD COLUMN projects_retrieved TEXT NOT NULL DEFAULT ''",
		"ALTER TABLE embeddings ADD COLUMN source_type TEXT NOT NULL DEFAULT 'internal'",
		"ALTER TABLE embeddings ADD COLUMN confidence REAL NOT NULL DEFAULT 1.0",
		"ALTER TABLE embeddings ADD COLUMN memory_type TEXT NOT NULL DEFAULT ''", // TASK-9: correction, reflection, or empty
		"ALTER TABLE embeddings ADD COLUMN retrieval_timestamps TEXT NOT NULL DEFAULT ''", // TASK-9: JSON array of RFC3339 timestamps
		"ALTER TABLE embeddings ADD COLUMN promoted INTEGER NOT NULL DEFAULT 0",  // ISSUE-184: 1 if in CLAUDE.md
		"ALTER TABLE embeddings ADD COLUMN promoted_at TEXT NOT NULL DEFAULT ''", // ISSUE-184: RFC3339 timestamp of promotion
		"ALTER TABLE embeddings ADD COLUMN observation_type TEXT NOT NULL DEFAULT ''",  // ISSUE-188: pattern, correction, preference, etc.
		"ALTER TABLE embeddings ADD COLUMN concepts TEXT NOT NULL DEFAULT ''",          // ISSUE-188: comma-separated concept tags
		"ALTER TABLE embeddings ADD COLUMN principle TEXT NOT NULL DEFAULT ''",         // ISSUE-188: extracted principle/rule
		"ALTER TABLE embeddings ADD COLUMN anti_pattern TEXT NOT NULL DEFAULT ''",      // ISSUE-188: what to avoid
		"ALTER TABLE embeddings ADD COLUMN rationale TEXT NOT NULL DEFAULT ''",         // ISSUE-188: why the principle matters
		"ALTER TABLE embeddings ADD COLUMN enriched_content TEXT NOT NULL DEFAULT ''",  // ISSUE-188: LLM-enriched content for embedding
		"ALTER TABLE embeddings ADD COLUMN model_version TEXT NOT NULL DEFAULT ''",     // ISSUE-221: Track which embedding model was used
		"ALTER TABLE embeddings ADD COLUMN flagged_for_review INTEGER NOT NULL DEFAULT 0",   // ISSUE-214: 1 if flagged for review
		"ALTER TABLE embeddings ADD COLUMN flagged_for_rewrite INTEGER NOT NULL DEFAULT 0",  // ISSUE-214: 1 if flagged for rewrite
		"ALTER TABLE generated_skills ADD COLUMN demoted_from_claude_md TEXT NOT NULL DEFAULT ''", // TASK-2: RFC3339 timestamp if demoted from CLAUDE.md
		"ALTER TABLE generated_skills ADD COLUMN claude_md_promoted INTEGER NOT NULL DEFAULT 0",   // TASK-3: 1 if promoted to CLAUDE.md
		"ALTER TABLE generated_skills ADD COLUMN promoted_at TEXT NOT NULL DEFAULT ''",             // TASK-3: RFC3339 timestamp of promotion
		"ALTER TABLE generated_skills ADD COLUMN merge_source_ids TEXT NOT NULL DEFAULT ''",        // TASK-10: JSON array of skill IDs that were merged into this one
		"ALTER TABLE generated_skills ADD COLUMN split_from_id INTEGER NOT NULL DEFAULT 0",         // TASK-10: Parent skill ID if this skill was created from a split
	}
	for _, stmt := range alterStatements {
		_, _ = db.Exec(stmt) // Ignore "duplicate column" errors
	}

	// ISSUE-184: Create metadata table for key-value storage (e.g., last_optimized_at)
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS metadata (key TEXT PRIMARY KEY, value TEXT)`)

	// ISSUE-214: Create feedback table
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS feedback (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		embedding_id INTEGER NOT NULL,
		feedback_type TEXT NOT NULL,
		created_at TEXT NOT NULL,
		FOREIGN KEY (embedding_id) REFERENCES embeddings(id)
	)`)

	return db, nil
}

// getMetadata retrieves a value from the metadata table. Returns empty string if key not found.
func getMetadata(db *sql.DB, key string) (string, error) {
	var value string
	err := db.QueryRow("SELECT value FROM metadata WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get metadata %q: %w", key, err)
	}
	return value, nil
}

// setMetadata sets a value in the metadata table (upsert).
func setMetadata(db *sql.DB, key, value string) error {
	_, err := db.Exec(
		"INSERT INTO metadata (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value",
		key, value,
	)
	if err != nil {
		return fmt.Errorf("failed to set metadata %q: %w", key, err)
	}
	return nil
}

// InitDBForTest is a test-accessible wrapper that initializes and returns the embeddings DB.
// The caller is responsible for closing the returned *sql.DB.
func InitDBForTest(memoryRoot string) (*sql.DB, error) {
	dbPath := filepath.Join(memoryRoot, "embeddings.db")
	return initEmbeddingsDB(dbPath)
}

// GetMetadataForTest is a test-accessible wrapper around getMetadata.
func GetMetadataForTest(memoryRoot, key string) (string, error) {
	dbPath := filepath.Join(memoryRoot, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	if err != nil {
		return "", err
	}
	defer func() { _ = db.Close() }()
	return getMetadata(db, key)
}

// SetMetadataForTest is a test-accessible wrapper around setMetadata.
func SetMetadataForTest(memoryRoot, key, value string) error {
	dbPath := filepath.Join(memoryRoot, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	return setMetadata(db, key, value)
}

// hasFTS5 checks whether the embeddings_fts table exists in the database.
func hasFTS5(db *sql.DB) bool {
	var name string
	err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='embeddings_fts'").Scan(&name)
	return err == nil && name == "embeddings_fts"
}

// insertFTS5 inserts a row into the FTS5 table if it exists. Errors are silently ignored.
func insertFTS5(db *sql.DB, rowID int64, content string) {
	if !hasFTS5(db) {
		return
	}
	_, _ = db.Exec(`INSERT INTO embeddings_fts(rowid, content) VALUES (?, ?)`, rowID, content)
}

// deleteFTS5 deletes a row from the FTS5 table if it exists. Errors are silently ignored.
func deleteFTS5(db *sql.DB, rowID int64) {
	if !hasFTS5(db) {
		return
	}
	_, _ = db.Exec(`DELETE FROM embeddings_fts WHERE rowid = ?`, rowID)
}

// migrateFTS5 populates the FTS5 table from existing embeddings on first access.
// Idempotent: uses INSERT OR IGNORE pattern by checking for existing rowids.
func migrateFTS5(db *sql.DB) error {
	// Only insert rows that aren't already in FTS5.
	// FTS5 rowid matches embeddings.id.
	_, err := db.Exec(`
		INSERT INTO embeddings_fts(rowid, content)
		SELECT e.id, e.content FROM embeddings e
		WHERE e.id NOT IN (SELECT rowid FROM embeddings_fts)
	`)
	return err
}

// migrateModelVersion re-embeds all entries with the new e5-small-v2 model (ISSUE-221).
// Idempotent: checks metadata "model_version_migrated" to skip if already done.
func migrateModelVersion(db *sql.DB, modelPath string) error {
	// Check if migration already done
	migrated, err := getMetadata(db, "model_version_migrated")
	if err != nil {
		return fmt.Errorf("failed to check migration status: %w", err)
	}
	if migrated == "e5-small-v2" {
		return nil // Already migrated
	}

	// Query all embeddings that need migration (where model_version != 'e5-small-v2' or empty)
	rows, err := db.Query(`SELECT id, content FROM embeddings WHERE model_version != 'e5-small-v2' OR model_version = ''`)
	if err != nil {
		return fmt.Errorf("failed to query embeddings for migration: %w", err)
	}
	defer rows.Close()

	type entry struct {
		id      int64
		content string
	}
	var entries []entry
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.id, &e.content); err != nil {
			return fmt.Errorf("failed to scan entry: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating rows: %w", err)
	}

	// Re-embed each entry with "passage: " prefix
	for _, e := range entries {
		// Generate new embedding with passage prefix
		embedding, _, _, err := generateEmbeddingONNX("passage: "+e.content, modelPath)
		if err != nil {
			// Log error but continue with other entries
			fmt.Fprintf(os.Stderr, "Warning: failed to re-embed entry %d: %v\n", e.id, err)
			continue
		}

		embeddingBlob, err := sqlite_vec.SerializeFloat32(embedding)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to serialize embedding for entry %d: %v\n", e.id, err)
			continue
		}

		// Get current embedding_id
		var oldEmbeddingID int64
		err = db.QueryRow("SELECT embedding_id FROM embeddings WHERE id = ?", e.id).Scan(&oldEmbeddingID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to get embedding_id for entry %d: %v\n", e.id, err)
			continue
		}

		// Delete old vec row
		if _, err := db.Exec("DELETE FROM vec_embeddings WHERE rowid = ?", oldEmbeddingID); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to delete old vec row for entry %d: %v\n", e.id, err)
			continue
		}

		// Insert new vec row
		result, err := db.Exec("INSERT INTO vec_embeddings(embedding) VALUES (?)", embeddingBlob)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to insert new vec row for entry %d: %v\n", e.id, err)
			continue
		}
		newEmbeddingID, _ := result.LastInsertId()

		// Update metadata with new embedding_id and model_version
		_, err = db.Exec("UPDATE embeddings SET embedding_id = ?, model_version = 'e5-small-v2' WHERE id = ?",
			newEmbeddingID, e.id)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to update metadata for entry %d: %v\n", e.id, err)
			continue
		}
	}

	// Mark migration as complete
	if err := setMetadata(db, "model_version_migrated", "e5-small-v2"); err != nil {
		return fmt.Errorf("failed to set migration metadata: %w", err)
	}

	return nil
}

// searchBM25 performs full-text search using FTS5 BM25 ranking.
func searchBM25(db *sql.DB, queryText string, limit int) ([]QueryResult, error) {
	// Sanitize query: strip FTS5 operators and special chars, keep only words
	sanitized := sanitizeFTS5Query(queryText)
	if sanitized == "" {
		return nil, nil
	}

	// ISSUE-188: join back to embeddings to get retrieval_count and projects_retrieved
	// ISSUE-214: also fetch e.id for feedback
	rows, err := db.Query(
		`SELECT COALESCE(e.id, 0), f.content, f.rank, COALESCE(e.memory_type, ''),
		        COALESCE(e.retrieval_count, 0), COALESCE(e.projects_retrieved, '')
		 FROM embeddings_fts f
		 LEFT JOIN embeddings e ON e.id = f.rowid
		 WHERE embeddings_fts MATCH ? ORDER BY f.rank LIMIT ?`,
		sanitized, limit,
	)
	if err != nil {
		// FTS5 syntax errors → return empty results gracefully
		return nil, nil
	}
	defer func() { _ = rows.Close() }()

	var results []QueryResult
	for rows.Next() {
		var id int64
		var content string
		var rank float64
		var memoryType string
		var retrievalCount int
		var projectsRaw string
		if err := rows.Scan(&id, &content, &rank, &memoryType, &retrievalCount, &projectsRaw); err != nil {
			continue
		}
		// FTS5 rank is negative (lower = better match). Normalize to positive score.
		score := -rank
		if score < 0 {
			score = 0
		}
		results = append(results, QueryResult{
			ID:                id,
			Content:           content,
			Score:             score,
			Source:            "memory",
			MemoryType:        memoryType,
			RetrievalCount:    retrievalCount,
			ProjectsRetrieved: parseProjectsList(projectsRaw),
			MatchType:         "bm25",
		})
	}

	return results, rows.Err()
}

// sanitizeFTS5Query strips FTS5 operators and special characters, keeping only words.
func sanitizeFTS5Query(query string) string {
	var words []string
	for _, word := range strings.Fields(query) {
		// Strip non-alphanumeric characters
		cleaned := strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
				return r
			}
			return -1
		}, word)
		// Skip FTS5 operators and empty strings
		upper := strings.ToUpper(cleaned)
		if cleaned != "" && upper != "AND" && upper != "OR" && upper != "NOT" && upper != "NEAR" {
			words = append(words, cleaned)
		}
	}
	return strings.Join(words, " ")
}

// hybridSearch combines vector similarity and BM25 full-text search using Reciprocal Rank Fusion.
func hybridSearch(db *sql.DB, queryEmbedding []float32, queryText string, limit int, k int) ([]QueryResult, error) {
	// Fetch 2*limit from each source
	fetchLimit := 2 * limit

	vectorResults, err := searchSimilar(db, queryEmbedding, fetchLimit)
	if err != nil {
		return nil, err
	}

	bm25Results, _ := searchBM25(db, queryText, fetchLimit) // Ignore BM25 errors, fall back to vector-only

	// If no BM25 results, just return vector results (truncated to limit)
	if len(bm25Results) == 0 {
		if len(vectorResults) > limit {
			return vectorResults[:limit], nil
		}
		return vectorResults, nil
	}

	// ISSUE-188: Track which source lists each result comes from
	type rrfEntry struct {
		result   QueryResult
		rrfScore float64
		inVector bool
		inBM25   bool
	}

	scoreMap := make(map[string]*rrfEntry)

	// Add vector results (already sorted by score DESC, rank = position)
	for i, r := range vectorResults {
		rank := i + 1 // 1-indexed
		entry, exists := scoreMap[r.Content]
		if !exists {
			entry = &rrfEntry{result: r}
			scoreMap[r.Content] = entry
		}
		entry.rrfScore += 1.0 / float64(k+rank)
		entry.inVector = true
	}

	// Add BM25 results
	for i, r := range bm25Results {
		rank := i + 1
		entry, exists := scoreMap[r.Content]
		if !exists {
			entry = &rrfEntry{result: r}
			scoreMap[r.Content] = entry
		}
		entry.rrfScore += 1.0 / float64(k+rank)
		entry.inBM25 = true
	}

	// Collect and sort by RRF score DESC
	entries := make([]rrfEntry, 0, len(scoreMap))
	for _, e := range scoreMap {
		entries = append(entries, *e)
	}

	// Sort by RRF score descending
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[j].rrfScore > entries[i].rrfScore {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	// Return top limit results with RRF score and MatchType based on provenance
	results := make([]QueryResult, 0, limit)
	for i := 0; i < len(entries) && i < limit; i++ {
		r := entries[i].result
		r.Score = entries[i].rrfScore
		// ISSUE-188: Set MatchType based on which source lists the result appeared in
		switch {
		case entries[i].inVector && entries[i].inBM25:
			r.MatchType = "hybrid"
		case entries[i].inVector:
			r.MatchType = "vector"
		case entries[i].inBM25:
			r.MatchType = "bm25"
		}
		results = append(results, r)
	}

	return results, nil
}



// downloadONNXRuntime downloads and extracts the ONNX Runtime library.
func downloadONNXRuntime(modelDir string) (string, error) {
	// Determine OS-specific library name and download URL
	var libName, archiveURL string
	switch runtime.GOOS {
	case "darwin":
		if runtime.GOARCH == "arm64" {
			libName = "libonnxruntime.dylib"
			archiveURL = fmt.Sprintf("https://github.com/microsoft/onnxruntime/releases/download/v%s/onnxruntime-osx-arm64-%s.tgz", onnxRuntimeVersion, onnxRuntimeVersion)
		} else {
			libName = "libonnxruntime.dylib"
			archiveURL = fmt.Sprintf("https://github.com/microsoft/onnxruntime/releases/download/v%s/onnxruntime-osx-x86_64-%s.tgz", onnxRuntimeVersion, onnxRuntimeVersion)
		}
	case "linux":
		libName = "libonnxruntime.so"
		archiveURL = fmt.Sprintf("https://github.com/microsoft/onnxruntime/releases/download/v%s/onnxruntime-linux-x64-%s.tgz", onnxRuntimeVersion, onnxRuntimeVersion)
	case "windows":
		libName = "onnxruntime.dll"
		archiveURL = fmt.Sprintf("https://github.com/microsoft/onnxruntime/releases/download/v%s/onnxruntime-win-x64-%s.zip", onnxRuntimeVersion, onnxRuntimeVersion)
	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	libPath := filepath.Join(modelDir, libName)

	// Check if library already exists
	if _, err := os.Stat(libPath); err == nil {
		return libPath, nil
	}

	// Download the archive
	resp, err := http.Get(archiveURL)
	if err != nil {
		return "", fmt.Errorf("failed to download ONNX Runtime: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download ONNX Runtime: HTTP %d", resp.StatusCode)
	}

	// Extract the library from the archive
	if runtime.GOOS == "windows" {
		return "", fmt.Errorf("windows ZIP extraction not implemented")
	}

	// Extract from tar.gz
	gzr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer func() { _ = gzr.Close() }()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("failed to read tar: %w", err)
		}

		// Look for the versioned library file (e.g., libonnxruntime.1.23.2.dylib)
		// Skip symlinks and only extract the actual file
		baseName := filepath.Base(header.Name)
		if strings.Contains(header.Name, "/lib/") &&
			strings.HasPrefix(baseName, "libonnxruntime") &&
			strings.HasSuffix(baseName, ".dylib") &&
			header.Typeflag != tar.TypeSymlink &&
			header.Typeflag != tar.TypeDir {

			outFile, err := os.Create(libPath)
			if err != nil {
				return "", fmt.Errorf("failed to create library file: %w", err)
			}

			if _, err := io.Copy(outFile, tr); err != nil {
				_ = outFile.Close()
				return "", fmt.Errorf("failed to extract library: %w", err)
			}

			_ = outFile.Close()

			// Make executable on Unix-like systems
			if runtime.GOOS != "windows" {
				_ = os.Chmod(libPath, 0755)
			}

			return libPath, nil
		}
	}

	return "", fmt.Errorf("library not found in archive")
}

// ensureCorrectModel checks if the cached model matches the current e5SmallModelURL.
// If the URL has changed (stale model), it deletes the cached model and returns true.
// Returns (needsDownload, error).
func ensureCorrectModel(db *sql.DB, modelPath string) (bool, error) {
	// Check if model file exists
	_, statErr := os.Stat(modelPath)
	if os.IsNotExist(statErr) {
		// Set metadata for new databases
		if err := setMetadata(db, "model_url", e5SmallModelURL); err != nil {
			return false, fmt.Errorf("failed to set model_url metadata: %w", err)
		}
		return true, nil // Model doesn't exist, needs download
	}

	// Check metadata for current model URL
	currentURL, err := getMetadata(db, "model_url")
	if err != nil {
		return false, fmt.Errorf("failed to get model_url metadata: %w", err)
	}

	// If this is a new database (empty metadata), just set it and use existing model
	if currentURL == "" {
		if err := setMetadata(db, "model_url", e5SmallModelURL); err != nil {
			return false, fmt.Errorf("failed to set model_url metadata: %w", err)
		}
		return false, nil // Model exists, no download needed
	}

	// If metadata exists and doesn't match current URL, delete stale model
	if currentURL != e5SmallModelURL {
		if err := os.Remove(modelPath); err != nil && !os.IsNotExist(err) {
			return false, fmt.Errorf("failed to delete stale model: %w", err)
		}
		// Update metadata to new URL
		if err := setMetadata(db, "model_url", e5SmallModelURL); err != nil {
			return false, fmt.Errorf("failed to set model_url metadata: %w", err)
		}
		return true, nil // Stale model deleted, needs download
	}

	return false, nil // Model is up to date
}

// downloadModel downloads the e5-small-v2 model from HuggingFace if not already present.
// Thread-safe: uses mutex to prevent concurrent downloads.
func downloadModel(modelPath string) error {
	// Fast path: check without lock
	if _, err := os.Stat(modelPath); err == nil {
		return nil // Model already exists
	}

	// Acquire lock for download
	modelDownloadMu.Lock()
	defer modelDownloadMu.Unlock()

	// Check again after acquiring lock (another goroutine may have downloaded)
	if _, err := os.Stat(modelPath); err == nil {
		return nil // Model already exists
	}

	// Create model directory
	modelDir := filepath.Dir(modelPath)
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		return fmt.Errorf("failed to create model directory: %w", err)
	}

	// Download the model
	resp, err := http.Get(e5SmallModelURL)
	if err != nil {
		return fmt.Errorf("failed to download model: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download model: HTTP %d", resp.StatusCode)
	}

	// Create temp file
	tempPath := modelPath + ".tmp"
	out, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() { _ = out.Close() }()

	// Copy model data
	if _, err := io.Copy(out, resp.Body); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to save model: %w", err)
	}

	// Close the file before renaming (required on Windows)
	_ = out.Close()

	// Rename temp file to final path
	if err := os.Rename(tempPath, modelPath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to finalize model: %w", err)
	}

	return nil
}

// initializeONNXRuntime initializes the ONNX Runtime environment with the downloaded library.
func initializeONNXRuntime(modelDir string) error {
	if onnxRuntimeInitialized {
		return nil
	}

	// Download ONNX Runtime library
	libPath, err := downloadONNXRuntime(modelDir)
	if err != nil {
		return fmt.Errorf("failed to download ONNX Runtime: %w", err)
	}

	// Set the library path
	ort.SetSharedLibraryPath(libPath)

	// Initialize the environment
	if err := ort.InitializeEnvironment(); err != nil {
		return fmt.Errorf("failed to initialize ONNX Runtime: %w", err)
	}

	onnxRuntimeInitialized = true
	return nil
}

// getOrCreateSession retrieves a cached session or creates a new one.
// Returns the session and whether it was newly created in THIS call.
func getOrCreateSession(modelPath string) (*ort.DynamicAdvancedSession, bool, error) {
	// Try read lock first for fast path (session already exists)
	sessionCacheMu.RLock()
	if session, exists := sessionCache[modelPath]; exists {
		sessionCacheMu.RUnlock()
		return session, false, nil
	}
	sessionCacheMu.RUnlock()

	// Get or create the sync.Once for this model path
	sessionOnceMu.Lock()
	once, exists := sessionOnce[modelPath]
	if !exists {
		once = &sync.Once{}
		sessionOnce[modelPath] = once
	}
	sessionOnceMu.Unlock()

	// Track if we actually created the session in THIS goroutine
	var session *ort.DynamicAdvancedSession
	var err error
	createdInThisCall := false

	once.Do(func() {
		session, err = ort.NewDynamicAdvancedSession(modelPath,
			[]string{"input_ids", "attention_mask", "token_type_ids"},
			[]string{"last_hidden_state"},
			nil)
		if err != nil {
			return
		}

		// Store in cache
		sessionCacheMu.Lock()
		sessionCache[modelPath] = session
		sessionCacheMu.Unlock()

		// Increment init count
		sessionInitMu.Lock()
		sessionInitCount++
		sessionInitMu.Unlock()

		createdInThisCall = true
	})

	if err != nil {
		return nil, false, fmt.Errorf("failed to create ONNX session: %w", err)
	}

	// If we didn't create it in this call, retrieve from cache
	if !createdInThisCall {
		sessionCacheMu.RLock()
		session = sessionCache[modelPath]
		sessionCacheMu.RUnlock()
	}

	return session, createdInThisCall, nil
}

// generateEmbeddingONNX generates an embedding using the e5-small-v2 ONNX model.
// Returns the embedding, whether a new session was created, and whether a session was reused.
func generateEmbeddingONNX(text string, modelPath string) ([]float32, bool, bool, error) {
	// Use WordPiece tokenizer
	tokenizer := NewTokenizer()
	tokens := tokenizer.Tokenize(text)

	// Prepare input tensors with padding/truncation
	inputSize := 512 // Max sequence length for e5-small-v2
	inputIDs := make([]int64, inputSize)
	attentionMask := make([]int64, inputSize)
	tokenTypeIDs := make([]int64, inputSize) // All zeros for single-sequence input

	// Fill input arrays with token IDs (truncate or pad as needed)
	for i := 0; i < inputSize; i++ {
		if i < len(tokens) {
			inputIDs[i] = tokens[i]
			attentionMask[i] = 1 // Mark as valid token
		}
		// tokenTypeIDs remains 0 (already initialized)
	}

	// Get or create cached session
	session, sessionCreated, err := getOrCreateSession(modelPath)
	if err != nil {
		return nil, false, false, err
	}
	// Don't destroy the session - it's cached for reuse

	// Create input tensors
	inputShape := ort.NewShape(1, int64(inputSize))
	inputIDsTensor, err := ort.NewTensor(inputShape, inputIDs)
	if err != nil {
		return nil, false, false, fmt.Errorf("failed to create input_ids tensor: %w", err)
	}
	defer func() { _ = inputIDsTensor.Destroy() }()

	attentionMaskTensor, err := ort.NewTensor(inputShape, attentionMask)
	if err != nil {
		return nil, false, false, fmt.Errorf("failed to create attention_mask tensor: %w", err)
	}
	defer func() { _ = attentionMaskTensor.Destroy() }()

	tokenTypeIDsTensor, err := ort.NewTensor(inputShape, tokenTypeIDs)
	if err != nil {
		return nil, false, false, fmt.Errorf("failed to create token_type_ids tensor: %w", err)
	}
	defer func() { _ = tokenTypeIDsTensor.Destroy() }()

	// Create output tensor
	outputShape := ort.NewShape(1, int64(inputSize), int64(embeddingDim))
	outputTensor, err := ort.NewEmptyTensor[float32](outputShape)
	if err != nil {
		return nil, false, false, fmt.Errorf("failed to create output tensor: %w", err)
	}
	defer func() { _ = outputTensor.Destroy() }()

	// Run inference
	inputs := []ort.Value{inputIDsTensor, attentionMaskTensor, tokenTypeIDsTensor}
	outputs := []ort.Value{outputTensor}
	if err := session.Run(inputs, outputs); err != nil {
		return nil, false, false, fmt.Errorf("failed to run inference: %w", err)
	}

	// Extract embeddings from output
	// The output is [batch_size, sequence_length, hidden_size]
	// We need to pool it to get [batch_size, hidden_size]
	outputData := outputTensor.GetData()

	// Mean pooling over sequence dimension
	embedding := make([]float32, embeddingDim)

	// Simple mean pooling
	validTokens := 0
	for i := 0; i < inputSize; i++ {
		if attentionMask[i] != 0 {
			validTokens++
			for j := 0; j < embeddingDim; j++ {
				idx := i*embeddingDim + j
				if idx < len(outputData) {
					embedding[j] += outputData[idx]
				}
			}
		}
	}

	// Average the pooled values
	if validTokens > 0 {
		for j := 0; j < embeddingDim; j++ {
			embedding[j] /= float32(validTokens)
		}
	}

	// Normalize to unit vector
	var magnitude float32
	for _, v := range embedding {
		magnitude += v * v
	}
	magnitude = float32(math.Sqrt(float64(magnitude)))
	if magnitude > 0 {
		for i := range embedding {
			embedding[i] /= magnitude
		}
	}

	sessionReused := !sessionCreated

	return embedding, sessionCreated, sessionReused, nil
}

// searchSimilar finds the most similar embeddings using cosine similarity.
func searchSimilar(db *sql.DB, queryEmbedding []float32, limit int) ([]QueryResult, error) {
	// Use sqlite-vec's distance function for similarity search
	// Weight by confidence for TASK-43
	// ISSUE-188: also fetch retrieval_count and projects_retrieved
	// ISSUE-214: also fetch e.id for feedback
	query := `
		SELECT e.id, e.content, e.source, e.source_type, e.confidence, e.memory_type,
		       e.retrieval_count, e.projects_retrieved,
		       (1 - vec_distance_cosine(v.embedding, ?)) * e.confidence as score
		FROM vec_embeddings v
		JOIN embeddings e ON e.embedding_id = v.rowid
		ORDER BY score DESC
		LIMIT ?
	`

	queryBlob, err := sqlite_vec.SerializeFloat32(queryEmbedding)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(query, queryBlob, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var results []QueryResult
	for rows.Next() {
		var r QueryResult
		var projectsRaw string
		if err := rows.Scan(&r.ID, &r.Content, &r.Source, &r.SourceType, &r.Confidence, &r.MemoryType,
			&r.RetrievalCount, &projectsRaw, &r.Score); err != nil {
			return nil, err
		}
		// Parse comma-separated projects into slice
		r.ProjectsRetrieved = parseProjectsList(projectsRaw)
		r.MatchType = "vector"
		// Clamp score to [0, 1] due to floating point precision
		if r.Score > 1.0 {
			r.Score = 1.0
		}
		if r.Score < 0.0 {
			r.Score = 0.0
		}
		results = append(results, r)
	}

	return results, rows.Err()
}

// parseProjectsList splits a comma-separated projects string into a slice,
// trimming whitespace and filtering empty entries.
func parseProjectsList(raw string) []string {
	if raw == "" {
		return nil
	}
	var projects []string
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			projects = append(projects, p)
		}
	}
	return projects
}

// rawSimilarResult holds a raw (non-confidence-weighted) similarity result.
type rawSimilarResult struct {
	id         int64
	content    string
	similarity float64
}

// searchRawSimilar finds the most similar embeddings using raw cosine similarity
// (NOT weighted by confidence). Used for dedup checks where we want to compare
// semantic content regardless of confidence level.
func searchRawSimilar(db *sql.DB, queryEmbedding []float32, limit int) ([]rawSimilarResult, error) {
	query := `
		SELECT e.id, e.content,
		       (1 - vec_distance_cosine(v.embedding, ?)) as similarity
		FROM vec_embeddings v
		JOIN embeddings e ON e.embedding_id = v.rowid
		ORDER BY similarity DESC
		LIMIT ?
	`

	queryBlob, err := sqlite_vec.SerializeFloat32(queryEmbedding)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(query, queryBlob, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var results []rawSimilarResult
	for rows.Next() {
		var r rawSimilarResult
		if err := rows.Scan(&r.id, &r.content, &r.similarity); err != nil {
			return nil, err
		}
		results = append(results, r)
	}

	return results, rows.Err()
}

// rawResultsToExistingMemories converts raw similarity results to ExistingMemory structs
// for passing to the LLM Decide method.
func rawResultsToExistingMemories(results []rawSimilarResult) []ExistingMemory {
	existing := make([]ExistingMemory, 0, len(results))
	for _, r := range results {
		existing = append(existing, ExistingMemory{
			ID:         r.id,
			Content:    r.content,
			Similarity: r.similarity,
		})
	}
	return existing
}

// updateEmbeddingContent replaces an existing embedding's content and re-embeds it.
// Sequence: delete old vec row, insert new vec row, update metadata, refresh FTS5.
func updateEmbeddingContent(db *sql.DB, id int64, newContent string, newEmbedding []byte,
	observationType, conceptsCSV, principle, antiPattern, rationale, enrichedContent string) error {
	// Get the current embedding_id for the old vec row
	var oldEmbeddingID int64
	err := db.QueryRow("SELECT embedding_id FROM embeddings WHERE id = ?", id).Scan(&oldEmbeddingID)
	if err != nil {
		return fmt.Errorf("updateEmbeddingContent: failed to get old embedding_id: %w", err)
	}

	// Delete old vec row
	if _, err := db.Exec("DELETE FROM vec_embeddings WHERE rowid = ?", oldEmbeddingID); err != nil {
		return fmt.Errorf("updateEmbeddingContent: failed to delete old vec row: %w", err)
	}

	// Insert new vec row
	result, err := db.Exec("INSERT INTO vec_embeddings(embedding) VALUES (?)", newEmbedding)
	if err != nil {
		return fmt.Errorf("updateEmbeddingContent: failed to insert new vec row: %w", err)
	}
	newEmbeddingID, _ := result.LastInsertId()

	// Update metadata
	_, err = db.Exec(`UPDATE embeddings SET content = ?, embedding_id = ?,
		observation_type = ?, concepts = ?, principle = ?, anti_pattern = ?,
		rationale = ?, enriched_content = ?
		WHERE id = ?`,
		newContent, newEmbeddingID,
		observationType, conceptsCSV, principle, antiPattern, rationale, enrichedContent,
		id)
	if err != nil {
		return fmt.Errorf("updateEmbeddingContent: failed to update metadata: %w", err)
	}

	// Refresh FTS5
	deleteFTS5(db, id)
	insertFTS5(db, id, newContent)

	return nil
}

// supersedeEmbedding soft-deletes a memory by setting confidence to 0.
// The existing prune pipeline in optimize.go cleans up zero-confidence entries.
func supersedeEmbedding(db *sql.DB, id int64) error {
	_, err := db.Exec("UPDATE embeddings SET confidence = 0 WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("supersedeEmbedding: %w", err)
	}
	deleteFTS5(db, id)
	return nil
}

// boostExistingConfidence increments an existing memory's confidence by 0.05 (capped at 1.0).
func boostExistingConfidence(db *sql.DB, id int64) error {
	_, err := db.Exec("UPDATE embeddings SET confidence = MIN(1.0, confidence + 0.05) WHERE id = ?", id)
	return err
}

// learnToEmbeddings creates an embedding for a learning entry in the DB.
func learnToEmbeddings(opts LearnOpts) error {
	// Determine model directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	modelDir := filepath.Join(homeDir, ".claude", "models")

	// Ensure model directory exists
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		return fmt.Errorf("failed to create model directory: %w", err)
	}

	// Initialize ONNX Runtime
	if err := initializeONNXRuntime(modelDir); err != nil {
		return fmt.Errorf("failed to initialize ONNX Runtime: %w", err)
	}

	// Model path
	modelPath := filepath.Join(modelDir, "e5-small-v2.onnx")

	// Download model if needed
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		if err := downloadModel(modelPath); err != nil {
			return fmt.Errorf("failed to download model: %w", err)
		}
	}

	// Initialize embeddings database
	dbPath := filepath.Join(opts.MemoryRoot, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize embeddings database: %w", err)
	}
	defer func() { _ = db.Close() }()

	// ISSUE-221: Check for stale model and re-download if needed
	needsDownload, err := ensureCorrectModel(db, modelPath)
	if err != nil {
		return fmt.Errorf("failed to check model validity: %w", err)
	}
	if needsDownload {
		if err := downloadModel(modelPath); err != nil {
			return fmt.Errorf("failed to download model: %w", err)
		}
	}

	// ISSUE-221: Run model version migration if needed
	if err := migrateModelVersion(db, modelPath); err != nil {
		return fmt.Errorf("failed to migrate model version: %w", err)
	}

	// Determine source_type and confidence
	sourceType := opts.Source
	if sourceType == "" {
		sourceType = "internal"
	}
	confidence := 1.0
	if sourceType == "external" {
		confidence = 0.7
	}

	// Build formatted entry for DB storage and display
	timestamp := time.Now().Format("2006-01-02 15:04")
	var content string
	if opts.Project != "" {
		content = fmt.Sprintf("- %s: [%s] %s", timestamp, opts.Project, opts.Message)
	} else {
		content = fmt.Sprintf("- %s: %s", timestamp, opts.Message)
	}

	// ISSUE-188: Attempt LLM extraction if extractor is provided
	var observationType, conceptsCSV, principle, antiPattern, rationale, enrichedContent string
	if opts.Extractor != nil {
		obs, extractErr := opts.Extractor.Extract(context.Background(), opts.Message)
		if extractErr == nil && obs != nil {
			observationType = obs.Type
			conceptsCSV = strings.Join(obs.Concepts, ",")
			principle = obs.Principle
			antiPattern = obs.AntiPattern
			rationale = obs.Rationale
			enrichedContent = fmt.Sprintf("[%s] %s - Context: %s", obs.Type, obs.Principle, obs.Rationale)
		}
		// On failure (including ErrLLMUnavailable): fall through with empty strings
	}

	// ISSUE-216: Write-time validation after LLM enrichment but before DB insert
	if opts.Extractor != nil {
		// Validate metadata when LLM was used
		var validationErrors []string

		// enriched_content non-empty OR observation_type non-empty
		if enrichedContent == "" && observationType == "" {
			validationErrors = append(validationErrors, "enriched_content and observation_type are both empty")
		}

		// concepts has at least one entry
		if conceptsCSV == "" {
			validationErrors = append(validationErrors, "concepts is empty")
		}

		// confidence in [0.0, 1.0] (already validated by sourceType logic, but double-check)
		if confidence < 0.0 || confidence > 1.0 {
			validationErrors = append(validationErrors, fmt.Sprintf("confidence out of range: %f", confidence))
		}

		if len(validationErrors) > 0 {
			return fmt.Errorf("LLM enrichment validation failed: %s", strings.Join(validationErrors, "; "))
		}
	} else {
		// No extractor = --no-llm mode: warn but allow
		if enrichedContent == "" && observationType == "" && conceptsCSV == "" {
			fmt.Fprintf(os.Stderr, "Warning: learning stored without LLM enrichment (use --extractor for better quality)\n")
		}
	}

	// Determine embedding text: use enriched content when available for better embeddings
	embeddingText := opts.Message
	if enrichedContent != "" {
		embeddingText = enrichedContent
	}

	// Generate embedding with "passage: " prefix for e5-small-v2 (ISSUE-221)
	embedding, _, _, err := generateEmbeddingONNX("passage: "+embeddingText, modelPath)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	embeddingBlob, err := sqlite_vec.SerializeFloat32(embedding)
	if err != nil {
		return err
	}

	// ISSUE-184/188: Learn-time dedup — LLM-driven ingest decisions when extractor available,
	// with fallback to threshold-based boost for backward compatibility.
	// Exception: corrections are always stored (they're used for contradiction detection).
	if opts.Type != "correction" {
		dupResult, err := searchRawSimilar(db, embedding, 3) // top-3 for LLM context

		// Track whether we processed a valid LLM decision (to skip threshold fallback)
		llmDecisionProcessed := false

		// LLM-driven decision if extractor available and similar results exist above threshold
		if opts.Extractor != nil && err == nil && len(dupResult) > 0 && dupResult[0].similarity > 0.5 {
			existing := rawResultsToExistingMemories(dupResult)
			decision, decideErr := opts.Extractor.Decide(context.Background(), opts.Message, existing)
			if decideErr == nil && decision != nil {
				// Validate TargetID: must match one of the returned dupResult IDs
				targetValid := decision.Action == IngestAdd
				if !targetValid {
					for _, r := range dupResult {
						if r.id == decision.TargetID {
							targetValid = true
							break
						}
					}
				}

				if targetValid {
					llmDecisionProcessed = true
					switch decision.Action {
					case IngestUpdate:
						if err := updateEmbeddingContent(db, decision.TargetID, content, embeddingBlob,
							observationType, conceptsCSV, principle, antiPattern, rationale, enrichedContent); err == nil {
							return nil
						}
						// On update failure, fall through to insert
					case IngestDelete:
						_ = supersedeEmbedding(db, decision.TargetID)
						// Fall through to insert new memory below
					case IngestNoop:
						_ = boostExistingConfidence(db, decision.TargetID)
						return nil
					case IngestAdd:
						// Fall through to insert
					}
				}
				// Invalid TargetID — fall through to threshold-based dedup
			}
			// LLM failed — fall through to threshold-based dedup
		}

		// Threshold fallback (original ISSUE-184 behavior)
		// Use 0.98 threshold to require near-exact matches before deduplicating
		// Skip if we already processed a valid LLM decision (prevents boosting superseded entries)
		if !llmDecisionProcessed && err == nil && len(dupResult) > 0 && dupResult[0].similarity > 0.98 {
			_ = boostExistingConfidence(db, dupResult[0].id)
			return nil
		}
	}

	// Insert into vec table
	vecStmt := `INSERT INTO vec_embeddings(embedding) VALUES (?)`
	result, err := db.Exec(vecStmt, embeddingBlob)
	if err != nil {
		return err
	}

	embeddingID, _ := result.LastInsertId()

	// Determine memory type (TASK-9)
	memoryType := opts.Type

	// Insert into metadata table with the full formatted content and observation columns
	metaStmt := `INSERT INTO embeddings(content, source, source_type, confidence, memory_type, embedding_id,
		observation_type, concepts, principle, anti_pattern, rationale, enriched_content, model_version)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	metaResult, err := db.Exec(metaStmt, content, "memory", sourceType, confidence, memoryType, embeddingID,
		observationType, conceptsCSV, principle, antiPattern, rationale, enrichedContent, "e5-small-v2")
	if err != nil {
		return err
	}

	// Insert into FTS5 table if available (rowid matches embeddings.id)
	metaID, _ := metaResult.LastInsertId()
	insertFTS5(db, metaID, content)

	return nil
}

// updateRetrievalTracking updates retrieval statistics for query results.
func updateRetrievalTracking(db *sql.DB, results []QueryResult, project string) error {
	timestamp := time.Now().Format(time.RFC3339)

	for _, result := range results {
		// Get current projects_retrieved and retrieval_timestamps
		var currentProjects, currentTimestamps string
		err := db.QueryRow("SELECT projects_retrieved, retrieval_timestamps FROM embeddings WHERE content = ?", result.Content).Scan(&currentProjects, &currentTimestamps)
		if err != nil {
			continue // Skip if not found
		}

		// Parse and deduplicate projects
		projectMap := make(map[string]bool)
		if currentProjects != "" {
			for _, p := range strings.Split(currentProjects, ",") {
				p = strings.TrimSpace(p)
				if p != "" {
					projectMap[p] = true
				}
			}
		}

		// Track the project context (use "(unspecified)" for queries without a project)
		if project != "" {
			projectMap[project] = true
		} else {
			projectMap["(unspecified)"] = true
		}

		// Rebuild comma-separated list
		var projects []string
		for p := range projectMap {
			projects = append(projects, p)
		}
		newProjects := strings.Join(projects, ",")

		// TASK-9: Append timestamp to retrieval_timestamps JSON array
		var timestamps []string
		if currentTimestamps != "" {
			if err := json.Unmarshal([]byte(currentTimestamps), &timestamps); err == nil {
				timestamps = append(timestamps, timestamp)
			} else {
				// If parsing fails, start fresh
				timestamps = []string{timestamp}
			}
		} else {
			timestamps = []string{timestamp}
		}
		timestampsJSON, err := json.Marshal(timestamps)
		if err != nil {
			return err
		}

		// Update the row (always increment retrieval_count, update timestamp, boost confidence)
		// ISSUE-184: Retrieval boost — +0.05 per retrieval, capped at 1.0
		updateStmt := `
			UPDATE embeddings
			SET retrieval_count = retrieval_count + 1,
			    last_retrieved = ?,
			    projects_retrieved = ?,
			    retrieval_timestamps = ?,
			    confidence = MIN(1.0, confidence + 0.05)
			WHERE content = ?
		`
		_, err = db.Exec(updateStmt, timestamp, newProjects, string(timestampsJSON), result.Content)
		if err != nil {
			return err
		}
	}

	return nil
}

