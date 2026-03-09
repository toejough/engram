# Test List

Behavioral tests derived from architecture items. Each T-* item specifies a test case in Given/When/Then format.

---

## UC-23: Unified Instruction Registry Tests

### T-182: Registry JSONL bounded growth — one line per instruction

**Traces to:** ARCH-52

Given: A registry with 100 registered instructions, each with different source types
When: I read instruction-registry.jsonl
Then: File contains exactly 100 lines, one per instruction ID

---

### T-183: Registry ID format determinism

**Traces to:** ARCH-53

Given: A memory instruction "always-use-targ-reminder.toml" and a CLAUDE.md instruction "use-targ"
When: I register both instructions
Then: Memory ID is "memory:always-use-targ-reminder.toml", CLAUDE.md ID is "claude-md:CLAUDE.md:use-targ"

---

### T-184: Effectiveness ratio computation

**Traces to:** ARCH-54

Given: An instruction with followed=20, contradicted=5, ignored=3
When: I read the instruction's effectiveness signal
Then: Effectiveness is 20/28 = 0.714, not stored in registry

---

### T-185: Frecency blend with decay

**Traces to:** ARCH-54

Given: Instruction A surfaced 100 times, last surfaced 1 day ago; Instruction B surfaced 50 times, last surfaced 7 days ago
When: I compute frecency for both with 7-day half-life
Then: Instruction A has higher frecency despite fewer total surfacings

---

### T-186: Content hash change detection

**Traces to:** ARCH-55

Given: A memory instruction with content_hash="abc123"
When: The memory TOML is edited and re-registered
Then: New content_hash differs from "abc123"

---

### T-187: Absorbed history preserves counters

**Traces to:** ARCH-56

Given: Instruction X (surfaced_count=500, followed=30, contradicted=20) and Instruction Y
When: I merge X → Y
Then: Y's absorbed array contains one entry from X with all counters intact

---

### T-188: Idempotent merge

**Traces to:** ARCH-56

Given: I have merged X → Y once, creating one absorbed entry
When: I merge X → Y again with same instructions
Then: No duplicate absorbed entry is created, operation is idempotent

---

### T-189: Concurrent writes safety — read-all write-full

**Traces to:** ARCH-57

Given: Two concurrent hook calls to RecordSurfacing for different instructions
When: Both calls write to the registry
Then: Both updates persist, no data corruption (acceptable: one update may not include the other's frequency delta)

---

### T-190: Backfill aggregation — surfacing log

**Traces to:** ARCH-58

Given: surfacing-log.jsonl with 10 surface events for memory X, last at timestamp T
When: I run engram registry init
Then: Registry entry for X has surfaced_count=10, last_surfaced=T

---

### T-191: Backfill retirement mapping

**Traces to:** ARCH-58

Given: Retired memory "old-targ-reminder" with retired_by="CLAUDE.md:use-targ", surfaced_count=200, followed=15
When: I run engram registry init
Then: Registry entry for "claude-md:CLAUDE.md:use-targ" has absorbed array with one entry from retired memory

---

### T-192: Quadrant classification — Working

**Traces to:** ARCH-59

Given: An instruction with surfaced_count=150 (> threshold), effectiveness=0.85 (> threshold)
When: I classify it
Then: Quadrant is "Working"

---

### T-193: Quadrant classification — Leech

**Traces to:** ARCH-59

Given: A CLAUDE.md instruction with surfaced_count=200 (always-loaded, maximal), effectiveness=0.20 (< threshold)
When: I classify it
Then: Quadrant is "Leech" (binary, not HiddenGem)

---

### T-194: Quadrant classification — HiddenGem

**Traces to:** ARCH-59

Given: A rule instruction with surfaced_count=5 (< threshold), effectiveness=0.95 (> threshold)
When: I classify it
Then: Quadrant is "HiddenGem"

---

### T-195: Registry interface DI boundary

**Traces to:** ARCH-60

Given: A test that injects mock Registry interface
When: The test calls internal/registry functions
Then: No os.*, io.*, or file operations happen in internal/ — all I/O deferred to concrete implementation

---

### T-196: CLI subcommand registry init — dry-run

**Traces to:** ARCH-61

Given: `engram registry init --dry-run`
When: I run the command
Then: Registry file is not written, only summary of what would be created is printed

---

### T-197: CLI subcommand review — quadrant output

**Traces to:** ARCH-61

Given: `engram review --format json`
When: I run the command and the registry has 10 instructions (3 Working, 2 Leech, 3 HiddenGem, 2 Noise)
Then: Output is JSON array grouped by quadrant, showing all entries

---

### T-198: CLI subcommand merge — absorbs and deletes

**Traces to:** ARCH-61

Given: Two instructions X and Y, and `engram registry merge --source memory:X.toml --target claude-md:CLAUDE.md:X`
When: I run the command
Then: Y's absorbed array includes X's counters, X is deleted from registry (and source file deleted if applicable)

---

### T-199: Hook auto-integration — surfacing

**Traces to:** ARCH-61

Given: A surfacing hook that calls Registry.RecordSurfacing("instruction-id")
When: The hook fires
Then: Registry entry is updated: surfaced_count incremented, last_surfaced set to current time

---

### T-200: Hook auto-integration — evaluation

**Traces to:** ARCH-61

Given: An evaluation hook that calls Registry.RecordEvaluation("instruction-id", "followed")
When: The hook fires
Then: Registry entry is updated: followed counter incremented

---

## L4 → ARCH Traceability (UC-23)

| ARCH Item | Test Coverage |
|-----------|--------------|
| ARCH-52 | T-182 |
| ARCH-53 | T-183 |
| ARCH-54 | T-184, T-185 |
| ARCH-55 | T-186 |
| ARCH-56 | T-187, T-188 |
| ARCH-57 | T-189 |
| ARCH-58 | T-190, T-191 |
| ARCH-59 | T-192, T-193, T-194 |
| ARCH-60 | T-195 |
| ARCH-61 | T-196, T-197, T-198, T-199, T-200 |

All UC-23 ARCH items have test coverage.
