package surface_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/memory"
	"engram/internal/surface"
)

func TestBuildGateUserPrompt_ContainsUserMessageAndSlugs(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	candidates := []*memory.Stored{
		{
			FilePath:  "mem/commit-safety.toml",
			Situation: "when committing",
			Behavior:  "manual commit",
			Impact:    "slow workflow",
			Action:    "use /commit",
		},
		{
			FilePath:  "mem/build-tools.toml",
			Situation: "when building",
			Behavior:  "go build directly",
			Impact:    "misses checks",
			Action:    "use targ build",
		},
	}

	prompt := surface.ExportBuildGateUserPrompt(candidates, "I want to commit code")

	g.Expect(prompt).To(ContainSubstring("I want to commit code"))
	g.Expect(prompt).To(ContainSubstring("commit-safety"))
	g.Expect(prompt).To(ContainSubstring("build-tools"))
	g.Expect(prompt).To(ContainSubstring("when committing"))
	g.Expect(prompt).To(ContainSubstring("use /commit"))
	g.Expect(strings.Count(prompt, "slug:")).To(Equal(2))
}

func TestFilterBySlug_AllMatch(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	candidates := []*memory.Stored{
		{FilePath: "mem/alpha.toml"},
		{FilePath: "mem/beta.toml"},
	}

	result := surface.ExportFilterBySlug(candidates, []string{"alpha", "beta"})

	g.Expect(result).To(HaveLen(2))
}

func TestFilterBySlug_EmptySlugsReturnsEmpty(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	candidates := []*memory.Stored{
		{FilePath: "mem/alpha.toml"},
	}

	result := surface.ExportFilterBySlug(candidates, []string{})

	g.Expect(result).To(BeEmpty())
}

func TestFilterBySlug_NoMatchesReturnsEmpty(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	candidates := []*memory.Stored{
		{FilePath: "mem/alpha.toml"},
		{FilePath: "mem/beta.toml"},
	}

	result := surface.ExportFilterBySlug(candidates, []string{"gamma"})

	g.Expect(result).To(BeEmpty())
}

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

func TestGateMemories_EmptyCandidates_ReturnsEmpty(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	callerCalled := false

	caller := func(_ context.Context, _, _, _ string) (string, error) {
		callerCalled = true

		return `[]`, nil
	}

	result, err := surface.GateMemories(
		context.Background(), []*memory.Stored{}, "any message", caller, "system",
	)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(BeEmpty())
	g.Expect(callerCalled).To(BeFalse())
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

func TestGateMemories_ParseError_ReturnsAllCandidates(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	candidates := []*memory.Stored{
		{FilePath: "mem/alpha.toml", Situation: "alpha"},
		{FilePath: "mem/beta.toml", Situation: "beta"},
	}

	caller := func(_ context.Context, _, _, _ string) (string, error) {
		return `not valid json`, nil
	}

	result, err := surface.GateMemories(
		context.Background(), candidates, "test message", caller, "system",
	)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(HaveLen(2))
}

func TestParseGateResponse_EmptyArray_ReturnsEmptySlice(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	slugs, err := surface.ExportParseGateResponse(`[]`)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(slugs).To(BeEmpty())
}

func TestParseGateResponse_InvalidJSON_ReturnsError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	slugs, err := surface.ExportParseGateResponse("not valid json at all")

	g.Expect(err).To(HaveOccurred())

	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("parsing gate response"))
	}

	g.Expect(slugs).To(BeNil())
}

func TestParseGateResponse_ValidJSON_ReturnsSlugs(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	slugs, err := surface.ExportParseGateResponse(`["commit-safety", "build-tools"]`)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(slugs).To(ConsistOf("commit-safety", "build-tools"))
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
