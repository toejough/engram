package chat

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"

	"engram/internal/watch"
)

// FileWatcher watches a chat file for messages matching agent and type filter.
// All I/O is injected.
type FileWatcher struct {
	FilePath  string
	FSWatcher watch.Watcher
	ReadFile  func(path string) ([]byte, error)
}

// Watch blocks until a matching message arrives after cursor in the chat file.
// agent matches messages where the To field contains the agent name or "all".
// msgTypes filters by message type; empty or nil slice matches all types.
// Returns the matching message and the new cursor (total line count).
func (w *FileWatcher) Watch(ctx context.Context, agent string, cursor int, msgTypes []string) (Message, int, error) {
	for {
		data, readErr := w.ReadFile(w.FilePath)
		if readErr != nil {
			return Message{}, 0, fmt.Errorf("reading chat file: %w", readErr)
		}

		msg, newCursor, found := findMessage(data, agent, cursor, msgTypes)
		if found {
			return msg, newCursor, nil
		}

		waitErr := w.FSWatcher.WaitForChange(ctx, w.FilePath)
		if waitErr != nil {
			return Message{}, 0, waitErr
		}
	}
}

// ParseMessages deserializes TOML chat data into a Message slice.
// Returns an error if the TOML is malformed; returns nil slice for empty data.
func ParseMessages(data []byte) ([]Message, error) {
	var parsed struct {
		Messages []Message `toml:"message"`
	}

	err := toml.Unmarshal(data, &parsed)
	if err != nil {
		return nil, fmt.Errorf("parsing chat TOML: %w", err)
	}

	return parsed.Messages, nil
}

// findMessage scans data for the first message after cursor that matches agent and msgTypes.
// Returns the message, new cursor (total line count), and whether a match was found.
func findMessage(data []byte, agent string, cursor int, msgTypes []string) (Message, int, bool) {
	startIdx := messageIndexAtCursor(data, cursor)
	newCursor := bytes.Count(data, []byte("\n"))

	messages, err := ParseMessages(data)
	if err != nil {
		return Message{}, newCursor, false
	}

	for i := startIdx; i < len(messages); i++ {
		msg := messages[i]
		if matchesAgent(msg.To, agent) && matchesType(msg.Type, msgTypes) {
			return msg, newCursor, true
		}
	}

	return Message{}, newCursor, false
}

// matchesAgent reports whether the To field targets the given agent.
// The To field may be "all", a single agent name, or comma-separated names.
func matchesAgent(to, agent string) bool {
	for part := range strings.SplitSeq(to, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed == "all" || trimmed == agent {
			return true
		}
	}

	return false
}

// matchesType reports whether msgType is in the allowed types list.
// An empty list matches all types.
func matchesType(msgType string, allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}

	return slices.Contains(allowed, msgType)
}

// messageIndexAtCursor counts [[message]] occurrences in data up to (but not including) line cursor.
// Returns the index of the first message that starts at or after cursor.
func messageIndexAtCursor(data []byte, cursor int) int {
	if cursor <= 0 {
		return 0
	}

	lines := bytes.Split(data, []byte("\n"))
	header := []byte("[[message]]")
	count := 0

	for lineNum, line := range lines {
		if lineNum >= cursor {
			break
		}

		if bytes.Equal(bytes.TrimSpace(line), header) {
			count++
		}
	}

	return count
}
