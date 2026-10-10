# Parallel Memory Agent - Decision Log

These are choices made while relocating the draft requirements into this change (#793, node [792c8b In-flight design relocation](../792_documentation-framework-adoption/792c_implementation-plan.md#792c8b-in-flight-design-relocation) of
#792), recorded for Joe to validate. Questions that need Joe's input are in
[`793c_open-questions.md`](./793c_open-questions.md).

## 793b1 One new capability, `automatic-memory`

- **Choice:** every requirement in [793a Parallel Memory Agent - Product Spec](./793a_product-spec.md#parallel-memory-agent---product-spec) is an Added operation against one new canonical file,
  `docs/specs/product/automatic-memory.md` (title `Automatic Memory - Product Spec`).
- **Rationale:** the feature is one user-facing capability, automatic recall and capture independent
  of the working agent. It reuses behavior of `recall`, `learn` and `ambient-guidance` ([792a14 recall](../792_documentation-framework-adoption/792a_product-spec.md#792a14-added-recall),
  [792a15 learn](../792_documentation-framework-adoption/792a_product-spec.md#792a15-added-learn), [792a19 ambient-guidance](../792_documentation-framework-adoption/792a_product-spec.md#792a19-added-ambient-guidance)) but replaces the agent's own initiative, so it fits none of them.
- **Alternatives:** split across [792a14 recall](../792_documentation-framework-adoption/792a_product-spec.md#792a14-added-recall), [792a15 learn](../792_documentation-framework-adoption/792a_product-spec.md#792a15-added-learn) and [792a19 ambient-guidance](../792_documentation-framework-adoption/792a_product-spec.md#792a19-added-ambient-guidance). Rejected: each requirement
  would then depend on the agent-initiated behavior those capabilities describe.

## 793b2 ID mapping

- **Choice:** the draft's four groups became four Added sections, and each requirement kept its order.

| Draft | Section | Draft IDs to new IDs |
| --- | --- | --- |
| 4.1 Automatic recall | [793a2 automatic-memory (automatic recall)](./793a_product-spec.md#793a2-added-automatic-memory-automatic-recall) | REC-1..REC-7 to [793a2a Recall at a human prompt](./793a_product-spec.md#793a2a-recall-at-a-human-prompt)..[793a2g Human-originated moments only](./793a_product-spec.md#793a2g-human-originated-moments-only) |
| 4.2 Automatic capture | [793a3 automatic-memory (automatic capture)](./793a_product-spec.md#793a3-added-automatic-memory-automatic-capture) | CAP-1..CAP-6 to [793a3a Evaluate every completed turn](./793a_product-spec.md#793a3a-evaluate-every-completed-turn)..[793a3f Non-blocking capture](./793a_product-spec.md#793a3f-non-blocking-capture) |
| 4.3 Operator control and safety | [793a4 automatic-memory (operator control and safety)](./793a_product-spec.md#793a4-added-automatic-memory-operator-control-and-safety) | CTL-1..CTL-4 to [793a4a Operating modes](./793a_product-spec.md#793a4a-operating-modes)..[793a4d Sensitive content](./793a_product-spec.md#793a4d-sensitive-content) |
| 4.4 Independence, coexistence, measurement | [793a5 automatic-memory (independence, coexistence, and measurement)](./793a_product-spec.md#793a5-added-automatic-memory-independence-coexistence-and-measurement) | IND-1..IND-3 to [793a5a Independence from agent compliance](./793a_product-spec.md#793a5a-independence-from-agent-compliance)..[793a5c Environments without the capability](./793a_product-spec.md#793a5c-environments-without-the-capability), NFR-1..NFR-3 to [793a5d Latency](./793a_product-spec.md#793a5d-latency)..[793a5f Effect on outcomes](./793a_product-spec.md#793a5f-effect-on-outcomes) |

- **Rationale:** each requirement keeps "(formerly X)" so the draft's review comments still map.
  Cross-references inside requirements and scenarios were rewritten to the new IDs.

## 793b3 Scenarios added for uncovered MUST branches

- **Choice:** two scenarios were added to [793a2f Bounded volume](./793a_product-spec.md#793a2f-bounded-volume), [793a2f1b Prompt-time bound holds](./793a_product-spec.md#793a2f1-uat-scenarios) (prompt-time bound) and [793a2f1c Tool-result bound holds](./793a_product-spec.md#793a2f1-uat-scenarios)
  (tool-result bound). The draft had a scenario only for the per-turn bound.
- **Rationale:** the product spec guide requires coverage for every distinct mandatory branch. Both
  new scenarios restate the draft's own proposed bounds. No other scenario was added or changed.
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
| Learn fires about 5% | [793a3a Evaluate every completed turn](./793a_product-spec.md#793a3a-evaluate-every-completed-turn), [793a3b Confirmed lessons only](./793a_product-spec.md#793a3b-confirmed-lessons-only), [793a3e Capture quality](./793a_product-spec.md#793a3e-capture-quality), [793a5a Independence from agent compliance](./793a_product-spec.md#793a5a-independence-from-agent-compliance) |
| 0 of 4 subagents received memory | [793a2c Recall at subagent dispatch](./793a_product-spec.md#793a2c-recall-at-subagent-dispatch) |
| Shim refused as injection 6 of 6 | [793a2d Informational, attributable presentation](./793a_product-spec.md#793a2d-informational-attributable-presentation), [793a5a Independence from agent compliance](./793a_product-spec.md#793a5a-independence-from-agent-compliance) |
| Prose fixes plateau | [793a5a Independence from agent compliance](./793a_product-spec.md#793a5a-independence-from-agent-compliance), [793a5f Effect on outcomes](./793a_product-spec.md#793a5f-effect-on-outcomes) |
| Past noise and over-firing concerns | [793a2e Relevance and non-repetition](./793a_product-spec.md#793a2e-relevance-and-non-repetition), [793a2f Bounded volume](./793a_product-spec.md#793a2f-bounded-volume), [793a2g Human-originated moments only](./793a_product-spec.md#793a2g-human-originated-moments-only), [793a5d Latency](./793a_product-spec.md#793a5d-latency) |
| Safety of an unattended process | [793a4a Operating modes](./793a_product-spec.md#793a4a-operating-modes), [793a4b Failure isolation](./793a_product-spec.md#793a4b-failure-isolation), [793a4c Memory purview only](./793a_product-spec.md#793a4c-memory-purview-only), [793a4d Sensitive content](./793a_product-spec.md#793a4d-sensitive-content), [793a5e Auditability and measurability](./793a_product-spec.md#793a5e-auditability-and-measurability) |

## 793b6 Values stay proposed; scenarios kept as drafted

- **Choice:** every threshold the draft marked [proposed] is still marked so ([793a2d1a Injected content is attributable and non-imperative](./793a_product-spec.md#793a2d1-uat-scenarios), [793a2f Bounded volume](./793a_product-spec.md#793a2f-bounded-volume), [793a3e Capture quality](./793a_product-spec.md#793a3e-capture-quality),
  [793a5d Latency](./793a_product-spec.md#793a5d-latency), [793a5f1a Treatment beats control](./793a_product-spec.md#793a5f1-uat-scenarios)). Scenario wording is the draft's, including the few that measure rates over many
  runs or inspect the decision log ([793a2b1b Every post-tool model call is covered](./793a_product-spec.md#793a2b1-uat-scenarios), [793a5e1a Decision log joins to outcomes](./793a_product-spec.md#793a5e1-uat-scenarios), [793a5f1a Treatment beats control](./793a_product-spec.md#793a5f1-uat-scenarios)). The draft's "dry-run log" in
  [793a3a1a User correction is captured without agent action](./793a_product-spec.md#793a3a1-uat-scenarios) stays.
- **Rationale:** the guide forbids exceeding the behavior the user specified. Whether those scenarios
  are end-user verifiable is a review question for Joe, together with [793c1 What are the capture precision bar and capture-rate target?](./793c_open-questions.md#793c1-what-are-the-capture-precision-bar-and-capture-rate-target) to [793c6 Does capture also run inside subagent sessions?](./793c_open-questions.md#793c6-does-capture-also-run-inside-subagent-sessions).
