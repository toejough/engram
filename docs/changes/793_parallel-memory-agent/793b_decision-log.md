# Parallel Memory Agent - Decision Log

These are choices made while relocating the draft requirements into this change (#793, node In-flight design relocation of
#792) and while applying Joe's review answers, recorded for Joe to validate. Questions that need Joe's input are in
[`793c_open-questions.md`](./793c_open-questions.md). Every entry names who decided it; entries by the author, the
orchestrator or a reviewer await the owner's validation.

## 793b1 One new capability, `automatic-memory`

- **Decided by:** Author (agent); awaiting owner validation.

- **Choice:** every requirement in [Parallel Memory Agent - Product Spec](./793a_product-spec.md#parallel-memory-agent---product-spec) is an Added operation against one new canonical file,
  `docs/specs/product/automatic-memory.md` (title `Automatic Memory - Product Spec`).
- **Rationale:** the feature is one user-facing capability, automatic recall and capture independent
  of the working agent. It reuses behavior of `recall`, `learn` and `ambient-guidance` ([recall](../../specs/product/recall.md),
  [learn](../../specs/product/learn.md), [ambient-guidance](../../specs/product/ambient-guidance.md)) but replaces the agent's own initiative, so it fits none of them.
- **Alternatives:** split across [recall](../../specs/product/recall.md), [learn](../../specs/product/learn.md) and [ambient-guidance](../../specs/product/ambient-guidance.md). Rejected: each requirement
  would then depend on the agent-initiated behavior those capabilities describe.

## 793b2 ID mapping

- **Decided by:** Author (agent); awaiting owner validation.

- **Choice:** the draft's four groups became four Added sections, and each requirement kept its order.
  Requirements whose meaning later reversed were retired and replaced by new IDs ([793b14 ID handling](#793b14-id-handling)).

| Draft | Section | Draft IDs to IDs |
| --- | --- | --- |
| 4.1 Automatic recall | [793a2 Added: automatic-memory (automatic recall)](793a_product-spec.md#793a2-added-automatic-memory-automatic-recall) | REC-1..REC-5 to [793a2a Recall at a human prompt](793a_product-spec.md#793a2a-recall-at-a-human-prompt)..[793a2e Relevance and no re-injection](793a_product-spec.md#793a2e-relevance-and-no-re-injection); REC-6 and REC-7 to [793a2j Volume equals the depth's output](793a_product-spec.md#793a2j-volume-equals-the-depths-output) and [793a2i Recall for every prompt regardless of source](793a_product-spec.md#793a2i-recall-for-every-prompt-regardless-of-source) (new IDs); [793a2h Recall depth per moment](793a_product-spec.md#793a2h-recall-depth-per-moment) is new |
| 4.2 Automatic capture | [793a3 Added: automatic-memory (automatic capture)](793a_product-spec.md#793a3-added-automatic-memory-automatic-capture) | CAP-1..CAP-4 and CAP-6 to [793a3a Evaluate every completed turn](793a_product-spec.md#793a3a-evaluate-every-completed-turn)..[793a3d Provenance](793a_product-spec.md#793a3d-provenance) and [793a3f Capture does not delay the session](793a_product-spec.md#793a3f-capture-does-not-delay-the-session); CAP-5 to [793a3h Capture rate exceeds the baseline](793a_product-spec.md#793a3h-capture-rate-exceeds-the-baseline) and [793a3i Precision is graded independently](793a_product-spec.md#793a3i-precision-is-graded-independently) (new IDs); [793a3g Deep recall's writes are the System's](793a_product-spec.md#793a3g-deep-recalls-writes-are-the-systems) and [793a3j Direct writes go to the local vault](793a_product-spec.md#793a3j-direct-writes-go-to-the-local-vault) are new |
| 4.3 Operator control and safety | [793a4 Added: automatic-memory (operator control and safety)](793a_product-spec.md#793a4-added-automatic-memory-operator-control-and-safety) | CTL-2..CTL-4 to [793a4b Failure isolation](793a_product-spec.md#793a4b-failure-isolation)..[793a4d Sensitive content](793a_product-spec.md#793a4d-sensitive-content); CTL-1 to [793a4e Operating modes and measurement](793a_product-spec.md#793a4e-operating-modes-and-measurement) (new ID); [793a4f No recall or capture in its own evaluation sessions](793a_product-spec.md#793a4f-no-recall-or-capture-in-its-own-evaluation-sessions) is new |
| 4.4 Independence, coexistence, measurement | [793a5 Added: automatic-memory (independence, coexistence, and measurement)](793a_product-spec.md#793a5-added-automatic-memory-independence-coexistence-and-measurement) | IND-1..IND-3 to [793a5a Independence from agent compliance](793a_product-spec.md#793a5a-independence-from-agent-compliance)..[793a5c Environments without the capability](793a_product-spec.md#793a5c-environments-without-the-capability); NFR-2 and NFR-3 to [793a5e Auditability and measurability](793a_product-spec.md#793a5e-auditability-and-measurability) and [793a5f Effect on outcomes](793a_product-spec.md#793a5f-effect-on-outcomes); NFR-1 to [793a5g Latency per recall moment](793a_product-spec.md#793a5g-latency-per-recall-moment) (new ID) |

- **Rationale:** unchanged requirements keep "(formerly X)" so the draft's review comments still map.
  Cross-references inside requirements and scenarios were rewritten to the current IDs.

## 793b3 Scenarios added for uncovered MUST branches

- **Decided by:** Author (agent); awaiting owner validation.

- **Choice:** every MUST and MUST NOT branch has a scenario. Added in this revision: the in-context
  branches of [793a2e Relevance and no re-injection](793a_product-spec.md#793a2e-relevance-and-no-re-injection) (retrieved by the agent, parent versus subagent, compaction, truncation,
  resume), the depth branches of [793a2h Recall depth per moment](793a_product-spec.md#793a2h-recall-depth-per-moment), the direct-write branches of [793a3j Direct writes go to the local vault](793a_product-spec.md#793a3j-direct-writes-go-to-the-local-vault), the concurrent
  and parent-reported duplicates of [793a3c No duplicates](793a_product-spec.md#793a3c-no-duplicates), the routine-success exclusion of [793a3b Confirmed lessons only](793a_product-spec.md#793a3b-confirmed-lessons-only), the
  no-error-to-the-agent branch of [793a4b Failure isolation](793a_product-spec.md#793a4b-failure-isolation), and a scenario per mode and per measurement branch of
  [793a4e Operating modes and measurement](793a_product-spec.md#793a4e-operating-modes-and-measurement).
- **Rationale:** the product spec guide requires coverage for every distinct mandatory branch.
- **Alternatives:** one scenario per requirement. Rejected: the guide calls that insufficient.

## 793b4 Draft sections that are not requirements

- **Decided by:** Author (agent); awaiting owner validation. The widening of the System's Lesson definition mentioned here is Owner (Joe), [793c7 Review answers (2026-10-11)](793c_open-questions.md#793c7-review-answers-2026-10-11) item 9 (see [793b17 Lesson is wider than learn's definition](793b_decision-log.md#793b17-lesson-is-wider-than-learns-definition)).

- **Choice:**
  - **Terms (draft §2)** became concise context in the problem statement ([793a1 Problem Statement](793a_product-spec.md#793a1-problem-statement)).
  - **Scope (draft §3)** is not kept as a section, because the guide allows only the problem
    statement, requirements and UAT. The in-scope items are the requirements themselves. The enduring
    boundary, no effect on anything but injected reference notes and vault writes, is already the
    MUST NOT in [793a4c Memory purview only](793a_product-spec.md#793a4c-memory-purview-only). The remaining out-of-scope items are recorded here as context: changing note
    format or retrieval ranking, memory-unrelated behavior (task decisions, permissions, model
    choice, code review), and designing the solution. They bound this change, so they are not
    promoted as requirements. Changing what counts as a lesson was out of scope in the draft; Joe's
    review widened the System's definition ([793b17 Lesson is wider than learn's definition](#793b17-lesson-is-wider-than-learns-definition)).
  - **Acknowledged coverage limits (draft §5)** are context, not requirements:
    - A turn where the working agent produces a final text-only answer with no further model call has
      no later moment for recall. Capture of that turn still applies.
    - Some session types (forked sessions, subtasks, teammate agents, cloud or diskless sessions,
      other harnesses) may not present all moments. Requirements apply to the moments the
      environment exposes.
    - The mechanism may be unavailable by remote rollout decision. [793a5c Environments without the capability](793a_product-spec.md#793a5c-environments-without-the-capability) covers it.
    The solution document must state how each is handled or accepted.
  - **Status and solution boundary.** The draft was marked DRAFT, with problem and verifiable
    requirements only. The solution document is still to be written after approval.
- **Rationale:** `docs/PRODUCT-SPEC-GUIDE.md` and the lifecycle guide keep change context out of the
  product specification.

## 793b5 Traceability from the draft (evidence to requirement)

- **Decided by:** Author (agent); awaiting owner validation.

| Problem evidence | Requirements |
| --- | --- |
| Learn fires about 5% | [793a3a Evaluate every completed turn](793a_product-spec.md#793a3a-evaluate-every-completed-turn), [793a3b Confirmed lessons only](793a_product-spec.md#793a3b-confirmed-lessons-only), [793a3h Capture rate exceeds the baseline](793a_product-spec.md#793a3h-capture-rate-exceeds-the-baseline), [793a3i Precision is graded independently](793a_product-spec.md#793a3i-precision-is-graded-independently), [793a5a Independence from agent compliance](793a_product-spec.md#793a5a-independence-from-agent-compliance) |
| 0 of 4 subagents received memory | [793a2c Recall at subagent dispatch](793a_product-spec.md#793a2c-recall-at-subagent-dispatch) |
| Shim refused as injection 6 of 6 | [793a2d Attributable presentation](793a_product-spec.md#793a2d-attributable-presentation), [793a5a Independence from agent compliance](793a_product-spec.md#793a5a-independence-from-agent-compliance) |
| Prose fixes plateau | [793a5a Independence from agent compliance](793a_product-spec.md#793a5a-independence-from-agent-compliance), [793a5f Effect on outcomes](793a_product-spec.md#793a5f-effect-on-outcomes) |
| Past noise and over-firing concerns | [793a2e Relevance and no re-injection](793a_product-spec.md#793a2e-relevance-and-no-re-injection), [793a2h Recall depth per moment](793a_product-spec.md#793a2h-recall-depth-per-moment), [793a2i Recall for every prompt regardless of source](793a_product-spec.md#793a2i-recall-for-every-prompt-regardless-of-source), [793a2j Volume equals the depth's output](793a_product-spec.md#793a2j-volume-equals-the-depths-output), [793a5g Latency per recall moment](793a_product-spec.md#793a5g-latency-per-recall-moment) |
| Safety of an unattended process | [793a4b Failure isolation](793a_product-spec.md#793a4b-failure-isolation), [793a4c Memory purview only](793a_product-spec.md#793a4c-memory-purview-only), [793a4d Sensitive content](793a_product-spec.md#793a4d-sensitive-content), [793a4e Operating modes and measurement](793a_product-spec.md#793a4e-operating-modes-and-measurement), [793a4f No recall or capture in its own evaluation sessions](793a_product-spec.md#793a4f-no-recall-or-capture-in-its-own-evaluation-sessions), [793a5e Auditability and measurability](793a_product-spec.md#793a5e-auditability-and-measurability) |

## 793b6 No proposed values remain

- **Decided by:** Owner (Joe), [793c7 Review answers (2026-10-11)](793c_open-questions.md#793c7-review-answers-2026-10-11) item 12, accepted N = 20 and the removal of the proposed marker. Author (agent), removal of the refusal-rate floor follows [793b15 Attributable presentation only](793b_decision-log.md#793b15-attributable-presentation-only); awaiting owner validation.

- **Choice:** [793a5f Effect on outcomes](793a_product-spec.md#793a5f-effect-on-outcomes) keeps N = 20 runs per arm, accepted by Joe, and loses its proposed marker. The
  95% / N=20 refusal-rate floor was removed with the refusal check ([793b15 Attributable presentation only](#793b15-attributable-presentation-only)).
- **Rationale:** the guide forbids exceeding the behavior the user specified, and Joe accepted the one
  remaining value.

## 793b7 Volume and latency equal the chosen depth's output

- **Decided by:** Owner (Joe), [793c2 Are the volume bounds and latency budget accepted?](793c_open-questions.md#793c2-are-the-volume-bounds-and-latency-budget-accepted) and [793c7 Review answers (2026-10-11)](793c_open-questions.md#793c7-review-answers-2026-10-11) items 1 to 4, decided that bounds equal recall's output, drop the per-turn and per-session bounds, apply at tool-result moments, and cap latency at the depth's query latency. Author (agent), interpreting [793c7 Review answers (2026-10-11)](793c_open-questions.md#793c7-review-answers-2026-10-11), chose to make the bounds reference the current query instead of fixed numbers and chose the recall-query citations; awaiting owner validation.

- **Choice:** [793a2j Volume equals the depth's output](793a_product-spec.md#793a2j-volume-equals-the-depths-output) sets the injected volume at each recall moment to what that depth's queries
  return (glance: one query with about 3 phrases plus the re-entry query; deep: one query with 10
  phrases, the recommendation check and the re-entry query), less notes already in context. That output
  is floor and ceiling. [793a5g Latency per recall moment](793a_product-spec.md#793a5g-latency-per-recall-moment) caps the added latency at the latency of those queries on the same
  vault and input. The earlier floor-only wording ("not lower than one run of the current recall
  query"), and the per-turn and per-session bounds, are gone.
- **Choice:** the bound applies at prompts, tool results and subagent dispatches alike.
- **Choice:** the recall-query citations link [792a7b The payload](../../specs/product/memory-query.md#792a7b-the-payload), [792a14c Recall has two depths](../../specs/product/recall.md#792a14c-recall-has-two-depths), [792a14f One unified query](../../specs/product/recall.md#792a14f-one-unified-query), [792a14e Ten phrases from ten angles](../../specs/product/recall.md#792a14e-ten-phrases-from-ten-angles) and
  [792a14o An emergent recommendation is checked against the vault](../../specs/product/recall.md#792a14o-an-emergent-recommendation-is-checked-against-the-vault), not [ambient-guidance](../../specs/product/ambient-guidance.md), because those requirements define the query and its payload.
- **Rationale:** Joe's decision: bounds equal what recall delivers today, with no further reduction
  required. A fixed number would go stale when the query's defaults or the vault change; the reference
  stays true and an operator can verify it by running the queries. Reductions beyond that depend on
  #845 and are not normative here.
- **Alternatives:** keep per-turn and per-session bounds. Rejected by Joe.

## 793b8 Per-moment record sits in the existing log requirement

- **Decided by:** Owner (Joe), [793c7 Review answers (2026-10-11)](793c_open-questions.md#793c7-review-answers-2026-10-11) item 4, decided that per-moment volume and latency are recorded. Author (agent), interpreting [793c7 Review answers (2026-10-11)](793c_open-questions.md#793c7-review-answers-2026-10-11) item 4, chose to log the depth and the phrases queried and to place the record in the existing requirement; awaiting owner validation.

- **Choice:** the volume and latency record is part of [793a5e Auditability and measurability](793a_product-spec.md#793a5e-auditability-and-measurability), with scenarios [793a5e1b Volume and latency recorded per moment](793a_product-spec.md#793a5e1-uat-scenarios) and
  [793a5e1c A moment with no injection is recorded](793a_product-spec.md#793a5e1-uat-scenarios), not a new requirement. The log is available by session id, each entry carries session
  id, turn id and moment, and a moment with no injection is recorded with volume 0.
- **Choice:** the log also records the depth and the phrases queried, so that [793a2h1a A routine decision uses glance](793a_product-spec.md#793a2h1-uat-scenarios) and
  [793a2j1a Delivered notes equal the depth's output](793a_product-spec.md#793a2j1-uat-scenarios) can be checked from the operator-accessible log. This goes beyond Joe's wording and is
  raised in [793c7 Review answers (2026-10-11)](793c_open-questions.md#793c7-review-answers-2026-10-11).
- **Rationale:** it is one more field of the decision log that requirement already mandates.

## 793b9 Capture rate and precision are separate requirements

- **Decided by:** Owner (Joe), [793c1 What are the capture precision bar and capture-rate target?](793c_open-questions.md#793c1-what-are-the-capture-precision-bar-and-capture-rate-target) and [793c7 Review answers (2026-10-11)](793c_open-questions.md#793c7-review-answers-2026-10-11) items 7 and 8, decided that the rate must beat the baseline and that a separate grader reports precision with its sample size. Author (agent), interpreting [793c7 Review answers (2026-10-11)](793c_open-questions.md#793c7-review-answers-2026-10-11) item 8, chose the comparator 5.6% with no margin, and the split of 793a3e into a rate requirement and a precision requirement; awaiting owner validation.

- **Choice:** [793a3h Capture rate exceeds the baseline](793a_product-spec.md#793a3h-capture-rate-exceeds-the-baseline) requires the capture rate to exceed 5.6% on the point estimate, with no
  margin; the 60-moment labeled set is kept from the draft. [793a3i Precision is graded independently](793a_product-spec.md#793a3i-precision-is-graded-independently) requires a separate grader inside
  the System, independent of the capture judgment, and reports precision with its sample size; the
  30-note sample is kept. Neither has a precision threshold.
- **Rationale:** 5.6% is the higher of the two measured baselines (4/71 = 5.6%, 3/66 = 4.5%; LEDGER
  `739-audit-w2-learn-fired`, `learn-rate-remeasure-w2`). Splitting rate from precision gives each one
  statement and its own scenarios; the grader keeps the capture judge from marking its own work.

## 793b10 Default mode and measurement tied to mode

- **Decided by:** Owner (Joe), [793c3 What is the default mode for a fresh install?](793c_open-questions.md#793c3-what-is-the-default-mode-for-a-fresh-install) and [793c7 Review answers (2026-10-11)](793c_open-questions.md#793c7-review-answers-2026-10-11) item 6, decided the active default and that measurement is not switchable apart from the mode. Author (agent) chose the scenario set and the "off records nothing" wording; awaiting owner validation.

- **Choice:** [793a4e Operating modes and measurement](793a_product-spec.md#793a4e-operating-modes-and-measurement) makes the default MUST active, and requires measurement whenever the mode is
  observe or active, not switchable off independently. Off records nothing, so scenario [793a4e1b Off mode is inert](793a_product-spec.md#793a4e1-uat-scenarios) asserts
  no log entry. Scenario [793a4e1e Measurement cannot be switched off apart from the mode](793a_product-spec.md#793a4e1-uat-scenarios) covers an attempt to switch measurement off apart from the mode.
- **Rationale:** an off System has nothing to measure; observing and active both produce decisions to
  record.

## 793b11 Direct capture: where it is stated

- **Decided by:** Owner (Joe), [793c4 Are captured lessons written directly, or held for `curate` review?](793c_open-questions.md#793c4-are-captured-lessons-written-directly-or-held-for-curate-review), decided that captured lessons are written directly. Author (agent), interpreting [793c4 Are captured lessons written directly, or held for `curate` review?](793c_open-questions.md#793c4-are-captured-lessons-written-directly-or-held-for-curate-review), chose where the rule is stated and which assertions the scenarios make; awaiting owner validation.

- **Choice:** the direct-write rule and the System-judges rule are stated in [793a3a Evaluate every completed turn](793a_product-spec.md#793a3a-evaluate-every-completed-turn). Scenarios
  [793a3a1a User correction is captured without agent action](793a_product-spec.md#793a3a1-uat-scenarios) and [793a3a1b Explicit save-request](793a_product-spec.md#793a3a1-uat-scenarios) assert the vault, not a dry-run log. [793a3a1a User correction is captured without agent action](793a_product-spec.md#793a3a1-uat-scenarios) also asserts, from the
  transcript, that the agent invoked no learn command and saw no request to evaluate or write.
  Observe-mode logging remains in [793a4e1a Observe mode has no side effects](793a_product-spec.md#793a4e1-uat-scenarios).
- **Rationale:** those assertions are what "direct, without polluting the main agent's context" means
  to an end user.

## 793b12 Recall for every prompt

- **Decided by:** Owner (Joe), [793c5 Does 793a2i Recall for every prompt regardless of source keep human-originated prompts as the default?](793c_open-questions.md#793c5-does-793a2i-recall-for-every-prompt-regardless-of-source-keep-human-originated-prompts-as-the-default), decided that recall applies to every prompt. Author (agent), interpreting [793c5 Does 793a2i Recall for every prompt regardless of source keep human-originated prompts as the default?](793c_open-questions.md#793c5-does-793a2i-recall-for-every-prompt-regardless-of-source-keep-human-originated-prompts-as-the-default) and [793c7 Review answers (2026-10-11)](793c_open-questions.md#793c7-review-answers-2026-10-11) item 5, chose to keep the re-injection MUST NOT in 793a2e and to use a Scenario Outline; awaiting owner validation.

- **Choice:** [793a2i Recall for every prompt regardless of source](793a_product-spec.md#793a2i-recall-for-every-prompt-regardless-of-source) replaces the retired `793a2g` and keeps only the source-independence MUST. The
  in-context MUST NOT moved to [793a2e Relevance and no re-injection](793a_product-spec.md#793a2e-relevance-and-no-re-injection), which now owns "never inject a note already in the receiving
  agent's context". [793a2a Recall at a human prompt](793a_product-spec.md#793a2a-recall-at-a-human-prompt) is unchanged; it is the human case of the rule.
- **Choice:** the scenario is a Scenario Outline over the three non-human sources the old requirement
  named.
- **Rationale:** the three sources behave identically, so one outline suffices; keeping the
  in-context rule in one requirement avoids a duplicate rule.

## 793b13 Capture in subagent sessions

- **Decided by:** Owner (Joe), [793c6 Does capture also run inside subagent sessions?](793c_open-questions.md#793c6-does-capture-also-run-inside-subagent-sessions), decided that capture runs in subagent sessions. Author (agent) chose the scenario set; awaiting owner validation.

- **Choice:** [793a3a Evaluate every completed turn](793a_product-spec.md#793a3a-evaluate-every-completed-turn) names "both top-level and subagent sessions". Scenarios [793a3a1c A subagent's own guess overturned by evidence is captured](793a_product-spec.md#793a3a1-uat-scenarios) (the
  subagent's own overturned guess), [793a3a1d A correction from the dispatching agent is captured](793a_product-spec.md#793a3a1-uat-scenarios) (a correction from the dispatching agent) and
  [793a3a1e Capture survives the subagent's context being discarded](793a_product-spec.md#793a3a1-uat-scenarios) (capture survives the discarded context) cover them; [793a3c1b A lesson reported to the parent yields one note](793a_product-spec.md#793a3c1-uat-scenarios) covers a lesson also
  reported to the parent.
- **Rationale:** the lesson kinds apply to a subagent's work as to the top level's.

## 793b14 ID handling

- **Decided by:** Owner (Joe), [793c7 Review answers (2026-10-11)](793c_open-questions.md#793c7-review-answers-2026-10-11) item 11, decided that nodes whose meaning reversed get new IDs and old IDs are retired. Author (agent) chose the ID map and which nodes count as modified in place; awaiting owner validation.

- **Choice:** per the lifecycle rule that a materially new node gets a new ID, nodes whose meaning
  reversed in the earlier revision were retired and replaced. Retired IDs are never reused.

| Retired ID | Replaced by | Why |
| --- | --- | --- |
| `793a2f`, `793a2f1a`..`793a2f1c` | [793a2j Volume equals the depth's output](793a_product-spec.md#793a2j-volume-equals-the-depths-output), [793a2j1a Delivered notes equal the depth's output](793a_product-spec.md#793a2j1-uat-scenarios), [793a2j1b A vault holding more than 20 notes about the topic](793a_product-spec.md#793a2j1-uat-scenarios) | the bound reversed from a ceiling to a floor, then to depth output as floor and ceiling |
| `793a2g` | [793a2i Recall for every prompt regardless of source](793a_product-spec.md#793a2i-recall-for-every-prompt-regardless-of-source) (source independence) and [793a2e Relevance and no re-injection](793a_product-spec.md#793a2e-relevance-and-no-re-injection) (in-context rule) | "human only" reversed to "every prompt" |
| `793a2g1a` | [793a2i1a Non-human prompt receives recall](793a_product-spec.md#793a2i1-uat-scenarios) | skip reversed to inject |
| `793a2g1b`, `793a2g1c` | [793a2e1a Note delivered earlier is not delivered again](793a_product-spec.md#793a2e1-uat-scenarios), [793a2e1c Note the agent retrieved itself is not injected](793a_product-spec.md#793a2e1-uat-scenarios) | the in-context rule moved to its owner |
| `793a3e`, `793a3e1a`, `793a3e1b` | [793a3h Capture rate exceeds the baseline](793a_product-spec.md#793a3h-capture-rate-exceeds-the-baseline) and [793a3i Precision is graded independently](793a_product-spec.md#793a3i-precision-is-graded-independently), [793a3i1a Precision is reported with its sample size](793a_product-spec.md#793a3i1-uat-scenarios), [793a3h1a Capture rate exceeds the baseline](793a_product-spec.md#793a3h1-uat-scenarios) | a 70% / 50% bar reversed to "beat the baseline, record precision" |
| `793a4a`, `793a4a1a`..`793a4a1c` | [793a4e Operating modes and measurement](793a_product-spec.md#793a4e-operating-modes-and-measurement), [793a4e1a Observe mode has no side effects](793a_product-spec.md#793a4e1-uat-scenarios), [793a4e1b Off mode is inert](793a_product-spec.md#793a4e1-uat-scenarios), [793a4e1c Fresh install is active and measured](793a_product-spec.md#793a4e1-uat-scenarios) | the default reversed from observe to active |
| `793a5d`, `793a5d1a` | [793a5g Latency per recall moment](793a_product-spec.md#793a5g-latency-per-recall-moment), [793a5g1a Added latency stays within the depth's query latency](793a_product-spec.md#793a5g1-uat-scenarios) | the latency ceiling reversed to a floor, then to the depth's query latency |

- **Choice:** requirements modified in place keep their IDs: [793a2a Recall at a human prompt](793a_product-spec.md#793a2a-recall-at-a-human-prompt), [793a2b Recall after each tool result](793a_product-spec.md#793a2b-recall-after-each-tool-result), [793a2c Recall at subagent dispatch](793a_product-spec.md#793a2c-recall-at-subagent-dispatch),
  [793a2d Attributable presentation](793a_product-spec.md#793a2d-attributable-presentation) (retitled), [793a2e Relevance and no re-injection](793a_product-spec.md#793a2e-relevance-and-no-re-injection) (retitled), [793a3a Evaluate every completed turn](793a_product-spec.md#793a3a-evaluate-every-completed-turn), [793a3b Confirmed lessons only](793a_product-spec.md#793a3b-confirmed-lessons-only), [793a3c No duplicates](793a_product-spec.md#793a3c-no-duplicates), [793a3f Capture does not delay the session](793a_product-spec.md#793a3f-capture-does-not-delay-the-session)
  (retitled), [793a4b Failure isolation](793a_product-spec.md#793a4b-failure-isolation), [793a4c Memory purview only](793a_product-spec.md#793a4c-memory-purview-only), [793a4d Sensitive content](793a_product-spec.md#793a4d-sensitive-content), [793a5e Auditability and measurability](793a_product-spec.md#793a5e-auditability-and-measurability) and [793a5f Effect on outcomes](793a_product-spec.md#793a5f-effect-on-outcomes).
- **Choice:** new nodes: [793a2h Recall depth per moment](793a_product-spec.md#793a2h-recall-depth-per-moment), [793a3g Deep recall's writes are the System's](793a_product-spec.md#793a3g-deep-recalls-writes-are-the-systems), [793a3j Direct writes go to the local vault](793a_product-spec.md#793a3j-direct-writes-go-to-the-local-vault), [793a4f No recall or capture in its own evaluation sessions](793a_product-spec.md#793a4f-no-recall-or-capture-in-its-own-evaluation-sessions), plus the scenarios named in
  [793b3 Scenarios added for uncovered MUST branches](#793b3-scenarios-added-for-uncovered-must-branches). The earlier log called `793a2g`'s change "Modified"; it was a replacement and is
  corrected here. These IDs are change-internal and never integrated.
- **Rationale:** the lifecycle guide keeps IDs stable through modification and gives a materially new
  node a new ID.

## 793b15 Attributable presentation only

- **Decided by:** Owner (Joe), [793c7 Review answers (2026-10-11)](793c_open-questions.md#793c7-review-answers-2026-10-11) item 13, decided to keep only attribution and drop the non-imperative rule and the refusal check.

- **Choice:** [793a2d Attributable presentation](793a_product-spec.md#793a2d-attributable-presentation) keeps only "injected notes MUST identify their source note" and is retitled
  "Attributable presentation". The non-imperative rule and the 95%-of-20-runs refusal scenario are
  removed. Imperative memory such as runbooks and skills is allowed.
- **Rationale:** [793a5f Effect on outcomes](793a_product-spec.md#793a5f-effect-on-outcomes), whether injected memory gets applied, is the real test.

## 793b16 Observable scenarios

- **Decided by:** Author (agent); awaiting owner validation.

- **Choice:** scenarios assert what the session transcript shows (note content delivered before the
  agent's first response or tool call, or a subagent's transcript) or what the operator-accessible log
  shows. None asserts an internal score or "context". The term Context stays in requirements, where it
  defines presence.
- **Choice:** `793a2f`'s retired scenarios are replaced by [793a2j1b A vault holding more than 20 notes about the topic](793a_product-spec.md#793a2j1-uat-scenarios), which sets up "a vault holding
  more than 20 notes about the prompt's topic" and asserts only what the requirement states.
- **Rationale:** the product spec guide requires scenarios an end user can verify.

## 793b17 Lesson is wider than learn's definition

- **Decided by:** Owner (Joe), [793c7 Review answers (2026-10-11)](793c_open-questions.md#793c7-review-answers-2026-10-11) item 9, decided that a Lesson is any corrected failure and that it is wider than learn's definition. Author (agent), interpreting [793c7 Review answers (2026-10-11)](793c_open-questions.md#793c7-review-answers-2026-10-11) item 9, widened the Lesson term in the term and rationale and kept the subagent scenario; awaiting owner validation.

- **Choice:** the term Lesson in [793a1 Problem Statement](793a_product-spec.md#793a1-problem-statement) means any corrected failure, including an agent's own
  mid-task mistake overturned by evidence, plus corrections, explicit save-requests and confirmed
  approaches, and links [792a15e The four kinds of explicit lesson](../../specs/product/learn.md#792a15e-the-four-kinds-of-explicit-lesson). [793a3b Confirmed lessons only](793a_product-spec.md#793a3b-confirmed-lessons-only) keeps excluding unconfirmed hypotheses, routine successes
  and pleasantries. The rationale in [793a3a Evaluate every completed turn](793a_product-spec.md#793a3a-evaluate-every-completed-turn) says it is wider than learn's current definition and
  that #846 tracks widening learn's.
- **Choice:** the earlier "test overturns the subagent's own guess" scenario stays as [793a3a1c A subagent's own guess overturned by evidence is captured](793a_product-spec.md#793a3a1-uat-scenarios).
- **Rationale:** a mistake the agent catches and fixes is as reusable as one the user catches.

## 793b18 Edge cases

- **Decided by:** Owner (Joe), [793c7 Review answers (2026-10-11)](793c_open-questions.md#793c7-review-answers-2026-10-11) item 10, decided that secrets are a MUST NOT. Author (agent), interpreting Joe's request in review to pin down boundaries, chose the other edge cases; awaiting owner validation.

- **Choice:** the System never recalls into or captures from its own evaluation sessions ([793a4f No recall or capture in its own evaluation sessions](793a_product-spec.md#793a4f-no-recall-or-capture-in-its-own-evaluation-sessions));
  two subagents capturing the same lesson concurrently yield one note ([793a3c1c Two subagents capture the same lesson concurrently](793a_product-spec.md#793a3c1-uat-scenarios)); a resumed
  session's restored content counts as present ([793a2e1g A resumed session's restored content counts as present](793a_product-spec.md#793a2e1-uat-scenarios)); direct writes go to the local vault with
  parent offers following normal note-writing behavior, citing [parent-sync](../../specs/product/parent-sync.md) and [792a21n A local write is offered to the parent](../../specs/product/parent-sync.md#792a21n-a-local-write-is-offered-to-the-parent)
  ([793a3j Direct writes go to the local vault](793a_product-spec.md#793a3j-direct-writes-go-to-the-local-vault)); secrets are a MUST NOT ([793a4d Sensitive content](793a_product-spec.md#793a4d-sensitive-content)).
- **Rationale:** each is a boundary Joe asked to pin down in review.

## 793b19 Capture does not delay the session

- **Decided by:** Author (agent); awaiting owner validation.

- **Choice:** [793a3f Capture does not delay the session](793a_product-spec.md#793a3f-capture-does-not-delay-the-session) is retitled "Capture does not delay the session" and states the only capture
  latency rule: capture MUST NOT delay the working agent's response or the processing of the next
  prompt. The overlapping capture clause was removed from [793a5g Latency per recall moment](793a_product-spec.md#793a5g-latency-per-recall-moment), which now covers recall latency
  only. [793a4b Failure isolation](793a_product-spec.md#793a4b-failure-isolation) also requires the failure to be logged for the operator.
- **Rationale:** one rule, in one place.

## 793b20 Deep recall's writes and no self-links

- **Decided by:** Owner (Joe), [793c7 Review answers (2026-10-11)](793c_open-questions.md#793c7-review-answers-2026-10-11) items 1 and 14, decided that deep recall's writes are the System's and that requirement lines carry no self-links. Author (agent) chose the requirement's wording; awaiting owner validation.

- **Choice:** [793a3g Deep recall's writes are the System's](793a_product-spec.md#793a3g-deep-recalls-writes-are-the-systems) makes the System's capture, not the main agent, perform deep recall's coverage
  writes and persisted conclusions. `Requirement:` lines no longer link themselves; the canonical
  specs' gaps are tracked in #847.
- **Rationale:** Joe's decisions; a definition is not a reference.

## 793b21 Today's recall is both floor and ceiling of the bounds

- **Decided by:** Orchestrator (Claude), interpreting [793c2 Are the volume bounds and latency budget accepted?](793c_open-questions.md#793c2-are-the-volume-bounds-and-latency-budget-accepted); awaiting owner validation.
- **Choice:** read [793c2 Are the volume bounds and latency budget accepted?](793c_open-questions.md#793c2-are-the-volume-bounds-and-latency-budget-accepted) "start with today's values" as making today's recall output both the floor and the
  ceiling of the bounds. Joe later confirmed this for the one-query-per-moment case in his review answers ([793c7 Review answers (2026-10-11)](793c_open-questions.md#793c7-review-answers-2026-10-11) item 1).
- **Rationale:** "today's values" names a level to match, not a minimum; the review answers confirmed it for one query per moment.
- **Alternatives:** treat today's values as a floor only (the earlier wording). Rejected in review.

## 793b22 "Learn from all failures" splits into the System's definition and #846

- **Decided by:** Orchestrator (Claude), interpreting [793c7 Review answers (2026-10-11)](793c_open-questions.md#793c7-review-answers-2026-10-11) item 9; awaiting owner validation.
- **Choice:** map Joe's "learn from all failures" ([793c7 Review answers (2026-10-11)](793c_open-questions.md#793c7-review-answers-2026-10-11) item 9) to two things: (i) widening the System's Lesson
  definition in 793a ([793b17 Lesson is wider than learn's definition](793b_decision-log.md#793b17-lesson-is-wider-than-learns-definition)), and (ii) filing #846 to widen learn's own [792a15e The four kinds of explicit lesson](../../specs/product/learn.md#792a15e-the-four-kinds-of-explicit-lesson) separately. Joe confirmed this split in the review answers.
- **Rationale:** the System's definition is this change's to set; learn's definition belongs to the canonical learn capability.
- **Alternatives:** widen learn's definition inside 793. Rejected: it changes a canonical capability outside this change.

## 793b23 The per-depth recall triggers are the System's depth rule

- **Decided by:** Orchestrator (Claude), interpreting [793c7 Review answers (2026-10-11)](793c_open-questions.md#793c7-review-answers-2026-10-11) item 1; awaiting owner validation.
- **Choice:** treat the per-depth recall triggers as the System's depth rule ([793a2h Recall depth per moment](793a_product-spec.md#793a2h-recall-depth-per-moment)): deep when the decision is weighty or
  irreversible, when glance flags a gap, or for a recent standard ([792a14d Glance escalates to deep for a recent standard](../../specs/product/recall.md#792a14d-glance-escalates-to-deep-for-a-recent-standard)); glance otherwise. Joe chose this as option (a) in Q1b.
- **Rationale:** it reuses the triggers recall already applies, so the System and the instruction path pick the same depth.
- **Alternatives:** a new System-only depth rule. Not chosen.

## 793b24 Follow-up issues #845, #846 and #847 instead of addressing them in 793

- **Decided by:** Orchestrator (Claude), interpreting [793c7 Review answers (2026-10-11)](793c_open-questions.md#793c7-review-answers-2026-10-11) items 4, 9 and 14; awaiting owner validation.
- **Choice:** file #845 (measure before reducing bounds), #846 (learn's definition) and #847 (self-links in canonical specs) instead of addressing them in this change.
- **Rationale:** each concerns something outside this change or needs data this change only starts to collect.
- **Alternatives:** fix them inside 793. Rejected: scope.

## 793b25 H2: drop the double count of the re-entry query

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Finding:** The re-entry query is [792a14o An emergent recommendation is checked against the vault](../../specs/product/recall.md#792a14o-an-emergent-recommendation-is-checked-against-the-vault)'s recommendation check. It runs at both depths, conditionally, so [793a2j Volume equals the depth's output](793a_product-spec.md#793a2j-volume-equals-the-depths-output) and [793b7 Volume and latency equal the chosen depth's output](793b_decision-log.md#793b7-volume-and-latency-equal-the-chosen-depths-output) count it twice.
- **Fix to be applied:** Reword [793a2j Volume equals the depth's output](793a_product-spec.md#793a2j-volume-equals-the-depths-output) and [793b7 Volume and latency equal the chosen depth's output](793b_decision-log.md#793b7-volume-and-latency-equal-the-chosen-depths-output) to list the conditional recommendation check once, at both depths, with no separate re-entry query.
- **Status:** accepted, not yet applied. The second review of commit 058c70b3 found it; the orchestrator accepted it as a fix to make without asking Joe.

## 793b26 H3: lazy mode truncates only chunks, never notes

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Finding:** Lazy mode truncates only chunks. [793a2e Relevance and no re-injection](793a_product-spec.md#793a2e-relevance-and-no-re-injection) treats a truncated note as not present, which cannot occur.
- **Fix to be applied:** [793a2e Relevance and no re-injection](793a_product-spec.md#793a2e-relevance-and-no-re-injection) becomes "an item (note or chunk) shown without its content doesn't count as present"; no re-injection covers chunks too; [793a2e1f A truncated note is injected in full when relevant](793a_product-spec.md#793a2e1-uat-scenarios) is rewritten around a chunk.
- **Status:** accepted, not yet applied. The second review of commit 058c70b3 found it; the orchestrator accepted it as a fix to make without asking Joe.

## 793b27 H4: an escalated moment covers both depths' queries

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Finding:** When glance escalates to deep, the moment runs the queries of both depths, but [793a2j Volume equals the depth's output](793a_product-spec.md#793a2j-volume-equals-the-depths-output) and [793a5g Latency per recall moment](793a_product-spec.md#793a5g-latency-per-recall-moment) bound only one.
- **Fix to be applied:** An escalated moment's volume and latency cover both depths' queries (the union). Add subagent dispatch to [793a5g1a Added latency stays within the depth's query latency](793a_product-spec.md#793a5g1-uat-scenarios).
- **Status:** accepted, not yet applied. The second review of commit 058c70b3 found it; the orchestrator accepted it as a fix to make without asking Joe.

## 793b28 M1: observe mode logs the would-be volume

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Finding:** Observe mode injects nothing, so its log cannot show the volume that would have been injected.
- **Fix to be applied:** Observe mode logs the would-be volume, in bytes and item count ([793a4e Operating modes and measurement](793a_product-spec.md#793a4e-operating-modes-and-measurement), [793a5e Auditability and measurability](793a_product-spec.md#793a5e-auditability-and-measurability)).
- **Status:** accepted, not yet applied. The second review of commit 058c70b3 found it; the orchestrator accepted it as a fix to make without asking Joe.

## 793b29 M2: the System's own sessions are identifiable

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Finding:** [793a4f No recall or capture in its own evaluation sessions](793a_product-spec.md#793a4f-no-recall-or-capture-in-its-own-evaluation-sessions) depends on the System recognizing its own sessions, but no requirement says how they are identified.
- **Fix to be applied:** [793a4f No recall or capture in its own evaluation sessions](793a_product-spec.md#793a4f-no-recall-or-capture-in-its-own-evaluation-sessions): the System logs the session ids it starts itself as System sessions. Add a scenario for the depth and phrase choice.
- **Status:** accepted, not yet applied. The second review of commit 058c70b3 found it; the orchestrator accepted it as a fix to make without asking Joe.

## 793b30 M3: deep recall's writes carry provenance

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Finding:** [793a3g Deep recall's writes are the System's](793a_product-spec.md#793a3g-deep-recalls-writes-are-the-systems) writes have no provenance, and [793a3i Precision is graded independently](793a_product-spec.md#793a3i-precision-is-graded-independently) grades more than captures.
- **Fix to be applied:** [793a3g Deep recall's writes are the System's](793a_product-spec.md#793a3g-deep-recalls-writes-are-the-systems) writes carry [793a3d Provenance](793a_product-spec.md#793a3d-provenance) provenance plus their derived marker. Scope [793a3i Precision is graded independently](793a_product-spec.md#793a3i-precision-is-graded-independently) to [793a3a Evaluate every completed turn](793a_product-spec.md#793a3a-evaluate-every-completed-turn) captures. [793a3g1a A coverage write is made without the working agent](793a_product-spec.md#793a3g1-uat-scenarios) and [793a3g1b A sound conclusion is persisted without the working agent](793a_product-spec.md#793a3g1-uat-scenarios) assert that no write request appears in the transcript.
- **Status:** accepted, not yet applied. The second review of commit 058c70b3 found it; the orchestrator accepted it as a fix to make without asking Joe.

## 793b31 M4: coexistence defers to the owning requirements

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Finding:** [793a5b Coexistence without double-firing](793a_product-spec.md#793a5b-coexistence-without-double-firing) restates duplicate rules that [793a2e Relevance and no re-injection](793a_product-spec.md#793a2e-relevance-and-no-re-injection) and [793a3c No duplicates](793a_product-spec.md#793a3c-no-duplicates) own, and the outcome of [793a5b1a Both paths active](793a_product-spec.md#793a5b1-uat-scenarios) contradicts them.
- **Fix to be applied:** [793a5b Coexistence without double-firing](793a_product-spec.md#793a5b-coexistence-without-double-firing) defers to [793a2e Relevance and no re-injection](793a_product-spec.md#793a2e-relevance-and-no-re-injection) and [793a3c No duplicates](793a_product-spec.md#793a3c-no-duplicates). Fix the [793a5b1a Both paths active](793a_product-spec.md#793a5b1-uat-scenarios) outcome.
- **Status:** accepted, not yet applied. The second review of commit 058c70b3 found it; the orchestrator accepted it as a fix to make without asking Joe.

## 793b32 M5: the purview rule restricts the System, not the main agent

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Finding:** [793a4c Memory purview only](793a_product-spec.md#793a4c-memory-purview-only) reads as if the System could never affect the agent at all, though it injects notes by design.
- **Fix to be applied:** [793a4c Memory purview only](793a_product-spec.md#793a4c-memory-purview-only) becomes "MUST NOT itself make or override …". Fix its scenario.
- **Status:** accepted, not yet applied. The second review of commit 058c70b3 found it; the orchestrator accepted it as a fix to make without asking Joe.

## 793b33 M6: grade every capture independently

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Finding:** [793a3i Precision is graded independently](793a_product-spec.md#793a3i-precision-is-graded-independently) samples 30 notes, which hides the grader's coverage.
- **Fix to be applied:** [793a3i Precision is graded independently](793a_product-spec.md#793a3i-precision-is-graded-independently): grade every [793a3a Evaluate every completed turn](793a_product-spec.md#793a3a-evaluate-every-completed-turn) capture independently, and log the verdicts, the sample size and the precision.
- **Status:** accepted, not yet applied. The second review of commit 058c70b3 found it; the orchestrator accepted it as a fix to make without asking Joe.

## 793b34 M7: a scenario stops asserting "context"

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Finding:** [793a2j1a Delivered notes equal the depth's output](793a_product-spec.md#793a2j1-uat-scenarios) asserts on "context", which an end user cannot observe ([793b16 Observable scenarios](793b_decision-log.md#793b16-observable-scenarios)).
- **Fix to be applied:** [793a2j1a Delivered notes equal the depth's output](793a_product-spec.md#793a2j1-uat-scenarios) stops asserting "context" and asserts what the transcript shows.
- **Status:** accepted, not yet applied. The second review of commit 058c70b3 found it; the orchestrator accepted it as a fix to make without asking Joe.

## 793b35 M8: "Reinforced" is not backed by a MUST

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Finding:** [793a3c1a Existing note already covers the lesson](793a_product-spec.md#793a3c1-uat-scenarios) says an existing note is "reinforced", but [793a3c No duplicates](793a_product-spec.md#793a3c-no-duplicates) has no requirement for it.
- **Fix to be applied:** Either [793a3c No duplicates](793a_product-spec.md#793a3c-no-duplicates) gets a MUST for reinforcement, or the word is dropped from [793a3c1a Existing note already covers the lesson](793a_product-spec.md#793a3c1-uat-scenarios). The orchestrator recommends dropping it from the scenario unless Joe wants a MUST.
- **Status:** accepted, not yet applied. The second review of commit 058c70b3 found it; the orchestrator accepted it as a fix to make without asking Joe.

## 793b36 L1: scenario Given refers to the recent-activity channel

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Finding:** [793a2h1d A recently surfaced standard uses deep](793a_product-spec.md#793a2h1-uat-scenarios)'s Given does not name the recent-activity channel the standard comes from.
- **Fix to be applied:** [793a2h1d A recently surfaced standard uses deep](793a_product-spec.md#793a2h1-uat-scenarios)'s Given refers to the recent-activity channel.
- **Status:** accepted, not yet applied. The second review of commit 058c70b3 found it; the orchestrator accepted it as a fix to make without asking Joe.

## 793b37 L2: next response or tool call

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Finding:** [793a2b Recall after each tool result](793a_product-spec.md#793a2b-recall-after-each-tool-result) says "next response", but a tool call can come first.
- **Fix to be applied:** [793a2b Recall after each tool result](793a_product-spec.md#793a2b-recall-after-each-tool-result) says "next response or tool call".
- **Status:** accepted, not yet applied. The second review of commit 058c70b3 found it; the orchestrator accepted it as a fix to make without asking Joe.

## 793b38 L3: drop "Added:" from link labels

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Finding:** Some link labels carry the delta-operation word "Added:", which docs/AGENTS.md "References to Addressable Content" forbids.
- **Fix to be applied:** Drop "Added:" from those link labels.
- **Status:** accepted, not yet applied. The second review of commit 058c70b3 found it; the orchestrator accepted it as a fix to make without asking Joe.

## 793b39 L4: rationale states the why only

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Finding:** Rationale text in [793a3h Capture rate exceeds the baseline](793a_product-spec.md#793a3h-capture-rate-exceeds-the-baseline) and the #846 mention in [793a3a Evaluate every completed turn](793a_product-spec.md#793a3a-evaluate-every-completed-turn) describe history instead of the why, and "(formerly REC-n)" notes remain.
- **Fix to be applied:** Rationale states the why only in [793a3h Capture rate exceeds the baseline](793a_product-spec.md#793a3h-capture-rate-exceeds-the-baseline) and in [793a3a Evaluate every completed turn](793a_product-spec.md#793a3a-evaluate-every-completed-turn) (the #846 mention); strip the "(formerly REC-n)" notes at promotion.
- **Status:** accepted, not yet applied. The second review of commit 058c70b3 found it; the orchestrator accepted it as a fix to make without asking Joe.

## 793b40 L5: correct the ID map

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Finding:** The [793b14 ID handling](793b_decision-log.md#793b14-id-handling) ID map calls [793a2e1a Note delivered earlier is not delivered again](793a_product-spec.md#793a2e1-uat-scenarios) new though it was rewritten, and omits material modifications.
- **Fix to be applied:** Correct the [793b14 ID handling](793b_decision-log.md#793b14-id-handling) map: [793a2e1a Note delivered earlier is not delivered again](793a_product-spec.md#793a2e1-uat-scenarios) was rewritten, not new; name [793a2e Relevance and no re-injection](793a_product-spec.md#793a2e-relevance-and-no-re-injection) and [793a4d Sensitive content](793a_product-spec.md#793a4d-sensitive-content) as material modifications; list the scenarios modified in place.
- **Status:** accepted, not yet applied. The second review of commit 058c70b3 found it; the orchestrator accepted it as a fix to make without asking Joe.

## 793b41 L6: a scenario for a discarded capture

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Finding:** [793a5e Auditability and measurability](793a_product-spec.md#793a5e-auditability-and-measurability) has no scenario for a capture the System discards.
- **Fix to be applied:** Add a [793a5e Auditability and measurability](793a_product-spec.md#793a5e-auditability-and-measurability) scenario for a discarded capture.
- **Status:** accepted, not yet applied. The second review of commit 058c70b3 found it; the orchestrator accepted it as a fix to make without asking Joe.

## 793b42 L7: trim 793a2j1b

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Finding:** [793a2j1b A vault holding more than 20 notes about the topic](793a_product-spec.md#793a2j1-uat-scenarios) asserts more than its requirement states.
- **Fix to be applied:** Trim [793a2j1b A vault holding more than 20 notes about the topic](793a_product-spec.md#793a2j1-uat-scenarios).
- **Status:** accepted, not yet applied. The second review of commit 058c70b3 found it; the orchestrator accepted it as a fix to make without asking Joe.

## 793b43 Author's choice accepted: log depth, phrases and escalation

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Choice:** the reviewer accepted the author's choice to log depth and phrases ([793b8 Per-moment record sits in the existing log requirement](793b_decision-log.md#793b8-per-moment-record-sits-in-the-existing-log-requirement)), extended to each query's phrases and whether glance escalated.
- **Rationale:** the depth scenarios stay checkable from the operator-accessible log.
- **Status:** accepted; the extension is not yet applied.

## 793b44 Author's choice accepted: the rate/precision split

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Choice:** the reviewer accepted splitting 793a3e into [793a3h Capture rate exceeds the baseline](793a_product-spec.md#793a3h-capture-rate-exceeds-the-baseline) and [793a3i Precision is graded independently](793a_product-spec.md#793a3i-precision-is-graded-independently) ([793b9 Capture rate and precision are separate requirements](793b_decision-log.md#793b9-capture-rate-and-precision-are-separate-requirements)).
- **Rationale:** each gets one statement and its own scenarios.
- **Status:** accepted.

## 793b45 Author's choice accepted: a conditional recommendation check

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Choice:** the reviewer accepted the conditional recommendation check ([792a14o An emergent recommendation is checked against the vault](../../specs/product/recall.md#792a14o-an-emergent-recommendation-is-checked-against-the-vault)), worded as the H2 fix says.
- **Rationale:** the check runs only when a recommendation is emergent, so counting it unconditionally overstates the output.
- **Status:** accepted, not yet applied (see the H2 entry).
