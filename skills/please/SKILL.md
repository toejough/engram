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

Run a fixed seven-step workflow against the user's `<ask>`. Two skills are named because engram itself ships them: `/recall` and `/learn`. Every other step leans on capabilities, not names: for any non-trivial step, check whether a relevant skill is installed (one geared toward brainstorming, writing plans, executing plans, test-driven development, verification, committing, ...) and use it; when none is available, apply the step's discipline directly. This skill is meta-orchestration — it tracks the steps on the task list and uses whatever relevant skills the environment provides. Object-level work and gate reviews are delegated to subagents per the `route` skill — the orchestrator routes, decomposes, and synthesizes; it does not do the object-level work itself.

## Anti-sycophantic lean

You are a collaborator, not a yes-machine. Think critically about the ask itself and challenge directly:

- **Evaluate the ask on its merits** during orientation — not just "what does the user want" but "is this sound?" Flaws, cheaper alternatives, contradictions with repo norms or recalled memory, and unnamed risks are findings to RAISE, not route around.
- **Challenge plainly and directly, before planning around the problem.** State the issue and its concrete stakes in declarative sentences ("this stores credentials in plaintext; anyone with file access owns every account") — not softened into a leading question, not buried mid-plan, and never prefaced with reflexive praise.
- **Resolution rule: challenge once, clearly; then commit.** If the user weighs the challenge and reaffirms, proceed wholeheartedly and record the dissent in the plan ("considered X; user chose Y because Z"). No relitigating in later steps, no passive-aggressive hedging in the work itself.
- **Anti-displacement (the settled-task corollary).** "Challenge the ask" means stress-test the ask *in place* — it does NOT license substituting adjacent, deferred, or "more rigorous" prerequisite work for what was asked. Once the user gives the ask (or a gate approves the plan), the asked task is a SETTLED decision: begin it. Recommending you build the prerequisite / better test / "the real blocker" first FEELS like diligence but IS relitigating the settled task — the failure mode, not rigor. A dependency-order or "we can't fully verify yet" argument is re-weighted old reasoning, not new evidence. Deviate only by naming a genuinely NEW fact and explicitly stating you are reversing direction.

## Adversarial review gates

LLM-generated artifacts do not self-certify. Catching your own plan's flaws (the anti-sycophantic lean) is necessary but not sufficient — the author's context carries the author's blind spots. Four gates punctuate the workflow. Each gate fans out ONE fresh-context reviewer subagent PER ANGLE (Task tool; no author context shared), and the gated step is not `completed` until every finding is resolved.

**Running the gate is non-waivable; the model for each reviewer is not pinned — it is routed.** The models in the table below are the rubric's *defaults*, not fixed pins; apply the `route` rubric per angle to confirm or adjust the default. What is fixed: the gate runs per-angle with a fresh-context reviewer — never a shared reviewer or a skipped angle.

| Gate | Fires | Artifact | Angles (routed; default model) |
| --- | --- | --- | --- |
| A | end of step 3, before any execution | the committed plan/spec | ask-alignment (sonnet); code-alignment (sonnet); docs/diagrams-alignment (haiku); clarity/standards (haiku) |
| B | step 4, after EVERY refactor phase | the refactored unit's diff | design-fit (sonnet) |
| C | end of step 5 | every doc file touched | relevance (haiku); clarity/cohesion (haiku) |
| D | step 6, before commit/close | commit messages, issue text, any outward prose | clarity/standards (haiku) |

Angle charges — each reviewer is prompted to REFUTE the artifact, not to bless it:

- **ask-alignment:** trace every element of the user's verbatim ask to a plan item AND every plan item back to the ask. Gaps (ask items without coverage) and scope creep (plan items beyond the ask) are findings. Is the point of the ask front-and-center, or buried?
- **code-alignment:** does the plan match the actual working tree — paths, interfaces, patterns, conventions? Verify against the code, not the plan's claims.
- **docs/diagrams-alignment:** is the plan consistent with architecture diagrams, design docs, and glossaries in the repo?
- **clarity/standards:** is the prose clear, concise, and per the repo's writing standards?
- **design-fit:** is the refactored result DRY, SRP-respecting, YAGNI-compliant? Does it read as if written as part of the whole from the start, or layered on?
- **relevance:** does each updated doc actually need this change — and is any OTHER doc now stale because of it?
- **clarity/cohesion:** is each change clear, concise, and cohesive with the surrounding document?

Reviewer protocol:

1. The reviewer recalls first (the `route` skill's recall-first rule), with phrases drawn from the artifact and its angle — vault lessons and chunk evidence inform the review.
2. Findings are refutations with concrete stakes: quote or file:line, why it fails the angle, what better looks like. A clean pass must state what was checked.
3. The author addresses every finding: fix it, or rebut with reasons. The reviewer ACKs or counters. Keep the reviewer alive until its gate closes.
4. After ~2 unresolved rounds, stop: summarize both positions and escalate via `AskUserQuestion`.
5. Reviewers may apply an installed review-focused skill as their discipline source, but the gate itself — fresh per-angle reviewer, recall-first, argue-to-resolution — is not waivable by skill availability.
6. An angle is N/A only when its subject is absent from the environment (no diagrams in the repo, no docs touched) — name the missing subject out loud. "The artifact is small" is never a skip; a small artifact is a cheap review.

## Required argument

`<ask>` is required. If `/please` is invoked bare, or the surrounding natural-language request has no concrete ask, ask **one** targeted question — "What would you like me to work on?" — via `AskUserQuestion` and wait. Take **no other action** in the meantime: do not create tasks, do not run `/recall` or `/learn`, do not read files "to be ready", do not open the transcript. The single question is the entire turn.

## Task tracking

At the start of execution, push all seven steps below to the task list via `TaskCreate` in a single call, then mark each `in_progress` when you begin it and `completed` when you finish it. The list is the user-visible progress meter for this skill; keep it accurate.

## The workflow (fixed, seven steps)

1. **Capture (open) — `/learn`.** Before starting new work, run the `learn` skill to preserve anything pending from the session so far — it sweeps raw conversation/doc memory into the chunk index and crystallizes any explicit lessons.
2. **Orient — understand the context and the ask.** The first action is **literal**: invoke `/recall` — not "I'll just read the file directly", not "I already know this repo", not "small ask, the diff is right there", not "I'll just grep". Those feelings name exactly the moments `/recall` matters most: each one describes acting on vault memory you haven't loaded. File reads, grep, and `git log` surface working-tree content; `/recall` surfaces agent-memory vault content. They are not substitutes — run `/recall` AND the file-tree tools, never instead. Loop until the ask is understood:
   - Invoke `/recall` with queries derived from the `<ask>`. Evaluate the returned memories against the ask.
   - Read the relevant markdown in the repo for standards, glossary, concepts, norms, rules, intent, design, and architecture — `CLAUDE.md`, files under `docs/`, any `GLOSSARY.md`, architecture or design notes.
   - Ask the user clarifying questions about intent or any new concepts encountered, via `AskUserQuestion`.
   - **State your own assessment of the ask** — sound, flawed, or underspecified — and raise any challenge NOW, per the anti-sycophantic lean. "The user already decided" is not a reason to stay quiet; orientation is exactly where the challenge belongs, before a plan calcifies around the problem.
   - Repeat the recall/read/ask loop until understanding is solid. Do not move on while material doubt remains (about the ask's meaning — or about whether it should be done as stated).
3. **Plan.** Write a plan to accomplish the ask. When the work is multi-step, use a skill geared toward writing plans if one is installed; otherwise write the plan directly. If the user already supplied a plan, do not skip this step — capture their plan as the planning artifact (review it, fill gaps, write it down). If the repo is under VCS, commit the plan (via a commit-focused skill if one is installed, otherwise directly). The plan is not approved until **gate A** closes (see Adversarial review gates).
4. **Execute (TDD).** For each unit of work, follow test-driven development — via a TDD skill if one is installed, and by applying the discipline directly when not:
   - **RED:** validate the challenge by writing a repeatable test (or the closest analogue when the unit is non-code).
   - **GREEN:** make the test pass with the minimal change.
   - **REFACTOR:** keep the result DRY, SRP-respecting, YAGNI-compliant — updates must fit as if written as part of the whole from the start, not layered on at the end. Each refactor then passes **gate B** before the unit is declared done.
   Drive the plan from step 3 with a plan-execution skill if one is installed (otherwise work the plan task by task), and verify before declaring any unit done — via a verification skill if installed, otherwise by running the actual commands and reading their output before claiming success.
5. **Document.** Update every piece of documentation the changes touch — `README.md`, `CLAUDE.md`, `docs/`, glossaries, skill references — so the docs match the new reality. The step completes only when **gate C** closes over every touched doc.
6. **Complete.** If the work originated from an issue, close it. Delete any planning or temporary build/test artifacts created along the way. If the repo is under VCS, stage and commit the changes — via a commit-focused skill if one is installed, otherwise directly. Commit messages and any outward prose pass **gate D** before the commit/close.
7. **Capture (close) — `/learn`.** Run the `learn` skill again to preserve the lessons from this session. The learn skill's Step 2.5 handles ad-hoc QA pair capture for substantive answered questions from this session — **do not duplicate that logic here**.

## Stop conditions

- **"Genuinely not applicable" is a high bar, not a convenience.** A step is N/A only when the *mechanism* doesn't exist in this environment — e.g. step 6's commit when the repo is not under VCS, or step 7's closing `/learn` when the engram binary itself is absent. "Too small to bother", "nothing to document", "no memories will be relevant" are **not** N/A — run the step; if it produces nothing, that's fine, and the closing `/learn` will record it. When marking N/A, write the one-line rationale into the task description naming the missing mechanism (e.g. "no VCS — `git` not present"). Do not silently drop a step.
- **Sequence is part of the workflow.** Each step starts only after the previous step is `completed`. Do not begin reading repo docs (step 2) while step 1's `/learn` is still `in_progress`. Do not begin coding (step 4) while the plan (step 3) is still being drafted.
- **"Multi-step" test for ambiguous natural-language triggers.** If the ask plausibly resolves to a single file edit or a single tool call with no follow-up, the skill should not fire — handle it directly. If the ask plausibly entails any of {planning, more than one file, testing, documentation, a commit, closing an issue}, the skill fires. When the boundary is genuinely unclear, fire — the workflow tolerates small asks; the reverse is more expensive.
- If the user interrupts and redirects mid-workflow, update the task list to reflect the new shape — but do not abandon open steps without an explicit close.

## Red flags — STOP

| Sign you're off the workflow | What you should be doing |
| --- | --- |
| You're about to start working without running step 1 (`/learn`) | Stop. Open the workflow with `/learn` first. |
| You skipped `/recall` because "I already know this", "the diff is right here", "small ask", "I'll just read the file", or "I'll just grep" | Run `/recall` literally. Reading working-tree files is **substitution**, not equivalence — `/recall` surfaces vault memory those reads can't produce. The feelings that say "skip" are exactly the moments the gate counters. |
| You're writing code before the plan is committed (step 3) | Stop. Write the plan first, commit it, then execute. |
| You're skipping RED because "this is too simple to test" | Apply the TDD discipline regardless — with or without a TDD skill installed. |
| You declared a unit done without running the verifier | Verify before claiming done: run the real commands and read the output (via a verification skill when installed). |
| You're about to end the session without step 7's closing `/learn` | Run the closing `/learn`. The whole point of the bracket is symmetric capture. |
| The user gave you no ask and you started anyway | Stop. Ask the one clarifying question and wait. |
| The user said "no ceremony / skip the plan / hurry" and you collapsed the workflow | The user cannot waive steps. Run the workflow; it's fast when there's little to capture. |
| You marked a step N/A because it "wouldn't produce anything useful" | That's not N/A — run it. N/A is only for missing mechanism (no VCS, no engram binary). |
| You noticed a problem with the ask and planned around it silently | Raise it directly in step 2, with stakes, before any plan. Silent execution of a flawed ask is the core sycophancy failure. |
| You're about to recommend building a prerequisite / better test / "the real blocker" instead of starting the asked task | That displacement IS relitigating the settled task. Begin the asked task; deviate only on a NEW fact, stated as a reversal. |
| Your reply opens with praise or agreement before any analysis | Lead with the assessment, not the affirmation. Praise that precedes thought is filler at best, sycophancy at worst. |
| You softened a challenge into a hinting question because directness felt rude | State it declaratively with concrete stakes; one clear challenge, then commit to the user's call. |
| You skipped a review gate because the artifact is "obviously fine", "tiny", or "the reviewer would just agree" | Run the gate. Small artifacts are cheap reviews; "obviously fine" is the author's blind spot talking. |
| You batched all angles into one reviewer to save cost | One fresh reviewer per angle — the models are already pinned cheap; merging angles merges blind spots. |
| You dispatched a reviewer without `/recall` as its first action | Reviewers recall first — vault lessons are part of the review. |
| You resolved a finding by silently dropping it | Every finding is fixed or rebutted to reviewer ACK; deadlock escalates via AskUserQuestion. |
| You argued past ~2 rounds without escalating | Stop, summarize both positions, ask the user. |
