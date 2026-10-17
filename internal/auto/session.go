package auto

import (
	"bytes"
	"encoding/json"
	"strings"
)

// HeldText returns the text of what a session's agent holds of the vault, from the session's record (793d4d): the file
// name on each note line of the blocks delivered beside prompts, which the record keeps as `hook_additional_context`
// attachments, and the command and the result of each lookup the agent ran itself, a Bash command that runs `engram
// query` or `engram show`, whose result is the `tool_result` block that carries the ID of the command's `tool_use`.
func HeldText(record []byte) string {
	held := holdings{lookups: map[string]string{}}

	for line := range bytes.SplitSeq(record, []byte("\n")) {
		var row recordRow

		if json.Unmarshal(line, &row) == nil {
			held.read(row)
		}
	}

	return held.text.String()
}

// unexported constants.
const (
	// deliveredType is the type of the attachment that a session's record keeps for the context a hook put beside a
	// prompt.
	deliveredType = "hook_additional_context"
)

// contentBlock is a block of the content of a message: a tool the agent called, or the result of one.
type contentBlock struct {
	Type      string          `json:"type"`
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Input     json.RawMessage `json:"input"`
	ToolUseID string          `json:"tool_use_id"` //nolint:tagliatelle // Claude Code's record names it so
	Content   json.RawMessage `json:"content"`
}

// holdings is what has been read of a record so far: the text that the session holds, and the lookups the agent has
// started, by the ID of the tool call, until their results arrive.
type holdings struct {
	text    strings.Builder
	lookups map[string]string
}

// read adds what one row of the record gives the agent.
func (h *holdings) read(row recordRow) {
	switch row.Type {
	case "attachment":
		h.readDelivered(row)
	case "assistant":
		h.readLookups(row)
	case "user":
		h.readResults(row)
	}
}

// readDelivered adds the notes of the blocks delivered beside a prompt.
func (h *holdings) readDelivered(row recordRow) {
	if row.Attachment.Type != deliveredType {
		return
	}

	for _, entry := range entriesOf(row.Attachment.Content) {
		h.text.WriteString(noteNamesOf(entry))
	}
}

// readLookups notes the lookups the agent starts.
func (h *holdings) readLookups(row recordRow) {
	for _, block := range blocksOf(row.Message.Content) {
		if command := lookupOf(block); command != "" {
			h.lookups[block.ID] = command
		}
	}
}

// readResults adds the lookups whose results arrive, each with its command.
func (h *holdings) readResults(row recordRow) {
	for _, block := range blocksOf(row.Message.Content) {
		command, isLookup := h.lookups[block.ToolUseID]
		if block.Type == "tool_result" && isLookup {
			h.text.WriteString(command + "\n" + textOf(block.Content) + "\n")
		}
	}
}

// recordRow is one line of a session's record, with the fields that tell what the agent was given.
type recordRow struct {
	Type    string `json:"type"`
	Message struct {
		Content json.RawMessage `json:"content"`
	} `json:"message"`
	Attachment struct {
		Type    string          `json:"type"`
		Content json.RawMessage `json:"content"`
	} `json:"attachment"`
}

// blocksOf reads the blocks of a message's content; the content of a prompt, a string, has none.
func blocksOf(content json.RawMessage) []contentBlock {
	var blocks []contentBlock

	if json.Unmarshal(content, &blocks) != nil {
		return nil
	}

	return blocks
}

// entriesOf reads the entries of the context a hook put beside a prompt: a string, or a list of strings.
func entriesOf(raw json.RawMessage) []string {
	var entry string

	if json.Unmarshal(raw, &entry) == nil {
		return []string{entry}
	}

	var entries []string

	if json.Unmarshal(raw, &entries) != nil {
		return nil
	}

	return entries
}

// lookupOf returns the command of a block that is the agent running a lookup of the vault, a Bash command whose text
// runs `engram query` or `engram show`, or nothing for any other block.
func lookupOf(block contentBlock) string {
	if block.Type != "tool_use" || block.Name != "Bash" {
		return ""
	}

	var input struct {
		Command string `json:"command"`
	}

	if json.Unmarshal(block.Input, &input) != nil {
		return ""
	}

	if !strings.Contains(input.Command, "engram query") && !strings.Contains(input.Command, "engram show") {
		return ""
	}

	return input.Command
}

// noteNamesOf returns the file names on the note lines of a block that automatic memory delivered, one to a line, or
// nothing for any other entry. A note line begins with a dash and a space, and its file name runs up to the space
// before the situation's parenthesis.
func noteNamesOf(entry string) string {
	if !strings.HasPrefix(entry, marker) {
		return ""
	}

	var names strings.Builder

	for line := range strings.SplitSeq(entry, "\n") {
		rest, isNote := strings.CutPrefix(line, "- ")
		if !isNote {
			continue
		}

		name, _, _ := strings.Cut(rest, " (")
		names.WriteString(name + "\n")
	}

	return names.String()
}

// textOf reads the text of the content of a tool's result: a string, or a list of blocks that have a text.
func textOf(raw json.RawMessage) string {
	var text string

	if json.Unmarshal(raw, &text) == nil {
		return text
	}

	var blocks []struct {
		Text string `json:"text"`
	}

	if json.Unmarshal(raw, &blocks) != nil {
		return ""
	}

	parts := make([]string, 0, len(blocks))

	for _, block := range blocks {
		parts = append(parts, block.Text)
	}

	return strings.Join(parts, "\n")
}
