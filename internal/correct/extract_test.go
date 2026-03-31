package correct_test

import (
	"context"
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/correct"
	"engram/internal/memory"
)

func TestExtract_EmptyCandidates(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	var capturedPrompt string

	responseJSON := `{
		"situation": "s",
		"behavior": "b",
		"impact": "i",
		"action": "a",
		"filename_slug": "slug",
		"project_scoped": false,
		"candidates": []
	}`

	mockCaller := func(_ context.Context, _, _, userPrompt string) (string, error) {
		capturedPrompt = userPrompt
		return responseJSON, nil
	}

	result, err := correct.Extract(
		context.Background(),
		mockCaller,
		"message",
		"context",
		nil,
		"system prompt",
	)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(capturedPrompt).NotTo(ContainSubstring("Existing similar memories"))
	g.Expect(result.Candidates).To(BeEmpty())
}

func TestExtract_ErrorOnInvalidJSON(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	mockCaller := func(_ context.Context, _, _, _ string) (string, error) {
		return "not valid json at all", nil
	}

	_, err := correct.Extract(
		context.Background(),
		mockCaller,
		"message",
		"context",
		nil,
		"system prompt",
	)

	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("extraction:"))
}

func TestExtract_HandlesJSONInMarkdownFence(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	responseWithFence := "```json\n" + `{
		"situation": "fenced situation",
		"behavior": "fenced behavior",
		"impact": "fenced impact",
		"action": "fenced action",
		"filename_slug": "fenced-slug",
		"project_scoped": true,
		"candidates": []
	}` + "\n```"

	mockCaller := func(_ context.Context, _, _, _ string) (string, error) {
		return responseWithFence, nil
	}

	result, err := correct.Extract(
		context.Background(),
		mockCaller,
		"message",
		"context",
		nil,
		"system prompt",
	)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(result.Situation).To(Equal("fenced situation"))
	g.Expect(result.ProjectScoped).To(BeTrue())
	g.Expect(result.FilenameSlug).To(Equal("fenced-slug"))
}

func TestExtract_HandlesJSONInPlainFence(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	responseWithFence := "```\n" + `{
		"situation": "plain fenced",
		"behavior": "plain behavior",
		"impact": "plain impact",
		"action": "plain action",
		"filename_slug": "plain-slug",
		"project_scoped": false,
		"candidates": []
	}` + "\n```"

	mockCaller := func(_ context.Context, _, _, _ string) (string, error) {
		return responseWithFence, nil
	}

	result, err := correct.Extract(
		context.Background(),
		mockCaller,
		"message",
		"context",
		nil,
		"system prompt",
	)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(result.Situation).To(Equal("plain fenced"))
	g.Expect(result.FilenameSlug).To(Equal("plain-slug"))
}

func TestExtract_ParsesSonnetResponse(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	responseJSON := `{
		"situation": "writing Go code",
		"behavior": "using short variable names",
		"impact": "code is harder to read",
		"action": "use descriptive variable names",
		"filename_slug": "descriptive-variable-names",
		"project_scoped": false,
		"candidates": []
	}`

	mockCaller := func(_ context.Context, _, _, _ string) (string, error) {
		return responseJSON, nil
	}

	result, err := correct.Extract(
		context.Background(),
		mockCaller,
		"always use descriptive variable names",
		"some transcript context",
		nil,
		"system prompt",
	)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(result.Situation).To(Equal("writing Go code"))
	g.Expect(result.Behavior).To(Equal("using short variable names"))
	g.Expect(result.Impact).To(Equal("code is harder to read"))
	g.Expect(result.Action).To(Equal("use descriptive variable names"))
	g.Expect(result.FilenameSlug).To(Equal("descriptive-variable-names"))
	g.Expect(result.ProjectScoped).To(BeFalse())
	g.Expect(result.Candidates).To(BeEmpty())
}

func TestExtract_PropagatesCallerError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	callerErr := errors.New("caller failed")

	mockCaller := func(_ context.Context, _, _, _ string) (string, error) {
		return "", callerErr
	}

	_, err := correct.Extract(
		context.Background(),
		mockCaller,
		"message",
		"context",
		nil,
		"system prompt",
	)

	g.Expect(err).To(MatchError(callerErr))
}

func TestExtract_WithCandidates(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	candidates := []*memory.Stored{
		{
			Situation: "writing Go code",
			Behavior:  "short names",
			Impact:    "hard to read",
			Action:    "use long names",
			FilePath:  "/memories/descriptive-names.toml",
		},
	}

	var capturedPrompt string

	responseJSON := `{
		"situation": "writing Go code",
		"behavior": "using short variable names",
		"impact": "code is harder to read",
		"action": "use descriptive variable names",
		"filename_slug": "descriptive-variable-names",
		"project_scoped": false,
		"candidates": [
			{"name": "descriptive-names", "disposition": "supersedes", "reason": "same topic, more specific"}
		]
	}`

	mockCaller := func(_ context.Context, _, _, userPrompt string) (string, error) {
		capturedPrompt = userPrompt
		return responseJSON, nil
	}

	result, err := correct.Extract(
		context.Background(),
		mockCaller,
		"always use descriptive variable names",
		"transcript",
		candidates,
		"system prompt",
	)

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).NotTo(BeNil())

	if result == nil {
		return
	}

	g.Expect(capturedPrompt).To(ContainSubstring("descriptive-names"))
	g.Expect(capturedPrompt).To(ContainSubstring("writing Go code"))

	g.Expect(result.Candidates).To(HaveLen(1))
	g.Expect(result.Candidates[0].Name).To(Equal("descriptive-names"))
	g.Expect(result.Candidates[0].Disposition).To(Equal("supersedes"))
	g.Expect(result.Candidates[0].Reason).To(Equal("same topic, more specific"))
}
