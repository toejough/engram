package enrich_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/enrich"
	"engram/internal/memory"
)

// fakeHTTPDoer is a test double for enrich.HTTPDoer that returns a canned response.
type fakeHTTPDoer struct {
	response *http.Response
	called   bool
}

func (f *fakeHTTPDoer) Do(_ *http.Request) (*http.Response, error) {
	f.called = true
	return f.response, nil
}

// TestT5_EnrichmentWithAPIKeyProducesAllFields verifies that a valid API key and a
// well-formed LLM response result in an Enriched with all structured fields set.
func TestT5_EnrichmentWithAPIKeyProducesAllFields(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	llmPayload := map[string]any{
		"title":            "Remember to Use Targ",
		"content":          "remember to use targ for all build operations",
		"observation_type": "reminder",
		"concepts":         []string{"build-tools", "testing"},
		"keywords":         []string{"targ", "build", "test"},
		"principle":        "Use targ for all build operations",
		"anti_pattern":     "Running go test directly",
		"rationale":        "Targ provides consistent test runner behavior",
		"filename_summary": "remember use targ builds",
	}

	llmJSON, err := json.Marshal(llmPayload)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	apiEnvelope := map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": string(llmJSON)},
		},
	}

	apiJSON, err := json.Marshal(apiEnvelope)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	doer := &fakeHTTPDoer{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(apiJSON)),
		},
	}

	enricher := enrich.New("test-api-key", doer)
	match := &memory.PatternMatch{
		Pattern:    `\bremember\s+(that|to)`,
		Label:      "reminder",
		Confidence: "A",
	}

	before := time.Now()
	mem, err := enricher.Enrich(context.Background(), "remember to use targ for all build operations", match)
	after := time.Now()

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(mem).NotTo(BeNil())

	if mem == nil {
		return
	}

	g.Expect(mem.Title).To(Equal("Remember to Use Targ"))
	g.Expect(mem.Content).To(Equal("remember to use targ for all build operations"))
	g.Expect(mem.ObservationType).To(Equal("reminder"))
	g.Expect(mem.Concepts).To(ConsistOf("build-tools", "testing"))
	g.Expect(mem.Keywords).To(ConsistOf("targ", "build", "test"))
	g.Expect(mem.Principle).To(Equal("Use targ for all build operations"))
	g.Expect(mem.AntiPattern).To(Equal("Running go test directly"))
	g.Expect(mem.Rationale).To(Equal("Targ provides consistent test runner behavior"))
	g.Expect(mem.FilenameSummary).To(Equal("remember use targ builds"))
	g.Expect(mem.Confidence).To(Equal("A"))
	g.Expect(mem.CreatedAt).To(BeTemporally(">=", before))
	g.Expect(mem.CreatedAt).To(BeTemporally("<=", after))
	g.Expect(mem.UpdatedAt).To(Equal(mem.CreatedAt))
}

// TestT6_EnrichmentWithoutAPIKeyReturnsError verifies that an empty API key
// causes ErrNoAPIKey to be returned without making any HTTP call.
func TestT6_EnrichmentWithoutAPIKeyReturnsError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	doer := &fakeHTTPDoer{} // response is nil — would panic if called
	enricher := enrich.New("", doer)

	match := &memory.PatternMatch{
		Pattern:    `\bremember\s+(that|to)`,
		Label:      "reminder",
		Confidence: "A",
	}

	mem, err := enricher.Enrich(context.Background(), "remember to use targ", match)

	g.Expect(err).To(MatchError(enrich.ErrNoAPIKey))
	g.Expect(mem).To(BeNil())

	// No HTTP call must be made.
	g.Expect(doer.called).To(BeFalse())
}

// TestT7_InvalidLLMResponseReturnsError verifies that an unparseable HTTP
// response body causes the enricher to return an error.
func TestT7_InvalidLLMResponseReturnsError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	doer := &fakeHTTPDoer{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(`{not valid json at all}`)),
		},
	}

	enricher := enrich.New("test-api-key", doer)

	match := &memory.PatternMatch{
		Pattern:    `\bremember\s+(that|to)`,
		Label:      "reminder",
		Confidence: "A",
	}

	mem, err := enricher.Enrich(context.Background(), "remember to use targ", match)

	g.Expect(err).To(HaveOccurred())
	g.Expect(mem).To(BeNil())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("enrichment"))
	}
}
