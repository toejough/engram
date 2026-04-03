# Spec 1: Memory Format, Migration & Agent Facts

**Status:** Draft v2
**Depends on:** Nothing (foundational)
**Blocks:** Spec 2 (chat protocol), Spec 3 (tmux lead)

## Goal

After Spec 1 ships, a user can run the engram-agent alongside other agents and get both feedback surfacing AND fact surfacing/learning working. The agent watches `chat.toml` for intent messages using the existing file-comms protocol, and reacts with both memory types.

## Non-Goals

- Full `use-engram-chat-as` skill rename (Spec 2)
- `learned` message type (Spec 2) — facts are learned from conversation observation, not agent self-report
- `engram-tmux-lead` orchestrator (Spec 3)
- Multi-channel chat, BM25 pre-filtering, maintenance skills (deferred)

## Temporal Coupling Note

Spec 2 introduces the `learned` message type, which gives the engram-agent a high-confidence (1.0) signal for fact extraction from agent self-reports. This spec defines fact learning at 0.7/0.4 confidence from conversation observation only. When Spec 2 ships, the engram-agent will gain an additional fact learning path (`learned` → 1.0 confidence). The two paths coexist — `learned` messages are a higher-confidence supplement, not a replacement for conversation observation.

---

## 1. Unified Memory Format

### 1.1 Two Types

| Type | Purpose | Content Fields |
|------|---------|---------------|
| **Feedback** | Behavioral corrections — "when X, don't Y, do Z" | `behavior`, `impact`, `action` |
| **Fact** | Propositional knowledge — subject-predicate-object triples | `subject`, `predicate`, `object` |

Types are mutually exclusive. A memory file is exactly one type.

### 1.2 File Format

```toml
schema_version = 1
type = "feedback"  # or "fact"
situation = "When running build commands in the engram project"
source = "user correction, 2026-04-02"
core = false  # user-pinned to always-loaded set
project_scoped = false
project_slug = ""

[content]
# feedback type:
behavior = "Running go test directly"
impact = "Misses coverage thresholds and lint checks"
action = "Use targ test instead"

# OR fact type (mutually exclusive):
# subject = "engram"
# predicate = "uses"
# object = "targ for all build, test, and check operations"

# Tracking (shared, all types)
surfaced_count = 0
followed_count = 0
not_followed_count = 0
irrelevant_count = 0
missed_count = 0
initial_confidence = 1.0
created_at = "2026-04-02T10:00:00Z"
updated_at = "2026-04-02T10:00:00Z"
```

### 1.3 Changes from Legacy Format

Legacy files (pre-v2) have `schema_version` absent or `schema_version = 1` with no `type` field. The new format is differentiated by the presence of the `type` field.

| Field | Legacy | New |
|-------|--------|----|
| `schema_version` | absent or 1 | 1 |
| `type` | absent (implicit feedback) | required: `"feedback"` or `"fact"` |
| `source` | absent | optional (free text: origin of the memory, default `""`) |
| `core` | absent | optional, default `false` |
| `behavior`, `impact`, `action` | top-level | nested under `[content]` |
| `subject`, `predicate`, `object` | N/A | nested under `[content]` (fact type only) |
| `pending_evaluations` | present | removed |

### 1.4 Validation Rules

- `type` must be `"feedback"` or `"fact"`
- Feedback: `content.behavior`, `content.impact`, `content.action` all required, non-empty
- Fact: `content.subject`, `content.predicate`, `content.object` all required, non-empty
- Feedback files must NOT have `subject`/`predicate`/`object` in `[content]`
- Fact files must NOT have `behavior`/`impact`/`action` in `[content]`
- `situation` required, non-empty (both types)
- `source` optional, default `""` (empty string is valid)
- `initial_confidence` must be in `[0.0, 1.0]`
- All counter fields default to 0

---

## 2. Data Layout

### 2.1 New Directory Structure

```
~/.local/share/engram/
├── chat/
│   └── <project-slug>.toml       # per-project coordination (Spec 2 fully defines)
├── memory/
│   ├── facts/
│   │   └── engram-uses-targ.toml
│   └── feedback/
│       └── mandatory-full-build-before-merge.toml
├── archive/                       # unchanged
├── projects/                      # unchanged
├── creation-log.jsonl             # unchanged
├── policy.toml                    # unchanged
└── ...                            # other existing files unchanged
```

### 2.2 Directory Changes

| Old | New | Notes |
|-----|-----|-------|
| `data/memories/*.toml` | `data/memory/feedback/*.toml` | All existing memories are feedback type |
| N/A | `data/memory/facts/*.toml` | New directory for fact memories |
| N/A | `data/chat/` | Created but not heavily used until Spec 2 |

### 2.3 File Naming

- Feedback: slug derived from situation, same convention as today (kebab-case, max 80 chars, numeric suffix `-2`, `-3` on collision)
- Facts: slug derived from `subject-predicate-object` (kebab-case, max 80 chars, numeric suffix `-2`, `-3` on collision)

---

## 3. Migration

### 3.1 Scope

Migrate all files in `~/.local/share/engram/memories/` to the new format and location. Current count: ~269 files.

### 3.2 Migration Script

Go program at `cmd/migrate-v2/main.go`. Idempotent — safe to run multiple times.

**Per file:**

1. Read the legacy TOML file
2. Parse existing fields
3. Transform:
   - Set `schema_version = 1`
   - Set `type = "feedback"`
   - Set `source = ""` (unknown origin for legacy memories)
   - Set `core = false`
   - Nest `behavior`, `impact`, `action` under `[content]`
   - Preserve all tracking counters and metadata
   - Strip `pending_evaluations`
4. Write to `data/memory/feedback/<same-filename>.toml` using atomic write (temp + rename)
5. After ALL files written successfully, rename `data/memories/` to `data/memories.v1-backup/`

**Safety:**

- Do not delete source files until all migrations succeed
- Validate each output file parses correctly before proceeding
- Log each file migrated to stdout
- Exit non-zero on any failure, reporting which files failed
- If `data/memory/feedback/` already contains files, skip files that already exist (idempotent)

### 3.3 Rollback

If migration fails partway:
- Source files in `data/memories/` are untouched
- Partially written files in `data/memory/feedback/` can be deleted and migration re-run
- Backup at `data/memories.v1-backup/` only created on full success

---

## 4. Go Binary Updates

### 4.1 MemoryRecord Struct

Update `internal/memory/record.go`:

```go
type ContentFields struct {
    // Feedback fields
    Behavior string `toml:"behavior,omitempty"`
    Impact   string `toml:"impact,omitempty"`
    Action   string `toml:"action,omitempty"`

    // Fact fields
    Subject   string `toml:"subject,omitempty"`
    Predicate string `toml:"predicate,omitempty"`
    Object    string `toml:"object,omitempty"`
}

type MemoryRecord struct {
    SchemaVersion     int           `toml:"schema_version"`
    Type              string        `toml:"type"`
    Situation         string        `toml:"situation"`
    Source            string        `toml:"source,omitempty"`
    Core              bool          `toml:"core,omitempty"`
    ProjectScoped     bool          `toml:"project_scoped"`
    ProjectSlug       string        `toml:"project_slug,omitempty"`
    Content           ContentFields `toml:"content"`
    SurfacedCount     int           `toml:"surfaced_count"`
    FollowedCount     int           `toml:"followed_count"`
    NotFollowedCount  int           `toml:"not_followed_count"`
    IrrelevantCount   int           `toml:"irrelevant_count"`
    MissedCount       int           `toml:"missed_count"`
    InitialConfidence float64       `toml:"initial_confidence,omitempty"`
    CreatedAt         string        `toml:"created_at"`
    UpdatedAt         string        `toml:"updated_at"`
}
```

### 4.2 Stored Struct

Update `internal/memory/stored.go` (or wherever `Stored` lives):

```go
type Stored struct {
    Type              string  // "feedback" or "fact"
    Situation         string
    Content           ContentFields
    Core              bool
    ProjectScoped     bool
    ProjectSlug       string
    SurfacedCount     int
    FollowedCount     int
    NotFollowedCount  int
    IrrelevantCount   int
    UpdatedAt         time.Time
    FilePath          string
    InitialConfidence float64
}
```

### 4.3 Directory Path Updates

All code that references `data/memories/` must be updated to read from the new paths.

**Path resolution precedence:** If `data/memory/feedback/` exists and is non-empty, read from new paths only (`data/memory/feedback/` + `data/memory/facts/`). If only `data/memories/` exists, read from legacy path. Never read both simultaneously — this prevents double-counting during partial migration states.

This affects:
- `internal/memory/` — file listing, reading, writing
- `internal/surface/` — memory loading for surfacing
- `internal/cli/` — `recall` and `show` commands
- Session-start hook path references (if any)

### 4.4 recall Command

- Reads from both `memory/feedback/` and `memory/facts/` (or legacy `memories/` per precedence rule)
- `=== MEMORIES ===` section now includes both types
- Feedback displayed as before (SBIA format)
- Facts displayed as: `[FACT] <subject> <predicate> <object>` with situation context
- BM25 scoring works on `situation` field for both types (existing behavior extended)

### 4.5 show Command

- Reads from both directories
- Display format extended for facts:
  ```
  Type: fact
  Situation: <situation>
  Subject: <subject>
  Predicate: <predicate>
  Object: <object>
  Source: <source>
  Effectiveness: <ratio>%
  ...
  ```

### 4.6 Backward Compatibility

The Go binary handles legacy files transparently:
- If `type` field is absent, treat as legacy feedback: read `behavior`/`impact`/`action` from top level
- If `type` field is present, read from `[content]` section
- Writing always produces new format

---

## 5. engram-agent Updates

### 5.1 Architecture

The engram-agent remains a single reactive agent that handles both feedback and facts. The unified design spec says to split into two agents only if miscategorization becomes a problem.

### 5.2 Chat Protocol (Minimal Subset)

The engram-agent uses the existing file-comms protocol as-is. This section documents what it uses, not new protocol. Spec 2 handles the rename and new message types.

**Chat file location:** `chat.toml` in the project root (same as current `file-comms` protocol).

> **Migration note:** The unified spec places chat files at `~/.local/share/engram/chat/<project-slug>.toml`. This spec keeps the existing `chat.toml` location for now. Spec 2 will handle the relocation as part of the full `use-engram-chat-as` skill.

**Message format** (unchanged from current file-comms):

```toml
[[message]]
from = "agent-name"
type = "intent"  # | "ack" | "wait" | "info" | "done"
message = "content here"
```

**Message types used by engram-agent:**

| Type | Direction | Usage |
|------|-----------|-------|
| `intent` | reads | Triggers memory matching against situation fields |
| `wait` | writes | Surfaces matching feedback (starts argument) |
| `info` | writes | Surfaces matching facts (informational, no argument) |
| `ack` | writes | Concedes during argument |
| `done` | writes | Argument resolution recording |
| `info` | writes | Heartbeat every 5 minutes |

**Watch loop:** `fswatch -1 chat.toml` background pattern (unchanged from current memory-agent).

**Locking:** `mkdir chat.toml.lock` before append, `rmdir chat.toml.lock` after. Same as current file-comms.

### 5.3 Feedback Behavior (Existing, Updated)

No functional changes to feedback matching and surfacing. Updates:

- Reads from `data/memory/feedback/` instead of `data/memories/`
- Writes new feedback memories to `data/memory/feedback/`
- Uses new format for all writes
- Can still read legacy format files (backward compat during transition)
- Learning triggers unchanged:
  - 1.0 confidence: explicit user corrections
  - 0.7 confidence: correction-like language, observed failures
  - 0.4 confidence: medium-confidence third-party signals
  - 0.2 confidence: inferred patterns

### 5.4 Fact Behavior (New)

#### Fact Surfacing

1. On each `intent` message, after checking feedback matches:
2. Load all fact `situation` fields + `content.subject`/`content.object` into context
3. Match against the intent's situation using subject/object overlap
4. On match, read the full fact file
5. Search for related facts by overlapping subjects/objects
6. Surface as `info` message: `[FACT] <subject> <predicate> <object> (situation: <situation>)`
7. Facts do NOT trigger arguments — they are informational only

#### Fact Learning

The engram-agent extracts facts from conversation messages using LLM judgment guided by the knowledge patterns below. This is inherently a judgment call — the agent is an LLM, not a rule engine.

**Trigger messages:** Only `intent` and `done` messages trigger fact extraction. `info`, `ack`, and `wait` messages are skipped (too noisy, too reactive).

**Confidence levels:**
- 0.7: Clear factual assertions ("we use Redis for caching", "the API returns JSON")
- 0.4: Inferred from context (tool usage patterns, implicit architectural decisions)

**Knowledge patterns** (extraction guidance from the design spec):

| Knowledge | How to encode |
|-----------|--------------|
| Simple fact | One fact triple: `subject → predicate → object` |
| Concept/definition | Multiple facts sharing a subject: `X → is → definition`, `X → contains → Y` |
| Decision | Fact cluster: `X → chose → Y` + `X → rejected → Z` per alternative + `X → because → rationale` |
| Excerpt/quote | `source → says → content` |
| Process/procedure | Ordered facts sharing a subject: `X → step-1 → Y`, `X → step-2 → Z` |

**Negative examples — do NOT extract:**
- Proposals: "we should use Redis" — this is intent, not established fact
- Questions: "does the API use REST?" — unknown, not asserted
- Negations in hypothetical context: "if we didn't use targ..." — counterfactual
- Future plans: "we'll migrate to PostgreSQL" — not yet true
- Opinions without consensus: "I think React is better" — subjective

**Situation field:** Derive from the conversation context where the fact was observed (e.g., "When discussing the engram project's build system").

#### Fact Conflict Resolution

Two-level lookup:

1. **Dedup check** (three-field: `subject + predicate + object`, exact string match): If all three match an existing fact, skip — already known.
2. **Conflict check** (two-field: `subject + predicate`): If subject and predicate match but object differs, apply conflict resolution:
   - If existing fact is `core = true`: do NOT overwrite. Create a new fact with the updated object and lower confidence. Surface both to the user for manual resolution.
   - If existing fact has higher `initial_confidence` than the new fact: do NOT overwrite. Create a new fact. The agent surfaces the conflict as an INFO message.
   - If existing fact has equal or lower confidence: update the object, bump `updated_at`, preserve the higher confidence value.
3. **No match:** Create new file in `data/memory/facts/`.

This means multi-valued predicates (e.g., "engram → depends-on → Redis" and "engram → depends-on → PostgreSQL") trigger conflict resolution. The confidence-based rules produce the correct outcome: if both facts are observed at similar confidence, the second creates a new fact (because it can't overwrite a same-or-higher confidence existing fact). If the user pins one as core, it's protected from automatic updates.

> **Note:** The `learned` message type (Spec 2) will provide a higher-confidence signal (1.0) for fact extraction from agent self-reports. Until then, fact learning operates at 0.7/0.4 confidence from conversation observation only. When Spec 2 ships, `learned` messages become a third fact learning path alongside conversation observation.

### 5.5 Processing Order

Per incoming message:

1. Check feedback triggers first (corrections, failure patterns)
2. Check fact triggers (knowledge extraction from `intent` and `done` messages only)
3. Check surfacing matches (feedback first, then facts)
4. Feedback surfacing → WAIT (argument). Fact surfacing → INFO (no argument).

### 5.6 Tiered Loading

Startup loading strategy:

| Tier | What | When |
|------|------|------|
| **Core** | `core = true` memories (user-pinned + auto-promoted) | Always loaded |
| **Recent** | `updated_at` within last 7 days | Loaded on startup |
| **On-demand** | Everything else | Searched when a core/recent match found |

"Recent" uses a time-based heuristic (7 days) rather than session counting, since memory files have no session ID field and `updated_at` is always available.

**Auto-promotion:** Memories with `followed_count / surfaced_count > 0.7` and `surfaced_count >= 5` auto-promote to `core = true`.

**Auto-demotion:** Auto-promoted core memories (not user-pinned) with `followed_count == 0` and `surfaced_count >= 10` demote (`core = false`). Zero follows required — a memory followed even once in a critical moment is worth keeping.

**Core set cap:** Maximum 20 core memories. When the cap is hit, the oldest auto-promoted memory (by `updated_at`) is demoted to make room. User-pinned memories (`core = true` set by the user) do not count toward the cap.

### 5.7 Locking

Unchanged from current memory-agent:
- Per-file locks (not directory-wide)
- Atomic writes (temp file + rename)
- Stale lock recovery (PID-based, 300s mkdir timeout)
- No multi-file locking

---

## 6. Skill File Changes

### 6.1 Update: `skills/memory-agent/SKILL.md`

Rename to `skills/engram-agent/SKILL.md`. Update content to reflect:
- Two memory types (feedback + facts)
- New data paths (`memory/feedback/`, `memory/facts/`)
- Fact surfacing behavior (INFO messages, no arguments)
- Fact learning behavior (conversation observation, knowledge patterns, deduplication)
- Fact conflict resolution rules
- Tiered loading with core/recent/on-demand
- New file format with `[content]` section

### 6.2 Update: `skills/recall/SKILL.md`

Update directory paths and output format to include facts.

### 6.3 No Change: `skills/file-comms/SKILL.md`

Keep as-is. Spec 2 handles the rename to `use-engram-chat-as`.

---

## 7. Session-Start Hook

Update `hooks/session-start.sh` if it references `data/memories/` path. The hook builds the binary and announces `/recall` — path changes in the binary are handled by the Go code, not the hook. Verify and update if needed.

---

## 8. Acceptance Criteria

1. **Migration script** runs successfully on all ~269 existing memory files
2. **All migrated files** parse as valid new format, type = "feedback"
3. **`engram recall`** returns results from both `memory/feedback/` and `memory/facts/`
4. **`engram show`** displays both feedback and fact memories correctly
5. **engram-agent** surfaces feedback memories on matching intents (existing behavior preserved)
6. **engram-agent** surfaces fact memories as INFO messages on matching intents
7. **engram-agent** learns new facts from `intent` and `done` messages, writes to `memory/facts/`
8. **engram-agent** deduplicates facts by subject + predicate + object
9. **engram-agent** handles fact conflicts per resolution rules (core protection, confidence ordering)
10. **Backward compat** — Go binary reads legacy files from old path if migration hasn't run yet
11. **`targ check-full`** passes with no new lint errors
12. **All existing tests** continue to pass after struct/path changes
13. **Core set** never exceeds 20 auto-promoted memories

---

## 9. Implementation Order

1. Update `MemoryRecord` struct and `ContentFields` (Go)
2. Add legacy backward-compat reading logic
3. Update directory path constants/config with precedence rule
4. Update `recall` and `show` commands
5. Write migration script (`cmd/migrate-v2/`)
6. Run migration on real data, verify
7. Update `skills/engram-agent/SKILL.md` (rename from memory-agent)
8. Update `skills/recall/SKILL.md`
9. Update session-start hook if needed
10. `targ check-full` + manual verification
