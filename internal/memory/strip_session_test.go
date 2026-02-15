// strip_session_test.go
package memory_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/toejough/projctl/internal/memory"
)

func TestStripSession_BasicBlocks(t *testing.T) {
	// Create temp JSONL file with representative blocks
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")

	lines := []string{
		// User text (should keep)
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"fix the bug"}]}}`,
		// System reminder (should strip)
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"<system-reminder>hook output</system-reminder>real message"}]}}`,
		// Assistant text (should keep)
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"I'll fix that."}]}}`,
		// Tool use - Edit with diff (should show diff)
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","name":"Edit","input":{"file_path":"auth.go","old_string":"ExpiresAt string","new_string":"ExpiresAt any"}}]}}`,
		// Tool result success (should omit)
		`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"1","content":"File edited successfully"}]}}`,
		// Tool result error (should keep)
		`{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"2","content":"Error: file not found","is_error":true}]}}`,
		// Thinking block (should skip)
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"thinking","thinking":"let me think..."}]}}`,
		// Progress message (should skip entirely)
		`{"type":"progress","message":"loading"}`,
	}

	content := ""
	for _, l := range lines {
		content += l + "\n"
	}
	os.WriteFile(path, []byte(content), 0644)

	result, err := memory.StripSession(path)
	if err != nil {
		t.Fatalf("StripSession failed: %v", err)
	}

	// Should contain user text
	if !contains(result, "fix the bug") {
		t.Error("missing user text")
	}
	// Should strip system-reminder but keep real message
	if contains(result, "hook output") {
		t.Error("system-reminder not stripped")
	}
	if !contains(result, "real message") {
		t.Error("real message after system-reminder stripped")
	}
	// Should contain assistant text
	if !contains(result, "I'll fix that") {
		t.Error("missing assistant text")
	}
	// Should show Edit diff
	if !contains(result, "ExpiresAt string") || !contains(result, "ExpiresAt any") {
		t.Error("Edit diff not shown")
	}
	// Should omit successful tool result
	if contains(result, "File edited successfully") {
		t.Error("successful tool result not omitted")
	}
	// Should keep error tool result
	if !contains(result, "file not found") {
		t.Error("error tool result omitted")
	}
	// Should not contain thinking
	if contains(result, "let me think") {
		t.Error("thinking block not stripped")
	}
}

func TestStripSession_TeammateMessages(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")

	content := `{"type":"user","message":{"role":"user","content":[{"type":"text","text":"<teammate-message teammate_id=\"worker-a\" summary=\"task done\">Task 1 complete, all tests pass.</teammate-message>"}]}}` + "\n"
	os.WriteFile(path, []byte(content), 0644)

	result, err := memory.StripSession(path)
	if err != nil {
		t.Fatalf("StripSession failed: %v", err)
	}

	// Should keep teammate message content with sender
	if !contains(result, "worker-a") || !contains(result, "Task 1 complete") {
		t.Errorf("teammate message not preserved: %s", result)
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) && containsStr(s, substr))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
