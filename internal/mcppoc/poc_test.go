package mcppoc_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	. "github.com/onsi/gomega"

	"engram/internal/mcppoc"
)

func TestNewPOCServer_PingToolReturnsPong(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	ctx := context.Background()

	server := mcppoc.ExportNewPOCServer()

	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	_, connErr := server.Connect(ctx, serverTransport, nil)
	g.Expect(connErr).NotTo(HaveOccurred())

	if connErr != nil {
		return
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client"}, nil)

	clientSession, clientErr := client.Connect(ctx, clientTransport, nil)
	g.Expect(clientErr).NotTo(HaveOccurred())

	if clientErr != nil {
		return
	}

	defer func() { _ = clientSession.Close() }()

	result, callErr := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "engram_ping",
		Arguments: map[string]any{
			"message": "hello",
		},
	})
	g.Expect(callErr).NotTo(HaveOccurred())

	if callErr != nil {
		return
	}

	g.Expect(result.Content).NotTo(BeEmpty())

	textContent, ok := result.Content[0].(*mcp.TextContent)
	g.Expect(ok).To(BeTrue())
	g.Expect(textContent.Text).To(ContainSubstring("pong"))
}

func TestNewPOCServer_ReturnsNonNilServer(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	server := mcppoc.ExportNewPOCServer()

	g.Expect(server).NotTo(BeNil())
}

func TestRunNotificationLoopReal_WhenContextCancelledBefore_ExitsCleanly(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before the loop enters init delay

	fake := &fakeNotifier{}

	done := make(chan struct{})

	go func() {
		mcppoc.ExportRunNotificationLoopReal(ctx, fake)
		close(done)
	}()

	select {
	case <-done:
		// OK - should exit quickly since ctx is already cancelled
	case <-time.After(5 * time.Second):
		t.Error("runNotificationLoop did not exit after context cancellation")
	}

	g.Expect(fake.Messages()).To(BeEmpty())
}

func TestRunNotificationLoop_SendsMessageThenCancels(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	ctx, cancel := context.WithCancel(context.Background())

	fake := &fakeNotifier{}

	done := make(chan struct{})

	go func() {
		mcppoc.ExportRunNotificationLoop(ctx, fake)
		close(done)
	}()

	// Wait until at least one message is sent.
	g.Eventually(fake.Messages, 10*time.Second, 10*time.Millisecond).ShouldNot(BeEmpty())

	cancel()

	select {
	case <-done:
		// OK
	case <-time.After(2 * time.Second):
		t.Error("RunNotificationLoop did not exit after context cancellation")
	}

	msgs := fake.Messages()
	g.Expect(msgs).NotTo(BeEmpty())
	g.Expect(msgs[0].content).To(ContainSubstring("Memory surfaced #1"))
	g.Expect(msgs[0].meta["from"]).To(Equal("engram-agent"))
}

func TestRunNotificationLoop_WhenContextCancelledBeforeInit_SendsNothing(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancelled before loop starts

	fake := &fakeNotifier{}

	mcppoc.ExportRunNotificationLoop(ctx, fake)

	g.Expect(fake.Messages()).To(BeEmpty())
}

func TestRunNotificationLoop_WhenSendFails_Exits(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	ctx := context.Background()

	fake := &fakeNotifier{sendErr: errFakeSend}

	done := make(chan struct{})

	go func() {
		mcppoc.ExportRunNotificationLoop(ctx, fake)
		close(done)
	}()

	select {
	case <-done:
		g.Expect(fake.Messages()).NotTo(BeEmpty()) // one send was attempted
	case <-time.After(10 * time.Second):
		t.Error("RunNotificationLoop did not exit after send failure")
	}
}

func TestRunWithDeps_WhenTransportReturnsEOF_ReturnsNilError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var buf bytes.Buffer

	transport := &eofTransport{}

	err := mcppoc.ExportRunWithDeps(t.Context(), &buf, transport)

	// server.Run returns nil when transport returns io.EOF (clean session end).
	g.Expect(err).NotTo(HaveOccurred())
}

func TestRun_WhenContextAlreadyCancelled_ReturnsError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before Run is called

	// With a cancelled context, StdioTransport exits and Run returns context.Canceled.
	err := mcppoc.Run(ctx)

	g.Expect(err).To(HaveOccurred())
}

func TestWriterNotifier_Send_JSONIsValidAndContainsFields(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var buf bytes.Buffer

	notifier := mcppoc.ExportNewWriterNotifier(&buf)

	err := notifier.Send("test content", map[string]string{"key": "value"})

	g.Expect(err).NotTo(HaveOccurred())

	var parsed map[string]any

	parseErr := json.Unmarshal(buf.Bytes(), &parsed)
	g.Expect(parseErr).NotTo(HaveOccurred())

	g.Expect(parsed["jsonrpc"]).To(Equal("2.0"))
	g.Expect(parsed["method"]).To(Equal("notifications/claude/channel"))

	params, ok := parsed["params"].(map[string]any)
	g.Expect(ok).To(BeTrue())
	g.Expect(params["content"]).To(Equal("test content"))
}

func TestWriterNotifier_Send_WritesJSONRPCNotification(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var buf bytes.Buffer

	notifier := mcppoc.ExportNewWriterNotifier(&buf)

	err := notifier.Send("hello memory", map[string]string{"from": "engram-agent"})

	g.Expect(err).NotTo(HaveOccurred())

	line := buf.String()
	g.Expect(line).To(ContainSubstring(`"jsonrpc":"2.0"`))
	g.Expect(line).To(ContainSubstring(`"notifications/claude/channel"`))
	g.Expect(line).To(ContainSubstring(`"hello memory"`))
}

// unexported variables.
var (
	errFakeSend = fakeError("send failed")
)

// eofConnection is a fake mcp.Connection that returns io.EOF on Read.
type eofConnection struct{}

// Close implements mcp.Connection.
func (*eofConnection) Close() error { return nil }

// Read implements mcp.Connection.
func (*eofConnection) Read(_ context.Context) (jsonrpc.Message, error) { return nil, io.EOF }

// SessionID implements mcp.Connection.
func (*eofConnection) SessionID() string { return "poc-test" }

// Write implements mcp.Connection.
func (*eofConnection) Write(_ context.Context, _ jsonrpc.Message) error { return nil }

// eofTransport is a fake mcp.Transport that returns io.EOF on first Read.
type eofTransport struct{}

// Connect implements mcp.Transport.
func (*eofTransport) Connect(_ context.Context) (mcp.Connection, error) {
	return &eofConnection{}, nil
}

// fakeError is a simple error type for testing.
type fakeError string

func (e fakeError) Error() string { return string(e) }

// fakeNotifier records Send calls for testing.
type fakeNotifier struct {
	mu      sync.Mutex
	sent    []sentMessage
	sendErr error
}

func (fn *fakeNotifier) Messages() []sentMessage {
	fn.mu.Lock()
	defer fn.mu.Unlock()

	snapshot := make([]sentMessage, len(fn.sent))
	copy(snapshot, fn.sent)

	return snapshot
}

func (fn *fakeNotifier) Send(content string, meta map[string]string) error {
	fn.mu.Lock()
	defer fn.mu.Unlock()

	fn.sent = append(fn.sent, sentMessage{content: content, meta: meta})

	return fn.sendErr
}

type sentMessage struct {
	content string
	meta    map[string]string
}
