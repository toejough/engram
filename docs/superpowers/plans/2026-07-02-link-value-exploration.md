# Link-value exploration — plan

> **For agentic workers:** research + isolated-harness eval plan (no production binary/skill changes
> this round). Orchestrator delegates research and fans out probe stages. (Amended post-Gate-A:
> recall-adaptation as a first-class variant dimension; all six compound-engineering roles
> interrogated; parity + autonomy elements added to the stop-point; probe mechanics pinned — the
> harness shells out to `engram query` for ranked lists and applies traversal math in Python.)

**Ask (Joe, 2026-07-02, condensed):** run the 3-step link path — (1) prove/refute link worth, (2) if
worth proven, add the `--relation` habit to learn, (3) retroactive linking sweep — but do #1 as a
*designed exploration*: research first (incl. every.to's compound-engineering guide), postulate 5–10
variations of link criteria and traversal algorithms, quick PoCs, prune obvious failures, a more
rigorous round on the survivors, iterate to 1–2 winning concepts, **then STOP and talk to Joe**.
Consider what kinds of links might prove useful to recall AND how recall must be adapted to use them
productively. Engram should end up at least as capable as the compound-engineering capture skill and
fire more autonomously. Vision-relevant-but-out-of-scope findings go in a followup doc.

**Stop-point interpretation (encoded, conservative):** steps 2 & 3 are approved *in principle*
contingent on #1, but their shape depends on which link concept wins — so the workflow ends at the
winners-conversation; 2 & 3 execute as a follow-on after Joe's go.

**Scope decision (named):** the memory-system review's exploration #6 posed a minimal *free* probe
("links stripped vs present"). This plan deliberately escalates to a designed multi-variant
exploration with an LLM-judged rigor round — because that is the ask ("don't just naively link
things… another eval round with more rigor"), and because the free strip-probe would measure Δ=0 by
construction (today's ranking never reads edges).

**Spend estimate (no cap; runs to completion):** fabric construction ~$5–15; PoC probes ~free
(binary replays + Python math); rigor round ~$20–60. Total ~$30–80.

## The evidence boundary this design honors

- **Settled null (note 73, 2026-06-23):** one-hop graph expansion of the matched set, on deep
  10-phrase recall, over the then-fabric = 0 marginal value — because cosine breadth *already
  co-surfaced* the bridge ("given the right notes co-surfaced, opus already composes"). T1×L1 below
  re-runs this ONLY as a harness-sanity control.
- **Open aspiration (note 68):** compositional/transitive/analogical retrieval needs a relational
  representation; `internal/vaultgraph` is the unused substrate.
- **Therefore the north star:** headroom exists only in the **miss population** — queries where
  cosine does NOT surface a needed note (glance's n=3 phrase floor is set by C6-badge, buried at
  n<3 — `2026-06-29-glance-delivery-661.md` Phase 1), buries it below the payload cut, or surfaces
  a superseded note without its superseder. Every variant is judged on recovery in that population.
- **Fabric reality (measured 2026-07-02, under `vaultgraph`'s exact-basename resolver — the actual
  substrate):** 135 notes; 84 directed wikilinks — **77 resolving, 7 broken** (3 slug-only
  prefix-drift links whose targets exist, fixable via the shipped `engram migrate-links`; 4 pointing
  at auto-memory files outside the vault, broken under any resolver and excluded from L1); 58% of
  notes isolated; one 45-note component. Today's fabric is too sparse to test traversal fairly — so
  fabric-construction variants are half the exploration.

## Stage S0 — Research (delegated; sonnet + web)

- Fetch https://every.to/guides/compound-engineering (WebFetch) + 2–4 adjacent primary sources.
- **Interrogate ALL SIX roles of their capture skill for link-criteria/traversal implications** (not
  just the finder): context analyzer (→ situation-framing edges? L4?), solution extractor (→ what
  gets linkable?), **related docs finder (→ their link kinds + how links are consumed at read
  time)**, prevention strategist (→ supersession/prevention edges? L5?), category classifier (→ tag
  taxonomy? L6?), documentation writer (→ frontmatter fields recall could key on?). Also: (b) how
  do their docs get FOUND in future sessions (search? tags? frontmatter?), (c) capture-quality
  practices we lack, (d) trigger model — how human-dependent is their fire, really?
- REUSE, don't re-research: `docs/design/artifacts/2026-07-01-memory-review-techniques-survey.md`
  already covers HippoRAG/PPR, GraphRAG, A-Mem, Generative-Agents scoring — the researcher reads it
  first.
- Outputs: (i) amendments to the variant catalog (the catalog is a prior, not a cage); (ii) the
  **followup doc** (`docs/design/2026-07-02-compound-engineering-followups.md`) of
  vision-relevant-but-out-of-scope findings — **drafted and committed at S0 completion** (parked ≠
  unrecorded, note 154), not held to the stop-point.

## The variant catalog (prior; S0 may amend)

**Link-criteria variants** (what edges exist; every fabric is a JSON edge-list on disk under
`dev/eval/links/fabrics/` — **the live vault is never written**):

| ID | Edges from | Cost | Rationale |
|---|---|---|---|
| L1 | status-quo fabric (**77 resolved edges, measured post-pre-step** — `migrate-links` handles bare-ID links, not the 3 slug-only drift links, which repaired 0; those 3 stay broken/excluded, deferred to the stop-point) | $0 | control |
| L2 | corpus-wide link-on-write: per note, top-K embedding candidates over the whole vault → **harness-local LLM pass replicating recall 2.6's GENERATE/JUSTIFY/PERSIST gate** (relation menu + shared-key + hub test) → surviving edges into the fabric JSON | ~$5–10 | the compound-eng "related docs finder" mechanism; fixes "only co-surfaced pairs ever considered" |
| L3 | shared rare tokens (TF-IDF-rare concrete tokens shared across notes) | $0 | note 153: the concrete token is load-bearing |
| L4 | situation-field cosine (embed situation handles only; near-family situations) | $0 | same-moment lessons should co-surface |
| L5 | supersession/temporal edges (LLM-typed: updates / narrows / refutes) | ~$3–5 | conflict surfacing; C4i as persistent structure |
| L6 | tag/category taxonomy (controlled vocabulary per note; hub-by-design, used for filter/discovery not ranking) | ~$2–4 | compound-eng classifier; tests whether hub-kill overcorrects for discovery |
| L7 | provenance/episode edges (same origin session/source) | $0 | cheap temporal cohesion |

**Traversal variants** (how recall consumes edges) — each carries **"recall changes required"**, a
first-class viability factor in S2 pruning (heavy adaptation weighs against a marginal recovery win):

| ID | Mechanism | Recall changes required | Claim it tests |
|---|---|---|---|
| T1 | one-hop payload expansion of matched notes | binary: expansion pass post-union | control — must reproduce the null on L1 |
| T2 | PPR/spreading activation seeded by matched set; score = α·cosine + β·ppr | binary: graph-aware scoring stage (largest change) | HippoRAG-lite associative recovery |
| T3 | neighbor rank-boost (edges never ADD items, only re-rank buried/below-floor notes upward) | binary: score-adjust pass; payload shape unchanged | recovery without payload growth |
| T4 | candidate_l2s enrichment (neighbors join CANDIDATE nomination for 2.5 coverage only) | binary: candidate nomination only; skill unchanged | better dedup/coverage at zero payload cost |
| T5 | typed-selective traversal (follow ONLY supersession/contradiction edges; superseder rides along with any matched superseded note) | binary: small typed lookup; skill: 2.5B consumes the flag | conflict-correctness generalized |
| T6 | glance-breadth substitute (under 3-phrase glance, one-hop from top matches) | binary: conditional on phrase-count; glance skill text unchanged | links as cheap breadth-recovery for the frequent rung |

Not every L×T cell runs — the S2 matrix, exactly: **L1×T1** (the settled-null control, 1 cell);
**{L2, L3, L4, L7} × {T1, T2, T3, T6}** (16 cells — L2 is LLM-built, L3/L4/L7 mechanical); **L5×T5**
(1); **L6 × filter-lookup** (tag-match adds candidates, T4-style; 1). **Total = 19 scored cells.**
T4 itself rides along free: wherever a cell recovers a note, report whether that note would also have
entered candidate_l2s nomination (the T4 claim) — no separate cells.

## Stage S1 — Build the miss population + fabrics

- **P1 real-query misses:** replay phrase sets from real recorded recalls (session transcripts +
  saved payloads) at n=3 (glance) and n=10 (deep) via the real binary; a sonnet judge sweeps ALL
  135 notes per query and emits `{query_id, missed_note_basename, why_needed}` — needed = the
  note's lesson bears on the query's task and it is absent from that replay's delivered output. A
  miss is recorded per (query, n) pair — glance-misses (n=3) and deep-misses (n=10) are tallied
  separately, never conflated.
- **P2 constructed bridges:** C6-badge-style two-hop cases from the real vault (~6–10).
- **P3 supersession pairs:** known pairs (120→153; the 350s→190s mislabel; 82's fixed leg; qanchor
  park) — query matches the OLD note; does the NEW one surface?
- **S1 pre-step:** run `engram migrate-links` to repair the 3 prefix-drift links, then re-measure
  and record L1's edge count. This is the ONE sanctioned vault write this round — a shipped
  maintenance command fixing pre-existing drift, adding no experimental content (named exception to
  the no-vault-writes constraint).
- Build L2–L7 fabrics; report fabric stats (edges added, degree distribution) — no silent caps.
- **Gate S1 (pre-registered):** if P1+P2+P3 yield **< 8** distinct real miss cases, STOP EARLY and
  report — a thin miss population is itself the finding.

## Stage S2 — PoC probes + prune

- **Mechanics (pinned):** the harness shells out to the real `engram query` for each case's ranked
  list + scores (the `retrieval_probe.py` pattern), then applies each traversal's math in Python
  over the fabric JSONs + sidecar `.vec.json` vectors (384-dim, verified present for 135/135 notes;
  chunk vectors live in the index .jsonl). No phrase-embedding capability is needed or assumed.
- Metrics per L×T cell: **recovery@10 (primary; @5 and @20 reported)**, rank movement, payload
  delta (items added), and **collateral** on a 20-query no-regression set.
- **Prune rules (pre-registered):** kill any cell with (a) recovery@10 ≤ its control, (b) payload
  growth > +20% without recovery gain, (c) **collateral regression = ≥1 no-regression query where any baseline top-5 note (kind
  fact/feedback) falls out of the top-5** — no relevance judging; displacement of any baseline
  top-5 note counts (conservative pre-registration; S3 can exonerate a survivor if the displacer
  proves better), or (d) recovery wins achievable only with
  a "recall changes required" burden comparable to or heavier than T2's while recovering no more
  than a lighter variant. Report the kill list with numbers.

## Stage S3 — Rigor round (survivors; LLM-judged)

- Per survivor (expect 2–4): delivery-style blind eval on recovered cases — variant payload vs
  control payload injected into a task prompt; opus judge scores whether the answer APPLIES the
  recovered note's knowledge (qanchor method; name-agnostic judge).
- Reps ≥ 2 per case per arm; verdict per the 2σ rule (sub-noise gap = "can't distinguish", never a
  win).
- **Iterate trigger (pre-registered):** re-iterate ONCE if ≥ 2 survivors are sub-noise on the same
  population; mutation/hybridization of the top-2 only; **no return to S2**. Then stop.

## Stage S4 — STOP: present winners to Joe

The conversation covers, per winner (or the honest null):
1. Recovery + delivery numbers (labeled tables, units, n, dates).
2. **Recall-adaptation cost** — the concrete binary/skill changes its traversal needs (from the
   T-table column), so implementation weight is on the table, not deferred.
3. **Parity statement vs compound-engineering** (qualitative, honest bounds): given S0's analysis,
   does the winning concept + existing engram close the capability gap with their 6-role capture
   skill — and where does it still fall short?
4. **Autonomy implications** (from S0(d)): does the winner change how engram should FIRE (trigger
   model), and what routes to Track A (decision-moment hooks) vs this workstream? Explicitly routed,
   not silently parked.
5. The implied shape of steps 2 (which link kinds the learn template should prompt for) & 3 (which
   fabric builder runs for real). Steps 2 & 3 execute only after this conversation.

## Deliverables

- `dev/eval/links/` — fabric builders + probe harness + results JSONs (pure scoring logic
  unit-tested, qanchor pattern: `qanchor_score.py`/`test_qanchor.py`).
- `docs/design/2026-07-02-link-value-exploration.md` — results doc (labeled tables with units,
  dates, vintage flags if the system changes mid-exploration).
- `docs/design/2026-07-02-compound-engineering-followups.md` — committed at S0 completion.
- This plan (committed before execution).

## Constraints

- Isolated harness only; fabrics are JSON on disk; **the live vault is never written** this round —
  with the single named exception of the S1 `engram migrate-links` pre-step (shipped maintenance
  repairing 3 pre-existing prefix-drift links; no experimental content).
- T1×L1 is a control, not a candidate — the settled null must reproduce or the harness is suspect.
- Results as labeled tables with units + n; every number dated.
- Pre-registered gates and prune rules above — no post-hoc survivor selection.
- No repo-wide tooling; scope-check the diff before every commit.
