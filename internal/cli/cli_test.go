package cli_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
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

	mdName, sidecarName := splitMdAndSidecar(entries)
	g.Expect(mdName).To(MatchRegexp(`^1\.\d{4}-\d{2}-\d{2}\.ctx-fact\.md$`))
	g.Expect(sidecarName).To(MatchRegexp(`^1\.\d{4}-\d{2}-\d{2}\.ctx-fact\.vec\.json$`))

	body, err := os.ReadFile(filepath.Join(expectedPath, mdName))
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(body)).To(ContainSubstring("type: fact"))
	g.Expect(string(body)).To(ContainSubstring(
		"Information learned: when in concurrent Go code, goroutines leak when ctx is ignored.",
	))

	expectSidecarValid(g, filepath.Join(expectedPath, sidecarName))
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

	mdName, sidecarName := splitMdAndSidecar(entries)
	g.Expect(mdName).To(MatchRegexp(`^1\.\d{4}-\d{2}-\d{2}\.ctx-rule\.md$`))
	g.Expect(sidecarName).To(MatchRegexp(`^1\.\d{4}-\d{2}-\d{2}\.ctx-rule\.vec\.json$`))

	body, err := os.ReadFile(filepath.Join(expectedPath, mdName))
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(body)).To(ContainSubstring("type: feedback"))
	g.Expect(string(body)).
		To(ContainSubstring("Lesson learned: when writing concurrent Go code, check ctx.Done()."))
	g.Expect(string(body)).To(ContainSubstring("Related to:"))

	expectSidecarValid(g, filepath.Join(expectedPath, sidecarName))
}

func TestEngramRecall_AnchorsEndToEnd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	mocsDir := filepath.Join(vault, "MOCs")
	g.Expect(os.MkdirAll(mocsDir, 0o700)).To(Succeed())
	g.Expect(os.WriteFile(
		filepath.Join(mocsDir, "1.2026-03-15.alpha.md"),
		[]byte("# alpha MOC\nFraming prose with [[1a.2026-03-16.beta]]"),
		0o600,
	)).To(Succeed())

	binPath := filepath.Join(t.TempDir(), "engram")
	build := exec.Command("go", "build", "-o", binPath, "./cmd/engram")
	build.Dir = projectRoot(t)
	buildOut, buildErr := build.CombinedOutput()
	g.Expect(buildErr).NotTo(HaveOccurred(), "build failed: %s", buildOut)

	if buildErr != nil {
		return
	}

	run := exec.Command(binPath, "recall", "--vault", vault)
	runOut, runErr := run.CombinedOutput()
	g.Expect(runErr).NotTo(HaveOccurred(), "run failed: %s", runOut)

	if runErr != nil {
		return
	}

	g.Expect(string(runOut)).To(ContainSubstring("1.2026-03-15.alpha"))
}

func TestEngramRecall_RecentEndToEnd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	permDir := filepath.Join(vault, "Permanent")
	g.Expect(os.MkdirAll(permDir, 0o700)).To(Succeed())

	for _, name := range []string{
		"1.2026-01-01.old.md",
		"2.2026-05-01.new.md",
	} {
		g.Expect(os.WriteFile(filepath.Join(permDir, name), []byte("body"), 0o600)).To(Succeed())
	}

	binPath := filepath.Join(t.TempDir(), "engram")
	build := exec.Command("go", "build", "-o", binPath, "./cmd/engram")
	build.Dir = projectRoot(t)
	buildOut, buildErr := build.CombinedOutput()
	g.Expect(buildErr).NotTo(HaveOccurred(), "build failed: %s", buildOut)

	if buildErr != nil {
		return
	}

	run := exec.Command(binPath, "recall", "--vault", vault, "--recent", "--limit", "1")
	runOut, runErr := run.CombinedOutput()
	g.Expect(runErr).NotTo(HaveOccurred(), "run failed: %s", runOut)

	if runErr != nil {
		return
	}

	g.Expect(string(runOut)).To(ContainSubstring("2.2026-05-01.new"))
	g.Expect(string(runOut)).NotTo(ContainSubstring("1.2026-01-01.old"))
}

func TestEngramTranscript_DateRangeEndToEnd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	transcriptDir := t.TempDir()
	transcriptPath := filepath.Join(transcriptDir, "session-abc.jsonl")
	body := `{"type":"user","message":{"content":"hello-smoke-test"}}` + "\n"
	g.Expect(os.WriteFile(transcriptPath, []byte(body), 0o600)).To(Succeed())

	binPath := filepath.Join(t.TempDir(), "engram")
	build := exec.Command("go", "build", "-o", binPath, "./cmd/engram")
	build.Dir = projectRoot(t)
	buildOut, buildErr := build.CombinedOutput()
	g.Expect(buildErr).NotTo(HaveOccurred(), "build failed: %s", buildOut)

	if buildErr != nil {
		return
	}

	// Use a wide date range so the file mtime (today) falls in.
	run := exec.Command(binPath, "transcript",
		"--from", "2020-01-01",
		"--to", "2099-12-31",
		"--transcript-dir", transcriptDir,
	)
	runOut, runErr := run.CombinedOutput()
	g.Expect(runErr).NotTo(HaveOccurred(), "run failed: %s", runOut)

	if runErr != nil {
		return
	}

	g.Expect(string(runOut)).To(ContainSubstring("hello-smoke-test"))
}

// expectSidecarValid asserts the sidecar file parses as a Sidecar with
// non-zero dims and a vector of the declared length.
func expectSidecarValid(g Gomega, path string) {
	data, err := os.ReadFile(path)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	//nolint:tagliatelle // mirrors the spec-contract JSON keys from internal/embed.Sidecar
	var parsed struct {
		EmbeddingModelID string    `json:"embedding_model_id"`
		Dims             int       `json:"dims"`
		Vector           []float32 `json:"vector"`
		ContentHash      string    `json:"content_hash"`
	}

	g.Expect(json.Unmarshal(data, &parsed)).NotTo(HaveOccurred())
	g.Expect(parsed.EmbeddingModelID).NotTo(BeEmpty())
	g.Expect(parsed.Dims).To(BeNumerically(">", 0))
	g.Expect(parsed.Vector).To(HaveLen(parsed.Dims))
	g.Expect(parsed.ContentHash).To(HavePrefix("sha256:"))
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

// splitMdAndSidecar returns the .md and .vec.json basenames found in
// entries. Tests use it to verify both files exist after a learn with
// auto-embed.
func splitMdAndSidecar(entries []os.DirEntry) (md, sidecar string) {
	for _, entry := range entries {
		name := entry.Name()

		switch {
		case strings.HasSuffix(name, ".vec.json"):
			sidecar = name
		case strings.HasSuffix(name, ".md"):
			md = name
		}
	}

	return md, sidecar
}
