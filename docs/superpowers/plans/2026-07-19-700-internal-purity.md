# #700 Internal Purity — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Zero I/O-capable imports in `internal/` non-test code: raw impure capabilities enter as `cli.Primitives` func values from `cmd/engram` (whose package main stays declaration-free and single-statement — `targ check-thin-api` enforced), ALL adapter composition/lifecycle logic lives in `internal/`, env/clock/FS capabilities thread from the edge via `cli.NewDeps`/`cli.Deps`, and depguard default-deny + forbidigo enforcement lands with no in-boundary carve-outs (issue #700).

**Architecture:** `cmd/engram` stays radically thin — package main is a single-statement `main()` over a `cli.Primitives` literal of raw capability references and sanctioned closures: single-call signature-erasers plus exactly two enumerated stdlib-equivalent primitive closures (S-1 `WriteFileExcl`, C-1 `RunCommand` — see the revised composition doctrine's closure rule) (targ's `check-thin-api` builtin enforces this and is authoritative); `internal/cli` composes ALL production adapters from those primitives via `cli.NewDeps` (EdgeFS wrapping + ADR-0013 dances, flock lifecycle, debug sink, signal force-exit) behind `cli.Targets(deps cli.Deps)`, unit-tested with fake primitives and integration-tested with real os/syscall funcs in internal `_test` files. Enforcement is config-only (depguard allow-list default-deny + forbidigo call-level rules; both exclude `_test.go`), landing as the FINAL task so every prior task keeps `targ check-full` AND `targ check-thin-api` green. See "Revised composition doctrine" under Design flags (user correction, 2026-07-19).

**Tech Stack:** Go, targ (build/check), golangci-lint v2 (depguard, forbidigo), imptest + rapid + gomega, git worktree `worktree-700-internal-purity`.

## Global Constraints

- NEVER run `go test`/`go vet` directly — use `targ test` / `targ check-full`; binary install is `go install ./cmd/engram` (no targ build target).
- Every new/changed test: `t.Parallel()` in parent AND every subtest; no shared mutable state between subtests; exported test functions before private constants/types (reorder-decls); gomega assertions with `if err != nil { return }` after error expectations (nilaway); wrap errors with `%w` context; line length < 120; named constants over magic numbers; `make([]T, 0, cap)` when size known.
- `fs.FileMode`/`fs.FileInfo`/`fs.DirEntry` come from `io/fs`, never `os`, in internal code.
- ADR-0013 (vault flock + atomic rename) semantics are safety-critical: lock acquisition stays at `Run*` entry points; the concurrent-writers regression test must survive every task; never weaken atomic-rename.
- Implement the validated recipe from issue #700 — do not redesign mid-task (vault note 238). Any forced departure is a DESIGN FLAG escalated to the orchestrator, not an improvisation.
- `internal/update/update.go` diff stays minimal (concurrent Pi worktrees touch it).
- Each task ends with `targ check-full` green (or `targ test` where the task says so) AND `targ check-thin-api` PASS, and a commit (mid-task RED steps are expected to fail by design; only the task-final gate must be green) with an `AI-Used: [claude]` trailer. A thin-api finding is never suppressed or worked around — escalate it (doctrine flag SIG-1).

## Design flags resolved at plan time

**Revised composition doctrine (user correction, 2026-07-19 — BINDING; wins over any conflicting flag or task-body text below):**

Joe's rule (vault note 303): ALL actual logic lives in `internal/`; cmd wiring is EXPLICITLY very
thin, and targ's `check-thin-api` builtin gate — which walks every non-internal, non-test `.go`
file — is the AUTHORITATIVE enforcement. The T1 that landed at de484526 implemented real adapter
logic in `cmd/engram` and FAILED it with 9 non-thin declarations; that layout is REJECTED and
reworked by Task T1-rework. Checker semantics (verified from targ's `dev/targets.go`
`checkFuncThinness`/`checkTypeSpecThinness`/`checkValueSpecThinness`): THIN = a func whose body is
one statement calling another package (or returning an external call/selector/basic-literal/ident,
or the 2–3-statement `x, err := pkg.F(); if err != nil {...}; return` wrapper), interface types,
type aliases, EMPTY structs, consts with literal/re-export values. NOT thin: any func — INCLUDING
`func main()` — with two or more non-wrapper statements; structs with fields; vars whose value is a
composite literal (so no `var _ cli.EdgeFS = x{}` conformance vars in cmd — put them in internal).
Closures (`ast.FuncLit`) inside expressions are NOT walked by the checker — a CHECKER limitation,
not a license. The doctrine's closure rule: a cmd Primitives closure is sanctioned ONLY as a
**stdlib-equivalent primitive** — linear open/configure/do/close plumbing with mechanical error
propagation and nothing else (no policy, no retry, no branching decisions beyond
`if err != nil { return err }`); the yardstick is os.WriteFile's own multi-syscall body. Every such
closure gets (a) an enumerated doctrine flag, (b) a signature-extension guard — when behavior must
change (timeout, env, output policy, retry), extend the primitive's SIGNATURE and put the logic
internal; NEVER grow the closure body — and (c) a named behavior-mirror integration test over the
real primitive. The enumerated survivor list (exactly two: S-1 `WriteFileExcl`, C-1 `RunCommand`)
is the human-enforced complement to the checker gap; single-call signature-erasure closures and
the SIG-1 signal starter remain sanctioned under their own recorded flags. Any NEW closure beyond
the enumerated set is a review DEFECT — orchestration hidden in a closure violates the DESIGN even
where the checker cannot see it. The corrected shape:

1. **`cli.Primitives`** (internal/cli/primitives.go, landed by T1-rework) carries raw impure
   capabilities as func fields; cmd populates it with direct references (`os.ReadFile`,
   `filepath.WalkDir`, `time.Now`), single-call closures where a signature must be erased, or an
   enumerated stdlib-equivalent survivor closure (S-1/C-1 — closure rule above). The
   canonical struct — downstream briefs consume these field names verbatim:

```go
type Primitives struct {
	// Filesystem (direct os/filepath references).
	ReadFile  func(path string) ([]byte, error)                      // os.ReadFile
	WriteFile func(path string, data []byte, perm fs.FileMode) error // os.WriteFile
	MkdirAll  func(path string, perm fs.FileMode) error              // os.MkdirAll
	MkdirTemp func(dir, pattern string) (string, error)              // os.MkdirTemp
	Stat      func(path string) (fs.FileInfo, error)                 // os.Stat
	ReadDir   func(path string) ([]fs.DirEntry, error)               // os.ReadDir
	Remove    func(path string) error                                // os.Remove
	RemoveAll func(path string) error                                // os.RemoveAll
	Rename    func(oldPath, newPath string) error                    // os.Rename
	WalkDir   func(root string, fn fs.WalkDirFunc) error             // filepath.WalkDir
	Chmod     func(path string, mode fs.FileMode) error              // os.Chmod

	// Exclusive create (doctrine survivor S-1 — a stdlib-equivalent
	// primitive closure: os.WriteFile's own body with O_CREATE|O_EXCL;
	// behavior changes extend this SIGNATURE, never the cmd body).
	WriteFileExcl func(path string, data []byte, perm fs.FileMode) error

	// Process, env, clock (direct references).
	Getenv      func(key string) string // os.Getenv
	Now         func() time.Time        // time.Now
	Getwd       func() (string, error)  // os.Getwd
	UserHomeDir func() (string, error)  // os.UserHomeDir

	// Advisory file locking (single-syscall closures; lifecycle internal).
	OpenLockFile   func(path string, perm fs.FileMode) (uintptr, error) // syscall.Open O_CREAT|O_RDWR
	FlockExclusive func(fd uintptr) error                               // syscall.Flock LOCK_EX
	FlockUnlock    func(fd uintptr) error                               // syscall.Flock LOCK_UN
	CloseFD        func(fd uintptr) error                               // syscall.Close

	// Debug sink (single-call closure; empty-path branch + sync policy internal).
	OpenDebugFile func(path string, perm fs.FileMode) (WriteSyncer, error) // os.OpenFile O_APPEND|O_CREATE|O_WRONLY

	// Signal (starter closure; forwarding via internal ForwardAsPulses).
	StartSignalPulses func(pulses chan<- struct{}, buffer int)
}
```

2. **`cli.NewDeps(prims Primitives, stdout, stderr io.Writer, exit func(int)) Deps`** composes
   EVERYTHING internally: `primFS` (internal/cli/edgefs.go — the EdgeFS impl with all `%w`
   wrapping + the ADR-0013 atomic-write dance; primitives return RAW os errors, so
   `errors.Is(err, fs.ErrNotExist)`/`fs.ErrExist` chains survive trivially and the wrap happens
   exactly once, internally), `primLocker` (internal/cli/locker.go — open→flock→unlock-then-close
   lifecycle + unlock-error semantics), the debug sink (internal/cli/debugsink.go —
   `openDebugSink` empty-path/failed-open→nil branch + per-write-Sync `syncWriter`; `type
   WriteSyncer interface { io.Writer; Sync() error }` is exported from internal/cli and *os.File
   satisfies it), the force-exit signal watcher (`startForceExit`), and (from T2) `Deps.Embed`.
   All of it is unit-tested with fake primitives and integration-tested with REAL os/syscall
   funcs in internal `_test` files — sanctioned: the T-final-1 purity lint excludes `!$test`, and
   the ADR-0013 regression tests ride there.
3. **Signal:** generic `ForwardAsPulses[T any](in <-chan T, out chan<- struct{})`
   (internal/cli/signal.go) erases os.Signal without importing os; tests drive it with `chan int`.
   `ForceExitOnRepeatedSignal` stays as-is.
4. **cmd/engram/main.go final shape (T2):** package main is DECLARATION-FREE — `main()` is ONE
   statement, `targ.Main(cli.Targets(cli.NewDeps(cli.Primitives{...}, os.Stdout, os.Stderr,
   os.Exit))...)`, with every capability inline in the literal. Signal registration happens inside
   NewDeps during argument evaluation — before targ.Main runs. The issue-AC "targ.Main called from
   cmd" holds.
5. **Per-task gate:** every task's final gate includes `targ check-thin-api` PASS alongside
   `targ check-full`. If a sanctioned closure or the literal itself ever trips the checker,
   ESCALATE the exact finding to the orchestrator — do not suppress, do not restructure ad hoc.

Recorded design flags (choices within the correction's stated latitude):

- **SIG-1 (escalated + resolved):** the correction's sketch put the signal statements directly in
  `main()`; that literally FAILS the authoritative gate — `checkFuncThinness` has NO main()
  exemption, so a ~6-statement main() is a "has 6 statements" violation (finding derived from the
  checker source; the landed main passed only because it was a single external call). Resolution
  preserving every other element: `Primitives.StartSignalPulses` carries a cmd closure whose body
  is three single-call lines (`make` sigCh → `signal.Notify(SIGINT, SIGTERM)` → `go
  cli.ForwardAsPulses(sigCh, pulses)`); internal `startForceExit` (called by NewDeps, nil-skipped
  for test Deps) owns the pulse channel, buffer size, and ForceExitOnRepeatedSignal registration.
  If Joe prefers a different resolution (e.g. teaching targ to exempt `func main`), the swap is
  confined to one Primitives field + startForceExit.
- **P-1:** permissions thread through primitives as `fs.FileMode` args (`OpenLockFile(path,
  perm)`, `OpenDebugFile(path, perm)`) so perm POLICY (0o600 lock, 0o644 debug log) stays internal
  and cmd declares zero constants.
- **P-2:** flock is three semantic single-syscall primitives (FlockExclusive/FlockUnlock/CloseFD
  over a `uintptr` fd) rather than `Flock(fd, how)` + re-exported LOCK_* consts — no const
  duplication, every closure single-call, unlock-error semantics internal.
- **P-3:** the lock-open primitive wraps `syscall.Open`, NOT `os.OpenFile(...).Fd()` — dropping
  the *os.File after extracting its fd lets the runtime finalizer close the fd and silently
  release the flock mid-hold. syscall.Open has no finalizer.
- **P-4:** the unique-temp-name policy is INTERNAL. The atomic dance derives candidate temp names
  from the target path + `Now` nanos + an attempt counter, creates each exclusively via the
  `WriteFileExcl` primitive (S-1) with the data at the target perm, retries a fresh candidate on
  `fs.ErrExist` (bounded: `const maxTempAttempts = 10`; exceeded → wrapped error), then explicitly
  chmods the created temp to the exact target perm via the restored `Chmod` primitive —
  umask-independent, parity with the pre-#700 dance (chmod runs AFTER the exclusive create/write
  and BEFORE rename; reordering it earlier is a defect) — then renames onto the target; any
  failure after creation (including a chmod failure) removes the temp. Same-directory rename
  atomicity — the ADR-0013 primitive — is unchanged. Umask-filtered atomic writes (matching plain
  WriteFile) were considered and deferred — behavior preservation governs this refactor;
  normalizing perms policy is a candidate follow-up issue.
- **S-1 (enumerated survivor — exclusive create):**
  `Primitives.WriteFileExcl func(path string, data []byte, perm fs.FileMode) error` — cmd's
  literal value is ONE stdlib-equivalent closure: `os.OpenFile(path,
  os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm)` + write + close with os.WriteFile's exact error
  plumbing, raw error returned (the `*fs.PathError` keeps `errors.Is(err, fs.ErrExist)` alive).
  Lands in T1-rework's BASE Primitives: it backs both the atomic dance's unique-temp creation
  (P-4) and `EdgeFS.WriteFileExcl` (X-1). Signature-extension guard: behavior changes (perm
  policy, fsync, retry) extend the SIGNATURE with the logic internal — never grow this body.
  Behavior-mirror integration test: `TestRealEdgeFS_WriteFileExclRefusesExistingFile`
  (internal/cli/primitives_integration_test.go, lands T3; T1-rework's real-primitive dance suite
  exercises the same closure).
- **D-1 (amends R6 and the wiring-core cmd-wiring sequencing flag):** `Deps.Embed` is wired INSIDE
  NewDeps at T2 (`embed.NewLazyEmbedder(CacheDirFromHome(homeOrEmpty(deps), embed.BundledModelID,
  prims.Getenv))`, guarded on non-nil Getenv); cmd carries no embed wiring line. T14's 3-arg
  constructor change edits that internal line and adds backend/cache capability fields to
  Primitives whose cmd-side values are single-call method bodies on EMPTY structs (empty structs
  and single-call methods pass the checker; any stateful session/cache/temp-dir orchestration
  stays internal, parameterized over primitives — a cmd struct WITH fields is a gate failure).
- **C-1 (T16/T17; enumerated survivor — command run):** the commander uses the injected-sentinel
  form — Primitives gains a raw `RunCommand` capability plus a `NotFoundErr error` field (cmd
  wires `exec.ErrNotFound`); the run-and-collect logic + `errors.Is(err, notFoundErr)` →
  `update.ErrCommandNotFound` translation live internal. The `RunCommand` closure is the second
  enumerated survivor: `*exec.Cmd` cannot cross the boundary, so the closure is construction +
  field assignments + ONE invocation (`exec.CommandContext` → `Dir`/`Stdout`/`Stderr` assignments
  → `return cmd.Run()`), zero branching — semantically one operation. Signature-extension guard:
  behavior changes (timeout, env, output policy, retry) extend the SIGNATURE with the logic
  internal — never grow this body. Behavior-mirror integration test:
  `TestCommanderIntegration_RunsInDir` (+ siblings, internal/cli/commander_integration_test.go,
  T17). T17's brief reviser owns the exact field shapes.
- **X-1 (T3 forward-guidance; RESOLVED into survivor S-1):** the EdgeFS `WriteFileExcl` method
  (learn-family flag) is the single internal `%w` wrap over the BASE `WriteFileExcl` primitive
  (S-1, landed in T1-rework — the same exclusive create the atomic dance uses); the composed
  error must keep satisfying `errors.Is(err, fs.ErrExist)`. T3 consumes and verifies — it adds NO
  new primitive and NO cmd line.
- **DRIFT:** cli_test's `realPrimitives()` helper mirrors cmd/engram/main.go's literal and can
  drift from it; cli_test.go's end-to-end binary tests are the guard on the production literal.

Supersession map (downstream task briefs — T3 through T-final-2 — read their bodies through this):

- Wiring-core flag "the final cmd/engram/main.go wires `Commander: &osCommander{}` ... `Embed:
  embed.NewLazyEmbedder(...)`" → SUPERSEDED by D-1/C-1 (no cmd wiring lines; internal composition).
- Wiring-core flag "T1's cmd/engram/os_signal.go needs the pure ... signature" → AMENDED: no
  cmd/engram/os_signal.go exists post-rework; the SIGUSR2 real-delivery integration test lives in
  internal/cli/signal_integration_test.go.
- Wiring-core flag "lint risk in T1 — registerForceExit/openDebugSink unused until T2" →
  SUPERSEDED: those symbols are internal and consumed by NewDeps; the surviving T1-rework residual
  is the `SetupSignalHandling` 0% coverage gap, resolved by T2's deletion.
- Wiring-core flag "EdgeFS adapter methods wrap errors with %w" and every learn/query/maintenance/
  update-family obligation on "cmd/engram osFS must wrap with %w / preserve errors.Is chains /
  keep RemoveAll nil-on-absent" → obligations TRANSFER VERBATIM to internal/cli/edgefs.go
  (`primFS`); the raw primitives return unwrapped os errors, preserving sentinel chains by
  construction. Learn-family cmd-integration-test obligations (flock-on-unwritable-path,
  WriteFileExcl→fs.ErrExist) land as internal `_test` files instead.
- Maintenance-family flag "sequencing — cmd/engram/os_fs.go's osFS has no production caller" and
  M1's "create-or-append cmd/engram/os_fs.go" → OBSOLETE: there is no cmd/engram/os_fs.go;
  T10/M1's `WriteFileAtomic` relocation target is internal/cli/edgefs.go, ALREADY landed by
  T1-rework — T10 reduces to migrating its internal consumers onto `deps.FS.WriteFileAtomic`
  (T13's grep gate is unchanged).
- Embed-family flags 1/8 and T14's cmd/engram/hugot.go: the Backend/PipelineHandle interface
  erasure stands, but cmd's implementations must be single-call method bodies on empty structs
  wired as Primitives/NewDeps inputs (D-1); session/cache orchestration stays internal.
- Update-family "coordination with os_fs cluster" flag → binds internal/cli/edgefs.go; the nine
  deleted TestOsUpdateFS_* round-trips hand coverage to internal/cli/primitives_integration_test.go.
- Update-family flags "only osCommander physically moves to cmd/engram/os_update.go" and "the
  `Commander: &osCommander{},` field lands in cmd/engram's Deps literal ... requires
  cmd/engram/os_fs.go to exist" → SUPERSEDED by doctrine flag C-1: no cmd commander type, no
  os_update.go, no cmd Deps literal — cmd contributes the `RunCommand` survivor closure (C-1) +
  `NotFoundErr` value in the Primitives literal; internal `primCommander` owns run-and-collect +
  the sentinel translation (T17).
- Ingest-family flag 4's "production flockLocker mutual-exclusion coverage moves to cmd/engram;
  confirm the adapters cluster's plan includes a flockLocker exclusivity + fresh-dir integration
  test" → the coverage lives INTERNAL: `TestRealFlockLocker_SecondLockWaitsForUnlock`
  (primitives_integration_test.go) + the `TestPrimLocker_*` fake suite (locker_test.go), both
  T1-rework; fresh-dir mkdir-before-lock rides T8's manifest-lock composition test + T9's
  real-binary check. No cmd-side lock test exists.
- T-final-1 Step 5.5 (coverage stance): cmd/engram now holds ONLY the declaration-free main.go —
  an entry point, coverage-excluded per the user's global rules; the former cmd adapter tests live
  as internal `_test` files and count toward internal coverage. The deliberate revisit reduces to
  confirming the entry-point exclusion still applies.
- Any remaining task-body reference to `cmd/engram/os_fs.go`, `os_signal.go`,
  `debuglog_sink.go`, `osFS`, `flockLocker` (cmd), `syncWriter` (cmd), `registerForceExit`,
  `forwardAsPulses` (cmd), or `newOsDeps` → read as the internal equivalents: `primFS`,
  `primLocker`, `syncWriter` (internal/cli/debugsink.go), `startForceExit` +
  `Primitives.StartSignalPulses`, `cli.ForwardAsPulses`, and `cli.NewDeps`.

**Wiring-core (foundation — binding for all tasks):**

- DESIGN FLAG: The embed interface is `embed.Embedder` (internal/embed/embedder.go:54: `Embed(ctx, text) ([]float32, error)`, `ModelID() string`, `Dims() int`). There is no `embed.Backend`. Per the spec's own fallback clause, `Deps.Embed` is typed `embed.Embedder`.
- DESIGN FLAG: `resolveVault` (internal/cli/learn.go:443) and `ResolveChunksDir` (internal/cli/ingest.go:63) are ALREADY pure — `home` and `getenv` are injected parameters. Only their call sites in targets.go change (`os.Getenv` → `deps.Getenv`, `homeOrEmpty()` → `homeOrEmpty(deps)`). No edits to learn.go/ingest.go from this cluster.
- DESIGN FLAG: debuglog's per-line `file.Sync()` (debuglog.go:67) is load-bearing for `tail -F` liveness. `io.Writer` has no Sync, so the cmd sink must wrap `*os.File` in a per-write-syncing writer (`syncWriter` below), or behavior regresses.
- DESIGN FLAG: current `debuglog.New("")` returns a NON-nil `&Logger{}` no-op and callers rely on the `l.file == nil` guard. The new API returns a nil `*Logger` for a nil writer; a zero-value `&Logger{}` becomes invalid (nil `now` func would panic). All construction must go through `New` — the only production caller is internal/cli/main.go:23, which this cluster deletes.
- DESIGN FLAG (sequencing): targets.go closures call `newOsXxxDeps()` constructors owned by other clusters. T2 keeps those calls (they compile unchanged) and swaps only `os.Getenv`/`os.UserHomeDir`/`stdout` for deps fields; the `newXxxDeps(deps)` swap is one line per closure, owned by each command cluster (exhaustive table in T2 step 3). Similarly `runLearnFromFeedbackArgs`/`runLearnFromFactArgs` (learn.go:497,520) and `runUpdate` (update.go:276) construct their own os deps internally — their deps-param conversion belongs to the learn/update clusters.
- DESIGN FLAG (sequencing): the final cmd/engram/main.go wires `Commander: &osCommander{}` (adapter destined for cmd/engram/os_update.go, update cluster) and `Embed: embed.NewLazyEmbedder(...)` (hugot construction moves to cmd/engram/hugot.go, embed cluster). Both wiring lines compile at T2 time ONLY if sequenced as noted in T2 step 5: `embed.NewLazyEmbedder`/`embed.BundledModelID` exist today (internal/embed/hugot.go:149,18) so `Embed` wires now; `osCommander` does not exist in package main yet, so T2 omits the `Commander:` field line (nothing consumes it until runUpdate converts) and the update cluster adds it.
- DESIGN FLAG: `cli.Targets` has 8 test call sites, not 4 — targets_test.go:153,289,307,439 AND vocab_commands_test.go:3580,3606,3630,3655 (all `package cli_test`, so one shared helper fixes all).
- DESIGN FLAG: T1's cmd/engram/os_signal.go needs the pure `ForceExitOnRepeatedSignal(<-chan struct{}, func(int))` signature, so the signal.go signature conversion happens in T1 (with an interim adapter inside `SetupSignalHandling`, which T2 then deletes). Also: real-signal integration testing uses SIGUSR2 self-delivery; pending-signal coalescing can swallow a rapid second delivery, so the test paces re-sends until exit fires (over-delivery still means exit — no false pass).
- DESIGN FLAG: `lint` risk in T1 — `registerForceExit`/`openDebugSink` are unused by main.go until T2. Their `_test.go` files in the same package use them, which satisfies golangci's `unused` (tests included by default). If `targ check-full` still flags them, fold T1 and T2 into one commit rather than suppressing.
- DESIGN FLAG: EdgeFS adapter methods wrap errors with `%w`, which PRESERVES `errors.Is(err, fs.ErrNotExist)` chains that internal callers rely on (e.g. embed.go readerFS:181). Safe to wrap everywhere per house rules.

**Learn-family:**

- DESIGN FLAG: EdgeFS has no exclusive-create primitive. `osLearnFS.WriteNew` (cli.go:115, `O_CREATE|O_EXCL`, the documented K1 backstop) and `osLearnFS.WriteFileIfMissing` (cli.go:90, `O_EXCL` + swallow-`ErrExist`, runs BEFORE lock acquisition — two concurrent first-learns race on vault bootstrap) cannot be composed from the spec's EdgeFS: `WriteFile` truncates, and `Stat`-then-`WriteFile` is TOCTOU. This draft adds ONE method to EdgeFS: `WriteFileExcl(path string, data []byte, perm fs.FileMode) error` (must satisfy `errors.Is(err, fs.ErrExist)` when path exists). Fallback if rejected: `Stat`+`WriteFileAtomic` — preserves under-lock behavior but drops the O_EXCL backstop and reopens the pre-lock bootstrap race. Not recommended (ADR-0013 safety-critical).
- DESIGN FLAG: `LearnDeps.Getenv` and `LearnQADeps.Getenv` are dead fields — assigned in the constructors (learn.go:353, qa.go:267) but never read (vault resolution happens in targets.go via `resolveVault`). Kept and wired from `d.Getenv` for surface stability; candidate for a follow-up removal.
- DESIGN FLAG (ordering): `osLearnFS.Lock` has four consumers OUTSIDE this cluster (activate.go:123, amend.go:345, resituate.go:163, vocab_commands.go:1211); `flockPath` two more via `osManifestLock` (ingest.go:491, prune.go:107); `logWarningToStderrf` (defined in learn.go:332) is consumed by activate/amend/resituate/vocab/qa constructors; `osFileReader` (cli.go:27) is consumed by ingest.go:488. Task L1 therefore keeps `osLearnFS` (Lock-only), `flockPath`, `osManifestLock`, and relocates `logWarningToStderrf` into cli.go; Task L2 (the cli.go purge) MUST be sequenced after the activate/amend/resituate/vocab/ingest/prune constructor migrations.
- DESIGN FLAG (coordination, os_fs.go owner): cmd/engram `osFS.ReadDir`/`Stat`/`WriteFileExcl` must return errors satisfying `errors.Is(err, fs.ErrNotExist)` / `fs.ErrExist` (wrap with `%w` or return the raw `*fs.PathError`) — the learn compositions replace `os.IsNotExist` with `errors.Is`. Also: cmd integration tests must cover flock-on-unwritable-path error (replaces `TestOsLearnFS_Lock_BadVaultReturnsError`, deleted in L2) and `WriteFileExcl` → `fs.ErrExist` on existing file.
- DESIGN FLAG (coordination, ingest cluster; superseded by R7/R10 — those win): `TestManifest_ConcurrentWritersDoNotLoseEntries` (ingest_test.go:319) consumes `cli.ExportFlockPath` today. Per R7, T8 deletes `ExportFlockPath` and repoints the test's two Lock closures to its test-local real-flock `testFlocker{}` (still a real syscall flock; test files are exempt from enforcement) — nobody re-implements it, and L2/T4 only grep-verifies zero hits. `TestOsManifestLock_MkdirError` (testhelpers_test.go) dies with `osManifestLock` in T9 per R10 — the mkdir-before-lock behavior is re-covered by the ingest cluster's replacement composition.
- DESIGN FLAG (coordination, targets/foundation cluster): this draft assumes the foundation task lands first (`internal/cli/deps.go` with `Deps`/`EdgeFS`/`FileLocker`, `cmd/engram` `osFS`+`flockLocker`, `Targets(d Deps)` threading `d` into `learnUpdateTargets`, and `executeForTest` in targets_test.go re-wired to a real-FS test Deps). The shared test doubles `osEdgeFSForTest`/`flockLockerForTest` created in L1 (testhelpers_test.go) are intended for reuse by targets_test and the other family clusters — L1's separate `realFSDepsForTest` builder was later collapsed into `newTestDeps` (Gate B #700 T3 review finding 2; targets_test.go's single builder is the sole real-Deps test constructor). Shared compose helpers (`statDirFromFS`, `listMDFromFS`, `logWarningTo`, `vaultLockFromLocker`, `writeAtomicFromFS`) are declared ONCE in `internal/cli/deps_compose.go` by this task — amend/resituate/vocab/activate clusters must consume, not re-declare.
- DESIGN FLAG: cli_test.go's end-to-end tests (`TestEngramLearn_Fact_EndToEnd` etc.) build and run the real binary — they gate the cmd/engram wiring automatically and need no changes.

**Query-family:**

- DESIGN FLAG: `osVaultFS` (vault_fs.go:14) is consumed by SEVEN non-cluster files: amend.go:342, learn.go:349, embed.go:156, qa.go:262, resituate.go:160, vocab_commands.go:1213/1215/1237/1238, plus test shim `ExportNewOsVaultFS` (export_test.go:573) used by vocab_trigger_test.go:251,441 and vocab_commands_test.go (10 sites). Deleting it inside this cluster's window breaks their compile. Split: Task Q1 adds the pure `vaultFS` and migrates this cluster's three consumers; Task Q3 (purge) deletes `osVaultFS` + its `os` import and MUST be sequenced after those clusters migrate to `newVaultFS(d.FS)`. Until Q3, vault_fs.go temporarily retains its `os` import (grep-gated in Q3).
- DESIGN FLAG (superseded by R3's staged cutover — R3 wins): `listJSONLIndexes` (query_chunks.go:138) has four consumers: query.go:1295, amend.go:365, prune.go:115, show_chunk.go:72. R4 runs T6 BEFORE the prune (T9) and amend (T12) conversions, so a single-commit four-site flip is impossible (those files have no `d Deps` in scope at T6 time). Per R3: T6 lands the curried canonical lister, renames the legacy os-backed one to `osListJSONLIndexes` (mechanical rename at amend.go:365/prune.go:115), and flips only the call sites in files it converts itself (query.go, query_chunks.go, show_chunk.go — T6 owns show-chunk); T9/T12 flip prune/amend when they convert; T12 (last consumer) deletes the legacy lister grep-gated.
- DESIGN FLAG: the `os.IsNotExist` → `errors.Is(err, fs.ErrNotExist)` change at query_chunks.go:141 is load-bearing, not cosmetic: `os.IsNotExist` does NOT unwrap `%w`-wrapped errors, and EdgeFS implementations wrap. The cmd/engram `osFS` impl must wrap with `%w` (contract: `errors.Is(err, fs.ErrNotExist)` survives the adapter); Q1/Q2 add tests proving the internal side unwraps.
- DESIGN FLAG: `newQueryDeps` needs the LogWarning hook. Current `logWarningToStderrf` (learn.go:332) writes to `os.Stderr` — learn cluster's file. This plan consumes `logWarningTo(w io.Writer) func(format string, args ...any)`; exact definition included in Q2 step 3 with instructions to add it to deps.go ONLY if the learn cluster has not already landed it (coordinate — one definition, two consumers).
- DESIGN FLAG: preconditions from other clusters: (a) foundation task must have landed `internal/cli/deps.go` (`Deps`, `EdgeFS` — neither exists yet, verified by grep); (b) targets-cluster task must have landed `Targets(d Deps)` with `d` threaded into `ingestQueryTargets` (targets.go:144) — all five constructor renames edit call sites there; (c) `executeForTest` (targets_test.go:434-448) migration is the targets cluster's; `TestTargets_CountEmptyVault` (count_test.go:562) rides on it unchanged.
- DESIGN FLAG: `ExportNewOsChunkQueryDeps` is called from ingest_integration_test.go:100,204 (ingest cluster's test file) — Q2 updates those two lines in the same commit as the shim rename.
- DESIGN FLAG: query_nominations.go has ZERO os/time references (the grep hit at line 136 is the word "time" in a comment). Nothing to migrate in that file.

**Ingest-family:**

- DESIGN FLAG: the manifest lock is not a bare flock. `osManifestLock` (internal/cli/cli.go:227) does `MkdirAll(chunksDir, 0o700)` BEFORE flocking `chunksDir/.manifest.lock` — a shipped regression fix (prune on a fresh dir errored without it). `FileLocker.Lock(path)` alone cannot express this. The MkdirAll must stay in internal composition (`manifestLockFrom` below, via `deps.FS.MkdirAll`). "Lock acquisition stays at Run* entry points via injected FileLocker" holds only through this composition, never by handing `FileLocker.Lock` directly to Run*.
- DESIGN FLAG: `FileLocker.Lock` returns `(func() error, error)` but `IngestDeps.Lock`/`PruneDeps.Lock` and every Run* call site use `func()` releases. The composition adapts by discarding the unlock error (`func() { _ = unlock() }`) — identical to today's `flockPath` release, which already swallows unlock/close errors.
- DESIGN FLAG: hidden I/O not named in the central task list — `walkSourcesExcluding` (ingest.go:654) calls `filepath.WalkDir` directly (real disk walk) with an `os.DirEntry` callback. Migrated below via `deps.FS.WalkDir` + `fs.DirEntry` (`sweepListerFrom`).
- DESIGN FLAG: `defaultSessionDir` (ingest.go:247-258) reads BOTH `ENGRAM_TRANSCRIPT_DIR` (os.Getenv) and `os.UserHomeDir` — the migration needs `deps.Getenv` AND `deps.UserHomeDir`, not Getenv alone.
- DESIGN FLAG (cross-cluster, coordination required):
  1. `listJSONLIndexes` (query_chunks.go:138, `os.ReadDir`) is shared by prune + query-chunks + show-chunk + query + amend. Superseded by R3's staged cutover (R3 wins): T9 declares NO lister helper — it consumes T6's canonical curried `listJSONLIndexes(d.FS)` (T6 runs earlier per R4 and has already renamed the legacy os-backed lister to `osListJSONLIndexes`, which T12 — amend, the last consumer — deletes grep-gated). The cmd `osFS.ReadDir` MUST wrap errors with `%w` so `errors.Is(err, fs.ErrNotExist)` survives (cold-start = empty index, not error).
  2. `flockPath` (cli.go:169, syscall) is also used by `osLearnFS.Lock` — my tasks delete `osManifestLock` (its consumers are exactly ingest+prune) but must NOT delete `flockPath`; the learn cluster owns that removal.
  3. `osFileReader` (cli.go:27) has exactly one production consumer: ingest.go:488. Task I1 deletes it + its adapter tests (adapters_test.go:14-39) + `ExportNewOsFileReader`. If the core cluster also drafted this deletion, dedupe.
  4. The ADR-0013 concurrency regression test (ingest_test.go:319) today injects the REAL flock via `cli.ExportFlockPath`. After migration it uses a test-local syscall flock (test files are exempt from enforcement) — this preserves the Run*-holds-lock-across-read-modify-write proof, but production `flockLocker` mutual-exclusion coverage moves to cmd/engram; confirm the adapters cluster's plan includes a flockLocker exclusivity + fresh-dir integration test, else production-lock coverage regresses.
  5. `realFS.write` test helper (ingest_test.go:899) uses `cli.ExportAtomicWriteFile`; Task I1 switches it to a test-local atomic write so the writesafe cluster can delete `atomicWriteFile` without breaking ingest tests.
  6. Both tasks assume the foundation task has landed: `internal/cli/deps.go` (`Deps`, `EdgeFS`, `FileLocker`) and `deps Deps` threaded into `ingestQueryTargets`. Verify `internal/cli/deps.go` exists before starting; the two targets.go line edits below name that dependency explicitly.
  7. cli_test test-harness naming: Task I1 creates `ingest_family_deps_test.go` holding `osTestFS`/`testFlocker`/`testDeps` (package cli_test is one namespace across files) — if another cluster drafts an equivalent harness, consolidate into one file.

**Maintenance-family:**

- DESIGN FLAG: `atomicWriteFile` has callers OUTSIDE this cluster: internal/cli/learn.go:371 (LearnDeps.WriteNote), internal/cli/cli.go:144 (osLearnFS.WriteSidecar), internal/cli/embed.go:164 (osEmbedFS.Write), internal/cli/qa.go:283 (QA deps). Deleting internal/cli/writesafe.go must be gated until those clusters migrate — split into Task M4 with an explicit grep gate.
- DESIGN FLAG: internal/cli/ingest_test.go:899 (`realFS.write`, part of the ADR-0013-adjacent concurrent-manifest regression infra) calls `cli.ExportAtomicWriteFile` today. It needs REAL temp+rename semantics (torn-read protection under the race). T8 (running earlier per R4) repoints it at its test-local `testAtomicWrite` (same real dance) in step 6; M4/T13 only verifies that repoint held before deleting writesafe.go. M2's `ExportNewTestOsDeps` still carries a real `WriteFileAtomic` for the wiring tests it serves.
- DESIGN FLAG: shared-helper collision risk. My constructors need four composition helpers other clusters also need: a `vaultgraph.VaultFS` adapter over EdgeFS (replaces osVaultFS), a `.luhmann.lock` adapter over FileLocker (replaces osLearnFS.Lock), a stderr warn-logger (replaces learn.go's `logWarningToStderrf`), and an injected `.jsonl` lister (replaces query_chunks.go's os-backed `listJSONLIndexes`, which reads via `os.ReadDir` at query_chunks.go:139). I draft them once in internal/cli/deps_compose.go; orchestrator must dedupe against the learn/query/embed cluster drafts and have those clusters consume these helpers.
- DESIGN FLAG: EdgeFS error contract — the canonical `vaultFS.ListMD` (T5's `newVaultFS`) and `listJSONLIndexes` (T6) rely on `errors.Is(err, fs.ErrNotExist)` for the missing-dir→empty contract (current code uses `os.IsNotExist` on the raw error). The cmd osFS `ReadDir`/`Stat` implementations MUST wrap with `%w` (never `%v`) so sentinel matching survives. Ditto the test EdgeFS.
- DESIGN FLAG: targets.go call sites (lines 108, 113, 173, 278, 282, 286, 290) wire my constructors but the surrounding `Targets(deps Deps)` threading is the wiring cluster's charge. M3 lists the exact call-expression diffs; the wiring cluster owns adding the `deps Deps` parameter to `amendResituateTargets`/`ingestQueryTargets`/`vocabTargets`.
- DESIGN FLAG: sequencing — cmd/engram/os_fs.go's `osFS` type has no production caller until the cmd wiring task lands `Deps{FS: osFS{}}` in main.go. If M1 merges before wiring, `targ check-full`'s unused-symbol lint may flag `osFS`. Order M1 after (or in the same merge window as) the cmd-wiring task's os_fs.go creation; M1 below is written create-or-append.
- DESIGN FLAG: cross-cluster test-file touches — learn_test.go:132 uses `ExportNewOsAmendDeps` (my constructor, learn cluster's file); os_adapters_test.go:150 tests `logWarningToStderrf` (learn cluster should delete it when adopting `logWarningTo`); targets_test.go:413 covers resituate wiring through `Targets()` (wiring cluster updates). One-line diffs for the first are included in M3.
- DESIGN FLAG: vault_init.go and vocab.go are ALREADY pure (verified: vault_init.go imports fmt/io\/fs/path\/filepath only; vocab.go imports slices/strings/yaml/embed only; no os, no time.Now). No migration needed — M3 carries verify-only steps.

**Embed-family (numbered — Task T14/T15 text cites these as "DESIGN FLAG n"):**

1. **`embed.Backend` does not exist today.** The existing exported interface in `internal/embed/embedder.go:54` is `Embedder` (`Embed(ctx, text) ([]float32, error)`, `ModelID() string`, `Dims() int`). Per the spec's own escape hatch, the Deps field must be `Embed embed.Embedder`. This plan *additionally exports* a new `embed.Backend` (rename of today's unexported `hugotBackend`, `internal/embed/hugot.go:227`) as the constructor seam the cmd hugot adapter implements. Both names end up existing with distinct roles: `Deps.Embed` is the constructed `Embedder`; `Backend` is what cmd implements to build it.
2. **`sharedEmbedder` is a cross-cluster singleton.** `internal/cli/embed.go:110` constructs it; it is consumed by 7 files OUTSIDE this cluster (`qa.go:275`, `query_chunks.go:193`, `vocab_commands.go:1228`, `ingest.go:513`, `learn.go:360`, `resituate.go:176`, `amend.go:358`). Deleting it breaks them; leaving it breaks purity (its construction reads `os.UserHomeDir`/`os.Getenv` and, post-migration, needs the hugot backend that no longer exists in internal). Resolution: a race-safe transitional **bridge** (atomic pointer + forwarder value, full code in Task A) wired from `Targets(deps)`; each command cluster later replaces `Embedder: sharedEmbedder` with `d.Embed`; the enforcement sweep deletes the bridge.
3. **`newOsEmbedDeps()` has two external consumers**: `targets.go:226,230` (embed group closures) and `query.go:1288` (`newOsQueryDeps`). Task B swaps the targets.go call sites and makes a minimal `newOsQueryDeps(d Deps)` signature change + its call site `targets.go:155`. This overlaps the query cluster's territory — coordinate; the query task's full `newQueryDeps(d)` rewrite subsumes this edit if it lands first (then Task B skips step 6).
4. **Dead production machinery found**: `unpackModelToTemp`, `tempFS`, `productionTempFS` (`internal/embed/hugot.go:314-452`) are referenced by NO production code — `extractToCache` replaced them. Plan **deletes** them plus `unpack_test.go`/`tempfs_test.go` instead of relocating (git is the fallback). Note: `nonEmptyTestFS` (go:embed) is declared in `unpack_test.go:65-66` but used by `cache_test.go` — its declaration moves into `cache_test.go`.
5. **`parity_test.go`** (build tag `parity`) imports hugot directly and reads `assets/model` from disk. It is a test file and is excluded from normal builds; the enforcement task must exempt `_test.go` files (or this tagged file) from depguard, otherwise relocate it to cmd/engram in that task — not handled here.
6. **Behavioral contract shift on `CacheFS.Rename`** (verified, load-bearing; REVISED per the composition doctrine): today `commitCache` sniffs `*os.LinkError`/ENOTEMPTY strings via `isExistErr` (`cache.go:196`). The internal contract becomes `errors.Is(err, fs.ErrExist)`; the classification does NOT move to cmd (multi-statement logic fails `check-thin-api`) — it lives in internal/embed's `renameIsExist` over the RAW error the `Rename` primitive returns (T14 flag E-3: `errors.Is` + string fallback, no os import). The fakes at `cache_test.go:54,107` currently return raw `&os.LinkError{...directory not empty...}` — they MUST be updated to the new contract or the race tests go red for the wrong reason (Task T14 step 8).
7. **`CacheDirFromHome` stays in `internal/cli/targets.go:56`** (pure, exported; REVISED per D-1): its production caller is `cli.NewDeps`'s internal Embed line (`homeOrEmpty(deps)` + `prims.Getenv`) — cmd never calls it and `cmd/engram/hugot.go` imports only `hugot` + `internal/embed`.
8. **Foundation dependency** (REVISED per D-1): both tasks assume T1-rework/T2 have landed — `Deps`/`EdgeFS`, `Targets(deps Deps) []any`, `cli.NewDeps` with the guarded 1-arg Embed line, and the declaration-free cmd main over `cli.Primitives`. Task T14 edits that internal NewDeps line to the 3-arg composition, adds `Primitives.EmbedRuntime`, adds the `EmbedRuntime: hugotRuntime{},` literal line in cmd, and inserts one `wireSharedEmbedder(deps.Embed)` statement in `Targets` — coordinate the merge. EdgeFS contract notes for foundation: `ReadFile`/`ReadDir` errors must satisfy `errors.Is(err, fs.ErrNotExist)` for missing paths (state classification and ListMD-missing-dir semantics depend on it).
9. **`internal/embed/embedder.go` and `internal/embed/state.go` are already pure** (imports: `context/errors/fmt` and `errors/io/fs`) — zero changes needed in this cluster's charge for them.
10. **Coverage config** (REVISED per the composition doctrine): cmd/engram is coverage-excluded as an entry point (user global rules); the orchestration coverage now lives in internal (runtime_test/cachefs_test/cachefs_integration_test count toward internal coverage). cmd's only test file is the sanctioned hugot wiring-smoke; if `targ check-full` flags cmd coverage, that's a dev-tooling config question for the enforcement task (T-final-1 step 5.5 confirms the entry-point exclusion).

**Update-family:**

- DESIGN FLAG: osUpdateFS/osUpdateEnv cannot literally "move to cmd/engram/os_update.go" and stay wired: the fixed cli.Deps carries no `update.Filesystem`/`update.Env` field (only `FS EdgeFS` + env func fields), and `cli.Targets(deps Deps)` is the sole channel into internal/cli. Moving them verbatim would strand them as production-dead code in cmd (hoarding). This draft absorbs them instead: only osCommander physically moves to cmd/engram/os_update.go (wired as `Deps.Commander`); osUpdateFS becomes a pure EdgeFS→update.Filesystem bridge in internal/cli (`updateFSFromEdge`, zero I/O — `fs.DirEntry`/`fs.FileInfo` structurally satisfy `update.DirEntry`/`update.FileInfo`, so `osDirEntry`/`osFileInfo` wrappers die too); osUpdateEnv becomes a pure Deps-func bridge (`updateEnvFromDeps`).
- DESIGN FLAG: spec cites exec.ErrNotFound at update.go:437/:545; actual worktree lines are 436 and 544, plus the doc comment at 541–542 names exec.ErrNotFound (updated in Task UF-1).
- DESIGN FLAG: two test files inject exec.ErrNotFound to simulate the commander — internal/update/runner_test.go:556 and internal/cli/invariants_u1_test.go:36. Both must switch to `update.ErrCommandNotFound` in the same commit as the sentinel cutover or the suite goes red (ErrGitNotFound/ErrGoNotFound classification tests).
- DESIGN FLAG (coordination with os_fs cluster): update's `isNotExist` and its planners tolerate missing dirs via `errors.Is(err, fs.ErrNotExist)`. The production EdgeFS impl (cmd/engram/os_fs.go) MUST preserve that chain on ReadFile/ReadDir/Stat (no chain-breaking wraps), and RemoveAll must keep os.RemoveAll's nil-on-absent semantics. The deleted TestOsUpdateFS_* round-trips (nine tests in internal/cli/update_test.go:275-441) hand their real-FS coverage to that cluster's cmd/engram/os_fs_test.go.
- DESIGN FLAG (coordination with wiring cluster): Task UF-2's targets.go call site assumes `learnUpdateTargets` receives `deps Deps` after the Targets(deps) migration; the `Commander: &osCommander{},` field lands in cmd/engram's Deps literal built by that cluster. Sequencing: UF-1 is independent and can land first; UF-2 requires deps.go (Deps+EdgeFS), Targets(deps) threading, and cmd/engram/os_fs.go to exist.
- DESIGN FLAG: export_test.go (package cli) does not currently import internal/update; UF-2's new export helper adds that import.
- DESIGN FLAG: this worktree's internal/update/update.go is clean; the uncommitted Pi-harness edit exists only in the main tree. The internal/update diff below is 4 hunks (sentinel var, two errors.Is swaps, one comment) + import line — minimal for concurrent Pi worktrees.

*Deduplicated: the learn/query/ingest/maintenance drafts each independently re-confirmed the `embed.Embedder`-not-`embed.Backend` finding (collapsed into the first wiring-core bullet and embed-family flag 1), and the maintenance draft restated the FileLocker unlock-error-discard decision (collapsed into the ingest-family FileLocker bullet).*

## Cross-task resolutions (BINDING — this table wins over any conflicting task-body text)

Parallel drafting produced overlapping symbol names and one ordering contradiction. These
resolutions are authoritative: where a task body's symbol name or file-mode (Create vs Modify)
conflicts with this section, THIS SECTION WINS. Every task dispatch must include this section
verbatim alongside the task text.

**R1 — `internal/cli/deps_compose.go` has ONE creator.** T3 (L1) creates it. T11 (M2) MODIFIES it
(its "Create" is downgraded): T11 consumes T3's helpers where they overlap and adds only the
genuinely new ones. Canonical helper names (losers in parentheses are NOT declared anywhere):
`vaultLockFromLocker` (not `vaultLuhmannLock`), `logWarningTo` (not `warnLoggerTo`). T11 still
adds `ExportNewTestOsDeps() Deps` (test-only) and any helper with no T3 equivalent.

**R2 — ONE `vaultgraph.VaultFS`-over-EdgeFS adapter.** T5's `vaultFS` via
`newVaultFS(fsys EdgeFS) *vaultFS` is canonical. T11 does NOT declare `edgeVaultFS`; T15 does NOT
declare `depsVaultFS`; both consume `newVaultFS(d.FS)`.

**R3 — ONE `.jsonl` index lister, staged cutover (single story; T6/T9/T12 bodies conform).**
T6's curried `listJSONLIndexes(fsys EdgeFS) func(dir string) ([]string, error)` is canonical.
T6 lands it and, in the same commit, RENAMES the legacy os-backed lister to `osListJSONLIndexes`
(query_chunks.go keeps its `"os"` import for it until T12); the two foreign references
(amend.go:365, prune.go:115) get that mechanical rename ONLY — no deps flip, because under R4
those files still sit inside `newOsAmendDeps()`/`newOsPruneDeps()` with no `d Deps` in scope.
Each consumer's own task then flips its line to `listJSONLIndexes(d.FS)` when it converts its
constructor: T6 itself for query.go/query_chunks.go/show_chunk.go (T6 owns the show-chunk
conversion), T9 for prune.go, T12 for amend.go. T12 — amend.go is the LAST consumer — deletes
`osListJSONLIndexes` grep-gated and drops query_chunks.go's `"os"` import with it. T9 consumes
`listJSONLIndexes(d.FS)` and does NOT declare `jsonlIndexListerFrom`; T11 does NOT declare
`jsonlIndexesLister`.

**R4 — EXECUTION ORDER (binding; document order is NOT execution order):**
T1 → T2 → T3 → T5 → T6 → T8 → T9 → T10 → T11 → T12 → T4 → T14 → T15 → T7 → T13 → T16 → T17 → T-final-1 → T-final-2.
("T1" here means Task T1-rework, which occupies T1's slot: the original T1 landed at de484526 and
is reworked in place per the composition doctrine.)
Rationale: T4 (purge cli.go adapters) requires T8/T9/T12 done (its own heading says so — it sits
mid-document only because drafts were assembled by family); T7 (purge osVaultFS) runs after T15
(embed.go is osVaultFS's last consumer); T13 (purge internal atomic write) also runs after T15
(embed.go:164's `osEmbedFS.Write` calls `atomicWriteFile` until T15 deletes it — T13's gate
cannot pass earlier); deletions are grep-gated so a premature run fails loud.

**R5 — `osFileReader` is deleted ONCE, by T8.** T4's corresponding step becomes a grep
verification (`rg -n "osFileReader" internal/` → zero hits), not a second deletion.

**R6 — `embed.NewLazyEmbedder` arity handoff (REVISED per the composition doctrine, flag D-1).**
T2 wires `Embed` INSIDE `cli.NewDeps` (internal/cli/primitives.go):
`embed.NewLazyEmbedder(CacheDirFromHome(homeOrEmpty(deps), embed.BundledModelID, prims.Getenv))`
— today's 1-arg form, guarded on non-nil Getenv; cmd/engram carries NO embed wiring line. T14
changes the constructor to its 3-arg form AND updates that internal NewDeps line in the same
commit, adding the backend/cache capabilities as Primitives fields whose cmd-side values are
single-call method bodies on empty structs; `cmd/engram/main.go`'s Primitives literal gains those
field lines in T14 (both files are in T14's file list — the executor must not skip either).

**R8 — FIXME(#700) marker lifecycle.** T2 deletes the marker's host file (`internal/cli/main.go`) for wiring reasons, so T2 RELOCATES the marker into `cmd/engram/main.go` as the comment block directly ABOVE `func main()` (exact text in T2's step 5b — comments are legal in the declaration-free package main; `check-thin-api` sees only declarations); it stays grep-able through the whole migration and is deleted ONLY by T-final-2 after the enforcement gate is verified green. Any state where `rg "FIXME\(#700\)"` returns zero hits before T-final-2 is a defect.

**R9 — depguard files-glob vs the issue AC's literal `internal/**`.** The issue AC says root-anchored `'internal/**'`; the plan-time prototype only confirmed `'**/internal/**'` (the root-anchored form was never exercised). T-final-1 resolves this empirically: Step 5's negative probe runs FIRST with `files = ['internal/**', '!$test']`; if the probe fires, keep the AC's literal form. If it does not fire, switch to the confirmed `'**/internal/**'`, record the probe output in the commit body, and post a one-line comment on issue #700 amending the AC wording to the verified form. No silent substitution in either direction.

**R7 — `ExportFlockPath`.** T8 deletes it from `export_test.go` AND, in the same commit, repoints
its only two consumers — the `TestManifest_ConcurrentWritersDoNotLoseEntries` Lock closures
(ingest_test.go:375-377 and 402-404) — to T8's own test-local real-flock `testFlocker{}` (T8 step 6).
NOBODY re-implements it: T4's former re-implementation step is dead work and is collapsed to a grep
verification (`rg -n "ExportFlockPath" internal/` → zero hits). No other task touches it.

**R10 — `osManifestLock` has ONE deleter: T9.** T9 (I2) deletes all three artifacts: `osManifestLock`
from cli.go (its step 3), `ExportOsManifestLock` from export_test.go (its step 5), and
`TestOsManifestLock_MkdirError` from testhelpers_test.go (its step 6). T4 (running later per R4)
does NOT re-delete any of them — its corresponding Files-list entries and steps are grep
verifications (`rg -n "osManifestLock" internal/` → zero hits), mirroring R5's pattern.

**R11 — `newTestDeps` field extensions have single owners.** T2 lands `newTestDeps`
(targets_test.go) with Stdout/Stderr/Exit/Getenv/Now/Getwd/UserHomeDir only. Each edge field is
added by the FIRST task (per R4 order) whose constructor flip makes an executed targets-level test
dereference it. That task is T3 for BOTH `FS` and `Lock`: T3 flips the learn closures to
`runLearnFrom*Args(…, d, …)`/`newQaDeps(d)`, and the executed learn tests
(targets_test.go:206-263 — feedback/fact/qa through `executeForTest`, with the qa test asserting
real note files on disk) drive `newLearnDeps(d)`/`newQaDeps(d)` through `d.FS` (StatDir/InitVault/
ListIDs/WriteNew) and `d.Lock` (vaultLockFromLocker) — nil either field is a nil-interface panic.
T3 therefore extends `newTestDeps` with `FS: osEdgeFSForTest{}` and `Lock: flockLockerForTest{}`
(its own step-1 doubles, same package cli_test; exact diff in T3 step 7). Every later executed path
(count T5, show/show-chunk T5/T6, ingest T8, prune T9, vocab/amend/resituate/activate T12) rides
those same two fields — no other task extends `newTestDeps`. `Embed` (RESOLVED): `newTestDeps.Embed` stays NIL globally — post-T3, nil Embed makes
`autoEmbedNote`/`embedDefinitionNote` skip silently, which is byte-identical in observable output
(empty stderr) to today's silently-succeeding real sharedEmbedder, so the learn-qa
(targets_test.go:241-263) and vocab-propose (vocab_commands_test.go:~3597) empty-stderr assertions
keep passing unchanged. T15 — whose embed-closure flip makes `TestTargets_EmbedApplyDryRun` /
`TestTargets_EmbedStatus` (targets_test.go:340/355) dereference `deps.Embedder.ModelID()`
(embed.go:63; tallyStates embed.go:275) — wires a fail-loud stub ONLY in those two tests via a
local deps override (`d := newTestDeps(...); d.Embed = stubEmbedderForTargets{}`): the stub is
named `stubEmbedderForTargets` (NOT `stubEmbedder` — that name already exists in cli_test at
embed_test.go:213), with ModelID() returning `embed.BundledModelID` (matches today's lazy-embedder
output without loading the model), Dims() returning 384, and Embed() returning an error
("stubEmbedderForTargets: Embed not expected in targets-level tests") so real embedding through
it fails loud. The production lazy-embedder alternative is structurally unavailable: after T14 its
constructor needs the hugot Backend + CacheFS built in package main, unreachable from cli_test.
Exact stub declaration + per-test wiring in T15's steps.

**R12 — `ExportNewOsVaultFS` call-site migration owner: T12; shim deleter: T7.** T5 keeps the
`ExportNewOsVaultFS` shim (vocab tests still consume it). T12, which owns the vocab test files,
migrates EVERY remaining call site — vocab_trigger_test.go:251,441 and
vocab_commands_test.go:96,131,198,231,543,559,613,651,3856,3874 (12 sites; vault_fs_test.go's five
sites are already replaced wholesale by T5) — from `cli.ExportNewOsVaultFS()` to
`cli.ExportNewVaultFS(osTestEdgeFS{})` (T5's export over the cli_test real-FS EdgeFS double; same
`ListMD`/`ReadFile` interface shape and semantics). T7 then deletes the shim, and its gate grep is
extended to `osVaultFS\|ExportNewOsVaultFS` — the lowercase-only pattern would MISS the shim
(capital-O `OsVaultFS`) and let T7's deletion break the compile silently.

**R13 — `fakeEdgeFS` has ONE cli_test declaration: T8's.** T8 (ingest) declares the func-field
`fakeEdgeFS` in package cli_test. T17's draft declares a same-named `fstest.MapFS`-backed fake in
the same package — a redeclaration compile error if both land. T17's fake is RENAMED
`updateFakeEdgeFS` (its read-only errUnsupported write-side semantics differ from T8's
func-injection design, so reuse does not fit); every reference in T17's steps and test code uses
the new name. The executor note "reuse it and delete this one if so" is superseded by this rule.

## Issue-AC traceability (Gate A finding 3)

| Issue #700 acceptance criterion (verbatim key) | Owning task(s) |
|---|---|
| depguard `internal-purity` rule active (root-anchored `internal/**`, no negation globs, no companion adapter rule); forbidigo enabled with the custom clock/PRNG/rand-v1 list; no fmt.Print flagging | T-final-1 (glob per R9) |
| Zero non-test files under `internal/` import os, os/exec, os/signal, syscall, net, net/http, database/sql, or any third-party I/O package (hugot) | T1–T17 cumulative; enforced + verified T-final-1 |
| Zero time.Now/Since/Tick, global-PRNG, or math/rand (v1) references in non-test internal/ code | T3, T6, T8, T12 (Now threading); enforced T-final-1 |
| All eight former adapters decomposed: primitives referenced in cmd/engram's declaration-free main(), composition logic in internal/ with fake-primitive unit tests + real-primitive integration tests; targ check-thin-api PASS *(issue AC text updated 2026-07-19 after the doctrine correction; this row quotes the CURRENT wording)* | Read per the revised composition doctrine (user correction 2026-07-19, vault note 303): raw PRIMITIVES relocate to cmd (func references + sanctioned closures in the Primitives literal); adapter COMPOSITION lives in internal/cli, integration-tested in internal `_test` files — T1-rework (signal, debuglog sink, EdgeFS/flock), T14 (hugot + cache), T17 (commander); T10 reduces to consumer migration; absorption-into-EdgeFS flags cover vault_fs/learn-FS (T5/T3); `targ check-thin-api` enforces the cmd side |
| debuglog is pure (writer + clock injected); both direct env reads eliminated; env/clock threaded from cmd/engram; targ.Main called from cmd | T2 (debuglog, ENGRAM_DEBUG_LOG, targ.Main), T8 (ENGRAM_TRANSCRIPT_DIR) |
| internal/update no longer imports os/exec (sentinel translated in the cmd adapter) | T16, T17 — read per doctrine flag C-1: cmd contributes only a raw run primitive plus a `NotFoundErr error` Primitives field wiring `exec.ErrNotFound`; the `errors.Is` translation to `update.ErrCommandNotFound` lives internal (zero cmd logic) |
| Coverage stance for cmd/engram revisited deliberately | T-final-1 Step 5.5 |
| targ check-full green; truncation setting (max-issues-per-linter) decided deliberately | every task's final gate (check-full green AND `targ check-thin-api` PASS, per Global Constraints); T-final-1 Step 3 |
| FIXME at internal/cli/main.go removed — last, after everything above is green | T2 relocates the marker (R8); T-final-2 deletes it |

## Tasks

### Task T1-rework: compose EdgeFS/flock/sink internally from Primitives; thin cmd/engram (REWORK of landed commit de484526)

> **Context:** the original T1 landed at de484526 with the production adapters (`osFS`,
> `flockLocker`, `syncWriter`/`openDebugSink`, `registerForceExit`/`forwardAsPulses`) implemented
> in `cmd/engram` — and `targ check-thin-api` FAILED with 9 non-thin declarations across the 3 new
> cmd files. That layout is REJECTED (user correction 2026-07-19, vault note 303; see "Revised
> composition doctrine" under Design flags). This task REWORKS the landed state: the logic moves
> into `internal/cli` composed from a new `cli.Primitives` carrier of raw capability funcs, the six
> cmd/engram adapter files are deleted, and the relocated integration tests run against the
> composed implementations with REAL os/syscall primitives in internal `_test` files (sanctioned —
> the T-final-1 purity lint excludes `!$test`). What SURVIVES from de484526 unchanged:
> `internal/cli/deps.go` (Deps/EdgeFS/FileLocker, incl. the `//nolint:interfacebloat`), the pure
> `ForceExitOnRepeatedSignal` + its signal_test.go tests, the `SetupSignalHandling` shim +
> `internal/cli/main.go` (both die in T2), and `cmd/engram/main.go`'s single-statement form.

**Files:**
- Create: `internal/cli/primitives.go` (Primitives, WriteSyncer, NewDeps, envOrEmpty)
- Create: `internal/cli/edgefs.go` (primFS — EdgeFS composed from primitives, %w wraps + ADR-0013 atomic dance)
- Create: `internal/cli/locker.go` (primLocker — flock lifecycle over fd primitives)
- Create: `internal/cli/debugsink.go` (openDebugSink, syncWriter, debugLogEnvVar/debugLogPerm)
- Create: `internal/cli/primitives_test.go`, `internal/cli/edgefs_test.go`, `internal/cli/locker_test.go` (unit, fake primitives)
- Create: `internal/cli/primitives_integration_test.go`, `internal/cli/signal_integration_test.go` (real os/syscall — relocated cmd suites)
- Modify: `internal/cli/signal.go` (add generic `ForwardAsPulses` + `startForceExit`; shim goroutine now calls ForwardAsPulses)
- Modify: `internal/cli/signal_test.go` (add ForwardAsPulses test, chan int)
- Delete: `cmd/engram/os_fs.go`, `cmd/engram/os_fs_test.go`, `cmd/engram/os_signal.go`, `cmd/engram/os_signal_test.go`, `cmd/engram/debuglog_sink.go`, `cmd/engram/debuglog_sink_test.go`
- Unchanged: `internal/cli/deps.go`, `cmd/engram/main.go` (still the 1-statement `cli.Main(...)` — T2 rewrites it), `internal/cli/main.go`

**Interfaces:**
- Consumes: `cli.Deps`/`cli.EdgeFS`/`cli.FileLocker` (landed, unchanged); `cli.ForceExitOnRepeatedSignal`, `cli.ExitCodeSigInt`.
- Produces: `cli.Primitives` (struct — exact fields in the doctrine subsection, consume verbatim); `type WriteSyncer interface { io.Writer; Sync() error }`; `func NewDeps(prims Primitives, stdout, stderr io.Writer, exit func(int)) Deps`; `func ForwardAsPulses[T any](in <-chan T, out chan<- struct{})`; unexported `primFS`, `primLocker`, `openDebugSink`, `syncWriter`, `startForceExit`, `envOrEmpty`, consts `lockFilePerm`/`debugLogPerm`/`debugLogEnvVar`/`maxTempAttempts`, sentinel `errTempNameExhausted`.

**Steps:**

1. [ ] RED — create the unit-test files against the not-yet-existing composition seams. All three compile-fail (`undefined: cli.Primitives`, `undefined: cli.NewDeps`) — that is the RED; the behaviors they pin (wrap-with-%w, temp-cleanup-on-failure, unique-temp-name collision retry, fd-close-on-flock-failure, per-write Sync, force-exit watcher registration) were UNTESTABLE against the real-os cmd adapters, which is exactly the seam this rework buys.

   `internal/cli/primitives_test.go`:

```go
package cli_test

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

func TestNewDeps_ComposesCarrierFromPrimitives(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	var stdout, stderr bytes.Buffer

	exitCodes := make([]int, 0, 1)
	fixed := time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC)

	prims := cli.Primitives{
		Getenv:      func(string) string { return "" },
		Now:         func() time.Time { return fixed },
		Getwd:       func() (string, error) { return "/work", nil },
		UserHomeDir: func() (string, error) { return "/home/x", nil },
	}

	deps := cli.NewDeps(prims, &stdout, &stderr, func(code int) { exitCodes = append(exitCodes, code) })

	g.Expect(deps.Stdout).To(gomega.BeIdenticalTo(&stdout))
	g.Expect(deps.Stderr).To(gomega.BeIdenticalTo(&stderr))
	g.Expect(deps.Now()).To(gomega.Equal(fixed))

	wd, err := deps.Getwd()
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(wd).To(gomega.Equal("/work"))

	home, homeErr := deps.UserHomeDir()
	g.Expect(homeErr).NotTo(gomega.HaveOccurred())
	g.Expect(home).To(gomega.Equal("/home/x"))

	g.Expect(deps.FS).NotTo(gomega.BeNil())
	g.Expect(deps.Lock).NotTo(gomega.BeNil())

	deps.Exit(3)
	g.Expect(exitCodes).To(gomega.Equal([]int{3}))
}

func TestNewDeps_DebugSinkEmptyEnvOrFailedOpenIsNil(t *testing.T) {
	t.Parallel()

	t.Run("empty env yields nil sink without opening", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		prims := cli.Primitives{
			Getenv: func(string) string { return "" },
			OpenDebugFile: func(string, fs.FileMode) (cli.WriteSyncer, error) {
				t.Error("open must not be called for an empty path")

				return nil, nil
			},
		}

		g.Expect(cli.NewDeps(prims, io.Discard, io.Discard, func(int) {}).DebugLog).To(gomega.BeNil())
	})

	t.Run("failed open yields nil sink", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		prims := cli.Primitives{
			Getenv: func(string) string { return "/nope/debug.log" },
			OpenDebugFile: func(string, fs.FileMode) (cli.WriteSyncer, error) {
				return nil, errors.New("open failed")
			},
		}

		g.Expect(cli.NewDeps(prims, io.Discard, io.Discard, func(int) {}).DebugLog).To(gomega.BeNil())
	})
}

func TestNewDeps_DebugSinkSyncsEveryWrite(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	sink := &recordingSyncer{}
	prims := cli.Primitives{
		Getenv: func(key string) string {
			if key == "ENGRAM_DEBUG_LOG" {
				return "/dev/fake/debug.log"
			}

			return ""
		},
		OpenDebugFile: func(path string, _ fs.FileMode) (cli.WriteSyncer, error) {
			g.Expect(path).To(gomega.Equal("/dev/fake/debug.log"))

			return sink, nil
		},
	}

	deps := cli.NewDeps(prims, io.Discard, io.Discard, func(int) {})
	g.Expect(deps.DebugLog).NotTo(gomega.BeNil())

	if deps.DebugLog == nil {
		return
	}

	_, err := deps.DebugLog.Write([]byte("line one\n"))
	g.Expect(err).NotTo(gomega.HaveOccurred())

	_, err = deps.DebugLog.Write([]byte("line two\n"))
	g.Expect(err).NotTo(gomega.HaveOccurred())

	g.Expect(sink.contents()).To(gomega.Equal("line one\nline two\n"))
	g.Expect(sink.syncCount()).To(gomega.Equal(2), "per-line Sync is the tail -F liveness contract")
}

func TestNewDeps_StartsForceExitWatcherFromPrimitive(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	pulsesCh := make(chan chan<- struct{}, 1)
	exitCodes := make(chan int, 1)

	prims := cli.Primitives{
		StartSignalPulses: func(pulses chan<- struct{}, buffer int) {
			g.Expect(buffer).To(gomega.BeNumerically(">", 0))
			pulsesCh <- pulses
		},
	}

	cli.NewDeps(prims, io.Discard, io.Discard, func(code int) { exitCodes <- code })

	var pulses chan<- struct{}

	select {
	case pulses = <-pulsesCh:
	case <-time.After(time.Second):
		t.Fatal("StartSignalPulses was not invoked by NewDeps")
	}

	pulses <- struct{}{}

	pulses <- struct{}{}

	select {
	case code := <-exitCodes:
		g.Expect(code).To(gomega.Equal(cli.ExitCodeSigInt))
	case <-time.After(time.Second):
		t.Fatal("exit not called after two pulses")
	}
}

func TestNewDeps_ZeroPrimitivesDisablesOptionalEdges(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	deps := cli.NewDeps(cli.Primitives{}, io.Discard, io.Discard, func(int) {})

	g.Expect(deps.DebugLog).To(gomega.BeNil())
}

// recordingSyncer is a fake WriteSyncer that records writes and counts Sync
// calls (safe for concurrent use).
type recordingSyncer struct {
	mu    sync.Mutex
	data  strings.Builder
	syncs int
}

func (r *recordingSyncer) Sync() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.syncs++

	return nil
}

func (r *recordingSyncer) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.data.Write(p)
}

func (r *recordingSyncer) contents() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.data.String()
}

func (r *recordingSyncer) syncCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.syncs
}
```

   `internal/cli/edgefs_test.go`:

```go
package cli_test

import (
	"errors"
	"io"
	"io/fs"
	"path/filepath"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

func TestEdgeFS_PreservesSentinelChainsThroughWrapping(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	fsys := fsFromPrims(cli.Primitives{
		ReadFile: func(string) ([]byte, error) {
			return nil, &fs.PathError{Op: "open", Path: "x", Err: fs.ErrNotExist}
		},
	})

	_, err := fsys.ReadFile("x")
	g.Expect(err).To(gomega.MatchError(fs.ErrNotExist), "%w wrapping must preserve errors.Is chains")
	g.Expect(err.Error()).To(gomega.ContainSubstring("x"), "wrap must add path context")
}

func TestEdgeFS_WriteFileAtomicFailuresRemoveTemp(t *testing.T) {
	t.Parallel()

	boom := errors.New("boom")

	t.Run("rename failure removes the created temp", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		var created string

		removed := make([]string, 0, 1)
		prims := cli.Primitives{
			Now: func() time.Time { return time.Unix(0, fakeDanceNanos) },
			WriteFileExcl: func(path string, _ []byte, _ fs.FileMode) error {
				created = path

				return nil
			},
			Rename: func(string, string) error { return boom },
			Remove: func(path string) error {
				removed = append(removed, path)

				return nil
			},
		}

		err := fsFromPrims(prims).WriteFileAtomic(filepath.Join("d", "n"), []byte("x"), atomicPerm)
		g.Expect(err).To(gomega.MatchError(boom))
		g.Expect(err.Error()).To(gomega.ContainSubstring("rename"))
		g.Expect(removed).To(gomega.Equal([]string{created}),
			"a failed dance must remove the temp file it created")
	})

	t.Run("chmod failure removes the created temp", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		var created string

		removed := make([]string, 0, 1)
		prims := cli.Primitives{
			Now: func() time.Time { return time.Unix(0, fakeDanceNanos) },
			WriteFileExcl: func(path string, _ []byte, _ fs.FileMode) error {
				created = path

				return nil
			},
			Chmod: func(string, fs.FileMode) error { return boom },
			Remove: func(path string) error {
				removed = append(removed, path)

				return nil
			},
		}

		err := fsFromPrims(prims).WriteFileAtomic(filepath.Join("d", "n"), []byte("x"), atomicPerm)
		g.Expect(err).To(gomega.MatchError(boom))
		g.Expect(err.Error()).To(gomega.ContainSubstring("chmod"))
		g.Expect(removed).To(gomega.Equal([]string{created}),
			"a failed dance must remove the temp file it created")
	})

	t.Run("exclusive-create failure aborts with nothing to clean", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		prims := cli.Primitives{
			Now:           func() time.Time { return time.Unix(0, fakeDanceNanos) },
			WriteFileExcl: func(string, []byte, fs.FileMode) error { return boom },
			Remove: func(string) error {
				t.Error("nothing was created, so nothing may be removed")

				return nil
			},
		}

		err := fsFromPrims(prims).WriteFileAtomic(filepath.Join("d", "n"), []byte("x"), atomicPerm)
		g.Expect(err).To(gomega.MatchError(boom))
		g.Expect(err.Error()).To(gomega.ContainSubstring("create temp"))
	})
}

func TestEdgeFS_WriteFileAtomicHappyPathDance(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	calls := &callRecorder{}
	target := filepath.Join("some", "dir", "note.md")

	fsys := fsFromPrims(cli.Primitives{
		Now: func() time.Time { return time.Unix(0, fakeDanceNanos) },
		WriteFileExcl: func(path string, data []byte, perm fs.FileMode) error {
			g.Expect(filepath.Dir(path)).To(gomega.Equal(filepath.Join("some", "dir")),
				"temp must be created in the target's dir — same-directory rename is the ADR-0013 primitive")
			g.Expect(filepath.Base(path)).To(gomega.Equal(".note.md.tmp-12345-0"),
				"candidate names derive from target base + clock nanos + attempt counter (P-4)")
			g.Expect(string(data)).To(gomega.Equal("v2"), "the data lands in the exclusive create itself")
			g.Expect(perm).To(gomega.Equal(atomicPerm), "the target perm reaches the exclusive create")
			calls.add("writeexcl " + filepath.Base(path))

			return nil
		},
		Chmod: func(path string, perm fs.FileMode) error {
			g.Expect(perm).To(gomega.Equal(atomicPerm),
				"chmod must force the EXACT target perm regardless of umask")
			calls.add("chmod " + filepath.Base(path))

			return nil
		},
		Rename: func(oldPath, newPath string) error {
			calls.add("rename " + filepath.Base(oldPath) + "->" + filepath.Base(newPath))

			return nil
		},
		Remove: func(path string) error {
			calls.add("remove " + filepath.Base(path))

			return nil
		},
	})

	g.Expect(fsys.WriteFileAtomic(target, []byte("v2"), atomicPerm)).To(gomega.Succeed())
	g.Expect(calls.list()).To(gomega.Equal([]string{
		"writeexcl .note.md.tmp-12345-0",
		"chmod .note.md.tmp-12345-0",
		"rename .note.md.tmp-12345-0->note.md",
	}), "success path must not remove the renamed file")
}

func TestEdgeFS_WriteFileAtomicUniqueNameRetry(t *testing.T) {
	t.Parallel()

	t.Run("collision retries a fresh candidate then succeeds", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		target := filepath.Join("some", "dir", "note.md")
		tried := make([]string, 0, 2)

		var renamed string

		prims := cli.Primitives{
			Now: func() time.Time { return time.Unix(0, fakeDanceNanos) },
			WriteFileExcl: func(path string, _ []byte, _ fs.FileMode) error {
				tried = append(tried, path)
				if len(tried) == 1 {
					return &fs.PathError{Op: "open", Path: path, Err: fs.ErrExist}
				}

				return nil
			},
			Rename: func(oldPath, _ string) error {
				renamed = oldPath

				return nil
			},
			Remove: func(string) error {
				t.Error("a colliding candidate was not created by the dance and must not be removed")

				return nil
			},
		}

		g.Expect(fsFromPrims(prims).WriteFileAtomic(target, []byte("v2"), atomicPerm)).To(gomega.Succeed())
		g.Expect(tried).To(gomega.HaveLen(2))
		g.Expect(tried[0]).NotTo(gomega.Equal(tried[1]), "each retry must try a FRESH candidate name")
		g.Expect(renamed).To(gomega.Equal(tried[1]), "the created candidate is the one renamed into place")
	})

	t.Run("exhausted candidates yield a bounded wrapped error", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		attempts := 0
		prims := cli.Primitives{
			Now: func() time.Time { return time.Unix(0, fakeDanceNanos) },
			WriteFileExcl: func(path string, _ []byte, _ fs.FileMode) error {
				attempts++

				return &fs.PathError{Op: "open", Path: path, Err: fs.ErrExist}
			},
		}

		err := fsFromPrims(prims).WriteFileAtomic(filepath.Join("d", "n"), []byte("x"), atomicPerm)
		g.Expect(err).To(gomega.MatchError(fs.ErrExist), "the last collision stays in the error chain")
		g.Expect(err.Error()).To(gomega.ContainSubstring("create temp"))
		g.Expect(err.Error()).To(gomega.ContainSubstring("attempts"))
		g.Expect(attempts).To(gomega.Equal(danceMaxAttempts), "the retry loop must be BOUNDED")
	})
}

// fsFromPrims composes the production EdgeFS from fake primitives via the
// public composition root.
func fsFromPrims(prims cli.Primitives) cli.EdgeFS {
	return cli.NewDeps(prims, io.Discard, io.Discard, func(int) {}).FS
}

// callRecorder records call labels in order (single-goroutine use).
type callRecorder struct{ calls []string }

func (c *callRecorder) add(call string) { c.calls = append(c.calls, call) }

func (c *callRecorder) list() []string { return c.calls }

// unexported constants.
const (
	atomicPerm fs.FileMode = 0o600

	// danceMaxAttempts mirrors edgefs.go's maxTempAttempts — the spec'd
	// bound on unique-temp-name candidates (doctrine flag P-4).
	danceMaxAttempts = 10

	// fakeDanceNanos is the fixed clock reading the dance fakes inject;
	// candidate temp names embed it.
	fakeDanceNanos = 12345
)
```

   `internal/cli/locker_test.go`:

```go
package cli_test

import (
	"errors"
	"io"
	"io/fs"
	"testing"

	"github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

func TestPrimLocker_FlockFailureClosesDescriptor(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	const fakeFD = uintptr(7)

	closed := make([]uintptr, 0, 1)
	boom := errors.New("flock boom")

	locker := lockerFromPrims(cli.Primitives{
		OpenLockFile:   func(string, fs.FileMode) (uintptr, error) { return fakeFD, nil },
		FlockExclusive: func(uintptr) error { return boom },
		CloseFD: func(fd uintptr) error {
			closed = append(closed, fd)

			return nil
		},
	})

	_, err := locker.Lock("/vault/.lock")
	g.Expect(err).To(gomega.MatchError(boom))
	g.Expect(err.Error()).To(gomega.ContainSubstring("/vault/.lock"))
	g.Expect(closed).To(gomega.Equal([]uintptr{fakeFD}), "flock failure must not leak the fd")
}

func TestPrimLocker_OpenFailureWrapsWithPath(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	boom := errors.New("open boom")

	locker := lockerFromPrims(cli.Primitives{
		OpenLockFile: func(string, fs.FileMode) (uintptr, error) { return 0, boom },
		FlockExclusive: func(uintptr) error {
			t.Error("flock must not run after a failed open")

			return nil
		},
	})

	_, err := locker.Lock("/vault/.lock")
	g.Expect(err).To(gomega.MatchError(boom))
	g.Expect(err.Error()).To(gomega.ContainSubstring("open lock /vault/.lock"))
}

func TestPrimLocker_UnlockLifecycle(t *testing.T) {
	t.Parallel()

	t.Run("unlock flocks LOCK_UN then closes", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		calls := &callRecorder{}
		locker := lockerFromPrims(cli.Primitives{
			OpenLockFile: func(string, fs.FileMode) (uintptr, error) { return 4, nil },
			FlockExclusive: func(uintptr) error {
				calls.add("flock-ex")

				return nil
			},
			FlockUnlock: func(uintptr) error {
				calls.add("flock-un")

				return nil
			},
			CloseFD: func(uintptr) error {
				calls.add("close")

				return nil
			},
		})

		unlock, err := locker.Lock("/vault/.lock")
		g.Expect(err).NotTo(gomega.HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(unlock()).To(gomega.Succeed())
		g.Expect(calls.list()).To(gomega.Equal([]string{"flock-ex", "flock-un", "close"}))
	})

	t.Run("funlock error reported and fd still closed", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		boom := errors.New("funlock boom")
		closed := make([]uintptr, 0, 1)
		locker := lockerFromPrims(cli.Primitives{
			OpenLockFile:   func(string, fs.FileMode) (uintptr, error) { return 4, nil },
			FlockExclusive: func(uintptr) error { return nil },
			FlockUnlock:    func(uintptr) error { return boom },
			CloseFD: func(fd uintptr) error {
				closed = append(closed, fd)

				return nil
			},
		})

		unlock, err := locker.Lock("/vault/.lock")
		g.Expect(err).NotTo(gomega.HaveOccurred())

		if err != nil {
			return
		}

		unlockErr := unlock()
		g.Expect(unlockErr).To(gomega.MatchError(boom))
		g.Expect(unlockErr.Error()).To(gomega.ContainSubstring("funlock"))
		g.Expect(closed).To(gomega.HaveLen(1), "unlock error must not leak the fd")
	})

	t.Run("close error reported when funlock succeeds", func(t *testing.T) {
		t.Parallel()
		g := gomega.NewWithT(t)

		boom := errors.New("close boom")
		locker := lockerFromPrims(cli.Primitives{
			OpenLockFile:   func(string, fs.FileMode) (uintptr, error) { return 4, nil },
			FlockExclusive: func(uintptr) error { return nil },
			FlockUnlock:    func(uintptr) error { return nil },
			CloseFD:        func(uintptr) error { return boom },
		})

		unlock, err := locker.Lock("/vault/.lock")
		g.Expect(err).NotTo(gomega.HaveOccurred())

		if err != nil {
			return
		}

		unlockErr := unlock()
		g.Expect(unlockErr).To(gomega.MatchError(boom))
		g.Expect(unlockErr.Error()).To(gomega.ContainSubstring("close lock"))
	})
}

// lockerFromPrims composes the production FileLocker from fake primitives
// via the public composition root.
func lockerFromPrims(prims cli.Primitives) cli.FileLocker {
	return cli.NewDeps(prims, io.Discard, io.Discard, func(int) {}).Lock
}
```

   Also modify `internal/cli/signal_test.go` — add (same imports; `cli` already imported):

```go
func TestForwardAsPulses_ForwardsEachValue(t *testing.T) {
	t.Parallel()

	const valueCount = 2

	in := make(chan int, valueCount)
	pulses := make(chan struct{}, valueCount)

	go cli.ForwardAsPulses(in, pulses)

	in <- 1

	in <- 2

	const pulseTimeout = time.Second

	for range valueCount {
		select {
		case <-pulses:
		case <-time.After(pulseTimeout):
			t.Fatal("pulse not forwarded within timeout")
		}
	}
}
```

   Run `targ test` — expect compile failure in internal/cli tests (`undefined: cli.Primitives`, `undefined: cli.NewDeps`, `undefined: cli.ForwardAsPulses`). That is the RED.

2. [ ] GREEN — create the internal composition. `internal/cli/primitives.go` (the Primitives struct is the doctrine subsection's canonical inventory — byte-identical):

```go
package cli

import (
	"io"
	"io/fs"
	"time"
)

// Primitives carries raw impure capabilities as func values. cmd/engram
// populates it with direct references to os/syscall/filepath/time functions,
// single-call closures where a signature must be erased (fd instead of
// *os.File, WriteSyncer instead of *os.File, pulses instead of os.Signal),
// or an enumerated stdlib-equivalent survivor closure (doctrine survivors:
// S-1 WriteFileExcl here; C-1 RunCommand lands in T17).
// ALL composition, error wrapping, and lifecycle logic lives in internal/cli;
// targ check-thin-api enforces that the cmd side stays declaration-free (#700).
type Primitives struct {
	// Filesystem (direct os/filepath references).
	ReadFile  func(path string) ([]byte, error)                      // os.ReadFile
	WriteFile func(path string, data []byte, perm fs.FileMode) error // os.WriteFile
	MkdirAll  func(path string, perm fs.FileMode) error              // os.MkdirAll
	MkdirTemp func(dir, pattern string) (string, error)              // os.MkdirTemp
	Stat      func(path string) (fs.FileInfo, error)                 // os.Stat
	ReadDir   func(path string) ([]fs.DirEntry, error)               // os.ReadDir
	Remove    func(path string) error                                // os.Remove
	RemoveAll func(path string) error                                // os.RemoveAll
	Rename    func(oldPath, newPath string) error                    // os.Rename
	WalkDir   func(root string, fn fs.WalkDirFunc) error             // filepath.WalkDir
	Chmod     func(path string, mode fs.FileMode) error              // os.Chmod

	// Exclusive create (doctrine survivor S-1 — a stdlib-equivalent
	// primitive closure: os.WriteFile's own body with O_CREATE|O_EXCL;
	// behavior changes extend this SIGNATURE, never the cmd body).
	WriteFileExcl func(path string, data []byte, perm fs.FileMode) error

	// Process, env, clock (direct references).
	Getenv      func(key string) string // os.Getenv
	Now         func() time.Time        // time.Now
	Getwd       func() (string, error)  // os.Getwd
	UserHomeDir func() (string, error)  // os.UserHomeDir

	// Advisory file locking (single-syscall closures; lifecycle internal —
	// design flags P-2/P-3: semantic per-op funcs over a raw uintptr fd,
	// via syscall.Open, never os.OpenFile().Fd()).
	OpenLockFile   func(path string, perm fs.FileMode) (uintptr, error) // syscall.Open O_CREAT|O_RDWR
	FlockExclusive func(fd uintptr) error                               // syscall.Flock LOCK_EX
	FlockUnlock    func(fd uintptr) error                               // syscall.Flock LOCK_UN
	CloseFD        func(fd uintptr) error                               // syscall.Close

	// Debug sink (single-call closure; empty-path branch + sync policy internal).
	OpenDebugFile func(path string, perm fs.FileMode) (WriteSyncer, error) // os.OpenFile O_APPEND|O_CREATE|O_WRONLY

	// Signal (single-purpose starter closure; pulse forwarding is internal
	// via ForwardAsPulses; buffer/pulse-channel/force-exit policy internal).
	StartSignalPulses func(pulses chan<- struct{}, buffer int)
}

// WriteSyncer is the debug-sink capability surface (*os.File satisfies it).
type WriteSyncer interface {
	io.Writer
	Sync() error
}

// NewDeps composes the production Deps carrier from raw primitives: the
// EdgeFS implementation (contextual %w wrapping + the ADR-0013 atomic-write
// dance), the flock lifecycle, the debug sink (ENGRAM_DEBUG_LOG; empty path
// or failed open → nil → no-op logger), and the repeated-signal force-exit
// watcher. cmd/engram calls this exactly once from main(); tests call it
// with fake primitives to unit-test the composition (#700).
func NewDeps(prims Primitives, stdout, stderr io.Writer, exit func(int)) Deps {
	startForceExit(prims, exit)

	return Deps{
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
}

// envOrEmpty reads key via getenv, tolerating a nil (unwired) capability.
func envOrEmpty(getenv func(string) string, key string) string {
	if getenv == nil {
		return ""
	}

	return getenv(key)
}
```

   `internal/cli/edgefs.go` — the landed cmd/engram/os_fs.go `osFS` logic verbatim with `os.X`/`filepath.WalkDir` swapped for `f.prims.X`, PLUS the atomic-write dance re-sequenced for internal unique-temp-name generation over the exclusive-create `WriteFileExcl` primitive (design flags P-4/S-1 — same-directory rename atomicity unchanged):

```go
package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
)

// Compile-time interface conformance (internal — the thin-api checker does
// not walk internal/).
var _ EdgeFS = primFS{}

// primFS is the production EdgeFS: it composes the injected raw primitives
// with contextual error wrapping (%w preserves errors.Is chains such as
// fs.ErrNotExist) and the ADR-0013 atomic-write dance. All orchestration
// lives here in internal/; cmd/engram contributes only raw os/filepath
// references (#700).
type primFS struct {
	prims Primitives
}

// MkdirAll creates path with any missing parents; no-op when path exists.
func (f primFS) MkdirAll(path string, perm fs.FileMode) error {
	err := f.prims.MkdirAll(path, perm)
	if err != nil {
		return fmt.Errorf("mkdir %s: %w", path, err)
	}

	return nil
}

// MkdirTemp creates a fresh unique directory in dir matching pattern.
func (f primFS) MkdirTemp(dir, pattern string) (string, error) {
	made, err := f.prims.MkdirTemp(dir, pattern)
	if err != nil {
		return "", fmt.Errorf("mkdir temp in %s: %w", dir, err)
	}

	return made, nil
}

// ReadDir returns the directory entries of path.
func (f primFS) ReadDir(path string) ([]fs.DirEntry, error) {
	entries, err := f.prims.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", path, err)
	}

	return entries, nil
}

// ReadFile reads the file at path.
func (f primFS) ReadFile(path string) ([]byte, error) {
	data, err := f.prims.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	return data, nil
}

// Remove deletes the file or empty directory at path.
func (f primFS) Remove(path string) error {
	err := f.prims.Remove(path)
	if err != nil {
		return fmt.Errorf("remove %s: %w", path, err)
	}

	return nil
}

// RemoveAll deletes path and any children; no-op when path is absent.
func (f primFS) RemoveAll(path string) error {
	err := f.prims.RemoveAll(path)
	if err != nil {
		return fmt.Errorf("remove all %s: %w", path, err)
	}

	return nil
}

// Rename atomically renames oldPath to newPath (same-directory renames are
// atomic on POSIX — the ADR-0013 primitive).
func (f primFS) Rename(oldPath, newPath string) error {
	err := f.prims.Rename(oldPath, newPath)
	if err != nil {
		return fmt.Errorf("rename %s -> %s: %w", oldPath, newPath, err)
	}

	return nil
}

// Stat returns the fs.FileInfo for path.
func (f primFS) Stat(path string) (fs.FileInfo, error) {
	info, err := f.prims.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}

	return info, nil
}

// WalkDir walks the file tree rooted at root, calling fn for each entry.
func (f primFS) WalkDir(root string, fn fs.WalkDirFunc) error {
	err := f.prims.WalkDir(root, fn)
	if err != nil {
		return fmt.Errorf("walk %s: %w", root, err)
	}

	return nil
}

// WriteFile writes data to path with perm.
func (f primFS) WriteFile(path string, data []byte, perm fs.FileMode) error {
	err := f.prims.WriteFile(path, data, perm)
	if err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	return nil
}

// WriteFileAtomic writes data to path atomically: it derives a unique temp
// name in filepath.Dir(path) from the target base + the injected clock's
// nanos + an attempt counter, creates it exclusively (data written at perm)
// via the WriteFileExcl primitive — retrying fresh candidates on
// fs.ErrExist, bounded by maxTempAttempts — then chmods the temp to the
// exact target perm (umask-independent) and renames into place. A
// same-directory rename is atomic on POSIX — a concurrent reader sees
// either the old or the new file, never a partial one. On any failure
// after creation the temp file is removed and the original (if any) is
// left untouched (ADR-0013; design flag P-4: the unique-temp-name policy
// is INTERNAL — cmd contributes only the stdlib-equivalent WriteFileExcl
// primitive, doctrine survivor S-1, plus the restored direct Chmod
// primitive for umask-independent perms).
func (f primFS) WriteFileAtomic(path string, data []byte, perm fs.FileMode) error {
	tmpName, err := f.createUniqueTemp(path, data, perm)
	if err != nil {
		return fmt.Errorf("atomic write %s: create temp: %w", path, err)
	}

	// chmod after write (temp is never wider than final); explicit chmod
	// keeps atomic-write perms umask-independent — parity with the
	// pre-#700 dance. Do NOT reorder chmod before the data write.
	chmodErr := f.prims.Chmod(tmpName, perm)
	if chmodErr != nil {
		_ = f.prims.Remove(tmpName)

		return fmt.Errorf("atomic write %s: chmod temp: %w", path, chmodErr)
	}

	renameErr := f.prims.Rename(tmpName, path)
	if renameErr != nil {
		// Cleanup on any failure after creation (P-4).
		_ = f.prims.Remove(tmpName)

		return fmt.Errorf("atomic write %s: rename: %w", path, renameErr)
	}

	return nil
}

// createUniqueTemp writes data exclusively to a fresh candidate temp name
// beside path (".<base>.tmp-<nanos>-<attempt>"). A candidate that already
// exists (fs.ErrExist) is retried with the next attempt counter, bounded
// by maxTempAttempts; any other error aborts immediately — nothing was
// created, so there is nothing to clean.
func (f primFS) createUniqueTemp(path string, data []byte, perm fs.FileMode) (string, error) {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	nanos := f.prims.Now().UnixNano()

	var lastErr error

	for attempt := range maxTempAttempts {
		candidate := filepath.Join(dir, fmt.Sprintf(".%s.tmp-%d-%d", base, nanos, attempt))

		lastErr = f.prims.WriteFileExcl(candidate, data, perm)
		if lastErr == nil {
			return candidate, nil
		}

		if !errors.Is(lastErr, fs.ErrExist) {
			return "", lastErr
		}
	}

	return "", fmt.Errorf("%w after %d attempts: %w", errTempNameExhausted, maxTempAttempts, lastErr)
}

// unexported variables.
var errTempNameExhausted = errors.New("no unique temp name available")

// unexported constants.
const (
	// maxTempAttempts bounds the fs.ErrExist retry when deriving a unique
	// temp name for the atomic-write dance (doctrine flag P-4).
	maxTempAttempts = 10
)
```

   `internal/cli/locker.go` — the landed `flockLocker.Lock` lifecycle verbatim over the fd primitives:

```go
package cli

import "fmt"

// Compile-time interface conformance.
var _ FileLocker = primLocker{}

// primLocker is the production FileLocker: an exclusive advisory flock via
// the injected syscall primitives. Open-then-flock, unlock-then-close, and
// the unlock-error semantics all live here (ADR-0013; #700). The fd-shaped
// primitives deliberately avoid *os.File (design flag P-3: a dropped
// *os.File's finalizer would close the fd and silently release the lock
// mid-hold).
type primLocker struct {
	prims Primitives
}

// Lock acquires an exclusive flock on path, creating the file if absent.
func (l primLocker) Lock(path string) (func() error, error) {
	fd, err := l.prims.OpenLockFile(path, lockFilePerm)
	if err != nil {
		return nil, fmt.Errorf("open lock %s: %w", path, err)
	}

	flockErr := l.prims.FlockExclusive(fd)
	if flockErr != nil {
		_ = l.prims.CloseFD(fd)

		return nil, fmt.Errorf("flock %s: %w", path, flockErr)
	}

	unlock := func() error {
		unlockErr := l.prims.FlockUnlock(fd)
		closeErr := l.prims.CloseFD(fd)

		if unlockErr != nil {
			return fmt.Errorf("funlock %s: %w", path, unlockErr)
		}

		if closeErr != nil {
			return fmt.Errorf("close lock %s: %w", path, closeErr)
		}

		return nil
	}

	return unlock, nil
}

// unexported constants.
const (
	lockFilePerm = 0o600
)
```

   `internal/cli/debugsink.go` — the landed cmd sink logic, parameterized over the open primitive (perm policy internal, design flag P-1):

```go
package cli

import (
	"fmt"
	"io"
	"io/fs"
)

// openDebugSink builds the debug-log sink: nil for an empty path, an
// unwired open capability, or a failed open — debuglog treats a nil writer
// as "logging disabled", so the CLI still runs (pre-#700 behavior
// preserved). Otherwise every write is followed by Sync so `tail -F` shows
// progress live.
func openDebugSink(path string, open func(string, fs.FileMode) (WriteSyncer, error)) io.Writer {
	if path == "" || open == nil {
		return nil
	}

	file, err := open(path, debugLogPerm)
	if err != nil {
		return nil
	}

	return &syncWriter{file: file}
}

// syncWriter flushes after every write. debuglog is documented tail -F
// friendly; the Logger sees only an io.Writer, so the per-line sync lives
// here in the composed sink.
type syncWriter struct {
	file WriteSyncer
}

// Write appends p and syncs the underlying sink.
func (w *syncWriter) Write(p []byte) (int, error) {
	n, err := w.file.Write(p)
	if err != nil {
		return n, fmt.Errorf("debug log write: %w", err)
	}

	_ = w.file.Sync()

	return n, nil
}

// unexported constants.
const (
	debugLogEnvVar             = "ENGRAM_DEBUG_LOG"
	debugLogPerm   fs.FileMode = 0o644
)
```

   Modify `internal/cli/signal.go` — add `ForwardAsPulses` + `startForceExit`, and point the shim's goroutine at the generic (full replacement file):

```go
package cli

import (
	"io"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"

	"github.com/toejough/engram/internal/debuglog"
)

// Exported constants.
const (
	ExitCodeSigInt = 130
)

// ForceExitOnRepeatedSignal starts a goroutine that waits for two pulses
// on the given channel, then calls exitFn. The first pulse allows graceful
// shutdown; the second forces immediate exit. Pulses are pure struct{}
// units — cmd/engram adapts real os.Signal deliveries into them, so no
// os.Signal type crosses into internal/ (#700).
func ForceExitOnRepeatedSignal(signals <-chan struct{}, exitFn func(int)) {
	var signalCount atomic.Int32

	go func() {
		for range signals {
			count := signalCount.Add(1)
			if count >= secondSignal {
				// Second signal or later: force exit immediately
				exitFn(ExitCodeSigInt)

				return
			}
			// First signal: will be handled by targ's context cancellation
		}
	}()
}

// ForwardAsPulses forwards each value received on in as a unit pulse on
// out. It is generic so cmd/engram can feed a chan os.Signal without
// os.Signal entering any internal signature (#700); tests drive it with a
// chan int.
func ForwardAsPulses[T any](in <-chan T, out chan<- struct{}) {
	for range in {
		out <- struct{}{}
	}
}

// SetupSignalHandling registers signal handlers and starts the force-exit goroutine.
// Returns the configured targets for targ.Main.
//
// Deprecated: interim shim only — deleted by the #700 wiring task; cmd/engram
// wires signal pulses through cli.Primitives.StartSignalPulses instead.
func SetupSignalHandling(
	stdout, stderr io.Writer,
	exitFn func(int),
	logger *debuglog.Logger,
) []any {
	sigCh := make(chan os.Signal, signalChannelBuffer)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	pulses := make(chan struct{}, signalChannelBuffer)

	go ForwardAsPulses(sigCh, pulses)

	ForceExitOnRepeatedSignal(pulses, exitFn)

	return Targets(stdout, stderr, exitFn, logger)
}

// startForceExit starts the repeated-signal force-exit watcher from the
// injected starter primitive. A nil primitive or exit func (minimal test
// Deps) skips registration. The pulse channel, buffer size, and force-exit
// policy live here — cmd only subscribes and forwards (#700).
func startForceExit(prims Primitives, exit func(int)) {
	if prims.StartSignalPulses == nil || exit == nil {
		return
	}

	pulses := make(chan struct{}, signalChannelBuffer)
	prims.StartSignalPulses(pulses, signalChannelBuffer)
	ForceExitOnRepeatedSignal(pulses, exit)
}

// unexported constants.
const (
	secondSignal        = 2  // Force exit on second signal
	signalChannelBuffer = 10 // Buffer size for signal + pulse channels
)
```

   Run `targ test` — expect green (unit tests pass; existing suite untouched; shim behavior identical through ForwardAsPulses).

3. [ ] Integration tests — relocate the cmd/engram adapter suites into internal `_test` files running the COMPOSED implementations over REAL os/syscall primitives (this preserves the ADR-0013 regression coverage the cmd tests carried; test files are exempt from the T-final-1 purity lint by design). Create `internal/cli/primitives_integration_test.go`:

```go
package cli_test

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// This file exercises the internally-composed EdgeFS/FileLocker/debug-sink
// implementations over REAL os/syscall primitives — the relocated
// cmd/engram adapter integration suite (#700 rework). realPrimitives()
// mirrors cmd/engram/main.go's Primitives literal (doctrine flag DRIFT:
// cli_test.go's end-to-end binary tests guard the production literal).

func TestRealDebugSink_AppendsAcrossOpens(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	path := filepath.Join(t.TempDir(), "debug.log")

	first := debugSinkAt(path)
	g.Expect(first).NotTo(gomega.BeNil())

	if first == nil {
		return
	}

	_, err := first.Write([]byte("line one\n"))
	g.Expect(err).NotTo(gomega.HaveOccurred())

	// Re-open the same path: append mode must preserve the first line —
	// the tail -F contract debuglog documents.
	second := debugSinkAt(path)
	g.Expect(second).NotTo(gomega.BeNil())

	if second == nil {
		return
	}

	_, err = second.Write([]byte("line two\n"))
	g.Expect(err).NotTo(gomega.HaveOccurred())

	contents, readErr := os.ReadFile(path)
	g.Expect(readErr).NotTo(gomega.HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(string(contents)).To(gomega.Equal("line one\nline two\n"))
}

func TestRealDebugSink_UnopenablePathYieldsNilSink(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	// Parent is a regular file, so opening a child path fails -> nil sink
	// (the CLI must run without debug logging rather than fail).
	dir := t.TempDir()
	blocked := filepath.Join(dir, "isfile")
	g.Expect(os.WriteFile(blocked, []byte("x"), realFSFilePerm)).To(gomega.Succeed())

	g.Expect(debugSinkAt(filepath.Join(blocked, "debug.log"))).To(gomega.BeNil())
}

func TestRealEdgeFS_MkdirAllStatReadDir(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	nested := filepath.Join(dir, "a", "b")
	fsys := realFSForTest()

	g.Expect(fsys.MkdirAll(nested, realFSDirPerm)).To(gomega.Succeed())

	info, err := fsys.Stat(nested)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(info.IsDir()).To(gomega.BeTrue())

	entries, readErr := fsys.ReadDir(filepath.Join(dir, "a"))
	g.Expect(readErr).NotTo(gomega.HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(entries).To(gomega.HaveLen(1))
	g.Expect(entries[0].Name()).To(gomega.Equal("b"))
}

func TestRealEdgeFS_MkdirTempAndWalkDir(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	fsys := realFSForTest()

	tmpDir, err := fsys.MkdirTemp(dir, "pat-*")
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(filepath.Base(tmpDir)).To(gomega.HavePrefix("pat-"))
	g.Expect(fsys.WriteFile(filepath.Join(tmpDir, "leaf.txt"), []byte("x"), realFSFilePerm)).To(gomega.Succeed())

	visited := make([]string, 0, 3)
	walkErr := fsys.WalkDir(dir, func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		visited = append(visited, path)

		return nil
	})
	g.Expect(walkErr).NotTo(gomega.HaveOccurred())
	g.Expect(visited).To(gomega.ContainElement(filepath.Join(tmpDir, "leaf.txt")))
}

func TestRealEdgeFS_ReadFileMissingSatisfiesErrNotExist(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	fsys := realFSForTest()

	_, err := fsys.ReadFile(filepath.Join(t.TempDir(), "missing.txt"))
	g.Expect(err).To(gomega.MatchError(fs.ErrNotExist))
}

func TestRealEdgeFS_ReadWriteRoundTrip(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	fsys := realFSForTest()

	g.Expect(fsys.WriteFile(path, []byte("hello"), realFSFilePerm)).To(gomega.Succeed())

	data, err := fsys.ReadFile(path)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(data)).To(gomega.Equal("hello"))
}

func TestRealEdgeFS_RenameRemoveRemoveAll(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	fsys := realFSForTest()

	oldPath := filepath.Join(dir, "old.txt")
	newPath := filepath.Join(dir, "new.txt")

	g.Expect(fsys.WriteFile(oldPath, []byte("x"), realFSFilePerm)).To(gomega.Succeed())
	g.Expect(fsys.Rename(oldPath, newPath)).To(gomega.Succeed())
	g.Expect(newPath).To(gomega.BeAnExistingFile())
	g.Expect(oldPath).NotTo(gomega.BeAnExistingFile())

	g.Expect(fsys.Remove(newPath)).To(gomega.Succeed())
	g.Expect(newPath).NotTo(gomega.BeAnExistingFile())

	sub := filepath.Join(dir, "sub")
	g.Expect(fsys.MkdirAll(sub, realFSDirPerm)).To(gomega.Succeed())
	g.Expect(fsys.WriteFile(filepath.Join(sub, "f"), []byte("x"), realFSFilePerm)).To(gomega.Succeed())
	g.Expect(fsys.RemoveAll(sub)).To(gomega.Succeed())
	g.Expect(sub).NotTo(gomega.BeADirectory())
}

func TestRealEdgeFS_WriteFileAtomicReplacesContentAndCleansTemp(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	fsys := realFSForTest()

	g.Expect(fsys.WriteFile(path, []byte("v1"), realFSFilePerm)).To(gomega.Succeed())
	g.Expect(fsys.WriteFileAtomic(path, []byte("v2"), realFSFilePerm)).To(gomega.Succeed())

	data, err := fsys.ReadFile(path)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(data)).To(gomega.Equal("v2"))

	entries, readErr := fsys.ReadDir(dir)
	g.Expect(readErr).NotTo(gomega.HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(entries).To(gomega.HaveLen(1), "temp files must be renamed or removed")
}

// TestRealEdgeFS_WriteFileAtomicPermsAreUmaskIndependent proves the restored
// Chmod step (P-4) makes WriteFileAtomic's final perm exact regardless of
// the process umask — parity with the pre-#700 dance.
func TestRealEdgeFS_WriteFileAtomicPermsAreUmaskIndependent(t *testing.T) {
	// serial: syscall.Umask is process-global; parallel file-creating tests would flake
	g := gomega.NewWithT(t)

	old := syscall.Umask(umaskParityRestrictiveMask)
	defer syscall.Umask(old)

	dir := t.TempDir()
	target := filepath.Join(dir, "note.md")
	fsys := realFSForTest()

	g.Expect(fsys.WriteFileAtomic(target, []byte("v1"), umaskParityPerm)).To(gomega.Succeed())

	info, err := os.Stat(target)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(info.Mode().Perm()).To(gomega.Equal(umaskParityPerm))
}

// TestRealFlockLocker_SecondLockWaitsForUnlock is the relocated ADR-0013
// lock regression guard: a second locker on the same path must block until
// the first unlocks — never proceed concurrently, never fail.
func TestRealFlockLocker_SecondLockWaitsForUnlock(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	lockPath := filepath.Join(t.TempDir(), "test.lock")
	locker := realDepsForTest().Lock

	unlock, err := locker.Lock(lockPath)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	acquired := make(chan struct{})

	go func() {
		secondUnlock, secondErr := locker.Lock(lockPath)
		if secondErr == nil {
			_ = secondUnlock()
		}

		close(acquired)
	}()

	const holdWindow = 100 * time.Millisecond

	select {
	case <-acquired:
		t.Fatal("second locker acquired while first still held the lock")
	case <-time.After(holdWindow):
		// good — second locker is blocked while the lock is held
	}

	g.Expect(unlock()).To(gomega.Succeed())

	const releaseTimeout = 2 * time.Second

	select {
	case <-acquired:
		// good — released lock was acquired
	case <-time.After(releaseTimeout):
		t.Fatal("second locker did not acquire after unlock")
	}
}

// debugSinkAt composes a real debug sink for path via NewDeps (Getenv fake
// points ENGRAM_DEBUG_LOG at path; the open primitive is real).
func debugSinkAt(path string) io.Writer {
	prims := realPrimitives()
	prims.Getenv = func(key string) string {
		if key == "ENGRAM_DEBUG_LOG" {
			return path
		}

		return ""
	}

	return cli.NewDeps(prims, io.Discard, io.Discard, func(int) {}).DebugLog
}

// realDepsForTest composes production Deps over real OS primitives.
func realDepsForTest() cli.Deps {
	return cli.NewDeps(realPrimitives(), io.Discard, io.Discard, func(int) {})
}

// realFSForTest composes the production EdgeFS over real OS primitives.
func realFSForTest() cli.EdgeFS {
	return realDepsForTest().FS
}

// realPrimitives mirrors cmd/engram/main.go's production Primitives literal
// (minus the signal starter — tests must not subscribe process signals).
func realPrimitives() cli.Primitives {
	return cli.Primitives{
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
			//nolint:gosec // test helper, path from test
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
			return os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, perm)
		},
	}
}

// unexported constants.
const (
	realFSDirPerm  fs.FileMode = 0o750
	realFSFilePerm fs.FileMode = 0o600

	// umaskParityPerm is the target perm for the umask-independence parity
	// test (P-4 restored chmod step).
	umaskParityPerm fs.FileMode = 0o644

	// umaskParityRestrictiveMask is a deliberately restrictive umask (would
	// mask 0o644 down to 0o600 without the explicit chmod step).
	umaskParityRestrictiveMask = 0o077
)
```

   And `internal/cli/signal_integration_test.go` (relocated real-signal test):

```go
package cli_test

import (
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

// TestForceExit_RealSignalDeliveryThroughForwardAsPulses replicates
// cmd/engram/main.go's StartSignalPulses closure with SIGUSR2 (registered
// only by this test; Notify overrides the default terminate disposition, so
// the test run is safe) and proves the full chain: real signal -> Notify ->
// ForwardAsPulses -> ForceExitOnRepeatedSignal. Pending same-signal
// coalescing can swallow a rapid second delivery, so the test keeps
// signalling (paced) until exit fires — over-delivery still means "force
// exit", so this can never false-pass.
func TestForceExit_RealSignalDeliveryThroughForwardAsPulses(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	const buffer = 10

	exitCodes := make(chan int, 1)

	sigCh := make(chan os.Signal, buffer)
	signal.Notify(sigCh, syscall.SIGUSR2)

	pulses := make(chan struct{}, buffer)

	go cli.ForwardAsPulses(sigCh, pulses)
	cli.ForceExitOnRepeatedSignal(pulses, func(code int) { exitCodes <- code })

	proc, err := os.FindProcess(os.Getpid())
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	const (
		pace    = 20 * time.Millisecond
		maxWait = 5 * time.Second
	)

	deadline := time.After(maxWait)

	for {
		g.Expect(proc.Signal(syscall.SIGUSR2)).To(gomega.Succeed())

		select {
		case code := <-exitCodes:
			g.Expect(code).To(gomega.Equal(cli.ExitCodeSigInt))

			return
		case <-deadline:
			t.Fatal("force-exit not fired within timeout")
		case <-time.After(pace):
		}
	}
}
```

   Run `targ test` — expect green (the relocated suites pass against the composed implementations).

4. [ ] Thin cmd — delete all six cmd/engram adapter files (their logic now lives in internal/cli; their tests were relocated in steps 1 and 3):

```
git rm cmd/engram/os_fs.go cmd/engram/os_fs_test.go \
       cmd/engram/os_signal.go cmd/engram/os_signal_test.go \
       cmd/engram/debuglog_sink.go cmd/engram/debuglog_sink_test.go
```

   `cmd/engram/main.go` is untouched — still the single-statement `cli.Main(os.Stdout, os.Stderr, os.Exit)` (T2 rewrites it over NewDeps). Run `targ test` — expect green (cli_test.go's end-to-end binary tests still build and run the real binary).

5. [ ] Gate — run `targ check-thin-api`. Expect PASS: `All N public API files are thin wrappers.` — cmd/engram now contains only main.go (one single-external-call statement, zero other declarations). If ANY finding remains, escalate the exact finding to the orchestrator (do not suppress, do not restructure ad hoc — doctrine flag SIG-1 documents the checker's exact rules). Then run `targ check-full` — the ONLY tolerated new-vs-baseline residual is the KNOWN `SetupSignalHandling` 0% function-coverage gap (T1-report concern #1; resolved when T2 deletes the shim). Run `targ reorder-decls` if `reorder-decls-check` flags the new files (revert any out-of-scope `dev/eval/**/testdata` touches before committing, as the landed T1 did). Any OTHER new failure: fix before commit.

6. [ ] Commit:

```
refactor(cli): compose I/O adapters internally from Primitives (#700)

Rework of the landed T1 (de484526): targ check-thin-api rejected real
adapter logic in cmd/engram (9 non-thin declarations). EdgeFS wrapping +
the ADR-0013 atomic-write dance, the flock lifecycle, and the syncing
debug sink move behind internal/cli composition (cli.Primitives +
cli.NewDeps), unit-tested with fake primitives and integration-tested
with real os/syscall funcs in internal _test files. Adds generic
ForwardAsPulses; deletes all six cmd/engram adapter files. cmd keeps its
single-statement main until T2 wires NewDeps.

AI-Used: [claude]
```

---

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

   Expected: no `os.` CODE references; exactly these residual COMMENT-only hits are correct and must NOT be scrubbed: targets.go:~55 and :~66 (`pass os.Getenv in production` doc comments, preserved by step 3), signal.go ForceExitOnRepeatedSignal doc (`cmd/engram adapts real os.Signal deliveries`), signal.go ForwardAsPulses doc (`can feed a chan os.Signal without os.Signal entering`). debuglog.go: zero hits of any kind. Any OTHER hit (or any hit outside a comment) is a real failure — STOP; no `SetupSignalHandling` anywhere; no `time.Now`/`time.Since` in debuglog.go; `ls` errors (file deleted). Also verify the FIXME survived relocation: `rg "FIXME\(#700\)" cmd/engram/main.go` → exactly one hit (R8).

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

### Task T3 (L1): learn-family — compose LearnDeps/LearnQADeps purely from cli.Deps

**Files**
- Create: `internal/cli/deps_compose.go` (shared EdgeFS/FileLocker composition helpers)
- Create: `internal/cli/deps_compose_test.go` (RED tests for the compositions)
- Modify: `internal/cli/deps.go` (foundation file: add `WriteFileExcl` to EdgeFS — flagged addition)
- Modify: `internal/cli/edgefs.go` (add `primFS.WriteFileExcl` — the single `%w` wrap over the base `WriteFileExcl` primitive)
- Modify: `internal/cli/edgefs_test.go` (fake-primitive contract tests for `WriteFileExcl`)
- Modify: `internal/cli/primitives_integration_test.go` (real-FS `WriteFileExcl` → `fs.ErrExist` round-trip — survivor S-1's behavior-mirror test; `realPrimitives()` already carries the `WriteFileExcl` closure from T1-rework)
- Verify-only (NO edit — flag X-1 is consume/verify): `internal/cli/primitives.go`, `cmd/engram/main.go` (the `WriteFileExcl` primitive and its literal closure landed with T1-rework/T2)
- Modify: `internal/cli/learn.go` (delete `newOsLearnDeps` + `logWarningToStderrf`; add `newLearnDeps(d Deps)`; re-sign `runLearnFrom*Args`; drop `os` import)
- Modify: `internal/cli/qa.go` (delete `newOsLearnQADeps`; add `newQaDeps(d Deps)`; drop `os` import)
- Modify: `internal/cli/cli.go` (re-parameterize `listRootNotes`; shrink `osLearnFS` to Lock-only; receive relocated `logWarningToStderrf`)
- Modify: `internal/cli/targets.go` (learn-group closures only)
- Modify: `internal/cli/targets_test.go` (extend `newTestDeps` with `FS`/`Lock`, `Embed` forced nil — R11; this task's closure flip makes the executed learn tests dereference both; per Gate B #700 T3 review finding 2, `newTestDeps` is the single real-Deps test builder, composed over T1-rework's cli_test helpers — `testhelpers_test.go` gets NO separate `realFSDepsForTest`, see step 1 and the NOTE(R11-naming) in step 7)
- Modify: `internal/cli/export_test.go` (re-sign fact/feedback exports; add `ExportNewLearnDeps`/`ExportNewQaDeps`; drop `ExportNewOsLearnFS` uses of deleted methods)
- Modify: `internal/cli/invariants_k1_property_test.go` (K1 drives production `newLearnDeps` over the composed primFS/primLocker with real primitives)
- Modify: `internal/cli/learn_adapters_test.go` (delete `TestOsLearnFS_*` except Lock test; thread Deps into `ExportRunLearnFrom*Args` calls)

**Interfaces**
- Consumes (from foundation `internal/cli/deps.go`): `Deps{Stdout, Stderr io.Writer; Now func() time.Time; Getenv func(string) string; FS EdgeFS; Lock FileLocker; Embed embed.Embedder; ...}`, `EdgeFS`, `FileLocker{ Lock(path string) (unlock func() error, err error) }`
- Consumes (from T1-rework): `cli.Primitives` + `cli.NewDeps` (the base struct already carries the `WriteFileExcl` primitive — flag X-1 is consume/verify, doctrine survivor S-1), `primFS`/`primLocker` (unexported, reached only through `NewDeps`), and the cli_test helpers `realPrimitives()`, `realDepsForTest()`, `realFSForTest()`, `fsFromPrims`, `lockerFromPrims`, const `atomicPerm`/`realFSFilePerm`
- Produces:
  - `func newLearnDeps(d Deps) LearnDeps`
  - `func newQaDeps(d Deps) LearnQADeps`
  - `func runLearnFromFactArgs(ctx context.Context, a LearnFactArgs, d Deps, stdout io.Writer) error`
  - `func runLearnFromFeedbackArgs(ctx context.Context, a LearnFeedbackArgs, d Deps, stdout io.Writer) error`
  - Shared helpers: `statDirFromFS(fsys EdgeFS) func(string) error`, `initVaultFromFS(fsys EdgeFS) func(string) error`, `listIDsFromFS`, `listBasenamesFromFS`, `listMDFromFS(fsys EdgeFS) func(string) ([]string, error)`, `vaultLockFromLocker(locker FileLocker) func(string) (func(), error)`, `writeNewFromFS`, `writeAtomicFromFS(fsys EdgeFS, perm fs.FileMode, opName string) func(string, []byte) error`, `logWarningTo(w io.Writer) func(string, ...any)`
  - EdgeFS addition (flagged): `WriteFileExcl(path string, data []byte, perm fs.FileMode) error`
  - `primFS.WriteFileExcl` (flag X-1 resolution: the single internal `%w` wrap over T1-rework's `WriteFileExcl` primitive — NO new primitive, NO cmd edit)

**Steps**

- [ ] 1. **RED — composed test Deps + contract tests.** NO os-backed doubles are declared in this task — the composition doctrine forbids hand-rolled adapter mirrors. The FS/Lock capabilities for every test below are the PRODUCTION `primFS`/`primLocker` compositions, reached through T1-rework's cli_test helpers (`realDepsForTest()` = `cli.NewDeps(realPrimitives(), io.Discard, io.Discard, func(int) {})`). SUPERSEDED (Gate B #700 T3 review finding 2 — "three overlapping real-Deps test builders"): do NOT declare a separate `realFSDepsForTest` in `internal/cli/testhelpers_test.go`. Instead use `newTestDeps(io.Discard, io.Discard)` — step 7 below is amended to extend `newTestDeps` (targets_test.go, package `cli_test`) with `FS`/`Lock`/`Embed`-nil FIRST, in this task, so it becomes the single real-Deps builder every family cluster (this task included) consumes; `testhelpers_test.go` gets no new declaration:

```go
// newTestDeps builds a cli.Deps wired to real OS capabilities with captured
// stdout/stderr — the test analog of the production cli.NewDeps composition,
// built over realDepsForTest (T1-rework's primitives_integration_test.go
// helper) with Stdout/Stderr swapped in and Embed forced nil (unit tests
// must not load the bundled model — the embed-on-write path stays covered
// by cli_test.go's real-binary end-to-end test). No signal registration
// occurs: realPrimitives() omits StartSignalPulses, so startForceExit
// nil-skips (doctrine flag SIG-1).
func newTestDeps(stdout, stderr io.Writer) cli.Deps {
	d := realDepsForTest()
	d.Stdout = stdout
	d.Stderr = stderr
	d.Embed = nil

	return d
}
```

Add to `internal/cli/edgefs_test.go` (package `cli_test` — T1-rework's fake-primitive EdgeFS suite; reuses its `fsFromPrims` helper, `atomicPerm` const, and existing imports) the `WriteFileExcl` contract tests. These fail to compile until step 2 lands the `primFS.WriteFileExcl` method — `WriteFileExcl` is not yet in `cli.EdgeFS` (RED):

```go
func TestEdgeFS_WriteFileExclPreservesErrExistAndAddsPath(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	fsys := fsFromPrims(cli.Primitives{
		WriteFileExcl: func(path string, _ []byte, _ fs.FileMode) error {
			return &fs.PathError{Op: "open", Path: path, Err: fs.ErrExist}
		},
	})

	err := fsys.WriteFileExcl("existing.md", []byte("x"), atomicPerm)
	g.Expect(err).To(gomega.MatchError(fs.ErrExist),
		"K1 backstop: errors.Is(err, fs.ErrExist) must survive the internal wrap")
	g.Expect(err.Error()).To(gomega.ContainSubstring("existing.md"), "wrap must add path context")
}

func TestEdgeFS_WriteFileExclPassesDataAndPermToPrimitive(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	var (
		gotPath string
		gotData []byte
		gotPerm fs.FileMode
	)

	fsys := fsFromPrims(cli.Primitives{
		WriteFileExcl: func(path string, data []byte, perm fs.FileMode) error {
			gotPath = path
			gotData = append([]byte(nil), data...)
			gotPerm = perm

			return nil
		},
	})

	g.Expect(fsys.WriteFileExcl("new.md", []byte("body"), atomicPerm)).To(gomega.Succeed())
	g.Expect(gotPath).To(gomega.Equal("new.md"))
	g.Expect(string(gotData)).To(gomega.Equal("body"))
	g.Expect(gotPerm).To(gomega.Equal(atomicPerm),
		"the caller's perm must reach the raw primitive unchanged")
}
```

Add to `internal/cli/primitives_integration_test.go` (package `cli_test`; existing `io/fs`/`path/filepath` imports suffice) the real-primitive round-trip — survivor S-1's named behavior-mirror test; `realPrimitives()` already carries the real `WriteFileExcl` closure (T1-rework), so this is RED only until step 2 lands the EdgeFS method:

```go
func TestRealEdgeFS_WriteFileExclRefusesExistingFile(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	path := filepath.Join(t.TempDir(), "note.md")
	fsys := realFSForTest()

	g.Expect(fsys.WriteFileExcl(path, []byte("first"), realFSFilePerm)).To(gomega.Succeed())

	err := fsys.WriteFileExcl(path, []byte("second"), realFSFilePerm)
	g.Expect(err).To(gomega.MatchError(fs.ErrExist), "O_EXCL contract: existing path must satisfy fs.ErrExist")

	data, readErr := fsys.ReadFile(path)
	g.Expect(readErr).NotTo(gomega.HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(string(data)).To(gomega.Equal("first"), "the losing writer must not clobber the existing note")
}
```

Then create `internal/cli/deps_compose_test.go` (package `cli_test`) — these fail to compile until step 3 (RED):

```go
package cli_test

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

func TestNewLearnDeps_StatDir_MissingReturnsErrNotExist(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := cli.ExportNewLearnDeps(newTestDeps(io.Discard, io.Discard))

	err := deps.StatDir(filepath.Join(t.TempDir(), "absent"))
	g.Expect(errors.Is(err, fs.ErrNotExist)).To(BeTrue())
}

func TestNewLearnDeps_StatDir_FileIsNotADirectory(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	path := filepath.Join(t.TempDir(), "file.txt")
	g.Expect(os.WriteFile(path, []byte("x"), 0o600)).To(Succeed())

	deps := cli.ExportNewLearnDeps(newTestDeps(io.Discard, io.Discard))
	g.Expect(deps.StatDir(path)).To(MatchError(ContainSubstring("not a directory")))
}

func TestNewLearnDeps_WriteNew_PreservesErrExist(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	path := filepath.Join(t.TempDir(), "existing.md")
	g.Expect(os.WriteFile(path, []byte("already"), 0o600)).To(Succeed())

	deps := cli.ExportNewLearnDeps(newTestDeps(io.Discard, io.Discard))
	err := deps.WriteNew(path, []byte("nope"))
	g.Expect(errors.Is(err, fs.ErrExist)).To(BeTrue(), "O_EXCL backstop must survive composition")

	got, readErr := os.ReadFile(path)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(got)).To(Equal("already"))
}

func TestNewLearnDeps_InitVault_IdempotentAndPreservesEdits(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := filepath.Join(t.TempDir(), "vault")
	deps := cli.ExportNewLearnDeps(newTestDeps(io.Discard, io.Discard))

	g.Expect(deps.InitVault(vault)).To(Succeed())

	readme := filepath.Join(vault, "README.md")
	g.Expect(os.WriteFile(readme, []byte("user edit"), 0o644)).To(Succeed())

	g.Expect(deps.InitVault(vault)).To(Succeed())

	got, readErr := os.ReadFile(readme)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(got)).To(Equal("user edit"), "re-init must not clobber existing files")
}

func TestNewLearnDeps_ListIDs_MissingVaultIsEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := cli.ExportNewLearnDeps(newTestDeps(io.Discard, io.Discard))
	got, err := deps.ListIDs(filepath.Join(t.TempDir(), "absent"))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(got).To(BeEmpty())
}

func TestNewLearnDeps_ListBasenames_SkipsSubdirsAndNonLuhmann(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	g.Expect(os.MkdirAll(filepath.Join(vault, "MOCs"), 0o700)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(vault, "1.2026-05-09.foo.md"), nil, 0o600)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(vault, "README.md"), nil, 0o600)).To(Succeed())

	deps := cli.ExportNewLearnDeps(newTestDeps(io.Discard, io.Discard))
	got, err := deps.ListBasenames(vault)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(got).To(ConsistOf("1.2026-05-09.foo"))
}

func TestNewLearnDeps_Lock_AcquiresVaultLuhmannLockFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()

	deps := cli.ExportNewLearnDeps(newTestDeps(io.Discard, io.Discard))
	release, err := deps.Lock(vault)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	release()

	_, statErr := os.Stat(filepath.Join(vault, ".luhmann.lock"))
	g.Expect(statErr).NotTo(HaveOccurred(), "lock must live at vault/.luhmann.lock")
}

func TestNewLearnDeps_LogWarning_WritesToDepsStderr(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var stderr bytes.Buffer

	d := newTestDeps(io.Discard, io.Discard)
	d.Stderr = &stderr

	deps := cli.ExportNewLearnDeps(d)
	deps.LogWarning("hello %s", "world")

	g.Expect(stderr.String()).To(Equal("warning: hello world\n"))
}

func TestNewQaDeps_WiresRemoveAndReadThroughFS(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "f.txt")
	g.Expect(os.WriteFile(path, []byte("data"), 0o600)).To(Succeed())

	deps := cli.ExportNewQaDeps(newTestDeps(io.Discard, io.Discard))

	got, readErr := deps.ReadFile(path)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(got)).To(Equal("data"))

	g.Expect(deps.RemoveFile(path)).To(Succeed())
	_, statErr := os.Stat(path)
	g.Expect(errors.Is(statErr, fs.ErrNotExist)).To(BeTrue())
}
```

Run: `targ test` → expected FAIL (compile errors: `WriteFileExcl` not in `cli.EdgeFS`; `ExportNewLearnDeps`/`ExportNewQaDeps` undefined). This is the RED.

- [ ] 2. **GREEN (flag X-1) — `EdgeFS.WriteFileExcl`, the internal wrap over the base exclusive-create primitive.** X-1 resolution (recorded): there is NO new primitive and NO cmd edit — T1-rework's base `Primitives` already carries `WriteFileExcl` (doctrine survivor S-1: the stdlib-equivalent exclusive-create closure that also backs the atomic dance, P-4), T2's literal already wires it, and `realPrimitives()` already mirrors it. This task CONSUMES it: the EdgeFS method adds the single internal `%w` wrap (path context; the raw `*fs.PathError` keeps `errors.Is(err, fs.ErrExist)` alive). NO cmd/engram adapter file exists or is created — the supersession map re-points every "cmd/engram os_fs.go `osFS`" obligation to internal/cli/edgefs.go. Two sub-edits + one verify, one commit:

   **2a.** In the foundation's `internal/cli/deps.go`, add to `EdgeFS`:

```go
	// WriteFileExcl creates path exclusively (O_CREATE|O_EXCL semantics): it
	// errors with an error satisfying errors.Is(err, fs.ErrExist) when path
	// already exists. The learn family's ID-collision backstop (ADR-0013 K1)
	// and idempotent vault bootstrap both require exclusive create.
	WriteFileExcl(path string, data []byte, perm fs.FileMode) error
```

   **2b.** In `internal/cli/edgefs.go`, add to `primFS`:

```go
// WriteFileExcl creates path exclusively via the base WriteFileExcl
// primitive (O_CREATE|O_EXCL — the ADR-0013 K1 collision backstop). The raw
// primitive error is wrapped exactly once here, preserving the fs.ErrExist
// chain (doctrine flags X-1/S-1: the exclusive create itself is the
// enumerated stdlib-equivalent cmd primitive; only the wrap lives here).
func (p primFS) WriteFileExcl(path string, data []byte, perm fs.FileMode) error {
	err := p.prims.WriteFileExcl(path, data, perm)
	if err != nil {
		return fmt.Errorf("write excl %s: %w", path, err)
	}

	return nil
}
```

   **2c.** Verify the base primitive is in place (consume, never re-declare): `rg -n "WriteFileExcl" internal/cli/primitives.go internal/cli/primitives_integration_test.go cmd/engram/main.go` → the struct field + its doc group (T1-rework), the `realPrimitives()` closure (T1-rework), and the cmd literal closure (T2) all present. Any miss → an upstream task is incomplete; STOP and escalate — do NOT add the missing piece here.

Run `targ test` → the step-1 `WriteFileExcl` contract + integration tests go green; the deps_compose tests stay RED until steps 3-8 land.

- [ ] 3. **GREEN — create `internal/cli/deps_compose.go`** (package `cli`; imports `errors`, `fmt`, `io`, `io/fs`, `path/filepath`, `strings` only — no `os`, no `syscall`, no `time`):

```go
package cli

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
)

// unexported constants.
const (
	notePerm    fs.FileMode = 0o600
	sidecarPerm fs.FileMode = 0o600
)

// initVaultFromFS returns an InitVault func composed over the injected EdgeFS.
func initVaultFromFS(fsys EdgeFS) func(string) error {
	return func(path string) error { return initializeVault(edgeVaultInitFS{fsys: fsys}, path) }
}

// edgeVaultInitFS adapts EdgeFS to the VaultInitFS bootstrap surface.
// WriteFileIfMissing swallows fs.ErrExist so re-initialization is idempotent
// and never clobbers user-edited starter files.
type edgeVaultInitFS struct{ fsys EdgeFS }

func (e edgeVaultInitFS) MkdirAll(path string, perm fs.FileMode) error {
	err := e.fsys.MkdirAll(path, perm)
	if err != nil {
		return fmt.Errorf("init vault mkdir: %w", err)
	}

	return nil
}

func (e edgeVaultInitFS) WriteFileIfMissing(path string, data []byte, perm fs.FileMode) error {
	err := e.fsys.WriteFileExcl(path, data, perm)
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			return nil
		}

		return fmt.Errorf("write if missing: %w", err)
	}

	return nil
}

// listBasenamesFromFS returns a ListBasenames func: note basenames (filename
// minus .md) for luhmann-id notes at the flat vault root (D1).
func listBasenamesFromFS(fsys EdgeFS) func(string) ([]string, error) {
	return func(vault string) ([]string, error) {
		return listRootNotes(fsys.ReadDir, vault, func(name string) (string, bool) {
			if _, ok := extractLuhmannFromFilename(name); !ok {
				return "", false
			}

			return strings.TrimSuffix(name, ".md"), true
		})
	}
}

// listIDsFromFS returns a ListIDs func: Luhmann IDs from .md filenames at the
// flat vault root.
func listIDsFromFS(fsys EdgeFS) func(string) ([]string, error) {
	return func(vault string) ([]string, error) {
		return listRootNotes(fsys.ReadDir, vault, extractLuhmannFromFilename)
	}
}

// listMDFromFS returns a ListMD func with osVaultFS.ListMD semantics: the .md
// filenames directly inside dir; a missing dir yields (nil, nil).
func listMDFromFS(fsys EdgeFS) func(string) ([]string, error) {
	return func(dir string) ([]string, error) {
		entries, err := fsys.ReadDir(dir)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil, nil
			}

			return nil, fmt.Errorf("list md: %w", err)
		}

		out := make([]string, 0, len(entries))

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}

			out = append(out, entry.Name())
		}

		return out, nil
	}
}

// logWarningTo returns the production LogWarning hook writing to w — the
// Deps-threaded replacement for the old os.Stderr-bound logWarningToStderrf.
func logWarningTo(w io.Writer) func(string, ...any) {
	return func(format string, args ...any) {
		_, _ = fmt.Fprintf(w, "warning: "+format+"\n", args...)
	}
}

// statDirFromFS returns a StatDir func: fs.ErrNotExist when the directory is
// missing, errNotADirectory when the path is a file, wrapped error otherwise.
func statDirFromFS(fsys EdgeFS) func(string) error {
	return func(path string) error {
		info, err := fsys.Stat(path)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return fs.ErrNotExist
			}

			return fmt.Errorf("vault stat: %w", err)
		}

		if !info.IsDir() {
			return fmt.Errorf("%w: %s", errNotADirectory, path)
		}

		return nil
	}
}

// vaultLockFromLocker returns a vault-lock func over the injected FileLocker:
// an exclusive flock on vault/.luhmann.lock (ADR-0013). The locker's
// unlock-with-error is adapted to the deps structs' plain release func.
func vaultLockFromLocker(locker FileLocker) func(string) (func(), error) {
	return func(vault string) (func(), error) {
		unlock, err := locker.Lock(filepath.Join(vault, luhmannLockFile))
		if err != nil {
			return nil, fmt.Errorf("lock vault: %w", err)
		}

		return func() { _ = unlock() }, nil
	}
}

// writeNewFromFS returns a WriteNew func: exclusive create, preserving
// errors.Is(err, fs.ErrExist) — the K1 collision backstop under the vault lock.
func writeNewFromFS(fsys EdgeFS) func(string, []byte) error {
	return func(path string, data []byte) error {
		err := fsys.WriteFileExcl(path, data, notePerm)
		if err != nil {
			return fmt.Errorf("write new: %w", err)
		}

		return nil
	}
}

// writeAtomicFromFS returns an atomic-rewrite func at the given perm (temp+
// rename via EdgeFS.WriteFileAtomic — ADR-0013's atomic-rename edge). opName
// labels the wrapped error (e.g. "write note", "write sidecar") — the single
// atomic-write composition shared by the note and sidecar call sites (Gate B
// #700 T3 review finding 3: writeNoteAtomicFromFS/writeSidecarFromFS collapsed
// into this one helper).
func writeAtomicFromFS(fsys EdgeFS, perm fs.FileMode, opName string) func(string, []byte) error {
	return func(path string, data []byte) error {
		err := fsys.WriteFileAtomic(path, data, perm)
		if err != nil {
			return fmt.Errorf("%s: %w", opName, err)
		}

		return nil
	}
}
```

- [ ] 4. **cli.go: re-parameterize `listRootNotes`; shrink `osLearnFS` to Lock-only; receive `logWarningToStderrf`.** Replace cli.go:194-221 (`listRootNotes` — current signature `listRootNotes(vault string, extract func(name string) (string, bool))` reading `os.ReadDir(vault)` and checking `os.IsNotExist(err)`) with the injected-ReadDir form (this stays in cli.go, now pure):

```go
// listRootNotes reads the flat vault root via the injected readDir and
// collects one string per non-dir entry for which extract returns ok. A
// missing vault is treated as empty. Shared by listIDsFromFS and
// listBasenamesFromFS so the flat-root traversal lives in exactly one place.
func listRootNotes(
	readDir func(string) ([]fs.DirEntry, error),
	vault string,
	extract func(name string) (string, bool),
) ([]string, error) {
	out := []string{}

	entries, err := readDir(vault)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return out, nil
		}

		return nil, fmt.Errorf("read vault root: %w", err)
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		if s, ok := extract(e.Name()); ok {
			out = append(out, s)
		}
	}

	return out, nil
}
```

Delete `osLearnFS` methods `ListBasenames` (cli.go:39-47), `ListIDs` (49-52), `MkdirAll` (59-67), `StatDir` (69-86), `WriteFileIfMissing` (88-112), `WriteNew` (114-135), `WriteSidecar` (137-150). Keep — verbatim, temporarily, until Task L2 — `osLearnFS` with only its `Lock` method (cli.go:54-57), `flockPath` (165-192), `osManifestLock` (223-236), `osFileReader` (27-31), and `acquireOptionalLock`/`pathOf`/constants/vars. Append the relocated production stderr hook (moved from learn.go:330-334, still needed by activate/amend/resituate/vocab until their migrations):

```go
// logWarningToStderrf is the transitional os.Stderr-bound LogWarning hook.
// Deps-migrated constructors use logWarningTo(d.Stderr) instead; this stays
// only for the not-yet-migrated constructors and dies in the cli.go purge task.
func logWarningToStderrf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, "warning: "+format+"\n", args...)
}
```

- [ ] 5. **learn.go: pure constructor + threaded wrappers.** Delete `logWarningToStderrf` (learn.go:330-334, relocated in step 4). Replace `newOsLearnDeps` (learn.go:347-378) with:

```go
// newLearnDeps composes LearnDeps purely from the injected edge capabilities.
// All I/O flows through d.FS / d.Lock / d.Embed / d.Stderr — no direct os use.
func newLearnDeps(d Deps) LearnDeps {
	return LearnDeps{
		Now:           d.Now,
		Getenv:        d.Getenv,
		StatDir:       statDirFromFS(d.FS),
		InitVault:     initVaultFromFS(d.FS),
		ListIDs:       listIDsFromFS(d.FS),
		ListBasenames: listBasenamesFromFS(d.FS),
		Lock:          vaultLockFromLocker(d.Lock),
		WriteNew:      writeNewFromFS(d.FS),
		Embedder:      d.Embed,
		WriteSidecar:  writeAtomicFromFS(d.FS, sidecarPerm, "write sidecar"),
		LogWarning:    logWarningTo(d.Stderr),
		// Vocab assignment wiring: no-op when the vault has no term notes.
		// Uses stored member centroids (vocab.centroids.json) when present,
		// falling back to description embeddings per term.
		LoadTermVectors: func(vault string) ([]TermWithVector, error) {
			return loadAssignmentTermVectors(vault, listMDFromFS(d.FS), d.FS.ReadFile)
		},
		ReadSidecar: d.FS.ReadFile,
		WriteNote:   writeAtomicFromFS(d.FS, vocabNotePerm, "write note"),
		// ListMD provides full .md filenames for the vocab trigger scan.
		// Must be full filenames (not stripped basenames) — ListBasenames
		// filters to Luhmann IDs, causing 100% false-fire on the untagged trigger.
		ListMD: listMDFromFS(d.FS),
	}
}
```

Re-sign the two wrappers (current learn.go:497-541, `deps := newOsLearnDeps()`):

```go
func runLearnFromFactArgs(ctx context.Context, a LearnFactArgs, d Deps, stdout io.Writer) error {
	return runLearn(ctx, LearnArgs{
		Type:         typeFact,
		Slug:         a.Slug,
		Vault:        a.Vault,
		Target:       a.Target,
		Position:     a.Position,
		Source:       a.Source,
		Project:      a.Project,
		Issue:        a.Issue,
		Tier:         a.Tier,
		Supersedes:   a.Supersedes,
		ChunkSources: a.ChunkSources,
		Tags:         a.Tags,
		Situation:    a.Situation,
		Subject:      a.Subject,
		Predicate:    a.Predicate,
		Object:       a.Object,
	}, newLearnDeps(d), stdout)
}

func runLearnFromFeedbackArgs(ctx context.Context, a LearnFeedbackArgs, d Deps, stdout io.Writer) error {
	return runLearn(ctx, LearnArgs{
		Type:         typeFeedback,
		Slug:         a.Slug,
		Vault:        a.Vault,
		Target:       a.Target,
		Position:     a.Position,
		Source:       a.Source,
		Project:      a.Project,
		Issue:        a.Issue,
		Tier:         a.Tier,
		Supersedes:   a.Supersedes,
		ChunkSources: a.ChunkSources,
		Tags:         a.Tags,
		Situation:    a.Situation,
		Behavior:     a.Behavior,
		Impact:       a.Impact,
		Action:       a.Action,
	}, newLearnDeps(d), stdout)
}
```

Remove `"os"` from learn.go's import block (learn.go:8 — after these edits nothing in the file references `os`; `time` stays for `time.Time` types only, no `time.Now` reference remains).

- [ ] 6. **qa.go: pure constructor.** Replace `newOsLearnQADeps` (qa.go:260-286) with:

```go
// newQaDeps composes LearnQADeps purely from the injected edge capabilities.
func newQaDeps(d Deps) LearnQADeps {
	return LearnQADeps{
		Now:          d.Now,
		Getenv:       d.Getenv,
		StatDir:      statDirFromFS(d.FS),
		InitVault:    initVaultFromFS(d.FS),
		ListMD:       listMDFromFS(d.FS),
		Lock:         vaultLockFromLocker(d.Lock),
		WriteNew:     writeNewFromFS(d.FS),
		RemoveFile:   d.FS.Remove,
		ReadFile:     d.FS.ReadFile,
		Embedder:     d.Embed,
		WriteSidecar: writeAtomicFromFS(d.FS, sidecarPerm, "write sidecar"),
		LogWarning:   logWarningTo(d.Stderr),
		LoadTermVectors: func(vault string) ([]TermWithVector, error) {
			return loadAssignmentTermVectors(vault, listMDFromFS(d.FS), d.FS.ReadFile)
		},
		ReadSidecar: d.FS.ReadFile,
		WriteNote:   writeAtomicFromFS(d.FS, vocabNotePerm, "write note"),
	}
}
```

Remove `"os"` and `"time"`?—no: keep `"time"` (`time.Time` in signatures); remove `"os"` from qa.go's imports (qa.go:9 — `os.Getenv`/`os.Remove` were its only uses).

- [ ] 7. **targets.go: thread Deps through the learn group.** In `learnUpdateTargets` (targets.go:198-234; foundation task threads `d Deps` into this function's signature — coordinate), replace the three learn closures (current lines 206-218):

```go
		targ.Group("learn",
			targ.Targ(func(ctx context.Context, a LearnFeedbackArgs) {
				a.Vault = resolveVault(a.Vault, home, os.Getenv)
				errHandler(runLearnFromFeedbackArgs(withLog(ctx), a, d, stdout))
			}).Name("feedback").Description("Write a feedback note to the vault"),
			targ.Targ(func(ctx context.Context, a LearnFactArgs) {
				a.Vault = resolveVault(a.Vault, home, os.Getenv)
				errHandler(runLearnFromFactArgs(withLog(ctx), a, d, stdout))
			}).Name("fact").Description("Write a fact note to the vault"),
			targ.Targ(func(ctx context.Context, a LearnQAArgs) {
				a.Vault = resolveVault(a.Vault, home, os.Getenv)
				errHandler(RunLearnQA(withLog(ctx), a, newQaDeps(d), stdout))
			}).Name("qa").Description("Write a QA pair (Q+A notes) to the vault"),
		),
```

(The `resolveVault(a.Vault, home, os.Getenv)` → `d.Getenv`/`d.UserHomeDir` swap is the targets cluster's charge — leave those expressions to that task.)

In the same commit, extend `newTestDeps` in `internal/cli/targets_test.go` (R11 — this task owns the FS/Lock extension because its closure flip is the FIRST to make an executed targets-level test dereference them: the learn feedback/fact/qa tests at targets_test.go:206-263 run through `executeForTest`, and the qa test asserts real note files on disk; nil `FS`/`Lock` is a nil-interface panic inside `newLearnDeps`/`newQaDeps`). SUPERSEDED (Gate B #700 T3 review finding 2 — "three overlapping real-Deps test builders"): rather than adding `FS`/`Lock` fields to T2's hand-built `cli.Deps` literal, rebuild `newTestDeps` over `realDepsForTest()` (T1-rework's `primitives_integration_test.go` helper) so `FS`/`Lock` — and every other real-OS field — come from the single production composition, with `Stdout`/`Stderr` swapped in and `Embed` forced nil. Exact diff (same package `cli_test`):

```go
func newTestDeps(stdout, stderr io.Writer) cli.Deps {
	d := realDepsForTest()
	d.Stdout = stdout
	d.Stderr = stderr
	d.Embed = nil

	return d
}
```

NOTE(R11-naming): R11's text names the draft doubles `osEdgeFSForTest{}`/`flockLockerForTest{}` as the field values; under the composition doctrine those hand-rolled os doubles are NOT declared anywhere — the values are the composed `primFS`/`primLocker` reached through T1-rework's helpers, now via `realDepsForTest()` directly rather than duplicated field-by-field. R11's OWNERSHIP rule is unchanged: T3 is the sole extender, and post-collapse `newTestDeps` is the ONLY real-Deps test builder (`FS`/`Lock` ride in `realDepsForTest()`; `Embed` explicitly nil). No signal registration occurs (`realPrimitives()` omits `StartSignalPulses`, SIG-1) and the transiently-constructed lazy embedder built inside `realDepsForTest()`'s `cli.NewDeps` call is discarded (overwritten by the explicit nil), so T2's "no embedder/signal wiring" intent holds behaviorally.

No later task extends `newTestDeps` further (R11); every later executed targets-level path (count/show T5, show-chunk T6, ingest T8, prune T9, vocab/amend/resituate/activate T12) rides these same two fields. (`Embed` stays absent — see R11's recorded CONFLICT for the T15 embed closures.)

- [ ] 8. **export_test.go:** re-sign the two wrappers' exports (current lines 836-846) and add the constructor exports:

```go
// ExportNewLearnDeps exposes the pure Deps→LearnDeps composition for tests.
func ExportNewLearnDeps(d Deps) LearnDeps { return newLearnDeps(d) }

// ExportNewQaDeps exposes the pure Deps→LearnQADeps composition for tests.
func ExportNewQaDeps(d Deps) LearnQADeps { return newQaDeps(d) }

// ExportRunLearnFromFactArgs invokes the unexported runLearnFromFactArgs for testing.
func ExportRunLearnFromFactArgs(ctx context.Context, a LearnFactArgs, d Deps, stdout io.Writer) error {
	return runLearnFromFactArgs(ctx, a, d, stdout)
}

// ExportRunLearnFromFeedbackArgs invokes the unexported runLearnFromFeedbackArgs for testing.
func ExportRunLearnFromFeedbackArgs(
	ctx context.Context,
	a LearnFeedbackArgs,
	d Deps,
	stdout io.Writer,
) error {
	return runLearnFromFeedbackArgs(ctx, a, d, stdout)
}
```

Keep `ExportNewOsLearnFS` and `ExportFlockPath` untouched here (`ExportNewOsLearnFS`'s Lock-only consumer survives until T4/L2; `ExportFlockPath` is deleted later by T8, which also repoints its consumers — R7).

- [ ] 9. **learn_adapters_test.go:** delete `TestOsLearnFS_ListBasenames_*`, `TestOsLearnFS_ListIDs_*`, `TestOsLearnFS_MkdirAll_*`, `TestOsLearnFS_StatDir_*`, `TestOsLearnFS_WriteFileIfMissing_*`, `TestOsLearnFS_WriteNew_*` (lines 58-122, 133-288 — behavior now covered by deps_compose_test.go against the production composition + the internal primitives integration suite, `internal/cli/primitives_integration_test.go`; the former cmd adapter suites were deleted by T1-rework). Keep `TestOsLearnFS_Lock_BadVaultReturnsError` (124-131) until L2. Update every `cli.ExportRunLearnFromFactArgs(context.Background(), args, io.Discard)` / `...FeedbackArgs(...)` call (lines 37, 310, 371, 408, 456, 489) to `cli.ExportRunLearnFromFactArgs(context.Background(), args, newTestDeps(io.Discard, io.Discard), io.Discard)` (same for feedback). Note: with `Embed` forced nil in `newTestDeps` these tests skip auto-embed (they only assert `.md` presence/absence — assertions unchanged; the embed-on-write path stays covered by cli_test.go's real-binary end-to-end test, which asserts the `.vec.json` sidecar).

- [ ] 10. **K1 regression test survives (ADR-0013).** In `invariants_k1_property_test.go`, replace `k1RealLockDeps` (lines 122-140) — the test body (30-120) is unchanged:

```go
// unexported variables.
var errK1VaultMissing = errors.New("k1: vault should already exist")

// k1RealLockDeps wires LearnDeps through the PRODUCTION composition
// (newLearnDeps) over the internally-composed primFS EdgeFS and primLocker
// FileLocker with real OS primitives — the exact flock + exclusive-create
// (EdgeFS.WriteFileExcl over the base WriteFileExcl primitive) path the shipped binary builds
// via cli.NewDeps. Embed is nil (newTestDeps forces it) so auto-embed
// skips; InitVault errors because the caller pre-creates the vault.
func k1RealLockDeps(vault string) cli.LearnDeps {
	deps := cli.ExportNewLearnDeps(newTestDeps(io.Discard, io.Discard))

	deps.InitVault = func(string) error {
		return fmt.Errorf("%w: %s", errK1VaultMissing, vault)
	}

	return deps
}
```

(Add `"errors"` to the file's imports; the old hand-wired deps' `"time"`/`os.Getenv` uses disappear — drop those imports if nothing else in the file needs them.) This upgrade means K1 now races the production composition layer itself — lock file `vault/.luhmann.lock`, span ListIDs→WriteNew, O_EXCL backstop through `primFS.WriteFileExcl` — not a hand-wired double of it.

- [ ] 11. Run `targ test` → all green (RED tests from step 1 now pass; K1 passes at workers=2,5,10,20). Run `targ check-full` → clean. Run `targ check-thin-api` → PASS: this task touches NO cmd/engram file — the `WriteFileExcl` primitive and its literal closure landed with T1-rework/T2 (flag X-1: consume/verify). If the checker flags ANYTHING, escalate the exact finding to the orchestrator (global constraint / doctrine item 5) — do not suppress, do not restructure ad hoc. Then run `go install ./cmd/engram && cd "$(mktemp -d)" && engram learn fact --slug smoke --vault "$(mktemp -d)/v" --position top --source smoke --situation "smoke" --subject s --predicate p --object o` → prints the note path; the note and `.vec.json` sidecar exist.

- [ ] 12. Commit:

```
refactor(cli): compose learn-family deps from Deps (#700)

newLearnDeps/newQaDeps replace newOsLearnDeps/newOsLearnQADeps; all learn/qa
I/O flows through EdgeFS/FileLocker/Embed/Stderr. Adds the EdgeFS
WriteFileExcl method as the single internal wrap over the base
WriteFileExcl primitive (doctrine flags X-1/S-1), so the ADR-0013 K1
O_EXCL backstop survives composition;
K1 concurrency property now drives the production composition over real flock.

AI-Used: [claude]
```

---

### Task T4 (L2): purge os/syscall adapters from internal/cli/cli.go (ORDER: after activate/amend/resituate/vocab/ingest/prune constructor migrations, before enforcement)

**Depends on:** T8, T9, T12 complete (ingest/prune/maintenance constructors already compose from Deps) — see binding execution order R4.

**Files**
- Modify: `internal/cli/cli.go` (delete `osLearnFS`, `flockPath`, `logWarningToStderrf`; drop `os`/`syscall` imports; `osFileReader` and `osManifestLock` are already gone — deleted by T8 per R5 and T9 per R10, grep-verified here)
- Modify: `internal/cli/export_test.go` (delete `ExportNewOsLearnFS`, `ExportLogWarningToStderr`; `ExportFlockPath` and `ExportOsManifestLock` are already gone — deleted by T8 per R7 and T9 per R10, grep-verified here)
- Modify: `internal/cli/learn_adapters_test.go` (delete `TestOsLearnFS_Lock_BadVaultReturnsError`)
- Modify: `internal/cli/os_adapters_test.go` (delete `TestExportLogWarningToStderr_FormatsAndWrites` — superseded by `TestNewLearnDeps_LogWarning_WritesToDepsStderr`)
- Verify-only (no edit): `internal/cli/testhelpers_test.go` (`TestOsManifestLock_MkdirError` already deleted by T9 per R10), `internal/cli/ingest_test.go` (Lock closures already ride T8's `testFlocker{}` per R7)

**Interfaces**
- Consumes: nothing new. Produces: none — pure deletion.

NOTE (anchors): line numbers cited below are from the pristine tree and shift after T3's deletions (and the other families' migrations that land before T4 per R4); the symbol-based gates govern, not absolute lines.

**Steps**

- [ ] 1. **Precondition gate (must pass before any edit):**
`grep -rn "osLearnFS\|flockPath\|logWarningToStderrf" internal/ --include="*.go" | grep -v "_test.go"` → expected: hits ONLY inside `internal/cli/cli.go` (the definitions themselves). Any hit in activate.go/amend.go/resituate.go/vocab_commands.go/ingest.go/prune.go means a family migration hasn't landed — STOP and reorder. Additionally verify the earlier owners' deletions landed: `rg -n "osFileReader" internal/` → zero hits (T8, R5) and `rg -n "osManifestLock" internal/` → zero hits (T9, R10). Any hit → that task did not complete — STOP.

- [ ] 2. **Grep-verify the flock handoff (dead-work guard — R7).** T8 already deleted `ExportFlockPath` and repointed the `TestManifest_ConcurrentWritersDoNotLoseEntries` Lock closures to its test-local real-flock `testFlocker{}`, so the concurrent-writers regression keeps racing real OS locks without any production flock symbol and NOTHING is re-implemented here. Verify: `rg -n "ExportFlockPath" internal/` → zero hits (any hit → T8 incomplete, STOP). (`ExportManifestLockFile` stays — the constant it returns survives; `ExportOsManifestLock` was deleted by T9 per R10, covered by step 1's grep.) Then delete `ExportNewOsLearnFS` (553-554) and the `ExportLogWarningToStderr` alias (line 70) from export_test.go.

- [ ] 3. **Delete from cli.go:** `osLearnFS` + its `Lock` method (33-57 remnant), `flockPath` (165-192), `logWarningToStderrf` (relocated block from L1 step 4). (`osFileReader` and `osManifestLock` are NOT deleted here — T8/T9 already did, per R5/R10; step 1's greps proved it.) Remove `"os"`, `"syscall"`, and `"path/filepath"` (if now unused — `filepath.Join` was only used by the deleted lock helpers; `vaultLockFromLocker` in deps_compose.go has its own import) from the import block. cli.go retains: package doc, `luhmannLockFile`/`manifestLockFile` constants (consumed by `vaultLockFromLocker` and the ingest cluster's manifest-lock composition + `ExportManifestLockFile`), `errNotADirectory` (consumed by `statDirFromFS`), `acquireOptionalLock` (152-163, pure), `listRootNotes` (pure since L1), `pathOf` (240-243).

- [ ] 4. Delete `TestOsLearnFS_Lock_BadVaultReturnsError` (learn_adapters_test.go:124-131) and `TestExportLogWarningToStderr_FormatsAndWrites` (os_adapters_test.go:150-172). (`TestOsManifestLock_MkdirError` was already deleted by T9 per R10 — verify with `rg -n "TestOsManifestLock_MkdirError" internal/` → zero hits.) Confirm the replacement coverage exists before deleting: the internal locker suite covers lock-open failure (`TestPrimLocker_OpenFailureWrapsWithPath`, locker_test.go — T1-rework; per the supersession map the cmd-integration-test obligation landed as internal `_test` files); the ingest cluster's manifest-lock composition test covers mkdir-before-lock; `TestNewLearnDeps_LogWarning_WritesToDepsStderr` covers the warning format.

- [ ] 5. Verify purity of the migrated files: `grep -n "\"os\"\|\"syscall\"\|os\.\|syscall\." internal/cli/cli.go internal/cli/learn.go internal/cli/qa.go internal/cli/deps_compose.go` → no hits. Run `targ test` → green, including `TestInvariant_K1_ConcurrentLearnNeverCollides` and `TestManifest_ConcurrentWritersDoNotLoseEntries`. Run `targ check-full` → clean. Run `targ check-thin-api` → PASS (`All N public API files are thin wrappers.`); this task adds no cmd/engram declarations, so any finding predates it — escalate per Global Constraints, never suppress. `go install ./cmd/engram` and re-run the step-11 smoke from L1.

- [ ] 6. Commit:

```
refactor(cli): delete os/syscall adapters from cli.go (#700)

osLearnFS, flockPath, and the stderr warning hook are gone from internal
(osFileReader/osManifestLock fell earlier to T8/T9 — grep-verified);
raw I/O enters only as cmd/engram primitives composed by cli.NewDeps.
The concurrent-writers regression suites keep racing real OS flocks
via T8's test-local testFlocker (test files are exempt from the
purity deny).

AI-Used: [claude]
```

---

**ADR-0013 survival summary:** lock semantics are preserved exactly — same lock files (`vault/.luhmann.lock` via `vaultLockFromLocker`, `chunksDir/.manifest.lock` on the ingest side), same acquisition points (`writeLearnUnderLock` learn.go:657, `writeQANotesUnderLock` qa.go:427, both inside `Run*` entry flows), same span (ListIDs→WriteNew under one exclusive flock), same O_EXCL backstop (`EdgeFS.WriteFileExcl`), same atomic temp+rename edge (`EdgeFS.WriteFileAtomic`). The two regression tests survive: `TestInvariant_K1_ConcurrentLearnNeverCollides` (adapted in L1 step 10 to drive the production `newLearnDeps` composition over a real flock — strictly stronger than before) and `TestManifest_ConcurrentWritersDoNotLoseEntries` (its Lock closures repointed by T8 to the test-local real-flock `testFlocker{}` per R7 — still real OS flocks; L2/T4 only grep-verifies `ExportFlockPath` is gone).

## Complete os/time inventory for the cluster (line → replacement)

| File:Line | Current | Replacement |
|---|---|---|
| query.go:14 | `"time"` import | RETAINED — type-only use (`time.Time`/`time.Duration`) after fix |
| query.go:66,274-276,282-284,671,1302-1307,1403-1410,1426 | injected-clock/type usage | unchanged (already DI) |
| query.go:1288 | `newOsEmbedDeps()` (os-backed via embed.go) | direct composition: `ScanVault(newVaultFS(d.FS), …)` + `d.FS.ReadFile` + `d.Embed` |
| query.go:1294 | `logWarningToStderrf` (os.Stderr via learn.go:333) | `logWarningTo(d.Stderr)` |
| query.go:1295 | `ListChunkIndexes: listJSONLIndexes` (os.ReadDir) | `listJSONLIndexes(d.FS)` |
| query.go:1296 | `Now: time.Now` | `Now: d.Now` |
| query_chunks.go:8 | `"os"` import | retained through T6 (transitional osListJSONLIndexes), deleted in T12 (R3); add `"io/fs"` |
| query_chunks.go:139 | `os.ReadDir(dir)` | `fsys.ReadDir(dir)` (EdgeFS, via closure) |
| query_chunks.go:141 | `os.IsNotExist(err)` | `errors.Is(err, fs.ErrNotExist)` |
| query_chunks.go:188 | `fs := &osEmbedFS{}` | `d.FS` (newChunkQueryDeps) |
| query_chunks.go:193 | `Embedder: sharedEmbedder` | `Embedder: d.Embed` |
| query_chunks.go:12,206 | `"time"` / `now time.Time` param | RETAINED — pure |
| query_nominations.go | — | no os/time references |
| count.go:123 | `&osVaultFS{}` | `newVaultFS(d.FS)`; `newOsCountDeps()` → `newCountDeps(d Deps)` |
| show.go:69 | `&osVaultFS{}` | `newVaultFS(d.FS)`; `newOsShowDeps()` → `newShowDeps(d Deps)` |
| check.go:223 | `&osVaultFS{}` | `newVaultFS(d.FS)`; `newOsCheckDeps()` → `newCheckDeps(d Deps)` |
| vault_fs.go:6 | `"os"` import | deleted in Q3 (retained through Q1/Q2 for legacy `osVaultFS`) |
| vault_fs.go:23 | `os.ReadFile(filepath.Clean(path))` | `v.fs.ReadFile(filepath.Clean(path))` |
| vault_fs.go:35 | `os.ReadDir(dir)` | injected `readDir(dir)` (production = `EdgeFS.ReadDir`) |
| vault_fs.go:37 | `errors.Is(err, os.ErrNotExist)` | `errors.Is(err, fs.ErrNotExist)` (same sentinel; `io/fs`) |
| vault_fs.go:14-29 | `osVaultFS` type + methods | deleted in Q3 |
| targets.go:155,169,177,182,190 | `newOsQueryDeps()` / `newOsChunkQueryDeps()` / `newOsCountDeps()` / `newOsShowDeps()` / `newOsCheckDeps()` | `newQueryDeps(d)` / `newChunkQueryDeps(d)` / `newCountDeps(d)` / `newShowDeps(d)` / `newCheckDeps(d)` (targets cluster owns the file; same-commit edits) |

---

### Task T5 (Q1): Pure `vaultFS` adapter over EdgeFS; migrate count/show/check

**Files:**
- Create: `internal/cli/edgefs_os_test.go` (skip creation if an identical os-backed test EdgeFS already landed from another cluster — grep `osTestEdgeFS` first)
- Modify: `internal/cli/vault_fs.go`, `internal/cli/count.go`, `internal/cli/show.go`, `internal/cli/check.go`, `internal/cli/export_test.go`, `internal/cli/vault_fs_test.go`, `internal/cli/count_test.go`, `internal/cli/check_test.go`, `internal/cli/show_test.go`, `internal/cli/targets.go` (3 call-site lines)
- Delete: none (osVaultFS deletion deferred to Q3)

**Interfaces:**
- Consumes: `cli.Deps` / `cli.EdgeFS` (foundation task, internal/cli/deps.go); `vaultgraph.VaultFS` (`ListMD(dir string) ([]string, error)`; `ReadFile(path string) ([]byte, error)` — internal/vaultgraph/scanner.go:20); targets-cluster `d Deps` in `ingestQueryTargets`
- Produces: `func newVaultFS(fsys EdgeFS) *vaultFS` (satisfies vaultgraph.VaultFS); `func newCountDeps(d Deps) CountDeps`; `func newShowDeps(d Deps) ShowDeps`; `func newCheckDeps(d Deps) CheckDeps`; test shims `ExportNewVaultFS(fsys EdgeFS)`, `ExportNewCheckDeps(fsys EdgeFS) CheckDeps`, `ExportNewCountDeps(fsys EdgeFS) CountDeps`, `ExportNewShowDeps(fsys EdgeFS) ShowDeps`; shared test double `osTestEdgeFS` + `wrappedNotExistEdgeFS` (package cli_test)

**Steps:**

1. [ ] RED — create `internal/cli/edgefs_os_test.go` (package cli_test) with the real-FS EdgeFS double, and rewrite `internal/cli/vault_fs_test.go` to drive the not-yet-existing `cli.ExportNewVaultFS`. Run `targ test` — expected: compile failure (`undefined: cli.ExportNewVaultFS`, `undefined: cli.EdgeFS` absent if foundation not landed — foundation is a hard precondition).

   `edgefs_os_test.go` (complete):
   ```go
   package cli_test

   import (
   	"fmt"
   	"io/fs"
   	"os"
   	"path/filepath"
   )

   // osTestEdgeFS is a real-filesystem cli.EdgeFS for integration-style tests
   // over t.TempDir() fixtures. The production EdgeFS impl is internal/cli's
   // unexported primFS (composed by cli.NewDeps — T1-rework); test files are
   // exempt from the internal/ purity rules, so this double calls os directly.
   // Errors are wrapped with %w, matching the production contract that
   // errors.Is(err, fs.ErrNotExist) / errors.Is(err, fs.ErrExist) must
   // survive the adapter.
   // WriteFileAtomic is a plain write — ADR-0013 atomicity is exercised by
   // the internal edgefs/primitives integration suites (T1-rework), never
   // through this double.
   // WriteFileExcl is a real exclusive create (O_CREATE|O_EXCL) matching
   // T3's EdgeFS addition.
   type osTestEdgeFS struct{}

   func (osTestEdgeFS) ReadFile(path string) ([]byte, error) {
   	data, err := os.ReadFile(path) //nolint:gosec // test fixture path
   	if err != nil {
   		return nil, fmt.Errorf("test edgefs: reading %s: %w", path, err)
   	}

   	return data, nil
   }

   func (osTestEdgeFS) WriteFile(path string, data []byte, perm fs.FileMode) error {
   	err := os.WriteFile(path, data, perm)
   	if err != nil {
   		return fmt.Errorf("test edgefs: writing %s: %w", path, err)
   	}

   	return nil
   }

   func (osTestEdgeFS) WriteFileAtomic(path string, data []byte, perm fs.FileMode) error {
   	err := os.WriteFile(path, data, perm)
   	if err != nil {
   		return fmt.Errorf("test edgefs: writing %s: %w", path, err)
   	}

   	return nil
   }

   func (osTestEdgeFS) WriteFileExcl(path string, data []byte, perm fs.FileMode) error {
   	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm) //nolint:gosec // test fixture path
   	if err != nil {
   		return fmt.Errorf("test edgefs: opening excl %s: %w", path, err)
   	}

   	defer func() { _ = file.Close() }()

   	_, err = file.Write(data)
   	if err != nil {
   		return fmt.Errorf("test edgefs: writing excl %s: %w", path, err)
   	}

   	return nil
   }

   func (osTestEdgeFS) MkdirAll(path string, perm fs.FileMode) error {
   	err := os.MkdirAll(path, perm)
   	if err != nil {
   		return fmt.Errorf("test edgefs: mkdir %s: %w", path, err)
   	}

   	return nil
   }

   func (osTestEdgeFS) MkdirTemp(dir, pattern string) (string, error) {
   	path, err := os.MkdirTemp(dir, pattern)
   	if err != nil {
   		return "", fmt.Errorf("test edgefs: mkdtemp %s: %w", dir, err)
   	}

   	return path, nil
   }

   func (osTestEdgeFS) Stat(path string) (fs.FileInfo, error) {
   	info, err := os.Stat(path)
   	if err != nil {
   		return nil, fmt.Errorf("test edgefs: stat %s: %w", path, err)
   	}

   	return info, nil
   }

   func (osTestEdgeFS) ReadDir(path string) ([]fs.DirEntry, error) {
   	entries, err := os.ReadDir(path)
   	if err != nil {
   		return nil, fmt.Errorf("test edgefs: readdir %s: %w", path, err)
   	}

   	return entries, nil
   }

   func (osTestEdgeFS) Remove(path string) error {
   	err := os.Remove(path)
   	if err != nil {
   		return fmt.Errorf("test edgefs: remove %s: %w", path, err)
   	}

   	return nil
   }

   func (osTestEdgeFS) RemoveAll(path string) error {
   	err := os.RemoveAll(path)
   	if err != nil {
   		return fmt.Errorf("test edgefs: removeall %s: %w", path, err)
   	}

   	return nil
   }

   func (osTestEdgeFS) Rename(oldPath, newPath string) error {
   	err := os.Rename(oldPath, newPath)
   	if err != nil {
   		return fmt.Errorf("test edgefs: rename %s: %w", oldPath, err)
   	}

   	return nil
   }

   func (osTestEdgeFS) WalkDir(root string, fn fs.WalkDirFunc) error {
   	err := filepath.WalkDir(root, fn)
   	if err != nil {
   		return fmt.Errorf("test edgefs: walk %s: %w", root, err)
   	}

   	return nil
   }
   ```

   `vault_fs_test.go` (complete replacement; current file drives `cli.ExportNewOsVaultFS()` at lines 22/37/56/65/77):
   ```go
   package cli_test

   import (
   	"fmt"
   	"io/fs"
   	"os"
   	"path/filepath"
   	"testing"

   	. "github.com/onsi/gomega"

   	"github.com/toejough/engram/internal/cli"
   )

   func TestVaultFS_ListMD_FiltersDirsAndNonMd(t *testing.T) {
   	t.Parallel()
   	g := NewWithT(t)

   	dir := t.TempDir()
   	g.Expect(os.MkdirAll(filepath.Join(dir, "subdir"), 0o750)).To(Succeed())
   	g.Expect(os.WriteFile(filepath.Join(dir, "note.md"), []byte("x"), 0o600)).To(Succeed())
   	g.Expect(os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("y"), 0o600)).To(Succeed())

   	vfs := cli.ExportNewVaultFS(osTestEdgeFS{})
   	names, err := vfs.ListMD(dir)
   	g.Expect(err).NotTo(HaveOccurred())

   	if err != nil {
   		return
   	}

   	g.Expect(names).To(ConsistOf("note.md"))
   }

   func TestVaultFS_ListMD_MissingDirReturnsEmpty(t *testing.T) {
   	t.Parallel()
   	g := NewWithT(t)

   	vfs := cli.ExportNewVaultFS(osTestEdgeFS{})
   	names, err := vfs.ListMD("/nonexistent/vault/dir")
   	g.Expect(err).NotTo(HaveOccurred())

   	if err != nil {
   		return
   	}

   	g.Expect(names).To(BeEmpty())
   }

   func TestVaultFS_ListMD_NonExistError(t *testing.T) {
   	t.Parallel()
   	g := NewWithT(t)

   	// dir is a regular file, not a directory → ReadDir returns ENOTDIR (not not-exist).
   	path := filepath.Join(t.TempDir(), "file")
   	g.Expect(os.WriteFile(path, []byte("x"), 0o600)).To(Succeed())

   	vfs := cli.ExportNewVaultFS(osTestEdgeFS{})
   	_, err := vfs.ListMD(path)
   	g.Expect(err).To(HaveOccurred())
   }

   func TestVaultFS_ListMD_WrappedNotExistReturnsEmpty(t *testing.T) {
   	t.Parallel()
   	g := NewWithT(t)

   	// EdgeFS implementations wrap errors with %w; missing-dir detection must
   	// survive wrapping (errors.Is unwraps; os.IsNotExist would not).
   	vfs := cli.ExportNewVaultFS(wrappedNotExistEdgeFS{})
   	names, err := vfs.ListMD("/anything")
   	g.Expect(err).NotTo(HaveOccurred())

   	if err != nil {
   		return
   	}

   	g.Expect(names).To(BeEmpty())
   }

   func TestVaultFS_ReadFile_MissingPathError(t *testing.T) {
   	t.Parallel()
   	g := NewWithT(t)

   	vfs := cli.ExportNewVaultFS(osTestEdgeFS{})
   	_, err := vfs.ReadFile("/nonexistent/path.md")
   	g.Expect(err).To(HaveOccurred())
   }

   func TestVaultFS_ReadFile_Success(t *testing.T) {
   	t.Parallel()
   	g := NewWithT(t)

   	path := filepath.Join(t.TempDir(), "f.md")
   	g.Expect(os.WriteFile(path, []byte("hello"), 0o600)).To(Succeed())

   	vfs := cli.ExportNewVaultFS(osTestEdgeFS{})
   	data, err := vfs.ReadFile(path)
   	g.Expect(err).NotTo(HaveOccurred())

   	if err != nil {
   		return
   	}

   	g.Expect(string(data)).To(Equal("hello"))
   }

   // wrappedNotExistEdgeFS overrides ReadDir to return a WRAPPED fs.ErrNotExist,
   // proving missing-dir detection unwraps through EdgeFS error wrapping.
   type wrappedNotExistEdgeFS struct{ osTestEdgeFS }

   func (wrappedNotExistEdgeFS) ReadDir(string) ([]fs.DirEntry, error) {
   	return nil, fmt.Errorf("listing: %w", fs.ErrNotExist)
   }
   ```

2. [ ] GREEN — rewrite `internal/cli/vault_fs.go`. Replace the whole file body EXCEPT keep the legacy `osVaultFS` (with its own inline `os.ReadDir` call routed through the shared helper) until Q3:
   ```go
   package cli

   import (
   	"errors"
   	"fmt"
   	"io/fs"
   	"os"
   	"path/filepath"
   	"strings"
   )

   // vaultFS adapts the injected EdgeFS to vaultgraph.VaultFS — pure
   // composition, all I/O flows through the injected EdgeFS (#700).
   // Listing a non-existent directory returns an empty slice (not an error) —
   // the scanner uses this to skip missing subdirs (e.g. an absent MOCs/ on a
   // brand-new vault).
   type vaultFS struct {
   	fs EdgeFS
   }

   // newVaultFS returns a vaultgraph.VaultFS view over fsys.
   func newVaultFS(fsys EdgeFS) *vaultFS {
   	return &vaultFS{fs: fsys}
   }

   // ListMD returns the .md filenames in dir. Missing dir → empty, nil.
   func (v *vaultFS) ListMD(dir string) ([]string, error) {
   	return listDirBySuffix(v.fs.ReadDir, dir, ".md")
   }

   // ReadFile reads the file at path.
   func (v *vaultFS) ReadFile(path string) ([]byte, error) {
   	data, err := v.fs.ReadFile(filepath.Clean(path))
   	if err != nil {
   		return nil, fmt.Errorf("reading %s: %w", path, err)
   	}

   	return data, nil
   }

   // osVaultFS is the LEGACY direct-os adapter satisfying vaultgraph.VaultFS.
   // Deleted by the #700 purge task once amend/learn/qa/resituate/embed/vocab
   // migrate to newVaultFS(d.FS). Do not add new consumers.
   type osVaultFS struct{}

   // ListMD returns the .md filenames in dir. Missing dir → empty, nil.
   func (*osVaultFS) ListMD(dir string) ([]string, error) {
   	return listDirBySuffix(os.ReadDir, dir, ".md")
   }

   // ReadFile reads the file at path.
   func (*osVaultFS) ReadFile(path string) ([]byte, error) {
   	data, err := os.ReadFile(filepath.Clean(path))
   	if err != nil {
   		return nil, fmt.Errorf("reading %s: %w", path, err)
   	}

   	return data, nil
   }

   // listDirBySuffix returns the filenames directly inside dir whose name has
   // the given suffix, using the injected readDir (EdgeFS.ReadDir in
   // production). Missing dir → empty, nil — matched via errors.Is so wrapped
   // not-exist errors from EdgeFS adapters are recognized.
   func listDirBySuffix(
   	readDir func(string) ([]fs.DirEntry, error),
   	dir, suffix string,
   ) ([]string, error) {
   	entries, err := readDir(dir)
   	if err != nil {
   		if errors.Is(err, fs.ErrNotExist) {
   			return nil, nil
   		}

   		return nil, fmt.Errorf("reading dir %s: %w", dir, err)
   	}

   	out := make([]string, 0, len(entries))

   	for _, entry := range entries {
   		if entry.IsDir() {
   			continue
   		}

   		if !strings.HasSuffix(entry.Name(), suffix) {
   			continue
   		}

   		out = append(out, entry.Name())
   	}

   	return out, nil
   }
   ```
   (`os.ReadDir` is directly assignable to `func(string) ([]fs.DirEntry, error)` — `os.DirEntry` is a type alias of `fs.DirEntry`; `os.ErrNotExist` IS `fs.ErrNotExist`, so behavior is identical.)

3. [ ] Convert the three constructors (current code verified):
   - count.go:121-132, current:
     ```go
     // newOsCountDeps wires RunCount to the real filesystem vault reader/scanner.
     func newOsCountDeps() CountDeps {
     	fsys := &osVaultFS{}

     	return CountDeps{
     		ListMD:   fsys.ListMD,
     		ReadFile: fsys.ReadFile,
     		Scan: func(vault string) ([]vaultgraph.Note, error) {
     			return vaultgraph.ScanVault(fsys, vault)
     		},
     	}
     }
     ```
     replacement:
     ```go
     // newCountDeps wires RunCount from the injected CLI capabilities — pure
     // composition over EdgeFS (#700).
     func newCountDeps(d Deps) CountDeps {
     	fsys := newVaultFS(d.FS)

     	return CountDeps{
     		ListMD:   fsys.ListMD,
     		ReadFile: fsys.ReadFile,
     		Scan: func(vault string) ([]vaultgraph.Note, error) {
     			return vaultgraph.ScanVault(fsys, vault)
     		},
     	}
     }
     ```
   - show.go:67-77, current:
     ```go
     // newOsShowDeps wires RunShow to the real filesystem vault scanner and reader.
     func newOsShowDeps() ShowDeps {
     	fsys := &osVaultFS{}

     	return ShowDeps{
     		Scan: func(vault string) ([]vaultgraph.Note, error) {
     			return vaultgraph.ScanVault(fsys, vault)
     		},
     		Read: fsys.ReadFile,
     	}
     }
     ```
     replacement:
     ```go
     // newShowDeps wires RunShow from the injected CLI capabilities — pure
     // composition over EdgeFS (#700).
     func newShowDeps(d Deps) ShowDeps {
     	fsys := newVaultFS(d.FS)

     	return ShowDeps{
     		Scan: func(vault string) ([]vaultgraph.Note, error) {
     			return vaultgraph.ScanVault(fsys, vault)
     		},
     		Read: fsys.ReadFile,
     	}
     }
     ```
   - check.go:221-232, current:
     ```go
     // newOsCheckDeps wires RunCheck to the real filesystem vault scanner.
     func newOsCheckDeps() CheckDeps {
     	fsys := &osVaultFS{}

     	return CheckDeps{
     		Scan: func(vault string) ([]vaultgraph.Note, error) {
     			return vaultgraph.ScanVault(fsys, vault)
     		},
     		ReadNote:    fsys.ReadFile,
     		ReadSidecar: fsys.ReadFile,
     	}
     }
     ```
     replacement:
     ```go
     // newCheckDeps wires RunCheck from the injected CLI capabilities — pure
     // composition over EdgeFS (#700).
     func newCheckDeps(d Deps) CheckDeps {
     	fsys := newVaultFS(d.FS)

     	return CheckDeps{
     		Scan: func(vault string) ([]vaultgraph.Note, error) {
     			return vaultgraph.ScanVault(fsys, vault)
     		},
     		ReadNote:    fsys.ReadFile,
     		ReadSidecar: fsys.ReadFile,
     	}
     }
     ```

4. [ ] Update targets.go call sites (lines 177, 182, 190; requires targets-cluster `d Deps` in scope of `ingestQueryTargets` — adapt the variable name to what that task landed):
   - `errHandler(RunCount(a, newOsCountDeps(), stdout))` → `errHandler(RunCount(a, newCountDeps(d), stdout))`
   - `errHandler(RunShow(withLog(ctx), a, newOsShowDeps(), stdout))` → `errHandler(RunShow(withLog(ctx), a, newShowDeps(d), stdout))`
   - `errHandler(RunCheck(withLog(ctx), a, newOsCheckDeps(), stdout))` → `errHandler(RunCheck(withLog(ctx), a, newCheckDeps(d), stdout))`

   The executed targets-level tests riding these flips (`TestTargets_CountEmptyVault` count_test.go:562, the show test targets_test.go:160-177) dereference `d.FS` through `newVaultFS(d.FS)` — `newTestDeps` already carries `FS`/`Lock` since T3 (R11); no `newTestDeps` edit here.

5. [ ] Update export_test.go: delete the three var-block entries (lines 77, 78, 80):
   ```go
   ExportNewOsCheckDeps                   = newOsCheckDeps
   ExportNewOsCountDeps                   = newOsCountDeps
   ExportNewOsShowDeps                    = newOsShowDeps
   ```
   and add in the exported-functions section (alphabetical):
   ```go
   // ExportNewCheckDeps builds production CheckDeps over the given EdgeFS.
   func ExportNewCheckDeps(fsys EdgeFS) CheckDeps { return newCheckDeps(Deps{FS: fsys}) }

   // ExportNewCountDeps builds production CountDeps over the given EdgeFS.
   func ExportNewCountDeps(fsys EdgeFS) CountDeps { return newCountDeps(Deps{FS: fsys}) }

   // ExportNewShowDeps builds production ShowDeps over the given EdgeFS.
   func ExportNewShowDeps(fsys EdgeFS) ShowDeps { return newShowDeps(Deps{FS: fsys}) }

   // ExportNewVaultFS returns the pure EdgeFS→vaultgraph.VaultFS adapter.
   func ExportNewVaultFS(fsys EdgeFS) interface {
   	ListMD(dir string) ([]string, error)
   	ReadFile(path string) ([]byte, error)
   } {
   	return newVaultFS(fsys)
   }
   ```
   KEEP `ExportNewOsVaultFS` (vocab cluster tests still use it; deleted in Q3).

6. [ ] Update test call sites:
   - check_test.go lines 65, 86, 205, 235: `cli.ExportNewOsCheckDeps()` → `cli.ExportNewCheckDeps(osTestEdgeFS{})`
   - show_test.go line 89: `cli.ExportNewOsShowDeps()` → `cli.ExportNewShowDeps(osTestEdgeFS{})`
   - count_test.go line 464: `cli.ExportNewOsCountDeps()` → `cli.ExportNewCountDeps(osTestEdgeFS{})`; update the comment at 454-455 from "exercises newOsCountDeps" to "exercises newCountDeps over a real-FS EdgeFS" and at 560 from "newOsCountDeps + resolveVault" to "newCountDeps + resolveVault"

7. [ ] Run `targ test` — expected: all green (new vaultFS tests pass; count/show/check suites pass unchanged). Run `targ check-full` — expected: no new findings. Run `targ check-thin-api` — expected: PASS (`All N public API files are thin wrappers.`); this task adds no cmd/engram declarations, so any finding predates it — escalate per Global Constraints, never suppress.
8. [ ] Commit: `refactor(cli): vault reads via EdgeFS-backed vaultFS (#700)`

---

### Task T6 (Q2): query + query-chunks + show-chunk compose from Deps (EdgeFS lister, injected clock)

Sequencing: AFTER Q1 (`newVaultFS`, `osTestEdgeFS`). Per R4, T6 runs BEFORE the prune (T9) and amend (T12) conversions — so T6 flips `listJSONLIndexes` call sites ONLY in the files it converts itself (query.go, query_chunks.go, show_chunk.go; T6 owns the show-chunk conversion) and keeps the legacy os-backed lister alive, renamed `osListJSONLIndexes`, for amend/prune until T9/T12 flip their own lines. T12 (last consumer) deletes it, grep-gated. See R3.

**Files:**
- Modify: `internal/cli/query_chunks.go`, `internal/cli/query.go`, `internal/cli/show_chunk.go` (full `newShowChunkDeps` conversion — step 5), `internal/cli/export_test.go`, `internal/cli/query_chunks_test.go`, `internal/cli/ingest_integration_test.go` (2 lines), `internal/cli/targets.go` (3 lines), `internal/cli/amend.go` (1 line, mechanical rename only), `internal/cli/prune.go` (1 line, mechanical rename only), `internal/cli/deps.go` (only if `logWarningTo` not yet landed by the learn cluster)
- Delete: none

**Interfaces:**
- Consumes: `cli.Deps{FS, Embed, Stderr, Now}`; `logWarningTo(w io.Writer) func(format string, args ...any)`
- Produces: `func listJSONLIndexes(fsys EdgeFS) func(dir string) ([]string, error)` (CANONICAL final shape — T6 consumes it in-file for query/query-chunks/show-chunk; T9 (prune) and T12 (amend) consume it as `listJSONLIndexes(d.FS)` when they convert); `osListJSONLIndexes` (TRANSITIONAL — the renamed legacy os-backed lister, deleted by T12 grep-gated); `func newChunkQueryDeps(d Deps) ChunkQueryDeps`; `func newQueryDeps(d Deps) QueryDeps`; `func newShowChunkDeps(d Deps) ShowChunkDeps`; test shim `ExportNewChunkQueryDeps(fsys EdgeFS, emb embed.Embedder) ChunkQueryDeps`

**Steps:**

1. [ ] RED — add to `internal/cli/query_chunks_test.go` (package cli_test; this file uses `gomega.NewWithT`, non-dot import — add `"os"`, `"path/filepath"` to its imports):
   ```go
   func TestChunkQueryDeps_ListIndexes_WrappedNotExistIsEmptyIndex(t *testing.T) {
   	t.Parallel()
   	g := gomega.NewWithT(t)

   	deps := cli.ExportNewChunkQueryDeps(wrappedNotExistEdgeFS{}, nil)
   	paths, err := deps.ListIndexes("/any/chunks/dir")
   	g.Expect(err).NotTo(gomega.HaveOccurred())

   	if err != nil {
   		return
   	}

   	g.Expect(paths).To(gomega.BeEmpty())
   }

   func TestChunkQueryDeps_ListIndexes_ListsOnlyJSONLFiles(t *testing.T) {
   	t.Parallel()
   	g := gomega.NewWithT(t)

   	dir := t.TempDir()
   	g.Expect(os.MkdirAll(filepath.Join(dir, "sub.jsonl"), 0o750)).To(gomega.Succeed())
   	g.Expect(os.WriteFile(filepath.Join(dir, "a.jsonl"), []byte("{}"), 0o600)).To(gomega.Succeed())
   	g.Expect(os.WriteFile(filepath.Join(dir, "manifest.json"), []byte("{}"), 0o600)).To(gomega.Succeed())

   	deps := cli.ExportNewChunkQueryDeps(osTestEdgeFS{}, nil)
   	paths, err := deps.ListIndexes(dir)
   	g.Expect(err).NotTo(gomega.HaveOccurred())

   	if err != nil {
   		return
   	}

   	g.Expect(paths).To(gomega.ConsistOf(filepath.Join(dir, "a.jsonl")))
   }
   ```
   Run `targ test` — expected: compile failure (`undefined: cli.ExportNewChunkQueryDeps` — the old shim is `ExportNewOsChunkQueryDeps`).

2. [ ] GREEN — rewrite `internal/cli/query_chunks.go` I/O seams. Imports: add `"io/fs"`; KEEP `"os"` (the transitional lister below still uses `os.ReadDir`/`os.IsNotExist`; T12 deletes both). Do NOT delete the legacy os-backed lister (current lines 136-157) — amend.go and prune.go still reference it and have no `d Deps` in scope yet (R3). Instead RENAME it to `osListJSONLIndexes`, body unchanged, with this replacement doc comment:
   ```go
   // osListJSONLIndexes is the TRANSITIONAL os-backed .jsonl lister (#700).
   // Remaining consumers: amend.go (newOsAmendDeps) and prune.go
   // (newOsPruneDeps); T9/T12 flip those lines to listJSONLIndexes(d.FS) when
   // they convert, and T12 (last consumer) deletes this func + the "os" import.
   func osListJSONLIndexes(dir string) ([]string, error) {
   ```
   and ADD the canonical curried lister alongside it:
   ```go
   // listJSONLIndexes returns a lister over fsys for the .jsonl files directly
   // under a dir. A missing dir is an empty index (cold start), not an error —
   // matched via errors.Is so EdgeFS implementations may wrap the not-exist
   // error (os.IsNotExist would not unwrap a %w chain).
   func listJSONLIndexes(fsys EdgeFS) func(dir string) ([]string, error) {
   	return func(dir string) ([]string, error) {
   		entries, err := fsys.ReadDir(dir)
   		if err != nil {
   			if errors.Is(err, fs.ErrNotExist) {
   				return nil, nil
   			}

   			return nil, fmt.Errorf("listing chunk indexes: %w", err)
   		}

   		var paths []string

   		for _, entry := range entries {
   			if !entry.IsDir() && filepath.Ext(entry.Name()) == jsonlExt {
   				paths = append(paths, filepath.Join(dir, entry.Name()))
   			}
   		}

   		return paths, nil
   	}
   }
   ```
   and replace lines 185-195 (current `newOsChunkQueryDeps` using `fs := &osEmbedFS{}` + `sharedEmbedder`) with:
   ```go
   // newChunkQueryDeps wires `engram query-chunks` from the injected CLI
   // capabilities — pure composition (#700).
   func newChunkQueryDeps(d Deps) ChunkQueryDeps {
   	return ChunkQueryDeps{
   		ListIndexes: listJSONLIndexes(d.FS),
   		ReadFile:    d.FS.ReadFile,
   		Embedder:    d.Embed,
   	}
   }
   ```

3. [ ] Replace query.go:1286-1298, current:
   ```go
   // newOsQueryDeps wires the production scan + read for the query command.
   func newOsQueryDeps() QueryDeps {
   	embedDeps := newOsEmbedDeps()

   	return QueryDeps{
   		Scan:             embedDeps.Scan,
   		Read:             embedDeps.Read,
   		Embedder:         embedDeps.Embedder,
   		LogWarning:       logWarningToStderrf,
   		ListChunkIndexes: listJSONLIndexes,
   		Now:              time.Now,
   	}
   }
   ```
   with:
   ```go
   // newQueryDeps wires the query command from the injected CLI capabilities —
   // pure composition, every I/O flows through d (#700).
   func newQueryDeps(d Deps) QueryDeps {
   	vfs := newVaultFS(d.FS)

   	return QueryDeps{
   		Scan: func(vault string) ([]vaultgraph.Note, error) {
   			return vaultgraph.ScanVault(vfs, vault)
   		},
   		Read:             d.FS.ReadFile,
   		Embedder:         d.Embed,
   		LogWarning:       logWarningTo(d.Stderr),
   		ListChunkIndexes: listJSONLIndexes(d.FS),
   		Now:              d.Now,
   	}
   }
   ```
   (`"time"` import stays — `time.Time`/`time.Duration` types remain throughout query.go; `time.Now` is now gone. Behavioral note: `Read` error wrap text changes from osEmbedFS's `"read: %w"` to the cmd adapter's wrap — non-behavioral, both consumers at query.go:1068 and :1434 only branch on error presence.) If the learn cluster has not yet landed `logWarningTo`, add to `internal/cli/deps.go`:
   ```go
   // logWarningTo returns a LogWarning hook writing "warning: ..." lines to w —
   // the pure replacement for the legacy logWarningToStderrf (#700).
   func logWarningTo(w io.Writer) func(format string, args ...any) {
   	return func(format string, args ...any) {
   		_, _ = fmt.Fprintf(w, "warning: "+format+"\n", args...)
   	}
   }
   ```

4. [ ] Mechanical rename of the two remaining foreign references (same commit; rename ONLY — these constructors are still `newOsAmendDeps()`/`newOsPruneDeps()` with no `d Deps` in scope, so NO deps flip here; T12/T9 own those flips per R3):
   - amend.go:365 `ListIndexes: listJSONLIndexes,` → `ListIndexes: osListJSONLIndexes,`
   - prune.go:115 `ListIndexes: listJSONLIndexes,` → `ListIndexes: osListJSONLIndexes,`
   - amend.go:361-362 comment `// listJSONLIndexes (query_chunks.go) lists *.jsonl chunk indexes, treats` → start it with `// osListJSONLIndexes (query_chunks.go) lists ...` (T12 replaces the whole constructor, comment included, when it converts).

5. [ ] Convert `internal/cli/show_chunk.go` — the query family owns show-chunk; this also retires the last `osEmbedFS` consumer outside the files T8/T9/T12/T15 already handle, so T15's `osEmbedFS` deletion compiles. Current code (show_chunk.go:66-75, verified — `&osEmbedFS{}` at :69, lister reference at :72):
   ```go
   // newOsShowChunkDeps wires the production filesystem index loader for
   // `engram show-chunk`. No embedder is needed — lookup is by id, not similarity.
   func newOsShowChunkDeps() ShowChunkDeps {
   	fs := &osEmbedFS{}

   	return ShowChunkDeps{
   		ListIndexes: listJSONLIndexes,
   		ReadFile:    fs.Read,
   	}
   }
   ```
   Replace with:
   ```go
   // newShowChunkDeps wires `engram show-chunk` from the injected CLI
   // capabilities — pure composition (#700). No embedder is needed — lookup
   // is by id, not similarity.
   func newShowChunkDeps(d Deps) ShowChunkDeps {
   	return ShowChunkDeps{
   		ListIndexes: listJSONLIndexes(d.FS),
   		ReadFile:    d.FS.ReadFile,
   	}
   }
   ```
   No import changes in show_chunk.go (it never imported `os`; `osEmbedFS` came from package-mate embed.go). Behavioral note: `ReadFile`'s error wrap text changes from osEmbedFS's `"read: %w"` to the EdgeFS adapter's wrap — non-behavioral; `loadChunkRecords` only re-wraps. Test adjustments: NONE — show_chunk_test.go's three tests (lines 18, 39, 61) inject `cli.ShowChunkDeps{...}` literals directly and never touch the constructor; there is no `ExportNewOsShowChunkDeps` shim (verified); the wiring is exercised by targets_test.go:179's show-chunk test through `executeForTest` and the step-9 suite run.

6. [ ] Update export_test.go lines 514-521, current:
   ```go
   // ExportNewOsChunkQueryDeps returns production ChunkQueryDeps with an
   // injected embedder, mirroring ExportNewOsIngestDeps.
   func ExportNewOsChunkQueryDeps(emb embed.Embedder) ChunkQueryDeps {
   	deps := newOsChunkQueryDeps()
   	deps.Embedder = emb

   	return deps
   }
   ```
   replacement:
   ```go
   // ExportNewChunkQueryDeps returns production ChunkQueryDeps over the given
   // EdgeFS with an injected embedder.
   func ExportNewChunkQueryDeps(fsys EdgeFS, emb embed.Embedder) ChunkQueryDeps {
   	deps := newChunkQueryDeps(Deps{FS: fsys})
   	deps.Embedder = emb

   	return deps
   }
   ```
   Update ingest_integration_test.go lines 100 and 204: `cli.ExportNewOsChunkQueryDeps(fakeIngestEmbedder{})` → `cli.ExportNewChunkQueryDeps(osTestEdgeFS{}, fakeIngestEmbedder{})`.

7. [ ] Update targets.go call sites (identifier is `deps`, per T2's landed `ingestQueryTargets(deps Deps, ...)`): line 155 `newOsQueryDeps()` → `newQueryDeps(deps)`; line 169 `newOsChunkQueryDeps()` → `newChunkQueryDeps(deps)`; line 186 `newOsShowChunkDeps()` → `newShowChunkDeps(deps)` (line numbers are pre-T2 anchors — locate by constructor name).
8. [ ] Verify purity of the migrated files: `grep -n '"os"' internal/cli/query.go internal/cli/show_chunk.go` — expected: no output. `grep -n 'time\.Now' internal/cli/query.go` — expected: no output. `grep -n 'os\.' internal/cli/query_chunks.go` — expected: hits ONLY inside `osListJSONLIndexes` (the transitional lister; full query_chunks.go purity is T12's exit criterion, not this task's). `grep -rn 'newOsShowChunkDeps\|osEmbedFS' internal/cli/show_chunk.go` — expected: no output.
9. [ ] Run `targ test` — expected: all green (step-1 tests now pass; show-chunk, ingest integration + query suites unchanged). Run `targ check-full` — expected: clean. Run `targ check-thin-api` — expected: PASS (`All N public API files are thin wrappers.`); this task adds no cmd/engram declarations, so any finding predates it — escalate per Global Constraints, never suppress.
10. [ ] Commit: `refactor(cli): query + show-chunk compose from Deps (#700)`

---

### Task T7 (Q3): Purge legacy `osVaultFS` (grep-gated; runs after T12/T15 per R4)

Sequencing: after the amend/learn/qa/resituate/embed/vocab clusters migrate to `newVaultFS(d.FS)` and T12 migrates the vocab tests off `ExportNewOsVaultFS` (R12). Runs after T15 and T12 (R4) — all its preconditions precede it; T13/T16/T17 follow it in R4.

**Files:**
- Modify: `internal/cli/vault_fs.go` (delete `osVaultFS` + methods + `"os"` import), `internal/cli/export_test.go` (delete `ExportNewOsVaultFS`, lines currently 572-578)
- Delete: none

**Interfaces:**
- Consumes: nothing new. Produces: a pure vault_fs.go (zero I/O-capable imports).

**Steps:**
1. [ ] Gate: `grep -rn "osVaultFS\|ExportNewOsVaultFS" internal/cli --include='*.go'` — expected: hits ONLY in vault_fs.go (definition) and export_test.go (shim). Any other hit → STOP; that cluster has not migrated (the `ExportNewOsVaultFS` pattern is load-bearing — R12: the lowercase-only `osVaultFS` grep cannot see the capital-O shim call sites, so without it this task's deletion is a silent compile break); do not proceed.
2. [ ] Delete from vault_fs.go: the `osVaultFS` type, its `ListMD`/`ReadFile` methods, and the `"os"` import (all other imports stay: errors, fmt, io/fs, path/filepath, strings).
3. [ ] Delete from export_test.go:
   ```go
   // ExportNewOsVaultFS returns the production osVaultFS adapter for testing.
   func ExportNewOsVaultFS() interface {
   	ListMD(dir string) ([]string, error)
   	ReadFile(path string) ([]byte, error)
   } {
   	return &osVaultFS{}
   }
   ```
4. [ ] Verify purity: `grep -n '"os"' internal/cli/vault_fs.go` — expected: no output.
5. [ ] Run `targ test` then `targ check-full` — expected: all green, no findings. Run `targ check-thin-api` — expected: PASS (`All N public API files are thin wrappers.`); this task adds no cmd/engram declarations, so any finding predates it — escalate per Global Constraints, never suppress.
6. [ ] Commit: `refactor(cli): #700 delete legacy osVaultFS adapter`

---

Files read (worktree `/Users/joe/repos/personal/engram/.claude/worktrees/700-internal-purity`): `internal/cli/{query.go,query_chunks.go,query_nominations.go,count.go,show.go,check.go,vault_fs.go,targets.go,main.go,embed.go(80-269),learn.go(320-378),export_test.go,vault_fs_test.go,count_test.go(440-599),check_test.go(40-95),query_chunks_test.go(1-25),targets_test.go(420-455)}`, `internal/vaultgraph/scanner.go(14-30)`, `internal/embed/embedder.go(50-70)`, `cmd/engram/main.go`.

### Task T8 (I1): Migrate `engram ingest` wiring to cli.Deps composition

**Files:**
- Modify: `internal/cli/ingest.go` (replace `newOsIngestDeps`/`defaultSessionDir`/`walkSourcesExcluding` with pure compositions)
- Modify: `internal/cli/cli.go` (delete `osFileReader` only)
- Modify: `internal/cli/targets.go` (ingest call site — depends on foundation's `deps Deps` threading)
- Modify: `internal/cli/export_test.go` (swap `ExportNewOsIngestDeps` → `ExportNewIngestDeps`; delete `ExportFlockPath`, `ExportNewOsFileReader`)
- Create: `internal/cli/ingest_family_deps_test.go` (pure composition tests + cli_test OS harness)
- Modify: `internal/cli/ingest_auto_test.go`, `internal/cli/ingest_integration_test.go`, `internal/cli/ingest_test.go`, `internal/cli/adapters_test.go`

**Interfaces:**
- Consumes (foundation): `type Deps struct { …; Getenv func(string) string; Now func() time.Time; Getwd func() (string, error); UserHomeDir func() (string, error); FS EdgeFS; Lock FileLocker; Embed embed.Embedder; … }`; `EdgeFS` with `ReadFile/WriteFile/WriteFileAtomic/WriteFileExcl/MkdirAll/MkdirTemp/Stat/ReadDir/Remove/RemoveAll/Rename/WalkDir` (`WriteFileExcl` is T3's flagged addition); `FileLocker.Lock(path string) (unlock func() error, err error)`; `transcript.NewJSONLReader(reader sessionctx.FileReader) *JSONLReader` where `sessionctx.FileReader` is `Read(path string) ([]byte, error)` (internal/context/context.go:7).
- Produces: `func newIngestDeps(d Deps) IngestDeps`; `func manifestLockFrom(d Deps) func(chunksDir string) (func(), error)`; `func sessionDirFrom(getenv func(string) string, userHomeDir func() (string, error)) func(string) string`; `func sweepListerFrom(walkDir func(root string, fn fs.WalkDirFunc) error) func(SweepRoot) ([]string, error)`; `type fsFileReader struct{ fs EdgeFS }` with `Read(path string) ([]byte, error)`; test export `func ExportNewIngestDeps(d Deps, emb embed.Embedder) IngestDeps`.

**Steps:**

- [ ] 1. RED — create `internal/cli/ingest_family_deps_test.go` (package `cli_test`). It references `cli.ExportNewIngestDeps` and `cli.Deps`, which do not exist yet, so the test package fails to compile. Full content:

```go
package cli_test

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
)

func TestIngestDepsLockMkdirsChunksDirBeforeLocking(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	var calls []string

	d := cli.Deps{
		FS: fakeEdgeFS{
			mkdirAll: func(path string, perm fs.FileMode) error {
				calls = append(calls, fmt.Sprintf("mkdirall %s %o", path, perm))

				return nil
			},
		},
		Lock: fakeLocker{lock: func(path string) (func() error, error) {
			calls = append(calls, "lock "+path)

			return func() error {
				calls = append(calls, "unlock")

				return nil
			}, nil
		}},
	}

	release, err := cli.ExportNewIngestDeps(d, fakeIngestEmbedder{}).Lock("/chunks")
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil || release == nil {
		return
	}

	release()

	g.Expect(calls).To(gomega.Equal([]string{
		"mkdirall /chunks 700",
		"lock " + filepath.Join("/chunks", cli.ExportManifestLockFile()),
		"unlock",
	}), "MkdirAll must precede flock (fresh-dir regression), release must unlock")
}

func TestIngestDepsLockMkdirErrorPropagates(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	d := cli.Deps{
		FS: fakeEdgeFS{
			mkdirAll: func(string, fs.FileMode) error { return errBoom },
		},
		Lock: fakeLocker{lock: func(string) (func() error, error) {
			t.Fatal("Lock must not be called when MkdirAll fails")

			return nil, nil
		}},
	}

	_, err := cli.ExportNewIngestDeps(d, fakeIngestEmbedder{}).Lock("/chunks")
	g.Expect(err).To(gomega.MatchError(errBoom))
}

func TestIngestDepsSessionDirDefaultsToClaudeProjects(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	d := testDeps()
	d.Getenv = func(string) string { return "" }
	d.UserHomeDir = func() (string, error) { return "/home/u", nil }

	deps := cli.ExportNewIngestDeps(d, fakeIngestEmbedder{})

	g.Expect(deps.SessionDir("/anywhere")).
		To(gomega.Equal(filepath.Join("/home/u", ".claude", "projects")))

	d.UserHomeDir = func() (string, error) { return "", errBoom }
	deps = cli.ExportNewIngestDeps(d, fakeIngestEmbedder{})

	g.Expect(deps.SessionDir("/anywhere")).To(gomega.BeEmpty(),
		"unresolvable home yields empty session dir, not a panic")
}

func TestIngestDepsSessionDirHonorsTranscriptDirEnv(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	d := testDeps()
	d.Getenv = func(key string) string {
		if key == "ENGRAAM_NEVER" { // guard: only the real key may match below
			return ""
		}

		if key == "ENGRAM_TRANSCRIPT_DIR" {
			return "/custom/sessions"
		}

		return ""
	}

	deps := cli.ExportNewIngestDeps(d, fakeIngestEmbedder{})

	g.Expect(deps.SessionDir("/anywhere")).To(gomega.Equal("/custom/sessions"))
}

func TestIngestDepsStatMapsFileInfoToSourceStat(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	mtime := time.Date(2026, 7, 19, 10, 0, 0, 0, time.UTC)

	const wantSize = int64(42)

	d := testDeps()
	d.FS = fakeEdgeFS{stat: func(string) (fs.FileInfo, error) {
		return fakeFileInfo{mtime: mtime, size: wantSize}, nil
	}}

	deps := cli.ExportNewIngestDeps(d, fakeIngestEmbedder{})

	stat, err := deps.Stat("/src.md")
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stat).To(gomega.Equal(cli.SourceStat{
		MtimeUnixNano: mtime.UnixNano(),
		Size:          wantSize,
	}))

	d.FS = fakeEdgeFS{stat: func(string) (fs.FileInfo, error) { return nil, errBoom }}
	deps = cli.ExportNewIngestDeps(d, fakeIngestEmbedder{})

	_, err = deps.Stat("/src.md")
	g.Expect(err).To(gomega.MatchError(errBoom))
}

func TestIngestDepsWriteFileMkdirsParentThenWritesAtomically(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	var calls []string

	d := testDeps()
	d.FS = fakeEdgeFS{
		mkdirAll: func(path string, perm fs.FileMode) error {
			calls = append(calls, fmt.Sprintf("mkdirall %s %o", path, perm))

			return nil
		},
		writeFileAtomic: func(path string, _ []byte, perm fs.FileMode) error {
			calls = append(calls, fmt.Sprintf("writeatomic %s %o", path, perm))

			return nil
		},
	}

	deps := cli.ExportNewIngestDeps(d, fakeIngestEmbedder{})

	g.Expect(deps.WriteFile("/chunks/idx.jsonl", []byte("x"))).To(gomega.Succeed())
	g.Expect(calls).To(gomega.Equal([]string{
		"mkdirall /chunks 700",
		"writeatomic /chunks/idx.jsonl 600",
	}), "parent dir created before atomic 0600 write")
}

// --- cli_test OS harness: real-FS EdgeFS + real flock FileLocker ---
// Test files are exempt from the internal I/O purity enforcement; the
// production adapters are composed in internal/cli from cmd-supplied
// primitives (T1-rework). These doubles back the ingest-family
// integration tests and the ADR-0013 concurrency regression.

type fakeEdgeFS struct {
	readFile        func(string) ([]byte, error)
	writeFile       func(string, []byte, fs.FileMode) error
	writeFileAtomic func(string, []byte, fs.FileMode) error
	writeFileExcl   func(string, []byte, fs.FileMode) error
	mkdirAll        func(string, fs.FileMode) error
	mkdirTemp       func(string, string) (string, error)
	stat            func(string) (fs.FileInfo, error)
	readDir         func(string) ([]fs.DirEntry, error)
	remove          func(string) error
	removeAll       func(string) error
	rename          func(string, string) error
	walkDir         func(string, fs.WalkDirFunc) error
}

func (f fakeEdgeFS) ReadFile(path string) ([]byte, error) { return f.readFile(path) }

func (f fakeEdgeFS) WriteFile(path string, data []byte, perm fs.FileMode) error {
	return f.writeFile(path, data, perm)
}

func (f fakeEdgeFS) WriteFileAtomic(path string, data []byte, perm fs.FileMode) error {
	return f.writeFileAtomic(path, data, perm)
}

func (f fakeEdgeFS) WriteFileExcl(path string, data []byte, perm fs.FileMode) error {
	return f.writeFileExcl(path, data, perm)
}

func (f fakeEdgeFS) MkdirAll(path string, perm fs.FileMode) error { return f.mkdirAll(path, perm) }

func (f fakeEdgeFS) MkdirTemp(dir, pattern string) (string, error) {
	return f.mkdirTemp(dir, pattern)
}

func (f fakeEdgeFS) Stat(path string) (fs.FileInfo, error)     { return f.stat(path) }
func (f fakeEdgeFS) ReadDir(path string) ([]fs.DirEntry, error) { return f.readDir(path) }
func (f fakeEdgeFS) Remove(path string) error                   { return f.remove(path) }
func (f fakeEdgeFS) RemoveAll(path string) error                { return f.removeAll(path) }

func (f fakeEdgeFS) Rename(oldPath, newPath string) error { return f.rename(oldPath, newPath) }

func (f fakeEdgeFS) WalkDir(root string, fn fs.WalkDirFunc) error { return f.walkDir(root, fn) }

// fakeFileInfo satisfies fs.FileInfo for Stat-mapping tests.
type fakeFileInfo struct {
	mtime time.Time
	size  int64
	dir   bool
}

func (f fakeFileInfo) Name() string       { return "fake" }
func (f fakeFileInfo) Size() int64        { return f.size }
func (f fakeFileInfo) Mode() fs.FileMode  { return 0o600 }
func (f fakeFileInfo) ModTime() time.Time { return f.mtime }
func (f fakeFileInfo) IsDir() bool        { return f.dir }
func (f fakeFileInfo) Sys() any           { return nil }

// fakeLocker satisfies cli.FileLocker with an injected func.
type fakeLocker struct {
	lock func(string) (func() error, error)
}

func (l fakeLocker) Lock(path string) (func() error, error) { return l.lock(path) }

// osTestFS implements cli.EdgeFS over the real filesystem.
type osTestFS struct{}

func (osTestFS) ReadFile(path string) ([]byte, error) { return os.ReadFile(path) }

func (osTestFS) WriteFile(path string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(path, data, perm)
}

func (osTestFS) WriteFileAtomic(path string, data []byte, perm fs.FileMode) error {
	return testAtomicWrite(path, data, perm)
}

func (osTestFS) WriteFileExcl(path string, data []byte, perm fs.FileMode) error {
	return testWriteFileExcl(path, data, perm)
}

func (osTestFS) MkdirAll(path string, perm fs.FileMode) error { return os.MkdirAll(path, perm) }

func (osTestFS) MkdirTemp(dir, pattern string) (string, error) {
	return os.MkdirTemp(dir, pattern)
}

func (osTestFS) Stat(path string) (fs.FileInfo, error)      { return os.Stat(path) }
func (osTestFS) ReadDir(path string) ([]fs.DirEntry, error) { return os.ReadDir(path) }
func (osTestFS) Remove(path string) error                   { return os.Remove(path) }
func (osTestFS) RemoveAll(path string) error                { return os.RemoveAll(path) }

func (osTestFS) Rename(oldPath, newPath string) error { return os.Rename(oldPath, newPath) }

func (osTestFS) WalkDir(root string, fn fs.WalkDirFunc) error {
	return filepath.WalkDir(root, fn)
}

// testFlocker implements cli.FileLocker with a real syscall flock, preserving
// the ADR-0013 concurrency regression's real-lock semantics inside cli_test.
type testFlocker struct{}

func (testFlocker) Lock(path string) (func() error, error) {
	const lockPerm = 0o600

	lockFile, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, lockPerm)
	if err != nil {
		return nil, fmt.Errorf("open lock: %w", err)
	}

	fileDescriptor := int(lockFile.Fd())

	flockErr := syscall.Flock(fileDescriptor, syscall.LOCK_EX)
	if flockErr != nil {
		_ = lockFile.Close()

		return nil, fmt.Errorf("flock: %w", flockErr)
	}

	return func() error {
		_ = syscall.Flock(fileDescriptor, syscall.LOCK_UN)

		if closeErr := lockFile.Close(); closeErr != nil {
			return fmt.Errorf("close lock: %w", closeErr)
		}

		return nil
	}, nil
}

// testAtomicWrite is temp+rename, mirroring production WriteFileAtomic.
func testAtomicWrite(path string, data []byte, perm fs.FileMode) error {
	tmp := path + ".tmp-test"

	err := os.WriteFile(tmp, data, perm)
	if err != nil {
		return fmt.Errorf("atomic write temp: %w", err)
	}

	err = os.Rename(tmp, path)
	if err != nil {
		_ = os.Remove(tmp)

		return fmt.Errorf("atomic write rename: %w", err)
	}

	return nil
}

// testWriteFileExcl is exclusive create (O_CREATE|O_EXCL), mirroring the
// production WriteFileExcl; a wrapped fs.ErrExist survives errors.Is.
func testWriteFileExcl(path string, data []byte, perm fs.FileMode) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm)
	if err != nil {
		return fmt.Errorf("excl write open: %w", err)
	}

	defer func() { _ = file.Close() }()

	_, writeErr := file.Write(data)
	if writeErr != nil {
		return fmt.Errorf("excl write: %w", writeErr)
	}

	return nil
}

// testDeps returns a real-OS cli.Deps for integration tests.
func testDeps() cli.Deps {
	return cli.Deps{
		FS:          osTestFS{},
		Lock:        testFlocker{},
		Getenv:      os.Getenv,
		Now:         time.Now,
		Getwd:       os.Getwd,
		UserHomeDir: os.UserHomeDir,
	}
}

var _ = errors.Is // keep errors imported if assertions above change shape
```

(Drop the trailing `var _ = errors.Is` line and the `errors` import if unused after final assembly — goimports will flag it.) Run: `targ test`. Expected: FAIL — `undefined: cli.ExportNewIngestDeps`, `undefined: cli.Deps` fields as applicable. This is the compile-RED for the whole task.

- [ ] 2. GREEN — rewrite `internal/cli/ingest.go` wiring. Replace the import block line `"os"` → (removed) and add `"io/fs"`. Exact current import block (lines 3-18) becomes:

```go
import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/embed"
	"github.com/toejough/engram/internal/transcript"
)
```

Update the two stale doc comments in `IngestDeps` (current lines 39-41 and 43-46): replace `// release func. Wired to flockPath(chunksDir/.manifest.lock) in newOsIngestDeps.` with `// release func. Wired to manifestLockFrom (MkdirAll + FileLocker flock) in newIngestDeps.` and replace `// callers guard with "if deps.Now != nil" before calling. Wire time.Now in` / `// newOsIngestDeps.` with `// callers guard with "if deps.Now != nil" before calling. Wired to deps.Now in` / `// newIngestDeps.`

Add named perms to the unexported constants block (current lines 129-138), after `manifestName`:

```go
	chunksDirPerm = 0o700
	indexFilePerm = 0o600
```

DELETE `defaultSessionDir` (current lines 244-258) entirely. DELETE `newOsIngestDeps` (current lines 483-523) entirely. DELETE `walkSourcesExcluding` (current lines 652-689) entirely. ADD these replacements (alphabetical placement per existing file order — `fsFileReader` near top-of-types is fine; the repo's reorder-decls lint will confirm):

```go
// fsFileReader adapts EdgeFS.ReadFile to the context.FileReader interface
// consumed by transcript.NewJSONLReader (production transcript stripping).
type fsFileReader struct {
	fs EdgeFS
}

// Read reads path via the injected edge filesystem.
func (r fsFileReader) Read(path string) ([]byte, error) {
	return r.fs.ReadFile(path)
}

// manifestLockFrom composes the ADR-0013 manifest lock from the CLI-edge
// Deps: ensure chunksDir exists first (regression: prune's lock on a fresh
// dir errored without MkdirAll), then take an exclusive advisory lock on
// chunksDir/.manifest.lock via the injected FileLocker. The unlock error is
// discarded to preserve the historical release-func() contract at every Run*
// entry point. Shared by newIngestDeps and newPruneDeps so the MkdirAll+lock
// pair lives in exactly one place.
func manifestLockFrom(d Deps) func(chunksDir string) (func(), error) {
	return func(chunksDir string) (func(), error) {
		err := d.FS.MkdirAll(chunksDir, chunksDirPerm)
		if err != nil {
			return nil, fmt.Errorf("creating chunks dir for lock: %w", err)
		}

		unlock, err := d.Lock.Lock(filepath.Join(chunksDir, manifestLockFile))
		if err != nil {
			return nil, fmt.Errorf("locking %s: %w", manifestLockFile, err)
		}

		return func() { _ = unlock() }, nil
	}
}

// newIngestDeps composes production IngestDeps from the CLI-edge Deps: every
// filesystem, clock, and environment capability flows through d — internal/
// stays free of direct os.* calls. WriteFile creates the chunks directory on
// demand so first ingest into a fresh dir succeeds.
func newIngestDeps(d Deps) IngestDeps {
	reader := transcript.NewJSONLReader(fsFileReader{fs: d.FS})

	return IngestDeps{
		Lock:     manifestLockFrom(d),
		ReadFile: d.FS.ReadFile,
		WriteFile: func(path string, data []byte) error {
			err := d.FS.MkdirAll(filepath.Dir(path), chunksDirPerm)
			if err != nil {
				return fmt.Errorf("ingest: creating chunks dir: %w", err)
			}

			err = d.FS.WriteFileAtomic(path, data, indexFilePerm)
			if err != nil {
				return fmt.Errorf("ingest: writing %s: %w", path, err)
			}

			return nil
		},
		Stat: func(path string) (SourceStat, error) {
			info, err := d.FS.Stat(path)
			if err != nil {
				return SourceStat{}, fmt.Errorf("ingest: stat %s: %w", path, err)
			}

			return SourceStat{MtimeUnixNano: info.ModTime().UnixNano(), Size: info.Size()}, nil
		},
		ListSources:    sweepListerFrom(d.FS.WalkDir),
		ReadTranscript: reader.ReadFrom,
		Embedder:       d.Embed,
		Now:            d.Now,
		IsDir: func(path string) bool {
			info, err := d.FS.Stat(path)

			return err == nil && info.IsDir()
		},
		Getwd:      d.Getwd,
		SessionDir: sessionDirFrom(d.Getenv, d.UserHomeDir),
	}
}

// sessionDirFrom resolves the root of ALL recorded session transcripts:
// ENGRAM_TRANSCRIPT_DIR when set (headless/eval cells get only their own
// sessions), else <home>/.claude/projects — every project, every conversation.
func sessionDirFrom(
	getenv func(string) string,
	userHomeDir func() (string, error),
) func(string) string {
	return func(_ string) string {
		if dir := getenv("ENGRAM_TRANSCRIPT_DIR"); dir != "" {
			return dir
		}

		home, err := userHomeDir()
		if err != nil {
			return ""
		}

		return filepath.Join(home, ".claude", "projects")
	}
}

// sweepListerFrom returns a SweepRoot lister walking via the injected walkDir
// (EdgeFS.WalkDir in production), pruning excluded directory names
// (build/dependency trees) and, when SkipHidden, every dot-directory.
func sweepListerFrom(
	walkDir func(root string, fn fs.WalkDirFunc) error,
) func(SweepRoot) ([]string, error) {
	return func(root SweepRoot) ([]string, error) {
		excluded := make(map[string]struct{}, len(root.ExcludeDirs))
		for _, name := range root.ExcludeDirs {
			excluded[name] = struct{}{}
		}

		var paths []string

		err := walkDir(root.Path, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if entry.IsDir() {
				if path == root.Path {
					return nil
				}

				hidden := root.SkipHidden && strings.HasPrefix(entry.Name(), ".")
				if hidden || shouldSkipDir(entry.Name(), excluded, root.ExcludePrefixes) {
					return filepath.SkipDir
				}

				return nil
			}

			paths = append(paths, path)

			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("ingest: walking %s: %w", root.Path, err)
		}

		return paths, nil
	}
}
```

- [ ] 3. Delete `osFileReader` from `internal/cli/cli.go` — exact current lines 25-31:

```go
// I/O adapters for context package DI interfaces.

type osFileReader struct{}

func (r *osFileReader) Read(path string) ([]byte, error) {
	return os.ReadFile(path) //nolint:gosec,wrapcheck // thin I/O adapter
}
```

(Its only production consumer was ingest.go:488, replaced in step 2. Do NOT touch `flockPath` or `osManifestLock` in this task — `osManifestLock` still backs `newOsPruneDeps` until Task I2; `flockPath` still backs `osLearnFS.Lock`.)

- [ ] 4. Update `internal/cli/targets.go` ingest call site. Current lines 157-160:

```go
		targ.Targ(func(ctx context.Context, a IngestArgs) {
			a.ChunksDir = ResolveChunksDir(a.ChunksDir, home, os.Getenv)
			errHandler(RunIngest(withLog(ctx), a, newOsIngestDeps(), stdout))
		}).Name("ingest").Description("Chunk+embed transcripts/markdown into a chunk index (zero-LLM)"),
```

Replacement (assumes the foundation task threads `deps Deps` into `ingestQueryTargets`; adopt its exact parameter name):

```go
		targ.Targ(func(ctx context.Context, a IngestArgs) {
			a.ChunksDir = ResolveChunksDir(a.ChunksDir, home, deps.Getenv)
			errHandler(RunIngest(withLog(ctx), a, newIngestDeps(deps), stdout))
		}).Name("ingest").Description("Chunk+embed transcripts/markdown into a chunk index (zero-LLM)"),
```

- [ ] 5. Update `internal/cli/export_test.go`. Replace (current lines 543-551):

```go
// ExportNewOsIngestDeps returns production IngestDeps with an injected
// embedder so coverage tests can drive the wiring without unpacking the
// lazy bundled embedder.
func ExportNewOsIngestDeps(emb embed.Embedder) IngestDeps {
	deps := newOsIngestDeps()
	deps.Embedder = emb

	return deps
}
```

with:

```go
// ExportNewIngestDeps returns production IngestDeps composed from d with an
// injected embedder so coverage tests can drive the wiring without unpacking
// the lazy bundled embedder.
func ExportNewIngestDeps(d Deps, emb embed.Embedder) IngestDeps {
	deps := newIngestDeps(d)
	deps.Embedder = emb

	return deps
}
```

Delete `ExportFlockPath` (current lines 342-347) and `ExportNewOsFileReader` (current lines 536-541).

- [ ] 6. Adapt `internal/cli/ingest_test.go`. In `TestManifest_ConcurrentWritersDoNotLoseEntries`, both Lock closures (current lines 375-377 and 402-404):

```go
			Lock: func(dir string) (func(), error) {
				return cli.ExportFlockPath(dir + "/" + cli.ExportManifestLockFile())
			},
```

become (both occurrences):

```go
			Lock: func(dir string) (func(), error) {
				unlock, lockErr := testFlocker{}.Lock(dir + "/" + cli.ExportManifestLockFile())
				if lockErr != nil {
					return nil, lockErr
				}

				return func() { _ = unlock() }, nil
			},
```

And `realFS.write` (current lines 898-900):

```go
func (r *realFS) write(_, path string, data []byte) error {
	return cli.ExportAtomicWriteFile(path, data, 0o600)
}
```

becomes:

```go
func (r *realFS) write(_, path string, data []byte) error {
	return testAtomicWrite(path, data, 0o600)
}
```

- [ ] 7. Adapt `internal/cli/ingest_auto_test.go` (three call sites). `TestDefaultSessionDirHonorsTranscriptDirEnv` (current lines 48-57, uses `t.Setenv`) is DELETED — superseded by the pure `TestIngestDepsSessionDirHonorsTranscriptDirEnv` from step 1 (now parallel-safe, no env mutation). In `TestDefaultSessionDirIsAllProjects` (line 63) and `TestOsListSourcesSkipsExcludedDirs` (line 95), replace `deps := cli.ExportNewOsIngestDeps(fakeIngestEmbedder{})` with `deps := cli.ExportNewIngestDeps(testDeps(), fakeIngestEmbedder{})`. In `internal/cli/ingest_integration_test.go` line 188 replace `ingestDeps := cli.ExportNewOsIngestDeps(fakeIngestEmbedder{})` with `ingestDeps := cli.ExportNewIngestDeps(testDeps(), fakeIngestEmbedder{})`. Remove the now-unused `os` import from ingest_auto_test.go only if no other use remains (TestOsListSourcesSkipsExcludedDirs still uses os.MkdirAll/os.WriteFile — keep it).

- [ ] 8. Adapt `internal/cli/adapters_test.go`: delete `TestOsFileReader_Read` (lines 14-30) and `TestOsFileReader_ReadError` (lines 32-39). The transcript-read path is now covered by the pure `fsFileReader` composition (exercised end-to-end by `TestOsIngestThenChunkQuery` through `testDeps()`), and raw `os.ReadFile` adapter coverage lives in the internal real-primitives EdgeFS suite (`TestRealEdgeFS_ReadWriteRoundTrip` / `TestRealEdgeFS_ReadFileMissingSatisfiesErrNotExist`, primitives_integration_test.go — T1-rework). Keep the `osLearnFS` tests untouched.

- [ ] 9. Verify: `targ test` — expected PASS (all new pure tests green; `TestOsIngestThenChunkQuery`, `TestManifest_ConcurrentWritersDoNotLoseEntries`, sweep/auto tests green). Then purity greps — both must print nothing:
  - `grep -nE '\bos\.|filepath\.WalkDir|time\.Now|syscall' internal/cli/ingest.go`
  - `grep -n 'osFileReader' internal/cli/*.go` (only historical mentions in comments must be gone too)
  Then `targ check-full` — expected: no new lint findings (fix any reorder-decls/lll it reports in the touched files before proceeding). Then `targ check-thin-api` — expected: PASS (`All N public API files are thin wrappers.`); this task adds no cmd/engram declarations, so any finding predates it — escalate per Global Constraints, never suppress.

- [ ] 10. Commit:

```
git add internal/cli/ingest.go internal/cli/cli.go internal/cli/targets.go \
  internal/cli/export_test.go internal/cli/ingest_family_deps_test.go \
  internal/cli/ingest_auto_test.go internal/cli/ingest_integration_test.go \
  internal/cli/ingest_test.go internal/cli/adapters_test.go
git commit -m "refactor(cli): compose ingest deps from cli.Deps (#700)

ENGRAM_TRANSCRIPT_DIR + home via deps.Getenv/UserHomeDir, walk via
EdgeFS.WalkDir, manifest lock via FileLocker behind MkdirAll composition
(ADR-0013 semantics preserved; concurrency regression adapted to a
test-local real flock).

AI-Used: [claude]"
```

---

### Task T9 (I2): Migrate `engram prune` wiring to cli.Deps composition and retire osManifestLock

**Files:**
- Modify: `internal/cli/prune.go` (replace `newOsPruneDeps` with `newPruneDeps`; flip its lister line from the transitional `osListJSONLIndexes` to T6's canonical `listJSONLIndexes(d.FS)` — per R3, T9 declares NO lister helper)
- Modify: `internal/cli/cli.go` (delete `osManifestLock` — last consumer gone)
- Modify: `internal/cli/targets.go` (prune call site)
- Modify: `internal/cli/export_test.go` (swap `ExportNewOsPruneDeps` → `ExportNewPruneDeps`; delete `ExportOsManifestLock`)
- Modify: `internal/cli/ingest_family_deps_test.go` (add prune composition tests)
- Modify: `internal/cli/prune_integration_test.go`, `internal/cli/testhelpers_test.go`

**Interfaces:**
- Consumes: `Deps`/`EdgeFS`/`FileLocker` (foundation), `manifestLockFrom` (Task I1), `chunksDirPerm`/`indexFilePerm` consts (Task I1), `listJSONLIndexes(fsys EdgeFS)` (T6's canonical curried lister — landed earlier per R4).
- Produces: `func newPruneDeps(d Deps) PruneDeps`. Per R3, T9 declares NO lister helper (`jsonlIndexListerFrom` is not created anywhere): it flips prune.go's line off the transitional `osListJSONLIndexes` onto `listJSONLIndexes(d.FS)`; the transitional lister itself is deleted later by T12 (amend, its last consumer), grep-gated.

**Steps:**

- [ ] 1. RED — append to `internal/cli/ingest_family_deps_test.go` (references `cli.ExportNewPruneDeps`, which does not exist yet → compile RED). Also add `fakeDirEntry` to the private helpers section:

```go
func TestPruneDepsExistsAndRemoveGoThroughFS(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	var removed []string

	d := testDeps()
	d.FS = fakeEdgeFS{
		stat: func(path string) (fs.FileInfo, error) {
			if path == "/live.jsonl" {
				return fakeFileInfo{}, nil
			}

			return nil, fs.ErrNotExist
		},
		remove: func(path string) error {
			removed = append(removed, path)

			return nil
		},
	}

	deps := cli.ExportNewPruneDeps(d)

	g.Expect(deps.Exists("/live.jsonl")).To(gomega.BeTrue())
	g.Expect(deps.Exists("/dead.jsonl")).To(gomega.BeFalse())
	g.Expect(deps.Remove("/chunks/empty.jsonl")).To(gomega.Succeed())
	g.Expect(removed).To(gomega.Equal([]string{"/chunks/empty.jsonl"}))
}

func TestPruneDepsListIndexesColdStartAndFiltering(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	d := testDeps()
	d.FS = fakeEdgeFS{readDir: func(string) ([]fs.DirEntry, error) {
		return nil, fmt.Errorf("reading dir: %w", fs.ErrNotExist)
	}}

	deps := cli.ExportNewPruneDeps(d)

	paths, err := deps.ListIndexes("/absent")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(paths).To(gomega.BeEmpty(), "missing chunks dir is a cold start, not an error")

	d.FS = fakeEdgeFS{readDir: func(string) ([]fs.DirEntry, error) {
		return []fs.DirEntry{
			fakeDirEntry{name: "a.jsonl"},
			fakeDirEntry{name: "notes.md"},
			fakeDirEntry{name: "sub", dir: true},
			fakeDirEntry{name: "b.jsonl"},
		}, nil
	}}
	deps = cli.ExportNewPruneDeps(d)

	paths, err = deps.ListIndexes("/chunks")
	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(paths).To(gomega.Equal([]string{
		filepath.Join("/chunks", "a.jsonl"),
		filepath.Join("/chunks", "b.jsonl"),
	}), "only non-dir .jsonl entries, joined under dir")

	d.FS = fakeEdgeFS{readDir: func(string) ([]fs.DirEntry, error) { return nil, errBoom }}
	deps = cli.ExportNewPruneDeps(d)

	_, err = deps.ListIndexes("/chunks")
	g.Expect(err).To(gomega.MatchError(errBoom))
}
```

and in the private-helpers section:

```go
// fakeDirEntry satisfies fs.DirEntry for index-lister tests.
type fakeDirEntry struct {
	name string
	dir  bool
}

func (e fakeDirEntry) Name() string { return e.name }
func (e fakeDirEntry) IsDir() bool  { return e.dir }

func (e fakeDirEntry) Type() fs.FileMode {
	if e.dir {
		return fs.ModeDir
	}

	return 0
}

func (e fakeDirEntry) Info() (fs.FileInfo, error) { return nil, fs.ErrInvalid }
```

Run `targ test` — expected FAIL: `undefined: cli.ExportNewPruneDeps`.

- [ ] 2. GREEN — rewrite `internal/cli/prune.go`. Import block (current lines 3-10; `filepath` stays for the `filepath.Join` calls in `RunPrune`) becomes:

```go
import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
)
```

Update the stale `PruneDeps.Lock` doc comment (current line 22 — the full line, prefix included, so a literal find/replace works): `// release func. Wired to flockPath(chunksDir/.manifest.lock) in newOsPruneDeps.` → `// release func. Wired to manifestLockFrom (MkdirAll + FileLocker flock) in newPruneDeps.` Replace `newOsPruneDeps` (current lines 102-118; note the `ListIndexes` line reads `osListJSONLIndexes` at this point — T6's mechanical rename landed earlier per R4):

```go
// newOsPruneDeps wires the production filesystem for `engram prune`.
func newOsPruneDeps() PruneDeps {
	fs := &osEmbedFS{}

	return PruneDeps{
		Lock:      osManifestLock,
		ReadFile:  fs.Read,
		WriteFile: fs.Write,
		Exists: func(path string) bool {
			_, statErr := os.Stat(path)

			return statErr == nil
		},
		ListIndexes: osListJSONLIndexes,
		Remove:      os.Remove,
	}
}
```

with (per R3: NO lister helper is declared here — the lister is T6's canonical `listJSONLIndexes(d.FS)`; prune.go was one of `osListJSONLIndexes`'s two remaining consumers, and after this flip only amend.go remains, so T12 deletes it there, grep-gated):

```go
// newPruneDeps composes production PruneDeps from the CLI-edge Deps.
func newPruneDeps(d Deps) PruneDeps {
	return PruneDeps{
		Lock:     manifestLockFrom(d),
		ReadFile: d.FS.ReadFile,
		WriteFile: func(path string, data []byte) error {
			err := d.FS.WriteFileAtomic(path, data, indexFilePerm)
			if err != nil {
				return fmt.Errorf("prune: writing %s: %w", path, err)
			}

			return nil
		},
		Exists: func(path string) bool {
			_, statErr := d.FS.Stat(path)

			return statErr == nil
		},
		ListIndexes: listJSONLIndexes(d.FS),
		Remove:      d.FS.Remove,
	}
}
```

- [ ] 3. Delete `osManifestLock` from `internal/cli/cli.go` — exact current lines 223-236 (function + doc comment). Its two consumers (`newOsIngestDeps`, `newOsPruneDeps`) are now both gone. Leave `flockPath` in place (still consumed by `osLearnFS.Lock`; the learn cluster removes it).

- [ ] 4. Update `internal/cli/targets.go` prune call site. Current lines 161-166:

```go
		targ.Targ(func(ctx context.Context, a PruneArgs) {
			a.ChunksDir = ResolveChunksDir(a.ChunksDir, home, os.Getenv)
			errHandler(RunPrune(withLog(ctx), a, newOsPruneDeps(), stdout))
		}).Name("prune").Description(
```

becomes:

```go
		targ.Targ(func(ctx context.Context, a PruneArgs) {
			a.ChunksDir = ResolveChunksDir(a.ChunksDir, home, deps.Getenv)
			errHandler(RunPrune(withLog(ctx), a, newPruneDeps(deps), stdout))
		}).Name("prune").Description(
```

- [ ] 5. Update `internal/cli/export_test.go`: in the Exported-variables block delete the line `ExportNewOsPruneDeps = newOsPruneDeps` (current line 79); delete `ExportOsManifestLock` (current lines 687-690). Add near the other constructor exports:

```go
// ExportNewPruneDeps returns production PruneDeps composed from d.
func ExportNewPruneDeps(d Deps) PruneDeps { return newPruneDeps(d) }
```

- [ ] 6. Adapt tests. `internal/cli/prune_integration_test.go` line 48: `cli.ExportNewOsPruneDeps()` → `cli.ExportNewPruneDeps(testDeps())`. `internal/cli/testhelpers_test.go`: delete `TestOsManifestLock_MkdirError` (current lines 13-28) and its now-unused imports if any (`os`, `filepath`, `cli`, `gomega` — check remaining uses; `sliceIndex` stays) — its coverage is replaced by the pure `TestIngestDepsLockMkdirErrorPropagates` (Task I1) plus the shared `manifestLockFrom` path now exercised by both constructors.

- [ ] 7. Verify: `targ test` — expected PASS, including `TestOsPruneDetachesDeadSource` (real FS through `testDeps()`), `TestRunPrune_LocksManifestAroundReadModifyWrite` (untouched — pure injected deps), and `TestManifest_ConcurrentWritersDoNotLoseEntries`. Purity grep, must print nothing: `grep -nE '\bos\.|time\.Now|syscall' internal/cli/prune.go`. Then `targ check-full` — expected clean. Then `targ check-thin-api` — expected PASS (`All N public API files are thin wrappers.`); this task adds no cmd/engram declarations, so any finding predates it — escalate per Global Constraints, never suppress. Then run the real flow once (passing tests are not a usable system): `go install ./cmd/engram && cd $(mktemp -d) && ENGRAM_CHUNKS_DIR=$PWD/chunks engram prune` — expected stdout `prune: no manifest, nothing to prune` and a created `chunks/.manifest.lock` (proves the MkdirAll-before-lock composition against the real wired binary; requires the foundation cmd wiring to be in place).

- [ ] 8. Commit:

```
git add internal/cli/prune.go internal/cli/cli.go internal/cli/targets.go \
  internal/cli/export_test.go internal/cli/ingest_family_deps_test.go \
  internal/cli/prune_integration_test.go internal/cli/testhelpers_test.go
git commit -m "refactor(cli): compose prune deps, retire osManifestLock (#700)

Stat/Remove/ReadDir via EdgeFS, manifest lock via shared manifestLockFrom
(MkdirAll-before-flock fresh-dir regression preserved). Prune's index lister
now flows through the canonical listJSONLIndexes(d.FS) (T6); the transitional
osListJSONLIndexes keeps amend.go as its sole consumer until T12 deletes it.

AI-Used: [claude]"
```

---

Verified source anchors: internal/cli/ingest.go (os at 248/252/496/504/516/520, time.Now at 514, filepath.WalkDir at 662), internal/cli/prune.go (os.Stat 111, os.Remove 116), internal/cli/cli.go (osFileReader 27, flockPath 169, osManifestLock 227, manifestLockFile 17), internal/cli/targets.go (157-166), internal/cli/query_chunks.go (listJSONLIndexes 138), internal/cli/embed.go (osEmbedFS 140, sharedEmbedder 110), internal/context/context.go (FileReader 7), internal/embed/embedder.go (Embedder 54), tests: ingest_test.go 319/376/403/899, ingest_auto_test.go 48-105, ingest_integration_test.go 188, prune_integration_test.go 48, testhelpers_test.go 13-28, adapters_test.go 14-39, export_test.go 79/342/536/543/687.

### Task T10 (M1): Atomic-write parity on the internal composition (relocation absorbed by T1-rework; consumers migrate in T12/T4/T15)

**Doctrine note (supersedes this task's original relocate-to-cmd brief):** under the revised composition doctrine there is no `cmd/engram/os_fs.go` to relocate onto — the ADR-0013 dance already lives INTERNAL as `primFS.WriteFileAtomic` (`internal/cli/edgefs.go`, landed by T1-rework; flag P-4 sequence: internal unique-name candidates (target base + Now nanos + attempt counter) → exclusive create via the `WriteFileExcl` primitive (bounded fs.ErrExist retry) → Chmod to the exact target perm via the restored `Chmod` primitive (umask-independent, parity with the pre-#700 dance) → Rename, + Remove on any post-creation failure (including a chmod failure)), unit-tested with fake primitives (`TestEdgeFS_WriteFileAtomicHappyPathDance`, `TestEdgeFS_WriteFileAtomicFailuresRemoveTemp`, `TestEdgeFS_WriteFileAtomicUniqueNameRetry` in edgefs_test.go) and integration-tested with real ones (`TestRealEdgeFS_WriteFileAtomicReplacesContentAndCleansTemp`, `TestRealEdgeFS_WriteFileAtomicPermsAreUmaskIndependent` in primitives_integration_test.go). The supersession map's "T10 reduces to migrating its internal consumers onto `deps.FS.WriteFileAtomic`" is realized BY the owning tasks, not here: every remaining caller sits inside an os-backed constructor/adapter that its own task replaces wholesale — learn.go/qa.go (T3, already landed at this slot in R4's order), amend/resituate/activate/vocab (T12), cli.go's `osLearnFS.WriteSidecar` (T4), embed.go's `osEmbedFS.Write` (T15) — exactly T13's gate ledger, which is UNCHANGED. A standalone call-site flip in this task is meaningless (the enclosing adapters die whole). What the M1 slot still owes the cluster before T13 may delete writesafe.go + writesafe_test.go: re-prove, against the composed implementation over REAL primitives, the writesafe regression behaviors T1-rework did not carry over, and verify the caller ledger matches T13's gate.

Behavior-parity ledger (writesafe_test.go behavior → coverage on the composed `primFS`):

| writesafe_test.go behavior | Re-proven by | Landed in |
|---|---|---|
| `TestAtomicWriteFile_OverwritesExistingFile` | `TestRealEdgeFS_WriteFileAtomicReplacesContentAndCleansTemp` | T1-rework |
| `TestAtomicWriteFile_NoLeftoverTempFiles` | same test's closing `HaveLen(1)` temp-count assertion | T1-rework |
| `TestAtomicWriteFile_WritesNewFile` | `TestRealEdgeFS_WriteFileAtomicWritesNewFile` | THIS TASK |
| `TestAtomicWriteFile_FailureDoesNotTouchOriginal` | `TestRealEdgeFS_WriteFileAtomicExclCreateFailureLeavesOriginalUntouched` (exclusive-create failure/retry leaves original untouched) | THIS TASK |
| `TestAtomicWriteFile_RenameFailure_CleansTempAndLeavesOriginalUntouched` | `TestRealEdgeFS_WriteFileAtomicRenameFailureCleansTempAndOriginal` (real FS, injected `Rename` primitive — no export shim needed) plus the fake-prims rename case of `TestEdgeFS_WriteFileAtomicFailuresRemoveTemp` | THIS TASK |
| *(none — the pre-#700 dance's chmod-driven umask independence had no explicit writesafe_test.go coverage)* | `TestRealEdgeFS_WriteFileAtomicPermsAreUmaskIndependent` (restored `Chmod` primitive, P-4) | T1-rework |

**Files**
- Modify (append tests, one import, one const block, one sentinel var): `internal/cli/primitives_integration_test.go` (created by T1-rework)
- (No production edits. No deletions — writesafe.go keeps its callers until T12/T4/T15 migrate them; T13 deletes it.)

**Interfaces**
- Consumes: `cli.EdgeFS.WriteFileAtomic` through T1-rework's cli_test helpers — `realFSForTest()`, `realPrimitives()`, `fsFromPrims(...)` (edgefs_test.go; package `cli_test` is one namespace across files) — and const `realFSFilePerm`.
- Produces: the three THIS-TASK parity tests in the ledger; test-only consts `writableDirPerm`/`readOnlyDirPerm` and sentinel `errInjectedRename` (collision-checked free in package `cli_test`). No production symbols. (The original brief's cmd-side `osFS.WriteFileAtomic`/`doAtomicWrite` are NOT produced — nothing may reference them.)

**Steps**

1. [ ] Preflight — verify the T1-rework/T2 landed state this task builds on (any miss → STOP: an upstream task is incomplete; escalate rather than building the missing piece here):
   - `rg -n "func \(f primFS\) WriteFileAtomic" internal/cli/edgefs.go` → exactly one hit.
   - `rg -n "TestEdgeFS_WriteFileAtomicHappyPathDance|TestEdgeFS_WriteFileAtomicFailuresRemoveTemp|TestEdgeFS_WriteFileAtomicUniqueNameRetry" internal/cli/edgefs_test.go` → all three present.
   - `rg -n "TestRealEdgeFS_WriteFileAtomicReplacesContentAndCleansTemp|func realPrimitives|func realFSForTest" internal/cli/primitives_integration_test.go` → all present.
   - `ls cmd/engram/` → `main.go` only (no os_fs.go / os_signal.go / debuglog_sink.go — the pre-rework layout is gone; a trivial wiring smoke test, if T2 kept one, is also acceptable).

2. [ ] Parity tests — relocation onto ALREADY-LANDED code, so this step is verify-form, not RED/GREEN: all three tests are expected GREEN on arrival. Any FAILURE is a genuine parity defect in the landed dance — STOP, fix `internal/cli/edgefs.go` under its unit suite, re-run; never bend the test to the defect. Append to `internal/cli/primitives_integration_test.go`, adding `"errors"` to its import block:

```go
func TestRealEdgeFS_WriteFileAtomicWritesNewFile(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	target := filepath.Join(dir, "out.txt")
	content := []byte("hello atomic world")
	fsys := realFSForTest()

	g.Expect(fsys.WriteFileAtomic(target, content, realFSFilePerm)).To(gomega.Succeed())

	got, readErr := os.ReadFile(target)
	g.Expect(readErr).NotTo(gomega.HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(got).To(gomega.Equal(content), "file must contain exactly the written bytes")
}

func TestRealEdgeFS_WriteFileAtomicExclCreateFailureLeavesOriginalUntouched(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	fsys := realFSForTest()

	// A read-only directory makes the exclusive-create (WriteFileExcl)
	// primitive fail with a non-ErrExist error, so the dance aborts before
	// creating anything (relocated writesafe_test.go behavior; the
	// fs.ErrExist retry path is covered by the fake-prims
	// TestEdgeFS_WriteFileAtomicUniqueNameRetry).
	subdir := filepath.Join(dir, "sub")
	g.Expect(os.Mkdir(subdir, writableDirPerm)).To(gomega.Succeed())

	target := filepath.Join(subdir, "original.txt")
	original := []byte("original untouched content")
	g.Expect(os.WriteFile(target, original, realFSFilePerm)).To(gomega.Succeed())

	g.Expect(os.Chmod(subdir, readOnlyDirPerm)).To(gomega.Succeed())

	// Restore permissions so TempDir cleanup can succeed.
	t.Cleanup(func() { _ = os.Chmod(subdir, writableDirPerm) })

	err := fsys.WriteFileAtomic(target, []byte("new content"), realFSFilePerm)
	g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("create temp")),
		"write into a read-only dir must fail at temp creation")

	// Make the directory readable again for the assertions.
	g.Expect(os.Chmod(subdir, writableDirPerm)).To(gomega.Succeed())

	got, readErr := os.ReadFile(target)
	g.Expect(readErr).NotTo(gomega.HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(got).To(gomega.Equal(original), "original file must be untouched after failure")

	tmpFiles, globErr := filepath.Glob(filepath.Join(subdir, ".original.txt.tmp-*"))
	g.Expect(globErr).NotTo(gomega.HaveOccurred())
	g.Expect(tmpFiles).To(gomega.BeEmpty(), "no leftover .tmp-* files must remain after failure")
}

func TestRealEdgeFS_WriteFileAtomicRenameFailureCleansTempAndOriginal(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	dir := t.TempDir()
	target := filepath.Join(dir, "file.txt")
	original := []byte("original content")

	g.Expect(os.WriteFile(target, original, realFSFilePerm)).To(gomega.Succeed())

	// Real primitives except Rename — parameterizing the dance over
	// Primitives is what lets this test inject the rename failure with no
	// export shim (replaces writesafe_test.go's injected-rename test).
	var tmpSeen string

	prims := realPrimitives()
	prims.Rename = func(oldPath, _ string) error {
		tmpSeen = oldPath

		return errInjectedRename
	}

	err := fsFromPrims(prims).WriteFileAtomic(target, []byte("new content"), realFSFilePerm)
	g.Expect(err).To(gomega.MatchError(errInjectedRename))
	g.Expect(err).To(gomega.MatchError(gomega.ContainSubstring("rename")),
		"error must name the failing dance step")

	got, readErr := os.ReadFile(target)
	g.Expect(readErr).NotTo(gomega.HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(got).To(gomega.Equal(original), "original must be untouched after rename failure")

	g.Expect(tmpSeen).NotTo(gomega.BeEmpty(), "the dance must reach the rename step")
	g.Expect(tmpSeen).NotTo(gomega.BeAnExistingFile(), "temp file must be removed by the failure cleanup")

	tmpFiles, globErr := filepath.Glob(filepath.Join(dir, ".file.txt.tmp-*"))
	g.Expect(globErr).NotTo(gomega.HaveOccurred())
	g.Expect(tmpFiles).To(gomega.BeEmpty(), "no leftover .tmp-* files must remain after failure")
}

// errInjectedRename is the sentinel injected by the rename-failure parity test.
var errInjectedRename = errors.New("injected rename failure")

// Directory permission modes for the read-only-dir parity test.
const (
	writableDirPerm fs.FileMode = 0o700
	readOnlyDirPerm fs.FileMode = 0o500
)
```

3. [ ] Run `targ test` — expect PASS (the three parity tests green against the landed dance; internal/cli/writesafe_test.go's originals also still green — they are deleted by T13, not here).

4. [ ] Caller-ledger check (verify-only, ZERO edits): `grep -rn "atomicWriteFile(" internal/cli --include="*.go" | grep -v _test | grep -v writesafe.go` → exactly the six callers T13's gate assigns to later tasks (current numbering): amend.go:351 + resituate.go:169 + activate.go:136 + vocab_commands.go:1217 (→ T12), cli.go:144 (→ T4), embed.go:164 (→ T15). A learn.go or qa.go hit → T3 incomplete, STOP. Any caller OUTSIDE this ledger → T13's gate can never clear; escalate to the orchestrator with the exact hit (do not migrate it ad hoc — the owning task's brief must absorb it).

5. [ ] Run `targ check-full` — expect clean. Run `targ check-thin-api` — expect PASS (this task touches no cmd code, so any finding predates it → escalate per Global Constraints, never suppress).

6. [ ] Commit:

```
test(cli): prove writesafe parity on internal atomic write (#700)

The ADR-0013 dance lives on internal/cli's primFS.WriteFileAtomic
(landed by the T1 rework). This closes the writesafe_test.go parity
gap with real-primitive regression tests — exclusive-create failure
leaves the original untouched, injected rename failure cleans the
temp, and new-file write — so the purge task (T13) can delete
writesafe.go with zero behavior-coverage loss. Call-site migration
stays with the owning tasks (T12/T4/T15) per T13's gate.

AI-Used: [claude]
```

---

### Task T11 (M2): os-backed test Deps + contract tests over the landed canonical helpers

**Files**
- Create: `internal/cli/deps_compose_internal_test.go` (fake-driven contract tests over the LANDED canonical helpers)
- Create: `internal/cli/testdeps_test.go` (`ExportNewTestOsDeps` built through `cli.NewDeps` over a real-OS `Primitives` literal — NO hand-rolled adapter mirrors; composition doctrine)
- Verify-only (NO edit — R1): `internal/cli/deps_compose.go`. T3 created it and it already carries everything this task's draft once declared: the four parallel-drafted helpers (`edgeVaultFS`, `vaultLuhmannLock`, `warnLoggerTo`, `jsonlIndexesLister`) are LOSERS per R1/R2/R3 — their canonical equivalents are T5's `newVaultFS(fsys EdgeFS)` (vault_fs.go), T3's `vaultLockFromLocker`/`logWarningTo` (deps_compose.go), and T6's `listJSONLIndexes(fsys EdgeFS)` (query_chunks.go). T11 appends NOTHING to deps_compose.go; NEVER apply a full-file `package cli` replacement (it would clobber T3's landed `vaultLockFromLocker`/`logWarningTo`/`statDirFromFS`/`initVaultFromFS`/`list*FromFS`/`write*FromFS` helpers).

**Interfaces**
- Consumes: `type Deps struct{...}`, `type EdgeFS interface{...}`, `type FileLocker interface{...}` from internal/cli/deps.go (foundation task — M2 is blocked on it); `luhmannLockFile` const (internal/cli/cli.go:16, pure, stays); the landed canonical helpers `newVaultFS` (T5), `listJSONLIndexes` (T6), `vaultLockFromLocker` + `logWarningTo` (T3) — all in place before T11 per R4; `NewDeps`, `Primitives` (T1-rework/T2 — the base struct already carries the `WriteFileExcl` exclusive-create primitive, survivor S-1), and `WriteSyncer` for the step-5 literal.
- Produces:
  - Test-only: `func ExportNewTestOsDeps() Deps` (package cli, _test.go file; body is one `NewDeps` call over an inline real-OS `Primitives` literal) — this task's ONLY new symbol.
  - Additional fake-driven test coverage of the canonical helpers' contracts (wrapped-ErrNotExist handling, lock path, warning format).

**Steps**

1. [ ] Create `internal/cli/deps_compose_internal_test.go` (package `cli`, internal-test precedent: resituate_internal_test.go). These tests drive the LANDED canonical helpers (T3/T5/T6 — no new production code in this task, so there is no compile-RED; they add fake-driven contract coverage the real-FS suites can't express as directly):

```go
package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func TestNewVaultFS_ListMD_FiltersToMDFilesSkippingDirs(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vfs := newVaultFS(fakeEdgeFS{readDir: func(string) ([]fs.DirEntry, error) {
		return []fs.DirEntry{
			fakeDirEntry{name: "1.2026-01-01.note.md"},
			fakeDirEntry{name: "sidecar.vec.json"},
			fakeDirEntry{name: "subdir", dir: true},
		}, nil
	}})

	names, err := vfs.ListMD("/vault")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(names).To(Equal([]string{"1.2026-01-01.note.md"}))
}

func TestNewVaultFS_ListMD_MissingDirIsEmptyNotError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vfs := newVaultFS(fakeEdgeFS{readDir: func(string) ([]fs.DirEntry, error) {
		return nil, fmt.Errorf("read dir: %w", fs.ErrNotExist)
	}})

	names, err := vfs.ListMD("/missing")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(names).To(BeEmpty())
}

func TestNewVaultFS_ReadFile_WrapsErrorWithPath(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vfs := newVaultFS(fakeEdgeFS{readFile: func(string) ([]byte, error) {
		return nil, errInjectedCompose
	}})

	_, err := vfs.ReadFile("/vault/x.md")
	g.Expect(err).To(MatchError(errInjectedCompose))
	g.Expect(err).To(MatchError(ContainSubstring("/vault/x.md")))
}

func TestListJSONLIndexes_FiltersAndTreatsMissingDirAsEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	lister := listJSONLIndexes(fakeEdgeFS{readDir: func(string) ([]fs.DirEntry, error) {
		return []fs.DirEntry{
			fakeDirEntry{name: "s.jsonl"},
			fakeDirEntry{name: "manifest.json"},
			fakeDirEntry{name: "nested", dir: true},
		}, nil
	}})

	paths, err := lister("/chunks")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(paths).To(Equal([]string{"/chunks/s.jsonl"}))

	missing := listJSONLIndexes(fakeEdgeFS{readDir: func(string) ([]fs.DirEntry, error) {
		return nil, fmt.Errorf("read dir: %w", fs.ErrNotExist)
	}})

	paths, err = missing("/gone")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(paths).To(BeEmpty())
}

func TestVaultLockFromLocker_LocksVaultLuhmannLockFileAndReleases(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var lockedPath string

	unlocked := false
	locker := fakeLocker{lock: func(path string) (func() error, error) {
		lockedPath = path

		return func() error { unlocked = true; return nil }, nil
	}}

	lock := vaultLockFromLocker(locker)

	release, err := lock("/vault")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(lockedPath).To(Equal("/vault/.luhmann.lock"))

	release()
	g.Expect(unlocked).To(BeTrue())
}

func TestLogWarningTo_FormatsWithWarningPrefixAndNewline(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var buf bytes.Buffer

	logWarningTo(&buf)("amend: %s failed after %d tries", "embed", 2)
	g.Expect(buf.String()).To(Equal("warning: amend: embed failed after 2 tries\n"))
}

// errInjectedCompose is the sentinel injected by the compose-helper tests.
var errInjectedCompose = errors.New("injected compose failure")

// fakeDirEntry is a minimal fs.DirEntry for ListMD/lister tests.
type fakeDirEntry struct {
	name string
	dir  bool
}

func (f fakeDirEntry) Name() string               { return f.name }
func (f fakeDirEntry) IsDir() bool                { return f.dir }
func (f fakeDirEntry) Type() fs.FileMode          { return 0 }
func (f fakeDirEntry) Info() (fs.FileInfo, error) { return nil, fs.ErrInvalid }

// fakeEdgeFS overrides only the EdgeFS methods a test exercises; calling an
// un-overridden method panics via the embedded nil interface (test bug, loud).
type fakeEdgeFS struct {
	EdgeFS

	readDir  func(string) ([]fs.DirEntry, error)
	readFile func(string) ([]byte, error)
}

func (f fakeEdgeFS) ReadDir(path string) ([]fs.DirEntry, error) { return f.readDir(path) }

func (f fakeEdgeFS) ReadFile(path string) ([]byte, error) { return f.readFile(path) }

// fakeLocker records the locked path.
type fakeLocker struct {
	lock func(string) (func() error, error)
}

func (f fakeLocker) Lock(path string) (func() error, error) { return f.lock(path) }

// silence unused-import when time is only used by other tests in this package.
var _ = time.Now
```

   (Drop the trailing `var _ = time.Now` and the `time` import if the linter flags them — they exist only if a later step in this file needs a clock; expected final state has neither.)

2. [ ] Run `targ test` — expect PASS immediately (the canonical helpers already landed: `vaultLockFromLocker`/`logWarningTo` in T3's deps_compose.go, `newVaultFS` in T5's vault_fs.go, `listJSONLIndexes` in T6's query_chunks.go). There is NO compile-RED in this task because it adds no production code. If any of the four is undefined, an earlier task did not land — STOP and fix the ordering, do NOT declare the missing helper here.

3. [ ] Verify deps_compose.go needs nothing from this task (R1 guard — verify, never edit): `rg -n "func vaultLockFromLocker|func logWarningTo" internal/cli/deps_compose.go` → both present; `rg -n "func newVaultFS" internal/cli/vault_fs.go` → present; `rg -n "func listJSONLIndexes" internal/cli/query_chunks.go` → present. Also confirm no loser symbol was ever declared: `rg -n "edgeVaultFS|jsonlIndexesLister|vaultLuhmannLock|warnLoggerTo" internal/` → zero hits outside this plan's prose (i.e., none in the tree).

4. [ ] (Absorbed into steps 2-3 — no separate GREEN; this task's only production-adjacent artifact is the test-only Deps in step 5.)

5. [ ] Create `internal/cli/testdeps_test.go` (package `cli` — test file, exempt from purity enforcement). Under the composition doctrine this file contains NO adapter implementations: the production `primFS`/`primLocker`/debug-sink impls live in internal (T1-rework) and are reached only through `NewDeps`. The ONLY declaration is `ExportNewTestOsDeps`; its real-OS `Primitives` literal mirrors `cmd/engram/main.go`'s (doctrine flag DRIFT — cli_test.go's end-to-end binary tests guard the production literal; a package-`cli` test file cannot consume cli_test's `realPrimitives()`, hence this second one-screen mirror) minus `StartSignalPulses`, which stays nil so `startForceExit` skips (SIG-1: tests never subscribe process signals). Primitive closures return RAW os errors BY CONTRACT — the `%w` wrap happens exactly once, inside `primFS`/`primLocker`; do NOT add wrapping here (it would double-wrap), and if a linter flags the raw returns in this _test file, escalate rather than wrap. `Deps.Embed` comes out as the NewDeps-wired lazy embedder — production parity with today's `sharedEmbedder`-backed constructors (nothing loads until an embed call); wiring tests needing determinism override the per-command deps field after construction (existing patterns: `fakeEmbedder`, `successEmbedder`). The ADR-0013 concurrent-manifest regression rides `primFS.WriteFileAtomic` — the REAL production dance, not a copy:

```go
package cli

// Test-only composition of production Deps over real OS primitives, so
// wiring-integration tests drive the exact primFS/primLocker/debug-sink
// implementations the binary ships. Composition doctrine (#700): NewDeps is
// the single composition root — no hand-rolled adapter mirrors anywhere.

import (
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// ExportNewTestOsDeps returns production-composed Deps for wiring tests:
// one NewDeps call over an inline real-OS Primitives literal (mirrors
// cmd/engram/main.go's — doctrine flag DRIFT — minus StartSignalPulses,
// nil so startForceExit skips per SIG-1). Closures return raw os errors by
// primitive contract; primFS/primLocker add the single %w wrap.
func ExportNewTestOsDeps() Deps {
	return NewDeps(Primitives{
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
			//nolint:gosec // test helper, path from test
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
		OpenDebugFile: func(path string, perm fs.FileMode) (WriteSyncer, error) {
			return os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, perm) //nolint:gosec // test helper, operator-controlled path
		},
	}, os.Stdout, os.Stderr, func(int) {})
}
```

6. [ ] Run `targ test` then `targ check-full` — expect PASS/clean. Run `targ check-thin-api` — expect PASS (this task adds internal test files only; per Global Constraints any finding escalates to the orchestrator, never suppressed). (`ExportNewTestOsDeps` is unreferenced until M3; if the unused linter flags it, land M2+M3 as one PR — do not suppress.)
7. [ ] Commit:

```
test(cli): NewDeps-composed test Deps + compose-contract tests (#700)

ExportNewTestOsDeps builds Deps through cli.NewDeps over real OS
primitives — the same primFS/primLocker composition the binary ships —
so wiring tests exercise production composition, not a hand-rolled os
adapter mirror (composition doctrine). Fake-driven contract tests cover
the landed canonical helpers (newVaultFS, listJSONLIndexes,
vaultLockFromLocker, logWarningTo). The M2 draft's own helpers were
losers per R1/R2/R3 and were never declared.

AI-Used: [claude]
```

---

### Task T12 (M3): Maintenance-family constructors compose from Deps

**Files**
- Modify: `internal/cli/amend.go`, `internal/cli/resituate.go`, `internal/cli/activate.go`, `internal/cli/vocab_commands.go`
- Modify: `internal/cli/query_chunks.go` (delete the transitional `osListJSONLIndexes` + its `"os"` import — amend.go, converted here, was its last consumer; step 8, grep-gated per R3)
- Modify: `internal/cli/export_test.go`, `internal/cli/activate_test.go`, `internal/cli/amend_test.go`, `internal/cli/resituate_test.go`, `internal/cli/vocab_commands_test.go`, `internal/cli/vocab_trigger_test.go`, `internal/cli/learn_test.go` (one line — cross-cluster, see flag)
- Modify (call expressions only; signature threading owned by wiring cluster): `internal/cli/targets.go`
- Verify-only (no edits): `internal/cli/vocab.go`, `internal/cli/vault_init.go`

**Interfaces**
- Consumes: `Deps` (deps.go), the canonical composition helpers — T5's `newVaultFS(d.FS)`, T3's `vaultLockFromLocker(d.Lock)` + `logWarningTo(d.Stderr)` (deps_compose.go), T6's `listJSONLIndexes(fsys EdgeFS)` curried lister — plus `d.FS.WriteFileAtomic`, `d.Embed embed.Embedder`. (The M2 draft's `edgeVaultFS`/`vaultLuhmannLock`/`warnLoggerTo` are losers per R1/R2 and exist nowhere.)
- Produces: `func newAmendDeps(d Deps) AmendDeps`, `func newResituateDeps(d Deps) ResituateDeps`, `func newActivateDeps(d Deps) ActivateDeps`, `func newVocabDeps(d Deps) VocabDeps`, `func newVocabStatsDeps(d Deps) VocabStatsDeps` — replacing `newOsAmendDeps()`, `newOsResituateDeps()`, `newOsActivateDeps()`, `newOsVocabDeps()`, `newOsVocabStatsDeps()`. Deletes `osWriteSidecar` and (per R3, grep-gated) the transitional `osListJSONLIndexes`.

**Steps**

1. [ ] RED (refactor form — the existing wiring-integration suite IS the safety net; first make the test call sites demand the new signatures): apply these test edits, run `targ test`, expect compile FAIL (undefined new names):
   - activate_test.go:121: `deps := cli.ExportNewOsActivateDeps()` → `deps := cli.ExportNewActivateDeps(cli.ExportNewTestOsDeps())`
   - amend_test.go:818: `deps := cli.ExportNewOsAmendDeps()` → `deps := cli.ExportNewAmendDeps(cli.ExportNewTestOsDeps())`
   - learn_test.go:132: `deps := cli.ExportNewOsAmendDeps()` → `deps := cli.ExportNewAmendDeps(cli.ExportNewTestOsDeps())`
   - vocab_commands_test.go:270, 737, 1020, 3790 and vocab_trigger_test.go:411: `cli.ExportNewOsVocabDeps()` → `cli.ExportNewVocabDeps(cli.ExportNewTestOsDeps())`
   - resituate_test.go:251, 295, 409, 519: `cli.ExportNewOsResituateDeps(successEmbedder{})` → `cli.ExportNewResituateDeps(cli.ExportNewTestOsDeps(), successEmbedder{})`
   - vocab_commands_test.go:737 comment header `// ── Coverage: newOsVocabDeps closures ──…` and the TestNewOsVocabDeps_ClosuresCalled name/doc: rename to `TestNewVocabDeps_ClosuresCalled` / "closures inside newVocabDeps".

2. [ ] export_test.go shims. In the var block (keep alphabetical; these lines currently read as shown), replace:

```go
	ExportNewOsActivateDeps                = newOsActivateDeps
	ExportNewOsAmendDeps                   = newOsAmendDeps
```
with
```go
	ExportNewActivateDeps                  = newActivateDeps
	ExportNewAmendDeps                     = newAmendDeps
```
and
```go
	ExportNewOsVocabDeps                   = newOsVocabDeps
```
with
```go
	ExportNewVocabDeps                     = newVocabDeps
```
(re-sort the block; `ExportNewActivateDeps`/`ExportNewAmendDeps` sort before `ExportNewErrHandler`). Replace the resituate func shim:

```go
// ExportNewOsResituateDeps returns production ResituateDeps with an injected
// embedder so coverage tests can drive Scan/Read/Write without unpacking the
// lazy bundled embedder.
func ExportNewOsResituateDeps(emb embed.Embedder) ResituateDeps {
	deps := newOsResituateDeps()
	deps.Embedder = emb

	return deps
}
```
with
```go
// ExportNewResituateDeps returns production-composed ResituateDeps with an
// injected embedder so coverage tests can drive Scan/Read/Write without
// unpacking the lazy bundled embedder.
func ExportNewResituateDeps(d Deps, emb embed.Embedder) ResituateDeps {
	deps := newResituateDeps(d)
	deps.Embedder = emb

	return deps
}
```

3. [ ] GREEN — amend.go. Replace `newOsAmendDeps` (current code at amend.go:337-378, shown in Files context above) with:

```go
// newAmendDeps composes RunAmend's dependencies from the injected edge Deps
// (pure composition — no direct I/O; #700). ChunksDir flows through
// AmendArgs, not here.
func newAmendDeps(d Deps) AmendDeps {
	const perm = 0o600

	vfs := newVaultFS(d.FS)

	return AmendDeps{
		Lock: vaultLockFromLocker(d.Lock),
		Scan: func(vault string) ([]vaultgraph.Note, error) {
			return vaultgraph.ScanVault(vfs, vault)
		},
		Read: vfs.ReadFile,
		Write: func(path string, data []byte) error {
			err := d.FS.WriteFileAtomic(path, data, perm)
			if err != nil {
				return fmt.Errorf("write %s: %w", path, err)
			}

			return nil
		},
		Embedder:     d.Embed,
		Now:          d.Now,
		LoadChunkIDs: buildChunkIDSet,
		// listJSONLIndexes(d.FS) lists *.jsonl chunk indexes, treats an absent
		// dir as empty (not an error), and never matches manifest.json —
		// exactly the contract the transitional os-backed osListJSONLIndexes
		// provided (deleted in step 8 now that this, its last consumer, flips).
		ListIndexes: listJSONLIndexes(d.FS),
		LogWarning:  logWarningTo(d.Stderr),
		// Vocab assignment wiring: no-op when the vault has no term notes.
		// Uses stored member centroids (vocab.centroids.json) when present,
		// falling back to description embeddings per term.
		LoadTermVectors: func(vault string) ([]TermWithVector, error) {
			return loadAssignmentTermVectors(vault, vfs.ListMD, vfs.ReadFile)
		},
		// ListMD provides full .md filenames for the vocab trigger scan.
		// Must use ListMD (not stripped basenames) — basename filtering causes
		// false-fire on the untagged trigger.
		ListMD: vfs.ListMD,
	}
}
```
Doc-comment touch-ups in the same file: AmendDeps struct comment line 43 `The production wiring in newOsAmendDeps supplies os.ReadDir/os.ReadFile via closures.` → `The production wiring in newAmendDeps supplies the injected EdgeFS via closures.`; Lock field comment line 47 `Wired to vaultFS.Lock in newOsAmendDeps.` → `Wired via vaultLockFromLocker in newAmendDeps.`

4. [ ] resituate.go. Replace `newOsResituateDeps` (resituate.go:155-184) with:

```go
// newResituateDeps composes RunResituate's dependencies from the injected
// edge Deps (pure composition — no direct I/O; #700).
func newResituateDeps(d Deps) ResituateDeps {
	const perm = 0o600

	vfs := newVaultFS(d.FS)

	return ResituateDeps{
		Lock: vaultLockFromLocker(d.Lock),
		Scan: func(vault string) ([]vaultgraph.Note, error) {
			return vaultgraph.ScanVault(vfs, vault)
		},
		Read: vfs.ReadFile,
		Write: func(path string, data []byte) error {
			err := d.FS.WriteFileAtomic(path, data, perm)
			if err != nil {
				return fmt.Errorf("write %s: %w", path, err)
			}

			return nil
		},
		Embedder: d.Embed,
		LoadTermVectors: func(vault string) ([]TermWithVector, error) {
			return loadAssignmentTermVectors(vault, vfs.ListMD, vfs.ReadFile)
		},
		ListMD:     vfs.ListMD,
		LogWarning: logWarningTo(d.Stderr),
		Now:        d.Now,
	}
}
```
ResituateDeps.Lock comment line 28-29 `Wired to vaultFS.Lock in newOsResituateDeps.` → `Wired via vaultLockFromLocker in newResituateDeps.`

5. [ ] activate.go. Delete the `os` import; replace `newOsActivateDeps` + `osWriteSidecar` (activate.go:120-137) with:

```go
// newActivateDeps composes RunActivate's dependencies from the injected edge
// Deps (pure composition — no direct I/O; #700). Sidecar writes go through
// WriteFileAtomic (temp+rename) so concurrent readers always see either the
// old or new file.
func newActivateDeps(d Deps) ActivateDeps {
	const sidecarPerm = 0o600

	return ActivateDeps{
		Lock: vaultLockFromLocker(d.Lock),
		Now:  d.Now,
		Read: d.FS.ReadFile,
		Write: func(path string, data []byte) error {
			return d.FS.WriteFileAtomic(path, data, sidecarPerm)
		},
		LogWarning: logWarningTo(d.Stderr),
	}
}
```
Comment touch-ups: ActivateDeps.Lock comment line 23 `Wired to vaultFS.Lock in newOsActivateDeps.` → `Wired via vaultLockFromLocker in newActivateDeps.`; bumpLastUsed comment lines 86-87 `Sidecar writes go through atomicWriteFile (temp+rename) AND RunActivate holds the vault flock` → `Sidecar writes go through the injected atomic write (WriteFileAtomic, temp+rename) AND RunActivate holds the vault flock`.

6. [ ] vocab_commands.go. Delete the `os` import; replace `newOsVocabDeps` + `newOsVocabStatsDeps` (vocab_commands.go:1208-1240) with (behavior parity: WriteFile/DeleteFile error text preserved; WriteSidecar keeps osEmbedFS.Write's `"write: %w"` wrap):

```go
// newVocabDeps composes VocabDeps from the injected edge Deps (pure
// composition — no direct I/O; #700).
func newVocabDeps(d Deps) VocabDeps {
	const sidecarPerm = 0o600

	vfs := newVaultFS(d.FS)

	return VocabDeps{
		Lock:     vaultLockFromLocker(d.Lock),
		ListMD:   vfs.ListMD,
		ReadFile: vfs.ReadFile,
		WriteFile: func(path string, data []byte) error {
			return d.FS.WriteFileAtomic(path, data, vocabNotePerm)
		},
		DeleteFile: func(path string) error {
			deleteErr := d.FS.Remove(filepath.Clean(path))
			if deleteErr != nil {
				return fmt.Errorf("deleting %s: %w", path, deleteErr)
			}

			return nil
		},
		WriteSidecar: func(path string, data []byte) error {
			err := d.FS.WriteFileAtomic(path, data, sidecarPerm)
			if err != nil {
				return fmt.Errorf("write: %w", err)
			}

			return nil
		},
		Embedder:   d.Embed,
		LogWarning: logWarningTo(d.Stderr),
		Now:        d.Now,
	}
}

// newVocabStatsDeps composes the read-only vocab stats deps from the injected
// edge Deps.
func newVocabStatsDeps(d Deps) VocabStatsDeps {
	vfs := newVaultFS(d.FS)

	return VocabStatsDeps{
		ListMD:   vfs.ListMD,
		ReadFile: vfs.ReadFile,
	}
}
```

- [ ] 6.5. **Migrate the `ExportNewOsVaultFS` call sites (R12 — this task owns the vocab test files).** Replace every `osFS := cli.ExportNewOsVaultFS()` with `osFS := cli.ExportNewVaultFS(osTestEdgeFS{})` (T5's export over the cli_test real-FS EdgeFS double — same `ListMD`/`ReadFile` shape and semantics; `osTestEdgeFS` lives in T5's edgefs_os_test.go, same package cli_test). Sites (verified in the pristine tree; locate by the call expression, not line): vocab_trigger_test.go:251, 441; vocab_commands_test.go:96, 131, 198, 231, 543, 559, 613, 651, 3856, 3874. After the edit: `rg -n "ExportNewOsVaultFS" internal/cli --include='*_test.go'` → hits ONLY in export_test.go (the shim definition, which T7 deletes). Without this step T7's shim deletion is a compile break its gate grep cannot see (R12).

7. [ ] targets.go call-expression updates (coordinate with wiring cluster's `deps Deps` threading through `amendResituateTargets`/`ingestQueryTargets`/`vocabTargets`; only the constructor expressions belong to this task):
   - line 108: `newOsResituateDeps()` → `newResituateDeps(deps)`
   - line 113: `newOsAmendDeps()` → `newAmendDeps(deps)`
   - line 173: `newOsActivateDeps()` → `newActivateDeps(deps)`
   - lines 278/286/290: `newOsVocabDeps()` → `newVocabDeps(deps)`
   - line 282: `newOsVocabStatsDeps()` → `newVocabStatsDeps(deps)`

8. [ ] Delete the transitional lister (per R3 — amend.go, converted in step 3, was its LAST consumer). Gate first: `grep -rn "osListJSONLIndexes" internal/ --include='*.go'` — expected: the definition in query_chunks.go ONLY (step 3 already removed amend.go's reference). Any hit in another file → STOP: that file's task has not landed; do not delete. Then delete `osListJSONLIndexes` (func + doc comment) from query_chunks.go and its now-unused `"os"` import. Verify: re-run the grep — zero hits; `grep -n '"os"\|os\.' internal/cli/query_chunks.go` — no output (query_chunks.go fully pure as of this task).
9. [ ] Run `targ test` — expect PASS: the relocated wiring-integration tests (activate/amend/resituate/vocab against real t.TempDir vaults) prove the composed deps behave identically; resituate tests still inject `successEmbedder{}`; vocab tests still override `deps.Embedder = &fakeEmbedder{}`. The executed targets-level tests riding this task's flips (vocab bootstrap/propose/refit/stats via targ.Execute, activate/resituate/amend via executeForTest) dereference `d.FS` and — on the propose success path — `d.Lock`; both fields are already in `newTestDeps` since T3 (R11), and `Embed` stays nil (vocab's embed path skips on nil Embedder at vocab_commands.go:833).
10. [ ] Purity verification for this cluster (enforcement task lands later; this is the leave-nothing-behind check the central spec demands):
   - `grep -n "\"os\"\|os\.\|syscall\|time\.Now\|time\.Since\|time\.Tick" internal/cli/amend.go internal/cli/resituate.go internal/cli/activate.go internal/cli/vocab.go internal/cli/vocab_commands.go internal/cli/vault_init.go` — expected: NO import of `os`/`syscall`, no `time.Now/Since/Tick` calls; only comment mentions (scrub remaining comment references: amend.go:43 handled in step 3; vocab_commands.go:1126 `os.ReadDir sorts by name` → reword to `the OS-backed lister sorts by name`; resituate.go:128 `wiring provides time.Now` → `wiring provides the injected clock`).
   - Verify-only: vocab.go and vault_init.go unchanged (imports already pure; `fs.FileMode` from io/fs stays per spec).
11. [ ] Run `targ check-full` — expect clean (lint + coverage; the composed constructors are covered by the wiring tests, matching the coverage intent behind the old named `osWriteSidecar`/`logWarningToStderrf` pattern). Run `targ check-thin-api` — expect PASS (`All N public API files are thin wrappers.`); this task adds no cmd/engram declarations, so any finding predates it — escalate per Global Constraints, never suppress.
12. [ ] Commit:

```
refactor(cli): compose maintenance deps from Deps (#700)

newAmendDeps/newResituateDeps/newActivateDeps/newVocabDeps/newVocabStatsDeps
replace their newOsXxx forms: flock via FileLocker (.luhmann.lock at Run*
entry only, ADR-0013), atomic note/sidecar writes via EdgeFS.WriteFileAtomic,
clock via Deps.Now, warnings via Deps.Stderr, embedder via Deps.Embed.
activate.go and vocab_commands.go drop their os imports; vocab.go and
vault_init.go verified already pure. The transitional osListJSONLIndexes
(T6) dies here with query_chunks.go's os import — amend was its last
consumer (grep-gated).

AI-Used: [claude]
```

---

### Task T13 (M4): Purge internal atomic write (gated)

**GATE (do not start until true; per R4 this task runs after T15):** `grep -rn "atomicWriteFile" internal/cli --include="*.go" | grep -v _test | grep -v writesafe.go` returns EMPTY — i.e. every internal caller has been migrated by its own task: learn.go:371 + qa.go:283 (T3), amend.go:351 + resituate.go:169 + activate.go:136 + vocab_commands.go:1217 (T12), cli.go:144 (T4), and embed.go:164 (`osEmbedFS.Write` — deleted by T15, the LAST caller standing, which is why R4 orders T13 after T15).

**Files**
- Delete: `internal/cli/writesafe.go`, `internal/cli/writesafe_test.go`
- Modify: `internal/cli/export_test.go` (remove two shims)
- Verify-only (no edit): `internal/cli/ingest_test.go` (`realFS.write` already repointed by T8 step 6 — step 1 verifies); `internal/cli/edgefs_test.go` + `internal/cli/primitives_integration_test.go` (the surviving atomic-write coverage — step 2 verifies presence, no edits)

**Interfaces**
- Removes: `atomicWriteFile`, `doAtomicWrite`, `ExportAtomicWriteFile`, `ExportDoAtomicWrite` from internal/cli.

**Steps**

1. [ ] Verify the ADR-0013 concurrent-manifest regression infra is already repointed (must survive per spec — T8 step 6 moved `realFS.write` off `cli.ExportAtomicWriteFile` onto its test-local `testAtomicWrite`, which carries the same real temp+rename semantics; no edit here). Check ingest_test.go's `realFS.write` reads:

```go
func (r *realFS) write(_, path string, data []byte) error {
	return testAtomicWrite(path, data, 0o600)
}
```

and `rg -n "ExportAtomicWriteFile" internal/cli/ingest_test.go` → zero hits. Any `ExportAtomicWriteFile` reference remaining outside writesafe_test.go → T8 incomplete, STOP.

2. [ ] Delete `internal/cli/writesafe.go` and `internal/cli/writesafe_test.go`. All five writesafe behaviors live on INTERNALLY against the composed `primFS.WriteFileAtomic` (internal/cli/edgefs.go, landed by T1-rework — the revised doctrine's relocation target; nothing relocates to cmd/engram, which holds only the declaration-free main.go): the fake-prims dance suite (`TestEdgeFS_WriteFileAtomicHappyPathDance`, `TestEdgeFS_WriteFileAtomicFailuresRemoveTemp`, `TestEdgeFS_WriteFileAtomicUniqueNameRetry` — edgefs_test.go) plus the real-primitive suite (`TestRealEdgeFS_WriteFileAtomic*` — primitives_integration_test.go), completed by T10's parity tests per T10's behavior-parity ledger. Verify before deleting: `rg -n "TestRealEdgeFS_WriteFileAtomicWritesNewFile|TestRealEdgeFS_WriteFileAtomicExclCreateFailureLeavesOriginalUntouched|TestRealEdgeFS_WriteFileAtomicRenameFailureCleansTempAndOriginal" internal/cli/primitives_integration_test.go` → all three present (any miss → T10 incomplete, STOP).

3. [ ] In export_test.go, delete the two function shims (lines 204-207 and 331-340 in current numbering):

```go
// ExportAtomicWriteFile exposes atomicWriteFile for writesafe tests.
func ExportAtomicWriteFile(path string, data []byte, perm os.FileMode) error {
	return atomicWriteFile(path, data, perm)
}
```
and
```go
// ExportDoAtomicWrite exposes doAtomicWrite for writesafe tests that need to
// inject a failing rename to cover the rename-error and defer-cleanup paths.
func ExportDoAtomicWrite(
	path string,
	data []byte,
	perm os.FileMode,
	rename func(oldpath, newpath string) error,
) error {
	return doAtomicWrite(path, data, perm, rename)
}
```
(If these were export_test.go's last uses of the `os` import, drop that import too — check compile.)

4. [ ] Verify gate held: `grep -rn "atomicWriteFile\|doAtomicWrite" internal/` — expected EMPTY.
5. [ ] Run `targ test` — expect PASS, including the ingest concurrent-writers regression test (its lock is still real flock via T8's test-local `testFlocker` — R7 — and its writer is T8's `testAtomicWrite`) and the surviving internal atomic-write suites named in step 2.
6. [ ] Run `targ check-full` — expect clean. Run `targ check-thin-api` — expect PASS (this task touches no cmd code; per Global Constraints any finding escalates to the orchestrator, never suppressed).
7. [ ] Commit:

```
refactor(cli): delete internal atomic-write (#700)

writesafe.go's dance now lives solely on internal/cli/edgefs.go's
primFS.WriteFileAtomic, composed from Primitives and covered by the
fake-prims dance suite plus the real-primitive parity suite (T10's
ledger carries all five writesafe behaviors). The ADR-0013
concurrent-manifest regression test writes through T8's test-local
testAtomicWrite with identical temp+rename semantics.

AI-Used: [claude]
```

---

Key file paths: /Users/joe/repos/personal/engram/.claude/worktrees/700-internal-purity/internal/cli/{writesafe.go, writesafe_test.go, edgefs.go, edgefs_test.go, primitives_integration_test.go, amend.go, resituate.go, activate.go, vocab.go, vocab_commands.go, vault_init.go, cli.go, targets.go, export_test.go, ingest_test.go, learn_test.go}, /Users/joe/repos/personal/engram/.claude/worktrees/700-internal-purity/cmd/engram/main.go.

### Task T14 (A): internal/embed purification — Backend/CacheFS composed internally; thin hugot Runtime at the cmd edge (doctrine D-1)

**Doctrine note (BINDING — this rework applies the revised composition doctrine; it supersedes the pre-correction embed draft):** cmd/engram contributes ONLY thin declarations for the embedder path: one EMPTY struct `hugotRuntime` whose two methods are checker-verified thin shapes, plus one field line in `cmd/engram/main.go`'s `cli.Primitives` literal. ALL orchestration — session→pipeline lifecycle with destroy-on-failure, pipeline config policy (`engram-embed` / `model.onnx`), cache extraction, sentinel policy, permission policy, exist-error classification, and every `%w` wrap — lives in `internal/embed`, parameterized over injected capabilities. `Deps.Embed` stays wired INSIDE `cli.NewDeps` (R6/D-1); this task flips that line to the 3-arg constructor. Task-local design flags (within D-1's stated latitude — the doctrine's supersession map delegates "the exact field shapes" to this brief):

- **E-1 (runtime erasure shape — checker-derived):** `Runtime.NewPipeline` returns the pipeline's run function (`RunPipelineFunc`) as a closure over the concrete hugot pipeline, instead of an opaque handle that a separate `Run` method re-asserts. Reason, verified against targ's `checkFuncThinness`/`isSimpleErrorWrapper` source: the wrapper pattern requires statement 1's RHS to be a call whose receiver is a bare identifier — `pipeline.(*T).RunPipeline(...)` (type-asserted receiver) provably FAILS the gate, so an empty-struct `Run(pipeline any, ...)` mapping method is impossible. The closure form passes: `NewPipeline`'s body is the sanctioned 3-statement wrapper (`x, err := hugot.NewPipeline(...)`, `if err != nil`, `return`), whose third statement need only BE a return; the returned closure is doctrine-capped at a trivially-sequenced single-call body (call on the captured ident, err-check, selector return). No `any`, no re-assertion, no `pipelines` import in cmd.
- **E-2 (no new cache fields on Primitives):** D-1 names "backend/cache capability fields"; the cache side needs NO new fields — the T1-rework `cli.Primitives` FS fields (`Stat`, `MkdirAll`, `MkdirTemp`, `WriteFile`, `Rename`, `RemoveAll`) already carry every raw capability the cache composition needs, and `cli.NewDeps` forwards them into `embed.CacheFSPrims` verbatim. Only ONE new Primitives field lands: `EmbedRuntime embed.Runtime`.
- **E-3 (exist-classification moves internal, os-free):** the old `isExistErr`/`renameIsExist` sniffing cannot live in cmd (multi-statement) nor import `os` in internal. Internal `renameIsExist` uses `errors.Is(err, fs.ErrExist)` — which already covers EEXIST and, via `syscall.Errno.Is`'s ENOTEMPTY mapping, the macOS dir-over-dir case through `*os.LinkError`'s `Unwrap` — plus a `strings.Contains` fallback on the message ("file exists" / "directory not empty") preserving the previous defensive sniffing. The real-OS integration test (rename onto populated dir) keeps this honest on the actual platform.

**Files**

- Create: `internal/embed/runtime.go` (Runtime seam + `NewRuntimeBackend` composition), `internal/embed/cachefs.go` (`CacheFSPrims` + `NewCacheFS` composition), `internal/embed/runtime_test.go`, `internal/embed/cachefs_test.go`, `internal/embed/cachefs_integration_test.go` (real-os `_test` — sanctioned by the purity lint's `!$test` exclusion)
- Modify: `internal/embed/hugot.go` (full rewrite below)
- Modify: `internal/embed/cache.go` (full rewrite below)
- Modify: `internal/embed/export_test.go` (full rewrite below)
- Modify: `internal/embed/hugot_test.go`, `internal/embed/cache_test.go`, `internal/embed/buildembedder_test.go`, `internal/embed/overlength_test.go`, `internal/embed/embedder_fake_test.go`
- Delete: `internal/embed/production_cache_test.go`, `internal/embed/production_hugot_test.go`, `internal/embed/unpack_test.go`, `internal/embed/tempfs_test.go`
- Create: `cmd/engram/hugot.go` (THIN: empty `hugotRuntime` struct + two thin methods, NOTHING else), `cmd/engram/hugot_test.go` (the sanctioned cmd wiring-smoke tests — `_test.go` is exempt from `check-thin-api`), `cmd/engram/testdata/model-stub.txt`
- Modify: `internal/cli/primitives.go` (Primitives gains `EmbedRuntime embed.Runtime`; `NewDeps`'s guarded Embed line → 3-arg composition — the R6 arity flip lands HERE, not in cmd), `internal/cli/embed.go` (sharedEmbedder → bridge; delete `modelCacheDir`), `internal/cli/targets.go` (wire bridge), `internal/cli/export_test.go` (bridge export), `cmd/engram/main.go` (Primitives literal gains ONE line: `EmbedRuntime: hugotRuntime{},`)
- Create: `internal/cli/embed_bridge_test.go`

**Interfaces**

- Produces (internal/embed): `embed.Backend` — `OpenPipeline(ctx context.Context, modelDir string) (PipelineHandle, error)`; `embed.PipelineHandle` — `RunPipeline(ctx context.Context, inputs []string) (FeatureOutput, error)`, `Destroy() error`; `embed.FeatureOutput{ Embeddings [][]float32 }`; `embed.CacheFS` (exported rename of `cacheFS`, same 7 methods, Rename contract = `errors.Is(err, fs.ErrExist)`); `embed.RawSession` — `Destroy() error`; `embed.RunPipelineFunc func(ctx context.Context, inputs []string) ([][]float32, error)`; `embed.Runtime` — `NewSession(ctx context.Context) (RawSession, error)`, `NewPipeline(session RawSession, modelPath, name, onnxFilename string) (RunPipelineFunc, error)`; `embed.NewRuntimeBackend(runtime Runtime) Backend`; `embed.CacheFSPrims` (six func fields with signatures identical to the matching `cli.Primitives` fields); `embed.NewCacheFS(prims CacheFSPrims) CacheFS`; `embed.ErrRuntimeMissing`; `embed.BundledModelFS() stdembed.FS`; `embed.BundledModelDir = "assets/model"`; new constructor signatures `NewBundledHugotEmbedder(ctx, backend Backend, cfs CacheFS, cacheDir string)`, `NewHugotEmbedderFromDir(ctx, backend Backend, modelDir, modelID string)`, `NewHugotEmbedderFromFS(ctx, backend Backend, cfs CacheFS, modelFS stdembed.FS, modelDir, modelID, cacheDir string)`, `NewLazyEmbedder(backend Backend, cfs CacheFS, cacheDir string)`.
- Produces (internal/cli): `Primitives.EmbedRuntime embed.Runtime` field; the 3-arg NewDeps Embed composition; `wireSharedEmbedder(embed.Embedder)` (unexported, called from `Targets`).
- Produces (cmd/engram): `hugotRuntime` — EMPTY struct implementing `embed.Runtime` with exactly two thin methods; NO other new declarations in package main.
- Consumes: T2's landed `NewDeps` guarded Embed wiring (`internal/cli/primitives.go` — exact before-text in step 9); `cli.CacheDirFromHome(home, modelID string, getenv func(string) string) string` (targets.go:56, unchanged); foundation's `cli.Deps.Embed embed.Embedder` field; `hugot.NewGoSession` / `hugot.NewPipeline` / `hugot.FeatureExtractionConfig` (cmd `_test` and `cmd/engram/hugot.go` only).

**Steps**

- [ ] 1. **RED — internal composition tests first.** Create `internal/embed/runtime_test.go` (unit tests of the internally-composed backend lifecycle over a fake Runtime — these are the relocated semantics of the old cmd backend-branch tests):

```go
package embed_test

import (
	"context"
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/embed"
)

// fakeRuntime scripts the raw runtime seam so every branch of the
// internally-composed backend is unit-testable without hugot.
type fakeRuntime struct {
	sessionErr     error
	pipelineErr    error
	destroyErr     error
	destroyCalls   int
	pipelineCalled bool
	runFn          embed.RunPipelineFunc
	gotModelPath   string
	gotName        string
	gotOnnxFile    string
}

func (f *fakeRuntime) NewPipeline(
	_ embed.RawSession, modelPath, name, onnxFilename string,
) (embed.RunPipelineFunc, error) {
	f.pipelineCalled = true
	f.gotModelPath = modelPath
	f.gotName = name
	f.gotOnnxFile = onnxFilename

	if f.pipelineErr != nil {
		return nil, f.pipelineErr
	}

	return f.runFn, nil
}

func (f *fakeRuntime) NewSession(context.Context) (embed.RawSession, error) {
	if f.sessionErr != nil {
		return nil, f.sessionErr
	}

	return fakeRuntimeSession{runtime: f}, nil
}

type fakeRuntimeSession struct{ runtime *fakeRuntime }

func (s fakeRuntimeSession) Destroy() error {
	s.runtime.destroyCalls++

	return s.runtime.destroyErr
}

// TestRuntimeBackend_NilRuntimeFailsLoud asserts a Deps carrier built from
// Primitives without EmbedRuntime surfaces a clear error, never a panic.
func TestRuntimeBackend_NilRuntimeFailsLoud(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	_, err := embed.NewRuntimeBackend(nil).OpenPipeline(t.Context(), "/tmp/x")
	g.Expect(err).To(MatchError(embed.ErrRuntimeMissing))
}

// TestRuntimeBackend_SessionFailPropagates exercises the first error branch
// of the composed OpenPipeline: NewSession returns an error.
func TestRuntimeBackend_SessionFailPropagates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	sessionErr := errors.New("session blocked")
	runtime := &fakeRuntime{sessionErr: sessionErr}

	_, err := embed.NewRuntimeBackend(runtime).OpenPipeline(t.Context(), "/tmp/x")
	g.Expect(err).To(MatchError(ContainSubstring("hugot session")))
	g.Expect(err).To(MatchError(ContainSubstring("session blocked")))
	g.Expect(runtime.pipelineCalled).To(BeFalse(),
		"NewPipeline must not be called when NewSession fails")
}

// TestRuntimeBackend_PipelineFailDestroysSession exercises the second error
// branch: NewPipeline fails and the session's Destroy is called.
func TestRuntimeBackend_PipelineFailDestroysSession(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	pipeErr := errors.New("pipeline blocked")
	runtime := &fakeRuntime{pipelineErr: pipeErr}

	_, err := embed.NewRuntimeBackend(runtime).OpenPipeline(t.Context(), "/tmp/x")
	g.Expect(err).To(MatchError(ContainSubstring("hugot pipeline")))
	g.Expect(err).To(MatchError(ContainSubstring("pipeline blocked")))
	g.Expect(runtime.destroyCalls).
		To(Equal(1), "session.Destroy must be called on pipeline failure")
}

// TestRuntimeBackend_ConfigPolicyIsInternal proves the pipeline config
// policy (name, onnx filename) lives internal: the raw runtime receives
// the values without cmd declaring them.
func TestRuntimeBackend_ConfigPolicyIsInternal(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	runtime := &fakeRuntime{
		runFn: func(context.Context, []string) ([][]float32, error) {
			return [][]float32{{1}}, nil
		},
	}

	_, err := embed.NewRuntimeBackend(runtime).OpenPipeline(t.Context(), "/models/m1")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(runtime.gotModelPath).To(Equal("/models/m1"))
	g.Expect(runtime.gotName).To(Equal("engram-embed"))
	g.Expect(runtime.gotOnnxFile).To(Equal("model.onnx"))
}

// TestRuntimeBackend_RunMapsOutput drives the happy path through the
// returned handle: raw [][]float32 maps into embed.FeatureOutput.
func TestRuntimeBackend_RunMapsOutput(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	runtime := &fakeRuntime{
		runFn: func(context.Context, []string) ([][]float32, error) {
			return [][]float32{{1, 2}}, nil
		},
	}

	handle, err := embed.NewRuntimeBackend(runtime).OpenPipeline(t.Context(), "/tmp/x")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	out, runErr := handle.RunPipeline(t.Context(), []string{"hello"})
	g.Expect(runErr).NotTo(HaveOccurred())
	g.Expect(out.Embeddings).To(Equal([][]float32{{1, 2}}))
}

// TestRuntimeBackend_RunErrorPropagates exercises the run error branch of
// the composed handle.
func TestRuntimeBackend_RunErrorPropagates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	runErr := errors.New("run blocked")
	runtime := &fakeRuntime{
		runFn: func(context.Context, []string) ([][]float32, error) {
			return nil, runErr
		},
	}

	handle, err := embed.NewRuntimeBackend(runtime).OpenPipeline(t.Context(), "/tmp/x")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	_, err = handle.RunPipeline(t.Context(), []string{"hello"})
	g.Expect(err).To(MatchError(ContainSubstring("hugot run")))
	g.Expect(err).To(MatchError(ContainSubstring("run blocked")))
}

// TestRuntimeBackend_DestroyErrorPropagates exercises the error branch of
// the composed handle's Destroy.
func TestRuntimeBackend_DestroyErrorPropagates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	destroyErr := errors.New("destroy blocked")
	runtime := &fakeRuntime{
		destroyErr: destroyErr,
		runFn: func(context.Context, []string) ([][]float32, error) {
			return [][]float32{{1}}, nil
		},
	}

	handle, err := embed.NewRuntimeBackend(runtime).OpenPipeline(t.Context(), "/tmp/x")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	err = handle.Destroy()
	g.Expect(err).To(MatchError(ContainSubstring("hugot session destroy")))
	g.Expect(err).To(MatchError(ContainSubstring("destroy blocked")))
}
```

Create `internal/embed/cachefs_test.go` (unit tests of the composed CacheFS over fake primitives — sentinel policy, permission policy, wraps, and the exist contract; the relocated semantics of the old cmd `osCacheFS` method tests):

```go
package embed_test

import (
	"errors"
	"io/fs"
	"os"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/embed"
)

// cachePrimRecorder scripts and records the raw-primitive calls the
// composed CacheFS makes, so sentinel/permission policy and error wraps
// are assertable without a real disk. Each (sub)test builds its own
// recorder — no shared mutable state across parallel tests.
type cachePrimRecorder struct {
	statPath     string
	statErr      error
	mkdirAllPath string
	mkdirAllPerm fs.FileMode
	mkdirAllErr  error
	mkdirTempErr error
	writePath    string
	writeData    []byte
	writePerm    fs.FileMode
	writeErr     error
	renameErr    error
	removeAllErr error
}

func (r *cachePrimRecorder) prims() embed.CacheFSPrims {
	return embed.CacheFSPrims{
		Stat: func(path string) (fs.FileInfo, error) {
			r.statPath = path

			return nil, r.statErr
		},
		MkdirAll: func(path string, perm fs.FileMode) error {
			r.mkdirAllPath = path
			r.mkdirAllPerm = perm

			return r.mkdirAllErr
		},
		MkdirTemp: func(_, _ string) (string, error) {
			return "/tmp/fake-extract", r.mkdirTempErr
		},
		WriteFile: func(path string, data []byte, perm fs.FileMode) error {
			r.writePath = path
			r.writeData = data
			r.writePerm = perm

			return r.writeErr
		},
		Rename: func(_, _ string) error {
			return r.renameErr
		},
		RemoveAll: func(_ string) error {
			return r.removeAllErr
		},
	}
}

// TestCacheFS_StatSentinel covers the sentinel-probe branches and proves
// the ".complete" sentinel name is internal policy (the raw Stat sees the
// joined path).
func TestCacheFS_StatSentinel(t *testing.T) {
	t.Parallel()

	t.Run("missing sentinel is false, nil", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		recorder := &cachePrimRecorder{statErr: fs.ErrNotExist}

		present, err := embed.NewCacheFS(recorder.prims()).StatSentinel("/cache/m1")
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(present).To(BeFalse())
		g.Expect(recorder.statPath).To(Equal("/cache/m1/.complete"))
	})

	t.Run("stat failure wraps", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		recorder := &cachePrimRecorder{statErr: errors.New("disk gone")}

		_, err := embed.NewCacheFS(recorder.prims()).StatSentinel("/cache/m1")
		g.Expect(err).To(MatchError(ContainSubstring("stat sentinel")))
		g.Expect(err).To(MatchError(ContainSubstring("disk gone")))
	})

	t.Run("present sentinel is true, nil", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		recorder := &cachePrimRecorder{}

		present, err := embed.NewCacheFS(recorder.prims()).StatSentinel("/cache/m1")
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(present).To(BeTrue())
	})
}

// TestCacheFS_MkdirAll asserts the internal dir-perm policy (0o755) and
// the error wrap.
func TestCacheFS_MkdirAll(t *testing.T) {
	t.Parallel()

	t.Run("passes 0o755 and succeeds", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		recorder := &cachePrimRecorder{}

		g.Expect(embed.NewCacheFS(recorder.prims()).MkdirAll("/cache")).To(Succeed())
		g.Expect(recorder.mkdirAllPath).To(Equal("/cache"))
		g.Expect(recorder.mkdirAllPerm).To(Equal(fs.FileMode(0o755)))
	})

	t.Run("failure wraps", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		recorder := &cachePrimRecorder{mkdirAllErr: errors.New("denied")}

		err := embed.NewCacheFS(recorder.prims()).MkdirAll("/cache")
		g.Expect(err).To(MatchError(ContainSubstring("mkdir all")))
		g.Expect(err).To(MatchError(ContainSubstring("denied")))
	})
}

// TestCacheFS_MkdirTemp covers passthrough and wrap.
func TestCacheFS_MkdirTemp(t *testing.T) {
	t.Parallel()

	t.Run("returns the created dir", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		recorder := &cachePrimRecorder{}

		tmp, err := embed.NewCacheFS(recorder.prims()).MkdirTemp("/cache", ".tmp-*")
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(tmp).To(Equal("/tmp/fake-extract"))
	})

	t.Run("failure wraps", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		recorder := &cachePrimRecorder{mkdirTempErr: errors.New("full")}

		_, err := embed.NewCacheFS(recorder.prims()).MkdirTemp("/cache", ".tmp-*")
		g.Expect(err).To(MatchError(ContainSubstring("mkdir temp")))
	})
}

// TestCacheFS_WriteFile asserts the internal file-perm policy (0o600) and
// the error wrap.
func TestCacheFS_WriteFile(t *testing.T) {
	t.Parallel()

	t.Run("passes 0o600 and succeeds", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		recorder := &cachePrimRecorder{}

		g.Expect(embed.NewCacheFS(recorder.prims()).WriteFile("/tmp/x/model.onnx", []byte("m"))).
			To(Succeed())
		g.Expect(recorder.writePath).To(Equal("/tmp/x/model.onnx"))
		g.Expect(recorder.writePerm).To(Equal(fs.FileMode(0o600)))
	})

	t.Run("failure wraps", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		recorder := &cachePrimRecorder{writeErr: errors.New("denied")}

		err := embed.NewCacheFS(recorder.prims()).WriteFile("/tmp/x/model.onnx", []byte("m"))
		g.Expect(err).To(MatchError(ContainSubstring("write file")))
	})
}

// TestCacheFS_WriteSentinel proves the sentinel write is an empty
// ".complete" file under the internal perm policy.
func TestCacheFS_WriteSentinel(t *testing.T) {
	t.Parallel()

	t.Run("writes empty .complete", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		recorder := &cachePrimRecorder{}

		g.Expect(embed.NewCacheFS(recorder.prims()).WriteSentinel("/tmp/extract")).To(Succeed())
		g.Expect(recorder.writePath).To(Equal("/tmp/extract/.complete"))
		g.Expect(recorder.writeData).To(BeEmpty())
		g.Expect(recorder.writePerm).To(Equal(fs.FileMode(0o600)))
	})

	t.Run("failure wraps", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		recorder := &cachePrimRecorder{writeErr: errors.New("denied")}

		err := embed.NewCacheFS(recorder.prims()).WriteSentinel("/tmp/extract")
		g.Expect(err).To(MatchError(ContainSubstring("write sentinel")))
	})
}

// TestCacheFS_RenameExistContract pins the load-bearing contract: every
// destination-exists flavor the raw primitive can produce must surface as
// errors.Is(err, fs.ErrExist); everything else wraps without the sentinel.
func TestCacheFS_RenameExistContract(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		raw       error
		wantExist bool
	}{
		"raw fs.ErrExist":                  {raw: fs.ErrExist, wantExist: true},
		"LinkError wrapping ErrExist":      {raw: &os.LinkError{Op: "rename", Old: "a", New: "b", Err: os.ErrExist}, wantExist: true},
		"LinkError directory not empty":    {raw: &os.LinkError{Op: "rename", Old: "a", New: "b", Err: errors.New("directory not empty")}, wantExist: true},
		"bare directory-not-empty message": {raw: errors.New("rename a b: directory not empty"), wantExist: true},
		"unrelated error":                  {raw: os.ErrPermission, wantExist: false},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			recorder := &cachePrimRecorder{renameErr: testCase.raw}

			err := embed.NewCacheFS(recorder.prims()).Rename("/tmp/src", "/cache/dst")
			g.Expect(err).To(HaveOccurred())
			g.Expect(errors.Is(err, fs.ErrExist)).To(Equal(testCase.wantExist))

			if !testCase.wantExist {
				g.Expect(err).To(MatchError(ContainSubstring("rename")))
			}
		})
	}

	t.Run("success is nil", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		recorder := &cachePrimRecorder{}

		g.Expect(embed.NewCacheFS(recorder.prims()).Rename("/tmp/src", "/cache/dst")).To(Succeed())
	})
}

// TestCacheFS_RemoveAllPassesThroughRaw pins the nil-on-missing contract:
// the raw primitive's error (or nil) passes through unwrapped.
func TestCacheFS_RemoveAllPassesThroughRaw(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(embed.NewCacheFS((&cachePrimRecorder{}).prims()).RemoveAll("/tmp/x")).To(Succeed())

	rawErr := errors.New("busy")
	recorder := &cachePrimRecorder{removeAllErr: rawErr}

	err := embed.NewCacheFS(recorder.prims()).RemoveAll("/tmp/x")
	g.Expect(err).To(MatchError(rawErr))
}

```

(The old cmd `TestRenameIsExist` branch matrix is absorbed by `TestCacheFS_RenameExistContract` above — the classifier is now an unexported internal helper exercised through the composed `Rename`.)

Create `internal/embed/cachefs_integration_test.go` (REAL os functions through the composition — sanctioned in internal `_test` files; this carries the extraction + platform-quirk coverage the old cmd adapter tests held):

```go
package embed_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/embed"
)

// errBackendUnused aborts embedder construction after extraction so
// extraction-only behavior is assertable without a hugot runtime.
var errBackendUnused = errors.New("backend intentionally failing")

// failingBackend implements embed.Backend and always refuses to open.
type failingBackend struct{}

func (failingBackend) OpenPipeline(context.Context, string) (embed.PipelineHandle, error) {
	return nil, errBackendUnused
}

// realCacheFSForTest builds the production CacheFS composition over the
// raw os functions — the same wiring cli.NewDeps performs from
// cli.Primitives.
func realCacheFSForTest() embed.CacheFS {
	return embed.NewCacheFS(embed.CacheFSPrims{
		Stat:      os.Stat,
		MkdirAll:  os.MkdirAll,
		MkdirTemp: os.MkdirTemp,
		WriteFile: os.WriteFile,
		Rename:    os.Rename,
		RemoveAll: os.RemoveAll,
	})
}

// TestCacheFS_RealOS_SentinelRoundTrip proves sentinel write + probe
// against a real tempdir.
func TestCacheFS_RealOS_SentinelRoundTrip(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cfs := realCacheFSForTest()
	dir := t.TempDir()

	present, err := cfs.StatSentinel(dir)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(present).To(BeFalse())

	g.Expect(cfs.WriteSentinel(dir)).To(Succeed())

	_, statErr := os.Stat(filepath.Join(dir, ".complete"))
	g.Expect(statErr).NotTo(HaveOccurred())

	present, err = cfs.StatSentinel(dir)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(present).To(BeTrue())
}

// TestCacheFS_RealOS_RenameOntoPopulatedDir keeps the exist-classification
// honest on the actual OS: on macOS the raw rename error is ENOTEMPTY, and
// the composed Rename must still satisfy the fs.ErrExist contract.
func TestCacheFS_RealOS_RenameOntoPopulatedDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	parent := t.TempDir()
	src := filepath.Join(parent, "src")
	dst := filepath.Join(parent, "dst")
	g.Expect(os.Mkdir(src, 0o755)).To(Succeed())
	g.Expect(os.Mkdir(dst, 0o755)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(dst, "f.txt"), []byte("hi"), 0o600)).To(Succeed())

	err := realCacheFSForTest().Rename(src, dst)
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, fs.ErrExist)).To(BeTrue(),
		"CacheFS.Rename contract: destination-exists must satisfy errors.Is(err, fs.ErrExist)")
}

// TestExtractToCache_RealOS drives the internal extraction through the
// composed CacheFS on real disk: first call extracts and stamps the
// sentinel; second call reuses without re-extracting. The injected backend
// fails so no hugot runtime is needed (extraction happens before the
// backend opens). nonEmptyTestFS is declared in cache_test.go (same
// embed_test package; its move from unpack_test.go happens in step 8).
func TestExtractToCache_RealOS(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cacheDir := filepath.Join(t.TempDir(), "models", "stub@1")

	_, err := embed.NewHugotEmbedderFromFS(
		t.Context(), failingBackend{}, realCacheFSForTest(),
		nonEmptyTestFS, "testdata", "stub@1", cacheDir)
	g.Expect(err).To(MatchError(errBackendUnused))

	_, sentinelErr := os.Stat(filepath.Join(cacheDir, ".complete"))
	g.Expect(sentinelErr).NotTo(HaveOccurred(),
		".complete sentinel must be written after first extraction")

	entries1, readErr1 := os.ReadDir(cacheDir)
	g.Expect(readErr1).NotTo(HaveOccurred())

	fileCount1 := len(entries1)
	g.Expect(fileCount1).To(BeNumerically(">", 1), "cache dir must contain model files + sentinel")

	_, err = embed.NewHugotEmbedderFromFS(
		t.Context(), failingBackend{}, realCacheFSForTest(),
		nonEmptyTestFS, "testdata", "stub@1", cacheDir)
	g.Expect(err).To(MatchError(errBackendUnused))

	entries2, readErr2 := os.ReadDir(cacheDir)
	g.Expect(readErr2).NotTo(HaveOccurred())
	g.Expect(entries2).To(HaveLen(fileCount1),
		"second call must not add/modify files — cache reused as-is")
}
```

(This replaces the old `internal/embed/production_cache_test.go` real-OS coverage and the deleted cmd-side extract test; the ADR-adjacent concurrent-race branches stay covered by cache_test.go's fake-driven race tests.)

- [ ] 2. **RED — cmd wiring-smoke tests.** Create `cmd/engram/testdata/model-stub.txt` containing `stub model payload` (one line). Create `cmd/engram/hugot_test.go` — the sanctioned cmd smoke suite (`_test.go` files are exempt from `check-thin-api`); it drives the REAL `hugotRuntime` through the internally-composed backend, which is the only direct coverage the thin cmd type gets (the production Primitives literal itself is guarded by cli_test's end-to-end binary tests, doctrine flag DRIFT):

```go
package main

import (
	stdembed "embed"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/embed"
)

//go:embed testdata
var testModelFS stdembed.FS

// realCacheFS mirrors the CacheFSPrims wiring cli.NewDeps builds from the
// production Primitives literal.
func realCacheFS() embed.CacheFS {
	return embed.NewCacheFS(embed.CacheFSPrims{
		Stat:      os.Stat,
		MkdirAll:  os.MkdirAll,
		MkdirTemp: os.MkdirTemp,
		WriteFile: os.WriteFile,
		Rename:    os.Rename,
		RemoveAll: os.RemoveAll,
	})
}

// TestBundledEmbedder_Smoke exercises the full production wiring
// end-to-end: real hugot runtime (cmd's thin hugotRuntime), internally
// composed backend + cache FS, bundled model assets. Skipped under -short
// because it unpacks the ~90MB ONNX.
func TestBundledEmbedder_Smoke(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping bundled-embedder smoke test under -short")
	}

	t.Parallel()

	g := NewWithT(t)

	embedder, err := embed.NewBundledHugotEmbedder(
		t.Context(), embed.NewRuntimeBackend(hugotRuntime{}), realCacheFS(),
		filepath.Join(t.TempDir(), "model-cache"))
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	defer func() {
		_ = embedder.Close()
	}()

	const expectedDims = 384

	g.Expect(embedder.ModelID()).To(Equal(embed.BundledModelID))
	g.Expect(embedder.Dims()).To(Equal(expectedDims))

	vec, embErr := embedder.Embed(t.Context(), "hello world")
	g.Expect(embErr).NotTo(HaveOccurred())
	g.Expect(vec).To(HaveLen(expectedDims))

	const longLen = 4000

	longText := make([]byte, longLen)
	for i := range longText {
		longText[i] = 'a' + byte(i%26)
	}

	vec2, embErr2 := embedder.Embed(t.Context(), string(longText))
	g.Expect(embErr2).NotTo(HaveOccurred())
	g.Expect(vec2).To(HaveLen(expectedDims))
}

// TestHugotRejectsInvalidModelDir exercises the embedder-construction
// error branch through the real runtime: extraction succeeds (files
// exist) but hugot rejects the directory because it has no valid
// model.onnx.
func TestHugotRejectsInvalidModelDir(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cacheDir := filepath.Join(t.TempDir(), "model-cache")
	_, err := embed.NewHugotEmbedderFromFS(
		t.Context(), embed.NewRuntimeBackend(hugotRuntime{}), realCacheFS(),
		testModelFS, "testdata", "fake@1", cacheDir)
	g.Expect(err).To(HaveOccurred())
}
```

Run: `targ test` — expected RED (compile failures: `embed.Runtime`, `embed.RunPipelineFunc`, `embed.RawSession`, `embed.NewRuntimeBackend`, `embed.CacheFSPrims`, `embed.NewCacheFS`, `embed.ErrRuntimeMissing`, `hugotRuntime`, and the new `embed.*` constructor arities don't exist yet).

- [ ] 3. **GREEN — create `internal/embed/runtime.go`** (the Runtime seam + the internally-composed production Backend; this is where the old cmd `hugotBackend`/`hugotPipeline` orchestration now lives):

```go
package embed

import (
	"context"
	"errors"
	"fmt"
)

// Exported variables.
var (
	// ErrRuntimeMissing reports an embed attempt through a backend composed
	// from a nil Runtime (a Deps carrier whose Primitives had no
	// EmbedRuntime wired). Production main.go always wires one; minimal
	// test Primitives may not — this surfaces that as a clean lazy-init
	// error instead of a panic.
	ErrRuntimeMissing = errors.New(
		"embed runtime not wired: cli.Primitives.EmbedRuntime is required for embedding")
)

// RawSession is the minimal runtime-session surface the composed backend
// needs: cleanup on pipeline-creation failure and on normal close.
// *hugot.Session satisfies it structurally.
type RawSession interface {
	Destroy() error
}

// RunPipelineFunc runs an opened embedding pipeline on inputs and returns
// one vector per input. cmd's Runtime.NewPipeline returns one as a closure
// over the concrete pipeline, erasing the runtime's types without any
// re-assertion at call time (doctrine flag E-1).
type RunPipelineFunc func(ctx context.Context, inputs []string) ([][]float32, error)

// Runtime is the raw model-runtime capability surface. The production
// implementation is cmd/engram's EMPTY hugotRuntime struct whose two
// methods are single-call bodies (targ check-thin-api); ALL lifecycle
// orchestration and config policy live here, behind NewRuntimeBackend
// (#700, doctrine flag D-1).
type Runtime interface {
	// NewSession opens a runtime session.
	NewSession(ctx context.Context) (RawSession, error)
	// NewPipeline opens a feature-extraction pipeline for the model at
	// modelPath on session and returns its run function.
	NewPipeline(session RawSession, modelPath, name, onnxFilename string) (RunPipelineFunc, error)
}

// NewRuntimeBackend composes the production Backend from a raw Runtime:
// the open-session → open-pipeline → destroy-on-failure lifecycle, the
// pipeline config policy, and all error wrapping happen here, internally.
func NewRuntimeBackend(runtime Runtime) Backend {
	return runtimeBackend{runtime: runtime}
}

// unexported constants.
const (
	// pipelineName and pipelineOnnxFilename are the feature-extraction
	// pipeline config policy — kept internal so cmd passes values through
	// without declaring any constants (thin-api).
	pipelineName         = "engram-embed"
	pipelineOnnxFilename = "model.onnx"
)

// runtimeBackend implements Backend over a raw Runtime.
type runtimeBackend struct {
	runtime Runtime
}

// OpenPipeline opens a session, then a feature-extraction pipeline on it,
// destroying the session if pipeline creation fails.
func (b runtimeBackend) OpenPipeline(
	ctx context.Context, modelDir string,
) (PipelineHandle, error) {
	if b.runtime == nil {
		return nil, ErrRuntimeMissing
	}

	session, err := b.runtime.NewSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("hugot session: %w", err)
	}

	run, pipeErr := b.runtime.NewPipeline(session, modelDir, pipelineName, pipelineOnnxFilename)
	if pipeErr != nil {
		_ = session.Destroy()

		return nil, fmt.Errorf("hugot pipeline: %w", pipeErr)
	}

	return &runtimePipeline{session: session, run: run}, nil
}

// runtimePipeline pairs a pipeline run function with the session that owns
// it so Destroy releases both together.
type runtimePipeline struct {
	session RawSession
	run     RunPipelineFunc
}

// Destroy releases the owning session (which owns the pipeline).
func (p *runtimePipeline) Destroy() error {
	err := p.session.Destroy()
	if err != nil {
		return fmt.Errorf("hugot session destroy: %w", err)
	}

	return nil
}

// RunPipeline runs the model and maps the raw vectors into the
// runtime-neutral FeatureOutput shape.
func (p *runtimePipeline) RunPipeline(
	ctx context.Context, inputs []string,
) (FeatureOutput, error) {
	out, err := p.run(ctx, inputs)
	if err != nil {
		return FeatureOutput{}, fmt.Errorf("hugot run: %w", err)
	}

	return FeatureOutput{Embeddings: out}, nil
}
```

- [ ] 4. **Rewrite `internal/embed/hugot.go`** — hugot and os imports leave internal; `Backend`/`PipelineHandle`/`FeatureOutput` exported; constructors take injected seams; `buildEmbedder` folds into `NewHugotEmbedderFromDir`; dead `tempFS`/`productionTempFS`/`unpackModelToTemp` deleted. Full replacement:

```go
package embed

import (
	"context"
	stdembed "embed"
	"errors"
	"fmt"
	"sync"
)

// Exported constants.
const (
	BundledModelID = "minilm-l6-v2@384"
	// BundledModelDir is the directory inside BundledModelFS holding the
	// bundled model files.
	BundledModelDir = "assets/model"
)

// Exported variables.
var (
	ErrBundledModelUnavailable = errors.New(
		"bundled model missing or empty — rebuild the binary with the model in place, " +
			"or set ENGRAM_MODEL_PATH to a directory containing model.onnx",
	)
	ErrHugotEmbedEmpty = errors.New("hugot embed: empty result")
	ErrHugotProbeEmpty = errors.New("hugot probe returned no embedding")
)

// Backend opens an embedding pipeline for an on-disk model directory. The
// production implementation is composed internally by NewRuntimeBackend
// (runtime.go) over the raw Runtime that cmd wires into cli.Primitives —
// no hugot import anywhere in internal (#700); tests inject fakes to
// exercise every constructor branch.
type Backend interface {
	OpenPipeline(ctx context.Context, modelDir string) (PipelineHandle, error)
}

// FeatureOutput is the embedding shape returned by
// PipelineHandle.RunPipeline, mirrored here so implementations don't leak
// their runtime's own output types.
type FeatureOutput struct {
	Embeddings [][]float32
}

// HugotEmbedder wraps an embedding pipeline. Safe for concurrent use — the
// production pipeline runs the model under its own lock.
type HugotEmbedder struct {
	pipeline interface {
		RunPipeline(ctx context.Context, inputs []string) (out FeatureOutput, err error)
	}
	modelID string
	dims    int

	// Capture the close logic at construction time so the destroy chain
	// stays encapsulated even if the backend's session type changes.
	close func() error
}

// PipelineHandle is the runtime surface of an opened pipeline plus its
// owning session; Destroy releases both together.
type PipelineHandle interface {
	RunPipeline(ctx context.Context, inputs []string) (FeatureOutput, error)
	Destroy() error
}

// BundledModelFS returns the go:embed-ed bundled model assets, rooted at
// BundledModelDir. Exposed so cmd/engram (and its integration tests) can
// hand the bundled assets to the injectable constructors.
func BundledModelFS() stdembed.FS { return bundledModel }

// NewBundledHugotEmbedder is the production constructor: bundled assets FS,
// fixed model directory, fixed model ID, and caller-supplied backend, cache
// FS, and cache dir. The cache dir is the XDG-keyed path where the model is
// extracted once and reused across all subsequent invocations.
func NewBundledHugotEmbedder(
	ctx context.Context, backend Backend, cfs CacheFS, cacheDir string,
) (*HugotEmbedder, error) {
	return NewHugotEmbedderFromFS(
		ctx, backend, cfs, bundledModel, BundledModelDir, BundledModelID, cacheDir)
}

// NewHugotEmbedderFromDir constructs an embedder reading the model from a
// directory on disk via the injected backend, probing once to learn the
// embedding dimensionality. Every error branch (pipeline open, probe run,
// empty probe) is unit-testable with a fake Backend.
func NewHugotEmbedderFromDir(
	ctx context.Context, backend Backend, modelDir, modelID string,
) (*HugotEmbedder, error) {
	handle, openErr := backend.OpenPipeline(ctx, modelDir)
	if openErr != nil {
		return nil, openErr
	}

	probe, probeErr := handle.RunPipeline(ctx, []string{"probe"})
	if probeErr != nil {
		_ = handle.Destroy()

		return nil, probeErr
	}

	if len(probe.Embeddings) == 0 || len(probe.Embeddings[0]) == 0 {
		_ = handle.Destroy()

		return nil, ErrHugotProbeEmpty
	}

	runner := &pipelineRunner{run: handle.RunPipeline}

	return &HugotEmbedder{
		pipeline: runner,
		modelID:  modelID,
		dims:     len(probe.Embeddings[0]),
		close:    handle.Destroy,
	}, nil
}

// NewHugotEmbedderFromFS constructs an embedder from any stdembed.FS rooted
// at modelDir. cacheDir is the stable directory where the model is
// extracted once via cfs and reused across invocations (XDG-keyed). Tests
// pass an empty FS to verify UAT 10's clear-error path.
func NewHugotEmbedderFromFS(
	ctx context.Context, backend Backend, cfs CacheFS,
	modelFS stdembed.FS, modelDir, modelID, cacheDir string,
) (*HugotEmbedder, error) {
	dir, extractErr := extractToCache(cfs, modelFS, modelDir, cacheDir)
	if extractErr != nil {
		return nil, extractErr
	}

	return NewHugotEmbedderFromDir(ctx, backend, dir, modelID)
}

// Close releases the underlying session. Safe to call multiple times. The
// model cache dir is NOT removed — it is a shared, persistent cache reused
// across all engram invocations.
func (h *HugotEmbedder) Close() error {
	if h.close != nil {
		err := h.close()
		h.close = nil

		return err
	}

	return nil
}

// Dims reports the embedding dimensionality.
func (h *HugotEmbedder) Dims() int { return h.dims }

// Embed runs the pipeline on text (truncated to fit the model's context
// window) and returns the resulting vector.
//
// The char guard assumes prose density; code-dense text can still exceed the
// model's 512-token positional limit within the char limit (observed: 1500
// chars of transcript tokenizing to 538 tokens, panicking graph compilation).
// On failure the input is halved and retried until it succeeds or bottoms out,
// so a single dense chunk degrades to a shorter prefix instead of failing the
// whole ingest.
func (h *HugotEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	if len(text) > hugotInputCharLimit {
		text = text[:hugotInputCharLimit]
	}

	for {
		out, err := h.pipeline.RunPipeline(ctx, []string{text})
		if err != nil {
			if len(text) >= hugotRetryFloorChars {
				text = text[:len(text)/2]

				continue
			}

			return nil, err
		}

		if len(out.Embeddings) == 0 {
			return nil, ErrHugotEmbedEmpty
		}

		return out.Embeddings[0], nil
	}
}

// ModelID reports the configured model identifier.
func (h *HugotEmbedder) ModelID() string { return h.modelID }

// LazyEmbedder defers construction of an embedder until first use so
// commands that don't need it (help, update, transcript) don't pay the
// model-unpack cost or die if model loading fails. The construction is
// factory-injected so tests can drive both the success and failure
// init paths without a real backend.
type LazyEmbedder struct {
	once    sync.Once
	factory func() (*HugotEmbedder, error)
	emb     *HugotEmbedder
	initErr error
}

// NewLazyEmbedder returns a wrapper around NewBundledHugotEmbedder that
// extracts the bundled model to cacheDir at most once (on first Embed /
// ModelID / Dims call) using the injected backend and cache FS. cacheDir
// should be the XDG-keyed stable cache path for the model, e.g.
// $XDG_CACHE_HOME/engram/models/<model_id>/.
func NewLazyEmbedder(backend Backend, cfs CacheFS, cacheDir string) *LazyEmbedder {
	return &LazyEmbedder{
		// Background context: the lazy init runs at most once per process;
		// a request-scoped context could cancel construction partway through
		// model extraction and leave a partial temp dir.
		factory: func() (*HugotEmbedder, error) {
			return NewBundledHugotEmbedder(context.Background(), backend, cfs, cacheDir)
		},
	}
}

// Dims lazily constructs the embedder, then delegates. Returns 0 when
// construction failed; callers should detect via an Embed error.
func (l *LazyEmbedder) Dims() int {
	l.init()

	if l.initErr != nil {
		return 0
	}

	return l.emb.Dims()
}

// Embed lazily constructs the embedder, then delegates.
func (l *LazyEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	l.init()

	if l.initErr != nil {
		return nil, fmt.Errorf("embedder unavailable: %w", l.initErr)
	}

	return l.emb.Embed(ctx, text)
}

// ModelID lazily constructs the embedder, then delegates. Returns the
// bundled model id when construction has not been attempted yet so
// status-style callers can avoid paying the unpack cost.
func (l *LazyEmbedder) ModelID() string {
	if l.emb == nil && l.initErr == nil {
		return BundledModelID
	}

	if l.initErr != nil {
		return BundledModelID
	}

	return l.emb.ModelID()
}

// init runs at most once per LazyEmbedder via sync.Once. The factory
// is provided at construction time so tests can drive both success and
// failure init paths without a real backend.
func (l *LazyEmbedder) init() {
	l.once.Do(func() {
		l.emb, l.initErr = l.factory()
	})
}

// unexported constants.
const (
	hugotInputCharLimit = 1500
	// hugotRetryFloorChars stops the over-length halving retry: below this
	// the failure is not a token-budget problem and must surface.
	hugotRetryFloorChars = 100
)

//go:embed assets/model/*
var bundledModel stdembed.FS

// pipelineRunner adapts a PipelineHandle's run function to the small
// interface HugotEmbedder depends on; isolating the dependency makes
// backend version bumps a surgical edit instead of a sweep.
type pipelineRunner struct {
	run func(ctx context.Context, inputs []string) (FeatureOutput, error)
}

func (p *pipelineRunner) RunPipeline(ctx context.Context, inputs []string) (FeatureOutput, error) {
	return p.run(ctx, inputs)
}
```

- [ ] 5. **Rewrite `internal/embed/cache.go`** — `CacheFS` exported, os import gone, exist-classification becomes the `fs.ErrExist` contract (the classification itself lives in cachefs.go, step 6). Full replacement:

```go
package embed

import (
	stdembed "embed"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
)

// CacheFS is the I/O surface extractToCache depends on. The production
// implementation is composed internally by NewCacheFS (cachefs.go) over
// raw filesystem primitives; tests inject fakes to exercise every branch
// without touching the real disk.
type CacheFS interface {
	// StatSentinel reports whether the cache dir already has a .complete sentinel.
	StatSentinel(cacheDir string) (bool, error)
	// MkdirAll ensures the parent directory of the cache dir exists.
	MkdirAll(path string) error
	// MkdirTemp creates a temporary directory sibling of cacheDir for atomic extraction.
	MkdirTemp(parent, pattern string) (string, error)
	// WriteFile writes data to path (used to copy model files into the temp dir).
	WriteFile(path string, data []byte) error
	// WriteSentinel writes the .complete sentinel into tmpDir.
	WriteSentinel(tmpDir string) error
	// Rename renames src to dst atomically. When dst already exists
	// (concurrent-race scenario), the returned error MUST satisfy
	// errors.Is(err, fs.ErrExist) — implementations translate platform
	// quirks (e.g. macOS ENOTEMPTY on dir-over-dir renames) before returning.
	Rename(src, dst string) error
	// RemoveAll deletes path recursively (used to clean up temp on rename race).
	RemoveAll(path string) error
}

// commitCache atomically renames tmp into cacheDir. If the rename fails with
// a destination-exists error (concurrent-process race), it re-checks the
// sentinel: if the winner completed the cache, discards tmp and returns
// cacheDir. Otherwise returns the rename error.
func commitCache(cfs CacheFS, tmp, cacheDir string) (string, error) {
	renameErr := cfs.Rename(tmp, cacheDir)
	if renameErr == nil {
		return cacheDir, nil
	}

	// If the rename failed because the destination exists, check whether
	// another process just won the race and completed the cache. If so,
	// discard our temp.
	if errors.Is(renameErr, fs.ErrExist) {
		complete, statErr := cfs.StatSentinel(cacheDir)
		if statErr == nil && complete {
			_ = cfs.RemoveAll(tmp)

			return cacheDir, nil
		}
	}

	// True rename failure (or sentinel absent after race check).
	_ = cfs.RemoveAll(tmp)

	return "", fmt.Errorf("cache rename: %w", renameErr)
}

// copyModelFiles copies every non-directory entry from modelFS/modelDir into tmpDir.
func copyModelFiles(cfs CacheFS, modelFS stdembed.FS, modelDir, tmpDir string) error {
	entries, _ := modelFS.ReadDir(modelDir) // already validated by caller

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		data, readErr := modelFS.ReadFile(filepath.Join(modelDir, entry.Name()))
		if readErr != nil {
			return fmt.Errorf("read embedded %s: %w", entry.Name(), readErr)
		}

		writeErr := cfs.WriteFile(filepath.Join(tmpDir, entry.Name()), data)
		if writeErr != nil {
			return fmt.Errorf("unpack %s: %w", entry.Name(), writeErr)
		}
	}

	return nil
}

// extractToCache ensures that <cacheDir> contains the fully extracted model
// and the .complete sentinel. On first call it extracts into a sibling temp
// dir and atomically renames it into place. On subsequent calls (sentinel
// present) it returns immediately without any I/O. A concurrent-process race
// (rename fails because another process just won) is handled by discarding
// the temp dir and using the pre-existing complete cache.
func extractToCache(
	cfs CacheFS,
	modelFS stdembed.FS,
	modelDir string,
	cacheDir string,
) (string, error) {
	// Fast path: already extracted.
	ok, statErr := cfs.StatSentinel(cacheDir)
	if statErr != nil {
		return "", statErr
	}

	if ok {
		return cacheDir, nil
	}

	return populateCache(cfs, modelFS, modelDir, cacheDir)
}

// populateCache handles the slow path of extractToCache: verifying the model
// FS, creating a sibling temp dir, copying model files, and atomically renaming
// the temp dir into place.
func populateCache(
	cfs CacheFS,
	modelFS stdembed.FS,
	modelDir string,
	cacheDir string,
) (string, error) {
	// Verify the model FS has files before creating any directories.
	entries, dirErr := modelFS.ReadDir(modelDir)
	if dirErr != nil || len(entries) == 0 {
		return "", fmt.Errorf("%w: dir %s (underlying: %w)",
			ErrBundledModelUnavailable, modelDir, dirErr,
		)
	}

	// Ensure the parent directory exists.
	parent := filepath.Dir(cacheDir)

	mkdirErr := cfs.MkdirAll(parent)
	if mkdirErr != nil {
		return "", fmt.Errorf("cache parent dir: %w", mkdirErr)
	}

	// Extract into a sibling temp dir so the rename is atomic.
	tmp, tmpErr := cfs.MkdirTemp(parent, ".tmp-engram-model-*")
	if tmpErr != nil {
		return "", fmt.Errorf("cache temp dir: %w", tmpErr)
	}

	copyErr := copyModelFiles(cfs, modelFS, modelDir, tmp)
	if copyErr != nil {
		_ = cfs.RemoveAll(tmp)

		return "", copyErr
	}

	sentinelErr := cfs.WriteSentinel(tmp)
	if sentinelErr != nil {
		_ = cfs.RemoveAll(tmp)

		return "", fmt.Errorf("cache sentinel: %w", sentinelErr)
	}

	return commitCache(cfs, tmp, cacheDir)
}
```

- [ ] 6. **Create `internal/embed/cachefs.go`** — the composed production CacheFS over raw primitives (the old cmd `osCacheFS` logic, now internal; sentinel + perm policy + exist-classification live here):

```go
package embed

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

// CacheFSPrims carries the raw filesystem capabilities the production
// CacheFS composition needs. Field signatures are identical to the
// matching cli.Primitives fields — cli.NewDeps forwards its Primitives
// fields into this struct verbatim (doctrine flag E-2: no new cache
// fields on cli.Primitives). The funcs return RAW os errors; all wrapping
// and exist-classification happen here, internally.
type CacheFSPrims struct {
	Stat      func(path string) (fs.FileInfo, error)
	MkdirAll  func(path string, perm fs.FileMode) error
	MkdirTemp func(dir, pattern string) (string, error)
	WriteFile func(path string, data []byte, perm fs.FileMode) error
	Rename    func(oldPath, newPath string) error
	RemoveAll func(path string) error
}

// NewCacheFS composes the production CacheFS from raw filesystem
// primitives: sentinel policy, permission policy, error wrapping, and the
// fs.ErrExist rename contract all live here (#700).
func NewCacheFS(prims CacheFSPrims) CacheFS {
	return primCacheFS{prims: prims}
}

// unexported constants.
const (
	// sentinelFileName marks a fully extracted model cache dir.
	sentinelFileName = ".complete"

	cacheDirPerm  fs.FileMode = 0o755
	cacheFilePerm fs.FileMode = 0o600
)

// primCacheFS is the CacheFS composition over raw primitives.
type primCacheFS struct {
	prims CacheFSPrims
}

// MkdirAll ensures the parent directory of the cache dir exists.
func (c primCacheFS) MkdirAll(path string) error {
	err := c.prims.MkdirAll(path, cacheDirPerm)
	if err != nil {
		return fmt.Errorf("mkdir all: %w", err)
	}

	return nil
}

// MkdirTemp creates a temporary sibling dir for atomic extraction.
func (c primCacheFS) MkdirTemp(parent, pattern string) (string, error) {
	tmp, err := c.prims.MkdirTemp(parent, pattern)
	if err != nil {
		return "", fmt.Errorf("mkdir temp: %w", err)
	}

	return tmp, nil
}

// RemoveAll deletes path. The raw primitive's contract (os.RemoveAll: nil
// on missing paths) is already caller-friendly; the error passes through
// unwrapped.
func (c primCacheFS) RemoveAll(path string) error {
	return c.prims.RemoveAll(path) //nolint:wrapcheck // see comment above
}

// Rename renames src to dst atomically. When the destination already
// exists (including macOS ENOTEMPTY for dir-over-dir renames), the
// returned error satisfies errors.Is(err, fs.ErrExist) per the CacheFS
// contract.
func (c primCacheFS) Rename(src, dst string) error {
	err := c.prims.Rename(src, dst)
	if err != nil {
		if renameIsExist(err) {
			return fmt.Errorf("%w: %w", fs.ErrExist, err)
		}

		return fmt.Errorf("rename: %w", err)
	}

	return nil
}

// StatSentinel reports whether cacheDir already has a .complete sentinel.
func (c primCacheFS) StatSentinel(cacheDir string) (bool, error) {
	_, err := c.prims.Stat(filepath.Join(cacheDir, sentinelFileName))
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("stat sentinel: %w", err)
	}

	return true, nil
}

// WriteFile writes data to path (copies model files into the temp dir).
func (c primCacheFS) WriteFile(path string, data []byte) error {
	err := c.prims.WriteFile(path, data, cacheFilePerm)
	if err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

// WriteSentinel writes the .complete sentinel into tmpDir.
func (c primCacheFS) WriteSentinel(tmpDir string) error {
	err := c.prims.WriteFile(filepath.Join(tmpDir, sentinelFileName), []byte{}, cacheFilePerm)
	if err != nil {
		return fmt.Errorf("write sentinel: %w", err)
	}

	return nil
}

// renameIsExist reports whether err (raw from the rename primitive) is a
// destination-exists error. errors.Is(err, fs.ErrExist) covers EEXIST
// and — via syscall.Errno's Is mapping — ENOTEMPTY through
// *os.LinkError's Unwrap; the string fallback preserves the previous
// defensive platform sniffing for chains that don't unwrap to a mapped
// errno, without importing os (doctrine flag E-3).
func renameIsExist(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, fs.ErrExist) {
		return true
	}

	message := err.Error()

	return strings.Contains(message, "file exists") ||
		strings.Contains(message, "directory not empty")
}
```

- [ ] 7. **Create `cmd/engram/hugot.go`** — the THIN production runtime. This is the entire file; `targ check-thin-api` walks it, and every declaration is a verified-thin shape (empty struct; single-return external call; 3-statement simple-error-wrapper whose closing return carries the doctrine-capped closure — flag E-1):

```go
// Thin hugot capability wrappers. This file (plus its _test siblings) is
// the only place in the repo outside internal/embed's _test files that
// imports hugot — and it holds NO logic: hugotRuntime is an EMPTY struct
// whose methods are single-call / simple-error-wrapper bodies. The
// session/pipeline lifecycle, config policy, output mapping, and error
// wrapping all live in internal/embed (#700).
package main

import (
	"context"

	"github.com/knights-analytics/hugot"

	"github.com/toejough/engram/internal/embed"
)

// hugotRuntime implements embed.Runtime over the real hugot library.
type hugotRuntime struct{}

// NewPipeline opens a feature-extraction pipeline on session and returns
// its run function, erasing hugot's pipeline type via closure capture
// (doctrine flag E-1): the closure body is the sanctioned
// trivially-sequenced single-call shape — run on the captured pipe,
// err-check, selector return.
func (hugotRuntime) NewPipeline(
	session embed.RawSession, modelPath, name, onnxFilename string,
) (embed.RunPipelineFunc, error) {
	//nolint:forcetypeassert // production invariant: sessions come from NewSession
	pipe, err := hugot.NewPipeline(session.(*hugot.Session), hugot.FeatureExtractionConfig{
		ModelPath:    modelPath,
		Name:         name,
		OnnxFilename: onnxFilename,
	})
	if err != nil {
		return nil, err
	}

	return func(ctx context.Context, inputs []string) ([][]float32, error) {
		out, runErr := pipe.RunPipeline(ctx, inputs)
		if runErr != nil {
			return nil, runErr
		}

		return out.Embeddings, nil
	}, nil
}

// NewSession opens a Go-backend hugot session. *hugot.Session satisfies
// embed.RawSession structurally.
func (hugotRuntime) NewSession(ctx context.Context) (embed.RawSession, error) {
	return hugot.NewGoSession(ctx)
}
```

Checker verification (derived from targ's `checkFuncThinness` source — escalate, do not restructure, if the gate disagrees): `type hugotRuntime struct{}` = empty struct → thin; `NewSession` = single return of an external call (`hugot.NewGoSession`) → thin; `NewPipeline` = `isSimpleErrorWrapper` (stmt 1: `pipe, err := hugot.NewPipeline(...)` — RHS is a `pkg.Func` call, arguments are not walked, so the inline type assertion and composite literal are legal; stmt 2: `if err != nil`; stmt 3: any return statement — contents unchecked). Raw errors, zero constants, zero `pipelines` import: config values and wraps are internal.

- [ ] 8. **Adapt internal/embed tests.** Delete files: `internal/embed/production_cache_test.go`, `internal/embed/production_hugot_test.go`, `internal/embed/unpack_test.go`, `internal/embed/tempfs_test.go`. Rewrite `internal/embed/export_test.go` (full replacement):

```go
package embed

import (
	"context"
	stdembed "embed"
)

// Exported variables.
var (
	ExportNotExist = notExist
)

// ExportExtractToCache exposes the unexported extractToCache helper so
// tests can exercise the sentinel / race / error branches with a fake
// CacheFS without touching the real disk.
func ExportExtractToCache(
	cfs CacheFS,
	modelFS stdembed.FS,
	modelDir string,
	cacheDir string,
) (string, error) {
	return extractToCache(cfs, modelFS, modelDir, cacheDir)
}

// NewHugotEmbedderWithPipelineForTest constructs a HugotEmbedder around
// a caller-supplied pipeline implementation. Tests use this to exercise
// the Embed/Close error branches without depending on a real backend.
func NewHugotEmbedderWithPipelineForTest(
	modelID string, dims int,
	runFn func(text string) ([][]float32, error),
	closeFn func() error,
) *HugotEmbedder {
	runner := &pipelineRunner{
		run: func(_ context.Context, inputs []string) (FeatureOutput, error) {
			out, err := runFn(inputs[0])
			if err != nil {
				return FeatureOutput{}, err
			}

			return FeatureOutput{Embeddings: out}, nil
		},
	}

	return &HugotEmbedder{
		pipeline: runner,
		modelID:  modelID,
		dims:     dims,
		close:    closeFn,
	}
}

// NewLazyEmbedderWithFactoryForTest constructs a LazyEmbedder with a
// caller-supplied factory so tests can drive both init success and
// failure paths without a real backend.
func NewLazyEmbedderWithFactoryForTest(factory func() (*HugotEmbedder, error)) *LazyEmbedder {
	return &LazyEmbedder{factory: factory}
}

// SetCacheDirForTest is a no-op test helper for the Close-does-not-delete
// test. HugotEmbedder no longer holds a tmpDir field — Close only closes the
// backend session and never removes any directory. The function is kept for
// test readability; the test creates its own dir and verifies it survives.
func SetCacheDirForTest(_ *HugotEmbedder, _ string) {}
```

Then mechanical edits, all in `internal/embed/*_test.go`:
  - Everywhere: `embed.ExportHugotBackend` → `embed.Backend`; `embed.ExportHugotPipelineHandle` → `embed.PipelineHandle`; `embed.ExportFeatureOutput` → `embed.FeatureOutput`; `embed.ExportCacheFS` → `embed.CacheFS`; `embed.BuildEmbedderForTest(` → `embed.NewHugotEmbedderFromDir(` (identical argument lists — verified).
  - `cache_test.go`: (a) delete `TestExtractToCache_RealOS` (lines 118-149); (b) replace both race-fake errors at lines 54 and 107 — current: `renameErr: &os.LinkError{Op: "rename", Err: errors.New("directory not empty")}` — new: `renameErr: fmt.Errorf("%w: directory not empty", fs.ErrExist)` (add `"fmt"`, `"io/fs"` imports); (c) type-assertion line 212: `_ embed.ExportCacheFS = (*fakeCacheFS)(nil)` → `_ embed.CacheFS = (*fakeCacheFS)(nil)`; (d) move the `nonEmptyTestFS` declaration here from the deleted `unpack_test.go` (add `stdembed "embed"` import):

```go
//go:embed testdata/gen-reference.py
var nonEmptyTestFS stdembed.FS
```

  - `hugot_test.go`: delete `TestBundledHugotEmbedder_Smoke` (relocated to `cmd/engram/hugot_test.go`, step 2 — where the real `hugotRuntime` lives); adapt T10 (fakes never reached — extraction fails first on the empty FS):

```go
func TestT10_MissingBundledModel_ClearError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	cacheDir := filepath.Join(t.TempDir(), "model-cache")
	_, err := embed.NewHugotEmbedderFromFS(
		context.Background(), &fakeBackend{}, &fakeCacheFS{}, emptyFS, "assets/model", "x@1", cacheDir)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("ENGRAM_MODEL_PATH"))
}
```

  (`fakeBackend` lives in `buildembedder_test.go`, `fakeCacheFS` in `cache_test.go` — same `embed_test` package; zero-value `fakeCacheFS.StatSentinel` returns `(false, nil)` — verified.)
  - `embedder_fake_test.go`: `embed.NewLazyEmbedder(t.TempDir())` → `embed.NewLazyEmbedder(&fakeBackend{}, &fakeCacheFS{}, t.TempDir())`.

- [ ] 9. **Wire internal/cli: Primitives field, NewDeps 3-arg flip, bridge, cmd literal line.** Four sub-edits, one commit with the rest of the task. **9a.** In `internal/cli/embed.go` delete the `modelCacheDir()` helper (its `os.UserHomeDir`/`os.Getenv` reads die with it — the `"os"` import leaves this file) and replace the `sharedEmbedder` singleton block:

Current:
```go
// unexported variables.
var (
	//nolint:gochecknoglobals // shared lazy singleton across CLI commands
	sharedEmbedder = embed.NewLazyEmbedder(modelCacheDir())
)
```

Replacement (add `"sync/atomic"` import):
```go
// unexported variables.
var (
	// errEmbedderUnwired reports an Embed call before Targets wired Deps.Embed.
	errEmbedderUnwired = errors.New("embedder not wired: cli.Targets(deps) has not run")

	// sharedEmbedderPtr holds the Deps-wired production embedder, stored by
	// wireSharedEmbedder (called from Targets). Atomic because tests build
	// Targets concurrently.
	// TRANSITIONAL (#700): deleted once every per-command deps constructor
	// takes Deps and reads d.Embed directly.
	//nolint:gochecknoglobals // transitional bridge, see comment
	sharedEmbedderPtr atomic.Pointer[embed.Embedder]

	// sharedEmbedder is the value legacy per-command constructors wire into
	// their deps structs; it forwards to the Targets-wired embedder.
	// TRANSITIONAL (#700) — same removal condition as sharedEmbedderPtr.
	//nolint:gochecknoglobals // transitional bridge, see comment
	sharedEmbedder embed.Embedder = bridgeEmbedder{ptr: &sharedEmbedderPtr}
)

// bridgeEmbedder forwards Embedder calls to the embedder wired by Targets.
// Pre-wiring fallbacks mirror LazyEmbedder's pre-init behavior: ModelID
// reports the bundled constant, Dims reports 0, Embed errors.
type bridgeEmbedder struct {
	ptr *atomic.Pointer[embed.Embedder]
}

// Dims delegates to the wired embedder; 0 before wiring.
func (b bridgeEmbedder) Dims() int {
	emb := b.load()
	if emb == nil {
		return 0
	}

	return emb.Dims()
}

// Embed delegates to the wired embedder; errors before wiring.
func (b bridgeEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	emb := b.load()
	if emb == nil {
		return nil, errEmbedderUnwired
	}

	return emb.Embed(ctx, text)
}

// ModelID delegates to the wired embedder; bundled constant before wiring
// (keeps status-style callers unpack-free, matching LazyEmbedder).
func (b bridgeEmbedder) ModelID() string {
	emb := b.load()
	if emb == nil {
		return embed.BundledModelID
	}

	return emb.ModelID()
}

// load returns the wired embedder or nil before wiring.
func (b bridgeEmbedder) load() embed.Embedder {
	ptr := b.ptr.Load()
	if ptr == nil || *ptr == nil {
		return nil
	}

	return *ptr
}

// wireSharedEmbedder points the transitional sharedEmbedder bridge at the
// Deps-wired embedder. Called by Targets.
func wireSharedEmbedder(embedder embed.Embedder) {
	sharedEmbedderPtr.Store(&embedder)
}
```

**9b.** In `internal/cli/targets.go`, insert as the first statement of `Targets` (T2's landed shape `func Targets(deps Deps) []any`): `wireSharedEmbedder(deps.Embed)`.

**9c.** In `internal/cli/primitives.go`: add ONE field to `Primitives` (grouped after the debug-sink field, with the doctrine cross-reference) —

```go
	// Embedding runtime (cmd wires an EMPTY struct with single-call
	// methods; all lifecycle/config/cache policy is internal — doctrine
	// flags D-1/E-1/E-2).
	EmbedRuntime embed.Runtime
```

— and flip `NewDeps`'s guarded Embed wiring (T2's landed R6 handoff line) to the 3-arg composition. Current (landed by T2):

```go
	// The lazy embedder is constructed once here, preserving the
	// one-unpack-per-process property of the old sharedEmbedder singleton
	// (guarded: minimal fake Primitives without Getenv skip it). R6: T14
	// swaps this line to the 3-arg constructor over cmd-injected backend
	// and cache capabilities.
	if prims.Getenv != nil {
		deps.Embed = embed.NewLazyEmbedder(
			CacheDirFromHome(homeOrEmpty(deps), embed.BundledModelID, prims.Getenv))
	}
```

New:

```go
	// The lazy embedder is constructed once here, preserving the
	// one-unpack-per-process property of the old sharedEmbedder singleton
	// (guarded: minimal fake Primitives without Getenv skip it). R6/D-1:
	// backend composed from the raw EmbedRuntime, cache FS from the raw
	// filesystem primitives — no embed wiring in cmd. A nil EmbedRuntime
	// surfaces as embed.ErrRuntimeMissing on first use (fail-loud lazy),
	// never a panic.
	if prims.Getenv != nil {
		deps.Embed = embed.NewLazyEmbedder(
			embed.NewRuntimeBackend(prims.EmbedRuntime),
			embed.NewCacheFS(embed.CacheFSPrims{
				Stat:      prims.Stat,
				MkdirAll:  prims.MkdirAll,
				MkdirTemp: prims.MkdirTemp,
				WriteFile: prims.WriteFile,
				Rename:    prims.Rename,
				RemoveAll: prims.RemoveAll,
			}),
			CacheDirFromHome(homeOrEmpty(deps), embed.BundledModelID, prims.Getenv))
	}
```

**9d.** In `cmd/engram/main.go`'s `cli.Primitives` literal, add ONE field line (after `WalkDir:`, keeping the literal's direct-reference grouping):

```go
			EmbedRuntime: hugotRuntime{},
```

Package main stays declaration-free — `hugotRuntime{}` is a composite-literal ARGUMENT expression (unchecked by the gate); the type itself is step 7's empty struct. NOTE(cli_test DRIFT flag): `realPrimitives()` in internal/cli tests does NOT gain an EmbedRuntime (cli_test cannot reference package main's `hugotRuntime`); Deps built from it get the fail-loud `ErrRuntimeMissing` lazy embedder, which no cli-level test may trigger (R11's `stubEmbedderForTargets` covers targets-level embed tests; the production literal is guarded by cli_test's end-to-end binary tests).

- [ ] 10. **Bridge behavior tests (parallel-safe, no global state).** Add to `internal/cli/export_test.go`:

```go
// ExportNewBridgeEmbedder returns a fresh transitional shared-embedder
// bridge plus its wire function, backed by an isolated pointer so bridge
// behavior tests never touch the package-global embedder.
func ExportNewBridgeEmbedder() (embed.Embedder, func(embed.Embedder)) {
	ptr := &atomic.Pointer[embed.Embedder]{}
	bridge := bridgeEmbedder{ptr: ptr}
	wire := func(e embed.Embedder) { ptr.Store(&e) }

	return bridge, wire
}
```

Create `internal/cli/embed_bridge_test.go`:

```go
package cli_test

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

// TestBridgeEmbedder_UnwiredFallbacks asserts the pre-wiring behavior
// mirrors LazyEmbedder's pre-init semantics.
func TestBridgeEmbedder_UnwiredFallbacks(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	bridge, _ := cli.ExportNewBridgeEmbedder()

	g.Expect(bridge.ModelID()).To(Equal(embed.BundledModelID))
	g.Expect(bridge.Dims()).To(Equal(0))

	_, err := bridge.Embed(context.Background(), "x")
	g.Expect(err).To(MatchError(ContainSubstring("embedder not wired")))
}

// TestBridgeEmbedder_DelegatesAfterWire asserts all three methods forward
// to the wired embedder.
func TestBridgeEmbedder_DelegatesAfterWire(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	bridge, wire := cli.ExportNewBridgeEmbedder()
	wire(bridgeStubEmbedder{})

	g.Expect(bridge.ModelID()).To(Equal("stub@4"))
	g.Expect(bridge.Dims()).To(Equal(4))

	vec, err := bridge.Embed(context.Background(), "x")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(vec).To(Equal([]float32{0, 0, 0, 0}))
}

type bridgeStubEmbedder struct{}

func (bridgeStubEmbedder) Dims() int { return 4 }

func (bridgeStubEmbedder) Embed(context.Context, string) ([]float32, error) {
	return []float32{0, 0, 0, 0}, nil
}

func (bridgeStubEmbedder) ModelID() string { return "stub@4" }
```

- [ ] 11. **Verify.** Run `targ test` — expected: all green (internal runtime/cachefs suites green with fakes; real-os integration green; cmd smoke green; bridge tests green; `TestTargets_EmbedStatus` stays green via bridge → LazyEmbedder pre-init ModelID). Run `targ check-full` — expected: clean. Run `targ check-thin-api` — expected: PASS (cmd/engram/hugot.go's declarations are the verified-thin shapes from step 7; the main.go literal line is an argument expression). If the gate flags ANY declaration, ESCALATE the exact finding to the orchestrator — do not suppress, do not restructure ad hoc (Global Constraints / doctrine item 5). Confirm `grep -rn '"os"\|knights-analytics' internal/embed/*.go | grep -v _test` returns nothing. Run `go install ./cmd/engram && engram embed status --vault "$(mktemp -d)"` from a non-repo cwd — expected: the six status lines with all-zero counts (real-binary check per house rules).
- [ ] 12. **Commit:**

```
refactor(embed): internal backend/cache composition, thin cmd (#700)

internal/embed now owns ALL embedder orchestration: session/pipeline
lifecycle (NewRuntimeBackend over the raw Runtime seam) and cache
extraction + sentinel/perm policy + the errors.Is(fs.ErrExist) rename
classification (NewCacheFS over raw FS primitives). cmd/engram/hugot.go
shrinks to an EMPTY hugotRuntime struct with two single-call methods
(targ check-thin-api PASS); cli.NewDeps composes Deps.Embed from
Primitives internally (R6 3-arg flip, doctrine D-1/E-1/E-2). Dead
tempFS/unpackModelToTemp machinery deleted. cli gains the transitional
sharedEmbedder bridge wired from Deps.Embed.

AI-Used: [claude]
```

---

### Task T15 (B): internal/cli/embed.go — compose EmbedDeps from cli.Deps, delete osEmbedFS

**Depends on:** Task T14 (A) + T1-rework/T2 landed (Deps.FS `EdgeFS`; `Deps.Embed` composed INSIDE `cli.NewDeps` per R6/D-1 — this task never touches cmd/engram or the embed wiring, only internal composition; verified unaffected by the T14 doctrine rework).

**Files**

- Modify: `internal/cli/embed.go` (delete `osEmbedFS`, `newOsEmbedDeps`; add `newEmbedDeps`)
- Modify: `internal/cli/targets.go` (lines 226, 230, 155), `internal/cli/query.go` (line 1287-1288)
- Modify: `internal/cli/export_test.go` (replace `ExportNewOsEmbedDeps`), `internal/cli/os_adapters_test.go`
- NOT touched (R2): `internal/cli/vault_fs.go` — this task declares NO VaultFS adapter; the draft's `depsVaultFS` is a loser. It consumes T5's landed `newVaultFS(d.FS)`.

**Interfaces**

- Produces: `newEmbedDeps(d Deps) EmbedDeps` (pure composition); `ExportNewEmbedDeps(d Deps) EmbedDeps`. (No `depsVaultFS` — R2; the vaultgraph.VaultFS view comes from T5's `newVaultFS`.)
- Consumes: `Deps.FS EdgeFS` (`ReadFile`, `WriteFileAtomic(path, data, perm fs.FileMode)`, `ReadDir(path) ([]fs.DirEntry, error)`), `Deps.Embed embed.Embedder`, `vaultgraph.ScanVault(fs VaultFS, vaultPath string) ([]Note, error)` with `VaultFS{ ListMD(dir string) ([]string, error); ReadFile(path string) ([]byte, error) }` (verified at `internal/vaultgraph/scanner.go:20-32`).

**Steps**

- [ ] 1. **RED — adapt the integration tests to the composed deps first.** In `internal/cli/os_adapters_test.go`: replace the three `cli.ExportNewOsEmbedDeps(<embedder>)` calls (lines 89, 125, 190) with `cli.ExportNewEmbedDeps(cli.Deps{FS: osTestEdgeFS{}, Embed: <same embedder>})`; rename `TestOsEmbedFS_ReadWriteScanRoundTrip` → `TestEmbedDeps_ReadWriteScanRoundTrip` and update its comment to say it exercises the composed Scan/Read/Write against a real tempdir vault. Do NOT append the EdgeFS double below if `osTestEdgeFS` already exists in package cli_test — per R4, T5's edgefs_os_test.go landed it before this task runs, so the expected action is to CONSUME that one and skip this block (a second declaration in the same package is a compile error; the block below survives only for the contingency that T5's was somehow renamed/removed — DESIGN FLAG 9):

```go
// osTestEdgeFS implements cli.EdgeFS over the real filesystem for
// integration tests. Test files are exempt from the internal purity rule.
type osTestEdgeFS struct{}

func (osTestEdgeFS) MkdirAll(path string, perm fs.FileMode) error {
	return os.MkdirAll(path, perm) //nolint:wrapcheck // thin test adapter
}

func (osTestEdgeFS) MkdirTemp(dir, pattern string) (string, error) {
	return os.MkdirTemp(dir, pattern) //nolint:wrapcheck // thin test adapter
}

func (osTestEdgeFS) ReadDir(path string) ([]fs.DirEntry, error) {
	return os.ReadDir(path) //nolint:wrapcheck // thin test adapter
}

func (osTestEdgeFS) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path) //nolint:wrapcheck // thin test adapter
}

func (osTestEdgeFS) Remove(path string) error {
	return os.Remove(path) //nolint:wrapcheck // thin test adapter
}

func (osTestEdgeFS) RemoveAll(path string) error {
	return os.RemoveAll(path) //nolint:wrapcheck // thin test adapter
}

func (osTestEdgeFS) Rename(oldPath, newPath string) error {
	return os.Rename(oldPath, newPath) //nolint:wrapcheck // thin test adapter
}

func (osTestEdgeFS) Stat(path string) (fs.FileInfo, error) {
	return os.Stat(path) //nolint:wrapcheck // thin test adapter
}

func (osTestEdgeFS) WalkDir(root string, fn fs.WalkDirFunc) error {
	return filepath.WalkDir(root, fn) //nolint:wrapcheck // thin test adapter
}

func (osTestEdgeFS) WriteFile(path string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(path, data, perm) //nolint:wrapcheck // thin test adapter
}

func (osTestEdgeFS) WriteFileAtomic(path string, data []byte, perm fs.FileMode) error {
	tmp := path + ".tmp-test"

	err := os.WriteFile(tmp, data, perm)
	if err != nil {
		return fmt.Errorf("writing temp %s: %w", tmp, err)
	}

	err = os.Rename(tmp, path)
	if err != nil {
		return fmt.Errorf("renaming %s: %w", tmp, err)
	}

	return nil
}

func (osTestEdgeFS) WriteFileExcl(path string, data []byte, perm fs.FileMode) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm) //nolint:gosec // thin test adapter
	if err != nil {
		return fmt.Errorf("opening excl %s: %w", path, err)
	}

	defer func() { _ = file.Close() }()

	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("writing excl %s: %w", path, err)
	}

	return nil
}
```

(add `"fmt"` and `"io/fs"` to the file's imports). Run `targ test` — expected RED: `ExportNewEmbedDeps` undefined.

- [ ] 2. **GREEN — compose.** In `internal/cli/embed.go` delete `osEmbedFS` and its three methods (lines 136-170) and `newOsEmbedDeps` (lines 241-252); the `"os"` import goes with them. Add:

```go
// newEmbedDeps composes the embed-command dependencies from the CLI-wide
// impure capability set. Pure composition — all I/O flows through d.FS and
// d.Embed, wired via cli.NewDeps at the edge. Sidecar writes go through WriteFileAtomic
// (temp+rename) so concurrent readers always see either the old or new
// file, never a torn write (ADR-0013 semantics preserved).
func newEmbedDeps(d Deps) EmbedDeps {
	const sidecarPerm = 0o600

	return EmbedDeps{
		Scan: func(vault string) ([]vaultgraph.Note, error) {
			return vaultgraph.ScanVault(newVaultFS(d.FS), vault)
		},
		Read: d.FS.ReadFile,
		Write: func(path string, data []byte) error {
			err := d.FS.WriteFileAtomic(path, data, sidecarPerm)
			if err != nil {
				return fmt.Errorf("write: %w", err)
			}

			return nil
		},
		Embedder: d.Embed,
	}
}
```

No VaultFS adapter is declared here (R2): `newVaultFS` already landed in T5's vault_fs.go and provides exactly the vaultgraph.VaultFS-over-EdgeFS view (missing dir → empty, wrapped-ErrNotExist unwrapped via errors.Is) — vault_fs.go is not touched by this task.

In `internal/cli/export_test.go`, replace `ExportNewOsEmbedDeps` (lines 526-534):

```go
// ExportNewEmbedDeps exposes newEmbedDeps so integration tests can drive
// the composed Scan/Read/Write against a test EdgeFS without waking the
// bundled embedder (set Deps.Embed to a stub).
func ExportNewEmbedDeps(d Deps) EmbedDeps { return newEmbedDeps(d) }
```

- [ ] 3. **Rewire call sites.** `internal/cli/targets.go` embed group (current lines 226 and 230): `newOsEmbedDeps()` → `newEmbedDeps(deps)` (identifier per foundation's threading of Deps into `coreTargets`). `internal/cli/query.go:1287-1288` (coordinate with query cluster, DESIGN FLAG 3 — skip if its migration already landed):

Current:
```go
// newOsQueryDeps wires the production scan + read for the query command.
func newOsQueryDeps() QueryDeps {
	embedDeps := newOsEmbedDeps()
```

New:
```go
// newOsQueryDeps wires the production scan + read for the query command.
// TRANSITIONAL (#700): takes Deps for the embed composition; the query
// cluster's migration replaces the remaining os-backed fields.
func newOsQueryDeps(d Deps) QueryDeps {
	embedDeps := newEmbedDeps(d)
```

and its call site `internal/cli/targets.go:155`: `newOsQueryDeps()` → `newOsQueryDeps(deps)`.

- [ ] 3.5. **Wire the targets-level embed tests per R11.** `newTestDeps.Embed` is nil by design (R11 — `newTestDeps` builds the `cli.Deps` literal directly and never calls `NewDeps`, so T14's internal Embed composition does not reach it; VERIFIED unaffected by the T14 doctrine rework: the stub below satisfies `embed.Embedder`, which the rework leaves unchanged, and `embed.BundledModelID` survives). The two executed embed tests dereference `ModelID()`, so give them a local fail-loud stub. In `internal/cli/targets_test.go`, add (exported-test-func-before-private-decls per reorder-decls — place the type after the last Test func or in the file's existing helper region):

```go
// stubEmbedderForTargets satisfies embed.Embedder for targets-level tests that
// only need ModelID/Dims. Embed fails loud: no targets-level test may silently
// real-embed (R11). Named to avoid cli_test's existing stubEmbedder (embed_test.go).
type stubEmbedderForTargets struct{}

func (stubEmbedderForTargets) Embed(context.Context, string) ([]float32, error) {
	return nil, errors.New("stubEmbedderForTargets: Embed not expected in targets-level tests")
}

func (stubEmbedderForTargets) ModelID() string { return embed.BundledModelID }

func (stubEmbedderForTargets) Dims() int { return 384 }
```

In `TestTargets_EmbedApplyDryRun` (targets_test.go:340) and `TestTargets_EmbedStatus` (:355), where the test builds its deps, override: `d := newTestDeps(stdout, stderr); d.Embed = stubEmbedderForTargets{}` (adapt to the tests' actual deps-construction shape — they currently ride `cli.Targets(newTestDeps(...))`; introduce the local variable form for these two tests only). Add the `context`/`errors`/`embed` imports if absent.

- [ ] 4. **Verify.** `targ test` — expected green (embed_test.go's in-memory deps untouched; adapted os_adapters tests pass through `newEmbedDeps` + `osTestEdgeFS`; `TestTargets_EmbedApplyDryRun` / `TestTargets_EmbedStatus` green through the new wiring). `targ check-full` — clean; confirm `grep -n '"os"' internal/cli/embed.go` returns nothing. `targ check-thin-api` — expected PASS (this task touches no cmd/engram file, so a failure here means an earlier task regressed the thin edge — ESCALATE the exact finding, do not fix ad hoc). Real-binary check: `go install ./cmd/engram`, then in a temp dir: create `note.md` with a body, run `engram embed apply --vault . --dry-run` (expect `would-embed note.md (missing)`), then `engram embed apply --vault .` (expect `embedded  note.md (missing)` and a `note.vec.json` sidecar with `"embedding_model_id": "minilm-l6-v2@384"`), then `engram embed status --vault .` (expect `with-embeddings: 1`).
- [ ] 5. **Commit:**

```
refactor(cli): compose embed command deps from cli.Deps (#700)

newEmbedDeps(d Deps) replaces newOsEmbedDeps: Scan/Read/Write flow
through the injected EdgeFS (WriteFileAtomic preserves the ADR-0013
temp+rename sidecar semantics) and the embedder comes from Deps.Embed.
osEmbedFS deleted; internal/cli/embed.go no longer imports os.

AI-Used: [claude]
```

---

**Post-cluster residue for the enforcement task** (not handled here): delete the `sharedEmbedder`/`bridgeEmbedder` transitional block in `internal/cli/embed.go` once `grep -rn "sharedEmbedder" internal/cli --include='*.go' | grep -v _test` shows only its own definition; decide `parity_test.go` exemption (DESIGN FLAG 5); `osVaultFS` deletion is T7's, gated on all consumers having migrated to `newVaultFS(d.FS)` (R2).

### Task T16 (UF-1): `update.ErrCommandNotFound` sentinel + commander translation (drops os/exec from internal/update)

**Files**
- Modify: `internal/update/update.go` (add sentinel; swap two `errors.Is` checks; drop `os/exec` import; fix one comment)
- Modify: `internal/update/runner_test.go` (inject sentinel instead of exec.ErrNotFound; drop `os/exec` import)
- Modify: `internal/cli/update.go` (osCommander translates exec.ErrNotFound → sentinel)
- Modify: `internal/cli/update_test.go` (new RED test)
- Modify: `internal/cli/invariants_u1_test.go` (inject sentinel; drop `os/exec` import; comment fixes)

**Interfaces**
- Produces: `var ErrCommandNotFound = errors.New("command not found")` in package `update` — the Commander contract: implementations translate their platform not-found error to this sentinel before returning.
- Consumes: `update.Commander` (unchanged), `exec.ErrNotFound` (now only in the transitional internal adapter; after T17 only as the injected `Primitives.NotFoundErr` value in cmd/engram's literal — doctrine flag C-1).

**Steps**

1. [ ] Add the sentinel (pure addition, keeps everything compiling for the RED test). In `internal/update/update.go`, replace the exported-variables block (lines 34–44):
   ```go
   // Exported variables.
   var (
   	ErrGitNotFound = errors.New("git binary not found on PATH")
   	ErrGoNotFound  = errors.New("go binary not found on PATH")
   	// ErrModelLFSStub means the cloned model.onnx is a Git-LFS pointer file,
   	// not the real model — building from it would embed a 133-byte stub and
   	// every embedding call would fail (issue #645).
   	ErrModelLFSStub     = errors.New("model.onnx is a git-lfs pointer stub")
   	ErrNoHarness        = errors.New("no supported harness found")
   	ErrSkillsSrcMissing = errors.New("skills source dir missing")
   )
   ```
   with:
   ```go
   // Exported variables.
   var (
   	// ErrCommandNotFound is the Commander contract for "binary not on PATH":
   	// implementations translate their platform's not-found error (e.g.
   	// exec.ErrNotFound) to this sentinel before returning, keeping this
   	// package free of os/exec (#700).
   	ErrCommandNotFound = errors.New("command not found")
   	ErrGitNotFound     = errors.New("git binary not found on PATH")
   	ErrGoNotFound      = errors.New("go binary not found on PATH")
   	// ErrModelLFSStub means the cloned model.onnx is a Git-LFS pointer file,
   	// not the real model — building from it would embed a 133-byte stub and
   	// every embedding call would fail (issue #645).
   	ErrModelLFSStub     = errors.New("model.onnx is a git-lfs pointer stub")
   	ErrNoHarness        = errors.New("no supported harness found")
   	ErrSkillsSrcMissing = errors.New("skills source dir missing")
   )
   ```
   Run `targ test` → expect PASS (no behavior change yet).

2. [ ] RED: in `internal/cli/update_test.go`, insert after `TestOsCommander_RunsCommand` (line 256, alphabetical order preserved):
   ```go
   func TestOsCommander_TranslatesNotFound(t *testing.T) {
   	t.Parallel()

   	g := NewWithT(t)

   	cmd := cli.ExportNewOsCommander()

   	_, _, err := cmd.Run(context.Background(), "", "engram-no-such-binary-7f3a")
   	g.Expect(err).To(MatchError(update.ErrCommandNotFound))
   }
   ```
   Run `targ test` → expect FAIL on exactly this test (current wrap `fmt.Errorf("%s %v: %w", name, args, err)` carries exec.ErrNotFound, not the sentinel).

3. [ ] GREEN: in `internal/cli/update.go`, inside `(*osCommander).Run` (lines 54–57), replace:
   ```go
   	err := cmd.Run()
   	if err != nil {
   		return stdout.Bytes(), stderr.Bytes(), fmt.Errorf("%s %v: %w", name, args, err)
   	}
   ```
   with:
   ```go
   	err := cmd.Run()
   	if err != nil {
   		if errors.Is(err, exec.ErrNotFound) {
   			return stdout.Bytes(), stderr.Bytes(),
   				fmt.Errorf("%s %v: %w: %w", name, args, update.ErrCommandNotFound, err)
   		}

   		return stdout.Bytes(), stderr.Bytes(), fmt.Errorf("%s %v: %w", name, args, err)
   	}
   ```
   (`errors` is already imported at line 6; go 1.26 supports the double `%w`.) Run `targ test` → expect PASS (internal/update still checks exec.ErrNotFound, which remains in the chain — both checks are satisfied during this step). This internal adapter and its os/exec import are TRANSITIONAL — T17 deletes them and re-homes the translation into the primCommander composition over the injected `NotFoundErr` primitive (doctrine flag C-1).

4. [ ] Cut internal/update over to the sentinel — all four hunks in one pass, tests updated in the same step:
   - `internal/update/update.go` line 436, replace:
     ```go
     		if errors.Is(cloneErr, exec.ErrNotFound) {
     ```
     with:
     ```go
     		if errors.Is(cloneErr, ErrCommandNotFound) {
     ```
   - `internal/update/update.go` lines 540–544, replace:
     ```go
     // classifyGoInstallErr maps a `go install` failure to ErrGoNotFound when the go
     // binary is absent from PATH (exec.ErrNotFound), otherwise wrapping the raw
     // error with the install mode for context.
     func classifyGoInstallErr(mode string, runErr error) error {
     	if errors.Is(runErr, exec.ErrNotFound) {
     ```
     with:
     ```go
     // classifyGoInstallErr maps a `go install` failure to ErrGoNotFound when the go
     // binary is absent from PATH (ErrCommandNotFound from the Commander), otherwise
     // wrapping the raw error with the install mode for context.
     func classifyGoInstallErr(mode string, runErr error) error {
     	if errors.Is(runErr, ErrCommandNotFound) {
     ```
   - `internal/update/update.go` imports (lines 6–15): delete the line `"os/exec"`.
   - `internal/update/runner_test.go` line 556, replace `cmd := &fakeCmd{err: exec.ErrNotFound}` with `cmd := &fakeCmd{err: update.ErrCommandNotFound}`; delete `"os/exec"` from its imports (line 8 — only use in the file).
   - `internal/cli/invariants_u1_test.go`: replace line 36
     ```go
     		Cmd: &u1FailCmd{err: fmt.Errorf(`go [install ./cmd/engram/]: %w`, exec.ErrNotFound)},
     ```
     with:
     ```go
     		Cmd: &u1FailCmd{err: fmt.Errorf(`go [install ./cmd/engram/]: %w`, update.ErrCommandNotFound)},
     ```
     delete `"os/exec"` from imports (line 8); replace comment lines 21–22 (`// missing 'go' binary (surfaced as exec.ErrNotFound from the go install` / `// command) must make update FAIL...`) with `// missing 'go' binary (surfaced as update.ErrCommandNotFound from the injected` / `// Commander) must make update FAIL with the update.ErrGoNotFound sentinel, and`; replace comment lines 32–33 with `// Commander fails the way the production adapter does when 'go' is absent:` / `// the update.ErrCommandNotFound chain is preserved through its %w wrap.`
   Run `targ test` → expect PASS. Run `targ check-full` → expect clean (verifies no unused-import leftovers, line lengths). Run `targ check-thin-api` → expect PASS (this task touches no cmd/engram file; a finding here means unrelated drift — escalate the exact finding, never suppress).

5. [ ] Verify zero os/exec in internal non-test files of this family: `grep -rn '"os/exec"' internal/update/ internal/cli/update.go | grep -v _test.go` → expect only `internal/cli/update.go` (the transitional adapter — T17 deletes it along with the import; after T17 this grep returns zero hits).

6. [ ] Commit (via the commit skill):
   ```
   refactor(update): add ErrCommandNotFound sentinel (#700)

   internal/update no longer imports os/exec: the Commander implementation
   now translates the platform not-found error to the sentinel; the two
   errors.Is call sites classify against it. T17 re-homes the translation
   into the primCommander composition over an injected NotFoundErr
   primitive (doctrine flag C-1).

   AI-Used: [claude]
   ```

---

### Task T17 (UF-2): commander via injected run primitive + NotFoundErr (doctrine flag C-1); compose update deps purely from cli.Deps

Sequencing precondition: Task T1-rework (`cli.Primitives`, `cli.NewDeps`, `internal/cli/primitives_integration_test.go` with `realPrimitives()`/`realDepsForTest()`) and Task T2 (declaration-free `cmd/engram/main.go`; `Targets(deps Deps)` threading `deps` into `learnUpdateTargets`) have landed; Task T16 landed the sentinel + the internal/update cutover.

**C-1 field shapes (resolved here — the doctrine assigns this brief the exact shapes; BINDING for this task):**

- `Primitives.RunCommand func(ctx context.Context, dir, name string, args []string, stdout, stderr io.Writer) error` — cmd's literal value is ONE closure whose body is `exec.CommandContext` + three field assignments on the returned handle (`Dir`, `Stdout`, `Stderr`) + `return cmd.Run()`: the second enumerated stdlib-equivalent survivor (doctrine flag C-1: `*exec.Cmd` cannot cross the boundary, so the closure is construction + field assignments + ONE invocation, zero branching — semantically one operation; the checker does not walk closures, and `main()` stays one statement; behavior changes — timeout, env, output policy, retry — extend the SIGNATURE, never this body). `args` is a slice, not variadic, because the writer params must follow it. `*exec.Cmd` never crosses the boundary — internal sees only this erased func shape.
- `Primitives.NotFoundErr error` — cmd wires the bare identifier `exec.ErrNotFound` (the kernel's preferred injected-sentinel-value form; zero cmd logic).
- Everything else lives in internal `primCommander` (internal/cli/commander.go): output collection (the `bytes.Buffer` lifecycle), contextual `%w` wrapping, and the `errors.Is(runErr, prims.NotFoundErr)` → `update.ErrCommandNotFound` translation. `errors.Is` with a nil target matches no non-nil error, so a fake `Primitives` without `NotFoundErr` merely never translates — no nil guard needed.

**Files**
- Modify: `internal/cli/primitives.go` (Primitives gains `RunCommand`/`NotFoundErr`; NewDeps wires `Commander: primCommander{prims: prims}`; imports gain `context`)
- Create: `internal/cli/commander.go` (primCommander — collection + wrapping + translation)
- Create: `internal/cli/commander_test.go` (unit tests, fake run primitive through NewDeps)
- Create: `internal/cli/commander_integration_test.go` (real-exec integration tests — the relocated `TestOsCommander_*` coverage)
- Modify: `internal/cli/primitives_integration_test.go` (extend `realPrimitives()` with the two new fields — doctrine flag DRIFT)
- Modify: `cmd/engram/main.go` (two field lines in the `cli.Primitives` literal; imports gain `context`, `io`, `os/exec`)
- Modify: `internal/cli/update.go` (delete osCommander/osUpdateFS/osUpdateEnv/osDirEntry/osFileInfo; add updateDeps/newUpdateDeps/updateFSFromEdge/updateEnvFromDeps; new runUpdate signature; drop `os`, `os/exec` imports)
- Modify: `internal/cli/targets.go` (update target call site)
- Modify: `internal/cli/export_test.go` (drop 3 adapter exports; add updateDeps exports + internal/update import)
- Modify: `internal/cli/update_test.go` (delete 13 adapter tests — 3 commander + 1 env + 9 FS; rewrite 2 runUpdate smoke tests over test doubles)
- Create: `internal/cli/update_deps_test.go` (pure-composition unit tests)

**Interfaces**
- Consumes: `cli.Primitives`/`cli.NewDeps` (T1-rework); `cli.Deps` fields `FS EdgeFS`, `Getenv func(string) string`, `Getwd func() (string, error)`, `UserHomeDir func() (string, error)`, `Commander update.Commander` (deps.go:35 — field already landed), `Stdout io.Writer`; EdgeFS methods `ReadFile/WriteFile/MkdirAll/Stat/ReadDir/RemoveAll`; `update.ErrCommandNotFound` (T16).
- Produces: `Primitives.RunCommand` + `Primitives.NotFoundErr` (shapes above); unexported `primCommander` (the production `update.Commander`); `func newUpdateDeps(d Deps) updateDeps` (pure); `func runUpdate(ctx context.Context, args UpdateArgs, deps updateDeps, stdout io.Writer) error`.

**Steps**

1. [ ] RED: create `internal/cli/commander_test.go` — unit tests driving the composed commander from fake primitives through the real `NewDeps` wiring path (doctrine item 2: composition unit-tested with fake primitives):
   ```go
   package cli_test

   import (
   	"context"
   	"errors"
   	"fmt"
   	"io"
   	"testing"

   	. "github.com/onsi/gomega"

   	"github.com/toejough/engram/internal/cli"
   	"github.com/toejough/engram/internal/update"
   )

   // commanderOver builds the composed update.Commander from a fake RunCommand
   // primitive and an injected platform not-found sentinel, through the real
   // NewDeps wiring path (nil Getenv skips Embed; nil exit skips the
   // force-exit watcher).
   func commanderOver(
   	run func(ctx context.Context, dir, name string, args []string, stdout, stderr io.Writer) error,
   	notFound error,
   ) update.Commander {
   	prims := cli.Primitives{RunCommand: run, NotFoundErr: notFound}

   	return cli.NewDeps(prims, io.Discard, io.Discard, nil).Commander
   }

   func TestCommander_CollectsOutputOnSuccess(t *testing.T) {
   	t.Parallel()

   	g := NewWithT(t)

   	run := func(_ context.Context, _, _ string, _ []string, stdout, stderr io.Writer) error {
   		_, _ = stdout.Write([]byte("out-bytes"))
   		_, _ = stderr.Write([]byte("err-bytes"))

   		return nil
   	}

   	stdout, stderr, err := commanderOver(run, nil).Run(context.Background(), "", "tool")
   	g.Expect(err).NotTo(HaveOccurred())
   	g.Expect(string(stdout)).To(Equal("out-bytes"))
   	g.Expect(string(stderr)).To(Equal("err-bytes"))
   }

   func TestCommander_NilNotFoundErrNeverTranslates(t *testing.T) {
   	t.Parallel()

   	g := NewWithT(t)

   	errSpawn := errors.New("spawn failed")
   	run := func(_ context.Context, _, _ string, _ []string, _, _ io.Writer) error {
   		return errSpawn
   	}

   	_, _, err := commanderOver(run, nil).Run(context.Background(), "", "tool")
   	g.Expect(err).To(MatchError(errSpawn))
   	g.Expect(err).NotTo(MatchError(update.ErrCommandNotFound))
   }

   func TestCommander_PassesCallThrough(t *testing.T) {
   	t.Parallel()

   	g := NewWithT(t)

   	var gotDir, gotName string

   	var gotArgs []string

   	run := func(_ context.Context, dir, name string, args []string, _, _ io.Writer) error {
   		gotDir, gotName, gotArgs = dir, name, args

   		return nil
   	}

   	_, _, err := commanderOver(run, nil).Run(context.Background(), "/work", "git", "clone", "url")
   	g.Expect(err).NotTo(HaveOccurred())
   	g.Expect(gotDir).To(Equal("/work"))
   	g.Expect(gotName).To(Equal("git"))
   	g.Expect(gotArgs).To(Equal([]string{"clone", "url"}))
   }

   func TestCommander_TranslatesInjectedNotFound(t *testing.T) {
   	t.Parallel()

   	g := NewWithT(t)

   	errPlatformNotFound := errors.New("platform: executable file not found")
   	run := func(_ context.Context, _, _ string, _ []string, _, _ io.Writer) error {
   		return fmt.Errorf("spawning: %w", errPlatformNotFound)
   	}

   	_, _, err := commanderOver(run, errPlatformNotFound).Run(context.Background(), "", "ghost")
   	g.Expect(err).To(MatchError(update.ErrCommandNotFound))
   	g.Expect(err).To(MatchError(errPlatformNotFound))
   }

   func TestCommander_WrapsFailureAndKeepsOutput(t *testing.T) {
   	t.Parallel()

   	g := NewWithT(t)

   	errBoom := errors.New("boom")
   	run := func(_ context.Context, _, _ string, _ []string, stdout, stderr io.Writer) error {
   		_, _ = stdout.Write([]byte("partial"))
   		_, _ = stderr.Write([]byte("diagnostic"))

   		return errBoom
   	}

   	stdout, stderr, err := commanderOver(run, errors.New("not-found")).Run(
   		context.Background(), "", "tool", "arg")
   	g.Expect(err).To(MatchError(errBoom))
   	g.Expect(err).NotTo(MatchError(update.ErrCommandNotFound))
   	g.Expect(err).To(MatchError(ContainSubstring("tool [arg]")))
   	g.Expect(string(stdout)).To(Equal("partial"))
   	g.Expect(string(stderr)).To(Equal("diagnostic"))
   }
   ```
   Run `targ test` → expect FAIL (compile: `RunCommand`/`NotFoundErr` are not fields of `cli.Primitives` yet — the composition does not exist).

2. [ ] GREEN: add the primitives and the internal composition.
   - **2a.** Modify `internal/cli/primitives.go`. Add `"context"` to the import block, and insert a new field group into `Primitives` between the debug-sink group and the signal group (the doctrine's canonical struct grows here exactly as its future-task hooks anticipate — same mechanism T14 uses for backend/cache):
     ```go
     	// External command execution (doctrine flag C-1: one erased run closure
     	// + the platform not-found sentinel value; collection, wrapping, and
     	// not-found translation live internal in primCommander).
     	RunCommand func(
     		ctx context.Context, dir, name string, args []string, stdout, stderr io.Writer,
     	) error // closure: exec.CommandContext; Dir/Stdout/Stderr assignment; Run
     	NotFoundErr error // exec.ErrNotFound
     ```
     In `NewDeps`, add one field line to the `deps := Deps{...}` literal directly below `Lock:` (gofmt realigns the literal):
     ```go
     		Commander: primCommander{prims: prims},
     ```
     Wiring is unconditional, matching primFS/primLocker: a Deps built from fake primitives without `RunCommand` panics only if update actually runs — the same posture as every other unwired capability.
   - **2b.** Create `internal/cli/commander.go`:
     ```go
     package cli

     import (
     	"bytes"
     	"context"
     	"errors"
     	"fmt"

     	"github.com/toejough/engram/internal/update"
     )

     // Compile-time interface conformance (internal — the thin-api checker
     // does not walk internal/).
     var _ update.Commander = primCommander{}

     // primCommander is the production update.Commander: it composes the
     // injected raw run primitive with output collection, contextual %w
     // wrapping, and the platform-not-found → update.ErrCommandNotFound
     // translation (doctrine flag C-1). cmd/engram contributes only the
     // exec.CommandContext closure and the exec.ErrNotFound sentinel value;
     // ALL policy lives here (#700).
     type primCommander struct {
     	prims Primitives
     }

     // Run executes name with args in dir (empty dir inherits the process
     // cwd), returning captured stdout and stderr. A failure whose chain
     // matches the injected NotFoundErr is additionally tagged
     // update.ErrCommandNotFound per the Commander contract; errors.Is with
     // a nil target matches no non-nil error, so an unwired NotFoundErr
     // merely disables translation.
     func (c primCommander) Run(
     	ctx context.Context, dir, name string, args ...string,
     ) ([]byte, []byte, error) {
     	stdout := &bytes.Buffer{}
     	stderr := &bytes.Buffer{}

     	runErr := c.prims.RunCommand(ctx, dir, name, args, stdout, stderr)
     	if runErr != nil {
     		if errors.Is(runErr, c.prims.NotFoundErr) {
     			return stdout.Bytes(), stderr.Bytes(),
     				fmt.Errorf("%s %v: %w: %w", name, args, update.ErrCommandNotFound, runErr)
     		}

     		return stdout.Bytes(), stderr.Bytes(), fmt.Errorf("%s %v: %w", name, args, runErr)
     	}

     	return stdout.Bytes(), stderr.Bytes(), nil
     }
     ```
   Run `targ test` → expect PASS (step-1 suite green). Run `targ check-full` → expect clean.

3. [ ] Integration relocation — the former cmd adapter suite runs the COMPOSED primCommander over the REAL exec primitive in internal `_test` files (sanctioned: the T-final-1 purity lint excludes `!$test`).
   - **3a.** Modify `internal/cli/primitives_integration_test.go`: extend `realPrimitives()`'s returned literal after the `OpenDebugFile` entry (it must keep mirroring cmd/engram/main.go's literal — doctrine flag DRIFT), and add `"context"` and `"os/exec"` to the file's imports (`io` is already there):
     ```go
     		RunCommand: func(
     			ctx context.Context, dir, name string, args []string, stdout, stderr io.Writer,
     		) error {
     			cmd := exec.CommandContext(ctx, name, args...) //nolint:gosec // test-chosen name/args
     			cmd.Dir = dir
     			cmd.Stdout = stdout
     			cmd.Stderr = stderr

     			return cmd.Run() //nolint:wrapcheck // raw platform error; wrapping is primCommander's job
     		},
     		NotFoundErr: exec.ErrNotFound,
     ```
   - **3b.** Create `internal/cli/commander_integration_test.go` (the relocated `TestOsCommander_*` coverage):
     ```go
     package cli_test

     import (
     	"context"
     	"path/filepath"
     	"strings"
     	"testing"

     	. "github.com/onsi/gomega"

     	"github.com/toejough/engram/internal/update"
     )

     // These tests drive the composed primCommander over the REAL exec
     // primitive (realPrimitives mirrors cmd/engram/main.go's literal —
     // doctrine flag DRIFT): the relocated TestOsCommander_* coverage
     // (#700 rework — integration tests with real os funcs live in
     // internal _test files).

     func TestCommanderIntegration_ReportsFailure(t *testing.T) {
     	t.Parallel()

     	g := NewWithT(t)

     	commander := realDepsForTest().Commander

     	_, _, err := commander.Run(context.Background(), "", "false")
     	g.Expect(err).To(HaveOccurred())
     }

     func TestCommanderIntegration_RunsCommand(t *testing.T) {
     	t.Parallel()

     	g := NewWithT(t)

     	commander := realDepsForTest().Commander

     	stdout, _, err := commander.Run(context.Background(), "", "echo", "hello world")
     	g.Expect(err).NotTo(HaveOccurred())
     	g.Expect(strings.TrimSpace(string(stdout))).To(Equal("hello world"))
     }

     func TestCommanderIntegration_RunsInDir(t *testing.T) {
     	t.Parallel()

     	g := NewWithT(t)

     	commander := realDepsForTest().Commander
     	dir := t.TempDir()

     	// macOS TempDir sits under a symlink (/tmp → /private/tmp); compare
     	// against the resolved path so `pwd` output matches.
     	resolved, evalErr := filepath.EvalSymlinks(dir)
     	g.Expect(evalErr).NotTo(HaveOccurred())

     	if evalErr != nil {
     		return
     	}

     	stdout, _, err := commander.Run(context.Background(), dir, "pwd")
     	g.Expect(err).NotTo(HaveOccurred())
     	g.Expect(strings.TrimSpace(string(stdout))).To(Equal(resolved))
     }

     func TestCommanderIntegration_TranslatesNotFound(t *testing.T) {
     	t.Parallel()

     	g := NewWithT(t)

     	commander := realDepsForTest().Commander

     	_, _, err := commander.Run(context.Background(), "", "engram-no-such-binary-7f3a")
     	g.Expect(err).To(MatchError(update.ErrCommandNotFound))
     }
     ```
   Run `targ test` → expect PASS (refactor-parallel: the transitional internal osCommander suite still passes alongside; its deletion is the next step).

4. [ ] Rewrite `internal/cli/update.go` — delete the adapters, add pure composition, retarget runUpdate.
   - Imports (lines 3–17): delete `"os"` and `"os/exec"`; keep `bytes`, `context`, `errors`, `fmt`, `io`, `io/fs`, `path/filepath`, `slices`, `strings`, and the `internal/update` import.
   - Replace the unexported-variables block (lines 34–40):
     ```go
     // unexported variables.
     var (
     	_                      update.Filesystem = (*osUpdateFS)(nil)
     	errSomeHarnessesFailed                   = errors.New(
     		"update: one or more detected harnesses failed",
     	)
     )
     ```
     with:
     ```go
     // unexported variables.
     var (
     	_ update.Env        = (*updateEnvFromDeps)(nil)
     	_ update.Filesystem = (*updateFSFromEdge)(nil)

     	errSomeHarnessesFailed = errors.New(
     		"update: one or more detected harnesses failed",
     	)
     )
     ```
   - Delete entirely: `osCommander` type + `Run` method (pre-T16 lines 42–60; T16's translation grew Run, so delete by symbol), `osDirEntry` + methods (62–66), `osFileInfo` + method (68–70), `osUpdateEnv` + methods (72–84), the `// --- production adapters ---` comment (86), `osUpdateFS` + all six methods (88–151).
   - In their place add the pure composition (zero I/O — depguard-safe):
     ```go
     // updateDeps carries the injected surfaces Updater.Run needs. Composed
     // from the CLI-wide Deps by newUpdateDeps — pure plumbing, no I/O (#700).
     type updateDeps struct {
     	FS  update.Filesystem
     	Cmd update.Commander
     	Env update.Env
     }

     // newUpdateDeps composes update's dependency surface from cli.Deps.
     func newUpdateDeps(d Deps) updateDeps {
     	return updateDeps{
     		FS:  &updateFSFromEdge{fs: d.FS},
     		Cmd: d.Commander,
     		Env: &updateEnvFromDeps{
     			getenv:      d.Getenv,
     			getwd:       d.Getwd,
     			userHomeDir: d.UserHomeDir,
     		},
     	}
     }

     // updateEnvFromDeps adapts cli.Deps' env funcs to update.Env.
     type updateEnvFromDeps struct {
     	getenv      func(string) string
     	getwd       func() (string, error)
     	userHomeDir func() (string, error)
     }

     func (e *updateEnvFromDeps) Getenv(key string) string { return e.getenv(key) }

     func (e *updateEnvFromDeps) Getwd() (string, error) { return e.getwd() }

     func (e *updateEnvFromDeps) UserHomeDir() (string, error) { return e.userHomeDir() }

     // updateFSFromEdge adapts the CLI-wide EdgeFS to update.Filesystem. Pure
     // interface plumbing: fs.DirEntry / fs.FileInfo structurally satisfy
     // update.DirEntry / update.FileInfo. Errors pass through unwrapped so
     // errors.Is(err, fs.ErrNotExist) checks in the update package keep working.
     type updateFSFromEdge struct {
     	fs EdgeFS
     }

     func (a *updateFSFromEdge) MkdirAll(path string, perm fs.FileMode) error {
     	return a.fs.MkdirAll(path, perm) //nolint:wrapcheck // pass-through; update core adds context
     }

     func (a *updateFSFromEdge) ReadDir(path string) ([]update.DirEntry, error) {
     	entries, err := a.fs.ReadDir(path)
     	if err != nil {
     		//nolint:wrapcheck // caller distinguishes fs.ErrNotExist via errors.Is
     		return nil, err
     	}

     	out := make([]update.DirEntry, 0, len(entries))
     	for _, entry := range entries {
     		out = append(out, entry)
     	}

     	return out, nil
     }

     func (a *updateFSFromEdge) ReadFile(path string) ([]byte, error) {
     	data, err := a.fs.ReadFile(path)
     	if err != nil {
     		//nolint:wrapcheck // caller distinguishes fs.ErrNotExist via errors.Is
     		return nil, err
     	}

     	return data, nil
     }

     func (a *updateFSFromEdge) RemoveAll(path string) error {
     	return a.fs.RemoveAll(path) //nolint:wrapcheck // pass-through; update core adds context
     }

     func (a *updateFSFromEdge) Stat(path string) (update.FileInfo, error) {
     	info, err := a.fs.Stat(path)
     	if err != nil {
     		//nolint:wrapcheck // caller distinguishes fs.ErrNotExist via errors.Is
     		return nil, err
     	}

     	return info, nil
     }

     func (a *updateFSFromEdge) WriteFile(path string, data []byte, perm fs.FileMode) error {
     	return a.fs.WriteFile(path, data, perm) //nolint:wrapcheck // pass-through; update core adds context
     }
     ```
   - Replace `runUpdate` (lines 275–295):
     ```go
     // runUpdate wires production adapters and invokes Updater.Run.
     func runUpdate(ctx context.Context, args UpdateArgs, stdout io.Writer) error {
     	updater := &update.Updater{
     		FS:  &osUpdateFS{},
     		Cmd: &osCommander{},
     		Env: &osUpdateEnv{},
     	}

     	report, runErr := updater.Run(ctx, update.Options{
     		DryRun:       args.DryRun,
     		WithGuidance: args.WithGuidance,
     	})
     	if runErr == nil {
     		vaultPath := resolveVault("", report.Home, updater.Env.Getenv)
     		report.VaultHasOldVocabFiles = oldVocabFilesPresent(vaultPath, updater.FS)
     		chunksDir := ResolveChunksDir("", report.Home, updater.Env.Getenv)
     		report.ChunkIndexHasEmptyFiles = chunkIndexHasEmptyFiles(chunksDir, updater.FS)
     	}

     	return finishUpdate(stdout, report, runErr)
     }
     ```
     with:
     ```go
     // runUpdate invokes Updater.Run over the injected dependency surface.
     func runUpdate(ctx context.Context, args UpdateArgs, deps updateDeps, stdout io.Writer) error {
     	updater := &update.Updater{
     		FS:  deps.FS,
     		Cmd: deps.Cmd,
     		Env: deps.Env,
     	}

     	report, runErr := updater.Run(ctx, update.Options{
     		DryRun:       args.DryRun,
     		WithGuidance: args.WithGuidance,
     	})
     	if runErr == nil {
     		vaultPath := resolveVault("", report.Home, deps.Env.Getenv)
     		report.VaultHasOldVocabFiles = oldVocabFilesPresent(vaultPath, deps.FS)
     		chunksDir := ResolveChunksDir("", report.Home, deps.Env.Getenv)
     		report.ChunkIndexHasEmptyFiles = chunkIndexHasEmptyFiles(chunksDir, deps.FS)
     	}

     	return finishUpdate(stdout, report, runErr)
     }
     ```
     (ENGRAM_VAULT_PATH / ENGRAM_CHUNKS_DIR reads thereby flow through `deps.Env.Getenv` ← `cli.Deps.Getenv` — no separate env work needed for this family.)

5. [ ] Retarget the call site in `internal/cli/targets.go` (post-T2, `learnUpdateTargets` has `deps Deps` in scope and the closure reads `deps.Stdout`). Replace:
   ```go
   		targ.Targ(func(ctx context.Context, a UpdateArgs) {
   			errHandler(runUpdate(withLog(ctx), a, deps.Stdout))
   		}).Name("update").Description("Refresh engram binary and harness skills"),
   ```
   with:
   ```go
   		targ.Targ(func(ctx context.Context, a UpdateArgs) {
   			errHandler(runUpdate(withLog(ctx), a, newUpdateDeps(deps), deps.Stdout))
   		}).Name("update").Description("Refresh engram binary and harness skills"),
   ```

6. [ ] Wire the primitives in `cmd/engram/main.go` — add exactly two field lines to the `cli.Primitives` literal (directly below the `OpenDebugFile` entry, mirroring step 3a's `realPrimitives()` extension) and `"context"`, `"io"`, `"os/exec"` to the import block. `main()` remains ONE statement and package main remains declaration-free; the closure is an expression the checker does not walk, and its body is the enumerated stdlib-equivalent survivor shape sanctioned by doctrine flag C-1 (construction + field assignments + one invocation, zero branching — behavior changes extend the SIGNATURE, never this body):
   ```go
   			RunCommand: func(
   				ctx context.Context, dir, name string, args []string, stdout, stderr io.Writer,
   			) error {
   				cmd := exec.CommandContext(ctx, name, args...) //nolint:gosec // name/args from internal callers
   				cmd.Dir = dir
   				cmd.Stdout = stdout
   				cmd.Stderr = stderr

   				return cmd.Run() //nolint:wrapcheck // raw platform error; wrapping is internal policy (C-1)
   			},
   			NotFoundErr: exec.ErrNotFound,
   ```
   If `targ check-thin-api` flags anything in this file after the edit, ESCALATE the exact finding to the orchestrator (doctrine item 5) — do not suppress, do not restructure ad hoc.

7. [ ] `internal/cli/export_test.go`:
   - Delete lines 523–524 (`ExportNewOsCommander`), 566–567 (`ExportNewOsUpdateEnv`), 569–570 (`ExportNewOsUpdateFS`) including their doc comments.
   - `ExportRunUpdate = runUpdate` (line 108) stays — the alias picks up the new signature.
   - Add to the import block `"github.com/toejough/engram/internal/update"`, and add:
     ```go
     // ExportNewUpdateDeps exposes the production pure composition for tests.
     var ExportNewUpdateDeps = newUpdateDeps

     // ExportNewUpdateDepsFrom builds the unexported updateDeps from explicit
     // surfaces so black-box tests can drive runUpdate with test doubles.
     func ExportNewUpdateDepsFrom(fs update.Filesystem, cmd update.Commander, env update.Env) updateDeps {
     	return updateDeps{FS: fs, Cmd: cmd, Env: env}
     }
     ```
     (Place the var alphabetically in the existing var block; the func with the other Export funcs.)

8. [ ] `internal/cli/update_test.go`:
   - Delete `TestOsCommander_ReportsFailure`, `TestOsCommander_RunsCommand`, `TestOsCommander_TranslatesNotFound` (T16's RED test — all three re-covered by step 3b's integration suite), `TestOsUpdateEnv_ReturnsValues`, and all nine `TestOsUpdateFS_*` tests plus the `// osUpdateFS round-trip tests:` comment (pre-T16 lines 235–441, minus `TestPluralFile` at 443; T16's insertion shifts later numbers — delete by name). The nine FS round-trips hand their real-FS coverage to `internal/cli/primitives_integration_test.go` (supersession map).
   - Rewrite the two runUpdate smoke tests and add the test doubles (file-local; `os` and `io/fs` already importable in _test.go — `fs` needs adding to imports):
     ```go
     // liveUpdateEnv adapts the real process environment to update.Env for the
     // dry-run smoke tests (production Env is composed from cli.Deps).
     type liveUpdateEnv struct{}

     func (liveUpdateEnv) Getenv(key string) string { return os.Getenv(key) }

     func (liveUpdateEnv) Getwd() (string, error) {
     	return os.Getwd() //nolint:wrapcheck // test adapter
     }

     func (liveUpdateEnv) UserHomeDir() (string, error) {
     	return os.UserHomeDir() //nolint:wrapcheck // test adapter
     }

     // liveUpdateFS is an os-backed update.Filesystem for the dry-run smoke
     // tests (dry-run never writes; write methods exist to satisfy the interface).
     type liveUpdateFS struct{}

     func (liveUpdateFS) MkdirAll(path string, perm fs.FileMode) error {
     	return os.MkdirAll(path, perm) //nolint:wrapcheck // test adapter
     }

     func (liveUpdateFS) ReadDir(path string) ([]update.DirEntry, error) {
     	entries, err := os.ReadDir(path)
     	if err != nil {
     		return nil, err //nolint:wrapcheck // errors.Is(fs.ErrNotExist) must survive
     	}

     	out := make([]update.DirEntry, 0, len(entries))
     	for _, entry := range entries {
     		out = append(out, entry)
     	}

     	return out, nil
     }

     func (liveUpdateFS) ReadFile(path string) ([]byte, error) {
     	return os.ReadFile(path) //nolint:wrapcheck,gosec // test adapter; test-chosen paths
     }

     func (liveUpdateFS) RemoveAll(path string) error {
     	return os.RemoveAll(path) //nolint:wrapcheck // test adapter
     }

     func (liveUpdateFS) Stat(path string) (update.FileInfo, error) {
     	info, err := os.Stat(path)
     	if err != nil {
     		return nil, err //nolint:wrapcheck // errors.Is(fs.ErrNotExist) must survive
     	}

     	return info, nil
     }

     func (liveUpdateFS) WriteFile(path string, data []byte, perm fs.FileMode) error {
     	return os.WriteFile(path, data, perm) //nolint:wrapcheck // test adapter
     }

     // stubCommander satisfies update.Commander; dry-run local mode never runs it.
     type stubCommander struct{}

     func (stubCommander) Run(context.Context, string, string, ...string) ([]byte, []byte, error) {
     	return nil, nil, nil
     }
     ```
     and replace the two tests' bodies:
     ```go
     func TestRunUpdate_DryRunFromCwd(t *testing.T) {
     	t.Parallel()

     	g := NewWithT(t)

     	stdout := &bytes.Buffer{}
     	deps := cli.ExportNewUpdateDepsFrom(liveUpdateFS{}, stubCommander{}, liveUpdateEnv{})

     	// Dry-run against the live filesystem: cwd is inside the engram
     	// worktree, so source resolution picks local mode without `go install`.
     	err := cli.ExportRunUpdate(context.Background(), cli.UpdateArgs{DryRun: true}, deps, stdout)
     	out := stdout.String()

     	if err != nil {
     		g.Expect(err.Error()).To(ContainSubstring("update"))

     		return
     	}

     	g.Expect(out).To(ContainSubstring("[dry-run] engram update"))
     	g.Expect(out).To(ContainSubstring("source: local clone at "))
     }

     func TestRunUpdate_WithGuidanceFlagMapsToOptions(t *testing.T) {
     	t.Parallel()

     	g := NewWithT(t)

     	stdout := &bytes.Buffer{}
     	deps := cli.ExportNewUpdateDepsFrom(liveUpdateFS{}, stubCommander{}, liveUpdateEnv{})

     	// Dry-run with --with-guidance; only verifies the flag maps to Options.
     	err := cli.ExportRunUpdate(
     		context.Background(), cli.UpdateArgs{DryRun: true, WithGuidance: true}, deps, stdout)
     	if err != nil {
     		g.Expect(err.Error()).To(ContainSubstring("update"))
     	}
     }
     ```
     Add `"io/fs"` to the file's imports (as `fs`), keep `os`, `context`, etc.

9. [ ] Create `internal/cli/update_deps_test.go` — pure-composition unit tests over a fake EdgeFS (hand fakes match this family's precedent: u1FS/fakeCmd):
    ```go
    package cli_test

    import (
    	"context"
    	"errors"
    	"io/fs"
    	"testing"
    	"testing/fstest"

    	. "github.com/onsi/gomega"

    	"github.com/toejough/engram/internal/cli"
    	"github.com/toejough/engram/internal/update"
    )

    func TestNewUpdateDeps_CommanderPassesThrough(t *testing.T) {
    	t.Parallel()

    	g := NewWithT(t)

    	cmd := stubCommander{}
    	deps := cli.ExportNewUpdateDeps(cli.Deps{Commander: cmd, FS: updateFakeEdgeFS{}})

    	stdout, stderr, err := deps.Cmd.Run(context.Background(), "", "x")
    	g.Expect(err).NotTo(HaveOccurred())
    	g.Expect(stdout).To(BeNil())
    	g.Expect(stderr).To(BeNil())
    }

    func TestNewUpdateDeps_EnvDelegatesToDepsFuncs(t *testing.T) {
    	t.Parallel()

    	g := NewWithT(t)

    	deps := cli.ExportNewUpdateDeps(cli.Deps{
    		FS:          updateFakeEdgeFS{},
    		Getenv:      func(key string) string { return "env:" + key },
    		Getwd:       func() (string, error) { return "/cwd", nil },
    		UserHomeDir: func() (string, error) { return "/home/x", nil },
    	})

    	g.Expect(deps.Env.Getenv("K")).To(Equal("env:K"))

    	cwd, cwdErr := deps.Env.Getwd()
    	g.Expect(cwdErr).NotTo(HaveOccurred())
    	g.Expect(cwd).To(Equal("/cwd"))

    	home, homeErr := deps.Env.UserHomeDir()
    	g.Expect(homeErr).NotTo(HaveOccurred())
    	g.Expect(home).To(Equal("/home/x"))
    }

    func TestNewUpdateDeps_FSAdapterPreservesNotExist(t *testing.T) {
    	t.Parallel()

    	g := NewWithT(t)

    	deps := cli.ExportNewUpdateDeps(cli.Deps{FS: updateFakeEdgeFS{}})

    	_, readErr := deps.FS.ReadFile("/missing")
    	g.Expect(errors.Is(readErr, fs.ErrNotExist)).To(BeTrue())

    	_, dirErr := deps.FS.ReadDir("/missing")
    	g.Expect(errors.Is(dirErr, fs.ErrNotExist)).To(BeTrue())

    	_, statErr := deps.FS.Stat("/missing")
    	g.Expect(errors.Is(statErr, fs.ErrNotExist)).To(BeTrue())
    }

    func TestNewUpdateDeps_FSAdapterReadsThroughEdgeFS(t *testing.T) {
    	t.Parallel()

    	g := NewWithT(t)

    	deps := cli.ExportNewUpdateDeps(cli.Deps{FS: updateFakeEdgeFS{
    		"skills/learn/SKILL.md": &fstest.MapFile{Data: []byte("learn")},
    	}})

    	data, readErr := deps.FS.ReadFile("skills/learn/SKILL.md")
    	g.Expect(readErr).NotTo(HaveOccurred())
    	g.Expect(string(data)).To(Equal("learn"))

    	entries, dirErr := deps.FS.ReadDir("skills")
    	g.Expect(dirErr).NotTo(HaveOccurred())

    	if dirErr != nil {
    		return
    	}

    	g.Expect(entries).To(HaveLen(1))
    	g.Expect(entries[0].Name()).To(Equal("learn"))
    	g.Expect(entries[0].IsDir()).To(BeTrue())

    	info, statErr := deps.FS.Stat("skills/learn/SKILL.md")
    	g.Expect(statErr).NotTo(HaveOccurred())

    	if statErr != nil || info == nil {
    		return
    	}

    	g.Expect(info.IsDir()).To(BeFalse())
    }

    // updateFakeEdgeFS is a read-only in-memory cli.EdgeFS over fstest.MapFS.
    // Write-side methods return errUnsupported: the update dry-run/read paths
    // under test never invoke them.
    type updateFakeEdgeFS fstest.MapFS

    func (m updateFakeEdgeFS) MkdirAll(string, fs.FileMode) error { return errUnsupported }

    func (m updateFakeEdgeFS) MkdirTemp(string, string) (string, error) { return "", errUnsupported }

    func (m updateFakeEdgeFS) ReadDir(path string) ([]fs.DirEntry, error) {
    	return fs.ReadDir(fstest.MapFS(m), path) //nolint:wrapcheck // fake passes chains through
    }

    func (m updateFakeEdgeFS) ReadFile(path string) ([]byte, error) {
    	return fs.ReadFile(fstest.MapFS(m), path) //nolint:wrapcheck // fake passes chains through
    }

    func (m updateFakeEdgeFS) Remove(string) error { return errUnsupported }

    func (m updateFakeEdgeFS) RemoveAll(string) error { return errUnsupported }

    func (m updateFakeEdgeFS) Rename(string, string) error { return errUnsupported }

    func (m updateFakeEdgeFS) Stat(path string) (fs.FileInfo, error) {
    	return fs.Stat(fstest.MapFS(m), path) //nolint:wrapcheck // fake passes chains through
    }

    func (m updateFakeEdgeFS) WalkDir(root string, fn fs.WalkDirFunc) error {
    	return fs.WalkDir(fstest.MapFS(m), root, fn) //nolint:wrapcheck // fake passes chains through
    }

    func (m updateFakeEdgeFS) WriteFile(string, []byte, fs.FileMode) error { return errUnsupported }

    func (m updateFakeEdgeFS) WriteFileAtomic(string, []byte, fs.FileMode) error { return errUnsupported }

    func (m updateFakeEdgeFS) WriteFileExcl(string, []byte, fs.FileMode) error { return errUnsupported }

    // unexported variables.
    var errUnsupported = errors.New("updateFakeEdgeFS: write path not supported")
    ```
    Note for the executor: sync `updateFakeEdgeFS`'s method set with the LANDED `cli.EdgeFS` (R13: T8 owns the cli_test `fakeEdgeFS` name; this one is `updateFakeEdgeFS`, distinct by design). `fstest.MapFS` paths are slash-relative (no leading `/`), hence the relative paths above; its error chains wrap `fs.ErrNotExist`, which is exactly the property under test.
    Run `targ test` → expect PASS. Run `targ check-full` → expect clean.

10. [ ] Purity + gate verification for this family:
    - `grep -rn '"os"\|"os/exec"' internal/cli/update.go internal/update/update.go` → no hits.
    - `grep -rln '"os/exec"' internal/ | grep -v _test` → no hits (os/exec now enters only through cmd/engram's Primitives literal; `_test` files are sanctioned by the doctrine).
    - `targ test` → green; `targ check-full` → clean.
    - `targ check-thin-api` → PASS (cmd/engram still holds only the declaration-free main.go; the RunCommand closure is an expression the checker does not walk — enumerated survivor C-1, human-enforced via the doctrine's survivor list and its behavior-mirror test). If it flags ANYTHING, escalate the exact finding — never suppress (Global Constraints).
    - `go install ./cmd/engram && engram update --dry-run` from the worktree root → expect `[dry-run] engram update` + `source: local clone at ...` output (real-binary check per house rule; exercises Primitives.RunCommand → primCommander → newUpdateDeps → Updater.Run end to end).

11. [ ] Commit (via the commit skill):
    ```
    refactor(cli): commander via injected run primitive (#700)

    os/exec leaves internal/ entirely: cmd/engram's Primitives literal
    contributes one erased exec.CommandContext run closure (RunCommand)
    plus the exec.ErrNotFound sentinel value (NotFoundErr, doctrine flag
    C-1); internal primCommander owns output collection, %w wrapping, and
    the update.ErrCommandNotFound translation. osUpdateFS/osUpdateEnv are
    absorbed into pure bridges over cli.Deps (EdgeFS bridge + env-func
    bridge); runUpdate takes an injected updateDeps.
    internal/cli/update.go is now I/O-import-free.

    AI-Used: [claude]
    ```

### Task T-final-1: Enforcement flip — depguard + forbidigo land with zero carve-outs

**Files:**
- Modify: `dev/golangci-lint.toml`

**Interfaces:**
- Consumes: the fully-migrated tree (all prior tasks complete; no os/exec/signal/syscall/hugot imports and no time.Now/Since/Tick references remain in internal/ non-test code).
- Produces: the enforced purity boundary; `targ check-full` fails on any internal-side regression, and `targ check-thin-api` (authoritative, unchanged by this task) keeps guarding the cmd side.

- [ ] **Step 1: RED — add the depguard rule and confirm it currently passes only because migration is done.** Add to `dev/golangci-lint.toml` alongside the existing `[linters.settings.depguard.rules.all]`:

```toml
# #700: internal/ purity — default-deny. Anything not prefix-matched below is denied
# in internal non-test code. NO file carve-outs: raw I/O primitives enter from
# cmd/engram's declaration-free main (cli.Primitives; targ check-thin-api enforces that
# side), ALL adapter composition is injected in internal/cli, and the only real-os code
# under internal/ sits in _test files — excluded via '!$test' (sanctioned by the revised
# composition doctrine).
# Glob form per R9: start with the issue-AC literal 'internal/**'; Step 5's negative
# probe validates it fires. Fall back to the prototype-confirmed '**/internal/**'
# ONLY if the probe stays silent, and amend the issue AC wording (see R9).
[linters.settings.depguard.rules.internal-purity]
files = ['internal/**', '!$test']
allow = [
	'strings', 'fmt', 'errors', 'sort', 'slices', 'maps', 'strconv', 'unicode',
	'bufio', 'bytes', 'io', 'path', 'regexp',
	'encoding/json', 'encoding/hex', 'crypto/sha256', 'hash/fnv', 'math', 'time',
	'context', 'sync', 'embed',
	'github.com/toejough/engram',
	'go.yaml.in/yaml/v3',
	'github.com/toejough/targ',
]
```

Notes: prefix matching means `io` admits `io/fs`, `path` admits `path/filepath`, `sync` admits `sync/atomic`, `math` admits `math/rand/v2` (kmeans' seeded PCG — misuse is forbidigo's job below), `time` admits types/parsing (clock calls are forbidigo's job). No `deny` entries: a `math/rand` deny would prefix-catch the legal seeded v2 import.

- [ ] **Step 2: enable forbidigo with the custom list.** Remove `'forbidigo'` from the `disable` list and add:

```toml
[linters.settings.forbidigo]
# Custom list REPLACES the fmt.Print defaults — printing stays legal (repo prints on purpose).
analyze-types = true

[[linters.settings.forbidigo.forbid]]
pattern = '^time\.Now$'
msg = 'inject a clock via cli.Deps.Now (#700)'

[[linters.settings.forbidigo.forbid]]
pattern = '^time\.Since$'
msg = 'inject a clock via cli.Deps.Now (#700)'

[[linters.settings.forbidigo.forbid]]
pattern = '^time\.Tick$'
msg = 'inject a clock via cli.Deps.Now (#700)'

[[linters.settings.forbidigo.forbid]]
pattern = '^targ\.Main$'
msg = 'targ dispatch is edge work — call from cmd/engram only (#700)'

[[linters.settings.forbidigo.forbid]]
pattern = '.*'
pkg = '^math/rand$'
msg = 'math/rand v1 is banned — use a seeded math/rand/v2 *rand.Rand (#700)'

[[linters.settings.forbidigo.forbid]]
pattern = '^rand\.(N|Int32|Int64|Int32N|Int64N|IntN|Int|Uint32|Uint64|Uint32N|Uint64N|UintN|Uint|Float32|Float64|NormFloat64|ExpFloat64|Perm|Shuffle)$'
pkg = '^math/rand/v2$'
msg = 'auto-seeded global PRNG is banned — use a seeded *rand.Rand (#700)'
```

And scope it with exclusion rules (forbidigo applies only to internal non-test code):

```toml
[[linters.exclusions.rules]]
linters = ['forbidigo']
path-except = '^internal/'

[[linters.exclusions.rules]]
linters = ['forbidigo']
path = '_test\.go$'
```

- [ ] **Step 3: set `max-issues-per-linter = 0`** (deliberate: never truncate findings; the default 10 hid 14 of 24 findings during the plan-time prototype). Find the existing `max-issues-per-linter` line and set it to `0`.

- [ ] **Step 4: verify — `targ check-full` AND `targ check-thin-api`.** Expected: check-full GREEN, check-thin-api PASS (`All N public API files are thin wrappers.`). If depguard/forbidigo report findings, the migration missed a site — fix the SITE (relocate/thread it per the relevant task's pattern); NEVER add a carve-out, nolint, or allow-list entry to make it pass (that violates the issue's zero-grandfathering acceptance criteria; escalate to the orchestrator if a finding looks structural). If check-thin-api reports a finding, a prior task regressed the declaration-free cmd shape — escalate the exact finding (doctrine flag SIG-1); never suppress.

- [ ] **Step 5: negative self-test of the gate (temporary, not committed).** Add `_ = os.Getenv("PROBE")` (+ `"os"` import) to any internal/ non-test file; run `targ check-full`; expect a depguard finding naming `internal-purity`. Revert the probe. This proves the rule fires (a green gate that can't fail is no gate).

- [ ] **Step 5.2: loser-symbol grep gate.** `rg -n "edgeVaultFS|depsVaultFS|jsonlIndexesLister|jsonlIndexListerFrom|vaultLuhmannLock|warnLoggerTo|osListJSONLIndexes|ExportNewOsVaultFS" internal/ cmd/` must return ZERO hits: the parallel-drafting loser symbols (R1/R2/R3 — never legally declared anywhere) and the transitional shims (`osListJSONLIndexes`, deleted by T12; `ExportNewOsVaultFS`, call sites migrated by T12 per R12 and shim deleted by T7) must not exist in the final tree. Any hit → a task landed a loser or skipped its gated deletion; fix the SITE per the owning resolution (R1/R2/R3/R12) before proceeding — never rename-in-place to dodge the grep.

- [ ] **Step 5.5: coverage stance for cmd/engram (issue AC).** Post-rework, `cmd/engram` holds ONLY the declaration-free `main.go` — a single-statement `main()` over the `cli.Primitives` literal, `targ check-thin-api`-enforced, with no testable logic. The adapter logic and the tests that used to sit beside it in cmd now live in `internal/cli`: unit tests with fake primitives plus integration `_test` files with real os/syscall funcs, all counting toward internal coverage like any other internal tests; the production literal itself is guarded by cli_test.go's end-to-end binary build. Inspect how the coverage gate treats `cmd/engram`: read the check-full output's coverage section (and `rg -n "cmd/" dev/targs.go dev/*.toml` for exclusion patterns). Record the finding + decision as a one-paragraph note in the commit body of Step 6: the expected call is (a) `cmd/engram` stays coverage-excluded as an entry point (the repo's entry-point exclusion doctrine — wiring only, no testable logic). If the tooling instead shows main.go entering coverage uncovered, that is call (b): decide deliberately between an explicit entry-point exclusion and a trivial main-wiring smoke test (the most cmd may keep), and surface the choice to the orchestrator. Do not silently leave the stance undecided — the issue AC requires a deliberate call.

- [ ] **Step 6: Commit.**

```bash
git add dev/golangci-lint.toml
git commit -m "check(#700): enforce internal purity via depguard + forbidigo

Zero carve-outs: raw I/O primitives enter from cmd/engram's
declaration-free main (check-thin-api-enforced), adapter composition is
injected in internal/cli, and real-os code under internal/ is confined
to _test files ('!$test'), so the rule needs no file exceptions. Custom
forbidigo list replaces print defaults (printing stays legal).
max-issues-per-linter=0 so findings never truncate.

AI-Used: [claude]"
```

### Task T-final-2: FIXME removal + issue closure prep

**Files:**
- Modify: `cmd/engram/main.go` (the FIXME(#700) marker's home since T2's relocation — see R8)

**Interfaces:**
- Consumes: T-final-1 complete (`targ check-full` green with enforcement active).
- Produces: the resolved FIXME per the user's rule ("remove the FIXME only when the issue is resolved").

- [ ] **Step 1: verify the enforcement is green**: `targ check-full` → GREEN AND `targ check-thin-api` → PASS (fresh runs, not cached claims).
- [ ] **Step 2: delete the relocated `FIXME(#700)` marker block from `cmd/engram/main.go`** (T2 carried it there per R8: the comment block directly ABOVE `func main()`, beginning `// FIXME(#700): internal-purity migration in progress`). Delete ONLY that comment block — the declaration-free package (single-statement `main()`) is untouched. This is a real deletion of a marker that MUST still exist at this point — if `rg -n "FIXME\(#700\)" .` returns zero hits BEFORE this step, that is a defect (the marker was removed early, violating the user's rule): STOP and escalate to the orchestrator. After deletion, re-run the grep — zero hits is the deliverable.
- [ ] **Step 2.5: re-run the task-final gates** — `targ check-full` GREEN + `targ check-thin-api` PASS (a comment-only deletion must change neither; the checker sees only declarations, so any new thin-api finding here means the edit touched more than the comment — revert and redo Step 2).
- [ ] **Step 3: Commit.**

```bash
git add -A
git commit -m "chore(#700): remove resolved FIXME — purity boundary enforced

AI-Used: [claude]"
```

## Documentation surface (step-5 dispositions, Gate C verifies)

| File | Disposition | Reason |
|---|---|---|
| `CLAUDE.md` | update | directory-structure + Key Files: `internal/cli` becomes the composition root (`cli.NewDeps` builds every production adapter from injected `cli.Primitives`); `cmd/engram` stays a declaration-free single-statement entry point supplying raw primitives (`targ check-thin-api`-enforced); DI bullet gains "lint-enforced (depguard/forbidigo + check-thin-api, #700)"; line 43 stale ADR range `(ADR-0001..0003)` → `(ADR-0001..0020)` (or current top ADR at edit time) |
| `README.md` | update | line 127 "cmd/engram/ CLI entry point (thin wiring layer)" stays TRUE and is sharpened, not reversed: declaration-free `main()` over a `cli.Primitives` literal of raw capability references; all adapter composition lives in `internal/cli` (`cli.NewDeps`); enforced by `targ check-thin-api` |
| `docs/architecture/c3-components.md` | update | K11 row: replace with `\| K11 \| internal/debuglog \| tail-friendly sink (pure: writer+clock injected) \| Cross-cutting debug log threaded through every CLI target (targets.go); sink composition (openDebugSink + per-write-Sync syncWriter, env-gated) lives in internal/cli/debugsink.go; cmd/engram supplies only the raw OpenDebugFile primitive (#700). \| — \|`; ADD an edge-primitives row (next free K-id — K13 per the current K1–K12 inventory; re-verify at edit time): `\| K<n> \| cmd/engram \| edge primitives + entry point \| Declaration-free single-statement main() populating cli.Primitives (raw os/syscall/filepath/hugot/exec capability references + sanctioned closures); targ check-thin-api-enforced; ALL adapter composition (EdgeFS, FileLocker, commander, hugot backend, debug sink, signal force-exit) lives in internal/cli via cli.NewDeps, integration-tested there with real FS/env (#700). \| — \|`; mirror both in the mermaid block |
| `docs/architecture/adr.md` | update | Append to ADR-0001's Status line: `; #700 (2026-07): raw I/O primitives relocated to cmd/engram (declaration-free package main over cli.Primitives, targ check-thin-api-enforced); ALL adapter composition + wiring live in internal/cli (cli.NewDeps); internal/ is import-pure (lint-enforced, ADR-0020)`. Append to ADR-0013's Status line: `; #700 (2026-07): flock/atomic-rename lifecycle composed in internal/cli (primFS/primLocker over raw os/syscall primitives supplied by cmd/engram) — semantics unchanged, lock-at-Run*-entry convention preserved, concurrent-writers regression test carried (now an internal/cli integration test)`. Add NEW ADR-0020 with this draft text (Gate C polishes wording, not substance): **ADR-0020 — Enforced internal/ purity: raw I/O assignment in cmd/engram, all logic in internal/.** Status: Accepted (shipped via #700). Context: the DI doctrine ("wire at the edges" — CLAUDE.md's summary bullet, under ADR-0001..0003's authority) was convention-only; production I/O adapters lived inside internal/cli, internal/debuglog, internal/embed, and direct env reads had crept in (the #700 FIXME); testing internal code meant working around real I/O; and cmd thinness (targ's check-thin-api gate) forbids moving real adapter logic into package main. Decision: the boundary is absolute and two-sided — internal/ non-test code holds interfaces + ALL logic (adapter composition, error wrapping, lifecycle: EdgeFS atomic-write dance, flock open/lock/unlock-closure semantics, debug sink, signal force-exit, commander run-and-collect, embedder session/cache orchestration — built by cli.NewDeps from injected cli.Primitives) but imports no I/O packages; cmd/engram (package main) is declaration-free — a single-statement main() populating cli.Primitives with raw capability references (os.ReadFile, time.Now, filepath.WalkDir, syscall wrappers) and sanctioned closures (single-call signature-erasers plus the two enumerated stdlib-equivalent survivors, WriteFileExcl and RunCommand), zero orchestration; enforcement is config-only and two-gate — depguard default-deny allow-list over internal/ non-test files (zero file carve-outs; real-os integration tests live in internal _test files via the sanctioned '!$test' exclusion) + forbidigo call-level bans (time.Now/Since/Tick, math/rand v1, auto-seeded rand/v2 globals, targ.Main) on the internal side, and targ check-thin-api (authoritative) on the cmd side. Consequences: every internal package is testable by injection alone (unit tests with fake primitives; real-os integration tests as internal _test files); a new I/O capability requires a Primitives field + internal composition, both visible in review; both gates fail loud on regression; cmd/engram carries no testable logic and stays coverage-exempt as an entry point; seeded math/rand/v2 stays legal (deterministic computation) |
| `docs/GLOSSARY.md` | keep (verify) | cited files remain and `targets.go` still wires subcommands — verification only, no edit expected; if any entry describes os-level wiring, escalate rather than silently rewrite |
| `docs/architecture/c1-system-context.md` | keep (verify citations) | flows unchanged; update-flow + query citations still valid |
| `docs/architecture/c2-containers.md` | keep (verify) | C1/C2 skill-binary seam unchanged |
| `dev/eval/LEDGER.md` | keep | historical vintage-stamped measurement records — never retro-edited |
| `docs/superpowers/plans/2026-07-18-646-recency-value-proof.md` | keep | historical plan artifact |
| `skills/`, `commands/`, `guidance/` | n/a | grep-verified 2026-07-19: no Go-path references |
| `docs/design/2026-07-01-engram-recall-subprocess-design.md` | keep (verify) | line 84 states the "DI everywhere, no os/exec" invariant — remains TRUE (strengthened) post-refactor; verify wording needs no update |

## Merge protocol (repo rules)

Review-before-merge with argumentation; rebase on main + re-test before merging; `git merge --ff-only` only; rebase loop if another branch (two live Pi worktrees!) lands first; never push unreviewed work.
