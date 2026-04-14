# Lead Phase 5 Compatibility (#543) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix four conflicts between the engram-tmux-lead skill and the Phase 5 stateless engram-agent model: spurious SILENT nudge, false heartbeat threshold, missing binary-managed ACTIVE→SILENT state transition, and inability to distinguish binary-managed SILENT from truly-dead SILENT.

**Architecture:** Add a `last-silent-at` timestamp field to `AgentRecord` that the binary writes atomically alongside `state = "SILENT"` when entering the watch loop. The lead skill reads this field from `engram agent list` output and pairs it with a tmux pane-existence check to distinguish healthy idle SILENT (watch loop running) from dead SILENT (pane gone). The 6-min heartbeat check is removed; the nudge cycle is skipped for engram-agent in SILENT state.

**Tech Stack:** Go 1.23+, `engram/internal/agent` (AgentRecord), `engram/internal/cli` (cli_agent.go, export_test.go), Gomega test matchers, BurntSushi/toml v2. Use `targ test` and `targ check-full` for build operations — never `go test` directly.

---

## File Structure

| File | Role |
|------|------|
| `internal/agent/agent.go` | Add `LastSilentAt time.Time` to `AgentRecord` |
| `internal/agent/agent_test.go` | Add `TestAgentRecord_LastSilentAt_RoundTrips` |
| `internal/cli/cli_agent.go` | Add `writeAgentSilentState`; update `watchAndResume` |
| `internal/cli/cli_test.go` | Add two property tests for the SILENT timestamp |
| `skills/engram-tmux-lead/SKILL.md` | Update §3, §3.1, §3.2, §6.2 |

---

## Phase 1: Property Tests (RED)

### Task 1: Add `TestAgentRecord_LastSilentAt_RoundTrips`

**Files:**
- Modify: `internal/agent/agent_test.go`

- [ ] **Step 1: Write the failing test**

Add after `TestParseStateFile_WithAgents_RoundTrip` (around line 170):

```go
func TestAgentRecord_LastSilentAt_RoundTrips(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	lastSilentAt := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	original := agent.StateFile{
		Agents: []agent.AgentRecord{
			{
				Name:         "engram-agent",
				PaneID:       "main:1.1",
				SessionID:    "sess-xyz",
				State:        "SILENT",
				SpawnedAt:    time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC),
				LastSilentAt: lastSilentAt,
			},
		},
	}

	data, marshalErr := agent.MarshalStateFile(original)
	g.Expect(marshalErr).NotTo(HaveOccurred())

	if marshalErr != nil {
		return
	}

	got, parseErr := agent.ParseStateFile(data)
	g.Expect(parseErr).NotTo(HaveOccurred())

	if parseErr != nil {
		return
	}

	g.Expect(got.Agents).To(HaveLen(1))
	g.Expect(got.Agents[0].LastSilentAt.UTC()).To(Equal(lastSilentAt))
}
```

- [ ] **Step 2: Run test to confirm it fails**

```bash
targ test 2>&1 | grep -A5 "LastSilentAt"
```

Expected: compile error — `agent.AgentRecord` has no field `LastSilentAt`.

---

### Task 2: Add `TestOuterWatchLoop_WritesSilentAtWithSilentState`

**Files:**
- Modify: `internal/cli/cli_test.go`

- [ ] **Step 1: Write the failing test**

Add after `TestOuterWatchLoop_WriteStateSilentAfterSession` (around line 1204):

```go
func TestOuterWatchLoop_WritesSilentAtWithSilentState(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	stateFile := filepath.Join(dir, "state.toml")

	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	// Write initial state using MarshalStateFile for correct TOML keys.
	initialState := agentpkg.StateFile{
		Agents: []agentpkg.AgentRecord{
			{
				Name:      "engram-agent",
				PaneID:    "",
				SessionID: "",
				State:     "STARTING",
				SpawnedAt: time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC),
			},
		},
	}
	stateData, marshalErr := agentpkg.MarshalStateFile(initialState)
	g.Expect(marshalErr).NotTo(HaveOccurred())
	g.Expect(os.WriteFile(stateFile, stateData, 0o600)).To(Succeed())

	fakeClaude := filepath.Join(dir, "claude")
	doneJSON := `{"type":"assistant","session_id":"sess-abc",` +
		`"message":{"content":[{"type":"text","text":"DONE: All done."}]}}`
	script := "#!/bin/sh\nprintf '%s\\n' '" + doneJSON + "'\n"
	g.Expect(os.WriteFile(fakeClaude, []byte(script), 0o700)).To(Succeed())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	watchForIntent := func(_ context.Context, _, _ string, _ int) (chat.Message, int, error) {
		cancel()
		return chat.Message{}, 0, context.Canceled
	}

	err := cli.ExportRunConversationLoopWith(
		ctx,
		"engram-agent", "hello", chatFile, stateFile, fakeClaude,
		io.Discard,
		func(_ context.Context, _, _ string, _ int) (string, error) { return "Proceed.", nil },
		watchForIntent,
		func(_ string, _ int) ([]string, error) { return nil, nil },
	)
	g.Expect(err).NotTo(HaveOccurred())

	data, readErr := os.ReadFile(stateFile)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	// Both fields must be written atomically in the same RMW call.
	g.Expect(string(data)).To(ContainSubstring(`state = "SILENT"`))
	g.Expect(string(data)).To(ContainSubstring(`last-silent-at =`))
}
```

- [ ] **Step 2: Run test to confirm it fails**

```bash
targ test 2>&1 | grep -A5 "LastSilentAt\|last-silent-at"
```

Expected: compile error or test failure — `last-silent-at` is not written.

---

### Task 3: Add `TestRunAgentList_LastSilentAtIncludedInOutput`

**Files:**
- Modify: `internal/cli/cli_test.go`

- [ ] **Step 1: Write the failing test**

Add after `TestRunAgentList_MultipleAgents_NDJSON` (around line 2095):

```go
func TestRunAgentList_LastSilentAtIncludedInOutput(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.toml")

	lastSilentAt := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	state := agentpkg.StateFile{
		Agents: []agentpkg.AgentRecord{
			{
				Name:         "engram-agent",
				PaneID:       "main:1.1",
				SessionID:    "sess-xyz",
				State:        "SILENT",
				SpawnedAt:    time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC),
				LastSilentAt: lastSilentAt,
			},
		},
	}
	data, _ := agentpkg.MarshalStateFile(state)
	g.Expect(os.WriteFile(stateFile, data, 0o600)).To(Succeed())

	var stdout bytes.Buffer

	err := cli.Run([]string{"engram", "agent", "list", "--state-file", stateFile}, &stdout, io.Discard, nil)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	g.Expect(lines).To(HaveLen(1))

	var rec map[string]any

	g.Expect(json.Unmarshal([]byte(lines[0]), &rec)).To(Succeed())
	g.Expect(rec).To(HaveKey("last-silent-at"))
	g.Expect(rec["last-silent-at"]).NotTo(BeEmpty())
}
```

- [ ] **Step 2: Run all three RED tests together**

```bash
targ test 2>&1 | grep -E "FAIL|PASS|compile"
```

Expected: All three new tests fail (compile error or assertion failure). Existing tests pass.

---

## Phase 2: Implementation (GREEN)

### Task 4: Add `LastSilentAt` to `AgentRecord`

**Files:**
- Modify: `internal/agent/agent.go`

- [ ] **Step 1: Add the field**

In `internal/agent/agent.go`, locate `AgentRecord` (around line 18). Add `LastSilentAt` after `LastResumedAt`:

```go
type AgentRecord struct {
	Name           string    `json:"name"                     toml:"name"`
	PaneID         string    `json:"pane-id"                  toml:"pane-id"`
	SessionID      string    `json:"session-id"               toml:"session-id"`
	State          string    `json:"state"                    toml:"state"` // STARTING | ACTIVE | SILENT | DEAD
	SpawnedAt      time.Time `json:"spawned-at"               toml:"spawned-at"`
	LastResumedAt  time.Time `json:"last-resumed-at,omitzero" toml:"last-resumed-at,omitzero"`
	LastSilentAt   time.Time `json:"last-silent-at,omitzero"  toml:"last-silent-at,omitzero"`
	ArgumentWith   string    `json:"argument-with"            toml:"argument-with"`
	ArgumentCount  int       `json:"argument-count"           toml:"argument-count"`
	ArgumentThread string    `json:"argument-thread"          toml:"argument-thread"`
}
```

- [ ] **Step 2: Run agent tests green**

```bash
targ test 2>&1 | grep -E "agent_test|FAIL|ok"
```

Expected: `TestAgentRecord_LastSilentAt_RoundTrips` now passes.

---

### Task 5: Add `writeAgentSilentState` and update `watchAndResume`

**Files:**
- Modify: `internal/cli/cli_agent.go`

- [ ] **Step 1: Add `writeAgentSilentState` after `writeAgentState`**

In `internal/cli/cli_agent.go`, locate `writeAgentState` (around line 1359). Add the new function immediately after it:

```go
// writeAgentSilentState writes state=SILENT and last-silent-at atomically.
// Used by watchAndResume to record binary-managed SILENT transitions (Phase 5).
// Using a dedicated function prevents the two writes from being split across
// separate RMW calls, which would leave the state file inconsistent.
func writeAgentSilentState(stateFilePath, agentName string) error {
	return readModifyWriteStateFile(stateFilePath, func(stateFile agentpkg.StateFile) agentpkg.StateFile {
		for i, rec := range stateFile.Agents {
			if rec.Name == agentName {
				stateFile.Agents[i].State = "SILENT"
				stateFile.Agents[i].LastSilentAt = time.Now().UTC()

				return stateFile
			}
		}

		return stateFile
	})
}
```

- [ ] **Step 2: Update `watchAndResume` to call `writeAgentSilentState`**

In `watchAndResume` (around line 1301), replace:

```go
silentErr := writeAgentState(stateFilePath, agentName, "SILENT")
if silentErr != nil {
    _, _ = fmt.Fprintf(stdout,
        "[engram] warning: failed to write SILENT state: %v\n",
        silentErr)
}
```

With:

```go
silentErr := writeAgentSilentState(stateFilePath, agentName)
if silentErr != nil {
    _, _ = fmt.Fprintf(stdout,
        "[engram] warning: failed to write SILENT state: %v\n",
        silentErr)
}
```

- [ ] **Step 3: Run all tests green**

```bash
targ test
```

Expected: All three new tests pass. All existing tests pass. Zero failures.

- [ ] **Step 4: Commit Phase 1+2**

```bash
git add internal/agent/agent.go internal/agent/agent_test.go \
        internal/cli/cli_agent.go internal/cli/cli_test.go
git commit -m "$(cat <<'EOF'
feat(agent): add last-silent-at timestamp to binary-managed SILENT state (#543)

Binary now writes last-silent-at atomically alongside state=SILENT when
entering the Phase 5 watch loop. Lead skill can read this via engram agent
list to distinguish healthy idle SILENT from truly-dead SILENT.

AI-Used: [claude]
EOF
)"
```

---

## Phase 3: Simplify and Update Skill

### Task 6: Run `targ check-full` and fix any issues

**Files:** None modified (verification only)

- [ ] **Step 1: Run full check**

```bash
targ check-full 2>&1 | head -60
```

Expected: Zero lint/vet/coverage issues. If issues appear, fix them before proceeding. Do NOT suppress issues with `//nolint` unless the linter is provably wrong.

---

### Task 7: Update `skills/engram-tmux-lead/SKILL.md`

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md`

> **HARD RULE:** Use `superpowers:writing-skills` to edit this file. That skill enforces TDD on skill edits: baseline behavior test → edit → verify behavioral change. Do NOT edit SKILL.md directly without invoking the skill first.

Invoke `superpowers:writing-skills`. When prompted for what to change, provide the following four diffs:

---

**Change 1 — §3 State Machine: add binary-managed ACTIVE→SILENT transition**

In the ASCII state machine block (around line 418), add a new transition line after `STARTING ──(first chat message)──> ACTIVE`:

```
STARTING ──(first chat message)──> ACTIVE
ACTIVE ──(binary writes SILENT after session end)──> SILENT    [engram-agent Phase 5]
ACTIVE ──(no message for silence_threshold)──> SILENT
```

Update the transition count comment from "12 transitions (up from 6)." to "13 transitions (up from 6)."

---

**Change 2 — §3.1 SILENT Definition: distinguish binary-managed vs. timeout-driven**

Replace the SILENT row in the State Definitions table (around line 441):

OLD:
```
| **SILENT** | No chat message for `silence_threshold` (3 min for task agents, 6 min for engram-agent). Detected on 2-minute health check. | Nudge via chat + tmux (see 3.2). |
```

NEW:
```
| **SILENT** | No chat message for `silence_threshold` (3 min for task agents). **engram-agent (Phase 5):** binary writes SILENT after each watch-loop invocation ends — this is the normal idle state between intents. Detected on 2-minute health check. | **Task agents:** Nudge via chat + tmux (see 3.2). **engram-agent:** Check pane existence via `engram agent list` + tmux; if pane gone → DEAD → respawn. No nudge. |
```

---

**Change 3 — §3.2 Nudging: add engram-agent special case**

At the top of Section 3.2 (before "When an agent enters SILENT:"), insert this block:

```markdown
**Special case — engram-agent in Phase 5:**

In Phase 5, engram-agent's normal resting state between intents is SILENT. The binary writes `state = "SILENT"` AND `last-silent-at = <now>` atomically when the watch loop is idle. The pane remains alive as long as `engram agent run` is running.

**Do NOT send a chat or tmux nudge to engram-agent for SILENT state.** Instead, check pane existence:

```bash
AGENT_INFO=$(engram agent list | jq -r 'select(.name=="engram-agent")')
PANE_ID=$(echo "$AGENT_INFO" | jq -r '.["pane-id"]')
LAST_SILENT=$(echo "$AGENT_INFO" | jq -r '.["last-silent-at"]')

if tmux list-panes -F '#{pane_id}' | grep -q "$PANE_ID"; then
  # Pane alive → binary watch loop healthy → skip nudge
  # (Optional diagnostic log: "engram-agent healthy SILENT since $LAST_SILENT")
else
  # Pane gone → truly dead → transition immediately to DEAD → respawn (Section 3.3)
fi
```

For **task agents** (not engram-agent), use the standard nudge procedure below.
```

---

**Change 4 — §6.2 Health Check: replace heartbeat with pane-existence check**

In Section 6.2 (around line 970), replace the heartbeat item:

OLD:
```
4. If engram-agent missed heartbeat (>6 min since last), nudge immediately
```

NEW:
```
4. For engram-agent: check pane existence (§3.2 special case). If pane gone → DEAD → respawn. If pane alive → healthy SILENT, no action needed.
```

---

- [ ] **Step 2: Verify skill update passes writing-skills pressure tests**

The `superpowers:writing-skills` skill handles this — follow its TDD cycle to completion.

- [ ] **Step 3: Commit skill update**

```bash
git add skills/engram-tmux-lead/SKILL.md
git commit -m "$(cat <<'EOF'
fix(lead): update state machine and health check for Phase 5 stateless model (#543)

Four conflicts resolved:
1. SILENT nudge no longer fires on engram-agent (pane-existence check instead)
2. 6-min heartbeat threshold removed (inapplicable to stateless model)
3. Binary-managed ACTIVE→SILENT transition added to state machine
4. Pane-existence + last-silent-at distinguish healthy SILENT from dead SILENT

AI-Used: [claude]
EOF
)"
```

---

## Acceptance Criteria Verification

- [ ] `targ test` passes (all three new tests + all existing tests green)
- [ ] `targ check-full` passes (zero lint/coverage issues)
- [ ] State file after a session contains both `state = "SILENT"` and `last-silent-at =`
- [ ] `engram agent list` NDJSON includes `last-silent-at` for SILENT agents
- [ ] Lead skill §3 documents the binary-managed ACTIVE→SILENT transition
- [ ] Lead skill §3.2 skips nudge for engram-agent; uses pane-existence check
- [ ] Lead skill §6.2 no longer references 6-min heartbeat threshold
