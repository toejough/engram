package memory_test

import (
	"os"
	"testing"
	"time"

	"github.com/toejough/projctl/internal/memory"
)

func TestRecordProcessedSession(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "record-session-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	db, err := memory.InitDBForTest(tmpDir)
	if err != nil {
		t.Fatalf("failed to initialize DB: %v", err)
	}
	defer db.Close()

	// Record a session
	err = memory.RecordProcessedSession(db, "session-123", "my-project", 5, "success")
	if err != nil {
		t.Fatalf("RecordProcessedSession failed: %v", err)
	}

	// Verify it was recorded
	var sessionID, project, status string
	var itemsFound int
	err = db.QueryRow(`
		SELECT session_id, project, items_found, status
		FROM processed_sessions
		WHERE session_id = ?
	`, "session-123").Scan(&sessionID, &project, &itemsFound, &status)

	if err != nil {
		t.Fatalf("failed to query session: %v", err)
	}

	if sessionID != "session-123" {
		t.Errorf("expected session_id 'session-123', got '%s'", sessionID)
	}
	if project != "my-project" {
		t.Errorf("expected project 'my-project', got '%s'", project)
	}
	if itemsFound != 5 {
		t.Errorf("expected items_found 5, got %d", itemsFound)
	}
	if status != "success" {
		t.Errorf("expected status 'success', got '%s'", status)
	}

	// Test INSERT OR REPLACE behavior - update existing session
	err = memory.RecordProcessedSession(db, "session-123", "my-project-updated", 10, "error")
	if err != nil {
		t.Fatalf("RecordProcessedSession failed on replace: %v", err)
	}

	// Verify it was replaced
	err = db.QueryRow(`
		SELECT project, items_found, status
		FROM processed_sessions
		WHERE session_id = ?
	`, "session-123").Scan(&project, &itemsFound, &status)

	if err != nil {
		t.Fatalf("failed to query updated session: %v", err)
	}

	if project != "my-project-updated" {
		t.Errorf("expected project 'my-project-updated', got '%s'", project)
	}
	if itemsFound != 10 {
		t.Errorf("expected items_found 10, got %d", itemsFound)
	}
	if status != "error" {
		t.Errorf("expected status 'error', got '%s'", status)
	}
}

func TestIsSessionProcessed(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "is-session-processed-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	db, err := memory.InitDBForTest(tmpDir)
	if err != nil {
		t.Fatalf("failed to initialize DB: %v", err)
	}
	defer db.Close()

	// Check unknown session - should return false
	processed, err := memory.IsSessionProcessed(db, "unknown-session")
	if err != nil {
		t.Fatalf("IsSessionProcessed failed: %v", err)
	}
	if processed {
		t.Error("expected unknown session to return false")
	}

	// Record a session
	err = memory.RecordProcessedSession(db, "session-456", "test-project", 3, "success")
	if err != nil {
		t.Fatalf("RecordProcessedSession failed: %v", err)
	}

	// Check known session - should return true
	processed, err = memory.IsSessionProcessed(db, "session-456")
	if err != nil {
		t.Fatalf("IsSessionProcessed failed: %v", err)
	}
	if !processed {
		t.Error("expected known session to return true")
	}
}

func TestResetLastNSessions(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "reset-sessions-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	db, err := memory.InitDBForTest(tmpDir)
	if err != nil {
		t.Fatalf("failed to initialize DB: %v", err)
	}
	defer db.Close()

	// Record 3 sessions with different timestamps
	now := time.Now()
	sessions := []struct {
		id        string
		timestamp time.Time
	}{
		{"session-1", now.Add(-2 * time.Hour)},
		{"session-2", now.Add(-1 * time.Hour)},
		{"session-3", now},
	}

	for _, s := range sessions {
		_, err := db.Exec(`
			INSERT INTO processed_sessions (session_id, project, processed_at, items_found, status)
			VALUES (?, ?, ?, ?, ?)
		`, s.id, "test-project", s.timestamp.Format(time.RFC3339), 1, "success")
		if err != nil {
			t.Fatalf("failed to insert session %s: %v", s.id, err)
		}
	}

	// Reset the 2 most recent sessions
	deleted, err := memory.ResetLastNSessions(db, 2)
	if err != nil {
		t.Fatalf("ResetLastNSessions failed: %v", err)
	}
	if deleted != 2 {
		t.Errorf("expected 2 sessions deleted, got %d", deleted)
	}

	// Verify session-1 still exists
	processed, err := memory.IsSessionProcessed(db, "session-1")
	if err != nil {
		t.Fatalf("IsSessionProcessed failed: %v", err)
	}
	if !processed {
		t.Error("expected session-1 to still exist")
	}

	// Verify session-2 and session-3 were deleted
	processed, err = memory.IsSessionProcessed(db, "session-2")
	if err != nil {
		t.Fatalf("IsSessionProcessed failed: %v", err)
	}
	if processed {
		t.Error("expected session-2 to be deleted")
	}

	processed, err = memory.IsSessionProcessed(db, "session-3")
	if err != nil {
		t.Fatalf("IsSessionProcessed failed: %v", err)
	}
	if processed {
		t.Error("expected session-3 to be deleted")
	}
}
