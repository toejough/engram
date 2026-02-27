package memory_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/memory"
)

// TestDirectAPIExtractor_CallAPIWithMessages_AuthError verifies 401 triggers non-retryable error.
func TestDirectAPIExtractor_CallAPIWithMessages_AuthError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	srv := buildMockAPIServer(t, http.StatusUnauthorized, map[string]any{
		"error": map[string]any{"message": "unauthorized"},
	})
	defer srv.Close()

	ext := memory.NewDirectAPIExtractor("fake-token",
		memory.WithBaseURL(srv.URL),
		memory.WithTimeout(5*time.Second),
	)

	params := memory.APIMessageParams{
		Messages:  []memory.APIMessage{{Role: "user", Content: "hello"}},
		MaxTokens: 100,
	}

	result, err := ext.CallAPIWithMessages(context.Background(), params)

	g.Expect(err).To(HaveOccurred())
	g.Expect(result).To(BeNil())
}

// TestDirectAPIExtractor_CallAPIWithMessages_ContextCancelled verifies cancelled context returns error.
func TestDirectAPIExtractor_CallAPIWithMessages_ContextCancelled(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	srv := buildSuccessAPIServer(t, "response")
	defer srv.Close()

	ext := memory.NewDirectAPIExtractor("fake-token",
		memory.WithBaseURL(srv.URL),
		memory.WithTimeout(5*time.Second),
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancel

	params := memory.APIMessageParams{
		Messages:  []memory.APIMessage{{Role: "user", Content: "test"}},
		MaxTokens: 100,
	}

	result, err := ext.CallAPIWithMessages(ctx, params)

	g.Expect(err).To(HaveOccurred())
	g.Expect(result).To(BeNil())
}

// TestDirectAPIExtractor_CallAPIWithMessages_ForbiddenError verifies 403 triggers non-retryable error.
func TestDirectAPIExtractor_CallAPIWithMessages_ForbiddenError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	srv := buildMockAPIServer(t, http.StatusForbidden, map[string]any{
		"error": map[string]any{"message": "forbidden"},
	})
	defer srv.Close()

	ext := memory.NewDirectAPIExtractor("fake-token",
		memory.WithBaseURL(srv.URL),
		memory.WithTimeout(5*time.Second),
	)

	params := memory.APIMessageParams{
		Messages:  []memory.APIMessage{{Role: "user", Content: "test"}},
		MaxTokens: 100,
	}

	result, err := ext.CallAPIWithMessages(context.Background(), params)

	g.Expect(err).To(HaveOccurred())
	g.Expect(result).To(BeNil())
}

// TestDirectAPIExtractor_CallAPIWithMessages_Success verifies successful API call returns content.
func TestDirectAPIExtractor_CallAPIWithMessages_Success(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	srv := buildSuccessAPIServer(t, "test response text")
	defer srv.Close()

	ext := memory.NewDirectAPIExtractor("fake-token",
		memory.WithBaseURL(srv.URL),
		memory.WithTimeout(5*time.Second),
	)

	params := memory.APIMessageParams{
		System:    "You are helpful.",
		Messages:  []memory.APIMessage{{Role: "user", Content: "hello"}},
		MaxTokens: 100,
		Model:     "claude-haiku-4-5-20251001",
	}

	result, err := ext.CallAPIWithMessages(context.Background(), params)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(string(result)).To(ContainSubstring("test response text"))
}

// TestDirectAPIExtractor_ExtractBatch_Empty verifies ExtractBatch returns nil for empty input.
func TestDirectAPIExtractor_ExtractBatch_Empty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ext := memory.NewDirectAPIExtractor("fake-token",
		memory.WithTimeout(5*time.Second),
	)

	results, err := ext.ExtractBatch(context.Background(), nil)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).To(BeNil())
}

// TestDirectAPIExtractor_ExtractBatch_MultipleWithMockServer verifies batch extraction.
func TestDirectAPIExtractor_ExtractBatch_MultipleWithMockServer(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	batchResp := `[{"type":"correction","concepts":["tdd"],"principle":"Always test first","anti_pattern":"skip","rationale":"quality"},{"type":"pattern","concepts":["git"],"principle":"Always commit often","anti_pattern":"large commits","rationale":"history"}]`
	srv := buildSuccessAPIServer(t, batchResp)

	defer srv.Close()

	ext := memory.NewDirectAPIExtractor("fake-token",
		memory.WithBaseURL(srv.URL),
		memory.WithTimeout(5*time.Second),
	)

	contents := []string{"test first", "commit often"}

	results, err := ext.ExtractBatch(context.Background(), contents)

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).To(HaveLen(2))
}

// TestDirectAPIExtractor_ExtractBatch_SingleWithMockServer verifies ExtractBatch single item path.
func TestDirectAPIExtractor_ExtractBatch_SingleWithMockServer(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	obs := `{"type":"correction","concepts":["testing"],"principle":"Always test","anti_pattern":"skip tests","rationale":"quality"}`
	srv := buildSuccessAPIServer(t, obs)

	defer srv.Close()

	ext := memory.NewDirectAPIExtractor("fake-token",
		memory.WithBaseURL(srv.URL),
		memory.WithTimeout(5*time.Second),
	)

	results, err := ext.ExtractBatch(context.Background(), []string{"always run tests before committing"})

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).ToNot(BeNil())
	g.Expect(results).To(HaveLen(1))

	if len(results) > 0 && results[0] != nil {
		g.Expect(results[0].Type).To(Equal("correction"))
	}
}

// TestDirectAPIExtractor_PostEval_WithMockServer verifies PostEval parses the API response.
func TestDirectAPIExtractor_PostEval_WithMockServer(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	srv := buildSuccessAPIServer(t, `{"faithfulness":0.9,"signal":"positive"}`)
	defer srv.Close()

	ext := memory.NewDirectAPIExtractor("fake-token",
		memory.WithBaseURL(srv.URL),
		memory.WithTimeout(5*time.Second),
	)

	result, err := ext.PostEval(context.Background(), "always use TDD", "help me write tests")

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())

	if result == nil {
		t.Fatal("result is nil")
	}

	g.Expect(result.Faithfulness).To(BeNumerically("~", 0.9, 0.001))
	g.Expect(result.Signal).To(Equal("positive"))
}

// TestDirectAPIExtractor_doAPICall_APIError_NoTag covers lines 534-537 (result.Error != nil).
// "invalid_request_error" is non-retryable → exits immediately, no backoff wait.
func TestDirectAPIExtractor_doAPICall_APIError_NoTag(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"error":{"type":"invalid_request_error","message":"bad request"}}`))
	}))
	defer srv.Close()

	ext := memory.NewDirectAPIExtractor("fake-token",
		memory.WithBaseURL(srv.URL),
		memory.WithTimeout(5*time.Second),
	)

	params := memory.APIMessageParams{
		Messages:  []memory.APIMessage{{Role: "user", Content: "test"}},
		MaxTokens: 100,
	}

	result, err := ext.CallAPIWithMessages(context.Background(), params)

	g.Expect(err).To(HaveOccurred())
	g.Expect(result).To(BeNil())
}

// TestDirectAPIExtractor_doAPICall_EmptyContent_NoTag covers lines 539-541 (empty content).
// Uses a channel-based server: first request returns 200+empty-content (retryable=true),
// second request returns 401 (non-retryable) so the retry loop exits after ~1s backoff.
func TestDirectAPIExtractor_doAPICall_EmptyContent_NoTag(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	responses := make(chan func(http.ResponseWriter), 2)
	responses <- func(w http.ResponseWriter) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"content":[]}`))
	}

	responses <- func(w http.ResponseWriter) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"done"}}`))
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		handle := <-responses
		handle(w)
	}))
	defer srv.Close()

	ext := memory.NewDirectAPIExtractor("fake-token",
		memory.WithBaseURL(srv.URL),
		memory.WithTimeout(5*time.Second),
	)

	params := memory.APIMessageParams{
		Messages:  []memory.APIMessage{{Role: "user", Content: "test"}},
		MaxTokens: 100,
	}

	result, err := ext.CallAPIWithMessages(context.Background(), params)

	g.Expect(err).To(HaveOccurred())
	g.Expect(result).To(BeNil())
}

// TestDirectAPIExtractor_doAPICall_JSONDecodeError_NoTag covers lines 530-532 (JSON decode error).
// Uses a channel-based server: first request returns 200+invalid-JSON (retryable=true),
// second request returns 401 (non-retryable) so the retry loop exits after ~1s backoff.
func TestDirectAPIExtractor_doAPICall_JSONDecodeError_NoTag(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	responses := make(chan func(http.ResponseWriter), 2)
	responses <- func(w http.ResponseWriter) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not valid json content at all`))
	}

	responses <- func(w http.ResponseWriter) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"done"}}`))
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		handle := <-responses
		handle(w)
	}))
	defer srv.Close()

	ext := memory.NewDirectAPIExtractor("fake-token",
		memory.WithBaseURL(srv.URL),
		memory.WithTimeout(5*time.Second),
	)

	params := memory.APIMessageParams{
		Messages:  []memory.APIMessage{{Role: "user", Content: "test"}},
		MaxTokens: 100,
	}

	result, err := ext.CallAPIWithMessages(context.Background(), params)

	g.Expect(err).To(HaveOccurred())
	g.Expect(result).To(BeNil())
}

// TestDirectAPIExtractor_doAPICall_RetryAfterHeader_NoTag covers lines 514-516 (Retry-After header).
// Uses a channel-based server: first request returns 429+Retry-After (retryable),
// second request returns 401 (non-retryable) so the retry loop exits after ~1s backoff.
func TestDirectAPIExtractor_doAPICall_RetryAfterHeader_NoTag(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	responses := make(chan func(http.ResponseWriter), 2)
	responses <- func(w http.ResponseWriter) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	}

	responses <- func(w http.ResponseWriter) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"done"}}`))
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		handle := <-responses
		handle(w)
	}))
	defer srv.Close()

	ext := memory.NewDirectAPIExtractor("fake-token",
		memory.WithBaseURL(srv.URL),
		memory.WithTimeout(5*time.Second),
	)

	params := memory.APIMessageParams{
		Messages:  []memory.APIMessage{{Role: "user", Content: "test"}},
		MaxTokens: 100,
	}

	result, err := ext.CallAPIWithMessages(context.Background(), params)

	g.Expect(err).To(HaveOccurred())
	g.Expect(result).To(BeNil())
}

// TestDirectAPIExtractor_doAPICall_RetryableStatus_NoTag covers lines 510-519 (retryable status).
// Uses a channel-based server: first request returns 429 (retryable),
// second request returns 401 (non-retryable) so the retry loop exits after ~1s backoff.
func TestDirectAPIExtractor_doAPICall_RetryableStatus_NoTag(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	responses := make(chan func(http.ResponseWriter), 2)
	responses <- func(w http.ResponseWriter) {
		w.WriteHeader(http.StatusTooManyRequests)
	}

	responses <- func(w http.ResponseWriter) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"done"}}`))
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		handle := <-responses
		handle(w)
	}))
	defer srv.Close()

	ext := memory.NewDirectAPIExtractor("fake-token",
		memory.WithBaseURL(srv.URL),
		memory.WithTimeout(5*time.Second),
	)

	params := memory.APIMessageParams{
		Messages:  []memory.APIMessage{{Role: "user", Content: "test"}},
		MaxTokens: 100,
	}

	result, err := ext.CallAPIWithMessages(context.Background(), params)

	g.Expect(err).To(HaveOccurred())
	g.Expect(result).To(BeNil())
}

// buildMockAPIServer creates a test HTTP server returning the given JSON body.
func buildMockAPIServer(t *testing.T, statusCode int, body map[string]any) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)

		_ = json.NewEncoder(w).Encode(body)
	}))
}

// buildSuccessAPIServer creates a mock API server returning a valid text response.
func buildSuccessAPIServer(t *testing.T, text string) *httptest.Server {
	t.Helper()

	return buildMockAPIServer(t, http.StatusOK, map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": text},
		},
	})
}
