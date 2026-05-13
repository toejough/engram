# Remove Vestigial Memory-Record Code Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans (inline) to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Delete the pre-vault memory-record CLI surface (`update`, `show`, `list`, `cycle`, `quick`, `reminder`) and the seven internal packages that only support it, leaving the vault-backed `recall` / `learn` / `transcript` / `build-self` surface intact.

**Architecture:** Deletion-only refactor on branch `opencode-plugin`. Six commits, each removing one subcommand and any internal package(s) that become dead as a result. After each commit, `targ check-full` and `targ test` must pass — that is the TDD-for-deletion verification.

**Tech Stack:** Go 1.25+, `targ` build system, imptest + rapid + gomega test stack. No new code is written by this plan.

---

## Pre-flight

- [ ] **Step 0a: Verify baseline is green**

  Run `targ check-full && targ test`. Expected: both pass. If they don't, stop and fix before deletion begins — we need a green baseline to detect deletion regressions.

- [ ] **Step 0b: Capture the surviving subcommand surface**

  Run `engram --help 2>&1 | head -30` and save the output mentally / in a scratch note. Expected to include: `recall`, `transcript`, `cycle`, `show`, `list`, `quick`, `update`, `reminder`, `build-self`, `learn`. After all six deletion commits, the same command should show only: `recall`, `transcript`, `build-self`, `learn`.

---

### Task 1: Delete `engram update` subcommand

`update` edits fields on a memory-record TOML in the pre-vault data dir. Zero external callers.

**Files:**
- Delete: `internal/cli/update.go`
- Delete: `internal/cli/update_test.go`
- Modify: `internal/cli/targets.go` (remove `UpdateArgs` struct + the `targ.Targ(...).Name("update")` registration)

- [ ] **Step 1: Delete the implementation files**

  ```bash
  git rm internal/cli/update.go internal/cli/update_test.go
  ```

- [ ] **Step 2: Remove the wiring in targets.go**

  Open `internal/cli/targets.go`. Delete the `UpdateArgs` type (near line 87) AND the `targ.Targ(func(ctx context.Context, a UpdateArgs) { errHandler(runUpdate(...)) }).Name("update")...` block (near line 166). Also drop any other reference to `UpdateArgs` or `runUpdate` in this file.

- [ ] **Step 3: Verify**

  ```bash
  targ check-full && targ test
  ```

  Expected: passes. If compilation errors mention `UpdateArgs` / `runUpdate` elsewhere, those references also need removal (likely in `export_test.go` or `cli_test.go`).

- [ ] **Step 4: Commit**

  ```bash
  git add -A
  git commit -m "$(cat <<'EOF'
  chore(cli): remove engram update subcommand

  The update command edited fields on a pre-vault memory-record TOML. No
  current skill or workflow invokes it. Part of issue #619 cleanup of the
  vestigial pre-vault storage layer.

  AI-Used: [claude]
  EOF
  )"
  ```

---

### Task 2: Delete `engram show` subcommand

`show` prints a single memory-record TOML. Zero external callers.

**Files:**
- Delete: `internal/cli/show.go`
- Delete: `internal/cli/show_test.go`
- Modify: `internal/cli/targets.go`

- [ ] **Step 1: Delete the implementation files**

  ```bash
  git rm internal/cli/show.go internal/cli/show_test.go
  ```

- [ ] **Step 2: Remove the wiring**

  In `internal/cli/targets.go`, delete the `ShowArgs` struct (near line 79) AND the `targ.Targ(...).Name("show")...` registration. Drop any other reference to `ShowArgs` / `runShow` in the package.

- [ ] **Step 3: Verify**

  ```bash
  targ check-full && targ test
  ```

  Expected: passes.

- [ ] **Step 4: Commit**

  ```bash
  git add -A
  git commit -m "$(cat <<'EOF'
  chore(cli): remove engram show subcommand

  Show displayed a single pre-vault memory-record TOML by slug. No current
  skill invokes it; show by basename is part of recall's vault traversal
  now. Part of issue #619.

  AI-Used: [claude]
  EOF
  )"
  ```

---

### Task 3: Delete `engram list` + retire `internal/memory` + `internal/tomlwriter` + `internal/cli/memory.go`

`list` enumerates pre-vault memory-records. It is the last surviving caller of `internal/cli/memory.go`, which is the last surviving caller of `internal/memory/` and `internal/tomlwriter/`. So this commit empties three packages.

**Files:**
- Delete: `internal/cli/list.go`, `internal/cli/list_test.go`
- Delete: `internal/cli/memory.go`, `internal/cli/memory_test.go` (the in-CLI helper for memory-record I/O — no longer needed)
- Delete: `internal/memory/` (entire directory: record.go, maintenance_test.go, record_test.go, readmodifywrite_test.go, memory_test.go)
- Delete: `internal/tomlwriter/` (entire directory: tomlwriter.go, tomlwriter_test.go)
- Modify: `internal/cli/targets.go` (remove `ListArgs` and registration)

- [ ] **Step 1: Delete the implementation files**

  ```bash
  git rm internal/cli/list.go internal/cli/list_test.go
  git rm internal/cli/memory.go internal/cli/memory_test.go
  git rm -r internal/memory internal/tomlwriter
  ```

- [ ] **Step 2: Remove the wiring**

  In `internal/cli/targets.go`, delete `ListArgs` (near line 62) and the `targ.Targ(...).Name("list")...` registration. Drop any remaining `import "engram/internal/memory"` or `import "engram/internal/tomlwriter"` in the package — there should be none left.

- [ ] **Step 3: Verify**

  ```bash
  targ check-full && targ test
  ```

  Expected: passes. If anything still references `engram/internal/memory`, find it via `grep -rln 'engram/internal/memory' --include='*.go'` and remove. (No live caller is expected; any hit is a dead reference.)

- [ ] **Step 4: Commit**

  ```bash
  git add -A
  git commit -m "$(cat <<'EOF'
  chore(cli): remove engram list and retire pre-vault memory-record packages

  Removes the engram list subcommand, the in-CLI memory helper, and the
  two pre-vault storage packages (internal/memory, internal/tomlwriter).
  After update and show were removed, list was the last surviving caller
  of these packages. The vault is the durable surface now.

  Part of issue #619.

  AI-Used: [claude]
  EOF
  )"
  ```

---

### Task 4: Delete `engram cycle` + retire `internal/cycle`, `internal/recall`, `internal/llmcmd`, `internal/externalsources`

`cycle` was the auto-extract / auto-recall pipeline driven by the removed `SessionStart` hook (Claude Code) and `experimental.chat.system.transform` (OpenCode). Both drivers are gone. The supporting packages have no other callers.

**Files:**
- Delete: `internal/cli/cycle.go`, `internal/cli/cycle_test.go`, `internal/cli/cycle_adapters_test.go`
- Delete: `internal/cycle/` (entire directory)
- Delete: `internal/recall/` (entire directory: orchestrate.go + automemory_phase.go + claudemd_phase.go + skill_phase.go + summarize.go + cost.go + their tests)
- Delete: `internal/llmcmd/` (entire directory)
- Delete: `internal/externalsources/` (entire directory)
- Modify: `internal/cli/targets.go` (remove `CycleArgs` and registration)
- Modify: `internal/cli/recall_test.go` if it imports the old `internal/recall` (current live `recall` test should NOT — verify before assuming)

- [ ] **Step 1: Sanity check the kept `recall` test**

  ```bash
  grep -l 'engram/internal/recall' internal/cli/recall_test.go
  ```

  Expected: no match. (The live `engram recall` subcommand lives in `internal/cli/cli.go` and uses `internal/transcript` + `internal/vaultgraph`, not `internal/recall`.) If it does match, that's load-bearing and the whole deletion plan needs to back up.

- [ ] **Step 2: Delete the implementation files**

  ```bash
  git rm internal/cli/cycle.go internal/cli/cycle_test.go internal/cli/cycle_adapters_test.go
  git rm -r internal/cycle internal/recall internal/llmcmd internal/externalsources
  ```

- [ ] **Step 3: Remove the wiring**

  In `internal/cli/targets.go`, delete `CycleArgs` and the `targ.Targ(...).Name("cycle")...` registration. Drop any `import "engram/internal/cycle"` / `engram/internal/recall` / `engram/internal/llmcmd` / `engram/internal/externalsources` from any remaining file. Also check `internal/cli/export_test.go` for now-stale exports.

- [ ] **Step 4: Verify**

  ```bash
  targ check-full && targ test
  ```

  Expected: passes. If `engram/internal/recall` still has any reference, grep for it and remove. The live `recall` subcommand should not be affected.

- [ ] **Step 5: Commit**

  ```bash
  git add -A
  git commit -m "$(cat <<'EOF'
  chore(cli): remove engram cycle and retire its supporting packages

  cycle was the auto-extract/auto-recall pipeline driven by the
  SessionStart hook and the opencode experimental.chat.system.transform
  hook. Both drivers were removed earlier on this branch, leaving cycle
  with zero callers.

  Deletes:
    - internal/cli/cycle.go (+ tests + adapters)
    - internal/cycle/        (the orchestration package)
    - internal/recall/       (the pre-vault recall pipeline — orchestrate,
                              automemory/claudemd/skill phase files, cost,
                              summarize)
    - internal/llmcmd/       (LLM shell command runner, only cycle used it)
    - internal/externalsources/ (only the old internal/recall used it)

  The live engram recall (vault-backed, in internal/cli/cli.go) uses
  internal/transcript + internal/vaultgraph and is not affected.

  Part of issue #619.

  AI-Used: [claude]
  EOF
  )"
  ```

---

### Task 5: Delete `engram quick` subcommand

`quick` writes a fleeting note to the vault's `Fleeting/` directory. The vault collapse removed the `Fleeting/` tier; the subcommand now writes to a directory that the live `recall` and `learn` skills explicitly say doesn't exist. No internal package becomes dead from this deletion (`internal/luhmann` is shared with `learn`).

**Files:**
- Delete: `internal/cli/quick.go`, `internal/cli/quick_test.go`
- Modify: `internal/cli/targets.go` (remove `QuickArgs` and registration)

- [ ] **Step 1: Delete the implementation files**

  ```bash
  git rm internal/cli/quick.go internal/cli/quick_test.go
  ```

- [ ] **Step 2: Remove the wiring**

  In `internal/cli/targets.go`, delete `QuickArgs` (near line 71) and the `targ.Targ(...).Name("quick")...` registration.

- [ ] **Step 3: Verify**

  ```bash
  targ check-full && targ test
  ```

  Expected: passes.

- [ ] **Step 4: Commit**

  ```bash
  git add -A
  git commit -m "$(cat <<'EOF'
  chore(cli): remove engram quick subcommand

  quick wrote a fleeting note to <vault>/Fleeting/, but the vault
  fleeting tier was removed in the tier-collapse (single-stage /learn
  writes Permanent/ and MOCs/ only). quick now targets a directory the
  learn/recall skills explicitly say doesn't exist.

  Part of issue #619.

  AI-Used: [claude]
  EOF
  )"
  ```

---

### Task 6: Delete `engram reminder` + retire `internal/reminders`

`reminder` emitted canonical text strings that the removed `SessionStart` / `UserPromptSubmit` / `PostToolUse` hooks injected into the model's context. Hooks gone, callers gone.

**Files:**
- Delete: `internal/cli/reminder.go`, `internal/cli/reminder_test.go`
- Delete: `internal/reminders/` (entire directory)
- Modify: `internal/cli/targets.go` (remove `ReminderArgs` and registration)

- [ ] **Step 1: Delete the implementation files**

  ```bash
  git rm internal/cli/reminder.go internal/cli/reminder_test.go
  git rm -r internal/reminders
  ```

- [ ] **Step 2: Remove the wiring**

  In `internal/cli/targets.go`, delete `ReminderArgs` and the `targ.Targ(...).Name("reminder")...` registration. Drop the `engram/internal/reminders` import if it lingers.

- [ ] **Step 3: Verify**

  ```bash
  targ check-full && targ test
  ```

  Expected: passes.

- [ ] **Step 4: Commit**

  ```bash
  git add -A
  git commit -m "$(cat <<'EOF'
  chore(cli): remove engram reminder and internal/reminders

  reminder emitted reminder text for the SessionStart, UserPromptSubmit,
  and PostToolUse hooks. Those hooks were removed earlier on this branch,
  leaving reminder with no callers.

  Part of issue #619.

  AI-Used: [claude]
  EOF
  )"
  ```

---

### Task 7: Top-level cleanup sweep

After the six deletion commits, the CLI surface is right but stale references may linger in CLAUDE.md, architecture docs, and the C4 directory.

**Files:**
- Modify: `CLAUDE.md` (drop any reference to removed subcommands)
- Modify: `README.md` (top-level — the recent commit `295130e8` already pruned some C4 docs, but the install section and any "commands" section may reference cycle/show/list/update/quick/reminder)
- Audit: `architecture/` and `docs/superpowers/specs/` for stale references

- [ ] **Step 1: Find stale references**

  ```bash
  grep -rln 'engram update\|engram show\|engram list\|engram cycle\|engram quick\|engram reminder' \
    README.md CLAUDE.md architecture/ docs/ opencode/ skills/ 2>/dev/null
  ```

  For each hit: open the file, decide whether the line should be deleted (stale instruction) or rewritten (still-useful context that named the removed thing).

- [ ] **Step 2: Apply changes**

  Edit each affected file. Keep diffs surgical — drop only what references removed commands, not entire sections.

- [ ] **Step 3: Verify nothing else broke**

  ```bash
  targ check-full && targ test
  ```

  Expected: passes.

- [ ] **Step 4: Commit**

  ```bash
  git add -A
  git commit -m "$(cat <<'EOF'
  docs: prune references to removed engram subcommands

  Drops mentions of engram update/show/list/cycle/quick/reminder from
  README, CLAUDE.md, and architecture docs now that those subcommands
  are gone.

  Part of issue #619.

  AI-Used: [claude]
  EOF
  )"
  ```

---

### Task 8: Manual E2E validation

- [ ] **Step 1: Rebuild**

  ```bash
  targ build
  ```

  Expected: builds clean.

- [ ] **Step 2: Confirm CLI surface**

  ```bash
  engram --help
  ```

  Expected: subcommands list shows only `recall`, `transcript`, `build-self`, `learn`. Removed: `update`, `show`, `list`, `cycle`, `quick`, `reminder`.

- [ ] **Step 3: Smoke-test each surviving subcommand**

  ```bash
  engram recall --vault /Users/joe/repos/personal/agent-memory | head -10
  engram recall --vault /Users/joe/repos/personal/agent-memory --recent --limit 5
  engram transcript --help
  engram learn --help
  engram build-self --help
  ```

  Expected: each runs without error and emits sensible output. `recall` should still anchor/recent against the live vault.

---

### Task 9: Close issue #619

- [ ] **Step 1: Comment and close**

  ```bash
  gh issue close 619 --comment "$(cat <<'EOF'
  Closed via 6 deletion commits on branch opencode-plugin (see commit
  range for exact SHAs). Surviving CLI: recall, transcript, build-self,
  learn {fact|feedback|moc}. Internal packages removed: memory, cycle,
  recall (the old pre-vault one), llmcmd, externalsources, reminders,
  tomlwriter. Unblocks #620.
  EOF
  )"
  ```

---

## Spec coverage check

Spec (the issue body) named these for evaluation: `engram update`, `engram show`, `engram list`, `engram cycle`, `engram quick`, `engram quick`. Coverage:

- `engram update` → Task 1.
- `engram show` → Task 2.
- `engram list` → Task 3.
- `engram cycle` → Task 4.
- `engram quick` → Task 5.

Additionally addressed (within scope of "vestigial code"):

- `engram reminder` + `internal/reminders` — dead after hook removal. Task 6.
- `internal/memory`, `internal/tomlwriter` — Task 3.
- `internal/cycle`, `internal/recall`, `internal/llmcmd`, `internal/externalsources` — Task 4.

Docs follow-up: Task 7.
Verification: Task 8.
Closure: Task 9.

## Notes on TDD-for-deletion

There is no "red" phase for removing code — the analog is the green baseline (`targ check-full && targ test` pass in step 0a). After each deletion task, the same command pair must still pass. A failure means the deletion was unsafe (something still depended on the removed code). The "refactor" phase here is trivial — there's nothing to refactor after a clean deletion.
