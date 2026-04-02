package maintain_test

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/anthropic"
	"engram/internal/maintain"
	"engram/internal/policy"
)

func TestRunSonnetAnalyses_RunsConcurrently(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")

	g.Expect(os.MkdirAll(memDir, 0o755)).To(Succeed())

	g.Expect(os.WriteFile(filepath.Join(memDir, "mem-a.toml"), []byte(workingMemory), 0o644)).To(Succeed())
	g.Expect(os.WriteFile(filepath.Join(memDir, "mem-b.toml"), []byte(workingMemory), 0o644)).To(Succeed())

	defaults := policy.Defaults()

	var barrier sync.WaitGroup

	barrier.Add(2)

	mockCaller := anthropic.CallerFunc(
		func(_ context.Context, _, systemPrompt, _ string) (string, error) {
			barrier.Done() // signal arrival
			barrier.Wait() // block until both goroutines have arrived

			if systemPrompt == defaults.MaintainConsolidatePrompt {
				return `[{"survivor":"mem-a.toml","members":["mem-a.toml","mem-b.toml"],"rationale":"similar"}]`, nil
			}

			return `[{"field":"maintain_min_surfaced","value":"10","rationale":"increase"}]`, nil
		},
	)

	cfg := maintain.Config{
		Policy:        defaults,
		DataDir:       dataDir,
		Caller:        mockCaller,
		ChangeHistory: []policy.ChangeEntry{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	proposals, err := maintain.Run(ctx, cfg)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	actions := make([]string, 0, len(proposals))

	for _, proposal := range proposals {
		actions = append(actions, proposal.Action)
	}

	g.Expect(actions).To(ContainElement(maintain.ActionMerge))
	g.Expect(actions).To(ContainElement(maintain.ActionUpdate))
}

func TestRun_CallerError_ReturnsProposalsAndError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")

	err := os.MkdirAll(memDir, 0o755)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	err = os.WriteFile(
		filepath.Join(memDir, "high-irrelevance.toml"),
		[]byte(highIrrelevanceMemory), 0o644,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Need 2+ memories so consolidation actually runs (and fails).
	err = os.WriteFile(filepath.Join(memDir, "working.toml"), []byte(workingMemory), 0o644)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	failingCaller := anthropic.CallerFunc(
		func(_ context.Context, _, _, _ string) (string, error) {
			return "", anthropic.ErrAPIError
		},
	)

	cfg := maintain.Config{
		Policy:  policy.Defaults(),
		DataDir: dataDir,
		Caller:  failingCaller,
	}

	proposals, runErr := maintain.Run(context.Background(), cfg)

	// Error is surfaced loudly — caller knows consolidation/adapt failed.
	g.Expect(runErr).To(HaveOccurred())
	g.Expect(runErr.Error()).To(ContainSubstring("finding merges"))
	g.Expect(runErr.Error()).To(ContainSubstring("adapt analysis"))

	// Decision tree + gate accuracy proposals are still returned alongside the error.
	g.Expect(proposals).To(HaveLen(2))

	if len(proposals) == 0 {
		return
	}

	g.Expect(proposals[0].Action).To(Equal(maintain.ActionDelete))
	g.Expect(proposals[0].Target).To(ContainSubstring("high-irrelevance"))
}

func TestRun_EmptyMemoryDir(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")

	err := os.MkdirAll(memDir, 0o755)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	cfg := maintain.Config{
		Policy:  policy.Defaults(),
		DataDir: dataDir,
	}

	proposals, err := maintain.Run(context.Background(), cfg)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(proposals).To(BeNil())
}

func TestRun_GateAccuracyProposal_HighIrrelevance(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")

	g.Expect(os.MkdirAll(memDir, 0o755)).To(Succeed())

	// Memory with 50% irrelevance rate (above 10% threshold).
	highIrrelevance := `situation = "when doing X"
behavior = "always fails"
impact = "wastes time"
action = "stop doing X"
created_at = "2024-01-01T00:00:00Z"
updated_at = "2024-01-01T00:00:00Z"
surfaced_count = 10
followed_count = 0
not_followed_count = 0
irrelevant_count = 5
`
	g.Expect(os.WriteFile(
		filepath.Join(memDir, "high-irrel.toml"),
		[]byte(highIrrelevance), 0o644,
	)).To(Succeed())

	cfg := maintain.Config{
		Policy:  policy.Defaults(),
		DataDir: dataDir,
		Caller:  nil,
	}

	proposals, err := maintain.Run(context.Background(), cfg)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Should contain a gate-accuracy recommend proposal.
	var gateProposal *maintain.Proposal

	for idx := range proposals {
		if proposals[idx].ID == "gate-accuracy" {
			gateProposal = &proposals[idx]

			break
		}
	}

	g.Expect(gateProposal).NotTo(BeNil())

	if gateProposal == nil {
		return
	}

	g.Expect(gateProposal.Action).To(Equal(maintain.ActionRecommend))
	g.Expect(gateProposal.Rationale).To(ContainSubstring("SurfaceGateHaikuPrompt"))
}

func TestRun_MissingMemoryDir(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	// Don't create the memories subdirectory.

	cfg := maintain.Config{
		Policy:  policy.Defaults(),
		DataDir: dataDir,
	}

	_, err := maintain.Run(context.Background(), cfg)
	g.Expect(err).To(HaveOccurred())
}

func TestRun_NoCaller_UpdateProposalsHaveEmptyValue(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")

	err := os.MkdirAll(memDir, 0o755)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	err = os.WriteFile(
		filepath.Join(memDir, "needs-rewrite.toml"),
		[]byte(needsRewriteMemory), 0o644,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	cfg := maintain.Config{
		Policy:  policy.Defaults(),
		DataDir: dataDir,
		Caller:  nil, // no LLM — decision tree only
	}

	proposals, err := maintain.Run(context.Background(), cfg)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(proposals).NotTo(BeEmpty())

	if len(proposals) == 0 {
		return
	}

	// Without a caller, update proposals should still have empty Value.
	g.Expect(proposals[0].Action).To(Equal(maintain.ActionUpdate))
	g.Expect(proposals[0].Value).To(BeEmpty())
}

func TestRun_ProducesProposals(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")

	err := os.MkdirAll(memDir, 0o755)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	err = os.WriteFile(
		filepath.Join(memDir, "high-irrelevance.toml"),
		[]byte(highIrrelevanceMemory), 0o644,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	err = os.WriteFile(filepath.Join(memDir, "working.toml"), []byte(workingMemory), 0o644)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	cfg := maintain.Config{
		Policy:  policy.Defaults(),
		DataDir: dataDir,
		Caller:  nil, // no LLM — decision tree only
	}

	proposals, err := maintain.Run(context.Background(), cfg)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// The high-irrelevance memory produces a delete proposal + gate-accuracy recommend.
	g.Expect(proposals).To(HaveLen(2))
	g.Expect(proposals).NotTo(BeNil())

	if len(proposals) == 0 {
		return
	}

	g.Expect(proposals[0].Action).To(Equal(maintain.ActionDelete))
	g.Expect(proposals[0].Target).To(ContainSubstring("high-irrelevance"))
	g.Expect(proposals[1].ID).To(Equal("gate-accuracy"))
}

func TestRun_RewritesUpdateProposalValues(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")

	g.Expect(os.MkdirAll(memDir, 0o755)).To(Succeed())

	// This memory triggers priority 4b (rewrite action):
	// not_followed_rate = 5/8 = 62.5% (above default 50% threshold)
	g.Expect(os.WriteFile(
		filepath.Join(memDir, "needs-rewrite.toml"),
		[]byte(needsRewriteMemory), 0o644,
	)).To(Succeed())

	defaults := policy.Defaults()

	mockCaller := anthropic.CallerFunc(
		func(_ context.Context, _, systemPrompt, _ string) (string, error) {
			if systemPrompt == defaults.MaintainRewritePrompt {
				return "clearer action text from LLM", nil
			}

			return "[]", nil
		},
	)

	cfg := maintain.Config{
		Policy:  defaults,
		DataDir: dataDir,
		Caller:  mockCaller,
	}

	proposals, err := maintain.Run(context.Background(), cfg)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(proposals).NotTo(BeEmpty())

	updateProposal := findUpdateProposal(proposals, "action")
	g.Expect(updateProposal).NotTo(BeNil())

	if updateProposal == nil {
		return
	}

	g.Expect(updateProposal.Value).To(Equal("clearer action text from LLM"))
}

func TestRun_WithCaller_IncludesConsolidationAndAdapt(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dataDir := t.TempDir()
	memDir := filepath.Join(dataDir, "memories")

	err := os.MkdirAll(memDir, 0o755)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Write two memories so consolidation has enough records.
	err = os.WriteFile(filepath.Join(memDir, "mem-a.toml"), []byte(workingMemory), 0o644)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	err = os.WriteFile(filepath.Join(memDir, "mem-b.toml"), []byte(workingMemory), 0o644)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	defaults := policy.Defaults()

	mockCaller := anthropic.CallerFunc(
		func(_ context.Context, _, systemPrompt, _ string) (string, error) {
			if systemPrompt == defaults.MaintainConsolidatePrompt {
				return `[{"survivor":"mem-a.toml","members":["mem-a.toml","mem-b.toml"],"rationale":"similar"}]`, nil
			}

			return `[{"field":"maintain_min_surfaced","value":"10","rationale":"increase threshold"}]`, nil
		},
	)

	cfg := maintain.Config{
		Policy:        defaults,
		DataDir:       dataDir,
		Caller:        mockCaller,
		ChangeHistory: []policy.ChangeEntry{},
	}

	proposals, err := maintain.Run(context.Background(), cfg)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Decision tree produces 0 proposals (both working memories are healthy).
	// Consolidation produces 1, adapt produces 1.
	g.Expect(len(proposals)).To(BeNumerically(">=", 2))

	actions := make([]string, 0, len(proposals))
	for _, proposal := range proposals {
		actions = append(actions, proposal.Action)
	}

	g.Expect(actions).To(ContainElement(maintain.ActionMerge))
	g.Expect(actions).To(ContainElement(maintain.ActionUpdate))
}

// unexported constants.
const (
	highIrrelevanceMemory = `situation = "debugging Go tests"
behavior = "always restart from scratch"
impact = "wastes time"
action = "try incremental fixes first"
surfaced_count = 10
followed_count = 1
not_followed_count = 0
irrelevant_count = 7
`
	needsRewriteMemory = `situation = "reviewing pull requests"
behavior = "approve without reading diffs"
impact = "bugs reach production"
action = "read every diff line"
surfaced_count = 8
followed_count = 2
not_followed_count = 5
irrelevant_count = 1
`
	workingMemory = `situation = "writing new Go code"
behavior = "write tests first"
impact = "catches bugs early"
action = "use TDD"
surfaced_count = 10
followed_count = 8
not_followed_count = 1
irrelevant_count = 1
`
)

// findUpdateProposal returns the first update proposal matching the given field.
func findUpdateProposal(proposals []maintain.Proposal, field string) *maintain.Proposal {
	for idx := range proposals {
		if proposals[idx].Action == maintain.ActionUpdate &&
			proposals[idx].Field == field {
			return &proposals[idx]
		}
	}

	return nil
}
