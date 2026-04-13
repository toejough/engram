# Stage 2: MCP Server — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build an MCP server that wraps the engram API server, exposing tools (`engram_post`, `engram_intent`, `engram_learn`, `engram_status`) and pushing surfaced memories to the agent as channel events.

**Architecture:** A Go binary (`cmd/engram-mcp/main.go`) implements the MCP protocol over stdio using the official Go SDK. It auto-starts the API server if not running. It exposes 4 MCP tools that call the API via `internal/apiclient`. A background goroutine long-polls `GET /subscribe` and pushes results as `notifications/claude/channel` events. This is a two-way channel: it pushes events AND exposes tools. Registered via `.mcp.json` at the plugin root.

**Tech Stack:** `github.com/modelcontextprotocol/go-sdk` (MCP Go SDK), `internal/apiclient` (HTTP client), stdio transport, `encoding/json`.

**Principles:** Read `docs/exec-planning.md`. DI for all I/O. Context flows from top. Property-based tests where applicable. Full TDD cycle.

---

## File Structure

```
cmd/engram-mcp/
  main.go              — MCP server entry point (stdio transport, thin wiring)

internal/mcpserver/
  server.go            — MCP server setup: tool registration, channel capability, subscribe loop
  server_test.go       — MCP server tests
  tools.go             — Tool handlers: engram_post, engram_intent, engram_learn, engram_status
  tools_test.go        — Tool handler tests

.mcp.json              — Plugin MCP server registration
```

---

### Task 1: Add MCP Go SDK dependency + basic server skeleton

**Files:**
- Create: `cmd/engram-mcp/main.go`
- Create: `internal/mcpserver/server.go`
- Modify: `go.mod` (add MCP SDK dependency)

- [ ] **Step 1: Add the MCP Go SDK dependency**

```bash
go get github.com/modelcontextprotocol/go-sdk@latest
```

- [ ] **Step 2: Create the MCP server skeleton**

Create `internal/mcpserver/server.go`:
```go
// Package mcpserver implements the engram MCP server (two-way channel).
package mcpserver

import (
	"context"
	"log/slog"

	"engram/internal/apiclient"
)

// Config configures the MCP server.
type Config struct {
	APIClient *apiclient.Client  // HTTP client for the engram API server.
	AgentName string             // Name for subscribe (identifies this agent).
	Logger    *slog.Logger
}

// NOTE: Actual MCP SDK types and server setup will be implemented after
// we verify the SDK import works and understand the exact API.
```

Create `cmd/engram-mcp/main.go`:
```go
package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"engram/internal/apiclient"
	_ "engram/internal/mcpserver"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	apiAddr := os.Getenv("ENGRAM_API_ADDR")
	if apiAddr == "" {
		apiAddr = "http://localhost:7932"
	}

	client := apiclient.New(apiAddr, http.DefaultClient)
	_ = client
	_ = logger

	fmt.Fprintln(os.Stderr, "engram-mcp: starting")
	// TODO: Will be filled in after SDK exploration in Step 3.
}
```

- [ ] **Step 3: Explore the MCP Go SDK API**

The SDK may differ from the TypeScript examples. Read the Go SDK source to understand:
- How to create a server with stdio transport
- How to register tools
- How to declare channel capability (`experimental: { 'claude/channel': {} }`)
- How to send `notifications/claude/channel`

Run: `go doc github.com/modelcontextprotocol/go-sdk/...`
Or browse: `ls $(go env GOMODCACHE)/github.com/modelcontextprotocol/go-sdk@*/`

Adapt the implementation to the actual SDK API.

- [ ] **Step 4: Implement MCP server with tools and channel capability**

Based on the SDK exploration, implement the full server in `internal/mcpserver/server.go`:
- Create MCP server with name "engram", version "1.0.0"
- Declare capabilities: `experimental: { 'claude/channel': {} }` + `tools: {}`
- Set instructions: "Events from engram arrive as <channel source=\"engram\">. These contain surfaced memories. React to them as additional context."
- Register 4 tools (see Task 2)
- Start stdio transport

- [ ] **Step 5: Verify it compiles**

```bash
go build ./cmd/engram-mcp/
```

- [ ] **Step 6: Commit**

```bash
git add cmd/engram-mcp/ internal/mcpserver/ go.mod go.sum
git commit -m "feat(mcp): add MCP server skeleton with Go SDK

AI-Used: [claude]"
```

---

### Task 2: MCP tool handlers

Implement the 4 MCP tool handlers. Each wraps the corresponding `apiclient` method.

**Files:**
- Create: `internal/mcpserver/tools.go`
- Create: `internal/mcpserver/tools_test.go`

- [ ] **Step 1: Write failing tests for each tool handler**

Each tool handler takes MCP tool input (parsed from JSON Schema) and returns a tool result. Test with fake `apiclient.API` (imptest or httptest).

Tools to implement:
- `engram_post(from, to, text)` → calls `client.PostMessage`, returns cursor
- `engram_intent(from, to, situation, planned_action)` → calls `client.PostMessage` + `client.WaitForResponse`, returns surfaced memories
- `engram_learn(from, type, situation, ...)` → calls `client.PostMessage`, returns success/error
- `engram_status()` → calls `client.Status`, returns health

- [ ] **Step 2: Implement tool handlers**

Each handler:
1. Extracts parameters from the MCP tool input
2. Calls the appropriate `apiclient` method
3. Returns the result as MCP tool content

- [ ] **Step 3: Register tools with the MCP server**

Register each tool with its JSON Schema input definition:

```go
// engram_post
{
    "name": "engram_post",
    "description": "Post a message to the engram chat",
    "inputSchema": {
        "type": "object",
        "properties": {
            "from": {"type": "string", "description": "sender agent name"},
            "to": {"type": "string", "description": "recipient agent name"},
            "text": {"type": "string", "description": "message content"}
        },
        "required": ["from", "to", "text"]
    }
}
```

Similar schemas for intent (from, to, situation, planned_action), learn (from, type, situation, + content fields), status (no params).

- [ ] **Step 4: Run tests, verify pass**

- [ ] **Step 5: Commit**

```bash
git add internal/mcpserver/tools.go internal/mcpserver/tools_test.go
git commit -m "feat(mcp): add engram_post, engram_intent, engram_learn, engram_status tools

AI-Used: [claude]"
```

---

### Task 3: Async channel push — subscribe loop

A background goroutine that long-polls `GET /subscribe` from the API server and pushes surfaced memories to the agent via `notifications/claude/channel`.

**Files:**
- Modify: `internal/mcpserver/server.go`
- Modify: `internal/mcpserver/server_test.go`

- [ ] **Step 1: Write failing test — subscribe loop pushes channel events**

Test that when the API server returns messages via subscribe, the MCP server sends `notifications/claude/channel` with the message content.

- [ ] **Step 2: Implement subscribe loop**

```go
// subscribeLoop long-polls GET /subscribe and pushes results as channel events.
func (s *MCPServer) subscribeLoop(ctx context.Context) {
    cursor := 0
    for {
        resp, err := s.apiClient.Subscribe(ctx, apiclient.SubscribeRequest{
            Agent:       s.agentName,
            AfterCursor: cursor,
        })
        if err != nil {
            s.logger.Error("subscribe failed", "err", err)
            // Context cancelled = shutdown. Other errors = retry.
            if ctx.Err() != nil {
                return
            }
            continue
        }

        for _, msg := range resp.Messages {
            s.pushChannelEvent(msg)
        }

        cursor = resp.Cursor
    }
}

func (s *MCPServer) pushChannelEvent(msg apiclient.ChatMessage) {
    // Send notifications/claude/channel with msg content.
    s.mcpServer.Notification(ctx, "notifications/claude/channel", map[string]any{
        "content": fmt.Sprintf("[%s] %s", msg.From, msg.Text),
        "meta":    map[string]string{"from": msg.From},
    })
}
```

- [ ] **Step 3: Start subscribe loop when MCP server starts**

Launch as a goroutine in the server's Run method.

- [ ] **Step 4: Run tests, verify pass**

- [ ] **Step 5: Commit**

```bash
git add internal/mcpserver/server.go internal/mcpserver/server_test.go
git commit -m "feat(mcp): add subscribe loop for async channel push

AI-Used: [claude]"
```

---

### Task 4: Auto-start API server + .mcp.json registration

The MCP server checks if the API server is running. If not, starts it as a subprocess. Register the MCP server in `.mcp.json`.

**Files:**
- Modify: `cmd/engram-mcp/main.go`
- Create: `.mcp.json`

- [ ] **Step 1: Implement API server auto-start**

In `main.go`, before creating the MCP server:
```go
// Check if API server is running.
client := apiclient.New(apiAddr, http.DefaultClient)
_, statusErr := client.Status(context.Background())
if statusErr != nil {
    // Start API server as subprocess.
    logger.Info("starting API server", "addr", apiAddr)
    cmd := exec.Command("engram", "server", "up", "--addr", apiAddr)
    cmd.Stderr = os.Stderr
    if startErr := cmd.Start(); startErr != nil {
        logger.Error("failed to start API server", "err", startErr)
        os.Exit(1)
    }
    // Wait for it to be ready.
    for range 30 {
        time.Sleep(500 * time.Millisecond)
        if _, err := client.Status(context.Background()); err == nil {
            break
        }
    }
}
```

- [ ] **Step 2: Create .mcp.json**

```json
{
  "mcpServers": {
    "engram": {
      "command": "${CLAUDE_PLUGIN_ROOT}/bin/engram-mcp-server",
      "env": {
        "ENGRAM_API_ADDR": "http://localhost:7932"
      }
    }
  }
}
```

- [ ] **Step 3: Add build step for engram-mcp binary**

The session-start.sh hook already builds the engram binary. Add engram-mcp-server build:
```bash
go build -o "${PLUGIN_BIN}/engram-mcp-server" ./cmd/engram-mcp/
```

- [ ] **Step 4: Commit**

```bash
git add cmd/engram-mcp/main.go .mcp.json hooks/session-start.sh
git commit -m "feat(mcp): auto-start API server, register in .mcp.json

AI-Used: [claude]"
```

---

### Task 5: Skill updates for Stage 2

Update skills to reference MCP tools instead of CLI commands.

**Files:**
- Modify: `skills/engram-lead/SKILL.md`
- Modify: `skills/engram-up/SKILL.md`
- Modify: `skills/engram-down/SKILL.md`

- [ ] **Step 1: Update engram-lead**

Replace CLI commands with MCP tools:
- `engram intent` → `engram_intent` MCP tool
- `engram learn` → `engram_learn` MCP tool
- `engram post` → `engram_post` MCP tool
- Add: memories arrive as `<channel source="engram">` events between turns
- Add: skill refresh reminders arrive as channel events

- [ ] **Step 2: Update engram-up**

- Startup: MCP server auto-starts API server. No need to run `engram server up` manually.
- Load skills: `/use-engram-chat-as` and `/engram-lead`
- Tmux panes unchanged

- [ ] **Step 3: Update engram-down**

- Shutdown: `engram_post` MCP tool instead of CLI

- [ ] **Step 4: Commit**

```bash
git add skills/engram-lead/SKILL.md skills/engram-up/SKILL.md skills/engram-down/SKILL.md
git commit -m "feat(skills): update skills for MCP tools and channel events

AI-Used: [claude]"
```

---

### Task 6: Quality check + e2e testing

- [ ] **Step 1: Run full test suite**

Run: `targ test`

- [ ] **Step 2: Run quality check**

Run: `targ check-full`

- [ ] **Step 3: E2E test**

Build both binaries. Start the MCP server (which auto-starts the API server). Verify tools work via direct JSON-RPC over stdio.

```bash
go build -o /tmp/engram-mcp ./cmd/engram-mcp/

# Test tools/list
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | /tmp/engram-mcp 2>/dev/null

# Test engram_status tool
echo '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"engram_status","arguments":{}}}' | /tmp/engram-mcp 2>/dev/null
```

- [ ] **Step 4: Fix issues, commit**
