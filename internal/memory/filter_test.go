package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/onsi/gomega"
)

func TestFilterEmptyCandidates(t *testing.T) {
	g := NewWithT(t)

	// Server should NOT be called for empty candidates
	serverCalled := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverCalled = true

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	extractor := NewDirectAPIExtractor("test-token", WithBaseURL(server.URL))

	results, err := extractor.Filter(context.Background(), "test query", nil)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).To(BeEmpty())
	g.Expect(serverCalled).To(BeFalse(), "API should not be called for empty candidates")
}

func TestFilterGracefulDegradation(t *testing.T) {
	g := NewWithT(t)

	// Server returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Internal Server Error")
	}))
	defer server.Close()

	extractor := NewDirectAPIExtractor("test-token", WithBaseURL(server.URL))

	candidates := []QueryResult{
		{ID: 1, Content: "memory one", Score: 0.9, MemoryType: "correction"},
		{ID: 2, Content: "memory two", Score: 0.7, MemoryType: ""},
	}

	results, err := extractor.Filter(context.Background(), "test query", candidates)
	g.Expect(err).ToNot(HaveOccurred(), "filter should not return error on API failure")
	g.Expect(results).To(HaveLen(2))

	// All candidates returned as relevant with degradation sentinel
	for _, r := range results {
		g.Expect(r.Relevant).To(BeTrue())
		g.Expect(r.RelevanceScore).To(Equal(-1.0), "degradation sentinel")
	}
}

func TestFilterMalformedJSON(t *testing.T) {
	g := NewWithT(t)

	// Server returns non-JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]any{
			"id":    "msg_test",
			"type":  "message",
			"model": "claude-haiku-4-5-20251001",
			"content": []map[string]any{
				{"type": "text", "text": "this is not valid JSON"},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	extractor := NewDirectAPIExtractor("test-token", WithBaseURL(server.URL))

	candidates := []QueryResult{
		{ID: 1, Content: "memory one", Score: 0.9},
	}

	results, err := extractor.Filter(context.Background(), "test query", candidates)
	g.Expect(err).ToNot(HaveOccurred(), "filter should degrade on malformed JSON")

	if len(results) < 1 {
		t.Fatal("expected at least 1 result from Filter")
	}

	g.Expect(results).To(HaveLen(1))
	g.Expect(results[0].Relevant).To(BeTrue())
	g.Expect(results[0].RelevanceScore).To(Equal(-1.0))
}

func TestFilterReturnsStructuredResults(t *testing.T) {
	g := NewWithT(t)

	// Mock Haiku server returning structured filter results
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]any{
			"id":    "msg_test",
			"type":  "message",
			"model": "claude-haiku-4-5-20251001",
			"content": []map[string]any{
				{
					"type": "text",
					"text": `[
						{"memory_id": 1, "relevant": true, "tag": "relevant", "relevance_score": 0.95, "should_synthesize": false},
						{"memory_id": 2, "relevant": false, "tag": "noise", "relevance_score": 0.15, "should_synthesize": false},
						{"memory_id": 3, "relevant": true, "tag": "should-be-hook", "relevance_score": 0.80, "should_synthesize": false},
						{"memory_id": 4, "relevant": false, "tag": "should-be-earlier", "relevance_score": 0.30, "should_synthesize": true}
					]`,
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	extractor := NewDirectAPIExtractor("test-token", WithBaseURL(server.URL))

	candidates := []QueryResult{
		{ID: 1, Content: "Use AI-Used trailer for commits", Score: 0.9, MemoryType: "correction"},
		{ID: 2, Content: "I like coffee", Score: 0.5, MemoryType: ""},
		{ID: 3, Content: "Always format commit messages conventionally", Score: 0.85, MemoryType: "correction"},
		{ID: 4, Content: "Check git status before destructive ops", Score: 0.6, MemoryType: "pattern"},
	}

	results, err := extractor.Filter(context.Background(), "create a git commit", candidates)
	g.Expect(err).ToNot(HaveOccurred())

	if len(results) < 4 {
		t.Fatalf("expected 4 results from Filter, got %d", len(results))
	}

	g.Expect(results).To(HaveLen(4)) // All candidates returned (both kept and filtered)

	// Check tags
	g.Expect(results[0].Relevant).To(BeTrue())
	g.Expect(results[0].Tag).To(Equal("relevant"))
	g.Expect(results[0].RelevanceScore).To(BeNumerically("~", 0.95, 0.01))

	g.Expect(results[1].Relevant).To(BeFalse())
	g.Expect(results[1].Tag).To(Equal("noise"))

	g.Expect(results[2].Tag).To(Equal("should-be-hook"))

	g.Expect(results[3].Tag).To(Equal("should-be-earlier"))
	g.Expect(results[3].ShouldSynthesize).To(BeTrue())
}

func TestFilterShouldSynthesizePerCandidate(t *testing.T) {
	g := NewWithT(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]any{
			"id":    "msg_test",
			"type":  "message",
			"model": "claude-haiku-4-5-20251001",
			"content": []map[string]any{
				{
					"type": "text",
					"text": `[
						{"memory_id": 1, "relevant": true, "tag": "relevant", "relevance_score": 0.9, "should_synthesize": true},
						{"memory_id": 2, "relevant": true, "tag": "relevant", "relevance_score": 0.85, "should_synthesize": true},
						{"memory_id": 3, "relevant": true, "tag": "relevant", "relevance_score": 0.7, "should_synthesize": false}
					]`,
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	extractor := NewDirectAPIExtractor("test-token", WithBaseURL(server.URL))

	candidates := []QueryResult{
		{ID: 1, Content: "commit trailer format", Score: 0.9},
		{ID: 2, Content: "commit message style", Score: 0.85},
		{ID: 3, Content: "unrelated memory", Score: 0.7},
	}

	results, err := extractor.Filter(context.Background(), "create a commit", candidates)
	g.Expect(err).ToNot(HaveOccurred())

	if len(results) < 3 {
		t.Fatalf("expected 3 results from Filter, got %d", len(results))
	}

	g.Expect(results[0].ShouldSynthesize).To(BeTrue())
	g.Expect(results[1].ShouldSynthesize).To(BeTrue())
	g.Expect(results[2].ShouldSynthesize).To(BeFalse())
}
