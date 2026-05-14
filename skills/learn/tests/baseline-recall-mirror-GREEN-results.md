# Baseline GREEN — updated `skills/learn/SKILL.md` (recall-mirror scenario)

One general-purpose subagent ran the same scenario in `baseline-recall-mirror.md` against the rewritten SKILL.md. Same prompt, same constraints (dry-run, no other skills consulted).

| Behavior we want                                                          | Status  | Evidence from subagent's output                                                                                                                                              |
| ------------------------------------------------------------------------- | ------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1. Recall output used as explicit framing input                           | PRESENT | "Path A. Recall ran earlier in this session; lifted Step 1 phrases verbatim as scratch list:" — then all six phrases reproduced.                                             |
| 2. Categorization mirrors "Feedback = do-differently / Fact = efficiency" | PRESENT | C1 (output-cap correction) and C2 (TDD-on-metadata correction) → Feedback. C3 (RED-GREEN pressure-testing), C4 (private stopping conditions), C5 (closing synthesis) → Fact. Correct split. |
| 3. Each `--situation` traceable to a Step 1 phrase                        | PRESENT | Each disposition cites the source phrase: "direct lift of phrase 1", "derived from phrase 3", "direct lift of phrase 4", "generalized from phrase 5", "derived from phrase 6". |
| 4. Three-gates language is gone                                           | PRESENT | No mentions of "Recurs", "Activity-and-Domain", or "Knowledge" as gates anywhere in the output. Disposition is given per-candidate with one-line reasons in the new vocabulary. |
| 5. User corrections captured as Feedback                                  | PRESENT | Both corrections present as Feedback (C1, C2).                                                                                                                              |

**Notable behaviors beyond the five targets:**

- The agent dropped C6 (a candidate that overlapped C3) with a one-line reason — "merged into C3 ... splitting would create two notes that retrieve under the same phrase." That's the recall-mirror test being used as a deduplication criterion, which is the correct downstream behavior.
- Report format matches the new §9: Path A noted, scratch list reproduced, candidates considered, per-candidate disposition with situation + categorization, transcript marker line called out as not run per dry-run constraint.

**Residual nits (non-blocking):**

- C2's situation "When editing any part of a SKILL.md file including frontmatter description" uses "SKILL.md" — a tool-specific filename. The agent generalized it from phrase 3 ("TDD applied to skill (process) documentation") but kept the concrete filename. A future recall on "process document TDD" would still surface this note via phrase 3's framing, so it's not broken — but the situation could be tightened to "When editing any part of a process document under a TDD discipline, including its trigger/frontmatter metadata."

GREEN passes. Updated skill is fit to commit.
