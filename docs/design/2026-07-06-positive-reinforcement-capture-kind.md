# Plan — Positive-reinforcement (confirmed-approach) capture kind for `learn` Step 2

**Cycle scaffolding.** Committed at /please Step 3, retired at Step 6 (git log recovers it).
Durable records land in the skill text, GLOSSARY, the C1/C2 diagrams, ADR-0017, and ROADMAP.
Revised after Gate A (all four angles); every finding is resolved in the Design, Files-touched,
and TDD sections below.

Issue: #668. Extends the ask live (Joe, session 2026-07-06): capture not only user-praised
behavior but also self-validated guesses/uncertainty that panned out.

## Ask (verbatim scope)

1. Add a positive-reinforcement / confirmed-approach capture kind to `skills/learn/SKILL.md`
   Step 2. Today Step 2 scans only three **failure-shaped** kinds (corrections, save-requests,
   reversals); a confirmed-good behavior has no capture kind and survives only as weak raw chunks.
2. **Extension (Joe, 2026-07-06; verbatim with dictation stutters bracketed):** "I don't want a
   memory of every successful tool call, but whenever we make a guess, or are uncertain about a
   plan[,] concept or idea and try it out[] and it works, that should also be recorded as positive
   reinforcement and learning."
3. Reconcile the recently-shipped routing-updates work, which depends on this for recording what
   worked in a session (`skills/route/SKILL.md:105`, `docs/architecture/adr.md:412` both name #668
   as the "cleaner home" for the pure-confirmation tier signal).

## Design — the fourth capture kind

Add **kind 4 — Confirmed approaches (positive reinforcement)** to Step 2, after reversals. It is
the *positive mirror* of the two negative kinds, spanning both attribution axes:

| | Negative (something was wrong) | Positive (something was right) |
| --- | --- | --- |
| **User-attributed** | kind 1 — correction | **kind 4a** — user confirms a specific behavior |
| **Self-discovered** | kind 3 — reversal | **kind 4b** — a genuine bet/uncertainty that panned out |

(kind 2, save-requests, is orthogonal — an explicit instruction, not a valence.)

### Trigger 4a — user-confirmed behavior

Fires when the user explicitly praises/thanks a **specific behavior you would recognize and
reapply** in a future similar situation. The behavior must be *named or clearly implied*.

- **Operational test:** could you state the behavior as a reusable tactic ("attach deliverable
  files rather than pasting them inline")? If yes → fires. If the praise names no behavior
  ("thanks!", "great work", "appreciate it") → does **not** fire.
- **Example (note 175):** "thanks for attaching the file — easier to read from my phone" →
  behavior = attach deliverable files instead of inlining; impact = the user's quote; action =
  attach for phone-reading, with its trigger (delivering a report/doc).

### Trigger 4b — self-validated bet

The exact positive mirror of kind 3 (reversal). Fires when **all three** hold:

1. **Genuine uncertainty at decision time** — you faced a real bet where you could have been
   wrong, and you can name the doubt or the alternative you did not take. (A reversal is such a
   bet that *failed*; a confirmed approach is one that *succeeded*.)
2. **You acted on it** — the bet was embodied in the work: a chosen approach, a written
   implementation, a plan step, a command actually run. Not a passing internal thought.
3. **Observable confirmation** — a **session-recorded** outcome resolved the uncertainty in the
   bet's favor: a test passed, the user said it worked, an artifact was produced and functioned, a
   blocker was cleared. Passage of time or the mere absence of a visible error is **not**
   confirmation; if the outcome is ambiguous, do **not** capture.

Note the bet need not have been *presented to the user* (that would under-cover the ask's
"whenever we make a guess") — acting on it in the work is the bound, plus observable confirmation.

- **Example 1 (approach bet):** "unsure whether a retry loop or fail-fast was right for the flaky
  call; chose the retry loop and ran it — zero timeouts across the test suite → confirmed the
  approach worked." action = prefer the retry loop for this class of flaky call.
- **Example 2 (self-validated guess):** "wasn't sure swapping the embedder model would preserve
  search quality; ran the baseline recall eval — recall@5 held at 0.83 → the uncertainty
  resolved." action = the model swap is safe on this retrieval metric.

### write-memory handoff (same shape as kind 1 — reuse the worker, note 166)

kind=feedback, slug, source, situation (retrieval-shaped: when would this approach apply again),
behavior = what worked, impact = the confirming evidence (the user's quote for 4a, OR the observed
outcome that resolved the uncertainty for 4b), action = keep doing it + its trigger conditions.
Plus supersedes details if it corrects an existing note.

### Over-capture guard (the crux — Joe: "I don't want a memory of every successful tool call")

Kind 4 fires only when a **specific, generalizable approach** meets a **genuine goodness signal**
(4a explicit confirmation, or 4b's acted-on bet + observable confirmation). It is stated in both
positive form (what fires, above) and negative form below **by design** — per note 137, a
behavioral rule names the action *and* explicitly forbids the substitute, and the negatives are
the exact pressure-test targets.

Does **not** fire on: generic pleasantries with no behavior attached; routine confident execution
that merely happened to succeed (no bet was placed); the mere absence of failure. The parallel is
exact: just as a repo-doc CORRECTION does not count as reversal-capture, a routine success does not
count as reinforcement-capture. The load-bearing signal is a **resolved uncertainty** or an
**explicit specific confirmation**, never "it worked."

## Files touched

Primary skill + its mirrors:
- `skills/learn/SKILL.md` — add kind 4; update intro (l14), frontmatter description (l4-9),
  "exactly three kinds" → four (l56), Rules "no ... reversals" list (l95-96), red-flags table
  (l124 + a new over-capture row). Re-sync to `~/.claude/skills/learn/SKILL.md` after (deployed
  copy is what fires and what the eval harness copies from `$HOME_CFG`).
- `dev/eval/guards/candidate/learn.md` (l56, l96, l124) — the **shipped-form** guard fixture
  (introduced by commit 9ab7c156 "candidates carry … the shipped form"; currently near-identical
  to the skill, one stale ROADMAP line aside). Apply the same kind-4 edits so the guard eval keeps
  testing the shipped skill, not a stale three-kind invariant.

Docs stating the three-kind invariant (doc-scrub-is-part-of-the-change, notes 62/64):
- `docs/GLOSSARY.md` — add a `### confirmed approach (capture kind)` entry beside `reversal
  (capture kind)` (l70); bump "exactly three capture kinds" → four in the Step-2 gate entry (l350);
  add the failure-shaped cross-ref clause to the `lessons audit` entry (l78-86, see please below).
- `docs/architecture/c1-system-context.md` (l225) — the **L1** learn sequence-diagram Note says
  "scan THIS session for exactly three kinds …"; update to four + the confirmed-approach branch.
  ADR-0016 requires mermaid diagrams be verified against code at edit time — non-optional.
- `docs/architecture/c2-containers.md` — **two** spots: the sequence-diagram Note (l184, "scan for
  the three explicit lesson kinds …") and the capture-kinds flowchart (l196-222). Add the kind-4
  branch to the flowchart; bump the "four capture kinds" prose (l198) to five (4 Step-2 kinds + 1
  Step-2.5 QA). Update the l184 note to four.
- `README.md` (l45) — skill table: "corrections, explicit save-requests, and self-discovered
  reversals" → add "and user-confirmed / self-validated confirmed approaches".
- `docs/FEATURES.md` (l89-96) — the write-memory/capture-guards feature blurb (l93 "Learn also
  captures self-discovered reversals as their own lesson kind") → add the confirmed-approach kind.

Dependency reconciliation (route work depends on this):
- `skills/route/SKILL.md` (l104-105) — replace the "save-request to yourself … would be its cleaner
  home" text. **New prose:** "crystallize it via `/learn`: a **confirmed-approach** note (kind 4)
  when a tier's outcome confirms the routing (e.g. 'cheap sufficed for work-kind K'), or a
  **reversal** if it overturns a prior tier assumption."
- `docs/architecture/adr.md` (l411-413) — replace "has no dedicated `/learn` moment-kind yet …
  #668 … is the cleaner home." **New prose:** "The pure-confirmation signal ('cheap sufficed for
  K', overturning nothing) is captured by `/learn`'s kind-4 confirmed-approaches moment (positive
  reinforcement, shipped 2026-07-06): a tier that passed for a work-kind crystallizes as a
  confirmed approach, a tier that failed as a reversal."
- `docs/ROADMAP.md` (l119-120) — mark the #668 item shipped/closed.

please step-7 consistency (issue AC — "check for consistency; its corpus enumeration is
failure-shaped too"):
- **Decision:** keep the step-7 lessons audit failure-shaped. Its corpus is *mechanical* (fired
  STOPs, gate FAILs, marker-commits, escalations) — a completeness gate against silently dropping
  failure lessons. Positive moments have no mechanical marker analogue; they are caught by the
  closing /learn's Step-2 kind-4 behavioral scan. (YAGNI: do not invent a mechanical positive
  marker.)
- `skills/please/SKILL.md` (Step 7, ~l92-100) **and** `docs/GLOSSARY.md` `lessons audit` entry
  (l78-86) — add **this cross-ref clause:** "The audit is failure-shaped by design — its corpus is
  the cycle's mechanical failure markers. Positive reinforcement (confirmed approaches,
  self-validated bets) has no mechanical marker and is captured by the closing `/learn`'s Step-2
  kind-4 scan, not here."

Not touched (verified): `commands/learn.md` (thin pointer — "Invoke the learn skill", no
capture-kind content); `docs/GLOSSARY.md` `capture guards (G1–G6)` entry (l604 — that named set is
specifically the reversal-blind-spot guard family, unaffected by this positive mirror).

## TDD (superpowers:writing-skills — non-waivable, repo rule + notes 26/296)

- **RED baseline:** two fixtures ending in a positive-reinforcement moment run against the CURRENT
  skill text write nothing — (a) user thanks a specific behavior; (b) a self-validated guess that
  worked. Confirms both fall through the three failure-shaped kinds.
- **GREEN:** the new skill text makes each fixture fire a write-memory handoff (kind=feedback).
- **Pressure tests (guard + regression):** (a) generic "thanks!" → no fire; (b) a routine
  successful tool call with no bet → no fire; (c) an ambiguous-outcome guess (no observable
  confirmation) → no fire; (d) the three existing kinds unchanged (correction → k1, save-request →
  k2, reversal → k3).
- Test mechanics follow writing-skills; for behavioral RED/GREEN use fresh headless framing, not
  session-inheriting subagents, and neutral (not spotlighted) prompts (notes 138 + headless-not-
  subagents + 137: name the action, forbid the substitute).

## Acceptance criteria (from issue #668 + extension)

- [ ] `skills/learn/SKILL.md` Step 2 has kind 4 covering both 4a (user-confirmed) and 4b
      (self-validated bet, with the three-part guard); over-capture guard present; three existing
      kinds unchanged.
- [ ] writing-skills RED→GREEN→pressure evidence (generic praise no-fire; routine success no-fire;
      ambiguous-outcome no-fire).
- [ ] GLOSSARY gains the confirmed-approach capture-kind entry; "three" → "four" count updated.
- [ ] C1 + C2 diagrams updated (both notes + flowchart); ADR-0017 + ROADMAP updated; route SKILL.md
      dependency reconciled with the new prose above.
- [ ] README.md + FEATURES.md capture-kind summaries updated.
- [ ] `dev/eval/guards/candidate/learn.md` synced to the shipped kind-4 form.
- [ ] please step-7 consistency checked; cross-ref clause added to please + GLOSSARY.
- [ ] Deployed `~/.claude/skills/learn/` re-synced.
- [ ] Issue #668 closed.
