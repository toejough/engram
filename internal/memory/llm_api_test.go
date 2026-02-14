package memory_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/memory"
)

func TestDirectAPIExtractor_Extract_ReturnsObservation(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	obs := memory.Observation{
		Type:      "correction",
		Concepts:  []string{"git"},
		Principle: "Never amend pushed commits",
	}
	obsJSON, _ := json.Marshal(obs)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		g.Expect(r.Header.Get("Authorization")).To(Equal("Bearer test-token"))
		g.Expect(r.Header.Get("anthropic-beta")).To(Equal("oauth-2025-04-20"))
		g.Expect(r.Header.Get("anthropic-version")).To(Equal("2023-06-01"))

		// Verify body
		var body struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		g.Expect(body.Messages).To(HaveLen(1))

		// Return content block with the JSON observation
		resp := map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": string(obsJSON)},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	extractor := memory.NewDirectAPIExtractor("test-token",
		memory.WithBaseURL(srv.URL),
	)

	result, err := extractor.Extract(context.Background(), "test content")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Type).To(Equal("correction"))
	g.Expect(result.Principle).To(Equal("Never amend pushed commits"))
}

func TestDirectAPIExtractor_Extract_ReturnsErrLLMUnavailableOn401(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"message": "invalid token"},
		})
	}))
	defer srv.Close()

	extractor := memory.NewDirectAPIExtractor("bad-token",
		memory.WithBaseURL(srv.URL),
	)

	result, err := extractor.Extract(context.Background(), "content")
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, memory.ErrLLMUnavailable)).To(BeTrue())
	g.Expect(result).To(BeNil())
}

func TestDirectAPIExtractor_Extract_ReturnsErrOnBadJSON(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": "not valid json"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	extractor := memory.NewDirectAPIExtractor("token",
		memory.WithBaseURL(srv.URL),
	)

	result, err := extractor.Extract(context.Background(), "content")
	g.Expect(err).To(HaveOccurred())
	g.Expect(result).To(BeNil())
}

func TestDirectAPIExtractor_Extract_RespectsContextCancellation(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done() // block until cancelled
	}))
	defer srv.Close()

	extractor := memory.NewDirectAPIExtractor("token",
		memory.WithBaseURL(srv.URL),
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := extractor.Extract(ctx, "content")
	g.Expect(err).To(HaveOccurred())
}

func TestDirectAPIExtractor_Extract_SendsCorrectModel(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var capturedModel string
	obs := memory.Observation{Type: "pattern", Concepts: []string{"x"}, Principle: "p"}
	obsJSON, _ := json.Marshal(obs)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Model string `json:"model"`
		}
		bodyBytes, _ := io.ReadAll(r.Body)
		json.Unmarshal(bodyBytes, &body)
		capturedModel = body.Model
		resp := map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": string(obsJSON)},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	extractor := memory.NewDirectAPIExtractor("token",
		memory.WithBaseURL(srv.URL),
		memory.WithModel("claude-haiku-4-5-20251001"),
	)

	_, err := extractor.Extract(context.Background(), "content")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(capturedModel).To(Equal("claude-haiku-4-5-20251001"))
}

func TestDirectAPIExtractor_ImplementsLLMExtractor(t *testing.T) {
	t.Parallel()

	var _ memory.LLMExtractor = &memory.DirectAPIExtractor{}
}

func TestDirectAPIExtractor_ImplementsSkillCompiler(t *testing.T) {
	t.Parallel()

	var _ memory.SkillCompiler = &memory.DirectAPIExtractor{}
}

func TestDirectAPIExtractor_ImplementsSpecificityDetector(t *testing.T) {
	t.Parallel()

	var _ memory.SpecificityDetector = &memory.DirectAPIExtractor{}
}
