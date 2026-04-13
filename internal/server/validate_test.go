package server_test

import (
	"testing"

	"engram/internal/server"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"
)

func TestValidateLearn_InvalidJSONRejected(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := server.ValidateLearnMessage(`not json at all`)
	g.Expect(err).To(MatchError(ContainSubstring("invalid JSON")))
}

func TestValidateLearn_InvalidTypeAlwaysRejected(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := server.ValidateLearnMessage(`{"type":"bogus","situation":"s"}`)
	g.Expect(err).To(MatchError(ContainSubstring("must be 'feedback' or 'fact'")))
}

func TestValidateLearn_MissingFactFieldRejected(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Missing "object" field.
	err := server.ValidateLearnMessage(
		`{"type":"fact","situation":"s","subject":"x","predicate":"p"}`,
	)
	g.Expect(err).To(MatchError(ContainSubstring("object")))
}

func TestValidateLearn_MissingFieldAlwaysRejected(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Missing "action" field.
	err := server.ValidateLearnMessage(
		`{"type":"feedback","situation":"s","behavior":"b","impact":"i"}`,
	)
	g.Expect(err).To(MatchError(ContainSubstring("action")))
}

func TestValidateLearn_ValidFactAlwaysAccepted(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		text := `{"type":"fact","situation":"` +
			rapid.StringMatching(`[a-z ]{5,30}`).Draw(rt, "sit") +
			`","subject":"` +
			rapid.StringMatching(`[a-z ]{5,30}`).Draw(rt, "subj") +
			`","predicate":"` +
			rapid.StringMatching(`[a-z ]{5,30}`).Draw(rt, "pred") +
			`","object":"` +
			rapid.StringMatching(`[a-z ]{5,30}`).Draw(rt, "obj") + `"}`

		err := server.ValidateLearnMessage(text)
		g.Expect(err).NotTo(HaveOccurred())
	})
}

func TestValidateLearn_ValidFeedbackAlwaysAccepted(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		text := `{"type":"feedback","situation":"` +
			rapid.StringMatching(`[a-z ]{5,30}`).Draw(rt, "sit") +
			`","behavior":"` +
			rapid.StringMatching(`[a-z ]{5,30}`).Draw(rt, "beh") +
			`","impact":"` +
			rapid.StringMatching(`[a-z ]{5,30}`).Draw(rt, "imp") +
			`","action":"` +
			rapid.StringMatching(`[a-z ]{5,30}`).Draw(rt, "act") + `"}`

		err := server.ValidateLearnMessage(text)
		g.Expect(err).NotTo(HaveOccurred())
	})
}
