---
name: recall
description: >
  Use after any user request that might entail more than a single tool call or anything more than quick, shallow
  thinking. This surfaces relevant memories that are VITAL to recall for a good user experience and a greater chance at
  first-pass success for the user's request.
---

# Recall from the Agent-Memory Vault

Surface relevant notes from the agent-memory vault, lay your planned actions against them, and report — out loud — whether the surfaced memories changed that plan.

## Overview

The vault is a Zettelkasten — past-you's notes for future-you. Recall has three jobs, in this order:

1. **Make your plan visible** before retrieving anything. If the plan stays in your head, neither you nor the user can tell whether memory changed it.
2. **Cascade through the wikilink graph** from anchors + recent + explicit-query frontier, following links of any note that scored relevant, until the budget is spent or the frontier empties.
3. **Synthesize impact on the plan** — for each load-bearing surfaced note, say whether it confirms, adjusts, or contradicts what you were about to do.

Most of the skill's value is in (1) and (3). The cascade in (2) is mechanical; without (1) and (3) it is just a note dump.

## Vault structure

The binary resolves the vault automatically — `--vault` and
`ENGRAM_VAULT_PATH` are overrides, not requirements. Default:
`$XDG_DATA_HOME/engram/vault` (typically `~/.local/share/engram/vault`).
**Do not pass `--vault` in `engram recall` invocations unless the user
explicitly tells you the vault is elsewhere.**

```
<vault>/
  Permanent/   atomic principle-stated notes; <Luhmann-ID>.YYYY-MM-DD.<slug>.md
  MOCs/        Maps of Content with framing prose and in-prose wikilinks; same filename format
```

Notes are LLM-voiced. Wikilinks appear in prose with surrounding context — that context is the relevance signal. Luhmann IDs (`1`, `1a`, `1a1`) signal lineage; following wikilinks reaches Luhmann-adjacent notes by construction.

## Modes

| Mode                                  | Trigger                                                                                                              | Explicit query                                                                                |
| ------------------------------------- | -------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------- |
| **Default — after request received**  | Any user request that might entail more than a single tool call or more than shallow thinking. Self-fire.            | Seed phrased from the request + your pre-recall plan; combined with situational baseline.     |
| **No-arg recap**                      | `/recall` with no topic, or self-invoke for orientation.                                                              | Anchors (`engram recall`) seeded with `engram recall --recent`.                               |
| **Topic query**                       | A topic is named, or you formed one from context.                                                                    | The topic, phrased as given.                                                                  |

**Skip only** for pure single-call lookups, trivial edits (typo, rename, one-liner), or follow-on turns where recall already ran for the same task. If a topic is ambiguous ("recall that thing"), make a best-effort phrasing rather than asking — if results are off-target, the user can refine.

## The retrieval pipeline

### Step 0 — Print your upfront judgement

**Before any `engram recall` call**, print three short blocks to the user. Plain prose, no headers needed:

- **Ask:** what the user is asking, in your own words. One or two sentences.
- **Situation:** the concrete context — repo, branch, the operation underway, what's loaded into your context. One short paragraph.
- **Plan:** the action(s) you would take next absent any guidance from memory. Numbered list or short paragraph.

Skipping Step 0 is forbidden. The whole purpose of recall is to test a stated plan against memory; an unstated plan cannot be tested. If you find yourself starting a `engram recall` tool call before this print appears in your user-facing output, stop and back up.

### Step 1 — Phrase queries from your plan and situation

The plan you just printed is the primary query seed. Re-read it and write down — internally — 5 to 15 short queryable phrases. Mix two kinds:

- **Plan-grounded** — phrases drawn directly from the actions you said you would take ("wire OpenCode reader alongside Claude Code", "advance per-harness marker").
- **Situational** — features continuously true around the action (tooling, language, the kind of operation, the user's role, what's loaded into context).

**Apply this test to each candidate:**

> If a future-me on a fresh context were dropped into roughly this same situation, and there were one memory about _this phrase alone_, would it be worth surfacing?

If yes, keep it. If you can't imagine what that memory would even be about, the phrase is either too generic ("coding") or too specific to this exact moment ("a bug at line 47").

This list is internal scratch — not part of the user-facing output.

### Step 2 — Form the explicit query

For no-arg: combine anchors and recent activity (see Step 3 cascade).

For topic / self-invoked: phrase as the user gave it, or as you'd phrase a search.

**Query by task, not by fear.** What are you trying to do? Not what might go wrong. "implementing Claude Code hooks" — not "common mistakes when writing hooks." Memories are written to match task descriptions, so query the same way.

### Step 3 — Cascade with visible progress

The skill drives the cascade as a loop calling `engram recall`. The binary is a thin graph primitive; relevance evaluation lives here.

**Initial frontier:**

```bash
engram recall
engram recall --recent --limit 20
```

Union the outputs. Each line is a vault-relative path like `Permanent/<basename>.md` or `MOCs/<basename>.md` — pass it directly to your file-read tool; no path-guessing.

**Loop (until ≥100 surfaced memories OR frontier empties):**

1. **Read every file in the current frontier** and score relevance against (a) the explicit query and (b) every phrase from Step 1. Use parallel subagents when the frontier is large (≥10) to keep raw note content out of parent context; small frontiers can be read inline.
2. **Track read files** in a cumulative `--already-read` set covering every file read so far, whether surfaced or not.
3. **Print a one-line progress update** before expanding:

   ```
   round <N>: <relevant> relevant / <irrelevant> irrelevant / <budget-left> budget left / <follow-count> links to follow
   ```

   This line is part of the user-facing output. It makes "I have enough" stop being a private decision.

4. **Expand the frontier** for every note that scored relevant:

   ```bash
   engram recall \
     --follow Permanent/A.md,MOCs/B.md \
     --already-read Permanent/X.md,Permanent/Y.md,...
   ```

   `--follow` = all relevant notes from this round. `--already-read` = the cumulative set. **Inputs are the full relative paths recall emitted** (`<Subdir>/<basename>.md`). Bare basenames are rejected with an error — pass paths back exactly as recall printed them.

5. **Repeat** from step 1 with the new frontier.

**Termination — and only these:**

- ≥100 surfaced notes → stop.
- Empty frontier → stop.

"I have enough" / "this is plenty" / "the recent ones cover it" are **not** valid stopping conditions. If the frontier is non-empty and the budget isn't spent, keep going.

**Contradictions.** If two surfaced notes make incompatible claims about the same thing, mark them. The vault preserves contradictions; recall surfaces both, never picks a side.

### Step 4 — Closing synthesis: did the memories change the plan?

The cascade has produced a set of surfaced notes. Now metabolize them against the plan you printed in Step 0. Two outputs follow.

#### 4a. Structured form → agent context, NOT the user's screen

The full sectioned block — vault state, query matches, situational matches, contradictions, with wikilinks and Context excerpts — is for the parent LLM's working context only. It is already present as tool-call results from the cascade. **Do not re-emit it as your user-facing reply.**

If the structured block hasn't been materialized anywhere (e.g., the cascade went direct-read and no subagent assembled it), have your _last_ cascade subagent assemble it and return it — don't compose it in the parent reply, because composing it there leaks it to the user.

**Structured-form template** (internal — for 4a, not user-visible):

```
## Recall — <mode>

### Vault state
(omit unless something structurally surprising — layout drift, unusual sparsity. Brief.)

### From your query: <explicit query phrasing>
- [[<wikilink>]] — <one-line claim or principle>
  Context: <1–2 sentence excerpt of in-prose framing>
- ...

### From your situation
- [[<wikilink>]] — <one-line claim>
  Matches: <plan-grounded or situational phrase(s) this applies to>
  Context: <1–2 sentence excerpt>
- ...

### Contradictions in the vault
(omit section if none)
- [[<note A>]] vs [[<note B>]] on <topic>
  <one-line summary of the disagreement>
```

Empty section in the structured form (4a) — write `(no matches)` rather than omitting. Exception: if a section is empty _because_ its matches were consolidated under another section, write `(matches consolidated above)`. These placeholders live in 4a only.

**Dedup rule.** If a note matches both the explicit query and a Step 1 phrase, surface it once under the more specifically relevant section and add a `Also matches: <phrase>` line. Don't duplicate.

#### 4b. User-facing synthesis — how the memories did or did not change the plan

The user sees this. Rules:

- **Open with the count.** One sentence: "Cascade surfaced N notes across M rounds." That is it for recap framing.
- **Walk the plan from Step 0 in order.** For each numbered action or distinct piece of the plan, say plainly whether the memories **confirmed**, **adjusted** (and how), **contradicted**, or were **silent**. One short bullet per action.
- **Surface load-bearing contradictions inline as prose**, not as a separate section. If a contradiction matters for the next action, call it out where that action is being discussed.
- **No wikilinks.** Name notes by what they say, not by filename.
- **No `(no matches)` / `(matches consolidated above)`** placeholders — those belong in 4a.
- **Length:** as long as it needs to honestly cover the plan; no filler. If memories were silent on every action, one sentence is fine.
- **If nothing surfaced at all**, say so in one sentence and stop.

#### Red flags — STOP, you are leaking the structured form

If you catch yourself doing any of these in the user-facing reply, rewrite:

| Sign you're leaking                                                  | What you should be doing                                                              |
| -------------------------------------------------------------------- | ------------------------------------------------------------------------------------- |
| Writing `[[…]]` wikilinks in the reply                               | Paraphrase the claim; no wikilinks in 4b                                              |
| Writing a `### Contradictions` section                               | Contradictions stay inline where they affect the plan; no separate section            |
| Writing `### From your query` / `### From your situation` headers    | No structured-form headers in 4b — walk the plan instead                              |
| Writing `Context:` excerpts under each bullet                        | Excerpts belong to 4a only                                                            |
| Recap is a generic "highlights:" bullet list with no plan reference  | You skipped synthesis. Restart Step 4b: walk the plan from Step 0 and judge each piece |
| You never printed Step 0                                             | Back up. The whole skill is a no-op without it                                        |
| You stopped the cascade because "I have enough"                      | Resume until 100 surfaced or frontier empty                                           |
| `(no matches)` or `(matches consolidated above)` appears in 4b       | Those are 4a placeholders; in 4b just say "memories were silent on this action"       |

## Failure modes

| Situation                                            | Behavior                                                                                                                             |
| ---------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| Default vault directory does not exist               | `engram recall` reports "vault not found"; tell the user to run `engram learn` (which bootstraps it) or set `ENGRAM_VAULT_PATH` to an existing vault. Do not create the directory yourself. |
| Vault exists but is empty                            | Report "vault is empty; no recall produced." Do not fabricate.                                                                       |
| `engram recall` command not found                    | Fall back: read every `.md` under `MOCs/` and `Permanent/` directly, scoring as in Step 3. Note the missing binary in _Vault state_. |
| No matches for explicit query                        | `(no matches)` for that section in 4a. Situational baseline may still produce.                                                       |
| No matches anywhere                                  | State plainly in 4b. Normal early in a vault's life.                                                                                 |
| A note read fails                                    | Log which note, continue with the rest. One bad note ≠ abort.                                                                        |

## What this skill is not for

- Reading session transcripts. Use `engram transcript` if you need past-session activity.
- Writing to the vault. Capture is the `learn` skill.
- Inventing memories. If recall would surface nothing, surface nothing.
- Inventing classifications (confidence tiers, freshness scores, priority ranks) the upstream skills don't produce.
- Deduplicating against your prior context. The parent agent handles that.

## Discovery and trigger ceiling

This skill fires when the model recognizes the situation as recall-relevant from the description. Some genuinely relevant moments will be missed because the model didn't realize recall applied. That ceiling is accepted; proactive triggering (hooks) is a separate concern and out of scope for the skill itself.
