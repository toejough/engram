package enforce_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/enforce"
	"engram/internal/memory"
)

// These tests verify the enforce package's JudgeViolation at the unit level.
// T-36/37/38/39 integration behavior is tested via the surface package.

func TestJudgeViolation_EmptyAPIResponse(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	client := &fakeHTTPClient{
		response: makeResponse(`{"content": []}`), //nolint:bodyclose // closed by JudgeViolation
	}

	e := enforce.New(client)
	mem := &memory.Stored{Principle: "x", AntiPattern: "y"}

	violated, err := e.JudgeViolation(context.Background(), "Bash", "git commit", mem, "test-token")

	g.Expect(err).To(HaveOccurred())
	g.Expect(violated).To(BeFalse())
}

func TestJudgeViolation_MalformedJudgmentJSON(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	resp := makeResponse( //nolint:bodyclose // closed by JudgeViolation
		`{"content": [{"type": "text", "text": "not json"}]}`,
	)
	client := &fakeHTTPClient{response: resp}

	e := enforce.New(client)
	mem := &memory.Stored{Principle: "x", AntiPattern: "y"}

	violated, err := e.JudgeViolation(context.Background(), "Bash", "git commit", mem, "test-token")

	g.Expect(err).To(HaveOccurred())
	g.Expect(violated).To(BeFalse())
}

func TestJudgeViolation_MalformedOuterJSON(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	client := &fakeHTTPClient{
		response: makeResponse(`not valid json`), //nolint:bodyclose // closed by JudgeViolation
	}

	e := enforce.New(client)
	mem := &memory.Stored{Principle: "x", AntiPattern: "y"}

	violated, err := e.JudgeViolation(context.Background(), "Bash", "git commit", mem, "test-token")

	g.Expect(err).To(HaveOccurred())
	g.Expect(violated).To(BeFalse())
}

func TestJudgeViolation_MarkdownFenceNoClosing(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// Fence with no closing backticks — stripped up to the newline only.
	fenced := "```json\\n{\\\"violated\\\": false, \\\"reason\\\": \\\"not violated\\\"}"
	body := `{"content": [{"type": "text", "text": "` + fenced + `"}]}`
	client := &fakeHTTPClient{
		response: makeResponse(body), //nolint:bodyclose // closed by JudgeViolation
	}

	e := enforce.New(client)
	mem := &memory.Stored{
		Principle:   "use /commit for commits",
		AntiPattern: "manual git commit",
	}

	violated, err := e.JudgeViolation(
		context.Background(),
		"Bash",
		"git status",
		mem,
		"test-token",
	)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(violated).To(BeFalse())
}

func TestJudgeViolation_MarkdownFencedResponse(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// LLM occasionally wraps JSON in a markdown code fence — stripMarkdownFence handles this.
	fenced := "```json\\n{\\\"violated\\\": true, \\\"reason\\\": \\\"direct violation\\\"}\\n```"
	body := `{"content": [{"type": "text", "text": "` + fenced + `"}]}`
	client := &fakeHTTPClient{
		response: makeResponse(body), //nolint:bodyclose // closed by JudgeViolation
	}

	e := enforce.New(client)
	mem := &memory.Stored{
		Principle:   "use /commit for commits",
		AntiPattern: "manual git commit",
	}

	violated, err := e.JudgeViolation(
		context.Background(),
		"Bash",
		"git commit -m 'fix'",
		mem,
		"test-token",
	)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(violated).To(BeTrue())
}

func TestJudgeViolation_NoToken(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	client := &fakeHTTPClient{}
	e := enforce.New(client)
	mem := &memory.Stored{Principle: "x", AntiPattern: "y"}

	violated, err := e.JudgeViolation(context.Background(), "Bash", "git commit", mem, "")

	g.Expect(err).To(MatchError(enforce.ErrNoToken))
	g.Expect(violated).To(BeFalse())
	g.Expect(client.called).To(BeFalse())
}

func TestJudgeViolation_NotViolated(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	resp := makeResponse( //nolint:bodyclose // closed by JudgeViolation
		`{"content": [{"type": "text", "text": "{\"violated\": false, \"reason\": \"not a violation\"}"}]}`,
	)
	client := &fakeHTTPClient{response: resp}

	e := enforce.New(client)
	mem := &memory.Stored{
		Principle:   "use /commit for commits",
		AntiPattern: "manual git commit",
	}

	violated, err := e.JudgeViolation(context.Background(), "Bash", "git status", mem, "test-token")

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(violated).To(BeFalse())
}

func TestJudgeViolation_Timeout(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	// Create a context that's already expired.
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	client := &fakeHTTPClient{
		err: context.DeadlineExceeded,
	}
	e := enforce.New(client)
	mem := &memory.Stored{Principle: "x", AntiPattern: "y"}

	violated, err := e.JudgeViolation(ctx, "Bash", "git commit", mem, "test-token")

	g.Expect(err).To(HaveOccurred())
	g.Expect(violated).To(BeFalse())
}

func TestJudgeViolation_Violated(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	resp := makeResponse( //nolint:bodyclose // closed by JudgeViolation
		`{"content": [{"type": "text", "text": "{\"violated\": true, \"reason\": \"direct violation detected\"}"}]}`,
	)
	client := &fakeHTTPClient{response: resp}

	e := enforce.New(client)
	mem := &memory.Stored{
		Principle:   "use /commit for commits",
		AntiPattern: "manual git commit",
	}

	violated, err := e.JudgeViolation(
		context.Background(),
		"Bash",
		"git commit -m 'fix'",
		mem,
		"test-token",
	)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(violated).To(BeTrue())
}

// fakeHTTPClient is a test double for enforce.HTTPDoer.
type fakeHTTPClient struct {
	response *http.Response
	err      error
	called   bool
}

func (f *fakeHTTPClient) Do(_ *http.Request) (*http.Response, error) {
	f.called = true

	if f.err != nil {
		return nil, f.err
	}

	return f.response, nil
}

func makeResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}
}
