---
name: recall
description: Use after any user request that might entail more than a single tool call or anything more than quick, shallow thinking. This surfaces relevant memories that are VITAL to recall for a good user experience and a greater chance at first-pass success for the user's request.
---

# Recall from Unified Memory (Chunks + Vault) — with L2 Crystallization

Surface relevant memories — raw auto-ingested chunks AND crystallized vault notes — in one query, synthesize durable lessons from groups of near-matching chunks into the vault, and lay your planned actions against everything surfaced.

> **EVAL-LOCAL VARIANT (chunks + vault L2).** Memory has two layers retrieved in ONE call: raw
> transcript chunks embedded by the binary, and vault notes (lessons you crystallize at recall).
> `engram query` merges both into a single ranking — items tagged `kind: chunk` are raw events;
> `kind: fact`/`feedback` are crystallized lessons.

## The procedure

### Step 0 — Print your upfront judgement

**Before any query call**, print three short blocks. Plain prose, no headers needed:

- **Ask:** what is being asked, in your own words. One or two sentences.
- **Situation:** the concrete context — directory, the operation underway. One short paragraph.
- **Plan:** the action(s) you would take next absent any guidance from memory. Numbered list.

Skipping Step 0 is forbidden — an unstated plan cannot be tested against memory.

### Step 1 — Phrase queries from your plan and situation

Write 5 to 10 short queryable phrases: plan-grounded (the actions you'll take) and situational
(tooling, language, kind of operation). Query by task, not by fear.

### Step 2 — Run ONE unified `engram query` call with all phrases

```bash
engram query \
  --phrase "<phrase 1>" \
  --phrase "<phrase 2>"
  # ... one --phrase per Step 1 phrase
```

One call only. The chunk index and vault are both configured via environment — do NOT pass
`--vault` or `--chunks-dir`. The payload's `items` mix `kind: chunk` (raw events) with vault
notes, ranked together by relevance.

If nothing surfaces, say so in one sentence, skip Step 2.5, and proceed with your plan.

### Step 2.5 — Crystallize lessons from near-match chunk groups (L2)

Scan the surfaced items for groups of **3+ chunks that evidence the SAME underlying convention,
decision, or correction** (e.g. a reviewer flagging the same standard across projects). For each
such group:

- **If a surfaced vault note already states the principle** → do nothing (it's crystallized).
- **Otherwise** → write ONE vault note stating the general principle:

```bash
engram learn fact --slug <kebab-slug> --position top \
  --source "synthesized from chunks at recall, <date>" \
  --situation "<retrieval-shaped phrase: when does this apply>" \
  "<the principle, 2-4 sentences, stated generally enough to transfer to the next project>"
```

Use `engram learn feedback` instead when the principle is a correction about how to work (a
reviewer correcting an approach) rather than a fact about the domain. Run `engram learn fact --help`
if flags are unclear.

Rules: state the GENERAL principle, not the app-specific instance. One note per principle. A
vocabulary coincidence is not a lesson — if no group binds a principle, write nothing. Cap at ~5
new notes per recall. WAIT for the writes to finish before continuing — the lessons are for THIS
build too.

### Step 3 — Closing synthesis: did the memories change the plan?

- **Open with the count:** "Query surfaced N items; crystallized K new lessons."
- **Walk the plan from Step 0 in order** — per action: confirmed / adjusted (how) / contradicted / silent.
- **Frame load-bearing conventions as requirements:** "Apply these as requirements:" then the list,
  drawing from lessons and raw chunks alike. Extract the convention a chunk evidences; don't quote wholesale.
- **No payload dumps.** If memory was silent on everything, one sentence is fine.

## Red flags — STOP and fix

| Sign you're off-script | What you should be doing |
| --- | --- |
| You never printed Step 0 | Back up — the skill is a no-op without it |
| You passed `--vault` or `--chunks-dir` | Environment configures both; plain `engram query` |
| Separate query calls per phrase | One call, repeatable `--phrase` flags |
| A lesson restates one chunk | Lessons bind MULTIPLE near-matches; singletons stay raw |
| You skipped Step 2.5 because "the chunks are enough" | Checking for bindable groups IS the step; "none found" is a valid outcome, skipping the check is not |
| Reply is an item dump with no plan reference | Restart Step 3: walk the plan |
