# Layer 0: Foundation - Design

**Project:** layer-0-foundation
**Phase:** Design
**Created:** 2026-02-04

**Traces to:** ISSUE-45

---

## Overview

This design covers the user experience and interfaces for Layer 0 foundation components: memory extraction CLI, centralized yield path generation API, and documentation organization. The design prioritizes both human debugging and machine consumption, with strict validation and clear error reporting.

---

## Design Elements

### DES-001: Memory Extract CLI Interface

Command-line interface for extracting insights from yield/result TOML files.

**Command Structure:**
```bash
projctl memory extract --result <path>
projctl memory extract --yield <path>
```

**Flags:**
- `--result <path>`: Extract from result.toml file (mutually exclusive with --yield)
- `--yield <path>`: Extract from yield.toml file (mutually exclusive with --result)
- Exactly one flag required (error if both or neither provided)

**Output Format:**
TOML format with human-readable status and machine-parseable counts:

```toml
[extract]
status = "success"
file = "/path/to/result.toml"
items_extracted = 5

[[extracted]]
type = "decision"
context = "API design"
content = "Prefer recoverable errors to panics"

[[extracted]]
type = "learning"
content = "UUID ensures uniqueness for parallel execution"
```

**Error Cases:**
```toml
[extract]
status = "error"
file = "/path/to/invalid.toml"
error = "TOML parse failed: invalid syntax at line 15"
items_extracted = 0
```

**Validation:**
- Check file exists before parsing
- Validate TOML schema before extracting
- Report schema violations with line numbers
- Fail fast on invalid structure (strict validation approach)

**Traces to:** REQ-1

---

### DES-002: Memory Extract Success Feedback

Terminal output shown after successful extraction.

**Success Output:**
```
✓ Extracted 5 items from pm-result.toml
  - 3 decisions
  - 2 learnings
Stored in semantic memory (~/.claude/memory/embeddings.db)
```

**Error Output:**
```
✗ Failed to extract from invalid.toml
Error: TOML parse failed at line 15
  Expected string, found integer
```

**Design Rationale:**
- Human-readable summary for debugging
- Machine-parseable TOML for orchestrator consumption
- Both outputs serve their respective audiences

**Traces to:** REQ-1

---

### DES-003: Yield Path Generation API

Internal function for generating unique yield/result file paths with timestamp and UUID-based uniqueness.

**Function Signature:**
```go
func GenerateYieldPath(projectDir, phase, taskID string) string
```

**Path Pattern:**
```
.claude/context/{creation-date}-{project}-{uuid}/{datetime}-{phase}-{taskID}-{uuid}.toml
```

**Example Outputs:**
- Sequential: `.claude/context/2026-02-04-layer-0-abc123/2026-02-04.12-45-30-pm-def456.toml`
- Parallel: `.claude/context/2026-02-04-layer-0-abc123/2026-02-04.12-45-30-pm-TASK-001-def456.toml`

**Date/Time Formats:**
- Creation date: `YYYY-MM-DD` (e.g., `2026-02-04`)
- Datetime: `YYYY-MM-DD.HH-mm-SS` (e.g., `2026-02-04.12-45-30`)
- Ensures filesystem-safe names (no colons in timestamps)

**UUID Usage:**
- Project-level UUID: Unique per project creation
- File-level UUID: Unique per yield file generation
- Prevents conflicts even when same task runs in parallel

**Consumer Contract:**
- Returns absolute path (not relative)
- Creates parent directories if needed
- Fails fast on permission errors
- Idempotent: calling twice returns different paths (intentional)

**Design Rationale:**
- Consumer is the orchestrator (machine) and developers debugging (human)
- Timestamp for human navigation/sorting
- UUID for machine uniqueness guarantee
- Structured hierarchy for project-based organization

**Traces to:** REQ-2

---

### DES-004: Context Write Yield Path Integration

Enhancement to `projctl context write` to include generated yield_path in output.

**Context File Addition:**
```toml
[output]
artifact_path = ".claude/projects/my-project/design.md"
yield_path = "/abs/path/.claude/context/2026-02-04-my-project-abc123/2026-02-04.12-45-30-design-def456.toml"
```

**Skill Usage Pattern:**
1. Skill reads context file
2. Extracts `output.yield_path` field
3. Writes result to absolute path specified
4. Orchestrator reads result from known path

**Error Handling:**
- If `yield_path` generation fails → return error before writing context
- If parent directories can't be created → return error with permission details
- If path is not absolute → panic (programmer error, not user error)

**Traces to:** REQ-2

---

### DES-005: Documentation Organization Strategy

Approach for organizing Layer 0 documentation.

**Enhancement, Not Fragmentation:**
- Enhance existing `docs/commands/memory.md` (or create if missing)
- Enhance existing `docs/commands/trace.md` (or create if missing)
- Enhance existing `docs/commands/context.md` for yield_path pattern
- Add `docs/layer-0-implementation.md` as summary/index

**Avoid Creating:**
- Separate files per subcommand (e.g., memory-extract.md, memory-query.md)
- Duplicated command reference in multiple locations
- New top-level docs that should be in existing sections

**Rationale:**
- User preference: "Enhance existing rather than creating new files"
- Easier to maintain single source of truth
- Simpler navigation for users

**Traces to:** REQ-5, REQ-6, REQ-7

---

### DES-006: Memory Command Documentation Structure

Organization of memory command documentation within single file.

**File:** `docs/commands/memory.md`

**Structure:**
```markdown
# Memory Commands

## Overview
[Semantic memory vs. grep, when to use each]

## Commands

### memory query
[Semantic search examples]

### memory learn
[Store arbitrary insights]

### memory decide
[Store decisions with context]

### memory extract
[Extract from yield/result files]

### memory grep
[Pattern-based search]

### memory session-end
[End-of-session summary]

## Architecture

### Embedding Model
[ONNX e5-small, 384 dimensions]

### Storage
[SQLite-vec location: ~/.claude/memory]

### First Use
[Auto-download behavior, platforms supported]

## Examples
[Real-world usage scenarios]
```

**Traces to:** REQ-6, REQ-8

---

### DES-007: Trace Repair Documentation Structure

Organization of trace repair documentation.

**File:** `docs/commands/trace.md`

**Add Section:**
```markdown
## trace repair

Automatically fixes duplicate IDs and escalates dangling references.

### Auto-Fixed: Duplicate IDs
[Renumbering behavior, examples]

### Escalated: Dangling References
[Manual fix required, escalation file format, examples]

### Examples
[Output for each case, escalation file format]
```

**Traces to:** REQ-5

---

### DES-008: Layer 0 Implementation Summary

High-level summary document for Layer 0 foundation.

**File:** `docs/layer-0-implementation.md`

**Structure:**
```markdown
# Layer 0: Foundation Implementation

## Overview
[What Layer 0 provides before Layer 1]

## Command Inventory

### State Management (Existing)
- state get
- state transition
- state next

### Context Management (Enhanced)
- context write [NEW: yield_path generation]
- context read

### ID Management (Existing)
- id next

### Trace Management (Existing)
- trace validate
- trace repair

### Territory Management (Existing)
- territory map
- territory show

### Memory Management (New + Existing)
- memory query [NEW: semantic search]
- memory learn [NEW: arbitrary insights]
- memory grep [EXISTING]
- memory extract [NEW: from yield/result files]
- memory session-end [EXISTING]

## Architecture
[ONNX + e5-small + SQLite-vec]

## Key Patterns
[Yield path generation, auto-download, parallel safety]

## Documentation
[Links to memory.md, trace.md, context.md]

## References
[Link to orchestration-system.md Section 13.3]
```

**Traces to:** REQ-7

---

### DES-009: Error Message Design

Consistent error message format across Layer 0 commands.

**Format:**
```
✗ {Command} failed: {brief reason}
Error: {detailed technical message}
  {additional context}
  {suggestion for fix if applicable}
```

**Examples:**

**Schema Validation:**
```
✗ memory extract failed: invalid TOML structure
Error: missing required field [payload.decisions]
  File: pm-result.toml
  Expected: array of decision objects
```

**Path Generation:**
```
✗ context write failed: cannot create yield path
Error: permission denied
  Directory: .claude/context/2026-02-04-project-abc/
  Suggestion: check directory permissions or run with appropriate access
```

**Rationale:**
- Strict validation approach requires clear error reporting
- Human-readable for debugging
- Structured for automated parsing if needed

**Traces to:** REQ-1, REQ-2

---

### DES-010: Auto-Download User Experience

First-use experience when ONNX model needs to be downloaded.

**First Run:**
```bash
$ projctl memory query "error handling patterns"

⬇ Downloading e5-small model (~130MB)...
  [████████████████████░░░░] 75% (98MB/130MB) 2.5MB/s

✓ Model downloaded to ~/.claude/models/e5-small
✓ Initializing ONNX runtime...

Results (2 found):
1. [2026-02-03] Decision: Prefer recoverable errors to panics
   ...
```

**Subsequent Runs:**
```bash
$ projctl memory query "error handling patterns"

Results (2 found):
1. [2026-02-03] Decision: Prefer recoverable errors to panics
   ...
```

**Rationale:**
- User shouldn't be surprised by downloads
- Progress indicator for large file download
- Clear feedback that download is one-time only

**Traces to:** REQ-8

---

## Design Decisions

### Decision 1: TOML Output Format

**Context:** Memory extract needs output format that serves both human debugging and machine consumption.

**Choice:** TOML format with structured sections for status, counts, and extracted items.

**Reason:** User specified "mix of human/machine readable" and "TOML satisfies both." Format is consistent with other projctl outputs (context files, yield files).

**Alternatives Considered:**
- JSON (too verbose for human reading)
- Plain text (not machine-parseable)
- Two separate outputs (unnecessary complexity)

---

### Decision 2: Strict Schema Validation

**Context:** Memory extract must handle malformed yield/result files.

**Choice:** Validate schema before extracting, fail fast with detailed error messages.

**Reason:** User specified "check schema before extracting." Prevents silent failures and data corruption.

**Alternatives Considered:**
- Best-effort extraction (could miss required fields)
- Lenient parsing (could extract invalid data)

---

### Decision 3: Enhance Existing Documentation

**Context:** Need to document new memory commands and trace repair.

**Choice:** Add to existing docs/commands/*.md files rather than creating new files.

**Reason:** User specified "add to existing README/docs rather than creating new files."

**Alternatives Considered:**
- One file per command (rejected: too fragmented)
- Single mega-doc (rejected: hard to navigate)
- Current approach: one file per command group (accepted)

---

### Decision 4: Timestamp in Yield Path

**Context:** Yield paths need uniqueness and human navigability.

**Choice:** Include both date (YYYY-MM-DD) and datetime (YYYY-MM-DD.HH-mm-SS) with UUID.

**Reason:** User specified exact format. Timestamp provides chronological ordering for humans, UUID provides uniqueness for machines.

**Alternatives Considered:**
- UUID only (rejected: hard for humans to navigate)
- Timestamp only (rejected: not unique for parallel execution)
- Sequential counter (rejected: requires coordination/locking)

---

### Decision 5: Absolute Paths in Yield Path

**Context:** Skills need to write results to orchestrator-specified locations.

**Choice:** `output.yield_path` is absolute path, not relative.

**Reason:** User specified "absolute path (not relative)." Avoids working directory ambiguity when skills spawn subprocesses.

**Alternatives Considered:**
- Relative paths (rejected: ambiguous with different working directories)
- Both formats supported (rejected: unnecessary complexity)

---

## Out of Scope

The following are explicitly not covered by this design:

- Implementation details (Go code structure, package layout)
- Test strategies (covered by REQ-3, REQ-4, REQ-9 acceptance criteria)
- Performance optimization (semantic query speed)
- Windows platform support (documented as future work)
- GUI/TUI interfaces (CLI only)
- Migration tools for existing memory data

---

## References

- **Specification:** docs/orchestration-system.md Section 13.3 Layer 0: Foundation
- **Requirements:** .claude/projects/layer-0-foundation/requirements.md
- **Issue:** ISSUE-45
