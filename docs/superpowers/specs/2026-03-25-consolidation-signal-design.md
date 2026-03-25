# Consolidation Signal in Maintenance (#373, sub-feature 2)

## Problem

`engram maintain` silently auto-merges keyword-overlap clusters using mechanical rules (longest principle wins, keyword union). This produces low-quality merges without human judgment. The user never sees what was merged or gets to approve it.

## Design

### Delete keyword-overlap auto-merge

Remove the mechanical merge path from `consolidate.go`:
- `Consolidate()` — auto-merge entry point (returns `ConsolidateResult` — delete the struct too, or rename to avoid collision with `maintain.ConsolidateResult`)
- `mergeCluster()` — performs mechanical merge
- `selectSurvivor()` — picks winner by effectiveness
- `processAbsorbed()` — backs up and deletes losers
- `recomputeLinks()` — link recomputation after merge
- `collectPrinciples()` — only called by `mergeCluster`
- `countNewKeywords()` — only called by `mergeCluster`
- `ConsolidateResult` struct (the batch result type with `ClustersFound`, `MemoriesMerged`, `Errors`)
- `With*` options only used by the merge path (verify each — keep any used by semantic path)
- All tests in `consolidate_test.go` that call `Consolidate()` or test merge internals (~34 tests)
- All tests in `consolidate_migrate_test.go` that test `ConsolidateBatch` (~12 tests)

### Keep

- `Plan()` — returns `[]MergePlan` (clusters without merging)
- `buildClusters()` — Union-Find cluster detection
- `clusterConfidence()` — TF-IDF similarity scoring
- `NewConsolidator` and `Consolidator` struct (trimmed of dead fields)
- `consolidate_semantic.go` — LLM-driven merge via `ExtractPrinciple()` (used at apply-time when user confirms)
- `consolidate_transfer.go` — counter transfer rules
- `consolidate_types.go` — shared types

### Emit consolidation proposals

In `RunMaintain` (`internal/cli/cli.go`), replace `consolidator.Consolidate(ctx)` with:

1. Call `consolidator.Plan(ctx)` to get clusters
2. Convert each `MergePlan` to a `maintain.Proposal`:

```json
{
  "memory_path": "memories/survivor.toml",
  "quadrant": "",
  "diagnosis": "3 memories share keywords [deploy, production] (confidence: 0.82). Consider consolidating.",
  "action": "consolidate",
  "details": {
    "members": [
      {"path": "memories/a.toml", "title": "Deploy tips"},
      {"path": "memories/b.toml", "title": "Production deploy"},
      {"path": "memories/c.toml", "title": "Deploy checklist"}
    ],
    "shared_keywords": ["deploy", "production"],
    "confidence": 0.82
  }
}
```

`memory_path` is the first member (arbitrary — the LLM synthesizes a new memory at apply-time, not a survivor selection).

3. Append consolidation proposals to the main proposals list

### Hook rendering

In `session-start.sh`, add consolidation to triage (follow existing pattern for noise/leech/etc.):

```bash
CONSOLIDATE_COUNT=$(echo "$SIGNAL_OUTPUT" | jq '[.[] | select(.action == "consolidate")] | length' 2>/dev/null) || CONSOLIDATE_COUNT=0
```

- Include in summary line: `"N consolidation candidates"` alongside existing counts
- Add details section: `"## Consolidation Candidates"` with member titles and shared keywords (same pattern as Noise/Leech detail sections)

### Apply-time flow

When the user confirms a consolidation proposal (via the memory-triage skill), the conversation model:
1. Calls the semantic consolidation path (`ExtractPrinciple` via LLM) to synthesize a new generalized memory
2. `TransferFields` applies counter transfer rules
3. Archives originals
4. This is existing functionality in `consolidate_semantic.go` — no changes needed

## Files changed

| File | Change |
|------|--------|
| `internal/signal/consolidate.go` | Delete `Consolidate`, `mergeCluster`, `selectSurvivor`, `processAbsorbed`, `recomputeLinks`, dead `With*` options and struct fields |
| `internal/signal/consolidate_test.go` (and `consolidate_migrate_test.go`) | Delete tests for removed functions |
| `internal/cli/cli.go` | Replace `consolidator.Consolidate(ctx)` with `Plan(ctx)` → proposals |
| `internal/maintain/maintain.go` | Add `"consolidate"` action constant |
| `hooks/session-start.sh` | Add consolidation count and details to triage rendering |

## Not in scope

- Modifying the semantic consolidation path (`consolidate_semantic.go`)
- Modifying `TransferFields`
- Building a new `apply-proposal --action consolidate` CLI path (the memory-triage skill handles this conversationally)
- Changing the keyword-overlap detection algorithm in `buildClusters()`
