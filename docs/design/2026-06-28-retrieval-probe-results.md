# Retrieval probe — is engram's vectorizing good enough for nuanced semantic search?

> **Question (Joe, 2026-06-28).** "Semantic search is effectively our core memory use case. If engram's
> vectorizing can't handle subtle semantic meaning, I'm concerned about its overall usefulness." This probe
> answers it with measurement. Data trail: `2026-06-28-retrieval-probe-data/`.

## Verdict

**The embedder is NOT the weak link — the ranking architecture is.** MiniLM-L6 surfaces the right
crystallized note for a *paraphrased, low-overlap, cross-domain* situation cue **81% in the top-5 / 92% in
the top-10** when measured in isolation. But in the real unified query that drop to **19%**, because the
chunk index **drowns the notes**: the crystallized lesson scores *lower cosine* than verbose raw transcript
chunks on the same topic, so notes occupy only ~2% of the top slots the agent sees. **Do not swap the
model. Fix the note-vs-chunk ranking** — that is where engram's retrieval value is being lost.

## Method

| Item | Value |
|---|---|
| Probe set | **36 vault notes** (27 abstract principles, 9 narrow/impl-specific), ids 24–92 |
| Per note | a **nuanced query** (concrete future situation, paraphrased, low lexical overlap, cross-domain) + a **lexical control** (reuses the note's terms) |
| Query validation | haiku generated, **sonnet validated** each nuanced query genuinely maps to its note AND is genuinely low-overlap (0 dropped, 6 fixed) |
| Component under test | the **real `engram query` binary** + the real MiniLM embedder (no re-implementation) |
| **Isolation path** | `ENGRAM_CHUNKS_DIR=<empty>` → notes-only ranking = the embedder's note-matching, no chunk competition |
| **Real path** | full chunk index → ranking among ALL items = what the agent actually sees |
| Metric | target note's rank → recall@k, MRR, miss rate (miss = not in the surfaced/relevance-floored output) |

## Result — the decomposition

| path | query | r@1 | r@5 | r@10 | MRR | miss |
|---|---|--:|--:|--:|--:|--:|
| **isolation (MiniLM, notes-only)** | lexical (control) | 0.86 | **0.94** | 0.97 | 0.90 | 0% |
| **isolation (MiniLM, notes-only)** | **nuanced** | 0.36 | **0.81** | 0.92 | 0.54 | 8% |
| real-path (all items, agent sees) | lexical | 0.72 | 0.75 | 0.75 | 0.74 | 25% |
| real-path (all items, agent sees) | **nuanced** | 0.17 | **0.19** | 0.19 | 0.18 | **81%** |

Two separable effects:

- **Nuance tax (the embedder's own limit) — modest.** Isolation, lexical→nuanced recall@5: 0.94 → **0.81**
  (−0.14). The embedder often doesn't rank the *exact* note **first** (recall@1 0.86→0.36) but reliably puts
  it in the **top-5/10**. The 8% genuine misses are all the most abstract conceptual notes (aggregation-vs-
  synthesis, multiphrase-subsumes-graph, task-displacement) — verified as fair queries, not rigged.
- **Structural drowning (the ranking architecture) — severe.** Nuanced isolation→real-path recall@5: 0.81 →
  **0.19** (−0.61, **4× the nuance tax**). The note MiniLM ranks top-5 in isolation is buried under chunks in
  the real query. Across 10 drowned notes, the top-8 returned items are **2% notes, 98% chunks**.

By note type (isolation, nuanced recall@5): abstract 0.74, narrow 1.00 — the embedder's only real weakness is
the most abstract concept→situation matches, and even there recall@10 is 0.89.

## Why notes lose: terse lesson vs verbose chunk

A crystallized note's `situation_vector` is a short, distilled phrase. A raw chunk (e.g. a subagent's
dispatch prompt discussing the same topic) is verbose and lexically rich, so it scores **higher cosine**
against a natural-language situation query (drowning chunks scored 0.62–0.69; the matched notes score lower).
The unified ranking mixes two populations whose cosine magnitudes aren't comparable, and the verbose
population wins. This is **note 82 ("recall-miss-is-structural-not-retrieval") quantified at scale.**

**Confounds ruled out:** (a) the drowning is **89% pre-existing chunks**, only 9% this-session's
failure-mining chunks (index = 3,135 source files; this session = 1) — steady-state, not contamination.
(b) the lexical control hitting **0.94 isolation / median rank 1** proves the harness + embedder + scorer
work — the nuanced drop is real signal, not a broken pipe. (c) the isolation misses are fair (valid mapping,
low overlap), so the 8% is honest embedder limit, not query rigging.

## Is the ranking fix tautological? The value test

"Notes are drowned" ≠ "the right knowledge fails to surface" — a chunk on the same topic might carry the
knowledge. Prioritizing notes *qua* notes would be circular. So a second test measured **outcome**, not
note-rank: for the 22 situations where a relevant note was drowned, run an agent in three conditions and
blind-judge whether each plan **applies the lesson / avoids the failure** (0–2):

- **none** — situation only (the model's prior)
- **chunks** — situation + the real recall payload today (the drowning chunks; the note excluded)
- **chunks + note** — same, plus the de-drowned note

| knowledge source (cheapest sufficient) | count | share | meaning |
|---|--:|--:|---|
| prior (model already knew) | 8 | 36% | note redundant here |
| chunks (real recall delivered it) | 4 | 18% | recall already works |
| **NOTE-ONLY** | 9 | **41%** | neither priors nor chunks delivered it — only the de-drowned note did |
| nobody (note insufficient) | 1 | 5% | capture ceiling |

| condition | mean score (0–2) | success (=2) |
|---|--:|--:|
| none (prior) | 1.14 | 36% |
| **chunks (real recall today)** | **1.23** | 36% |
| chunks + note | **1.91** | **95%** |

**The fix is not tautological — two measured facts make it:**

1. **The note delivers knowledge nothing else does, 41% of drowned cases** (≈25% of all relevant-note cues,
   given the 61% drown rate). Judged by plan *behavior*, not note-rank — e.g. on the recency-pipeline note
   the note-plan named "applyChunkRecency + sortScoredDesc on `[]scoredChunk` before the item-build loop"
   while the others "addressed an entirely different problem." Genuine, not parroting.
2. **The chunks doing the drowning score like noise.** Mean chunks 1.23 ≈ prior 1.14 (+0.09) — the real
   recall payload adds almost nothing over the model's own knowledge (one situation surfaced a
   *PBT-survey prompt* for a "swap the calculation engine" task). De-drowning displaces low-value chunks.

So a relevant crystallized lesson is the *sole* source of needed knowledge ~25% of the time and is buried
under chunks that behave like noise. **Bound (honest):** 54% of drowned cases were redundant (prior or
chunks sufficed), so this is not "notes always win," and it does NOT say a chunk can never be the right
answer (when the agent needs a specific past decision, the chunk *is* the knowledge — those weren't tested).
The clean claim: **where a relevant note is drowned by noise-chunks, surfacing it recovers knowledge nothing
else supplies, at little cost.** N=22 (9 note-only), one vault.

## Recommendation

1. **Do not swap MiniLM, and do not run the stronger-embedder comparison.** At 81% top-5 / 92% top-10 on
   nuanced cues, the embedder is adequate; a stronger model would optimize the part that already works while
   the broken part (ranking) is model-independent. (Reverses the earlier "compare models" option.)
2. **Fix note-vs-chunk ranking — now justified on *knowledge*, not notes-qua-notes.** The drowning loses
   sole-source knowledge ~25% of the time while the displaced chunks behave like noise. Frame the fix as
   *"a high-relevance note must not be buried under lower-value chunks,"* NOT "notes always rank above chunks."
3. **This reframes the ROADMAP.** Crystallized lessons — engram's product — are ~2% of the top slots in real
   recall. No payload-size or procedure-time lever matters as much as making relevant notes visible.

## First gated experiment — matched-note floor (SHIPPED 2026-06-28)

**Result: real-path note recall@5 0.222 → 0.833 — the embedder's isolation ceiling (drowning eliminated).**
(The 0.222 baseline is the §Result table's 0.19 *re-measured on the current, larger chunk index* just before
the change — same metric, fair before/after pair; the index grew between the original probe and this build.)
`capWithNoteFloor` in `mergePhraseIntoUnion` reserves up to `noteFloorK=5` per-phrase slots for notes that
already clear the relevance floor (commit `33821e64`). Trap gate C3–C6 GREEN (no win regression); payload
neutral-to-smaller (notes swap for chunks within the existing 30-cap). The crystallization audit
(`2026-06-28-crystallization-audit.md`) is the follow-up: the floor makes a good note surface, but ~half of
cluster-driven notes are not question-useful, so **question-shaped crystallization** is the next lever (⛔ that
lever was evaluated and **PARKED** 2026-07-01 — `docs/design/2026-07-01-question-anchored-distillation.md`
§Results: no delivery benefit (tie within noise) + a clear retrieval loss; the wording gap is delivery-inert). The
chunk-down-weight (Track 2) was deferred — the floor saturated the note-recall gauge, so it needs its own
chunk-quality gauge before it's worth shipping.

The spec as proposed (now realized):

- **Change:** in `engram query`'s merge, after the relevance floor, **reserve slots for the top-K_note notes
  that already cleared the floor** (e.g. K_note=5), so a *relevance-qualified* note is never fully drowned by
  higher-cosine chunks. (Notes below the floor are NOT force-surfaced — it stays relevance-gated, so it can't
  surface irrelevant notes.) Principled alternative for the follow-up: **per-population score normalization**
  (z-score notes vs chunks before merge) — root-cause but harder to tune; do the floor first.
- **Primary metric — the probe is the regression harness.** Re-run `score_probe.py`: real-path note
  recall@5 should climb from **0.19 toward the 0.81 isolation ceiling**. That number is the pass/fail.
- **Gate (must-not-regress):** trap harness `dev/eval/traps/gate.py` C3–C6 GREEN before+after (a note floor
  must not break chunk-evidence-dependent wins); `recall_cost` `$METER` (≈5 short notes is a small payload
  add — confirm it doesn't balloon); lazy-chunks payload-size unchanged.
- **TDD:** RED = an integration test asserting a known-relevant note surfaces in the real-path top-K for a
  query where it currently drowns (reproduce a probe miss as a unit/integration test); GREEN = the floor;
  verify with the probe harness + the trap gate.
- **Risk/bound:** displaced chunks are low-value here (measured), so the floor is low-risk — but the trap
  gate (which leans on chunk evidence) is the backstop; if any C3–C6 axis regresses, the floor is displacing
  a *needed* chunk and must be retuned (lower K_note, or switch to normalization).

## Honest limits

- **N=36, single vault.** The contrasts are large (0.81 vs 0.19 is 4× the noise-scale nuance tax), but this
  is one vault of 94 notes. Re-run on a larger/again-crowded vault before treating the exact numbers as fixed.
- **Relevance-floor metric.** A "miss" means the note fell below engram's surfaced/floored output (~11–48
  items) — i.e., the agent wouldn't see it. That is the real behavior, but it means ranks deep in the tail
  aren't resolved (recall@10 is the deepest reliable cut).
- **Isolation uses an empty chunk index**, which is not a shipped mode — it's a measurement instrument to
  isolate the embedder. The real path is the agent's experience.
