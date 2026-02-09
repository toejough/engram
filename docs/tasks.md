# Tasks: ISSUE-160 — Ambient Learning System

**Traces to:** requirements.md, ISSUE-160

## Overview

This document decomposes the ambient learning system into executable implementation tasks with explicit dependencies and acceptance criteria.

---

## Simplicity Rationale

The task breakdown follows the three-phase structure defined in the issue and approved plan:
1. **Extract & Store** (TASK-1 to TASK-3) — Passive capture at session boundaries
2. **Inject & Retrieve** (TASK-4 to TASK-5) — Context injection at startup
3. **Maintain & Promote** (TASK-6 to TASK-8) — Memory hygiene and user-controlled promotion

**Why this is appropriately simple:**
- Each phase delivers independently usable functionality
- Tasks align to single commands/features (no mega-tasks)
- Dependencies are minimal and explicit (DAG structure)
- Cross-cutting tasks (TASK-9, TASK-10) are isolated and can be implemented at any point
- Rule-based extraction (Phase 1) defers LLM complexity to future iterations
- Builds entirely on existing memory infrastructure (ISSUE-152)

**Alternatives considered:**
- **Single-phase delivery:** Rejected — too large, harder to test in isolation
- **LLM-based extraction in Phase 1:** Deferred — rule-based heuristics are simpler, faster, and work offline
- **MCP server approach:** Rejected per issue non-goals — hooks + CLI commands only

---

## Dependency Graph

```
Foundation (Phase 1):
TASK-1 (extract-session command)
    |
    +---- TASK-2 (Stop hook)
    |
    +---- TASK-3 (PreCompact hook)

Context Injection (Phase 2):
TASK-4 (context-inject command)
    |
    +---- TASK-5 (SessionStart hook)

Maintenance (Phase 3):
TASK-6 (consolidate command)
    |
    +---- TASK-7 (promote --review)

TASK-8 (contradiction detection) — independent

Cross-cutting:
TASK-9 (ACT-R scoring) — independent
TASK-10 (hook installation) — depends on TASK-2, TASK-3, TASK-5
```

**Parallel opportunities:**
- TASK-1 and TASK-4 can be implemented concurrently (independent commands)
- TASK-6 and TASK-8 can be implemented concurrently (independent features)
- TASK-9 can be implemented at any point (refactors existing logic)

---

## Phase 1: Extract & Store

### TASK-1: Extract-session command implementation

**Description:** Implement `projctl memory extract-session --transcript <path>` command that reads Claude Code transcript JSONL files and extracts learnings using rule-based heuristics with tiered confidence scoring.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Command `projctl memory extract-session --transcript <path>` exists and is registered in CLI
- [ ] Command reads Claude Code transcript JSONL from provided file path
- [ ] Tier A extraction (confidence 1.0): detects "remember this" phrases, explicit corrections, CLAUDE.md edit events
- [ ] Tier B extraction (confidence 0.7): detects error→fix sequences, repeated pattern occurrences
- [ ] Extracted learnings are stored via existing `memory.Learn()` or `memory.LearnWithConflictCheck()`
- [ ] Internal function `ExtractSession(opts ExtractSessionOpts) (*ExtractSessionResult, error)` implemented in `internal/memory/extract_session.go`
- [ ] Parsing is resilient to transcript format changes with fallback handling
- [ ] Command returns summary: items extracted, confidence distribution, storage status
- [ ] Unit tests for transcript parsing with sample JSONL fixtures
- [ ] Unit tests for confidence tier assignment logic
- [ ] Integration test with real transcript file

**Files:**
- Create: `internal/memory/extract_session.go`
- Create: `cmd/projctl/memory_extract_session.go`
- Modify: `cmd/projctl/main.go` (register command)

**Dependencies:** None

**Traces to:** REQ-1, REQ-2

---

### TASK-2: Stop hook integration

**Description:** Configure Claude Code Stop hook to automatically run extract-session command asynchronously when a session ends, preserving learnings without blocking session exit.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Stop hook configuration documented in project README or settings guide
- [ ] Hook invokes `projctl memory extract-session --transcript $TRANSCRIPT_PATH` asynchronously
- [ ] Hook does not block session termination (runs in background)
- [ ] Extraction failures are logged but do not prevent session exit
- [ ] Hook configuration added to `~/.claude/settings.json` (manual or via install command)
- [ ] Verification test: stop session, confirm extraction runs, confirm session exits immediately
- [ ] Documentation includes hook JSON structure and installation instructions

**Files:**
- Create or modify: Documentation file (README or hooks guide)
- Modify: `~/.claude/settings.json` (via user installation or hooks command)

**Dependencies:** TASK-1

**Traces to:** REQ-3

---

### TASK-3: PreCompact hook integration

**Description:** Configure Claude Code PreCompact hook to automatically run extract-session command asynchronously before context compaction, capturing learnings before they're lost to compression.

**Status:** Ready

**Acceptance Criteria:**
- [ ] PreCompact hook configuration documented in project README or settings guide
- [ ] Hook invokes `projctl memory extract-session --transcript $TRANSCRIPT_PATH` asynchronously
- [ ] Hook does not block context compaction (runs in background)
- [ ] Learnings are extracted before compacted content is lost
- [ ] Extraction failures are logged but do not prevent compaction
- [ ] Hook configuration added to `~/.claude/settings.json` (manual or via install command)
- [ ] Verification test: trigger compaction, confirm extraction runs, confirm compaction proceeds immediately
- [ ] Documentation includes hook JSON structure and installation instructions

**Files:**
- Create or modify: Documentation file (README or hooks guide)
- Modify: `~/.claude/settings.json` (via user installation or hooks command)

**Dependencies:** TASK-1

**Traces to:** REQ-4

---

## Phase 2: Inject & Retrieve

### TASK-4: Context-inject command implementation

**Description:** Implement `projctl memory context-inject` command that queries high-confidence recent memories and formats them as compact markdown suitable for Claude's system prompt.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Command `projctl memory context-inject` exists and is registered in CLI
- [ ] Command queries memory using existing `Query()` infrastructure
- [ ] Only memories with confidence > 0.3 are included (confidence gating)
- [ ] Output is formatted as compact markdown block suitable for system prompt
- [ ] Output is bounded by entry count (max N entries, configurable)
- [ ] Output is bounded by token count (max tokens, configurable)
- [ ] Query prioritizes high-confidence and recently-retrieved memories
- [ ] Internal function `ContextInject(opts ContextInjectOpts) (string, error)` implemented in `internal/memory/context_inject.go`
- [ ] Unit tests for confidence gating, output bounding, markdown formatting
- [ ] Performance test: command completes in < 1s for typical memory database

**Files:**
- Create: `internal/memory/context_inject.go`
- Create: `cmd/projctl/memory_context_inject.go`
- Modify: `cmd/projctl/main.go` (register command)

**Dependencies:** None

**Traces to:** REQ-5

---

### TASK-5: SessionStart hook integration

**Description:** Configure Claude Code SessionStart hook to run context-inject command synchronously at session startup, injecting relevant memories into Claude's initial context.

**Status:** Ready

**Acceptance Criteria:**
- [ ] SessionStart hook configuration documented in project README or settings guide
- [ ] Hook invokes `projctl memory context-inject` synchronously (output included in context)
- [ ] Hook output is included in Claude's initial system context
- [ ] Hook completes within performance budget (< 2 seconds target per REQ-7)
- [ ] Hook configuration added to `~/.claude/settings.json` (manual or via install command)
- [ ] Verification test: start session, confirm context-inject runs, confirm output appears in context
- [ ] Performance test: measure hook execution time, ensure < 2s on typical database
- [ ] Documentation includes hook JSON structure and installation instructions

**Files:**
- Create or modify: Documentation file (README or hooks guide)
- Modify: `~/.claude/settings.json` (via user installation or hooks command)

**Dependencies:** TASK-4

**Traces to:** REQ-6, REQ-7

---

## Phase 3: Maintain & Promote

### TASK-6: Consolidate command implementation

**Description:** Implement `projctl memory consolidate` command that performs periodic memory maintenance by running decay, pruning, deduplication, and surfacing promotion candidates.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Command `projctl memory consolidate` exists and is registered in CLI
- [ ] Command runs existing `Decay()` function on all memories
- [ ] Command runs existing `Prune()` function to remove low-confidence entries
- [ ] Command identifies and merges duplicate memories using semantic similarity
- [ ] Command identifies promotion candidates (high confidence, high retrieval frequency)
- [ ] Command reports summary: entries decayed, entries pruned, duplicates merged, promotion candidates count
- [ ] Command runs unattended (no user interaction required)
- [ ] Internal function `Consolidate(opts ConsolidateOpts) (*ConsolidateResult, error)` implemented in `internal/memory/consolidate.go`
- [ ] Unit tests for decay application, pruning logic, deduplication
- [ ] Integration test with sample memory database

**Files:**
- Create: `internal/memory/consolidate.go`
- Create: `cmd/projctl/memory_consolidate.go`
- Modify: `cmd/projctl/main.go` (register command)

**Dependencies:** None

**Traces to:** REQ-8

---

### TASK-7: Promote --review interactive mode

**Description:** Enhance existing `projctl memory promote` command with `--review` flag that enables interactive mode for user-approved promotion of high-confidence memories to CLAUDE.md.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Command `projctl memory promote --review` enables interactive mode
- [ ] Promotion candidates are shown one at a time with full context
- [ ] User can approve or reject each candidate interactively
- [ ] Approved candidates are appended to `~/.claude/CLAUDE.md` under appropriate section
- [ ] Rejected candidates remain in memory database (not deleted)
- [ ] User is sole gatekeeper — no automatic CLAUDE.md updates without explicit approval
- [ ] Command reports summary: candidates reviewed, approved count, rejected count
- [ ] Unit tests for interactive prompt flow (mocked input)
- [ ] Integration test verifying CLAUDE.md is updated correctly

**Files:**
- Modify: `internal/memory/memory.go` (enhance Promote function)
- Modify: `cmd/projctl/memory_promote.go` (add --review flag and interactive flow)

**Dependencies:** TASK-6

**Traces to:** REQ-9, REQ-13

---

### TASK-8: Contradiction detection on memory write

**Description:** Enhance `LearnWithConflictCheck()` to detect and surface contradictions (semantically overlapping memories with opposing content) at write time, prompting user resolution instead of silently storing both.

**Status:** Ready

**Acceptance Criteria:**
- [ ] `LearnWithConflictCheck()` detects semantic overlap using existing 0.85 similarity threshold
- [ ] Function identifies opposing content via negation patterns and reversed advice
- [ ] Function returns structured conflict info: existing entry text, similarity score, conflict type
- [ ] Contradictions are surfaced to caller (does not silently store both entries)
- [ ] Caller can present conflict to user for resolution
- [ ] Unit tests for negation detection (e.g., "always X" vs "never X")
- [ ] Unit tests for reversed advice detection
- [ ] Integration test with real embeddings and conflict scenarios

**Files:**
- Modify: `internal/memory/memory.go` (enhance LearnWithConflictCheck)
- Modify: `internal/memory/embeddings.go` (if contradiction logic needs embedding support)

**Dependencies:** None

**Traces to:** REQ-10

---

## Cross-Cutting Tasks

### TASK-9: ACT-R activation scoring implementation

**Description:** Replace existing flat decay factor with ACT-R base-level activation model (B_i = ln(Σ t_j^(-d))) to score memory confidence based on recency and retrieval frequency, with type-based retention policies.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Activation formula implemented: B_i = ln(Σ t_j^(-d)) where t_j are retrieval timestamps
- [ ] Each memory retrieval adds a timestamp to the activation sum
- [ ] Frequently and recently retrieved memories maintain high activation
- [ ] Type-based retention policies: corrections = indefinite base, reflections = 30-day sliding window
- [ ] Replaces existing flat decay factor in `Decay()` function
- [ ] Migration path for existing memories (initial activation scores computed from stored metadata)
- [ ] Unit tests for activation calculation with various retrieval patterns
- [ ] Performance test: activation calculation does not degrade query performance

**Files:**
- Modify: `internal/memory/memory.go` (refactor Decay function)
- Modify: `internal/memory/embeddings.go` (update retrieval tracking)

**Dependencies:** None

**Traces to:** REQ-11

---

### TASK-10: Hook installation tooling

**Description:** Implement `projctl memory hooks install` and `projctl memory hooks show` commands to simplify hook configuration, avoiding manual JSON editing.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Command `projctl memory hooks install` writes Stop, PreCompact, and SessionStart hooks to `~/.claude/settings.json`
- [ ] Command `projctl memory hooks show` displays current hook configuration from settings file
- [ ] Install command handles existing hooks gracefully (merges, does not overwrite unrelated hooks)
- [ ] Install command creates settings file if it doesn't exist
- [ ] Show command formats output as readable JSON or TOML
- [ ] Documentation explains manual installation alternative for users who prefer it
- [ ] Unit tests for settings file merging logic
- [ ] Integration test: install hooks, verify settings file content, run show command

**Files:**
- Create: `internal/memory/hooks.go`
- Create: `cmd/projctl/memory_hooks.go`
- Modify: `cmd/projctl/main.go` (register commands)

**Dependencies:** TASK-2, TASK-3, TASK-5

**Traces to:** REQ-12

---

## Implementation Sequence

**Recommended order for maximum value delivery:**

1. **Phase 1 (foundational):** TASK-1 → TASK-2 → TASK-3
   - Delivers passive capture capability immediately
   - Each task can be tested independently

2. **Phase 2 (context injection):** TASK-4 → TASK-5
   - Depends on Phase 1 for useful memory database
   - Can start TASK-4 in parallel with TASK-2/TASK-3

3. **Phase 3 (maintenance):** TASK-6 → TASK-7, TASK-8
   - TASK-7 depends on TASK-6 for candidates
   - TASK-8 is independent and can run in parallel

4. **Cross-cutting (anytime):**
   - TASK-9 can be implemented at any point (refactor only)
   - TASK-10 requires hooks to exist first (TASK-2, TASK-3, TASK-5)

**Parallelization strategy:**
- TASK-1 and TASK-4 can be developed concurrently by different developers
- TASK-6 and TASK-8 can be developed concurrently
- TASK-9 can be developed concurrently with any other task

---

## Summary Statistics

| Metric | Count |
|--------|-------|
| Total tasks | 10 |
| Phase 1 tasks | 3 |
| Phase 2 tasks | 2 |
| Phase 3 tasks | 3 |
| Cross-cutting tasks | 2 |
| Parallel opportunities | 3 pairs |

---

## Coverage Matrix

| Requirement | Tasks |
|-------------|-------|
| REQ-1 (Automatic session extraction) | TASK-1 |
| REQ-2 (Rule-based parsing with tiered confidence) | TASK-1 |
| REQ-3 (Stop hook integration) | TASK-2 |
| REQ-4 (PreCompact hook integration) | TASK-3 |
| REQ-5 (Context injection at session start) | TASK-4 |
| REQ-6 (SessionStart hook integration) | TASK-5 |
| REQ-7 (Performance constraints on startup injection) | TASK-5 |
| REQ-8 (Memory consolidation) | TASK-6 |
| REQ-9 (Interactive promotion workflow) | TASK-7 |
| REQ-10 (Contradiction detection on memory write) | TASK-8 |
| REQ-11 (ACT-R activation scoring) | TASK-9 |
| REQ-12 (Hook installation tooling) | TASK-10 |
| REQ-13 (User control over CLAUDE.md promotion) | TASK-7 |

**Architecture coverage:** All tasks trace back to requirements.md and ISSUE-160.

---

## Success Metrics

Per requirements.md success criteria:

1. **Passive capture works:** Users see memories appearing after sessions without manual commands
2. **Context injection provides value:** Claude references past learnings in new sessions
3. **Promotion workflow is used:** Users run consolidate + promote --review periodically
4. **Performance is acceptable:** SessionStart < 2s, Stop/PreCompact hooks do not block UX
5. **Memory stays healthy:** Consolidation prevents bloat, contradictions are resolved

---

## Traceability

**Upstream artifacts:**
- ISSUE-160: Ambient Learning System issue definition
- requirements.md: REQ-1 through REQ-13

**Downstream validation:**
Run `projctl trace validate` after implementation to ensure complete traceability chain from requirements through tasks to implementation.
