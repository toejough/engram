package server

import (
	"context"

	"engram/internal/chat"
)

// AgentLoop is a per-agent goroutine that reads and filters chat messages.
type AgentLoop struct {
	config AgentLoopConfig
	cursor int
}

// NewAgentLoop creates an AgentLoop.
func NewAgentLoop(config AgentLoopConfig) *AgentLoop {
	return &AgentLoop{config: config}
}

// Run blocks, processing messages until ctx is cancelled.
func (a *AgentLoop) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-a.config.Notify:
			a.processNewMessages()
		}
	}
}

func (a *AgentLoop) processNewMessages() {
	messages, newCursor, err := a.config.ReadMessages(a.cursor)
	if err != nil {
		return // logged by caller; don't crash the goroutine
	}

	a.cursor = newCursor

	for _, msg := range messages {
		if a.config.WatchAll || chat.MatchesAgent(msg.To, a.config.Name) {
			a.config.OnMessage(msg)
		}
	}
}

// AgentLoopConfig configures an agent goroutine.
type AgentLoopConfig struct {
	Name         string             // Agent name for recipient filtering.
	WatchAll     bool               // If true, delivers ALL messages (engram-agent).
	Notify       <-chan struct{}    // Notification channel from SharedWatcher.
	ReadMessages ReadMessagesFunc   // Reads messages from chat file at cursor.
	OnMessage    func(chat.Message) // Called for each matching message.
}

// ReadMessagesFunc reads messages from the chat file starting at cursor.
// Returns the messages found and the new cursor position.
type ReadMessagesFunc func(cursor int) ([]chat.Message, int, error)
