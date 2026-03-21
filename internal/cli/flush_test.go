package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
)

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

func TestFlush_MissingDataDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "flush"},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("--data-dir"))
	}
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
