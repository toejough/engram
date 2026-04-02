package maintain_test

import (
	"context"
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/maintain"
	"engram/internal/memory"
	"engram/internal/policy"
)

func TestAdaptAnalyze_CallerError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	caller := func(
		_ context.Context, _, _, _ string,
	) (string, error) {
		return "", errors.New("API unavailable")
	}

	records := []memory.StoredRecord{
		{Path: "a.toml", Record: memory.MemoryRecord{Situation: "s1", Action: "a1"}},
	}

	adapter := maintain.NewAdapter(caller, "prompt")
	_, err := adapter.Analyze(context.Background(), records, policy.Defaults(), nil)
	g.Expect(err).To(MatchError(ContainSubstring("calling adapt model")))
}

func TestAdaptAnalyze_EmptyRecords(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	callerCalled := false

	caller := func(
		_ context.Context, _, _, _ string,
	) (string, error) {
		callerCalled = true

		return "", nil
	}

	adapter := maintain.NewAdapter(caller, "prompt")
	proposals, err := adapter.Analyze(context.Background(), nil, policy.Defaults(), nil)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(callerCalled).To(BeFalse())
	g.Expect(proposals).To(BeNil())
}

func TestAdaptAnalyze_EmptyResponse(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	caller := func(
		_ context.Context, _, _, _ string,
	) (string, error) {
		return "[]", nil
	}

	records := []memory.StoredRecord{
		{Path: "a.toml", Record: memory.MemoryRecord{Situation: "s1", Action: "a1"}},
	}

	adapter := maintain.NewAdapter(caller, "prompt")
	proposals, err := adapter.Analyze(context.Background(), records, policy.Defaults(), nil)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(proposals).To(BeEmpty())
}

func TestAdaptAnalyze_IncludesChangeHistory(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var capturedUserPrompt string

	caller := func(
		_ context.Context, _, _, userPrompt string,
	) (string, error) {
		capturedUserPrompt = userPrompt

		return "[]", nil
	}

	records := []memory.StoredRecord{
		{Path: "a.toml", Record: memory.MemoryRecord{Situation: "s1", Action: "a1"}},
	}

	changeHistory := []policy.ChangeEntry{
		{
			Action:    "update",
			Target:    policy.Filename,
			Field:     "SurfaceBM25Threshold",
			OldValue:  "0.3",
			NewValue:  "0.25",
			Status:    "applied",
			Rationale: "lower threshold helps",
			ChangedAt: "2026-03-30T10:00:00Z",
		},
	}

	adapter := maintain.NewAdapter(caller, "prompt")
	_, err := adapter.Analyze(
		context.Background(), records, policy.Defaults(), changeHistory,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(capturedUserPrompt).To(ContainSubstring("SurfaceBM25Threshold"))
	g.Expect(capturedUserPrompt).To(ContainSubstring("0.3"))
	g.Expect(capturedUserPrompt).To(ContainSubstring("0.25"))
	g.Expect(capturedUserPrompt).To(ContainSubstring("lower threshold helps"))
}

func TestAdaptAnalyze_InvalidJSON(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	caller := func(
		_ context.Context, _, _, _ string,
	) (string, error) {
		return "this is not valid JSON at all", nil
	}

	records := []memory.StoredRecord{
		{Path: "a.toml", Record: memory.MemoryRecord{Situation: "s1", Action: "a1"}},
	}

	adapter := maintain.NewAdapter(caller, "prompt")
	_, err := adapter.Analyze(context.Background(), records, policy.Defaults(), nil)
	g.Expect(err).To(MatchError(ContainSubstring("parsing adapt response")))
}

func TestAdaptAnalyze_ProposesParameterChange(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	callerResponse := `[
		{
			"field": "SurfaceBM25Threshold",
			"value": "0.25",
			"rationale": "Lower threshold would surface more relevant memories"
		}
	]`

	var capturedUserPrompt string

	caller := func(
		_ context.Context, model, systemPrompt, userPrompt string,
	) (string, error) {
		g.Expect(model).To(Equal("claude-sonnet-4-20250514"))
		g.Expect(systemPrompt).To(Equal("test-system-prompt"))

		capturedUserPrompt = userPrompt

		return callerResponse, nil
	}

	records := []memory.StoredRecord{
		{
			Path: "a.toml",
			Record: memory.MemoryRecord{
				Situation:        "coding Go",
				Action:           "use descriptive names",
				SurfacedCount:    10,
				FollowedCount:    6,
				NotFollowedCount: 2,
				IrrelevantCount:  2,
			},
		},
		{
			Path: "b.toml",
			Record: memory.MemoryRecord{
				Situation:        "writing tests",
				Action:           "add t.Parallel",
				SurfacedCount:    5,
				FollowedCount:    4,
				NotFollowedCount: 1,
				IrrelevantCount:  0,
			},
		},
	}

	pol := policy.Defaults()
	adapter := maintain.NewAdapter(caller, "test-system-prompt")

	proposals, err := adapter.Analyze(context.Background(), records, pol, nil)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(proposals).To(HaveLen(1))

	if len(proposals) == 0 {
		return
	}

	g.Expect(proposals[0].Action).To(Equal(maintain.ActionUpdate))
	g.Expect(proposals[0].Target).To(Equal(policy.Filename))
	g.Expect(proposals[0].Field).To(Equal("SurfaceBM25Threshold"))
	g.Expect(proposals[0].Value).To(Equal("0.25"))
	g.Expect(proposals[0].Rationale).To(Equal(
		"Lower threshold would surface more relevant memories",
	))

	// Verify user prompt contains aggregate metrics.
	g.Expect(capturedUserPrompt).To(ContainSubstring("15")) // total surfaced
	g.Expect(capturedUserPrompt).To(ContainSubstring("10")) // total followed
	g.Expect(capturedUserPrompt).To(ContainSubstring("SurfaceBM25Threshold"))
}
