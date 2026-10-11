# Parallel Memory Agent - Open Questions

These are the Open Decisions carried over from the draft requirements. Joe answered the first six on
2026-10-11, and the answers are applied in [`793a_product-spec.md`](./793a_product-spec.md). His later
review answers are recorded in [793c7 Review answers (2026-10-11)](#793c7-review-answers-2026-10-11), and his answers to [793c8 What does the System inject when only incidental items come back?](#793c8-what-does-the-system-inject-when-only-incidental-items-come-back), [793c9 Does the System activate notes?](#793c9-does-the-system-activate-notes), [793c11 What does the latency bound cover?](#793c11-what-does-the-latency-bound-cover), [793c12 Are operating modes needed at all?](#793c12-are-operating-modes-needed-at-all), [793c13 Does the System activate notes the agent retrieved itself?](#793c13-does-the-system-activate-notes-the-agent-retrieved-itself) and [793c14 Should reinforcing an existing note be a MUST?](#793c14-should-reinforcing-an-existing-note-be-a-must) are applied too. The phrasing choices
made in applying them are in [`793b_decision-log.md`](./793b_decision-log.md). His answers to [793c15 Does the System inject chunks with their content?](#793c15-does-the-system-inject-chunks-with-their-content) to [793c22 There is no uninstall command today](#793c22-there-is-no-uninstall-command-today) are applied too, as are his answers to [793c23 Is a note amended as near injected?](#793c23-is-a-note-amended-as-near-injected) and [793c24 Does the covering note's injection add anything to the depth's output?](#793c24-does-the-covering-notes-injection-add-anything-to-the-depths-output). Four questions, [793c25 When are the covering and amended notes from a deep moment injected?](#793c25-when-are-the-covering-and-amended-notes-from-a-deep-moment-injected), [793c26 What does the System's recommendation check (792a14o) key on?](#793c26-what-does-the-systems-recommendation-check-792a14o-key-on), [793c27 Does the System sweep the chunk index (792a14b)?](#793c27-does-the-system-sweep-the-chunk-index-792a14b) and [793c28 How is 793a5g's latency comparison measured?](#793c28-how-is-793a5gs-latency-comparison-measured), were open when Joe's answers to [793c29 Is the System a delegate for the existing memory skills?](#793c29-is-the-system-a-delegate-for-the-existing-memory-skills) and [793c30 Which earlier decisions stay as deliberate differences from the skills?](#793c30-which-earlier-decisions-stay-as-deliberate-differences-from-the-skills) settled them: [793c25 When are the covering and amended notes from a deep moment injected?](#793c25-when-are-the-covering-and-amended-notes-from-a-deep-moment-injected), [793c26 What does the System's recommendation check (792a14o) key on?](#793c26-what-does-the-systems-recommendation-check-792a14o-key-on) and [793c28 How is 793a5g's latency comparison measured?](#793c28-how-is-793a5gs-latency-comparison-measured) are retired and [793c27 Does the System sweep the chunk index (792a14b)?](#793c27-does-the-system-sweep-the-chunk-index-792a14b) is answered. Those two answers restructured [`793a_product-spec.md`](./793a_product-spec.md) as a delegation of the memory skills, and many earlier answers now govern requirements that no longer exist; their applied-in lines name those requirements in code font. His answers to [793c31 Which canonical nodes that ask the agent to run recall or learn change while the Integration is installed?](#793c31-which-canonical-nodes-that-ask-the-agent-to-run-recall-or-learn-change-while-the-integration-is-installed) to [793c35 How is the delay comparison in 793a5g1a made given timing noise?](#793c35-how-is-the-delay-comparison-in-793a5g1a-made-given-timing-noise) are applied too: [793c31 Which canonical nodes that ask the agent to run recall or learn change while the Integration is installed?](#793c31-which-canonical-nodes-that-ask-the-agent-to-run-recall-or-learn-change-while-the-integration-is-installed) as the operations of [793a7 ambient-guidance, route and please (the working agent no longer runs recall or learn while the Integration is installed)](793a_product-spec.md#793a7-modified-ambient-guidance-route-and-please-the-working-agent-no-longer-runs-recall-or-learn-while-the-integration-is-installed), and [793c32 Does the System keep a rule against concurrent duplicate captures?](#793c32-does-the-system-keep-a-rule-against-concurrent-duplicate-captures) to [793c35 How is the delay comparison in 793a5g1a made given timing noise?](#793c35-how-is-the-delay-comparison-in-793a5g1a-made-given-timing-noise) by accepting the specification as written. His answers to [793c36 Does the shim's follow-frame apply to notes the System delivers?](#793c36-does-the-shims-follow-frame-apply-to-notes-the-system-delivers) to [793c39 Does tier evidence reach the orchestrator before it routes?](#793c39-does-tier-evidence-reach-the-orchestrator-before-it-routes) are applied too: [793c36 Does the shim's follow-frame apply to notes the System delivers?](#793c36-does-the-shims-follow-frame-apply-to-notes-the-system-delivers) as no change ([792a19l Each returned kind is treated for what it is](../../specs/product/ambient-guidance.md#792a19l-each-returned-kind-is-treated-for-what-it-is) to [792a19q Linked and truncated runbooks get the same frame](../../specs/product/ambient-guidance.md#792a19q-linked-and-truncated-runbooks-get-the-same-frame) stay unmodified), [793c37 What is the shape of please's workflow while the Integration is installed?](#793c37-what-is-the-shape-of-pleases-workflow-while-the-integration-is-installed) as [793a3o Please's lessons audit and surfacing question are performed by the System](793a_product-spec.md#793a3o-pleases-lessons-audit-and-surfacing-question-are-performed-by-the-system) and the operations [793a7l The seven steps are tracked, ordered and non-waivable](793a_product-spec.md#793a7l-modified-the-seven-steps-are-tracked-ordered-and-non-waivable), [793a7o The closing learn is preceded by a lessons audit](793a_product-spec.md#793a7o-modified-the-closing-learn-is-preceded-by-a-lessons-audit) and [793a7p A lesson about to be captured asks what should have surfaced it](793a_product-spec.md#793a7p-modified-a-lesson-about-to-be-captured-asks-what-should-have-surfaced-it), [793c38 Do the orchestrator's route-evidence writes stay?](#793c38-do-the-orchestrators-route-evidence-writes-stay) as [793a3p Route's evidence writes are performed by the System](793a_product-spec.md#793a3p-routes-evidence-writes-are-performed-by-the-system) and the operations [793a7q Each dispatch is also written to the vault as a tagged evidence note](793a_product-spec.md#793a7q-modified-each-dispatch-is-also-written-to-the-vault-as-a-tagged-evidence-note) and [793a7r Each work-kind keeps one aggregate note](793a_product-spec.md#793a7r-modified-each-work-kind-keeps-one-aggregate-note), and [793c39 Does tier evidence reach the orchestrator before it routes?](#793c39-does-tier-evidence-reach-the-orchestrator-before-it-routes) as the dispatch-decision moment of [793a2c Moments are noticed without the agent](793a_product-spec.md#793a2c-moments-are-noticed-without-the-agent) and the operation [793a7i The starting tier comes from recalled evidence, otherwise the cheapest tier](793a_product-spec.md#793a7i-modified-the-starting-tier-comes-from-recalled-evidence-otherwise-the-cheapest-tier). His answer to [793c40 Does the user still see please's audit list?](#793c40-does-the-user-still-see-pleases-audit-list), raised in applying [793c37 What is the shape of please's workflow while the Integration is installed?](#793c37-what-is-the-shape-of-pleases-workflow-while-the-integration-is-installed) is applied too, as [793a5h A memory-activity summary is shown to the user](793a_product-spec.md#793a5h-a-memory-activity-summary-is-shown-to-the-user). The fifth review raised four open questions, [793c41 At a recall moment, what does the System put in front of the working agent?](#793c41-at-a-recall-moment-what-does-the-system-put-in-front-of-the-working-agent) to [793c44 Where do the delegated skills' messages to the user go?](#793c44-where-do-the-delegated-skills-messages-to-the-user-go). Joe's answer to [793c45 Is the Integration installed for every harness, or only for Claude Code?](#793c45-is-the-integration-installed-for-every-harness-or-only-for-claude-code) is applied, and [793c46 On Claude Code with the Integration installed, does engram update still deploy the recall and learn skills and guidance files?](#793c46-on-claude-code-with-the-integration-installed-does-engram-update-still-deploy-the-recall-and-learn-skills-and-guidance-files) and [793c47 Does "init" mean `engram init` or the first-time install by `engram update`?](#793c47-does-init-mean-engram-init-or-the-first-time-install-by-engram-update), which it raised, are answered too and applied. [793c48 How is the widening of `engram init` (change #795) sequenced with this change?](#793c48-how-is-the-widening-of-engram-init-change-795-sequenced-with-this-change), raised by the latter, is open.

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
- **Applied in:** `793a2j Volume equals the depth's output`, [793a5g No added delay](793a_product-spec.md#793a5g-no-added-delay) and [793a5e Auditability and measurability](793a_product-spec.md#793a5e-auditability-and-measurability).

## 793c3 What is the default mode for a fresh install?

- **Status:** answered (Joe, 2026-10-11).
- **Answer:** active, with measurement always on, so data accumulates for later optimization. Observe
  and off remain operator choices.
- **Applied in:** nothing now: `793a4e Operating modes and measurement` is retired by [793c12 Are operating modes needed at all?](#793c12-are-operating-modes-needed-at-all). The active default is [793a4g Active whenever installed](793a_product-spec.md#793a4g-active-whenever-installed) scenario [793a4g1a An Integration installed by update is active](793a_product-spec.md#793a4g1-uat-scenarios).

## 793c4 Are captured lessons written directly, or held for `curate` review?

- **Status:** answered (Joe, 2026-10-11).
- **Answer:** written directly to the vault. The System, the parallel agent, owns judging what is
  worth saving, so the main agent's context is not polluted with memory evaluation. Precision
  recording ([793c1 What are the capture precision bar and capture-rate target?](#793c1-what-are-the-capture-precision-bar-and-capture-rate-target)) is the quality guard.
- **Applied in:** [793a3a Learn is performed for the working agent](793a_product-spec.md#793a3a-learn-is-performed-for-the-working-agent) (scenarios [793a3a1a User correction is captured without agent action](793a_product-spec.md#793a3a1-uat-scenarios) and [793a3a1b Explicit save-request](793a_product-spec.md#793a3a1-uat-scenarios)) and [793a3d Provenance](793a_product-spec.md#793a3d-provenance) (scenario [793a3d1a Provenance present](793a_product-spec.md#793a3d1-uat-scenarios)).

## 793c5 Does 793a2i Recall for every prompt regardless of source keep human-originated prompts as the default?

- **Status:** answered (Joe, 2026-10-11).
- **Answer:** no. Recall applies to every prompt regardless of source. Memory is always worth
  checking; a relevant hit not already in the working agent's context is worth injecting.
- **Applied in:** [793a2i Recall for every prompt regardless of source](793a_product-spec.md#793a2i-recall-for-every-prompt-regardless-of-source) and [793a2e No re-injection](793a_product-spec.md#793a2e-no-re-injection), replacing the retired "Human-originated moments only" requirement.

## 793c6 Does capture also run inside subagent sessions?

- **Status:** answered (Joe, 2026-10-11).
- **Answer:** yes, in both top-level and subagent sessions.
- **Applied in:** [793a3a Learn is performed for the working agent](793a_product-spec.md#793a3a-learn-is-performed-for-the-working-agent), scenario `793a3a1c A subagent's own guess overturned by evidence is captured`.

## 793c7 Review answers (2026-10-11)

Joe's answers to the review of the first revision, all applied in [`793a_product-spec.md`](./793a_product-spec.md).

1. **Bounds equal recall's output.** Each recall moment picks a depth by the existing triggers; the
   System injects at most what that depth's queries return, which is also the floor, less notes already
   in context. Deep's writes are the System's. Applied in `793a2h Recall depth per moment`, `793a2j Volume equals the depth's output`, `793a3g Deep recall's writes are the System's`.
2. **Per-turn and per-session bounds are dropped.**
3. **The per-moment bound applies at tool-result moments too.**
4. **Latency** covers every recall moment and MUST NOT exceed the depth's query latency on the same vault
   and input. Per-moment volume and latency are recorded ([793a5e Auditability and measurability](793a_product-spec.md#793a5e-auditability-and-measurability), [793a5g No added delay](793a_product-spec.md#793a5g-no-added-delay)). Reductions wait on #845.
5. **Never re-inject** means never a note already in the receiving agent's context; term Context added;
   subagents are separate; a truncated note is not present. Applied in [793a2e No re-injection](793a_product-spec.md#793a2e-no-re-injection), [793a2i Recall for every prompt regardless of source](793a_product-spec.md#793a2i-recall-for-every-prompt-regardless-of-source).
6. **Measurement** is recorded in observe and active, not switchable apart from the mode; off records
   nothing (`793a4e Operating modes and measurement`). Superseded by [793c12 Are operating modes needed at all?](#793c12-are-operating-modes-needed-at-all): measurement is always recorded ([793a5e Auditability and measurability](793a_product-spec.md#793a5e-auditability-and-measurability)).
7. **Precision** is judged by a separate grader, reported with its sample size ([793a3i Precision is graded independently](793a_product-spec.md#793a3i-precision-is-graded-independently)).
8. **Capture rate** must exceed 5.6% on the point estimate, no margin ([793a3h Capture rate exceeds the baseline](793a_product-spec.md#793a3h-capture-rate-exceeds-the-baseline)).
9. **Lesson** is any corrected failure, including the agent's own overturned mistake; wider than learn's
   definition, tracked in #846 ([793a1 Problem Statement](793a_product-spec.md#793a1-problem-statement), [793a3a Learn is performed for the working agent](793a_product-spec.md#793a3a-learn-is-performed-for-the-working-agent)).
10. **Secrets** are a MUST NOT ([793a4d Sensitive content](793a_product-spec.md#793a4d-sensitive-content)).
11. **New IDs** for nodes whose meaning reversed; old IDs retired ([793b14 ID handling](793b_decision-log.md#793b14-id-handling)).
12. **N = 20** per arm accepted, proposed marker removed ([793a5f Effect on outcomes](793a_product-spec.md#793a5f-effect-on-outcomes)).
13. **Attributable presentation only**; the non-imperative rule and refusal check are dropped ([793a2d Attributable presentation](793a_product-spec.md#793a2d-attributable-presentation)).
14. **No self-links** on `Requirement:` lines; canonical specs tracked in #847.

Open for Joe: whether the log keeps the extra fields it records beyond his answers is [793c19 Does the log keep the depth, each query's phrases and the glance escalation?](#793c19-does-the-log-keep-the-depth-each-querys-phrases-and-the-glance-escalation).

## 793c8 What does the System inject when only incidental items come back?

- **Status:** answered (Joe, 2026-10-11): (a) inject what recall delivers today, the depth's whole output including incidental items. Trimming is a follow-up, [#848](https://github.com/toejough/engram/issues/848), gated on #845 data.
- **Context:** At a moment whose queries return only incidental items (explore picks per [792a7i Half of the notes are explored](../../specs/product/memory-query.md#792a7i-half-of-the-notes-are-explored), newest chunks per [792a7o The newest chunks](../../specs/product/memory-query.md#792a7o-the-newest-chunks), ride-alongs per [792a7n Superseding notes ride along](../../specs/product/memory-query.md#792a7n-superseding-notes-ride-along)) and no direct match, `793a2j Volume equals the depth's output` would inject all of it.
- **Options:**
  - (a) everything the queries return.
  - (b) only direct matches and their ride-alongs.
  - (c) the full output when at least one direct match exists, nothing otherwise.
- **Recommendation:** (c). It keeps the "depth's output" rule when something matches and avoids injecting noise when nothing does.
- **Applied in:** [793a2e No re-injection](793a_product-spec.md#793a2e-no-re-injection), `793a2j Volume equals the depth's output` and `793a2a1c Only incidental items come back`; `793a2a1b` and `793a2e1b` are retired. Decision: [793b46 Inject what recall delivers today](793b_decision-log.md#793b46-inject-what-recall-delivers-today).
- **Blocks:** [793a2e No re-injection](793a_product-spec.md#793a2e-no-re-injection), `793a2j Volume equals the depth's output`, `793a2a1b No relevant note exists`, `793a2e1b Irrelevant note suppressed`.

## 793c9 Does the System activate notes?

- **Status:** answered (Joe, 2026-10-11): (c) notes the agent went on to use, judged at turn-end capture.
- **Context:** Recall activates the notes it used (Step 2.7, [792a14l Only the notes that were used are activated](../../specs/product/recall.md#792a14l-only-the-notes-that-were-used-are-activated)). When the System injects notes, nobody activates the ones the agent applies.
- **Options:**
  - (a) never.
  - (b) every injected note.
  - (c) notes the agent went on to use, judged at turn-end capture.
- **Recommendation:** (c). It matches recall's "only the notes that were used" rule at a moment the System already evaluates.
- **Applied in:** [793a3k Use is inferred from the transcript](793a_product-spec.md#793a3k-use-is-inferred-from-the-transcript). Decision: [793b47 Used notes are activated at turn-end capture](793b_decision-log.md#793b47-used-notes-are-activated-at-turn-end-capture).
- **Blocks:** a new requirement.

## 793c10 What is the default mode for an existing install that updates?

- **Status:** retired by [793c12 Are operating modes needed at all?](#793c12-are-operating-modes-needed-at-all) (Joe, 2026-10-11): there are no modes.
- **Context:** `793a4e Operating modes and measurement` makes active the default for a fresh install ([793c3 What is the default mode for a fresh install?](#793c3-what-is-the-default-mode-for-a-fresh-install)). An existing install that updates has its own mode choice, and an update could start writing to its vault.
- **Options:**
  - (a) active.
  - (b) observe.
  - (c) off until opt-in.
- **Recommendation:** (a), consistent with the fresh-install answer; measurement stays on either way.
- **Blocks:** `793a4e Operating modes and measurement`.

## 793c11 What does the latency bound cover?

- **Status:** answered (Joe, 2026-10-11): (b) the instruction path's full lookup, phrasing plus queries, on the same input.
- **Context:** [793a5g No added delay](793a_product-spec.md#793a5g-no-added-delay) caps added latency at the latency of the depth's queries. The instruction path's lookup also includes the agent phrasing its query, which the System does too.
- **Options:**
  - (a) queries only.
  - (b) the instruction path's full lookup (phrasing plus queries) on the same input.
  - (c) queries plus a fixed allowance.
- **Recommendation:** (b). It compares like with like and needs no fixed number that would go stale.
- **Applied in:** [793a5g No added delay](793a_product-spec.md#793a5g-no-added-delay). Decision: [793b48 Latency covers the instruction path's full lookup](793b_decision-log.md#793b48-latency-covers-the-instruction-paths-full-lookup).
- **Blocks:** [793a5g No added delay](793a_product-spec.md#793a5g-no-added-delay).

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
- **Context:** [793a3k Use is inferred from the transcript](793a_product-spec.md#793a3k-use-is-inferred-from-the-transcript) covers the notes the System injected during the turn, per [793c9 Does the System activate notes?](#793c9-does-the-system-activate-notes). When recall runs as an instruction, it activates the notes it used ([792a14l Only the notes that were used are activated](../../specs/product/recall.md#792a14l-only-the-notes-that-were-used-are-activated)). A note the agent retrieved by its own lookup and used is activated only if that lookup was a recall.
- **Options:**
  - (a) injected notes only.
  - (b) also notes the agent retrieved itself and used.
- **Recommendation:** (a). Recall already activates its own retrievals, and (b) widens the System's judgment to content it did not deliver.
- **Blocks:** [793a3k Use is inferred from the transcript](793a_product-spec.md#793a3k-use-is-inferred-from-the-transcript).

## 793c14 Should reinforcing an existing note be a MUST?

- **Status:** answered (Joe, 2026-10-11): (b) reinforcement is a MUST; `793a3c No duplicates` and `793a3c1a Existing note already covers the lesson` updated. Decision: [793b52 Reinforcement is required](793b_decision-log.md#793b52-reinforcement-is-required).
- **Context:** `793a3c No duplicates` says the System SHOULD reinforce an equivalent existing note. The second review ([793b35 M8: "Reinforced" is not backed by a MUST](793b_decision-log.md#793b35-m8-reinforced-is-not-backed-by-a-must)) noted the scenario asserted a mark the requirement did not require, and the scenario no longer does.
- **Options:**
  - (a) keep SHOULD, with no scenario.
  - (b) make it a MUST, with a scenario asserting the existing note's recency or reinforcement record changed.
- **Recommendation:** (a). The owner has not asked for more, and a duplicate is already forbidden.
- **Blocks:** `793a3c No duplicates`.

## 793c15 Does the System inject chunks with their content?

- **Status:** answered (Joe, 2026-10-11): (b) lazy chunks, as recall runs the query; add a rule against re-injecting the same chunk stub, and retire the chunk-content scenario.
- **Applied in:** [793a2e No re-injection](793a_product-spec.md#793a2e-no-re-injection), `793a2j Volume equals the depth's output`, `793a2j1e Chunks are delivered as stubs`, [793a2e1h A chunk whose content the agent fetched is not injected](793a_product-spec.md#793a2e1-uat-scenarios) and [793a2e1i A chunk stub already shown is not injected again](793a_product-spec.md#793a2e1-uat-scenarios); `793a2e1f` is retired. Decision: [793b62 Chunks are injected lazily, as stubs](793b_decision-log.md#793b62-chunks-are-injected-lazily-as-stubs).
- **Applied in commit:** 1adbb77e.
- **Context:** [792a14f One unified query](../../specs/product/recall.md#792a14f-one-unified-query) requires `--lazy-chunks`, which prints chunks as path and score only ([792a7s Lazy chunks](../../specs/product/memory-query.md#792a7s-lazy-chunks)). The baseline measurement in [793c2 Are the volume bounds and latency budget accepted?](#793c2-are-the-volume-bounds-and-latency-budget-accepted) used the query's defaults, which print chunk content. `793a2e1f A chunk shown without content is delivered with it` has the System deliver a chunk with its content.
- **Options:**
  - (a) yes: the System runs the query with the defaults, without `--lazy-chunks`, so `793a2e1f A chunk shown without content is delivered with it` stands.
  - (b) no: the System runs it lazy as recall does, which needs a "do not re-inject the same stub" rule and retires `793a2e1f A chunk shown without content is delivered with it`.
- **Recommendation:** (a). It matches the volume that was measured, and the System has no agent turn in which to fetch a stub.
- **Blocks:** [793a2e No re-injection](793a_product-spec.md#793a2e-no-re-injection), `793a2e1f A chunk shown without content is delivered with it`, [793a2e1h A chunk whose content the agent fetched is not injected](793a_product-spec.md#793a2e1-uat-scenarios) and the volume figures in [793a5e Auditability and measurability](793a_product-spec.md#793a5e-auditability-and-measurability).

## 793c16 At a deep moment, does the System's coverage write activate the note it covers?

- **Status:** answered (Joe, 2026-10-11): when a note covers a cluster, attach the cluster's chunks to it as evidence and inject the covering note into the working agent (subject to no re-injection); mark it used only on the same transcript evidence as any injected note ([793c21 How does the System know a note was used?](#793c21-how-does-the-system-know-a-note-was-used)).
- **Applied in:** `793a3g Deep recall's writes are the System's` (scenarios `793a3g1c A covered cluster gets evidence and its note is injected`, `793a3g1d A covering note already in context is not injected again` and `793a3g1e A near cluster amends the note`) and [793a3k Use is inferred from the transcript](793a_product-spec.md#793a3k-use-is-inferred-from-the-transcript). Decision: [793b63 A coverage write attaches evidence and injects, and never activates](793b_decision-log.md#793b63-a-coverage-write-attaches-evidence-and-injects-and-never-activates).
- **Applied in commit:** 1adbb77e.
- **Context:** at a deep moment the System writes coverage ([792a14j Coverage is judged by the agent and written once per cluster](../../specs/product/recall.md#792a14j-coverage-is-judged-by-the-agent-and-written-once-per-cluster), `793a3g Deep recall's writes are the System's`) by adding chunk sources to an existing note with `engram amend`. `engram amend --activate` would also bump that note's recency, which [793a3k Use is inferred from the transcript](793a_product-spec.md#793a3k-use-is-inferred-from-the-transcript) otherwise reserves for notes the agent used ([793c9 Does the System activate notes?](#793c9-does-the-system-activate-notes)).
- **Options:**
  - (a) no: the System adds chunk sources without `--activate`, and only agent-used notes are activated ([793a3k Use is inferred from the transcript](793a_product-spec.md#793a3k-use-is-inferred-from-the-transcript)).
  - (b) yes: a coverage write activates the note it covers, as an exception to [793a3k Use is inferred from the transcript](793a_product-spec.md#793a3k-use-is-inferred-from-the-transcript).
- **Recommendation:** (a). A coverage write records that a note covers a cluster, not that the agent used it, so activating it would keep unused notes fresh.
- **Blocks:** `793a3g Deep recall's writes are the System's` and [793a3k Use is inferred from the transcript](793a_product-spec.md#793a3k-use-is-inferred-from-the-transcript).

## 793c17 Does the latency bound include reading the glance results?

- **Status:** answered (Joe, 2026-10-11): (a) the bound includes reading the glance results that escalation and the recommendation check depend on.
- **Applied in:** [793a5g No added delay](793a_product-spec.md#793a5g-no-added-delay). Decision: [793b64 Latency includes reading the results](793b_decision-log.md#793b64-latency-includes-reading-the-results).
- **Applied in commit:** 1adbb77e.
- **Context:** [793a5g No added delay](793a_product-spec.md#793a5g-no-added-delay) bounds added latency by the instruction path's full lookup on the same input ([793c11 What does the latency bound cover?](#793c11-what-does-the-latency-bound-cover)). Escalating from glance to deep (`793a2h Recall depth per moment`, [792a14d Glance escalates to deep for a recent standard](../../specs/product/recall.md#792a14d-glance-escalates-to-deep-for-a-recent-standard)) and the recommendation check ([792a14o An emergent recommendation is checked against the vault](../../specs/product/recall.md#792a14o-an-emergent-recommendation-is-checked-against-the-vault)) depend on reading the glance results, which takes agent time.
- **Options:**
  - (a) yes: reading the results is part of the instruction path's full lookup, so the bound includes it.
  - (b) no: the bound covers phrases plus queries only.
  - (c) drop the result-dependent escalation triggers and fix the depth up front.
- **Recommendation:** (a). It compares like with like and keeps the escalation triggers.
- **Blocks:** [793a5g No added delay](793a_product-spec.md#793a5g-no-added-delay), `793a2h Recall depth per moment`, `793a2h1c An unresolved gap uses deep` and [793a5e1f An escalated moment records both depths](793a_product-spec.md#793a5e1-uat-scenarios).

## 793c18 How is the System installed and turned off?

- **Status:** answered (Joe, 2026-10-11): (a), with installation in this spec and without the harness's own disable mechanism of option (a). Extend the existing install so it installs what the System needs; existing systems get it through `engram update`. The System is turned off by uninstalling, and uninstall is extended to remove it. See [793c22 There is no uninstall command today](#793c22-there-is-no-uninstall-command-today).
- **Applied in:** [793a4h Installed by engram update](793a_product-spec.md#793a4h-installed-by-engram-update), [793a4g Active whenever installed](793a_product-spec.md#793a4g-active-whenever-installed) and the Integration term in [793a1 Problem Statement](793a_product-spec.md#793a1-problem-statement); `793a4g1b` is retired. Decision: [793b65 Installation is part of this spec](793b_decision-log.md#793b65-installation-is-part-of-this-spec).
- **Applied in commit:** 1adbb77e.
- **Context:** [793a4g Active whenever installed](793a_product-spec.md#793a4g-active-whenever-installed) makes the System active whenever the integration is installed and enabled, and inert when disabled or uninstalled. It does not say how the integration gets installed, or how an operator disables it.
- **Options:**
  - (a) `engram update` installs it by default, and it is disabled through the harness's own disable mechanism or by uninstalling.
  - (b) opt-in like guidance ([792a11l Guidance is managed only after opt-in](../../specs/product/install-and-update.md#792a11l-guidance-is-managed-only-after-opt-in)), then active.
- **Recommendation:** (a). It matches "active whenever installed" and adds no engram setting.
- **Blocks:** [793a4g Active whenever installed](793a_product-spec.md#793a4g-active-whenever-installed).

## 793c19 Does the log keep the depth, each query's phrases and the glance escalation?

- **Status:** answered (Joe, 2026-10-11): (a) keep the log fields.
- **Applied in:** [793a5e Auditability and measurability](793a_product-spec.md#793a5e-auditability-and-measurability) (the depth, phrase and escalation fields are kept). Decision: [793b66 The log keeps depth, phrases and escalation](793b_decision-log.md#793b66-the-log-keeps-depth-phrases-and-escalation).
- **Applied in commit:** 1adbb77e.
- **Context:** raised unnumbered at the end of [793c7 Review answers (2026-10-11)](#793c7-review-answers-2026-10-11). [793a5e Auditability and measurability](793a_product-spec.md#793a5e-auditability-and-measurability) also logs the depth, the phrases of each query and whether glance escalated, which is more than Joe's answers asked for ([793b8 Per-moment record sits in the existing log requirement](793b_decision-log.md#793b8-per-moment-record-sits-in-the-existing-log-requirement), [793b43 Author's choice accepted: log depth, phrases and escalation](793b_decision-log.md#793b43-authors-choice-accepted-log-depth-phrases-and-escalation)).
- **Options:**
  - (a) keep the fields.
  - (b) drop them, which makes the `793a2h1a A routine decision uses glance` and `793a2h1c An unresolved gap uses deep` scenarios unverifiable from the log.
- **Recommendation:** (a). The depth and escalation scenarios can only be checked from the operator-accessible log.
- **Blocks:** [793a5e Auditability and measurability](793a_product-spec.md#793a5e-auditability-and-measurability), `793a2h1a A routine decision uses glance`, `793a2h1c An unresolved gap uses deep` and [793a5e1f An escalated moment records both depths](793a_product-spec.md#793a5e1-uat-scenarios).

## 793c20 Must injected chunks identify their source like notes?

- **Status:** answered (Joe, 2026-10-11): (a) every injected item identifies its source: a note by its file name, a chunk by its id.
- **Applied in:** [793a2d Attributable presentation](793a_product-spec.md#793a2d-attributable-presentation) and [793a2d1b Injected chunk names its id](793a_product-spec.md#793a2d1-uat-scenarios). Decision: [793b67 Every injected item names its source](793b_decision-log.md#793b67-every-injected-item-names-its-source).
- **Applied in commit:** 1adbb77e.
- **Context:** [793a2d Attributable presentation](793a_product-spec.md#793a2d-attributable-presentation) requires injected notes to identify their source note. The System also injects chunks, which have an id but no note file name.
- **Options:**
  - (a) yes: "Injected items MUST identify their source (a note's file name, a chunk's id)".
  - (b) chunks are exempt.
- **Recommendation:** (a). A chunk the agent cannot trace to its source cannot be weighed or looked up.
- **Blocks:** [793a2d Attributable presentation](793a_product-spec.md#793a2d-attributable-presentation).

## 793c21 How does the System know a note was used?

- **Status:** answered (Joe, 2026-10-11): (a) infer use from the transcript at turn-end capture, log each verdict with its evidence, and have the precision grader sample activation verdicts.
- **Applied in:** [793a3k Use is inferred from the transcript](793a_product-spec.md#793a3k-use-is-inferred-from-the-transcript), [793a3l Activation accuracy is graded independently](793a_product-spec.md#793a3l-activation-accuracy-is-graded-independently) and [793a5e Auditability and measurability](793a_product-spec.md#793a5e-auditability-and-measurability). Decision: [793b68 Use is inferred from the transcript and graded](793b_decision-log.md#793b68-use-is-inferred-from-the-transcript-and-graded).
- **Applied in commit:** 1adbb77e.
- **Context:** Joe asked (2026-10-11, on [793c16 At a deep moment, does the System's coverage write activate the note it covers?](#793c16-at-a-deep-moment-does-the-systems-coverage-write-activate-the-note-it-covers)) whether the System can know what the agent actually used. Today, recall's activation step ([792a14l Only the notes that were used are activated](../../specs/product/recall.md#792a14l-only-the-notes-that-were-used-are-activated)) relies on the agent declaring which notes it drew on. The System has no such declaration. It can only infer use from the session transcript after the turn: the agent quoting or citing the note, following its steps, or taking an action the note prescribes. Inference misses unstated use and can mistake coincidence for use. Asking the working agent to declare use would put memory bookkeeping back into its context.
- **Options:**
  - (a) Infer use from the transcript at turn-end capture, log each verdict with its evidence, and have the [793a3i Precision is graded independently](793a_product-spec.md#793a3i-precision-is-graded-independently) grader also sample activation verdicts so their accuracy is measured.
  - (b) Treat every injected note as used. Activation is then recency of injection, with no judgment.
  - (c) Don't activate at all until a reliable signal exists. Record it as a follow-up.
- **Recommendation:** (a). It keeps the main agent's context clean, matches how capture already judges turns, and makes the inference's accuracy measurable instead of assumed.
- **Blocks:** [793a3k Use is inferred from the transcript](793a_product-spec.md#793a3k-use-is-inferred-from-the-transcript) and [793c16 At a deep moment, does the System's coverage write activate the note it covers?](#793c16-at-a-deep-moment-does-the-systems-coverage-write-activate-the-note-it-covers).

## 793c22 There is no uninstall command today

- **Status:** answered (Joe, 2026-10-11): (b) a general `engram uninstall` in its own change, [#849](https://github.com/toejough/engram/issues/849); this change depends on it.
- **Applied in:** [793a4g Active whenever installed](793a_product-spec.md#793a4g-active-whenever-installed) (uninstalling removes the System); the dependency on #849 is recorded in [793b69 The uninstall command is a separate change](793b_decision-log.md#793b69-the-uninstall-command-is-a-separate-change).
- **Applied in commit:** 1adbb77e.
- **Context:** [793c18 How is the System installed and turned off?](#793c18-how-is-the-system-installed-and-turned-off) says to extend the existing uninstall. `engram --help` has no uninstall command. The install-and-update capability ([install-and-update](../../specs/product/install-and-update.md)) covers installing and syncing only; today, removing engram means deleting the binary and the deployed links by hand.
- **Options:**
  - (a) Add an `engram uninstall` command in this change that removes the System, and only the System, so turning it off is one command.
  - (b) Add a general `engram uninstall` that removes everything `engram update` installed, the System included, as its own issue and change. This change depends on it.
  - (c) Document manual removal steps for the System, with no command.
- **Recommendation:** (b). Uninstall is a capability of install-and-update, not of the parallel agent. Building it separately keeps one owner and gives every installed piece a removal path.
- **Blocks:** the turn-off requirement in [793a4g Active whenever installed](793a_product-spec.md#793a4g-active-whenever-installed) and the new install requirement.

## 793c23 Is a note amended as near injected?

- **Status:** answered (Joe, 2026-10-11): (b) inject the amended note; a changed note is not present in its changed form.
- **Applied in:** [793a2e No re-injection](793a_product-spec.md#793a2e-no-re-injection), `793a3g Deep recall's writes are the System's`, [793a2e1j An amended note is delivered in its changed form](793a_product-spec.md#793a2e1-uat-scenarios) and `793a3g1e A near cluster amends the note`. Decision: [793b71 A near-amended note is injected](793b_decision-log.md#793b71-a-near-amended-note-is-injected).
- **Applied in commit:** da69f814.
- **Context:** [793c16 At a deep moment, does the System's coverage write activate the note it covers?](#793c16-at-a-deep-moment-does-the-systems-coverage-write-activate-the-note-it-covers) was answered for the covered outcome only: the covering note is injected. `793a3g Deep recall's writes are the System's` says nothing about injecting a near note after it is amended with re-synthesized content ([792a14j Coverage is judged by the agent and written once per cluster](../../specs/product/recall.md#792a14j-coverage-is-judged-by-the-agent-and-written-once-per-cluster)). The version the working agent saw, if it saw one, differs from the amended note.
- **Options:**
  - (a) no: a near note is amended and not injected again.
  - (b) yes: the amended note is injected, which needs a rule that an amended version is not the version already present under [793a2e No re-injection](793a_product-spec.md#793a2e-no-re-injection).
- **When this happens:** only at deep moments (a weighty or irreversible decision, a gap glance could not resolve, or a recently surfaced standard), and only for a cluster of results whose content overlaps an existing note but adds to or changes it. An example is a note saying "use targ for tests" and a cluster of recent sessions also saying "and pass -tags=integration for dev tests". Recall rewrites the note to include the new detail.
- **Recommendation (revised 2026-10-11 by Orchestrator (Claude)):** (b). In recall today the agent that does the deep pass also writes the amendment, so the amended content is in its context. Injecting the amended note keeps that behavior. The earlier recommendation (a) would have left the agent with the older text.
- **Blocks:** `793a3g Deep recall's writes are the System's` and `793a3g1e A near cluster amends the note`.

## 793c24 Does the covering note's injection add anything to the depth's output?

- **Status:** answered (Joe, 2026-10-11): (a) keep the explicit clause and inject the cluster's items too; "covering note instead of its cluster" is added to [#848](https://github.com/toejough/engram/issues/848).
- **Applied in:** `793a3g Deep recall's writes are the System's` (the clause is kept) and `793a2j Volume equals the depth's output` (the cluster's items are injected as before). Decision: [793b72 Covering note and cluster are both injected](793b_decision-log.md#793b72-covering-note-and-cluster-are-both-injected).
- **Applied in commit:** da69f814.
- **Context:** [793c16 At a deep moment, does the System's coverage write activate the note it covers?](#793c16-at-a-deep-moment-does-the-systems-coverage-write-activate-the-note-it-covers) has the System inject the covering note. A covering note is a candidate from the moment's query results, which `793a2j Volume equals the depth's output` already injects, so the clause matters only when the note is not among the injected results.
- **Options:**
  - (a) keep the clause in `793a3g Deep recall's writes are the System's` as an explicit guarantee.
  - (b) drop it, since `793a2j Volume equals the depth's output` covers the case.
- **Owner follow-up (Joe, 2026-10-11):** is it a reasonable optimization to inject only the covering note and not the cluster's own items?
- **Assessment (Orchestrator (Claude)):** the gain is small and something is lost. Under [793c15 Does the System inject chunks with their content?](#793c15-does-the-system-inject-chunks-with-their-content) the cluster's chunks are already only stubs (path and score), so dropping them saves little. Dropping them removes the agent's pointers to the specific past sessions behind the note. Those sessions carry examples, edge cases and context the note's summary leaves out, which the agent can fetch when it needs them. This is a trim of today's output, so per [793c8 What does the System inject when only incidental items come back?](#793c8-what-does-the-system-inject-when-only-incidental-items-come-back) it belongs with [#848](https://github.com/toejough/engram/issues/848) and needs measurement.
- **Recommendation:** (a). Keep the explicit clause, inject the cluster's items as well, and add "covering note instead of its cluster" to #848.
- **Blocks:** `793a3g Deep recall's writes are the System's` and `793a3g1c A covered cluster gets evidence and its note is injected`.

## 793c25 When are the covering and amended notes from a deep moment injected?

- **Status:** retired by [793c29 Is the System a delegate for the existing memory skills?](#793c29-is-the-system-a-delegate-for-the-existing-memory-skills) (Joe, 2026-10-11): the System does what the skill does.
- **Owner follow-up (Joe, 2026-10-11):** why do note writes only happen at the end of a turn? See [793c29 Is the System a delegate for the existing memory skills?](#793c29-is-the-system-a-delegate-for-the-existing-memory-skills).
- **Context:** `793a3g Deep recall's writes are the System's` says the System's capture performs the deep moment's writes, but capture runs asynchronously at turn end ([793a3a Learn is performed for the working agent](793a_product-spec.md#793a3a-learn-is-performed-for-the-working-agent), `793a3f Capture does not delay the session`), while the covering and amended notes it injects must reach the working agent. A finished subagent has no context left to receive anything. The coverage judgment and the writes add time outside [793a5g No added delay](793a_product-spec.md#793a5g-no-added-delay)'s comparison.
- **Options:**
  - (a) At the same moment, before the agent's next action. The coverage judgment and writes are added to [793a5g No added delay](793a_product-spec.md#793a5g-no-added-delay)'s comparison, because the instruction path's deep recall performs the same step.
  - (b) At the agent's next recall moment. Nothing is injected if the agent has finished.
  - (c) At the same moment if it fits in the [793a5g No added delay](793a_product-spec.md#793a5g-no-added-delay) bound; otherwise as (b).
- **Recommendation:** (a). The instruction path's deep recall judges coverage and writes before the agent's next action, so (a) delivers the same notes at the same time, and a finished subagent still gets them.
- **Blocks:** `793a3g Deep recall's writes are the System's` and [793a5g No added delay](793a_product-spec.md#793a5g-no-added-delay). Raised by finding H4 of the fourth review ([793b89 Fourth review: findings waiting on open questions](793b_decision-log.md#793b89-fourth-review-findings-waiting-on-open-questions)).

## 793c26 What does the System's recommendation check (792a14o) key on?

- **Status:** retired by [793c29 Is the System a delegate for the existing memory skills?](#793c29-is-the-system-a-delegate-for-the-existing-memory-skills) (Joe, 2026-10-11): the System does what the skill does.
- **Owner follow-up (Joe, 2026-10-11):** leaning (c). Asked why the System can't simply take over the memory actions from the main agent without exceptions. See [793c29 Is the System a delegate for the existing memory skills?](#793c29-is-the-system-a-delegate-for-the-existing-memory-skills).
- **Context:** [792a14o An emergent recommendation is checked against the vault](../../specs/product/recall.md#792a14o-an-emergent-recommendation-is-checked-against-the-vault) fires on a recommendation that is not in the Step 0 plan. The System has no plan and never ships a recommendation, yet `793a2j Volume equals the depth's output` and [793a5g No added delay](793a_product-spec.md#793a5g-no-added-delay) include the check.
- **Options:**
  - (a) The System's own synthesis, with the query text standing in for the plan.
  - (b) Drop the check at System moments, and remove it from `793a2j Volume equals the depth's output` and [793a5g No added delay](793a_product-spec.md#793a5g-no-added-delay).
  - (c) Run it at the next moment, keyed on a recommendation the working agent stated.
- **Recommendation:** (b). The System ships no recommendation, so there is nothing for the check to key on, and (a) and (c) invent a trigger that recall does not have.
- **Blocks:** `793a2j Volume equals the depth's output` and [793a5g No added delay](793a_product-spec.md#793a5g-no-added-delay). Raised by finding M2 of the fourth review ([793b89 Fourth review: findings waiting on open questions](793b_decision-log.md#793b89-fourth-review-findings-waiting-on-open-questions)).

## 793c27 Does the System sweep the chunk index (792a14b)?

- **Status:** answered (Joe, 2026-10-11): (c) sweep asynchronously at turn-end capture.
- **Applied in:** [793a2l The chunk index is swept after the turn](793a_product-spec.md#793a2l-the-chunk-index-is-swept-after-the-turn) (kept as a difference from recall's own sweep). Decision: [793b95 The sweep runs after the turn](793b_decision-log.md#793b95-the-sweep-runs-after-the-turn).
- **Applied in commit:** 63b15d9b.
- **Context:** [792a14b The index is swept once per session](../../specs/product/recall.md#792a14b-the-index-is-swept-once-per-session) has recall run `engram ingest --auto` before its first query of a session, so the index holds the latest sessions and document edits. The product spec does not say whether the System does. A sweep takes time before the first query and writes to the chunk index.
- **Options:**
  - (a) Once per session before its first query, as recall does. Its time counts in [793a5g No added delay](793a_product-spec.md#793a5g-no-added-delay), and its index writes are added to [793a4c Memory purview only](793a_product-spec.md#793a4c-memory-purview-only).
  - (b) Never.
  - (c) Asynchronously at turn-end capture.
- **Recommendation:** (a), which matches [793c8 What does the System inject when only incidental items come back?](#793c8-what-does-the-system-inject-when-only-incidental-items-come-back) "what recall delivers today".
- **Blocks:** `793a2j Volume equals the depth's output`, [793a4c Memory purview only](793a_product-spec.md#793a4c-memory-purview-only) and [793a5g No added delay](793a_product-spec.md#793a5g-no-added-delay). Raised by finding M3 of the fourth review ([793b89 Fourth review: findings waiting on open questions](793b_decision-log.md#793b89-fourth-review-findings-waiting-on-open-questions)).

## 793c28 How is 793a5g's latency comparison measured?

- **Status:** retired by [793c29 Is the System a delegate for the existing memory skills?](#793c29-is-the-system-a-delegate-for-the-existing-memory-skills) (Joe, 2026-10-11): the System does what the skill does.
- **Owner follow-up (Joe, 2026-10-11):** what is being measured? See [793c29 Is the System a delegate for the existing memory skills?](#793c29-is-the-system-a-delegate-for-the-existing-memory-skills).
- **Context:** [793a5g No added delay](793a_product-spec.md#793a5g-no-added-delay) compares the latency the System adds at a moment with the time the instruction path's lookup takes at that moment. Both are timings, and each varies between runs.
- **Options:**
  - (a) Median added latency against median lookup latency, over N repeats per moment kind.
  - (b) A single sample per moment.
  - (c) Aggregate per session.
- **Recommendation:** (a), because a single sample of two noisy timings cannot pass reliably.
- **Blocks:** [793a5g No added delay](793a_product-spec.md#793a5g-no-added-delay). Raised by finding M6 of the fourth review ([793b89 Fourth review: findings waiting on open questions](793b_decision-log.md#793b89-fourth-review-findings-waiting-on-open-questions)).

## 793c29 Is the System a delegate for the existing memory skills?

- **Status:** answered (Joe, 2026-10-11): (a) restructure 793a as a delegation of the existing memory skills. [793c25 When are the covering and amended notes from a deep moment injected?](#793c25-when-are-the-covering-and-amended-notes-from-a-deep-moment-injected), [793c26 What does the System's recommendation check (792a14o) key on?](#793c26-what-does-the-systems-recommendation-check-792a14o-key-on) and [793c28 How is 793a5g's latency comparison measured?](#793c28-how-is-793a5gs-latency-comparison-measured) are retired by it.
- **Applied in:** the whole of [`793a_product-spec.md`](./793a_product-spec.md): [793a2a Recall is performed for the working agent](793a_product-spec.md#793a2a-recall-is-performed-for-the-working-agent), [793a3a Learn is performed for the working agent](793a_product-spec.md#793a3a-learn-is-performed-for-the-working-agent) and [793a2k The working agent does not run recall or learn](793a_product-spec.md#793a2k-the-working-agent-does-not-run-recall-or-learn) state the delegation; the requirements that re-specified skill behavior are retired. Decisions: [793b90 Restructure 793a as a delegation of the memory skills](793b_decision-log.md#793b90-restructure-793a-as-a-delegation-of-the-memory-skills), [793b92 Requirements that re-specified skill behavior are retired](793b_decision-log.md#793b92-requirements-that-re-specified-skill-behavior-are-retired), [793b94 No added delay, and how it is measured](793b_decision-log.md#793b94-no-added-delay-and-how-it-is-measured) and [793b97 ID map for the restructure](793b_decision-log.md#793b97-id-map-for-the-restructure).
- **Applied in commit:** 63b15d9b.
- **Context:** Joe (2026-10-11): "I really don't understand why the watching agent can't just take the memory actions over from the main agent. Why all these exceptions? I'm not trying to do anything different. I'm trying to offload the context." The draft, and every revision since, re-specified recall and learn behavior requirement by requirement. The draft defined capture as evaluating each completed turn (formerly CAP-1), which is why writes happen at turn end. That and the other re-specifications produced the exceptions in [793c23 Is a note amended as near injected?](#793c23-is-a-note-amended-as-near-injected) through [793c28 How is 793a5g's latency comparison measured?](#793c28-how-is-793a5gs-latency-comparison-measured). The existing skills already define when they run and what they do: [recall](../../specs/product/recall.md), [learn](../../specs/product/learn.md), and the moments in [ambient-guidance](../../specs/product/ambient-guidance.md).
- **Options:**
  - (a) Restructure 793a as a delegation. The System runs recall and learn exactly as their canonical specs define them, at the moments the guidance defines, on the working agent's behalf, including their writes, activation and sweep, when and as those skills do them. The working agent receives only what the skill would have put in front of it, and no longer runs the skills itself. 793a then specifies only what is new: noticing the moments without the agent, handing the results to the agent, failure isolation, the System's own sessions, measurement, and install and uninstall. Most re-specified requirements, and the questions about them (including [793c25 When are the covering and amended notes from a deep moment injected?](#793c25-when-are-the-covering-and-amended-notes-from-a-deep-moment-injected), [793c26 What does the System's recommendation check (792a14o) key on?](#793c26-what-does-the-systems-recommendation-check-792a14o-key-on) and [793c28 How is 793a5g's latency comparison measured?](#793c28-how-is-793a5gs-latency-comparison-measured)), are retired.
  - (b) Keep the current structure and answer [793c25 When are the covering and amended notes from a deep moment injected?](#793c25-when-are-the-covering-and-amended-notes-from-a-deep-moment-injected)–c28 individually.
- **Recommendation:** (a). It states Joe's intent directly and removes the exceptions by construction. Owner decisions that change the skills' behavior stay, as deliberate differences from the skills: direct writes with no main-agent involvement, the wider lesson definition (#846), inferred use, and the volume and latency measurements. A follow-up question will list each one for Joe to confirm or drop.
- **Blocks:** every requirement in 793a, and [793c25 When are the covering and amended notes from a deep moment injected?](#793c25-when-are-the-covering-and-amended-notes-from-a-deep-moment-injected)–[793c28 How is 793a5g's latency comparison measured?](#793c28-how-is-793a5gs-latency-comparison-measured).

## 793c30 Which earlier decisions stay as deliberate differences from the skills?

- **Status:** answered (Joe, 2026-10-11):
  - D1 keep.
  - D2 keep.
  - D3 keep, and more: agents no longer report lessons back at all, because the System captures them. While the Integration is installed, the LESSONS-line contract ([lessons-contract](../../specs/product/lessons-contract.md)) is replaced, not supplemented.
  - D4 keep: the System uses the wider definition now, ahead of #846.
  - D5 drop: follow learn's handling of an existing equivalent note.
  - Keep all of N1–N8.
- **Applied in:** D1 [793a2i Recall for every prompt regardless of source](793a_product-spec.md#793a2i-recall-for-every-prompt-regardless-of-source); D2 [793a2e No re-injection](793a_product-spec.md#793a2e-no-re-injection); D3 [793a3m Lessons from subagent sessions are captured directly](793a_product-spec.md#793a3m-lessons-from-subagent-sessions-are-captured-directly) and the operations of [793a6 lessons-contract (the LESSONS line is replaced while the Integration is installed)](793a_product-spec.md#793a6-modified-lessons-contract-the-lessons-line-is-replaced-while-the-integration-is-installed); D4 [793a3n Any corrected failure is a lesson](793a_product-spec.md#793a3n-any-corrected-failure-is-a-lesson); D5 dropped; N1 [793a2c Moments are noticed without the agent](793a_product-spec.md#793a2c-moments-are-noticed-without-the-agent); N2 [793a4f No recall or capture in its own sessions](793a_product-spec.md#793a4f-no-recall-or-capture-in-its-own-sessions); N3 [793a4b Failure isolation](793a_product-spec.md#793a4b-failure-isolation); N4 [793a4h Installed by engram update](793a_product-spec.md#793a4h-installed-by-engram-update) and [793a4g Active whenever installed](793a_product-spec.md#793a4g-active-whenever-installed); N5 [793a5e Auditability and measurability](793a_product-spec.md#793a5e-auditability-and-measurability); N6 [793a3h Capture rate exceeds the baseline](793a_product-spec.md#793a3h-capture-rate-exceeds-the-baseline); N7 [793a3i Precision is graded independently](793a_product-spec.md#793a3i-precision-is-graded-independently) and [793a3l Activation accuracy is graded independently](793a_product-spec.md#793a3l-activation-accuracy-is-graded-independently); N8 [793a2d Attributable presentation](793a_product-spec.md#793a2d-attributable-presentation). Decisions: [793b91 Which differences and new capabilities stay](793b_decision-log.md#793b91-which-differences-and-new-capabilities-stay) and [793b93 The LESSONS contract is modified, not removed](793b_decision-log.md#793b93-the-lessons-contract-is-modified-not-removed).
- **Applied in commit:** 63b15d9b.
- **Context:** under [793c29 Is the System a delegate for the existing memory skills?](#793c29-is-the-system-a-delegate-for-the-existing-memory-skills) the System does what recall and learn do. Some earlier owner decisions change that behavior. The items below are in one of three groups. **Inherent** items follow from delegation itself and stay. **New** items are capabilities the skills don't have, and stay unless dropped. **Differences** change what the skills do; each needs a keep or drop.
- **Inherent (stay):**
  - the working agent no longer runs recall or learn ([793c4 Are captured lessons written directly, or held for `curate` review?](#793c4-are-captured-lessons-written-directly-or-held-for-curate-review));
  - use is inferred from the transcript, because the System cannot ask the agent ([793c21 How does the System know a note was used?](#793c21-how-does-the-system-know-a-note-was-used));
  - volume equals what the skill would have shown the agent ([793c8 What does the System inject when only incidental items come back?](#793c8-what-does-the-system-inject-when-only-incidental-items-come-back)).
- **New (stay unless dropped):**
  - N1: moments are noticed without the agent, including subagent dispatch.
  - N2: the System's own sessions are excluded from recall and capture.
  - N3: failures don't affect the working session.
  - N4: install through `engram update`, and off by uninstalling ([793c18 How is the System installed and turned off?](#793c18-how-is-the-system-installed-and-turned-off), #849).
  - N5: an always-on measurement log ([793c19 Does the log keep the depth, each query's phrases and the glance escalation?](#793c19-does-the-log-keep-the-depth-each-querys-phrases-and-the-glance-escalation)).
  - N6: a capture-rate target of more than 5.6% ([793c1 What are the capture precision bar and capture-rate target?](#793c1-what-are-the-capture-precision-bar-and-capture-rate-target)).
  - N7: precision and activation graders ([793c7 Review answers (2026-10-11)](#793c7-review-answers-2026-10-11), [793c21 How does the System know a note was used?](#793c21-how-does-the-system-know-a-note-was-used)).
  - N8: injected items name their source ([793c20 Must injected chunks identify their source like notes?](#793c20-must-injected-chunks-identify-their-source-like-notes)).
- **Differences (keep or drop each):**
  - D1: recall at every prompt regardless of source ([793c5 Does 793a2i Recall for every prompt regardless of source keep human-originated prompts as the default?](#793c5-does-793a2i-recall-for-every-prompt-regardless-of-source-keep-human-originated-prompts-as-the-default)). The guidance fires recall at its own moments, not at every prompt. Drop means using the guidance's moments.
  - D2: never inject an item already in the agent's context ([793a2e No re-injection](793a_product-spec.md#793a2e-no-re-injection)). Recall today shows its results regardless. Drop means showing what the skill would show.
  - D3: capture in subagent sessions ([793c6 Does capture also run inside subagent sessions?](#793c6-does-capture-also-run-inside-subagent-sessions)). Today, subagents report lessons to the parent through the LESSONS line ([lessons-contract](../../specs/product/lessons-contract.md)) and the parent learns. Keep means the System also learns directly from subagent sessions.
  - D4: the wider lesson definition, "learn from all failures" ([793c7 Review answers (2026-10-11)](#793c7-review-answers-2026-10-11), #846). Drop means using learn's current definition until #846 widens it for both.
  - D5: reinforcement as a MUST ([793c14 Should reinforcing an existing note be a MUST?](#793c14-should-reinforcing-an-existing-note-be-a-must)). Drop means following learn's own dedup behavior.
- **Options:** for each of D1–D5, (k) keep or (d) drop. Separately, name any N item to drop.
- **Recommendation:**
  - D1: keep. You asked for it, and it widens recall without changing what recall does.
  - D2: keep. It only removes duplicates.
  - D3: keep, because #739 shows lessons die with subagents.
  - D4: drop here, and let #846 change learn for both, so the System never diverges from learn.
  - D5: drop, and follow learn.
  - Keep all N items.
- **Blocks:** the 793a restructure.

## 793c31 Which canonical nodes that ask the agent to run recall or learn change while the Integration is installed?

- **Status:** answered (Joe, 2026-10-11): (a) conditional Modified operations against every canonical node that tells the agent to run recall or learn.
- **Applied in:** the operations of [793a7 ambient-guidance, route and please (the working agent no longer runs recall or learn while the Integration is installed)](793a_product-spec.md#793a7-modified-ambient-guidance-route-and-please-the-working-agent-no-longer-runs-recall-or-learn-while-the-integration-is-installed), one for each of [793a7a Recall fires at four decision moments](793a_product-spec.md#793a7a-modified-recall-fires-at-four-decision-moments) to [793a7p A lesson about to be captured asks what should have surfaced it](793a_product-spec.md#793a7p-modified-a-lesson-about-to-be-captured-asks-what-should-have-surfaced-it), and the consequent changes to [793a6b Every dispatch is told to report lessons](793a_product-spec.md#793a6b-modified-every-dispatch-is-told-to-report-lessons), [793a6f The closing learn gets the whole list](793a_product-spec.md#793a6f-modified-the-closing-learn-gets-the-whole-list), [793a6h Every dispatch hands over an exact spec](793a_product-spec.md#793a6h-modified-every-dispatch-hands-over-an-exact-spec) and [793a6i The closing learn gets the lessons and the audit](793a_product-spec.md#793a6i-modified-the-closing-learn-gets-the-lessons-and-the-audit). The candidates examined and left unchanged are in [793b101 Candidates examined, changed and left](793b_decision-log.md#793b101-candidates-examined-changed-and-left).
- **Applied in commit:** 857cee6a.
- **Context:** [793a2k The working agent does not run recall or learn](793a_product-spec.md#793a2k-the-working-agent-does-not-run-recall-or-learn) says the working agent is not asked to run recall or learn while the Integration is installed. The Modified operations of [793a6 lessons-contract (the LESSONS line is replaced while the Integration is installed)](793a_product-spec.md#793a6-modified-lessons-contract-the-lessons-line-is-replaced-while-the-integration-is-installed) cover only the nodes that name the LESSONS line. Other canonical nodes tell the agent to run recall or learn: [ambient-guidance](../../specs/product/ambient-guidance.md)'s [792a19a Recall fires at four decision moments](../../specs/product/ambient-guidance.md#792a19a-recall-fires-at-four-decision-moments), [792a19b Recall escalates from glance to deep](../../specs/product/ambient-guidance.md#792a19b-recall-escalates-from-glance-to-deep), [792a19c Learn fires at the correction moment](../../specs/product/ambient-guidance.md#792a19c-learn-fires-at-the-correction-moment) and [792a19d The capture is learn's fast path](../../specs/product/ambient-guidance.md#792a19d-the-capture-is-learns-fast-path), and the shim's [792a19j The first action on every request is an engram query](../../specs/product/ambient-guidance.md#792a19j-the-first-action-on-every-request-is-an-engram-query) and [792a19t The query runs again at four moments mid-task](../../specs/product/ambient-guidance.md#792a19t-the-query-runs-again-at-four-moments-mid-task); [route](../../specs/product/route.md)'s [792a17h Every subagent recalls first, and large units are decomposed](../../specs/product/route.md#792a17h-every-subagent-recalls-first-and-large-units-are-decomposed); and [please](../../specs/product/please.md)'s seven-step workflow, which opens with learn and orients with recall ([792a18d The workflow opens with learn and orients with recall](../../specs/product/please.md#792a18d-the-workflow-opens-with-learn-and-orients-with-recall)). Without a change, a harness that imports that guidance still tells the agent to do what the System now does.
- **Options:**
  - (a) Modified operations, conditional on the Integration, against each of those nodes, as for the LESSONS line.
  - (b) Do it for the LESSONS nodes now, as written, and decide the rest separately, because it reaches please's seven-step workflow, the shim and route and is a larger change than this one.
  - (c) Change nothing canonical; the agent's own redundant runs are harmless under [793a2e No re-injection](793a_product-spec.md#793a2e-no-re-injection).
- **Recommendation:** (b). The LESSONS operations were the scope of the restructure; the other nodes need their own review, and [793a2k The working agent does not run recall or learn](793a_product-spec.md#793a2k-the-working-agent-does-not-run-recall-or-learn) states the observable requirement meanwhile.
- **Blocks:** [793a2k The working agent does not run recall or learn](793a_product-spec.md#793a2k-the-working-agent-does-not-run-recall-or-learn) when promoted (the canonical nodes would contradict it), and the nodes listed.

## 793c32 Does the System keep a rule against concurrent duplicate captures?

- **Status:** answered (Joe, 2026-10-11): (a) keep the concurrent-duplicate rule.
- **Applied in:** [793a3m Lessons from subagent sessions are captured directly](793a_product-spec.md#793a3m-lessons-from-subagent-sessions-are-captured-directly) and [793a3m1 UAT Scenarios](793a_product-spec.md#793a3m1-uat-scenarios) as written; nothing further changed.
- **Applied in commit:** 63b15d9b.
- **Context:** The draft's rule (formerly CAP-3) said the System writes one note when two subagents capture the same lesson concurrently. With learn delegated, learn's own handling of an existing equivalent note applies (D5 dropped), but two System sessions can race, which no skill specifies. [793a3m Lessons from subagent sessions are captured directly](793a_product-spec.md#793a3m-lessons-from-subagent-sessions-are-captured-directly) keeps the concurrent case, with [793a3m1d Two subagents contain the same lesson](793a_product-spec.md#793a3m1-uat-scenarios).
- **Options:**
  - (a) Keep it in [793a3m Lessons from subagent sessions are captured directly](793a_product-spec.md#793a3m-lessons-from-subagent-sessions-are-captured-directly), with the scenario.
  - (b) Drop it; learn's handling governs and concurrency is a technical concern.
- **Recommendation:** (a). The race exists only because the System runs captures in parallel, and a duplicate note is user-visible.
- **Blocks:** [793a3m Lessons from subagent sessions are captured directly](793a_product-spec.md#793a3m-lessons-from-subagent-sessions-are-captured-directly) and [793a3m1d Two subagents contain the same lesson](793a_product-spec.md#793a3m1-uat-scenarios).

## 793c33 Do the requirements outside the D1-D5 and N1-N8 lists stay?

- **Status:** answered (Joe, 2026-10-11): (a) as applied.
- **Applied in:** the kept requirements [793a3d Provenance](793a_product-spec.md#793a3d-provenance), [793a4d Sensitive content](793a_product-spec.md#793a4d-sensitive-content), [793a5c Environments without the capability](793a_product-spec.md#793a5c-environments-without-the-capability) and [793a5f Effect on outcomes](793a_product-spec.md#793a5f-effect-on-outcomes) as written; nothing further changed.
- **Applied in commit:** 63b15d9b.
- **Context:** [793c30 Which earlier decisions stay as deliberate differences from the skills?](#793c30-which-earlier-decisions-stay-as-deliberate-differences-from-the-skills) did not mention these. Applied meanwhile, as in [793b92 Requirements that re-specified skill behavior are retired](793b_decision-log.md#793b92-requirements-that-re-specified-skill-behavior-are-retired) and [793b96 Other interpretations made in the restructure](793b_decision-log.md#793b96-other-interpretations-made-in-the-restructure): kept are [793a3d Provenance](793a_product-spec.md#793a3d-provenance) (the graders and the log need to know which notes are the System's), [793a4d Sensitive content](793a_product-spec.md#793a4d-sensitive-content) (no canonical skill has a secrets rule), [793a5c Environments without the capability](793a_product-spec.md#793a5c-environments-without-the-capability) (it is what makes the operations of [793a6 lessons-contract (the LESSONS line is replaced while the Integration is installed)](793a_product-spec.md#793a6-modified-lessons-contract-the-lessons-line-is-replaced-while-the-integration-is-installed) conditional) and [793a5f Effect on outcomes](793a_product-spec.md#793a5f-effect-on-outcomes) (a SHOULD about the whole System). Retired as skill behavior are confirmed lessons only, no duplicates, direct writes to the local vault, independence from agent compliance and coexistence. The earlier decisions about the deep moment's writes ([793c16 At a deep moment, does the System's coverage write activate the note it covers?](#793c16-at-a-deep-moment-does-the-systems-coverage-write-activate-the-note-it-covers), [793c23 Is a note amended as near injected?](#793c23-is-a-note-amended-as-near-injected), [793c24 Does the covering note's injection add anything to the depth's output?](#793c24-does-the-covering-notes-injection-add-anything-to-the-depths-output)) are treated as retired by delegation. The sweep difference ([793a2l The chunk index is swept after the turn](793a_product-spec.md#793a2l-the-chunk-index-is-swept-after-the-turn)) means the first moment of a session may query a staler index than the skill would.
- **Options:**
  - (a) As applied.
  - (b) Also keep some of the retired ones, naming which.
  - (c) Drop some of the kept ones, naming which.
- **Recommendation:** (a).
- **Blocks:** the requirements named above.

## 793c34 What stands in for learn's closing learn?

- **Status:** answered (Joe, 2026-10-11): (a) after each completed turn.
- **Applied in:** [793a3a Learn is performed for the working agent](793a_product-spec.md#793a3a-learn-is-performed-for-the-working-agent) as written (the capture after each completed turn); nothing further changed. [793a7l The seven steps are tracked, ordered and non-waivable](793a_product-spec.md#793a7l-modified-the-seven-steps-are-tracked-ordered-and-non-waivable) and [793a7o The closing learn is preceded by a lessons audit](793a_product-spec.md#793a7o-modified-the-closing-learn-is-preceded-by-a-lessons-audit) record that please runs no closing learn while the Integration is installed.
- **Applied in commit:** 63b15d9b.
- **Context:** [learn](../../specs/product/learn.md) runs at the end of work and at the moments a lesson is confirmed. The System has no end of work, only turns and sessions. [793a3a Learn is performed for the working agent](793a_product-spec.md#793a3a-learn-is-performed-for-the-working-agent) runs learn at the moments it fires and, in place of the closing learn, after each completed turn.
- **Options:**
  - (a) After each completed turn.
  - (b) At the end of the session.
  - (c) At the end of each unit of work the agent completes.
- **Recommendation:** (a). It matches the turn-end capture of [793c27 Does the System sweep the chunk index (792a14b)?](#793c27-does-the-system-sweep-the-chunk-index-792a14b) and covers a subagent session whose end the System may not see.
- **Blocks:** [793a3a Learn is performed for the working agent](793a_product-spec.md#793a3a-learn-is-performed-for-the-working-agent).

## 793c35 How is the delay comparison in 793a5g1a made given timing noise?

- **Status:** answered (Joe, 2026-10-11): (a) one logged time against one inline time; repeat counts are test design.
- **Applied in:** [793a5g No added delay](793a_product-spec.md#793a5g-no-added-delay) and [793a5g1 UAT Scenarios](793a_product-spec.md#793a5g1-uat-scenarios) as written; nothing further changed.
- **Applied in commit:** 63b15d9b.
- **Context:** [793a5g1a Recall adds no more delay than the agent's own recall](793a_product-spec.md#793a5g1-uat-scenarios) compares one logged latency with one inline timing. Both vary between runs, which was the point of the retired [793c28 How is 793a5g's latency comparison measured?](#793c28-how-is-793a5gs-latency-comparison-measured).
- **Options:**
  - (a) As written: one logged latency against one inline timing, leaving noise handling to the technical test design.
  - (b) Median over N repeats per moment kind.
- **Recommendation:** (a). Joe asked for one simple requirement, and the repeat count is a test-design choice.
- **Blocks:** [793a5g1a Recall adds no more delay than the agent's own recall](793a_product-spec.md#793a5g1-uat-scenarios).

## 793c36 Does the shim's follow-frame apply to notes the System delivers?

- **Status:** answered (Joe, 2026-10-11): (a) leave [792a19l Each returned kind is treated for what it is](../../specs/product/ambient-guidance.md#792a19l-each-returned-kind-is-treated-for-what-it-is)–q unchanged; they govern treatment of an item whoever fetched it.
- **Applied in:** nothing changed: [792a19l Each returned kind is treated for what it is](../../specs/product/ambient-guidance.md#792a19l-each-returned-kind-is-treated-for-what-it-is) to [792a19q Linked and truncated runbooks get the same frame](../../specs/product/ambient-guidance.md#792a19q-linked-and-truncated-runbooks-get-the-same-frame) stay unmodified, and no operation of [793a7 ambient-guidance, route and please (the working agent no longer runs recall or learn while the Integration is installed)](793a_product-spec.md#793a7-modified-ambient-guidance-route-and-please-the-working-agent-no-longer-runs-recall-or-learn-while-the-integration-is-installed) targets them. Recorded in [793b106 793c36: the shim follow-frame nodes stay unmodified](793b_decision-log.md#793b106-793c36-the-shim-follow-frame-nodes-stay-unmodified).
- **Applied in commit:** 185fd4af.
- **Context:** [792a19l Each returned kind is treated for what it is](../../specs/product/ambient-guidance.md#792a19l-each-returned-kind-is-treated-for-what-it-is) to [792a19q Linked and truncated runbooks get the same frame](../../specs/product/ambient-guidance.md#792a19q-linked-and-truncated-runbooks-get-the-same-frame) tell the agent how to treat what "a query returns" or what "a returned note" is: facts, feedback, runbooks (announce, restate the steps as the plan, read the red flags, treat `done_when` as the bar, stop and ask) and chunks. While the Integration is installed the agent runs no query ([793a7e The first action on every request is an engram query](793a_product-spec.md#793a7e-modified-the-first-action-on-every-request-is-an-engram-query)); the System delivers the notes ([793a2d Attributable presentation](793a_product-spec.md#793a2d-attributable-presentation) says a runbook is a legitimate injection). Read literally, the frame would then never trigger. [792a19r One floor beneath every runbook](../../specs/product/ambient-guidance.md#792a19r-one-floor-beneath-every-runbook) and [792a19s A signal of unrequested work is a standing instruction](../../specs/product/ambient-guidance.md#792a19s-a-signal-of-unrequested-work-is-a-standing-instruction) are generic and not affected in the same way.
- **Options:**
  - (a) Leave the nodes unchanged. They govern how the agent treats an item it has, whoever retrieved it; the shim and guidance text says "returned or delivered" when it is rewritten at implementation, which needs the writing-skills cycle.
  - (b) Add Modified operations to [793a7 ambient-guidance, route and please (the working agent no longer runs recall or learn while the Integration is installed)](793a_product-spec.md#793a7-modified-ambient-guidance-route-and-please-the-working-agent-no-longer-runs-recall-or-learn-while-the-integration-is-installed) for [792a19l Each returned kind is treated for what it is](../../specs/product/ambient-guidance.md#792a19l-each-returned-kind-is-treated-for-what-it-is) to [792a19q Linked and truncated runbooks get the same frame](../../specs/product/ambient-guidance.md#792a19q-linked-and-truncated-runbooks-get-the-same-frame), widening each to "a query returns or the System delivers".
- **Recommendation:** (a). The behavior asked of the agent is the same, and the spec nodes describe the treatment, not the retrieval. (b) is the safer reading if the nodes are to be read literally.
- **Blocks:** [792a19l Each returned kind is treated for what it is](../../specs/product/ambient-guidance.md#792a19l-each-returned-kind-is-treated-for-what-it-is) to [792a19q Linked and truncated runbooks get the same frame](../../specs/product/ambient-guidance.md#792a19q-linked-and-truncated-runbooks-get-the-same-frame) when promoted, if (b) is chosen.

## 793c37 What is the shape of please's workflow while the Integration is installed?

- **Status:** answered (Joe, 2026-10-11): (c) the System runs please's lessons audit and the which-artifact-should-have-surfaced-it question and reword from the transcript, as part of capture; please drops them.
- **Applied in:** [793a3o Please's lessons audit and surfacing question are performed by the System](793a_product-spec.md#793a3o-pleases-lessons-audit-and-surfacing-question-are-performed-by-the-system) (new); the operations [793a7l The seven steps are tracked, ordered and non-waivable](793a_product-spec.md#793a7l-modified-the-seven-steps-are-tracked-ordered-and-non-waivable), [793a7o The closing learn is preceded by a lessons audit](793a_product-spec.md#793a7o-modified-the-closing-learn-is-preceded-by-a-lessons-audit) and [793a7p A lesson about to be captured asks what should have surfaced it](793a_product-spec.md#793a7p-modified-a-lesson-about-to-be-captured-asks-what-should-have-surfaced-it), and the consequent edit to [793a6i The closing learn gets the lessons and the audit](793a_product-spec.md#793a6i-modified-the-closing-learn-gets-the-lessons-and-the-audit). Interpretations in [793b107 793c37: please drops its learn steps, its audit and its surfacing question](793b_decision-log.md#793b107-793c37-please-drops-its-learn-steps-its-audit-and-its-surfacing-question) and [793b108 Interpretations made in applying 793c37](793b_decision-log.md#793b108-interpretations-made-in-applying-793c37); the one open point is [793c40 Does the user still see please's audit list?](#793c40-does-the-user-still-see-pleases-audit-list).
- **Applied in commit:** 185fd4af.
- **Context:** [792a18c The seven steps are tracked, ordered and non-waivable](../../specs/product/please.md#792a18c-the-seven-steps-are-tracked-ordered-and-non-waivable) fixes please at seven steps, and steps 1 (the opening learn) and 7 (the closing learn) are learn and nothing else; [792a18d The workflow opens with learn and orients with recall](../../specs/product/please.md#792a18d-the-workflow-opens-with-learn-and-orients-with-recall) makes the orienting recall literal; [792a18p The closing learn is preceded by a lessons audit](../../specs/product/please.md#792a18p-the-closing-learn-is-preceded-by-a-lessons-audit) audits the cycle's failures to hand them to the closing learn; [792a18q A lesson about to be captured asks what should have surfaced it](../../specs/product/please.md#792a18q-a-lesson-about-to-be-captured-asks-what-should-have-surfaced-it) asks, for each lesson about to be captured, what should have surfaced it. With the working agent no longer running learn or recall, each of those has to change ([793a7l The seven steps are tracked, ordered and non-waivable](793a_product-spec.md#793a7l-modified-the-seven-steps-are-tracked-ordered-and-non-waivable), [793a7m The workflow opens with learn and orients with recall](793a_product-spec.md#793a7m-modified-the-workflow-opens-with-learn-and-orients-with-recall), [793a7o The closing learn is preceded by a lessons audit](793a_product-spec.md#793a7o-modified-the-closing-learn-is-preceded-by-a-lessons-audit), [793a7p A lesson about to be captured asks what should have surfaced it](793a_product-spec.md#793a7p-modified-a-lesson-about-to-be-captured-asks-what-should-have-surfaced-it), [793a6i The closing learn gets the lessons and the audit](793a_product-spec.md#793a6i-modified-the-closing-learn-gets-the-lessons-and-the-audit)).
- **Options:**
  - (a) As applied: five tracked steps (2 to 6, keeping their numbers); no opening or closing learn; the audit stays and reports unmapped items in the closing report instead of handing them to a learn; the surfacing question, the covering-note search and the reword are dropped because please captures nothing.
  - (b) Keep step 7 as a report-only step that holds the audit, so that the seven-step shape stays.
  - (c) Keep the surfacing question and the reword in please, as advice to the System's capture, which needs a new interface.
- **Recommendation:** (a). It removes learn and recall from please without adding a step or an interface; (b) renames the same thing, and (c) is new scope. The cost of (a) is that the question of which artifact should have surfaced a lesson is no longer asked anywhere.
- **Blocks:** [792a18c The seven steps are tracked, ordered and non-waivable](../../specs/product/please.md#792a18c-the-seven-steps-are-tracked-ordered-and-non-waivable), [792a18d The workflow opens with learn and orients with recall](../../specs/product/please.md#792a18d-the-workflow-opens-with-learn-and-orients-with-recall), [792a18p The closing learn is preceded by a lessons audit](../../specs/product/please.md#792a18p-the-closing-learn-is-preceded-by-a-lessons-audit), [792a18q A lesson about to be captured asks what should have surfaced it](../../specs/product/please.md#792a18q-a-lesson-about-to-be-captured-asks-what-should-have-surfaced-it) and [792a18r The closing learn gets the lessons and the audit](../../specs/product/please.md#792a18r-the-closing-learn-gets-the-lessons-and-the-audit) when promoted.

## 793c38 Do the orchestrator's route-evidence writes stay?

- **Status:** answered (Joe, 2026-10-11): (b) the System writes route's evidence notes and aggregates, reading tier, model and outcome from route's report row in the transcript ([792a17i Each dispatch is recorded in a mini-report row](../../specs/product/route.md#792a17i-each-dispatch-is-recorded-in-a-mini-report-row)); the orchestrator writes none.
- **Applied in:** [793a3p Route's evidence writes are performed by the System](793a_product-spec.md#793a3p-routes-evidence-writes-are-performed-by-the-system) (new); the operations [793a7q Each dispatch is also written to the vault as a tagged evidence note](793a_product-spec.md#793a7q-modified-each-dispatch-is-also-written-to-the-vault-as-a-tagged-evidence-note) and [793a7r Each work-kind keeps one aggregate note](793a_product-spec.md#793a7r-modified-each-work-kind-keeps-one-aggregate-note). [793a2k The working agent does not run recall or learn](793a_product-spec.md#793a2k-the-working-agent-does-not-run-recall-or-learn) keeps its wording ("write any memory"), which now holds. Interpretations in [793b109 793c38: the System writes route's evidence notes](793b_decision-log.md#793b109-793c38-the-system-writes-routes-evidence-notes) and [793b110 A report row must be in the transcript](793b_decision-log.md#793b110-a-report-row-must-be-in-the-transcript).
- **Applied in commit:** 185fd4af.
- **Context:** [793a2k The working agent does not run recall or learn](793a_product-spec.md#793a2k-the-working-agent-does-not-run-recall-or-learn) says the System does not ask the working agent to "evaluate, approve or write any memory". Route's evidence loop has the orchestrator hand each dispatch's tagged evidence note to write-memory ([792a17k Each dispatch is also written to the vault as a tagged evidence note](../../specs/product/route.md#792a17k-each-dispatch-is-also-written-to-the-vault-as-a-tagged-evidence-note)) and amend or create the work-kind aggregate itself with `engram learn` and a lookup by `engram query` ([792a17l Each work-kind keeps one aggregate note](../../specs/product/route.md#792a17l-each-work-kind-keeps-one-aggregate-note)). Those notes are measurements built from what only the orchestrator knows (tier, model, outcome, duration, cost), not lessons the System can find in a transcript; they are not recall's lookup, and not learn's judgment. [792a17o The loop improves the rubric](../../specs/product/route.md#792a17o-the-loop-improves-the-rubric), which crystallizes a finding with learn, is changed ([793a7k The loop improves the rubric](793a_product-spec.md#793a7k-modified-the-loop-improves-the-rubric)).
- **Options:**
  - (a) Leave [792a17k Each dispatch is also written to the vault as a tagged evidence note](../../specs/product/route.md#792a17k-each-dispatch-is-also-written-to-the-vault-as-a-tagged-evidence-note) and [792a17l Each work-kind keeps one aggregate note](../../specs/product/route.md#792a17l-each-work-kind-keeps-one-aggregate-note) unchanged, and later narrow [793a2k The working agent does not run recall or learn](793a_product-spec.md#793a2k-the-working-agent-does-not-run-recall-or-learn) from "any memory" to "memory the System writes for recall and learn".
  - (b) Have the System write the evidence notes, which needs it to see tier, model and outcome, and is a new capability.
  - (c) Modify [792a17k Each dispatch is also written to the vault as a tagged evidence note](../../specs/product/route.md#792a17k-each-dispatch-is-also-written-to-the-vault-as-a-tagged-evidence-note) and [792a17l Each work-kind keeps one aggregate note](../../specs/product/route.md#792a17l-each-work-kind-keeps-one-aggregate-note) so the orchestrator writes none while the Integration is installed, which ends the tier-evidence loop.
- **Recommendation:** (a). The evidence loop is route's own record, shipped and measured, and the System has no way to know its contents. The wording of [793a2k The working agent does not run recall or learn](793a_product-spec.md#793a2k-the-working-agent-does-not-run-recall-or-learn) is the part that is too wide.
- **Blocks:** [793a2k The working agent does not run recall or learn](793a_product-spec.md#793a2k-the-working-agent-does-not-run-recall-or-learn), [792a17k Each dispatch is also written to the vault as a tagged evidence note](../../specs/product/route.md#792a17k-each-dispatch-is-also-written-to-the-vault-as-a-tagged-evidence-note) and [792a17l Each work-kind keeps one aggregate note](../../specs/product/route.md#792a17l-each-work-kind-keeps-one-aggregate-note) when promoted.

## 793c39 Does tier evidence reach the orchestrator before it routes?

- **Status:** answered (Joe, 2026-10-11): (b) the orchestrator's decision to dispatch is a recall moment; the System delivers the work-kind's tier evidence before routing.
- **Applied in:** [793a2c Moments are noticed without the agent](793a_product-spec.md#793a2c-moments-are-noticed-without-the-agent) (the dispatch-decision moment) and [793a2c1d Tier evidence is delivered before a dispatch is routed](793a_product-spec.md#793a2c1-uat-scenarios); the operation [793a7i The starting tier comes from recalled evidence, otherwise the cheapest tier](793a_product-spec.md#793a7i-modified-the-starting-tier-comes-from-recalled-evidence-otherwise-the-cheapest-tier). Interpretations in [793b111 793c39: the decision to dispatch is a recall moment](793b_decision-log.md#793b111-793c39-the-decision-to-dispatch-is-a-recall-moment) and [793b112 Interpretations made in applying 793c39](793b_decision-log.md#793b112-interpretations-made-in-applying-793c39).
- **Applied in commit:** 185fd4af.
- **Context:** [792a17b The starting tier comes from recalled evidence, otherwise the cheapest tier](../../specs/product/route.md#792a17b-the-starting-tier-comes-from-recalled-evidence-otherwise-the-cheapest-tier) has the orchestrator recall tier-performance evidence before it dispatches. While the Integration is installed it does not run recall ([793a7i The starting tier comes from recalled evidence, otherwise the cheapest tier](793a_product-spec.md#793a7i-modified-the-starting-tier-comes-from-recalled-evidence-otherwise-the-cheapest-tier)), so it can use only the evidence the System has already put in its context at a prompt or at one of the four moments. A work-kind's aggregate is delivered only if it happens to match one of those. When it is not delivered the unit starts at the cheapest tier, the safe default, and an evidenced escalation is missed.
- **Options:**
  - (a) As applied: use the evidence already delivered; absent it, the cheapest tier. Measure whether routing regresses.
  - (b) Make the decision to dispatch a recall moment for the orchestrator, keyed to the work-kind, so the aggregate is delivered before routing.
  - (c) Let the orchestrator run the one lookup of the aggregate itself, as [792a17l Each work-kind keeps one aggregate note](../../specs/product/route.md#792a17l-each-work-kind-keeps-one-aggregate-note) already does.
- **Recommendation:** (a). It adds nothing to the System's moments, the default errs cheap, and [793a5f Effect on outcomes](793a_product-spec.md#793a5f-effect-on-outcomes) measures the effect on outcomes; (b) is the follow-up if the measurement shows missed escalations.
- **Blocks:** [793a7i The starting tier comes from recalled evidence, otherwise the cheapest tier](793a_product-spec.md#793a7i-modified-the-starting-tier-comes-from-recalled-evidence-otherwise-the-cheapest-tier).

## 793c40 Does the user still see please's audit list?

- **Status:** answered (Joe, 2026-10-11): (d), Joe's own option. At every opportunity the System prints the counts of relevant memory events directly to the user (surfaced x, used y, added z, and so on), through the harness's own display surface ("with mods, the system can just print stuff to the user"). None of it enters the working agent's context. Applied as interpreted in [793b114 A memory-activity summary is shown to the user](793b_decision-log.md#793b114-a-memory-activity-summary-is-shown-to-the-user).
- **Applied in:** [793a5h A memory-activity summary is shown to the user](793a_product-spec.md#793a5h-a-memory-activity-summary-is-shown-to-the-user); the operations [793a3o Please's lessons audit and surfacing question are performed by the System](793a_product-spec.md#793a3o-pleases-lessons-audit-and-surfacing-question-are-performed-by-the-system) and [793a7o The closing learn is preceded by a lessons audit](793a_product-spec.md#793a7o-modified-the-closing-learn-is-preceded-by-a-lessons-audit) reference it.
- **Applied in commit:** 9295802e.
- **Context:** [792a18p The closing learn is preceded by a lessons audit](../../specs/product/please.md#792a18p-the-closing-learn-is-preceded-by-a-lessons-audit) has please show the audit list (item to note, or "no lesson: why") in the cycle's closing report to the user. Joe's answer to [793c37 What is the shape of please's workflow while the Integration is installed?](#793c37-what-is-the-shape-of-pleases-workflow-while-the-integration-is-installed) moves the audit to the System ([793a3o Please's lessons audit and surfacing question are performed by the System](793a_product-spec.md#793a3o-pleases-lessons-audit-and-surfacing-question-are-performed-by-the-system)), which records its outcome in the vault and the log. Please's closing report then carries no list, so the user sees the audit only by reading the log or the notes.
- **Options:**
  - (a) The list is in the vault and the log only, as applied.
  - (b) The System also presents the list to the user at the end of the run, which needs a way for the System to address the user directly.
  - (c) Please reads the System's log at the end of the run and includes the list in its closing report, which asks the working agent to handle memory bookkeeping again, against [793a2k The working agent does not run recall or learn](793a_product-spec.md#793a2k-the-working-agent-does-not-run-recall-or-learn).
- **Recommendation:** (a). It is what the answer asked for (the outcome in the vault and the log, the orchestrator doing neither), and (b) and (c) each add an interface.
- **Blocks:** [793a3o Please's lessons audit and surfacing question are performed by the System](793a_product-spec.md#793a3o-pleases-lessons-audit-and-surfacing-question-are-performed-by-the-system) and [793a7o The closing learn is preceded by a lessons audit](793a_product-spec.md#793a7o-modified-the-closing-learn-is-preceded-by-a-lessons-audit), on whether the user must be shown the list.

## 793c41 At a recall moment, what does the System put in front of the working agent?

- **Status:** answered (Joe, 2026-10-11), Joe's own option: recall's planning part is what the agent does with what was recalled. The System returns the results with an instruction to work them into the agent's plan, or to replan entirely. The agent does recall's plan, synthesis and reconciliation (792a14a, 792a14m, 792a14n, 792a14o) itself, in its own reasoning. Joe: "The watcher won't have all of the same context as the main agent. Its job is to handle memory, not play puppet master."
- **Context:** Joe's intent ([793c29 Is the System a delegate for the existing memory skills?](#793c29-is-the-system-a-delegate-for-the-existing-memory-skills)) is the same behavior, offloaded from the working agent's context. When the agent runs recall, its context gets the query results plus recall's plan ([792a14a The plan is stated before anything is retrieved](../../specs/product/recall.md#792a14a-the-plan-is-stated-before-anything-is-retrieved)), its closing synthesis ([792a14m The closing synthesis says whether memory changed the plan](../../specs/product/recall.md#792a14m-the-closing-synthesis-says-whether-memory-changed-the-plan)) and its reconciliation of a recommendation ([792a14n A recommendation is reconciled against the asked task](../../specs/product/recall.md#792a14n-a-recommendation-is-reconciled-against-the-asked-task), [792a14o An emergent recommendation is checked against the vault](../../specs/product/recall.md#792a14o-an-emergent-recommendation-is-checked-against-the-vault)). [792a14c Recall has two depths](../../specs/product/recall.md#792a14c-recall-has-two-depths) keeps the synthesis even at glance, and [792a14m The closing synthesis says whether memory changed the plan](../../specs/product/recall.md#792a14m-the-closing-synthesis-says-whether-memory-changed-the-plan) forbids raw scores and paths in it. [793a2a Recall is performed for the working agent](793a_product-spec.md#793a2a-recall-is-performed-for-the-working-agent), [793a2d Attributable presentation](793a_product-spec.md#793a2d-attributable-presentation) and [793a2e No re-injection](793a_product-spec.md#793a2e-no-re-injection) instead assume that the System injects raw items, each named by its source.
- **Options:**
  - (a) The query results only; recall's plan, synthesis and reconciliation are not performed at System moments. This is a deliberate difference from the skill.
  - (b) Everything recall would have produced in the agent's context: the results plus recall's closing synthesis, with the System inferring the plan from the transcript.
  - (c) The synthesis only.
- **Recommendation:** (b). It is the only option with no exception to the delegation.
- **Blocks:** [793a2a Recall is performed for the working agent](793a_product-spec.md#793a2a-recall-is-performed-for-the-working-agent), [793a2d Attributable presentation](793a_product-spec.md#793a2d-attributable-presentation), [793a2e No re-injection](793a_product-spec.md#793a2e-no-re-injection) and their scenarios. Raised by the fifth review ([793b115 Fifth review: findings held for 793c41 to 793c44](793b_decision-log.md#793b115-fifth-review-findings-held-for-793c41-to-793c44)).

## 793c42 What runs at every prompt (D1)?

- **Status:** answered (Joe, 2026-10-11): (a) the shim's lookup at every prompt.
- **Context:** [793a2i Recall for every prompt regardless of source](793a_product-spec.md#793a2i-recall-for-every-prompt-regardless-of-source) has the System perform recall at every prompt, a difference from the guidance ([793c30 Which earlier decisions stay as deliberate differences from the skills?](#793c30-which-earlier-decisions-stay-as-deliberate-differences-from-the-skills), D1). The skills define no per-prompt action, but the shim's per-prompt action is one `engram query` with its phrase shapes ([792a19j The first action on every request is an engram query](../../specs/product/ambient-guidance.md#792a19j-the-first-action-on-every-request-is-an-engram-query), [792a19k The second phrase names a kind of work-handling decision](../../specs/product/ambient-guidance.md#792a19k-the-second-phrase-names-a-kind-of-work-handling-decision)), the lookup [793a7e The first action on every request is an engram query](793a_product-spec.md#793a7e-modified-the-first-action-on-every-request-is-an-engram-query) takes over. Recall's canonical default depth is deep ([792a14c Recall has two depths](../../specs/product/recall.md#792a14c-recall-has-two-depths)), which writes to the vault.
- **Options:**
  - (a) The shim's lookup, delegated exactly from [792a19j The first action on every request is an engram query](../../specs/product/ambient-guidance.md#792a19j-the-first-action-on-every-request-is-an-engram-query): one `engram query` with its phrase shapes, results delivered.
  - (b) A glance recall.
  - (c) Recall at its canonical default depth, which is deep, with vault writes on every prompt.
- **Recommendation:** (a). Today the per-prompt action is the shim's query, so delegating it changes nothing but who runs it.
- **Blocks:** [793a2i Recall for every prompt regardless of source](793a_product-spec.md#793a2i-recall-for-every-prompt-regardless-of-source), [793a2a Recall is performed for the working agent](793a_product-spec.md#793a2a-recall-is-performed-for-the-working-agent) and [793a7e The first action on every request is an engram query](793a_product-spec.md#793a7e-modified-the-first-action-on-every-request-is-an-engram-query). Raised by the fifth review ([793b115 Fifth review: findings held for 793c41 to 793c44](793b_decision-log.md#793b115-fifth-review-findings-held-for-793c41-to-793c44)).

## 793c43 Who handles pending offers and the curate step?

- **Status:** answered (Joe, 2026-10-11): (a) the System runs curate after its recall, and the hint is removed from what the agent sees.
- **Context:** [792a14l Only the notes that were used are activated](../../specs/product/recall.md#792a14l-only-the-notes-that-were-used-are-activated) has recall leave pulled copies of parent notes pending, finish the request and run the curate skill as a separate step; [792a22o Pending offers are surfaced at three points](../../specs/product/curate.md#792a22o-pending-offers-are-surfaced-at-three-points) surfaces pending offers in the query payload; [792a19s A signal of unrequested work is a standing instruction](../../specs/product/ambient-guidance.md#792a19s-a-signal-of-unrequested-work-is-a-standing-instruction) tells the agent to obey that hint after finishing the request. With recall delegated, the agent runs neither recall nor curate, yet the hint would still reach it and ask for memory work.
- **Options:**
  - (a) The System runs curate after its recall, as recall defines it. [793a4c Memory purview only](793a_product-spec.md#793a4c-memory-purview-only) lists curation's effects, and the hint is removed from what the agent sees.
  - (b) The working agent still curates, with [793a2k The working agent does not run recall or learn](793a_product-spec.md#793a2k-the-working-agent-does-not-run-recall-or-learn) narrowed.
  - (c) Nobody curates during the session; the summary ([793a5h A memory-activity summary is shown to the user](793a_product-spec.md#793a5h-a-memory-activity-summary-is-shown-to-the-user)) counts pending offers.
- **Recommendation:** (a). It keeps recall's behavior and the agent's context free of the work.
- **Blocks:** [793a2k The working agent does not run recall or learn](793a_product-spec.md#793a2k-the-working-agent-does-not-run-recall-or-learn), [793a4c Memory purview only](793a_product-spec.md#793a4c-memory-purview-only), and the nodes [792a19s A signal of unrequested work is a standing instruction](../../specs/product/ambient-guidance.md#792a19s-a-signal-of-unrequested-work-is-a-standing-instruction), [792a22o Pending offers are surfaced at three points](../../specs/product/curate.md#792a22o-pending-offers-are-surfaced-at-three-points) and [792a14l Only the notes that were used are activated](../../specs/product/recall.md#792a14l-only-the-notes-that-were-used-are-activated). Raised by the fifth review ([793b115 Fifth review: findings held for 793c41 to 793c44](793b_decision-log.md#793b115-fifth-review-findings-held-for-793c41-to-793c44)).

## 793c44 Where do the delegated skills' messages to the user go?

- **Status:** answered (Joe, 2026-10-11): (b) all of these messages are shown to the user on the summary's display surface. Joe will open a new change if it's too much.
- **Context:** the skills address the user through the agent: recall's synthesis to the user ([792a14m The closing synthesis says whether memory changed the plan](../../specs/product/recall.md#792a14m-the-closing-synthesis-says-whether-memory-changed-the-plan)), learn's report that a write is saved or queued for the parent ([792a15j A write to a parent is reported as done](../../specs/product/learn.md#792a15j-a-write-to-a-parent-is-reported-as-done)), please's repository-change proposal ([792a18q A lesson about to be captured asks what should have surfaced it](../../specs/product/please.md#792a18q-a-lesson-about-to-be-captured-asks-what-should-have-surfaced-it)) and route's drowning report ([792a17n Drowning is reported, not patched](../../specs/product/route.md#792a17n-drowning-is-reported-not-patched)). With the System performing them, the agent is not the speaker, and [793a5h A memory-activity summary is shown to the user](793a_product-spec.md#793a5h-a-memory-activity-summary-is-shown-to-the-user) shows counts only.
- **Options:**
  - (a) Log only, plus counts in the [793a5h A memory-activity summary is shown to the user](793a_product-spec.md#793a5h-a-memory-activity-summary-is-shown-to-the-user) summary.
  - (b) Shown to the user on the display surface of [793a5h A memory-activity summary is shown to the user](793a_product-spec.md#793a5h-a-memory-activity-summary-is-shown-to-the-user).
  - (c) Injected for the agent to relay.
- **Recommendation:** (b) for items that ask the user to act (proposals, drowning reports) and (a) for routine reports. Applied meanwhile to the drowning report ([793a7u Drowning is reported, not patched](793a_product-spec.md#793a7u-modified-drowning-is-reported-not-patched)).
- **Blocks:** [793a5h A memory-activity summary is shown to the user](793a_product-spec.md#793a5h-a-memory-activity-summary-is-shown-to-the-user), [793a7p A lesson about to be captured asks what should have surfaced it](793a_product-spec.md#793a7p-modified-a-lesson-about-to-be-captured-asks-what-should-have-surfaced-it) and the route drowning operation ([793a7u Drowning is reported, not patched](793a_product-spec.md#793a7u-modified-drowning-is-reported-not-patched)). Raised by the fifth review ([793b115 Fifth review: findings held for 793c41 to 793c44](793b_decision-log.md#793b115-fifth-review-findings-held-for-793c41-to-793c44)).

## 793c45 Is the Integration installed for every harness, or only for Claude Code?

- **Status:** answered (Joe, 2026-10-11): "We need to keep the existing skills for non-Claude agents. The spec should specify that. Init and update should do different things on Claude harnesses than on others."
- **Applied in:** [793a4h Installed by engram update](793a_product-spec.md#793a4h-installed-by-engram-update), [793a4g Active whenever installed](793a_product-spec.md#793a4g-active-whenever-installed) and [793a5c Environments without the capability](793a_product-spec.md#793a5c-environments-without-the-capability), with the scenarios [793a4g1d One machine, a delegated session and a session that is not](793a_product-spec.md#793a4g1-uat-scenarios), [793a4h1d One update serves both harnesses differently](793a_product-spec.md#793a4h1-uat-scenarios) and [793a5c1b A Pi session on a machine whose Claude Code has the Integration](793a_product-spec.md#793a5c1-uat-scenarios); the Integration term in [793a1 Problem Statement](793a_product-spec.md#793a1-problem-statement); and the preambles of [793a6 lessons-contract (the LESSONS line is replaced while the Integration is installed)](793a_product-spec.md#793a6-modified-lessons-contract-the-lessons-line-is-replaced-while-the-integration-is-installed) and [793a7 ambient-guidance, route and please (the working agent no longer runs recall or learn while the Integration is installed)](793a_product-spec.md#793a7-modified-ambient-guidance-route-and-please-the-working-agent-no-longer-runs-recall-or-learn-while-the-integration-is-installed), which make the condition per harness and per agent session. Interpretations in [793b133 793c45: the Integration is installed for Claude Code only](793b_decision-log.md#793b133-793c45-the-integration-is-installed-for-claude-code-only) and [793b134 Interpretations made in applying 793c45, 793c46 and 793c47](793b_decision-log.md#793b134-interpretations-made-in-applying-793c45-793c46-and-793c47).
- **Applied in commit:** 89e6a3a0.
- **Context:** [793a4h Installed by engram update](793a_product-spec.md#793a4h-installed-by-engram-update) installed the Integration for each harness in which the System can run, and [793a5c Environments without the capability](793a_product-spec.md#793a5c-environments-without-the-capability) kept the instruction-based path only where it cannot run. Pi has no way to run the System today, but the text did not say that a harness the System could run in but that is not Claude Code keeps the skills.
- **Options:**
  - (a) The Integration is installed for Claude Code only. Every other harness keeps the existing skills, guidance, shim and LESSONS contract, and its working agent keeps running recall and learn. Joe's answer.
  - (b) The Integration is installed for every harness in which the System can run, as written before.
- **Recommendation:** (a), chosen by the owner.
- **Blocks:** [793a4h Installed by engram update](793a_product-spec.md#793a4h-installed-by-engram-update), [793a4g Active whenever installed](793a_product-spec.md#793a4g-active-whenever-installed), [793a5c Environments without the capability](793a_product-spec.md#793a5c-environments-without-the-capability) and the conditions of [793a6 lessons-contract (the LESSONS line is replaced while the Integration is installed)](793a_product-spec.md#793a6-modified-lessons-contract-the-lessons-line-is-replaced-while-the-integration-is-installed) and [793a7 ambient-guidance, route and please (the working agent no longer runs recall or learn while the Integration is installed)](793a_product-spec.md#793a7-modified-ambient-guidance-route-and-please-the-working-agent-no-longer-runs-recall-or-learn-while-the-integration-is-installed).

## 793c46 On Claude Code with the Integration installed, does engram update still deploy the recall and learn skills and guidance files?

- **Status:** answered (Joe, 2026-10-11): (b) on Claude Code with the Integration installed, `engram update` removes the recall and learn skills and their guidance; they are not deployed there at all. Other harnesses keep them, per [793c45 Is the Integration installed for every harness, or only for Claude Code?](#793c45-is-the-integration-installed-for-every-harness-or-only-for-claude-code).
- **Applied in:** [793a4h Installed by engram update](793a_product-spec.md#793a4h-installed-by-engram-update) and its scenario [793a4h1e Claude Code gets no recall or learn skills or guidance](793a_product-spec.md#793a4h1-uat-scenarios). Interpretations in [793b134 Interpretations made in applying 793c45, 793c46 and 793c47](793b_decision-log.md#793b134-interpretations-made-in-applying-793c45-793c46-and-793c47).
- **Applied in commit:** 89e6a3a0.
- **Context:** [793c45 Is the Integration installed for every harness, or only for Claude Code?](#793c45-is-the-integration-installed-for-every-harness-or-only-for-claude-code) makes the install differ between Claude Code and other harnesses. The Integration makes the agent's own recall and learn runs unnecessary, but the skills are also how the user invokes `/recall` and `/learn` explicitly, and the guidance imports and the shim are what tell the agent to run them.
- **Options:**
  - (a) Deploy the skills so the user can still invoke `/recall` or `/learn` explicitly. Do not activate the guidance imports and the shim that tell the agent to run them. (recommended)
  - (b) Remove them from Claude Code entirely.
  - (c) Deploy everything as today and rely on the conditional operations of [793a6 lessons-contract (the LESSONS line is replaced while the Integration is installed)](793a_product-spec.md#793a6-modified-lessons-contract-the-lessons-line-is-replaced-while-the-integration-is-installed) and [793a7 ambient-guidance, route and please (the working agent no longer runs recall or learn while the Integration is installed)](793a_product-spec.md#793a7-modified-ambient-guidance-route-and-please-the-working-agent-no-longer-runs-recall-or-learn-while-the-integration-is-installed).
- **Recommendation:** (a), which keeps explicit control for the user. The owner chose (b).
- **Blocks:** [793a4h Installed by engram update](793a_product-spec.md#793a4h-installed-by-engram-update) and [793a4h1e Claude Code gets no recall or learn skills or guidance](793a_product-spec.md#793a4h1-uat-scenarios).

## 793c47 Does "init" mean `engram init` or the first-time install by `engram update`?

- **Status:** answered (Joe, 2026-10-11): (b), in his words: "init should initialize everything, idempotently. That way updates can also just run it." `engram init` sets up the vault and each harness's integration, and `engram update` runs it.
- **Applied in:** [793a4h Installed by engram update](793a_product-spec.md#793a4h-installed-by-engram-update) and its scenarios [793a4h1f A first-time update runs init](793a_product-spec.md#793a4h1-uat-scenarios) and [793a4h1g Init sets up everything and is idempotent](793a_product-spec.md#793a4h1-uat-scenarios). Interpretations in [793b134 Interpretations made in applying 793c45, 793c46 and 793c47](793b_decision-log.md#793b134-interpretations-made-in-applying-793c45-793c46-and-793c47); the dependency on change #795 is [793b136 Change 793 depends on change 795 being widened](793b_decision-log.md#793b136-change-793-depends-on-change-795-being-widened) and its sequencing is [793c48 How is the widening of `engram init` (change #795) sequenced with this change?](#793c48-how-is-the-widening-of-engram-init-change-795-sequenced-with-this-change).
- **Applied in commit:** 89e6a3a0.
- **Context:** Joe said "Init and update should do different things on Claude harnesses than on others" ([793c45 Is the Integration installed for every harness, or only for Claude Code?](#793c45-is-the-integration-installed-for-every-harness-or-only-for-claude-code)). The planned `engram init` (change #795) creates a vault; `engram update` installs the harness integrations, first time or later.
- **Options:**
  - (a) The first-time install path of `engram update`. `engram init` only creates a vault and is harness-agnostic. (recommended)
  - (b) `engram init` should also install harness integrations.
- **Recommendation:** (a). The owner chose (b).
- **Blocks:** [793a4h Installed by engram update](793a_product-spec.md#793a4h-installed-by-engram-update).

## 793c48 How is the widening of `engram init` (change #795) sequenced with this change?

- **Status:** answered (Joe, 2026-10-11): (a) widen #795. Its issue is retitled and its scope extended (init initializes everything idempotently, and update runs it). #795's spec is widened as part of #795's own work, and #793 builds on it.
- **Context:** [793c47 Does "init" mean `engram init` or the first-time install by `engram update`?](#793c47-does-init-mean-engram-init-or-the-first-time-install-by-engram-update) makes `engram init` set up the vault and every harness's integration, idempotently, with `engram update` running it. Change #795 (`docs/changes/795_vault-init-command/`) scopes init to vault creation only, and says `engram update` with no vault skips its vault steps and never creates one. [793a4h Installed by engram update](793a_product-spec.md#793a4h-installed-by-engram-update) now depends on init having the wider scope, and #795's update behavior conflicts with update running init ([793b136 Change 793 depends on change 795 being widened](793b_decision-log.md#793b136-change-793-depends-on-change-795-being-widened)).
- **Options:**
  - (a) Widen #795's own spec to cover harness setup, with update running init, so this change builds on it. (recommended)
  - (b) This change specifies the harness part of init itself, and #795 stays vault-only.
  - (c) A new issue for "init sets up everything".
- **Recommendation:** (a). One change owns `engram init`, so the two specs cannot disagree.
- **Blocks:** [793a4h Installed by engram update](793a_product-spec.md#793a4h-installed-by-engram-update), and #795's scope and its statement that update never creates a vault. This change does not edit #795's files.
