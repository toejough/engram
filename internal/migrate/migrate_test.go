package migrate_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"engram/internal/migrate"
)

// T-255: --dry-run shows plan without writing, TOML files unchanged, JSONL not deleted.
func TestMigrator_DryRun(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	fixedTime := time.Date(2026, 3, 10, 14, 30, 0, 0, time.UTC)

	lines := []string{
		jsonlLine(t, "memory", "memories/mem-a.toml", 5, 3, 1, 1, fixedTime, nil),
		jsonlLine(t, "memory", "memories/mem-b.toml", 8, 4, 2, 2, fixedTime, nil),
		jsonlLine(t, "memory", "memories/mem-c.toml", 20, 12, 4, 4, fixedTime, nil),
	}
	jsonlContent := strings.Join(lines, "\n") + "\n"

	var (
		writeCount  int
		removeCount int
		output      strings.Builder
	)

	m := migrate.New(
		migrate.WithReadFile(func(name string) ([]byte, error) {
			if strings.HasSuffix(name, "instruction-registry.jsonl") {
				return []byte(jsonlContent), nil
			}

			return []byte("title = \"Test\"\n"), nil
		}),
		migrate.WithWriteFile(func(_ string, _ []byte, _ os.FileMode) error {
			writeCount++

			return nil
		}),
		migrate.WithRenameFile(func(_, _ string) error { return nil }),
		migrate.WithStat(func(_ string) (os.FileInfo, error) { return fakeFileInfo{}, nil }),
		migrate.WithRemove(func(_ string) error {
			removeCount++

			return nil
		}),
		migrate.WithStdout(&output),
	)

	err := m.Run("/data/instruction-registry.jsonl", "/data/memories", true)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	// No writes or deletes in dry-run mode.
	g.Expect(writeCount).To(gomega.Equal(0))
	g.Expect(removeCount).To(gomega.Equal(0))

	// Output describes what would happen.
	out := strings.ToLower(output.String())
	g.Expect(out).To(gomega.ContainSubstring("dry-run"))
	g.Expect(out).To(gomega.ContainSubstring("mem-a.toml"))
}

// mergeIntoTOML error path: write failure propagates.
func TestMigrator_MergeIntoTOML_WriteError(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	fixedTime := time.Date(2026, 3, 10, 14, 30, 0, 0, time.UTC)

	lines := []string{
		jsonlLine(t, "memory", "memories/mem-a.toml", 5, 3, 1, 1, fixedTime, nil),
	}
	jsonlContent := strings.Join(lines, "\n") + "\n"

	writeErr := errors.New("disk full")

	migrator := migrate.New(
		migrate.WithReadFile(func(name string) ([]byte, error) {
			if strings.HasSuffix(name, "instruction-registry.jsonl") {
				return []byte(jsonlContent), nil
			}

			return []byte("title = \"Test\"\n"), nil
		}),
		migrate.WithWriteFile(func(_ string, _ []byte, _ os.FileMode) error {
			return writeErr
		}),
		migrate.WithRenameFile(func(_, _ string) error { return nil }),
		migrate.WithStat(func(_ string) (os.FileInfo, error) {
			return fakeFileInfo{}, nil
		}),
		migrate.WithRemove(func(_ string) error { return nil }),
		migrate.WithStdout(io.Discard),
	)

	err := migrator.Run(
		"/data/instruction-registry.jsonl", "/data/memories", false,
	)
	g.Expect(err).To(gomega.HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(gomega.ContainSubstring("disk full"))
	}
}

// T-252 supplement: migration with links exercises buildUpdates links branch.
func TestMigrator_MergesLinksIntoTOMLs(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	fixedTime := time.Date(2026, 3, 10, 14, 30, 0, 0, time.UTC)

	entry := map[string]any{
		"id":             "memory:mem-links.toml",
		"source_type":    "memory",
		"source_path":    "memories/mem-links.toml",
		"title":          "Test",
		"content_hash":   "abc123",
		"registered_at":  "2026-01-15T00:00:00Z",
		"updated_at":     "2026-01-15T00:00:00Z",
		"surfaced_count": 5,
		"last_surfaced":  fixedTime.Format(time.RFC3339),
		"evaluations": map[string]any{
			"followed": 3, "contradicted": 1, "ignored": 1,
		},
		"enforcement_level": "advisory",
		"links": []map[string]any{
			{"target": "memories/other.toml", "weight": 0.8, "basis": "co-surfacing"},
		},
	}

	data, marshalErr := json.Marshal(entry)
	if marshalErr != nil {
		t.Fatalf("marshal failed: %v", marshalErr)
	}

	jsonlContent := string(data) + "\n"

	const baseToml = "title = \"Test Memory\"\ncontent = \"Some content\"\n"

	var lastWrittenData []byte

	migrator := migrate.New(
		migrate.WithReadFile(func(name string) ([]byte, error) {
			if strings.HasSuffix(name, "instruction-registry.jsonl") {
				return []byte(jsonlContent), nil
			}

			return []byte(baseToml), nil
		}),
		migrate.WithWriteFile(func(_ string, written []byte, _ os.FileMode) error {
			lastWrittenData = written

			return nil
		}),
		migrate.WithRenameFile(func(_, _ string) error { return nil }),
		migrate.WithStat(func(_ string) (os.FileInfo, error) {
			return fakeFileInfo{}, nil
		}),
		migrate.WithRemove(func(_ string) error { return nil }),
		migrate.WithStdout(io.Discard),
	)

	err := migrator.Run(
		"/data/instruction-registry.jsonl", "/data/memories", false,
	)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	content := string(lastWrittenData)
	g.Expect(content).To(gomega.ContainSubstring("target"))
	g.Expect(content).To(gomega.ContainSubstring("other.toml"))
	g.Expect(content).To(gomega.ContainSubstring("co-surfacing"))
}

// T-251: Migration merges JSONL metrics into 3 matching TOMLs, deletes JSONL.
func TestMigrator_MergesMetricsIntoTOMLs(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	fixedTime := time.Date(2026, 3, 10, 14, 30, 0, 0, time.UTC)

	lines := []string{
		jsonlLine(t, "memory", "memories/mem-a.toml", 15, 10, 2, 3, fixedTime, nil),
		jsonlLine(t, "memory", "memories/mem-b.toml", 8, 5, 1, 2, fixedTime, nil),
		jsonlLine(t, "memory", "memories/mem-c.toml", 20, 12, 4, 4, fixedTime, nil),
	}
	jsonlContent := strings.Join(lines, "\n") + "\n"

	const baseToml = `title = "Test Memory"` + "\n" + `content = "Some content"` + "\n"

	var (
		renameCount     int
		removedPath     string
		lastWrittenData []byte
	)

	m := migrate.New(
		migrate.WithReadFile(func(name string) ([]byte, error) {
			if strings.HasSuffix(name, "instruction-registry.jsonl") {
				return []byte(jsonlContent), nil
			}

			return []byte(baseToml), nil
		}),
		migrate.WithWriteFile(func(_ string, data []byte, _ os.FileMode) error {
			lastWrittenData = data

			return nil
		}),
		migrate.WithRenameFile(func(_, _ string) error {
			renameCount++

			return nil
		}),
		migrate.WithStat(func(_ string) (os.FileInfo, error) { return fakeFileInfo{}, nil }),
		migrate.WithRemove(func(name string) error {
			removedPath = name

			return nil
		}),
		migrate.WithStdout(io.Discard),
	)

	err := m.Run("/data/instruction-registry.jsonl", "/data/memories", false)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	// JSONL deleted.
	g.Expect(removedPath).To(gomega.Equal("/data/instruction-registry.jsonl"))

	// 3 TOMLs rewritten.
	g.Expect(renameCount).To(gomega.Equal(3))

	// Written TOML contains expected metric fields.
	content := string(lastWrittenData)
	g.Expect(content).To(gomega.ContainSubstring("surfaced_count"))
	g.Expect(content).To(gomega.ContainSubstring("followed_count"))
	g.Expect(content).To(gomega.ContainSubstring("contradicted_count"))
	g.Expect(content).To(gomega.ContainSubstring("ignored_count"))
	g.Expect(content).To(gomega.ContainSubstring("last_surfaced_at"))
}

// T-254: Missing JSONL = success (exit 0), idempotent.
func TestMigrator_MissingJSONL_Idempotent(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	var (
		writeCount int
		output     strings.Builder
	)

	m := migrate.New(
		migrate.WithStat(func(_ string) (os.FileInfo, error) { return nil, os.ErrNotExist }),
		migrate.WithReadFile(func(_ string) ([]byte, error) { return nil, os.ErrNotExist }),
		migrate.WithWriteFile(func(_ string, _ []byte, _ os.FileMode) error {
			writeCount++

			return nil
		}),
		migrate.WithRenameFile(func(_, _ string) error { return nil }),
		migrate.WithRemove(func(_ string) error { return nil }),
		migrate.WithStdout(&output),
	)

	err := m.Run("/data/instruction-registry.jsonl", "/data/memories", false)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(strings.ToLower(output.String())).To(gomega.ContainSubstring("nothing to migrate"))
	g.Expect(writeCount).To(gomega.Equal(0))
}

// T-256: Migration preserves absorbed history from JSONL.
func TestMigrator_PreservesAbsorbedHistory(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	fixedTime := time.Date(2026, 3, 10, 14, 30, 0, 0, time.UTC)
	mergedAt := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	absorbed := []map[string]any{
		{
			"from":           "old-mem-1",
			"surfaced_count": 50,
			"evaluations":    map[string]any{"followed": 30, "contradicted": 10, "ignored": 10},
			"content_hash":   "xyz1",
			"merged_at":      mergedAt.Format(time.RFC3339),
		},
		{
			"from":           "old-mem-2",
			"surfaced_count": 20,
			"evaluations":    map[string]any{"followed": 10, "contradicted": 5, "ignored": 5},
			"content_hash":   "xyz2",
			"merged_at":      mergedAt.Format(time.RFC3339),
		},
	}

	lines := []string{
		jsonlLine(t, "memory", "memories/mem-a.toml", 15, 10, 2, 3, fixedTime, absorbed),
	}
	jsonlContent := strings.Join(lines, "\n") + "\n"

	var lastWrittenData []byte

	m := migrate.New(
		migrate.WithReadFile(func(name string) ([]byte, error) {
			if strings.HasSuffix(name, "instruction-registry.jsonl") {
				return []byte(jsonlContent), nil
			}

			return []byte("title = \"Test Memory\"\n"), nil
		}),
		migrate.WithWriteFile(func(_ string, data []byte, _ os.FileMode) error {
			lastWrittenData = data

			return nil
		}),
		migrate.WithRenameFile(func(_, _ string) error { return nil }),
		migrate.WithStat(func(_ string) (os.FileInfo, error) { return fakeFileInfo{}, nil }),
		migrate.WithRemove(func(_ string) error { return nil }),
		migrate.WithStdout(io.Discard),
	)

	err := m.Run("/data/instruction-registry.jsonl", "/data/memories", false)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	// Absorbed records present in written TOML.
	g.Expect(lastWrittenData).NotTo(gomega.BeNil())

	content := string(lastWrittenData)
	g.Expect(content).To(gomega.ContainSubstring("absorbed"))
	g.Expect(content).To(gomega.ContainSubstring("old-mem-1"))
	g.Expect(content).To(gomega.ContainSubstring("old-mem-2"))
}

// T-252: Migration skips non-memory entries, logs skip count.
func TestMigrator_SkipsNonMemoryEntries(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	fixedTime := time.Date(2026, 3, 10, 14, 30, 0, 0, time.UTC)

	lines := []string{
		jsonlLine(t, "memory", "memories/mem-a.toml", 5, 3, 1, 1, fixedTime, nil),
		jsonlLine(t, "memory", "memories/mem-b.toml", 8, 4, 2, 2, fixedTime, nil),
		jsonlLine(t, "claude-md", "claude.md", 0, 0, 0, 0, fixedTime, nil),
		jsonlLine(t, "rule", "rules/foo.md", 0, 0, 0, 0, fixedTime, nil),
		jsonlLine(t, "skill", "skills/bar.md", 0, 0, 0, 0, fixedTime, nil),
	}
	jsonlContent := strings.Join(lines, "\n") + "\n"

	var (
		renameCount int
		output      strings.Builder
	)

	m := migrate.New(
		migrate.WithReadFile(func(name string) ([]byte, error) {
			if strings.HasSuffix(name, "instruction-registry.jsonl") {
				return []byte(jsonlContent), nil
			}

			return []byte("title = \"Test\"\n"), nil
		}),
		migrate.WithWriteFile(func(_ string, _ []byte, _ os.FileMode) error { return nil }),
		migrate.WithRenameFile(func(_, _ string) error {
			renameCount++

			return nil
		}),
		migrate.WithStat(func(_ string) (os.FileInfo, error) { return fakeFileInfo{}, nil }),
		migrate.WithRemove(func(_ string) error { return nil }),
		migrate.WithStdout(&output),
	)

	err := m.Run("/data/instruction-registry.jsonl", "/data/memories", false)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	// Only 2 memory TOMLs updated.
	g.Expect(renameCount).To(gomega.Equal(2))

	// Skip count (3 non-memory) appears in output.
	out := output.String()
	g.Expect(out).To(gomega.ContainSubstring("3"))
	g.Expect(strings.ToLower(out)).To(gomega.ContainSubstring("skip"))
}

// T-253: Migration skips unmatched memory entries (deleted TOML), logs count.
func TestMigrator_SkipsUnmatchedEntries(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)

	fixedTime := time.Date(2026, 3, 10, 14, 30, 0, 0, time.UTC)

	lines := []string{
		jsonlLine(t, "memory", "memories/mem-a.toml", 5, 3, 1, 1, fixedTime, nil),
		jsonlLine(t, "memory", "memories/mem-deleted.toml", 8, 4, 2, 2, fixedTime, nil),
	}
	jsonlContent := strings.Join(lines, "\n") + "\n"

	var (
		renameCount int
		output      strings.Builder
	)

	m := migrate.New(
		migrate.WithReadFile(func(name string) ([]byte, error) {
			if strings.HasSuffix(name, "instruction-registry.jsonl") {
				return []byte(jsonlContent), nil
			}

			if strings.Contains(name, "mem-deleted") {
				return nil, os.ErrNotExist
			}

			return []byte("title = \"Test\"\n"), nil
		}),
		migrate.WithWriteFile(func(_ string, _ []byte, _ os.FileMode) error { return nil }),
		migrate.WithRenameFile(func(_, _ string) error {
			renameCount++

			return nil
		}),
		migrate.WithStat(func(name string) (os.FileInfo, error) {
			if strings.Contains(name, "mem-deleted") {
				return nil, os.ErrNotExist
			}

			return fakeFileInfo{}, nil
		}),
		migrate.WithRemove(func(_ string) error { return nil }),
		migrate.WithStdout(&output),
	)

	err := m.Run("/data/instruction-registry.jsonl", "/data/memories", false)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	// Only 1 TOML updated (the one that exists).
	g.Expect(renameCount).To(gomega.Equal(1))

	// Unmatched entry appears in output.
	g.Expect(strings.ToLower(output.String())).To(gomega.ContainSubstring("unmatched"))
}

// fakeFileInfo is a minimal os.FileInfo stub for tests.
type fakeFileInfo struct{}

func (fakeFileInfo) IsDir() bool { return false }

func (fakeFileInfo) ModTime() time.Time { return time.Time{} }

func (fakeFileInfo) Mode() fs.FileMode { return 0 }

func (fakeFileInfo) Name() string { return "" }

func (fakeFileInfo) Size() int64 { return 0 }

func (fakeFileInfo) Sys() any { return nil }

// jsonlLine builds a single JSONL line for a given entry configuration.
func jsonlLine(
	t *testing.T,
	sourceType, sourcePath string,
	surfacedCount, followed, contradicted, ignored int,
	lastSurfaced time.Time,
	absorbed []map[string]any,
) string {
	t.Helper()

	entry := map[string]any{
		"id":                fmt.Sprintf("%s:%s", sourceType, filepath.Base(sourcePath)),
		"source_type":       sourceType,
		"source_path":       sourcePath,
		"title":             "Test",
		"content_hash":      "abc123",
		"registered_at":     "2026-01-15T00:00:00Z",
		"updated_at":        "2026-01-15T00:00:00Z",
		"surfaced_count":    surfacedCount,
		"last_surfaced":     lastSurfaced.Format(time.RFC3339),
		"evaluations":       map[string]any{"followed": followed, "contradicted": contradicted, "ignored": ignored},
		"enforcement_level": "advisory",
	}

	if len(absorbed) > 0 {
		entry["absorbed"] = absorbed
	}

	data, marshalErr := json.Marshal(entry)
	if marshalErr != nil {
		t.Fatalf("jsonlLine: marshal failed: %v", marshalErr)
	}

	return string(data)
}
