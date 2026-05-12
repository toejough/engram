---
name: recall
description: >
  Use when the user invokes recall ("/recall", "recall about X", "what do we
  know about Y"), references prior work ("like we did before", "the auth
  refactor", "didn't we already build"), or before starting new work,
  switching tasks, beginning a feature, changing direction, tackling an issue,
  or any other non-trivial action where prior vault guidance may apply.
  Retrieves relevant notes from the agent-memory vault and structures them for
  the LLM caller.
---

# Recall from the Agent-Memory Vault

Surface relevant notes from the agent-memory vault and inject them into the parent agent's context. Output is structured for an LLM reader; format is human-readable too.

## Overview

The vault is a Zettelkasten — past-you's notes for future-you. Two queries always run in parallel:

1. **Explicit query** — the topic the user named (or recent/anchors for no-arg).
2. **Situational baseline** — features of where you are right now, treated as continuous queries against memory.

Most of this skill's value lives in the situational baseline. Your instinct is to retrieve topically; the situational baseline is how you surface things you didn't know to ask about — tooling gotchas, language conventions, role reminders.

## Vault structure

```
/Users/joe/repos/personal/agent-memory/
  Fleeting/    raw captures, transient
  Permanent/   atomic principle-stated notes; <Luhmann-ID>.YYYY-MM-DD.<slug>.md
  MOCs/        Maps of Content with framing prose and in-prose wikilinks; same filename format
  MEMORY.md    index — names notes; substance is in the notes themselves
```

Permanents are higher-quality than fleetings (promoted, principle-stated, wikilinks established). Fleetings are recent raw signal; missing them means missing the most recent material. Surface fleetings as raw observation, not polished claims — promotion does the principle-stating; recall just preserves the shape.

Notes are LLM-voiced. Wikilinks appear in prose with surrounding context — that context is the relevance signal. Luhmann IDs (`1`, `1a`, `1a1`) signal lineage; following wikilinks reaches Luhmann-adjacent notes by construction.

## Modes

| Mode | Trigger | Explicit query |
|------|---------|----------------|
| **No-arg recap** | User said `/recall` with no topic, or you self-invoke for orientation | `engram recall` (anchors) seeded with `engram recall --recent` (latest activity) |
| **Topic query** | User named a topic, or you formed one from context | The topic, phrased as the user gave it |
| **Self-invoked** | You decided recall applies | Phrase your own seed; treated as a topic query |

If the user's invocation is ambiguous ("recall that thing"), make a best-effort phrasing rather than asking. If results are off-target, the user can refine.

## The retrieval pipeline

### Step 1 — Enumerate your current situation

Before retrieving, list features of your current situation — each becomes a query against memory.

**Apply this test to each candidate:**

> If a future-me on a fresh context were dropped into roughly this same situation, and there were one memory about *this feature alone*, would it be worth surfacing?

If yes, list it. If you can't imagine what that memory would even be about, the feature is either too generic ("coding") or too specific to this exact moment ("a bug at line 47").

**Cast wide.** Your situation includes everything continuously true — tooling, language, platform, project conventions, the kind of operation underway, the user's role and goal, what's loaded into your context. There are usually features you'll only notice on a second pass.

**Produce a list** — 5–15 short queryable phrases. Internal scratch, not part of the final output.

### Step 2 — Form the explicit query

For no-arg: combine anchors and recent activity (see Step 3 cascade).

For topic / self-invoked: phrase as the user gave it, or as you'd phrase a search.

**Query by task, not by fear.** What are you trying to do? Not what might go wrong. "implementing Claude Code hooks" — not "common mistakes when writing hooks." Memories are written to match task descriptions, so query the same way.

### Step 3 — Cascade retrieval via frontier expansion

The skill drives the cascade as a loop calling `engram recall`. The binary is a thin graph primitive; relevance evaluation lives here.

**Initial frontier:**

```bash
engram recall --vault /Users/joe/repos/personal/agent-memory
engram recall --vault /Users/joe/repos/personal/agent-memory --recent --limit 20
```

Union the outputs. These are the initial files to evaluate.

**Loop (until ≥100 surfaced memories OR frontier empties):**

1. **Evaluate the current frontier in parallel via subagents.** For each file in the frontier, dispatch a subagent (or batch a few per subagent) that reads the file and scores relevance against (a) the explicit query and (b) every situational feature from Step 1. Surface notes are those scoring above the relevance threshold.
2. **Track read files.** Maintain a cumulative `--already-read` set that includes every file read so far, whether surfaced or not.
3. **Expand the frontier** by calling:

   ```bash
   engram recall --vault <path> \
     --follow A,B,C \
     --already-read X,Y,Z,...
   ```

   `--follow` = basenames that scored above threshold *and* whose surrounding prose signaled there is more worth chasing. `--already-read` = the cumulative set. Basenames are extension-less (no `.md`).

4. **Repeat** from step 1 with the new frontier.

**Termination:**
- ≥100 surfaced notes → stop and synthesize.
- Empty frontier → stop and synthesize.

**Contradictions.** If two surfaced notes make incompatible claims about the same thing, mark them. The vault preserves contradictions; recall surfaces both, never picks a side.

### Step 4 — Synthesize for context injection

Format for an LLM reader (the parent agent). Wikilinks are required — the parent may re-read source notes for depth. Stay human-readable too.

**If a note matches both the explicit query and a situational feature**, surface it once under the more specifically relevant section and list both signals (e.g., add a `Also matches: <feature>` line). Don't duplicate the same note under two sections.

**Fleetings get their own section.** Fleetings are raw observation, not principle-stated; surface them under `### From recent fleetings` with the observation as-written. Don't translate to a principle.

**Output template** (use the structure; phrase content naturally):

```
## Recall — <mode>

### Vault state
(omit unless something structurally surprising about the vault is worth flagging — layout drift, unusual sparsity. Brief.)

### From your query: <explicit query phrasing>
- [[<wikilink>]] — <one-line claim or principle>
  Context: <1–2 sentence excerpt of in-prose framing>
- ...

### From your situation
- [[<wikilink>]] — <one-line claim>
  Matches: <situational feature(s) this applies to>
  Context: <1–2 sentence excerpt>
- ...

### From recent fleetings (pre-promotion, raw)
(omit section if no fleeting matches; if it has matches, list them here regardless of which seed they matched)
- [[<fleeting-wikilink>]] (fleeting) — <raw observation, as-written, 1–2 lines>
  Matches: <signal it matched — query / situation feature>
- ...

### Contradictions in the vault
(omit section if none)
- [[<note A>]] vs [[<note B>]] on <topic>
  <one-line summary of the disagreement>
```

Empty section — write `(no matches)` rather than omitting. Exception: if a section is empty *because* its matches were consolidated under another section per the dedup rule above, write `(matches consolidated above)`.

## Failure modes

| Situation | Behavior |
|-----------|----------|
| `--vault` not provided and `ENGRAM_VAULT_PATH` unset | `engram recall` errors; report "vault path required" and stop. |
| Vault directory does not exist | Report "vault not found" and stop. Do not create. |
| Vault exists but is empty | Report "vault is empty; no recall produced." Do not fabricate. |
| `engram recall` command not found | Fall back: read every `.md` under `MOCs/` and `Permanent/` and `Fleeting/` directly, scoring as in Step 3. Note the missing binary in *Vault state*. |
| No matches for explicit query | `(no matches)` for that section. Situational baseline may still produce. |
| No matches anywhere | State plainly. Normal early in a vault's life. |
| A note read fails | Log which note, continue with the rest. One bad note ≠ abort. |

## What this skill is not for

- Reading session transcripts. Use `engram transcript --from --to` if you need past-session activity.
- Writing to the vault. Capture and promotion are separate skills (`learn`).
- Inventing memories. If recall would surface nothing, surface nothing.
- Inventing classifications (confidence tiers, freshness scores, priority ranks) the upstream skills don't produce.
- Deduplicating against your prior context. The parent agent handles that; this skill returns full surfaced content.

## Discovery and trigger ceiling

This skill fires when the model recognizes the situation as recall-relevant from the description. Some genuinely relevant moments will be missed because the model didn't realize recall applied. That ceiling is accepted; proactive triggering (hooks) is a separate concern and out of scope for the skill itself.
