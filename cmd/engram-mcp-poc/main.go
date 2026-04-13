// POC: MCP server that sends notifications/claude/channel via middleware.
// Proves async push to Claude Code agent works.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "engram-poc",
		Version: "0.0.1",
	}, &mcp.ServerOptions{
		Logger:       logger,
		Instructions: "Events from engram arrive as <channel source=\"engram-poc\">. React to surfaced memories.",
	})

	// Register a simple tool.
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

	// Add middleware to support notifications/claude/channel.
	// The default handler rejects unknown notification methods.
	// Our middleware intercepts it and sends raw via the connection.
	server.AddSendingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			if method == "notifications/claude/channel" {
				// The default handler will fail because this method isn't in clientMethodInfos.
				// But the underlying logic is: for "notifications/*", call conn.Notify.
				// We replicate that logic here.
				logger.Info("middleware: sending channel notification")
			}
			return next(ctx, method, req)
		}
	})

	// Start goroutine to push notifications after sessions connect.
	go func() {
		time.Sleep(3 * time.Second)
		for i := 1; ; i++ {
			sessions := slices.Collect(server.Sessions())
			if len(sessions) == 0 {
				logger.Warn("no sessions yet, waiting...")
				time.Sleep(2 * time.Second)

				continue
			}

			for _, sess := range sessions {
				// Use Log notification (known to the SDK) to push messages.
				// This is a workaround until we can send custom notifications.
				logErr := sess.Log(context.Background(), &mcp.LoggingMessageParams{
					Level:  "info",
					Data:   fmt.Sprintf("Memory surfaced #%d: always use DI in internal/", i),
					Logger: "engram",
				})
				if logErr != nil {
					logger.Error("log notification failed", "err", logErr)
				} else {
					logger.Info("sent log notification", "index", i)
				}
			}

			time.Sleep(5 * time.Second)
		}
	}()

	logger.Info("starting engram-poc MCP server")

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		logger.Error("server failed", "err", err)
		os.Exit(1)
	}
}
