# Phase 0 — Source Review: intended contracts, decisions, inconsistencies

Date: 2026-06-04. Read-only review of the last 5 days (2026-05-30→06-04) to ground
the rigor effort. Every claim is tied to a note id / file / commit / segment, and
the load-bearing ones were witnessed by running the binary or reading the code.
This is the Phase-0 artifact; it feeds Phases 1 (invariants), 2–4 (C4/diagrams/ADRs).

## CORRECTIONS — post-antagonist (agreed; canon updated in the rigor doc)
The Phase-0 adversary round overturned one keystone claim and added findings:
- **D1 RETRACTED — `--tier` is NOT broken.** I (the original reviewer) counted
  `clusters[].members` (intentionally tier-agnostic) as `items`. Re-verified 3 ways:
  `--tier L1`→29/29 L1, `--tier L2`→11/11 L2, `--tier L3`→0 items. The real point is a
  **spec gap (M-1):** the three channels (items / cluster-members / down-links) were never
  named; `--tier` constrains `items` only (INV-T1a), clusters are tier-agnostic by design
  (INV-T1b), `nearest_l3` must survive the filter for §6b (INV-T1c).
- **Eval validity → OPEN, not invalid** (O-2): items isolation holds, so tier cells weren't
  contaminated via items; they *could* be via cluster-members IF recall consumed the whole
  payload. Phase-6 question.
- **D2 downgraded** "three contradictory specs" → **two** (design-prose vs Decision-3; shipped
  follows Decision-3).
- **NEW M2-segments** (code-verified): `emitSegments` advances the marker to file Mtime even when
  `SegmentsFrom` truncated at the budget (no `Partial` on the segments path) — INV-M2 holds for
  `emitTranscripts`, unenforced for `emitSegments`. Latent (the skill runs `--segments` without
  `--mark`), but real.
- **NEW missed contracts:** INV-K1 vault write-lock (flock + O_EXCL), INV-U1 `update` idempotence.
- **UPHELD/verified:** INV-E4, INV-E5, no-adr-kind (D4), no-tripwire (D7), L1↔L2 orphaning,
  L2→L3 sparsity, marker forward-progress (non-segments), recall pipeline shape.

> Tooling caveat (also a finding, D5): `engram transcript --segments` defaults to
> `--max-bytes 200000` and, unlike `--mark`, carries no marker — against the 20 MB
> active session it silently truncated at 29 turns (~1 of 5 days). The full 113-turn
> spine (through 2026-06-04T04:10) required `--max-bytes=200000000`. This review uses
> the full spine.

## A. Intended contracts
Sources: `docs/architecture/c1-system-context.md`, `skills/{learn,recall,please}/SKILL.md`
(verified byte-identical to deployed), `docs/superpowers/specs/2026-05-31-l3-synthesis-tier-design.md`.

- **A1 Learn pipeline** L0 raw transcript (external, not a built tier) → **L1 episode**
  (one noise-filtered chunk per *work-arc*; harness-injected USER turns stripped, tool
  summary + ASSISTANT decisions kept; `tier:L1` hard-set, `learn.go`) → **L2 fact/feedback**
  (specific lessons, default `tier:L2`, each `--relation "<episode>|extracted from this chunk"`)
  → **L3 ADR** (§6b: scenario-seed 3–6 situations; query each; if new L2 doesn't rank, revise
  its `situation` + re-embed; update-or-create by **centroid cosine ≥0.9**; write `fact --tier L3`
  mandatory + self-verify).
- **A2 Recall pipeline** skill phrases 5–15 query strings → one `engram query` (per-phrase
  sub-pipeline: embed → top-k cosine → 3-hop wikilink subgraph cap 200 → k-means+silhouette
  clusters → top-5 in-degree hubs → server-side merge) → **Step 3a per-cluster synthesis gate**
  (parent gates: ≥3 members + coherent rep; dispatches fire-and-forget subagent) → subagent
  **binding-principle judgement + link-to-bind** (if principle already in an anchor member, add
  `--relation` spokes; if net-new, write a fact; else nothing) → **Step 4 "as-requirements"**.
- **A3 Tier model** L1 episode / L2 fact-feedback / L3 ADR; down-links L3→L2→L1; `--tier X`
  caps returned items to tier X; **default is blended/kind-agnostic** (Decision 3).
- **A4 Wikilink graph** authored wikilinks walked by the binary; dangling silently dropped;
  drives subgraph + hub computation; link-to-bind adds edges at recall time.
- **A5 Embed-on-write** sidecar per note; **episodes embed `situation`, others embed body**
  (`embed.Text`, commit a9c3bce6).
- **A6 Markers** per-harness RFC3339; "Mtime OR inner row timestamp" (4901bf78) enables
  intra-session splitting; strict-greater boundary; "never advance past the earliest row not
  read" (5c16c784).
- **A7 please** seven-step bracket, no subcommand, steps non-waivable.

## B. Architectural decisions (ADR seeds)
- **B1** Skills + slim binary split (deterministic compute in Go; LLM judgment in skills).
- **B2** Capture abstraction = **generic-actionable**, set at learn-time. 5×5 matrix (note 124):
  correctness non-monotonic, peaks at generic-actionable (~12.4/17) vs generic (5.4) / recipe (8.8).
  Corollary (note 255): correctness set at learn-time; recall amplifies, doesn't create.
- **B3** L3 = scenario-discoverable ADRs over L2 clusters.
- **B4** L3 update-or-create by **centroid cosine ≥0.9**, not membership overlap (3a574706, 066cceab).
- **B5** `--tier` is an optional cap; **blended is default** — tier-isolated empirically weaker
  than blended (note 160; L3-chain 3.3–3.7 vs 8–10).
- **B6** Embed episodes by `situation` (a9c3bce6).
- **B7** Episode = per-arc chunk, harness-noise stripped (98c962ea, 88e9efeb, 56f59083).
- **B8** Episode provenance stores resolved transcript file path, cwd-independent (b4e24f76, d29be7d7).
- **B9** Marker forward-progress: strict-greater + intra-session split + multi-source independence
  (4901bf78, 5c16c784).
- **B10** "Curate, don't regenerate" (May 31, notes 126/128) → later overridden by deliberate
  from-scratch rebuild (Jun 3), after B9 fixes — evolution, not contradiction.
- **B11** Tier as frontmatter field w/ type-derived defaults + override; **no `adr` kind**
  (errLearnUnknownType = feedback/fact/episode; the 1 L3 is `type:fact tier:L3`).

## C. Candidate invariants (additions beyond rigor §1 — all fold into the canon)
- **INV-E4 (freshness-hash ⊇ embed-source): VIOLATED for all 64 episodes.** `ContentHash` hashes
  body (`hash.go:12-13`); episodes embed `situation` from frontmatter (`hash.go:61-71`). Disjoint
  → situation edits leave `content_hash` unchanged → stale vector reported fresh. **Verified in code.**
- **INV-E5 (episode-situation non-empty):** `Text` falls back to body when `situation` empty
  (`hash.go:67-69`) → silently self-violates INV-E3. **Verified in code.**
- **INV-T2 re-phrase (asymmetric):** episode⇒L1 is rigid; fact/feedback may be L2 **or** L3
  (override is a feature, 83d9f8ec). Restate: *episodes must be L1; fact/feedback may be L2 or L3;
  no note is L1 unless it is an episode.* (Original INV-T2 wording contradicts the override.)
- **INV-T3 (top-tier default): DROP/RE-PHRASE.** Encodes the abandoned "top-tier-only" design prose;
  shipped behavior is blended default (Decision 3, witnessed). Re-state to the blended-default contract.
- **INV-C1 (cluster/AutoK determinism):** k-means+silhouette+AutoK deterministic given fixed vault+phrase.
- **INV-L3-1 (centroid-cosine match soundness):** a cluster matching an existing L3 must update it,
  not spawn a near-duplicate; the ≥0.9 boundary must be stable.

## D. Inconsistencies (intent vs impl, and cross-5-day)
- **D1** `--tier` does not isolate tiers (witnessed; taints the eval's tier-regime cells). [INV-T1]
- **D2** "Default tier" specified **three contradictory ways**: design-prose "top tier only" vs
  Decision-3 "kind-agnostic blended" vs shipped blended. Rigor INV-T3 encodes the abandoned side.
- **D3** Freshness hash ⟂ embed source for episodes. [INV-E4, verified]
- **D4** `adr` designed as a new *kind* (design doc l.24) but shipped as a *tier override on fact*;
  no adr kind exists. Checks/docs expecting an ADR kind will not find one.
- **D5** `--segments` (recovery spine) silently truncates at the 200 KB default — under-reports the
  audited history with no marker/signal. (A reliability gap in the audit tool itself.)
- **D6** "Don't regenerate" → full rebuild: time-ordered evolution w/ cause (record as a reversed
  decision, not a logic contradiction).
- **D7** No automated invariant tripwire exists (no vault `check`/`audit` subcommand) — the root
  cause of this whole effort.
