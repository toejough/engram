// Package memory provides memory management operations for storing learnings.
package memory

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// LearnOpts holds options for learning storage.
type LearnOpts struct {
	Message    string
	Project    string
	Source     string // "internal" or "external" (defaults to "internal")
	Type       string // "correction", "reflection", or empty for default
	MemoryRoot string
	Extractor  LLMExtractor // Optional LLM extractor for structured knowledge extraction (ISSUE-188)
}

// Learn stores a learning in the memory index.
func Learn(opts LearnOpts) error {
	if opts.Message == "" {
		return fmt.Errorf("message is required")
	}

	// Ensure memory directory exists
	if err := os.MkdirAll(opts.MemoryRoot, 0755); err != nil {
		return fmt.Errorf("failed to create memory directory: %w", err)
	}

	indexPath := filepath.Join(opts.MemoryRoot, "index.md")

	// Open file for appending (create if doesn't exist)
	f, err := os.OpenFile(indexPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open index file: %w", err)
	}
	defer func() { _ = f.Close() }()

	// Format entry: - YYYY-MM-DD HH:MM: [project] message
	timestamp := time.Now().Format("2006-01-02 15:04")
	var entry string
	if opts.Project != "" {
		entry = fmt.Sprintf("- %s: [%s] %s\n", timestamp, opts.Project, opts.Message)
	} else {
		entry = fmt.Sprintf("- %s: %s\n", timestamp, opts.Message)
	}

	if _, err := f.WriteString(entry); err != nil {
		return fmt.Errorf("failed to write entry: %w", err)
	}

	// Also create embedding in DB
	if err := learnToEmbeddings(opts); err != nil {
		return fmt.Errorf("failed to create embedding: %w", err)
	}

	return nil
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

// Decide logs a decision with reasoning and alternatives.
func Decide(opts DecideOpts) (*DecideResult, error) {
	if opts.Context == "" {
		return nil, fmt.Errorf("context is required")
	}
	if opts.Choice == "" {
		return nil, fmt.Errorf("choice is required")
	}
	if opts.Reason == "" {
		return nil, fmt.Errorf("reason is required")
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
	entry := map[string]interface{}{
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

// SessionEndOpts holds options for session end summary.
type SessionEndOpts struct {
	Project    string
	MemoryRoot string
}

// SessionEndResult contains the result of creating a session summary.
type SessionEndResult struct {
	FilePath string
}

// SessionEnd generates a compressed session summary.
func SessionEnd(opts SessionEndOpts) (*SessionEndResult, error) {
	if opts.Project == "" {
		return nil, fmt.Errorf("project is required")
	}

	// Ensure sessions directory exists
	sessionsDir := filepath.Join(opts.MemoryRoot, "sessions")
	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create sessions directory: %w", err)
	}

	// Build filename: {DATE}-{PROJECT}.md
	today := time.Now().Format("2006-01-02")
	filename := fmt.Sprintf("%s-%s.md", today, opts.Project)
	filePath := filepath.Join(sessionsDir, filename)

	// Read today's decisions if they exist
	decisionsPath := filepath.Join(opts.MemoryRoot, "decisions", today+"-"+opts.Project+".jsonl")
	decisions := readDecisions(decisionsPath)

	// Generate summary
	summary := generateSessionSummary(opts.Project, today, decisions)

	// Ensure under 2000 characters
	if len(summary) > 2000 {
		summary = truncateSummary(summary, 2000)
	}

	// Write the summary
	if err := os.WriteFile(filePath, []byte(summary), 0644); err != nil {
		return nil, fmt.Errorf("failed to write session summary: %w", err)
	}

	return &SessionEndResult{
		FilePath: filePath,
	}, nil
}

// readDecisions reads decisions from a JSONL file.
func readDecisions(path string) []map[string]interface{} {
	var decisions []map[string]interface{}

	data, err := os.ReadFile(path)
	if err != nil {
		return decisions
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err == nil {
			decisions = append(decisions, entry)
		}
	}

	return decisions
}

// generateSessionSummary creates a markdown summary.
func generateSessionSummary(project, date string, decisions []map[string]interface{}) string {
	var sb strings.Builder

	sb.WriteString("# Session Summary\n\n")
	sb.WriteString(fmt.Sprintf("**Project:** %s\n", project))
	sb.WriteString(fmt.Sprintf("**Date:** %s\n\n", date))

	if len(decisions) > 0 {
		sb.WriteString("## Decisions\n\n")
		for i, d := range decisions {
			if i >= 5 { // Limit to 5 decisions for brevity
				sb.WriteString(fmt.Sprintf("... and %d more decisions\n", len(decisions)-5))
				break
			}
			choice, _ := d["choice"].(string)
			reason, _ := d["reason"].(string)
			// Truncate reason if too long
			if len(reason) > 50 {
				reason = reason[:47] + "..."
			}
			sb.WriteString(fmt.Sprintf("- **%s**: %s\n", choice, reason))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// truncateSummary truncates the summary to maxLen while keeping markdown valid.
func truncateSummary(summary string, maxLen int) string {
	if len(summary) <= maxLen {
		return summary
	}

	// Find a good break point
	truncated := summary[:maxLen-20]

	// Find the last newline
	lastNewline := strings.LastIndex(truncated, "\n")
	if lastNewline > maxLen/2 {
		truncated = truncated[:lastNewline]
	}

	return truncated + "\n\n...(truncated)\n"
}

// GrepOpts holds options for memory grep.
type GrepOpts struct {
	Pattern          string
	Project          string
	IncludeDecisions bool
	MemoryRoot       string
}

// GrepMatch represents a single grep match.
type GrepMatch struct {
	File    string
	LineNum int
	Line    string
}

// GrepResult contains the results of a grep search.
type GrepResult struct {
	Matches []GrepMatch
}

// Grep searches memory files for a pattern.
func Grep(opts GrepOpts) (*GrepResult, error) {
	if opts.Pattern == "" {
		return nil, fmt.Errorf("pattern is required")
	}

	var matches []GrepMatch
	pattern := strings.ToLower(opts.Pattern)

	// Search index.md
	indexPath := filepath.Join(opts.MemoryRoot, "index.md")
	matches = append(matches, searchFile(indexPath, pattern, "")...)

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

// QueryOpts holds options for memory query.
type QueryOpts struct {
	Text       string
	Limit      int
	Project    string // Project name for tracking retrievals
	MemoryRoot string
	ModelDir   string // Directory for ONNX models (default: ~/.claude/models)
}

// QueryResult represents a single query result.
type QueryResult struct {
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
	Results              []QueryResult
	VectorStorage        string
	EmbeddingModel       string
	APICallsMade         bool
	EmbeddingsCount      int
	NewEmbeddingsCreated int
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
}

// Query searches memory for semantically similar content using embeddings.
func Query(opts QueryOpts) (*QueryResults, error) {
	startTime := time.Now()

	if opts.Text == "" {
		return nil, fmt.Errorf("text is required")
	}

	limit := opts.Limit
	if limit == 0 {
		limit = 5
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
	if err := initializeONNXRuntime(modelDir); err != nil {
		return nil, fmt.Errorf("failed to initialize ONNX Runtime: %w", err)
	}

	// Model path
	modelPath := filepath.Join(modelDir, "e5-small-v2.onnx")

	// Check if model needs to be downloaded
	modelDownloaded := false
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		if err := downloadModel(modelPath); err != nil {
			return nil, fmt.Errorf("failed to download model: %w", err)
		}
		modelDownloaded = true
	}

	// Initialize embeddings database
	dbPath := filepath.Join(opts.MemoryRoot, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize embeddings database: %w", err)
	}
	defer func() { _ = db.Close() }()

	// Count existing embeddings before processing
	var existingCount int
	err = db.QueryRow("SELECT COUNT(*) FROM embeddings WHERE embedding_id IS NOT NULL").Scan(&existingCount)
	if err != nil {
		existingCount = 0
	}

	// Collect all memory content
	var contents []string

	// Read index.md
	indexPath := filepath.Join(opts.MemoryRoot, "index.md")
	if data, err := os.ReadFile(indexPath); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				contents = append(contents, line)
			}
		}
	}

	// Read session summaries
	sessionsDir := filepath.Join(opts.MemoryRoot, "sessions")
	if entries, err := os.ReadDir(sessionsDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				path := filepath.Join(sessionsDir, entry.Name())
				if data, err := os.ReadFile(path); err == nil {
					lines := strings.Split(string(data), "\n")
					for _, line := range lines {
						if strings.TrimSpace(line) != "" {
							contents = append(contents, line)
						}
					}
				}
			}
		}
	}

	// Create embeddings for new content using ONNX model
	newEmbeddings, sessionCreated, sessionReused, err := createEmbeddings(db, contents, modelPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create embeddings: %w", err)
	}

	// Generate query embedding using ONNX model
	queryEmbedding, querySessionCreated, querySessionReused, err := generateEmbeddingONNX(opts.Text, modelPath)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// If session wasn't created during content embedding, check query embedding
	if !sessionCreated && querySessionCreated {
		sessionCreated = true
		sessionReused = false
	} else if !sessionReused && querySessionReused {
		sessionReused = true
	}

	// Search using hybrid search (BM25 + vector + RRF)
	bm25Available := hasFTS5(db)
	results, err := hybridSearch(db, queryEmbedding, opts.Text, limit, 60)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	// Update retrieval tracking for TASK-41
	if err := updateRetrievalTracking(db, results, opts.Project); err != nil {
		// Log error but don't fail the query
		fmt.Fprintf(os.Stderr, "Warning: failed to update retrieval tracking: %v\n", err)
	}

	duration := time.Since(startTime)

	return &QueryResults{
		Results:              results,
		VectorStorage:        "sqlite-vec",
		EmbeddingModel:       "e5-small-v2",
		EmbeddingDimensions:  384,
		APICallsMade:         false,
		UsedONNXRuntime:      true,
		ModelDownloaded:      modelDownloaded,
		ModelPath:            modelPath,
		ModelLoaded:          true,
		ModelType:            "onnx",
		InferenceExecuted:    true,
		UsedMockEmbeddings:   false,
		EmbeddingsCount:      existingCount + newEmbeddings,
		NewEmbeddingsCreated: newEmbeddings,
		SessionCreatedNew:    sessionCreated,
		SessionReused:        sessionReused,
		QueryDuration:        duration,
		UsedHybridSearch:     true,
		BM25Enabled:          bm25Available,
	}, nil
}

// PromoteOpts holds options for memory promotion.
type PromoteOpts struct {
	MemoryRoot    string
	MinRetrievals int // Minimum retrieval count (default: 3)
	MinProjects   int // Minimum unique projects (default: 2)
}

// PromoteCandidate represents a candidate for promotion.
type PromoteCandidate struct {
	Content        string
	RetrievalCount int
	UniqueProjects int
}

// PromoteResult contains the result of memory promotion.
type PromoteResult struct {
	Candidates []PromoteCandidate
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

// LearnConflictResult contains the result of learning with conflict check.
type LearnConflictResult struct {
	HasConflict   bool
	ConflictEntry string
	Similarity    float64
	ConflictType  string // "duplicate" or "contradiction"
	Stored        bool
}

// Promote identifies memory entries that meet retrieval thresholds for promotion to global memory.
func Promote(opts PromoteOpts) (*PromoteResult, error) {
	if opts.MemoryRoot == "" {
		return nil, fmt.Errorf("memory root is required")
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
		var content string
		var retrievalCount int
		var projectsRetrieved string

		if err := rows.Scan(&content, &retrievalCount, &projectsRetrieved); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Count unique projects
		uniqueProjects := 0
		if projectsRetrieved != "" {
			projectMap := make(map[string]bool)
			projects := strings.Split(projectsRetrieved, ",")
			for _, p := range projects {
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
		if err := rows.Scan(&e.id, &e.embeddingID); err != nil {
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

// PromoteInteractiveOpts holds options for interactive memory promotion.
type PromoteInteractiveOpts struct {
	MemoryRoot    string
	MinRetrievals int    // Minimum retrieval count (default: 3)
	MinProjects   int    // Minimum unique projects (default: 2)
	Review        bool   // Enable interactive review mode
	ReviewFunc    func(PromoteCandidate) (bool, error) // Function to review each candidate
	ClaudeMDPath  string // Path to CLAUDE.md (default: ~/.claude/CLAUDE.md)
}

// PromoteInteractiveResult contains the result of interactive memory promotion.
type PromoteInteractiveResult struct {
	CandidatesReviewed int
	CandidatesApproved int
	CandidatesRejected int
}

// PromoteInteractive identifies memory entries for promotion and optionally reviews them interactively.
func PromoteInteractive(opts PromoteInteractiveOpts) (*PromoteInteractiveResult, error) {
	if opts.MemoryRoot == "" {
		return nil, fmt.Errorf("memory root is required")
	}

	// If review mode is enabled, ReviewFunc is required
	if opts.Review && opts.ReviewFunc == nil {
		return nil, fmt.Errorf("review function is required when review mode is enabled")
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
		if err := appendToClaudeMD(claudeMDPath, approvedLearnings); err != nil {
			return nil, fmt.Errorf("failed to append to CLAUDE.md: %w", err)
		}
	}

	return result, nil
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

// RemoveFromClaudeMD removes entries from the "## Promoted Learnings" section of CLAUDE.md.
// Each entry in entries is matched as a substring against the lines in the section.
// Lines containing any of the entries are removed. Other sections are untouched.
// Returns nil if the file doesn't exist or is empty.
func RemoveFromClaudeMD(claudeMDPath string, entries []string) error {
	content, err := os.ReadFile(claudeMDPath)
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

	return os.WriteFile(claudeMDPath, []byte(strings.Join(result, "\n")), 0644)
}

// LearnWithConflictCheck stores a learning but first checks for similar existing entries.
func LearnWithConflictCheck(opts LearnOpts) (*LearnConflictResult, error) {
	if opts.Message == "" {
		return nil, fmt.Errorf("message is required")
	}

	// First check for conflicts using Query
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	modelDir := filepath.Join(homeDir, ".claude", "models")

	// Initialize ONNX Runtime and download model if needed
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create model directory: %w", err)
	}
	if err := initializeONNXRuntime(modelDir); err != nil {
		return nil, fmt.Errorf("failed to initialize ONNX Runtime: %w", err)
	}
	modelPath := filepath.Join(modelDir, "e5-small-v2.onnx")
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		if err := downloadModel(modelPath); err != nil {
			return nil, fmt.Errorf("failed to download model: %w", err)
		}
	}

	// Open DB to check for similar entries
	dbPath := filepath.Join(opts.MemoryRoot, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer func() { _ = db.Close() }()

	// Generate embedding for the new message
	embedding, _, _, err := generateEmbeddingONNX(opts.Message, modelPath)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Search for similar entries
	results, err := searchSimilar(db, embedding, 1)
	if err != nil {
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

// ============================================================================
// TASK-9: ACT-R activation scoring
// ============================================================================

// ActivationStatsOpts holds options for getting activation statistics.
type ActivationStatsOpts struct {
	MemoryRoot string
	Content    string // Substring to match in content
}

// ActivationStats contains activation statistics for a memory entry.
type ActivationStats struct {
	Activation       float64 // ACT-R base-level activation B_i
	RetrievalCount   int
	TimestampCount   int // Total timestamps recorded
	ActiveTimestamps int // Timestamps within retention window
	DecayParameter   float64
	SessionCount     int     // Number of distinct retrieval sessions
	SessionBonus     float64 // Multiplier bonus from cross-session retrievals
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

// GetActivationStats retrieves ACT-R activation statistics for a memory entry.
func GetActivationStats(opts ActivationStatsOpts) (*ActivationStats, error) {
	dbPath := filepath.Join(opts.MemoryRoot, "embeddings.db")
	db, err := initEmbeddingsDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer func() { _ = db.Close() }()

	// Find entry by content substring
	var retrievalTimestamps, memoryType string
	var retrievalCount int
	query := "SELECT retrieval_timestamps, retrieval_count, memory_type FROM embeddings WHERE content LIKE ? LIMIT 1"
	err = db.QueryRow(query, "%"+opts.Content+"%").Scan(&retrievalTimestamps, &retrievalCount, &memoryType)
	if err != nil {
		return nil, fmt.Errorf("failed to find entry: %w", err)
	}

	// Parse timestamps
	var timestamps []string
	if retrievalTimestamps != "" {
		if err := json.Unmarshal([]byte(retrievalTimestamps), &timestamps); err != nil {
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

	return &ActivationStats{
		Activation:       activation,
		RetrievalCount:   retrievalCount,
		TimestampCount:   len(timestamps),
		ActiveTimestamps: activeTimestamps,
		DecayParameter:   d,
		SessionCount:     sessionCount,
		SessionBonus:     sessionBonus,
	}, nil
}

// SimulateTimeOpts holds options for simulating time passage (test-only).
type SimulateTimeOpts struct {
	MemoryRoot string
	Content    string
	DaysToAge  int
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
		if err := json.Unmarshal([]byte(retrievalTimestamps), &timestamps); err != nil {
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

// MigrateToACTROpts holds options for migrating to ACT-R.
type MigrateToACTROpts struct {
	MemoryRoot string
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
	updated := 0
	for rows.Next() {
		var id int64
		var retrievalCount int
		if err := rows.Scan(&id, &retrievalCount); err != nil {
			continue
		}

		// Initialize timestamps based on retrieval count
		// If retrieval_count > 0, create that many timestamps spread over past 30 days
		var timestamps []string
		if retrievalCount > 0 {
			for i := 0; i < retrievalCount; i++ {
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
		result, err := db.Exec(updateStmt, string(timestampsJSON), id)
		if err == nil {
			affected, _ := result.RowsAffected()
			updated += int(affected)
		}
	}

	if err := rows.Err(); err != nil {
		return err
	}

	// Note: It's OK if updated == 0, that just means all entries already have timestamps
	return nil
}

// ClearTimestampsOpts holds options for clearing timestamps (test-only).
type ClearTimestampsOpts struct {
	MemoryRoot string
	Content    string
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
