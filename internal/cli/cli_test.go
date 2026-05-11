package cli_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
)

func TestEngramLearn_Fact_EndToEnd(t *testing.T) {
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

	run := exec.Command(binPath, "learn", "fact",
		"--slug", "ctx-fact",
		"--vault", vault,
		"--position", "top",
		"--source", "smoke test",
		"--situation", "concurrent Go code",
		"--subject", "goroutines",
		"--predicate", "leak when",
		"--object", "ctx is ignored",
	)
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
	g.Expect(name).To(MatchRegexp(`^1\.\d{4}-\d{2}-\d{2}\.ctx-fact\.md$`))

	body, err := os.ReadFile(filepath.Join(expectedPath, name))
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(body)).To(ContainSubstring("type: fact"))
	g.Expect(string(body)).To(ContainSubstring(
		"Information learned: when in concurrent Go code, goroutines leak when ctx is ignored.",
	))
}

func TestEngramLearn_Feedback_EndToEnd(t *testing.T) {
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

	run := exec.Command(binPath, "learn", "feedback",
		"--slug", "ctx-rule",
		"--vault", vault,
		"--position", "top",
		"--source", "smoke test",
		"--situation", "writing concurrent Go code",
		"--behavior", "ignoring ctx",
		"--impact", "leaks goroutines",
		"--action", "check ctx.Done()",
		"--relation", "X|adjacent",
	)
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

func TestEngramLearn_MOC_EndToEnd(t *testing.T) {
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

	run := exec.Command(binPath, "learn", "moc",
		"--slug", "ctx-cluster",
		"--vault", vault,
		"--position", "top",
		"--source", "smoke test",
		"--topic", "context handling",
		"--framing", "Notes about how ctx flows through the system.",
	)
	runOut, runErr := run.CombinedOutput()
	g.Expect(runErr).NotTo(HaveOccurred(), "run failed: %s", runOut)

	if runErr != nil {
		return
	}

	mocPath := filepath.Join(vault, "MOCs")
	entries, err := os.ReadDir(mocPath)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(entries).To(HaveLen(1))
	name := entries[0].Name()
	g.Expect(name).To(MatchRegexp(`^1\.\d{4}-\d{2}-\d{2}\.ctx-cluster\.md$`))

	permEntries, err := os.ReadDir(filepath.Join(vault, "Permanent"))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(permEntries).To(BeEmpty(), "MOC must not be written to Permanent/")

	body, err := os.ReadFile(filepath.Join(mocPath, name))
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(body)).To(ContainSubstring("type: moc"))
	g.Expect(string(body)).To(ContainSubstring("topic: context handling"))
	g.Expect(string(body)).To(ContainSubstring("Notes about how ctx flows through the system."))
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
