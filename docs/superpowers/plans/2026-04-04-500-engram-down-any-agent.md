# engram-down: Work From Any Agent (Issue #500)

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:writing-skills` for ALL edits to SKILL.md (TDD: baseline test → edit → pressure test). Use `superpowers:subagent-driven-development` (recommended) or `superpowers:executing-plans` to implement task-by-task.

**Goal:** Fix `skills/engram-down/SKILL.md` so the shutdown sequence works correctly when invoked from any agent, not just lead.

**Architecture:** Three targeted edits to `skills/engram-down/SKILL.md`: (1) replace hardcoded `from = "lead"` with the calling agent's actual name, (2) replace the tracked-pane-ID model (Steps 2+3) with tmux session discovery — post shutdown to all, wait 10s, then kill all Claude panes in the current session except the caller's own pane, (3) guard background task draining so non-lead agents don't try to drain task IDs they never set. No Go code changes. No new files.

**Tech Stack:** Skill Markdown, bash, tmux.

---

## File Structure

**Modified:**
- `skills/engram-down/SKILL.md` — three targeted section edits

---

## Task 1: Replace hardcoded `from = "lead"` with dynamic agent name

**Files:**
- Modify: `skills/engram-down/SKILL.md` (Step 1, lines 14–26)

### Background

Step 1 posts the broadcast `shutdown` message with `from = "lead"` hardcoded. When any non-lead agent invokes this skill, their shutdown broadcast will appear to come from `lead`, which is incorrect and could confuse other agents watching the chat.

### Baseline Test (RED)

Read `skills/engram-down/SKILL.md` lines 14–26. Verify:
- The `[[message]]` block contains `from = "lead"` (literal string, not a placeholder)
- There is no instruction telling the executor to substitute their own name

### Edit

In `skills/engram-down/SKILL.md`, replace the entire Step 1 section with:

```markdown
### Step 1: Broadcast shutdown

Before posting, identify your agent name. This is the name you used in your `ready` message when you joined chat (e.g., `lead`, `engram-agent`, `executor-1`). Substitute it for `<your-agent-name>` below.

Post `shutdown` to chat addressed to `"all"` so every agent knows the session is ending:

```toml
[[message]]
from = "<your-agent-name>"
to = "all"
thread = "lifecycle"
type = "shutdown"
ts = "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
text = "Session complete. Shutting down."
```
```

### Pressure Test (GREEN)

Read `skills/engram-down/SKILL.md` Step 1. Verify:
- `from = "<your-agent-name>"` (placeholder, not literal `"lead"`)
- The step includes an instruction to substitute the actual agent name
- `ts` uses a shell substitution expression rather than a literal `<now>` placeholder

- [ ] Read `skills/engram-down/SKILL.md` lines 14–30 to confirm current state (baseline)
- [ ] Verify `from = "lead"` is present and no substitution instruction exists (RED)
- [ ] Apply the edit above
- [ ] Read the edited section to confirm `from = "<your-agent-name>"` and the substitution instruction are present (GREEN)
- [ ] Commit: `git commit -m "fix(engram-down): replace hardcoded from=lead with dynamic agent name (#500)"`

---

## Task 2: Replace tracked-pane-ID kill model with session-scoped discovery

**Files:**
- Modify: `skills/engram-down/SKILL.md` (Steps 2–3, lines 28–50)

### Background

Steps 2 and 3 require the calling agent to have a registry of task agent pane IDs (step 2) and the engram-agent pane ID (step 3). This state is only available to the lead. Non-lead agents have no pane registry.

The fix: drop the two separate tracked-kill steps in favour of a single discovery step that (a) waits 10 seconds for all agents to finish in-flight work after the broadcast shutdown, then (b) uses `tmux list-panes -s` to find all Claude panes in the current tmux session, excludes the caller's own pane, and kills the rest.

The 10s wait is the longer of the two existing waits (step 3 already waited 10s for engram-agent). Task agents finish in well under 10s; engram-agent also finishes within 10s. The chat protocol delivers the broadcast `shutdown` simultaneously to all agents — they all begin wrapping up at the same moment. The 10s wait is empirically sufficient for all agents to complete in-flight work; it does not enforce sequential ordering by protocol. In practice, task agents (simple planners/executors) complete faster than engram-agent (which processes final `learned` messages), so the ordering holds — but this is an empirical observation, not a protocol guarantee.

Step 4 (kill chat tail pane) already uses `-a` and is unchanged. This task replaces only Steps 2 and 3.

### Baseline Test (RED)

Read `skills/engram-down/SKILL.md` lines 28–50. Verify:
- Step 2 says "For each tracked task agent" and references "tracked pane ID"
- Step 3 references "engram-agent-pane-id" as a tracked variable

### Edit

In `skills/engram-down/SKILL.md`, replace Steps 2 and 3 (lines 28–50) with the following single step. Renumber former Steps 4–7 to Steps 3–6 accordingly.

```markdown
### Step 2: Wait, then kill all agent panes

Wait 10 seconds for all agents to complete in-flight work and post their final messages:

```bash
sleep 10
```

Then kill all Claude panes in this tmux **session** (not all sessions), excluding your own pane:

```bash
OWN_PANE=$(tmux display-message -p '#{pane_id}')
tmux list-panes -s -F '#{pane_id} #{pane_current_command}' \
  | grep -i claude \
  | grep -v "^$OWN_PANE " \
  | awk '{print $1}' \
  | xargs -I{} tmux kill-pane -t {}
tmux select-layout main-vertical
```

**Why `-i` (case-insensitive grep):** `pane_current_command` shows the foreground binary name. Claude Code's CLI binary is named `claude` (lowercase), so `grep -i claude` matches it regardless of case variants. Note: if Claude Code ever runs inside a wrapper process (e.g., `node` or `sh`), this filter will not match it — that is a known limitation. In standard engram usage, agents run `claude` directly.

**Why `-s` (not `-a`):** `-s` lists panes in the current tmux session only, so you don't accidentally kill Claude panes from unrelated projects open in other sessions. The chat tail pane (Step 3) uses `-a` because it is identified by command (`tail`), not by agent identity.

**Why exclude own pane:** The caller's pane must stay alive to post the session summary (Step 5). The user reads the summary from this pane.
```

### Pressure Test (GREEN)

Read `skills/engram-down/SKILL.md` Steps 2–3 after edit. Verify:
- Only ONE step 2 exists (no separate step 3 for engram-agent)
- Step 2 uses `tmux list-panes -s` with pane discovery
- Step 2 uses `grep -i claude` (case-insensitive) with a note about the wrapper-process limitation
- Step 2 excludes the caller's own pane via `tmux display-message -p '#{pane_id}'`
- Step 2 explains the ordering tradeoff as "empirically sufficient", not "protocol-preserved"
- Former steps 4–7 are renumbered to 3–6 (check that "Step 3" now describes the chat tail pane, "Step 4" the background tasks, etc.)

- [ ] Read `skills/engram-down/SKILL.md` lines 28–50 to confirm Steps 2+3 current state (baseline)
- [ ] Verify "tracked task agent" and "tracked pane ID" language is present (RED)
- [ ] Apply the edit: replace Steps 2+3 with new Step 2, renumber remaining steps
- [ ] Read the edited file end-to-end to confirm new Step 2 content and correct renumbering (GREEN)
- [ ] Commit: `git commit -m "fix(engram-down): replace tracked-pane-id kill with session-scoped tmux discovery (#500)"`

---

## Task 3: Guard background task draining for non-lead callers

**Files:**
- Modify: `skills/engram-down/SKILL.md` (Step 5 → Step 4 after renumbering, ~lines 70–78)

### Background

Step 5 (now Step 4 after renumbering) instructs the executor to drain `CHAT_FSWATCH_TASK_ID` and other tracked background task IDs. These IDs are only set by the lead during startup. A non-lead agent invoking engram-down will not have these IDs. If the skill instructs the agent to drain an unset ID, it either errors or silently passes — in either case, the instruction is confusing and wrong for non-lead callers.

The fix: restate the step to drain only what the calling agent actually has tracked, with explicit "skip if not set" guidance.

### Baseline Test (RED)

Read `skills/engram-down/SKILL.md` Step 4 (formerly Step 5). Verify:
- The step unconditionally references `CHAT_FSWATCH_TASK_ID` without any guard
- There is no "skip if not set" instruction

### Edit

In `skills/engram-down/SKILL.md`, replace the body of Step 4 (formerly Step 5) with:

```markdown
### Step 4: Drain background task IDs

Prevent zombie shell accumulation in Claude Code's background task queue.

Drain **only** the background task IDs you have tracked in this session. **Skip any ID you never set.** Common IDs:

- `CHAT_FSWATCH_TASK_ID` (chat file watcher — set by lead and most agents): if set, call `TaskOutput(task_id=CHAT_FSWATCH_TASK_ID, block=False)`
- `HEALTH_CHECK_TASK_ID` (lead-only): if set, call `TaskOutput(task_id=HEALTH_CHECK_TASK_ID, block=False)`
- Hold detection task IDs (lead-only): drain each with `TaskOutput(task_id=<id>, block=False)`

Non-lead agents typically only have `CHAT_FSWATCH_TASK_ID`. If you have none, skip this step.
```

### Pressure Test (GREEN)

Read `skills/engram-down/SKILL.md` Step 4 after edit. Verify:
- Drain instructions are conditional ("if set")
- The step explicitly names `CHAT_FSWATCH_TASK_ID` as the one most non-lead agents will have
- `HEALTH_CHECK_TASK_ID` and hold detection IDs are marked lead-only
- "Skip if not set" guidance is present

- [ ] Read the current Step 4 content (baseline)
- [ ] Verify unconditional `CHAT_FSWATCH_TASK_ID` reference without guard (RED)
- [ ] Apply the edit
- [ ] Read the edited Step 4 to confirm conditional drain language and "skip if not set" (GREEN)
- [ ] Commit: `git commit -m "fix(engram-down): guard background task draining for non-lead callers (#500)"`

---

## Task 4: Update Common Mistakes table

**Files:**
- Modify: `skills/engram-down/SKILL.md` (Common Mistakes section, ~lines 90–99)

### Background

The Common Mistakes table has two entries that are now wrong or misleading after the discovery-based changes:
1. `"Kill by window index or name | Always kill by tracked pane ID"` — the new model uses session-scoped discovery, not tracked pane IDs. This entry should be updated.
2. New failure modes introduced by the fix need entries:
   - Killing your own pane (forgetting to exclude it)
   - Using `-a` instead of `-s` when killing agent panes (would hit other sessions)
   - Draining task IDs you never set

### Edit

In `skills/engram-down/SKILL.md`, replace the Common Mistakes table with:

```markdown
## Common Mistakes

| Mistake | Fix |
|---------|-----|
| Kill engram-agent before task agents | Broadcast shutdown to all first, wait 10s — all agents get the message simultaneously and wrap up in order |
| Use `tmux list-panes` without `-a` for chat tail | Works only in current window — use `-a` so the tail pane is found regardless of which window is active |
| Use `tmux list-panes -a` instead of `-s` for agent panes | `-a` kills Claude panes across ALL sessions — use `-s` to scope to the current session only |
| Kill your own pane | Exclude `$(tmux display-message -p '#{pane_id}')` from the kill list — you need your pane for the summary |
| Skip draining background task IDs | Zombie shells accumulate across sessions — drain all IDs you have tracked |
| Drain task IDs you never set | Only drain IDs you actually set in this session — skip those that belong to other roles (e.g., lead-only IDs) |
| Truncate chat file | Chat file is persistent and append-only — never truncate |
| Post `from = "lead"` when not lead | Substitute your actual agent name from your `ready` message |
```

- [ ] Read `skills/engram-down/SKILL.md` Common Mistakes section (baseline)
- [ ] Verify stale "tracked pane ID" entry is present and new failure modes are absent (RED)
- [ ] Apply the edit
- [ ] Read the updated Common Mistakes table to confirm all 8 entries are correct (GREEN)
- [ ] Commit: `git commit -m "fix(engram-down): update Common Mistakes for discovery-based shutdown (#500)"`

---

## Self-Review

**Spec coverage check:**

| Issue #500 requirement | Task |
|------------------------|------|
| Not hardcoded `from = "lead"` | Task 1 |
| Works from non-lead agent (no pane registry needed) | Task 2 |
| Works from lead too (tmux -s scoped, own pane excluded) | Task 2 |
| Fall back to pane discovery if no tracked IDs | Task 2 |
| Background task draining works for non-lead | Task 3 |
| Docs/mistakes updated | Task 4 |

All requirements covered. No gaps.

**Placeholder scan:** No TBD, TODO, or incomplete sections. Code blocks are complete and runnable.

**Consistency check:**
- Step renumbering in Tasks 2–4 must be applied in order (Task 2 shifts Step 5→4, Step 6→5, Step 7→6; Tasks 3–4 reference the renumbered steps). The executor should complete Task 2 before Tasks 3–4.
- `tmux display-message -p '#{pane_id}'` appears in Task 2 (edit) and Task 4 (Common Mistakes) — consistent.
