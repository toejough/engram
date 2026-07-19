# T1 Report: Pure signal seam, cli.Deps carrier, cmd/engram OS adapters

## Status

DONE_WITH_CONCERNS — deliverable complete and correct per the brief; two
`targ check-full` gates are red for reasons that are architecturally
load-bearing / self-resolving, not defects in this task's code. Details below.

## Commit

`de484526` — `refactor(cli): add Deps carrier + pure signal seam (#700)`

```
 cmd/engram/debuglog_sink.go      |  50 +++++++++
 cmd/engram/debuglog_sink_test.go |  67 ++++++++++++
 cmd/engram/os_fs.go              | 224 +++++++++++++++++++++++++++++++++++++++
 cmd/engram/os_fs_test.go         | 211 ++++++++++++++++++++++++++++++++++++
 cmd/engram/os_signal.go          |  37 +++++++
 cmd/engram/os_signal_test.go     |  81 ++++++++++++++
 internal/cli/deps.go             |  72 +++++++++++++
 internal/cli/signal.go           |  24 ++++++++++++++-----
 internal/cli/signal_test.go      |  32 +++++-----------------
 9 files changed, 769 insertions(+), 28 deletions(-)
```

`.superpowers/` remains untracked (pre-existing at session start, unrelated
to T1 — excluded from this commit).

## Steps completed (brief checkboxes 1–11)

1. RED — replaced `internal/cli/signal_test.go` with the pure `struct{}`-pulse
   version verbatim. `targ test` failed exactly as predicted:
   `cannot use pulses (variable of type chan struct{}) as <-chan os.Signal
   value in argument to cli.ForceExitOnRepeatedSignal` (2 sites).
2. GREEN — replaced `internal/cli/signal.go` verbatim (pure
   `ForceExitOnRepeatedSignal(<-chan struct{}, func(int))` + interim
   `SetupSignalHandling` adapter). `targ test` green, all packages.
3. Created `internal/cli/deps.go` verbatim (`Deps`, `EdgeFS`, `FileLocker`),
   with one deviation (interfacebloat nolint — see Deviations).
4. RED — created `cmd/engram/os_fs_test.go` per the brief, applying the
   brief's own noted adjustment (no `"os"` import / no `var _ = os.Getpid`,
   since `os` is unreferenced in that file as written). `targ test` failed:
   `undefined: osFS` (6 sites) and `undefined: flockLocker` (1 site).
5. GREEN — created `cmd/engram/os_fs.go` verbatim (osFS + flockLocker).
   `targ test` green. One deviation applied after the check-full pass (see
   Deviations: unused `//nolint:gosec` on `WriteFile`).
6. RED — created `cmd/engram/os_signal_test.go` verbatim. `targ test` failed:
   `undefined: registerForceExit` (compiler stopped at the first undefined
   symbol; `forwardAsPulses` is also undefined at this point).
7. GREEN — created `cmd/engram/os_signal.go` verbatim. `targ test` green,
   including the real-signal `TestRegisterForceExit_SecondSignalForcesExit`
   (SIGUSR2 self-delivery).
8. RED — created `cmd/engram/debuglog_sink_test.go` verbatim. `targ test`
   failed: `undefined: openDebugSink`.
9. GREEN — created `cmd/engram/debuglog_sink.go` verbatim. `targ test` green.
10. Ran `targ check-full`. Did NOT hit the anticipated
    `registerForceExit`/`openDebugSink` unused-symbol risk — `deadcode` and
    `lint-full`'s `unused` both pass, confirming the brief's own prediction
    that the `_test.go` files' references satisfy them. Two *unanticipated*
    gate failures were found and are documented under Deviations/Concerns
    below; none required suppression or redesign.
11. Committed exactly as specified (subject/body verbatim, `AI-Used: [claude]`
    trailer kept).

## Final gate evidence (real output)

`targ test` (final, post-commit state, all packages):

```
ok  	github.com/toejough/engram/cmd/engram	3.400s	coverage: 1.8% of statements in ./...
ok  	github.com/toejough/engram/internal/chunk	1.951s	coverage: 2.0% of statements in ./...
ok  	github.com/toejough/engram/internal/cli	81.100s	coverage: 79.5% of statements in ./...
ok  	github.com/toejough/engram/internal/cluster	3.498s	coverage: 4.5% of statements in ./...
ok  	github.com/toejough/engram/internal/context	1.309s	coverage: 4.8% of statements in ./...
ok  	github.com/toejough/engram/internal/debuglog	3.736s	coverage: 0.5% of statements in ./...
ok  	github.com/toejough/engram/internal/embed	11.257s	coverage: 6.0% of statements in ./...
ok  	github.com/toejough/engram/internal/luhmann	2.209s	coverage: 0.6% of statements in ./...
ok  	github.com/toejough/engram/internal/transcript	1.510s	coverage: 4.2% of statements in ./...
ok  	github.com/toejough/engram/internal/update	1.762s	coverage: 5.7% of statements in ./...
ok  	github.com/toejough/engram/internal/vaultgraph	5.267s	coverage: 3.0% of statements in ./...
ok  	github.com/toejough/engram/skills	4.600s	coverage: 0.0% of statements in ./...
```

`targ check-full` (final, post-commit, `PASS:3 FAIL:5`):

```
  FAIL      check-coverage-for-fail  (53.873s)  exit status 1
  FAIL      check-uncommitted        (32ms)  uncommitted changes found:      <- resolved by the commit itself; this run predates it
  FAIL      reorder-decls-check      (2.938s)  2 file(s) need reordering
  PASS      lint-fast                (3.832s)
  FAIL      lint-full                (3.837s)  exit status 1
  PASS      deadcode                 (50.311s)
  FAIL      check-thin-api           (10ms)  found 9 non-thin declarations
  PASS      check-nils-for-fail      (1m25.198s)
```

I isolated every FAIL against a true baseline (T1's files entirely removed /
reverted, `git stash` + moving `deps.go` aside) to separate pre-existing noise
from T1-introduced findings:

| Gate | Baseline (no T1 changes) | With T1 (final) | New from T1? |
|---|---|---|---|
| `check-coverage-for-fail` | FAIL (flaky panic in `TestTargets_QueryEmptyVault`, `os/exec` goroutine trace, pre-existing) | FAIL (same panic) **+ `SetupSignalHandling` 0.0%** | Yes — one new line, see Concerns #1 |
| `check-uncommitted` | PASS | FAIL pre-commit / PASS post-commit | No — timing artifact only |
| `reorder-decls-check` | FAIL (2 pre-existing `dev/eval/**/testdata` fixture files) | FAIL (same 2 files, byte-identical) | No |
| `lint-full` | FAIL (19 issues, all in `internal/transcript` and `internal/vaultgraph` generated mock test files) | FAIL (same 19 issues, identical set) | No |
| `deadcode` | PASS | PASS | No |
| `check-thin-api` | PASS | **FAIL (9 declarations, all in T1's 3 new cmd/engram files)** | Yes — see Concerns #2 |
| `check-nils-for-fail` | PASS | PASS | No |

So T1 introduces exactly two new findings, both understood and neither
addressable by a minimal, honest fix within T1's scope (see Concerns).
Everything else is either pre-existing/out-of-scope noise or resolves at
commit time.

## Files touched

- `internal/cli/deps.go` (new) — `Deps`, `EdgeFS`, `FileLocker`
- `internal/cli/signal.go` (modified) — pure `ForceExitOnRepeatedSignal`, interim `SetupSignalHandling` adapter
- `internal/cli/signal_test.go` (modified) — pure-signature tests
- `cmd/engram/os_fs.go` (new) — `osFS`, `flockLocker`
- `cmd/engram/os_fs_test.go` (new)
- `cmd/engram/os_signal.go` (new) — `registerForceExit`, `forwardAsPulses`
- `cmd/engram/os_signal_test.go` (new)
- `cmd/engram/debuglog_sink.go` (new) — `openDebugSink`, `syncWriter`
- `cmd/engram/debuglog_sink_test.go` (new)

## Deviations from the brief's verbatim code (all minimal, no redesign)

1. **`internal/cli/deps.go` — `//nolint:interfacebloat` added to `EdgeFS`.**
   The brief's `EdgeFS` interface (11 methods) is exactly as specified in
   `constraints-and-resolutions.md` (which even documents a future task
   adding a 12th method, `WriteFileExcl`). `golangci-lint`'s `interfacebloat`
   linter (default threshold 10, enabled via targ's `default = 'all'`
   config, no local repo override exists) flags this as `internal/cli/deps.go:45:13
   interfacebloat the interface has more than 10 methods: 11` — confirmed
   this is the *only* new lint finding versus baseline (isolated by
   temporarily removing deps.go and diffing `targ lint-full` output).
   Splitting EdgeFS would be an architectural redesign explicitly forbidden
   by the constraints doc ("do not redesign mid-task... any forced departure
   is a DESIGN FLAG"), so I added a targeted, justified `//nolint` comment
   (the same convention already used in the brief's own `os_fs.go` code for
   `//nolint:gosec`) rather than suppressing at the config level (which the
   user's CLAUDE.md forbids) or redesigning the interface.

2. **`cmd/engram/os_fs.go` — removed the `//nolint:gosec // path from caller`
   comment on `WriteFile`.** `nolintlint` flagged it as unused: gosec's G304
   ("potential file inclusion via variable") only fires for read-oriented
   `os.*` calls (confirmed: the `ReadFile` and `flockLocker.Lock`
   `//nolint:gosec` comments in the same file are NOT flagged as unused —
   only `WriteFile`'s is). Removed the single unused directive; left the
   others as written.

3. **`cmd/engram/os_fs_test.go` — added one blank line** before
   `g.Expect(fsys.WriteFile(oldPath, ...))` in
   `TestOsFS_RenameRemoveRemoveAll` (`wsl_v5: missing whitespace above this
   line (no shared variables above expr)`). Purely cosmetic; no logic
   change.

4. **`cmd/engram/os_fs_test.go` — omitted the `"os"` import and the
   `var _ = os.Getpid` silence-line.** This is not really a deviation — the
   brief's own step 4 text explicitly instructs this exact adjustment
   ("Drop the final `var _` line if `os` ends up referenced; as written `os`
   is unreferenced in this file — remove the `"os"` import and the `var _`
   line instead. Final import list: `io/fs`, `path/filepath`, `testing`,
   `time`, gomega."), so I applied it directly rather than writing the
   dead-code version first.

5. **Ran `targ reorder-decls`** (repo-wide auto-fixer) to fix declaration
   ordering in the 6 new/modified T1 files that `reorder-decls-check`
   flagged. This also touched 2 unrelated `dev/eval/**/testdata` fixture
   files (pre-existing baseline failures, part of an eval harness's test
   fixtures, out of T1's scope and possibly load-bearing for that harness's
   own test semantics) — I reverted those 2 files with `git checkout --`
   before committing, keeping T1's diff scoped to its file list.

## Self-review notes

- Diffed every new/modified file against the brief's verbatim code blocks;
  confirmed byte-for-byte match except the 5 deviations above, all recorded.
- Confirmed `t.Parallel()` present on every test and subtest (brief's own
  code already does this).
- Confirmed nilaway-safe error-guard pattern (`if err != nil { return }`
  after every `g.Expect(err).NotTo(HaveOccurred())`) is present throughout
  — brief's code already followed this; `check-nils-for-fail` passes clean.
- Confirmed `TestRegisterForceExit_SecondSignalForcesExit` (real SIGUSR2
  self-delivery) and `TestFlockLocker_SecondLockWaitsForUnlock` (real flock
  concurrency) both pass reliably (ran `targ test` multiple times across the
  session with no flakes from these two).
- Verified `git diff` for `internal/cli/signal.go` / `signal_test.go`
  matches the brief's replacement text exactly.
- Verified no stray files (`.superpowers/`) were swept into the commit.
- Verified `internal/update/update.go` was not touched (out of T1's file
  list; global constraint requires it stay minimal for concurrent Pi
  worktrees).

## Concerns for the orchestrator

**#1 — `SetupSignalHandling` 0.0% function coverage (check-coverage-for-fail).**
The brief's step 1 explicitly deletes `TestSetupSignalHandling_ReturnsTargets`
(the only test that called `SetupSignalHandling`), reasoning its
target-count assertion is redundant with `targets_test.go:146`. That's true
for the *target-count* assertion, but no other test now calls
`SetupSignalHandling` itself (confirmed via `rg -n "SetupSignalHandling"` —
only production caller is `internal/cli/main.go:25`, plus binary-subprocess
end-to-end tests which don't register in `go test -cover`'s instrumentation).
This drops `SetupSignalHandling`'s function coverage to 0%, failing the 80%
threshold gate. `SetupSignalHandling` is explicitly marked
`Deprecated: interim shim only — deleted by the #700 wiring task` in the
brief's own code — T2 deletes it entirely, which resolves this gate
automatically. I did not restore the old test (that would contradict the
brief's explicit "replace the whole file" instruction and reintroduce a
`chan os.Signal` reference this task is meant to eliminate) or add a
coverage-ignore directive (no such per-function suppression convention
exists in this repo, and it would mask a real, if temporary, gap). Flagging
per this task's own step-10 precedent ("if it flags X... do NOT suppress:
fold this commit together with T2's instead") — since T1 and T2 are separate
dispatches I cannot literally fold them, so I'm surfacing this instead:
**verify `check-coverage-for-fail` goes green for `internal/cli/signal.go`
once T2 deletes `SetupSignalHandling`.**

**#2 — `check-thin-api` fails structurally for all 3 new cmd/engram files
(9 declarations: `os_fs.go`, `os_signal.go`, `debuglog_sink.go`).** This is
architecturally significant, not a T1-specific bug. `check-thin-api` is a
targ-builtin gate (part of `check-full`'s dependency graph) that walks every
non-test, non-internal `.go` file in the repo and requires "thin" functions
(≤1 statement, or simple error handling) and no struct-with-fields types —
i.e., it enforces exactly the convention in the user's global CLAUDE.md
("Entry points... excluded from coverage — only re-exports and thin
wrappers, no testable logic"). It has no config/exclusion mechanism beyond
build tags, `_test.go` suffix, `examples/`, or `generated_`-prefixed /
"Code generated"-marked files — none of which honestly apply to real,
hand-written adapter code. I confirmed via a clean baseline run (before any
T1 file existed) that `check-thin-api` currently PASSES repo-wide — this is
a brand new conflict, and it is fundamental to the #700 migration's design:
the entire point of tasks like T1 (and later T10 atomic-write, T14
hugot/cache, T17 commander) is to relocate substantive I/O adapter logic
*into* cmd/engram — e.g. `osFS.WriteFileAtomic` (17 statements, ADR-0013
atomic-write dance), `flockLocker.Lock` (7 statements, flock+unlock), and
`syncWriter` (a struct with a field) are exactly the kind of code
`check-thin-api` was built to keep OUT of non-internal packages under the
*old* architecture (where cmd/engram was a 1-line `main()` calling into
`internal/cli.Main`). The constraints doc anticipated an analogous tension
for the *coverage* gate only ("Coverage config: cmd/engram may be excluded
from coverage as an entry point... that's a dev-tooling config question for
the enforcement task, not a reason to keep adapters in internal" — Embed-family
DESIGN FLAG 10) but does not mention `check-thin-api` anywhere. I did not
attempt a fix: relocating logic back into `internal/` to satisfy the
"thinness" checker would directly redesign/reverse #700's core goal (forbidden
per the constraints doc); marking these files as build-tagged or
"generated" to dodge the checker would be dishonest and would also blunt
other linters' generated-file leniency on real production code. **This needs
an explicit orchestrator/Joe decision, analogous to the already-anticipated
coverage carve-out**: either (a) `check-thin-api` gets a documented
exclusion for `cmd/engram`'s adapter files (mirroring the coverage
precedent, likely resolved at T-final-1 alongside "Coverage stance for
cmd/engram revisited deliberately"), or (b) some other resolution. Every
later task that adds non-trivial cmd/engram code (T10, T14, T17 at minimum)
will hit this identical wall — recommend resolving the policy once, early,
rather than re-discovering it in each of those tasks.

Neither concern blocks T2 (which doesn't depend on `check-full` being fully
green, and doesn't touch these gates) — both are process/policy findings for
the orchestrator to route, not defects in this task's deliverable.
