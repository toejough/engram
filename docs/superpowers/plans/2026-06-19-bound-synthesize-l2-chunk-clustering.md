# Bound `--synthesize-l2` Chunk Clustering Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `engram query --synthesize-l2` (the path `/recall` always uses) cluster and emit a bounded top-`limit` slice of matched chunks instead of the entire corpus, so recall stays fast as memory grows.

**Architecture:** `appendSynthesisChunks` in `internal/cli/query.go` currently appends *every* scored chunk to the clustering subgraph and to `items[]` — no sort, no cap. `AutoK`'s silhouette scoring is O(n²·d) per K across 6 K-values, so at the current 13,804-record corpus a single recall pins one core for 10+ minutes. The non-synthesis sibling `mergeChunkSpace` already solved this exact problem by sorting scored chunks by score and clustering only `scored[:limit]`. This plan applies the identical, established bound to the synthesis path: sort by score, truncate to `limit`, build both the clustering members and the returned chunk items from that capped slice. The cap is `limit` (default 20) for consistency with `mergeChunkSpace`, `RunChunkQuery`, and `unionDirectHits` — not an invented constant. No new behavior; purely a bound restoring the invariant the code's own comment already states ("unions every phrase's DIRECT HITS … truncated to limit").

**Tech Stack:** Go; `targ` build system; tests use imptest-style DI + `gomega` + `rapid` (property tests). `internal/cli` package.

---

## Background / Root Cause (verified)

- `internal/cli/query.go` `appendSynthesisChunks` (≈ lines 448–488): loops `for _, s := range scored` over **all** chunk records, appending each to `subgraph.members` (clustered by `AutoK`) and to `chunkItems` (rendered in `items[]`). No sort/cap/floor.
- `runSynthesizeL2Query` (≈ line 2102) calls it, then `clusterUnionForSynthesis` → `cluster.AutoK(vectors, clusterMinK=2, clusterMaxK=7, …)`. `cluster.Silhouette` is O(n²) cosine distances per K (profile confirmed 99% in `cluster.meanIntraDistance`/`meanGroupDistance`→`CosineDistance`).
- `renderQueryPayload` does **not** cap `items[]` to `limit`, and `runSynthesizeL2Query` skips the recency-band cap — so today the synthesis path *also* dumps all 13,804 chunk records into `items[]`. Bounding the appended set fixes clustering cost **and** payload bloat in one change.
- Corpus today: 13,804 records across 1,130 `.jsonl` index files (`/Users/joe/.local/share/engram/chunks`). n=13,804 → ~4×10¹¹ float ops single-threaded → the observed 10+ min.
- Sibling already-correct path for reference — `mergeChunkSpace` (≈ line 1418):
  ```go
  // Cluster the top-limit slice so the O(n²) silhouette stays bounded.
  clusterView := merged.resolvedItems
  if len(clusterView) > limit { clusterView = clusterView[:limit] }
  ```
- `--synthesis` (L3, `runSynthesisQuery`) does **not** append chunks — out of scope. Only `--synthesize-l2` is affected.

## File Structure

- Modify: `internal/cli/query.go`
  - `appendSynthesisChunks` — add a `limit int` parameter; sort scored chunks descending and truncate to `limit` before building members/items.
  - `runSynthesizeL2Query` — pass `limit` to `appendSynthesisChunks` (it is already in scope).
- Test: `internal/cli/query_synthesis_test.go` — add a property test (rapid) locking the bound. Existing tests in this file (`TestQuery_SynthesizeL2_IncludesMatchedChunksInClustering`, etc.) use ≤2 chunks and must still pass unchanged.
- Reuse (no change): `sortScoredDesc` in `internal/cli/recency.go:341`.

## Existing-test compatibility check

- `TestQuery_SynthesizeL2_IncludesMatchedChunksInClustering`: 2 chunks, default limit 20 → 2 ≤ 20, unchanged.
- `TestQuery_SynthesizeL2_NearestL2FromFullVaultNotJustClustered`: notes-only union (no chunks dir), Limit:2 → unaffected.
- `synthesize_l2_property_test.go`: notes only → unaffected.

---

### Task 1: Bound the synthesis chunk set to top-`limit`

**Files:**
- Modify: `internal/cli/query.go` — `appendSynthesisChunks` (≈ 448) and its single caller `runSynthesizeL2Query` (≈ 2130). (`scoreChunks` returns `[]scoredChunk`, see `query_chunks.go:194,211`, so `sortScoredDesc(scored []scoredChunk)` applies directly.)
- Create: `internal/cli/query_synthesis_chunk_bound_property_test.go` (`package cli_test`) — the repo's established per-property-file pattern (cf. `synthesize_l2_property_test.go`), so the `pgregory.net/rapid` import lands cleanly without disturbing `query_synthesis_test.go`.
- Reuse: `sortScoredDesc` (`internal/cli/recency.go:341`).

- [ ] **Step 1: Write the failing property test**

Use the `property-rigor` skill to confirm the property before writing. The invariant: for any matched-chunk count `C > limit`, `--synthesize-l2` returns no more than `limit` chunk-kind items. Create `internal/cli/query_synthesis_chunk_bound_property_test.go` with this content (note the `pgregory.net/rapid` import — the reason for a new file):

```go
package cli_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"go.yaml.in/yaml/v3"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/chunk"
	"github.com/toejough/engram/internal/cli"
)

// TestProperty_SynthesizeL2_ChunkItemsBoundedByLimit locks the perf-critical
// bound: matched chunks fed to the unified clustering (and emitted in items[])
// are capped at limit, so silhouette stays O(limit^2) regardless of corpus
// size. Without the cap, appendSynthesisChunks clusters and emits the entire
// corpus, making recall O(corpus^2).
func TestProperty_SynthesizeL2_ChunkItemsBoundedByLimit(t *testing.T) {
	t.Parallel()

	queryVec := []float32{1, 0, 0, 0}

	rapid.Check(t, func(rt *rapid.T) {
		limit := rapid.IntRange(1, 8).Draw(rt, "limit")
		// Strictly more matched chunks than the limit.
		chunkCount := limit + rapid.IntRange(1, 25).Draw(rt, "extraChunks")

		vault := t.TempDir()
		memFS := newInMemoryFS()

		// One L1 note so the union is non-empty and clustering runs.
		plantDualVector(t, memFS, vault, "1.ep.md",
			"---\ntype: episode\ntier: L1\nsituation: alpha\n---\n\nb\n", queryVec, queryVec)

		records := make([]chunk.Record, 0, chunkCount)
		for i := range chunkCount {
			text := fmt.Sprintf("chunk alpha %d", i)
			records = append(records, chunk.Record{
				Source:      "/sessions/s.jsonl",
				Anchor:      fmt.Sprintf("turn-%d", i),
				ContentHash: chunk.HashText(text),
				Text:        text,
				Vector:      queryVec,
			})
		}

		data, encErr := chunk.EncodeRecords(records)
		if encErr != nil {
			rt.Fatalf("encode records: %v", encErr)
		}

		memFS.files["/chunks/s.jsonl"] = data

		deps := newQueryDeps(memFS)
		deps.Embedder = fixedVectorEmbedder{modelID: "m@4", vector: queryVec}
		deps.ListChunkIndexes = func(string) ([]string, error) {
			return []string{"/chunks/s.jsonl"}, nil
		}

		var out bytes.Buffer

		err := cli.RunQuery(context.Background(),
			cli.QueryArgs{
				Phrases:      []string{"alpha"},
				VaultPath:    vault,
				SynthesizeL2: true,
				ChunksDir:    "/chunks",
				Limit:        limit,
			},
			deps, &out)
		if err != nil {
			rt.Fatalf("RunQuery: %v", err)
		}

		var raw map[string]any
		if err := yaml.Unmarshal(out.Bytes(), &raw); err != nil {
			rt.Fatalf("unmarshal: %v", err)
		}

		items, ok := raw["items"].([]any)
		if !ok {
			rt.Fatalf("payload has no items[] list (got %T)", raw["items"])
		}

		chunkItems := 0
		for _, item := range items {
			mapped, _ := item.(map[string]any)
			if kind, _ := mapped["kind"].(string); kind == "chunk" {
				chunkItems++
			}
		}

		if chunkItems > limit {
			rt.Fatalf("chunk items = %d, must be <= limit %d (chunkCount=%d)",
				chunkItems, limit, chunkCount)
		}
	})
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `targ test`
Expected: FAIL in `TestProperty_SynthesizeL2_ChunkItemsBoundedByLimit` — `chunk items = N, must be <= limit L` (the corpus is emitted uncapped).

- [ ] **Step 3: Add `limit` to `appendSynthesisChunks` and bound the scored set**

In `internal/cli/query.go`, change the signature and the post-scoring block. Current:

```go
func appendSynthesisChunks(
	ctx context.Context,
	args QueryArgs,
	deps QueryDeps,
	subgraph *expandedSubgraph,
) ([]resolvedItem, error) {
	records, loadErr := loadChunkRecords(args.ChunksDir, ChunkQueryDeps{
		ListIndexes: deps.ListChunkIndexes, ReadFile: deps.Read, Embedder: deps.Embedder,
	})
	if loadErr != nil {
		return nil, loadErr
	}

	scored, scoreErr := scoreChunks(ctx, args.Phrases, records, deps.Embedder)
	if scoreErr != nil {
		return nil, scoreErr
	}

	chunkItems := make([]resolvedItem, 0, len(scored))
```

Replace with:

```go
func appendSynthesisChunks(
	ctx context.Context,
	args QueryArgs,
	deps QueryDeps,
	subgraph *expandedSubgraph,
	limit int,
) ([]resolvedItem, error) {
	records, loadErr := loadChunkRecords(args.ChunksDir, ChunkQueryDeps{
		ListIndexes: deps.ListChunkIndexes, ReadFile: deps.Read, Embedder: deps.Embedder,
	})
	if loadErr != nil {
		return nil, loadErr
	}

	scored, scoreErr := scoreChunks(ctx, args.Phrases, records, deps.Embedder)
	if scoreErr != nil {
		return nil, scoreErr
	}

	// Bound the clustered + returned chunk set to the top-limit by score.
	// cluster.Silhouette is O(n^2) per K, so clustering the whole corpus is
	// prohibitively slow on large indices — the same bound mergeChunkSpace
	// already applies for the non-synthesis path.
	sortScoredDesc(scored)

	if len(scored) > limit {
		scored = scored[:limit]
	}

	chunkItems := make([]resolvedItem, 0, len(scored))
```

(The existing `for _, s := range scored { … }` body and `return chunkItems, nil` stay as-is — they now iterate the capped slice.)

- [ ] **Step 4: Pass `limit` at the call site**

In `runSynthesizeL2Query` (≈ line 2130), change:

```go
		chunkItems, err = appendSynthesisChunks(ctx, args, deps, &subgraph)
```

to:

```go
		chunkItems, err = appendSynthesisChunks(ctx, args, deps, &subgraph, limit)
```

- [ ] **Step 5: Run the test to verify it passes**

Run: `targ test`
Expected: PASS for `TestProperty_SynthesizeL2_ChunkItemsBoundedByLimit` and all existing `internal/cli` tests (the ≤2-chunk synthesis tests are unaffected).

- [ ] **Step 6: Full check**

Run: `targ check-full`
Expected: green (lint + coverage + nilaway). No new magic numbers (uses existing `limit`), descriptive names, comment explains the bound.

- [ ] **Step 7: Manual verification against the real index (REQUIRED — passing tests ≠ usable system)**

Build and run the real binary against the real 13,804-record index from a non-data-dir cwd:

Run:
```
targ build
time engram query --synthesize-l2 --phrase "k-means silhouette clustering performance" --phrase "engram cli vault traversal" --limit 20 >/tmp/recall-after.yaml
```
Expected: completes in seconds (not minutes); `/tmp/recall-after.yaml` is non-empty, contains `kind: chunk` items capped at ≤ limit (20), and `clusters:` are present. Confirm CPU does not pin a core for minutes.

- [ ] **Step 8: Commit**

```bash
git add internal/cli/query.go internal/cli/query_synthesis_chunk_bound_property_test.go
git commit -m "fix(query): bound --synthesize-l2 chunk clustering to top-limit

appendSynthesisChunks clustered and emitted the entire chunk corpus; at
13,804 records the O(n^2) silhouette pinned one core for 10+ minutes on
every recall. Cap the matched-chunk set to limit by score, mirroring the
mergeChunkSpace bound already used by the non-synthesis path.

AI-Used: [claude]"
```

- [ ] **Step 9: File the deferred index-hygiene issue (do not let the deferral get lost)**

```bash
gh issue create \
  --title "engram ingest/recall: prune dead eval-run chunk sources to curb index bloat" \
  --body "Split from the recall-perf fix (bound --synthesize-l2 chunk clustering). ~1,130 of the ~1,130 .jsonl chunk index files are dead /private/tmp/cummatrix-* and gate-* eval-run transcripts (13,804 records total). They cause the 'skip … no such file' noise on every \`engram ingest --auto\` and pollute recall with eval-run narration. The algorithmic bound already shipped keeps recall fast regardless, so this is hygiene, not a perf cure. Decide: (a) should \`engram ingest --auto\` skip non-persistent workspaces (e.g. /private/tmp)? (b) is an \`engram\` subcommand warranted to prune chunks whose source file no longer exists?"
```

- [ ] **Step 10: Doc confirmation (defer to please Step 5 / Gate C)**

After the fix lands, confirm `docs/architecture/c3-components.md` (K6/K8 rows + the synthesis flow) still describes clustering accurately — the synthesis path now matches the non-synthesis bounding discipline, so no contradiction is introduced; update only if a doc statement implies the synthesis path clusters an unbounded set.

---

## Out of scope (tracked by Step 9's issue)

- **Chunk-index hygiene:** ~1,130 source files are dead `/private/tmp/cummatrix-*` / `gate-*` eval-run transcripts (cause of the `skip … no such file` noise on every `engram ingest --auto` and of eval-flavored chunks polluting recall). Pruning them lowers `n` today but is **not** the cure — only the algorithmic bound keeps recall fast as legitimate memory grows. Filed via Step 9.

## Self-Review

1. **Spec coverage:** Goal = bound the synthesis chunk set → Task 1 Steps 3–4 implement it; Steps 1–2 prove the bug; Steps 5–7 verify (incl. real binary). Covered.
2. **Placeholder scan:** None — all code shown in full, exact commands and expected output given.
3. **Type consistency:** `appendSynthesisChunks` gains `limit int`; the only caller (`runSynthesizeL2Query`) updated to pass `limit`. `sortScoredDesc(scored []scoredChunk)` exists in `recency.go:341`. `scoredChunk`, `chunk.Record`, `chunk.EncodeRecords`, `chunk.HashText`, `fixedVectorEmbedder`, `plantDualVector`, `newQueryDeps`, `newInMemoryFS` all used by existing tests in this file. Consistent.
