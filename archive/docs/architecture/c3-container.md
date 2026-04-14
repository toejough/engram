# C3: Container

What's inside engram. Each box is a deployable unit. See [C4: Context](c4-context.md) for the system boundary. See [C2: Component](c2-component.md) for what's inside each container.

```mermaid
C4Container
    title Container Diagram: Engram

    Person(user, "User")
    System_Ext(claudeCode, "Claude Code")
    System_Ext(claudeBinary, "claude -p")

    System_Boundary(engram, "Engram") {
        Container(apiServer, "API Server", "Go, net/http", "HTTP server on localhost. Watches chat file, manages agent goroutines, invokes engram-agent via claude -p.")
        Container(mcpServer, "MCP Server", "Go, MCP SDK", "Stdio MCP server. Exposes tools to Claude Code. Pushes surfaced memories via notifications/claude/channel.")
        Container(cli, "CLI Client", "Go", "Thin HTTP client. Commands: post, intent, learn, subscribe, status.")
        Container(hooks, "Hooks", "Shell scripts", "UserPromptSubmit, Stop, SubagentStop. Call CLI/API to post user/agent output.")
        ContainerDb(chatFile, "Chat File", "TOML", "Source of truth for inter-agent messages. Append-only with file locking.")
        ContainerDb(memoryFiles, "Memory Files", "TOML", "One file per memory. Feedback (SBIA) and facts (SPO) with evaluation counters.")
        Container(skills, "Skills", "Markdown", "engram-agent, engram-lead, use-engram-chat-as, engram-up, engram-down, recall")
    }

    Rel(claudeCode, mcpServer, "MCP stdio: tools + channel push")
    Rel(claudeCode, hooks, "Hook events: UserPromptSubmit, Stop, SubagentStop")
    Rel(claudeCode, skills, "Loads skill content into agent context")
    Rel(hooks, cli, "Calls engram intent/post")
    Rel(cli, apiServer, "HTTP: POST /message, GET /wait-for-response, etc.")
    Rel(mcpServer, apiServer, "HTTP: same endpoints as CLI")
    Rel(apiServer, chatFile, "Read/write via FilePoster + fsnotify watch")
    Rel(apiServer, claudeBinary, "claude -p --resume for engram-agent")
    Rel(apiServer, memoryFiles, "Engram-agent reads/writes memories")
    Rel(user, claudeCode, "Terminal interaction")
```

## Containers

| Container | Binary/Process | Purpose | Port/Transport |
|-----------|---------------|---------|----------------|
| **API Server** | `engram server up` | All intelligence: routing, validation, agent lifecycle, skill refresh | HTTP on localhost:7932 |
| **MCP Server** | `engram-mcp` | Thin API client exposing MCP tools + async channel push | Stdio (JSON-RPC) |
| **CLI Client** | `engram post/intent/learn/...` | Thin HTTP client for hooks and manual use | HTTP to API server |
| **Hooks** | Shell scripts in `hooks/` | Automatically post user/agent output to API | Called by Claude Code |
| **Chat File** | `~/.local/share/engram/chat/<slug>.toml` | Persistent message log, source of truth | File I/O with locking |
| **Memory Files** | `~/.local/share/engram/memory/{facts,feedback}/` | One TOML file per memory with evaluation counters | File I/O |
| **Skills** | `skills/*/SKILL.md` | Agent behavior instructions loaded into context | Read by Claude Code |

## Key Relationships

- **MCP Server auto-starts API Server** if not running (subprocess launch + health poll)
- **API Server is client-agnostic** — doesn't know if CLI, MCP, or hooks are calling
- **Chat file is the source of truth** — all containers communicate through it
- **Skill refresh** is server-side: API server posts refresh reminders to chat every 13 interactions

## Cross-references

- Each container's internals: [C2: Component](c2-component.md)
- Data flowing between containers: [Sequences](sequences.md)
- Why these containers exist: [Intent](../intent.md)
