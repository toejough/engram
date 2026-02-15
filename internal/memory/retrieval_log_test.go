package memory_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/memory"
)

// ============================================================================
// Task 2: Retrieval log infrastructure tests
// ============================================================================

// TestLogRetrievalWritesValidJSONL verifies that LogRetrieval appends a valid
// JSON line to retrievals.jsonl.
func TestLogRetrievalWritesValidJSONL(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()

	entry := memory.RetrievalLogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Hook:      "SessionStart",
		Query:     "recent important learnings",
		Results: []memory.RetrievalResult{
			{ID: 1, Content: "use AI-Used trailer", Score: 0.92, Tier: "embedding"},
			{ID: 2, Content: "always run tests", Score: 0.85, Tier: "embedding"},
		},
		FilteredCount: 3,
		SessionID:     "sess-abc-123",
	}

	err := memory.LogRetrieval(dir, entry)
	g.Expect(err).NotTo(HaveOccurred())

	// Read the file and verify it's valid JSONL
	logPath := filepath.Join(dir, "retrievals.jsonl")
	data, err := os.ReadFile(logPath)
	g.Expect(err).NotTo(HaveOccurred())

	var decoded memory.RetrievalLogEntry
	err = json.Unmarshal(data, &decoded)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(decoded.Hook).To(Equal("SessionStart"))
	g.Expect(decoded.Query).To(Equal("recent important learnings"))
	g.Expect(decoded.Results).To(HaveLen(2))
	g.Expect(decoded.Results[0].Content).To(Equal("use AI-Used trailer"))
	g.Expect(decoded.FilteredCount).To(Equal(3))
	g.Expect(decoded.SessionID).To(Equal("sess-abc-123"))
}

// TestLogRetrievalAppendsMultipleEntries verifies that multiple calls append
// correctly, producing one JSON object per line.
func TestLogRetrievalAppendsMultipleEntries(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()

	for i := 0; i < 3; i++ {
		entry := memory.RetrievalLogEntry{
			Timestamp: time.Now().Format(time.RFC3339),
			Hook:      "UserPromptSubmit",
			Query:     "query " + string(rune('A'+i)),
			Results: []memory.RetrievalResult{
				{ID: int64(i + 1), Content: "result", Score: 0.9, Tier: "embedding"},
			},
			SessionID: "sess-multi",
		}
		err := memory.LogRetrieval(dir, entry)
		g.Expect(err).NotTo(HaveOccurred())
	}

	logPath := filepath.Join(dir, "retrievals.jsonl")
	data, err := os.ReadFile(logPath)
	g.Expect(err).NotTo(HaveOccurred())

	// Split by newline, filter empty
	lines := splitNonEmpty(string(data))
	g.Expect(lines).To(HaveLen(3))

	// Each line must be valid JSON
	for _, line := range lines {
		var entry memory.RetrievalLogEntry
		err := json.Unmarshal([]byte(line), &entry)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(entry.Hook).To(Equal("UserPromptSubmit"))
	}
}

// TestLogRetrievalWithMetadata verifies that optional metadata is included.
func TestLogRetrievalWithMetadata(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()

	entry := memory.RetrievalLogEntry{
		Timestamp:     time.Now().Format(time.RFC3339),
		Hook:          "PreToolUse",
		Query:         "git commit",
		Results:       []memory.RetrievalResult{},
		FilteredCount: 5,
		SessionID:     "sess-meta",
		Metadata: map[string]string{
			"tool_name": "Bash",
			"project":   "projctl",
		},
	}

	err := memory.LogRetrieval(dir, entry)
	g.Expect(err).NotTo(HaveOccurred())

	logPath := filepath.Join(dir, "retrievals.jsonl")
	data, err := os.ReadFile(logPath)
	g.Expect(err).NotTo(HaveOccurred())

	var decoded memory.RetrievalLogEntry
	err = json.Unmarshal(data, &decoded)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(decoded.Metadata).To(HaveKeyWithValue("tool_name", "Bash"))
	g.Expect(decoded.Metadata).To(HaveKeyWithValue("project", "projctl"))
}

// TestLogRetrievalCreatesDirectory verifies that LogRetrieval creates the
// directory if it doesn't exist.
func TestLogRetrievalCreatesDirectory(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := filepath.Join(t.TempDir(), "nested", "dir")

	entry := memory.RetrievalLogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Hook:      "SessionStart",
		Query:     "test",
		SessionID: "sess-mkdir",
	}

	err := memory.LogRetrieval(dir, entry)
	g.Expect(err).NotTo(HaveOccurred())

	logPath := filepath.Join(dir, "retrievals.jsonl")
	_, err = os.Stat(logPath)
	g.Expect(err).NotTo(HaveOccurred())
}

// TestReadRetrievalLogsAll verifies that ReadRetrievalLogs reads all entries.
func TestReadRetrievalLogsAll(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()

	// Write 3 entries
	hooks := []string{"SessionStart", "UserPromptSubmit", "PreToolUse"}
	for _, hook := range hooks {
		entry := memory.RetrievalLogEntry{
			Timestamp: time.Now().Format(time.RFC3339),
			Hook:      hook,
			Query:     "query for " + hook,
			Results: []memory.RetrievalResult{
				{ID: 1, Content: "result", Score: 0.9, Tier: "embedding"},
			},
			SessionID: "sess-read",
		}
		err := memory.LogRetrieval(dir, entry)
		g.Expect(err).NotTo(HaveOccurred())
	}

	entries, err := memory.ReadRetrievalLogs(dir, memory.RetrievalLogFilter{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(entries).To(HaveLen(3))
}

// TestReadRetrievalLogsFilterByHook verifies filtering by hook type.
func TestReadRetrievalLogsFilterByHook(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()

	hooks := []string{"SessionStart", "UserPromptSubmit", "PreToolUse", "UserPromptSubmit"}
	for _, hook := range hooks {
		entry := memory.RetrievalLogEntry{
			Timestamp: time.Now().Format(time.RFC3339),
			Hook:      hook,
			Query:     "query",
			SessionID: "sess-filter",
		}
		err := memory.LogRetrieval(dir, entry)
		g.Expect(err).NotTo(HaveOccurred())
	}

	entries, err := memory.ReadRetrievalLogs(dir, memory.RetrievalLogFilter{
		Hook: "UserPromptSubmit",
	})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(entries).To(HaveLen(2))
	for _, e := range entries {
		g.Expect(e.Hook).To(Equal("UserPromptSubmit"))
	}
}

// TestReadRetrievalLogsFilterBySession verifies filtering by session ID.
func TestReadRetrievalLogsFilterBySession(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()

	sessions := []string{"sess-1", "sess-2", "sess-1", "sess-2", "sess-1"}
	for _, sid := range sessions {
		entry := memory.RetrievalLogEntry{
			Timestamp: time.Now().Format(time.RFC3339),
			Hook:      "SessionStart",
			Query:     "query",
			SessionID: sid,
		}
		err := memory.LogRetrieval(dir, entry)
		g.Expect(err).NotTo(HaveOccurred())
	}

	entries, err := memory.ReadRetrievalLogs(dir, memory.RetrievalLogFilter{
		SessionID: "sess-1",
	})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(entries).To(HaveLen(3))
}

// TestReadRetrievalLogsFilterBySince verifies filtering by time.
func TestReadRetrievalLogsFilterBySince(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()

	// Write an entry with old timestamp
	oldEntry := memory.RetrievalLogEntry{
		Timestamp: time.Now().Add(-48 * time.Hour).Format(time.RFC3339),
		Hook:      "SessionStart",
		Query:     "old query",
		SessionID: "sess-old",
	}
	err := memory.LogRetrieval(dir, oldEntry)
	g.Expect(err).NotTo(HaveOccurred())

	// Write an entry with recent timestamp
	newEntry := memory.RetrievalLogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Hook:      "SessionStart",
		Query:     "new query",
		SessionID: "sess-new",
	}
	err = memory.LogRetrieval(dir, newEntry)
	g.Expect(err).NotTo(HaveOccurred())

	since := time.Now().Add(-1 * time.Hour)
	entries, err := memory.ReadRetrievalLogs(dir, memory.RetrievalLogFilter{
		Since: &since,
	})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(entries).To(HaveLen(1))
	g.Expect(entries[0].Query).To(Equal("new query"))
}

// TestReadRetrievalLogsEmptyFile returns empty slice for missing file.
func TestReadRetrievalLogsEmptyFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()

	entries, err := memory.ReadRetrievalLogs(dir, memory.RetrievalLogFilter{})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(entries).To(BeEmpty())
}

// splitNonEmpty splits a string by newline and filters empty strings.
func splitNonEmpty(s string) []string {
	var result []string
	for _, line := range strings.Split(s, "\n") {
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}
