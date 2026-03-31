package surface_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/surface"
)

func TestGateMemories_CallerError_ReturnsAllCandidates(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	candidates := []*memory.Stored{
		{FilePath: "mem/commit-safety.toml", Situation: "committing code", Action: "use /commit"},
		{FilePath: "mem/build-tools.toml", Situation: "building", Action: "use targ build"},
	}

	caller := func(_ context.Context, _, _, _ string) (string, error) {
		return "", errors.New("API error")
	}

	result, err := surface.GateMemories(
		context.Background(), candidates, "I want to commit", caller, "system prompt",
	)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(HaveLen(2))
}

func TestGateMemories_EmptyResponse_ReturnsNone(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	candidates := []*memory.Stored{
		{FilePath: "mem/commit-safety.toml", Situation: "committing code", Action: "use /commit"},
		{FilePath: "mem/build-tools.toml", Situation: "building", Action: "use targ build"},
	}

	caller := func(_ context.Context, _, _, _ string) (string, error) {
		return `[]`, nil
	}

	result, err := surface.GateMemories(
		context.Background(), candidates, "I want to commit", caller, "system prompt",
	)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(BeEmpty())
}

func TestGateMemories_FiltersIrrelevant(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	candidates := []*memory.Stored{
		{FilePath: "mem/commit-safety.toml", Situation: "committing code", Action: "use /commit"},
		{FilePath: "mem/build-tools.toml", Situation: "building", Action: "use targ build"},
	}

	caller := func(_ context.Context, _, _, _ string) (string, error) {
		return `["commit-safety"]`, nil
	}

	result, err := surface.GateMemories(
		context.Background(), candidates, "I want to commit", caller, "system prompt",
	)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(HaveLen(1))
	g.Expect(result[0].FilePath).To(Equal("mem/commit-safety.toml"))
}

func TestWithHaikuGate_WiresCallerOnSurfacer(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var callerCalled bool

	caller := func(_ context.Context, _, _, _ string) (string, error) {
		callerCalled = true

		return `["commit-safety"]`, nil
	}

	memories := []*memory.Stored{
		{Situation: "commit context", Behavior: "bad commit", Action: "good commit",
			FilePath: "mem/commit-safety.toml"},
		{Situation: "build context", Behavior: "bad build", Action: "good build",
			FilePath: "mem/build-tools.toml"},
		{Situation: "review context", Behavior: "bad review", Action: "good review",
			FilePath: "mem/review.toml"},
		{Situation: "deploy context", Behavior: "bad deploy", Action: "good deploy",
			FilePath: "mem/deploy.toml"},
	}

	retriever := &fakeRetriever{memories: memories}
	surfacer := surface.New(
		retriever,
		surface.WithHaikuGate(caller),
		surface.WithSurfaceConfig(surface.SurfaceConfig{
			BM25Threshold:       0.0,
			CandidateCountMax:   8,
			IrrelevanceHalfLife: 5,
			GateHaikuPrompt:     "filter memories",
		}),
	)

	var buf bytes.Buffer

	err := surfacer.Run(context.Background(), &buf, surface.Options{
		Mode:    surface.ModePrompt,
		DataDir: "/data",
		Message: "commit build",
	})

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(callerCalled).To(BeTrue())
}
