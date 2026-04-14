# Lead Lifecycle Intents Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **REQUIRED SUB-SKILL for all SKILL.md edits:** Use `superpowers:writing-skills` before every task that modifies SKILL.md. It enforces TDD (baseline test → edit → verify) and pressure testing.

**Goal:** Add five intent-posting steps to `skills/engram-tmux-lead/SKILL.md` so the lead posts an intent addressed to `engram-agent` before every agent lifecycle action (spawn, kill, respawn, session shutdown, routing decision).

**Architecture:** Pure skill text edits — five insertion sites in one file. Each site gets an explicit TOML intent block with ACK-wait instruction. No code changes. Uses `superpowers:writing-skills` TDD discipline for each edit.

**Tech Stack:** Markdown, engram chat TOML protocol, `superpowers:writing-skills` skill.

---

## Files

- Modify: `skills/engram-tmux-lead/SKILL.md`

---

### Task 1: Add pre-spawn intent to Section 2.1 (Spawn Template)

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md` (Section 2.1, lines ~240–258)

- [ ] **Step 1: Invoke `superpowers:writing-skills`**

Invoke the skill. It will guide TDD for this SKILL.md edit.

- [ ] **Step 2: Run baseline behavior test (RED)**

Describe this scenario to yourself: "The lead is about to spawn `planner-3` for issue #507. Does the current skill text require the lead to post an intent first?"

Read Section 2.1. Confirm: the section jumps straight to `tmux split-window` with no intent step. Document this as the baseline: **no intent is required**. This is the RED state.

- [ ] **Step 3: Add the spawn intent step**

In `skills/engram-tmux-lead/SKILL.md`, replace:

```
### 2.1 Spawn Template

Every agent the lead spawns gets a **pane** in the coordinator's window (NOT a separate window):

```bash
# Split a new pane to the right, capturing the new pane ID atomically
```

with:

```
### 2.1 Spawn Template

**Step 0: Post spawn intent (required before every agent spawn)**

Before creating any pane, post an intent to `engram-agent` describing the agent you are about to spawn. Wait for ACK before proceeding (standard online/offline rules from `use-engram-chat-as`).

```toml
[[message]]
from = "lead"
to = "engram-agent"
thread = "lifecycle"
type = "intent"
ts = "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
text = """
Situation: About to spawn <role> named <agent-name>. Task: <task description>.
Behavior: Will create a new tmux pane, start claude, and send the role prompt.
"""
```

Apply standard ACK-wait timing: 5s implicit ACK if engram-agent offline; wait up to 30s and escalate to user if online but silent.

Every agent the lead spawns gets a **pane** in the coordinator's window (NOT a separate window):

```bash
# Split a new pane to the right, capturing the new pane ID atomically
```

- [ ] **Step 4: Verify behavior change (GREEN)**

Re-read Section 2.1. Confirm: the section now requires a TOML intent block + ACK-wait **before** the tmux commands. This is the GREEN state.

- [ ] **Step 5: Pressure test**

Describe this scenario: "Lead is about to spawn `exec-2` for 'fix bug #498'. What does it post before opening the pane?"

Expected answer from the updated skill: Posts intent with `from = "lead"`, `to = "engram-agent"`, `type = "intent"`, text saying "About to spawn executor named exec-2. Task: fix bug #498." Then waits for ACK.

- [ ] **Step 6: Commit**

```bash
git add skills/engram-tmux-lead/SKILL.md
git commit -m "fix(engram-tmux-lead): require intent before spawning agents (#507)

AI-Used: [claude]"
```

---

### Task 2: Add pre-kill intent to Section 3.1 (DONE state)

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md` (Section 3.1, DONE row in state table, lines ~431–437)

- [ ] **Step 1: Invoke `superpowers:writing-skills`**

Invoke the skill for this edit.

- [ ] **Step 2: Run baseline behavior test (RED)**

Describe this scenario: "Agent `planner-2` just posted `done`. No holds remain. Does the current DONE state require the lead to post intent before sending shutdown?"

Read Section 3.1 DONE row. Confirm: DONE behavior lists "1. Post shutdown… 2. Kill pane…" — no intent step. **RED.**

- [ ] **Step 3: Expand DONE behavior below the table**

The state table is already dense. Instead of cramming intent text into the table cell, add a note directly below the table:

In `skills/engram-tmux-lead/SKILL.md`, after the line:

```
**NEVER kill the engram-agent.** It runs for the entire session. Only task agents transition to DONE.
```

add:

```
**DONE state pre-kill intent (required):** Before executing the DONE transition, post an intent to `engram-agent`:

```toml
[[message]]
from = "lead"
to = "engram-agent"
thread = "lifecycle"
type = "intent"
ts = "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
text = """
Situation: <agent-name> has posted done. Hold registry confirms no incoming holds remain.
Behavior: Will send shutdown message to <agent-name> then kill its pane.
"""
```

Wait for ACK before proceeding with the shutdown + kill-pane sequence. Apply standard online/offline timing rules.
```

And update the DONE row's Lead Behavior to start with: `0. Post kill intent (see note below table).` followed by the existing steps.

- [ ] **Step 4: Verify behavior change (GREEN)**

Describe the scenario again. Confirm the updated skill now requires an intent with `type = "intent"` addressed to `engram-agent` before the shutdown is sent.

- [ ] **Step 5: Pressure test**

Scenario: "`exec-1` just posted `done`. No holds. What does the lead do next?"

Expected: Post intent "Situation: exec-1 posted done. No holds remain. Behavior: Will send shutdown then kill pane." → wait ACK → send shutdown → kill-pane.

- [ ] **Step 6: Commit**

```bash
git add skills/engram-tmux-lead/SKILL.md
git commit -m "fix(engram-tmux-lead): require intent before killing agent panes (#507)

AI-Used: [claude]"
```

---

### Task 3: Add pre-respawn intent to Section 3.3 (Respawn Policy)

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md` (Section 3.3, lines ~485–489)

- [ ] **Step 1: Invoke `superpowers:writing-skills`**

Invoke the skill for this edit.

- [ ] **Step 2: Run baseline behavior test (RED)**

Describe this scenario: "engram-agent died. The lead is about to respawn it (attempt 1/3). Does the skill require an intent first?"

Read Section 3.3. Confirm: respawn procedure goes straight to "1. Kill existing pane…". No intent. **RED.**

- [ ] **Step 3: Add the respawn intent step**

In `skills/engram-tmux-lead/SKILL.md`, replace:

```
Respawn procedure:
1. Kill existing pane: `tmux kill-pane -t <pane-id> 2>/dev/null` then `tmux select-layout main-vertical` (single-column mode only — see Section 2.4).
```

with:

```
Respawn procedure:

**Step 0: Post respawn intent**

Post an intent before respawning. For engram-agent respawns, engram-agent is offline (dead), so apply the 5s implicit ACK rule — wait 5 seconds then proceed regardless of response. For task agent respawns where other agents are still running, use standard online/offline timing.

```toml
[[message]]
from = "lead"
to = "engram-agent"
thread = "lifecycle"
type = "intent"
ts = "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
text = """
Situation: <agent-name> is DEAD (respawn attempt N). Pane confirmed gone or unresponsive.
Behavior: Will kill existing pane and spawn a fresh instance with the same role parameters.
"""
```

1. Kill existing pane: `tmux kill-pane -t <pane-id> 2>/dev/null` then `tmux select-layout main-vertical` (single-column mode only — see Section 2.4).
```

**Note on double-intent:** The fresh spawn in step 2 of the respawn procedure goes through Section 2.1, which (after Task 1) will fire a second spawn intent. This is expected — the respawn intent covers the decision to kill and respawn a dead agent; the per-spawn intent in Section 2.1 covers the specific pane creation. Two intents, two distinct questions for engram-agent.

- [ ] **Step 4: Verify behavior change (GREEN)**

Confirm the respawn procedure now opens with a Step 0 intent block. Confirm the special case for engram-agent offline respawns (5s implicit ACK) is documented.

- [ ] **Step 5: Pressure test**

Scenario: "engram-agent died unexpectedly. Lead is triggering a respawn. What happens before `kill-pane`?"

Expected: Post intent "Situation: engram-agent is DEAD (respawn attempt 1). Behavior: Will kill existing pane and spawn fresh instance." Then wait 5s (offline rule since engram-agent is the recipient and it's dead). Then kill-pane.

- [ ] **Step 6: Commit**

```bash
git add skills/engram-tmux-lead/SKILL.md
git commit -m "fix(engram-tmux-lead): require intent before respawning agents (#507)

AI-Used: [claude]"
```

---

### Task 4: Add pre-shutdown intent to Section 3.4 (Shutdown)

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md` (Section 3.4, lines ~491–497)

- [ ] **Step 1: Invoke `superpowers:writing-skills`**

Invoke the skill for this edit.

- [ ] **Step 2: Run baseline behavior test (RED)**

Scenario: "User says 'stand down'. Does the current skill require the lead to post intent before invoking engram:engram-down?"

Read Section 3.4. Confirm: it says "Delegate to the `engram:engram-down` skill. Invoke it immediately." No intent step. **RED.**

- [ ] **Step 3: Add the session shutdown intent step**

In `skills/engram-tmux-lead/SKILL.md`, replace:

```
**Delegate to the `engram:engram-down` skill.** Invoke it immediately — it contains the full shutdown sequence including agent ordering, pane cleanup, background task draining, and session summary reporting.
```

with:

```
**Before delegating, post a session shutdown intent to `engram-agent`:**

```toml
[[message]]
from = "lead"
to = "engram-agent"
thread = "lifecycle"
type = "intent"
ts = "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
text = """
Situation: User requested session shutdown.
Behavior: Will invoke engram:engram-down skill to shut down all agents, drain background tasks, and report session summary.
"""
```

Wait for ACK from `engram-agent` before proceeding. Apply standard online/offline timing.

**Then delegate to the `engram:engram-down` skill.** It contains the full shutdown sequence including agent ordering, pane cleanup, background task draining, and session summary reporting.
```

- [ ] **Step 4: Verify behavior change (GREEN)**

Confirm Section 3.4 now opens with intent + ACK-wait before the engram-down delegation.

- [ ] **Step 5: Pressure test**

Scenario: "User says 'done, shut it down'. What does the lead do before invoking engram:engram-down?"

Expected: Post intent with "Situation: User requested session shutdown. Behavior: Will invoke engram:engram-down…" → wait ACK → invoke engram:engram-down.

- [ ] **Step 6: Commit**

```bash
git add skills/engram-tmux-lead/SKILL.md
git commit -m "fix(engram-tmux-lead): require intent before session shutdown (#507)

AI-Used: [claude]"
```

---

### Task 5: Add pre-routing intent to Section 4.1 (Routing Decision Table)

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md` (Section 4.1, lines ~630–647)

- [ ] **Step 1: Invoke `superpowers:writing-skills`**

Invoke the skill for this edit.

- [ ] **Step 2: Run baseline behavior test (RED)**

Scenario: "User says 'tackle issue #512'. The lead classifies this as Plan-Execute-Review. Does the current skill require posting intent before spawning the planner?"

Read Section 4.1. Confirm: the routing table says "Route" and "Agents Spawned" but says nothing about posting intent before acting. **RED.** (The Section 2.1 spawn intent from Task 1 covers individual spawns, but engram-agent also needs to weigh in on the routing *strategy* before any spawn happens.)

- [ ] **Step 3: Add the routing intent step**

In `skills/engram-tmux-lead/SKILL.md`, replace:

```
### 4.1 Routing Decision Table

Classify each user request and route accordingly. Use LLM judgment, not keyword matching.
```

with:

```
### 4.1 Routing Decision Table

Classify each user request and route accordingly. Use LLM judgment, not keyword matching.

**Before executing the routing decision, post a routing intent to `engram-agent`:**

```toml
[[message]]
from = "lead"
to = "engram-agent"
thread = "routing"
type = "intent"
ts = "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
text = """
Situation: User request: "<brief summary of user request>". Classifying as: <pattern from table below>.
Behavior: Will spawn <agent roles and count, e.g., "planner-3 then executor-4 then reviewer-2"> to handle this request.
"""
```

Wait for ACK from `engram-agent` before spawning any agents. Apply standard online/offline timing. Note: each individual spawn also requires a per-spawn intent (Section 2.1). The routing intent covers the strategic decision ("is this the right pattern?"); the spawn intents cover tactical execution ("is now the right time to spawn this agent?").
```

- [ ] **Step 4: Verify behavior change (GREEN)**

Confirm Section 4.1 now opens with a routing intent block before the table. Confirm the note distinguishes routing intent (strategic) from spawn intent (tactical).

- [ ] **Step 5: Pressure test**

Scenario: "User says 'research how the TF-IDF scoring works'. Lead classifies as Research. What happens before spawning researcher-1?"

Expected: Post routing intent "Situation: User request: 'research how TF-IDF scoring works'. Classifying as: Research. Behavior: Will spawn researcher-1." → wait ACK → then spawn intent per Section 2.1 "About to spawn researcher named researcher-1…" → ACK → spawn.

- [ ] **Step 6: Commit**

```bash
git add skills/engram-tmux-lead/SKILL.md
git commit -m "fix(engram-tmux-lead): require intent before routing decisions (#507)

AI-Used: [claude]"
```

---

### Task 6: Close issue and verify

**Files:**
- No file changes

- [ ] **Step 1: Re-read the issue requirements**

```bash
gh issue view 507 --repo toejough/engram
```

Verify all four Expected items from the issue are covered:
- Spawning a new agent → Task 1 (Section 2.1) + Task 5 (Section 4.1) ✓
- Killing/shutting down an agent → Task 2 (Section 3.1) ✓
- Routing a user request to a specific agent configuration → Task 5 (Section 4.1) ✓
- Making architectural decisions about agent topology → Task 5 (Section 4.1) ✓
- Bonus: respawn → Task 3 (Section 3.3) ✓
- Bonus: session shutdown → Task 4 (Section 3.4) ✓

- [ ] **Step 2: Close the issue**

```bash
gh issue close 507 --repo toejough/engram --comment "Fixed by adding five intent-posting steps to SKILL.md: pre-spawn (Section 2.1), pre-kill (Section 3.1), pre-respawn (Section 3.3), pre-session-shutdown (Section 3.4), and pre-routing (Section 4.1). Each requires ACK from engram-agent before the lifecycle action proceeds."
```
