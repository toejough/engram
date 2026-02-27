package memory

import (
	"testing"

	. "github.com/onsi/gomega"
)

// TestParseJSONResponse_DirectUnmarshal verifies direct JSON parsing.
func TestParseJSONResponse_DirectUnmarshal(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	type result struct {
		Value string `json:"value"`
		Score int    `json:"score"`
	}

	var r result

	err := parseJSONResponse([]byte(`{"value":"test","score":42}`), &r)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(r.Value).To(Equal("test"))
	g.Expect(r.Score).To(Equal(42))
}

// TestParseJSONResponse_EmptyBraces verifies empty JSON object can be parsed.
func TestParseJSONResponse_EmptyBraces(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	type result struct {
		Value string `json:"value"`
	}

	var r result

	err := parseJSONResponse([]byte(`{}`), &r)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(r.Value).To(BeEmpty())
}

// TestParseJSONResponse_ExtractFromText verifies JSON extraction from surrounding text.
func TestParseJSONResponse_ExtractFromText(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	type result struct {
		Action string `json:"action"`
	}

	raw := `Here is my analysis: {"action":"ADD"} and some more text.`

	var r result

	err := parseJSONResponse([]byte(raw), &r)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(r.Action).To(Equal("ADD"))
}

// TestParseJSONResponse_InvalidJSON verifies error on completely invalid input.
func TestParseJSONResponse_InvalidJSON(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	type result struct {
		Value string `json:"value"`
	}

	var r result

	err := parseJSONResponse([]byte("not json at all, no braces"), &r)

	g.Expect(err).To(HaveOccurred())
}

// TestParseJSONResponse_MarkdownFencing verifies JSON extraction from markdown code block.
func TestParseJSONResponse_MarkdownFencing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	type result struct {
		Valid bool `json:"valid"`
	}

	raw := "```json\n{\"valid\":true}\n```"

	var r result

	err := parseJSONResponse([]byte(raw), &r)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(r.Valid).To(BeTrue())
}
