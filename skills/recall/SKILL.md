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
4. **Link** — gate and persist precision-checked edges across clusters (default DROP; no topical floods).
5. **Synthesize impact on the plan** — confirm / adjust / contradict / silent, per planned action.

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
  # ... one --phrase per Step 1 phrase (always 10)
```

One call; the binary merges ranking server-side. `engram query` always runs the unified D1
clustering of the matched notes+chunks in one pass and emits `candidate_l2s: [{path, cosine}]`
per cluster. Do NOT collapse phrases, do NOT run per-phrase calls.

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
and notes. Each cluster carries `candidate_l2s: [{path, cosine}]` — the top-5 existing notes
ranked from within that cluster's own matched members (NOT the full vault). A note that did not
match any phrase will never appear as a candidate. A cluster with no note members yields an empty
`candidate_l2s` list; skip to the next cluster when that happens. **Process every cluster.** For
each:

**A. Read candidates and members**

Run `engram show <path>` on every entry in `candidate_l2s` (up to K calls, blocking). For
note-kind cluster members already in the payload's `items[]` list, use their `content` field
directly — no additional `engram show` call needed on already-surfaced members. For chunk
members, the content is NOT in the payload (chunks carry path/source only under `--lazy-chunks`;
the cluster's `members` list never carries content) — `engram show-chunk <source#anchor>` to read the
evidence on-demand. Do not judge coverage before you have read the candidate content.

**B. Apply the recency weight to resolve conflicts**

Evidence **conflicts** when a newer member explicitly negates or reverses an older claim. Reversal
cues: "no longer", "replaced by", "use X not Y", or the same subject+predicate appearing with a
different object in a newer item. When conflict is present: **recent wins**. When no conflict:
treat older and newer evidence as independently valid — do not demote a stable convention merely
because it lacks a recent instance.

**C. Judge coverage against the recency-weighted view — in this order**

| Outcome | Criterion | Action |
| --- | --- | --- |
| **Covered** | A candidate's claim states the cluster's principle with **no material omission** vs the recency-weighted members | `engram amend --target <candidate-path> --activate --relation <new-note-sources> --chunk-source <new-chunk-ids>` — link-enrich only; **do not rewrite content** |
| **Near** | A candidate addresses the same situation but omits ≥ 1 substantive claim the members evidence (judge against the recency-weighted view — a candidate that only matches the superseded content is **near**, not covered) | `engram amend --target <candidate-path> --relation <note-sources> --chunk-source <chunk-ids> --subject ... --predicate ... --object ...` (or `--behavior/--impact/--action`) — re-synthesize content from all members, recency-weighted |
| **Absent** | No candidate addresses the situation | `engram learn fact\|feedback --position top --relation <note-sources> --chunk-source <chunk-ids> --source "<descriptive>" --situation "..." --subject/--predicate/--object (or --behavior/--impact/--action)` |

**One write per cluster; one representative note per cluster.** The representative is always a note
(never a chunk). For `absent`, write exactly one note (fact *or* feedback) covering
the cluster's principle. Do not write one fact and one feedback note for the same cluster.

For `amend` (covered or near), pass one `--relation "<wikilink-target>|<one-line rationale>"`
(repeatable) for every **note** source in the cluster that passes the Step 2.6 precision gate (below)
— not for every co-occurring member. Pass one `--chunk-source <source#anchor>` (repeatable) for every
**chunk** source (provenance, not wikilinks). For `learn`, pass the same flags. The `--source` flag on
`learn` is the human-readable provenance string (unchanged); `--chunk-source` is the chunk-id list (new).

**WAIT for each write before moving to the next cluster.** Writes are blocking and inline — the
note created or updated by one cluster may be a candidate for another.

**As you finish each cluster, record its representative** — the basename you amended (Covered/Near)
or created (Absent). Step 2.6 needs the post-2.5 vault state these writes produce.

### Step 2.6 — Cross-cluster linking (the precision gate, agent-judged)

Gate and persist edges *across* clusters. Recall will form cross-cluster edges whether or not you gate
them — **ungated, it floods** (links a note to property-*mismatched* notes: "needs sweetness" →
"provides texture"). Step 2.6 is the **precision gate**: after the per-cluster loop — and **before**
Step 2.7 activation — run ONE reasoning pass over
**all surfaced note members across clusters** (use their `content` in `items[]`, or the cluster
`members`) and persist a cross-cluster edge ONLY when it passes the gate. Use each note's **basename
exactly as in the payload** (Luhmann-prefixed) as the `--target`/`<B>` — `engram amend` resolves
relation targets strictly against existing basenames and errors on a bare slug.

**A. GENERATE (loose, persists nothing).** Scan members across clusters for *candidate* relationships
— use analogy and "what here relates to what" freely. Proposes only; never writes. (Analogy
generates, it does not justify.)

**B. JUSTIFY (strict — default DROP). Emit one audit line per candidate** so the drop is observable:
`<A> ~ <B> | relation=<…> shared_key=<…> | PERSIST|DROP`. PERSIST only if ALL hold: (1) a
relation TYPE from the menu, (2) the SHARED KEY that passes that relation's test, (3) the key is a
*specific property/entity/effect* — NOT a domain/topic word or generic adjective ("both baking",
"both Go"). Any missing → **DROP. Default is DROP.**

**The hub test (the flood-killer).** A valid shared key pairs notes ~**1:1** — it joins a specific A to
a specific B. If a candidate key would license linking ONE note to MANY others (a **hub** like "the
cake", "properties of a cake", "errors", "Go"), it is topical, not structural → **DROP every edge it
licenses.** Concretely: "cake-needs-sweetness" and "cake-needs-texture" both name the whole "cake," but
"cake" is a hub (it would mesh all six notes) → no edge between them. Their only valid links are
means-ends to their *distinct* providers ("sweetness"→sugar, "texture"→flour) — keys that pair 1:1. If
you find yourself linking three or more notes through one key, that key is a hub: stop and DROP.

| relation (persist if…) | shared-key TEST | direction |
|---|---|---|
| **means-ends / requires-provides** — A needs X, B provides X | the need term in A is the provided effect in B (**same X** — "sweetness"≠"texture") | directed need→provider A→B |
| **causal / transitive** — A causes/depends-on B | A names a cause/dependency whose effect term is B's subject (bridge term in both) | directed cause→effect A→B (chains compose only if each hop passes) |
| **contradiction** — A asserts X, B asserts ¬X | same subject+predicate, opposite/negated object | symmetric A↔B (flag conflict; resolution out of scope) |

**Only these three relations are enabled.** Do NOT link two notes because they share a common whole
("both parts of the cake" — part-whole) or a common schema ("both instances of X" — abstraction):
those keys are always hubs, and they are the flood vector. No edge from them this pass.

**C. PERSIST.** For each surviving link: `engram amend --target <A> --relation "<B>|<TYPE>: <shared
key> — <one-line>"`. The rationale encodes the relation TYPE so the edge is typed. **No `--activate`**
(2.6 writes an edge, it does not mark coverage). Both directions for symmetric relations; one for
directed. **Bound:** the matched set is small per query; propose only the property-sharing candidates
the GENERATE step surfaces — not all pairs.

**Cross-cluster linking is handled by Step 2.6.** Cross-cluster *supersession* — reconciling a
conflict whose evidence did not cosine-cluster — remains deferred: 2.6 *flags* a contradiction (the
contradiction row) but does not *resolve* the supersession.

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

The user sees this. Rules:

- **Open with the count.** One sentence: "Query surfaced N items (K chunks, M notes); crystallized J lessons."
- **Walk the plan from Step 0 in order.** Per numbered action: **confirmed**, **adjusted** (and how), **contradicted**, or **silent**. One short bullet each.
- **Frame load-bearing conventions as requirements.** Lead with "Apply these as requirements:" and list them — drawn from lessons and chunk evidence alike. A plan step memory confirms still inherits the convention's concrete specifics as requirements.
- **Reconcile every recommendation you produce — not just the Step-0 plan (anti-displacement).** Before stating any option, lever, or recommendation the work generated, reconcile it against the ASKED TASK and decisions already in context. Substituting a prerequisite, a better test, or a "more rigorous foundation" as the thing to do *first* FEELS like diligence but IS relitigating the settled task. Reasoning already in context is not new evidence — reconcile against prior OUTCOMES, not mere mentions. Default to the asked task; deviate only by (a) naming a genuinely NEW fact and (b) stating you are reversing direction.
- **Surface contradictions inline** where they affect the action, as prose.
- **No payload dumps** — never re-emit YAML, paste whole chunks, or list raw scores/paths.
- **Length:** as long as honesty about the plan requires; if memory was silent on everything, one sentence.

### Step 4 — Persist the reasoned conclusion (linked to the inputs that produced it)

When your closing synthesis reaches a **sound, non-trivial conclusion that no existing note states** —
something a future session (or a *less capable model* that can't re-derive it) would want, and that you
or a human may later **inspect or correct** — crystallize it. Reasoning that is never written down
evaporates; this records the *outcome* and grows the web.

Write ONE synthesis note for the conclusion with `engram learn fact|feedback`:

- **The note IS the conclusion** (the reasoned lesson), phrased as such — not a restatement of an input.
- **Certainty by inference mode:** deduction → state it as following necessarily; **abduction / induction
  → mark it _probable / best-explanation / defeasible_**, never as certain. (Note 69: a non-truth-
  preserving inference is a hypothesis, not a fact.)
- **Mark it as derived** in `--source`, e.g. `--source "synthesis (abduction) from recalled memory"`, so
  a human or a weaker model can tell it is a reasoned conclusion to review — not a primitive fact.
- **Link every input that contributed, with a _good relationship note_:** one
  `--relation "<input-basename>|<the role that input played in the inference>"` per contributing note —
  the rationale states HOW it fed the conclusion (e.g. `--relation "<…rx9-rejects-old>|rule: RX-9
  silently rejects pre-2021 badges"` and `--relation "<…badge-reader-swap>|condition: RX-9 readers are
  now the installed lobby hardware"`). These typed edges connect the conclusion to its evidence so the
  derivation is auditable and traversable.

**Gate — do not rot the vault (notes 68/69):** persist ONLY conclusions you judge sound. If it is a
hunch, you'd hedge below "probable", or it merely re-aggregates one note, do NOT persist. One synthesis
note per conclusion; link all of its inputs.

## Red flags — STOP and re-read

| Sign you're off-script | What you should be doing |
| --- | --- |
| You never printed Step 0 | Back up — the skill is a no-op without it |
| You skipped the Step 0.5 sweep | It costs seconds and keeps memory current |
| `--vault` or `--chunks-dir` on the query | `engram query --phrase ...` only — the binary always runs the unified D1 clustering and emits `candidate_l2s` |
| Separate query calls per phrase | One call, repeatable `--phrase` flags |
| You quoted chunks wholesale into the reply | Extract the principle a chunk evidences; paraphrase |
| You dispatched cluster-synthesis subagents | Gone — Step 2.5 crystallizes inline from the payload's clusters |
| You judged coverage before reading the candidate content with `engram show` | Read first — cosine alone cannot decide coverage |
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
| You persisted a cross-cluster edge from an analogy with no named shared key | DROP it — analogy generates candidates; only a menu relation type + a passing shared key persists (Step 2.6) |
| You linked "needs X" to a property-mismatched "provides Y" (X≠Y) | The means-ends shared-key test requires the **same** key — DROP the mismatch (this is the flood Step 2.6 exists to stop) |
| You passed `--activate` on a Step 2.6 amend | 2.6 writes an edge, not a coverage mark — `--activate` is Step 2.5's |
| Reply is a memory dump with no plan reference | Restart Step 3: walk the plan and judge each piece |
| You're recommending a prerequisite or better test as the first step, not the asked task | That displacement IS relitigating the settled task — old reasoning isn't new evidence. Do the asked task; displace only on a NEW fact, stated as a reversal |
