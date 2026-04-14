# Stage 1 Remaining: Hooks + Skill Rewrites + Retire Old Code

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete Stage 1 by adding hooks that post user/agent output to the API server, rewriting all 5 engram skills to describe the new server-driven model, and deleting the old dispatch/chat/agent/hold CLI code.

**Architecture:** Hooks are shell scripts registered in `hooks/hooks.json` that call the `engram` CLI client. Skills are SKILL.md files describing agent behavior. Old code deletion removes `cli_dispatch.go`, `cli_agent.go`, and related files/functions. Order: hooks first (needed for e2e), then skills (describe the system), then delete old code (cleanup).

**Tech Stack:** Shell (hooks), Markdown (skills), Go (deletion).

**Principles:** Read `docs/exec-planning.md`. Skills are rewrites not patches (per spec). Old code is deleted immediately, not deprecated.

---

## Section A: Hooks

### Task 1: Add UserPromptSubmit, Stop, and SubagentStop hooks

Add three hooks to `hooks/hooks.json`. Each calls the engram CLI client.

**Files:**
- Modify: `hooks/hooks.json`
- Create: `hooks/user-prompt.sh`
- Create: `hooks/agent-stop.sh`
- Create: `hooks/subagent-stop.sh`

- [ ] **Step 1: Write the UserPromptSubmit hook script**

Create `hooks/user-prompt.sh`:
```bash
#!/usr/bin/env bash
# UserPromptSubmit hook: posts user prompt to engram API server.
# In stage 1 (pre-MCP), uses engram intent (blocking) to get surfaced memories.
# The agent name is set during /use-engram setup via ENGRAM_AGENT_NAME.

set -euo pipefail

if [ -z "${ENGRAM_AGENT_NAME:-}" ]; then
  exit 0  # Engram not active for this session.
fi

# $PROMPT is provided by Claude Code's hook system.
engram intent \
  --from "${ENGRAM_AGENT_NAME}:user" \
  --to engram-agent \
  --situation "${PROMPT:-}" \
  --planned-action ""
```

- [ ] **Step 2: Write the Stop hook script**

Create `hooks/agent-stop.sh`:
```bash
#!/usr/bin/env bash
# Stop hook: posts agent output to engram API server.
# In stage 1 (pre-MCP), uses engram intent (blocking) to get surfaced memories.

set -euo pipefail

if [ -z "${ENGRAM_AGENT_NAME:-}" ]; then
  exit 0
fi

# $STOP_RESPONSE is provided by Claude Code's hook system.
engram intent \
  --from "${ENGRAM_AGENT_NAME}" \
  --to engram-agent \
  --situation "${STOP_RESPONSE:-}" \
  --planned-action ""
```

- [ ] **Step 3: Write the SubagentStop hook script**

Create `hooks/subagent-stop.sh`:
```bash
#!/usr/bin/env bash
# SubagentStop hook: posts subagent output to engram chat.
# Addressed to engram-agent (the lead already sees subagent output natively).

set -euo pipefail

if [ -z "${ENGRAM_AGENT_NAME:-}" ]; then
  exit 0
fi

# $SUBAGENT_ID and $SUBAGENT_OUTPUT are provided by Claude Code's hook system.
engram post \
  --from "${ENGRAM_AGENT_NAME}:subagent:${SUBAGENT_ID:-unknown}" \
  --to engram-agent \
  --text "${SUBAGENT_OUTPUT:-}"
```

- [ ] **Step 4: Register hooks in hooks.json**

Replace the contents of `hooks/hooks.json`:
```json
{
  "hooks": {
    "SessionStart": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "${CLAUDE_PLUGIN_ROOT}/hooks/session-start.sh",
            "timeout": 10
          }
        ]
      }
    ],
    "UserPromptSubmit": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "${CLAUDE_PLUGIN_ROOT}/hooks/user-prompt.sh",
            "timeout": 30
          }
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "${CLAUDE_PLUGIN_ROOT}/hooks/agent-stop.sh",
            "timeout": 30
          }
        ]
      }
    ],
    "SubagentStop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "${CLAUDE_PLUGIN_ROOT}/hooks/subagent-stop.sh",
            "timeout": 10
          }
        ]
      }
    ]
  }
}
```

- [ ] **Step 5: Make scripts executable**

```bash
chmod +x hooks/user-prompt.sh hooks/agent-stop.sh hooks/subagent-stop.sh
```

- [ ] **Step 6: Verify hooks.json is valid JSON**

```bash
python3 -m json.tool hooks/hooks.json > /dev/null && echo "valid JSON"
```

- [ ] **Step 7: Commit**

```bash
git add hooks/hooks.json hooks/user-prompt.sh hooks/agent-stop.sh hooks/subagent-stop.sh
git commit -m "feat(hooks): add UserPromptSubmit, Stop, SubagentStop hooks

Stage 1 hooks call engram CLI client. UserPromptSubmit and Stop use
engram intent (blocking) for synchronous memory surfacing. SubagentStop
uses engram post (fire-and-forget).

AI-Used: [claude]"
```

---

## Section B: Skill Rewrites

NOTE: The writing-skills skill says skills must be tested with pressure scenarios (TDD for docs). For this plan, the "test" is: does the skill accurately describe the system as it exists after stages 0-1b? Read the current skill, compare to the spec, rewrite.

### Task 2: Rewrite engram-agent skill

The engram-agent skill is a near-complete rewrite. The current skill describes a self-directed agent with cursor management, speech markers, and rate limiting. The new skill describes a server-invoked agent that responds with structured JSON.

**Files:**
- Modify: `skills/engram-agent/SKILL.md`

- [ ] **Step 1: Read the current skill**

Read `skills/engram-agent/SKILL.md` to understand what exists.

- [ ] **Step 2: Rewrite the skill**

The new skill must cover (from spec section 8, Stage 1):
- Agent is invoked by the server via `claude -p --resume`
- Every response MUST be structured JSON: `{"action": "surface"|"log-only"|"learn", "to": "...", "text": "..."}`
- `surface`: return relevant memories for the requesting agent
- `log-only`: nothing relevant found, record for the log
- `learn`: evaluate a learning and decide whether to save as a memory file
- Preserve: memory judgment logic (what to surface, what to learn, what to ignore), failure correlation, memory quality evaluation
- Server will re-prompt if output isn't structured
- Server will ask to reload skill if issues persist
- Skill refresh: "The engram server will periodically reload your skills by injecting reload instructions into the prompt."

- [ ] **Step 3: Commit**

```bash
git add skills/engram-agent/SKILL.md
git commit -m "feat(skills): rewrite engram-agent for server-driven model

Replaces self-directed model (cursors, speech markers, rate limiting)
with server-invoked model (structured JSON output, server-managed lifecycle).

AI-Used: [claude]"
```

### Task 3: Rewrite use-engram-chat-as skill

**Files:**
- Modify: `skills/use-engram-chat-as/SKILL.md`

- [ ] **Step 1: Read the current skill**

- [ ] **Step 2: Rewrite the skill**

The new skill must cover (from spec section 8, Stage 1):
- Remove: subprocess-watching, cursor tracking, polling, INTENT:/ACK:/WAIT:/DONE: markers, argument protocol, RESUME_REASON handling
- Replace with: hooks handle posting user/agent output, agents use CLI commands
- Learn message format: structured JSON matching memory TOML (feedback: situation/behavior/impact/action, fact: situation/subject/predicate/object)
- Agents interact via CLI: `engram post`, `engram learn`, `engram intent`
- Old ack/wait protocol is retired, not deferred
- Skill refresh protocol: server will periodically ask to reload skills

- [ ] **Step 3: Commit**

```bash
git add skills/use-engram-chat-as/SKILL.md
git commit -m "feat(skills): rewrite use-engram-chat-as for server model

Retires ack/wait protocol, subprocess watching, speech markers.
Replaces with hooks + CLI commands + structured learn messages.

AI-Used: [claude]"
```

### Task 4: Update engram-lead skill

**Files:**
- Modify: `skills/engram-lead/SKILL.md`

- [ ] **Step 1: Read the current skill**

- [ ] **Step 2: Rewrite the skill**

The new skill must cover (from spec section 8, Stage 1):
- Remove: dispatch management, hold patterns, compaction recovery
- Lead uses CLI: `engram intent` before significant actions, `engram learn` after learning something, `engram post` for general messages
- Lead spawns subagents via Claude Code's native subagent mechanism (not dispatch)
- SubagentStop hooks post output to engram-agent
- Holds, fan-in coordination, worker lifecycle are retired
- Skill refresh: server will post refresh reminders to chat

- [ ] **Step 3: Commit**

```bash
git add skills/engram-lead/SKILL.md
git commit -m "feat(skills): rewrite engram-lead for server model

Retires dispatch, holds, worker management. Lead uses CLI commands
and spawns subagents via Claude Code native mechanism.

AI-Used: [claude]"
```

### Task 5: Rewrite engram-up skill

Currently a ~5 line stub. Needs real content.

**Files:**
- Modify: `skills/engram-up/SKILL.md`

- [ ] **Step 1: Read the current skill**

- [ ] **Step 2: Rewrite the skill**

The new skill must cover (from spec section 8, Stage 1):
- Startup sequence: `engram server up --chat-file <path> --log-file <path>`, then load skills via `/use-engram-chat-as` and `/engram-lead`
- If inside tmux ($TMUX set): open a pane tailing the chat file, open a pane tailing the debug log
- Document server flags and expected output
- Set ENGRAM_AGENT_NAME env var (user chooses unique name during setup)

- [ ] **Step 3: Commit**

```bash
git add skills/engram-up/SKILL.md
git commit -m "feat(skills): rewrite engram-up with server startup sequence

AI-Used: [claude]"
```

### Task 6: Update engram-down skill

**Files:**
- Modify: `skills/engram-down/SKILL.md`

- [ ] **Step 1: Read the current skill**

- [ ] **Step 2: Rewrite the skill**

The new skill must cover (from spec section 8, Stage 1):
- Shutdown: `engram post --from <agent> --to engram-agent --text "shutdown"`, then POST /shutdown to server
- If inside tmux: kill chat tail and debug log tail panes
- Remove: dispatch drain, dispatch stop, LEARNED message scan

- [ ] **Step 3: Commit**

```bash
git add skills/engram-down/SKILL.md
git commit -m "feat(skills): rewrite engram-down for server shutdown

Retires dispatch drain/stop, LEARNED scan. Uses engram post + server shutdown.

AI-Used: [claude]"
```

---

## Section C: Retire Old Code

### Task 7: Delete old dispatch, chat, agent, hold CLI commands and backing code

Delete all old commands and their internal implementations. The new CLI commands (post, intent, learn, subscribe, status, server up) are the only interface.

**Files to delete:**
- `internal/cli/cli_dispatch.go` — dispatch loop, worker management
- `internal/cli/cli_dispatch_test.go` — dispatch tests
- `internal/cli/cli_agent.go` — agent spawn/kill/list/wait-ready/run, tmux management
- `internal/cli/cli_agent_test.go` — agent tests (NOTE: check if export_test.go has agent exports that need cleanup)

**Files to modify:**
- `internal/cli/cli.go` — remove cases: "chat", "hold", "agent", "dispatch"
- `internal/cli/targets.go` — remove: BuildChatGroup, BuildDispatchGroup, BuildHoldGroup, BuildAgentGroup, and all their Args/Flags types
- `internal/cli/targets_test.go` — remove tests for deleted args/flags
- `internal/cli/export_test.go` — remove exports for deleted functions

- [ ] **Step 1: Delete the dispatch and agent implementation files**

```bash
git rm internal/cli/cli_dispatch.go internal/cli/cli_dispatch_test.go
git rm internal/cli/cli_agent.go internal/cli/cli_agent_test.go
```

- [ ] **Step 2: Remove old cases from Run() in cli.go**

Remove these cases from the `switch cmd` in `Run()`:
```go
case "chat":
case "hold":
case "agent":
case "dispatch":
```

Keep: "recall", "show", "post", "intent", "learn", "status", "server", "subscribe"

- [ ] **Step 3: Remove old groups from Targets() in targets.go**

Remove from `Targets()`:
```go
BuildChatGroup(stdout, stderr, stdin),
BuildDispatchGroup(stdout, stderr, stdin),
BuildHoldGroup(stdout, stderr, stdin),
BuildAgentGroup(stdout, stderr, stdin),
```

Remove all `Build*Group` functions for chat, dispatch, hold, agent.
Remove all `*Args` structs and `*Flags` functions for chat, dispatch, hold, agent.

- [ ] **Step 4: Remove old tests from targets_test.go**

Remove tests for deleted args/flags (TestChatPostFlags, TestDispatchStartFlags, TestHoldAcquireFlags, TestAgentRunFlags, etc.)

- [ ] **Step 5: Clean up export_test.go**

Remove exports for deleted functions (ExportDispatchLoop, ExportRouteMessage, ExportBuildClaudeCmd, ExportRunConversationLoopWith, etc.)

- [ ] **Step 6: Clean up cli.go imports and functions**

Remove:
- `runChatDispatch` and its subcommand handlers
- `runHoldDispatch` and its subcommand handlers
- `runAgentDispatch` and its subcommand handlers
- Any helper functions only used by deleted code (newFilePoster, newFileWatcher may still be used by cli_server.go — check before deleting)
- Unused imports

- [ ] **Step 7: Run tests — expect many failures from missing functions**

```bash
targ test 2>&1 | grep FAIL
```

Fix any remaining references to deleted code. This may require iterating.

- [ ] **Step 8: Run full quality check**

```bash
targ check-full
```

Fix lint, coverage, nilaway issues from the deletion.

- [ ] **Step 9: Commit**

```bash
git add -A
git commit -m "feat(cli): retire old dispatch/chat/agent/hold commands

Deletes cli_dispatch.go, cli_agent.go, and all backing code.
New CLI commands (post, intent, learn, subscribe, status, server up)
are the only interface.

AI-Used: [claude]"
```

---

### Task 8: Quality check + e2e testing

- [ ] **Step 1: Run full test suite**

Run: `targ test`
Expected: All pass

- [ ] **Step 2: Run full quality check**

Run: `targ check-full`
Expected: All pass

- [ ] **Step 3: E2E test — verify old commands are gone, new commands work**

```bash
go build -o /tmp/engram-stage1 ./cmd/engram/

# Old commands should fail
/tmp/engram-stage1 chat post --from a --to b --text c 2>&1  # should error: unknown command
/tmp/engram-stage1 dispatch start 2>&1                       # should error: unknown command
/tmp/engram-stage1 agent spawn --name x --prompt y 2>&1      # should error: unknown command

# New commands should work
/tmp/engram-stage1 post --from a --to b --text c 2>&1       # should error: connection refused (no server)
/tmp/engram-stage1 status 2>&1                                # should error: connection refused
/tmp/engram-stage1 server up --help 2>&1                     # should show flags

# Hooks should exist and be executable
ls -la hooks/user-prompt.sh hooks/agent-stop.sh hooks/subagent-stop.sh
cat hooks/hooks.json | python3 -m json.tool
```

- [ ] **Step 4: Fix any issues**

- [ ] **Step 5: Commit fixes**

```bash
git add -A
git commit -m "fix: address quality and e2e issues from stage 1 completion

AI-Used: [claude]"
```
