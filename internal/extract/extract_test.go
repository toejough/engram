package extract_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/extract"
)

// TestGeneralizabilityFieldIsParsed verifies that a JSON response containing
// "generalizability": 4 is correctly mapped to CandidateLearning.Generalizability.
func TestGeneralizabilityFieldIsParsed(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	llmArray := []map[string]any{
		{
			"tier":             "B",
			"title":            "Generalizability Test Learning",
			"content":          "content for generalizability test",
			"observation_type": "correction",
			"concepts":         []string{"generalizability"},
			"keywords":         []string{"generalizability", "scoring"},
			"principle":        "Score memories by generalizability",
			"anti_pattern":     "",
			"rationale":        "Filters out session-specific noise",
			"filename_summary": "generalizability score test",
			"generalizability": 4,
		},
	}

	llmJSON, err := json.Marshal(llmArray)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	apiJSON := buildAPIResponse(t, g, string(llmJSON))

	doer := &fakeHTTPDoer{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(apiJSON)),
		},
	}

	extractor := extract.New("test-api-key", doer)

	learnings, err := extractor.Extract(context.Background(), "some transcript")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(learnings).To(HaveLen(1))
	g.Expect(learnings[0].Generalizability).To(Equal(4))
}

// TestMarkdownFencedJSONArrayIsParsed verifies that LLM responses wrapped in markdown
// code fences (```json ... ```) are parsed successfully as a JSON array.
func TestMarkdownFencedJSONArrayIsParsed(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	llmArray := []map[string]any{
		{
			"title":            "Fence Test Learning",
			"content":          "content from fenced response",
			"observation_type": "constraint",
			"concepts":         []string{"fencing"},
			"keywords":         []string{"fence", "json"},
			"principle":        "Strip fences before parsing",
			"anti_pattern":     "Failing on fenced JSON",
			"rationale":        "LLMs often add markdown fences",
			"filename_summary": "fence test",
		},
	}

	llmJSON, err := json.Marshal(llmArray)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Wrap in markdown code fence like Haiku often does.
	fencedJSON := "```json\n" + string(llmJSON) + "\n```"

	apiJSON := buildAPIResponse(t, g, fencedJSON)

	doer := &fakeHTTPDoer{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(apiJSON)),
		},
	}

	extractor := extract.New("test-api-key", doer)

	learnings, err := extractor.Extract(context.Background(), "some transcript")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(learnings).NotTo(BeNil())

	if learnings == nil {
		return
	}

	g.Expect(learnings).To(HaveLen(1))
	g.Expect(learnings[0].Title).To(Equal("Fence Test Learning"))
	g.Expect(learnings[0].Principle).To(Equal("Strip fences before parsing"))
}

// TestNilResponseReturnsError verifies that a nil HTTP response without a transport
// error causes the extractor to return an error.
func TestNilResponseReturnsError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	doer := &fakeHTTPDoer{response: nil, err: nil}
	extractor := extract.New("test-api-key", doer)

	learnings, err := extractor.Extract(context.Background(), "some transcript")

	g.Expect(err).To(HaveOccurred())
	g.Expect(learnings).To(BeNil())
}

// TestSystemPromptIncludesTierDefinitions verifies that the system prompt
// contains A/B/C tier definitions and anti-pattern gating rules.
func TestSystemPromptIncludesTierDefinitions(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	doer := &fakeHTTPDoer{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(
				bytes.NewBufferString(`{"content":[{"type":"text","text":"[]"}]}`),
			),
		},
	}

	extractor := extract.New("test-api-key", doer)

	_, err := extractor.Extract(context.Background(), "some transcript")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(doer.lastRequest).NotTo(BeNil())

	if doer.lastRequest == nil {
		return
	}

	reqBody, readErr := io.ReadAll(doer.lastRequest.Body)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	reqBodyStr := string(reqBody)

	// System prompt must include tier definitions.
	g.Expect(reqBodyStr).To(ContainSubstring("tier"))
	g.Expect(reqBodyStr).To(ContainSubstring("explicit instruction"))
	g.Expect(reqBodyStr).To(ContainSubstring("teachable correction"))
	g.Expect(reqBodyStr).To(ContainSubstring("contextual fact"))

	// System prompt must include anti-pattern gating rules.
	g.Expect(reqBodyStr).To(ContainSubstring("anti_pattern"))
}

// TestT47_ExtractionWithTokenProducesCandidateLearnings verifies that a valid API
// token and well-formed LLM response produce CandidateLearnings with all fields set.
func TestT47_ExtractionWithTokenProducesCandidateLearnings(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	llmArray := []map[string]any{
		{
			"title":            "Use targ for all build operations",
			"content":          "Always run targ test instead of go test directly",
			"observation_type": "reminder",
			"concepts":         []string{"build-tools", "testing"},
			"keywords":         []string{"targ", "build", "test"},
			"principle":        "Use targ for consistent test runner behavior",
			"anti_pattern":     "Running go test directly",
			"rationale":        "Targ encodes hard-won lessons about test configuration",
			"filename_summary": "use targ builds",
			"tier":             "A",
		},
	}

	llmJSON, err := json.Marshal(llmArray)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	apiJSON := buildAPIResponse(t, g, string(llmJSON))

	doer := &fakeHTTPDoer{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(apiJSON)),
		},
	}

	extractor := extract.New("test-api-key", doer)

	learnings, err := extractor.Extract(
		context.Background(),
		"remember to use targ for all build operations",
	)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(learnings).NotTo(BeNil())

	if learnings == nil {
		return
	}

	g.Expect(learnings).To(HaveLen(1))

	learning := learnings[0]
	g.Expect(learning.Title).To(Equal("Use targ for all build operations"))
	g.Expect(learning.Content).To(Equal("Always run targ test instead of go test directly"))
	g.Expect(learning.ObservationType).To(Equal("reminder"))
	g.Expect(learning.Concepts).To(ConsistOf("build-tools", "testing"))
	g.Expect(learning.Keywords).To(ConsistOf("targ", "build", "test"))
	g.Expect(learning.Principle).To(Equal("Use targ for consistent test runner behavior"))
	g.Expect(learning.AntiPattern).To(Equal("Running go test directly"))
	g.Expect(learning.Rationale).To(Equal("Targ encodes hard-won lessons about test configuration"))
	g.Expect(learning.FilenameSummary).To(Equal("use targ builds"))
	g.Expect(learning.Tier).To(Equal("A"))

	// Verify Bearer auth header.
	g.Expect(doer.lastRequest).NotTo(BeNil())

	if doer.lastRequest != nil {
		g.Expect(doer.lastRequest.Header.Get("Authorization")).To(Equal("Bearer test-api-key"))
		g.Expect(doer.lastRequest.Header.Get("X-Api-Key")).To(BeEmpty())
		g.Expect(doer.lastRequest.Header.Get("Anthropic-Beta")).To(Equal("oauth-2025-04-20"))
	}
}

// TestT48_ExtractionWithoutTokenReturnsErrNoToken verifies that an empty API token
// causes ErrNoToken to be returned without making any HTTP call.
func TestT48_ExtractionWithoutTokenReturnsErrNoToken(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	doer := &fakeHTTPDoer{} // response is nil — would panic if called
	extractor := extract.New("", doer)

	learnings, err := extractor.Extract(context.Background(), "some transcript")

	g.Expect(err).To(MatchError(extract.ErrNoToken))
	g.Expect(learnings).To(BeNil())

	// No HTTP call must be made.
	g.Expect(doer.called).To(BeFalse())
}

// TestT49_InvalidLLMResponseReturnsError verifies that an unparseable HTTP
// response body causes the extractor to return an error.
func TestT49_InvalidLLMResponseReturnsError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	doer := &fakeHTTPDoer{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(`{not valid json at all}`)),
		},
	}

	extractor := extract.New("test-api-key", doer)

	learnings, err := extractor.Extract(context.Background(), "some transcript")

	g.Expect(err).To(HaveOccurred())
	g.Expect(learnings).To(BeNil())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("extraction"))
	}
}

// TestT50_EmptyExtractionReturnsEmptySlice verifies that a valid API response
// with an empty JSON array returns an empty (non-nil) slice.
func TestT50_EmptyExtractionReturnsEmptySlice(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	apiJSON := buildAPIResponse(t, g, "[]")

	doer := &fakeHTTPDoer{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(apiJSON)),
		},
	}

	extractor := extract.New("test-api-key", doer)

	learnings, err := extractor.Extract(context.Background(), "some transcript with no learnings")

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(learnings).NotTo(BeNil())
	g.Expect(learnings).To(BeEmpty())
}

// TestT51_QualityGateInSystemPrompt verifies that the system prompt contains
// explicit quality gate instructions for both what to extract and what to reject.
func TestT51_QualityGateInSystemPrompt(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	doer := &fakeHTTPDoer{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(
				bytes.NewBufferString(`{"content":[{"type":"text","text":"[]"}]}`),
			),
		},
	}

	extractor := extract.New("test-api-key", doer)

	_, err := extractor.Extract(context.Background(), "some transcript")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(doer.lastRequest).NotTo(BeNil())

	if doer.lastRequest == nil {
		return
	}

	// Read the request body to inspect the system prompt.
	reqBody, readErr := io.ReadAll(doer.lastRequest.Body)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	reqBodyStr := string(reqBody)

	// T-51: System prompt must reject mechanical patterns.
	g.Expect(reqBodyStr).To(ContainSubstring("mechanical"))

	// T-51: System prompt must reject vague generalizations.
	g.Expect(reqBodyStr).To(ContainSubstring("vague"))

	// T-51: System prompt must reject overly narrow observations.
	g.Expect(reqBodyStr).To(ContainSubstring("narrow"))

	// T-51: System prompt must mention extracting missed corrections.
	g.Expect(reqBodyStr).To(ContainSubstring("missed correction"))

	// T-51: System prompt must mention architectural decisions.
	g.Expect(reqBodyStr).To(ContainSubstring("architectural"))

	// T-51: System prompt must mention discovered constraints.
	g.Expect(reqBodyStr).To(ContainSubstring("constraint"))

	// T-51: System prompt must mention working solutions.
	g.Expect(reqBodyStr).To(ContainSubstring("working solution"))

	// T-51: System prompt must mention implicit preferences.
	g.Expect(reqBodyStr).To(ContainSubstring("implicit preference"))

	// T-51: System prompt must define JSON array output format.
	g.Expect(reqBodyStr).To(ContainSubstring("JSON array"))
}

// TestTierGatedAntiPattern verifies that tier A always has anti_pattern,
// and tier C has empty anti_pattern per ARCH-15 anti-pattern gating rules.
func TestTierGatedAntiPattern(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	llmArray := []map[string]any{
		{
			"title":            "Always use DI",
			"content":          "explicit instruction to use DI",
			"observation_type": "correction",
			"concepts":         []string{"di"},
			"keywords":         []string{"di", "inject"},
			"principle":        "Use dependency injection",
			"anti_pattern":     "Direct I/O in internal/",
			"rationale":        "Testability",
			"filename_summary": "use di everywhere",
			"tier":             "A",
		},
		{
			"title":            "Project uses SQLite",
			"content":          "contextual fact about database",
			"observation_type": "constraint",
			"concepts":         []string{"database"},
			"keywords":         []string{"sqlite"},
			"principle":        "Use SQLite for storage",
			"anti_pattern":     "",
			"rationale":        "Embedded database",
			"filename_summary": "project uses sqlite",
			"tier":             "C",
		},
	}

	llmJSON, err := json.Marshal(llmArray)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	apiJSON := buildAPIResponse(t, g, string(llmJSON))

	doer := &fakeHTTPDoer{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(apiJSON)),
		},
	}

	extractor := extract.New("test-api-key", doer)
	learnings, extractErr := extractor.Extract(context.Background(), "some transcript")

	g.Expect(extractErr).NotTo(HaveOccurred())

	if extractErr != nil {
		return
	}

	g.Expect(learnings).To(HaveLen(2))

	// Tier A always has anti_pattern.
	g.Expect(learnings[0].Tier).To(Equal("A"))
	g.Expect(learnings[0].AntiPattern).NotTo(BeEmpty())

	// Tier C has empty anti_pattern.
	g.Expect(learnings[1].Tier).To(Equal("C"))
	g.Expect(learnings[1].AntiPattern).To(BeEmpty())
}

// fakeHTTPDoer is a test double for extract.HTTPDoer that returns a canned response.
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

// buildAPIResponse assembles a canned Anthropic API response envelope with the given LLM text.
func buildAPIResponse(t *testing.T, g *WithT, llmText string) []byte {
	t.Helper()

	apiEnvelope := map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": llmText},
		},
	}

	apiJSON, err := json.Marshal(apiEnvelope)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return nil
	}

	return apiJSON
}
