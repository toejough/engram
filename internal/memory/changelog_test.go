package memory_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/toejough/projctl/internal/memory"
)

func TestWriteChangelogEntry_WritesValidJSONL(t *testing.T) {
	dir := t.TempDir()

	entry := memory.ChangelogEntry{
		Action:          "store",
		SourceTier:      "session",
		DestinationTier: "embeddings",
		ContentID:       "42",
		ContentSummary:  "Use AI-Used trailer, not Co-Authored-By",
		Reason:          "new correction extracted",
		SessionID:       "sess-abc-123",
		Metadata:        map[string]string{"project": "projctl"},
	}

	err := memory.WriteChangelogEntry(dir, entry)
	if err != nil {
		t.Fatalf("WriteChangelogEntry failed: %v", err)
	}

	// Read the file and verify valid JSONL
	data, err := os.ReadFile(filepath.Join(dir, "changelog.jsonl"))
	if err != nil {
		t.Fatalf("failed to read changelog.jsonl: %v", err)
	}

	var decoded memory.ChangelogEntry
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("invalid JSON in changelog.jsonl: %v", err)
	}

	if decoded.Action != "store" {
		t.Errorf("expected action 'store', got %q", decoded.Action)
	}
	if decoded.SourceTier != "session" {
		t.Errorf("expected source_tier 'session', got %q", decoded.SourceTier)
	}
	if decoded.DestinationTier != "embeddings" {
		t.Errorf("expected destination_tier 'embeddings', got %q", decoded.DestinationTier)
	}
	if decoded.ContentID != "42" {
		t.Errorf("expected content_id '42', got %q", decoded.ContentID)
	}
	if decoded.ContentSummary != "Use AI-Used trailer, not Co-Authored-By" {
		t.Errorf("expected content_summary preserved, got %q", decoded.ContentSummary)
	}
	if decoded.Reason != "new correction extracted" {
		t.Errorf("expected reason preserved, got %q", decoded.Reason)
	}
	if decoded.SessionID != "sess-abc-123" {
		t.Errorf("expected session_id 'sess-abc-123', got %q", decoded.SessionID)
	}
	if decoded.Metadata["project"] != "projctl" {
		t.Errorf("expected metadata project 'projctl', got %q", decoded.Metadata["project"])
	}
	if decoded.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}

func TestWriteChangelogEntry_TruncatesContentSummary(t *testing.T) {
	dir := t.TempDir()

	longContent := ""
	for i := 0; i < 20; i++ {
		longContent += "0123456789"
	}
	// longContent is 200 chars

	entry := memory.ChangelogEntry{
		Action:         "promote",
		ContentSummary: longContent,
	}

	err := memory.WriteChangelogEntry(dir, entry)
	if err != nil {
		t.Fatalf("WriteChangelogEntry failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "changelog.jsonl"))
	if err != nil {
		t.Fatalf("failed to read changelog.jsonl: %v", err)
	}

	var decoded memory.ChangelogEntry
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if len(decoded.ContentSummary) > 100 {
		t.Errorf("expected content_summary truncated to 100 chars, got %d", len(decoded.ContentSummary))
	}
}

func TestWriteChangelogEntry_AppendOnly(t *testing.T) {
	dir := t.TempDir()

	entries := []memory.ChangelogEntry{
		{Action: "store", ContentID: "1", ContentSummary: "first"},
		{Action: "promote", ContentID: "2", ContentSummary: "second"},
		{Action: "prune", ContentID: "3", ContentSummary: "third"},
	}

	for _, entry := range entries {
		if err := memory.WriteChangelogEntry(dir, entry); err != nil {
			t.Fatalf("WriteChangelogEntry failed: %v", err)
		}
	}

	// Read and parse each line
	data, err := os.ReadFile(filepath.Join(dir, "changelog.jsonl"))
	if err != nil {
		t.Fatalf("failed to read changelog.jsonl: %v", err)
	}

	lines := splitJSONLines(data)
	if len(lines) != 3 {
		t.Fatalf("expected 3 JSONL lines, got %d", len(lines))
	}

	for i, line := range lines {
		var decoded memory.ChangelogEntry
		if err := json.Unmarshal(line, &decoded); err != nil {
			t.Fatalf("line %d: invalid JSON: %v", i, err)
		}
		if decoded.Action != entries[i].Action {
			t.Errorf("line %d: expected action %q, got %q", i, entries[i].Action, decoded.Action)
		}
	}
}

func TestWriteChangelogEntry_CreatesDirIfMissing(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nonexistent", "subdir")

	entry := memory.ChangelogEntry{
		Action:         "decay",
		ContentSummary: "test",
	}

	err := memory.WriteChangelogEntry(dir, entry)
	if err != nil {
		t.Fatalf("WriteChangelogEntry failed with missing dir: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(filepath.Join(dir, "changelog.jsonl")); err != nil {
		t.Fatalf("changelog.jsonl not created: %v", err)
	}
}

func TestReadChangelogEntries_All(t *testing.T) {
	dir := t.TempDir()

	// Write several entries
	actions := []string{"store", "promote", "prune", "decay"}
	for _, action := range actions {
		entry := memory.ChangelogEntry{
			Action:         action,
			ContentSummary: "test " + action,
		}
		if err := memory.WriteChangelogEntry(dir, entry); err != nil {
			t.Fatalf("WriteChangelogEntry failed: %v", err)
		}
	}

	// Read all entries
	entries, err := memory.ReadChangelogEntries(dir, memory.ChangelogFilter{})
	if err != nil {
		t.Fatalf("ReadChangelogEntries failed: %v", err)
	}

	if len(entries) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(entries))
	}
}

func TestReadChangelogEntries_FilterByAction(t *testing.T) {
	dir := t.TempDir()

	actions := []string{"store", "promote", "store", "prune"}
	for _, action := range actions {
		entry := memory.ChangelogEntry{
			Action:         action,
			ContentSummary: "test " + action,
		}
		if err := memory.WriteChangelogEntry(dir, entry); err != nil {
			t.Fatalf("WriteChangelogEntry failed: %v", err)
		}
	}

	// Filter by action
	entries, err := memory.ReadChangelogEntries(dir, memory.ChangelogFilter{
		Action: "store",
	})
	if err != nil {
		t.Fatalf("ReadChangelogEntries failed: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 'store' entries, got %d", len(entries))
	}
	for _, e := range entries {
		if e.Action != "store" {
			t.Errorf("expected action 'store', got %q", e.Action)
		}
	}
}

func TestReadChangelogEntries_FilterByTier(t *testing.T) {
	dir := t.TempDir()

	tiers := []struct {
		source string
		dest   string
	}{
		{"embeddings", "skills"},
		{"skills", "claude-md"},
		{"embeddings", "skills"},
	}
	for _, tier := range tiers {
		entry := memory.ChangelogEntry{
			Action:          "promote",
			SourceTier:      tier.source,
			DestinationTier: tier.dest,
			ContentSummary:  "test",
		}
		if err := memory.WriteChangelogEntry(dir, entry); err != nil {
			t.Fatalf("WriteChangelogEntry failed: %v", err)
		}
	}

	// Filter by source tier
	entries, err := memory.ReadChangelogEntries(dir, memory.ChangelogFilter{
		SourceTier: "embeddings",
	})
	if err != nil {
		t.Fatalf("ReadChangelogEntries failed: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries with source_tier 'embeddings', got %d", len(entries))
	}

	// Filter by destination tier
	entries, err = memory.ReadChangelogEntries(dir, memory.ChangelogFilter{
		DestinationTier: "claude-md",
	})
	if err != nil {
		t.Fatalf("ReadChangelogEntries failed: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry with destination_tier 'claude-md', got %d", len(entries))
	}
}

func TestReadChangelogEntries_FilterBySince(t *testing.T) {
	dir := t.TempDir()

	// Write an entry
	entry := memory.ChangelogEntry{
		Action:         "store",
		ContentSummary: "recent entry",
	}
	if err := memory.WriteChangelogEntry(dir, entry); err != nil {
		t.Fatalf("WriteChangelogEntry failed: %v", err)
	}

	// Filter with a time in the past (should include the entry)
	pastTime := time.Now().Add(-1 * time.Hour)
	entries, err := memory.ReadChangelogEntries(dir, memory.ChangelogFilter{
		Since: pastTime,
	})
	if err != nil {
		t.Fatalf("ReadChangelogEntries failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry since 1h ago, got %d", len(entries))
	}

	// Filter with a time in the future (should exclude the entry)
	futureTime := time.Now().Add(1 * time.Hour)
	entries, err = memory.ReadChangelogEntries(dir, memory.ChangelogFilter{
		Since: futureTime,
	})
	if err != nil {
		t.Fatalf("ReadChangelogEntries failed: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries since future time, got %d", len(entries))
	}
}

func TestReadChangelogEntries_EmptyFile(t *testing.T) {
	dir := t.TempDir()

	// Read from non-existent file
	entries, err := memory.ReadChangelogEntries(dir, memory.ChangelogFilter{})
	if err != nil {
		t.Fatalf("ReadChangelogEntries should not error on missing file: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries from missing file, got %d", len(entries))
	}
}

// splitJSONLines splits JSONL data into individual JSON objects.
func splitJSONLines(data []byte) [][]byte {
	var lines [][]byte
	for _, line := range splitLines(data) {
		trimmed := trimBytes(line)
		if len(trimmed) > 0 {
			lines = append(lines, trimmed)
		}
	}
	return lines
}

func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			lines = append(lines, data[start:i])
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}

func trimBytes(b []byte) []byte {
	start := 0
	for start < len(b) && (b[start] == ' ' || b[start] == '\t' || b[start] == '\r') {
		start++
	}
	end := len(b)
	for end > start && (b[end-1] == ' ' || b[end-1] == '\t' || b[end-1] == '\r') {
		end--
	}
	return b[start:end]
}
