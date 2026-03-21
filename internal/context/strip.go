package context

import (
	"regexp"
	"strings"
)

// Strip filters and cleans JSONL transcript lines:
//   - keeps only user and assistant entries (by "type" field)
//   - replaces base64 strings >100 chars with placeholder
//   - truncates lines >2000 chars
func Strip(lines []string) []string {
	result := make([]string, 0, len(lines))

	for _, line := range lines {
		if !isKeptType(line) {
			continue
		}

		cleaned := replaceBase64(line)
		cleaned = truncateContent(cleaned)

		result = append(result, cleaned)
	}

	return result
}

// unexported constants.
const (
	base64Placeholder    = "[base64 removed]"
	maxContentBlockLen   = 2000
	minBase64Len         = 100
	truncatedPlaceholder = "[truncated]"
)

// unexported variables.
var (
	base64Pattern = regexp.MustCompile(`[A-Za-z0-9+/=]{` + "100" + `,}`)
)

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
		strings.Contains(line, `"role": "assistant"`) ||
		strings.Contains(line, `"role":"toolUse"`) ||
		strings.Contains(line, `"role": "toolUse"`)
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
