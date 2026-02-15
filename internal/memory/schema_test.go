package memory_test

import (
	"os"
	"testing"

	"github.com/toejough/projctl/internal/memory"
)

func TestProcessedSessionsTableExists(t *testing.T) {
	// Create a temporary directory for the test database
	tmpDir, err := os.MkdirTemp("", "processed-sessions-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Initialize the database
	db, err := memory.InitDBForTest(tmpDir)
	if err != nil {
		t.Fatalf("failed to initialize DB: %v", err)
	}
	defer db.Close()

	// Try to insert into processed_sessions table
	_, err = db.Exec(`
		INSERT INTO processed_sessions (session_id, project, processed_at, items_found, status)
		VALUES (?, ?, ?, ?, ?)
	`, "test-session-123", "test-project", "2026-02-15T10:00:00Z", 5, "success")

	if err != nil {
		t.Fatalf("failed to insert into processed_sessions: %v", err)
	}

	// Verify the insert worked by querying the table
	var sessionID, project, processedAt, status string
	var itemsFound int
	err = db.QueryRow(`
		SELECT session_id, project, processed_at, items_found, status
		FROM processed_sessions
		WHERE session_id = ?
	`, "test-session-123").Scan(&sessionID, &project, &processedAt, &itemsFound, &status)

	if err != nil {
		t.Fatalf("failed to query processed_sessions: %v", err)
	}

	// Verify the values match
	if sessionID != "test-session-123" {
		t.Errorf("expected session_id 'test-session-123', got '%s'", sessionID)
	}
	if project != "test-project" {
		t.Errorf("expected project 'test-project', got '%s'", project)
	}
	if processedAt != "2026-02-15T10:00:00Z" {
		t.Errorf("expected processed_at '2026-02-15T10:00:00Z', got '%s'", processedAt)
	}
	if itemsFound != 5 {
		t.Errorf("expected items_found 5, got %d", itemsFound)
	}
	if status != "success" {
		t.Errorf("expected status 'success', got '%s'", status)
	}
}
