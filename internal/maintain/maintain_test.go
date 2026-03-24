package maintain_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/onsi/gomega"

	"engram/internal/maintain"
	"engram/internal/memory"
	"engram/internal/review"
)

// T-174: Hidden gem produces LLM-powered broadening proposal.
func TestGenerate_HiddenGemBroaden(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	llmResponse := `{"additional_keywords": ["debugging", "profiling"], "rationale": "Useful in more contexts"}`

	fakeLLM := func(_ context.Context, _, _, _ string) (string, error) {
		return llmResponse, nil
	}

	gen := maintain.New(
		maintain.WithLLMCaller(fakeLLM),
		maintain.WithNow(fixedNow),
	)

	classified := []review.ClassifiedMemory{
		{Name: "gem-mem", Quadrant: review.HiddenGem},
	}
	memories := map[string]*memory.Stored{
		"gem-mem": {
			Title:    "Good memory",
			Keywords: []string{"testing"},
		},
	}

	proposals := gen.Generate(context.Background(), classified, memories)

	g.Expect(proposals).To(gomega.HaveLen(1))
	g.Expect(proposals[0].MemoryPath).To(gomega.Equal("gem-mem"))
	g.Expect(proposals[0].Quadrant).To(gomega.Equal("Hidden Gem"))
	g.Expect(proposals[0].Action).To(gomega.Equal("broaden_keywords"))
	g.Expect(proposals[0].Details).NotTo(gomega.BeEmpty())
}

// T-370: Non-JSON LLM response produces valid JSON Details (not a marshal crash).
func TestGenerate_HiddenGem_NonJSONResponse_WrapsAsString(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	nonJSONResponse := "# Here are my suggestions\n- Add keyword: debugging"

	fakeLLM := func(_ context.Context, _, _, _ string) (string, error) {
		return nonJSONResponse, nil
	}

	gen := maintain.New(
		maintain.WithLLMCaller(fakeLLM),
		maintain.WithNow(fixedNow),
	)

	classified := []review.ClassifiedMemory{
		{Name: "gem-mem", Quadrant: review.HiddenGem},
	}
	memories := map[string]*memory.Stored{
		"gem-mem": {
			Title:    "Good memory",
			Keywords: []string{"testing"},
		},
	}

	proposals := gen.Generate(context.Background(), classified, memories)

	g.Expect(proposals).To(gomega.HaveLen(1))

	// Details must be valid JSON — either the raw response if it was JSON,
	// or the response wrapped as a JSON string.
	detailsBytes, marshalErr := json.Marshal(proposals[0].Details)
	g.Expect(marshalErr).NotTo(gomega.HaveOccurred())

	if marshalErr != nil {
		return
	}

	g.Expect(detailsBytes).NotTo(gomega.BeEmpty())
}

// T-343: High-irrelevance memory produces refine_keywords proposal.
func TestGenerate_HighIrrelevance_ProposesRefineKeywords(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	gen := maintain.New(maintain.WithNow(fixedNow))

	classified := []review.ClassifiedMemory{
		{Name: "generic-mem", Quadrant: review.Working},
	}
	memories := map[string]*memory.Stored{
		"generic-mem": {
			Title:           "Generic Memory",
			Keywords:        []string{"code"},
			FollowedCount:   3,
			IrrelevantCount: 8,
			UpdatedAt:       fixedNow(),
		},
	}

	proposals := gen.Generate(context.Background(), classified, memories)

	g.Expect(proposals).To(gomega.HaveLen(1))
	g.Expect(proposals[0].MemoryPath).To(gomega.Equal("generic-mem"))
	g.Expect(proposals[0].Action).To(gomega.Equal("refine_keywords"))
	g.Expect(proposals[0].Diagnosis).To(gomega.ContainSubstring("irrelevant"))
}

// T-176: Insufficient data memory produces no proposal.
func TestGenerate_InsufficientDataSkipped(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	gen := maintain.New(maintain.WithNow(fixedNow))

	classified := []review.ClassifiedMemory{
		{Name: "new-mem", Quadrant: review.InsufficientData},
	}
	memories := map[string]*memory.Stored{
		"new-mem": {},
	}

	proposals := gen.Generate(context.Background(), classified, memories)

	g.Expect(proposals).To(gomega.BeEmpty())
}

// T-343d: Memory with insufficient feedback skips irrelevance check.
func TestGenerate_InsufficientFeedback_SkipsIrrelevanceCheck(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	now := fixedNow()

	const daysAgo = 91

	gen := maintain.New(maintain.WithNow(func() time.Time { return now }))

	classified := []review.ClassifiedMemory{
		{Name: "few-feedback", Quadrant: review.Working},
	}
	memories := map[string]*memory.Stored{
		"few-feedback": {
			Title:           "Few Feedback Memory",
			FollowedCount:   1,
			IrrelevantCount: 3,
			// Total = 4, below refineKeywordsMinFeedback (5).
			UpdatedAt: now.AddDate(0, 0, -daysAgo),
		},
	}

	proposals := gen.Generate(context.Background(), classified, memories)

	// Should fall through to Working handler (stale) — not refine_keywords.
	g.Expect(proposals).To(gomega.HaveLen(1))
	g.Expect(proposals[0].Action).To(gomega.Equal("review_staleness"))
}

// T-177: LLM failure for one memory does not block others.
func TestGenerate_LLMFailurePartial(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	callCount := 0
	fakeLLM := func(_ context.Context, _, _, _ string) (string, error) {
		callCount++
		if callCount == 1 {
			return "", errors.New("network timeout")
		}

		return `{"proposed_keywords": ["new"], "proposed_principle": "Be clear", "rationale": "Fix"}`, nil
	}

	gen := maintain.New(
		maintain.WithLLMCaller(fakeLLM),
		maintain.WithNow(fixedNow),
	)

	classified := []review.ClassifiedMemory{
		{Name: "leech-a", Quadrant: review.Leech},
		{Name: "leech-b", Quadrant: review.Leech},
	}
	memories := map[string]*memory.Stored{
		"leech-a": {Title: "First"},
		"leech-b": {Title: "Second"},
	}

	proposals := gen.Generate(context.Background(), classified, memories)

	g.Expect(proposals).To(gomega.HaveLen(1))
	g.Expect(proposals[0].MemoryPath).To(gomega.Equal("leech-b"))
}

// T-173: Leech memory produces LLM-powered rewrite proposal.
func TestGenerate_LeechRewrite(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	llmResponse := `{"proposed_keywords": ["testing", "quality"],` +
		` "proposed_principle": "Always test",` +
		` "rationale": "Current wording is vague"}`

	fakeLLM := func(_ context.Context, _, _, _ string) (string, error) {
		return llmResponse, nil
	}

	gen := maintain.New(
		maintain.WithLLMCaller(fakeLLM),
		maintain.WithNow(fixedNow),
	)

	classified := []review.ClassifiedMemory{
		{Name: "leech-mem", Quadrant: review.Leech},
	}
	memories := map[string]*memory.Stored{
		"leech-mem": {
			Title:    "Bad memory",
			Keywords: []string{"old"},
		},
	}

	proposals := gen.Generate(context.Background(), classified, memories)

	g.Expect(proposals).To(gomega.HaveLen(1))
	g.Expect(proposals[0].MemoryPath).To(gomega.Equal("leech-mem"))
	g.Expect(proposals[0].Quadrant).To(gomega.Equal("Leech"))
	g.Expect(proposals[0].Action).To(gomega.Equal("rewrite"))
	g.Expect(proposals[0].Details).NotTo(gomega.BeEmpty())
}

// T-371: Non-JSON LLM response for leech also produces valid JSON Details.
func TestGenerate_Leech_NonJSONResponse_WrapsAsString(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	nonJSONResponse := "The memory needs rewriting because..."

	fakeLLM := func(_ context.Context, _, _, _ string) (string, error) {
		return nonJSONResponse, nil
	}

	gen := maintain.New(
		maintain.WithLLMCaller(fakeLLM),
		maintain.WithNow(fixedNow),
	)

	classified := []review.ClassifiedMemory{
		{Name: "leech-mem", Quadrant: review.Leech},
	}
	memories := map[string]*memory.Stored{
		"leech-mem": {
			Title:    "Bad memory",
			Keywords: []string{"testing"},
		},
	}

	proposals := gen.Generate(context.Background(), classified, memories)

	g.Expect(proposals).To(gomega.HaveLen(1))

	detailsBytes, marshalErr := json.Marshal(proposals[0].Details)
	g.Expect(marshalErr).NotTo(gomega.HaveOccurred())

	if marshalErr != nil {
		return
	}

	g.Expect(detailsBytes).NotTo(gomega.BeEmpty())
}

// T-343b: Low-irrelevance memory does not trigger refine_keywords.
func TestGenerate_LowIrrelevance_NoRefineKeywords(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	gen := maintain.New(maintain.WithNow(fixedNow))

	classified := []review.ClassifiedMemory{
		{Name: "ok-mem", Quadrant: review.Working},
	}
	memories := map[string]*memory.Stored{
		"ok-mem": {
			Title:           "OK Memory",
			Keywords:        []string{"testing"},
			FollowedCount:   8,
			IrrelevantCount: 2,
			UpdatedAt:       fixedNow(),
		},
	}

	proposals := gen.Generate(context.Background(), classified, memories)

	// No refine_keywords — irrelevance ratio is 20% (below 60%).
	// Working with recent UpdatedAt should produce no proposal.
	g.Expect(proposals).To(gomega.BeEmpty())
}

// T-178: Nil LLM caller skips leech and hidden gem, produces only noise proposals.
func TestGenerate_NilLLMCallerSkipsLLMQuadrants(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	gen := maintain.New(maintain.WithNow(fixedNow))

	classified := []review.ClassifiedMemory{
		{Name: "leech-mem", Quadrant: review.Leech},
		{Name: "gem-mem", Quadrant: review.HiddenGem},
		{
			Name:               "noise-mem",
			Quadrant:           review.Noise,
			SurfacedCount:      1,
			EffectivenessScore: 10.0,
			EvaluationCount:    6,
		},
	}
	memories := map[string]*memory.Stored{
		"leech-mem": {},
		"gem-mem":   {},
		"noise-mem": {},
	}

	proposals := gen.Generate(context.Background(), classified, memories)

	g.Expect(proposals).To(gomega.HaveLen(1))
	g.Expect(proposals[0].MemoryPath).To(gomega.Equal("noise-mem"))
	g.Expect(proposals[0].Action).To(gomega.Equal("remove"))
}

// T-343c: Memory with nil stored skips irrelevance check without panic.
func TestGenerate_NilStored_SkipsIrrelevanceCheck(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	gen := maintain.New(maintain.WithNow(fixedNow))

	classified := []review.ClassifiedMemory{
		{Name: "missing-mem", Quadrant: review.Working},
	}
	// Memory not in map — stored will be nil.
	memories := map[string]*memory.Stored{}

	proposals := gen.Generate(context.Background(), classified, memories)

	// Should produce no proposal (nil stored falls through to Working handler which also nil-guards).
	g.Expect(proposals).To(gomega.BeEmpty())
}

// T-175: Noise memory produces removal proposal with evidence.
func TestGenerate_NoiseRemoval(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	gen := maintain.New(maintain.WithNow(fixedNow))

	classified := []review.ClassifiedMemory{
		{
			Name:               "noise-mem",
			Quadrant:           review.Noise,
			SurfacedCount:      2,
			EffectivenessScore: 15.0,
			EvaluationCount:    8,
		},
	}
	memories := map[string]*memory.Stored{
		"noise-mem": {},
	}

	proposals := gen.Generate(context.Background(), classified, memories)

	g.Expect(proposals).To(gomega.HaveLen(1))
	g.Expect(proposals[0].MemoryPath).To(gomega.Equal("noise-mem"))
	g.Expect(proposals[0].Quadrant).To(gomega.Equal("Noise"))
	g.Expect(proposals[0].Action).To(gomega.Equal("remove"))

	var details map[string]any

	err := json.Unmarshal(proposals[0].Details, &details)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(details["surfaced_count"]).To(gomega.BeNumerically("==", 2))
	g.Expect(details["effectiveness_score"]).To(gomega.BeNumerically("==", 15.0))
	g.Expect(details["evaluation_count"]).To(gomega.BeNumerically("==", 8))
}

func TestGenerate_Noise_ConsolidateProceed_ProposesRemoval(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	cons := &stubConsolidator{
		result: maintain.ConsolidateResult{Type: maintain.ConsolidateProceed},
	}
	loader := func(_ string) (*memory.MemoryRecord, error) {
		return &memory.MemoryRecord{Title: "Noisy"}, nil
	}

	gen := maintain.New(
		maintain.WithNow(fixedNow),
		maintain.WithConsolidator(cons, loader),
	)

	classified := []review.ClassifiedMemory{
		{Name: "noise-mem", Quadrant: review.Noise, SurfacedCount: 1},
	}
	memories := map[string]*memory.Stored{
		"noise-mem": {Title: "Noisy"},
	}

	proposals := gen.Generate(context.Background(), classified, memories)

	g.Expect(proposals).To(gomega.HaveLen(1))
	g.Expect(proposals[0].Action).To(gomega.Equal("remove"))
}

func TestGenerate_Noise_ConsolidateSkip_NoProposal(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	cons := &stubConsolidator{
		result: maintain.ConsolidateResult{Type: maintain.ConsolidateSkip},
	}
	loader := func(_ string) (*memory.MemoryRecord, error) {
		return &memory.MemoryRecord{Title: "Noisy"}, nil
	}

	gen := maintain.New(
		maintain.WithNow(fixedNow),
		maintain.WithConsolidator(cons, loader),
	)

	classified := []review.ClassifiedMemory{
		{Name: "noise-mem", Quadrant: review.Noise},
	}
	memories := map[string]*memory.Stored{
		"noise-mem": {Title: "Noisy"},
	}

	proposals := gen.Generate(context.Background(), classified, memories)

	g.Expect(proposals).To(gomega.BeEmpty())
}

func TestGenerate_Noise_ConsolidatorNil_ProposesRemoval(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	gen := maintain.New(maintain.WithNow(fixedNow))

	classified := []review.ClassifiedMemory{
		{Name: "noise-mem", Quadrant: review.Noise, SurfacedCount: 1},
	}
	memories := map[string]*memory.Stored{
		"noise-mem": {Title: "Noisy"},
	}

	proposals := gen.Generate(context.Background(), classified, memories)

	g.Expect(proposals).To(gomega.HaveLen(1))
	g.Expect(proposals[0].Action).To(gomega.Equal("remove"))
}

func TestGenerate_Noise_LoaderError_ProposesRemoval(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	cons := &stubConsolidator{
		result: maintain.ConsolidateResult{Type: maintain.ConsolidateSkip},
	}
	loader := func(_ string) (*memory.MemoryRecord, error) {
		return nil, errors.New("file not found")
	}

	gen := maintain.New(
		maintain.WithNow(fixedNow),
		maintain.WithConsolidator(cons, loader),
	)

	classified := []review.ClassifiedMemory{
		{Name: "noise-mem", Quadrant: review.Noise, SurfacedCount: 1},
	}
	memories := map[string]*memory.Stored{
		"noise-mem": {Title: "Noisy"},
	}

	proposals := gen.Generate(context.Background(), classified, memories)

	g.Expect(proposals).To(gomega.HaveLen(1))
	g.Expect(proposals[0].Action).To(gomega.Equal("remove"))
}

// T-172: Working memory beyond staleness threshold produces review_staleness proposal.
func TestGenerate_WorkingBeyondThreshold(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	now := fixedNow()

	const daysAgo = 91

	updatedAt := now.AddDate(0, 0, -daysAgo)

	gen := maintain.New(maintain.WithNow(func() time.Time { return now }))

	classified := []review.ClassifiedMemory{
		{Name: "stale-working", Quadrant: review.Working},
	}
	memories := map[string]*memory.Stored{
		"stale-working": {UpdatedAt: updatedAt},
	}

	proposals := gen.Generate(context.Background(), classified, memories)

	g.Expect(proposals).To(gomega.HaveLen(1))
	g.Expect(proposals[0].MemoryPath).To(gomega.Equal("stale-working"))
	g.Expect(proposals[0].Quadrant).To(gomega.Equal("Working"))
	g.Expect(proposals[0].Action).To(gomega.Equal("review_staleness"))

	var details map[string]any

	err := json.Unmarshal(proposals[0].Details, &details)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(details).To(gomega.HaveKey("age_days"))
}

// T-171: Working memory within staleness threshold produces no proposal.
func TestGenerate_WorkingWithinThreshold(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	now := fixedNow()

	const daysAgo = 89

	updatedAt := now.AddDate(0, 0, -daysAgo)

	gen := maintain.New(maintain.WithNow(func() time.Time { return now }))

	classified := []review.ClassifiedMemory{
		{Name: "fresh-working", Quadrant: review.Working},
	}
	memories := map[string]*memory.Stored{
		"fresh-working": {UpdatedAt: updatedAt},
	}

	proposals := gen.Generate(context.Background(), classified, memories)

	g.Expect(proposals).To(gomega.BeEmpty())
}

type stubConsolidator struct {
	result maintain.ConsolidateResult
	err    error
}

func (s *stubConsolidator) BeforeRemove(
	_ context.Context, _ *memory.MemoryRecord,
) (maintain.ConsolidateResult, error) {
	return s.result, s.err
}

// fixedNow returns a deterministic time for testing.
func fixedNow() time.Time {
	return time.Date(2026, 3, 7, 0, 0, 0, 0, time.UTC)
}
