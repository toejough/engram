# Plan: #704 — share one engram binary build across the internal/cli e2e suite

## Ask (verbatim)

"do 704" — issue #704: *internal/cli e2e tests: share one binary build across the suite (five per-run
builds of the 90MB-embed binary dominate cost).* The issue's acceptance sketch has three bullets:
one shared build, warm-cache env inheritance preserved for the model-loading tests, and the stale
`NewLazyEmbedder` doc comment fixed.

## Verified pre-flight facts

- Exactly five build sites, each `exec.Command("go", "build", "-o", binPath, "./cmd/engram")` into its
  own `t.TempDir()`: `internal/cli/cli_test.go:23,85,158,235` and `internal/cli/query_integration_test.go:67`.
  (The issue body cites older line numbers from before main advanced; Gate A's code reviewer re-verified
  the current five and diffed them: semantically identical — same `cmd.Dir = projectRoot(t)`, same
  `CombinedOutput()` + gomega assertion. Unification is safe.)
- Both files are `package cli_test`; `projectRoot(t)` (cli_test.go:336) is package-scoped and reusable.
- No test mutates or moves its built binary (grep for Rename/Chmod/Remove on binPath: zero hits).
- Fresh failure evidence (#705 cycle, posted on #704): under check-full's 8-way load the coverage leg
  timed out at 10m with three concurrent builds observed live; the same leg runs 21.5s isolated.
- Notes 129/131: parallel tests must not share MUTABLE state. A once-built, never-mutated binary path is
  shared immutable state; each test keeps its own vault/cache/XDG tempdirs.
- `internal/embed/hugot.go:187-191` (`NewLazyEmbedder` doc comment) still claims init triggers on
  "first Embed / ModelID / Dims call"; `ModelID()` (line 229) short-circuits to `BundledModelID` without
  constructing the embedder — the comment is stale, and fixing it is an acceptance bullet of #704.
- Note 408 sweep: the package's only `time.Sleep` (ingest_test.go:909) deliberately widens a race window
  — load-safe, stays.
- Doc-surface grep ("five per-run builds"/"five builds"/"per-test build" over docs/, .superpowers/,
  CLAUDE.md, README.md, .claude/): zero hits, independently re-verified by Gate A's docs reviewer with
  its own broader terms. No doc scrub needed.

## Design

Add one helper in a new `internal/cli/testbinary_test.go` (package `cli_test`):

- `sharedEngramBinary(t *testing.T) string` — `t.Helper()`; `sync.Once`-guarded build:
  `os.MkdirTemp("", "engram-e2e-bin")`, `go build -o <dir>/engram ./cmd/engram` with
  `cmd.Dir = projectRoot(t)` (same resolution the five sites use today), storing path and build error in
  package-level vars. The helper itself calls `t.Fatalf` when the stored error is non-nil and returns
  ONLY the path — call sites carry no error handling, which also sidesteps nilaway's package-var
  tracking (Gate A code-alignment finding).
- The binary and its dir are written once inside the `Once`, never mutated — shared immutable state
  (notes 129/131). `t.Parallel()` stays on every test (note 364).
- No `TestMain` teardown: the dir lives in the OS temp root, same lifecycle as today's panic-leaked
  `t.TempDir()` residue (YAGNI). No unit tests for the helper itself: it is test infrastructure whose
  behavior is exercised by the five consuming tests under `-race`; testing `sync.Once` semantics tests
  the stdlib (repo test-categorization rule).
- Replace the five build blocks with `binPath := sharedEngramBinary(t)`. No other behavior changes:
  each subprocess invocation keeps `append(os.Environ(), ...)` env construction and per-test
  `XDG_CACHE_HOME`/vault tempdirs.

## Tasks

1. Drift check (STOP guard): `grep -n 'exec.Command("go", "build"' internal/cli/*_test.go` must return
   exactly the five pre-flight sites. Any other count or different files: STOP, report, re-verify before
   implementing. Then record the RED baseline: build-site count 5, and time `targ test` (before-measurement).
2. Implement `testbinary_test.go` + the five call-site replacements per Design.
3. GREEN: build-site grep count = 1 (the helper); `targ test` green with all five e2e tests running
   (not skipped); record the after-time. Verify warm-cache inheritance explicitly: the three
   model-loading tests' invocation envs still start from `os.Environ()` (grep) and the suite shows no
   cold-extract slowdown vs the Task-1 baseline.
4. Fix `internal/embed/hugot.go:187-191` doc comment to the actual trigger set (first Embed or Dims
   call; ModelID returns the bundled ID without init).
5. Commit, then run `targ check-full` from the repo root (blocking gate). If any gate fails: HALT —
   do not proceed to any later task, do not retry-fix; collect the complete failure list (all gates, not
   first-failure) and report to Joe, waiting on his disposition. Sole exception: one isolated re-run is
   allowed for a coverage-leg timeout, per the #705 load-contention precedent.
6. Gate B (design-fit review) over the diff — this unit IS a refactor.
7. Close #704 citing build-count 1, suite timings before/after, and the full-gate summary. Delete this
   plan file in the same final commit.

## Risks

- check-full remains load-sensitive until this lands (see Task 5's single-retry rule).
- Run all targ commands from the engram checkout only (note 359).
