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

G0 (the write-form and resolve-form of a link differ) is [792a9b](../../changes/792_documentation-framework-adoption/792a_product-spec.md#792a9b-links-that-should-resolve).

| id | invariant (testable property) | sev | status (BINARY's real graph) |
|---|---|---|---|
| **G0 (link-form normalization, root cause)** | Ported to [792a9b](../../changes/792_documentation-framework-adoption/792a_product-spec.md#792a9b-links-that-should-resolve). | — | — |
| **G1** | **[RETIRED — L1/L2/L3 tier ladder removed in recall-v2; no episode kind; notes are now untiered fact/feedback.]** ~~Tier ladder downward: every L3 links ≥1 L2; every L2 links ≥1 L1.~~ | — | RETIRED |
| **G2** | Not run by `engram check` ([792d100](../../changes/792_documentation-framework-adoption/792d_decision-log.md#792d100-vault-invariant-checker-scope-and-the-tests-it-leaves)). | — | — |
| **G3** | Ported to [792b26c2](../../changes/792_documentation-framework-adoption/792b_technical-spec.md#792b26c2-link-resolution). | — | — |
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
| **E1** | Ported to [792b26c4](../../changes/792_documentation-framework-adoption/792b_technical-spec.md#792b26c4-sidecar-schema) (a missing sidecar is a warning there). | — |
| **E3** | **[RETIRED — episode kind removed; all notes (fact/feedback) embed body. `embed.Text` still has situational logic but the episode-specific branch is dead.]** ~~Embed source by kind: episodes embed `situation`; all other kinds embed body.~~ | — | RETIRED |
| **E4** | **[RETIRED — episode kind (and its `situation`-only embed source) removed. The hash/embed mismatch for episodes no longer applies. For fact/feedback, body is both embedded and hashed — no disjoint path.]** | — | RETIRED |
| **E5** | **[RETIRED — episode kind removed; no episode `situation` field to be empty.]** | — | RETIRED |

## D. Provenance / evidence — `[VC]`
| id | invariant | status |
|---|---|---|
| **P1** | **[RETIRED — episode kind removed; chunk-source provenance is now frontmatter on fact/feedback notes and is not a path-validity concern (chunks are indexed separately).]** ~~Episode provenance valid: every episode names ≥1 session id whose transcript source path exists on disk.~~ | — | RETIRED |

## F. Recall / clustering determinism — `[PT]`
C1 (clustering determinism) is [792b19c2](../../changes/792_documentation-framework-adoption/792b_technical-spec.md#792b19c2-k-means-partition).
R1 (the recall mirror: a note written with situation S is found by a query phrased as S) is [792b24c22](../../changes/792_documentation-framework-adoption/792b_technical-spec.md#792b24c22-a-note-is-found-by-its-situation).

| id | invariant | status |
|---|---|---|
| **L3-1** | **[RETIRED — L3 note kind removed; candidate synthesis now uses `candidate_l2s` within-cluster nomination, not a centroid-cosine L3 match-stability gate.]** | — | RETIRED |
| **R2** | **[RETIRED — recall no longer does subgraph/hub expansion; `vaultgraph` is used only by `check`/`amend`, not in the query path.]** ~~Graph expansion really traverses links: recall's subgraph/cluster/hub computation uses the wikilink graph.~~ | — | RETIRED |

## G. Concurrency — `[PT]`
| id | invariant | status |
|---|---|---|
| **K1** | Ported to [792a3m](../../changes/792_documentation-framework-adoption/792a_product-spec.md#792a3m-concurrent-writes-never-collide) and [792b8](../../changes/792_documentation-framework-adoption/792b_technical-spec.md#792b8-added-vault-write-lock). | — |

## H. Update / deployment — `[IT]`
| id | invariant | status |
|---|---|---|
| **U1** | Ported to [792b29c16](../../changes/792_documentation-framework-adoption/792b_technical-spec.md#792b29-added-harness-deploy-sync) (idempotence) and [792b28c3](../../changes/792_documentation-framework-adoption/792b_technical-spec.md#792b28-added-self-installer), [792b28c4](../../changes/792_documentation-framework-adoption/792b_technical-spec.md#792b28-added-self-installer) and [792b28c15](../../changes/792_documentation-framework-adoption/792b_technical-spec.md#792b28-added-self-installer) (the sentinel errors). | — |

## Acceptance / self-tests already in the system (keep, don't duplicate)
- **RT-1** **[RETIRED — `--tier` flag and L3 ADR kind removed in recall-v2; §6b self-verify via `engram query --tier L3` is no longer meaningful.]** ~~§6b self-verify: after writing an L3 ADR, `engram query --tier L3 --phrase "<seed>"` must return it in `items`.~~

## Phase-1 antagonist additions + dispositions (agreed)

**New invariants (verified or census-clean-but-unguarded).** Numbered **M4–M8**, continuing the
marker block (M1–M3). (M4 embed-homogeneity, M5 situation-presence, M6 idempotency,
M7 marker-monotonicity, M8 luhmann-uniqueness.)

| id | invariant | sev | status |
|---|---|---|---|
| **M4** | Ported to [792b10c7](../../changes/792_documentation-framework-adoption/792b_technical-spec.md#792b10c7-state-classification) (792d66). | — | — |
| **M5 (situation on facts/feedback)** | Ported to [792b26c3](../../changes/792_documentation-framework-adoption/792b_technical-spec.md#792b26c3-situation-presence) (the format side is 792b6c3). | — | — |
| **M6 (learn idempotency)** | re-running `/learn` over the same window does not spawn duplicate/near-duplicate notes (marker is the only dedup; arcs may overlap). | WARN | untested |
| **M7 (marker monotonicity)** | per source, `marker_after ≥ marker_before` across runs (never regress → never silently re-emit history). | **FAIL** | untested (companion to the M1–M3 transcript invariants) |
| **M8** | Ported to [792b7](../../changes/792_documentation-framework-adoption/792b_technical-spec.md#792b7-added-luhmann-addressing). | — | — |

**Dispositions:**
- **R1 (recall-mirror) and C1 (clustering determinism) are DETERMINISTIC `[PT]`, not "untested-as-judgment".** R1 = embed `situation` S, query S, assert top-k by cosine contains the note (no LLM). C1 seed is `FNV-1a(query)` (`query.go:1364`) → run twice, assert identical.
- **R2** **[RETIRED — recall no longer does graph traversal; disposition moot.]**
- **Agent-discipline items are RT-only, NEVER checker-gated:** recall Step-0 plan-print, §3a synthesis gate, binding-principle judgement, please-runbook step-ordering. (RT-1 itself is RETIRED — see above.)
- **P1 status correction:** **[RETIRED — episode kind and transcript-path provenance removed.]**
