# Tiered memory — research log

Started: 2026-05-22. Companion to
`docs/superpowers/specs/2026-05-14-tiered-memory-design.md` (current
design snapshot, now under pressure from research).

This is a **prioritized question board**, not a journal. Each question
carries: why it matters, current hypothesis, what would resolve it,
dependencies, status, priority. Findings live in linked artifacts;
this file is the index of what's open and what's settled.

---

## Current focus list (locked 2026-05-23)

The board below has 28 R-entries, most now settled or dropped. The
active focus is this curated short list. **Additions require a
deliberate decision; the list is not a free-form parking lot.**

| # | Question | Status | Lean |
|---|----------|--------|------|
| **F1** | **R28** — Do episodes earn their place as a third Permanent kind, or do we embed transcript chunks and extend `engram transcript` search instead? Gates v2 spec shape. | `resolved` 2026-05-23 | **Add episodes** as a third kind. |
| **F2** | **R3/R10** — Embedder choice. Pure-Go required (user constraint). | `resolved` 2026-05-24 | **Hugot + GoMLX simplego + Arctic-xs** (default); **MiniLM-L6-v2** fallback if spike fails. Bundled in binary. See spike spec. |
| **F3** | Embedding scope — facts/feedback/episodes always; transcript chunks not needed (F1 = yes makes this unnecessary); MOC content depends on F4. | `resolved` 2026-05-24 | **All three Permanent kinds** (facts, feedback, episodes). MOC handling is F4's concern. |
| **F4** | MOC migration — drop, archive, or convert to fact-kind Permanents on the dropping of MOCs as a kind. | `resolved` 2026-05-24 | **Migrate to facts/feedback** (mixed; per-MOC judgment) with meta-abstraction for multi-principle MOCs. Procedure: `2026-05-24-moc-migration-procedure.md`. |
| **F5** | Link-as-we-go mechanism — sidecar location, weight semantics, retrieval-merge with authored wikilinks. | `resolved` 2026-05-24 | **Drop entirely.** Embeddings already provide clustering signal; F9.1 covers real clustering in v3. |
| **F6** | 3-hop traversal budget — top-k per hop, decay per hop, similarity cutoff. | `resolved` 2026-05-24 | **F6 alone dropped; merged into F9.1 (subgraph clustering).** Link expansion only ships as input to clustering, not as a re-ranking pool. |
| **F7** | **Recall payload return format** — what does engram hand to the LLM? Per-item fields, grouping, role-marking, citation/anchor shape, how the LLM is expected to use it. Subsumes prior R20. | `resolved` 2026-05-24 | YAML; front-loaded full-content items with deduplicated provenance tags; clusters as structural reference. |
| **F8** | **R24/R27** — Proactive feedback surfacing (situation-start hook). Build for v2 or defer? | `resolved` 2026-05-24 | **Drop entirely** (not just defer). Requires harness integration we can't affect. |
| **F9** | **F9.1 (subgraph clustering) promoted to v2** as part of `engram query` design. F9.1 (auto-synthesis) stays deferred — synthesis lives in the consuming skill, not engram. F9.2 (granularity) still deferred. | `resolved` 2026-05-24 (split) | F9.1 clustering in v2; auto-synthesis deferred; F9.2 deferred. |

F1 is the gating decision. F2–F8 are independent enough to settle
in any order, but F1 affects F3. **Working order: F1 → F2 → F3 →
F4 → F5 → F6 → F7 → F8.**

Out of scope for v2 (explicit): R20-as-tier (working memory is the
LLM context; F7 is the discipline we apply to what we emit, not a
new tier).

---

## Pre-analysis notes per focus item

Stashed in-progress thinking. Each subsection is loaded when we
reach its focus item; until then it's reference material that
saved us the work of re-deriving it.

### F8 — proactive feedback surfacing (closed 2026-05-24: drop)

**Decision:** Drop entirely. Not deferred to v3 — dropped.

**Rationale:** Proactive surfacing requires a situation-start
hook in the harness (Claude Code / OpenCode) that fires
engram queries automatically when the agent enters new kinds
of work. Engram can't provide that hook; the harness has to.
We don't control the harness, so building the engram-side
mechanism produces nothing usable.

**What v2 covers instead:** `/please` step 2 (Orient) already
invokes `/recall`, which now (post-v2) returns cluster + hub
structure with situation-cued feedback in the payload. For
bracketed work, that's effectively semi-proactive — feedback
that matches the current task fires automatically at task start.

**The gap:** ad-hoc work outside `/please` orchestration. The
agent has to remember to `/recall`. v2 doesn't fix that; v3
won't either unless harness behavior changes.

**Revisit if:** the harness exposes a SessionStart or task-start
hook we can wire to. Until then, dropped.

### F6 + F9.1 — subgraph clustering at query time (resolved 2026-05-24)

**Decision:** `engram query` returns direct_hits + clusters
(with full member lists + representatives) + hubs (top-5 by
in-degree within the subgraph). No auto-synthesis in engram.
The consuming skill decides whether to synthesize and persist.

**Pipeline:**

1. Embed query → top-k cosine over all stored vectors (direct
   hits).
2. Expand via authored wikilinks 3 hops from direct hits → the
   subgraph (~30-80 notes typical at v2 scale).
3. Auto-k-means (silhouette-selected k from 2-7) over the
   subgraph's embeddings → clusters.
4. Compute in-degree per subgraph note → top-5 hubs.
5. Return payload with all three (direct_hits, clusters,
   hubs).

No registry, no dedup, no staleness tracking, no subagent
dispatch in engram. Engram stays mechanical / pure-Go / offline.

**Algorithm choice:** auto-k-means via silhouette (gonum's
mature k-means run at k=2..7, pick best silhouette). Sidesteps
"pick k" without depending on uncertain pure-Go HDBSCAN ports.

**Hop depth:** 3 (per user choice; subgraph density caps the
practical reach).

**F9 split:**
- F9.1 *clustering* (subgraph-level): in v2.
- F9.1 *auto-synthesis* (engram dispatching subagents to
  synthesize): NOT in v2. Lives in the consuming skill.
- F9.1 *whole-vault clustering*: still v3 (needs scale).
- F9.2 (granularity — block-level / field-level): still
  deferred.

**Why not auto-synthesis in engram:**
- Adds a registry, dedup, staleness, marking, query-as-write
  semantics — significant complexity.
- LLM call from engram or via subagent is a synthesis-decision
  best made with caller's context (skill knows current task,
  user preferences, recall-mirror discipline).
- Append-only with `supersedes` handles cleanup if the caller
  writes a bad synthesis; same as today's `/learn`.

### Synthesis expectations for the consuming skill

The skill that consumes `engram query` output (presumably an
updated `/recall` skill) follows these rules:

**Per-cluster decision (synthesis gate):**

1. Read the cluster's representative (one note, cheap).
2. If the cluster looks coherent and members share a binding
   principle not stated in any single member:
   - Spawn a subagent to read all members (context isolation).
   - Subagent answers: "is there a binding fact or feedback
     principle worth capturing?"
   - If yes → write via `engram learn fact|feedback` with
     `--relation` bullets to each constituent. Same operation
     as F4's MOC migration, applied on demand.
   - If no → do not write; include members as context if
     relevant to the user's query.
3. If the cluster is noise (vocabulary-only, unrelated themes,
   already-stated principle), do nothing.

**Synthesis criteria:**

- Cluster has ≥3 members.
- Binding principle is **not already stated** in any individual
  member.
- Principle passes the recall-mirror test (future query about
  this kind of work would find it).
- Principle is generalizable, not project-specific.

**Synthesis-write discipline** (same as `/learn` today):

- Path A/B/C classification.
- Recall-mirror test on `--situation`.
- `--source "synthesized from cluster, <date>, context: <query>"`.
- `--relation "<luhmann-id>|specific instance of <pattern>"`
  per constituent.
- Fact vs feedback per F4 discipline.

**Hub handling:** read hubs (anchor concepts past-you connected
to repeatedly in this area). Surface them in the skill's output
to the user as "anchor concepts." Do **not** synthesize from
hubs alone — hubs are individual notes, not clusters.

### F5 — link-as-we-go (resolved 2026-05-24: drop)

**Decision:** Drop emergent co-retrieval edges entirely from v2.
No `.engram/emergent.json`, no co-retrieval counter, no
emergent-vs-authored distinction in the retrieval payload.

**Rationale:**

- Embeddings already cluster semantically similar notes; co-
  retrieval is a derivative, slower signal.
- The non-redundant value (catching usage patterns the
  embedder misses) is subtle and unproven.
- F9.1 (deferred to v3) provides real clustering — a stronger
  mechanism than emergent edge counts.
- Less complexity in v2 first-slice; more focus on validating
  the embedding pipeline.

**Implications:**

- F6 (3-hop traversal) now operates only on authored wikilinks
  — there are no emergent edges to traverse.
- F7 (recall payload) simplifies: provenance role markers
  only need `direct-hit` / `linked` (authored wikilink hop) /
  `recent`. No `edge_type: emergent` distinction.
- No `.engram/emergent.json` file; no query logging required
  for retroactive population.

**Revisit if:**

- v2 usage reveals specific cases where embeddings miss
  usage-based clusters that emergent edges would catch. Even
  then, F9.1 in v3 is the better mechanism; emergent edges
  remain dropped.

### F4 — MOC migration (resolved 2026-05-24)

**Decision:** Migrate the 25 existing MOCs into facts and/or
feedback notes via per-MOC LLM judgment. Multi-principle MOCs
get meta-abstraction analysis (write a binding higher-level
note when one emerges). Drop `engram learn moc` from the
binary after migration completes. Original MOC files move to
`_legacy/MOCs/` for one release cycle.

**First-principles framing:**

- MOCs as a writable kind don't add ongoing value beyond what
  query-time LLM synthesis + F9.1 clustering cover.
- The 25 existing MOCs *contain* valuable synthesis content
  (LLM-voiced framing prose stating cross-cutting principles).
- Freezing them in place is a cop-out. The principled move:
  preserve the content as proper fact/feedback notes; drop the
  MOC mechanism entirely.

**Fact-vs-feedback nuance:**

Earlier framing assumed MOCs → facts only. Wrong. MOC framing
prose typically contains both:
- *Statements of how things are* → fact
- *Statements of what to do (or not do) differently* → feedback

A single MOC can produce multiple notes of mixed kinds. Per-MOC
judgment decides which is which.

**Meta-abstraction:**

For MOCs that split into multiple principles, examine whether a
higher-level abstraction binds them. The MOC's framing often
names this explicitly (e.g., MOC/66 names "dilution moves" as
the pattern across three specific dilution failures). When a
binding abstraction is real, write it as its own note;
constituents relate to it as instances.

**Connection to F9 — validation:**

This exercise *is* the F9 operation done manually. F9
automates: cluster related notes → LLM synthesizes binding
abstraction → write as new note. The MOC migration is one
synthesis pass over each historical cluster. If migration
produces valuable meta-abstractions (likely), F9 has
demonstrated value — the manual exercise proves the automated
operation worth building.

**Execution status:** spec'd, not started. ~4-8 hours of
focused work; 50-100 derived notes expected. Procedure in
`2026-05-24-moc-migration-procedure.md`. Trigger: after v2
spike passes but before shipping v2 publicly.

### F3 — embedding scope (resolved 2026-05-24)

**Decision:** All three Permanent kinds (facts, feedback,
episodes) get embedded. No transcript-chunk embedding. MOC
content disposition deferred to F4.

**Rationale:**
- Facts/feedback are the workhorse retrieval target — must be
  embedded.
- Episodes (F1 = yes) carry project-specific narrative —
  embedding them is the *only* embeddable surface for system-
  keyed queries (per the discussion that led to F1).
- Transcript chunks ruled out by F1's cost analysis (~50–100×
  the embedding cost per `/learn` vs episodes; episodes give
  better narrative output).
- MOC content depends on F4: if MOCs convert to facts during
  migration, they get embedded as facts; if archived, they
  don't get embedded.

**Spike scope:** facts and feedback are the existing 138 notes
in the vault; the spike's UAT case 1 (`engram embed --all`)
backfills all of them. Episodes don't yet exist — they'll
appear after `engram learn episode` ships in v2.

**Block-level / field-level granularity** (per F9.2): out of
scope here. Whole-note embedding for v2 first-slice. Revisit
when v2.1 or when query misses reveal granularity issues.

### F2 — embedder choice (resolved 2026-05-24)

**Decision:** Pure-Go embedder via **Hugot + GoMLX `simplego`
backend**, running **Snowflake-arctic-embed-xs** as the default
model with **MiniLM-L6-v2** as the verified fallback.

**Why pure-Go (not Voyage/OpenAI):** user constraint — no
network calls from engram if at all possible. API embedders
ruled out by that constraint, regardless of cost or quality
advantages.

**Why Arctic-xs over MiniLM-L6-v2 as default:**

- 22 MB (same size as MiniLM-L6); same `simplego` op coverage
  (`ReduceL2` added explicitly for this family).
- 512-token context (vs MiniLM-L6's 256) — engram's 200–500-word
  notes (~250–625 tokens) often exceed MiniLM-L6's window;
  Arctic-xs handles them whole.
- ~62 MTEB avg vs ~57 — meaningful improvement at the small-
  model end of the curve.
- Both untested end-to-end in Hugot+simplego — Arctic-xs gets
  the *should-work* with the strongest theoretical case;
  MiniLM-L6 is the only *verified* model. Spike resolves which
  ships.

**Spike spec:**
[`2026-05-24-engram-query-spike.md`](2026-05-24-engram-query-spike.md).
13 UAT cases, including a reference-parity gate (case 13) that
decides Arctic-xs vs MiniLM-L6. Either outcome lands a working
`engram query` command — only the embedder changes.

**Settled decisions during F2:**

1. Content hash scope → **body only** (frontmatter changes
   don't trigger re-embed).
2. Model file location → **bundled in binary**. Binary grows
   ~10MB → ~30MB. Engram works on install with no model
   download step.
3. Missing-embeddings query behavior → **error explicitly**
   ("run `engram embed --all`"); no implicit slow first query.
4. Auto-embed failure in `engram learn` → **warn-and-proceed**.
   The Luhmann write is atomic; embedding is a separate step
   that can fail without losing the note.
5. Default `--limit` for query → **20** (matches `engram recall
   --recent`).

**BM25 deferral confirmed:** during F2 it was clarified that
`engram recall` today does **not** do BM25 or any text search —
the binary returns paths (anchors / recent / follow), the LLM
scores. Adding BM25 alongside embeddings is a real choice;
deferred to v3 unless query misses on lexical-strong queries
become an observed pain.

**Out of scope of v2 (from F2 discussion):**

- Voyage / OpenAI API embedders.
- ONNX-with-CGO path (`fastembed-go` archived; `yalue/onnxruntime_go`
  requires native libs — kills single-binary).
- Ollama opt-in.
- Cross-encoder rerankers (no pure-Go path).
- BM25 (deferred to v3).

### F9 — clustering + synthesis + granularity (deferred 2026-05-24)

**Status:** deferred to v3. Tracked here so the embedding work in v2
keeps a clear path to enabling clustering later.

**Two related questions parked here:**

#### F9.1 — Clustering + bottom-up synthesis

Embeddings enable more than k-NN search. Clustering (k-means,
HDBSCAN, agglomerative) groups vectors by proximity. Paired with
LLM synthesis per cluster, this is **bottom-up pattern
discovery** — the LLM extracts patterns from groups of notes
that share an embedding neighborhood, finding principles the
write-time agent didn't articulate explicitly.

**Pipeline shape:**

```
Stored vectors → clustering algorithm → groups of related notes
                                       ↓
                                    LLM (in skill) reads each cluster
                                       ↓
                                    Generated pattern / MOC-equivalent
```

Binary does clustering (cheap, deterministic, offline). Skill
does LLM synthesis (expensive, deliberate). Offline-recall rule
preserved.

**Why defer to v3:**

- Vault size: at 138 notes, clustering yields weak groups. Need
  ~500+ notes for cluster quality; ~1000+ for reliable patterns.
- Stability: adding notes shifts cluster boundaries (same
  problem L3 regeneration had).
- False-positive patterns: embedders cluster on vocabulary
  overlap, not semantic equivalence. LLM synthesis can be
  misled.
- V2 value (semantic recall via `engram query`) ships without
  this; the value compounds as the vault grows.

**Real differentiation from Smart Connections:** they
explicitly do not cluster and do not auto-synthesize across
groups. F9 is a genuine engram-unique capability if built.

**Validation from F4 (2026-05-24):** the MOC migration spec'd
in F4 *is* F9 done manually. Each MOC is an early hand-curated
cluster; migrating it means extracting the binding abstraction
into a fact/feedback note via LLM judgment. F9 automates this
operation across embedding-detected clusters. The manual
exercise will demonstrate whether the automated version is
worth building — if MOC migration produces valuable meta-
abstractions (highly likely), F9 has demonstrated value.

**Cost when shipped:**

- Clustering library (gonum k-means; pure-Go HDBSCAN options
  exist but less mature) — ~50–150 LOC of wiring.
- New command: `engram cluster` (or `engram discover` /
  `engram patterns`).
- Skill side: read cluster output, ask LLM to synthesize each
  cluster, surface as markdown artifacts.
- Optional persistence: cluster assignments as ephemeral
  recompute vs stored for stability.

#### F9.2 — Embedding granularity (block-level vs whole-note vs field-level)

Current plan: whole-note embedding for v2 first-slice (one
vector per note). **Smart Connections embeds at the block level
too** — per-heading sub-note chunks (toggleable in their
settings). Verified by reading their docs.

Block-level embedding implications for engram:

- **Longer notes** (especially episodes — narrative, multi-
  topic) benefit because querying for a specific system or task
  finds the relevant section, not a diluted whole-note average.
- **Atomic short notes** (facts/feedback, ~200–500 words) chunk
  to themselves; block-level reduces to whole-note for these.
- **Structured fields** (facts: situation/subject/predicate/
  object; feedback: situation/behavior/impact/action) might
  render as distinct blocks in the markdown — if so, block-
  level embedding gives field-level granularity for free.

**Open question to revisit during v2.1 (or as soon as the v2
first-slice exposes a query miss caused by granularity):**

- Do facts/feedback render with field headings or as flat
  prose? (Needs verification against actual `engram learn`
  output.)
- If field headings: block-level subsumes field-level. One
  mechanism, two benefits.
- If flat prose: block-level still helps episodes; field-level
  is a separate optimization for facts/feedback (embed
  `situation` separately as the primary retrieval target).

**Connection to F9.1:** if we cluster on block-level vectors
instead of whole-note vectors, cluster granularity is finer.
Probably better signal at the cost of more vectors to cluster
over. Worth experimenting when F9.1 ships.

**Don't decide now — verify the rendering when v2 implementation
starts, and pick granularity based on what's there.**

#### Discipline tension to revisit

Clustering's value scales with the bottom-up philosophy:
capture more raw context (including project-specific stuff);
let clustering surface patterns. That conflicts with `/learn`'s
current recall-mirror discipline (no project names in
situations, must be abstract principle).

Resolutions when F9 lands:

- **Strict discipline preserved:** episodes carry project-
  specific narrative; facts/feedback stay abstract; clustering
  finds patterns within and across.
- **Discipline relaxed:** facts/feedback can be project-named;
  clustering does more abstraction work; vault is less directly
  retrievable in the interim until clusters are synthesized.

Lean: keep strict for v2 (episodes do the project-named work),
relax later only if the synthesis output shows it's needed.

---

### F1 — episodes earn their place (resolved 2026-05-23)

**Decision:** Add episodes as a third Permanent kind alongside
facts and feedback. Drop the alternative (transcript-only with
embedded chunks).

**Why (the cost analysis flipped the lean):**

Initial lean was "don't add episodes — embed transcripts
instead." That assumed transcript embedding was cheap. It isn't:
~50–100 embeddings per session (one per chunk) vs. 1 per
episode. Episodes are ~50–100× cheaper to embed and store.

Marginal cost at `/learn` is small: the LLM is already
synthesizing the transcript to identify facts and feedback.
Adding "and also emit a one-paragraph episode summary" is
incremental.

Episodes give qualitatively better output for narrative
queries:

- "What did I do this week" returns prose, not chunks.
- "Find the work that led to decision Y" — episodes name
  decisions; transcripts have them diffuse.
- Pattern-recognition over time is tractable; chunk-aggregation
  isn't.

The motivating use case (`a22ad7f7`-style recovery) works
either way. Episodes win on the introspection use cases.

**Sub-decisions deferred to v2 spec (sketch):**

- **Storage location:** lean is same `Permanent/` dir with
  `kind: episode` in frontmatter. Alternative: separate
  `Episodes/` dir. Decide during v2 spec.
- **Discipline:** episodes are narrative, not principle-stated.
  Project names, dates, "I did X" framing all OK — relaxed from
  the path A/B/C / recall-mirror rules that govern facts and
  feedback. Need a separate discipline doc.
- **Schema fields:** situation, summary, outcomes, provenance
  (sessions + transcript range), related. Finalize during spec.
- **Command:** `engram learn episode` with flags for the schema
  fields above.
- **Wikilinking:** episodes link to facts/feedback they
  spawned, with per-link rationale. Backlinks from facts/
  feedback to their originating episode come from those notes'
  `Related to:` bullets.

**Downstream effects:**

- F3 (embedding scope) simplifies: facts/feedback/episodes all
  get embedded; no transcript-chunk embedding needed.
- F7 (recall payload) gains `kind: episode` as a third value
  in the per-item record; provenance role marking includes
  "episode-derived" as a possible value.
- `/learn` skill SKILL.md gains an episode-writing section
  alongside facts/feedback.
- Migration: existing sessions get no backfill — episodes start
  empty and accumulate forward.

### F7 — recall payload return format (resolved 2026-05-24)

**Decision:** YAML payload from `engram query`. Front-loaded
full-content items with deduplicated provenance tags; clusters
as structural reference; no separate hubs section.

**Final payload shape:**

```yaml
version: 1
query: "memory architecture"

items:                              # front-loaded full content, dedup'd
  - path: Permanent/65...
    kind: fact
    score: 0.85
    provenances: [direct, cluster_rep, hub]
    cluster_id: 0                   # iff cluster_rep in provenances
    in_degree: 9                    # iff hub in provenances
    content: |
      <full text of .md>
  - path: ...
  # ... ordered by:
  #   1. provenance count descending (more provenances = higher)
  #   2. highest-priority provenance within set (direct > cluster_rep > hub)
  #   3. score descending

clusters:                           # structural reference
  - id: 0
    size: 12
    silhouette: 0.43
    members:
      - path: Permanent/65...
        score: 0.85
        is_representative: true
      - path: Permanent/7...
        score: 0.71
      - path: ...
        score: ...                  # non-rep members: path + score only
  - id: 1
    ...

budget:
  subgraph_size: 60
  hops_traversed: 3
  clusters_found: 3
  hubs_returned: 5
  direct_hits_returned: 20
  items_with_full_content: 28
  limit: 20
```

**Key rules:**

- **Format:** YAML only. No JSON or paths-only flags in v2.
- **Deduplication:** a path appears once in `items` regardless of
  how many roles it fills. `provenances` lists the roles (any
  non-empty subset of `{direct, cluster_rep, hub}`).
- **Role-specific fields:** `cluster_id` appears iff the item is
  a cluster_rep; `in_degree` appears iff it's a hub.
- **Full-content scope:** every direct hit, every cluster
  representative, and every hub gets `content: <full .md text>`.
  Non-rep cluster members get path + score only, in
  `clusters.members`.
- **Ordering inside `items`:**
  1. Provenance count descending (3 > 2 > 1). More provenances
     wins regardless of priority.
  2. Highest-priority provenance within the set as tiebreak
     (direct > cluster_rep > hub).
  3. Score descending as final tiebreak.
- **Cluster representative:** marked inline in
  `clusters.members` with `is_representative: true`. Not a
  separate field.
- **Hubs:** no separate top-level section. Found by filtering
  `items` on `provenances` containing `hub`.
- **`items.content`** is the full text of the `.md` file
  (frontmatter + body). Consumer parses what it needs.
  Typical item ~1 KB; 25-30 items per response ~25-30 KB
  total — comfortable for any modern LLM context.
- **`clusters.members.content`** intentionally absent. Consumer
  reads non-rep members by path lookup if interested.

**R20 fully subsumed:** the "role marking" hypothesized in R20
is satisfied by `provenances`. Within-session decay /
just-retrieved-vs-stale is out of v2 scope — possibly v3 if
needed.

### F7 — recall payload return format (pre-analysis 2026-05-23 — superseded by resolution above)

Today the binary returns a list of vault-relative paths
(`Permanent/<id>.<slug>.md`); the skill reads each file inline.
That's the entire format. It works, but pushes all the "what's
this and why is it here?" decision-making onto the LLM.

**The full question splits into five sub-questions:**

1. **What's the unit?** Path (today), full note body, chunk, or
   structured record with fields? Today's path-list is the
   lowest-fidelity option — fine if the LLM is about to read the
   file anyway, but loses the chance to add metadata.
2. **What fields per item?** Candidates:
   - `path`, `kind` (fact/feedback/episode if F1 lands), `score`
   - `situation` field, `slug`
   - `provenance` — direct hit vs. linked via `<edge>` from
     `<other-item>`
   - `co-retrieved-with` — counts from emergent edges (F5)
   - `confidence` / `age`
   - Excerpt vs. full body
3. **What's the grouping/ordering?**
   - Flat ranked list (today).
   - Bucketed by kind (facts / feedback / episodes).
   - Bucketed by provenance (direct / linked / recent).
   - Ranked but with role tags.
   - Grouping shapes how the LLM attends. Flat invites first-N-
     matters; bucketed-by-provenance signals "direct hits are
     load-bearing; linked ones are scaffolding."
4. **What's the citation/anchor shape for synthesis?** When the
   LLM writes a reply based on recall, how does it cite?
   Wikilink-by-ID? Path? A short tag the binary returns?
   Citation discipline is what keeps R22 (reconstructive
   retrieval) honest.
5. **How should the LLM use it?** Format and consuming-
   discipline are coupled. A richer return format means a
   richer reading discipline in `/recall` SKILL.md.

**Lean (subject to revision when F7 lands):** structured
JSON or YAML payload (not just paths). Per-item:

```yaml
- path: Permanent/1a3.2026-05-22.foo-bar.md
  kind: fact            # fact | feedback | episode (if F1=yes)
  score: 0.87           # composite: semantic + lexical + recency
  provenance:
    role: direct-hit    # direct-hit | linked | recent
    via: null           # or {path, edge-type, hop} when linked
  excerpt: "..."        # 1-2 sentences from the body, for triage
```

Top-level:

```yaml
direct_hits:   [...]   # ranked
linked_hits:   [...]   # ranked, with via:{} provenance
recent:        [...]   # last-N by mtime
budget:
  surfaced: 23
  read: 17
  remaining: 77
```

**Three properties this buys:**

- **Triage without reads** — LLM decides which to read fully
  from excerpt + score + role.
- **Honest reconstruction** — cites by `path`; the `via` chain
  shows what was actually relied on.
- **Visible budget** — F6 budgets surface to the LLM, not hide.

**Costs:** real return-format spec; small bump in skill
complexity; obligation to keep the format stable (or versioned).

**Sub-decisions still open within F7:**

- JSON vs YAML — YAML reads cleaner in transcripts; JSON parses
  simpler. Lean YAML.
- Excerpt length — 1-2 sentences vs. `--situation` field vs.
  configurable. Lean: `--situation` if present, else first
  sentence.
- `/recall` SKILL.md changes — cascade loop becomes "consume
  payload, decide what to read fully" instead of "read
  everything." Smaller diff than it sounds.
- Backwards compatibility — change format and skill in the same
  release; today's path-list output has only the current
  `/recall` consumer.

**Open ingredients from R20 to carry in:** within-session decay,
recency weighting, the distinction between "just retrieved,
unused" and "retrieved earlier, possibly stale." May or may not
make it into v2; document explicitly when we get to F7.

---

## Board

### Foundational (P0 — block re-deciding the shape)

#### R1. Is the tiered/hourglass topology right?

- **Why it matters:** Everything else in the design assumes a tiered
  shape with a narrow L2 waist. If the right shape is different
  (2 tiers per CLS, organic graph, narrowing-top), the rest of the
  spec is moot.
- **Hypothesis:** Pressured by R2 — CLS argues for two systems
  (episodic + semantic), not four tiers. L0/L1 may collapse;
  L2/L3 may collapse. Four tiers may be visual neatness over
  ontological substance.
- **Resolution:** Sketch the 2-tier alternative side-by-side with
  the current 4-tier; pick one with stated reasoning.
- **Deps:** R2 (informs), R4/R5 (empirical evidence on what tier
  boundaries the agent actually uses).
- **Status:** `open` — newly pressured by R2 findings.
- **Priority:** P0.

#### R2. What does human memory research say that should constrain our design?

- **Why it matters:** We risk reinventing structure that cognitive
  science has already characterized — or worse, forcing structure
  human memory doesn't have.
- **Hypothesis:** N/A — empirical.
- **Resolution:** Lit summary with takeaways mapped to design
  choices.
- **Deps:** None.
- **Status:** `exploring` — first pass complete.
- **Artifact:** [`2026-05-22-human-memory-literature-summary.md`](2026-05-22-human-memory-literature-summary.md)
- **Findings that change the board:**
  1. **CLS argues for two stores, not four** (McClelland et al.
     1995; Kumaran/Hassabis/McClelland 2016). Spawns R15.
  2. **Forgetting is functional, append-only fights biology**
     (Anderson RIF 2003; Hardt/Nader/Nadel 2013). Refines R8 →
     becomes R16 (concrete: confidence decay + retrieval filter).
  3. **Schemas drive consolidation, not the reverse** (Tse et al.
     2007; Gilboa & Marlatte 2017). Inverts L3↔L2 direction.
     Spawns R17.
  4. **Edge types with empirical support: temporal, contextual,
     semantic, contradictory, causal** (in that order). Refines R5
     → becomes R18.
  5. **Salience = prediction error + novelty + affect** (Schultz
     1998; Lisman & Grace 2005). Spawns R19.
  6. **No working-memory analog in Engram** (Baddeley 2000). Spawns
     R20.
  7. **Sleep/replay does consolidation + downscaling** (Klinzing
     et al. 2019). Spawns R21.
  8. **Retrieval is reconstructive, not lookup** (Schacter; Tulving
     & Thomson 1973). Spawns R22.
- **Priority:** P0 — done as input; outputs now drive other Rs.

#### R3. Should the "no CGO" rule be relaxed to admit ONNX-class embedders/rerankers?

- **Why it matters:** Smart Connections (obsidian-smart-connections)
  demonstrates what local transformer-quality semantic search
  enables for memory. Pure-Go embedding has a ceiling.
- **Hypothesis:** Relax the rule for embedders/rerankers
  specifically; keep "no LLM in binary at read time" intact.
- **Resolution:** Cost/benefit doc; decision.
- **Deps:** Informed by R5/R22 (does retrieval quality actually
  bottleneck on embedder?).
- **Status:** `open`.
- **Priority:** P0.

### Empirical (P0 — cheap, informs everything)

#### R4. Why does recall stop short of the 100-memory limit?

- **Why it matters:** If the cap is model fatigue, design more
  layers / richer summarization. If it's graph sparsity, design
  denser links. We don't know which.
- **Hypothesis:** Mix — agent stops when marginal relevance falls
  off, which happens earlier in a sparse graph.
- **Resolution:** Structured experiment varying graph density and
  link-type richness; measure depth reached and load-bearing-note
  surface rate.
- **Deps:** None.
- **Status:** `open`.
- **Priority:** P0.

#### R5. What link/relationship types will the agent actually follow during recall?

- **Why it matters:** Authoring edges that don't get followed is
  waste; missing edges that would have helped is silent failure.
- **Hypothesis:** Pressured by R2 — temporal, contextual,
  contradictory, causal in approximately that order. Semantic
  similarity should be computed at retrieval, not stored.
- **Resolution:** Ablation experiment — strip link types one at a
  time, measure recall quality.
- **Deps:** R4 (shares experimental infra).
- **Status:** `open`.
- **Priority:** P0.

### Structural (P1 — depend on R1/R2)

#### R6. How many layers, and what's each layer's job?

- **Why it matters:** Tier count is the most visible architectural
  decision and the hardest to change later.
- **Hypothesis:** Two stores (CLS) with explicit "raw" and
  "consolidated" representations within each is the literature-
  aligned baseline. The 4-tier sketch may collapse to 2.
- **Resolution:** Design sketch per option; judge against R2 and
  R4/R5 results.
- **Deps:** R1, R2, R4, R5.
- **Status:** `open`.
- **Priority:** P1.

#### R7. How do we preserve history without re-introducing substrate discard?

- **Why it matters:** User concern. Original substrate (session
  JSONL) was discarded by the original design; the motivating
  case (`a22ad7f7`, `677d4acf`) proved that was a mistake.
- **Hypothesis:** L0 stays as a provenance index over external
  JSONL; transcripts are never copied but always reachable.
- **Resolution:** Explicit history-preservation policy per tier.
- **Deps:** R1, R6.
- **Status:** `open`.
- **Priority:** P1.

#### R8. What curation mechanisms exist beyond write-time gating?

- **Why it matters:** User concern about lack of curation. Without
  curation the vault drifts toward unfindable.
- **Hypothesis:** Subsumed by R16 (confidence decay + retrieval
  filter) and R21 (maintenance pass).
- **Resolution:** See R16, R21.
- **Deps:** Spawned R16, R21.
- **Status:** `decomposed` → see R16, R21.
- **Priority:** P1.

#### R9. Should L3 dimensions be constrained or organic?

- **Why it matters:** Constrained dimensions risk tags-in-disguise;
  organic dimensions risk noise + tag-soup.
- **Hypothesis:** Organic with framing-prose requirement (per
  Permanent/4a). MOC = framing prose + sentence-explained links.
- **Resolution:** Prototype both on existing vault; compare.
- **Deps:** R1, R6.
- **Status:** `open`.
- **Priority:** P1.

### Substrate (P1 — depends on R3)

#### R10. If CGO is relaxed, what's the embedder + reranker stack?

- **Why it matters:** Concrete tech choice unlocks (or blocks)
  retrieval quality.
- **Hypothesis:** ONNX-runtime + a small reranker (BGE-reranker-v2
  class) is the Smart Connections analog.
- **Resolution:** Spike + benchmark vs. current Voyage-only path.
- **Deps:** R3.
- **Status:** `open`.
- **Priority:** P1.

#### R11. What does Smart Connections do that we don't?

- **Why it matters:** Concrete reference architecture exists; learn
  from it before reinventing.
- **Hypothesis:** Per-block embeddings + similarity-driven sidebar
  + local model option are the load-bearing features.
- **Resolution:** Feature inventory + adopt/reject decisions.
- **Deps:** None.
- **Status:** `open`.
- **Priority:** P1.

### Spawned by R2 lit findings (P1 — refine the design)

#### R15. How many LTM *representations* does engram need? (refined from "collapse to two tiers")

- **Why it matters:** CLS argues for two stores, but the underlying
  argument is about catastrophic interference in neural networks —
  a problem that doesn't exist at engram's file-substrate level.
  The *functional* part of the argument does survive translation,
  but it's about representations, not physical stores.
- **Refined hypothesis (2026-05-22 discussion):**
  - The CLS plasticity-stability argument is **substrate-specific
    to neural networks** and does not bind engram directly. Writing
    a new file doesn't damage existing files; there's no
    catastrophic interference at the file layer.
  - The *functional* claim does survive: episodic and semantic
    queries need different representations because they answer
    different questions.
    - Episodic query: "what did I do last Tuesday with the auth
      middleware?" Needs verbatim, situational, time-bound.
    - Semantic query: "how should I approach auth middleware
      refactors?" Needs abstracted, principle-stated, generalized.
  - You cannot serve both well from the same representation. Same
    content, two access shapes.
  - **Therefore: engram needs (at least) two LTM representations
    (episodic-rich and semantic-distilled).** Whether those are
    two physical stores, one store with two views, or one store
    with two indices is an implementation decision the literature
    does not dictate.
  - The "raw vs. distilled" sub-distinction within each (current
    L0/L1 and L2/L3) is also a representation question, not
    necessarily a tier question — could be one tier with two
    indices/views.
- **Resolution:** Sketch the LTM-representation space — what
  representations are needed, how they relate, what each is
  optimized for. *Then* decide how many physical tiers serve them.
- **Deps:** R1, R4, R5.
- **Status:** `exploring` — first concrete sketch on the table
  (see Synthesis note 2026-05-23).
- **Working sketch (2026-05-23 discussion):**

  | Tier | Content | Operation |
  |------|---------|-----------|
  | L0 | Raw sources, pointers only | Address, don't copy |
  | L1 | Noise-removed extraction | Content-preserving cleanup |
  | L2 | Analyzed by kind (episodes, facts, feedback); searchable temporally + semantically | LLM-mediated interpretation |
  | L3? | Open — see Synthesis 2026-05-23 | — |

  Four query shapes drove the schema:
  - What happened yesterday? (episodes × temporal)
  - What did we learn yesterday? (facts/feedback × temporal)
  - What have we done with X? (episodes × semantic)
  - What have we learned about X? (facts/feedback × semantic)

  Departures from current spec: L1 becomes content-extraction
  (not situational framing in agent voice); L2 split by content
  *kind* rather than abstraction level; L3 status open.
- **Open questions raised by the sketch:** see R23, R24, R25 below
  and the Synthesis 2026-05-23 entry.
- **Priority:** P1.

#### R16. Add confidence/strength decay + retrieval-time filter?

- **Why it matters:** Append-only fights the literature's
  unanimous finding that forgetting is functional and adaptive.
- **Hypothesis:** Keep append-only at substrate; add
  `confidence` and `last_retrieved` fields; retrieval applies a
  soft filter. Ebbinghaus-style decay with boost-on-retrieval.
- **Resolution:** Decay function design; prototype; eval.
- **Deps:** R6 (depends on final tier count).
- **Status:** `open`.
- **Priority:** P1.

#### R17. Should L3 drive L2 ingestion (schema-congruence gate)?

- **Why it matters:** Schema-assimilation work shows schemas drive
  consolidation, not the reverse. Current design has L3 derived
  from L2, which inverts the biological direction.
- **Hypothesis:** Yes — incoming candidate L2 atoms are scored
  against the current MOC graph for congruence/incongruence. Both
  ends of the U-shape (highly-congruent, highly-incongruent) are
  preferentially admitted.
- **Resolution:** Mechanism design (cheap proxy: embedding
  distance to MOC centroids?); prototype; eval.
- **Deps:** R1, R6.
- **Status:** `open`.
- **Priority:** P1.

#### R18. Adopt temporal + causal edges; drop authored "similar"?

- **Why it matters:** R2 found these edge types are well-supported
  empirically; "similar" should be a retrieval-time computation,
  not an authored link.
- **Hypothesis:** Add temporal (follows, precedes, session-of) and
  causal (caused-by, enables, blocks); remove any authored
  similarity edges; keep `contradicts:` and `supersedes:`.
- **Resolution:** Schema update + L1/L2 emission rules.
- **Deps:** R5 (validate via ablation), R6.
- **Status:** `open`.
- **Priority:** P1.

#### R19. Explicit salience gate at L0→L1?

- **Why it matters:** Not every session produces a worthwhile L1
  segment. Without a gate, L1 grows unboundedly and dilutes
  retrieval.
- **Hypothesis:** Multi-signal gate: prediction error vs. current
  L3 MOC graph; novelty (entities/situation absent from existing
  L1/L2); user affect/emphasis. Any one triggers L1 emission.
- **Resolution:** Mechanism design; prototype; eval.
- **Deps:** R6, R17.
- **Status:** `open`.
- **Priority:** P1.

#### R20. Working-memory analog: structure the recall payload (refined; subsumed by F7)

**Status (2026-05-23):** Subsumed into focus list item **F7
(recall payload return format)** at the top of this file. R20's
core finding — "make recall payload structure explicit, since
the LLM context doesn't impose any" — survives; the specific
mechanism (role-marking, ordering, decay-within-session) is now
one ingredient of F7 alongside per-item fields, grouping, and
citation shape. See F7 for the active discussion.

- **Why it matters:** The LLM context window functionally *is*
  working memory — bounded capacity, active manipulation, attention
  control, gates consolidation. But it's *undifferentiated* and the
  LLM has to re-derive structure each turn.
- **Refined hypothesis (2026-05-22 discussion):**
  - R20 is **not** a new tier outside the context window.
  - Where LLM context falls short of Baddeley-style WM:
    - **Undifferentiated.** WM routes input to specialized buffers
      and weights them differently; context is a flat token stream
      where system prompt, user turn, tool output, and retrieved
      memory all share one channel.
    - **No activation gradient.** Human WM retrieval *activates*
      items, which decays and primes related ones. LLM retrieval
      just appends tokens — no priming, no relevance-decay signal.
    - **No within-session selective retention.** WM selectively
      keeps goal-relevant items as it fills; context compaction
      summarizes uniformly.
    - **No role marking.** WM distinguishes "currently being
      considered" from "background." Context doesn't natively
      distinguish "just retrieved, unused" from "retrieved 20
      turns ago, stale."
  - The engram-side intervention is a **design discipline on the
    recall payload**, not a new store:
    - Bounded with a stated rationale (not "as much as fits").
    - Ordered by relevance × recency, not arbitrary.
    - Each item tagged with role (background / just-retrieved /
      contradicting-current / supporting-current).
    - Items marked so the next turn can decay or boost based on
      whether they were referenced.
- **Resolution:** Recall-payload spec section defining the four
  bullets above. No new tier.
- **Deps:** R6 (depends on what LTM emits).
- **Status:** `exploring`.
- **Priority:** P1 — re-prioritized; this is design discipline that
  shapes every recall, not deferred polish.

#### R21. Maintenance/sleep pass — consolidate, replay, downscale?

- **Why it matters:** Sleep-dependent consolidation is the
  mechanism by which episodic→semantic transfer happens and weak
  traces are downscaled. Engram has no analog.
- **Hypothesis:** Scheduled pass: replay (re-rank), consolidate
  (promote L1→L2 / merge), downscale (decay confidence on
  unretrieved). Runs on idle or after batches of L1 emission.
- **Resolution:** Design; trigger choice; eval.
- **Deps:** R6, R16, R17.
- **Status:** `open`.
- **Priority:** P1.

#### R22. Reconstructive retrieval — synthesize a passage from atoms vs. return raw?

- **Why it matters:** Schacter et al. show recall is
  reconstructive, not lookup. The calling agent will reconstruct
  anyway; should engram do it explicitly to ground the
  reconstruction?
- **Hypothesis:** Yes — recall returns an LLM-synthesized passage
  using retrieved atoms as scaffold, with atoms cited inline.
  Confabulation risk managed by citation requirement.
- **Resolution:** Prototype; eval against raw-atom return.
- **Deps:** R3, R6.
- **Status:** `open`.
- **Priority:** P2.

#### R23. When does L1/L2 analysis fire?

- **Status:** `resolved` (2026-05-23e) — `/learn` already does
  this and already brackets `/please`. The trigger exists.
- **Today (verified):** `/learn` runs `engram transcript --mark`
  to fetch transcripts since the last marker, identifies
  candidates via path A/B/C discipline + recall-mirror test,
  categorizes as Feedback or Fact, and writes via `engram learn
  feedback|fact|moc`. `/please` brackets every invocation with
  `/learn` at steps 1 and 7.
- **What changes under the proposed delta:**
  - `/learn` gains episode-writing alongside facts/feedback (if
    R28 lands as "yes, episodes earn their place").
  - `/learn` loses MOC-writing (MOCs dropped).
  - The path A/B/C discipline still applies to facts and
    feedback. Episodes need a different discipline (see R28).
- **Priority:** P1 (was) — closed.

#### R24. How does feedback differ from facts? (refined — *not* procedural)

- **Why it matters:** The L2 split treats feedback as a third
  kind alongside episodes and facts. Original framing called it
  "procedural" — that was wrong on two counts.
- **Correction (2026-05-23):**
  1. CLS does **not** model procedural memory. Procedural lives
     in basal ganglia / cerebellum, outside the
     hippocampal-cortical system CLS describes.
  2. Strict procedural memory is *implicit* and expressed through
     action automaticity. Engram can only retrieve *verbalized*
     rules; it cannot make an LLM's actions automatic. So
     feedback at engram's level is **semantic content**, not
     procedural.
- **Refined hypothesis:** Feedback is semantic content
  distinguished by **situation-cued retrieval**, not topic-cued
  retrieval. The L2 split is about retrieval *shape*, not memory
  *kind*.
  - **Episodes:** topic + temporal cues. "What did we do with X."
  - **Facts:** topic / semantic cues. "What's true about X."
  - **Feedback:** situation cues. "I'm in situation S — what
    applies?"
  - Grounded in encoding-specificity (Tulving & Thomson 1973):
    feedback is encoded with a situation cue, so it must be
    retrieved with one. Topic similarity will miss it.
- **Practical implications:**
  - All three kinds can share L2 substrate (one tier, not three).
  - They diverge in **indexing**: episodes → temporal + entity;
    facts → semantic; feedback → situation-embedding.
  - Feedback fires **proactively at situation start**; episodes
    and facts fire on query.
- **Resolution:** Spec the three retrieval pipelines; validate
  against current auto-memory usage patterns.
- **Deps:** R15, R5.
- **Status:** `open` — framing settled, mechanism to spec.
- **Priority:** P1.

#### R26. Does the LTM design support a research-log-shaped workflow?

- **Why it matters:** This very research log is a working
  prototype of the memory system we're designing. The mapping is
  clean (see Synthesis 2026-05-23c). If the L0/L1/L2 sketch
  can't elegantly support the workflow we're using *right now*
  to design it, that's a strong signal we're wrong.
- **Acceptance check:** The design must support, in observable
  form:
  1. **Query-driven retrieval** — pull relevant prior state on
     demand (we navigate the log by R# / topic / temporal /
     situation).
  2. **Write/extraction without query** — capture state as it
     emerges (synthesis notes, R-spawns, change log entries).
  3. **Consolidation without query** (R21) — restructure across
     R entries when patterns emerge (the R15+R20 reframing was
     consolidation, not retrieval).
  4. **Proactive surfacing** (R24-shape) — bring up applicable
     feedback at the start of a similar situation, not on
     demand. Not yet exercised in this workflow; we rely on the
     user to set context.
  5. **Refinement of stored items** (R16) — R24's "feedback is
     procedural" → "situation-cued semantic" is a live example;
     the mechanism must support this.
  6. **Spawning new items from stored items** — R2 spawned
     R15–R22; R15 spawned R23–R25; R26 spawns from the meta-
     observation. Forward references must be cheap.
- **Hypothesis:** The current sketch supports (1), (2), (5), (6)
  natively. (3) requires R21 to be specified. (4) requires the
  situation-cued retrieval shape from R24 to be wired up.
- **Resolution:** Walk the sketch against the six operations
  above; identify gaps; close them or accept them as scope.
- **Deps:** R15, R16, R21, R24.
- **Status:** `resolved` — walk complete 2026-05-23.
- **Artifact:** [`2026-05-23-r26-acceptance-walk.md`](2026-05-23-r26-acceptance-walk.md)
- **Verdict:** Sketch is **viable but incomplete**. No fatal
  flaws; five mechanisms need sharpening (R16, R19, R21, R22,
  R23) and one new R spawned (R27 — situation-start hook).
- **Key findings:**
  1. Open questions fit as low-confidence facts with an
     "open-work" flag, not a fourth L2 kind. Reuses R16's
     confidence field.
  2. **Proactive surfacing has no hook** — biggest structural
     gap; spawned R27.
  3. Query routing + multi-kind fusion need spec in R15.
  4. Consolidation actions per tier need spec in R21.
  5. R16 remains the largest single unspecified mechanism.
- **Priority:** P1 — acceptance test for the LTM sketch.

#### R25. Does L3 (persistent abstraction layer) still exist?

- **Why it matters:** The L0/L1/L2 sketch stops at L2. The current
  spec has L3 (MOCs, synthesis, patterns-of-facts). Three
  possibilities, each producing a different system:
  - **(a) L3 persists** — MOCs as a mutable, regenerated tier
    above L2. Current spec direction.
  - **(b) L3 dissolves** — no persistent abstraction; synthesis
    happens at retrieval time via R22 (reconstructive recall)
    when search aggregates facts/episodes.
  - **(c) Patterns become a fourth L2 kind** — alongside
    episodes/facts/feedback, with `derives-from:` edges.
- **Hypothesis:** Lean (b) — if R22 lands, persistent MOCs may
  be obviated by recall-time synthesis. But (a) and (c) have
  navigation/browsability benefits that (b) loses.
- **Resolution:** Decide before tier count is finalized in R6.
- **Deps:** R6, R15, R22.
- **Status:** `open`.
- **Priority:** P1.

#### R28. Do episodes earn their place as a third Permanent kind?

- **Why it matters:** Today, `engram learn` writes facts and
  feedback (plus MOCs). There is no per-task narrative kind.
  The convergence proposal adds episodes — but transcripts
  already capture the narrative externally. Episodes would
  duplicate that unless they earn something transcripts don't.
- **What episodes would buy:**
  - Fast temporal scans ("what did I do yesterday") without
    transcript dives.
  - Lighter recall payload for narrative queries.
  - A semantic-embedding target for "what have we done with X"
    queries that transcripts don't expose.
- **What they cost:**
  - New Permanent kind → new schema, new write path, new
    discipline (path A/B/C is principle-shaped, not narrative-
    shaped — see deps).
  - Duplication risk: transcripts already have the content.
  - More `/learn` work per task.
- **Two paths:**
  - **(a) Add episodes.** New `engram learn episode` command.
    Episodes get a narrative-shaped discipline (not principle-
    stated). Indexed temporally + semantically.
  - **(b) Don't add episodes.** Use transcript anchors for
    narrative queries — extend `engram transcript` with date-
    range and topic search, possibly with embeddings over
    transcript chunks.
- **Discriminator:** how often does the user actually want
  narrative recall without wanting principle-stated lessons?
  Empirical question.
- **Hypothesis:** Lean (b) — transcripts already have the
  content; embedding-indexing transcripts is a smaller change
  than introducing a new kind with its own discipline.
- **Resolution:** Decide before the v2 design spec.
- **Deps:** R23 (trigger), R3/R10 (embedding scope — if (b),
  transcripts get embedded).
- **Status:** `open`.
- **Priority:** P0 — gates the v2 spec shape.

#### R27. Situation-start hook + situation definition

- **Why it matters:** R24's proactive surfacing requires a
  "situation start" event the sketch doesn't have. This is the
  biggest single structural gap from the R26 walk.
- **Two coupled questions:**
  1. **What is a "situation"?** Candidates: a task (current
     spec's task-boundary unit), a turn, a tool-call sequence,
     a topic change, a SessionStart event. The definition must
     be operational — something the harness can detect cheaply.
  2. **What hook fires the situation-start probe?** Candidates:
     SessionStart, PreToolUse on selected tools, a per-turn
     lightweight probe, an explicit `engram situation` API.
- **Hypothesis:** A situation is a *task* (matching the current
  spec's L1 task-boundary unit), and the hook is a task-start
  event the active skill emits. Per-turn probing is too
  expensive; SessionStart is too coarse.
- **Resolution:** Spec the hook + the engram API; prototype;
  measure cost per situation.
- **Deps:** R24 (defines the retrieval shape), R15 (defines
  what gets indexed).
- **Status:** `open`.
- **Priority:** P1 — unblocks R24.

### Refinement (P2)

#### R12. Default MOC dimension set

- **Status:** `parked` until R9, R17 settle.

#### R13. L1 segment frequency / shape

- **Status:** `parked` until R6, R19 settle.

#### R14. Migration path from current vault

- **Status:** `parked` until R6 settles.

---

## Dependency graph (after R2)

```
R2 (memory research) ─┬─► R1 (topology) ──┬─► R6 (layer count) ──┬─► R12 (dims)
                      │                    │                       ├─► R13 (L1 shape)
                      │                    │                       └─► R14 (migration)
                      ├─► R15 (CLS 2-tier?) ─┘
                      ├─► R16 (decay + filter)
                      ├─► R17 (schemas drive ingestion)
                      ├─► R18 (edge types)
                      ├─► R19 (salience gate)
                      ├─► R20 (working memory)
                      ├─► R21 (sleep pass)
                      └─► R22 (reconstructive recall)

R3 (CGO rule) ──► R10 (stack) ──► R11 (Smart Connections port)
                                ┘
                          R22 ──┘ (better embedder enables synthesis)

R4 (recall depth)    ─┬─► informs R1, R6, R15
R5 (link ablation)   ─┘
```

---

## Synthesis notes

### 2026-05-23 — First concrete LTM sketch (Joe-proposed)

User proposed a representation sketch oriented around the four
query shapes engram actually serves:

- **L0** — raw sources (transcripts, git logs, web pages,
  markdown). Pointers only, never copied.
- **L1** — content-preserving extraction. Reader-mode HTML,
  `git log --format`, transcript cleanup. Mechanical, reversible.
- **L2** — LLM-mediated analysis into three kinds (episodes,
  facts, feedback), each searchable temporally + semantically.
- **L3** — left open in the sketch; see R25.

Departures from current spec:

- L1 becomes content-extraction, not situational framing in the
  agent's voice. Interpretation moves up to L2.
- L2 splits by content **kind** (episodes / facts / feedback)
  rather than abstraction level (atomic facts vs. principles).
- L3 status is open — possibly dissolved if R22 lands.

Issues raised during critique:

1. **Code is not memory.** Code lives in repos; engram remembers
   things *about* code (decisions, lessons, episodes). C4
   diagrams are derived artifacts, not memory. L0 holds streams
   the agent *processed*, not its working materials.
2. **"Facts evolve through use" is not R20.** It's R16 (decay +
   strengthen on retrieval) plus R21 (maintenance pass) applied
   to persistent L2 entries. R20 — recall payload shape — is a
   separate problem and stays open.
3. **Feedback is procedural, not semantic.** Spawned R24.
4. **L1→L2 analysis trigger is unspecified.** Spawned R23.
5. **L3 status unresolved.** Spawned R25.
6. **L2 evolution mechanism is underspecified.** R16 needs to
   pick a concrete combination (decay, supersede, synthesize-on-
   use) before the sketch is buildable.

Next: pull on one of R23, R24, R25 (the three new questions the
sketch raised), or sharpen R16's mechanism.

### 2026-05-23b — CLS scope correction; R24 refined

During R24 discussion, an earlier framing error was caught: CLS
posits only two systems (episodic + semantic), both declarative.
It does **not** model procedural memory — that lives outside
CLS in basal ganglia / cerebellum (Squire taxonomy).

Implication: calling feedback "procedural" was wrong on two
counts. (1) CLS doesn't authorize the label. (2) Procedural
memory is implicit and expressed via action automaticity; engram
can only retrieve verbalized rules, so feedback at engram's
level is **semantic content**.

The valuable distinction between facts and feedback is
**retrieval cue**, not memory kind:

- Facts → topic-cued retrieval ("what's true about X").
- Feedback → situation-cued retrieval ("I'm in S, what
  applies?"), grounded in encoding specificity.

R24 updated to reflect this. The L2 split (episodes / facts /
feedback) is now framed as a *retrieval-shape* split over shared
semantic substrate, not three kinds of memory.

Broader lesson for the log: CLS is a tighter theory than its
shorthand suggests. Decisions to add memory kinds should be
justified on retrieval-engineering grounds (which engram cares
about) rather than claimed as CLS-authorized.

### 2026-05-23e — Architecture-reality reconciliation

Re-read the actual `/please`, `/learn`, `/recall` skill source +
`internal/update/`. Multiple prior synthesis claims about the
current system were wrong. Corrections:

1. **`engram update` is unrelated to memory.** It refreshes the
   binary via `go install` and copies harness skills/commands
   into install dirs. The transcript-progression mechanism is
   `engram transcript --mark`, invoked by `/learn`.
2. **`/please` already brackets with `/learn`.** Step 1 is
   `/learn` (capture-open), step 7 is `/learn` (capture-close).
   Ingestion is already part of `/please`, not a proposed
   addition.
3. **`/learn` already writes L2-equivalent content.** It
   identifies candidates via path A/B/C discipline + recall-
   mirror test, categorizes as Feedback or Fact (plus MOCs), and
   writes via `engram learn`. The "extract facts and feedback
   from transcripts" loop is the current system, not a future
   one.
4. **`/recall` already does cascade graph traversal.** Anchors +
   `--recent --limit 20` initial frontier, read-and-score until
   100 memories or empty frontier, follow wikilinks from any
   scoring note. Subagents for large frontiers. What's missing:
   embeddings-based semantic NN.
5. **The vault has two dirs: `Permanent/` and `MOCs/`.**
   Permanents hold facts + feedback by kind. Filename:
   `<Luhmann-ID>.YYYY-MM-DD.<slug>.md`. No L0/L1/L2 dirs exist.

**Corrected delta from current → proposed:**

| Area | Today | Proposed |
|------|-------|----------|
| Permanent kinds | feedback, fact (+ moc) | feedback, fact, episode (if R28 = yes); drop moc |
| Retrieval | wikilink cascade + anchors + recent | + embeddings-based semantic NN + temporal index |
| Edges | authored wikilinks only | + emergent co-retrieval edges (sidecar) |
| Cascade | unbudgeted, follow-everything | budgeted 3-hop with decay |
| MOCs | judgment-based MOC writing | dropped; LLM-side synthesis at recall-time |
| `/please` orchestration | 7-step bracket via `/learn` | unchanged |
| `/learn` discipline | path A/B/C + recall-mirror | unchanged for facts/feedback; episodes need new discipline (R28) |
| `engram transcript --mark` | exists | unchanged |

**Settled by reconciliation:**

- R23 (L1→L2 trigger): `/learn` already exists; episode-writing
  joins it if R28 lands.
- R7 (history): transcripts via `engram transcript --mark`
  already preserve history; no new mechanism needed.
- R19 (salience gate): `/learn` already gates via path A/B/C +
  recall-mirror discipline. The gate exists.

**New question raised by reconciliation:** R28 — *do episodes
earn their place?* This is the gating decision for whether the
v2 spec adds a new Permanent kind or just extends transcript
search.

### 2026-05-23d — R26 acceptance walk

Walked the L0/L1/L2 sketch against the six operations R26
defined. Full walk in
[`2026-05-23-r26-acceptance-walk.md`](2026-05-23-r26-acceptance-walk.md).

**Verdict:** sketch is viable but incomplete. No fatal flaws.

**Five mechanisms need sharpening:**
- R16 (confidence/decay/supersedes)
- R19 (salience gate)
- R21 (per-tier consolidation actions + trigger)
- R22 (reconstructive retrieval mechanism)
- R23 (L1→L2 trigger)

**One new R spawned:**
- R27 (situation-start hook + situation definition) — the
  biggest single structural gap, blocks R24.

**Key qualitative findings:**
1. **Open questions fit as low-confidence facts with an
   "open-work" flag.** Reuses R16's confidence field. No fourth
   L2 kind needed.
2. **Query routing + multi-kind fusion** must be added to R15.
   Three retrieval pipelines (temporal / topic / situation)
   need a router and a result fusion rule.
3. **Consolidation actions per tier** must be added to R21.
4. The sketch handles writes-without-query, refinement,
   spawning, and basic query-driven retrieval reasonably well.
   The weakest area is proactive surfacing (R24/R27) and the
   structural ambiguity around open questions (resolved here
   by interpretation (b) above).

**Recommended next-action order:** R16 → R27 → R21 → R19/R22/R23
in parallel. R16 first because it underlies refinement, open-
questions, and consolidation.

### 2026-05-23c — Meta: this log is a working prototype

User observation: the process we're running to design engram is
*itself* a query-based memory workflow. The mapping is clean
enough that the log can serve as an acceptance test for the
design.

**Structural mapping:**

| Our process | Memory-system equivalent |
|-------------|--------------------------|
| Lit summary, original spec, external papers | L0/L1 — raw and extracted sources |
| Change log entries | Episodes (temporally indexed) |
| Refined hypotheses in R entries | Facts (current best understanding) |
| Meta-lessons (e.g., "don't claim CLS authorization for things outside its scope") | Feedback (applies in similar future situations) |
| The open-questions board | Active L2 items, each R a situation needing resolution |
| Synthesis notes | Consolidated learnings with prose framing |
| Current conversation context | Working memory |

**Dynamics mapping:**

- New question → search existing state → use if covered, work if
  not → capture back.
- Old answers get refined (R24: procedural → situation-cued).
- Questions spawn questions (R2 → R15–R22; R15 → R23–R25; meta
  → R26).
- Working set is bounded — not every R is loaded every turn;
  only what's relevant.
- Three retrieval shapes observable: temporal (change log),
  topic (R#), situation (synthesis notes).

**Two sharpening insights:**

1. **Reconstructive retrieval (R22) is what we've been doing the
   whole time.** Each response is a synthesis from partial prior
   state, not a fetch. The log captures the synthesis. This
   de-risks R22 — we're already living it successfully.
2. **The L2 split (episodes / facts / feedback) is visible in
   this artifact.** Episodes in the change log; facts inside R
   entries; feedback in meta-lessons. R24's retrieval shapes are
   observable in how we navigate.

**Caveat — "query-based" is incomplete:**

Memory isn't only query-driven. The workflow exercises four
operations:
- Query-driven retrieval ✓
- Write/extraction without query ✓
- Maintenance/consolidation (R21) ✓ (the R15+R20 reframing)
- Proactive surfacing (R24) ✗ — relies on user to set context

The design must serve all four. R26 captures this as an explicit
acceptance test.

### 2026-05-22 — R15 + R20 reframing

The original framing ("how many tiers?") conflates two separate
questions that should be answered independently:

1. **How many LTM representations does engram need?** (R15) The CLS
   "two stores" claim derives from a neural-network plasticity
   problem that does not exist at engram's file substrate. What
   survives translation is the functional claim that episodic and
   semantic queries need different *representations* because they
   answer different questions. The literature does not dictate how
   many physical tiers serve those representations.
2. **What shape should the transient recall payload take?** (R20)
   The LLM context window is already functionally working memory.
   It lacks differentiation, activation gradient, selective
   retention, and role marking. The engram-side fix is to *shape
   what we emit into the context*, not to build a new store.

This reframing dissolves the "2 tiers vs. 4 tiers" debate as posed.
The right sequence is:

1. Enumerate the LTM representations needed (episodic-rich,
   semantic-distilled, and any others R6/R17 surface).
2. Decide what each representation is optimized for.
3. *Then* decide how many physical tiers / files / indices serve
   those representations. Implementation follows function.

This also pulls R20 forward in priority: payload shape is design
discipline applied to every recall, not deferred polish.

---

## Next actions (suggested order)

1. **R4 + R5 in parallel.** Cheap experiments. Results inform
   R1/R6/R15 (topology + tier count) and R18 (edge types).
2. **R3 decision.** Standalone judgment call; unblocks R10/R11.
3. **R1 + R15 together.** Side-by-side topology sketches; once R4/R5
   land, pick.
4. **R17, R19, R21 as a group.** All three are schema/salience/
   maintenance mechanisms that interact — they should be designed
   together, not separately.

---

## Log of board changes

- **2026-05-22:** Board created. R2 lit summary delivered;
  R8 decomposed; R15–R22 spawned.
- **2026-05-22:** R15 + R20 reframed via discussion. R15 is now
  "how many LTM representations" not "collapse to two tiers";
  R20 is design discipline on the recall payload, not a new
  tier; R20 priority raised P2 → P1. Synthesis note added.
- **2026-05-23:** First concrete LTM sketch on the table (L0
  pointers / L1 extraction / L2 episodes+facts+feedback / L3
  open). R23 (L1→L2 trigger), R24 (feedback as procedural),
  R25 (L3 status) spawned. R15 status `exploring`; working
  sketch captured inline. Synthesis note added.
- **2026-05-23b:** CLS scope corrected — covers only episodic +
  semantic, not procedural. R24 refined: feedback is semantic
  content with situation-cued retrieval, not procedural. The L2
  split is a retrieval-shape distinction over shared substrate.
  Synthesis note 2026-05-23b added.
- **2026-05-23c:** Meta — this log is itself a working prototype
  of the memory system being designed. R26 spawned as an
  explicit acceptance test ("does the sketch support the
  workflow we're using to build it?"). Synthesis 2026-05-23c
  captures the structural + dynamic mapping and the
  query-based-is-incomplete caveat.
- **2026-05-23d:** R26 walk complete. Verdict: viable but
  incomplete. Five mechanisms need sharpening (R16/R19/R21/R22/
  R23); R27 spawned for the situation-start hook (biggest
  structural gap). Open questions resolved as low-confidence
  facts (R16 reuse), not a fourth L2 kind. R26 status
  `resolved` with artifact link.
- **2026-05-23e:** Architecture-reality reconciliation. Multiple
  prior synthesis claims about today's system were wrong;
  corrected. `/please` already brackets with `/learn`; `/learn`
  already writes facts/feedback; `engram update` is binary
  refresh, not memory. R23, R7, R19 closed by reconciliation
  (the mechanisms exist). R28 spawned: do episodes earn their
  place as a third Permanent kind? P0 — gates the v2 spec.
- **2026-05-23f:** Focus list (F1–F8) locked at top of file.
  R20 subsumed into F7 (recall payload return format) —
  expanded from "role-marking" to the full question of how
  engram presents results to the LLM. The board below is now
  reference; F1–F8 are the active work.
- **2026-05-23g:** Pre-analysis section added. F7 thinking
  (five sub-questions, lean payload shape, sub-decisions still
  open) stashed there so it's available when F7 lands. Working
  order locked: F1 → F2 → F3 → F4 → F5 → F6 → F7 → F8.
- **2026-05-23h:** **F1 resolved — add episodes as a third
  Permanent kind.** Initial lean ("don't add") reversed by cost
  analysis: transcript embedding is ~50–100× the per-`/learn`
  cost vs. episodes, and episodes give qualitatively better
  output for narrative queries. F3 simplified accordingly
  (transcript chunks no longer needed). Sub-decisions for the
  v2 spec captured in the F1 resolution section.
- **2026-05-24:** **F9 added, deferred to v3.** Clustering +
  bottom-up synthesis + embedding granularity (block-level vs
  whole-note vs field-level). Verified Smart Connections is
  k-NN only — no clustering, no auto-synthesis — making F9 real
  engram differentiation if built. Block-level embedding may
  subsume field-level granularity for structured facts/
  feedback; verify markdown rendering shape before deciding.
- **2026-05-24b:** **F2 resolved — Hugot + GoMLX simplego +
  Arctic-xs (default) with MiniLM-L6 fallback.** Pure-Go, no
  network calls. Spike spec written to
  `2026-05-24-engram-query-spike.md` — 13 UAT cases including
  a reference-parity gate (case 13) that decides Arctic-xs vs
  MiniLM-L6. Five sub-decisions settled: body-only content hash;
  model bundled in binary; explicit error on missing
  embeddings; warn-and-proceed on auto-embed failure; default
  limit 20. BM25 confirmed deferred to v3 (engram has no text
  search today; embeddings are the first retrieval mechanism
  added).
- **2026-05-24c:** **F3 resolved — embed all three Permanent
  kinds.** Facts, feedback, episodes. No transcript chunks
  (F1's cost analysis ruled them out). MOC content disposition
  is F4's concern. Block-level/field-level granularity deferred
  to F9.2.
- **2026-05-24d:** **F4 resolved — migrate MOCs to facts and
  feedback (mixed) with meta-abstraction analysis.** Drop
  `engram learn moc` after migration. ~50-100 derived notes
  expected from 25 MOCs. Procedure at
  `2026-05-24-moc-migration-procedure.md`. Originals archived
  to `_legacy/MOCs/`. Insight captured: this exercise validates
  F9 — the migration *is* F9 done manually, and if it produces
  valuable meta-abstractions, the F9 automation is worth
  building. F9 pre-analysis updated to reflect this.
- **2026-05-24e:** **F5 resolved — drop emergent edges
  entirely.** No co-retrieval counter, no `.engram/emergent.json`,
  no emergent-vs-authored distinction. Rationale: embeddings
  already cluster semantically similar notes; F9.1 in v3 is the
  stronger mechanism for usage-pattern clustering if needed.
  Simplifies F6 (authored wikilinks only) and F7 (no emergent
  edge type in payload).
- **2026-05-24f:** **F6 + F9 resolved together.** F6 alone
  dropped (link traversal without structural analysis is
  pointless). F9.1 split: clustering (subgraph-level) promoted
  to v2 as part of `engram query`; auto-synthesis stays
  caller-side (in the updated `/recall` skill). Algorithm:
  auto-k-means via silhouette. Hop depth: 3. Synthesis
  expectations spec'd for the consuming skill — same
  operation as F4's MOC migration, applied at query time on
  demand. F9.2 (granularity) still deferred.
- **2026-05-24g:** **F7 resolved — recall payload shape.**
  YAML only. Front-loaded `items` section with full content,
  deduplicated by path, provenance-tagged (`direct` /
  `cluster_rep` / `hub`). Ordering: provenance count desc →
  provenance priority desc → score desc. `clusters` section
  is structural reference; non-rep members get path+score
  only. No separate hubs section (filter `items` on
  provenance). R20 fully subsumed by the `provenances` field.
- **2026-05-24h:** **F8 closed — drop entirely.** Proactive
  feedback surfacing requires harness integration we can't
  affect. `/please`'s `/recall` bracket gives semi-proactive
  coverage for bracketed work. Ad-hoc work outside `/please`
  remains uncovered; not addressable without harness hooks.
- **2026-05-24i:** **V2 focus list complete (F1-F9).** All
  resolved or explicitly dropped. Ready to draft the v2 design
  spec (replacement for `2026-05-14-tiered-memory-design.md`)
  and execute the F2 spike.
- **2026-05-24j:** Spike spec
  (`2026-05-24-engram-query-spike.md`) updated to incorporate
  F1–F9 conclusions. Scope kept deliberately narrow (foundation
  only — embedder validation + embed-on-write pipeline + basic
  k-NN query). Post-spike v2 work order added: MOC migration
  (F4) → drop `engram learn moc` → episodes (F1) → subgraph
  clustering (F6+F9.1) → updated `/recall` skill. Settled
  decisions extended (#6: YAML payload only, from F7).
