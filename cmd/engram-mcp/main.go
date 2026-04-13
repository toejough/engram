// Package main provides the engram MCP server entry point.
// It auto-starts the engram API server if not running, then serves MCP tools
// over stdio. The API address is resolved from ENGRAM_API_ADDR (default: http://localhost:7932).
package main

import (
	"context"
	"log"

	"engram/internal/mcpserver"
)

func main() {
	err := mcpserver.Run(context.Background())
	if err != nil {
		log.Printf("engram-mcp: %v", err)
	}
}
