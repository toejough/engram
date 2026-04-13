package mcpserver

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

// ChannelNotifier pushes notifications/claude/channel events.
// In production, writes raw JSON-RPC to the MCP stdio transport (stdout).
type ChannelNotifier interface {
	Notify(content string, meta map[string]string) error
}

// StdoutChannelNotifier writes raw JSON-RPC channel notifications to a writer.
// Uses a mutex to avoid interleaving with the MCP SDK's stdio transport.
type StdoutChannelNotifier struct {
	mu     sync.Mutex
	writer io.Writer
}

// NewStdoutChannelNotifier creates a notifier writing to the given writer.
// In production, pass os.Stdout.
func NewStdoutChannelNotifier(writer io.Writer) *StdoutChannelNotifier {
	return &StdoutChannelNotifier{writer: writer}
}

// Notify sends a notifications/claude/channel JSON-RPC message.
func (n *StdoutChannelNotifier) Notify(content string, meta map[string]string) error {
	msg := map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/claude/channel",
		"params": map[string]any{
			"content": content,
			"meta":    meta,
		},
	}

	data, marshalErr := json.Marshal(msg)
	if marshalErr != nil {
		return fmt.Errorf("marshaling channel notification: %w", marshalErr)
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	_, writeErr := fmt.Fprintf(n.writer, "%s\n", data)
	if writeErr != nil {
		return fmt.Errorf("writing channel notification: %w", writeErr)
	}

	return nil
}
