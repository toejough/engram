// Package mcppoc implements the engram MCP proof-of-concept server.
// It proves that notifications/claude/channel works by sending periodic
// test notifications to a Claude Code MCP session.
package mcppoc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Exported constants.
const (
	// NotificationInitDelay is how long to wait before sending the first notification.
	NotificationInitDelay = 3 * time.Second
	// NotificationInterval is how long to wait between notifications.
	NotificationInterval = 5 * time.Second
)

// Run starts the POC MCP server and blocks until it exits.
func Run(ctx context.Context) error {
	return runWithDeps(ctx, os.Stdout, &mcp.StdioTransport{})
}

// unexported constants.
const (
	pocServerInstructions = "Engram memory POC. Surfaced memories arrive" +
		" as <channel source=\"engram-poc\"> events."
	pocServerName    = "engram-poc"
	pocServerVersion = "0.0.1"
)

// notifier sends channel notifications.
type notifier interface {
	Send(content string, meta map[string]string) error
}

// unexported types.

// pingArgs are the parameters for the engram_ping tool.
type pingArgs struct {
	Message string `json:"message" jsonschema:"a test message"`
}

// writerNotifier sends channel notifications to an injected writer.
type writerNotifier struct {
	mu     sync.Mutex
	writer io.Writer
}

// Send writes a raw notifications/claude/channel JSON-RPC message to the writer.
// This bypasses the SDK because the SDK doesn't support custom notification methods.
func (n *writerNotifier) Send(content string, meta map[string]string) error {
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

// unexported functions.

// newPOCServer creates and configures the POC MCP server.
func newPOCServer(logger *slog.Logger) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    pocServerName,
		Version: pocServerVersion,
	}, &mcp.ServerOptions{
		Logger:       logger,
		Instructions: pocServerInstructions,
		Capabilities: &mcp.ServerCapabilities{
			Logging: &mcp.LoggingCapabilities{},
			Experimental: map[string]any{
				"claude/channel": map[string]any{},
			},
		},
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "engram_ping",
		Description: "Echo a message back (POC tool)",
	}, func(_ context.Context, _ *mcp.CallToolRequest, args pingArgs) (*mcp.CallToolResult, any, error) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "pong: " + args.Message},
			},
		}, nil, nil
	})

	return server
}

// runNotificationLoop sends periodic channel notifications until ctx is cancelled.
func runNotificationLoop(ctx context.Context, logger *slog.Logger, n notifier) {
	runNotificationLoopWithDelays(ctx, logger, n, NotificationInitDelay, NotificationInterval)
}

// runNotificationLoopWithDelays is the testable core of runNotificationLoop.
func runNotificationLoopWithDelays(
	ctx context.Context,
	logger *slog.Logger,
	n notifier,
	initDelay, interval time.Duration,
) {
	if initDelay > 0 {
		select {
		case <-ctx.Done():
			return
		case <-time.After(initDelay):
		}
	}

	for count := 1; ; count++ {
		if ctx.Err() != nil {
			return
		}

		content := fmt.Sprintf(
			"Memory surfaced #%d: always use dependency injection in internal/", count,
		)

		sendErr := n.Send(content, map[string]string{
			"from":     "engram-agent",
			"severity": "info",
		})
		if sendErr != nil {
			logger.Error("channel notification failed", "err", sendErr)

			return
		}

		logger.Info("sent channel notification", "index", count)

		if interval > 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(interval):
			}
		}
	}
}

// unexported functions.

// runWithDeps starts the POC server with injectable writer and transport.
func runWithDeps(ctx context.Context, writer io.Writer, transport mcp.Transport) error {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	notifier := &writerNotifier{writer: writer}
	server := newPOCServer(logger)

	go runNotificationLoop(ctx, logger, notifier)

	logger.Info("starting engram-poc MCP server")

	runErr := server.Run(ctx, transport)
	if runErr != nil {
		return fmt.Errorf("poc server: %w", runErr)
	}

	return nil
}
