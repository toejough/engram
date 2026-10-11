# Parallel Memory Agent - Open Questions

These are the Open Decisions carried over from the draft requirements. Joe answered all six on
2026-10-11, and the answers are applied in [`793a_product-spec.md`](./793a_product-spec.md). The
phrasing choices made in applying them are in [`793b_decision-log.md`](./793b_decision-log.md).

## 793c1 What are the capture precision bar and capture-rate target?

- **Status:** answered (Joe, 2026-10-11).
- **Answer:** the capture rate MUST exceed the current baseline (learn fires on about 5% of
  lesson-worthy moments: 4/71 = 5.6% and 3/66 = 4.5%). No fixed precision threshold; the System MUST
  record its capture precision, the share of captured notes judged valid lessons on a sample.
- **Applied in:** [793a3e Capture rate and recorded precision](./793a_product-spec.md#793a3e-capture-rate-and-recorded-precision).

## 793c2 Are the volume bounds and latency budget accepted?

- **Status:** answered (Joe, 2026-10-11).
- **Answer:** start with today's values. The bounds MUST NOT be lower than what the main agent
  receives today from the instruction-based recall query (measured 2026-10-11 on Joe's vault: about
  135 KB, 45 items, 10.6-11.0 s per query; `engram query` defaults `--limit 20`, content budget 15).
  The System MUST record, per moment, the injected volume and the added latency; reducing the bounds
  needs that data (follow-up #845).
- **Applied in:** [793a2f Bounded volume](./793a_product-spec.md#793a2f-bounded-volume), [793a5d Latency](./793a_product-spec.md#793a5d-latency) and [793a5e Auditability and measurability](./793a_product-spec.md#793a5e-auditability-and-measurability).

## 793c3 What is the default mode for a fresh install?

- **Status:** answered (Joe, 2026-10-11).
- **Answer:** active, with measurement always on, so data accumulates for later optimization. Observe
  and off remain operator choices.
- **Applied in:** [793a4a Operating modes](./793a_product-spec.md#793a4a-operating-modes).

## 793c4 Are captured lessons written directly, or held for `curate` review?

- **Status:** answered (Joe, 2026-10-11).
- **Answer:** written directly to the vault. The System, the parallel agent, owns judging what is
  worth saving, so the main agent's context is not polluted with memory evaluation. Precision
  recording ([793c1 What are the capture precision bar and capture-rate target?](#793c1-what-are-the-capture-precision-bar-and-capture-rate-target)) is the quality guard.
- **Applied in:** [793a3a Evaluate every completed turn](./793a_product-spec.md#793a3a-evaluate-every-completed-turn) (scenarios [793a3a1a User correction is captured without agent action](./793a_product-spec.md#793a3a1-uat-scenarios) and [793a3a1b Explicit save-request](./793a_product-spec.md#793a3a1-uat-scenarios)) and [793a3d Provenance](./793a_product-spec.md#793a3d-provenance) (scenario [793a3d1a Provenance present](./793a_product-spec.md#793a3d1-uat-scenarios)).

## 793c5 Does 793a2g keep human-originated prompts as the default?

- **Status:** answered (Joe, 2026-10-11).
- **Answer:** no. Recall applies to every prompt regardless of source. Memory is always worth
  checking; a relevant hit not already in the working agent's context is worth injecting.
- **Applied in:** [793a2g Recall for every prompt regardless of source](./793a_product-spec.md#793a2g-recall-for-every-prompt-regardless-of-source), retitled from "Human-originated moments only".

## 793c6 Does capture also run inside subagent sessions?

- **Status:** answered (Joe, 2026-10-11).
- **Answer:** yes, in both top-level and subagent sessions.
- **Applied in:** [793a3a Evaluate every completed turn](./793a_product-spec.md#793a3a-evaluate-every-completed-turn), scenario [793a3a1c Lesson in a subagent session is captured](./793a_product-spec.md#793a3a1-uat-scenarios).
