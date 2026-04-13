package mcpserver

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"engram/internal/apiclient"
)

// AgentNameCapture captures the agent name from the first tool call.
// Zero value is not usable — create via NewAgentNameCapture.
type AgentNameCapture struct {
	ch   chan string
	once sync.Once
}

// NewAgentNameCapture creates a new capture with a buffered channel.
func NewAgentNameCapture() *AgentNameCapture {
	return &AgentNameCapture{ch: make(chan string, 1)}
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

// RunSubscribeLoop long-polls GET /subscribe and pushes each arriving message
// to the Claude Code agent via notifications/claude/channel.
// Blocks waiting for the agent name, then runs until ctx is cancelled.
func RunSubscribeLoop(
	ctx context.Context,
	apiClient apiclient.API,
	notifier ChannelNotifier,
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

	logger.Info("subscribe loop started", "agent", agentName)

	cursor := 0

	for {
		if ctx.Err() != nil {
			return
		}

		resp, ok := doSubscribe(ctx, apiClient, agentName, cursor, logger)
		if !ok {
			return
		}

		if resp == nil {
			continue
		}

		pushViaChannel(resp.Messages, notifier, logger)

		if resp.Cursor > cursor {
			cursor = resp.Cursor
		}
	}
}

// unexported constants.
const (
	subscribeRetryDelay = 2 * time.Second
)

// unexported functions.

// doSubscribe calls the API Subscribe endpoint. Returns (response, true) on
// success, (nil, true) on retryable error, or (nil, false) on terminal error.
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

	logger.Error("subscribe failed, retrying", "agent", agentName, "err", err)

	select {
	case <-ctx.Done():
		return nil, false
	case <-time.After(subscribeRetryDelay):
		return nil, true
	}
}

// pushViaChannel sends each message as a notifications/claude/channel event.
func pushViaChannel(
	messages []apiclient.ChatMessage,
	notifier ChannelNotifier,
	logger *slog.Logger,
) {
	for _, msg := range messages {
		content := "[" + msg.From + "] " + msg.Text
		meta := map[string]string{
			"from": msg.From,
			"to":   msg.To,
		}

		notifyErr := notifier.Notify(content, meta)
		if notifyErr != nil {
			logger.Error("channel notification failed", "err", notifyErr, "from", msg.From)
		}
	}
}
