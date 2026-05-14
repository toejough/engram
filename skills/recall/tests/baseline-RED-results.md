# Baseline RED — current `skills/recall/SKILL.md`

One general-purpose subagent ran the scenario in `baseline-judgement-and-cascade.md` with the current SKILL.md loaded and instructions to follow it exactly.

| Behavior we want                                   | Status   | Evidence from subagent's output                                                                                                       |
| -------------------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------------- |
| 1. Upfront judgement (ask / situation / plan)      | ABSENT   | Subagent went straight to `engram recall` calls without stating what was being asked, the current situation, or its pre-recall plan. |
| 2. Initial frontier query                          | PRESENT  | Ran `engram recall` (anchors) and `engram recall --recent`.                                                                           |
| 3. Cascade with follow-up `--follow` calls         | PARTIAL  | Did follow some second-level links; unclear whether it ran to budget or just stopped when it "had enough" — wording: "I have enough surfaced content." |
| 4. Per-round progress line                         | ABSENT   | No `N relevant / M irrelevant / P budget left / Q to follow` line at any point.                                                       |
| 5. Closing synthesis (did memories change plan?)   | ABSENT   | Output is a 10-bullet recap of surfaced notes per the existing 4b template; no statement of how the plan was confirmed, adjusted, or unchanged. |

**Rationalizations captured verbatim:** "I have enough surfaced content. The advisor isn't needed; the user wanted a recall run. Let me synthesize the user-facing reply per section 4b (≤10 lines, bullets, no wikilinks, no headers, no Context lines)."

The ≤10-line cap and the 4b bullet template actively encourage the agent to stop short of synthesis-with-judgement. The skill needs to make upfront judgement and closing plan-impact synthesis non-skippable, and the cascade needs a visible per-round progress line so "I have enough" stops being a private decision.
