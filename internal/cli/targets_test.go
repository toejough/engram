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

func TestAuditFlags(t *testing.T) {
	t.Parallel()

	t.Run("populated fields", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.AuditFlags(cli.AuditArgs{DataDir: "/data", Timestamp: "2024-01-01T00:00:00Z"})
		g.Expect(result).
			To(gomega.Equal([]string{"--data-dir", "/data", "--timestamp", "2024-01-01T00:00:00Z"}))
	})

	t.Run("empty fields skipped", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.AuditFlags(cli.AuditArgs{})
		g.Expect(result).To(gomega.BeEmpty())
	})

	t.Run("partial fields", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.AuditFlags(cli.AuditArgs{Timestamp: "2024-01-01T00:00:00Z"})
		g.Expect(result).To(gomega.Equal([]string{"--timestamp", "2024-01-01T00:00:00Z"}))
	})
}

func TestAutomateFlags(t *testing.T) {
	t.Parallel()

	t.Run("populated fields", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.AutomateFlags(cli.AutomateArgs{DataDir: "/data"})
		g.Expect(result).To(gomega.Equal([]string{"--data-dir", "/data"}))
	})

	t.Run("empty fields skipped", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.AutomateFlags(cli.AutomateArgs{})
		g.Expect(result).To(gomega.BeEmpty())
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
		// 13 individual commands + 1 registry group = 14 entries
		g.Expect(len(targets)).To(gomega.BeNumerically(">=", 14))
	})

	t.Run("each subcommand wires to correct name", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		var calls []string

		targets := cli.BuildTargets(func(subcmd string, _ []string) {
			calls = append(calls, subcmd)
		})

		subcmds := []string{
			"audit", "automate", "correct", "evaluate", "review",
			"maintain", "surface", "learn", "remind", "instruct",
			"context-update", "promote", "demote",
		}
		for _, sub := range subcmds {
			_, _ = targ.Execute([]string{"engram", sub}, targets...)
		}

		g.Expect(calls).To(gomega.Equal(subcmds))
	})

	t.Run("registry subcommands wire correctly", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		var calls []string

		targets := cli.BuildTargets(func(subcmd string, flags []string) {
			if len(flags) > 0 {
				calls = append(calls, subcmd+"/"+flags[0])
			}
		})

		registrySubs := []string{"init", "register-source", "merge"}
		for _, sub := range registrySubs {
			_, _ = targ.Execute([]string{"engram", "registry", sub}, targets...)
		}

		g.Expect(calls).To(gomega.Equal([]string{
			"registry/init",
			"registry/register-source",
			"registry/merge",
		}))
	})
}

func TestContextUpdateFlags(t *testing.T) {
	t.Parallel()

	t.Run("populated fields", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.ContextUpdateFlags(cli.ContextUpdateArgs{
			TranscriptPath: "/transcript",
			SessionID:      "sess-1",
			DataDir:        "/data",
			ContextPath:    "/ctx",
		})
		g.Expect(result).To(gomega.Equal([]string{
			"--transcript-path", "/transcript",
			"--session-id", "sess-1",
			"--data-dir", "/data",
			"--context-path", "/ctx",
		}))
	})

	t.Run("empty fields skipped", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.ContextUpdateFlags(cli.ContextUpdateArgs{})
		g.Expect(result).To(gomega.BeEmpty())
	})

	t.Run("partial fields", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.ContextUpdateFlags(cli.ContextUpdateArgs{
			TranscriptPath: "/transcript",
			DataDir:        "/data",
		})
		g.Expect(result).To(gomega.Equal([]string{
			"--transcript-path", "/transcript",
			"--data-dir", "/data",
		}))
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
		})
		g.Expect(result).To(gomega.Equal([]string{
			"--message", "fix this",
			"--data-dir", "/data",
			"--transcript-path", "/transcript",
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

func TestDemoteFlags(t *testing.T) {
	t.Parallel()

	t.Run("all flags set", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.DemoteFlags(cli.DemoteArgs{DataDir: "/data", ToSkill: true, Yes: true})
		g.Expect(result).To(gomega.Equal([]string{"--data-dir", "/data", "--to-skill", "--yes"}))
	})

	t.Run("bools false", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.DemoteFlags(cli.DemoteArgs{DataDir: "/data", ToSkill: false, Yes: false})
		g.Expect(result).To(gomega.Equal([]string{"--data-dir", "/data"}))
	})

	t.Run("empty fields skipped", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.DemoteFlags(cli.DemoteArgs{})
		g.Expect(result).To(gomega.BeEmpty())
	})

	t.Run("partial bools only to-skill", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.DemoteFlags(cli.DemoteArgs{DataDir: "/data", ToSkill: true, Yes: false})
		g.Expect(result).To(gomega.Equal([]string{"--data-dir", "/data", "--to-skill"}))
	})
}

func TestEvaluateFlags(t *testing.T) {
	t.Parallel()

	t.Run("populated fields", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.EvaluateFlags(cli.EvaluateArgs{DataDir: "/data"})
		g.Expect(result).To(gomega.Equal([]string{"--data-dir", "/data"}))
	})

	t.Run("empty fields skipped", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.EvaluateFlags(cli.EvaluateArgs{})
		g.Expect(result).To(gomega.BeEmpty())
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

func TestLearnFlags(t *testing.T) {
	t.Parallel()

	t.Run("populated fields", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.LearnFlags(cli.LearnArgs{
			DataDir:        "/data",
			TranscriptPath: "/transcript",
			SessionID:      "sess-1",
		})
		g.Expect(result).To(gomega.Equal([]string{
			"--data-dir", "/data",
			"--transcript-path", "/transcript",
			"--session-id", "sess-1",
		}))
	})

	t.Run("empty fields skipped", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.LearnFlags(cli.LearnArgs{})
		g.Expect(result).To(gomega.BeEmpty())
	})

	t.Run("partial fields", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.LearnFlags(cli.LearnArgs{DataDir: "/data", SessionID: "sess-1"})
		g.Expect(result).To(gomega.Equal([]string{
			"--data-dir", "/data",
			"--session-id", "sess-1",
		}))
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

func TestPromoteFlags(t *testing.T) {
	t.Parallel()

	t.Run("all flags set with threshold", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.PromoteFlags(cli.PromoteArgs{
			DataDir:    "/data",
			ToSkill:    true,
			ToClaudeMD: true,
			Threshold:  100,
			Yes:        true,
		})
		g.Expect(result).To(gomega.Equal([]string{
			"--data-dir", "/data",
			"--to-skill", "--to-claude-md", "--yes",
			"--threshold", "100",
		}))
	})

	t.Run("threshold zero omitted", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.PromoteFlags(cli.PromoteArgs{DataDir: "/data"})
		g.Expect(result).To(gomega.Equal([]string{"--data-dir", "/data"}))
	})

	t.Run("threshold positive included", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.PromoteFlags(cli.PromoteArgs{Threshold: 50})
		g.Expect(result).To(gomega.Equal([]string{"--threshold", "50"}))
	})

	t.Run("empty fields skipped", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.PromoteFlags(cli.PromoteArgs{})
		g.Expect(result).To(gomega.BeEmpty())
	})

	t.Run("to-skill true to-claude-md false", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.PromoteFlags(
			cli.PromoteArgs{DataDir: "/data", ToSkill: true, ToClaudeMD: false},
		)
		g.Expect(result).To(gomega.Equal([]string{"--data-dir", "/data", "--to-skill"}))
	})

	t.Run("to-skill false to-claude-md true yes true", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.PromoteFlags(cli.PromoteArgs{DataDir: "/data", ToClaudeMD: true, Yes: true})
		g.Expect(result).
			To(gomega.Equal([]string{"--data-dir", "/data", "--to-claude-md", "--yes"}))
	})

	t.Run("negative threshold omitted", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.PromoteFlags(cli.PromoteArgs{DataDir: "/data", Threshold: -1})
		g.Expect(result).To(gomega.Equal([]string{"--data-dir", "/data"}))
	})
}

func TestRegistryInitFlags(t *testing.T) {
	t.Parallel()

	t.Run("with dry run", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.RegistryInitFlags(cli.RegistryInitArgs{DataDir: "/data", DryRun: true})
		g.Expect(result).To(gomega.Equal([]string{"--data-dir", "/data", "--dry-run"}))
	})

	t.Run("without dry run", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.RegistryInitFlags(cli.RegistryInitArgs{DataDir: "/data"})
		g.Expect(result).To(gomega.Equal([]string{"--data-dir", "/data"}))
	})

	t.Run("empty fields skipped", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.RegistryInitFlags(cli.RegistryInitArgs{})
		g.Expect(result).To(gomega.BeEmpty())
	})

	t.Run("dry-run true no data-dir", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.RegistryInitFlags(cli.RegistryInitArgs{DryRun: true})
		g.Expect(result).To(gomega.Equal([]string{"--dry-run"}))
	})
}

func TestRegistryMergeFlags(t *testing.T) {
	t.Parallel()

	t.Run("populated fields", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.RegistryMergeFlags(cli.RegistryMergeArgs{
			DataDir:  "/data",
			SourceID: "src-1",
			TargetID: "tgt-1",
		})
		g.Expect(result).To(gomega.Equal([]string{
			"--data-dir", "/data",
			"--source", "src-1",
			"--target", "tgt-1",
		}))
	})

	t.Run("empty fields skipped", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.RegistryMergeFlags(cli.RegistryMergeArgs{})
		g.Expect(result).To(gomega.BeEmpty())
	})

	t.Run("partial fields", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.RegistryMergeFlags(cli.RegistryMergeArgs{SourceID: "src-1"})
		g.Expect(result).To(gomega.Equal([]string{"--source", "src-1"}))
	})
}

func TestRegistryRegisterSourceFlags(t *testing.T) {
	t.Parallel()

	t.Run("populated fields", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.RegistryRegisterSourceFlags(cli.RegistryRegisterSourceArgs{
			DataDir:    "/data",
			SourceType: "claude-md",
			Path:       "/path/to/source",
		})
		g.Expect(result).To(gomega.Equal([]string{
			"--data-dir", "/data",
			"--type", "claude-md",
			"--path", "/path/to/source",
		}))
	})

	t.Run("empty fields skipped", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.RegistryRegisterSourceFlags(cli.RegistryRegisterSourceArgs{})
		g.Expect(result).To(gomega.BeEmpty())
	})

	t.Run("partial fields", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.RegistryRegisterSourceFlags(cli.RegistryRegisterSourceArgs{
			SourceType: "memory-md",
			Path:       "/memories.md",
		})
		g.Expect(result).To(gomega.Equal([]string{
			"--type", "memory-md",
			"--path", "/memories.md",
		}))
	})
}

func TestRemindFlags(t *testing.T) {
	t.Parallel()

	t.Run("populated fields", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.RemindFlags(cli.RemindArgs{DataDir: "/data", FilePath: "/file.go"})
		g.Expect(result).
			To(gomega.Equal([]string{"--data-dir", "/data", "--file-path", "/file.go"}))
	})

	t.Run("empty fields skipped", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.RemindFlags(cli.RemindArgs{})
		g.Expect(result).To(gomega.BeEmpty())
	})

	t.Run("partial fields", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.RemindFlags(cli.RemindArgs{FilePath: "/file.go"})
		g.Expect(result).To(gomega.Equal([]string{"--file-path", "/file.go"}))
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

func TestSurfaceFlags(t *testing.T) {
	t.Parallel()

	t.Run("populated fields", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.SurfaceFlags(cli.SurfaceArgs{
			Mode:      "prompt",
			DataDir:   "/data",
			Message:   "hello",
			ToolName:  "Read",
			ToolInput: `{"path":"/foo"}`,
			Format:    "json",
		})
		g.Expect(result).To(gomega.Equal([]string{
			"--mode", "prompt",
			"--data-dir", "/data",
			"--message", "hello",
			"--tool-name", "Read",
			"--tool-input", `{"path":"/foo"}`,
			"--format", "json",
		}))
	})

	t.Run("partial fields", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.SurfaceFlags(cli.SurfaceArgs{Mode: "session-start", DataDir: "/data"})
		g.Expect(result).
			To(gomega.Equal([]string{"--mode", "session-start", "--data-dir", "/data"}))
	})

	t.Run("empty fields skipped", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.SurfaceFlags(cli.SurfaceArgs{})
		g.Expect(result).To(gomega.BeEmpty())
	})

	t.Run("tool mode partial", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.SurfaceFlags(cli.SurfaceArgs{
			Mode:     "tool",
			ToolName: "Read",
			Format:   "json",
		})
		g.Expect(result).To(gomega.Equal([]string{
			"--mode", "tool",
			"--tool-name", "Read",
			"--format", "json",
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
		g.Expect(len(targets)).To(gomega.BeNumerically(">=", 14))
	})

	t.Run("closure wiring invokes RunSafe with injected IO", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		var stderr bytes.Buffer

		// Execute one target to exercise the closure body.
		// I/O goes to injected bytes.Buffer — no real side effects.
		targets := cli.Targets(&bytes.Buffer{}, &stderr, strings.NewReader(""))
		_, _ = targ.Execute([]string{"engram", "review", "--data-dir", t.TempDir()}, targets...)

		// review with empty dir produces no error output.
		g.Expect(stderr.String()).To(gomega.BeEmpty())
	})
}
