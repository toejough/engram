# Fix Protocol Skill: Early Ready (#496) + Stale Timestamps (#499)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **REQUIRED:** Use `superpowers:writing-skills` when editing SKILL.md files. No exceptions.

**Goal:** Fix two protocol bugs in `skills/use-engram-chat-as/SKILL.md` — agents appear dead during initialization (ready message is step 7 instead of step 1), and stale timestamps break online/offline detection and session replay (no fresh-timestamp requirement).

**Architecture:** Pure prose changes to one skill file. No Go code, no tests beyond the skill's own TDD process (writing-skills skill). Two independent edits that can be applied sequentially. Writing-skills TDD cycle: baseline behavior test → edit → pressure test.

**Tech Stack:** Markdown/TOML prose. `superpowers:writing-skills` skill for TDD. `gh` CLI for closing issues.

---

## File Structure

- **Modify only:** `skills/use-engram-chat-as/SKILL.md`
  - Section: `## Agent Lifecycle` (lines 510–531) — move ready + add cursor init + add init-complete signal
  - Section: `## Ready Messages` semantics paragraph (lines 461–464)
  - Section: `## Ready Messages` "Who waits for whom" (lines 466–470)
  - Section: `### Joining Late` (line 236–238)
  - Section: `## Message Type Catalog` `ready` row (line 135)
  - Section: `## Common Mistakes` table (lines 661–691)
  - Section: `## Compaction Recovery` Step 6 reference (line 644–646)
  - Section: `## Writing Messages` code block + new Timestamp Freshness subsection (lines 139–166)
  - Section: `## Message Format` field list, `ts` field description (line 122)

---

## Task 1: Fix #496 — Move Ready to Step 1 of Agent Lifecycle

**Files:**
- Modify: `skills/use-engram-chat-as/SKILL.md` (Agent Lifecycle, Ready Messages semantics, Common Mistakes)

### Before you start: invoke the writing-skills skill

- [ ] **Step 1.1: Invoke `superpowers:writing-skills`**

  This enforces TDD for skill edits. Follow it exactly — baseline test first, then edit, then pressure test.

### Baseline behavior test (RED)

- [ ] **Step 1.2: Write baseline test in the writing-skills test file**

  Document the current (broken) behavior:
  - Agent reads 20 messages, loads resources, spawns monitor — THEN posts ready
  - An observer watching chat sees 1–3 minutes of silence before the agent appears
  - Ready message contains initialization stats (e.g., "Loaded 47 feedback memories")
  - Issue: silence is indistinguishable from the agent being dead

### Edit (GREEN): Agent Lifecycle section

- [ ] **Step 1.3: Update the `Agent Lifecycle` numbered list**

  Current (lines 512–530):
  ```
  1. Derive chat file path from $PWD
  2. Create chat directory if needed
  3. Read last 20 messages to catch up (read further back if needed)
  4. Read chat file (catch up on history)
  5. Load resources (memories, configs, etc.)
  6. Spawn background monitor Agent (Background Monitor Pattern, above)
  7. Post ready message (with ts)
  8. Wait for monitor Agent notification
  9. Monitor Agent returns semantic event -> process event if addressed to you
  10. If acting:
      ...
  11. Post response (with lock)
  12. Go to step 8 -- ALWAYS. Even after completing a task.
  ```

  Replace with:
  ```
  1. Derive chat file path from $PWD
  2. Create chat directory if needed
  3. Initialize cursor: CURSOR=$(wc -l < "$CHAT_FILE") — BEFORE posting ready so the monitor
     captures any work routed by lead between your ready message and monitor startup
  4. Post ready message (with fresh ts) — announce presence immediately, before reading history
  5. Read last 20 messages to catch up (read further back if needed)
  6. Load resources (memories, configs, etc.)
  7. Spawn background monitor Agent (Background Monitor Pattern, above) using CURSOR from step 3
  8. Post info: "Initialization complete. Monitor active." — signals lead that agent is operational
  9. Wait for monitor Agent notification
  10. Monitor Agent returns semantic event -> process event if addressed to you
  11. If acting:
      a. Post intent to (engram-agent + any other relevant recipients)
      b. Wait for explicit ACK from all TO recipients (see Intent Protocol)
      c. Act
      d. Pre-done cursor-check: spawn background Agent to tail CHAT_FILE from cursor, grep for unresolved WAITs
         If any WAIT addressed to you and unresolved: engage before posting done
      e. Post result
  12. Post response (with lock)
  13. Go to step 9 -- ALWAYS. Even after completing a task.
  ```

  Key changes:
  - Steps 3 and 4 were duplicated ("Read last 20..." and "Read chat file...") — consolidate
  - New step 3: cursor initialization (BEFORE posting ready — ensures no messages missed)
  - New step 4: ready message (moved from old step 7)
  - New step 8: "initialization complete" info message (so lead knows when to route work)
  - Step 12 "Go to step N" updated to step 9

### Edit: Ready Messages semantics paragraph

- [ ] **Step 1.4: Update the Ready Messages semantics paragraph**

  Current (lines 461–464):
  ```
  - Posted **once**, after the agent has: (1) read full chat history, (2) loaded resources, (3) spawned its background monitor Agent.
  - Addressed to `all`. Every agent posts `ready` regardless of role.
  - The `text` field contains agent-specific initialization stats. No required format.
  ```

  Replace with:
  ```
  - Posted **once**, as the agent's **first action** after deriving the chat file path — before reading history, loading resources, or spawning the monitor. Announcing presence early prevents observers from mistaking initialization silence for a dead agent.
  - The `text` field should reflect current init status. If still initializing: `"Joining chat — reading history and loading resources. Will be fully operational shortly."` If fast init: include stats as before.
  - Addressed to `all`. Every agent posts `ready` regardless of role.
  ```

### Edit: Common Mistakes table

- [ ] **Step 1.5: Update the Common Mistakes row for `ready`**

  Current (line 675):
  ```
  | Skip `ready` message | Always post `ready` after initialization, before entering watch loop |
  ```

  Replace with:
  ```
  | Skip `ready` message or post it late | Post `ready` as your FIRST action — before reading history or loading resources. Presence before initialization, not after. |
  ```

### Edit: Compaction Recovery cross-reference

- [ ] **Step 1.6: Update step number reference in Compaction Recovery**

  Current (line 646):
  ```
  Continue the lifecycle from step 8 of the Agent Lifecycle. Do not re-post a `ready` message — `info` is sufficient.
  ```

  Replace with:
  ```
  Continue the lifecycle from step 9 of the Agent Lifecycle. Do not re-post a `ready` message — `info` is sufficient.
  ```

### Edit: "Who waits for whom" — lead must wait for init-complete, not just ready

- [ ] **Step 1.7: Update "Who waits for whom" in Ready Messages section**

  Current (lines 466–470):
  ```
  - **Lead setup:** The lead waits for `ready` from spawned agents before routing work (30s timeout).
  - **Standalone setup:** Agents don't wait for each other. `ready` is informational.
  - **Late joiners:** Read full history on join. `ready` announces presence but doesn't replay missed intents.
  - **Reactive agents:** Post `ready` but do not wait for anyone else.
  ```

  Replace with:
  ```
  - **Lead setup:** The lead waits for the agent's "initialization complete" `info` message before routing work (30s timeout from that message, not from `ready`). The initial `ready` message only announces presence — the agent may still be reading history. Routing work before the init-complete signal risks the agent processing the assignment before its monitor is watching.
  - **Standalone setup:** Agents don't wait for each other. `ready` and the init-complete `info` are informational.
  - **Late joiners:** Post `ready` first to announce presence, then read full history before spawning the monitor and posting the init-complete `info`.
  - **Reactive agents:** Post `ready` and the init-complete `info`, but do not wait for anyone else.
  ```

### Edit: "Joining Late" section — align with new lifecycle

- [ ] **Step 1.8: Update the "Joining Late" section**

  Current (lines 236–238):
  ```
  ### Joining Late

  If you join a channel that already has messages, read the entire file first to catch up before posting or watching.
  ```

  Replace with:
  ```
  ### Joining Late

  If you join a channel that already has messages: post `ready` first to announce presence, then read history to catch up, then spawn the monitor and post the init-complete `info`. Do not read history before posting `ready` — observers cannot distinguish a late-starting agent from a dead one during the silence.
  ```

### Edit: Message Type Catalog — update `ready` description

- [ ] **Step 1.9: Update the `ready` row in the Message Type Catalog**

  Current (line 135):
  ```
  | `ready` | Agent initialization complete, watching chat | Any agent | No (but spawners may wait for it) |
  ```

  Replace with:
  ```
  | `ready` | Agent joining chat — announces presence (may still be initializing) | Any agent | No (but spawners wait for the subsequent init-complete `info` before routing work) |
  ```

### Pressure test and commit

- [ ] **Step 1.10: Run pressure test per writing-skills skill**

  Verify that an agent following the updated skill would:
  - Initialize cursor before posting ready (so monitor catches work routed by lead after ready)
  - Post ready immediately as first visible action (before reading history)
  - Have ready message text signaling "joining, may still be initializing"
  - Post an "initialization complete, monitor active" info message after spawning the monitor
  - "Joining Late" and lifecycle steps no longer contradict each other
  - Lead knows to wait for the init-complete info message, not just the ready

- [ ] **Step 1.11: Commit**

  ```bash
  git add skills/use-engram-chat-as/SKILL.md
  git commit -m "fix(protocol-skill): post ready immediately on join; add init-complete signal (#496)

  Moves 'post ready' from step 7 to step 4 of Agent Lifecycle (after cursor
  init). Agents announce presence before reading history, eliminating 1-3 min
  initialization blackout. Adds init-complete info message after monitor
  spawns so lead knows when to route work. Updates Joining Late, Message Type
  Catalog, and Who-waits-for-whom sections for consistency.

  AI-Used: [claude]"
  ```

---

## Task 2: Fix #499 — Add Fresh Timestamp Requirement

**Files:**
- Modify: `skills/use-engram-chat-as/SKILL.md` (Message Format ts field, Writing Messages code block + new subsection, Common Mistakes)

### Baseline behavior test (RED)

- [ ] **Step 2.1: Write baseline test in the writing-skills test file**

  Document current (broken) behavior:
  - The `ts` field description says only "ISO 8601 timestamp (required on all message types)"
  - The Writing Messages code block shows a hardcoded literal: `ts = "2026-04-03T14:30:00Z"`
  - No requirement to call `date -u` fresh per message
  - Result: agents that set `TS=$(date -u ...)` once at loop start and reuse it produce stale timestamps
  - Real observed impact: 4–7 minute timestamp inversions, false offline detection, heartbeat timing errors

### Edit: `ts` field description in Message Format

- [ ] **Step 2.2: Update the `ts` field description**

  Current (line 122):
  ```
  - **ts**: ISO 8601 timestamp (required on all message types)
  ```

  Replace with:
  ```
  - **ts**: ISO 8601 UTC timestamp, **generated fresh at the moment of posting** — never cached. Use `$(date -u +"%Y-%m-%dT%H:%M:%SZ")` inline in the heredoc for each message.
  ```

### Edit: Writing Messages code block

- [ ] **Step 2.3: Update the Writing Messages code block to show fresh timestamp pattern**

  Current heredoc in the code block (lines 151–163):
  ```bash
  cat >> "$CHAT_FILE" << 'EOF'

  [[message]]
  from = "myname"
  to = "recipient"
  thread = "topic"
  type = "info"
  ts = "2026-04-03T14:30:00Z"
  text = """
  Content here.
  """
  EOF
  ```

  Replace with (note: single-quote heredoc `'EOF'` prevents variable expansion, so switch to unquoted `EOF` to allow `$(date)` substitution):
  ```bash
  cat >> "$CHAT_FILE" << EOF

  [[message]]
  from = "myname"
  to = "recipient"
  thread = "topic"
  type = "info"
  ts = "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
  text = """
  Content here.
  """
  EOF
  ```

  **Important:** The heredoc delimiter must change from `<< 'EOF'` (literal, no expansion) to `<< EOF` (expansion enabled) so the `$(date ...)` substitution runs at write time.

### Edit: Add Timestamp Freshness subsection in Writing Messages

- [ ] **Step 2.4: Add "### Timestamp Freshness" subsection after the Writing Messages code block**

  Insert after the closing triple-backtick of the lock/append/unlock code block (line 164), before the `If \`shlock\` isn't available` fallback note. Do NOT insert inside the code block — the new content is a markdown prose section at the `###` heading level:

  ```markdown
  ### Timestamp Freshness

  **Every message must use a freshly generated timestamp.** Never cache a timestamp in a variable and reuse it across multiple messages.

  ```bash
  # ❌ BAD: cached — all messages in this loop iteration share the same ts
  TS=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
  cat >> "$CHAT_FILE" << EOF
  ts = "$TS"
  EOF
  # ... later ...
  cat >> "$CHAT_FILE" << EOF
  ts = "$TS"   # BUG: minutes or hours stale
  EOF

  # ✅ GOOD: fresh per message via unquoted heredoc
  cat >> "$CHAT_FILE" << EOF
  ts = "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
  EOF
  ```

  **Why it matters:** The online/offline detection protocol compares message `ts` values against wall clock time. A stale `ts` makes an active agent appear offline (triggering wrong timeout behavior) or makes a dead agent appear online. It also makes session replay and debugging unreliable.
  ```

### Edit: Common Mistakes table

- [ ] **Step 2.5: Add row to Common Mistakes table**

  After the last row (line 691), add:
  ```
  | Reusing a cached TS variable across messages | Call `$(date -u +"%Y-%m-%dT%H:%M:%SZ")` fresh inline in each heredoc. Use unquoted `<< EOF` not `<< 'EOF'` to enable substitution. |
  ```

### Pressure test and commit

- [ ] **Step 2.6: Run pressure test per writing-skills skill**

  Verify that an agent following the updated skill would:
  - Use unquoted heredoc (`<< EOF`) for message writes
  - Call `$(date -u ...)` inline within the heredoc, not via a cached variable
  - Produce messages with timestamps accurate to the moment of posting

- [ ] **Step 2.7: Commit**

  ```bash
  git add skills/use-engram-chat-as/SKILL.md
  git commit -m "fix(protocol-skill): require fresh timestamp per message, never cache (#499)

  Adds Timestamp Freshness subsection to Writing Messages with explicit
  bad/good examples. Updates ts field description in Message Format.
  Switches code block heredoc from << 'EOF' to << EOF so $(date -u ...)
  expands at write time. Adds Common Mistakes row.

  AI-Used: [claude]"
  ```

---

## Task 3: Close Issues

- [ ] **Step 3.1: Close issue #496**

  ```bash
  gh issue close 496 --comment "Fixed in [commit hash]: moved ready message to step 1 of Agent Lifecycle in skills/use-engram-chat-as/SKILL.md. Agents now announce presence before reading history or loading resources."
  ```

- [ ] **Step 3.2: Close issue #499**

  ```bash
  gh issue close 499 --comment "Fixed in [commit hash]: added Timestamp Freshness subsection to Writing Messages, updated ts field description, switched heredoc to << EOF for $(date -u ...) expansion, added Common Mistakes row."
  ```
