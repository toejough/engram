# Judged Automatic Memory - Open Questions

A default stands until the owner answers; his answer is recorded in the entry with its date. A question that blocks nothing is an author choice and is not asked: it is a decision in [855b Judged Automatic Memory - Decision Log](./855b_decision-log.md#judged-automatic-memory---decision-log) (`docs/AGENTS.md`).

## 855c1 How long may a judgement or delivery run before it is given up?

- **Context:** the owner dropped the hold on a [prompt](../793_automatic-memory/793a_product-spec.md#793a1a-terms) ([855b4 No time bound before a prompt reaches the agent](./855b_decision-log.md#855b4-no-time-bound-before-a-prompt-reaches-the-agent)). A judgement or delivery that never answers would then hold the prompt forever. The host's type definitions give a hook a budget of 10 seconds that does not count a model call or a command, and a model call has no time limit unless one is given ([855b14 A slow step is waited for, a stalled one is not](./855b_decision-log.md#855b14-a-slow-step-is-waited-for-a-stalled-one-is-not)).
- **Options:** (a) Hold until the user interrupts. (b) A liveness bound of 2 minutes, far above any normal judgement and not a latency target, after which the delivery steps aside, delivers nothing and shows a failure line. (c) The host's own ceiling, if it had one.
- **Recommendation:** (b): the host has no ceiling to use, and (a) leaves the user unable to tell a stalled step from a slow one.
- **Blocks:** [855a3c1a A judgement that never answers is stepped aside after 2 minutes](./855a_product-spec.md#855a3c1-uat-scenarios) and the stall clause of [855a3c 793a2d Never in the way](./855a_product-spec.md#855a3c-modified-793a2d-never-in-the-way).
- **Status:** answered (Owner, 2026-10-18): option (b); recorded in [855b14 A slow step is waited for, a stalled one is not](./855b_decision-log.md#855b14-a-slow-step-is-waited-for-a-stalled-one-is-not).
