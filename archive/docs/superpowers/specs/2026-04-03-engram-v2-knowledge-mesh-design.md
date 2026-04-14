# Engram V2: Knowledge Mesh Design

Evolves engram from behavioral-correction-only memory (SBIA) to a unified knowledge system with two memory types (feedback + facts), project-scoped agent coordination, and an orchestrator that manages agent teams through tmux.

## Part 1: Unified Memory Format

### Two Types

| Type | Purpose | Content Fields |
|------|---------|---------------|
| **Feedback** | Behavioral corrections — "when X, don't do Y, do Z" | behavior, impact, action |
| **Fact** | Propositional knowledge — subject-predicate-object triples | subject, predicate, object |

### File Format

```toml
schema_version = 1
type = "feedback"  # or "fact"
situation = "When running build commands in the engram project"
source = "user correction, 2026-04-02"
core = false  # user-pinned to always-loaded set

[content]
# feedback fields:
behavior = "Running go test directly"
impact = "Misses coverage thresholds and lint checks"
action = "Use targ test instead"

# fact fields (mutually exclusive with feedback):
# subject = "engram"
# predicate = "uses"
# object = "targ for all build, test, and check operations"

# Tracking (shared)
surfaced_count = 0
followed_count = 0
not_followed_count = 0
irrelevant_count = 0
missed_count = 0
initial_confidence = 1.0
created_at = "2026-04-02T10:00:00Z"
updated_at = "2026-04-02T10:00:00Z"
```

### Data Layout

```
~/.local/share/engram/
├── chat/
│   └── <project-slug>.toml       # per-project coordination
└── memory/
    ├── facts/
    │   └── engram-uses-targ.toml
    └── feedback/
        └── mandatory-full-build-before-merge.toml
```

Project slug is derived from `$PWD` (same convention as `engram recall`).

### Retrieval Strategy

No links, no tags. Structure in the content fields provides connectivity.

1. **Load core + recent** memory situations into context. Core = user-pinned (`core = true`) + auto-promoted (high effectiveness ratio). Recent = created or surfaced in last N sessions.
2. **Match situations** against the current intent using LLM judgment.
3. **On match**, read the full memory file.
4. **Search for related** memories by overlapping subjects/objects (facts) or similar situations (feedback). Read those on the fly.
5. **Surface** with feedback taking priority over facts. Feedback triggers arguments (WAIT). Facts are surfaced as additional context (INFO).

### Auto-Promotion to Core

Memories with high `followed_count` relative to `surfaced_count` auto-promote to core. Memories with high `surfaced_count` and zero `followed_count` (repeatedly surfaced, never useful) decay out over time. User can manually pin/unpin via the `core` field.

### Knowledge Patterns

These are extraction guidance for the engram-agent, not storage types. All knowledge is stored as feedback or facts.

| Knowledge | How to encode |
|-----------|--------------|
| Simple fact | One fact triple |
| Concept/definition | Multiple facts sharing a subject: "X → is → definition", "X → contains → Y" |
| Decision | Fact cluster: "X → chose → Y" + "X → rejected → Z" per alternative + "X → because → rationale" |
| Excerpt/quote | "source → says → content" |
| Process/procedure | Ordered facts sharing a subject: "X → step 1 → Y", "X → step 2 → Z" |
| Behavioral correction | One feedback memory with situation + behavior/impact/action |

## Part 2: Skills

### Skill Inventory

| Skill | Replaces | Type |
|-------|----------|------|
| `use-engram-chat-as` | `file-comms` | Coordination protocol |
| `engram-agent` | `memory-agent` | Reactive memory watcher |
| `engram-tmux-lead` | NEW | Orchestrator |
| `recall` | (unchanged) | Cross-session context search |

### use-engram-chat-as

Join the project's chat as a named agent with a role. Includes the full coordination protocol.

**Chat file:** `~/.local/share/engram/chat/<project-slug>.toml`, derived from `$PWD`.

**Message types:** `intent`, `ack`, `wait`, `info`, `done`, `learned`

- `intent` — announce situation + planned behavior before acting
- `ack` — no objection, or early concession during argument
- `wait` — objection, memory to surface, or request to pause
- `info` — informational, status updates, user-parroted input
- `done` — task/action completed
- `learned` — agent announces knowledge it extracted from its work

**Roles:**
- **Active** — broadcasts intent, waits for responses, parrots user input. Can react to others.
- **Reactive** — never broadcasts intent, only reacts. Skips intent protocol for own actions.

**User input parroting:** Active agents must parrot user submissions as `info` messages. Honor-system.

**Learned messages:** Active agents should announce what they learned from their work as `learned` messages. This gives the engram-agent a high-confidence signal for fact extraction.

**Argument protocol:** Factual initiator, aggressive reactor, 3 argument inputs max, 4th escalation via initiating agent's UX. Early concession via `ack`. Resolution recorded by reactor.

**Watch loop:** Background task `fswatch -1` pattern. Start fswatch as background Bash command, wait for notification, process, start new fswatch. Never complete turn while loop is running.

**Heartbeat:** Reactive agents post every 5 minutes with stats.

**Role argument:** Free-form text shaping behavior:
- `reactive memory agent named engram-agent`
- `reviewer named bob, who uses code review skills`
- `/engram-tmux-lead` — special case, loads lead orchestrator behavior

### engram-agent

Reactive memory watcher. Uses `use-engram-chat-as` to join as reactive. One agent handling both feedback and facts.

**Feedback responsibilities (existing, updated):**
- Surfaces feedback memories when intent situations match
- Learns from explicit user corrections (confidence 1.0) and observed failures (0.7 high-confidence, 0.4 medium, 0.2 inferred)
- Spawns subagents for arguments (aggressive reactor, max 3, monotonic IDs, thread exclusivity)
- Writes to `~/.local/share/engram/memory/feedback/`

**Fact responsibilities (new):**
- Surfaces relevant facts when intent subjects/objects overlap with known facts
- Learns facts from `learned` messages (high confidence — agent self-reported)
- Learns facts from general conversation (lower confidence — inferred)
- Extracts subject/predicate/object triples
- Deduplicates by subject + predicate (update object if new info, skip if identical)
- Writes to `~/.local/share/engram/memory/facts/`

**Processing order:** Per message, check feedback triggers first (corrections, failures), then fact triggers. Feedback and facts are different — don't confuse them.

**Surfacing priority:** Feedback triggers arguments (WAIT). Facts are surfaced as additional context (INFO). Feedback takes priority when both match.

**Tiered loading:**
- Core (pinned + auto-promoted): always loaded
- Recent (last N sessions): loaded on startup
- On-demand: search by overlapping subjects/objects or similar situations when a match is found

**Split signal:** If the agent miscategorizes (feedback as facts or vice versa) or misses one type because it's focused on the other, split into two agents.

**Locking:** Per-file locks, atomic writes (temp + rename), stale lock recovery (PID-based, 300s mkdir timeout), no multi-file locking.

### engram-tmux-lead

The user's primary agent. User talks only to the lead. All other agents are behind the scenes.

**Startup sequence:**
1. User starts claude, says `/use-engram-chat-as /engram-tmux-lead`
2. Lead joins chat as active agent named "lead"
3. Lead spins up engram-agent (reactive, memory + facts) in a tmux window
4. Lead spins up a general-purpose executor (active) in a tmux window
5. Lead is ready for user instructions

**Core responsibilities:**

- **Proxy:** All user input goes to the lead. Lead decides which agent(s) need it and relays via chat. Agent questions come back through chat, lead curates and presents them to the user. Other agents never talk to the user directly.
- **Lifecycle:** Spins up agents in tmux windows. Tears them down when done. Nudges via tmux if an agent goes quiet in chat.
- **Routing:** Default work goes to the executor. Specialized work gets specialized agents (reviewer, researcher, planner). Lead decides based on the task.
- **Opinionated defaults:** "Tackle issue 528" → lead reads the issue, spins up planner + executor + reviewer. "File an issue" → spins up issue-filer. User can override.

**Agent instructions template:** Every agent the lead spins up gets:
- `/use-engram-chat-as [role] named [name]`
- Funnel all questions for the user through chat addressed to "lead"
- Never ask the user directly
- Use relevant skills for your role

**Monitoring:** Lead watches chat for agent activity. If an agent hasn't posted in a reasonable time, lead checks its tmux window and nudges or reports to the user.

**The lead monitors chat via fswatch** like any other participant. When agents post questions or status updates, the lead wakes up, reads the message, and decides whether to surface it to the user or handle it autonomously. The lead is an active participant — posting user instructions, relaying questions, managing lifecycle — while also watching for agent activity between user interactions.

## Part 3: Migration

### Memory file migration (269 files)

Per file in `~/.local/share/engram/memories/`:
1. Add `schema_version = 1`
2. Add `type = "feedback"`
3. Add `source = ""`
4. Nest `behavior`, `impact`, `action` under `[content]`
5. Strip `pending_evaluations` if present
6. Move to `~/.local/share/engram/memory/feedback/`

Write a one-time migration script.

### Go binary updates

- `MemoryRecord` struct updated to new format (type, source, nested content)
- `recall` and `show` read from new directory paths (`memory/feedback/`, `memory/facts/`)
- Session-start hook updated if directory paths changed

### Skill changes

- Delete: `skills/file-comms/SKILL.md`, `skills/memory-agent/SKILL.md`
- Create: `skills/use-engram-chat-as/SKILL.md`, `skills/engram-agent/SKILL.md`, `skills/engram-tmux-lead/SKILL.md`
- Update: `skills/recall/SKILL.md` (new directory paths)

### What stays

- engram Go binary (recall + show)
- Session-start hook (build + recall announcement)
- Memory TOML files (migrated to new format + location)

## Part 4: Deferred

- Multi-channel chat (#472) — per-topic or per-pair channels
- Hook-based intent strengthening (#470) — PreToolUse hooks that auto-write intents
- BM25 pre-filtering (#471) — needed when memories exceed ~1000
- Maintenance, consolidation, adapt analysis (#471) — memory hygiene skills
- Temporal validity on facts (valid_from/valid_until) — facts that change over time
