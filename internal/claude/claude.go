// Package claude owns the claude -p stream pipeline for one agent session.
// All I/O is injected — no os.* calls in this package.
package claude

import (
	"bufio"
	"fmt"
	"io"

	"engram/internal/chat"
	"engram/internal/streamjson"
)

// Runner owns the claude -p stream pipeline for one agent session.
type Runner struct {
	AgentName string
	Pane      io.Writer   // filtered output destination; os.Stdout when run in-pane
	Poster    chat.Poster // posts relayed speech markers to the chat file

	// WriteSessionID is called with the Claude conversation UUID extracted from the
	// first JSONL event (before any speech act). Injected by runAgentRun to update
	// the state file without creating an internal/claude → internal/cli import cycle.
	// May be nil (skips session-id write — used in tests that don't need it).
	WriteSessionID func(sessionID string) error

	// WriteState is called with "ACTIVE" when the first READY: marker is detected.
	// Nil = skip (tests that don't need state transitions).
	WriteState func(state string) error
}

// ProcessStream reads JSONL from src, applies the display filter, relays speech
// markers to chat via Poster, calls WriteSessionID on the first event, and returns
// a StreamResult describing what was detected. The stream ends when src returns io.EOF.
func (r *Runner) ProcessStream(src io.Reader) (StreamResult, error) {
	scanner := bufio.NewScanner(src)

	var result StreamResult

	sessionIDWritten := false

	for scanner.Scan() {
		line := scanner.Bytes()

		event, parseErr := streamjson.Parse(line)
		if parseErr != nil {
			// Non-JSON lines (startup noise before stream begins): pass through to pane.
			_, _ = fmt.Fprintf(r.Pane, "%s\n", line)

			continue
		}

		if !sessionIDWritten {
			sessionIDWritten = r.maybeWriteSessionID(event.SessionID, &result)
		}

		r.handleEvent(event, &result)
	}

	scanErr := scanner.Err()
	if scanErr != nil {
		return result, fmt.Errorf("scanning stream: %w", scanErr)
	}

	return result, nil
}

// handleEvent dispatches a parsed event to the appropriate output handler.
func (r *Runner) handleEvent(event streamjson.Event, result *StreamResult) {
	switch event.Type {
	case "assistant":
		if event.Text != "" {
			_, _ = fmt.Fprintf(r.Pane, "%s\n", event.Text)
		}

		for _, marker := range streamjson.DetectSpeechMarkers(event.Text) {
			r.handleMarker(marker, result)
		}

		if prose := streamjson.NonMarkerText(event.Text); prose != "" {
			r.relayConversation(prose)
		}
	case "user":
		if event.Text != "" {
			_, _ = fmt.Fprintf(r.Pane, "%s\n", event.Text)
		}
	default:
		// system, tool_use, result, error: display-filtered (not written to pane).
	}
}

// handleMarker processes a single speech marker, updating result flags and relaying to chat.
func (r *Runner) handleMarker(marker streamjson.SpeechMarker, result *StreamResult) {
	switch marker.Prefix {
	case "INTENT":
		result.IntentDetected = true
	case "DONE":
		result.DoneDetected = true
	case "READY":
		if !result.ReadyDetected {
			result.ReadyDetected = true

			r.maybeWriteState("ACTIVE")
		}
	case "WAIT":
		result.WaitDetected = true
	}

	relayErr := r.relayMarker(marker)
	if relayErr != nil {
		_, _ = fmt.Fprintf(r.Pane, "[engram] warning: relay failed: %v\n", relayErr)
	}
}

// maybeWriteSessionID writes the session ID if not yet written and returns true when written.
func (r *Runner) maybeWriteSessionID(sessionID string, result *StreamResult) bool {
	if sessionID == "" || sessionID == "PENDING" {
		return false
	}

	result.SessionID = sessionID

	if r.WriteSessionID == nil {
		return true
	}

	writeErr := r.WriteSessionID(sessionID)
	if writeErr != nil {
		_, _ = fmt.Fprintf(r.Pane, "[engram] warning: failed to write session-id: %v\n", writeErr)

		// Returning false retries the write on the next event. This prevents a silent
		// permanent failure where the session-id is never persisted.
		return false
	}

	return true
}

// maybeWriteState calls WriteState if the callback is injected.
func (r *Runner) maybeWriteState(state string) {
	if r.WriteState == nil {
		return
	}

	stateErr := r.WriteState(state)
	if stateErr != nil {
		_, _ = fmt.Fprintf(r.Pane, "[engram] warning: state transition failed: %v\n", stateErr)
	}
}

// relayConversation posts non-marker prose from an agent turn to chat as type "conversation".
func (r *Runner) relayConversation(text string) {
	msg := chat.Message{
		From:   r.AgentName,
		To:     "all",
		Thread: "speech-relay",
		Type:   "conversation",
		Text:   text,
	}

	_, err := r.Poster.Post(msg)
	if err != nil {
		_, _ = fmt.Fprintf(r.Pane, "[engram] warning: conversation relay failed: %v\n", err)
	}
}

// relayMarker maps a SpeechMarker prefix to a chat message type and posts it via Poster.
func (r *Runner) relayMarker(marker streamjson.SpeechMarker) error {
	msgType := markerToMsgType(marker.Prefix)
	if msgType == "" {
		return nil
	}

	msg := chat.Message{
		From:   r.AgentName,
		To:     "engram-agent",
		Thread: "speech-relay",
		Type:   msgType,
		Text:   marker.Text,
	}

	_, err := r.Poster.Post(msg)
	if err != nil {
		return fmt.Errorf("posting marker %q: %w", marker.Prefix, err)
	}

	return nil
}

// StreamResult is the outcome of processing one JSONL stream (one claude -p turn).
type StreamResult struct {
	IntentDetected bool   // true if at least one INTENT: prefix marker was detected
	DoneDetected   bool   // true if a DONE: prefix marker was detected
	WaitDetected   bool   // true if a WAIT: prefix marker was detected
	ReadyDetected  bool   // true if a READY: prefix marker was detected (triggers STARTING→ACTIVE)
	SessionID      string // Claude conversation UUID extracted from the first JSONL event
}

// markerToMsgType converts a speech prefix to a chat message type.
func markerToMsgType(prefix string) string {
	switch prefix {
	case "INTENT":
		return "intent"
	case "ACK":
		return "ack"
	case "WAIT":
		return "wait"
	case "DONE":
		return "done"
	case "LEARNED":
		return "learned"
	case "INFO":
		return "info"
	case "READY":
		return "ready"
	case "ESCALATE":
		return "escalate"
	default:
		return ""
	}
}
