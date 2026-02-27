package memory

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// StripSession reads a JSONL session transcript and returns stripped text
// containing only learning-relevant content.
// If startOffset > 0, seeks to that byte position before reading (for incremental extraction).
// Returns the stripped text and the byte offset after the last line read.
func StripSession(path string, startOffset int64) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, fmt.Errorf("open session: %w", err)
	}

	defer func() { _ = f.Close() }()

	if startOffset > 0 {
		if _, err := f.Seek(startOffset, 0); err != nil {
			return "", 0, fmt.Errorf("seek to offset %d: %w", startOffset, err)
		}
	}

	bytesRead := startOffset

	var lines []string

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024) // 10MB line buffer

	for scanner.Scan() {
		bytesRead += int64(len(scanner.Bytes())) + 1 // +1 for newline

		var msg map[string]any

		err := json.Unmarshal(scanner.Bytes(), &msg)
		if err != nil {
			continue
		}

		msgType, _ := msg["type"].(string)
		if msgType != "user" && msgType != "assistant" {
			continue
		}

		message, ok := msg["message"].(map[string]any)
		if !ok {
			continue
		}

		role, _ := message["role"].(string)

		contentArr, ok := message["content"].([]any)
		if !ok {
			continue
		}

		for _, item := range contentArr {
			block, ok := item.(map[string]any)
			if !ok {
				continue
			}

			blockType, _ := block["type"].(string)

			switch blockType {
			case "text":
				text, _ := block["text"].(string)

				text = stripNoise(text)
				if text != "" {
					lines = append(lines, fmt.Sprintf("[%s] %s", role, text))
				}

			case "tool_use":
				name, _ := block["name"].(string)
				input, _ := block["input"].(map[string]any)

				line := formatToolUse(name, input)
				if line != "" {
					lines = append(lines, fmt.Sprintf("[%s] %s", role, line))
				}

			case "tool_result":
				isError, _ := block["is_error"].(bool)
				if !isError {
					continue // omit successful results
				}

				text, _ := block["content"].(string)
				if text == "" {
					text, _ = block["text"].(string)
				}

				if len(text) > 300 {
					text = text[:300] + "..."
				}

				lines = append(lines, fmt.Sprintf("[%s] ERROR: %s", role, text))

			case "thinking":
				// skip
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", 0, err
	}

	return strings.Join(lines, "\n\n"), bytesRead, nil
}

// unexported variables.
var (
	systemReminderRe = regexp.MustCompile(`(?s)<system-reminder>.*?</system-reminder>`)
	teammateRe       = regexp.MustCompile(`(?s)<teammate-message[^>]*teammate_id="([^"]*)"[^>]*>(.*?)</teammate-message>`)
)

// computeEditDiff shows only what changed between old and updated strings.
func computeEditDiff(old, updated string) string {
	// For short edits, show full old/updated lines
	if len(old) < 200 && len(updated) < 200 {
		var result strings.Builder
		result.WriteString("  - " + old + "\n")
		result.WriteString("  + " + updated)

		return result.String()
	}

	// Find common prefix
	prefixLen := 0

	for i := 0; i < len(old) && i < len(updated); i++ {
		if old[i] != updated[i] {
			break
		}

		prefixLen = i + 1
	}

	// Find common suffix
	suffixLen := 0

	for i := 1; i <= len(old)-prefixLen && i <= len(updated)-prefixLen; i++ {
		if old[len(old)-i] != updated[len(updated)-i] {
			break
		}

		suffixLen = i
	}

	oldChanged := old[prefixLen : len(old)-suffixLen]
	newChanged := updated[prefixLen : len(updated)-suffixLen]

	// Add context (up to 60 chars around the change)
	ctx := 60
	ctxBefore := ""

	if prefixLen > 0 {
		start := max(prefixLen-ctx, 0)

		ctxBefore = old[start:prefixLen]
	}

	var result strings.Builder
	if ctxBefore != "" {
		result.WriteString("  ..." + ctxBefore + "\n")
	}

	if oldChanged != "" {
		result.WriteString("  - " + oldChanged + "\n")
	}

	if newChanged != "" {
		result.WriteString("  + " + newChanged)
	}

	return result.String()
}

// formatToolUse formats a tool invocation for stripped output.
func formatToolUse(name string, input map[string]any) string {
	switch name {
	case "Bash":
		cmd, _ := input["command"].(string)
		// Truncate heredocs
		if idx := strings.Index(cmd, "<<"); idx > 0 {
			if nl := strings.Index(cmd[idx:], "\n"); nl > 0 {
				cmd = cmd[:idx+nl] + fmt.Sprintf("... [%d chars]", len(cmd))
			}
		}

		return "TOOL:Bash $ " + cmd

	case "Edit":
		fp, _ := input["file_path"].(string)
		old, _ := input["old_string"].(string)
		newStr, _ := input["new_string"].(string)

		if old == "" {
			return "TOOL:Edit " + fp
		}

		diff := computeEditDiff(old, newStr)

		return fmt.Sprintf("TOOL:Edit %s\n%s", fp, diff)

	case "Write":
		fp, _ := input["file_path"].(string)
		return "TOOL:Write " + fp

	case "Read", "Glob", "Grep":
		b, _ := json.Marshal(input)

		s := string(b)
		if len(s) > 150 {
			s = s[:150] + "..."
		}

		return fmt.Sprintf("TOOL:%s %s", name, s)

	default:
		b, _ := json.Marshal(input)

		s := string(b)
		if len(s) > 150 {
			s = s[:150] + "..."
		}

		return fmt.Sprintf("TOOL:%s %s", name, s)
	}
}

// isSkillContent detects skill loading patterns.
func isSkillContent(text string) bool {
	return strings.Contains(text, "Base directory for this skill:") ||
		strings.Contains(text, "Launching skill:")
}

// stripNoise removes system-reminders and extracts teammate messages.
func stripNoise(text string) string {
	// Extract teammate messages first (before stripping system-reminders)
	text = teammateRe.ReplaceAllStringFunc(text, func(match string) string {
		m := teammateRe.FindStringSubmatch(match)
		if len(m) >= 3 {
			return fmt.Sprintf("[teammate %s] %s", m[1], strings.TrimSpace(m[2]))
		}

		return match
	})

	// Strip system-reminders
	text = systemReminderRe.ReplaceAllString(text, "")

	// Collapse skill content
	if isSkillContent(text) {
		return "(skill loaded)"
	}

	return strings.TrimSpace(text)
}
