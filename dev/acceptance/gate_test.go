//go:build network

package acceptance_test

import (
	"encoding/json"
	"fmt"
	"path/filepath"
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

// unexported constants.
const (
	// gatePrompt makes the agent make two tool calls, one after the other.
	gatePrompt = "Run the shell command `echo one` with the Bash tool and wait for its result, then run the " +
		"shell command `echo two` with the Bash tool, then say done."
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
