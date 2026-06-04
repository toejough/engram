# Engram Memory-System Invariants (canonical, testable)

Date: 2026-06-04. The contract the memory system MUST satisfy, each phrased as a
**testable property** with an explicit **test method** and **current status**. This is
the spec the executable checker (Phase 8) implements. Statuses reflect Phase-0 evidence
(antagonist-reconciled). Phase 1 deliverable; **pending its own antagonist round**.

Test methods:
- **VC** = vault checker (a new `engram check` over the on-disk vault — graph/tier/embed/provenance).
- **PT** = binary property test (rapid; deterministic over synthetic inputs).
- **IT** = integration test (real I/O wiring, e.g. `update`).
- **RT** = runtime assertion / acceptance (e.g. the skill's own §6b self-verify).

## A. Graph / structural integrity — `[VC]`

> **KEYSTONE (Phase-1 antagonist, verified): the write-form ≠ resolve-form bug.**
> `learn` writes relations as **bare Luhmann IDs** (`[[105]]` — per skill §6a/§6b) but
> `vaultgraph.BuildGraph` resolves edges **only by full basename** (`graph.go:64-78`), and
> recall builds its subgraph through that same resolver (`query.go:557`). Verified counts:
> **Of 183 authored link-instances, 28 resolve to graph edges and 155 drop: 151 are bare-id
> (`[[105]]`), unresolvable by the basename resolver, plus 4 dangle to nonexistent notes; the other
> 28 are basename-form and all resolve. Result: 138/171 notes orphaned (80%), mean out-degree 0.16.
> (Empirically re-confirmed against the live 171-note vault on 2026-06-04: 151 + 28 + 4 = 183.)**
> My earlier graph numbers (23
> orphans, ~1.1 links/note) were **ID-aware** — I resolved `[[105]]`→leading-id `105` by hand.
> **Neither the binary NOR Obsidian does that** (both resolve only full basenames/aliases — verified
> against Obsidian's docs 2026-06-04: no prefix matching). That count was a *hypothetical post-fix*
> graph, not any tool's real behavior, and overstated the binary's real graph ~6×. **Recall's graph-expansion (subgraph/clusters/hubs)
> therefore runs on a near-empty graph — INV-R2 is effectively FAILING, not "unchecked".**
> Same bug-class as INV-E4 (writer and reader keyed on disjoint representations). Fixable:
> normalize bare-id→basename in the resolver, or write basenames.

| id | invariant (testable property) | sev | status (BINARY's real graph) |
|---|---|---|---|
| **G0 (link-form normalization, root cause)** | every authored `[[target]]` resolves through `BuildGraph` to a node (target is a basename, or the resolver normalizes id→basename). | **FAIL** | **BROKEN** — 155/183 edges dropped |
| **G1** | Tier ladder downward: every L3 links ≥1 L2; every L2 links ≥1 L1 (the `synthesis:true` escape clause is **dropped — no such field exists** in the writer; add it or omit the carve-out). | WARN | **BROKEN** — 87/106 L2 cite no resolved L1; **0/106** under an ADR |
| **G2** | No orphans: no note has in-degree 0 AND out-degree 0. (WARN — a legitimately standalone principle is allowed; this is corpus health, not correctness.) | WARN | **BROKEN** — 138 (mostly masked by G0) |
| **G3** | No dangling wikilinks; the checker surfaces them. (The binary silently drops dangling at build — `graph.go:78` — so a dangling link is a lost intended edge.) | WARN | **BROKEN** — 155 edge-instances on the real graph |
| **G4** | Provenance edges present: L2→its episode; L3→each constituent L2. | WARN | partially BROKEN (subsumed by G0) |
| **G5 (episode-body wikilink isolation)** | graph edges come from authored relation context, NOT from `[[x]]` strings embedded in episode verbatim transcript bodies (else tool-output/prose like `[[target]]` manufactures false edges). | FAIL | gap — binary parses whole body (`scanner.go`) |

## B. Tier / retrieval correctness — `[PT + VC]`
| id | invariant | status |
|---|---|---|
| **T1a (tier isolation — ALL channels)** | `query --tier X` exposes ONLY tier-X notes — across `items[]`, `clusters[].members`, `nearest_l3`, and `hubs`. (Operator decision 2026-06-04: the tier flag constrains everything it exposes, not just items.) | items **OK**; `clusters`/`nearest_l3` currently NOT filtered → the cross-tier leak. **FIX (FAIL).** |
| **T1b (blended default preserved)** | absent `--tier`, every channel is blended/kind-agnostic (the normal recall path). §6b synthesis issues **un-tiered** queries, so tightening `--tier` does not starve cluster/`nearest_l3` update-or-create. | OK |
| **T1c (retired)** | ~~`--tier` must not suppress `nearest_l3`~~ — superseded by T1a: `--tier X` now constrains `nearest_l3` too; §6b relies on its un-tiered query for cross-tier synthesis. | superseded |
| **T2** | Tier↔kind (asymmetric): episodes MUST be L1 (rigid); fact/feedback MAY be L2 or L3 (`--tier` override is a feature); no note is L1 unless it is an episode. | **OK** — 0 violations / 171 |

## C. Embedding integrity — `[VC]`
| id | invariant | status |
|---|---|---|
| **E1** | Presence: every note has a sibling `.vec.json`. | **OK** 171/171 |
| **E3** | Embed source by kind: episodes embed `situation`; all other kinds embed body. | **OK** (verified in `embed.Text`) |
| **E4** | Freshness hash ⊇ embed source: the staleness hash must cover *whatever is embedded*. (Currently `ContentHash` hashes body, episodes embed `situation` → disjoint → situation edits invisible to staleness for all 64 L1.) | **BROKEN** (code-verified `hash.go`) |
| **E5** | Episode `situation` non-empty: else `embed.Text` silently falls back to body, self-violating E3. | gap (code-verified `hash.go:67-69`) |

## D. Provenance / evidence — `[VC]`
| id | invariant | status |
|---|---|---|
| **P1** | Episode provenance valid: every episode names ≥1 session id whose transcript source path exists on disk, with a parseable range. | spot-OK |

## E. Marker / forward-progress — `[PT]`
| id | invariant | status |
|---|---|---|
| **M1** | Never skip: scanning [from, now] visits every learnable row exactly once across runs — no row skipped, none re-emitted forever. | FIXED (non-segments, 5c16c784); no property test yet |
| **M2** | Never past unread: a source's marker never advances past the earliest row not actually read this run. | FIXED for `emitTranscripts`; **BROKEN for `emitSegments`** (M2-segments: no `Partial` on segments path → marker → file Mtime) |
| **M3** | Multi-source independence: one source consuming the byte budget never advances another source's marker. | FIXED (5c16c784) |

## F. Recall / clustering determinism — `[PT]`
| id | invariant | status |
|---|---|---|
| **C1** | Clustering determinism: `AutoK(k-means + silhouette)` returns the same result for a fixed vault + phrase. | unchecked |
| **L3-1** | L3 match stability: a cluster with centroid cosine ≥0.9 to an existing L3 UPDATES it (never spawns a near-duplicate); the boundary is stable. | unchecked |
| **R1** | Recall-mirror: a note written with `situation` S is retrievable by a query phrased as S (learn and recall are inverse over the situation field). | untested |
| **R2** | Graph expansion really traverses links: recall's subgraph/cluster/hub computation uses the wikilink graph and degrades gracefully on a sparse graph. | unchecked |

## G. Concurrency — `[PT]`
| id | invariant | status |
|---|---|---|
| **K1** | Vault write-lock: concurrent `engram learn` never computes the same next Luhmann id and never overwrites a note (flock on `.luhmann.lock` spans id-compute→write; `O_EXCL` backstops). | enforced in code; untested as a property |

## H. Update / deployment — `[IT]`
| id | invariant | status |
|---|---|---|
| **U1** | `update` idempotence: re-running `engram update` with identical source is a copy-equivalent no-op; missing-go / no-harness / missing-skills fail with sentinels (`ErrGoNotFound` / `ErrNoHarness` / `ErrSkillsSrcMissing`). | uncaptured surface |

## Acceptance / self-tests already in the system (keep, don't duplicate)
- **RT-1** §6b self-verify: after writing an L3 ADR, `engram query --tier L3 --phrase "<seed>"`
  must return it in `items` (skill `learn` §6b). Depends on T1a; this is the system's own
  acceptance test that T1a works.

## Phase-1 antagonist additions + dispositions (agreed)

**New invariants (verified or census-clean-but-unguarded).** Numbered **M4–M8**, continuing the
marker block (M1–M3). (M4 embed-homogeneity, M5 situation-presence, M6 idempotency,
M7 marker-monotonicity, M8 luhmann-uniqueness.)

| id | invariant | sev | status |
|---|---|---|---|
| **M4 (embed model homogeneity)** | all sidecars share one `embedding_model_id`; `loadCompatibleSidecars` (`query.go:803-833`) **silently drops** mismatches → a model swap silently empties recall with no error. | **FAIL** | clean now (171× `minilm-l6-v2@384`), **unguarded** |
| **M5 (situation on L2/L3)** | every fact/feedback has non-empty `situation` (R1 depends on it); CLI marks it `required` only for episodes (`targets.go:36` vs `:48,:58`). | **FAIL** | clean now (107/107), unguarded |
| **M6 (learn idempotency)** | re-running `/learn` over the same window does not spawn duplicate/near-duplicate notes (marker is the only dedup; arcs may overlap). | WARN | untested |
| **M7 (marker monotonicity)** | per source, `marker_after ≥ marker_before` across runs (never regress → never silently re-emit history). | **FAIL** | untested (companion to the M1–M3 transcript invariants) |
| **M8 (Luhmann-id uniqueness/well-formed)** | leading ids unique across the vault and match `[0-9]+[a-z0-9]*` (K1's outcome). | WARN | clean now (171 distinct), unguarded |

**Dispositions:**
- **R1 (recall-mirror) and C1 (clustering determinism) are DETERMINISTIC `[PT]`, not "untested-as-judgment".** R1 = embed `situation` S, query S, assert top-k by cosine contains the note (no LLM). C1 seed is `FNV-1a(query)` (`query.go:1364`) → run twice, assert identical.
- **R2** keep the "traversal uses the graph" half as `[PT]`; the "degrades gracefully" half is a guideline, not a checkable invariant.
- **Agent-discipline items are RT-only, NEVER checker-gated:** recall Step-0 plan-print, §3a synthesis gate, binding-principle judgement, please step-ordering. (Like RT-1.)
- **P1 status correction:** not "spot-OK" — 64/65 episode transcript paths exist, **1 missing** (note 126). One real violation.
- **Severity model (required so the checker can PASS):** FAIL = breaks correctness (G0, G5, T1a, E1, E4, M2-segments, M4, M5, M7). WARN = corpus health (G1–G4, P1, M6, M8, S1, S2). The Phase-8 checker FAILs CI only on FAIL-class; WARN is reported, non-blocking — else a day-one vault with 138 orphans makes the gate permanently red and it gets disabled.
- **Checker robustness requirement:** parse UTF-8/bytes-robustly. BSD `grep` aborts on invalid multibyte sequences and silently skipped 7 non-UTF-8 notes in the antagonist's first census (false "7 untagged"). A grep-based checker would false-positive exactly the way both of us did.

## Phase-3 additions (surfaced by sequence-diagram grounding; agreed)

Skill↔binary boundary + representation invariants the flow diagrams exposed. Both are code-verified
and currently BROKEN. (The skill layer is RT-gated, so S1 is an RT/structural check, not a VC gate.)

| id | invariant (testable property) | sev | status |
|---|---|---|---|
| **S1 (no unmediated vault WRITES)** | the skill layer must not *edit/write* vault note files outside a sync-preserving `engram` path. Reading member files directly in recall §3a is **intended** — the agent decides per-cluster whether the abstract signal is worth investigating; that lazy, judgment-based read stays. | WARN | §3a member read: **OK by design** (operator 2026-06-04 — keep paths-only payload). §6b situation-edit: **FIX via `engram resituate`** (D4) — today a manual file poke that can desync the two `situation` copies (INV-S2). |
| **S2 (single embedded representation)** | a datum that affects retrieval has ONE source representation; no shadow copy that is expected to change retrieval but isn't the embedded text. | WARN | **BROKEN for facts** — `situation` is stored in BOTH frontmatter (`renderFactFrontmatter`) and the body formula (`renderFactBody`, `learn.go:568`); only the body is embedded (`embed.Text`, non-episode→body) and hashed (`ContentHash`). A §6b frontmatter-only situation edit is a retrieval no-op and invisible to `embed apply --stale`. Same write≠read family as E4/G0. **Fix (D4):** `engram resituate` rewrites BOTH copies + re-embeds; `engram check` asserts frontmatter `situation` == the body-formula situation. |

## What the checker (`engram check`, Phase 8) must implement
VC-class invariants (A, C, D) + the tier/graph parts of B over the real vault, exit non-zero
on violation, wired as a `targ` gate. PT-class (E-marker, F, G) become rapid tests in the
relevant packages. Tier isolation across all channels (T1a) is now a **code** fix — `applyTierFilter` must extend from `items` to `clusters`/`nearest_l3`/`hubs`.
