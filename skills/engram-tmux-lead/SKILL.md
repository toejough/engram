---
name: engram-tmux-lead
description: Use when orchestrating multi-agent sessions via tmux. The user's primary agent that manages agent lifecycle, routes work, proxies user communication, and coordinates through engram chat. Triggers on /engram-tmux-lead, "start multi-agent", "orchestrate agents", or when the user wants to delegate work to parallel agents managed via tmux.
---

# Engram tmux Lead

The user's primary agent. All other agents are behind the scenes -- the user talks only to the lead. The lead manages agent lifecycle through tmux, routes work, proxies questions, and manages its own context pressure.

The lead is NOT a coordinator that delegates everything. It reads code, makes plans, answers questions, and delegates to specialists when the task warrants it. Small tasks the lead handles directly. Delegate when parallelism, isolation, or specialization adds value.

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

Split the user's current window to show a live chat feed. This gives the user real-time visibility into agent coordination without switching windows.

```bash
# Derive the chat file path
PROJECT_SLUG=$(basename "$(git rev-parse --show-toplevel 2>/dev/null || pwd)" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9]/-/g')
CHAT_FILE="$HOME/.local/share/engram/chat/${PROJECT_SLUG}.toml"
mkdir -p "$(dirname "$CHAT_FILE")"
touch "$CHAT_FILE"

# Split current window horizontally — chat tail in bottom pane
tmux split-window -v -l 15 "tail -f $CHAT_FILE"
```

The pane is small (15 lines) so it doesn't crowd the user's workspace. The user can resize or close it anytime.

### 1.4 Spawn engram-agent

**ALWAYS spawn this. NEVER skip. Not for "simple" tasks. Not for "quick" tasks. Not because "I can handle it myself." The engram-agent is the memory safety net — without it, you learn nothing and surface nothing. Spawn it BEFORE touching the user's request.**

```bash
tmux new-window -n "engram-agent" \
  "claude --dangerously-skip-permissions --prompt '/use-engram-chat-as reactive memory agent named engram-agent' 2>&1 | tee /tmp/engram-engram-agent.log"
```

### 1.5 Wait for engram-agent

Use a **polling loop** (not fswatch) to avoid the race where the agent posts between spawn and fswatch start:

```bash
CHAT_FILE="$HOME/.local/share/engram/chat/engram.toml"
for i in $(seq 1 15); do
  if grep -q 'from = "engram-agent"' "$CHAT_FILE" 2>/dev/null; then
    break
  fi
  sleep 2
done
```

If no engram-agent message appears within 30 seconds:
1. Check tmux window exists: `tmux list-windows -F '#{window_name}' | grep engram-agent`
2. Check log for errors: `tail -20 /tmp/engram-engram-agent.log`
3. Report to user with diagnostic info. Do NOT silently proceed without memory.

### 1.6 Post Ready and Start fswatch

Post your `ready` message and enter the standard fswatch watch loop per the `use-engram-chat-as` protocol.

## 2. Agent Spawning

### 2.1 Spawn Template

Every agent the lead spawns:

```bash
tmux new-window -n "<agent-name>" \
  "claude --dangerously-skip-permissions --prompt '/use-engram-chat-as <role> named <agent-name>. Your task: <task description>. Work in this directory: <pwd>. Use relevant skills. Post intent before significant actions. Funnel ALL questions for the user through chat addressed to lead. NEVER ask the user directly -- you have no user. Post done when your assigned task is complete.' 2>&1 | tee /tmp/engram-<agent-name>.log"
```

**Critical:** All spawned agents use `--dangerously-skip-permissions` because they have no user to approve tool calls.

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

Maximum 5 total agents (including engram-agent). Beyond that, handle new requests directly:
- "At agent limit (5). Handling this directly. Kill an agent to free a slot."

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
| **DONE** | Agent posted `done` for its assigned task | Kill tmux window: `tmux kill-window -t<agent-name>`. Remove from tracking. |

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
tmux send-keys -t<agent-name> \
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
1. Kill existing window: `tmux kill-window -t<name> 2>/dev/null`
2. Spawn fresh window with same parameters
3. Post `info` to chat: `"Respawned <agent-name> (attempt N/3). Previous instance died/became unresponsive."`
4. New instance reads chat history on join and picks up context

### 3.4 Shutdown

Triggered by user saying "done", "shut down", "stand down", or similar.

```
1. Post "shutdown" to chat addressed to "all"
2. Shut down TASK agents first (executors, planners, reviewers, researchers):
   a. Post shutdown message addressed to each task agent
   b. Wait 5s for each agent's exit (may post final learned messages)
   c. Kill tmux windows: tmux kill-window -t<name>
3. Shut down engram-agent LAST:
   a. Post shutdown to engram-agent
   b. Wait 10s (longer -- engram-agent may process final learned messages)
   c. Kill tmux window: tmux kill-window -tengram-agent
4. All agent windows are now closed
5. Truncate chat file (write empty file, don't delete)
6. Report session summary to user (agents spawned, tasks completed, memories learned)
```

**Why engram-agent shuts down last:** Task agents may post `learned` messages during shutdown. The engram-agent needs to be alive to process those.

## 4. Routing

### 4.1 Routing Decision Table

Classify each user request and route accordingly. Use LLM judgment, not keyword matching.

| User Request Pattern | Route | Agents Spawned |
|---------------------|-------|----------------|
| Simple question about code/project | **Lead handles directly** | None |
| Quick edit (file + location known) | **Lead handles directly** | None |
| "Fix bug X" / "Implement feature Y" (single-scope) | **Executor** | 1 executor |
| "Tackle issue #N" / "Work on #N" | **Plan-Execute-Review pipeline** | Sequential: planner -> executor -> reviewer |
| "Review PR #N" / "Review this code" | **Reviewer** | 1 reviewer |
| "Research X" / "How does X work?" (deep) | **Researcher** | 1 researcher |
| "File an issue" / "Create a PR" | **Lead handles directly** | None (use gh CLI) |
| "Do A and B and C" (independent tasks) | **Parallel executors** (worktree isolation) | N executors |
| "Refactor X across the codebase" | **Executor with reviewer** | 1 executor + 1 reviewer |

### 4.2 Plan-Execute-Review Pipeline

For issue-sized work, orchestrate three sequential phases:

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
- "Just do it yourself" -> lead handles directly
- "Use two executors" -> spawn as requested
- "Skip the review" -> omit Phase 3
- "I want to talk to the executor" -> relay messages bidirectionally (still proxied)

## 5. User Proxy Pattern

### 5.1 Inbound (User -> Agents)

Every user message flows through the lead:

- **New task/request** -> Route per Section 4
- **Answer to pending question** -> Relay to asking agent as `info` in chat
- **Correction** -> Parrot as `info` (engram-agent will detect and learn)
- **Status query** -> Check agent states and respond directly

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

### 6.2 Periodic Health Check (Every 2 Minutes)

1. Check all tracked agents against silence thresholds
2. Verify tmux windows exist: `tmux list-windows -F '#{window_name}'`
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

After 300 messages: degraded mode -- minimal context retention, checkpoint every 50 messages, bias toward handling tasks directly.

### 7.4 Lead Restart Recovery

If the lead dies and user restarts:
1. Read chat.toml from beginning -- reconstruct agent registry
2. Check tmux session: `tmux list-windows -F '#{window_name}'`
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
