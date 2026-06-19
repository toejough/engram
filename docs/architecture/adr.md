# Architecture Decision Records — engram

Retrospective ADRs. The decisions below were mostly made implicitly across the system's
evolution; this log records them so they can be challenged, and ties each to the **verified
defects it produced** (the `⚠ KNOWN` lines), which feed Phase 6 (built-vs-docs) and Phase 7
(fix plan) of the [memory-system rigor effort](../superpowers/specs/2026-06-04-memory-system-rigor.md).

Status legend: **Accepted** · **Accepted (known defect)** — sound decision, buggy as-built ·
**Superseded**. Evidence: commit hashes, `file:line`, and the C4 set
([L1](c1-system-context.md) / [L2](c2-containers.md) / [L3](c3-components.md)) +
[invariants](../superpowers/specs/2026-06-04-memory-invariants.md).

---

## ADR-0001 — Skills + slim binary split

**Status:** Accepted (INV-S1 seam resolved 2026-06 via `engram amend`, `internal/cli/amend.go`)

**Context.** The work divides into LLM judgment (which lessons to capture, how to frame a
`situation`, whether a cluster shares a binding principle) and deterministic compute (cosine,
graph BFS, k-means, marker arithmetic). Mixing them makes the judgment untestable and couples
model behavior to Go code.

**Decision.** Behavior lives in **markdown skills** (C1, executed by the agent in the harness);
deterministic compute lives in a **slim Go binary** (C2). They communicate **only** through C2's
CLI surface (args in, stdout out) and the vault on disk. Each `engram <subcommand>` is a separate
OS process; subcommands never call one another in-process.

**Consequences.** The invariant checker gates C2 (everything in C2 is deterministic and testable);
skills are gated only by RT acceptance tests. INV-S1 (resolved): the skill no longer touches the
vault directly — recall reads via `engram show` / the query payload, and `engram amend`
(`internal/cli/amend.go`) now provides the sync-preserving in-place edit path (rewrites both copies
+ re-embeds), closing the INV-S1 write-half ("no `engram` edit subcommand").

---

## ADR-0002 — Pure-Go embedded model; no external embedding API

**Status:** Accepted (known defect: M4) · supersedes the 2026-05-14 external-Voyage design

**Context.** An early design embedded via an external Voyage API — network dependency, per-call
cost, latency, and sending vault content off-box.

**Decision.** Bundle `all-MiniLM-L6-v2` (384-d) into the binary via `go:embed`; run inference in
**pure Go** through Hugot + GoMLX's `simplego` backend (no CGO). The only external API in the
system is the LLM that runs the agent itself — never embeddings.

**Consequences.** Deterministic, offline, zero per-embed cost; the embedder is a *container of S2*,
not an L1 external. A single `embedding_model_id` is stamped into every sidecar. ⚠ KNOWN (M8):
`loadCompatibleSidecars` (`query.go:803`) silently drops sidecars whose `model_id ≠` the active
model — a model swap silently empties recall unless `engram embed apply --force` re-embeds first.
No guard except the all-empty error path.

---

## ADR-0003 — Embed-on-write with per-note `.vec.json` sidecars

**Status:** Accepted (known defect: E4)

**Context.** Semantic search needs vectors, but a vector DB or a separate rebuild step adds a
moving part that can drift from the notes.

**Decision.** Every `engram learn` writes a sibling `<note>.vec.json` (`vector` + `model_id` +
`content_hash`) as part of the same operation. Note + sidecar creation is serialized under
`flock(.luhmann.lock)` spanning id-compute→write, with `O_EXCL` to prevent clobber.

**Consequences.** No index to maintain or invalidate; sidecars travel with notes (a vault copy is
self-contained, no re-embed). `content_hash` is meant to detect staleness. ⚠ KNOWN (E4):
`ContentHash` hashes the **body**, but episodes embed the **`situation`** frontmatter field — the
two are disjoint, so a `situation` edit leaves the hash unchanged and the stale vector reports
fresh (all L1 notes). ⚠ KNOWN (K1): concurrency correctness rests on the flock spanning the entire
id→write critical section — enforced in code, untested as a property.

---

## ADR-0004 — Three tiers; blended-default retrieval; tier is a frontmatter field

**Status:** Accepted · supersedes the "top-tier-only default" design prose (Decision 3)

**Context.** Memory has three useful grains: raw episodes (L1), specific facts/feedback (L2), and
distilled standards (L3). Retrieval must pick the right grain. Early prose proposed defaulting to
the top tier only; empirically, **blended** retrieval scored better (note 160).

**Decision.** Default retrieval is **blended / kind-agnostic**. `--tier X` is an **optional cap**
that constrains **all exposed channels** (`items`, `clusters[].members`, `nearest_l3`, `hubs`) to
tier X — **operator decision 2026-06-04**, superseding the original items-only design; recall-time
lazy-L2 synthesis runs `engram query --synthesize-l2` **un-tiered**, so it still sees cross-tier
clusters and their `candidate_l2s`. Tier is a **frontmatter field** with
type-derived defaults: fact/feedback → L2 (default, overridable to L3).
There is **no `adr` kind** — an ADR is `type:fact tier:L3`.

**Consequences.** Items-isolation holds today (verified: L1 29/29, L2 11/11, L3 0). ⚠ KNOWN (T1a,
FAIL): until the all-channel fix lands, `--tier` constrains only `items` — `clusters`/`nearest_l3`
still leak other tiers (the channel-misread early in this effort traced to exactly this unnamed-channel
gap; it is also the eval-contamination risk the operator closed by tightening `--tier`). The override
is a feature, so tier↔kind is asymmetric (T2).

---

## ADR-0005 — L3 ADRs are scenario-discoverable, synthesized from L2 clusters by centroid cosine

**Status:** Superseded by the 2026-06-09 lazy-L2 synthesis design ([docs/DESIGN-HISTORY.md](../DESIGN-HISTORY.md)) — L3-ADR-synthesis-at-learn-time is retired; crystallization is now recall-time, agent-judged lazy-L2 (covered/near/absent) via engram amend/learn.

**Context.** An L2 fact only surfaces if you query its keywords — but the agent who needs it does
not know it exists. Standards must be discoverable from the **situation** the agent is in.

**Decision.** §6b: when a `/learn` pass writes L2, seed 3–6 **scenario** situations, run
`engram query` per seed, and for each returned cluster **update** the nearest existing L3 if
centroid cosine ≥ 0.9, else **create** a new L3 (`fact --tier L3`). The loop is **skill-orchestrated**
— there is no `engram synthesize`; the binary only answers separate query/embed/learn calls.

**Consequences.** Standards retrieve by situation, not by lesson-keyword. ⚠ KNOWN: per-pass
write-sparsity starves `AutoK` (silhouette threshold), so clusters rarely form at write time —
the rebuilt vault has only 1 L3 from 106 L2. ⚠ KNOWN (INV-S2): §6b "revise its `situation`" assumes
tuning the situation changes retrieval, but a **fact** stores the situation twice (frontmatter +
the body "formula") and only the body is embedded/hashed — a frontmatter-only edit is a retrieval
no-op and invisible to `embed apply --stale`. Superseded: per-pass write-time synthesis is replaced
by recall-time lazy-L2 — the write-sparsity that starved AutoK no longer applies, and INV-S2's
frontmatter/body desync is resolved by `engram amend` (which rewrites both copies + re-embeds).

---

## ADR-0006 — Embed source by kind: episodes embed `situation`, others embed body

**Status:** Superseded — episode type retired (`engram learn episode` removed in 2026-06-19 cleanup); `embed.Text` now embeds body for all note types. E4/E5 defects are resolved by retirement. The `situation` field is still authored in fact/feedback bodies but is no longer a routing key for embedding.

**Context.** Episodes are retrieved by **situation** (the task you were doing — the recall-mirror);
facts/feedback are retrieved by their content.

**Decision.** `embed.Text` routes `type:episode` → the `situation` frontmatter field; every other
kind → the body (`hash.go:48-72`).

**Consequences.** Episodes match task-shaped queries the way recall phrases them. ⚠ KNOWN (E4):
the staleness hash covers the body, not the embedded `situation`, for episodes (see ADR-0003).
⚠ KNOWN (E5): an empty `situation` silently falls back to the body, self-violating the routing.
⚠ KNOWN (M5, FAIL): fact/feedback retrieval also leans on `situation` (it is rendered into the body
formula and feeds recall-mirror), yet the CLI marks `situation` `required` only for episodes — an
empty fact/feedback situation is unguarded (census-clean 107/107). This is the FAIL-class
situation-presence invariant's architectural home.

---

## ADR-0007 — The wikilink graph is authored and walked by the binary; dangling links dropped

**Status:** Accepted (known defect: G0/G5)

**Context.** Navigation should live in **authored relations** (wikilinks in note bodies), not a
separate graph store that can drift. Recall expands a subgraph from direct hits to find clusters
and hubs.

**Decision.** `vaultgraph.ScanVault` parses wikilinks at scan time; `BuildGraph` builds a directed
graph **keyed by basename**; recall does a 3-hop BFS (cap 200) + in-degree top-5 hubs. Dangling
targets are silently dropped at build.

**Consequences.** The graph is derived and always fresh. ⚠ KNOWN (G0): `learn` writes relations as
**bare Luhmann ids** (`[[105]]`) but `BuildGraph` resolves by **basename** — 155/183 link-instances
unresolved (151 of them bare-id), 138/171 notes orphaned, mean out-degree 0.16, so recall's graph expansion runs on a
near-empty graph. ⚠ KNOWN (G5): verbatim `[[x]]` strings inside episode transcript bodies become
false edges (no episode special-casing at scan).

---

## ADR-0008 — Per-arc episodes as the L1 evidence layer

**Status:** Superseded — episode type retired; `engram learn episode` and `engram transcript` removed in 2026-06-19 cleanup. Chunks ingested via `engram ingest --auto` are now the L1 evidence layer, referenced from facts/feedback via `--chunk-source`. · commits 98c962ea, b4e24f76, 4901bf78

**Context.** "What did we do yesterday" needs the literal interactions — tool calls, file paths,
the back-and-forth — not a narrative summary. A session interleaves multiple arcs of work.

**Decision.** Write **one episode per work-ARC** (a coherent thread; may be non-contiguous and may
overlap other arcs). The body is the noise-filtered transcript chunk, assembled from one or more
**repeatable** `--from-transcript-range` spans. Facts/feedback derived from a chunk link back via
`--relation "<episode-luhmann>|extracted from this chunk"`. Provenance stores the **resolved**
transcript file path (cwd-independent).

**Consequences.** High-fidelity recall of prior sessions; avoids both failure modes (one giant
session-spanning episode; losing the interactions). Episodes bypass the fact/feedback machinery
(no locus classification, no recall-mirror test) — they are retrieved through the situational
stream, not phrase-matching.

---

## ADR-0009 — Marker forward-progress: strict-greater, intra-session split, multi-source independent

**Status:** Superseded — `engram transcript --mark` and the `learnmarker` package retired in 2026-06-19 cleanup; marker logic subsumed into `engram ingest --auto`. M2-segments defect retired with the `--segments` path. · commits 4901bf78, 5c16c784

**Context.** `engram transcript --mark` must visit every learnable row **exactly once** across
runs — never skip, never re-emit forever — across multiple harness sources (Claude `.jsonl`,
OpenCode SQLite) and even within a single oversized session.

**Decision.** A per-`(project, source)` RFC3339 marker. Scan **strictly after** the marker within
a byte budget; on mid-session truncation (Partial) advance to the **last included row's**
timestamp, else to the session Mtime; **never advance past the earliest row not read** this run;
sources advance independently.

**Consequences.** Resumable, multi-source-safe forward progress; intra-session splitting lets an
oversized session be consumed across runs. ⚠ KNOWN (M2-segments): the `emitSegments` path
(`engram transcript --segments`) advances the marker to Mtime **unconditionally** — `SegmentsFrom`
carries no `Partial` flag — so it over-advances on truncation. Latent today (the skill runs
`--segments` without `--mark`) but real.

---

## ADR-0010 — Sessions are read behind reader/finder interfaces; a composite dispatches across backends

**Status:** Accepted · `internal/transcript/opencode.go`; previously wired via `newTranscriptDeps` (removed in 2026-06-19 cleanup; wiring now lives in `engram ingest`)

**Context.** Engram must read session transcripts from more than one harness — Claude Code stores
them as per-session `.jsonl` files; OpenCode stores them in a SQLite database. The marker,
byte-budget, noise-strip, and emit logic must not care which backend a session came from.

**Decision.** Define `Finder` (locate sessions) and `Reader` / `SegmentsReader` (read rows / arc
segments) interfaces. Provide two backends — `JSONLReader` + `SessionFinder` (Claude) and
`OpencodeTranscriptReader` + `OpencodeSessionFinder` (OpenCode SQLite) — plus a
`CompositeSessionFinder` / `CompositeTranscriptReader` that wrap a list and dispatch to the **first
backend that succeeds** (`opencode.go`, first-success dispatch). The CLI wires the composite over both backends in
`newTranscriptDeps` (`internal/cli/cli.go`); `SegmentsFrom` dispatches only to readers that implement `SegmentsReader`.

**Consequences.** Marker forward-progress (ADR-0009), stripping, and emit are backend-agnostic —
they run on the composite, never on a concrete backend. Session-id **scheme** dispatch (bare UUID →
Claude `.jsonl`; `opencode://…` → SQLite) is part of the same seam. Adding a third harness is an
interface implementation, not a change to the read pipeline.

## Decisions deliberately NOT made into ADRs

- **"Curate, don't regenerate" → full rebuild** (B10): a reversed operational decision, not an
  architectural one — recorded as a dated reversal in Phase 0, not an ADR.
- **Capture abstraction = generic-actionable** (B2): a *skill-authoring* convention (how to phrase
  a note), gated by RT/eval, not a C2 architecture decision.
