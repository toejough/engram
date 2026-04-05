package cli_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
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

func TestBuildChatGroup(t *testing.T) {
	t.Parallel()

	t.Run("returns a non-nil group", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		group := cli.BuildChatGroup(&bytes.Buffer{}, &bytes.Buffer{}, strings.NewReader(""))
		g.Expect(group).NotTo(gomega.BeNil())
	})

	t.Run("executes post subcommand via closure", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		dir := t.TempDir()
		chatFile := filepath.Join(dir, "chat.toml")

		var stdout bytes.Buffer

		targets := cli.Targets(&stdout, &bytes.Buffer{}, strings.NewReader(""))
		_, _ = targ.Execute([]string{
			"engram", "chat", "post",
			"--chat-file", chatFile,
			"--from", "x", "--to", "all", "--thread", "t", "--type", "info", "--text", "hi",
		}, targets...)

		// post outputs the new cursor (a non-negative integer)
		g.Expect(strings.TrimSpace(stdout.String())).To(gomega.MatchRegexp(`^\d+$`))
	})

	t.Run("executes cursor subcommand via closure", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		dir := t.TempDir()
		chatFile := filepath.Join(dir, "chat.toml")
		g.Expect(os.WriteFile(chatFile, []byte("a\nb\nc\n"), 0o600)).To(gomega.Succeed())

		var stdout bytes.Buffer

		targets := cli.Targets(&stdout, &bytes.Buffer{}, strings.NewReader(""))
		_, _ = targ.Execute([]string{
			"engram", "chat", "cursor",
			"--chat-file", chatFile,
		}, targets...)

		g.Expect(strings.TrimSpace(stdout.String())).To(gomega.Equal("3"))
	})

	t.Run("executes watch subcommand via closure", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		dir := t.TempDir()
		chatFile := filepath.Join(dir, "chat.toml")

		// Pre-write a matching message so watch returns immediately (cursor=0 finds it).
		postErr := cli.Run([]string{
			"engram", "chat", "post",
			"--chat-file", chatFile,
			"--from", "sender",
			"--to", "watcher",
			"--thread", "test",
			"--type", "info",
			"--text", "hello",
		}, io.Discard, io.Discard, nil)
		g.Expect(postErr).NotTo(gomega.HaveOccurred())

		if postErr != nil {
			return
		}

		var stdout bytes.Buffer

		targets := cli.Targets(&stdout, &bytes.Buffer{}, strings.NewReader(""))
		_, _ = targ.Execute([]string{
			"engram", "chat", "watch",
			"--chat-file", chatFile,
			"--agent", "watcher",
			"--cursor", "0",
			"--type", "info",
		}, targets...)

		g.Expect(stdout.String()).To(gomega.ContainSubstring(`"from":"sender"`))
	})
}

func TestBuildChatGroup_AckWait(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")

	// Pre-post an ACK so the ack-wait call returns immediately.
	postErr := cli.Run([]string{
		"engram", "chat", "post",
		"--chat-file", chatFile,
		"--from", "engram-agent", "--to", "tester", "--thread", "t",
		"--type", "ack", "--text", "ok",
	}, io.Discard, io.Discard, nil)
	g.Expect(postErr).NotTo(gomega.HaveOccurred())

	if postErr != nil {
		return
	}

	var stdout bytes.Buffer

	targets := cli.Targets(&stdout, &bytes.Buffer{}, strings.NewReader(""))
	_, _ = targ.Execute([]string{
		"engram", "chat", "ack-wait",
		"--chat-file", chatFile,
		"--agent", "tester",
		"--cursor", "0",
		"--recipients", "engram-agent",
		"--max-wait", "5",
	}, targets...)

	g.Expect(stdout.String()).To(gomega.ContainSubstring(`"result"`))
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

func TestBuildHoldGroup(t *testing.T) {
	t.Parallel()

	t.Run("returns a non-nil group", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		group := cli.BuildHoldGroup(&bytes.Buffer{}, &bytes.Buffer{}, strings.NewReader(""))
		g.Expect(group).NotTo(gomega.BeNil())
	})

	t.Run("executes acquire subcommand via closure", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		dir := t.TempDir()
		chatFile := filepath.Join(dir, "chat.toml")

		var stdout bytes.Buffer

		targets := cli.Targets(&stdout, &bytes.Buffer{}, strings.NewReader(""))
		_, _ = targ.Execute([]string{
			"engram", "hold", "acquire",
			"--chat-file", chatFile,
			"--holder", "lead", "--target", "exec-1",
		}, targets...)

		// acquire outputs the hold-id (UUID format)
		g.Expect(strings.TrimSpace(stdout.String())).NotTo(gomega.BeEmpty())
	})

	t.Run("executes release subcommand via closure", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		dir := t.TempDir()
		chatFile := filepath.Join(dir, "chat.toml")

		// Acquire a hold first so there is something to release.
		var acquireOut bytes.Buffer

		acquireErr := cli.Run([]string{
			"engram", "hold", "acquire",
			"--chat-file", chatFile,
			"--holder", "lead", "--target", "exec-1",
		}, &acquireOut, io.Discard, nil)
		g.Expect(acquireErr).NotTo(gomega.HaveOccurred())

		if acquireErr != nil {
			return
		}

		holdID := strings.TrimSpace(acquireOut.String())

		var stdout bytes.Buffer

		targets := cli.Targets(&stdout, &bytes.Buffer{}, strings.NewReader(""))
		_, _ = targ.Execute([]string{
			"engram", "hold", "release",
			"--chat-file", chatFile,
			"--hold-id", holdID,
		}, targets...)

		// release outputs "OK"
		g.Expect(strings.TrimSpace(stdout.String())).To(gomega.Equal("OK"))
	})

	t.Run("executes list subcommand via closure", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		dir := t.TempDir()
		chatFile := filepath.Join(dir, "chat.toml")

		acquireErr := cli.Run([]string{
			"engram", "hold", "acquire",
			"--chat-file", chatFile,
			"--holder", "lead", "--target", "exec-1",
		}, io.Discard, io.Discard, nil)
		g.Expect(acquireErr).NotTo(gomega.HaveOccurred())

		if acquireErr != nil {
			return
		}

		var stdout bytes.Buffer

		targets := cli.Targets(&stdout, &bytes.Buffer{}, strings.NewReader(""))
		_, _ = targ.Execute([]string{
			"engram", "hold", "list",
			"--chat-file", chatFile,
		}, targets...)

		g.Expect(stdout.String()).NotTo(gomega.BeEmpty())
	})

	t.Run("executes check subcommand via closure", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		dir := t.TempDir()
		chatFile := filepath.Join(dir, "chat.toml")

		// check on empty/nonexistent file produces no output (no active holds to release)
		var stdout bytes.Buffer

		targets := cli.Targets(&stdout, &bytes.Buffer{}, strings.NewReader(""))
		_, _ = targ.Execute([]string{
			"engram", "hold", "check",
			"--chat-file", chatFile,
		}, targets...)

		g.Expect(stdout.String()).To(gomega.BeEmpty())
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

func TestChatAckWaitFlags(t *testing.T) {
	t.Parallel()

	t.Run("includes all non-zero fields", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.ChatAckWaitFlags(cli.ChatAckWaitArgs{
			Agent:      "tester",
			Cursor:     10,
			Recipients: "a,b",
			MaxWait:    30,
			ChatFile:   "/tmp/chat.toml",
		})
		g.Expect(result).To(gomega.ContainElements(
			"--agent", "tester",
			"--recipients", "a,b",
			"--chat-file", "/tmp/chat.toml",
			"--cursor", "10",
			"--max-wait", "30",
		))
	})

	t.Run("omits cursor and max-wait when zero", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.ChatAckWaitFlags(cli.ChatAckWaitArgs{Agent: "tester"})
		g.Expect(result).To(gomega.Equal([]string{"--agent", "tester"}))
		g.Expect(result).NotTo(gomega.ContainElement("--cursor"))
		g.Expect(result).NotTo(gomega.ContainElement("--max-wait"))
	})
}

func TestChatCursorFlags(t *testing.T) {
	t.Parallel()

	t.Run("includes chat-file when set", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.ChatCursorFlags(cli.ChatCursorArgs{ChatFile: "/tmp/chat.toml"})
		g.Expect(result).To(gomega.Equal([]string{"--chat-file", "/tmp/chat.toml"}))
	})

	t.Run("returns empty when chat-file empty", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.ChatCursorFlags(cli.ChatCursorArgs{})
		g.Expect(result).To(gomega.BeEmpty())
	})
}

func TestChatPostFlags(t *testing.T) {
	t.Parallel()

	t.Run("populated fields", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.ChatPostFlags(cli.ChatPostArgs{
			From:     "alice",
			To:       "bob",
			Thread:   "main",
			MsgType:  "info",
			Text:     "hello",
			ChatFile: "/tmp/chat.toml",
		})
		g.Expect(result).To(gomega.Equal([]string{
			"--from", "alice",
			"--to", "bob",
			"--thread", "main",
			"--type", "info",
			"--text", "hello",
			"--chat-file", "/tmp/chat.toml",
		}))
	})

	t.Run("empty fields omitted", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.ChatPostFlags(cli.ChatPostArgs{
			From: "alice", To: "all", Thread: "t", MsgType: "info", Text: "hi",
		})
		g.Expect(result).To(gomega.Equal([]string{
			"--from", "alice", "--to", "all", "--thread", "t", "--type", "info", "--text", "hi",
		}))
	})
}

func TestChatWatchFlags(t *testing.T) {
	t.Parallel()

	t.Run("includes cursor and max-wait when non-zero", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.ChatWatchFlags(cli.ChatWatchArgs{
			Agent:    "bob",
			Cursor:   42,
			Types:    "info",
			MaxWait:  10,
			ChatFile: "/tmp/chat.toml",
		})
		g.Expect(result).To(gomega.ContainElements(
			"--agent", "bob", "--type", "info",
			"--chat-file", "/tmp/chat.toml", "--cursor", "42", "--max-wait", "10",
		))
	})

	t.Run("omits cursor and max-wait when zero", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.ChatWatchFlags(cli.ChatWatchArgs{Agent: "bob"})
		g.Expect(result).To(gomega.Equal([]string{"--agent", "bob"}))
		g.Expect(result).NotTo(gomega.ContainElement("--cursor"))
		g.Expect(result).NotTo(gomega.ContainElement("--max-wait"))
	})
}

func TestDataDirFromHome(t *testing.T) {
	// Not parallel: subtests use t.Setenv which modifies process environment.
	t.Run("returns XDG data path when no env override", func(t *testing.T) {
		g := gomega.NewWithT(t)

		t.Setenv("XDG_DATA_HOME", "")

		dir := cli.DataDirFromHome("/Users/joe")
		g.Expect(dir).To(gomega.Equal("/Users/joe/.local/share/engram"))
	})

	t.Run("respects XDG_DATA_HOME when set", func(t *testing.T) {
		g := gomega.NewWithT(t)

		t.Setenv("XDG_DATA_HOME", "/custom/data")

		dir := cli.DataDirFromHome("/Users/joe")
		g.Expect(dir).To(gomega.Equal("/custom/data/engram"))
	})
}

func TestHoldAcquireFlags(t *testing.T) {
	t.Parallel()

	t.Run("includes all non-empty fields", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.HoldAcquireFlags(cli.HoldAcquireArgs{
			Holder:    "lead",
			Target:    "exec-1",
			Condition: "done:lead",
			Tag:       "codesign-1",
			ChatFile:  "/tmp/chat.toml",
		})
		g.Expect(result).To(gomega.Equal([]string{
			"--holder", "lead",
			"--target", "exec-1",
			"--condition", "done:lead",
			"--tag", "codesign-1",
			"--chat-file", "/tmp/chat.toml",
		}))
	})

	t.Run("omits empty fields", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.HoldAcquireFlags(cli.HoldAcquireArgs{Holder: "lead", Target: "exec-1"})
		g.Expect(result).To(gomega.Equal([]string{"--holder", "lead", "--target", "exec-1"}))
	})
}

func TestHoldCheckFlags(t *testing.T) {
	t.Parallel()

	t.Run("includes chat-file when set", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.HoldCheckFlags(cli.HoldCheckArgs{ChatFile: "/tmp/chat.toml"})
		g.Expect(result).To(gomega.Equal([]string{"--chat-file", "/tmp/chat.toml"}))
	})

	t.Run("returns empty when chat-file empty", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.HoldCheckFlags(cli.HoldCheckArgs{})
		g.Expect(result).To(gomega.BeEmpty())
	})
}

func TestHoldListFlags(t *testing.T) {
	t.Parallel()

	t.Run("includes all non-empty fields", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.HoldListFlags(cli.HoldListArgs{
			Holder:   "lead",
			Target:   "exec-1",
			Tag:      "codesign-1",
			ChatFile: "/tmp/chat.toml",
		})
		g.Expect(result).To(gomega.Equal([]string{
			"--holder", "lead",
			"--target", "exec-1",
			"--tag", "codesign-1",
			"--chat-file", "/tmp/chat.toml",
		}))
	})

	t.Run("omits empty fields", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.HoldListFlags(cli.HoldListArgs{Tag: "plan-review-1"})
		g.Expect(result).To(gomega.Equal([]string{"--tag", "plan-review-1"}))
	})
}

func TestHoldReleaseFlags(t *testing.T) {
	t.Parallel()

	t.Run("includes hold-id and chat-file", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.HoldReleaseFlags(cli.HoldReleaseArgs{
			HoldID:   "abc-123",
			ChatFile: "/tmp/chat.toml",
		})
		g.Expect(result).To(gomega.Equal([]string{
			"--hold-id", "abc-123",
			"--chat-file", "/tmp/chat.toml",
		}))
	})

	t.Run("omits empty fields", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		result := cli.HoldReleaseFlags(cli.HoldReleaseArgs{HoldID: "xyz-456"})
		g.Expect(result).To(gomega.Equal([]string{"--hold-id", "xyz-456"}))
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
		g.Expect(targets).To(gomega.HaveLen(4))
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
