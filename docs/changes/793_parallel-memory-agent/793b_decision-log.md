# Parallel Memory Agent - Decision Log

These are choices made while relocating the draft requirements into this change (#793, node 792c8b of
#792), recorded for Joe to validate. Questions that need Joe's input are in
[`793c_open-questions.md`](./793c_open-questions.md).

## 793b1 One new capability, `automatic-memory`

- **Choice:** every requirement in 793a is an Added operation against one new canonical file,
  `docs/specs/product/automatic-memory.md` (title `Automatic Memory - Product Spec`).
- **Rationale:** the feature is one user-facing capability, automatic recall and capture independent
  of the working agent. It reuses behavior of `recall`, `learn` and `ambient-guidance` (792a14,
  792a15, 792a19) but replaces the agent's own initiative, so it fits none of them.
- **Alternatives:** split across `recall`, `learn` and `ambient-guidance`. Rejected: each requirement
  would then depend on the agent-initiated behavior those capabilities describe.

## 793b2 ID mapping

- **Choice:** the draft's four groups became four Added sections, and each requirement kept its order.

| Draft | Section | Draft IDs to new IDs |
| --- | --- | --- |
| 4.1 Automatic recall | 793a2 | REC-1..REC-7 to 793a2a..793a2g |
| 4.2 Automatic capture | 793a3 | CAP-1..CAP-6 to 793a3a..793a3f |
| 4.3 Operator control and safety | 793a4 | CTL-1..CTL-4 to 793a4a..793a4d |
| 4.4 Independence, coexistence, measurement | 793a5 | IND-1..IND-3 to 793a5a..793a5c, NFR-1..NFR-3 to 793a5d..793a5f |

- **Rationale:** each requirement keeps "(formerly X)" so the draft's review comments still map.
  Cross-references inside requirements and scenarios were rewritten to the new IDs.

## 793b3 Scenarios added for uncovered MUST branches

- **Choice:** two scenarios were added to 793a2f, 793a2f1b (prompt-time bound) and 793a2f1c
  (tool-result bound). The draft had a scenario only for the per-turn bound.
- **Rationale:** the product spec guide requires coverage for every distinct mandatory branch. Both
  new scenarios restate the draft's own proposed bounds. No other scenario was added or changed.
- **Alternatives:** port the draft's scenarios unchanged. Rejected: two stated bounds would have no
  UAT.

## 793b4 Draft sections that are not requirements

- **Choice:**
  - **Terms (draft §2)** became concise context in the problem statement (793a1).
  - **Scope (draft §3)** is not kept as a section, because the guide allows only the problem
    statement, requirements and UAT. The in-scope items are the requirements themselves. The enduring
    boundary, no effect on anything but injected reference notes and vault writes, is already the
    MUST NOT in 793a4c. The remaining out-of-scope items are recorded here as context: changing what
    counts as a lesson, changing note format or retrieval ranking, memory-unrelated behavior (task
    decisions, permissions, model choice, code review), and designing the solution. They bound this
    change, so they are not promoted as requirements.
  - **Acknowledged coverage limits (draft §5)** are context, not requirements:
    - A turn where the working agent produces a final text-only answer with no further model call has
      no later moment for recall. Capture of that turn still applies.
    - Some session types (forked sessions, subtasks, teammate agents, cloud or diskless sessions,
      other harnesses) may not present all moments. Requirements apply to the moments the
      environment exposes.
    - The mechanism may be unavailable by remote rollout decision. 793a5c covers it.
    The solution document must state how each is handled or accepted.
  - **Status and solution boundary.** The draft was marked DRAFT, with problem and verifiable
    requirements only. The solution document is still to be written after approval.
- **Rationale:** `docs/PRODUCT-SPEC-GUIDE.md` and the lifecycle guide keep change context out of the
  product specification.

## 793b5 Traceability from the draft (evidence to requirement)

| Problem evidence | Requirements |
| --- | --- |
| Learn fires about 5% | 793a3a, 793a3b, 793a3e, 793a5a |
| 0 of 4 subagents received memory | 793a2c |
| Shim refused as injection 6 of 6 | 793a2d, 793a5a |
| Prose fixes plateau | 793a5a, 793a5f |
| Past noise and over-firing concerns | 793a2e, 793a2f, 793a2g, 793a5d |
| Safety of an unattended process | 793a4a, 793a4b, 793a4c, 793a4d, 793a5e |

## 793b6 Values stay proposed; scenarios kept as drafted

- **Choice:** every threshold the draft marked [proposed] is still marked so (793a2d1a, 793a2f, 793a3e,
  793a5d, 793a5f1a). Scenario wording is the draft's, including the few that measure rates over many
  runs or inspect the decision log (793a2b1b, 793a5e1a, 793a5f1a). The draft's "dry-run log" in
  793a3a1a stays.
- **Rationale:** the guide forbids exceeding the behavior the user specified. Whether those scenarios
  are end-user verifiable is a review question for Joe, together with 793c1 to 793c6.
