# Parallel Memory Agent - Decision Log

These are choices made while relocating the draft requirements into this change (#793, node In-flight design relocation of
#792) and while applying Joe's review answers, recorded for Joe to validate. Questions that need Joe's input are in
[`793c_open-questions.md`](./793c_open-questions.md).

## 793b1 One new capability, `automatic-memory`

- **Choice:** every requirement in [Parallel Memory Agent - Product Spec](./793a_product-spec.md#parallel-memory-agent---product-spec) is an Added operation against one new canonical file,
  `docs/specs/product/automatic-memory.md` (title `Automatic Memory - Product Spec`).
- **Rationale:** the feature is one user-facing capability, automatic recall and capture independent
  of the working agent. It reuses behavior of `recall`, `learn` and `ambient-guidance` ([recall](../../specs/product/recall.md),
  [learn](../../specs/product/learn.md), [ambient-guidance](../../specs/product/ambient-guidance.md)) but replaces the agent's own initiative, so it fits none of them.
- **Alternatives:** split across [recall](../../specs/product/recall.md), [learn](../../specs/product/learn.md) and [ambient-guidance](../../specs/product/ambient-guidance.md). Rejected: each requirement
  would then depend on the agent-initiated behavior those capabilities describe.

## 793b2 ID mapping

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

- **Choice:** every MUST and MUST NOT branch has a scenario. Added in this revision: the in-context
  branches of [793a2e Relevance and no re-injection](793a_product-spec.md#793a2e-relevance-and-no-re-injection) (retrieved by the agent, parent versus subagent, compaction, truncation,
  resume), the depth branches of [793a2h Recall depth per moment](793a_product-spec.md#793a2h-recall-depth-per-moment), the direct-write branches of [793a3j Direct writes go to the local vault](793a_product-spec.md#793a3j-direct-writes-go-to-the-local-vault), the concurrent
  and parent-reported duplicates of [793a3c No duplicates](793a_product-spec.md#793a3c-no-duplicates), the routine-success exclusion of [793a3b Confirmed lessons only](793a_product-spec.md#793a3b-confirmed-lessons-only), the
  no-error-to-the-agent branch of [793a4b Failure isolation](793a_product-spec.md#793a4b-failure-isolation), and a scenario per mode and per measurement branch of
  [793a4e Operating modes and measurement](793a_product-spec.md#793a4e-operating-modes-and-measurement).
- **Rationale:** the product spec guide requires coverage for every distinct mandatory branch.
- **Alternatives:** one scenario per requirement. Rejected: the guide calls that insufficient.

## 793b4 Draft sections that are not requirements

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

| Problem evidence | Requirements |
| --- | --- |
| Learn fires about 5% | [793a3a Evaluate every completed turn](793a_product-spec.md#793a3a-evaluate-every-completed-turn), [793a3b Confirmed lessons only](793a_product-spec.md#793a3b-confirmed-lessons-only), [793a3h Capture rate exceeds the baseline](793a_product-spec.md#793a3h-capture-rate-exceeds-the-baseline), [793a3i Precision is graded independently](793a_product-spec.md#793a3i-precision-is-graded-independently), [793a5a Independence from agent compliance](793a_product-spec.md#793a5a-independence-from-agent-compliance) |
| 0 of 4 subagents received memory | [793a2c Recall at subagent dispatch](793a_product-spec.md#793a2c-recall-at-subagent-dispatch) |
| Shim refused as injection 6 of 6 | [793a2d Attributable presentation](793a_product-spec.md#793a2d-attributable-presentation), [793a5a Independence from agent compliance](793a_product-spec.md#793a5a-independence-from-agent-compliance) |
| Prose fixes plateau | [793a5a Independence from agent compliance](793a_product-spec.md#793a5a-independence-from-agent-compliance), [793a5f Effect on outcomes](793a_product-spec.md#793a5f-effect-on-outcomes) |
| Past noise and over-firing concerns | [793a2e Relevance and no re-injection](793a_product-spec.md#793a2e-relevance-and-no-re-injection), [793a2h Recall depth per moment](793a_product-spec.md#793a2h-recall-depth-per-moment), [793a2i Recall for every prompt regardless of source](793a_product-spec.md#793a2i-recall-for-every-prompt-regardless-of-source), [793a2j Volume equals the depth's output](793a_product-spec.md#793a2j-volume-equals-the-depths-output), [793a5g Latency per recall moment](793a_product-spec.md#793a5g-latency-per-recall-moment) |
| Safety of an unattended process | [793a4b Failure isolation](793a_product-spec.md#793a4b-failure-isolation), [793a4c Memory purview only](793a_product-spec.md#793a4c-memory-purview-only), [793a4d Sensitive content](793a_product-spec.md#793a4d-sensitive-content), [793a4e Operating modes and measurement](793a_product-spec.md#793a4e-operating-modes-and-measurement), [793a4f No recall or capture in its own evaluation sessions](793a_product-spec.md#793a4f-no-recall-or-capture-in-its-own-evaluation-sessions), [793a5e Auditability and measurability](793a_product-spec.md#793a5e-auditability-and-measurability) |

## 793b6 No proposed values remain

- **Choice:** [793a5f Effect on outcomes](793a_product-spec.md#793a5f-effect-on-outcomes) keeps N = 20 runs per arm, accepted by Joe, and loses its proposed marker. The
  95% / N=20 refusal-rate floor was removed with the refusal check ([793b15 Attributable presentation only](#793b15-attributable-presentation-only)).
- **Rationale:** the guide forbids exceeding the behavior the user specified, and Joe accepted the one
  remaining value.

## 793b7 Volume and latency equal the chosen depth's output

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

- **Choice:** the volume and latency record is part of [793a5e Auditability and measurability](793a_product-spec.md#793a5e-auditability-and-measurability), with scenarios [793a5e1b Volume and latency recorded per moment](793a_product-spec.md#793a5e1-uat-scenarios) and
  [793a5e1c A moment with no injection is recorded](793a_product-spec.md#793a5e1-uat-scenarios), not a new requirement. The log is available by session id, each entry carries session
  id, turn id and moment, and a moment with no injection is recorded with volume 0.
- **Choice:** the log also records the depth and the phrases queried, so that [793a2h1a A routine decision uses glance](793a_product-spec.md#793a2h1-uat-scenarios) and
  [793a2j1a Delivered notes equal the depth's output](793a_product-spec.md#793a2j1-uat-scenarios) can be checked from the operator-accessible log. This goes beyond Joe's wording and is
  raised in [793c7 Review answers (2026-10-11)](793c_open-questions.md#793c7-review-answers-2026-10-11).
- **Rationale:** it is one more field of the decision log that requirement already mandates.

## 793b9 Capture rate and precision are separate requirements

- **Choice:** [793a3h Capture rate exceeds the baseline](793a_product-spec.md#793a3h-capture-rate-exceeds-the-baseline) requires the capture rate to exceed 5.6% on the point estimate, with no
  margin; the 60-moment labeled set is kept from the draft. [793a3i Precision is graded independently](793a_product-spec.md#793a3i-precision-is-graded-independently) requires a separate grader inside
  the System, independent of the capture judgment, and reports precision with its sample size; the
  30-note sample is kept. Neither has a precision threshold.
- **Rationale:** 5.6% is the higher of the two measured baselines (4/71 = 5.6%, 3/66 = 4.5%; LEDGER
  `739-audit-w2-learn-fired`, `learn-rate-remeasure-w2`). Splitting rate from precision gives each one
  statement and its own scenarios; the grader keeps the capture judge from marking its own work.

## 793b10 Default mode and measurement tied to mode

- **Choice:** [793a4e Operating modes and measurement](793a_product-spec.md#793a4e-operating-modes-and-measurement) makes the default MUST active, and requires measurement whenever the mode is
  observe or active, not switchable off independently. Off records nothing, so scenario [793a4e1b Off mode is inert](793a_product-spec.md#793a4e1-uat-scenarios) asserts
  no log entry. Scenario [793a4e1e Measurement cannot be switched off apart from the mode](793a_product-spec.md#793a4e1-uat-scenarios) covers an attempt to switch measurement off apart from the mode.
- **Rationale:** an off System has nothing to measure; observing and active both produce decisions to
  record.

## 793b11 Direct capture: where it is stated

- **Choice:** the direct-write rule and the System-judges rule are stated in [793a3a Evaluate every completed turn](793a_product-spec.md#793a3a-evaluate-every-completed-turn). Scenarios
  [793a3a1a User correction is captured without agent action](793a_product-spec.md#793a3a1-uat-scenarios) and [793a3a1b Explicit save-request](793a_product-spec.md#793a3a1-uat-scenarios) assert the vault, not a dry-run log. [793a3a1a User correction is captured without agent action](793a_product-spec.md#793a3a1-uat-scenarios) also asserts, from the
  transcript, that the agent invoked no learn command and saw no request to evaluate or write.
  Observe-mode logging remains in [793a4e1a Observe mode has no side effects](793a_product-spec.md#793a4e1-uat-scenarios).
- **Rationale:** those assertions are what "direct, without polluting the main agent's context" means
  to an end user.

## 793b12 Recall for every prompt

- **Choice:** [793a2i Recall for every prompt regardless of source](793a_product-spec.md#793a2i-recall-for-every-prompt-regardless-of-source) replaces the retired `793a2g` and keeps only the source-independence MUST. The
  in-context MUST NOT moved to [793a2e Relevance and no re-injection](793a_product-spec.md#793a2e-relevance-and-no-re-injection), which now owns "never inject a note already in the receiving
  agent's context". [793a2a Recall at a human prompt](793a_product-spec.md#793a2a-recall-at-a-human-prompt) is unchanged; it is the human case of the rule.
- **Choice:** the scenario is a Scenario Outline over the three non-human sources the old requirement
  named.
- **Rationale:** the three sources behave identically, so one outline suffices; keeping the
  in-context rule in one requirement avoids a duplicate rule.

## 793b13 Capture in subagent sessions

- **Choice:** [793a3a Evaluate every completed turn](793a_product-spec.md#793a3a-evaluate-every-completed-turn) names "both top-level and subagent sessions". Scenarios [793a3a1c A subagent's own guess overturned by evidence is captured](793a_product-spec.md#793a3a1-uat-scenarios) (the
  subagent's own overturned guess), [793a3a1d A correction from the dispatching agent is captured](793a_product-spec.md#793a3a1-uat-scenarios) (a correction from the dispatching agent) and
  [793a3a1e Capture survives the subagent's context being discarded](793a_product-spec.md#793a3a1-uat-scenarios) (capture survives the discarded context) cover them; [793a3c1b A lesson reported to the parent yields one note](793a_product-spec.md#793a3c1-uat-scenarios) covers a lesson also
  reported to the parent.
- **Rationale:** the lesson kinds apply to a subagent's work as to the top level's.

## 793b14 ID handling

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

- **Choice:** [793a2d Attributable presentation](793a_product-spec.md#793a2d-attributable-presentation) keeps only "injected notes MUST identify their source note" and is retitled
  "Attributable presentation". The non-imperative rule and the 95%-of-20-runs refusal scenario are
  removed. Imperative memory such as runbooks and skills is allowed.
- **Rationale:** [793a5f Effect on outcomes](793a_product-spec.md#793a5f-effect-on-outcomes), whether injected memory gets applied, is the real test.

## 793b16 Observable scenarios

- **Choice:** scenarios assert what the session transcript shows (note content delivered before the
  agent's first response or tool call, or a subagent's transcript) or what the operator-accessible log
  shows. None asserts an internal score or "context". The term Context stays in requirements, where it
  defines presence.
- **Choice:** `793a2f`'s retired scenarios are replaced by [793a2j1b A vault holding more than 20 notes about the topic](793a_product-spec.md#793a2j1-uat-scenarios), which sets up "a vault holding
  more than 20 notes about the prompt's topic" and asserts only what the requirement states.
- **Rationale:** the product spec guide requires scenarios an end user can verify.

## 793b17 Lesson is wider than learn's definition

- **Choice:** the term Lesson in [793a1 Problem Statement](793a_product-spec.md#793a1-problem-statement) means any corrected failure, including an agent's own
  mid-task mistake overturned by evidence, plus corrections, explicit save-requests and confirmed
  approaches, and links [792a15e The four kinds of explicit lesson](../../specs/product/learn.md#792a15e-the-four-kinds-of-explicit-lesson). [793a3b Confirmed lessons only](793a_product-spec.md#793a3b-confirmed-lessons-only) keeps excluding unconfirmed hypotheses, routine successes
  and pleasantries. The rationale in [793a3a Evaluate every completed turn](793a_product-spec.md#793a3a-evaluate-every-completed-turn) says it is wider than learn's current definition and
  that #846 tracks widening learn's.
- **Choice:** the earlier "test overturns the subagent's own guess" scenario stays as [793a3a1c A subagent's own guess overturned by evidence is captured](793a_product-spec.md#793a3a1-uat-scenarios).
- **Rationale:** a mistake the agent catches and fixes is as reusable as one the user catches.

## 793b18 Edge cases

- **Choice:** the System never recalls into or captures from its own evaluation sessions ([793a4f No recall or capture in its own evaluation sessions](793a_product-spec.md#793a4f-no-recall-or-capture-in-its-own-evaluation-sessions));
  two subagents capturing the same lesson concurrently yield one note ([793a3c1c Two subagents capture the same lesson concurrently](793a_product-spec.md#793a3c1-uat-scenarios)); a resumed
  session's restored content counts as present ([793a2e1g A resumed session's restored content counts as present](793a_product-spec.md#793a2e1-uat-scenarios)); direct writes go to the local vault with
  parent offers following normal note-writing behavior, citing [parent-sync](../../specs/product/parent-sync.md) and [792a21n A local write is offered to the parent](../../specs/product/parent-sync.md#792a21n-a-local-write-is-offered-to-the-parent)
  ([793a3j Direct writes go to the local vault](793a_product-spec.md#793a3j-direct-writes-go-to-the-local-vault)); secrets are a MUST NOT ([793a4d Sensitive content](793a_product-spec.md#793a4d-sensitive-content)).
- **Rationale:** each is a boundary Joe asked to pin down in review.

## 793b19 Capture does not delay the session

- **Choice:** [793a3f Capture does not delay the session](793a_product-spec.md#793a3f-capture-does-not-delay-the-session) is retitled "Capture does not delay the session" and states the only capture
  latency rule: capture MUST NOT delay the working agent's response or the processing of the next
  prompt. The overlapping capture clause was removed from [793a5g Latency per recall moment](793a_product-spec.md#793a5g-latency-per-recall-moment), which now covers recall latency
  only. [793a4b Failure isolation](793a_product-spec.md#793a4b-failure-isolation) also requires the failure to be logged for the operator.
- **Rationale:** one rule, in one place.

## 793b20 Deep recall's writes and no self-links

- **Choice:** [793a3g Deep recall's writes are the System's](793a_product-spec.md#793a3g-deep-recalls-writes-are-the-systems) makes the System's capture, not the main agent, perform deep recall's coverage
  writes and persisted conclusions. `Requirement:` lines no longer link themselves; the canonical
  specs' gaps are tracked in #847.
- **Rationale:** Joe's decisions; a definition is not a reference.
