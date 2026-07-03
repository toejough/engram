package cli_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/onsi/gomega"
	"github.com/toejough/targ"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/cli"
)

func TestCacheDirFromHome(t *testing.T) {
	t.Parallel()

	const modelID = "minilm-l6-v2@384"

	t.Run("returns XDG cache path when no env override", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		dir := cli.CacheDirFromHome("/Users/joe", modelID, func(string) string { return "" })
		g.Expect(dir).To(gomega.Equal("/Users/joe/.cache/engram/models/minilm-l6-v2@384"))
	})

	t.Run("respects XDG_CACHE_HOME when set", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		dir := cli.CacheDirFromHome("/Users/joe", modelID, func(key string) string {
			if key == "XDG_CACHE_HOME" {
				return "/custom/cache"
			}

			return ""
		})
		g.Expect(dir).To(gomega.Equal("/custom/cache/engram/models/minilm-l6-v2@384"))
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
		// learn (group), update, embed (group), query, ingest, query-chunks,
		// activate, show, show-chunk, check, resituate, amend, prune, vocab (group)
		g.Expect(targets).To(gomega.HaveLen(14))
	})

	t.Run("show parses positional ref through targ", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		vault := t.TempDir()
		perm := vault
		g.Expect(os.MkdirAll(perm, 0o750)).To(gomega.Succeed())
		g.Expect(os.WriteFile(filepath.Join(perm, "1.note.md"),
			[]byte("---\ntype: fact\n---\nbody\n"), 0o600)).To(gomega.Succeed())

		// Exercises targ's struct-tag parsing + positional wiring end-to-end —
		// a comma in the desc would make targ reject the tag ("invalid tag"),
		// which unit tests constructing ShowArgs directly cannot catch.
		stderr := executeForTest(t, []string{"engram", "show", "1.note", "--vault", vault})

		g.Expect(stderr).NotTo(gomega.ContainSubstring("invalid tag"))
		g.Expect(stderr).To(gomega.BeEmpty())
	})

	t.Run("show-chunk parses positional ref through targ", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		chunksDir := t.TempDir()
		records := []chunk.Record{
			{Source: "/s/a.jsonl", Anchor: "turn-1", ContentHash: "sha256:aa", Text: "chunk evidence text"},
		}

		data, err := chunk.EncodeRecords(records)
		g.Expect(err).NotTo(gomega.HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(os.WriteFile(filepath.Join(chunksDir, "idx.jsonl"), data, 0o600)).To(gomega.Succeed())

		// Exercises targ's struct-tag parsing + positional wiring + the os deps
		// constructor end-to-end (a comma in the desc would make targ reject the
		// tag), which constructing ShowChunkArgs directly cannot catch.
		stderr := executeForTest(t, []string{"engram", "show-chunk", "/s/a.jsonl#turn-1", "--chunks-dir", chunksDir})

		g.Expect(stderr).NotTo(gomega.ContainSubstring("invalid tag"))
		g.Expect(stderr).To(gomega.BeEmpty())
	})

	t.Run("invokes learn feedback closure", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		vault := t.TempDir()
		g.Expect(os.MkdirAll(vault, 0o750)).To(gomega.Succeed())

		stderr := executeForTest(t, []string{
			"engram", "learn", "feedback",
			"--slug", "test-slug",
			"--vault", vault,
			"--source", "agent",
			"--situation", "running tests",
		})
		// May or may not error; goal is to invoke the closure.
		_ = stderr
	})

	t.Run("invokes learn fact closure", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		vault := t.TempDir()
		g.Expect(os.MkdirAll(vault, 0o750)).To(gomega.Succeed())

		stderr := executeForTest(t, []string{
			"engram", "learn", "fact",
			"--slug", "test-slug",
			"--vault", vault,
			"--source", "agent",
			"--situation", "running tests",
		})
		_ = stderr
	})

	t.Run("learn feedback errors when --source is missing", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		vault := t.TempDir()
		g.Expect(os.MkdirAll(vault, 0o750)).To(gomega.Succeed())

		targets := cli.Targets(&bytes.Buffer{}, &bytes.Buffer{}, func(int) {}, nil)
		result, err := targ.Execute([]string{
			"engram", "learn", "feedback",
			"--slug", "test-slug",
			"--vault", vault,
			"--situation", "x", "--behavior", "x", "--impact", "x", "--action", "x",
		}, targets...)
		g.Expect(err).To(gomega.HaveOccurred())
		g.Expect(result.Output).To(gomega.ContainSubstring("source"))
	})

	t.Run("learn fact errors when --source is missing", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		vault := t.TempDir()
		g.Expect(os.MkdirAll(vault, 0o750)).To(gomega.Succeed())

		targets := cli.Targets(&bytes.Buffer{}, &bytes.Buffer{}, func(int) {}, nil)
		result, err := targ.Execute([]string{
			"engram", "learn", "fact",
			"--slug", "test-slug",
			"--vault", vault,
			"--situation", "x", "--subject", "x", "--predicate", "x", "--object", "x",
		}, targets...)
		g.Expect(err).To(gomega.HaveOccurred())
		g.Expect(result.Output).To(gomega.ContainSubstring("source"))
	})

	t.Run("invokes update closure in dry-run mode", func(t *testing.T) {
		t.Parallel()

		_ = executeForTest(t, []string{
			"engram", "update", "--dry-run",
		})
	})
}

// TestTargets_ActivateNoNotes exercises the activate closure end-to-end
// through Targets() with no --note flags so newOsActivateDeps wiring is
// covered. The empty-notes fast path returns nil (0 failures, 0 notes).
func TestTargets_ActivateNoNotes(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	stderr := executeForTest(t, []string{"engram", "activate"})
	g.Expect(stderr).To(gomega.BeEmpty())
}

// TestTargets_EmbedApplyDryRun exercises embed apply closure with --dry-run
// against an empty vault. Lazy embedder's ModelID() doesn't unpack model.
func TestTargets_EmbedApplyDryRun(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(vault, 0o750)).To(gomega.Succeed())
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
	g.Expect(os.MkdirAll(vault, 0o750)).To(gomega.Succeed())
	g.Expect(os.MkdirAll(filepath.Join(vault, "MOCs"), 0o750)).To(gomega.Succeed())

	stderr := executeForTest(t, []string{"engram", "embed", "status", "--vault", vault})
	g.Expect(stderr).To(gomega.BeEmpty())
}

// TestTargets_IngestAndQueryChunksEmpty exercises the ingest and query-chunks
// target closures (ResolveChunksDir wiring) on empty inputs — both fast paths
// that never wake the bundled embedder.
func TestTargets_IngestAndQueryChunksEmpty(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	chunks := t.TempDir()

	stderr := executeForTest(t, []string{"engram", "ingest", "--chunks-dir", chunks})
	g.Expect(stderr).To(gomega.BeEmpty())

	stderr = executeForTest(t, []string{
		"engram", "query-chunks", "--chunks-dir", chunks, "--phrase", "anything",
	})
	g.Expect(stderr).To(gomega.BeEmpty())
}

// TestTargets_PruneEmpty exercises the prune target closure end-to-end on an
// empty chunks dir — the "no manifest" fast path, which verifies the wiring
// without creating real files.
func TestTargets_PruneEmpty(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	chunks := t.TempDir()

	stderr := executeForTest(t, []string{"engram", "prune", "--chunks-dir", chunks})
	g.Expect(stderr).To(gomega.BeEmpty())
}

// TestTargets_QueryEmptyVault exercises the query closure on an empty
// vault — fast path returns items:[] without invoking the embedder.
func TestTargets_QueryEmptyVault(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(vault, 0o750)).To(gomega.Succeed())
	g.Expect(os.MkdirAll(filepath.Join(vault, "MOCs"), 0o750)).To(gomega.Succeed())

	stderr := executeForTest(t, []string{"engram", "query", "--phrase", "anything", "--vault", vault})
	g.Expect(stderr).To(gomega.BeEmpty())
}

// TestTargets_Resituate exercises the resituate closure end-to-end through
// Targets() so the newOsResituateDeps wiring is covered. The target is a
// non-existent note in an empty vault: the real ScanVault runs and finds
// nothing, so the not-found sentinel surfaces on stderr without unpacking
// the bundled embedder (no matching note to re-embed).
func TestTargets_Resituate(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(vault, 0o750)).To(gomega.Succeed())
	g.Expect(os.MkdirAll(filepath.Join(vault, "MOCs"), 0o750)).To(gomega.Succeed())

	stderr := executeForTest(t, []string{
		"engram", "resituate", "--vault", vault, "--note", "9zz", "--situation", "new topic",
	})
	g.Expect(stderr).To(gomega.ContainSubstring("not found"))
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
