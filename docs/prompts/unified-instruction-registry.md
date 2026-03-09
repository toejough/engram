# Unified Instruction Registry

## Problem

Engram tracks effectiveness only for one instruction source (memories). Other sources — CLAUDE.md entries, MEMORY.md entries, rules, skills, hooks — have no feedback loop. When a memory duplicates a CLAUDE.md instruction, deleting the memory destroys its violation history because nothing links it to the covering instruction.

More broadly: the system can't answer "which CLAUDE.md entries are being violated?" or "should this rule be promoted to CLAUDE.md?" because it only measures memory effectiveness.

## Solution

A **unified instruction registry** that tracks effectiveness, frecency, and lifecycle state for ALL instruction sources. This replaces the current fragmented tracking (surfacing-log.jsonl, creation-log.jsonl, evaluations/*.jsonl, inline memory TOML stats) with a single bounded store.

## Design Principles

1. **One store, bounded growth.** The registry has one line per instruction. No append-only event logs that grow unbounded.
2. **Overlay, not replacement.** Instructions still live in their native sources (TOML files, CLAUDE.md, rules). The registry indexes them and tracks effectiveness alongside.
3. **Same quadrant model, all sources.** The existing Working/Leech/HiddenGem/Noise classification extends to every registered instruction.
4. **Consolidation, not addition.** The registry replaces 3 existing stores (surfacing-log, creation-log, evaluations/), reducing total data files from 6 to 4.

---

## Taxonomy

### Instruction Sources

| Source Type | Loading Behavior | Granularity | Identity Pattern |
|-------------|-----------------|-------------|-----------------|
| `claude-md` | Always (every turn) | Per bullet/entry | `claude-md:<file-slug>:<entry-slug>` |
| `memory-md` | Always (every turn) | Per bullet/entry | `memory-md:<entry-slug>` |
| `memory` | BM25 match | Per file | `memory:<filename>` |
| `rule` | File-pattern match | Per file | `rule:<filename>` |
| `skill` | Similarity/invocation | Per skill | `skill:<name>` |
| `hook` | Event-triggered | Per hook output | `hook:<event>:<name>` |

**Salience hierarchy** (higher = more reliably seen by model, higher maintenance cost):

```
deterministic code > claude-md > rule > memory-md > skill > memory
```

Instructions move UP when they need more reliability (promotion), DOWN when they're too narrow for their current tier (demotion).

### Signals Per Instruction

| Signal | What It Measures | How Updated |
|--------|-----------------|-------------|
| `surfaced_count` | Times presented to model | Increment on surfacing |
| `last_surfaced` | Most recent presentation | Timestamp on surfacing |
| `followed` | Times model complied | Increment on positive evaluation |
| `contradicted` | Times model did the opposite | Increment on negative evaluation |
| `ignored` | No evidence of awareness | Increment on null evaluation |
| `effectiveness` | `followed / (followed + contradicted + ignored)` | Computed on read |
| `frecency` | Frequency x recency blend | Computed on read (decay function) |
| `content_hash` | Detect instruction changes | Updated on registration/content change |
| `absorbed` | History from deleted duplicates | Appended on merge |

**Always-loaded sources** (claude-md, memory-md) have definitionally maximal surfacing frequency. Their quadrant collapses to binary: Working (followed) or Leech (not followed). The full quadrant model only applies to conditionally-loaded sources (memory, rule, skill).

### Quadrant Classification

| Quadrant | Surfacing | Effectiveness | Applicable Sources |
|----------|-----------|--------------|-------------------|
| **Working** | High | High | All |
| **Leech** | High | Low | All |
| **Hidden Gem** | Low | High | memory, rule, skill |
| **Noise** | Low | Low | memory, rule, skill |
| **Insufficient** | Any | < N evaluations | All |

### Lifecycle Actions

| Action | Trigger | Effect |
|--------|---------|--------|
| **register** | New instruction detected | Add to registry with source, content hash |
| **evaluate** | Session end / PreCompact | Assess compliance, update outcome counters |
| **classify** | Sufficient evaluations | Assign quadrant |
| **rewrite** | Leech — content quality problem | Improve wording, update content hash |
| **escalate** | Leech — rewrite didn't help | Move up enforcement ladder (advisory -> reminder -> blocking) |
| **automate** | Leech — mechanical pattern | Convert to deterministic code, delete instruction |
| **broaden** | Hidden gem | Expand keywords/triggers |
| **promote** | High effectiveness + broad applicability | Move up salience hierarchy |
| **demote** | Narrow applicability in high-salience source | Move down salience hierarchy |
| **merge** | Duplicate detected across sources | Absorb effectiveness into surviving instruction, delete duplicate |
| **remove** | Noise — no value | Delete instruction, preserve final stats in absorbed history of nearest surviving relative |

---

## Data Model

### Final State: 4 Bounded Stores

| Store | Purpose | Growth Bound |
|-------|---------|-------------|
| `data/instruction-registry.jsonl` | Effectiveness state per instruction | One line per instruction |
| `data/memories/` | Memory instruction content (no inline stats) | One file per memory |
| `data/learn-offset.json` | Incremental learning cursor | Fixed size |
| `data/session-context.md` | Session continuity | Capped at 1KB (QW-3) |

### Replaces (Migration Required)

| Old Store | Absorbed By |
|-----------|------------|
| `data/surfacing-log.jsonl` | `instruction-registry.jsonl` (surfaced_count, last_surfaced fields) |
| `data/creation-log.jsonl` | `instruction-registry.jsonl` (registered_at field) |
| `data/evaluations/*.jsonl` | `instruction-registry.jsonl` (followed/contradicted/ignored counters) |
| Memory TOML `surfaced_count`, `last_surfaced`, `surfacing_contexts` | `instruction-registry.jsonl` |
| Memory TOML `retired_by`, `retired_at` | Merge events in registry `absorbed` field |

### Registry Entry Schema

```jsonl
{
  "id": "claude-md:project:use-targ",
  "source_type": "claude-md",
  "source_path": "CLAUDE.md",
  "title": "Use targ for all build/test/check",
  "content_hash": "a1b2c3d4",
  "registered_at": "2026-03-01T00:00:00Z",
  "updated_at": "2026-03-08T00:00:00Z",
  "surfaced_count": 1847,
  "last_surfaced": "2026-03-08T15:30:00Z",
  "evaluations": {
    "followed": 45,
    "contradicted": 12,
    "ignored": 3
  },
  "absorbed": [
    {
      "from": "memory:always-use-targ-reminder.toml",
      "surfaced_count": 926,
      "evaluations": {"followed": 12, "contradicted": 35, "ignored": 8},
      "merged_at": "2026-03-08T18:00:00Z"
    },
    {
      "from": "memory:always-use-targ-reminder-2.toml",
      "surfaced_count": 412,
      "evaluations": {"followed": 5, "contradicted": 20, "ignored": 4},
      "merged_at": "2026-03-08T18:00:00Z"
    }
  ]
}
```

### Memory TOML Simplification

Before (content + effectiveness mixed):
```toml
title = "Always use targ"
content = "remember to always use targ"
keywords = ["targ", "always use"]
principle = "Use targ consistently"
anti_pattern = "Using raw go test"
confidence = "A"
created_at = "2026-03-05T01:50:56Z"
updated_at = "2026-03-05T01:50:56Z"
# effectiveness fields — REMOVE, tracked in registry
surfaced_count = 926
last_surfaced = "2026-03-07T15:33:17-05:00"
surfacing_contexts = ["tool", "tool", "tool"]
retired_by = "CLAUDE.md:use-targ-build-system"
retired_at = "2026-03-08T18:00:00Z"
```

After (content only):
```toml
title = "Always use targ"
content = "remember to always use targ"
keywords = ["targ", "always use"]
principle = "Use targ consistently"
anti_pattern = "Using raw go test"
confidence = "A"
created_at = "2026-03-05T01:50:56Z"
updated_at = "2026-03-05T01:50:56Z"
```

Effectiveness, surfacing history, retirement status — all in the registry.

---

## Pipeline Changes

### Surfacing (surface.go)

**Current:** Logs to surfacing-log.jsonl. Reads surfaced_count from memory TOML.
**New:** Updates instruction-registry.jsonl (increment surfaced_count, update last_surfaced). Reads frecency from registry for ranking.

Filter logic: instead of checking `RetiredBy != ""` on the memory struct, check the registry for a `merged_at` or absent entry. Or simpler: if the memory file doesn't exist, it's been deleted — skip it.

### Evaluation (evaluate)

**Current:** Writes per-session JSONL files to evaluations/ directory. Only evaluates surfaced memories.
**New:** Updates instruction-registry.jsonl (increment followed/contradicted/ignored counters). Evaluates ALL active instructions, not just memories — claude-md and rule entries that were active during the session also get evaluated.

Evaluation priority (to manage LLM cost):
- Always evaluate: instructions with low effectiveness, recently changed instructions, instructions with absorbed duplicates
- Sample evaluate: working instructions (spot-check)
- Skip: instructions with insufficient surfacing in this session

### Review (review)

**Current:** Aggregates from surfacing-log + evaluations/ to classify memories into quadrants.
**New:** Reads registry directly. Classification is trivial — all data is pre-aggregated.

### Maintain (maintain)

**Current:** Generates proposals for classified memories only.
**New:** Generates proposals for ALL classified instructions. A leech CLAUDE.md entry gets a rewrite proposal just like a leech memory. A noise rule gets a remove proposal.

### Escalation (UC-21 escalation engine)

**Current:** Operates on memory paths.
**New:** Operates on instruction IDs. Can escalate any instruction type. The `absorbed` field provides historical context — "this instruction has absorbed 4 duplicates with 2000+ combined surfacings and 20% effectiveness, suggesting chronic violation."

### Learn (learn)

**Current:** Creates memory TOML files with inline stats fields.
**New:** Creates memory TOML files (content only) + registers the new memory in instruction-registry.jsonl.

### Instruct Audit (UC-20)

**Current:** Cross-source analysis via LLM.
**New:** Registry makes cross-source analysis data-driven. Duplicates detected by content_hash similarity. Quality problems identified by low effectiveness across multiple sources covering the same concept.

---

## Migration Plan

### Phase 1: Build Registry + Backfill

1. Define `InstructionEntry` struct in `internal/registry/` with all fields from schema above
2. Define `Registry` interface: `Register`, `RecordSurfacing`, `RecordEvaluation`, `Merge`, `Remove`, `List`, `Get`
3. Implement JSONL-backed registry (read all lines on load, write full file on save — simple, bounded)
4. Build backfill command: `engram registry init`
   - Scan `memories/` for all TOML files — register each as `memory:<filename>`
   - Read `surfacing-log.jsonl` — aggregate surfaced_count and last_surfaced per memory path
   - Read `evaluations/*.jsonl` — aggregate followed/contradicted/ignored per memory path
   - Read `creation-log.jsonl` — set registered_at per memory path
   - Write `instruction-registry.jsonl`

### Phase 2: Wire Surfacing + Evaluation

1. Update `surface.go` to call `Registry.RecordSurfacing` instead of writing surfacing-log.jsonl
2. Update `surface.go` to read frecency from registry instead of memory TOML surfaced_count
3. Update `evaluate` to call `Registry.RecordEvaluation` instead of writing to evaluations/
4. Update `learn` to call `Registry.Register` instead of writing to creation-log.jsonl
5. Update `filterRetired` to check registry for merged/removed status instead of memory TOML RetiredBy field

### Phase 3: Register Non-Memory Sources

1. Build instruction extractor for CLAUDE.md: parse markdown into individual instruction entries, assign stable slug IDs
2. Register CLAUDE.md entries via `engram registry register-source --type claude-md --path CLAUDE.md`
3. Extend to rules, skills (same pattern — parse file, extract instructions, register)
4. Extend evaluation pipeline to assess compliance for non-memory instructions at session end

### Phase 4: Simplify Memory TOMLs + Cleanup

1. Strip effectiveness fields from memory TOMLs (surfaced_count, last_surfaced, surfacing_contexts, retired_by, retired_at)
2. Delete old stores: surfacing-log.jsonl, creation-log.jsonl, evaluations/
3. Update all code that reads inline memory stats to read from registry instead

### Phase 5: Merge + Lifecycle

1. Implement `engram registry merge --source <id> --target <id>` — absorbs effectiveness, deletes source
2. Implement promote/demote proposals in maintain pipeline
3. Wire absorbed history into escalation engine decisions

---

## Immediate Action: Delete the 33 Retired Duplicates

Before the registry exists, we need to handle the 33 memories we retired earlier today. The pragmatic approach:

1. For each retired memory, capture its effectiveness snapshot (surfaced_count from TOML, evaluation data from evaluations/ files if available)
2. Write these snapshots to a temporary `data/pending-attributions.jsonl` file — one line per merge, same schema as the registry `absorbed` field
3. Delete the retired memory TOML files
4. When Phase 1 backfill runs, it reads `pending-attributions.jsonl` and incorporates the absorbed data into the covering instruction's registry entry
5. Delete `pending-attributions.jsonl` after backfill

This preserves the violation history without building the full registry now.

---

## Relationship to Existing UCs

| Open Issue | How Registry Helps |
|-----------|-------------------|
| #38 UC-5 (CLAUDE.md Management) | Registry tracks CLAUDE.md entry effectiveness — data-driven promote/demote proposals |
| #37 UC-4 (Skill Generation) | Registry identifies high-effectiveness memories that should become skills (promote action) |
| #40 UC-7 (Working Artifact Maintenance) | Registry surfaces stale working instructions across all sources, not just memories |
| #41 UC-8 (Leech Diagnosis) | Registry provides cross-source leech data — a CLAUDE.md leech is diagnosed the same as a memory leech |
| #42 UC-9 (Hidden Gem Discovery) | Registry identifies hidden gems across all sources |
| #43 UC-10 (Noise Pruning) | Registry identifies noise across all sources |
| #60 Frecency ranking | Registry computes frecency from surfaced_count + last_surfaced |
| #59 BM25 pruning | Registry effectiveness scores can weight BM25 results |
| #64 Strip preprocessing for evaluate | Orthogonal — improves evaluation quality, which feeds better data into registry |
| #65 Periodic learn extraction | Orthogonal — creates more memories, which get registered |

The registry is the **foundation** for the entire next UC batch, the same way UC-17 (budget) was the foundation for UC-17..22.

---

## Candidate UC Ordering

| Priority | UC | Rationale |
|----------|-----|-----------|
| 1 | **Registry (new)** | Foundation — all other UCs consume registry data |
| 2 | UC-8 (Leech Diagnosis) | Highest immediate value — identifies what's broken across all sources |
| 3 | UC-5 (CLAUDE.md Management) | Promote/demote based on registry effectiveness |
| 4 | UC-10 (Noise Pruning) | Clean up low-value instructions using registry data |
| 5 | UC-9 (Hidden Gem Discovery) | Broaden under-surfaced high-value instructions |
| 6 | UC-7 (Working Maintenance) | Staleness detection across all sources |
| 7 | UC-4 (Skill Generation) | Promote high-value memory clusters to skills |

#59 (BM25 pruning), #60 (frecency), #64 (strip for evaluate), #65 (periodic learn) are enhancements that can be woven into the relevant UCs rather than standalone work items.
