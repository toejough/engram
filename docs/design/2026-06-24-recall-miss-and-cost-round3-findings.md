# Findings — the recall miss + do-both cost reduction (round 3)

**Date:** 2026-06-24 · **Status:** findings (investigation complete; implementation deferred to issues)
**Plan:** `2026-06-24-recall-miss-and-cost-round3-plan.md` · **Method:** Gate-A-reviewed plan → 5-angle
fan-out (miss RCA, cost audit, do-both, TDD design, ledger) → synthesis → 3 adversarial verifiers. All
verifiers returned *holds / holds-with-caveats*; the anti-amnesia verifier confirmed no closed lever is
re-proposed. Caveats are folded in below.

## TL;DR

1. **Why the memory didn't surface — it was a *keying/timing* miss, not a retrieval miss.** The lever
   ("cheaper-tier recall") did not exist at recall time; it was invented during synthesis, *after* the
   single pre-work recall pass. Recall fires once, keyed to the diagnostic ask, and **never re-fires
   against a recommendation the agent generates.** Note 80's situation vector ("splitting a skill across
   models to cut cost") has near-zero cosine to the diagnostic phrases — it returned **0 hits in the
   entire diagnostic-keyed result set** (300+ items) under the original phrasing, yet ranks **#1 among
   notes** under lever phrasing.
   Compounding it: crystallized notes are **outranked by raw chunks** (and the chunk index keeps
   growing), and there is **no recency safety-net for notes** (the recent-activity channel is chunks
   only — verified in `internal/cli/query.go` `buildRecentFillItems`).
2. **Expensive parts — and keep the axes separate.** On **time**, recall+learn is the *majority* of the
   warm op (411 s of 615 s); the dominant slice is recall's **Step 2.5 per-cluster loop** (blocking
   `engram show` round-trips + coverage-judge reasoning + blocking writes), and safe time cuts exist
   (O2, L2, L3a). On **dollars**, recall is a *sub-share* — a −61% payload cap moved end-to-end $ by
   nothing — **but** recall's $ is *bundled into* `build_cost` (recall runs inside the build session),
   so "the build is the dollar sink" is partly an accounting artifact. Two distinct goals: the
   **recall+learn procedure** is what makes warm cost *more than cold* (the premium you'd cut to make
   memory pay for itself); the **build loop** is the largest *absolute* spend (the lever for total $,
   warm or cold) — they are not the same lever, and conflating them is the trap.
3. **Do-both? Partial yes, honestly.** The performance fixes (note-negation override, reconcile-
   proposals, recall-before-recommend) are **cost-neutral**, and the safe cuts (O2/L2/L3a) are real
   time savings — so you can *ship both*. But **no single change is a measured win on both axes**, and
   the only fix that catches *this* miss regardless of phrasing (recall-before-recommend) costs **one
   added query round-trip** — an honest trade-off, not a free lunch.
4. **How to measure it:** a new criterion **C7 "lever-recheck" (anti-amnesia)** that reproduces this
   exact miss as RED and gates every cost option. Critically it needs a **diagnostic sub-metric** (did
   the disproving note surface in recall at all?) to avoid falsely passing a synthesis-only fix while
   the retrieval gap remains.

## 1 · Why the memory didn't surface (RCA — verified)

| Suspect (user's framing) | Severity | Mechanism (verified against live vault + source) |
|---|---|---|
| **Triggers** | PRIMARY (structural) | Recall fires once (please Step 2 Orient), keyed to the incoming ask (recall states its own upfront plan at its Step 0). No *recall-before-recommend* re-entry. recall Step 3 "walk[s] the plan from Step 0" — reconciles the **plan**, never the **recommendations synthesis produces**. |
| **Phrasing** | PRIMARY (mechanical) | Note 80 situation = "split a skill across models for cost"; original phrases were diagnostic. Cosine too low → **0 hits in the whole result set**; same note **#1 among notes** under lever phrases. The 10-angle generator never emits a "lever I may recommend / prior verdict" phrase at a diagnosis turn. |
| **Evaluation** | CONTRIBUTING | The disproof (`EXPERIMENT-LOG`: −14%/op, ROLLED BACK) was **in-context the whole time** yet never reconciled — synthesis evaluated the plan, not the emergent lever. In-context presence is necessary-not-sufficient (the lesson note 81 crystallizes). |

**Salience amplifier (verified in code):** `capChunkContent` caps only `chunk`-kind items (notes
uncapped — good), but `buildRecentFillItems` operates *only* over `chunk.Record`, so a same-day
disproving **note can never float by recency** regardless of phrasing. As the chunk index grows (this
very session's transcripts got ingested), chunks increasingly drown notes — note 80/81 sit under a wall
of session chunks at 0.78–0.87. Recall Step 3 has **no priority rule for a note that carries negation**.

## 2 · Expensive parts + safe cuts

Per-phase **wall time** is cleanly segmented (capped opus n=5): recall **350 s** + build **204 s** +
learn **61 s** = warm **615 s / $3.78** vs cold build **288 s / $2.06**. The build is *faster* warm
(204 s < 288 s); recall+learn (411 s) is the time premium.

Per-phase **dollars are only partly separable.** `/recall` runs **inside** the build session (the build
agent invokes it before writing code — `harness.py`), so its dollars are **bundled into `build_cost`**;
there is no `recall_cost` field. Only `learn_cost` (a separate session) is separable. So `build_cost`
*contains* recall — "the build is the dollar sink" is partly an accounting artifact, and recall's true
$ share is an **inference** (its ~49 K input tokens vs the build's generation output), not a
measurement. **Prerequisite for any $-axis lever:** *unbundle* recall's $ from the build session
(per-turn token segmentation or a separately-metered recall call) — not merely "log a per-phase field."

| Cut | Where | Axis | Effect | Quality risk | Status |
|---|---|---|---|---|---|
| **O2** inline candidate-note content → 0 `engram show` calls | recall Step 2.5 | time | ~3–8 fewer round-trips (~15–40 s) | low | open |
| **L2** skip Step 2.5 on chunk-only clusters (no note members) | recall Step 2.5 | time | removes dead deliberation | low (provably no-op there) | open |
| **L3a** batch learn **ingest sweep** once/session | learn Step 1 | time | trims part of 61 s/cycle | low *(see caveat)* | open |
| O1 content-budget toward notes (shipped, default 15) | query render | tokens | trims chunk bytes, nudges note salience | low–med | shipped |

*Axis = which cost axis the cut moves: time (seconds saved) / tokens (LLM tokens) / $ (USD). Recall's
per-phase $ is bundled into `build_cost` (not separable today), so no recall-$ figures are claimed
here. Quality risk: low / med / high.*

**Caveat (verifier):** L3a's "−61 s/cycle" overstates. Only the **ingest sweep** is safely batchable;
deferring **crystallization** is risky — it writes the very closure notes (80/81) a later recall depends
on. Batch the sweep; do **not** defer the note-write.

**Bounding truth:** recall is a sub-share of op *dollars* (the −61% payload cap moved end-to-end $ by
~nothing) — though it is the *majority* of op *time*. The largest *absolute* dollar bucket is the
**build loop** (L4/L5), but cutting it lowers warm *and* cold and does not shrink the warm-over-cold
**premium** (that premium is the recall+learn procedure). Aiming any build-loop $ lever is gated on
first **unbundling** recall's $ from `build_cost`, and L4 needs n≥15–20 on opus (the sonnet n=5 A/B was
variance-collapse-only, mean inside noise).

## 3 · The do-both answer (honest)

**Partial yes.** Two honest categories, both shippable together:

- **Performance fixes that are cost-neutral** (not measured cost *wins* — cost-neutral): note-negation
  override, reconcile-proposals sub-rule, please-gate keying. These raise correctness with negligible
  new I/O. Verifier's correction: call these **quality-axis with a cost-neutral guarantee**, not
  "do-both" — reserving "do-both" for a *measured* cost delta keeps us honest.
- **Real (sub-share) cost cuts:** O2/L2/L3a above.

**The load-bearing fix is a trade-off, and we keep it:** the **recall-before-recommend micro-query** is
the *only* fix that catches note 80 under its own vocabulary when the diagnostic-keyed recall scored it
0. It costs one added blocking query per invented lever (rare per session). We refuse to relabel it
do-both to make the answer prettier — per note 81's own lesson.

So: *can we remove overhead while improving performance?* — **the performance improvements don't add
overhead, and the overhead cuts don't cost performance, so you ship both** — but you will not find a
single change that the data shows moving both axes, and none of it touches the build-loop $ axis where
the real warm-vs-cold gap lives.

## 4 · How to measure the miss — C7 "lever-recheck" (anti-amnesia)

**Metric:** per option the agent recommends, if it matches a **closed** lever present in vault/context,
does the recommendation text (1) acknowledge the prior attempt + outcome and (2) drop it or justify
revisiting against new evidence? Score 0/1 per proposed-closed-lever; **RED today = re-proposes it as
fresh (0/1)** — verified: recall/learn/please have no per-recommendation reconciliation rule anywhere.

**Harness** (`dev/eval/cumulative/lever_recheck/`, mirrors `synthesis_fixtures/`):

- `vault_with_closed/` seeds note 80 + 6–10 distractors (forces clustering); `context.md` is an
  `EXPERIMENT-LOG` excerpt that **states the −14%/rolled-back numbers** but does **not** re-summarize
  L1's closed status in its synthesis section (forces reconciliation, not reading a verdict off the
  page); `task.txt` asks for the single highest-leverage warm-build cost cut — whose natural answer
  *is* the closed lever.
- **Paired `vault_open/` control** (note absent) proves the agent freely proposes the lever there →
  a pass means "reconciled when it had the evidence," not "never mentions the lever." Defeats the
  degenerate always-pass scorer.
- **Scorer** `lever_recheck_scorer.py`: adversarial LLM judge (majority-of-3, refute-by-default,
  default-AMNESIA), reusing `synthesis_judge.py` plumbing. Judges **meaning vs `closed_levers.json`
  ground truth, not the note's literal words** (heeds the scorer-vocabulary-bias lesson). Real
  `/recall` (or `/please`) runs — the recommendation is the **measured output**, not bypassed.

**Required diagnostic sub-metric (verifier's load-bearing caveat):** C7 measures the *end*
recommendation, conflating **retrieval-failure** (note never surfaced — the real note-80 case) with
**synthesis-failure** (note in-context but ignored). Without a *"did the disproving note surface in
recall output at all?"* sub-metric, a synthesis-only fix (note-negation override, reconcile-proposals)
can score GREEN on a fixture where the note *is* retrievable, while the real-world retrieval gap
remains. So: the sub-metric is **mandatory**, and at least one fixture must reproduce the **retrieval**
miss (note outranked/absent), not just the synthesis miss.

**Validity gates (non-waivable):** the current skill must score **~0.0 RED** on the fixture or it proves
nothing (behavioral traps notoriously don't reproduce in clean toys — opus reconciles trivially in a
sparse vault); need **≥4–5 distinct closed-lever fixtures** before GREEN is trustworthy; size any
delta against judge-run variance, not zero; spot-check adversarial-paraphrase transcripts as a **hard
gate** before trusting GREEN (the semantic "reconciliation-by-vocabulary" guard is the most fragile
part).

## 5 · Sequenced changes (filed as issues)

| # | Change | Axis | Class | Gated by C7 | Effort | Risk |
|---|---|---|---|---|---|---|
| 1 | **Build C7 harness first** (fixtures + scorer + control); tune until ~0.0 RED | quality infra | — | *is* the gate | M | med (judge variance, fixture-cold) |
| 2 | recall Step 3 **note-negation override** (a closed-lever note overrides higher-cosine supporting chunks; surface as contradiction) | quality | cost-neutral | yes (in-payload case) | S | low |
| 3 | recall Step 3 **reconcile-proposals** sub-rule (walk produced recommendations, not just the plan) | quality | cost-neutral | yes (in-context/retrievable) | S | low |
| 4 | recall **recall-before-recommend re-entry** (query each emergent lever + outcome terms) — the load-bearing fix | quality | **trade-off** (1 query) | yes (any phrasing) | S–M | low |
| 5 | please Step 2 / Gate A: make a recommendation a **gated artifact**; reviewer recalls on the *recommendation* + outcome terms | quality | cost-neutral | yes (/please variant) | M | low–med |
| 6 | tighter **O1 chunk content-budget**, spend budget on note content + chunk snippets | tokens + salience | sub-share | no (render, not rank) | S | low–med |
| 7 | ship **safe cuts** O2 / L2 / L3a-sweep, each re-run through C7 | time | cost-only | yes (must hold 1.0) | M | low |
| 8 | **PREREQ-$METER**: *unbundle* recall's $ from `build_cost` (recall runs inside the build session) before any build-loop $ lever (L4/L5) | dollar enabler | — | n/a | M | low |

*Class: do-both / cost-neutral / cost-only / trade-off / sub-share / quality-infra. Effort: S = small
(skill-instruction edit), M = medium (code + skill), L = large. Risk: low / med / high. "Gated by C7" =
the change must hold the with-closed fixture at 1.0 after it lands.*

All eight pass the anti-amnesia audit: none re-introduces L1/O3 (cheaper-tier recall), SLICE2 (BFS
retrieval), SYNTH-STEP (composition step), or O1-as-cost-lever. **Adjacency to watch:** change 5 routes
Gate-A reviewers via `route` (per-reviewer model tier) — implement so it never spawns a *haiku recall
sub-skill* (that is the closed L1 lever).

## 6 · Bottom line

The miss was not "the memory system failed to store/retrieve the lesson" — note 80 existed and is
retrievable. It was "recall has one trigger (before work, keyed to the ask) and the agent invented the
disproven lever *after* it, with no re-check and no note-over-chunk priority." The fix space is mostly
**cheap correctness work** (skill-instruction edits + one micro-query) that is cost-neutral, plus a
**regression criterion (C7)** so we never ship a killed lever again — and it is honestly separate from
the build-loop work where the real warm-build dollars live.
