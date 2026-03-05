package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
	"engram/internal/learn"
)

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
	g.Expect(output).To(ContainSubstring("Test Memory"))
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
	g.Expect(output).To(ContainSubstring("Commit Rules"))
	g.Expect(output).To(ContainSubstring("commit"))
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

// T-62b: learn with missing --data-dir returns error.
func TestT62b_LearnMissingDataDir(t *testing.T) {
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

// T-62c: learn with invalid flag returns error.
func TestT62c_LearnInvalidFlag(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var stdout, stderr bytes.Buffer

	err := cli.Run([]string{
		"engram", "learn",
		"--unknown-flag",
	}, &stdout, &stderr, strings.NewReader(""))

	g.Expect(err).To(HaveOccurred())
}

// T-62d: learn with token but API failure returns error (covers post-token pipeline path).
func TestT62d_LearnWithTokenAndAPIFailureReturnsError(t *testing.T) {
	// Cannot use t.Parallel() — t.Setenv mutates process environment.
	g := NewGomegaWithT(t)

	t.Setenv("ENGRAM_API_TOKEN", "fake-token-for-coverage")

	dataDir := t.TempDir()

	var stdout, stderr bytes.Buffer

	err := cli.Run([]string{
		"engram", "learn",
		"--data-dir", dataDir,
	}, &stdout, &stderr, strings.NewReader("transcript content"))

	// The extractor will fail (no real API), so we expect an error.
	g.Expect(err).To(HaveOccurred())
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

func writeTestTOML(t *testing.T, dir, filename, content string) {
	t.Helper()

	path := filepath.Join(dir, filename)

	err := os.WriteFile(path, []byte(content), 0o640)
	if err != nil {
		t.Fatalf("writeTestTOML: %v", err)
	}
}
