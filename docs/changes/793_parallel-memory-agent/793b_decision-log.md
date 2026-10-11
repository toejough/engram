# Parallel Memory Agent - Decision Log

These are choices made while relocating the draft requirements into this change (#793, node In-flight design relocation of
#792), recorded for Joe to validate. Questions that need Joe's input are in
[`793c_open-questions.md`](./793c_open-questions.md).

## 793b1 One new capability, `automatic-memory`

- **Choice:** every requirement in [793a Parallel Memory Agent - Product Spec](./793a_product-spec.md#parallel-memory-agent---product-spec) is an Added operation against one new canonical file,
  `docs/specs/product/automatic-memory.md` (title `Automatic Memory - Product Spec`).
- **Rationale:** the feature is one user-facing capability, automatic recall and capture independent
  of the working agent. It reuses behavior of `recall`, `learn` and `ambient-guidance` ([recall](../../specs/product/recall.md),
  [learn](../../specs/product/learn.md), [ambient-guidance](../../specs/product/ambient-guidance.md)) but replaces the agent's own initiative, so it fits none of them.
- **Alternatives:** split across [recall](../../specs/product/recall.md), [learn](../../specs/product/learn.md) and [ambient-guidance](../../specs/product/ambient-guidance.md). Rejected: each requirement
  would then depend on the agent-initiated behavior those capabilities describe.

## 793b2 ID mapping

- **Choice:** the draft's four groups became four Added sections, and each requirement kept its order.

| Draft | Section | Draft IDs to new IDs |
| --- | --- | --- |
| 4.1 Automatic recall | [793a2 automatic-memory (automatic recall)](./793a_product-spec.md#793a2-added-automatic-memory-automatic-recall) | REC-1..REC-7 to [793a2a Recall at a human prompt](./793a_product-spec.md#793a2a-recall-at-a-human-prompt)..[793a2g Recall for every prompt regardless of source](./793a_product-spec.md#793a2g-recall-for-every-prompt-regardless-of-source) |
| 4.2 Automatic capture | [793a3 automatic-memory (automatic capture)](./793a_product-spec.md#793a3-added-automatic-memory-automatic-capture) | CAP-1..CAP-6 to [793a3a Evaluate every completed turn](./793a_product-spec.md#793a3a-evaluate-every-completed-turn)..[793a3f Non-blocking capture](./793a_product-spec.md#793a3f-non-blocking-capture) |
| 4.3 Operator control and safety | [793a4 automatic-memory (operator control and safety)](./793a_product-spec.md#793a4-added-automatic-memory-operator-control-and-safety) | CTL-1..CTL-4 to [793a4a Operating modes](./793a_product-spec.md#793a4a-operating-modes)..[793a4d Sensitive content](./793a_product-spec.md#793a4d-sensitive-content) |
| 4.4 Independence, coexistence, measurement | [793a5 automatic-memory (independence, coexistence, and measurement)](./793a_product-spec.md#793a5-added-automatic-memory-independence-coexistence-and-measurement) | IND-1..IND-3 to [793a5a Independence from agent compliance](./793a_product-spec.md#793a5a-independence-from-agent-compliance)..[793a5c Environments without the capability](./793a_product-spec.md#793a5c-environments-without-the-capability), NFR-1..NFR-3 to [793a5d Latency](./793a_product-spec.md#793a5d-latency)..[793a5f Effect on outcomes](./793a_product-spec.md#793a5f-effect-on-outcomes) |

- **Rationale:** each requirement keeps "(formerly X)" so the draft's review comments still map.
  Cross-references inside requirements and scenarios were rewritten to the new IDs.

## 793b3 Scenarios added for uncovered MUST branches

- **Choice:** two scenarios were added to [793a2f Bounded volume](./793a_product-spec.md#793a2f-bounded-volume), [793a2f1b Prompt-time bound holds](./793a_product-spec.md#793a2f1-uat-scenarios) (prompt-time bound) and [793a2f1c Tool-result bound holds](./793a_product-spec.md#793a2f1-uat-scenarios)
  (tool-result bound). The draft had a scenario only for the per-turn bound.
- **Rationale:** the product spec guide requires coverage for every distinct mandatory branch. Both
  new scenarios restated the draft's own proposed bounds, until [793b7 Volume and latency bounds reference the current recall query](#793b7-volume-and-latency-bounds-reference-the-current-recall-query) replaced them. No other scenario was added or changed.
- **Alternatives:** port the draft's scenarios unchanged. Rejected: two stated bounds would have no
  UAT.

## 793b4 Draft sections that are not requirements

- **Choice:**
  - **Terms (draft §2)** became concise context in the problem statement ([793a1 Problem Statement](./793a_product-spec.md#793a1-problem-statement)).
  - **Scope (draft §3)** is not kept as a section, because the guide allows only the problem
    statement, requirements and UAT. The in-scope items are the requirements themselves. The enduring
    boundary, no effect on anything but injected reference notes and vault writes, is already the
    MUST NOT in [793a4c Memory purview only](./793a_product-spec.md#793a4c-memory-purview-only). The remaining out-of-scope items are recorded here as context: changing what
    counts as a lesson, changing note format or retrieval ranking, memory-unrelated behavior (task
    decisions, permissions, model choice, code review), and designing the solution. They bound this
    change, so they are not promoted as requirements.
  - **Acknowledged coverage limits (draft §5)** are context, not requirements:
    - A turn where the working agent produces a final text-only answer with no further model call has
      no later moment for recall. Capture of that turn still applies.
    - Some session types (forked sessions, subtasks, teammate agents, cloud or diskless sessions,
      other harnesses) may not present all moments. Requirements apply to the moments the
      environment exposes.
    - The mechanism may be unavailable by remote rollout decision. [793a5c Environments without the capability](./793a_product-spec.md#793a5c-environments-without-the-capability) covers it.
    The solution document must state how each is handled or accepted.
  - **Status and solution boundary.** The draft was marked DRAFT, with problem and verifiable
    requirements only. The solution document is still to be written after approval.
- **Rationale:** `docs/PRODUCT-SPEC-GUIDE.md` and the lifecycle guide keep change context out of the
  product specification.

## 793b5 Traceability from the draft (evidence to requirement)

| Problem evidence | Requirements |
| --- | --- |
| Learn fires about 5% | [793a3a Evaluate every completed turn](./793a_product-spec.md#793a3a-evaluate-every-completed-turn), [793a3b Confirmed lessons only](./793a_product-spec.md#793a3b-confirmed-lessons-only), [793a3e Capture rate and recorded precision](./793a_product-spec.md#793a3e-capture-rate-and-recorded-precision), [793a5a Independence from agent compliance](./793a_product-spec.md#793a5a-independence-from-agent-compliance) |
| 0 of 4 subagents received memory | [793a2c Recall at subagent dispatch](./793a_product-spec.md#793a2c-recall-at-subagent-dispatch) |
| Shim refused as injection 6 of 6 | [793a2d Informational, attributable presentation](./793a_product-spec.md#793a2d-informational-attributable-presentation), [793a5a Independence from agent compliance](./793a_product-spec.md#793a5a-independence-from-agent-compliance) |
| Prose fixes plateau | [793a5a Independence from agent compliance](./793a_product-spec.md#793a5a-independence-from-agent-compliance), [793a5f Effect on outcomes](./793a_product-spec.md#793a5f-effect-on-outcomes) |
| Past noise and over-firing concerns | [793a2e Relevance and non-repetition](./793a_product-spec.md#793a2e-relevance-and-non-repetition), [793a2f Bounded volume](./793a_product-spec.md#793a2f-bounded-volume), [793a2g Recall for every prompt regardless of source](./793a_product-spec.md#793a2g-recall-for-every-prompt-regardless-of-source), [793a5d Latency](./793a_product-spec.md#793a5d-latency) |
| Safety of an unattended process | [793a4a Operating modes](./793a_product-spec.md#793a4a-operating-modes), [793a4b Failure isolation](./793a_product-spec.md#793a4b-failure-isolation), [793a4c Memory purview only](./793a_product-spec.md#793a4c-memory-purview-only), [793a4d Sensitive content](./793a_product-spec.md#793a4d-sensitive-content), [793a5e Auditability and measurability](./793a_product-spec.md#793a5e-auditability-and-measurability) |

## 793b6 Remaining [proposed] values; scenarios kept as drafted

- **Choice:** only two values are still marked [proposed]: the 95% / N=20 refusal-rate floor in
  [793a2d1a Injected content is attributable and non-imperative](./793a_product-spec.md#793a2d1-uat-scenarios) and N=20 in [793a5f1a Treatment beats control](./793a_product-spec.md#793a5f1-uat-scenarios). Joe's answers ([793c1 What are the capture precision bar and capture-rate target?](./793c_open-questions.md#793c1-what-are-the-capture-precision-bar-and-capture-rate-target) to [793c6 Does capture also run inside subagent sessions?](./793c_open-questions.md#793c6-does-capture-also-run-inside-subagent-sessions)) replaced the others. Scenario wording is the draft's
  except where those answers changed it, including the few that measure rates over many runs or
  inspect the decision log ([793a2b1b Every post-tool model call is covered](./793a_product-spec.md#793a2b1-uat-scenarios), [793a5e1a Decision log joins to outcomes](./793a_product-spec.md#793a5e1-uat-scenarios), [793a5f1a Treatment beats control](./793a_product-spec.md#793a5f1-uat-scenarios)).
- **Rationale:** the guide forbids exceeding the behavior the user specified. Whether those scenarios
  are end-user verifiable is a review question for Joe.

## 793b7 Volume and latency bounds reference the current recall query

- **Choice:** [793a2f Bounded volume](./793a_product-spec.md#793a2f-bounded-volume) and [793a5d Latency](./793a_product-spec.md#793a5d-latency) state each bound as "not lower than what one run of the current
  recall query delivers under its defaults", and the scenarios compare against the documented bound and that
  reference. No KB, item-count or seconds value appears in normative text.
- **Rationale:** a fixed number would go stale when the query's defaults or the vault change, while the
  reference stays true and an end user can verify it by running the query. Reference measurement,
  2026-10-11 on Joe's vault: about 135 KB, 45 items, 10.6-11.0 s per query, with `engram query` defaults
  `--limit 20` and content budget 15.
- **Choice:** the per-turn and per-session bounds, which the current query has no equivalent of, are held to the same floor
  (each at least one query's worth) and are otherwise left to the solution.
- **Alternatives:** hard-code 135 KB / 45 items / 11 s. Rejected: stale on the next vault change.

## 793b8 Per-moment record sits in the existing log requirement

- **Choice:** the volume and latency record is added to [793a5e Auditability and measurability](./793a_product-spec.md#793a5e-auditability-and-measurability), with scenario [793a5e1b Volume and latency recorded per moment](./793a_product-spec.md#793a5e1-uat-scenarios), not a new requirement.
- **Rationale:** it is one more field of the decision log that requirement already mandates.
- **Choice:** "any reduction of the bounds requires that data" is not normative. It is tracked as
  follow-up #845 and recorded here.

## 793b9 Capture rate comparator and precision recording

- **Choice:** [793a3e Capture rate and recorded precision](./793a_product-spec.md#793a3e-capture-rate-and-recorded-precision) is retitled (same ID). The comparator in scenario [793a3e1b Capture rate exceeds the baseline](./793a_product-spec.md#793a3e1-uat-scenarios) is 5.6%, the
  higher of the two measured baselines (4/71 = 5.6%, 3/66 = 4.5%; LEDGER `739-audit-w2-learn-fired`,
  `learn-rate-remeasure-w2`), with no required margin. The 60-moment labeled set is kept from the draft.
- **Choice:** precision has no threshold; scenario [793a3e1a Precision recorded on a sample](./793a_product-spec.md#793a3e1-uat-scenarios) checks that verdicts, sample size and the
  resulting precision are recorded. The 30-note sample is kept from the draft.
- **Rationale:** follows Joe's answer ([793c1 What are the capture precision bar and capture-rate target?](./793c_open-questions.md#793c1-what-are-the-capture-precision-bar-and-capture-rate-target)): beat the baseline, measure precision, set no bar.

## 793b10 Default mode and always-on measurement

- **Choice:** [793a4a Operating modes](./793a_product-spec.md#793a4a-operating-modes) changes the default from SHOULD observe to MUST active, and requires measurement to be recorded
  in observe and active modes. Off stays inert, as [793a4a1b Off mode is inert](./793a_product-spec.md#793a4a1-uat-scenarios) already says (no recall, no capture, no
  log entries). Scenario [793a4a1c Fresh install is active and measured](./793a_product-spec.md#793a4a1-uat-scenarios) is new.
- **Rationale:** "measurement always on" is read as: not separately switchable from the mode, and
  present whenever the System does anything. An off System has nothing to measure. Raised as an ambiguity for Joe.

## 793b11 Direct capture: where it is stated

- **Choice:** the direct-write rule and the System-judges rule are stated in [793a3a Evaluate every completed turn](./793a_product-spec.md#793a3a-evaluate-every-completed-turn) (a MUST on judging and on
  writing directly). Scenarios [793a3a1a User correction is captured without agent action](./793a_product-spec.md#793a3a1-uat-scenarios), [793a3a1b Explicit save-request](./793a_product-spec.md#793a3a1-uat-scenarios) and [793a3d1a Provenance present](./793a_product-spec.md#793a3d1-uat-scenarios) now assert the vault, and the "dry-run log"
  alternative is removed. Scenario [793a3a1a User correction is captured without agent action](./793a_product-spec.md#793a3a1-uat-scenarios) also asserts no pending offer and no
  evaluation request in the agent's context.
- **Rationale:** those assertions are what "direct, without polluting the main agent's context"
  means to an end user. Observe-mode logging remains in [793a4a1a Observe mode has no side effects](./793a_product-spec.md#793a4a1-uat-scenarios).

## 793b12 Recall for every prompt

- **Choice:** [793a2g Recall for every prompt regardless of source](./793a_product-spec.md#793a2g-recall-for-every-prompt-regardless-of-source) is retitled (same ID, Modified). The new MUST NOT on re-injecting a note already
  in the agent's context lives there, because [793a2e Relevance and non-repetition](./793a_product-spec.md#793a2e-relevance-and-non-repetition) only has a SHOULD NOT about notes this System injected earlier
  and does not cover notes the agent got by other means. [793a2e Relevance and non-repetition](./793a_product-spec.md#793a2e-relevance-and-non-repetition) is unchanged.
- **Choice:** [793a2a Recall at a human prompt](./793a_product-spec.md#793a2a-recall-at-a-human-prompt) is unchanged; it is the human case of the rule. Scenario [793a2g1a Non-human prompt receives recall](./793a_product-spec.md#793a2g1-uat-scenarios) (same ID, reversed
  meaning: skip became inject) is a Scenario Outline over the three non-human sources the old requirement
  named. [793a2g1b Note injected earlier is not injected again](./793a_product-spec.md#793a2g1-uat-scenarios) and [793a2g1c Note the agent already retrieved is not injected](./793a_product-spec.md#793a2g1-uat-scenarios) are new.
- **Rationale:** the three sources behave identically, so one outline suffices.

## 793b13 Capture in subagent sessions

- **Choice:** [793a3a Evaluate every completed turn](./793a_product-spec.md#793a3a-evaluate-every-completed-turn) names "both top-level and subagent sessions". Scenario [793a3a1c Lesson in a subagent session is captured](./793a_product-spec.md#793a3a1-uat-scenarios) uses an overturned conclusion
  as the lesson, since a subagent session has no human user to correct it.
- **Rationale:** the four lesson kinds apply to a subagent's work as to the top level's.

## 793b14 ID handling

- **Choice:** no ID was removed or reused. Retitled with the same ID: [793a2g Recall for every prompt regardless of source](./793a_product-spec.md#793a2g-recall-for-every-prompt-regardless-of-source), [793a3e Capture rate and recorded precision](./793a_product-spec.md#793a3e-capture-rate-and-recorded-precision), and scenarios
  [793a2g1a Non-human prompt receives recall](./793a_product-spec.md#793a2g1-uat-scenarios), [793a3e1a Precision recorded on a sample](./793a_product-spec.md#793a3e1-uat-scenarios), [793a3e1b Capture rate exceeds the baseline](./793a_product-spec.md#793a3e1-uat-scenarios). New scenarios: [793a2g1b Note injected earlier is not injected again](./793a_product-spec.md#793a2g1-uat-scenarios), [793a2g1c Note the agent already retrieved is not injected](./793a_product-spec.md#793a2g1-uat-scenarios), [793a3a1c Lesson in a subagent session is captured](./793a_product-spec.md#793a3a1-uat-scenarios), [793a4a1c Fresh install is active and measured](./793a_product-spec.md#793a4a1-uat-scenarios),
  [793a5e1b Volume and latency recorded per moment](./793a_product-spec.md#793a5e1-uat-scenarios).
- **Rationale:** each requirement keeps its subject, so each is a modification, per the lifecycle guide.
