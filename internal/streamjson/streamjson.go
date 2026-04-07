// Package streamjson parses JSONL events from claude -p --output-format=stream-json.
package streamjson

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Exported variables.
var (
	// ErrEmptyLine is returned when Parse receives an empty input line.
	ErrEmptyLine = errors.New("streamjson: empty line")
)

// Event is a single parsed JSONL event from claude -p --verbose --output-format=stream-json.
// All events carry session_id (the Claude conversation UUID).
// Text is populated for assistant events only.
type Event struct {
	Type      string // "assistant", "tool_use", "result", "system", "user", "error"
	SessionID string // Claude conversation UUID — present on all events
	Text      string // assembled text from content blocks (assistant events only)
}

// SpeechMarker is a detected prefix marker in an assistant text block.
// Markers must appear at the start of a line (column 0) to be detected.
type SpeechMarker struct {
	Prefix string // "INTENT", "ACK", "WAIT", "DONE", "LEARNED", "INFO", "READY", "ESCALATE"
	Text   string // content from the "PREFIX: " through end of speech act (trimmed)
}

// DetectSpeechMarkers scans the Text field of an assistant event for prefix markers.
// A marker is detected when a line starts with "PREFIX: " (case-sensitive, column 0).
// Multi-line speech acts are terminated by the next prefix marker or end of text.
// Blank lines within a speech act are included in the body (trimmed on flush).
func DetectSpeechMarkers(text string) []SpeechMarker {
	lines := strings.Split(text, "\n")
	markers := make([]SpeechMarker, 0)

	var current *SpeechMarker

	var currentLines []string

	flush := func() {
		if current != nil {
			current.Text = strings.TrimSpace(strings.Join(currentLines, "\n"))
			markers = append(markers, *current)
			current = nil
			currentLines = nil
		}
	}

	for _, line := range lines {
		prefix, rest, found := detectPrefix(line)
		if found {
			flush()

			current = &SpeechMarker{Prefix: prefix}
			currentLines = []string{rest}

			continue
		}

		if current != nil {
			currentLines = append(currentLines, line)
		}
	}

	flush()

	return markers
}

// Parse parses one JSONL line into an Event.
// Returns an error for malformed JSON or empty input.
func Parse(line []byte) (Event, error) {
	if len(line) == 0 {
		return Event{}, ErrEmptyLine
	}

	var raw rawEvent

	err := json.Unmarshal(line, &raw)
	if err != nil {
		return Event{}, fmt.Errorf("streamjson: malformed JSON: %w", err)
	}

	event := Event{
		Type:      raw.Type,
		SessionID: raw.SessionID,
	}

	if raw.Type == "assistant" && raw.Message != nil {
		var texts []string

		for _, block := range raw.Message.Content {
			if block.Type == "text" && block.Text != "" {
				texts = append(texts, block.Text)
			}
		}

		event.Text = strings.Join(texts, "\n")
	}

	return event, nil
}

// unexported variables.
var (
	knownPrefixes = []string{ //nolint:gochecknoglobals
		"INTENT", "ACK", "WAIT", "DONE", "LEARNED", "INFO", "READY", "ESCALATE",
	}
)

type rawContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// rawEvent is the JSON shape of a claude -p stream-json event.
//
//nolint:tagliatelle // Claude stream-json protocol uses snake_case: session_id
type rawEvent struct {
	Type      string      `json:"type"`
	SessionID string      `json:"session_id"`
	Message   *rawMessage `json:"message,omitempty"`
}

type rawMessage struct {
	Content []rawContent `json:"content"`
}

// detectPrefix checks if a line starts with a known "PREFIX: " marker.
func detectPrefix(line string) (prefix, rest string, found bool) {
	for _, p := range knownPrefixes {
		marker := p + ": "

		if after, ok := strings.CutPrefix(line, marker); ok {
			return p, after, true
		}

		if line == p+":" {
			return p, "", true
		}
	}

	return "", "", false
}
