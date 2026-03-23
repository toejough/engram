# Memory Consolidation Design

**Date:** 2026-03-23
**Issues:** #368 (consolidation), #360 (retroactive scoring), #346 (keyword refinement wiring)
**Status:** Draft

## Problem

The memory system loses important principles through a signal loss race condition:

1. User repeatedly teaches the same principle ("do DI") across sessions
2. Haiku over-specifies each extraction: "use DI for database access in auth module", "inject HTTP clients in payment service", etc.
3. Each memory has narrow keywords → surfaces rarely → never reaches the 5-evaluation threshold for maintenance
4. Generalizability scoring labels them project-specific (2-3) → penalized in cross-project surfacing
5. Keyword refinement narrows keywords further on irrelevant surfacing → even less visibility
6. Decay/removal kills them after sustained low activity
7. The principle "always use DI" — the actual signal — is destroyed before consolidation can extract it

Implementing scoring (#360) and keyword refinement (#346) without consolidation (#368) accelerates this problem. Scoring makes specific memories look disposable, keyword refinement reduces their visibility, and decay kills them off — all before the shared principle can be detected.

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Intervention points | All three: extraction-time, feedback-time, decay-guard | Defense in depth — prevents loss at source, catches misses, last safety net |
| Irrelevant feedback response | Consolidation check first, keyword refinement fallback | The principle may be right even when the specific memory surfaces wrong |
| Original memories after consolidation | Archive to `memories/.archive/` | Reversible without surfacing-pipeline suppression logic |
| Similarity detection | BM25 filter → Haiku confirmation | BM25 is free and fast; Haiku only fires when candidates exist |
| Feedback counter transfer | Carry `followed_count` + `contradicted_count`, reset `irrelevant_count` + `ignored_count` + `surfaced_count` | Positive signal preserved; negative signal was caused by bad keywords, not bad principle |
| Provenance | Existing `AbsorbedRecord` schema | Already in the data model; full originals available in archive |
| Consolidated memory confidence | B | System-inferred principle, not user-invoked (no "always"/"never"/"remember") |
| Cluster overlap resolution | Smallest cluster wins | Large clusters survive losing a member; small clusters at threshold are fragile |

## Architecture

### Relationship to `internal/signal.Consolidator`

The existing `internal/signal.Consolidator` handles keyword-overlap clustering (>50% overlap, Union-Find) with backup, deletion, link recomputation, principle synthesis, and dry-run support. This design **extends** the existing consolidator rather than replacing it:

- **Reuse:** Backup (`BackupWriter`), deletion (`FileDeleter`), link recomputation (`LinkRecomputer`), principle synthesis (`PrincipleSynthesizer`), file writing (`MemoryWriter`), effectiveness scoring (`EffectivenessReader`), and the `AbsorbedRecord`/merge infrastructure.
- **Replace:** Keyword-overlap clustering (`buildClusters`/`overlaps`) with BM25 → Haiku confirmation. The existing approach misses semantic similarity ("dependency injection" vs "invert dependencies") and produces false positives on generic keyword overlap.
- **Add:** Three intervention points (`BeforeStore`, `OnIrrelevant`, `BeforeRemove`), counter transfer logic, `RefinementContext` for #346, and the `migrate-scores` subcommand.

The new clustering logic lives in `internal/signal` alongside the existing consolidator, extending it with BM25 candidate retrieval and Haiku confirmation as new `ConsolidatorOption` dependencies. The existing `Consolidate()` and `Plan()` methods are updated to use the new clustering when a `Scorer` and `Confirmer` are provided, falling back to keyword-overlap when they are not (backward compatibility for tests and environments without API access).

### Interfaces

The consolidator reuses existing interfaces from `internal/signal` and adds:

```go
// Scorer retrieves candidate memories similar to a query memory.
type Scorer interface {
    FindSimilar(ctx context.Context, query *memory.Stored, exclude []string) ([]ScoredCandidate, error)
}

// ScoredCandidate is a memory with its BM25 similarity score.
type ScoredCandidate struct {
    Memory *memory.Stored
    Score  float64
}

// Confirmer asks an LLM whether candidate memories share a principle.
type Confirmer interface {
    ConfirmClusters(ctx context.Context, query *memory.Stored, candidates []ScoredCandidate) ([]ConfirmedCluster, error)
}

// ConfirmedCluster is a group of memories confirmed to share a principle.
type ConfirmedCluster struct {
    Members   []*memory.Stored
    Principle string // one-sentence description of the shared principle
}

// Extractor creates a generalized memory from a confirmed cluster.
type Extractor interface {
    ExtractPrinciple(ctx context.Context, cluster ConfirmedCluster) (*memory.MemoryRecord, error)
}
```

### Action Types

```go
type ActionType int

const (
    StoreAsIs ActionType = iota
    Consolidated
    RefineKeywords
    ProceedWithRemoval
)

type Action struct {
    Type             ActionType
    Consolidated     *memory.MemoryRecord  // non-nil when Type == Consolidated
    Archived         []string              // slugs archived during consolidation
    RefinementContext *RefinementContext    // non-nil when Type == RefineKeywords
}

type RefinementContext struct {
    Memory          *memory.MemoryRecord
    SurfacingQuery  string   // query that caused the false positive
    MatchedKeywords []string // which keywords matched
    ToolName        string   // if surfaced during tool use
    ToolInput       string   // if surfaced during tool use
}
```

`RefinementContext` is the wiring contract for #346. This design produces it; #346 implements the consumer.

### Public Methods

Three methods, one per intervention point:

- `BeforeStore(ctx, candidate) → (Action, error)` — extract pipeline, before writing new memory
- `OnIrrelevant(ctx, memory) → (Action, error)` — feedback pipeline, after recording irrelevant mark
- `BeforeRemove(ctx, memory) → (Action, error)` — maintain/decay pipeline, before removal proposal

## Clustering Algorithm

### Step 1: BM25 Candidate Retrieval

- **Query:** memory's `title` + `principle` + `keywords` (exclude `content` — too verbose)
- **Scoring:** BM25 against all existing memories' `title + principle + keywords`
- **Threshold:** Starting point of score ≥ 0.3, calibrated empirically during migration dry-run. BM25 scores are not normalized — the right threshold depends on corpus size and document length. The migration dry-run is the calibration opportunity: inspect proposed clusters, adjust threshold until clusters match human judgment.
- **Cap:** top 10 candidates
- **Exclusions:** the query memory itself; memories already absorbed by another

### Step 2: Haiku Confirmation

Prompt:

> "Here is a memory and N candidate memories. Group any that express the same underlying principle. A cluster must share a generalizable teaching — not just similar keywords or the same project. Do any contradict each other? Return clusters as groups of memory slugs, with a one-sentence description of the shared principle for each cluster. Exclude contradictory members. If no memories share a principle, return empty."

**Cluster minimum size:** 3 total memories (the query memory + at least 2 BM25 candidates). This matches the CLAUDE.md tier 2 promotion rule ("generated from memory clusters, size >= 3"). The existing `internal/signal` uses `minClusterSize = 2`; this design raises it to 3 for the semantic clustering path because Haiku confirmation adds confidence that justifies requiring one more member.

**Cost bound:** Haiku only fires when BM25 finds ≥ 2 candidates above threshold (query + 2 = 3 total, meeting the minimum). Most memories (no siblings) → BM25-only, zero API cost.

## Principle Extraction & Memory Creation

When a cluster is confirmed, Haiku creates the generalized memory:

> "These N memories all express the same underlying principle. Create a single generalized memory that captures the principle without project-specific details. The memory should be useful to any developer in any project. Return: title, principle, anti_pattern, content, keywords, concepts, generalizability (1-5)."

### Field Construction

| Field | Source |
|-------|--------|
| `title` | Haiku-generated (generalized) |
| `principle` | Haiku-generated |
| `anti_pattern` | Haiku-generated |
| `content` | Haiku-generated |
| `keywords` | Haiku-generated, then IDF-filtered (using the post-archival memory corpus — archived memories excluded from IDF calculation) |
| `concepts` | Haiku-generated |
| `generalizability` | Haiku-scored (cluster of 2s should produce 4-5) |
| `confidence` | `"B"` (system-inferred, not user-invoked) |
| `project_slug` | Empty (generalized, not project-bound) |
| `followed_count` | Sum of originals |
| `irrelevant_count` | 0 (reset) |
| `ignored_count` | 0 (reset) |
| `contradicted_count` | Sum of originals |
| `surfaced_count` | 0 (reset) |
| `absorbed` | One `AbsorbedRecord` per original |
| `enforcement_level` | Max of originals' levels (escalation preserved; `transitions` records from originals are not carried over — the archive preserves the full audit trail) |

### Archival

- Move originals to `memories/.archive/` with TOML unchanged
- Rewrite graph links pointing to archived memories → point to consolidated memory (reuse existing `LinkRecomputer.RecomputeAfterMerge()` from `internal/signal`)

### Updating Existing Consolidated Memory

When a new sibling arrives for an already-consolidated memory: update the consolidated memory's content/principle (Haiku may refine), add new `AbsorbedRecord`, add new candidate's `followed_count`. Archive the new candidate. No new memory created.

## Integration Points

### Extract Pipeline (`internal/learn`, `internal/correct`)

After dedup, before writing to disk:

```
candidate → consolidator.BeforeStore(ctx, candidate)
  → Consolidated: write consolidated memory, archive cluster, skip storing candidate
  → StoreAsIs: write candidate as normal
```

The candidate becomes part of the cluster — never written as an individual.

### Feedback Pipeline (`internal/cli/feedback.go`)

After recording `--irrelevant`:

```
memory → consolidator.OnIrrelevant(ctx, memory)
  → Consolidated: write consolidated memory, archive cluster
  → RefineKeywords: populate RefinementContext (consumer is #346, not yet implemented)
```

Feedback pipeline must pass surfacing context (query, tool name/input) through to `OnIrrelevant` so `RefinementContext` can be populated.

### Maintain/Decay Pipeline (`internal/maintain`)

Before generating a `remove` proposal for Noise-quadrant memory:

```
memory → consolidator.BeforeRemove(ctx, memory)
  → Consolidated: write consolidated memory, archive cluster, skip removal
  → ProceedWithRemoval: generate removal proposal as normal
```

No memory dies if it has unconsolidated siblings.

## Retroactive Scoring & Migration (#360)

### New Subcommand: `engram migrate-scores`

1. **Scan** all memories with `generalizability == 0` (unscored)
2. **Batch score** via Haiku (groups of 10-20). Same generalizability litmus test as extraction time.
3. **Write scores** back to TOML files
4. **Cluster scan** — run consolidation clustering across all newly-scored memories (batch mode, full corpus)
5. **Propose consolidations** — output proposed clusters and generalized principles. Dry-run by default.
6. **Apply** — with `--apply`, execute consolidations

### Dry-Run Output

```
Scored 3,140 memories.

Found 47 clusters:

Cluster 1 (5 memories) → "Always use dependency injection for I/O"
  - di-for-database-access (gen: 2, project: engram)
  - inject-http-clients (gen: 2, project: payments)
  - constructor-injection-pattern (gen: 3, project: engram)
  - di-for-file-operations (gen: 2, project: cli-tools)
  - mock-interfaces-not-implementations (gen: 3, project: engram)

Cluster 2 (3 memories) → "Check VCS state before destructive operations"
  ...

Unclusterable: 2,891 memories (no siblings above threshold)
Low generalizability (< 2): 312 memories (candidates for removal)
```

### Cost

~160 Haiku calls for scoring (3,140 / 20 per batch) + ~50 for cluster confirmation. Under $1.

One-time operation. After migration, new memories are scored at extraction time.

## Error Handling & Edge Cases

### Haiku Unavailable

All intervention points degrade gracefully:
- `BeforeStore` → `StoreAsIs`
- `OnIrrelevant` → `RefineKeywords` (populate context for future #346). Since #346 is not yet implemented, the feedback pipeline logs a warning and takes no action when it receives `RefineKeywords`. The `RefinementContext` is constructed but discarded — this is expected until #346 ships.
- `BeforeRemove` → `ProceedWithRemoval` (log warning — safety net failing)

### Cluster Overlap

A memory belonging to multiple clusters is assigned to the **smallest cluster first**. Large clusters survive losing a member; a cluster of 3 losing one drops below threshold and the principle is lost.

### Contradictory Cluster Members

Haiku confirmation prompt asks for contradictions. Contradictory members are excluded. A cluster of 5 with 2 contradictions reduces to 3 (still valid) or drops below threshold.

### Archive Directory

Create `memories/.archive/` on first archival via `os.MkdirAll` at the wiring layer.

### Concurrent Sessions

Two sessions running `BeforeStore` simultaneously with candidates that would form a cluster together. Atomic writes (temp + rename) prevent corruption. Worst case: both store individual memories. Next `maintain` run catches the cluster via `BeforeRemove`.

## #346 Wiring Contract

This design produces `RefinementContext` when `OnIrrelevant` finds no cluster. The contract for #346:

**Producer (this design):** `OnIrrelevant` returns `Action{Type: RefineKeywords, RefinementContext: &RefinementContext{...}}` with the memory, surfacing query, matched keywords, and tool context populated.

**Consumer (#346):** Receives `RefinementContext`, identifies which keywords caused the false positive (high corpus frequency + matched the irrelevant query), removes or narrows them, writes updated keywords back to the memory TOML.

**Feedback pipeline responsibility:** Must pass surfacing context (what query/tool caused this memory to surface) through to `OnIrrelevant`. This context is not currently available in the feedback command — it will need to be added.

### Surfacing Context Data Flow

The surfacing context originates in the hooks that invoke `engram feedback`:

1. **Hook captures context:** `post-tool-use.sh` and `user-prompt-submit.sh` already have access to `$TOOL_NAME`, `$TOOL_INPUT`, and the surfacing query (the text that triggered surfacing). These hooks call `engram feedback`.
2. **New CLI flags:** `engram feedback` gains optional flags: `--surfacing-query`, `--tool-name`, `--tool-input`. These are best-effort — if the hook doesn't provide them, the fields are empty strings.
3. **Feedback command passes to consolidator:** `feedback.go` constructs an `OnIrrelevantInput` with the memory + surfacing context and calls `consolidator.OnIrrelevant(ctx, input)`.
4. **Consolidator populates `RefinementContext`:** When returning `RefineKeywords`, the consolidator fills `RefinementContext` with whatever surfacing context was provided. Empty fields are acceptable — #346 should handle partial context gracefully.

## Testing Strategy

### Unit Tests (DI, no I/O)

- `BeforeStore`: 0, 1, 2, 3+ similar memories → correct action type
- `OnIrrelevant`: cluster found → `Consolidated`; no cluster → `RefineKeywords` with populated context
- `BeforeRemove`: cluster found → `Consolidated`; no cluster → `ProceedWithRemoval`
- Counter transfer: `followed_count` summed, `irrelevant_count` zeroed, `contradicted_count` summed
- Cluster overlap: smallest-first assignment, no double-consolidation
- Contradictory members excluded
- Haiku unavailable → graceful degradation
- Existing consolidated memory updated on new sibling arrival

### Integration Tests (thin wiring)

- Archive directory creation
- TOML read/write round-trip for consolidated memory with `absorbed` records
- Graph link rewriting from archived → consolidated

### Migration Command Tests

- Dry-run outputs proposed clusters without writing
- `--apply` creates consolidated memories and archives originals
- Idempotent: running twice doesn't re-score or re-consolidate. Idempotency mechanism: scored memories have `generalizability > 0` (skipped by scan). Consolidated memories have non-empty `absorbed` records (excluded from clustering as candidates). Archived memories are not in `memories/` (not scanned).
