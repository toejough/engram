// POC: MCP server that sends notifications/claude/channel.
// Proves channel push to Claude Code works.
//
// The Go SDK doesn't support notifications/claude/channel natively.
// We handle it by writing raw JSON-RPC notifications to stdout alongside
// the SDK's stdio transport. The SDK owns stdout for request/response;
// we write channel notifications between those exchanges.
//
// To test: register this in .mcp.json and start a Claude Code session.
// Channel events should appear as <channel source="engram-poc"> tags.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// stdoutMu serializes writes to stdout.
// The SDK's StdioTransport writes tool responses; we write channel notifications.
// Both must not interleave.
var stdoutMu sync.Mutex

// sendChannelNotification writes a raw notifications/claude/channel JSON-RPC
// message to stdout. This bypasses the SDK because the SDK doesn't support
// custom notification methods.
func sendChannelNotification(content string, meta map[string]string) error {
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

	stdoutMu.Lock()
	defer stdoutMu.Unlock()

	_, writeErr := fmt.Fprintf(os.Stdout, "%s\n", data)

	return writeErr
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "engram-poc",
		Version: "0.0.1",
	}, &mcp.ServerOptions{
		Logger:       logger,
		Instructions: "Engram memory POC. Surfaced memories arrive as <channel source=\"engram-poc\"> events.",
		Capabilities: &mcp.ServerCapabilities{
			Logging: &mcp.LoggingCapabilities{},
			Experimental: map[string]any{
				"claude/channel": map[string]any{},
			},
		},
	})

	// Register a tool so we know the server is working.
	type pingArgs struct {
		Message string `json:"message" jsonschema:"a test message"`
	}

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

	// Background goroutine: send channel notifications every 5 seconds.
	go func() {
		// Wait for the MCP session to initialize.
		time.Sleep(3 * time.Second)

		for i := 1; ; i++ {
			content := fmt.Sprintf("Memory surfaced #%d: always use dependency injection in internal/", i)
			err := sendChannelNotification(content, map[string]string{
				"from":     "engram-agent",
				"severity": "info",
			})
			if err != nil {
				logger.Error("channel notification failed", "err", err)

				return
			}

			logger.Info("sent channel notification", "index", i)

			time.Sleep(5 * time.Second)
		}
	}()

	logger.Info("starting engram-poc MCP server")

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		logger.Error("server failed", "err", err)
		os.Exit(1)
	}
}
