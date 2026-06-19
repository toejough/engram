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

## Step 2 — Crystallize explicit lessons (only when they exist)

Scan THIS session for exactly two kinds of moments:

1. **Corrections** — the user corrected your approach or behavior ("don't suppress lint warnings —
   fix the underlying issue", "never amend pushed commits"). Write feedback:

```bash
engram learn feedback --slug <kebab-slug> --position top \
  --source "session <date>, context: <one-line what-was-happening>" \
  --situation "<retrieval-shaped phrase: when does this apply>" \
  --behavior "<what was done>" --impact "<why it was wrong/costly>" --action "<what to do instead>"
```

2. **Explicit save-requests** — the user said "remember this/that X", "note for next time",
   "write this down". Write a fact:

```bash
engram learn fact --slug <kebab-slug> --position top \
  --source "session <date>, context: <one-line what-was-happening>" \
  --situation "<retrieval-shaped phrase: when does this apply>" \
  --subject "<the thing>" --predicate "<requires / must use / is>" --object "<the standard or value>"
```

Rules:
- **State the general principle**, not the session-specific instance — future-you recalls by
  situation similarity, not by remembering this session.
- The `--situation` phrase is the retrieval handle: phrase it the way a future task would be
  described ("releasing a Go module", "writing eval harness metrics").
- One note per distinct principle. An explicit save-request ALWAYS gets its note, immediately —
  "remember this" means stop and write before anything else.
- **No moments of either kind → write nothing.** Routine work is already captured by Step 1;
  a session with no corrections and no save-requests is a two-command learn (sweep + report).

## What learn does NOT do anymore

| Old behavior | Why it's gone |
| --- | --- |
| `engram transcript --mark` / `--segments`, arc detection | The sweep ingests raw transcripts whole; chunking is mechanical |
| `engram learn episode` (L1 episodes, boundary rationales) | Chunks ARE the raw event memory — summarized episodes are redundant and cost LLM turns |
| ADR / L3 synthesis at learn time | Deferred entirely; crystallization happens at recall, from evidence |
| Eager L2 sweeps ("what facts did this session teach?") | Only EXPLICIT corrections and save-requests crystallize at learn; everything else stays raw until recall surfaces a pattern |

## Red flags — STOP and re-read

| Sign you're off-script | What you should be doing |
| --- | --- |
| You ran `engram transcript` anything | Step 1's `engram ingest --auto` replaced the whole transcript workflow |
| You're writing an episode or summarizing the session into a note | Don't — raw chunks already hold it |
| You're writing facts for things nobody asked you to remember | Only corrections and explicit save-requests crystallize here |
| You skipped the sweep because "nothing changed" | The sweep IS the check — it costs seconds when nothing changed |
| `--tier` flags or L3/ADR writing | Tiers are not part of learn anymore |
