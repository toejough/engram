package cycle_test

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cycle"
)

type fakeRunner struct {
	calls     []string
	responses []string
}

func (f *fakeRunner) Run(_ context.Context, prompt string) (string, error) {
	idx := len(f.calls)
	f.calls = append(f.calls, prompt)

	if idx >= len(f.responses) {
		return "", nil
	}

	return f.responses[idx], nil
}

type fakeTranscript struct {
	content string
}

func (f *fakeTranscript) Read(_ string, _ int) (string, error) {
	return f.content, nil
}

type feedbackCall struct {
	Situation, Behavior, Impact, Action string
}

type factCall struct {
	Situation, Subject, Predicate, Object string
}

type fakePersister struct {
	feedbackCalls []feedbackCall
	factCalls     []factCall
}

func (f *fakePersister) WriteFeedback(
	_ context.Context,
	situation, behavior, impact, action string,
) (string, bool, error) {
	f.feedbackCalls = append(f.feedbackCalls, feedbackCall{
		Situation: situation, Behavior: behavior, Impact: impact, Action: action,
	})

	return situation, true, nil
}

func (f *fakePersister) WriteFact(
	_ context.Context,
	situation, subject, predicate, object string,
) (string, bool, error) {
	f.factCalls = append(f.factCalls, factCall{
		Situation: situation, Subject: subject, Predicate: predicate, Object: object,
	})

	return situation, true, nil
}

type fakeRecaller struct {
	reports map[string]string
}

func (f *fakeRecaller) Recall(_ context.Context, _, query string) (string, error) {
	return f.reports[query], nil
}

func TestCycle_EmptyTranscriptReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	runner := &fakeRunner{responses: []string{`[]`, `NO QUERIES`}}
	transcripts := &fakeTranscript{content: ""}

	c := cycle.New(runner, transcripts, nil, nil)

	out, err := c.Run(context.Background(), "/tmp/anything")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out.Learned).To(BeEmpty())
	g.Expect(out.Recalled).To(BeEmpty())
}

func TestCycle_PersistsLearnedFromLLMResponseA(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	llmA := `[{"type":"feedback","situation":"doing X","behavior":"b","impact":"i","action":"a"}]`
	runner := &fakeRunner{responses: []string{llmA, "NO QUERIES"}}
	transcripts := &fakeTranscript{content: "USER: did X\nASSISTANT: ok"}

	persister := &fakePersister{}
	c := cycle.New(runner, transcripts, persister, nil)

	out, err := c.Run(context.Background(), "/tmp/x")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(persister.feedbackCalls).To(HaveLen(1))
	g.Expect(persister.feedbackCalls[0].Situation).To(Equal("doing X"))
	g.Expect(out.Learned).To(HaveLen(1))
}

func TestCycle_RunsRecallPerProposedQuery(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	runner := &fakeRunner{responses: []string{
		"query one\nquery two",
	}}
	transcripts := &fakeTranscript{content: "transcript"}
	recaller := &fakeRecaller{reports: map[string]string{
		"query one": "report one",
		"query two": "report two",
	}}

	c := cycle.New(runner, transcripts, nil, recaller)

	out, err := c.Run(context.Background(), "/tmp/x")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out.Recalled).To(HaveLen(2))
	g.Expect(out.Recalled[0].Query).To(Equal("query one"))
	g.Expect(out.Recalled[0].Report).To(Equal("report one"))
}
