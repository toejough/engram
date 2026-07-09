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
4. **Tag-nominate and ride-along** — the binary nominates notes sharing a vocab term with the top-3 delivered notes and inserts superseded-note supersessors at the next rank; the agent judges the surfaced candidates, never links.
5. **Synthesize impact on the plan** — confirm / adjust / contradict / silent, per planned action.
6. **Re-enter for emergent recommendations** — a recommendation conceived mid-work gets its own
   lever-keyed query before it ships (Step 3.5).

The binary resolves the vault and chunk index automatically (`$XDG_DATA_HOME/engram/...`;
`ENGRAM_VAULT_PATH` / `ENGRAM_CHUNKS_DIR` override). **Do not pass `--vault` or `--chunks-dir`.**

## Modes — `glance` vs `deep` (the depth dial)

Recall runs in one of two **modes**, selected by the caller (the mode word is the skill argument; absent → `deep`):

- **`deep` (default).** The full procedure below — all 10 phrases and the write side (Steps 2.5C, Step 4).
  It both *applies* memory to this decision **and** *grows the vault* (crystallizes, persists synthesis).
  Use it when the decision is weighty or irreversible, when you want recall to also learn, or when in doubt.
- **`glance` (opt-in, cheap — for firing often).** A pass that is **read-only with respect to vault knowledge**
  (Step 2.7 `activate` still bumps the used-notes recency metadata — that is kept, not a knowledge write). Run
  Steps 0–3.5 with **~3 phrases** (not 10) and **keep the read side** — Step 2.5A (read candidates), **Step 2.5B
  (apply the recency weight)**, Step 2.7 (activate used notes), the Step 3 synthesis, and Step 3.5 (the
  re-entry query, when triggered) — but **skip the write side**: Step 2.5C (coverage amend/learn), Step 4
  (synthesis-persist). Glance *applies* memory to this decision; it does **not** grow the vault's knowledge.

**Escalate `glance` → `deep` for recency-channel standards (C5).** Glance reliably *surfaces* a recent-activity
(Channel 2) item but does **not** elevate it to a requirement — measured: glance honors a recent-channel
standard **0/5** where deep honors it **4/5** (#661 full-bars). So if your decision turns on **honoring a
standard that surfaced in the recent-activity channel** (a "use X going forward" / "the new convention is Y"
item in Channel 2), **switch to `deep`**. Glance is validated as deep-equivalent only for applying conventions
(C3), recency *supersession within the matched set* (C4i, via 2.5B), and abduction/synthesis (C6).

Everything below is the `deep` procedure; a **[glance: …]** note marks each step that differs under `glance`.

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

> **[glance: generate ~3 phrases, not 10 — the measured retrieval floor, #661 Phase 1. Breadth is for crystallization; glance only needs this decision's lesson.]**

Always generate exactly **10** short queryable phrases, one from each of these angles:

1. **Situation/setting** — the concrete environment you are operating in.
2. **User's intent/goal** — what the user ultimately wants to achieve.
3. **Current concrete action** — the specific thing you are about to do next.
4. **Problem/blocker** — the obstacle or constraint you are addressing.
5. **Candidate solution/approach** — the technique or strategy you plan to apply.
6. **Tooling/tech in play** — the specific tools, libraries, or languages involved.
7. **Prior related work** — previous work in this area you may be building on.
8. **Adjacent technique** — a related approach worth comparing or cross-checking.
9. **Failure mode to avoid** — a known pitfall or anti-pattern relevant to this work.
10. **Domain/concept** — the broader conceptual area the task lives in.

Each phrase is short and specific. No pre-filtering: you can't know what's in memory before you
query. Drop only obvious dross (a bare noun like "coding"). **Query by task, not by fear** —
"implementing Claude Code hooks", not "common mistakes when writing hooks". The binary caps
results to the top-30 matches per phrase.

### Step 2 — Run ONE unified `engram query` with all phrases

```bash
engram query --lazy-chunks \
  --phrase "<phrase 1>" \
  --phrase "<phrase 2>"
  # ... one --phrase per Step 1 phrase (deep: 10; glance: ~3)
```

One call; the binary merges ranking server-side. `engram query` always runs the unified D1
clustering of the matched notes+chunks in one pass and emits `candidate_l2s: [{path, cosine, content}]`
per cluster. The candidate pool includes the within-cluster top-5 **plus tag-nominated notes** — notes
sharing a vocab term with the top-3 delivered notes (budget fields `tag_nominations_added`/`dropped`
report the pool size). Do NOT collapse phrases, do NOT run per-phrase calls.

The payload has **two channels**:

**Channel 1 — Relevance (clustered matched items):** Items matched by your 10 phrases, bounded
to ~300 (top-30 per phrase, unioned, relevance-floor applied). These are clustered and carry
`candidate_l2s` per cluster (see Step 2.5). Read this channel to surface applicable lessons and
judge coverage. The payload's `items` mix:

- `kind: chunk` — raw transcript/doc fragments with source + anchor. These are EVIDENCE:
  extract the convention, decision, or correction they show (a reviewer correcting code, a
  stated standard); never quote them wholesale. **Under `--lazy-chunks` (recall's default
  invocation — confirm via `budget.lazy_chunks: true`) chunk items carry path + source/anchor
  but NO `content` field: `engram show-chunk <source#anchor>` to read a chunk's evidence on-demand.**
- `kind: fact` / `feedback` — crystallized lessons; apply directly (notes always carry full content inline).

**Channel 2 — Recent activity (un-clustered):** Items tagged `provenance: recent` — the newest
chunks by ingest time, appended after the matched set, NOT cluster members. Read this block
first for situational continuity — re-immerse in recent work before diving into the clustered
results. These items are NOT used for coverage or synthesis judgment. Do not treat them as
matched results; they have no cluster membership and no `candidate_l2s`. Under `--lazy-chunks`
recent items also carry path/source only (no content) — the paths show where your recent activity
was; `engram show-chunk <source#anchor>` for detail if a specific one matters.

- **Recent items are your own recent activity.** Chunks from a recent source with `turn-N`
  anchors are first-person `ASSISTANT:` narration you produced in a just-prior or
  pre-context-clear session. Treat them as your own past actions — do not re-derive them,
  do not express surprise at them, and dedup against what is already in your context.

If the matched items (Channel 1) are empty, say so in one sentence, skip Step 2.5, and proceed
with your plan. (A non-empty recent-activity block alone does not count as "something surfaced"
for coverage purposes.)

### Step 2.5 — Lazy note synthesis from the clustering (agent-judged)

The query output's `clusters` list contains the unified clustering of matched chunks
and notes. Each cluster carries `candidate_l2s: [{path, cosine, content}]` — the within-cluster
top-5 notes ranked from the cluster's own matched members, **plus any tag-nominated notes** whose
vocab terms overlap the top-3 delivered notes (nominated notes may cross cluster boundaries). A note
that did not match any phrase AND was not nominated will never appear as a candidate. Superseded-note
ride-alongs are inserted at the next rank after the note they supersede. A cluster with no note
members yields an empty `candidate_l2s` list; skip to the next cluster when that happens.
**Process every cluster.** For each:

**A. Read candidates and members**

`candidate_l2s` entries carry their `content` inline — read it directly; **no `engram show` calls for
candidates** (the same content is also on the matching `items[]` note member). For chunk
members, the content is NOT in the payload (chunks carry path/source only under `--lazy-chunks`;
the cluster's `members` list never carries content) — `engram show-chunk <source#anchor>` to read the
evidence on-demand. Do not judge coverage before you have read the candidate content.

**B. Apply the recency weight to resolve conflicts**

Evidence **conflicts** when a newer member explicitly negates or reverses an older claim. Reversal
cues: "no longer", "replaced by", "use X not Y", or the same subject+predicate appearing with a
different object in a newer item. When conflict is present: **recent wins**. When no conflict:
treat older and newer evidence as independently valid — do not demote a stable convention merely
because it lacks a recent instance.

> **[glance: SKIP Step 2.5C — it is the write side. Read 2.5A + apply 2.5B, do not amend/learn — continue to Step 2.7 (activate).]**

**C. Judge coverage against the recency-weighted view — in this order**

| Outcome | Criterion | Action |
| --- | --- | --- |
| **Covered** | A candidate's claim states the cluster's principle with **no material omission** vs the recency-weighted members | `engram amend --target <candidate-path> --activate --chunk-source <new-chunk-ids>` — provenance-enrich only; **do not rewrite content**. If this note CORRECTS/narrows/refutes a surfaced note, also pass `--supersedes "<basename>\|<type>\|<claim>"` (types: `updates\|narrows\|refutes`). |
| **Near** | A candidate addresses the same situation but omits ≥ 1 substantive claim the members evidence (judge against the recency-weighted view — a candidate that only matches the superseded content is **near**, not covered) | `engram amend --target <candidate-path> --chunk-source <chunk-ids> --subject ... --predicate ... --object ...` (or `--behavior/--impact/--action`) — re-synthesize content from all members, recency-weighted. Add `--supersedes "<basename>\|<type>\|<claim>"` if this note corrects a surfaced note. |
| **Absent** | No candidate addresses the situation | Invoke the **write-memory** skill with this handoff — kind=fact or feedback (pick per the cluster's principle), situation + content fields, `--source "<descriptive>"`, the cluster's chunk-source IDs, plus supersedes details if the new note corrects a surfaced note. write-memory composes, executes, and reports the note path. |

**One write per cluster; one representative note per cluster.** The representative is always a note
(never a chunk). For `absent`, write exactly one note (fact *or* feedback) covering
the cluster's principle. Do not write one fact and one feedback note for the same cluster.

Pass one `--chunk-source <source#anchor>` (repeatable) for every **chunk** source in the cluster
(provenance tracking). For `learn`, pass the same flag plus `--source` (human-readable provenance).
Vocab tags are assigned **automatically** by the binary on every write — do not hand-author them.

**WAIT for each write before moving to the next cluster.** Writes are blocking and inline — the
note created or updated by one cluster may be a candidate for another.

### Step 2.7 — Activation (use-driven, after synthesis)

After processing all clusters, call `engram activate`
on the notes you actually drew on — the `candidate_l2s` you judged Covered or Near at the
coverage table, and any notes you cited in the Step 3 synthesis:

```bash
engram activate \
  --note "<path of note you judged Covered or Near>" \
  --note "<path of note you cited in Step 3>"
  # ... one --note per used note only
```

Do NOT activate every returned note. Do NOT activate recent-channel items (chunks are never
activated). Activating only what you used lets superseded-but-surfaced notes fade via recency
rank — bumping every returned note would defeat the recency-competition mechanism. Skip this
call when you drew on no notes (e.g., payload was empty or Step 2.5 was skipped).

### Step 3 — Closing synthesis: did the memories change the plan?

> **[glance: before synthesizing, check for a load-bearing Channel 2 (recent-activity) standard. If your decision turns on honoring a recent convention, escalate to `deep` now — glance surfaces Channel 2 items but does not elevate them to requirements (C5, #661).]**

The user sees this. Rules:

- **Open with the count.** One sentence: "Query surfaced N items (K chunks, M notes); crystallized J lessons."
- **Walk the plan from Step 0 in order.** Per numbered action: **confirmed**, **adjusted** (and how), **contradicted**, or **silent**. One short bullet each.
- **Frame load-bearing conventions as requirements.** Lead with "Apply these as requirements:" and list them — drawn from lessons and chunk evidence alike. A plan step memory confirms still inherits the convention's concrete specifics as requirements.
- **Reconcile every recommendation you produce — not just the Step-0 plan (anti-displacement).** Before stating any option, lever, or recommendation the work generated, reconcile it against the ASKED TASK and decisions already in context. Substituting a prerequisite, a better test, or a "more rigorous foundation" as the thing to do *first* FEELS like diligence but IS relitigating the settled task. Reasoning already in context is not new evidence — reconcile against prior OUTCOMES, not mere mentions. Default to the asked task; deviate only by (a) naming a genuinely NEW fact and (b) stating you are reversing direction. If a recommendation here is new (not in the Step-0 plan), STOP — complete Step 3.5 before this reply is shown to the user.
- **Surface contradictions inline** where they affect the action, as prose.
- **No payload dumps** — never re-emit YAML, paste whole chunks, or list raw scores/paths.
- **Length:** as long as honesty about the plan requires; if memory was silent on everything, one sentence.

### Step 3.5 — Re-entry: a recommendation not in the Step-0 plan

Before the Step 3 synthesis is shown to the user: if it is about to ship a recommendation, lever, or
approach — named as the thing to do — that does **not** appear in the Step-0 plan (it was conceived
during the work), run ONE more query first, keyed to the recommendation itself, not the original ask:

```bash
engram query --lazy-chunks \
  --phrase "<the recommendation, in its own words>" \
  --phrase "<the recommendation> rolled back rejected not worth it superseded" \
  --phrase "<the recommendation> tried measured outcome"
```

Apply Step 2.5B's recency weight to what returns. A note asserting this approach was
closed/rolled-back/rejected/superseded is a **contradiction to surface and honor**: acknowledge the
prior attempt and its outcome, then drop the recommendation or justify revisiting it against NEW
evidence. This is a new vault query, not the in-context reconcile in Step 3 — that bullet checks
reasoning already in your context; this step checks the vault for evidence outside it. If you cite
a note this query surfaced, include it in the Step 2.7 `engram activate` call (run it again if 2.7
already ran). Runs in both modes (a query is a read; glance keeps it). Skipping the re-entry when
the trigger fires is forbidden.

### Step 4 — Persist the reasoned conclusion (linked to the inputs that produced it)

> **[glance: SKIP Step 4 — write side. Escalate to `deep` if this decision is worth crystallizing.]**

When your closing synthesis reaches a **sound, non-trivial conclusion that no existing note states** —
something a future session (or a *less capable model* that can't re-derive it) would want, and that you
or a human may later **inspect or correct** — crystallize it. Reasoning that is never written down
evaporates; this records the *outcome* and grows the web.

Hand ONE synthesis note per conclusion to the **write-memory** skill (kind=fact or feedback, per the conclusion's shape):

- **The note IS the conclusion** (the reasoned lesson), phrased as such — not a restatement of an input.
- **Certainty by inference mode:** deduction → state it as following necessarily; **abduction / induction
  → mark it _probable / best-explanation / defeasible_**, never as certain. (Note 69: a non-truth-
  preserving inference is a hypothesis, not a fact.)
- **Mark it as derived** in `--source`, e.g. `--source "synthesis (abduction) from recalled memory"`, so
  a human or a weaker model can tell it is a reasoned conclusion to review — not a primitive fact.
- **If the synthesis conclusion CORRECTS, narrows, or refutes an existing surfaced note**, include
  the superseded note's basename, type (`updates|narrows|refutes`), and claim in the write-memory
  handoff — the binary maintains the inverse automatically. Otherwise no link ritual is needed; the
  binary's vocab-tag assignment connects the new note to related notes structurally. Do not
  hand-author wikilinks for structural linking.

**Gate — do not rot the vault (notes 68/69):** persist ONLY conclusions you judge sound. If it is a
hunch, you'd hedge below "probable", or it merely re-aggregates one note, do NOT persist. One synthesis
note per conclusion; link all of its inputs.

**After the synthesis note: if the synthesis body contains ≥1 `[[full-basename]]` wikilink,
ALSO invoke the write-memory skill** with kind=qa — verbatim question, the synthesis conclusion
as the answer, certainty matching the synthesis note's label, contributors = the wikilink
basenames, source "recall Step 4, session <date>".

Contributors are auto-extracted from the `[[full-basename]]` wikilinks you already wrote in the
synthesis body — do NOT free-list ("what notes did you use?"). If the synthesis body contains no
wikilinks, skip the QA capture (D2 bar: ≥1 citation required).

## Red flags — STOP and re-read

| Sign you're off-script | What you should be doing |
| --- | --- |
| You never printed Step 0 | Back up — the skill is a no-op without it |
| You skipped the Step 0.5 sweep | It costs seconds and keeps memory current |
| `--vault` or `--chunks-dir` on the query | `engram query --phrase ...` only — the binary always runs the unified D1 clustering and emits `candidate_l2s` |
| Separate query calls per phrase | One call, repeatable `--phrase` flags |
| You quoted chunks wholesale into the reply | Extract the principle a chunk evidences; paraphrase |
| You dispatched cluster-synthesis subagents | Gone — Step 2.5 crystallizes inline from the payload's clusters |
| You judged coverage before reading the candidate content (now inline in `candidate_l2s`) | Read first — cosine alone cannot decide coverage |
| You applied a cosine threshold to decide covered/near/absent | Coverage is agent-judged from content; cosine only nominates candidates |
| A candidate matching only the superseded content → you marked it "covered" | Apply the recency weight first; a candidate that misses the conflict is "near" |
| You wrote two notes (a fact AND a feedback) for one cluster | One representative note per cluster — pick the right kind |
| You called `engram learn --target` to update a note in place | Updates use `engram amend`; `engram learn` is create-only |
| A `≥0.95` cluster → you activated without reading the candidates | Read first; high cosine nominates, it does not decide |
| You called `engram show` on a note already in `items[]` | NOTE members in `items[]` carry `content` — use it directly. CHUNK items carry no content under `--lazy-chunks` (`budget.lazy_chunks: true`) — `engram show-chunk <source#anchor>` to read their evidence. |
| You assumed a chunk's content is inline and skipped its evidence | Under `--lazy-chunks` chunks carry path/source only — `engram show-chunk <source#anchor>` on-demand before judging coverage |
| You grouped chunks by eye instead of using the payload's clusters | The binary's k-means grouping is the ground truth; read every cluster |
| You skipped Step 2.5 or read chunk-only results as "nothing surfaces" | Processing every cluster IS the step; "nothing surfaces" means an EMPTY payload — clusters present means Step 2.5 runs |
| You activated every returned note | Activate only the notes you actually USED — judged Covered/Near or cited in Step 3 |
| You activated recent-channel items | Chunks are never activated; recent-block items are not activation targets |
| You skipped `engram activate` after drawing on notes | Call it after synthesis — used notes must stay warm or the recency-competition mechanism breaks |
| You're about to write `--relation` or hand-author wikilinks for structural linking | The binary removed `--relation`; vocab tags are automatic; use `--supersedes` only when the note corrects/narrows/refutes a surfaced note |
| You composed an engram learn command yourself at a write site | Write sites hand off to write-memory — parents judge, the worker writes |
| Reply is a memory dump with no plan reference | Restart Step 3: walk the plan and judge each piece |
| You're recommending a prerequisite or better test as the first step, not the asked task | That displacement IS relitigating the settled task — old reasoning isn't new evidence. Do the asked task; displace only on a NEW fact, stated as a reversal |
| You're shipping a recommendation that wasn't in your Step-0 plan, without a lever-keyed re-query | Step 3.5: one query keyed to the recommendation + outcome terms, before it ships |
| You ran the write side (2.5C/Step 4) while in `glance` mode | Glance is read-only w.r.t. vault knowledge — skip the write side; switch to `deep` if you need to crystallize |
| A recency-channel (Channel 2) standard is load-bearing and you stayed in `glance` | Escalate to `deep` — glance surfaces the recent item but won't elevate it to a requirement (C5, #661) |
