# Sequence Diagrams

How data flows across boundaries at each level. Cross-referenced with [C4](c4-context.md), [C3](c3-container.md), [C2](c2-component.md), and [C1](c1-code.md).

## System Level: User Interaction (C4 boundary)

```mermaid
sequenceDiagram
    participant User
    participant ClaudeCode as Claude Code
    participant Engram
    participant Filesystem

    User->>ClaudeCode: types prompt
    ClaudeCode->>Engram: UserPromptSubmit hook
    Engram->>Filesystem: post user words to chat
    Engram->>Engram: engram-agent surfaces memories
    Engram-->>ClaudeCode: surfaced memories (additionalContext or channel push)
    ClaudeCode->>User: response with memory-informed context

    User->>ClaudeCode: "remember: always use DI"
    ClaudeCode->>Engram: engram_learn MCP tool
    Engram->>Filesystem: write memory TOML file
    Engram-->>ClaudeCode: confirmation
```

**Actors:** [C4: Context](c4-context.md)

## Container Level: Synchronous Intent Flow (C3 boundary)

The agent explicitly asks for memories before acting.

```mermaid
sequenceDiagram
    participant Agent as Claude Code Agent
    participant MCP as MCP Server
    participant API as API Server
    participant Chat as Chat File (TOML)
    participant EA as Engram-Agent (claude -p)

    Agent->>MCP: engram_intent(from, to, situation, action)
    MCP->>API: POST /message (intent)
    API->>Chat: append intent message
    Note over API: fsnotify detects change
    API->>EA: claude -p --resume (with intent)
    EA-->>API: {"action":"surface","to":"lead","text":"Memory: ..."}
    API->>Chat: append surface response
    API-->>MCP: GET /wait-for-response returns
    MCP-->>Agent: surfaced memories (tool result)
```

**Containers:** [C3: Container](c3-container.md)

## Container Level: Async Channel Push (C3 boundary)

Memories arrive without the agent requesting them.

```mermaid
sequenceDiagram
    participant Hook as Stop Hook
    participant CLI as CLI Client
    participant API as API Server
    participant Chat as Chat File
    participant EA as Engram-Agent
    participant MCP as MCP Server
    participant Agent as Claude Code Agent

    Hook->>CLI: engram post (agent output)
    CLI->>API: POST /message
    API->>Chat: append message
    Note over API: fsnotify -> engram-agent goroutine
    API->>EA: claude -p --resume
    EA-->>API: {"action":"surface","to":"lead","text":"..."}
    API->>Chat: append response
    Note over MCP: subscribe loop sees new message
    MCP->>API: GET /subscribe returns
    MCP-->>Agent: notifications/claude/channel
    Note over Agent: sees <channel source="engram"> on next turn
```

**Containers:** [C3: Container](c3-container.md)

## Component Level: API Server Message Routing (C2 boundary)

Inside the API server: how a posted message reaches the engram-agent.

```mermaid
sequenceDiagram
    participant Client as HTTP Client
    participant Handler as HandlePostMessage
    participant Poster as FilePoster
    participant Chat as Chat File
    participant Fanout as SharedWatcher
    participant Loop as AgentLoop (engram-agent)
    participant EA as EngramAgent
    participant Claude as claude -p

    Client->>Handler: POST /message {from, to, text}
    Handler->>Handler: validate learn message (if applicable)
    Handler->>Poster: Post(Message)
    Poster->>Chat: append with file lock
    Poster-->>Handler: cursor
    Handler-->>Client: {cursor: N}

    Note over Fanout: fsnotify fires
    Fanout->>Loop: buffered channel notification
    Loop->>Loop: read from cursor, filter messages
    Loop->>EA: OnMessage(msg)
    EA->>EA: check skill refresh (every 13)
    EA->>Claude: claude -p --resume (prompt)
    Claude-->>EA: stream-json output
    EA->>EA: ParseStreamResponse -> AgentResponse
    EA->>Poster: Post(response message)
```

**Components:** [C2: Component](c2-component.md)

## Component Level: Error Recovery (C2 boundary)

The engram-agent's error recovery ladder.

```mermaid
sequenceDiagram
    participant Loop as AgentLoop
    participant EA as EngramAgent
    participant Claude as claude -p

    Loop->>EA: ProcessWithRecovery(msg)

    rect rgb(255, 240, 240)
        Note over EA,Claude: Attempt on current session (up to 3)
        EA->>Claude: claude -p --resume (prompt)
        Claude-->>EA: malformed output
        EA->>Claude: re-prompt with format guidance
        Claude-->>EA: malformed again
        EA->>Claude: re-prompt (3rd attempt)
        Claude-->>EA: malformed (3 failures)
    end

    rect rgb(240, 240, 255)
        Note over EA: Session reset
        EA->>EA: ResetSession() clears session ID
        Note over EA,Claude: Attempt on fresh session (up to 3)
        EA->>Claude: claude -p (fresh, full skill load + last 3 messages)
        Claude-->>EA: valid response
        EA-->>Loop: success
    end

    Note over EA: If fresh session also fails 3x:
    EA->>EA: escalate: post error to chat, log critical, stop invoking
```

**Components:** [C2: Component](c2-component.md), Types: [C1: Code](c1-code.md)

## Component Level: MCP Subscribe Loop (C2 boundary)

How the MCP server pushes memories to the agent.

```mermaid
sequenceDiagram
    participant Tool as First Tool Call
    participant Capture as AgentNameCapture
    participant Sub as Subscribe Loop
    participant API as API Server
    participant Notifier as StdoutChannelNotifier
    participant Agent as Claude Code Agent

    Tool->>Capture: Set("lead-1") on first call
    Capture-->>Sub: Wait() returns "lead-1"

    loop Long-poll
        Sub->>API: GET /subscribe?agent=lead-1&after-cursor=N
        API-->>Sub: {messages: [...], cursor: M}
        Sub->>Notifier: Notify(content, meta) for each message
        Notifier->>Agent: {"jsonrpc":"2.0","method":"notifications/claude/channel",...}
        Note over Sub: cursor = M
    end
```

**Components:** [C2: Component](c2-component.md), Types: [C1: Code](c1-code.md)

## Entity Level: Memory Lifecycle (C1 boundary)

How a MemoryRecord moves through states. See [Memory Lifecycle](../design/memory-lifecycle.md) for the full state diagram.

```mermaid
sequenceDiagram
    participant Agent as Claude Code Agent
    participant Learn as engram_learn tool
    participant EA as Engram-Agent
    participant FS as Memory File (TOML)

    Agent->>Learn: engram_learn(type=feedback, situation, behavior, impact, action)
    Learn->>Learn: validate fields (situation + behavior + impact + action required)
    Learn->>EA: post to chat as learn message

    EA->>EA: evaluate: worth saving?
    alt Save
        EA->>FS: write MemoryRecord.toml
        Note over FS: surfaced_count=0, followed_count=0
    else Discard
        EA->>EA: log decision, no file write
    end

    Note over FS: Later: surfaced at prompt
    FS->>FS: surfaced_count++

    Note over FS: Session end: evaluation
    alt Followed
        FS->>FS: followed_count++
    else Not followed
        FS->>FS: not_followed_count++
    else Irrelevant
        FS->>FS: irrelevant_count++
    end

    Note over FS: Maintenance: diagnose quadrant
    FS->>FS: effectiveness = followed / (followed + not_followed + irrelevant)
```

**Types:** [C1: Code](c1-code.md)

## Cross-references

| Sequence | Level | Related Diagrams |
|----------|-------|-----------------|
| User Interaction | C4 | [Context](c4-context.md) |
| Intent Flow | C3 | [Container](c3-container.md) |
| Async Push | C3 | [Container](c3-container.md) |
| Message Routing | C2 | [Component](c2-component.md) |
| Error Recovery | C2 | [Component](c2-component.md), [Code](c1-code.md) |
| Subscribe Loop | C2 | [Component](c2-component.md), [Code](c1-code.md) |
| Memory Lifecycle | C1 | [Code](c1-code.md) |
