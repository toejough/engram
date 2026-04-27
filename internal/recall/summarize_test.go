package recall_test

import (
	"context"
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/recall"
)

func TestExtractRelevant_CallsHaikuCallerWithQueryInPrompt(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	caller := &fakeHaikuCaller{result: "relevant excerpt"}
	summarizer := recall.NewSummarizer(caller)

	result, err := summarizer.ExtractRelevant(
		context.Background(),
		"full transcript",
		"error handling",
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(caller.called).To(BeTrue())
	g.Expect(caller.systemPrompt).To(ContainSubstring("Extract only content relevant"))
	g.Expect(caller.userPrompt).To(ContainSubstring("error handling"))
	g.Expect(caller.userPrompt).To(ContainSubstring("full transcript"))
	g.Expect(result).To(Equal("relevant excerpt"))
}

func TestExtractRelevant_ReturnsErrorOnCallerFailure(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	caller := &fakeHaikuCaller{err: errors.New("rate limited")}
	summarizer := recall.NewSummarizer(caller)

	_, err := summarizer.ExtractRelevant(context.Background(), "content", "query")
	g.Expect(err).To(MatchError(ContainSubstring("rate limited")))
	g.Expect(err).To(MatchError(ContainSubstring("extracting relevant")))
}

func TestNoopSummarizer_ReturnsEmptyForBothMethods(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	noop := recall.NoopSummarizer{}

	extracted, extractErr := noop.ExtractRelevant(context.Background(), "content", "query")
	g.Expect(extractErr).NotTo(HaveOccurred())
	g.Expect(extracted).To(BeEmpty())

	summary, summaryErr := noop.SummarizeFindings(context.Background(), "content", "query")
	g.Expect(summaryErr).NotTo(HaveOccurred())
	g.Expect(summary).To(BeEmpty())
}

func TestSummarizeFindings_CallsHaikuCallerWithSummaryPrompt(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	caller := &fakeHaikuCaller{result: "structured summary"}
	summarizer := recall.NewSummarizer(caller)

	result, err := summarizer.SummarizeFindings(
		context.Background(),
		"memory excerpts and session snippets",
		"targ argument parsing",
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(caller.called).To(BeTrue())
	g.Expect(caller.systemPrompt).To(ContainSubstring("structured summary"))
	g.Expect(caller.userPrompt).To(ContainSubstring("targ argument parsing"))
	g.Expect(caller.userPrompt).To(ContainSubstring("memory excerpts"))
	g.Expect(result).To(Equal("structured summary"))
}

func TestSummarizeFindings_ReturnsErrorOnCallerFailure(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	caller := &fakeHaikuCaller{err: errors.New("rate limited")}
	summarizer := recall.NewSummarizer(caller)

	_, err := summarizer.SummarizeFindings(context.Background(), "content", "query")
	g.Expect(err).To(MatchError(ContainSubstring("rate limited")))
	g.Expect(err).To(MatchError(ContainSubstring("summarizing findings")))
}

// --- Fake implementations ---

type fakeHaikuCaller struct {
	result       string
	err          error
	systemPrompt string
	userPrompt   string
	called       bool
}

func (f *fakeHaikuCaller) Call(_ context.Context, system, user string) (string, error) {
	f.called = true
	f.systemPrompt = system
	f.userPrompt = user

	return f.result, f.err
}
