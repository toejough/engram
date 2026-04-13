# C2: Component

What's inside each container. See [C3: Container](c3-container.md) for the container boundaries. See [C1: Code](c1-code.md) for the types and interfaces.

## API Server Components

```mermaid
C4Component
    title Components: API Server (internal/server)

    Container_Boundary(apiServer, "API Server") {
        Component(handlers, "HTTP Handlers", "handlers.go", "POST /message, GET /wait-for-response, GET /subscribe, GET /status, POST /shutdown, POST /reset-agent")
        Component(server, "Server", "server.go", "HTTP listener, routing, graceful shutdown, slog config")
        Component(fanout, "SharedWatcher", "fanout.go", "Single fsnotify watcher, fans out to agent goroutines via buffered channels")
        Component(agentLoop, "AgentLoop", "agent.go", "Per-agent goroutine: cursor tracking, message filtering, WatchAll for engram-agent")
        Component(engramAgent, "EngramAgent", "engram.go", "Session lifecycle: claude -p invocation, skill refresh, error recovery ladder")
        Component(streamParser, "StreamParser", "stream.go", "Parses stream-json JSONL from claude -p, extracts structured AgentResponse")
        Component(validate, "Validator", "validate.go", "Learn message validation: feedback fields, fact fields")
        Component(refresh, "RefreshTracker", "refresh.go", "Counts interactions per agent, triggers skill refresh every 13")
    }

    ComponentDb(chatFile, "Chat File", "TOML")
    Component_Ext(claudeBinary, "claude -p")

    Rel(server, handlers, "Routes HTTP requests to handlers")
    Rel(handlers, chatFile, "PostMessage writes via FilePoster")
    Rel(fanout, chatFile, "fsnotify watches for changes")
    Rel(fanout, agentLoop, "Notifies via buffered channel")
    Rel(agentLoop, engramAgent, "OnMessage callback invokes Process")
    Rel(engramAgent, claudeBinary, "RunClaudeFunc: claude -p --resume")
    Rel(engramAgent, streamParser, "Parses stdout")
    Rel(engramAgent, chatFile, "Routes responses back to chat")
    Rel(handlers, validate, "Validates learn messages before posting")
    Rel(engramAgent, refresh, "Checks if skill reload needed")
```

## MCP Server Components

```mermaid
C4Component
    title Components: MCP Server (internal/mcpserver)

    Container_Boundary(mcpServer, "MCP Server") {
        Component(tools, "Tool Handlers", "tools.go", "engram_post, engram_intent, engram_learn, engram_status")
        Component(mcpSetup, "Server Setup", "server.go", "MCP server creation, tool registration, channel capability declaration")
        Component(subscribe, "Subscribe Loop", "subscribe.go", "Long-polls GET /subscribe, pushes via ChannelNotifier")
        Component(channel, "ChannelNotifier", "channel.go", "Writes raw notifications/claude/channel JSON-RPC to stdout")
        Component(startup, "Auto-Start", "startup.go", "Checks API server health, launches subprocess if needed")
        Component(capture, "AgentNameCapture", "subscribe.go", "Captures agent name from first tool call for subscribe")
    }

    Component_Ext(apiServer, "API Server", "HTTP")
    Component_Ext(claudeCode, "Claude Code", "Stdio")

    Rel(claudeCode, mcpSetup, "JSON-RPC over stdio")
    Rel(tools, apiServer, "HTTP via apiclient.API")
    Rel(subscribe, apiServer, "GET /subscribe long-poll")
    Rel(subscribe, channel, "Pushes surfaced memories")
    Rel(channel, claudeCode, "notifications/claude/channel")
    Rel(startup, apiServer, "GET /status health check")
```

## CLI Client Components

```mermaid
C4Component
    title Components: CLI Client (internal/cli + internal/apiclient)

    Container_Boundary(cliClient, "CLI Client") {
        Component(apiClient, "API Client", "apiclient/client.go", "PostMessage, WaitForResponse, Subscribe, Status")
        Component(doPost, "doPost", "cli/cli_api.go", "Pure handler: posts message via API interface")
        Component(doIntent, "doIntent", "cli/cli_api.go", "Pure handler: post + wait-for-response two-step")
        Component(doLearn, "doLearn", "cli/cli_api.go", "Pure handler: builds learn JSON, posts via API")
        Component(doSubscribe, "doSubscribe", "cli/cli_api.go", "Pure handler: long-poll loop printing messages")
        Component(doStatus, "doStatus", "cli/cli_api.go", "Pure handler: prints server health JSON")
        Component(wiring, "Thin Wiring", "cli/cli_api.go + cli.go", "Flag parsing, http.DefaultClient construction, context creation")
    }

    Component_Ext(apiServer, "API Server", "HTTP")

    Rel(wiring, doPost, "Parses flags, passes API interface")
    Rel(wiring, doIntent, "Parses flags, passes API interface")
    Rel(wiring, doLearn, "Parses flags, passes API interface")
    Rel(doPost, apiClient, "PostMessage")
    Rel(doIntent, apiClient, "PostMessage + WaitForResponse")
    Rel(doLearn, apiClient, "PostMessage")
    Rel(apiClient, apiServer, "HTTP requests")
```

## Component Summary

| Container | Component | File | Responsibility |
|-----------|-----------|------|---------------|
| API Server | Handlers | `handlers.go` | HTTP request/response |
| API Server | Server | `server.go` | Listener, routing, shutdown |
| API Server | SharedWatcher | `fanout.go` | fsnotify fan-out to goroutines |
| API Server | AgentLoop | `agent.go` | Per-agent cursor + filtering |
| API Server | EngramAgent | `engram.go` | claude -p lifecycle + recovery |
| API Server | StreamParser | `stream.go` | Parse structured JSON from stream-json |
| API Server | Validator | `validate.go` | Learn message field validation |
| API Server | RefreshTracker | `refresh.go` | Skill refresh counter |
| MCP Server | Tools | `tools.go` | 4 MCP tool handlers |
| MCP Server | Subscribe Loop | `subscribe.go` | Long-poll + channel push |
| MCP Server | ChannelNotifier | `channel.go` | Raw JSON-RPC to stdout |
| MCP Server | Auto-Start | `startup.go` | API server subprocess launch |
| CLI | API Client | `apiclient/client.go` | HTTP client with DI |
| CLI | Pure Handlers | `cli/cli_api.go` | doPost/doIntent/doLearn/doSubscribe/doStatus |
| CLI | Thin Wiring | `cli/cli.go` | Flag parsing, I/O construction |

## Cross-references

- Types and interfaces used by these components: [C1: Code](c1-code.md)
- Data flowing between components: [Sequences](sequences.md)
- Container boundaries: [C3: Container](c3-container.md)
