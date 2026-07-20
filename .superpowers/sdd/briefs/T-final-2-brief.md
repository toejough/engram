# DISPATCH HEADER (orchestrator)

- Worktree: `/Users/joe/repos/personal/engram/.claude/worktrees/700-internal-purity` (branch `worktree-700-internal-purity`). Work ONLY here — never cd to the main checkout.
- BASE-T-final-2: 13631846 (T-final-1 complete, enforcement ACTIVE; docs-only ledger commits atop are fine). Constraints mirror: `.superpowers/sdd/constraints-and-resolutions.md`.
- **Step-3 AMENDMENT (binding):** the plan's `git add -A` is superseded — stage the EXPLICIT path only: `git add cmd/engram/main.go`. Never `-A`/`-u`.
- The marker MUST exist before deletion (`rg -n "FIXME\(#700\)" .` → exactly one hit in cmd/engram/main.go); zero-hits-before = defect → STOP and escalate. Zero hits AFTER is the deliverable.
- gates run FOREGROUND; comment-only deletion must change no gate outcome.
- REPORT: `.superpowers/sdd/briefs/T-final-2-report.md` BEFORE your final message — status, commit SHA, verbatim gate outcomes, the before/after grep outputs, concerns. Final message: STATUS line, SHA, summary, concerns.

---

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
