package server_test

import (
	"strings"
	"testing"

	"engram/internal/server"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"
)

func TestParseStreamResponse_AlwaysExtractsSessionID(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		sessionID := rapid.StringMatching(`[a-z0-9\-]{5,20}`).Draw(rt, "session_id")
		input := makeStreamInput(sessionID, surfaceJSON)

		result, parseErr := server.ParseStreamResponse(strings.NewReader(input))
		g.Expect(parseErr).NotTo(HaveOccurred())

		if parseErr != nil {
			return
		}

		g.Expect(result.SessionID).To(Equal(sessionID))
	})
}

func TestParseStreamResponse_LearnAction(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	input := makeStreamInput("sess-789", learnJSON)

	result, parseErr := server.ParseStreamResponse(strings.NewReader(input))
	g.Expect(parseErr).NotTo(HaveOccurred())

	if parseErr != nil {
		return
	}

	g.Expect(result.SessionID).To(Equal("sess-789"))
	g.Expect(result.Action).To(Equal("learn"))
	g.Expect(result.To).To(Equal("lead-1"))
	g.Expect(result.Text).To(Equal("Learned: always DI"))
	g.Expect(result.Saved).To(BeTrue())
	g.Expect(result.Path).To(Equal("/mem/foo.md"))
}

func TestParseStreamResponse_LogOnlyAction(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	input := makeStreamInput("sess-456", logOnlyJSON)

	result, parseErr := server.ParseStreamResponse(strings.NewReader(input))
	g.Expect(parseErr).NotTo(HaveOccurred())

	if parseErr != nil {
		return
	}

	g.Expect(result.SessionID).To(Equal("sess-456"))
	g.Expect(result.Action).To(Equal("log-only"))
	g.Expect(result.Text).To(Equal("Internal note"))
}

func TestParseStreamResponse_MalformedJSON_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	input := makeStreamInput("sess-abc", "not json at all")

	_, parseErr := server.ParseStreamResponse(strings.NewReader(input))
	g.Expect(parseErr).To(MatchError(server.ErrMalformedAction))
}

func TestParseStreamResponse_NoAssistantText_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Only system event, no assistant event.
	input := `{"type":"system","session_id":"sess-123"}`

	_, parseErr := server.ParseStreamResponse(strings.NewReader(input))
	g.Expect(parseErr).To(MatchError(server.ErrNoAssistantText))
}

func TestParseStreamResponse_SurfaceAction(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	input := makeStreamInput("sess-123", surfaceJSON)

	result, parseErr := server.ParseStreamResponse(strings.NewReader(input))
	g.Expect(parseErr).NotTo(HaveOccurred())

	if parseErr != nil {
		return
	}

	g.Expect(result.SessionID).To(Equal("sess-123"))
	g.Expect(result.Action).To(Equal("surface"))
	g.Expect(result.To).To(Equal("lead-1"))
	g.Expect(result.Text).To(Equal("Memory: use DI"))
}

// unexported constants.
const (
	// learnJSON is the inner JSON for a learn action with all fields.
	learnJSON = `{"action":"learn","to":"lead-1","text":"Learned: always DI","saved":true,"path":"/mem/foo.md"}`
	// logOnlyJSON is the inner JSON for a log-only action.
	logOnlyJSON = `{"action":"log-only","text":"Internal note"}`
	// surfaceJSON is the inner JSON for a surface action.
	surfaceJSON = `{"action":"surface","to":"lead-1","text":"Memory: use DI"}`
)

// makeStreamInput builds a two-line stream-json JSONL input with the given
// session ID and inner JSON text (the assistant's response payload).
func makeStreamInput(sessionID, innerJSON string) string {
	systemLine := `{"type":"system","session_id":"` + sessionID + `"}`
	escapedInner := strings.ReplaceAll(innerJSON, `"`, `\"`)
	assistantLine := `{"type":"assistant","message":{"content":[` +
		`{"type":"text","text":"` + escapedInner + `"}]}}`

	return strings.Join([]string{systemLine, assistantLine}, "\n")
}
