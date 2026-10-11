# Parallel Memory Agent - Open Questions

These are the Open Decisions carried over from the draft requirements. Joe answered the first six on
2026-10-11, and the answers are applied in [`793a_product-spec.md`](./793a_product-spec.md). His later
review answers are recorded in [793c7 Review answers (2026-10-11)](#793c7-review-answers-2026-10-11). The phrasing choices made in applying them are in
[`793b_decision-log.md`](./793b_decision-log.md). The last four questions are open and await Joe.

## 793c1 What are the capture precision bar and capture-rate target?

- **Status:** answered (Joe, 2026-10-11).
- **Answer:** the capture rate MUST exceed the current baseline (learn fires on about 5% of
  lesson-worthy moments: 4/71 = 5.6% and 3/66 = 4.5%). No fixed precision threshold; the System MUST
  record its capture precision, the share of captured notes judged valid lessons on a sample.
- **Applied in:** [793a3h Capture rate exceeds the baseline](793a_product-spec.md#793a3h-capture-rate-exceeds-the-baseline) and [793a3i Precision is graded independently](793a_product-spec.md#793a3i-precision-is-graded-independently).

## 793c2 Are the volume bounds and latency budget accepted?

- **Status:** answered (Joe, 2026-10-11); refined by [793c7 Review answers (2026-10-11)](#793c7-review-answers-2026-10-11).
- **Answer:** start with today's values: the main agent receives from the instruction-based recall query
  about 135 KB, 45 items and 10.6-11.0 s per query (measured 2026-10-11 on Joe's vault; `engram query`
  defaults `--limit 20`, content budget 15). The System MUST record, per moment, the injected volume
  and the added latency; reducing the bounds needs that data (follow-up #845).
- **Applied in:** [793a2j Volume equals the depth's output](793a_product-spec.md#793a2j-volume-equals-the-depths-output), [793a5g Latency per recall moment](793a_product-spec.md#793a5g-latency-per-recall-moment) and [793a5e Auditability and measurability](793a_product-spec.md#793a5e-auditability-and-measurability).

## 793c3 What is the default mode for a fresh install?

- **Status:** answered (Joe, 2026-10-11).
- **Answer:** active, with measurement always on, so data accumulates for later optimization. Observe
  and off remain operator choices.
- **Applied in:** [793a4e Operating modes and measurement](793a_product-spec.md#793a4e-operating-modes-and-measurement).

## 793c4 Are captured lessons written directly, or held for `curate` review?

- **Status:** answered (Joe, 2026-10-11).
- **Answer:** written directly to the vault. The System, the parallel agent, owns judging what is
  worth saving, so the main agent's context is not polluted with memory evaluation. Precision
  recording ([793c1 What are the capture precision bar and capture-rate target?](#793c1-what-are-the-capture-precision-bar-and-capture-rate-target)) is the quality guard.
- **Applied in:** [793a3a Evaluate every completed turn](793a_product-spec.md#793a3a-evaluate-every-completed-turn) (scenarios [793a3a1a User correction is captured without agent action](793a_product-spec.md#793a3a1-uat-scenarios) and [793a3a1b Explicit save-request](793a_product-spec.md#793a3a1-uat-scenarios)) and [793a3d Provenance](793a_product-spec.md#793a3d-provenance) (scenario [793a3d1a Provenance present](793a_product-spec.md#793a3d1-uat-scenarios)).

## 793c5 Does 793a2i Recall for every prompt regardless of source keep human-originated prompts as the default?

- **Status:** answered (Joe, 2026-10-11).
- **Answer:** no. Recall applies to every prompt regardless of source. Memory is always worth
  checking; a relevant hit not already in the working agent's context is worth injecting.
- **Applied in:** [793a2i Recall for every prompt regardless of source](793a_product-spec.md#793a2i-recall-for-every-prompt-regardless-of-source) and [793a2e Relevance and no re-injection](793a_product-spec.md#793a2e-relevance-and-no-re-injection), replacing the retired "Human-originated moments only" requirement.

## 793c6 Does capture also run inside subagent sessions?

- **Status:** answered (Joe, 2026-10-11).
- **Answer:** yes, in both top-level and subagent sessions.
- **Applied in:** [793a3a Evaluate every completed turn](793a_product-spec.md#793a3a-evaluate-every-completed-turn), scenario [793a3a1c A subagent's own guess overturned by evidence is captured](793a_product-spec.md#793a3a1-uat-scenarios).

## 793c7 Review answers (2026-10-11)

Joe's answers to the review of the first revision, all applied in [`793a_product-spec.md`](./793a_product-spec.md).

1. **Bounds equal recall's output.** Each recall moment picks a depth by the existing triggers; the
   System injects at most what that depth's queries return, which is also the floor, less notes already
   in context. Deep's writes are the System's. Applied in [793a2h Recall depth per moment](793a_product-spec.md#793a2h-recall-depth-per-moment), [793a2j Volume equals the depth's output](793a_product-spec.md#793a2j-volume-equals-the-depths-output), [793a3g Deep recall's writes are the System's](793a_product-spec.md#793a3g-deep-recalls-writes-are-the-systems).
2. **Per-turn and per-session bounds are dropped.**
3. **The per-moment bound applies at tool-result moments too.**
4. **Latency** covers every recall moment and MUST NOT exceed the depth's query latency on the same vault
   and input. Per-moment volume and latency are recorded ([793a5e Auditability and measurability](793a_product-spec.md#793a5e-auditability-and-measurability), [793a5g Latency per recall moment](793a_product-spec.md#793a5g-latency-per-recall-moment)). Reductions wait on #845.
5. **Never re-inject** means never a note already in the receiving agent's context; term Context added;
   subagents are separate; a truncated note is not present. Applied in [793a2e Relevance and no re-injection](793a_product-spec.md#793a2e-relevance-and-no-re-injection), [793a2i Recall for every prompt regardless of source](793a_product-spec.md#793a2i-recall-for-every-prompt-regardless-of-source).
6. **Measurement** is recorded in observe and active, not switchable apart from the mode; off records
   nothing ([793a4e Operating modes and measurement](793a_product-spec.md#793a4e-operating-modes-and-measurement)).
7. **Precision** is judged by a separate grader, reported with its sample size ([793a3i Precision is graded independently](793a_product-spec.md#793a3i-precision-is-graded-independently)).
8. **Capture rate** must exceed 5.6% on the point estimate, no margin ([793a3h Capture rate exceeds the baseline](793a_product-spec.md#793a3h-capture-rate-exceeds-the-baseline)).
9. **Lesson** is any corrected failure, including the agent's own overturned mistake; wider than learn's
   definition, tracked in #846 ([793a1 Problem Statement](793a_product-spec.md#793a1-problem-statement), [793a3a Evaluate every completed turn](793a_product-spec.md#793a3a-evaluate-every-completed-turn)).
10. **Secrets** are a MUST NOT ([793a4d Sensitive content](793a_product-spec.md#793a4d-sensitive-content)).
11. **New IDs** for nodes whose meaning reversed; old IDs retired ([793b14 ID handling](793b_decision-log.md#793b14-id-handling)).
12. **N = 20** per arm accepted, proposed marker removed ([793a5f Effect on outcomes](793a_product-spec.md#793a5f-effect-on-outcomes)).
13. **Attributable presentation only**; the non-imperative rule and refusal check are dropped ([793a2d Attributable presentation](793a_product-spec.md#793a2d-attributable-presentation)).
14. **No self-links** on `Requirement:` lines; canonical specs tracked in #847.

Open for Joe: the log now also records the depth and the phrases queried ([793b8 Per-moment record sits in the existing log requirement](793b_decision-log.md#793b8-per-moment-record-sits-in-the-existing-log-requirement)), so that the depth
and volume scenarios are checkable from the operator-accessible log. This adds two log fields beyond his
answers.

## 793c8 What does the System inject when only incidental items come back?

- **Status:** answered (Joe, 2026-10-11): (a) inject what recall delivers today, the depth's whole output including incidental items. Trimming is a follow-up, [#848](https://github.com/toejough/engram/issues/848), gated on #845 data.
- **Context:** At a moment whose queries return only incidental items (explore picks per [792a7i Half of the notes are explored](../../specs/product/memory-query.md#792a7i-half-of-the-notes-are-explored), newest chunks per [792a7o The newest chunks](../../specs/product/memory-query.md#792a7o-the-newest-chunks), ride-alongs per [792a7n Superseding notes ride along](../../specs/product/memory-query.md#792a7n-superseding-notes-ride-along)) and no direct match, [793a2j Volume equals the depth's output](793a_product-spec.md#793a2j-volume-equals-the-depths-output) would inject all of it.
- **Options:**
  - (a) everything the queries return.
  - (b) only direct matches and their ride-alongs.
  - (c) the full output when at least one direct match exists, nothing otherwise.
- **Recommendation:** (c). It keeps the "depth's output" rule when something matches and avoids injecting noise when nothing does.
- **Blocks:** [793a2e Relevance and no re-injection](793a_product-spec.md#793a2e-relevance-and-no-re-injection), [793a2j Volume equals the depth's output](793a_product-spec.md#793a2j-volume-equals-the-depths-output), [793a2a1b No relevant note exists](793a_product-spec.md#793a2a1-uat-scenarios), [793a2e1b Irrelevant note suppressed](793a_product-spec.md#793a2e1-uat-scenarios).

## 793c9 Does the System activate notes?

- **Status:** answered (Joe, 2026-10-11): (c) notes the agent went on to use, judged at turn-end capture.
- **Context:** Recall activates the notes it used (Step 2.7, [792a14l Only the notes that were used are activated](../../specs/product/recall.md#792a14l-only-the-notes-that-were-used-are-activated)). When the System injects notes, nobody activates the ones the agent applies.
- **Options:**
  - (a) never.
  - (b) every injected note.
  - (c) notes the agent went on to use, judged at turn-end capture.
- **Recommendation:** (c). It matches recall's "only the notes that were used" rule at a moment the System already evaluates.
- **Blocks:** a new requirement.

## 793c10 What is the default mode for an existing install that updates?

- **Status:** superseded by [793c12 Are operating modes needed at all?](#793c12-are-operating-modes-needed-at-all), pending its answer.
- **Context:** [793a4e Operating modes and measurement](793a_product-spec.md#793a4e-operating-modes-and-measurement) makes active the default for a fresh install ([793c3 What is the default mode for a fresh install?](793c_open-questions.md#793c3-what-is-the-default-mode-for-a-fresh-install)). An existing install that updates has its own mode choice, and an update could start writing to its vault.
- **Options:**
  - (a) active.
  - (b) observe.
  - (c) off until opt-in.
- **Recommendation:** (a), consistent with the fresh-install answer; measurement stays on either way.
- **Blocks:** [793a4e Operating modes and measurement](793a_product-spec.md#793a4e-operating-modes-and-measurement).

## 793c11 What does the latency bound cover?

- **Status:** answered (Joe, 2026-10-11): (b) the instruction path's full lookup, phrasing plus queries, on the same input.
- **Context:** [793a5g Latency per recall moment](793a_product-spec.md#793a5g-latency-per-recall-moment) caps added latency at the latency of the depth's queries. The instruction path's lookup also includes the agent phrasing its query, which the System does too.
- **Options:**
  - (a) queries only.
  - (b) the instruction path's full lookup (phrasing plus queries) on the same input.
  - (c) queries plus a fixed allowance.
- **Recommendation:** (b). It compares like with like and needs no fixed number that would go stale.
- **Blocks:** [793a5g Latency per recall moment](793a_product-spec.md#793a5g-latency-per-recall-moment).

## 793c12 Are operating modes needed at all?

- **Status:** open.
- **Context:** [793a4e Operating modes and measurement](793a_product-spec.md#793a4e-operating-modes-and-measurement) carries the draft's off/observe/active modes (formerly CTL-1). Joe asked (2026-10-11) why not keep today's behavior: if the integration is installed, it is active. The draft's reasons for modes were an observe-first rollout (measure before injecting or writing) and an emergency off switch without uninstalling. Joe's later answers remove the first need: default active ([793c3 What is the default mode for a fresh install?](#793c3-what-is-the-default-mode-for-a-fresh-install)), measurement always recorded ([793c7 Review answers (2026-10-11)](#793c7-review-answers-2026-10-11)), and direct writes with git as the fallback. Disabling or uninstalling the integration covers the second need.
- **Options:**
  - (a) Remove modes. The System is active whenever it is installed, measurement is always recorded, and disabling or uninstalling the integration turns it off. 793a4e, its scenarios and [793c10 What is the default mode for an existing install that updates?](#793c10-what-is-the-default-mode-for-an-existing-install-that-updates) are retired.
  - (b) Keep only an off switch, with no observe mode.
  - (c) Keep off, observe and active.
- **Recommendation:** (a). It matches today's behavior and removes a configuration surface and its tests.
- **Blocks:** [793a4e Operating modes and measurement](793a_product-spec.md#793a4e-operating-modes-and-measurement), [793c10 What is the default mode for an existing install that updates?](#793c10-what-is-the-default-mode-for-an-existing-install-that-updates), and the observe-mode findings in 793b (M1).

