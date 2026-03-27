# Stale Spec Cleanup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Update 5 spec files to remove stale SessionStart surfacing references and delete 8 tombstone sections.

**Architecture:** Pure documentation edits across `docs/specs/{use-cases,architecture,requirements,design,tests}.toml`. No code changes. Each task is one file, committed independently.

**Tech Stack:** TOML spec files, git

**Spec:** `docs/superpowers/specs/2026-03-25-stale-spec-cleanup-design.md`

---

### Task 1: Update use-cases.toml

**Files:**
- Modify: `docs/specs/use-cases.toml`

- [ ] **Step 1: Update UC-1 creation visibility bullet (line 54)**

Replace:
```
- **Creation visibility (deferred):** PreCompact and Stop/SessionEnd hooks have no output mechanism to show the user what was created. Instead, creation events are logged to a file (`<data-dir>/creation-log.jsonl`) with timestamp, title, tier, and file path. UC-2's SessionStart surfacing reports these at the start of the next session so the user sees what was learned. The log is cleared after successful reporting.
```

With:
```
- **Creation visibility (deferred):** PreCompact and Stop/SessionEnd hooks have no output mechanism to show the user what was created. Instead, creation events are logged to a file (`<data-dir>/creation-log.jsonl`) with timestamp, title, tier, and file path. The creation log is written but not yet surfaced — SessionStart no longer reads it (session-recall redesign, 2026-03-21).
```

- [ ] **Step 2: Update UC-2 starting state (line 267)**

Replace:
```
**Starting state:** The memory store contains TOML files written by UC-3. A hook fires (SessionStart, UserPromptSubmit, or PreToolUse).
```

With:
```
**Starting state:** The memory store contains TOML files written by UC-3. A hook fires (UserPromptSubmit or PreToolUse).
```

- [ ] **Step 3: Update UC-2 end state (line 269)**

Replace:
```
**End state:** Relevant memories are surfaced as system reminders at all three hook points (SessionStart, UserPromptSubmit, PreToolUse). The agent uses these advisories with full session context to exercise judgment. Each surfacing event is recorded in the memory's TOML file (count, timestamp, context type) for effectiveness measurement.
```

With:
```
**End state:** Relevant memories are surfaced as system reminders at two hook points (UserPromptSubmit, PreToolUse). SessionStart runs `engram maintain` and emits a `/recall` hint but does not surface memories (session-recall redesign, 2026-03-21). Each surfacing event is recorded in the memory's TOML file (count, timestamp) for effectiveness measurement.
```

- [ ] **Step 4: Replace UC-2 SessionStart bullet (line 275)**

Replace:
```
- **SessionStart — passive surfacing:** Surface the top 20 memories by recency as a system reminder. No matching needed — recency is the only signal. Provides context priming at session start. The reminder lists each surfaced memory's title and file path so the user can inspect or edit them. Additionally, if a creation log exists (`<data-dir>/creation-log.jsonl`), report the memories created during prior sessions (by UC-1 at PreCompact/SessionEnd) in the `systemMessage` so the user sees what was learned. Clear the log after reporting.
```

With:
```
- **SessionStart — maintain + recall hint:** Runs `engram maintain` for background triage and emits a static `/recall` reminder. No memory surfacing at session start — context loading is on-demand via the `/recall` skill (session-recall redesign, 2026-03-21).
```

- [ ] **Step 5: Update surfacing_contexts (line 290)**

Replace:
```
  - `surfacing_contexts` (string array) — bounded list of recent context types (`session-start`, `prompt`, `tool`), capped at 10 entries
```

With:
```
  - `surfacing_contexts` (string array) — bounded list of recent context types (`prompt`, `tool`), capped at 10 entries
```

- [ ] **Step 6: Delete UC-4 tombstone (lines 894–929)**

Delete the entire `[UC-4]` section from `[UC-4]` through the closing `'''`.

- [ ] **Step 7: Delete UC-5 tombstone (lines 931–967)**

Delete the entire `[UC-5]` section from `[UC-5]` through the closing `'''`.

- [ ] **Step 8: Delete UC-22 tombstone (lines 355–365)**

Delete the entire `[UC-22]` section from `[UC-22]` through the closing `'''` and trailing blank line.

- [ ] **Step 9: Commit**

```bash
git add docs/specs/use-cases.toml
git commit -m "docs: update UC-2 SessionStart, remove UC-4/5/22 tombstones (#377)"
```

### Task 2: Update architecture.toml

**Files:**
- Modify: `docs/specs/architecture.toml`

- [ ] **Step 1: Update ARCH-12 mode routing (line 65–69)**

Replace:
```
engram surface --mode <session-start|prompt|tool> --data-dir <path> [--format json] [mode-specific flags]
```

With:
```
engram surface --mode <prompt|tool> --data-dir <path> [--format json] [mode-specific flags]
```

Replace:
```
- `--mode session-start`: Read creation log (ARCH-21 LogReader.ReadAndClear) → emit creation report if entries exist. Then call MemoryRetriever.ListMemories, sort by UpdatedAt desc, take top 20, emit DES-5 format. Both sections combined in output.
```

With:
```
- `--mode session-start`: *(removed — session-recall redesign, 2026-03-21)* SessionStart hook now runs `engram maintain` and emits a `/recall` hint. No `engram surface` call.
```

- [ ] **Step 2: Update ARCH-13 SessionStart hook flow (line 97)**

Replace:
```
- **SessionStart:** After build step, call `engram surface --mode session-start --format json`. Reshape into `{systemMessage, additionalContext}`. Creation report (from creation log, if any) is included in `systemMessage`.
```

With:
```
- **SessionStart:** After build step, run `engram maintain` (background triage). Emit static `/recall` hint in `systemMessage`. No `engram surface` call (session-recall redesign, 2026-03-21).
```

- [ ] **Step 3: Update ARCH-21 reader section (lines 431–447)**

Replace:
```
**Reader (used by SessionStart surfacing):**

```go
type LogReader struct {
    readFile   func(string) ([]byte, error)
    removeFile func(string) error
}

func (r *LogReader) ReadAndClear(dataDir string) ([]LogEntry, error)
```

Implementation:
1. Read `<data-dir>/creation-log.jsonl`.
2. Parse each line as JSON → `LogEntry`.
3. Delete the file after successful read.
4. Return entries (or empty slice if file missing).
5. Read/delete errors logged to stderr, non-fatal.
```

With:
```
**Reader (deferred — no caller since session-recall redesign, 2026-03-21):**

The `LogReader` and `ReadAndClear` method exist in `internal/creationlog/` but have no production caller. SessionStart no longer surfaces memories or reads the creation log. The reader is retained for potential future use.
```

- [ ] **Step 4: Update ARCH-21 traces (line 451)**

Replace:
```
**Traces to:** REQ-23 (creation log format), REQ-24 (read and clear at SessionStart), REQ-25 (write during learn)
```

With:
```
**Traces to:** REQ-23 (creation log format), REQ-24 (read and clear — deferred), REQ-25 (write during learn)
```

- [ ] **Step 5: Delete ARCH-51 tombstone (lines 1353–1362)**

Delete the entire `[ARCH-51]` section. Also delete the traceability table header that follows if it references only removed items.

- [ ] **Step 6: Delete ARCH-70 tombstone (lines 1718–1728)**

Delete the entire `[ARCH-70]` section from `[ARCH-70]` through the closing `'''`.

- [ ] **Step 7: Commit**

```bash
git add docs/specs/architecture.toml
git commit -m "docs: update ARCH-12/13/21 SessionStart, remove ARCH-51/70 tombstones (#377)"
```

### Task 3: Update requirements.toml

**Files:**
- Modify: `docs/specs/requirements.toml`

- [ ] **Step 1: Mark REQ-9 as removed (lines 1384–1395)**

Replace the `[REQ-9]` body with:
```
[REQ-9]
name = "SessionStart surfacing — top 20 by frecency *(removed — session-recall redesign, 2026-03-21)*"
traces_to = ["UC-2"]
hash = "87b4bf47"
body = '''
**Status:** Removed. SessionStart no longer surfaces memories. The hook runs `engram maintain` and emits a `/recall` hint instead.

- Traces to: UC-2 (SessionStart — removed)

---
'''
```

- [ ] **Step 2: Mark REQ-24 as removed (lines 428–451)**

Replace the `[REQ-24]` body with:
```
[REQ-24]
name = "SessionStart creation report — read and clear creation log *(removed — session-recall redesign, 2026-03-21)*"
traces_to = ["UC-2"]
hash = "3be9fdb1"
body = '''
**Status:** Removed. SessionStart no longer surfaces memories or reads the creation log. The creation log is still written (REQ-23, REQ-25) but has no reader in the current hook flow.

- Traces to: UC-2 (SessionStart creation visibility — removed)

---
'''
```

- [ ] **Step 3: Delete REQ-83 tombstone (lines 1288–1298)**

Delete the entire `[REQ-83]` section.

- [ ] **Step 4: Delete REQ-90 and REQ-91 tombstones (lines 1398–1420)**

Delete both `[REQ-90]` and `[REQ-91]` sections.

- [ ] **Step 5: Commit**

```bash
git add docs/specs/requirements.toml
git commit -m "docs: mark REQ-9/24 removed, delete REQ-83/90/91 tombstones (#377)"
```

### Task 4: Update design.toml

**Files:**
- Modify: `docs/specs/design.toml`

- [ ] **Step 1: Update DES-5 SessionStart format (lines 381–420)**

Replace the `[DES-5]` body with:
```
[DES-5]
name = "SessionStart surfacing reminder format *(removed — session-recall redesign, 2026-03-21)*"
traces_to = ["UC-2"]
hash = "49edc0cd"
body = '''
**Status:** Removed. SessionStart no longer surfaces memories or reads the creation log. The hook emits a static `/recall` hint and runs `engram maintain`.

The DES-5 format (creation report + recency surfacing in system-reminder XML) is no longer produced.

- Traces to: UC-2 (SessionStart — removed)

---
'''
```

- [ ] **Step 2: Commit**

```bash
git add docs/specs/design.toml
git commit -m "docs: mark DES-5 SessionStart format as removed (#377)"
```

### Task 5: Update tests.toml

**Files:**
- Modify: `docs/specs/tests.toml`

These tests reference SessionStart surfacing behavior that no longer exists. Mark each as removed with a one-line status note. Keep the test ID to avoid renumbering.

- [ ] **Step 1: Mark SessionStart surfacing tests as removed**

Mark the following tests as removed (replace body with `**Status:** Removed. SessionStart no longer surfaces memories (session-recall redesign, 2026-03-21).`):

| Test | Line | Name |
|------|------|------|
| T-27 | 2365 | SessionStart surfaces top 20 by recency |
| T-28 | 2473 | SessionStart with fewer than 20 memories surfaces all |
| T-40 | 3520 | Mode session-start routes to SessionStart surfacing |
| T-92 | 4309 | SessionStart includes creation report before recency surfacing |
| T-93 | 4321 | SessionStart with no creation log produces recency-only output |
| T-94 | 4333 | SessionStart with creation log but no memories produces creation-only output |
| T-99 | 4403 | SessionStart hook puts creation report in systemMessage |
| T-169 | 978 | SessionStart uses pure frecency ranking |
| T-283 | 2497 | Session-start mode triggers auto-registration |
| T-284 | 2511 | Non-session-start modes skip auto-registration |

- [ ] **Step 2: Update T-165 frecency test to remove session-start context (line 935)**

Replace:
```
**Given** a memory with SurfacedCount=10, LastSurfaced=2h ago, SurfacingContexts=["session-start","prompt","tool"], and effectiveness score of 80%,
```

With:
```
**Given** a memory with SurfacedCount=10, LastSurfaced=2h ago, SurfacingContexts=["prompt","tool"], and effectiveness score of 80%,
```

- [ ] **Step 3: Update T-77 recorder test to remove session-start mode (line 4088)**

Replace:
```
**When** `RecordSurfacing` is called with mode `"session-start"`,
**Then** the file has `surfaced_count = 4`, `last_surfaced` updated to current time, and `surfacing_contexts = ["prompt", "tool", "prompt", "session-start"]`.
```

With:
```
**When** `RecordSurfacing` is called with mode `"prompt"`,
**Then** the file has `surfaced_count = 4`, `last_surfaced` updated to current time, and `surfacing_contexts = ["prompt", "tool", "prompt", "prompt"]`.
```

- [ ] **Step 4: Remove "## SessionStart Surfacing" heading (line 2202)**

Delete the line:
```
## SessionStart Surfacing (ARCH-9, ARCH-12)
```

- [ ] **Step 5: Update T-193 budget config test to remove SessionStart reference (line 1371)**

Replace:
```
**Given** a config file with custom budget values (e.g., SessionStart = 1000),
```

With:
```
**Given** a config file with custom budget values (e.g., Prompt = 1000),
```

- [ ] **Step 6: Update T-195 budget warning test to remove SessionStart reference (lines 1399–1401)**

Replace:
```
**Given** surfacing logs showing 6 out of 10 SessionStart invocations exceed budget cap,
**When** `engram review` computes cap hit rate,
**Then** the warning is raised: "⚠ Hitting cap on 60% of invocations" for SessionStart hook.
```

With:
```
**Given** surfacing logs showing 6 out of 10 Prompt invocations exceed budget cap,
**When** `engram review` computes cap hit rate,
**Then** the warning is raised: "⚠ Hitting cap on 60% of invocations" for Prompt hook.
```

- [ ] **Step 7: Commit**

```bash
git add docs/specs/tests.toml
git commit -m "docs: mark SessionStart test specs as removed, update references (#377)"
```

### Task 6: Close issue

- [ ] **Step 1: Close GitHub issue #377**

```bash
gh issue close 377 --comment "Fixed: updated UC-2, ARCH-12/13/21, REQ-9/24, DES-5 to reflect SessionStart removal. Marked 10 test specs as removed. Deleted 8 tombstone sections (UC-4/5/22, ARCH-51/70, REQ-83/90/91)."
```
