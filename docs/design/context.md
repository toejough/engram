# System Context

Engram is a Claude Code plugin (Go binary + shell hooks + skills) that manages memory extraction, surfacing, evaluation, and maintenance for LLM agent sessions.

## Actors

| Actor | Description |
|-------|-------------|
| **Developer** | Uses Claude Code in a terminal session |
| **Claude Code** | Anthropic's CLI agent that dispatches hook events and skill invocations |
| **Engram** | Self-correcting memory plugin -- learns, surfaces, evaluates, and maintains memories |
| **Anthropic API** | LLM service used for classification, extraction, and consolidation |
| **Local Filesystem** | Memory TOML files, surfacing logs, and policy config on disk |

## C4 Context Diagram

```mermaid
C4Context
    title Engram System Context

    Person(user, "Developer", "Uses Claude Code for software engineering tasks")
    System(claudeCode, "Claude Code", "Anthropic's CLI agent — dispatches hook events and skill invocations")
    System(engram, "Engram", "Self-correcting memory plugin — learns, surfaces, evaluates, and maintains memories")
    System_Ext(anthropicAPI, "Anthropic API", "LLM service for classification, extraction, and consolidation")
    SystemDb(filesystem, "Local Filesystem", "Memory TOML files, surfacing logs, policy config")

    Rel(user, claudeCode, "Interacts via terminal")
    Rel(claudeCode, engram, "Hook events, skill invocations")
    Rel(engram, anthropicAPI, "LLM calls for classify, extract, consolidate")
    Rel(engram, filesystem, "Read/write memories, logs, policy")
    Rel(claudeCode, user, "Responses with injected memory context")
```

## Relationships

- **Claude Code -> Engram:** Claude Code fires hook events (`SessionStart`, `UserPromptSubmit`, `Stop`) and engram responds with surfaced memories and corrections.
- **Engram -> Anthropic API:** Engram calls the Anthropic API for LLM-dependent operations (extraction, classification, consolidation).
- **Engram -> Filesystem:** All persistent state lives on the local filesystem as TOML files and JSONL logs.
