package chat

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
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

// MatchesAgent reports whether the To field targets the given agent.
// The To field may be "all", a single agent name, or comma-separated names.
// An empty agent string matches any To field (wildcard — used by dispatchLoop).
func MatchesAgent(to, agent string) bool {
	if agent == "" {
		return true // empty = match all recipients
	}

	for part := range strings.SplitSeq(to, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed == "all" || trimmed == agent {
			return true
		}
	}

	return false
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

// ParseMessagesSafe deserializes TOML chat data tolerating per-message corruption.
// Fast path: attempts full-file ParseMessages (zero allocation overhead for clean files).
// Fallback: splits on [[message]] boundaries, parses each block independently.
// Corrupt blocks are logged via slog.Warn and skipped. Never returns an error.
func ParseMessagesSafe(data []byte) []Message {
	if len(data) == 0 {
		return nil
	}

	msgs, err := ParseMessages(data)
	if err == nil {
		return msgs
	}

	slog.Warn("ParseMessagesSafe: full parse failed, falling back to per-block parsing", "err", err)

	const boundary = "\n[[message]]"

	parts := bytes.Split(data, []byte(boundary))

	result := make([]Message, 0, len(parts))

	for i, part := range parts {
		var block []byte

		if i == 0 {
			block = bytes.TrimSpace(part)
			if len(block) == 0 {
				continue
			}
		} else {
			block = append([]byte("[[message]]"), part...)
		}

		blockMsgs, blockErr := ParseMessages(block)
		if blockErr != nil {
			slog.Warn("ParseMessagesSafe: skipping corrupt block", "block_index", i, "err", blockErr)
			continue
		}

		result = append(result, blockMsgs...)
	}

	return result
}

// findMessage scans data for the first message after cursor that matches agent and msgTypes.
// Returns the message, new cursor (total line count), and whether a match was found.
// Only the bytes after the cursor line are parsed, so corrupt historical data before
// the cursor does not prevent finding new messages.
func findMessage(data []byte, agent string, cursor int, msgTypes []string) (Message, int, bool) {
	newCursor := bytes.Count(data, []byte("\n"))

	suffix := suffixAtLine(data, cursor)

	messages := ParseMessagesSafe(suffix)

	for _, msg := range messages {
		if matchesAgent(msg.To, agent) && matchesType(msg.Type, msgTypes) {
			return msg, newCursor, true
		}
	}

	return Message{}, newCursor, false
}

// matchesAgent is the package-internal alias for MatchesAgent.
func matchesAgent(to, agent string) bool {
	return MatchesAgent(to, agent)
}

// matchesType reports whether msgType is in the allowed types list.
// An empty list matches all types.
func matchesType(msgType string, allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}

	return slices.Contains(allowed, msgType)
}

// suffixAtLine returns the portion of data starting at the given line number.
// Returns the full data if lineNum <= 0, nil if lineNum exceeds the line count.
func suffixAtLine(data []byte, lineNum int) []byte {
	if lineNum <= 0 {
		return data
	}

	offset := 0
	for range lineNum {
		idx := bytes.IndexByte(data[offset:], '\n')
		if idx < 0 {
			return nil
		}

		offset += idx + 1
	}

	return data[offset:]
}
