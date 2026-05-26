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
2. **Run a single `engram query`** to get back direct hits + clusters + hubs + a budget block, computed by the binary in one shot.
3. **Synthesize impact on the plan** — for each load-bearing surfaced note, say whether it confirms, adjusts, or contradicts what you were about to do.

Most of the skill's value is in (1) and (3). The query in (2) is mechanical; without (1) and (3) it is just a note dump.

## Vault structure

The binary resolves the vault automatically — `--vault` and
`ENGRAM_VAULT_PATH` are overrides, not requirements. Default:
`$XDG_DATA_HOME/engram/vault` (typically `~/.local/share/engram/vault`).
**Do not pass `--vault` in `engram query` invocations unless the user
explicitly tells you the vault is elsewhere.**

```
<vault>/
  Permanent/   atomic principle-stated notes; <Luhmann-ID>.YYYY-MM-DD.<slug>.md
  MOCs/        bootstrap stub only; no active content (historical MOCs live in <vault>/_legacy/MOCs/, audit only)
```

Notes are LLM-voiced. Luhmann IDs (`1`, `1a`, `1a1`) signal lineage; wikilinks inside notes encode authored relations. `engram query` walks the authored-wikilink graph itself — you do not chase links by hand.

## Modes

| Mode                                  | Trigger                                                                                                              | Explicit query                                                                                |
| ------------------------------------- | -------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------- |
| **Default — after request received**  | Any user request that might entail more than a single tool call or more than shallow thinking. Self-fire.            | Seed phrased from the request + your pre-recall plan; combined with situational baseline.     |
| **Topic query**                       | A topic is named, or you formed one from context.                                                                    | The topic, phrased as given.                                                                  |
| **No-arg recap**                      | `/recall` with no topic, or self-invoke for orientation.                                                              | A short phrase describing what you are doing right now (the active situation). Use a query string — `engram query` does not have a no-arg recap mode. |

**Skip only** for pure single-call lookups, trivial edits (typo, rename, one-liner), or follow-on turns where recall already ran for the same task. If a topic is ambiguous ("recall that thing"), make a best-effort phrasing rather than asking — if results are off-target, the user can refine.

## The retrieval pipeline

### Step 0 — Print your upfront judgement

**Before any `engram query` call**, print three short blocks to the user. Plain prose, no headers needed:

- **Ask:** what the user is asking, in your own words. One or two sentences.
- **Situation:** the concrete context — repo, branch, the operation underway, what's loaded into your context. One short paragraph.
- **Plan:** the action(s) you would take next absent any guidance from memory. Numbered list or short paragraph.

Skipping Step 0 is forbidden. The whole purpose of recall is to test a stated plan against memory; an unstated plan cannot be tested. If you find yourself starting an `engram query` tool call before this print appears in your user-facing output, stop and back up.

### Step 1 — Phrase queries from your plan and situation

The plan you just printed is the primary query seed. Re-read it and write down — internally — 5 to 15 short queryable phrases. Mix two kinds:

- **Plan-grounded** — phrases drawn directly from the actions you said you would take ("wire OpenCode reader alongside Claude Code", "advance per-harness marker").
- **Situational** — features continuously true around the action (tooling, language, the kind of operation, the user's role, what's loaded into context).

**Apply this test to each candidate:**

> If a future-me on a fresh context were dropped into roughly this same situation, and there were one memory about _this phrase alone_, would it be worth surfacing?

If yes, keep it. If you can't imagine what that memory would even be about, the phrase is either too generic ("coding") or too specific to this exact moment ("a bug at line 47").

This list is internal scratch — not part of the user-facing output.

### Step 2 — Form the explicit query

Collapse Step 1 into one or two short phrases for the `engram query` invocation. The binary embeds your query string, ranks every compatible-sidecar note by cosine similarity, expands a 3-hop subgraph over authored wikilinks, clusters that subgraph, and identifies in-degree hubs — all of that runs from a single query string, so spend your effort on the phrasing.

**Query by task, not by fear.** What are you trying to do? Not what might go wrong. "implementing Claude Code hooks" — not "common mistakes when writing hooks." Memories are written to match task descriptions, so query the same way.

### Step 3 — Run `engram query` once

One call, no loop, no follow-ups:

```bash
engram query "<your phrase from Step 2>"
```

The binary returns a YAML payload (full schema in `docs/superpowers/research/2026-05-25-f6-f91-spec.md`):

```yaml
version: 1
query: "..."
items:                              # direct hits ∪ cluster reps ∪ hubs, deduped
  - path: Permanent/...
    kind: fact
    score: 0.85
    provenances: [direct, cluster_rep, hub]
    cluster_id: 0                   # iff cluster_rep ∈ provenances
    in_degree: 9                    # iff hub ∈ provenances
    content: |
      <full text>
clusters:                           # may be empty []
  - id: 0
    size: 12
    silhouette: 0.43
    members:
      - { path: Permanent/..., score: 0.85, is_representative: true }
      - { path: Permanent/..., score: 0.71, is_representative: false }
budget:
  total_notes: 480
  with_embeddings: 480
  subgraph_size: 67
  subgraph_size_capped: false
  hops_traversed: 3
  clusters_found: 3
  hubs_returned: 5
  direct_hits_returned: 20
  items_with_full_content: 28
  limit: 20
```

`--limit N` caps direct hits considered for expansion. Cluster reps and hubs can appear in `items` beyond `--limit`. Default limit is 20; raise it (`--limit 50`) only when the topic is genuinely broad — wider limits expand the subgraph more aggressively. Lower it (`--limit 5`) when you want a narrow cone around a precise concept.

**No `--follow`, no `--already-read`, no rounds.** The 3-hop expansion, clustering, and hub identification happened inside the binary. Do not invoke `engram recall` here — that command exists for a different (legacy) flow; the F6+F9.1 pipeline runs through `engram query` only.

### Step 3a — Per-cluster synthesis gate

**Read this callout first. It controls everything below.**

> The rules in this section (path A/B/C, recall-mirror test, fact-vs-feedback categorization, Luhmann positioning) govern the **dispatched synthesis subagent**, not the /recall agent reading this skill. /recall's job here is one decision per cluster — *is there a binding principle worth capturing?* — and, if yes, dispatch. The /learn discipline lives inside the subagent's context after dispatch. Do not apply path A/B/C or the recall-mirror test to how you frame the recall reply to the user; those tests are for vault writes, not for the synthesis output you are about to produce.

**Step 3a is not optional.** Running it on every cluster is what makes clustering valuable. The binary produces clusters because that is the *only* place a binding principle across multiple notes can be detected at query time — no single member states it, so no embedding search will find it. Skipping the gate reduces /recall to a note-dump and silently throws away the one job clustering exists to enable. Time pressure, "user only asked for recall", and "the clusters are already there" are not exits from Step 3a — they are the moments the gate matters most, because that is when past-you's principles get lost in the surface.

For each cluster in `clusters[]` returned by `engram query`:

1. **Read the cluster representative's content.** It is already in `items[]` (dedup'd, full text). No extra read needed.
2. **Decide one thing only:** *is there a binding principle visible across cluster members that no single member states?* This is the only judgement /recall makes here.
   - **Cluster size < 3 members** → no. Skip.
   - **Cluster silhouette unimpressive** (the binary already filters silhouette < 0.10; everything you see passed that floor) → still ask the question; small silhouette can still bind around a shared principle.
   - **The shared content is generic vocabulary, not a principle** → no. Skip.
   - **The principle is already stated in any one member** → no. The vault already has it; skip.
3. **If yes, dispatch a subagent.** Pass it: the full member list (paths only — it will read them), the user's query string, and the criteria below. /recall does NOT wait for the subagent's output. The subagent writes (or declines to write) via `engram learn` in its own context; the result lands in the vault. Your user-facing reply (Step 4b) does not include the synthesis output — that is a vault write, not a recall surface.
4. **If no, the cluster stays as context.** Its members are already in `items[]` (the representative) and in `clusters[].members` (the rest). They will inform Step 4 like any other surfaced note.

**Synthesis criteria the subagent applies (NOT the /recall agent):**

- Cluster has ≥ 3 members. (Cheap re-check inside the subagent.)
- Binding principle is **not already stated** in any individual member.
- Principle passes the recall-mirror test (a future query about this kind of work would surface it).
- Principle is generalizable, not project-specific.
- `--source "synthesized from cluster, <YYYY-MM-DD>, context: <query>"`.
- `--relation "<luhmann-id>|<one-line rationale per constituent>"`, one bullet per cluster member.
- Path A/B/C and the fact-vs-feedback split apply per the /learn skill — the subagent loads /learn itself.

**Dispatch shape.** One subagent per cluster you decide passes the binding-principle test. Subagents are independent; dispatch them in parallel where the orchestration allows.

**The only carve-out for not dispatching:** the harness genuinely does not expose a subagent-dispatch tool in this environment. Verify that empirically — try to invoke the dispatch tool; if the tool is absent or refuses, note the cluster members in 4a as context and proceed. **"User is waiting", "clusters look organized enough", "the principle might already be in a member I haven't read", and "synthesis adds latency" are not the carve-out** — they are exactly the moments past-you's binding principle gets dropped on the floor. Do not inline-synthesize from /recall's own context as a workaround; that is the rationalization the gate exists to prevent.

**Hubs do not go through the gate.** Hubs are individual notes (high in-degree in the subgraph), not clusters. Surface them in Step 4b as orientation; do not synthesize from a hub alone.

### Step 4 — Closing synthesis: did the memories change the plan?

The query has produced a payload. Now metabolize it against the plan you printed in Step 0. Two outputs follow.

#### 4a. Structured form → agent context, NOT the user's screen

The full YAML payload — items, clusters, hubs (inline on items via `in_degree`), budget — is the parent LLM's working context. It is already present as the `engram query` tool-call result. **Do not re-emit the YAML in your user-facing reply.**

If you need a narrated, organized version for context propagation (e.g. you are handing off to a subagent that needs the recall surface but not the raw YAML), assemble it from the YAML using the template below — but only post it as agent-context, never as the user-facing reply.

**Structured-form template** (internal — for 4a, not user-visible):

```
## Recall — <mode>

### Anchor concepts (hubs)
- [[<wikilink>]] — in_degree: <n> — <one-line claim>
- ...

### From your query: <explicit query phrasing>
- [[<wikilink>]] — score <s> — <one-line claim or principle>
- ...

### Clusters
- Cluster <id> (size N, silhouette s): <one-line theme> — representative [[<wikilink>]]
- ...

### Contradictions in the vault
(omit section if none)
- [[<note A>]] vs [[<note B>]] on <topic>
  <one-line summary of the disagreement>
```

Empty section in 4a — write `(no matches)` rather than omitting. Exception: if a section is empty _because_ its matches were consolidated under another section, write `(matches consolidated above)`. These placeholders live in 4a only.

#### 4b. User-facing synthesis — how the memories did or did not change the plan

The user sees this. Rules:

- **Open with one short Anchor-concepts preface.** One sentence (or a tight comma list) naming the hubs as orientation: "Anchor concepts in this area: X, Y, Z." Then drop straight into the recap. The preface is the only place hubs surface in 4b. No bulleted hub list, no scores, no path strings. If no hubs returned, omit the preface entirely.
- **Open the recap with the count.** One sentence: "Query surfaced N items across C clusters, S in subgraph." That is it for recap framing.
- **Walk the plan from Step 0 in order.** For each numbered action or distinct piece of the plan, say plainly whether the memories **confirmed**, **adjusted** (and how), **contradicted**, or were **silent**. One short bullet per action.
- **Surface load-bearing contradictions inline as prose**, not as a separate section. If a contradiction matters for the next action, call it out where that action is being discussed.
- **No wikilinks.** Name notes by what they say, not by filename.
- **No `(no matches)` / `(matches consolidated above)`** placeholders — those belong in 4a.
- **No synthesis-write output.** If you dispatched a synthesis subagent in 3a, the user does not see its work here — that subagent writes to the vault directly. Do not narrate "I dispatched a subagent to synthesize cluster N" in the user reply; that is process scaffolding.
- **Length:** as long as it needs to honestly cover the plan; no filler. If memories were silent on every action, one sentence is fine.
- **If nothing surfaced at all**, say so in one sentence and stop.

#### Red flags — STOP, you are leaking the structured form or mis-routing /learn rules

If you catch yourself doing any of these in the user-facing reply, rewrite:

| Sign you're leaking or mis-routing                                          | What you should be doing                                                              |
| --------------------------------------------------------------------------- | ------------------------------------------------------------------------------------- |
| Writing `[[…]]` wikilinks in the reply                                      | Paraphrase the claim; no wikilinks in 4b                                              |
| Writing a `### Contradictions` section                                      | Contradictions stay inline where they affect the plan; no separate section            |
| Writing `### From your query` / `### Clusters` / `### Anchor concepts` headers in 4b | No structured-form headers in 4b — preface + walk-the-plan only                       |
| Recap is a generic "highlights:" bullet list with no plan reference         | You skipped synthesis. Restart Step 4b: walk the plan from Step 0 and judge each piece |
| You never printed Step 0                                                    | Back up. The whole skill is a no-op without it                                        |
| You wrote `engram recall --follow` / `--already-read` / multiple rounds     | The cascade is gone. Re-do Step 3 with a single `engram query` call                   |
| You applied path A/B/C or the recall-mirror test to how you wrote 4b        | Those rules are for synthesis subagents, not for the /recall reply. Re-read the 3a callout |
| You called fact-vs-feedback categorization on a note you were summarizing   | Same mis-routing. Categorization is a /learn-write decision, not a recall surface     |
| You inline-synthesized a cluster instead of dispatching                     | Dispatch one subagent per cluster that passed the gate. Inline-synthesis is forbidden — it is the rationalization 3a exists to prevent. |
| You skipped Step 3a because clusters "look organized" or the user "only asked for recall" | The clusters being organized is a precondition for the gate, not an exit from it. Run 3a on every cluster. The only carve-out is a missing dispatch tool — verify empirically. |
| `(no matches)` or `(matches consolidated above)` appears in 4b              | Those are 4a placeholders; in 4b just say "memories were silent on this action"       |
| You called `engram recall` anywhere in the pipeline                         | `engram recall` is the legacy command. The F6+F9.1 pipeline is `engram query` only.   |

## Failure modes

| Situation                                            | Behavior                                                                                                                                       |
| ---------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| `engram query` reports vault not found (`query: scan: ...`) | Tell the user the vault is missing and to run `engram learn` (which bootstraps it) or set `ENGRAM_VAULT_PATH` to an existing vault. Do not create the directory yourself. |
| Vault exists but is empty (`items: []`, `clusters: []`, `total_notes: 0`) | Report "vault is empty; no recall produced." Do not fabricate.                                                                                |
| Vault has notes but no current-model embeddings (`query: vault has notes but no current-model embeddings; run \`engram embed apply --all\``) | Surface the binary's instruction verbatim. Do not score by hand. Do not invent a fallback that pretends embeddings exist.                      |
| No `items` for your query (subgraph empty, clusters empty, hubs empty) but vault otherwise healthy | State plainly in 4b: "Query surfaced nothing." Normal for sparsely-covered topics.                                                             |
| `engram query` command not found                     | Degraded mode: read every `.md` under `<vault>/Permanent/` directly, score relevance against the explicit query inline, surface top matches in 4b with a note that the binary was missing. No clustering, no hubs, no Anchor-concepts preface. Skip Step 3a entirely — there are no clusters to gate. |
| A single note read fails (degraded mode)             | Log which note, continue with the rest. One bad note ≠ abort.                                                                                  |

## What this skill is not for

- Reading session transcripts. Use `engram transcript` if you need past-session activity.
- Writing to the vault from /recall directly. Capture is the `learn` skill. (The synthesis subagent dispatched in 3a writes — but it is a /learn invocation in a separate context, not a /recall write.)
- Driving a manual link-cascade. `engram recall --follow` is the legacy primitive; do not invoke it from this skill.
- Computing the Luhmann ID, choosing path A/B/C, or applying the recall-mirror test in the /recall reply. Those are /learn responsibilities; if 3a dispatches a synthesis subagent, that subagent handles them.
- Inventing memories. If `engram query` returns nothing, surface nothing.
- Inventing classifications (confidence tiers, freshness scores, priority ranks) the binary does not produce.
- Deduplicating against your prior context. The parent agent handles that.

## Discovery and trigger ceiling

This skill fires when the model recognizes the situation as recall-relevant from the description. Some genuinely relevant moments will be missed because the model didn't realize recall applied. That ceiling is accepted; proactive triggering (hooks) is a separate concern and out of scope for the skill itself.
