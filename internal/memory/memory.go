// Package memory provides memory management operations for storing learnings.
package memory

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Exported constants.
const (
	DefaultSimilarityThreshold = 0.7
)

// ActivationStats contains activation statistics for a memory entry.
type ActivationStats struct {
	Activation       float64 // ACT-R base-level activation B_i
	RetrievalCount   int
	TimestampCount   int // Total timestamps recorded
	ActiveTimestamps int // Timestamps within retention window
	DecayParameter   float64
	SessionCount     int     // Number of distinct retrieval sessions
	SessionBonus     float64 // Multiplier bonus from cross-session retrievals
	Effectiveness    float64 // B_i + α × impact_score (FR-016)
}

// ============================================================================
// TASK-9: ACT-R activation scoring
// ============================================================================

// ActivationStatsOpts holds options for getting activation statistics.
type ActivationStatsOpts struct {
	MemoryRoot string
	Content    string // Substring to match in content
}

// ClearTimestampsOpts holds options for clearing timestamps (test-only).
type ClearTimestampsOpts struct {
	MemoryRoot string
	Content    string
}

// DecayOpts holds options for memory decay.
type DecayOpts struct {
	MemoryRoot string
	Factor     float64 // Decay factor (default: 0.9, only used in legacy mode)
	UseLegacy  bool    // If true, use old flat decay; if false, use ACT-R (TASK-9)
}

// DecayResult contains the result of memory decay.
type DecayResult struct {
	EntriesAffected int
	Factor          float64
	MinConfidence   float64
	MaxConfidence   float64
}

// DecideOpts holds options for decision logging.
type DecideOpts struct {
	Context      string
	Choice       string
	Reason       string
	Alternatives []string
	Project      string
	MemoryRoot   string
}

// DecideResult contains the result of logging a decision.
type DecideResult struct {
	FilePath string
}

// FileSystem provides file system operations for memory management.
type FileSystem interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm os.FileMode) error
	ReadDir(path string) ([]os.DirEntry, error)
	Stat(path string) (os.FileInfo, error)
	Rename(oldPath, newPath string) error
	Remove(path string) error
	MkdirAll(path string, perm os.FileMode) error
}

// GrepMatch represents a single grep match.
type GrepMatch struct {
	File    string
	LineNum int
	Line    string
}

// GrepOpts holds options for memory grep.
type GrepOpts struct {
	Pattern          string
	Project          string
	IncludeDecisions bool
	MemoryRoot       string
}

// GrepResult contains the results of a grep search.
type GrepResult struct {
	Matches []GrepMatch
}

// LearnConflictResult contains the result of learning with conflict check.
type LearnConflictResult struct {
	HasConflict   bool
	ConflictEntry string
	Similarity    float64
	ConflictType  string // "duplicate" or "contradiction"
	Stored        bool
}

// LearnOpts holds options for learning storage.
type LearnOpts struct {
	Message        string
	Project        string
	Source         string // "internal" or "external" (defaults to "internal")
	Type           string // "correction", "reflection", or empty for default
	MemoryRoot     string
	ModelDir       string       // Directory for ONNX models (default: ~/.claude/models)
	HTTPClient     *http.Client // HTTP client for downloads (default: http.DefaultClient)
	VectorEmbedder Embedder     // Injected embedder; bypasses ONNX init/download when set
	Extractor      LLMExtractor // Optional LLM extractor for structured knowledge extraction (ISSUE-188)

	// PrecomputedObservation skips the Extract() API call when set.
	// Used by batch extraction to avoid redundant per-item API calls.
	PrecomputedObservation *Observation
}

// MigrateToACTROpts holds options for migrating to ACT-R.
type MigrateToACTROpts struct {
	MemoryRoot string
}

// PromoteCandidate represents a candidate for promotion.
type PromoteCandidate struct {
	Content        string
	RetrievalCount int
	UniqueProjects int
}

// PromoteInteractiveOpts holds options for interactive memory promotion.
type PromoteInteractiveOpts struct {
	MemoryRoot    string
	MinRetrievals int                                  // Minimum retrieval count (default: 3)
	MinProjects   int                                  // Minimum unique projects (default: 2)
	Review        bool                                 // Enable interactive review mode
	ReviewFunc    func(PromoteCandidate) (bool, error) // Function to review each candidate
	ClaudeMDPath  string                               // Path to CLAUDE.md (default: ~/.claude/CLAUDE.md)
}

// PromoteInteractiveResult contains the result of interactive memory promotion.
type PromoteInteractiveResult struct {
	CandidatesReviewed int
	CandidatesApproved int
	CandidatesRejected int
}

// PromoteOpts holds options for memory promotion.
type PromoteOpts struct {
	MemoryRoot    string
	MinRetrievals int // Minimum retrieval count (default: 3)
	MinProjects   int // Minimum unique projects (default: 2)
}

// PromoteResult contains the result of memory promotion.
type PromoteResult struct {
	Candidates []PromoteCandidate
}

// PruneOpts holds options for memory pruning.
type PruneOpts struct {
	MemoryRoot string
	Threshold  float64 // Confidence threshold (default: 0.1)
}

// PruneResult contains the result of memory pruning.
type PruneResult struct {
	EntriesRemoved  int
	EntriesRetained int
	Threshold       float64
}

// QueryOpts holds options for memory query.
type QueryOpts struct {
	Text                string
	Limit               int
	Project             string // Project name for tracking retrievals
	MemoryRoot          string
	ModelDir            string       // Directory for ONNX models (default: ~/.claude/models)
	HTTPClient          *http.Client // HTTP client for downloads (default: http.DefaultClient)
	VectorEmbedder      Embedder     // Injected embedder; bypasses ONNX init/download when set
	MinScore            float64      // Minimum similarity score threshold (0.0 = no filtering)
	SpreadingActivation bool         // FR-017: secondary search for memories similar to top-K results
}

// QueryResult represents a single query result.
type QueryResult struct {
	ID                int64 // Database ID of the embedding entry (ISSUE-214)
	Content           string
	Score             float64
	Source            string   // File source ("memory")
	SourceType        string   // "internal" or "external"
	Confidence        float64  // Confidence score (0.0-1.0)
	MemoryType        string   // "correction", "reflection", or "" (ISSUE-178)
	RetrievalCount    int      // Number of times this entry has been retrieved (ISSUE-188)
	ProjectsRetrieved []string // Projects that have retrieved this entry (ISSUE-188)
	MatchType         string   // "vector", "bm25", or "hybrid" (ISSUE-188)
}

// QueryResults contains the results of a query.
type QueryResults struct {
	Results        []QueryResult
	VectorStorage  string
	EmbeddingModel string
	APICallsMade   bool
	// ONNX model fields (TASK-052: Real ONNX inference)
	EmbeddingDimensions int
	UsedONNXRuntime     bool
	ModelDownloaded     bool
	ModelPath           string
	ModelLoaded         bool
	ModelType           string
	InferenceExecuted   bool
	UsedMockEmbeddings  bool
	// Session caching fields (ISSUE-48)
	SessionCreatedNew bool
	SessionReused     bool
	QueryDuration     time.Duration
	// Hybrid search fields (ISSUE-181)
	UsedHybridSearch bool
	BM25Enabled      bool
	// Retrieval logging (Task 2: self-reinforcing learning)
	FilteredCount int // Number of results excluded by MinScore threshold
	// Spreading activation (FR-017)
	SpreadingActivationApplied bool
}

// RealFS implements FileSystem using the real file system.
type RealFS struct{}

// MkdirAll creates a directory and all parent directories using os.MkdirAll.
func (RealFS) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

// ReadDir reads a directory using os.ReadDir.
func (RealFS) ReadDir(path string) ([]os.DirEntry, error) {
	return os.ReadDir(path)
}

// ReadFile reads a file using os.ReadFile.
func (RealFS) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// Remove removes a file or directory using os.Remove.
func (RealFS) Remove(path string) error {
	return os.Remove(path)
}

// Rename renames a file or directory using os.Rename.
func (RealFS) Rename(oldPath, newPath string) error {
	return os.Rename(oldPath, newPath)
}

// Stat returns file info using os.Stat.
func (RealFS) Stat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

// WriteFile writes a file using os.WriteFile.
func (RealFS) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

// SimulateTimeOpts holds options for simulating time passage (test-only).
type SimulateTimeOpts struct {
	MemoryRoot string
	Content    string
	DaysToAge  int
}

// CleanupReQueryArtifacts removes stale re-query detection artifacts (ISSUE-232).
// Resets flagged_for_review on all embeddings and deletes last_query.json.
// Returns the number of entries that had their flag reset.
func CleanupReQueryArtifacts(memoryRoot string) (int, error) {
	dbPath := filepath.Join(memoryRoot, "embeddings.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return 0, fmt.Errorf("failed to open database: %w", err)
	}

	defer func() { _ = db.Close() }()

	// Reset flagged_for_review
	result, err := db.Exec("UPDATE embeddings SET flagged_for_review = 0 WHERE flagged_for_review = 1")
	if err != nil {
		return 0, fmt.Errorf("failed to reset flagged_for_review: %w", err)
	}

	count, _ := result.RowsAffected()

	// Delete last_query.json
	_ = os.Remove(filepath.Join(memoryRoot, "last_query.json"))

	return int(count), nil
}

// ClearTimestampsForTest clears retrieval_timestamps for testing migration.
// This is a test-only function.
func ClearTimestampsForTest(opts ClearTimestampsOpts) error {
	dbPath := filepath.Join(opts.MemoryRoot, "embeddings.db")

	db, err := initEmbeddingsDB(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	defer func() { _ = db.Close() }()

	updateStmt := "UPDATE embeddings SET retrieval_timestamps = '' WHERE content LIKE ?"

	result, err := db.Exec(updateStmt, "%"+opts.Content+"%")
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("no rows matched pattern %q", opts.Content)
	}

	return nil
}

// ClusterIntoSessions groups RFC3339 timestamps into sessions separated by gaps
// exceeding the given threshold. Timestamps are sorted before clustering.
// A gap exactly equal to the threshold does NOT start a new session (must exceed it).
func ClusterIntoSessions(timestamps []string, gap time.Duration) [][]string {
	if len(timestamps) == 0 {
		return nil
	}

	// Parse and sort timestamps
	type parsedTS struct {
		raw    string
		parsed time.Time
	}

	var parsed []parsedTS

	for _, ts := range timestamps {
		t, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			continue
		}

		parsed = append(parsed, parsedTS{raw: ts, parsed: t})
	}

	if len(parsed) == 0 {
		return nil
	}

	sort.Slice(parsed, func(i, j int) bool {
		return parsed[i].parsed.Before(parsed[j].parsed)
	})

	// Cluster by gap
	sessions := [][]string{{parsed[0].raw}}
	for i := 1; i < len(parsed); i++ {
		if parsed[i].parsed.Sub(parsed[i-1].parsed) > gap {
			sessions = append(sessions, []string{parsed[i].raw})
		} else {
			sessions[len(sessions)-1] = append(sessions[len(sessions)-1], parsed[i].raw)
		}
	}

	return sessions
}

// Decay reduces confidence of all memory entries by a factor.
func Decay(opts DecayOpts) (*DecayResult, error) {
	// Set default factor
	factor := opts.Factor
	if factor == 0 {
		factor = 0.9
	}

	// Open DB
	dbPath := filepath.Join(opts.MemoryRoot, "embeddings.db")

	db, err := initEmbeddingsDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	defer func() { _ = db.Close() }()

	// Get min/max confidence before decay
	var minBefore, maxBefore float64

	err = db.QueryRow("SELECT MIN(confidence), MAX(confidence) FROM embeddings").Scan(&minBefore, &maxBefore)
	if err != nil {
		minBefore = 0
		maxBefore = 1
	}

	// Apply decay
	updateStmt := `UPDATE embeddings SET confidence = confidence * ?`

	result, err := db.Exec(updateStmt, factor)
	if err != nil {
		return nil, fmt.Errorf("failed to decay confidence: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()

	// Get min/max confidence after decay
	var minAfter, maxAfter float64

	err = db.QueryRow("SELECT MIN(confidence), MAX(confidence) FROM embeddings").Scan(&minAfter, &maxAfter)
	if err != nil {
		minAfter = 0
		maxAfter = 1
	}

	return &DecayResult{
		EntriesAffected: int(rowsAffected),
		Factor:          factor,
		MinConfidence:   minAfter,
		MaxConfidence:   maxAfter,
	}, nil
}

// Decide logs a decision with reasoning and alternatives.
func Decide(opts DecideOpts) (*DecideResult, error) {
	if opts.Context == "" {
		return nil, errors.New("context is required")
	}

	if opts.Choice == "" {
		return nil, errors.New("choice is required")
	}

	if opts.Reason == "" {
		return nil, errors.New("reason is required")
	}

	// Ensure decisions directory exists
	decisionsDir := filepath.Join(opts.MemoryRoot, "decisions")
	if err := os.MkdirAll(decisionsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create decisions directory: %w", err)
	}

	// Build filename: {DATE}-{PROJECT}.jsonl
	today := time.Now().Format("2006-01-02")
	filename := fmt.Sprintf("%s-%s.jsonl", today, opts.Project)
	filePath := filepath.Join(decisionsDir, filename)

	// Open file for appending (create if doesn't exist)
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open decisions file: %w", err)
	}

	defer func() { _ = f.Close() }()

	// Build JSON entry
	entry := map[string]any{
		"timestamp":    time.Now().Format(time.RFC3339),
		"context":      opts.Context,
		"choice":       opts.Choice,
		"reason":       opts.Reason,
		"alternatives": opts.Alternatives,
	}

	// Marshal and write
	data, err := json.Marshal(entry)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal entry: %w", err)
	}

	if _, err := f.Write(data); err != nil {
		return nil, fmt.Errorf("failed to write entry: %w", err)
	}

	if _, err := f.WriteString("\n"); err != nil {
		return nil, fmt.Errorf("failed to write newline: %w", err)
	}

	return &DecideResult{
		FilePath: filePath,
	}, nil
}

// FilterByMinScore returns only results with Score >= minScore.
// A minScore of 0.0 disables filtering and returns all results.
func FilterByMinScore(results []QueryResult, minScore float64) []QueryResult {
	if minScore <= 0 {
		return results
	}

	filtered := make([]QueryResult, 0, len(results))
	for _, r := range results {
		if r.Score >= minScore {
			filtered = append(filtered, r)
		}
	}

	return filtered
}

// GetActivationStats retrieves ACT-R activation statistics for a memory entry.
func GetActivationStats(opts ActivationStatsOpts) (*ActivationStats, error) {
	dbPath := filepath.Join(opts.MemoryRoot, "embeddings.db")

	db, err := initEmbeddingsDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	defer func() { _ = db.Close() }()

	// Find entry by content substring
	var (
		retrievalTimestamps, memoryType string
		retrievalCount                  int
		importanceScore, impactScore    float64
	)

	query := "SELECT retrieval_timestamps, retrieval_count, memory_type, importance_score, impact_score FROM embeddings WHERE content LIKE ? LIMIT 1"

	err = db.QueryRow(query, "%"+opts.Content+"%").Scan(&retrievalTimestamps, &retrievalCount, &memoryType, &importanceScore, &impactScore)
	if err != nil {
		return nil, fmt.Errorf("failed to find entry: %w", err)
	}

	// Read alpha_weight from metadata (default 0.5)
	alphaStr, err := getMetadata(db, "alpha_weight")
	if err != nil {
		return nil, fmt.Errorf("failed to read alpha_weight: %w", err)
	}

	alphaWeight := 0.5

	if alphaStr != "" {
		if v, parseErr := strconv.ParseFloat(alphaStr, 64); parseErr == nil {
			alphaWeight = v
		}
	}

	// Parse timestamps
	var timestamps []string
	if retrievalTimestamps != "" {
		err := json.Unmarshal([]byte(retrievalTimestamps), &timestamps)
		if err != nil {
			return nil, fmt.Errorf("failed to parse timestamps: %w", err)
		}
	}

	// Calculate ACT-R activation: B_i = ln(Σ t_j^(-d))
	// Default decay parameter d = 0.5 (ACT-R standard)
	d := 0.5

	// For corrections, use minimal decay (close to 0) for indefinite retention
	if memoryType == "correction" {
		d = 0.1
	}

	now := time.Now()

	var sumPowerTerms float64

	activeTimestamps := 0

	for _, ts := range timestamps {
		parsedTime, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			continue
		}

		// For reflections, apply 30-day sliding window
		if memoryType == "reflection" {
			age := now.Sub(parsedTime)
			if age > 30*24*time.Hour {
				continue // Skip timestamps older than 30 days
			}
		}

		activeTimestamps++

		// Calculate time since retrieval in seconds
		age := now.Sub(parsedTime).Seconds()
		if age == 0 {
			age = 1 // Avoid division by zero
		}

		// Add power term: t_j^(-d)
		sumPowerTerms += math.Pow(age, -d)
	}

	// Detect sessions from timestamps (30min gap heuristic)
	sessions := ClusterIntoSessions(timestamps, 30*time.Minute)

	sessionCount := len(sessions)
	if sessionCount == 0 {
		sessionCount = 1
	}

	// Apply session multiplier when retrievals span multiple sessions
	var sessionBonus float64
	if sessionCount > 1 {
		sessionBonus = 0.5 // 1.5x multiplier = base + 0.5 bonus
		sumPowerTerms *= 1.5
	}

	// Calculate activation
	var activation float64
	if sumPowerTerms > 0 {
		activation = math.Log(sumPowerTerms)
	} else {
		activation = 0
	}

	// FR-016: effectiveness = B_i + α × impact_score
	effectiveness := importanceScore + alphaWeight*impactScore

	return &ActivationStats{
		Activation:       activation,
		RetrievalCount:   retrievalCount,
		TimestampCount:   len(timestamps),
		ActiveTimestamps: activeTimestamps,
		DecayParameter:   d,
		SessionCount:     sessionCount,
		SessionBonus:     sessionBonus,
		Effectiveness:    effectiveness,
	}, nil
}

// Grep searches memory files for a pattern.
func Grep(opts GrepOpts) (*GrepResult, error) {
	if opts.Pattern == "" {
		return nil, errors.New("pattern is required")
	}

	var matches []GrepMatch

	pattern := strings.ToLower(opts.Pattern)

	// Search embeddings DB for memory content
	dbPath := filepath.Join(opts.MemoryRoot, "embeddings.db")
	if _, statErr := os.Stat(dbPath); statErr == nil {
		db, openErr := sql.Open("sqlite3", dbPath)
		if openErr == nil {
			defer func() { _ = db.Close() }()

			rows, queryErr := db.Query("SELECT id, content FROM embeddings WHERE LOWER(content) LIKE '%' || LOWER(?) || '%'", pattern)
			if queryErr == nil {
				defer func() { _ = rows.Close() }()

				for rows.Next() {
					var (
						id      int
						content string
					)

					if rows.Scan(&id, &content) == nil {
						matches = append(matches, GrepMatch{
							File:    "memory",
							LineNum: id,
							Line:    content,
						})
					}
				}
			}
		}
	}

	// Search sessions directory
	sessionsDir := filepath.Join(opts.MemoryRoot, "sessions")
	sessionMatches := searchDirectory(sessionsDir, pattern, opts.Project)
	matches = append(matches, sessionMatches...)

	// Search decisions if flag is set
	if opts.IncludeDecisions {
		decisionsDir := filepath.Join(opts.MemoryRoot, "decisions")
		decisionMatches := searchDirectory(decisionsDir, pattern, opts.Project)
		matches = append(matches, decisionMatches...)
	}

	return &GrepResult{
		Matches: matches,
	}, nil
}

// IsSessionBoilerplate returns true if the line is structural/boilerplate content
// from session summary files that should not be embedded.
func IsSessionBoilerplate(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return true
	}

	// Markdown headers
	if strings.HasPrefix(trimmed, "#") {
		return true
	}

	// Metadata lines
	if strings.HasPrefix(trimmed, "**Project:**") || strings.HasPrefix(trimmed, "**Date:**") {
		return true
	}

	// Horizontal rules and ellipsis
	if trimmed == "---" || trimmed == "..." || strings.TrimLeft(trimmed, "-") == "" || strings.TrimLeft(trimmed, ".") == "" {
		return true
	}

	// Short lines after stripping markdown formatting
	stripped := stripMarkdownFormatting(trimmed)
	if len(stripped) < 10 {
		return true
	}

	// Fragment detection: fewer than 5 words can't express a useful learning
	if len(strings.Fields(stripped)) < 5 {
		return true
	}

	// Legacy canned extraction strings (pre-rich-context extractors)
	if isLegacyCannedExtraction(trimmed) {
		return true
	}

	return false
}

// Learn stores a learning in the memory index.
func Learn(opts LearnOpts) error {
	if opts.Message == "" {
		return errors.New("message is required")
	}

	// Ensure memory directory exists
	err := os.MkdirAll(opts.MemoryRoot, 0755)
	if err != nil {
		return fmt.Errorf("failed to create memory directory: %w", err)
	}

	err = learnToEmbeddings(opts)
	if err != nil {
		return fmt.Errorf("failed to create embedding: %w", err)
	}

	return nil
}

// LearnWithConflictCheck stores a learning but first checks for similar existing entries.
func LearnWithConflictCheck(opts LearnOpts) (*LearnConflictResult, error) {
	if opts.Message == "" {
		return nil, errors.New("message is required")
	}

	// Open DB to check for similar entries
	dbPath := filepath.Join(opts.MemoryRoot, "embeddings.db")

	db, err := initEmbeddingsDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	var embedder Embedder
	if opts.VectorEmbedder != nil {
		embedder = opts.VectorEmbedder
	} else {
		// Determine model directory
		modelDir := opts.ModelDir
		if modelDir == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				_ = db.Close()
				return nil, fmt.Errorf("failed to get home directory: %w", err)
			}

			modelDir = filepath.Join(homeDir, ".claude", "models")
		}

		client := opts.HTTPClient
		if client == nil {
			client = http.DefaultClient
		}

		// Initialize ONNX Runtime and download model if needed
		if err := os.MkdirAll(modelDir, 0755); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("failed to create model directory: %w", err)
		}

		if err := initializeONNXRuntimeWithClient(modelDir, client); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("failed to initialize ONNX Runtime: %w", err)
		}

		modelPath := filepath.Join(modelDir, "e5-small-v2.onnx")
		if _, err := os.Stat(modelPath); os.IsNotExist(err) {
			if err := downloadModel(modelPath, client); err != nil {
				_ = db.Close()
				return nil, fmt.Errorf("failed to download model: %w", err)
			}
		}

		embedder = &onnxEmbedder{modelPath: modelPath}
	}

	// Generate embedding for the new message with "passage: " prefix for e5-small-v2 (ISSUE-221)
	// This is passage-to-passage comparison for deduplication, not a user query
	embedding, err := embedder.Embed("passage: " + opts.Message)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Search for similar entries
	results, err := searchSimilar(db, embedding, 1)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	hasConflict := false
	conflictEntry := ""
	similarity := 0.0
	conflictType := ""

	if len(results) > 0 {
		similarity = results[0].Score
		if similarity > 0.85 {
			hasConflict = true
			conflictEntry = results[0].Content

			// Determine if it's a contradiction or duplicate
			conflictType = detectConflictType(opts.Message, results[0].Content)
		}
	}

	// Close DB before Learn() to release SQLite write lock — Learn calls
	// learnToEmbeddings which opens its own connection.
	_ = db.Close()

	// Store the learning regardless
	if err := Learn(opts); err != nil {
		return nil, fmt.Errorf("failed to store learning: %w", err)
	}

	return &LearnConflictResult{
		HasConflict:   hasConflict,
		ConflictEntry: conflictEntry,
		Similarity:    similarity,
		ConflictType:  conflictType,
		Stored:        true,
	}, nil
}

// MigrateToACTR migrates existing entries from flat decay to ACT-R activation.
// For entries without retrieval_timestamps, initializes with current time.
func MigrateToACTR(opts MigrateToACTROpts) error {
	dbPath := filepath.Join(opts.MemoryRoot, "embeddings.db")

	db, err := initEmbeddingsDB(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	defer func() { _ = db.Close() }()

	// Find entries with empty retrieval_timestamps
	query := "SELECT id, retrieval_count FROM embeddings WHERE retrieval_timestamps = '' OR retrieval_timestamps IS NULL"

	rows, err := db.Query(query)
	if err != nil {
		return err
	}

	defer func() { _ = rows.Close() }()

	timestamp := time.Now().Format(time.RFC3339)

	for rows.Next() {
		var (
			id             int64
			retrievalCount int
		)

		if err := rows.Scan(&id, &retrievalCount); err != nil {
			continue
		}

		// Initialize timestamps based on retrieval count
		// If retrieval_count > 0, create that many timestamps spread over past 30 days
		var timestamps []string

		if retrievalCount > 0 {
			for i := range retrievalCount {
				// Spread timestamps evenly over past 30 days
				daysAgo := (30 * i) / retrievalCount
				ts := time.Now().Add(-time.Duration(daysAgo) * 24 * time.Hour)
				timestamps = append(timestamps, ts.Format(time.RFC3339))
			}
		} else {
			// No retrievals yet, set single initial timestamp
			timestamps = []string{timestamp}
		}

		timestampsJSON, err := json.Marshal(timestamps)
		if err != nil {
			continue
		}

		updateStmt := "UPDATE embeddings SET retrieval_timestamps = ? WHERE id = ?"

		_, err = db.Exec(updateStmt, string(timestampsJSON), id)
		_ = err // best-effort update
	}

	if err := rows.Err(); err != nil {
		return err
	}

	return nil
}

// Promote identifies memory entries that meet retrieval thresholds for promotion to global memory.
func Promote(opts PromoteOpts) (*PromoteResult, error) {
	if opts.MemoryRoot == "" {
		return nil, errors.New("memory root is required")
	}

	// Set defaults
	minRetrievals := opts.MinRetrievals
	if minRetrievals == 0 {
		minRetrievals = 3
	}

	minProjects := opts.MinProjects
	if minProjects == 0 {
		minProjects = 2
	}

	// Open DB
	dbPath := filepath.Join(opts.MemoryRoot, "embeddings.db")

	db, err := initEmbeddingsDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	defer func() { _ = db.Close() }()

	// Query for candidates
	query := `
		SELECT content, retrieval_count, projects_retrieved
		FROM embeddings
		WHERE retrieval_count >= ?
	`

	rows, err := db.Query(query, minRetrievals)
	if err != nil {
		return nil, fmt.Errorf("failed to query candidates: %w", err)
	}

	defer func() { _ = rows.Close() }()

	var candidates []PromoteCandidate

	for rows.Next() {
		var (
			content           string
			retrievalCount    int
			projectsRetrieved string
		)

		err := rows.Scan(&content, &retrievalCount, &projectsRetrieved)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Count unique projects
		uniqueProjects := 0

		if projectsRetrieved != "" {
			projectMap := make(map[string]bool)

			projects := strings.SplitSeq(projectsRetrieved, ",")
			for p := range projects {
				p = strings.TrimSpace(p)
				if p != "" {
					projectMap[p] = true
				}
			}

			uniqueProjects = len(projectMap)
		}

		// Check if meets minimum projects threshold
		if uniqueProjects >= minProjects {
			candidates = append(candidates, PromoteCandidate{
				Content:        content,
				RetrievalCount: retrievalCount,
				UniqueProjects: uniqueProjects,
			})
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return &PromoteResult{
		Candidates: candidates,
	}, nil
}

// PromoteInteractive identifies memory entries for promotion and optionally reviews them interactively.
func PromoteInteractive(opts PromoteInteractiveOpts) (*PromoteInteractiveResult, error) {
	if opts.MemoryRoot == "" {
		return nil, errors.New("memory root is required")
	}

	// If review mode is enabled, ReviewFunc is required
	if opts.Review && opts.ReviewFunc == nil {
		return nil, errors.New("review function is required when review mode is enabled")
	}

	// Set defaults
	minRetrievals := opts.MinRetrievals
	if minRetrievals == 0 {
		minRetrievals = 3
	}

	minProjects := opts.MinProjects
	if minProjects == 0 {
		minProjects = 2
	}

	claudeMDPath := opts.ClaudeMDPath
	if claudeMDPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}

		claudeMDPath = filepath.Join(homeDir, ".claude", "CLAUDE.md")
	}

	// Get promotion candidates
	promoteResult, err := Promote(PromoteOpts{
		MemoryRoot:    opts.MemoryRoot,
		MinRetrievals: minRetrievals,
		MinProjects:   minProjects,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get promotion candidates: %w", err)
	}

	result := &PromoteInteractiveResult{}

	// If review mode is disabled, just return empty result
	if !opts.Review {
		return result, nil
	}

	// Review each candidate
	var approvedLearnings []string

	for _, candidate := range promoteResult.Candidates {
		result.CandidatesReviewed++

		isApproved, err := opts.ReviewFunc(candidate)
		if err != nil {
			return nil, fmt.Errorf("review failed: %w", err)
		}

		if isApproved {
			result.CandidatesApproved++

			approvedLearnings = append(approvedLearnings, candidate.Content)
		} else {
			result.CandidatesRejected++
		}
	}

	// Append approved candidates to CLAUDE.md
	if len(approvedLearnings) > 0 {
		err := appendToClaudeMD(claudeMDPath, approvedLearnings)
		if err != nil {
			return nil, fmt.Errorf("failed to append to CLAUDE.md: %w", err)
		}
	}

	return result, nil
}

// Prune removes memory entries below a confidence threshold.
func Prune(opts PruneOpts) (*PruneResult, error) {
	// Set default threshold
	threshold := opts.Threshold
	if threshold == 0 {
		threshold = 0.1
	}

	// Open DB
	dbPath := filepath.Join(opts.MemoryRoot, "embeddings.db")

	db, err := initEmbeddingsDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	defer func() { _ = db.Close() }()

	// Count total before pruning
	var totalBefore int

	err = db.QueryRow("SELECT COUNT(*) FROM embeddings").Scan(&totalBefore)
	if err != nil {
		totalBefore = 0
	}

	// Get ids and embedding_ids to delete from vec_embeddings and FTS5
	rows, err := db.Query("SELECT id, embedding_id FROM embeddings WHERE confidence < ?", threshold)
	if err != nil {
		return nil, fmt.Errorf("failed to query for pruning: %w", err)
	}

	type pruneEntry struct {
		id          int64
		embeddingID int64
	}

	var toDelete []pruneEntry

	for rows.Next() {
		var e pruneEntry

		err := rows.Scan(&e.id, &e.embeddingID)
		if err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		toDelete = append(toDelete, e)
	}

	_ = rows.Close()

	// Delete from embeddings table
	deleteMetaStmt := `DELETE FROM embeddings WHERE confidence < ?`

	result, err := db.Exec(deleteMetaStmt, threshold)
	if err != nil {
		return nil, fmt.Errorf("failed to prune entries: %w", err)
	}

	rowsDeleted, _ := result.RowsAffected()

	// Delete from vec_embeddings and FTS5 tables
	for _, e := range toDelete {
		_, _ = db.Exec("DELETE FROM vec_embeddings WHERE rowid = ?", e.embeddingID)
		deleteFTS5(db, e.id)
	}

	// Count total after pruning
	var totalAfter int

	err = db.QueryRow("SELECT COUNT(*) FROM embeddings").Scan(&totalAfter)
	if err != nil {
		totalAfter = 0
	}

	return &PruneResult{
		EntriesRemoved:  int(rowsDeleted),
		EntriesRetained: totalAfter,
		Threshold:       threshold,
	}, nil
}

// Query searches memory for semantically similar content using embeddings.
func Query(opts QueryOpts) (*QueryResults, error) {
	startTime := time.Now()

	if opts.Text == "" {
		return nil, errors.New("text is required")
	}

	limit := opts.Limit
	if limit == 0 {
		limit = 5
	}

	// Initialize embeddings database first (cheap) to check if there's anything to search
	dbPath := filepath.Join(opts.MemoryRoot, "embeddings.db")

	db, err := initEmbeddingsDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize embeddings database: %w", err)
	}

	defer func() { _ = db.Close() }()

	// Short-circuit: if the DB has no embeddings, skip expensive ONNX initialization
	if empty, checkErr := isEmbeddingsEmpty(db); checkErr == nil && empty {
		return &QueryResults{
			Results:       []QueryResult{},
			QueryDuration: time.Since(startTime),
		}, nil
	}

	var (
		modelDownloaded bool
		sessionCreated  bool
		sessionReused   bool
		queryEmbedding  []float32
		modelPath       string
	)

	if opts.VectorEmbedder != nil {
		// Injected embedder path — no ONNX init or model downloads.
		var err error

		queryEmbedding, err = opts.VectorEmbedder.Embed("query: " + opts.Text)
		if err != nil {
			return nil, fmt.Errorf("failed to generate query embedding: %w", err)
		}
	} else {
		// ONNX runtime path.
		client := opts.HTTPClient
		if client == nil {
			client = http.DefaultClient
		}

		// Determine model directory
		modelDir := opts.ModelDir
		if modelDir == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("failed to get home directory: %w", err)
			}

			modelDir = filepath.Join(homeDir, ".claude", "models")
		}

		// Ensure model directory exists
		if err := os.MkdirAll(modelDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create model directory: %w", err)
		}

		// Initialize ONNX Runtime
		if err := initializeONNXRuntimeWithClient(modelDir, client); err != nil {
			return nil, fmt.Errorf("failed to initialize ONNX Runtime: %w", err)
		}

		// Model path
		modelPath = filepath.Join(modelDir, "e5-small-v2.onnx")

		// Check if model needs to be downloaded
		if _, err := os.Stat(modelPath); os.IsNotExist(err) {
			if err := downloadModel(modelPath, client); err != nil {
				return nil, fmt.Errorf("failed to download model: %w", err)
			}

			modelDownloaded = true
		}

		// ISSUE-221: Check for stale model and re-download if needed
		needsDownload, err := ensureCorrectModel(db, modelPath)
		if err != nil {
			return nil, fmt.Errorf("failed to check model validity: %w", err)
		}

		if needsDownload {
			if err := downloadModel(modelPath, client); err != nil {
				return nil, fmt.Errorf("failed to download model: %w", err)
			}

			modelDownloaded = true
		}

		// ISSUE-221: Run model version migration if needed
		if err := migrateModelVersion(db, &onnxEmbedder{modelPath: modelPath}); err != nil {
			return nil, fmt.Errorf("failed to migrate model version: %w", err)
		}

		// Generate query embedding using ONNX model with "query: " prefix for e5-small-v2 (ISSUE-221)
		var err2 error

		queryEmbedding, sessionCreated, sessionReused, err2 = generateEmbeddingONNX("query: "+opts.Text, modelPath)
		if err2 != nil {
			return nil, fmt.Errorf("failed to generate query embedding: %w", err2)
		}
	}

	// Search using hybrid search (BM25 + vector + RRF)
	bm25Available := hasFTS5(db)

	results, err := hybridSearch(db, queryEmbedding, opts.Text, limit, 60)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	if results == nil {
		results = []QueryResult{}
	}

	// ISSUE-217: Filter results by minimum score threshold
	var filteredCount int

	if opts.MinScore > 0 {
		preFilterCount := len(results)

		var maxScore float64
		if preFilterCount > 0 {
			maxScore = results[0].Score
		}

		results = FilterByMinScore(results, opts.MinScore)

		filteredCount = preFilterCount - len(results)
		if len(results) == 0 && preFilterCount > 0 {
			fmt.Fprintf(os.Stderr, "Warning: all %d results filtered by MinScore threshold %.2f (max score: %.4f)\n",
				preFilterCount, opts.MinScore, maxScore)
		}
	}

	// FR-017: Apply spreading activation — secondary search for memories similar to top-K results.
	// This is interaction-scoped; no writes to the database.
	if opts.SpreadingActivation {
		spreadResults, spreadErr := applySpreadingActivation(db, results, spreadingActivationThreshold)
		if spreadErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: spreading activation failed: %v\n", spreadErr)
		} else {
			results = spreadResults
		}
	}

	// Update retrieval tracking for TASK-41
	if err := updateRetrievalTracking(db, results, opts.Project); err != nil {
		// Log error but don't fail the query
		fmt.Fprintf(os.Stderr, "Warning: failed to update retrieval tracking: %v\n", err)
	}

	duration := time.Since(startTime)

	usedONNX := opts.VectorEmbedder == nil

	return &QueryResults{
		Results:                    results,
		VectorStorage:              "sqlite-vec",
		EmbeddingModel:             "e5-small-v2",
		EmbeddingDimensions:        384,
		APICallsMade:               false,
		UsedONNXRuntime:            usedONNX,
		ModelDownloaded:            modelDownloaded,
		ModelPath:                  modelPath,
		ModelLoaded:                usedONNX,
		ModelType:                  "onnx",
		InferenceExecuted:          true,
		UsedMockEmbeddings:         !usedONNX,
		SessionCreatedNew:          sessionCreated,
		SessionReused:              sessionReused,
		QueryDuration:              duration,
		UsedHybridSearch:           true,
		BM25Enabled:                bm25Available,
		FilteredCount:              filteredCount,
		SpreadingActivationApplied: opts.SpreadingActivation,
	}, nil
}

// RemoveFromClaudeMD removes entries from the "## Promoted Learnings" section of CLAUDE.md.
// Each entry in entries is matched as a substring against the lines in the section.
// Lines containing any of the entries are removed. Other sections are untouched.
// Returns nil if the file doesn't exist or is empty.
func RemoveFromClaudeMD(fs FileSystem, claudeMDPath string, entries []string) error {
	content, err := fs.ReadFile(claudeMDPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf("failed to read CLAUDE.md: %w", err)
	}

	if len(content) == 0 {
		return nil
	}

	lines := strings.Split(string(content), "\n")

	const sectionHeader = "## Promoted Learnings"

	inPromotedSection := false

	var result []string

	for _, line := range lines {
		// Detect section boundaries
		if strings.HasPrefix(line, "## ") {
			inPromotedSection = strings.TrimSpace(line) == sectionHeader
		}

		if inPromotedSection && strings.HasPrefix(strings.TrimSpace(line), "- ") {
			// Check if this line matches any entry to remove
			shouldRemove := false

			for _, entry := range entries {
				if strings.Contains(line, entry) {
					shouldRemove = true
					break
				}
			}

			if shouldRemove {
				continue // Skip this line
			}
		}

		result = append(result, line)
	}

	return fs.WriteFile(claudeMDPath, []byte(strings.Join(result, "\n")), 0644)
}

// SimulateTimePassage ages retrieval timestamps by moving them back in time.
// This is a test-only function for verifying ACT-R decay behavior.
func SimulateTimePassage(opts SimulateTimeOpts) error {
	dbPath := filepath.Join(opts.MemoryRoot, "embeddings.db")

	db, err := initEmbeddingsDB(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	defer func() { _ = db.Close() }()

	// Find entry
	var retrievalTimestamps string

	query := "SELECT retrieval_timestamps FROM embeddings WHERE content LIKE ? LIMIT 1"

	err = db.QueryRow(query, "%"+opts.Content+"%").Scan(&retrievalTimestamps)
	if err != nil {
		return fmt.Errorf("failed to find entry: %w", err)
	}

	// Parse timestamps
	var timestamps []string
	if retrievalTimestamps != "" {
		err := json.Unmarshal([]byte(retrievalTimestamps), &timestamps)
		if err != nil {
			return fmt.Errorf("failed to parse timestamps: %w", err)
		}
	}

	// Age timestamps by subtracting days
	var agedTimestamps []string

	for _, ts := range timestamps {
		parsedTime, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			continue
		}

		aged := parsedTime.Add(-time.Duration(opts.DaysToAge) * 24 * time.Hour)
		agedTimestamps = append(agedTimestamps, aged.Format(time.RFC3339))
	}

	// Update database
	agedJSON, err := json.Marshal(agedTimestamps)
	if err != nil {
		return err
	}

	updateStmt := "UPDATE embeddings SET retrieval_timestamps = ? WHERE content LIKE ?"
	_, err = db.Exec(updateStmt, string(agedJSON), "%"+opts.Content+"%")

	return err
}

// appendToClaudeMD appends approved learnings to CLAUDE.md.
// If a "## Promoted Learnings" section already exists, new learnings are
// appended into it. Otherwise a new section is created at the end.
func appendToClaudeMD(claudeMDPath string, learnings []string) error {
	if err := os.MkdirAll(filepath.Dir(claudeMDPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Build the new learning lines
	var newLines strings.Builder
	for _, learning := range learnings {
		newLines.WriteString("- " + learning + "\n")
	}

	// Read existing content (empty if file doesn't exist)
	existing, err := os.ReadFile(claudeMDPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read CLAUDE.md: %w", err)
	}

	content := string(existing)

	const sectionHeader = "## Promoted Learnings"

	idx := strings.Index(content, sectionHeader)
	if idx >= 0 {
		// Section exists — find the end of it (next ## header or EOF)
		afterHeader := idx + len(sectionHeader)
		rest := content[afterHeader:]

		// Find the next ## header after the Promoted Learnings section
		nextSection := strings.Index(rest, "\n## ")

		var insertPos int
		if nextSection >= 0 {
			insertPos = afterHeader + nextSection
		} else {
			insertPos = len(content)
		}

		// Ensure there's a newline before the new learnings
		prefix := content[:insertPos]
		if !strings.HasSuffix(prefix, "\n") {
			prefix += "\n"
		}

		content = prefix + newLines.String() + content[insertPos:]
	} else {
		// No existing section — append one at the end
		if len(content) > 0 && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}

		content += "\n" + sectionHeader + "\n\n" + newLines.String()
	}

	if err := os.WriteFile(claudeMDPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write CLAUDE.md: %w", err)
	}

	return nil
}

// detectConflictType determines whether a conflict is a contradiction or duplicate.
// Returns "contradiction" if negation patterns or opposing advice detected, otherwise "duplicate".
func detectConflictType(newMessage, existingContent string) string {
	newLower := strings.ToLower(newMessage)
	existingLower := strings.ToLower(existingContent)

	// Extract message content (remove timestamp/project prefix from existing)
	existingMsg := extractMessageContent(existingLower)

	// Negation patterns that indicate contradiction
	negationPatterns := []string{
		"never", "don't", "do not", "avoid", "shouldn't",
		"should not", "can't", "cannot", "won't", "will not",
		"mustn't", "must not",
	}

	// Opposing action pairs
	opposingPairs := [][2]string{
		{"use", "avoid"},
		{"always", "never"},
		{"do", "don't"},
		{"should", "shouldn't"},
		{"must", "mustn't"},
		{"can", "can't"},
		{"will", "won't"},
		{"include", "exclude"},
		{"enable", "disable"},
		{"allow", "prevent"},
		{"prefer", "avoid"},
	}

	// Check for negation patterns in new message combined with similarity
	hasNegationInNew := false

	for _, pattern := range negationPatterns {
		if strings.Contains(newLower, pattern) {
			hasNegationInNew = true
			break
		}
	}

	hasNegationInExisting := false

	for _, pattern := range negationPatterns {
		if strings.Contains(existingMsg, pattern) {
			hasNegationInExisting = true
			break
		}
	}

	// If one has negation and the other doesn't, likely contradiction
	if hasNegationInNew != hasNegationInExisting {
		return "contradiction"
	}

	// Check for opposing action pairs
	for _, pair := range opposingPairs {
		// Check if new has first action and existing has second
		if (strings.Contains(newLower, pair[0]) && strings.Contains(existingMsg, pair[1])) ||
			(strings.Contains(newLower, pair[1]) && strings.Contains(existingMsg, pair[0])) {
			return "contradiction"
		}
	}

	// No contradiction detected, must be duplicate
	return "duplicate"
}

// extractMessageContent removes timestamp and project prefix from memory content.
func extractMessageContent(content string) string {
	// Format: - YYYY-MM-DD HH:MM: [project] message
	// Remove leading "- "
	content = strings.TrimPrefix(content, "- ")

	// Remove timestamp (YYYY-MM-DD HH:MM:)
	if idx := strings.Index(content, ": "); idx > 0 {
		content = content[idx+2:]
	}

	// Remove project tag [project]
	if idx := strings.Index(content, "] "); idx > 0 {
		content = content[idx+2:]
	}

	return strings.TrimSpace(content)
}

// isLegacyCannedExtraction detects content-free canned strings from the old
// session extractors that stored observations instead of lessons.
func isLegacyCannedExtraction(s string) bool {
	lower := strings.ToLower(s)
	switch {
	case strings.HasPrefix(lower, "used ") && strings.HasSuffix(lower, " successfully in session"):
		return true
	case strings.HasPrefix(lower, "autonomously fixed"):
		return true
	case lower == "claude.md was edited":
		return true
	case strings.HasPrefix(lower, "consistently used ") && strings.HasSuffix(lower, " throughout session"):
		return true
	case strings.HasPrefix(lower, "tests passed using "):
		return true
	case strings.HasPrefix(lower, "session behavior aligns with:"):
		return true
	// Old verbose formats (pre-concise extractors)
	case strings.HasPrefix(lower, "frequently used command"):
		return true
	case strings.HasPrefix(lower, "successful outcome:\ncommand:"):
		return true
	case strings.HasPrefix(lower, "consistent use of '") && strings.Contains(lower, "across") && strings.Contains(lower, "messages"):
		return true
	case strings.HasPrefix(lower, "self-corrected") && strings.Contains(lower, "failure:\nfailed:"):
		return true
	}

	return false
}

// searchDirectory searches all files in a directory for a pattern.
func searchDirectory(dir, pattern, projectFilter string) []GrepMatch {
	var matches []GrepMatch

	entries, err := os.ReadDir(dir)
	if err != nil {
		return matches
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		fileMatches := searchFile(path, pattern, projectFilter)
		matches = append(matches, fileMatches...)
	}

	return matches
}

// searchFile searches a single file for a pattern.
func searchFile(path, pattern, projectFilter string) []GrepMatch {
	var matches []GrepMatch

	// If project filter is set, check if filename matches
	if projectFilter != "" {
		if !strings.Contains(filepath.Base(path), projectFilter) {
			return matches
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return matches
	}

	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), pattern) {
			matches = append(matches, GrepMatch{
				File:    path,
				LineNum: i + 1,
				Line:    line,
			})
		}
	}

	return matches
}

// stripMarkdownFormatting removes common markdown formatting characters.
func stripMarkdownFormatting(s string) string {
	s = strings.ReplaceAll(s, "**", "")
	s = strings.ReplaceAll(s, "*", "")
	s = strings.ReplaceAll(s, "`", "")
	// Strip leading list markers
	s = strings.TrimLeft(s, "- ")

	return strings.TrimSpace(s)
}
