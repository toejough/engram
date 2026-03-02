# Design

Interaction model and UX specification for the engram memory system. Each DES-N item traces to UC (not REQ — they are peers at L2).

---

## Horizontal Primitives

These four primitives unify the interaction model across all UCs. Every DES item below uses them.

### P1: System Reminder Format

All agent-visible communication uses `<system-reminder source="engram">` tags. The `[engram]` prefix identifies the source within the reminder. Two content types share this channel:

- **Memory surfacing** (UC-2): `[engram] N memories for this context:` followed by numbered entries
- **Correction feedback** (UC-3): `[engram] Correction captured.` followed by action details

### P2: Hook Output Contract

Every hook invocation follows: hook shell script → Go binary → stdout. The Go binary does all work. Stdout contains system reminder text (surfaced to agent) or is empty (nothing to surface). Exit 0 always — hook failures are logged, not surfaced.

### P3: Feedback Channel

The system communicates with the agent exclusively via system reminders. No side-channel, no tool output, no file-based signaling.

- UC-2: memories appear as system reminders
- UC-3: correction acknowledgment appears as a system reminder in the same hook response
- UC-1: no agent-visible output (background, post-session)

### P4: Audit Contract

All memory operations produce structured log entries written by the Go binary to a log file (not stdout — stdout is reserved for system reminders). Format: key-value structured log, one line per event. Fields: timestamp, operation, action, memory_id (when applicable), and action-specific fields.

---

## DES-1: Memory Surfacing Reminder Format

**Traces to:** UC-2

When memories are surfaced at hook time, the agent sees:

```
<system-reminder source="engram">
[engram] 3 memories for this context:

1. Always use `git add <specific-files>` not `git add -A` (A, high)
   Staging with -A has included .env files and large binaries in past sessions.

2. This project uses `targ` build system, not `make` (A, medium)
   Build commands: `targ test`, `targ lint`, `targ build`.

3. DI pattern: no direct I/O in internal/ packages (B, new)
   All file/network/DB access through injected interfaces.
</system-reminder>
```

**Format rules:**

- Header: `[engram] N memories for this context:` (or `1 memory` singular)
- Each entry: numbered, title on first line, parenthetical `(confidence, impact)`
- Body: 1-2 lines of actionable guidance, indented
- Confidence: `A` (user stated), `B` (agent inferred, user could have corrected), `C` (agent inferred post-session)
- Impact: `high` (top quartile), `medium`, `low`, `new` (insufficient evaluation data)
- No scores, no math — just the labels. Diagnostic detail available via commands, not surfacing.

**PreToolUse single-line variant:**

PreToolUse is latency-critical and surfaces at most 1 memory. It uses a compact single-line format:

```
<system-reminder source="engram">
[engram] Use `targ test` not `go test` in this project (A, high)
</system-reminder>
```

No numbering, no body. Minimizes context overhead on the hottest path.

**Empty result:**

No output at all. The hook produces empty stdout. No system reminder appears. No "no memories found" message.

---

## DES-2: Per-Hook Surfacing Scenarios

**Traces to:** UC-2

### SessionStart — broadest context, most memories

Trigger: session begins. Query: project context (directory name, project CLAUDE.md, README). Budget: K=5 (default).

```
$ claude
> Starting session...

<system-reminder source="engram">
[engram] 3 memories for this context:

1. This project uses `targ` build system, not `make` (A, high)
   Build commands: `targ test`, `targ lint`, `targ build`.

2. Always use `git add <specific-files>` not `git add -A` (A, high)
   Staging with -A has included .env files and large binaries in past sessions.

3. CLAUDE.md trailer is `AI-Used: [claude]` not Co-Authored-By (A, medium)
   This applies to all commits in this project.
</system-reminder>
```

### UserPromptSubmit — task-scoped, fewer memories

Trigger: user sends a message. Query: the user's message text. Budget: K=3 (default).

```
user: "Add a caching layer to the retrieval pipeline"

<system-reminder source="engram">
[engram] 1 memory for this context:

1. DI pattern: no direct I/O in internal/ packages (B, new)
   All file/network/DB access through injected interfaces. Wire at the edges.
</system-reminder>
```

### PreToolUse — latency-critical, single memory

Trigger: agent is about to use a tool. Query: tool name + arguments. Budget: K=1 (default).

```
agent about to run: Bash("go test ./...")

<system-reminder source="engram">
[engram] Use `targ test` not `go test` in this project (A, high)
</system-reminder>
```

### Empty result — no memories match

No system reminder injected. Silent. The agent's context is not polluted with "nothing found" messages.

---

## DES-3: Correction Feedback Reminder Format

**Traces to:** UC-3

When an inline correction is detected and reconciled, the agent sees one of two formats:

### Existing memory enriched (overlap found)

```
user: "no, don't use git add -A — use specific files"

<system-reminder source="engram">
[engram] Correction captured.
  Enriched: "Always use `git add <specific-files>` not `git add -A`"
  Added context: prohibition reinforced with explicit negative example
  Keywords: [git, staging, add, specific-files, no-git-add-A]
</system-reminder>
```

### New memory created (no overlap)

```
user: "wait, this project uses bun not npm"

<system-reminder source="engram">
[engram] Correction captured.
  Created: "This project uses bun, not npm"
  Keywords: [bun, npm, package-manager, install, run]
</system-reminder>
```

**Format rules:**

- Header: `[engram] Correction captured.`
- Action: `Enriched:` or `Created:` with the memory title in quotes
- Context line (enriched only): what was added/changed
- Keywords: the retrieval terms added, in brackets
- Concise — appears in the same hook response as the user's message

---

## DES-4: Correction Detection and Surfacing Coexistence

**Traces to:** UC-2, UC-3

UserPromptSubmit does dual duty: surfacing memories AND checking for corrections. When both produce output, correction feedback appears first (more urgent — the agent just made a mistake), followed by surfaced memories:

```
user: "no, use targ test not go test"

<system-reminder source="engram">
[engram] Correction captured.
  Enriched: "This project uses `targ` build system, not `make`"
  Added context: `targ test` explicitly, not `go test`
  Keywords: [targ, test, go-test, build-system]

[engram] 1 memory for this context:

1. DI pattern: no direct I/O in internal/ packages (B, new)
   All file/network/DB access through injected interfaces.
</system-reminder>
```

Both sections are in the same `<system-reminder>` block. Correction first, then surfacing. The surfaced memories are retrieved against the user's full message (including the correction), so they reflect the current context — not the mistake.

---

## DES-5: False Positive Handling

**Traces to:** UC-3

The correction pattern corpus may match non-corrections (e.g., "remember to run tests" as a general instruction, not a re-teaching). The system captures it anyway.

**Rationale:** False positive cost is low — one extra memory that decays via frecency if never relevant in future sessions. The alternative (confirmation prompts, LLM classification at hook time) adds latency and complexity to a path that must be fast. The agent sees the correction-captured reminder and continues normally.

**No confirmation prompt.** No "did you mean to correct me?" The system captures, the agent continues. If the resulting memory is noise, it naturally decays: it gets surfaced in contexts where it's irrelevant, receives `irrelevant` evaluations, its impact stays low, its frecency drops, and it falls out of future surfacing.

**Edge case — pattern match on quoted text:** If the user's message contains a correction pattern inside quoted code or prose (e.g., `"the error message says 'that's not valid'"`), the corpus still matches. Same rationale — low-cost false positive, natural decay handles it.

---

## DES-6: Session-End Extraction Flow

**Traces to:** UC-1

UC-1 is entirely background — the agent is gone, the user sees nothing. The interaction is between the Stop hook and the Go binary.

### Flow

```
Stop hook fires
  → shell script invokes: engram extract --session <transcript-path>
  → Go binary:
      1. Read transcript
      2. LLM enrichment (sonnet): extract learnings with structured metadata
         - observation_type, concepts, principle, anti_pattern, rationale, enriched_content
         - LLM-generated retrieval keywords
      3. Quality gate: reject vague/mechanical patterns
         - "Always check things carefully" → rejected (not actionable)
         - "Use targ test instead of go test in this project" → accepted
      4. Confidence tier assignment (haiku): A/B/C per learning
      5. For each surviving learning:
         a. Local similarity retrieval (top-3 candidates)
         b. Haiku overlap gate per candidate
         c. Overlap → enrich existing memory
         d. No overlap → create new memory
      6. Deduplicate against mid-session corrections (UC-3)
         - Skip learnings that match memories already created/enriched this session
      7. Write audit log entries
  → exit 0
```

No system reminder output (agent is gone). All output goes to audit log. The user sees nothing — extraction is invisible unless they run diagnostic commands.

### Edge cases

- **Empty session:** No learnings extracted. Audit log records: `extract completed count=0 reason=empty-session`.
- **All learnings rejected by quality gate:** Audit log records each rejection with reason.
- **All learnings deduplicated against mid-session corrections:** Audit log records: `extract skipped memory_id=<id> reason=mid-session-duplicate`.

---

## DES-7: Audit Log Format

**Traces to:** UC-1, UC-3

Every memory operation produces a structured log entry. The log is the system's memory of its own decisions — essential for debugging retrieval quality and diagnosing leech memories.

### Format

Key-value structured log, one line per event:

```
2026-02-27T16:30:00Z extract created memory_id=m_7f3a title="Use targ build system" confidence=B quality_score=0.85
2026-02-27T16:30:00Z extract enriched memory_id=m_2b1c title="DI pattern in internal/" confidence=A overlap_score=0.72
2026-02-27T16:30:01Z extract rejected reason=vague content="Always check things carefully"
2026-02-27T16:30:01Z extract skipped memory_id=m_2b1c reason=mid-session-duplicate
2026-02-27T16:30:02Z correct created memory_id=m_9e2f title="Use bun not npm" confidence=A pattern="^wait"
2026-02-27T16:30:02Z correct enriched memory_id=m_2b1c title="DI pattern in internal/" confidence=A pattern="\\bremember\\s+(that|to)" overlap_score=0.68
2026-02-27T16:30:03Z surface returned hook=SessionStart count=3 query_tokens=45 latency_ms=12
2026-02-27T16:30:03Z surface returned hook=PreToolUse count=1 query_tokens=8 latency_ms=3
2026-02-27T16:30:04Z reclass decreased memory_id=m_4a1b reason=correction-implicated old_impact=0.7 new_impact=0.3
```

### Fields

- **Timestamp:** RFC 3339 UTC
- **Operation:** `extract` (UC-1), `correct` (UC-3), `surface` (UC-2), `reclass` (UC-3 reclassification)
- **Action:** `created`, `enriched`, `rejected`, `skipped`, `returned`, `decreased`
- **Common fields:** `memory_id`, `title`, `confidence`
- **Operation-specific fields:** `quality_score`, `overlap_score`, `reason`, `pattern`, `hook`, `count`, `query_tokens`, `latency_ms`, `old_impact`, `new_impact`

### Storage

Log file at `<plugin-data-dir>/audit.log`. Append-only. Rotation policy is an architecture decision (not specified here).

---

## DES-8: Session-End Correction Catch-Up Flow

**Traces to:** UC-3

The Stop hook runs two distinct operations: session-end extraction (DES-6, UC-1) and correction catch-up (this item, UC-3). Catch-up finds corrections the inline pattern corpus missed.

### Flow

```
Stop hook fires (after DES-6 extraction completes)
  → shell script invokes: engram catchup --session <transcript-path>
  → Go binary:
      1. Read transcript
      2. LLM evaluation (haiku): identify user corrections not captured mid-session
         - Compare transcript corrections against mid-session correction log
         - Flag corrections that lack a corresponding mid-session capture
      3. For each missed correction:
         a. Extract correction content + context
         b. Reconcile against existing memories (same as REQ-14)
         c. Create or enrich memory
      4. Pattern corpus update: extract correction phrases from missed corrections
         - Add new patterns to the persisted corpus
         - e.g., "you didn't shut them down" → `\byou didn't\b` candidate
         - New patterns are candidates — validated by occurrence in future sessions
      5. Write audit log entries
  → exit 0
```

### Audit log entries

```
2026-02-27T16:30:05Z catchup found count=1 missed="you didn't shut them down"
2026-02-27T16:30:05Z catchup created memory_id=m_b3e7 title="Always shut down teammates before ending session" confidence=A
2026-02-27T16:30:06Z catchup corpus_added pattern="\\byou didn't\\b" source="you didn't shut them down"
```

### Edge cases

- **No missed corrections:** Audit log records: `catchup completed count=0`. Common case — the inline corpus catches ~85%.
- **Missed correction overlaps with extraction:** Reconciliation (REQ-14) handles it — the extraction may have already created a memory from the same transcript content. Haiku overlap gate prevents duplication.
