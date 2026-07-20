package cli_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/onsi/gomega"
	"github.com/toejough/targ"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
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

		targets := cli.Targets(newTestDeps(&bytes.Buffer{}, &bytes.Buffer{}))
		// learn (group), update, embed (group), query, ingest, query-chunks,
		// activate, count, show, show-chunk, check, resituate, amend, prune,
		// vocab (group)
		g.Expect(targets).To(gomega.HaveLen(15))
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

	t.Run("invokes learn qa closure", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		vault := t.TempDir()
		g.Expect(os.MkdirAll(vault, 0o750)).To(gomega.Succeed())

		stderr := executeForTest(t, []string{
			"engram", "learn", "qa",
			"--slug", "test-qa",
			"--vault", vault,
			"--question", "What is X?",
			"--answer", "X is Y.",
			"--source", "targets test",
		})
		g.Expect(stderr).To(gomega.BeEmpty())

		qPath := filepath.Join(vault, "qa."+timeNowDateForTest()+".test-qa.q.md")
		aPath := filepath.Join(vault, "qa."+timeNowDateForTest()+".test-qa.a.md")

		g.Expect(qPath).To(gomega.BeAnExistingFile())
		g.Expect(aPath).To(gomega.BeAnExistingFile())
	})

	t.Run("learn qa errors when --question is missing", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		vault := t.TempDir()
		g.Expect(os.MkdirAll(vault, 0o750)).To(gomega.Succeed())

		stderr := executeForTest(t, []string{
			"engram", "learn", "qa",
			"--slug", "test-qa",
			"--vault", vault,
			"--answer", "X is Y.",
			"--source", "targets test",
		})
		g.Expect(stderr).To(gomega.ContainSubstring("question"))
	})

	t.Run("learn feedback errors when --source is missing", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		vault := t.TempDir()
		g.Expect(os.MkdirAll(vault, 0o750)).To(gomega.Succeed())

		targets := cli.Targets(newTestDeps(&bytes.Buffer{}, &bytes.Buffer{}))
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

		targets := cli.Targets(newTestDeps(&bytes.Buffer{}, &bytes.Buffer{}))
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
// through Targets() with no --note flags so newActivateDeps wiring is
// covered. The empty-notes fast path returns nil (0 failures, 0 notes).
func TestTargets_ActivateNoNotes(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	stderr := executeForTest(t, []string{"engram", "activate"})
	g.Expect(stderr).To(gomega.BeEmpty())
}

// TestTargets_EmbedApplyDryRun exercises the embed apply closure with
// --dry-run against an empty vault. newTestDeps.Embed is nil by design
// (R11), and RunEmbedApply dereferences Embedder.ModelID(), so the test
// overrides Embed with the fail-loud stubEmbedderForTargets — only
// ModelID() answers; any real Embed call fails the test loudly.
func TestTargets_EmbedApplyDryRun(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(vault, 0o750)).To(gomega.Succeed())
	g.Expect(os.MkdirAll(filepath.Join(vault, "MOCs"), 0o750)).To(gomega.Succeed())

	stderr := executeForTestWithDeps(t,
		[]string{"engram", "embed", "apply", "--dry-run", "--vault", vault},
		func(d *cli.Deps) { d.Embed = stubEmbedderForTargets{} })
	g.Expect(stderr).To(gomega.BeEmpty())
}

// TestTargets_EmbedStatus exercises the embed status closure end-to-end
// through Targets() so the newEmbedDeps(deps) wiring is covered. Empty
// vault; tallyStates dereferences Embedder.ModelID(), so the test wires
// the fail-loud stubEmbedderForTargets (R11) — no model unpack, and any
// real Embed call fails loudly.
func TestTargets_EmbedStatus(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(vault, 0o750)).To(gomega.Succeed())
	g.Expect(os.MkdirAll(filepath.Join(vault, "MOCs"), 0o750)).To(gomega.Succeed())

	stderr := executeForTestWithDeps(t,
		[]string{"engram", "embed", "status", "--vault", vault},
		func(d *cli.Deps) { d.Embed = stubEmbedderForTargets{} })
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
// vault. Unlike query-chunks, RunQuery has no notes/records short-circuit
// ahead of the per-phrase embed call (buildMatchedSetFromPhrases always
// calls deps.Embedder.Embed), so a nil Embed (newTestDeps.Embed stays nil
// globally per R11) would panic here. This test overrides Embed with the
// deterministic stubEmbedder (embed_test.go) via executeForTestWithDeps —
// the same pattern R11 sanctions for the embed-family targets tests'
// ModelID()/Embed() dereferences (#700 T6 finding: the query cluster's
// newQueryDeps(d) conversion surfaces this one task earlier than R11's
// embed-cluster enumeration anticipated).
func TestTargets_QueryEmptyVault(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(vault, 0o750)).To(gomega.Succeed())
	g.Expect(os.MkdirAll(filepath.Join(vault, "MOCs"), 0o750)).To(gomega.Succeed())

	stderr := executeForTestWithDeps(t,
		[]string{"engram", "query", "--phrase", "anything", "--vault", vault},
		func(d *cli.Deps) { d.Embed = stubEmbedder{modelID: "test-model", dims: 8} })
	g.Expect(stderr).To(gomega.BeEmpty())
}

// TestTargets_Resituate exercises the resituate closure end-to-end through
// Targets() so the newResituateDeps wiring is covered. The target is a
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

// stubEmbedderForTargets satisfies embed.Embedder for targets-level tests that
// only need ModelID/Dims. Embed fails loud: no targets-level test may silently
// real-embed (R11). Named to avoid cli_test's existing stubEmbedder (embed_test.go).
type stubEmbedderForTargets struct{}

func (stubEmbedderForTargets) Dims() int { return 384 }

func (stubEmbedderForTargets) Embed(context.Context, string) ([]float32, error) {
	return nil, errors.New("stubEmbedderForTargets: Embed not expected in targets-level tests")
}

func (stubEmbedderForTargets) ModelID() string { return embed.BundledModelID }

// executeForTest runs an engram CLI command through targ, returning stderr content.
// Command-level errors are written to stderr (errHandler contract), not returned as Go errors.
// Targ-level errors (unknown flags, missing required args) are also written to stderr.
func executeForTest(t *testing.T, args []string) string {
	t.Helper()

	return executeForTestWithDeps(t, args, nil)
}

// executeForTestWithDeps runs an engram CLI command through targ like
// executeForTest, but lets the caller customize the test deps (e.g. wire a
// stub embedder into Embed) before Targets runs. customize may be nil.
func executeForTestWithDeps(t *testing.T, args []string, customize func(*cli.Deps)) string {
	t.Helper()

	var stdout, stderr bytes.Buffer

	deps := newTestDeps(&stdout, &stderr)
	if customize != nil {
		customize(&deps)
	}

	_, err := targ.Execute(args, cli.Targets(deps)...)
	if err != nil {
		stderr.WriteString(err.Error())
		stderr.WriteString("\n")
	}

	return stderr.String()
}

// newTestDeps builds a cli.Deps wired to real OS capabilities with captured
// stdout/stderr — the test analog of the production cli.NewDeps composition,
// built over realDepsForTest (T1-rework's primitives_integration_test.go
// helper) with Stdout/Stderr swapped in and Embed forced nil (R11 — unit
// tests must not load the bundled model; the embed-on-write path stays
// covered by cli_test.go's real-binary end-to-end test).
func newTestDeps(stdout, stderr io.Writer) cli.Deps {
	d := realDepsForTest()
	d.Stdout = stdout
	d.Stderr = stderr
	d.Embed = nil

	return d
}

// timeNowDateForTest returns today's date in the vault filename format,
// matching the production Now() wiring in newOsLearnQADeps.
func timeNowDateForTest() string {
	return time.Now().Format("2006-01-02")
}
