# Parallel Memory Agent - Open Questions

These are the Open Decisions carried over from the draft requirements. Each needs Joe's input before
the product spec in [`793a_product-spec.md`](./793a_product-spec.md) is final. The values marked
**[proposed]** there are the draft's suggested starting values. Choices that need only validation are
in [`793b_decision-log.md`](./793b_decision-log.md).

## 793c1 What are the capture precision bar and capture-rate target?

- **Context:** 793a3e requires that automatically captured notes be judged valid lessons at a
  minimum rate, and that the System capture materially more lesson-worthy moments than the current
  baseline of about 5%.
- **Options:**
  1. Precision at least 70% and capture rate at least 50%.
  2. Other thresholds, named by Joe.
- **Recommendation:** the draft proposed option 1 and gave no further recommendation.
- **Blocks:** 793a3e1a and 793a3e1b. Default: option 1.

## 793c2 Are the volume bounds and latency budget accepted?

- **Context:** 793a2f bounds injections at 3 notes / 6 KB at a prompt, 2 notes / 3 KB at a tool
  result, and 4 tool-result injections / 12 KB per turn. 793a5d allows 1.5 s median added latency per
  human prompt.
- **Options:**
  1. Accept the proposed numbers.
  2. Change one or more of them.
- **Recommendation:** the draft proposed option 1 and gave no further recommendation.
- **Blocks:** 793a2f1a–793a2f1c and 793a5d1a. Default: option 1.

## 793c3 What is the default mode for a fresh install?

- **Context:** 793a4a lets the operator choose off, observe or active, and says the default SHOULD be
  observe.
- **Options:**
  1. Observe: decisions are logged and nothing is injected or written.
  2. Active: injection and writes occur from the first session.
- **Recommendation:** 1 (the draft's proposal).
- **Blocks:** the default in 793a4a. Default: option 1.

## 793c4 Are captured lessons written directly, or held for `curate` review?

- **Context:** 793a3 describes capture as writing notes. The draft left open whether those writes go
  straight into the vault or land as pending offers that the `curate` skill reviews first.
- **Options:**
  1. Write notes directly to the vault.
  2. Land them as pending offers for `curate` review.
- **Recommendation:** none in the draft.
- **Blocks:** 793a3a1a, 793a3a1b and 793a3d1a, which assert where a captured note appears. Default:
  none. The wording of those scenarios waits.

## 793c5 Does 793a2g keep human-originated prompts as the default?

- **Context:** 793a2g says the System SHOULD NOT inject recall for prompts that did not originate
  from the human, unless the operator configures otherwise.
- **Options:**
  1. Keep human-originated-only as the default.
  2. Also inject for automated prompts by default.
- **Recommendation:** 1 (the draft's current requirement).
- **Blocks:** 793a2g. Default: option 1.

## 793c6 Does capture also run inside subagent sessions?

- **Context:** 793a3a evaluates "each completed turn". It is undecided whether that includes turns in
  subagent sessions or only the top-level session. 793a2c (recall at subagent dispatch) is unaffected.
- **Options:**
  1. Top-level session only.
  2. Top-level and subagent sessions.
- **Recommendation:** none in the draft.
- **Blocks:** the scope of 793a3a. Default: none. The scope waits.
