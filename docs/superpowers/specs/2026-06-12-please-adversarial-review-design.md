# Please-skill adversarial review gates — design

**Goal:** the please workflow stops trusting its own LLM-generated artifacts. Specs/plans,
refactors, doc updates, and completion prose each pass an adversarial review gate before the
workflow proceeds. Reviewers are fresh-context subagents — one per angle — pinned to cheap
models, primed with `/recall`, prompted to refute, and argued with to consensus.

**Decided with Joe (2026-06-12):**

- Mechanism: fresh-subagent adversarial review (not inline self-checklists).
- Gated artifacts: spec/design + plan, doc updates, all LLM-written prose, and the REFACTOR
  phase of TDD (red/green stays as-is; the refactor itself gets reviewed).
- Batching: one reviewer per angle; model pinned per angle (sonnet or haiku by judgment-load);
  every reviewer runs `/recall` before reviewing.
- Resolution: argue to consensus; after ~2 rounds of deadlock, summarize and escalate to the
  user via AskUserQuestion.

**Proposed by the agent, ratified by Joe (2026-06-12, "this is great, please proceed"):**

1. The angle taxonomy below (Joe set "per angle" but did not enumerate angles).
2. The sonnet/haiku split below.
3. Gate B fires after EVERY REFACTOR phase (a 6-unit plan ⇒ 6 refactor reviews).
4. Proportionality via angle-level N/A: an angle whose subject is absent (no diagrams in repo,
   no docs touched) is skipped with the missing mechanism named — the skill's existing N/A bar.
5. Ask-alignment is bidirectional: gaps (ask items with no plan coverage) AND scope creep
   (plan items beyond the ask) are findings.

## Gate placement in the seven-step workflow

| Gate | Fires | Reviews |
| --- | --- | --- |
| A | end of step 3, before any execution | the committed plan/spec |
| B | inside step 4, after each REFACTOR | the refactored unit's diff |
| C | end of step 5 | every doc file touched |
| D | inside step 6, before push/close | commit messages, issue text, any other outward prose |

## Angles, reviewer charge, and models

| Gate | Angle | Reviewer's charge (prompted to REFUTE) | Model |
| --- | --- | --- | --- |
| A | ask-alignment | Trace every element of the user's ask to a plan item and every plan item back to the ask. Gaps and scope creep are findings. Is the point of the ask front-and-center, or buried? | sonnet |
| A | code-alignment | Does the plan match the actual code — paths, interfaces, patterns, conventions? Verify against the working tree, not the plan's claims. | sonnet |
| A | docs/diagrams-alignment | Is the plan consistent with architecture diagrams, design docs, glossaries? | haiku |
| A | clarity/standards | Is the plan clear, concise, and per repo writing standards? | haiku |
| B | design-fit | Is the refactored result DRY, SRP-respecting, YAGNI-compliant? Does it read as if written as part of the whole from the start, or layered on? | sonnet |
| C | relevance | Does each updated doc actually need this change? Is any OTHER doc now stale because of it? | haiku |
| C | clarity/cohesion | Is each change clear, concise, and cohesive with its surrounding document? | haiku |
| D | clarity/standards | Are commit messages / issue text clear, concise, and per convention (Conventional Commits, repo trailer rules)? | haiku |

## Reviewer protocol (one subagent per angle)

1. Dispatched fresh (no author context) with: the artifact, the verbatim ask, the angle charge,
   and the model pin.
2. First action: run `/recall` with phrases drawn from the artifact + angle, so vault lessons
   and chunk evidence inform the review.
3. Produce findings as refutations with concrete stakes — file:line or quote, why it fails the
   angle, what better looks like. "Looks fine" requires having stated what was checked.
4. Author addresses every finding: fix, or rebut with reasons. Reviewer ACKs or counters.
5. After ~2 unresolved rounds: stop, summarize both positions, AskUserQuestion.
6. The gate is closed only when every finding is fixed or reviewer-ACKed (or user-resolved).
   Keep the reviewer alive until its gate closes (repo norm: arguing agents stay alive together).

## Proportionality

- Angle-level N/A (missing subject) is the only skip, and it is stated out loud with the
  missing mechanism named. "The artifact is small" is never a skip — a small artifact is a
  cheap review.
- Cost ceiling per ask ≈ 4 sonnet + 4–5 haiku reviews for a typical single-unit ask; gate B
  scales with refactor count.

## Skill-text changes (skills/please/SKILL.md)

- New "Adversarial review gates" section defining gates A–D, the angle table, reviewer
  protocol, and resolution rule — in capability language (a code-review skill, if installed,
  may serve as gate B/D's discipline source; the gate itself is not waivable).
- Step 3 gains: "the plan is not approved until gate A closes."
- Step 4's REFACTOR gains gate B; step 5 gains gate C; step 6 gains gate D.
- Red-flags table gains rows: skipping a gate because the artifact "is obviously fine" /
  "reviewer would just agree"; reviewers dispatched without `/recall`; gate findings addressed
  by silently dropping them; arguing past 2 rounds without escalating.
- Stop-conditions: gates are steps — the user-cannot-waive rule applies to them too.

## Testing (writing-skills TDD)

- **RED:** fresh agent runs the current skill on a toy ask whose plan contains a planted
  misalignment (scope creep + a gap) and a planted incohesive doc edit; document that the
  current text lets both through unreviewed.
- **GREEN:** same scenario on the edited skill; agent dispatches per-angle reviewers, the
  planted defects surface as findings, argumentation resolves, gates close before proceeding.
- **Pressure tests:** "tiny ask, skip the gate"; "I already self-reviewed"; "reviewer is
  expensive, batch all angles into one"; "skip /recall in the reviewer, it has the artifact";
  "deadlock — just defer to the reviewer instead of escalating".

## Out of scope

- No new engram binary support; gates are pure skill-text orchestration.
- `commands/please.md` is a thin pointer — unaffected.
- The constituent skills (brainstorming's self-review, code-review skills) keep their own
  internal checks; the gates sit above them in the orchestrator.
