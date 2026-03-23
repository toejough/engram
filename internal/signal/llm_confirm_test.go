package signal_test

import (
	"context"
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/signal"
)

func TestLLMConfirmer_ConfirmsClusters(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	mockCaller := func(
		_ context.Context, _, _, _ string,
	) (string, error) {
		return `{
			"clusters": [
				{
					"member_indices": [0, 1],
					"principle": "Always use dependency injection for I/O"
				}
			],
			"contradictions": []
		}`, nil
	}

	confirmer := signal.NewLLMConfirmer(mockCaller)

	query := &memory.MemoryRecord{
		Title:     "DI for testing",
		Principle: "Use DI for testability",
		Keywords:  []string{"dependency-injection", "testing"},
	}
	candidates := []signal.ScoredCandidate{
		{
			Memory: &memory.MemoryRecord{
				Title:     "Inject interfaces",
				Principle: "Inject interfaces instead of concrete types",
				Keywords:  []string{"interfaces", "injection"},
			},
			Score: 0.85,
		},
		{
			Memory: &memory.MemoryRecord{
				Title:     "Mock via DI",
				Principle: "Mock dependencies via injection",
				Keywords:  []string{"mocking", "di"},
			},
			Score: 0.80,
		},
	}

	clusters, err := confirmer.ConfirmClusters(context.Background(), query, candidates)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(clusters).To(HaveLen(1))
	g.Expect(clusters[0].Principle).To(Equal("Always use dependency injection for I/O"))
	g.Expect(clusters[0].Members).To(HaveLen(2))
}

func TestLLMConfirmer_ExcludesContradictions(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	mockCaller := func(
		_ context.Context, _, _, _ string,
	) (string, error) {
		return `{
			"clusters": [
				{
					"member_indices": [0, 1, 2],
					"principle": "Use TDD for all changes"
				}
			],
			"contradictions": [2]
		}`, nil
	}

	confirmer := signal.NewLLMConfirmer(mockCaller)

	query := &memory.MemoryRecord{
		Title:    "TDD workflow",
		Keywords: []string{"tdd"},
	}
	candidates := []signal.ScoredCandidate{
		{
			Memory: &memory.MemoryRecord{Title: "Red green refactor"},
			Score:  0.9,
		},
		{
			Memory: &memory.MemoryRecord{Title: "Write tests first"},
			Score:  0.85,
		},
		{
			Memory: &memory.MemoryRecord{Title: "Skip tests for prototypes"},
			Score:  0.7,
		},
	}

	clusters, err := confirmer.ConfirmClusters(context.Background(), query, candidates)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(clusters).To(HaveLen(1))
	g.Expect(clusters[0].Members).To(HaveLen(2))

	for _, member := range clusters[0].Members {
		g.Expect(member.Title).NotTo(Equal("Skip tests for prototypes"))
	}
}

func TestLLMConfirmer_InvalidJSON(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	mockCaller := func(
		_ context.Context, _, _, _ string,
	) (string, error) {
		return "not valid json at all", nil
	}

	confirmer := signal.NewLLMConfirmer(mockCaller)

	query := &memory.MemoryRecord{Title: "test"}

	_, err := confirmer.ConfirmClusters(context.Background(), query, nil)
	g.Expect(err).To(HaveOccurred())
}

func TestLLMConfirmer_LLMError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	errLLM := errors.New("LLM service unavailable")
	mockCaller := func(
		_ context.Context, _, _, _ string,
	) (string, error) {
		return "", errLLM
	}

	confirmer := signal.NewLLMConfirmer(mockCaller)

	query := &memory.MemoryRecord{Title: "test"}

	_, err := confirmer.ConfirmClusters(context.Background(), query, nil)
	g.Expect(err).To(MatchError(ContainSubstring("confirming clusters")))
}

func TestLLMConfirmer_MarkdownFencedNoNewline(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Edge case: fenced but no newline after opening fence
	mockCaller := func(
		_ context.Context, _, _, _ string,
	) (string, error) {
		return "```", nil
	}

	confirmer := signal.NewLLMConfirmer(mockCaller)
	query := &memory.MemoryRecord{Title: "test"}

	_, err := confirmer.ConfirmClusters(context.Background(), query, nil)
	g.Expect(err).To(HaveOccurred())
}

func TestLLMConfirmer_MarkdownFencedResponse(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	mockCaller := func(
		_ context.Context, _, _, _ string,
	) (string, error) {
		return "```json\n{\"clusters\": [], \"contradictions\": []}\n```", nil
	}

	confirmer := signal.NewLLMConfirmer(mockCaller)
	query := &memory.MemoryRecord{Title: "test"}

	clusters, err := confirmer.ConfirmClusters(context.Background(), query, nil)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(clusters).To(BeEmpty())
}

func TestLLMConfirmer_NoCluster(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	mockCaller := func(
		_ context.Context, _, _, _ string,
	) (string, error) {
		return `{"clusters": [], "contradictions": []}`, nil
	}

	confirmer := signal.NewLLMConfirmer(mockCaller)

	query := &memory.MemoryRecord{
		Title:    "Unrelated memory",
		Keywords: []string{"unrelated"},
	}

	clusters, err := confirmer.ConfirmClusters(context.Background(), query, nil)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(clusters).To(BeEmpty())
}

func TestLLMConfirmer_OutOfBoundsIndices(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	mockCaller := func(
		_ context.Context, _, _, _ string,
	) (string, error) {
		return `{"clusters": [{"member_indices": [0, 99], "principle": "test"}], "contradictions": []}`, nil
	}

	confirmer := signal.NewLLMConfirmer(mockCaller)
	query := &memory.MemoryRecord{Title: "test"}

	clusters, err := confirmer.ConfirmClusters(context.Background(), query, nil)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Only index 0 (query) is valid; index 99 is out of bounds
	g.Expect(clusters).To(HaveLen(1))
	g.Expect(clusters[0].Members).To(HaveLen(1))
}

func TestLLMExtractor_ExtractsPrinciple(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	mockCaller := func(
		_ context.Context, _, _, _ string,
	) (string, error) {
		return `{
			"title": "Dependency Injection for I/O",
			"principle": "Inject all I/O dependencies",
			"anti_pattern": "Calling os.Open directly in business logic",
			"content": "All I/O should be injected via interfaces",
			"keywords": ["dependency-injection", "interfaces", "testing"],
			"concepts": ["DI", "testability"],
			"generalizability": 4
		}`, nil
	}

	extractor := signal.NewLLMExtractor(mockCaller)

	cluster := signal.ConfirmedCluster{
		Members: []*memory.MemoryRecord{
			{
				Title:     "DI for testing",
				Content:   "Use DI to make code testable",
				Principle: "Inject interfaces",
				Keywords:  []string{"di", "testing"},
			},
			{
				Title:     "Mock via interfaces",
				Content:   "Create interface wrappers for I/O",
				Principle: "Interface-based mocking",
				Keywords:  []string{"interfaces", "mocking"},
			},
		},
		Principle: "Use dependency injection for I/O",
	}

	record, err := extractor.ExtractPrinciple(context.Background(), cluster)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(record).NotTo(BeNil())

	if record == nil {
		return
	}

	g.Expect(record.Title).To(Equal("Dependency Injection for I/O"))
	g.Expect(record.Principle).To(Equal("Inject all I/O dependencies"))
	g.Expect(record.AntiPattern).To(Equal("Calling os.Open directly in business logic"))
	g.Expect(record.Content).To(Equal("All I/O should be injected via interfaces"))
	g.Expect(record.Keywords).To(ConsistOf("dependency-injection", "interfaces", "testing"))
	g.Expect(record.Concepts).To(ConsistOf("DI", "testability"))
}

func TestLLMExtractor_LLMError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	errLLM := errors.New("LLM timeout")
	mockCaller := func(
		_ context.Context, _, _, _ string,
	) (string, error) {
		return "", errLLM
	}

	extractor := signal.NewLLMExtractor(mockCaller)

	cluster := signal.ConfirmedCluster{
		Members: []*memory.MemoryRecord{
			{Title: "Test"},
		},
		Principle: "Test principle",
	}

	_, err := extractor.ExtractPrinciple(context.Background(), cluster)
	g.Expect(err).To(MatchError(ContainSubstring("extracting principle")))
}

func TestLLMExtractor_MarkdownFencedResponse(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	mockCaller := func(
		_ context.Context, _, _, _ string,
	) (string, error) {
		return "```json\n" + `{
			"title": "Test",
			"principle": "Test",
			"anti_pattern": "",
			"content": "Test",
			"keywords": ["test"],
			"concepts": ["test"],
			"generalizability": 3
		}` + "\n```", nil
	}

	extractor := signal.NewLLMExtractor(mockCaller)
	cluster := signal.ConfirmedCluster{
		Members:   []*memory.MemoryRecord{{Title: "Test"}},
		Principle: "Test",
	}

	record, err := extractor.ExtractPrinciple(context.Background(), cluster)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(record).NotTo(BeNil())

	if record == nil {
		return
	}

	g.Expect(record.Title).To(Equal("Test"))
}

func TestLLMExtractor_SetsGeneralizability(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const expectedGeneralizability = 5

	mockCaller := func(
		_ context.Context, _, _, _ string,
	) (string, error) {
		return `{
			"title": "Universal principle",
			"principle": "Always test",
			"anti_pattern": "",
			"content": "Testing is universal",
			"keywords": ["testing"],
			"concepts": ["quality"],
			"generalizability": 5
		}`, nil
	}

	extractor := signal.NewLLMExtractor(mockCaller)

	cluster := signal.ConfirmedCluster{
		Members: []*memory.MemoryRecord{
			{Title: "Test everything", Content: "Always test"},
		},
		Principle: "Always test",
	}

	record, err := extractor.ExtractPrinciple(context.Background(), cluster)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(record).NotTo(BeNil())

	if record == nil {
		return
	}

	g.Expect(record.Generalizability).To(Equal(expectedGeneralizability))
}
