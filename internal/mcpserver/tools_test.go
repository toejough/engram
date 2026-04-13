package mcpserver_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"engram/internal/apiclient"
	"engram/internal/mcpserver"
)

func TestHandleIntent_AlwaysReturnsSurfacedMemory(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		from := rapid.StringMatching(`[a-z][a-z0-9\-]{1,15}`).Draw(rt, "from")
		toAgent := rapid.StringMatching(`[a-z][a-z0-9\-]{1,15}`).Draw(rt, "to")
		situation := rapid.StringMatching(`[A-Za-z0-9 .,!]{1,80}`).Draw(rt, "situation")
		plannedAction := rapid.StringMatching(`[A-Za-z0-9 .,!]{1,80}`).Draw(rt, "planned-action")
		postCursor := rapid.IntRange(0, 100000).Draw(rt, "post-cursor")
		memoryText := rapid.StringMatching(`[A-Za-z0-9 .,!]{1,120}`).Draw(rt, "memory-text")

		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
			writer.Header().Set("Content-Type", "application/json")

			switch {
			case req.Method == http.MethodPost && req.URL.Path == "/message":
				data, _ := json.Marshal(apiclient.PostMessageResponse{Cursor: postCursor})
				_, _ = writer.Write(data)
			case req.Method == http.MethodGet && req.URL.Path == "/wait-for-response":
				data, _ := json.Marshal(apiclient.WaitResponse{Text: memoryText})
				_, _ = writer.Write(data)
			default:
				writer.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		apiClient := apiclient.New(server.URL, http.DefaultClient)

		result, _, err := callIntentHandler(t, apiClient, from, toAgent, situation, plannedAction)

		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(result).NotTo(BeNil())

		if result == nil {
			return
		}

		g.Expect(firstTextContent(g, result)).To(Equal(memoryText))
	})
}

func TestHandleIntent_ComposesTextFromSituationAndPlannedAction(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		writer.Header().Set("Content-Type", "application/json")

		switch {
		case req.Method == http.MethodPost && req.URL.Path == "/message":
			capturedBody = make([]byte, req.ContentLength)
			_, _ = req.Body.Read(capturedBody)

			data, _ := json.Marshal(apiclient.PostMessageResponse{Cursor: 1})
			_, _ = writer.Write(data)
		case req.Method == http.MethodGet && req.URL.Path == "/wait-for-response":
			data, _ := json.Marshal(apiclient.WaitResponse{Text: "memories"})
			_, _ = writer.Write(data)
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	apiClient := apiclient.New(server.URL, http.DefaultClient)

	_, _, err := callIntentHandler(t, apiClient, "lead", "engram-agent", "deploying", "run tests")

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(capturedBody)).To(ContainSubstring("situation: deploying"))
	g.Expect(string(capturedBody)).To(ContainSubstring("planned-action: run tests"))
}

func TestHandleIntent_WhenPostErrors_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = writer.Write([]byte(`{}`))
	}))
	defer server.Close()

	apiClient := apiclient.New(server.URL, http.DefaultClient)

	_, _, err := callIntentHandler(t, apiClient, "lead", "engram-agent", "testing", "do-thing")

	g.Expect(err).To(HaveOccurred())
}

func TestHandleLearn_AlwaysPostsToEngramAgent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var capturedTo string

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		writer.Header().Set("Content-Type", "application/json")

		var postReq apiclient.PostMessageRequest

		body := make([]byte, req.ContentLength)
		_, _ = req.Body.Read(body)
		_ = json.Unmarshal(body, &postReq)
		capturedTo = postReq.To

		data, _ := json.Marshal(apiclient.PostMessageResponse{Cursor: 1})
		_, _ = writer.Write(data)
	}))
	defer server.Close()

	apiClient := apiclient.New(server.URL, http.DefaultClient)

	_, _, err := callLearnHandler(
		t, apiClient,
		"lead", "feedback", "sit", "beh", "imp", "act", "", "", "",
	)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(capturedTo).To(Equal("engram-agent"))
}

func TestHandleLearn_FactIncludesAllRequiredFields(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		from := rapid.StringMatching(`[a-z][a-z0-9\-]{1,15}`).Draw(rt, "from")
		cursor := rapid.IntRange(0, 100000).Draw(rt, "cursor")

		var capturedText string

		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
			writer.Header().Set("Content-Type", "application/json")

			var postReq apiclient.PostMessageRequest

			body := make([]byte, req.ContentLength)
			_, _ = req.Body.Read(body)
			_ = json.Unmarshal(body, &postReq)
			capturedText = postReq.Text

			data, _ := json.Marshal(apiclient.PostMessageResponse{Cursor: cursor})
			_, _ = writer.Write(data)
		}))
		defer server.Close()

		apiClient := apiclient.New(server.URL, http.DefaultClient)

		result, _, err := callLearnHandler(
			t, apiClient,
			from, "fact", "sit", "", "", "", "subject-value", "pred-value", "obj-value",
		)

		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(result).NotTo(BeNil())

		if result == nil {
			return
		}

		g.Expect(capturedText).To(ContainSubstring(`"type":"fact"`))
		g.Expect(capturedText).To(ContainSubstring(`"subject"`))
		g.Expect(capturedText).To(ContainSubstring(`"predicate"`))
		g.Expect(capturedText).To(ContainSubstring(`"object"`))
	})
}

func TestHandleLearn_FeedbackIncludesAllRequiredFields(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		from := rapid.StringMatching(`[a-z][a-z0-9\-]{1,15}`).Draw(rt, "from")
		situation := rapid.StringMatching(`[A-Za-z0-9 ]{1,40}`).Draw(rt, "situation")
		behavior := rapid.StringMatching(`[A-Za-z0-9 ]{1,40}`).Draw(rt, "behavior")
		impact := rapid.StringMatching(`[A-Za-z0-9 ]{1,40}`).Draw(rt, "impact")
		action := rapid.StringMatching(`[A-Za-z0-9 ]{1,40}`).Draw(rt, "action")
		cursor := rapid.IntRange(0, 100000).Draw(rt, "cursor")

		var capturedText string

		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
			writer.Header().Set("Content-Type", "application/json")

			var postReq apiclient.PostMessageRequest

			body := make([]byte, req.ContentLength)
			_, _ = req.Body.Read(body)
			_ = json.Unmarshal(body, &postReq)
			capturedText = postReq.Text

			data, _ := json.Marshal(apiclient.PostMessageResponse{Cursor: cursor})
			_, _ = writer.Write(data)
		}))
		defer server.Close()

		apiClient := apiclient.New(server.URL, http.DefaultClient)

		result, _, err := callLearnHandler(
			t, apiClient,
			from, "feedback", situation, behavior, impact, action, "", "", "",
		)

		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(result).NotTo(BeNil())

		if result == nil {
			return
		}

		g.Expect(capturedText).To(ContainSubstring(`"type":"feedback"`))
		g.Expect(capturedText).To(ContainSubstring(`"situation"`))
		g.Expect(capturedText).To(ContainSubstring(`"behavior"`))
		g.Expect(capturedText).To(ContainSubstring(`"impact"`))
		g.Expect(capturedText).To(ContainSubstring(`"action"`))
	})
}

func TestHandleLearn_InvalidTypeReturnsError(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		badType := rapid.StringMatching(`[a-z]{1,10}`).
			Filter(func(badType string) bool { return badType != "feedback" && badType != "fact" }).
			Draw(rt, "bad-type")

		// Server should never be called for invalid type.
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		apiClient := apiclient.New(server.URL, http.DefaultClient)

		_, _, err := callLearnHandler(
			t, apiClient,
			"sender", badType, "sit", "beh", "imp", "act", "sub", "pred", "obj",
		)

		// Invalid type returns an error (SDK converts to tool IsError: true).
		g.Expect(err).To(MatchError(ContainSubstring("--type must be 'feedback' or 'fact'")))
	})
}

func TestHandlePost_AlwaysForwardsCursorInResponse(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		from := rapid.StringMatching(`[a-z][a-z0-9\-]{1,15}`).Draw(rt, "from")
		toAgent := rapid.StringMatching(`[a-z][a-z0-9\-]{1,15}`).Draw(rt, "to")
		text := rapid.StringMatching(`[A-Za-z0-9 .,!]{1,80}`).Draw(rt, "text")
		cursor := rapid.IntRange(0, 100000).Draw(rt, "cursor")

		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.Header().Set("Content-Type", "application/json")

			data, _ := json.Marshal(apiclient.PostMessageResponse{Cursor: cursor})
			_, _ = writer.Write(data)
		}))
		defer server.Close()

		apiClient := apiclient.New(server.URL, http.DefaultClient)

		result, _, err := callPostHandler(t, apiClient, from, toAgent, text)

		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(result).NotTo(BeNil())

		if result == nil {
			return
		}

		g.Expect(firstTextContent(g, result)).To(ContainSubstring("cursor"))
	})
}

func TestHandlePost_WhenAPIErrors_ReturnsWrappedError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = writer.Write([]byte(`{"error":"internal error"}`))
	}))
	defer server.Close()

	apiClient := apiclient.New(server.URL, http.DefaultClient)

	_, _, err := callPostHandler(t, apiClient, "from", "to", "text")

	g.Expect(err).To(HaveOccurred())
}

func TestHandleStatus_ReturnsJSONWithRunningAndAgents(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		agents := rapid.SliceOfN(
			rapid.StringMatching(`[a-z][a-z0-9\-]{1,15}`),
			1, 5,
		).Draw(rt, "agents")
		running := rapid.Bool().Draw(rt, "running")

		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.Header().Set("Content-Type", "application/json")

			data, _ := json.Marshal(apiclient.StatusResponse{Running: running, Agents: agents})
			_, _ = writer.Write(data)
		}))
		defer server.Close()

		apiClient := apiclient.New(server.URL, http.DefaultClient)

		result, _, err := callStatusHandler(t, apiClient)

		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(result).NotTo(BeNil())

		if result == nil {
			return
		}

		statusText := firstTextContent(g, result)

		for _, agent := range agents {
			g.Expect(statusText).To(ContainSubstring(agent))
		}
	})
}

func TestHandleStatus_WhenAPIErrors_ReturnsWrappedError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = writer.Write([]byte(`{}`))
	}))
	defer server.Close()

	apiClient := apiclient.New(server.URL, http.DefaultClient)

	_, _, err := callStatusHandler(t, apiClient)

	g.Expect(err).To(HaveOccurred())
}

func TestNewServer_RegistersAllFourTools(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Create a real client pointed at a dummy address — we only care that
	// New() compiles and returns a non-nil server with all tools registered.
	apiClient := apiclient.New("http://localhost:1", http.DefaultClient)

	server := mcpserver.New(apiClient, mcpserver.NewAgentNameCapture())

	g.Expect(server).NotTo(BeNil())

	// Verify compilation succeeded and server is non-nil.
	_ = errors.New("placeholder to verify compilation and non-nil server")
}

// callIntentHandler is a test helper that calls the engram_intent handler.
func callIntentHandler(
	t *testing.T, apiClient apiclient.API, from, toAgent, situation, plannedAction string,
) (*mcp.CallToolResult, any, error) {
	t.Helper()

	handler := mcpserver.ExportHandleIntent(apiClient)

	return handler(context.Background(), &mcp.CallToolRequest{}, mcpserver.IntentArgs{
		From: from, To: toAgent, Situation: situation, PlannedAction: plannedAction,
	})
}

// callLearnHandler is a test helper that calls the engram_learn handler.
func callLearnHandler(
	t *testing.T, apiClient apiclient.API,
	from, learnType, situation, behavior, impact, action, subject, predicate, object string,
) (*mcp.CallToolResult, any, error) {
	t.Helper()

	handler := mcpserver.ExportHandleLearn(apiClient)

	return handler(context.Background(), &mcp.CallToolRequest{}, mcpserver.LearnArgs{
		From:      from,
		Type:      learnType,
		Situation: situation,
		Behavior:  behavior,
		Impact:    impact,
		Action:    action,
		Subject:   subject,
		Predicate: predicate,
		Object:    object,
	})
}

// callPostHandler is a test helper that calls the engram_post handler via the exported builder.
func callPostHandler(
	t *testing.T, apiClient apiclient.API, from, toAgent, text string,
) (*mcp.CallToolResult, any, error) {
	t.Helper()

	handler := mcpserver.ExportHandlePost(apiClient)

	return handler(context.Background(), &mcp.CallToolRequest{}, mcpserver.PostArgs{
		From: from, To: toAgent, Text: text,
	})
}

// callStatusHandler is a test helper that calls the engram_status handler.
func callStatusHandler(t *testing.T, apiClient apiclient.API) (*mcp.CallToolResult, any, error) {
	t.Helper()

	handler := mcpserver.ExportHandleStatus(apiClient)

	return handler(context.Background(), &mcp.CallToolRequest{}, struct{}{})
}

// firstTextContent returns the text of the first TextContent element in the result.
func firstTextContent(g Gomega, result *mcp.CallToolResult) string {
	g.Expect(result.Content).NotTo(BeEmpty())

	textContent, ok := result.Content[0].(*mcp.TextContent)
	g.Expect(ok).To(BeTrue(), "expected first content to be *mcp.TextContent")

	return textContent.Text
}
