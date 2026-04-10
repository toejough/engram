# Issue #538: Conversation Relay Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Forward all agent prose to chat as `type = "conversation"` so every word an agent says is visible to all other agents, not just explicitly-marked coordination signals.

**Architecture:** `streamjson.NonMarkerText` extracts prose that precedes the first speech marker in a turn. `claude.Runner.handleEvent` calls it after processing markers and posts non-empty prose via a new `relayConversation` helper. Pure-marker turns produce no conversation message; pure-prose turns produce one conversation message; mixed turns produce both. Skills are updated via `superpowers:writing-skills`.

**Tech Stack:** Go, `internal/streamjson`, `internal/claude`, `internal/chat`, `skills/use-engram-chat-as/SKILL.md`, `skills/engram-agent/SKILL.md`

---

### File Map

| File | Change |
|------|--------|
| `internal/streamjson/streamjson.go` | Add `NonMarkerText(text string) string` |
| `internal/streamjson/streamjson_test.go` | Tests for `NonMarkerText` |
| `internal/claude/claude.go` | Add `relayConversation`; call from `handleEvent` |
| `internal/claude/claude_test.go` | Tests for conversation relay behaviour |
| `skills/use-engram-chat-as/SKILL.md` | Add `conversation` to message type catalog |
| `skills/engram-agent/SKILL.md` | Document watching `conversation` messages |

---

### Task 1: `NonMarkerText` in streamjson

**Files:**
- Modify: `internal/streamjson/streamjson.go`
- Modify: `internal/streamjson/streamjson_test.go`

- [ ] **Step 1: Write failing tests**

Add to `internal/streamjson/streamjson_test.go`:

```go
func TestNonMarkerText_EmptyInput_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	g.Expect(NonMarkerText("")).To(Equal(""))
}

func TestNonMarkerText_PureProse_ReturnsAll(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	text := "I am confused about what to do next.\nMaybe try option B."
	g.Expect(NonMarkerText(text)).To(Equal(text))
}

func TestNonMarkerText_StartsWithMarker_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	g.Expect(NonMarkerText("INTENT: Situation: X.")).To(Equal(""))
}

func TestNonMarkerText_ProseBeforeMarker_ReturnsProse(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	text := "Here is my thinking.\nINTENT: Situation: X."
	g.Expect(NonMarkerText(text)).To(Equal("Here is my thinking."))
}

func TestNonMarkerText_WhitespaceOnly_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	g.Expect(NonMarkerText("   \n  ")).To(Equal(""))
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
targ test
```

Expected: FAIL — `NonMarkerText` undefined.

- [ ] **Step 3: Implement `NonMarkerText`**

Add to `internal/streamjson/streamjson.go` after `DetectSpeechMarkers`:

```go
// NonMarkerText returns the text from an assistant turn that precedes the first
// speech marker. If there are no markers, the entire trimmed text is returned.
// If the turn begins with a marker, empty string is returned.
func NonMarkerText(text string) string {
	lines := strings.Split(text, "\n")

	var proseLines []string

	for _, line := range lines {
		_, _, found := detectPrefix(line)
		if found {
			break
		}

		proseLines = append(proseLines, line)
	}

	return strings.TrimSpace(strings.Join(proseLines, "\n"))
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
targ test
```

Expected: PASS for all five new tests.

- [ ] **Step 5: Commit**

```bash
git add internal/streamjson/streamjson.go internal/streamjson/streamjson_test.go
git commit -m "feat(streamjson): add NonMarkerText to extract pre-marker prose"
```

---

### Task 2: Conversation relay in `claude.Runner`

**Files:**
- Modify: `internal/claude/claude.go`
- Modify: `internal/claude/claude_test.go`

- [ ] **Step 1: Write failing tests**

Add to `internal/claude/claude_test.go`:

```go
func TestProcessStream_PureProse_PostedAsConversation(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	poster := &mockPoster{}

	runner := claude.Runner{
		AgentName:      "test-agent",
		Pane:           io.Discard,
		Poster:         poster,
		WriteSessionID: func(string) error { return nil },
	}

	assistantJSON := `{"type":"assistant","session_id":"abc",` +
		`"message":{"content":[{"type":"text","text":"I am confused about what to do."}]}}`
	stream := strings.NewReader(assistantJSON + "\n")

	_, err := runner.ProcessStream(stream)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	g.Expect(poster.posted).To(HaveLen(1))
	g.Expect(poster.posted[0].Type).To(Equal("conversation"))
	g.Expect(poster.posted[0].To).To(Equal("all"))
	g.Expect(poster.posted[0].Text).To(Equal("I am confused about what to do."))
}

func TestProcessStream_ProseBeforeMarker_PostedAsConversationAndMarker(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	poster := &mockPoster{}

	runner := claude.Runner{
		AgentName:      "test-agent",
		Pane:           io.Discard,
		Poster:         poster,
		WriteSessionID: func(string) error { return nil },
	}

	text := "I will proceed now.\\nINTENT: Situation: X. Behavior: Y."
	assistantJSON := `{"type":"assistant","session_id":"abc",` +
		`"message":{"content":[{"type":"text","text":"I will proceed now.\nINTENT: Situation: X. Behavior: Y."}]}}`
	stream := strings.NewReader(assistantJSON + "\n")
	_ = text

	_, err := runner.ProcessStream(stream)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	types := make([]string, 0, len(poster.posted))
	for _, msg := range poster.posted {
		types = append(types, msg.Type)
	}

	g.Expect(types).To(ContainElements("conversation", "intent"))
	var convMsg chat.Message
	for _, msg := range poster.posted {
		if msg.Type == "conversation" {
			convMsg = msg
		}
	}
	g.Expect(convMsg.Text).To(Equal("I will proceed now."))
	g.Expect(convMsg.To).To(Equal("all"))
}

func TestProcessStream_PureMarker_NoConversationPosted(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	poster := &mockPoster{}

	runner := claude.Runner{
		AgentName:      "test-agent",
		Pane:           io.Discard,
		Poster:         poster,
		WriteSessionID: func(string) error { return nil },
	}

	intentJSON := `{"type":"assistant","session_id":"abc",` +
		`"message":{"content":[{"type":"text","text":"INTENT: Situation: X. Behavior: Y."}]}}`
	stream := strings.NewReader(intentJSON + "\n")

	_, err := runner.ProcessStream(stream)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	for _, msg := range poster.posted {
		g.Expect(msg.Type).NotTo(Equal("conversation"))
	}
}

func TestProcessStream_ConversationRelayError_WarnsOnPane(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var pane bytes.Buffer

	runner := claude.Runner{
		AgentName:      "test-agent",
		Pane:           &pane,
		Poster:         &errorPoster{err: errors.New("disk full")},
		WriteSessionID: func(string) error { return nil },
	}

	assistantJSON := `{"type":"assistant","session_id":"abc",` +
		`"message":{"content":[{"type":"text","text":"Just some prose."}]}}`
	stream := strings.NewReader(assistantJSON + "\n")

	_, err := runner.ProcessStream(stream)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	g.Expect(pane.String()).To(ContainSubstring("conversation relay failed"))
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
targ test
```

Expected: FAIL — `conversation` type not yet posted.

- [ ] **Step 3: Add `relayConversation` and update `handleEvent`**

In `internal/claude/claude.go`, update `handleEvent`:

```go
func (r *Runner) handleEvent(event streamjson.Event, result *StreamResult) {
	switch event.Type {
	case "assistant":
		if event.Text != "" {
			_, _ = fmt.Fprintf(r.Pane, "%s\n", event.Text)
		}

		for _, marker := range streamjson.DetectSpeechMarkers(event.Text) {
			r.handleMarker(marker, result)
		}

		if prose := streamjson.NonMarkerText(event.Text); prose != "" {
			r.relayConversation(prose)
		}
	case "user":
		if event.Text != "" {
			_, _ = fmt.Fprintf(r.Pane, "%s\n", event.Text)
		}
	default:
		// system, tool_use, result, error: display-filtered (not written to pane).
	}
}
```

Add `relayConversation` after `relayMarker`:

```go
// relayConversation posts non-marker prose from an agent turn to chat as type "conversation".
func (r *Runner) relayConversation(text string) {
	msg := chat.Message{
		From:   r.AgentName,
		To:     "all",
		Thread: "speech-relay",
		Type:   "conversation",
		Text:   text,
	}

	_, err := r.Poster.Post(msg)
	if err != nil {
		_, _ = fmt.Fprintf(r.Pane, "[engram] warning: conversation relay failed: %v\n", err)
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
targ test
```

Expected: all new tests PASS. Existing tests unaffected (marker-only turns still work; display filter test unaffected because it uses `io.Discard` for poster inspection).

- [ ] **Step 5: Run full checks**

```bash
targ check-full
```

Expected: 8/8 PASS. Fix any lint issues before committing.

- [ ] **Step 6: Commit**

```bash
git add internal/claude/claude.go internal/claude/claude_test.go
git commit -m "feat(claude): relay non-marker prose to chat as conversation type

Fixes #538. All agent prose that precedes speech markers is now posted
to chat as type='conversation' addressed to 'all'. Pure-marker turns
produce no conversation message. Agents that go off-script are now
visible to every other agent in the room."
```

---

### Task 3: Update skills

**Files:**
- Modify: `skills/use-engram-chat-as/SKILL.md`
- Modify: `skills/engram-agent/SKILL.md`

**Use `superpowers:writing-skills` for both edits.** Do not edit SKILL.md files directly.

- [ ] **Step 1: Update `use-engram-chat-as` message type catalog**

Add `conversation` row to the Message Type Catalog table (around line 150):

```
| `conversation` | Non-marker prose from a headless worker turn — the agent's natural output when no explicit marker was emitted. Addressed to `all`. All agents should watch and interpret this. | Binary (auto-posted from agent stdout) | No |
```

Also add to the Common Mistakes table:

```
| Ignore `conversation` messages | All agents should watch `conversation` messages, not just typed coordination signals. Markers are clearer signal, not the only signal. |
```

- [ ] **Step 2: Update `engram-agent` skill**

In the section describing what messages engram-agent watches for and reacts to, add:

```
- **`conversation` messages**: watch and interpret — agent may be reasoning aloud, expressing confusion, or describing a problem without a formal marker. Surface relevant memories if the prose matches a known situation.
```

- [ ] **Step 3: Verify skill edits with `superpowers:writing-skills`**

The `writing-skills` skill runs baseline → edit → pressure test. Follow its protocol fully before marking this task done.

---

### Self-Review

**Spec coverage:**
- ✓ Non-marker prose → `conversation` type (Task 2)
- ✓ Markers → their specific type, no double-posting (Task 2 test: `TestProcessStream_PureMarker_NoConversationPosted`)
- ✓ All agents addressed via `to = "all"` (Task 2 `relayConversation`)
- ✓ One message per turn (Task 1 `NonMarkerText` returns prose as one string, Task 2 posts it once)
- ✓ Pane display unchanged (Task 2: `handleEvent` still writes to `r.Pane` before processing)
- ✓ Skill docs updated (Task 3)

**Placeholder scan:** None found. All test code is concrete, all implementation is shown.

**Type consistency:** `chat.Message`, `streamjson.NonMarkerText`, `streamjson.DetectSpeechMarkers` — all consistent with existing usage in `claude.go`.
