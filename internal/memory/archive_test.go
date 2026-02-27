package memory

import (
	"testing"
)

// TestArchiveEmbedding_NotFound tests archiving a non-existent embedding (should succeed silently).
func TestArchiveEmbedding_NotFound(t *testing.T) {
	db, err := InitDBForTest(t.TempDir())
	if err != nil {
		t.Fatalf("InitDBForTest failed: %v", err)
	}
	defer db.Close()

	// Try to archive non-existent embedding
	err = ArchiveEmbedding(db, 999999, "prune", "test")
	if err != nil {
		t.Errorf("ArchiveEmbedding should succeed silently for non-existent ID, got error: %v", err)
	}

	// Verify no archive entry was created
	var count int

	err = db.QueryRow("SELECT COUNT(*) FROM embeddings_archive WHERE embedding_id = 999999").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query archive: %v", err)
	}

	if count != 0 {
		t.Errorf("expected 0 archive entries for non-existent ID, got %d", count)
	}
}

// TestArchiveEmbedding_Success tests archiving an existing embedding.
func TestArchiveEmbedding_Success(t *testing.T) {
	db, err := InitDBForTest(t.TempDir())
	if err != nil {
		t.Fatalf("InitDBForTest failed: %v", err)
	}
	defer db.Close()

	// Insert test embedding
	result, err := db.Exec("INSERT INTO embeddings (content, source) VALUES (?, ?)", "test content", "internal")
	if err != nil {
		t.Fatalf("failed to insert test embedding: %v", err)
	}

	embeddingID, _ := result.LastInsertId()

	// Archive it
	err = ArchiveEmbedding(db, embeddingID, "prune", "low confidence")
	if err != nil {
		t.Fatalf("ArchiveEmbedding failed: %v", err)
	}

	// Verify archive entry exists
	var count int

	err = db.QueryRow("SELECT COUNT(*) FROM embeddings_archive WHERE embedding_id = ?", embeddingID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to query archive: %v", err)
	}

	if count != 1 {
		t.Errorf("expected 1 archive entry, got %d", count)
	}

	// Verify content was saved
	var content string

	err = db.QueryRow("SELECT content FROM embeddings_archive WHERE embedding_id = ?", embeddingID).Scan(&content)
	if err != nil {
		t.Fatalf("failed to query archive content: %v", err)
	}

	if content != "test content" {
		t.Errorf("expected content 'test content', got %q", content)
	}
}

// TestListArchive_Empty tests listing an empty archive.
func TestListArchive_Empty(t *testing.T) {
	db, err := InitDBForTest(t.TempDir())
	if err != nil {
		t.Fatalf("InitDBForTest failed: %v", err)
	}
	defer db.Close()

	entries, err := ListArchive(db, 50)
	if err != nil {
		t.Fatalf("ListArchive failed: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("expected empty archive, got %d entries", len(entries))
	}
}

// TestListArchive_Limit tests that the limit parameter is respected.
func TestListArchive_Limit(t *testing.T) {
	db, err := InitDBForTest(t.TempDir())
	if err != nil {
		t.Fatalf("InitDBForTest failed: %v", err)
	}
	defer db.Close()

	// Insert and archive 5 embeddings
	for i := 1; i <= 5; i++ {
		_, _ = db.Exec("INSERT INTO embeddings (id, content, source) VALUES (?, ?, ?)", i, "content", "internal")
		_ = ArchiveEmbedding(db, int64(i), "prune", "test")
	}

	// List with limit of 2
	entries, err := ListArchive(db, 2)
	if err != nil {
		t.Fatalf("ListArchive failed: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("expected 2 archive entries (limit), got %d", len(entries))
	}
}

// TestListArchive_WithEntries tests listing archive entries in reverse chronological order.
func TestListArchive_WithEntries(t *testing.T) {
	db, err := InitDBForTest(t.TempDir())
	if err != nil {
		t.Fatalf("InitDBForTest failed: %v", err)
	}
	defer db.Close()

	// Insert test embeddings
	_, _ = db.Exec("INSERT INTO embeddings (id, content, source) VALUES (1, 'first', 'internal')")
	_, _ = db.Exec("INSERT INTO embeddings (id, content, source) VALUES (2, 'second', 'internal')")
	_, _ = db.Exec("INSERT INTO embeddings (id, content, source) VALUES (3, 'third', 'internal')")

	// Archive them in order
	_ = ArchiveEmbedding(db, 1, "prune", "reason1")
	_ = ArchiveEmbedding(db, 2, "prune", "reason2")
	_ = ArchiveEmbedding(db, 3, "prune", "reason3")

	// List archive
	entries, err := ListArchive(db, 50)
	if err != nil {
		t.Fatalf("ListArchive failed: %v", err)
	}

	if len(entries) != 3 {
		t.Fatalf("expected 3 archive entries, got %d", len(entries))
	}

	// Verify reverse chronological order (most recent first)
	if entries[0].EmbeddingID != 3 {
		t.Errorf("expected first entry to be embedding 3, got %d", entries[0].EmbeddingID)
	}

	if entries[1].EmbeddingID != 2 {
		t.Errorf("expected second entry to be embedding 2, got %d", entries[1].EmbeddingID)
	}

	if entries[2].EmbeddingID != 1 {
		t.Errorf("expected third entry to be embedding 1, got %d", entries[2].EmbeddingID)
	}
}

// TestPruneArchive tests removing old archive entries.
func TestPruneArchive(t *testing.T) {
	db, err := InitDBForTest(t.TempDir())
	if err != nil {
		t.Fatalf("InitDBForTest failed: %v", err)
	}
	defer db.Close()

	// Insert archived entries with different timestamps
	// Old entry (100 days ago)
	_, _ = db.Exec(`INSERT INTO embeddings_archive (embedding_id, content, action, reason, archived_at)
		VALUES (1, 'old content', 'prune', 'test', datetime('now', '-100 days'))`)

	// Recent entry (10 days ago)
	_, _ = db.Exec(`INSERT INTO embeddings_archive (embedding_id, content, action, reason, archived_at)
		VALUES (2, 'recent content', 'prune', 'test', datetime('now', '-10 days'))`)

	// Prune entries older than 30 days
	affected, err := PruneArchive(db, 30)
	if err != nil {
		t.Fatalf("PruneArchive failed: %v", err)
	}

	if affected != 1 {
		t.Errorf("expected 1 row affected, got %d", affected)
	}

	// Verify only recent entry remains
	var count int

	err = db.QueryRow("SELECT COUNT(*) FROM embeddings_archive").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query archive: %v", err)
	}

	if count != 1 {
		t.Errorf("expected 1 archive entry remaining, got %d", count)
	}

	// Verify it's the recent one
	var embeddingID int64

	err = db.QueryRow("SELECT embedding_id FROM embeddings_archive").Scan(&embeddingID)
	if err != nil {
		t.Fatalf("failed to query remaining entry: %v", err)
	}

	if embeddingID != 2 {
		t.Errorf("expected embedding_id 2 to remain, got %d", embeddingID)
	}
}
