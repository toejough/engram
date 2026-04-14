# Phase 3 — Agent Lifecycle Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace all bash-based agent spawning, killing, listing, and readiness-waiting in engram-tmux-lead with binary commands (`engram agent spawn/kill/list/wait-ready`). Ship a state file (`~/.local/share/engram/state/<slug>.toml`) that tracks agent registry and hold state as mutable binary-owned snapshot data. Fix the latent full-file TOML parse fragility in hold commands. Migrate engram-agent's main loop from fswatch to Background Monitor Pattern.

**Architecture:** New `internal/agent/` package for pure domain types (AgentRecord, HoldEntry, StateFile, pure mutations). OS wiring in `internal/cli/cli_agent.go`. State file R-M-W under a 5s lockfile. Hold state lives in the state file alongside agents so hold check/list never need full-file chat TOML parse. `engram agent wait-ready` wraps `FileWatcher.Watch` with `context.WithTimeout` for the watchDeadline contract.

**Spec:** `docs/superpowers/specs/engram-deterministic-coordination-design.md` §6.3, §7 (Phase 3)

**Codesign session:** planners 5 (arch lead), 6 (Go binary), 7 (skill coverage), 8 (phasing/prioritization) — 2026-04-06. All four perspectives converged before this plan was written.

**Tech Stack:** Go, BurntSushi/toml (already in go.mod), github.com/fsnotify/fsnotify (Phase 1), os/exec for tmux

---

## Pre-Flight (issue audit — do before starting tasks)

These are bookkeeping actions, not implementation work. Complete before Task 1.

| Issue | Action | Rationale |
|-------|--------|-----------|
| #509 ("engram-agent skill still uses shlock/heredoc for chat writes") | Close with note | Mislabeled: chat writes already correct (uses engram chat post). Shlock in skills is for memory file writes only, which is appropriate. |
| #520 (fswatch-1 main loop) | Confirm closed | Commit 3246f71 already shipped Background Monitor Pattern in engram-agent. Task 10B is verification only — confirm pattern is in place, mark #520 done, skip Steps 6–8. |
| #505 ("lead ignores two-column splitting rules") | Mark "closes with Phase 3" | Bug lives in SPAWN-PANE code that Phase 3 deletes entirely. |
| #506 ("lead miscounts panes") | Mark "closes with Phase 3" | Same rationale as #505. |
| #519 (ack-wait timeout) | Confirm closed | b22dc0c is merged. |
| #502 (ACK-wait sleep polling) | Confirm closed | Phase 2 shipped `engram chat ack-wait`. |
| #523 ("lead spawns monitor agents with improvised polling instead of engram chat watch") | Confirm closed | Current `engram-tmux-lead/SKILL.md` already uses Background Monitor Pattern (line 959), `engram chat watch` (line 1066), `CHAT_MONITOR_TASK_ID` variable, and drain-before-spawn pattern — the fix was shipped before Phase 3 launched. writing-skills RED step during Task 10A confirms no legacy polling remains. |

---

## E2E Acceptance Criteria

Phase 3 is done when all five criteria pass. These supersede the one-liner in the spec.

### Criterion 1: Binary in Real Session
Run `engram agent spawn/kill/list/wait-ready` in a real multi-agent session using the engram-tmux-lead skill. All four commands execute without error against a live tmux session. The lead skill successfully spawns two agents using binary commands (not SPAWN-PANE bash), waits for their ready signals via `engram agent wait-ready`, and kills them via `engram agent kill`.

**#503 auto-ACK observable check:** After `engram agent spawn` completes, inspect the chat file and confirm a `type = "ack"` (or `type = "hold-release"`) message was posted by the binary on behalf of the intent it emitted — with **no manual user action**. The intent and its ACK must both appear in the chat file, posted by `"system"`, before spawn returns. No user intervention required. (A partial fix — binary posts intent but ack-wait has wrong recipients or misparses the ACK — would still pass Criterion 1's basic test but fail this check.)

### Criterion 2: jq-Parseable Output
Parse all structured output with `jq` — the tool skills actually use:
- `engram agent list` output: `engram agent list | jq -r '.name'`
- `engram agent wait-ready` output: `engram agent wait-ready ... | jq -r '.cursor'`

No command **except `engram agent spawn`** should require `cut`/`awk` for structured fields. Spawn output is `pane-id|session-id` (pipe-delimited — codesign decision, locked; see Codesign Decisions table). _(Lesson from #516: TSV→NDJSON was a production catch, not a CI catch.)_

### Criterion 3: Historical Chat File
Run against a chat file with 3+ sessions of historical data (multi-thousand-line file). No parse errors. `hold check`/`hold list` must not fail due to historical TOML corruption.

### Criterion 4: Flag Validation
- `engram agent spawn` without `--name`: human-readable error
- `engram agent spawn` without `--prompt`: human-readable error
- `engram agent wait-ready` without `--name`: human-readable error
- `engram agent kill` without `--name`: human-readable error

Error messages match the quality of the #518 fix.

### Criterion 5: watchDeadline Fires
```bash
TMP=$(mktemp -d) && echo "" > "$TMP/chat.toml"
timeout 8 engram agent wait-ready --name nonexistent --cursor 0 --max-wait 3 --chat-file "$TMP/chat.toml"
```
Must exit within ~4s (not a 30s hang). Confirms the watchDeadline pattern flows through to the fsnotify loop.

---

## Codesign Decisions

These decisions were argued and resolved during planning. Do NOT revisit without reading the codesign-reassess thread (2026-04-06).

| Decision | Resolved | Rationale |
|----------|----------|-----------|
| `internal/agent/` new package for domain logic | Yes | Pure domain types, no I/O — same DI-at-edges pattern as `internal/chat/` |
| No `internal/tmux/` package | Yes | DI interface (`spawnerFunc`) lives in `cli_agent.go` — one-time use, premature abstraction if promoted |
| State file includes hold state alongside agent registry | Yes | Design once in Phase 3; retrofitting in Phase 4+ requires breaking schema change. Log (chat) vs snapshot (state file) separation. |
| NDJSON output for `engram agent list` | Yes | Lesson from #516: skills use `jq`. All structured binary output must be NDJSON. |
| `engram agent spawn` output: `pane-id\|session-id` | Yes | Two clean identifiers; pipe-delimited is fine. Only list/watch-result need NDJSON. |
| watchDeadline as first-class design contract | Yes | `context.WithTimeout` must flow to `watcher.Watch()` inner call. Not just outer loop. Same class as #519 bug. |
| State file lock timeout: 5s | Yes | R-M-W wider critical section than chat file append (1s). Concurrent spawning from lead risks spurious lock failures at 1s. |
| ParseMessagesSafe in `internal/chat/` | Yes | Fast path: full parse; fallback: per-block, warn on failures. `loadChatMessages` uses it. |
| `agent kill` calls domain functions directly | Yes | No self-invocation. `chat.ScanActiveHolds` + `chat.EvaluateCondition` + `poster.Post` composed internally. |
| `agent wait-ready` = thin wrapper over chat Watch | Yes | Watches for `type=ready` messages addressed to `--name`. Reuses WatchResult JSON format. |
| cli.go split before adding agent code | Yes | cli.go is 1006 lines; Phase 3 adds ~200 more. Split first, then add. |
| engram-agent main loop migration in Phase 3 PR | Yes | Skill-only change (~30 lines). Phase 3 already touches engram-tmux-lead. Adding engram-agent fix costs ~30 lines and prevents Phase 4 shipping with a broken watch mechanism. |
| State file reconstruction in Phase 3 (not deferred) | Yes (planner-9) | Spec §6.3 explicit: "cheap in Phase 3, expensive after Phase 5." Task 7 implements in-memory reconstruction via ScanActiveHolds + ready-message scan. Write-back deferred (requires live tmux cross-reference). |
| reconstructStateFileFromChat excludes done/shutdown agents | Yes (planner-9) | Agents that posted done or shutdown are excluded from reconstruction. Remaining agents tagged State=UNKNOWN (can't verify live status from chat alone without tmux). |
| AgentListArgs gets ChatFile field | Yes (planner-9) | runAgentList needs chat file for reconstruction fallback. Adds --chat-file flag (defaults to standard derived path). Consistent with spawn/kill/wait-ready pattern. |
| osStateFileLock as separate function (not parameterized osLockFile) | Yes | Two similar functions (chat: 200×5ms=1s; state: 200×25ms=5s) is acceptable. Three callers would warrant abstraction; two does not. CLAUDE.md: "three similar lines is better than premature abstraction." |
| testSpawner/testPaneKiller as package-level vars | Yes | Follows existing HoldNowFunc/AnthropicAPIURL pattern. No test-race risk in current test set (one parallel test per global). Flag for future: if >1 spawn test added, consider t.TempDir-keyed approach. |
| Speech-to-Chat NOT in Phase 3 | Yes | Phase 2's production bug count shows entangling internal/tmux + internal/streamjson risks dirty bisection. Keep phases separate. |
| STARTING→ACTIVE state transition: deferred to Phase 5 (planner-11/12) | Yes | Phase 3 state file tracks spawn identity (pane-id, session-id, state=STARTING). ACTIVE/SILENT/DEAD states are defined now for schema stability — same rationale as HoldEntry. Actual state transition logic deferred to Phase 5 (engram agent resume). Health monitoring remains skill-level in Phase 3–4. Implementer note: `engram agent list` will show state=STARTING for all agents until Phase 5. |
| `agent kill` returns error on unmet holds (planner-11) | Yes | A `lead-release` hold with unmet condition must block kill. Proceeding silently leaves dangling hold-acquire with no matching hold-release — breaks downstream hold consumers. Auto-evaluatable conditions (`done:`, `first-intent:`) release normally. |

---

## File Structure

```
internal/
  agent/
    agent.go          NEW — AgentRecord, HoldEntry, StateFile, ParseStateFile, MarshalStateFile, AddAgent, RemoveAgent, AddHold, RemoveHold
    agent_test.go     NEW — unit tests for all pure functions

  chat/
    watcher.go        MODIFY — add ParseMessagesSafe (fast path + per-block fallback)
    watcher_test.go   MODIFY — add ParseMessagesSafe tests

  cli/
    cli.go            MODIFY (split) — keep Run(), constants, errVars, shared adapters; remove chat/hold runners
    cli_chat.go       NEW (split from cli.go) — runChatDispatch, runChatPost, runChatWatch, runChatCursor, runChatAckWait
    cli_hold.go       NEW (split from cli.go) — runHoldDispatch, runHoldAcquire, runHoldRelease, runHoldList, runHoldCheck
    cli_agent.go      NEW — runAgentDispatch, runAgentSpawn, runAgentKill, runAgentList, runAgentWaitReady, osStateFileLock, readModifyWriteStateFile, deriveStateFilePath, resolveStateFile
    targets.go        MODIFY — add AgentSpawnArgs, AgentKillArgs, AgentListArgs, AgentWaitReadyArgs, flag builder functions, BuildAgentGroup
    export_test.go    MODIFY — export new test-visible symbols
    cli_test.go       MODIFY — add agent command tests

skills/
  engram-tmux-lead/SKILL.md   MODIFY — delete SPAWN-PANE/KILL-PANE/concurrency/pane-registry/layout-diagrams (~85 lines), add Agent Lifecycle section (~15 lines)
  engram-agent/SKILL.md       MODIFY — replace fswatch -1 main loop with Background Monitor Pattern (~30 lines replaced)
  engram-down/SKILL.md        MODIFY — replace tmux-based kill sequence with engram agent list/kill; fix CHAT_FSWATCH_TASK_ID → CHAT_MONITOR_TASK_ID (~15 lines)
```

---

## Task 1: ParseMessagesSafe in `internal/chat/`

**Files:**
- Modify: `internal/chat/watcher.go`
- Modify: `internal/chat/watcher_test.go`

This task fixes the latent TOML parse bug in `hold check`/`hold list`. Same attack surface as #515. The watcher fix used suffix-at-line (cursor-based) to avoid historical corruption. hold commands need ALL history, so they can't use suffix — they need ParseMessagesSafe instead.

- [ ] **Step 1: Write failing tests for ParseMessagesSafe**

```go
// internal/chat/watcher_test.go — add these tests

func TestParseMessagesSafe_CleanFile_AllMessages(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	data := []byte(`
[[message]]
from = "lead"
to = "all"
thread = "test"
type = "info"
ts = 2026-04-06T12:00:00Z
text = """hello"""

[[message]]
from = "executor"
to = "lead"
thread = "test"
type = "done"
ts = 2026-04-06T12:01:00Z
text = """done"""
`)
	msgs := chat.ParseMessagesSafe(data)
	g.Expect(msgs).To(HaveLen(2))
	g.Expect(msgs[0].From).To(Equal("lead"))
	g.Expect(msgs[1].From).To(Equal("executor"))
}

func TestParseMessagesSafe_OneCorruptBlock_OthersReturned(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Embed a null byte in the middle block to cause TOML parse failure.
	data := []byte("[[message]]\nfrom = \"lead\"\nto = \"all\"\nthread = \"t\"\ntype = \"info\"\nts = 2026-04-06T12:00:00Z\ntext = \"\"\"good\"\"\"\n\n[[message]]\nfrom = \"corrupt\"\nto = \"all\"\nthread = \"t\"\ntype = \"info\"\nts = 2026-04-06T12:01:00Z\ntext = \"\"\"bad\x00bytes\"\"\"\n\n[[message]]\nfrom = \"executor\"\nto = \"lead\"\nthread = \"t\"\ntype = \"done\"\nts = 2026-04-06T12:02:00Z\ntext = \"\"\"also good\"\"\"\n")

	msgs := chat.ParseMessagesSafe(data)
	froms := make([]string, 0, len(msgs))
	for _, m := range msgs {
		froms = append(froms, m.From)
	}
	g.Expect(froms).To(ContainElement("lead"))
	g.Expect(froms).To(ContainElement("executor"))
	g.Expect(froms).NotTo(ContainElement("corrupt"))
}

func TestParseMessagesSafe_EmptyData_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(chat.ParseMessagesSafe(nil)).To(BeEmpty())
	g.Expect(chat.ParseMessagesSafe([]byte(""))).To(BeEmpty())
}
```

- [ ] **Step 2: Run failing tests**

```bash
targ test
```

Expected: FAIL — `chat.ParseMessagesSafe undefined`

- [ ] **Step 3: Implement ParseMessagesSafe in `internal/chat/watcher.go`**

Add after the existing `ParseMessages` function:

```go
// ParseMessagesSafe deserializes TOML chat data tolerating per-message corruption.
// Fast path: attempts full-file ParseMessages (zero allocation overhead for clean files).
// Fallback: splits on [[message]] boundaries, parses each block independently.
// Corrupt blocks are logged via slog.Warn and skipped. Never returns an error.
func ParseMessagesSafe(data []byte) []Message {
	if len(data) == 0 {
		return nil
	}

	msgs, err := ParseMessages(data)
	if err == nil {
		return msgs
	}

	slog.Warn("ParseMessagesSafe: full parse failed, falling back to per-block parsing", "err", err)

	const boundary = "\n[[message]]"
	parts := bytes.Split(data, []byte(boundary))

	result := make([]Message, 0, len(parts))

	for i, part := range parts {
		var block []byte

		if i == 0 {
			block = bytes.TrimSpace(part)
			if len(block) == 0 {
				continue
			}
		} else {
			block = append([]byte("[[message]]"), part...)
		}

		blockMsgs, blockErr := ParseMessages(block)
		if blockErr != nil {
			slog.Warn("ParseMessagesSafe: skipping corrupt block", "block_index", i, "err", blockErr)
			continue
		}

		result = append(result, blockMsgs...)
	}

	return result
}
```

`bytes` is already imported in `watcher.go` (used by `bytes.Count` and `bytes.Split`).

- [ ] **Step 4: Run tests to verify pass**

```bash
targ test
```

Expected: PASS

- [ ] **Step 5: Update `loadChatMessages` in `internal/cli/cli.go` to use ParseMessagesSafe**

```go
// loadChatMessages reads and parses a TOML chat file using the provided readFile func.
// Uses ParseMessagesSafe to tolerate per-message corruption (same attack surface as #515).
// Returns nil slice (no error) when the file does not exist.
func loadChatMessages(chatFilePath string, readFile func(string) ([]byte, error)) ([]chat.Message, error) {
	data, err := readFile(chatFilePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("reading chat file: %w", err)
	}

	return chat.ParseMessagesSafe(data), nil
}
```

- [ ] **Step 6: Run full check**

```bash
targ check-full
```

Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/chat/watcher.go internal/chat/watcher_test.go internal/cli/cli.go
git commit -m "feat(chat): add ParseMessagesSafe with per-block fallback for corrupt TOML

hold check/list call loadChatMessages which previously used ParseMessages — a
full-file TOML unmarshal that fails completely if any historical message block
is corrupt. Same attack surface as #515 (fixed for Watch via suffix parsing).

ParseMessagesSafe fast-paths to ParseMessages for clean files, then falls back
to per-block parsing on failure. loadChatMessages now uses ParseMessagesSafe."
```

---

## Task 2: `internal/agent/` — Pure Domain Package

**Files:**
- Create: `internal/agent/agent.go`
- Create: `internal/agent/agent_test.go`

Pure domain types and pure mutations. No I/O. Follows `internal/chat/` pattern. State file holds both agent registry (`[[agent]]`) and hold registry (`[[hold]]`) so Phase 3 gives hold check/list a state file authority path.

- [ ] **Step 1: Write failing tests**

```go
// internal/agent/agent_test.go

package agent_test

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/agent"
)

func TestParseStateFile_Empty_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	sf, err := agent.ParseStateFile(nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(sf.Agents).To(BeEmpty())
	g.Expect(sf.Holds).To(BeEmpty())
}

func TestParseStateFile_WithAgents_RoundTrip(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	spawnedAt := time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC)
	original := agent.StateFile{
		Agents: []agent.AgentRecord{
			{
				Name:           "executor-1",
				PaneID:         "main:1.2",
				SessionID:      "abc123",
				State:          "ACTIVE",
				SpawnedAt:      spawnedAt,
				ArgumentWith:   "",
				ArgumentCount:  0,
				ArgumentThread: "",
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
	g.Expect(got.Agents[0].Name).To(Equal("executor-1"))
	g.Expect(got.Agents[0].PaneID).To(Equal("main:1.2"))
	g.Expect(got.Agents[0].State).To(Equal("ACTIVE"))
	g.Expect(got.Agents[0].SpawnedAt.UTC()).To(Equal(spawnedAt))
}

func TestParseStateFile_WithHolds_RoundTrip(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	acquiredAt := time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC)
	original := agent.StateFile{
		Holds: []agent.HoldEntry{
			{
				HoldID:     "uuid-1234",
				Holder:     "lead",
				Target:     "executor-1",
				Condition:  "lead-release:phase3",
				Tag:        "phase3",
				AcquiredTS: acquiredAt,
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

	g.Expect(got.Holds).To(HaveLen(1))
	g.Expect(got.Holds[0].HoldID).To(Equal("uuid-1234"))
	g.Expect(got.Holds[0].AcquiredTS.UTC()).To(Equal(acquiredAt))
}

func TestAddAgent_AppendsToEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	sf := agent.StateFile{}
	rec := agent.AgentRecord{Name: "planner-1", State: "STARTING"}
	got := agent.AddAgent(sf, rec)
	g.Expect(got.Agents).To(HaveLen(1))
	g.Expect(got.Agents[0].Name).To(Equal("planner-1"))
	g.Expect(sf.Agents).To(BeEmpty()) // original unchanged
}

func TestRemoveAgent_RemovesNamedAgent(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	sf := agent.StateFile{
		Agents: []agent.AgentRecord{
			{Name: "executor-1"},
			{Name: "reviewer-1"},
		},
	}
	got := agent.RemoveAgent(sf, "executor-1")
	g.Expect(got.Agents).To(HaveLen(1))
	g.Expect(got.Agents[0].Name).To(Equal("reviewer-1"))
}

func TestRemoveAgent_MissingName_NoChange(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	sf := agent.StateFile{Agents: []agent.AgentRecord{{Name: "executor-1"}}}
	got := agent.RemoveAgent(sf, "nonexistent")
	g.Expect(got.Agents).To(HaveLen(1))
}

func TestAddHold_AppendsHold(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	sf := agent.StateFile{}
	got := agent.AddHold(sf, agent.HoldEntry{HoldID: "h1", Target: "executor-1"})
	g.Expect(got.Holds).To(HaveLen(1))
	g.Expect(got.Holds[0].HoldID).To(Equal("h1"))
}

func TestRemoveHold_RemovesById(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	sf := agent.StateFile{
		Holds: []agent.HoldEntry{{HoldID: "h1"}, {HoldID: "h2"}},
	}
	got := agent.RemoveHold(sf, "h1")
	g.Expect(got.Holds).To(HaveLen(1))
	g.Expect(got.Holds[0].HoldID).To(Equal("h2"))
}
```

- [ ] **Step 2: Run failing tests**

```bash
targ test
```

Expected: FAIL — `engram/internal/agent` package not found

- [ ] **Step 3: Create `internal/agent/agent.go`**

```go
// Package agent provides pure domain types for the engram agent lifecycle.
// All functions are pure (no I/O). OS wiring lives in internal/cli.
package agent

import (
	"bytes"
	"fmt"
	"time"

	"github.com/BurntSushi/toml"
)

// AgentRecord holds binary bookkeeping for a spawned agent (spec §6.3).
// Argument state fields enforce the 3-argument cap from SPEECH-2/SKILL-2
// and must be persisted across engram agent resume invocations (Phase 5).
//
//nolint:tagliatelle // state file uses kebab-case field names
type AgentRecord struct {
	Name           string    `toml:"name"            json:"name"`
	PaneID         string    `toml:"pane-id"          json:"pane-id"`
	SessionID      string    `toml:"session-id"       json:"session-id"`
	State          string    `toml:"state"            json:"state"` // STARTING | ACTIVE | SILENT | DEAD
	SpawnedAt      time.Time `toml:"spawned-at"       json:"spawned-at"`
	LastResumedAt  time.Time `toml:"last-resumed-at,omitempty" json:"last-resumed-at,omitempty"`
	ArgumentWith   string    `toml:"argument-with"    json:"argument-with"`
	ArgumentCount  int       `toml:"argument-count"   json:"argument-count"`
	ArgumentThread string    `toml:"argument-thread"  json:"argument-thread"`
}

// HoldEntry is the state-file representation of an active hold.
// Mirrors chat.HoldRecord but stored as TOML struct (not JSON-in-text).
// The state file is the snapshot authority for hold state after Phase 3.
//
//nolint:tagliatelle // state file uses kebab-case field names
type HoldEntry struct {
	HoldID     string    `toml:"hold-id"`
	Holder     string    `toml:"holder"`
	Target     string    `toml:"target"`
	Condition  string    `toml:"condition,omitempty"`
	Tag        string    `toml:"tag,omitempty"`
	AcquiredTS time.Time `toml:"acquired-ts"`
}

// StateFile holds the parsed contents of the state TOML file.
// The binary is the sole writer — no skill writes this file directly.
type StateFile struct {
	Agents []AgentRecord `toml:"agent"`
	Holds  []HoldEntry   `toml:"hold"`
}

// ParseStateFile deserializes TOML state file data.
// Returns an empty StateFile for nil or empty data.
func ParseStateFile(data []byte) (StateFile, error) {
	if len(data) == 0 {
		return StateFile{}, nil
	}

	var sf StateFile
	if err := toml.Unmarshal(data, &sf); err != nil {
		return StateFile{}, fmt.Errorf("parsing state file: %w", err)
	}

	return sf, nil
}

// MarshalStateFile serializes a StateFile to TOML bytes.
func MarshalStateFile(sf StateFile) ([]byte, error) {
	var buf bytes.Buffer

	if err := toml.NewEncoder(&buf).Encode(sf); err != nil {
		return nil, fmt.Errorf("marshaling state file: %w", err)
	}

	return buf.Bytes(), nil
}

// AddAgent returns a new StateFile with record appended to Agents.
// The original StateFile is not modified (pure function).
func AddAgent(sf StateFile, record AgentRecord) StateFile {
	result := sf
	result.Agents = append(append(make([]AgentRecord, 0, len(sf.Agents)+1), sf.Agents...), record)

	return result
}

// RemoveAgent returns a new StateFile with the named agent removed from Agents.
// If no agent with that name exists, returns sf unchanged.
// The original StateFile is not modified (pure function).
func RemoveAgent(sf StateFile, name string) StateFile {
	filtered := make([]AgentRecord, 0, len(sf.Agents))

	for _, rec := range sf.Agents {
		if rec.Name != name {
			filtered = append(filtered, rec)
		}
	}

	result := sf
	result.Agents = filtered

	return result
}

// AddHold returns a new StateFile with hold appended to Holds.
// The original StateFile is not modified (pure function).
func AddHold(sf StateFile, hold HoldEntry) StateFile {
	result := sf
	result.Holds = append(append(make([]HoldEntry, 0, len(sf.Holds)+1), sf.Holds...), hold)

	return result
}

// RemoveHold returns a new StateFile with the hold identified by holdID removed.
// If no hold with that ID exists, returns sf unchanged.
// The original StateFile is not modified (pure function).
func RemoveHold(sf StateFile, holdID string) StateFile {
	filtered := make([]HoldEntry, 0, len(sf.Holds))

	for _, hold := range sf.Holds {
		if hold.HoldID != holdID {
			filtered = append(filtered, hold)
		}
	}

	result := sf
	result.Holds = filtered

	return result
}
```

- [ ] **Step 4: Run tests to verify pass**

```bash
targ check-full
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/agent/
git commit -m "feat(agent): add internal/agent package with pure domain types

AgentRecord, HoldEntry, StateFile, ParseStateFile, MarshalStateFile, and
pure mutation functions (AddAgent, RemoveAgent, AddHold, RemoveHold).
No I/O — follows internal/chat/ pattern.

State file includes both agent registry ([[agent]]) and hold registry
([[hold]]) so hold check/list can read from state file instead of doing
full-file chat TOML parse. Designed once in Phase 3 to avoid retrofitting
a breaking schema change in later phases."
```

---

## Task 3: Split `cli.go` (Pre-Phase-3 Housekeeping)

**Files:**
- Modify: `internal/cli/cli.go` (retain Run(), constants, errVars, shared adapters only)
- Create: `internal/cli/cli_chat.go` (move runChatDispatch and its helpers)
- Create: `internal/cli/cli_hold.go` (move runHoldDispatch and its helpers)

Pure refactor. No new functionality. All existing tests must pass unchanged after the split.

- [ ] **Step 1: Create `internal/cli/cli_chat.go`**

New file with package declaration and exactly these moved functions (copy verbatim from cli.go):
- `runChatAckWait`
- `runChatCursor`
- `runChatDispatch`
- `runChatPost`
- `runChatWatch`
- `marshalAndWriteWatchResult`
- `watchResult` type definition

File header:
```go
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"engram/internal/chat"
)
```

Remove any imports not used by the moved functions. Add any needed that were previously satisfied by cli.go's import block.

- [ ] **Step 2: Create `internal/cli/cli_hold.go`**

Move these functions from cli.go:
- `runHoldAcquire`
- `runHoldCheck`
- `runHoldDispatch`
- `runHoldList`
- `runHoldRelease`
- `filterHolds`
- `marshalReleasePayload`

File header:
```go
package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"engram/internal/chat"
)
```

- [ ] **Step 3: Remove moved functions from `cli.go`**

After moving, cli.go retains:
- Package declaration and imports (trim to only what remains)
- `AnthropicAPIURL` and `HoldNowFunc` exported vars
- `Run()` — add `"agent"` dispatch case here (pointing to `runAgentDispatch` from cli_agent.go Task 5)
- All unexported constants (`anthropicMaxTokens`, `chatDirMode`, `chatFileMode`, `lockRetryDelay`, `maxLockRetries`, `minArgs`, uuid constants)
- All unexported error vars (`errUsage`, `errUnknownCommand`, `errLockTimeout`, `errAgentRequired`, `errRecipientsRequired`, etc.)
- `applyDataDirDefault`, `applyProjectSlugDefault`
- `deriveChatFilePath`, `resolveChatFile`
- `loadChatMessages`
- `newFilePoster`, `newFileWatcher`
- `newFlagSet`
- `osLockFile`
- `generateUUIDv4`
- `makeAnthropicCaller`
- `runRecall`, `runShow` (or move to show.go if preferred)
- `signalContext`

Add the agent case to `Run()`:
```go
case "agent":
    return runAgentDispatch(subArgs, stdout)
```

- [ ] **Step 4: Verify no regressions**

```bash
targ check-full
```

Expected: PASS — all existing tests pass; only file locations changed

- [ ] **Step 5: Commit**

```bash
git add internal/cli/cli.go internal/cli/cli_chat.go internal/cli/cli_hold.go
git commit -m "refactor(cli): split cli.go into cli_chat.go and cli_hold.go

cli.go was 1006 lines. Phase 3 adds ~200 lines of agent wiring. Split into
focused files before adding new code to keep each file under ~400 lines.
No behavior changes — pure move of existing functions.

Also adds 'agent' dispatch case to Run() for Task 5."
```

---

## Task 4: `targets.go` — Agent Command Arg Types

**Files:**
- Modify: `internal/cli/targets.go`
- Modify: `internal/cli/targets_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/cli/targets_test.go — add

func TestAgentSpawnFlags_AllFields(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	args := cli.AgentSpawnFlags(cli.AgentSpawnArgs{
		Name:      "executor-1",
		Prompt:    "You are an executor agent.",
		ChatFile:  "/tmp/chat.toml",
		StateFile: "/tmp/state.toml",
	})
	g.Expect(args).To(ContainElements(
		"--name", "executor-1",
		"--prompt", "You are an executor agent.",
		"--chat-file", "/tmp/chat.toml",
		"--state-file", "/tmp/state.toml",
	))
}

func TestAgentKillFlags_AllFields(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	args := cli.AgentKillFlags(cli.AgentKillArgs{
		Name:      "executor-1",
		ChatFile:  "/tmp/chat.toml",
		StateFile: "/tmp/state.toml",
	})
	g.Expect(args).To(ContainElements(
		"--name", "executor-1",
		"--chat-file", "/tmp/chat.toml",
		"--state-file", "/tmp/state.toml",
	))
}

func TestAgentListFlags_AllFields(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	args := cli.AgentListFlags(cli.AgentListArgs{StateFile: "/tmp/state.toml", ChatFile: "/tmp/chat.toml"})
	g.Expect(args).To(ContainElements("--state-file", "/tmp/state.toml", "--chat-file", "/tmp/chat.toml"))
}

func TestAgentWaitReadyFlags_AllFields(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	args := cli.AgentWaitReadyFlags(cli.AgentWaitReadyArgs{
		Name:     "executor-1",
		Cursor:   42,
		MaxWait:  30,
		ChatFile: "/tmp/chat.toml",
	})
	g.Expect(args).To(ContainElements(
		"--name", "executor-1",
		"--cursor", "42",
		"--max-wait", "30",
		"--chat-file", "/tmp/chat.toml",
	))
}
```

- [ ] **Step 2: Run failing tests**

```bash
targ test
```

Expected: FAIL — types undefined

- [ ] **Step 3: Add arg types and flag builders to `targets.go`**

```go
// AgentSpawnArgs holds flags for `engram agent spawn`.
type AgentSpawnArgs struct {
	Name      string
	Prompt    string
	ChatFile  string
	StateFile string
}

// AgentKillArgs holds flags for `engram agent kill`.
type AgentKillArgs struct {
	Name      string
	ChatFile  string
	StateFile string
}

// AgentListArgs holds flags for `engram agent list`.
type AgentListArgs struct {
	StateFile string
	ChatFile  string // used for reconstruction fallback when state file is missing
}

// AgentWaitReadyArgs holds flags for `engram agent wait-ready`.
type AgentWaitReadyArgs struct {
	Name     string
	Cursor   int
	MaxWait  int // seconds
	ChatFile string
}

// AgentSpawnFlags returns the CLI flag args for the agent spawn subcommand.
func AgentSpawnFlags(a AgentSpawnArgs) []string {
	return BuildFlags(
		"--name", a.Name,
		"--prompt", a.Prompt,
		"--chat-file", a.ChatFile,
		"--state-file", a.StateFile,
	)
}

// AgentKillFlags returns the CLI flag args for the agent kill subcommand.
func AgentKillFlags(a AgentKillArgs) []string {
	return BuildFlags("--name", a.Name, "--chat-file", a.ChatFile, "--state-file", a.StateFile)
}

// AgentListFlags returns the CLI flag args for the agent list subcommand.
func AgentListFlags(a AgentListArgs) []string {
	return BuildFlags("--state-file", a.StateFile, "--chat-file", a.ChatFile)
}

// AgentWaitReadyFlags returns the CLI flag args for the agent wait-ready subcommand.
func AgentWaitReadyFlags(a AgentWaitReadyArgs) []string {
	return BuildFlags(
		"--name", a.Name,
		"--cursor", strconv.Itoa(a.Cursor),
		"--max-wait", strconv.Itoa(a.MaxWait),
		"--chat-file", a.ChatFile,
	)
}
```

Add `"strconv"` to `targets.go` imports if not already present.

- [ ] **Step 4: Run tests to verify pass**

```bash
targ check-full
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/cli/targets.go internal/cli/targets_test.go
git commit -m "feat(cli): add agent command arg types and flag builders to targets.go"
```

---

## Task 5: State File Infrastructure in `cli_agent.go`

**Files:**
- Create: `internal/cli/cli_agent.go`
- Modify: `internal/cli/export_test.go`
- Modify: `internal/cli/cli_test.go`

Creates the file skeleton with path resolution, lock, and R-M-W helper. The four command runners follow in Tasks 6–9.

- [ ] **Step 1: Write failing tests**

```go
// internal/cli/cli_test.go — add
// (import agentpkg "engram/internal/agent" at top of file)

func TestResolveStateFile_WithOverride_ReturnsOverride(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.toml")

	path, err := cli.ExportResolveStateFile(stateFile, "agent spawn", os.UserHomeDir, os.Getwd)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(path).To(Equal(stateFile))
}

func TestResolveStateFile_NoOverride_UsesStateSubdir(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	path, err := cli.ExportResolveStateFile("", "agent spawn", os.UserHomeDir, os.Getwd)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(path).To(ContainSubstring("/state/"))
	g.Expect(path).To(HaveSuffix(".toml"))
}

func TestReadModifyWriteStateFile_CreatesFileWhenAbsent(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.toml")

	err := cli.ExportReadModifyWriteStateFile(stateFile, func(sf agentpkg.StateFile) agentpkg.StateFile {
		return agentpkg.AddAgent(sf, agentpkg.AgentRecord{Name: "test-agent", State: "STARTING"})
	})
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	data, readErr := os.ReadFile(stateFile)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(data)).To(ContainSubstring("test-agent"))
}

func TestReadModifyWriteStateFile_UpdatesExistingFile(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.toml")

	err1 := cli.ExportReadModifyWriteStateFile(stateFile, func(sf agentpkg.StateFile) agentpkg.StateFile {
		return agentpkg.AddAgent(sf, agentpkg.AgentRecord{Name: "agent-1", State: "ACTIVE"})
	})
	g.Expect(err1).NotTo(HaveOccurred())

	err2 := cli.ExportReadModifyWriteStateFile(stateFile, func(sf agentpkg.StateFile) agentpkg.StateFile {
		return agentpkg.AddAgent(sf, agentpkg.AgentRecord{Name: "agent-2", State: "ACTIVE"})
	})
	g.Expect(err2).NotTo(HaveOccurred())

	data, _ := os.ReadFile(stateFile)
	g.Expect(string(data)).To(ContainSubstring("agent-1"))
	g.Expect(string(data)).To(ContainSubstring("agent-2"))
}

func TestReadModifyWriteStateFile_ConcurrentCallers_BothAgentsPresent(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.toml")

	const callers = 5
	var wg sync.WaitGroup
	wg.Add(callers)

	for i := range callers {
		agentName := fmt.Sprintf("agent-%d", i)
		go func() {
			defer wg.Done()
			innerErr := cli.ExportReadModifyWriteStateFile(stateFile, func(sf agentpkg.StateFile) agentpkg.StateFile {
				return agentpkg.AddAgent(sf, agentpkg.AgentRecord{Name: agentName, State: "STARTING"})
			})
			g.Expect(innerErr).NotTo(HaveOccurred())
		}()
	}
	wg.Wait()

	data, readErr := os.ReadFile(stateFile)
	g.Expect(readErr).NotTo(HaveOccurred())
	for i := range callers {
		g.Expect(string(data)).To(ContainSubstring(fmt.Sprintf("agent-%d", i)))
	}
}
```

Add `"sync"` and `"fmt"` to imports if not already present in cli_test.go.

- [ ] **Step 2: Add exports to `export_test.go`**

```go
// internal/cli/export_test.go — add to var block

ExportResolveStateFile = func(override, cmd string, homeDir func() (string, error), getwd func() (string, error)) (string, error) {
    return resolveStateFile(override, cmd, homeDir, getwd)
}

ExportReadModifyWriteStateFile = func(path string, modify func(agentpkg.StateFile) agentpkg.StateFile) error {
    return readModifyWriteStateFile(path, modify)
}
```

Add `agentpkg "engram/internal/agent"` to export_test.go imports.

- [ ] **Step 3: Run failing tests**

```bash
targ test
```

Expected: FAIL — functions undefined

- [ ] **Step 4: Create `internal/cli/cli_agent.go`**

```go
package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"context"

	agentpkg "engram/internal/agent"
	"engram/internal/chat"
)

// stateFileLockRetries × stateFileLockDelay = 5s total timeout for state file R-M-W.
// Chat file append uses maxLockRetries=200 × lockRetryDelay=5ms = 1s.
// State file R-M-W is wider (read + marshal + write vs append-only), so we use 5s.
const (
	stateFileLockRetries = 200
	stateFileLockDelay   = 25 * time.Millisecond
)

// testSpawner and testPaneKiller are overridden in tests to avoid real tmux invocations.
var (
	testSpawner    func(name, prompt string) (paneID, sessionID string, err error) //nolint:gochecknoglobals
	testPaneKiller func(paneID string) error                                       //nolint:gochecknoglobals
)

// deriveStateFilePath mirrors deriveChatFilePath but uses the "state" subdirectory.
func deriveStateFilePath(override string, homeDir func() (string, error), getwd func() (string, error)) (string, error) {
	if override != "" {
		return override, nil
	}

	home, err := homeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}

	cwd, cwdErr := getwd()
	if cwdErr != nil {
		return "", fmt.Errorf("resolving working directory: %w", cwdErr)
	}

	return filepath.Join(DataDirFromHome(home, os.Getenv), "state", ProjectSlugFromPath(cwd)+".toml"), nil
}

// resolveStateFile derives the state file path, wrapping errors with the subcommand name.
func resolveStateFile(override, cmd string, homeDir func() (string, error), getwd func() (string, error)) (string, error) {
	path, err := deriveStateFilePath(override, homeDir, getwd)
	if err != nil {
		return "", fmt.Errorf("%s: %w", cmd, err)
	}

	return path, nil
}

// osStateFileLock acquires a lockfile with a 5s timeout for the wider R-M-W critical section.
func osStateFileLock(name string) (func() error, error) {
	for range stateFileLockRetries {
		f, err := os.OpenFile(name, os.O_CREATE|os.O_EXCL, chatFileMode) //nolint:gosec
		if err == nil {
			return func() error {
				_ = f.Close()

				return os.Remove(name)
			}, nil
		}

		if !os.IsExist(err) {
			return nil, fmt.Errorf("creating state lock: %w", err)
		}

		time.Sleep(stateFileLockDelay)
	}

	return nil, fmt.Errorf("state file lock timeout after 5s")
}

// readModifyWriteStateFile performs a locked read-modify-write on the state file.
// Creates the file and its parent directory if they do not exist.
func readModifyWriteStateFile(stateFilePath string, modify func(agentpkg.StateFile) agentpkg.StateFile) error {
	lockPath := stateFilePath + ".lock"

	unlock, lockErr := osStateFileLock(lockPath)
	if lockErr != nil {
		return fmt.Errorf("acquiring state file lock: %w", lockErr)
	}

	defer func() { _ = unlock() }()

	data, readErr := os.ReadFile(stateFilePath)
	if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
		return fmt.Errorf("reading state file: %w", readErr)
	}

	sf, parseErr := agentpkg.ParseStateFile(data)
	if parseErr != nil {
		return fmt.Errorf("parsing state file: %w", parseErr)
	}

	sf = modify(sf)

	newData, marshalErr := agentpkg.MarshalStateFile(sf)
	if marshalErr != nil {
		return fmt.Errorf("marshaling state file: %w", marshalErr)
	}

	dir := filepath.Dir(stateFilePath)
	if mkdirErr := os.MkdirAll(dir, chatDirMode); mkdirErr != nil {
		return fmt.Errorf("creating state directory: %w", mkdirErr)
	}

	if writeErr := os.WriteFile(stateFilePath, newData, chatFileMode); writeErr != nil {
		return fmt.Errorf("writing state file: %w", writeErr)
	}

	return nil
}

// runAgentDispatch routes agent subcommands (spawn|kill|list|wait-ready).
func runAgentDispatch(subArgs []string, stdout io.Writer) error {
	if len(subArgs) < 1 {
		return fmt.Errorf("%w: agent requires a subcommand (spawn|kill|list|wait-ready)", errUsage)
	}

	switch subArgs[0] {
	case "spawn":
		return runAgentSpawn(subArgs[1:], stdout)
	case "kill":
		return runAgentKill(subArgs[1:], stdout)
	case "list":
		return runAgentList(subArgs[1:], stdout)
	case "wait-ready":
		return runAgentWaitReady(subArgs[1:], stdout)
	default:
		return fmt.Errorf("%w: agent %s", errUnknownCommand, subArgs[0])
	}
}

// Stub runners — replaced in Tasks 6–9.
func runAgentSpawn(_ []string, _ io.Writer) error    { return fmt.Errorf("not implemented") }
func runAgentKill(_ []string, _ io.Writer) error     { return fmt.Errorf("not implemented") }
func runAgentList(_ []string, _ io.Writer) error     { return fmt.Errorf("not implemented") }
func runAgentWaitReady(_ []string, _ io.Writer) error { return fmt.Errorf("not implemented") }
```

- [ ] **Step 5: Run tests to verify pass**

```bash
targ check-full
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/cli/cli_agent.go internal/cli/export_test.go internal/cli/cli_test.go
git commit -m "feat(cli): add state file infrastructure in cli_agent.go

Adds deriveStateFilePath (mirrors deriveChatFilePath but state/ subdir),
osStateFileLock (5s timeout: 200 × 25ms vs 1s for chat file append),
readModifyWriteStateFile (locked create-or-update), and runAgentDispatch
with stub runners. Wires 'agent' case into Run()."
```

---

## Task 6: `runAgentSpawn`

**Files:**
- Modify: `internal/cli/cli_agent.go` (replace runAgentSpawn stub)
- Modify: `internal/cli/cli_test.go`
- Modify: `internal/cli/export_test.go`

The `testSpawner` var enables testing without real tmux. The OS implementation calls `tmux new-window`. Binary auto-posts spawn intent and waits for engram-agent ACK (fixes #503).

- [ ] **Step 1: Write failing tests**

```go
// internal/cli/cli_test.go — add

func TestRunAgentSpawn_MissingName_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	err := cli.Run([]string{"engram", "agent", "spawn",
		"--prompt", "You are an executor.",
		"--chat-file", filepath.Join(dir, "chat.toml"),
		"--state-file", filepath.Join(dir, "state.toml"),
	}, io.Discard, io.Discard, nil)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("name"))
}

func TestRunAgentSpawn_MissingPrompt_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	err := cli.Run([]string{"engram", "agent", "spawn",
		"--name", "executor-1",
		"--chat-file", filepath.Join(dir, "chat.toml"),
		"--state-file", filepath.Join(dir, "state.toml"),
	}, io.Discard, io.Discard, nil)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("prompt"))
}

func TestRunAgentSpawn_WritesStateFileAndOutputsPaneID(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	stateFile := filepath.Join(dir, "state.toml")
	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	cli.SetTestSpawner(func(_, _ string) (string, string, error) {
		return "main:1.2", "sess123", nil
	})
	defer cli.SetTestSpawner(nil)

	var stdout bytes.Buffer
	err := cli.Run([]string{"engram", "agent", "spawn",
		"--name", "executor-1",
		"--prompt", "You are executor-1.",
		"--chat-file", chatFile,
		"--state-file", stateFile,
	}, &stdout, io.Discard, nil)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	g.Expect(strings.TrimSpace(stdout.String())).To(Equal("main:1.2|sess123"))

	data, readErr := os.ReadFile(stateFile)
	g.Expect(readErr).NotTo(HaveOccurred())
	g.Expect(string(data)).To(ContainSubstring("executor-1"))
	g.Expect(string(data)).To(ContainSubstring("main:1.2"))
}
```

- [ ] **Step 2: Add test hook export to `export_test.go`**

```go
SetTestSpawner = func(f func(name, prompt string) (paneID, sessionID string, err error)) {
    testSpawner = f
}
```

- [ ] **Step 3: Run failing tests**

```bash
targ test
```

Expected: FAIL — "not implemented"

- [ ] **Step 4: Replace runAgentSpawn stub in `cli_agent.go`**

```go
func runAgentSpawn(args []string, stdout io.Writer) error {
	fs := newFlagSet("agent spawn")
	name := fs.String("name", "", "agent name (required)")
	prompt := fs.String("prompt", "", "initial prompt for the agent (required)")
	intentMsg := fs.String("intent-text", "", "task description included in spawn intent message (optional — improves engram-agent memory lookup quality)")
	chatFile := fs.String("chat-file", "", "override chat file path (testing only)")
	stateFile := fs.String("state-file", "", "override state file path (testing only)")

	if parseErr := fs.Parse(args); errors.Is(parseErr, flag.ErrHelp) {
		return nil
	} else if parseErr != nil {
		return fmt.Errorf("agent spawn: %w", parseErr)
	}

	if *name == "" {
		return fmt.Errorf("agent spawn: --name is required")
	}

	if *prompt == "" {
		return fmt.Errorf("agent spawn: --prompt is required")
	}

	chatFilePath, pathErr := resolveChatFile(*chatFile, "agent spawn", os.UserHomeDir, os.Getwd)
	if pathErr != nil {
		return pathErr
	}

	stateFilePath, statePathErr := resolveStateFile(*stateFile, "agent spawn", os.UserHomeDir, os.Getwd)
	if statePathErr != nil {
		return statePathErr
	}

	spawnFn := testSpawner
	if spawnFn == nil {
		spawnFn = osTmuxSpawn
	}

	paneID, sessionID, spawnErr := spawnFn(*name, *prompt)
	if spawnErr != nil {
		return fmt.Errorf("agent spawn: launching pane: %w", spawnErr)
	}

	rmwErr := readModifyWriteStateFile(stateFilePath, func(sf agentpkg.StateFile) agentpkg.StateFile {
		return agentpkg.AddAgent(sf, agentpkg.AgentRecord{
			Name:      *name,
			PaneID:    paneID,
			SessionID: sessionID,
			State:     "STARTING",
			SpawnedAt: time.Now().UTC(),
		})
	})
	if rmwErr != nil {
		return fmt.Errorf("agent spawn: updating state file: %w", rmwErr)
	}

	// Post spawn intent and wait for engram-agent ACK (fixes #503).
	// --intent-text provides task context so engram-agent can surface relevant memories.
	// Without it, the generic intent has no task description and memory lookup returns nothing useful.
	// Use FileAckWaiter internally — no self-invocation.
	intentText := fmt.Sprintf(
		"Situation: About to spawn agent %q in pane %s.\nBehavior: Agent will post ready when initialized.",
		*name, paneID,
	)
	if *intentMsg != "" {
		intentText = fmt.Sprintf(
			"Situation: About to spawn agent %q in pane %s. Task: %s\nBehavior: Agent will post ready when initialized.",
			*name, paneID, *intentMsg,
		)
	}

	poster := newFilePoster(chatFilePath)
	cursor, postErr := poster.Post(chat.Message{
		From:   "system",
		To:     "engram-agent",
		Thread: "lifecycle",
		Type:   "intent",
		Text:   intentText,
	})
	if postErr != nil {
		return fmt.Errorf("agent spawn: posting spawn intent: %w", postErr)
	}

	waiter := &chat.FileAckWaiter{
		FilePath: chatFilePath,
		Watcher:  newFileWatcher(chatFilePath),
		ReadFile: os.ReadFile,
		NowFunc:  time.Now,
		MaxWait:  30 * time.Second, //nolint:mnd
	}

	ctx, cancel := signalContext()
	defer cancel()

	if _, ackErr := waiter.AckWait(ctx, "system", cursor, []string{"engram-agent"}); ackErr != nil {
		return fmt.Errorf("agent spawn: waiting for engram-agent ACK: %w", ackErr)
	}

	_, writeErr := fmt.Fprintf(stdout, "%s|%s\n", paneID, sessionID)

	return writeErr
}

// osTmuxSpawn creates a new tmux window for the agent and returns pane-id and session-id.
func osTmuxSpawn(name, prompt string) (paneID, sessionID string, err error) {
	out, cmdErr := exec.Command( //nolint:gosec
		"tmux", "new-window",
		"-d",
		"-n", name,
		"-P", "-F", "#{pane_id} #{session_id}",
		"--", "sh", "-c", prompt,
	).Output()
	if cmdErr != nil {
		return "", "", fmt.Errorf("tmux new-window: %w", cmdErr)
	}

	parts := strings.Fields(strings.TrimSpace(string(out)))
	const expectedParts = 2
	if len(parts) != expectedParts {
		return "", "", fmt.Errorf("unexpected tmux output: %q", string(out))
	}

	return parts[0], parts[1], nil
}
```

- [ ] **Step 5: Run tests to verify pass**

```bash
targ check-full
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/cli/cli_agent.go internal/cli/cli_test.go internal/cli/export_test.go
git commit -m "feat(cli): implement engram agent spawn with state file write and auto-ACK

Spawns tmux pane via tmux new-window, writes AgentRecord to state file
(R-M-W with 5s lock), posts spawn intent and waits for engram-agent ACK
(fixes #503), then outputs pane-id|session-id. Spawner is DI-injected
for testability. --intent-text optional flag includes task description in
spawn intent so engram-agent memory lookup has useful context."
```

---

## Task 7: `runAgentList` — NDJSON Output with Reconstruction Fallback

**Files:**
- Modify: `internal/cli/cli_agent.go` (replace runAgentList stub, add reconstructStateFileFromChat)
- Modify: `internal/cli/cli_test.go`

**Spec requirement (§6.3):** "The `engram agent list` command should detect a missing state file and attempt reconstruction before erroring. This is cheap to implement when Phase 3 ships; expensive to add after Phase 5 when full-file chat parsing paths may have been removed."

Reconstruction strategy: scan chat for active holds (via `chat.ScanActiveHolds`) and lifecycle `ready` messages (agents that announced presence). Agents list will be empty or partial, but holds will be recovered. Emit `slog.Warn` on reconstruction path. Do NOT write reconstructed state back to disk — reconstruction is in-memory only (write-back requires knowing which agents are still alive, which requires tmux cross-reference beyond Phase 3 scope).

- [ ] **Step 1: Write failing tests**

```go
// internal/cli/cli_test.go — add

func TestRunAgentList_EmptyStateFile_NoOutput(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.toml")
	g.Expect(os.WriteFile(stateFile, []byte(""), 0o600)).To(Succeed())

	var stdout bytes.Buffer
	err := cli.Run([]string{"engram", "agent", "list", "--state-file", stateFile}, &stdout, io.Discard, nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(strings.TrimSpace(stdout.String())).To(BeEmpty())
}

func TestRunAgentList_AbsentStateFile_AttemptsReconstruction(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	// No state file — reconstruction path. Chat file also absent (empty reconstruction).
	var stdout bytes.Buffer
	err := cli.Run([]string{"engram", "agent", "list",
		"--state-file", filepath.Join(dir, "state.toml"),
		"--chat-file", filepath.Join(dir, "chat.toml"),
	}, &stdout, io.Discard, nil)
	// No error — reconstruction attempt made, result is empty agent list.
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(strings.TrimSpace(stdout.String())).To(BeEmpty())
}

func TestRunAgentList_AbsentStateFile_ChatHasReadyMessages_ReconstructsAgents(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")

	// Write chat file with two ready messages.
	chatContent := `
[[message]]
from = "executor-1"
to = "all"
thread = "lifecycle"
type = "ready"
ts = 2026-04-06T12:00:00Z
text = """Joining chat."""

[[message]]
from = "reviewer-1"
to = "all"
thread = "lifecycle"
type = "ready"
ts = 2026-04-06T12:01:00Z
text = """Joining chat."""
`
	g.Expect(os.WriteFile(chatFile, []byte(chatContent), 0o600)).To(Succeed())

	var stdout bytes.Buffer
	err := cli.Run([]string{"engram", "agent", "list",
		"--state-file", filepath.Join(dir, "state.toml"),
		"--chat-file", chatFile,
	}, &stdout, io.Discard, nil)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	output := strings.TrimSpace(stdout.String())
	g.Expect(output).To(ContainSubstring("executor-1"))
	g.Expect(output).To(ContainSubstring("reviewer-1"))
}

func TestRunAgentList_MultipleAgents_NDJSON(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.toml")

	sf := agentpkg.StateFile{
		Agents: []agentpkg.AgentRecord{
			{Name: "executor-1", PaneID: "main:1.2", SessionID: "sess1", State: "ACTIVE"},
			{Name: "reviewer-1", PaneID: "main:1.3", SessionID: "sess2", State: "ACTIVE"},
		},
	}
	data, _ := agentpkg.MarshalStateFile(sf)
	g.Expect(os.WriteFile(stateFile, data, 0o600)).To(Succeed())

	var stdout bytes.Buffer
	err := cli.Run([]string{"engram", "agent", "list", "--state-file", stateFile}, &stdout, io.Discard, nil)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	g.Expect(lines).To(HaveLen(2))

	var rec1 map[string]any
	g.Expect(json.Unmarshal([]byte(lines[0]), &rec1)).To(Succeed())
	g.Expect(rec1["name"]).To(Equal("executor-1"))
	g.Expect(rec1["pane-id"]).To(Equal("main:1.2"))
	g.Expect(rec1["state"]).To(Equal("ACTIVE"))

	var rec2 map[string]any
	g.Expect(json.Unmarshal([]byte(lines[1]), &rec2)).To(Succeed())
	g.Expect(rec2["name"]).To(Equal("reviewer-1"))
}
```

- [ ] **Step 2: Run failing tests**

```bash
targ test
```

Expected: FAIL — "not implemented"

- [ ] **Step 3: Replace runAgentList stub in `cli_agent.go`**

```go
func runAgentList(args []string, stdout io.Writer) error {
	fs := newFlagSet("agent list")
	stateFile := fs.String("state-file", "", "override state file path (testing only)")
	chatFile := fs.String("chat-file", "", "override chat file path (used for reconstruction fallback)")

	if parseErr := fs.Parse(args); errors.Is(parseErr, flag.ErrHelp) {
		return nil
	} else if parseErr != nil {
		return fmt.Errorf("agent list: %w", parseErr)
	}

	stateFilePath, statePathErr := resolveStateFile(*stateFile, "agent list", os.UserHomeDir, os.Getwd)
	if statePathErr != nil {
		return statePathErr
	}

	data, readErr := os.ReadFile(stateFilePath)
	if errors.Is(readErr, os.ErrNotExist) {
		// Spec §6.3: detect missing state file and attempt reconstruction from chat history.
		// This is cheap in Phase 3; expensive after Phase 5 when full-file parse paths may be removed.
		chatFilePath, chatPathErr := resolveChatFile(*chatFile, "agent list", os.UserHomeDir, os.Getwd)
		if chatPathErr != nil {
			slog.Warn("agent list: state file missing, reconstruction skipped (could not resolve chat file)", "err", chatPathErr)
			return nil
		}

		return runAgentListFromChat(chatFilePath, stdout)
	}

	if readErr != nil {
		return fmt.Errorf("agent list: reading state file: %w", readErr)
	}

	sf, parseStateErr := agentpkg.ParseStateFile(data)
	if parseStateErr != nil {
		return fmt.Errorf("agent list: %w", parseStateErr)
	}

	enc := json.NewEncoder(stdout)

	for _, rec := range sf.Agents {
		if encErr := enc.Encode(rec); encErr != nil {
			return fmt.Errorf("agent list: encoding record: %w", encErr)
		}
	}

	return nil
}

// runAgentListFromChat is the reconstruction fallback for runAgentList.
// Called when the state file is missing. Emits a warning, reconstructs an
// in-memory StateFile from chat history, and lists agents from that.
func runAgentListFromChat(chatFilePath string, stdout io.Writer) error {
	sf, reconErr := reconstructStateFileFromChat(chatFilePath, os.ReadFile)
	if reconErr != nil {
		slog.Warn("agent list: state file missing, reconstruction failed", "err", reconErr)
		return nil
	}

	slog.Warn("agent list: state file missing, using reconstructed state from chat history (agent list may be incomplete)")

	enc := json.NewEncoder(stdout)

	for _, rec := range sf.Agents {
		if encErr := enc.Encode(rec); encErr != nil {
			return fmt.Errorf("agent list: encoding reconstructed record: %w", encErr)
		}
	}

	return nil
}

// reconstructStateFileFromChat builds a best-effort StateFile from the chat history.
// Reads all lifecycle messages to extract agent names from `ready` messages.
// Reads all hold-acquire/release pairs via chat.ScanActiveHolds.
// Result is in-memory only — NOT written to disk. Agent list may be partial
// (agents that posted ready but whose done/shutdown was missed are included;
// agents that never posted ready are absent).
func reconstructStateFileFromChat(chatFilePath string, readFile func(string) ([]byte, error)) (agentpkg.StateFile, error) {
	messages, loadErr := loadChatMessages(chatFilePath, readFile)
	if loadErr != nil {
		return agentpkg.StateFile{}, fmt.Errorf("loading chat for reconstruction: %w", loadErr)
	}

	sf := agentpkg.StateFile{}

	// Reconstruct holds from chat log — ScanActiveHolds already handles acquire/release pairs.
	activeHolds := chat.ScanActiveHolds(messages)
	for _, hold := range activeHolds {
		sf = agentpkg.AddHold(sf, agentpkg.HoldEntry{
			HoldID:     hold.HoldID,
			Holder:     hold.Holder,
			Target:     hold.Target,
			Condition:  hold.Condition,
			Tag:        hold.Tag,
			AcquiredTS: hold.AcquiredTS,
		})
	}

	// Reconstruct agents from ready messages. Track which agents posted done/shutdown
	// so we can exclude them (they exited cleanly).
	doneAgents := make(map[string]bool)
	for _, msg := range messages {
		if msg.Type == "done" || msg.Type == "shutdown" {
			doneAgents[msg.From] = true
		}
	}

	seen := make(map[string]bool)
	for _, msg := range messages {
		if msg.Type != "ready" || seen[msg.From] || doneAgents[msg.From] {
			continue
		}

		seen[msg.From] = true
		sf = agentpkg.AddAgent(sf, agentpkg.AgentRecord{
			Name:  msg.From,
			State: "UNKNOWN", // reconstruction cannot verify live state without tmux
		})
	}

	return sf, nil
}
```

`json.Encoder.Encode` appends a newline after each record — correct NDJSON behavior.

Note: `slog` is already imported in `watcher.go` (same module). Add `"log/slog"` to `cli_agent.go` imports.

- [ ] **Step 4: Run tests to verify pass**

```bash
targ check-full
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/cli/cli_agent.go internal/cli/cli_test.go
git commit -m "feat(cli): implement engram agent list with NDJSON output and reconstruction fallback

Reads state file and outputs one JSON object per agent record via
json.Encoder (appends newline = NDJSON format). Lesson from #516: never
ship structured binary output as non-JSON.

Spec §6.3: on missing state file, attempts in-memory reconstruction from
chat history (active holds via ScanActiveHolds, agent names from ready
messages). Warns via slog.Warn. Does not write reconstructed state to disk.
Implements spec's 'cheap in Phase 3, expensive after Phase 5' requirement."
```

---

## Task 8: `runAgentKill`

**Files:**
- Modify: `internal/cli/cli_agent.go` (replace runAgentKill stub)
- Modify: `internal/cli/cli_test.go`
- Modify: `internal/cli/export_test.go`

Reads state file to find agent's pane-id, evaluates `done:` hold conditions using `chat.ScanActiveHolds` + `chat.EvaluateCondition` (internal domain calls — no self-invocation), removes agent from state file, kills the tmux pane.

**Hold guard invariant (planner-11 Finding 3):** Iterate only holds whose `Target` matches `--name`. If any such hold's condition is NOT met, `runAgentKill` MUST return an error — it must NOT proceed to kill. Automatically-evaluatable conditions (`done:<agent>`, `first-intent:<agent>`) are released normally. Only `lead-release` conditions require explicit lead action; killing with an unsatisfied `lead-release` hold leaves a dangling unreleased hold in the chat file. Without the target filter, holds on other agents in the session would cause spurious kill failures.

**Dead pane handling (planner-11 Finding 5):** `osTmuxKillPane` must filter `no such pane` from tmux errors and return nil. When an agent's Claude process exits cleanly, its tmux pane auto-closes. Calling `engram agent kill` after this returns a non-zero tmux error that breaks the happy-path graceful shutdown flow.

- [ ] **Step 1: Write failing tests**

```go
// internal/cli/cli_test.go — add

func TestRunAgentKill_MissingName_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	g.Expect(os.WriteFile(filepath.Join(dir, "chat.toml"), []byte(""), 0o600)).To(Succeed())
	err := cli.Run([]string{"engram", "agent", "kill",
		"--state-file", filepath.Join(dir, "state.toml"),
		"--chat-file", filepath.Join(dir, "chat.toml"),
	}, io.Discard, io.Discard, nil)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("name"))
}

func TestRunAgentKill_RemovesAgentFromStateFile(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.toml")
	chatFile := filepath.Join(dir, "chat.toml")

	sf := agentpkg.StateFile{
		Agents: []agentpkg.AgentRecord{
			{Name: "executor-1", PaneID: "main:1.2", SessionID: "sess1", State: "ACTIVE"},
			{Name: "reviewer-1", PaneID: "main:1.3", SessionID: "sess2", State: "ACTIVE"},
		},
	}
	data, _ := agentpkg.MarshalStateFile(sf)
	g.Expect(os.WriteFile(stateFile, data, 0o600)).To(Succeed())
	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	cli.SetTestPaneKiller(func(_ string) error { return nil })
	defer cli.SetTestPaneKiller(nil)

	err := cli.Run([]string{"engram", "agent", "kill",
		"--name", "executor-1",
		"--state-file", stateFile,
		"--chat-file", chatFile,
	}, io.Discard, io.Discard, nil)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	remaining, _ := os.ReadFile(stateFile)
	g.Expect(string(remaining)).NotTo(ContainSubstring("executor-1"))
	g.Expect(string(remaining)).To(ContainSubstring("reviewer-1"))
}

func TestRunAgentKill_ActiveUnmetHold_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.toml")
	chatFile := filepath.Join(dir, "chat.toml")

	sf := agentpkg.StateFile{
		Agents: []agentpkg.AgentRecord{
			{Name: "executor-1", PaneID: "main:1.2", SessionID: "sess1", State: "STARTING"},
		},
	}
	data, _ := agentpkg.MarshalStateFile(sf)
	g.Expect(os.WriteFile(stateFile, data, 0o600)).To(Succeed())

	// Chat file contains an unsatisfied lead-release hold on executor-1.
	chatContent := `
[[message]]
from = "lead"
to = "executor-1"
thread = "hold"
type = "hold-acquire"
ts = 2026-04-06T12:00:00Z
text = """{"hold-id":"test-hold-1","holder":"lead","target":"executor-1","condition":"lead-release:phase3","tag":"phase3","acquired-ts":"2026-04-06T12:00:00Z"}"""
`
	g.Expect(os.WriteFile(chatFile, []byte(chatContent), 0o600)).To(Succeed())

	err := cli.Run([]string{"engram", "agent", "kill",
		"--name", "executor-1",
		"--state-file", stateFile,
		"--chat-file", chatFile,
	}, io.Discard, io.Discard, nil)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("active hold"))
}

func TestRunAgentKill_PaneAlreadyDead_NoError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	stateFile := filepath.Join(dir, "state.toml")
	chatFile := filepath.Join(dir, "chat.toml")

	sf := agentpkg.StateFile{
		Agents: []agentpkg.AgentRecord{
			{Name: "executor-1", PaneID: "main:1.2", SessionID: "sess1", State: "STARTING"},
		},
	}
	data, _ := agentpkg.MarshalStateFile(sf)
	g.Expect(os.WriteFile(stateFile, data, 0o600)).To(Succeed())
	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	// Simulate pane already dead: tmux returns "no such pane" error.
	cli.SetTestPaneKiller(func(_ string) error {
		return fmt.Errorf("exit status 1: no such pane: main:1.2")
	})
	defer cli.SetTestPaneKiller(nil)

	err := cli.Run([]string{"engram", "agent", "kill",
		"--name", "executor-1",
		"--state-file", stateFile,
		"--chat-file", chatFile,
	}, io.Discard, io.Discard, nil)
	// Dead pane is not an error — agent is already gone.
	g.Expect(err).NotTo(HaveOccurred())
}
```

- [ ] **Step 2: Add test hook export to `export_test.go`**

```go
SetTestPaneKiller = func(f func(paneID string) error) {
    testPaneKiller = f
}
```

- [ ] **Step 3: Run failing tests**

```bash
targ test
```

Expected: FAIL — "not implemented"

- [ ] **Step 4: Replace runAgentKill stub in `cli_agent.go`**

```go
func runAgentKill(args []string, stdout io.Writer) error {
	fs := newFlagSet("agent kill")
	name := fs.String("name", "", "agent name to kill (required)")
	chatFile := fs.String("chat-file", "", "override chat file path (testing only)")
	stateFile := fs.String("state-file", "", "override state file path (testing only)")

	if parseErr := fs.Parse(args); errors.Is(parseErr, flag.ErrHelp) {
		return nil
	} else if parseErr != nil {
		return fmt.Errorf("agent kill: %w", parseErr)
	}

	if *name == "" {
		return fmt.Errorf("agent kill: --name is required")
	}

	chatFilePath, pathErr := resolveChatFile(*chatFile, "agent kill", os.UserHomeDir, os.Getwd)
	if pathErr != nil {
		return pathErr
	}

	stateFilePath, statePathErr := resolveStateFile(*stateFile, "agent kill", os.UserHomeDir, os.Getwd)
	if statePathErr != nil {
		return statePathErr
	}

	// Evaluate hold conditions using domain functions directly (no self-invocation).
	messages, loadErr := loadChatMessages(chatFilePath, os.ReadFile)
	if loadErr != nil {
		return fmt.Errorf("agent kill: %w", loadErr)
	}

	activeHolds := chat.ScanActiveHolds(messages)
	poster := newFilePoster(chatFilePath)

	for _, hold := range activeHolds {
		// Only evaluate holds targeting this specific agent — not other agents' holds.
		if hold.Target != *name {
			continue
		}

		met, _ := chat.EvaluateCondition(hold, messages)
		if !met {
			// An unmet hold means the condition (e.g. lead-release) has not been satisfied.
			// Kill must fail — proceeding would leave a dangling unreleased hold in the chat file.
			return fmt.Errorf("agent kill: active hold %s condition not satisfied (condition: %s); release it first",
				hold.HoldID, hold.Condition)
		}

		releaseText, marshalErr := marshalReleasePayload(hold.HoldID)
		if marshalErr != nil {
			return fmt.Errorf("agent kill: marshaling release: %w", marshalErr)
		}

		if _, postErr := poster.Post(chat.Message{
			From:   "system",
			To:     "all",
			Thread: "hold",
			Type:   "hold-release",
			Text:   string(releaseText),
		}); postErr != nil {
			return fmt.Errorf("agent kill: posting release for %s: %w", hold.HoldID, postErr)
		}
	}

	// Find pane-id and remove agent from state file.
	var paneID string

	if rmwErr := readModifyWriteStateFile(stateFilePath, func(sf agentpkg.StateFile) agentpkg.StateFile {
		for _, rec := range sf.Agents {
			if rec.Name == *name {
				paneID = rec.PaneID
				break
			}
		}

		return agentpkg.RemoveAgent(sf, *name)
	}); rmwErr != nil {
		return fmt.Errorf("agent kill: updating state file: %w", rmwErr)
	}

	// Kill tmux pane.
	killFn := testPaneKiller
	if killFn == nil {
		killFn = osTmuxKillPane
	}

	if paneID != "" {
		if killErr := killFn(paneID); killErr != nil && !strings.Contains(killErr.Error(), "no such pane") {
			return fmt.Errorf("agent kill: killing pane %s: %w", paneID, killErr)
		}
	}

	_, writeErr := fmt.Fprintf(stdout, "killed %s\n", *name)

	return writeErr
}

// osTmuxKillPane kills the tmux pane with the given pane-id.
// Returns nil if the pane is already gone ("no such pane") — graceful shutdown
// may have auto-closed the pane before kill is called.
func osTmuxKillPane(paneID string) error {
	out, err := exec.Command("tmux", "kill-pane", "-t", paneID).CombinedOutput() //nolint:gosec
	if err != nil && !strings.Contains(string(out), "no such pane") {
		return fmt.Errorf("tmux kill-pane: %w", err)
	}

	return nil
}
```

- [ ] **Step 5: Run tests to verify pass**

```bash
targ check-full
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/cli/cli_agent.go internal/cli/cli_test.go internal/cli/export_test.go
git commit -m "feat(cli): implement engram agent kill with hold guard and dead-pane handling

Evaluates done: hold conditions via chat.ScanActiveHolds +
chat.EvaluateCondition (no self-invocation), releases matching holds,
removes agent from state file (R-M-W with 5s lock), and kills the tmux
pane. Fixes #500 (engram-down can use 'engram agent kill' for any agent).

Unmet-hold guard: returns error if any active hold condition is not met
(e.g. lead-release) — prevents dangling unreleased holds in chat file.

Dead-pane handling: osTmuxKillPane filters 'no such pane' from tmux errors.
After graceful shutdown, the agent's tmux pane may auto-close; kill must
succeed in that case."
```

---

## Task 9: `runAgentWaitReady` — watchDeadline Pattern

**Files:**
- Modify: `internal/cli/cli_agent.go` (replace runAgentWaitReady stub)
- Modify: `internal/cli/cli_test.go`

**CRITICAL INVARIANT:** `--max-wait` MUST flow into `watcher.Watch()` via `context.WithTimeout`. Without this the inner fsnotify loop blocks forever — same root cause as pre-b22dc0c #519. This is a first-class design contract, not a style preference.

- [ ] **Step 1: Write failing tests**

```go
// internal/cli/cli_test.go — add

func TestRunAgentWaitReady_MissingName_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	err := cli.Run([]string{"engram", "agent", "wait-ready",
		"--cursor", "0",
		"--chat-file", chatFile,
	}, io.Discard, io.Discard, nil)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("name"))
}

func TestRunAgentWaitReady_SeesReadyMessage_OutputsJSON(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	// Append a ready message after a short delay to simulate agent startup.
	go func() {
		time.Sleep(50 * time.Millisecond)
		readyTOML := "\n[[message]]\nfrom = \"executor-1\"\nto = \"all\"\nthread = \"lifecycle\"\ntype = \"ready\"\nts = 2026-04-06T12:00:00Z\ntext = \"\"\"Joining chat.\"\"\"\n"
		f, _ := os.OpenFile(chatFile, os.O_APPEND|os.O_WRONLY, 0o600)
		if f != nil {
			_, _ = f.WriteString(readyTOML)
			_ = f.Close()
		}
	}()

	var stdout bytes.Buffer
	err := cli.Run([]string{"engram", "agent", "wait-ready",
		"--name", "executor-1",
		"--cursor", "0",
		"--max-wait", "5",
		"--chat-file", chatFile,
	}, &stdout, io.Discard, nil)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	var result map[string]any
	g.Expect(json.Unmarshal([]byte(strings.TrimSpace(stdout.String())), &result)).To(Succeed())
	g.Expect(result["type"]).To(Equal("ready"))
	g.Expect(result["from"]).To(Equal("executor-1"))
	g.Expect(result["cursor"]).NotTo(BeZero())
}

func TestRunAgentWaitReady_MaxWaitExpires_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	g.Expect(os.WriteFile(chatFile, []byte(""), 0o600)).To(Succeed())

	start := time.Now()
	err := cli.Run([]string{"engram", "agent", "wait-ready",
		"--name", "nonexistent-agent",
		"--cursor", "0",
		"--max-wait", "1", // 1 second
		"--chat-file", chatFile,
	}, io.Discard, io.Discard, nil)
	elapsed := time.Since(start)

	g.Expect(err).To(HaveOccurred()) // deadline exceeded
	g.Expect(elapsed).To(BeNumerically("<", 5*time.Second)) // must not block indefinitely
}
```

- [ ] **Step 2: Run failing tests**

```bash
targ test
```

Expected: FAIL — "not implemented"

- [ ] **Step 3: Replace runAgentWaitReady stub in `cli_agent.go`**

```go
func runAgentWaitReady(args []string, stdout io.Writer) error {
	fs := newFlagSet("agent wait-ready")
	name := fs.String("name", "", "agent name to wait for (required)")
	cursor := fs.Int("cursor", 0, "line position to start watching from")
	maxWaitS := fs.Int("max-wait", 30, "seconds to wait before giving up (default 30)") //nolint:mnd
	chatFile := fs.String("chat-file", "", "override chat file path (testing only)")

	if parseErr := fs.Parse(args); errors.Is(parseErr, flag.ErrHelp) {
		return nil
	} else if parseErr != nil {
		return fmt.Errorf("agent wait-ready: %w", parseErr)
	}

	if *name == "" {
		return fmt.Errorf("agent wait-ready: --name is required")
	}

	chatFilePath, pathErr := resolveChatFile(*chatFile, "agent wait-ready", os.UserHomeDir, os.Getwd)
	if pathErr != nil {
		return pathErr
	}

	// WATCHDEADLINE PATTERN: --max-wait MUST flow through context.WithTimeout into
	// the inner watcher.Watch() blocking call (fsnotify loop). Checking the deadline
	// only in the outer loop is insufficient — fsnotify blocks indefinitely without
	// a context deadline. Same class as pre-b22dc0c #519 bug.
	ctx, cancel := signalContext()
	defer cancel()

	if *maxWaitS > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(*maxWaitS)*time.Second)
		defer cancel()
	}

	watcher := newFileWatcher(chatFilePath)

	msg, newCursor, watchErr := watcher.Watch(ctx, *name, *cursor, []string{"ready"})
	if watchErr != nil {
		return fmt.Errorf("agent wait-ready: %w", watchErr)
	}

	result := watchResult{
		From:   msg.From,
		To:     msg.To,
		Thread: msg.Thread,
		Type:   msg.Type,
		TS:     msg.TS,
		Text:   msg.Text,
		Cursor: newCursor,
	}

	return marshalAndWriteWatchResult(stdout, result)
}
```

`watchResult` and `marshalAndWriteWatchResult` are defined in `cli_chat.go` (same package, no import needed).

- [ ] **Step 4: Run tests to verify pass**

```bash
targ check-full
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/cli/cli_agent.go internal/cli/cli_test.go
git commit -m "feat(cli): implement engram agent wait-ready with watchDeadline pattern

Watches for type=ready messages addressed to --name. Applies --max-wait via
context.WithTimeout which flows into watcher.Watch() (fsnotify level) —
the watchDeadline pattern. Without this, fsnotify blocks forever regardless
of --max-wait value. Same root cause as pre-b22dc0c #519. Outputs JSON
matching chat watch format."
```

---

## Task 10: Skill Updates

**Files:**
- Modify: `skills/engram-tmux-lead/SKILL.md`
- Modify: `skills/engram-agent/SKILL.md`
- Modify: `skills/engram-down/SKILL.md`

**SUB-SKILL REQUIRED:** Use `superpowers:writing-skills` for EVERY skill edit in this task. No exceptions — this enforces TDD (baseline behavior test RED, update skill GREEN, verify behavioral change, run pressure tests).

### Part A: `engram-tmux-lead` — Delete SPAWN-PANE/KILL-PANE, Add Agent Lifecycle

- [ ] **Step 1: Invoke writing-skills skill**

Before any edits: invoke `superpowers:writing-skills`.

- [ ] **Step 2: Identify deletion targets in `skills/engram-tmux-lead/SKILL.md`**

Locate and delete these sections (verify line ranges before cutting):
- Session initialization variables: `RIGHT_PANE_COUNT=0`, `MIDDLE_COL_LAST_PANE=""`, `RIGHT_COL_LAST_PANE=""` setup block (~5 lines)
- SPAWN-PANE block: from `#### SPAWN-PANE` header through the closing `RIGHT_PANE_COUNT=$((RIGHT_PANE_COUNT + 1))` (~30 lines including HARD GATE comment and if/elif/else tmux split-window logic)
- Two-column layout ASCII diagrams: the SPAWN-PANE column-filling algorithm diagrams immediately following the SPAWN-PANE block (~25 lines). These explain SPAWN-PANE layout logic; with SPAWN-PANE deleted they become dead content. Delete entirely — the binary manages layout automatically.
- KILL-PANE block: from `#### KILL-PANE` header through the pane registry update instructions (~20 lines)
- Concurrency tracking: all inline RIGHT_PANE_COUNT increment/decrement lines elsewhere in the skill
- Pane registry: MIDDLE_COL_LAST_PANE and RIGHT_COL_LAST_PANE tracking instructions
- Shutdown kill-pane sequence: the block that calls KILL-PANE or tmux kill-pane on shutdown
- Red flag: "Calling `tmux split-window` directly instead of using SPAWN-PANE from Section 1.3"
- Verify no legacy polling in monitoring section (#523 — confirm closed): the skill already uses Background Monitor Pattern and `engram chat watch`; this is a no-op confirmation during the RED step.

Total deletions: approximately 85 lines.

- [ ] **Step 3: Add Agent Lifecycle section (~15 lines)**

After the session initialization section, add:

```markdown
## Agent Lifecycle

Use binary commands for all pane management. Never call tmux directly.

**Spawn agent** (capture cursor BEFORE spawn — required for wait-ready to see the ready message):
```bash
PRE_SPAWN_CURSOR=$(engram chat cursor)
RESULT=$(engram agent spawn --name <name> --prompt "<prompt>")
PANE_ID=$(echo "$RESULT" | cut -d'|' -f1)
SESSION_ID=$(echo "$RESULT" | cut -d'|' -f2)
```

**Wait for agent ready** (pass PRE_SPAWN_CURSOR, not a post-spawn cursor — the ready message arrives after spawn and must be visible):
```bash
WAIT_RESULT=$(engram agent wait-ready --name <name> --cursor $PRE_SPAWN_CURSOR --max-wait 30)
NEW_CURSOR=$(echo "$WAIT_RESULT" | jq -r '.cursor')
```

**Kill agent** (DONE state — send shutdown via chat FIRST so agent's protocol state is aligned before pane death):
```bash
# 1. Send shutdown via chat (lead's responsibility — binary does NOT post this)
engram chat post --from lead --to <name> --thread lifecycle --type shutdown --text "Session complete."
# 2. Kill agent (releases met holds + removes from state file + kills tmux pane)
engram agent kill --name <name>
```

**List agents (NDJSON — parse with jq):**
```bash
engram agent list | jq -r '.name'
```
```

Also update the red flags section: **replace** the existing "Calling `tmux split-window` directly instead of using SPAWN-PANE from Section 1.3" red flag with "Calling `tmux` commands directly — use `engram agent spawn/kill` instead." (The old red flag references SPAWN-PANE which no longer exists; leaving it causes confusion. Delete-and-replace, not addition.)

- [ ] **Step 4: Run writing-skills TDD cycle and commit**

Follow the writing-skills skill exactly. Commit after pressure tests pass:

```bash
git add skills/engram-tmux-lead/SKILL.md
git commit -m "feat(skills): replace SPAWN-PANE/KILL-PANE bash with engram agent commands

Phase 3 skill update. Deletes ~85 lines: SPAWN-PANE definition, KILL-PANE
definition, two-column layout ASCII diagrams, concurrency tracking bash,
pane registry instructions, and shutdown kill-pane sequence. Adds ~15-line
Agent Lifecycle section using engram agent spawn/kill/list/wait-ready binary
commands.

Closes-with-Phase-3: #505/#506 (bugs in SPAWN-PANE code path now deleted)."
```

### Part B: `engram-agent` — Verify Background Monitor Pattern (pre-flight complete)

> **NOTE:** Issue #520 (commit 3246f71) already shipped the Background Monitor Pattern in `skills/engram-agent/SKILL.md`. Task 10B is **verification only** — confirm the pattern is in place, mark #520 done in pre-flight, and skip Steps 6–8 below.

- [ ] **Step 5: Verify Background Monitor Pattern is in place**

Confirm `skills/engram-agent/SKILL.md` already uses `engram chat watch` (not `fswatch -1`) in its main loop. Check approximately lines 105–140. Expected content:

```
RESULT=$(engram chat watch --agent engram-agent --cursor CURSOR)
```

If this is present: Task 10B is complete. Mark #520 as confirmed closed in the pre-flight table and skip Steps 6–8.

If the old pattern is still present (unexpected): proceed with Steps 6–8.

- [ ] **Step 6: (Skip if Step 5 passed) Invoke writing-skills skill**

Fresh invocation before editing engram-agent SKILL.md.

- [ ] **Step 7: (Skip if Step 5 passed) Replace with Background Monitor Pattern**

Replace the `fswatch -1` Steps 1-4 and loop diagram with the Background Monitor Pattern (see use-engram-chat-as skill). Also update startup steps: replace "Enter the fswatch loop" with "Spawn background monitor Agent (cursor = cursor from step above)".

- [ ] **Step 8: (Skip if Step 5 passed) Run writing-skills TDD cycle and commit**

```bash
git add skills/engram-agent/SKILL.md
git commit -m "feat(skills): replace engram-agent fswatch -1 loop with Background Monitor Pattern

Phase 3 skill scope (per planner-7). The fswatch -1 approach was the
pre-Phase-1 pattern: raw bash background task with visible tool-call noise,
no cursor-based filtering, incompatible with context compaction recovery.

Replaced with Background Monitor Pattern: background Agent calling
'engram chat watch --agent engram-agent --cursor CURSOR'. Kernel-driven
(fsnotify), cursor-tracked, consistent with use-engram-chat-as skill."
```


### Part C: `engram-down` — Replace tmux Kill Sequence, Fix Task ID Name

**Context:** Issue #500 (engram-down only works from lead) is mapped to "Task 8 + Task 10A skill update". However Task 10A only touches `engram-tmux-lead`. The `engram-down/SKILL.md` has two separate bugs that Phase 3 must fix for #500 to be fully addressed in the skill layer.

**Bug 1 (Step 2):** Kill sequence uses raw `tmux list-panes | grep claude | xargs kill-pane`. With `engram agent list/kill` available, this should use the binary instead. The tmux-grep approach misses panes where a wrapper is the foreground process.

**Bug 2 (Step 4):** References `CHAT_FSWATCH_TASK_ID` for draining the background watcher. The current lead skill uses `CHAT_MONITOR_TASK_ID` (background Agent task ID, not an fswatch task). This name is stale since Phase 1/2 migration.

- [ ] **Step 9: Invoke writing-skills skill again**

Fresh invocation before editing engram-down SKILL.md.

- [ ] **Step 10: Identify the two fix targets in `skills/engram-down/SKILL.md`**

1. Locate the kill sequence (Step 2 of the teardown procedure): find the `tmux list-panes | grep claude` block.
2. Locate the drain step (Step 4): find `CHAT_FSWATCH_TASK_ID` reference.

- [ ] **Step 11: Apply fixes**

**Fix 1 — Kill sequence (Step 2):** Replace the raw tmux kill block with:

```bash
# Kill all running agents via binary
engram agent list | jq -r '.name' | while read -r agent_name; do
  engram agent kill --name "$agent_name"
done
```

**Fix 2 — Task ID name (Step 4):** Replace `CHAT_FSWATCH_TASK_ID` with `CHAT_MONITOR_TASK_ID` throughout. Update any surrounding instruction text to match (the background Agent task, not an fswatch pid).

- [ ] **Step 12: Run writing-skills TDD cycle and commit**

```bash
git add skills/engram-down/SKILL.md
git commit -m "feat(skills): update engram-down to use engram agent list/kill for teardown

Phase 3 skill update (Task 10C). Replaces tmux list-panes grep with
engram agent list | jq + engram agent kill loop. Fixes reliability
regression where wrapper processes weren't matched by grep. Also renames
CHAT_FSWATCH_TASK_ID to CHAT_MONITOR_TASK_ID to match current lead skill.

Contributes-to: #500 (engram-down usable from any pane, not just lead)."
```

---

## E2E Check

- [ ] **Step 1: Full test suite**

```bash
targ check-full
```

Expected: PASS

- [ ] **Step 2: Build binary**

```bash
targ build
```

Expected: `engram` binary created

- [ ] **Step 3: Verify help text**

```bash
engram agent spawn --help
engram agent kill --help
engram agent list --help
engram agent wait-ready --help
```

Expected: help text, exit 0

- [ ] **Step 4: Smoke test list**

```bash
TMP=$(mktemp -d)
engram agent list --state-file "$TMP/state.toml"
echo "Exit: $?"
```

Expected: no output, exit 0

- [ ] **Step 5: Verify jq parsing (Criterion 2)**

```bash
TMP=$(mktemp -d)
cat > "$TMP/state.toml" << 'EOF'
[[agent]]
name = "test-agent"
pane-id = "main:1.2"
session-id = "sess123"
state = "ACTIVE"
spawned-at = 2026-04-06T12:00:00Z
argument-with = ""
argument-count = 0
argument-thread = ""
EOF

engram agent list --state-file "$TMP/state.toml" | jq -r '.name'
```

Expected: `test-agent`

- [ ] **Step 6: Verify watchDeadline fires (Criterion 5)**

```bash
TMP=$(mktemp -d) && echo "" > "$TMP/chat.toml"
time engram agent wait-ready \
  --name nonexistent-agent \
  --cursor 0 \
  --max-wait 3 \
  --chat-file "$TMP/chat.toml"
```

Expected: exits in ~3s with deadline-exceeded error, NOT a 30s hang

---

## Issues Fixed by This Phase

| Issue | Fix |
|-------|-----|
| #505, #506 | SPAWN-PANE/KILL-PANE code path deleted in Task 10A |
| #500 | `engram agent kill` handles any agent (Task 8 + Task 10A skill update + Task 10C engram-down kill-sequence + CHAT_MONITOR_TASK_ID fix) |
| #503 | Binary auto-posts spawn intent + waits for ACK (Task 6) |
| #520 (fswatch-1 main loop) | Already shipped (commit 3246f71); Task 10B is verification only |
| #523 (lead spawns monitor agents with improvised polling) | Already fixed before Phase 3 — Background Monitor Pattern and `engram chat watch` already in use in `engram-tmux-lead/SKILL.md`. Task 10A Step 2 includes a no-op verification during the RED step. |
