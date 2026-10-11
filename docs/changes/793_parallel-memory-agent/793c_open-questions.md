# Parallel Memory Agent - Open Questions

These are the Open Decisions carried over from the draft requirements. Joe answered the first six on
2026-10-11, and the answers are applied in [`793a_product-spec.md`](./793a_product-spec.md). His later
review answers are recorded in [793c7 Review answers (2026-10-11)](#793c7-review-answers-2026-10-11), and his answers to [793c8 What does the System inject when only incidental items come back?](#793c8-what-does-the-system-inject-when-only-incidental-items-come-back), [793c9 Does the System activate notes?](#793c9-does-the-system-activate-notes), [793c11 What does the latency bound cover?](#793c11-what-does-the-latency-bound-cover), [793c12 Are operating modes needed at all?](#793c12-are-operating-modes-needed-at-all), [793c13 Does the System activate notes the agent retrieved itself?](#793c13-does-the-system-activate-notes-the-agent-retrieved-itself) and [793c14 Should reinforcing an existing note be a MUST?](#793c14-should-reinforcing-an-existing-note-be-a-must) are applied too. The phrasing choices
made in applying them are in [`793b_decision-log.md`](./793b_decision-log.md). Six questions, [793c15 Does the System inject chunks with their content?](#793c15-does-the-system-inject-chunks-with-their-content) to [793c20 Must injected chunks identify their source like notes?](#793c20-must-injected-chunks-identify-their-source-like-notes), are open and await Joe.

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
- **Applied in:** nothing now: `793a4e Operating modes and measurement` is retired by [793c12 Are operating modes needed at all?](#793c12-are-operating-modes-needed-at-all). The active default is [793a4g Active whenever installed](793a_product-spec.md#793a4g-active-whenever-installed) scenario [793a4g1a A fresh install is active](793a_product-spec.md#793a4g1-uat-scenarios).

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
- **Applied in:** [793a2i Recall for every prompt regardless of source](793a_product-spec.md#793a2i-recall-for-every-prompt-regardless-of-source) and [793a2e No re-injection](793a_product-spec.md#793a2e-no-re-injection), replacing the retired "Human-originated moments only" requirement.

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
   subagents are separate; a truncated note is not present. Applied in [793a2e No re-injection](793a_product-spec.md#793a2e-no-re-injection), [793a2i Recall for every prompt regardless of source](793a_product-spec.md#793a2i-recall-for-every-prompt-regardless-of-source).
6. **Measurement** is recorded in observe and active, not switchable apart from the mode; off records
   nothing (`793a4e Operating modes and measurement`). Superseded by [793c12 Are operating modes needed at all?](#793c12-are-operating-modes-needed-at-all): measurement is always recorded ([793a5e Auditability and measurability](793a_product-spec.md#793a5e-auditability-and-measurability)).
7. **Precision** is judged by a separate grader, reported with its sample size ([793a3i Precision is graded independently](793a_product-spec.md#793a3i-precision-is-graded-independently)).
8. **Capture rate** must exceed 5.6% on the point estimate, no margin ([793a3h Capture rate exceeds the baseline](793a_product-spec.md#793a3h-capture-rate-exceeds-the-baseline)).
9. **Lesson** is any corrected failure, including the agent's own overturned mistake; wider than learn's
   definition, tracked in #846 ([793a1 Problem Statement](793a_product-spec.md#793a1-problem-statement), [793a3a Evaluate every completed turn](793a_product-spec.md#793a3a-evaluate-every-completed-turn)).
10. **Secrets** are a MUST NOT ([793a4d Sensitive content](793a_product-spec.md#793a4d-sensitive-content)).
11. **New IDs** for nodes whose meaning reversed; old IDs retired ([793b14 ID handling](793b_decision-log.md#793b14-id-handling)).
12. **N = 20** per arm accepted, proposed marker removed ([793a5f Effect on outcomes](793a_product-spec.md#793a5f-effect-on-outcomes)).
13. **Attributable presentation only**; the non-imperative rule and refusal check are dropped ([793a2d Attributable presentation](793a_product-spec.md#793a2d-attributable-presentation)).
14. **No self-links** on `Requirement:` lines; canonical specs tracked in #847.

Open for Joe: whether the log keeps the extra fields it records beyond his answers is [793c19 Does the log keep the depth, each query's phrases and the glance escalation?](#793c19-does-the-log-keep-the-depth-each-querys-phrases-and-the-glance-escalation).

## 793c8 What does the System inject when only incidental items come back?

- **Status:** answered (Joe, 2026-10-11): (a) inject what recall delivers today, the depth's whole output including incidental items. Trimming is a follow-up, [#848](https://github.com/toejough/engram/issues/848), gated on #845 data.
- **Context:** At a moment whose queries return only incidental items (explore picks per [792a7i Half of the notes are explored](../../specs/product/memory-query.md#792a7i-half-of-the-notes-are-explored), newest chunks per [792a7o The newest chunks](../../specs/product/memory-query.md#792a7o-the-newest-chunks), ride-alongs per [792a7n Superseding notes ride along](../../specs/product/memory-query.md#792a7n-superseding-notes-ride-along)) and no direct match, [793a2j Volume equals the depth's output](793a_product-spec.md#793a2j-volume-equals-the-depths-output) would inject all of it.
- **Options:**
  - (a) everything the queries return.
  - (b) only direct matches and their ride-alongs.
  - (c) the full output when at least one direct match exists, nothing otherwise.
- **Recommendation:** (c). It keeps the "depth's output" rule when something matches and avoids injecting noise when nothing does.
- **Applied in:** [793a2e No re-injection](793a_product-spec.md#793a2e-no-re-injection), [793a2j Volume equals the depth's output](793a_product-spec.md#793a2j-volume-equals-the-depths-output) and [793a2a1c Only incidental items come back](793a_product-spec.md#793a2a1-uat-scenarios); `793a2a1b` and `793a2e1b` are retired. Decision: [793b46 Inject what recall delivers today](793b_decision-log.md#793b46-inject-what-recall-delivers-today).
- **Blocks:** [793a2e No re-injection](793a_product-spec.md#793a2e-no-re-injection), [793a2j Volume equals the depth's output](793a_product-spec.md#793a2j-volume-equals-the-depths-output), `793a2a1b No relevant note exists`, `793a2e1b Irrelevant note suppressed`.

## 793c9 Does the System activate notes?

- **Status:** answered (Joe, 2026-10-11): (c) notes the agent went on to use, judged at turn-end capture.
- **Context:** Recall activates the notes it used (Step 2.7, [792a14l Only the notes that were used are activated](../../specs/product/recall.md#792a14l-only-the-notes-that-were-used-are-activated)). When the System injects notes, nobody activates the ones the agent applies.
- **Options:**
  - (a) never.
  - (b) every injected note.
  - (c) notes the agent went on to use, judged at turn-end capture.
- **Recommendation:** (c). It matches recall's "only the notes that were used" rule at a moment the System already evaluates.
- **Applied in:** [793a3k Used notes are activated](793a_product-spec.md#793a3k-used-notes-are-activated). Decision: [793b47 Used notes are activated at turn-end capture](793b_decision-log.md#793b47-used-notes-are-activated-at-turn-end-capture).
- **Blocks:** a new requirement.

## 793c10 What is the default mode for an existing install that updates?

- **Status:** retired by [793c12 Are operating modes needed at all?](#793c12-are-operating-modes-needed-at-all) (Joe, 2026-10-11): there are no modes.
- **Context:** `793a4e Operating modes and measurement` makes active the default for a fresh install ([793c3 What is the default mode for a fresh install?](793c_open-questions.md#793c3-what-is-the-default-mode-for-a-fresh-install)). An existing install that updates has its own mode choice, and an update could start writing to its vault.
- **Options:**
  - (a) active.
  - (b) observe.
  - (c) off until opt-in.
- **Recommendation:** (a), consistent with the fresh-install answer; measurement stays on either way.
- **Blocks:** `793a4e Operating modes and measurement`.

## 793c11 What does the latency bound cover?

- **Status:** answered (Joe, 2026-10-11): (b) the instruction path's full lookup, phrasing plus queries, on the same input.
- **Context:** [793a5g Latency per recall moment](793a_product-spec.md#793a5g-latency-per-recall-moment) caps added latency at the latency of the depth's queries. The instruction path's lookup also includes the agent phrasing its query, which the System does too.
- **Options:**
  - (a) queries only.
  - (b) the instruction path's full lookup (phrasing plus queries) on the same input.
  - (c) queries plus a fixed allowance.
- **Recommendation:** (b). It compares like with like and needs no fixed number that would go stale.
- **Applied in:** [793a5g Latency per recall moment](793a_product-spec.md#793a5g-latency-per-recall-moment). Decision: [793b48 Latency covers the instruction path's full lookup](793b_decision-log.md#793b48-latency-covers-the-instruction-paths-full-lookup).
- **Blocks:** [793a5g Latency per recall moment](793a_product-spec.md#793a5g-latency-per-recall-moment).

## 793c12 Are operating modes needed at all?

- **Status:** answered (Joe, 2026-10-11): (a) remove modes; the System is active whenever installed, measurement is always recorded, and disabling or uninstalling turns it off.
- **Context:** `793a4e Operating modes and measurement` carries the draft's off/observe/active modes (formerly CTL-1). Joe asked (2026-10-11) why not keep today's behavior: if the integration is installed, it is active. The draft's reasons for modes were an observe-first rollout (measure before injecting or writing) and an emergency off switch without uninstalling. Joe's later answers remove the first need: default active ([793c3 What is the default mode for a fresh install?](#793c3-what-is-the-default-mode-for-a-fresh-install)), measurement always recorded ([793c7 Review answers (2026-10-11)](#793c7-review-answers-2026-10-11)), and direct writes with git as the fallback. Disabling or uninstalling the integration covers the second need.
- **Options:**
  - (a) Remove modes. The System is active whenever it is installed, measurement is always recorded, and disabling or uninstalling the integration turns it off. `793a4e`, its scenarios and [793c10 What is the default mode for an existing install that updates?](#793c10-what-is-the-default-mode-for-an-existing-install-that-updates) are retired.
  - (b) Keep only an off switch, with no observe mode.
  - (c) Keep off, observe and active.
- **Recommendation:** (a). It matches today's behavior and removes a configuration surface and its tests.
- **Blocks:** `793a4e Operating modes and measurement`, [793c10 What is the default mode for an existing install that updates?](#793c10-what-is-the-default-mode-for-an-existing-install-that-updates), and the observe-mode finding [793b28 M1: observe mode logs the would-be volume](793b_decision-log.md#793b28-m1-observe-mode-logs-the-would-be-volume).
- **Applied in:** [793a4g Active whenever installed](793a_product-spec.md#793a4g-active-whenever-installed) and [793a5e Auditability and measurability](793a_product-spec.md#793a5e-auditability-and-measurability); `793a4e` is retired. Decision: [793b49 No operating modes](793b_decision-log.md#793b49-no-operating-modes).

## 793c13 Does the System activate notes the agent retrieved itself?

- **Status:** answered (Joe, 2026-10-11): (a) only notes the System injected; explicit recall activates its own.
- **Context:** [793a3k Used notes are activated](793a_product-spec.md#793a3k-used-notes-are-activated) covers the notes the System injected during the turn, per [793c9 Does the System activate notes?](#793c9-does-the-system-activate-notes). When recall runs as an instruction, it activates the notes it used ([792a14l Only the notes that were used are activated](../../specs/product/recall.md#792a14l-only-the-notes-that-were-used-are-activated)). A note the agent retrieved by its own lookup and used is activated only if that lookup was a recall.
- **Options:**
  - (a) injected notes only.
  - (b) also notes the agent retrieved itself and used.
- **Recommendation:** (a). Recall already activates its own retrievals, and (b) widens the System's judgment to content it did not deliver.
- **Blocks:** [793a3k Used notes are activated](793a_product-spec.md#793a3k-used-notes-are-activated).

## 793c14 Should reinforcing an existing note be a MUST?

- **Status:** answered (Joe, 2026-10-11): (b) reinforcement is a MUST; [793a3c No duplicates](793a_product-spec.md#793a3c-no-duplicates) and [793a3c1a Existing note already covers the lesson](793a_product-spec.md#793a3c1-uat-scenarios) updated. Decision: [793b52 Reinforcement is required](793b_decision-log.md#793b52-reinforcement-is-required).
- **Context:** [793a3c No duplicates](793a_product-spec.md#793a3c-no-duplicates) says the System SHOULD reinforce an equivalent existing note. The second review ([793b35 M8: "Reinforced" is not backed by a MUST](793b_decision-log.md#793b35-m8-reinforced-is-not-backed-by-a-must)) noted the scenario asserted a mark the requirement did not require, and the scenario no longer does.
- **Options:**
  - (a) keep SHOULD, with no scenario.
  - (b) make it a MUST, with a scenario asserting the existing note's recency or reinforcement record changed.
- **Recommendation:** (a). The owner has not asked for more, and a duplicate is already forbidden.
- **Blocks:** [793a3c No duplicates](793a_product-spec.md#793a3c-no-duplicates).

## 793c15 Does the System inject chunks with their content?

- **Status:** open.
- **Context:** [792a14f One unified query](../../specs/product/recall.md#792a14f-one-unified-query) requires `--lazy-chunks`, which prints chunks as path and score only ([792a7s Lazy chunks](../../specs/product/memory-query.md#792a7s-lazy-chunks)). The baseline measurement in [793c2 Are the volume bounds and latency budget accepted?](#793c2-are-the-volume-bounds-and-latency-budget-accepted) used the query's defaults, which print chunk content. [793a2e1f A chunk shown without content is delivered with it](793a_product-spec.md#793a2e1-uat-scenarios) has the System deliver a chunk with its content.
- **Options:**
  - (a) yes: the System runs the query with the defaults, without `--lazy-chunks`, so [793a2e1f A chunk shown without content is delivered with it](793a_product-spec.md#793a2e1-uat-scenarios) stands.
  - (b) no: the System runs it lazy as recall does, which needs a "do not re-inject the same stub" rule and retires [793a2e1f A chunk shown without content is delivered with it](793a_product-spec.md#793a2e1-uat-scenarios).
- **Recommendation:** (a). It matches the volume that was measured, and the System has no agent turn in which to fetch a stub.
- **Blocks:** [793a2e No re-injection](793a_product-spec.md#793a2e-no-re-injection), [793a2e1f A chunk shown without content is delivered with it](793a_product-spec.md#793a2e1-uat-scenarios), [793a2e1h A chunk delivered earlier is not delivered again](793a_product-spec.md#793a2e1-uat-scenarios) and the volume figures in [793a5e Auditability and measurability](793a_product-spec.md#793a5e-auditability-and-measurability).

## 793c16 At a deep moment, does the System's coverage write activate the note it covers?

- **Status:** open.
- **Context:** at a deep moment the System writes coverage ([792a14j Coverage is judged by the agent and written once per cluster](../../specs/product/recall.md#792a14j-coverage-is-judged-by-the-agent-and-written-once-per-cluster), [793a3g Deep recall's writes are the System's](793a_product-spec.md#793a3g-deep-recalls-writes-are-the-systems)) by adding chunk sources to an existing note with `engram amend`. `engram amend --activate` would also bump that note's recency, which [793a3k Used notes are activated](793a_product-spec.md#793a3k-used-notes-are-activated) otherwise reserves for notes the agent used ([793c9 Does the System activate notes?](#793c9-does-the-system-activate-notes)).
- **Options:**
  - (a) no: the System adds chunk sources without `--activate`, and only agent-used notes are activated ([793a3k Used notes are activated](793a_product-spec.md#793a3k-used-notes-are-activated)).
  - (b) yes: a coverage write activates the note it covers, as an exception to [793a3k Used notes are activated](793a_product-spec.md#793a3k-used-notes-are-activated).
- **Recommendation:** (a). A coverage write records that a note covers a cluster, not that the agent used it, so activating it would keep unused notes fresh.
- **Blocks:** [793a3g Deep recall's writes are the System's](793a_product-spec.md#793a3g-deep-recalls-writes-are-the-systems) and [793a3k Used notes are activated](793a_product-spec.md#793a3k-used-notes-are-activated).

## 793c17 Does the latency bound include reading the glance results?

- **Status:** open.
- **Context:** [793a5g Latency per recall moment](793a_product-spec.md#793a5g-latency-per-recall-moment) bounds added latency by the instruction path's full lookup on the same input ([793c11 What does the latency bound cover?](#793c11-what-does-the-latency-bound-cover)). Escalating from glance to deep ([793a2h Recall depth per moment](793a_product-spec.md#793a2h-recall-depth-per-moment), [792a14d Glance escalates to deep for a recent standard](../../specs/product/recall.md#792a14d-glance-escalates-to-deep-for-a-recent-standard)) and the recommendation check ([792a14o An emergent recommendation is checked against the vault](../../specs/product/recall.md#792a14o-an-emergent-recommendation-is-checked-against-the-vault)) depend on reading the glance results, which takes agent time.
- **Options:**
  - (a) yes: reading the results is part of the instruction path's full lookup, so the bound includes it.
  - (b) no: the bound covers phrases plus queries only.
  - (c) drop the result-dependent escalation triggers and fix the depth up front.
- **Recommendation:** (a). It compares like with like and keeps the escalation triggers.
- **Blocks:** [793a5g Latency per recall moment](793a_product-spec.md#793a5g-latency-per-recall-moment), [793a2h Recall depth per moment](793a_product-spec.md#793a2h-recall-depth-per-moment), [793a2h1c An unresolved gap uses deep](793a_product-spec.md#793a2h1-uat-scenarios) and [793a5e1f An escalated moment records both depths](793a_product-spec.md#793a5e1-uat-scenarios).

## 793c18 How is the System installed and turned off?

- **Status:** open.
- **Context:** [793a4g Active whenever installed](793a_product-spec.md#793a4g-active-whenever-installed) makes the System active whenever the integration is installed and enabled, and inert when disabled or uninstalled. It does not say how the integration gets installed, or how an operator disables it.
- **Options:**
  - (a) `engram update` installs it by default, and it is disabled through the harness's own disable mechanism or by uninstalling.
  - (b) opt-in like guidance ([792a11l Guidance is managed only after opt-in](../../specs/product/install-and-update.md#792a11l-guidance-is-managed-only-after-opt-in)), then active.
- **Recommendation:** (a). It matches "active whenever installed" and adds no engram setting.
- **Blocks:** [793a4g Active whenever installed](793a_product-spec.md#793a4g-active-whenever-installed).

## 793c19 Does the log keep the depth, each query's phrases and the glance escalation?

- **Status:** open.
- **Context:** raised unnumbered at the end of [793c7 Review answers (2026-10-11)](#793c7-review-answers-2026-10-11). [793a5e Auditability and measurability](793a_product-spec.md#793a5e-auditability-and-measurability) also logs the depth, the phrases of each query and whether glance escalated, which is more than Joe's answers asked for ([793b8 Per-moment record sits in the existing log requirement](793b_decision-log.md#793b8-per-moment-record-sits-in-the-existing-log-requirement), [793b43 Author's choice accepted: log depth, phrases and escalation](793b_decision-log.md#793b43-authors-choice-accepted-log-depth-phrases-and-escalation)).
- **Options:**
  - (a) keep the fields.
  - (b) drop them, which makes the [793a2h1a A routine decision uses glance](793a_product-spec.md#793a2h1-uat-scenarios) and [793a2h1c An unresolved gap uses deep](793a_product-spec.md#793a2h1-uat-scenarios) scenarios unverifiable from the log.
- **Recommendation:** (a). The depth and escalation scenarios can only be checked from the operator-accessible log.
- **Blocks:** [793a5e Auditability and measurability](793a_product-spec.md#793a5e-auditability-and-measurability), [793a2h1a A routine decision uses glance](793a_product-spec.md#793a2h1-uat-scenarios), [793a2h1c An unresolved gap uses deep](793a_product-spec.md#793a2h1-uat-scenarios) and [793a5e1f An escalated moment records both depths](793a_product-spec.md#793a5e1-uat-scenarios).

## 793c20 Must injected chunks identify their source like notes?

- **Status:** open.
- **Context:** [793a2d Attributable presentation](793a_product-spec.md#793a2d-attributable-presentation) requires injected notes to identify their source note. The System also injects chunks, which have an id but no note file name.
- **Options:**
  - (a) yes: "Injected items MUST identify their source (a note's file name, a chunk's id)".
  - (b) chunks are exempt.
- **Recommendation:** (a). A chunk the agent cannot trace to its source cannot be weighed or looked up.
- **Blocks:** [793a2d Attributable presentation](793a_product-spec.md#793a2d-attributable-presentation).
