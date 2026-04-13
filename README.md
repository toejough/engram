# Engram

> **Alpha** — Under active development. APIs and data formats may change. Feedback and contributions welcome via [issues](https://github.com/toejough/engram/issues).

Self-correcting memory for LLM agents. A [Claude Code plugin](https://docs.anthropic.com/en/docs/claude-code/plugins) that learns from sessions, surfaces relevant memories at the right moment, measures whether they're actually followed, and fixes the ones that aren't.

## The problem

Claude Code has several instruction sources — CLAUDE.md, rules, skills — but no way to know if they're working. An instruction that's always loaded but never followed wastes context budget. A great instruction that only matches narrow keywords goes unseen. Without measurement, instruction sets decay: duplicates accumulate, stale guidance persists, and useful patterns stay buried.

## How engram solves it

Engram maintains structured memories (corrections as SBIA feedback, knowledge as SPO facts), surfaces them when situationally relevant, measures whether they improve outcomes, and diagnoses failures. See [Intent & Rationale](docs/intent.md) for the research and design decisions behind this approach.

## Quick start

### Prerequisites

- Go 1.25+
- Claude Code 2.1.81+
- The engram plugin enabled in Claude Code

### Installation

```bash
git clone https://github.com/toejough/engram.git
```

Enable in Claude Code. The binaries auto-build on first session start.

### Starting a session with engram

**Option A: MCP mode (recommended)** — memories push to you automatically:

```bash
# The MCP server auto-starts the API server
claude --dangerously-load-development-channels 'plugin:engram@engram'
```

Then load the skills: `/use-engram-chat-as` and `/engram-lead`.

**Option B: CLI mode** — you explicitly request memories:

```bash
# Start the API server
engram server up --chat-file ~/.local/share/engram/chat/my-project.toml --log-file /tmp/engram.log

# In another terminal, use CLI commands
engram intent --from my-session --to engram-agent --situation "about to refactor auth module"
engram learn --from my-session --type feedback --situation "writing tests" --behavior "skipped edge cases" --impact "bugs in production" --action "always test error paths"
```

### CLI commands

| Command | Purpose | Blocking? |
|---------|---------|-----------|
| `engram post --from X --to Y --text "..."` | Post a message to chat | No |
| `engram intent --from X --to Y --situation "..." --planned-action "..."` | Ask for memories before acting | Yes (waits for response) |
| `engram learn --from X --type feedback\|fact --situation "..." [fields]` | Record a learning | No |
| `engram subscribe --agent X` | Watch for messages addressed to you | Yes (long-poll loop) |
| `engram status` | Check API server health | No |
| `engram server up --chat-file PATH --log-file PATH --addr HOST:PORT` | Start the API server | Yes (runs until stopped) |

### MCP tools (when using MCP mode)

| Tool | Purpose |
|------|---------|
| `engram_post` | Post a message to chat |
| `engram_intent` | Ask for memories before acting (synchronous) |
| `engram_learn` | Record a feedback or fact |
| `engram_status` | Check server health |

Surfaced memories arrive as `<channel source="engram">` events between turns.

### Shutting down

```bash
# Via CLI
engram post --from my-session --to engram-agent --text "shutdown"
curl -X POST http://localhost:7932/shutdown

# The MCP server shuts down when the Claude Code session ends
```

## Memory formats

### Feedback (SBIA)

Corrections anchored to specific situations. [Why SBIA?](docs/intent.md#feedback-memories-sbia)

```toml
type = "feedback"
situation = "writing Go code that creates temporary files"
[content]
behavior = "using predictable temp file names, not cleaning up on failure"
impact = "security vulnerabilities, resource leaks"
action = "use os.CreateTemp, defer cleanup on every failure path"
```

### Facts (SPO)

Propositional knowledge as subject-predicate-object triples. [Why SPO?](docs/intent.md#fact-memories-spo)

```toml
type = "fact"
situation = "referencing engram chat file paths"
[content]
subject = "engram chat files"
predicate = "stored-in"
object = "~/.local/share/engram/chat/<project-slug>.toml"
```

## Effectiveness quadrants

|  | High effectiveness | Low effectiveness |
|--|-------------------|------------------|
| **Often surfaced** | **Working** — Keep as-is | **Leech** — Rewrite or escalate |
| **Rarely surfaced** | **Hidden Gem** — Broaden retrieval | **Noise** — Remove |

Effectiveness = followed / (followed + not_followed + irrelevant). See [Formulas](docs/design/formulas.md).

## Common edge cases

### Server won't start
- Check if port 7932 is already in use: `lsof -i :7932`
- Check the log file for errors: `tail -f /tmp/engram.log | jq .`
- The MCP server auto-starts the API server — you don't need to start both

### Memories not surfacing
- Verify the server is running: `engram status`
- Check the chat file for recent messages: `tail ~/.local/share/engram/chat/<slug>.toml`
- Check the debug log for engram-agent errors
- The engram-agent may need a session reset: `curl -X POST http://localhost:7932/reset-agent`

### Learn messages rejected
- Feedback requires: `situation`, `behavior`, `impact`, `action`
- Facts require: `situation`, `subject`, `predicate`, `object`
- The error message tells you which field is missing

### Channel events not appearing (MCP mode)
- Requires `--dangerously-load-development-channels 'plugin:engram@engram'`
- Channels are a research preview feature in Claude Code
- Events appear between turns, not during tool execution

## Data files

| Path | Purpose |
|------|---------|
| `~/.local/share/engram/chat/<slug>.toml` | Chat file (inter-agent messages) |
| `~/.local/share/engram/memory/facts/*.toml` | Fact memories |
| `~/.local/share/engram/memory/feedback/*.toml` | Feedback memories |

## Skills

| Skill | Purpose |
|-------|---------|
| `/use-engram-chat-as` | Communication protocol for agents using engram |
| `/engram-lead` | Lead agent orchestration guide |
| `/engram-agent` | Memory agent behavior (structured JSON output) |
| `/engram-up` | Start engram session |
| `/engram-down` | Shut down engram session |
| `/recall` | Load context from previous sessions |

## Architecture

- [Intent & Rationale](docs/intent.md) — Why engram exists and what must survive rewrites
- [Architecture Diagrams](docs/architecture/README.md) — C4 model from system context to code entities
- [Memory Lifecycle](docs/design/memory-lifecycle.md) — Creation to deletion with state diagram
- [Formulas](docs/design/formulas.md) — Effectiveness, ranking, budget formulas
- [Spec Design Principles](docs/spec-design.md) — Principles for designing engram specs
- [Execution Planning Principles](docs/exec-planning.md) — Principles for implementation planning

## Design principles

- **Content quality > mechanical sophistication** — Fix the memory, not the retrieval machinery
- **Measure impact, not frequency** — surfaced_count is vanity; followed_count is value
- **DI everywhere** — No `internal/` function calls `os.*`, `http.*`, or I/O directly
- **Server-side intelligence, thin clients** — Deterministic routing in the server, not unreliable agents
- **Observable by default** — Chat log, debug log, structured JSON logging
- **Pure Go, no CGO** — BM25 for retrieval, external API for LLM judgment only

## What about built-in memory?

Anthropic has introduced built-in auto-memory and dreaming features for Claude Code. Memory is one of the primary problems that needs solving, and lots of smart people are working on it.

I tried auto-memory when it was introduced and found it unhelpful for the same reasons I'd had trouble with "please write this down somewhere" — non-deterministic recording, unreliable surfacing. I ended up turning it off because it was conflicting with engram. I haven't tried the new dreaming features yet — maybe they've materially improved things.

If the built-in memory management becomes good enough that most of engram is unnecessary, I'll be glad — less to maintain. Till then, engram exists because I wanted memory that **measures outcomes**, **self-corrects**, and **keeps the user in control** of what changes.
