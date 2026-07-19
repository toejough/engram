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
Closures (`ast.FuncLit`) inside expressions are NOT walked by the checker, but the doctrine still
caps cmd closures at single-call (or trivially-sequenced single-call) bodies — orchestration hidden
in a closure violates the DESIGN even where the checker cannot see it. The corrected shape:

1. **`cli.Primitives`** (internal/cli/primitives.go, landed by T1-rework) carries raw impure
   capabilities as func fields; cmd populates it with direct references (`os.ReadFile`,
   `filepath.WalkDir`, `time.Now`) or single-call closures where a signature must be erased. The
   canonical struct — downstream briefs consume these field names verbatim:

```go
type Primitives struct {
	// Filesystem (direct os/filepath references).
	ReadFile   func(path string) ([]byte, error)                      // os.ReadFile
	WriteFile  func(path string, data []byte, perm fs.FileMode) error // os.WriteFile
	MkdirAll   func(path string, perm fs.FileMode) error              // os.MkdirAll
	MkdirTemp  func(dir, pattern string) (string, error)              // os.MkdirTemp
	Stat       func(path string) (fs.FileInfo, error)                 // os.Stat
	ReadDir    func(path string) ([]fs.DirEntry, error)               // os.ReadDir
	Remove     func(path string) error                                // os.Remove
	RemoveAll  func(path string) error                                // os.RemoveAll
	Rename     func(oldPath, newPath string) error                    // os.Rename
	Chmod      func(path string, perm fs.FileMode) error              // os.Chmod
	WalkDir    func(root string, fn fs.WalkDirFunc) error             // filepath.WalkDir
	CreateTemp func(dir, pattern string) (string, error)              // closure: os.CreateTemp + Close; returns the unique name

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
   `targ check-full`. If a single-call closure or the literal itself ever trips the checker,
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
- **P-4:** `CreateTemp(dir, pattern)` returns the NAME of a created-then-closed unique temp file;
  the internal atomic dance is CreateTemp→Chmod→WriteFile→Rename (+Remove on any failure).
  Same-directory rename atomicity — the ADR-0013 primitive — is unchanged.
- **D-1 (amends R6 and the wiring-core cmd-wiring sequencing flag):** `Deps.Embed` is wired INSIDE
  NewDeps at T2 (`embed.NewLazyEmbedder(CacheDirFromHome(homeOrEmpty(deps), embed.BundledModelID,
  prims.Getenv))`, guarded on non-nil Getenv); cmd carries no embed wiring line. T14's 3-arg
  constructor change edits that internal line and adds backend/cache capability fields to
  Primitives whose cmd-side values are single-call method bodies on EMPTY structs (empty structs
  and single-call methods pass the checker; any stateful session/cache/temp-dir orchestration
  stays internal, parameterized over primitives — a cmd struct WITH fields is a gate failure).
- **C-1 (T16/T17):** the commander uses the injected-sentinel form — Primitives gains a raw run
  capability plus a `NotFoundErr error` field (cmd wires `exec.ErrNotFound`); the run-and-collect
  logic + `errors.Is(err, notFoundErr)` → `update.ErrCommandNotFound` translation live internal.
  Zero cmd logic beyond single-call wrapping; T17's brief reviser owns the exact field shapes.
- **X-1 (T3 forward-guidance):** the EdgeFS `WriteFileExcl` method (learn-family flag) is
  implemented INTERNALLY over a new semantic exclusive-create primitive (e.g. `OpenExcl(path,
  perm) (uintptr, error)` = syscall.Open O_CREAT|O_EXCL|O_WRONLY plus a write/close over existing
  fd primitives, or an equivalent single-call shape) — the composed error must keep satisfying
  `errors.Is(err, fs.ErrExist)`. T3's brief reviser owns the exact shape under this doctrine.
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
  os_update.go, no cmd Deps literal — cmd contributes the `RunCommand` single-call closure +
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
- DESIGN FLAG (coordination, targets/foundation cluster): this draft assumes the foundation task lands first (`internal/cli/deps.go` with `Deps`/`EdgeFS`/`FileLocker`, `cmd/engram` `osFS`+`flockLocker`, `Targets(d Deps)` threading `d` into `learnUpdateTargets`, and `executeForTest` in targets_test.go re-wired to a real-FS test Deps). The shared test doubles `osEdgeFSForTest`/`flockLockerForTest`/`realFSDepsForTest` created in L1 (testhelpers_test.go) are intended for reuse by targets_test and the other family clusters. Shared compose helpers (`statDirFromFS`, `listMDFromFS`, `logWarningTo`, `vaultLockFromLocker`, `writeNoteAtomicFromFS`) are declared ONCE in `internal/cli/deps_compose.go` by this task — amend/resituate/vocab/activate clusters must consume, not re-declare.
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
| All eight adapter implementations relocated to cmd/engram (package main) with integration tests; internal keeps interfaces + logic only | Read per the revised composition doctrine (user correction 2026-07-19, vault note 303): raw PRIMITIVES relocate to cmd (func references + single-call closures in the Primitives literal); adapter COMPOSITION lives in internal/cli, integration-tested in internal `_test` files — T1-rework (signal, debuglog sink, EdgeFS/flock), T14 (hugot + cache), T17 (commander); T10 reduces to consumer migration; absorption-into-EdgeFS flags cover vault_fs/learn-FS (T5/T3); `targ check-thin-api` enforces the cmd side |
| debuglog is pure (writer + clock injected); both direct env reads eliminated; env/clock threaded from cmd/engram; targ.Main called from cmd | T2 (debuglog, ENGRAM_DEBUG_LOG, targ.Main), T8 (ENGRAM_TRANSCRIPT_DIR) |
| internal/update no longer imports os/exec (sentinel translated in the cmd adapter) | T16, T17 — read per doctrine flag C-1: cmd contributes only a raw run primitive plus a `NotFoundErr error` Primitives field wiring `exec.ErrNotFound`; the `errors.Is` translation to `update.ErrCommandNotFound` lives internal (zero cmd logic) |
| Coverage stance for cmd/engram revisited deliberately | T-final-1 Step 5.5 |
| targ check-full green; truncation setting (max-issues-per-linter) decided deliberately | every task's final gate (check-full green AND `targ check-thin-api` PASS, per Global Constraints); T-final-1 Step 3 |
| FIXME at internal/cli/main.go removed — last, after everything above is green | T2 relocates the marker (R8); T-final-2 deletes it |

