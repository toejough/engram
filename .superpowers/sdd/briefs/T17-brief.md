# DISPATCH HEADER (orchestrator)

- Worktree: `/Users/joe/repos/personal/engram/.claude/worktrees/700-internal-purity` (branch `worktree-700-internal-purity`). Work ONLY here — never cd to the main checkout.
- BASE-T17: e6a6efc5 (T16 complete; docs-only ledger commits atop are fine). Constraints mirror: `.superpowers/sdd/constraints-and-resolutions.md` — READ IT FIRST; supersession map governs.
- **T13 landed before you:** the post-task sweep expectation resolves to ZERO — after your task, `rg -n '"os"|"syscall"|"os/exec"' internal/cli internal/update --type go | grep -v _test` must return NOTHING (writesafe.go died at T13, update.go's os+os/exec die HERE). This is the last impurity in internal/.
- ACCUMULATED DISPATCH NOTES (binding):
  - **T4 lesson:** before deleting/renaming ANY symbol, `rg` it across `internal/` and `cmd/` INCLUDING `_test.go` files — a missed test consumer is a compile-forced deviation to handle and report, not a STOP.
  - R13: this brief's fake-EdgeFS needs are served by `updateFakeEdgeFS` per R13 — check what already exists in the test files before declaring (T10/T11 name-collision protocol: `rg` the name, check the claiming file's package line).
  - ALL cited line numbers are pristine-tree — locate by text; symbol gates govern.
  - Surgical edits only on shared files (primitives.go, targets.go, export_test.go, cmd/engram/main.go) — never full-file replacement.
  - The C-1 RunCommand closure is the SECOND sanctioned multi-statement primitive closure (S-1 WriteFileExcl is the first): its body shape is doctrine-capped (construction + field assignments + one invocation, zero branching) and needs the signature-extension guard comment + behavior-mirror test per the doctrine — the brief's steps prescribe them; do not improvise the shape.
  - `targ check-thin-api` gates main.go staying a declaration-free single statement — if your literal edit trips it, capture the finding and STOP (escalate; never suppress).
  - **reorder-decls HAZARD:** `targ reorder-decls` is UNSCOPED — rewrites the 2 protected dev/eval please_step3_probe fixtures; if run, `git restore` those two paths explicitly afterward.
  - gates run FOREGROUND (no background-run-and-yield); stage EXPLICIT paths only (never `git add -A`/`-u`)
  - check-full residual set (NOT yours to fix): e2e-under-load coverage flake (re-run check-coverage-for-fail standalone to confirm) + the 2 dev/eval reorder fixtures; lint-full must be 0
  - POST-TASK SWEEP to report: `rg -n '"os"|"syscall"|"os/exec"' internal/cli internal/update --type go | grep -v _test` — expected after you land: ONLY writesafe.go (T13's, if T13 hasn't landed yet — check the ledger; if T13 landed, expected ZERO). This is T-final-1's readiness gate; report the exact output.
- House rules: `t.Parallel()` on every test; gomega + nilaway guards; named constants; <120 char lines.
- REPORT: write `.superpowers/sdd/briefs/T17-report.md` BEFORE your final message — status, commit SHA(s), verbatim gate outcomes, the post-task sweep result, every deviation with rationale, concerns. Final message: STATUS line, SHAs, one-paragraph summary, concerns.

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

