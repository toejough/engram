---
name: learn
description: >
  Use after completing any action we started with a recall call, or after any work that involved more than one tool
  call or more than quick shallow thinking. Also use immediately on explicit save-requests like "remember this",
  "remember that X", "save that for later", "note for next time", "don't forget X", "write this down". This preserves
  relevant memories that are VITAL to recall for a good user experience and a greater chance at first-pass success
  for similar work in the future.
---

# Learn — Sweep Raw Memory, Crystallize Explicit Lessons

Two jobs, in order: (1) mechanically true up the chunk index so every conversation and doc is searchable memory, and (2) write a vault note for anything the user explicitly taught or corrected this session. Nothing else.

> **Raw event memory is AUTOMATIC.** `engram ingest` chunks and embeds session transcripts and
> markdown itself — no summaries, no episode notes, no arc detection. The agent's only writing
> job is crystallizing EXPLICIT lessons. Do not reconstruct the old episode workflow
> (`engram transcript`, `engram learn episode`) — it is gone.

## Step 1 — Sweep (always, first)

```bash
engram ingest --auto
```

That's it. The binary stats every known source (repo markdown, `.claude` dirs, all session
transcripts), re-chunks and re-embeds only what changed — existing chunks are never deleted (append-only history). Unchanged
corpus → returns in seconds. Report the one-line tally it prints (or "memory index up to date").

If the command fails, surface the error and continue to Step 2 — explicit lessons must not be
lost to an ingest hiccup.

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
"QA round-2 validation is due (≥20 pairs captured). Please schedule `docs/design/2026-07-03-qa-memory-proposals.md`
round-2 gates: P2′ attribution fidelity, P3′ distribution, Arm V larger-n." Do NOT run round-2 validation
autonomously — it requires Joe's oversight.

## Step 2 — Crystallize explicit lessons (only when they exist)

Scan THIS session for exactly two kinds of moments:

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

Rules:
- **State the general principle**, not the session-specific instance — future-you recalls by
  situation similarity, not by remembering this session.
- The `--situation` phrase is the retrieval handle: phrase it the way a future task would be
  described ("releasing a Go module", "writing eval harness metrics").
- One note per distinct principle. An explicit save-request ALWAYS gets its note, immediately —
  "remember this" means stop and write before anything else.
- If the new lesson CORRECTS, narrows, or refutes an existing vault note, include the superseded note's basename, type, and claim in the handoff.
- **No moments of either kind → write nothing.** Routine work is already captured by Step 1;
  a session with no corrections and no save-requests is a two-command learn (sweep + report).

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
content verbatim and pass it to `engram learn qa --contributors`. Validation happens at write time.
If `engram learn qa` rejects a contributor, report the rejection and the command you called.
If no `[[...]]` wikilinks appear in the answer and no note was crystallized, skip (D2 bar not met).

**Gate — do not duplicate:** if a QA pair was already written (e.g. by recall's Step 4 during
this session), do not write it again here. One pair per distinct answered question.

## Red flags — STOP and re-read

| Sign you're off-script | What you should be doing |
| --- | --- |
| You ran `engram transcript` anything | Step 1's `engram ingest --auto` replaced the whole transcript workflow |
| You're writing an episode or summarizing the session into a note | Don't — raw chunks already hold it |
| You're writing facts for things nobody asked you to remember | Only corrections and explicit save-requests crystallize here |
| You skipped the sweep because "nothing changed" | The sweep IS the check — it costs seconds when nothing changed |
| `--tier` flags or L3/ADR writing | Tiers are not part of learn anymore |
