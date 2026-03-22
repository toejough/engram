package context

import (
	"encoding/json"
	"regexp"
	"strings"
)

// Strip parses JSONL transcript lines and extracts clean conversation text.
// It returns "USER: ..." and "ASSISTANT: ..." lines, dropping:
//   - non-user/assistant entries (progress, system, etc.)
//   - tool_use and tool_result content blocks
//   - system-reminder text blocks
//   - base64 strings >100 chars (replaced with placeholder)
//   - lines >2000 chars (truncated)
func Strip(lines []string) []string {
	result := make([]string, 0, len(lines))

	for _, line := range lines {
		if !isKeptType(line) {
			continue
		}

		cleaned := replaceBase64(line)

		extracted := extractText(cleaned)
		if extracted == "" {
			continue
		}

		extracted = truncateContent(extracted)

		result = append(result, extracted)
	}

	return result
}

// unexported constants.
const (
	base64Placeholder    = "[base64 removed]"
	maxContentBlockLen   = 2000
	minBase64Len         = 100
	roleAssistant        = "assistant"
	roleUser             = "user"
	systemReminderOpen   = "<system-reminder"
	truncatedPlaceholder = "[truncated]"
)

// unexported variables.
var (
	base64Pattern = regexp.MustCompile(`[A-Za-z0-9+/=]{` + "100" + `,}`)
)

// contentBlock represents a content block in a message.
type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type jsonMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

// jsonlLine is a minimal representation of a JSONL transcript line.
type jsonlLine struct {
	Type    string      `json:"type"`
	Message jsonMessage `json:"message"`
}

// extractContentText extracts text from a content field.
// Content can be a plain string or an array of content blocks.
func extractContentText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	// Try string first.
	var str string

	strErr := json.Unmarshal(raw, &str)
	if strErr == nil {
		if isSystemReminder(str) {
			return ""
		}

		return str
	}

	// Try array of content blocks.
	var blocks []contentBlock

	blocksErr := json.Unmarshal(raw, &blocks)
	if blocksErr != nil {
		return ""
	}

	texts := make([]string, 0, len(blocks))

	for _, block := range blocks {
		if block.Type != "text" {
			continue
		}

		if isSystemReminder(block.Text) {
			continue
		}

		trimmed := strings.TrimSpace(block.Text)
		if trimmed != "" {
			texts = append(texts, trimmed)
		}
	}

	return strings.Join(texts, " ")
}

// extractText parses a JSONL line and returns clean "ROLE: text" output.
// Returns empty string if no useful text is found.
func extractText(line string) string {
	var entry jsonlLine

	err := json.Unmarshal([]byte(line), &entry)
	if err != nil {
		return ""
	}

	role := normalizeRole(entry)
	if role == "" {
		return ""
	}

	prefix := "USER: "
	if role == roleAssistant {
		prefix = "ASSISTANT: "
	}

	text := extractContentText(entry.Message.Content)
	if text == "" {
		return ""
	}

	return prefix + text
}

// isKeptType returns true if the line is a user or assistant entry.
// Checks both "type" (actual JSONL format) and "role" (legacy format).
func isKeptType(line string) bool {
	return strings.Contains(line, `"type":"user"`) ||
		strings.Contains(line, `"type": "user"`) ||
		strings.Contains(line, `"type":"assistant"`) ||
		strings.Contains(line, `"type": "assistant"`) ||
		strings.Contains(line, `"role":"user"`) ||
		strings.Contains(line, `"role": "user"`) ||
		strings.Contains(line, `"role":"assistant"`) ||
		strings.Contains(line, `"role": "assistant"`)
}

// isSystemReminder returns true if the text starts with a system-reminder tag.
func isSystemReminder(text string) bool {
	trimmed := strings.TrimSpace(text)
	return strings.HasPrefix(trimmed, systemReminderOpen)
}

// normalizeRole returns "user" or "assistant" from the entry, or empty string.
func normalizeRole(entry jsonlLine) string {
	// Check outer type field first (actual JSONL format).
	switch entry.Type {
	case roleUser:
		return roleUser
	case roleAssistant:
		return roleAssistant
	}

	// Fall back to message role (legacy format).
	switch entry.Message.Role {
	case roleUser:
		return roleUser
	case roleAssistant:
		return roleAssistant
	}

	return ""
}

// replaceBase64 replaces long base64-encoded strings with a placeholder.
func replaceBase64(line string) string {
	return base64Pattern.ReplaceAllString(line, base64Placeholder)
}

// truncateContent truncates lines exceeding maxContentBlockLen.
func truncateContent(line string) string {
	if len(line) <= maxContentBlockLen {
		return line
	}

	return line[:maxContentBlockLen] + truncatedPlaceholder
}
