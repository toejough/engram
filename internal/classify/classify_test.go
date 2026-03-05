package classify_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/classify"
)

// TestClassify_BearerAuthAndBetaHeader verifies correct auth headers.
func TestClassify_BearerAuthAndBetaHeader(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	llmResp := llmClassifyResponse{Tier: ""}
	doer := newFakeDoer(t, g, llmResp)
	classifier := classify.New("my-token", doer)

	_, err := classifier.Classify(context.Background(), "hello", "")
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(doer.lastRequest).NotTo(BeNil())

	if doer.lastRequest == nil {
		return
	}

	g.Expect(doer.lastRequest.Header.Get("Authorization")).To(Equal("Bearer my-token"))
	g.Expect(doer.lastRequest.Header.Get("Anthropic-Beta")).To(Equal("oauth-2025-04-20"))
}

// TestClassify_HTTPErrorReturnsError verifies HTTP transport errors propagate.
func TestClassify_HTTPErrorReturnsError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	doer := &fakeHTTPDoer{
		err: io.ErrUnexpectedEOF,
	}
	classifier := classify.New("test-token", doer)

	result, err := classifier.Classify(context.Background(), "remember to test", "")
	g.Expect(err).To(HaveOccurred())
	g.Expect(result).To(BeNil())
}

// TestClassify_InvalidJSONReturnsError verifies invalid LLM output returns error.
func TestClassify_InvalidJSONReturnsError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	apiBody := `{"content":[{"type":"text","text":"not json at all"}]}`
	doer := &fakeHTTPDoer{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(apiBody)),
		},
	}
	classifier := classify.New("test-token", doer)

	result, err := classifier.Classify(context.Background(), "remember to test", "")
	g.Expect(err).To(HaveOccurred())
	g.Expect(result).To(BeNil())
}

// TestClassify_MarkdownFencedJSONIsParsed verifies LLM output wrapped in fences works.
func TestClassify_MarkdownFencedJSONIsParsed(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	llmJSON := `{"tier":"B","title":"Check Tests","content":"check tests",` +
		`"observation_type":"correction","concepts":["testing"],` +
		`"keywords":["test"],"principle":"Run tests",` +
		`"anti_pattern":"Skipping tests","rationale":"Quality",` +
		`"filename_summary":"check tests"}`
	fencedJSON := "```json\n" + llmJSON + "\n```"

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

	classifier := classify.New("test-token", doer)

	result, classifyErr := classifier.Classify(
		context.Background(), "check tests first", "",
	)
	g.Expect(classifyErr).NotTo(HaveOccurred())

	if classifyErr != nil {
		return
	}

	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(result.Tier).To(Equal("B"))
	g.Expect(result.Title).To(Equal("Check Tests"))
}

// TestClassify_NoTokenNonFastPathReturnsNil verifies graceful degradation.
func TestClassify_NoTokenNonFastPathReturnsNil(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	doer := &fakeHTTPDoer{}
	classifier := classify.New("", doer)

	result, err := classifier.Classify(
		context.Background(), "hello world", "",
	)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result).To(BeNil())
	g.Expect(doer.called).To(BeFalse())
}

// TestClassify_NoTokenReturnsError verifies ErrNoToken when no token configured.
func TestClassify_NoTokenReturnsError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	doer := &fakeHTTPDoer{}
	classifier := classify.New("", doer)

	result, err := classifier.Classify(context.Background(), "remember to test", "")
	g.Expect(err).To(MatchError(classify.ErrNoToken))
	g.Expect(result).To(BeNil())
	g.Expect(doer.called).To(BeFalse())
}

// TestFastPath_WholeWordBoundaries verifies keywords embedded in other words
// do not trigger fast-path (exercises isWordChar for all character classes).
func TestFastPath_WholeWordBoundaries(t *testing.T) {
	t.Parallel()

	// Messages where keyword is NOT a whole word — should go to LLM, not fast-path
	nonWholeWords := []string{
		"Xremember this",    // uppercase before 'remember'
		"9never do that",    // digit before 'never'
		"_always check",     // underscore before 'always'
		"remembering stuff", // keyword is prefix of longer word
	}

	for _, msg := range nonWholeWords {
		t.Run(msg, func(t *testing.T) {
			t.Parallel()

			g := NewGomegaWithT(t)

			// LLM returns null tier → nil result
			llmResp := llmClassifyResponse{Tier: ""}
			doer := newFakeDoer(t, g, llmResp)
			classifier := classify.New("test-token", doer)

			result, err := classifier.Classify(context.Background(), msg, "")
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(result).To(BeNil())
		})
	}
}

// T-1: Fast-path keywords trigger tier-A classification
func TestT1_FastPathKeywordsTriggerTierA(t *testing.T) {
	t.Parallel()

	keywords := []string{
		"remember to use targ",
		"Always use DI in internal",
		"NEVER delete memory files",
		"Remember: check tests first",
	}

	for _, msg := range keywords {
		t.Run(msg, func(t *testing.T) {
			t.Parallel()

			g := NewGomegaWithT(t)

			// Fast-path triggers tier A, then LLM enriches
			llmResp := llmClassifyResponse{
				Tier:            "A",
				Title:           "Use Targ for Builds",
				Content:         msg,
				ObservationType: "explicit-instruction",
				Concepts:        []string{"build-tools"},
				Keywords:        []string{"targ", "build"},
				Principle:       "Always use targ",
				AntiPattern:     "Running go test directly",
				Rationale:       "Targ encodes build conventions",
				FilenameSummary: "use targ for builds",
			}

			doer := newFakeDoer(t, g, llmResp)
			classifier := classify.New("test-token", doer)

			result, err := classifier.Classify(context.Background(), msg, "")
			g.Expect(err).NotTo(HaveOccurred())

			if err != nil {
				return
			}

			g.Expect(result).NotTo(BeNil())

			if result == nil {
				return
			}

			g.Expect(result.Tier).To(Equal("A"))
			g.Expect(result.Title).NotTo(BeEmpty())
			g.Expect(result.Content).NotTo(BeEmpty())
			g.Expect(result.Keywords).NotTo(BeEmpty())
			g.Expect(result.AntiPattern).NotTo(BeEmpty())
		})
	}
}

// T-2: Non-signal message returns nil
func TestT2_NonSignalMessageReturnsNil(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	llmResp := llmClassifyResponse{Tier: ""}
	doer := newFakeDoer(t, g, llmResp)
	classifier := classify.New("test-token", doer)

	result, err := classifier.Classify(context.Background(), "hold on", "")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result).To(BeNil())
}

// T-3: LLM classifier returns tier A (explicit instruction)
func TestT3_LLMClassifierReturnsTierA(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	llmResp := llmClassifyResponse{
		Tier:            "A",
		Title:           "Use Fish Shell",
		Content:         "Use fish shell exclusively in this project",
		ObservationType: "explicit-instruction",
		Concepts:        []string{"shell", "tooling"},
		Keywords:        []string{"fish", "shell"},
		Principle:       "Use fish shell exclusively",
		AntiPattern:     "Using bash or zsh",
		Rationale:       "Project convention",
		FilenameSummary: "use fish shell",
	}

	doer := newFakeDoer(t, g, llmResp)
	classifier := classify.New("test-token", doer)

	result, err := classifier.Classify(
		context.Background(),
		"Use fish shell exclusively in this project",
		"",
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(result.Tier).To(Equal("A"))
	g.Expect(result.AntiPattern).NotTo(BeEmpty())
}

// T-4: LLM classifier returns tier B/C with tier-gated anti-pattern
func TestT4_TierGatedAntiPattern(t *testing.T) {
	t.Parallel()

	t.Run("tier B with anti-pattern", func(t *testing.T) {
		t.Parallel()

		g := NewGomegaWithT(t)

		llmResp := llmClassifyResponse{
			Tier:            "B",
			Title:           "Check Tests Before Commit",
			Content:         "you should have checked the tests",
			ObservationType: "teachable-correction",
			Concepts:        []string{"testing"},
			Keywords:        []string{"test", "commit"},
			Principle:       "Run tests before committing",
			AntiPattern:     "Committing without running tests",
			Rationale:       "Prevents broken builds",
			FilenameSummary: "check tests before commit",
		}

		doer := newFakeDoer(t, g, llmResp)
		classifier := classify.New("test-token", doer)

		result, err := classifier.Classify(
			context.Background(),
			"you should have checked the tests",
			"",
		)
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(result).NotTo(BeNil())

		if result == nil {
			return
		}

		g.Expect(result.Tier).To(Equal("B"))
		g.Expect(result.AntiPattern).To(Equal("Committing without running tests"))
	})

	t.Run("tier C has empty anti-pattern", func(t *testing.T) {
		t.Parallel()

		g := NewGomegaWithT(t)

		llmResp := llmClassifyResponse{
			Tier:            "C",
			Title:           "Project Uses SQLite",
			Content:         "this project uses SQLite for storage",
			ObservationType: "contextual-fact",
			Concepts:        []string{"database"},
			Keywords:        []string{"sqlite", "storage"},
			Principle:       "SQLite is the storage backend",
			AntiPattern:     "",
			Rationale:       "Architectural decision",
			FilenameSummary: "project uses sqlite",
		}

		doer := newFakeDoer(t, g, llmResp)
		classifier := classify.New("test-token", doer)

		result, err := classifier.Classify(
			context.Background(),
			"this project uses SQLite for storage",
			"",
		)
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(result).NotTo(BeNil())

		if result == nil {
			return
		}

		g.Expect(result.Tier).To(Equal("C"))
		g.Expect(result.AntiPattern).To(BeEmpty())
	})
}

// T-7: Classifier includes transcript context in LLM call
func TestT7_ClassifierIncludesTranscriptContext(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	llmResp := llmClassifyResponse{
		Tier:            "B",
		Title:           "Use DI Pattern",
		Content:         "use dependency injection",
		ObservationType: "correction",
		Concepts:        []string{"di"},
		Keywords:        []string{"dependency-injection"},
		Principle:       "Use DI everywhere",
		AntiPattern:     "Direct I/O calls",
		Rationale:       "Testability",
		FilenameSummary: "use di pattern",
	}

	doer := newFakeDoer(t, g, llmResp)
	classifier := classify.New("test-token", doer)

	transcriptCtx := "Earlier in the session we discussed testing patterns..."

	result, err := classifier.Classify(
		context.Background(),
		"use dependency injection",
		transcriptCtx,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeNil())

	// Verify the transcript context was included in the request
	g.Expect(doer.lastRequest).NotTo(BeNil())

	if doer.lastRequest == nil {
		return
	}

	body, readErr := io.ReadAll(doer.lastRequest.Body)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(string(body)).To(ContainSubstring(transcriptCtx))
}

// fakeHTTPDoer is a test double for classify.HTTPDoer.
type fakeHTTPDoer struct {
	response    *http.Response
	err         error
	called      bool
	lastRequest *http.Request
}

func (f *fakeHTTPDoer) Do(req *http.Request) (*http.Response, error) {
	f.called = true

	// Save a copy of the request body for inspection
	if req.Body != nil {
		bodyBytes, _ := io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		f.lastRequest = req
		f.lastRequest.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	} else {
		f.lastRequest = req
	}

	return f.response, f.err
}

// --- Test helpers ---

// llmClassifyResponse is the JSON structure the fake LLM returns.
//
//nolint:tagliatelle // LLM prompt specifies snake_case JSON field names.
type llmClassifyResponse struct {
	Tier            string   `json:"tier"`
	Title           string   `json:"title"`
	Content         string   `json:"content"`
	ObservationType string   `json:"observation_type"`
	Concepts        []string `json:"concepts"`
	Keywords        []string `json:"keywords"`
	Principle       string   `json:"principle"`
	AntiPattern     string   `json:"anti_pattern"`
	Rationale       string   `json:"rationale"`
	FilenameSummary string   `json:"filename_summary"`
}

func mustMarshal(g Gomega, v any) string {
	b, err := json.Marshal(v)
	g.Expect(err).NotTo(HaveOccurred())

	return string(b)
}

func newFakeDoer(t *testing.T, g Gomega, resp llmClassifyResponse) *fakeHTTPDoer {
	t.Helper()

	// Handle null tier by marshaling tier as null
	var tierJSON string
	if resp.Tier == "" {
		tierJSON = "null"
	} else {
		tierBytes, err := json.Marshal(resp.Tier)
		g.Expect(err).NotTo(HaveOccurred())

		tierJSON = string(tierBytes)
	}

	// Build the inner JSON manually to support null tier
	llmJSON := `{` +
		`"tier":` + tierJSON + `,` +
		`"title":` + mustMarshal(g, resp.Title) + `,` +
		`"content":` + mustMarshal(g, resp.Content) + `,` +
		`"observation_type":` + mustMarshal(g, resp.ObservationType) + `,` +
		`"concepts":` + mustMarshal(g, resp.Concepts) + `,` +
		`"keywords":` + mustMarshal(g, resp.Keywords) + `,` +
		`"principle":` + mustMarshal(g, resp.Principle) + `,` +
		`"anti_pattern":` + mustMarshal(g, resp.AntiPattern) + `,` +
		`"rationale":` + mustMarshal(g, resp.Rationale) + `,` +
		`"filename_summary":` + mustMarshal(g, resp.FilenameSummary) +
		`}`

	apiEnvelope := map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": llmJSON},
		},
	}

	apiJSON, err := json.Marshal(apiEnvelope)
	g.Expect(err).NotTo(HaveOccurred())

	return &fakeHTTPDoer{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(apiJSON)),
		},
	}
}
