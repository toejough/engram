package mcpserver

import (
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"engram/internal/apiclient"
)

// New creates and configures an MCP server with all engram tools registered.
// The provided apiClient is injected into each tool handler.
// The agentCapture is notified on the first tool call with a "from" parameter.
func New(apiClient apiclient.API, agentCapture *AgentNameCapture) *mcp.Server {
	logger := slog.Default()

	server := mcp.NewServer(&mcp.Implementation{
		Name:    serverName,
		Version: serverVersion,
	}, &mcp.ServerOptions{
		Logger:       logger,
		Instructions: serverInstructions,
		Capabilities: &mcp.ServerCapabilities{
			Logging: &mcp.LoggingCapabilities{},
			Experimental: map[string]any{
				"claude/channel": map[string]any{},
			},
		},
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "engram_post",
		Description: "Post a message to the engram chat",
	}, handlePost(apiClient, agentCapture))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "engram_intent",
		Description: "Post an intent and synchronously receive surfaced memories from the engram agent",
	}, handleIntent(apiClient, agentCapture))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "engram_learn",
		Description: "Record a learning (feedback or fact) for the engram agent to store",
	}, handleLearn(apiClient, agentCapture))

	mcp.AddTool(server, &mcp.Tool{
		Name:        "engram_status",
		Description: "Get the current status of the engram API server",
	}, handleStatus(apiClient))

	return server
}

// unexported constants.
const (
	serverInstructions = "Engram memory agent. Surfaced memories arrive as" +
		" <channel source=\"engram\"> events between turns." +
		" Use engram_intent before significant actions." +
		" Use engram_learn after learning something."
	serverName    = "engram"
	serverVersion = "1.0.0"
)
