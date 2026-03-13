package signal_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"engram/internal/signal"
)

// TestAppend_CreateTempError verifies Append returns error when createTemp fails.
func TestAppend_CreateTempError(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	tmpDir := t.TempDir()
	queuePath := filepath.Join(tmpDir, "graduation-queue.jsonl")

	store := signal.NewGraduationStore(
		signal.WithGraduationCreateTemp(func(_, _ string) (*os.File, error) {
			return nil, errors.New("disk full")
		}),
	)

	entry := signal.GraduationEntry{
		ID:         "abc",
		MemoryPath: "mem/foo.toml",
		Status:     "pending",
		DetectedAt: time.Now(),
	}

	err := store.Append(entry, queuePath)
	g.Expect(err).To(gomega.HaveOccurred())
}

// TestAppend_ReadError verifies Append returns error when readFile fails with non-ErrNotExist.
func TestAppend_ReadError(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	tmpDir := t.TempDir()
	queuePath := filepath.Join(tmpDir, "graduation-queue.jsonl")

	store := signal.NewGraduationStore(
		signal.WithGraduationReadFile(func(_ string) ([]byte, error) {
			return nil, errors.New("permission denied")
		}),
	)

	entry := signal.GraduationEntry{
		ID:         "abc",
		MemoryPath: "mem/foo.toml",
		Status:     "pending",
		DetectedAt: time.Now(),
	}

	err := store.Append(entry, queuePath)
	g.Expect(err).To(gomega.HaveOccurred())
}

// TestAppend_RenameError verifies Append returns error when rename fails.
func TestAppend_RenameError(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	tmpDir := t.TempDir()
	queuePath := filepath.Join(tmpDir, "graduation-queue.jsonl")

	store := signal.NewGraduationStore(
		signal.WithGraduationRename(func(_, _ string) error {
			return errors.New("rename failed")
		}),
		signal.WithGraduationRemove(func(_ string) error { return nil }),
	)

	entry := signal.GraduationEntry{
		ID:         "abc",
		MemoryPath: "mem/foo.toml",
		Status:     "pending",
		DetectedAt: time.Now(),
	}

	err := store.Append(entry, queuePath)
	g.Expect(err).To(gomega.HaveOccurred())
}

// TestAppend_WriteError verifies Append returns error when the temp file write fails.
// We achieve this by returning an already-closed *os.File from createTemp.
func TestAppend_WriteError(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	tmpDir := t.TempDir()
	queuePath := filepath.Join(tmpDir, "graduation-queue.jsonl")

	store := signal.NewGraduationStore(
		signal.WithGraduationCreateTemp(func(dir, pattern string) (*os.File, error) {
			f, err := os.CreateTemp(dir, pattern)
			if err != nil {
				return nil, err
			}

			_ = f.Close() // close first so WriteString fails

			return f, nil
		}),
		signal.WithGraduationRemove(func(_ string) error { return nil }),
	)

	entry := signal.GraduationEntry{
		ID:         "abc",
		MemoryPath: "mem/foo.toml",
		Status:     "pending",
		DetectedAt: time.Now(),
	}

	err := store.Append(entry, queuePath)
	g.Expect(err).To(gomega.HaveOccurred())
}

// T-P6f-di: GraduationStore DI options are applied without error.
func TestGraduationStore_DIOptions(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	store := signal.NewGraduationStore(
		signal.WithGraduationReadFile(func(string) ([]byte, error) { return []byte{}, nil }),
		signal.WithGraduationCreateTemp(func(_, _ string) (*os.File, error) {
			return nil, errors.New("mock: not implemented")
		}),
		signal.WithGraduationRename(func(_, _ string) error { return nil }),
		signal.WithGraduationRemove(func(_ string) error { return nil }),
	)
	g.Expect(store).NotTo(gomega.BeNil())
}

// T-P6f-1: Append writes entry to JSONL.
func TestP6f1_AppendWritesEntryToJSONL(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	// Create temp file for the queue
	tmpDir := t.TempDir()
	queuePath := filepath.Join(tmpDir, "graduation-queue.jsonl")

	store := signal.NewGraduationStore()

	entry := signal.GraduationEntry{
		ID:             "abc123",
		MemoryPath:     "mem/foo.toml",
		Recommendation: "CLAUDE.md",
		Status:         "pending",
		DetectedAt:     time.Now(),
	}

	err := store.Append(entry, queuePath)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	// Read back
	data, _ := os.ReadFile(queuePath)
	g.Expect(data).To(gomega.ContainSubstring("abc123"))
	g.Expect(data).To(gomega.ContainSubstring("mem/foo.toml"))
}

// T-P6f-2: List reads all entries, skips malformed.
func TestP6f2_ListReadsEntriesSkipsMalformed(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	tmpDir := t.TempDir()
	queuePath := filepath.Join(tmpDir, "graduation-queue.jsonl")

	// Write one valid, one malformed, one valid
	entryA := `{"id":"abc","memory_path":"mem/a.toml","recommendation":"CLAUDE.md",` +
		`"status":"pending","detected_at":"2026-03-10T12:00:00Z","resolved_at":"","issue_url":""}`
	entryB := `{"id":"def","memory_path":"mem/b.toml","recommendation":"skill",` +
		`"status":"pending","detected_at":"2026-03-10T12:00:00Z","resolved_at":"","issue_url":""}`
	content := entryA + "\nthis is not json\n" + entryB + "\n"
	_ = os.WriteFile(queuePath, []byte(content), 0o600)

	store := signal.NewGraduationStore()
	entries, err := store.List(queuePath)

	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(entries).To(gomega.HaveLen(2))
	g.Expect(entries[0].ID).To(gomega.Equal("abc"))
	g.Expect(entries[1].ID).To(gomega.Equal("def"))
}

// T-P6f-3: List returns empty slice for missing file.
func TestP6f3_ListReturnEmptyForMissingFile(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	tmpDir := t.TempDir()
	queuePath := filepath.Join(tmpDir, "graduation-queue.jsonl") // does not exist

	store := signal.NewGraduationStore()
	entries, err := store.List(queuePath)

	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(entries).To(gomega.BeEmpty())
}

// T-P6f-4: SetStatus updates matching entry.
func TestP6f4_SetStatusUpdatesMatchingEntry(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	tmpDir := t.TempDir()
	queuePath := filepath.Join(tmpDir, "graduation-queue.jsonl")

	// Write initial entry
	line := `{"id":"abc123","memory_path":"mem/foo.toml","recommendation":"CLAUDE.md",` +
		`"status":"pending","detected_at":"2026-03-10T12:00:00Z","resolved_at":"","issue_url":""}` + "\n"
	_ = os.WriteFile(queuePath, []byte(line), 0o600)

	store := signal.NewGraduationStore()
	err := store.SetStatus(
		queuePath,
		"abc123",
		"accepted",
		"2026-03-10T13:00:00Z",
		"https://github.com/x/y/issues/1",
	)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	// Verify updated
	entries, listErr := store.List(queuePath)
	g.Expect(listErr).NotTo(gomega.HaveOccurred())

	if listErr != nil {
		return
	}

	g.Expect(entries).To(gomega.HaveLen(1))
	g.Expect(entries[0].Status).To(gomega.Equal("accepted"))
	g.Expect(entries[0].IssueURL).To(gomega.Equal("https://github.com/x/y/issues/1"))
	g.Expect(entries[0].ResolvedAt).NotTo(gomega.BeEmpty())
}

// T-P6f-5: SetStatus returns ErrGraduationNotFound for unknown ID.
func TestP6f5_SetStatusReturnsErrorForUnknownID(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	tmpDir := t.TempDir()
	queuePath := filepath.Join(tmpDir, "graduation-queue.jsonl")

	line := `{"id":"abc","memory_path":"mem/a.toml","recommendation":"CLAUDE.md",` +
		`"status":"pending","detected_at":"2026-03-10T12:00:00Z","resolved_at":"","issue_url":""}` + "\n"
	_ = os.WriteFile(queuePath, []byte(line), 0o600)

	store := signal.NewGraduationStore()
	err := store.SetStatus(queuePath, "unknown", "accepted", "2026-03-10T13:00:00Z", "https://...")

	g.Expect(err).To(gomega.MatchError(signal.ErrGraduationNotFound))
}
