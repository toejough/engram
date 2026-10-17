package auto

import (
	"bytes"
	"encoding/json"
	"strings"
)

// HeldText returns the text of what a session's agent holds of the vault, from the session's record (793d4d): every
// block that was delivered beside a prompt, which the record keeps as a `hook_additional_context` attachment.
func HeldText(record []byte) string {
	var held strings.Builder

	for line := range bytes.SplitSeq(record, []byte("\n")) {
		var row struct {
			Type       string `json:"type"`
			Attachment struct {
				Type    string          `json:"type"`
				Content json.RawMessage `json:"content"`
			} `json:"attachment"`
		}

		if json.Unmarshal(line, &row) != nil || row.Type != "attachment" || row.Attachment.Type != deliveredType {
			continue
		}

		var blocks []string

		if json.Unmarshal(row.Attachment.Content, &blocks) == nil {
			held.WriteString(strings.Join(blocks, "\n") + "\n")
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
