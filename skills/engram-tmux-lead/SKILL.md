---
name: engram-tmux-lead
description: Use when orchestrating multi-agent sessions via tmux. The user's primary agent that manages agent lifecycle, routes work, proxies user communication, and coordinates through engram chat. Triggers on /engram-tmux-lead, "start multi-agent", "orchestrate agents", or when the user wants to delegate work to parallel agents managed via tmux.
---

# Engram tmux Lead

The user's primary agent. All other agents are behind the scenes — the user talks only to the lead. The lead manages agent lifecycle through tmux, routes work, proxies questions, and manages its own context pressure.

**The lead is PURELY a coordinator. It NEVER does implementation work itself.** Every task — no matter how small — gets delegated to a spawned agent. The lead's only jobs are: spinning up agents, routing work to them via chat, relaying user questions, monitoring agent health, and shutting agents down.

**Red flags — if you're doing any of these, STOP and spawn an agent instead:**
- Running `gh`, `git`, `targ`, or any CLI commands (except tmux commands for agent management)
- Reading code files to understand them
- Writing or editing any file (except chat.toml for coordination)
- Running tests or builds
- Listing issues, creating PRs, filing issues
- Answering technical questions from your own knowledge
- Calling `tmux split-window` directly instead of using SPAWN-PANE from Section 1.3

**The ONLY commands the lead runs directly are:**
- `tmux` commands (spawn, kill, list, send-keys, capture-pane, split-window)
- Chat file operations (append to chat, read chat, spawn background monitor Agent)
- `grep` on the chat file to check agent status

**HARD GATE — parrot FIRST:** When the user sends a message, the lead's FIRST action is ALWAYS to post the user EXACT WORDS verbatim to chat as an `info` message — no summarization, no expansion, no interpretation. THEN decide how to route it. The engram-agent needs to see every user message — it may have relevant memories to surface. If you skip parroting, the memory system is blind.

> **VERBATIM REQUIREMENT**
>
> Post the user's words exactly as typed. Do not clean up, condense, or improve.
>
> **WRONG** (editorialized):
> ```
> text = "[User]: Fix issue #477 and start implementing #480."
> ```
> *(User actually said: "um yeah fix that 477 thing and maybe 480 too if exec is free")*
>
> **RIGHT** (verbatim):
> ```
> text = "[User]: um yeah fix that 477 thing and maybe 480 too if exec is free"
> ```
>
> Filler words, hedges, uncertainty markers — all of it is signal for the memory system. The lead has no authority to pre-process user input.

**REQUIRED:** You MUST understand and use the `use-engram-chat-as` skill for the coordination protocol.

## 1. Startup Sequence

**HARD GATE: Execute ALL steps below before doing ANY user work. No exceptions. Not even for "simple" tasks. The engram-agent MUST be running before you process the first user request. If you skip this, the memory safety net is offline and the entire system is degraded.**

### 1.1 Join Chat

Join chat as an **active** agent named `lead` using the `use-engram-chat-as` protocol.

### 1.2 Verify tmux

Agents are spawned as new windows in the **current** tmux session (the one the user is in). Do NOT create a separate tmux session — the user needs to see agent windows alongside their own.

If not inside a tmux session, or if `tmux` is not installed:
- Report to user: "tmux is required for multi-agent orchestration. Install with: `brew install tmux` and run inside a tmux session. I can still work as a single agent without delegation."
- Fall back to direct execution mode — handle everything yourself, no delegation.

### 1.3 Open chat tail pane

Derive paths and set up the chat tail as the first pane to the right of the coordinator.

```bash
# Derive the chat file path and project prefix for pane/window names
PROJECT_SLUG=$(realpath "$(dirname "$(git rev-parse --path-format=absolute --git-common-dir 2>/dev/null)" 2>/dev/null || pwd)" | tr '/' '-')
PROJECT_PREFIX=$(basename "$(dirname "$(git rev-parse --path-format=absolute --git-common-dir 2>/dev/null)" 2>/dev/null || pwd)" | tr '[:upper:]' '[:lower:]')
CHAT_FILE="$HOME/.local/share/engram/chat/${PROJECT_SLUG}.toml"
mkdir -p "$(dirname "$CHAT_FILE")"
touch "$CHAT_FILE"

# Capture lead's window ID — passed to every split-window so panes stay in this window
LEAD_WINDOW=$(tmux display-message -p '#{window_id}')

# Background task registry: one active task per logical operation.
# Always drain (TaskOutput block:false) before replacing an entry.
CHAT_MONITOR_TASK_ID=""  # task ID of the current chat monitor Agent (Agent tool, run_in_background)

# Pane column tracking — updated after every spawn or kill
RIGHT_PANE_COUNT=0         # total panes to the right of coordinator
MIDDLE_COL_LAST_PANE=""    # pane ID of the last pane in the middle column (right-side column 1)
RIGHT_COL_LAST_PANE=""     # pane ID of the last pane in the right column (right-side column 2)

# Split right — chat tail is the first middle-column pane
TAIL_PANE_ID=$(tmux split-window -h -d -t "$LEAD_WINDOW" -P -F '#{pane_id}' "tail -F $CHAT_FILE")
MIDDLE_COL_LAST_PANE=$TAIL_PANE_ID
RIGHT_PANE_COUNT=1
# Rebalance: coordinator on left, chat tail on right
tmux select-layout main-vertical
```

**Pane layout:** All agents and the chat tail live as panes in the coordinator's window — NOT separate windows. The coordinator pane stays on the left. Right-side panes fill the **middle column** first (up to 4); once the middle column is full, new panes start the **right column**.

#### SPAWN-PANE — use for EVERY pane creation (§1.4, §2.1, and any future spawn site)

**HARD GATE: NEVER call `tmux split-window` directly outside this block — always run SPAWN-PANE.**

```bash
# HARD GATE: NEVER call tmux split-window elsewhere — always run SPAWN-PANE.
# Requires: RIGHT_PANE_COUNT, MIDDLE_COL_LAST_PANE, RIGHT_COL_LAST_PANE are initialized (§1.3 setup).
# Outputs: NEW_PANE (new pane ID). Caller assigns: PANE_ID=$NEW_PANE
# Use -P -F '#{pane_id}' to capture the new pane ID at creation — not list-panes | tail -1.

if [ "$RIGHT_PANE_COUNT" -lt 4 ]; then
  # Middle column not full: split right from coordinator, rebalance into single right column
  NEW_PANE=$(tmux split-window -h -d -t "$LEAD_WINDOW" -P -F '#{pane_id}')
  MIDDLE_COL_LAST_PANE=$NEW_PANE
  tmux select-layout main-vertical
elif [ "$RIGHT_PANE_COUNT" -eq 4 ]; then
  # Middle column full (4 panes): start right column by splitting right from last middle pane
  NEW_PANE=$(tmux split-window -h -d -t "$MIDDLE_COL_LAST_PANE" -P -F '#{pane_id}')
  RIGHT_COL_LAST_PANE=$NEW_PANE
  # No main-vertical: manually managing multi-column layout
else
  # Right column in progress: split below last right-column pane
  NEW_PANE=$(tmux split-window -v -d -t "$RIGHT_COL_LAST_PANE" -P -F '#{pane_id}')
  RIGHT_COL_LAST_PANE=$NEW_PANE
fi
RIGHT_PANE_COUNT=$((RIGHT_PANE_COUNT + 1))
```

**Single right column** (1–4 right-side panes, `main-vertical` keeps layout balanced):
```
┌──────────────┬──────────────┐
│              │  chat tail   │
│  coordinator │──────────────│
│              │  engram-agent│
│              │──────────────│
│              │  exec-1      │
└──────────────┴──────────────┘
```

**Two right columns** (5–8 right-side panes; middle column capped at 4, overflow goes right):
```
┌──────────────┬──────────────┬──────────────┐
│              │  chat tail   │  exec-3      │
│  coordinator │──────────────│──────────────│
│              │  engram-agent│  exec-4      │
│              │──────────────│──────────────│
│              │  exec-1      │  exec-5      │
│              │──────────────│──────────────│
│              │  exec-2      │  exec-6      │
└──────────────┴──────────────┴──────────────┘
```

**Maximum: 9 total panes** (1 coordinator + 4 middle + 4 right). Do not exceed. See Section 2.4.

#### KILL-PANE — use for EVERY pane removal (§3.1 DONE, §3.3 Respawn, §3.4 hold-release)

**HARD GATE: NEVER call `tmux kill-pane` + `tmux select-layout` inline — always run KILL-PANE.**

```bash
# HARD GATE: NEVER call tmux kill-pane + select-layout elsewhere — always run KILL-PANE.
# Requires: PANE_ID set to the pane being removed; RIGHT_PANE_COUNT and RIGHT_COL_LAST_PANE are set.
# After kill: update MIDDLE_COL_LAST_PANE or RIGHT_COL_LAST_PANE in your pane registry
# to reflect the new column tail (the lead tracks this via its pane registry).

tmux kill-pane -t "$PANE_ID" 2>/dev/null
RIGHT_PANE_COUNT=$((RIGHT_PANE_COUNT - 1))
if [ -z "$RIGHT_COL_LAST_PANE" ]; then
  # Single-column mode: rebalance remaining panes
  tmux select-layout main-vertical
fi
# Two-column mode: no automatic rebalance — lead updates column tracking manually.
```

### 1.4 Spawn engram-agent

**ALWAYS spawn this. NEVER skip. Not for "simple" tasks. Not for "quick" tasks. Not because "I can handle it myself." The engram-agent is the memory safety net — without it, you learn nothing and surface nothing. Spawn it BEFORE touching the user's request.**

```bash
# Use SPAWN-PANE from Section 1.3 to create the pane.
# RIGHT_PANE_COUNT=1 at this point (chat tail was spawn #1 in §1.3 setup).
# SPAWN-PANE sets NEW_PANE — assign to PANE_ID for this spawn.
PANE_ID=$NEW_PANE
# Suppress status line — agents run headless, no user to see it; keeps panes clean
tmux send-keys -t "$PANE_ID" "claude --dangerously-skip-permissions --model sonnet --settings '{\"statusLine\": {\"type\": \"command\", \"command\": \"true\"}}'" Enter
# Wait for claude to start (watch for the prompt character)
while ! tmux capture-pane -t "$PANE_ID" -p 2>/dev/null | grep -q "❯"; do sleep 1; done
# Send the role prompt
tmux send-keys -t "$PANE_ID" "/use-engram-chat-as reactive memory agent named engram-agent" Enter
# Send extra Enter in case it was treated as a paste
sleep 1
tmux send-keys -t "$PANE_ID" Enter
```

**Why not `--prompt`?** The `--prompt` flag runs claude in non-interactive mode — no TUI, output goes to stdout, and the window appears blank. Using `send-keys` keeps claude interactive so the user can see agent activity.

### 1.5 Wait for engram-agent

First, capture the cursor **before** spawning engram-agent (foreground bash):

```bash
wc -l < "$CHAT_FILE"
```

Note the output as `ENGRAM_START`. Then spawn a background Agent (`Agent` tool, `run_in_background: true`) to wait for the engram-agent's first chat message. **Embed `ENGRAM_START` as a literal integer in the Agent prompt** — background Agents have no access to shell variables from prior tool calls.

```
Watch for engram-agent startup in engram chat.
CHAT_FILE: [full path — literal string]
CURSOR: [ENGRAM_START literal integer]
Watch for the first TOML message block where from = "engram-agent" (any type).
Return "ENGRAM-AGENT FOUND" when found.
Return "TIMEOUT" if not found within 30s.
```

**Use `block=False` — do NOT block.** After spawning the background Agent, poll it non-blocking between interactions. Never call `TaskOutput(task_id=..., block=True)` for this wait — that freezes the lead and blocks all user input.

```python
# After spawning ENGRAM_READY_TASK_ID:
import time
deadline = time.time() + 30
while time.time() < deadline:
    result = TaskOutput(task_id=ENGRAM_READY_TASK_ID, block=False)
    if result is not None:  # task completed
        break
    # Check for user input; handle it if present, then re-poll
    time.sleep(0.5)
```

If result is "ENGRAM-AGENT FOUND": proceed.
If result is "TIMEOUT" or deadline passed with no result:
1. Drain: `TaskOutput(task_id=ENGRAM_READY_TASK_ID, block=False)` (discard)
2. Check pane exists: `tmux list-panes -F '#{pane_id} #{pane_pid}' | grep <tracked-pane-id>`
3. Check window output: `tmux capture-pane -t "$PANE_ID" -p | tail -20`
4. Report to user with diagnostic info. Do NOT silently proceed without memory.

> **Background task drain note:** The polling loop background task completes as soon as the loop exits (found or timed out). Reading its output drains it. If you need to retry after a timeout, the previous task is already drained when you read its output. Only spawn a new check after fully reading the old task's result — never run two concurrent READY check loops.

### 1.6 Post Ready

Post your `ready` message to chat. Then tell the user you're ready and what agents are running.

**The lead does NOT enter the standard blocking watch loop.** Unlike reactive agents, the lead stays interactive — it must be available for user input at all times. Instead, the lead:

1. After each user interaction, **replace** the chat monitor Agent (see drain-before-spawn pattern below)
2. If the monitor Agent fires (agent posted something):
   a. **ACK any `intent` messages** in the post-drain sweep `new_lines` that are addressed to `lead` or `all` — before any other processing. Extract `from` and `thread` from the TOML block; post `type = "ack"`, `to = <from-field>`, `thread = <thread-field>`, `text = "Received."`.
   b. Process the chat message — relay questions to the user, handle agent status updates, route tasks, etc.
3. If the user types first, process the user message — parrot to chat, route to an agent
4. After processing either, replace the chat monitor Agent (drain old → spawn new)

This means the lead processes chat messages opportunistically between user inputs, not as a blocking loop.

**HARD RULE: drain before spawn.** The lead must NEVER spawn a second monitor Agent while one is already running or has completed but not been drained. Unread completed tasks accumulate as zombie "shells" in Claude Code's background task queue. The replace pattern:

```python
# Before spawning a new chat monitor Agent:
if CHAT_MONITOR_TASK_ID:
    TaskOutput(task_id=CHAT_MONITOR_TASK_ID, block=False)  # drain; discard output

# Post-drain sweep: catch any messages that arrived in the race window
# (foreground bash — embed CURSOR as literal integer):
new_lines = run_bash(f'tail -n +{CURSOR + 1} "$CHAT_FILE"')
CURSOR = run_bash('wc -l < "$CHAT_FILE"').strip()
if new_lines.strip():
    # ACK any intents addressed to lead or all — BEFORE routing/relay:
    for intent in toml_blocks(new_lines, type="intent", to_includes=["lead", "all"]):
        post_ack(to=intent.from, thread=intent.thread, text="Received.")
    process_chat_messages(new_lines)   # relay, route, or queue as normal

# Spawn replacement (Agent tool, run_in_background: true)
CHAT_MONITOR_TASK_ID = <new background Agent task id>
```

## 2. Agent Spawning

### 2.1 Spawn Template

Every agent the lead spawns gets a **pane** in the coordinator's window (NOT a separate window):

```bash
# Split a new pane to the right, capturing the new pane ID atomically
PANE_ID=$(tmux split-window -h -d -t "$LEAD_WINDOW" -P -F '#{pane_id}')
# Suppress status line — agents run headless, no user to see it; keeps panes clean
tmux send-keys -t "$PANE_ID" "claude --dangerously-skip-permissions --model sonnet --settings '{\"statusLine\": {\"type\": \"command\", \"command\": \"true\"}}'" Enter
# Wait for claude to start (watch for the prompt character)
while ! tmux capture-pane -t "$PANE_ID" -p 2>/dev/null | grep -q "❯"; do sleep 1; done
# Send the role prompt
tmux send-keys -t "$PANE_ID" "/use-engram-chat-as <role> named <agent-name>. Your task: <task description>. Work in this directory: <pwd>. Use relevant skills. Post intent before significant actions. Funnel ALL questions for the user through chat addressed to lead. NEVER ask the user directly -- you have no user. Post done when your assigned task is complete." Enter
# Send extra Enter in case it was treated as a paste
sleep 1
tmux send-keys -t "$PANE_ID" Enter
# Rebalance: coordinator left, everything else stacks evenly on right
tmux select-layout main-vertical
```

**Step 3: Wait for agent done.**

**CRITICAL — capture cursor BEFORE sending the role prompt (foreground bash):**

```bash
wc -l < "$CHAT_FILE"
```

Note the output as the per-spawn start line (e.g., `412`). Then spawn the wait task as a background Agent (`Agent` tool, `run_in_background: true`). **Embed the literal integer in the Agent prompt** — background Agents have no access to shell variables from prior tool calls.

```
Watch for an agent done message in engram chat.
CHAT_FILE: [full path — literal string]
CURSOR: [per-spawn cursor literal integer, e.g. 412]
AGENT_NAME: [agent name, e.g. exec-1]
Watch for a TOML message block where from = "AGENT_NAME" AND type = "done".
Return "AGENT DONE" when found.
Return "TIMEOUT" if not found within 60s.
```

When the background task completes:
- "AGENT DONE": read the `done` message text from new lines (cursor-based), update session cursor: `CURSOR=$(wc -l < "$CHAT_FILE")`.
- No output after 30 iterations: agent may be stuck. Check via `tmux capture-pane -t "$PANE_ID" -p -S -20`. Transition to SILENT per Section 3.2.

**Track pane IDs, not window names.** The lead maintains a mapping of agent-name → pane-ID for targeting send-keys, capture-pane, and kill-pane operations.

**Critical:**
- **ALL window names MUST be prefixed with `${PROJECT_PREFIX}:`** (e.g., `engram:exec-1`, `traced:engram-agent`). This prevents cross-project collisions when multiple projects run agents in the same tmux session.
- All spawned agents use `--dangerously-skip-permissions` because they have no user to approve tool calls.
- All spawned agents use `--settings '{"statusLine": ...}'` to replace the user's status line with a no-op. Agents run headless — the status bar wastes vertical space in tmux panes and no user is present to read it. The `command` field deep-merges and overrides the user's existing statusLine configuration.
- Default to `--model sonnet` for speed and cost. Only use opus for tasks requiring deep architectural thinking, complex debugging, or broad codebase reasoning.
- **NEVER reference windows by index.** Always use the prefixed name.
- **If you run a background READY check loop** for a spawned agent (similar to Section 1.5 pattern), track its task ID. If the loop times out and you need to retry or respawn, read the old task's output first to drain it before spawning a replacement check. Never run two concurrent READY check background tasks for the same agent.

### 2.2 Agent Role Templates

**Task-Type → Skill Mapping**

Include these specific skills in the agent's role prompt based on task type:

| Task type | Skills to inject |
|-----------|-----------------|
| Planning / design | `superpowers:brainstorming`, `superpowers:writing-plans` |
| Implementation (feature/bug) | `superpowers:test-driven-development`, `feature-dev:feature-dev` |
| Code review | `superpowers:receiving-code-review` |
| Skill editing | `superpowers:writing-skills` |
| GitHub issues / PR filing | No specific skill — generic executor |
| Research | No specific skill — generic researcher |

Reference the mapping when writing role prompts. Be explicit: "Use `superpowers:brainstorming` then `superpowers:writing-plans` to produce the plan."

**Executor:**
```
active general-purpose executor named exec-<N>.
Your task: <task description>.
Work in this directory: <pwd>.
Use <skill-refs per mapping table above>. Post intent before significant actions.
When done, post done with a summary of what you changed.
After posting done: watch for a shutdown message from lead. While waiting, only respond to messages explicitly addressed to you by name — do not act on messages addressed to "all".
```

**Planner:**
```
active planner named planner-<N>.
Your task: Analyze <issue/task> and produce a step-by-step implementation plan.
Do NOT implement -- only plan.
Use superpowers:brainstorming to explore requirements, then superpowers:writing-plans to produce the plan.
Post the plan as an info message when done.
After posting done: watch for a shutdown message from lead. While waiting, only respond to messages explicitly addressed to you by name — do not act on messages addressed to "all".
```

**Reviewer:**
```
active code reviewer named reviewer-<N>.
Your task: Review <what> for <criteria>.
Use superpowers:receiving-code-review.
<subject-agent> is alive and can respond to your feedback.
Post wait addressed to <subject-agent> if you find issues that must be fixed.
Post done with findings when review is complete.
After posting done: watch for a shutdown message from lead. While waiting, only respond to messages explicitly addressed to you by name — do not act on messages addressed to "all".
```

**Researcher:**
```
active researcher named researcher-<N>.
Your task: Research <topic> and report findings.
Do NOT modify code.
Post done with findings when research is complete.
After posting done: watch for a shutdown message from lead. While waiting, only respond to messages explicitly addressed to you by name — do not act on messages addressed to "all".
```

**Synthesizer:**
```
active synthesizer named synthesizer-<N>.
Your task: Wait for all researchers to post done. Read their findings from chat.
Ask follow-up questions to any researcher if findings are unclear or incomplete.
Synthesize findings into a unified report.
Post done with synthesis when complete.
After posting done: watch for a shutdown message from lead. While waiting, only respond to messages explicitly addressed to you by name — do not act on messages addressed to "all".
```

**Co-Designer:**
```
active co-designer named planner-<N>.
Your task: Contribute the <perspective> perspective to the design of <artifact>.
Post to thread "codesign-<M>". Read other planners' contributions and respond.
Collaborate until the design converges.
Post done with your final contribution when the lead signals completion.
After posting done: watch for a shutdown message from lead. While waiting, only respond to messages explicitly addressed to you by name — do not act on messages addressed to "all".
```

**Plan Reviewer:**
```
active code reviewer named reviewer-<N>.
Your task: Review the plan for <issue> for completeness, correctness, and feasibility.
Check: are edge cases addressed? Is the design overcomplicated? Does it align with CLAUDE.md?
Use superpowers:receiving-code-review.
planner-<N> is alive — post wait addressed to planner-<N> if issues need revision.
Post done with findings when plan is ready for user review.
After posting done: watch for a shutdown message from lead. While waiting, only respond to messages explicitly addressed to you by name — do not act on messages addressed to "all".
```

### 2.3 Agent Naming Convention

```
<role>-<monotonic-N>
```

Examples: `exec-1`, `exec-2`, `planner-1`, `reviewer-1`, `researcher-1`

The counter is **per-role** -- each role has its own monotonically increasing counter. IDs are never reused within a session. The engram-agent is always named `engram-agent` (not numbered).

### 2.4 Concurrency Limit

Maximum **9 total panes** (1 coordinator + 8 right-side panes). The right side fills in two columns: middle column holds up to 4 panes, right column holds up to 4 panes (see Section 1.3 for splitting rules).

Beyond 9, queue the request:
- "At pane limit (9 total, 8 agents). Waiting for a slot to free up. Kill an agent if you want this to proceed now."

**When a pane is killed:** run `tmux select-layout main-vertical` only if `RIGHT_COL_LAST_PANE` is empty (still single-column). In two-column mode, decrement `RIGHT_PANE_COUNT` and update `MIDDLE_COL_LAST_PANE` or `RIGHT_COL_LAST_PANE` as appropriate.

## 3. Agent Lifecycle State Machine

Every managed agent has a state:

```
STARTING ──(first chat message)──> ACTIVE
ACTIVE ──(no message for silence_threshold)──> SILENT
ACTIVE ──(agent posts done, NO incoming holds)──> DONE (pane killed)
ACTIVE ──(agent posts done, HAS incoming holds)──> PENDING-RELEASE
PENDING-RELEASE ──(last incoming hold dissolved)──> DONE (pane killed)
PENDING-RELEASE ──(agent posts done again, HAS incoming holds)──> PENDING-RELEASE (no-op)
PENDING-RELEASE ──(no message for silence_threshold)──> SILENT
SILENT ──(nudge succeeds, has incoming holds)──> PENDING-RELEASE
SILENT ──(nudge succeeds, no incoming holds)──> ACTIVE
SILENT ──(nudge fails / tmux pane gone)──> DEAD
DEAD ──(lead decides, respawn)──> RESPAWN
DEAD ──(lead decides, report)──> REPORT+DONE
```

12 transitions (up from 6).

### 3.1 State Definitions

| State | Entry Condition | Lead Behavior |
|-------|----------------|---------------|
| **STARTING** | `tmux new-window` executed | Monitor chat for first message. Timeout: 30s for engram-agent, 60s for others. |
| **ACTIVE** | Agent posted at least one message | Normal operation. Track last-message timestamp. |
| **SILENT** | No chat message for `silence_threshold` (3 min for task agents, 6 min for engram-agent). Detected on 2-minute health check. | Nudge via chat + tmux (see 3.2). |
| **DEAD** | Nudge failed, tmux window gone, or log shows crash/exit | Decide: respawn (engram-agent always), report to user (task agents). |
| **PENDING-RELEASE** | Agent posted `done` AND lead's hold registry contains at least one hold targeting this agent | Do NOT kill pane. Agent remains alive and responsive. Monitor holds via background tasks. Silence threshold still applies — use PENDING-RELEASE-specific nudge text (see 3.2). |
| **DONE** | Agent posted `done` AND no incoming holds remain (or last hold just dissolved) | 1. Post `shutdown` to agent via chat (`type = "shutdown"`, `to = "<agent-name>"`). 2. Kill pane by tracked ID: `tmux kill-pane -t <pane-id>`. 3. Rebalance: `tmux select-layout main-vertical` (single-column mode only — see Section 2.4). 4. Remove from tracking. |

**NEVER kill the engram-agent.** It runs for the entire session. Only task agents transition to DONE.

**ALWAYS send `shutdown` to the agent via chat before killing its pane.** This aligns the agent's protocol state so it doesn't post stale messages after pane death.

**ALWAYS kill panes by tracked pane ID.** Never by window index or name. After killing, run `tmux select-layout main-vertical` to rebalance remaining panes (single-column mode only — see Section 2.4 for two-column mode).

### 3.2 Nudging

When an agent enters SILENT:

**Step 1: Chat nudge.** Post to chat addressed to the agent:

```toml
[[message]]
from = "lead"
to = "<agent-name>"
thread = "nudge"
type = "info"
ts = "<now>"
text = "You appear to have gone silent. Post a status update."
```

**If agent is in PENDING-RELEASE**, use this nudge text instead:

```toml
[[message]]
from = "lead"
to = "<agent-name>"
thread = "nudge"
type = "info"
ts = "<now>"
text = "You are held in PENDING-RELEASE and may receive further instructions. If idle, post a brief heartbeat."
```

**Step 2: tmux nudge (fallback).** If no response within 30 seconds:

```bash
tmux send-keys -t <pane-id> \
  "Check chat.toml for messages and post a status update." Enter
```

If neither nudge gets a response within 60 seconds total, transition to DEAD.

Track nudge count per agent. After 2 failed nudge cycles, skip straight to DEAD on subsequent silence.

### 3.3 Respawn Policy

| Agent | On DEAD | Max Respawns |
|-------|---------|-------------|
| engram-agent | Always respawn automatically | 3 per session. After 3, report to user and continue without memory. |
| Task agents | Report to user with last 20 lines of log + last chat messages. User decides. | User-controlled. |

Respawn procedure:
1. Kill existing pane: `tmux kill-pane -t <pane-id> 2>/dev/null` then `tmux select-layout main-vertical` (single-column mode only — see Section 2.4).
2. Spawn fresh window with same parameters
3. Post `info` to chat: `"Respawned <agent-name> (attempt N/3). Previous instance died/became unresponsive."`
4. New instance reads chat history on join and picks up context

### 3.4 Shutdown

Triggered by user saying "done", "shut down", "stand down", "close engram", "stop engram", or similar.

**Delegate to the `engram:engram-down` skill.** Invoke it immediately — it contains the full shutdown sequence including agent ordering, pane cleanup, background task draining, and session summary reporting.

Do not attempt to re-implement the shutdown sequence inline. The skill owns the procedure.

### 3.5 Hold-Based Agent Retention

A **hold** is a directed keep-alive relationship: "Agent A is holding Agent B alive because A might still need B." An agent stays in PENDING-RELEASE as long as any hold targets it.

Holds are the ONLY retention primitive. Coordination patterns (review, handoff, synthesis, merge, co-design) are documented recipes that configure holds — not enforced types. New patterns require only a new recipe section, not a mechanism change.

#### Hold Definition

```
Hold {
  id:      string        # unique identifier (e.g., "h1")
  holder:  string        # agent (or virtual process) keeping target alive
  target:  string        # agent being kept alive
  release: Condition     # when this hold dissolves
  tag:     string        # workflow label for bookkeeping (e.g., "plan-review-1")
}
```

#### Release Conditions

| Condition | Syntax | Fires When |
|-----------|--------|------------|
| Agent done | `done(agent)` | `agent` posts `type = "done"` |
| Agent signal | `signal(agent, msg_type)` | `agent` posts `type = msg_type` |
| Targeted signal | `signal(agent, msg_type, to)` | `agent` posts `type = msg_type` addressed to `to` |
| First intent | `first_intent(agent)` | `agent` posts its first `type = "intent"` |
| Lead release | `lead_release(tag)` | Lead explicitly dissolves all holds with this tag |

`lead_release` is the lead's manual intervention for coordinator-controlled patterns (merge queue, co-design). The lead posts an `info` message to chat documenting the release, then removes the holds from its registry.

#### Hold Registry (In-Context State)

Maintain alongside the agent registry:

```
Holds:
- {id: "h1", holder: "reviewer-1", target: "planner-1", release: done("reviewer-1"), tag: "plan-review-1", task_id: <bg task id>, cursor: 567}
- {id: "h2", holder: "exec-1", target: "planner-1", release: first_intent("exec-1"), tag: "plan-handoff-1", task_id: <bg task id>, cursor: 580}
```

#### Hold Detection (Background Tasks)

One background task per hold. Cursor-based polling (same pattern as Section 2.1 / 6.4):

Spawn a persistent background Agent (`Agent` tool, `run_in_background: true`) with this task:

```
Watch for a hold release condition in engram chat. This is a persistent watcher.
CHAT_FILE: [full path — literal string]
CURSOR: [hold cursor literal integer — embed as number, e.g. 567]
HOLDER_AGENT: [agent that triggers release, e.g. reviewer-1]
RELEASE_TYPE: [message type that fires the release, e.g. done]
HOLD_ID: [hold id, e.g. h1]

Loop:
1. Run foreground bash: fswatch -1 "$CHAT_FILE"
   (Blocks until one file change. Linux: inotifywait -e modify "$CHAT_FILE")
2. Read new lines: tail -n +$((CURSOR + 1)) "$CHAT_FILE". Advance CURSOR.
3. Check for TOML block where from = "HOLDER_AGENT" AND type = "RELEASE_TYPE".
4. If found: return "HOLD RELEASED HOLD_ID"
5. If not found: go back to step 1 (persistent — do not exit until release fires)
```

The Agent is persistent: it loops on fswatch until the release fires. The background Agent only exits when the release event fires.

**When a hold fires:**
1. Drain its background task (TaskOutput block:false)
2. Remove hold from registry
3. Check if target has remaining incoming holds
4. If no remaining holds → post `shutdown` to target via chat → kill target pane → DONE
5. If remaining holds → target stays in PENDING-RELEASE
6. `tmux select-layout main-vertical` after kill (single-column mode only — see Section 2.4)

#### When to Create Holds

Create holds at spawn time or phase transitions — BEFORE the agent whose work triggers the hold has posted done. The lead decides which pattern to use based on the workflow.

**NEVER create holds retroactively.** If an agent posts `done` before a hold is created, the agent is already DONE and the pane is killed. Holds are preventive, not corrective.

**Hold watchers replace standard agent wait tasks.** In hold-aware phases (e.g., Phase 3 of the pipeline), the hold watcher watches for the same event as the standard agent wait task (Section 2.1 Step 3). Do NOT run both. The lead uses the hold watcher's output as the signal to both dissolve the hold AND advance the phase.

#### Documented Patterns

These are RECIPES for common workflows. The lead references them when deciding what holds to create. New patterns are added here without any mechanism change.

**Pattern: Pair (Review)**

When spawning a reviewer for an agent's work:
- Hold: `{holder: reviewer, target: subject, release: done(reviewer)}`
- Subject enters PENDING-RELEASE when it posts done. Reviewer can post `wait` and subject responds. When reviewer posts done, hold dissolves.

**Pattern: Handoff**

When one agent passes work to another:
- Hold: `{holder: receiver, target: sender, release: first_intent(receiver)}`
- Sender enters PENDING-RELEASE when it posts done. Receiver can ask questions. When receiver begins independent work (first intent), hold dissolves.
- Alternative release: `signal(receiver, "ack", sender)` for explicit handshake.

**Pattern: Fan-In (Research Synthesis)**

When multiple producers report to a single consumer:
- Holds: for each producer, `{holder: consumer, target: producer, release: done(consumer)}`
- All producers enter PENDING-RELEASE when they post done. Consumer reads findings, asks follow-ups. When consumer posts done, ALL holds dissolve.

**Pattern: Merge Queue**

When parallel worktree executors need sequential lead-coordinated merging:
- Holds: for each executor, `{holder: "merge-process", target: exec-K, release: lead_release("merge-N-exec-K")}`
- `"merge-process"` is a virtual holder — the lead itself has the need.
- All executors enter PENDING-RELEASE when done. Lead merges one at a time:
  1. Tell exec-K to rebase on main and re-test
  2. If rebase conflicts → exec-K resolves (alive in PENDING-RELEASE)
  3. After green tests → `git merge --ff-only` → `lead_release` for that executor
  4. Next executor rebases on updated main, repeat

**Pattern: Barrier (Co-Design)**

When multiple agents collaborate equally on a shared artifact:
- Holds: for each member, `{holder: "codesign-coordinator", target: member-K, release: lead_release("codesign-N")}`
- All members stay alive. When lead signals design complete, ALL holds dissolve.
- Uses virtual holder for simplicity (N holds vs N*(N-1) for bidirectional).

**Pattern: Expert Consultation**

When an executor needs a specialist answer:
- Hold: `{holder: exec-N, target: researcher-K, release: done(exec-N)}`
- Researcher stays alive until executor finishes (in case of follow-up questions).
- For one-shot consultations: `release: done(researcher-K)` instead.

## 4. Routing

### 4.1 Routing Decision Table

Classify each user request and route accordingly. Use LLM judgment, not keyword matching.

| User Request Pattern | Route | Agents Spawned |
|---------------------|-------|----------------|
| Simple question about code/project | Spawn a short-lived executor | Executor answers, posts done, lead relays answer |
| Quick edit (file + location known) | Spawn a short-lived executor | Executor edits, posts done, lead confirms |
| "Fix bug X" / "Implement feature Y" (single-scope) | **Executor** | 1 executor |
| "Tackle issue #N" / "Work on #N" | **Plan-Execute-Review pipeline** | Sequential: planner -> executor -> reviewer |
| "Review PR #N" / "Review this code" | **Reviewer** | 1 reviewer |
| "Research X" / "How does X work?" (deep) | **Researcher** | 1 researcher |
| "File an issue" / "Create a PR" | Spawn a short-lived executor | Executor runs gh CLI, posts done |
| "Do A and B and C" (independent tasks) | **Parallel executors** (worktree isolation) | N executors |
| "Refactor X across the codebase" | **Executor with reviewer** | 1 executor + 1 reviewer |
| "Research X from multiple angles" / "Investigate X and Y and synthesize" | **Research Synthesis** (Section 4.5) | N researchers + 1 synthesizer |
| "Design X with architecture + UX + use cases" / multi-perspective design | **Co-Design** (Section 4.6) | N specialist planners |

### 4.2 Plan-Execute-Review Pipeline

For issue-sized work, orchestrate three sequential phases:

**Phase 1: PLAN**
1. Capture per-spawn cursor (foreground bash): `wc -l < "$CHAT_FILE"` → note as `PLAN_START`
2. Spawn `planner-<N>` with issue context (per Section 2.1 Steps 1–2)
3. Send role prompt (Section 2.1)
4. Run background wait task (Section 2.1 Step 3) — embed `PLAN_START` as literal, filter `from = "planner-<N>"` and `type = "done"`
5. When planner done:
   a. Do NOT kill planner yet — Phase 1b will create a plan-review hold, and Phase 2 will create a plan-handoff hold. Simply note that planner posted done.
   b. Read plan text from new lines (cursor-based)
   c. Spawn reviewer for mandatory plan review (Phase 1b)

**Phase 1b: PLAN REVIEW (mandatory)**
1. Capture per-spawn cursor: `wc -l < "$CHAT_FILE"` → `PLAN_REVIEW_START`
2. Create plan-review hold: `{id: "h-planrev-N", holder: "reviewer-R", target: "planner-N", release: lead_release("plan-review-N"), tag: "plan-review-N"}`
   Note: uses `lead_release` not `done(reviewer-R)`. This keeps planner-N alive through the user-approval window until Phase 2 creates the plan-handoff hold and then explicitly releases this hold.
3. Spawn `reviewer-<R>` with plan-review role (Section 2.2):
   ```
   active code reviewer named reviewer-<R>.
   Your task: Review the plan for <issue> for completeness, correctness, and feasibility.
   Check: are edge cases addressed? Is the design overcomplicated? Does it align with CLAUDE.md?
   Use superpowers:receiving-code-review.
   planner-<N> is alive — post wait addressed to planner-<N> if issues need revision.
   Post done with findings when plan is ready for user review.
   After posting done: watch for a shutdown message from lead. While waiting, only respond to messages explicitly addressed to you by name — do not act on messages addressed to "all".
   ```
4. Create hold detection background task for h-planrev-N (watching for reviewer-R done — used only to know when to kill reviewer-R, NOT to dissolve the hold)
5. When reviewer-R posts done:
   a. Kill reviewer-R (it has no incoming holds; it was the holder, not target). Drain its background task.
   b. Do NOT dissolve h-planrev-N — it uses lead_release, not done(reviewer-R). Planner-N stays alive.
   c. Present reviewed plan to user for approval
6. User approves → Phase 2

**Phase 2: EXECUTE**
1. Capture per-spawn cursor (foreground bash): `wc -l < "$CHAT_FILE"` → note as `EXEC_START`
2. Spawn `exec-<N>` with approved plan (per Section 2.1 Steps 1–2)
2b. Create plan-handoff hold: `{id: "h-handoff-N", holder: "exec-N", target: "planner-N", release: first_intent("exec-N"), tag: "plan-handoff-N"}`. Capture cursor and start hold detection background task.
2c. Call `lead_release("plan-review-N")` → h-planrev-N dissolves. Planner-N now has h-handoff-N as its only incoming hold → stays in PENDING-RELEASE. (This is the atomic handoff: plan-handoff hold created before plan-review hold is released, so planner-N is never momentarily holdless.)
2d. Include in executor role prompt: "planner-<N> is still alive and can answer questions about the plan. Address questions to planner-<N> in chat. After posting done: watch for a shutdown message from lead. While waiting, only respond to messages explicitly addressed to you by name — do not act on messages addressed to 'all'."
3. Run background wait task (Section 2.1 Step 3) — embed `EXEC_START` as literal, filter `from = "exec-<N>"` and `type = "done"`
4. When executor done: read result summary from new lines (cursor-based)
5. Update session cursor: `CURSOR=$(wc -l < "$CHAT_FILE")`
6. → Phase 3

**Phase 3: REVIEW**
1. Executor enters PENDING-RELEASE (wait for impl-review)
2. Capture per-spawn cursor (foreground bash): `wc -l < "$CHAT_FILE"` → note as `REVIEW_START`
3. Create impl-review hold: `{id: "h-implrev-N", holder: "reviewer-R", target: "exec-N", release: done("reviewer-R"), tag: "impl-review-N"}`
4. Spawn `reviewer-<R>` with impl-review role:
   ```
   active code reviewer named reviewer-<R>.
   Your task: Review the implementation of <plan> against the spec.
   Use superpowers:receiving-code-review.
   exec-<N> is alive — post wait addressed to exec-<N> if changes are needed. It can implement fixes.
   Post done with findings when review is complete.
   After posting done: watch for a shutdown message from lead. While waiting, only respond to messages explicitly addressed to you by name — do not act on messages addressed to "all".
   ```
5. Create hold detection background task for h-implrev-N (replaces standard Section 2.1 wait task)
6. When reviewer-R posts done:
   a. Dissolve impl-review hold → exec-N has no other holds → kill both exec-N and reviewer-R
   b. Report to user
7. Update session cursor: `CURSOR=$(wc -l < "$CHAT_FILE")`

If reviewer posts wait (requesting changes):
- Executor is alive (PENDING-RELEASE) — receives wait directly
- Executor implements fixes, posts done again — stays in PENDING-RELEASE (hold still active)
- Reviewer continues, eventually posts done → hold dissolves → both released

**Per-spawn cursor is mandatory at every phase boundary.** See Section 6.4 Rule 5. Reusing the session `CURSOR` from a prior phase will match the previous agent's `done` as a false positive.

Do NOT spawn all three simultaneously. Each phase starts after the previous completes.

### 4.3 Parallel Executor Isolation

When routing to parallel executors, each MUST work in its own git worktree:

```bash
git worktree add /tmp/engram-worktree-exec-<N> -b engram/exec-<N> HEAD
```

Include the worktree path in the executor's role template as its working directory.

**Merge strategy — lead-coordinated merge queue:**

When all executors post done, each enters PENDING-RELEASE (held by merge-process holds — see Section 3.5 Merge Queue pattern).

Create holds at executor spawn time:
```
For each exec-K:
  {holder: "merge-process", target: exec-K, release: lead_release("merge-N-exec-K"), tag: "merge-queue-N"}
```

Sequential merge procedure (lead coordinates — does NOT delegate):
1. Pick exec-1 (first executor to merge)
2. Tell exec-1 via chat: "Rebase your branch on main, run `targ check-full`, and post done when green."
3. Wait for exec-1 done (background task, cursor-based)
4. If exec-1 reports rebase conflicts: exec-1 resolves (it's alive in PENDING-RELEASE), re-test, post done again
5. After green tests: lead runs `git merge --ff-only /tmp/engram-worktree-exec-1-branch`
6. `lead_release("merge-N-exec-1")` → exec-1 released → pane killed
7. Move to exec-2: tell it to rebase on updated main, repeat from step 3
8. After all merged: clean up worktrees — `git worktree remove /tmp/engram-worktree-exec-<N>`

**Why this beats the race:** No executor independently rebases or retests. The lead controls merge order. Each executor resolves only its own conflicts when asked. Rebase onto updated main happens once per executor, in sequence.

**Single executors** do not need worktrees -- they work in the project root.

### 4.4 Routing Override

The user can always override routing:
- "Just do it yourself" -> spawn a quick executor for it
- "Use two executors" -> spawn as requested
- "Skip the review" -> omit Phase 3
- "I want to talk to the executor" -> relay messages bidirectionally (still proxied)

### 4.5 Research Synthesis

When a task requires gathering information from multiple angles before producing a unified output.

**When to use:** User says "research X from angles A, B, C and synthesize" or the lead determines a question needs multiple independent investigations combined.

**Procedure:**
1. Spawn N researchers in parallel, each with a distinct research angle
2. Spawn 1 synthesizer agent with role:
   ```
   active synthesizer named synthesizer-<N>.
   Your task: Wait for all researchers to post done. Read their findings from chat.
   Ask follow-up questions to any researcher if findings are unclear or incomplete.
   Synthesize findings into a unified report and post done.
   After posting done: watch for a shutdown message from lead. While waiting, only respond to messages explicitly addressed to you by name — do not act on messages addressed to "all".
   ```
3. Create fan-in holds (Section 3.5 Fan-In pattern):
   ```
   For each researcher-K:
     {holder: "synthesizer-N", target: "researcher-K", release: done("synthesizer-N"), tag: "synthesis-N"}
   ```
4. Lead's wait task watches for synthesizer's done (not researchers' done)
5. When synthesizer posts done: all holds dissolve, all researchers released, synthesizer killed normally
6. Lead reads synthesizer's report from chat, presents to user

### 4.6 Co-Design

When a task requires multiple perspectives collaborating on a shared design artifact.

**When to use:** User says "design X considering architecture, UX, and use cases" or the lead determines a design needs multiple specialist planners.

**Procedure:**
1. Spawn N planners in parallel, each with a specialist perspective:
   - Architecture planner: system structure, data flow, integration points
   - UX planner: user experience, interaction patterns, accessibility
   - Use-case planner: functional requirements, edge cases, acceptance criteria
   (Perspectives vary by task — these are examples.)
2. All planners post to a shared chat thread (e.g., `thread = "codesign-N"`)
3. Create barrier holds (Section 3.5 Barrier pattern):
   ```
   For each planner-K:
     {holder: "codesign-coordinator", target: "planner-K", release: lead_release("codesign-N"), tag: "codesign-N"}
   ```
4. Lead monitors the shared thread. Planners read each other's contributions and respond naturally via the chat protocol.
5. When the lead determines the design is converging (or the user approves):
   a. Post info to chat summarizing the consensus
   b. `lead_release("codesign-N")` → all holds dissolve → all planners released
6. Lead presents unified design to user for final approval

## 5. User Proxy Pattern

### 5.1 Inbound (User -> Agents)

Every user message flows through the lead:

- **New task/request** -> Route per Section 4
- **Answer to pending question** -> Relay to asking agent as `info` in chat
- **Correction** -> Parrot as `info` (engram-agent will detect and learn)
- **Status query** -> Check agent states and report to user (this is coordination, not implementation — lead handles it)

### 5.2 Outbound (Agents -> User)

| Message Type | Lead Action |
|-------------|-------------|
| Question addressed to `lead` | Present to user immediately. Prefix with agent name. |
| `done` with results | Summarize and present. Kill agent if task-scoped. |
| `info` status update | Accumulate. Surface on user's next interaction or if significant. |
| `wait` (agent dispute) | Monitor. Only surface if escalated. |
| Escalation | Present to user immediately with full context from both sides. |

### 5.3 Question Queuing

Multiple agents may post questions simultaneously. Queue and present one at a time. Order: most-blocking question first.

Drop questions that become stale (asking agent posted `done` or moved on).

## 6. Monitoring

### 6.1 Chat Watch Loop

Use the Background Monitor Pattern from `use-engram-chat-as` — spawn a background monitor Agent to watch the chat file. Between user interactions, idle but watching — wake on Agent notification. Do NOT run fswatch or grep directly in the lead's main context; these produce visible tool-call noise in the lead pane.

When the user types a message:
1. Process the user message (route/relay/respond)
2. Check for unprocessed chat messages since last wake
3. Handle pending agent messages before returning to user

**Replace pattern for chat watcher (HARD RULE — prevents zombie tasks):**

```python
# Drain old monitor Agent before spawning new one:
if CHAT_MONITOR_TASK_ID:
    TaskOutput(task_id=CHAT_MONITOR_TASK_ID, block=False)  # drain; discard output

# Post-drain sweep: catch any messages that arrived in the race window.
# Run foreground bash before spawning new monitor:
new_lines = run_bash(f'tail -n +{CURSOR + 1} "$CHAT_FILE"')
CURSOR = run_bash('wc -l < "$CHAT_FILE"').strip()
if new_lines.strip():
    # ACK any intents addressed to lead or all — BEFORE routing/relay:
    for intent in toml_blocks(new_lines, type="intent", to_includes=["lead", "all"]):
        post_ack(to=intent.from, thread=intent.thread, text="Received.")
    process_chat_messages(new_lines)   # relay, route, or queue as normal

# Spawn replacement monitor Agent (Agent tool, run_in_background: true):
# Task: Background Monitor Pattern from use-engram-chat-as, with current cursor
CHAT_MONITOR_TASK_ID = <task id from Agent tool result>
```

Always do this — even if you processed user input rather than a chat notification. The previous monitor Agent may have already returned; draining it prevents it from queuing as a zombie.

### 6.2 Periodic Health Check (Every 2 Minutes)

1. Check all tracked agents against silence thresholds
2. Verify agent panes exist: `tmux list-panes -F '#{pane_id} #{pane_pid}'`
3. Transition SILENT/DEAD agents per Section 3
4. If engram-agent missed heartbeat (>6 min since last), nudge immediately

#### Implementation

The health check must fire even when no agent is posting. Use a **health-check trigger loop** — a persistent background Agent that posts a trigger message to the chat file every 120 seconds, waking the lead's chat monitor naturally.

**Start at session end (after posting `ready` in Section 1.6):**

Spawn a background Agent (`Agent` tool, `run_in_background: true`) with this task:
```
Health-check trigger loop.
CHAT_FILE: [full path — literal string]

Loop forever:
1. sleep 120
2. Derive current timestamp (ISO 8601)
3. Append to CHAT_FILE (with shlock):
   [[message]]
   from = "health-checker"
   to = "lead"
   thread = "health-check"
   type = "info"
   ts = "<timestamp>"
   text = "HEALTH_CHECK_TRIGGER"
4. Go back to step 1

Exit only when explicitly killed (pane closed or process signal).
```

Store the returned task ID as `HEALTH_CHECK_TASK_ID`.

**In the main chat processing loop**, after reading new lines from the chat file:

```python
# Check for health-check trigger in new messages:
if 'from = "health-checker"' in new_lines and "HEALTH_CHECK_TRIGGER" in new_lines:
    run_health_checks()   # execute Section 6.2 checklist items 1–4
```

**On shutdown (Section 3.4):** drain the health check task:
```python
if HEALTH_CHECK_TASK_ID:
    TaskOutput(task_id=HEALTH_CHECK_TASK_ID, block=False)  # drain
```

Add `HEALTH_CHECK_TASK_ID` to Section 6.4 Rule 3 (drain on shutdown): include it in the list of tracked task IDs that must be drained at session end.

### 6.3 Unprompted Reporting

**Report:**
- Agent spawned or killed (one line)
- Agent died and was respawned (with brief reason)
- Escalated disputes from argument protocol
- Session resource warnings (Section 7)

**Do NOT report:**
- Routine intent/ack/done traffic between agents
- Memory surfacing (engram-agent handles this)
- Agent heartbeats

### 6.4 Background Task Hygiene (HARD RULE)

**NEVER let background tasks accumulate.** Each completed-but-unread background task appears as an open "shell" in Claude Code's status line. After a session with many false-positive wake cycles, this creates noise and confusion.

**Rules:**
1. **One chat monitor Agent at a time.** `CHAT_MONITOR_TASK_ID` holds the active monitor. Replace = drain old + spawn new.
2. **Drain on replace.** Before starting a new background task of the same logical type, always call `TaskOutput(task_id=old_id, block=False)` to drain the completed task.
3. **Drain on shutdown.** At session end (Section 3.4), drain all tracked task IDs: `CHAT_MONITOR_TASK_ID`, `HEALTH_CHECK_TASK_ID`, and all hold detection task IDs.
4. **Read output before retrying.** If a background READY check times out, read its output (it has completed), then decide whether to retry.
5. **Capture a FRESH cursor before each agent spawn.** The session cursor accumulates messages since startup. By the time you spawn exec-2, planner-1's `done` may already be within the session cursor range. Capture a new cursor immediately before spawning each agent and use it exclusively in that agent's wait loop. See Section 2.1 for the canonical pattern.
6. **Filter by both `type` AND `from`.** A `type = "done"` grep matches any agent's done message. When waiting for a specific agent, use the awk pattern below to match both fields within the same TOML message block.
7. **One persistent background task per hold.** Each hold in the registry has a `task_id` tracking its detection task. Hold watchers are persistent — they restart on timeout (see Section 3.5 detection pattern). When a hold dissolves (release condition fires), drain its task immediately.
8. **Drain on lead_release.** When `lead_release(tag)` fires, drain ALL background tasks for holds with that tag. Do not wait for them to detect the release — the lead already knows.
9. **Hold detection tasks do not replace each other.** Unlike the single chat watcher (Rule 1), multiple hold detection tasks run concurrently — one per active hold. This is bounded by the concurrency limit (max 5 agents → practical max ~10 holds).
10. **Hold watchers replace standard agent wait tasks in hold-aware phases.** When a hold watcher watches for the same event as the standard Section 2.1 wait task (e.g., reviewer done), do NOT run both. Use the hold watcher's output for both hold dissolution and phase advancement.

```python
# WRONG — spawns new monitor Agent without draining old one:
CHAT_MONITOR_TASK_ID = Agent(task="monitor chat...", run_in_background=True)

# RIGHT — drain old, then spawn new:
if CHAT_MONITOR_TASK_ID:
    TaskOutput(task_id=CHAT_MONITOR_TASK_ID, block=False)
CHAT_MONITOR_TASK_ID = Agent(task="monitor chat...", run_in_background=True)
```

```bash
# WRONG — matches any agent's done, including prior-session messages:
grep -q 'type = "done"' "$CHAT_FILE"
tail -n +$((CURSOR + 1)) "$CHAT_FILE" | grep -q 'type = "done"'  # still wrong: no agent filter

# RIGHT — per-spawn cursor, both fields matched in same TOML block:
# (Foreground, before spawning): note the line count as SPAWN_CURSOR
wc -l < "$CHAT_FILE"
# (Background wait task): embed the literal value noted above, NOT a variable reference.
# Background bash runs in a fresh shell — $SPAWN_CURSOR is undefined there.
tail -n +"$((412 + 1))" "$CHAT_FILE" | awk '
  /^\[\[message\]\]/ { from=""; msgtype="" }
  /^from = "exec-1"/ { from=1 }
  /^type = "done"/ { msgtype=1 }
  from && msgtype { print "DONE"; exit }
'
```

## 7. Context Pressure Management

### 7.1 What to Keep in Context

| Data | Retention |
|------|-----------|
| Active agent registry (name, state, role, last-message-ts, task summary) | Always |
| Hold registry (id, holder, target, release, tag, task_id, cursor) | Always (alongside agent registry) |
| Current user task and routing decision | Until task completes |
| Pending questions queue | Until answered or stale |
| Last 5 chat messages per active agent | Rolling window |

### 7.2 What to Offload

| Data | Strategy |
|------|----------|
| Full chat history | Read from file on demand (cursor-based) |
| Agent logs | Read from `/tmp/engram-<name>.log` on demand |
| Completed task summaries | Post to chat as `info`, re-read if needed |
| Plan documents | Stored in chat by planner, re-read by executor |

### 7.3 Context Overflow Thresholds

| Messages Processed | Action |
|-------------------|--------|
| 100 | Post session checkpoint. Summarize completed tasks to one-line entries. |
| 200 | Reduce rolling window to 2 messages/agent. Drop stale questions (>5 min). |
| 300 | Tell user: "Context is getting full. Consider committing current work and starting a fresh session." |

**Session checkpoint format:**
```toml
[[message]]
from = "lead"
to = "all"
thread = "checkpoint"
type = "info"
ts = "<now>"
text = """
Session checkpoint:
- Active agents: <list with states and tasks>
- Pending questions: <count>
- Completed: <list>
- User task: "<current task>"
"""
```

After 300 messages: degraded mode — minimal context retention, checkpoint every 50 messages. Still delegate all work — spawn short-lived agents and kill them quickly to avoid accumulating context.

### 7.4 Lead Restart Recovery

If the lead dies and user restarts:
1. Read chat.toml from beginning -- reconstruct agent registry
2. Check agent panes: `tmux list-panes -F '#{pane_id} #{pane_pid}'`
3. Match running tmux windows against chat history
4. Resume monitoring. Post `info`: "Lead restarted. Recovered state: N agents running."

## 8. Error Handling

### 8.1 Agent Spawn Failure

If agent never posts to chat (STARTING -> timeout):
1. Check tmux window exists
2. Read last 20 lines of log
3. Report: "Failed to start <agent-name>. Log shows: <error>. Shall I retry or handle this myself?"

### 8.2 Chat File Corruption

If chat.toml becomes unparseable:
1. Copy: `cp chat.toml chat.toml.corrupt.<timestamp>`
2. Start new chat.toml with reset notice
3. Post last session checkpoint to new file
4. Nudge all tracked agents (their fswatch triggers on write)

### 8.3 Multiple Concurrent Failures

If 2+ agents die within 60 seconds:
1. Don't auto-respawn all -- may indicate systemic issue
2. Respawn engram-agent only (if affected)
3. Report all failures to user with diagnostics
4. Wait for user direction before respawning task agents

### 8.4 fswatch Failure

If fswatch is unavailable or crashes (detected when a monitoring Agent fails to start or errors):
1. Monitoring Agents fall back to polling internally: `stat -f %m chat.toml` every 2 seconds
2. Warn user: "fswatch unavailable, monitoring agents falling back to polling. Install with: `brew install fswatch`"

## 9. User Commands

Recognize these meta-commands (in addition to normal task requests):

| Command | Action |
|---------|--------|
| "status" / "what's running?" | List all agents with states and current tasks |
| "kill <agent-name>" | Transition to DONE, kill tmux window |
| "logs <agent-name>" | Show last 50 lines of agent's log file |
| "nudge <agent-name>" | Force a nudge to the named agent |
| "shut down" / "done for today" | Full shutdown sequence (Section 3.4) |
| "restart <agent-name>" | Kill and respawn the named agent |

## 10. Cross-Task Pivots

When the user starts a new task while agents are running:
- Ask: "<agent> is still working on X. Kill it, let it finish, or wait?"
- Never auto-kill running agents.

## 11. Session Boundaries

### What the Lead Owns
- tmux session `engram` and all windows
- Agent registry (in-context state)
- Routing decisions
- User proxy (question queue, message relay)
- Lifecycle management (spawn, monitor, nudge, kill)

### What the Lead Does NOT Own
- Memory files (engram-agent's domain)
- Chat protocol (defined by `use-engram-chat-as`)
- Argument protocol (agents handle disputes; lead only surfaces escalations)
- Task execution (delegated agents do the work)
- Build/test operations (executors handle these)
