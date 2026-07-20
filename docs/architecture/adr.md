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

**Status:** Accepted (INV-S1 seam resolved 2026-06 via `engram amend`, `internal/cli/amend.go`; #700 (2026-07): raw I/O primitives relocated to `cmd/engram` — declaration-free `package main` over `cli.Primitives`, `targ check-thin-api`-enforced; ALL adapter composition + wiring live in `internal/cli` (`cli.NewDeps`); `internal/` is import-pure — lint-enforced, ADR-0020)

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
the relational-retrieval mechanism.

**Representation update (2026-07-10, #678, per ADR-0019):** the fixed term set now lives as bare-`vocab`-tagged definition fact notes (`vocab-<term>-definition`) and member terms as `tags: [vocab/<term>]`. The centroid-assignment and query-nomination mechanism is unchanged.

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
live in `skills/route/SKILL.md`). Evidence notes stay in recall — excluding them would regress on
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
declaration-free: a single-statement `main()` populating `cli.Primitives` with raw capability
references (`os.ReadFile`, `time.Now`, `filepath.WalkDir`, syscall wrappers) and sanctioned
closures (single-call signature-erasers plus the two enumerated stdlib-equivalent survivors,
`WriteFileExcl` and `RunCommand`) — zero orchestration. Enforcement is config-only and
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

## Decisions deliberately NOT made into ADRs

- **"Curate, don't regenerate" → full rebuild** (B10): a reversed operational decision, not an
  architectural one — recorded as a dated reversal in Phase 0, not an ADR.
- **Capture abstraction = generic-actionable** (B2): a *skill-authoring* convention (how to phrase
  a note), gated by RT/eval, not a C2 architecture decision.
