package memory_test

import (
	"os"
	"path/filepath"
	"strings"
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

	var contentSb37 strings.Builder
	for _, l := range lines {
		contentSb37.WriteString(l + "\n")
	}

	content += contentSb37.String()

	os.WriteFile(path, []byte(content), 0644)

	result, endOffset, err := memory.StripSession(path, 0)
	if err != nil {
		t.Fatalf("StripSession failed: %v", err)
	}

	if endOffset == 0 {
		t.Error("endOffset should be > 0")
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

func TestStripSession_OffsetResumption(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")

	// Write 3 initial lines
	initialLines := []string{
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"first message"}]}}`,
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"first response"}]}}`,
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"second message"}]}}`,
	}

	content := ""

	var contentSb94 strings.Builder
	for _, l := range initialLines {
		contentSb94.WriteString(l + "\n")
	}

	content += contentSb94.String()

	os.WriteFile(path, []byte(content), 0644)

	// Strip from 0 — should see all 3 messages
	result1, offset1, err := memory.StripSession(path, 0)
	if err != nil {
		t.Fatalf("first strip failed: %v", err)
	}

	if !contains(result1, "first message") || !contains(result1, "second message") {
		t.Error("first strip should contain all messages")
	}

	// Append 2 more lines
	newLines := []string{
		`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"second response"}]}}`,
		`{"type":"user","message":{"role":"user","content":[{"type":"text","text":"third message"}]}}`,
	}

	f, _ := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	for _, l := range newLines {
		f.WriteString(l + "\n")
	}

	f.Close()

	// Strip from stored offset — should see only new content
	result2, offset2, err := memory.StripSession(path, offset1)
	if err != nil {
		t.Fatalf("second strip failed: %v", err)
	}

	if contains(result2, "first message") || contains(result2, "second message") {
		t.Error("second strip should NOT contain old messages")
	}

	if !contains(result2, "second response") || !contains(result2, "third message") {
		t.Errorf("second strip should contain new messages, got: %s", result2)
	}

	if offset2 <= offset1 {
		t.Errorf("second offset (%d) should be > first offset (%d)", offset2, offset1)
	}

	// Strip from final offset — should produce empty result
	result3, offset3, err := memory.StripSession(path, offset2)
	if err != nil {
		t.Fatalf("third strip failed: %v", err)
	}

	if result3 != "" {
		t.Errorf("third strip should be empty, got: %s", result3)
	}

	if offset3 != offset2 {
		t.Errorf("third offset (%d) should equal second offset (%d)", offset3, offset2)
	}
}

func TestStripSession_TeammateMessages(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")

	content := `{"type":"user","message":{"role":"user","content":[{"type":"text","text":"<teammate-message teammate_id=\"worker-a\" summary=\"task done\">Task 1 complete, all tests pass.</teammate-message>"}]}}` + "\n"
	os.WriteFile(path, []byte(content), 0644)

	result, _, err := memory.StripSession(path, 0)
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
