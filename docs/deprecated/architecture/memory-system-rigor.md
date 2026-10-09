# Memory-System Rigor & Recovery Plan

> **Reconciled 2026-06-20:** retired invariants marked inline below; survivors restated as-is. The L1/L2/L3 tier surface, episode kind, `--tier` flag, `nearest_l3`/`hubs` fields, and OpenCode transcript backend were removed in the recall-v2 / 2026-06-20 deep clean.

Date: 2026-06-04. Owner: this effort exists because the memory system shipped
structural breaks that went uncaught for a long time, and the testing cycle
never validated the system's own internals — only downstream build conformance.
This document is the durable plan + the trust mechanism. Do not drop a phase.

## 0. The trust break (what went wrong, honestly)

Confirmed broken (survived the Phase-0 antagonist round):
- **L1↔L2 linking** **[RETIRED — L1 episode kind removed; `--chunk-source` frontmatter provenance is now the L1 evidence link, not wikilinks to episode notes.]** ~~87/106 L2 facts/feedback cite no resolved L1 episode.~~ The G0 link-form bug (bare-id vs basename) still affects `BuildGraph` and is relevant to `check`/`amend`. Canonical census: [memory-invariants](memory-invariants.md) KEYSTONE.
- **L2→L3 synthesis** **[RETIRED — L3 note kind and `tier: L3` removed in recall-v2.]** ~~1 real `tier: L3` ADR from 106 L2 facts.~~
- **INV-E4 episode freshness hash** **[RETIRED — episode kind removed; no disjoint situation-vs-body hash issue for fact/feedback (both embed and hash body).]** ~~situation edits invisible to staleness check across all 64 L1 notes.~~
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
adversary caught it — which is the process working. **[RETIRED — `--tier` flag removed in recall-v2; INV-T1a/b/c are all retired. This historical finding remains for audit purposes.]**

**Eval validity — OPEN, not settled (antagonist O-2):** **[RETIRED — the tier-regime eval and the `--tier L3` query it relied on were removed in recall-v2. The open question is moot.]** ~~tier-regime cells contamination via cluster-members channel — neither invalidated nor rescued. Phase 6 resolves it.~~

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
  near-duplicate notes; whether rebuilt facts are *good*.
- **[RETIRED from needs-check: episode-provenance validity (episode kind removed); whether recall's graph-expansion actually traverses links (subgraph/hub path removed).]**

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
- **T1a leak (live):** **[RETIRED — `--tier` flag and `nearest_l3`/`hubs` removed in recall-v2; the tier-isolation issue is moot.]** ~~`engram query --tier L3` → `items[]` = {} but `clusters[].members` = 37 L2 + 7 L1 + 5 L3 — 44 non-L3 leaked.~~
- **M4:** all 171 sidecars `minilm-l6-v2@384` (homogeneous; swap-empties-recall risk latent + unguarded). — **SURVIVOR**
- **M5:** 0 of 107 fact/feedback missing `situation` (clean, unguarded). — **SURVIVOR**
- **E4/E5:** **[RETIRED — episode kind removed; no disjoint situation/body issue for fact/feedback.]** ~~64/64 episodes carry a non-empty `situation` while `ContentHash` covers the body — disjoint.~~
- **G5:** **[RETIRED — episode kind removed; `[[x]]` in chunk bodies are no longer parsed as vault edges.]** ~~22 episodes carry 34 `[[…]]` instances in their bodies, all parsed as edges.~~
- **Code-verified (not vault-census-able):** M2-segments, INV-S1, INV-S2, K1, M6/M7/M8.

**Finding: the documents faithfully describe the built system.** No doc claim was contradicted by
the live vault — the gap is real defects, not doc drift. Fix plan: the 2026-06-04 memory-fixes plan
(`DESIGN-HISTORY.md` deleted 2026-07; git log recovers §6's narrative).

## 1. Property-based invariants (the contract the system MUST satisfy)

Each is phrased to be **testable** — it becomes either a vault-invariant checker
(a `targ` gate over the real vault) or a binary property test (rapid). `[status]`
is the current best understanding; Phase 6 confirms each with evidence.

### Graph / structural integrity
- **INV-G1 (tier ladder, downward):** **[RETIRED — L1/L2/L3 tier structure and episode kind removed in recall-v2.]** ~~every L3 ADR links to ≥1 L2; every L2 links to ≥1 L1 episode.~~
- **INV-G2 (no orphans):** no note has 0 in-links AND 0 out-links. `[BROKEN: 138/171 — binary-real per G0; earlier "23" was ID-aware]` — **SURVIVOR**
- **INV-G3 (link validity):** every wikilink target resolves to an existing note;
  no dangling references. `[~OK: 1 real dangling target (15a); rest were regex artifacts]` — **SURVIVOR**
- **INV-G4 (provenance link):** **[RETIRED — L2/episode wikilink provenance removed; chunk-source is now frontmatter, not wikilinks.]** ~~an L2 extracted from an episode chunk links to that episode; an L3 links to every constituent L2.~~

### Tier / retrieval correctness
**[RETIRED — entire tier/retrieval-correctness section removed. `--tier` flag, `nearest_l3`, `hubs`, L1/L2/L3 kinds, and episode kind were all removed in recall-v2. The historical findings below are preserved for audit only.]**

- **INV-T1a** **[RETIRED]** ~~`query --tier X` returns only tier-X notes in `items[]`.~~ (Was VERIFIED OK before removal.)
- **INV-T1b** **[RETIRED]** ~~cluster channel is intentionally tier-agnostic.~~
- **INV-T1c** **[RETIRED]** ~~`nearest_l3` survives the tier filter.~~
- **INV-T2** **[RETIRED]** ~~tier↔kind asymmetric: episodes must be L1; fact/feedback may be L2 or L3.~~
- **INV-T3** **[DROPPED/RETIRED]** ~~top-tier default.~~

### Embedding integrity
- **INV-E1 (presence):** every note has a sibling `.vec.json` sidecar. `[OK: 171/171, 0 missing]`
- **INV-E2 (freshness):** each sidecar's stored hash matches the note's current
  embed-source; no stale vectors. `[unchecked]`
- **INV-E3** **[RETIRED — episode kind removed; all notes embed body. `embed.Text` still exists but the episode-specific `situation` branch is dead code.]** ~~episodes embed `situation`; facts/feedback/ADRs embed body.~~
- **INV-E4** **[RETIRED — episode kind and the situation/body disjoint hash problem removed with it. For fact/feedback, body is both embedded and hashed.]** ~~freshness hash ⊇ embed source.~~
- **INV-E5** **[RETIRED — episode kind removed; no episode `situation` field.]** ~~episode situation non-empty.~~

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
  query phrased as S (learn and recall are inverse over the situation field). `[untested as a property]` — **SURVIVOR**
- **INV-R2 (graph expansion really traverses links):** **[RETIRED — recall no longer does subgraph/hub expansion; `vaultgraph` is used only by `check`/`amend`, not in the query path.]** ~~recall's subgraph/cluster/hub computation actually uses the wikilink graph.~~

### Clustering / synthesis determinism (added Phase 0)
- **INV-C1 (clustering determinism):** k-means + silhouette + AutoK return the same
  result given a fixed vault + phrase (recall reproducibility). `[unchecked]` — **SURVIVOR**
- **INV-L3-1** **[RETIRED — L3 note kind removed; `candidate_l2s` within-cluster nomination replaced the centroid-cosine L3 match-stability gate.]** ~~a cluster whose centroid cosine ≥0.9 to an existing L3 must update that L3, never spawn a near-duplicate.~~

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

C4 set authored and agreed: [L1](c1-system-context.md) /
[L2](c2-containers.md) / [L3](c3-components.md).

**Phases 0–3 COMPLETE.** Phase 3 antagonist caught 4 component-attribution errors in the L3
sequence diagrams (e.g. wikilink reads drawn inside vaultgraph that actually happen at scan time;
`nextLuhmannID` drawn as the K10 kernel when it lives in `cli/luhmann.go`) — all fixed, then a
round-2 antagonist agreed. Sequence + flow diagrams now exist at L1 (c1), L2 (c2 skills↔binary
boundary), and L3 (c3 component internals), for recall / learn / **[RETIRED: §6b L3-synthesis — L3 kind removed]** / marker. Two
new findings surfaced and recorded for Phase 6 (INV-S1 skill→vault direct access; INV-S2 duplicated
fact `situation`). **Phase 4 (ADRs) COMPLETE** — [10 ADRs](adr.md); the antagonist
added the dual-source-reader ADR (ADR-0010) and got INV-S1/S2 promoted into the canonical invariants.
**Phases 0–5 COMPLETE — AT CHECKPOINT 1 (stop & talk).** The Phase-5 holistic antagonist found
2 Critical cross-doc ID collisions (I had reused M3/M4 for both marker invariants and the Phase-1
additions, making the checker's severity list ambiguous) + 2 Major (stale ID-aware census in §0/§1;
an orphaned FAIL invariant) + 2 minor; all reconciled (renumbered the Phase-1 additions to a
contiguous M4–M8, segments bug standardized as M2-segments, census made binary-real) and a round-2
antagonist verified internal consistency. **[RETIRED from current-status: `--tier` tightening (T1a) and `--tier L3` edit — both moot; `--tier` flag removed in recall-v2. Phase 8 G0/link-form fixes remain relevant for `engram check`/`amend`.]**
