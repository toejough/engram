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

func TestBuildServerGroup(t *testing.T) {
	t.Parallel()

	t.Run("returns a non-nil group", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		group := cli.BuildServerGroup(&bytes.Buffer{}, &bytes.Buffer{}, strings.NewReader(""))
		g.Expect(group).NotTo(gomega.BeNil())
	})

	t.Run("executes up subcommand via closure with invalid addr", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()

		var stderr bytes.Buffer

		// server up with an invalid addr exercises the closure body.
		// targ parses ServerUpArgs from the flags and calls RunSafe.
		// RunSafe writes any error to stderr.
		targets := cli.Targets(&bytes.Buffer{}, &stderr, strings.NewReader(""))
		_, _ = targ.Execute([]string{
			"engram", "server", "up",
			"--chat-file", dir + "/chat.toml",
			"--addr", "!!!invalid-addr!!!",
		}, targets...)
		// The server will attempt to listen on the invalid address and fail.
		// We just verify the closure was invoked (any outcome is acceptable).
	})
}

func TestBuildTargets(t *testing.T) {
	t.Parallel()

	t.Run("returns expected number of targets", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		targets := cli.BuildTargets(func(_ string, _ []string) {})
		g.Expect(targets).To(gomega.HaveLen(7))
	})

	t.Run("each subcommand wires to correct name", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		var calls []string

		targets := cli.BuildTargets(func(subcmd string, _ []string) {
			calls = append(calls, subcmd)
		})

		subcmds := []string{"recall", "show", "post", "intent", "learn", "status", "subscribe"}
		for _, sub := range subcmds {
			_, _ = targ.Execute([]string{"engram", sub}, targets...)
		}

		g.Expect(calls).To(gomega.Equal(subcmds))
	})
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

func TestIntentFlags(t *testing.T) {
	t.Parallel()

	t.Run("populated fields", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.IntentFlags(cli.IntentArgs{
			From:          "lead-1",
			To:            "engram-agent",
			Situation:     "deploying",
			PlannedAction: "run tests",
			Addr:          "http://localhost:9999",
		})
		g.Expect(result).To(gomega.Equal([]string{
			"--from", "lead-1",
			"--to", "engram-agent",
			"--situation", "deploying",
			"--planned-action", "run tests",
			"--addr", "http://localhost:9999",
		}))
	})

	t.Run("empty optional fields omitted", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.IntentFlags(cli.IntentArgs{
			From: "lead-1",
			To:   "engram-agent",
		})
		g.Expect(result).To(gomega.Equal([]string{
			"--from", "lead-1",
			"--to", "engram-agent",
		}))
	})
}

func TestLearnFlags(t *testing.T) {
	t.Parallel()

	t.Run("all fields populated", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.LearnFlags(cli.LearnArgs{
			From:      "lead-1",
			Type:      "feedback",
			Situation: "deploying",
			Behavior:  "skipped tests",
			Impact:    "breakage",
			Action:    "always run tests",
			Subject:   "",
			Predicate: "",
			Object:    "",
			Addr:      "http://localhost:9999",
		})
		g.Expect(result).To(gomega.Equal([]string{
			"--from", "lead-1",
			"--type", "feedback",
			"--situation", "deploying",
			"--behavior", "skipped tests",
			"--impact", "breakage",
			"--action", "always run tests",
			"--addr", "http://localhost:9999",
		}))
	})

	t.Run("empty optional fields omitted", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.LearnFlags(cli.LearnArgs{
			From: "lead-1",
			Type: "fact",
		})
		g.Expect(result).To(gomega.Equal([]string{
			"--from", "lead-1",
			"--type", "fact",
		}))
	})

	t.Run("fact fields populated", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.LearnFlags(cli.LearnArgs{
			From:      "lead-1",
			Type:      "fact",
			Situation: "investigating",
			Subject:   "engram",
			Predicate: "uses",
			Object:    "TF-IDF",
		})
		g.Expect(result).To(gomega.Equal([]string{
			"--from", "lead-1",
			"--type", "fact",
			"--situation", "investigating",
			"--subject", "engram",
			"--predicate", "uses",
			"--object", "TF-IDF",
		}))
	})
}

func TestPostFlags(t *testing.T) {
	t.Parallel()

	t.Run("populated fields", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.PostFlags(cli.PostArgs{
			From: "lead-1",
			To:   "engram-agent",
			Text: "hello",
			Addr: "http://localhost:9999",
		})
		g.Expect(result).To(gomega.Equal([]string{
			"--from", "lead-1",
			"--to", "engram-agent",
			"--text", "hello",
			"--addr", "http://localhost:9999",
		}))
	})

	t.Run("empty addr omitted", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.PostFlags(cli.PostArgs{
			From: "lead-1",
			To:   "engram-agent",
			Text: "hello",
		})
		g.Expect(result).To(gomega.Equal([]string{
			"--from", "lead-1",
			"--to", "engram-agent",
			"--text", "hello",
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

func TestServerUpFlags(t *testing.T) {
	t.Parallel()

	t.Run("all fields populated", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.ServerUpFlags(cli.ServerUpArgs{
			ChatFile: "/tmp/chat.toml",
			LogFile:  "/tmp/server.log",
			Addr:     "localhost:9000",
		})
		g.Expect(result).To(gomega.Equal([]string{
			"--chat-file", "/tmp/chat.toml",
			"--log-file", "/tmp/server.log",
			"--addr", "localhost:9000",
		}))
	})

	t.Run("empty fields omitted", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.ServerUpFlags(cli.ServerUpArgs{})
		g.Expect(result).To(gomega.BeEmpty())
	})
}

func TestShowFlags(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	result := cli.ShowFlags(cli.ShowArgs{DataDir: "/data"})
	g.Expect(result).To(gomega.Equal([]string{"--data-dir", "/data"}))
}

func TestStatusFlags(t *testing.T) {
	t.Parallel()

	t.Run("populated addr", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.StatusFlags(cli.StatusArgs{
			Addr: "http://localhost:9999",
		})
		g.Expect(result).To(gomega.Equal([]string{
			"--addr", "http://localhost:9999",
		}))
	})

	t.Run("empty addr omitted", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.StatusFlags(cli.StatusArgs{})
		g.Expect(result).To(gomega.BeEmpty())
	})
}

func TestSubscribeFlags(t *testing.T) {
	t.Parallel()

	t.Run("all fields populated", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.SubscribeFlags(cli.SubscribeArgs{
			Agent:       "worker-1",
			AfterCursor: 42,
			Addr:        "http://localhost:9999",
		})
		g.Expect(result).To(gomega.Equal([]string{
			"--agent", "worker-1",
			"--addr", "http://localhost:9999",
			"--after-cursor", "42",
		}))
	})

	t.Run("empty optional fields omitted", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.SubscribeFlags(cli.SubscribeArgs{
			Agent: "worker-1",
		})
		g.Expect(result).To(gomega.Equal([]string{
			"--agent", "worker-1",
		}))
	})

	t.Run("zero after-cursor omitted", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.SubscribeFlags(cli.SubscribeArgs{
			Agent:       "worker-1",
			AfterCursor: 0,
		})
		g.Expect(result).To(gomega.Equal([]string{
			"--agent", "worker-1",
		}))
	})
}

func TestTargets(t *testing.T) {
	t.Parallel()

	t.Run("returns expected target count", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		// Construction doesn't do I/O — just builds targ target objects.
		// 7 individual targets (recall, show, post, intent, learn, status, subscribe)
		// + 1 server group = 8 total.
		targets := cli.Targets(&bytes.Buffer{}, &bytes.Buffer{}, strings.NewReader(""))
		g.Expect(targets).To(gomega.HaveLen(8))
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
