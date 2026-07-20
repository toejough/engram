# DISPATCH HEADER (orchestrator)

- Worktree: `/Users/joe/repos/personal/engram/.claude/worktrees/700-internal-purity` (branch `worktree-700-internal-purity`). Work ONLY here — never cd to the main checkout.
- BASE-T7: 742d80ed (T15 complete; any docs-only ledger commit atop it is fine). Constraints mirror: `.superpowers/sdd/constraints-and-resolutions.md` — READ IT FIRST; supersession map governs.
- ACCUMULATED DISPATCH NOTES (binding):
  - This brief was AMENDED post-T12 (Gate B-mandated): the Files list and step-1 gate below already reflect the three sanctioned legacy tests in vault_fs_test.go and the listDirBySuffix grep-gated deletion — the amended text governs.
  - **T4 lesson:** before deleting ANY symbol, `rg` it across `internal/` and `cmd/` INCLUDING `_test.go` files — a missed test consumer is a compile-forced deviation to handle and report, not a STOP.
  - Every step-1 gate condition is a hard STOP: an unexpected hit means an upstream task didn't land as ledgered — report, don't improvise.
  - **reorder-decls HAZARD:** `targ reorder-decls` is UNSCOPED — rewrites the 2 protected dev/eval please_step3_probe fixtures; if run, `git restore` those two paths explicitly afterward.
  - gates run FOREGROUND (no background-run-and-yield); stage EXPLICIT paths only (never `git add -A`/`-u`)
  - check-full residual set (NOT yours to fix): e2e-under-load coverage flake (re-run check-coverage-for-fail standalone to confirm) + the 2 dev/eval reorder fixtures; lint-full must be 0
- House rules: gomega + nilaway guards on touched tests; <120 char lines.
- REPORT: write `.superpowers/sdd/briefs/T7-report.md` BEFORE your final message — status, commit SHA(s), verbatim gate outcomes, every deviation with rationale, concerns. Final message: STATUS line, SHAs, one-paragraph summary, concerns.

---

### Task T7 (Q3): Purge legacy `osVaultFS` (grep-gated; runs after T12/T15 per R4)

Sequencing: after the amend/learn/qa/resituate/embed/vocab clusters migrate to `newVaultFS(d.FS)` and T12 migrates the vocab tests off `ExportNewOsVaultFS` (R12). Runs after T15 and T12 (R4) — all its preconditions precede it; T13/T16/T17 follow it in R4.

**Files:**
- Modify: `internal/cli/vault_fs.go` (delete `osVaultFS` + methods + `listDirBySuffix` [legacy-only helper — grep-gate first: its sole consumer must be `osVaultFS.ListMD`] + `"os"` import), `internal/cli/export_test.go` (delete `ExportNewOsVaultFS`, lines currently 572-578)
- Modify (T12+T15-fallout amendment, Gate B-mandated): `internal/cli/vault_fs_test.go` — delete the FOUR legacy-adapter tests that die with their subject: `TestOsVaultFS_ListMD_*` ×2 (from f52df6de, the listDirBySuffix branch coverage), `TestOsVaultFS_ReadFile_MissingPathError` (added by T12), and `TestOsVaultFS_RoundTrip_ListMDAndReadFile` (added by T15, coverage-gate-forced when embed migrated off the adapter). Delete tests + adapter + shim in the SAME commit.
- Delete: none

**Interfaces:**
- Consumes: nothing new. Produces: a pure vault_fs.go (zero I/O-capable imports).

**Steps:**
1. [ ] Gate (T12+T15-fallout amendment — expected-hit set widened): `grep -rn "osVaultFS\|ExportNewOsVaultFS" internal/cli --include='*.go'` — expected: hits ONLY in (a) vault_fs.go (definition), (b) export_test.go (shim), (c) vault_fs_test.go (the FOUR sanctioned legacy tests named in Files, all deleted by this task), and (d) the deps_compose.go:97 doc-comment mention (comment-only — reword it in this task: drop the osVaultFS name, describe the semantics). T15 landed, so any embed.go hit means the tree doesn't match the ledger → STOP. Any hit outside (a)-(d) → STOP; that cluster has not migrated (the `ExportNewOsVaultFS` pattern is load-bearing — R12: the lowercase-only `osVaultFS` grep cannot see the capital-O shim call sites, so without it this task's deletion is a silent compile break); do not proceed.
1.5. [ ] Delete the four `TestOsVaultFS_*` tests from vault_fs_test.go (they exist solely to cover the adapter you are deleting; removing them FIRST avoids a phantom coverage drop mid-task).
2. [ ] Delete from vault_fs.go: the `osVaultFS` type, its `ListMD`/`ReadFile` methods, and the `"os"` import (all other imports stay: errors, fmt, io/fs, path/filepath, strings). Then gate-check `listDirBySuffix`: `rg -n "listDirBySuffix" internal/cli --type go` — if its only remaining reference is its own definition, delete it too (legacy-only helper; its branch tests died in step 1.5). Any live consumer → keep it and report.
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

