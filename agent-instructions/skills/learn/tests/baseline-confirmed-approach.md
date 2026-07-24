# Baseline pressure test — kind 4: confirmed approaches (positive reinforcement)

Reusable RED/GREEN scenario inputs for the fourth Step-2 capture kind. RED = the pre-kind-4
three-kind skill writes nothing for these; GREEN = the current skill hands each a write-memory
handoff (kind=feedback). The guard cases must NOT fire under either version.

## GREEN scenarios (current skill must WRITE a confirmed-approach handoff)

**4a — user-confirmed (pure praise, no instruction).**
> ASSISTANT: I'll send the monthly summary as an attached CSV rather than pasting the table inline.
> USER: oh nice — attaching the CSV directly was way better, I could open it straight on my tablet
> instead of scrolling a giant inline table.

Expected: WRITE kind=feedback — behavior = attach deliverable files instead of inlining; impact =
the user's quote; action = keep attaching deliverables, trigger = producing a data/report deliverable.
(Note: no "do that going forward" — that would ALSO make it a kind-2 save-request. Pure praise is
the case kind 4a uniquely fills.)

**4b — self-validated bet.**
> ASSISTANT: Not sure whether to load the whole 4GB file into memory or stream it. I'll bet on a
> streaming parser. [implements it, runs the suite] Tests pass and peak memory stayed flat at ~40MB.

Expected: WRITE kind=feedback — behavior = stream large files rather than load fully; impact = tests
passed + flat memory confirmed the bet; action = prefer streaming for large-file parsing.

## Guard scenarios (must NOT write — over-capture guard)

- **Generic pleasantry:** USER: "thanks, this is great!" after a routine build fix → NO write (bare
  pleasantry names no behavior).
- **Routine success, no bet:** assistant reads a config, reports standard values, all fine → NO write
  (routine confident execution, no bet placed).
- **Ambiguous outcome:** assistant guesses a TTL, sets it, session ends before anything runs → NO
  write (no observable confirmation; the uncertainty is unresolved).

## RED (pre-kind-4 skill) vs GREEN (current skill)

- **RED:** the three-kind skill (corrections / save-requests / reversals) writes NOTHING for the two
  GREEN scenarios — a confirmed-good behavior and a self-validated bet match none of the three
  failure-shaped kinds. (Measured 2026-07-06: 6/6 reps wrote nothing.)
- **GREEN:** the current skill writes kind=feedback for both (6/6 reps), while the three guard cases
  stay silent (9/9 reps). Existing kinds 1–3 unchanged (9/9 reps).

## Failure modes that must FAIL this test

- Writing for any guard scenario (generic thanks / routine success / ambiguous outcome) — over-capture.
- NOT writing for either GREEN scenario — the gap kind 4 exists to close.
- Misfiling a GREEN scenario as kind 1/2/3 instead of kind 4 (4a is not a correction; 4b is not a
  reversal — a reversal is a bet that FAILED, a confirmed approach is one that SUCCEEDED).
