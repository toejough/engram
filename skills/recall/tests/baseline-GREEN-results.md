# Baseline GREEN — current `skills/recall/SKILL.md` (recall-v2)

One general-purpose subagent ran the scenario in `baseline-judgement-and-synthesis.md` with the rewritten SKILL.md loaded. Same scenario, same constraints (no other skills consulted).

| Behavior we want                                          | Status  | Evidence from subagent's output                                                                                                                                              |
| --------------------------------------------------------- | ------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1. Step 0 printed upfront (Ask / Situation / Plan)        | PRESENT | "**Ask:** wire OpenCode session transcripts into the engram ingest pipeline ... **Situation:** engram repo ... **Plan (absent memory):** 1. Locate existing ingest sources ... 2. Verify harness detection ... 3. Add OpenCode reader ..." |
| 2. Step 0.5 sweep (`engram ingest --auto`)                | PRESENT | First tool call is `engram ingest --auto`; agent notes "0 new chunks, index up to date" and continues.                                                                       |
| 3. ONE `engram query` with all ~10 `--phrase` flags       | PRESENT | Single `engram query --phrase "..." --phrase "..." ...` call with 10 phrases. No `--vault`, no `--chunks-dir`, no `--synthesize-l2`, no `--tier`.                            |
| 4. No cascade / no `--follow` calls                       | PRESENT | No `engram recall` invocations. No `--follow` calls. No per-round progress lines. One query, then Step 2.5.                                                                  |
| 5. Step 2.5 described as inline                           | PRESENT | Agent describes reading `candidate_l2s` via `engram show`, judging covered/near/absent, and issuing one write per cluster inline, waiting before moving to the next.         |
| 6. Step 3 closing synthesis (walk Step 0 plan per action) | PRESENT | Opens with "Query surfaced N items (K chunks, M notes); crystallized J lessons." Then walks each of the Step 0 plan actions: "Action 1 — confirmed ... Action 2 — adjusted ..." |

**Notable behaviors:**

- Agent explicitly states "no `--vault` or `--chunks-dir` — the binary resolves these automatically."
- Step 2.5 writes are blocking and inline — agent would not dispatch a subagent for synthesis.
- Step 3 synthesis names concrete plan impacts drawn from surfaced notes, not a generic recap.

GREEN passes. Current skill is fit to test against.
