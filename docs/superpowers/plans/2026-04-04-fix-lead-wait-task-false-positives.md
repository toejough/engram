# Fix Lead Background Wait Task False Positives (Issue #478) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix `skills/engram-tmux-lead/SKILL.md` so all background wait tasks use per-agent cursors and filter by both `type` and `from`, eliminating false positives from prior-session and same-session stale messages.

**Architecture:** Pure skill document edit — no Go code changes. Four targeted sections get updated: Section 6.3 (rules), Section 2.1 (spawn template), Section 4.2 (pipeline), and Section 1.5 (engram-agent wait). Changes establish (1) per-spawn cursor capture before every agent spawn, and (2) TOML-aware agent-name filtering in every wait loop.

**Tech Stack:** Markdown, bash examples embedded in skill documentation.

---

## Background: Root Causes

Three compounding bugs cause false positives:

**Bug A — Stale session cursor:** `CURSOR` is captured once at startup (Section 1.3). By the time planner-2 is spawned (after planner-1 and exec-1 have already finished), the session cursor is stale — planner-1's `done` message is within its range. A wait loop for planner-2 that starts from the session cursor can match planner-1's `done`.

**Bug B — No agent-name filter:** The `type = "done"` grep in Section 6.3's "RIGHT" example matches any agent's done message. A wait for `exec-1` that only checks `type = "done"` will fire when `planner-1` posts its done.

**Bug C — No canonical "wait for done" template:** Section 2.1 (spawn template) shows waiting for the claude prompt but not waiting for an agent's `done`. Without a canonical pattern, lead agents implement ad hoc loops that introduce both bugs A and B.

**Fix strategy:**
1. Add Rule 4 to Section 6.3: per-spawn cursor + both-field filter
2. Add "Wait for Agent Done" template to Section 2.1 with correct pattern
3. Add per-phase wait blocks to Section 4.2 referencing the template
4. Fix Section 1.5 to use the per-spawn cursor pattern

---

### Task 1: Fix Section 6.3 — Add Per-Spawn Cursor Rule and Fix "RIGHT" Example

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md` (lines ~464–478, Section 6.3)

The existing Rules 1–3 are correct but incomplete. The "RIGHT" example still matches any agent's `done`, not a specific agent's `done`. We need Rule 4 and a corrected code example.

- [ ] **Step 1: Verify current Section 6.3 Rules block**

```bash
grep -n "" skills/engram-tmux-lead/SKILL.md | sed -n '464,479p'
```

Expected output: the existing three-rule block ending with `CURSOR=$(wc -l < "$CHAT_FILE")`.

- [ ] **Step 2: Replace the Rules block and code example**

In `skills/engram-tmux-lead/SKILL.md`, find:

```
**Rules:**
1. **CURSOR is set at startup** (Section 1.3). Every chat read after that uses `tail -n +$((CURSOR + 1))`.
2. **Update CURSOR after every read:** `CURSOR=$(wc -l < "$CHAT_FILE")` — do this after processing each batch of new lines.
3. **Read first, summarize accurately when relaying to user.** Extract the actual `text` field from new lines; then provide an intelligent summary to the user. Never fabricate, predict, or invent what the agent said.

```bash
# WRONG — matches old messages, causes false positives:
grep -q 'type = "done"' "$CHAT_FILE"
grep 'from = "researcher-1"' "$CHAT_FILE" | tail -1

# RIGHT — only new messages since the lead joined:
tail -n +$((CURSOR + 1)) "$CHAT_FILE" | grep -q 'type = "done"'
NEW_LINES=$(tail -n +$((CURSOR + 1)) "$CHAT_FILE")
CURSOR=$(wc -l < "$CHAT_FILE")
```
```

Replace with:

```
**Rules:**
1. **CURSOR is set at startup** (Section 1.3). Every chat read after that uses `tail -n +$((CURSOR + 1))`.
2. **Update CURSOR after every read:** `CURSOR=$(wc -l < "$CHAT_FILE")` — do this after processing each batch of new lines.
3. **Read first, summarize accurately when relaying to user.** Extract the actual `text` field from new lines; then provide an intelligent summary to the user. Never fabricate, predict, or invent what the agent said.
4. **Capture a FRESH cursor before each agent spawn.** The session cursor accumulates messages since startup. By the time you spawn exec-2, planner-1's `done` may already be within the session cursor range. Capture a new cursor immediately before spawning each agent and use it exclusively in that agent's wait loop. See Section 2.1 for the canonical pattern.
5. **Filter by both `type` AND `from`.** A `type = "done"` grep matches any agent's done message. When waiting for a specific agent, use the awk pattern from Section 2.1 to match both fields within the same TOML message block.

```bash
# WRONG — matches old messages and any agent's done:
grep -q 'type = "done"' "$CHAT_FILE"
grep 'from = "researcher-1"' "$CHAT_FILE" | tail -1
tail -n +$((CURSOR + 1)) "$CHAT_FILE" | grep -q 'type = "done"'  # still wrong: no agent filter

# RIGHT — per-spawn cursor, both fields, awk for TOML block matching:
SPAWN_CURSOR=$(wc -l < "$CHAT_FILE")   # capture BEFORE spawning
# ... spawn the agent ...
# In background wait task (embed literal $SPAWN_CURSOR value, not the variable name):
tail -n +$((SPAWN_CURSOR + 1)) "$CHAT_FILE" | awk '
  /^\[\[message\]\]/ { from=""; msgtype="" }
  /^from = "researcher-1"/ { from=1 }
  /^type = "done"/ { msgtype=1 }
  from && msgtype { print "DONE"; exit }
'
```
```

- [ ] **Step 3: Verify the replacement looks correct**

```bash
grep -n "" skills/engram-tmux-lead/SKILL.md | sed -n '464,495p'
```

Expected: five-rule block, updated WRONG/RIGHT example with awk pattern.

- [ ] **Step 4: Commit**

```bash
git add skills/engram-tmux-lead/SKILL.md
git commit -m "docs(skill): add per-spawn cursor + agent-name filter rules to lead section 6.3

Fixes: missing Rule 4 (per-spawn cursor) and Rule 5 (both-field filter).
Updates the WRONG/RIGHT example to show the awk TOML block pattern.

AI-Used: [claude]"
```

---

### Task 2: Add "Wait for Agent Done" Template to Section 2.1

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md` (lines ~182–203, Section 2.1)

Section 2.1 currently shows: (1) spawn pane, (2) wait for claude prompt. It stops there. The lead is left to improvise for waiting for an agent's `done`. Add Step 3: Wait for agent done, immediately after the "send role prompt" block.

- [ ] **Step 1: Verify the end of the current spawn template**

```bash
grep -n "" skills/engram-tmux-lead/SKILL.md | sed -n '182,204p'
```

Expected: the role-prompt send-keys block ending at line ~188, then "Splitting rules recap" at ~190.

- [ ] **Step 2: Insert "Wait for Agent Done" block after the role-prompt block**

In `skills/engram-tmux-lead/SKILL.md`, find:

```
When the background task completes, verify it printed "READY", then send the role prompt:

```bash
tmux send-keys -t "$PANE_ID" "/use-engram-chat-as <role> named <agent-name>. Your task: <task description>. Work in this directory: <pwd>. Use relevant skills. Post intent before significant actions. Funnel ALL questions for the user through chat addressed to lead. NEVER ask the user directly -- you have no user. Post done when your assigned task is complete." Enter
sleep 1
tmux send-keys -t "$PANE_ID" Enter
```

**Splitting rules recap:**
```

Replace with:

```
When the background task completes, verify it printed "READY", then send the role prompt:

```bash
tmux send-keys -t "$PANE_ID" "/use-engram-chat-as <role> named <agent-name>. Your task: <task description>. Work in this directory: <pwd>. Use relevant skills. Post intent before significant actions. Funnel ALL questions for the user through chat addressed to lead. NEVER ask the user directly -- you have no user. Post done when your assigned task is complete." Enter
sleep 1
tmux send-keys -t "$PANE_ID" Enter
```

**Step 3: Wait for agent done.**

Run a **background** Bash command (`run_in_background: true`) to detect the agent's `done` message.

**CRITICAL — per-spawn cursor:** Capture the cursor value **before sending the role prompt**, not at session startup. By the time you spawn exec-2, planner-1's `done` is already in the file. Reusing the session cursor gives a false positive.

```bash
# Run this BEFORE sending the role prompt (foreground, not background):
wc -l < "$CHAT_FILE"
```

Note the output as `<AGENT_NAME>_START` (e.g., `EXEC1_START=412`). Then run the wait task as background:

```bash
# Background wait — embed <AGENT_NAME>_START as a literal number, NOT a shell variable.
# Replace AGENT_NAME_START_LINE with the literal value you noted above (e.g., 412).
# Replace exec-1 with the actual agent name.
AGENT_START=AGENT_NAME_START_LINE
CHAT_FILE="$HOME/.local/share/engram/chat/$(basename "$(git rev-parse --show-toplevel 2>/dev/null || realpath "$PWD")" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9]/-/g').toml"
for i in $(seq 1 30); do
  RESULT=$(tail -n +"$((AGENT_START + 1))" "$CHAT_FILE" | awk '
    /^\[\[message\]\]/ { from=""; msgtype="" }
    /^from = "exec-1"/ { from=1 }
    /^type = "done"/ { msgtype=1 }
    from && msgtype { print "DONE"; exit }
  ')
  if [ "$RESULT" = "DONE" ]; then echo "AGENT DONE"; break; fi
  sleep 2
done
```

When the background task completes:
- If it printed "AGENT DONE": process the agent's result. Read the `done` message text from new lines (cursor-based), then update your session cursor: `CURSOR=$(wc -l < "$CHAT_FILE")`.
- If it did NOT print "AGENT DONE" after 30 iterations (60 seconds): the agent may be stuck. Check via `tmux capture-pane -t "$PANE_ID" -p -S -20`. Transition to SILENT per Section 3.2.

**Splitting rules recap:**
```

- [ ] **Step 3: Verify the new template block looks correct**

```bash
grep -n "" skills/engram-tmux-lead/SKILL.md | sed -n '182,240p'
```

Expected: role-prompt block, then "Step 3: Wait for agent done", then "Splitting rules recap".

- [ ] **Step 4: Commit**

```bash
git add skills/engram-tmux-lead/SKILL.md
git commit -m "docs(skill): add canonical wait-for-agent-done template to lead section 2.1

Adds Step 3 to the spawn template: per-spawn cursor capture + awk-based TOML
block matching for both type=done AND from=<agent-name>. Prevents false positives
from same-session messages and prior-session carryover.

AI-Used: [claude]"
```

---

### Task 3: Fix Section 4.2 — Add Explicit Wait-for-Done at Phase Boundaries

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md` (lines ~364–386, Section 4.2)

Section 4.2 says "Executor posts `done` -> Phase 3" but gives no code. The lead must know to use the Section 2.1 template at each phase boundary, and to capture a fresh cursor before each spawn.

- [ ] **Step 1: Verify current Section 4.2**

```bash
grep -n "" skills/engram-tmux-lead/SKILL.md | sed -n '364,388p'
```

Expected: three-phase bullet list with no code blocks.

- [ ] **Step 2: Replace Section 4.2 content to add explicit wait-for-done guidance**

In `skills/engram-tmux-lead/SKILL.md`, find:

```
**Phase 1: PLAN**
1. Spawn `planner-<N>` with issue context
2. Planner reads code, analyzes, posts plan as `info` message
3. Lead presents plan to user for approval
4. User approves (or modifies) -> Phase 2

**Phase 2: EXECUTE**
1. Spawn `exec-<N>` with approved plan
2. Executor implements, posting intent before each significant step
3. engram-agent watches intents for memory matches
4. Executor posts `done` -> Phase 3

**Phase 3: REVIEW**
1. Spawn `reviewer-<N>` with original plan + `git diff`
2. Reviewer inspects, posts `wait` for issues or `done` for approval
3. If issues: relay to user, may re-enter Phase 2
4. If approved: report to user, clean up agents

Do NOT spawn all three simultaneously. Each phase starts after the previous completes.
```

Replace with:

```
**Phase 1: PLAN**
1. Capture per-spawn cursor (foreground bash): `wc -l < "$CHAT_FILE"` → note as `PLAN_START`
2. Spawn `planner-<N>` with issue context (per Section 2.1 Steps 1–2)
3. Send role prompt (Section 2.1)
4. Run background wait task (Section 2.1 Step 3) — embed `PLAN_START` as literal, filter `from = "planner-<N>"` and `type = "done"`
5. When planner done: read plan text from new lines (cursor-based), present to user
6. Update session cursor: `CURSOR=$(wc -l < "$CHAT_FILE")`
7. User approves (or modifies) -> Phase 2

**Phase 2: EXECUTE**
1. Capture per-spawn cursor (foreground bash): `wc -l < "$CHAT_FILE"` → note as `EXEC_START`
2. Spawn `exec-<N>` with approved plan (per Section 2.1 Steps 1–2)
3. Send role prompt with the approved plan text (Section 2.1)
4. Run background wait task (Section 2.1 Step 3) — embed `EXEC_START` as literal, filter `from = "exec-<N>"` and `type = "done"`
5. When executor done: read result summary from new lines (cursor-based)
6. Update session cursor: `CURSOR=$(wc -l < "$CHAT_FILE")`
7. -> Phase 3

**Phase 3: REVIEW**
1. Capture per-spawn cursor (foreground bash): `wc -l < "$CHAT_FILE"` → note as `REVIEW_START`
2. Spawn `reviewer-<N>` with original plan + `git diff` output (per Section 2.1 Steps 1–2)
3. Send role prompt (Section 2.1)
4. Run background wait task (Section 2.1 Step 3) — embed `REVIEW_START` as literal, filter `from = "reviewer-<N>"` and **either** `type = "wait"` (issues found) or `type = "done"` (approved)
5. When reviewer done:
   - `wait`: relay issues to user. Decide: fix (re-enter Phase 2) or accept as-is.
   - `done`: report to user, clean up agents
6. Update session cursor: `CURSOR=$(wc -l < "$CHAT_FILE")`

**Per-spawn cursor is mandatory at every phase boundary.** See Section 6.3 Rule 4. Reusing the session `CURSOR` from a prior phase will match the previous agent's `done` as a false positive.

Do NOT spawn all three simultaneously. Each phase starts after the previous completes.
```

- [ ] **Step 3: Verify the replacement**

```bash
grep -n "" skills/engram-tmux-lead/SKILL.md | sed -n '364,410p'
```

Expected: three phases with numbered steps including cursor capture and wait-task references.

- [ ] **Step 4: Commit**

```bash
git add skills/engram-tmux-lead/SKILL.md
git commit -m "docs(skill): add per-phase cursor capture and wait-for-done guidance to lead section 4.2

Each phase boundary now explicitly captures a fresh cursor and references the
Section 2.1 wait-for-done template with agent-name filtering. Prevents planner
done from triggering exec wait, and exec done from triggering review wait.

AI-Used: [claude]"
```

---

### Task 4: Fix Section 1.5 — Per-Spawn Cursor for engram-agent Wait

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md` (lines ~122–135, Section 1.5)

Section 1.5's wait loop currently uses `$CURSOR` (the session cursor from Section 1.3). While this works correctly for engram-agent (startup cursor is usually fine since engram-agent is the first thing spawned), it should be made consistent with the pattern established in Task 2: capture a named per-spawn cursor immediately before spawning, embed it as a literal in the background task. This also fixes the issue if anything is written to the chat file between 1.3 and 1.5.

- [ ] **Step 1: Verify current Section 1.5**

```bash
grep -n "" skills/engram-tmux-lead/SKILL.md | sed -n '122,141p'
```

Expected: "Run a background Bash command..." with the loop using `$CURSOR`.

- [ ] **Step 2: Replace Section 1.5 wait task**

In `skills/engram-tmux-lead/SKILL.md`, find:

```
Run a **background** Bash command (`run_in_background: true`) to check for the engram-agent's first chat message:

```bash
# Use cursor — only check NEW messages posted after the lead joined.
# Never grep the full file: it matches old engram-agent messages from prior sessions.
for i in $(seq 1 15); do
  if tail -n +$((CURSOR + 1)) "$CHAT_FILE" 2>/dev/null | grep -q 'from = "engram-agent"'; then
    echo "ENGRAM-AGENT FOUND"; break
  fi
  sleep 2
done
```
```

Replace with:

```
First, capture the cursor **before** spawning engram-agent (foreground bash):

```bash
wc -l < "$CHAT_FILE"
```

Note the output as `ENGRAM_START`. Then run a **background** Bash command (`run_in_background: true`) to check for the engram-agent's first chat message. Embed `ENGRAM_START` as a literal number:

```bash
# ENGRAM_START_LINE is the literal value noted above (e.g., 87). NOT a variable reference.
# Background tasks run in a fresh shell — shell variables from prior bash calls are unavailable.
ENGRAM_START=ENGRAM_START_LINE
for i in $(seq 1 15); do
  if tail -n +"$((ENGRAM_START + 1))" "$CHAT_FILE" 2>/dev/null | grep -q 'from = "engram-agent"'; then
    echo "ENGRAM-AGENT FOUND"; break
  fi
  sleep 2
done
```
```

- [ ] **Step 3: Verify the replacement**

```bash
grep -n "" skills/engram-tmux-lead/SKILL.md | sed -n '122,148p'
```

Expected: foreground cursor-capture step, then background wait task with `ENGRAM_START` variable and literal-embed comment.

- [ ] **Step 4: Final consistency check — scan for any remaining full-file greps**

```bash
grep -n 'grep.*CHAT_FILE\|grep.*chat.*toml' skills/engram-tmux-lead/SKILL.md | grep -v 'tail -n'
```

Expected: no output, or only lines that are in the "WRONG" example block (which is intentional). Any hit outside the WRONG example block is a remaining bug — fix it before committing.

- [ ] **Step 5: Commit**

```bash
git add skills/engram-tmux-lead/SKILL.md
git commit -m "docs(skill): fix engram-agent wait in section 1.5 to use per-spawn cursor

Replaces session-cursor reference with a foreground cursor-capture step before
spawning, then embeds the literal value in the background wait task. Consistent
with the canonical pattern in Section 2.1.

AI-Used: [claude]"
```

---

## Self-Review

**Spec coverage check:**
- Bug A (stale cursor): addressed in Tasks 2, 3, 4 (per-spawn cursor at every spawn point)
- Bug B (no agent-name filter): addressed in Tasks 1, 2, 3 (awk pattern + Rule 5)
- Bug C (no canonical template): addressed in Task 2 (Section 2.1 Step 3)

**Placeholder scan:** All code blocks contain concrete examples. No TBDs.

**Type/name consistency:**
- `ENGRAM_START` used in Task 4 is consistent with `SPAWN_CURSOR` / `EXEC_START` / `PLAN_START` naming convention in Tasks 2–3 (each agent gets its own named cursor).
- `awk` pattern in Task 1 uses `exec-1` as example, Task 2 uses `exec-1` as placeholder — both labeled as "replace with actual agent name".

**Potential issue:** The awk pattern in Task 2's background wait template embeds the literal `AGENT_NAME_START_LINE` as a placeholder — the executor must replace this with the actual number noted from the foreground `wc -l` step. This is clearly labeled in comments.
