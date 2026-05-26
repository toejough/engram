package cli_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/onsi/gomega"
	"github.com/toejough/targ"

	"github.com/toejough/engram/internal/cli"
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
		stderr := executeForTest(t, []string{"engram", "nonexistent-subcommand"})
		g.Expect(stderr).NotTo(gomega.BeEmpty())
	})
}

func TestTargets(t *testing.T) {
	t.Parallel()

	t.Run("returns expected target count", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		targets := cli.Targets(&bytes.Buffer{}, &bytes.Buffer{}, func(int) {}, nil)
		// transcript, learn (group), update, embed (group), query
		g.Expect(targets).To(gomega.HaveLen(5))
	})

	t.Run("invokes learn feedback closure", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		vault := t.TempDir()
		g.Expect(os.MkdirAll(filepath.Join(vault, "Permanent"), 0o750)).To(gomega.Succeed())

		stderr := executeForTest(t, []string{
			"engram", "learn", "feedback",
			"--slug", "test-slug",
			"--vault", vault,
			"--source", "agent",
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

		stderr := executeForTest(t, []string{
			"engram", "learn", "fact",
			"--slug", "test-slug",
			"--vault", vault,
			"--source", "agent",
			"--relation", "top",
		})
		_ = stderr
	})

	t.Run("invokes learn episode closure", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		vault := t.TempDir()
		g.Expect(os.MkdirAll(filepath.Join(vault, "Permanent"), 0o750)).To(gomega.Succeed())

		stderr := executeForTest(t, []string{
			"engram", "learn", "episode",
			"--slug", "test-slug",
			"--vault", vault,
			"--source", "agent",
			"--situation", "x",
			"--summary", "x",
			"--outcome", "x",
			"--session", "x",
			"--transcript-range", "2026-05-25T22:00:00Z..2026-05-25T23:00:00Z",
		})
		_ = stderr
	})

	t.Run("learn episode errors when --source is missing", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		vault := t.TempDir()
		g.Expect(os.MkdirAll(filepath.Join(vault, "Permanent"), 0o750)).To(gomega.Succeed())

		targets := cli.Targets(&bytes.Buffer{}, &bytes.Buffer{}, func(int) {}, nil)
		result, err := targ.Execute([]string{
			"engram", "learn", "episode",
			"--slug", "test-slug",
			"--vault", vault,
			"--situation", "x",
			"--summary", "x",
			"--outcome", "x",
			"--session", "x",
			"--transcript-range", "2026-05-25T22:00:00Z..2026-05-25T23:00:00Z",
		}, targets...)
		g.Expect(err).To(gomega.HaveOccurred())
		g.Expect(result.Output).To(gomega.ContainSubstring("source"))
	})

	t.Run("learn episode errors when --session is missing", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		vault := t.TempDir()
		g.Expect(os.MkdirAll(filepath.Join(vault, "Permanent"), 0o750)).To(gomega.Succeed())

		targets := cli.Targets(&bytes.Buffer{}, &bytes.Buffer{}, func(int) {}, nil)
		result, err := targ.Execute([]string{
			"engram", "learn", "episode",
			"--slug", "test-slug",
			"--vault", vault,
			"--source", "src",
			"--situation", "x",
			"--summary", "x",
			"--outcome", "x",
			"--transcript-range", "2026-05-25T22:00:00Z..2026-05-25T23:00:00Z",
		}, targets...)
		g.Expect(err).To(gomega.HaveOccurred())
		g.Expect(result.Output).To(gomega.ContainSubstring("session"))
	})

	t.Run("learn episode errors when --transcript-range is missing", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		vault := t.TempDir()
		g.Expect(os.MkdirAll(filepath.Join(vault, "Permanent"), 0o750)).To(gomega.Succeed())

		targets := cli.Targets(&bytes.Buffer{}, &bytes.Buffer{}, func(int) {}, nil)
		result, err := targ.Execute([]string{
			"engram", "learn", "episode",
			"--slug", "test-slug",
			"--vault", vault,
			"--source", "src",
			"--situation", "x",
			"--summary", "x",
			"--outcome", "x",
			"--session", "x",
		}, targets...)
		g.Expect(err).To(gomega.HaveOccurred())
		g.Expect(result.Output).To(gomega.ContainSubstring("transcript-range"))
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

	t.Run("invokes transcript closure", func(t *testing.T) {
		t.Parallel()

		_ = executeForTest(t, []string{
			"engram", "transcript",
			"--from", "2026-01-01",
			"--to", "2026-12-31",
			"--transcript-dir", t.TempDir(),
		})
	})

	t.Run("invokes update closure in dry-run mode", func(t *testing.T) {
		t.Parallel()

		_ = executeForTest(t, []string{
			"engram", "update", "--dry-run",
		})
	})
}

// TestTargets_EmbedApplyDryRun exercises embed apply closure with --dry-run
// against an empty vault. Lazy embedder's ModelID() doesn't unpack model.
func TestTargets_EmbedApplyDryRun(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(filepath.Join(vault, "Permanent"), 0o750)).To(gomega.Succeed())
	g.Expect(os.MkdirAll(filepath.Join(vault, "MOCs"), 0o750)).To(gomega.Succeed())

	stderr := executeForTest(t, []string{"engram", "embed", "apply", "--dry-run", "--vault", vault})
	g.Expect(stderr).To(gomega.BeEmpty())
}

// TestTargets_EmbedStatus exercises the embed status closure end-to-end
// through Targets() so newOsEmbedDeps wiring is covered. Uses an empty
// vault so the LazyEmbedder's ModelID() path doesn't trigger model unpack.
func TestTargets_EmbedStatus(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(filepath.Join(vault, "Permanent"), 0o750)).To(gomega.Succeed())
	g.Expect(os.MkdirAll(filepath.Join(vault, "MOCs"), 0o750)).To(gomega.Succeed())

	stderr := executeForTest(t, []string{"engram", "embed", "status", "--vault", vault})
	g.Expect(stderr).To(gomega.BeEmpty())
}

// TestTargets_QueryEmptyVault exercises the query closure on an empty
// vault — fast path returns items:[] without invoking the embedder.
func TestTargets_QueryEmptyVault(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(filepath.Join(vault, "Permanent"), 0o750)).To(gomega.Succeed())
	g.Expect(os.MkdirAll(filepath.Join(vault, "MOCs"), 0o750)).To(gomega.Succeed())

	stderr := executeForTest(t, []string{"engram", "query", "anything", "--vault", vault})
	g.Expect(stderr).To(gomega.BeEmpty())
}

// executeForTest runs an engram CLI command through targ, returning stderr content.
// Command-level errors are written to stderr (errHandler contract), not returned as Go errors.
// Targ-level errors (unknown flags, missing required args) are also written to stderr.
func executeForTest(t *testing.T, args []string) string {
	t.Helper()

	var stdout, stderr bytes.Buffer

	targets := cli.Targets(&stdout, &stderr, func(int) {}, nil)

	_, err := targ.Execute(args, targets...)
	if err != nil {
		stderr.WriteString(err.Error())
		stderr.WriteString("\n")
	}

	return stderr.String()
}
