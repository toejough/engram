# Architecture Decision Records — engram

Retrospective ADRs. The decisions below were mostly made implicitly across the system's
evolution; this log records them so they can be challenged, and ties each to the **verified
defects it produced** (the `⚠ KNOWN` lines). Rigor cross-reference: the
[memory-system rigor effort](memory-system-rigor.md).

History: the founding design narrative (tiered-memory research, embedder choice, lazy-L2
synthesis, and the decisions superseded along the way) lived in `docs/DESIGN-HISTORY.md`,
deleted 2026-07 — `git log` recovers it.

Status legend: **Accepted** · **Accepted (known defect)** — sound decision, buggy as-built ·
**Superseded**. Evidence: commit hashes, `file:line`, and the C4 set
([L1](c1-system-context.md) / [L2](c2-containers.md) / [L3](c3-components.md)) +
[invariants](memory-invariants.md).

---

## ADR-0001 — Skills + slim binary split

**Status:** Accepted (INV-S1 seam resolved 2026-06 via `engram amend`, `internal/cli/amend.go`; #700 (2026-07): raw I/O primitives relocated to `cmd/engram` — wiring-only `package main` over grouped `cli.Primitives`, `targ check-thin-api`-enforced; ALL adapter composition + wiring live in `internal/cli` (`cli.NewDeps`); `internal/` is import-pure — lint-enforced, ADR-0020)

**Context.** The work divides into LLM judgment (which lessons to capture, how to frame a
`situation`, whether a cluster shares a binding principle) and deterministic compute (cosine,
graph BFS, k-means, marker arithmetic). Mixing them makes the judgment untestable and couples
model behavior to Go code.

**Decision.** Behavior lives in **markdown skills** (C1, executed by the agent in the harness);
deterministic compute lives in a **slim Go binary** (C2). They communicate **only** through C2's
CLI surface (args in, stdout out) and the vault on disk. Each `engram <subcommand>` is a separate
OS process; subcommands never call one another in-process.

**Consequences.** The invariant checker gates C2 (everything in C2 is deterministic and testable);
skills are gated only by RT acceptance tests. INV-S1 (resolved): the skill no longer touches the
vault directly — recall reads via `engram show-chunk` / the query payload (notes carry content inline), and `engram amend`
(`internal/cli/amend.go`) now provides the sync-preserving in-place edit path (rewrites both copies
+ re-embeds), closing the INV-S1 write-half ("no `engram` edit subcommand").

---

## ADR-0002 — Pure-Go embedded model; no external embedding API

**Status:** Accepted (known defect: M4) · supersedes the 2026-05-14 external-Voyage design

**Context.** An early design embedded via an external Voyage API — network dependency, per-call
cost, latency, and sending vault content off-box.

**Decision.** Bundle `all-MiniLM-L6-v2` (384-d) into the binary via `go:embed`; run inference in
**pure Go** through Hugot + GoMLX's `simplego` backend (no CGO). The only external API in the
system is the LLM that runs the agent itself — never embeddings.

**Consequences.** Deterministic, offline, zero per-embed cost; the embedder is a *container of S2*,
not an L1 external. A single `embedding_model_id` is stamped into every sidecar. ⚠ KNOWN (M4):
`loadCompatibleSidecars` (`query.go`) silently drops sidecars whose `model_id ≠` the active
model — a model swap silently empties recall unless `engram embed apply --force` re-embeds first.
No guard except the all-empty error path.

---

## ADR-0003 — Embed-on-write with per-note `.vec.json` sidecars

**Status:** Accepted (known defect: K1)

**Context.** Semantic search needs vectors, but a vector DB or a separate rebuild step adds a
moving part that can drift from the notes.

**Decision.** Every `engram learn` writes a sibling `<note>.vec.json` (`vector` + `model_id` +
`content_hash`) as part of the same operation. Note + sidecar creation is serialized under
`flock(.luhmann.lock)` spanning id-compute→write, with `O_EXCL` to prevent clobber.

**Consequences.** No index to maintain or invalidate; sidecars travel with notes (a vault copy is
self-contained, no re-embed). `content_hash` is meant to detect staleness. E4 (episodes embed `situation` but `ContentHash` hashes the body) was resolved by the episode retirement (2026-06-19, alongside ADR-0006/0008). ⚠ KNOWN (K1): concurrency correctness rests on the flock spanning the entire
id→write critical section — enforced in code, untested as a property.

---

## ADR-0004 — Three tiers; blended-default retrieval; tier is a frontmatter field

**Status:** Accepted · supersedes the "top-tier-only default" design prose (`docs/DESIGN-HISTORY.md` Decision 3 — deleted 2026-07; git log recovers it)

**Context.** Memory has three useful grains: raw episodes (L1), specific facts/feedback (L2), and
distilled standards (L3). Retrieval must pick the right grain. Early prose proposed defaulting to
the top tier only; empirically, **blended** retrieval scored better (2026-05 tier-retrieval eval; the
tiered model was itself removed in the 2026-06-20 deep clean, so this decision is now largely historical).

**Decision.** Default retrieval is **blended / kind-agnostic**. `--tier X` was an **optional cap** —
this flag was removed in the 2026-06-20 deep clean; unified clustering is now the sole
query path and operates un-tiered (cross-tier clusters). Tier is a **frontmatter field** with
type-derived defaults: fact/feedback → L2 (default, overridable to L3).
There is **no `adr` kind** — an ADR is `type:fact tier:L3`.
Recall-time lazy-L2 synthesis via `candidate_l2s` + covered/near/absent supersedes the
`nearest_l3` annotation (ADR-0005). The `nearest_l3` and `hubs` payload channels are removed.

**Consequences.** Items-isolation holds (verified: L1 29/29, L2 11/11, L3 0). The override
is a feature, so tier↔kind is asymmetric (T2).

---

## ADR-0005 — L3 ADRs are scenario-discoverable, synthesized from L2 clusters by centroid cosine

**Status:** Superseded by the 2026-06-09 lazy-L2 synthesis design (`docs/DESIGN-HISTORY.md` §7 — deleted 2026-07; git log recovers it) — L3-ADR-synthesis-at-learn-time is retired; crystallization is now recall-time, agent-judged lazy-L2 (covered/near/absent) via engram amend/learn.

**Context.** An L2 fact only surfaces if you query its keywords — but the agent who needs it does
not know it exists. Standards must be discoverable from the **situation** the agent is in.

**Decision.** §6b: when a `/learn` pass writes L2, seed 3–6 **scenario** situations, run
`engram query` per seed, and for each returned cluster **update** the nearest existing L3 if
centroid cosine ≥ 0.9, else **create** a new L3 (`fact --tier L3`). The loop is **skill-orchestrated**
— there is no `engram synthesize`; the binary only answers separate query/embed/learn calls.

**Consequences.** Standards retrieve by situation, not by lesson-keyword. ⚠ KNOWN: per-pass
write-sparsity starves `AutoK` (silhouette threshold), so clusters rarely form at write time —
the rebuilt vault has only 1 L3 from 106 L2. ⚠ KNOWN (INV-S2): §6b "revise its `situation`" assumes
tuning the situation changes retrieval, but a **fact** stores the situation twice (frontmatter +
the body "formula") and only the body is embedded/hashed — a frontmatter-only edit is a retrieval
no-op and invisible to `embed apply --stale`. Superseded: per-pass write-time synthesis is replaced
by recall-time lazy-L2 — the write-sparsity that starved AutoK no longer applies, and INV-S2's
frontmatter/body desync is resolved by `engram amend` (which rewrites both copies + re-embeds).

---

## ADR-0006 — Embed source by kind: episodes embed `situation`, others embed body

**Status:** Superseded — episode type retired (`engram learn episode` removed in 2026-06-19 cleanup); `embed.Text` now embeds body for all note types. E4/E5 defects are resolved by retirement. The `situation` field is still authored in fact/feedback bodies but is no longer a routing key for embedding.

**Context.** Episodes are retrieved by **situation** (the task you were doing — the recall-mirror);
facts/feedback are retrieved by their content.

**Decision.** `embed.Text` routes `type:episode` → the `situation` frontmatter field; every other
kind → the body (`hash.go:48-72`).

**Consequences (historical — this ADR is superseded).** Episodes matched task-shaped queries the way recall phrases them. ✅ RESOLVED (E4, E5) by the 2026-06-19 episode retirement: `embed.Text` now embeds the body for every note type, so neither the staleness-hash-vs-`situation` mismatch (E4) nor the empty-`situation` body fallback (E5) can occur.
⚠ KNOWN (M5, FAIL): fact/feedback retrieval also leans on `situation` (it is rendered into the body
formula and feeds recall-mirror), yet the CLI marks `situation` `required` only for episodes — an
empty fact/feedback situation is unguarded (census-clean 107/107). This is the FAIL-class
situation-presence invariant's architectural home.

---

## ADR-0007 — The wikilink graph is authored and walked by the binary; dangling links dropped

**Status:** Accepted (known defect: G0; G5 RETIRED — episode kind removed, `[[x]]` in chunk bodies no longer parsed as vault edges, per memory-invariants.md)

**Context.** Navigation should live in **authored relations** (wikilinks in note bodies), not a
separate graph store that can drift. Recall expands a subgraph from direct hits to find clusters
and hubs.

**Decision.** `vaultgraph.ScanVault` parses wikilinks at scan time; `BuildGraph` builds a directed
graph **keyed by basename**; recall does a 3-hop BFS (cap 200) + in-degree top-5 hubs. Dangling
targets are silently dropped at build.

**Superseded by recall-v2 / the 2026-06-20 deep clean:** the subgraph/hub path was removed; `vaultgraph` is now used only by `check`/`amend`, not in the query path.

**Consequences.** The graph is derived and always fresh. ⚠ KNOWN (G0): `learn` writes relations as
**bare Luhmann ids** (`[[105]]`) but `BuildGraph` resolves by **basename** — 155/183 link-instances
unresolved (151 of them bare-id), 138/171 notes orphaned, mean out-degree 0.16, so recall's graph expansion runs on a
near-empty graph. (G5 — verbatim `[[x]]` strings inside chunk bodies becoming false edges — is **RETIRED**:
the episode kind was removed and chunk bodies are no longer parsed as vault edges.)

---

## ADR-0008 — Per-arc episodes as the L1 evidence layer

**Status:** Superseded — episode type retired; `engram learn episode` and `engram transcript` removed in 2026-06-19 cleanup. Chunks ingested via `engram ingest --auto` are now the L1 evidence layer, referenced from facts/feedback via `--chunk-source`. · commits 98c962ea, b4e24f76, 4901bf78

**Context.** "What did we do yesterday" needs the literal interactions — tool calls, file paths,
the back-and-forth — not a narrative summary. A session interleaves multiple arcs of work.

**Decision.** Write **one episode per work-ARC** (a coherent thread; may be non-contiguous and may
overlap other arcs). The body is the noise-filtered transcript chunk, assembled from one or more
**repeatable** `--from-transcript-range` spans. Facts/feedback derived from a chunk link back via
`--relation "<episode-luhmann>|extracted from this chunk"`. Provenance stores the **resolved**
transcript file path (cwd-independent).

**Consequences.** High-fidelity recall of prior sessions; avoids both failure modes (one giant
session-spanning episode; losing the interactions). Episodes bypass the fact/feedback machinery
(no locus classification, no recall-mirror test) — they are retrieved through the situational
stream, not phrase-matching.

---

## ADR-0009 — Marker forward-progress: strict-greater, intra-session split, multi-source independent

**Status:** Superseded — `engram transcript --mark` and the `learnmarker` package retired in 2026-06-19 cleanup; marker logic subsumed into `engram ingest --auto`. M2-segments defect retired with the `--segments` path. · commits 4901bf78, 5c16c784

**Context.** `engram transcript --mark` must visit every learnable row **exactly once** across
runs — never skip, never re-emit forever — across multiple harness sources (Claude `.jsonl`,
OpenCode SQLite) and even within a single oversized session.

**Decision.** A per-`(project, source)` RFC3339 marker. Scan **strictly after** the marker within
a byte budget; on mid-session truncation (Partial) advance to the **last included row's**
timestamp, else to the session Mtime; **never advance past the earliest row not read** this run;
sources advance independently.

**Consequences.** Resumable, multi-source-safe forward progress; intra-session splitting lets an
oversized session be consumed across runs. The former M2-segments defect (the `emitSegments` /
`engram transcript --segments` path over-advanced the marker on truncation) is **retired** — the
`--segments` path was removed with the episode/transcript surface in the 2026-06-19 cleanup, so it is
no longer reachable.

---

## ADR-0010 — Sessions are read behind reader/finder interfaces; a composite dispatches across backends

**Status:** Superseded (partial) · the OpenCode SQLite backend (`internal/transcript/opencode.go`) was deleted in the 2026-06-20 deep clean — `git log` recovers it; only the JSONL reader (`internal/transcript/transcript.go`) survives, wired by `engram ingest`

**Superseded (partial):** The OpenCode SQLite backend (`OpencodeTranscriptReader`, `OpencodeSessionFinder`, `CompositeSessionFinder`, `CompositeTranscriptReader`) was never wired into production ingest and was removed in the 2026-06-20 deep clean. Engram reads JSONL only (`~/.claude/projects/<slug>/*.jsonl`). The `JSONLReader`/`Finder` interfaces remain as the sole production path.

**Context.** Engram must read session transcripts from more than one harness — Claude Code stores
them as per-session `.jsonl` files; OpenCode stores them in a SQLite database. The marker,
byte-budget, noise-strip, and emit logic must not care which backend a session came from.

**Decision.** Define `Finder` (locate sessions) and `Reader` (read rows) interfaces. Provide two backends — `JSONLReader` + `SessionFinder` (Claude) and `OpencodeTranscriptReader` + `OpencodeSessionFinder` (OpenCode SQLite) — plus a `CompositeSessionFinder` / `CompositeTranscriptReader` that wrap a list and dispatch to the **first backend that succeeds** (as originally implemented in the now-removed `opencode.go`, first-success dispatch). `engram ingest` wires the composite over both backends; the `SegmentsFrom`/`SegmentsReader` path and the `--segments` flag retired with the episode surface (ADR-0008/ADR-0009).

**Consequences.** Marker forward-progress (ADR-0009), stripping, and emit are backend-agnostic —
they run on the composite, never on a concrete backend. Session-id **scheme** dispatch (bare UUID →
Claude `.jsonl`; `opencode://…` → SQLite) is part of the same seam. Adding a third harness is an
interface implementation, not a change to the read pipeline.

---

## ADR-0011 — Controlled-vocab tag nomination over graph traversal

**Status:** Accepted (2026-07-02/03) · supersedes graph-traversal (PPR / spreading-activation) as
the relational-retrieval mechanism. **Superseded (2026-07-29) by ADR-0025** — tag nomination's
candidate-side "join the pool on shared vocab term" mechanism is replaced by centroid-proximity
explore sampling; this ADR's context/decision/consequences below remain the historical record of
why candidate-side augmentation beat graph traversal in the first place.

**Representation update (2026-07-10, #678, per ADR-0019):** the fixed term set now lives as bare-`vocab`-tagged definition fact notes (`vocab-<term>-definition`) and member terms as `tags: [vocab/<term>]`. The centroid-assignment and query-nomination mechanism is unchanged.

**Representation update (2026-07-27, openspec change `vocab-definition-self-tags`):** per-term definition notes additionally carry their own `vocab/<term>` self-tag (display-only — connects each definition to its member cluster in tag-based views; the family note stays bare-`vocab`-only). Member semantics unchanged: assignment, stats/trigger math, and tag nomination all exclude definitions via the bare marker.

**Context.** The wikilink graph (ADR-0007) is authored and walked by the binary, but resolves by
basename against bare-Luhmann-id links — most edges never resolve (⚠ KNOWN G0/G5) — and even a
healthy graph leaves open how a relational miss (a note topically related to the matched set but
never phrase-matched) should be recovered at query time. Two mechanisms were evaluated head to
head: ranking-side graph traversal (PPR / spreading-activation / one-hop expansion) vs.
candidate-side nomination through a controlled vocabulary.

**Decision.** Reject graph traversal as the retrieval mechanism. Ship controlled-vocabulary tag
nomination: a fixed term set (`vocab.<term>.md` — representation as of 2026-07; see #678), dual-channel
term assignment at every learn/amend/resituate write (representation as of 2026-07; see #678), and at
query time a note sharing a vocab term with the top-3
delivered notes in a cluster is nominated into that cluster's `candidate_l2s` alongside the
within-cluster top-5 (budget fields `tag_nominations_added`/`dropped` report pool size). A typed
`--supersedes` flag (`updates`/`narrows`/`refutes`) lets a note carry an explicit edge to an older
one, surfaced as a ride-along at the next candidate rank.

**Consequences.** PPR/spreading-activation is ⛔ KILLED on this vault — it drops non-activated
baseline notes and regressed collateral notes; one-hop expansion reproduced the same settled null
(`dev/eval/LEDGER.md#ppr-killed`). Tag nomination recovered a majority of verified retrieval misses
with zero collateral and moved blind delivery above the noise floor, most on cross-domain bridges
(`dev/eval/LEDGER.md#vocab-tag-nomination-l6xtag` owns the figures) — link/tag
value pays where vocabulary is remote from task phrasing (bridges), not on single-hop misses.
Migration classified the pre-existing 84 "Related to:" edges against pinned criteria: 7 true
supersessions, 77 dropped as a non-supersession link *type* (76 thematic/cross-reference/sibling,
1 dangling) — full disposition table in `docs/design/artifacts/2026-07-02-retired-relation-rationales.md`,
deleted 2026-07 with the docs restructure; `git log` recovers it.
Typed supersession ride-along is mechanism-proven but fabric-starved (few edges qualify as true
supersessions). The wikilink graph itself is unaffected: it remains authored-and-walked only by
`check`/`amend` (ADR-0007), never by the query path.

---

## ADR-0012 — D5′ asymmetric QA participation

**Status:** Accepted (2026-07-03) · supersedes D5 (full QA exclusion)

**Context.** An earlier design (D5) proposed excluding all QA-derived notes from the main matched
set, treating captured question/answer pairs as a channel apart from facts/feedback. That
treats a qa-answer and a qa-question identically, but they are not: a qa-answer is a synthesized,
pre-reasoned conclusion with provenance, while a qa-question is situational wording that
measurably loses retrieval against content-bearing notes (the question-anchored-crystallization
finding — no delivery benefit and 10/10 retrieval lost when notes were re-anchored to their
question; `dev/eval/LEDGER.md#qanchor-park`, vintage 2026-07-01, ⛔ PARKED).

**Decision.** qa-answer notes COMPETE in the main matched set on the same standing as any other
fact/feedback note. qa-question notes are EXCLUDED from the main set at all four
query-pipeline seam points (`isQueryExcludedKind`) and are reachable only via a dedicated q-space
channel with an `answered_by` ride-along — deferred to round 3, gated. Decision record:
`docs/design/2026-07-03-qa-memory-proposals.md` (deleted 2026-07 with the docs restructure —
`git log` recovers it).

**Consequences.** Round-1 (capture) shipped 2026-07-03: `engram learn qa`, D5′ exclusion,
`stripMachineLines` QA markers, `qa pairs:` / `qa round-2 gate:` lines in `engram vocab stats`.
**Caveat carried forward to round-2/3 gating:** D5′'s asymmetry rests on n=5 synthetic pairs
(source: the decision record above, vintage 2026-07-03), not yet re-validated at corpus scale —
round-2 gates on ≥20 real captured pairs (or ~2026-07-17, whichever comes first) against
pre-registered bands (PASS ≥8, BORDERLINE 6–7, FAIL <6; P2′/P3′ definitions, `docs/ROADMAP.md` →
GATED — Q&A memory round-2/3). The dedicated q-space channel needs its own premise check (Arm V) to reach PASS (≥80%)
before round-3 is licensed; Arm V large-n came in BORDERLINE 63% (19/30)
(`dev/eval/LEDGER.md#qa-arm-v-borderline`, vintage 2026-07-03), so round-3 remains unlicensed
pending a further check.

---

## ADR-0013 — Vault flock + atomic-rename write safety

**Status:** Accepted (shipped 2026-07-01, commit `f7f6b389`; closed #660 + #666; #700 (2026-07): flock/atomic-rename lifecycle composed in `internal/cli` — `primFS`/`primLocker` over raw os/syscall primitives supplied by `cmd/engram` — semantics unchanged, lock-at-`Run*`-entry convention preserved, concurrent-writers regression test carried, now an `internal/cli` real-FS integration test)

**Context.** The planned payload-prune production build spawns many parallel sub-recalls that
write the vault and chunk index concurrently. Before this fix, only `learn`'s Luhmann-ID
sequencing was flock-protected (`writeLearnUnderLock`, `learn.go:571`); `ingest`/`prune`'s manifest
read-modify-write, `amend`/`resituate`'s vault-note read-modify-write, and `activate`'s sidecar
rewrite were all unlocked, non-atomic writes (`os.WriteFile` assumed atomic — it is not). Any two
concurrent `engram ingest`/`amend` runs corrupted state, independent of retrieval quality or cost —
and this bit in production.

**Decision.** Extend the existing vault flock (`internal/cli/cli.go`) to every read-modify-write
writer: `.manifest.lock` guards `ingest`+`prune`'s manifest RMW; `.luhmann.lock` guards
`amend`+`resituate`+`activate`'s vault-note/sidecar RMW. Locks are acquired only at `Run*` entry
points; shared write helpers (`bumpLastUsed`, `writeManifestFile`, `reEmbedAndActivate`) stay
lock-free, to be called only by a `Run*` that already holds the lock (avoids self-deadlock). Every
writer's edge also gets one shared atomic-temp-rename helper, replacing bare `os.WriteFile`.

**Consequences.** `targ check-full` green plus a concurrent-writers regression test gate the fix
(no eval-ledger row — correctness is locked by the regression test, commit `f7f6b389`, 2026-07-01). Payload-prune production is
unblocked — the concurrency correctness ADR-0003 flagged as untested (⚠ KNOWN K1, "enforced in
code, untested as a property") is now enforced for every RMW writer, not just note+sidecar
creation. Deadlock-avoidance is a convention (lock at `Run*` entry points only), not a checked
invariant — a future writer that acquires the lock inside a shared helper reintroduces the risk.

---

## ADR-0014 — Memory-backed tier discount (route)

**Status:** Accepted (shipped 2026-06-28, commit `2bf959f4`; vault note 135)

**Context.** The `route` skill encodes engram's delegate-everything doctrine, including which
model tier to dispatch a unit of work to. Measured: sonnet+memory fully matched opus+memory across
C3 apply-conventions (15/15), C4i recency-supersession (3/3), and C6 abduction (6/6), while
sonnet *cold* failed the same axes — memory democratizes reasoning across model tiers rather than
only amplifying the strongest model.

**Decision.** Route by capability *tier*, not model name (the roster backing each tier can
change), and drop one tier for memory-backed units — a unit where the model applies recalled
knowledge rather than derives it from scratch is routed one tier cheaper than the same unit cold.

**Consequences.** RED/GREEN showed the router had been over-provisioning memory-backed units to
mid-tier before this rule; the discount corrects that (`dev/eval/LEDGER.md#tier-routing-parity` owns
the figures) and is the single largest whole-task-cost lever found to date — bigger than
any payload-byte-level cut (`dev/eval/LEDGER.md#payload-prune-smoke`). Bound: measured at the
deep→mid tier boundary only; other tier boundaries are inferred, not separately measured — the
existing upgrade-if-cheaper-fails rule is the safety net for a wrong discount. The C5
(recency-standard-honoring) axis flaked in this measurement round and was not re-run.

## ADR-0015 — Skill decomposition stops at the write seam

**Status:** Accepted (2026-07-04)

**Context.** The atomic-skills exploration evaluated decomposing the five skills (recall, learn,
write-memory, please, route) into shared behavioral atoms — read-memory, write-memory, route-a-task,
orchestrate — to remove overlap without producing N skills that all do the same thing.

**Decision.** Extract exactly one atom: `write-memory`, a worker invoked at the write seams (recall
and learn hand off; the worker composes, executes, verifies, reports). Do NOT extract read-memory —
recall's read+judge+write pipeline is sequential cohesion worth keeping. Leave `please` and `route`
untouched (route already maps 1:1 to its atom). A skill-share is a worker invoked as the next whole
action, never a mid-procedure reference fetch. Decision record:
`docs/design/2026-07-04-atomic-skills-options.md` (deleted 2026-07 with the docs restructure —
`git log` recovers it).

**Consequences.** Five skills remain; the worker pattern is the sanctioned shape for future skill
shares. The interim reference-card variant's "0/27 mid-procedure dereference" measurement is
instrument-invalid and binds nothing (`dev/eval/LEDGER.md#write-memory-atom-dereference-invalid`,
vintage 2026-07-04); the worker form's fire-rate validation is
`dev/eval/LEDGER.md#write-memory-worker-fire-rates` (vintage 2026-07-04).

---

## ADR-0016 — Architecture diagrams are hand-authored mermaid, verified against code

**Status:** Accepted (2026-07-05)

**Context.** A deployed user-level `c4` skill exists for generating C4 diagrams, but its mechanism is
JSON source specs under `architecture/c4/` rendered/audited by a `targ c4-audit` target — none of
which has any footprint in this repo (no JSON specs, no such targ target).

**Decision.** Keep the C4 diagrams (`c1`/`c2`/`c3`) and the feature flow diagrams as hand-authored
mermaid living in `docs/architecture/`, each verified directly against the current code. Do NOT adopt
the `c4` skill's JSON-spec pipeline here: a path-only move to `architecture/c4/` would satisfy only the
skill's directory convention while leaving its audit half unmet, faking compatibility.

**Consequences.** Diagram currency is maintained by direct code review at edit time (as this
restructure did), not by a generator. Adopting the skill later would be a deliberate migration
(JSON re-derivation of every diagram + a new targ target), not a file move.

---

## ADR-0017 — Evidence-based route rubric

**Status:** Accepted (2026-07-06). Extends ADR-0014.

**Context.** ADR-0014 established route-by-tier plus the memory tier discount, measured only at the
deep→mid boundary; the remaining tier assignments lived in a hard-coded task-character table
(mechanical→cheap, moderate→mid, complex→deep) that was asserted, never measured. A RED baseline
confirmed the cost: reading the old skill, an agent over-provisioned 5/6 generic units to mid/deep by
surface look, and a no-guidance control did the same — the table encoded the model's untested "this
looks hard → strong model" instinct.

**Decision.** Make the rubric memory-based, not a fixed table. Every unit starts at the cheapest
tier; only recalled evidence — or a failed review — raises it. Perceived difficulty is not evidence
("genuinely hard" and "looks hard" are the same hunch), so there is no cold-start exception for
hard-looking work. Failures escalate spec-first: the first fail rewrites the handoff and retries the
same tier, a second fail escalates one tier. Every dispatch is recorded (work-kind, tier, concrete
model, why, review-sourced outcome); records auto-ingest as recallable memory and crystallize via
`/learn`, so the effective rubric improves through the record→learn→recall loop rather than by
editing the skill. The memory tier discount (ADR-0014) survives as the one evidence-backed entry of
the cold-start priors.

**Consequences.** An initial draft failed: it merely said "don't upgrade on a looks-hard hunch" but
still routed perceived difficulty to the deep tier. Adding an explicit "genuinely-hard ≠ evidence"
test closed the loophole, and writing-skills TDD then flipped 5/6 over-provision to 0/6 (everything
cheap). Cold-start cost rises for genuinely-hard work (up to two failed escalations before it reaches
the tier it needs), an accepted bootstrap; the evidence loop recovers this overhead by improving the
rubric for future similar work. Harness-agnostic: the dispatch record is built from what the
orchestrator already knows plus the review verdict, needing no per-subagent cost/token telemetry. The
pure-confirmation signal ("cheap sufficed for K", overturning nothing) is captured by `/learn`'s
kind-4 confirmed-approaches moment (positive reinforcement, shipped 2026-07-06 via #668): a tier
that passed for a work-kind crystallizes as a confirmed approach, a tier that failed as a reversal. **Deferred:** structured routing-evidence ledger (#669 — resolved 2026-07-10: bespoke store subsumed by
tags-based evidence notes + evidence-linked aggregate fact notes read via plain recall, with `engram
count` as the audit surface; build re-scoped into #674, vocab migration #678 shipped 2026-07-10), periodic
rubric-refit (#670), parallel-builders (#671), cost/duration telemetry (#672). RED/GREEN evidence is
transient (`git log`); the memory-discount figures remain at `dev/eval/LEDGER.md#tier-routing-parity`.

---

## ADR-0018 — Counting/aggregation is a surface distinct from similarity recall

**Status:** Accepted (shipped 2026-07-08)

**Context.** `engram query` answers "what's relevant to this phrase" via recency-biased cosine
similarity over a bounded, truncated matched set (ADR-0004) — by design a fuzzy-ranked *sample*,
not a complete enumeration. A question like "how many notes carry the `vocab/retrieval-design` tag" or
"how many notes link to `[[foo.alpha]]`" needs an exact, typed count over the *whole* vault, not a
top-N approximation — riding it on the recall path would silently return a truncated, similarity-
ordered subset with no signal that it isn't the real count.

**Decision.** Ship `engram count` (`internal/cli/count.go`) as a read-only counting/aggregation
surface, deliberately off the recall/similarity path — it never embeds, never scores cosine, never
clusters. Two mutually exclusive modes: `--group-by <attr>` counts DISTINCT note membership per
frontmatter-attribute value (a scalar attr contributes its one value; a list attr contributes one
per distinct element — a note listing a value twice still counts once), optionally restricted by
repeatable AND-ed `--filter attr=value` predicates (scalar equality or list-contains).
`--backlinks-of <basename>` prints the wikilink in-degree (plus sorted linkers) of a vault-graph
node via `vaultgraph.ScanVault`/`BuildGraph` (ADR-0007). The design goal is **per-mode
Obsidian-verifiability**: each mode is independently hand-checkable against its own Obsidian view —
`--group-by` against a frontmatter/property/tag filter (or Dataview), `--backlinks-of` against the
note's backlinks panel.

**Consequences.** The two modes are **not** interchangeable — they measure different things and
legitimately diverge by the count of *non-member linkers*: an index/MOC page (e.g.
`vocab.index.md`) links `[[vocab.<term>]]` for every term without carrying that term in its own
`vocab:` frontmatter, so it counts toward `--backlinks-of` but not `--group-by`. The relationship:
`backlinks-of(node) == group-by(attr) count for that value + (# non-member linkers)`. Verified on
the live vault: `--group-by vocab` counts 33 members for the value `retrieval-design`;
`--backlinks-of vocab.retrieval-design` reports in-degree 34 — the +1 is `vocab.index.md`
(example vintage 2026-07-08; `vocab.index.md` retired 2026-07-10 under #678 — vocab terms no longer
produce wikilink edges, so `--backlinks-of vocab.<term>` reads 0; the divergence relationship itself
remains valid for any non-member linker). Do not
report count parity as unqualified "count == Obsidian backlinks"; state per-mode verifiability plus
the divergence relationship, or the two modes read as redundant when they are complementary.
`TestRunCount_GroupByBacklinksAgreement` locks the clean-fixture case (no non-member linkers, so the
two agree); `TestRunCount_BacklinksExceedGroupByForNonMemberLinkers` locks the divergence case.

---

## ADR-0019 — Tags are the sole categorical representation; recall reads, count audits

**Status:** Accepted (2026-07-10 — Joe's decision recorded on #669; shipped via #674)

**Context.** Route's dispatch records were free transcript text — recallable as fuzzy chunks but
not aggregable (ADR-0017's deferred ledger, #669). ADR-0018 shipped `engram count` as a general
aggregation surface. The overlap needed one representation for low-cardinality categoricals
(work-kind, tier, outcome) that recall, counting, and Obsidian can all read without a bespoke
store.

**Decision.** Frontmatter `tags:` — a plain YAML string list written by the repeatable
`engram learn --tag <family>[/<value>]` flag (fact/feedback only; not qa, not amend, though amend
round-trips an existing list) — is the **sole** categorical mechanism: no attr nodes, no
categorical wikilinks, no bespoke tables (#676 closed moot; #669 closed subsumed). Three note
roles ride on it: **evidence notes** (one per route dispatch, tagged `work-kind/<k>`, `tier/<t>`,
`outcome/<o>`; ordinary recallable facts — no query exclusion); **aggregate notes** (one per
work-kind, slug `route-evidence-<work-kind>`, object text = tier tallies + wikilinks to every
summarized evidence note; amended per dispatch; untagged); **family definition notes** (bare
family tag = definition, nested `family/value` = member; tier: cheap|mid|deep, outcome:
pass|fail, work-kind: open kebab-case set). Route's read path is **plain recall** — aggregates
surface as normal memories. `engram count --group-by tags --filter tags=...` is the **audit**
surface: it recomputes true tallies from evidence tags to verify/repair the LLM-maintained
aggregates, and is never on the read path. (`--group-by work-kind` does not apply — work-kind is
a tag value, not a frontmatter attribute.)

**Consequences.** LLM-maintained tallies WILL drift; count makes them falsifiable (audit commands
live in `agent-instructions/skills/route/SKILL.md`). Evidence notes stay in recall — excluding them would regress on
the already-recallable free-text records they replace. The aggregate-drowning risk (many
near-identical evidence notes outranking their aggregate on the read query) is gauged, not
pre-engineered: a scratch-vault gauge (20 sibling evidence notes + 1 aggregate; PASS = the
aggregate's path appears in the read query's items or candidate_l2s) passed 2026-07-10, and the
same check is documented in the route skill as a standing drowning audit. **Pre-registered
follow-up** if drowning is ever measured on the real vault: (a) a "summarizes" ride-along edge
(supersession-shaped insertion of the aggregate when its evidence surfaces) or (b) demoting
evidence notes to the chunk-population ranking tier — choose with the measured case in hand, per
the standing rule that a new edge type must first demonstrate retrieval value (ADR-0011;
`docs/ROADMAP.md` → Standing constraint; vault note 73). Vocab's hub-note channel migrated to this tags
convention 2026-07-10 (#678): definitions are recallable bare-`vocab`-tagged fact notes,
`vocab_version` lives on `vocab-definition`, and the vocab query exclusions are deleted.

---

## ADR-0020 — Enforced `internal/` purity: raw I/O assignment in `cmd/engram`, all logic in `internal/`

**Status:** Accepted (shipped via #700, 2026-07)

**Context.** The DI doctrine ("wire at the edges" — CLAUDE.md's summary bullet, under
ADR-0001..0003's authority) was convention-only: production I/O adapters lived inside
`internal/cli`, `internal/debuglog`, and `internal/embed`; direct env reads had crept in (the
#700 FIXME); and testing internal code meant working around real I/O. Meanwhile cmd thinness
(targ's `check-thin-api` gate) forbids moving real adapter logic into `package main`.

**Decision.** The boundary is absolute and two-sided. `internal/` non-test code holds interfaces
plus ALL logic — adapter composition, error wrapping, lifecycle (the EdgeFS atomic-write dance,
flock open/lock/unlock-closure semantics, the debug sink, signal force-exit, commander
run-and-collect, embedder session/cache orchestration — built by `cli.NewDeps` from injected
`cli.Primitives`) — but imports no I/O packages. `cmd/engram` (`package main`) is
wiring-only: a single-statement `main()` composing `cli.Primitives` from checker-thin
per-capability-group functions (each returning its composite literal directly, no carrier call)
that populate raw capability references (`os.ReadFile`, `time.Now`,
`filepath.WalkDir`, syscall wrappers) and sanctioned closures (single-call signature-erasers
plus exactly two multi-statement stdlib-equivalent closures: C-1 `RunCommand` and SIG-1
`StartSignalPulses`) — zero orchestration. Enforcement is config-only and
two-gate: on the internal side, a depguard default-deny allow-list over `internal/` non-test
files (zero file carve-outs; real-os integration tests live in internal `_test` files via the
sanctioned `!$test` exclusion) plus forbidigo call-level bans (`time.Now`/`Since`/`Tick`,
math/rand v1, auto-seeded rand/v2 globals, `targ.Main`); on the cmd side, `targ check-thin-api`
(authoritative).

**Consequences.** Every internal package is testable by injection alone — unit tests with fake
primitives, real-os integration tests as internal `_test` files. A new I/O capability requires a
`Primitives` field plus internal composition, both visible in review. Both gates fail loud on
regression. `cmd/engram` carries no testable logic and stays coverage-exempt as an entry point.
Seeded `math/rand/v2` stays legal (deterministic computation).

## ADR-0021 — Chunk-index dedup by content hash + canonical precedence; the record-subset eviction gate; update notifies rather than acts

**Status:** Accepted (shipped 2026-07-25/26)

**Context.** `engram ingest --auto` sweeps every known source (repo markdown, ancestor `.claude`/`.pi`
dirs, session logs) and, before this cycle, indexed each swept path independently — one `.jsonl`
index file + manifest entry per path, with no cross-source view. The same content reachable from
more than one path (a project file and a copy of it swept in from an ancestor `.claude` dir; a
session transcript copied into a worktree; a vault note snapshotted into a job's scratch dir) was
therefore indexed once per copy. Measured on a real ~62,000-source manifest: 79% of sources were
exact-content duplicates of another source, and duplicated content made up a real share of
`engram query`'s returned items (9.6% mean, 28.6% worst-case in a lazy-chunks production sample).
A path-name blocklist (excluding known throwaway roots) cannot reach most of this, because the
duplication comes from legitimately-swept roots (worktrees, ancestor `.claude` dirs, backup/restore
trees), not from a fixed set of junk directories — one nameable exception is `.claude`'s own
`jobs/` scratch subdirectory (decision 4 below), which the `.pi` ancestor sweep already excluded
and the `.claude` sweep, until this cycle, did not.

**Decision.**

1. **Dedup key: content hash + chunking class.** Sources are grouped by `(FileHash, chunkingClass)`
   — never by hash alone. A `.md` file and a `.jsonl` file can share identical bytes yet must never
   be treated as duplicates of each other: `chunkSource` dispatches on extension and produces
   genuinely different, non-empty chunk records for each (markdown chunks the literal bytes;
   a transcript is stripped into `USER:`/`ASSISTANT:`-shaped text first). Groups are additionally
   partitioned so this boundary can never be crossed by hash-match alone.

2. **Canonical precedence, not arrival order.** Within a hash group, `selectCanonical` picks the
   member to actually index, first match wins: (1) explicit `--transcript`/`--markdown`/
   `--pi-sessions` sources, (2) the repo-markdown root, (3) an ancestor `.claude`/`.pi` dir —
   closest ancestor first, (4) anything else (session logs, manual `--sweep`, extra roots); ties
   break on fewest path separators then byte-wise lexicographic path. This is a pure function of
   the candidate set (order-independent — `TestSelectCanonicalIsOrderIndependent`), so the same
   group always resolves to the same canonical regardless of walk order. Every other group member
   is recorded as a duplicate (`manifestEntry.DuplicateOf`) and never indexed; a later run does not
   re-read a known duplicate to re-derive the same answer.

3. **Eviction, not just skipping, on a later higher-precedence arrival.** A low-precedence copy
   indexed on day 1 and a higher-precedence twin appearing on day 5 must converge: `engram ingest`
   removes the stale copy's index file + manifest entry (index file first, then the manifest entry
   — the reverse order would strand an unreferenced index file nothing cleans up) and re-indexes
   under the new canonical.

4. **Restoring `.pi`/`.claude` sweep parity — stop duplication at the source.** The `.pi` ancestor
   sweep already excluded a `jobs/` subdirectory from its walk (agent-harness scratch — the same
   subdirectory shape exists under both harnesses); the `.claude` ancestor sweep never carried the
   matching exclude, so `~/.claude/jobs` — which can hold whole snapshot copies of the vault — was
   swept and indexed like any other project content. `ClaudeExcludeDirs` now carries the same
   `jobs` exclude the `.pi` side always had, restoring parity between the two sweeps. This
   complements rather than substitutes for the hash-based dedup above: excluding `jobs/` stops one
   large, nameable source of duplication before it is ever indexed, while content-hash dedup
   remains necessary for duplication the sweep configuration cannot name (worktrees, ad hoc
   backup/restore trees, deliberate `--sweep`/`ExtraRoots` overlaps).

5. **The record-subset eviction gate — REQUIRED, not an optional safety margin.** The original
   premise — "byte-identical sources are interchangeable, so it's always safe to keep one and
   delete the other's index" — is FALSE, and was falsified during this cycle's own ship-readiness
   review, not assumed correct going in. `mergeChunkRecords` is append-only WITHIN one source: a
   source's `.jsonl` index file accumulates every chunk record from every past ingest of that path,
   not just its current content. Two sources can therefore be byte-identical RIGHT NOW (same
   `FileHash`, same hash group) while their index files hold entirely disjoint historical chunk
   records — deleting one on the strength of the other's mere existence can silently destroy real,
   unique content. So eviction (both the forward, ingest-time case and the retroactive
   `prune --duplicates` case) additionally requires, immediately before removing a duplicate: the
   retained canonical's index file exists **right now**, AND every one of the duplicate's own index
   records (by content hash) is already present among the canonical's. When this cannot be
   confirmed, the removal is refused rather than performed, and the refusal is reported (bulk-
   summarized for the common structural case — the canonical has no index file at all, a
   zero-chunk source with nothing to lose — and named individually for the rarer anomalous case,
   where a sibling's surviving index proves the group holds real content). The risk is not
   hypothetical: measured on the real corpus, 124 of 6,612 surviving index files carry chunk
   records from more than one ingest day — exactly the shape a byte-hash-only gate cannot
   distinguish from a safe-to-delete duplicate. The weaker, existence-only gate (a canonical index
   file merely existing, with no check on its contents) shipped earlier in this same cycle and ran
   once, live, against the real chunk index before this stronger gate replaced it, removing 1,960
   index files (commit `cb6b9540`). **This is no longer an open question — a forensic
   reconstruction resolved it.** Using a pre-migration manifest snapshot and file listings
   preserved from that run (session scratch), cross-checked against the current live manifest and
   on-disk state, every one of the 1,960 removed entries was classified into an exact partition:
   **1,568 regenerable** (source present, unchanged — a later re-ingest reproduces the identical
   content, nothing to recover). **380 with sources no longer present at their original path**:
   280 `/tmp` eval-harness fixtures; 52 duplicate Claude Code subagent-workflow transcripts
   recorded under two different project-slug directories for the same session (verified directly
   — several such pairs still exist on disk today, byte-identical at matching relative paths under
   both a main-repo project slug and a git-worktree's own project slug for the same session id);
   46 repo files this repo's own earlier refactors had since relocated; and 2 deployed
   `~/.claude/skills` copies whose content survives under a sibling file (unrelated to the 2
   orphans below — these 2 do have surviving twins). **4 with sources whose content has changed**
   since removal (neither gone nor unchanged). **8 unresolved**: the same subagent-workflow-
   transcript naming pattern as the 52 above, but neither the pre-migration manifest snapshot nor
   the current live manifest contains a source path whose slug matches any of these 8 filenames
   (checked exhaustively — zero slug collisions across all 62,397 pre-manifest entries) — so,
   unlike the 52, a surviving twin for these 8 is **not** independently confirmed here; they are
   reported as an unresolved residual rather than asserted as safe. 1,568 + 380 + 4 + 8 = 1,960
   exactly.

   Of those same 1,960, **exactly 2 have no surviving byte-identical twin anywhere in the current
   index** — a property of two specific entries, not an additional bucket: one of the 46 relocated
   repo files (`skills/route/SKILL.md`, moved by #697's `agent-instructions/` restructure) and one
   of the 4 changed-content entries (an ancestor `.pi` sweep copy of the same file,
   `/Users/joe/.pi/agent/skills/route/SKILL.md`, whose source path still exists but has since been
   overwritten with newer content rather than removed). Both are the same lost content — a
   superseded draft of the route skill (content hash `sha256:de575213…`) — recoverable from git
   history (`git show eb538e05:skills/route/SKILL.md`, verified to reproduce the exact removed
   bytes). Commit `cb6b9540`'s "0 unsafe removals" claim is **superseded by this finding**: that
   audit ran under the existence-only gate this decision replaces, so it measured the wrong thing —
   a duplicate's mere presence under a surviving canonical, not whether every one of the
   duplicate's own chunk records (its accumulated history, rule 6 below) was actually covered. It
   happened to be right for 1,958 of the 1,960 (plus the 8 unresolved, not independently
   re-checked here); it was wrong for these 2. The honest caveat that remains: finer-grained loss
   *within* the 1,568 regenerable entries — a duplicate's accumulated history holding a record its
   canonical twin never had — cannot be quantified without the deleted bytes; a read-only recheck
   of the current, post-fix index (no eviction performed) found 0 of 9 residual duplicate groups
   holding a record their canonical twin lacks, so it is empirically rare in this corpus, but it is
   not provably zero.

6. **The cross-source "never delete" guarantee is narrowed, not reversed.** Two distinct guarantees
   previously hid under one phrase ("append-only, never deletes"). *Within* a source
   (`mergeChunkRecords`), the guarantee is unchanged: a re-chunk never drops a prior record.
   *Across* sources, the guarantee is now: a duplicate's index file may be removed, but only when
   it is provably a subset of a retained twin's — so no content ever becomes unsearchable. The
   user-facing promise — "deleting a source file never loses the recovered memory" — still holds
   exactly; what changed is that a byte-identical *second copy* of a still-present source may no
   longer carry its own index file once dedup runs.

7. **Rule B — drifted vault copies are dropped, not deduped.** A swept `.md` file with a sibling
   `.vec.json` sidecar is a vault-note copy (the vault is 1:1 `.md`/`.vec.json`). Exact-content
   dedup (rules 1-3 above, "Rule A") cannot reach a *drifted* copy — a vault note amended since a
   snapshot was taken hashes differently from the live note, so both would independently survive
   dedup as distinct "canonical" content. Rule B instead drops any such file found outside the
   resolved vault path, unconditionally, before dedup ever runs — measured live: 50 of 256 shadowed
   vault-note copies had drifted from their live original. Rule B is vault-note-specific; a drifted
   copy of a non-vault document (e.g. an edited fork of a doc in a worktree) is out of scope for
   both rules and is not addressed by this ADR.

8. **`engram prune --duplicates` — retroactive cleanup, not automatic.** The forward-looking dedup
   above only prevents new duplication; it does nothing for the backlog already indexed before it
   shipped. `engram prune --duplicates` (+ `--dry-run`) re-derives the live duplicate set from the
   manifest at run time and applies the same canonical-selection + record-subset gate to clean it
   up, convergently (a second run removes nothing).

9. **`engram update` notifies; it does not run the cleanup itself.** An earlier design considered
   having `engram update` run the retroactive prune automatically, once, on upgrade (a sentinel
   file marking it done). This was reversed: `update` now only detects whether a live duplicate
   group exists in the manifest (one cheap read + grouping pass, no per-file index opens) and
   prints a one-line notice pointing at `engram prune --duplicates` and the README Upgrading
   section — the same notify-only convention `update` already uses for the #694 empty-file
   backlog. Reasoning: removing index files is a destructive, one-shot migration over a user's
   memory store, and — unlike the empty-file case, which is unconditionally safe (empty files hold
   zero records) — a duplicate removal's safety depends on the record-subset gate holding at the
   moment it runs. A destructive migration should run only when the user asks for it, not silently
   as a side effect of a routine binary/skills refresh. Detection is stateless: no sentinel file,
   because there is nothing to mark "done" — the check simply reports the live count truthfully on
   every run and goes quiet once the backlog is actually cleared. Refined by #713 (2026-07-27):
   detection now runs prune's own dry-run coverage gate, so the notice fires only when a removal
   would actually happen, and it names the command directly (the README Upgrading section it used
   to point at is removed).

10. **One-time canonical-reselection churn is expected, and is bounded.** `prune --duplicates`
    selects a group's canonical origin-blind, from whatever is in the manifest at that moment. A
    later `engram ingest --auto` may then discover a higher-precedence copy of the same content
    (e.g. the repo-markdown copy, swept after the retroactive prune already picked an ancestor
    `.claude` copy as canonical) and evict the prune's canonical in favor of the new one. This is a
    single, bounded re-selection per group when it happens — not an oscillation — because rule 2's
    canonical-selection is deterministic and rule 3 only evicts a *lower*-precedence member in favor
    of a *higher*-precedence one; there is no cycle back the other way.

**Measured bound.** On a real ~62,000-source chunk index, `prune --duplicates` removed 1,960 of
8,572 index files (−23%). Forensics on that removal attributed 1,397 of the 1,960 (71%) to the
single `~/.claude/jobs` tree — decision 4's sweep-side fix stops that specific source from
recurring, so a future retroactive prune on a freshly-ingested index should not need to repeat it
at anywhere near that scale. Most of the remaining redundancy in that corpus sits in orphan index
files with no manifest entry at all (index files left behind by prior ingests whose manifest entry
was later dropped or never written) — a manifest-keyed dedup pass has no way to reach those; they
are a distinct cleanup surface, out of scope here.

**Consequences.** Ingest is no longer safely describable as unconditionally "append-only, never
deletes" across sources — every doc surface making that claim (README, GLOSSARY, the L1 sequence
diagram, the `learn` skill) is narrowed to say so accurately. A duplicate's manifest entry now
carries an explicit `DuplicateOf` pointer rather than relying on "no index file" being an
unambiguous signal (which was indistinguishable from a crash-window state). `engram prune
--duplicates` and the forward dedup share one `removeDuplicateIndex` + record-subset-gate
implementation, so their safety guarantee is identical by construction rather than by
coincidence between two independent implementations.

## ADR-0022 — Deployment as sync: engram-owned roots with symlinked surfaces and removals-propagate semantics

**Status:** Accepted (shipped 2026-07-27 via #706)

**Context.** Before this cycle, `engram update` copied harness artifacts (skills, commands, guidance) additively into user directories (`~/.claude/skills`, `~/.claude/engram`, `~/.config/opencode/{skills,commands}`, `~/.pi/agent/{skills,guidance}`). Deletion existed only within an artifact still present in the source (e.g., clearing a skill's target dir before re-copy); an artifact deleted from `agent-instructions/` generated no copy operation and was never removed from any harness, so deployed state drifted from intent. A stray metadata file (not a skill, but deployed as one) persisted across updates until manually deleted. The absence of removal propagation also meant skill/command discovery (both harness-native and verification harnesses) could not distinguish intentional deployments from orphans.

**Decision.**

1. **One engram-owned artifacts root per harness.** Each harness gets a single root managed by engram: `~/.claude/engram/` (Claude Code), `~/.config/opencode/engram/` (OpenCode), `~/.pi/agent/engram/` (Pi), each containing `skills/`, `commands/`, `guidance/` subtrees as applicable. Real artifact files live **only** in these roots; every harness-visible surface outside a root is either a symlink into one or a user-owned file left untouched. This preserves the existing `~/.claude/engram` directory (already used for guidance staging), reuses it as the structured root, and keeps other harnesses' roots inside their own config trees for survivability under harness-tree relocation.

2. **Explicit ownership: the `.engram-owned` marker.** Every engram-owned root carries a marker file (`.engram-owned`, created at root creation or adoption). The sync-deletion engine runs only inside marker-stamped roots; a root lacking the marker is either not yet migrated (first-sync adoption path) or was explicitly unmarked by the user (fail-safe: refuses deletion, reports the state). This makes ownership inspectable and handles the edge case where a user happens to have a directory named `engram/` — the marker, not path convention alone, proves engram ownership.

3. **Symlink materialization, granular by artifact type.** Skills surface as one symlink per skill directory (`~/.claude/skills/recall` → `~/.claude/engram/skills/recall`); commands and guidance surface per-file (their discovery units are individual files, not directories). This matches the discovery unit granularity and keeps symlink counts low. The `Filesystem` interface gains `Symlink`, `ReadLink`, and `Lstat`-shaped type probing, wired via DI through `cli.Primitives` and `cmd/engram`'s checker-thin adapter functions (ADR-0020).

4. **Sync target is the intended deploy set, not the raw source tree.** The existing planners (`planSkillCopies`, `planCommandCopies`, `planGuidanceCopies`) compute the intended artifact set per harness, including the `--with-guidance` opt-in gate. The sync engine diffs the engram-owned root against that intended set: missing → create, different → overwrite, present-but-unintended → delete (removals propagate). Guidance opt-out means the root's `guidance/` subtree remains unmanaged and untouched; this preserves the `--with-guidance` opt-in semantics.

5. **Dangling-link cleanup via lexical target identity.** On every update, the sync engine scans each harness's surface directory one level deep. For each symlink, it resolves the link's literal target against the link's parent directory (lexical `filepath.Clean`, no `EvalSymlinks`) and prefix-matches against the engram root path. Match + target missing → delete the link (dangling); match + target present → healthy engram link, leave; no match → foreign symlink (user-owned), never touched. Real files remain untouched. Lexical comparison is essential: `EvalSymlinks` on macOS rewrites `/var` → `/private/var`, breaking identity for symlink-free trees and defeating the ownership test. The owned root's path and written link targets are both logical paths; comparing them lexically is the identity transform (internal/update/update.go `planCleanupDanglingLinks`).

6. **First-sync migration: adopt intended-set copies, report unknowns.** When syncing a harness for the first time (no marker present), the engine creates the root + marker, then iterates the intended deploy set. For each artifact, if a real file/dir already exists at the harness path (a pre-sync copy), the artifact is written into the root and the harness path is replaced with a symlink — the repo is source-of-truth, so no content comparison is needed. Real files at harness paths **not** in the intended set are left in place and listed in the update report; engram cannot prove ownership of what it never recorded writing, so it does not delete them. This is the only time the root is created and marked; subsequent runs are pure sync operations.

7. **Per-harness deploy mode, gated by symlink-discovery verification; manifest fallback.** Every `HarnessSpec` carries a `DeployMode`: `DeployModeSymlink` (default, all three currently-supported harnesses per verification-verdicts.md tasks 1.1-1.3) or `DeployModeManifest`. Symlink mode writes the real files into the engram root and materializes symlinks at the harness surfaces (decisions 1-3 above). Manifest mode copies real files as before and records every written path in an internal manifest inside the engram root (`.engram-manifest.json`, opaque to the user); sync-deletion then operates on the recorded manifest instead of symlink identity, preserving all sync semantics without requiring symlink discovery to work. Before a harness ships in symlink mode, its native discovery (how it locates skills/commands/guidance) is verified against a real symlink: does the harness actually find and load a skill/command/guidance when it reaches that artifact through a symlink? All three currently-supported harnesses verified yes (internal/update/update.go `DeployMode` doc comment cites verification-verdicts.md). A harness that fails verification runs in manifest mode, which is equally sound from a sync/deletion perspective.

8. **Dry-run contract: operation classification and preview rendering.** Every materialization and sync operation flows through the dry-run prefix discipline. Each planned action is classified (`create-link`, `sync-write`, `sync-delete`, `cleanup-link`, `migrate`, etc.) and rendered with the classification as a prefix when `--dry-run` is set. Dry-run does **not** create the root or marker — the full operation plan (including first-sync migration) is previewed without mutating the filesystem, supporting user review before any commit.

9. **Claude guidance compat symlinks for import-line stability.** Claude guidance moves from `~/.claude/engram/*.md` (flat, next to CLAUDE.md imports) to `~/.claude/engram/guidance/*.md` (structured, inside the root). Migration creates compat symlinks at the old flat paths (`~/.claude/engram/recall.md` → `guidance/recall.md`) so user `@import` lines in `CLAUDE.md` continue to resolve without change; the update report tells the user the new canonical paths and that the compat symlinks exist. These compat links are part of the intended set and remain managed; a later change can retire them once imports are migrated.

**Consequences.**

- **Removals propagate:** an artifact deleted from `agent-instructions/` disappears from every harness on the next `engram update`. Deployed state matches intent, and orphaned files are cleaned up automatically.
- **Ownership is bounded:** sync-deletion runs only inside marker-stamped roots engram created or adopted; a user-deleted marker degrades gracefully to refusing deletion (fail-safe). The update report lists every deletion so removal choices are visible.
- **Symlink semantics are verifiable:** before a harness ships in symlink mode, its discovery through symlinks is tested; manifest mode provides a fallback without loss of sync semantics.
- **First-sync is dark:** the first update after upgrade performs migration (adopts pre-existing intended-set copies to symlinks) in one pass, then proceeds with normal sync; no separate migration step or marker ceremony is visible to the user.
- **Harness discovery becomes canonical:** symlinks provide the single source of truth for what is deployed — querying the harness (asking it to find a skill by name) and querying the symlinks (scanning what engram linked in) will always agree.
- **Rollback is seamless:** the engram-owned root plus symlinks are functionally equivalent to the old real-file copies; reverting the binary and running the old `update` re-copies real files over the symlinks (the old code's `RemoveAll`-then-write path), restoring the status quo ante without data loss.

**Risks / Trade-offs:**

- [Symlinks at harness-visible paths break discovery] → D7's verification gate runs before each harness ships in symlink mode; manifest mode preserves all sync semantics if discovery fails.
- [User deletes the `.engram-owned` marker] → deletion is then refused, reported, and the marker can be restored; this is a fail-safe, not a failure mode.
- [Sync deletes a user-owned file inside an engram-owned root] → only possible if the user manually places a file inside a marked root after creation; the update report lists every deletion.
- [Compat symlinks for Claude guidance linger indefinitely] → they live inside the engram-owned root and remain managed; a later change can retire them.
- [`--dry-run` preview creates a root or marker] → dry-run renders the full operation plan including first-sync without writing; covered by scenario tests.

**Update 2026-08-08:** harness scope narrowed to Claude Code + Pi; OpenCode support (and the commands-deploy mechanism, which OpenCode was the sole consumer of) removed — see issue #721.

---

## ADR-0023 — `engram update` re-execs the freshly installed binary for the sync phase

**Status:** Accepted (2026-07-28)

**Context.** `Updater.Run` installed the new binary via `go install ./cmd/engram/` (local clone) or a git-clone build (remote, #645) and then planned and applied harness sync using the *old, running* process's in-process logic — stale relative to the binary it just installed. There was no re-exec or self-restart mechanism anywhere in the repo.

**Decision.** After a successful install, the parent process writes a handoff report (install result, attributed to the parent) via `HandoffReporter`, then spawns the freshly installed binary (at the resolved installed path, never `os.Args[0]`) as a child with inherited stdio and `ENGRAM_UPDATE_REEXEC=1` set in its environment, passing through the original update args (`reexecArgsFrom`). The parent waits and exits with the child's exit code (`deps.Exit(*report.ReexecExitCode)` in `runUpdate`, gated before the vault/vocab/chunk-index check block). The child observes the sentinel, skips `resolveSource` entirely (no install, no further re-exec — loop guard), suppresses the install-claim header, and performs all sync/checks with fresh logic. A `Handoff.WriteHandoff` failure is a hard error that aborts `Run` (a stdout-write failure, not a re-exec failure — smoothing it into the fallback would just resurface the same broken stdout there). If spawning the child fails (binary missing/not executable), the parent falls back to completing sync/checks in-process with the old logic and records the failure reason on the report (`ReexecFallbackErr`). `--dry-run` never installs and therefore never re-execs. Source: `internal/update/update.go` (`reexecAfterInstall`, `resolveSource`, `HandoffReporter`, `ReexecSentinelEnvVar`, `Spawner`) and `internal/cli/update.go` (`runUpdate`, `reexecArgsFrom`, `writeReexecHandoffReport`).

**Consequences.**

- **Fresh logic every run:** the sync/check phase always executes in the binary that was just installed, eliminating the stale-in-process-logic gap.
- **The bootstrap run is the last stale one:** the very first `engram update` after this change ships still runs the *previous* binary's in-process logic end to end (it re-execs into the newly installed binary, but that binary is this change's own code — by the next invocation the loop is fully fresh).
- **Version-skew adjacency (out of scope here):** local mode installs from the clone it runs against, not from a fresh pull — an old checkout still downgrades `~/go/bin/engram` on re-exec exactly as it did before this change (`git diff HEAD` / worktree parity is the operator's responsibility; see also the remote mode's clone-based build, ADR context above). A version-skew guard remains a separate, unscoped follow-up.
- **Single coherent report:** install output is attributed to the parent and exactly one sync/check report is produced, by the re-execed child — never duplicated.

**Risks / Trade-offs:**

- [Recursion if the sentinel is dropped] → the sentinel is set explicitly on the child's env slice; a sentinel-bearing run never calls the installer.
- [Spawn failure loses the install] → in-process fallback completes the run with pre-update logic and reports the failure reason; the update is not aborted.

---

## ADR-0024 — Derivational vocab refit: geometry-derived terms, growth-only trigger, plan flow removed

**Status:** Accepted (2026-07-28)

**Context.** The controlled vocabulary (ADR-0011) was maintained by a plan-based refit flow (`--emit-request`/`--plan`) that was an additive-only ratchet: an LLM judged merges/splits/renames against the *existing* term list, so terms accumulated and never tracked the vault's actual semantic geometry — observed as ~150-term drift on the production vault. The untagged-rate and hub-concentration triggers pushed more LLM-judged refits without addressing the ratchet.

**Decision.** `engram vocab refit` derives the vocabulary from the vault's embedding geometry instead of editing the term list. Whole-vault k-means with silhouette-selected auto-K (argmax over K∈[2,40]); a floor of 0.02 acts as a noise-rejection bound only — K=0 (keep existing vocabulary) when no structure clears it. Cluster centroids are greedily matched to existing term vectors at cosine ≥ 0.80 (`vocabNameMatchThreshold`); matched terms survive with refreshed centroids, unmatched clusters are emitted as fingerprinted `naming_requests` JSON for the agent to answer via `--names <file>` (the echoed fingerprint proves the answer names this vault state), and unmatched *derived-origin* terms are retired. An `origin` field (`derived`/`proposed`) in `vocab.centroids.json` is a provenance shield: `origin: proposed` terms (operator-minted via `vocab propose`) are never auto-retired. The trigger becomes growth-only (≥40 new notes AND ≥14 days since last refit); untagged-rate and hub concentration remain `vocab stats` diagnostics but never set `refit_pending`. Silhouette hysteresis (ε = 0.02) keeps the previous K unless a new K clearly wins. Calibration is measured, not borrowed: the production 597-vector corpus peaks at silhouette 0.0987 (K=9), so recall's 0.10 query-subset floor does not transfer to whole-vault MiniLM geometry — hence the 0.02 noise bound. Source: `internal/cli/vocab_derive.go`, `internal/cli/vocab_apply.go`, `internal/cli/vocab_trigger.go`, `internal/cli/vocab_commands.go` (`RunVocabRefit`).

**Consequences.**

- **Plan flow removed (BREAKING):** `--emit-request` and `--plan` no longer exist; the refit CLI surface is `--dry-run` and `--names <file>`. The learn skill's Step 1.5 runs derive → answer naming requests → apply.
- **Retirement semantics:** retired terms are demoted definition notes recorded as supersession entries on the family note (major version bump) — history preserved, no deletion; member `vocab/<term>` tags are rewritten.
- **First-apply caveat:** measured K=9 vs 29 current terms means the first production apply would retire most existing derived terms — the `--dry-run` review before a real apply is load-bearing, not ceremonial.

**Risks / Trade-offs:**

- [Geometry churn renames stable vocabulary] → the ≥0.80 centroid match keeps semantically stable terms; hysteresis damps K flapping; `origin: proposed` shields operator intent.
- [Stale naming answer applied to a moved vault] → the derivation-input fingerprint must be echoed verbatim or the `--names` run is rejected.

---

## ADR-0025 — Centroid-proximity explore sampling replaces tag nomination in recall queries

**Status:** Accepted (2026-07-29) · supersedes ADR-0011's candidate-side tag-nomination mechanism.

**Context.** ADR-0011's tag nomination joined a note into a cluster's `candidate_l2s` pool whenever
it shared a vocab term with the top-3 delivered notes in that cluster, regardless of the note's
actual cosine to the query. This coupled candidate augmentation to an accident of which three notes
happened to rank highest, gave every term-sharing note equal footing regardless of relevance within
the term, and required per-cluster pool-cap bookkeeping (`nominationCapPerCluster`) to bound cost.
`internal/cli/query_nominations.go` implemented the join; `openspec/changes/recall-centroid-sampling`
proposed and specced the replacement.

**Decision.** Replace tag nomination with an explore half sampled directly from vocab-term
centroids, allocated by proximity to the query rather than by co-occurrence with top-ranked notes.
The note channel is now exploit (unchanged: existing cosine-nearest matched notes, floors/caps as
before) + explore (new). Explore budget B equals the count of distinct exploit-half NOTE items
(chunks never count); B=0 skips explore entirely. Allocation across terms is
`softmax(cosine(query vector, term centroid) / τ)`, where the query vector is the mean of the
recall call's per-phrase embeddings and τ is set from calibration (task 3.1) — no radius cutoff, so
every vocab term with a centroid is eligible, weighted by proximity. Terms with ≥1 member already in
the exploit half get a flat additive similarity bonus δ=0.05 pre-softmax (a match-evidence boost;
non-stacking — one bonus per term regardless of exploit-half member count). Within a term, members
(notes carrying `vocab/<term>`, definition notes excluded) are selected core-first: descending
cosine to the term centroid. Explore picks are deduped against the exploit half and across terms;
freed slots backfill in allocation order. Delivery changed shape: explore notes are top-level
`items[]` entries — inserted after ride-along assembly, before the chunk/recency channels — never
cluster members and never in `candidate_l2s` (`candidate_l2s` is now purely within-cluster top-5,
nothing else). Each explore item carries `provenance: explore` and `source_term`. The stage key
`nominate` in `--timings` output is kept (renamed work, not renamed key) but now measures explore
sampling + ride-along assembly. Source: `internal/cli/query_explore.go`,
`internal/cli/query_vault_meta.go` (renamed from `query_nominations.go`).

**Consequences (BREAKING).**

- `tag_nominations_added`/`tag_nominations_dropped` budget fields are gone. The replacement is
  `explore_allocated`, a term → delivered-count map (post-dedupe), ALWAYS present in the payload —
  `{}` when `vocab.centroids.json` is missing or unreadable, a loud degrade to exploit-only rather
  than a silent one.
- `nominationCapPerCluster`/`topNForNomination` constants are gone; explore sizing is governed by
  the exploit-half note count (B) and the softmax allocation, not a fixed per-cluster cap.
- Consumers reading `candidate_l2s` for anything beyond within-cluster top-5 (e.g. treating it as
  "all surfaced note candidates") must also read top-level `items[]` filtered to
  `provenance: explore` to see the full candidate set — `agent-instructions/skills/recall/SKILL.md`
  (task 4.1) and the two-channel payload spec (task 4.3) were updated accordingly.
- Task 3.2 measures recovery: for each τ-calibration query, the fraction of before-arm
  nomination-only deliveries still reachable somewhere in the after-arm payload (exploit, explore,
  or ride-along); the keep/revert verdict is Joe's, evidenced by `dev/eval/LEDGER.md`.

**Risks / Trade-offs:**

- [τ mis-calibration flattens or spikes allocation] → task 3.1 calibrates τ against the production
  vault before this ADR's decision ships to the default payload.
- [Softmax-with-no-radius-cutoff always allocates something to every term] → the δ=0.05 bonus and
  core-first within-term selection keep low-relevance terms from crowding out evidenced ones; B=0
  (no exploit notes) skips explore outright.

Link: `openspec/changes/recall-centroid-sampling`.

---

## ADR-0026 — `runbook`: a fourth content kind for task-shaped strategies, symmetric with fact/feedback

**Status:** Accepted (2026-08-24).

**Context.** Kind-4 confirmed-approaches ("what worked, do it again") lived awkwardly inside
`feedback` notes' `action` field as a single sentence — no structural way to represent an ordered,
multi-step procedure or a distinct ending condition. SPL (optillm's System Prompt Learning plugin)
research showed procedural-strategy injection lifts weak-model performance materially
(OptiLLMBench +4%, AIME24 +7%, Arena-Hard +8.6%), motivating a first-class kind rather than a tag
convention on `feedback`. Two design-review corrections during implementation (both driven by
inline comments on the openspec change's design.md, `.review/events.jsonl` threads
`6c7d2c3de70a93ece18b99c0d6cd5e75` and `01727d1c429bb66e76a439e1167ff005`) materially changed the
shipped shape from the original proposal:

1. A `task_type` classification field + `engram query --task-type` pre-filter was originally
   planned, justified by the SPL numbers above. Those numbers measure SPL's whole bundled system
   (type-classification + strategy injection + refinement/pruning) — no ablation in the source
   material isolates type-classification's own contribution, so they don't specifically justify a
   new ranking mechanism and its regression risk to existing retrieval. Dropped entirely (not just
   the pre-filter — the schema field too).
2. `runbook` was originally going to use a kind-prefixed flat filename (`runbook.<date>.<slug>.md`,
   no Luhmann ID), modeled on `qa.*`, on the reasoning that runbooks lack a natural parent idea.
   That reasoning didn't hold: `nextLuhmannID`'s `position=top` branch mints a fresh top-level ID
   with no target required (`internal/cli/luhmann.go`), so "no relationship yet" was never a real
   blocker — every `fact`/`feedback` note already handles that case the same way. `qa` turned out to
   be the wrong template entirely: it's the one kind that deliberately skips Luhmann disposition
   (`internal/cli/qa.go`) and gets partially retrieval-excluded (`isQueryExcludedKind`) — neither of
   which applies to `runbook`.

**Decision.** `type: runbook` (Joe, 2026-08-23, vault note 789 — chosen over `procedure`, `function`,
`strategy`; deliberately non-consonant with `fact`/`feedback`, the ops vibe is intended; `procedure`
collides with existing "recall procedure" prose, `function` with tool-calling vocabulary). Schema
answers exactly three questions: *when should you use this runbook* (`situation`, embedded as the
situation vector exactly like fact/feedback), *what are the steps* (body, numbered, may
`[[wikilink]]` fact/feedback notes to consider), *what should be true when you're done*
(`done_when`, required — also makes runbooks scorable for #718's future efficacy tracking, deferred,
not shipped here). No `inputs`/calling-convention field (considered during the #728 adapter
experiment, rejected). Capture reuses the existing `fact`/`feedback` pipeline as a third `learn`
targ.Group subcommand (`internal/cli/targets.go`/`learn.go`: `LearnRunbookArgs` embeds
`CommonLearnArgs`, gets `--target`/`--position` Luhmann disposition for free) — not a bespoke
standalone implementation. Filename/ID: `<luhmann-id>.<date>.<slug>.md`, identical to fact/feedback,
assigned via the same top/continuation/sibling disposition judgment; `--reparent-luhmann` covers
runbook notes automatically, with no code change, since it operates on Luhmann IDs generically.
Retrieval: no exclusion treatment (`isQueryExcludedKind` untouched — a mechanism only `qa-question`
uses), ranks purely by situation-similarity in the main matched set, identical to fact/feedback — no
new query flag, no new ranking mechanism, no recall Step-1 changes. `learn`'s Step 2 kind-4
(confirmed approaches) now routes by shape: a reusable multi-step procedure for a recurring task →
`kind=runbook`; a single behavioral tweak with no step structure → `kind=feedback`, as before. A
runbook capture is itself the "remember to do it this way" note — it does not also fire a redundant
kind=fact save-request when the user says "do that going forward."

**Consequences.**

- Four content kinds now exist: `fact | feedback | qa | runbook`. `qa`'s asymmetric
  capture-and-exclude shape remains a special case for that kind alone, not a template other kinds
  should follow.
- Migrating the vault's *existing* kind-4 feedback content into `runbook` notes is explicitly out of
  scope here — tracked separately as #730 (untriaged: mechanism, disposition, and timing are all
  still open).
- No validation gate (vault supply check / A/B recall test / SPL methodology cross-check) shipped —
  the originally-planned gate existed specifically to de-risk the now-dropped `task_type` pre-filter;
  with that mechanism gone, `runbook` is structurally identical to how `fact`/`feedback` already
  work, so there was no novel ranking mechanism left to de-risk.
- `internal/cli/qa_test.go`'s `TestIsQueryExcludedKind` and a new
  `TestRunQuery_RunbookCompetesInMainMatchedSet` (`internal/cli/query_runbook_test.go`) both passed
  on first run with zero production-code changes for the retrieval side — the "no exclusion"
  property was already true by construction once capture landed (Decision 2 is an omission, not an
  addition).

Link: `openspec/changes/runbook-note-kind`.

---

## ADR-0027 — The human-memory taxonomy as coverage checklist (not architecture): north star for memory-system decisions

**Status:** Accepted (2026-08-30).

**Context.** Engram's mechanisms accumulated capability-by-capability (chunks, fact/feedback notes, runbooks, supersession, activation, curate) without a stated model of what a *complete* memory system covers, so direction questions (what is #735's firing cue really for? should runbooks absorb skills? does composition need machinery?) were being decided ad hoc per issue. Joe's stated north star: recognize/record/retrieve/use the memory categories humans have — episodic (this happened), semantic (what it means), procedural (how to apply it) — because matching human memory categories matches user expectations of the agent. A four-briefing research sweep (human memory science; LLM agent-memory field survey; procedure-composition mechanisms; a repo-verified engram inventory) plus an adversarial critic pass was run 2026-08-30 to test that north star against the literature and against engram's actual mechanisms (`docs/research/2026-08-30-memory-taxonomy-*.md`; per the `design/`-charter those files are extractable — this ADR is the durable record and is written to survive their deletion). Findings that shaped the decision:

1. The episodic/semantic/procedural frame is now the *mainstream descriptive taxonomy* of the LLM agent-memory field (2025–26 surveys organize by it; LangMem/MIRIX ship it as SDKs) — but **no published ablation shows the store-split itself causes wins**. Its value is as a completeness checklist, not a partitioning scheme.
2. Mining procedural memory from experience is the **best-replicated positive result in the field** (AWM: mined workflows beat human-authored by 7.9%, +51% relative on WebArena; Memp: strong-model-built procedures transfer to weaker models — independently replicating engram's tier-routing finding; ReasoningBank; Voyager). The runbook direction stands in the proven corner.
3. Strictly, a retrieved runbook is **declarative memory *about* a procedure**, not procedural memory (which in humans is compiled and automatic; in an agent stack only model weights and executable code are procedural in that sense). The agent re-enters the novice "reading the recipe" stage on every retrieval — predicting the field's best-documented failure mode ("retrieved-but-not-applied", now a named benchmark category = #738 link (c)) and the expected error profile (step-skipping and mis-ordering, not habit errors).
4. Firing is **prospective memory**: cues work when *focal* (part of what the agent is already processing) and phrased as implementation intentions ("when X → do Y", action named); strategic monitoring ("check every step whether recall is warranted") is the architecture whose cost signature engram already measured as the rejected 147×–380× hook (`recall-overfire-hook-rejected`).
5. Retrieval keying is **encoding specificity**: retrieval succeeds when the cue at retrieval matches what was encoded — the `situation:` field is this, and the qanchor result (idiosyncratic tokens load-bearing, `qanchor-park`) is what that literature predicts.
6. Composition of simultaneously-applicable procedures decomposes into three classes — **constraint-like** (blend safely by conjunction), **plan-like** (need sequencing at seams), **same-slot** (need arbitration; LLMs measurably resolve these conflicts *silently*, picking one side unflagged). Read-time composition of prose procedures is unsolved field-wide; the proven mechanisms are write-time and retrieval-time (supersession, scope-specificity, conflict-surfacing, merge-then-supersede) — exactly the points a zettelkasten controls.
7. A scope limit in existing doctrine: "memory pays only on idiosyncratic content" (`c1-c2-warm-op-negatives`, `crowded-vault-capability-robustness` vintage evals) was measured on *facts and conventions*; the field's procedural wins are mostly *generic* procedures. Whether the rule binds for runbooks is unmeasured.

**Decision.** Adopt the human-memory taxonomy as engram's **coverage checklist and decision lens — explicitly not as an architecture mandate**. No store re-partitioning, no new kinds motivated by taxonomy symmetry alone. Concretely:

- **Vocabulary:** "runbook = declarative memory about a procedure" is the sanctioned framing; runbooks are permanent novice-stage scaffolding whose value concentrates where the model lacks priors, and whose failure profile (step-skipping, mis-ordering — not habit errors) is what apply-side evals (#736) should score. Keep the name `runbook`.
- **Gap map (the actionable half of the checklist).** Each human-memory function maps to owned work; new memory-direction proposals should locate themselves on this map before proposing mechanisms:
  - *Prospective memory (firing)* → #735: cue framed as **situation-recognition, broadened** — implementation-intention wording naming the action, "the vault may hold a runbook for this **or a similar** situation" (no exact-match implication), recurring-routine as a positive example inside the cue, fixtures spanning named-routine + situational-bind + near-match shapes, fire-unit pinned at task-init.
  - *Apply (declarative→enacted)* → #736, scoring the cognitive-stage failure profile; decision-point injection (retrieved runbook as instructions at task start, not reference material) is the field's best mitigation and belongs in #736's option set.
  - *Procedural value + capture scope* → #737 gains a **generic-vs-idiosyncratic fixture dimension** (some migrated-pair fixtures idiosyncratic, some generic-but-multi-step); runbook capture policy follows the measurement, not the fact-scoped doctrine.
  - *Strength & forgetting* → #718: two-variable design — storage strength (validated correctness; never self-decays) separate from retrieval strength (recency **and frequency**: use-count history per ACT-R base-level activation, not the current single `last_used` date where ten activations equal one).
  - *Reconsolidation (update-on-use)* → **flag-and-queue**: glance stays read-only for content; an observed mismatch (note said X, in-context evidence shows Y) records a pending flag on the note; the closing learn or next deep recall processes the queue with full judgment. The prediction-error gate is the principle: update on mismatch, strengthen (activate) on confirmation, never churn on mere surfacing. Mirrors the curate gate's judged-write pattern; avoids opening the cheap fast mode as an auto-evolution rot surface.
  - *Interference (supersession)* → **probe before build**: today `applySupersedesRideAlong` delivers the superseded note in full and inserts the superseder after it, relying on the agent's recency-weight rule every time. Three convergent external lines (cue-overload, write-time conflict resolution as the field's proven composition mechanism #1, the experience-following 13%-vs-39% curation result) say suppress instead. Hypothesis registered: suppression (supersessor-only or tombstone) wins; a cheap A/B with the C4i recency-supersession trap as regression guard decides. No ship without the measurement.
  - *Composition* → a tracked mechanism set (conflict-surfacing rule; scope-specificity metadata; merge-then-supersede after adjudication), **hard-gated on #737 proving the runbook kind carries value**. No read-time auto-merge, no numeric priorities — both unsupported by evidence.
  - *Promotion/demotion (loading-tier migration)* ↔ #728's adapter probe (vault-ward direction); the hot-runbook→always-loaded direction is future work contingent on #737.
- **Deferred, with rationale (revisit conditions named, not silently dropped):**
  - *Background consolidation (replay over never-queried episodes):* complementary-learning-systems theory calls it mandatory; the engineering evidence (sleep-time compute) is first-party and thin. Engram's consolidation stays pull/moment-triggered. Revisit when third-party replications land or a measured gap (valuable episodes provably never consolidating) appears.
  - *Memory poisoning / write-provenance trust tiers:* `ingest --auto` persists transcripts unjudged and `serve` accepts external offers; unaddressed by design for today's single-operator vault. Revisit as the vault graph (ADR-topology, note 784) grows beyond one trusted operator. Known unowned gap, deliberately not an issue yet (Joe, 2026-08-30).

**Consequences.**

- Three new issues filed alongside this ADR: #739 multi-agent memory routing (lessons discovered in discarded subagent contexts; recalled memory not reaching dispatched subagents), #740 a lexical exact-token retrieval channel probe (dense MiniLM is weak at rare-token matching while verbatim idiosyncratic tokens are the proven value carrier; no lexical channel exists in `internal/` today), and #741 the gated composition mechanism set. Plus #742, the supersession-suppression probe. #735/#737/#718 updated with their decisions above.
- The idiosyncratic-only doctrine is downgraded from "rule" to "fact-scoped finding" pending #737's generic dimension.
- Evaluation continuity: every gap-map item keeps the house eval discipline (pre-registered bars, headless arms where behavior is measured, LEDGER rows) — the taxonomy adds *what to cover*, never a pass on *how to prove it*.
- The research docs under `docs/research/2026-08-30-memory-taxonomy-*` follow the `design/` charter (extractable once conclusions graduate); this ADR carries every load-bearing conclusion so their later deletion loses nothing decision-relevant.

**Risks / Trade-offs:**

- [Taxonomy-as-checklist drifts into taxonomy-as-architecture (new kinds/stores for symmetry)] → the "coverage checklist, not architecture" clause is the explicit test: a proposal must name the *measured gap* on the map, not the unfilled box.
- [Human-analogy overreach — importing decorative analogies (capacity constants, neuroanatomy mappings, decay-curve mimicry) as requirements] → the research synthesis's load-bearing-vs-decorative table is the reference; only computational rationales (CLS, encoding specificity, PM multiprocess, two-variable strength) ground decisions.
- [Deferred items rot silently] → each carries a named revisit condition here; the ROADMAP cites this ADR rather than restating.

Link: `docs/research/2026-08-30-memory-taxonomy-synthesis.md` (+ 5 sibling briefings); #735 #736 #737 #718 #728 #738; `dev/eval/LEDGER.md` anchors `recall-overfire-hook-rejected`, `qanchor-park`, `c1-c2-warm-op-negatives`, `734-runbook-retrieval-probe`.

---

## ADR-0028 — Engram as the experiential learning layer: the memory-loop audit, judged-only credit, situation-conditioned experience

**Status:** Accepted (2026-08-30).

**Context.** ADR-0027 adopted the human-memory taxonomy as a coverage checklist and mapped the gaps to owned issues. Two same-day events forced the next structural decision. First, Joe reversed #739 Phase 1's scope (review comment, thread `5974d361…`, on the delegation-boundary change): audit **main transcripts as well as subagent ones** — "otherwise we don't know how well the main agent is doing at recalling relevant memories" — with a per-moment rubric spanning the whole loop (moment identified → was a relevant memory in the vault → searched reasonably → surfaced at what relative priority → acted in accordance → reasonable moment to learn/update → reasonable write; for subagents, prompt-injected memories count as searched-and-surfaced). Second, Joe named the north star behind the rubric: **engram is the learning/reinforcement layer for statically-weighted LLMs.** The weights never learn from experience; the memory system must — successes reinforce the parts that work, failures devalue the parts that don't, and missing functionality/data/process gets identified, tested, integrated. The field's best-replicated agent-memory results (AWM, Memp, ReasoningBank) share exactly that shape — experience → outcome labeling → memory update, learning from successes AND failures — and engram has every stage of the loop except the outcome labeler. Prior findings binding the design: op-value is unmeasurable in synthetic builds (`harder-regime-op-cost-unmeasurable`; vault notes 98/288 — real long-session work is the only non-tautological value regime); observational audits size gaps while headless RED/GREEN arms prove fixes — different questions, both needed (the 2026-06-28 mining → `recall-moments-headless-flip` precedent pair); unjudged memory auto-evolution rots (the curate gate; "memory misevolution", taxonomy critic §1a).

**Decision** (Joe, 2026-08-30, D1–D8):

- **D1 — Standing artifact, not a one-shot.** The memory-loop audit is a permanent instrument with three run modes: **observe** (real transcripts → per-moment funnel rates + draft credit records), **replay** (retrieval stages re-scored offline against a candidate vault state / ranking / note shape — pre-merge evidence for retrieval-side changes), and **mine** (audited moments → headless RED/GREEN fixtures — pre-merge evidence for skill/guidance changes). Replay and mine exist because observation alone cannot evaluate an unshipped change: the transcripts already happened under the old system.
- **D2 — The audit's questions are the loop's loss stages.** Per moment — failures AND successes, with dispatch as an explicit moment type: *exists* (which memory kind; generic vs idiosyncratic) → *fired* (recall run, or memory injected) → *keyed* (phrases match the situation — scored separately from fired, per ADR-0027's firing/encoding-specificity split) → *surfaced* (rank; superseded-note-above-current flag) → *applied* (deviation shape: ignored / partial / step-skip / mis-order / contradicted) on the read side; *learn-moment* → *written* (situation keyed as a future task, supersession used, non-duplicate) → *strengthened* (used notes activated, mismatches flagged) on the write side; plus the delegation-boundary slice. Classification per vault runbook 824 (addressable / capture-ceiling / application-or-ranking-miss); fire-unit pinned before any ratio; every number DERIVED or ESTIMATE.
- **D3 — Judged-only credit.** The audit emits credit records (note ref, situation, outcome, evidence pointer); nothing in the vault changes unjudged. A curate-shaped consumer judges each record → amend / supersede / resituate / prune / keep. No automatic strength decay, no auto-promotion, no auto-prune. (Rejected: mechanical strength tallies with judged content; fully automatic evolution.)
- **D4 — Situation-conditioned experience, not scalar ratings.** A note that hurt in one situation may help in another (the same UI-dev note: fast-PoC vs principled build). Each note gains an **experience record**: a capped, frecency-evicted list of (situation text + vector, outcome, timestamp, evidence pointer), admitted only by the judged pass. Grounding: ACT-R associative (context-conditioned) activation — the retrieval-time mirror of the `situation:` field's encoding specificity. Ranking use is phased, probe-before-build: v1 = admitted entries ride along agent-visible in the query payload (like supersession ride-alongs); v2 = a mechanical similarity-weighted multiplier, only after replay proves it on recorded moments, with the C4i trap as regression guard. No entries → exactly today's ranking; absence of data is never a penalty. #718's retrieval-strength half becomes this mechanism; storage strength (validated correctness; never self-decays) is unchanged.
- **D5 — Product / dev-eval split.** Product owns the vault-content loop: audit extraction, credit-record emission, the judged consumer, and experience records ship as engram capabilities (binary + skills) so every user's vault learns from their own transcripts. `dev/eval` owns evaluating engram's own skills and shapes: observe reports on the maintainer corpus, replay, mine, LEDGER recording. Build order: the instrument lands Python-first under `dev/eval` (measure before productizing; the gate #739 Phase 2 waits on needs the measurement, not the product plumbing); promotion into an `engram audit` subcommand lands with the credit-loop change.
- **D6 — The versioned surface is the judged surface.** Git is a hard precondition of the learning-loop capabilities — not of core engram (recall/learn on a plain directory keep working). With git: the existed-at-T oracle uses read-only `git worktree` checkouts at the last commit ≤ T (DERIVED). Without git: the oracle degrades to a `created ≤ T` filter labeled ESTIMATE, and the judged consumer refuses **prune** specifically (irreversible without history). Commits are judged-pass discipline — one commit per judged session (closing learn, curate, the credit consumer), evidence pointers in the message — never per-command auto-commits (concurrency with the vault flock; commit spam; surprising side effects). Tracked: note `.md` files + experience records (a sibling `.exp.json`, so admissions do not change the note's content hash or trigger re-embeds). Ignored: `.vec.json` sidecars (derived vector + mechanical `last_used`; tracking them would bury judged history in activation noise). Pending offers sit uncommitted until curated — `git status` on the vault is a literal unjudged-material indicator. Commit cadence, not ceremony, controls oracle resolution: sparse history in old vaults means ESTIMATE for old moments; no backfill.
- **D7 — The build-value harness is retired; the LEDGER is restructured additively, never rebuilt.** The warm-vs-cold build value-proof direction (`dev/eval/cumulative` matrix/harness as value instrument; the #642 hard-regime plan) is closed — the audit's observe mode replaces it on the value question, per the `harder-regime-op-cost-unmeasurable` redirect. The traps gate and the per-issue probes stay as the interventional half (C4i remains #742's regression guard). `dev/eval/LEDGER.md` keeps every row and every anchor (cited by ADRs, ROADMAP, and openspec specs) and gains a **coverage-map index**: memory function × kind → the rows evidencing it + the audit's measured real-work rate.
- **D8 — #739 Phase 2 is hard-gated on the audit.** The dispatch-injection + completion-report-LESSONS change proceeds only on a pre-registered GREEN from the audit's #739 slice. Option B (full per-subagent memory loops) stays parked with its ADR-0027 revisit conditions.

**Consequences.**

- The `delegation-boundary-gap-measurement` change is re-proposed as `memory-loop-audit` (observe instrument + first measurement + the Phase 2 gate); its two capability specs are absorbed into one `memory-loop-audit` capability (they were never synced to `openspec/specs/`). Follow-on changes, in order and gated: credit loop (record consumption, curate-shaped consumer, experience records v1, `engram audit` productization, vault-git commit discipline), then replay + mine.
- #739's issue body is rewritten as the one coherent current brief (issue body is the delivery mechanism — vault note 819) with Phase 2 explicitly blocked on the gate; #718 is narrowed (retrieval strength := the experience mechanism; storage strength unchanged); #742 gains a note (situation-conditioned demotion is soft-suppression evidence).
- Evaluation continuity holds: pre-registered bars, hand-labeled calibration, miss-rate gating, scorer hand-reads, #708 isolation, LEDGER rows. The audit adds an observational instrument; it never substitutes for headless proof of a fix.

**Risks / Trade-offs:**

- [Judged-only throughput: credit queues rot unprocessed] → the queue is surfaced like pending offers (visible at recall/update time); processing cadence rides the judged moments that already exist (closing learn, curate).
- [The semantic auditor under-counts subtle moments and skews the funnel] → inherit the 2026-06-28 discipline: semantic detection, over-match→prune, miss-rate gate on hand labels, all fractions labeled conservative; scorer hand-read before any paid run (vault note 505).
- [Experience records add a second ranking signal that could fight recency/supersession] → phased introduction (agent-visible v1; mechanical v2 only on replay proof) with the C4i trap gate as the regression guard.
- [Git precondition fragments the user base] → degradation is defined per capability (ESTIMATE oracle; prune refused); core engram is unaffected.

Link: ADR-0027; review thread `5974d361…` (Joe's rubric comment, 2026-08-30); `dev/eval/LEDGER.md` anchors `harder-regime-op-cost-unmeasurable`, `failure-mining-mid-task-gap`, `recall-moments-headless-flip`, `687-surprise-harvest`, `c1-c2-warm-op-negatives`; vault notes 98, 112, 288, 495, 505, 803, 819, 824, 853, 855; #739 #718 #742 #737 #736 #735.

---

## Decisions deliberately NOT made into ADRs

- **"Curate, don't regenerate" → full rebuild** (B10): a reversed operational decision, not an
  architectural one — recorded as a dated reversal in Phase 0, not an ADR.
- **Capture abstraction = generic-actionable** (B2): a *skill-authoring* convention (how to phrase
  a note), gated by RT/eval, not a C2 architecture decision.
