---
name: learn
description: >
  Use after completing any action we started with a recall call, or after any work that involved more than one tool
  call or more than quick shallow thinking. Also use immediately on explicit save-requests like "remember this",
  "remember that X", "save that for later", "note for next time", "don't forget X", "write this down". Also use at
  session end when a conclusion, design, or finding you presented was later overturned or corrected. Also use when a
  specific approach was confirmed to work — the user explicitly praised a specific behavior, or a genuine guess or
  uncertain call you acted on turned out right. This preserves relevant memories that are VITAL to recall for a good
  user experience and a greater chance at first-pass success for similar work in the future.
---

# Learn — Sweep Raw Memory, Crystallize Explicit Lessons

Two jobs, in order: (1) mechanically true up the chunk index so every conversation and doc is searchable memory, and (2) write a vault note for every explicit lesson this session — user corrections, explicit save-requests, presented conclusions that were later overturned, and confirmed approaches (user-praised or self-validated). Nothing else.

> **Raw event memory is AUTOMATIC.** `engram ingest` chunks and embeds session transcripts and
> markdown itself — no summaries, no episode notes, no arc detection. The agent's only writing
> job is crystallizing EXPLICIT lessons. Do not reconstruct the old episode workflow
> (`engram transcript`, `engram learn episode`) — it is gone.

## Step 1 — Sweep (first; the closing learn always sweeps)

```bash
engram ingest --auto
```

That's it. The binary stats every known source (repo markdown, `.claude` dirs, all session
transcripts), re-chunks and re-embeds only what changed — existing chunks are never deleted (append-only history). Unchanged
corpus → returns in seconds. Report the one-line tally it prints (or "memory index up to date").

If the command fails, surface the error and continue to Step 2 — explicit lessons must not be
lost to an ingest hiccup.

Skip the sweep if one already ran earlier in THIS session, UNLESS this is the closing learn of a
work cycle — the closing learn ALWAYS sweeps (it captures the session's tail for future sessions).
This holds even under a "something might have changed outside this session (another terminal,
another agent)" worry — that gets picked up by the next sweep (yours or theirs), not by
re-sweeping now on suspicion.

## Step 1.5 — Vocab liveness check

Run `engram vocab stats`.

If the output includes a line matching `verdict: REFIT_PENDING (<reason>)`, run the vocab
refit flow autonomously — do not defer to the user:

1. Run `engram vocab refit --emit-request`. Save its JSON output.
2. Derive a YAML refit plan from the JSON (review terms, propose merges/splits/removals for
   orphans < 2 members and hubs > 25%). Write the plan to `/tmp/vocab-refit-plan.yaml`.
3. Run `engram vocab refit --plan /tmp/vocab-refit-plan.yaml` to apply the plan.
4. **Report loudly:** "Vocab refit applied: <version bump>. Triggered by: <reason>."

If the verdict is `verdict: OK`, continue to Step 2 with no further vocab action.

Also check the QA round-2 gate line. If the output includes `qa round-2 gate: READY (...)`, report to Joe:
"QA round-2 validation is due (≥20 pairs captured). Please schedule the round-2 gates recorded in
docs/ROADMAP.md (The roadmap → GATED → Q&A memory round-2): P2′ attribution fidelity, P3′ distribution, Arm V larger-n." Do NOT run round-2 validation
autonomously — it requires Joe's oversight.

## Step 2 — Crystallize explicit lessons (only when they exist)

Scan THIS session for exactly four kinds of moments:

1. **Corrections** — the user corrected your approach or behavior ("don't suppress lint warnings —
   fix the underlying issue", "never amend pushed commits").

   **REQUIRED SUB-SKILL:** invoke the **write-memory** skill with this handoff — kind=feedback,
   slug, source ("session <date>, context: <one-line what-was-happening>"), situation
   (retrieval-shaped), behavior, impact, action; plus supersedes details if this correction
   corrects an existing vault note. write-memory composes, executes, and reports the note path.

2. **Explicit save-requests** — the user said "remember this/that X", "note for next time",
   "write this down".

   **REQUIRED SUB-SKILL:** invoke the **write-memory** skill with this handoff — kind=fact,
   slug, source ("session <date>, context: <one-line what-was-happening>"), situation
   (retrieval-shaped), subject, predicate, object; plus supersedes details if this fact
   corrects an existing vault note. write-memory composes, executes, and reports the note path.

3. **Reversals** — a conclusion, design, or verdict that was PRESENTED (to the user, a review
   gate, or a committed plan) and later OVERTURNED — by you, a reviewer, or an instrument
   (a superseded design, a retro-invalidated finding, an instrument-invalid measurement, a
   redrawn boundary). Nobody needs to have SAID the correction — self-discovered reversals
   qualify, and a repo-doc CORRECTION section or postscript does NOT count as capture
   (record-correction ≠ lesson-capture). For each reversal, **REQUIRED SUB-SKILL:** invoke the
   **write-memory** skill with this handoff — kind=feedback, slug, source ("session <date>,
   context: <one-line what-was-happening>"), situation (retrieval-shaped: when does this
   failure mode apply), behavior = what the original reasoning did wrong, impact = what the
   reversal cost, action = the guard that would have prevented it — the ROOT CAUSE, not a
   narrative of the flip; plus supersedes details if the reversal corrects an existing vault
   note.

4. **Confirmed approaches (positive reinforcement)** — a specific, generalizable approach was
   validated as good, either by the user or by the outcome. The positive mirror of kinds 1 and 3;
   it fires on EITHER trigger:
   - **4a — user-confirmed:** the user explicitly praised or thanked a SPECIFIC behavior you could
     restate as a reusable tactic ("thanks for attaching the file — easier to read from my phone" →
     "attach deliverable files instead of inlining them"). A bare pleasantry naming no behavior
     ("thanks!", "great work") does NOT qualify. (If the user also instructs "do that going
     forward", that is additionally a save-request — kind 2.)
   - **4b — self-validated bet:** you made a genuine guess, or were uncertain about a plan, concept,
     or idea, ACTED on it (embodied it in the work — a chosen approach, an implementation, a plan
     step, a command run), and an OBSERVABLE, session-recorded outcome then confirmed it worked (a
     test passed, the user confirmed it, an artifact functioned, a blocker cleared). The exact
     positive mirror of a reversal: a bet that succeeded instead of failing.

   The action is capturing what WORKED — NOT logging every success. A bare pleasantry, a routine
   success with no bet behind it, or an unconfirmed guess is never the signal — just as a repo-doc
   CORRECTION does not count as reversal-capture, a routine success does not count as
   reinforcement-capture; the signal is a resolved uncertainty or an explicit specific
   confirmation, never "it worked". For each confirmed approach, **REQUIRED SUB-SKILL:** invoke the
   **write-memory** skill with this handoff — kind=feedback, slug, source ("session <date>,
   context: <one-line what-was-happening>"), situation (retrieval-shaped: when would this approach
   apply again), behavior = what worked, impact = the confirming evidence (the user's quote for 4a,
   or the observed outcome that resolved the uncertainty for 4b), action = keep doing it + its
   trigger conditions; plus supersedes details if this confirmation corrects an existing vault note.

Rules:
- **State the general principle**, not the session-specific instance — future-you recalls by
  situation similarity, not by remembering this session.
- The `--situation` phrase is the retrieval handle: phrase it the way a future task would be
  described ("releasing a Go module", "writing eval harness metrics").
- One note per distinct principle. An explicit save-request ALWAYS gets its note, immediately —
  "remember this" means stop and write before anything else.
- If the new lesson CORRECTS, narrows, or refutes an existing vault note, include the superseded note's basename, type, and claim in the handoff.
- **No moments of any kind → write nothing.** Routine work is already captured by Step 1;
  a session with no corrections, no save-requests, no reversals, and no confirmed approaches is a two-command learn (sweep + report).

## Step 2.5 — Ad-hoc QA capture (only when a new substantive Q&A occurred this session)

Scan THIS session for substantive answered questions: a question was substantively answered if
the answer body contains ≥1 `[[full-basename]]` wikilink OR if you crystallized a
new vault note (Step 2) as the answer. Both conditions make the answer traceable (D2 observable
bar). Skip questions answered with generic advice or without `[[...]]` wikilinks.

For each uncaptured substantive Q&A from this session, **invoke the write-memory skill** with
this handoff — kind=qa, slug, verbatim question, answer body (copy; no re-derive), contributor
basenames, certainty, source ("ad-hoc capture, learn session <date>").

Contributors come ONLY from `[[full-basename]]` wikilinks in the written answer — never
free-listed. Do NOT pre-validate whether contributors exist in the vault; extract the wikilink
content verbatim and include the basenames in the write-memory handoff. Validation happens at
write time; if write-memory reports a contributor rejection, surface it.
If no `[[...]]` wikilinks appear in the answer and no note was crystallized, skip (D2 bar not met).

**Gate — do not duplicate:** if a QA pair was already written (e.g. by recall's Step 4 during
this session), do not write it again here. One pair per distinct answered question.

## Red flags — STOP and re-read

| Sign you're off-script | What you should be doing |
| --- | --- |
| You ran `engram transcript` anything | Step 1's `engram ingest --auto` replaced the whole transcript workflow |
| You're writing an episode or summarizing the session into a note | Don't — raw chunks already hold it |
| You're writing facts for things nobody asked you to remember | Only corrections, save-requests, reversals, and confirmed approaches crystallize here |
| You're writing a confirmed-approach note for a routine success or a bare "thanks!" | Kind 4 needs a genuine bet with observable confirmation, or explicit praise of a SPECIFIC behavior — never mere success |
| You skipped the sweep because "nothing changed" | The sweep IS the check — it costs seconds when nothing changed — skipping because a sweep already ran this session is the prescribed exception |
| `--tier` flags or L3/ADR writing | Tiers are not part of learn anymore |
| You corrected a repo doc (CORRECTION/postscript) and skipped the vault note | Write the vault note for the reversal's root cause — record-correction is not capture |
