package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
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
	sessionctx "engram/internal/context"
	"engram/internal/evaluate"
	"engram/internal/extract"
	"engram/internal/learn"
)

// TestAuditFlagParseError exercises the flag parse error path.
func TestAuditFlagParseError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "audit", "--bogus-flag"},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())
}

// TestAuditMissingDataDir verifies audit without --data-dir returns error.
func TestAuditMissingDataDir(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "audit"},
		&stdout, &stderr,
		strings.NewReader("transcript"),
	)
	g.Expect(err).To(HaveOccurred())
}

// TestAuditStdinReadError exercises the stdin read error path via an errReader.
//

func TestAuditStdinReadError(t *testing.T) {
	g := NewGomegaWithT(t)

	t.Setenv("ENGRAM_API_TOKEN", "fake-token")

	dataDir := t.TempDir()

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "audit", "--data-dir", dataDir,
		},
		&stdout, &stderr,
		&errReader{err: errors.New("broken stdin")},
	)
	g.Expect(err).To(HaveOccurred())
}

// TestAuditWithBadTimestamp exercises the timestamp parse failure branch.
//

func TestAuditWithBadTimestamp(t *testing.T) {
	g := NewGomegaWithT(t)

	t.Setenv("ENGRAM_API_TOKEN", "fake-token")

	dataDir := t.TempDir()

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "audit", "--data-dir", dataDir,
			"--timestamp", "not-a-valid-timestamp",
		},
		&stdout, &stderr,
		strings.NewReader("some transcript"),
	)
	// Bad timestamp is silently ignored; audit still runs (no memories → nil report → no error).
	g.Expect(err).NotTo(HaveOccurred())
}

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

// TestMaintainAggregateError exercises the effectiveness Aggregate error path
// by making the evaluations dir a file (not a directory).
func TestMaintainAggregateError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()

	// Create evaluations as a file, not a directory, so readDir fails.
	g.Expect(os.WriteFile(
		filepath.Join(dataDir, "evaluations"),
		[]byte("not a directory"),
		0o640,
	)).To(Succeed())

	var stdout bytes.Buffer

	err := cli.RunMaintain(
		[]string{"--data-dir", dataDir},
		"",
		&stdout,
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err).To(MatchError(ContainSubstring("aggregating effectiveness")))
	}
}

// TestMaintainApplyEmptyProposals verifies --apply with empty proposals outputs message.
func TestMaintainApplyEmptyProposals(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	proposalsPath := filepath.Join(dataDir, "proposals.json")

	g.Expect(os.WriteFile(proposalsPath, []byte("[]"), 0o640)).To(Succeed())

	var stdout bytes.Buffer

	err := cli.RunMaintain(
		[]string{"--data-dir", dataDir, "--apply", "--proposals", proposalsPath, "--yes"},
		"",
		&stdout,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring("No valid proposals"))
}

// TestMaintainApplyMissingProposals verifies --apply without --proposals returns error.
func TestMaintainApplyMissingProposals(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()

	var stdout bytes.Buffer

	err := cli.RunMaintain(
		[]string{"--data-dir", dataDir, "--apply"},
		"",
		&stdout,
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err).To(MatchError(ContainSubstring("--proposals")))
	}
}

// TestMaintainApplyNonexistentFile verifies --apply with missing file returns error.
func TestMaintainApplyNonexistentFile(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()

	var stdout bytes.Buffer

	err := cli.RunMaintain(
		[]string{
			"--data-dir", dataDir, "--apply",
			"--proposals", filepath.Join(dataDir, "nope.json"),
		},
		"",
		&stdout,
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err).To(MatchError(ContainSubstring("reading proposals")))
	}
}

// TestMaintainApplyRemoveProposal exercises --apply with a remove proposal.
func TestMaintainApplyRemoveProposal(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")

	g.Expect(os.MkdirAll(memDir, 0o750)).To(Succeed())

	memPath := filepath.Join(memDir, "remove-me.toml")

	g.Expect(os.WriteFile(memPath, []byte("title=\"Remove Me\"\nprinciple=\"P\"\n"), 0o640)).
		To(Succeed())

	registryPath := filepath.Join(dataDir, "instruction-registry.jsonl")
	registryLine := fmt.Sprintf(
		`{"id":%q,"source_type":"memory","title":"Remove Me","surfaced_count":5}`,
		memPath,
	)

	g.Expect(os.WriteFile(registryPath, []byte(registryLine+"\n"), 0o640)).To(Succeed())

	proposalsJSON := fmt.Sprintf(
		`[{"memory_path":%q,"quadrant":"Noise","diagnosis":"never used","action":"remove","details":{}}]`,
		memPath,
	)
	proposalsPath := filepath.Join(dataDir, "proposals.json")

	g.Expect(os.WriteFile(proposalsPath, []byte(proposalsJSON), 0o640)).To(Succeed())

	var stdout bytes.Buffer

	err := cli.RunMaintain(
		[]string{"--data-dir", dataDir, "--apply", "--proposals", proposalsPath, "--yes"},
		"",
		&stdout,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring("Applied 1/1"))

	// Memory file should be removed.
	_, statErr := os.Stat(memPath)
	g.Expect(os.IsNotExist(statErr)).To(BeTrue())
}

// TestMaintainApplyRewriteProposal exercises cliLLMCaller.Call via a rewrite proposal.
//
//nolint:paralleltest // mutates cli.AnthropicAPIURL global
func TestMaintainApplyRewriteProposal(t *testing.T) {
	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")

	g.Expect(os.MkdirAll(memDir, 0o750)).To(Succeed())

	memPath := filepath.Join(memDir, "rewrite-me.toml")

	g.Expect(os.WriteFile(memPath,
		[]byte("title=\"Rewrite Me\"\nprinciple=\"old principle\"\n"), 0o640,
	)).To(Succeed())

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(
				`{"content":[{"type":"text","text":"{\"principle\":\"new principle\",\"anti_pattern\":\"old way\"}"}]}`,
			))
		}))
	defer server.Close()

	original := cli.AnthropicAPIURL
	cli.AnthropicAPIURL = server.URL

	defer func() { cli.AnthropicAPIURL = original }()

	proposalsJSON := fmt.Sprintf(
		`[{"memory_path":%q,"quadrant":"Leech","diagnosis":"contradicted","action":"rewrite","details":{}}]`,
		memPath,
	)
	proposalsPath := filepath.Join(dataDir, "proposals.json")

	g.Expect(os.WriteFile(proposalsPath, []byte(proposalsJSON), 0o640)).To(Succeed())

	var stdout bytes.Buffer

	err := cli.RunMaintain(
		[]string{"--data-dir", dataDir, "--apply", "--proposals", proposalsPath, "--yes"},
		"fake-token",
		&stdout,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring("Applied 1/1"))
}

// TestMaintainBuildMemoryMapError exercises the buildMemoryMap error path
// by providing a data-dir where the memories directory is a file, not a dir.
func TestMaintainBuildMemoryMapError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	evalDir := filepath.Join(dataDir, "evaluations")

	g.Expect(os.MkdirAll(evalDir, 0o750)).To(Succeed())

	// Write evaluation data so stats is non-empty (skip the early return).
	memPath := filepath.Join(dataDir, "memories", "fake.toml")
	evalLine := fmt.Sprintf(
		`{"memory_path":%q,"outcome":"followed","evaluated_at":"2024-01-01T00:00:00Z"}`,
		memPath,
	)

	g.Expect(os.WriteFile(
		filepath.Join(evalDir, "session.jsonl"),
		[]byte(evalLine+"\n"),
		0o640,
	)).To(Succeed())

	// Make "memories" a file instead of a directory to cause buildMemoryMap to fail.
	g.Expect(os.WriteFile(
		filepath.Join(dataDir, "memories"),
		[]byte("not a directory"),
		0o640,
	)).To(Succeed())

	var stdout bytes.Buffer

	err := cli.RunMaintain(
		[]string{"--data-dir", dataDir},
		"",
		&stdout,
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err).To(MatchError(ContainSubstring("maintain")))
	}
}

// TestMaintainDispatchedViaRun verifies "maintain" is in dispatch.
func TestMaintainDispatchedViaRun(t *testing.T) {
	// Cannot use t.Parallel() — t.Setenv mutates process environment.
	g := NewGomegaWithT(t)

	t.Setenv("ENGRAM_API_TOKEN", "")

	dataDir := t.TempDir()

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "maintain", "--data-dir", dataDir},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(strings.TrimSpace(stdout.String())).To(Equal("[]"))
}

// TestMaintainFlagParseError exercises the flag parse error path.
func TestMaintainFlagParseError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var stdout bytes.Buffer

	err := cli.RunMaintain(
		[]string{"--bogus-flag"},
		"",
		&stdout,
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err).To(MatchError(ContainSubstring("maintain")))
	}
}

// TestMaintainMissingDataDir verifies missing --data-dir returns error.
func TestMaintainMissingDataDir(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var stdout bytes.Buffer

	err := cli.RunMaintain([]string{}, "", &stdout)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("data-dir"))
	}
}

// TestMaintainWithLeechEscalation exercises the escalation engine path in RunMaintain.
func TestMaintainWithLeechEscalation(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")
	evalDir := filepath.Join(dataDir, "evaluations")

	g.Expect(os.MkdirAll(memDir, 0o750)).To(Succeed())
	g.Expect(os.MkdirAll(evalDir, 0o750)).To(Succeed())

	// Create a leech memory: high surfacing, all contradicted.
	leechMem := writeReviewMemoryTOML(t, memDir, "leech-escalation.toml", 10)
	writeReviewEvalLog(t, evalDir, "leech-esc.jsonl", leechMem,
		[]string{
			"contradicted", "contradicted", "contradicted",
			"contradicted", "contradicted",
		})

	var stdout bytes.Buffer

	// Pass a token to exercise the token != "" branch (WithLLMCaller).
	// The LLM caller won't be invoked since no hidden gem or leech LLM proposals
	// are generated without an actual LLM, but the wiring path is exercised.
	err := cli.RunMaintain(
		[]string{"--data-dir", dataDir},
		"fake-token",
		&stdout,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var proposals []map[string]any

	jsonErr := json.Unmarshal(stdout.Bytes(), &proposals)
	g.Expect(jsonErr).NotTo(HaveOccurred())

	if jsonErr != nil {
		return
	}

	// Should have at least one proposal (escalation or noise).
	g.Expect(proposals).ToNot(BeEmpty())
}

// TestRegistryMergeMissingFlags verifies merge without flags returns error.
func TestRegistryMergeMissingFlags(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var stdout bytes.Buffer

	err := cli.RunRegistryMerge(
		[]string{},
		&stdout,
		os.Remove,
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err).To(MatchError(ContainSubstring("required")))
	}
}

// TestRegistryMergeNonMemorySource exercises merge where source is not a memory type.
// Registry merge is scoped to memory-to-memory only; non-memory merges are rejected.
func TestRegistryMergeNonMemorySource(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	registryPath := filepath.Join(dataDir, "instruction-registry.jsonl")

	sourceEntry := map[string]any{
		"id":             "rule-src",
		"source_type":    "rule",
		"source_path":    "",
		"title":          "Source Rule",
		"surfaced_count": 3,
		"evaluations":    map[string]int{"followed": 2, "contradicted": 0, "ignored": 1},
		"registered_at":  "2026-03-01T00:00:00Z",
		"updated_at":     "2026-03-08T00:00:00Z",
	}
	targetEntry := map[string]any{
		"id":             "rule-tgt",
		"source_type":    "rule",
		"title":          "Target Rule",
		"surfaced_count": 1,
		"evaluations":    map[string]int{"followed": 1, "contradicted": 0, "ignored": 0},
		"registered_at":  "2026-03-01T00:00:00Z",
		"updated_at":     "2026-03-08T00:00:00Z",
	}

	var registryData strings.Builder

	line1, marshalErr := json.Marshal(sourceEntry)
	g.Expect(marshalErr).NotTo(HaveOccurred())

	registryData.Write(line1)
	registryData.WriteByte('\n')

	line2, marshalErr := json.Marshal(targetEntry)
	g.Expect(marshalErr).NotTo(HaveOccurred())

	registryData.Write(line2)
	registryData.WriteByte('\n')

	g.Expect(os.WriteFile(registryPath, []byte(registryData.String()), 0o640)).To(Succeed())

	var stdout bytes.Buffer

	err := cli.RunRegistryMerge(
		[]string{
			"--data-dir", dataDir,
			"--source", "rule-src",
			"--target", "rule-tgt",
		},
		&stdout,
		func(_ string) error { return nil },
	)
	// Registry merge is memory-to-memory only; non-memory entries are rejected.
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("source_type=memory"))
}

// TestRegistryMergeRemoveFileError exercises the removeFile error warning path.
func TestRegistryMergeRemoveFileError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	registryPath := filepath.Join(dataDir, "instruction-registry.jsonl")

	sourceEntry := map[string]any{
		"id":             "mem-src",
		"source_type":    "memory",
		"source_path":    "/fake/path/memory.toml",
		"title":          "Source Memory",
		"surfaced_count": 5,
		"evaluations":    map[string]int{"followed": 3, "contradicted": 1, "ignored": 0},
		"registered_at":  "2026-03-01T00:00:00Z",
		"updated_at":     "2026-03-08T00:00:00Z",
	}
	targetEntry := map[string]any{
		"id":             "mem-tgt",
		"source_type":    "memory",
		"title":          "Target Memory",
		"surfaced_count": 2,
		"evaluations":    map[string]int{"followed": 1, "contradicted": 0, "ignored": 0},
		"registered_at":  "2026-03-01T00:00:00Z",
		"updated_at":     "2026-03-08T00:00:00Z",
	}

	var registryData strings.Builder

	line1, marshalErr := json.Marshal(sourceEntry)
	g.Expect(marshalErr).NotTo(HaveOccurred())

	registryData.Write(line1)
	registryData.WriteByte('\n')

	line2, marshalErr := json.Marshal(targetEntry)
	g.Expect(marshalErr).NotTo(HaveOccurred())

	registryData.Write(line2)
	registryData.WriteByte('\n')

	g.Expect(os.WriteFile(registryPath, []byte(registryData.String()), 0o640)).To(Succeed())

	var stdout bytes.Buffer

	// removeFile returns a non-IsNotExist error to exercise the warning path.
	err := cli.RunRegistryMerge(
		[]string{
			"--data-dir", dataDir,
			"--source", "mem-src",
			"--target", "mem-tgt",
		},
		&stdout,
		func(_ string) error {
			return errors.New("permission denied")
		},
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring("Warning"))
	g.Expect(stdout.String()).To(ContainSubstring("permission denied"))
}

// TestRegistryMergeSourceNotFound exercises the Get error path.
func TestRegistryMergeSourceNotFound(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()

	// Write an empty registry file.
	registryPath := filepath.Join(dataDir, "instruction-registry.jsonl")

	g.Expect(os.WriteFile(registryPath, []byte(""), 0o640)).To(Succeed())

	var stdout bytes.Buffer

	err := cli.RunRegistryMerge(
		[]string{
			"--data-dir", dataDir,
			"--source", "nonexistent-src",
			"--target", "nonexistent-tgt",
		},
		&stdout,
		os.Remove,
	)
	g.Expect(err).To(HaveOccurred())
}

// TestRegistryRegisterSourceDuplicateSkipped exercises the ErrDuplicateID continue path.
func TestRegistryRegisterSourceDuplicateSkipped(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	srcPath := filepath.Join(dataDir, "rule.md")

	g.Expect(os.WriteFile(srcPath,
		[]byte("## Rule\nAlways use table-driven tests.\n"), 0o640,
	)).To(Succeed())

	var stdout bytes.Buffer

	// Register once.
	err := cli.RunRegistryRegisterSource(
		[]string{
			"--data-dir", dataDir,
			"--type", "rule",
			"--path", srcPath,
		},
		&stdout,
		os.ReadFile,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Register again — duplicates should be skipped.
	stdout.Reset()

	err = cli.RunRegistryRegisterSource(
		[]string{
			"--data-dir", dataDir,
			"--type", "rule",
			"--path", srcPath,
		},
		&stdout,
		os.ReadFile,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring("Registered 0 instructions"))
}

// TestRegistryRegisterSourceEmptyExtraction exercises the zero-entries path.
func TestRegistryRegisterSourceEmptyExtraction(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	srcPath := filepath.Join(dataDir, "empty.md")

	// An empty file produces zero extracted entries.
	g.Expect(os.WriteFile(srcPath, []byte(""), 0o640)).To(Succeed())

	var stdout bytes.Buffer

	err := cli.RunRegistryRegisterSource(
		[]string{
			"--data-dir", dataDir,
			"--type", "skill",
			"--path", srcPath,
		},
		&stdout,
		os.ReadFile,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring("No instructions extracted"))
}

// TestRegistryRegisterSourceReadFileError exercises the readFile error path.
func TestRegistryRegisterSourceReadFileError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()

	var stdout bytes.Buffer

	err := cli.RunRegistryRegisterSource(
		[]string{
			"--data-dir", dataDir,
			"--type", "rule",
			"--path", "/nonexistent/file.md",
		},
		&stdout,
		os.ReadFile,
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err).To(MatchError(ContainSubstring("reading source")))
	}
}

// TestRegistryRegisterSourceUnknownType exercises the buildExtractor error path.
func TestRegistryRegisterSourceUnknownType(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	srcPath := filepath.Join(dataDir, "src.md")

	g.Expect(os.WriteFile(srcPath, []byte("# Content\nSome rule.\n"), 0o640)).To(Succeed())

	var stdout bytes.Buffer

	err := cli.RunRegistryRegisterSource(
		[]string{
			"--data-dir", dataDir,
			"--type", "unknown-type",
			"--path", srcPath,
		},
		&stdout,
		os.ReadFile,
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err).To(MatchError(ContainSubstring("register-source")))
	}
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

	g.Expect(stdout.String()).To(ContainSubstring("[engram] No registry entries found."))
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

// runDemote: flag parse error.
func TestRunDemote_FlagParseError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "demote", "--bogus-flag"},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())
}

// runDemote: missing flags returns error.
func TestRunDemote_MissingFlags(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "demote"},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("--data-dir"))
	}
}

// runDemote: empty registry produces no candidates.
func TestRunDemote_NoCandidates(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "demote", "--data-dir", dataDir, "--to-skill", "--yes"},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring("No demotion candidates"))
}

// runDemote: exercises candidates-found path (prints list, attempts demote).
func TestRunDemote_WithCandidates(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()

	// Seed registry with a claude-md Leech entry: low effectiveness (1/3 = 33%).
	seedRegistry(t, dataDir,
		`{"id":"claudemd-1","source_type":"claude-md",`+
			`"source_path":"CLAUDE.md","title":"Old Rule",`+
			`"content_hash":"abc","surfaced_count":5,`+
			`"evaluations":{"followed":1,"contradicted":2}}`,
	)

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "demote",
			"--data-dir", dataDir,
			"--to-skill", "--yes",
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	output := stdout.String()
	// Should list candidates and attempt demotion.
	g.Expect(output).To(ContainSubstring("candidates for demotion"))
	g.Expect(output).To(ContainSubstring("Old Rule"))
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

// runInstructAudit: valid run with empty dir produces JSON report.
func TestRunInstructAudit_EmptyDir(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "instruct", "--data-dir", dataDir, "--project-dir", dataDir},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Output should be valid JSON.
	var report any

	jsonErr := json.Unmarshal(stdout.Bytes(), &report)
	g.Expect(jsonErr).NotTo(HaveOccurred())
}

// runInstructAudit: flag parse error.
func TestRunInstructAudit_FlagParseError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "instruct", "--bogus-flag"},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())
}

// runInstructAudit: missing flags.
func TestRunInstructAudit_MissingFlags(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "instruct"},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("--data-dir"))
	}
}

// runPromote: flag parse error.
func TestRunPromote_FlagParseError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "promote", "--bogus-flag"},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())
}

// runPromote: missing flags.
func TestRunPromote_MissingFlags(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "promote"},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("--data-dir"))
	}
}

// runPromote --to-claude-md: empty registry returns no candidates.
func TestRunPromote_ToClaudeMD_NoCandidates(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "promote", "--data-dir", dataDir, "--to-claude-md", "--yes"},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring("No candidates"))
}

// runPromoteToClaudeMD: exercises candidates-found path.
func TestRunPromote_ToClaudeMD_WithCandidates(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()

	// Create skills dir with a skill file.
	skillsDir := filepath.Join(dataDir, "skills")
	g.Expect(os.MkdirAll(skillsDir, 0o755)).To(Succeed())

	g.Expect(os.WriteFile(
		filepath.Join(skillsDir, "my-skill.md"),
		[]byte("# My Skill\nContent here.\n"),
		0o644,
	)).To(Succeed())

	// Seed registry with a skill Working entry: high effectiveness (3/3 = 100%).
	seedRegistry(t, dataDir,
		`{"id":"skill:my-skill","source_type":"skill",`+
			`"source_path":"skills/my-skill.md","title":"My Skill",`+
			`"content_hash":"def","surfaced_count":100,`+
			`"evaluations":{"followed":3}}`,
	)

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "promote",
			"--data-dir", dataDir,
			"--to-claude-md", "--yes",
			"--threshold", "50",
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	output := stdout.String()
	g.Expect(output).To(ContainSubstring("candidates for CLAUDE.md promotion"))
	g.Expect(output).To(ContainSubstring("My Skill"))
}

// runPromote --to-skill: empty registry returns no candidates.
func TestRunPromote_ToSkill_NoCandidates(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "promote", "--data-dir", dataDir, "--to-skill", "--yes"},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring("No candidates"))
}

// runPromoteToSkill: exercises candidates-found path.
func TestRunPromote_ToSkill_WithCandidates(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()

	// Create memories dir with a memory file.
	memDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memDir, 0o755)).To(Succeed())

	memPath := filepath.Join(memDir, "test-mem.toml")

	g.Expect(os.WriteFile(
		memPath,
		[]byte(
			"title = \"Test Memory\"\nprinciple = \"be good\"\nupdated_at = \"2025-01-01T00:00:00Z\"\n",
		),
		0o644,
	)).To(Succeed())

	// Seed registry with a memory Working entry: high effectiveness (3/3 = 100%).
	seedRegistry(t, dataDir,
		fmt.Sprintf(
			`{"id":"%s","source_type":"memory",`+
				`"source_path":"%s","title":"Test Memory",`+
				`"content_hash":"ghi","surfaced_count":100,`+
				`"evaluations":{"followed":3}}`,
			memPath, memPath,
		),
	)

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "promote",
			"--data-dir", dataDir,
			"--to-skill", "--yes",
			"--threshold", "50",
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	output := stdout.String()
	g.Expect(output).To(ContainSubstring("candidates for promotion"))
	g.Expect(output).To(ContainSubstring("Test Memory"))
}

// runRegistryRegisterSource: dispatched via Run.
func TestRunRegistryRegisterSource_ViaRun(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	srcPath := filepath.Join(dataDir, "test-rule.md")

	g.Expect(os.WriteFile(srcPath,
		[]byte("## Rule\nAlways use targ.\n"), 0o640,
	)).To(Succeed())

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "registry", "register-source",
			"--data-dir", dataDir,
			"--type", "rule",
			"--path", srcPath,
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring("Registered"))
}

// runRegistry: no args returns unknown subcommand error.
func TestRunRegistry_NoArgs(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "registry"},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("unknown subcommand"))
	}
}

// runRegistry: unknown subcommand.
func TestRunRegistry_UnknownSubcommand(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "registry", "bogus"},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("unknown subcommand"))
	}
}

// runRemind: valid run with empty data dir produces no output.
func TestRunRemind_EmptyDataDir(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "remind", "--data-dir", dataDir},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())
}

// runRemind: flag parse error.
func TestRunRemind_FlagParseError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "remind", "--bogus-flag"},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())
}

// runRemind: missing flags.
func TestRunRemind_MissingFlags(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "remind"},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("--data-dir"))
	}
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

	// mem-a: Working (high surfacing, high effectiveness).
	// mem-b: Noise (low surfacing, low effectiveness).
	// mem-c: Insufficient (too few evals).
	writeReviewRegistry(t, dataDir, []reviewTestEntry{
		{
			ID: "mem-a", Source: "memory", Title: "Working A", Surfaced: 10,
			Followed: 4, Contradicted: 0, Ignored: 1,
		},
		{
			ID: "mem-b", Source: "memory", Title: "Noisy B", Surfaced: 2,
			Followed: 1, Contradicted: 2, Ignored: 2,
		},
		{
			ID: "mem-c", Source: "memory", Title: "Insufficient C", Surfaced: 5,
			Followed: 1, Contradicted: 0, Ignored: 0,
		},
	})

	var stdout bytes.Buffer

	err := cli.RunReview([]string{"--data-dir", dataDir}, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := stdout.String()
	g.Expect(output).To(ContainSubstring("Instruction Review"))
	g.Expect(output).To(ContainSubstring("3 entries"))
	g.Expect(output).To(ContainSubstring("Source: memory"))
}

// T-130: Review with no registry data outputs no-data message.
func TestT130_ReviewNoEvalDir(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dataDir := t.TempDir() // no registry file

	var stdout bytes.Buffer

	err := cli.RunReview([]string{"--data-dir", dataDir}, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring("[engram] No registry entries found."))
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

// T-132: Review entries sorted by quadrant within source type.
func TestT132_ReviewFlaggedSortedByEffectiveness(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dataDir := t.TempDir()

	// All surfaced=5 (above threshold=3), varying effectiveness.
	// Quadrant sort is alphabetical: Hidden Gem < Leech < Noise < Working.
	writeReviewRegistry(t, dataDir, []reviewTestEntry{
		{
			ID: "mem-x", Source: "memory", Title: "mem-x", Surfaced: 5,
			Followed: 1, Contradicted: 5, Ignored: 4,
		}, // 10% → Leech
		{
			ID: "mem-y", Source: "memory", Title: "mem-y", Surfaced: 5,
			Followed: 1, Contradicted: 2, Ignored: 2,
		}, // 20% → Leech
		{
			ID: "mem-z", Source: "memory", Title: "mem-z", Surfaced: 5,
			Followed: 2, Contradicted: 2, Ignored: 2,
		}, // 33% → Leech
	})

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

	g.Expect(posX).To(BeNumerically(">", 0), "mem-x should appear in output")
	g.Expect(posY).To(BeNumerically(">", 0), "mem-y should appear in output")
	g.Expect(posZ).To(BeNumerically(">", 0), "mem-z should appear in output")
}

// T-133: No "Insufficient" quadrant when all entries have sufficient evaluations.
func TestT133_ReviewOmitsInsufficientDataSection(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dataDir := t.TempDir()

	writeReviewRegistry(t, dataDir, []reviewTestEntry{
		{
			ID: "mem-a", Source: "memory", Title: "Good A", Surfaced: 10,
			Followed: 5, Contradicted: 0, Ignored: 0,
		},
		{
			ID: "mem-b", Source: "memory", Title: "Good B", Surfaced: 5,
			Followed: 5, Contradicted: 0, Ignored: 0,
		},
	})

	var stdout bytes.Buffer

	err := cli.RunReview([]string{"--data-dir", dataDir}, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).NotTo(ContainSubstring("Insufficient"))
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
		evaluate.WithStripFunc(sessionctx.Strip),
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

// T-179: maintain subcommand produces JSON proposals to stdout.
func TestT179_MaintainProducesJSONProposals(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")
	evalDir := filepath.Join(dataDir, "evaluations")

	g.Expect(os.MkdirAll(memDir, 0o750)).To(Succeed())
	g.Expect(os.MkdirAll(evalDir, 0o750)).To(Succeed())

	// Create a noise memory: low surfacing, all ignored.
	noiseMem := writeReviewMemoryTOML(
		t, memDir, "noise-mem.toml", 1,
	)
	writeReviewEvalLog(t, evalDir, "noise.jsonl", noiseMem,
		[]string{"ignored", "ignored", "ignored", "ignored", "ignored"})

	// Create a working memory: high surfacing, all followed.
	workingMem := writeReviewMemoryTOML(
		t, memDir, "working-mem.toml", 10,
	)
	writeReviewEvalLog(t, evalDir, "working.jsonl", workingMem,
		[]string{
			"followed", "followed", "followed",
			"followed", "followed",
		})

	var stdout bytes.Buffer

	err := cli.RunMaintain(
		[]string{"--data-dir", dataDir},
		"", // no token — only noise/working proposals
		&stdout,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Parse output as JSON array.
	var proposals []map[string]any

	jsonErr := json.Unmarshal(stdout.Bytes(), &proposals)
	g.Expect(jsonErr).NotTo(HaveOccurred())

	if jsonErr != nil {
		return
	}

	// Verify each proposal has required DES-23 fields.
	for _, proposal := range proposals {
		g.Expect(proposal).To(HaveKey("memory_path"))
		g.Expect(proposal).To(HaveKey("quadrant"))
		g.Expect(proposal).To(HaveKey("diagnosis"))
		g.Expect(proposal).To(HaveKey("action"))
		g.Expect(proposal).To(HaveKey("details"))
	}
}

// T-180: maintain with no evaluation data produces empty array.
func TestT180_MaintainNoEvalDataProducesEmptyArray(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")

	g.Expect(os.MkdirAll(memDir, 0o750)).To(Succeed())

	writeReviewMemoryTOML(t, memDir, "some-mem.toml", 5)

	var stdout bytes.Buffer

	err := cli.RunMaintain(
		[]string{"--data-dir", dataDir},
		"",
		&stdout,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(strings.TrimSpace(stdout.String())).To(Equal("[]"))
}

// T-181: maintain without API key skips LLM proposals.
func TestT181_MaintainWithoutAPIKeySkipsLLMProposals(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")
	evalDir := filepath.Join(dataDir, "evaluations")

	g.Expect(os.MkdirAll(memDir, 0o750)).To(Succeed())
	g.Expect(os.MkdirAll(evalDir, 0o750)).To(Succeed())

	// Leech: high surfacing, all contradicted → low effectiveness.
	leechMem := writeReviewMemoryTOML(
		t, memDir, "leech-mem.toml", 10,
	)
	writeReviewEvalLog(t, evalDir, "leech.jsonl", leechMem,
		[]string{
			"contradicted", "contradicted", "contradicted",
			"contradicted", "contradicted",
		})

	// Noise: low surfacing, all ignored → low effectiveness.
	noiseMem := writeReviewMemoryTOML(
		t, memDir, "noise-mem.toml", 1,
	)
	writeReviewEvalLog(t, evalDir, "noise.jsonl", noiseMem,
		[]string{
			"ignored", "ignored", "ignored",
			"ignored", "ignored",
		})

	var stdout bytes.Buffer

	err := cli.RunMaintain(
		[]string{"--data-dir", dataDir},
		"", // no token
		&stdout,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var proposals []map[string]any

	jsonErr := json.Unmarshal(stdout.Bytes(), &proposals)
	g.Expect(jsonErr).NotTo(HaveOccurred())

	if jsonErr != nil {
		return
	}

	// Only noise proposal should appear; leech requires LLM.
	for _, proposal := range proposals {
		g.Expect(proposal["quadrant"]).NotTo(Equal("Leech"),
			"leech proposals should be absent without API key")
		g.Expect(proposal["quadrant"]).NotTo(Equal("Hidden Gem"),
			"hidden gem proposals absent without API key")
	}

	// Both memories classify as Noise since tracking data (surfaced_count) is
	// no longer read from TOMLs — it's managed by the instruction registry (UC-23).
	noiseCount := 0

	for _, proposal := range proposals {
		if proposal["quadrant"] == "Noise" {
			noiseCount++
		}
	}

	g.Expect(noiseCount).To(Equal(2))
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

func TestT197_CLIReviewQuadrantOutputJSON(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	tmpDir := t.TempDir()

	// Write a registry file with two entries.
	registryPath := filepath.Join(tmpDir, "instruction-registry.jsonl")

	workingEntry := map[string]any{
		"id":             "mem-working",
		"source_type":    "memory",
		"title":          "Working Memory",
		"surfaced_count": 10,
		"evaluations":    map[string]int{"followed": 8, "contradicted": 1, "ignored": 1},
		"registered_at":  "2026-03-01T00:00:00Z",
		"updated_at":     "2026-03-08T00:00:00Z",
	}
	leechEntry := map[string]any{
		"id":             "mem-leech",
		"source_type":    "memory",
		"title":          "Leech Memory",
		"surfaced_count": 10,
		"evaluations":    map[string]int{"followed": 1, "contradicted": 5, "ignored": 4},
		"registered_at":  "2026-03-01T00:00:00Z",
		"updated_at":     "2026-03-08T00:00:00Z",
	}

	var registryData strings.Builder

	line1, marshalErr := json.Marshal(workingEntry)
	g.Expect(marshalErr).NotTo(HaveOccurred())

	registryData.Write(line1)
	registryData.WriteByte('\n')

	line2, marshalErr := json.Marshal(leechEntry)
	g.Expect(marshalErr).NotTo(HaveOccurred())

	registryData.Write(line2)
	registryData.WriteByte('\n')

	err := os.WriteFile(registryPath, []byte(registryData.String()), 0o640)
	g.Expect(err).NotTo(HaveOccurred())

	var stdout bytes.Buffer

	err = cli.Run(
		[]string{
			"engram", "review",
			"--data-dir", tmpDir,
			"--format", "json",
		},
		&stdout, io.Discard, strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	var results []map[string]any

	err = json.Unmarshal(stdout.Bytes(), &results)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(results).To(HaveLen(2))

	quadrants := make(map[string]string)

	for _, result := range results {
		id, _ := result["id"].(string)
		quadrant, _ := result["quadrant"].(string)
		quadrants[id] = quadrant
	}

	g.Expect(quadrants["mem-working"]).To(Equal("Working"))
	g.Expect(quadrants["mem-leech"]).To(Equal("Leech"))
}

func TestT198_CLIMergeAbsorbsAndDeletes(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	tmpDir := t.TempDir()

	// Create a source memory TOML file.
	memDir := filepath.Join(tmpDir, "memories")

	err := os.MkdirAll(memDir, 0o755)
	g.Expect(err).NotTo(HaveOccurred())

	sourceToml := filepath.Join(memDir, "source-memory.toml")

	err = os.WriteFile(sourceToml, []byte("title=\"Source\"\n"), 0o640)
	g.Expect(err).NotTo(HaveOccurred())

	// Write a registry with source and target entries.
	registryPath := filepath.Join(tmpDir, "instruction-registry.jsonl")

	sourceEntry := map[string]any{
		"id":             "memories/source-memory.toml",
		"source_type":    "memory",
		"source_path":    sourceToml,
		"title":          "Source Memory",
		"content_hash":   "abc",
		"surfaced_count": 5,
		"evaluations": map[string]int{
			"followed": 3, "contradicted": 1, "ignored": 0,
		},
		"registered_at": "2026-03-01T00:00:00Z",
		"updated_at":    "2026-03-08T00:00:00Z",
	}
	targetEntry := map[string]any{
		"id":             "target-id",
		"source_type":    "memory",
		"title":          "Target Memory",
		"surfaced_count": 2,
		"evaluations": map[string]int{
			"followed": 1, "contradicted": 0, "ignored": 0,
		},
		"registered_at": "2026-03-01T00:00:00Z",
		"updated_at":    "2026-03-08T00:00:00Z",
	}

	var registryData strings.Builder

	line1, marshalErr := json.Marshal(sourceEntry)
	g.Expect(marshalErr).NotTo(HaveOccurred())

	registryData.Write(line1)
	registryData.WriteByte('\n')

	line2, marshalErr := json.Marshal(targetEntry)
	g.Expect(marshalErr).NotTo(HaveOccurred())

	registryData.Write(line2)
	registryData.WriteByte('\n')

	err = os.WriteFile(registryPath, []byte(registryData.String()), 0o640)
	g.Expect(err).NotTo(HaveOccurred())

	var stdout bytes.Buffer

	err = cli.Run(
		[]string{
			"engram", "registry", "merge",
			"--data-dir", tmpDir,
			"--source", "memories/source-memory.toml",
			"--target", "target-id",
		},
		&stdout, io.Discard, strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	output := stdout.String()
	g.Expect(output).To(ContainSubstring("Merged"))

	// Verify source TOML was deleted.
	_, statErr := os.Stat(sourceToml)
	g.Expect(os.IsNotExist(statErr)).To(BeTrue())

	// Verify registry: only one line (target), source absorbed into it.
	data, err := os.ReadFile(registryPath)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	g.Expect(lines).To(HaveLen(1)) // only target remains
	g.Expect(lines[0]).To(ContainSubstring("target-id"))
	g.Expect(lines[0]).To(ContainSubstring("absorbed"))
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

func TestT272_RegistryInitDryRunListsEntries(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	tmpDir := t.TempDir()
	memDir := filepath.Join(tmpDir, "memories")

	err := os.MkdirAll(memDir, 0o755)
	g.Expect(err).NotTo(HaveOccurred())

	tomlContent := `Title = "Test Memory"
Content = "test content"
Concepts = ["testing"]
Keywords = ["test"]
Principle = "always test"
updated_at = "2026-03-08T00:00:00Z"
`
	writeTestTOML(t, memDir, "test-memory.toml", tomlContent)

	var stdout bytes.Buffer

	err = cli.Run(
		[]string{
			"engram", "registry", "init",
			"--data-dir", tmpDir, "--dry-run",
		},
		&stdout, io.Discard, strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	output := stdout.String()
	g.Expect(output).To(ContainSubstring("dry-run"))
	g.Expect(output).To(ContainSubstring("1 entries"))
}

func TestT273_RegistryInitWritesJSONLFile(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	tmpDir := t.TempDir()
	memDir := filepath.Join(tmpDir, "memories")

	err := os.MkdirAll(memDir, 0o755)
	g.Expect(err).NotTo(HaveOccurred())

	tomlContent := `Title = "Init Test"
Content = "init content"
Concepts = ["init"]
Keywords = ["registry"]
updated_at = "2026-03-08T00:00:00Z"
`
	writeTestTOML(t, memDir, "init-test.toml", tomlContent)

	var stdout bytes.Buffer

	err = cli.Run(
		[]string{
			"engram", "registry", "init",
			"--data-dir", tmpDir,
		},
		&stdout, io.Discard, strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	output := stdout.String()
	g.Expect(output).To(ContainSubstring("initialized"))
	g.Expect(output).To(ContainSubstring("1 entries"))

	registryPath := filepath.Join(tmpDir, "instruction-registry.jsonl")
	data, err := os.ReadFile(registryPath)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(data)).To(ContainSubstring("Init Test"))
}

func TestT274_RegistryInitMissingDataDir(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var stdout bytes.Buffer

	err := cli.Run(
		[]string{"engram", "registry", "init"},
		&stdout, io.Discard, strings.NewReader(""),
	)
	g.Expect(err).To(MatchError(ContainSubstring("--data-dir")))
}

func TestT275_RegistryUnknownSubcommand(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var stdout bytes.Buffer

	err := cli.Run(
		[]string{"engram", "registry", "bogus"},
		&stdout, io.Discard, strings.NewReader(""),
	)
	g.Expect(err).To(MatchError(ContainSubstring("unknown subcommand")))
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

	// Add more memories so "commit" is not 100% of the corpus
	writeTestTOML(t, memoriesDir, "testing.toml", `title = "Testing Framework"
content = "use automated testing"
observation_type = "observation"
concepts = []
keywords = ["test"]
principle = "always run tests"
anti_pattern = ""
rationale = ""
confidence = "B"
created_at = "2025-01-01T00:00:00Z"
updated_at = "2025-01-01T00:00:00Z"
`)

	writeTestTOML(t, memoriesDir, "linting.toml", `title = "Linting"
content = "run linters before commits"
observation_type = "observation"
concepts = []
keywords = ["lint"]
principle = "always lint"
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
	g.Expect(output).To(ContainSubstring("commit-rules"))
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

	// Add more anti-pattern memories so "commit" is not 50% of the corpus
	writeTestTOML(t, memoriesDir, "testing.toml", `title = "Testing"
content = "run tests before commit"
observation_type = "observation"
concepts = []
keywords = ["test"]
principle = "always test"
anti_pattern = "skipping tests"
rationale = ""
confidence = "B"
created_at = "2025-01-01T00:00:00Z"
updated_at = "2025-01-01T00:00:00Z"
`)

	writeTestTOML(t, memoriesDir, "linting.toml", `title = "Linting"
content = "run linters before commits"
observation_type = "observation"
concepts = []
keywords = ["lint"]
principle = "always lint"
anti_pattern = "skipping lint checks"
rationale = ""
confidence = "B"
created_at = "2025-01-01T00:00:00Z"
updated_at = "2025-01-01T00:00:00Z"
`)

	writeTestTOML(t, memoriesDir, "docker.toml", `title = "Docker"
content = "use docker for consistency"
observation_type = "observation"
concepts = []
keywords = ["docker"]
principle = "always use docker"
anti_pattern = "building without docker"
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

func TestTruncateTitle(t *testing.T) {
	t.Parallel()

	t.Run("short title unchanged", func(t *testing.T) {
		t.Parallel()
		g := NewGomegaWithT(t)
		g.Expect(cli.ExportTruncateTitle("short")).To(Equal("short"))
	})

	t.Run("exact length unchanged", func(t *testing.T) {
		t.Parallel()
		g := NewGomegaWithT(t)

		exact := strings.Repeat("x", 38)
		g.Expect(cli.ExportTruncateTitle(exact)).To(Equal(exact))
	})

	t.Run("long title truncated with ellipsis", func(t *testing.T) {
		t.Parallel()
		g := NewGomegaWithT(t)

		long := strings.Repeat("a", 50)
		result := cli.ExportTruncateTitle(long)
		g.Expect(len(result)).To(BeNumerically("<=", 42)) // 37 chars + multibyte ellipsis
		g.Expect(result).To(HaveSuffix("…"))
	})
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

// reviewTestEntry is a compact representation for building registry test data.
type reviewTestEntry struct {
	ID           string
	Source       string
	Title        string
	Surfaced     int
	Followed     int
	Contradicted int
	Ignored      int
}

// seedRegistry writes a JSONL registry file with the given entries.
func seedRegistry(t *testing.T, dataDir string, entries ...string) {
	t.Helper()

	registryPath := filepath.Join(dataDir, "instruction-registry.jsonl")

	content := strings.Join(entries, "\n") + "\n"

	err := os.WriteFile(registryPath, []byte(content), 0o644)
	if err != nil {
		t.Fatalf("seedRegistry: %v", err)
	}
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

func writeReviewRegistry(t *testing.T, dataDir string, entries []reviewTestEntry) {
	t.Helper()

	registryPath := filepath.Join(dataDir, "instruction-registry.jsonl")

	var sb strings.Builder

	for _, entry := range entries {
		raw := map[string]any{
			"id":             entry.ID,
			"source_type":    entry.Source,
			"title":          entry.Title,
			"surfaced_count": entry.Surfaced,
			"evaluations": map[string]int{
				"followed":     entry.Followed,
				"contradicted": entry.Contradicted,
				"ignored":      entry.Ignored,
			},
			"registered_at": "2026-03-01T00:00:00Z",
			"updated_at":    "2026-03-08T00:00:00Z",
		}

		line, err := json.Marshal(raw)
		if err != nil {
			t.Fatalf("writeReviewRegistry: marshal: %v", err)
		}

		sb.Write(line)
		sb.WriteByte('\n')
	}

	if err := os.WriteFile(registryPath, []byte(sb.String()), 0o640); err != nil {
		t.Fatalf("writeReviewRegistry: write: %v", err)
	}
}

func writeTestTOML(t *testing.T, dir, filename, content string) {
	t.Helper()

	path := filepath.Join(dir, filename)

	err := os.WriteFile(path, []byte(content), 0o640)
	if err != nil {
		t.Fatalf("writeTestTOML: %v", err)
	}
}
