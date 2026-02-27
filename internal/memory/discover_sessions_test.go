package memory_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/toejough/projctl/internal/memory"
)

func TestDiscoverSessions(t *testing.T) {
	// Create temp directory structure mimicking ~/.claude/projects/
	tmpDir := t.TempDir()

	// Create project directories with encoded paths
	proj1 := filepath.Join(tmpDir, "-Users-joe-repos-personal-projctl")
	proj2 := filepath.Join(tmpDir, "-Users-joe-repos-work-myapp")
	subagentDir := filepath.Join(proj1, "subagents")

	if err := os.MkdirAll(proj1, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(proj2, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(subagentDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create session files with controlled timestamps
	now := time.Now()
	sessions := []struct {
		path string
		age  time.Duration
		size int64
	}{
		// Valid sessions in proj1
		{filepath.Join(proj1, "session1.jsonl"), 0, 1024},
		{filepath.Join(proj1, "session2.jsonl"), 24 * time.Hour, 2048},
		{filepath.Join(proj1, "session3.jsonl"), 48 * time.Hour, 512},
		// Session in subagents dir (should be skipped)
		{filepath.Join(subagentDir, "subagent.jsonl"), 0, 1024},
		// Valid session in proj2
		{filepath.Join(proj2, "session4.jsonl"), 12 * time.Hour, 4096},
		// Non-jsonl file (should be skipped)
		{filepath.Join(proj1, "readme.txt"), 0, 100},
		// Small file (may be filtered by MinSize)
		{filepath.Join(proj1, "tiny.jsonl"), 0, 10},
	}

	for _, s := range sessions {
		// Write file with specified size
		content := make([]byte, s.size)
		for i := range content {
			content[i] = 'x'
		}

		if err := os.WriteFile(s.path, content, 0644); err != nil {
			t.Fatal(err)
		}

		modTime := now.Add(-s.age)
		if err := os.Chtimes(s.path, modTime, modTime); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("finds all jsonl files excluding subagents", func(t *testing.T) {
		opts := memory.DiscoverOpts{
			ProjectsDir: tmpDir,
		}

		results, err := memory.DiscoverSessions(opts)
		if err != nil {
			t.Fatalf("DiscoverSessions failed: %v", err)
		}

		// Should find 5 .jsonl files (3 in proj1, 1 in proj2, 1 tiny) - excluding subagent.jsonl
		if len(results) != 5 {
			t.Errorf("expected 5 sessions, got %d", len(results))
		}

		// Verify subagent session was skipped
		for _, r := range results {
			if filepath.Base(r.Path) == "subagent.jsonl" {
				t.Errorf("subagent.jsonl should have been skipped")
			}
		}
	})

	t.Run("filters by MinSize", func(t *testing.T) {
		opts := memory.DiscoverOpts{
			ProjectsDir: tmpDir,
			MinSize:     500, // Should exclude tiny.jsonl (10 bytes)
		}

		results, err := memory.DiscoverSessions(opts)
		if err != nil {
			t.Fatalf("DiscoverSessions failed: %v", err)
		}

		// Should find 4 .jsonl files (session1, session2, session3, session4)
		if len(results) != 4 {
			t.Errorf("expected 4 sessions with MinSize=500, got %d", len(results))
		}

		// Verify tiny.jsonl was filtered out
		for _, r := range results {
			if filepath.Base(r.Path) == "tiny.jsonl" {
				t.Errorf("tiny.jsonl should have been filtered by MinSize")
			}
		}
	})

	t.Run("filters by Days", func(t *testing.T) {
		opts := memory.DiscoverOpts{
			ProjectsDir: tmpDir,
			Days:        1, // Only sessions modified in last 24 hours
		}

		results, err := memory.DiscoverSessions(opts)
		if err != nil {
			t.Fatalf("DiscoverSessions failed: %v", err)
		}

		// Should find sessions modified in last 24 hours (session1, session4, tiny, subagent excluded)
		// session1 (0h), session4 (12h), tiny (0h) = 3 sessions
		if len(results) != 3 {
			t.Errorf("expected 3 sessions within 1 day, got %d", len(results))
		}
	})

	t.Run("filters by Last N sessions", func(t *testing.T) {
		opts := memory.DiscoverOpts{
			ProjectsDir: tmpDir,
			Last:        2,
		}

		results, err := memory.DiscoverSessions(opts)
		if err != nil {
			t.Fatalf("DiscoverSessions failed: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("expected 2 most recent sessions, got %d", len(results))
		}
	})

	t.Run("sorts by most-recent-first", func(t *testing.T) {
		opts := memory.DiscoverOpts{
			ProjectsDir: tmpDir,
			MinSize:     500, // Exclude tiny.jsonl for clarity
		}

		results, err := memory.DiscoverSessions(opts)
		if err != nil {
			t.Fatalf("DiscoverSessions failed: %v", err)
		}

		// Verify results are sorted by ModTime descending
		for i := 1; i < len(results); i++ {
			if results[i].ModTime.After(results[i-1].ModTime) {
				t.Errorf("results not sorted by most-recent-first: [%d] %v is newer than [%d] %v",
					i, results[i].ModTime, i-1, results[i-1].ModTime)
			}
		}
	})

	t.Run("derives project name correctly", func(t *testing.T) {
		opts := memory.DiscoverOpts{
			ProjectsDir: tmpDir,
			MinSize:     500,
		}

		results, err := memory.DiscoverSessions(opts)
		if err != nil {
			t.Fatalf("DiscoverSessions failed: %v", err)
		}

		// Check that project names are correctly derived
		projectNames := make(map[string]bool)
		for _, r := range results {
			projectNames[r.Project] = true
		}

		if !projectNames["projctl"] {
			t.Errorf("expected project 'projctl' to be derived from -Users-joe-repos-personal-projctl")
		}

		if !projectNames["myapp"] {
			t.Errorf("expected project 'myapp' to be derived from -Users-joe-repos-work-myapp")
		}
	})
}

func TestReverseClaudeProjectDir(t *testing.T) {
	tests := []struct {
		name     string
		dirName  string
		expected string
	}{
		{
			name:     "simple path",
			dirName:  "-Users-joe-repos-personal-projctl",
			expected: "/Users/joe/repos/personal/projctl",
		},
		{
			name:     "root path",
			dirName:  "-home-user-project",
			expected: "/home/user/project",
		},
		{
			name:     "single segment",
			dirName:  "-project",
			expected: "/project",
		},
		{
			name:     "empty string",
			dirName:  "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := memory.ReverseClaudeProjectDir(tt.dirName)
			if result != tt.expected {
				t.Errorf("ReverseClaudeProjectDir(%q) = %q, want %q", tt.dirName, result, tt.expected)
			}
		})
	}
}
