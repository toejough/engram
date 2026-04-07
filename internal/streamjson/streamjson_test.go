// internal/streamjson/streamjson_test.go
package streamjson_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/streamjson"
)

func TestDetectSpeechMarkers_IntentPrefix_Detected(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	text := "INTENT: Situation: About to run targ check-full.\nBehavior: Will execute the check command."
	markers := streamjson.DetectSpeechMarkers(text)

	g.Expect(markers).To(HaveLen(1))
	g.Expect(markers[0].Prefix).To(Equal("INTENT"))
	g.Expect(markers[0].Text).To(ContainSubstring("Situation: About to run targ check-full."))
}

func TestDetectSpeechMarkers_MultipleMarkers_AllDetected(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	text := "INTENT: Situation: X. Behavior: Y.\n\nSome prose.\n\nACK: No objection, proceed."
	markers := streamjson.DetectSpeechMarkers(text)

	g.Expect(markers).To(HaveLen(2))
	g.Expect(markers[0].Prefix).To(Equal("INTENT"))
	g.Expect(markers[1].Prefix).To(Equal("ACK"))
}

func TestDetectSpeechMarkers_NoMarkers_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	markers := streamjson.DetectSpeechMarkers("Just regular prose with no markers.")

	g.Expect(markers).To(BeEmpty())
}

func TestDetectSpeechMarkers_WaitWithRecipient_Detected(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	text := "WAIT: (to engram-agent) I have a concern about the approach."
	markers := streamjson.DetectSpeechMarkers(text)

	g.Expect(markers).To(HaveLen(1))
	g.Expect(markers[0].Prefix).To(Equal("WAIT"))
	g.Expect(markers[0].Text).To(ContainSubstring("I have a concern"))
}

func TestParse_AssistantEvent_ExtractsSessionIDAndText(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	sessionID := "550e8400-e29b-41d4-a716-446655440000"
	line := []byte(`{"type":"assistant","session_id":"` + sessionID +
		`","message":{"content":[{"type":"text","text":"Hello world"}]}}`)
	event, err := streamjson.Parse(line)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(event.Type).To(Equal("assistant"))
	g.Expect(event.SessionID).To(Equal(sessionID))
	g.Expect(event.Text).To(Equal("Hello world"))
}

func TestParse_EmptyLine_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	_, err := streamjson.Parse([]byte(``))

	g.Expect(err).To(HaveOccurred())
}

func TestParse_MalformedJSON_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	_, err := streamjson.Parse([]byte(`{not json`))

	g.Expect(err).To(HaveOccurred())
}

func TestParse_SystemEvent_ExtractsSessionIDOnly(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	sessionID := "550e8400-e29b-41d4-a716-446655440000"
	line := []byte(`{"type":"system","session_id":"` + sessionID + `","subtype":"init"}`)
	event, err := streamjson.Parse(line)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(event.Type).To(Equal("system"))
	g.Expect(event.SessionID).To(Equal(sessionID))
	g.Expect(event.Text).To(BeEmpty())
}
