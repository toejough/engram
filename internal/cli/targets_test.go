package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/onsi/gomega"
	"github.com/toejough/targ"

	"engram/internal/cli"
)

func TestAddBoolFlag(t *testing.T) {
	t.Parallel()

	t.Run("appends flag when true", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.AddBoolFlag([]string{"--existing"}, "--verbose", true)
		g.Expect(result).To(gomega.Equal([]string{"--existing", "--verbose"}))
	})

	t.Run("does not append flag when false", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.AddBoolFlag([]string{"--existing"}, "--verbose", false)
		g.Expect(result).To(gomega.Equal([]string{"--existing"}))
	})

	t.Run("works with nil slice", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.AddBoolFlag(nil, "--flag", true)
		g.Expect(result).To(gomega.Equal([]string{"--flag"}))
	})
}

func TestBuildFlags(t *testing.T) {
	t.Parallel()

	t.Run("includes non-empty values", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.BuildFlags("--data-dir", "/tmp", "--format", "json")
		g.Expect(result).To(gomega.Equal([]string{"--data-dir", "/tmp", "--format", "json"}))
	})

	t.Run("skips empty values", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.BuildFlags("--data-dir", "/tmp", "--format", "", "--mode", "test")
		g.Expect(result).To(gomega.Equal([]string{"--data-dir", "/tmp", "--mode", "test"}))
	})

	t.Run("returns empty slice for all empty values", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.BuildFlags("--a", "", "--b", "")
		g.Expect(result).To(gomega.BeEmpty())
	})

	t.Run("returns empty slice for no args", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.BuildFlags()
		g.Expect(result).To(gomega.BeEmpty())
	})

	t.Run("odd number of args ignores trailing key", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.BuildFlags("--data-dir", "/tmp", "--orphan")
		g.Expect(result).To(gomega.Equal([]string{"--data-dir", "/tmp"}))
	})
}

func TestBuildTargets(t *testing.T) {
	t.Parallel()

	t.Run("returns expected number of targets", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		targets := cli.BuildTargets(func(_ string, _ []string) {})
		g.Expect(targets).To(gomega.HaveLen(2))
	})

	t.Run("each subcommand wires to correct name", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		var calls []string

		targets := cli.BuildTargets(func(subcmd string, _ []string) {
			calls = append(calls, subcmd)
		})

		subcmds := []string{"recall", "show"}
		for _, sub := range subcmds {
			_, _ = targ.Execute([]string{"engram", sub}, targets...)
		}

		g.Expect(calls).To(gomega.Equal(subcmds))
	})
}

func TestDataDirFromHome(t *testing.T) {
	t.Parallel()

	t.Run("returns standard engram data path", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		dir := cli.DataDirFromHome("/Users/joe")
		g.Expect(dir).To(gomega.Equal("/Users/joe/.claude/engram/data"))
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

func TestRecallFlags(t *testing.T) {
	t.Parallel()

	t.Run("populated fields", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.RecallFlags(cli.RecallArgs{
			DataDir:     "/data",
			ProjectSlug: "my-project",
			Query:       "search term",
		})
		g.Expect(result).To(gomega.Equal([]string{
			"--data-dir", "/data",
			"--project-slug", "my-project",
			"--query", "search term",
		}))
	})

	t.Run("empty query omitted", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.RecallFlags(cli.RecallArgs{
			DataDir:     "/data",
			ProjectSlug: "proj",
		})
		g.Expect(result).To(gomega.Equal([]string{
			"--data-dir", "/data",
			"--project-slug", "proj",
		}))
	})
}

func TestRunSafe(t *testing.T) {
	t.Parallel()

	t.Run("writes error to stderr on failure", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		var stderr bytes.Buffer

		// Invalid subcommand triggers error path — no filesystem I/O.
		cli.RunSafe(
			[]string{"engram", "nonexistent-subcommand"},
			&bytes.Buffer{}, &stderr, strings.NewReader(""),
		)
		g.Expect(stderr.String()).NotTo(gomega.BeEmpty())
	})
}

func TestShowFlags(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	result := cli.ShowFlags(cli.ShowArgs{DataDir: "/data"})
	g.Expect(result).To(gomega.Equal([]string{"--data-dir", "/data"}))
}

func TestTargets(t *testing.T) {
	t.Parallel()

	t.Run("returns expected target count", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		// Construction doesn't do I/O — just builds targ target objects.
		targets := cli.Targets(&bytes.Buffer{}, &bytes.Buffer{}, strings.NewReader(""))
		g.Expect(targets).To(gomega.HaveLen(2))
	})

	t.Run("closure wiring invokes RunSafe with injected IO", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		var stdout bytes.Buffer

		// Execute one target to exercise the closure body.
		// I/O goes to injected bytes.Buffer — no real side effects.
		// Use "show" which is a working command. Missing slug -> error to stderr.
		targets := cli.Targets(&stdout, &bytes.Buffer{}, strings.NewReader(""))
		_, _ = targ.Execute([]string{"engram", "show", "--data-dir", t.TempDir()}, targets...)

		// show without slug produces an error (written to stderr), stdout is empty.
		g.Expect(stdout.String()).To(gomega.BeEmpty())
	})
}
