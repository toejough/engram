# C1: Code (Entity-Relationship Diagrams)

Key types and their relationships, mapped to source files. See [C2: Component](c2-component.md) for which components own these types.

## Chat Protocol Entities

```mermaid
erDiagram
    Message {
        string From "sender agent name"
        string To "recipient: agent name, 'all', or comma-separated"
        string Thread "conversation thread"
        string Type "intent, ack, wait, done, learned, info, ready, shutdown, escalate"
        time TS "timestamp"
        string Text "message content"
    }

    FilePoster {
        string FilePath "chat TOML file path"
        func Lock "LockFile injected"
        func AppendFile "injected"
        func LineCount "injected"
        func NowFunc "injected time source"
    }

    FileWatcher {
        string FilePath "chat TOML file path"
        interface FSWatcher "WaitForChange injected"
        func ReadFile "injected"
    }

    FilePoster ||--o{ Message : "appends"
    FileWatcher ||--o{ Message : "reads and filters"
```

**Source:** `internal/chat/chat.go`, `internal/chat/poster.go`, `internal/chat/watcher.go`

## API Client Entities

```mermaid
erDiagram
    API_Interface {
        method PostMessage "ctx, PostMessageRequest -> PostMessageResponse, error"
        method WaitForResponse "ctx, WaitRequest -> WaitResponse, error"
        method Subscribe "ctx, SubscribeRequest -> SubscribeResponse, error"
        method Status "ctx -> StatusResponse, error"
    }

    Client {
        string baseURL
        interface doer "HTTPDoer injected"
    }

    PostMessageRequest {
        string From
        string To
        string Text
    }

    PostMessageResponse {
        int Cursor
        string Error
    }

    WaitRequest {
        string From
        string To
        int AfterCursor
    }

    WaitResponse {
        string Text
        int Cursor
        string From
        string To
    }

    SubscribeRequest {
        string Agent
        int AfterCursor
    }

    SubscribeResponse {
        int Cursor
    }

    ChatMessage {
        string From
        string To
        string Text
    }

    StatusResponse {
        bool Running
    }

    Client --|> API_Interface : "implements"
    Client ||--o{ PostMessageRequest : "sends"
    Client ||--o{ PostMessageResponse : "receives"
    Client ||--o{ WaitRequest : "sends"
    Client ||--o{ WaitResponse : "receives"
    SubscribeResponse ||--o{ ChatMessage : "contains"
```

**Source:** `internal/apiclient/client.go`

## Server Entities

```mermaid
erDiagram
    Server {
        ptr httpServer
        ptr listener
        ptr logger
    }

    Config {
        string Addr
        ptr Logger
        func PostFunc
        func WatchFunc
        func SubscribeFunc
        func ResetAgentFunc
    }

    Deps {
        func PostMessage "PostFunc"
        func WatchForMessage
        func SubscribeMessages
        func ResetAgent
        ptr Logger
        func ShutdownFn
    }

    AgentLoop {
        ptr config "AgentLoopConfig"
        int cursor
    }

    AgentLoopConfig {
        string Name "agent name"
        bool WatchAll "true for engram-agent"
        chan Notify "from SharedWatcher"
        func ReadMessages "ReadMessagesFunc injected"
        func OnMessage "callback"
    }

    SharedWatcher {
        func waitForChange "WaitFunc injected"
    }

    EngramAgent {
        ptr config "EngramAgentConfig"
        string sessionID
        int invocations
        ptr refresh "RefreshTracker"
    }

    EngramAgentConfig {
        func RunClaude "RunClaudeFunc injected"
        func PostToChat "PostFunc"
        ptr Logger
    }

    AgentResponse {
        string SessionID
        string Action "surface, log-only, learn"
        string To
        string Text
        bool Saved
        string Path
    }

    RefreshTracker {
        int interval
        int count
    }

    Server ||--|| Config : "created from"
    Server ||--|| Deps : "handlers use"
    SharedWatcher ||--o{ AgentLoop : "notifies via channels"
    AgentLoop ||--|| AgentLoopConfig : "configured by"
    AgentLoop }|--|| EngramAgent : "OnMessage invokes"
    EngramAgent ||--|| EngramAgentConfig : "configured by"
    EngramAgent ||--o{ AgentResponse : "parses from claude -p"
    EngramAgent ||--|| RefreshTracker : "skill refresh"
```

**Source:** `internal/server/server.go`, `internal/server/agent.go`, `internal/server/engram.go`, `internal/server/fanout.go`, `internal/server/refresh.go`, `internal/server/stream.go`

## Memory Entities

```mermaid
erDiagram
    MemoryRecord {
        int SchemaVersion
        string Type "feedback or fact"
        string Source
        bool Core
        string Situation "when this memory applies"
        int SurfacedCount
        int FollowedCount
        int NotFollowedCount
        int IrrelevantCount
        int MissedCount
        float InitialConfidence "A=1.0, B=0.7, C=0.4"
        bool ProjectScoped
        string ProjectSlug
        string CreatedAt
        string UpdatedAt
    }

    ContentFields {
        string Behavior "feedback: what was done"
        string Impact "feedback: what resulted"
        string Action "feedback: what to do instead"
        string Subject "fact: subject"
        string Predicate "fact: predicate"
        string Object "fact: object"
    }

    PendingEvaluation {
        string SurfacedAt
        string UserPrompt
        string SessionID
        string ProjectSlug
    }

    Stored {
        string Type
        string Situation
        bool Core
        float InitialConfidence
        bool ProjectScoped
        string ProjectSlug
        int SurfacedCount
        int FollowedCount
        int NotFollowedCount
        int IrrelevantCount
        time UpdatedAt
        string FilePath
    }

    MemoryRecord ||--|| ContentFields : "contains"
    MemoryRecord ||--o{ PendingEvaluation : "pending evaluations"
    MemoryRecord ||--|| Stored : "converts to (in-memory)"
    Stored ||--|| ContentFields : "contains"
```

**Source:** `internal/memory/record.go`, `internal/memory/memory.go`

## MCP Server Entities

```mermaid
erDiagram
    StdoutChannelNotifier {
        mutex mu
        writer io_Writer "stdout"
    }

    AgentNameCapture {
        chan ch "buffered string channel"
        once sync_Once
    }

    ChannelNotifier_Interface {
        method Notify "content, meta -> error"
    }

    ServerStarter_Interface {
        method Start "ctx, apiAddr -> error"
    }

    StdoutChannelNotifier --|> ChannelNotifier_Interface : "implements"
    AgentNameCapture }|--|| StdoutChannelNotifier : "subscribe loop uses"
```

**Source:** `internal/mcpserver/channel.go`, `internal/mcpserver/subscribe.go`, `internal/mcpserver/startup.go`

## Cross-references

- Components that own these types: [C2: Component](c2-component.md)
- How data flows between types: [Sequences](sequences.md)
- Container boundaries: [C3: Container](c3-container.md)
