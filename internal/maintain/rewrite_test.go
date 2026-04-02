package maintain_test

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/anthropic"
	"engram/internal/maintain"
	"engram/internal/memory"
)

func TestBuildRewritePrompt_AllFields(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	record := &memory.MemoryRecord{
		Situation: "sit-val",
		Behavior:  "beh-val",
		Impact:    "imp-val",
		Action:    "act-val",
	}

	for _, field := range []string{"situation", "behavior", "impact", "action"} {
		prompt := maintain.BuildRewritePrompt(field, record)
		g.Expect(prompt).To(ContainSubstring(field))
	}

	// Unknown field produces empty current value.
	prompt := maintain.BuildRewritePrompt("unknown", record)
	g.Expect(prompt).To(ContainSubstring("Current value: \"\""))
}

func TestBuildRewritePrompt_IncludesFieldAndRecord(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	record := &memory.MemoryRecord{
		Situation:        "writing Go code",
		Action:           "run tests first",
		SurfacedCount:    10,
		FollowedCount:    3,
		NotFollowedCount: 5,
		IrrelevantCount:  2,
	}

	prompt := maintain.BuildRewritePrompt("action", record)

	g.Expect(prompt).To(ContainSubstring("action"))
	g.Expect(prompt).To(ContainSubstring("run tests first"))
	g.Expect(prompt).To(ContainSubstring("10"))
	g.Expect(prompt).To(ContainSubstring("situation = \"writing Go code\""))
}

func TestRewriteProposals_CallerError_ReturnsOriginalProposals(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	records := []memory.StoredRecord{
		{
			Path: "memories/fail.toml",
			Record: memory.MemoryRecord{
				Situation:        "coding",
				Action:           "old action",
				SurfacedCount:    8,
				FollowedCount:    2,
				NotFollowedCount: 5,
				IrrelevantCount:  1,
			},
		},
	}

	proposals := []maintain.Proposal{
		{
			ID:        "diag-fail-rewrite",
			Action:    maintain.ActionUpdate,
			Target:    "memories/fail.toml",
			Field:     "action",
			Rationale: "not-followed rate high",
		},
	}

	mockCaller := anthropic.CallerFunc(
		func(_ context.Context, _, _, _ string) (string, error) {
			return "", anthropic.ErrAPIError
		},
	)

	rewriter := maintain.NewRewriter(mockCaller, "prompt")

	result, err := rewriter.RewriteProposals(
		context.Background(), proposals, records,
	)

	// Should return error but still return the original proposals unchanged.
	g.Expect(err).To(HaveOccurred())
	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(result).To(HaveLen(1))
	g.Expect(result[0].Value).To(BeEmpty())
}

func TestRewriteProposals_FillsValueForUpdateProposals(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	records := []memory.StoredRecord{
		{
			Path: "memories/narrow-me.toml",
			Record: memory.MemoryRecord{
				Situation:        "writing Go code",
				Behavior:         "skip tests",
				Impact:           "bugs slip through",
				Action:           "run tests first",
				SurfacedCount:    10,
				FollowedCount:    7,
				NotFollowedCount: 0,
				IrrelevantCount:  5,
			},
		},
		{
			Path: "memories/rewrite-me.toml",
			Record: memory.MemoryRecord{
				Situation:        "reviewing PRs",
				Behavior:         "approve without reading",
				Impact:           "bugs in production",
				Action:           "read every diff line",
				SurfacedCount:    8,
				FollowedCount:    2,
				NotFollowedCount: 5,
				IrrelevantCount:  1,
			},
		},
	}

	proposals := []maintain.Proposal{
		{
			ID:        "diag-narrow-me-narrow",
			Action:    maintain.ActionUpdate,
			Target:    "memories/narrow-me.toml",
			Field:     "situation",
			Rationale: "irrelevant rate 50% — situation too broad",
		},
		{
			ID:        "diag-rewrite-me-rewrite",
			Action:    maintain.ActionUpdate,
			Target:    "memories/rewrite-me.toml",
			Field:     "action",
			Rationale: "not-followed rate 63% — action unclear",
		},
		{
			ID:        "diag-delete-something",
			Action:    maintain.ActionDelete,
			Target:    "memories/bad.toml",
			Rationale: "ineffective",
		},
	}

	mockCaller := anthropic.CallerFunc(
		func(_ context.Context, _, _, userPrompt string) (string, error) {
			if userPrompt == "" {
				t.Error("empty user prompt")
			}

			return "improved field value from LLM", nil
		},
	)

	rewriter := maintain.NewRewriter(
		mockCaller,
		"You are rewriting a memory field.",
	)

	result, err := rewriter.RewriteProposals(
		context.Background(), proposals, records,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Should have same number of proposals.
	g.Expect(result).To(HaveLen(3))

	// Update proposals should now have Value filled.
	g.Expect(result[0].Value).To(Equal("improved field value from LLM"))
	g.Expect(result[1].Value).To(Equal("improved field value from LLM"))

	// Delete proposal should be unchanged.
	g.Expect(result[2].Action).To(Equal(maintain.ActionDelete))
	g.Expect(result[2].Value).To(BeEmpty())
}

func TestRewriteProposals_PartialFailure_PreservesSuccessfulRewrites(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	records := []memory.StoredRecord{
		{
			Path:   "memories/succeed.toml",
			Record: memory.MemoryRecord{Action: "old action", SurfacedCount: 8, NotFollowedCount: 5},
		},
		{
			Path:   "memories/fail.toml",
			Record: memory.MemoryRecord{Action: "other action", SurfacedCount: 8, NotFollowedCount: 5},
		},
	}

	proposals := []maintain.Proposal{
		{ID: "p1", Action: maintain.ActionUpdate, Target: "memories/succeed.toml", Field: "action"},
		{ID: "p2", Action: maintain.ActionUpdate, Target: "memories/fail.toml", Field: "action"},
	}

	callCount := 0

	mockCaller := anthropic.CallerFunc(
		func(_ context.Context, _, _, _ string) (string, error) {
			callCount++

			if callCount == 1 {
				return "rewritten value", nil
			}

			return "", anthropic.ErrAPIError
		},
	)

	rewriter := maintain.NewRewriter(mockCaller, "prompt")

	result, err := rewriter.RewriteProposals(context.Background(), proposals, records)

	// Should return error for the failure.
	g.Expect(err).To(HaveOccurred())
	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(result).To(HaveLen(2))

	// First proposal should have the successful rewrite preserved.
	g.Expect(result[0].Value).To(Equal("rewritten value"))

	// Second proposal should still have empty Value (LLM failed).
	g.Expect(result[1].Value).To(BeEmpty())
}

func TestRewriteProposals_SkipsProposalsWithExistingValue(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	records := []memory.StoredRecord{
		{
			Path: "memories/already-has-value.toml",
			Record: memory.MemoryRecord{
				Situation:     "testing",
				Action:        "old action",
				SurfacedCount: 10,
			},
		},
	}

	proposals := []maintain.Proposal{
		{
			ID:        "diag-already-has-value-rewrite",
			Action:    maintain.ActionUpdate,
			Target:    "memories/already-has-value.toml",
			Field:     "action",
			Value:     "already set",
			Rationale: "some rationale",
		},
	}

	callCount := 0

	mockCaller := anthropic.CallerFunc(
		func(_ context.Context, _, _, _ string) (string, error) {
			callCount++
			return "should not be called", nil
		},
	)

	rewriter := maintain.NewRewriter(mockCaller, "prompt")

	result, err := rewriter.RewriteProposals(
		context.Background(), proposals, records,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(HaveLen(1))
	g.Expect(result[0].Value).To(Equal("already set"))
	g.Expect(callCount).To(Equal(0))
}
