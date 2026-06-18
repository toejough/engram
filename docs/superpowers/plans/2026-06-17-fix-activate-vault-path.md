# Fix: `engram activate` doesn't resolve `--note` paths against the vault root

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans. Steps use checkbox (`- [ ]`) syntax.

**Goal:** `engram activate --note <vault-relative-basename>` works from any cwd (resolves the vault root like every sibling command), fixing the "missing files" / "all note paths failed" error.

**Root cause (confirmed by read + reproduce):** `RunActivate` (`internal/cli/activate.go:33-34`) calls `embed.SidecarPath(notePath)` on the raw `--note` arg; `deps.Read = os.ReadFile` reads it relative to **cwd**. `ActivateArgs` has no `Vault` field and the activate target (`targets.go:144`) skips `resolveVault` — unlike `query`/`learn`/`resituate`/`show`, which all carry a `Vault`/`VaultPath` flag resolved via `resolveVault(a.Vault, homeOrEmpty(), os.Getenv)` before the `Run*` call. The recall skill passes vault-relative basenames (the query payload lists notes by basename), so the read misses whenever cwd ≠ vault. Reproduced: `engram activate --note 24.<...>.md` from the repo dir → `reading sidecar 24.<...>.vec.json: open ...: no such file or directory` → `activate: all note paths failed`; the same note via full path succeeds.

**Architecture:** Follow the established vault-resolution pattern — add a `Vault` flag to `ActivateArgs`, resolve it in the target, and have `RunActivate` join the vault root to each (relative) note path before `SidecarPath`. Absolute `--note` paths keep working (don't double-join). (Field name `Vault` matches `resituate` — the closest sibling, which also mutates a note; `show`/`query` happen to call theirs `VaultPath`. The codebase is split on this name; either resolves identically via `resolveVault`.)

> Test surface note: `RunActivate` is called from `cli_test` via `cli.ExportRunActivate` (export_test.go), per repo convention — tests below use that, not `cli.RunActivate`. `cli.ActivateArgs`/`cli.ActivateDeps` are used directly (exported types).

**Tech Stack:** Go 1.26, `internal/cli`, gomega + `package cli_test` blackbox tests, `targ test`/`targ check-full`.

---

## Task 1: `RunActivate` resolves note paths against the vault root

**Files:** Modify `internal/cli/activate.go`; Modify `internal/cli/activate_test.go`; `export_test.go` if needed.

- [ ] **Step 1: Write the failing test** (`activate_test.go`, `package cli_test`): a relative basename + a set Vault → `bumpLastUsed` hits the VAULT-JOINED sidecar path (the in-memory Read is keyed by the joined `.vec.json`); without the join it would miss and the run would fail.

```go
func TestRunActivateResolvesRelativeNoteAgainstVault(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := "/vault"
	noteBase := "24.2026-06-12.foo.md"
	sidecar := embed.Sidecar{
		SchemaVersion: embed.SidecarSchemaVersion, EmbeddingModelID: "m@1", Dims: 1,
		SituationVector: []float32{0.1}, BodyVector: []float32{0.2}, ContentHash: "sha256:x",
	}
	joined := "/vault/24.2026-06-12.foo.vec.json"
	store := map[string][]byte{joined: embed.MarshalSidecar(sidecar)}

	deps := cli.ActivateDeps{
		Now:        func() time.Time { return time.Date(2026, 6, 17, 0, 0, 0, 0, time.UTC) },
		Read:       func(p string) ([]byte, error) { b, ok := store[p]; if !ok { return nil, os.ErrNotExist }; return b, nil },
		Write:      func(p string, b []byte) error { store[p] = b; return nil },
		LogWarning: func(string, ...any) {},
	}

	err := cli.ExportRunActivate(cli.ActivateArgs{Vault: vault, Notes: []string{noteBase}}, deps)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil { return }

	got, derr := embed.UnmarshalSidecar(store[joined])
	g.Expect(derr).NotTo(HaveOccurred())
	if derr != nil { return }
	g.Expect(got.LastUsed).To(Equal("2026-06-17"))
}

func TestRunActivateAcceptsAbsoluteNotePath(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	abs := "/elsewhere/n.md"
	sc := embed.Sidecar{SchemaVersion: embed.SidecarSchemaVersion, EmbeddingModelID: "m@1", Dims: 1,
		SituationVector: []float32{0.1}, BodyVector: []float32{0.2}, ContentHash: "sha256:x"}
	store := map[string][]byte{"/elsewhere/n.vec.json": embed.MarshalSidecar(sc)}
	deps := cli.ActivateDeps{
		Now: func() time.Time { return time.Date(2026, 6, 17, 0, 0, 0, 0, time.UTC) },
		Read: func(p string) ([]byte, error) { b, ok := store[p]; if !ok { return nil, os.ErrNotExist }; return b, nil },
		Write: func(p string, b []byte) error { store[p] = b; return nil },
		LogWarning: func(string, ...any) {},
	}
	// Vault set, but an ABSOLUTE note must NOT be joined to it.
	err := cli.ExportRunActivate(cli.ActivateArgs{Vault: "/vault", Notes: []string{abs}}, deps)
	g.Expect(err).NotTo(HaveOccurred())
}
```

- [ ] **Step 2: Run** `targ test` → FAIL (`unknown field Vault` / read misses the joined path).

- [ ] **Step 3: Implement** (`activate.go`): add `Vault string` to `ActivateArgs`; resolve each path in `RunActivate`; import `path/filepath`.

```go
// ActivateArgs holds parsed flags for `engram activate`.
type ActivateArgs struct {
	Vault string   `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root (default $XDG_DATA_HOME/engram/vault)"`
	Notes []string `targ:"flag,name=note,desc=note path to mark used (repeatable)"`
}
```

In `RunActivate`, replace `sidecarPath := embed.SidecarPath(notePath)` with:

```go
		full := notePath
		if !filepath.IsAbs(full) {
			full = filepath.Join(args.Vault, notePath)
		}

		sidecarPath := embed.SidecarPath(full)
```

(Keep the log-and-continue loop and the all-failed return otherwise unchanged. The `LogWarning` message should report `notePath` as given, so the user sees what they passed.)

- [ ] **Step 4: Run** `targ test` → PASS (both new tests + existing activate tests).
- [ ] **Step 5: Commit** `fix(cli): resolve engram activate --note paths against the vault root` + `AI-Used: [claude]`.

---

## Task 2: Wire `resolveVault` into the activate target

**Files:** Modify `internal/cli/targets.go`

- [ ] **Step 1: No new unit test for the wiring** (decisive): `resolveVault` is already unit-tested (`cli.ExportResolveVault` in export_test.go), the vault-join behavior is covered by Task 1's `RunActivate` tests, and the target closure is a one-line mirror of six existing siblings (targets.go:131/147/151/173/...). The wiring is verified end-to-end by Task 3's real-binary run. `targets_test.go` exists but tests command dispatch, not per-target vault resolution — don't add a bespoke harness for one line.

- [ ] **Step 2: Implement** — in the `engram activate` target (targets.go:~143-145), add the resolve line before the call, mirroring the `show`/`resituate` targets:

```go
		targ.Targ(func(ctx context.Context, a ActivateArgs) {
			a.Vault = resolveVault(a.Vault, homeOrEmpty(), os.Getenv)
			errHandler(RunActivate(a, newOsActivateDeps()))
		}).Name("activate").Description("Mark note(s) as recently used (bumps LastUsed in sidecar)"),
```

- [ ] **Step 3: Run** `targ test` + `targ check-full` → green.
- [ ] **Step 4: Commit** `fix(cli): resolve vault in the activate target (env/default)`.

### Task 2b: README note (step-5/Gate-C doc touch)

- [ ] In `README.md`'s command table, update the `engram activate` line to note that `--note` paths are **vault-relative** (resolved against the vault root / `ENGRAM_VAULT_PATH`), so a reader knows cwd doesn't matter. (Gate C reviews this with the other docs.)

---

## Task 3: Verify against the REAL binary (the step that was missing)

**Files:** none (manual verification — this is the boundary test whose absence caused the bug).

- [ ] **Step 1:** `targ build` (or `engram update` from the clone) to get the fixed binary.
- [ ] **Step 2:** From a cwd that is NOT the vault, run `engram activate --note <a real vault note basename>` → expect SUCCESS (exit 0, no warning), and confirm the note's sidecar `LastUsed` is today (`grep last_used <vault>/<basename>.vec.json`).
- [ ] **Step 3:** Run `engram activate --note <nonexistent>.md` → expect a single skip warning, and (if it's the only path) a non-zero-or-logged failure — and crucially confirm a genuinely-missing single sidecar among several good ones is skipped, not fatal.
- [ ] **Step 4:** Re-confirm the recall path: `engram query --phrase "..." --activate`-style flow isn't applicable (activation is skill-driven), but spot-check that the recall skill's `engram activate --note <basename>` call (basename from a real payload) now succeeds.

---

## Task 4: Exit-code check (secondary; scope decision)

- [ ] **Check** (no masking pipe): `engram activate --note /nope.md; echo "exit=$?"`. Decision table:
  - exits **non-zero** → correct, no action.
  - exits **0** despite `errActivateAllFailed` **and** the cause is local to the activate target wiring → fix it here (small, in-scope).
  - exits **0** because of shared `errHandler`/targ behavior (affects other commands too) → **do NOT fix here** (out of scope); file a follow-up issue and note it in the terminal report.
  This keeps the secondary concern from silently expanding the bugfix.

---

## Self-Review

- **Root cause → fix:** Task 1 (join vault in RunActivate) + Task 2 (resolve vault in target) directly fix the confirmed path-resolution bug; Task 3 is the real-binary boundary verification that was missing. ✓
- **Mirrors the established pattern:** `Vault` flag with `env=ENGRAM_VAULT_PATH`, `resolveVault` in the target, exactly like `resituate`/`show`/`query`. ✓
- **Absolute paths preserved** (IsAbs guard) — the full-path repro keeps working. ✓
- **No placeholders:** complete test + impl code; the one judgment (Task 4 exit-code) is bounded with a "don't expand scope" rule. ✓
- **Type consistency:** `ActivateArgs.Vault`, `RunActivate` unchanged signature (reads `args.Vault`), `resolveVault(a.Vault, homeOrEmpty(), os.Getenv)`. ✓
