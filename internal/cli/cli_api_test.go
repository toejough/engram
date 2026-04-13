package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"engram/internal/apiclient"
	"engram/internal/cli"

	. "github.com/onsi/gomega"
	"github.com/toejough/imptest/match"
	"pgregory.net/rapid"
)

//go:generate impgen apiclient.API --dependency --import-path engram/internal/apiclient

// NOTE: impgen --target can't find test-file exports, so the StartExportDoLearn wrapper
// is hand-written in generated_StartExportDoLearn_test.go following the impgen pattern.

type API = apiclient.API

type PostMessageRequest = apiclient.PostMessageRequest

type PostMessageResponse = apiclient.PostMessageResponse

type StatusResponse = apiclient.StatusResponse

type SubscribeRequest = apiclient.SubscribeRequest

type SubscribeResponse = apiclient.SubscribeResponse

type WaitRequest = apiclient.WaitRequest

type WaitResponse = apiclient.WaitResponse

func TestDoIntent_AlwaysPostsThenWaitsAndReturnsSurfacedMemory(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		from := rapid.StringMatching(`[a-z][a-z0-9\-]{1,15}`).Draw(rt, "from")
		toAgent := rapid.StringMatching(`[a-z][a-z0-9\-]{1,15}`).Draw(rt, "to")
		situation := rapid.StringMatching(`[A-Za-z0-9 .,!]{1,80}`).Draw(rt, "situation")
		plannedAction := rapid.StringMatching(`[A-Za-z0-9 .,!]{1,80}`).Draw(rt, "planned-action")
		postCursor := rapid.IntRange(0, 100000).Draw(rt, "post-cursor")
		memoryText := rapid.StringMatching(`[A-Za-z0-9 .,!]{1,120}`).Draw(rt, "memory-text")

		mock, imp := MockAPI(rt)

		var stdout bytes.Buffer

		expectedText := "situation: " + situation + "\nplanned-action: " + plannedAction

		call := StartExportDoIntent(
			rt, cli.ExportDoIntent, rt.Context(), mock,
			from, toAgent, situation, plannedAction, &stdout,
		)

		// Step 1: PostMessage is called with from/to and composed text.
		imp.PostMessage.ArgsShould(
			match.BeAny,
			Equal(apiclient.PostMessageRequest{From: from, To: toAgent, Text: expectedText}),
		).Return(apiclient.PostMessageResponse{Cursor: postCursor}, nil)

		// Step 2: WaitForResponse is called with from/to swapped and cursor from post.
		imp.WaitForResponse.ArgsShould(
			match.BeAny,
			Equal(apiclient.WaitRequest{From: toAgent, To: from, AfterCursor: postCursor}),
		).Return(apiclient.WaitResponse{Text: memoryText}, nil)

		call.ReturnsShould(BeNil())

		g.Expect(stdout.String()).To(Equal(memoryText + "\n"))
	})
}

func TestDoIntent_WhenPostErrors_ReturnsWrappedError(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		apiErr := errors.New("connection refused")
		mock, imp := MockAPI(rt)

		var stdout bytes.Buffer

		call := StartExportDoIntent(
			rt, cli.ExportDoIntent, rt.Context(), mock,
			"sender", "engram", "testing", "do-thing", &stdout,
		)

		imp.PostMessage.ArgsShould(match.BeAny, match.BeAny).Return(
			apiclient.PostMessageResponse{},
			apiErr,
		)

		call.ReturnsShould(MatchError(ContainSubstring("connection refused")))
	})
}

func TestDoIntent_WhenWaitErrors_ReturnsWrappedError(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		waitErr := errors.New("timeout waiting")
		mock, imp := MockAPI(rt)

		var stdout bytes.Buffer

		call := StartExportDoIntent(
			rt, cli.ExportDoIntent, rt.Context(), mock,
			"sender", "engram", "testing", "do-thing", &stdout,
		)

		imp.PostMessage.ArgsShould(match.BeAny, match.BeAny).Return(
			apiclient.PostMessageResponse{Cursor: 5}, nil,
		)

		imp.WaitForResponse.ArgsShould(match.BeAny, match.BeAny).Return(
			apiclient.WaitResponse{},
			waitErr,
		)

		call.ReturnsShould(MatchError(ContainSubstring("timeout waiting")))
	})
}

func TestDoLearn_AlwaysPostsToEngramAgent(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		from := rapid.StringMatching(`[a-z][a-z0-9\-]{1,15}`).Draw(rt, "from")
		learnType := rapid.SampledFrom([]string{"feedback", "fact"}).Draw(rt, "type")
		cursor := rapid.IntRange(0, 100000).Draw(rt, "cursor")

		mock, imp := MockAPI(rt)

		var stdout bytes.Buffer

		call := StartExportDoLearn(
			rt, cli.ExportDoLearn, rt.Context(), mock,
			from, learnType, "sit", "beh", "imp", "act",
			"sub", "pred", "obj", &stdout,
		)

		imp.PostMessage.ArgsShould(
			match.BeAny,
			WithTransform(
				func(r apiclient.PostMessageRequest) string { return r.To },
				Equal("engram-agent"),
			),
		).Return(apiclient.PostMessageResponse{Cursor: cursor}, nil)

		call.ReturnsShould(BeNil())
	})
}

func TestDoLearn_FactAlwaysIncludesAllFields(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		from := rapid.StringMatching(`[a-z][a-z0-9\-]{1,15}`).Draw(rt, "from")
		situation := rapid.StringMatching(`[A-Za-z0-9 ]{1,40}`).Draw(rt, "situation")
		subject := rapid.StringMatching(`[A-Za-z0-9 ]{1,40}`).Draw(rt, "subject")
		predicate := rapid.StringMatching(`[A-Za-z0-9 ]{1,40}`).Draw(rt, "predicate")
		object := rapid.StringMatching(`[A-Za-z0-9 ]{1,40}`).Draw(rt, "object")
		cursor := rapid.IntRange(0, 100000).Draw(rt, "cursor")

		mock, imp := MockAPI(rt)

		var stdout bytes.Buffer

		call := StartExportDoLearn(
			rt, cli.ExportDoLearn, rt.Context(), mock,
			from, "fact", situation, "", "", "",
			subject, predicate, object, &stdout,
		)

		postCall := imp.PostMessage.ArgsShould(match.BeAny, match.BeAny)
		gotArgs := postCall.GetArgs()

		g.Expect(gotArgs.Req.Text).To(ContainSubstring(`"type":"fact"`))
		g.Expect(gotArgs.Req.Text).To(ContainSubstring(`"situation"`))
		g.Expect(gotArgs.Req.Text).To(ContainSubstring(`"subject"`))
		g.Expect(gotArgs.Req.Text).To(ContainSubstring(`"predicate"`))
		g.Expect(gotArgs.Req.Text).To(ContainSubstring(`"object"`))

		postCall.Return(apiclient.PostMessageResponse{Cursor: cursor}, nil)

		call.ReturnsShould(BeNil())

		g.Expect(stdout.String()).To(Equal(fmt.Sprintf("%d\n", cursor)))
	})
}

func TestDoLearn_FeedbackAlwaysIncludesAllFields(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		from := rapid.StringMatching(`[a-z][a-z0-9\-]{1,15}`).Draw(rt, "from")
		situation := rapid.StringMatching(`[A-Za-z0-9 ]{1,40}`).Draw(rt, "situation")
		behavior := rapid.StringMatching(`[A-Za-z0-9 ]{1,40}`).Draw(rt, "behavior")
		impact := rapid.StringMatching(`[A-Za-z0-9 ]{1,40}`).Draw(rt, "impact")
		action := rapid.StringMatching(`[A-Za-z0-9 ]{1,40}`).Draw(rt, "action")
		cursor := rapid.IntRange(0, 100000).Draw(rt, "cursor")

		mock, imp := MockAPI(rt)

		var stdout bytes.Buffer

		call := StartExportDoLearn(
			rt, cli.ExportDoLearn, rt.Context(), mock,
			from, "feedback", situation, behavior, impact, action,
			"", "", "", &stdout,
		)

		postCall := imp.PostMessage.ArgsShould(match.BeAny, match.BeAny)
		gotArgs := postCall.GetArgs()

		g.Expect(gotArgs.Req.Text).To(ContainSubstring(`"type":"feedback"`))
		g.Expect(gotArgs.Req.Text).To(ContainSubstring(`"situation"`))
		g.Expect(gotArgs.Req.Text).To(ContainSubstring(`"behavior"`))
		g.Expect(gotArgs.Req.Text).To(ContainSubstring(`"impact"`))
		g.Expect(gotArgs.Req.Text).To(ContainSubstring(`"action"`))

		postCall.Return(apiclient.PostMessageResponse{Cursor: cursor}, nil)

		call.ReturnsShould(BeNil())

		g.Expect(stdout.String()).To(Equal(fmt.Sprintf("%d\n", cursor)))
	})
}

func TestDoLearn_InvalidTypeAlwaysErrors(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		// Generate a type that is neither "feedback" nor "fact"
		badType := rapid.StringMatching(`[a-z]{1,10}`).
			Filter(func(s string) bool { return s != "feedback" && s != "fact" }).
			Draw(rt, "bad-type")

		mock, _ := MockAPI(rt)

		var stdout bytes.Buffer

		call := StartExportDoLearn(
			rt, cli.ExportDoLearn, rt.Context(), mock,
			"sender", badType, "sit", "beh", "imp", "act",
			"sub", "pred", "obj", &stdout,
		)

		call.ReturnsShould(MatchError(ContainSubstring("--type must be 'feedback' or 'fact'")))
	})
}

func TestDoLearn_WhenAPIErrors_ReturnsWrappedError(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		apiErr := errors.New("connection refused")
		mock, imp := MockAPI(rt)

		var stdout bytes.Buffer

		call := StartExportDoLearn(
			rt, cli.ExportDoLearn, rt.Context(), mock,
			"sender", "feedback", "sit", "beh", "imp", "act",
			"", "", "", &stdout,
		)

		imp.PostMessage.ArgsShould(match.BeAny, match.BeAny).Return(
			apiclient.PostMessageResponse{},
			apiErr,
		)

		call.ReturnsShould(MatchError(ContainSubstring("connection refused")))
	})
}

func TestDoPost_AlwaysPassesFromToTextToAPI(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		from := rapid.StringMatching(`[a-z][a-z0-9\-]{1,15}`).Draw(rt, "from")
		toAgent := rapid.StringMatching(`[a-z][a-z0-9\-]{1,15}`).Draw(rt, "to")
		text := rapid.StringMatching(`[A-Za-z0-9 .,!]{1,80}`).Draw(rt, "text")
		cursor := rapid.IntRange(0, 100000).Draw(rt, "cursor")

		mock, imp := MockAPI(rt)

		var stdout bytes.Buffer

		call := StartExportDoPost(rt, cli.ExportDoPost, rt.Context(), mock, from, toAgent, text, &stdout)

		imp.PostMessage.ArgsShould(
			match.BeAny,
			Equal(apiclient.PostMessageRequest{From: from, To: toAgent, Text: text}),
		).Return(apiclient.PostMessageResponse{Cursor: cursor}, nil)

		call.ReturnsShould(BeNil())

		g.Expect(stdout.String()).To(Equal(fmt.Sprintf("%d\n", cursor)))
	})
}

func TestDoPost_WhenAPIErrors_ReturnsWrappedError(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		apiErr := errors.New("connection refused")
		mock, imp := MockAPI(rt)

		var stdout bytes.Buffer

		call := StartExportDoPost(rt, cli.ExportDoPost, rt.Context(), mock, "sender", "receiver", "hello", &stdout)

		imp.PostMessage.ArgsShould(match.BeAny, match.BeAny).Return(
			apiclient.PostMessageResponse{},
			apiErr,
		)

		call.ReturnsShould(MatchError(ContainSubstring("connection refused")))
	})
}

func TestRunAPIDispatch_UnknownCommand_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := cli.ExportRunAPIDispatch(context.Background(), "unknown-cmd", nil, &bytes.Buffer{})

	g.Expect(err).To(HaveOccurred())
}

func TestRunIntent_WiresHTTPClientAndCallsDoIntent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const testCursor = 10

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		writer.Header().Set("Content-Type", "application/json")

		switch {
		case req.Method == http.MethodPost && req.URL.Path == "/message":
			resp := apiclient.PostMessageResponse{Cursor: testCursor}
			data, _ := json.Marshal(resp)
			_, _ = writer.Write(data)
		case req.Method == http.MethodGet && req.URL.Path == "/wait-for-response":
			resp := apiclient.WaitResponse{Text: "relevant memory"}
			data, _ := json.Marshal(resp)
			_, _ = writer.Write(data)
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	var stdout bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "intent",
			"--from", "lead-1",
			"--to", "engram-agent",
			"--situation", "deploying",
			"--planned-action", "run tests",
			"--addr", server.URL,
		},
		&stdout,
		&bytes.Buffer{},
		nil,
	)

	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(stdout.String()).To(Equal("relevant memory\n"))
}

func TestRunLearn_InvalidTypeReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	var stdout bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "learn",
			"--from", "lead-1",
			"--type", "invalid",
			"--addr", server.URL,
		},
		&stdout,
		&bytes.Buffer{},
		nil,
	)

	g.Expect(err).To(MatchError(ContainSubstring("--type must be 'feedback' or 'fact'")))
}

func TestRunLearn_MissingFromReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var stdout bytes.Buffer

	err := cli.Run(
		[]string{"engram", "learn", "--type", "feedback"},
		&stdout,
		&bytes.Buffer{},
		nil,
	)

	g.Expect(err).To(MatchError(ContainSubstring("--from is required")))
}

func TestRunLearn_WiresHTTPClientAndCallsDoLearn(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const testCursor = 77

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		resp := apiclient.PostMessageResponse{Cursor: testCursor}
		data, _ := json.Marshal(resp)

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write(data)
	}))
	defer server.Close()

	var stdout bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "learn",
			"--from", "lead-1",
			"--type", "feedback",
			"--situation", "deploying",
			"--behavior", "skipped tests",
			"--impact", "breakage",
			"--action", "always run tests",
			"--addr", server.URL,
		},
		&stdout,
		&bytes.Buffer{},
		nil,
	)

	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(stdout.String()).To(Equal(fmt.Sprintf("%d\n", testCursor)))
}

func TestRunPost_WiresHTTPClientAndCallsDoPost(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const testCursor = 42

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		resp := apiclient.PostMessageResponse{Cursor: testCursor}
		data, _ := json.Marshal(resp)

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write(data)
	}))
	defer server.Close()

	var stdout bytes.Buffer

	err := cli.Run(
		[]string{"engram", "post", "--from", "alpha", "--to", "beta", "--text", "hello", "--addr", server.URL},
		&stdout,
		&bytes.Buffer{},
		nil,
	)

	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(stdout.String()).To(Equal(fmt.Sprintf("%d\n", testCursor)))
}
