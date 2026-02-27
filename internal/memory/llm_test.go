package memory_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/memory"
)

// --- Interface compliance ---

func TestClaudeCLIExtractorImplementsLLMExtractor(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var _ memory.LLMExtractor = &memory.ClaudeCLIExtractor{}

	g.Expect(true).To(BeTrue()) // compiles = interface satisfied
}

func TestCuratePassesQueryAndCandidatesInPrompt(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var capturedArgs []string

	curatedJSON := []memory.CuratedResult{{Content: "c1", Relevance: "r1", MemoryType: "t1"}}
	jsonBytes, _ := json.Marshal(curatedJSON)

	extractor := &memory.ClaudeCLIExtractor{
		Model:   "haiku",
		Timeout: 30,
		CommandRunner: func(_ context.Context, name string, args ...string) ([]byte, error) {
			capturedArgs = append([]string{name}, args...)
			return jsonBytes, nil
		},
	}

	candidates := []memory.QueryResult{
		{Content: "candidate alpha"},
		{Content: "candidate beta"},
	}

	_, err := extractor.Curate(context.Background(), "my search query", candidates)
	g.Expect(err).ToNot(HaveOccurred())

	foundQuery := false
	foundCandidate := false

	for _, arg := range capturedArgs {
		if llmTestContains(arg, "my search query") {
			foundQuery = true
		}

		if llmTestContains(arg, "candidate alpha") {
			foundCandidate = true
		}
	}

	g.Expect(foundQuery).To(BeTrue(), "Expected prompt to contain the query")
	g.Expect(foundCandidate).To(BeTrue(), "Expected prompt to contain candidates")
}

func TestCurateReturnsErrLLMUnavailableWhenCommandFails(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	extractor := &memory.ClaudeCLIExtractor{
		Model:   "haiku",
		Timeout: 30,
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return nil, errors.New("timeout exceeded")
		},
	}

	results, err := extractor.Curate(context.Background(), "query", []memory.QueryResult{{Content: "mem1"}})
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, memory.ErrLLMUnavailable)).To(BeTrue())
	g.Expect(results).To(BeNil())
}

func TestCurateReturnsErrorOnInvalidJSON(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	extractor := &memory.ClaudeCLIExtractor{
		Model:   "haiku",
		Timeout: 30,
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte("not valid json at all"), nil
		},
	}

	results, err := extractor.Curate(context.Background(), "query", []memory.QueryResult{{Content: "mem1"}})
	g.Expect(err).To(HaveOccurred())
	g.Expect(results).To(BeNil())
}

// --- Curate tests ---

func TestCurateReturnsFilteredResults(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	curatedJSON := []memory.CuratedResult{
		{Content: "Use TDD always", Relevance: "Directly answers the query about testing methodology", MemoryType: "correction"},
		{Content: "Prefer gomega matchers", Relevance: "Related to testing tools", MemoryType: "pattern"},
	}
	jsonBytes, _ := json.Marshal(curatedJSON)

	extractor := &memory.ClaudeCLIExtractor{
		Model:   "haiku",
		Timeout: 30,
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return jsonBytes, nil
		},
	}

	candidates := []memory.QueryResult{
		{Content: "Use TDD always", Score: 0.9},
		{Content: "Prefer gomega matchers", Score: 0.85},
		{Content: "Unrelated memory", Score: 0.3},
	}

	results, err := extractor.Curate(context.Background(), "how should I test?", candidates)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(results).To(HaveLen(2))

	if len(results) < 2 {
		t.Fatalf("expected 2 results from Curate, got %d", len(results))
	}

	g.Expect(results[0].Content).To(Equal("Use TDD always"))
	g.Expect(results[0].Relevance).ToNot(BeEmpty())
	g.Expect(results[1].Content).To(Equal("Prefer gomega matchers"))
}

func TestExtractPassesContentInPrompt(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var capturedArgs []string

	obs := memory.Observation{Type: "pattern", Concepts: []string{"test"}, Principle: "test", AntiPattern: "", Rationale: ""}
	jsonBytes, _ := json.Marshal(obs)

	extractor := &memory.ClaudeCLIExtractor{
		Model:   "haiku",
		Timeout: 30,
		CommandRunner: func(_ context.Context, name string, args ...string) ([]byte, error) {
			capturedArgs = append([]string{name}, args...)
			return jsonBytes, nil
		},
	}

	_, err := extractor.Extract(context.Background(), "my memory content here")
	g.Expect(err).ToNot(HaveOccurred())
	// The prompt arg should contain the memory content
	g.Expect(capturedArgs).ToNot(BeEmpty())

	found := false

	for _, arg := range capturedArgs {
		if len(arg) > 0 && llmTestContains(arg, "my memory content here") {
			found = true
			break
		}
	}

	g.Expect(found).To(BeTrue(), "Expected prompt to contain the memory content")
}

func TestExtractReturnsErrLLMUnavailableWhenCommandFails(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	extractor := &memory.ClaudeCLIExtractor{
		Model:   "haiku",
		Timeout: 30,
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return nil, errors.New("executable file not found")
		},
	}

	result, err := extractor.Extract(context.Background(), "some content")
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, memory.ErrLLMUnavailable)).To(BeTrue())
	g.Expect(result).To(BeNil())
}

func TestExtractReturnsErrorOnInvalidJSON(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	extractor := &memory.ClaudeCLIExtractor{
		Model:   "haiku",
		Timeout: 30,
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte("this is not json"), nil
		},
	}

	result, err := extractor.Extract(context.Background(), "some content")
	g.Expect(err).To(HaveOccurred())
	g.Expect(result).To(BeNil())
}

// ============================================================================
// ISSUE-188 Task 3: LLM extractor interface and Claude CLI implementation
// ============================================================================

// --- Extract tests ---

func TestExtractReturnsObservationFromValidJSON(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	obs := memory.Observation{
		Type:        "correction",
		Concepts:    []string{"git", "safety"},
		Principle:   "Never amend pushed commits",
		AntiPattern: "Using git commit --amend after push",
		Rationale:   "Rewriting shared history causes collaboration issues",
	}
	jsonBytes, err := json.Marshal(obs)
	g.Expect(err).ToNot(HaveOccurred())

	extractor := &memory.ClaudeCLIExtractor{
		Model:   "haiku",
		Timeout: 30,
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return jsonBytes, nil
		},
	}

	result, err := extractor.Extract(context.Background(), "Never amend pushed commits because it rewrites shared history")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).ToNot(BeNil())

	if result == nil {
		t.Fatal("Extract returned nil result")
	}

	g.Expect(result.Type).To(Equal("correction"))
	g.Expect(result.Concepts).To(ContainElement("git"))
	g.Expect(result.Principle).To(Equal("Never amend pushed commits"))
	g.Expect(result.AntiPattern).To(Equal("Using git commit --amend after push"))
	g.Expect(result.Rationale).To(Equal("Rewriting shared history causes collaboration issues"))
}

// --- Model and timeout are passed through ---

func TestExtractUsesConfiguredModel(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var capturedArgs []string

	obs := memory.Observation{Type: "pattern", Concepts: []string{"x"}, Principle: "p"}
	jsonBytes, _ := json.Marshal(obs)

	extractor := &memory.ClaudeCLIExtractor{
		Model:   "sonnet",
		Timeout: 60,
		CommandRunner: func(_ context.Context, name string, args ...string) ([]byte, error) {
			capturedArgs = append([]string{name}, args...)
			return jsonBytes, nil
		},
	}

	_, err := extractor.Extract(context.Background(), "test")
	g.Expect(err).ToNot(HaveOccurred())
	// Should contain "--model" followed by "sonnet" somewhere in args
	foundModel := false

	for i, arg := range capturedArgs {
		if arg == "--model" && i+1 < len(capturedArgs) && capturedArgs[i+1] == "sonnet" {
			foundModel = true
			break
		}
	}

	g.Expect(foundModel).To(BeTrue(), "Expected --model sonnet in command args")
}

func TestIsNarrowLearning_LLMUnavailable(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	extractor := &memory.ClaudeCLIExtractor{
		Model:   "haiku",
		Timeout: 30,
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return nil, errors.New("executable file not found")
		},
	}

	isNarrow, reason, err := extractor.IsNarrowLearning(context.Background(), "some learning content")
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, memory.ErrLLMUnavailable)).To(BeTrue())
	g.Expect(isNarrow).To(BeFalse())
	g.Expect(reason).To(BeEmpty())
}

func TestIsNarrowLearning_MalformedJSON(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	extractor := &memory.ClaudeCLIExtractor{
		Model:   "haiku",
		Timeout: 30,
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte("this is not json"), nil
		},
	}

	isNarrow, reason, err := extractor.IsNarrowLearning(context.Background(), "some learning content")
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, memory.ErrLLMUnavailable)).To(BeTrue())
	g.Expect(isNarrow).To(BeFalse())
	g.Expect(reason).To(BeEmpty())
}

func TestIsNarrowLearning_NarrowPattern(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	response := map[string]any{
		"is_narrow":  true,
		"reason":     "References specific file path 'src/config.yaml' - project-specific",
		"confidence": 0.92,
	}
	jsonBytes, _ := json.Marshal(response)

	extractor := &memory.ClaudeCLIExtractor{
		Model:   "haiku",
		Timeout: 30,
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return jsonBytes, nil
		},
	}

	isNarrow, reason, err := extractor.IsNarrowLearning(context.Background(), "Fix the bug in src/config.yaml by updating the timeout value")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(isNarrow).To(BeTrue())
	g.Expect(reason).To(Equal("References specific file path 'src/config.yaml' - project-specific"))
}

// --- IsNarrowLearning tests (TASK-1) ---

func TestIsNarrowLearning_UniversalPattern(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	response := map[string]any{
		"is_narrow":  false,
		"reason":     "Universal testing principle applicable across all contexts",
		"confidence": 0.95,
	}
	jsonBytes, _ := json.Marshal(response)

	extractor := &memory.ClaudeCLIExtractor{
		Model:   "haiku",
		Timeout: 30,
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return jsonBytes, nil
		},
	}

	isNarrow, reason, err := extractor.IsNarrowLearning(context.Background(), "Always run tests before committing code")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(isNarrow).To(BeFalse())
	g.Expect(reason).To(Equal("Universal testing principle applicable across all contexts"))
}

// --- Constructor tests ---

func TestNewClaudeCLIExtractorSetsDefaults(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	extractor := memory.NewClaudeCLIExtractor()
	g.Expect(extractor).ToNot(BeNil())
	g.Expect(extractor.Model).To(Equal("haiku"))
	g.Expect(extractor.Timeout).To(BeNumerically("==", 30e9)) // 30 seconds in nanoseconds
	g.Expect(extractor.CommandRunner).ToNot(BeNil())
}

// --- Property tests ---

func TestPropertyExtractAlwaysReturnsValidTypeOrError(t *testing.T) {
	t.Parallel()

	validTypes := map[string]bool{
		"correction": true,
		"pattern":    true,
		"decision":   true,
		"discovery":  true,
	}

	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		obsType := rapid.SampledFrom([]string{"correction", "pattern", "decision", "discovery"}).Draw(rt, "type")
		obs := memory.Observation{
			Type:      obsType,
			Concepts:  []string{"concept"},
			Principle: "principle",
		}
		jsonBytes, _ := json.Marshal(obs)

		extractor := &memory.ClaudeCLIExtractor{
			Model:   "haiku",
			Timeout: 30,
			CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
				return jsonBytes, nil
			},
		}

		result, err := extractor.Extract(context.Background(), "test content")
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(validTypes[result.Type]).To(BeTrue(),
			fmt.Sprintf("Extract returned invalid type %q", result.Type))
	})
}

func TestPropertyExtractCommandRunnerFailureAlwaysReturnsErrLLMUnavailable(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewWithT(t)

		errMsg := rapid.StringMatching(`[a-zA-Z ]{1,50}`).Draw(rt, "errorMessage")

		extractor := &memory.ClaudeCLIExtractor{
			Model:   "haiku",
			Timeout: 30,
			CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
				return nil, errors.New(errMsg)
			},
		}

		_, err := extractor.Extract(context.Background(), "content")
		g.Expect(err).To(HaveOccurred())
		g.Expect(errors.Is(err, memory.ErrLLMUnavailable)).To(BeTrue())
	})
}

func TestSynthesizePassesMemoriesInPrompt(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var capturedArgs []string

	extractor := &memory.ClaudeCLIExtractor{
		Model:   "haiku",
		Timeout: 30,
		CommandRunner: func(_ context.Context, name string, args ...string) ([]byte, error) {
			capturedArgs = append([]string{name}, args...)
			return []byte("synthesized principle"), nil
		},
	}

	_, err := extractor.Synthesize(context.Background(), []string{"memory alpha", "memory beta"})
	g.Expect(err).ToNot(HaveOccurred())

	found := false

	for _, arg := range capturedArgs {
		if llmTestContains(arg, "memory alpha") && llmTestContains(arg, "memory beta") {
			found = true
			break
		}
	}

	g.Expect(found).To(BeTrue(), "Expected prompt to contain all memories")
}

func TestSynthesizeReturnsErrLLMUnavailableWhenCommandFails(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	extractor := &memory.ClaudeCLIExtractor{
		Model:   "haiku",
		Timeout: 30,
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return nil, errors.New("command not found")
		},
	}

	result, err := extractor.Synthesize(context.Background(), []string{"memory1", "memory2"})
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, memory.ErrLLMUnavailable)).To(BeTrue())
	g.Expect(result).To(BeEmpty())
}

// --- Synthesize tests ---

func TestSynthesizeReturnsPrincipleFromLLM(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	extractor := &memory.ClaudeCLIExtractor{
		Model:   "haiku",
		Timeout: 30,
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte("Always run tests before committing code to catch regressions early."), nil
		},
	}

	result, err := extractor.Synthesize(context.Background(), []string{
		"ran tests before commit, caught bug",
		"forgot to test, broken deploy",
		"test-first approach saved time",
	})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(ContainSubstring("tests"))
}

// helper - use strings package instead of redeclaring containsSubstring
func llmTestContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}
