package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/onsi/gomega"
	"github.com/toejough/targ"

	"engram/internal/cli"
)

// executeForTest runs an engram CLI command through targ, returning stdout content.
// Command-level errors are written to stderr (RunSafe contract), not returned as Go errors.
// Targ-level errors (unknown flags, missing required args) are also written to stderr.
func executeForTest(t *testing.T, args []string) (stdoutStr, stderrStr string) {
	t.Helper()

	var stdout, stderr bytes.Buffer

	targets := cli.Targets(&stdout, &stderr, strings.NewReader(""))
	_, err := targ.Execute(args, targets...)

	if err != nil {
		stderr.WriteString(err.Error())
		stderr.WriteString("\n")
	}

	return stdout.String(), stderr.String()
}

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

func TestAddIntFlag(t *testing.T) {
	t.Parallel()

	t.Run("appends flag when non-zero", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.AddIntFlag([]string{"--existing"}, "--limit", 5)
		g.Expect(result).To(gomega.Equal([]string{"--existing", "--limit", "5"}))
	})

	t.Run("does not append flag when zero", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.AddIntFlag([]string{"--existing"}, "--limit", 0)
		g.Expect(result).To(gomega.Equal([]string{"--existing"}))
	})

	t.Run("works with nil slice", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.AddIntFlag(nil, "--limit", 10)
		g.Expect(result).To(gomega.Equal([]string{"--limit", "10"}))
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
		g.Expect(targets).To(gomega.HaveLen(5))
	})

	t.Run("each subcommand wires to correct name", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		var calls []string

		targets := cli.BuildTargets(func(subcmd string, _ []string) {
			calls = append(calls, subcmd)
		})

		subcmds := []string{"recall", "show", "list"}
		for _, sub := range subcmds {
			_, _ = targ.Execute([]string{"engram", sub}, targets...)
		}

		// Learn subcommands need the group prefix
		_, _ = targ.Execute([]string{
			"engram", "learn", "feedback",
			"--source", "human",
		}, targets...)
		_, _ = targ.Execute([]string{
			"engram", "learn", "fact",
			"--source", "agent",
		}, targets...)
		_, _ = targ.Execute([]string{
			"engram", "update",
			"--name", "test",
		}, targets...)

		g.Expect(calls).To(gomega.Equal([]string{
			"recall", "show", "list",
			"learn feedback", "learn fact", "update",
		}))
	})
}

func TestBuildTargets_LearnFactWiring(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	var capturedSubcmd string

	var capturedFlags []string

	targets := cli.BuildTargets(func(subcmd string, flags []string) {
		capturedSubcmd = subcmd
		capturedFlags = flags
	})

	_, err := targ.Execute([]string{
		"engram", "learn", "fact",
		"--situation", "Go projects",
		"--source", "agent",
		"--subject", "engram",
		"--predicate", "uses",
		"--object", "targ",
		"--data-dir", "/tmp/test",
	}, targets...)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(capturedSubcmd).To(gomega.Equal("learn fact"))
	g.Expect(capturedFlags).To(gomega.ContainElements(
		"--subject", "engram",
		"--predicate", "uses",
		"--object", "targ",
	))
}

func TestBuildTargets_LearnFeedbackWiring(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	var capturedSubcmd string

	var capturedFlags []string

	targets := cli.BuildTargets(func(subcmd string, flags []string) {
		capturedSubcmd = subcmd
		capturedFlags = flags
	})

	_, err := targ.Execute([]string{
		"engram", "learn", "feedback",
		"--situation", "test-sit",
		"--source", "human",
		"--behavior", "test-beh",
		"--impact", "test-imp",
		"--action", "test-act",
		"--no-dup-check",
		"--data-dir", "/tmp/test",
	}, targets...)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(capturedSubcmd).To(gomega.Equal("learn feedback"))
	g.Expect(capturedFlags).To(gomega.ContainElements(
		"--situation", "test-sit",
		"--source", "human",
		"--behavior", "test-beh",
		"--data-dir", "/tmp/test",
	))
	g.Expect(capturedFlags).To(gomega.ContainElement("--no-dup-check"))
}

func TestBuildTargets_UpdateWiring(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	var capturedSubcmd string

	var capturedFlags []string

	targets := cli.BuildTargets(func(subcmd string, flags []string) {
		capturedSubcmd = subcmd
		capturedFlags = flags
	})

	_, err := targ.Execute([]string{
		"engram", "update",
		"--name", "test-mem",
		"--situation", "new-sit",
		"--data-dir", "/tmp/test",
	}, targets...)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(capturedSubcmd).To(gomega.Equal("update"))
	g.Expect(capturedFlags).To(gomega.ContainElements(
		"--name", "test-mem",
		"--situation", "new-sit",
	))
}

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

func TestListFlags(t *testing.T) {
	t.Parallel()

	t.Run("populated data dir", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.ListFlags(cli.ListArgs{DataDir: "/data"})
		g.Expect(result).To(gomega.Equal([]string{"--data-dir", "/data"}))
	})

	t.Run("empty data dir omitted", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.ListFlags(cli.ListArgs{})
		g.Expect(result).To(gomega.BeEmpty())
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

	t.Run("memories-only flag included when true", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.RecallFlags(cli.RecallArgs{
			DataDir:      "/data",
			Query:        "test",
			MemoriesOnly: true,
		})
		g.Expect(result).To(gomega.ContainElement("--memories-only"))
		g.Expect(result).To(gomega.ContainElements("--query", "test"))
	})

	t.Run("memories-only flag omitted when false", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.RecallFlags(cli.RecallArgs{
			DataDir: "/data",
			Query:   "test",
		})
		g.Expect(result).NotTo(gomega.ContainElement("--memories-only"))
	})

	t.Run("limit flag included when non-zero", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.RecallFlags(cli.RecallArgs{
			DataDir: "/data",
			Query:   "test",
			Limit:   5,
		})
		g.Expect(result).To(gomega.ContainElements("--limit", "5"))
	})

	t.Run("limit flag omitted when zero", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.RecallFlags(cli.RecallArgs{
			DataDir: "/data",
			Query:   "test",
		})
		g.Expect(result).NotTo(gomega.ContainElement("--limit"))
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

		targets := cli.Targets(&bytes.Buffer{}, &bytes.Buffer{}, strings.NewReader(""))
		g.Expect(targets).To(gomega.HaveLen(5))
	})

	t.Run("closure wiring invokes RunSafe with injected IO", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		var stdout bytes.Buffer

		targets := cli.Targets(&stdout, &bytes.Buffer{}, strings.NewReader(""))
		_, _ = targ.Execute([]string{"engram", "show", "--data-dir", t.TempDir()}, targets...)

		// show without slug produces an error (written to stderr), stdout is empty.
		g.Expect(stdout.String()).To(gomega.BeEmpty())
	})
}
