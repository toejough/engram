# Baseline RED — current `skills/learn/SKILL.md` (recall-mirror scenario)

One general-purpose subagent ran the scenario in `baseline-recall-mirror.md` with the current SKILL.md loaded, recall output supplied in the prompt, and instructions to print invocations rather than write.

| Behavior we want                                                          | Status  | Evidence from subagent's output                                                                                                                |
| ------------------------------------------------------------------------- | ------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| 1. Recall output used as explicit framing input                           | ABSENT  | Agent processed the in-context recall as background and never referenced the Step 1 phrases as the literal queries the writes must match.    |
| 2. Categorization mirrors "Feedback = do-differently / Fact = efficiency" | PARTIAL | All four candidates written as Feedback. Two of them (visible-stopping-conditions, realistic-RED-baseline) are methodological principles with no mistake or correction — under the new framing these are Facts. |
| 3. Each `--situation` traceable to a Step 1 phrase                        | PARTIAL | Situations are reasonable activity+domain framings ("When designing a multi-round cascading process with internal stopping conditions") but not deliberately mirrored to Step 1 ("per-round progress lines as a way to make private stopping conditions visible"). A future recall on the exact Step 1 phrasing might miss them. |
| 4. Three-gates language is gone                                           | ABSENT  | Heavy explicit invocation: "Recurs: PASS", "Activity-and-Domain: PASS", "Knowledge: PASS", plus an explicit "FAIL Recurs ... Reframe once ... PASS on re-gate" sequence for C2.                                                  |
| 5. User corrections captured as Feedback                                  | PRESENT | Both corrections (hard-cap removal, TDD-on-metadata) became Feedback notes. Correct on the new framing too.                                    |

**Rationalizations / structural artifacts of the current skill:**

- Gate-driven reasoning made the agent reach for activity-and-domain phrasings independently rather than read its own pre-action query frame from the recall packet.
- "Single parallel tool-use block" is correctly internalized.
- Transcript-fetch sequencing is correctly preserved.

**Invocations emitted (summary):**

1. `engram learn feedback --slug synthesis-output-caps` — situation "When designing the output section of a process template that ends in synthesis". Step 1 had "updating a skill whose user-facing output is bounded by a hard line cap" — partial overlap; would surface on some queries, miss on others.
2. `engram learn feedback --slug process-doc-tdd-scope` — situation "When editing a process document governed by a TDD discipline". Step 1 did not surface this concept (it was a user correction post-hoc); reasonable phrasing but not mirror-derived.
3. `engram learn feedback --slug visible-stopping-conditions` — situation "When designing a multi-round cascading process with internal stopping conditions". Step 1 had "per-round progress lines as a way to make private stopping conditions visible" — semantically same idea, different phrasing. Likely retrievable, but not deliberately matched.
4. `engram learn feedback --slug red-baseline-realism` — situation "When writing a RED baseline test for a process or skill change". Step 1 had "RED → GREEN pressure-testing a skill change with subagents" — close.

**Verdict.** The current skill produces defensible writes, but framing is gate-driven rather than recall-mirror-driven, and the Feedback/Fact split is collapsed because gate machinery doesn't distinguish them. The rewrite must: drop the gates, install recall-mirror as the primary test, sharpen Feedback/Fact to the user's wording, and explicitly instruct the agent to lift framings from the in-context recall (or to reconstruct one when absent).
