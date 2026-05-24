# `engram query` spike — spec (rev. 2026-05-24)

Companion to the tiered-memory research log
(`2026-05-22-tiered-memory-research-log.md` → focus list
resolutions F1–F9).

This spike is the **prerequisite** for all other v2 work. It
validates the embedder choice and the embed-on-write +
semantic-query pipeline against the existing vault. If the
spike passes, the code becomes the foundation for v2's
remaining iterations (episodes, clustering, MOC migration).
If it fails, the embedder swaps to MiniLM-L6 (verified) and
the pipeline is the same shape.

The spike is **deliberately narrow.** Clustering, hubs,
provenance-rich payloads, episodes, and MOC migration are
**post-spike v2 work**, not spike scope.

## Goal

Deliver a working semantic-search command (`engram query`)
against the existing vault (138 facts and feedback notes,
25 MOCs) with the embed-on-write pipeline in place. The
narrow go/no-go question on the embedder is UAT case 13. The
other 12 cases verify the pipeline works regardless of which
model.

## Embedder choice

**Default:** Snowflake-arctic-embed-xs via Hugot + GoMLX's
`simplego` backend. 22 MB ONNX, 384 dims, 512-token context,
~62 MTEB.

**Fallback if UAT 13 fails:** `sentence-transformers/all-MiniLM-L6-v2`
same path. 22 MB, 384 dims, 256-token context (truncation
risk on longer notes), ~57 MTEB.

Migration story between the two is one re-embed: every sidecar
tagged with `embedding_model_id` makes the swap explicit.

## What gets embedded in the spike

- All existing facts and feedback in `Permanent/`.
- All existing MOCs in `MOCs/` (treated as ordinary markdown
  for embedding purposes — F4 migration is post-spike).
- **Not in spike:** episode kind (F1 ships in next v2
  iteration with `engram learn episode`).

## UAT cases

### Pipeline correctness

1. **Backfill.** `engram embed --all` populates `.vec.json`
   sidecars for every note in `Permanent/` and `MOCs/` from
   scratch. Each sidecar matches the format below.

2. **Sidecar format.** Sidecar contains `embedding_model_id`,
   `dims`, `vector`, `content_hash`. JSON; one file per note
   alongside the `.md` file.

3. **Auto-embed on write.** After `engram learn fact …`
   succeeds, the new note has its `.vec.json` sidecar before
   the command returns. If embedding fails, the note is still
   written; embedding failure is a warning, not an error.

4. **Stale re-embed.** After editing a note's body markdown,
   `engram embed --stale` identifies it as stale via
   `content_hash` mismatch and re-embeds it. Other notes are
   not touched.

### Query correctness

5. `engram query "verifying current behavior before claiming a
   delta"` returns Permanent/132 in `items` with
   `provenances: [direct]` and a score among the top three.

6. `engram query "exploration sprawl"` returns Permanent/133
   similarly.

7. `items` is ordered by `score` descending; scores
   monotonically decrease down the list. Verifiable by
   hand-computing one pair.

### State inspection

8. `engram embed status` reports five distinct counts: total,
   with-embeddings, without-embeddings, stale, model-mismatch.
   Output format is human-readable (one line per category).

### Edge cases

9. **Empty vault.** `engram query "anything"` returns an
   `items: []` payload, exits 0, no crash.

10. **Missing model file.** `engram query` errors with a clear
    message naming the next action. (Since the model is bundled
    in the binary, this case primarily covers a corrupted
    binary or wrong build — but the error path must exist.)

11. **Stale vs incompatible distinction.** A sidecar whose
    `content_hash` doesn't match the note's body is *stale*
    (re-embedable with current model). A sidecar whose
    `embedding_model_id` doesn't match the binary's current
    model is *incompatible* (requires `--force` to re-embed,
    or model rollback). These two states are reported
    distinctly in `status` and require different actions.

12. **Partial corruption.** A sidecar with malformed JSON or a
    `dims` field that doesn't match `len(vector)` is detected
    and reported as broken in `status`. Offered for re-embed
    under `--stale`.

### Spike-specific verification (the go/no-go gate)

13. **Reference parity.** Pick 5 hand-chosen sentence pairs
    that range from clearly-similar to clearly-different. Embed
    each sentence twice: once in Hugot+Arctic-xs (Go), once in
    `sentence-transformers/snowflake-arctic-embed-xs` (Python).
    Compute cosine similarity per pair. **The Go and Python
    cosines must agree to ~3 decimal places on every pair.**

    Failure modes for this gate:
    - Model fails to load → Arctic-xs not actually supported by
      simplego; fall back to MiniLM-L6.
    - Cosines disagree → embedding implementation differs
      somewhere; investigate before falling back.
    - Loads but produces NaNs or zero vectors → broken
      implementation; fall back to MiniLM-L6.

## Design surface

### CLI

```
engram query <string> [--limit N]
engram embed [--all | --missing | --stale | --force | --dry-run]
engram embed status
```

`engram learn …` (existing) gains an auto-embed step after the
write completes.

### Sidecar file format

Sibling to the `.md` file:

```
Permanent/132.2026-05-23.verify-current-behavior-before-claiming-delta.md
Permanent/132.2026-05-23.verify-current-behavior-before-claiming-delta.vec.json
```

JSON shape:

```json
{
  "embedding_model_id": "snowflake-arctic-embed-xs@384",
  "dims": 384,
  "vector": [0.012, -0.034, ...],
  "content_hash": "sha256:..."
}
```

- `embedding_model_id` — `<name>@<dims>` form. Matryoshka
  truncation explicit. Guards against silent cross-model
  comparison.
- `content_hash` — sha256 over the markdown **body**
  (frontmatter stripped). Frontmatter changes (e.g., adding
  a relation) do not trigger re-embed.
- `vector` — float32 array.
- `dims` — sanity check against `len(vector)`.

### Internal Go interfaces

```go
package embed

type Embedder interface {
    Embed(text string) ([]float32, error)
    ModelID() string
    Dims() int
}

type Sidecar struct {
    EmbeddingModelID string    `json:"embedding_model_id"`
    Dims             int       `json:"dims"`
    Vector           []float32 `json:"vector"`
    ContentHash      string    `json:"content_hash"`
}

type State int
const (
    StateOK State = iota
    StateMissing       // no sidecar
    StateStale         // content_hash mismatch
    StateIncompatible  // model_id mismatch
    StateBroken        // malformed
)
```

### Spike query output (YAML)

The spike implements a **subset** of F7's full payload — only
the parts needed to validate direct-hit retrieval. Forward-
compatible: when clustering ships post-spike, the `clusters`
section is added without breaking the `items` section.

```yaml
version: 1
query: "verifying current behavior"

items:
  - path: Permanent/132.2026-05-23.verify-current-behavior-before-claiming-delta.md
    kind: feedback
    score: 0.91
    provenances: [direct]
    content: |
      ---
      type: feedback
      situation: When restating a user proposal as a delta from an existing system
      ...
      ---
      <full body>
  - path: Permanent/137...
    kind: fact
    score: 0.78
    provenances: [direct]
    content: |
      ...
  # ... up to --limit (default 20)

budget:
  total_notes: 163
  with_embeddings: 163
  direct_hits_returned: 20
  limit: 20
```

- Every item has `provenances: [direct]` in the spike — no
  cluster_rep or hub provenances exist yet.
- `content` includes the full `.md` text (frontmatter + body).
- Ordering: score descending.
- No `clusters` section in the spike. That ships post-spike.

`engram embed status` output (plain text):

```
total:           163
with-embeddings: 0
without:         163
stale:           0
incompatible:    0
broken:          0
```

## Settled decisions

| # | Decision | Choice |
|---|---|---|
| 1 | Content hash scope | **Body only.** Frontmatter changes don't trigger re-embed. |
| 2 | Model file location | **Bundled in binary.** Binary grows from ~10 MB to ~30 MB. Engram works on first install with no download step. |
| 3 | Backfill on missing model state | **Error.** `engram query` exits non-zero when no embeddings exist; user runs `engram embed --all` explicitly. |
| 4 | Auto-embed failure in `engram learn` | **Warn-and-proceed.** Luhmann write is atomic; embed is separate. Failed embed marks the note as missing. |
| 5 | Default `--limit` for query | **20.** Matches `engram recall --recent --limit 20`. |
| 6 | Payload format | **YAML.** No JSON or paths-only flag in spike or v2. |

## Acceptance gate

All 13 UAT cases pass. Case 13 is the embedder go/no-go.

If case 13 fails → swap `embedding_model_id` and model file to
MiniLM-L6-v2; re-run cases 1–12 to verify pipeline still works
with the fallback model. Case 13's failure is recoverable as
long as MiniLM-L6 works (which is verified).

## Out of scope of the spike

These are explicitly deferred to v2 iterations *after* the
spike passes. The spike must not block on any of them.

- **Episodes (F1).** New `engram learn episode` command +
  episode kind. Post-spike v2 iteration.
- **Clustering + hubs (F6 + F9.1).** Subgraph expansion,
  auto-k-means clustering, hub identification. Post-spike v2
  iteration; expands `engram query` payload with `clusters`
  section.
- **MOC migration (F4).** One-time conversion of 25 MOCs into
  facts/feedback with meta-abstraction analysis. Post-spike;
  procedure in `2026-05-24-moc-migration-procedure.md`.
- **F7 full provenance payload.** Spike has `items` with
  `provenances: [direct]` only. Cluster_rep and hub
  provenance values appear when clustering ships post-spike.
- **Kind filter** (`--kind=fact|feedback|episode`) — useful,
  not load-bearing. Defer.
- **Source filter** — defer.
- **Anchor mode** (`--like <path>`) — Smart-Connections-style
  similar-to-existing-note discovery. Defer.
- **JSON / paths-only output formats** — defer.
- **Concurrency / locking on sidecar writes** — defer until
  contention is observed.
- **Block-level / field-level embedding** (F9.2) — defer.
- **BM25 hybrid** — deferred to v3 per F2 discussion.
- **Proactive feedback surfacing** (F8) — dropped entirely.
- **Emergent edges** (F5) — dropped entirely.

## Post-spike v2 work order

Once the spike's 13 UAT cases pass, the following work
proceeds in this order:

1. **MOC migration (F4).** Run the procedure in
   `2026-05-24-moc-migration-procedure.md`. ~50–100 derived
   notes; ~4–8 hours focused work. Sidecars regenerate as part
   of the existing embed pipeline.
2. **Drop `engram learn moc`** from the binary and `/learn`
   SKILL.md. Drop the `MOCs/` directory after archival to
   `_legacy/`.
3. **Episode kind (F1).** Add `engram learn episode` command,
   episode discipline doc (narrative-OK, date-OK), and
   `/learn` SKILL.md section. Auto-embed pipeline already in
   place from spike.
4. **Subgraph clustering (F6 + F9.1).** Add 3-hop link
   expansion, auto-k-means clustering, hub identification.
   Expand `engram query` payload to include `clusters`
   section and richer `items.provenances`.
5. **Updated `/recall` SKILL.md.** Replace cascade logic with
   `engram query` invocation; add synthesis-gate per-cluster
   discipline (write fact/feedback when a binding principle
   emerges).

Each step is independent of the others except where noted
(e.g., step 2 depends on step 1; step 5 depends on step 4).

## What changes for the agent executing this spike

A fresh agent picking this up should:

1. Read this spec end-to-end before any code work.
2. Read the relevant existing engram code:
   - `internal/cli/` (especially `cli.go`, `targets.go`,
     `learn.go`, `recall.go`)
   - `internal/vaultgraph/` (graph primitives)
   - `internal/transcript/` (marker mechanism)
   - `skills/learn/SKILL.md` and `skills/recall/SKILL.md`
3. Pick up the Hugot + GoMLX simplego dependency.
4. Implement against the 13 UAT cases (TDD per project
   convention: `targ test`).
5. Run UAT case 13 (Arctic-xs reference parity) **first** —
   this is the gate. If it fails, swap to MiniLM-L6 before
   investing in pipeline work.
6. Surface anything ambiguous in the spec back to the user
   before guessing.
