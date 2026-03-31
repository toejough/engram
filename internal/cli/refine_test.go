package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
	"engram/internal/memory"
)

func TestFindTranscriptForMemory_MatchesClosest(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	projectsDir := t.TempDir()

	// Create three transcript files with different mtimes.
	base := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	transcripts := make([]string, 3)

	for i, offset := range []time.Duration{-10 * time.Hour, -2 * time.Hour, -36 * time.Hour} {
		path := filepath.Join(projectsDir, "project1", "session"+string(rune('A'+i))+".jsonl")
		g.Expect(os.MkdirAll(filepath.Dir(path), 0o755)).To(Succeed())
		g.Expect(os.WriteFile(path, []byte(`{}`), 0o644)).To(Succeed())

		mtime := base.Add(offset)
		g.Expect(os.Chtimes(path, mtime, mtime)).To(Succeed())

		transcripts[i] = path
	}

	// Memory created at base time — closest should be the 2h before file.
	record := memory.MemoryRecord{
		CreatedAt: base.Format(time.RFC3339),
	}

	result := cli.ExportFindTranscriptForMemory(record, transcripts)
	g.Expect(result).NotTo(BeEmpty(), "should find a matching transcript")

	// The 2h-before file (index 1) should be closest.
	g.Expect(result).To(Equal(transcripts[1]))
}

func TestFindTranscriptForMemory_NoMatchBeyond24h(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	projectsDir := t.TempDir()

	base := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	// All transcripts are more than 24h away.
	path := filepath.Join(projectsDir, "project1", "session.jsonl")
	g.Expect(os.MkdirAll(filepath.Dir(path), 0o755)).To(Succeed())
	g.Expect(os.WriteFile(path, []byte(`{}`), 0o644)).To(Succeed())

	farMtime := base.Add(-48 * time.Hour)
	g.Expect(os.Chtimes(path, farMtime, farMtime)).To(Succeed())

	record := memory.MemoryRecord{
		CreatedAt: base.Format(time.RFC3339),
	}

	result := cli.ExportFindTranscriptForMemory(record, []string{path})
	g.Expect(result).To(BeEmpty(), "should return empty string when all transcripts >24h away")
}

func TestRunRefine_DryRun(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memoriesDir, 0o755)).To(Succeed())

	memTOML := `situation = "when writing Go code"
behavior = "returning bare errors"
impact = "hard to debug"
action = "wrap errors with context"
created_at = "2024-01-01T00:00:00Z"
updated_at = "2024-01-01T00:00:00Z"
`
	g.Expect(os.WriteFile(
		filepath.Join(memoriesDir, "wrap-errors.toml"),
		[]byte(memTOML),
		0o644,
	)).To(Succeed())

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "refine",
			"--data-dir", dataDir,
			"--dry-run",
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// In dry-run mode, the file must not be modified.
	original, readErr := os.ReadFile(filepath.Join(memoriesDir, "wrap-errors.toml"))
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(string(original)).To(ContainSubstring("wrap errors with context"),
		"dry-run must not modify memory files")

	// Output should report what it found.
	output := stdout.String()
	g.Expect(output).To(ContainSubstring("refine"),
		"output should mention refine operation")
}
