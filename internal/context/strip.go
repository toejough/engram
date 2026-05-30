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
	stripped, _ := stripIndexed(lines)

	return stripped
}

// unexported constants.
const (
	assistantPrefix      = "ASSISTANT: "
	base64Placeholder    = "[base64 removed]"
	blockTypeText        = "text"
	blockTypeToolResult  = "tool_result"
	blockTypeToolUse     = "tool_use"
	commandArgsClose     = "</command-args>"
	commandArgsOpen      = "<command-args>"
	commandNameOpen      = "<command-name>"
	localCommandCaveat   = "<local-command-caveat>"
	localCommandStdout   = "<local-command-stdout>"
	maxContentBlockLen   = 2000
	minBase64Len         = 100
	roleAssistant        = "assistant"
	roleUser             = "user"
	skillBodyPrefix      = "Base directory for this skill:"
	systemReminderOpen   = "<system-reminder"
	truncatedPlaceholder = "[truncated]"
	userPrefix           = "USER: "
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

// cleanHarnessInjection detects and cleans harness-injected USER turn text.
// Returns (cleaned, drop): if drop is true, the turn should be omitted entirely.
// Three cases are handled:
//
//  1. Skill body injection: text starting with "Base directory for this skill:" → drop.
//  2. Local-command noise: text starting with "<local-command-stdout>" or
//     "<local-command-caveat>" → drop.
//  3. Slash-command block: text containing "<command-name>" → extract the inner
//     text of <command-args> if present; if absent, drop.
func cleanHarnessInjection(text string) (string, bool) {
	trimmed := strings.TrimSpace(text)

	if strings.HasPrefix(trimmed, skillBodyPrefix) {
		return "", true
	}

	if strings.HasPrefix(trimmed, localCommandStdout) ||
		strings.HasPrefix(trimmed, localCommandCaveat) {
		return "", true
	}

	if strings.Contains(trimmed, commandNameOpen) {
		start := strings.Index(trimmed, commandArgsOpen)
		if start == -1 {
			return "", true
		}

		start += len(commandArgsOpen)

		end := strings.Index(trimmed[start:], commandArgsClose)
		if end == -1 {
			return "", true
		}

		args := strings.TrimSpace(trimmed[start : start+end])
		if args == "" {
			return "", true
		}

		return args, false
	}

	return text, false
}

// cleanText returns the text after applying system-reminder and harness-injection
// filters. Returns ("", true) if the text should be dropped entirely.
func cleanText(text string) (string, bool) {
	if isSystemReminder(text) {
		return "", true
	}

	return cleanHarnessInjection(text)
}

// extractContentBlocks joins non-noise text blocks from a content block array.
func extractContentBlocks(blocks []contentBlock) string {
	texts := make([]string, 0, len(blocks))

	for _, block := range blocks {
		if block.Type != blockTypeText {
			continue
		}

		cleaned, drop := cleanText(block.Text)
		if drop {
			continue
		}

		trimmed := strings.TrimSpace(cleaned)
		if trimmed != "" {
			texts = append(texts, trimmed)
		}
	}

	return strings.Join(texts, " ")
}

// extractContentText extracts text from a content field.
// Content can be a plain string or an array of content blocks.
func extractContentText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	// Try string first.
	var str string

	if json.Unmarshal(raw, &str) == nil {
		cleaned, drop := cleanText(str)
		if drop {
			return ""
		}

		return cleaned
	}

	// Try array of content blocks.
	var blocks []contentBlock

	if json.Unmarshal(raw, &blocks) != nil {
		return ""
	}

	return extractContentBlocks(blocks)
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

	prefix := userPrefix
	if role == roleAssistant {
		prefix = assistantPrefix
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

// stripIndexed mirrors Strip but additionally returns the index of the
// input line that produced each output line.
func stripIndexed(lines []string) ([]string, []int) {
	result := make([]string, 0, len(lines))
	srcIdx := make([]int, 0, len(lines))

	for i, line := range lines {
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
		srcIdx = append(srcIdx, i)
	}

	return result, srcIdx
}

// truncateContent truncates lines exceeding maxContentBlockLen.
func truncateContent(line string) string {
	if len(line) <= maxContentBlockLen {
		return line
	}

	return line[:maxContentBlockLen] + truncatedPlaceholder
}
