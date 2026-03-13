package merge_test

import (
	"context"
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/merge"
)

func TestMergePrinciples_CallsLLM(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	client := &fakeLLMClient{response: "merged principle"}
	merger := merge.New(client)

	result, err := merger.MergePrinciples(context.Background(), "old principle", "new principle")

	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result).To(Equal("merged principle"))
	g.Expect(client.called).To(BeTrue())
}

func TestMergePrinciples_PropagatesLLMError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	llmErr := errors.New("llm unavailable")
	client := &fakeLLMClient{err: llmErr}
	merger := merge.New(client)

	_, err := merger.MergePrinciples(context.Background(), "old", "new")

	g.Expect(err).To(MatchError(llmErr))
}

func TestNew_CreatesLLMMerger(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	client := &fakeLLMClient{response: "merged principle"}
	merger := merge.New(client)

	g.Expect(merger).NotTo(BeNil())
}

type fakeLLMClient struct {
	called   bool
	response string
	err      error
}

func (f *fakeLLMClient) Call(_ context.Context, _ string) (string, error) {
	f.called = true

	return f.response, f.err
}
