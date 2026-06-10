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
2. **Run a single `engram query --synthesize-l2`** to get back direct hits + per-cluster `nearest_l2` bands + a budget block, computed by the binary in one shot, then crystallize the L2 your task needs from the bands.
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

Notes are LLM-voiced. Luhmann IDs (`1`, `1a`, `1a1`) signal lineage; wikilinks inside notes encode authored relations. For a blended (untiered) `engram query`, the binary walks the authored-wikilink graph itself — you do not chase links by hand. **The exception is a tier-capped or distilled read:** when a read returns only a subset of tiers (e.g. a `--tier`-capped query, or the L1+L2 union in `--synthesize-l2` mode), each item also carries `outbound_links` (the basenames one hop away) and you fetch any of them on demand with `engram show <basename>` — see [Following a cited note on demand](#following-a-cited-note-on-demand-engram-show).

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

The plan you just printed is the primary query seed. Re-read it and write down 5 to 15 short queryable phrases. Mix two kinds:

- **Plan-grounded** — phrases drawn directly from the actions you said you would take ("wire OpenCode reader alongside Claude Code", "advance per-harness marker").
- **Situational** — features continuously true around the action (tooling, language, the kind of operation, the user's role, what's loaded into context).

**No pre-filter test.** Don't ask "would a memory about this phrase be worth surfacing" — you can't know what's in the vault before you query. Generate the phrases; the binary ranks every compatible-sidecar note for each, and Step 3a's per-cluster gate handles relevance downstream. The only phrases to drop are obvious dross (a literal single noun like "coding", or a phrase that names a specific line of code in this session).

This list is internal scratch — not part of the user-facing output, but it IS the input to Step 3 (every phrase becomes a `--phrase` flag in the single `engram query` call).

### Step 2 — Use Step 1's phrases as-is

**Do not collapse Step 1 into "one or two short phrases."** Each Step 1 phrase becomes a separate `--phrase` flag in the Step 3 `engram query` call. Paraphrasing or merging phrases loses the distinct retrieval angles the original wording carried — a collapsed phrase risks weak matches that miss the specific details each Step 1 phrase was probing.

**Query by task, not by fear.** What are you trying to do? Not what might go wrong. "implementing Claude Code hooks" — not "common mistakes when writing hooks." Memories are written to match task descriptions, so query the same way.

### Step 3 — Run one `engram query --synthesize-l2` with all Step 1 phrases as `--phrase` flags

Issue a single `engram query` call in **`--synthesize-l2` mode**, passing each Step 1 phrase as a separate `--phrase` flag. In this mode the binary unions the matched **L1 + L2** neighborhood, clusters it **once** with each note positioned by the vector that matched it, and returns — per cluster — a raw **`nearest_l2 {path, cosine}`**: the centroid→nearest-existing-L2 cosine (the stronger of the situation/body axes). The binary runs one sub-pipeline per phrase, merges results (max score, union provenances, per-phrase clusters), and returns a single aggregated payload.

```bash
engram query --synthesize-l2 \
  --phrase "<Step 1 phrase 1>" \
  --phrase "<Step 1 phrase 2>" \
  --phrase "<Step 1 phrase 3>"
  # ... one --phrase per Step 1 phrase
```

**`--synthesize-l2` is the locked default read.** The 2026-06-08 cumulative-accumulation eval found L2 (the distilled facts/feedback) is the highest-value tier to surface for build/convention work and that surfacing L1 episodes or L3 ADRs by default did not help (often hurt weak models). L2 is what you build against. But under lazy synthesis the covering L2 may not exist yet — so this mode unions the matched **L1 + L2** evidence, clusters it, and hands you the raw `nearest_l2.cosine` per cluster so Step 3a can crystallize the missing L2 on demand. **The binary applies no decision — it emits the raw cosine; the three bands in Step 3a are the skill's.** You may still reach a specific cited note on demand with `engram show <basename>` (see [Following a cited note on demand](#following-a-cited-note-on-demand-engram-show)).

The binary returns a YAML payload (full schema in `docs/superpowers/research/2026-05-25-f6-f91-spec.md`):

```yaml
version: 1
phrases: ["...", "...", "..."]      # all queried phrases
items:                              # direct hits ∪ cluster reps, deduped across phrases (no hubs in this mode)
  - path: Permanent/...
    kind: fact
    score: 0.85
    provenances: [direct, cluster_rep]
    cluster_id: 0                   # iff cluster_rep ∈ provenances
    content: |
      <full text>
clusters:                           # per-phrase clusters from the matched L1+L2 union; each tagged with phrase
  - id: 0
    phrase: "..."                   # which phrase produced this cluster
    size: 4
    silhouette: 0.43
    members:
      - { path: Permanent/..., score: 0.85, is_representative: true }
      - { path: Permanent/..., score: 0.71, is_representative: false }
    nearest_l2:                     # RAW centroid→nearest-existing-L2 cosine; NO band decision applied
      path: Permanent/12.2026-04-02.storage-atomicity.md
      cosine: 0.86
budget:
  phrases_queried: 3
  total_notes: 480
  with_embeddings: 480
  subgraph_size: 67
  clusters_found: 3
  direct_hits_returned: 20
  items_with_full_content: 28
  limit: 20
```

`--limit N` caps direct hits per phrase considered for expansion. Cluster reps can appear in `items` beyond `--limit`. Default limit is 20; raise it (`--limit 50`) only when the topic is genuinely broad. Lower it (`--limit 5`) when you want a narrow cone around a precise concept.

**No agent-side merging.** The binary deduplicates items by path (max score, union provenances), retains clusters per-phrase (each with its `nearest_l2`), and aggregates the budget. Step 3a iterates per cluster across all `clusters[]` in the single payload, banding each on its `nearest_l2.cosine`.

### Step 3a — Per-cluster three-band L2 synthesis (blocking, dispatch-driven, role-split)

**Read this callout first. It controls everything below.**

> Step 3a is a **role-split** between two agents — the parent /recall agent reading `nearest_l2.cosine` and choosing a band, and the dispatched synthesis subagent reading every member from disk and writing the L2. The parent never reads all cluster members; it sees only the representative (already in `items[]`) and the cluster's `nearest_l2 {path, cosine}`. **The band is the parent's call** — it has the cosine in the payload and the subagent does not (the subagent has no centroid, so it cannot recompute the band). The subagent *executes the band's assigned action* (fold-in vs new L2), reading members for synthesis quality and the recency tiebreaker — it does NOT re-decide whether to write. The parent does NOT apply the fact-vs-feedback split — that governs the subagent's /learn invocation, not the recall reply.

**Step 3a is not optional.** Running the band rule on every cluster is what makes clustering valuable. The binary unions the matched L1+L2 evidence and clusters it because under lazy synthesis the covering L2 may not exist yet — Step 3a is where a missing L2 gets crystallized, or a stale one folded forward, from evidence no single note states. Skipping the gate reduces /recall to a note-dump and silently throws away the one job clustering exists to enable. Time pressure, "user only asked for recall", and "the clusters are already there" are not exits from Step 3a — they are the moments the gate matters most, because that is when the L2 your current task needs never gets written.

#### Parent /recall: the three-band rule on `nearest_l2.cosine`

For each cluster in the payload's `clusters[]` list:

1. **Read the cluster representative's content** (already in `items[]`, no extra read needed) and note the cluster's `nearest_l2 {path, cosine}`.
2. **Precondition — cluster size ≥ 3 members.** Otherwise skip — the cluster is too thin to synthesize an L2 from. (Sub-threshold clusters stay as context for Step 4, like any other surfaced note.)
3. **Band on `nearest_l2.cosine`** (defaults `0.95` / `0.80` — **the harness may override them by naming different values in the recall instruction; use those if given**):

   - **cosine ≥ 0.95 → NO-OP.** Skip. An existing L2 (`nearest_l2.path`) already represents this cluster; there is nothing to crystallize. **Do not dispatch, do not `engram learn`.** Its members stay as context for Step 4.

   - **0.80 ≤ cosine < 0.95 → UPDATE.** Dispatch a synthesis subagent to **fold the cluster's members into the nearest existing L2**. The subagent issues:

     ```bash
     engram learn fact|feedback --target <luhmann-id> --position continuation \
       --source "synthesized from cluster, <YYYY-MM-DD>, context: <query>"
     ```

     The `<luhmann-id>` is the **filename prefix of `nearest_l2.path`** — e.g. `Permanent/12.2026-04-02.storage-atomicity.md` → `--target 12`; `Permanent/12a.2026-05-01.filestore-interface.md` → `--target 12a`. **NO `--tier` flag** — absence means L2. Adding `--tier L3` is the easy mistake; it is **forbidden** (this skill never writes L3). (Where the members disagree, the dispatched subagent folds in the **more-recently-`created`** member's stance on the conflict — see the recency-bias step below.)

   - **cosine < 0.80 → CREATE.** Dispatch a synthesis subagent to **write a NEW L2** synthesizing the cluster — a Fact **and/or** a Feedback per the fact-vs-feedback split in `/learn` §4. The subagent issues, for each new note:

     ```bash
     engram learn fact|feedback --position top \
       --relation "<member-luhmann>|<one-line rationale>" \   # one --relation per member: every L1 AND L2
       --source "synthesized from cluster, <YYYY-MM-DD>, context: <query>"
     ```

     One `--relation` bullet per cluster member (all of them — L1s and L2s alike). **NO `--tier` flag** — absence means L2. `--tier L3` is **forbidden**.

4. **BLOCKING — wait for the writes, then apply them. This is the key behavior.** Unlike fire-and-forget synthesis (which served *future* recalls), these L2 writes are for **your current task**. Dispatch the per-cluster UPDATE/CREATE subagents (one per banded cluster, in a single parallel tool-use block), **WAIT for every one to finish**, then **read the resulting notes** (`engram show <basename>`, or the path the subagent reports) and **apply them to your current task** before proceeding. Do not move on while writes are in flight; do not fire-and-forget. (The NO-OP band dispatches nothing, so there is nothing to wait on there.)

**Dispatch shape.** One subagent per cluster in the UPDATE or CREATE band that meets the size precondition. Subagents are independent; dispatch them in a single parallel tool-use block, then block on the whole batch.

**The only carve-out for not dispatching:** the harness does not expose a subagent-dispatch tool in this environment. Verify empirically — try to invoke the dispatch tool; if it is absent or refuses, still record the band decision and the **exact `engram learn` invocation(s)** each banded cluster would issue (the UPDATE/CREATE commands above, with the resolved `--target`/`--relation`), note the cluster members in 4a as context, and proceed. **"User is waiting", "clusters look organized enough", and "synthesis adds latency" are not the carve-out** — they are exactly the moments the L2 your task needs never gets written. Do not inline-synthesize from /recall's own context as a workaround; the parent has only seen the rep, so any inline synthesis is uninformed by definition.

#### Dispatched synthesis subagent: read members, synthesize, recency-bias

The subagent receives the cluster's member list (paths), the band's assigned action (UPDATE `--target <id>`, or CREATE), the query string, and the rep's content. It then:

1. **Reads all member notes from disk.** This is non-negotiable. Without the full member content the synthesis is impossible — and the read is also where it captures each member's **`created` frontmatter** for the recency tiebreaker below.
2. **Executes the band's action** — it does NOT re-decide whether to write (the parent's band already decided that):
   - **UPDATE** → fold the members' substance into the nearest L2 via `engram learn fact|feedback --target <id> --position continuation …` (no `--tier`).
   - **CREATE** → write a new L2 (Fact and/or Feedback) synthesizing the members via `engram learn fact|feedback --position top --relation … --source …` (no `--tier`), `--relation`-linked to **every** member.
3. **Recency-bias on divergence.** Where members **diverge or conflict**, prefer the **more-recently-`created` member** (compare the `created` frontmatter read in step 1) as the binding stance. This is a **tiebreaker on conflict**, not a discard of non-conflicting older content — fold in everything the members agree on; only on a genuine contradiction does the newer member win.

**Subagent criteria for synthesis writes:**

- Cluster has ≥ 3 members. (Cheap re-check inside the subagent.)
- `--source "synthesized from cluster, <YYYY-MM-DD>, context: <query>"`.
- CREATE writes: `--relation "<luhmann-id>|<one-line rationale per constituent>"`, one bullet per cluster member (every L1 and L2). UPDATE writes target the nearest L2 with `--target <id> --position continuation` and need no per-member `--relation`.
- **Never `--tier`.** Absence = L2. `--tier L3` is forbidden — this skill writes only L2.
- The fact-vs-feedback split applies per the /learn skill — the subagent loads /learn itself.

(Project-specific principles — those that name a project, issue, or named decision — are NOT excluded at this layer. Tracking project metadata in vault notes is a follow-up captured in `docs/issues.md`; for now, write the principle as it is and let the next iteration sharpen the framing.)

### Step 4 — Closing synthesis: did the memories change the plan?

The query has produced a payload. Now metabolize it against the plan you printed in Step 0. Two outputs follow.

#### 4a. Structured form → agent context, NOT the user's screen

The full YAML payload — items, clusters (with `nearest_l2`), budget — is the parent LLM's working context. It is already present as the `engram query` tool-call result. **Do not re-emit the YAML in your user-facing reply.**

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
- **Frame load-bearing conventions as requirements, not background.** When a surfaced note carries a convention or principle relevant to what you're about to build, present it as a **requirement to implement** — lead with "Apply these as requirements:" and list them — not as a memory that merely "confirmed" the plan. Treat each surfaced convention as a must-do **even when the memory states it generally**; a plan step the memory confirms still inherits that convention's concrete specifics as requirements on the step. This is the framing the matrix found wins; passive background framing is the weaker behavior it penalized.
- **Surface load-bearing contradictions inline as prose**, not as a separate section. If a contradiction matters for the next action, call it out where that action is being discussed.
- **No wikilinks.** Name notes by what they say, not by filename.
- **No `(no matches)` / `(matches consolidated above)`** placeholders — those belong in 4a.
- **Apply the freshly-minted L2s.** If you dispatched UPDATE/CREATE subagents in 3a, you **waited** for them and read the resulting notes (Step 3a is blocking). You may briefly state which conventions you crystallized (or folded into an existing L2) and that you are **applying them to the current task** — that is load-bearing, not scaffolding. Do not narrate the dispatch mechanics ("I spawned a subagent for cluster N"), do not paste `[[wikilinks]]`, and do not dump the raw note YAML — paraphrase the convention you are now applying.
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
| You ran separate `engram query "<phrase>"` calls instead of `--phrase` flags | One `engram query --phrase "<p1>" --phrase "<p2>" ...` call does the merging server-side. Go back to Step 3 and re-issue as a single call. |
| You applied the fact-vs-feedback split or /learn placement to how you wrote 4b | Those rules are for the synthesis subagent's /learn invocation, not the /recall reply. The parent only chooses the band. Re-read the 3a callout |
| You called fact-vs-feedback categorization on a note you were summarizing   | Same mis-routing. Categorization is a /learn-write decision, not a recall surface     |
| You fired-and-forgot the L2 writes instead of waiting | The L2 synthesis is **BLOCKING** — unlike fire-and-forget synthesis for future recalls, these L2s serve your *current* task. Dispatch the UPDATE/CREATE subagents, **wait** for them, read the resulting notes, and apply them before proceeding. |
| You added `--tier L3` (or any `--tier`) to a synthesis write | This skill writes only L2; **absence of `--tier` means L2**. `--tier L3` is forbidden. Re-issue the `engram learn` without any `--tier` flag. |
| You wrote an `engram learn` for a cluster whose `nearest_l2.cosine ≥ 0.95` | That band is **NO-OP** — an existing L2 already covers it. Do not dispatch, do not write. Leave the members as context. |
| You inline-synthesized a cluster instead of dispatching                     | Dispatch one subagent per UPDATE/CREATE-band cluster meeting the size precondition. Inline-synthesis is forbidden — the parent has only seen the rep, so any inline synthesis is uninformed. |
| You skipped Step 3a because clusters "look organized" or the user "only asked for recall" | The clusters being organized is a precondition for the gate, not an exit from it. Run the band rule on every cluster. The only carve-out is a missing dispatch tool — verify empirically. |
| `(no matches)` or `(matches consolidated above)` appears in 4b              | Those are 4a placeholders; in 4b just say "memories were silent on this action"       |
| You read all cluster members from the parent /recall agent                  | The parent reads only the representative. Member reads happen inside the dispatched subagent. If you find yourself opening member files in /recall's own context, you're inline-synthesizing — dispatch instead. |
| You reported surfaced conventions as passive background ("memory confirmed your DI plan") instead of as requirements | Frame load-bearing conventions as requirements to implement — "Apply these as requirements: …". Passive-background framing is the weaker behavior the matrix penalized; the consuming agent should treat each surfaced convention as a must-do. |

## Following a cited note on demand (`engram show`)

A blended (untiered) query already expands a 3-hop subgraph for you — there you do not chase links by hand. But when the read is narrowed — a `--tier`-capped query, the L1+L2 union of `--synthesize-l2` mode, or acting on a distilled higher-tier note — engram surfaces only that subset, and each item carries an `outbound_links` list: the basenames one hop away (typically the lower-tier notes a distilled L3/L2 standard was built from). `engram show` is also how you read back an L2 a Step-3a synthesis subagent just wrote, once you have blocked on it.

When a surfaced note cites a constituent you need and its content was not returned (the tier cap excluded it), fetch it on demand — do **not** drop the cap and re-query, and do not read files from disk:

```bash
engram show <basename>     # e.g. engram show 14.2026-06-05.headless-agents-do-not-self-fire-skills
```

`engram show` is read-only. It takes a full basename, a `[[wikilink]]`, a trailing `.md`, or a bare Luhmann id, and prints the note's full content **plus its own `outbound_links`**, so one fetch reveals the next hop. This is the sanctioned follow-on-demand path under a tier-capped or distilled read: it lets you hold a small distilled surface and pull the underlying specifics only when a note actually needs them, without re-surfacing everything. It does not contradict "do not chase links by hand" — that rule forbids re-implementing the blended 3-hop expansion, not fetching a specific cited note.

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
- General capture of what just happened. That is the `learn` skill. Recall is, however, a **deliberate sometimes-writer**: the three-band rule in Step 3a crystallizes (CREATE) or folds-forward (UPDATE) the L2 your current task needs, blocking on those writes and applying the result in-task. So "recall never writes" is wrong — recall does not do *general* capture, but it *does* synthesize L2 on demand and use the output. The writes themselves still go through `engram learn` (in the dispatched subagent's context); recall decides the band, waits, and consumes.
- Driving a manual link-cascade *in place of* `engram query`'s blended 3-hop expansion — that expansion happens inside the binary; you do not re-implement it by hand. (Fetching a specific cited note under a tier-capped or distilled read is a different, sanctioned move — see [Following a cited note on demand](#following-a-cited-note-on-demand-engram-show).)
- Applying the fact-vs-feedback split, or the /learn placement discipline, in the /recall reply. Those are /learn responsibilities; the synthesis subagent dispatched in 3a loads /learn and handles them. The parent /recall only chooses the band (no-op / update / create) from `nearest_l2.cosine` and the `--target` Luhmann id (the filename prefix of `nearest_l2.path`).
- Inventing memories. If `engram query` returns nothing, surface nothing.
- Inventing classifications (confidence tiers, freshness scores, priority ranks) the binary does not produce.
- Deduplicating against your prior context. The parent agent handles that.
- Field-targeted queries (subject/object/predicate, time-window filters). The binary does not yet expose `--field` flags; if you find yourself wanting to query by frontmatter field, that's an engram feature gap captured in `docs/superpowers/research/2026-05-26-v3-field-query-research.md`, not a /recall workaround.

## Discovery and trigger ceiling

This skill fires when the model recognizes the situation as recall-relevant from the description. Some genuinely relevant moments will be missed because the model didn't realize recall applied. That ceiling is accepted; proactive triggering (hooks) is a separate concern and out of scope for the skill itself.
