# Engram Memory-System Invariants (canonical, testable)

> **Reconciled 2026-06-20:** retired invariants marked inline below; survivors restated as-is. The L1/L2/L3 tier surface, episode kind, `--tier` flag, subgraph/hub path, and `nearest_l3`/`hubs` fields were removed in the recall-v2 / 2026-06-20 deep clean.

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
> `vaultgraph.BuildGraph` resolves edges **only by full basename** (`graph.go:64-78`). **[RETIRED — recall no longer uses the wikilink graph; `vaultgraph` is now used only by `check`/`amend`, not in the query path. The G0 bug in `BuildGraph` still exists and is relevant to `check`/`amend`, but the claim that "`query.go:557` recall builds its subgraph through that resolver" is no longer accurate.]**
> Verified counts:
> **Of 183 authored link-instances, 28 resolve to graph edges and 155 drop: 151 are bare-id
> (`[[105]]`), unresolvable by the basename resolver, plus 4 dangle to nonexistent notes; the other
> 28 are basename-form and all resolve. Result: 138/171 notes orphaned (80%), mean out-degree 0.16.
> (Empirically re-confirmed against the live 171-note vault on 2026-06-04: 151 + 28 + 4 = 183.)**
> My earlier graph numbers (23
> orphans, ~1.1 links/note) were **ID-aware** — I resolved `[[105]]`→leading-id `105` by hand.
> **Neither the binary NOR Obsidian does that** (both resolve only full basenames/aliases — verified
> against Obsidian's docs 2026-06-04: no prefix matching). That count was a *hypothetical post-fix*
> graph, not any tool's real behavior, and overstated the binary's real graph ~6×. **[RETIRED — recall no longer does graph-expansion (subgraph/clusters/hubs); INV-R2 below is correspondingly retired. The G0 basename-vs-id mismatch still affects `BuildGraph` and is relevant to `engram check`/`amend`.]**
> Same bug-class as INV-E4 (writer and reader keyed on disjoint representations). Fixable:
> normalize bare-id→basename in the resolver, or write basenames.

| id | invariant (testable property) | sev | status (BINARY's real graph) |
|---|---|---|---|
| **G0 (link-form normalization, root cause)** | every authored `[[target]]` resolves through `BuildGraph` to a node (target is a basename, or the resolver normalizes id→basename). | **FAIL** | **BROKEN** — 155/183 edges dropped |
| **G1** | **[RETIRED — L1/L2/L3 tier ladder removed in recall-v2; no episode kind; notes are now untiered fact/feedback.]** ~~Tier ladder downward: every L3 links ≥1 L2; every L2 links ≥1 L1.~~ | — | RETIRED |
| **G2** | No orphans: no note has in-degree 0 AND out-degree 0. (WARN — a legitimately standalone principle is allowed; this is corpus health, not correctness.) | WARN | **BROKEN** — 138 (mostly masked by G0) |
| **G3** | No dangling wikilinks; the checker surfaces them. (The binary silently drops dangling at build — `graph.go:78` — so a dangling link is a lost intended edge.) | WARN | **BROKEN** — 155 edge-instances on the real graph |
| **G4** | **[RETIRED — L2/episode provenance links removed in recall-v2; chunk-source provenance is now recorded as frontmatter, not wikilinks.]** ~~Provenance edges present: L2→its episode; L3→each constituent L2.~~ | — | RETIRED |
| **G5 (episode-body wikilink isolation)** | **[RETIRED — episode kind removed; `[[x]]` in chunk bodies no longer parsed as vault edges.]** ~~graph edges come from authored relation context, NOT from `[[x]]` strings embedded in episode verbatim transcript bodies.~~ | — | RETIRED |

## B. Tier / retrieval correctness — `[PT + VC]`
| id | invariant | status |
|---|---|---|
| **T1a** | **[RETIRED — `--tier` flag removed in recall-v2; no `nearest_l3` or `hubs` fields in query payload.]** ~~`query --tier X` exposes ONLY tier-X notes across all channels.~~ | — | RETIRED |
| **T1b** | **[RETIRED — `--tier` flag removed; query is always kind-agnostic (blended) with no tier concept.]** | — | RETIRED |
| **T1c** | **[RETIRED — already superseded; further superseded by `--tier` removal.]** | — | RETIRED |
| **T2** | **[RETIRED — L1/L2/L3 tiers and episode kind removed in recall-v2; fact/feedback have no tier field.]** ~~Tier↔kind asymmetric: episodes must be L1; fact/feedback may be L2 or L3.~~ | — | RETIRED |

## C. Embedding integrity — `[VC]`
| id | invariant | status |
|---|---|---|
| **E1** | Presence: every note has a sibling `.vec.json`. | **OK** 171/171 |
| **E3** | **[RETIRED — episode kind removed; all notes (fact/feedback) embed body. `embed.Text` still has situational logic but the episode-specific branch is dead.]** ~~Embed source by kind: episodes embed `situation`; all other kinds embed body.~~ | — | RETIRED |
| **E4** | **[RETIRED — episode kind (and its `situation`-only embed source) removed. The hash/embed mismatch for episodes no longer applies. For fact/feedback, body is both embedded and hashed — no disjoint path.]** | — | RETIRED |
| **E5** | **[RETIRED — episode kind removed; no episode `situation` field to be empty.]** | — | RETIRED |

## D. Provenance / evidence — `[VC]`
| id | invariant | status |
|---|---|---|
| **P1** | **[RETIRED — episode kind removed; chunk-source provenance is now frontmatter on fact/feedback notes and is not a path-validity concern (chunks are indexed separately).]** ~~Episode provenance valid: every episode names ≥1 session id whose transcript source path exists on disk.~~ | — | RETIRED |

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
| **L3-1** | **[RETIRED — L3 note kind removed; candidate synthesis now uses `candidate_l2s` within-cluster nomination, not a centroid-cosine L3 match-stability gate.]** | — | RETIRED |
| **R1** | Recall-mirror: a note written with `situation` S is retrievable by a query phrased as S (learn and recall are inverse over the situation field). | untested — **SURVIVOR** |
| **R2** | **[RETIRED — recall no longer does subgraph/hub expansion; `vaultgraph` is used only by `check`/`amend`, not in the query path.]** ~~Graph expansion really traverses links: recall's subgraph/cluster/hub computation uses the wikilink graph.~~ | — | RETIRED |

## G. Concurrency — `[PT]`
| id | invariant | status |
|---|---|---|
| **K1** | Vault write-lock: concurrent `engram learn` never computes the same next Luhmann id and never overwrites a note (flock on `.luhmann.lock` spans id-compute→write; `O_EXCL` backstops). | enforced in code; untested as a property |

## H. Update / deployment — `[IT]`
| id | invariant | status |
|---|---|---|
| **U1** | `update` idempotence: re-running `engram update` with identical source is a copy-equivalent no-op; missing-go / no-harness / missing-skills fail with sentinels (`ErrGoNotFound` / `ErrNoHarness` / `ErrSkillsSrcMissing`). | uncaptured surface |

## Acceptance / self-tests already in the system (keep, don't duplicate)
- **RT-1** **[RETIRED — `--tier` flag and L3 ADR kind removed in recall-v2; §6b self-verify via `engram query --tier L3` is no longer meaningful.]** ~~§6b self-verify: after writing an L3 ADR, `engram query --tier L3 --phrase "<seed>"` must return it in `items`.~~

## Phase-1 antagonist additions + dispositions (agreed)

**New invariants (verified or census-clean-but-unguarded).** Numbered **M4–M8**, continuing the
marker block (M1–M3). (M4 embed-homogeneity, M5 situation-presence, M6 idempotency,
M7 marker-monotonicity, M8 luhmann-uniqueness.)

| id | invariant | sev | status |
|---|---|---|---|
| **M4 (embed model homogeneity)** | all sidecars share one `embedding_model_id`; `loadCompatibleSidecars` **silently drops** off-model sidecars → silent only under PARTIAL migration (a full-vault mismatch is guarded: `errQueryNoEmbeddings` when notes exist but no sidecar loads, plus a stderr model-mismatch warning). | **FAIL** | clean now (171× `minilm-l6-v2@384`); partial-mismatch case unguarded |
| **M5 (situation on facts/feedback)** | every fact/feedback has non-empty `situation` (R1 depends on it). **[Note: the claim "CLI marks it required only for episodes" is RETIRED — no episode kind; but the `situation` optionality for fact/feedback remains an unguarded gap.]** | **FAIL** | clean now, unguarded |
| **M6 (learn idempotency)** | re-running `/learn` over the same window does not spawn duplicate/near-duplicate notes (marker is the only dedup; arcs may overlap). | WARN | untested |
| **M7 (marker monotonicity)** | per source, `marker_after ≥ marker_before` across runs (never regress → never silently re-emit history). | **FAIL** | untested (companion to the M1–M3 transcript invariants) |
| **M8 (Luhmann-id uniqueness/well-formed)** | leading ids unique across the vault and match `[0-9]+[a-z0-9]*` (K1's outcome). | WARN | clean now (171 distinct), unguarded |

**Dispositions:**
- **R1 (recall-mirror) and C1 (clustering determinism) are DETERMINISTIC `[PT]`, not "untested-as-judgment".** R1 = embed `situation` S, query S, assert top-k by cosine contains the note (no LLM). C1 seed is `FNV-1a(query)` (`query.go:1364`) → run twice, assert identical.
- **R2** **[RETIRED — recall no longer does graph traversal; disposition moot.]**
- **Agent-discipline items are RT-only, NEVER checker-gated:** recall Step-0 plan-print, §3a synthesis gate, binding-principle judgement, please step-ordering. (RT-1 itself is RETIRED — see above.)
- **P1 status correction:** **[RETIRED — episode kind and transcript-path provenance removed.]**
- **Severity model (required so the checker can PASS):** FAIL = breaks correctness (G0, E1, M2-segments, M4, M5, M7). WARN = corpus health (G2–G3, M6, M8, S1, S2). **[Note: G1/G4/G5/T1a/E4/E5/P1 removed from severity lists — all RETIRED above.]** The Phase-8 checker FAILs CI only on FAIL-class; WARN is reported, non-blocking.
- **Checker robustness requirement:** parse UTF-8/bytes-robustly. BSD `grep` aborts on invalid multibyte sequences and silently skipped 7 non-UTF-8 notes in the antagonist's first census (false "7 untagged"). A grep-based checker would false-positive exactly the way both of us did.

## Phase-3 additions (surfaced by sequence-diagram grounding; agreed)

Skill↔binary boundary + representation invariants the flow diagrams exposed. Both are code-verified
and currently BROKEN. (The skill layer is RT-gated, so S1 is an RT/structural check, not a VC gate.)

| id | invariant (testable property) | sev | status |
|---|---|---|---|
| **S1 (no unmediated vault WRITES)** | the skill layer must not *edit/write* vault note files outside a sync-preserving `engram` path. Reading member files directly in recall §3a is **intended** — the agent decides per-cluster whether the abstract signal is worth investigating; that lazy, judgment-based read stays. | WARN | §3a member read: **OK by design** (operator 2026-06-04 — keep paths-only payload). §6b situation-edit: **FIX via `engram resituate`** (D4) — today a manual file poke that can desync the two `situation` copies (INV-S2). |
| **S2 (single embedded representation)** | a datum that affects retrieval has ONE source representation; no shadow copy that is expected to change retrieval but isn't the embedded text. | WARN | **BROKEN for facts** — `situation` is stored in BOTH frontmatter (`renderFactFrontmatter`) and the body formula (`renderFactBody`, `learn.go:568`); only the body is embedded (`embed.Text`, non-episode→body) and hashed (`ContentHash`). A §6b frontmatter-only situation edit is a retrieval no-op and invisible to `embed apply --stale`. Same write≠read family as E4/G0. **Fix (D4):** `engram resituate` rewrites BOTH copies + re-embeds; `engram check` asserts frontmatter `situation` == the body-formula situation. |

## What the checker (`engram check`, Phase 8) must implement
VC-class invariants (A, B[graph-only], D) over the real vault — G0 (link-form normalization), G2/G3 (orphans/dangling), S1/S2 (single-representation, no unmediated writes) — exit non-zero on violation, wired as a `targ` gate. PT-class (E-marker, F, G) become rapid tests in the relevant packages. **[RETIRED: `applyTierFilter` / `nearest_l3` / `hubs` — those fields and the tier-filter code no longer exist in the query path.]**
