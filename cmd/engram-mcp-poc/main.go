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
	"log"

	"engram/internal/mcppoc"
)

func main() {
	err := mcppoc.Run(context.Background())
	if err != nil {
		log.Printf("engram-mcp-poc: %v", err)
	}
}
