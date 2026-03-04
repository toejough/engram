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
	g.Expect(mem.Degraded).To(BeFalse())
	g.Expect(mem.CreatedAt).To(BeTemporally(">=", before))
	g.Expect(mem.CreatedAt).To(BeTemporally("<=", after))
	g.Expect(mem.UpdatedAt).To(Equal(mem.CreatedAt))
}

// TestT6_EnrichmentWithoutAPIKeyProducesDegradedMemory verifies that an empty API key
// causes a degraded memory to be returned without making any HTTP call.
func TestT6_EnrichmentWithoutAPIKeyProducesDegradedMemory(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	doer := &fakeHTTPDoer{} // response is nil — would panic if called
	enricher := enrich.New("", doer)

	message := "remember to use targ for all build operations and not go test directly ever please"
	match := &memory.PatternMatch{
		Pattern:    `\bremember\s+(that|to)`,
		Label:      "reminder",
		Confidence: "A",
	}

	mem, err := enricher.Enrich(context.Background(), message, match)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(mem).NotTo(BeNil())

	if mem == nil {
		return
	}

	// No HTTP call must be made.
	g.Expect(doer.called).To(BeFalse())

	// Degraded memory has content and label from match, but no enrichment fields.
	g.Expect(mem.Content).To(Equal(message))
	g.Expect(mem.ObservationType).To(Equal("reminder"))
	g.Expect(mem.Confidence).To(Equal("A"))
	g.Expect(mem.Degraded).To(BeTrue())

	// Title is first ~60 chars, truncated at a word boundary.
	g.Expect(len(mem.Title)).To(BeNumerically("<=", 60))
	g.Expect(mem.Title).NotTo(BeEmpty())

	// Enrichment fields are empty.
	g.Expect(mem.Concepts).To(BeEmpty())
	g.Expect(mem.Keywords).To(BeEmpty())
	g.Expect(mem.Principle).To(BeEmpty())
	g.Expect(mem.AntiPattern).To(BeEmpty())
	g.Expect(mem.Rationale).To(BeEmpty())
}

// TestT7_InvalidLLMResponseFallsBackToDegraded verifies that an unparseable HTTP
// response body causes the enricher to fall back to a degraded memory.
func TestT7_InvalidLLMResponseFallsBackToDegraded(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	doer := &fakeHTTPDoer{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(`{not valid json at all}`)),
		},
	}

	enricher := enrich.New("test-api-key", doer)

	message := "remember to use targ"
	match := &memory.PatternMatch{
		Pattern:    `\bremember\s+(that|to)`,
		Label:      "reminder",
		Confidence: "A",
	}

	mem, err := enricher.Enrich(context.Background(), message, match)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(mem).NotTo(BeNil())

	if mem == nil {
		return
	}

	// Falls back to degraded: no enrichment fields, only basics.
	g.Expect(mem.Content).To(Equal(message))
	g.Expect(mem.ObservationType).To(Equal("reminder"))
	g.Expect(mem.Confidence).To(Equal("A"))
	g.Expect(mem.Degraded).To(BeTrue())
	g.Expect(mem.Concepts).To(BeEmpty())
	g.Expect(mem.Keywords).To(BeEmpty())
	g.Expect(mem.Principle).To(BeEmpty())
	g.Expect(mem.AntiPattern).To(BeEmpty())
	g.Expect(mem.Rationale).To(BeEmpty())
}
