# C4: System Context

How engram fits into the broader environment. See [C3: Container](c3-container.md) for what's inside engram.

```mermaid
C4Context
    title System Context: Engram

    Person(user, "User", "Human developer using Claude Code")

    System(engram, "Engram", "Self-correcting memory for LLM agents. Records corrections and facts, surfaces them at the right moment, measures impact, diagnoses failures.")

    System_Ext(claudeCode, "Claude Code", "Anthropic's CLI agent. Hosts engram as a plugin. Provides hooks, MCP, and skills.")
    System_Ext(claudeBinary, "Claude CLI (claude -p)", "Non-interactive Claude invocations. Used by engram server to run the memory agent.")
    System_Ext(filesystem, "Local Filesystem", "TOML chat files, TOML memory files, session transcripts, debug logs.")

    Rel(user, claudeCode, "Interacts via terminal")
    Rel(claudeCode, engram, "Plugin: hooks, MCP tools, skills")
    Rel(engram, claudeBinary, "Invokes claude -p --resume for memory agent")
    Rel(engram, filesystem, "Reads/writes TOML chat + memory files")
    Rel(claudeCode, filesystem, "Reads session transcripts")
```

## Actors

| Actor | Role | Interaction |
|-------|------|-------------|
| **User** | Human developer | Prompts Claude Code; corrections become memories |
| **Claude Code** | Host agent | Runs engram hooks on prompt/stop; calls MCP tools; loads skills |
| **Claude CLI** | Memory agent runtime | Server invokes `claude -p --resume` to run the engram-agent |
| **Filesystem** | Persistence | Chat file (source of truth), memory TOML files, debug logs |

## Boundary

Everything inside "Engram" is detailed in [C3: Container](c3-container.md).
