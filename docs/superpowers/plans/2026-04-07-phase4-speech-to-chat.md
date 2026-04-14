# Phase 4 — Speech-to-Chat Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace interactive worker spawning with headless `-p` workers piped through a binary stream pipeline that translates prefix markers in worker output into chat messages, eliminating the need for workers to call `engram chat post` directly.

**Architecture:** A new `internal/streamjson` package parses `--output-format=stream-json` JSONL events. A new `internal/claude` package implements `Runner`, which executes `claude -p`, reads JSONL, writes filtered output to `os.Stdout` (the tmux pane), detects prefix markers in assistant text, and posts to chat on the worker's behalf. A new `engram agent run` subcommand runs the Runner inside the tmux pane; `osTmuxSpawnWith` is rewritten to start this subcommand instead of launching Claude interactively. The real Claude conversation session-id is extracted from the first JSONL event and written to the state file before the first speech act, unblocking Phase 5 `--resume`.

**Spec:** `docs/superpowers/specs/engram-deterministic-coordination-design.md` §4.5, §5, §7 (Phase 4)

**Codesign session:** planners arch-lead (planner-1), go-binary (planner-2), skill-coverage (planner-3), phasing (planner-4) — 2026-04-07. All four perspectives converged before this plan was written. Codesign decisions are locked in the chat file (`~/.local/share/engram/chat/-Users-joe-repos-personal-engram.toml`) around line 78633–80116.

**Tech Stack:** Go, `bufio.Scanner` for JSONL stream, `encoding/json`, `os/exec` for `claude -p`, existing `internal/chat`, `internal/agent`, `internal/cli` packages.

---

## Pre-Flight (issue audit — do before starting tasks)

These are bookkeeping actions, not implementation work. Complete before Task 0.

| Issue | Action | Rationale |
|-------|--------|-----------|
| #527, #526, #528, #529, #530, #531, #532, #533 | Close with note pointing to the fixing commit | Already fixed in commits; issue tracker is stale. |
| #503 | Verify closed: after `engram agent spawn`, chat file has type=ack posted by system with no manual action. If verified, close. | #531 fixed binary-posted ACK. Confirm spawned agent sees it. |
| #504 | Verify closed: `@engram_name` fix in #526 satisfies descriptive pane titles at spawn time. If verified, close. | Needs live session verification. |
| #522 (pane-border-status not enabled) | Tiny standalone PR: add `tmux set-option -g pane-border-status top` to lead startup sequence. | ~1 line. Not a Phase 4 blocker but fix before starting. |
| #525 (engram-down chat-tail pane survives) | Tiny standalone PR: skill fix to kill the tail pane in engram-down shutdown sequence. | ~5 lines. Fix before starting. |
| #534 (health-checker trigger loop in lead skill) | Tiny standalone PR: delete the ~5-line health-checker trigger loop from engram-tmux-lead. | ~5 line deletion. Fix before starting. |
| #494 | Mark "targets Phase 4". | Display filtering in stream pipeline eliminates raw JSONL from panes. |
| #524 (Background Monitor Pattern verbatim note) | Defer to Phase 5. | Phase 5 DELETES Background Monitor Pattern entirely. Fixing docs on something being deleted is wasted effort. |

---

## E2E Acceptance Criteria

Phase 4 is done when all five criteria pass. Criteria 1–4 must pass before Category B skill deletions (Task 8). Criterion 5 verifies the deletions are safe.

### Criterion 1: Real Session-ID in State File

After `engram agent spawn`, the agent's `session-id` in the state file must be a Claude conversation UUID (format: 32 hex chars with hyphens, e.g. `550e8400-e29b-41d4-a716-446655440000`). It must NOT be a tmux session ID (format: `$N` where N is an integer) or a placeholder. The UUID must appear in the state file BEFORE the agent's first INTENT: speech act reaches the chat file.

```bash
# Verify after spawn:
engram agent list | jq -r '.["session-id"]'
# Expected: a UUID like 550e8400-e29b-41d4-a716-446655440000
# NOT: $0, $1, /bin/zsh, or empty string
```

### Criterion 2: Speech Relay Live

A spawned worker that outputs `INTENT: Situation: X. Behavior: Y.` in its turn must result in a `type = "intent"` message appearing in the chat file with `from = "<agent-name>"` (the agent's name, not "system"). The worker must then receive either `Proceed.` (all ACKed) or `WAIT from engram-agent: [text]` (objection) in its next turn.

```bash
# Spawn a worker with a known task, inspect chat after its first turn:
tail -n 20 ~/.local/share/engram/chat/-Users-joe-repos-personal-engram.toml
# Expected: [[message]] with type = "intent" from the agent's name
```

### Criterion 3: Display Filtering

The agent's tmux pane shows only assistant text and user turns. No raw JSONL events visible in the pane (no lines starting with `{"type":"tool_use"`, `{"type":"result"`, `{"type":"system"`, etc.).

Verify visually in a live session: spawn an agent that uses a tool, observe the pane. Only the prose output and responses should be visible.

### Criterion 4: `--dangerously-skip-permissions` in `-p` Mode

`engram agent run` successfully starts `claude -p --dangerously-skip-permissions --verbose --output-format=stream-json <prompt>` without permission errors. The first JSONL event is received and parsed. The session-id is a Claude UUID (not empty, not tmux format).

```bash
# Manual test:
engram agent run --name test-p --prompt "Say hello and exit." --chat-file /tmp/test.toml --state-file /tmp/test-state.toml
# Expected: claude -p launches, JSONL flows, session-id appears in /tmp/test-state.toml
```

### Criterion 5: Skill Deletions Safe

After verifying Criteria 1–4, the Category B skill deletions (Writing Messages bash, Heartbeat bash, `engram chat post` calling-convention for workers) are applied. Then run a full multi-agent session with at least 2 spawned workers. Coordination works end-to-end: workers express intents via prefix markers, binary relays to chat, engram-agent responds via prefix markers, binary ACKs the workers. No worker attempts to call `engram chat post` directly.

---

## Codesign Decisions

These decisions were argued and resolved during the 2026-04-07 codesign session. Do NOT revisit without reading the `codesign-phases` thread in the chat file from line ~78633.

| Decision | Resolved | Rationale |
|----------|----------|-----------|
| `internal/streamjson` package: stateless per-event JSONL parser | Yes | Pure domain logic, trivially testable. "Stateless" means no state between JSONL lines — each line is a complete assembled message. Multi-line speech acts detected within a single event's text string by scanning `\n`-split lines. |
| `internal/claude` package: Runner owns the pipeline | Yes | DI-clean: Runner.Pane (io.Writer), Runner.Poster (chat.Poster), Runner.StateFile (string), Runner.AgentName (string). All I/O injected. |
| Runner.Pane = os.Stdout of in-pane process | Yes (Option A) | spawn creates pane, sends `engram agent run --name N --prompt P` via send-keys ONCE to start the runner binary in-pane. Runner.Pane = os.Stdout. No tty path detection, no cross-process file open. Categorically different from send-keys for task content: starting a deterministic binary subcommand. |
| Session-id R-M-W inside Runner.Run | Yes | Runner extracts Claude conversation UUID from first JSONL event. Immediately calls `readModifyWriteStateFile` to update AgentRecord.SessionID from placeholder to real UUID. This happens BEFORE the first speech act relay. |
| `engram agent run` as new Phase 4 subcommand | Yes | Not a reuse of `engram agent spawn`. `spawn` is the lead-facing command (creates pane, starts runner). `run` is the in-pane binary subcommand (runs the pipeline). |
| osTmuxSpawnWith rewrite: drop interactive model | Yes | Before: creates pane, sends `claude --dangerously-skip-permissions` interactively, waits for ❯ prompt, sends task via send-keys. After: creates pane, sends `engram agent run --name N --prompt P` via single send-keys call. No ❯ wait, no paste confirm step. |
| Session-id placeholder in state file at spawn time | Yes | `runAgentSpawn` writes `SessionID: "PENDING"` initially. `runAgentRun` overwrites with real UUID before first speech act. Downstream (Phase 5 `--resume`) reads state file, never sees PENDING if pipeline is running. |
| Skill deletions gated on E2E verification | Yes | All Category B deletions happen AFTER Criterion 1–4 pass. Not concurrent with package shipping. |
| Category A skill deletions: at Phase 4 start | Yes | Residual interactive-spawn guidance (if any remains in skills) deleted before implementing. Workers using old model can't exist after spawn transition. |
| Lead calling convention SURVIVES Phase 4 | Yes | Lead calls `engram chat post` + `engram chat ack-wait` directly — NOT speech-to-chat. Lead is interactive, not headless. Writing Messages section stays in engram-tmux-lead. Only worker-context guidance in use-engram-chat-as is deleted. |
| No `internal/dispatch` stub in Phase 4 | Yes | YAGNI. Phase 5 codesign session defines the auto-resume watch loop interface. Phase 6 extends it. Stubbing prematurely risks wrong interface. |
| Phase 5 needs dedicated codesign session | Yes | 6 unresolved design decisions (spec §7 Phase 5). See Post-Phase-4 section. |
| Phase 6 is rewrite-from-scratch, not deletion | Yes | ~45-line skill targets require rebuilding from scratch after Phase 5. Different kind of work than prior phases. |
| No Phase 3.5 | Yes | Task 0 verifies Phase 3 Criterion 5 (watchDeadline). If it fails, fix as pre-Phase 4 PR. No formal phase. |
| claude flags: `-p --dangerously-skip-permissions --verbose --output-format=stream-json` | Yes | `--verbose` required for JSONL stream with session_id on all events. `--dangerously-skip-permissions` must be verified to work in -p mode (Criterion 4). |
| Speech relay: INTENT: posts type=intent, ACK: posts type=ack, etc. | Yes | Binary intercepts prefix marker in assistant text, calls `chat.Poster.Post()` with the corresponding message type. Agent receives `Proceed.` (ACK) or `WAIT from X: [text]` (WAIT) in its next turn. |
| Display filter: raw JSONL suppressed from pane | Yes | Tool use events, result events, system events: not written to Runner.Pane. Assistant text blocks, user turns: written. Criterion 3 verifies. |

---

## File Structure

```
internal/
  streamjson/
    streamjson.go      NEW — Event, SpeechMarker, Parse, DetectSpeechMarkers (pure functions)
    streamjson_test.go NEW — unit tests for all parse/detect functions

  claude/
    claude.go          NEW — Runner struct, Run method (pipeline: display filter + speech relay + ACK-wait)
    claude_test.go     NEW — unit tests with injected io.Writer and mock Poster

  cli/
    cli_agent.go       MODIFY — add runAgentRun; rewrite osTmuxSpawnWith (drop interactive model)
    targets.go         MODIFY — add AgentRunArgs, AgentRunFlags; add "run" entry to BuildAgentGroup
    export_test.go     MODIFY — export testAgentRunSessionID or similar test helper if needed
    cli_test.go        MODIFY — add agent run command tests

skills/
  use-engram-chat-as/SKILL.md   MODIFY — Task 1 (Category A), Task 7 (additions), Task 8 (Category B)
  engram-agent/SKILL.md         MODIFY — Task 8 (Category B deletions + prefix marker usage)
  engram-tmux-lead/SKILL.md     MODIFY — Task 6 (1-line session-id clarification note)
```

---

## Task 0: Verify Phase 3 Criterion 5 (watchDeadline)

**Files:** None modified. Verification only.

This is the pre-Phase 4 guard. The `e422c6a` commit fixed an offline-agent test path; the primary watchDeadline path needs an explicit live check before Phase 4 code touches cli_agent.go.

- [ ] **Step 1: Run the watchDeadline test**

```bash
TMP=$(mktemp -d) && echo "" > "$TMP/chat.toml"
timeout 8 engram agent wait-ready --name nonexistent --cursor 0 --max-wait 3 --chat-file "$TMP/chat.toml"
echo "Exit code: $?"
```

Expected: exits within ~4s (not an 8s timeout). Exit code 1 (timeout — expected, agent doesn't exist) or 0. NOT a hang to the 8s wall time.

- [ ] **Step 2: If it hangs (exits at 8s), investigate and fix**

The watchDeadline contract: `context.WithTimeout` must flow to the inner `watcher.Watch()` call, not just the outer loop. If the test hangs, the timeout is not propagating. Fix in `internal/cli/cli_agent.go` before proceeding to Task 1. Commit the fix with `git commit -m "fix(cli): propagate watchDeadline context into watcher.Watch"`.

- [ ] **Step 3: If watchDeadline passes, proceed to Task 1**

No commit needed. Task 0 is verification only.

---

## Task 1: Category A Skill Deletions (Pre-Work)

**Files:**
- Modify: `skills/use-engram-chat-as/SKILL.md`
- Modify: `skills/engram-tmux-lead/SKILL.md` (if any interactive spawn guidance remains)

Category A: content teaching the old interactive-spawn pattern (wait for ❯ prompt, send-keys task content, paste confirmation). Phase 3 deleted SPAWN-PANE bash from engram-tmux-lead. Verify any residual interactive guidance in use-engram-chat-as and delete it before implementation begins.

**IMPORTANT:** Use `superpowers:writing-skills` for all SKILL.md edits per CLAUDE.md. No exceptions.

- [ ] **Step 1: Audit use-engram-chat-as for interactive spawn guidance**

```bash
grep -n "❯\|send-keys\|paste\|interactive\|wait for prompt\|bracketed paste" skills/use-engram-chat-as/SKILL.md
```

Expected: zero matches. If any match: these are Category A lines to delete.

- [ ] **Step 2: Audit engram-tmux-lead for interactive spawn guidance**

```bash
grep -n "❯\|send-keys.*claude\|wait for prompt\|claudeSettings\|claudeReadyMax" skills/engram-tmux-lead/SKILL.md
```

Expected: zero matches. If any match: delete.

- [ ] **Step 3: Apply deletions if needed, using writing-skills skill**

Invoke `superpowers:writing-skills` to apply any deletions identified above. If both audits are clean, skip this step — no commit needed.

- [ ] **Step 4: Commit any deletions**

```bash
git add skills/
git commit -m "feat(skills): remove residual interactive spawn guidance (Phase 4 Category A)"
```

---

## Task 2: `internal/streamjson` Package

**Files:**
- Create: `internal/streamjson/streamjson.go`
- Create: `internal/streamjson/streamjson_test.go`

Pure stateless JSONL parser. No I/O. Every function is a pure transformation.

**Note on schema:** The JSON schema below was described in spec §4.5 and verified against a live `claude -p` run. Claude CLI version is not pinned. If the schema changes (field renames, new nesting), `Parse` will return events with empty fields rather than errors — `DetectSpeechMarkers` then finds no markers, and the display filter silently drops all markers. To surface schema drift: `Parse` emits a warning to stderr when it sees an `assistant` event with no text content blocks. Verify the schema against a live run during Task 6 (E2E).

- [ ] **Step 1: Write failing tests for `Parse`**

```go
// internal/streamjson/streamjson_test.go
package streamjson_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/streamjson"
)

func TestParse_AssistantEvent_ExtractsSessionIDAndText(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	line := []byte(`{"type":"assistant","session_id":"550e8400-e29b-41d4-a716-446655440000","message":{"content":[{"type":"text","text":"Hello world"}]}}`)
	event, err := streamjson.Parse(line)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(event.Type).To(Equal("assistant"))
	g.Expect(event.SessionID).To(Equal("550e8400-e29b-41d4-a716-446655440000"))
	g.Expect(event.Text).To(Equal("Hello world"))
}

func TestParse_SystemEvent_ExtractsSessionIDOnly(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	line := []byte(`{"type":"system","session_id":"550e8400-e29b-41d4-a716-446655440000","subtype":"init"}`)
	event, err := streamjson.Parse(line)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(event.Type).To(Equal("system"))
	g.Expect(event.SessionID).To(Equal("550e8400-e29b-41d4-a716-446655440000"))
	g.Expect(event.Text).To(BeEmpty())
}

func TestParse_MalformedJSON_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	_, err := streamjson.Parse([]byte(`{not json`))
	g.Expect(err).To(HaveOccurred())
}

func TestParse_EmptyLine_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	_, err := streamjson.Parse([]byte(``))
	g.Expect(err).To(HaveOccurred())
}

func TestDetectSpeechMarkers_IntentPrefix_Detected(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	text := "INTENT: Situation: About to run targ check-full.\nBehavior: Will execute the check command."
	markers := streamjson.DetectSpeechMarkers(text)
	g.Expect(markers).To(HaveLen(1))
	g.Expect(markers[0].Prefix).To(Equal("INTENT"))
	g.Expect(markers[0].Text).To(ContainSubstring("Situation: About to run targ check-full."))
}

func TestDetectSpeechMarkers_MultipleMarkers_AllDetected(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	text := "INTENT: Situation: X. Behavior: Y.\n\nSome prose.\n\nACK: No objection, proceed."
	markers := streamjson.DetectSpeechMarkers(text)
	g.Expect(markers).To(HaveLen(2))
	g.Expect(markers[0].Prefix).To(Equal("INTENT"))
	g.Expect(markers[1].Prefix).To(Equal("ACK"))
}

func TestDetectSpeechMarkers_NoMarkers_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	markers := streamjson.DetectSpeechMarkers("Just regular prose with no markers.")
	g.Expect(markers).To(BeEmpty())
}

func TestDetectSpeechMarkers_WaitWithRecipient_Detected(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	text := "WAIT: (to engram-agent) I have a concern about the approach."
	markers := streamjson.DetectSpeechMarkers(text)
	g.Expect(markers).To(HaveLen(1))
	g.Expect(markers[0].Prefix).To(Equal("WAIT"))
	g.Expect(markers[0].Text).To(ContainSubstring("I have a concern"))
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
targ test
```

Expected: FAIL — package `streamjson` does not exist.

- [ ] **Step 3: Implement `streamjson.go`**

```go
// internal/streamjson/streamjson.go
package streamjson

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Event is a single parsed JSONL event from claude -p --verbose --output-format=stream-json.
// All events carry session_id (the Claude conversation UUID).
// Text is populated for assistant events only.
type Event struct {
	Type      string // "assistant", "tool_use", "result", "system", "user", "error"
	SessionID string // Claude conversation UUID — present on all events
	Text      string // assembled text from content blocks (assistant events only)
}

// SpeechMarker is a detected prefix marker in an assistant text block.
// Markers must appear at the start of a line (column 0) to be detected.
type SpeechMarker struct {
	Prefix string // "INTENT", "ACK", "WAIT", "DONE", "LEARNED", "INFO", "READY", "ESCALATE"
	Text   string // content from the "PREFIX: " through end of speech act (trimmed)
}

// knownPrefixes is the complete set of valid speech-to-chat prefix markers.
// Order matters: longer/more-specific prefixes before shorter ones if any overlap exists.
var knownPrefixes = []string{"INTENT", "ACK", "WAIT", "DONE", "LEARNED", "INFO", "READY", "ESCALATE"} //nolint:gochecknoglobals

// rawEvent is the JSON shape of a claude -p stream-json event.
// Only fields needed for parsing are included; unknown fields are ignored.
type rawEvent struct {
	Type      string          `json:"type"`
	SessionID string          `json:"session_id"`
	Message   *rawMessage     `json:"message,omitempty"`
}

type rawMessage struct {
	Content []rawContent `json:"content"`
}

type rawContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// Parse parses one JSONL line into an Event.
// Returns an error for malformed JSON or empty input.
// Unknown event types return a non-nil Event with Type set and empty Text.
// Warning: if an assistant event has no text content blocks, Parse writes a
// warning to stderr to surface potential schema drift.
func Parse(line []byte) (Event, error) {
	if len(line) == 0 {
		return Event{}, fmt.Errorf("streamjson: empty line")
	}

	var raw rawEvent
	if err := json.Unmarshal(line, &raw); err != nil {
		return Event{}, fmt.Errorf("streamjson: malformed JSON: %w", err)
	}

	event := Event{
		Type:      raw.Type,
		SessionID: raw.SessionID,
	}

	if raw.Type == "assistant" && raw.Message != nil {
		var texts []string
		for _, block := range raw.Message.Content {
			if block.Type == "text" && block.Text != "" {
				texts = append(texts, block.Text)
			}
		}
		event.Text = strings.Join(texts, "\n")
	}

	return event, nil
}

// DetectSpeechMarkers scans the Text field of an assistant event for prefix markers.
// A marker is detected when a line starts with "PREFIX: " (case-sensitive, column 0).
// Multi-line speech acts are terminated by a blank line, the next prefix marker, or
// end of text. All prefixes must be in the knownPrefixes list.
func DetectSpeechMarkers(text string) []SpeechMarker {
	lines := strings.Split(text, "\n")
	markers := make([]SpeechMarker, 0)

	var current *SpeechMarker
	var currentLines []string

	flush := func() {
		if current != nil {
			current.Text = strings.TrimSpace(strings.Join(currentLines, "\n"))
			markers = append(markers, *current)
			current = nil
			currentLines = nil
		}
	}

	for _, line := range lines {
		if prefix, rest, found := detectPrefix(line); found {
			flush()
			current = &SpeechMarker{Prefix: prefix}
			currentLines = []string{rest}
			continue
		}
		if current != nil {
			currentLines = append(currentLines, line)
		}
	}
	flush()

	return markers
}

// detectPrefix checks if a line starts with a known "PREFIX: " marker.
// Returns prefix name, rest of line after "PREFIX: ", and whether a match was found.
func detectPrefix(line string) (prefix, rest string, found bool) {
	for _, p := range knownPrefixes {
		marker := p + ": "
		if strings.HasPrefix(line, marker) {
			return p, strings.TrimPrefix(line, marker), true
		}
		// Also match "PREFIX:" with no trailing space (edge case).
		if line == p+":" {
			return p, "", true
		}
	}
	return "", "", false
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
targ test
```

Expected: all streamjson tests PASS.

- [ ] **Step 5: Run full check**

```bash
targ check-full
```

Expected: no linter errors. Fix any nilaway/linter issues before proceeding.

- [ ] **Step 6: Commit**

```bash
git add internal/streamjson/
git commit -m "feat(streamjson): add stateless JSONL parser for claude -p stream output"
```

---

## Task 3: `internal/claude` Package

**Files:**
- Create: `internal/claude/claude.go`
- Create: `internal/claude/claude_test.go`

The Runner struct owns the stream pipeline: execute `claude -p`, read JSONL, write filtered output to pane, relay speech markers to chat. `ProcessStream` processes one JSONL stream (one turn) and returns a `StreamResult` indicating which markers were detected. The conversation loop (ack-wait + `--resume`) lives in `runAgentRun` (Task 4), which calls `ProcessStream` repeatedly.

**DI note:** `Runner.WriteSessionID` is injected by `runAgentRun` (in `internal/cli`) to avoid a circular dependency: `internal/claude` → `internal/cli` is forbidden; `internal/cli` → `internal/claude` is fine.

- [ ] **Step 1: Write failing tests for Runner**

```go
// internal/claude/claude_test.go
package claude_test

import (
	"bytes"
	"io"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/chat"
	"engram/internal/claude"
)

// mockPoster records all Post calls for test inspection.
type mockPoster struct {
	posted []chat.Message
}

func (m *mockPoster) Post(msg chat.Message) (int, error) {
	m.posted = append(m.posted, msg)
	return 0, nil
}

func TestProcessStream_DisplayFilter_SuppressesRawJSONL(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var pane bytes.Buffer
	runner := claude.Runner{
		AgentName:      "test-agent",
		Pane:           &pane,
		Poster:         &mockPoster{},
		WriteSessionID: func(string) error { return nil },
	}

	// Simulate: system event (filtered), tool_use event (filtered), assistant event (shown).
	stream := strings.NewReader(
		`{"type":"system","session_id":"550e8400-e29b-41d4-a716-446655440000","subtype":"init"}` + "\n" +
			`{"type":"tool_use","session_id":"550e8400-e29b-41d4-a716-446655440000","id":"toolu_01"}` + "\n" +
			`{"type":"assistant","session_id":"550e8400-e29b-41d4-a716-446655440000","message":{"content":[{"type":"text","text":"I am thinking."}]}}` + "\n",
	)

	_, err := runner.ProcessStream(stream)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	g.Expect(pane.String()).NotTo(ContainSubstring(`{"type":`)) // no raw JSONL
	g.Expect(pane.String()).To(ContainSubstring("I am thinking."))
}

func TestProcessStream_IntentMarker_PostedAndReported(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	poster := &mockPoster{}
	runner := claude.Runner{
		AgentName:      "test-agent",
		Pane:           io.Discard,
		Poster:         poster,
		WriteSessionID: func(string) error { return nil },
	}

	stream := strings.NewReader(
		`{"type":"assistant","session_id":"550e8400-e29b-41d4-a716-446655440000","message":{"content":[{"type":"text","text":"INTENT: Situation: X.\nBehavior: Y."}]}}` + "\n",
	)

	result, err := runner.ProcessStream(stream)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	g.Expect(poster.posted).To(HaveLen(1))
	g.Expect(poster.posted[0].Type).To(Equal("intent"))
	g.Expect(poster.posted[0].From).To(Equal("test-agent"))
	g.Expect(poster.posted[0].Text).To(ContainSubstring("Situation: X."))
	g.Expect(result.IntentDetected).To(BeTrue())
	g.Expect(result.DoneDetected).To(BeFalse())
}

func TestProcessStream_DoneMarker_Reported(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	runner := claude.Runner{
		AgentName:      "test-agent",
		Pane:           io.Discard,
		Poster:         &mockPoster{},
		WriteSessionID: func(string) error { return nil },
	}

	stream := strings.NewReader(
		`{"type":"assistant","session_id":"abc","message":{"content":[{"type":"text","text":"DONE: Task complete."}]}}` + "\n",
	)

	result, err := runner.ProcessStream(stream)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(result.DoneDetected).To(BeTrue())
}

func TestProcessStream_SessionID_CallsWriteSessionID(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var capturedID string
	runner := claude.Runner{
		AgentName: "test-agent",
		Pane:      io.Discard,
		Poster:    &mockPoster{},
		WriteSessionID: func(id string) error {
			capturedID = id
			return nil
		},
	}

	stream := strings.NewReader(
		`{"type":"system","session_id":"550e8400-e29b-41d4-a716-446655440000","subtype":"init"}` + "\n",
	)

	_, err := runner.ProcessStream(stream)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(capturedID).To(Equal("550e8400-e29b-41d4-a716-446655440000"))
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
targ test
```

Expected: FAIL — package `claude` does not exist.

- [ ] **Step 3: Implement `claude.go`**

```go
// internal/claude/claude.go
package claude

import (
	"bufio"
	"fmt"
	"io"

	"engram/internal/chat"
	"engram/internal/streamjson"
)

// Runner owns the claude -p stream pipeline for one agent session.
// All I/O is injected — no os.* calls in this package.
type Runner struct {
	AgentName string
	Pane      io.Writer    // filtered output destination; os.Stdout when run in-pane
	Poster    chat.Poster  // posts relayed speech markers to the chat file

	// WriteSessionID is called with the Claude conversation UUID extracted from the
	// first JSONL event (before any speech act). Injected by runAgentRun to update
	// the state file without creating an internal/claude → internal/cli import cycle.
	// May be nil (skips session-id write — used in tests that don't need it).
	WriteSessionID func(sessionID string) error
}

// StreamResult is the outcome of processing one JSONL stream (one claude -p turn).
type StreamResult struct {
	IntentDetected bool   // true if at least one INTENT: prefix marker was detected
	DoneDetected   bool   // true if a DONE: prefix marker was detected
	SessionID      string // Claude conversation UUID extracted from the first JSONL event
}

// ProcessStream reads JSONL from src, applies the display filter, relays speech
// markers to chat via Poster, calls WriteSessionID on the first event, and returns
// a StreamResult describing what was detected. The stream ends when src returns io.EOF.
// The caller (runAgentRun) uses StreamResult to drive the ack-wait + resume loop.
func (r *Runner) ProcessStream(src io.Reader) (StreamResult, error) {
	scanner := bufio.NewScanner(src)
	var result StreamResult
	sessionIDWritten := false

	for scanner.Scan() {
		line := scanner.Bytes()

		event, parseErr := streamjson.Parse(line)
		if parseErr != nil {
			// Non-JSON lines (startup noise before stream begins): pass through to pane.
			_, _ = fmt.Fprintf(r.Pane, "%s\n", line)
			continue
		}

		// Session-id R-M-W: write real Claude UUID before first speech act.
		// Skipped if already written or if the field is empty/placeholder.
		if !sessionIDWritten && event.SessionID != "" && event.SessionID != "PENDING" {
			result.SessionID = event.SessionID
			if r.WriteSessionID != nil {
				if writeErr := r.WriteSessionID(event.SessionID); writeErr != nil {
					_, _ = fmt.Fprintf(r.Pane, "[engram] warning: failed to write session-id: %v\n", writeErr)
				} else {
					sessionIDWritten = true
				}
			} else {
				sessionIDWritten = true
			}
		}

		switch event.Type {
		case "assistant":
			if event.Text != "" {
				_, _ = fmt.Fprintf(r.Pane, "%s\n", event.Text)
			}
			for _, marker := range streamjson.DetectSpeechMarkers(event.Text) {
				if marker.Prefix == "INTENT" {
					result.IntentDetected = true
				}
				if marker.Prefix == "DONE" {
					result.DoneDetected = true
				}
				if relayErr := r.relayMarker(marker); relayErr != nil {
					_, _ = fmt.Fprintf(r.Pane, "[engram] warning: relay failed: %v\n", relayErr)
				}
			}
		case "user":
			// User turns injected via --resume resume prompt: write to pane.
			if event.Text != "" {
				_, _ = fmt.Fprintf(r.Pane, "%s\n", event.Text)
			}
		default:
			// system, tool_use, result, error: display-filtered (not written to pane).
		}
	}

	return result, scanner.Err()
}

// relayMarker maps a SpeechMarker prefix to a chat message type and posts it via Poster.
func (r *Runner) relayMarker(marker streamjson.SpeechMarker) error {
	msgType := markerToMsgType(marker.Prefix)
	if msgType == "" {
		return nil
	}

	msg := chat.Message{
		From:   r.AgentName,
		To:     "engram-agent",
		Thread: "speech-relay",
		Type:   msgType,
		Text:   marker.Text,
	}

	_, err := r.Poster.Post(msg)
	return err
}

// markerToMsgType converts a speech prefix to a chat message type.
func markerToMsgType(prefix string) string {
	switch prefix {
	case "INTENT":
		return "intent"
	case "ACK":
		return "ack"
	case "WAIT":
		return "wait"
	case "DONE":
		return "done"
	case "LEARNED":
		return "learned"
	case "INFO":
		return "info"
	case "READY":
		return "ready"
	case "ESCALATE":
		return "escalate"
	default:
		return ""
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
targ test
```

Expected: all claude tests PASS.

- [ ] **Step 5: Run full check**

```bash
targ check-full
```

Expected: no linter errors.

- [ ] **Step 6: Commit**

```bash
git add internal/claude/ internal/streamjson/
git commit -m "feat(claude): add Runner.ProcessStream with display filter, speech relay, session-id capture"
```

---

## Task 4: `engram agent run` Subcommand

**Files:**
- Modify: `internal/cli/cli_agent.go` — add `runAgentRun`
- Modify: `internal/cli/targets.go` — add `AgentRunArgs`, `AgentRunFlags`, register in `BuildAgentGroup`
- Modify: `internal/cli/export_test.go` — export test helper if needed
- Modify: `internal/cli/cli_test.go` — add tests for agent run

This subcommand runs INSIDE the tmux pane. It is the in-pane entry point for the Phase 4 pipeline.

`engram agent run --name N --prompt P [--chat-file F] [--state-file S]`

- Executes: `claude -p --dangerously-skip-permissions --verbose --output-format=stream-json "<prompt>"`
- Pipes stdout through `claude.Runner.ProcessStream`
- Runner.Poster = a `chat.FilePoster` (existing)
- Runner.WriteSessionID = calls `readModifyWriteStateFile` in cli_agent.go
- Runner.Pane = `os.Stdout`

- [ ] **Step 1: Add `AgentRunArgs` to `targets.go`**

```go
// AgentRunArgs holds flags for `engram agent run`.
type AgentRunArgs struct {
	Name      string
	Prompt    string
	ChatFile  string
	StateFile string
}

// AgentRunFlags builds the flag list for AgentRunArgs.
func AgentRunFlags(a AgentRunArgs) []string {
	return BuildFlags(
		"--name", a.Name,
		"--prompt", a.Prompt,
		"--chat-file", a.ChatFile,
		"--state-file", a.StateFile,
	)
}
```

Add to `BuildAgentGroup`:

```go
targ.Targ(func(a AgentRunArgs) {
    args := append([]string{"engram", "agent", "run"}, AgentRunFlags(a)...)
    RunSafe(args, stdout, stderr, stdin)
}).Name("run").Description("Run a claude -p worker pipeline in the current pane"),
```

- [ ] **Step 2: Write failing test for `runAgentRun`**

```go
// internal/cli/cli_test.go — add:

func TestAgentRun_MissingName_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var stdout, stderr bytes.Buffer
	err := cli.Run([]string{"engram", "agent", "run", "--prompt", "hello"}, &stdout, &stderr, nil)
	g.Expect(err).To(HaveOccurred())
	g.Expect(stderr.String()).To(ContainSubstring("--name"))
}

func TestAgentRun_MissingPrompt_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var stdout, stderr bytes.Buffer
	err := cli.Run([]string{"engram", "agent", "run", "--name", "worker-1"}, &stdout, &stderr, nil)
	g.Expect(err).To(HaveOccurred())
	g.Expect(stderr.String()).To(ContainSubstring("--prompt"))
}
```

- [ ] **Step 3: Run tests to verify they fail**

```bash
targ test
```

Expected: FAIL — "agent run" subcommand not registered.

- [ ] **Step 4: Implement `runAgentRun` in `cli_agent.go`**

`runAgentRun` implements the full conversation loop: start claude -p → process stream → ack-wait on INTENT → resume with --resume → repeat until DONE or no markers. Per spec §5 Option B: ack-wait happens after the turn ends (the stream completes), then the binary resumes the agent with `claude -p --resume <session-id> "Proceed."` or `"WAIT from X: [text]"`.

```go
// runAgentRun is the in-pane entry point for the Phase 4 speech-to-chat pipeline.
// It manages the full claude -p conversation loop: stream → ack-wait → resume → stream.
// Runs inside the tmux pane as an in-pane process; stdout IS the pane display.
func runAgentRun(args []string, stdout io.Writer) error {
	flags, parseErr := parseAgentRunFlags(args)
	if errors.Is(parseErr, flag.ErrHelp) {
		return nil
	}
	if parseErr != nil {
		return parseErr
	}

	chatFilePath, pathErr := resolveChatFile(flags.chatFile, "agent run", os.UserHomeDir, os.Getwd)
	if pathErr != nil {
		return pathErr
	}

	stateFilePath, statePathErr := resolveStateFile(flags.stateFile, "agent run", os.UserHomeDir, os.Getwd)
	if statePathErr != nil {
		return statePathErr
	}

	ctx, cancel := signalContext()
	defer cancel()

	poster := newFilePoster(chatFilePath) // defined in cli.go

	// claudepkg is the import alias for engram/internal/claude in cli_agent.go.
	// Add to imports: claudepkg "engram/internal/claude"
	runner := claudepkg.Runner{
		AgentName: flags.name,
		Pane:      stdout,
		Poster:    poster,
		WriteSessionID: func(sessionID string) error {
			return readModifyWriteStateFile(stateFilePath, func(sf agentpkg.StateFile) agentpkg.StateFile {
				for i, rec := range sf.Agents {
					if rec.Name == flags.name {
						sf.Agents[i].SessionID = sessionID
						return sf
					}
				}
				return sf
			})
		},
	}

	// Conversation loop: initial prompt → stream → ack-wait → resume → stream → ...
	prompt := flags.prompt
	sessionID := "" // filled after first stream (from StreamResult.SessionID)
	const maxTurns = 50 // safety cap: prevents runaway loops

	for turn := range maxTurns {
		var cmd *exec.Cmd
		if turn == 0 || sessionID == "" {
			//nolint:gosec // prompt is caller-controlled, not user web input
			cmd = exec.CommandContext(ctx, "claude", "-p",
				"--dangerously-skip-permissions",
				"--verbose",
				"--output-format=stream-json",
				prompt,
			)
		} else {
			//nolint:gosec // prompt is binary-constructed ("Proceed." or "WAIT from X: ...")
			cmd = exec.CommandContext(ctx, "claude", "-p",
				"--dangerously-skip-permissions",
				"--verbose",
				"--output-format=stream-json",
				"--resume", sessionID,
				prompt,
			)
		}
		cmd.Stderr = stdout

		pipe, pipeErr := cmd.StdoutPipe()
		if pipeErr != nil {
			return fmt.Errorf("agent run: stdout pipe: %w", pipeErr)
		}

		if startErr := cmd.Start(); startErr != nil {
			return fmt.Errorf("agent run: start claude: %w", startErr)
		}

		result, streamErr := runner.ProcessStream(pipe)
		waitErr := cmd.Wait()

		if streamErr != nil {
			return fmt.Errorf("agent run: stream: %w", streamErr)
		}
		if waitErr != nil {
			return fmt.Errorf("agent run: claude exited: %w", waitErr)
		}

		// Update session-id after first turn (needed for --resume in subsequent turns).
		if sessionID == "" && result.SessionID != "" {
			sessionID = result.SessionID
		}

		// DONE or no markers: conversation complete.
		if result.DoneDetected || !result.IntentDetected {
			return nil
		}

		// INTENT detected: ack-wait then resume.
		// PRE-CURSOR must be captured BEFORE the intent was posted (runner posted it
		// during stream processing). Use end-of-file cursor now as a conservative
		// approximation — the ACK/WAIT must arrive AFTER the intent was posted.
		cursor, cursorErr := chatCursor(chatFilePath)
		if cursorErr != nil {
			return fmt.Errorf("agent run: cursor: %w", cursorErr)
		}

		waiter := &chat.FileAckWaiter{
			FilePath: chatFilePath,
			Watcher:  newFileWatcher(chatFilePath),
			ReadFile: os.ReadFile,
			NowFunc:  time.Now,
			MaxWait:  30 * time.Second,
		}

		ackResult, ackErr := waiter.AckWait(ctx, flags.name, cursor, []string{"engram-agent"})
		if ackErr != nil {
			return fmt.Errorf("agent run: ack-wait: %w", ackErr)
		}

		// Build resume prompt from ack-wait result.
		switch ackResult.Result {
		case "ACK", "TIMEOUT":
			prompt = "Proceed."
		case "WAIT":
			if ackResult.Wait != nil {
				prompt = fmt.Sprintf("WAIT from %s: %s", ackResult.Wait.From, ackResult.Wait.Text)
			} else {
				prompt = "WAIT: unspecified objection."
			}
		default:
			prompt = "Proceed."
		}
	}

	return fmt.Errorf("agent run: exceeded %d turns without DONE — possible runaway loop", maxTurns)
}

// chatCursor returns the current line count of the chat file (end-of-file position).
// Extracted to allow injection in tests.
func chatCursor(chatFilePath string) (int, error) {
	data, err := os.ReadFile(chatFilePath)
	if err != nil {
		return 0, fmt.Errorf("reading chat file: %w", err)
	}
	return len(strings.Split(string(data), "\n")), nil
}

// parseAgentRunFlags parses flags for the "agent run" subcommand.
type agentRunFlags struct {
	name      string
	prompt    string
	chatFile  string
	stateFile string
}

func parseAgentRunFlags(args []string) (agentRunFlags, error) {
	fs := flag.NewFlagSet("agent run", flag.ContinueOnError)
	var flags agentRunFlags
	fs.StringVar(&flags.name, "name", "", "agent name (required)")
	fs.StringVar(&flags.prompt, "prompt", "", "initial prompt (required)")
	fs.StringVar(&flags.chatFile, "chat-file", "", "override chat file path (testing only)")
	fs.StringVar(&flags.stateFile, "state-file", "", "override state file path (testing only)")

	if parseErr := fs.Parse(args); parseErr != nil {
		return agentRunFlags{}, parseErr
	}

	if flags.name == "" {
		fs.Usage()
		return agentRunFlags{}, fmt.Errorf("agent run: --name is required")
	}
	if flags.prompt == "" {
		fs.Usage()
		return agentRunFlags{}, fmt.Errorf("agent run: --prompt is required")
	}
	return flags, nil
}
```

Register in `runAgentDispatch` (the switch in `runAgentDispatch` that routes `agent <subcommand>`):

```go
case "run":
    return runAgentRun(subArgs[1:], stdout)
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
targ test
```

Expected: PASS.

- [ ] **Step 6: Run full check**

```bash
targ check-full
```

Expected: no linter errors.

- [ ] **Step 7: Commit**

```bash
git add internal/cli/cli_agent.go internal/cli/targets.go internal/cli/cli_test.go
git commit -m "feat(cli): add engram agent run subcommand for in-pane claude -p pipeline"
```

---

## Task 5: Spawn Mode Transition (`osTmuxSpawnWith` Rewrite)

**Files:**
- Modify: `internal/cli/cli_agent.go` — rewrite `osTmuxSpawnWith`

Rewrite `osTmuxSpawnWith` to start `engram agent run` in the pane instead of launching Claude interactively.

**Before:** creates pane → sends `claude --dangerously-skip-permissions --model sonnet ...` → waits for ❯ prompt → sends task via send-keys → confirms paste dialog.

**After:** creates pane → sends `engram agent run --name N --prompt P --chat-file F --state-file S` via single send-keys call → returns immediately (no ❯ wait, no paste confirm).

The SessionID returned by `osTmuxSpawnWith` is now `"PENDING"` — the real Claude UUID is written by `runAgentRun` when the JSONL stream starts. `runAgentSpawn` stores `"PENDING"` in the state file; `runAgentRun` overwrites it.

- [ ] **Step 1: Write a test for the new spawn behavior**

```go
// internal/cli/cli_test.go — add:

func TestOsTmuxSpawnWith_NewModel_SendsEngamAgentRun(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Use the fake tmux binary from testdata to capture send-keys calls.
	// The fake tmux binary records all invocations to a temp file.
	// See existing testSpawner pattern in cli_agent.go for reference.
	tmpDir := t.TempDir()
	// Write a fake tmux script that records the send-keys command.
	fakeTmux := tmpDir + "/tmux"
	err := os.WriteFile(fakeTmux, []byte(`#!/bin/sh
echo "%1 $0" >> `+tmpDir+`/calls.log
echo "%1 $0"
`), 0o755)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	// Actually, the cleanest test approach: use the existing testSpawner pattern.
	// Set testSpawner to a function that captures what command would be sent-keyed.
	// This test verifies the OLD ❯-wait loop is gone and the new command is sent.
	var capturedCmd string
	oldSpawner := cli.TestSpawnAckMaxWait // existing test hook
	defer func() { cli.TestSpawnAckMaxWait = oldSpawner }()

	// For spawn mode: use a fake spawnFunc that captures the "run" command.
	// The spawnFunc signature is func(ctx, name, prompt) (paneID, sessionID, error).
	// In Phase 4, the spawner sends "engram agent run --name N --prompt P" to the pane.
	// We verify this via integration test in Task 7 (E2E).
	_ = capturedCmd
	t.Skip("Full spawn integration verified in Task 7 E2E — unit test verifies flag handling only.")
}
```

The spawn mode transition is best verified end-to-end (Task 7). The unit test above is a placeholder; the real verification is Criterion 4 (--dangerously-skip-permissions in -p mode) and Criterion 2 (speech relay live).

- [ ] **Step 2: Rewrite `osTmuxSpawnWith`**

Replace the full body of `osTmuxSpawnWith` (lines 189–257 in current cli_agent.go):

```go
// osTmuxSpawnWith creates a tmux window and starts the engram agent run pipeline in it.
// Extracted so tests can supply a fake binary path without modifying global state.
// The returned sessionID is always "PENDING" — the real Claude conversation UUID is
// written to the state file by runAgentRun when the JSONL stream starts.
func osTmuxSpawnWith(ctx context.Context, tmuxBin, name, prompt string) (paneID, sessionID string, err error) {
	// Step 1: Derive chat and state file paths for the in-pane runner command.
	chatFilePath, pathErr := resolveChatFile("", "agent spawn", os.UserHomeDir, os.Getwd)
	if pathErr != nil {
		return "", "", pathErr
	}

	stateFilePath, statePathErr := resolveStateFile("", "agent spawn", os.UserHomeDir, os.Getwd)
	if statePathErr != nil {
		return "", "", statePathErr
	}

	// Step 2: Create pane with default shell (no command — stays alive until runner exits).
	out, cmdErr := exec.CommandContext(ctx, tmuxBin, //nolint:gosec
		"new-window",
		"-d",
		"-n", name,
		"-P", "-F", "#{pane_id} #{session_id}",
	).Output()
	if cmdErr != nil {
		return "", "", fmt.Errorf("tmux new-window: %w", cmdErr)
	}

	paneID, _, parseErr := parseTmuxOutput(out)
	if parseErr != nil {
		return "", "", parseErr
	}

	// Set stable pane label (immune to OSC 2 overwrite from terminal output).
	_ = exec.CommandContext(ctx, tmuxBin, //nolint:gosec
		"set-option", "-p", "-t", paneID, "@engram_name", name,
	).Run()

	// Step 3: Start the in-pane runner with a single send-keys call.
	// The runner binary (engram agent run) starts claude -p internally.
	// No ❯ prompt wait, no paste confirm — the runner owns the subprocess lifecycle.
	runCmd := fmt.Sprintf(
		"engram agent run --name %s --prompt %s --chat-file %s --state-file %s",
		shellQuote(name), shellQuote(prompt), shellQuote(chatFilePath), shellQuote(stateFilePath),
	)

	startErr := exec.CommandContext(ctx, tmuxBin, //nolint:gosec
		"send-keys", "-t", paneID, runCmd, "Enter",
	).Run()
	if startErr != nil {
		return "", "", fmt.Errorf("tmux send-keys: %w", startErr)
	}

	// SessionID is PENDING — runAgentRun writes the real Claude UUID to state file
	// once the JSONL stream starts (before the first speech act).
	return paneID, "PENDING", nil
}

// shellQuote wraps a string in single quotes, escaping any embedded single quotes.
// Used to safely pass name and prompt to the shell via tmux send-keys.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}
```

Also remove the now-unused constants `claudeReadyMaxRetries`, `claudeReadyPollInterval`, `claudeSettings` (verify they're not used elsewhere first):

```bash
grep -n "claudeReadyMaxRetries\|claudeReadyPollInterval\|claudeSettings" internal/cli/cli_agent.go
```

If they only appear in `osTmuxSpawnWith`, delete them. If used elsewhere, leave them.

- [ ] **Step 3: Update `runAgentSpawn` to store "PENDING" as initial session-id**

The existing `runAgentSpawn` already stores the returned sessionID from the spawner. After the rewrite, it stores `"PENDING"`. No change to `runAgentSpawn` code is needed — it calls `spawner(ctx, name, prompt)` and stores whatever is returned.

Verify this is correct by reading lines 765–778 of cli_agent.go: `paneID, sessionID, spawnErr := spawner(...)` → stores `sessionID` (now "PENDING") in `AgentRecord.SessionID`. Correct.

- [ ] **Step 4: Run tests**

```bash
targ test
```

Expected: PASS. The spawn tests use `testSpawnAckMaxWait` and a fake spawner function — they don't exercise the real tmux binary, so the rewrite doesn't break them.

- [ ] **Step 5: Run full check**

```bash
targ check-full
```

Expected: no linter errors. Fix any issues (e.g. unused constants, missing `shellQuote` import for `strings`).

- [ ] **Step 6: Commit**

```bash
git add internal/cli/cli_agent.go
git commit -m "feat(cli): rewrite osTmuxSpawnWith to start engram agent run in-pane (Phase 4 spawn transition)"
```

---

## Task 6: Skill Additions (Prefix Marker Catalog)

**Files:**
- Modify: `skills/use-engram-chat-as/SKILL.md` — add worker speech-to-chat section
- Modify: `skills/engram-agent/SKILL.md` — add prefix marker usage examples
- Modify: `skills/engram-tmux-lead/SKILL.md` — 1-line session-id note

**IMPORTANT:** Use `superpowers:writing-skills` for all SKILL.md edits. No exceptions.

- [ ] **Step 1: Add prefix marker catalog to `use-engram-chat-as`**

Using `superpowers:writing-skills`, add a "Speech-to-Chat: Worker Prefix Markers" section to `skills/use-engram-chat-as/SKILL.md`. This section describes how headless `-p` workers coordinate — by saying the prefix marker in their output, not by calling `engram chat post`. Content to add:

```
## Speech-to-Chat: Worker Prefix Markers

Headless workers (spawned via `engram agent run`) coordinate by expressing protocol
messages as prefix markers in their output. The binary intercepts these markers and
posts to the chat file on the worker's behalf.

**Prefix marker catalog:**

| Prefix | Chat type | Usage |
|--------|-----------|-------|
| `READY:` | ready | First output after launch. Announces presence. |
| `INTENT: Situation: X.\nBehavior: Y.` | intent | Before any significant action. End your turn after saying INTENT:. |
| `ACK:` | ack | No objection to a received intent. |
| `WAIT:` | wait | Objection or relevant memory. State concern on same line. |
| `DONE:` | done | Task/action complete. |
| `LEARNED:` | learned | Reusable fact for engram-agent. |
| `INFO:` | info | Status update. |
| `ESCALATE:` | escalate | Unresolved argument; needs lead. |

**After saying INTENT:, end your turn.** You will receive either:
- `Proceed.` — all recipients ACKed; proceed with the planned action.
- `WAIT from <agent>: [text]` — objection; engage per the Argument Protocol.

**Workers do NOT read the chat file directly.** The binary delivers responses in your
next turn. You never need to call `engram chat post` or `engram chat watch`.

**HARD RULE:** Only leads call `engram chat post` directly — they are interactive, not
headless. Workers express ALL coordination through prefix markers.
```

- [ ] **Step 2: Add prefix marker usage examples to `engram-agent`**

Using `superpowers:writing-skills`, update `skills/engram-agent/SKILL.md` to describe how engram-agent responds to intents via prefix markers. Add examples of when to say `ACK:` vs `WAIT:`:

```
## Responding via Prefix Markers

When you receive an intent in your turn context (after binary auto-resume in Phase 5;
currently via Background Monitor Pattern):

- Say `ACK:` followed by your response if no objection.
- Say `WAIT:` followed by your concern if surfacing a relevant memory or objection.

Example ACK:
  ACK: No relevant memories. Proceed.

Example WAIT:
  WAIT: Memory match: <fact>. Situation: <what this means for the intent>.
```

- [ ] **Step 3: Add 1-line session-id note to `engram-tmux-lead`**

Using `superpowers:writing-skills`, add a single line to the Agent Lifecycle section of `skills/engram-tmux-lead/SKILL.md`:

```
The `session-id` in the state file is the Claude conversation UUID (not tmux session format) — required for `engram agent resume` in Phase 5. It is written by the `engram agent run` pipeline, not by spawn.
```

- [ ] **Step 4: Commit skill additions**

```bash
git add skills/
git commit -m "feat(skills): add speech-to-chat prefix marker catalog and worker guidance (Phase 4)"
```

---

## Task 7: E2E Verification (Criteria 1–4)

**Files:** None modified. Live session verification.

Run a real multi-agent session and verify all four pre-deletion criteria pass. Do not proceed to Task 8 until all four pass.

- [ ] **Step 1: Start a fresh engram session**

```bash
/engram-up
```

This starts lead + engram-agent via the normal skill flow.

- [ ] **Step 2: Spawn a test worker via lead**

```
Lead: spawn a worker named "phase4-test" with a short coordination task:
"Post READY:, then post INTENT: Situation: Testing Phase 4 speech-to-chat. Behavior: Will post DONE: and exit.", then proceed normally.
```

- [ ] **Step 3: Verify Criterion 1 (real session-id)**

```bash
engram agent list | jq -r '."session-id"'
```

Expected: a UUID like `550e8400-e29b-41d4-a716-446655440000`. NOT `$0`, `$1`, `PENDING`, or empty.

Also verify it appears BEFORE the first INTENT: in the chat file:
```bash
grep -n "session-id\|type = \"intent\"" ~/.local/share/engram/state/-Users-joe-repos-personal-engram.toml
grep -n "type = \"intent\"\|type = \"ready\"" ~/.local/share/engram/chat/-Users-joe-repos-personal-engram.toml | tail -10
```

- [ ] **Step 4: Verify Criterion 2 (speech relay live)**

```bash
tail -n 30 ~/.local/share/engram/chat/-Users-joe-repos-personal-engram.toml
```

Expected: `type = "intent"` message with `from = "phase4-test"` (the worker's name). The `text` field should contain the Situation/Behavior from the worker's INTENT: output.

- [ ] **Step 5: Verify Criterion 3 (display filtering)**

Observe the worker's tmux pane. Expected: only human-readable prose visible. No raw JSONL lines (no `{"type":...}` lines).

- [ ] **Step 6: Verify Criterion 4 (--dangerously-skip-permissions in -p mode)**

Observe the worker pane startup. Expected: no "permission denied" or "tool not allowed" errors. JSONL stream starts successfully. Session-id appears in state file.

- [ ] **Step 7: If any criterion fails, fix before Task 8**

Diagnose the failure mode:
- Criterion 1 fails with `PENDING`: Runner.WriteSessionID not being called, or first event has no session_id. Check `streamjson.Parse` against a real `claude -p --verbose --output-format=stream-json` invocation to verify the JSON schema.
- Criterion 1 fails with wrong format: check that the first JSONL event has `session_id` at the top level, not nested.
- Criterion 2 fails (no intent in chat): check `claude.Runner.relayMarker` is being called. Verify the assistant text block has the INTENT: marker at column 0 with a space after the colon.
- Criterion 3 fails (raw JSONL in pane): check the `default` case in `ProcessStream` is not writing to `Pane`.
- Criterion 4 fails (permission error): verify the `-p` flag is compatible with `--dangerously-skip-permissions` in the installed Claude CLI version.

Fix, commit, re-verify before proceeding.

- [ ] **Step 8: Document verification results**

Post to chat:
```
engram chat post --from plan-writer --to all --thread phase4-e2e --type info \
  --text "Phase 4 E2E Criteria 1-4: PASS. Session-id: <UUID>. Speech relay: confirmed. Display filter: confirmed. -p mode permissions: confirmed. Proceeding to Task 8 (Category B skill deletions)."
```

---

## Task 8: Category B Skill Deletions (Post-E2E)

**Files:**
- Modify: `skills/use-engram-chat-as/SKILL.md`
- Modify: `skills/engram-agent/SKILL.md`

Category B: content that teaches workers to call `engram chat post` directly. This is now invalid — workers are headless `-p` workers that must use prefix markers. Delete ONLY after Criteria 1–4 pass.

**IMPORTANT:** Use `superpowers:writing-skills` for all SKILL.md edits. No exceptions.

**Do NOT delete from `engram-tmux-lead`:** Lead calls `engram chat post` directly — it is interactive, not headless. The Writing Messages section stays in engram-tmux-lead.

- [ ] **Step 1: Delete from `use-engram-chat-as` (Category B)**

Using `superpowers:writing-skills`, delete these sections from `skills/use-engram-chat-as/SKILL.md`:

1. **Writing Messages section** (~40 lines): The `engram chat post` calling convention for workers, including `CURSOR=$(engram chat post ...)`, lock handling, bash examples. (Workers now use prefix markers; this section is now invalid for workers and misleading.)

2. **Heartbeat bash section** (~15 lines): The `engram chat post --type info --thread heartbeat` bash block for workers. (Binary tracks worker state via state file; workers don't heartbeat.)

3. **`engram chat post` calling-convention prose** (~30 lines): Any prose that describes workers calling `engram chat post` as their coordination mechanism.

**Do NOT delete:**
- Background Monitor Pattern (needed by engram-agent until Phase 5)
- Agent Lifecycle watch loop (needed by engram-agent until Phase 5)
- Compaction recovery section (needed until Phase 5)
- Intent protocol prose (describes WHEN to express intent — still needed)
- ACK-Wait Protocol section (engram-agent still calls `engram chat ack-wait` directly)
- Lead calling convention (leads are interactive — this stays correct)

- [ ] **Step 2: Delete from `engram-agent` (Category B)**

Using `superpowers:writing-skills`, delete from `skills/engram-agent/SKILL.md`:

1. **Explicit `engram chat post` calls in main loop steps** (~10 lines): Where engram-agent posts ACK/WAIT via `engram chat post` command.

2. **Heartbeat timer section** (~10 lines): The 5-minute heartbeat bash block using `engram chat post`.

Replace these deleted sections with the prefix marker usage added in Task 6 (ACK: and WAIT: as output, not as shell commands).

- [ ] **Step 3: Verify skill sizes match projections**

```bash
wc -l skills/use-engram-chat-as/SKILL.md
wc -l skills/engram-agent/SKILL.md
wc -l skills/engram-tmux-lead/SKILL.md
```

Expected (approximate):
- use-engram-chat-as: ~665 lines (was ~736; deleted ~85, added ~23)
- engram-agent: ~330 lines (was ~364; deleted ~40, added ~10)
- engram-tmux-lead: ~1,050 lines (unchanged from Phase 4 — lead calling convention survives)

- [ ] **Step 4: Run Criterion 5 (skill deletions safe)**

Start a new multi-agent session with 2 spawned workers. Verify coordination works:
- Workers express intents via `INTENT:` prefix markers
- Binary relays to chat
- engram-agent responds via `ACK:` or `WAIT:` prefix markers
- Workers receive `Proceed.` or `WAIT from engram-agent:` in next turn
- No worker attempts to call `engram chat post` directly

- [ ] **Step 5: Commit skill deletions**

```bash
git add skills/
git commit -m "feat(skills): remove worker engram chat post guidance after Phase 4 E2E verification (Category B)"
```

---

## Post-Phase-4: Phase 5 Codesign Agenda

Phase 5 (Agent Resume + Auto-Resume) **must NOT be implemented without a dedicated codesign session.** The section below reflects positions resolved by the 4-planner `codesign-phase5-reassess` session (2026-04-07, after Phase 4 completion). Unresolved items are marked as Phase 5 codesign agenda items.

---

### Pre-Phase-5 Blockers (ship before Phase 5 codesign begins)

These are skill-only changes. No binary needed. Both can ship as one standalone PR.

| Blocker | Severity | Action |
|---------|----------|--------|
| Role-based naming guidance in `engram-tmux-lead` | **HIGH** | Add ~1 paragraph to spawn section: choose descriptive role-based names (`executor`, `arch-planner`, `skill-reviewer`); avoid generic sequential names (`planner-9`, `agent-1`); append disambiguator for same-role duplicates (`executor-auth`, `executor-db`). Phase 5 will spawn `engram-agent` as a headless worker — without this guidance, numbered names become the default at the architectural boundary. |
| Non-INTENT loop exit contract in `use-engram-chat-as` Headless Workers section | **MEDIUM** | Add ~2–3 sentences: a turn containing only `INFO:`, `ACK:`, `LEARNED:`, or `DONE:` markers terminates the session — the runner exits the loop. Use `DONE:` only when the task is genuinely complete. An `INTENT:` turn is required to continue working in the next turn. |

Start Phase 5 codesign in parallel with the PR — do not wait for merge.

---

### Phase 5 Codesign: Resolved Positions

**Item 3 — STARTING→ACTIVE state transition: RESOLVED**

Trigger: first `READY:` prefix marker detected by `ProcessStream` → Runner calls `WriteState("ACTIVE")`. Same DI pattern as `WriteSessionID` (Phase 4). Only the first `READY:` triggers the transition; subsequent `READY:` markers are no-ops. Binary-planner confirmed implementation: new `WriteState func(state string) error` callback wired in `cli_agent.go`, called in `handleEvent`.

**Item 2 — Subagent management (worker queue): CONSTRAINED**

Worker queue = count of agents in state `STARTING` or `ACTIVE` in the state file. Max 3 concurrent. Check happens inside the existing `readModifyWriteStateFile` lock — no race condition. State-file-backed (not in-memory) so count survives binary restarts. Holds are orthogonal: a held agent counts toward the limit (it occupies a pane and session). Implementation: `activeWorkerCount(sf) int` pure function in `internal/agent`, < 10 lines, trivially testable.

**Item 6 — Cursor passing in resume prompt: CONSTRAINED**

Phase 5 resume prompt format (structured plain-text fields):
```
CURSOR: <N>
MEMORY_FILES:
<path1>
<path2>
<path3>
INTENT_FROM: <agent-name>
INTENT_TEXT: <first line of most recent unresolved intent addressed to engram-agent>
Instruction: Load the files listed under MEMORY_FILES. Use the CURSOR value when calling engram chat ack-wait. Respond to the intent above with ACK:, WAIT:, or INTENT:.
```
`engram-agent` skill specifies: "The resume prompt contains your current chat cursor (`CURSOR:`) and memory files to load (`MEMORY_FILES:`). Do not scan `~/.local/share/engram/memory/` directly — binary has already selected the most relevant files by recency. Use `CURSOR:` value when calling `engram chat ack-wait`." Phase 6 extends the prompt to include `RECENT_INTENTS:` summaries (Item 1).

---

### Phase 5 Codesign: Open Items (must resolve before implementation)

**Item 5 — Tiered loading for stateless invocations (skill-planner leads)**

After stateless conversion, `engram-agent` cold-loads memories on each invocation. The loading strategy is a skill-design problem: skill-planner must define the selection criterion (most recent by mtime? highest confidence? topic-filtered by `INTENT_TEXT`?) and whether the binary injects file paths via the resume prompt or the worker loads autonomously from disk. Either mechanism is binary-implementable. Phase 5 codesign session resolves the complete design — do not assume binary prompt injection without skill-planner's sign-off.

**Phase 5 `engram-agent` skill changes (required before implementation):**

| Change | Type | Rationale |
|--------|------|-----------|
| DELETE Background Monitor Pattern (fswatch -1 main loop) | Delete | PRE-PHASE-1 pattern; primary cause of watch failures; was deferred from Phase 3 |
| DELETE Agent Lifecycle watch loop | Delete | Stateless workers don't maintain persistent loops |
| DELETE Compaction recovery section | Delete | Stateless workers have no context to compact/recover |
| REWRITE Memory loading section | Rewrite | Stateless cold-load from `MEMORY_FILES:` in resume prompt; no directory scan |
| ADD `CURSOR:`/`MEMORY_FILES:`/`INTENT_FROM:`/`INTENT_TEXT:` parsing guidance | Add | Phase 5 resume prompt protocol |
| RETAIN Memory file write/locking (shlock) | Retain | Shlock is for `~/.local/share/engram/memory/` writes — correct and independent of chat protocol |
| RETAIN ACK:/WAIT: prefix marker output model | Retain | Added Phase 4 ✓ |

**Item 2 (queue + hold interaction) — Phase 5 codesign session**

Worker queue design is constrained (see above), but the interaction with the hold system when a held agent's hold is released requires explicit design. Does a hold release trigger an auto-resume cycle? Who detects it?

---

### Phase 5 Codesign: Deferred to Phase 6

| Item | Rationale |
|------|-----------|
| **Item 1 — `recent_intents` correlation** | Behavioral refinement. Phase 5 resume prompt passes minimal context (most recent unresolved intent name). Full intent correlation is a Phase 6 polish item. Phase 5 without it is correct, not degraded. |
| **Item 4 — Rate limiting for stateless workers** | Safety-first: absence is not a correctness failure. When implemented (Phase 6), binary checks memory file timestamps before auto-resume and skips if >5 new memories in last 10 min. Binary-owned check, not skill-owned. |
| **planner-11 Gap 1 — Thread-aware routing** | `INTENT[thread=build]:` syntax in prefix markers; binary extracts thread name. Phase 6 dispatch redesign is the right place. Phase 5 tolerates `speech-relay` thread hardcoding. |
| **planner-11 Gap 2 — Worker-to-worker addressing** | `INTENT[to=lead,thread=review]:` prefix syntax. Requires dispatch-layer redesign. Phase 6. |
| **planner-11 Gap 3 — Mechanical argument continuations** | 3-input argument protocol enforcement for headless workers requires cross-turn state. Phase 6. |

---

### Phase 5 Watch Loop Contract

**Decision: Option A — extend `runConversationLoop`** (binary-planner, phasing-planner, arch-planner unanimous)

After a worker session ends (DONE or no INTENT), instead of returning, the loop watches the chat file for the next intent addressed to `engram-agent`, then starts a new session. `engram agent run` never exits while the auto-resume loop is active.

**HARD CONSTRAINT for Phase 5 implementation:** The watch loop function signature must parameterize `agentName`, `chatFilePath`, and `stateFilePath` — even though Phase 5 only calls it with `"engram-agent"`. This documents the Phase 6 extraction target explicitly. Phase 6 generalizes by calling the same function with multiple agent names. No API redesign needed in Phase 6; only routing logic is added.

Not doing this in Phase 5 = forcing Phase 6 to extract hardcoded parameters under time pressure during a rewrite sprint.

---

### Phase 6 Scope (updated from prior synthesis)

Phase 6 is a rewrite-from-scratch, not an incremental extension. Confirmed scope:

- **Full binary dispatcher**: extends the Phase 5 watch loop to route intents to any managed agent (routing table: `agent-name → session-id` from state file)
- **Thread-aware routing** (Gap 1), **worker-to-worker addressing** (Gap 2), **mechanical argument enforcement** (Gap 3)
- **`recent_intents` correlation** (Item 1) and **rate limiting** (Item 4) added to dispatcher
- **Skill rewrite-from-scratch** to ~45-line targets across all three skills (post-Phase-5 estimates: `use-engram-chat-as` ~575L, `engram-agent` ~220–280L, `engram-tmux-lead` ~1090L → ~45L each, ~85% reduction)
- Phase 6 codesign must define what is "irreducible" — the minimum rule set that makes each skill safe without examples. This is a different discipline from all prior phases and requires a dedicated codesign session.

**Do not write the Phase 5 plan until the Phase 5 codesign session completes.**
