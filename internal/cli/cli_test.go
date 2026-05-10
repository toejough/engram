package cli_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
)

func TestEngramPromote_Feedback_EndToEnd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(filepath.Join(vault, "Permanent"), 0o700)).To(Succeed())
	g.Expect(os.MkdirAll(filepath.Join(vault, "MOCs"), 0o700)).To(Succeed())

	binPath := filepath.Join(t.TempDir(), "engram")
	cmd := exec.Command("go", "build", "-o", binPath, "./cmd/engram")
	cmd.Dir = projectRoot(t)
	out, err := cmd.CombinedOutput()
	g.Expect(err).NotTo(HaveOccurred(), "build failed: %s", out)

	if err != nil {
		return
	}

	run := exec.Command(binPath, "promote", "feedback",
		"--slug", "ctx-rule",
		"--vault", vault,
		"--relation", "top",
		"--source", "smoke test",
		"--situation", "writing concurrent Go code",
		"--behavior", "ignoring ctx",
		"--impact", "leaks goroutines",
		"--action", "check ctx.Done()",
	)
	run.Stdin = strings.NewReader("Related to:\n- [[X]] — adjacent.\n")
	runOut, runErr := run.CombinedOutput()
	g.Expect(runErr).NotTo(HaveOccurred(), "run failed: %s", runOut)

	if runErr != nil {
		return
	}

	expectedPath := filepath.Join(vault, "Permanent")
	entries, err := os.ReadDir(expectedPath)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(entries).To(HaveLen(1))
	name := entries[0].Name()
	g.Expect(name).To(MatchRegexp(`^1\.\d{4}-\d{2}-\d{2}\.ctx-rule\.md$`))

	body, err := os.ReadFile(filepath.Join(expectedPath, name))
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(body)).To(ContainSubstring("type: feedback"))
	g.Expect(string(body)).To(ContainSubstring("Lesson learned: when writing concurrent Go code, check ctx.Done()."))
	g.Expect(string(body)).To(ContainSubstring("Related to:"))
}

func TestRequireLLMCmd_ErrorsWhenMissing(t *testing.T) {
	g := NewWithT(t)

	t.Setenv("ENGRAM_LLM_CMD", "")

	err := cli.ExportRequireLLMCmd("")
	g.Expect(err).To(MatchError(ContainSubstring("llm-cmd is required")))
	g.Expect(err).To(MatchError(ContainSubstring("ENGRAM_LLM_CMD")))
}

func TestRequireLLMCmd_OkWhenFlagSet(t *testing.T) {
	g := NewWithT(t)

	t.Setenv("ENGRAM_LLM_CMD", "")

	err := cli.ExportRequireLLMCmd("opencode run -m foo")
	g.Expect(err).NotTo(HaveOccurred())
}

func TestResolveLLMCmd_EmptyWhenNeitherSet(t *testing.T) {
	g := NewWithT(t)

	t.Setenv("ENGRAM_LLM_CMD", "")

	got := cli.ExportResolveLLMCmd("")
	g.Expect(got).To(Equal(""))
}

func TestResolveLLMCmd_FallsBackToEnv(t *testing.T) {
	g := NewWithT(t)

	t.Setenv("ENGRAM_LLM_CMD", "from-env")

	got := cli.ExportResolveLLMCmd("")
	g.Expect(got).To(Equal("from-env"))
}

func TestResolveLLMCmd_PrefersFlagOverEnv(t *testing.T) {
	g := NewWithT(t)

	t.Setenv("ENGRAM_LLM_CMD", "from-env")

	got := cli.ExportResolveLLMCmd("from-flag")
	g.Expect(got).To(Equal("from-flag"))
}

func projectRoot(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// internal/cli → ../..
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}
