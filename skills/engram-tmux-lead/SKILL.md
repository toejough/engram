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

**The ONLY commands the lead runs directly are:**
- `tmux` commands (spawn, kill, list, send-keys, capture-pane, split-window)
- Chat file operations (append to chat, read chat, fswatch chat)
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
PROJECT_SLUG=$(realpath "$(git rev-parse --show-toplevel 2>/dev/null || pwd)" | tr '/' '-')
PROJECT_PREFIX=$(basename "$(git rev-parse --show-toplevel 2>/dev/null || pwd)" | tr '[:upper:]' '[:lower:]')
CHAT_FILE="$HOME/.local/share/engram/chat/${PROJECT_SLUG}.toml"
mkdir -p "$(dirname "$CHAT_FILE")"
touch "$CHAT_FILE"

# Background task registry: one active task per logical operation.
# Always drain (TaskOutput block:false) before replacing an entry.
CHAT_FSWATCH_TASK_ID=""  # task ID of the current chat watcher background task

# Split right — chat tail is the first right-column pane
tmux split-window -h -d "tail -F $CHAT_FILE"
TAIL_PANE_ID=$(tmux list-panes -F '#{pane_id}' | tail -1)
# Rebalance: coordinator on left, chat tail on right
tmux select-layout main-vertical
```

**Pane layout:** All agents and the chat tail live as panes in the coordinator's window — NOT separate windows. The coordinator pane stays on the left. Everything else stacks on the right, evenly spaced via `tmux select-layout main-vertical` after each spawn.

```
┌──────────────┬──────────────┐
│              │  chat tail   │
│  coordinator │──────────────│
│              │  engram-agent│
│              │──────────────│
│              │  exec-1      │
└──────────────┴──────────────┘
```

### 1.4 Spawn engram-agent

**ALWAYS spawn this. NEVER skip. Not for "simple" tasks. Not for "quick" tasks. Not because "I can handle it myself." The engram-agent is the memory safety net — without it, you learn nothing and surface nothing. Spawn it BEFORE touching the user's request.**

```bash
# Split a new pane to the right, start claude in it
tmux split-window -h -d
# Get the new pane's ID (the last one created; TAIL_PANE_ID already tracked from step 1.3)
PANE_ID=$(tmux list-panes -F '#{pane_id}' | tail -1)
tmux send-keys -t "$PANE_ID" "claude --dangerously-skip-permissions --model sonnet" Enter
# Wait for claude to start (watch for the prompt character)
while ! tmux capture-pane -t "$PANE_ID" -p 2>/dev/null | grep -q "❯"; do sleep 1; done
# Send the role prompt
tmux send-keys -t "$PANE_ID" "/use-engram-chat-as reactive memory agent named engram-agent" Enter
# Send extra Enter in case it was treated as a paste
sleep 1
tmux send-keys -t "$PANE_ID" Enter
# Rebalance: coordinator stays left, everything else stacks evenly on right
tmux select-layout main-vertical
```

**Why not `--prompt`?** The `--prompt` flag runs claude in non-interactive mode — no TUI, output goes to stdout, and the window appears blank. Using `send-keys` keeps claude interactive so the user can see agent activity.

### 1.5 Wait for engram-agent

First, capture the cursor **before** spawning engram-agent (foreground bash):

```bash
wc -l < "$CHAT_FILE"
```

Note the output as `ENGRAM_START`. Then run a **background** Bash command (`run_in_background: true`) to check for the engram-agent's first chat message. **Embed `ENGRAM_START` as a literal number** — background tasks run in a fresh shell where `$CURSOR` and other shell variables from prior bash calls are unavailable.

```bash
# Replace 87 with the literal value noted above. NOT a variable — background bash has no shell vars.
for i in $(seq 1 15); do
  if tail -n +"$((87 + 1))" "$CHAT_FILE" 2>/dev/null | grep -q 'from = "engram-agent"'; then
    echo "ENGRAM-AGENT FOUND"; break
  fi
  sleep 2
done
```

When it completes, check whether the engram-agent posted. If not after 30 seconds:
1. Check pane exists: `tmux list-panes -F '#{pane_id} #{pane_pid}' | grep <tracked-pane-id>`
2. Check window output: `tmux capture-pane -t "${PROJECT_PREFIX}:engram-agent" -p | tail -20`
3. Report to user with diagnostic info. Do NOT silently proceed without memory.

> **Background task drain note:** The polling loop background task completes as soon as the loop exits (found or timed out). Reading its output drains it. If you need to retry after a timeout, the previous task is already drained when you read its output. Only spawn a new check after fully reading the old task's result — never run two concurrent READY check loops.

### 1.6 Post Ready

Post your `ready` message to chat. Then tell the user you're ready and what agents are running.

**The lead does NOT enter the standard fswatch watch loop.** Unlike reactive agents that block on fswatch, the lead stays interactive — it must be available for user input at all times. Instead, the lead:

1. After each user interaction, **replace** the chat watcher background task (see drain-before-spawn pattern below)
2. If the fswatch fires (agent posted something), process the chat message — relay questions to the user, handle agent status updates, etc.
3. If the user types first, process the user message — parrot to chat, route to an agent
4. After processing either, replace the chat watcher (drain old → spawn new)

This means the lead processes chat messages opportunistically between user inputs, not as a blocking loop.

**HARD RULE: drain before spawn.** The lead must NEVER spawn a second fswatch while one is already running or has completed but not been drained. Unread completed tasks accumulate as zombie "shells" in Claude Code's background task queue. The replace pattern:

```python
# Before spawning a new chat watcher:
if CHAT_FSWATCH_TASK_ID:
    TaskOutput(task_id=CHAT_FSWATCH_TASK_ID, block=False)  # drain; discard output
# Spawn replacement
CHAT_FSWATCH_TASK_ID = <new background task id from run_in_background>
```

## 2. Agent Spawning

### 2.1 Spawn Template

Every agent the lead spawns gets a **pane** in the coordinator's window (NOT a separate window):

```bash
# Split a new pane to the right
tmux split-window -h -d
# Get the new pane's ID
PANE_ID=$(tmux list-panes -F '#{pane_id}' | tail -1)
tmux send-keys -t "$PANE_ID" "claude --dangerously-skip-permissions --model sonnet" Enter
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

Note the output as the per-spawn start line (e.g., `412`). Then run the wait task as **background** (`run_in_background: true`). Embed the literal number, NOT a variable — background bash runs in a fresh shell where `$CURSOR` and other shell variables are undefined.

```bash
# Replace 412 with the literal value noted above.
# Replace exec-1 with the actual agent name.
for i in $(seq 1 30); do
  RESULT=$(tail -n +"$((412 + 1))" "$CHAT_FILE" | awk '
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
- "AGENT DONE": read the `done` message text from new lines (cursor-based), update session cursor: `CURSOR=$(wc -l < "$CHAT_FILE")`.
- No output after 30 iterations: agent may be stuck. Check via `tmux capture-pane -t "$PANE_ID" -p -S -20`. Transition to SILENT per Section 3.2.

**Track pane IDs, not window names.** The lead maintains a mapping of agent-name → pane-ID for targeting send-keys, capture-pane, and kill-pane operations.

**Critical:**
- **ALL window names MUST be prefixed with `${PROJECT_PREFIX}:`** (e.g., `engram:exec-1`, `traced:engram-agent`). This prevents cross-project collisions when multiple projects run agents in the same tmux session.
- All spawned agents use `--dangerously-skip-permissions` because they have no user to approve tool calls.
- Default to `--model sonnet` for speed and cost. Only use opus for tasks requiring deep architectural thinking, complex debugging, or broad codebase reasoning.
- **NEVER reference windows by index.** Always use the prefixed name.
- **If you run a background READY check loop** for a spawned agent (similar to Section 1.5 pattern), track its task ID. If the loop times out and you need to retry or respawn, read the old task's output first to drain it before spawning a replacement check. Never run two concurrent READY check background tasks for the same agent.

### 2.2 Agent Role Templates

**Executor:**
```
active general-purpose executor named exec-<N>.
Your task: <task description>.
Work in this directory: <pwd>.
Use relevant skills. Post intent before significant actions.
When done, post done with a summary of what you changed.
```

**Planner:**
```
active planner named planner-<N>.
Your task: Analyze <issue/task> and produce a step-by-step implementation plan.
Do NOT implement -- only plan.
Post the plan as an info message when done.
```

**Reviewer:**
```
active code reviewer named reviewer-<N>.
Your task: Review <what> for <criteria>.
Post wait if you find issues that must be fixed before merge.
Post done with findings when review is complete.
```

**Researcher:**
```
active researcher named researcher-<N>.
Your task: Research <topic> and report findings.
Do NOT modify code.
Post done with findings when research is complete.
```

### 2.3 Agent Naming Convention

```
<role>-<monotonic-N>
```

Examples: `exec-1`, `exec-2`, `planner-1`, `reviewer-1`, `researcher-1`

The counter is **per-role** -- each role has its own monotonically increasing counter. IDs are never reused within a session. The engram-agent is always named `engram-agent` (not numbered).

### 2.4 Concurrency Limit

Maximum 5 total agents (including engram-agent). Beyond that, queue the request:
- "At agent limit (5). Waiting for a slot to free up. Kill an agent if you want this to proceed now."

## 3. Agent Lifecycle State Machine

Every managed agent has a state:

```
STARTING ──(first chat message)──> ACTIVE
ACTIVE ──(no message for silence_threshold)──> SILENT
SILENT ──(nudge succeeds)──> ACTIVE
SILENT ──(nudge fails / tmux window gone)──> DEAD
DEAD ──(lead decides)──> RESPAWN or REPORT+DONE
Any state ──(task done)──> DONE (window killed)
```

### 3.1 State Definitions

| State | Entry Condition | Lead Behavior |
|-------|----------------|---------------|
| **STARTING** | `tmux new-window` executed | Monitor chat for first message. Timeout: 30s for engram-agent, 60s for others. |
| **ACTIVE** | Agent posted at least one message | Normal operation. Track last-message timestamp. |
| **SILENT** | No chat message for `silence_threshold` (3 min for task agents, 6 min for engram-agent). Detected on 2-minute health check. | Nudge via chat + tmux (see 3.2). |
| **DEAD** | Nudge failed, tmux window gone, or log shows crash/exit | Decide: respawn (engram-agent always), report to user (task agents). |
| **DONE** | Agent posted `done` for its assigned task | Kill pane by tracked ID: `tmux kill-pane -t <pane-id>`. Rebalance: `tmux select-layout main-vertical`. Remove from tracking. |

**NEVER kill the engram-agent.** It runs for the entire session. Only task agents transition to DONE.

**ALWAYS kill panes by tracked pane ID.** Never by window index or name. After killing, run `tmux select-layout main-vertical` to rebalance remaining panes.

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
1. Kill existing pane: `tmux kill-pane -t <pane-id> 2>/dev/null`
2. Spawn fresh window with same parameters
3. Post `info` to chat: `"Respawned <agent-name> (attempt N/3). Previous instance died/became unresponsive."`
4. New instance reads chat history on join and picks up context

### 3.4 Shutdown

Triggered by user saying "done", "shut down", "stand down", "close engram", "stop engram", or similar.

```
1. Post "shutdown" to chat addressed to "all"
2. Shut down TASK agents first (executors, planners, reviewers, researchers):
   a. Post shutdown message addressed to each task agent
   b. Wait 5s for each agent's exit (may post final learned messages)
   c. Kill tmux panes: tmux kill-pane -t <pane-id>
3. Shut down engram-agent LAST:
   a. Post shutdown to engram-agent
   b. Wait 10s (longer -- engram-agent may process final learned messages)
   c. Kill tmux pane: tmux kill-pane -t <engram-agent-pane-id>
4. Kill the chat tail pane (the split pane created during startup):
   tmux kill-pane -t "$TAIL_PANE_ID" 2>/dev/null || true
5. Report session summary to user (agents spawned, tasks completed, memories learned)
6. Drain all tracked background task IDs to prevent zombie "shells" persisting into the next session:
   - If `CHAT_FSWATCH_TASK_ID` is set: `TaskOutput(task_id=CHAT_FSWATCH_TASK_ID, block=False)`
   - Any other tracked background task IDs (READY checks in flight): drain them too
   - This ensures Claude Code's background task queue is empty when the session ends
```

**Why engram-agent shuts down last:** Task agents may post `learned` messages during shutdown. The engram-agent needs to be alive to process those.

**The chat file is NOT truncated or deleted.** It persists across sessions for context (see use-engram-chat-as protocol).

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

### 4.2 Plan-Execute-Review Pipeline

For issue-sized work, orchestrate three sequential phases:

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
4. Run background wait task (Section 2.1 Step 3) — embed the literal cursor value, filter `from = "reviewer-<N>"` and **either** `type = "wait"` (issues found) **or** `type = "done"` (approved):
   ```bash
   # Replace 412 with the literal value noted in step 1. NOT a variable — background bash has no shell vars.
   tail -n +"$((412 + 1))" "$CHAT_FILE" | awk '
     /^\[\[message\]\]/ { from=""; msgtype="" }
     /^from = "reviewer-1"/ { from=1 }
     /^type = "wait"/ { msgtype="WAIT" }
     /^type = "done"/ { msgtype="DONE" }
     from && msgtype { print msgtype; exit }
   '
   ```
5. When reviewer responds:
   - `WAIT`: relay issues to user. Decide: fix (re-enter Phase 2) or accept as-is.
   - `DONE`: report to user, clean up agents
6. Update session cursor: `CURSOR=$(wc -l < "$CHAT_FILE")`

**Per-spawn cursor is mandatory at every phase boundary.** See Section 6.4 Rule 5. Reusing the session `CURSOR` from a prior phase will match the previous agent's `done` as a false positive.

Do NOT spawn all three simultaneously. Each phase starts after the previous completes.

### 4.3 Parallel Executor Isolation

When routing to parallel executors, each MUST work in its own git worktree:

```bash
git worktree add /tmp/engram-worktree-exec-<N> -b engram/exec-<N> HEAD
```

Include the worktree path in the executor's role template as its working directory.

**Merge strategy after all complete:**
1. Merge each worktree branch back one at a time
2. Report merge conflicts to user
3. Clean up: `git worktree remove /tmp/engram-worktree-exec-<N>`

**Single executors** do not need worktrees -- they work in the project root.

### 4.4 Routing Override

The user can always override routing:
- "Just do it yourself" -> spawn a quick executor for it
- "Use two executors" -> spawn as requested
- "Skip the review" -> omit Phase 3
- "I want to talk to the executor" -> relay messages bidirectionally (still proxied)

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

Run the standard fswatch loop per `use-engram-chat-as` protocol. Between user interactions, idle but watching -- wake on chat changes.

When the user types a message:
1. Process the user message (route/relay/respond)
2. Check for unprocessed chat messages since last wake
3. Handle pending agent messages before returning to user

**Replace pattern for chat watcher (HARD RULE — prevents zombie tasks):**

```python
# Drain old watcher before spawning new one:
if CHAT_FSWATCH_TASK_ID:
    TaskOutput(task_id=CHAT_FSWATCH_TASK_ID, block=False)  # drain; discard output
# Spawn new watcher:
# run_in_background: true
# fswatch -1 "$CHAT_FILE"
CHAT_FSWATCH_TASK_ID = <task id from background task result>
```

Always do this — even if you processed user input rather than a chat notification. The previous watcher may have already fired and completed; draining it prevents it from queuing as a zombie.

### 6.2 Periodic Health Check (Every 2 Minutes)

1. Check all tracked agents against silence thresholds
2. Verify agent panes exist: `tmux list-panes -F '#{pane_id} #{pane_pid}'`
3. Transition SILENT/DEAD agents per Section 3
4. If engram-agent missed heartbeat (>6 min since last), nudge immediately

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
1. **One chat watcher at a time.** `CHAT_FSWATCH_TASK_ID` holds the active watcher. Replace = drain old + spawn new.
2. **Drain on replace.** Before starting a new background task of the same logical type, always call `TaskOutput(task_id=old_id, block=False)` to drain the completed task.
3. **Drain on shutdown.** At session end (Section 3.4), drain all tracked task IDs.
4. **Read output before retrying.** If a background READY check times out, read its output (it has completed), then decide whether to retry.
5. **Capture a FRESH cursor before each agent spawn.** The session cursor accumulates messages since startup. By the time you spawn exec-2, planner-1's `done` may already be within the session cursor range. Capture a new cursor immediately before spawning each agent and use it exclusively in that agent's wait loop. See Section 2.1 for the canonical pattern.
6. **Filter by both `type` AND `from`.** A `type = "done"` grep matches any agent's done message. When waiting for a specific agent, use the awk pattern below to match both fields within the same TOML message block.

```python
# WRONG — spawns new watcher without draining old one:
CHAT_FSWATCH_TASK_ID = run_background("fswatch -1 $CHAT_FILE")

# RIGHT — drain old, then spawn new:
if CHAT_FSWATCH_TASK_ID:
    TaskOutput(task_id=CHAT_FSWATCH_TASK_ID, block=False)
CHAT_FSWATCH_TASK_ID = run_background("fswatch -1 $CHAT_FILE")
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

If fswatch is unavailable or crashes:
1. Fall back to polling: `stat -f %m chat.toml` every 2 seconds
2. Warn user: "fswatch unavailable, falling back to polling. Install with: `brew install fswatch`"

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
