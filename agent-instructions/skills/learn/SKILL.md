---
name: learn
description: >
  Use after work begun with recall, or any work requiring more than one tool call or non-trivial thought. Use immediately
  for explicit save requests ("remember this", "save that", "note for next time"), and the moment a review, failing check,
  or user correction confirms a lesson; capture it before applying the fix. Also use at session end when a presented
  conclusion, design, or finding was later overturned or corrected. Use when a specific approach is confirmed: the user
  praises a reusable behavior, or an uncertain choice succeeds with observable evidence. This preserves lessons for
  future recall.
---

# Learn — Sweep Raw Memory, Crystallize Explicit Lessons

Two jobs, in order: (1) mechanically true up the chunk index so every conversation and doc is searchable memory, and (2) write a vault note for every explicit lesson this session — user corrections, explicit save-requests, presented conclusions that were later overturned, and confirmed approaches (user-praised or self-validated). Nothing else.

> **Raw event memory is AUTOMATIC.** `engram ingest` chunks and embeds session transcripts and
> markdown itself — no summaries, no episode notes, no arc detection. The agent's only writing
> job is crystallizing EXPLICIT lessons. Do not reconstruct the old episode workflow
> (`engram transcript`, `engram learn episode`) — it is gone.

> **Mid-cycle capture (fast path).** When you fire at a CORRECTION moment mid-task — a review, a
> failing check, or the user just rejected your approach or named a different convention — this is a
> focused single-note capture, NOT the open/close of a cycle. **SKIP Step 1 (sweep) and Step 1.5
> (vocab); go straight to Step 2** and crystallize the ONE confirmed correction (hand off to
> write-memory as always). Do NOT run `engram ingest --auto` mid-task — the sweep is a
> cycle-boundary job, it is wasteful here and can block on a large corpus, and losing the capture to
> a hung sweep is the failure mode. The closing learn still sweeps.

## Step 1 — Sweep (first; the closing learn always sweeps)

```bash
engram ingest --auto
```

That's it. The binary stats every known source (repo markdown, `.claude` dirs, all session
transcripts), re-chunks and re-embeds only what changed — within one source, existing chunks are
never deleted (append-only history). Across sources, byte-identical content is deduplicated: only
one canonical copy is indexed, and a duplicate's index is removed only once its retained twin is
verified to cover its records (not on hash-match alone) — see `engram prune --duplicates` for the
retroactive cleanup mode. Unchanged corpus → returns in seconds. Report the one-line tally it
prints (or "memory index up to date").

If the command fails, surface the error and continue to Step 2 — explicit lessons must not be
lost to an ingest hiccup.

Skip the sweep if one already ran earlier in THIS session, UNLESS this is the closing learn of a
work cycle — the closing learn ALWAYS sweeps (it captures the session's tail for future sessions).
This holds even under a "something might have changed outside this session (another terminal,
another agent)" worry — that gets picked up by the next sweep (yours or theirs), not by
re-sweeping now on suspicion.

## Step 1.5 — Vocab liveness check

Run `engram vocab stats`.

If the verdict is `verdict: OK`, continue to Step 2 with no further vocab action.

If the output includes a line matching `verdict: REFIT_PENDING (<reason>)`, run the refit
autonomously — do not defer to the user:

1. Run `engram vocab refit`. The binary derives the clusters itself — there is no plan to
   author. (`--dry-run` prints the derivation diff without writing; use it on the first refit
   over a long-drifted vault if you want to sanity-check a mass retirement first.)
2. If it prints `vocab refit: no structure found; vocabulary unchanged`, nothing was written
   and there is no version bump — report exactly that ("Vocab refit ran: no structure found,
   vocabulary unchanged.") and continue to Step 2. Do NOT claim an applied refit.
   If every cluster matched an existing term, the run applies directly — skip to step 4.
   If there are new clusters, the command prints a JSON payload
   (`naming_requests` + `fingerprint`) and exits WITHOUT writing. Name each new cluster from
   its exemplars, then write an answer file:

   ```json
   {"names": [{"cluster": 0, "term": "kebab-case-term", "description": "one line"}],
    "fingerprint": "<echoed verbatim from the request payload>"}
   ```

   Cover every new cluster exactly once. Re-run: `engram vocab refit --names /path/answer.json`.
3. If the re-run fails with a stale-names error (the vault changed between runs), the requests
   are void — re-run `engram vocab refit` from scratch to regenerate them, and answer those.
4. **Report loudly:** "Vocab refit applied: <version bump>. Triggered by: <reason>."

Also check the QA round-2 gate line. If the output includes `qa round-2 gate: READY (...)`, report to Joe:
"QA round-2 validation is due (≥20 pairs captured). Please schedule the round-2 gates recorded in
docs/ROADMAP.md (The roadmap → GATED → Q&A memory round-2): P2′ attribution fidelity, P3′ distribution, Arm V larger-n." Do NOT run round-2 validation
autonomously — it requires Joe's oversight.

## Step 2 — Crystallize explicit lessons (only when they exist)

Scan THIS session for exactly four kinds of moments. For each note you crystallize, first decide
its Luhmann placement (below), then include the resulting `position`/`target` in the write-memory
handoff alongside the kind-specific fields.

**Disposition (placement) — decide before handing off:**

Grounded in Niklas Luhmann's own Zettelkasten practice and Sönke Ahrens, *How to Take Smart
Notes*: an ID's position encodes where a note sits in the thought tree, relative to notes written
or recalled earlier in THIS session (not a full-vault search). Apply this test in order:

1. Does the new note develop **one specific sub-point** raised inside an in-session note? →
   `position=continuation`, `target=<that note's ID>`.
2. Else, does it continue/extend the **same overall thought** as an in-session note, at the same
   level (not a sub-point of it)? → `position=sibling`, `target=<that note's ID>`.
3. Else → `position=top` (no target).

No in-session candidate note exists → always `position=top`; do not search the rest of the vault
for a placement target.

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
     ("thanks!", "great work") does NOT qualify.
   - **4b — self-validated bet:** you made a genuine guess, or were uncertain about a plan, concept,
     or idea, ACTED on it (embodied it in the work — a chosen approach, an implementation, a plan
     step, a command run), and an OBSERVABLE, session-recorded outcome then confirmed it worked (a
     test passed, the user confirmed it, an artifact functioned, a blocker cleared). The exact
     positive mirror of a reversal: a bet that succeeded instead of failing.

   The action is capturing what WORKED — NOT logging every success. A bare pleasantry, a routine
   success with no bet behind it, or an unconfirmed guess is never the signal — just as a repo-doc
   CORRECTION does not count as reversal-capture, a routine success does not count as
   reinforcement-capture; the signal is a resolved uncertainty or an explicit specific
   confirmation, never "it worked".

   **Runbook vs. feedback — pick by shape, not by trigger:** a confirmed approach that is a
   reusable, ordered, multi-step procedure for a recurring task (e.g. "run tests → tag the release
   → push the tag → update the changelog" for releasing a Go module; running an eval harness; a
   recall-moment discovery sequence) → **kind=runbook**. A confirmed approach that is a single
   behavioral tweak with no natural step structure (e.g. "attach deliverable files instead of
   inlining them") → **kind=feedback**, as before.

   For a **runbook**, **REQUIRED SUB-SKILL:** invoke the **write-memory** skill with this
   handoff — kind=runbook, slug, source ("session <date>, context: <one-line what-was-happening>"),
   situation (retrieval-shaped: when would this approach apply again), done_when = what should be
   true once the procedure is complete, body = the numbered steps (may `[[wikilink]]` fact/feedback
   notes worth reading along the way); plus supersedes details if this confirmation corrects an
   existing vault note. The runbook note itself already is the "remember to do it this way"
   capture — if the user also instructs "do that going forward," do NOT additionally fire a
   kind=fact save-request (kind 2) for the same content; that redundancy-avoidance is specific to
   the runbook case.

   For **feedback**, **REQUIRED SUB-SKILL:** invoke the **write-memory** skill with this handoff —
   kind=feedback, slug, source ("session <date>, context: <one-line what-was-happening>"), situation
   (retrieval-shaped: when would this approach apply again), behavior = what worked, impact = the
   confirming evidence (the user's quote for 4a, or the observed outcome that resolved the
   uncertainty for 4b), action = keep doing it + its trigger conditions; plus supersedes details if
   this confirmation corrects an existing vault note. (If the user also instructs "do that going
   forward", that is additionally a save-request — kind 2.)

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

## Batch mode — Luhmann re-eval answers (`--reparent-luhmann`)

**Trigger:** you were handed a derive-phase candidate JSON payload from
`engram update --reparent-luhmann` (fields: `candidates` — each with `note`, `note_excerpt`,
`target`, `target_excerpt`, `similarity` — plus `instruction` and `fingerprint`), and asked to
produce its answers file. This is a deliberate, explicitly-invoked mode — NOT part of the normal
Step 1→2.5 sequence a per-capture `learn` run walks; it runs standalone, once, over a batch.

For EACH candidate pair, apply the **same disposition test as Step 2's placement decision above**
— read `note_excerpt` and `target_excerpt` and judge the content relationship, not the
`similarity` score alone (a high score can still be an unrelated false positive: two notes can
share vocabulary while addressing different problems):

1. `note_excerpt` develops one specific sub-point of `target_excerpt`? → `position=continuation`.
2. Else, same overall thought as `target_excerpt`, at the same level? → `position=sibling`.
3. Else (unrelated despite the similarity score) → `position=top`.

In all three cases, `target` is only meaningful for `continuation`/`sibling`; a `top` verdict
still needs a full answer entry.

**Coverage requirement:** every candidate's `note` MUST get exactly one entry in the answers file
— there is no "skip this one" option. If a note appears as `note` in more than one candidate
(multiple targets above the floor), judge each candidate independently and answer with the ONE
`target`/`position` you find most correct; still exactly one output entry per distinct `note`.

Write the answers file as JSON, matching this shape exactly — do not rename fields or add others:

```json
{
  "reparenting": [
    {"note": "<id>", "position": "continuation|sibling|top", "target": "<id>"}
  ],
  "fingerprint": "<echoed VERBATIM from the input payload's fingerprint field>"
}
```

Save it to a file (e.g. `/tmp/reparent-answers.json`) and tell the user to run:

```
engram update --reparent-luhmann --answers <file> --dry-run
```

to review the resulting renames, then re-run the same command without `--dry-run` to apply. This
skill does not invoke either command itself — that is the acting agent's (or user's) next step.

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
