package maintain_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/anthropic"
	"engram/internal/maintain"
	"engram/internal/policy"
)

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

	// Decision tree proposals are still returned alongside the error.
	g.Expect(proposals).To(HaveLen(1))

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

	// The high-irrelevance memory should produce a proposal; the working one should not.
	g.Expect(proposals).To(HaveLen(1))
	g.Expect(proposals).NotTo(BeNil())

	if len(proposals) == 0 {
		return
	}

	g.Expect(proposals[0].Action).To(Equal(maintain.ActionDelete))
	g.Expect(proposals[0].Target).To(ContainSubstring("high-irrelevance"))
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

	callCount := 0

	mockCaller := anthropic.CallerFunc(
		func(_ context.Context, _, _, _ string) (string, error) {
			callCount++

			// First call = consolidation, second call = adapt.
			if callCount == 1 {
				return `[{"survivor":"mem-a.toml","members":["mem-a.toml","mem-b.toml"],"rationale":"similar"}]`, nil
			}

			return `[{"field":"maintain_min_surfaced","value":"10","rationale":"increase threshold"}]`, nil
		},
	)

	cfg := maintain.Config{
		Policy:        policy.Defaults(),
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
