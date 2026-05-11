package cli_test

import (
	"bytes"
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
		g.Expect(targets).To(gomega.HaveLen(9))
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

	t.Run("learn feedback errors when --source is missing", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		vault := t.TempDir()
		g.Expect(os.MkdirAll(filepath.Join(vault, "Permanent"), 0o750)).To(gomega.Succeed())

		targets := cli.Targets(&bytes.Buffer{}, &bytes.Buffer{}, func(int) {}, nil)
		result, err := targ.Execute([]string{
			"engram", "learn", "feedback",
			"--slug", "test-slug",
			"--vault", vault,
			"--relation", "top",
			"--situation", "x", "--behavior", "x", "--impact", "x", "--action", "x",
		}, targets...)
		g.Expect(err).To(gomega.HaveOccurred())
		g.Expect(result.Output).To(gomega.ContainSubstring("source"))
	})

	t.Run("learn fact errors when --source is missing", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		vault := t.TempDir()
		g.Expect(os.MkdirAll(filepath.Join(vault, "Permanent"), 0o750)).To(gomega.Succeed())

		targets := cli.Targets(&bytes.Buffer{}, &bytes.Buffer{}, func(int) {}, nil)
		result, err := targ.Execute([]string{
			"engram", "learn", "fact",
			"--slug", "test-slug",
			"--vault", vault,
			"--relation", "top",
			"--situation", "x", "--subject", "x", "--predicate", "x", "--object", "x",
		}, targets...)
		g.Expect(err).To(gomega.HaveOccurred())
		g.Expect(result.Output).To(gomega.ContainSubstring("source"))
	})

	t.Run("learn moc errors when --source is missing", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		vault := t.TempDir()
		g.Expect(os.MkdirAll(filepath.Join(vault, "MOCs"), 0o750)).To(gomega.Succeed())

		targets := cli.Targets(&bytes.Buffer{}, &bytes.Buffer{}, func(int) {}, nil)
		result, err := targ.Execute([]string{
			"engram", "learn", "moc",
			"--slug", "test-slug",
			"--vault", vault,
			"--relation", "top",
			"--topic", "x",
		}, targets...)
		g.Expect(err).To(gomega.HaveOccurred())
		g.Expect(result.Output).To(gomega.ContainSubstring("source"))
	})

	t.Run("invokes recall closure", func(t *testing.T) {
		t.Parallel()

		vault := t.TempDir()

		_, _ = executeForTest(t, []string{
			"engram", "recall", "--vault", vault,
		})
	})

	t.Run("invokes list closure", func(t *testing.T) {
		t.Parallel()

		_, _ = executeForTest(t, []string{
			"engram", "list", "--data-dir", t.TempDir(),
		})
	})

	t.Run("invokes reminder closure", func(t *testing.T) {
		t.Parallel()

		_, _ = executeForTest(t, []string{
			"engram", "reminder",
		})
	})

	t.Run("invokes update closure", func(t *testing.T) {
		t.Parallel()

		_, _ = executeForTest(t, []string{
			"engram", "update", "--name", "x", "--data-dir", t.TempDir(),
		})
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
