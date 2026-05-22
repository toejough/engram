# Plan — Issue #634: Path C (retrojective framing) for prior-session injection locus

## Goal

Add a third framing path to the learn skill that triggers when a lesson's
**injection locus** is a prior session, even though the candidate was
identified during current-session work. Currently Path A (recall ran this
session and brackets the candidate's segment) fires for such candidates and
anchors the `--situation` against current-session work — mis-framing the
retrieval target.

## Locus discriminator

A candidate is **retro-locus** when the action that caused the lesson
happened *before this session*. Signal sources (in order of strength):

1. **Concrete code/config locus** — `git blame` or `git log` on the
   offending line/file shows authorship before this session's first commit.
2. **Session-log locus** — `engram transcript --mark` (already in
   context for /learn) shows the mistake originating in a session prior to
   today.
3. **Behavioral / conceptual** — no file-level locus, but the lesson is
   about a misconception or correction the agent carried into this session
   from prior work. Discriminator: would I have done the wrong thing
   independently of this session's stated plan? If yes, retro.

A candidate is **current-locus** when the mistake or discovery
originates in *this session's* work — even if it touches old code.

## Changes

1. **`skills/learn/SKILL.md`:**
   - §1 Identify candidates: add the injection-locus signal — note that
     candidates surfaced from current-session work may still have a prior-session
     injection locus, and call out git blame / transcript-marker / behavioral
     signals.
   - §2 Anchor: introduce **Path C — retrojective framing** as a third
     selector. Path C applies when the candidate is retro-locus; reconstruct
     the scratch list from what the *injecting* agent was doing (commit
     message, prior-session transcript context, or behavioral inference),
     not from any recall that bracketed the current-session *discovery*.
     Selection order: classify locus first → then pick Path A / Path B / Path C.
   - §3 Apply mirror test: extend the failure-mode table with a row for
     "scratch list anchored on the discovery situation rather than the
     injection situation — re-select via Path C".
   - §4 Categorize: append a locus check — "who made the mistake — me this
     session or someone earlier?" If earlier, Path C applies.
   - Core principle blockquote (currently "the same kind of work I started
     **this session** with"): re-phrase to "the same kind of work the
     **injecting agent** was doing" or generalize to "the kind of work this
     candidate's scratch list targets" — session-agnostic.
   - Red-flags table: add row for "anchored on current-session framing for a
     retro-locus candidate".

2. **`docs/GLOSSARY.md`:**
   - Update §recall-mirror test (currently "the situation I started this
     session with") to be candidate-anchored, locus-agnostic.
   - Update §scratch list to mention Path C reconstruction from injecting
     agent's situation.
   - Update §Path A / Path B section → rename to §Path A / Path B / Path C
     with Path C definition (retro-locus → reconstruct from injecting
     agent's situation via git blame / transcript / behavioral inference).
   - Add a §injection locus entry.

## TDD per `superpowers:writing-skills`

- **RED:** dispatch a roleplay subagent with the *current* `skills/learn/SKILL.md`
  text and §2 in isolation; scenario: "today during a docs cleanup (a recall
  bracketed this cleanup) you discovered `--from` is wired to the wrong
  config field by commit abc123 six commits ago. Execute §2: which scratch
  list do you build?" Expected current behavior: anchors on docs-cleanup
  phrases (Path A). RED confirmed.
- **GREEN:** apply the §1-§4 edits above, re-dispatch with the same scenario,
  expect: anchors on config-wiring phrases (Path C).
- **REFACTOR / over-broadening pressure test:** near-miss scenario where a
  current-session candidate happens to touch old code but the lesson is
  about *this session's* action (e.g., "you wrote a flaky test today that
  references an old helper"). Path C should NOT fire — Path A/B applies as
  usual.
- All subagent tests use stubbed candidates per Permanent/12c rather than
  running the whole pipeline.

## Out of scope

- Adding any new flags to `engram learn` (Path C is a workflow-only change
  in the skill prose; situations and scratch lists are LLM-side).
- Touching `engram transcript` semantics.
- Updating any code in `internal/` — this is a skill + docs change.

## Done when

- Three roleplay subagent tests (RED reproduces, GREEN passes, REFACTOR
  near-miss does not over-fire) all return the expected behavior with the
  updated skill text.
- `targ check-full` clean.
- Issue #634 closed via commit.
- Plan file deleted before final commit (planning artifact, not durable).
