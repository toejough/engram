package memory

import (
	"database/sql"
	"fmt"
	"time"
)

// FeedbackType represents the type of feedback for a memory entry.
type FeedbackType string

const (
	// FeedbackHelpful indicates the memory was useful
	FeedbackHelpful FeedbackType = "helpful"
	// FeedbackWrong indicates the memory was incorrect or not useful
	FeedbackWrong FeedbackType = "wrong"
	// FeedbackUnclear indicates the memory was unclear or confusing
	FeedbackUnclear FeedbackType = "unclear"
)

// FeedbackStats contains feedback statistics for a memory entry.
type FeedbackStats struct {
	HelpfulCount int
	WrongCount   int
	UnclearCount int
}

// FlaggedEntry represents a memory entry flagged for review or rewrite.
type FlaggedEntry struct {
	ID      int64
	Content string
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
	defer rows.Close()

	var entries []FlaggedEntry
	for rows.Next() {
		var entry FlaggedEntry
		if err := rows.Scan(&entry.ID, &entry.Content); err != nil {
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
	defer rows.Close()

	var entries []FlaggedEntry
	for rows.Next() {
		var entry FlaggedEntry
		if err := rows.Scan(&entry.ID, &entry.Content); err != nil {
			return nil, fmt.Errorf("failed to scan flagged entry: %w", err)
		}
		entries = append(entries, entry)
	}

	return entries, rows.Err()
}
