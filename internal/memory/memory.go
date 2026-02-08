// Package memory provides memory management operations for storing learnings.
package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LearnOpts holds options for learning storage.
type LearnOpts struct {
	Message    string
	Project    string
	Source     string // "internal" or "external" (defaults to "internal")
	MemoryRoot string
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
	Content    string
	Score      float64
	Source     string  // File source ("memory")
	SourceType string  // "internal" or "external"
	Confidence float64 // Confidence score (0.0-1.0)
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

	// Search for similar embeddings
	results, err := searchSimilar(db, queryEmbedding, limit)
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
	Factor     float64 // Decay factor (default: 0.9)
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

	// Get embedding_ids to delete from vec_embeddings
	rows, err := db.Query("SELECT embedding_id FROM embeddings WHERE confidence < ?", threshold)
	if err != nil {
		return nil, fmt.Errorf("failed to query for pruning: %w", err)
	}

	var embeddingIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("failed to scan embedding_id: %w", err)
		}
		embeddingIDs = append(embeddingIDs, id)
	}
	_ = rows.Close()

	// Delete from embeddings table
	deleteMetaStmt := `DELETE FROM embeddings WHERE confidence < ?`
	result, err := db.Exec(deleteMetaStmt, threshold)
	if err != nil {
		return nil, fmt.Errorf("failed to prune entries: %w", err)
	}

	rowsDeleted, _ := result.RowsAffected()

	// Delete from vec_embeddings table
	for _, embID := range embeddingIDs {
		_, _ = db.Exec("DELETE FROM vec_embeddings WHERE rowid = ?", embID)
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

	if len(results) > 0 {
		similarity = results[0].Score
		if similarity > 0.85 {
			hasConflict = true
			conflictEntry = results[0].Content
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
		Stored:        true,
	}, nil
}

