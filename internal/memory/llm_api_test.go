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

// Synthesize tests
func TestDirectAPIExtractor_Synthesize_ReturnsPrinciple(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": "Always use TDD for code changes"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	extractor := memory.NewDirectAPIExtractor("token",
		memory.WithBaseURL(srv.URL),
	)

	result, err := extractor.Synthesize(context.Background(), []string{"mem1", "mem2"})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(Equal("Always use TDD for code changes"))
}

func TestDirectAPIExtractor_Synthesize_ReturnsErrOn401(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	extractor := memory.NewDirectAPIExtractor("bad-token",
		memory.WithBaseURL(srv.URL),
	)

	result, err := extractor.Synthesize(context.Background(), []string{"mem"})
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, memory.ErrLLMUnavailable)).To(BeTrue())
	g.Expect(result).To(BeEmpty())
}

// Curate tests
func TestDirectAPIExtractor_Curate_ReturnsCuratedResults(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	results := []memory.CuratedResult{
		{Content: "result1", Relevance: "highly relevant", MemoryType: "pattern"},
		{Content: "result2", Relevance: "somewhat relevant", MemoryType: "correction"},
	}
	resultsJSON, _ := json.Marshal(results)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": string(resultsJSON)},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	extractor := memory.NewDirectAPIExtractor("token",
		memory.WithBaseURL(srv.URL),
	)

	candidates := []memory.QueryResult{
		{Content: "result1", Score: 0.9},
		{Content: "result2", Score: 0.7},
	}
	result, err := extractor.Curate(context.Background(), "test query", candidates)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(HaveLen(2))
	g.Expect(result[0].Content).To(Equal("result1"))
	g.Expect(result[0].Relevance).To(Equal("highly relevant"))
}

func TestDirectAPIExtractor_Curate_ReturnsErrOn401(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	extractor := memory.NewDirectAPIExtractor("bad-token",
		memory.WithBaseURL(srv.URL),
	)

	result, err := extractor.Curate(context.Background(), "query", []memory.QueryResult{})
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, memory.ErrLLMUnavailable)).To(BeTrue())
	g.Expect(result).To(BeNil())
}

func TestDirectAPIExtractor_Curate_ReturnsErrOnBadJSON(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": "not valid json array"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	extractor := memory.NewDirectAPIExtractor("token",
		memory.WithBaseURL(srv.URL),
	)

	result, err := extractor.Curate(context.Background(), "query", []memory.QueryResult{})
	g.Expect(err).To(HaveOccurred())
	g.Expect(result).To(BeNil())
}

// Decide tests
func TestDirectAPIExtractor_Decide_ReturnsDecision(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	decision := memory.IngestDecision{
		Action:   memory.IngestAdd,
		TargetID: 0,
		Reason:   "New knowledge",
	}
	decisionJSON, _ := json.Marshal(decision)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": string(decisionJSON)},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	extractor := memory.NewDirectAPIExtractor("token",
		memory.WithBaseURL(srv.URL),
	)

	result, err := extractor.Decide(context.Background(), "new content", []memory.ExistingMemory{})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Action).To(Equal(memory.IngestAdd))
	g.Expect(result.Reason).To(Equal("New knowledge"))
}

func TestDirectAPIExtractor_Decide_ReturnsErrOn401(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	extractor := memory.NewDirectAPIExtractor("bad-token",
		memory.WithBaseURL(srv.URL),
	)

	result, err := extractor.Decide(context.Background(), "content", []memory.ExistingMemory{})
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, memory.ErrLLMUnavailable)).To(BeTrue())
	g.Expect(result).To(BeNil())
}

func TestDirectAPIExtractor_Decide_ReturnsErrOnBadJSON(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": "invalid json"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	extractor := memory.NewDirectAPIExtractor("token",
		memory.WithBaseURL(srv.URL),
	)

	result, err := extractor.Decide(context.Background(), "content", []memory.ExistingMemory{})
	g.Expect(err).To(HaveOccurred())
	g.Expect(result).To(BeNil())
}

// CompileSkill tests
func TestDirectAPIExtractor_CompileSkill_ReturnsSkillContent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	skillContent := "# My Skill\n\nThis is skill content."

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": skillContent},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	extractor := memory.NewDirectAPIExtractor("token",
		memory.WithBaseURL(srv.URL),
	)

	result, err := extractor.CompileSkill(context.Background(), "theme", []string{"mem1", "mem2"})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(Equal(skillContent))
}

func TestDirectAPIExtractor_CompileSkill_ReturnsErrOn401(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	extractor := memory.NewDirectAPIExtractor("bad-token",
		memory.WithBaseURL(srv.URL),
	)

	result, err := extractor.CompileSkill(context.Background(), "theme", []string{"mem"})
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, memory.ErrLLMUnavailable)).To(BeTrue())
	g.Expect(result).To(BeEmpty())
}

// IsNarrowLearning tests
func TestDirectAPIExtractor_IsNarrowLearning_ReturnsAnalysis(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	response := map[string]any{
		"is_narrow":  true,
		"reason":     "References specific file path",
		"confidence": 0.95,
	}
	responseJSON, _ := json.Marshal(response)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": string(responseJSON)},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	extractor := memory.NewDirectAPIExtractor("token",
		memory.WithBaseURL(srv.URL),
	)

	isNarrow, reason, err := extractor.IsNarrowLearning(context.Background(), "Fix bug in src/config.yaml")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(isNarrow).To(BeTrue())
	g.Expect(reason).To(Equal("References specific file path"))
}

func TestDirectAPIExtractor_IsNarrowLearning_ReturnsErrOn401(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	extractor := memory.NewDirectAPIExtractor("bad-token",
		memory.WithBaseURL(srv.URL),
	)

	isNarrow, reason, err := extractor.IsNarrowLearning(context.Background(), "learning")
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, memory.ErrLLMUnavailable)).To(BeTrue())
	g.Expect(isNarrow).To(BeFalse())
	g.Expect(reason).To(BeEmpty())
}

func TestDirectAPIExtractor_IsNarrowLearning_ReturnsErrOnBadJSON(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": "invalid json"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	extractor := memory.NewDirectAPIExtractor("token",
		memory.WithBaseURL(srv.URL),
	)

	isNarrow, reason, err := extractor.IsNarrowLearning(context.Background(), "learning")
	g.Expect(err).To(HaveOccurred())
	g.Expect(isNarrow).To(BeFalse())
	g.Expect(reason).To(BeEmpty())
}

// Rewrite tests
func TestDirectAPIExtractor_Rewrite_ReturnsRewrittenContent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	rewritten := "Always validate user input before processing"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": rewritten},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	extractor := memory.NewDirectAPIExtractor("token",
		memory.WithBaseURL(srv.URL),
	)

	result, err := extractor.Rewrite(context.Background(), "validate input")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(Equal(rewritten))
}

func TestDirectAPIExtractor_Rewrite_ReturnsErrOn401(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	extractor := memory.NewDirectAPIExtractor("bad-token",
		memory.WithBaseURL(srv.URL),
	)

	result, err := extractor.Rewrite(context.Background(), "content")
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, memory.ErrLLMUnavailable)).To(BeTrue())
	g.Expect(result).To(BeEmpty())
}

// AddRationale tests
func TestDirectAPIExtractor_AddRationale_ReturnsEnrichedContent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	enriched := "Never use global variables - they make testing difficult and create hidden dependencies"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": enriched},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	extractor := memory.NewDirectAPIExtractor("token",
		memory.WithBaseURL(srv.URL),
	)

	result, err := extractor.AddRationale(context.Background(), "Never use global variables")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(Equal(enriched))
}

func TestDirectAPIExtractor_AddRationale_ReturnsErrOn401(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	extractor := memory.NewDirectAPIExtractor("bad-token",
		memory.WithBaseURL(srv.URL),
	)

	result, err := extractor.AddRationale(context.Background(), "content")
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, memory.ErrLLMUnavailable)).To(BeTrue())
	g.Expect(result).To(BeEmpty())
}

// NewLLMExtractor factory tests
func TestNewLLMExtractor_ReturnsDirectAPIWhenTokenAvailable(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	creds := map[string]any{
		"claudeAiOauth": map[string]any{
			"accessToken": "sk-ant-oat01-valid",
			"expiresAt":   "2099-12-31T23:59:59Z",
		},
	}
	credsJSON, _ := json.Marshal(creds)

	auth := &memory.KeychainAuth{
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return credsJSON, nil
		},
	}

	extractor := memory.NewLLMExtractor(memory.WithAuth(auth))
	_, isDirectAPI := extractor.(*memory.DirectAPIExtractor)
	g.Expect(isDirectAPI).To(BeTrue())
}

func TestNewLLMExtractor_ReturnsNilWhenKeychainFails(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	auth := &memory.KeychainAuth{
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return nil, errors.New("keychain not available")
		},
	}

	extractor := memory.NewLLMExtractor(memory.WithAuth(auth))
	g.Expect(extractor).To(BeNil())
}
