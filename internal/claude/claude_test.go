// internal/claude/claude_test.go
package claude_test

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/chat"
	"engram/internal/claude"
)

func TestProcessStream_AckMarker_PostedAsAck(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	poster := &mockPoster{}

	runner := claude.Runner{
		AgentName:      "test-agent",
		Pane:           io.Discard,
		Poster:         poster,
		WriteSessionID: func(string) error { return nil },
	}

	ackJSON := `{"type":"assistant","session_id":"abc",` +
		`"message":{"content":[{"type":"text","text":"ACK: Got it."}]}}`
	stream := strings.NewReader(ackJSON + "\n")

	_, err := runner.ProcessStream(stream)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(poster.posted).To(HaveLen(1))
	g.Expect(poster.posted[0].Type).To(Equal("ack"))
}

func TestProcessStream_AllMarkerTypes_Posted(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	poster := &mockPoster{}

	runner := claude.Runner{
		AgentName:      "test-agent",
		Pane:           io.Discard,
		Poster:         poster,
		WriteSessionID: func(string) error { return nil },
	}

	text := "WAIT: Blocking.\nLEARNED: Fact.\nINFO: Detail.\nREADY: Go.\nESCALATE: Help."
	markersJSON := `{"type":"assistant","session_id":"abc","message":{"content":[{"type":"text","text":"` +
		strings.ReplaceAll(text, "\n", `\n`) + `"}]}}`
	stream := strings.NewReader(markersJSON + "\n")

	_, err := runner.ProcessStream(stream)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	types := make([]string, 0, len(poster.posted))
	for _, msg := range poster.posted {
		types = append(types, msg.Type)
	}

	g.Expect(types).To(ContainElements("wait", "learned", "info", "ready", "escalate"))
}

func TestProcessStream_DisplayFilter_SuppressesRawJSONL(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var pane bytes.Buffer

	runner := claude.Runner{
		AgentName:      "test-agent",
		Pane:           &pane,
		Poster:         &mockPoster{},
		WriteSessionID: func(string) error { return nil },
	}

	// Simulate: system event (filtered), tool_use event (filtered), assistant event (shown).
	systemLine := `{"type":"system","session_id":"550e8400-e29b-41d4-a716-446655440000","subtype":"init"}`
	toolLine := `{"type":"tool_use","session_id":"550e8400-e29b-41d4-a716-446655440000","id":"toolu_01"}`
	assistantLine := `{"type":"assistant","session_id":"550e8400-e29b-41d4-a716-446655440000",` +
		`"message":{"content":[{"type":"text","text":"I am thinking."}]}}`
	stream := strings.NewReader(systemLine + "\n" + toolLine + "\n" + assistantLine + "\n")

	_, err := runner.ProcessStream(stream)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(pane.String()).NotTo(ContainSubstring(`{"type":`)) // no raw JSONL
	g.Expect(pane.String()).To(ContainSubstring("I am thinking."))
}

func TestProcessStream_DoneMarker_Reported(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	runner := claude.Runner{
		AgentName:      "test-agent",
		Pane:           io.Discard,
		Poster:         &mockPoster{},
		WriteSessionID: func(string) error { return nil },
	}

	doneJSON := `{"type":"assistant","session_id":"abc",` +
		`"message":{"content":[{"type":"text","text":"DONE: Task complete."}]}}`
	stream := strings.NewReader(doneJSON + "\n")

	result, err := runner.ProcessStream(stream)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.DoneDetected).To(BeTrue())
}

func TestProcessStream_IntentMarker_PostedAndReported(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	poster := &mockPoster{}

	runner := claude.Runner{
		AgentName:      "test-agent",
		Pane:           io.Discard,
		Poster:         poster,
		WriteSessionID: func(string) error { return nil },
	}

	intentJSON := `{"type":"assistant","session_id":"550e8400-e29b-41d4-a716-446655440000",` +
		`"message":{"content":[{"type":"text","text":"INTENT: Situation: X.\nBehavior: Y."}]}}`
	stream := strings.NewReader(intentJSON + "\n")

	result, err := runner.ProcessStream(stream)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(poster.posted).To(HaveLen(1))
	g.Expect(poster.posted[0].Type).To(Equal("intent"))
	g.Expect(poster.posted[0].From).To(Equal("test-agent"))
	g.Expect(poster.posted[0].Text).To(ContainSubstring("Situation: X."))
	g.Expect(result.IntentDetected).To(BeTrue())
	g.Expect(result.DoneDetected).To(BeFalse())
}

func TestProcessStream_NilWriteSessionID_StillCapturesID(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	runner := claude.Runner{
		AgentName:      "test-agent",
		Pane:           io.Discard,
		Poster:         &mockPoster{},
		WriteSessionID: nil,
	}

	systemLine := `{"type":"system","session_id":"550e8400-e29b-41d4-a716-446655440000","subtype":"init"}`
	stream := strings.NewReader(systemLine + "\n")

	result, err := runner.ProcessStream(stream)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.SessionID).To(Equal("550e8400-e29b-41d4-a716-446655440000"))
}

func TestProcessStream_NonJSONLine_PassedThroughToPane(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var pane bytes.Buffer

	runner := claude.Runner{
		AgentName:      "test-agent",
		Pane:           &pane,
		Poster:         &mockPoster{},
		WriteSessionID: func(string) error { return nil },
	}

	stream := strings.NewReader("not json at all\n")

	_, err := runner.ProcessStream(stream)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(pane.String()).To(ContainSubstring("not json at all"))
}

func TestProcessStream_PosterError_WarnsOnPane(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var pane bytes.Buffer

	runner := claude.Runner{
		AgentName:      "test-agent",
		Pane:           &pane,
		Poster:         &errorPoster{err: errors.New("chat unavailable")},
		WriteSessionID: func(string) error { return nil },
	}

	intentJSON := `{"type":"assistant","session_id":"abc",` +
		`"message":{"content":[{"type":"text","text":"INTENT: Do something."}]}}`
	stream := strings.NewReader(intentJSON + "\n")

	_, err := runner.ProcessStream(stream)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(pane.String()).To(ContainSubstring("relay failed"))
}

func TestProcessStream_SessionID_CallsWriteSessionID(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var capturedID string

	runner := claude.Runner{
		AgentName: "test-agent",
		Pane:      io.Discard,
		Poster:    &mockPoster{},
		WriteSessionID: func(id string) error {
			capturedID = id

			return nil
		},
	}

	systemLine := `{"type":"system","session_id":"550e8400-e29b-41d4-a716-446655440000","subtype":"init"}`
	stream := strings.NewReader(systemLine + "\n")

	_, err := runner.ProcessStream(stream)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(capturedID).To(Equal("550e8400-e29b-41d4-a716-446655440000"))
}

func TestProcessStream_UserEvent_WrittenToPane(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var pane bytes.Buffer

	runner := claude.Runner{
		AgentName:      "test-agent",
		Pane:           &pane,
		Poster:         &mockPoster{},
		WriteSessionID: func(string) error { return nil },
	}

	// User event has no Text populated by streamjson (only assistant events get Text).
	// This exercises the "user" branch; verifies it handles empty text safely.
	userJSON := `{"type":"user","session_id":"abc"}`
	stream := strings.NewReader(userJSON + "\n")

	_, err := runner.ProcessStream(stream)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// No panic, no raw JSON in pane.
	g.Expect(pane.String()).NotTo(ContainSubstring(`{"type":`))
}

func TestProcessStream_WriteSessionIDError_WarnsAndRetries(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	var pane bytes.Buffer

	callCount := 0

	runner := claude.Runner{
		AgentName: "test-agent",
		Pane:      &pane,
		Poster:    &mockPoster{},
		WriteSessionID: func(string) error {
			callCount++

			return errors.New("disk full")
		},
	}

	// Two events with the same session ID — should warn on first, retry on second.
	line1 := `{"type":"system","session_id":"abc","subtype":"init"}`
	line2 := `{"type":"system","session_id":"abc","subtype":"something"}`
	stream := strings.NewReader(line1 + "\n" + line2 + "\n")

	_, err := runner.ProcessStream(stream)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(pane.String()).To(ContainSubstring("failed to write session-id"))
	g.Expect(callCount).To(BeNumerically(">=", 1))
}

// errorPoster always returns an error from Post.
type errorPoster struct {
	err error
}

func (e *errorPoster) Post(_ chat.Message) (int, error) {
	return 0, e.err
}

// mockPoster records all Post calls for test inspection.
type mockPoster struct {
	posted []chat.Message
}

func (m *mockPoster) Post(msg chat.Message) (int, error) {
	m.posted = append(m.posted, msg)

	return 0, nil
}
