package memory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
)

// LeechCandidate represents a memory classified as a leech (high surfacing, low impact).
type LeechCandidate struct {
	MemoryID        int64
	Content         string
	LeechCount      int
	ImportanceScore float64
	ImpactScore     float64
	SurfacingCount  int
}

// LeechDiagnosis holds the root cause analysis for a leech memory.
type LeechDiagnosis struct {
	MemoryID         int64
	Content          string
	DiagnosisType    string // "wrong_tier", "enforcement_gap", "content_quality", "retrieval_mismatch", "insufficient_data"
	Signal           string // Evidence description
	ProposedAction   string // "rewrite", "narrow_scope", "promote_to_claude_md", "convert_to_hook"
	SuggestedContent string // Rewritten content (for rewrite action only); empty from DiagnoseLeech
	Recommendation   *Recommendation
	Embedder         func(text string) ([]float32, error) // Optional: inject for testing; nil uses ONNX
}

// Recommendation describes an action proposal for CLAUDE.md, hook, or other system changes.
type Recommendation struct {
	Category    string // e.g., "claude-md-promotion", "hook-conversion", "claude-md-demotion-to-hook"
	Description string // leech diagnosis: what to add/enforce
	Evidence    string // leech diagnosis: evidence from surfacing history
	Text        string // quality gate: human-readable recommendation text
}

// ApplyLeechAction executes a user-approved leech remediation action.
// Memory-internal actions (rewrite, narrow_scope) execute within a single DB transaction.
// Non-memory actions (promote_to_claude_md, convert_to_hook) mark the memory as "action_recommended".
func ApplyLeechAction(db *sql.DB, diagnosis LeechDiagnosis, fs FileSystem) error {
	switch diagnosis.ProposedAction {
	case "rewrite", "narrow_scope":
		return applyReembedAction(db, diagnosis)
	case "promote_to_claude_md", "convert_to_hook":
		return markActionRecommended(db, diagnosis.MemoryID)
	default:
		return fmt.Errorf("ApplyLeechAction: unknown action %q", diagnosis.ProposedAction)
	}
}

// DiagnoseLeech analyzes surfacing_events history for a memory to determine root cause.
// Priority order: wrong_tier > enforcement_gap > content_quality > retrieval_mismatch.
// Returns "insufficient_data" when no surfacing events exist. Returns error if memory not found.
func DiagnoseLeech(db *sql.DB, memoryID int64) (*LeechDiagnosis, error) {
	var content string

	err := db.QueryRow("SELECT content FROM embeddings WHERE id = ?", memoryID).Scan(&content)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("DiagnoseLeech: memory %d not found", memoryID)
	}

	if err != nil {
		return nil, fmt.Errorf("DiagnoseLeech: query memory: %w", err)
	}

	diag := &LeechDiagnosis{
		MemoryID: memoryID,
		Content:  content,
	}

	// Fetch all surfacing events for this memory
	rows, err := db.Query(`
		SELECT haiku_relevant, faithfulness, outcome_signal, user_feedback
		FROM surfacing_events
		WHERE memory_id = ?
		ORDER BY timestamp ASC`, memoryID)
	if err != nil {
		return nil, fmt.Errorf("DiagnoseLeech: query events: %w", err)
	}

	defer func() { _ = rows.Close() }()

	type eventStats struct {
		haikuRelevant *bool
		faithfulness  *float64
		userFeedback  string
	}

	var events []eventStats

	for rows.Next() {
		var (
			e                           eventStats
			haikuRelevant               sql.NullBool
			faithfulness                sql.NullFloat64
			outcomeSignal, userFeedback sql.NullString
		)

		err := rows.Scan(&haikuRelevant, &faithfulness, &outcomeSignal, &userFeedback)
		if err != nil {
			return nil, fmt.Errorf("DiagnoseLeech: scan event: %w", err)
		}

		if haikuRelevant.Valid {
			v := haikuRelevant.Bool
			e.haikuRelevant = &v
		}

		if faithfulness.Valid {
			v := faithfulness.Float64
			e.faithfulness = &v
		}

		// outcomeSignal scanned but not yet consumed by diagnosis logic

		if userFeedback.Valid {
			e.userFeedback = userFeedback.String
		}

		events = append(events, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("DiagnoseLeech: rows: %w", err)
	}

	if len(events) == 0 {
		diag.DiagnosisType = "insufficient_data"
		diag.Signal = "No surfacing events recorded for this memory"

		return diag, nil
	}

	// Categorize events by signal combination
	var (
		wrongLowFaithCount  int // user_feedback='wrong' AND (faithfulness<0.3 or unscored)
		wrongHighFaithCount int // user_feedback='wrong' AND faithfulness>=0.3 (agent used it)
		lowFaithCount       int // faithfulness < 0.3 (regardless of user feedback)
		scoredCount         int // events with faithfulness set
	)

	for _, e := range events {
		hasFaith := e.faithfulness != nil
		isLowFaith := hasFaith && *e.faithfulness < 0.3
		isHighFaith := hasFaith && *e.faithfulness >= 0.3

		if hasFaith {
			scoredCount++
		}

		if isLowFaith {
			lowFaithCount++
		}

		if e.userFeedback == "wrong" {
			if isHighFaith {
				wrongHighFaithCount++
			} else {
				// wrong feedback with low/unscored faithfulness → memory not used
				wrongLowFaithCount++
			}
		}
	}

	// --- Priority 1: wrong_tier ---
	// Signal: user corrected in multiple events AND agent didn't use the memory
	// (faithfulness low or absent) → memory needs to be front-loaded (CLAUDE.md)
	if wrongLowFaithCount >= 2 {
		diag.DiagnosisType = "wrong_tier"
		diag.Signal = fmt.Sprintf("Surfaced %d times with user corrections and low/absent faithfulness — memory needs to be front-loaded", wrongLowFaithCount)
		diag.ProposedAction = "promote_to_claude_md"
		diag.Recommendation = &Recommendation{
			Category:    "claude-md-promotion",
			Description: fmt.Sprintf("Add the following content to CLAUDE.md in the appropriate section: %q", content),
			Evidence:    diag.Signal,
		}
		fmt.Fprintf(os.Stderr, "[memory:leech] diagnosed memory %d: %s (%s)\n", memoryID, diag.DiagnosisType, diag.ProposedAction)

		return diag, nil
	}

	// --- Priority 2: enforcement_gap ---
	// Signal: agent referenced memory content (faithfulness >= 0.3) but user still corrected
	// → guidance understood but not enforced strictly enough → convert to hook
	if wrongHighFaithCount >= 1 {
		diag.DiagnosisType = "enforcement_gap"
		diag.Signal = fmt.Sprintf("Agent used memory content (faithfulness >= 0.3) in %d event(s) but user still corrected — guidance understood but not enforced", wrongHighFaithCount)
		diag.ProposedAction = "convert_to_hook"
		diag.Recommendation = &Recommendation{
			Category:    "hook-conversion",
			Description: fmt.Sprintf("Convert the following rule into an enforced hook: %q. Add a PreToolUse hook that warns or blocks when this pattern is violated.", content),
			Evidence:    diag.Signal,
		}
		fmt.Fprintf(os.Stderr, "[memory:leech] diagnosed memory %d: %s (%s)\n", memoryID, diag.DiagnosisType, diag.ProposedAction)

		return diag, nil
	}

	// --- Priority 3: content_quality ---
	// Signal: majority of scored surfacings have low faithfulness (< 0.3) and no agent reference
	if scoredCount >= 1 && lowFaithCount > 0 && lowFaithCount*2 > scoredCount {
		diag.DiagnosisType = "content_quality"
		diag.Signal = fmt.Sprintf("Faithfulness < 0.3 in %d of %d scored surfacings — memory content is unclear or not actionable", lowFaithCount, scoredCount)
		diag.ProposedAction = "rewrite"
		// SuggestedContent left empty; caller uses PreviewLeechRewrite to populate
		fmt.Fprintf(os.Stderr, "[memory:leech] diagnosed memory %d: %s (%s)\n", memoryID, diag.DiagnosisType, diag.ProposedAction)

		return diag, nil
	}

	// --- Priority 4: retrieval_mismatch ---
	// Signal: haiku_relevant=false on >50% of surfacings
	totalWithHaiku := 0
	notRelevantCount := 0

	for _, e := range events {
		if e.haikuRelevant != nil {
			totalWithHaiku++

			if !*e.haikuRelevant {
				notRelevantCount++
			}
		}
	}

	if totalWithHaiku >= 2 && notRelevantCount*2 > totalWithHaiku {
		diag.DiagnosisType = "retrieval_mismatch"
		diag.Signal = fmt.Sprintf("Haiku rated irrelevant in %d of %d surfacings (>50%%) — E5 matches broadly but content isn't contextually relevant", notRelevantCount, totalWithHaiku)
		diag.ProposedAction = "narrow_scope"
		fmt.Fprintf(os.Stderr, "[memory:leech] diagnosed memory %d: %s (%s)\n", memoryID, diag.DiagnosisType, diag.ProposedAction)

		return diag, nil
	}

	// Fallback: insufficient data to diagnose
	diag.DiagnosisType = "insufficient_data"
	diag.Signal = "Not enough surfacing history to determine root cause"
	fmt.Fprintf(os.Stderr, "[memory:leech] diagnosed memory %d: %s\n", memoryID, diag.DiagnosisType)

	return diag, nil
}

// FormatLeechDiagnosis formats a LeechDiagnosis for CLI display.
func FormatLeechDiagnosis(diag *LeechDiagnosis) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Memory ID: %d\n", diag.MemoryID))
	sb.WriteString(fmt.Sprintf("Content: %s\n", diag.Content))
	sb.WriteString(fmt.Sprintf("Diagnosis: %s\n", diag.DiagnosisType))
	sb.WriteString(fmt.Sprintf("Signal: %s\n", diag.Signal))

	if diag.ProposedAction != "" {
		sb.WriteString(fmt.Sprintf("Proposed Action: %s\n", diag.ProposedAction))
	}

	if diag.SuggestedContent != "" {
		sb.WriteString(fmt.Sprintf("Suggested Content:\n  %s\n", diag.SuggestedContent))
	}

	if diag.Recommendation != nil {
		sb.WriteString(fmt.Sprintf("Recommendation [%s]: %s\n", diag.Recommendation.Category, diag.Recommendation.Description))
	}

	return sb.String()
}

// GetLeeches returns all memories where quadrant='leech' AND leech_count >= leech_threshold.
func GetLeeches(db *sql.DB) ([]LeechCandidate, error) {
	threshStr, err := getMetadata(db, "leech_threshold")
	if err != nil {
		return nil, fmt.Errorf("GetLeeches: read leech_threshold: %w", err)
	}

	threshold := 5

	if threshStr != "" {
		if v, parseErr := strconv.Atoi(threshStr); parseErr == nil {
			threshold = v
		}
	}

	rows, err := db.Query(`
		SELECT e.id, e.content, e.leech_count, e.importance_score, e.impact_score,
		       COUNT(se.id) as surfacing_count
		FROM embeddings e
		LEFT JOIN surfacing_events se ON se.memory_id = e.id
		WHERE e.quadrant = 'leech' AND e.leech_count >= ?
		GROUP BY e.id`, threshold)
	if err != nil {
		return nil, fmt.Errorf("GetLeeches: query: %w", err)
	}

	defer func() { _ = rows.Close() }()

	var candidates []LeechCandidate

	for rows.Next() {
		var c LeechCandidate

		err := rows.Scan(&c.MemoryID, &c.Content, &c.LeechCount, &c.ImportanceScore, &c.ImpactScore, &c.SurfacingCount)
		if err != nil {
			return nil, fmt.Errorf("GetLeeches: scan: %w", err)
		}

		candidates = append(candidates, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetLeeches: rows: %w", err)
	}

	return candidates, nil
}

// PreviewLeechRewrite generates a rewritten version of a leech memory's content via LLM.
// Only valid for content_quality diagnoses. Returns empty string on LLM failure (not an error).
// Returns error if called for a non-content_quality diagnosis.
func PreviewLeechRewrite(db *sql.DB, diagnosis LeechDiagnosis, llm LLMExtractor) (string, error) {
	if diagnosis.DiagnosisType != "content_quality" {
		return "", fmt.Errorf("PreviewLeechRewrite: only valid for content_quality diagnoses, got %q", diagnosis.DiagnosisType)
	}

	ctx := context.Background()

	rewritten, err := llm.Rewrite(ctx, diagnosis.Content)
	if err != nil {
		// LLM failure: return empty string, not an error
		return "", nil
	}

	return rewritten, nil
}

// applyReembedAction updates memory content and re-embeds within a transaction.
// If re-embedding fails, the transaction is rolled back and original content is preserved.
func applyReembedAction(db *sql.DB, diagnosis LeechDiagnosis) error {
	newContent := diagnosis.SuggestedContent
	if newContent == "" {
		return fmt.Errorf("ApplyLeechAction: %s action requires SuggestedContent", diagnosis.ProposedAction)
	}

	// Get current embedding_id for the memory
	var oldEmbeddingID sql.NullInt64

	err := db.QueryRow("SELECT embedding_id FROM embeddings WHERE id = ?", diagnosis.MemoryID).Scan(&oldEmbeddingID)
	if err != nil {
		return fmt.Errorf("ApplyLeechAction: get embedding_id: %w", err)
	}

	if !oldEmbeddingID.Valid || oldEmbeddingID.Int64 == 0 {
		return fmt.Errorf("ApplyLeechAction: memory %d has no embedding row", diagnosis.MemoryID)
	}

	// Generate new embedding via injected Embedder or ONNX fallback.
	embed := diagnosis.Embedder
	if embed == nil {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("ApplyLeechAction: get home dir: %w", err)
		}

		modelDir := filepath.Join(homeDir, ".claude", "models")
		modelPath := filepath.Join(modelDir, "e5-small-v2.onnx")

		if err := initializeONNXRuntime(modelDir); err != nil {
			return fmt.Errorf("ApplyLeechAction: init ONNX: %w", err)
		}

		embed = func(text string) ([]float32, error) {
			emb, _, _, onnxErr := generateEmbeddingONNX(text, modelPath)
			return emb, onnxErr
		}
	}

	embedding, err := embed("passage: " + newContent)
	if err != nil {
		return fmt.Errorf("ApplyLeechAction: generate embedding: %w", err)
	}

	embeddingBlob, err := sqlite_vec.SerializeFloat32(embedding)
	if err != nil {
		return fmt.Errorf("ApplyLeechAction: serialize embedding: %w", err)
	}

	// Execute within a transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("ApplyLeechAction: begin transaction: %w", err)
	}

	defer func() { _ = tx.Rollback() }()

	// Delete old vec row
	if _, err := tx.Exec("DELETE FROM vec_embeddings WHERE rowid = ?", oldEmbeddingID.Int64); err != nil {
		return fmt.Errorf("ApplyLeechAction: delete old vec: %w", err)
	}

	// Insert new vec row
	result, err := tx.Exec("INSERT INTO vec_embeddings(embedding) VALUES (?)", embeddingBlob)
	if err != nil {
		return fmt.Errorf("ApplyLeechAction: insert new vec: %w", err)
	}

	newEmbeddingID, _ := result.LastInsertId()

	// Update embeddings table
	if _, err := tx.Exec("UPDATE embeddings SET content = ?, embedding_id = ? WHERE id = ?",
		newContent, newEmbeddingID, diagnosis.MemoryID); err != nil {
		return fmt.Errorf("ApplyLeechAction: update content: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("ApplyLeechAction: commit: %w", err)
	}

	// Refresh FTS5 outside transaction (best-effort)
	deleteFTS5(db, diagnosis.MemoryID)
	insertFTS5(db, diagnosis.MemoryID, newContent)

	return nil
}

// markActionRecommended marks a leech memory as having a non-memory action recommended.
func markActionRecommended(db *sql.DB, memoryID int64) error {
	_, err := db.Exec("UPDATE embeddings SET leech_action = 'action_recommended' WHERE id = ?", memoryID)
	if err != nil {
		return fmt.Errorf("ApplyLeechAction: mark action_recommended: %w", err)
	}

	return nil
}
