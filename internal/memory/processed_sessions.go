package memory

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

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

// ResetLastNSessions deletes the N most recently processed sessions (by processed_at DESC).
// Also deletes their offset files so reprocessing starts from byte 0.
// Returns the number of rows deleted.
func ResetLastNSessions(db *sql.DB, n int, memoryRoot string) (int64, error) {
	// Query session IDs being reset so we can delete their offset files
	rows, err := db.Query(`
		SELECT session_id FROM processed_sessions
		ORDER BY processed_at DESC
		LIMIT ?
	`, n)
	if err != nil {
		return 0, fmt.Errorf("failed to query sessions to reset: %w", err)
	}

	var sessionIDs []string

	for rows.Next() {
		var id string

		err := rows.Scan(&id)
		if err != nil {
			_ = rows.Close()
			return 0, fmt.Errorf("failed to scan session ID: %w", err)
		}

		sessionIDs = append(sessionIDs, id)
	}

	_ = rows.Close()

	// Delete offset files
	offsetDir := filepath.Join(memoryRoot, "offsets")
	for _, id := range sessionIDs {
		_ = os.Remove(filepath.Join(offsetDir, id+".offset"))
	}

	// Delete DB rows
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
