//go:build network

package acceptance_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func Test855d7a1f_Gate_ABlockDeliveredWithinATurnIsHeld(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var chat *conversation

	step(t, "Given a real Claude Code session with the probe installed, which puts a block that begins "+
		"[engram memory] and names a note at the first tool call of the session", func() {
		machine.hasClaudeCode()
		machine.agentMayRun("echo")
		machine.hasProbe()
		chat = machine.startSession("a-project").converse()
	})

	var sent *turn

	step(t, "When the agent makes a tool call and then another tool call", func() {
		sent = chat.tell(gatePrompt)
	})

	step(t, "Then the transcript holds the block as an attachment of type hook_additional_context beginning "+
		"[engram memory], and the note is delivered once", func() {
		t.Logf("the record around the tool calls:\n%s", describeEvents(chat.session.hostEvents()))

		g.Expect(sent.toolCalls()).To(Equal(2), "the tool calls the agent made")
		g.Expect(chat.session.blocksNaming(probeNote)).To(Equal(1), "the blocks in the transcript that name the note")
	})
}

func Test855d7a1h_Gate_AContextPutAtAToolCallArrivesWithItsResult(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	var chat *conversation

	step(t, "Given a real Claude Code session with the probe installed, which puts a block that names a note at "+
		"the first tool call of the session", func() {
		machine.hasClaudeCode()
		machine.agentMayRun("echo")
		machine.hasProbe()
		chat = machine.startSession("a-project").converse()
	})

	step(t, "When the agent makes that tool call", func() {
		chat.tell(gateSingleCall)
	})

	step(t, "Then the order of the tool's result, the block and the agent's next message in the model's requests "+
		"is recorded, with the time the block arrived", func() {
		events := chat.session.hostEvents()
		t.Logf("the record around the tool call:\n%s", describeEvents(events))
		t.Logf("the probe reports: %q", chat.shown)

		order := chat.session.requestOrder()
		t.Logf("the items of the model's request that follows the tool call, in order: %v", order)

		g.Expect(order).NotTo(BeEmpty(), "the items of the model's request that follows the tool call")

		result := slices.Index(order, "tool result")
		block := slices.Index(order, "block")

		g.Expect(result).To(BeNumerically(">=", 0), "the tool's result is in the request")
		g.Expect(block).To(BeNumerically(">", result), "the block follows the tool's result in the request")

		resulted := firstEvent(events, 0, "tool result")
		blocked := firstEvent(events, resulted.row, "block")
		asked := firstEvent(events, blocked.row, "request")
		answered := firstEvent(events, asked.row, "reply")

		t.Logf("the tool's result at %s, the block %s later, the request %s after that, and the agent's next message "+
			"%s after the request", resulted.at.Format("15:04:05.000"), blocked.at.Sub(resulted.at),
			asked.at.Sub(blocked.at), answered.at.Sub(asked.at))

		g.Expect(resulted.row).To(BeNumerically(">", 0), "the record holds the tool's result")
		g.Expect(blocked.row).To(BeNumerically(">", resulted.row), "the record holds the block after the result")
		g.Expect(asked.row).To(BeNumerically(">", blocked.row), "the request follows the block")
		g.Expect(answered.row).To(BeNumerically(">", asked.row), "the agent's next message follows the request")
	})
}

// unexported constants.
const (
	// gatePrompt makes the agent make two tool calls, one after the other.
	gatePrompt = "Run the shell command `echo one` with the Bash tool and wait for its result, then run the " +
		"shell command `echo two` with the Bash tool, then say done."
	// gateSingleCall makes the agent make one tool call.
	gateSingleCall = "Run the shell command `echo one` with the Bash tool, then say done."
	// probeNote is the note the block of the probe names.
	probeNote = "probe-note.md"
)

// hostEvent is one row of the session's record that shows what the host did around a tool call.
type hostEvent struct {
	row    int // the index of the row in the record
	at     time.Time
	kind   string // tool call, tool result, block, request or reply
	detail string
}

// hasProbe installs the probe of the gates in the place of the mod (855b46).
func (m *machine) hasProbe() {
	m.t.Helper()

	root, err := moduleRoot()
	if err != nil {
		m.t.Fatalf("finding the module: %v", err)
	}

	m.installMod(filepath.Join(root, "dev", "acceptance", "testdata", "probe"))
}

// hostEvents reads, from the session's record, the type, the order and the time of the rows around a tool call: the
// calls the agent made, their results, the blocks a hook added, the requests to the model and the agent's replies.
func (s *session) hostEvents() []hostEvent {
	var events []hostEvent

	for index, row := range s.rows() {
		kind, detail := row.eventKind()
		if kind == "" {
			continue
		}

		at, _ := time.Parse(time.RFC3339Nano, row.Timestamp)
		events = append(events, hostEvent{row: index, at: at, kind: kind, detail: detail})
	}

	return events
}

// requestOrder returns the kinds of the items, in order, of the model's request that follows the first tool result.
// The record holds a request as the messages it keeps of an earlier request and the hashes of the messages it adds. A
// hash names a row of its own for a message that no other row holds, and the row of the agent's or the tool's message
// for the rest, so the request is put together from both.
func (s *session) requestOrder() []string {
	messages := map[string]transcriptRow{}
	requests := map[string][]string{}
	resultAt := -1

	var follows string

	for index, row := range s.rows() {
		kind, _ := row.eventKind()

		switch {
		case row.Type == "api-request-blob":
			messages[row.Hash] = row
		case row.Type == "api-request":
			kept := requests[row.Base]
			requests[row.ID] = slices.Concat(kept[:min(row.Keep, len(kept))], row.Tail)

			if resultAt >= 0 && follows == "" {
				follows = row.ID
			}
		case row.Message.Role != "":
			messages[messageHash(row.Message.Role, row.Message.Content)] = row

			if kind == "tool result" && resultAt < 0 {
				resultAt = index
			}
		}
	}

	var order []string

	for _, hash := range requests[follows] {
		message, found := messages[hash]
		if !found {
			order = append(order, "unknown")

			continue
		}

		order = append(order, message.itemKinds()...)
	}

	return order
}

// eventKind tells what a row shows the host did, or "" when the row shows nothing around a tool call.
func (r transcriptRow) eventKind() (string, string) {
	var blocks []struct {
		Type string `json:"type"`
		Name string `json:"name"`
		Text string `json:"text"`
	}

	switch {
	case r.Type == "api-request":
		return "request", ""
	case r.Type == "attachment" && r.Attachment.Type == "hook_additional_context":
		return "block", r.Attachment.HookName + " " + r.Attachment.HookEvent
	case r.Type == "assistant" && json.Unmarshal(r.Message.Content, &blocks) == nil:
		for _, block := range blocks {
			if block.Type == "tool_use" {
				return "tool call", block.Name
			}
		}

		return "reply", ""
	case r.Type == "user" && json.Unmarshal(r.Message.Content, &blocks) == nil:
		for _, block := range blocks {
			if block.Type == "tool_result" {
				return "tool result", ""
			}
		}
	}

	return "", ""
}

// itemKinds returns the kind of each item of the message a row holds: a tool call, a tool result, a block that begins
// [engram memory], or text.
func (r transcriptRow) itemKinds() []string {
	var text string

	if json.Unmarshal(r.Message.Content, &text) == nil {
		return []string{"text"}
	}

	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}

	if json.Unmarshal(r.Message.Content, &blocks) != nil {
		return nil
	}

	kinds := make([]string, 0, len(blocks))

	for _, block := range blocks {
		switch {
		case block.Type == "tool_use":
			kinds = append(kinds, "tool call")
		case block.Type == "tool_result":
			kinds = append(kinds, "tool result")
		case strings.Contains(block.Text, engramMarker):
			kinds = append(kinds, "block")
		default:
			kinds = append(kinds, "text")
		}
	}

	return kinds
}

// toolCalls counts the tool calls the agent made in the turn.
func (t *turn) toolCalls() int {
	count := 0

	for _, row := range t.rows {
		if kind, _ := row.eventKind(); kind == "tool call" {
			count++
		}
	}

	return count
}

// describeEvents writes the events one to a line: the row, the time and the kind.
func describeEvents(events []hostEvent) string {
	lines := make([]string, 0, len(events))

	for _, event := range events {
		lines = append(lines, fmt.Sprintf("row %d  %s  %s %s", event.row, event.at.Format("15:04:05.000"), event.kind,
			event.detail))
	}

	return strings.Join(lines, "\n")
}

// firstEvent returns the first event of a kind in a row after one, or the zero event.
func firstEvent(events []hostEvent, after int, kind string) hostEvent {
	for _, event := range events {
		if event.row > after && event.kind == kind {
			return event
		}
	}

	return hostEvent{}
}

// messageHash is the hash that a request names a message by: the SHA-256 of the message's role and content as compact
// JSON with its keys in order.
func messageHash(role string, content json.RawMessage) string {
	var value any

	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.UseNumber()

	if decoder.Decode(&value) != nil {
		return ""
	}

	var encoded bytes.Buffer

	encoder := json.NewEncoder(&encoded)
	encoder.SetEscapeHTML(false)

	if encoder.Encode(map[string]any{"role": role, "content": value}) != nil {
		return ""
	}

	sum := sha256.Sum256(bytes.TrimSuffix(encoded.Bytes(), []byte("\n")))

	return hex.EncodeToString(sum[:])
}
