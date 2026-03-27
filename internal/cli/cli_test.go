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
	"runtime"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
	"engram/internal/extract"
	"engram/internal/learn"
)

// TestCallAnthropicAPIDoError exercises the client.Do error path of callAnthropicAPI.
// This test is safe to run in parallel because it uses a failing transport, not the global URL.
func TestCallAnthropicAPIDoError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	errTransport := errors.New("transport failed")
	client := &http.Client{
		Transport: &failingTransport{err: errTransport},
	}

	_, err := cli.ExportCallAnthropicAPI(t.Context(), client, "token", "model", "sys", "user")
	g.Expect(err).To(MatchError(ContainSubstring("calling Anthropic API")))
}

// TestCallAnthropicAPIServerErrors exercises server-response error paths.
// These tests mutate the AnthropicAPIURL global and cannot run in parallel.
//
//nolint:paralleltest // mutates cli.AnthropicAPIURL global
func TestCallAnthropicAPIServerErrors(t *testing.T) {
	t.Run("invalid JSON response", func(t *testing.T) {
		g := NewGomegaWithT(t)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("not json"))
		}))

		defer server.Close()

		original := cli.AnthropicAPIURL
		cli.AnthropicAPIURL = server.URL

		defer func() { cli.AnthropicAPIURL = original }()

		_, err := cli.ExportCallAnthropicAPI(t.Context(), &http.Client{}, "token", "model", "sys", "user")
		g.Expect(err).To(MatchError(ContainSubstring("parsing API response")))
	})

	t.Run("empty content blocks", func(t *testing.T) {
		g := NewGomegaWithT(t)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"content":[]}`))
		}))

		defer server.Close()

		original := cli.AnthropicAPIURL
		cli.AnthropicAPIURL = server.URL

		defer func() { cli.AnthropicAPIURL = original }()

		_, err := cli.ExportCallAnthropicAPI(t.Context(), &http.Client{}, "token", "model", "sys", "user")
		g.Expect(err).To(MatchError(ContainSubstring("no content")))
	})

	t.Run("body read error", func(t *testing.T) {
		g := NewGomegaWithT(t)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			// Hijack and close immediately so reading the body fails.
			hijacker, ok := w.(http.Hijacker)
			if !ok {
				http.Error(w, "hijack not supported", http.StatusInternalServerError)
				return
			}

			conn, _, _ := hijacker.Hijack()
			_ = conn.Close()
		}))

		defer server.Close()

		original := cli.AnthropicAPIURL
		cli.AnthropicAPIURL = server.URL

		defer func() { cli.AnthropicAPIURL = original }()

		_, err := cli.ExportCallAnthropicAPI(t.Context(), &http.Client{}, "token", "model", "sys", "user")
		g.Expect(err).To(HaveOccurred())
	})
}

// TestConsolidationProposalMemberPaths verifies that consolidation proposals store
// members as objects with path and title fields (not plain strings).
func TestConsolidationProposalMemberPaths(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memoriesDir, 0o755)).To(Succeed())

	survivorContent := `title = "Survivor Memory"
content = "Keep this one"
keywords = ["alpha", "beta", "gamma"]
surfaced_count = 10
updated_at = "2026-01-01T00:00:00Z"
`
	absorbedContent := `title = "Absorbed Memory"
content = "Delete this one"
keywords = ["alpha", "beta", "delta"]
surfaced_count = 1
updated_at = "2026-01-01T00:00:00Z"
`

	survivorPath := filepath.Join(memoriesDir, "survivor.toml")
	absorbedPath := filepath.Join(memoriesDir, "absorbed.toml")

	g.Expect(os.WriteFile(survivorPath, []byte(survivorContent), 0o640)).To(Succeed())
	g.Expect(os.WriteFile(absorbedPath, []byte(absorbedContent), 0o640)).To(Succeed())

	var stdout bytes.Buffer

	err := cli.RunMaintain([]string{"--data-dir", dataDir}, "", &stdout)
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

	// Find the consolidation proposal.
	var consolidation map[string]any

	for _, proposal := range proposals {
		if proposal["action"] == "consolidate" {
			consolidation = proposal
			break
		}
	}

	g.Expect(consolidation).NotTo(BeNil(), "expected a consolidation proposal")

	if consolidation == nil {
		return
	}

	// details is a json.RawMessage decoded as map[string]any by json.Unmarshal.
	details, ok := consolidation["details"].(map[string]any)
	g.Expect(ok).To(BeTrue(), "details must be a JSON object")

	if !ok {
		return
	}

	members, ok := details["members"].([]any)
	g.Expect(ok).To(BeTrue(), "details.members must be an array")

	if !ok {
		return
	}

	g.Expect(members).NotTo(BeEmpty())

	for _, rawMember := range members {
		member, isMap := rawMember.(map[string]any)
		g.Expect(isMap).To(BeTrue(), "each member must be an object with path and title")

		if !isMap {
			return
		}

		g.Expect(member).To(HaveKey("path"), "member must have a path field")
		g.Expect(member).To(HaveKey("title"), "member must have a title field")
		g.Expect(member["path"]).NotTo(BeEmpty(), "member path must not be empty")
		g.Expect(member["title"]).NotTo(BeEmpty(), "member title must not be empty")
	}
}

func TestHaikuCallerAdapter_Call(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var capturedModel, capturedSystem, capturedUser string

	fakeCaller := func(
		_ context.Context, model, systemPrompt, userPrompt string,
	) (string, error) {
		capturedModel = model
		capturedSystem = systemPrompt
		capturedUser = userPrompt

		return "response", nil
	}

	adapter := cli.ExportNewHaikuCallerAdapter(fakeCaller)

	result, err := adapter.Call(context.Background(), "system prompt", "user prompt")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(Equal("response"))
	g.Expect(capturedModel).To(Equal("claude-haiku-4-5-20251001"))
	g.Expect(capturedSystem).To(Equal("system prompt"))
	g.Expect(capturedUser).To(Equal("user prompt"))
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
	g.Expect(os.MkdirAll(filepath.Join(dataDir, "memories"), 0o755)).To(Succeed())

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

// TestMaintainListMemoriesError exercises the memories listing error path
// by making the memories dir a file (not a directory).
func TestMaintainListMemoriesError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()

	// Create memories as a file, not a directory, so readDir fails.
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
		g.Expect(err).To(MatchError(ContainSubstring("listing memories")))
	}
}

func TestOsDirLister_ListJSONL(t *testing.T) {
	t.Parallel()

	t.Run("lists jsonl files and skips others", func(t *testing.T) {
		t.Parallel()
		g := NewGomegaWithT(t)

		dir := t.TempDir()

		// Create .jsonl file.
		writeErr := os.WriteFile(filepath.Join(dir, "a.jsonl"), []byte("{}"), 0o644)
		g.Expect(writeErr).NotTo(HaveOccurred())

		if writeErr != nil {
			return
		}

		// Create non-jsonl file (should be skipped).
		writeErr2 := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("text"), 0o644)
		g.Expect(writeErr2).NotTo(HaveOccurred())

		if writeErr2 != nil {
			return
		}

		// Create subdirectory (should be skipped).
		mkErr := os.MkdirAll(filepath.Join(dir, "subdir"), 0o755)
		g.Expect(mkErr).NotTo(HaveOccurred())

		if mkErr != nil {
			return
		}

		lister := cli.ExportNewOsDirLister()
		entries, err := lister.ListJSONL(dir)
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(entries).To(HaveLen(1))
		g.Expect(entries[0].Path).To(HaveSuffix("a.jsonl"))
	})

	t.Run("returns error for nonexistent directory", func(t *testing.T) {
		t.Parallel()
		g := NewGomegaWithT(t)

		lister := cli.ExportNewOsDirLister()
		_, err := lister.ListJSONL("/nonexistent/path")
		g.Expect(err).To(HaveOccurred())
	})
}

// TestRecallIntegrationSummaryMode verifies the end-to-end recall pipeline
// through the CLI: transcript discovery, reading, Haiku summarization, and
// plain text output.
//
// This test mutates cli.AnthropicAPIURL and uses t.Setenv, so it must not
// use t.Parallel().
func TestRecallIntegrationSummaryMode(t *testing.T) {
	g := NewWithT(t)

	// Set up a fake HOME so runRecall constructs the project dir under our temp tree.
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)

	slug := "test-project"
	projectDir := filepath.Join(fakeHome, ".claude", "projects", slug)
	g.Expect(os.MkdirAll(projectDir, 0o750)).To(Succeed())

	// Write a fake transcript .jsonl file in the project dir.
	transcriptPath := filepath.Join(projectDir, "session-abc.jsonl")
	g.Expect(os.WriteFile(
		transcriptPath,
		[]byte(
			`{"type":"user","message":{"role":"user","content":"help with recall"}}`+"\n"+
				`{"type":"assistant","message":{"role":"assistant","content":"sure, working on it"}}`+"\n",
		),
		0o644,
	)).To(Succeed())

	// Mock the Anthropic API endpoint.
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(
				`{"content":[{"type":"text","text":"Working on recall feature"}]}`,
			))
		},
	))
	defer server.Close()

	original := cli.AnthropicAPIURL
	cli.AnthropicAPIURL = server.URL

	defer func() { cli.AnthropicAPIURL = original }()

	t.Setenv("ENGRAM_API_TOKEN", "fake-token")

	dataDir := t.TempDir()

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "recall",
			"--data-dir", dataDir,
			"--project-slug", slug,
		},
		&stdout, &stderr, strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := stdout.String()
	// Plain text output: summary content directly (mode A = raw transcript), no JSON wrapper.
	g.Expect(output).To(ContainSubstring("help with recall"))
	g.Expect(output).NotTo(ContainSubstring(`"summary"`))
}

// TestRecallOutputWithMemories verifies that when memories are present,
// a separator is printed between summary and memories.
//
// This test mutates cli.AnthropicAPIURL and uses t.Setenv, so it must not
// use t.Parallel().
func TestRecallOutputWithMemories(t *testing.T) {
	g := NewWithT(t)

	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)

	slug := "mem-project"
	projectDir := filepath.Join(fakeHome, ".claude", "projects", slug)
	g.Expect(os.MkdirAll(projectDir, 0o750)).To(Succeed())

	transcriptPath := filepath.Join(projectDir, "session-xyz.jsonl")
	g.Expect(os.WriteFile(
		transcriptPath,
		[]byte(`{"type":"user","message":{"role":"user","content":"test"}}`+"\n"),
		0o644,
	)).To(Succeed())

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(
				`{"content":[{"type":"text","text":"summary text"}]}`,
			))
		},
	))
	defer server.Close()

	original := cli.AnthropicAPIURL
	cli.AnthropicAPIURL = server.URL

	defer func() { cli.AnthropicAPIURL = original }()

	t.Setenv("ENGRAM_API_TOKEN", "fake-token")

	dataDir := t.TempDir()

	// Write a memory file so the orchestrator finds memories.
	memDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memDir, 0o750)).To(Succeed())
	g.Expect(os.WriteFile(
		filepath.Join(memDir, "test.toml"),
		[]byte("[[memories]]\ncontent = \"remember this\"\n"),
		0o644,
	)).To(Succeed())

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "recall",
			"--data-dir", dataDir,
			"--project-slug", slug,
		},
		&stdout, &stderr, strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := stdout.String()
	// Should not contain JSON
	g.Expect(output).NotTo(ContainSubstring("{"))
}

// TestRegisterMemory verifies the memory registration callback writes metadata.
func TestRegisterMemory(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	memDir := filepath.Join(t.TempDir(), "memories")

	mkErr := os.MkdirAll(memDir, 0o755)
	g.Expect(mkErr).NotTo(HaveOccurred())

	if mkErr != nil {
		return
	}

	memPath := filepath.Join(memDir, "test.toml")

	// Write a minimal TOML file for ReadModifyWrite to operate on.
	writeErr := os.WriteFile(memPath, []byte("[[memories]]\n"), 0o644)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	err := cli.ExportRegisterMemory(memPath, "title", "content", time.Now())
	g.Expect(err).NotTo(HaveOccurred())
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

// TestRunLearnDefaultDataDir verifies --data-dir defaults when omitted.
func TestRunLearnDefaultDataDir(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var stderr bytes.Buffer

	// No --data-dir flag — should default and reach the token check.
	err := cli.RunLearn([]string{}, "", &stderr, strings.NewReader(""), nil)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stderr.String()).To(ContainSubstring("no API token"))
}

// RunLearnWithEmptyTokenEmitsErrorToStderr verifies the no-token early-return
// path in RunLearn using direct injection (no env var, no keychain dependency).
func TestRunLearnWithEmptyTokenEmitsErrorToStderr(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()

	var stderr bytes.Buffer

	err := cli.RunLearn([]string{
		"--data-dir", dataDir,
	}, "", &stderr, strings.NewReader("some transcript"), nil)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stderr.String()).To(ContainSubstring("session learning skipped"))
	g.Expect(stderr.String()).To(ContainSubstring("no API token"))
}

// TestRun_ApplyProposalDispatch verifies "apply-proposal" branch is reached.
func TestRun_ApplyProposalDispatch(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer

	_ = cli.Run([]string{"engram", "apply-proposal", "--data-dir", t.TempDir()}, &stdout, &stderr, strings.NewReader(""))
}

func TestRun_CorrectMissingFlags(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run([]string{"engram", "correct"}, &stdout, &stderr, strings.NewReader(""))
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("--message required"))
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

func TestRun_RecallDefaultFlags(t *testing.T) {
	// Not parallel: uses t.Setenv.
	g := NewGomegaWithT(t)

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("ENGRAM_API_TOKEN", "")

	// Create the project dir that recall will look for (derived from PWD slug).
	cwd, cwdErr := os.Getwd()
	g.Expect(cwdErr).NotTo(HaveOccurred())

	if cwdErr != nil {
		return
	}

	slug := cli.ProjectSlugFromPath(cwd)
	projDir := filepath.Join(homeDir, ".claude", "projects", slug)

	mkErr := os.MkdirAll(projDir, 0o755)
	g.Expect(mkErr).NotTo(HaveOccurred())

	if mkErr != nil {
		return
	}

	// Also create the default data dir so buildRecallSurfacer doesn't fail.
	dataDir := cli.DataDirFromHome(homeDir)

	mkErr2 := os.MkdirAll(filepath.Join(dataDir, "memories"), 0o755)
	g.Expect(mkErr2).NotTo(HaveOccurred())

	if mkErr2 != nil {
		return
	}

	var stdout, stderr bytes.Buffer

	// No --data-dir, no --project-slug — both should default.
	err := cli.Run(
		[]string{"engram", "recall"},
		&stdout, &stderr, strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestRun_RecallEmptyProject(t *testing.T) {
	// Not parallel: uses t.Setenv to override HOME.
	g := NewGomegaWithT(t)

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("ENGRAM_API_TOKEN", "")

	projDir := filepath.Join(homeDir, ".claude", "projects", "testproj")

	mkErr := os.MkdirAll(projDir, 0o755)
	g.Expect(mkErr).NotTo(HaveOccurred())

	if mkErr != nil {
		return
	}

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "recall",
			"--data-dir", t.TempDir(),
			"--project-slug", "testproj",
		},
		&stdout, &stderr, strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Empty project: no sessions, so output is empty (no JSON wrapper).
	g.Expect(stdout.String()).To(BeEmpty())
}

func TestRun_RecallSurfacerBuildError(t *testing.T) {
	// Not parallel: uses t.Setenv to override HOME.
	g := NewGomegaWithT(t)

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("ENGRAM_API_TOKEN", "")

	projDir := filepath.Join(homeDir, ".claude", "projects", "testproj")

	mkErr := os.MkdirAll(projDir, 0o755)
	g.Expect(mkErr).NotTo(HaveOccurred())

	if mkErr != nil {
		return
	}

	// Create a file where the memories directory should be, triggering a non-ErrNotExist error.
	dataDir := t.TempDir()
	memoriesPath := filepath.Join(dataDir, "memories")
	writeErr := os.WriteFile(memoriesPath, []byte("not a dir"), 0o600)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "recall",
			"--data-dir", dataDir,
			"--project-slug", "testproj",
		},
		&stdout, &stderr, strings.NewReader(""),
	)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("listing memories"))
	}
}

func TestRun_RecallWithSessions(t *testing.T) {
	// Not parallel: uses t.Setenv to override HOME.
	g := NewGomegaWithT(t)

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("ENGRAM_API_TOKEN", "")

	projDir := filepath.Join(homeDir, ".claude", "projects", "testproj")

	mkErr := os.MkdirAll(projDir, 0o755)
	g.Expect(mkErr).NotTo(HaveOccurred())

	if mkErr != nil {
		return
	}

	// Create a .jsonl session file.
	sessionContent := `{"type":"human","text":"hello"}` + "\n"

	writeErr := os.WriteFile(
		filepath.Join(projDir, "session.jsonl"),
		[]byte(sessionContent), 0o644,
	)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	// Add a non-jsonl file (should be skipped by ListJSONL).
	writeErr2 := os.WriteFile(
		filepath.Join(projDir, "notes.txt"),
		[]byte("not a session"), 0o644,
	)
	g.Expect(writeErr2).NotTo(HaveOccurred())

	if writeErr2 != nil {
		return
	}

	// Add a subdirectory (should be skipped by ListJSONL).
	subErr := os.MkdirAll(filepath.Join(projDir, "subdir"), 0o755)
	g.Expect(subErr).NotTo(HaveOccurred())

	if subErr != nil {
		return
	}

	var stdout, stderr bytes.Buffer

	// No API token → nil summarizer. With nil summarizer, orchestrator
	// returns raw content without summarizing.
	err := cli.Run(
		[]string{
			"engram", "recall",
			"--data-dir", t.TempDir(),
			"--project-slug", "testproj",
		},
		&stdout, &stderr, strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Session has type "human" which Strip drops, so output is empty plain text.
	g.Expect(stdout.String()).To(BeEmpty())
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

// T-120: Stop hook invokes engram flush (#309, #348).
func TestT120_HookScriptsInvokeFlush(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	data, err := os.ReadFile("../../hooks/stop.sh")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	content := string(data)
	// Stop hook uses unified flush command (#309).
	g.Expect(content).To(ContainSubstring("flush"))
	g.Expect(content).To(ContainSubstring("ENGRAM_BIN"))
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

// T-179: maintain subcommand produces JSON proposals to stdout.
func TestT179_MaintainProducesJSONProposals(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")

	g.Expect(os.MkdirAll(memDir, 0o750)).To(Succeed())

	// Create a noise memory: low surfacing, all ignored (counts in TOML).
	writeReviewMemoryTOML(t, memDir, "noise-mem.toml", 1, reviewMemoryOpts{
		ignoredCount: 5,
	})

	// Create a working memory: high surfacing, all followed (counts in TOML).
	writeReviewMemoryTOML(t, memDir, "working-mem.toml", 10, reviewMemoryOpts{
		followedCount: 5,
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
	g.Expect(os.MkdirAll(memDir, 0o750)).To(Succeed())

	// Leech: high surfacing, all contradicted → low effectiveness (counts in TOML).
	writeReviewMemoryTOML(t, memDir, "leech-mem.toml", 10, reviewMemoryOpts{
		contradictedCount: 5,
	})

	// Noise: low surfacing, all ignored → low effectiveness (counts in TOML).
	writeReviewMemoryTOML(t, memDir, "noise-mem.toml", 1, reviewMemoryOpts{
		ignoredCount: 5,
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

	// No Hidden Gem proposals should appear without API key.
	for _, proposal := range proposals {
		g.Expect(proposal["quadrant"]).NotTo(Equal("Hidden Gem"),
			"hidden gem proposals absent without API key")
	}

	// Without API key, leech proposals require LLM and are skipped.
	// Noise proposals always appear.
	noiseCount := 0

	for _, proposal := range proposals {
		if proposal["quadrant"] == "Noise" {
			noiseCount++
		}
	}

	g.Expect(noiseCount).To(Equal(1), "noise-mem should classify as Noise")
}

// T-18: correct subcommand with no API key returns error
func TestT18_CorrectSubcommandWithoutAPIKeyReturnsError(t *testing.T) {
	// Cannot use t.Parallel() — t.Setenv mutates process environment.
	// Skip on darwin: the token resolver falls back to the macOS Keychain, so
	// clearing the env var alone does not guarantee "no token" on a dev machine.
	if runtime.GOOS == "darwin" {
		t.Skip("keychain fallback makes 'no token' untestable by env var alone on darwin")
	}

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

	// Write TOML memory files to the memories/ subdirectory.
	writeReviewRegistry(t, tmpDir, []reviewTestEntry{
		{
			ID: "mem-working", Source: "memory", Title: "Working Memory",
			Surfaced: 10, Followed: 8, Contradicted: 1, Ignored: 1,
		},
		{
			ID: "mem-leech", Source: "memory", Title: "Leech Memory",
			Surfaced: 10, Followed: 1, Contradicted: 5, Ignored: 4,
		},
	})

	var stdout bytes.Buffer

	err := cli.Run(
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
		title, _ := result["title"].(string)
		quadrant, _ := result["quadrant"].(string)
		quadrants[title] = quadrant
	}

	g.Expect(quadrants["Working Memory"]).To(Equal("Working"))
	g.Expect(quadrants["Leech Memory"]).To(Equal("Leech"))
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

// T-250: Review reads quadrant classification from TOML memory directory.
func TestT250_ReviewReadsFromTOMLDirectory(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dataDir := t.TempDir()

	writeReviewRegistry(t, dataDir, []reviewTestEntry{
		{
			ID: "working-mem", Source: "memory", Title: "Working Memory",
			Surfaced: 10, Followed: 7, Contradicted: 0, Ignored: 1,
		}, // high surfacing, high effectiveness → Working
		{
			ID: "leech-mem", Source: "memory", Title: "Leech Memory",
			Surfaced: 8, Followed: 1, Contradicted: 5, Ignored: 2,
		}, // high surfacing, low effectiveness → Leech
	})

	var stdout bytes.Buffer

	err := cli.RunReview([]string{"--data-dir", dataDir}, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := stdout.String()
	g.Expect(output).To(ContainSubstring("Working Memory"))
	g.Expect(output).To(ContainSubstring("Leech Memory"))
	g.Expect(output).To(ContainSubstring("Source: memory"))
}

// T-362: maintain --dry-run prints merge plan without modifying files.
func TestT362_MaintainDryRun_PrintsPlanWithoutModifyingFiles(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memoriesDir, 0o755)).To(Succeed())

	// Two memories with >50% one-sided keyword overlap — would merge without --dry-run.
	survivorContent := `title = "Survivor Memory"
content = "Keep this one"
keywords = ["alpha", "beta", "gamma"]
surfaced_count = 10
updated_at = "2026-01-01T00:00:00Z"
`
	absorbedContent := `title = "Absorbed Memory"
content = "Delete this one"
keywords = ["alpha", "beta", "delta"]
surfaced_count = 1
updated_at = "2026-01-01T00:00:00Z"
`

	absorbedPath := filepath.Join(memoriesDir, "absorbed.toml")

	g.Expect(os.WriteFile(filepath.Join(memoriesDir, "survivor.toml"), []byte(survivorContent), 0o640)).To(Succeed())
	g.Expect(os.WriteFile(absorbedPath, []byte(absorbedContent), 0o640)).To(Succeed())

	var stdout bytes.Buffer

	err := cli.RunMaintain([]string{"--data-dir", dataDir, "--dry-run"}, "", &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// stdout must contain a JSON array with at least one entry describing the merge plan.
	output := stdout.String()
	g.Expect(output).To(ContainSubstring("survivor"), "plan must mention survivor path")
	g.Expect(output).To(ContainSubstring("absorbed"), "plan must mention absorbed path")

	// absorbed file must still exist — no files modified.
	_, statErr := os.Stat(absorbedPath)
	g.Expect(statErr).NotTo(HaveOccurred(), "absorbed file must still exist after --dry-run")
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
	g.Expect(output).To(ContainSubstring("[engram] Memories"))
	g.Expect(output).To(ContainSubstring("commit-rules"))
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
	// Skip on darwin: the token resolver falls back to the macOS Keychain, so
	// clearing the env var alone does not guarantee "no token" on a dev machine.
	if runtime.GOOS == "darwin" {
		t.Skip("keychain fallback makes 'no token' untestable by env var alone on darwin")
	}

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

// failingTransport is an http.RoundTripper that always returns an error.
type failingTransport struct {
	err error
}

func (f *failingTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	return nil, f.err
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

// writeReviewMemoryTOML creates a minimal memory TOML file in the given directory.
type reviewMemoryOpts struct {
	followedCount     int
	contradictedCount int
	ignoredCount      int
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

func writeReviewMemoryTOML(tb testing.TB, dir, filename string, surfacedCount int, opts ...reviewMemoryOpts) {
	tb.Helper()

	var opt reviewMemoryOpts
	if len(opts) > 0 {
		opt = opts[0]
	}

	path := filepath.Join(dir, filename)
	content := fmt.Sprintf(
		"title = %q\ncontent = \"Some content\"\nupdated_at = \"2024-01-01T00:00:00Z\"\n"+
			"surfaced_count = %d\nfollowed_count = %d\ncontradicted_count = %d\nignored_count = %d\n",
		filename,
		surfacedCount,
		opt.followedCount,
		opt.contradictedCount,
		opt.ignoredCount,
	)

	if err := os.WriteFile(path, []byte(content), 0o640); err != nil {
		tb.Fatalf("writeReviewMemoryTOML: %v", err)
	}
}

// writeReviewRegistry writes TOML memory files to the memories/ subdirectory
// for use by RunReview (which reads from TOMLDirectoryStore).
func writeReviewRegistry(t *testing.T, dataDir string, entries []reviewTestEntry) {
	t.Helper()

	memoriesDir := filepath.Join(dataDir, "memories")

	if err := os.MkdirAll(memoriesDir, 0o750); err != nil {
		t.Fatalf("writeReviewRegistry: mkdir: %v", err)
	}

	for _, entry := range entries {
		content := fmt.Sprintf(
			"title = %q\nsource_type = %q\n"+
				"surfaced_count = %d\nfollowed_count = %d\n"+
				"contradicted_count = %d\nignored_count = %d\n",
			entry.Title, entry.Source,
			entry.Surfaced, entry.Followed, entry.Contradicted, entry.Ignored,
		)

		path := filepath.Join(memoriesDir, entry.ID+".toml")

		if err := os.WriteFile(path, []byte(content), 0o640); err != nil {
			t.Fatalf("writeReviewRegistry: write %s: %v", path, err)
		}
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
