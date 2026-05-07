package cycle_test

import (
	"context"
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cycle"
)

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

func TestCycle_LLMCallAFailureProducesEmptyLearned(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Runner errors on call 0 (learn extraction); call 1 (query proposal) succeeds.
	runner := &fakeRunner{
		errors:    []error{errors.New("llm failed"), nil},
		responses: []string{"", "NO QUERIES"},
	}
	transcripts := &fakeTranscript{content: "anything"}

	c := cycle.New(runner, transcripts, &fakePersister{}, &fakeRecaller{})

	out, err := c.Run(context.Background(), "/tmp/x")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out.Learned).To(BeEmpty())
}

func TestCycle_LLMCallBFailureProducesEmptyRecalled(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Call 0 (learn extraction) returns []. Call 1 (query proposal) errors.
	runner := &fakeRunner{
		responses: []string{"[]", ""},
		errors:    []error{nil, errors.New("llm B failed")},
	}
	transcripts := &fakeTranscript{content: "anything"}

	c := cycle.New(runner, transcripts, &fakePersister{}, &fakeRecaller{})

	out, err := c.Run(context.Background(), "/tmp/x")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out.Recalled).To(BeEmpty())
}

func TestCycle_PerQueryRecallFailureSkipsEntry(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Two queries proposed. Recaller fails on "bad query" but succeeds on "good query".
	runner := &fakeRunner{responses: []string{"good query\nbad query"}}
	transcripts := &fakeTranscript{content: "anything"}
	recaller := &fakeRecaller{
		reports: map[string]string{"good query": "good report"},
		errors:  map[string]error{"bad query": errors.New("recall failed")},
	}

	c := cycle.New(runner, transcripts, nil, recaller)

	out, err := c.Run(context.Background(), "/tmp/x")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Only the successful query produces a RecalledReport.
	g.Expect(out.Recalled).To(HaveLen(1))
	g.Expect(out.Recalled[0].Query).To(Equal("good query"))
	g.Expect(out.Recalled[0].Report).To(Equal("good report"))
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

type factCall struct {
	Situation, Subject, Predicate, Object string
}

type fakePersister struct {
	feedbackCalls []feedbackCall
	factCalls     []factCall
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

func (f *fakePersister) WriteFeedback(
	_ context.Context,
	situation, behavior, impact, action string,
) (string, bool, error) {
	f.feedbackCalls = append(f.feedbackCalls, feedbackCall{
		Situation: situation, Behavior: behavior, Impact: impact, Action: action,
	})

	return situation, true, nil
}

type fakeRecaller struct {
	reports map[string]string
	errors  map[string]error // optional: if set for a query, returns that error
}

func (f *fakeRecaller) Recall(_ context.Context, _, query string) (string, error) {
	if f.errors != nil {
		if err, ok := f.errors[query]; ok && err != nil {
			return "", err
		}
	}

	return f.reports[query], nil
}

type fakeRunner struct {
	calls     []string
	responses []string
	errors    []error // index-aligned with responses; nil means no error
}

func (f *fakeRunner) Run(_ context.Context, prompt string) (string, error) {
	idx := len(f.calls)
	f.calls = append(f.calls, prompt)

	var err error
	if idx < len(f.errors) {
		err = f.errors[idx]
	}

	if err != nil {
		return "", err
	}

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
