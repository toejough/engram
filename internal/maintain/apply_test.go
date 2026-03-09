package maintain_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/onsi/gomega"

	"engram/internal/maintain"
)

// T-256: Proposal ingestion — valid JSON.
func TestIngestProposals_Valid(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	input := `[
		{
			"memory_path": "memories/stale.toml",
			"quadrant": "Working",
			"action": "review_staleness",
			"diagnosis": "Memory not updated in 91 days",
			"details": {"age_days": 91}
		},
		{
			"memory_path": "memories/leech.toml",
			"quadrant": "Leech",
			"action": "rewrite",
			"diagnosis": "Memory is frequently surfaced but rarely followed",
			"details": {"proposed_keywords": ["test"]}
		},
		{
			"memory_path": "memories/noise.toml",
			"quadrant": "Noise",
			"action": "remove",
			"diagnosis": "Memory is rarely surfaced",
			"details": {"surfaced_count": 1}
		}
	]`

	proposals, err := maintain.IngestProposals([]byte(input))
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(proposals).To(gomega.HaveLen(3))
	g.Expect(proposals[0].Quadrant).To(gomega.Equal("Working"))
	g.Expect(proposals[0].Action).To(gomega.Equal("review_staleness"))
	g.Expect(proposals[0].MemoryPath).To(gomega.Equal("memories/stale.toml"))
	g.Expect(proposals[1].Quadrant).To(gomega.Equal("Leech"))
	g.Expect(proposals[1].Action).To(gomega.Equal("rewrite"))
	g.Expect(proposals[2].Quadrant).To(gomega.Equal("Noise"))
	g.Expect(proposals[2].Action).To(gomega.Equal("remove"))
}

// T-257: Proposal ingestion — invalid schema skipped.
func TestIngestProposals_InvalidSkipped(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	input := `[
		{
			"memory_path": "memories/good.toml",
			"quadrant": "Noise",
			"action": "remove",
			"diagnosis": "Low effectiveness",
			"details": {}
		},
		{
			"memory_path": "",
			"quadrant": "",
			"action": "",
			"diagnosis": "",
			"details": {}
		},
		{
			"memory_path": "memories/also-good.toml",
			"quadrant": "Working",
			"action": "review_staleness",
			"diagnosis": "Stale memory",
			"details": {}
		}
	]`

	proposals, err := maintain.IngestProposals([]byte(input))
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(proposals).To(gomega.HaveLen(2))
	g.Expect(proposals[0].MemoryPath).To(gomega.Equal("memories/good.toml"))
	g.Expect(proposals[1].MemoryPath).To(gomega.Equal("memories/also-good.toml"))
}

// T-258: Working staleness — content rewrite via LLM.
func TestApplyWorking_ContentRewrite(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	var rewrittenPath string
	var rewrittenUpdates map[string]any

	rewriter := &fakeRewriter{
		rewriteFn: func(path string, updates map[string]any) error {
			rewrittenPath = path
			rewrittenUpdates = updates
			return nil
		},
	}

	llm := &fakeLLMCaller{
		response: `{"content": "Updated content about testing", "principle": "Always write tests first"}`,
	}

	confirmer := &fakeConfirmer{responses: []bool{true}}

	executor := maintain.NewExecutor(
		maintain.WithRewriter(rewriter),
		maintain.WithLLMCaller2(llm),
		maintain.WithConfirmer(confirmer),
	)

	proposals := []maintain.Proposal{
		{
			MemoryPath: "memories/stale.toml",
			Quadrant:   "Working",
			Action:     "review_staleness",
			Diagnosis:  "Memory not updated in 91 days",
			Details:    json.RawMessage(`{"age_days": 91}`),
		},
	}

	report := executor.Apply(context.Background(), proposals)

	g.Expect(report.Applied).To(gomega.Equal(1))
	g.Expect(report.Skipped).To(gomega.Equal(0))
	g.Expect(rewrittenPath).To(gomega.Equal("memories/stale.toml"))
	g.Expect(rewrittenUpdates).To(gomega.HaveKey("content"))
	g.Expect(rewrittenUpdates).To(gomega.HaveKey("principle"))
}

// T-259: Leech rewrite — root cause content_quality.
func TestApplyLeech_ContentQualityRewrite(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	var rewrittenUpdates map[string]any

	rewriter := &fakeRewriter{
		rewriteFn: func(_ string, updates map[string]any) error {
			rewrittenUpdates = updates
			return nil
		},
	}

	llm := &fakeLLMCaller{
		response: `{"principle": "Be specific about test expectations", "anti_pattern": "Vague assertions"}`,
	}

	confirmer := &fakeConfirmer{responses: []bool{true}}

	executor := maintain.NewExecutor(
		maintain.WithRewriter(rewriter),
		maintain.WithLLMCaller2(llm),
		maintain.WithConfirmer(confirmer),
	)

	proposals := []maintain.Proposal{
		{
			MemoryPath: "memories/leech.toml",
			Quadrant:   "Leech",
			Action:     "rewrite",
			Diagnosis:  "Frequently surfaced but rarely followed",
			Details: json.RawMessage(
				`{"proposed_principle": "Be specific", "rationale": "Current wording vague"}`,
			),
		},
	}

	report := executor.Apply(context.Background(), proposals)

	g.Expect(report.Applied).To(gomega.Equal(1))
	g.Expect(rewrittenUpdates).To(gomega.HaveKey("principle"))
	g.Expect(rewrittenUpdates).To(gomega.HaveKey("anti_pattern"))
}

// T-260: HiddenGem broadening — keywords added.
func TestApplyHiddenGem_KeywordsBroadened(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	var rewrittenUpdates map[string]any

	rewriter := &fakeRewriter{
		rewriteFn: func(_ string, updates map[string]any) error {
			rewrittenUpdates = updates
			return nil
		},
	}

	llm := &fakeLLMCaller{
		response: `{"additional_keywords": ["test", "check"]}`,
	}

	confirmer := &fakeConfirmer{responses: []bool{true}}

	executor := maintain.NewExecutor(
		maintain.WithRewriter(rewriter),
		maintain.WithLLMCaller2(llm),
		maintain.WithConfirmer(confirmer),
	)

	proposals := []maintain.Proposal{
		{
			MemoryPath: "memories/gem.toml",
			Quadrant:   "Hidden Gem",
			Action:     "broaden_keywords",
			Diagnosis:  "Rarely surfaced but high follow-through",
			Details:    json.RawMessage(`{"additional_keywords": ["test", "check"]}`),
		},
	}

	report := executor.Apply(context.Background(), proposals)

	g.Expect(report.Applied).To(gomega.Equal(1))
	g.Expect(rewrittenUpdates).To(gomega.HaveKey("keywords"))

	if rewrittenUpdates == nil {
		return
	}

	keywords, ok := rewrittenUpdates["keywords"].([]string)
	g.Expect(ok).To(gomega.BeTrue())

	if !ok {
		return
	}

	g.Expect(keywords).To(gomega.ContainElements("test", "check"))
}

// T-261: Noise removal — file deleted and registry entry removed.
func TestApplyNoise_RemovalAndRegistryCleanup(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	var removedPath string
	var removedRegistryID string

	remover := &fakeRemover{
		removeFn: func(path string) error {
			removedPath = path
			return nil
		},
	}

	registry := &fakeRegistryUpdater{
		removeFn: func(id string) error {
			removedRegistryID = id
			return nil
		},
	}

	confirmer := &fakeConfirmer{responses: []bool{true}}

	executor := maintain.NewExecutor(
		maintain.WithRemover(remover),
		maintain.WithRegistry(registry),
		maintain.WithConfirmer(confirmer),
	)

	proposals := []maintain.Proposal{
		{
			MemoryPath: "memories/noise.toml",
			Quadrant:   "Noise",
			Action:     "remove",
			Diagnosis:  "Rarely surfaced and ineffective",
			Details:    json.RawMessage(`{"surfaced_count": 1}`),
		},
	}

	report := executor.Apply(context.Background(), proposals)

	g.Expect(report.Applied).To(gomega.Equal(1))
	g.Expect(removedPath).To(gomega.Equal("memories/noise.toml"))
	g.Expect(removedRegistryID).To(gomega.Equal("memories/noise.toml"))
}

// T-262: User confirmation — skip and quit.
func TestApply_ConfirmSkipQuit(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	removeCount := 0
	remover := &fakeRemover{
		removeFn: func(_ string) error {
			removeCount++
			return nil
		},
	}

	registry := &fakeRegistryUpdater{
		removeFn: func(_ string) error { return nil },
	}

	confirmer := &fakeConfirmer{
		responses: []bool{true, false},
		quitAt:    3, // quit on third call
	}

	executor := maintain.NewExecutor(
		maintain.WithRemover(remover),
		maintain.WithRegistry(registry),
		maintain.WithConfirmer(confirmer),
	)

	proposals := []maintain.Proposal{
		{
			MemoryPath: "memories/a.toml",
			Quadrant:   "Noise",
			Action:     "remove",
			Diagnosis:  "Remove A",
			Details:    json.RawMessage(`{}`),
		},
		{
			MemoryPath: "memories/b.toml",
			Quadrant:   "Noise",
			Action:     "remove",
			Diagnosis:  "Remove B",
			Details:    json.RawMessage(`{}`),
		},
		{
			MemoryPath: "memories/c.toml",
			Quadrant:   "Noise",
			Action:     "remove",
			Diagnosis:  "Remove C",
			Details:    json.RawMessage(`{}`),
		},
	}

	report := executor.Apply(context.Background(), proposals)

	g.Expect(report.Applied).To(gomega.Equal(1))
	g.Expect(report.Skipped).To(gomega.Equal(1))
	g.Expect(report.NotReached).To(gomega.Equal(1))
	g.Expect(report.Total).To(gomega.Equal(3))
	g.Expect(removeCount).To(gomega.Equal(1))
}

// T-263: No-token behavior — LLM proposals skipped.
func TestApply_NoTokenSkipsLLMProposals(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	removeCount := 0
	remover := &fakeRemover{
		removeFn: func(_ string) error {
			removeCount++
			return nil
		},
	}

	registry := &fakeRegistryUpdater{
		removeFn: func(_ string) error { return nil },
	}

	confirmer := &fakeConfirmer{responses: []bool{true, true}}

	// No LLM caller set — simulates no API token.
	executor := maintain.NewExecutor(
		maintain.WithRemover(remover),
		maintain.WithRegistry(registry),
		maintain.WithConfirmer(confirmer),
	)

	proposals := []maintain.Proposal{
		{
			MemoryPath: "memories/stale.toml",
			Quadrant:   "Working",
			Action:     "review_staleness",
			Diagnosis:  "Stale memory",
			Details:    json.RawMessage(`{"age_days": 91}`),
		},
		{
			MemoryPath: "memories/noise.toml",
			Quadrant:   "Noise",
			Action:     "remove",
			Diagnosis:  "Remove noise",
			Details:    json.RawMessage(`{}`),
		},
	}

	report := executor.Apply(context.Background(), proposals)

	g.Expect(report.Applied).To(gomega.Equal(1))
	g.Expect(report.Skipped).To(gomega.Equal(1))
	g.Expect(report.SkipReasons).To(gomega.ContainElement("no token"))
	g.Expect(removeCount).To(gomega.Equal(1))
}

// --- Test fakes ---

type fakeRewriter struct {
	rewriteFn func(path string, updates map[string]any) error
}

func (f *fakeRewriter) Rewrite(path string, updates map[string]any) error {
	return f.rewriteFn(path, updates)
}

type fakeRemover struct {
	removeFn func(path string) error
}

func (f *fakeRemover) Remove(path string) error {
	return f.removeFn(path)
}

type fakeRegistryUpdater struct {
	removeFn func(id string) error
}

func (f *fakeRegistryUpdater) RemoveEntry(id string) error {
	return f.removeFn(id)
}

type fakeLLMCaller struct {
	response string
	err      error
}

func (f *fakeLLMCaller) Call(_ context.Context, _ string) (string, error) {
	return f.response, f.err
}

// ErrQuit is returned by fakeConfirmer when quitAt is reached.
var errQuit = errors.New("user quit")

type fakeConfirmer struct {
	responses []bool
	quitAt    int // 1-indexed; 0 means never quit
	callCount int
}

func (f *fakeConfirmer) Confirm(_ string) (bool, error) {
	f.callCount++
	if f.quitAt > 0 && f.callCount >= f.quitAt {
		return false, maintain.ErrUserQuit
	}

	idx := f.callCount - 1
	if idx < len(f.responses) {
		return f.responses[idx], nil
	}

	return false, nil
}
