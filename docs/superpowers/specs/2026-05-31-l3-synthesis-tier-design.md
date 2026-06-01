# L3 synthesis tier + tier-capped retrieval — design

Date: 2026-05-31

## Motivation

L2 is a pile of specific facts; a fact only surfaces if the agent queries the
matching keywords. But an agent about to make a mistake doesn't know the lesson
exists or how to phrase a query for it. **L3 generalizes a cluster of L2 facts
into a crisp, ADR-style standard that is discoverable from the *scenario* the
agent is in** — surfacing the standard *before* they act, whether or not they
know they need it. Recall returns the L3 standard; the agent can chase links
down to the specific L2s and L1 evidence if it wants.

(Future direction, NOT in scope: L4+ would synthesize clusters of L3
recursively. Design so the tier mechanism generalizes, but build only through L3.)

## Tiers (each note carries a `tier` tag)

| tier | kind | content |
| --- | --- | --- |
| L1 | episode | raw transcript-chunk evidence |
| L2 | fact / feedback | specific lessons / conventions |
| **L3** | **adr (new)** | a short ADR synthesizing ONE L2 cluster into a standard |

Down-links: L3 `--relation` → each covered L2; L2 `--relation` → its L1 episodes.

## Tier-capped retrieval

`engram query` returns **only the top tier present** (or an explicit `--tier`).
Lower tiers stay in the vault, linked but not *returned*, so the agent can chase
links by choice.
- L1 arm → episodes; L2 arm → facts only (L3 never generated); L3 arm → ADRs only.

## L3 generation / update (runs inside `/learn` when new L2 lands)

For each new or changed L2 fact:
1. **Scenario seeds** — enumerate *situations an agent could be in where this L2
   should be surfaced and considered before acting* (situational / plan-grounded,
   NOT lesson keywords). An agent queries by scenario, not by the lesson it
   doesn't know it needs.
2. **Search** those scenarios (existing multi-phrase clustering + silhouette).
3. **Ensure discoverability** — the new L2 must rank high for those scenario
   searches; if it doesn't, **tweak the L2's `situation`/framing until it does**.
4. **Update-or-create** — for each returned cluster: if it **>=90% overlaps an
   existing L3's source set -> update that L3** (regenerate, preferring recent L2s
   where they diverge); else **create a new L3**. Leave all other L3s untouched.
   One L2 may legitimately land in several clusters and feed several L3s.
5. **Write the L3 as a short ADR** — crisp and to the point when surfaced.
6. **Link** L3 -> each covered L2 with the relationship rationale.

## Architecture (binary vs skill — engram's established split)

- **Binary (Go; DI everywhere, TDD, `targ`):** the `tier` tag; tier-capped query
  (`--tier` / top-tier default); cluster source-set overlap computation.
  Clustering + silhouette + embedding already exist in the query pipeline.
- **Skill (`/learn` synthesis flow):** scenario-seed generation, ranking-check +
  L2-tweak, ADR authoring, linking, and the update-or-create orchestration. These
  are LLM-judgment steps; the binary stays pure compute.

## Resolved decisions

1. **Trigger:** L3-gen runs automatically at the end of every `/learn` that wrote
   L2 — skill-orchestrated inside `/learn`, part of the capture flow.
2. **Cluster -> L3 matching: SEMANTIC (centroid cosine).** Match a returned cluster
   to an existing L3 by the cosine similarity of the cluster's embedding centroid
   to the L3's embedding, threshold ~0.9: at/above -> **update** that L3 (regenerate,
   preferring recent L2s where they diverge); below -> **create** a new L3. The L3
   still records its source L2 slugs for links + audit, but membership-set overlap
   is NOT the match metric (it fragments a standard every time the topic gains a
   fact). The binary computes the centroid cosine; the skill decides update-vs-create.
3. **Tier-cap:** a `--tier` flag on `engram query`. Default keeps today's
   kind-agnostic behavior (existing recall unchanged); tier-isolated runs pass `--tier`.
4. **ADR shape: a TESTED VARIABLE, not fixed.** Implement the ADR template as
   swappable so alternative formats can be A/B-tested in the validation chain. Start
   from a candidate (title + 2-3 line context + the standard + derived-from L2 slugs)
   but treat the format as tunable, not load-bearing in the design.

## Validation

- TDD unit tests: `tier` round-trip, `--tier` query filtering, overlap
  computation, the update-vs-create decision.
- End-to-end: the **accumulation chain** — headless `claude -p` builds, `/recall`
  and `/learn` through the skills, 3 layers x 3 app-orders x 3 stages — as the
  behavioral test of whether L3 (vs L1/L2) accumulates usefully and survives the
  order permutation.
