package cli_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
)

func TestRunEvaluate_MissingSessionID(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	transcriptPath := filepath.Join(t.TempDir(), "t.jsonl")
	g.Expect(os.WriteFile(transcriptPath, []byte(""), 0o644)).To(Succeed())

	var stdout bytes.Buffer

	err := cli.ExportRunEvaluateWith(
		[]string{"--transcript-path", transcriptPath, "--data-dir", dataDir},
		&stdout,
		nil,
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("session-id"))
	}
}

func TestRunEvaluate_MissingTranscriptPath(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	var stdout bytes.Buffer

	err := cli.ExportRunEvaluateWith(
		[]string{"--session-id", "sess-1", "--data-dir", dataDir},
		&stdout,
		nil,
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("transcript-path"))
	}
}

func TestRunEvaluate_NoPendingMemories_PrintsEmptyResults(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memoriesDir, 0o755)).To(Succeed())

	transcriptPath := filepath.Join(t.TempDir(), "t.jsonl")
	g.Expect(os.WriteFile(transcriptPath, []byte(""), 0o644)).To(Succeed())

	var stdout bytes.Buffer

	err := cli.ExportRunEvaluateWith(
		[]string{
			"--session-id", "sess-none",
			"--transcript-path", transcriptPath,
			"--data-dir", dataDir,
		},
		&stdout,
		nil,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring("[]"))
}

func TestRunEvaluate_ParseFlagError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var stdout bytes.Buffer

	err := cli.ExportRunEvaluateWith(
		[]string{"--bogus-flag"},
		&stdout,
		nil,
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("evaluate"))
	}
}

func TestRunEvaluate_WithPendingMemory_UpdatesCounters(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memoriesDir, 0o755)).To(Succeed())

	// Write a memory TOML with a pending evaluation for session "sess-eval".
	memoryTOML := `situation = "when writing Go code"
behavior = "returning bare errors"
impact = "hard to debug"
action = "wrap errors with context"
created_at = "2024-01-01T00:00:00Z"
updated_at = "2024-01-01T00:00:00Z"
surfaced_count = 1
followed_count = 0
not_followed_count = 0
irrelevant_count = 0

[[pending_evaluations]]
surfaced_at = "2024-01-01T00:00:00Z"
user_prompt = "write some Go code"
session_id = "sess-eval"
project_slug = "test-project"
`
	memoryPath := filepath.Join(memoriesDir, "wrap-errors.toml")
	g.Expect(os.WriteFile(memoryPath, []byte(memoryTOML), 0o644)).To(Succeed())

	// Write a transcript with assistant content showing the action was followed.
	transcriptLine := `{"type":"assistant","message":{"content":[{"type":"text","text":"wrapped error"}]}}` + "\n"
	transcriptPath := filepath.Join(t.TempDir(), "transcript.jsonl")
	g.Expect(os.WriteFile(transcriptPath, []byte(transcriptLine), 0o644)).To(Succeed())

	// Mock caller always returns FOLLOWED.
	mockCaller := func(_ context.Context, _, _, _ string) (string, error) {
		return "FOLLOWED", nil
	}

	var stdout bytes.Buffer

	err := cli.ExportRunEvaluateWith(
		[]string{
			"--session-id", "sess-eval",
			"--transcript-path", transcriptPath,
			"--data-dir", dataDir,
		},
		&stdout,
		mockCaller,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Output should be JSON array with one result.
	output := stdout.String()
	g.Expect(output).To(ContainSubstring("wrap-errors"))
	g.Expect(output).To(ContainSubstring("FOLLOWED"))

	// The memory file should have followed_count incremented to 1.
	data, readErr := os.ReadFile(memoryPath)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(string(data)).To(ContainSubstring("followed_count = 1"))
}

func TestRun_Evaluate_Dispatch(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memoriesDir, 0o755)).To(Succeed())

	transcriptPath := filepath.Join(t.TempDir(), "t.jsonl")
	g.Expect(os.WriteFile(transcriptPath, []byte(""), 0o644)).To(Succeed())

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "evaluate",
			"--session-id", "sess-dispatch",
			"--transcript-path", transcriptPath,
			"--data-dir", dataDir,
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())
}
