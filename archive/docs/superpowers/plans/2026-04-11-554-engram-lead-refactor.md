# engram-lead Refactor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rename `engram-tmux-lead` to `engram-lead`, remove tmux from agent management (use `engram dispatch` instead), keep tmux only for the optional chat-tail pane, and make `engram-up` a thin entry point into `engram-lead`.

**Architecture:** Three skill files change. `engram-lead` is the core operational skill; `engram-up` delegates to it entirely; `engram-down` gains conditional tmux handling and fixes the `@engram_name` tail-kill bug. `engram-tmux-lead` becomes a deprecation redirect. Each skill gets behavioral bash tests (TDD: RED → GREEN).

**Tech Stack:** Bash behavioral tests, SKILL.md markdown, `engram dispatch` CLI, `engram:use-engram-chat-as`, `superpowers:writing-skills` for all SKILL.md edits.

---

## File Structure

| File | Action | Responsibility |
|------|--------|----------------|
| `skills/engram-lead/SKILL.md` | **Create** | Lead orchestration; routing table, hold patterns, dispatch-based agent mgmt, optional tmux tail pane |
| `skills/engram-lead/tests/behavioral_test.sh` | **Create** | Property tests for engram-lead SKILL.md |
| `skills/engram-up/SKILL.md` | **Modify** | Thin entry point: loads engram-lead via use-engram-chat-as |
| `skills/engram-up/tests/behavioral_test.sh` | **Create** | Property tests for engram-up SKILL.md |
| `skills/engram-down/SKILL.md` | **Modify** | Conditional tmux tail-kill using `@engram_name` |
| `skills/engram-down/tests/behavioral_test.sh` | **Create** | Property tests for engram-down SKILL.md |
| `skills/engram-tmux-lead/SKILL.md` | **Modify** | Deprecation notice pointing to engram-lead |

---

## Task 1: Behavioral tests for engram-lead (RED)

**Files:**
- Create: `skills/engram-lead/tests/behavioral_test.sh`

- [ ] **Step 1: Create the test directory and file**

```bash
mkdir -p skills/engram-lead/tests
```

- [ ] **Step 2: Write the behavioral test**

Create `skills/engram-lead/tests/behavioral_test.sh`:

```bash
#!/usr/bin/env bash
# Behavioral tests for skills/engram-lead/SKILL.md
# TDD: run before creating SKILL.md (RED), then after (GREEN).
# Usage: bash behavioral_test.sh

SKILL="$(dirname "$0")/../SKILL.md"
PASS=0
FAIL=0
FAILURES=()

pass() { echo "PASS: $1"; PASS=$((PASS + 1)); }
fail() { echo "FAIL: $1"; FAIL=$((FAIL + 1)); FAILURES+=("$1"); }

assert_contains() {
  local desc="$1" pattern="$2"
  if grep -qF "$pattern" "$SKILL"; then pass "$desc"; else fail "$desc [pattern not found: '$pattern']"; fi
}

assert_not_contains() {
  local desc="$1" pattern="$2"
  if ! grep -qF "$pattern" "$SKILL"; then pass "$desc"; else fail "$desc [pattern should NOT be present: '$pattern']"; fi
}

echo "=== engram-lead SKILL.md behavioral tests ==="
echo ""

echo "--- Group 1: Identity ---"
assert_contains     "name is engram-lead"                       'name: engram-lead'
assert_not_contains "name is NOT engram-tmux-lead"              'name: engram-tmux-lead'
assert_not_contains "description does not mention 'via tmux'"   'via tmux'
assert_contains     "description triggers on /engram-lead"      '/engram-lead'
assert_contains     "description triggers on orchestrate agents" 'orchestrate agents'

echo ""
echo "--- Group 2: No tmux agent management ---"
assert_not_contains "no tmux split-window"    'tmux split-window'
assert_not_contains "no tmux send-keys"       'tmux send-keys'
assert_not_contains "no pane-count tracking"  'RIGHT_PANE_COUNT'
assert_not_contains "no pane layout rules"    'main-vertical'
assert_not_contains "no tmux kill-pane for agents" 'kill-pane'

echo ""
echo "--- Group 3: Dispatch-based agent management ---"
assert_contains "dispatch assign used for spawning"  'engram dispatch assign'
assert_contains "dispatch stop used for stopping"    'engram dispatch stop'
assert_contains "dispatch drain in shutdown"         'engram dispatch drain'
assert_contains "dispatch status for compaction"     'engram dispatch status'

echo ""
echo "--- Group 4: Optional tmux tail pane ---"
assert_contains "TMUX env var check present"          '$TMUX'
assert_contains "tail -f used for chat observer"      'tail -f'
assert_contains "tmux new-window or split for tail"   'tmux'
assert_contains "@engram_name set for tail pane"      '@engram_name'

echo ""
echo "--- Group 5: Routing table ---"
assert_contains "routing table header"         '| User Request |'
assert_contains "Implement X route"            'Implement X'
assert_contains "Fix bug X route"              'Fix bug'
assert_contains "Review route"                 'Reviewer'
assert_contains "Tackle issue route"           'Planner'
assert_contains "parallel executors row"       'Parallel executor'

echo ""
echo "--- Group 6: Hold patterns ---"
assert_contains "hold acquire command"         'engram hold acquire'
assert_contains "hold release command"         'engram hold release'
assert_contains "hold check command"           'engram hold check'
assert_contains "Pair (Review) pattern"        'Pair (Review)'
assert_contains "Handoff pattern"              'Handoff'
assert_contains "Fan-In pattern"               'Fan-In'
assert_contains "Barrier pattern"              'Barrier'

echo ""
echo "--- Group 7: Spawn prompt template ---"
assert_contains "spawn prompt template"        'active <role>'
assert_contains "spawn template task field"    'Your task:'
assert_contains "spawn template DONE field"    'Post DONE:'
assert_contains "role naming convention"       'exec-auth'

echo ""
echo "--- Group 8: Operational sections ---"
assert_contains "escalation section"           'Escalation'
assert_contains "TIMEOUT from dead worker"     'TIMEOUT'
assert_contains "context pressure section"     'Context Pressure'
assert_contains "compaction recovery section"  'Compaction Recovery'
assert_contains "shutdown section"             'Shutdown'
assert_contains "never do implementation rule" 'Never do implementation yourself'

echo ""
echo "--- Group 9: use-engram-chat-as required ---"
assert_contains "use-engram-chat-as required"  'engram:use-engram-chat-as'

echo ""
echo "=== Results ==="
echo "PASS: $PASS"
echo "FAIL: $FAIL"
if [ "${#FAILURES[@]}" -gt 0 ]; then
  echo ""
  echo "Failed tests:"
  for f in "${FAILURES[@]}"; do echo "  - $f"; done
fi
if [ "$FAIL" -gt 0 ]; then exit 1; else echo "All tests passed."; exit 0; fi
```

- [ ] **Step 3: Run tests to verify RED**

```bash
bash skills/engram-lead/tests/behavioral_test.sh
```

Expected: FAIL (skill file doesn't exist yet — `grep` on missing file returns non-zero, all assert_contains fail).

---

## Task 2: Create engram-lead/SKILL.md (GREEN)

**Files:**
- Create: `skills/engram-lead/SKILL.md`

- [ ] **Step 1: Create the skill**

Create `skills/engram-lead/SKILL.md`:

```markdown
---
name: engram-lead
description: Use when orchestrating multi-agent sessions. The user's primary agent that manages agent lifecycle, routes work, proxies user communication, and coordinates through engram chat. Triggers on /engram-lead, /engram-tmux-lead, "start multi-agent", "orchestrate agents", or when the user wants to delegate work to parallel agents.
---

# Engram Lead

The user's primary agent. **Never do implementation yourself** — delegate every task to spawned agents. Your only jobs: route work, relay user messages, surface escalations, manage agent lifecycle.

**Red flags (spawn an agent instead):**
- Running gh, git, targ, or any build/test commands
- Reading code files, writing files, or answering technical questions

Parrot every user message verbatim to chat as an `info` message BEFORE routing. **REQUIRED:** Use `engram:use-engram-chat-as` for all coordination.

## Starting a Session

1. Run dispatch — keep it running in the foreground:
   ```
   engram dispatch start [--agent engram-agent] [--agent <name>...]
   ```
   Dispatch prints the chat file path on startup. Note it.

2. **Optional (tmux only):** If `$TMUX` is set, open a chat observer tail pane:
   ```bash
   TAIL_PANE=$(tmux split-window -h -P -F '#{pane_id}' "tail -f <chat-file>")
   tmux set-option -p -t "$TAIL_PANE" @engram_name "chat-tail"
   ```
   Skip this step silently if not in tmux.

3. Post your ready message to chat.

4. Assign work:
   ```
   engram dispatch assign --agent <name> --task '<task description>'
   ```

## Routing

Use LLM judgment, not keyword matching. Post a routing intent to `engram-agent` before spawning.

| User Request | Route | Skills to Inject |
|-------------|-------|-----------------|
| "Implement X" / "Fix bug X" | Executor | superpowers:test-driven-development, feature-dev:feature-dev |
| "Why is X failing?" / investigate root cause | Researcher | none |
| "Review this PR" / "Review this code" | Reviewer | superpowers:receiving-code-review |
| "Run tests and fix failures" | Executor | superpowers:test-driven-development |
| "Tackle issue #N" | Planner → Executor → Reviewer (sequential) | per role |
| "Do A and B" (independent tasks) | Parallel executors in separate worktrees | per role |

## Spawn Prompt Template

```
active <role> named <agent-name>.
Your task: <task description>.
Work in this directory: <pwd>.
Use <skills per routing table>. Post intent before significant actions.
Post DONE: when complete with a summary of what changed.
```

Role names use task descriptors: `exec-auth`, `exec-db`, `reviewer-auth`. Reserve sequential numbers (`exec-1`) only when tasks are genuinely interchangeable. The `engram-agent` is never numbered.

## Hold Patterns

Create holds at spawn time — before the target agent can post done.

| Pattern | When to Use | Condition Arg |
|---------|------------|---------------|
| **Pair (Review)** | Reviewer must question subject after subject done | done:reviewer |
| **Handoff** | Receiver needs to ask sender questions before taking over | first-intent:receiver |
| **Fan-In** | Consumer needs all producers alive for follow-up questions | done:consumer |
| **Barrier** | All collaborative agents stay until lead signals complete | lead-release:\<tag\> |

```
engram hold acquire --holder <H> --target <T> --condition <C> [--tag <label>]
engram hold check --target <name>
engram hold release --hold-id <id>
```

## Escalation

Surface to user immediately with full context from both sides:

```
[exec-auth <-> reviewer-auth argument, unresolved after 3 rounds]
reviewer-auth: The migration has no rollback plan — unsafe to deploy.
exec-auth: Rollback is explicitly out of scope per the spec.
Decision needed: approve migration as-is, or add rollback first?
```

## TIMEOUT from Dead Worker

If ack-wait returns TIMEOUT and `engram dispatch status` shows worker state DEAD: surface to user as "Agent X crashed during argument, argument lost" — not "Agent X refused to respond."

## Context Pressure

Check queue depth with `engram dispatch status`. At 100+ messages: summarize completed tasks to one-liners. At 300+: tell user to start a fresh session.

## Compaction Recovery

Run `engram dispatch status` to re-derive agent states. Run `engram hold check` to re-derive hold states. Post an info message announcing re-initialization. Resume routing.

## Shutdown

Triggered by "done", "shut down", "stand down", or similar.

1. Run dispatch drain (timeout 60s) to complete all in-flight work.
2. Run `engram dispatch stop` to send shutdown to all workers and exit dispatch.
3. Post your own done message and exit.
```

- [ ] **Step 2: Run tests to verify GREEN**

```bash
bash skills/engram-lead/tests/behavioral_test.sh
```

Expected: All tests PASS.

- [ ] **Step 3: Commit**

```bash
git add skills/engram-lead/
git commit -m "feat(skill): add engram-lead skill with dispatch-based agent management"
```

---

## Task 3: Behavioral tests for engram-up (RED)

**Files:**
- Create: `skills/engram-up/tests/behavioral_test.sh`

- [ ] **Step 1: Create the test directory and file**

```bash
mkdir -p skills/engram-up/tests
```

- [ ] **Step 2: Write the behavioral test**

Create `skills/engram-up/tests/behavioral_test.sh`:

```bash
#!/usr/bin/env bash
# Behavioral tests for skills/engram-up/SKILL.md
# TDD: run before editing SKILL.md (RED), then after (GREEN).
# Usage: bash behavioral_test.sh

SKILL="$(dirname "$0")/../SKILL.md"
PASS=0
FAIL=0
FAILURES=()

pass() { echo "PASS: $1"; PASS=$((PASS + 1)); }
fail() { echo "FAIL: $1"; FAIL=$((FAIL + 1)); FAILURES+=("$1"); }

assert_contains() {
  local desc="$1" pattern="$2"
  if grep -qF "$pattern" "$SKILL"; then pass "$desc"; else fail "$desc [pattern not found: '$pattern']"; fi
}

assert_not_contains() {
  local desc="$1" pattern="$2"
  if ! grep -qF "$pattern" "$SKILL"; then pass "$desc"; else fail "$desc [pattern should NOT be present: '$pattern']"; fi
}

echo "=== engram-up SKILL.md behavioral tests ==="
echo ""

echo "--- Group 1: Delegates to engram-lead ---"
assert_contains     "references engram-lead"                   'engram-lead'
assert_contains     "references use-engram-chat-as"            'engram:use-engram-chat-as'
assert_not_contains "does not duplicate routing table"         '| User Request |'
assert_not_contains "does not duplicate hold patterns"         'engram hold acquire'
assert_not_contains "does not duplicate spawn template"        'active <role>'
assert_not_contains "does not duplicate shutdown steps"        'dispatch drain'

echo ""
echo "--- Group 2: Triggers ---"
assert_contains "triggers on /engram"       '/engram'
assert_contains "triggers on /engram-up"    '/engram-up'
assert_contains "triggers on start engram"  'start engram'

echo ""
echo "=== Results ==="
echo "PASS: $PASS"
echo "FAIL: $FAIL"
if [ "${#FAILURES[@]}" -gt 0 ]; then
  echo ""
  echo "Failed tests:"
  for f in "${FAILURES[@]}"; do echo "  - $f"; done
fi
if [ "$FAIL" -gt 0 ]; then exit 1; else echo "All tests passed."; exit 0; fi
```

- [ ] **Step 3: Run tests to verify RED**

```bash
bash skills/engram-up/tests/behavioral_test.sh
```

Expected: FAIL — current engram-up duplicates startup steps, references `use-engram-chat-as` directly but not `engram-lead`, and doesn't mention `engram-lead`.

---

## Task 4: Update engram-up/SKILL.md (GREEN)

**Files:**
- Modify: `skills/engram-up/SKILL.md`

- [ ] **Step 1: Replace engram-up with thin entry point**

Replace the full content of `skills/engram-up/SKILL.md` with:

```markdown
---
name: engram-up
description: "Use when the user says /engram, /engram-up, \"start engram\", or wants to begin a multi-agent orchestrated session with memory."
---

# Engram Up

Use `engram:use-engram-chat-as` with role `engram-lead`.

All operational content — dispatch startup, routing, hold patterns, agent lifecycle, shutdown — lives in `engram-lead`.
```

- [ ] **Step 2: Run tests to verify GREEN**

```bash
bash skills/engram-up/tests/behavioral_test.sh
```

Expected: All tests PASS.

- [ ] **Step 3: Commit**

```bash
git add skills/engram-up/
git commit -m "refactor(skill): simplify engram-up to delegate to engram-lead"
```

---

## Task 5: Behavioral tests for engram-down (RED)

**Files:**
- Create: `skills/engram-down/tests/behavioral_test.sh`

- [ ] **Step 1: Create the test directory and file**

```bash
mkdir -p skills/engram-down/tests
```

- [ ] **Step 2: Write the behavioral test**

Create `skills/engram-down/tests/behavioral_test.sh`:

```bash
#!/usr/bin/env bash
# Behavioral tests for skills/engram-down/SKILL.md
# TDD: run before editing SKILL.md (RED), then after (GREEN).
# Usage: bash behavioral_test.sh

SKILL="$(dirname "$0")/../SKILL.md"
PASS=0
FAIL=0
FAILURES=()

pass() { echo "PASS: $1"; PASS=$((PASS + 1)); }
fail() { echo "FAIL: $1"; FAIL=$((FAIL + 1)); FAILURES+=("$1"); }

assert_contains() {
  local desc="$1" pattern="$2"
  if grep -qF "$pattern" "$SKILL"; then pass "$desc"; else fail "$desc [pattern not found: '$pattern']"; fi
}

assert_not_contains() {
  local desc="$1" pattern="$2"
  if ! grep -qF "$pattern" "$SKILL"; then pass "$desc"; else fail "$desc [pattern should NOT be present: '$pattern']"; fi
}

echo "=== engram-down SKILL.md behavioral tests ==="
echo ""

echo "--- Group 1: Conditional tmux tail kill ---"
assert_contains     "checks TMUX env var"               '$TMUX'
assert_contains     "uses @engram_name to find pane"    '@engram_name'
assert_not_contains "does not use pane_title to find"   'pane_title'
assert_contains     "kill-pane command present"         'kill-pane'
assert_contains     "conditional: skip if not in tmux"  'not in tmux'

echo ""
echo "--- Group 2: Core shutdown sequence intact ---"
assert_contains "dispatch drain present"   'dispatch drain'
assert_contains "dispatch stop present"    'dispatch stop'
assert_contains "scan LEARNED messages"    'LEARNED'
assert_contains "session summary step"     'session summary'
assert_contains "preserve chat file"       'chat file'

echo ""
echo "--- Group 3: Common mistakes table ---"
assert_contains "common mistakes table"    'Common Mistakes'

echo ""
echo "=== Results ==="
echo "PASS: $PASS"
echo "FAIL: $FAIL"
if [ "${#FAILURES[@]}" -gt 0 ]; then
  echo ""
  echo "Failed tests:"
  for f in "${FAILURES[@]}"; do echo "  - $f"; done
fi
if [ "$FAIL" -gt 0 ]; then exit 1; else echo "All tests passed."; exit 0; fi
```

- [ ] **Step 3: Run tests to verify RED**

```bash
bash skills/engram-down/tests/behavioral_test.sh
```

Expected: FAIL — current engram-down doesn't check `$TMUX`, uses `pane_title`, and has no conditional tail-kill at all.

---

## Task 6: Update engram-down/SKILL.md (GREEN)

**Files:**
- Modify: `skills/engram-down/SKILL.md`

- [ ] **Step 1: Add conditional tail-kill step after Step 3**

Replace the full `skills/engram-down/SKILL.md` content with:

```markdown
---
name: engram-down
description: Use when shutting down an engram multi-agent session, when the user says "done", "shut down", "stand down", "close engram", or "stop engram". Drains in-flight dispatch work and reports session summary.
---

# Engram Down

Shutdown skill for engram multi-agent sessions. Drains in-flight dispatch work, stops agents gracefully, and reports session stats.

## Shutdown Sequence

### Step 1: Broadcast shutdown

Post `shutdown` to `"all"` (use your agent name from your `ready` message):

```toml
[[message]]
from = "lead"
to = "all"
thread = "lifecycle"
type = "shutdown"
text = "Session complete. Shutting down."
```

### Step 2: Drain in-flight work

```
engram dispatch drain --secs 30
```

Blocks until all in-flight tasks complete or timeout elapses. Do not skip — stopping before drain risks losing work.

### Step 3: Stop dispatch

```
engram dispatch stop
```

### Step 4: Kill chat-tail pane (tmux only)

If `$TMUX` is set, kill the chat observer tail pane. Skip silently if not in tmux.

```bash
if [ -n "$TMUX" ]; then
  tmux list-panes -a -F '#{pane_id} #{@engram_name}' \
    | grep 'chat-tail' \
    | awk '{print $1}' \
    | xargs -I{} tmux kill-pane -t {}
fi
```

Uses `@engram_name` (tmux user option, immune to OSC 2 terminal overwrites) — not `pane_title`.

### Step 5: Scan for LEARNED messages

Before posting the session summary, read from your session-start cursor forward and collect all `LEARNED` messages. Skipping this risks posting a summary before final facts arrive.

### Step 6: Report session summary

Tell the user: agents spawned, tasks completed vs in-flight, decisions made, facts learned, open questions.

### Step 7: Preserve chat file

**Do NOT truncate or delete the chat file.** Persistent across sessions.

## Common Mistakes

| Mistake | Fix |
|---------|-----|
| Skip `dispatch drain` | In-flight tasks are lost |
| Post summary before scanning LEARNED | Facts may arrive after summary — scan first |
| Truncate chat file | Never — persistent record |
| Wrong `from` name | Use your name from the `ready` message, not `lead` |
| Use `pane_title` for tail-pane lookup | Use `@engram_name` — terminal OSC 2 overwrites `pane_title` |
| Kill tail pane outside tmux | Check `$TMUX` first — not in tmux means no tail pane exists |
```

- [ ] **Step 2: Run tests to verify GREEN**

```bash
bash skills/engram-down/tests/behavioral_test.sh
```

Expected: All tests PASS.

- [ ] **Step 3: Commit**

```bash
git add skills/engram-down/
git commit -m "fix(skill): engram-down conditional tmux tail-kill using @engram_name (#536)"
```

---

## Task 7: Deprecate engram-tmux-lead

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md`

- [ ] **Step 1: Replace engram-tmux-lead with deprecation notice**

Replace the full content of `skills/engram-tmux-lead/SKILL.md` with:

```markdown
---
name: engram-tmux-lead
description: DEPRECATED — use engram-lead instead. Retained for backward compatibility with /engram-tmux-lead triggers.
---

# Engram tmux Lead (Deprecated)

This skill has been renamed to **engram-lead**.

Use `engram:use-engram-chat-as` with role `engram-lead` instead.

All functionality — routing, hold patterns, dispatch-based agent management, optional tmux tail pane — is in `engram-lead`.
```

- [ ] **Step 2: Verify the deprecation notice is usable**

Read the file to confirm it is clear and non-empty:

```bash
cat skills/engram-tmux-lead/SKILL.md
```

- [ ] **Step 3: Commit**

```bash
git add skills/engram-tmux-lead/SKILL.md
git commit -m "deprecate(skill): redirect engram-tmux-lead to engram-lead (#554)"
```

---

## Task 8: Close issues and report

- [ ] **Step 1: Close related issues**

```bash
gh issue close 554 --comment "Implemented: engram-lead skill created with dispatch-based agent management; tmux limited to optional chat-tail pane; engram-up delegates to engram-lead; engram-down uses @engram_name for tail-kill."
gh issue close 505 --comment "Obsolete: engram-lead removes all tmux pane layout logic. No more split-window rules or pane counts."
gh issue close 506 --comment "Obsolete: engram-lead removes tmux-based concurrency tracking. Dispatch manages agent lifecycle."
gh issue close 523 --comment "Obsolete: engram-lead uses engram dispatch assign/stop, not tmux send-keys to spawn agents."
gh issue close 536 --comment "Fixed in engram-down: tail-kill now uses @engram_name (immune to OSC 2) and is conditional on \$TMUX."
gh issue close 541 --comment "Obsolete: engram-lead reads chat file for all diagnostics; tmux capture-pane removed entirely."
```

- [ ] **Step 2: Run all behavioral tests as final verification**

```bash
bash skills/engram-lead/tests/behavioral_test.sh && \
bash skills/engram-up/tests/behavioral_test.sh && \
bash skills/engram-down/tests/behavioral_test.sh
```

Expected: All three test suites pass.
