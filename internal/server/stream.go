package server

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// Exported variables.
var (
	// ErrMalformedAction is returned when the assistant text is not valid structured JSON.
	ErrMalformedAction = errors.New("stream: malformed action JSON in assistant text")
	// ErrNoAssistantText is returned when no assistant text is found in the stream.
	ErrNoAssistantText = errors.New("stream: no assistant text found")
)

// AgentResponse is the parsed structured output from the engram-agent.
type AgentResponse struct {
	SessionID string `json:"-"`      // From the system event, not the agent's JSON.
	Action    string `json:"action"` // "surface", "log-only", or "learn"
	To        string `json:"to"`     // Recipient for surface/learn actions.
	Text      string `json:"text"`   // Content (surfaced memory or learn outcome).
	Saved     bool   `json:"saved"`  // For learn action: whether memory was persisted.
	Path      string `json:"path"`   // For learn action: file path of saved memory.
}

// ParseStreamResponse reads stream-json JSONL from an io.Reader (the stdout of
// claude -p --output-format=stream-json) and extracts the session ID and
// structured JSON response from the assistant text.
func ParseStreamResponse(reader io.Reader) (AgentResponse, error) {
	sessionID, assistantText, scanErr := scanStream(reader)
	if scanErr != nil {
		return AgentResponse{}, scanErr
	}

	var resp AgentResponse

	jsonErr := json.Unmarshal([]byte(assistantText), &resp)
	if jsonErr != nil {
		return AgentResponse{}, fmt.Errorf("%w: %w", ErrMalformedAction, jsonErr)
	}

	resp.SessionID = sessionID

	return resp, nil
}

type streamBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// unexported types.

//nolint:tagliatelle // Claude stream-json protocol uses snake_case: session_id
type streamEvent struct {
	Type      string     `json:"type"`
	SessionID string     `json:"session_id"`
	Message   *streamMsg `json:"message"`
}

type streamMsg struct {
	Content []streamBlock `json:"content"`
}

// extractAssistantText returns the first non-empty text block from content.
func extractAssistantText(blocks []streamBlock) string {
	for _, block := range blocks {
		if block.Type == "text" && block.Text != "" {
			return block.Text
		}
	}

	return ""
}

// unexported functions.

// scanStream scans JSONL lines from reader and returns the session ID and
// last assistant text found.
func scanStream(reader io.Reader) (sessionID, assistantText string, err error) {
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var event streamEvent

		unmarshalErr := json.Unmarshal(line, &event)
		if unmarshalErr != nil {
			continue // Skip malformed lines.
		}

		if event.SessionID != "" {
			sessionID = event.SessionID
		}

		if event.Type == "assistant" && event.Message != nil {
			text := extractAssistantText(event.Message.Content)
			if text != "" {
				assistantText = text
			}
		}
	}

	if assistantText == "" {
		return "", "", ErrNoAssistantText
	}

	return sessionID, assistantText, nil
}
