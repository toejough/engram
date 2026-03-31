package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/onsi/gomega"
	"github.com/toejough/targ"

	"engram/internal/cli"
)

func TestAdaptFlags(t *testing.T) {
	t.Parallel()

	t.Run("all fields set", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.AdaptFlags(cli.AdaptArgs{
			DataDir: "/data",
			Approve: "pol-001",
			Reject:  "pol-002",
			Retire:  "pol-003",
			Status:  true,
		})
		g.Expect(result).To(gomega.ContainElements(
			"--data-dir", "/data",
			"--approve", "pol-001",
			"--reject", "pol-002",
			"--retire", "pol-003",
			"--status",
		))
	})

	t.Run("empty fields omits optional flags", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.AdaptFlags(cli.AdaptArgs{})
		g.Expect(result).NotTo(gomega.ContainElement("--status"))
		g.Expect(result).NotTo(gomega.ContainElement("--approve"))
	})
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

func TestApplyProposalFlags(t *testing.T) {
	t.Parallel()

	t.Run("populated fields", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.ApplyProposalFlags(cli.ApplyProposalArgs{
			DataDir:  "/data",
			Action:   "rewrite",
			Memory:   "/mem/foo.toml",
			Fields:   `{"title":"new"}`,
			Keywords: "a,b",
		})
		g.Expect(result).To(gomega.Equal([]string{
			"--data-dir", "/data",
			"--action", "rewrite",
			"--memory", "/mem/foo.toml",
			"--fields", `{"title":"new"}`,
			"--keywords", "a,b",
		}))
	})

	t.Run("empty fields skipped", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.ApplyProposalFlags(cli.ApplyProposalArgs{})
		g.Expect(result).To(gomega.BeEmpty())
	})

	t.Run("zero level omitted", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.ApplyProposalFlags(cli.ApplyProposalArgs{
			DataDir: "/data",
			Action:  "remove",
			Memory:  "/mem/foo.toml",
		})
		g.Expect(result).To(gomega.Equal([]string{
			"--data-dir", "/data",
			"--action", "remove",
			"--memory", "/mem/foo.toml",
		}))
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
		// Individual commands + registry group.
		g.Expect(len(targets)).To(gomega.BeNumerically(">=", 9))
	})

	t.Run("each subcommand wires to correct name", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		var calls []string

		targets := cli.BuildTargets(func(subcmd string, _ []string) {
			calls = append(calls, subcmd)
		})

		subcmds := []string{
			"correct", "review",
			"maintain", "surface", "instruct",
			"feedback", "refine", "show",
			"apply-proposal", "migrate-slugs",
		}
		for _, sub := range subcmds {
			_, _ = targ.Execute([]string{"engram", sub}, targets...)
		}

		g.Expect(calls).To(gomega.Equal(subcmds))
	})
}

func TestCorrectFlags(t *testing.T) {
	t.Parallel()

	t.Run("populated fields", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.CorrectFlags(cli.CorrectArgs{
			Message:        "fix this",
			DataDir:        "/data",
			TranscriptPath: "/transcript",
			ProjectSlug:    "my-project",
		})
		g.Expect(result).To(gomega.Equal([]string{
			"--message", "fix this",
			"--data-dir", "/data",
			"--transcript-path", "/transcript",
			"--project-slug", "my-project",
		}))
	})

	t.Run("empty fields skipped", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.CorrectFlags(cli.CorrectArgs{})
		g.Expect(result).To(gomega.BeEmpty())
	})

	t.Run("partial fields", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.CorrectFlags(cli.CorrectArgs{Message: "fix this"})
		g.Expect(result).To(gomega.Equal([]string{"--message", "fix this"}))
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

func TestFeedbackFlags(t *testing.T) {
	t.Parallel()

	t.Run("relevant and used", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.FeedbackFlags(cli.FeedbackArgs{DataDir: "/data", Relevant: true, Used: true})
		g.Expect(result).To(gomega.Equal([]string{"--data-dir", "/data", "--relevant", "--used"}))
	})

	t.Run("irrelevant only", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.FeedbackFlags(cli.FeedbackArgs{DataDir: "/data", Irrelevant: true})
		g.Expect(result).To(gomega.Equal([]string{"--data-dir", "/data", "--irrelevant"}))
	})
}


func TestInstructFlags(t *testing.T) {
	t.Parallel()

	t.Run("populated fields", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.InstructFlags(cli.InstructArgs{DataDir: "/data", ProjectDir: "/project"})
		g.Expect(result).
			To(gomega.Equal([]string{"--data-dir", "/data", "--project-dir", "/project"}))
	})

	t.Run("empty fields skipped", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.InstructFlags(cli.InstructArgs{})
		g.Expect(result).To(gomega.BeEmpty())
	})

	t.Run("partial fields", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.InstructFlags(cli.InstructArgs{ProjectDir: "/project"})
		g.Expect(result).To(gomega.Equal([]string{"--project-dir", "/project"}))
	})
}


func TestMaintainFlags(t *testing.T) {
	t.Parallel()

	t.Run("all flags set", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.MaintainFlags(cli.MaintainArgs{
			DataDir:   "/data",
			Proposals: "/proposals.json",
			Apply:     true,
			Yes:       true,
		})
		g.Expect(result).To(gomega.Equal([]string{
			"--data-dir", "/data",
			"--proposals", "/proposals.json",
			"--apply", "--yes",
		}))
	})

	t.Run("bools false", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.MaintainFlags(cli.MaintainArgs{DataDir: "/data"})
		g.Expect(result).To(gomega.Equal([]string{"--data-dir", "/data"}))
	})

	t.Run("empty fields skipped", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.MaintainFlags(cli.MaintainArgs{})
		g.Expect(result).To(gomega.BeEmpty())
	})

	t.Run("partial apply true yes false", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.MaintainFlags(cli.MaintainArgs{
			DataDir:   "/data",
			Proposals: "/p.json",
			Apply:     true,
			Yes:       false,
		})
		g.Expect(result).To(gomega.Equal([]string{
			"--data-dir", "/data",
			"--proposals", "/p.json",
			"--apply",
		}))
	})

	t.Run("partial apply false yes true", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.MaintainFlags(cli.MaintainArgs{
			DataDir: "/data",
			Apply:   false,
			Yes:     true,
		})
		g.Expect(result).To(gomega.Equal([]string{
			"--data-dir", "/data",
			"--yes",
		}))
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

func TestRefineFlags(t *testing.T) {
	t.Parallel()

	t.Run("populated fields", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.RefineFlags(cli.RefineArgs{DataDir: "/data", DryRun: true})
		g.Expect(result).To(gomega.Equal([]string{"--data-dir", "/data", "--dry-run"}))
	})

	t.Run("dry-run false omits flag", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.RefineFlags(cli.RefineArgs{DataDir: "/data"})
		g.Expect(result).To(gomega.Equal([]string{"--data-dir", "/data"}))
	})

	t.Run("empty fields skipped", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.RefineFlags(cli.RefineArgs{})
		g.Expect(result).To(gomega.BeEmpty())
	})
}

func TestReviewFlags(t *testing.T) {
	t.Parallel()

	t.Run("populated fields", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.ReviewFlags(cli.ReviewArgs{DataDir: "/data", Format: "json"})
		g.Expect(result).To(gomega.Equal([]string{"--data-dir", "/data", "--format", "json"}))
	})

	t.Run("empty fields skipped", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.ReviewFlags(cli.ReviewArgs{})
		g.Expect(result).To(gomega.BeEmpty())
	})

	t.Run("partial fields", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.ReviewFlags(cli.ReviewArgs{Format: "json"})
		g.Expect(result).To(gomega.Equal([]string{"--format", "json"}))
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

func TestSurfaceFlags(t *testing.T) {
	t.Parallel()

	t.Run("populated fields", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.SurfaceFlags(cli.SurfaceArgs{
			Mode:    "prompt",
			DataDir: "/data",
			Message: "hello",
			Format:  "json",
		})
		g.Expect(result).To(gomega.Equal([]string{
			"--mode", "prompt",
			"--data-dir", "/data",
			"--message", "hello",
			"--format", "json",
		}))
	})

	t.Run("partial fields", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.SurfaceFlags(cli.SurfaceArgs{Mode: "prompt", DataDir: "/data"})
		g.Expect(result).
			To(gomega.Equal([]string{"--mode", "prompt", "--data-dir", "/data"}))
	})

	t.Run("empty fields skipped", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.SurfaceFlags(cli.SurfaceArgs{})
		g.Expect(result).To(gomega.BeEmpty())
	})

	t.Run("stop mode with transcript path and session ID", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.SurfaceFlags(cli.SurfaceArgs{
			Mode:           "stop",
			DataDir:        "/data",
			TranscriptPath: "/t.jsonl",
			SessionID:      "s1",
			Format:         "json",
		})
		g.Expect(result).To(gomega.Equal([]string{
			"--mode", "stop",
			"--data-dir", "/data",
			"--format", "json",
			"--transcript-path", "/t.jsonl",
			"--session-id", "s1",
		}))
	})
}

func TestTargets(t *testing.T) {
	t.Parallel()

	t.Run("returns expected target count", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		// Construction doesn't do I/O — just builds targ target objects.
		targets := cli.Targets(&bytes.Buffer{}, &bytes.Buffer{}, strings.NewReader(""))
		g.Expect(len(targets)).To(gomega.BeNumerically(">=", 10))
	})

	t.Run("closure wiring invokes RunSafe with injected IO", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		var stdout bytes.Buffer

		// Execute one target to exercise the closure body.
		// I/O goes to injected bytes.Buffer — no real side effects.
		// Use "show" which is a working command. Missing slug → error to stderr.
		targets := cli.Targets(&stdout, &bytes.Buffer{}, strings.NewReader(""))
		_, _ = targ.Execute([]string{"engram", "show", "--data-dir", t.TempDir()}, targets...)

		// show without slug produces an error (written to stderr), stdout is empty.
		g.Expect(stdout.String()).To(gomega.BeEmpty())
	})
}
