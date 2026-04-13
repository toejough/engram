package mcpserver_test

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	. "github.com/onsi/gomega"

	"engram/internal/apiclient"
	"engram/internal/mcpserver"
)

func TestAgentNameCapture_SetOnlyFiredOnce(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	capture := mcpserver.NewAgentNameCapture()
	capture.Set("first")
	capture.Set("second") // should be ignored

	name, ok := capture.Wait(context.Background())

	g.Expect(ok).To(BeTrue())
	g.Expect(name).To(Equal("first"))
}

func TestAgentNameCapture_WaitReturnsFalseWhenContextCancelled(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	capture := mcpserver.NewAgentNameCapture()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	name, ok := capture.Wait(ctx)

	g.Expect(ok).To(BeFalse())
	g.Expect(name).To(BeEmpty())
}

func TestAgentNameCapture_WaitReturnsSetName(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	capture := mcpserver.NewAgentNameCapture()

	go func() {
		time.Sleep(10 * time.Millisecond)
		capture.Set("lead-42")
	}()

	name, ok := capture.Wait(context.Background())

	g.Expect(ok).To(BeTrue())
	g.Expect(name).To(Equal("lead-42"))
}

func TestMCPNotificationSender_SendLog_WhenSessionClosed_ReturnsError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	ctx := context.Background()

	server := mcp.NewServer(&mcp.Implementation{Name: "test"}, nil)
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	serverSession, connErr := server.Connect(ctx, serverTransport, nil)
	g.Expect(connErr).NotTo(HaveOccurred())

	if connErr != nil {
		return
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "client"}, nil)

	clientSession, clientErr := client.Connect(ctx, clientTransport, nil)
	g.Expect(clientErr).NotTo(HaveOccurred())

	if clientErr != nil {
		return
	}

	// Set log level so the server will attempt to send the notification.
	setErr := clientSession.SetLoggingLevel(ctx, &mcp.SetLoggingLevelParams{Level: "info"})
	g.Expect(setErr).NotTo(HaveOccurred())

	// Close both ends — server-side write should now fail.
	_ = clientSession.Close()
	_ = serverSession.Close()

	sender := mcpserver.ExportMCPNotificationSender()

	err := sender.SendLog(ctx, serverSession, &mcp.LoggingMessageParams{
		Level:  "info",
		Data:   "hello after close",
		Logger: "engram",
	})

	g.Expect(err).To(HaveOccurred())
}

func TestMCPNotificationSender_SendLog_WithRealSession_SendsWithoutError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	ctx := context.Background()

	server := mcp.NewServer(&mcp.Implementation{Name: "test"}, nil)
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	serverSession, connErr := server.Connect(ctx, serverTransport, nil)
	g.Expect(connErr).NotTo(HaveOccurred())

	if connErr != nil {
		return
	}

	defer func() { _ = serverSession.Close() }()

	client := mcp.NewClient(&mcp.Implementation{Name: "client"}, nil)

	clientSession, clientErr := client.Connect(ctx, clientTransport, nil)
	g.Expect(clientErr).NotTo(HaveOccurred())

	if clientErr != nil {
		return
	}

	defer func() { _ = clientSession.Close() }()

	// Set log level so the server will actually attempt to send the notification.
	setErr := clientSession.SetLoggingLevel(ctx, &mcp.SetLoggingLevelParams{Level: "info"})
	g.Expect(setErr).NotTo(HaveOccurred())

	sender := mcpserver.ExportMCPNotificationSender()

	err := sender.SendLog(ctx, serverSession, &mcp.LoggingMessageParams{
		Level:  "info",
		Data:   "hello from test",
		Logger: "engram",
	})

	g.Expect(err).NotTo(HaveOccurred())
}

func TestRunSubscribeLoop_CursorAdvancesAcrossCalls(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	callCursors := make([]int, 0, 3)
	mu := &sync.Mutex{}

	callNumber := 0

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		writer.Header().Set("Content-Type", "application/json")

		if req.Method == http.MethodGet && req.URL.Path == "/subscribe" {
			afterCursor := req.URL.Query().Get("after-cursor")

			var cursor int

			_, _ = fmt.Sscanf(afterCursor, "%d", &cursor)

			mu.Lock()

			callCursors = append(callCursors, cursor)
			callNum := callNumber
			callNumber++
			mu.Unlock()

			switch callNum {
			case 0:
				data, _ := json.Marshal(apiclient.SubscribeResponse{
					Cursor: 5,
					Messages: []apiclient.ChatMessage{
						{From: "a", To: "b", Text: "msg1"},
					},
				})
				_, _ = writer.Write(data)
			case 1:
				data, _ := json.Marshal(apiclient.SubscribeResponse{
					Cursor: 10,
					Messages: []apiclient.ChatMessage{
						{From: "a", To: "b", Text: "msg2"},
					},
				})
				_, _ = writer.Write(data)
			default:
				// Block to end the test.
				time.Sleep(5 * time.Second)

				data, _ := json.Marshal(apiclient.SubscribeResponse{Cursor: 10})
				_, _ = writer.Write(data)
			}

			return
		}

		writer.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	apiClient := apiclient.New(server.URL, http.DefaultClient)
	sessions := &fakeSessionProvider{}
	sessions.SetSessions([]*mcp.ServerSession{nil})

	sender := &fakeNotificationSender{}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	mcpserver.ExportRunSubscribeLoop(ctx, apiClient, sessions, sender, "agent")

	mu.Lock()
	captured := make([]int, len(callCursors))
	copy(captured, callCursors)
	mu.Unlock()

	g.Expect(len(captured)).To(BeNumerically(">=", 2))
	g.Expect(captured[0]).To(Equal(0))
	g.Expect(captured[1]).To(Equal(5))
}

func TestRunSubscribeLoop_WhenContextCancelledBeforeAgentNameSet_ExitsCleanly(t *testing.T) {
	t.Parallel()

	apiClient := apiclient.New("http://localhost:1", http.DefaultClient)
	capture := mcpserver.NewAgentNameCapture()
	apiServer := mcp.NewServer(&mcp.Implementation{Name: "test"}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before Set is called

	done := make(chan struct{})

	go func() {
		mcpserver.RunSubscribeLoop(ctx, apiClient, apiServer, capture, nil)
		close(done)
	}()

	select {
	case <-done:
		// OK — loop should exit immediately because ctx is already cancelled
	case <-time.After(1 * time.Second):
		t.Error("RunSubscribeLoop did not exit after context cancellation")
	}
}

func TestRunSubscribeLoop_WhenContextCancelled_Exits(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		writer.Header().Set("Content-Type", "application/json")

		if req.Method == http.MethodGet && req.URL.Path == "/subscribe" {
			// Block long enough to be cancelled.
			time.Sleep(5 * time.Second)

			data, _ := json.Marshal(apiclient.SubscribeResponse{Cursor: 0})
			_, _ = writer.Write(data)

			return
		}

		writer.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	apiClient := apiclient.New(server.URL, http.DefaultClient)
	sessions := &fakeSessionProvider{}
	sessions.SetSessions([]*mcp.ServerSession{nil})

	sender := &fakeNotificationSender{}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})

	go func() {
		mcpserver.ExportRunSubscribeLoop(ctx, apiClient, sessions, sender, "agent")
		close(done)
	}()

	// Give the loop a moment to reach the subscribe call, then cancel.
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// OK
	case <-time.After(2 * time.Second):
		g.Fail("subscribe loop did not exit after context cancellation")
	}
}

func TestRunSubscribeLoop_WhenMessagesArrive_PushesLogNotifications(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const agentName = "lead-test"

	subscribeCallCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		writer.Header().Set("Content-Type", "application/json")

		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/subscribe":
			subscribeCallCount++
			if subscribeCallCount == 1 {
				// First call returns two messages.
				data, _ := json.Marshal(apiclient.SubscribeResponse{
					Cursor: 2,
					Messages: []apiclient.ChatMessage{
						{From: "engram-agent", To: agentName, Text: "remember this"},
						{From: "engram-agent", To: agentName, Text: "and this"},
					},
				})
				_, _ = writer.Write(data)

				return
			}

			// Subsequent calls block until the test is done.
			time.Sleep(5 * time.Second)

			data, _ := json.Marshal(apiclient.SubscribeResponse{Cursor: 2})
			_, _ = writer.Write(data)
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	apiClient := apiclient.New(server.URL, http.DefaultClient)
	sessions := &fakeSessionProvider{}
	sessions.SetSessions([]*mcp.ServerSession{nil}) // one fake session (nil is ok, sender is faked)

	sender := &fakeNotificationSender{}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	mcpserver.ExportRunSubscribeLoop(ctx, apiClient, sessions, sender, agentName)

	logged := sender.Logged()
	g.Expect(logged).To(HaveLen(2))
	g.Expect(logged[0].Logger).To(Equal("engram"))
	g.Expect(logged[0].Data).To(ContainSubstring("remember this"))
	g.Expect(logged[1].Data).To(ContainSubstring("and this"))
}

func TestRunSubscribeLoop_WhenNoSessions_WaitsBeforeSubscribing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	subscribeCallCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		writer.Header().Set("Content-Type", "application/json")

		if req.Method == http.MethodGet && req.URL.Path == "/subscribe" {
			subscribeCallCount++

			data, _ := json.Marshal(apiclient.SubscribeResponse{Cursor: 0})
			_, _ = writer.Write(data)

			return
		}

		writer.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	apiClient := apiclient.New(server.URL, http.DefaultClient)
	sessions := &fakeSessionProvider{} // no sessions initially
	sender := &fakeNotificationSender{}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	mcpserver.ExportRunSubscribeLoop(ctx, apiClient, sessions, sender, "nobody")

	// With no sessions, the loop should never have called subscribe.
	g.Expect(subscribeCallCount).To(Equal(0))
}

func TestRunSubscribeLoop_WhenSubscribeFails_RetriesAfterDelay(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var callCount int64

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		writer.Header().Set("Content-Type", "application/json")

		if req.Method == http.MethodGet && req.URL.Path == "/subscribe" {
			count := atomic.AddInt64(&callCount, 1)

			if count == 1 {
				writer.WriteHeader(http.StatusInternalServerError)
				_, _ = writer.Write([]byte(`{}`))

				return
			}

			// Second call returns a message.
			data, _ := json.Marshal(apiclient.SubscribeResponse{
				Cursor: 1,
				Messages: []apiclient.ChatMessage{
					{From: "engram-agent", To: "agent", Text: "retry succeeded"},
				},
			})
			_, _ = writer.Write(data)

			return
		}

		writer.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	apiClient := apiclient.New(server.URL, http.DefaultClient)
	sessions := &fakeSessionProvider{}
	sessions.SetSessions([]*mcp.ServerSession{nil})

	sender := &fakeNotificationSender{}

	// Use a short timeout; the retry delay constant is 2s but we override it in tests
	// by verifying call count > 1 within a reasonable window (test runs long-poll retry).
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Run until we get the retry message or timeout.
	done := make(chan struct{})

	go func() {
		mcpserver.ExportRunSubscribeLoop(ctx, apiClient, sessions, sender, "agent")
		close(done)
	}()

	// Poll until sender gets a log notification or context times out.
	for {
		select {
		case <-ctx.Done():
			g.Fail("timed out waiting for retry to deliver message")

			return
		default:
		}

		if len(sender.Logged()) > 0 {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	cancel()
	<-done

	logged := sender.Logged()
	g.Expect(logged).NotTo(BeEmpty())
	g.Expect(logged[0].Data).To(ContainSubstring("retry succeeded"))
	g.Expect(atomic.LoadInt64(&callCount)).To(BeNumerically(">=", 2))
}

func TestRunSubscribeLoop_WithRealServerNoSessions_ExitsOnCancel(t *testing.T) {
	t.Parallel()

	apiClient := apiclient.New("http://localhost:1", http.DefaultClient)
	capture := mcpserver.NewAgentNameCapture()
	apiServer := mcp.NewServer(&mcp.Implementation{Name: "test"}, nil)

	ctx, cancel := context.WithCancel(context.Background())

	capture.Set("agent-real-server")

	done := make(chan struct{})

	go func() {
		// RunSubscribeLoop will call serverSessionProvider.Sessions() on the real server,
		// find no sessions, and wait. Cancel to exit.
		mcpserver.RunSubscribeLoop(ctx, apiClient, apiServer, capture, nil)
		close(done)
	}()

	// Give the loop time to enter waitForSession and call Sessions().
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// OK
	case <-time.After(2 * time.Second):
		t.Error("RunSubscribeLoop did not exit after context cancellation")
	}
}

// fakeNotificationSender records log params sent via SendLog.
type fakeNotificationSender struct {
	mu     sync.Mutex
	logged []*mcp.LoggingMessageParams
}

// Logged returns a snapshot of all logged params.
func (fns *fakeNotificationSender) Logged() []*mcp.LoggingMessageParams {
	fns.mu.Lock()
	defer fns.mu.Unlock()

	snapshot := make([]*mcp.LoggingMessageParams, len(fns.logged))
	copy(snapshot, fns.logged)

	return snapshot
}

// SendLog implements mcpserver.NotificationSender.
func (fns *fakeNotificationSender) SendLog(
	_ context.Context,
	_ *mcp.ServerSession,
	params *mcp.LoggingMessageParams,
) error {
	fns.mu.Lock()
	defer fns.mu.Unlock()

	fns.logged = append(fns.logged, params)

	return nil
}

// fakeSessionProvider returns a fixed slice of sessions as an iterator.
type fakeSessionProvider struct {
	mu       sync.Mutex
	sessions []*mcp.ServerSession
}

// Sessions implements mcpserver.SessionProvider.
func (fsp *fakeSessionProvider) Sessions() iter.Seq[*mcp.ServerSession] {
	fsp.mu.Lock()
	defer fsp.mu.Unlock()

	snapshot := make([]*mcp.ServerSession, len(fsp.sessions))
	copy(snapshot, fsp.sessions)

	return func(yield func(*mcp.ServerSession) bool) {
		for _, sess := range snapshot {
			if !yield(sess) {
				return
			}
		}
	}
}

// SetSessions replaces the session list.
func (fsp *fakeSessionProvider) SetSessions(sessions []*mcp.ServerSession) {
	fsp.mu.Lock()
	defer fsp.mu.Unlock()

	fsp.sessions = sessions
}
