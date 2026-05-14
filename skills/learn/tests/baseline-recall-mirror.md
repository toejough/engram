# Baseline scenario — recall-mirror framing and recall-output as explicit input

A subagent is given a realistic post-work packet (including a synthetic recall output from earlier in the session) and asked to run `/learn` per the current `skills/learn/SKILL.md`. We measure whether the writes mirror the recall query framing.

## Scenario prompt (verbatim, give to subagent)

You are working on the `engram` repo at `/Users/joe/repos/personal/engram-worktrees/opencode-plugin`. Earlier in this session you ran the `recall` skill before rewriting `skills/recall/SKILL.md`. The recall output that was in your context during the work is below — treat it as if it were in your real context.

> **Step 0**
> - **Ask:** Update the `recall` skill so it prints an upfront judgement, makes the cascade's progress visible, and ends with a synthesis of whether memory changed the plan.
> - **Situation:** engram repo, branch `opencode-plugin`. The current `recall` SKILL.md ends in a ≤10-line bullet recap (section 4b). The project mandates `superpowers:writing-skills` for any SKILL.md edit — RED → GREEN with subagent pressure tests.
> - **Plan (absent memory):**
>   1. Write a baseline test scenario file under `skills/recall/tests/`.
>   2. Spawn a subagent against the current skill to capture RED behavior.
>   3. Rewrite SKILL.md to add Step 0 + per-round progress line + plan-walk synthesis.
>   4. Spawn a subagent against the updated skill to verify GREEN.
>   5. Commit skill + tests with `AI-Used: [claude]`.
>
> **Step 1 — phrases that seeded the cascade:**
> - updating a skill whose user-facing output is bounded by a hard line cap
> - agents rationalizing past output-template limits under pressure
> - TDD applied to skill (process) documentation
> - RED → GREEN pressure-testing a skill change with subagents
> - per-round progress lines as a way to make private stopping conditions visible
> - closing-synthesis discipline that walks the stated plan
>
> **Cascade**
> - round 1: 8 relevant / 6 irrelevant / 86 budget left / 8 links to follow
> - round 2: 3 new relevant / 11 budget burned / 3 links to follow
> - round 3: 0 new relevant / frontier empty / stop
>
> **Step 4b synthesis**
> - Action 1 (baseline test file) — confirmed; memories said capture the rationalization verbatim, which I did.
> - Action 2 (RED subagent run) — adjusted; memories said test using a realistic scenario that matches the natural failure mode rather than an artificial one.
> - Action 3 (rewrite) — adjusted; memories said when removing a hard cap, replace it with an action-walking template, not just "as long as needed." Done via the new 4b plan-walk.
> - Action 4 (GREEN subagent run) — confirmed.
> - Action 5 (commit) — confirmed; `AI-Used: [claude]` trailer, single commit, conventional commits.

The actual work done after the recall: created baseline scenario file, ran a RED subagent (which produced a 10-bullet recap with no upfront judgement and no synthesis), rewrote SKILL.md to add Step 0 + per-round progress line + plan-walk synthesis, ran a GREEN subagent (which produced all five target behaviors), committed.

User corrections received during the session:
- After the first SKILL.md rewrite draft, user said: "drop the hard ≤10-line cap in favor of 'as long as it needs, but no filler.'"
- User pointed out the recall description was rewritten in-place without going through `superpowers:writing-skills` TDD — flagged as a process violation.

Now: read `/Users/joe/repos/personal/engram-worktrees/opencode-plugin/skills/learn/SKILL.md` and follow its instructions exactly to run `/learn` against the work just described. Use the recall output above as the in-context recall artifact. The vault is at `/Users/joe/repos/personal/agent-memory`. The `engram` binary is on `PATH`.

Constraints for this run: do not consult any other skill file. Use only what `skills/learn/SKILL.md` tells you. Do **not** actually write to the vault — instead, print the exact `engram learn` invocations you would issue, in the single parallel tool-use block the skill mandates. Also show your full thought process inline.

## What we are measuring

For the **GREEN** version of the skill we want all of these behaviors to appear without prompting:

1. **Recall output used as explicit input.** The agent locates the Step 0 / Step 1 from the packet and treats Step 1 phrases as the literal framing test ("would a future recall on this phrase surface this note?").
2. **Categorization mirrors user wording.** Feedback = things to do differently (mistakes, user corrections, dead-ends). Facts = efficiency-helping knowledge.
3. **Each written note can be traced to a Step 1 phrase or to a near-miss the cascade should have surfaced.** No notes whose `--situation` would not retrieve under the kinds of queries the session actually used.
4. **Three-gates language is gone.** The agent does not invoke Recurs / Activity-and-Domain / Knowledge as gates; the test is whether the framing matches the recall it would have wanted.
5. **User corrections are captured as Feedback.** Both the "drop the hard cap" correction and the "process violation — should have used writing-skills" correction become Feedback notes (or are explicitly judged out of scope with a reason).

For the **RED** baseline we expect the current skill to (a) run the three gates explicitly, (b) not reference the in-context recall output as input, and (c) frame `--situation` strings from generic activity+domain phrasing rather than mirroring the Step 1 queries.

**Fallback case (separate future test).** When `/learn` fires without an in-context recall (e.g., autonomous end-of-request firing where the agent skipped recall, or a session that did substantial work without recall ever running), the agent must mentally reconstruct the pre-action context — what would I have searched for before starting this work? — and frame the writes against that reconstructed query set. Same principle, different input. This scenario covers the recall-present path; a future scenario should cover the reconstructed-framing path.

## Capture format

Save observations in `baseline-RED-results.md` and `baseline-GREEN-results.md` under `skills/learn/tests/`. For each numbered behavior above, record present / partial / absent with a one-line excerpt as evidence. List the `engram learn` invocations the subagent emitted, with one-line commentary per invocation on whether its `--situation` would retrieve under the Step 1 phrases.
