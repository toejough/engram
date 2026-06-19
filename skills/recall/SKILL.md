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
engram query --synthesize-l2 \
  --phrase "<phrase 1>" \
  --phrase "<phrase 2>"
  # ... one --phrase per Step 1 phrase
```

One call; the binary merges ranking server-side. `--synthesize-l2` is REQUIRED: it runs the
unified D1 clustering of the matched chunks **and** notes in one pass and emits
`candidate_l2s: [{path, cosine}]` per cluster (it no longer bypasses the chunk space — matched
chunks are clustered alongside notes, and Step 2.5 reasons over those unified clusters). Do NOT
collapse phrases, do NOT run per-phrase calls, do NOT add `--tier`, `--vault`, or `--chunks-dir`.
The payload's `items` mix:

- `kind: chunk` — raw transcript/doc fragments with source + anchor. These are EVIDENCE:
  extract the convention, decision, or correction they show (a reviewer correcting code, a
  stated standard); never quote them wholesale.
  - **Recent items are your own recent activity.** Chunk items with `turn-N` anchors from a
    recent source are first-person `ASSISTANT:` narration you produced in a just-prior or
    pre-context-clear session. Treat them as your own past actions — do not re-derive them,
    do not express surprise at them, and dedup against what is already in your context.
- `kind: fact` / `feedback` — crystallized lessons; apply directly.
  - Items of this kind may carry `activated: true` — the binary flagged them as surfaced AND
    above the relevance cutoff.

If nothing surfaces, say so in one sentence, skip Step 2.5, and proceed with your plan.

**After the query**, collect every item where `activated: true` and issue ONE batched call:

```bash
engram activate \
  --note "<path of first activated note>" \
  --note "<path of second activated note>"
  # ... one --note per activated item
```

Forward every flagged path to `engram activate` to refresh that L2's recency (`LastUsed`) so it
stays warm. Skip this call when no items carry `activated: true`.

### Step 2.5 — Lazy L2 synthesis from the clustering (agent-judged)

The `--synthesize-l2` output's `clusters` list contains the unified clustering of matched chunks
and notes. Each cluster carries `candidate_l2s: [{path, cosine}]` — the top-K existing L2s
nearest the cluster centroid (K ≥ 3, centroid cosine). **Process every cluster.** For each:

**A. Read candidates and members**

Run `engram show <path>` on every entry in `candidate_l2s` (up to K calls, blocking). For
note-kind cluster members already in the payload's `items[]` list, use their `content` field
directly — no additional `engram show` call needed on already-surfaced members. For chunk
members not in `items[]`, use the chunk content from the cluster's `members` list. Do not
judge coverage before you have read the candidate content.

**B. Apply the recency weight to resolve conflicts**

Evidence **conflicts** when a newer member explicitly negates or reverses an older claim. Reversal
cues: "no longer", "replaced by", "use X not Y", or the same subject+predicate appearing with a
different object in a newer item. When conflict is present: **recent wins**. When no conflict:
treat older and newer evidence as independently valid — do not demote a stable convention merely
because it lacks a recent instance.

**C. Judge coverage against the recency-weighted view — in this order**

| Outcome | Criterion | Action |
| --- | --- | --- |
| **Covered** | A candidate's claim states the cluster's principle with **no material omission** vs the recency-weighted members | `engram amend <candidate-path> --activate --relation <new-note-sources> --chunk-source <new-chunk-ids>` — link-enrich only; **do not rewrite content** |
| **Near** | A candidate addresses the same situation but omits ≥ 1 substantive claim the members evidence (judge against the recency-weighted view — a candidate that only matches the superseded content is **near**, not covered) | `engram amend <candidate-path> --relation <note-sources> --chunk-source <chunk-ids> --subject ... --predicate ... --object ...` (or `--behavior/--impact/--action`) — re-synthesize content from all members, recency-weighted |
| **Absent** | No candidate addresses the situation | `engram learn fact\|feedback --position top --relation <note-sources> --chunk-source <chunk-ids> --source "<descriptive>" --situation "..." --subject/--predicate/--object (or --behavior/--impact/--action)` |

**One write per cluster; one representative L2 per cluster.** The representative is always an L2
(never an L1 note or a chunk). For `absent`, write exactly one note (fact *or* feedback) covering
the cluster's principle. Do not write one fact and one feedback note for the same cluster.

For `amend` (covered or near), pass `--relation` for every **note** source in the cluster (the
wikilink graph) and `--chunk-source` for every **chunk** source (provenance, not wikilinks). For
`learn`, pass the same flags. The `--source` flag on `learn` is the human-readable provenance
string (unchanged); `--chunk-source` is the chunk-id list (new).

**WAIT for each write before moving to the next cluster.** Writes are blocking and inline — the
L2 created or updated by one cluster may be a candidate for another.

**Known gap:** cross-cluster supersession — where the superseding evidence did not cosine-cluster
with the old — is not handled. Note the conflict in the synthesized content when you see it, but
do not attempt to resolve it across clusters.

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
| `--tier`, `--vault`, or `--chunks-dir` on the query | `engram query --synthesize-l2 --phrase ...` only — `--synthesize-l2` IS required (it runs the unified D1 clustering and emits `candidate_l2s`) |
| Separate query calls per phrase | One call, repeatable `--phrase` flags |
| You quoted chunks wholesale into the reply | Extract the principle a chunk evidences; paraphrase |
| You dispatched cluster-synthesis subagents | Gone — Step 2.5 crystallizes inline from the payload's clusters |
| You judged coverage before reading the candidate content with `engram show` | Read first — cosine alone cannot decide coverage |
| You applied a cosine threshold to decide covered/near/absent | Coverage is agent-judged from content; cosine only nominates candidates |
| A candidate matching only the superseded content → you marked it "covered" | Apply the recency weight first; a candidate that misses the conflict is "near" |
| You wrote two notes (a fact AND a feedback) for one cluster | One representative L2 per cluster — pick the right kind |
| You used `nearest_l2` instead of `candidate_l2s` | The v2 field is `candidate_l2s: [{path, cosine}]` — a list, not a singleton |
| You called `engram learn --target` to update a note in place | Updates use `engram amend`; `engram learn` is create-only |
| A `≥0.95` cluster → you activated without reading the candidates | Read first; high cosine nominates, it does not decide |
| You called `engram show` on a note already in `items[]` | Members already in `items[]` carry a `content` field — use it directly; `engram show` is only for candidates not in `items[]` |
| You grouped chunks by eye instead of using the payload's clusters | The binary's k-means grouping is the ground truth; read every cluster |
| You skipped Step 2.5 because "the chunks are enough" | Processing every cluster IS the step; skipping it is not an outcome |
| You read chunk-only results as Step 2's "nothing surfaces" and skipped 2.5 | "Nothing surfaces" means an EMPTY payload; clusters present means Step 2.5 runs |
| You saw `activated: true` items but skipped `engram activate` | One batched call per recall — forward all flagged paths; skipping means useful L2s never get refreshed |
| Reply is a memory dump with no plan reference | Restart Step 3: walk the plan and judge each piece |
