# Engram

Self-correcting memory for LLM agents.

## Overview

Engram is a [Claude Code plugin](https://docs.anthropic.com/en/docs/claude-code/plugins) that gives agents persistent memory. Four skills help agents prepare for work, recall context, learn from experience, and remember explicitly.

## Core loop

```
/prepare --> work --> /learn
   |                    |
   |  /recall  /remember |
   |  (search) (capture) |
   +--------------------+
```

`/prepare` loads relevant context before starting work. `/learn` extracts feedback after work completes. `/recall` searches session history and memories on demand. `/remember` captures explicit knowledge immediately.

## Skills

| Skill | Purpose |
|-------|---------|
| `/recall` | Search session transcripts and memories. No args = recent session summary. With query = Haiku-filtered search across sessions and memory files. |
| `/prepare` | Load context before starting work. Runs `/recall` for the current task, surfaces relevant memories, and primes the agent with project state. |
| `/learn` | Extract feedback after completing work. Reviews what happened, identifies patterns, and writes feedback memories for future sessions. |
| `/remember` | Capture explicit knowledge. "Remember that X" writes a fact or feedback memory immediately, with duplicate detection. |

## Binary commands

The `engram` binary provides CLI access to memory operations:

```
engram recall      Recall recent session context
engram list        List all memories with type, name, and situation
engram learn feedback --behavior "..." --impact "..." --action "..." --source human --situation "..."
engram learn fact    --subject "..." --predicate "..." --object "..." --source human --situation "..."
engram update      Modify fields of an existing memory (--name required)
engram show        Display full memory details (--name required)
```

## Memory format

Memories use a v2 TOML schema with two types: **feedback** and **fact**.

**Feedback** (behavioral observations):

```toml
schema_version = 2
type = "feedback"
source = "agent"
situation = "implementing new features"

[content]
behavior = "skipped writing tests before implementation"
impact = "bugs found late, rework required"
action = "always write failing test first (TDD red phase)"

created_at = "2026-04-14T12:00:00Z"
updated_at = "2026-04-14T12:00:00Z"
```

**Fact** (declarative knowledge):

```toml
schema_version = 2
type = "fact"
source = "human"
situation = "building engram"

[content]
subject = "engram"
predicate = "uses"
object = "targ build system for all build/test/check operations"

created_at = "2026-04-14T12:00:00Z"
updated_at = "2026-04-14T12:00:00Z"
```

## Data directory

All data lives in `~/.local/share/engram/` (respects `$XDG_DATA_HOME`):

```
~/.local/share/engram/
  memory/
    feedback/    Behavioral observation memories
    facts/       Declarative knowledge memories
```

## Installation

Requires Go 1.25+.

```bash
git clone https://github.com/toejough/engram.git
```

Enable in Claude Code via the `/plugin` command.

## Project structure

```
cmd/engram/          CLI entry point (thin wiring layer)
internal/            Business logic (DI boundaries)
skills/              Plugin skills (recall, prepare, learn, remember)
hooks/               Shell hooks for Claude Code integration
.claude-plugin/      Plugin manifest
archive/             Historical planning artifacts
```

## Design principles

- **DI everywhere** -- No function in `internal/` calls `os.*`, `http.*`, or any I/O directly. All I/O through injected interfaces, wired at CLI edges.
- **Pure Go, no CGO** -- TF-IDF for text similarity. External API for LLM classification only.
- **Plugin form factor** -- Skills for behavior, slim Go binary for computation.
- **Measure impact, not frequency** -- Content quality over mechanical sophistication.
