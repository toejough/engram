package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"engram/internal/apiclient"
)

// AgentNameCapture captures the agent name from the first tool call.
// Zero value is not usable — create via NewAgentNameCapture.
type AgentNameCapture struct {
	ch   chan string
	once sync.Once
}

// NewAgentNameCapture creates a new capture with a buffered channel.
// Use this in tests and production wiring (main.go / startup.go).
func NewAgentNameCapture() *AgentNameCapture {
	return newAgentNameCapture()
}

// Set records the agent name. Only the first call takes effect.
func (a *AgentNameCapture) Set(name string) {
	a.once.Do(func() {
		a.ch <- name
	})
}

// Wait blocks until an agent name is set, then returns it.
// Returns ("", false) if ctx is cancelled before a name is set.
func (a *AgentNameCapture) Wait(ctx context.Context) (string, bool) {
	select {
	case <-ctx.Done():
		return "", false
	case name := <-a.ch:
		// Put it back so subsequent calls also see it.
		a.ch <- name

		return name, true
	}
}

// NotificationSender sends a log notification to a single MCP session.
// Implemented in production by a thin wrapper over sess.Log().
type NotificationSender interface {
	SendLog(ctx context.Context, sess *mcp.ServerSession, params *mcp.LoggingMessageParams) error
}

// SessionProvider abstracts server.Sessions() for testing.
type SessionProvider interface {
	Sessions() iter.Seq[*mcp.ServerSession]
}

// RunSubscribeLoop long-polls GET /subscribe for the agent name captured in
// agentCapture and pushes each arriving message to all active MCP sessions
// via sess.Log(). It blocks waiting for the agent name, then runs until ctx
// is cancelled. If logger is nil, slog.Default() is used.
func RunSubscribeLoop(
	ctx context.Context,
	apiClient apiclient.API,
	server *mcp.Server,
	agentCapture *AgentNameCapture,
	logger *slog.Logger,
) {
	if logger == nil {
		logger = slog.Default()
	}

	agentName, ok := agentCapture.Wait(ctx)
	if !ok {
		return
	}

	provider := &serverSessionProvider{server: server}
	runSubscribeLoop(ctx, apiClient, provider, mcpNotificationSender{}, agentName, logger)
}

// unexported constants.
const (
	subscribeRetryDelay  = 2 * time.Second
	subscribeSessionWait = 500 * time.Millisecond
)

// unexported types.

// mcpNotificationSender sends log notifications via the MCP SDK.
type mcpNotificationSender struct{}

// SendLog implements NotificationSender.
func (mcpNotificationSender) SendLog(
	ctx context.Context,
	sess *mcp.ServerSession,
	params *mcp.LoggingMessageParams,
) error {
	err := sess.Log(ctx, params)
	if err != nil {
		return fmt.Errorf("sending log: %w", err)
	}

	return nil
}

// serverSessionProvider wraps *mcp.Server to implement SessionProvider.
type serverSessionProvider struct {
	server *mcp.Server
}

// Sessions implements SessionProvider.
func (sp *serverSessionProvider) Sessions() iter.Seq[*mcp.ServerSession] {
	return sp.server.Sessions()
}

// doSubscribe calls the API Subscribe endpoint. Returns (response, true) on
// success, (nil, true) on retryable error, or (nil, false) on terminal error
// (context cancelled/deadline exceeded).
func doSubscribe(
	ctx context.Context,
	apiClient apiclient.API,
	agentName string,
	cursor int,
	logger *slog.Logger,
) (*apiclient.SubscribeResponse, bool) {
	resp, err := apiClient.Subscribe(ctx, apiclient.SubscribeRequest{
		Agent:       agentName,
		AfterCursor: cursor,
	})
	if err == nil {
		return &resp, true
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return nil, false
	}

	logger.Error("subscribe loop: subscribe failed, retrying", "agent", agentName, "err", err)

	select {
	case <-ctx.Done():
		return nil, false
	case <-time.After(subscribeRetryDelay):
		return nil, true
	}
}

// unexported functions.

// newAgentNameCapture creates a new capture with a buffered channel.
func newAgentNameCapture() *AgentNameCapture {
	return &AgentNameCapture{ch: make(chan string, 1)}
}

// pushMessages sends each message to all active sessions via the sender.
func pushMessages(
	ctx context.Context,
	messages []apiclient.ChatMessage,
	activeSessions []*mcp.ServerSession,
	sender NotificationSender,
	logger *slog.Logger,
) {
	for _, msg := range messages {
		params := &mcp.LoggingMessageParams{
			Level:  "info",
			Data:   fmt.Sprintf("[%s → %s] %s", msg.From, msg.To, msg.Text),
			Logger: "engram",
		}

		for _, sess := range activeSessions {
			logErr := sender.SendLog(ctx, sess, params)
			if logErr != nil {
				logger.Error("subscribe loop: log notification failed",
					"err", logErr,
					"from", msg.From,
					"to", msg.To,
				)
			}
		}
	}
}

// runSubscribeLoop is the testable core of RunSubscribeLoop.
func runSubscribeLoop(
	ctx context.Context,
	apiClient apiclient.API,
	sessions SessionProvider,
	sender NotificationSender,
	agentName string,
	logger *slog.Logger,
) {
	cursor := 0

	for {
		if ctx.Err() != nil {
			return
		}

		if !waitForSession(ctx, sessions, logger) {
			return
		}

		resp, ok := doSubscribe(ctx, apiClient, agentName, cursor, logger)
		if !ok {
			return
		}

		if resp == nil {
			continue
		}

		activeSessions := slices.Collect(sessions.Sessions())
		pushMessages(ctx, resp.Messages, activeSessions, sender, logger)

		if resp.Cursor > cursor {
			cursor = resp.Cursor
		}
	}
}

// waitForSession blocks until at least one session is available or the context
// is cancelled. Returns false if the context was cancelled.
func waitForSession(ctx context.Context, sessions SessionProvider, logger *slog.Logger) bool {
	activeSessions := slices.Collect(sessions.Sessions())
	if len(activeSessions) > 0 {
		return true
	}

	logger.Debug("subscribe loop: no sessions yet, waiting")

	select {
	case <-ctx.Done():
		return false
	case <-time.After(subscribeSessionWait):
		return true
	}
}
