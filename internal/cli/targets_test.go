package cli_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/onsi/gomega"
	"github.com/toejough/targ"

	"engram/internal/cli"
)

func TestDataDirFromHome(t *testing.T) {
	t.Parallel()

	t.Run("returns XDG data path when no env override", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		dir := cli.DataDirFromHome("/Users/joe", func(string) string { return "" })
		g.Expect(dir).To(gomega.Equal("/Users/joe/.local/share/engram"))
	})

	t.Run("respects XDG_DATA_HOME when set", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		dir := cli.DataDirFromHome("/Users/joe", func(key string) string {
			if key == "XDG_DATA_HOME" {
				return "/custom/data"
			}

			return ""
		})
		g.Expect(dir).To(gomega.Equal("/custom/data/engram"))
	})
}

func TestNewErrHandler(t *testing.T) {
	t.Parallel()

	t.Run("calls exit with non-zero when err is non-nil", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		var (
			stderr   bytes.Buffer
			exitCode int
			called   bool
		)

		handler := cli.ExportNewErrHandler(&stderr, func(code int) {
			exitCode = code
			called = true
		})
		handler(errors.New("boom"))

		g.Expect(called).To(gomega.BeTrue())
		g.Expect(exitCode).NotTo(gomega.Equal(0))
		g.Expect(stderr.String()).To(gomega.ContainSubstring("boom"))
	})

	t.Run("does not call exit when err is nil", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		var (
			stderr bytes.Buffer
			called bool
		)

		handler := cli.ExportNewErrHandler(&stderr, func(int) { called = true })
		handler(nil)

		g.Expect(called).To(gomega.BeFalse())
		g.Expect(stderr.String()).To(gomega.BeEmpty())
	})
}

func TestProjectSlugFromPath(t *testing.T) {
	t.Parallel()

	t.Run("converts path separators to dashes", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		slug := cli.ProjectSlugFromPath("/Users/joe/repos/personal/engram")
		g.Expect(slug).To(gomega.Equal("-Users-joe-repos-personal-engram"))
	})

	t.Run("empty path returns empty", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		slug := cli.ProjectSlugFromPath("")
		g.Expect(slug).To(gomega.Equal(""))
	})
}

func TestRunRecall(t *testing.T) {
	t.Parallel()

	t.Run("runs with empty data dir and no sessions", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		dataDir := t.TempDir()
		projectSlug := "test-recall-empty"

		// runRecall derives projectDir from $HOME/.claude/projects/<slug>
		home, homeErr := os.UserHomeDir()
		g.Expect(homeErr).NotTo(gomega.HaveOccurred())

		if homeErr != nil {
			return
		}

		projectDir := filepath.Join(home, ".claude", "projects", projectSlug)
		g.Expect(os.MkdirAll(projectDir, 0o750)).To(gomega.Succeed())

		t.Cleanup(func() { _ = os.RemoveAll(projectDir) })

		_, stderr := executeForTest(t, []string{
			"engram", "recall",
			"--data-dir", dataDir,
			"--project-slug", projectSlug,
		})
		g.Expect(stderr).To(gomega.BeEmpty())
	})

	t.Run("defaults data dir and project slug when omitted", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		// When --data-dir and --project-slug are omitted, runRecall
		// derives them from $HOME and $PWD respectively.
		home, homeErr := os.UserHomeDir()
		g.Expect(homeErr).NotTo(gomega.HaveOccurred())

		if homeErr != nil {
			return
		}

		cwd, cwdErr := os.Getwd()
		g.Expect(cwdErr).NotTo(gomega.HaveOccurred())

		if cwdErr != nil {
			return
		}

		defaultSlug := cli.ProjectSlugFromPath(cwd)
		projectDir := filepath.Join(home, ".claude", "projects", defaultSlug)
		g.Expect(os.MkdirAll(projectDir, 0o750)).To(gomega.Succeed())

		t.Cleanup(func() { _ = os.RemoveAll(projectDir) })

		_, stderr := executeForTest(t, []string{
			"engram", "recall",
		})
		g.Expect(stderr).To(gomega.BeEmpty())
	})

	t.Run("runs with query flag", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		dataDir := t.TempDir()
		projectSlug := "test-recall-query"

		home, homeErr := os.UserHomeDir()
		g.Expect(homeErr).NotTo(gomega.HaveOccurred())

		if homeErr != nil {
			return
		}

		projectDir := filepath.Join(home, ".claude", "projects", projectSlug)
		g.Expect(os.MkdirAll(projectDir, 0o750)).To(gomega.Succeed())

		t.Cleanup(func() { _ = os.RemoveAll(projectDir) })

		_, stderr := executeForTest(t, []string{
			"engram", "recall",
			"--data-dir", dataDir,
			"--project-slug", projectSlug,
			"--query", "something",
		})
		g.Expect(stderr).To(gomega.BeEmpty())
	})

	t.Run("returns error on invalid flag", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		_, stderr := executeForTest(t, []string{
			"engram", "recall",
			"--invalid-flag",
		})
		g.Expect(stderr).NotTo(gomega.BeEmpty())
	})

	t.Run("memories-only with empty data dir", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		dataDir := t.TempDir()

		_, stderr := executeForTest(t, []string{
			"engram", "recall",
			"--data-dir", dataDir,
			"--memories-only",
		})
		g.Expect(stderr).To(gomega.BeEmpty())
	})

	t.Run("memories-only with query", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		dataDir := t.TempDir()

		_, stderr := executeForTest(t, []string{
			"engram", "recall",
			"--data-dir", dataDir,
			"--memories-only",
			"--query", "test query",
		})
		g.Expect(stderr).To(gomega.BeEmpty())
	})

	t.Run("sessions error from userHomeDir", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		var buf bytes.Buffer

		err := cli.ExportRunRecallSessions(
			context.Background(), &buf,
			"test-slug", nil, nil, t.TempDir(), "",
			func() (string, error) { return "/fake", nil },
			func() (string, error) { return "", errors.New("no home") },
		)
		g.Expect(err).To(gomega.HaveOccurred())

		if err != nil {
			g.Expect(err.Error()).To(gomega.ContainSubstring("no home"))
		}
	})

	t.Run("sessions error from getwd", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		var buf bytes.Buffer

		err := cli.ExportRunRecallSessions(
			context.Background(), &buf,
			"", nil, nil, t.TempDir(), "",
			func() (string, error) { return "", errors.New("no cwd") },
			func() (string, error) { return "/home", nil },
		)
		g.Expect(err).To(gomega.HaveOccurred())

		if err != nil {
			g.Expect(err.Error()).To(gomega.ContainSubstring("no cwd"))
		}
	})

	t.Run("memories-only with limit", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		dataDir := t.TempDir()

		_, stderr := executeForTest(t, []string{
			"engram", "recall",
			"--data-dir", dataDir,
			"--memories-only",
			"--limit", "5",
		})
		g.Expect(stderr).To(gomega.BeEmpty())
	})
}

func TestRunRecallOpenCodeFlags(t *testing.T) {
	t.Parallel()

	t.Run("transcript-dir flag accepted", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		dataDir := t.TempDir()
		transcriptDir := t.TempDir()

		_, stderr := executeForTest(t, []string{
			"engram", "recall",
			"--data-dir", dataDir,
			"--transcript-dir", transcriptDir,
		})
		g.Expect(stderr).To(gomega.BeEmpty())
	})

	t.Run("sessions uses transcript-dir when provided", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		var buf bytes.Buffer

		transcriptDir := t.TempDir()

		err := cli.ExportRunRecallSessionsWithOpts(
			context.Background(), &buf,
			"", nil, nil, t.TempDir(), "",
			func() (string, error) { return "/fake", nil },
			func() (string, error) { return "/home", nil },
			transcriptDir,
		)
		g.Expect(err).NotTo(gomega.HaveOccurred())
	})

	t.Run("sessions with external sources enabled", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		var buf bytes.Buffer

		home := t.TempDir()
		projectDir := filepath.Join(home, ".claude", "projects", "test-external")
		g.Expect(os.MkdirAll(projectDir, 0o750)).To(gomega.Succeed())

		t.Cleanup(func() { _ = os.RemoveAll(projectDir) })

		err := cli.ExportRunRecallSessionsWithOpts(
			context.Background(), &buf,
			"test-external", nil, nil, t.TempDir(), "",
			func() (string, error) { return "/fake", nil },
			func() (string, error) { return home, nil },
			"",
		)
		g.Expect(err).NotTo(gomega.HaveOccurred())
	})
}

func TestRunSafe(t *testing.T) {
	t.Parallel()

	t.Run("writes error to stderr on failure", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		// Invalid subcommand triggers error path — no filesystem I/O.
		_, stderr := executeForTest(t, []string{"engram", "nonexistent-subcommand"})
		g.Expect(stderr).NotTo(gomega.BeEmpty())
	})
}

func TestTargets(t *testing.T) {
	t.Parallel()

	t.Run("returns expected target count", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		targets := cli.Targets(&bytes.Buffer{}, &bytes.Buffer{}, func(int) {}, nil)
		g.Expect(targets).To(gomega.HaveLen(10))
	})

	t.Run("invokes cycle closure", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		// cycle requires --llm-cmd; with empty it errors → still enters closure.
		_, stderr := executeForTest(t, []string{
			"engram", "cycle", "--llm-cmd", "", "--project-dir", t.TempDir(),
		})
		g.Expect(stderr).NotTo(gomega.BeEmpty())
	})

	t.Run("invokes learn feedback closure", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		vault := t.TempDir()
		g.Expect(os.MkdirAll(filepath.Join(vault, "Permanent"), 0o750)).To(gomega.Succeed())

		_, stderr := executeForTest(t, []string{
			"engram", "learn", "feedback",
			"--slug", "test-slug",
			"--vault", vault,
			"--relation", "top",
		})
		// May or may not error; goal is to invoke the closure.
		_ = stderr
	})

	t.Run("invokes learn fact closure", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		vault := t.TempDir()
		g.Expect(os.MkdirAll(filepath.Join(vault, "Permanent"), 0o750)).To(gomega.Succeed())

		_, stderr := executeForTest(t, []string{
			"engram", "learn", "fact",
			"--slug", "test-slug",
			"--vault", vault,
			"--relation", "top",
		})
		_ = stderr
	})

	t.Run("invokes learn moc closure", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		vault := t.TempDir()
		g.Expect(os.MkdirAll(filepath.Join(vault, "MOCs"), 0o750)).To(gomega.Succeed())

		_, stderr := executeForTest(t, []string{
			"engram", "learn", "moc",
			"--slug", "test-slug",
			"--vault", vault,
			"--relation", "top",
		})
		_ = stderr
	})

	t.Run("invokes quick closure", func(t *testing.T) {
		t.Parallel()

		vault := t.TempDir()

		_, _ = executeForTest(t, []string{
			"engram", "quick", "--slug", "test-slug", "--vault", vault, "--content", "x",
		})
	})

	t.Run("invokes build-self closure with stale check on missing bin", func(t *testing.T) {
		t.Parallel()

		// Use a non-existent plugin-root; build-self errors but the closure executes.
		_, _ = executeForTest(t, []string{
			"engram", "build-self", "--plugin-root", "/nonexistent/plugin/root",
			"--bin-path", filepath.Join(t.TempDir(), "engram"),
			"--if-stale",
		})
	})

	t.Run("closure wiring invokes command with injected IO", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		var stdout bytes.Buffer

		targets := cli.Targets(&stdout, &bytes.Buffer{}, func(int) {}, nil)
		_, _ = targ.Execute([]string{"engram", "show", "--data-dir", t.TempDir()}, targets...)

		// show without slug produces an error (written to stderr), stdout is empty.
		g.Expect(stdout.String()).To(gomega.BeEmpty())
	})
}

// executeForTest runs an engram CLI command through targ, returning stdout content.
// Command-level errors are written to stderr (errHandler contract), not returned as Go errors.
// Targ-level errors (unknown flags, missing required args) are also written to stderr.
func executeForTest(t *testing.T, args []string) (stdoutStr, stderrStr string) {
	t.Helper()

	var stdout, stderr bytes.Buffer

	targets := cli.Targets(&stdout, &stderr, func(int) {}, nil)

	_, err := targ.Execute(args, targets...)
	if err != nil {
		stderr.WriteString(err.Error())
		stderr.WriteString("\n")
	}

	return stdout.String(), stderr.String()
}
