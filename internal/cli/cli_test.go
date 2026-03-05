package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
)

func TestRun_CorrectMissingFlags(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var buf bytes.Buffer

	err := cli.Run([]string{"engram", "correct"}, &buf, nil)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("--message"))
	}
}

func TestRun_NoArgs(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var buf bytes.Buffer

	err := cli.Run([]string{"engram"}, &buf, nil)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("usage"))
	}
}

func TestRun_UnknownCommand(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var buf bytes.Buffer

	err := cli.Run([]string{"engram", "bogus"}, &buf, nil)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("unknown command"))
	}
}

// T-18: `correct` subcommand with no API key returns error
func TestT18_CorrectSubcommandWithoutAPIKeyReturnsError(t *testing.T) {
	// Cannot use t.Parallel() — t.Setenv mutates process environment.
	g := NewGomegaWithT(t)

	dataDir := filepath.Join(t.TempDir(), "data")

	// Ensure no API token is set.
	t.Setenv("ENGRAM_API_TOKEN", "")

	var buf bytes.Buffer

	err := cli.Run([]string{
		"engram", "correct",
		"--message", "remember to use targ",
		"--data-dir", dataDir,
	}, &buf, nil)
	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("no API token"))
	}
}

// T-19: `correct` with non-matching message produces empty stdout
func TestT19_CorrectWithNonMatchingMessageProducesEmptyStdout(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := filepath.Join(t.TempDir(), "data")

	var buf bytes.Buffer

	err := cli.Run([]string{
		"engram", "correct",
		"--message", "hello world",
		"--data-dir", dataDir,
	}, &buf, nil)
	g.Expect(err).NotTo(HaveOccurred())

	// Verify stdout is empty
	g.Expect(buf.String()).To(BeEmpty())

	// Verify no memories directory was created
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

	writeMemoryTOML(t, memoriesDir, "test-memory.toml", `title = "Test Memory"
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

	var buf bytes.Buffer

	err = cli.Run([]string{
		"engram", "surface",
		"--mode", "session-start",
		"--data-dir", dataDir,
	}, &buf, nil)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := buf.String()
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

	writeMemoryTOML(t, memoriesDir, "commit-rules.toml", `title = "Commit Rules"
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

	var buf bytes.Buffer

	err = cli.Run([]string{
		"engram", "surface",
		"--mode", "prompt",
		"--message", "I want to commit this",
		"--data-dir", dataDir,
	}, &buf, nil)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	output := buf.String()
	g.Expect(output).To(ContainSubstring("[engram] Relevant memories:"))
	g.Expect(output).To(ContainSubstring("Commit Rules"))
	g.Expect(output).To(ContainSubstring("commit"))
}

// T-42: Mode tool routes to enforcement pipeline
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

	writeMemoryTOML(t, memoriesDir, "commit-rules.toml", `title = "Commit Rules"
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

	// Without API token, enforcement should degrade gracefully.
	t.Setenv("ENGRAM_API_TOKEN", "")

	var buf bytes.Buffer

	err = cli.Run([]string{
		"engram", "surface",
		"--mode", "tool",
		"--tool-name", "Bash",
		"--tool-input", `{"command": "git commit -m fix"}`,
		"--data-dir", dataDir,
	}, &buf, nil)

	g.Expect(err).NotTo(HaveOccurred())
	// With no token, enforcement is skipped — tool allowed silently.
	g.Expect(buf.String()).To(BeEmpty())
}

func writeMemoryTOML(t *testing.T, dir, filename, content string) {
	t.Helper()

	path := filepath.Join(dir, filename)

	err := os.WriteFile(path, []byte(content), 0o640)
	if err != nil {
		t.Fatalf("writeMemoryTOML: %v", err)
	}
}
