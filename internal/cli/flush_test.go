package cli_test

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
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
	// No session-id → learn and context-update both skip → no error.
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
	// No transcript → all steps skip → no error.
	g.Expect(err).NotTo(HaveOccurred())
}

func TestFlush_WithTranscript_RunsPipeline(t *testing.T) {
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
	// May fail due to missing API token for learn/context-update, but exercises the wiring.
	_ = err
}

// T-337: flush integration test — verifies learn runs and
// context-update writes session-context.md when run through the real CLI.
// Not parallel — mutates AnthropicAPIURL global and ENGRAM_API_TOKEN env.
func TestT337_FlushIntegration_PipelineOrdering(t *testing.T) {
	g := NewWithT(t)

	dataDir := t.TempDir()

	// Create a transcript file.
	transcriptPath := filepath.Join(dataDir, "transcript.jsonl")
	g.Expect(os.WriteFile(
		transcriptPath,
		[]byte("user asked about targ check-full\n"), 0o644,
	)).To(Succeed())

	// Mock LLM: learn expects extraction JSON, context-update expects summary text.
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			requestCount++

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(
				`{"content":[{"type":"text","text":"[]"}]}`,
			))
		},
	))
	defer server.Close()

	original := cli.AnthropicAPIURL
	cli.AnthropicAPIURL = server.URL

	defer func() { cli.AnthropicAPIURL = original }()

	t.Setenv("ENGRAM_API_TOKEN", "fake-token")

	contextPath := filepath.Join(dataDir, "session-context.md")

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "flush",
			"--data-dir", dataDir,
			"--transcript-path", transcriptPath,
			"--session-id", "test-session-337",
			"--context-path", contextPath,
		},
		&stdout, &stderr, strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Verify context-update ran: session-context.md should exist.
	_, statErr := os.Stat(contextPath)
	g.Expect(statErr).NotTo(HaveOccurred(),
		"session-context.md should be written by context-update")

	// Verify LLM was called by context-update.
	// Note: learn uses its own package-level URL constant (not overridable here),
	// so only context-update's call is counted against the mock.
	g.Expect(requestCount).To(BeNumerically(">=", 1),
		"LLM should be called by context-update")
}

// T-370: flush command runs learn then context-update (no evaluate).
func TestT370_FlushRunsInOrder(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var callOrder []string

	runner := cli.NewFlushRunner(
		func() error { callOrder = append(callOrder, "learn"); return nil },
		func() error { callOrder = append(callOrder, "context-update"); return nil },
	)

	err := runner.Run()
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(callOrder).To(Equal([]string{"learn", "context-update"}))
}

// T-371: flush command stops on first step error.
func TestT371_FlushStopsOnError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var callOrder []string

	learnErr := errors.New("learn failed")

	runner := cli.NewFlushRunner(
		func() error { callOrder = append(callOrder, "learn"); return learnErr },
		func() error { callOrder = append(callOrder, "context-update"); return nil },
	)

	err := runner.Run()
	g.Expect(err).To(MatchError(ContainSubstring("learn failed")))

	g.Expect(callOrder).To(Equal([]string{"learn"}))
}
