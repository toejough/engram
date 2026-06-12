---
name: recall
description: Use after any user request that might entail more than a single tool call or anything more than quick, shallow thinking. This surfaces relevant memories that are VITAL to recall for a good user experience and a greater chance at first-pass success for the user's request.
---

# Recall from the Auto-Chunked Memory Index

Surface relevant chunks from the auto-ingested memory index, lay your planned actions against them, and report — out loud — whether the surfaced memories changed that plan.

> **EVAL-LOCAL VARIANT.** This recall reads the chunk index (`engram query-chunks`), not the
> zettelkasten vault. The memory is raw transcript/markdown chunks embedded by the binary — there
> are no notes, no wikilinks, no tiers. Your job is unchanged: state your plan, query, and
> synthesize how memory changes the plan.

## The procedure

### Step 0 — Print your upfront judgement

**Before any query call**, print three short blocks to the user. Plain prose, no headers needed:

- **Ask:** what is being asked, in your own words. One or two sentences.
- **Situation:** the concrete context — directory, the operation underway, what's loaded into your context. One short paragraph.
- **Plan:** the action(s) you would take next absent any guidance from memory. Numbered list or short paragraph.

Skipping Step 0 is forbidden. The whole purpose of recall is to test a stated plan against memory; an unstated plan cannot be tested.

### Step 1 — Phrase queries from your plan and situation

Write down 5 to 10 short queryable phrases mixing two kinds:

- **Plan-grounded** — phrases drawn directly from the actions you said you would take.
- **Situational** — features continuously true around the action (tooling, language, the kind of operation).

Query by task, not by fear: "building a CLI notes app in Go", not "common mistakes in Go".

### Step 2 — Run ONE `engram query-chunks` call with all phrases

```bash
engram query-chunks --chunks-dir "$ENGRAM_CHUNKS_DIR" \
  --phrase "<phrase 1>" \
  --phrase "<phrase 2>" \
  --phrase "<phrase 3>"
  # ... one --phrase per Step 1 phrase
```

`$ENGRAM_CHUNKS_DIR` is set in your environment. Do NOT run separate calls per phrase — one call merges the ranking server-side. The payload returns `items` (top chunks with source/anchor/score/content) and optionally `clusters` (vector neighborhoods).

If the payload reports `total_chunks: 0`, memory is empty — say so in one sentence and proceed with your plan.

### Step 3 — Closing synthesis: did the memories change the plan?

Read every returned chunk's content. Then, user-facing:

- **Open with the count.** One sentence: "Query surfaced N chunks (of T total)."
- **Walk the plan from Step 0 in order.** For each numbered action, say plainly whether the chunks **confirmed**, **adjusted** (and how), **contradicted**, or were **silent**. One short bullet per action.
- **Frame load-bearing conventions as requirements.** When a surfaced chunk shows a convention, decision, or piece of feedback relevant to what you're about to build, present it as a **requirement to implement** — lead with "Apply these as requirements:" and list them. Chunks are raw conversation/doc fragments: extract the convention or decision they evidence (e.g. a reviewer correcting code, a stated standard), don't quote them wholesale.
- **No payload dumps.** Never re-emit the YAML or paste whole chunks into the reply; paraphrase what each load-bearing chunk teaches.
- **Length:** as long as it needs to honestly cover the plan; no filler. If memory was silent on every action, one sentence is fine.

## Red flags — STOP and fix

| Sign you're off-script | What you should be doing |
| --- | --- |
| You never printed Step 0 | Back up. The skill is a no-op without it |
| You ran `engram query` (vault) or hand-grepped files | This variant reads the CHUNK index via `engram query-chunks` |
| Separate query calls per phrase | One call, repeatable `--phrase` flags |
| Reply is a chunk dump with no plan reference | Restart Step 3: walk the plan and judge each piece |
