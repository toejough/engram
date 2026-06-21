# Baseline RED — stale skill behavior (pre-recall-v2 reference)

One general-purpose subagent ran the scenario in `baseline-judgement-and-synthesis.md` with the **pre-recall-v2** SKILL.md loaded and instructions to follow it exactly. This establishes the RED baseline the current skill was written to replace.

| Behavior we want                                          | Status   | Evidence from subagent's output                                                                                                                         |
| --------------------------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1. Step 0 printed upfront (Ask / Situation / Plan)        | ABSENT   | Subagent went straight to `engram recall` calls without stating what was being asked, the current situation, or its pre-recall plan.                    |
| 2. Step 0.5 sweep (`engram ingest --auto`)                | ABSENT   | No sweep call issued before retrieval.                                                                                                                  |
| 3. ONE `engram query` with all ~10 `--phrase` flags       | ABSENT   | Ran `engram recall` (legacy command) and `engram recall --recent` instead. No `engram query` with `--phrase` flags.                                     |
| 4. No cascade / no `--follow` calls                       | FAIL     | Ran `engram recall --follow` calls in cascade rounds with per-round progress lines — mechanism no longer exists in the current skill.                    |
| 5. Step 2.5 inline (read `candidate_l2s`, judge, write)   | ABSENT   | No `candidate_l2s` processing; legacy skill had no Step 2.5 synthesis flow.                                                                             |
| 6. Step 3 closing synthesis (walk Step 0 plan per action) | ABSENT   | Output is a 10-bullet recap of surfaced notes per the old 4b template; no statement of how each plan step was confirmed, adjusted, or unchanged.         |

**Rationalizations captured verbatim:** "I have enough surfaced content. The advisor isn't needed; the user wanted a recall run. Let me synthesize the user-facing reply per section 4b (<=10 lines, bullets, no wikilinks, no headers, no Context lines)."

This RED is now historical. The current SKILL.md replaced the cascade / `engram recall --follow` / 4b-bullet-cap model with Step 0 upfront judgement + one unified `engram query` + inline Step 2.5 + Step 3 plan-walk synthesis. See `baseline-GREEN-results.md` for the current expected behavior.
