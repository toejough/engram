package memory

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
)

func TestApplyLeechAction_PromoteToClaude_EmptyDB(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	// promote_to_claude_md calls markActionRecommended — UPDATE on non-existent ID is no-op
	diag := LeechDiagnosis{
		MemoryID:       999,
		ProposedAction: "promote_to_claude_md",
	}

	err = ApplyLeechAction(db, diag, RealFS{})

	g.Expect(err).ToNot(HaveOccurred())
}

func TestApplyLeechAction_RewriteNoContent(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	// Empty SuggestedContent returns error immediately before ONNX is needed
	diag := LeechDiagnosis{
		MemoryID:         1,
		ProposedAction:   "rewrite",
		SuggestedContent: "",
	}

	err = ApplyLeechAction(db, diag, RealFS{})

	g.Expect(err).To(MatchError(ContainSubstring("requires SuggestedContent")))
}

// ─── leech.go ────────────────────────────────────────────────────────────────

func TestApplyLeechAction_UnknownAction(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	diag := LeechDiagnosis{
		MemoryID:       1,
		ProposedAction: "unknown_action",
	}

	err = ApplyLeechAction(db, diag, RealFS{})

	g.Expect(err).To(MatchError(ContainSubstring("unknown action")))
}

// ─── transaction.go ──────────────────────────────────────────────────────────

func TestCopyDir_NonExistentSrc(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	err := copyDir(osDirOps{}, filepath.Join(t.TempDir(), "nonexistent"), filepath.Join(t.TempDir(), "dst"))

	g.Expect(err).To(HaveOccurred())
}

func TestCopyDir_ValidSrc(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	src := t.TempDir()

	err := os.WriteFile(filepath.Join(src, "file.txt"), []byte("content"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	dst := filepath.Join(t.TempDir(), "copy")

	err = copyDir(osDirOps{}, src, dst)

	g.Expect(err).ToNot(HaveOccurred())
}

func TestCopyFile_NonExistentSrc(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	err := copyFile(filepath.Join(t.TempDir(), "nonexistent.txt"), filepath.Join(t.TempDir(), "dst.txt"))

	g.Expect(err).To(HaveOccurred())
}

func TestCopyFile_ValidSrc(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	src := filepath.Join(t.TempDir(), "src.txt")

	err := os.WriteFile(src, []byte("hello world"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	dst := filepath.Join(t.TempDir(), "dst.txt")

	err = copyFile(src, dst)

	g.Expect(err).ToNot(HaveOccurred())
}

// ─── hooks.go ────────────────────────────────────────────────────────────────

func TestFindE5Score_Match(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	results := []QueryResult{
		{ID: 2, Score: 0.70},
		{ID: 1, Score: 0.85},
	}

	score := findE5Score(1, results)

	g.Expect(score).To(BeNumerically("~", 0.85, 0.001))
}

func TestLogRetrievalTo_WriteError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	w := &failWriteCloser{writeErr: errors.New("disk full")}
	entry := RetrievalLogEntry{Hook: "Stop", Query: "q"}

	err := logRetrievalTo(w, entry)

	g.Expect(err).To(MatchError(ContainSubstring("failed to write retrieval log entry")))
}

func TestLogRetrieval_OpenFileError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Make the log path a directory so os.OpenFile fails with "is a directory".
	dir := t.TempDir()

	err := os.MkdirAll(filepath.Join(dir, "retrievals.jsonl"), 0755)
	g.Expect(err).ToNot(HaveOccurred())

	entry := RetrievalLogEntry{Hook: "Stop", Query: "q"}

	err = LogRetrieval(dir, entry)

	g.Expect(err).To(MatchError(ContainSubstring("failed to open retrievals log")))
}

// ─── retrieval_log.go ────────────────────────────────────────────────────────

func TestLogRetrieval_Success(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dir := t.TempDir()
	entry := RetrievalLogEntry{
		Timestamp: "2026-01-01T00:00:00Z",
		Hook:      "Stop",
		Query:     "test query text",
	}

	err := LogRetrieval(dir, entry)

	g.Expect(err).ToNot(HaveOccurred())
}

func TestMarkActionRecommended_EmptyDB(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	// UPDATE on non-existent memoryID is a no-op — no error
	err = markActionRecommended(db, 999)

	g.Expect(err).ToNot(HaveOccurred())
}

// ─── archive.go ──────────────────────────────────────────────────────────────

func TestPruneArchive_EmptyDB(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	count, err := PruneArchive(db, 30)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(count).To(Equal(0))
}

// ─── feedback.go ─────────────────────────────────────────────────────────────

func TestRecordFeedback_Helpful(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	// embedding_id=999 does not exist; INSERT into feedback works, UPDATE is no-op
	err = RecordFeedback(db, 999, FeedbackHelpful)

	g.Expect(err).ToNot(HaveOccurred())
}

func TestRecordFeedback_Unclear(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	err = RecordFeedback(db, 999, FeedbackUnclear)

	g.Expect(err).ToNot(HaveOccurred())
}

func TestRecordFeedback_Wrong(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	err = RecordFeedback(db, 999, FeedbackWrong)

	g.Expect(err).ToNot(HaveOccurred())
}

func TestRecordHookEvent_NonZeroExit(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	err = RecordHookEvent(db, "PreToolUse", 1, 100)

	g.Expect(err).ToNot(HaveOccurred())
}

// ─── hooks_stats.go ──────────────────────────────────────────────────────────

func TestRecordHookEvent_Success(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	err = RecordHookEvent(db, "Stop", 0, 42)

	g.Expect(err).ToNot(HaveOccurred())
}

// ─── processed_sessions.go ───────────────────────────────────────────────────

func TestResetLastNSessions_EmptyDB(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	count, err := ResetLastNSessions(db, 5, t.TempDir())

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(count).To(Equal(int64(0)))
}

// ─── scoring.go ──────────────────────────────────────────────────────────────

func TestUpdateSurfacingOutcome_NoopUpdate(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	// eventID=999 does not exist; UPDATE is a no-op, no error expected
	err = UpdateSurfacingOutcome(db, 999, 0.8, "positive")

	g.Expect(err).ToNot(HaveOccurred())
}

// failWriteCloser returns an error from every Write call.
type failWriteCloser struct{ writeErr error }

func (f *failWriteCloser) Close() error { return nil }

func (f *failWriteCloser) Write(p []byte) (int, error) { return 0, f.writeErr }
