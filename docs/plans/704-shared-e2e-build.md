# Plan: #704 — share one engram binary build across the internal/cli e2e suite

## Ask (verbatim)

"do 704" — issue #704: *internal/cli e2e tests: share one binary build across the suite (five per-run
builds of the 90MB-embed binary dominate cost).*

## Verified pre-flight facts

- Exactly five build sites, each `exec.Command("go", "build", "-o", binPath, "./cmd/engram")` into its
  own `t.TempDir()`: `internal/cli/cli_test.go:23,85,158,235` and
  `internal/cli/query_integration_test.go:67`.
- Fresh failure evidence (#705 cycle, posted on #704): under check-full's 8-way load the coverage leg
  timed out at 10m with three concurrent builds observed live; the same leg runs 21.5s isolated.
- Notes 129/131: parallel tests must not share MUTABLE state. A once-built, never-mutated binary path is
  shared immutable state; each test keeps its own vault/cache/XDG tempdirs.
- Note 408 sweep: the package's only `time.Sleep` (ingest_test.go:909) deliberately widens a race window
  — load-safe, stays.
- Doc-surface grep ("five per-run builds"/"five builds"/"per-test build" over docs/, .superpowers/,
  CLAUDE.md, README.md, .claude/): zero hits. No doc scrub needed.

## Design

Add one helper in a new `internal/cli/testbinary_test.go` (package `cli_test`, same as the five tests):

- `sharedEngramBinary(t *testing.T) string` — `sync.Once`-guarded: `os.MkdirTemp("", "engram-e2e-bin")`,
  `go build -o <dir>/engram ./cmd/engram` (same working-dir resolution the five sites use today), store
  path + build error in package-level vars. Every caller asserts the stored build error via the same
  gomega/`t.Fatal` style the sites use now, then uses the stored path. Lazy (`sync.Once` in the helper,
  no `TestMain`) so non-e2e runs never pay the build.
- The binary and its temp dir are written once inside the `Once` and never mutated afterward — shared
  immutable state, per notes 129/131. `t.Parallel()` stays on every test (note 364).
- Cleanup: the dir lands in the OS temp root, same lifecycle as today's leaked `t.TempDir()` residue on
  panic; no `TestMain` teardown (YAGNI — the OS sweeps its temp root, and `go test`'s own tempdir
  handling already tolerates this).
- Replace the five build blocks with `binPath := sharedEngramBinary(t)`. No other behavior changes:
  each test keeps its own env pinning (`XDG_CACHE_HOME` to `t.TempDir()`, etc.).

## Tasks

1. RED (measured baseline, the closest analogue for a build-topology refactor): record
   `grep -c 'exec.Command("go", "build"' internal/cli/*_test.go` = 5, and time
   `targ test` as the before-measurement.
2. Implement the helper + five call-site replacements.
3. GREEN: grep count = 1 (the helper); `targ test` green with all five e2e tests running (not skipped);
   record the after-time.
4. Commit, then run `targ check-full` from the repo root. Expected all 8 gates green. Any failure stops
   the close: collect the complete list and report to Joe for disposition.
5. Gate B (design-fit review) over the diff — this unit IS a refactor.
6. Close #704 citing grep-count 1, suite timings before/after, and the full-gate summary. Then delete
   this plan file in the workflow's completion step as part of the final commit.

## Risks

- The five sites may differ subtly in build env/working dir — implementer must diff the five blocks
  before unifying; any divergence is a STOP-and-report, not a silent normalization.
- check-full remains load-sensitive until this lands; a coverage-leg timeout during Task 4 gets one
  isolated re-run (per the #705 precedent) before being treated as a failure of this change.
- Run all targ commands from the engram checkout only (note 359).
