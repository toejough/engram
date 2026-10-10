# Architecture Decision Records — engram

Retrospective ADRs. The decisions below were mostly made implicitly across the system's
evolution; this log records them so they can be challenged, and ties each to the **verified
defects it produced** (the `⚠ KNOWN` lines).

History: the founding design narrative (tiered-memory research, embedder choice, lazy-L2
synthesis, and the decisions superseded along the way) lived in `docs/DESIGN-HISTORY.md`,
deleted 2026-07 — `git log` recovers it.

Status legend: **Accepted** · **Accepted (known defect)** — sound decision, buggy as-built ·
**Superseded**. Evidence: commit hashes, `file:line`, and the C4 set
([L1](c1-system-context.md) / [L2](c2-containers.md) / [L3](c3-components.md)) +
[invariants](memory-invariants.md).

---

## ADR-0002 — Pure-Go embedded model; no external embedding API

Ported to
[792b9](../../changes/792_documentation-framework-adoption/792b_technical-spec.md#792b9-added-text-embedder),
with the M4 consequence in
[792b10c7](../../changes/792_documentation-framework-adoption/792b_technical-spec.md#792b10c7-state-classification).

---

## ADR-0003 — Embed-on-write with per-note `.vec.json` sidecars

Ported to
[792b10](../../changes/792_documentation-framework-adoption/792b_technical-spec.md#792b10-added-embedding-sidecar).

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

## ADR-0011 — Controlled-vocab tag nomination over graph traversal

**Status:** Accepted (2026-07-02/03) · supersedes graph-traversal (PPR / spreading-activation) as
the relational-retrieval mechanism. **Superseded (2026-07-29) by explore sampling** ([792a7i](../../changes/792_documentation-framework-adoption/792a_product-spec.md#792a7i-half-of-the-notes-are-explored)) — tag nomination's
candidate-side "join the pool on shared vocab term" mechanism is replaced by centroid-proximity
explore sampling; this ADR's context/decision/consequences below remain the historical record of
why candidate-side augmentation beat graph traversal in the first place.

**Representation update (2026-07-10, #678, per [792a6c](../../changes/792_documentation-framework-adoption/792a_product-spec.md#792a6c-categorical-tags-share-one-list)):** the fixed term set now lives as bare-`vocab`-tagged definition fact notes (`vocab-<term>-definition`) and member terms as `tags: [vocab/<term>]`. The centroid-assignment and query-nomination mechanism is unchanged.

**Representation update (2026-07-27, openspec change `vocab-definition-self-tags`):** per-term definition notes additionally carry their own `vocab/<term>` self-tag (display-only — connects each definition to its member cluster in tag-based views; the family note stays bare-`vocab`-only). Member semantics unchanged: assignment, stats/trigger math, and tag nomination all exclude definitions via the bare marker.

**Context.** The wikilink graph ([792b13](../../changes/792_documentation-framework-adoption/792b_technical-spec.md#792b13-added-wikilink-graph)) is authored and walked by the binary, but resolves by
basename against bare-Luhmann-id links — most edges never resolve (⚠ KNOWN G0/G5) — and even a
healthy graph leaves open how a relational miss (a note topically related to the matched set but
never phrase-matched) should be recovered at query time. Two mechanisms were evaluated head to
head: ranking-side graph traversal (PPR / spreading-activation / one-hop expansion) vs.
candidate-side nomination through a controlled vocabulary.

**Decision.** Reject graph traversal as the retrieval mechanism. Ship controlled-vocabulary tag
nomination: a fixed term set (`vocab.<term>.md` — representation as of 2026-07; see #678), term
assignment at every write (now [792b20](../../changes/792_documentation-framework-adoption/792b_technical-spec.md#792b20-added-vocabulary-engine)), and at
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
`check`/`amend` ([792b13](../../changes/792_documentation-framework-adoption/792b_technical-spec.md#792b13-added-wikilink-graph)), never by the query path.

---

## ADR-0012 — D5′ asymmetric QA participation

**Status:** Accepted (2026-07-03) · supersedes D5 (full QA exclusion)

The context and decision (linked capture, answers compete, questions excluded) are in
[792a3q](../../changes/792_documentation-framework-adoption/792a_product-spec.md#792a3q-question-and-answer-pairs)
and [792a3t](../../changes/792_documentation-framework-adoption/792a_product-spec.md#792a3t-question-and-answer-retrieval).

**Consequences.** The `vocab stats` QA lines are in
[792a6t](../../changes/792_documentation-framework-adoption/792a_product-spec.md#792a6t-vocabulary-statistics).
**Caveat carried forward to round-2/3 gating:** D5′'s asymmetry rests on n=5 synthetic pairs
(vintage 2026-07-03), not yet re-validated at corpus scale — round-2 gates on ≥20 real captured
pairs (or ~2026-07-17, whichever comes first) against pre-registered bands (PASS ≥8, BORDERLINE
6–7, FAIL <6; P2′/P3′ definitions, `docs/ROADMAP.md` → GATED — Q&A memory round-2/3). The dedicated
q-space channel needs its own premise check (Arm V) to reach PASS (≥80%) before round-3 is
licensed; Arm V large-n came in BORDERLINE 63% (19/30) (`dev/eval/LEDGER.md#qa-arm-v-borderline`,
vintage 2026-07-03), so round-3 remains unlicensed pending a further check.

---

## ADR-0013 — Vault flock + atomic-rename write safety

Ported: atomic replace to
[792b3](../../changes/792_documentation-framework-adoption/792b_technical-spec.md#792b3-added-durable-file-store),
and the vault and manifest locks to
[792b8](../../changes/792_documentation-framework-adoption/792b_technical-spec.md#792b8-added-vault-write-lock).

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

## ADR-0026 — `runbook`: a fourth content kind for task-shaped strategies, symmetric with fact/feedback

Ported to
[792a3c](../../changes/792_documentation-framework-adoption/792a_product-spec.md#792a3c-runbook-notes) and
[792a3e](../../changes/792_documentation-framework-adoption/792a_product-spec.md#792a3e-no-task-type-or-input-fields)
(capture), [792a7v](../../changes/792_documentation-framework-adoption/792a_product-spec.md#792a7v-runbooks-rank-like-other-notes)
(retrieval), [792a15g](../../changes/792_documentation-framework-adoption/792a_product-spec.md#792a15g-a-confirmed-approach-needs-a-bet-or-specific-praise-and-takes-the-kind-its-shape-calls-for)
(routing by shape), [792b22](../../changes/792_documentation-framework-adoption/792b_technical-spec.md#792b22-added-runbook-trigger-matching)
and [792a7za](../../changes/792_documentation-framework-adoption/792a_product-spec.md#792a7za-a-trigger-hit-comes-first)
(the lexical exception), and the four kinds and the treatment of each in
[792a19l](../../changes/792_documentation-framework-adoption/792a_product-spec.md#792a19l-each-returned-kind-is-treated-for-what-it-is).

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
  - *Memory poisoning / write-provenance trust tiers:* `ingest --auto` persists transcripts unjudged and `serve` accepts external offers; unaddressed by design for today's single-operator vault. Revisit as the vault graph (ADR-topology, note 784) grows beyond one trusted operator. Known unowned gap, deliberately not an issue yet (Joe, 2026-08-30). Forward pointer: ADR-0029's local-first parent sync (2026-09-27) does not close this gap — a served offer and a pulled-down parent note both land as an unjudged pending note, and it is local curation, not any edge trust check, that gates them into the live vault.

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

## ADR-0029 — Local-first parent sync: every vault local, exchange by offer, never chunks

**Status:** Accepted (2026-09-27).

**Context.** Engram reached a parent vault in two mutually exclusive ways: `ENGRAM_SERVER`, a thin
HTTP client with no local vault at all (every write landed only on the remote, and a down server
failed every command), and `ENGRAM_PARENT`, a read-only merge that never wrote upward. Neither did
what an environment actually needs — keep its own memory, share what it learns upward, pull down
what it uses — and a host configured with `ENGRAM_PARENT` could run for weeks with no local vault
(#766). `ENGRAM_SERVER`'s ten dispatch branches (`internal/cli/targets.go`) also meant every served
`learn`/`amend` bypassed local curation history, and the merge itself carried defects: explore
picks crowding out direct matches (#744), an all-zero budget block (#743), and parent *chunks*
riding along in merged results with no dedupe. On 2026-09-27 Joe replaced both modes with one
local-first, symmetric model (vault note 784a, updating note 784's exchange topology — a
single-parent tree, personal → team → org, never a mesh) and settled the round-1 review's open
questions the same day: no parent chunks in merged results ever (`show-chunk --parent` and the
whole parent chunk path removed); pull-down does a best-effort `/activate` bump on the parent;
`/query-chunks` is deleted; `serve` gets no exemption from the `ENGRAM_SERVER` hard error; and,
in a follow-up ruling, an accepted served offer propagates upward (M14) and a locally-folded
pulled note bouncing back up once is intended (B1).

**Decision** (design D1–D12, full detail in `openspec/changes/local-first-parent-sync/design.md`):

- **D1 — `ENGRAM_SERVER` is a hard error, for every command including `serve`.** A single
  pre-dispatch guard exits non-zero before any vault, chunk-index or network access, naming
  `ENGRAM_PARENT` as the replacement. No compat shim, no exemption.
- **D2 — Every environment always has a local vault, and a stable vault ID.** Ported to
  [792b5](../../changes/792_documentation-framework-adoption/792b_technical-spec.md#792b5-added-vault-identity-registry).
- **D3 — A new exchange hash covers exactly the offered content fields.** `xh1:` + SHA-256 over a
  canonical JSON of `type`, `situation`, the fact/feedback/runbook fields, `done_when`,
  `red_flags`, `triggers`, and the body — never identity, `pending`, `tags`, `sources`,
  `supersedes`, or exchange bookkeeping. It replaces `embed.ContentHash` in every exchange
  comparison (loop suppression, `offer.key`, pull-down skip/decline, rejected-offer re-arm, dedupe
  rule 2, `--expect-hash`); `embed.ContentHash` keeps its own job (sidecar staleness). A reflection
  test fails when a struct field is missing from the offered/not-offered classification table.
  Version-prefix mismatches compare as *unknown*: loop suppression, dedupe rule 2, and
  `--expect-hash` all require an *equal* hash and never pass on *unknown*; only the pull-down skip,
  decline matching, the rejected-offer re-arm, and the send/apply change check treat *unknown* as
  *not changed*.
  The canonical body of one layout, LF frontmatter followed by a CRLF blank separator line, changed
  in place under `xh1:` (fix-show-amend-reparent-frontmatter, ruling V2): it now hashes the same as
  its LF conversion. Across a binary-version skew, only notes in that layout get a spurious offer,
  a missed dedupe-rule-2 merge, a refused pull-down (hash mismatch), and a one-time re-arm of hashes
  recorded under the old binary. These effects stop once both sides upgrade and the next receipt,
  pull or decline re-records the hash. The prefix was not bumped to `xh2:` because a bump turns
  *every* note's comparison into *unknown* across versions, which is far more disruptive than one
  rare layout.
- **D4 — Multi-valued parent links, aliases, and a rename-stable exchange ID.** New optional
  frontmatter: `xid` (this note's own exchange ID, stamped lazily, never backfilled); `parent`
  (`vault`, `links: [{note, via: offered|pulled|covered, hash}]`, `author`) — at most one primary
  link (`offered`/`pulled`), any number of `covered` links; `aliases` (basenames this note answers
  to in its own vault); `offer` (a served offer's origin/key/prior_keys/for/path, kept after
  acceptance). Only the exchange paths set these; every other rewrite site preserves them
  byte-for-byte, each with its own survival test. Renames append the old basename to `aliases`, so
  a parent's or a child's reference to the renamed note keeps resolving. Dedupe treats a parent
  item and a local note as the same note when a link names the parent's basename/alias, or when
  both have equal, non-empty exchange hashes.
- **D5 — What's offered vs. bookkeeping.** `learn` (not `learn qa`), a content `amend`, and
  `resituate` are offered; `--activate`, most `--clear-pending`, `--discard`,
  `--supersedes`/`--chunk-source`-only amends, identity backfill, and anything whose exchange hash
  already equals the primary link's are bookkeeping (never offered). The one exception:
  `--clear-pending` on a note carrying `offer.origin` (accepting a served offer) is itself offered
  upward when this vault has its own parent (M14). The offer payload carries the note's own
  `user`/`repo`, never `target`/`position`/`chunkSources`, and never `tags` — tags are this vault's
  own vocabulary, the same reasoning as D8's pull-down strip.
- **D6 — A local outbox, drain, and backoff.** `<vault>/.engram/outbox.json` queues offers under
  the vault lock in the same critical section as the note write, so a local write never fails
  because of the parent. The drain builds each entry's payload from the note's *current* content at
  send time, sends oldest-first with no lock held, then re-locks and merges. A transport
  error/timeout/5xx keeps the entry queued and backs the parent off
  (`min(15m, 30s × 2^(failures-1))`); a 4xx marks it rejected until the note's hash changes. A
  receipt the local note refuses (an anchored `parent:`, or undecodable frontmatter) marks the
  entry `attention`: it keeps the receipt, warns once, is never re-sent, and the receipt is
  recorded locally, before the backoff gate and without parent contact, once the note is fixed; a
  kept receipt from another parent vault is discarded and the offer requeued. An offer with no
  `user:` and no detected user also waits in `attention` (never sent, one warning) and returns to
  queued once its payload builds (fix-frontmatter-remainder, #789).
  `offer.key = sha256(offer.origin + exchange hash)` makes retries idempotent.
- **D7 — `/learn` is the only write route; pending offers update in place.** The served set, the idempotency
  check, the in-place update, the same-origin case, the `offer.for` resolution and the `offer.path` bound and its 409
  are in
  [792a20a](../../changes/792_documentation-framework-adoption/792a_product-spec.md#792a20a-the-served-command-set-is-fixed),
  [792a20g](../../changes/792_documentation-framework-adoption/792a_product-spec.md#792a20g-a-served-learn-finds-the-note-an-offer-belongs-to)
  and
  [792a20h](../../changes/792_documentation-framework-adoption/792a_product-spec.md#792a20h-an-invalid-offer-is-refused-with-nothing-written).
- **D8 — Pull-down on activate.** `engram activate` resolves locally first; on a miss, a
  basename-shaped ref (never a bare Luhmann ID — IDs are per-vault) is fetched from the parent's
  `GET /show?raw=1` and written as a **new local pending offer**, never live — curation judges it
  before it counts as knowledge. A skip check (an existing link at the same hash, or a prior
  decline) avoids re-pulling unchanged notes; a changed parent note is new information and pulls
  again. After release of the lock, a best-effort `POST /activate` bumps the note's use on the
  parent — never fatal, never queued. A pulled note is never offered back up unless it's later
  edited locally, which offers it once (B1, the intended "bounce-once").
- **D9 — Merged query is the primary path, notes only.** The child requests `dedupe-keys=1` and
  drops every parent `kind: chunk` item before any other step — the recency channel and chunk
  budgets are local-only, and `show-chunk --parent` is gone. Dedupe runs before ordering; trigger
  hits lead (local then parent), then direct items by score, then explore items by score — fixing
  #744. A note floor keeps a minimum of qualifying note-kind items even under a tight `--limit`.
  The budget block reports merge-applied values — fixing #743. `pending_offers` reflects local
  offers only, since a child can't curate its parent.
- **D10 — Curation gains a fold, and stops re-stamping identity on bookkeeping.**
  `engram amend --discard --into <existing>` unions the offer's basename and aliases into
  `<existing>.aliases`, and folds its parent links into `<existing>`'s: an existing link keeps its
  role and takes the offer's hash; the offer's primary (`offered`/`pulled`) stays primary only when
  `<existing>` has none, otherwise it becomes `covered` — and a fold promotes a held `covered` link
  to the offer's primary role when `<existing>` has no primary of its own. When the offer links a
  different parent vault than `<existing>` does, `<existing>`'s links under the old vault are
  dropped (links count only under the current parent, matching a receipt's own replacement rule).
  `<existing>` keeps its own author; the offer's author is not carried (documented loss, the
  basename survives in `aliases`). A **judged-version check** guards every curation bookkeeping
  step on an offer (`--clear-pending`, `--discard`, `--discard --into`, pending or live): it
  requires `--expect-hash`, and only an **equal** hash passes — a hash of another version, or an
  *unknown*-classified one, fails exactly as a changed one does, so the curator re-judges rather
  than silently accepting or discarding content nobody reviewed. Bookkeeping amends
  (`--activate`, `--clear-pending`, `--discard --into`, a link-only receipt write) no longer
  re-stamp `repo`/`user`/`vault`, so an accepted offer keeps its declared author. `resituate`
  preserves every field it doesn't change.
- **D11 — Tests: DI fakes plus hermetic headless skill arms.** Go unit tests use the `Deps.Fetch`
  seam and FS/lock fakes. Skill TDD (recall/learn/curate) runs each RED/GREEN arm as a fresh
  headless `claude -p` process (never a subagent, which would inherit session context), under an
  `env -i` allowlist, a working directory outside any repo, and two confinement layers: Claude
  Code's own permission allow/deny lists, and its OS sandbox (Seatbelt on macOS) with
  `filesystem.denyRead: ["/Users/joe"]` and `autoAllowBashIfSandboxed: false` (which, if left
  `true`, was confirmed to let sandboxed Bash bypass the `Bash(engram:*)` allowlist entirely).
  **A residual gap was found and closed in the same pass:** the sandbox always allows sandboxed
  Bash *writes* into `/private/tmp/claude-<uid>` — the host session's own harness-script
  directory, which runs outside the sandbox — and neither `filesystem.allowWrite` scoping nor
  `CLAUDE_CODE_TMPDIR` stops that. The required narrowing is `filesystem.denyWrite` listing each
  existing entry of that directory (denying the directory itself breaks Bash), verified every
  batch by a probe that attempts a write into it and must fail. The residual after narrowing: an
  arm can still create *new* top-level files in that directory, and can still *read* anything under
  `/private/tmp`. Each batch's before/after check of the real vault and `~/.claude` (confirming
  nothing there is newer than the batch start) is saved to a file, not just asserted in the
  transcript. Real-binary verification runs `engram serve` over a scratch parent/child pair behind
  a recording TCP proxy; `update`'s drain gets no real-binary run (it re-execs a freshly installed
  binary) and is covered by unit tests only.
- **D12 — Multi-level propagation and the bounce-once.** Accepting a served offer counts as a local
  learn at that level and is enqueued upward (M14), skipped only when the offer's own origin vault
  is the configured parent (no loop in the single-parent tree) and otherwise checked against
  `offer.path` for longer misconfigured cycles. A pulled note folded by curation into a local note
  is offered up once more, as an amend targeting the local note's primary link, or a new learn-offer
  when it has none (B1) — the parent's curated result comes back down only on the next use.

**Consequences.**

- `ENGRAM_SERVER` is gone: every fleet host that set it must switch to `ENGRAM_PARENT` before
  upgrading, and its first command afterward creates an empty local vault.
- Every host that already sets `ENGRAM_PARENT` starts offering its writes on upgrade, so the
  parent's `pending_offers` queue grows by the fleet's write rate — curating it becomes expected
  upkeep, surfaced by the hint and by `engram update`'s notice.
- The real vault gets one deliberate `.engram-vault-id` commit, made the first time `engram serve`
  starts on it (or it first contacts a parent) after the upgrade; `engram update` flags the file
  until it's committed.
- Closes #766 (vault-on-first-use), #743 (zero budget block) and #744 (explore crowding); closes
  #745 by design (every environment now has local clusters, so no cross-vault cluster carrying is
  needed) and #746 as superseded (offers plus `activate --parent` replace it).
- `internal/cli`'s `serve_client.go`, `serve.go`, and the frontmatter structs change; new files
  land for the exchange hash, the outbox/state adapters, and pull-down — all through DI
  (ADR-0013, `targ check-thin-api`).

**Risks / Trade-offs:**

- [Fleet curation-queue growth] → surfaced by `pending_offers` and `engram update`'s notice; the
  proposal's Impact section calls this out as expected upkeep, not a defect.
- [An unlinked near-duplicate note shows twice across a merge] → accepted; a later `activate`
  pull-down and a curation fold heal it through the link graph.
- [A covered fold loses the offer's own author attribution] → documented (D10); the basename
  survives in `aliases`, and the losing side is deliberate (the note's content is now
  `<existing>`'s).
- [Version skew: an upgraded child against a pre-change parent] → an offer's receipt (D6) and a
  pull-down's envelope (D8) are both detected by a missing `vault_id`, and exchange stops with a
  "parent too old" error. A merged query does **not** carry this check (ruling S27, verified in
  code): `runMergedQuery`/`fetchQueryPayload` never inspect the query payload's `vault_id`, so a
  query against a pre-change parent still merges the parent's results, unrefused. Parent chunk
  items are still dropped unconditionally, and the merge's dedupe just finds no match on the old
  payload's missing `vault_id`/`exchange_hash` keys, so a note can show twice instead of being
  deduped — degraded, not refused.
- [`.engram/` is lost — a fresh clone, a cleanup] → child-side exchange pauses with a warning until
  `--claim`/`--regenerate` runs; queued offers re-queue on the next content write; forgotten
  declines can bring a declined parent note back once, for curation to discard again. No note
  content is lost — links and IDs live in tracked files.
- [A remote client aims `offer.for`/`offer.origin` at arbitrary notes] → it can only create or
  update *pending* notes, never a live one; curation reviews every one. The trust model is
  unchanged from today's unauthenticated `serve` (network reachability).

**Rejected alternatives** (design.md :72, :119–123, :158–160, :214–218, :280–283, :319–321,
:355–357, :543, :546):

- **D1.** An alias shim for `ENGRAM_SERVER` (Joe ruled it out), and exempting `serve` from the hard
  error (Q5).
- **D2.** Creating the vault only on `update`, or warning without creating (#766's other two
  options); putting the outbox in the vault root and appending the tracked `.gitignore` (dirties
  the vault repo on every host); keying state under `$XDG_DATA_HOME` by vault path (orphaned on a
  move, doesn't travel with the vault); keying links by URL (breaks on a hostname/IP change, M11).
- **D3.** Extending `embed.ContentHash` (would stale every existing sidecar and force a full
  re-embed); hashing the whole file (identity/link fields would change the hash, and loop
  suppression would never fire).
- **D4.** A single link per note (can't record that a note covers a second parent note once it
  already has a counterpart, H5); keying the outbox/idempotency by basename (a rename loses the
  entry, H2); rewriting outbox keys on rename (works, but `xid` makes it unnecessary); recording
  the child's basename on the parent (remote-set, and the child can rename).
- **D6.** A per-offer snapshot directory (sends stale content and duplicates); JSONL (coalescing
  needs rewrites); sending synchronously and failing the local write (contradicts "local writes
  always succeed"); no backoff (up to 30s per command while the parent is down, M9).
- **D7.** Keeping `/amend` (two offer routes for one concept); a new `/offer` route (duplicates
  `/learn`'s code path); a raw text body for `/show?raw=1` (an old parent's rendered response can't
  be told apart from it, H7).
- **D8.** A `parent:` path prefix in payloads (breaks consumers, including bare `engram show
  <basename>` in the shim); writing the pulled note live (bypasses curation); reusing `/show`'s
  rendered output (it's altered, G4).
- **D12 (M14).** Stopping propagation at the first level — inconsistent with near folds
  propagating, and it defeats the personal → team → org tree.
- **D12 (B1).** A `--from-pull` fold marker that suppresses the bounce-once offer — it would lose
  the local addition upstream, which is the point of decision 3.

Link: `openspec/changes/local-first-parent-sync/{proposal.md,design.md}`; vault notes 784, 784a;
`internal/cli/{serve.go,serve_client.go,serve_learn.go,serve_exchange_read.go,outbox.go,pulldown.go,merged_query.go,exchangehash.go,exchangefields.go,amend_fold.go}`;
#766 #743 #744 #745 #746.
