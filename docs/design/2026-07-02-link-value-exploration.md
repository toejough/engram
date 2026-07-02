# Link-value exploration — results

**Ask (Joe, 2026-07-02):** prove/refute link worth with a designed exploration — research the
landscape, postulate link-criteria × traversal variants, PoC → prune → rigor → 1–2 winners → stop.
Plan (+ Gate-A trail, pre-registered priors and prune rules):
`docs/superpowers/plans/2026-07-02-link-value-exploration.md`. Research followups (4-beat sweep):
`2026-07-02-research-followups.md`. Harness + data: `dev/eval/links/`. **Round discipline:** isolated
harness only — no production binary/skill changes; fabrics are JSON on disk; the live vault was never
written (even the sanctioned `migrate-links` pre-step was a measured no-op — it targets bare-ID
links, not the 3 slug-drift links, which stay broken/excluded).

**Verdict in one line: links pay at recall through exactly one tested mechanism — controlled-vocab
tag NOMINATION (L6×TAG), which delivers cross-domain bridge knowledge (+50pp on P2, above noise)
that cosine structurally misses; ranking-side traversal (PPR, rank-boost, one-hop) does not pay on
this vault.**

## What ran (system state: 135→136-note vault, note-floor ranking, glance/deep recall; 2026-07-02)

1. **S0 research** — 4 beats (compound-engineering; peer systems' linking; PKM practice; IR/KG
   literature + ecosystem tools). Priors pre-registered in the plan before any probe.
2. **S1 miss population** — 48 cases where recall demonstrably misses a needed note: P1 = 36
   judge-marked misses from 25 REAL recorded query replays (n=3 and n=10 tallied separately);
   P2 = 8 constructed two-hop bridges (verified-missing; all 8 UNLINKED in today's vault); P3 = 4
   verified supersession misses (old surfaces, superseder absent; 3 non-miss pairs recorded as
   negative evidence). Plus 7 fabrics: L1 status-quo 77 edges; L2 LLM-gated corpus-wide 212
   (isolation 80→17); L3 rare-token 936; L4 situation-cosine 225; L5 supersession-typed 25;
   L6 25-term controlled-vocab tags (136/136 tagged); L7 provenance 412.
3. **S2 probe matrix** — 19 pre-registered cells + the S2b harness correction (chunk-pinned
   note-only re-ranking — the plan's own T3 spec). 23/23 unit tests. Commits `96f6185c`, `608f367e`.
4. **S3 delivery rigor** — blind name-agnostic opus judge; arms A (baseline payload) vs B
   (+variant additions); 116 valid trials: 104 L6×TAG (26 recovered (query, needed-note) units
   across 14 unique queries, × 2 reps/arm) + 12 L5×T5 smoke (2 units × 3 reps/arm).

## S2 — retrieval recovery (48 miss cases; recovery@10 primary)

| Cell (survivors + control) | r@10 | Δpayload | collateral regressions | verdict |
|---|---|---|---|---|
| L1×T1 (control) | 0.0% | +22% | 0 | settled null REPRODUCED — harness sane |
| **L6×TAG** | **54.2%** (26/48) | +43% (nomination pool, not delivered payload) | **0** | **SURVIVOR — headline** |
| L5×T5 | 4.2% | +3.9% | 0 | survivor (mechanism exact; fabric only 25 edges) |
| L3×T1 / L3×T3 / L7×T3 | 2.1% each | +131–283% | 0 | marginal (1 boundary case each; T4 nomination reach 65–88%) |
| all T2 (PPR) cells | best 14.6% (L4 τ=0.6) | −29 to −89% | 32–36 | KILLED — pure PPR drops non-activated baseline notes (a traversal property, verified note-vs-note after chunk-pinning) |
| all other cells | 0% | — | 0–7 | KILLED (recovery ≤ control) |

The T1×L1 null reproduced exactly as pre-registered (and as HippoRAG's ablation predicted). T6
(glance-breadth one-hop) added nothing on any fabric. L2's richer fabric fixed isolation but its
edges point where cosine already reaches. **Only two mechanisms recovered misses with zero
collateral: tag-pool nomination and typed supersession ride-along.**

## S3 — delivery (does the recovered note change what the agent DOES?)

**Contamination handled per the degraded-build lesson:** the first S3 run spanned a session-limit
window — 40 trials returned zero-cost empty answers (auto-MISS, both arms equally, all in P1). All
40 were discarded and re-run after the window reset; the final 116 trials contain 0 degraded
entries. (An interim writeup based on the contaminated tally concluded "P1 delivers nothing, the
overall win is +12pp on bridges only" — partially superseded: the clean re-run kept the bridge
result and softened the P1 zeros to a within-noise positive trend, lifting the overall win to
+17.3pp.)

| L6×TAG | A: baseline payload | B: + tag-nominated pool | B−A | 2σ | verdict |
|---|---|---|---|---|---|
| **Overall** (26 units × 2 reps/arm) | 7.7% (4/52) | **25.0% (13/52)** | **+17.3pp** | 14.1pp | **B WINS — above noise** |
| P2 bridges (5 tag-recovered of 8) | 10% (1/10) | **60% (6/10)** | **+50pp** | 36pp | **B WINS — the carrier** |
| P1-n3 | 9% (2/22) | 18% (4/22) | +9pp | 21pp | within noise |
| P1-n10 | 6% (1/16) | 13% (2/16) | +6pp | 20pp | within noise |
| P3 supersession | 0/4 | 1/4 | +25pp | 43pp | within noise (n=4) |

Per-case texture (P2): 3 of 5 tag-recovered bridges delivered **2/2 with the pool and 0/2 without**
(B7 calibrate-defaults, B8 metric-conflation, B9 synthesis-persistence) — repeatable, not flukes.

| L5×T5 smoke | A 2/6 | B 1/6 | **UNDERPOWERED (n=2 cases) — mechanism smoke only, no delivery verdict** |
|---|---|---|---|

**The P1 pattern is itself a finding:** the sonnet catalog-judge marked 36 notes "needed", but even
delivered in-payload they moved delivery only within noise — "bears on the task" over-matches
relative to "changes what the agent does" (note 119's proxy-vs-delivery lesson, reproduced inside
our own eval). Links genuinely pay in the **bridge population**: cross-domain lessons whose
vocabulary is remote from the task's phrasing — exactly what the AAR prior predicted ("association
is not similarity") and Mem0g's ablation called the win zone. Honest caveat: P2 is a CONSTRUCTED
population (authored bridge cases, binary-verified as real misses); the P1 real-query population
shows only a within-noise positive trend.

## Winners (1–2, per the ask)

1. **WINNER — L6×TAG: controlled-vocabulary tag nomination.** 54.2% retrieval recovery, zero
   collateral, delivery +17.3pp overall / +50pp on bridges, both above noise. Lightest
   recall-adaptation burden of any variant: tags are frontmatter + a fabric file; nomination
   extends `candidate_l2s` (binary: tag-match candidates join nomination; the skill's Step 2.5
   already reads candidates). Whole-vault vocabulary assignment cost $1.43.
2. **RUNNER-UP (concept proven, fabric-starved) — L5×T5: typed supersession ride-along.** The
   canonical case works end-to-end (old note surfaces → edge inserts superseder at rank 5), zero
   collateral, +3.9% payload; but 25 edges reach only 2/48 misses and the delivery smoke is
   underpowered. Carry it inside step 3's sweep (grow the fabric, re-smoke) — not as its own build.

**Killed with evidence:** PPR/spreading activation (drops baseline notes — traversal property, not
harness artifact; no ecosystem tool runs it at query time either); one-hop expansion
(field-confirmed harmful); glance-breadth links (T6 zero on every fabric); rank-boost at safe
weights; situation-cosine and provenance fabrics as delivery levers.

## Parity vs compound-engineering + autonomy (S4 items 3–4)

Their system does NO recall-time graph traversal — flat LLM judgment over structured frontmatter;
their "related docs finder" is write-time dedup. Our winner is the most compound-engineering-shaped
variant we tested — **controlled-vocab tags in frontmatter used for discovery** — but consumed
mechanically (nomination) rather than by an LLM reading every doc, which scales better. With L6
nomination shipped, engram matches their capture-side discovery and exceeds their retrieval
mechanics; remaining gaps are capture-side (two-track schema, write-time dedup — followups report)
and trigger-side. Autonomy: their recall fires automatically inside `/ce-plan`; ours is
human-invoked — that routes to Track A (decision-moment hooks), independent of this result.

## Honest ledger + bounds

- **Spend:** S1 $16.38 (miss-judge $7.36 + LLM fabrics $9.02) + S3 valid trials $152.39 + S3 sunk
  (two runs killed in the session-limit window) $54.16 = **$222.93** vs the plan's $30–80 estimate
  (~2.8× over; drivers: delivery-arm payload sizes at $0.65–2.00/trial, and the lost runs). S2
  probes cost $0 as designed.
- **Bounds:** one vault (136 notes), one user's queries. P2's +50pp clears its own 2σ but rests on
  5 cases × 2 reps, and P2 is constructed (though binary-verified). P1/P3 effects are
  unresolved-within-noise, not ties. L5×T5 unpowered. The L6 vocabulary was LLM-derived once —
  drift under vault growth untested. Delivery arms were injected payloads, not full recall
  sessions — production delivery must be re-gated (trap suite C3–C6 + no-regression replay) when
  the mechanism ships.
- **Vintage:** all numbers 2026-07-02, post-floor post-glance system, fabrics from the
  135/136-note snapshot.

## Implied shape of steps 2 & 3 (pending the stop-point conversation)

- **Step 2 (learn habit):** the evidence favors TAGS over free edges — learn should assign
  controlled-vocab tags at write time (plus a supersession relation when a new note corrects an
  old one), not generic related-to links.
- **Step 3 (retroactive sweep):** persist the L6 vocabulary + assignments into note frontmatter as
  the durable fabric (one-time ~$1.50, re-runnable); optionally grow L5 edges in the same pass.
  The 77-edge wikilink graph stays — its recall value is unproven, its authoring/navigation value
  stands.
- **Binary:** extend candidate nomination with tag-match (small, DI-clean); gate with C3–C6 traps
  + a no-regression replay before ship.
