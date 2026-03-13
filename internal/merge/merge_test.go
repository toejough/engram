package merge_test

import (
	"context"
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/merge"
)

func TestMergePrinciples_PropagatesClientError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	client := &fakeLLMClient{err: errors.New("llm error")}
	m := merge.New(client)

	_, err := m.MergePrinciples(context.Background(), "existing", "candidate")
	g.Expect(err).To(MatchError(ContainSubstring("llm error")))
}

func TestMergePrinciples_ReturnsClientResponse(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	client := &fakeLLMClient{response: "combined principle"}
	m := merge.New(client)

	result, err := m.MergePrinciples(context.Background(), "existing", "candidate")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result).To(Equal("combined principle"))
}

func TestNew_ReturnsMerger(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	client := &fakeLLMClient{response: "merged"}
	m := merge.New(client)

	g.Expect(m).NotTo(BeNil())
}

// fakeLLMClient is a test double for LLMClient.
type fakeLLMClient struct {
	response string
	err      error
}

func (f *fakeLLMClient) Call(_ context.Context, _ string) (string, error) {
	return f.response, f.err
}
