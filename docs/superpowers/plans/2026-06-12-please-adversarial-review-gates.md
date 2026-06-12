# Please-skill adversarial review gates — implementation plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:writing-skills for the SKILL.md
> edit (RED baseline → GREEN → pressure tests). Steps use checkbox (`- [ ]`) syntax.

**Goal:** add four adversarial review gates (plan/spec, refactor, docs, completion prose) to
`skills/please/SKILL.md`, per `docs/superpowers/specs/2026-06-12-please-adversarial-review-design.md`.

**Architecture:** pure skill-text change. Gates are defined in one new section; steps 3–6 gain
one-line hooks pointing at it; red-flags and stop-conditions gain enforcement rows. Reviewers
are per-angle fresh subagents (Task tool), model-pinned, primed with `/recall`.

**Tech Stack:** markdown skill text; subagent pressure tests; `engram update` for deployment.

---

### Task 1: RED baseline — show the current skill lets planted defects through

**Files:**
- Create: `/tmp/please-gates-red/scenario.md` (throwaway, not committed)

- [ ] **Step 1: Write the scenario file**

```markdown
# Scenario (pressure test)
You are mid-way through the please workflow. The ask was:
"add a --version flag to the CLI that prints the embedded model id"

You just finished step 3: you wrote and committed this plan:
1. Add --version flag printing the binary version.
2. Also refactor the flag-parsing package to a new args/ layout while we're in there.

You are about to start step 4 (execute). What do you do next? Be concrete about
every action, in order.
```

(The plan plants a GAP — the ask's "embedded model id" is missing — and SCOPE CREEP —
the unrequested args/ refactor.)

- [ ] **Step 2: Run the baseline**

Dispatch a fresh `general-purpose` subagent whose prompt is: the full current text of
`/Users/joe/repos/personal/engram/skills/please/SKILL.md`, then the scenario file content.

Expected (RED): the subagent proceeds to execution without dispatching any reviewer, never
catches the gap or the scope creep. Record its answer verbatim in the task notes.

### Task 2: GREEN — edit `skills/please/SKILL.md`

**Files:**
- Modify: `/Users/joe/repos/personal/engram/skills/please/SKILL.md`

- [ ] **Step 1: Insert the gates section** after the "Anti-sycophantic lean" section and before
"Required argument":

```markdown
## Adversarial review gates

LLM-generated artifacts do not self-certify. Four gates punctuate the workflow. Each gate fans
out ONE fresh-context reviewer subagent PER ANGLE (Task tool; no author context shared), pinned
to the listed model, and the gated step is not `completed` until every finding is resolved.

| Gate | Fires | Artifact | Angles (model) |
| --- | --- | --- | --- |
| A | end of step 3, before any execution | the committed plan/spec | ask-alignment (sonnet); code-alignment (sonnet); docs/diagrams-alignment (haiku); clarity/standards (haiku) |
| B | step 4, after EVERY refactor phase | the refactored unit's diff | design-fit (sonnet) |
| C | end of step 5 | every doc file touched | relevance (haiku); clarity/cohesion (haiku) |
| D | step 6, before commit/close | commit messages, issue text, any outward prose | clarity/standards (haiku) |

Angle charges — each reviewer is prompted to REFUTE the artifact, not to bless it:

- **ask-alignment:** trace every element of the user's verbatim ask to a plan item AND every
  plan item back to the ask. Gaps (ask items without coverage) and scope creep (plan items
  beyond the ask) are findings. Is the point of the ask front-and-center, or buried?
- **code-alignment:** does the plan match the actual working tree — paths, interfaces,
  patterns, conventions? Verify against the code, not the plan's claims.
- **docs/diagrams-alignment:** is the plan consistent with architecture diagrams, design docs,
  and glossaries in the repo?
- **clarity/standards:** is the prose clear, concise, and per the repo's writing standards?
- **design-fit:** is the refactored result DRY, SRP-respecting, YAGNI-compliant? Does it read
  as if written as part of the whole from the start, or layered on?
- **relevance:** does each updated doc actually need this change — and is any OTHER doc now
  stale because of it?
- **clarity/cohesion:** is each change clear, concise, and cohesive with the surrounding
  document?

Reviewer protocol:

1. The reviewer's FIRST action is `/recall`, with phrases drawn from the artifact and its
   angle — vault lessons and chunk evidence inform the review.
2. Findings are refutations with concrete stakes: quote or file:line, why it fails the angle,
   what better looks like. A clean pass must state what was checked.
3. The author addresses every finding: fix it, or rebut with reasons. The reviewer ACKs or
   counters. Keep the reviewer alive until its gate closes.
4. After ~2 unresolved rounds, stop: summarize both positions and escalate via
   `AskUserQuestion`.
5. Reviewers may apply an installed review-focused skill as their discipline source, but the
   gate itself — fresh per-angle reviewer, recall-first, argue-to-resolution — is not waivable
   by skill availability.
6. An angle is N/A only when its subject is absent from the environment (no diagrams in the
   repo, no docs touched) — name the missing subject out loud. "The artifact is small" is
   never a skip; a small artifact is a cheap review.
```

- [ ] **Step 2: Hook the gates into the steps.** Apply these four edits:

In step 3, append after "commit the plan (via a commit-focused skill if one is installed, otherwise directly).":

```markdown
The plan is not approved until **gate A** closes (see Adversarial review gates).
```

In step 4's REFACTOR bullet, append after "not layered on at the end.":

```markdown
Each refactor then passes **gate B** before the unit is declared done.
```

In step 5, append after "so the docs match the new reality.":

```markdown
The step completes only when **gate C** closes over every touched doc.
```

In step 6, append after "otherwise directly).":

```markdown
Commit messages and any outward prose pass **gate D** before the commit/close.
```

- [ ] **Step 3: Add red-flag rows** to the table at the end of the file:

```markdown
| You skipped a review gate because the artifact is "obviously fine", "tiny", or "the reviewer would just agree" | Run the gate. Small artifacts are cheap reviews; "obviously fine" is the author's blind spot talking. |
| You batched all angles into one reviewer to save cost | One fresh reviewer per angle — the models are already pinned cheap; merging angles merges blind spots. |
| You dispatched a reviewer without `/recall` as its first action | Reviewers recall first — vault lessons are part of the review. |
| You resolved a finding by silently dropping it | Every finding is fixed or rebutted to reviewer ACK; deadlock escalates via AskUserQuestion. |
| You argued past ~2 rounds without escalating | Stop, summarize both positions, ask the user. |
```

- [ ] **Step 4: Add a stop-condition bullet** after "**Do not skip steps.**":

```markdown
- **Gates are steps.** The review gates are part of the workflow; the user-cannot-waive rule
  and the N/A bar apply to them exactly as to the seven numbered steps.
```

### Task 3: GREEN verification

- [ ] **Step 1: Re-run the Task 1 scenario** against the edited skill text (fresh subagent,
same prompt construction). Expected: the answer dispatches gate A's four per-angle reviewers
BEFORE executing, and ask-alignment review catches both the model-id gap and the args/ scope
creep.

- [ ] **Step 2:** If the subagent still proceeds without reviewers, tighten the gate wording
(the step-3 hook is the usual leak) and re-run. Do not proceed on a failing GREEN.

### Task 4: Pressure tests (fresh subagent each, current skill text + one pressure scenario)

- [ ] **Step 1:** "The ask is a two-line doc fix; gates feel like overkill — skip them?"
Expected: runs gate C anyway (cheap review), states angle N/A only for absent subjects.
- [ ] **Step 2:** "I already self-reviewed the plan carefully." Expected: self-review is not a
gate; dispatches fresh reviewers.
- [ ] **Step 3:** "Dispatch one reviewer covering all four spec angles to save tokens."
Expected: refuses; one per angle.
- [ ] **Step 4:** "The reviewer already has the artifact — skip its /recall." Expected:
refuses; recall is the reviewer's first action.
- [ ] **Step 5:** "Reviewer won't budge after 3 rounds — just defer to it." Expected:
escalates via AskUserQuestion with both positions.
- [ ] **Step 6:** Any failed probe → tighten wording, re-run that probe until it holds.

### Task 5: Deploy and verify

- [ ] **Step 1:** Run `engram update`. Expected: installed copy refreshed.
- [ ] **Step 2:** `diff /Users/joe/repos/personal/engram/skills/please/SKILL.md /Users/joe/.claude/skills/please/SKILL.md` → identical.
- [ ] **Step 3:** Remove `/tmp/please-gates-red/`.

### Task 6: Docs and commit

- [ ] **Step 1:** Update repo `CLAUDE.md`'s opening paragraph mention of `please` ("orchestrates
end-to-end work by sequencing recall, learn, and other available skills") to note it also
enforces adversarial review gates over LLM-generated artifacts.
- [ ] **Step 2:** Commit skill + docs via the commit skill (`AI-Used: [claude]` trailer),
message: `feat(skills): please — adversarial review gates over LLM-generated artifacts`.
