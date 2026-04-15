# Hook-Based /prepare and /learn Nudges

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace passive skill-embedded /prepare and /learn reminders with active hook-based nudges at natural work boundaries.

**Architecture:** Two new shell hooks (Stop, UserPromptSubmit) inject static reminders into agent context. All self-query, reinforcement, and session-start memory surfacing for /prepare and /learn timing is removed from skills and hooks. The FormatResult framing line is also removed since it only supported the self-query mechanism.

**Tech Stack:** Bash hooks, Go (FormatResult change), SKILL.md edits

---

### Task 1: Add Stop hook with /learn nudge

**Files:**
- Create: `hooks/stop.sh`
- Modify: `hooks/hooks.json`

- [ ] **Step 1: Create `hooks/stop.sh`**

```bash
#!/usr/bin/env bash
set -euo pipefail

# Stop hook — nudge agent to consider /learn after completing work.

jq -n '{hookSpecificOutput: {hookEventName: "Stop", additionalContext: "You just finished responding. Consider: did you just complete a task, resolve a bug, change direction, or make a commit? If so, call /learn to capture what was discovered."}}'
```

- [ ] **Step 2: Make it executable**

Run: `chmod +x hooks/stop.sh`

- [ ] **Step 3: Wire into `hooks/hooks.json`**

Add `Stop` entry to hooks.json. The full file should be:

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
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "${CLAUDE_PLUGIN_ROOT}/hooks/stop.sh",
            "timeout": 5
          }
        ]
      }
    ]
  }
}
```

- [ ] **Step 4: Commit**

```bash
git add hooks/stop.sh hooks/hooks.json
git commit -m "feat(hooks): add Stop hook with /learn nudge"
```

### Task 2: Add UserPromptSubmit hook with /prepare nudge

**Files:**
- Create: `hooks/user-prompt-submit.sh`
- Modify: `hooks/hooks.json`

- [ ] **Step 1: Create `hooks/user-prompt-submit.sh`**

```bash
#!/usr/bin/env bash
set -euo pipefail

# UserPromptSubmit hook — nudge agent to consider /prepare before new work.

jq -n '{hookSpecificOutput: {hookEventName: "UserPromptSubmit", additionalContext: "A new user message just arrived. Consider: is this new work, a task switch, a new issue, or a debugging session? If so, call /prepare to load relevant context."}}'
```

- [ ] **Step 2: Make it executable**

Run: `chmod +x hooks/user-prompt-submit.sh`

- [ ] **Step 3: Wire into `hooks/hooks.json`**

Add `UserPromptSubmit` entry. The full file should be:

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
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "${CLAUDE_PLUGIN_ROOT}/hooks/stop.sh",
            "timeout": 5
          }
        ]
      }
    ],
    "UserPromptSubmit": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "${CLAUDE_PLUGIN_ROOT}/hooks/user-prompt-submit.sh",
            "timeout": 5
          }
        ]
      }
    ]
  }
}
```

- [ ] **Step 4: Commit**

```bash
git add hooks/user-prompt-submit.sh hooks/hooks.json
git commit -m "feat(hooks): add UserPromptSubmit hook with /prepare nudge"
```

### Task 3: Remove self-query and reinforcement from all 4 skills

**Files:**
- Modify: `skills/learn/SKILL.md`
- Modify: `skills/prepare/SKILL.md`
- Modify: `skills/recall/SKILL.md`
- Modify: `skills/remember/SKILL.md`

- [ ] **Step 1: Edit `skills/learn/SKILL.md`**

Remove the entire "Step 1: Self-query" section (lines 14-20) including the heading, explanation, and code block.

Remove the entire "Reinforce" section at the bottom (lines 83-88) including the heading and all bullet points.

Renumber remaining steps: Step 2 becomes Step 1, Step 3 becomes Step 2, etc.

- [ ] **Step 2: Edit `skills/prepare/SKILL.md`**

Remove the entire "Step 1: Self-query" section (lines 14-22) including the heading, explanation, and code block.

Remove the entire "Reinforce" section at the bottom (lines 50-55) including the heading and all bullet points.

Remove "Step 5: Internalize for your own use" (lines 46-48) — this was about internalizing self-query results.

Renumber remaining steps: Step 2 becomes Step 1, Step 3 becomes Step 2, Step 4 becomes Step 3.

- [ ] **Step 3: Edit `skills/recall/SKILL.md`**

Remove the entire "Self-query" section (lines 13-20) including the heading, explanation, and code block.

Remove the entire "Reinforce" section at the bottom (lines 45-50) including the heading and all bullet points.

- [ ] **Step 4: Edit `skills/remember/SKILL.md`**

Remove the entire "Step 1: Self-query" section (lines 13-20) including the heading, explanation, and code block.

Remove the entire "Reinforce" section at the bottom (lines 76-81) including the heading and all bullet points.

Renumber remaining steps: Step 2 becomes Step 1, Step 3 becomes Step 2, etc.

- [ ] **Step 5: Commit**

```bash
git add skills/learn/SKILL.md skills/prepare/SKILL.md skills/recall/SKILL.md skills/remember/SKILL.md
git commit -m "refactor(skills): remove self-query and reinforcement sections"
```

### Task 4: Strip session-start memory surfacing

**Files:**
- Modify: `hooks/session-start.sh`

- [ ] **Step 1: Simplify `hooks/session-start.sh`**

Remove the memory query block (lines 14-24). The sync portion should just output the static announcement. The new sync portion:

```bash
# --- Sync portion: announce skills ---
STATIC_MSG="[engram] Memory skills available. Call /prepare before starting new work. Call /learn after completing work. Call /recall to load previous session context. Call /remember to save something explicitly."

jq -n --arg ctx "$STATIC_MSG" \
    '{hookSpecificOutput: {hookEventName: "SessionStart", additionalContext: $ctx}}'
```

Keep the entire async build portion unchanged.

- [ ] **Step 2: Commit**

```bash
git add hooks/session-start.sh
git commit -m "refactor(hooks): remove memory surfacing from session-start"
```

### Task 5: Remove FormatResult framing line (TDD)

**Files:**
- Modify: `internal/recall/orchestrate_test.go:55-67`
- Modify: `internal/recall/orchestrate.go:312-315`

- [ ] **Step 1: Update test to expect output without framing line**

In `internal/recall/orchestrate_test.go`, change the "with memories" test expectation. Replace:

```go
		expected := "session content\n=== MEMORIES ===\n" +
				"These are standing user instructions. " +
				"Follow them with the same priority as direct requests.\n" +
				"memory1\nmemory2"
			g.Expect(buf.String()).To(Equal(expected))
```

With:

```go
		g.Expect(buf.String()).To(Equal("session content\n=== MEMORIES ===\nmemory1\nmemory2"))
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test 2>&1 | tail -20`
Expected: FAIL — actual output still has framing line.

- [ ] **Step 3: Revert FormatResult to simple format**

In `internal/recall/orchestrate.go`, replace:

```go
		_, err = fmt.Fprintf(w, "\n=== MEMORIES ===\n%s\n%s",
			"These are standing user instructions. Follow them with the same priority as direct requests.",
			result.Memories)
```

With:

```go
		_, err = fmt.Fprintf(w, "\n=== MEMORIES ===\n%s", result.Memories)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test 2>&1 | tail -20`
Expected: PASS

- [ ] **Step 5: Run full quality check**

Run: `targ check-full 2>&1 | tail -30`
Expected: No new failures.

- [ ] **Step 6: Rebuild binary**

Run: `go build -o ~/.claude/engram/bin/engram ./cmd/engram/`

- [ ] **Step 7: Commit**

```bash
git add internal/recall/orchestrate.go internal/recall/orchestrate_test.go
git commit -m "refactor(recall): remove framing line from FormatResult"
```

### Task 6: Manual E2E verification

- [ ] **Step 1: Verify Stop hook output**

Run: `bash hooks/stop.sh`
Expected: JSON with `additionalContext` containing the /learn nudge.

- [ ] **Step 2: Verify UserPromptSubmit hook output**

Run: `bash hooks/user-prompt-submit.sh`
Expected: JSON with `additionalContext` containing the /prepare nudge.

- [ ] **Step 3: Verify session-start no longer surfaces memories**

Run: `CLAUDE_PLUGIN_ROOT=/Users/joe/repos/personal/engram bash hooks/session-start.sh`
Expected: JSON with static skill announcement only, no `=== MEMORIES ===` section.

- [ ] **Step 4: Verify engram binary output has no framing line**

Run: `engram recall --memories-only --query "when to call /learn" 2>/dev/null | head -3`
Expected: `=== MEMORIES ===` followed directly by memory content, no framing line.
