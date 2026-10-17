package auto

import (
	"bytes"
	"encoding/json"
	"strings"
)

// HeldText returns the text of what a session's agent holds of the vault, from the session's record (793d4d): every
// block that was delivered beside a prompt, which the record keeps as a `hook_additional_context` attachment, and
// every result of a tool the agent ran, which it keeps as a `tool_result` block of a `user` record.
func HeldText(record []byte) string {
	var held strings.Builder

	for line := range bytes.SplitSeq(record, []byte("\n")) {
		var row struct {
			Type    string `json:"type"`
			Message struct {
				Content json.RawMessage `json:"content"`
			} `json:"message"`
			Attachment struct {
				Type    string          `json:"type"`
				Content json.RawMessage `json:"content"`
			} `json:"attachment"`
		}

		if json.Unmarshal(line, &row) != nil {
			continue
		}

		switch {
		case row.Type == "attachment" && row.Attachment.Type == deliveredType:
			held.WriteString(textOf(row.Attachment.Content) + "\n")
		case row.Type == "user":
			held.WriteString(toolResultsOf(row.Message.Content))
		}
	}

	return held.String()
}

// unexported constants.
const (
	// deliveredType is the type of the attachment that a session's record keeps for the context a hook put beside a
	// prompt.
	deliveredType = "hook_additional_context"
)

// textOf reads the text of a JSON value that is a string, or a list of strings, or a list of blocks that have a text.
func textOf(raw json.RawMessage) string {
	var text string

	if json.Unmarshal(raw, &text) == nil {
		return text
	}

	var texts []string

	if json.Unmarshal(raw, &texts) == nil {
		return strings.Join(texts, "\n")
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

// toolResultsOf returns the text of the results of tools in the content of a user record, one to a line; a prompt,
// whose content is a string, has none.
func toolResultsOf(content json.RawMessage) string {
	var blocks []struct {
		Type    string          `json:"type"`
		Content json.RawMessage `json:"content"`
	}

	if json.Unmarshal(content, &blocks) != nil {
		return ""
	}

	var results strings.Builder

	for _, block := range blocks {
		if block.Type == "tool_result" {
			results.WriteString(textOf(block.Content) + "\n")
		}
	}

	return results.String()
}
