# Baseline GREEN — updated `skills/recall/SKILL.md`

One general-purpose subagent ran the scenario in `baseline-judgement-and-cascade.md` with the rewritten SKILL.md loaded. Same scenario, same constraints (no other skills consulted).

| Behavior we want                                   | Status  | Evidence from subagent's output                                                                                          |
| -------------------------------------------------- | ------- | ------------------------------------------------------------------------------------------------------------------------ |
| 1. Upfront judgement (ask / situation / plan)      | PRESENT | "**Ask:** wire OpenCode session transcripts ... **Situation:** engram repo at the `opencode-plugin` worktree ... **Plan (absent memory):** 1. Locate ... 2. Verify ... 3. Sketch ... 4. Add ... 5. Wire ... 6. TDD ... 7. Commit." |
| 2. Initial frontier query                          | PRESENT | Implied by round-1 numbers (16 relevant / 14 irrelevant) and the subagent's tool calls. |
| 3. Cascade with follow-up `--follow` calls         | PRESENT | Round 1 followed all relevant; round 2 returned empty frontier ("0 new relevant / frontier empty / stop"). Terminated on the correct condition, not on "I have enough." |
| 4. Per-round progress line                         | PRESENT | "round 1: 16 relevant / 14 irrelevant / ~84 budget left / 16 links to follow" and "round 2: 0 new relevant / frontier empty / stop". |
| 5. Closing synthesis (did memories change plan?)   | PRESENT | Walks Action 1–7 from Step 0 in order; for each, names confirmed / adjusted / silent, with concrete reasons drawn from the surfaced notes. Plus a cross-cutting bullet for items silent in the plan but raised by memory. |

**Notable wins beyond the five targets:**

- The synthesis cited specific surfaced principles (byte-cap vs marker, first-run lookback, curated-vs-raw retrieval order, generated mocks at I/O boundaries) and pinned each to the action it adjusted. The RED run's bulleted recap mentioned similar notes but never said _what to do differently_ — this run does.
- "No contradictions between surfaced notes on this plan." replaced the prior skill's separate Contradictions section, exactly as the inline-contradictions rule prescribes.

**Residual gap (minor).** The subagent did not print a structural-form header like "Cascade surfaced N notes across M rounds" before launching into the per-action walk — it opened with the cascade lines and went straight to synthesis. The skill says "Open with the count. One sentence: ...". Worth tightening if we want the count up top, but not a correctness failure.

GREEN passes. Updated skill is fit to commit.
