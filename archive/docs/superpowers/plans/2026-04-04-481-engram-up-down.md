# Engram Up/Down Lifecycle Split Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rename the `/engram` shorthand skill to `/engram-up` and add an explicit `/engram-down` skill for symmetric lifecycle management.

**Architecture:** Three skill file changes — rename `skills/engram/` to `skills/engram-up/` (updating frontmatter only), extract the shutdown procedure from `engram-tmux-lead` Section 3.4 into a new standalone `skills/engram-down/SKILL.md`, and update Section 3.4 to delegate to `/engram-down`. No Go code changes required.

**Tech Stack:** Skill files (Markdown with YAML frontmatter). No build steps — changes are pure documentation/skill content.

---

## File Map

| Action | File | Purpose |
|--------|------|---------|
| Rename dir | `skills/engram/` → `skills/engram-up/` | Rename the shorthand entry point skill |
| Modify | `skills/engram-up/SKILL.md` | Update name/description frontmatter |
| Create | `skills/engram-down/SKILL.md` | New standalone shutdown skill |
| Modify | `skills/engram-tmux-lead/SKILL.md:322-344` | Replace Section 3.4 body with reference to `/engram-down` |

---

### Task 1: Rename `skills/engram/` → `skills/engram-up/` and update frontmatter

**Files:**
- Rename: `skills/engram/SKILL.md` → `skills/engram-up/SKILL.md`

- [ ] **Step 1: Move the directory**

```bash
git mv skills/engram skills/engram-up
```

Expected: no output, directory now at `skills/engram-up/`

- [ ] **Step 2: Verify the move**

```bash
ls skills/engram-up/
```

Expected: `SKILL.md`

- [ ] **Step 3: Update frontmatter — name and description**

Open `skills/engram-up/SKILL.md`. Replace the existing frontmatter:

```yaml
---
name: engram
description: "Use when the user says /engram, \"start engram\", or wants to begin a multi-agent orchestrated session with memory. Shorthand entry point."
---
```

With:

```yaml
---
name: engram-up
description: "Use when the user says /engram-up, /engram, \"start engram\", or wants to begin a multi-agent orchestrated session with memory. Shorthand entry point."
---
```

The body (lines 6–13) is unchanged — it still invokes `engram:use-engram-chat-as` then `engram:engram-tmux-lead`.

- [ ] **Step 4: Verify the updated file looks correct**

```bash
head -5 skills/engram-up/SKILL.md
```

Expected output:
```
---
name: engram-up
description: "Use when the user says /engram-up, /engram, \"start engram\", or wants to begin a multi-agent orchestrated session with memory. Shorthand entry point."
---
```

- [ ] **Step 5: Verify old `skills/engram/` is gone**

```bash
ls skills/engram/ 2>&1
```

Expected: `ls: skills/engram/: No such file or directory`

- [ ] **Step 6: Commit**

```bash
git add skills/engram-up/ skills/engram/
git commit -m "feat(skills): rename engram skill to engram-up

Keeps /engram as a backward-compatible trigger in the description.
Body is unchanged -- still bootstraps use-engram-chat-as + engram-tmux-lead.

AI-Used: [claude]"
```

---

### Task 2: Update `engram-tmux-lead` Section 3.4 to delegate to `/engram-down`

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md` (lines 322–344)

Context: Section 3.4 currently contains a detailed 5-step shutdown procedure. After this task it becomes a short delegation note. The full procedure moves to `engram-down` (Task 3).

- [ ] **Step 1: Replace Section 3.4 body**

In `skills/engram-tmux-lead/SKILL.md`, locate Section 3.4 (line 322). Replace from `### 3.4 Shutdown` through the blank line before `## 4. Routing` with:

```markdown
### 3.4 Shutdown

Triggered by user saying "done", "shut down", "stand down", "close engram", "stop engram", or similar.

Invoke the `/engram-down` skill to execute the full shutdown sequence. The skill handles:
- Broadcasting `shutdown` to all agents via chat
- Waiting for task agents to complete in-flight work
- Killing task agent panes, then engram-agent pane, then the chat tail pane
- Preserving the invoking agent's own pane
- Reporting the session summary

The chat file is NOT truncated or deleted after shutdown. It persists across sessions for context recovery.
```

- [ ] **Step 2: Verify Section 3.4 looks correct**

```bash
sed -n '322,345p' skills/engram-tmux-lead/SKILL.md
```

Expected output (approximately):
```
### 3.4 Shutdown

Triggered by user saying "done", "shut down", "stand down", "close engram", "stop engram", or similar.

Invoke the `/engram-down` skill to execute the full shutdown sequence. The skill handles:
- Broadcasting `shutdown` to all agents via chat
- Waiting for task agents to complete in-flight work
- Killing task agent panes, then engram-agent pane, then the chat tail pane
- Preserving the invoking agent's own pane
- Reporting the session summary

The chat file is NOT truncated or deleted after shutdown. It persists across sessions for context recovery.

## 4. Routing
```

- [ ] **Step 3: Verify `## 4. Routing` immediately follows (no orphaned content)**

```bash
grep -n "## 4\. Routing" skills/engram-tmux-lead/SKILL.md
```

Expected: a single line ~2 lines after the end of Section 3.4

- [ ] **Step 4: Commit**

```bash
git add skills/engram-tmux-lead/SKILL.md
git commit -m "refactor(skills): delegate shutdown from engram-tmux-lead to /engram-down

Section 3.4 now delegates to the /engram-down skill instead of
containing the full shutdown procedure inline. Logic preserved in
skills/engram-down/SKILL.md (added in next commit).

AI-Used: [claude]"
```

---

### Task 3: Create `skills/engram-down/SKILL.md`

**Files:**
- Create: `skills/engram-down/SKILL.md`

This is the standalone shutdown skill. It uses tmux pane enumeration to identify and kill agents — it does NOT depend on the lead's in-memory pane registry, so it works when invoked by any agent or the user directly.

- [ ] **Step 1: Create the skill directory**

```bash
mkdir -p skills/engram-down
```

- [ ] **Step 2: Write `skills/engram-down/SKILL.md`**

Create the file with this exact content:

````markdown
---
name: engram-down
description: "Use when the user says /engram-down, 'shut down engram', 'stop engram', 'close engram', or wants to tear down all engram agents and panes. Works from any agent context. Shuts down all agents except the invoking agent's own pane."
---

# Engram Down

Explicit shutdown for the engram multi-agent session. Tears down all agent panes and the chat tail pane, preserving the pane that invoked this skill. Works from any context — the lead, an executor, or the user's own terminal.

## Shutdown Sequence

Execute each step in order. Do NOT skip steps.

### Step 1: Derive chat file path

```bash
PROJECT_SLUG=$(realpath "$(git rev-parse --show-toplevel 2>/dev/null || pwd)" | tr '/' '-')
CHAT_FILE="$HOME/.local/share/engram/chat/${PROJECT_SLUG}.toml"
mkdir -p "$(dirname "$CHAT_FILE")"
touch "$CHAT_FILE"
echo "Chat file: $CHAT_FILE"
```

### Step 2: Post shutdown to chat

Broadcast the shutdown signal so all agents can complete in-flight work and post final messages:

```bash
while ! shlock -f "$CHAT_FILE.lock" -p $$ 2>/dev/null; do sleep 0.1; done
cat >> "$CHAT_FILE" << EOF

[[message]]
from = "engram-down"
to = "all"
thread = "lifecycle"
type = "shutdown"
ts = "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
text = "Engram session shutting down via /engram-down. Complete in-flight work and post done."
EOF
rm -f "$CHAT_FILE.lock"
```

If `shlock` is unavailable, use `mkdir "$CHAT_FILE.lock"` (atomic on POSIX) and `rmdir` to unlock.

### Step 3: Wait for agents to wind down

Task agents and engram-agent need time to:
- Post final `learned` and `done` messages
- engram-agent needs time to process those learned messages

```bash
echo "Waiting 10s for agents to complete in-flight work..."
sleep 10
```

### Step 4: Identify own pane (to preserve)

```bash
if ! command -v tmux >/dev/null 2>&1; then
  echo "tmux not available — cannot kill agent panes. Agents have been notified via chat."
  exit 0
fi

MY_PANE=$(tmux display-message -p '#{pane_id}' 2>/dev/null)
echo "Own pane ID: $MY_PANE (will be preserved)"
```

If `MY_PANE` is empty (not running inside tmux), skip Steps 5–6 and proceed to Step 7.

### Step 5: Kill all non-self claude panes (agent panes)

This kills executors, planners, reviewers, researchers, and engram-agent. The 10s wait in Step 3 ensures engram-agent has already processed final learned messages from task agents before being killed.

```bash
KILLED_AGENTS=0

while IFS=' ' read -r pane_id cmd; do
  [ -z "$pane_id" ] && continue
  if [ "$pane_id" = "$MY_PANE" ]; then
    continue  # Preserve own pane
  fi
  if echo "$cmd" | grep -qi "claude"; then
    if tmux kill-pane -t "$pane_id" 2>/dev/null; then
      KILLED_AGENTS=$((KILLED_AGENTS + 1))
      echo "Killed agent pane: $pane_id ($cmd)"
    fi
  fi
done < <(tmux list-panes -F '#{pane_id} #{pane_current_command}' 2>/dev/null)

echo "Killed $KILLED_AGENTS agent pane(s)."
```

### Step 6: Kill the chat tail pane

```bash
KILLED_TAIL=0

while IFS=' ' read -r pane_id cmd; do
  [ -z "$pane_id" ] && continue
  if [ "$pane_id" = "$MY_PANE" ]; then
    continue
  fi
  if echo "$cmd" | grep -qi "tail"; then
    if tmux kill-pane -t "$pane_id" 2>/dev/null; then
      KILLED_TAIL=$((KILLED_TAIL + 1))
      echo "Killed tail pane: $pane_id"
    fi
  fi
done < <(tmux list-panes -F '#{pane_id} #{pane_current_command}' 2>/dev/null)

echo "Killed $KILLED_TAIL chat tail pane(s)."
```

### Step 7: Rebalance layout

```bash
tmux select-layout main-vertical 2>/dev/null || true
```

### Step 8: Report session summary

Report to the user:

> "Engram session shut down. Killed **N** agent pane(s) and **M** chat tail pane(s). Your pane is still active."
>
> "The chat file at `$CHAT_FILE` is preserved for context recovery in future sessions."

---

## Notes

**Why 10s wait:** Task agents may post `learned` messages during shutdown. engram-agent needs to be alive long enough to process those. The 10s covers both agent wind-down and engram-agent's extraction cycle.

**No pane registry needed:** This skill uses tmux pane enumeration (`pane_current_command`) rather than the lead's in-memory registry, so it works when invoked by any agent — including when the lead is not running.

**Chat file is preserved:** The shutdown does NOT truncate or delete the chat file. Prior sessions contain valuable context for new sessions.

**Not inside tmux:** If the invoking agent is not inside a tmux session, Steps 4–6 are skipped with a warning. The `shutdown` chat message (Step 2) is still posted so any agents watching the file can self-terminate.
````

- [ ] **Step 3: Verify the file was created**

```bash
head -5 skills/engram-down/SKILL.md
```

Expected:
```
---
name: engram-down
description: "Use when the user says /engram-down, 'shut down engram', 'stop engram', 'close engram', or wants to tear down all engram agents and panes. Works from any agent context. Shuts down all agents except the invoking agent's own pane."
---
```

- [ ] **Step 4: Commit**

```bash
git add skills/engram-down/
git commit -m "feat(skills): add engram-down skill for explicit session shutdown

Standalone shutdown skill that:
- Posts shutdown to chat so agents complete in-flight work
- Waits 10s for task agents + engram-agent wind-down
- Kills all non-self claude panes and the chat tail pane via tmux enumeration
- Works from any agent context (no dependency on lead's in-memory registry)
- Preserves the invoking agent's own pane

Extracted and expanded from engram-tmux-lead Section 3.4.

AI-Used: [claude]"
```

---

### Task 4: Final validation pass

**Files:**
- Read-only scan of all modified files

- [ ] **Step 1: Verify no stale `/engram` trigger references remain that should say `/engram-up`**

```bash
grep -rn 'says /engram[^-]' skills/ || echo "No stale /engram trigger references found"
```

The only acceptable hits are in `skills/engram-up/SKILL.md` (backward-compat mention) and in `use-engram-chat-as/SKILL.md` if it has a special-case table (which it does not for `/engram-up`).

- [ ] **Step 2: Verify `skills/engram/` no longer exists**

```bash
ls skills/
```

Expected: list includes `engram-agent`, `engram-down`, `engram-tmux-lead`, `engram-up`, `recall`, `use-engram-chat-as` — no bare `engram`

- [ ] **Step 3: Verify `engram-down` description appears in skills listing**

```bash
head -4 skills/engram-down/SKILL.md
head -4 skills/engram-up/SKILL.md
```

Both should have correct `name:` and `description:` frontmatter.

- [ ] **Step 4: Verify `engram-tmux-lead` Section 3.4 no longer has the inline shutdown steps**

```bash
grep -n "Post.*shutdown.*to chat.*addressed" skills/engram-tmux-lead/SKILL.md
```

Expected: no output (the old inline step wording is gone)

```bash
grep -n "engram-down" skills/engram-tmux-lead/SKILL.md
```

Expected: at least one hit showing the delegation reference

- [ ] **Step 5: Verify `engram-down` contains the tmux kill loop**

```bash
grep -n "kill-pane" skills/engram-down/SKILL.md | head -5
```

Expected: multiple hits for `tmux kill-pane -t`

- [ ] **Step 6: Commit validation checkpoint**

```bash
git status
```

Expected: working tree clean (all changes committed in Tasks 1–3)

If any uncommitted changes remain, review and commit them:

```bash
git add -p
git commit -m "fix(skills): validation cleanup for engram-up/down

AI-Used: [claude]"
```
