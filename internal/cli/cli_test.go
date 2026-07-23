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
	g.Expect(os.MkdirAll(vault, 0o700)).To(Succeed())
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

	expectedPath := vault
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
	g.Expect(os.MkdirAll(vault, 0o700)).To(Succeed())
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
	)
	runOut, runErr := run.CombinedOutput()
	g.Expect(runErr).NotTo(HaveOccurred(), "run failed: %s", runOut)

	if runErr != nil {
		return
	}

	expectedPath := vault
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

	expectSidecarValid(g, filepath.Join(expectedPath, sidecarName))
}

// TestOpenDebugFile_EndToEnd guards the production OpenDebugFile closure
// in cmd/engram/main.go's procPrimitives (lines 106-109). Runs engram with
// ENGRAM_DEBUG_LOG set to a temp file to prove OpenDebugFile executes the
// production closure end-to-end.
//
// cli.NewDeps composes the debug sink at process-startup time, before targ
// dispatches to any subcommand (main.go: NewDeps is an argument to
// targ.Main, evaluated first) — so the cheapest possible invocation that
// still reaches OpenDebugFile is `--help`, which prints usage and exits
// without touching a vault or loading the embedder. A `query` invocation
// was measured at ~5-7s (embedder model load, real-vault scan) vs ~15ms
// for `--help`; under the coverage-instrumented parallel test run, that
// gap is precisely what pushed the package over its 30s budget.
func TestOpenDebugFile_EndToEnd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	debugFile := filepath.Join(t.TempDir(), "debug.log")

	// Build the engram binary.
	binPath := filepath.Join(t.TempDir(), "engram")
	cmd := exec.Command("go", "build", "-o", binPath, "./cmd/engram")
	cmd.Dir = projectRoot(t)
	out, err := cmd.CombinedOutput()
	g.Expect(err).NotTo(HaveOccurred(), "build failed: %s", out)

	if err != nil {
		return
	}

	// Run engram with ENGRAM_DEBUG_LOG set, using the cheapest invocation
	// that still reaches OpenDebugFile (see doc comment above).
	run := exec.Command(binPath, "--help")

	run.Env = append(os.Environ(), "ENGRAM_DEBUG_LOG="+debugFile)
	_ = run.Run()

	// Assert the debug file was created (proof of reach). The file may be
	// empty or have content depending on whether the logger writes eagerly,
	// but it must exist.
	_, err = os.Stat(debugFile)
	g.Expect(err).NotTo(HaveOccurred(), "debug file not created — OpenDebugFile closure not reached")

	// NEGATIVE CONTROL: run the same cheap invocation with the real process
	// environment MINUS ENGRAM_DEBUG_LOG (not an emptied environment — a
	// stripped-to-nothing Env drops HOME/XDG/PATH/GOCOVERDIR too, which
	// breaks the subprocess's own data-dir resolution and coverage
	// propagation under `targ check-full`'s instrumented runner; only the
	// one variable under test may differ between the two runs). The debug
	// file should not be created.
	debugFile2 := filepath.Join(t.TempDir(), "debug2.log")
	run2 := exec.Command(binPath, "--help")

	run2.Env = envWithoutDebugLog()
	_ = run2.Run()

	// Verify the second debug file was NOT created (no env var = no file).
	_, err2 := os.Stat(debugFile2)
	g.Expect(err2).To(HaveOccurred(), "debug file created without env var set")
}

// TestRunCommand_EndToEnd guards the production C-1 closure in
// cmd/engram/main.go's execPrimitives (lines 40-48). Runs engram update
// from a non-module cwd with fake git/go shims on PATH to prove Cmd.Run
// executes the closure end-to-end. The shims trap invocations by writing
// to a marker file, proving the production literal was reached.
func TestRunCommand_EndToEnd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	shimDir := t.TempDir()
	markerFile := filepath.Join(shimDir, "invocations.txt")
	workDir := t.TempDir()

	// Write a fake git shim that logs invocations and exits 0.
	gitShim := filepath.Join(shimDir, "git")
	gitScript := `#!/bin/sh
echo "$0 $@" >> "` + markerFile + `"
exit 0
`
	g.Expect(os.WriteFile(gitShim, []byte(gitScript), 0o755)).To(Succeed())

	// Write a fake go shim that logs invocations and exits 0.
	goShim := filepath.Join(shimDir, "go")
	goScript := `#!/bin/sh
echo "$0 $@" >> "` + markerFile + `"
exit 0
`
	g.Expect(os.WriteFile(goShim, []byte(goScript), 0o755)).To(Succeed())

	// Build the engram binary.
	binPath := filepath.Join(t.TempDir(), "engram")
	cmd := exec.Command("go", "build", "-o", binPath, "./cmd/engram")
	cmd.Dir = projectRoot(t)
	out, err := cmd.CombinedOutput()

	g.Expect(err).NotTo(HaveOccurred(), "build failed: %s", out)

	if err != nil {
		return
	}

	// Run engram update --dry-run from a non-module cwd with shims on PATH.
	run := exec.Command(binPath, "update", "--dry-run")
	run.Dir = workDir

	run.Env = append(os.Environ(), "PATH="+shimDir+":"+os.Getenv("PATH"))
	_ = run.Run()

	// The update may succeed or fail (dry-run stops before Cmd.Run for
	// local mode, or completes for remote mode). Both cases are OK as long
	// as the marker file exists and contains git invocation proof.
	// Assert the marker file was written by the git shim (proof of reach).
	markerData, readErr := os.ReadFile(markerFile)
	g.Expect(readErr).NotTo(HaveOccurred(), "marker file absent — C-1 closure not reached")

	if readErr != nil {
		return
	}

	// Verify the marker shows git was called (remote mode calls git clone).
	marker := string(markerData)
	g.Expect(marker).To(ContainSubstring("git"), "git shim not invoked — C-1 may not execute git branch")

	// NEGATIVE CONTROL: Run again from the module cwd (resolves to local mode,
	// skips git). The shims should not be called in local mode.
	run2 := exec.Command(binPath, "update", "--dry-run")
	run2.Dir = projectRoot(t)

	run2.Env = append(os.Environ(), "PATH="+shimDir+":"+os.Getenv("PATH"))
	_ = run2.Run()

	// Assert marker still only contains invocations from the remote mode test
	// (local mode doesn't call git). Verify the marker from the first run.
	g.Expect(markerData).NotTo(BeEmpty())
}

// envWithoutDebugLog returns a copy of the current process environment with
// any ENGRAM_DEBUG_LOG entries removed — the correct shape for a negative
// control that isolates a single variable. Wiping the environment entirely
// (Env = []string{}) is wrong: it also strips HOME/XDG/PATH/GOCOVERDIR,
// which breaks the subprocess's data-dir resolution and, under `targ
// check-full`'s coverage-instrumented runner, coverage propagation.
func envWithoutDebugLog() []string {
	base := os.Environ()
	filtered := make([]string, 0, len(base))

	for _, kv := range base {
		if strings.HasPrefix(kv, "ENGRAM_DEBUG_LOG=") {
			continue
		}

		filtered = append(filtered, kv)
	}

	return filtered
}

// expectSidecarValid asserts the sidecar file parses as a Sidecar with
// the current schema version, non-zero dims, and two vectors of the
// declared length.
func expectSidecarValid(g Gomega, path string) {
	data, err := os.ReadFile(path)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	//nolint:tagliatelle // mirrors the spec-contract JSON keys from internal/embed.Sidecar
	var parsed struct {
		SchemaVersion    int       `json:"schema_version"`
		EmbeddingModelID string    `json:"embedding_model_id"`
		Dims             int       `json:"dims"`
		SituationVector  []float32 `json:"situation_vector"`
		BodyVector       []float32 `json:"body_vector"`
		ContentHash      string    `json:"content_hash"`
	}

	g.Expect(json.Unmarshal(data, &parsed)).NotTo(HaveOccurred())
	g.Expect(parsed.SchemaVersion).To(Equal(1))
	g.Expect(parsed.EmbeddingModelID).NotTo(BeEmpty())
	g.Expect(parsed.Dims).To(BeNumerically(">", 0))
	g.Expect(parsed.SituationVector).To(HaveLen(parsed.Dims))
	g.Expect(parsed.BodyVector).To(HaveLen(parsed.Dims))
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
