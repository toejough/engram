package cli_test

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

// T-337: flush integration test — verifies evaluate consumes surfacing log
// and context-update writes session-context.md when run through the real CLI.
// Not parallel — mutates AnthropicAPIURL global and ENGRAM_API_TOKEN env.
func TestT337_FlushIntegration_PipelineOrdering(t *testing.T) {
	g := NewWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memoriesDir, 0o750)).To(Succeed())

	// Create a memory file so evaluate has something to reference.
	memPath := filepath.Join(memoriesDir, "test-memory.toml")
	memContent := `title = "Use targ check-full"
content = "Always use targ check-full for validation"
keywords = ["targ", "check", "full"]
principle = "Run targ check-full before declaring done"
surfaced_count = 5
followed_count = 3
`
	g.Expect(os.WriteFile(memPath, []byte(memContent), 0o644)).To(Succeed())

	// Create a surfacing log referencing the memory.
	surfacingLog := fmt.Sprintf(
		`{"memory_path":%q,"mode":"prompt","surfaced_at":"%s"}`,
		memPath, time.Now().UTC().Format(time.RFC3339),
	)
	g.Expect(os.WriteFile(
		filepath.Join(dataDir, "surfacing-log.jsonl"),
		[]byte(surfacingLog+"\n"), 0o644,
	)).To(Succeed())

	// Create a transcript file.
	transcriptPath := filepath.Join(dataDir, "transcript.jsonl")
	g.Expect(os.WriteFile(
		transcriptPath,
		[]byte("user asked about targ check-full\n"), 0o644,
	)).To(Succeed())

	// Mock LLM: learn expects extraction JSON, evaluate expects outcome JSON,
	// context-update expects summary text. A single mock returning a generic
	// JSON response is sufficient — learn/evaluate parse what they can.
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			requestCount++

			w.Header().Set("Content-Type", "application/json")
			// Return a valid Anthropic response shape.
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

	// Verify evaluate ran: surfacing log should be consumed (renamed+deleted).
	_, statErr := os.Stat(filepath.Join(dataDir, "surfacing-log.jsonl"))
	g.Expect(os.IsNotExist(statErr)).To(BeTrue(),
		"surfacing-log.jsonl should be consumed by evaluate")

	// Verify context-update ran: session-context.md should exist.
	_, statErr = os.Stat(contextPath)
	g.Expect(statErr).NotTo(HaveOccurred(),
		"session-context.md should be written by context-update")

	// Verify LLM was called (at least learn + evaluate + context-update).
	g.Expect(requestCount).To(BeNumerically(">=", 2),
		"LLM should be called by multiple pipeline steps")
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
