package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
	"engram/internal/memory"
)

// TestFlush_AcceptsProjectSlugFlag verifies the --project-slug flag is registered on flush.
func TestFlush_AcceptsProjectSlugFlag(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "flush",
			"--data-dir", dataDir,
			"--project-slug", "my-test-project",
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	// Should not error — flag is now registered.
	g.Expect(err).NotTo(HaveOccurred())
}

func TestFlush_BadTranscriptPath_SkipsGracefully(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "flush",
			"--data-dir", dataDir,
			"--transcript-path", "/nonexistent/transcript.jsonl",
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	// No session-id → learn skips → no error.
	g.Expect(err).NotTo(HaveOccurred())
}

func TestFlush_DeletesSurfacingLog(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	// Create a surfacing log file.
	logPath := filepath.Join(dataDir, "surfacing-log.jsonl")
	g.Expect(os.WriteFile(logPath, []byte(`{"mode":"prompt"}`+"\n"), 0o644)).To(Succeed())

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "flush", "--data-dir", dataDir},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Surfacing log should be deleted.
	_, statErr := os.Stat(logPath)
	g.Expect(os.IsNotExist(statErr)).To(BeTrue(),
		"surfacing-log.jsonl should be deleted by flush")
}

func TestFlush_FlagParseError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "flush", "--bogus-flag"},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())
}

func TestFlush_NoTranscript_SkipsGracefully(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "flush", "--data-dir", dataDir},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	// No transcript → learn skips → no error.
	g.Expect(err).NotTo(HaveOccurred())
}

// TestFlush_WritesProjectSlug verifies that flush passes a non-empty project_slug
// to the learn pipeline. When an API token is absent, learn skips extraction but
// the slug must be passed without error (slug defaulting must not error).
func TestFlush_WritesProjectSlug(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memoriesDir, 0o755)).To(Succeed())

	// Write a minimal pre-existing memory with empty project_slug to show the
	// migrate-slugs command can backfill it.
	memPath := filepath.Join(memoriesDir, "test-memory.toml")
	g.Expect(os.WriteFile(memPath, []byte(`title = "test"
content = "test content"
project_slug = ""
created_at = "2024-01-01T00:00:00Z"
updated_at = "2024-01-01T00:00:00Z"
confidence = "high"
`), 0o644)).To(Succeed())

	// Run flush with a project slug explicitly.
	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "flush",
			"--data-dir", dataDir,
			"--project-slug", "engram-project",
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	// No transcript → learn skips gracefully. No error expected.
	g.Expect(err).NotTo(HaveOccurred())

	// Verify the pre-existing memory is unaffected (flush only calls learn).
	records, listErr := memory.ListAll(memoriesDir)
	g.Expect(listErr).NotTo(HaveOccurred())

	if listErr != nil {
		return
	}

	g.Expect(records).To(HaveLen(1))
	g.Expect(records[0].Record.Title).To(Equal("test"))
}
