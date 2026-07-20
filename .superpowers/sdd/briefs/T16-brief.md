# DISPATCH HEADER (orchestrator)

- Worktree: `/Users/joe/repos/personal/engram/.claude/worktrees/700-internal-purity` (branch `worktree-700-internal-purity`). Work ONLY here — never cd to the main checkout.
- BASE-T16: <SET AT DISPATCH — after T13 ACK; verify `git log --oneline -1` matches>. Constraints mirror: `.superpowers/sdd/constraints-and-resolutions.md` — READ IT FIRST; supersession map governs.
- ACCUMULATED DISPATCH NOTES (binding):
  - **T4 lesson:** before swapping/deleting ANY symbol or error value, `rg` it across `internal/` and `cmd/` INCLUDING `_test.go` files — a missed test consumer is a compile-forced deviation to handle and report, not a STOP.
  - ALL cited line numbers are pristine-tree — locate by text; symbol gates govern.
  - **Warning-routing class (T12):** if any assertion touches stderr routing, pin exact text.
  - **reorder-decls HAZARD:** `targ reorder-decls` is UNSCOPED — rewrites the 2 protected dev/eval please_step3_probe fixtures; if run, `git restore` those two paths explicitly afterward.
  - gates run FOREGROUND (no background-run-and-yield); stage EXPLICIT paths only (never `git add -A`/`-u`)
  - check-full residual set (NOT yours to fix): e2e-under-load coverage flake (re-run check-coverage-for-fail standalone to confirm) + the 2 dev/eval reorder fixtures; lint-full must be 0
- House rules: `t.Parallel()` on every test; gomega + nilaway guards; named constants; <120 char lines.
- REPORT: write `.superpowers/sdd/briefs/T16-report.md` BEFORE your final message — status, commit SHA(s), verbatim gate outcomes, every deviation with rationale, concerns. Final message: STATUS line, SHAs, one-paragraph summary, concerns.

---

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

