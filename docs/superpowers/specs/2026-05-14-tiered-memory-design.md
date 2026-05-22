# Tiered memory architecture for engram — design

Status: draft for user review. No code yet. Companion research brief:
`2026-05-12-tiered-memory-research-prompt.md`.

This revision reorders the original Q1–Q12 answers into a flow-shaped
structure so each concept is defined before it is referenced. The
original Q labels are preserved as parentheticals on section
headings for cross-reference with the brief.

## 1. Vision

Memory is a lopsided hourglass. The bottom is wide — every session
the agent has ever run, addressed but not copied. The waist is narrow
— atomic facts that earned a place by passing a retrieval-mirror
test. The top re-widens slightly into synthesis: a small number of
maps-of-content that organize the facts along constrained dimensions,
and that get *regenerated* as the facts beneath them grow.

The lower three tiers (L0 references, L1 stripped segments, L2 atomic
facts) are append-only. The upper tier (L3 synthesis, including any
MOCs-of-MOCs and the single root MOC) is mutable and regenerated.
Conflicts are preserved at L2 and reconciled at L3.

The redesign exists because the current vault discards the substrate.
Every Permanent note distills a session into a principle, but the
session itself — and the un-distilled patterns inside it — are
invisible to recall. The motivating case ("you did it but you don't
remember") proved the information was *there* — in `~/.claude/projects/`
JSONL — and only the absence of an index made it unrecoverable.

Two failure modes the redesign must not produce:

- **Re-introducing the tier-collapse complexity.** Permanent/8d, 9g,
  and 8i argue that any tier whose work is subsumed by the parent
  LLM's reasoning over already-loaded context should be dropped. The
  Fleeting tier was eliminated for exactly this reason. L1 must
  justify its existence against this rule.
- **Smoothing contradictions** (Permanent/4c). The vault preserves
  disagreement; the redesign preserves it harder, with explicit
  `contradicts:` edges at L2 and reconciliation prose at L3.

## 2. Hard constraints (inherited; quoted)

These are not adjustable in this design pass. If a future round wants
to change them, do it explicitly.

- **"Pure Go, no CGO."** Rules out FAISS bindings, ONNX runtime,
  sqlite-vss, sqlite-vec via the standard `mattn/go-sqlite3` path.
- **"No LLM calls from the binary — keep `engram recall` snappy
  and offline."** External embedding API at write time is fine;
  read time stays offline.
- **"L0–L2 are append-only. L3+ is regenerated."**
- **"Every L2 and L3+ entry carries 2–3 tests."** Adopted with the
  retrieval-mirror consolidation explained under §11 (Tests).
- **MOC craft rules (Permanent/4, 4a, 4b, 8a)** — no global index,
  no bare `Related:` lists, LLM-voiced framing prose, every wikilink
  in a sentence that explains the connection. The dimension scheme
  in §6.1 must not become tags-in-disguise.

One constraint the brief mandates has been relaxed by user decision
during this design pass: **Luhmann IDs are retained at L2 only as a
secondary retrieval signal.** See §4.

## 3. The four tiers

Each tier has its own write path, retention rule, and role in recall.
Later sections define the mechanics; this section is the at-a-glance
reference.

- **L0 — provenance index.** Append-only inverted index over session
  JSONL (file paths from tool-use payloads, tool names, short
  snippets). No transcript content copied. Read on demand from
  `~/.claude/projects/` and `~/.opencode/...`. The widest tier; cheap
  to grow, expensive to read.
- **L1 — stripped task-boundary segments.** Append-only. Small
  LLM-voiced segments emitted by the active skill at task close,
  carrying situation framing, summary, entities, and L0 provenance.
  Indexed both lexically and via embeddings.
- **L2 — atomic facts.** Append-only with redirects for merges.
  Principle-stated notes that earned a place by passing a
  retrieval-mirror test. Carries `contradicts:` and `supersedes:`
  edges. This is the workhorse of recall.
- **L3 — synthesis.** Mutable, regenerated. MOCs and MOC-of-MOCs
  with LLM-voiced framing prose that organize L2 facts along
  constrained dimensions. A single root MOC sits at the apex.

Append-only tiers (L0–L2) survive forever; conflicts and supersession
are tracked by edges, not by deletion. L3 is regenerated whenever
drift over its constituents passes threshold.

## 4. Identity, links, and merges (Q2)

**UUIDs are the canonical handle at every tier; wikilinks resolve
through a UUID map; L2 retains Luhmann IDs as a redundant retrieval
signal only.**

User decision during design: Permanent/4e's argument that Luhmann
sibling-proximity is a load-bearing free retrieval signal is
accepted; the brief's blanket drop is relaxed. L2 is append-only, so
Luhmann IDs survive there. L3 is mutable, so Luhmann IDs are *not*
minted at L3 — L3 nodes have UUIDs only.

L2 frontmatter:

```yaml
uuid: 01HXAB...        # canonical handle, ULID
luhmann: "1a3"         # retrieval signal only, never load-bearing
slug: cascade-pruning-failure
created: 2026-05-14
source:
  session_id: a22ad7f7-...
  tool_use_ids: ["toolu_01ABC..."]
contradicts: []        # list of UUIDs
supersedes: []         # list of UUIDs (soft-deletion target)
```

Wikilinks are always `[[uuid]]` or `[[uuid|display text]]`. The
filename remains human-readable for grep ergonomics:
`L2/<luhmann>.<YYYY-MM-DD>.<slug>.<uuid8>.md` (last 8 hex of UUID
appended for collision safety). L3 filenames drop the Luhmann
segment: `L3/<YYYY-MM-DD>.<slug>.<uuid8>.md`.

**Merge/rename graph.** When two L2 segments are deduped at write
(see §5.1), the new one is the canonical UUID and the duplicate
becomes a `redirect` stub: a one-line file mapping its UUID to the
canonical. Resolvers follow redirects transparently. Inbound links
are *not* rewritten — the resolver handles it lazily. A nightly
`engram compact` job rewrites redirects older than 30 days into
direct links. This is the explicit answer to the brief's "If A
merges into B, anything that linked to A has to follow."

**Outbound links from a merged note** are unioned into the canonical
on first merge; subsequent recall sees the merged set.

## 5. Write paths

### 5.1 L2 — atomic facts (Q5)

**New L2 only on first dive; draft L3 only on subsequent dives or
threshold; mid-query extraction is durable.**

`/learn fact` writes L2. The binary mints a UUID, optionally allocates
a Luhmann ID via the existing locked allocator, embeds the note via
the embedding API (§8), writes a `.vec.json` sidecar, generates a
retrieval-mirror test (§11), updates the in-memory BM25 index, and
runs a dedup pass against existing L2 (top-5 vector + lexical; merge
if cosine > 0.92 AND BM25-overlap > 0.7).

When recall descends into L1 or L0 and finds a pattern, the cascade
write-back path (§10) also lands here:

1. **Always:** writes a new L2 fact with `source.cascade: true` and
   provenance pointing at the L1/L0 segments that fed it.
2. **Conditional:** marks the touched cluster as "regeneration
   candidate." The next `engram synthesize` invocation will pick it
   up if drift thresholds (§6.2) are crossed.
3. **Never:** rewrites L3 inline mid-recall. L3 regeneration is
   always a separate step with the user's LLM.

Mid-query extraction is durable — no confirmation step — because
(a) the retrieval-mirror test catches mis-extraction on the next
recall, and (b) L2 is append-only with `supersedes:` so a wrong
extraction can be soft-deleted later without information loss.

The brief asks "should every successful L0 dive write an L2 fact?"
The answer is yes by default, *with deduplication on write* (the
merge path in §4). The brief's worry ("creates churn") is handled by
the dedup gate, not by skipping writes.

### 5.2 L1 — task-boundary segments (Q1)

**Agent-emitted L1 segments at task boundaries are the default;
a PreCompact / Stop hook is the safety net; live inline indexing is
deferred to a later round.**

The brief offers three production paths, in increasing agent
involvement. Path 2 (task-boundary emission) is the workhorse. Path 1
(mechanical hook dumps) is mostly redundant with JSONL but catches
the case where the agent forgets to emit. Path 3 (live inline
indexing) is the most promising long-term but introduces an
in-context tool surface that hasn't been pressure-tested and would
require both a CLI primitive and a skill rewrite. Defer.

At task close, the active skill emits N small L1 segments through
`engram learn segment` (new subcommand). Each segment carries:

- `segment_id` (ULID-derived UUID, the canonical L1 handle)
- `source.session_id`, `source.tool_use_ids[]` (the L0 provenance)
- `situation` (phrased as the *task the agent was on*, per
  Permanent/8k — so write and read share framing)
- `summary` (1–3 sentences; LLM-voiced)
- `entities[]` (file paths, symbols, identifiers — verbatim, indexed
  lexically)
- `links_back[]` (related existing handles, resolved at write)

L1 gating uses a softer test than L2: "would a future query about
this *situation* benefit from finding this verbatim?" That's weaker
than the L2 retrieval-mirror test (§11), which asks "would this be
the right answer?" L1 is allowed to be redundant; L2 is not.

Cost ceiling: ~10–30 L1 segments per non-trivial session. At 10³
sessions/year that's 10⁴–3×10⁴ L1 segments — well inside the chosen
substrate (§8).

The PreCompact / Stop hook fires only when no L1 emission has
occurred for a session whose JSONL crossed a turn-count threshold
(default: 30 turns). It dumps a minimal "session abandoned without
L1 capture" stub pointing at the JSONL — recall can then dive into
L0 if the situation calls for it.

### 5.3 L0 — provenance index

`engram l0 ingest` updates the L0 inverted index from new JSONL
files since the last marker. It uses the existing `learnmarker`
per-harness state. Indexed fields: file paths from tool-use payloads,
tool names, short snippets. No transcript content is copied.

## 6. Synthesis and regeneration

### 6.1 MOC dimensions (Q4)

**N independent indices; each MOC commits to exactly one primary
dimension; a note appears in many MOCs.**

The brief's initial dimensions: time, reference metadata, subject /
predicate / object, situation / behavior / impact / action.

Risk Permanent/4a flags: independent indices that a note can be
filed under = the tag system, which is forbidden. The discriminator
is **in-prose framing with strength signal** (Permanent/4a, 8a). A
MOC is *not* a tag because:

- It has framing prose stating *why this cluster matters and how
  the constituents relate*, in the LLM's voice.
- Each constituent wikilink lives in a sentence that explains the
  connection. Bare lists are rejected at write.
- Constituent membership is judged at MOC regeneration, not by a
  classifier field on the constituent note.

To enforce this: the MOC writer (whether human or LLM) cannot just
list constituents; the synthesis prompt is structured to produce
framing-prose-first output. An `engram lint` check rejects MOCs
whose constituents are not introduced in prose.

MOCs-of-MOCs aggregate *across* dimensions: e.g., a MOC on "memory
system design" pulls from the subject dimension (notes on memory)
*and* the situation dimension (notes on recall failures). Each
MOC-of-MOC commits to a *cross-dimensional theme*, not to a single
dimension.

### 6.2 Regeneration cadence (Q3)

**Drift-detected + threshold-batched, never per-write, never on
read.**

Per-write L3 regeneration would thrash. Always-on read-time
regeneration violates the binary's no-LLM constraint. The middle
ground:

A regeneration job (`engram synthesize`) is invoked by the user (or
a Stop hook at session end). The job:

1. Computes per-MOC drift: count of new L2 facts in the MOC's
   constituent set since the MOC's last regeneration, plus a
   centroid-shift score (cosine distance between the old constituent
   centroid and the new one).
2. Regenerates MOCs whose drift exceeds `regenerate_when` thresholds
   (default: 5 new constituents OR centroid shift > 0.15).
3. Regeneration calls the *user's* LLM via an external prompt — *not*
   the engram binary — so the "no LLM in binary" constraint holds.
   The binary prepares the prompt (constituents + framing); the
   skill or shell wrapper executes it.

The single root MOC regenerates on user demand only.

### 6.3 The root MOC (Q8)

**A single regenerated artifact (`L3/root.md`), produced on user
demand, with provenance to its constituent MOCs.**

Not edited by hand. The root MOC is a regeneration of all top-level
MOC clusters into one navigation surface. It is LLM-voiced, not
human-curated, and it's explicitly *not* the entry point for recall
(the cascade starts at L3 via vector search, not at root).

## 7. Conflict semantics (Q6)

**Both preserved with `contradicts:` edges; surfaced together on
recall; L3 reconciles in prose, never elects a winner.**

Two L2 facts that disagree both stay. Each carries a `contradicts:`
field listing the UUID(s) it contradicts. Recall always surfaces
both when one matches — the resolver follows `contradicts:` edges
unconditionally.

L3 reconciliation is *prose synthesis*, not voting. The MOC's
framing prose explicitly names the disagreement, the context that
makes each side hold, and (if discoverable) the dimension along
which they're reconciled. This is the natural extension of
Permanent/4c: contradiction *is* the signal.

A `contradiction-flag` is a per-note metadata field surfaced in
every retrieval — the LLM reading recall output sees the conflict
immediately and doesn't need to derive it.

## 8. Retrieval substrate (Q12)

- **Embedding provider: Voyage `voyage-3-large` at 1024-dim.**
  Top of MTEB (65.1), strong on code retrieval, Matryoshka allows
  dimension reslicing without re-embed if quantization wanted later.
  Cost at 10⁴ × 500 tokens = 5×10⁶ tokens ≈ $0.30 per full re-embed.
  Pricing comparable to OpenAI text-embedding-3-large; the code
  retrieval edge is the decider for a vault with code-heavy
  transcripts. Fallback to OpenAI `text-embedding-3-small` is
  trivial (swap provider behind interface).

- **Vector store: `philippgille/chromem-go` flat backend.** Pure Go,
  zero CGO, brute-force cosine sub-10ms at 10⁴ vectors (benchmarked).
  Persistence is gob+gzip, one collection per tier. Upgrade path:
  `coder/hnsw` (also pure Go) behind the same interface when vault
  approaches 10⁵.

- **Lexical complement: hand-rolled in-memory BM25.** Vault size
  stays inside RAM (vocab and posting lists in tens of MB at 10⁴
  notes); rebuild on startup. Bleve is too heavy. The BM25 index
  serves both L2 and L1 (separate indices) and the L0 inverted
  index over session-JSONL entity strings (file paths, tool names,
  symbol snippets).

- **Fusion: Reciprocal Rank Fusion (k=60)** between vector and BM25
  results at L2 and L1. RRF bypassed when query has exact-match
  signal (file path, quoted phrase) — exact-match wins outright in
  that case, avoiding RRF's strong-signal dilution.

- **Cross-encoder re-ranking: out of scope.** All credible
  cross-encoders are ONNX/CGO; pure-Go inference doesn't exist in
  production form. Voyage-3-large's MTEB ceiling is sufficient for
  engram's recall@k target without reranking.

- **Sidecar storage per note:**
  `L2/<...>.vec.json` containing `{embedding_model_id, dims,
  vector: [...], chunks: [{start, end, vector: [...]}]}`. L1 same
  shape. L3 MOCs embed *framing prose only*, not constituent lists
  (per the cascade synthesis insight that MOC prose is the relevance
  signal, not its membership).

- **Recall@k target: 90% recall@10 on L2.** Measured via the
  per-note retrieval-mirror tests aggregated.

- **Latency budget: `engram recall` end-to-end under 200ms** at
  10⁴ L2 notes. Vector + BM25 at L2 ≈ 15ms; L3 negligible;
  cascade subagent reads are out of the binary's budget.

- **Graceful degrade: when embedding API unreachable at write,
  segment is queued and embedded later. At read, the binary never
  calls the API — all reads use pre-stored vectors. If a note has
  no stored vector (queued), BM25-only ranking is used for that
  note (it can still surface lexically).**

## 9. Data layout

```
agent-memory/
  L2/                            # atomic facts (append-only)
    <luhmann>.<date>.<slug>.<uuid8>.md
    <luhmann>.<date>.<slug>.<uuid8>.vec.json
    <luhmann>.<date>.<slug>.<uuid8>.tests.yaml
  L3/                            # synthesis (mutable, regenerated)
    <date>.<slug>.<uuid8>.md
    <date>.<slug>.<uuid8>.vec.json
    <date>.<slug>.<uuid8>.tests.yaml
    root.md                      # top-level synthesis
  L1/                            # stripped segments (append-only)
    <YYYY-MM>/                   # chunked by month for fs sanity
      <segment-uuid>.md
      <segment-uuid>.vec.json
  L0/                            # provenance index only
    sessions.idx                 # BM25 inverted index over JSONL
    sessions.cursor              # per-harness markers
                                 # (no JSONL copied; only pointers)
  .engram/
    bm25.l2.gob                  # rebuilt on startup
    bm25.l1.gob
    bm25.l0.gob
    uuid-map.gob                 # uuid → relative path
    redirects.gob                # merged uuid → canonical uuid
```

Filename note: `<uuid8>` is the last 8 hex chars of the ULID, for
collision-safe uniqueness without forcing 26-char filenames.

L0 stores *no JSONL content* — only an inverted index over file
paths, tool-use names, and short `old_string`/`new_string` snippets
(first/last 200 chars). Full JSONL is read on-demand from
`~/.claude/projects/` and `~/.opencode/...` via existing
`engram transcript` infrastructure.

## 10. Recall cascade (Q9)

**Start at L3; descend to L2 → L1 → L0 with per-tier budgets and
per-tier query shapes; L0 entry requires a structural signal (file
path, symbol, exact phrase).**

Round 0: vector + lexical hybrid retrieval over L3 (RRF, k=10).
This is the new anchor stage. The MOC layer's framing prose is the
embedding source, not the constituent list.

Round 1: for each surfaced L3 node, expand via wikilinks (the
current cascade pattern) AND via vector NN over L2 (top-k per L3
hit, k=5). Union into the L2 frontier.

Round 2: read L2 frontier (parallel subagents for >10), score
relevance against query + situation phrases (current skill
pattern). Expand frontier via wikilinks and `contradicts:` edges.

Round 3 (L1 entry): if L2 produces fewer than `min_l2_hits`
surfaced notes OR the query carries L1-shaped signal (file path,
symbol, "what did I do when…"), descend into L1. Vector + lexical
search over the L1 segment index. Surfaced L1 segments score
against situation phrases.

Round 4 (L0 entry): only entered when L1 also produces few hits
AND the query carries explicit L0-shaped signal (a file path that
matches no L1 entry, a tool-use payload, "in a prior session…").
The L0 inverted index is consulted; relevant session JSONLs are
read by a subagent (raw content stays out of parent context).
Patterns discovered are written back as L2 facts (§5.1).

Cost model: L0 reads are 100–1000× more expensive than L2 reads in
both token and wall-clock terms. The L0 gate is conservative by
design. Tier descent is *not* speculative — if L2/L1 produced
adequate results, L0 is not entered.

Visible progress per round (per Permanent/81): the binary emits a
status line each round (`round N: surfaced=X, frontier=Y,
budget_left=Z, next_tier=L1`).

## 11. Tests (Q7)

**Sidecar `.tests.yaml` per L2/L3 note; one retrieval-mirror test
by default, up to three for high-stakes notes; `engram test` runs
them; failure flags for review, never auto-deletes.**

The brief asks for 2–3 tests answering four questions: relevance,
search phrasing, retrieval surfacing, behavior change. Permanent/83
argues these collapse into a single retrieval-mirror test: "would
the query phrasings a future agent would use in this situation
surface this note?" The other questions are *facets* of that test,
not independent.

Default: one retrieval-mirror test per L2/L3 note, generated by the
write-time skill. High-stakes notes (those linked from MOCs-of-MOCs
or the root) may add a second test from a different situational
framing. A third test is reserved for notes flagged as
behavior-changing on first pass — these get a *cold-agent* test (a
subagent runs the situation cold, with and without the memory, to
verify behavior diverges).

Sidecar format:

```yaml
# L2/<...>.<uuid8>.tests.yaml
tests:
  - kind: retrieval-mirror
    situation: "implementing OpenCode harness wiring"
    query_phrases:
      - "OpenCode reader alongside Claude Code"
      - "per-harness session marker"
    expect_surfaced_in_top_k: 10
  - kind: cold-agent
    situation: "..."
    expect_diff_summary: "uses per-harness marker, not shared marker"
```

`engram test` runs the retrieval against the index, checks
expectations, and writes results to `.tests-status.json`. Failures
flag the note for review (`engram test --failed` lists them).
Auto-deletion is never appropriate — Permanent/4g3's "delete
corrupted memories" applies to *empty-action* notes, not to
test-failing ones, which may indicate retrieval-index drift, not
note staleness.

Cadence: pre-commit hook (changed notes only) + nightly full sweep.

## 12. Failure modes and diagnostics (Q11)

Spec'd diagnostics:

| Failure | Symptom | Diagnostic | Action |
|---------|---------|------------|--------|
| L1 unbounded growth | Segment count >10⁵; vector index >500 MB | `engram doctor --tier=L1` reports size, growth rate | Tighten task-boundary gate; soft-evict segments not surfaced in last 90 days (move to cold tier, not deleted) |
| L3 thrashing | Same MOC regenerated >3× per week | Regeneration log shows drift bouncing | Raise drift threshold; investigate whether L2 writes are duplicates passing dedup |
| Tests pass, behavior unchanged | Recall surfaces note, but cold-agent test shows no behavior diff | `engram test --kind=cold-agent --failed` | Rewrite note framing; if persistent, the note is *retrievable but useless* — flag for human review |
| Embeddings drift (model change) | `embedding_model_id` mismatch between query and stored vectors | Recall refuses cross-version queries | Re-embed batch job; tracked as a single migration |
| External API unreachable at write | `/learn` fails on embedding step | Skill retries with backoff; falls back to "stage without embedding" mode, marks segment as `embedding: pending` | Background job picks up pending and embeds when API returns |
| External API unreachable at read | (Should never happen — read is offline) | If a skill mistakenly attempts API at recall, lint blocks it | — |
| L0 dive returns nothing | Query had L0 shape but no JSONL matches | Status line reports `L0 frontier empty: 0 matches` | Normal; no action |
| L0 dive returns everything | Query matched too broadly; thousands of sessions | Budget-cap at top-1000 by mtime, surface to user | User refines query |

The diagnostic command `engram doctor` runs all of these and
reports counts; intended for periodic vault health checks.

## 13. Migration (Q10)

**Scripted one-shot migration; existing Permanent → L2 with UUID
minted; existing MOCs → L3 with UUID minted; Luhmann IDs preserved
at L2 only.**

`engram migrate` runs once:

1. Walks `Permanent/`, `MOCs/`, computes UUID per note (ULID), writes
   new file under `L2/` or `L3/` with frontmatter including the new
   UUID and original Luhmann ID.
2. Rewrites all wikilinks from `[[<luhmann>.<date>.<slug>]]` to
   `[[<uuid>|<slug>]]` using a luhmann→uuid map.
3. Generates retrieval-mirror tests for migrated notes (LLM-driven,
   one pass).
4. Generates embeddings for all migrated L2 notes (one batch call to
   Voyage).
5. Does *not* attempt L1 backfill or L3 regeneration. L1 starts
   empty and accumulates from new sessions. L3 starts as the
   migrated MOCs and regenerates from there.

Migration is not reversible. The old `Permanent/` and `MOCs/`
directories are kept (renamed to `_legacy/`) for one release cycle,
then removed.

## 14. Out of scope (deliberately)

- **Re-introducing Fleeting.** Tier-collapse removed this for good
  reason (Permanent/9g). L1 is *not* Fleeting reborn — L1 segments
  are situationally-keyed transcript distillations, not
  Permanent-shaped candidates.
- **In-binary LLM at recall time.** Inherited constraint.
- **Cross-encoder reranking.** Out of pure-Go reach today.
- **L3 hand-editing as primary path.** L3 is regenerated; users can
  edit, but edits are subject to being overwritten on regeneration.
  If permanent human prose is wanted, it lives at L2.
- **Mutable L2.** L2 is append-only with redirects for merges. No
  in-place edits. Permanent/4c1 is honored: identical content =
  skip; different content = new file.

## 15. Smallest first slice (build & measure)

**Goal of the slice:** demonstrate that semantic retrieval at L2 +
an L0 inverted index together solve the motivating "you did it but
don't remember" case, without committing to L1, L3 mutability, or
migration.

**In scope of slice:**

1. **L2 vector index.** Embed every existing Permanent note via
   Voyage at slice-start (one batch). Store `.vec.json` sidecar.
   Add `engram recall --semantic <query>` returning top-k by
   vector + BM25 RRF. Keep existing `--follow`/`--recent`/anchors
   for cascade compatibility.
2. **L0 inverted index.** `engram l0 ingest` walks
   `~/.claude/projects/`; `engram l0 search <query>` returns
   ranked session IDs with paths + tool-use IDs. No transcript
   content copied.
3. **UUID frontmatter on new notes only.** New L2 notes get UUIDs;
   existing notes keep their current handle. UUIDs and Luhmann
   coexist for the slice; migration of legacy notes happens later
   when L3 regeneration ships.
4. **One retrieval-mirror test per new note**, written by `/learn`.
   No `engram test` runner yet — tests are scored manually until
   the runner ships.

**Out of slice:**

- L1 capture (`/learn segment`).
- L3 regeneration.
- Test runner / `engram test`.
- Dedup-on-write merge graph (rely on existing slug-based behavior).

**Measurement:**

- Replay the "you did it but don't remember" recovery: can
  `engram l0 search "skills/recall/SKILL.md"` find sessions
  `a22ad7f7` and `677d4acf` in under 1 second? **Pass/fail.**
- Pick 10 recent recall sessions; for each, compare top-10 of new
  semantic recall vs current anchors+recent baseline. Score by
  whether the *load-bearing* notes for that session's task surface
  in top-10. **Target: ≥80% recall@10**, with the remaining gap
  documenting the cascade work still needed.
- Measure `engram recall --semantic` p95 latency. **Target: <50ms**
  at current vault size (~400 notes).

If the slice passes those three measurements, build L1 capture next.
If recall@10 < 60%, the substrate choice is wrong and §8 needs
revisiting before more tiers are added.

## 16. Open items the user should adjudicate before implementation

1. **The Luhmann relaxation (§4)** was decided in this design pass.
   Confirm.
2. **Voyage vs OpenAI** as the default embedder. Voyage recommended;
   OpenAI is a safe alternative if Voyage account setup is friction.
3. **L1 segment frequency cap.** Default "10–30 per session" is a
   guess; will tune from data.
4. **MOC dimension primary set.** The brief listed four candidate
   dimensions. The design assumes all four are kept. Confirm or
   trim before regeneration ships.
5. **Whether to gate `engram l0 ingest` on user consent for each
   external project's JSONL** (some sessions may include sensitive
   transcripts; the inverted index over file paths is low-risk but
   non-zero).
