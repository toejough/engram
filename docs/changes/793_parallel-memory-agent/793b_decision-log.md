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
| 4.1 Automatic recall | [793a2 automatic-memory (automatic recall)](793a_product-spec.md#793a2-added-automatic-memory-automatic-recall) | REC-1..REC-5 to [793a2a Recall at a human prompt](793a_product-spec.md#793a2a-recall-at-a-human-prompt)..[793a2e No re-injection](793a_product-spec.md#793a2e-no-re-injection); REC-6 and REC-7 to [793a2j Volume equals the depth's output](793a_product-spec.md#793a2j-volume-equals-the-depths-output) and [793a2i Recall for every prompt regardless of source](793a_product-spec.md#793a2i-recall-for-every-prompt-regardless-of-source) (new IDs); [793a2h Recall depth per moment](793a_product-spec.md#793a2h-recall-depth-per-moment) was introduced in an earlier revision, modified since |
| 4.2 Automatic capture | [793a3 automatic-memory (automatic capture)](793a_product-spec.md#793a3-added-automatic-memory-automatic-capture) | CAP-1..CAP-4 and CAP-6 to [793a3a Evaluate every completed turn](793a_product-spec.md#793a3a-evaluate-every-completed-turn)..[793a3d Provenance](793a_product-spec.md#793a3d-provenance) and [793a3f Capture does not delay the session](793a_product-spec.md#793a3f-capture-does-not-delay-the-session); CAP-5 to [793a3h Capture rate exceeds the baseline](793a_product-spec.md#793a3h-capture-rate-exceeds-the-baseline) and [793a3i Precision is graded independently](793a_product-spec.md#793a3i-precision-is-graded-independently) (new IDs); [793a3g Deep recall's writes are the System's](793a_product-spec.md#793a3g-deep-recalls-writes-are-the-systems) was introduced in an earlier revision, modified since; [793a3j Direct writes go to the local vault](793a_product-spec.md#793a3j-direct-writes-go-to-the-local-vault) [793a3k Used notes are activated](793a_product-spec.md#793a3k-used-notes-are-activated) and [793a3l Activation accuracy is graded independently](793a_product-spec.md#793a3l-activation-accuracy-is-graded-independently) are new |
| 4.3 Operator control and safety | [793a4 automatic-memory (operator control and safety)](793a_product-spec.md#793a4-added-automatic-memory-operator-control-and-safety) | CTL-2..CTL-4 to [793a4b Failure isolation](793a_product-spec.md#793a4b-failure-isolation)..[793a4d Sensitive content](793a_product-spec.md#793a4d-sensitive-content); CTL-1 (operating modes) was retired with `793a4e` ([793b49 No operating modes](#793b49-no-operating-modes)); [793a4f No recall or capture in its own evaluation sessions](793a_product-spec.md#793a4f-no-recall-or-capture-in-its-own-evaluation-sessions) was introduced in an earlier revision, modified since; [793a4g Active whenever installed](793a_product-spec.md#793a4g-active-whenever-installed) and [793a4h Installed by engram update](793a_product-spec.md#793a4h-installed-by-engram-update) are new |
| 4.4 Independence, coexistence, measurement | [793a5 automatic-memory (independence, coexistence, and measurement)](793a_product-spec.md#793a5-added-automatic-memory-independence-coexistence-and-measurement) | IND-1..IND-3 to [793a5a Independence from agent compliance](793a_product-spec.md#793a5a-independence-from-agent-compliance)..[793a5c Environments without the capability](793a_product-spec.md#793a5c-environments-without-the-capability); NFR-2 and NFR-3 to [793a5e Auditability and measurability](793a_product-spec.md#793a5e-auditability-and-measurability) and [793a5f Effect on outcomes](793a_product-spec.md#793a5f-effect-on-outcomes); NFR-1 to [793a5g Latency per recall moment](793a_product-spec.md#793a5g-latency-per-recall-moment) (new ID) |

- **Rationale:** unchanged requirements keep "(formerly X)" so the draft's review comments still map.
  Cross-references inside requirements and scenarios were rewritten to the current IDs.

## 793b3 Scenarios added for uncovered MUST branches

- **Decided by:** Author (agent); awaiting owner validation.

- **Choice:** every MUST and MUST NOT branch has a scenario. Added in this revision: the in-context
  branches of [793a2e No re-injection](793a_product-spec.md#793a2e-no-re-injection) (retrieved by the agent, parent versus subagent, compaction, truncation,
  resume), the depth branches of [793a2h Recall depth per moment](793a_product-spec.md#793a2h-recall-depth-per-moment), the direct-write branches of [793a3j Direct writes go to the local vault](793a_product-spec.md#793a3j-direct-writes-go-to-the-local-vault), the concurrent
  and parent-reported duplicates of [793a3c No duplicates](793a_product-spec.md#793a3c-no-duplicates), the routine-success exclusion of [793a3b Confirmed lessons only](793a_product-spec.md#793a3b-confirmed-lessons-only), the
  no-error-to-the-agent branch of [793a4b Failure isolation](793a_product-spec.md#793a4b-failure-isolation), and a scenario per mode and per measurement branch of
  `793a4e Operating modes and measurement`.
- **Rationale:** the product spec guide requires coverage for every distinct mandatory branch.
- **Alternatives:** one scenario per requirement. Rejected: the guide calls that insufficient.
- **Status:** the per-mode scenarios are retired with `793a4e`; see [793b49 No operating modes](#793b49-no-operating-modes).

## 793b4 Draft sections that are not requirements

- **Decided by:** Author (agent); awaiting owner validation. The widening of the System's Lesson definition mentioned here is Owner (Joe), [793c7 Review answers (2026-10-11)](793c_open-questions.md#793c7-review-answers-2026-10-11) item 9 (see [793b17 Lesson is wider than learn's definition](#793b17-lesson-is-wider-than-learns-definition)).

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
| Past noise and over-firing concerns | [793a2e No re-injection](793a_product-spec.md#793a2e-no-re-injection), [793a2h Recall depth per moment](793a_product-spec.md#793a2h-recall-depth-per-moment), [793a2i Recall for every prompt regardless of source](793a_product-spec.md#793a2i-recall-for-every-prompt-regardless-of-source), [793a2j Volume equals the depth's output](793a_product-spec.md#793a2j-volume-equals-the-depths-output), [793a5g Latency per recall moment](793a_product-spec.md#793a5g-latency-per-recall-moment) |
| Safety of an unattended process | [793a4b Failure isolation](793a_product-spec.md#793a4b-failure-isolation), [793a4c Memory purview only](793a_product-spec.md#793a4c-memory-purview-only), [793a4d Sensitive content](793a_product-spec.md#793a4d-sensitive-content), [793a4g Active whenever installed](793a_product-spec.md#793a4g-active-whenever-installed), [793a4f No recall or capture in its own evaluation sessions](793a_product-spec.md#793a4f-no-recall-or-capture-in-its-own-evaluation-sessions), [793a5e Auditability and measurability](793a_product-spec.md#793a5e-auditability-and-measurability) |

## 793b6 No proposed values remain

- **Decided by:** Owner (Joe), [793c7 Review answers (2026-10-11)](793c_open-questions.md#793c7-review-answers-2026-10-11) item 12, accepted N = 20 and the removal of the proposed marker. Author (agent), removal of the refusal-rate floor follows [793b15 Attributable presentation only](#793b15-attributable-presentation-only); awaiting owner validation.

- **Choice:** [793a5f Effect on outcomes](793a_product-spec.md#793a5f-effect-on-outcomes) keeps N = 20 runs per arm, accepted by Joe, and loses its proposed marker. The
  95% / N=20 refusal-rate floor was removed with the refusal check ([793b15 Attributable presentation only](#793b15-attributable-presentation-only)).
- **Rationale:** the guide forbids exceeding the behavior the user specified, and Joe accepted the one
  remaining value.

## 793b7 Volume and latency equal the chosen depth's output

- **Decided by:** Owner (Joe), [793c2 Are the volume bounds and latency budget accepted?](793c_open-questions.md#793c2-are-the-volume-bounds-and-latency-budget-accepted) and [793c7 Review answers (2026-10-11)](793c_open-questions.md#793c7-review-answers-2026-10-11) items 1 to 4, decided that bounds equal recall's output, drop the per-turn and per-session bounds, apply at tool-result moments, and cap latency at the depth's query latency. Author (agent), interpreting [793c7 Review answers (2026-10-11)](793c_open-questions.md#793c7-review-answers-2026-10-11), chose to make the bounds reference the current query instead of fixed numbers and chose the recall-query citations; awaiting owner validation.

- **Choice:** [793a2j Volume equals the depth's output](793a_product-spec.md#793a2j-volume-equals-the-depths-output) sets the injected volume at each recall moment to what that depth's queries
  return (glance: one query with about 3 phrases; deep: one query with 10 phrases; at either depth the
  recommendation check, which runs only when a recommendation emerges; both depths' queries when glance
  escalates), less items already in context. That output
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
- **Status:** the recommendation-check wording follows [793b25 H2: drop the double count of the re-entry query](#793b25-h2-drop-the-double-count-of-the-re-entry-query). The latency clause is replaced by [793b48 Latency covers the instruction path's full lookup](#793b48-latency-covers-the-instruction-paths-full-lookup), and the injected set now includes incidental items ([793b46 Inject what recall delivers today](#793b46-inject-what-recall-delivers-today)).

## 793b8 Per-moment record sits in the existing log requirement

- **Decided by:** Owner (Joe), [793c7 Review answers (2026-10-11)](793c_open-questions.md#793c7-review-answers-2026-10-11) item 4, decided that per-moment volume and latency are recorded. Author (agent), interpreting [793c7 Review answers (2026-10-11)](793c_open-questions.md#793c7-review-answers-2026-10-11) item 4, chose to log the depth and the phrases queried and to place the record in the existing requirement; awaiting owner validation.

- **Choice:** the volume and latency record is part of [793a5e Auditability and measurability](793a_product-spec.md#793a5e-auditability-and-measurability), with scenarios [793a5e1b Volume and latency recorded per moment](793a_product-spec.md#793a5e1-uat-scenarios) and
  [793a5e1c A moment with no injection is recorded](793a_product-spec.md#793a5e1-uat-scenarios), not a new requirement. The log is available by session id, each entry carries session
  id, turn id and moment, and a moment with no injection is recorded with volume 0.
- **Choice:** the log also records the depth and the phrases queried, so that [793a2h1a A routine decision uses glance](793a_product-spec.md#793a2h1-uat-scenarios) and
  [793a2j1a Delivered items equal the depth's output](793a_product-spec.md#793a2j1-uat-scenarios) can be checked from the operator-accessible log. This goes beyond Joe's wording and is
  raised in [793c7 Review answers (2026-10-11)](793c_open-questions.md#793c7-review-answers-2026-10-11).
- **Rationale:** it is one more field of the decision log that requirement already mandates.

## 793b9 Capture rate and precision are separate requirements

- **Decided by:** Owner (Joe), [793c1 What are the capture precision bar and capture-rate target?](793c_open-questions.md#793c1-what-are-the-capture-precision-bar-and-capture-rate-target) and [793c7 Review answers (2026-10-11)](793c_open-questions.md#793c7-review-answers-2026-10-11) items 7 and 8, decided that the rate must beat the baseline and that a separate grader reports precision with its sample size. Author (agent), interpreting [793c7 Review answers (2026-10-11)](793c_open-questions.md#793c7-review-answers-2026-10-11) item 8, chose the comparator 5.6% with no margin, and the split of `793a3e` into a rate requirement and a precision requirement; awaiting owner validation.

- **Choice:** [793a3h Capture rate exceeds the baseline](793a_product-spec.md#793a3h-capture-rate-exceeds-the-baseline) requires the capture rate to exceed 5.6% on the point estimate, with no
  margin; the 60-moment labeled set is kept from the draft. [793a3i Precision is graded independently](793a_product-spec.md#793a3i-precision-is-graded-independently) requires a separate grader inside
  the System, independent of the capture judgment, and reports precision with its sample size; the
  30-note sample is kept. Neither has a precision threshold.
- **Rationale:** 5.6% is the higher of the two measured baselines (4/71 = 5.6%, 3/66 = 4.5%; LEDGER
  `739-audit-w2-learn-fired`, `learn-rate-remeasure-w2`). Splitting rate from precision gives each one
  statement and its own scenarios; the grader keeps the capture judge from marking its own work.

## 793b10 Default mode and measurement tied to mode

- **Decided by:** Owner (Joe), [793c3 What is the default mode for a fresh install?](793c_open-questions.md#793c3-what-is-the-default-mode-for-a-fresh-install) and [793c7 Review answers (2026-10-11)](793c_open-questions.md#793c7-review-answers-2026-10-11) item 6, decided the active default and that measurement is not switchable apart from the mode. Author (agent) chose the scenario set and the "off records nothing" wording; awaiting owner validation.

- **Choice:** `793a4e Operating modes and measurement` makes the default MUST active, and requires measurement whenever the mode is
  observe or active, not switchable off independently. Off records nothing, so scenario `793a4e1b Off mode is inert` asserts
  no log entry. Scenario `793a4e1e Measurement cannot be switched off apart from the mode` covers an attempt to switch measurement off apart from the mode.
- **Rationale:** an off System has nothing to measure; observing and active both produce decisions to
  record.
- **Status:** superseded by [793b49 No operating modes](#793b49-no-operating-modes); `793a4e` is retired.

## 793b11 Direct capture: where it is stated

- **Decided by:** Owner (Joe), [793c4 Are captured lessons written directly, or held for `curate` review?](793c_open-questions.md#793c4-are-captured-lessons-written-directly-or-held-for-curate-review), decided that captured lessons are written directly. Author (agent), interpreting [793c4 Are captured lessons written directly, or held for `curate` review?](793c_open-questions.md#793c4-are-captured-lessons-written-directly-or-held-for-curate-review), chose where the rule is stated and which assertions the scenarios make; awaiting owner validation.

- **Choice:** the direct-write rule and the System-judges rule are stated in [793a3a Evaluate every completed turn](793a_product-spec.md#793a3a-evaluate-every-completed-turn). Scenarios
  [793a3a1a User correction is captured without agent action](793a_product-spec.md#793a3a1-uat-scenarios) and [793a3a1b Explicit save-request](793a_product-spec.md#793a3a1-uat-scenarios) assert the vault, not a dry-run log. [793a3a1a User correction is captured without agent action](793a_product-spec.md#793a3a1-uat-scenarios) also asserts, from the
  transcript, that the agent invoked no learn command and saw no request to evaluate or write.
  Observe-mode logging remains in `793a4e1a Observe mode has no side effects`.
- **Rationale:** those assertions are what "direct, without polluting the main agent's context" means
  to an end user.
- **Status:** the observe-mode clause is superseded by [793b49 No operating modes](#793b49-no-operating-modes).

## 793b12 Recall for every prompt

- **Decided by:** Owner (Joe), [793c5 Does 793a2i Recall for every prompt regardless of source keep human-originated prompts as the default?](793c_open-questions.md#793c5-does-793a2i-recall-for-every-prompt-regardless-of-source-keep-human-originated-prompts-as-the-default), decided that recall applies to every prompt. Author (agent), interpreting [793c5 Does 793a2i Recall for every prompt regardless of source keep human-originated prompts as the default?](793c_open-questions.md#793c5-does-793a2i-recall-for-every-prompt-regardless-of-source-keep-human-originated-prompts-as-the-default) and [793c7 Review answers (2026-10-11)](793c_open-questions.md#793c7-review-answers-2026-10-11) item 5, chose to keep the re-injection MUST NOT in [793a2e No re-injection](793a_product-spec.md#793a2e-no-re-injection) and to use a Scenario Outline; awaiting owner validation.

- **Choice:** [793a2i Recall for every prompt regardless of source](793a_product-spec.md#793a2i-recall-for-every-prompt-regardless-of-source) replaces the retired `793a2g` and keeps only the source-independence MUST. The
  in-context MUST NOT moved to [793a2e No re-injection](793a_product-spec.md#793a2e-no-re-injection), which now owns "never inject a note already in the receiving
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
| `793a2f`, `793a2f1a`..`793a2f1c` | [793a2j Volume equals the depth's output](793a_product-spec.md#793a2j-volume-equals-the-depths-output), [793a2j1a Delivered items equal the depth's output](793a_product-spec.md#793a2j1-uat-scenarios), [793a2j1b A vault holding more than 20 notes about the topic](793a_product-spec.md#793a2j1-uat-scenarios) | the bound reversed from a ceiling to a floor, then to depth output as floor and ceiling |
| `793a2g` | [793a2i Recall for every prompt regardless of source](793a_product-spec.md#793a2i-recall-for-every-prompt-regardless-of-source) (source independence) and [793a2e No re-injection](793a_product-spec.md#793a2e-no-re-injection) (in-context rule) | "human only" reversed to "every prompt" |
| `793a2g1a` | [793a2i1a Non-human prompt receives recall](793a_product-spec.md#793a2i1-uat-scenarios) | skip reversed to inject |
| `793a2g1b`, `793a2g1c` | [793a2e1a Note delivered earlier is not delivered again](793a_product-spec.md#793a2e1-uat-scenarios), [793a2e1c Note the agent retrieved itself is not injected](793a_product-spec.md#793a2e1-uat-scenarios) | the in-context rule moved to its owner |
| `793a3e`, `793a3e1a`, `793a3e1b` | [793a3h Capture rate exceeds the baseline](793a_product-spec.md#793a3h-capture-rate-exceeds-the-baseline) and [793a3i Precision is graded independently](793a_product-spec.md#793a3i-precision-is-graded-independently), [793a3i1a Precision is reported with the number graded](793a_product-spec.md#793a3i1-uat-scenarios), [793a3h1a Capture rate exceeds the baseline](793a_product-spec.md#793a3h1-uat-scenarios) | a 70% / 50% bar reversed to "beat the baseline, record precision" |
| `793a4a`, `793a4a1a`..`793a4a1c` | `793a4e Operating modes and measurement`, `793a4e1a Observe mode has no side effects`, `793a4e1b Off mode is inert`, `793a4e1c Fresh install is active and measured` | the default reversed from observe to active |
| `793a5d`, `793a5d1a` | [793a5g Latency per recall moment](793a_product-spec.md#793a5g-latency-per-recall-moment), [793a5g1a Added latency stays within the instruction path's lookup latency](793a_product-spec.md#793a5g1-uat-scenarios) | the latency ceiling reversed to a floor, then to the depth's query latency |
| `793a4e`, `793a4e1a`..`793a4e1e` | [793a4g Active whenever installed](793a_product-spec.md#793a4g-active-whenever-installed), [793a4g1a An Integration installed by update is active](793a_product-spec.md#793a4g1-uat-scenarios), `793a4g1b A disabled integration is inert`, [793a4g1c An uninstalled Integration is inert](793a_product-spec.md#793a4g1-uat-scenarios); the measurement rule moved into [793a5e Auditability and measurability](793a_product-spec.md#793a5e-auditability-and-measurability) and [793a5e1e Measurement cannot be switched off while the System is active](793a_product-spec.md#793a5e1-uat-scenarios) | the operating modes were removed ([793b49 No operating modes](#793b49-no-operating-modes)) |
| `793a2a1b` | [793a2a1c Only incidental items come back](793a_product-spec.md#793a2a1-uat-scenarios) | "no relevant note, nothing delivered" reversed to "deliver the depth's whole output" ([793b46 Inject what recall delivers today](#793b46-inject-what-recall-delivers-today)) |
| `793a2e1f` | [793a2e1i A chunk stub already shown is not injected again](793a_product-spec.md#793a2e1-uat-scenarios) ([793b62 Chunks are injected lazily, as stubs](#793b62-chunks-are-injected-lazily-as-stubs)); [793a2e1h A chunk whose content the agent fetched is not injected](793a_product-spec.md#793a2e1-uat-scenarios) is rewritten around a fetched chunk | the System's queries are lazy, so it never delivers a chunk's content |
| `793a4g1b` | none; [793a4g1a An Integration installed by update is active](793a_product-spec.md#793a4g1-uat-scenarios) and [793a4g1c An uninstalled Integration is inert](793a_product-spec.md#793a4g1-uat-scenarios) cover on and off | the disable mechanism was dropped; the System is turned off by uninstalling ([793b65 Installation is part of this spec](#793b65-installation-is-part-of-this-spec)) |
| `793a2e1b` | none; [793a2j Volume equals the depth's output](793a_product-spec.md#793a2j-volume-equals-the-depths-output) and [793a2a1c Only incidental items come back](793a_product-spec.md#793a2a1-uat-scenarios) cover the case | the relevance threshold was removed, so "irrelevant note suppressed" no longer holds |

- **Choice:** requirements modified in place keep their IDs: [793a2a Recall at a human prompt](793a_product-spec.md#793a2a-recall-at-a-human-prompt), [793a2b Recall after each tool result](793a_product-spec.md#793a2b-recall-after-each-tool-result), [793a2c Recall at subagent dispatch](793a_product-spec.md#793a2c-recall-at-subagent-dispatch), [793a2d Attributable presentation](793a_product-spec.md#793a2d-attributable-presentation), [793a2e No re-injection](793a_product-spec.md#793a2e-no-re-injection), [793a2h Recall depth per moment](793a_product-spec.md#793a2h-recall-depth-per-moment), [793a2j Volume equals the depth's output](793a_product-spec.md#793a2j-volume-equals-the-depths-output), [793a3a Evaluate every completed turn](793a_product-spec.md#793a3a-evaluate-every-completed-turn), [793a3b Confirmed lessons only](793a_product-spec.md#793a3b-confirmed-lessons-only), [793a3c No duplicates](793a_product-spec.md#793a3c-no-duplicates), [793a3d Provenance](793a_product-spec.md#793a3d-provenance), [793a3f Capture does not delay the session](793a_product-spec.md#793a3f-capture-does-not-delay-the-session), [793a3g Deep recall's writes are the System's](793a_product-spec.md#793a3g-deep-recalls-writes-are-the-systems), [793a3h Capture rate exceeds the baseline](793a_product-spec.md#793a3h-capture-rate-exceeds-the-baseline), [793a3i Precision is graded independently](793a_product-spec.md#793a3i-precision-is-graded-independently), [793a4b Failure isolation](793a_product-spec.md#793a4b-failure-isolation), [793a4c Memory purview only](793a_product-spec.md#793a4c-memory-purview-only), [793a4d Sensitive content](793a_product-spec.md#793a4d-sensitive-content), [793a4f No recall or capture in its own evaluation sessions](793a_product-spec.md#793a4f-no-recall-or-capture-in-its-own-evaluation-sessions), [793a5b Coexistence without double-firing](793a_product-spec.md#793a5b-coexistence-without-double-firing), [793a5e Auditability and measurability](793a_product-spec.md#793a5e-auditability-and-measurability), [793a5f Effect on outcomes](793a_product-spec.md#793a5f-effect-on-outcomes), [793a5g Latency per recall moment](793a_product-spec.md#793a5g-latency-per-recall-moment). Retitled in place: [793a2d Attributable presentation](793a_product-spec.md#793a2d-attributable-presentation), [793a2e No re-injection](793a_product-spec.md#793a2e-no-re-injection), [793a3f Capture does not delay the session](793a_product-spec.md#793a3f-capture-does-not-delay-the-session). Materially modified: [793a2e No re-injection](793a_product-spec.md#793a2e-no-re-injection) (the relevance threshold removed, chunks covered) and [793a4d Sensitive content](793a_product-spec.md#793a4d-sensitive-content) (secrets made a MUST NOT, per [793c7 Review answers (2026-10-11)](793c_open-questions.md#793c7-review-answers-2026-10-11) item 10).
- **Choice:** scenarios modified in place keep their IDs: [793a2e1a Note delivered earlier is not delivered again](793a_product-spec.md#793a2e1-uat-scenarios), [793a2h1c An unresolved gap uses deep](793a_product-spec.md#793a2h1-uat-scenarios), [793a2h1d A recently surfaced standard uses deep](793a_product-spec.md#793a2h1-uat-scenarios), [793a2j1a Delivered items equal the depth's output](793a_product-spec.md#793a2j1-uat-scenarios), [793a2j1b A vault holding more than 20 notes about the topic](793a_product-spec.md#793a2j1-uat-scenarios), [793a3c1a Existing note already covers the lesson](793a_product-spec.md#793a3c1-uat-scenarios), [793a3g1a A coverage write is made without the working agent](793a_product-spec.md#793a3g1-uat-scenarios), [793a3g1b A sound conclusion is persisted without the working agent](793a_product-spec.md#793a3g1-uat-scenarios), [793a3i1a Precision is reported with the number graded](793a_product-spec.md#793a3i1-uat-scenarios), [793a4c1a Tool calls and permissions unchanged](793a_product-spec.md#793a4c1-uat-scenarios), [793a4c1b The prompt is unchanged and the note is separate](793a_product-spec.md#793a4c1-uat-scenarios), [793a4f1a Evaluation sessions are left alone](793a_product-spec.md#793a4f1-uat-scenarios), [793a5b1a Both paths active](793a_product-spec.md#793a5b1-uat-scenarios), [793a5f1a Treatment beats control](793a_product-spec.md#793a5f1-uat-scenarios), [793a5g1a Added latency stays within the instruction path's lookup latency](793a_product-spec.md#793a5g1-uat-scenarios). [793a2e1a Note delivered earlier is not delivered again](793a_product-spec.md#793a2e1-uat-scenarios) was rewritten, not added.
- **Choice:** new nodes: [793a3j Direct writes go to the local vault](793a_product-spec.md#793a3j-direct-writes-go-to-the-local-vault); introduced in an earlier revision, modified since: [793a2h Recall depth per moment](793a_product-spec.md#793a2h-recall-depth-per-moment), [793a3g Deep recall's writes are the System's](793a_product-spec.md#793a3g-deep-recalls-writes-are-the-systems), [793a4f No recall or capture in its own evaluation sessions](793a_product-spec.md#793a4f-no-recall-or-capture-in-its-own-evaluation-sessions); plus the scenarios named in [793b3 Scenarios added for uncovered MUST branches](#793b3-scenarios-added-for-uncovered-must-branches), and from this revision: [793a3k Used notes are activated](793a_product-spec.md#793a3k-used-notes-are-activated), [793a3k1a A note the transcript shows used is activated](793a_product-spec.md#793a3k1-uat-scenarios), [793a3k1b An injected note with no evidence of use is not activated](793a_product-spec.md#793a3k1-uat-scenarios), [793a3k1c An injected chunk is not activated](793a_product-spec.md#793a3k1-uat-scenarios), [793a3k1d A used parent note is pulled down and left pending](793a_product-spec.md#793a3k1-uat-scenarios), [793a4g Active whenever installed](793a_product-spec.md#793a4g-active-whenever-installed), [793a4g1a An Integration installed by update is active](793a_product-spec.md#793a4g1-uat-scenarios), `793a4g1b A disabled integration is inert`, [793a4g1c An uninstalled Integration is inert](793a_product-spec.md#793a4g1-uat-scenarios), [793a4f1b The session that chooses depth and phrases is left alone](793a_product-spec.md#793a4f1-uat-scenarios), [793a2a1c Only incidental items come back](793a_product-spec.md#793a2a1-uat-scenarios), [793a2e1h A chunk whose content the agent fetched is not injected](793a_product-spec.md#793a2e1-uat-scenarios), [793a2j1c Incidental items are delivered with the direct matches](793a_product-spec.md#793a2j1-uat-scenarios), [793a2j1d An escalated moment delivers both depths' output](793a_product-spec.md#793a2j1-uat-scenarios), [793a5e1d A discarded capture is recorded with its reason](793a_product-spec.md#793a5e1-uat-scenarios), [793a5e1e Measurement cannot be switched off while the System is active](793a_product-spec.md#793a5e1-uat-scenarios), [793a5e1f An escalated moment records both depths](793a_product-spec.md#793a5e1-uat-scenarios), [793a5g1b An escalated moment is compared with both depths](793a_product-spec.md#793a5g1-uat-scenarios), [793a4c1c The only effect on the session is the injected items](793a_product-spec.md#793a4c1-uat-scenarios), [793a4c1d Effects outside the session are vault notes and the log](793a_product-spec.md#793a4c1-uat-scenarios). The earlier log called `793a2g`'s change "Modified"; it was a replacement and is corrected here. These IDs are change-internal and never integrated.
- **Choice:** from the answers to [793c15 Does the System inject chunks with their content?](793c_open-questions.md#793c15-does-the-system-inject-chunks-with-their-content) to [793c22 There is no uninstall command today](793c_open-questions.md#793c22-there-is-no-uninstall-command-today): requirements modified in place are [793a2d Attributable presentation](793a_product-spec.md#793a2d-attributable-presentation), [793a2e No re-injection](793a_product-spec.md#793a2e-no-re-injection), [793a2j Volume equals the depth's output](793a_product-spec.md#793a2j-volume-equals-the-depths-output), [793a3g Deep recall's writes are the System's](793a_product-spec.md#793a3g-deep-recalls-writes-are-the-systems), [793a3k Used notes are activated](793a_product-spec.md#793a3k-used-notes-are-activated), [793a4g Active whenever installed](793a_product-spec.md#793a4g-active-whenever-installed), [793a4f No recall or capture in its own evaluation sessions](793a_product-spec.md#793a4f-no-recall-or-capture-in-its-own-evaluation-sessions), [793a5e Auditability and measurability](793a_product-spec.md#793a5e-auditability-and-measurability) and [793a5g Latency per recall moment](793a_product-spec.md#793a5g-latency-per-recall-moment); scenarios modified in place are [793a2a1c Only incidental items come back](793a_product-spec.md#793a2a1-uat-scenarios), [793a2d1a Injected content names its source](793a_product-spec.md#793a2d1-uat-scenarios), [793a2e1h A chunk whose content the agent fetched is not injected](793a_product-spec.md#793a2e1-uat-scenarios) (its meaning changed from a chunk delivered with content to a chunk the agent fetched, and it was retitled), [793a2j1a Delivered items equal the depth's output](793a_product-spec.md#793a2j1-uat-scenarios), [793a2j1c Incidental items are delivered with the direct matches](793a_product-spec.md#793a2j1-uat-scenarios), [793a3k1a A note the transcript shows used is activated](793a_product-spec.md#793a3k1-uat-scenarios), [793a3k1b An injected note with no evidence of use is not activated](793a_product-spec.md#793a3k1-uat-scenarios), [793a4f1a Evaluation sessions are left alone](793a_product-spec.md#793a4f1-uat-scenarios), [793a4g1a An Integration installed by update is active](793a_product-spec.md#793a4g1-uat-scenarios), [793a4g1c An uninstalled Integration is inert](793a_product-spec.md#793a4g1-uat-scenarios), [793a5e1e Measurement cannot be switched off while the System is active](793a_product-spec.md#793a5e1-uat-scenarios), [793a5f1a Treatment beats control](793a_product-spec.md#793a5f1-uat-scenarios), [793a5g1a Added latency stays within the instruction path's lookup latency](793a_product-spec.md#793a5g1-uat-scenarios) and [793a5g1b An escalated moment is compared with both depths](793a_product-spec.md#793a5g1-uat-scenarios). New nodes are [793a2d1b Injected chunk names its id](793a_product-spec.md#793a2d1-uat-scenarios), [793a2e1i A chunk stub already shown is not injected again](793a_product-spec.md#793a2e1-uat-scenarios), [793a2j1e Chunks are delivered as stubs](793a_product-spec.md#793a2j1-uat-scenarios), [793a3g1c A covered cluster gets evidence and its note is injected](793a_product-spec.md#793a3g1-uat-scenarios), [793a3g1d A covering note already in context is not injected again](793a_product-spec.md#793a3g1-uat-scenarios), [793a3g1e A near cluster amends the note](793a_product-spec.md#793a3g1-uat-scenarios), [793a3k1e Recall's own activation step does not run](793a_product-spec.md#793a3k1-uat-scenarios), [793a3l Activation accuracy is graded independently](793a_product-spec.md#793a3l-activation-accuracy-is-graded-independently) with [793a3l1a A sampled activation verdict is graded](793a_product-spec.md#793a3l1-uat-scenarios) and [793a3l1b The grader does not defer to the activation judgment](793a_product-spec.md#793a3l1-uat-scenarios), and [793a4h Installed by engram update](793a_product-spec.md#793a4h-installed-by-engram-update) with [793a4h1a Update installs the Integration for the harness it updates](793a_product-spec.md#793a4h1-uat-scenarios), [793a4h1b An existing install gains the System on its next update](793a_product-spec.md#793a4h1-uat-scenarios) and [793a4h1c A preview installs none of the Integration](793a_product-spec.md#793a4h1-uat-scenarios). `793a2e1f` and `793a4g1b` are retired and never reused.
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
  definition in 793a ([793b17 Lesson is wider than learn's definition](#793b17-lesson-is-wider-than-learns-definition)), and (ii) filing #846 to widen learn's own [792a15e The four kinds of explicit lesson](../../specs/product/learn.md#792a15e-the-four-kinds-of-explicit-lesson) separately. Joe confirmed this split in the review answers.
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
- **Finding:** The re-entry query is [792a14o An emergent recommendation is checked against the vault](../../specs/product/recall.md#792a14o-an-emergent-recommendation-is-checked-against-the-vault)'s recommendation check. It runs at both depths, conditionally, so [793a2j Volume equals the depth's output](793a_product-spec.md#793a2j-volume-equals-the-depths-output) and [793b7 Volume and latency equal the chosen depth's output](#793b7-volume-and-latency-equal-the-chosen-depths-output) count it twice.
- **Fix to be applied:** Reword [793a2j Volume equals the depth's output](793a_product-spec.md#793a2j-volume-equals-the-depths-output) and [793b7 Volume and latency equal the chosen depth's output](#793b7-volume-and-latency-equal-the-chosen-depths-output) to list the conditional recommendation check once, at both depths, with no separate re-entry query.
- **Status:** applied in 5fd6712a. The second review of commit 058c70b3 found it; the orchestrator accepted it as a fix to make without asking Joe.

## 793b26 H3: lazy mode truncates only chunks, never notes

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Finding:** Lazy mode truncates only chunks. [793a2e No re-injection](793a_product-spec.md#793a2e-no-re-injection) treats a truncated note as not present, which cannot occur.
- **Fix to be applied:** [793a2e No re-injection](793a_product-spec.md#793a2e-no-re-injection) becomes "an item (note or chunk) shown without its content doesn't count as present"; no re-injection covers chunks too; `793a2e1f A chunk shown without content is delivered with it` is rewritten around a chunk.
- **Status:** applied in 5fd6712a. The second review of commit 058c70b3 found it; the orchestrator accepted it as a fix to make without asking Joe.

## 793b27 H4: an escalated moment covers both depths' queries

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Finding:** When glance escalates to deep, the moment runs the queries of both depths, but [793a2j Volume equals the depth's output](793a_product-spec.md#793a2j-volume-equals-the-depths-output) and [793a5g Latency per recall moment](793a_product-spec.md#793a5g-latency-per-recall-moment) bound only one.
- **Fix to be applied:** An escalated moment's volume and latency cover both depths' queries (the union). Add subagent dispatch to [793a5g1a Added latency stays within the instruction path's lookup latency](793a_product-spec.md#793a5g1-uat-scenarios).
- **Status:** applied in 5fd6712a. The second review of commit 058c70b3 found it; the orchestrator accepted it as a fix to make without asking Joe.

## 793b28 M1: observe mode logs the would-be volume

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Finding:** Observe mode injects nothing, so its log cannot show the volume that would have been injected.
- **Fix to be applied:** Observe mode logs the would-be volume, in bytes and item count (`793a4e Operating modes and measurement`, [793a5e Auditability and measurability](793a_product-spec.md#793a5e-auditability-and-measurability)).
- **Status:** superseded by [793c12 Are operating modes needed at all?](793c_open-questions.md#793c12-are-operating-modes-needed-at-all) (Joe, 2026-10-11): there are no modes, so there is no observe mode to log a would-be volume. Not applied; see [793b49 No operating modes](#793b49-no-operating-modes).

## 793b29 M2: the System's own sessions are identifiable

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Finding:** [793a4f No recall or capture in its own evaluation sessions](793a_product-spec.md#793a4f-no-recall-or-capture-in-its-own-evaluation-sessions) depends on the System recognizing its own sessions, but no requirement says how they are identified.
- **Fix to be applied:** [793a4f No recall or capture in its own evaluation sessions](793a_product-spec.md#793a4f-no-recall-or-capture-in-its-own-evaluation-sessions): the System logs the session ids it starts itself as System sessions. Add a scenario for the depth and phrase choice.
- **Status:** applied in 5fd6712a. The second review of commit 058c70b3 found it; the orchestrator accepted it as a fix to make without asking Joe.

## 793b30 M3: deep recall's writes carry provenance

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Finding:** [793a3g Deep recall's writes are the System's](793a_product-spec.md#793a3g-deep-recalls-writes-are-the-systems) writes have no provenance, and [793a3i Precision is graded independently](793a_product-spec.md#793a3i-precision-is-graded-independently) grades more than captures.
- **Fix to be applied:** [793a3g Deep recall's writes are the System's](793a_product-spec.md#793a3g-deep-recalls-writes-are-the-systems) writes carry [793a3d Provenance](793a_product-spec.md#793a3d-provenance) provenance plus their derived marker. Scope [793a3i Precision is graded independently](793a_product-spec.md#793a3i-precision-is-graded-independently) to [793a3a Evaluate every completed turn](793a_product-spec.md#793a3a-evaluate-every-completed-turn) captures. [793a3g1a A coverage write is made without the working agent](793a_product-spec.md#793a3g1-uat-scenarios) and [793a3g1b A sound conclusion is persisted without the working agent](793a_product-spec.md#793a3g1-uat-scenarios) assert that no write request appears in the transcript.
- **Status:** applied in 5fd6712a. The second review of commit 058c70b3 found it; the orchestrator accepted it as a fix to make without asking Joe.

## 793b31 M4: coexistence defers to the owning requirements

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Finding:** [793a5b Coexistence without double-firing](793a_product-spec.md#793a5b-coexistence-without-double-firing) restates duplicate rules that [793a2e No re-injection](793a_product-spec.md#793a2e-no-re-injection) and [793a3c No duplicates](793a_product-spec.md#793a3c-no-duplicates) own, and the outcome of [793a5b1a Both paths active](793a_product-spec.md#793a5b1-uat-scenarios) contradicts them.
- **Fix to be applied:** [793a5b Coexistence without double-firing](793a_product-spec.md#793a5b-coexistence-without-double-firing) defers to [793a2e No re-injection](793a_product-spec.md#793a2e-no-re-injection) and [793a3c No duplicates](793a_product-spec.md#793a3c-no-duplicates). Fix the [793a5b1a Both paths active](793a_product-spec.md#793a5b1-uat-scenarios) outcome.
- **Status:** applied in 5fd6712a. The second review of commit 058c70b3 found it; the orchestrator accepted it as a fix to make without asking Joe.

## 793b32 M5: the purview rule restricts the System, not the main agent

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Finding:** [793a4c Memory purview only](793a_product-spec.md#793a4c-memory-purview-only) reads as if the System could never affect the agent at all, though it injects notes by design.
- **Fix to be applied:** [793a4c Memory purview only](793a_product-spec.md#793a4c-memory-purview-only) becomes "MUST NOT itself make or override …". Fix its scenario.
- **Status:** applied in 5fd6712a. The second review of commit 058c70b3 found it; the orchestrator accepted it as a fix to make without asking Joe.

## 793b33 M6: grade every capture independently

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Finding:** [793a3i Precision is graded independently](793a_product-spec.md#793a3i-precision-is-graded-independently) samples 30 notes, which hides the grader's coverage.
- **Fix to be applied:** [793a3i Precision is graded independently](793a_product-spec.md#793a3i-precision-is-graded-independently): grade every [793a3a Evaluate every completed turn](793a_product-spec.md#793a3a-evaluate-every-completed-turn) capture independently, and log the verdicts, the sample size and the precision.
- **Status:** applied in 5fd6712a. The second review of commit 058c70b3 found it; the orchestrator accepted it as a fix to make without asking Joe.

## 793b34 M7: a scenario stops asserting "context"

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Finding:** [793a2j1a Delivered items equal the depth's output](793a_product-spec.md#793a2j1-uat-scenarios) asserts on "context", which an end user cannot observe ([793b16 Observable scenarios](#793b16-observable-scenarios)).
- **Fix to be applied:** [793a2j1a Delivered items equal the depth's output](793a_product-spec.md#793a2j1-uat-scenarios) stops asserting "context" and asserts what the transcript shows.
- **Status:** applied in 5fd6712a. The second review of commit 058c70b3 found it; the orchestrator accepted it as a fix to make without asking Joe.

## 793b35 M8: "Reinforced" is not backed by a MUST

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Finding:** [793a3c1a Existing note already covers the lesson](793a_product-spec.md#793a3c1-uat-scenarios) says an existing note is "reinforced", but [793a3c No duplicates](793a_product-spec.md#793a3c-no-duplicates) has no requirement for it.
- **Fix to be applied:** Either [793a3c No duplicates](793a_product-spec.md#793a3c-no-duplicates) gets a MUST for reinforcement, or the word is dropped from [793a3c1a Existing note already covers the lesson](793a_product-spec.md#793a3c1-uat-scenarios). The orchestrator recommends dropping it from the scenario unless Joe wants a MUST.
- **Status:** applied in 5fd6712a, by dropping the word from the scenario, as the orchestrator recommended. Reinforcement stays SHOULD ([793b50 Reinforcement stays SHOULD](#793b50-reinforcement-stays-should)). The second review of commit 058c70b3 found it; the orchestrator accepted it as a fix to make without asking Joe.

## 793b36 L1: scenario Given refers to the recent-activity channel

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Finding:** [793a2h1d A recently surfaced standard uses deep](793a_product-spec.md#793a2h1-uat-scenarios)'s Given does not name the recent-activity channel the standard comes from.
- **Fix to be applied:** [793a2h1d A recently surfaced standard uses deep](793a_product-spec.md#793a2h1-uat-scenarios)'s Given refers to the recent-activity channel.
- **Status:** applied in 5fd6712a. The second review of commit 058c70b3 found it; the orchestrator accepted it as a fix to make without asking Joe.

## 793b37 L2: next response or tool call

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Finding:** [793a2b Recall after each tool result](793a_product-spec.md#793a2b-recall-after-each-tool-result) says "next response", but a tool call can come first.
- **Fix to be applied:** [793a2b Recall after each tool result](793a_product-spec.md#793a2b-recall-after-each-tool-result) says "next response or tool call".
- **Status:** applied in 5fd6712a. The second review of commit 058c70b3 found it; the orchestrator accepted it as a fix to make without asking Joe.

## 793b38 L3: drop "Added:" from link labels

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Finding:** Some link labels carry the delta-operation word "Added:", which docs/AGENTS.md "References to Addressable Content" forbids.
- **Fix to be applied:** Drop "Added:" from those link labels.
- **Status:** applied in 5fd6712a. The second review of commit 058c70b3 found it; the orchestrator accepted it as a fix to make without asking Joe.

## 793b39 L4: rationale states the why only

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Finding:** Rationale text in [793a3h Capture rate exceeds the baseline](793a_product-spec.md#793a3h-capture-rate-exceeds-the-baseline) and the #846 mention in [793a3a Evaluate every completed turn](793a_product-spec.md#793a3a-evaluate-every-completed-turn) describe history instead of the why, and "(formerly REC-n)" notes remain.
- **Fix to be applied:** Rationale states the why only in [793a3h Capture rate exceeds the baseline](793a_product-spec.md#793a3h-capture-rate-exceeds-the-baseline) and in [793a3a Evaluate every completed turn](793a_product-spec.md#793a3a-evaluate-every-completed-turn) (the #846 mention); strip the "(formerly REC-n)" notes at promotion.
- **Status:** applied in 5fd6712a. The second review of commit 058c70b3 found it; the orchestrator accepted it as a fix to make without asking Joe.

## 793b40 L5: correct the ID map

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Finding:** The [793b14 ID handling](#793b14-id-handling) ID map calls [793a2e1a Note delivered earlier is not delivered again](793a_product-spec.md#793a2e1-uat-scenarios) new though it was rewritten, and omits material modifications.
- **Fix to be applied:** Correct the [793b14 ID handling](#793b14-id-handling) map: [793a2e1a Note delivered earlier is not delivered again](793a_product-spec.md#793a2e1-uat-scenarios) was rewritten, not new; name [793a2e No re-injection](793a_product-spec.md#793a2e-no-re-injection) and [793a4d Sensitive content](793a_product-spec.md#793a4d-sensitive-content) as material modifications; list the scenarios modified in place.
- **Status:** applied in 5fd6712a. The second review of commit 058c70b3 found it; the orchestrator accepted it as a fix to make without asking Joe.

## 793b41 L6: a scenario for a discarded capture

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Finding:** [793a5e Auditability and measurability](793a_product-spec.md#793a5e-auditability-and-measurability) has no scenario for a capture the System discards.
- **Fix to be applied:** Add a [793a5e Auditability and measurability](793a_product-spec.md#793a5e-auditability-and-measurability) scenario for a discarded capture.
- **Status:** applied in 5fd6712a. The second review of commit 058c70b3 found it; the orchestrator accepted it as a fix to make without asking Joe.

## 793b42 L7: trim 793a2j1b

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Finding:** [793a2j1b A vault holding more than 20 notes about the topic](793a_product-spec.md#793a2j1-uat-scenarios) asserts more than its requirement states.
- **Fix to be applied:** Trim [793a2j1b A vault holding more than 20 notes about the topic](793a_product-spec.md#793a2j1-uat-scenarios).
- **Status:** applied in 5fd6712a. The second review of commit 058c70b3 found it; the orchestrator accepted it as a fix to make without asking Joe.

## 793b43 Author's choice accepted: log depth, phrases and escalation

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Choice:** the reviewer accepted the author's choice to log depth and phrases ([793b8 Per-moment record sits in the existing log requirement](#793b8-per-moment-record-sits-in-the-existing-log-requirement)), extended to each query's phrases and whether glance escalated.
- **Rationale:** the depth scenarios stay checkable from the operator-accessible log.
- **Status:** applied in 5fd6712a: [793a5e Auditability and measurability](793a_product-spec.md#793a5e-auditability-and-measurability) logs the phrases of each query and whether glance escalated.

## 793b44 Author's choice accepted: the rate/precision split

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Choice:** the reviewer accepted splitting `793a3e` into [793a3h Capture rate exceeds the baseline](793a_product-spec.md#793a3h-capture-rate-exceeds-the-baseline) and [793a3i Precision is graded independently](793a_product-spec.md#793a3i-precision-is-graded-independently) ([793b9 Capture rate and precision are separate requirements](#793b9-capture-rate-and-precision-are-separate-requirements)).
- **Rationale:** each gets one statement and its own scenarios.
- **Status:** accepted; nothing to apply.

## 793b45 Author's choice accepted: a conditional recommendation check

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Choice:** the reviewer accepted the conditional recommendation check ([792a14o An emergent recommendation is checked against the vault](../../specs/product/recall.md#792a14o-an-emergent-recommendation-is-checked-against-the-vault)), worded as the H2 fix says.
- **Rationale:** the check runs only when a recommendation is emergent, so counting it unconditionally overstates the output.
- **Status:** applied in 5fd6712a (see [793b25 H2: drop the double count of the re-entry query](#793b25-h2-drop-the-double-count-of-the-re-entry-query)).

## 793b46 Inject what recall delivers today

- **Decided by:** Owner (Joe), [793c8 What does the System inject when only incidental items come back?](793c_open-questions.md#793c8-what-does-the-system-inject-when-only-incidental-items-come-back), chose (a): inject what recall delivers today. Author (agent), interpreting [793c8 What does the System inject when only incidental items come back?](793c_open-questions.md#793c8-what-does-the-system-inject-when-only-incidental-items-come-back), chose the wording below; awaiting owner validation.
- **Choice:** [793a2j Volume equals the depth's output](793a_product-spec.md#793a2j-volume-equals-the-depths-output) names the explore picks, ride-along notes and newest chunks as incidental items that are injected even with no direct match. What recall delivers is the lazy payload of [792a14f One unified query](../../specs/product/recall.md#792a14f-one-unified-query): notes with their content and chunks as stubs ([793b62 Chunks are injected lazily, as stubs](#793b62-chunks-are-injected-lazily-as-stubs)), so "today" in [793c8 What does the System inject when only incidental items come back?](793c_open-questions.md#793c8-what-does-the-system-inject-when-only-incidental-items-come-back) means that payload. The relevance-threshold MUST NOT is removed from [793a2e No re-injection](793a_product-spec.md#793a2e-no-re-injection), which is retitled "No re-injection" and keeps its no-re-injection rule for notes and chunks ([793b26 H3: lazy mode truncates only chunks, never notes](#793b26-h3-lazy-mode-truncates-only-chunks-never-notes)).
- **Choice:** the scenarios that said nothing is injected when no note is relevant are retired: `793a2a1b` is replaced by [793a2a1c Only incidental items come back](793a_product-spec.md#793a2a1-uat-scenarios) and `793a2e1b` has no replacement. [793a2j1c Incidental items are delivered with the direct matches](793a_product-spec.md#793a2j1-uat-scenarios) and [793a2j1d An escalated moment delivers both depths' output](793a_product-spec.md#793a2j1-uat-scenarios) are new. [793a2a Recall at a human prompt](793a_product-spec.md#793a2a-recall-at-a-human-prompt), [793a2b Recall after each tool result](793a_product-spec.md#793a2b-recall-after-each-tool-result) and [793a2c Recall at subagent dispatch](793a_product-spec.md#793a2c-recall-at-subagent-dispatch) now say "what recall returns" instead of "relevant notes", so they do not contradict [793a2j Volume equals the depth's output](793a_product-spec.md#793a2j-volume-equals-the-depths-output).
- **Rationale:** Joe chose to deliver the depth's whole output so that the System's effect is attributable to who initiates recall and not to what it delivers.
- **Alternatives:** (b) direct matches and their ride-alongs only, and (c) the full output only when something matches. Not chosen.
- **Follow-up:** trimming the incidental items is tracked in [#848](https://github.com/toejough/engram/issues/848), gated on the volume data of #845. It is not normative here.

## 793b47 Used notes are activated at turn-end capture

- **Decided by:** Owner (Joe), [793c9 Does the System activate notes?](793c_open-questions.md#793c9-does-the-system-activate-notes), chose (c): notes the agent went on to use, judged at turn-end capture. Author (agent), interpreting [793c9 Does the System activate notes?](793c_open-questions.md#793c9-does-the-system-activate-notes), chose the scope, the parent-note clause and the scenarios; awaiting owner validation.
- **Choice:** new requirement [793a3k Used notes are activated](793a_product-spec.md#793a3k-used-notes-are-activated), mirroring [792a14l Only the notes that were used are activated](../../specs/product/recall.md#792a14l-only-the-notes-that-were-used-are-activated): activate the injected notes the agent used, never an ignored note or a chunk. Recall's rule that a used `from_parent` note is activated by its path, so that a pending copy appears and stays pending, is included because [792a14l Only the notes that were used are activated](../../specs/product/recall.md#792a14l-only-the-notes-that-were-used-are-activated) states it for glance and deep alike ([792a14c Recall has two depths](../../specs/product/recall.md#792a14c-recall-has-two-depths) gives the rationale).
- **Choice:** scope is the notes the System injected during the turn. Whether notes the agent retrieved itself should also count is open in [793c13 Does the System activate notes the agent retrieved itself?](793c_open-questions.md#793c13-does-the-system-activate-notes-the-agent-retrieved-itself).
- **Choice:** activation decisions are logged in [793a5e Auditability and measurability](793a_product-spec.md#793a5e-auditability-and-measurability), so that an activation is visible in the log as well as in the note's recency metadata.
- **Rationale:** it applies recall's "only the notes that were used" rule at a moment the System already evaluates.
- **Alternatives:** (a) never, and (b) every injected note. Not chosen.

## 793b48 Latency covers the instruction path's full lookup

- **Decided by:** Owner (Joe), [793c11 What does the latency bound cover?](793c_open-questions.md#793c11-what-does-the-latency-bound-cover), chose (b): the instruction path's full lookup, phrasing plus queries, on the same input. Author (agent), interpreting [793c11 What does the latency bound cover?](793c_open-questions.md#793c11-what-does-the-latency-bound-cover) and [793b27 H4: an escalated moment covers both depths' queries](#793b27-h4-an-escalated-moment-covers-both-depths-queries), chose the wording and the scenarios; awaiting owner validation.
- **Choice:** [793a5g Latency per recall moment](793a_product-spec.md#793a5g-latency-per-recall-moment) bounds the System's added latency by the time to generate the query phrases plus run the chosen depth's queries, both depths' when glance escalates. [793a5g1a Added latency stays within the instruction path's lookup latency](793a_product-spec.md#793a5g1-uat-scenarios) adds subagent dispatch and compares with the whole lookup; [793a5g1b An escalated moment is compared with both depths](793a_product-spec.md#793a5g1-uat-scenarios) covers the escalated moment.
- **Rationale:** it compares like with like and needs no fixed number that would go stale.
- **Alternatives:** (a) queries only, and (c) queries plus a fixed allowance. Not chosen.

## 793b49 No operating modes

- **Decided by:** Owner (Joe), [793c12 Are operating modes needed at all?](793c_open-questions.md#793c12-are-operating-modes-needed-at-all), chose (a): no modes. Author (agent), interpreting [793c12 Are operating modes needed at all?](793c_open-questions.md#793c12-are-operating-modes-needed-at-all), chose the new requirement, the scenarios and where measurement moved; awaiting owner validation.
- **Choice:** `793a4e` and its scenarios `793a4e1a`..`793a4e1e` are retired and their IDs are not reused. New requirement [793a4g Active whenever installed](793a_product-spec.md#793a4g-active-whenever-installed): the System is active whenever the integration is installed and enabled, and injects nothing, captures nothing and records nothing when it is disabled or uninstalled; scenarios [793a4g1a An Integration installed by update is active](793a_product-spec.md#793a4g1-uat-scenarios), `793a4g1b A disabled integration is inert` and [793a4g1c An uninstalled Integration is inert](793a_product-spec.md#793a4g1-uat-scenarios).
- **Choice:** the measurement rule moved into [793a5e Auditability and measurability](793a_product-spec.md#793a5e-auditability-and-measurability): it is recorded whenever the System is active, and no configuration switches it off; scenario [793a5e1e Measurement cannot be switched off while the System is active](793a_product-spec.md#793a5e1-uat-scenarios) replaces `793a4e1e`. Scenarios that said the System is "off" or "in active mode" now say disabled or enabled ([793a4c1a Tool calls and permissions unchanged](793a_product-spec.md#793a4c1-uat-scenarios), [793a4c1b The prompt is unchanged and the note is separate](793a_product-spec.md#793a4c1-uat-scenarios), [793a5f1a Treatment beats control](793a_product-spec.md#793a5f1-uat-scenarios)). [793a4b Failure isolation](793a_product-spec.md#793a4b-failure-isolation) and [793a5c Environments without the capability](793a_product-spec.md#793a5c-environments-without-the-capability) mentioned no modes.
- **Choice:** [793c10 What is the default mode for an existing install that updates?](793c_open-questions.md#793c10-what-is-the-default-mode-for-an-existing-install-that-updates) is retired by the same answer, and M1 ([793b28 M1: observe mode logs the would-be volume](#793b28-m1-observe-mode-logs-the-would-be-volume)) is superseded.
- **Rationale:** the modes' reasons were an observe-first rollout, which default-active and always-on measurement remove, and an emergency off switch, which disabling or uninstalling covers.
- **Alternatives:** (b) an off switch only, and (c) keep all three modes. Not chosen.

## 793b50 Reinforcement stays SHOULD

- **Decided by:** Orchestrator (Claude), recommendation for [793b35 M8: "Reinforced" is not backed by a MUST](#793b35-m8-reinforced-is-not-backed-by-a-must), because the owner chose nothing different; awaiting owner validation. Applied by the author.
- **Choice:** [793a3c No duplicates](793a_product-spec.md#793a3c-no-duplicates) keeps "SHOULD reinforce" and [793a3c1a Existing note already covers the lesson](793a_product-spec.md#793a3c1-uat-scenarios) no longer asserts a "reinforced" mark. Whether reinforcement should be a MUST is open in [793c14 Should reinforcing an existing note be a MUST?](793c_open-questions.md#793c14-should-reinforcing-an-existing-note-be-a-must).
- **Rationale:** a scenario cannot hold the System to a behavior the requirement leaves optional.
- **Alternatives:** make reinforcement a MUST with a scenario. Not chosen without the owner's say.
- **Status:** superseded by [793b52 Reinforcement is required](#793b52-reinforcement-is-required) (Joe chose MUST in [793c14 Should reinforcing an existing note be a MUST?](793c_open-questions.md#793c14-should-reinforcing-an-existing-note-be-a-must)).

## 793b51 Phrasing choices in applying the second review

- **Decided by:** Author (agent), interpreting the findings of [793b25 H2: drop the double count of the re-entry query](#793b25-h2-drop-the-double-count-of-the-re-entry-query) to [793b45 Author's choice accepted: a conditional recommendation check](#793b45-authors-choice-accepted-a-conditional-recommendation-check); awaiting owner validation.
- **Choice:**
  - [793a2e No re-injection](793a_product-spec.md#793a2e-no-re-injection) covers "an item (a note or a chunk)"; `793a2e1f A chunk shown without content is delivered with it` is rewritten around a chunk, and [793a2e1h A chunk whose content the agent fetched is not injected](793a_product-spec.md#793a2e1-uat-scenarios) is new for a chunk delivered earlier.
  - [793a3d Provenance](793a_product-spec.md#793a3d-provenance) covers every note the System writes automatically, so [793a3g Deep recall's writes are the System's](793a_product-spec.md#793a3g-deep-recalls-writes-are-the-systems) writes carry it; a persisted conclusion also has its source marked as derived. [793a3g1a A coverage write is made without the working agent](793a_product-spec.md#793a3g1-uat-scenarios) and [793a3g1b A sound conclusion is persisted without the working agent](793a_product-spec.md#793a3g1-uat-scenarios) assert the provenance and that the transcript shows no request to write.
  - [793a3i Precision is graded independently](793a_product-spec.md#793a3i-precision-is-graded-independently) grades every capture from [793a3a Evaluate every completed turn](793a_product-spec.md#793a3a-evaluate-every-completed-turn) and logs each verdict, the number graded and the precision; the "sample" wording and the 30-note sample are replaced by the count of captures graded, and [793a3i1a Precision is reported with the number graded](793a_product-spec.md#793a3i1-uat-scenarios) is retitled.
  - [793a4f No recall or capture in its own evaluation sessions](793a_product-spec.md#793a4f-no-recall-or-capture-in-its-own-evaluation-sessions) requires the System to log the id of each session it starts as a System session; [793a4f1b The session that chooses depth and phrases is left alone](793a_product-spec.md#793a4f1-uat-scenarios) covers the session that chooses depth and phrases.
  - [793a4c Memory purview only](793a_product-spec.md#793a4c-memory-purview-only) reads "MUST NOT itself make or override"; [793a4c1a Tool calls and permissions unchanged](793a_product-spec.md#793a4c1-uat-scenarios) checks the transcript for the arguments, permission decisions and model instead of comparing two runs.
  - [793a5b Coexistence without double-firing](793a_product-spec.md#793a5b-coexistence-without-double-firing) defers to [793a2e No re-injection](793a_product-spec.md#793a2e-no-re-injection) and [793a3c No duplicates](793a_product-spec.md#793a3c-no-duplicates); [793a5b1a Both paths active](793a_product-spec.md#793a5b1-uat-scenarios) asserts that the System's injection is omitted when the agent's own lookup already displayed the note.
  - [793a5e Auditability and measurability](793a_product-spec.md#793a5e-auditability-and-measurability) also logs whether glance escalated and the phrases of each query ([793a5e1f An escalated moment records both depths](793a_product-spec.md#793a5e1-uat-scenarios)), and each activation decision; [793a5e1d A discarded capture is recorded with its reason](793a_product-spec.md#793a5e1-uat-scenarios) covers a discarded capture.
  - [793a2h1d A recently surfaced standard uses deep](793a_product-spec.md#793a2h1-uat-scenarios) names the recent-activity channel; [793a2h1c An unresolved gap uses deep](793a_product-spec.md#793a2h1-uat-scenarios) records the escalation; [793a2j1a Delivered items equal the depth's output](793a_product-spec.md#793a2j1-uat-scenarios) asserts on the transcript; [793a2j1b A vault holding more than 20 notes about the topic](793a_product-spec.md#793a2j1-uat-scenarios) is trimmed; [793a2b Recall after each tool result](793a_product-spec.md#793a2b-recall-after-each-tool-result) says "next response or tool call".
  - The "Added:" word is dropped from link labels, and the rationale text in [793a3a Evaluate every completed turn](793a_product-spec.md#793a3a-evaluate-every-completed-turn) and [793a3h Capture rate exceeds the baseline](793a_product-spec.md#793a3h-capture-rate-exceeds-the-baseline) states the why only.
- **Rationale:** each follows the finding's fix and keeps every MUST branch checkable from the transcript or the operator log.

## 793b52 Reinforcement is required

- **Choice:** [793a3c No duplicates](793a_product-spec.md#793a3c-no-duplicates) makes reinforcing an equivalent existing note a MUST, and [793a3c1a Existing note already covers the lesson](793a_product-spec.md#793a3c1-uat-scenarios) asserts that the log records a reinforced capture decision. This supersedes [793b50 Reinforcement stays SHOULD](#793b50-reinforcement-stays-should).
- **Decided by:** Owner (Joe), [793c14 Should reinforcing an existing note be a MUST?](793c_open-questions.md#793c14-should-reinforcing-an-existing-note-be-a-must); scenario wording by Orchestrator (Claude), awaiting owner validation.

## 793b53 M3: the ID map names the nodes added after the first revision

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Finding:** The [793b2 ID mapping](#793b2-id-mapping) map is incomplete: [793a3k Used notes are activated](793a_product-spec.md#793a3k-used-notes-are-activated) and [793a4g Active whenever installed](793a_product-spec.md#793a4g-active-whenever-installed) are new but missing from it, [793b14 ID handling](#793b14-id-handling) omits [793a3h Capture rate exceeds the baseline](793a_product-spec.md#793a3h-capture-rate-exceeds-the-baseline) from the modified-in-place list, and [793a2h Recall depth per moment](793a_product-spec.md#793a2h-recall-depth-per-moment), [793a3g Deep recall's writes are the System's](793a_product-spec.md#793a3g-deep-recalls-writes-are-the-systems) and [793a4f No recall or capture in its own evaluation sessions](793a_product-spec.md#793a4f-no-recall-or-capture-in-its-own-evaluation-sessions) are listed both as modified and as new.
- **Fix to be applied:** The 4.2 row of [793b2 ID mapping](#793b2-id-mapping) adds [793a3k Used notes are activated](793a_product-spec.md#793a3k-used-notes-are-activated) as new and the 4.3 row adds [793a4g Active whenever installed](793a_product-spec.md#793a4g-active-whenever-installed) as new. [793b14 ID handling](#793b14-id-handling) adds [793a3h Capture rate exceeds the baseline](793a_product-spec.md#793a3h-capture-rate-exceeds-the-baseline) to the modified-in-place list. Where [793a2h Recall depth per moment](793a_product-spec.md#793a2h-recall-depth-per-moment), [793a3g Deep recall's writes are the System's](793a_product-spec.md#793a3g-deep-recalls-writes-are-the-systems) and [793a4f No recall or capture in its own evaluation sessions](793a_product-spec.md#793a4f-no-recall-or-capture-in-its-own-evaluation-sessions) appeared both as modified and as new, they now say "introduced in an earlier revision, modified since".
- **Status:** applied in the commit that records the third review (round 3 of change 793).

## 793b54 M4: volume is a count of items and of bytes

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Finding:** [793a5e Auditability and measurability](793a_product-spec.md#793a5e-auditability-and-measurability) records "the volume injected" without a unit, so the record cannot be compared with the 135 KB and 45-item figures in [793c2 Are the volume bounds and latency budget accepted?](793c_open-questions.md#793c2-are-the-volume-bounds-and-latency-budget-accepted).
- **Fix to be applied:** [793a5e Auditability and measurability](793a_product-spec.md#793a5e-auditability-and-measurability) reads "the volume injected, as the number of items and the number of bytes".
- **Status:** applied in the commit that records the third review (round 3 of change 793).

## 793b55 M5: the purview rule names its real effects

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Finding:** [793a4c Memory purview only](793a_product-spec.md#793a4c-memory-purview-only) allowed "injecting reference notes and writing vault notes" as its only effects, but the System also activates notes, writes its log and starts System sessions, and "task decisions" is not checkable.
- **Fix to be applied:** [793a4c Memory purview only](793a_product-spec.md#793a4c-memory-purview-only) reads: MUST NOT itself make or override tool permissions, tool arguments, model choice, or the user's prompt text. Its only effect on the working session is the items it injects; outside the session it writes and activates vault notes and writes its log. "Task decisions" is dropped. The remaining branches have scenarios: [793a4c1a Tool calls and permissions unchanged](793a_product-spec.md#793a4c1-uat-scenarios) and [793a4c1b The prompt is unchanged and the note is separate](793a_product-spec.md#793a4c1-uat-scenarios) already covered permissions, arguments, model and prompt; new [793a4c1c The only effect on the session is the injected items](793a_product-spec.md#793a4c1-uat-scenarios) and [793a4c1d Effects outside the session are vault notes and the log](793a_product-spec.md#793a4c1-uat-scenarios) cover the two effect branches.
- **Status:** applied in the commit that records the third review (round 3 of change 793).

## 793b56 L1: capture and prompt recall exclude System sessions

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Finding:** [793a2i Recall for every prompt regardless of source](793a_product-spec.md#793a2i-recall-for-every-prompt-regardless-of-source) and [793a3a Evaluate every completed turn](793a_product-spec.md#793a3a-evaluate-every-completed-turn) apply to all sessions, which contradicts [793a4f No recall or capture in its own evaluation sessions](793a_product-spec.md#793a4f-no-recall-or-capture-in-its-own-evaluation-sessions).
- **Fix to be applied:** Both say "in every working session (excluding System sessions per [793a4f No recall or capture in its own evaluation sessions](793a_product-spec.md#793a4f-no-recall-or-capture-in-its-own-evaluation-sessions))".
- **Status:** applied in the commit that records the third review (round 3 of change 793).

## 793b57 L2: the incidental-items scenario names the phrases it replays

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Finding:** [793a2a1c Only incidental items come back](793a_product-spec.md#793a2a1-uat-scenarios)'s Given has the operator run the glance query without saying which phrases, so the operator's query may differ from the System's.
- **Fix to be applied:** [793a2a1c Only incidental items come back](793a_product-spec.md#793a2a1-uat-scenarios) has the operator run the glance query with the phrases the log records for that moment and find only newest chunks returned.
- **Status:** applied in the commit that records the third review (round 3 of change 793).

## 793b58 L3: no setting may switch the recording off

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Finding:** [793a5e Auditability and measurability](793a_product-spec.md#793a5e-auditability-and-measurability)'s "no configuration MUST switch the recording off" is not a requirement on the System, and [793a5e1e Measurement cannot be switched off while the System is active](793a_product-spec.md#793a5e1-uat-scenarios) does not say which setting was tried.
- **Fix to be applied:** [793a5e Auditability and measurability](793a_product-spec.md#793a5e-auditability-and-measurability) reads "MUST NOT provide, or honor, any setting that switches the recording off while the System is active". [793a5e1e Measurement cannot be switched off while the System is active](793a_product-spec.md#793a5e1-uat-scenarios)'s Given names the settings tried, an environment variable or config key that would disable logging.
- **Status:** applied in the commit that records the third review (round 3 of change 793).

## 793b59 L4: chunks and source identification (held for the owner)

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Finding:** [793a2d Attributable presentation](793a_product-spec.md#793a2d-attributable-presentation) requires injected notes to identify their source, but the System also injects chunks, and a chunk has no note file name.
- **Fix to be applied:** [793a2d Attributable presentation](793a_product-spec.md#793a2d-attributable-presentation) reads "Injected items MUST identify their source: a note by its file name and a chunk by its id", per Joe's answer to [793c20 Must injected chunks identify their source like notes?](793c_open-questions.md#793c20-must-injected-chunks-identify-their-source-like-notes) ([793b67 Every injected item names its source](#793b67-every-injected-item-names-its-source)).
- **Status:** applied in commit 1adbb77e.

## 793b60 L5: link live IDs, backtick retired ones

- **Decided by:** Reviewer (opus agent), accepted by Orchestrator (Claude); awaiting owner validation.
- **Finding:** Several live IDs in this change's artifacts are plain text (the decision line of [793b12 Recall for every prompt](#793b12-recall-for-every-prompt), the header of [`793c_open-questions.md`](793c_open-questions.md) and the "Applied in" line of [793c3 What is the default mode for a fresh install?](793c_open-questions.md#793c3-what-is-the-default-mode-for-a-fresh-install), and a few status lines), and the retired `793a3e` is plain text in [793b9 Capture rate and precision are separate requirements](#793b9-capture-rate-and-precision-are-separate-requirements) and [793b44 Author's choice accepted: the rate/precision split](#793b44-authors-choice-accepted-the-rateprecision-split).
- **Fix to be applied:** Live IDs are linked with "ID title" labels; the retired `793a3e` is in backticks in [793b9 Capture rate and precision are separate requirements](#793b9-capture-rate-and-precision-are-separate-requirements) and [793b44 Author's choice accepted: the rate/precision split](#793b44-authors-choice-accepted-the-rateprecision-split).
- **Status:** applied in the commit that records the third review (round 3 of change 793).

## 793b61 Third review: findings waiting on open questions

- **Decided by:** Reviewer (opus agent) found them and Orchestrator (Claude) held them. Owner (Joe) resolved each through the answers named below; Author (agent) applied them and awaits owner validation.
- **Note:** the review's original text is not kept in the repository. Each finding below is written from the question it raised and from the summary this entry carried before the answers, so its wording is the author's.
- **H1: lazy chunks versus chunk content.**
  - **Finding:** [792a14f One unified query](../../specs/product/recall.md#792a14f-one-unified-query) requires every recall query to run with `--lazy-chunks`, which prints a chunk as its id and score only ([792a7s Lazy chunks](../../specs/product/memory-query.md#792a7s-lazy-chunks)). The product spec had the System run the query under its defaults, which print chunks with content, and had `793a2e1f A chunk shown without content is delivered with it` (retired) deliver a chunk's content, so the System and the instruction path would deliver different payloads. The volume baseline in [793c2 Are the volume bounds and latency budget accepted?](793c_open-questions.md#793c2-are-the-volume-bounds-and-latency-budget-accepted) was also measured with chunk content.
  - **Fix to be applied:** decide whether the System's chunks carry content, then make [793a2j Volume equals the depth's output](793a_product-spec.md#793a2j-volume-equals-the-depths-output), [793a2e No re-injection](793a_product-spec.md#793a2e-no-re-injection) and the chunk scenarios say the same thing.
  - **Resolution:** [793c15 Does the System inject chunks with their content?](793c_open-questions.md#793c15-does-the-system-inject-chunks-with-their-content) answered (b), lazy chunks; applied in [793b62 Chunks are injected lazily, as stubs](#793b62-chunks-are-injected-lazily-as-stubs).
- **H2: a coverage write versus used-note activation.**
  - **Finding:** at a deep moment the coverage write ([793a3g Deep recall's writes are the System's](793a_product-spec.md#793a3g-deep-recalls-writes-are-the-systems), [792a14j Coverage is judged by the agent and written once per cluster](../../specs/product/recall.md#792a14j-coverage-is-judged-by-the-agent-and-written-once-per-cluster)) adds chunk sources to an existing note, and recall's covered outcome does so with `--activate`, which bumps the note's recency. [793a3k Used notes are activated](793a_product-spec.md#793a3k-used-notes-are-activated) reserves that bump for notes the agent used, so the two requirements could not both hold.
  - **Fix to be applied:** decide whether a coverage write activates the note it covers, and how the System knows a note was used.
  - **Resolution:** [793c16 At a deep moment, does the System's coverage write activate the note it covers?](793c_open-questions.md#793c16-at-a-deep-moment-does-the-systems-coverage-write-activate-the-note-it-covers) answered that the write attaches evidence and the covering note is injected but not activated, and [793c21 How does the System know a note was used?](793c_open-questions.md#793c21-how-does-the-system-know-a-note-was-used) answered that use is inferred from the transcript; applied in [793b63 A coverage write attaches evidence and injects, and never activates](#793b63-a-coverage-write-attaches-evidence-and-injects-and-never-activates) and [793b68 Use is inferred from the transcript and graded](#793b68-use-is-inferred-from-the-transcript-and-graded).
- **H3: the latency bound omitted the reading.**
  - **Finding:** [793a5g Latency per recall moment](793a_product-spec.md#793a5g-latency-per-recall-moment) bounded the added latency by generating the phrases and running the queries. Escalation from glance to deep ([793a2h Recall depth per moment](793a_product-spec.md#793a2h-recall-depth-per-moment), [792a14d Glance escalates to deep for a recent standard](../../specs/product/recall.md#792a14d-glance-escalates-to-deep-for-a-recent-standard)) and the recommendation check ([792a14o An emergent recommendation is checked against the vault](../../specs/product/recall.md#792a14o-an-emergent-recommendation-is-checked-against-the-vault)) depend on reading the glance results, which takes agent time the bound left out, so the System could be held to less work than the instruction path does.
  - **Fix to be applied:** decide whether the bound includes that reading.
  - **Resolution:** [793c17 Does the latency bound include reading the glance results?](793c_open-questions.md#793c17-does-the-latency-bound-include-reading-the-glance-results) answered (a), include it; applied in [793b64 Latency includes reading the results](#793b64-latency-includes-reading-the-results).
- **M1: the integration was undefined.**
  - **Finding:** [793a4g Active whenever installed](793a_product-spec.md#793a4g-active-whenever-installed) made the System active whenever "the engram integration" is installed and enabled, without saying what the integration is, how it gets installed, or how an operator turns it off.
  - **Fix to be applied:** define the term and specify installation and turning off.
  - **Resolution:** [793c18 How is the System installed and turned off?](793c_open-questions.md#793c18-how-is-the-system-installed-and-turned-off) answered (a) with installation in this spec, and [793c22 There is no uninstall command today](793c_open-questions.md#793c22-there-is-no-uninstall-command-today) answered (b), a general uninstall in its own change; applied in [793b65 Installation is part of this spec](#793b65-installation-is-part-of-this-spec) and [793b69 The uninstall command is a separate change](#793b69-the-uninstall-command-is-a-separate-change).
- **Rationale:** an answer may change the fix, so applying them first would risk rework.
- **Status:** resolved and applied in commit 1adbb77e. [793c19 Does the log keep the depth, each query's phrases and the glance escalation?](793c_open-questions.md#793c19-does-the-log-keep-the-depth-each-querys-phrases-and-the-glance-escalation) (keep the log fields) and [793c20 Must injected chunks identify their source like notes?](793c_open-questions.md#793c20-must-injected-chunks-identify-their-source-like-notes) (chunk sources identify themselves), raised by the same review, were answered and applied in [793b66 The log keeps depth, phrases and escalation](#793b66-the-log-keeps-depth-phrases-and-escalation) and [793b67 Every injected item names its source](#793b67-every-injected-item-names-its-source). [793c13 Does the System activate notes the agent retrieved itself?](793c_open-questions.md#793c13-does-the-system-activate-notes-the-agent-retrieved-itself) and [793c14 Should reinforcing an existing note be a MUST?](793c_open-questions.md#793c14-should-reinforcing-an-existing-note-be-a-must) were settled before this round and are not re-raised.

## 793b62 Chunks are injected lazily, as stubs

- **Decided by:** Owner (Joe), [793c15 Does the System inject chunks with their content?](793c_open-questions.md#793c15-does-the-system-inject-chunks-with-their-content), chose (b): lazy chunks as recall runs the query, a rule against re-injecting the same chunk stub, and the chunk-content scenario retired. Author (agent), interpreting [793c15 Does the System inject chunks with their content?](793c_open-questions.md#793c15-does-the-system-inject-chunks-with-their-content), chose the wording below; awaiting owner validation.
- **Choice:** [793a2j Volume equals the depth's output](793a_product-spec.md#793a2j-volume-equals-the-depths-output) says the System's queries use `--lazy-chunks` at glance, at deep and for the recommendation check, names a note as injected with its content and a chunk as a chunk stub, and says each query runs with its default result limit and content budget. The terms in [793a1 Problem Statement](793a_product-spec.md#793a1-problem-statement) gain "Chunk stub", and "Injection" covers it. [793a2e No re-injection](793a_product-spec.md#793a2e-no-re-injection) counts a chunk stub already shown as present and forbids injecting it again, with scenario [793a2e1i A chunk stub already shown is not injected again](793a_product-spec.md#793a2e1-uat-scenarios).
- **Choice:** `793a2e1f` is retired and not reused. [793a2e1h A chunk whose content the agent fetched is not injected](793a_product-spec.md#793a2e1-uat-scenarios) keeps its ID but now covers a chunk whose content the agent fetched itself, because under lazy chunks the System never delivers content; the scenario was retitled. [793a2j1e Chunks are delivered as stubs](793a_product-spec.md#793a2j1-uat-scenarios) asserts that a chunk arrives as id and score without content, and the scenarios for the incidental items and the depth's output say "chunk stubs".
- **Choice:** the volume figures of [793c2 Are the volume bounds and latency budget accepted?](793c_open-questions.md#793c2-are-the-volume-bounds-and-latency-budget-accepted) (135 KB, 45 items) were measured with chunk content, so under lazy chunks they are an upper bound. No normative text carries them.
- **Rationale:** the System should deliver what the instruction path's recall delivers, and recall runs lazy; a stub costs little and the agent can fetch the chunk if it needs it.
- **Alternatives:** (a) run the query without `--lazy-chunks` and deliver chunk content. Not chosen.

## 793b63 A coverage write attaches evidence and injects, and never activates

- **Decided by:** Owner (Joe), [793c16 At a deep moment, does the System's coverage write activate the note it covers?](793c_open-questions.md#793c16-at-a-deep-moment-does-the-systems-coverage-write-activate-the-note-it-covers), decided that when a note covers a cluster the cluster's chunks are attached to it as evidence and the covering note is injected into the working agent, subject to no re-injection, and is marked used only on the same transcript evidence as any injected note ([793c21 How does the System know a note was used?](793c_open-questions.md#793c21-how-does-the-system-know-a-note-was-used)). Author (agent), interpreting [793c16 At a deep moment, does the System's coverage write activate the note it covers?](793c_open-questions.md#793c16-at-a-deep-moment-does-the-systems-coverage-write-activate-the-note-it-covers), chose the wording below; awaiting owner validation.
- **Choice:** [793a3g Deep recall's writes are the System's](793a_product-spec.md#793a3g-deep-recalls-writes-are-the-systems) lists the three outcomes of [792a14j Coverage is judged by the agent and written once per cluster](../../specs/product/recall.md#792a14j-coverage-is-judged-by-the-agent-and-written-once-per-cluster) (covered, near, absent) and refers to that requirement for their definitions. For covered it attaches the chunks as evidence without rewriting the note, injects the note subject to [793a2e No re-injection](793a_product-spec.md#793a2e-no-re-injection), and does not activate it. Near amends the note with the cluster's chunks and the re-synthesized content, and absent writes one new note, as before.
- **Choice:** for an amend, provenance is defined as Joe asked: the chunk sources added and the log entry name the session and turn, and the note's original source is unchanged. [793a3d Provenance](793a_product-spec.md#793a3d-provenance) still governs notes the System writes.
- **Choice:** the near amend does not activate the note either. Joe's answer was about the covered outcome. The System never runs recall's own activation step ([793b68 Use is inferred from the transcript and graded](#793b68-use-is-inferred-from-the-transcript-and-graded)), so I read his rule that only transcript evidence activates a note as covering near, and asked about injecting a near note in [793c23 Is a note amended as near injected?](793c_open-questions.md#793c23-is-a-note-amended-as-near-injected). The overlap of the covering note's injection with [793a2j Volume equals the depth's output](793a_product-spec.md#793a2j-volume-equals-the-depths-output) is asked in [793c24 Does the covering note's injection add anything to the depth's output?](793c_open-questions.md#793c24-does-the-covering-notes-injection-add-anything-to-the-depths-output).
- **Choice:** scenarios [793a3g1c A covered cluster gets evidence and its note is injected](793a_product-spec.md#793a3g1-uat-scenarios), [793a3g1d A covering note already in context is not injected again](793a_product-spec.md#793a3g1-uat-scenarios) and [793a3g1e A near cluster amends the note](793a_product-spec.md#793a3g1-uat-scenarios) are new; the covered and near ones assert that the note's recency is unchanged by the write.
- **Rationale:** a coverage write records that a note covers a cluster, not that the agent used it.
- **Alternatives:** (b) the coverage write activates the covered note. Not chosen.

## 793b64 Latency includes reading the results

- **Decided by:** Owner (Joe), [793c17 Does the latency bound include reading the glance results?](793c_open-questions.md#793c17-does-the-latency-bound-include-reading-the-glance-results), chose (a): the bound includes reading the results that escalation and the recommendation check depend on. Author (agent), interpreting [793c17 Does the latency bound include reading the glance results?](793c_open-questions.md#793c17-does-the-latency-bound-include-reading-the-glance-results), chose the wording below; awaiting owner validation.
- **Choice:** [793a5g Latency per recall moment](793a_product-spec.md#793a5g-latency-per-recall-moment) bounds the added latency by the instruction path's full lookup at that moment: generating the phrases, running the chosen depth's queries, and reading the results that escalation ([792a14d Glance escalates to deep for a recent standard](../../specs/product/recall.md#792a14d-glance-escalates-to-deep-for-a-recent-standard)) and the recommendation check ([792a14o An emergent recommendation is checked against the vault](../../specs/product/recall.md#792a14o-an-emergent-recommendation-is-checked-against-the-vault)) depend on. At an escalated moment that is both depths' queries and that reading. [793a5g1a Added latency stays within the instruction path's lookup latency](793a_product-spec.md#793a5g1-uat-scenarios) and [793a5g1b An escalated moment is compared with both depths](793a_product-spec.md#793a5g1-uat-scenarios) say the same.
- **Rationale:** it compares like with like and keeps the escalation triggers.
- **Alternatives:** (b) phrases and queries only, and (c) drop the result-dependent triggers. Not chosen.

## 793b65 Installation is part of this spec

- **Decided by:** Owner (Joe), [793c18 How is the System installed and turned off?](793c_open-questions.md#793c18-how-is-the-system-installed-and-turned-off), chose (a) with installation in this spec: `engram update` installs what the System needs, existing installs get it on their next update, and the System is turned off by uninstalling. Author (agent), interpreting [793c18 How is the System installed and turned off?](793c_open-questions.md#793c18-how-is-the-system-installed-and-turned-off), chose the wording below; awaiting owner validation.
- **Choice:** the Integration is defined in [793a1 Problem Statement](793a_product-spec.md#793a1-problem-statement) as what `engram update` installs for the System. New requirement [793a4h Installed by engram update](793a_product-spec.md#793a4h-installed-by-engram-update) extends [install-and-update](../../specs/product/install-and-update.md), citing [792a11g The harnesses that are updated](../../specs/product/install-and-update.md#792a11g-the-harnesses-that-are-updated) (which harnesses are updated) and [792a11f Previewing an update](../../specs/product/install-and-update.md#792a11f-previewing-an-update) (a preview). A preview installing none of it follows from [792a11f Previewing an update](../../specs/product/install-and-update.md#792a11f-previewing-an-update)'s rule that a dry run changes nothing in a harness; it is stated because the Integration is new, with scenario [793a4h1c A preview installs none of the Integration](793a_product-spec.md#793a4h1-uat-scenarios). "In which the System can run" ties it to [793a5c Environments without the capability](793a_product-spec.md#793a5c-environments-without-the-capability); the spec does not say whether Pi is such a harness. What the Integration consists of is not specified here.
- **Choice:** [793a4g Active whenever installed](793a_product-spec.md#793a4g-active-whenever-installed) is rewritten: active in every session of a harness with the Integration installed, with no mode and no further step, and inert after the Integration is uninstalled. The "disabled via the harness's own mechanism" wording is gone. `793a4g1b` is retired; [793a4g1a An Integration installed by update is active](793a_product-spec.md#793a4g1-uat-scenarios) and [793a4g1c An uninstalled Integration is inert](793a_product-spec.md#793a4g1-uat-scenarios) remain. The scenario that an existing install gains the System on update is [793a4h1b An existing install gains the System on its next update](793a_product-spec.md#793a4h1-uat-scenarios), under the requirement that states the branch, so that two scenarios do not duplicate each other. Scenarios that spoke of the System "disabled and enabled" now say the Integration is uninstalled or installed ([793a5f1a Treatment beats control](793a_product-spec.md#793a5f1-uat-scenarios)).
- **Rationale:** operators already run `engram update`, and one state is easier to reason about than a mode.
- **Alternatives:** (b) opt-in like guidance ([792a11l Guidance is managed only after opt-in](../../specs/product/install-and-update.md#792a11l-guidance-is-managed-only-after-opt-in)). Not chosen.

## 793b66 The log keeps depth, phrases and escalation

- **Decided by:** Owner (Joe), [793c19 Does the log keep the depth, each query's phrases and the glance escalation?](793c_open-questions.md#793c19-does-the-log-keep-the-depth-each-querys-phrases-and-the-glance-escalation), chose (a): keep the fields. No spec change.
- **Choice:** [793a5e Auditability and measurability](793a_product-spec.md#793a5e-auditability-and-measurability) logs the depth, the phrases of each query and whether glance escalated, as before. The only change to it is that each activation decision is logged with its evidence ([793b68 Use is inferred from the transcript and graded](#793b68-use-is-inferred-from-the-transcript-and-graded)).
- **Rationale:** the depth and escalation scenarios can be checked only from the operator-accessible log.

## 793b67 Every injected item names its source

- **Decided by:** Owner (Joe), [793c20 Must injected chunks identify their source like notes?](793c_open-questions.md#793c20-must-injected-chunks-identify-their-source-like-notes), chose (a): every injected item identifies its source, a note by its file name and a chunk by its id. Author (agent), interpreting [793c20 Must injected chunks identify their source like notes?](793c_open-questions.md#793c20-must-injected-chunks-identify-their-source-like-notes), chose the wording below; awaiting owner validation.
- **Choice:** [793a2d Attributable presentation](793a_product-spec.md#793a2d-attributable-presentation) reads "Injected items MUST identify their source: a note by its file name and a chunk by its id". Its rationale notes that a chunk's id is what the agent fetches it by ([792a7t Reading one chunk](../../specs/product/memory-query.md#792a7t-reading-one-chunk)). [793a2d1a Injected content names its source](793a_product-spec.md#793a2d1-uat-scenarios) now asserts the file name, and [793a2d1b Injected chunk names its id](793a_product-spec.md#793a2d1-uat-scenarios) is new; its example id is illustrative, and the real form is `<source>#<anchor>` ([792a7b The payload](../../specs/product/memory-query.md#792a7b-the-payload)).
- **Rationale:** an item the agent cannot trace to its source cannot be weighed or looked up.

## 793b68 Use is inferred from the transcript and graded

- **Decided by:** Owner (Joe), [793c21 How does the System know a note was used?](793c_open-questions.md#793c21-how-does-the-system-know-a-note-was-used), chose (a): infer use from the transcript at turn-end capture, log each verdict with its evidence, and have the grader sample activation verdicts. Author (agent), interpreting [793c21 How does the System know a note was used?](793c_open-questions.md#793c21-how-does-the-system-know-a-note-was-used), chose the wording below; awaiting owner validation.
- **Choice:** [793a3k Used notes are activated](793a_product-spec.md#793a3k-used-notes-are-activated) has the System infer use from the transcript, with Joe's examples (quoting or citing the note, following its steps, taking the action it prescribes), log each verdict with its evidence, and not activate a note with no evidence. It adds that the System's recall does not run recall's own activation step (Step 2.7 of [792a14l Only the notes that were used are activated](../../specs/product/recall.md#792a14l-only-the-notes-that-were-used-are-activated)), so its activations are the only ones the System makes; scenario [793a3k1e Recall's own activation step does not run](793a_product-spec.md#793a3k1-uat-scenarios) covers it. Scenarios [793a3k1a A note the transcript shows used is activated](793a_product-spec.md#793a3k1-uat-scenarios) and [793a3k1b An injected note with no evidence of use is not activated](793a_product-spec.md#793a3k1-uat-scenarios) now speak of evidence in the transcript.
- **Choice:** grading is a sibling requirement, [793a3l Activation accuracy is graded independently](793a_product-spec.md#793a3l-activation-accuracy-is-graded-independently), not an extension of [793a3i Precision is graded independently](793a_product-spec.md#793a3i-precision-is-graded-independently), which grades captures; the sibling keeps that title accurate. It samples verdicts, grades each against the transcript, and logs each grade, the sample size and the accuracy. The sample size is not fixed.
- **Choice:** [793a4f No recall or capture in its own evaluation sessions](793a_product-spec.md#793a4f-no-recall-or-capture-in-its-own-evaluation-sessions) also covers the sessions that judge activation and grade its accuracy, and [793a5e Auditability and measurability](793a_product-spec.md#793a5e-auditability-and-measurability) logs the evidence with each activation decision.
- **Rationale:** the System has no declaration from the agent, and asking for one would put memory bookkeeping back into the agent's context; inference from the transcript is measurable.
- **Alternatives:** (b) treat every injected note as used, and (c) do not activate at all. Not chosen.

## 793b69 The uninstall command is a separate change

- **Decided by:** Owner (Joe), [793c22 There is no uninstall command today](793c_open-questions.md#793c22-there-is-no-uninstall-command-today), chose (b): a general `engram uninstall` in its own change, [#849](https://github.com/toejough/engram/issues/849).
- **Choice:** this change depends on #849. The uninstall branch of [793a4g Active whenever installed](793a_product-spec.md#793a4g-active-whenever-installed) and scenario [793a4g1c An uninstalled Integration is inert](793a_product-spec.md#793a4g1-uat-scenarios) cannot be verified until #849 ships, and the System cannot be turned off before then. This spec specifies no uninstall behavior beyond "when the Integration has been uninstalled, the System injects, captures and records nothing". #849 is named only in the rationale of [793a4g Active whenever installed](793a_product-spec.md#793a4g-active-whenever-installed) and here, not in normative text.
- **Rationale:** uninstalling is a capability of the install and update, not of this feature, and a general command gives every installed piece a removal path.
- **Alternatives:** (a) a command that removes only the System, and (c) documented manual steps. Not chosen.

## 793b70 Questions raised while applying the answers

- **Decided by:** Author (agent); awaiting owner validation.
- **Choice:** two points were ambiguous and were not decided silently: [793c23 Is a note amended as near injected?](793c_open-questions.md#793c23-is-a-note-amended-as-near-injected) (whether a note amended as near is injected) and [793c24 Does the covering note's injection add anything to the depth's output?](793c_open-questions.md#793c24-does-the-covering-notes-injection-add-anything-to-the-depths-output) (whether the covering note's injection adds anything beyond [793a2j Volume equals the depth's output](793a_product-spec.md#793a2j-volume-equals-the-depths-output)). Both carry a recommendation and block [793a3g Deep recall's writes are the System's](793a_product-spec.md#793a3g-deep-recalls-writes-are-the-systems).
- **Rationale:** the answers to [793c16 At a deep moment, does the System's coverage write activate the note it covers?](793c_open-questions.md#793c16-at-a-deep-moment-does-the-systems-coverage-write-activate-the-note-it-covers) covered the covered outcome only, and the injection clause overlaps an existing rule.

## 793b71 A near-amended note is injected

- **Choice:** [793a3g Deep recall's writes are the System's](793a_product-spec.md#793a3g-deep-recalls-writes-are-the-systems) injects the amended note after a near amend, without activating it. [793a2e No re-injection](793a_product-spec.md#793a2e-no-re-injection) treats a note changed after delivery as not present in its changed form, with new scenario [793a2e1j An amended note is delivered in its changed form](793a_product-spec.md#793a2e1-uat-scenarios). [793a3g1e A near cluster amends the note](793a_product-spec.md#793a3g1-uat-scenarios) asserts the delivery.
- **Rationale:** in recall the agent doing the deep pass writes the amendment, so the amended content is in its context. Injecting it keeps that behavior.
- **Decided by:** Owner (Joe), [793c23 Is a note amended as near injected?](793c_open-questions.md#793c23-is-a-note-amended-as-near-injected); wording and the 793a2e clause by Orchestrator (Claude), awaiting owner validation.

## 793b72 Covering note and cluster are both injected

- **Choice:** keep the explicit covering-note injection in 793a3g, and inject the cluster's items as [793a2j Volume equals the depth's output](793a_product-spec.md#793a2j-volume-equals-the-depths-output) requires. Injecting the covering note in place of its cluster is a trim, tracked in #848.
- **Decided by:** Owner (Joe), [793c24 Does the covering note's injection add anything to the depth's output?](793c_open-questions.md#793c24-does-the-covering-notes-injection-add-anything-to-the-depths-output).

