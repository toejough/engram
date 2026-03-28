# Session Lifecycle

A Claude Code session from engram's perspective.

## Steps

### 1. SessionStart (session-start.sh, 10s timeout)

- Emits system message: "Say /recall to load context from previous sessions, or /recall <query> to search session history"
- Background (async): rebuilds binary if stale, runs `engram maintain` to generate maintenance proposals
- Parses maintain output, counts proposals by category (noise, hidden gem, leech, refine keywords, escalation, consolidation)
- Checks policy.toml for pending adaptation proposals
- If proposals exist, writes pending-maintenance.json with triage summary

### 2. UserPromptSubmit (user-prompt-submit.sh, 30s timeout)

Fires on every user message.

- Rebuilds binary if stale
- Consumes pending-maintenance.json (atomic move to prevent double-read)
- Skips surfacing if message is a skill invocation (/recall, /adapt, etc.)
- Runs `engram correct` -- detects inline corrections in user message
- Runs `engram surface --mode prompt` -- surfaces relevant memories via BM25
- Merges pending + correct + surface outputs into response

### 3. Stop (stop-surface.sh, 15s timeout, blocking)

- Runs `engram surface --mode stop` -- checks agent output for relevant memories
- If memories match, blocks the agent with surfaced context
- Prevents infinite loops via stop_hook_active flag

### 4. Stop (stop.sh, 120s timeout, async)

- Runs `engram flush` -- extracts learnings from transcript, records outcomes
- Fire-and-forget -- always exits 0

## Sequence Diagram

```mermaid
sequenceDiagram
    participant U as User
    participant CC as Claude Code
    participant E as Engram

    Note over CC,E: Session Start
    CC->>E: SessionStart hook
    E-->>CC: "Say /recall to load context, or /recall <query> to search"
    E->>E: (async) build binary, run maintain
    E->>E: (async) write pending-maintenance.json

    Note over U,E: User Prompt Loop
    U->>CC: Message
    CC->>E: UserPromptSubmit hook
    E->>E: Consume pending-maintenance.json
    E->>E: engram correct (detect corrections)
    E->>E: engram surface --mode prompt
    E-->>CC: Surfaced memories + corrections + triage
    CC->>U: Response (with memory context)

    Note over CC,E: Agent Completion
    CC->>E: Stop hook (blocking)
    E->>E: engram surface --mode stop
    alt Relevant memories found
        E-->>CC: Block with memory context
        CC->>U: Additional context shown
    else No matches
        E-->>CC: Allow completion
    end

    Note over CC,E: Session End
    CC->>E: Stop hook (async)
    E->>E: engram flush (learn from transcript)
```
