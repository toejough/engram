package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"engram/internal/chat"
	"engram/internal/server"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"
)

// --- POST /message ---

func TestPostMessage_AlwaysReturnsCursorOnSuccess(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		from := rapid.StringMatching(`[a-z]{3,10}`).Draw(rt, "from")
		toAgent := rapid.StringMatching(`[a-z]{3,10}`).Draw(rt, "toAgent")
		text := rapid.StringMatching(`[A-Za-z0-9 ]{5,50}`).Draw(rt, "text")
		cursor := rapid.IntRange(1, 100000).Draw(rt, "cursor")

		deps := &server.Deps{
			PostMessage: func(_ chat.Message) (int, error) {
				return cursor, nil
			},
		}

		body, marshalErr := json.Marshal(map[string]string{
			"from": from, "to": toAgent, "text": text,
		})
		g.Expect(marshalErr).NotTo(HaveOccurred())

		req := httptest.NewRequest(http.MethodPost, "/message", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		server.HandlePostMessage(deps)(rec, req)

		g.Expect(rec.Code).To(Equal(http.StatusOK))

		var resp map[string]int

		decErr := json.NewDecoder(rec.Body).Decode(&resp)
		g.Expect(decErr).NotTo(HaveOccurred())
		g.Expect(resp["cursor"]).To(Equal(cursor))
	})
}

func TestPostMessage_ForwardsFieldsToPostFunc(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var receivedMsg chat.Message

	deps := &server.Deps{
		PostMessage: func(msg chat.Message) (int, error) {
			receivedMsg = msg

			return 1, nil
		},
	}

	body, marshalErr := json.Marshal(map[string]string{
		"from": "alice", "to": "bob", "text": "hello there",
	})
	g.Expect(marshalErr).NotTo(HaveOccurred())

	req := httptest.NewRequest(http.MethodPost, "/message", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	server.HandlePostMessage(deps)(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))
	g.Expect(receivedMsg.From).To(Equal("alice"))
	g.Expect(receivedMsg.To).To(Equal("bob"))
	g.Expect(receivedMsg.Text).To(Equal("hello there"))
}

func TestPostMessage_InvalidJSONReturns400(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := &server.Deps{
		PostMessage: func(_ chat.Message) (int, error) { return 0, nil },
	}

	req := httptest.NewRequest(http.MethodPost, "/message", bytes.NewReader([]byte(`not json`)))
	rec := httptest.NewRecorder()

	server.HandlePostMessage(deps)(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusBadRequest))
}

func TestPostMessage_MalformedLearnMessageReturns400(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := &server.Deps{
		PostMessage: func(_ chat.Message) (int, error) {
			return 0, nil
		},
	}

	// Learn message with type=feedback but missing "action" field.
	learnText := `{"type":"feedback","situation":"s","behavior":"b","impact":"i"}`

	body, marshalErr := json.Marshal(map[string]string{
		"from": "alice", "to": "engram-agent", "text": learnText,
	})
	g.Expect(marshalErr).NotTo(HaveOccurred())

	if marshalErr != nil {
		return
	}

	req := httptest.NewRequest(http.MethodPost, "/message", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	server.HandlePostMessage(deps)(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusBadRequest))

	var resp map[string]string

	decErr := json.NewDecoder(rec.Body).Decode(&resp)
	g.Expect(decErr).NotTo(HaveOccurred())
	g.Expect(resp["error"]).To(ContainSubstring("action"))
}

func TestPostMessage_NilLoggerUsesDefault(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := &server.Deps{
		Logger: nil,
		PostMessage: func(_ chat.Message) (int, error) {
			return 42, nil
		},
	}

	body, marshalErr := json.Marshal(map[string]string{"from": "a", "to": "b", "text": "hello"})
	g.Expect(marshalErr).NotTo(HaveOccurred())

	req := httptest.NewRequest(http.MethodPost, "/message", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	server.HandlePostMessage(deps)(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))
}

func TestPostMessage_NonLearnMessageSkipsValidation(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := &server.Deps{
		PostMessage: func(_ chat.Message) (int, error) {
			return 7, nil
		},
	}

	body, marshalErr := json.Marshal(map[string]string{
		"from": "alice", "to": "bob", "text": "just a plain text message",
	})
	g.Expect(marshalErr).NotTo(HaveOccurred())

	if marshalErr != nil {
		return
	}

	req := httptest.NewRequest(http.MethodPost, "/message", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	server.HandlePostMessage(deps)(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))
}

func TestPostMessage_NonNilLoggerIsUsed(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := &server.Deps{
		Logger: slog.Default(),
		PostMessage: func(_ chat.Message) (int, error) {
			return 7, nil
		},
	}

	body, marshalErr := json.Marshal(map[string]string{"from": "x", "to": "y", "text": "hello"})
	g.Expect(marshalErr).NotTo(HaveOccurred())

	req := httptest.NewRequest(http.MethodPost, "/message", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	server.HandlePostMessage(deps)(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))
}

func TestPostMessage_PostFuncErrorReturns500(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := &server.Deps{
		PostMessage: func(_ chat.Message) (int, error) {
			return 0, errors.New("disk full")
		},
	}

	body, marshalErr := json.Marshal(map[string]string{"from": "alice", "to": "bob", "text": "hi"})
	g.Expect(marshalErr).NotTo(HaveOccurred())

	req := httptest.NewRequest(http.MethodPost, "/message", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	server.HandlePostMessage(deps)(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusInternalServerError))
}

func TestPostMessage_ValidLearnMessageAccepted(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := &server.Deps{
		PostMessage: func(_ chat.Message) (int, error) {
			return 42, nil
		},
	}

	learnText := `{"type":"feedback","situation":"s","behavior":"b","impact":"i","action":"a"}`

	body, marshalErr := json.Marshal(map[string]string{
		"from": "alice", "to": "engram-agent", "text": learnText,
	})
	g.Expect(marshalErr).NotTo(HaveOccurred())

	if marshalErr != nil {
		return
	}

	req := httptest.NewRequest(http.MethodPost, "/message", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	server.HandlePostMessage(deps)(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))
}

// --- POST /reset-agent ---

func TestResetAgent_CallsResetAgentFn(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	called := false

	deps := &server.Deps{
		ResetAgent: func() { called = true },
	}

	req := httptest.NewRequest(http.MethodPost, "/reset-agent", nil)
	rec := httptest.NewRecorder()

	server.HandleResetAgent(deps)(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))
	g.Expect(called).To(BeTrue())
}

func TestResetAgent_ReturnsAcknowledgment(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := &server.Deps{
		ResetAgent: func() {},
	}

	req := httptest.NewRequest(http.MethodPost, "/reset-agent", nil)
	rec := httptest.NewRecorder()

	server.HandleResetAgent(deps)(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))

	var resp map[string]string

	decErr := json.NewDecoder(rec.Body).Decode(&resp)
	g.Expect(decErr).NotTo(HaveOccurred())
	g.Expect(resp["status"]).To(Equal("reset"))
}

// --- POST /shutdown ---

func TestShutdown_CallsShutdownFn(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	called := false

	deps := &server.Deps{
		ShutdownFn: func() { called = true },
	}

	req := httptest.NewRequest(http.MethodPost, "/shutdown", nil)
	rec := httptest.NewRecorder()

	server.HandleShutdown(deps)(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))
	g.Expect(called).To(BeTrue())
}

func TestShutdown_ReturnsAcknowledgment(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := &server.Deps{
		ShutdownFn: func() {},
	}

	req := httptest.NewRequest(http.MethodPost, "/shutdown", nil)
	rec := httptest.NewRecorder()

	server.HandleShutdown(deps)(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))

	var resp map[string]string

	decErr := json.NewDecoder(rec.Body).Decode(&resp)
	g.Expect(decErr).NotTo(HaveOccurred())
	g.Expect(resp).To(HaveKey("status"))
}

// --- GET /status ---

func TestStatus_AlwaysReturnsRunningTrue(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := &server.Deps{
		AgentRegistry: server.NewAgentRegistry(),
	}

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rec := httptest.NewRecorder()

	server.HandleStatus(deps)(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))

	var resp map[string]any

	decErr := json.NewDecoder(rec.Body).Decode(&resp)
	g.Expect(decErr).NotTo(HaveOccurred())
	g.Expect(resp["running"]).To(BeTrue())
}

func TestStatus_EmptyAgentsWhenNoneRegistered(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := &server.Deps{
		AgentRegistry: server.NewAgentRegistry(),
	}

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rec := httptest.NewRecorder()

	server.HandleStatus(deps)(rec, req)

	var resp map[string]any

	decErr := json.NewDecoder(rec.Body).Decode(&resp)
	g.Expect(decErr).NotTo(HaveOccurred())

	agents, ok := resp["agents"].([]any)
	g.Expect(ok).To(BeTrue())
	g.Expect(agents).To(BeEmpty())
}

func TestStatus_ReportsRegisteredAgents(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	registry := server.NewAgentRegistry()
	registry.Register("engram-agent")

	deps := &server.Deps{
		AgentRegistry: registry,
	}

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rec := httptest.NewRecorder()

	server.HandleStatus(deps)(rec, req)

	var resp map[string]any

	decErr := json.NewDecoder(rec.Body).Decode(&resp)
	g.Expect(decErr).NotTo(HaveOccurred())

	agents, ok := resp["agents"].([]any)
	g.Expect(ok).To(BeTrue())
	g.Expect(agents).To(HaveLen(1))
	g.Expect(agents[0]).To(Equal("engram-agent"))
}

func TestStatus_SetsContentTypeJSON(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := &server.Deps{}

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rec := httptest.NewRecorder()

	server.HandleStatus(deps)(rec, req)

	g.Expect(rec.Header().Get("Content-Type")).To(ContainSubstring("application/json"))
}

// --- GET /subscribe ---

func TestSubscribe_FaithfullyReturnsAllMessages(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		count := rapid.IntRange(0, 10).Draw(rt, "count")
		cursor := rapid.IntRange(1, 100000).Draw(rt, "cursor")

		messages := make([]chat.Message, count)
		for i := range messages {
			messages[i] = chat.Message{
				From: "sender",
				To:   "agent",
				Text: rapid.StringMatching(`[A-Za-z0-9]{5,20}`).Draw(rt, "text"),
			}
		}

		deps := &server.Deps{
			SubscribeMessages: func(
				_ context.Context, _ string, _ int,
			) ([]chat.Message, int, error) {
				return messages, cursor, nil
			},
		}

		req := httptest.NewRequest(http.MethodGet,
			"/subscribe?agent=myagent&after-cursor=0", nil)
		rec := httptest.NewRecorder()

		server.HandleSubscribe(deps)(rec, req)

		g.Expect(rec.Code).To(Equal(http.StatusOK))

		var resp subscribeResp

		decErr := json.NewDecoder(rec.Body).Decode(&resp)
		g.Expect(decErr).NotTo(HaveOccurred())
		g.Expect(resp.Messages).To(HaveLen(count))
		g.Expect(resp.Cursor).To(Equal(cursor))

		for i, msg := range messages {
			g.Expect(resp.Messages[i].Text).To(Equal(msg.Text))
		}
	})
}

func TestSubscribe_MissingAfterCursorDefaultsToZero(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var gotCursor int

	deps := &server.Deps{
		SubscribeMessages: func(
			_ context.Context, _ string, afterCursor int,
		) ([]chat.Message, int, error) {
			gotCursor = afterCursor

			return nil, 0, nil
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/subscribe?agent=myagent", nil)
	rec := httptest.NewRecorder()

	server.HandleSubscribe(deps)(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))
	g.Expect(gotCursor).To(Equal(0))
}

func TestSubscribe_ReturnsMessagesForAgent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	messages := []chat.Message{
		{From: "engram-agent", To: "lead-1", Text: "memory surfaced"},
	}

	deps := &server.Deps{
		SubscribeMessages: func(
			_ context.Context, _ string, _ int,
		) ([]chat.Message, int, error) {
			return messages, 8, nil
		},
	}

	req := httptest.NewRequest(http.MethodGet,
		"/subscribe?agent=lead-1&after-cursor=5", nil)
	rec := httptest.NewRecorder()

	server.HandleSubscribe(deps)(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))

	var resp subscribeResp

	decErr := json.NewDecoder(rec.Body).Decode(&resp)
	g.Expect(decErr).NotTo(HaveOccurred())
	g.Expect(resp.Messages).To(HaveLen(1))
	g.Expect(resp.Messages[0].Text).To(Equal("memory surfaced"))
	g.Expect(resp.Cursor).To(Equal(8))
}

func TestSubscribe_SubscribeMessagesErrorReturns500(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := &server.Deps{
		SubscribeMessages: func(_ context.Context, _ string, _ int) ([]chat.Message, int, error) {
			return nil, 0, errors.New("storage unavailable")
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/subscribe?agent=myagent&after-cursor=0", nil)
	rec := httptest.NewRecorder()

	server.HandleSubscribe(deps)(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusInternalServerError))
}

// --- GET /wait-for-response ---

func TestWaitForResponse_MissingAfterCursorReturns400(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := &server.Deps{
		WatchForMessage: func(
			_ context.Context, _, _ string, _ int,
		) (chat.Message, int, error) {
			return chat.Message{}, 0, nil
		},
	}

	req := httptest.NewRequest(http.MethodGet,
		"/wait-for-response?from=agent&to=other", nil)
	rec := httptest.NewRecorder()

	server.HandleWaitForResponse(deps)(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusBadRequest))
}

func TestWaitForResponse_PassesFromToAndCursorToWatch(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var gotFrom, gotTo string

	var gotCursor int

	deps := &server.Deps{
		WatchForMessage: func(
			_ context.Context, from, toAgent string, afterCursor int,
		) (chat.Message, int, error) {
			gotFrom = from
			gotTo = toAgent
			gotCursor = afterCursor

			return chat.Message{Text: "ok"}, afterCursor + 1, nil
		},
	}

	req := httptest.NewRequest(http.MethodGet,
		"/wait-for-response?from=alpha&to=beta&after-cursor=42", nil)
	rec := httptest.NewRecorder()

	server.HandleWaitForResponse(deps)(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))
	g.Expect(gotFrom).To(Equal("alpha"))
	g.Expect(gotTo).To(Equal("beta"))
	g.Expect(gotCursor).To(Equal(42))
}

func TestWaitForResponse_ReturnsMatchingMessage(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	expected := chat.Message{From: "engram-agent", To: "lead-1", Text: "memory found"}

	deps := &server.Deps{
		WatchForMessage: func(
			_ context.Context, _, _ string, _ int,
		) (chat.Message, int, error) {
			return expected, 10, nil
		},
	}

	req := httptest.NewRequest(http.MethodGet,
		"/wait-for-response?from=engram-agent&to=lead-1&after-cursor=5", nil)
	rec := httptest.NewRecorder()

	server.HandleWaitForResponse(deps)(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))

	var resp waitForResp

	decErr := json.NewDecoder(rec.Body).Decode(&resp)
	g.Expect(decErr).NotTo(HaveOccurred())
	g.Expect(resp.Text).To(Equal("memory found"))
	g.Expect(resp.Cursor).To(Equal(10))
}

func TestWaitForResponse_WatchErrorReturns500(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := &server.Deps{
		WatchForMessage: func(_ context.Context, _, _ string, _ int) (chat.Message, int, error) {
			return chat.Message{}, 0, errors.New("watch failed")
		},
	}

	req := httptest.NewRequest(http.MethodGet,
		"/wait-for-response?from=agent&to=other&after-cursor=0", nil)
	rec := httptest.NewRecorder()

	server.HandleWaitForResponse(deps)(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusInternalServerError))
}

// jsonMessage mirrors chat.Message for JSON decoding in tests.
// chat.Message uses toml tags only, so a local type is needed for json.Decode.
type jsonMessage struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Thread string `json:"thread"`
	Type   string `json:"type"`
	Text   string `json:"text"`
}

// subscribeResp mirrors the JSON shape of GET /subscribe for test decoding.
type subscribeResp struct {
	Messages []jsonMessage `json:"messages"`
	Cursor   int           `json:"cursor"`
}

// waitForResp mirrors the JSON shape of GET /wait-for-response for test decoding.
type waitForResp struct {
	Text   string `json:"text"`
	Cursor int    `json:"cursor"`
}
