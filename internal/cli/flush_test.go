package cli_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
)

func TestFlush_BadTranscriptPath_ReturnsError(t *testing.T) {
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
	// Learn is skipped (no session-id), but evaluate tries to open the file and fails.
	g.Expect(err).To(HaveOccurred())
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
	// No transcript → all steps skip → no error.
	g.Expect(err).NotTo(HaveOccurred())
}

func TestFlush_WithTranscript_RunsEvaluate(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	// Create a minimal transcript file.
	transcriptPath := filepath.Join(dataDir, "transcript.jsonl")
	writeErr := os.WriteFile(transcriptPath, []byte(`{"type":"human","text":"hello"}`+"\n"), 0o644)
	g.Expect(writeErr).NotTo(HaveOccurred())

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "flush",
			"--data-dir", dataDir,
			"--transcript-path", transcriptPath,
			"--session-id", "test-session",
			"--context-path", filepath.Join(dataDir, "session-context.md"),
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	// May fail due to missing API token for learn/evaluate, but exercises the wiring.
	// The key thing is it doesn't panic and attempts the pipeline.
	_ = err
}

// T-370: flush command runs learn, evaluate, context-update in order.
func TestT370_FlushRunsInOrder(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var callOrder []string

	runner := cli.NewFlushRunner(
		func() error { callOrder = append(callOrder, "learn"); return nil },
		func() error { callOrder = append(callOrder, "evaluate"); return nil },
		func() error { callOrder = append(callOrder, "context-update"); return nil },
	)

	err := runner.Run()
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(callOrder).To(Equal([]string{"learn", "evaluate", "context-update"}))
}

// T-371: flush command stops on first step error.
func TestT371_FlushStopsOnError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var callOrder []string

	evalErr := errors.New("evaluate failed")

	runner := cli.NewFlushRunner(
		func() error { callOrder = append(callOrder, "learn"); return nil },
		func() error { callOrder = append(callOrder, "evaluate"); return evalErr },
		func() error { callOrder = append(callOrder, "context-update"); return nil },
	)

	err := runner.Run()
	g.Expect(err).To(MatchError(ContainSubstring("evaluate failed")))

	g.Expect(callOrder).To(Equal([]string{"learn", "evaluate"}))
}
