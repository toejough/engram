# Link-value exploration — plan

> **For agentic workers:** research + isolated-harness eval plan (no production binary/skill changes
> this round). Orchestrator delegates research and fans out probe stages; PoCs run offline against
> sidecar embeddings + wikilink files.

**Ask (Joe, 2026-07-02, condensed):** run the 3-step link path — (1) prove/refute link worth, (2) if
worth proven, add the `--relation` habit to learn, (3) retroactive linking sweep — but do #1 as a
*designed exploration*: research first (incl. every.to's compound-engineering guide), postulate 5–10
link-criteria × traversal variants, quick PoCs, prune obvious failures, a more rigorous round on the
survivors, iterate to 1–2 winning concepts, **then STOP and talk to Joe**. Capture
vision-relevant-but-out-of-scope findings in a followup doc. Engram should end up at least as capable
as the compound-engineering capture skill, and fire more autonomously.

**Stop-point interpretation (encoded, conservative):** steps 2 & 3 are approved *in principle*
contingent on #1, but their shape depends on which link concept wins — so the workflow ends at the
winners-conversation; 2 & 3 execute as a follow-on after Joe's go.

**Spend estimate (no cap; runs to completion):** fabric construction ~$5–15; PoC probes ~free
(offline replays); rigor round ~$20–60. Total ~$30–80.

## The evidence boundary this design honors

- **Settled null (note 73, 2026-06-23):** one-hop graph expansion of the matched set, on deep
  10-phrase recall, over the then-fabric = 0 marginal value — because cosine breadth *already
  co-surfaced* the bridge. Mechanism, not just result: "given the right notes co-surfaced, opus
  already composes." T1 below re-runs this ONLY as a control.
- **Open aspiration (note 68):** compositional/transitive/analogical retrieval needs a relational
  representation; `internal/vaultgraph` is the unused substrate.
- **Therefore the north star:** headroom exists only in the **miss population** — queries where
  cosine does NOT surface a needed note (glance's 3-phrase breadth cliff, C6-badge n<3, is the
  known example), buries it below the payload cut, or surfaces a superseded note without its
  superseder. Every variant is judged on recovery in that population, never on cases cosine already
  wins (73 predicts null there; measuring it again is waste).
- **Fabric reality (measured 2026-07-02):** 135 notes, 80 edges, 58% isolated, one 45-note
  component. Today's fabric is too sparse to test traversal fairly — so fabric-construction variants
  (what edges COULD exist) are half the exploration.

## Stage S0 — Research (delegated; sonnet + web)

- Fetch https://every.to/guides/compound-engineering (WebFetch) + 2–4 adjacent primary sources it
  cites or that surveys surface. Focus questions: (a) what link/relation kinds does their "related
  docs finder" create — semantic, causal, categorical? (b) how do their docs get FOUND at recall
  time (search? tags? frontmatter fields?), (c) what do the other five roles (context analyzer,
  solution extractor, prevention strategist, category classifier, doc writer) imply about capture
  quality we don't do, (d) trigger model — how human-dependent is it really?
- REUSE, don't re-research: the committed 13-technique survey
  (`docs/design/artifacts/2026-07-01-memory-review-techniques-survey.md`) already covers HippoRAG/
  PPR, GraphRAG, A-Mem, Generative-Agents scoring — the researcher reads it first.
- Two outputs: (i) refinements/additions to the variant catalog below (research may add or mutate
  variants — the catalog is a prior, not a cage); (ii) a draft **followup doc** of findings relevant
  to engram's vision but NOT this link activity (`docs/design/2026-07-02-compound-engineering-followups.md`)
  — e.g. capture-role decomposition, autonomy triggers, frontmatter conventions.

## The variant catalog (prior; 7 link-criteria × 6 traversals — S0 may amend)

**Link-criteria variants** (what edges exist; L1 is the control fabric):
| ID | Edges from | Cost to build | Rationale |
|---|---|---|---|
| L1 | status-quo fabric (80 typed edges) | $0 | control |
| L2 | corpus-wide link-on-write: per note, top-K embedding candidates over the whole vault → LLM justify with the existing 2.6 precision gate | ~$5–10 | the compound-eng "related docs finder" mechanism; fixes "only co-surfaced pairs ever considered" |
| L3 | shared rare tokens (TF-IDF-rare concrete tokens shared across notes — "imptest", "flock", "capWithNoteFloor") | $0 (mechanical) | note 153: the concrete token is load-bearing; token-shared notes are mutually relevant |
| L4 | situation-field cosine (embed situation handles only; link near-family situations) | $0 (mechanical) | same-moment lessons should co-surface |
| L5 | supersession/temporal edges (LLM-typed: updates / narrows / refutes) | ~$3–5 | enables conflict surfacing; the C4i mechanism as persistent structure |
| L6 | tag/category taxonomy (controlled vocabulary per note; hub-by-design, used for filter/discovery not ranking) | ~$2–4 | compound-eng classifier role; tests whether our hub-kill overcorrects for discovery |
| L7 | provenance/episode edges (same origin session/source) | $0 (mechanical) | cheap temporal cohesion |

**Traversal variants** (how recall consumes edges; T1 is the settled-null control):
| ID | Mechanism | Claim it tests |
|---|---|---|
| T1 | one-hop payload expansion of matched notes | control — must reproduce the null on L1; may differ on richer fabrics |
| T2 | PPR/spreading activation seeded by the matched set; final score = α·cosine + β·ppr | HippoRAG-lite: associative recovery of unmatched-but-connected notes |
| T3 | neighbor rank-boost (edges never ADD items, only re-rank: a below-floor/buried note gets boosted if a strong match links to it) | recovery without payload growth |
| T4 | candidate_l2s enrichment (neighbors join CANDIDATE nomination for the 2.5 coverage judgment only) | better dedup/coverage, zero payload cost |
| T5 | typed-selective traversal (follow ONLY supersession/contradiction edges; surface the superseder alongside any matched superseded note) | conflict-correctness, the C4i mechanism generalized |
| T6 | glance-breadth substitute (under 3-phrase glance, one-hop from top matches) | links as cheap breadth-recovery for the frequent rung |

Not every L×T cell runs — S2 probes the sensible pairings (mechanical fabrics feed all traversals;
L6 pairs with a filter-style lookup, not PPR).

## Stage S1 — Build the miss population + fabrics (the eval's foundation)

- **P1 real-query misses:** replay the phrase sets from real recorded recalls (session transcripts +
  this session's saved payloads) at n=3 (glance) and n=10 (deep) against the current binary; an LLM
  judge (sonnet) marks needed-but-absent notes (needed = the note's lesson bears on the query's
  task; the vault is small enough to sweep). Output: {query → missed-note} cases.
- **P2 constructed bridges:** C6-badge-style two-hop cases from the real vault (note A relevant to
  the query only through B); ~6–10 cases.
- **P3 supersession pairs:** known pairs (120→153; the 350s→190s mislabel notes; 82's fixed leg;
  qanchor park) — query matches the OLD note; does the NEW one surface?
- Build L2–L7 fabrics (scripts under `dev/eval/links/`); report fabric stats (edges added, per-note
  degree) — no silent caps.
- **Gate S1:** if P1+P2+P3 together yield < ~8 real miss cases, STOP EARLY and report — the miss
  population itself may be too small for links to matter (that is a finding, not a failure).

## Stage S2 — PoC probes (free) + prune

- For each sensible L×T pairing: replay every miss case offline (sidecar `.vec.json` embeddings +
  wikilink files; no binary changes) and measure: **recovery@K** (missed note now in top-K), rank
  movement, payload delta (items added), and **collateral** (did known-good top-5s degrade on a
  20-query no-regression set?).
- Prune rules (pre-registered): kill any variant with recovery ≤ control, payload growth > +20%
  without recovery gain, or collateral regression. Report the kill list with numbers.

## Stage S3 — Rigor round (survivors only; LLM-judged)

- For each survivor (expect 2–4): a delivery-style blind eval on the recovered cases — inject the
  variant's payload vs the control payload into a task prompt; opus judge scores whether the answer
  APPLIES the recovered note's knowledge (same method as the qanchor eval; name-agnostic judge).
- Reps ≥ 2 per case per arm to size the noise floor; verdict per the 2σ rule (a sub-noise gap is
  "can't distinguish", never a win). Trap-gate C3–C6 smoke is N/A this round (no production change)
  but survivors' collateral check stands in for it.
- Iterate once (mutate/hybridize survivors) if results are suggestive-but-unclear; then stop.

## Stage S4 — STOP: present winners to Joe

1–2 winning concepts (or the honest null), each with: recovery + delivery numbers, cost to
productionize (binary work? skill work?), and the implied shape of steps 2 (learn `--relation`
habit — which link kinds the template should prompt for) & 3 (retroactive sweep — which fabric
builder to run for real). Steps 2 & 3 execute only after this conversation.

## Deliverables

- `dev/eval/links/` — fabric builders + probe harness + results JSONs (pure-logic scoring
  unit-tested where non-trivial, per the qanchor pattern).
- `docs/design/2026-07-02-link-value-exploration.md` — results doc (labeled tables with units,
  dates, vintage).
- `docs/design/2026-07-02-compound-engineering-followups.md` — the vision-followup doc Joe asked
  for.
- This plan (committed before execution).

## Constraints

- Isolated harness only; no production binary/skill changes this round (experimental-ask rule).
- T1×L1 is a control, not a candidate — the settled null must reproduce or the harness is suspect.
- Results as labeled tables with units + n; every number dated; vintage flagged if the system
  changes mid-exploration.
- Pre-registered prune rules (above) — no post-hoc survivor selection.
- No repo-wide tooling; scope-check the diff before every commit.
- Followup doc captures vision-relevant findings immediately (parked ≠ unrecorded, note 154).
