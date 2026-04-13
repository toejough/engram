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

type API = apiclient.API

type PostMessageRequest = apiclient.PostMessageRequest

type PostMessageResponse = apiclient.PostMessageResponse

type StatusResponse = apiclient.StatusResponse

type SubscribeRequest = apiclient.SubscribeRequest

type SubscribeResponse = apiclient.SubscribeResponse

type WaitRequest = apiclient.WaitRequest

type WaitResponse = apiclient.WaitResponse

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
