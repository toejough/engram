# Memory-System Rigor & Recovery Plan

Date: 2026-06-04. Owner: this effort exists because the memory system shipped
structural breaks that went uncaught for a long time, and the testing cycle
never validated the system's own internals — only downstream build conformance.
This document is the durable plan + the trust mechanism. Do not drop a phase.

## 0. The trust break (what went wrong, honestly)

Confirmed broken (survived the Phase-0 antagonist round):
- **L1↔L2 linking** (binary-real graph, per the G0 finding): **87/106** L2 facts/feedback cite no
  *resolved* L1 episode; **138/171** notes are fully isolated (0 in + 0 out). My first count (28
  orphaned, 23 isolated) was ID-aware like Obsidian and overstated the binary's real graph ~6×;
  the G0 link-form bug is why. Canonical census: [memory-invariants](2026-06-04-memory-invariants.md) KEYSTONE.
- **L2→L3 synthesis**: 1 real `tier: L3` ADR from 106 L2 facts; per-pass
  write-sparsity starves AutoK so clusters rarely form at write time.
- **INV-E4 episode freshness hash** (code-verified): situation edits invisible to the
  staleness check across all 64 L1 notes.
- **M2-segments path marker over-advance** (NEW, antagonist-found, code-verified):
  `emitSegments` advances the marker to file Mtime even when `SegmentsFrom` truncated at
  the byte budget (no `Partial` on the segments path) — violates 5c16c784's own invariant
  on the `--segments` path.
- **D7 no automated invariant tripwire** (no vault `check` subcommand).

**RETRACTED — `--tier` filter is NOT broken.** My earlier "`--tier L3` returns 1 L3 + 22
L2 + 29 L1; not tier-isolating" was a **channel-misread**: I counted `clusters[].members`
(intentionally tier-agnostic) as `items`. Verified three ways (arithmetic, live binary
`items: []`, frontmatter: L1 29/29, L2 11/11, L3 0): `items[]` is cleanly tier-isolated.
This was exactly the failure mode this effort exists to prevent, and the per-phase
adversary caught it — which is the process working. See INV-T1a/b/c.

**Eval validity — OPEN, not settled (antagonist O-2):** since `items` IS isolated, the
eval's tier-regime cells were not contaminated via items. They could still be contaminated
via the **cluster-members channel** IF recall fed the whole query payload (all tiers) to
the building agent instead of items-only. So "tier-regime results are suspect" is an **open
question pending a check of what recall actually consumes** — neither invalidated (my wrong
mechanism) nor rescued. Phase 6 resolves it.

### Added by the Phase-0 antagonist (missed contracts → new invariants)
- **INV-K1 (vault write-lock):** concurrent `engram learn` never computes the same next
  Luhmann id and never overwrites a note (flock on `.luhmann.lock` spans id→write; `O_EXCL`
  backstops). `[enforced in code; untested as a property]`
- **INV-U1 (`update` idempotence):** `engram update` re-run with identical source is a
  copy-equivalent no-op; missing-go / no-harness / missing-skills fail with sentinels.
  `[uncaptured surface — internal/update]`
- **INV-M2 scope:** holds for `emitTranscripts`; **unenforced for `emitSegments`** (M2-segments).

Needs-check (status unknown — Phase 6 will resolve):
- dangling wikilinks; stale embeddings (body changed, sidecar didn't);
  episode-provenance validity; near-duplicate notes; whether recall's
  graph-expansion actually traverses links; whether rebuilt facts are *good*.

**Discovered during Phase 3 (sequence-diagram grounding) — verify in Phase 6:**
- **INV-S1 — the skill layer touches the vault DIRECTLY in two spots**, contradicting c2's
  "skill never touches the vault directly." (write) learn §6b situation-revision — there is no
  `engram` note-edit subcommand (`learn` is create-only via `O_EXCL`, `cli.go:191`), so revising
  a note's `situation` before re-embed is the agent editing the file with its own tools (C1→C4
  write). (read) recall §3a — the synthesis subagent **reads** cluster-member files directly,
  because the query payload returns `clusters[].members` as *paths only*. Either add `engram`
  read/edit paths, or soften the c2 claim (softened in c2 already).
- **INV-S2 — facts carry `situation` in TWO places, only one embedded.** `renderFactBody`
  (`learn.go:568`) renders `stripLeadingWhen(situation)` into the body, AND
  `renderFactFrontmatter` writes a `situation:` field. For facts `embed.Text` returns the *body*
  (E3) and `ContentHash`/`embed apply --stale` key on the *body* (`hash.go`). So a §6b revision
  that edits only the frontmatter `situation` is embed-irrelevant AND invisible to `--stale`;
  only a body-formula edit changes retrieval. Same write≠read family as E4/G0 (duplicated
  representation that can silently diverge).

**Root cause of the miss:** trusted `grep`/agent self-reports over reading
frontmatter and running the binary; and **no invariants were defined**, so there
was no automated tripwire. The remedy is below: invariants become executable
checks, an adversary reviews the design, and the user gates the build.

## Phase 6 — empirical confirmation (live 171-note vault, 2026-06-04)

Every documented defect was re-confirmed by **running checks against the REAL vault**, not asserted:
- **G0:** 183 wikilink-instances → 151 bare-id + 28 basename-form + 4 dangling; **28 resolve,
  138/171 orphaned (80%), mean out-degree 0.16.** Matches the canonical census exactly.
- **T1a leak (live):** `engram query --tier L3` → `items[]` = {} (items isolation holds) but
  `clusters[].members` = **37 L2 + 7 L1 + 5 L3** — 44 non-L3 leaked. This is the eval-contamination
  the operator decision closes, and the source of the original channel-misread.
- **M4:** all 171 sidecars `minilm-l6-v2@384` (homogeneous; swap-empties-recall risk latent + unguarded).
- **M5:** 0 of 107 fact/feedback missing `situation` (clean, unguarded).
- **E4/E5:** 64/64 episodes carry a non-empty `situation` (the embed source) while `ContentHash`
  covers the body — disjoint, as documented.
- **G5:** 22 episodes carry 34 `[[…]]` instances in their bodies, all parsed as edges with no
  authored-vs-verbatim isolation.
- **Code-verified (not vault-census-able):** M2-segments, INV-S1, INV-S2, K1, M6/M7/M8.

**Finding: the documents faithfully describe the built system.** No doc claim was contradicted by
the live vault — the gap is real defects, not doc drift. Fix plan: `docs/superpowers/plans/2026-06-04-memory-fixes.md`.

## 1. Property-based invariants (the contract the system MUST satisfy)

Each is phrased to be **testable** — it becomes either a vault-invariant checker
(a `targ` gate over the real vault) or a binary property test (rapid). `[status]`
is the current best understanding; Phase 6 confirms each with evidence.

### Graph / structural integrity
- **INV-G1 (tier ladder, downward):** every L3 ADR links to ≥1 L2; every L2
  fact/feedback links to ≥1 L1 episode OR is explicitly flagged a pure synthesis
  with no single source chunk. `[BROKEN: 87/106 cite no resolved L1; 0/106 under an ADR — binary-real per G0]`
- **INV-G2 (no orphans):** no note has 0 in-links AND 0 out-links. `[BROKEN: 138/171 — binary-real per G0; earlier "23" was ID-aware]`
- **INV-G3 (link validity):** every wikilink target resolves to an existing note;
  no dangling references. `[~OK: 1 real dangling target (15a); rest were regex artifacts]`
- **INV-G4 (provenance link):** an L2 extracted from an episode chunk links to that
  episode; an L3 links to every constituent L2. `[partially broken]`

### Tier / retrieval correctness
- **INV-T1a (items isolation):** `query --tier X` returns only tier-X notes in `items[]`.
  `[VERIFIED OK — antagonist F-1: --tier L1 → 29/29 L1, --tier L2 → 11/11 L2, --tier L3 → 0
  items. My earlier "BROKEN" was a CHANNEL-MISREAD — I counted clusters[].members (38 L1 + 37
  L2 + 1 L3) and called them items. The filter is sound; the skill's own §6b self-verify
  ("ADR must appear in items") depends on it working.]`
- **INV-T1b (cluster channel is intentionally tier-agnostic):** `clusters[].members` and
  `nearest_l3` are NOT tier-filtered — by design they drive synthesis (§6b cosine match), not
  retrieval. Correct behavior, but the spec never named the **three channels** (items /
  cluster-members / down-links), which is *why* it was misread. `[design gap, not a bug — must
  be documented; **SUPERSEDED 2026-06-04** — `--tier` now constrains ALL channels (T1a), and the
  old items-only behavior is the cross-tier leak to fix]`
- **INV-T1c (nearest_l3 survives the filter):** `--tier` must NOT suppress `nearest_l3` (§6b
  update-or-create depends on it under `--tier L3`). `[RETIRED 2026-06-04 — superseded by all-channel
  T1a; §6b relies on its un-tiered query for cross-tier synthesis, not on nearest_l3 surviving --tier]`
- **INV-T2 (tier↔kind agreement, ASYMMETRIC — re-phrased per Phase 0):** episodes must
  be L1 (rigid); fact/feedback may be L2 **or** L3 (the `--tier` override is a feature);
  **no note is L1 unless it is an episode.** `[OK: 0 untagged, 0 violations over 171;
  64 L1 / 106 L2 / 1 L3]`
- ~~INV-T3 (top-tier default)~~ **DROPPED** — it encoded the *abandoned* "top-tier-only"
  design prose; the shipped + intended contract is **blended/kind-agnostic default**
  (L3 design Decision 3). The real invariant is folded into INV-T1 (tier-isolation only
  when `--tier` is passed).

### Embedding integrity
- **INV-E1 (presence):** every note has a sibling `.vec.json` sidecar. `[OK: 171/171, 0 missing]`
- **INV-E2 (freshness):** each sidecar's stored hash matches the note's current
  embed-source; no stale vectors. `[unchecked]`
- **INV-E3 (embed source by kind):** episodes embed `situation`; facts/feedback/ADRs
  embed body. `[verified in code: routes correctly in embed.Text]`
- **INV-E4 (freshness-hash ⊇ embed-source):** the staleness hash must cover whatever is
  actually embedded. `[BROKEN, verified hash.go: ContentHash hashes body, episodes embed
  situation (frontmatter) — disjoint; situation edits invisible to staleness for all 64 L1]`
- **INV-E5 (episode situation non-empty):** an episode's `situation` must be non-empty
  (else embed.Text silently falls back to body, self-violating INV-E3). `[verified gap hash.go:67-69]`

### Provenance / evidence
- **INV-P1 (episode provenance valid):** every episode names ≥1 real session id whose
  transcript source exists, with a parseable range. `[spot-checked OK in rebuild]`

### Marker / learn forward-progress (the bug just fixed — needs a permanent property test)
- **INV-M1 (never skip):** scanning [from,now] visits every learnable row exactly
  once across runs — no row skipped, none re-emitted forever. `[FIXED 5c16c784; no property test yet]`
- **INV-M2 (never past unread):** a source's marker never advances past the earliest
  row not actually read this run. `[FIXED; unit-tested; promote to property test]`
- **INV-M3 (multi-source independence):** one source filling the byte budget never
  advances another source's marker. `[FIXED; unit-tested]`

### Recall ↔ learn duality
- **INV-R1 (recall-mirror):** a note written with `situation` S is retrievable by a
  query phrased as S (learn and recall are inverse over the situation field). `[untested as a property]`
- **INV-R2 (graph expansion really traverses links):** recall's subgraph/cluster/hub
  computation actually uses the wikilink graph, and degrades gracefully on a sparse
  graph. `[unchecked]`

### Clustering / synthesis determinism (added Phase 0)
- **INV-C1 (clustering determinism):** k-means + silhouette + AutoK return the same
  result given a fixed vault + phrase (recall reproducibility). `[unchecked]`
- **INV-L3-1 (L3 match stability):** a cluster whose centroid cosine ≥0.9 to an existing
  L3 must *update* that L3, never spawn a near-duplicate; the boundary is stable. `[unchecked]`

## 2. Deliverables & phases (do not reorder past a STOP)

**Standing rule — per-phase adversarial loop.** EVERY phase below ends with an
antagonistic review: I author the artifact, then dispatch a fresh subagent whose
job is to *attack* it (find gaps, wrong abstractions, missing invariants,
unjustified decisions, divergence from evidence). We iterate — I defend or revise,
it re-attacks — **until we genuinely agree**, not until I'm tired of it. Only then
does the phase's output stand and the next phase begins. The two STOP-and-talk
checkpoints are *in addition to* the per-phase loops, at the points the user set.

| # | Phase | Output | Per-phase gate |
|---|---|---|---|
| 0 | Review last-5-days transcripts | intended-contracts + decisions notes | antagonist → agree |
| 1 | Property-based invariants | this §1, refined | antagonist → agree |
| 2 | C4 diagrams L1/L2/L3 | context / container / component | antagonist → agree |
| 3 | Sequence + flowcharts per C4 level | per use case (learn, recall, L3-synth, transcript/marker), reusing C4 blocks | antagonist → agree |
| 4 | ADRs | the real architectural decisions | antagonist → agree |
| 5 | Holistic adversarial review of the integrated design | challenged + reconciled doc set | antagonist → agree, then **STOP & talk (checkpoint 1)** |
| 6 | Built-vs-docs evaluation | evidence-based gap list (run the invariant checks) | antagonist → agree |
| 7 | Fix plan | prioritized TDD plan | antagonist → agree, then **STOP & talk (checkpoint 2)** |
| 8 | Implement fixes (TDD, targ) | code + invariant checker as targ gate | antagonist → agree (per fix, via spec+quality review) |

## 3. Added testing rigor (so this can't recur)
- **Executable vault-invariant checker** over the real vault (graph + tier + embedding
  + provenance invariants), wired as a `targ` gate so a violation fails CI.
- **Binary property tests** (rapid) for marker forward-progress (INV-M*) and tier-filter
  soundness (INV-T1).
- **Verify-don't-assert standing rule:** read frontmatter (never `grep` body for tier),
  run the binary and assert on behavior, inspect the graph — before reporting a result.
- **Adversarial review before building** on any new design.
- **Re-validate the eval** only after INV-T1 is fixed (tier isolation must be real
  before any tier-regime comparison means anything).

## Current status
**Phases 0–2 COMPLETE**, each closed by an antagonist round that caught a critical error —
which is the process working:
- Phase 0: `--tier` was **not** broken (my channel-misread: counted `clusters[].members` as `items`).
- Phase 1: the binary's real graph is ~6× emptier than I first measured (the G0 link-form bug).
- Phase 2: a fabricated in-process `query→learn` edge, then a *recurrence* nesting `embed` inside
  the learn process box with a cross-process `query→embed` arrow. Both found and fixed.

C4 set authored and agreed: [L1](../../architecture/c1-system-context.md) /
[L2](../../architecture/c2-containers.md) / [L3](../../architecture/c3-components.md).

**Phases 0–3 COMPLETE.** Phase 3 antagonist caught 4 component-attribution errors in the L3
sequence diagrams (e.g. wikilink reads drawn inside vaultgraph that actually happen at scan time;
`nextLuhmannID` drawn as the K10 kernel when it lives in `cli/luhmann.go`) — all fixed, then a
round-2 antagonist agreed. Sequence + flow diagrams now exist at L1 (c1), L2 (c2 skills↔binary
boundary), and L3 (c3 component internals), for recall / learn / §6b L3-synthesis / marker. Two
new findings surfaced and recorded for Phase 6 (INV-S1 skill→vault direct access; INV-S2 duplicated
fact `situation`). **Phase 4 (ADRs) COMPLETE** — [10 ADRs](../../architecture/adr.md); the antagonist
added the dual-source-reader ADR (ADR-0010) and got INV-S1/S2 promoted into the canonical invariants.
**Phases 0–5 COMPLETE — AT CHECKPOINT 1 (stop & talk).** The Phase-5 holistic antagonist found
2 Critical cross-doc ID collisions (I had reused M3/M4 for both marker invariants and the Phase-1
additions, making the checker's severity list ambiguous) + 2 Major (stale ID-aware census in §0/§1;
an orphaned FAIL invariant) + 2 minor; all reconciled (renumbered the Phase-1 additions to a
contiguous M4–M8, segments bug standardized as M2-segments, census made binary-real) and a round-2
antagonist verified internal consistency. Per the checkpoint-1 decisions, `--tier` was then tightened
to expose only the asked-for level across ALL channels (T1a; the eval-contamination fix). Recall `--tier L3` edit remains PAUSED —
superseded by this effort. The `--tier` filter works; what's missing is a *populated* L3 layer,
which Phase 8 addresses via the G0 link-form + §6b synthesis fixes.
