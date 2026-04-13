package mcpserver_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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
	cancel()

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

func TestRunSubscribeLoop_CursorAdvancesBetweenCalls(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	callCursors := make([]int, 0, 3)

	mu := &sync.Mutex{}
	callNumber := 0

	srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
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
					Cursor:   5,
					Messages: []apiclient.ChatMessage{{From: "a", To: "b", Text: "msg1"}},
				})
				_, _ = writer.Write(data)
			case 1:
				data, _ := json.Marshal(apiclient.SubscribeResponse{
					Cursor:   10,
					Messages: []apiclient.ChatMessage{{From: "a", To: "b", Text: "msg2"}},
				})
				_, _ = writer.Write(data)
			default:
				time.Sleep(5 * time.Second)

				data, _ := json.Marshal(apiclient.SubscribeResponse{Cursor: 10})
				_, _ = writer.Write(data)
			}

			return
		}

		writer.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	apiClient := apiclient.New(srv.URL, http.DefaultClient)
	recorder := &notifyRecorder{}
	capture := mcpserver.NewAgentNameCapture()
	capture.Set("agent")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	mcpserver.RunSubscribeLoop(ctx, apiClient, recorder, capture, nil)

	mu.Lock()
	captured := make([]int, len(callCursors))
	copy(captured, callCursors)
	mu.Unlock()

	g.Expect(len(captured)).To(BeNumerically(">=", 2))
	g.Expect(captured[0]).To(Equal(0))
	g.Expect(captured[1]).To(Equal(5))
}

func TestRunSubscribeLoop_WhenContextCancelledBeforeAgentName_ExitsCleanly(t *testing.T) {
	t.Parallel()

	apiClient := apiclient.New("http://localhost:1", http.DefaultClient)
	capture := mcpserver.NewAgentNameCapture()
	recorder := &notifyRecorder{}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})

	go func() {
		mcpserver.RunSubscribeLoop(ctx, apiClient, recorder, capture, nil)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Error("RunSubscribeLoop did not exit after context cancellation")
	}
}

func TestRunSubscribeLoop_WhenMessagesArrive_PushesChannelNotifications(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	subscribeCallCount := 0

	srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		writer.Header().Set("Content-Type", "application/json")

		if req.Method == http.MethodGet && req.URL.Path == "/subscribe" {
			subscribeCallCount++
			if subscribeCallCount == 1 {
				data, _ := json.Marshal(apiclient.SubscribeResponse{
					Cursor: 2,
					Messages: []apiclient.ChatMessage{
						{From: "engram-agent", To: "lead", Text: "remember this"},
						{From: "engram-agent", To: "lead", Text: "and this"},
					},
				})
				_, _ = writer.Write(data)

				return
			}

			time.Sleep(5 * time.Second)

			data, _ := json.Marshal(apiclient.SubscribeResponse{Cursor: 2})
			_, _ = writer.Write(data)

			return
		}

		writer.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	apiClient := apiclient.New(srv.URL, http.DefaultClient)
	recorder := &notifyRecorder{}
	capture := mcpserver.NewAgentNameCapture()
	capture.Set("lead")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	mcpserver.RunSubscribeLoop(ctx, apiClient, recorder, capture, nil)

	notifications := recorder.Notifications()
	g.Expect(notifications).To(HaveLen(2))
	g.Expect(notifications[0]).To(ContainSubstring("remember this"))
	g.Expect(notifications[1]).To(ContainSubstring("and this"))
}

func TestRunSubscribeLoop_WhenSubscribeFails_Retries(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var callCount int64

	srv := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		writer.Header().Set("Content-Type", "application/json")

		if req.Method == http.MethodGet && req.URL.Path == "/subscribe" {
			count := atomic.AddInt64(&callCount, 1)

			if count == 1 {
				writer.WriteHeader(http.StatusInternalServerError)
				_, _ = writer.Write([]byte(`{}`))

				return
			}

			data, _ := json.Marshal(apiclient.SubscribeResponse{
				Cursor:   1,
				Messages: []apiclient.ChatMessage{{From: "a", To: "b", Text: "retry worked"}},
			})
			_, _ = writer.Write(data)

			return
		}

		writer.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	apiClient := apiclient.New(srv.URL, http.DefaultClient)
	recorder := &notifyRecorder{}
	capture := mcpserver.NewAgentNameCapture()
	capture.Set("agent")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	done := make(chan struct{})

	go func() {
		mcpserver.RunSubscribeLoop(ctx, apiClient, recorder, capture, nil)
		close(done)
	}()

	// Wait for retry to deliver message.
	g.Eventually(func() int { return len(recorder.Notifications()) }).
		WithTimeout(8 * time.Second).
		Should(BeNumerically(">=", 1))

	cancel()
	<-done

	notifications := recorder.Notifications()
	g.Expect(notifications).NotTo(BeEmpty())
	g.Expect(notifications[0]).To(ContainSubstring("retry worked"))
	g.Expect(atomic.LoadInt64(&callCount)).To(BeNumerically(">=", 2))
}

func TestStdoutChannelNotifier_WritesValidJSONRPC(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var buf bytes.Buffer

	notifier := mcpserver.NewStdoutChannelNotifier(&buf)

	err := notifier.Notify("Memory: use DI", map[string]string{"from": "engram-agent"})
	g.Expect(err).NotTo(HaveOccurred())

	var msg map[string]any
	g.Expect(json.Unmarshal(buf.Bytes(), &msg)).To(Succeed())
	g.Expect(msg["jsonrpc"]).To(Equal("2.0"))
	g.Expect(msg["method"]).To(Equal("notifications/claude/channel"))

	params, ok := msg["params"].(map[string]any)
	g.Expect(ok).To(BeTrue())
	g.Expect(params["content"]).To(Equal("Memory: use DI"))
}

// notifyRecorder implements ChannelNotifier by recording all notifications.
type notifyRecorder struct {
	mu      sync.Mutex
	content []string
}

func (r *notifyRecorder) Notifications() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	snapshot := make([]string, len(r.content))
	copy(snapshot, r.content)

	return snapshot
}

func (r *notifyRecorder) Notify(content string, _ map[string]string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.content = append(r.content, content)

	return nil
}
