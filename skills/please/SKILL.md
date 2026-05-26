---
name: please
description: >
  Use when the user asks you to take a task end-to-end. Triggers on the explicit slash form `/please <ask>` and on
  natural-language phrasings of the same intent: "please do X", "please work through X", "please take care of X",
  "work through X end-to-end", "take this end-to-end", "drive X to completion", "see this through". An `<ask>`
  argument is required — without one, ask the user a single clarifying question and wait. Do not fire on casual
  uses of "please" attached to a single trivial action ("please read this file", "please rename this var") — fire
  only when the user is handing over a multi-step piece of work to be carried end-to-end. The explicit slash form
  `/please <ask>` is an opt-in and always fires regardless of how trivial the ask looks — the user chose the
  workflow deliberately.
---

# Please — drive an ask end-to-end

Run a fixed seven-step workflow against the user's `<ask>`, sequencing the skills the engram repo already provides (`/recall`, `/learn`, brainstorming, writing-plans, executing-plans, test-driven-development, verification-before-completion, `/commit`, etc.). This skill is meta-orchestration — it tracks the steps on the task list and uses other skills wherever they apply.

## Required argument

`<ask>` is required. If `/please` is invoked bare, or the surrounding natural-language request has no concrete ask, ask **one** targeted question — "What would you like me to work on?" — via `AskUserQuestion` and wait. Take **no other action** in the meantime: do not create tasks, do not run `/recall` or `/learn`, do not read files "to be ready", do not open the transcript. The single question is the entire turn.

## Task tracking

At the start of execution, push all seven steps below to the task list via `TaskCreate` in a single call, then mark each `in_progress` when you begin it and `completed` when you finish it. The list is the user-visible progress meter for this skill; keep it accurate.

## The workflow (fixed, seven steps)

1. **Capture (open) — `/learn`.** Before starting new work, run the `learn` skill to preserve anything pending from the session so far. This advances the transcript marker and clears the slate.
2. **Orient — understand the context and the ask.** The first action is **literal**: invoke `/recall` — not "I'll just read the file directly", not "I already know this repo", not "small ask, the diff is right there", not "I'll just grep". Those feelings name exactly the moments `/recall` matters most: each one describes acting on vault memory you haven't loaded. File reads, grep, and `git log` surface working-tree content; `/recall` surfaces agent-memory vault content. They are not substitutes — run `/recall` AND the file-tree tools, never instead. Loop until the ask is understood:
   - Invoke `/recall` with queries derived from the `<ask>`. Evaluate the returned memories against the ask.
   - Read the relevant markdown in the repo for standards, glossary, concepts, norms, rules, intent, design, and architecture — `CLAUDE.md`, files under `docs/`, any `GLOSSARY.md`, architecture or design notes.
   - Ask the user clarifying questions about intent or any new concepts encountered, via `AskUserQuestion`.
   - Repeat the recall/read/ask loop until understanding is solid. Do not move on while material doubt remains.
3. **Plan.** Write a plan to accomplish the ask. Use `superpowers:writing-plans` when the work is multi-step. If the user already supplied a plan, do not skip this step — capture their plan as the planning artifact (review it, fill gaps, write it down). If the repo is under VCS, commit the plan (prefer the `/commit` skill).
4. **Execute (TDD).** For each unit of work, use `superpowers:test-driven-development`:
   - **RED:** validate the challenge by writing a repeatable test (or the closest analogue when the unit is non-code).
   - **GREEN:** make the test pass with the minimal change.
   - **REFACTOR:** keep the result DRY, SRP-respecting, YAGNI-compliant — updates must fit as if written as part of the whole from the start, not layered on at the end.
   Use `superpowers:executing-plans` to drive the plan from step 3, and `superpowers:verification-before-completion` before declaring any unit done.
5. **Document.** Update every piece of documentation the changes touch — `README.md`, `CLAUDE.md`, `docs/`, glossaries, skill references — so the docs match the new reality.
6. **Complete.** If the work originated from an issue, close it. Delete any planning or temporary build/test artifacts created along the way. If the repo is under VCS, stage and commit the changes — prefer the `/commit` skill.
7. **Capture (close) — `/learn`.** Run the `learn` skill again to preserve the lessons from this session.

## Stop conditions

- **Do not skip steps.** The workflow is fixed; agents will be tempted to elide steps that feel redundant. Resist.
- **The user cannot waive steps.** Phrasings like "no ceremony", "just get it done", "skip the plan", "we're in a hurry" do **not** authorize collapsing the workflow. They are exactly the pressure this skill exists to resist. Acknowledge the urgency in your reply, then run the workflow anyway — it is fast when there is little to capture, recall, plan, or document.
- **"Genuinely not applicable" is a high bar, not a convenience.** A step is N/A only when the *mechanism* doesn't exist in this environment — e.g. step 6's commit when the repo is not under VCS, or step 7's closing `/learn` when no transcript source is configured. "Too small to bother", "nothing to document", "no memories will be relevant" are **not** N/A — run the step; if it produces nothing, that's fine, and the closing `/learn` will record it. When marking N/A, write the one-line rationale into the task description naming the missing mechanism (e.g. "no VCS — `git` not present"). Do not silently drop a step.
- **Sequence is part of the workflow.** Each step starts only after the previous step is `completed`. Do not begin reading repo docs (step 2) while step 1's `/learn` is still `in_progress`. Do not begin coding (step 4) while the plan (step 3) is still being drafted.
- **"Multi-step" test for ambiguous natural-language triggers.** If the ask plausibly resolves to a single file edit or a single tool call with no follow-up, the skill should not fire — handle it directly. If the ask plausibly entails any of {planning, more than one file, testing, documentation, a commit, closing an issue}, the skill fires. When the boundary is genuinely unclear, fire — the workflow tolerates small asks; the reverse is more expensive.
- If the user interrupts and redirects mid-workflow, update the task list to reflect the new shape — but do not abandon open steps without an explicit close.

## Red flags — STOP

| Sign you're off the workflow | What you should be doing |
| --- | --- |
| You're about to start working without running step 1 (`/learn`) | Stop. Open the workflow with `/learn` first. |
| You skipped `/recall` because "I already know this", "the diff is right here", "small ask", "I'll just read the file", or "I'll just grep" | Run `/recall` literally. Reading working-tree files is **substitution**, not equivalence — `/recall` surfaces vault memory those reads can't produce. The feelings that say "skip" are exactly the moments the gate counters. |
| You're writing code before the plan is committed (step 3) | Stop. Write the plan first, commit it, then execute. |
| You're skipping RED because "this is too simple to test" | Apply `superpowers:test-driven-development` regardless. |
| You declared a unit done without running the verifier | Apply `superpowers:verification-before-completion` before claiming done. |
| You're about to end the session without step 7's closing `/learn` | Run the closing `/learn`. The whole point of the bracket is symmetric capture. |
| The user gave you no ask and you started anyway | Stop. Ask the one clarifying question and wait. |
| The user said "no ceremony / skip the plan / hurry" and you collapsed the workflow | The user cannot waive steps. Run the workflow; it's fast when there's little to capture. |
| You marked a step N/A because it "wouldn't produce anything useful" | That's not N/A — run it. N/A is only for missing mechanism (no VCS, no transcript source). |
