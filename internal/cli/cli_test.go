package cli_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
	"engram/internal/evaluate"
	"engram/internal/extract"
	"engram/internal/learn"
)

// callAnthropicAPI error branches: invalid URL, bad JSON, empty content.
// Not parallel — sub-tests mutate the cli.AnthropicAPIURL global sequentially.
//
//nolint:paralleltest // subtests mutate cli.AnthropicAPIURL; cannot run in parallel
func TestCallAnthropicAPIErrorPaths(t *testing.T) {
	// setupEvalDir creates a tmpdir with a memory TOML and surfacing log.
	setupEvalDir := func(tb testing.TB) string {
		tb.Helper()

		dir := tb.TempDir()
		memDir := filepath.Join(dir, "memories")

		if mkErr := os.MkdirAll(memDir, 0o750); mkErr != nil {
			tb.Fatalf("MkdirAll: %v", mkErr)
		}

		memPath := filepath.Join(memDir, "m.toml")

		if wErr := os.WriteFile(
			memPath,
			[]byte("title=\"T\"\nprinciple=\"P\"\n"),
			0o640,
		); wErr != nil {
			tb.Fatalf("WriteFile memory: %v", wErr)
		}

		logLine := fmt.Sprintf(
			`{"memory_path":%q,"mode":"session-start","surfaced_at":"2024-01-01T00:00:00Z"}`,
			memPath,
		)

		if wErr := os.WriteFile(
			filepath.Join(dir, "surfacing-log.jsonl"),
			[]byte(logLine+"\n"),
			0o640,
		); wErr != nil {
			tb.Fatalf("WriteFile surfacing log: %v", wErr)
		}

		return dir
	}

	runEval := func(tb testing.TB, dataDir, apiURL string) error {
		tb.Helper()

		original := cli.AnthropicAPIURL
		cli.AnthropicAPIURL = apiURL

		defer func() { cli.AnthropicAPIURL = original }()

		var stdout, stderr bytes.Buffer

		return cli.RunEvaluate(
			[]string{"--data-dir", dataDir},
			"fake-token",
			&stdout, &stderr,
			strings.NewReader("transcript"),
		)
	}

	t.Run(
		"invalid URL returns error",
		func(t *testing.T) { //nolint:paralleltest // shares AnthropicAPIURL
			g := NewGomegaWithT(t)
			dataDir := setupEvalDir(t)

			err := runEval(t, dataDir, "://invalid-url")
			g.Expect(err).To(HaveOccurred())
			g.Expect(err.Error()).To(ContainSubstring("creating request"))
		},
	)

	t.Run(
		"bad JSON response returns error",
		func(t *testing.T) { //nolint:paralleltest // shares AnthropicAPIURL
			g := NewGomegaWithT(t)
			dataDir := setupEvalDir(t)

			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					_, _ = w.Write([]byte("not-json"))
				}),
			)
			defer server.Close()

			err := runEval(t, dataDir, server.URL)
			g.Expect(err).To(HaveOccurred())
			g.Expect(err.Error()).To(ContainSubstring("parsing API response"))
		},
	)

	t.Run(
		"empty content block returns error",
		func(t *testing.T) { //nolint:paralleltest // shares AnthropicAPIURL
			g := NewGomegaWithT(t)
			dataDir := setupEvalDir(t)

			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"content":[]}`))
				}),
			)
			defer server.Close()

			err := runEval(t, dataDir, server.URL)
			g.Expect(err).To(HaveOccurred())
			g.Expect(err.Error()).To(ContainSubstring("no content blocks"))
		},
	)
}

// Incremental learn path: --transcript-path + --session-id reads delta from file.
func TestLearnIncrementalPath(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	transcriptPath := filepath.Join(dataDir, "transcript.jsonl")

	// Write a transcript file with some content.
	err := os.WriteFile(
		transcriptPath,
		[]byte(`{"role":"user","content":"hello"}`+"\n"),
		0o644,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Fake HTTP doer returns empty learnings.
	fakeDoer := &fakeHTTPDoer{
		statusCode: http.StatusOK,
		body:       `{"content":[{"type":"text","text":"[]"}]}`,
	}

	var stderr bytes.Buffer

	runErr := cli.RunLearn(
		[]string{
			"--data-dir", dataDir,
			"--transcript-path", transcriptPath,
			"--session-id", "test-session",
		},
		"fake-token",
		&stderr,
		strings.NewReader(""),
		fakeDoer,
	)
	g.Expect(runErr).NotTo(HaveOccurred())

	if runErr != nil {
		return
	}

	g.Expect(stderr.String()).To(ContainSubstring("[engram]"))

	// Verify offset file was created.
	offsetPath := filepath.Join(dataDir, "learn-offset.json")
	_, statErr := os.Stat(offsetPath)
	g.Expect(statErr).NotTo(HaveOccurred())

	// Append more content so second run has new delta to process.
	appendFile, appendErr := os.OpenFile(
		transcriptPath, os.O_APPEND|os.O_WRONLY, 0o644,
	)
	g.Expect(appendErr).NotTo(HaveOccurred())

	if appendErr != nil {
		return
	}

	_, _ = appendFile.WriteString(`{"role":"user","content":"world"}` + "\n")
	_ = appendFile.Close()

	// Second run: offset file exists, exercises osOffsetStore.Read + Write.
	var stderr2 bytes.Buffer

	runErr = cli.RunLearn(
		[]string{
			"--data-dir", dataDir,
			"--transcript-path", transcriptPath,
			"--session-id", "test-session",
		},
		"fake-token",
		&stderr2,
		strings.NewReader(""),
		fakeDoer,
	)
	g.Expect(runErr).NotTo(HaveOccurred())
}

// learn with invalid flag returns error.
func TestLearnInvalidFlag(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run([]string{
		"engram", "learn",
		"--unknown-flag",
	}, &stdout, &stderr, strings.NewReader(""))

	g.Expect(err).To(HaveOccurred())
}

// learn with missing --data-dir returns error.
func TestLearnMissingDataDir(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run([]string{
		"engram", "learn",
	}, &stdout, &stderr, strings.NewReader(""))

	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("--data-dir"))
	}
}

// learn with a stdin reader that returns an error covers the reading-stdin error path.
func TestLearnStdinReadError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()

	var stderr bytes.Buffer

	err := cli.RunLearn(
		[]string{"--data-dir", dataDir},
		"fake-token",
		&stderr,
		&errReader{err: errors.New("disk full")},
		nil,
	)
	g.Expect(err).To(MatchError(ContainSubstring("reading stdin")))
}

// learn with token and fake HTTP doer returning empty learnings covers success path.
func TestLearnSuccessPath(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()

	var stderr bytes.Buffer

	// Fake HTTP doer returns a valid Anthropic response with no learnings.
	fakeDoer := &fakeHTTPDoer{
		statusCode: http.StatusOK,
		body:       `{"content":[{"type":"text","text":"[]"}]}`,
	}

	err := cli.RunLearn(
		[]string{"--data-dir", dataDir},
		"fake-token",
		&stderr,
		strings.NewReader("some transcript"),
		fakeDoer,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stderr.String()).To(ContainSubstring("[engram] No new learnings extracted."))
}

// TestRenderLearnResult_WithLearningsNoTierCounts verifies output when TierCounts is nil.
func TestRenderLearnResult_WithLearningsNoTierCounts(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var buf bytes.Buffer

	result := &learn.Result{
		CreatedPaths: []string{"/data/test.toml"},
		TierCounts:   map[string]int{"A": 0},
	}

	cli.RenderLearnResult(&buf, result)

	output := buf.String()
	g.Expect(output).To(ContainSubstring("[engram] Extracted 1 learnings"))
	g.Expect(output).To(ContainSubstring(`"test.toml"`))
}

// TestReviewDispatchedViaRun verifies the "review" subcommand is wired in cli.Run.
func TestReviewDispatchedViaRun(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dataDir := t.TempDir()

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "review", "--data-dir", dataDir},
		&stdout,
		&stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring("[engram] No evaluation data found."))
}

// T-158: context-update subcommand returns error when flags are missing.
func TestRunContextUpdate_MissingFlags(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "context-update"},
		&stdout, &stderr, strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("--transcript-path"))
	}
}

// T-159: context-update runs orchestrator with valid flags and temp files.
func TestRunContextUpdate_ValidFlags(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	tmpDir := t.TempDir()

	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")
	err := os.WriteFile(
		transcriptPath, []byte("line one\nline two\n"), 0o644,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var stdout, stderr bytes.Buffer

	// No API token → nil summarizer → orchestrator returns nil
	// (fire-and-forget, writes context file with empty summary).
	runErr := cli.Run(
		[]string{
			"engram", "context-update",
			"--transcript-path", transcriptPath,
			"--session-id", "test-session-123",
			"--data-dir", tmpDir,
		},
		&stdout, &stderr, strings.NewReader(""),
	)
	g.Expect(runErr).NotTo(HaveOccurred())
}

// T-160: context-update with API token exercises haikuClientAdapter.Summarize.
//

func TestRunContextUpdate_WithAPIToken(t *testing.T) {
	g := NewGomegaWithT(t)

	tmpDir := t.TempDir()

	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")
	err := os.WriteFile(
		transcriptPath, []byte("user asked about testing\n"), 0o644,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(
				`{"content":[{"type":"text","text":"Working on testing improvements."}]}`,
			))
		},
	))
	defer server.Close()

	original := cli.AnthropicAPIURL
	cli.AnthropicAPIURL = server.URL

	defer func() { cli.AnthropicAPIURL = original }()

	t.Setenv("ENGRAM_API_TOKEN", "fake-token")

	var stdout, stderr bytes.Buffer

	runErr := cli.Run(
		[]string{
			"engram", "context-update",
			"--transcript-path", transcriptPath,
			"--session-id", "test-session-456",
			"--data-dir", tmpDir,
		},
		&stdout, &stderr, strings.NewReader(""),
	)
	g.Expect(runErr).NotTo(HaveOccurred())

	// Verify context file was written with the summarized content.
	contextFile := filepath.Join(tmpDir, "session-context.md")
	content, readErr := os.ReadFile(contextFile)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(string(content)).To(ContainSubstring("Working on testing"))

	// Second call with same session ID exercises previousSummary branch
	// (context file already exists with a summary from the first call).
	err = os.WriteFile(
		transcriptPath, []byte("more work on testing\n"), 0o644,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	runErr = cli.Run(
		[]string{
			"engram", "context-update",
			"--transcript-path", transcriptPath,
			"--session-id", "test-session-456",
			"--data-dir", tmpDir,
		},
		&stdout, &stderr, strings.NewReader(""),
	)
	g.Expect(runErr).NotTo(HaveOccurred())
}

// runEvaluate covered via cli.Run with empty token (no-token path).
func TestRunEvaluateNoToken(t *testing.T) {
	// Cannot use t.Parallel() — t.Setenv mutates process environment.
	g := NewGomegaWithT(t)
	t.Setenv("ENGRAM_API_TOKEN", "")

	dataDir := t.TempDir()

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "evaluate", "--data-dir", dataDir},
		&stdout, &stderr,
		strings.NewReader("some transcript"),
	)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stderr.String()).To(ContainSubstring("no API token configured"))
}

func TestRun_CorrectMissingFlags(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run([]string{"engram", "correct"}, &stdout, &stderr, strings.NewReader(""))
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("--message"))
	}
}

func TestRun_NoArgs(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run([]string{"engram"}, &stdout, &stderr, strings.NewReader(""))
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("usage"))
	}
}

func TestRun_UnknownCommand(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run([]string{"engram", "bogus"}, &stdout, &stderr, strings.NewReader(""))
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("unknown command"))
	}
}

// effectivenessAdapter.Aggregate loop covered by surface call with pre-populated evaluations dir.
func TestSurfaceWithEffectivenessData(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memDir, 0o750)).To(Succeed())

	writeTestTOML(t, memDir, "mem.toml", `
title = "Use table-driven tests"
tier = "A"
keywords = ["test"]
principle = "Use table-driven tests"
`)

	evalDir := filepath.Join(dataDir, "evaluations")
	g.Expect(os.MkdirAll(evalDir, 0o750)).To(Succeed())

	memPath := filepath.Join(memDir, "mem.toml")
	evalLine := fmt.Sprintf(
		`{"memory_path":%q,"outcome":"followed","evidence":"used it","evaluated_at":"2024-01-01T00:00:00Z"}`,
		memPath,
	)
	g.Expect(os.WriteFile(
		filepath.Join(evalDir, "session.jsonl"),
		[]byte(evalLine+"\n"),
		0o640,
	)).To(Succeed())

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "surface", "--mode", "session-start", "--data-dir", dataDir},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())
}

// T-117: evaluate subcommand runs full pipeline.
func TestT117_EvaluateRunsFullPipeline(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()

	// Write a memory TOML file.
	memPath := filepath.Join(dataDir, "test-memory.toml")
	err := os.WriteFile(memPath, []byte(`title = "Test Memory"
content = "Some content"
principle = "Do the right thing"
anti_pattern = ""`), 0o644)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Write a surfacing log referencing the memory.
	logLine := fmt.Sprintf(
		`{"memory_path":%q,"mode":"session-start","surfaced_at":"2025-01-01T00:00:00Z"}`,
		memPath,
	)
	surfacingLog := filepath.Join(dataDir, "surfacing-log.jsonl")
	err = os.WriteFile(surfacingLog, []byte(logLine+"\n"), 0o644)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Fake LLM returns a valid outcome for the memory.
	fakeLLM := func(_ context.Context, _, _, _ string) (string, error) {
		return fmt.Sprintf(
			`[{"memory_path":%q,"outcome":"followed","evidence":"The agent complied."}]`,
			memPath,
		), nil
	}

	var stdout, stderr bytes.Buffer

	err = cli.RunEvaluate(
		[]string{"--data-dir", dataDir},
		"fake-token",
		&stdout, &stderr,
		strings.NewReader("session transcript content"),
		evaluate.WithLLMCaller(fakeLLM),
	)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring("[engram] Evaluated 1 memories"))
	g.Expect(stdout.String()).To(ContainSubstring("1 followed"))

	// Verify evaluation log was written to the evaluations dir.
	evalDir := filepath.Join(dataDir, "evaluations")
	entries, readErr := os.ReadDir(evalDir)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(entries).To(HaveLen(1))
}

// T-118: evaluate without API token emits error and exits 0.
func TestT118_EvaluateWithoutTokenEmitsError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()

	var stdout, stderr bytes.Buffer

	err := cli.RunEvaluate(
		[]string{"--data-dir", dataDir},
		"", // empty token
		&stdout, &stderr,
		strings.NewReader("some transcript"),
	)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stderr.String()).
		To(ContainSubstring("[engram] Error: evaluation skipped — no API token configured"))
	g.Expect(stdout.String()).To(BeEmpty())
}

// T-119: evaluate summary output format.
func TestT119_EvaluateSummaryFormat(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// Empty outcomes: no output written (covers early-return branch).
	var emptyBuf bytes.Buffer
	cli.RenderEvaluateResult(&emptyBuf, nil)
	g.Expect(emptyBuf.String()).To(BeEmpty())

	// All three outcome types covered.
	outcomes := []evaluate.Outcome{
		{MemoryPath: "a.toml", Outcome: "followed"},
		{MemoryPath: "b.toml", Outcome: "followed"},
		{MemoryPath: "c.toml", Outcome: "contradicted"},
		{MemoryPath: "d.toml", Outcome: "ignored"},
	}

	var buf bytes.Buffer

	cli.RenderEvaluateResult(&buf, outcomes)

	g.Expect(buf.String()).To(Equal(
		"[engram] Evaluated 4 memories: 2 followed, 1 contradicted, 1 ignored.\n",
	))
}

// T-120: Hook scripts invoke engram evaluate after learn.
func TestT120_HookScriptsInvokeEvaluate(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	for _, scriptPath := range []string{
		"../../hooks/pre-compact.sh",
	} {
		data, err := os.ReadFile(scriptPath)
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		content := string(data)
		// Script uses $ENGRAM_BIN variable; check for the subcommand and flag.
		g.Expect(content).To(ContainSubstring("evaluate --data-dir"))
		g.Expect(content).To(ContainSubstring("ENGRAM_DATA"))
	}
}

// T-121: callAnthropicAPI covered via httptest server (not parallel — mutates AnthropicAPIURL global).
//
//nolint:paralleltest // mutates cli.AnthropicAPIURL global
func TestT121_CallAnthropicAPICoverage(t *testing.T) {
	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memDir, 0o750)).To(Succeed())

	memPath := filepath.Join(memDir, "mem.toml")
	writeTestTOML(t, memDir, "mem.toml", `
title = "Use table-driven tests"
principle = "Use table-driven tests"
`)

	// LLM text response must be a JSON array matching memory_path.
	llmText := fmt.Sprintf(
		`[{"memory_path":%q,"outcome":"followed","evidence":"used it"}]`,
		memPath,
	)
	apiResp := fmt.Sprintf(`{"content":[{"type":"text","text":%q}]}`, llmText)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(apiResp))
	}))
	defer server.Close()

	original := cli.AnthropicAPIURL
	cli.AnthropicAPIURL = server.URL

	defer func() { cli.AnthropicAPIURL = original }()

	// surfacing-log.jsonl lives directly in dataDir (not a subdirectory).
	surfLogEntry := fmt.Sprintf(
		`{"memory_path":%q,"mode":"session-start","surfaced_at":"2024-01-01T00:00:00Z"}`,
		memPath,
	)
	g.Expect(os.WriteFile(
		filepath.Join(dataDir, "surfacing-log.jsonl"),
		[]byte(surfLogEntry+"\n"),
		0o640,
	)).To(Succeed())

	transcript := "I used table-driven tests in my implementation."

	var stdout, stderr bytes.Buffer

	err := cli.RunEvaluate(
		[]string{"--data-dir", dataDir},
		"fake-token",
		&stdout, &stderr,
		strings.NewReader(transcript),
	)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(ContainSubstring("followed"))
}

// T-129: Review with data outputs all four DES-16 sections.
func TestT129_ReviewOutputsAllSections(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")
	evalDir := filepath.Join(dataDir, "evaluations")

	g.Expect(os.MkdirAll(memDir, 0o750)).To(Succeed())
	g.Expect(os.MkdirAll(evalDir, 0o750)).To(Succeed())

	// mem-a: surfaced=10, 4 followed + 1 ignored = 80% effectiveness, 5 evals → Working.
	// mem-b: surfaced=2, 1 followed + 4 ignored = 20% effectiveness, 5 evals → Noise + flagged.
	// mem-c: surfaced=5, no evals → InsufficientData.
	// Median of tracking [2,5,10] = 5. mem-a (10) above median, mem-b (2) below.
	memA := writeReviewMemoryTOML(t, memDir, "mem-a.toml", 10)
	memB := writeReviewMemoryTOML(t, memDir, "mem-b.toml", 2)
	writeReviewMemoryTOML(t, memDir, "mem-c.toml", 5)

	writeReviewEvalLog(t, evalDir, "a.jsonl", memA,
		[]string{"followed", "followed", "followed", "followed", "ignored"})
	writeReviewEvalLog(t, evalDir, "b.jsonl", memB,
		[]string{"followed", "ignored", "ignored", "ignored", "ignored"})

	var stdout bytes.Buffer

	err := cli.RunReview([]string{"--data-dir", dataDir}, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := stdout.String()
	g.Expect(output).To(ContainSubstring("[engram] Memory Effectiveness Review"))
	g.Expect(output).To(ContainSubstring("Total: 3 memories"))
	g.Expect(output).To(ContainSubstring("Quadrant Summary:"))
	g.Expect(output).To(ContainSubstring("Flagged for action"))
	g.Expect(output).To(ContainSubstring("Insufficient data"))
}

// T-130: Review with no evaluations directory outputs no-data message.
func TestT130_ReviewNoEvalDir(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dataDir := t.TempDir() // no evaluations subdir

	var stdout bytes.Buffer

	err := cli.RunReview([]string{"--data-dir", dataDir}, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring("[engram] No evaluation data found."))
}

// T-131: Review without --data-dir outputs usage error.
func TestT131_ReviewMissingDataDir(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var stdout bytes.Buffer

	err := cli.RunReview([]string{}, &stdout)
	g.Expect(err).To(HaveOccurred())

	if err == nil {
		return
	}

	g.Expect(err.Error()).To(ContainSubstring("data-dir"))
}

// T-132: Flagged memories sorted by effectiveness ascending (worst first).
func TestT132_ReviewFlaggedSortedByEffectiveness(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")
	evalDir := filepath.Join(dataDir, "evaluations")

	g.Expect(os.MkdirAll(memDir, 0o750)).To(Succeed())
	g.Expect(os.MkdirAll(evalDir, 0o750)).To(Succeed())

	// All have surfaced=5, median=5, none > 5 → all below/equal median → Noise + flagged.
	// Effectiveness: mem-x=10% (1/10), mem-y=20% (1/5), mem-z=33% (2/6).
	memX := writeReviewMemoryTOML(t, memDir, "mem-x.toml", 5)
	memY := writeReviewMemoryTOML(t, memDir, "mem-y.toml", 5)
	memZ := writeReviewMemoryTOML(t, memDir, "mem-z.toml", 5)

	writeReviewEvalLog(t, evalDir, "x.jsonl", memX,
		[]string{
			"followed", "ignored", "ignored", "ignored", "ignored",
			"ignored", "ignored", "ignored", "ignored", "ignored",
		}) // 1/10 = 10%
	writeReviewEvalLog(t, evalDir, "y.jsonl", memY,
		[]string{"followed", "ignored", "ignored", "ignored", "ignored"}) // 1/5 = 20%
	writeReviewEvalLog(t, evalDir, "z.jsonl", memZ,
		[]string{"followed", "followed", "ignored", "ignored", "ignored", "ignored"}) // 2/6 = 33%

	var stdout bytes.Buffer

	err := cli.RunReview([]string{"--data-dir", dataDir}, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := stdout.String()
	posX := strings.Index(output, "mem-x")
	posY := strings.Index(output, "mem-y")
	posZ := strings.Index(output, "mem-z")

	g.Expect(posX).To(BeNumerically("<", posY), "mem-x (10%%) should appear before mem-y (20%%)")
	g.Expect(posY).To(BeNumerically("<", posZ), "mem-y (20%%) should appear before mem-z (33%%)")
}

// T-133: Insufficient-data section omitted when all memories have 5+ evaluations.
func TestT133_ReviewOmitsInsufficientDataSection(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")
	evalDir := filepath.Join(dataDir, "evaluations")

	g.Expect(os.MkdirAll(memDir, 0o750)).To(Succeed())
	g.Expect(os.MkdirAll(evalDir, 0o750)).To(Succeed())

	memA := writeReviewMemoryTOML(t, memDir, "mem-a.toml", 10)
	memB := writeReviewMemoryTOML(t, memDir, "mem-b.toml", 2)
	writeReviewEvalLog(t, evalDir, "a.jsonl", memA,
		[]string{"followed", "followed", "followed", "followed", "followed"}) // 100%
	writeReviewEvalLog(t, evalDir, "b.jsonl", memB,
		[]string{"followed", "followed", "followed", "followed", "followed"}) // 100%

	var stdout bytes.Buffer

	err := cli.RunReview([]string{"--data-dir", dataDir}, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).NotTo(ContainSubstring("Insufficient data"))
}

// T-161: evaluate applies Strip preprocessing to transcript before LLM call.
func TestT161_EvaluateStripsTranscript(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()

	// Write a memory TOML file.
	memPath := filepath.Join(dataDir, "strip-test.toml")
	err := os.WriteFile(memPath, []byte(`title = "Strip Test"
content = "Test content"
principle = "Always strip"
anti_pattern = ""`), 0o644)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Write a surfacing log referencing the memory.
	logLine := fmt.Sprintf(
		`{"memory_path":%q,"mode":"session-start","surfaced_at":"2025-01-01T00:00:00Z"}`,
		memPath,
	)
	surfacingLog := filepath.Join(dataDir, "surfacing-log.jsonl")
	err = os.WriteFile(surfacingLog, []byte(logLine+"\n"), 0o644)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Transcript with normal lines AND a toolResult line that Strip removes.
	transcript := strings.Join([]string{
		`{"role":"user","content":"please help me"}`,
		`{"role":"toolResult","content":[{"type":"text","text":"huge tool output that should be stripped"}]}`,
		`{"role":"assistant","content":"sure, I can help"}`,
	}, "\n")

	// Fake LLM that captures the user prompt it receives.
	var capturedPrompt string

	fakeLLM := func(_ context.Context, _, _, userPrompt string) (string, error) {
		capturedPrompt = userPrompt

		return fmt.Sprintf(
			`[{"memory_path":%q,"outcome":"followed","evidence":"Complied."}]`,
			memPath,
		), nil
	}

	var stdout, stderr bytes.Buffer

	err = cli.RunEvaluate(
		[]string{"--data-dir", dataDir},
		"fake-token",
		&stdout, &stderr,
		strings.NewReader(transcript),
		evaluate.WithLLMCaller(fakeLLM),
	)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// The toolResult line should have been stripped before reaching the LLM.
	g.Expect(capturedPrompt).To(ContainSubstring("please help me"))
	g.Expect(capturedPrompt).To(ContainSubstring("sure, I can help"))
	g.Expect(capturedPrompt).NotTo(ContainSubstring("huge tool output that should be stripped"))
}

// T-18: correct subcommand with no API key returns error
func TestT18_CorrectSubcommandWithoutAPIKeyReturnsError(t *testing.T) {
	// Cannot use t.Parallel() — t.Setenv mutates process environment.
	g := NewGomegaWithT(t)

	dataDir := filepath.Join(t.TempDir(), "data")

	t.Setenv("ENGRAM_API_TOKEN", "")

	var stdout, stderr bytes.Buffer

	err := cli.Run([]string{
		"engram", "correct",
		"--message", "remember to use targ",
		"--data-dir", dataDir,
	}, &stdout, &stderr, strings.NewReader(""))
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("no API token"))
	}
}

// T-19: correct with non-matching message produces empty stdout
func TestT19_CorrectWithNonMatchingMessageProducesEmptyStdout(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := filepath.Join(t.TempDir(), "data")

	var stdout, stderr bytes.Buffer

	err := cli.Run([]string{
		"engram", "correct",
		"--message", "hello world",
		"--data-dir", dataDir,
	}, &stdout, &stderr, strings.NewReader(""))
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(stdout.String()).To(BeEmpty())

	memoriesDir := filepath.Join(dataDir, "memories")
	_, statErr := os.Stat(memoriesDir)
	g.Expect(os.IsNotExist(statErr)).To(BeTrue())
}

// T-40: Mode session-start routes to SessionStart surfacing
func TestT40_SurfaceSessionStartRouting(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	err := os.MkdirAll(memoriesDir, 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	writeTestTOML(t, memoriesDir, "test-memory.toml", `title = "Test Memory"
content = "test"
observation_type = "correction"
concepts = []
keywords = ["test"]
principle = "test principle"
anti_pattern = ""
rationale = ""
confidence = "B"
created_at = "2025-01-01T00:00:00Z"
updated_at = "2025-01-01T00:00:00Z"
`)

	var stdout, stderr bytes.Buffer

	err = cli.Run([]string{
		"engram", "surface",
		"--mode", "session-start",
		"--data-dir", dataDir,
	}, &stdout, &stderr, strings.NewReader(""))

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := stdout.String()
	g.Expect(output).To(ContainSubstring("[engram] Loaded 1 memories."))
	g.Expect(output).To(ContainSubstring("test-memory"))
}

// T-41: Mode prompt routes to keyword surfacing
func TestT41_SurfacePromptRouting(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	err := os.MkdirAll(memoriesDir, 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	writeTestTOML(t, memoriesDir, "commit-rules.toml", `title = "Commit Rules"
content = "use /commit"
observation_type = "correction"
concepts = []
keywords = ["commit"]
principle = "use /commit for commits"
anti_pattern = ""
rationale = ""
confidence = "B"
created_at = "2025-01-01T00:00:00Z"
updated_at = "2025-01-01T00:00:00Z"
`)

	var stdout, stderr bytes.Buffer

	err = cli.Run([]string{
		"engram", "surface",
		"--mode", "prompt",
		"--message", "I want to commit this",
		"--data-dir", dataDir,
	}, &stdout, &stderr, strings.NewReader(""))

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := stdout.String()
	g.Expect(output).To(ContainSubstring("[engram] Relevant memories:"))
	g.Expect(output).To(ContainSubstring("commit-rules (matched: commit)"))
}

// T-42: Mode tool routes to advisory surfacing (not blocking enforcement)
func TestT42_SurfaceToolRouting(t *testing.T) {
	// Cannot use t.Parallel() — t.Setenv mutates process environment.
	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	err := os.MkdirAll(memoriesDir, 0o750)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	writeTestTOML(t, memoriesDir, "commit-rules.toml", `title = "Commit Rules"
content = "use /commit"
observation_type = "correction"
concepts = []
keywords = ["commit"]
principle = "use /commit for commits"
anti_pattern = "manual git commit"
rationale = ""
confidence = "B"
created_at = "2025-01-01T00:00:00Z"
updated_at = "2025-01-01T00:00:00Z"
`)

	t.Setenv("ENGRAM_API_TOKEN", "")

	var stdout, stderr bytes.Buffer

	err = cli.Run([]string{
		"engram", "surface",
		"--mode", "tool",
		"--tool-name", "Bash",
		"--tool-input", `{"command": "git commit -m fix"}`,
		"--data-dir", dataDir,
	}, &stdout, &stderr, strings.NewReader(""))

	g.Expect(err).NotTo(HaveOccurred())
	// Tool mode should emit advisory (now non-empty).
	output := stdout.String()
	g.Expect(output).To(ContainSubstring("[engram] Tool call advisory:"))
}

// T-61: RenderLearnResult with learnings emits DES-10 format with tier breakdown.
func TestT61_RenderLearnResult_WithLearnings(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var buf bytes.Buffer

	result := &learn.Result{
		CreatedPaths: []string{
			"/data/use-targ-for-builds.toml",
			"/data/di-everywhere.toml",
		},
		SkippedCount: 0,
		TierCounts:   map[string]int{"A": 1, "B": 1},
	}

	cli.RenderLearnResult(&buf, result)

	output := buf.String()
	g.Expect(output).To(ContainSubstring("[engram] Extracted 2 learnings from session."))
	g.Expect(output).To(ContainSubstring("A: 1"))
	g.Expect(output).To(ContainSubstring("B: 1"))
	g.Expect(output).To(ContainSubstring(`"use-targ-for-builds.toml"`))
	g.Expect(output).To(ContainSubstring(`"di-everywhere.toml"`))
	g.Expect(output).NotTo(ContainSubstring("Skipped"))
}

// T-62: learn without token emits error to stderr, returns nil (exit 0).
func TestT62_LearnWithoutTokenEmitsErrorToStderr(t *testing.T) {
	// Cannot use t.Parallel() — t.Setenv mutates process environment.
	g := NewGomegaWithT(t)

	t.Setenv("ENGRAM_API_TOKEN", "")

	dataDir := t.TempDir()

	var stdout, stderr bytes.Buffer

	err := cli.Run([]string{
		"engram", "learn",
		"--data-dir", dataDir,
	}, &stdout, &stderr, strings.NewReader("some transcript"))

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stderr.String()).To(ContainSubstring("session learning skipped"))
	g.Expect(stderr.String()).To(ContainSubstring("no API token"))
}

// T-63: RenderLearnResult with learnings and duplicates emits full DES-10 format with tier breakdown.
func TestT63_RenderLearnResult_WithLearningsAndSkipped(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var buf bytes.Buffer

	result := &learn.Result{
		CreatedPaths: []string{
			"/data/use-targ-for-builds.toml",
		},
		SkippedCount: 3,
		TierCounts:   map[string]int{"C": 1},
	}

	cli.RenderLearnResult(&buf, result)

	output := buf.String()
	g.Expect(output).To(ContainSubstring("[engram] Extracted 1 learnings from session."))
	g.Expect(output).To(ContainSubstring("C: 1"))
	g.Expect(output).To(ContainSubstring(`"use-targ-for-builds.toml"`))
	g.Expect(output).To(ContainSubstring("[engram] Skipped 3 duplicates."))
}

// T-64: RenderLearnResult with zero learnings emits DES-10 empty format.
func TestT64_RenderLearnResult_NoLearnings(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var buf bytes.Buffer

	result := &learn.Result{}

	cli.RenderLearnResult(&buf, result)

	g.Expect(buf.String()).To(Equal("[engram] No new learnings extracted.\n"))
}

// unexported variables.
var (
	_ extract.HTTPDoer = (*fakeHTTPDoer)(nil)
)

// errReader is an io.Reader that always returns an error.
type errReader struct {
	err error
}

func (e *errReader) Read(_ []byte) (int, error) {
	return 0, e.err
}

// fakeHTTPDoer implements extract.HTTPDoer for testing without real HTTP calls.
type fakeHTTPDoer struct {
	statusCode int
	body       string
}

func (f *fakeHTTPDoer) Do(_ *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: f.statusCode,
		Body:       io.NopCloser(strings.NewReader(f.body)),
	}, nil
}

// writeReviewEvalLog writes evaluation log entries for a memory to a .jsonl file.
func writeReviewEvalLog(tb testing.TB, evalDir, filename, memPath string, outcomes []string) {
	tb.Helper()

	lines := make([]string, 0, len(outcomes))
	for _, outcome := range outcomes {
		lines = append(lines, fmt.Sprintf(`{"memory_path":%q,"outcome":%q}`, memPath, outcome))
	}

	path := filepath.Join(evalDir, filename)
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o640); err != nil {
		tb.Fatalf("writeReviewEvalLog: %v", err)
	}
}

// writeReviewMemoryTOML creates a minimal memory TOML file and returns its full path.
func writeReviewMemoryTOML(tb testing.TB, dir, filename string, surfacedCount int) string {
	tb.Helper()

	path := filepath.Join(dir, filename)
	content := fmt.Sprintf(
		"title = %q\ncontent = \"Some content\"\nupdated_at = \"2024-01-01T00:00:00Z\"\nsurfaced_count = %d\n",
		filename,
		surfacedCount,
	)

	if err := os.WriteFile(path, []byte(content), 0o640); err != nil {
		tb.Fatalf("writeReviewMemoryTOML: %v", err)
	}

	return path
}

func writeTestTOML(t *testing.T, dir, filename, content string) {
	t.Helper()

	path := filepath.Join(dir, filename)

	err := os.WriteFile(path, []byte(content), 0o640)
	if err != nil {
		t.Fatalf("writeTestTOML: %v", err)
	}
}
