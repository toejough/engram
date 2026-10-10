# 792g16 lessons-contract - Adoption Audit

Audit of [792a16 lessons-contract](../792a_product-spec.md#792a16-added-lessons-contract). No [792b](../792b_technical-spec.md) requirement cites its scenarios. Method: read the LESSONS text in the shipped route, please and learn skills and the learn guidance, and compared it with the git history of those files (`git log -S`, `git blame`) and with the archived route, please and learn OpenSpec specs recovered by `git show <commit>:<path>`. Paths in the Evidence column are relative to `agent-instructions/skills/` for `SKILL.md` files (`write-memory`, `learn`, `route`, `please`, `recall`), to `agent-instructions/guidance/` for `learn.md` and `shim.md`, and to the repository root otherwise.

| Spec ID | Verdict | Evidence |
|---|---|---|
| [792a16a A completion report ends with a LESSONS line](../792a_product-spec.md#792a16a-a-completion-report-ends-with-a-lessons-line) | match | route/SKILL.md:70-73 `MUST end with a LESSONS: line - LESSONS: none, or 1-3 comma-separated one-line lessons (confirmed corrections, surprising findings, validated approaches ...) ... not vault notes the subagent decides on itself` |
| [792a16a1a A subagent with lessons reports them](../792a_product-spec.md#792a16a1-uat-scenarios) | match | same evidence as the requirement |
| [792a16a1b A subagent with nothing to report says so](../792a_product-spec.md#792a16a1-uat-scenarios) | match | same evidence as the requirement |
| [792a16a1c At most three lessons](../792a_product-spec.md#792a16a1-uat-scenarios) | match | same evidence as the requirement |
| [792a16b Every dispatch is told to report lessons](../792a_product-spec.md#792a16b-every-dispatch-is-told-to-report-lessons) | match | route/SKILL.md:60-76 `Every dispatch MUST hand the subagent` the five items with the completion-report contract last. The skill has no fork-specific sentence; `Every dispatch` covers a fork, and the archived route spec (git show 6e7cf8c0^:openspec/specs/route/spec.md) reasoned the same way (`that checklist already governs every dispatch`). Treated as a match |
| [792a16b1a A fresh-context dispatch carries the mandate](../792a_product-spec.md#792a16b1-uat-scenarios) | match | same evidence as the requirement |
| [792a16b1b A fork's handoff carries it too](../792a_product-spec.md#792a16b1-uat-scenarios) | match | same evidence as the requirement |
| [792a16b1c It follows the task description](../792a_product-spec.md#792a16b1-uat-scenarios) | match | same evidence as the requirement |
| [792a16c A missing line is asked for once](../792a_product-spec.md#792a16c-a-missing-line-is-asked-for-once) | match | route/SKILL.md:74-76 `send ONE targeted follow-up asking for it - never a second re-ask; if it's still missing after that one follow-up, record LESSONS: none and move on rather than blocking the pipeline`. A line supplied in answer is recorded like any other under please's step-4 text |
| [792a16c1a A missing line is re-asked](../792a_product-spec.md#792a16c1-uat-scenarios) | match | same evidence as the requirement |
| [792a16c1b The answer counts like an on-time line](../792a_product-spec.md#792a16c1-uat-scenarios) | match | same evidence as the requirement |
| [792a16c1c There is no second re-ask](../792a_product-spec.md#792a16c1-uat-scenarios) | match | same evidence as the requirement |
| [792a16d Every line is recorded as it returns](../792a_product-spec.md#792a16d-every-line-is-recorded-as-it-returns) | code bug #788 | please/SKILL.md:95-101 records the line verbatim, `none` included, when a unit's result comes back and before the next unit; marks were dropped in commit dcfe514c (2026-10-01), so the list is unmarked and verbatim. But the text is scoped to step 4 (`:122` `collected across step 4's dispatches`, `:157` red flag), while the requirement says `wherever in the workflow the dispatch lands`, which includes the gate reviewers dispatched in other steps; their LESSONS lines are not collected. The archived please spec (a9d39cb2, 2026-09-30) stated `wherever` and the skill never carried it (git log -S shows step 4 wording since de3c244e, 2026-09-07). Already filed as #788 item 6; the spec keeps the intent, and a skill fix needs the writing-skills cycle |
| [792a16d1a Lines are kept in order and verbatim](../792a_product-spec.md#792a16d1-uat-scenarios) | match | please/SKILL.md:95-98 verbatim running list, in dispatch order |
| [792a16d1b A none is still recorded](../792a_product-spec.md#792a16d1-uat-scenarios) | match | please/SKILL.md:96-98 `including LESSONS: none entries`; learn/SKILL.md:94-96 skips them trivially |
| [792a16d1c A gate does not filter the list](../792a_product-spec.md#792a16d1-uat-scenarios) | match | for a step-4 unit the line is recorded the moment its result returns (please/SKILL.md:96), before any gate verdict, so a later gate rejection does not remove it |
| [792a16e Each lesson is judged when it returns](../792a_product-spec.md#792a16e-each-lesson-is-judged-when-it-returns) | match | learn/SKILL.md:21-28 and :88-104 (judge each non-`none` line at the return, fast path, one note per lesson, no sweep, confirmed and general, discard silently), guidance/learn.md:30-37 (`before your next dispatch (or before you end your turn, if no dispatch is left)`; skip `none`, `done, tests pass`, `might be a race`), please/SKILL.md:99-100. The please skill omits `or before you end your turn` (#788 item 5); the always-loaded guidance carries it, so the requirement holds |
| [792a16e1a A confirmed correction is captured at the return](../792a_product-spec.md#792a16e1-uat-scenarios) | match | same evidence as the requirement |
| [792a16e1b None and bare success are left alone](../792a_product-spec.md#792a16e1-uat-scenarios) | match | same evidence as the requirement |
| [792a16e1c An unconfirmed lesson is left alone](../792a_product-spec.md#792a16e1-uat-scenarios) | match | same evidence as the requirement |
| [792a16e1d The last return is captured before the turn ends](../792a_product-spec.md#792a16e1-uat-scenarios) | match | same evidence as the requirement |
| [792a16f The closing learn gets the whole list](../792a_product-spec.md#792a16f-the-closing-learn-gets-the-whole-list) | match | please/SKILL.md:121-126 hands the full list (`none` included) with the audit map and drops no line; learn/SKILL.md:94-104 skips a line a vault note already covers and discards the rest silently; please/SKILL.md:129 forbids marking step 7 not applicable. An empty collection is the no-dispatch case of the same text. The dependency of the dedupe on the opening `engram query` is noted in #788 |
| [792a16f1a The closing learn receives the list](../792a_product-spec.md#792a16f1-uat-scenarios) | match | same evidence as the requirement |
| [792a16f1b No dispatch, empty collection](../792a_product-spec.md#792a16f1-uat-scenarios) | match | same evidence as the requirement |
| [792a16f1c A line captured at its return is skipped](../792a_product-spec.md#792a16f1-uat-scenarios) | match | same evidence as the requirement |
| [792a16f1d A missed line is caught](../792a_product-spec.md#792a16f1-uat-scenarios) | match | same evidence as the requirement |
| [792a16f1e A mixed batch](../792a_product-spec.md#792a16f1-uat-scenarios) | match | same evidence as the requirement |
| [792a16g A confirmed approach is looked for at the moment of confirmation](../792a_product-spec.md#792a16g-a-confirmed-approach-is-looked-for-at-the-moment-of-confirmation) | match | learn/SKILL.md:150-186 (completion-moment anchor includes reading a subagent's report and its LESSONS line; three real exemplars; bar unchanged: `a resolved uncertainty or an explicit specific confirmation, never it worked`); dev/eval/LEDGER.md rows `learn-rate-remeasure-w2` and `learn-at-return-guidance` exist and carry the figures the capability's rationale cites (3/66 = 4.5% against 4/71 = 5.6%) |
| [792a16g1a A completion anchors the scan](../792a_product-spec.md#792a16g1-uat-scenarios) | match | same evidence as the requirement |
| [792a16g1b An exemplar-shaped line is recognized](../792a_product-spec.md#792a16g1-uat-scenarios) | match | same evidence as the requirement |
| [792a16g1c A bare success is still not kind 4](../792a_product-spec.md#792a16g1-uat-scenarios) | match | same evidence as the requirement |

## Summary

Rows by verdict: code bug #788 1, match 30. Total rows 31 (31 from 792a, 0 from 792b).

Every requirement and scenario of the lessons-contract capability was compared with the LESSONS text in `route/SKILL.md`, `please/SKILL.md`, `learn/SKILL.md` and `guidance/learn.md`, and with the git history of those files and of the archived route and please specs. The contract has no binary command; its evidence is the skill text, the ledger rows and the eval harness under dev/eval/learn-at-return.

- One code (skill-text) problem, already filed as #788 item 6: [792a16d Every line is recorded as it returns](../792a_product-spec.md#792a16d-every-line-is-recorded-as-it-returns) says every dispatched unit's line is recorded wherever the dispatch lands, but please's text collects only step 4's dispatches, so gate-reviewer lines are lost. The spec states the intent and is left unchanged. Fixing it needs the writing-skills cycle on `please/SKILL.md`.
- Related wording nits in #788 (items 4 and 5) do not make any requirement false because the ambient guidance covers them.
- No doc fix was needed.
- Unclear: none.
