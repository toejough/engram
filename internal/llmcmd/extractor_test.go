package llmcmd_test

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/llmcmd"
)

func TestExtractor_ExtractRelevant_PromptIncludesContentAndQuery(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Echo the entire prompt back so we can inspect it.
	ext := llmcmd.NewExtractor(llmcmd.New("cat"))

	out, err := ext.ExtractRelevant(context.Background(), "the content body", "the query body")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).To(ContainSubstring("the content body"))
	g.Expect(out).To(ContainSubstring("the query body"))
	g.Expect(out).To(ContainSubstring("Extract only content relevant"))
}

func TestExtractor_SummarizeFindings_PromptIncludesBufferAndQuery(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ext := llmcmd.NewExtractor(llmcmd.New("cat"))

	out, err := ext.SummarizeFindings(context.Background(), "buffer contents", "the query")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).To(ContainSubstring("buffer contents"))
	g.Expect(out).To(ContainSubstring("the query"))
}
