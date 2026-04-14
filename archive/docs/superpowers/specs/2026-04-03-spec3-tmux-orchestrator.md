# Spec 3: tmux Orchestrator (engram-tmux-lead)

The user's primary agent. All other agents are behind the scenes — the user talks only to the lead. The lead manages agent lifecycle through tmux, routes work, proxies questions, and manages its own context pressure.

**Assumes:** Spec 1 (unified memory format) and Spec 2 (use-engram-chat-as + engram-agent) are shipped and operational.

## 1. Identity and Role

The lead joins chat as an **active** agent named `lead`. It uses the `use-engram-chat-as` skill with the special role `/engram-tmux-lead`.

The lead is NOT a coordinator that delegates everything. It is the user's agent — it reads code, makes plans, answers questions, and delegates to specialists when the task warrants it. Small tasks the lead handles directly. The lead delegates when parallelism, isolation, or specialization adds value.

## 2. Startup Sequence

```
User starts claude → says "/use-engram-chat-as /engram-tmux-lead"
                                    │
                                    ▼
                        ┌─────────────────────┐
                        │  1. Join chat as     │
                        │     active "lead"    │
                        └──────────┬──────────┘
                                   │
                                   ▼
                        ┌─────────────────────┐
                        │  2. Verify/create    │
                        │     tmux session     │
                        │     "engram"         │
                        └──────────┬──────────┘
                                   │
                                   ▼
                        ┌─────────────────────┐
                        │  3. Spawn engram-    │
                        │     agent (reactive) │
                        │     in tmux window   │
                        └──────────┬──────────┘
                                   │
                                   ▼
                        ┌─────────────────────┐
                        │  4. Wait for engram- │
                        │     agent heartbeat  │
                        │     (30s timeout)    │
                        └──────────┬──────────┘
                                   │
                                   ▼
                        ┌─────────────────────┐
                        │  5. Post "ready"     │
                        │     info to chat     │
                        │     Start fswatch    │
                        └─────────────────────┘
```

### 2.1 tmux Session Management

The lead creates or attaches to a tmux session named `engram`:

```bash
# Create session if it doesn't exist (detached — lead doesn't live in tmux)
tmux has-session -t engram 2>/dev/null || tmux new-session -d -s engram -n scratch
```

All spawned agents get their own tmux window within this session. The lead itself does NOT run inside tmux — it runs in the user's terminal. tmux is infrastructure for managing child agents.

### 2.2 Spawning an Agent in tmux

Template for every agent the lead spawns:

```bash
tmux new-window -t engram -n "<agent-name>" \
  "claude --prompt '/use-engram-chat-as <role> named <agent-name>' 2>&1 | tee /tmp/engram-<agent-name>.log"
```

The `tee` captures output for debugging dead agents. Log files are ephemeral — cleaned up on session end or overwritten on respawn.

Every spawned agent receives these instructions via its role text:
1. Use `/use-engram-chat-as <role> named <agent-name>`
2. Funnel ALL questions for the user through chat, addressed to `lead`
3. NEVER ask the user directly — you have no user
4. Use relevant skills for your role
5. Post `done` when your assigned task is complete

### 2.3 Startup Agent: engram-agent

Always spawned on startup. Non-negotiable — the memory safety net must be running before work begins.

```bash
tmux new-window -t engram -n "engram-agent" \
  "claude --prompt '/use-engram-chat-as reactive memory agent named engram-agent' 2>&1 | tee /tmp/engram-engram-agent.log"
```

The lead waits for the engram-agent to post its first message (introduction or heartbeat) using a **polling loop on the chat file**, not fswatch. This avoids the race where the agent posts between the spawn command and the fswatch start:

```bash
# Poll chat.toml for engram-agent's first message
# Check every 2 seconds, timeout after 30 seconds
for i in $(seq 1 15); do
  if grep -q 'from = "engram-agent"' chat.toml 2>/dev/null; then
    break
  fi
  sleep 2
done
```

This is the one place where polling is acceptable — it runs exactly once at startup, for at most 30 seconds, and eliminates the fswatch race condition. The main fswatch loop starts only after this completes.

If no engram-agent message appears within 30 seconds:
- Check tmux window exists: `tmux list-windows -t engram -F '#{window_name}' | grep engram-agent`
- Check log for errors: `tail -20 /tmp/engram-engram-agent.log`
- Report to user with diagnostic info. Do NOT silently proceed without memory.

## 3. Agent Lifecycle State Machine

Every agent the lead manages has a lifecycle state:

```
                    ┌──────────┐
         spawn ──→  │ STARTING │
                    └────┬─────┘
                         │ first message in chat
                         ▼
                    ┌──────────┐
              ┌──→  │  ACTIVE  │ ←── nudge succeeds
              │     └────┬─────┘
              │          │ no chat message for
              │          │ silence_threshold
              │          ▼
              │     ┌──────────┐
              └──── │  SILENT  │
                    └────┬─────┘
                         │ nudge fails OR
                         │ tmux window gone
                         ▼
                    ┌──────────┐
                    │   DEAD   │
                    └────┬─────┘
                         │ lead decides
                         ▼
                  ┌──────────────┐
                  │ RESPAWN or   │
                  │ REPORT+DONE  │
                  └──────────────┘

         ── or at any point ──

                    ┌──────────┐
         task done  │   DONE   │ ── window killed
                    └──────────┘
```

### 3.1 State Definitions

| State | Entry Condition | Lead Behavior |
|-------|----------------|---------------|
| **STARTING** | `tmux new-window` executed | Monitor chat for first message. Timeout: 30s for engram-agent, 60s for others. |
| **ACTIVE** | Agent posted at least one message to chat | Normal operation. Track last-message timestamp. |
| **SILENT** | No chat message for `silence_threshold` (default: 3 minutes for task agents, 6 minutes for engram-agent between heartbeats). Detected on the next 2-minute health check (Section 6.2). | Nudge via chat + tmux (see 3.2). Worst-case timeline from silence onset to DEAD: up to 2m (detection) + 60s (nudge timeout) = ~3 minutes for task agents, ~5 minutes for engram-agent. |
| **DEAD** | Nudge failed, OR tmux window no longer exists, OR log shows crash/exit | Decide: respawn (engram-agent always), report to user (task agents). |
| **DONE** | Agent posted `done` for its assigned task | Kill tmux window: `tmux kill-window -t engram:<agent-name>`. Remove from tracking. |

### 3.2 Nudging

When an agent enters SILENT, the lead uses a two-step nudge:

**Step 1: Chat nudge.** Post a message to chat.toml addressed to the agent:

```toml
[[message]]
from = "lead"
to = "<agent-name>"
thread = "nudge"
type = "info"
text = "You appear to have gone silent. Post a status update."
```

If the agent's fswatch loop is healthy, it will wake on the chat change and respond. This is the primary nudge mechanism — it works within the established protocol.

**Step 2: tmux nudge (fallback).** If no response within 30 seconds of the chat nudge, attempt tmux send-keys:

```bash
tmux send-keys -t engram:<agent-name> \
  "Check chat.toml for messages and post a status update." Enter
```

**Limitation:** `tmux send-keys` injects keystrokes into the pane's terminal. Whether `claude` processes these as user input depends on whether the process is reading stdin interactively. This is a best-effort fallback — if the agent's claude process is not reading stdin (e.g., it's blocked in a tool call), the keystrokes buffer until it does. If the process has exited, the keystrokes go nowhere.

If the agent posts a message within 60 seconds of the first nudge (step 1), it returns to ACTIVE. If neither nudge gets a response within 60 seconds total, it transitions to DEAD.

The lead tracks nudge count per agent. After 2 failed nudge cycles (no response within 60s each), skip straight to DEAD on subsequent silence.

### 3.3 Respawn Policy

| Agent | On DEAD | Max Respawns |
|-------|---------|-------------|
| engram-agent | Always respawn automatically | 3 per session. After 3, report to user and continue without memory. |
| Task agents | Report to user. Include last 20 lines of log + last chat messages from that agent. User decides. | User-controlled. |

Respawn procedure:
1. Kill existing window if it still exists: `tmux kill-window -t engram:<name> 2>/dev/null`
2. Spawn fresh window with same parameters
3. Post `info` to chat: `"Respawned <agent-name> (attempt N/3). Previous instance died/became unresponsive."`
4. The new instance reads chat history on join and picks up context

### 3.4 Shutdown

Triggered by user saying "done", "shut down", "stand down", or similar dismissal language.

```
1. Post "shutdown" info to chat
2. Shut down all TASK agents first (executors, planners, reviewers, researchers):
   a. Post "stand down" message addressed to each task agent
   b. Wait 5s for each agent's exit (agents may post final learned messages)
   c. Kill tmux windows: tmux kill-window -t engram:<name>
3. Shut down engram-agent LAST:
   a. Post "stand down" to engram-agent
   b. Wait 10s (longer — engram-agent may need to process final learned messages from step 2)
   c. Kill tmux window: tmux kill-window -t engram:engram-agent
4. Kill tmux session: tmux kill-session -t engram
5. Post final "session complete" info to chat
6. Report session summary to user (agents spawned, tasks completed, memories learned)
```

**Why engram-agent shuts down last:** Task agents may post `learned` messages during their shutdown. The engram-agent needs to be alive to process those messages and extract facts/feedback. This ordering is not accidental — it is a hard requirement regardless of spawn order.

## 4. Routing

### 4.1 Routing Decision Table

The lead classifies each user request and routes accordingly. Classification is LLM judgment, not keyword matching — these are patterns, not regex.

| User Request Pattern | Route | Agents Spawned |
|---------------------|-------|----------------|
| Simple question about code/project | **Lead handles directly** | None |
| Quick edit (file + location known) | **Lead handles directly** | None |
| "Fix bug X" / "Implement feature Y" (single-scope) | **Executor** | 1 executor (active) |
| "Tackle issue #N" / "Work on #N" | **Plan-Execute-Review pipeline** | 1 planner (active) → 1 executor (active) → 1 reviewer (active), sequentially |
| "Review PR #N" / "Review this code" | **Reviewer** | 1 reviewer (active) |
| "Research X" / "How does X work?" (deep) | **Researcher** | 1 researcher (active) |
| "File an issue" / "Create a PR" | **Lead handles directly** | None (lead uses gh CLI) |
| "Do A and B and C" (independent tasks) | **Parallel executors** (requires worktree isolation — see 4.5) | N executors (active), one per independent task |
| "Refactor X across the codebase" | **Executor with reviewer** | 1 executor + 1 reviewer (both active) |

### 4.2 Agent Role Templates

Each agent type has a predefined role string:

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
Do NOT implement — only plan.
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

### 4.3 Plan-Execute-Review Pipeline

For issue-sized work, the lead orchestrates a three-phase pipeline:

```
Phase 1: PLAN
  Lead spawns planner-1 with the issue context
  Planner reads code, analyzes, posts plan as info message
  Lead receives plan, presents to user for approval
  User approves (or modifies) → Phase 2

Phase 2: EXECUTE
  Lead spawns exec-1 with the approved plan
  Executor implements, posting intent before each significant step
  engram-agent watches intents for memory matches
  Executor posts done → Phase 3

Phase 3: REVIEW
  Lead spawns reviewer-1 with:
    - The original issue/plan
    - The executor's changes (git diff)
  Reviewer inspects, posts wait for issues or done for approval
  If issues: lead relays to user, may re-enter Phase 2
  If approved: lead reports to user, cleans up agents
```

The lead does NOT spawn all three agents simultaneously. Each phase starts after the previous completes. This prevents wasted work if the plan is rejected or the implementation needs rethinking.

### 4.5 Parallel Executor Isolation

When the lead routes to parallel executors, each executor MUST work in its own git worktree to prevent file conflicts:

```bash
# Lead creates a worktree for each parallel executor
git worktree add /tmp/engram-worktree-exec-<N> -b engram/exec-<N> HEAD
```

The executor's role template includes the worktree path as its working directory:

```
active general-purpose executor named exec-<N>.
Your task: <task description>.
Work in this directory: /tmp/engram-worktree-exec-<N>
...
```

**Merge strategy:** When all parallel executors complete:
1. Lead merges each worktree branch back to the main branch, one at a time
2. If merge conflicts occur, lead reports to user with the conflicting files
3. Worktrees are cleaned up after merge: `git worktree remove /tmp/engram-worktree-exec-<N>`

**When NOT to use worktrees:** If the parallel tasks are provably non-overlapping (e.g., one modifies only frontend files, another only backend), the lead MAY skip worktree isolation. But the default is always isolate — skipping requires the lead to verify non-overlap by examining the task scopes.

**Single executors** do not need worktrees — they work in the project root.

### 4.6 Routing Override

The user can always override routing:
- "Just do it yourself" → lead handles directly, no delegation
- "Use two executors for this" → lead spawns as requested
- "Skip the review" → lead omits Phase 3
- "I want to talk to the executor directly" → lead relays messages bidirectionally without filtering (but still proxies — the user never leaves the lead's terminal)

## 5. User Proxy Pattern

### 5.1 Inbound (User → Agents)

Every user message flows through the lead:

```
User types message
        │
        ▼
Lead receives message
        │
        ├── Is this a new task/request? → Route per Section 4
        │
        ├── Is this an answer to a pending question? → Relay to the
        │   asking agent as info message in chat
        │
        ├── Is this a correction? → Parrot as info (per file-comms protocol),
        │   engram-agent will detect and learn
        │
        └── Is this a status query? → Lead checks agent states and responds
```

### 5.2 Outbound (Agents → User)

Agents post messages to chat addressed to `lead`. The lead decides what to surface:

| Message Type | Lead Action |
|-------------|-------------|
| Question addressed to `lead` | Present to user immediately. Prefix with agent name for context. |
| `done` with results | Summarize and present to user. Kill agent if task-scoped. |
| `info` status update | Accumulate. Surface on user's next interaction or if significant. |
| `wait` (agent-to-agent dispute) | Monitor. Only surface if escalated (4th message in argument protocol). |
| Escalation (from argument protocol) | Present to user immediately with full context from both sides. |

### 5.3 Question Queuing

Multiple agents may post questions simultaneously. The lead queues them and presents them one at a time to avoid overwhelming the user. Order: most-blocking question first (the agent closest to being stuck waiting for an answer).

If a question becomes stale (the asking agent posted `done` or moved on before the user answered), the lead drops it silently.

## 6. Monitoring

### 6.1 Chat Watch Loop

The lead runs the standard fswatch loop (per file-comms protocol) to watch chat.toml. Between user interactions, the lead is idle but watching — it wakes on chat changes and processes agent messages.

When the user types a message, the lead:
1. Processes the user message (route/relay/respond)
2. Checks for unprocessed chat messages since last wake
3. Handles any pending agent messages before returning to the user

### 6.2 Periodic Health Check

Every 2 minutes (triggered by heartbeat timer, not a separate loop), the lead:

1. Checks all tracked agents against their silence thresholds
2. Verifies tmux windows still exist: `tmux list-windows -t engram -F '#{window_name}'`
3. Transitions SILENT/DEAD agents per Section 3
4. If engram-agent is missing its heartbeat window (>6 min since last heartbeat), nudge immediately

### 6.3 What the Lead Reports to the User Unprompted

- Agent spawned or killed (one line: "Spawned executor exec-1 for: implement login handler")
- Agent died and was respawned (with brief reason)
- Escalated disputes from argument protocol
- Session resource warnings (see Section 7)

What the lead does NOT report unprompted:
- Routine intent/ack/done traffic between agents
- Memory surfacing (engram-agent handles this via the argument protocol)
- Agent heartbeats

## 7. Context Pressure Management

The lead's own context window is finite. As a long-lived orchestrator, it must manage what it keeps vs. what it discards.

### 7.1 What the Lead Keeps in Context

| Data | Retention | Size Estimate |
|------|-----------|--------------|
| Active agent registry (name, state, role, last-message-ts, task summary) | Always | ~50 tokens/agent |
| Current user task and routing decision | Until task completes | ~200 tokens |
| Pending questions queue | Until answered or stale | ~100 tokens/question |
| Last 5 chat messages per active agent | Rolling window | ~500 tokens/agent |
| Routing decision table (internalized from skill) | Always | ~0 (part of skill prompt) |

### 7.2 What the Lead Offloads

| Data | Strategy |
|------|----------|
| Full chat.toml history | Read from file on demand (cursor-based) |
| Agent logs | Read from `/tmp/engram-<name>.log` on demand |
| Completed task summaries | Post to chat as `info`, re-read if needed |
| Plan documents | Stored in chat.toml by planner, re-read by executor |

### 7.3 Context Overflow Strategy

The lead tracks **total chat messages processed** as its context pressure signal. Thresholds:

| Messages Processed | Action |
|-------------------|--------|
| 100 | Post a session checkpoint (see below). Summarize completed tasks to one-line entries. |
| 200 | Reduce rolling window to last 2 messages per agent. Drop stale questions (>5 min unanswered). |
| 300 | Tell user: "Context is getting full. Consider committing current work and starting a fresh session." |

These thresholds are conservative estimates. At ~100 tokens per processed message (reading + routing overhead), 300 messages consumes ~30k tokens of accumulated context. The actual limit depends on the model and conversation history, but these checkpoints trigger mitigation well before context becomes critical.

**Session checkpoint** — posted at the 100-message mark and on every subsequent threshold:

```toml
[[message]]
from = "lead"
to = "all"
thread = "checkpoint"
type = "info"
text = """
Session checkpoint:
- Active agents: exec-1 (ACTIVE, implementing login handler), engram-agent (ACTIVE, last heartbeat 2m ago)
- Pending questions: 0
- Completed: planner-1 (produced plan), reviewer-1 (approved changes)
- User task: "Tackle issue #528 — implement OAuth login"
"""
```

The 300-message warning is the final escalation. If the user continues, the lead operates in degraded mode — minimal context retention, checkpoint on every 50 messages, and a bias toward handling tasks directly rather than spawning new agents (which generate more chat traffic).

### 7.4 Lead Restart Recovery

If the lead dies and the user restarts it:

1. Read chat.toml from the beginning — reconstruct agent registry from messages
2. Check tmux session: `tmux list-windows -t engram -F '#{window_name}'`
3. Match running tmux windows against agents found in chat history
4. Resume monitoring. Post `info`: "Lead restarted. Recovered state: N agents running."

Agents don't need to know the lead restarted — they communicate via chat.toml, not direct connection.

## 8. Error Handling

### 8.1 tmux Not Available

If `tmux` is not installed or not in PATH:

```
Lead detects on startup → reports to user:
  "tmux is required for multi-agent orchestration. Install with: brew install tmux
   I can still work as a single agent without delegation."
```

The lead falls back to direct execution mode — it handles everything itself, no delegation. The engram-agent can still be run manually by the user in a separate terminal using the existing file-comms pattern.

### 8.2 Agent Spawn Failure

If `tmux new-window` succeeds but the agent never posts to chat (STARTING → timeout):

1. Check if the tmux window still exists
2. Read last 20 lines of the agent's log file
3. Report to user: "Failed to start <agent-name>. Log shows: <last error line>. Shall I retry or handle this myself?"

### 8.3 Chat File Corruption

If chat.toml becomes unparseable (truncated write, disk full):

1. The lead detects this when reading new messages fails to parse
2. Copy corrupted file: `cp chat.toml chat.toml.corrupt.<timestamp>`
3. Start a new chat.toml with a reset notice as the first message
4. Post the lead's most recent session checkpoint (from Section 7.3) to the new file so agents can reconstruct context
5. Post chat nudges to all tracked agents — their fswatch loops will trigger on the new file write, and they'll re-read from the top on the next cycle

**No partial recovery.** TOML parsers reject the entire file on any syntax error. Attempting to read "up to the last valid block" requires a custom line-oriented parser, which is not worth the complexity. Backup and restart is the only strategy.

### 8.4 Multiple Concurrent Failures

If 2+ agents die within 60 seconds:

1. Don't auto-respawn all of them — this may indicate a systemic issue (disk full, chat corruption, etc.)
2. Respawn engram-agent only (if it was one of the dead agents)
3. Report all failures to user with diagnostics
4. Wait for user direction before respawning task agents

### 8.5 fswatch Failure

If fswatch is not available or crashes:

1. Fall back to polling: `stat -f %m chat.toml` every 2 seconds
2. Post warning to user: "fswatch unavailable, falling back to polling. Install with: brew install fswatch"
3. The polling loop replaces the fswatch background task but follows the same process-on-change logic

## 9. Agent Naming Convention

All agents spawned by the lead follow a naming scheme:

```
<role>-<monotonic-N>
```

Examples: `exec-1`, `exec-2`, `planner-1`, `reviewer-1`, `researcher-1`

The counter is **per-role** — each role has its own monotonically increasing counter. IDs are never reused within a session. So a plan-execute-review pipeline produces `planner-1`, `exec-1`, `reviewer-1` — not `planner-1`, `exec-2`, `reviewer-3`. This avoids confusing gaps in numbering.

The engram-agent is an exception — it's always named `engram-agent` (not numbered) since there's exactly one.

## 10. User Commands

The lead recognizes these meta-commands from the user (in addition to normal task requests):

| Command | Action |
|---------|--------|
| "status" / "what's running?" | List all agents with states and current tasks |
| "kill <agent-name>" | Transition agent to DONE, kill tmux window |
| "logs <agent-name>" | Show last 50 lines of agent's log file |
| "nudge <agent-name>" | Force a nudge to the named agent |
| "shut down" / "done for today" | Full shutdown sequence (Section 3.4) |
| "restart <agent-name>" | Kill and respawn the named agent |

## 11. Session Boundaries

### 11.1 What the Lead Owns

- The tmux session `engram` and all windows within it
- The agent registry (in-context state)
- Routing decisions
- User proxy (question queue, message relay)
- Lifecycle management (spawn, monitor, nudge, kill)

### 11.2 What the Lead Does NOT Own

- Memory files — engram-agent's domain
- Chat protocol — defined by `use-engram-chat-as` skill
- Argument protocol — agents handle their own disputes (lead only surfaces escalations)
- Task execution — delegated agents do the work
- Build/test operations — executors handle these

### 11.3 Interaction with engram-agent

The lead and engram-agent interact through the standard chat protocol. The lead does not have special privileges — it posts intents like any active agent, and engram-agent may WAIT on the lead's intents just like anyone else's.

The lead's additional responsibilities regarding engram-agent:
- Ensure it's running (respawn if dead)
- Monitor its heartbeats
- Relay its escalations to the user
- Parrot user input so engram-agent can detect corrections

## 12. Worked Example: "Tackle Issue #528"

```
User: "Tackle issue #528 — implement OAuth login"

Lead:
  1. Posts info to chat: parrots user input
  2. Classifies: issue-sized work → Plan-Execute-Review pipeline
  3. Reads issue #528 via gh (direct — no agent needed for this)
  4. Tells user: "Starting plan-execute-review pipeline for #528"

  PLAN PHASE:
  5. Spawns planner-1 in tmux with issue context
  6. planner-1 reads code, analyzes, posts plan to chat
  7. Lead reads plan from chat, presents to user
  8. User: "Looks good, but skip the database migration for now"
  9. Lead relays modified approval to planner-1 as info
  10. planner-1 posts updated plan, then done
  11. Lead kills planner-1 window

  EXECUTE PHASE:
  12. Spawns exec-1 with approved plan
  13. exec-1 posts intent: "Adding OAuth provider configuration"
  14. engram-agent checks memories — finds one about OAuth token storage
  15. engram-agent/sub-1 posts WAIT: "Memory says: store tokens encrypted, not plaintext"
  16. exec-1 responds factually, adjusts approach, posts ack
  17. exec-1 continues implementing, posting intents for each step
  18. exec-1 posts done: "OAuth login implemented. 3 files changed, tests passing."

  REVIEW PHASE:
  19. Lead spawns reviewer-1 with plan + git diff
  20. reviewer-1 reviews, posts done: "LGTM. One suggestion: add rate limiting to token endpoint."
  21. Lead presents review to user
  22. User: "Ship it, we'll add rate limiting in a follow-up"
  23. Lead kills reviewer-1, kills exec-1
  24. Lead: "Done. Want me to commit and create a PR?"
```

## 13. Resolved Design Decisions

1. **Agent concurrency limit:** 5 total agents (including engram-agent). Beyond that, the lead handles new requests directly rather than queuing spawns. The user is told: "At agent limit (5). Handling this directly. Kill an agent to free a slot."
2. **Cross-task pivot:** When the user starts a new task while agents are running, the lead asks: "exec-1 is still working on X. Kill it, let it finish, or wait?" Never auto-kill.
3. **Lead self-delegation:** Yes. The lead can spawn a researcher for itself — this is standard routing, not a special case.
