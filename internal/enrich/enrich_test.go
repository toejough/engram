package enrich_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/enrich"
	"engram/internal/memory"
)

// TestEnrichmentBodyReadErrorReturnsError verifies that an error reading
// the response body propagates as an enrichment error.
func TestEnrichmentBodyReadErrorReturnsError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	doer := &fakeHTTPDoer{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(&failingReader{}),
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

// TestEnrichmentEmptyContentBlocksReturnsError verifies that a valid API
// response with no content blocks returns an error.
func TestEnrichmentEmptyContentBlocksReturnsError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	doer := &fakeHTTPDoer{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(`{"content":[]}`)),
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

// TestEnrichmentHTTPErrorReturnsError verifies that an HTTP transport failure
// propagates as an enrichment error without panic.
func TestEnrichmentHTTPErrorReturnsError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	networkErr := errors.New("connection refused")
	doer := &fakeHTTPDoer{err: networkErr}
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
		g.Expect(err.Error()).To(ContainSubstring("Anthropic API"))
	}

	g.Expect(doer.called).To(BeTrue())
}

// TestEnrichmentInvalidLLMJSONReturnsError verifies that a valid API envelope
// containing invalid JSON in the LLM text field returns an error.
func TestEnrichmentInvalidLLMJSONReturnsError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	apiBody := `{"content":[{"type":"text","text":"not json"}]}`

	doer := &fakeHTTPDoer{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(apiBody)),
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

// TestEnrichmentMarkdownFencedJSONIsParsed verifies that LLM responses wrapped
// in markdown code fences (```json ... ```) are parsed successfully.
func TestEnrichmentMarkdownFencedJSONIsParsed(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	llmPayload := map[string]any{
		"title":            "Give Self-Consistent Responses",
		"content":          "do not contradict yourself",
		"observation_type": "correction",
		"concepts":         []string{"consistency"},
		"keywords":         []string{"summary", "contradiction"},
		"principle":        "Review findings before summarizing",
		"anti_pattern":     "Saying everything is fine then listing problems",
		"rationale":        "Contradictions waste the user's time",
		"filename_summary": "self consistent responses",
	}

	llmJSON, err := json.Marshal(llmPayload)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Wrap in markdown code fence like Haiku often does
	fencedJSON := "```json\n" + string(llmJSON) + "\n```"

	apiEnvelope := map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": fencedJSON},
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
		Pattern:    `\bno\b`,
		Label:      "correction",
		Confidence: "A",
	}

	mem, err := enricher.Enrich(context.Background(), "do not contradict yourself", match)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(mem).NotTo(BeNil())

	if mem == nil {
		return
	}

	g.Expect(mem.Title).To(Equal("Give Self-Consistent Responses"))
	g.Expect(mem.Principle).To(Equal("Review findings before summarizing"))
}

// TestEnrichmentNilResponseReturnsError verifies that a nil HTTP response
// (without transport error) returns ErrNilResponse.
func TestEnrichmentNilResponseReturnsError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	doer := &fakeHTTPDoer{response: nil, err: nil}
	enricher := enrich.New("test-api-key", doer)

	match := &memory.PatternMatch{
		Pattern:    `\bremember\s+(that|to)`,
		Label:      "reminder",
		Confidence: "A",
	}

	mem, err := enricher.Enrich(context.Background(), "remember to use targ", match)

	g.Expect(err).To(HaveOccurred())
	g.Expect(mem).To(BeNil())
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
	mem, err := enricher.Enrich(
		context.Background(),
		"remember to use targ for all build operations",
		match,
	)
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

	// Verify Bearer auth header (not X-Api-Key).
	g.Expect(doer.lastRequest).NotTo(BeNil())

	if doer.lastRequest != nil {
		g.Expect(doer.lastRequest.Header.Get("Authorization")).To(Equal("Bearer test-api-key"))
		g.Expect(doer.lastRequest.Header.Get("X-Api-Key")).To(BeEmpty())
		g.Expect(doer.lastRequest.Header.Get("Anthropic-Beta")).To(Equal("oauth-2025-04-20"))
	}
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

	g.Expect(err).To(MatchError(enrich.ErrNoToken))
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

// failingReader is an io.Reader that always returns an error.
type failingReader struct{}

func (f *failingReader) Read([]byte) (int, error) {
	return 0, errors.New("read failure")
}

// fakeHTTPDoer is a test double for enrich.HTTPDoer that returns a canned response.
type fakeHTTPDoer struct {
	response    *http.Response
	err         error
	called      bool
	lastRequest *http.Request
}

func (f *fakeHTTPDoer) Do(req *http.Request) (*http.Response, error) {
	f.called = true
	f.lastRequest = req

	return f.response, f.err
}
