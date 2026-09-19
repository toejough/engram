## Context

`agent-instructions/guidance/shim.md`'s follow-frame rule 5 (backed by
`guidance-runbook-follow-frame`'s "stop and ask" requirement) currently reads:

> 5. **Stop and ask when a step's outcome is genuinely uncertain — not for every rough edge in its
>    wording.** If a step's specific instruction and its own incidental example seem to clash,
>    follow the specific instruction; resolving that by rereading is not, by itself, a reason to
>    stop. Do stop when a step names a target that plainly isn't present in front of you (no GitHub
>    remote, no ticket queue, no CI) — inventing "the local equivalent" yourself is still a
>    deviation, not a resolution of the step — or when two things the runbook requires are genuinely
>    in tension and you cannot tell which one the person needs preserved. Name the step and the
>    specific gap, then end your turn. This is a clarity signal for the person who gave you the
>    task — they own supplying what's missing, now and for next time — never a failure, and never a
>    reason to substitute your own approach and continue. Weigh this most heavily before an
>    irreversible action: if the honest answer is "I'm guessing," and what you're about to do can't
>    be undone, stop and ask instead of proceeding on your own judgment.

This rule already names several NOT-a-reason-to-stop cases (a wording clash resolvable by
rereading) and several genuine-ambiguity cases (a missing target, two requirements in tension).
It does not name the case this change addresses: an agent treating *completion of an earlier
step* — specifically, the step that produces the task's most visible deliverable — as itself a
reason to check in before continuing to a later step the runbook already marks mandatory.

Concrete evidence (`route-skill-to-runbook`'s D8 rerun, trial `route-R-0`, transcript message 87):
the agent verified a CLI `--version` flag implementation correct, then wrote (verbatim):

> Per the runbook I still owe the evidence record (a `fact` note + an aggregate update/creation
> for `route-evidence-cli-flag-implementation`, since the earlier lookup found no match). Should I
> go ahead and write those now, or would you rather I stop here since the code task itself is
> verified and complete?

The agent's own sentence ("I still owe...") shows it correctly resolved what the runbook requires.
It stopped anyway. This is not the clarity-signal case rule 5 exists for — no wording clashed, no
target was missing, nothing was in tension. The rule's current text doesn't say this stop is out
of bounds, so nothing in the shim contradicted it.

Filed and decided in issue #762 (https://github.com/toejough/engram/issues/762) and vault note
`1037.2026-09-18.question-stop-default-scoped-to-genuine-ambiguity`, which narrows vault note
`1030.2026-09-13.a-question-stop-is-a-clarity-signal-not-a-failure` (the original, unscoped
"never proceed anyway" rule).

## Goals / Non-Goals

**Goals:**
- Name the excluded case explicitly in both the shim's rule 5 and the backing spec requirement,
  so a future agent (or reviewer) can see it is out of bounds without re-deriving it from first
  principles.
- Keep every existing genuine-ambiguity case in rule 5 intact and enforceable exactly as before —
  this is an addition, not a loosening of the whole rule.

**Non-Goals:**
- Reworking route's own runbook body or the `route-skill-to-runbook` change. That change is
  separate (already applied; D8 still not met, for a different, already-diagnosed reason — the
  `red_flags`-truncation bug, issue #763). This change is shim-general and does not touch
  `agent-instructions/skills/route/` or any runbook note's own content.
- Adding a new eval harness. See Risks below.
- Changing how `red_flags`, `done_when`, or the announce/restate requirements work — untouched.

## Decisions

**D1. Narrow rule 5's own text, not a separate new rule.** The excluded case is a clarification of
what already-generously-scoped rule 5 does NOT cover, not a new behavioral category — it belongs
next to the rule's other "not, by itself, a reason to stop" clauses (the wording-clash one),
matching the existing rhetorical pattern rather than introducing a new numbered rule or a "the
floor" bullet. Alternative considered: add a bullet to "The floor beneath every runbook" (the
section with the shortcut/substitution/letter-is-spirit rules) — rejected, because that section is
about executing a step faithfully once you're doing it; this is about the *decision to stop before
doing it*, which belongs with rule 5's other stop-vs-proceed guidance.

**Proposed new rule 5 text** (change is the bolded lead-in addition and one new sentence inserted
after the existing wording-clash sentence; everything else is unchanged):

> 5. **Stop and ask when a step's outcome is genuinely uncertain — not for every rough edge in its
>    wording, and not because an earlier step is already done.** If a step's specific instruction
>    and its own incidental example seem to clash, follow the specific instruction; resolving that
>    by rereading is not, by itself, a reason to stop. Completing one step — even the one that
>    produces the task's most visible deliverable — is not, by itself, evidence that a later step
>    is ambiguous; when your own restated plan already names that later step as required, proceed
>    to it without asking whether to. Do stop when a step names a target that plainly isn't present
>    in front of you (no GitHub remote, no ticket queue, no CI) — inventing "the local equivalent"
>    yourself is still a deviation, not a resolution of the step — or when two things the runbook
>    requires are genuinely in tension and you cannot tell which one the person needs preserved.
>    Name the step and the specific gap, then end your turn. This is a clarity signal for the
>    person who gave you the task — they own supplying what's missing, now and for next time —
>    never a failure, and never a reason to substitute your own approach and continue. Weigh this
>    most heavily before an irreversible action: if the honest answer is "I'm guessing," and what
>    you're about to do can't be undone, stop and ask instead of proceeding on your own judgment.

**D2. Update the backing spec requirement's normative text and add one new scenario; keep the
requirement's title and the existing scenario's title unchanged.** Per this repo's own delta-spec
convention (vault notes 643/651: a MODIFIED block must reuse the base spec's requirement and
scenario headers verbatim, adding new scenarios under new names rather than renaming), the
requirement stays titled "The agent SHALL stop and ask when a step is unclear or would be deviated
from" and keeps its existing scenario "Unclear step yields a question, not a substitute" verbatim;
the new carve-out is a new sentence in the requirement body plus one new scenario, "A verified
earlier step does not license a stop before an unambiguous mandatory step." See
`specs/guidance-runbook-follow-frame/spec.md` in this change for the full delta text.

**D3. No new eval harness; do not re-run `probe_phase2.py --task route`.** This change is
shim-general — it should be verified generically, not by re-exercising route's own D8 metric
(which is already blocked on a separate, confirmed bug, #763, unrelated to this rule). Checked
`dev/eval/cumulative/runbook_vs_skill/`: every existing harness invocation (`--task route`,
`--task history-rewrite`, `--task bisect-before-fix`) is scoped to one specific runbook and its
own fixture; none is built to exercise "a generic runbook with a verified early step and an
unambiguous later mandatory step" as a shim-level property independent of which runbook it is.
Building that generic harness is out of scope for this change (real API spend, needs its own
sign-off) — see Risks.

## Risks / Trade-offs

- **[Risk] No behavioral verification before/after this change — only a wording change, unmeasured
  against real agent behavior.** → Mitigation: this is consistent with how shim.md changes are
  normally made (`writing-skills`-equivalent TDD: RED baseline, GREEN, pressure test, per the
  file's own header comment) — a RED trial reproducing the route-R-0 shape (a task with one
  verified early step and one unambiguous later mandatory step, run against the CURRENT shim)
  should be captured before editing, then rerun after, as this change's own pressure test. This is
  cheap (no real API spend required to write the RED capture from the transcript already in hand;
  a fresh confirming rerun is optional and would need explicit spend sign-off, matching this
  session's standing eval-spend convention).
- **[Risk] The added sentence could be read as encouraging agents to invent that a later step is
  "obviously required" when it actually isn't, skipping a stop that should have happened.** →
  Mitigation: the new sentence is conditioned on "your own restated plan already names that later
  step as required" — i.e., it only fires when the agent's OWN prior restatement (rule 2 of the
  follow frame, already mandatory) already committed to the step being required. It does not
  license inferring new obligations; it forbids re-litigating an obligation already stated.
- **[Trade-off] This narrows a rule vault note 1030 established generally two weeks prior (2026-09-
  13), less than a week before this session (2026-09-18).** → Not a reversal of 1030's core claim
  (question-stops on genuine ambiguity remain preferred, "proceed anyway" language remains
  forbidden) — narrows its scope, per note 1037's explicit supersession record.

## Migration Plan

1. Capture a RED baseline: reproduce or reuse the route-R-0 trial's shape against the CURRENT
   (unmodified) shim.md, confirming the same unnecessary stop, per D1's pressure-test approach.
2. Edit `agent-instructions/guidance/shim.md` rule 5 to the proposed text (D1).
3. Sync the delta spec into `openspec/specs/guidance-runbook-follow-frame/spec.md` (D2) — via
   `opsx:sync` or `openspec archive`, not a hand-edit.
4. Re-run the same reproduction against the UPDATED shim.md; confirm the stop no longer occurs and
   no existing genuine-ambiguity scenario regresses (spot-check against a case that SHOULD still
   stop, e.g. a step naming a missing GitHub remote).
5. Redeploy via `engram update --with-guidance`; confirm byte-identical sync at
   `~/.claude/engram/guidance/shim.md` and `~/.pi/agent/guidance/shim.md` (matches the archived
   `runbook-shim-follow-frame` change's own redeploy-and-confirm pattern).
6. No rollback machinery beyond `git revert` — this is a text-only change to one guidance file plus
   its backing spec.

## Open Questions

- Should the RED-baseline pressure test (Migration step 1) be a fresh live trial (real API spend)
  or a re-read of the ALREADY-CAPTURED route-R-0 transcript as sufficient RED evidence, since the
  exact failure is already on record from this session? Recommend the latter (no new spend) unless
  implementation finds the existing transcript insufficient to demonstrate the fix's effect
  end-to-end.
