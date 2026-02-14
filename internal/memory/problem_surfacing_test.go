package memory

import (
	"database/sql"
	"testing"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	db, err := InitDBForTest(dir)
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// TestDetectRecurringProblems_HookFailures verifies detection of hooks with high failure rates.
func TestDetectRecurringProblems_HookFailures(t *testing.T) {
	db := setupTestDB(t)

	// Insert hook events: 10 events, 5 failures = 50% failure rate
	for i := 0; i < 10; i++ {
		exitCode := 0
		if i < 5 {
			exitCode = 2
		}
		_, err := db.Exec(`INSERT INTO hook_events (hook_name, fired_at, exit_code, duration_ms) VALUES (?, datetime('now'), ?, ?)`,
			"check-claudemd", exitCode, 50)
		if err != nil {
			t.Fatalf("failed to insert hook event: %v", err)
		}
	}

	// Detect problems with default threshold (0.3)
	opts := RecurringProblemOpts{
		HookFailureRate: 0.3,
		HookWindowDays:  7,
	}
	problems, err := DetectRecurringProblems(db, opts)
	if err != nil {
		t.Fatalf("DetectRecurringProblems failed: %v", err)
	}

	// Should detect 1 problem (50% failure rate > 30% threshold)
	if len(problems) != 1 {
		t.Errorf("expected 1 problem, got %d", len(problems))
	}

	if len(problems) > 0 {
		p := problems[0]
		if p.Source != "hook" {
			t.Errorf("expected Source='hook', got '%s'", p.Source)
		}
		if p.Name != "check-claudemd" {
			t.Errorf("expected Name='check-claudemd', got '%s'", p.Name)
		}
		if p.Count != 10 {
			t.Errorf("expected Count=10, got %d", p.Count)
		}
		if p.Rate < 0.49 || p.Rate > 0.51 {
			t.Errorf("expected Rate≈0.5, got %f", p.Rate)
		}
	}
}

// TestDetectRecurringProblems_FeedbackClusters verifies detection of feedback with multiple "wrong" responses.
func TestDetectRecurringProblems_FeedbackClusters(t *testing.T) {
	db := setupTestDB(t)

	// Insert test embedding
	_, err := db.Exec(`INSERT INTO embeddings (id, content, source) VALUES (?, ?, ?)`,
		1, "test content", "test")
	if err != nil {
		t.Fatalf("failed to insert embedding: %v", err)
	}

	// Insert 3 wrong feedback records
	for i := 0; i < 3; i++ {
		_, err := db.Exec(`INSERT INTO feedback (embedding_id, feedback_type, created_at) VALUES (?, 'wrong', datetime('now'))`, 1)
		if err != nil {
			t.Fatalf("failed to insert feedback: %v", err)
		}
	}

	// Detect problems with default threshold (2)
	opts := RecurringProblemOpts{
		FeedbackMinWrong: 2,
	}
	problems, err := DetectRecurringProblems(db, opts)
	if err != nil {
		t.Fatalf("DetectRecurringProblems failed: %v", err)
	}

	// Should detect 1 problem (3 wrong >= 2 threshold)
	if len(problems) != 1 {
		t.Errorf("expected 1 problem, got %d", len(problems))
	}

	if len(problems) > 0 {
		p := problems[0]
		if p.Source != "feedback" {
			t.Errorf("expected Source='feedback', got '%s'", p.Source)
		}
		if p.Count != 3 {
			t.Errorf("expected Count=3, got %d", p.Count)
		}
		if p.Rate != 0 {
			t.Errorf("expected Rate=0 for feedback, got %f", p.Rate)
		}
	}
}

// TestDetectRecurringProblems_NoProblems verifies that clean data returns empty results.
func TestDetectRecurringProblems_NoProblems(t *testing.T) {
	db := setupTestDB(t)

	// Insert hook events with low failure rate (1/10 = 10%)
	for i := 0; i < 10; i++ {
		exitCode := 0
		if i == 0 {
			exitCode = 1
		}
		_, err := db.Exec(`INSERT INTO hook_events (hook_name, fired_at, exit_code, duration_ms) VALUES (?, datetime('now'), ?, ?)`,
			"good-hook", exitCode, 50)
		if err != nil {
			t.Fatalf("failed to insert hook event: %v", err)
		}
	}

	opts := RecurringProblemOpts{
		HookFailureRate:  0.3, // 10% < 30% threshold
		FeedbackMinWrong: 2,
	}
	problems, err := DetectRecurringProblems(db, opts)
	if err != nil {
		t.Fatalf("DetectRecurringProblems failed: %v", err)
	}

	// Should detect no problems
	if len(problems) != 0 {
		t.Errorf("expected 0 problems, got %d", len(problems))
	}
}

// TestDetectRecurringProblems_Defaults verifies that zero-value opts uses sensible defaults.
func TestDetectRecurringProblems_Defaults(t *testing.T) {
	db := setupTestDB(t)

	// Insert hook events with 40% failure rate
	for i := 0; i < 10; i++ {
		exitCode := 0
		if i < 4 {
			exitCode = 1
		}
		_, err := db.Exec(`INSERT INTO hook_events (hook_name, fired_at, exit_code, duration_ms) VALUES (?, datetime('now'), ?, ?)`,
			"failing-hook", exitCode, 50)
		if err != nil {
			t.Fatalf("failed to insert hook event: %v", err)
		}
	}

	// Use zero-value opts (should apply defaults)
	problems, err := DetectRecurringProblems(db, RecurringProblemOpts{})
	if err != nil {
		t.Fatalf("DetectRecurringProblems failed: %v", err)
	}

	// Should detect problem with default 0.3 threshold (40% > 30%)
	if len(problems) != 1 {
		t.Errorf("expected 1 problem with defaults, got %d", len(problems))
	}
}

// TestProblemsToProposals verifies conversion to MaintenanceProposal format.
func TestProblemsToProposals(t *testing.T) {
	problems := []RecurringProblem{
		{
			Source:      "hook",
			Name:        "check-claudemd",
			Count:       10,
			Rate:        0.5,
			Description: "Hook check-claudemd has 50.0% failure rate (5 failures in last 7 days)",
		},
		{
			Source:      "feedback",
			Name:        "test content",
			Count:       3,
			Rate:        0,
			Description: "Content has 3 'wrong' feedback responses",
		},
	}

	proposals := ProblemsToProposals(problems)

	// Should create 2 proposals
	if len(proposals) != 2 {
		t.Fatalf("expected 2 proposals, got %d", len(proposals))
	}

	// Check hook proposal
	hookProposal := proposals[0]
	if hookProposal.Tier != "meta" {
		t.Errorf("expected Tier='meta', got '%s'", hookProposal.Tier)
	}
	if hookProposal.Action != "surface" {
		t.Errorf("expected Action='surface', got '%s'", hookProposal.Action)
	}
	if hookProposal.Target != "check-claudemd" {
		t.Errorf("expected Target='check-claudemd', got '%s'", hookProposal.Target)
	}
	if hookProposal.Reason == "" {
		t.Error("expected non-empty Reason")
	}
	if hookProposal.Preview == "" {
		t.Error("expected non-empty Preview")
	}

	// Check feedback proposal
	feedbackProposal := proposals[1]
	if feedbackProposal.Tier != "meta" {
		t.Errorf("expected Tier='meta', got '%s'", feedbackProposal.Tier)
	}
	if feedbackProposal.Action != "surface" {
		t.Errorf("expected Action='surface', got '%s'", feedbackProposal.Action)
	}
}
