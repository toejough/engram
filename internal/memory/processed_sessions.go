package memory

import (
	"database/sql"
	"fmt"
	"time"
)

// RecordProcessedSession records that a session has been processed, using INSERT OR REPLACE
// to allow updates to existing records.
func RecordProcessedSession(db *sql.DB, sessionID, project string, itemsFound int, status string) error {
	processedAt := time.Now().Format(time.RFC3339)
	_, err := db.Exec(`
		INSERT OR REPLACE INTO processed_sessions (session_id, project, processed_at, items_found, status)
		VALUES (?, ?, ?, ?, ?)
	`, sessionID, project, processedAt, itemsFound, status)
	if err != nil {
		return fmt.Errorf("failed to record processed session: %w", err)
	}
	return nil
}

// IsSessionProcessed checks if a session has already been processed.
func IsSessionProcessed(db *sql.DB, sessionID string) (bool, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM processed_sessions WHERE session_id = ?
	`, sessionID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check if session processed: %w", err)
	}
	return count > 0, nil
}

// ResetLastNSessions deletes the N most recently processed sessions (by processed_at DESC).
// Returns the number of rows deleted.
func ResetLastNSessions(db *sql.DB, n int) (int64, error) {
	result, err := db.Exec(`
		DELETE FROM processed_sessions
		WHERE session_id IN (
			SELECT session_id FROM processed_sessions
			ORDER BY processed_at DESC
			LIMIT ?
		)
	`, n)
	if err != nil {
		return 0, fmt.Errorf("failed to reset last N sessions: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}
	return rowsAffected, nil
}
