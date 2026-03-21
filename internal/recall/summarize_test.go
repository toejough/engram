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

	result, err := summarizer.ExtractRelevant(context.Background(), "full transcript", "error handling")
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

func TestSummarize_CallsHaikuCallerWithCorrectPrompts(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	caller := &fakeHaikuCaller{result: "summary of work done"}
	summarizer := recall.NewSummarizer(caller)

	result, err := summarizer.Summarize(context.Background(), "transcript content here")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(caller.called).To(BeTrue())
	g.Expect(caller.systemPrompt).To(ContainSubstring("Summarize these session transcripts"))
	g.Expect(caller.systemPrompt).To(ContainSubstring("No emoji"))
	g.Expect(caller.userPrompt).To(Equal("transcript content here"))
	g.Expect(result).To(Equal("summary of work done"))
}

func TestSummarize_NilCallerReturnsError(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	summarizer := recall.NewSummarizer(nil)

	_, err := summarizer.Summarize(context.Background(), "content")
	g.Expect(err).To(HaveOccurred())
}

func TestSummarize_ReturnsErrorOnCallerFailure(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	caller := &fakeHaikuCaller{err: errors.New("api timeout")}
	summarizer := recall.NewSummarizer(caller)

	_, err := summarizer.Summarize(context.Background(), "content")
	g.Expect(err).To(MatchError(ContainSubstring("api timeout")))
	g.Expect(err).To(MatchError(ContainSubstring("summarizing")))
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
