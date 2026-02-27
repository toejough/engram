package memory

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ArchiveEntry represents an archived embedding entry.
type ArchiveEntry struct {
	ID          int64
	EmbeddingID int64
	Content     string
	Action      string
	Reason      string
	ArchivedAt  string
}

// ArchiveEmbedding archives an embedding before deletion or modification.
// If the embedding doesn't exist, returns nil (idempotent behavior).
func ArchiveEmbedding(db *sql.DB, embeddingID int64, action string, reason string) error {
	// Query the embedding's content
	var content string

	err := db.QueryRow("SELECT content FROM embeddings WHERE id = ?", embeddingID).Scan(&content)
	if errors.Is(err, sql.ErrNoRows) {
		// Embedding doesn't exist - nothing to archive
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to query embedding content: %w", err)
	}

	// Insert into archive
	timestamp := time.Now().Format(time.RFC3339)

	_, err = db.Exec(`
		INSERT INTO embeddings_archive (embedding_id, content, action, reason, archived_at)
		VALUES (?, ?, ?, ?, ?)
	`, embeddingID, content, action, reason, timestamp)
	if err != nil {
		return fmt.Errorf("failed to insert archive entry: %w", err)
	}

	return nil
}

// ListArchive returns archived embedding entries in reverse chronological order.
func ListArchive(db *sql.DB, limit int) ([]ArchiveEntry, error) {
	query := `
		SELECT id, embedding_id, content, action, reason, archived_at
		FROM embeddings_archive
		ORDER BY id DESC
		LIMIT ?
	`

	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query archive: %w", err)
	}

	defer func() { _ = rows.Close() }()

	var entries []ArchiveEntry

	for rows.Next() {
		var e ArchiveEntry

		err := rows.Scan(&e.ID, &e.EmbeddingID, &e.Content, &e.Action, &e.Reason, &e.ArchivedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan archive entry: %w", err)
		}

		entries = append(entries, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating archive entries: %w", err)
	}

	return entries, nil
}

// PruneArchive removes archive entries older than the specified number of days.
// Returns the number of rows affected.
func PruneArchive(db *sql.DB, maxAgeDays int) (int, error) {
	result, err := db.Exec(`
		DELETE FROM embeddings_archive
		WHERE archived_at < datetime('now', '-' || ? || ' days')
	`, maxAgeDays)
	if err != nil {
		return 0, fmt.Errorf("failed to prune archive: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(affected), nil
}
