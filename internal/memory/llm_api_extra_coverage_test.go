//go:build sqlite_fts5

package memory_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/memory"
)

// TestDirectAPIExtractor_CallAPIWithMessages_APIErrorResponse verifies error field in response body is handled.
func TestDirectAPIExtractor_CallAPIWithMessages_APIErrorResponse(t *testing.T) {
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

// TestDirectAPIExtractor_CallAPIWithMessages_EmptyContent verifies empty content array returns error.
func TestDirectAPIExtractor_CallAPIWithMessages_EmptyContent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"content":[]}`))
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

// TestDirectAPIExtractor_CallAPIWithMessages_RetryAfterHeader verifies retry-after header path.
func TestDirectAPIExtractor_CallAPIWithMessages_RetryAfterHeader(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"rate limited"}}`))
	}))
	defer srv.Close()

	ext := memory.NewDirectAPIExtractor("fake-token",
		memory.WithBaseURL(srv.URL),
		memory.WithTimeout(5*time.Second),
	)

	// Use a short-lived context to abort the retry loop quickly
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	params := memory.APIMessageParams{
		Messages:  []memory.APIMessage{{Role: "user", Content: "test"}},
		MaxTokens: 100,
	}

	result, err := ext.CallAPIWithMessages(ctx, params)

	g.Expect(err).To(HaveOccurred())
	g.Expect(result).To(BeNil())
}

// TestDirectAPIExtractor_CallAPIWithMessages_RetryableError verifies 429 triggers retryable error path.
func TestDirectAPIExtractor_CallAPIWithMessages_RetryableError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"rate limited"}}`))
	}))
	defer srv.Close()

	ext := memory.NewDirectAPIExtractor("fake-token",
		memory.WithBaseURL(srv.URL),
		memory.WithTimeout(5*time.Second),
	)

	// Use a short-lived context to abort the retry loop quickly
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	params := memory.APIMessageParams{
		Messages:  []memory.APIMessage{{Role: "user", Content: "test"}},
		MaxTokens: 100,
	}

	result, err := ext.CallAPIWithMessages(ctx, params)

	g.Expect(err).To(HaveOccurred())
	g.Expect(result).To(BeNil())
}

// TestDirectAPIExtractor_PostEval_CallAPIError verifies PostEval propagates callAPI errors.
func TestDirectAPIExtractor_PostEval_CallAPIError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"unauthorized"}}`))
	}))
	defer srv.Close()

	ext := memory.NewDirectAPIExtractor("fake-token",
		memory.WithBaseURL(srv.URL),
		memory.WithTimeout(5*time.Second),
	)

	result, err := ext.PostEval(context.Background(), "always use TDD", "test query")

	g.Expect(err).To(HaveOccurred())
	g.Expect(result).To(BeNil())
}

// TestDirectAPIExtractor_PostEval_InvalidJSON verifies PostEval returns error when response is not valid PostEvalResult JSON.
func TestDirectAPIExtractor_PostEval_InvalidJSON(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Server returns text content that is valid JSON but not a PostEvalResult
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"not valid posteval json here"}]}`))
	}))
	defer srv.Close()

	ext := memory.NewDirectAPIExtractor("fake-token",
		memory.WithBaseURL(srv.URL),
		memory.WithTimeout(5*time.Second),
	)

	result, err := ext.PostEval(context.Background(), "always use TDD", "test query")

	// The response text "not valid posteval json here" cannot be unmarshalled into PostEvalResult
	g.Expect(err).To(HaveOccurred())
	g.Expect(result).To(BeNil())
}
