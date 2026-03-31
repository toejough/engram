package maintain_test

import (
	"context"
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/maintain"
	"engram/internal/memory"
)

func TestFindConsolidationCandidates_CallerError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	caller := func(
		_ context.Context, _, _, _ string,
	) (string, error) {
		return "", errors.New("API unavailable")
	}

	records := []memory.StoredRecord{
		{Path: "a.toml", Record: memory.MemoryRecord{Situation: "s1", Action: "a1"}},
		{Path: "b.toml", Record: memory.MemoryRecord{Situation: "s2", Action: "a2"}},
	}

	consolidator := maintain.NewConsolidator(caller, "prompt")
	_, err := consolidator.FindMerges(context.Background(), records)
	g.Expect(err).To(MatchError(ContainSubstring("calling consolidation model")))
}

func TestFindConsolidationCandidates_EmptyInput(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	callerCalled := false

	caller := func(
		_ context.Context, _, _, _ string,
	) (string, error) {
		callerCalled = true

		return "", nil
	}

	consolidator := maintain.NewConsolidator(caller, "prompt")
	proposals, err := consolidator.FindMerges(context.Background(), nil)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(callerCalled).To(BeFalse())
	g.Expect(proposals).To(BeNil())
}

func TestFindConsolidationCandidates_EmptyResponse(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	caller := func(
		_ context.Context, _, _, _ string,
	) (string, error) {
		return "[]", nil
	}

	records := []memory.StoredRecord{
		{Path: "a.toml", Record: memory.MemoryRecord{Situation: "s1", Action: "a1"}},
		{Path: "b.toml", Record: memory.MemoryRecord{Situation: "s2", Action: "a2"}},
	}

	consolidator := maintain.NewConsolidator(caller, "prompt")
	proposals, err := consolidator.FindMerges(context.Background(), records)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(proposals).To(BeEmpty())
}

func TestFindConsolidationCandidates_InvalidJSON(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	caller := func(
		_ context.Context, _, _, _ string,
	) (string, error) {
		return "this is not valid JSON at all", nil
	}

	records := []memory.StoredRecord{
		{Path: "a.toml", Record: memory.MemoryRecord{Situation: "s1", Action: "a1"}},
		{Path: "b.toml", Record: memory.MemoryRecord{Situation: "s2", Action: "a2"}},
	}

	consolidator := maintain.NewConsolidator(caller, "prompt")
	_, err := consolidator.FindMerges(context.Background(), records)
	g.Expect(err).To(MatchError(ContainSubstring("parsing consolidation response")))
}

func TestFindConsolidationCandidates_SimilarMemories(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	records := []memory.StoredRecord{
		{
			Path: "memories/use-descriptive-names.toml",
			Record: memory.MemoryRecord{
				Situation: "Writing Go code with variables",
				Action:    "Use descriptive variable names instead of single letters",
			},
		},
		{
			Path: "memories/avoid-short-vars.toml",
			Record: memory.MemoryRecord{
				Situation: "Naming variables in Go functions",
				Action:    "Avoid single-letter variable names; prefer descriptive names",
			},
		},
		{
			Path: "memories/run-tests.toml",
			Record: memory.MemoryRecord{
				Situation: "Before committing code",
				Action:    "Run the full test suite",
			},
		},
	}

	callerResponse := `[
		{
			"survivor": "memories/use-descriptive-names.toml",
			"members": [
				"memories/use-descriptive-names.toml",
				"memories/avoid-short-vars.toml"
			],
			"rationale": "Both memories address the same naming convention guidance"
		}
	]`

	callerCalled := false

	caller := func(
		_ context.Context, model, systemPrompt, userPrompt string,
	) (string, error) {
		callerCalled = true

		g.Expect(model).To(Equal("claude-sonnet-4-20250514"))
		g.Expect(systemPrompt).To(Equal("test-system-prompt"))
		g.Expect(userPrompt).To(ContainSubstring("use-descriptive-names.toml"))
		g.Expect(userPrompt).To(ContainSubstring("avoid-short-vars.toml"))
		g.Expect(userPrompt).To(ContainSubstring("run-tests.toml"))

		return callerResponse, nil
	}

	consolidator := maintain.NewConsolidator(caller, "test-system-prompt")
	proposals, err := consolidator.FindMerges(context.Background(), records)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(callerCalled).To(BeTrue())
	g.Expect(proposals).To(HaveLen(1))

	if len(proposals) == 0 {
		return
	}

	g.Expect(proposals[0].Action).To(Equal(maintain.ActionMerge))
	g.Expect(proposals[0].Target).To(Equal("memories/use-descriptive-names.toml"))
	g.Expect(proposals[0].Related).To(Equal([]string{"memories/avoid-short-vars.toml"}))
	g.Expect(proposals[0].Rationale).To(Equal(
		"Both memories address the same naming convention guidance",
	))
}

func TestFindConsolidationCandidates_SingleRecord(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	callerCalled := false

	caller := func(
		_ context.Context, _, _, _ string,
	) (string, error) {
		callerCalled = true

		return "", nil
	}

	records := []memory.StoredRecord{
		{
			Path:   "memories/only-one.toml",
			Record: memory.MemoryRecord{Situation: "only one", Action: "do thing"},
		},
	}

	consolidator := maintain.NewConsolidator(caller, "prompt")
	proposals, err := consolidator.FindMerges(context.Background(), records)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(callerCalled).To(BeFalse())
	g.Expect(proposals).To(BeNil())
}
