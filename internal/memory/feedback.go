package memory

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Exported constants.
const (
	// FeedbackHelpful indicates the memory was useful
	FeedbackHelpful FeedbackType = "helpful"
	// FeedbackUnclear indicates the memory was unclear or confusing
	FeedbackUnclear FeedbackType = "unclear"
	// FeedbackWrong indicates the memory was incorrect or not useful
	FeedbackWrong FeedbackType = "wrong"
)

// FeedbackStats contains feedback statistics for a memory entry.
type FeedbackStats struct {
	HelpfulCount int
	WrongCount   int
	UnclearCount int
}

// FeedbackType represents the type of feedback for a memory entry.
type FeedbackType string

// FlaggedEntry represents a memory entry flagged for review or rewrite.
type FlaggedEntry struct {
	ID      int64
	Content string
}

// GetFeedbackStats retrieves feedback statistics for a memory entry.
func GetFeedbackStats(db *sql.DB, embeddingID int64) (*FeedbackStats, error) {
	query := `
		SELECT
			COUNT(CASE WHEN feedback_type = 'helpful' THEN 1 END) as helpful_count,
			COUNT(CASE WHEN feedback_type = 'wrong' THEN 1 END) as wrong_count,
			COUNT(CASE WHEN feedback_type = 'unclear' THEN 1 END) as unclear_count
		FROM feedback
		WHERE embedding_id = ?
	`

	var stats FeedbackStats

	err := db.QueryRow(query, embeddingID).Scan(&stats.HelpfulCount, &stats.WrongCount, &stats.UnclearCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get feedback stats: %w", err)
	}

	return &stats, nil
}

// ListFlaggedForReview returns all memory entries flagged for review.
func ListFlaggedForReview(db *sql.DB) ([]FlaggedEntry, error) {
	query := `SELECT id, content FROM embeddings WHERE flagged_for_review = 1 ORDER BY id`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query flagged entries: %w", err)
	}

	defer func() { _ = rows.Close() }()

	var entries []FlaggedEntry

	for rows.Next() {
		var entry FlaggedEntry

		err := rows.Scan(&entry.ID, &entry.Content)
		if err != nil {
			return nil, fmt.Errorf("failed to scan flagged entry: %w", err)
		}

		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

// ListFlaggedForRewrite returns all memory entries flagged for rewrite.
func ListFlaggedForRewrite(db *sql.DB) ([]FlaggedEntry, error) {
	query := `SELECT id, content FROM embeddings WHERE flagged_for_rewrite = 1 ORDER BY id`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query flagged entries: %w", err)
	}

	defer func() { _ = rows.Close() }()

	var entries []FlaggedEntry

	for rows.Next() {
		var entry FlaggedEntry

		err := rows.Scan(&entry.ID, &entry.Content)
		if err != nil {
			return nil, fmt.Errorf("failed to scan flagged entry: %w", err)
		}

		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

// PropagateEmbeddingFeedbackToSkills propagates negative feedback from source embeddings to their derived skills.
func PropagateEmbeddingFeedbackToSkills(db *sql.DB) (int, error) {
	// Query all non-pruned skills with their source_memory_ids and feedback_propagated_at
	query := `
		SELECT id, source_memory_ids, feedback_propagated_at, alpha, beta, retrieval_count, last_retrieved
		FROM generated_skills
		WHERE pruned = 0
	`

	rows, err := db.Query(query)
	if err != nil {
		return 0, fmt.Errorf("failed to query skills: %w", err)
	}

	defer func() { _ = rows.Close() }()

	type skillRecord struct {
		id                   int64
		sourceMemoryIDs      string
		feedbackPropagatedAt string
		alpha                float64
		beta                 float64
		retrievalCount       int
		lastRetrieved        sql.NullString
	}

	var skills []skillRecord

	for rows.Next() {
		var s skillRecord

		err := rows.Scan(&s.id, &s.sourceMemoryIDs, &s.feedbackPropagatedAt, &s.alpha, &s.beta, &s.retrievalCount, &s.lastRetrieved)
		if err != nil {
			continue
		}

		skills = append(skills, s)
	}

	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("error iterating skills: %w", err)
	}

	affected := 0

	for _, skill := range skills {
		// Skip if already propagated
		if skill.feedbackPropagatedAt != "" {
			continue
		}

		// Parse source_memory_ids JSON array
		var sourceIDs []int64

		err := json.Unmarshal([]byte(skill.sourceMemoryIDs), &sourceIDs)
		if err != nil {
			continue
		}

		if len(sourceIDs) == 0 {
			continue
		}

		// Check if any source is flagged
		hasFlaggedSource := false

		for _, sourceID := range sourceIDs {
			var flaggedForReview int

			err := db.QueryRow("SELECT flagged_for_review FROM embeddings WHERE id = ?", sourceID).Scan(&flaggedForReview)
			if err == nil && flaggedForReview == 1 {
				hasFlaggedSource = true
				break
			}
		}

		if !hasFlaggedSource {
			continue
		}

		// Increment beta by 1.0
		newBeta := skill.beta + 1.0

		// Recompute utility using computeUtility from optimize.go
		newUtility := computeUtility(skill.alpha, newBeta, skill.retrievalCount, skill.lastRetrieved.String)

		// Set feedback_propagated_at to current RFC3339 timestamp
		now := time.Now().Format(time.RFC3339)

		// Update skill
		_, err = db.Exec(`
			UPDATE generated_skills
			SET beta = ?, utility = ?, feedback_propagated_at = ?
			WHERE id = ?
		`, newBeta, newUtility, now, skill.id)
		if err != nil {
			continue
		}

		affected++
	}

	return affected, nil
}

// RecordFeedback records user feedback for a memory entry and adjusts confidence.
func RecordFeedback(db *sql.DB, embeddingID int64, feedbackType FeedbackType) error {
	timestamp := time.Now().Format(time.RFC3339)

	// Insert feedback record
	insertStmt := `INSERT INTO feedback (embedding_id, feedback_type, created_at) VALUES (?, ?, ?)`

	_, err := db.Exec(insertStmt, embeddingID, string(feedbackType), timestamp)
	if err != nil {
		return fmt.Errorf("failed to insert feedback: %w", err)
	}

	// Adjust confidence and flags based on feedback type
	switch feedbackType {
	case FeedbackHelpful:
		// Increase confidence by 0.05, capped at 1.0
		updateStmt := `UPDATE embeddings SET confidence = MIN(1.0, confidence + 0.05) WHERE id = ?`

		_, err = db.Exec(updateStmt, embeddingID)
		if err != nil {
			return fmt.Errorf("failed to update confidence: %w", err)
		}

	case FeedbackWrong:
		// Decrease confidence by 0.1, set flagged_for_review
		updateStmt := `UPDATE embeddings
			SET confidence = MAX(0.0, confidence - 0.1), flagged_for_review = 1
			WHERE id = ?`

		_, err = db.Exec(updateStmt, embeddingID)
		if err != nil {
			return fmt.Errorf("failed to update confidence and flag: %w", err)
		}

	case FeedbackUnclear:
		// Set flagged_for_rewrite (no confidence change)
		updateStmt := `UPDATE embeddings SET flagged_for_rewrite = 1 WHERE id = ?`

		_, err = db.Exec(updateStmt, embeddingID)
		if err != nil {
			return fmt.Errorf("failed to flag for rewrite: %w", err)
		}
	}

	return nil
}
