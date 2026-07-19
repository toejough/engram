### Task T2: Targets(deps), single-statement cmd main over Primitives, delete SetupSignalHandling/Main, purify debuglog

**Files:**
- Modify: `internal/debuglog/debuglog.go` (pure New/Log/Timed), `internal/debuglog/debuglog_test.go`
- Modify: `internal/cli/targets.go` (Targets(deps Deps) + all helper funcs; 22 `os.Getenv` sites → `deps.Getenv`; `os` import dropped)
- Modify: `internal/cli/primitives.go` (NewDeps gains the internal Embed wiring line — R6/doctrine flag D-1)
- Modify: `internal/cli/signal.go` (delete `SetupSignalHandling` + its io/os/os-signal/syscall/debuglog imports; KEEP `signalChannelBuffer` — consumed by `startForceExit` — plus `ForwardAsPulses`/`startForceExit`)
- Modify: `internal/cli/targets_test.go`, `internal/cli/vocab_commands_test.go` (8 `cli.Targets` call sites → `newTestDeps` helper)
- Modify: `cmd/engram/main.go` (full rewrite: declaration-free package, single-statement main() over the `cli.Primitives` literal + relocated FIXME marker)
- Delete: `internal/cli/main.go` (old `Main`) — the FIXME(#700) marker at main.go:19–22 is NOT resolved here: T2 RELOCATES it into `cmd/engram/main.go` (see step 5); only T-final-2 deletes it, after enforcement is green

**Interfaces:**
- Consumes: `cli.Primitives`/`cli.NewDeps`/`cli.ForwardAsPulses`/`cli.WriteSyncer` (T1-rework); `debuglog.WithLogger(ctx, *Logger) context.Context`; `targ.Main(...any)`; `embed.NewLazyEmbedder(cacheDir string) *LazyEmbedder` (internal/embed/hugot.go:149); `embed.BundledModelID = "minilm-l6-v2@384"` (hugot.go:18); `cli.CacheDirFromHome(home, modelID string, getenv func(string) string) string` (targets.go:56).
- Produces: `func Targets(deps Deps) []any` (replaces `Targets(stdout, stderr io.Writer, exit func(int), logger *debuglog.Logger) []any`); `func New(w io.Writer, prefix string, now func() time.Time) *Logger` (replaces `New(path, comp string) (*Logger, error)`); `Deps.Embed` wired inside NewDeps.

**Steps:**

1. [ ] RED — rewrite `internal/debuglog/debuglog_test.go` against the pure API (deterministic clock, no filesystem). Compile failure against the old `New(path, comp) (*Logger, error)` is the RED. Full replacement:

```go
package debuglog_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	g "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/debuglog"
)

func TestLog_NilReceiverIsSafe(t *testing.T) {
	t.Parallel()

	var nilLogger *debuglog.Logger

	// Nil-receiver methods must not panic.
	nilLogger.Log("stage", "msg=%s", "value")
	closer := nilLogger.Timed("stage", "arg=%s", "v")
	closer()
}

func TestLog_NoopWhenDisabled(t *testing.T) {
	t.Parallel()

	gomega := g.NewWithT(t)

	// A nil writer means "logging disabled": New returns a nil *Logger whose
	// methods are all no-ops.
	logger := debuglog.New(nil, "", fixedNow)
	gomega.Expect(logger).To(g.BeNil())

	// Must not panic on a disabled (no-op) logger.
	logger.Log("stage", "msg=%s", "value")

	// Must also no-op when nothing is in ctx.
	debuglog.Log(context.Background(), "stage", "msg=%s", "value")
}

func TestLog_WritesTimestampedLine(t *testing.T) {
	t.Parallel()

	gomega := g.NewWithT(t)

	var out bytes.Buffer

	logger := debuglog.New(&out, "test", fixedNow)

	ctx := debuglog.WithLogger(context.Background(), logger)
	debuglog.Log(ctx, "some.stage", "key=%s val=%d", "hello", 42)

	gomega.Expect(out.String()).To(g.Equal(
		"2026-07-19T12:00:00Z [test] some.stage: key=hello val=42\n"))
}

func TestTimed_LogsStartAndEndWithDuration(t *testing.T) {
	t.Parallel()

	gomega := g.NewWithT(t)

	var out bytes.Buffer

	logger := debuglog.New(&out, "timed", steppingNow())

	ctx := debuglog.WithLogger(context.Background(), logger)
	closer := debuglog.Timed(ctx, "MyStage", "arg=%s", "val")
	closer()

	text := out.String()
	gomega.Expect(text).To(g.ContainSubstring("[timed] MyStage.start: arg=val"))
	gomega.Expect(text).To(g.ContainSubstring("[timed] MyStage.end: took=1s"))

	lines := strings.Split(strings.TrimSpace(text), "\n")
	gomega.Expect(lines).To(g.HaveLen(2))
}

func TestTimed_NoLoggerInContext(t *testing.T) {
	t.Parallel()

	// Package-level Timed with no logger in ctx returns a no-op closer.
	closer := debuglog.Timed(context.Background(), "stage", "arg=%s", "v")
	closer()
}

// fixedNow returns a constant instant so timestamp output is exact.
func fixedNow() time.Time {
	return time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC)
}

// steppingNow returns a clock that advances one second per call, making
// Timed's took= duration deterministic. Call sequence inside Timed:
// start-line timestamp, start capture, took argument, end-line timestamp —
// so took = 1s exactly.
func steppingNow() func() time.Time {
	current := fixedNow()

	return func() time.Time {
		now := current
		current = current.Add(time.Second)

		return now
	}
}
```

   Run `targ test` — expect compile failure in internal/debuglog tests (RED).

2. [ ] GREEN — rewrite `internal/debuglog/debuglog.go`. Current impure pieces: `os` import (line 13), `file *os.File` field (23), `os.OpenFile` (37), `time.Now()` (62, 79), `time.Since` (82), `filePerm` const (100–103). Full replacement (context.go is untouched; package-level `Log`/`Timed` kept verbatim):

```go
// Package debuglog provides a tail-friendly debug logger for engram
// pipelines. New wraps an injected io.Writer sink and returns a *Logger.
// Log calls write one line at a time under a mutex; the production sink
// (internal/cli's composed debug sink over the cmd-injected open primitive)
// syncs to disk after every write so `tail -F` shows progress live. The
// package itself performs no I/O and reads no clock — the sink and the now
// func are injected at the edge (#700).
//
// Loggers are threaded through context (see WithLogger / LoggerFromContext).
// The package-level Log and Timed helpers read the logger from ctx, so call
// sites stay short while production wiring stays explicit.
package debuglog

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"
)

// Logger writes structured debug lines to an injected sink. Methods are
// safe for concurrent use within one process and safe to call on a nil
// receiver (no-op), which means tests can pass a nil logger without panics.
type Logger struct {
	component string
	out       io.Writer
	now       func() time.Time
	mu        sync.Mutex
}

// New returns a *Logger tagged with prefix that writes to w, stamping each
// line via now. A nil w returns a nil *Logger — every method is a
// nil-receiver-safe no-op, preserving the "unset ENGRAM_DEBUG_LOG disables
// logging" behavior. now must be non-nil when w is non-nil.
func New(w io.Writer, prefix string, now func() time.Time) *Logger {
	if w == nil {
		return nil
	}

	return &Logger{component: prefix, out: w, now: now}
}

// Log writes a single line: <timestamp> [<component>] <stage>: <message>.
// Safe on a nil receiver (no-op) and safe for concurrent use.
//
//nolint:goprintffuncname // "Log" reads more naturally than "Logf" at call sites
func (l *Logger) Log(stage, format string, args ...any) {
	if l == nil {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := l.now().UTC().Format(time.RFC3339Nano)
	msg := fmt.Sprintf(format, args...)
	line := fmt.Sprintf("%s [%s] %s: %s\n", timestamp, l.component, stage, msg)

	_, _ = io.WriteString(l.out, line)
}

// Timed wraps a stage with .start and .end log entries plus duration.
// Returns a defer-friendly closer:
//
//	defer logger.Timed("Cycle.Run", "projectDir=%s", projectDir)()
//
// Safe on a nil receiver.
func (l *Logger) Timed(stage, format string, args ...any) func() {
	if l == nil {
		return func() {}
	}

	l.Log(stage+".start", format, args...)

	start := l.now()

	return func() {
		l.Log(stage+".end", "took=%s", l.now().Sub(start))
	}
}

// Log reads a *Logger from ctx and writes a line. No-op when ctx carries
// no logger.
//
//nolint:goprintffuncname // mirrors Logger.Log naming
func Log(ctx context.Context, stage, format string, args ...any) {
	LoggerFromContext(ctx).Log(stage, format, args...)
}

// Timed reads a *Logger from ctx and starts a timed entry. No-op closer
// when ctx carries no logger.
func Timed(ctx context.Context, stage, format string, args ...any) func() {
	return LoggerFromContext(ctx).Timed(stage, format, args...)
}
```

   `targ test` still RED overall: internal/cli/main.go:23 now fails to compile (`debuglog.New(os.Getenv(...), "engram")` — wrong arity). That is expected; proceed.

3. [ ] Rewrite `internal/cli/targets.go`. Lines 1–79 (imports through `ProjectSlugFromPath`) change only in the import block — drop `"io"`? No: `newErrHandler` keeps `io.Writer`; drop only `"os"`. New import block replaces lines 3–14:

```go
import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/toejough/targ"

	"github.com/toejough/engram/internal/debuglog"
)
```

   Lines 16–79 (`CommonLearnArgs`, `LearnFactArgs`, `LearnFeedbackArgs`, `CacheDirFromHome`, `DataDirFromHome`, `ProjectSlugFromPath`) are unchanged. Every function from line 81 down except `newErrHandler` (252–264, unchanged) is replaced:

```go
// Targets returns all targ targets for the engram CLI, wired from the
// single production capability carrier (#700). The debug logger is
// constructed from deps.DebugLog (nil sink → no-op logger) and attached to
// each handler's ctx so downstream code can call debuglog.Log without an
// explicit logger argument.
func Targets(deps Deps) []any {
	errHandler := newErrHandler(deps.Stderr, deps.Exit)
	logger := debuglog.New(deps.DebugLog, "engram", deps.Now)

	withLog := func(ctx context.Context) context.Context {
		return debuglog.WithLogger(ctx, logger)
	}

	return append(
		coreTargets(deps, withLog, errHandler),
		maintenanceTargets(deps, withLog, errHandler)...,
	)
}

// amendResituateTargets returns the amend and resituate subcommands. Split out
// of maintenanceTargets to stay within the per-function length budget.
func amendResituateTargets(
	deps Deps,
	withLog func(context.Context) context.Context,
	errHandler func(error),
	home string,
) []any {
	return []any{
		targ.Targ(func(ctx context.Context, a ResituateArgs) {
			a.Vault = resolveVault(a.Vault, home, deps.Getenv)
			errHandler(RunResituate(withLog(ctx), a, newOsResituateDeps(), deps.Stdout))
		}).Name("resituate").Description("Rewrite a note's situation in sync (frontmatter + body + sidecar) (D4/INV-S2)"),
		targ.Targ(func(ctx context.Context, a AmendArgs) {
			a.Vault = resolveVault(a.Vault, home, deps.Getenv)
			a.ChunksDir = ResolveChunksDir(a.ChunksDir, home, deps.Getenv)
			errHandler(RunAmend(withLog(ctx), a, newOsAmendDeps(), deps.Stdout))
		}).Name("amend").Description("Amend a note in place: supersedes, provenance-merge, field-replacement, activate"),
	}
}

// coreTargets returns the primary subcommands (learn, update, embed, query,
// show, check). Split from Targets to stay within the per-function length
// budget; the wiring mirrors maintenanceTargets exactly.
func coreTargets(
	deps Deps,
	withLog func(context.Context) context.Context,
	errHandler func(error),
) []any {
	return append(
		ingestQueryTargets(deps, withLog, errHandler),
		learnUpdateTargets(deps, withLog, errHandler)...,
	)
}

// homeOrEmpty returns the user's home directory via the injected capability,
// or "" when it cannot be resolved (or is unwired, as in minimal test Deps).
// resolveVault tolerates an empty home (it falls back to env / XDG), so the
// error is intentionally discarded.
func homeOrEmpty(deps Deps) string {
	if deps.UserHomeDir == nil {
		return ""
	}

	home, _ := deps.UserHomeDir()

	return home
}

// ingestQueryTargets returns the read/write-vault subcommands (query, ingest,
// query-chunks, activate, show, check). Split from coreTargets to stay within
// the per-function length budget.
func ingestQueryTargets(
	deps Deps,
	withLog func(context.Context) context.Context,
	errHandler func(error),
) []any {
	home := homeOrEmpty(deps)

	return []any{
		targ.Targ(func(ctx context.Context, a QueryArgs) {
			a.VaultPath = resolveVault(a.VaultPath, home, deps.Getenv)
			a.ChunksDir = ResolveChunksDir(a.ChunksDir, home, deps.Getenv)
			errHandler(RunQuery(withLog(ctx), a, newOsQueryDeps(), deps.Stdout))
		}).Name("query").Description("Semantic search over vault + chunk index (YAML output)"),
		targ.Targ(func(ctx context.Context, a IngestArgs) {
			a.ChunksDir = ResolveChunksDir(a.ChunksDir, home, deps.Getenv)
			errHandler(RunIngest(withLog(ctx), a, newOsIngestDeps(), deps.Stdout))
		}).Name("ingest").Description("Chunk+embed transcripts/markdown into a chunk index (zero-LLM)"),
		targ.Targ(func(ctx context.Context, a PruneArgs) {
			a.ChunksDir = ResolveChunksDir(a.ChunksDir, home, deps.Getenv)
			errHandler(RunPrune(withLog(ctx), a, newOsPruneDeps(), deps.Stdout))
		}).Name("prune").Description(
			"Detach chunk entries whose source file is gone: drop the stale manifest entry, " +
				"keep the embedded chunks (still searchable)"),
		targ.Targ(func(ctx context.Context, a ChunkQueryArgs) {
			a.ChunksDir = ResolveChunksDir(a.ChunksDir, home, deps.Getenv)
			errHandler(RunChunkQuery(withLog(ctx), a, newOsChunkQueryDeps(), deps.Stdout))
		}).Name("query-chunks").Description("Semantic search over the chunk index (YAML output)"),
		targ.Targ(func(_ context.Context, a ActivateArgs) {
			a.Vault = resolveVault(a.Vault, home, deps.Getenv)
			errHandler(RunActivate(a, newOsActivateDeps()))
		}).Name("activate").Description("Mark note(s) as recently used (bumps LastUsed in sidecar)"),
		targ.Targ(func(_ context.Context, a CountArgs) {
			a.Vault = resolveVault(a.Vault, home, deps.Getenv)
			errHandler(RunCount(a, newOsCountDeps(), deps.Stdout))
		}).Name("count").Description(
			"Count notes by a frontmatter attribute or a note's wikilink in-degree (read-only)"),
		targ.Targ(func(ctx context.Context, a ShowArgs) {
			a.VaultPath = resolveVault(a.VaultPath, home, deps.Getenv)
			errHandler(RunShow(withLog(ctx), a, newOsShowDeps(), deps.Stdout))
		}).Name("show").Description("Print a note and its outbound wikilink targets (read-only)"),
		targ.Targ(func(ctx context.Context, a ShowChunkArgs) {
			a.ChunksDir = ResolveChunksDir(a.ChunksDir, home, deps.Getenv)
			errHandler(RunShowChunk(withLog(ctx), a, newOsShowChunkDeps(), deps.Stdout))
		}).Name("show-chunk").Description("Print a chunk's text by its source#anchor id (read-only)"),
		targ.Targ(func(ctx context.Context, a CheckArgs) {
			a.VaultPath = resolveVault(a.VaultPath, home, deps.Getenv)
			errHandler(RunCheck(withLog(ctx), a, newOsCheckDeps(), deps.Stdout))
		}).Name("check").Description("Run vault-invariant checks (exit non-zero on FAIL)"),
	}
}

// learnUpdateTargets returns the learn and update subcommands (learn group,
// update, embed group). Split from coreTargets to stay within the
// per-function length budget.
func learnUpdateTargets(
	deps Deps,
	withLog func(context.Context) context.Context,
	errHandler func(error),
) []any {
	home := homeOrEmpty(deps)

	return []any{
		targ.Group("learn",
			targ.Targ(func(ctx context.Context, a LearnFeedbackArgs) {
				a.Vault = resolveVault(a.Vault, home, deps.Getenv)
				errHandler(runLearnFromFeedbackArgs(withLog(ctx), a, deps.Stdout))
			}).Name("feedback").Description("Write a feedback note to the vault"),
			targ.Targ(func(ctx context.Context, a LearnFactArgs) {
				a.Vault = resolveVault(a.Vault, home, deps.Getenv)
				errHandler(runLearnFromFactArgs(withLog(ctx), a, deps.Stdout))
			}).Name("fact").Description("Write a fact note to the vault"),
			targ.Targ(func(ctx context.Context, a LearnQAArgs) {
				a.Vault = resolveVault(a.Vault, home, deps.Getenv)
				errHandler(RunLearnQA(withLog(ctx), a, newOsLearnQADeps(), deps.Stdout))
			}).Name("qa").Description("Write a QA pair (Q+A notes) to the vault"),
		),
		targ.Targ(func(ctx context.Context, a UpdateArgs) {
			errHandler(runUpdate(withLog(ctx), a, deps.Stdout))
		}).Name("update").Description("Refresh engram binary and harness skills"),
		targ.Group("embed",
			targ.Targ(func(ctx context.Context, a EmbedApplyArgs) {
				a.VaultPath = resolveVault(a.VaultPath, home, deps.Getenv)
				errHandler(RunEmbedApply(withLog(ctx), a, newOsEmbedDeps(), deps.Stdout))
			}).Name("apply").Description("Embed notes (default: missing only)"),
			targ.Targ(func(ctx context.Context, a EmbedStatusArgs) {
				a.VaultPath = resolveVault(a.VaultPath, home, deps.Getenv)
				errHandler(RunEmbedStatus(withLog(ctx), a, newOsEmbedDeps(), deps.Stdout))
			}).Name("status").Description("Report embedding state counts"),
		),
	}
}

// maintenanceTargets returns the vault-maintenance subcommands (resituate,
// amend, vocab). Split out of Targets to keep each function within the length budget;
// the wiring mirrors the other targets exactly.
func maintenanceTargets(
	deps Deps,
	withLog func(context.Context) context.Context,
	errHandler func(error),
) []any {
	home := homeOrEmpty(deps)

	return append(
		amendResituateTargets(deps, withLog, errHandler, home),
		vocabTargets(deps, withLog, errHandler, home)...,
	)
}

// vocabTargets returns the vocab group subcommands (bootstrap, stats,
// propose, refit).
func vocabTargets(
	deps Deps,
	withLog func(context.Context) context.Context,
	errHandler func(error),
	home string,
) []any {
	return []any{
		targ.Group("vocab",
			targ.Targ(func(ctx context.Context, a VocabBootstrapArgs) {
				a.Vault = resolveVault(a.Vault, home, deps.Getenv)
				errHandler(RunVocabBootstrap(withLog(ctx), a, newOsVocabDeps(), deps.Stdout))
			}).Name("bootstrap").Description("Seed vocab term notes + tag all existing notes (idempotent)"),
			targ.Targ(func(_ context.Context, a VocabStatsArgs) {
				a.Vault = resolveVault(a.Vault, home, deps.Getenv)
				errHandler(RunVocabStats(a, newOsVocabStatsDeps(), deps.Stdout))
			}).Name("stats").Description("Print vocab health report (per-term counts, hubs, orphans, untagged rate)"),
			targ.Targ(func(ctx context.Context, a VocabProposeArgs) {
				a.Vault = resolveVault(a.Vault, home, deps.Getenv)
				errHandler(RunVocabPropose(withLog(ctx), a, newOsVocabDeps(), deps.Stdout))
			}).Name("propose").Description("Add a new vocab term note + minor version bump (LLM gate runs agent-side)"),
			targ.Targ(func(ctx context.Context, a VocabRefitArgs) {
				a.Vault = resolveVault(a.Vault, home, deps.Getenv)
				errHandler(RunVocabRefit(withLog(ctx), a, newOsVocabDeps(), deps.Stdout))
			}).Name("refit").Description("Apply a refit plan: renames, removals, additions, re-tag, major version bump"),
		),
	}
}
```

   Exhaustive closure inventory (current line numbers in targets.go; T2 change is `os.Getenv`→`deps.Getenv`, `stdout`→`deps.Stdout` shown above; final-form constructor swap is a later one-liner owned by each command cluster):

   | Closure | Lines | Constructor today (kept in T2) | Final form (owning cluster) |
   |---|---|---|---|
   | resituate | 106–109 | `newOsResituateDeps()` | `newResituateDeps(deps)` (resituate) |
   | amend | 110–114 | `newOsAmendDeps()` | `newAmendDeps(deps)` (amend) |
   | query | 152–156 | `newOsQueryDeps()` | `newQueryDeps(deps)` (query) |
   | ingest | 157–160 | `newOsIngestDeps()` | `newIngestDeps(deps)` (ingest) |
   | prune | 161–166 | `newOsPruneDeps()` | `newPruneDeps(deps)` (ingest/prune) |
   | query-chunks | 167–170 | `newOsChunkQueryDeps()` | `newChunkQueryDeps(deps)` (query) |
   | activate | 171–174 | `newOsActivateDeps()` | `newActivateDeps(deps)` (activate) |
   | count | 175–179 | `newOsCountDeps()` | `newCountDeps(deps)` (count) |
   | show | 180–183 | `newOsShowDeps()` | `newShowDeps(deps)` (show) |
   | show-chunk | 184–187 | `newOsShowChunkDeps()` | `newShowChunkDeps(deps)` (query — T6) |
   | check | 188–191 | `newOsCheckDeps()` | `newCheckDeps(deps)` (check) |
   | learn feedback | 207–210 | inside `runLearnFromFeedbackArgs` (learn.go:520) | `runLearnFromFeedbackArgs(ctx, a, deps, stdout)` → `newLearnDeps(deps)` (learn) |
   | learn fact | 211–214 | inside `runLearnFromFactArgs` (learn.go:497) | `runLearnFromFactArgs(ctx, a, deps, stdout)` → `newLearnDeps(deps)` (learn) |
   | learn qa | 215–218 | `newOsLearnQADeps()` | `newLearnQADeps(deps)` (learn) |
   | update | 220–222 | inside `runUpdate` (update.go:276) | `runUpdate(ctx, a, deps, stdout)` using `deps.FS`/`deps.Commander`/`deps.Getenv` (update) |
   | embed apply | 224–227 | `newOsEmbedDeps()` | `newEmbedDeps(deps)` using `deps.Embed` (embed) |
   | embed status | 228–231 | `newOsEmbedDeps()` | `newEmbedDeps(deps)` (embed) |
   | vocab bootstrap | 276–279 | `newOsVocabDeps()` | `newVocabDeps(deps)` (vocab) |
   | vocab stats | 280–283 | `newOsVocabStatsDeps()` | `newVocabStatsDeps(deps)` (vocab) |
   | vocab propose | 284–287 | `newOsVocabDeps()` | `newVocabDeps(deps)` (vocab) |
   | vocab refit | 288–291 | `newOsVocabDeps()` | `newVocabDeps(deps)` (vocab) |

   `os.Getenv` sites replaced (all 22): lines 107, 111, 112, 153, 154, 158, 162, 168, 172, 176, 181, 185, 189, 208, 212, 216, 225, 229, 277, 281, 285, 289. `os.UserHomeDir` site replaced: line 136 (`homeOrEmpty`).

4. [ ] Update the 8 test call sites. Add to `internal/cli/targets_test.go` (package cli_test; add `"io"` and keep `"os"`/`"time"` imports). NOTE (R11): `newTestDeps` builds a `cli.Deps` literal DIRECTLY — it does NOT call `cli.NewDeps`, so targets-level tests never construct the lazy embedder or register signal watchers; the composition path has its own tests (T1-rework):

```go
// newTestDeps builds a cli.Deps wired to real OS capabilities with captured
// stdout/stderr and a no-op exit — the test analog of the production
// cli.NewDeps composition (built directly so no embedder/signal wiring
// occurs). Command clusters extend this as their constructors convert to
// Deps-based composition (#700).
func newTestDeps(stdout, stderr io.Writer) cli.Deps {
	return cli.Deps{
		Stdout:      stdout,
		Stderr:      stderr,
		Exit:        func(int) {},
		Getenv:      os.Getenv,
		Now:         time.Now,
		Getwd:       os.Getwd,
		UserHomeDir: os.UserHomeDir,
	}
}
```

   Replace each site:
   - targets_test.go:153, 289, 307 — current: `targets := cli.Targets(&bytes.Buffer{}, &bytes.Buffer{}, func(int) {}, nil)` → `targets := cli.Targets(newTestDeps(&bytes.Buffer{}, &bytes.Buffer{}))`
   - targets_test.go:439 (inside `executeForTest`) — current: `targets := cli.Targets(&stdout, &stderr, func(int) {}, nil)` → `targets := cli.Targets(newTestDeps(&stdout, &stderr))`
   - vocab_commands_test.go:3580, 3606, 3630, 3655 — current: `targets := cli.Targets(&stdout, &stderr, func(int) {}, nil)` → `targets := cli.Targets(newTestDeps(&stdout, &stderr))`

5. [ ] Composition-root cutover, in four sub-edits (one commit):

   **5a.** Modify `internal/cli/primitives.go` — `NewDeps` gains the internal Embed wiring (this is the R6 arity-handoff point: T14 later edits THIS line to the 3-arg constructor). Restructure the return into a named `deps` so the guarded field-set reads cleanly, and add `"github.com/toejough/engram/internal/embed"` to the imports:

```go
func NewDeps(prims Primitives, stdout, stderr io.Writer, exit func(int)) Deps {
	startForceExit(prims, exit)

	deps := Deps{
		Stdout:      stdout,
		Stderr:      stderr,
		Exit:        exit,
		Getenv:      prims.Getenv,
		Now:         prims.Now,
		Getwd:       prims.Getwd,
		UserHomeDir: prims.UserHomeDir,
		FS:          primFS{prims: prims},
		Lock:        primLocker{prims: prims},
		DebugLog:    openDebugSink(envOrEmpty(prims.Getenv, debugLogEnvVar), prims.OpenDebugFile),
	}

	// The lazy embedder is constructed once here, preserving the
	// one-unpack-per-process property of the old sharedEmbedder singleton
	// (guarded: minimal fake Primitives without Getenv skip it). R6: T14
	// swaps this line to the 3-arg constructor over cmd-injected backend
	// and cache capabilities.
	if prims.Getenv != nil {
		deps.Embed = embed.NewLazyEmbedder(
			CacheDirFromHome(homeOrEmpty(deps), embed.BundledModelID, prims.Getenv))
	}

	return deps
}
```

   **5b.** Rewrite `cmd/engram/main.go` (replaces the whole file). Package main becomes DECLARATION-FREE: `main()` is ONE statement — a single external call, which is exactly what `checkFuncThinness` accepts (doctrine flag SIG-1: any second statement in main() FAILS the gate) — and every raw capability enters as a direct func reference or sanctioned closure inside the `cli.Primitives` literal (closures are expressions, not declarations; the checker does not walk them — the doctrine's closure rule caps them at single-call signature erasure or an enumerated stdlib-equivalent survivor: S-1 `WriteFileExcl` below, C-1 `RunCommand` in T17). Signal registration happens inside `cli.NewDeps` (via `StartSignalPulses` + internal `startForceExit`) during argument evaluation — strictly BEFORE `targ.Main` runs, preserving the handler-covers-the-whole-run property:

```go
// Package main provides the engram CLI binary entry point (ARCH-6). It is
// deliberately declaration-free: raw impure capabilities enter as func
// references and sanctioned closures in the cli.Primitives literal, and
// ALL composition, error wrapping, and lifecycle logic lives in
// internal/cli (targ check-thin-api enforces this shape; #700).
package main

import (
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/toejough/targ"

	"github.com/toejough/engram/internal/cli"
)

// FIXME(#700): internal-purity migration in progress — this marker tracks the
// unresolved issue. The original internal/cli/main.go os.Getenv violation is
// fixed (env enters via cli.Primitives.Getenv), but adapter/env-threading/
// enforcement tasks are still landing. Remove this marker ONLY in T-final-2,
// after the depguard/forbidigo gate is verified green.
func main() {
	targ.Main(cli.Targets(cli.NewDeps(cli.Primitives{
		ReadFile:    os.ReadFile,
		WriteFile:   os.WriteFile,
		MkdirAll:    os.MkdirAll,
		MkdirTemp:   os.MkdirTemp,
		Stat:        os.Stat,
		ReadDir:     os.ReadDir,
		Remove:      os.Remove,
		RemoveAll:   os.RemoveAll,
		Rename:      os.Rename,
		WalkDir:     filepath.WalkDir,
		Chmod:       os.Chmod,
		Getenv:      os.Getenv,
		Now:         time.Now,
		Getwd:       os.Getwd,
		UserHomeDir: os.UserHomeDir,
		WriteFileExcl: func(path string, data []byte, perm fs.FileMode) error {
			// Doctrine survivor S-1: os.WriteFile's own body with
			// O_CREATE|O_EXCL — mechanical error propagation only; behavior
			// changes extend the Primitives SIGNATURE, never this body.
			//nolint:gosec // operator-controlled path
			file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm)
			if err != nil {
				return err
			}

			_, err = file.Write(data)
			if closeErr := file.Close(); closeErr != nil && err == nil {
				err = closeErr
			}

			return err
		},
		OpenLockFile: func(path string, perm fs.FileMode) (uintptr, error) {
			fd, err := syscall.Open(path, syscall.O_CREAT|syscall.O_RDWR, uint32(perm))

			return uintptr(fd), err
		},
		FlockExclusive: func(fd uintptr) error {
			return syscall.Flock(int(fd), syscall.LOCK_EX)
		},
		FlockUnlock: func(fd uintptr) error {
			return syscall.Flock(int(fd), syscall.LOCK_UN)
		},
		CloseFD: func(fd uintptr) error {
			return syscall.Close(int(fd))
		},
		OpenDebugFile: func(path string, perm fs.FileMode) (cli.WriteSyncer, error) {
			// Path comes from operator-set ENGRAM_DEBUG_LOG, not user input.
			//nolint:gosec // operator-controlled path
			return os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, perm)
		},
		StartSignalPulses: func(pulses chan<- struct{}, buffer int) {
			sigCh := make(chan os.Signal, buffer)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

			go cli.ForwardAsPulses(sigCh, pulses)
		},
	}, os.Stdout, os.Stderr, os.Exit))...)
}
```

   If `targ check-full` flags additional gosec G304 sites in the closures, add the same targeted, justified `//nolint:gosec` line — never a config-level suppression. If `check-thin-api` itself flags ANYTHING in this file, escalate the exact finding (doctrine flag SIG-1).

   **5c.** Delete `internal/cli/main.go` (the old `Main`; its ENGRAM_DEBUG_LOG read now lives in `NewDeps` via the Getenv primitive — T1-rework). The FIXME(#700) marker does NOT die with the file: it is relocated to `cmd/engram/main.go` per R8 (the comment block above `func main()` in 5b — comments are legal in the declaration-free package).

   **5d.** Modify `internal/cli/signal.go` — delete `SetupSignalHandling` (the whole func), its `io`/`os`/`os/signal`/`syscall`/debuglog imports, and its doc comment. KEEP: `ExitCodeSigInt`, `ForceExitOnRepeatedSignal`, `ForwardAsPulses`, `startForceExit`, `secondSignal`, AND `signalChannelBuffer` (consumed by `startForceExit` — this corrects the pre-rework T2 text, which deleted the const). Final signal.go imports: `sync/atomic` only.

6. [ ] Run `targ test` — expect all green: debuglog tests (new API), cli tests (new Targets + T1-rework composition suites), plus cli_test.go's end-to-end binary build (`go build ./cmd/engram`) still passing.

7. [ ] Purity verification (expect zero matches from the first three; fourth expect an `ls` error — file gone):

```
grep -n "os\." internal/cli/targets.go internal/cli/signal.go internal/debuglog/debuglog.go
grep -rn "SetupSignalHandling" --include="*.go" .
grep -n "time.Now\|time.Since" internal/debuglog/debuglog.go
ls internal/cli/main.go
```

   Expected: no `os.` in the three files; no `SetupSignalHandling` anywhere; no `time.Now`/`time.Since` in debuglog.go; `ls` errors (file deleted). Also verify the FIXME survived relocation: `rg "FIXME\(#700\)" cmd/engram/main.go` → exactly one hit (R8).

8. [ ] Run `targ check-thin-api` — expect PASS (`All N public API files are thin wrappers.`): cmd/engram holds only the declaration-free main.go. Escalate any finding (doctrine flag SIG-1); do not suppress.

9. [ ] Run `targ check-full` — expect clean: T1-rework's known `SetupSignalHandling` coverage residual is resolved by 5d's deletion.

10. [ ] Run the real binary (usable-system check): `go install ./cmd/engram && engram show-chunk --help` and `ENGRAM_DEBUG_LOG=/tmp/engram-700.log engram count --vault "$(mktemp -d)" --attribute type` then `cat /tmp/engram-700.log` — expect help text, a zero count table, and timestamped `[engram]` debug lines proving the primitive → NewDeps → sink → logger wiring is live end-to-end.

11. [ ] Commit:

```
refactor(cli): single-statement main over Primitives literal (#700)

Targets now takes the cli.Deps capability carrier; cmd/engram/main.go
becomes a declaration-free package whose main() is one statement wiring
raw primitives (os/syscall/filepath references and sanctioned closures)
into cli.NewDeps. Deletes cli.Main and SetupSignalHandling, drops the os
import from targets.go, purifies debuglog (injected writer + clock, nil
no-op), and wires the lazy embedder inside NewDeps (R6/D-1). FIXME(#700)
marker relocates to cmd/engram/main.go per R8.

AI-Used: [claude]
```

---

Key file paths: `/Users/joe/repos/personal/engram/.claude/worktrees/700-internal-purity/internal/cli/targets.go` (81–294 rewritten), `/Users/joe/repos/personal/engram/.claude/worktrees/700-internal-purity/internal/cli/signal.go`, `/Users/joe/repos/personal/engram/.claude/worktrees/700-internal-purity/internal/cli/primitives.go` (+ edgefs.go, locker.go, debugsink.go from T1-rework), `/Users/joe/repos/personal/engram/.claude/worktrees/700-internal-purity/internal/cli/main.go` (deleted in T2), `/Users/joe/repos/personal/engram/.claude/worktrees/700-internal-purity/internal/debuglog/debuglog.go`, `/Users/joe/repos/personal/engram/.claude/worktrees/700-internal-purity/cmd/engram/main.go` (single-statement rewrite; the six landed cmd adapter files are deleted by T1-rework).

