---
name: recall
description: >
  Use after any user request that might entail more than a single tool call or anything more than quick, shallow
  thinking. This surfaces relevant memories that are VITAL to recall for a good user experience and a greater chance at
  first-pass success for the user's request.
---

# Recall from Unified Memory

Surface relevant memories — raw conversation/doc chunks AND crystallized vault lessons in one ranking — lay your planned actions against them, and report, out loud, whether memory changed the plan.

## Overview

Memory has two layers retrieved in ONE call: raw chunks (every past conversation and doc, embedded mechanically) and vault notes (lessons crystallized from them). Recall's jobs, in order:

1. **Make your plan visible** before retrieving anything — an unstated plan cannot be tested against memory.
2. **Sweep, then run ONE unified `engram query`.** Items tagged `kind: chunk` are raw fragments; `kind: fact`/`feedback` are crystallized lessons. They compete in the same top-N.
3. **Crystallize** — when several near-match chunks evidence the same principle and no note states it yet, write the vault note now.
4. **Synthesize impact on the plan** — confirm / adjust / contradict / silent, per planned action.

The binary resolves the vault and chunk index automatically (`$XDG_DATA_HOME/engram/...`;
`ENGRAM_VAULT_PATH` / `ENGRAM_CHUNKS_DIR` override). **Do not pass `--vault` or `--chunks-dir`.**

## The procedure

### Step 0 — Print your upfront judgement

**Before any engram call**, print three short blocks. Plain prose, no headers needed:

- **Ask:** what is being asked, in your own words. One or two sentences.
- **Situation:** the concrete context — repo, branch, the operation underway, what's loaded into your context. One short paragraph.
- **Plan:** the action(s) you would take next absent any guidance from memory. Numbered list.

Skipping Step 0 is forbidden. The whole purpose of recall is to test a stated plan against memory.

### Step 0.5 — Sweep

```bash
engram ingest --auto
```

Seconds when nothing changed; guarantees the index includes the latest sessions and doc edits.
If it errors, say so and continue — retrieval over a slightly-stale index beats no retrieval.

### Step 1 — Phrase queries from your plan and situation

Write down 5 to 15 short queryable phrases, two kinds:

- **Plan-grounded** — drawn directly from the actions you said you would take.
- **Situational** — features continuously true around the action (tooling, language, kind of operation, what's loaded).

No pre-filtering: you can't know what's in memory before you query. Drop only obvious dross
(a bare noun like "coding"). **Query by task, not by fear** — "implementing Claude Code hooks",
not "common mistakes when writing hooks".

### Step 2 — Run ONE unified `engram query` with all phrases

```bash
engram query \
  --phrase "<phrase 1>" \
  --phrase "<phrase 2>"
  # ... one --phrase per Step 1 phrase
```

One call; the binary merges ranking server-side. Do NOT collapse phrases, do NOT run per-phrase
calls, do NOT add `--tier` or `--synthesize-l2` (`--synthesize-l2` bypasses the chunk space).
The payload's `items` mix:

- `kind: chunk` — raw transcript/doc fragments with source + anchor. These are EVIDENCE:
  extract the convention, decision, or correction they show (a reviewer correcting code, a
  stated standard); never quote them wholesale.
- `kind: fact` / `feedback` — crystallized lessons; apply directly.

If nothing surfaces, say so in one sentence, skip Step 2.5, and proceed with your plan.

### Step 2.5 — Crystallize lessons from the payload's chunk clusters (band-driven)

The payload's `clusters` list includes entries with `phrase: "chunks"` — the binary's
deterministic grouping of the returned chunks (auto-k k-means, silhouette-validated). Each
carries `nearest_l2: {path, cosine}` — the closest existing vault lesson to that cluster.
**Process every chunk cluster; the bands decide, not your judgment:**

| `nearest_l2.cosine` | Action |
| --- | --- |
| `>= 0.95` | Do nothing — an existing note already covers it |
| `0.80 – 0.95` | UPDATE the nearest note: `engram learn fact\|feedback --target <luhmann-id from nearest_l2.path> --position continuation ...` |
| `< 0.80`, or no `nearest_l2` | CREATE a new note (`--position top`) |

Before an UPDATE/CREATE write, read the cluster's member chunks (already in `items`) and state
the principle they evidence:

```bash
# Durable convention/standard:
engram learn fact --slug <kebab-slug> --position top \
  --source "synthesized from chunk cluster at recall, <YYYY-MM-DD>" \
  --situation "<when this applies>" \
  --subject "<the thing>" --predicate "<requires / must use / is>" \
  --object "<the standard, stated generally enough to transfer>"

# Correction about how to work: engram learn feedback with
# --behavior/--impact/--action instead of subject/predicate/object.
```

Rules: general principle, not the instance; one write per cluster; a cluster whose members are
a vocabulary coincidence rather than a shared principle gets NO write (the bands gate novelty,
you still gate meaning — that is the one judgment left to you). WAIT for writes to finish —
the lessons apply to THIS task too.

### Step 3 — Closing synthesis: did the memories change the plan?

The user sees this. Rules:

- **Open with the count.** One sentence: "Query surfaced N items (K chunks, M notes); crystallized J lessons."
- **Walk the plan from Step 0 in order.** Per numbered action: **confirmed**, **adjusted** (and how), **contradicted**, or **silent**. One short bullet each.
- **Frame load-bearing conventions as requirements.** Lead with "Apply these as requirements:" and list them — drawn from lessons and chunk evidence alike. A plan step memory confirms still inherits the convention's concrete specifics as requirements.
- **Surface contradictions inline** where they affect the action, as prose.
- **No payload dumps** — never re-emit YAML, paste whole chunks, or list raw scores/paths.
- **Length:** as long as honesty about the plan requires; if memory was silent on everything, one sentence.

## Red flags — STOP and re-read

| Sign you're off-script | What you should be doing |
| --- | --- |
| You never printed Step 0 | Back up — the skill is a no-op without it |
| You skipped the Step 0.5 sweep | It costs seconds and keeps memory current |
| `--tier`, `--synthesize-l2`, `--vault`, or `--chunks-dir` on the query | Plain unified `engram query --phrase ...` only |
| Separate query calls per phrase | One call, repeatable `--phrase` flags |
| You quoted chunks wholesale into the reply | Extract the principle a chunk evidences; paraphrase |
| You dispatched cluster-synthesis subagents | Gone — Step 2.5 crystallizes inline from the payload's chunk clusters |
| You grouped chunks by eye instead of using the payload's `phrase: "chunks"` clusters | The binary's k-means grouping and `nearest_l2` cosine are the ground truth; apply the bands |
| You skipped Step 2.5 because "the chunks are enough" | Banding every chunk cluster IS the step; skipping it is not an outcome |
| Reply is a memory dump with no plan reference | Restart Step 3: walk the plan and judge each piece |
