# Deploy engram's recall-firing guidance via CLAUDE.md `@import` — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development (or executing-plans).
> Steps use `- [ ]`. Gate B/C/D markers are run by the `/please` orchestrator.

**Goal:** Make engram *own* its load-bearing recall-firing guidance as a versioned repo file and *deploy* it via
`engram update`, so a fresh install gets the always-loaded "fire `/recall` at decision moments" guidance (not
just the skills). The user's `~/.claude/CLAUDE.md` activates it with one stable line: `@~/.claude/engram/recall.md`.

**Why this shape (validated):** The guidance must be **always-loaded** (note 100: recall fires from always-loaded
context *before* any skill body loads — a skill can't trigger itself). Claude Code's `@path` import is confirmed
always-loaded-inline-equivalent ("imported files are expanded and loaded into context at launch"; supports global
`~/.claude/CLAUDE.md` home-path imports — official docs, validated via claude-code-guide 2026-07-01). So engram
ships the guidance as a **file** (its wheelhouse — like skills) and the user imports it — **not** by
programmatically editing CLAUDE.md (rejected approach A). Today the guidance lives ONLY in `~/.claude/CLAUDE.md`,
untracked + undeployed (verified: `engram update` copies binary+skills only).

**Architecture:** A new `guidance/` source dir in the repo (parallel to `skills/`, `commands/`) holds the canonical
guidance file. `internal/update` gains a `GuidanceTargetRel` per `HarnessSpec` + a `planGuidanceCopies` pass
(mirroring `planSkillCopies`) that deploys `guidance/*.md` to `<harness-root>/engram/`, gated by an opt-in
`--with-guidance` flag. A warn-if-missing check reads the harness CLAUDE.md and prints activation instructions when
the `@import` line is absent.

**Tech Stack:** Go (`internal/update/`, `internal/cli/`); the update package's injected `Filesystem`/`Env`/`Commander`
DI + property/table tests (`internal/update/update_test.go` uses rapid + gomega); `targ` for test/lint; real-binary
verify via `go install ./cmd/engram` then `engram update --with-guidance --dry-run`.

## Global Constraints
- `targ` for all Go test/lint (`targ test`, `targ check-full`) — NEVER `go test`/`go vet`. Binary: `go install ./cmd/engram`.
- DI everywhere: no `os.*` in `Run*`/planners — I/O through the injected `Filesystem`/`Env`. Follow the existing
  `planSkillCopies`/`applyOps` pattern exactly.
- nilaway/gomega guards per `.claude/rules/go.md`; line length < 120; descriptive names; `t.Parallel()` no shared state.
- **SCOPE: Claude Code only.** OpenCode (`AGENTS.md`) import support is UNVERIFIED — its `HarnessSpec.GuidanceTargetRel`
  stays empty (skip), with a code comment + a follow-up issue. Do NOT deploy guidance to OpenCode on an unverified feature.
- The guidance file is EXTRACTED VERBATIM from the current `~/.claude/CLAUDE.md` "Recall at the decision moments"
  section — this is a *move* of already-vetted content, NOT a wording edit, so writing-skills TDD is not triggered.
  (Any future *change* to the guidance wording is a separate writing-skills task — notes 137/144/145.)
- Commit trailer `AI-Used: [claude]`.

---

## Task 1: Own the guidance file in the repo

**Files:** Create `guidance/recall.md`.

- [ ] **Step 1.** Read the current `~/.claude/CLAUDE.md` and copy its entire "## Recall at the decision moments,
  not only at the start" section (through the end of that section, before the next `---`/heading) **verbatim** into
  `guidance/recall.md`. Add a 2-line header comment at the top: `<!-- engram-owned: recall-firing guidance. Deployed
  by 'engram update --with-guidance' to ~/.claude/engram/recall.md; activate via '@~/.claude/engram/recall.md' in
  CLAUDE.md. Edit via writing-skills TDD. -->`.
- [ ] **Step 2 — Verify** the file is the complete guidance block (the cues: before-declaring-done, after-failure,
  before-new-approach; the glance/deep escalation). No RED (content extraction, not code).

## Task 2: Add a guidance deploy pass to `internal/update` (Claude Code only)

**Files:** Modify `internal/update/update.go` (HarnessSpec + harness specs list + `planGuidanceCopies` + `Run`),
`internal/update/export_test.go` (export the planner for tests); Test `internal/update/update_test.go`.

**Interfaces:**
- `HarnessSpec` gains `GuidanceTargetRel string` — Claude Code = `".claude/engram"`; OpenCode = `""` (skip).
- Produces `func planGuidanceCopies(srcGuidance string, home string, harnesses []HarnessSpec, fs Filesystem)
  ([]CopyOp, error)` — for each harness with non-empty `GuidanceTargetRel`, plan a `CopyOp` per `*.md` in
  `srcGuidance` to `<home>/<GuidanceTargetRel>/<basename>`. **Mirror `planCommandCopies` (update.go:608)** —
  `ReadDir` + `mdFilesIn` (top-level `*.md` only, by basename), skip harnesses with empty `GuidanceTargetRel`, and
  **`return nil, nil` when `srcGuidance` is absent** (guidance is optional). Do NOT mirror `planSkillCopies`
  (update.go:651) — it recurses and errors via `ErrSkillsSrcMissing`, both wrong here.
- `CopyOp` gains a `GuidanceFile string` field (the basename), following the `SkillDir`/`CommandFile` discriminator
  precedent (exactly one set).

- [ ] **Step 1 — RED test.** Add `TestPlanGuidanceCopies_FilesUnderHome` (a **table test** — guidance is a fixed
  flat structure, so simpler than `TestPlanSkillCopies_FilesUnderHome_Property`'s `rapid.Check`): seed a fake
  `guidance/recall.md`, a Claude Code harness spec (GuidanceTargetRel `.claude/engram`) + an OpenCode spec
  (GuidanceTargetRel ``), assert (a) exactly one CopyOp, (b) Dst = `<home>/.claude/engram/recall.md`, (c) NO op for
  OpenCode. Also `TestPlanGuidanceCopies_MissingSrc` → **empty, nil error** (contrast `TestPlanSkillCopies_MissingSrc`
  which asserts `ErrSkillsSrcMissing`). Reference `ExportPlanGuidanceCopies` (add via the `var ExportX = x` pattern
  in `export_test.go`).
- [ ] **Step 2 — Run RED.** `targ test` → fails (undefined `planGuidanceCopies`/`ExportPlanGuidanceCopies`).
- [ ] **Step 3 — GREEN.** Implement `planGuidanceCopies`; add `GuidanceTargetRel` to `HarnessSpec` + set it on the
  two literals in **`supportedHarnesses()` (update.go:694)** (Claude Code `.claude/engram`; OpenCode `""`); add
  `CopyOp.GuidanceFile`; export the planner.
- [ ] **Step 4 — Run GREEN.** `targ test` → pass.
- [ ] **Step 5 — Gate B** (design-fit: mirrors planSkillCopies, DRY, OpenCode-skip explicit).

## Task 3: Gate deploy behind the opt-in `--with-guidance` flag + apply

**Files:** Modify `internal/update/update.go` (`Options`, `Run`, `applyOps` + `applyForHarness` + new
`applyGuidanceOps`, `HarnessReport`), `internal/cli/update.go` (`UpdateArgs` flag + `runUpdate` wiring); Test
`internal/update/update_test.go`, `internal/cli/update_test.go`.

**Interfaces:**
- `Options` gains `WithGuidance bool`. In `Run`, plan `guidanceOps` via `planGuidanceCopies` ONLY when
  `opts.WithGuidance` (else `nil`), and pass them to `applyOps`.
- **Thread `guidanceOps` through the apply path (verified against update.go):** `applyOps` (update.go:274) gains a
  `guidanceOps []CopyOp` parameter (parallel to `cmdOps`); its `HarnessReport` init (update.go:283) gains
  `GuidanceRoot: filepath.Join(home, spec.GuidanceTargetRel)`. The `Run` call site becomes
  `applyOps(harnesses, home, skillOps, cmdOps, guidanceOps, opts.DryRun)`. `applyForHarness` (update.go:233) gains a
  `guidanceOps` param and, after `applyCmdOps`, calls a new `applyGuidanceOps` that mirrors `applyCmdOps` (RemoveAll +
  `applyOne` per op — `applyOne` already MkdirAll's the dest dir). Each deployed basename appends to
  `HarnessReport.GuidanceFiles`.
- `HarnessReport` gains `GuidanceRoot string` + `GuidanceFiles []string`.
- `UpdateArgs` gains `WithGuidance bool targ:"flag,name=with-guidance,desc=also deploy engram's recall-firing guidance
  file for CLAUDE.md @import"`. `runUpdate` maps it to `Options.WithGuidance` (mirrors how `DryRun` is wired at
  cli/update.go:208; no `targets.go` change — targ reads struct tags).
- **Flag naming — `--with-guidance`, NOT the user's initial `--with-claude-file`:** the deployable unit is the
  guidance *file*, and (unlike rejected approach A) engram never edits the user's CLAUDE.md, so "claude-file" would
  mislead.
- **Opt-in decision:** deploy is opt-in (a flag) though the file itself is non-invasive (a new file, never edits
  CLAUDE.md); considered default-on (consistent with skills) but kept opt-in per the user's stated preference —
  plain `engram update` stays behaviorally unchanged and instead *warns* (Task 4).

- [ ] **Step 1 — RED test.** `TestRun_WithGuidance_DeploysToClaudeEngram`: Updater over a fake FS with a Claude Code
  harness + `guidance/recall.md`; `Run(ctx, Options{WithGuidance:true})` → report lists the guidance file at
  `~/.claude/engram/recall.md` and the FS has it. And `TestRun_WithoutGuidance_SkipsGuidance` → no guidance file
  written. Plus a cli-level `TestRunUpdate_WithGuidanceFlag` mapping the flag to Options (dry-run).
- [ ] **Step 2 — Run RED.** `targ test` → fails (no `WithGuidance`).
- [ ] **Step 3 — GREEN.** Add the field/flag/wiring + the apply path (mkdir `.claude/engram`, RemoveAll+copy).
- [ ] **Step 4 — Run GREEN.** `targ test` → pass.
- [ ] **Step 5 — Gate B.**

## Task 4: Warn-if-missing / activation UX

**Files:** Modify `internal/update/update.go` (detect the import line, add to report), `internal/cli/update.go`
(format the hint); Test both.

**Interfaces:**
- After planning, read `<home>/.claude/CLAUDE.md` via `u.FS.ReadFile`; compute `guidanceImported bool` = the file
  contains a line `@~/.claude/engram/recall.md` (tolerate `~` and the expanded `<home>/.claude/engram/recall.md`
  form; ignore matches inside fenced code blocks per the import rules; a missing CLAUDE.md → false, no error).
- Store it as a single **`Report`-level `GuidanceImported bool`** — Claude-Code-specific, so it goes on the top-level
  `Report`, NOT on `HarnessReport` (avoids Claude-Code fields leaking onto the generic per-harness struct + permanent
  `false`s for OpenCode). **"Deployed" is DERIVED, not stored** — it is `len(<Claude Code HarnessReport>.GuidanceFiles)
  > 0` (from Task 3); do NOT add a separate `GuidanceDeployed bool`.
- CLI formatting: when guidance was deployed but NOT imported → print: `guidance deployed to ~/.claude/engram/recall.md
  — add '@~/.claude/engram/recall.md' to ~/.claude/CLAUDE.md to activate it (Claude Code will ask you to approve the
  import once)`. When plain `engram update` runs (no flag) and guidance is not imported → print a one-line hint:
  `engram ships recall-firing guidance; run 'engram update --with-guidance' to deploy it`.

- [ ] **Step 1 — RED test.** `TestGuidanceImportDetection`: CLAUDE.md containing `@~/.claude/engram/recall.md` →
  imported=true; without it → false; the token inside a ```code fence``` → false (not a real import). And a
  formatter test asserting the activation hint appears iff deployed && !imported, and the plain-update hint appears
  iff !flag && !imported.
- [ ] **Step 2 — Run RED.** `targ test` → fails.
- [ ] **Step 3 — GREEN.** Implement detection + report fields + formatter branches.
- [ ] **Step 4 — Run GREEN.** `targ test` → pass.
- [ ] **Step 5 — Gate B.**

## Task 5: Verify with the real binary + `targ check-full`

- [ ] **Step 1 — `go install ./cmd/engram`.**
- [ ] **Step 2 — Dry-run then real (safe — deploys a file, never edits CLAUDE.md):**
```bash
engram update --with-guidance --dry-run   # shows the planned guidance copy + activation hint
engram update --with-guidance             # deploys ~/.claude/engram/recall.md
test -f ~/.claude/engram/recall.md && echo "guidance deployed OK"
engram update                             # plain: prints the 'run --with-guidance' hint (until imported)
```
- [ ] **Step 3 — `targ check-full`** green.
- [ ] **Step 4 — Scope check (note 150).** `git diff --stat` — confirm ONLY the expected paths changed (`guidance/`,
  `internal/update/`, `internal/cli/`, `docs/`, `README.md`, `CLAUDE.md`). Revert any out-of-scope change
  (`go.mod`/`go.sum` from a stray `go mod tidy`, repo-wide formatter runs on unrelated files) before committing — a
  green `check-full` proves validity, NOT scope.

## Task 6: Docs + close-out (Step 5/6 of /please)
- [ ] **Doc sweep (Gate C) — itemized (per Gate-A docs+ask review; verify each line at edit time, the numbers are hints):**
  - **`CLAUDE.md`** (project, repo root): add `guidance/` to the Directory Structure block (parallel to `skills/`,
    `commands/`); note `engram update --with-guidance` deploys it.
  - **`README.md`:** the `engram update` examples + the command listing — add the `--with-guidance` variant.
  - **`docs/architecture/c1-system-context.md`:** the R6 edge (~L61) + its diagram label (~L27) + the update-flow
    sequence loop/report (~L342-346) — extend "skills/commands" to include the opt-in guidance file.
  - **`docs/architecture/c2-containers.md`:** the update edge label (~L29) + the C2→S6 row (~L53) — add
    "`--with-guidance` deploys guidance to `.claude/engram/` (Claude Code only; OpenCode deferred, see the follow-up issue)".
  - **`docs/architecture/c3-components.md`:** the K9 update subgraph (~L43) + K9 catalog row (~L100) — add guidance.
  - **`docs/GLOSSARY.md`:** extend the `engram update` entry (~L362) with `--with-guidance`; add entries for the
    **guidance file** (`.claude/engram/recall.md`, the deployed recall-firing guidance) and **`@import` activation**.
  - Note whether this partially resolves `#647` (README/command-surface drift).
- [ ] **File a follow-up issue:** OpenCode guidance deploy — validate `AGENTS.md` import support, then wire
  `GuidanceTargetRel`.
- [ ] **Commit + push (Gate D).** `AI-Used: [claude]`.
- [ ] **Migration — a post-completion OFFER, not a task step that edits the user's file.** After commit, present to
  the user the exact diff to *their* `~/.claude/CLAUDE.md` (replace the inline "Recall at the decision moments"
  section with the single line `@~/.claude/engram/recall.md`) and the command `engram update --with-guidance`. Do
  NOT auto-edit their CLAUDE.md — apply only on their explicit approval.

## Self-review (writing-plans checklist)
- **Coverage:** own the file (T1); deploy pass (T2); flag+apply (T3); warn UX (T4); real-binary verify (T5); docs +
  OpenCode follow-up + migration (T6). Validation gate already passed (claude-code-guide) — recorded in Why.
  **Harness coverage is PARTIAL this iteration** — Claude Code only; OpenCode is a named follow-up issue (its
  `@import`/`AGENTS.md` support is unverified — don't ship on an unverified feature).
- **Scope:** Claude Code only (OpenCode deferred, named); guidance extracted verbatim (no wording edit → no
  writing-skills TDD); the file-deploy never edits CLAUDE.md (the rejected A).
- **DRY:** `planGuidanceCopies` mirrors **`planCommandCopies`** (flat `*.md`, nil-if-missing — NOT `planSkillCopies`,
  which recurses + errors); `CopyOp.GuidanceFile` follows the SkillDir/CommandFile discriminator precedent.
- **Type consistency:** `GuidanceTargetRel`/`GuidanceFile`/`WithGuidance`/`GuidanceImported` named consistently.
