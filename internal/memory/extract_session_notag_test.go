package memory

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
)

// ─── applyRecurrenceCheck tests ───────────────────────────────────────────────

func TestApplyRecurrenceCheck_EmptyResults(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := applyRecurrenceCheck(nil, "some correction", t.TempDir())
	g.Expect(err).ToNot(HaveOccurred())
}

func TestApplyRecurrenceCheck_HighScore(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	memoryDir := filepath.Join(t.TempDir(), "memory")

	err := os.MkdirAll(memoryDir, 0o755)
	if err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	results := []QueryResult{
		{ID: 1, Score: 0.95, Content: "always use targ instead of mage"},
	}

	err = applyRecurrenceCheck(results, "always use targ instead of mage", memoryDir)
	g.Expect(err).ToNot(HaveOccurred())

	// Verify changelog was written
	_, statErr := os.Stat(filepath.Join(memoryDir, "changelog.jsonl"))
	g.Expect(statErr).ToNot(HaveOccurred())
}

func TestApplyRecurrenceCheck_LowScore(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	memoryDir := filepath.Join(t.TempDir(), "memory")

	results := []QueryResult{
		{ID: 1, Score: 0.5, Content: "some unrelated content"},
	}

	// Score ≤ 0.8 → no changelog written, no error
	err := applyRecurrenceCheck(results, "test correction", memoryDir)
	g.Expect(err).ToNot(HaveOccurred())
}

func TestApplyRecurrenceCheck_WriteChangelogFails(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Create a file where memoryRoot should be → MkdirAll fails inside WriteChangelogEntry
	f, err := os.CreateTemp(t.TempDir(), "recurrence-test-*")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}

	_ = f.Close()

	results := []QueryResult{
		{ID: 2, Score: 0.99, Content: "always commit before rebasing"},
	}

	err = applyRecurrenceCheck(results, "always commit before rebasing", f.Name())
	g.Expect(err).To(HaveOccurred())
	g.Expect(err).To(MatchError(ContainSubstring("failed to write changelog entry")))
}

func TestDetectCorrectionRecurrence_EmptyDB(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// With empty DB, Query short-circuits → no results → no recurrence → nil
	err := detectCorrectionRecurrence("always use targ instead of mage", t.TempDir())

	g.Expect(err).ToNot(HaveOccurred())
}

func TestExtractLogf_Callable(t *testing.T) {
	t.Parallel()

	// nil writer is a no-op — must not panic
	extractLogf(nil, "test message %s %d", "value", 42)
}

func TestExtractLogf_WritesToWriter(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var buf bytes.Buffer

	extractLogf(&buf, "hello %s %d", "world", 99)

	g.Expect(buf.String()).To(ContainSubstring("[ExtractSession]"))
	g.Expect(buf.String()).To(ContainSubstring("hello world 99"))
}

func TestExtractSession_InvalidJSONL(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dir := t.TempDir()
	transcriptPath := filepath.Join(dir, "session.jsonl")

	// Invalid JSON lines are skipped; result still succeeds with 0 items
	err := os.WriteFile(transcriptPath, []byte("not json\nalso not json\n"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := ExtractSession(ExtractSessionOpts{
		TranscriptPath: transcriptPath,
		MemoryRoot:     dir,
	})

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())
}

// TestExtractSession_LearnFailPartial covers lines 211-214 (partial status when Learn fails)
// and lines 220-226 (correction recurrence error logged to stderr).
func TestExtractSession_LearnFailPartial(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dir := t.TempDir()
	transcriptPath := filepath.Join(dir, "session.jsonl")

	// Produces a correction item; Learn will fail because MemoryRoot is invalid.
	transcript := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Let me run the tests."}]}}
{"type":"user","message":{"role":"user","content":[{"type":"text","text":"No, never use mage. Always use targ for builds."}]}}
`

	err := os.WriteFile(transcriptPath, []byte(transcript), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	// Invalid MemoryRoot → Learn fails → result.Status = "partial".
	// detectCorrectionRecurrence also fails but error is only logged to stderr.
	result, err := ExtractSession(ExtractSessionOpts{
		TranscriptPath: transcriptPath,
		MemoryRoot:     "/dev/null/invalid",
	})

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())

	if result != nil {
		g.Expect(result.Status).To(Equal("partial"))
	}
}

func TestExtractSession_NonExistentTranscript(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	_, err := ExtractSession(ExtractSessionOpts{
		TranscriptPath: filepath.Join(t.TempDir(), "nonexistent.jsonl"),
		MemoryRoot:     t.TempDir(),
	})

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("failed to open transcript"))
}

// TestExtractSession_WithCorrectionItem covers lines 196-226 (item processing loop, Learn ok,
// correction recurrence) using a transcript that produces a "correction" item.
func TestExtractSession_WithCorrectionItem(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dir := t.TempDir()

	db, err := InitDBForTest(dir)
	g.Expect(err).ToNot(HaveOccurred())

	err = db.Close()
	g.Expect(err).ToNot(HaveOccurred())

	transcriptPath := filepath.Join(dir, "session.jsonl")

	// assistant text → user correction text: produces a "correction" SessionExtractedItem.
	transcript := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"I ran the tests."}]}}
{"type":"user","message":{"role":"user","content":[{"type":"text","text":"No, never use mage for builds. Always use targ."}]}}
`

	err = os.WriteFile(transcriptPath, []byte(transcript), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	result, err := ExtractSession(ExtractSessionOpts{
		TranscriptPath: transcriptPath,
		MemoryRoot:     dir,
	})

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())

	if result != nil {
		g.Expect(result.ItemsExtracted).To(BeNumerically(">", 0))
		g.Expect(result.Status).To(Equal("success"))
	}
}

func TestOpenExtractLog_Callable(t *testing.T) {
	t.Parallel()

	// openExtractLog should either succeed or fail gracefully
	f, err := openExtractLog()
	if err == nil {
		_ = f.Close()
	}
}
