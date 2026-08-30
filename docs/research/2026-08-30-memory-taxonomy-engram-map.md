> Landscape research (docs/README.md `design/`-charter sibling rule: conclusions graduate to ADR/specs/ROADMAP; file is deletable once extracted). Produced 2026-08-30 by the memory-taxonomy research workflow (4 parallel researchers + critic), explore session on #735 → memory north star. Engram mechanism inventory × taxonomy mapping (repo-verified).

# Engram Memory-Mechanism Inventory × Human Memory Taxonomy

All claims below verified against source at `/Users/joe/repos/personal/engram` (README.md, docs/README.md, docs/GLOSSARY.md, docs/architecture/adr.md ADR-0001..0026, all 35 specs under `openspec/specs/`, all 6 skills under `agent-instructions/skills/`, guidance under `agent-instructions/guidance/`, and `internal/cli/{query,recency,activate,amend,resituate,supersedes}.go`, `internal/chunk/chunk.go`).

---

## Part 1 — Inventory of engram's memory mechanisms

### 1.1 Storage substrates

| Substrate | What it holds | Written by | Key properties |
|---|---|---|---|
| **Chunk index** (`$XDG_DATA_HOME/engram/chunks/`: per-source `.jsonl` + `manifest.json`) | Raw session transcripts (Claude Code JSONL + Pi) and markdown docs, stripped (`internal/context`), split into embedding-sized chunks (`internal/chunk`: per-heading for markdown, `turn-N` anchors for transcripts), each embedded | `engram ingest [--auto]` — mechanical, no LLM judgment | Append-only within a source (a re-chunk never drops prior records); cross-source content-hash dedup with a record-subset eviction gate (ADR-0021); non-persistent-workspace paths and `.claude/jobs` scratch excluded from `--auto` sweeps |
| **Vault notes** (flat vault root, `<luhmann-id>.<YYYY-MM-DD>.<slug>.md`) | Crystallized lessons: `fact`, `feedback`, `runbook`, `qa-question`/`qa-answer`, vocab definition notes | `engram learn` (create-only), `engram amend`, `engram resituate` — always LLM/agent-judged, mostly via the `write-memory` worker skill | Luhmann-ID thought-tree placement (top/continuation/sibling, `internal/luhmann`); wikilink graph (`internal/vaultgraph`); all writes flock-serialized + atomic-rename (ADR-0013) |
| **Embedding sidecars** (`<note>.vec.json`) | Dual vectors: `situation_vector` + `body_vector`, plus `embedding_model_id`, `dims`, `content_hash`, **`last_used`** (YYYY-MM-DD, "drives ACT-R-style recency decay" per GLOSSARY) | Embed-on-write on every learn/amend/resituate; `engram activate` bumps `last_used` only | Bundled MiniLM-L6-v2 384-d, pure Go (ADR-0002/0003); `bestVector` picks the higher-cosine axis at query time |
| **Vocab layer** (`vocab.centroids.json` + `vocab/<term>` tags + definition fact notes) | A controlled vocabulary derived from vault embedding geometry (ADR-0024: whole-vault k-means, growth-only refit trigger ≥40 notes AND ≥14 days) | `vocab bootstrap/propose/refit`; terms auto-assigned on every note write | Feeds recall's explore channel (ADR-0025); unmatched derived terms retired via supersession on the family note (history preserved) |
| **Parent vault** (`ENGRAM_PARENT`) | A tree of vaults (personal → team → org); `engram query` merges parent results into one ranked payload with `from_parent` + `model_id` provenance | remote node | Served writes to any node land as **pending offers** (`pending: true`), excluded from query until the `curate` skill judges them |

### 1.2 The note-kind vocabulary (schemas)

Common frontmatter across kinds: `type`, `date`, `luhmann`, `situation` (the retrieval handle — embedded as `situation_vector`), `source` (free-text provenance), `sources:` (chunk `source#anchor` IDs), `tags:` (hand categorical + auto `vocab/<term>`), `supersedes: [{note, type: updates|narrows|refutes, claim}]` (+ machine-maintained inverse for ride-alongs + a `Supersedes: [[...]]` body line), auto-detected identity `repo:`/`user:`/`vault:`, and `pending:` for served offers.

- **fact** — `subject` / `predicate` / `object` (an S-P-O triple).
- **feedback** — `behavior` / `impact` / `action` (what was done, why it was wrong/costly, what to do instead).
- **runbook** — `done_when` (required) + numbered-steps body that may `[[wikilink]]` fact/feedback notes (ADR-0026; details in Part 3a).
- **qa** — a *pair*: `qa-question` (verbatim question; retrieval-**excluded** via `isQueryExcludedKind`) + `qa-answer` (competes normally), with `contributors` (validated wikilink basenames) and `certainty: high|medium|low` (ADR-0012's asymmetric participation).
- **vocab definition** — an ordinary fact note tagged `[vocab, vocab/<term>]`; body is the term's embedding text.

### 1.3 Commands (what each does to the store)

- **learn** — create-only note write (fact/feedback/qa/runbook) + embed + vocab assign + Luhmann placement.
- **query** — the one retrieval surface: per-phrase recency-biased cosine over notes+chunks → bounded matched set (top-30/phrase, floor 0.25, cap ~300, per-phrase matched-note floor of 5) → one AutoK k-means clustering emitting `candidate_l2s` (within-cluster top-5 notes) → supersession ride-alongs → explore channel (softmax centroid-proximity sampling from vocab terms) → un-clustered recency channel (newest 25 chunks). `--lazy-chunks` defers chunk text to `show-chunk`.
- **query-chunks** — chunk-space-only search.
- **amend** — in-place update: merge chunk provenance (idempotent), overwrite only supplied content fields, re-embed only on content change, `--activate` bumps `last_used`, `--supersedes` writes typed supersession + inverse, `--clear-pending`/`--discard` for curation.
- **resituate** — re-key a note's situation (Part 3c).
- **activate** — bump `last_used` on sidecars of notes the agent actually used.
- **ingest** — the mechanical episodic sweep (1.1).
- **embed apply/status** — bulk (re-)embed + state census (ok/missing/stale/incompatible/broken).
- **show / show-chunk** — read-only fetch (with `--parent` remote resolution).
- **count** — read-only aggregation (tag membership / backlink in-degree), off the similarity path (ADR-0018/0019).
- **prune** — chunk-index housekeeping: detach dead sources (keeping chunks searchable), remove empties, collapse duplicates. *Not* memory forgetting.
- **check** — vault invariants.
- **vocab bootstrap/propose/refit/stats/tag-definitions** — vocabulary lifecycle.
- **serve** — HTTP door for `query, query-chunks, show, show-chunk, activate, learn, amend` only; served writes land pending.
- **update** — deployment-as-sync of skills+guidance (ADR-0022/0023), plus `--reparent-luhmann` batch re-parenting.

### 1.4 Skills and guidance (behavior layer)

- **recall** (glance = read-only 3-phrase; deep = 10-phrase + write side): Step 0 print plan → 0.5 sweep → 1–2 unified query → 2.5 per-cluster covered/near/absent crystallization (deep only) → 2.7 activate used notes → 3 synthesis vs plan → 3.5 re-entry query for emergent recommendations → 4 persist synthesis conclusions + QA capture (deep only).
- **learn**: sweep → vocab liveness → crystallize exactly four moment-kinds (corrections, explicit save-requests, reversals, confirmed approaches — the last routed by shape to runbook vs feedback) → ad-hoc QA capture; mid-cycle fast path skips the sweep; batch mode answers `--reparent-luhmann` candidates.
- **write-memory**: the write worker — composes/executes/verifies handed-off `engram learn` commands; never judges.
- **curate**: judges pending offers covered/near/absent against the host vault; covered/near → enrich existing + `--discard` offer; absent → `--clear-pending`.
- **please**: seven-step end-to-end workflow bracketed by learn/recall, four adversarial review gates, Step-7 lessons audit (maps every STOP/gate-FAIL/CORRECTION-commit/escalation to a note or "no lesson: why").
- **route**: evidence-based tier rubric — every dispatch recorded as a tagged fact note + per-work-kind aggregate; recalled evidence (not a table) sets the next starting tier (ADR-0014/0017/0019).
- **Guidance docs** (`recall.md`, `delegate.md`, `learn.md`): always-loaded `@import` ambient cue lists — *when* to fire recall/delegation/learn.

### 1.5 The 35 capability specs (one-liners)

cli-count (read-only aggregation, Obsidian-verifiable) · eval-warm-config-fidelity (warm eval arms fail fast on bad skill sources) · guidance-delegate (delegation-firing doc) · guidance-recall-moments (recall-firing doc) · ingest-auto-sweep (mechanical sweep + throwaway-path exclusions) · ingest-chunk-dedup (content-hash canonical dedup + record-subset gate) · ingest-prune-detach (prune detaches, chunks stay searchable) · learn-branching-disposition (Luhmann sibling/child/top placement in learn) · learn-qa-capture (asymmetric Q/A pair) · learn-runbook-capture (fourth kind, three-question schema) · please-doc-enumeration-gate (non-waivable doc-surface grep + Gate A verify) · recall-centroid-sampling (exploit + explore halves) · recall-glance-deep-dial (read-only vs full rung; C5 escalation) · recall-matched-note-floor (5 note slots per phrase vs chunk flood) · recall-payload-cuts (--lazy-chunks, --recent-fill) · recall-query-timings (per-stage diagnostics) · recall-runbook-surfacing (runbooks compete symmetrically) · recall-two-channel-payload (relevance + recency channels) · retrieval-probe-signal-fidelity (probes read items[], not candidate_l2s) · route-dispatch-evidence (dispatch records as tagged fact notes) · route-evidence-rubric (recalled evidence sets tier) · update-deploy-sync (owned roots, symlinks, removals propagate) · update-flat-vault-luhmann-notice · update-local-install-safety (downgrade refusal) · update-reparent-luhmann-batch (derive→answer→apply re-parenting) · update-self-reexec · vault-concurrency-safe-writes (flock + atomic rename) · vault-embed-on-write (dual-vector sidecars) · vault-merged-recall (ENGRAM_PARENT) · vault-note-identity (repo/user/vault provenance) · vault-offer-curation (pending offers) · vault-query-model-provenance (model_id in payload) · vault-serve-api (fixed served command set) · vault-vocab-lifecycle (controlled vocabulary) · vault-wikilink-resolution (extension-less basename canonical form) · write-memory-worker (judgment/write seam + capture guards).

---

## Part 2 — Mapping onto the human memory taxonomy

| Engram mechanism | Human-memory analog | Fit | Notes |
|---|---|---|---|
| Chunk index (turn-anchored, timestamped, append-only raw transcripts) | **Episodic memory** (autobiographical traces) | **Strong** | Explicitly framed as "the L1 evidence layer" (ADR-0008 supersession note). Caveat: human episodic memory is reconstructive/gist-based; engram's is verbatim replay. Narrative episode notes were deliberately retired (2026-06-19). |
| Channel 2 recency channel ("re-immerse in recent work"; "recent items are your own recent activity") | **Working-memory reinstatement** after interruption / context loss | Partial | Restores content into the context window; engram itself has no WM store — the harness context window is WM, and engram only manages what enters it (payload budgets, lazy chunks). |
| fact notes (S-P-O triples), qa-answer notes, wikilink graph + Luhmann tree | **Semantic memory** (declarative knowledge network) | **Strong** | S-P-O is literally the classic semantic-network representation; Luhmann continuation/sibling = associative structure; supersession links = typed belief relations. |
| feedback notes (behavior/impact/action) | Semantic memory of **error-corrective lessons** (episodic-derived schemas) | Strong | The dominant note population; situation-keyed. |
| runbook notes (situation / steps / done_when) | **Procedural memory — declarative form** ("knowing *that* the steps are…") | Partial | Retrieved and *read*, then deliberately executed by the LLM; no compilation/automatization. Closer to a written checklist than a motor program. Symmetric retrieval (no exclusion, no boost) per recall-runbook-surfacing. |
| route evidence loop (dispatch records → recalled evidence sets next tier "via recall, not by editing this file") | **Procedural/skill learning** (behavior tuned by accumulated outcomes) | **Strong** (the best procedural-learning fit in the system) | Genuine closed loop: experience → record → changed future behavior without reprogramming. |
| SKILL.md files + guidance docs | **Overlearned procedures / habits** | Partial | They behave like automatized skills but are hand-authored and TDD-edited, not learned; no promotion path from vault memory into a skill. |
| Guidance firing cues (recall.md/delegate.md/learn.md: "run /recall glance before you proceed at these cues") + skill `description` trigger blocks | **Prospective memory** (event-based: remember-to-X-when-Y) | Partial (weak) | Static, hand-authored, always-loaded cue lists. No mechanism for vault content to install a *new* future trigger; retrieval is query-pull, never event-push. Your own eval memory records firing as the hardest gap (over-fire 147x at pinned fire-units; ~28% capture-ceiling). |
| `situation:` field as retrieval handle ("phrase it the way a future task would be described") | **Encoding specificity / transfer-appropriate processing** — and a retrieval-time stand-in for prospective cueing | Strong (as encoding), Partial (as prospective) | The when-to-use key only fires if a query happens at that moment. |
| `vocab refit_pending` trigger, QA round-2 gate line, update's pending-offer/duplicate-backlog notices | Prospective memory — **system-internal reminders** | Partial | Real condition-checked triggers, but only for engram's own housekeeping, not for task knowledge. |
| `engram activate` / `last_used` + `recencyMultiplier` (exp2(−age/60d), age = days since last_used-else-created) | **Retrieval-strength decay + use-based strengthening** (ACT-R base-level activation; testing effect) | Partial | Affects ranking only; single bump resets the clock fully (no frequency accumulation, no strength variable); selective activation ("only what you used") deliberately lets superseded notes fade in rank. |
| `engram amend` at recall Step 2.5C (covered → provenance-enrich+activate; near → re-synthesize from recency-weighted members) | **Reconsolidation** (retrieval destabilizes the trace; it's re-stored updated) | Partial→Strong | The clearest reconsolidation analog anywhere in the design — but deep-mode-only (glance is read-only) and only for cluster candidates. `resituate` is the manual re-keying variant. |
| `--supersedes updates|narrows|refutes` + inverse + ride-along surfacing + recall's recency-weight conflict rule ("recent wins") | **Belief revision / interference resolution** | **Strong** | Old traces persist but arrive flagged with their correction — retroactive-interference handling by annotation rather than overwrite. |
| Recall Steps 2.5/4 + learn Step 2 (chunks → clusters → agent-judged crystallization → notes with `--chunk-source` back-links) | **Episodic→semantic consolidation** (systems consolidation) | Partial→Strong | The core loop exists and preserves provenance. But it is entirely retrieval/event-triggered — no background ("sleep") pass; an episode that never matches a query and never coincides with a lesson-moment never consolidates. |
| Recall Step 4 abduction marking ("probable / best-explanation / defeasible", `--source "synthesis (abduction)…"`) + qa `certainty` | **Metamemory / source monitoring** (confidence + origin tagging) | Strong | Also `repo:`/`user:`/`vault:`/`model_id`/`from_parent` provenance. No *per-note efficacy* metric though (#718 open). |
| Explore channel (centroid-proximity softmax sampling of unqueried vocab-neighbor notes) | **Spreading activation / priming** | Partial | Surfaces associatively-near content the query didn't name — a deliberate exploit/explore split. |
| Vocab lifecycle (geometry-derived terms, refit, retirement) | **Category/schema formation** (concept extraction from experience) | Partial→Strong | Categories genuinely *derived* from the vault's own geometry (ADR-0024), not hand-imposed. |
| Pending offers + curate (covered/near/absent on served writes) | **Social memory gating** — evaluating testimony before integrating it into belief | Strong (for what it is) | "A served write is a caller outside the host asking to contribute, not the vault owner's own act of record." |
| ENGRAM_PARENT merged recall (personal→team→org tree) | **Transactive/collective memory** | Partial | Read-merge only; contribution upward goes through the offer gate. |
| Luhmann re-parenting (`update --reparent-luhmann`), `prune`, dedup | **Memory reorganization / housekeeping** | Partial | Structural re-org exists; content-level forgetting does not (see below). |

---

## Part 3 — Human-memory functions with NO (or only partial) engram analog

1. **Prospective memory / firing — the biggest gap.** Nothing in the vault can arm a future trigger. All firing is static prose (guidance cues, skill descriptions) or the agent's own discipline; a lesson saying "next time you see X, do Y" only helps if something *else* causes a query near X. Partial precursors: guidance docs, `situation:` keying, vocab `refit_pending`. (Your own eval corpus: hooking recall "before tool calls" measured 147x over-fire — the push-side is known-hard, not merely unbuilt.)
2. **Forgetting / decay of knowledge.** No note is ever weakened, archived, or deleted by the system. Recency decay is *rank-only* (gentle: half-life 60d); superseded notes persist and are deliberately re-surfaced as ride-alongs; `prune` is chunk-index GC, explicitly "not part of the recall/learn/please flows." Only vocab terms actually get retired. #718 (efficacy tracking → "rank/prune by them") is the open issue pointing at real forgetting.
3. **Reconsolidation / update-on-use — partial only.** Exists (amend at recall 2.5C, resituate) but gated to deep mode and cluster candidates; glance-mode use (the mode guidance says to fire *often*) touches only `last_used`.
4. **Working-memory management.** Absent as a mechanism. Engram budgets what enters the context window (limit/lazy-chunks/recent-fill/content-budget) but has no model of what's *currently held*, no displacement tracking, no rehearsal. Recall Step 0 (externalize the plan before retrieval) is the one WM-discipline element, and it lives in prose.
5. **Composition of multiple procedures.** Absent — see Part 4b.
6. **Episodic→semantic consolidation maturity.** Retrieval-triggered and moment-triggered only. No batch/offline consolidation over unretrieved episodes; no "replay." The learn skill explicitly forbids reconstructing the episode workflow. Consolidation coverage is therefore biased toward what gets queried and what produces explicit correction-moments.
7. **Skill automatization / proceduralization.** Absent. No pipeline promotes a repeatedly-successful runbook (or memory cluster) into a skill, and no demotion of skills back to notes. (Joe's global CLAUDE.md describes such a tier-promotion architecture as aspiration; engram does not implement it. The open #728 adapter experiment probes the *reverse* — moving skill content into vault function notes.)
8. **Frequency/strength-based memory (base-level learning).** `last_used` is a date, not a count; ten activations equal one. No per-note usage-outcome record (#718).
9. **Salience/importance weighting at encoding.** All notes encode equal; nothing like emotional tagging or surprise-weighted encoding (capture gates are binary: crystallize or don't).
10. **Reconstructive/gist episodic recall.** Chunks replay verbatim; gist exists only transiently in the agent's synthesis unless Step 4 persists it.

---

## Part 4 — The four precise questions

### (a) Runbook schema fields and why (ADR-0026; learn-runbook-capture/spec.md; recall-runbook-surfacing/spec.md)

Frontmatter: `type: runbook`, `date`, `luhmann` (full participation — same `nextLuhmannID` top/continuation/sibling machinery as fact/feedback, no opt-out), **`situation`** (required), **`done_when`** (required), `source`, optional `contributors`; body = **numbered steps**, which MAY `[[wikilink]]` fact/feedback notes to read along the way. Plus the cross-kind fields (tags/vocab auto-assign, chunk `sources:`, `supersedes`, repo/user/vault identity) and a dual-vector sidecar — no excluded half, unlike qa.

The schema answers **exactly three questions**: *when should you use this runbook* (`situation` — embedded as the situation vector, so retrieval is purely situation-similarity, symmetric with fact/feedback), *what are the steps* (body), *what should be true when you're done* (`done_when` — required partly to make runbooks **scorable** for #718's future efficacy tracking, which is deferred, not shipped).

Two fields were explicitly **rejected**: `task_type` + a `--task-type` query pre-filter (the SPL benchmark numbers motivating it measured SPL's whole bundled system with no ablation isolating type-classification, so they couldn't justify a new ranking mechanism and its regression risk), and `inputs`/calling-convention (considered during the #728 adapter experiment, rejected). The kind-prefixed qa-style filename was also rejected — qa is the deliberate special case (Luhmann-skipping, partially retrieval-excluded), the wrong template.

### (b) Any mechanism for composing/merging multiple retrieved runbooks?

**No.** Verified absent: runbooks rank independently in the matched set (no bundling, no boost, no query flag); recall's write side enforces *one representative note per cluster*; `amend` edits a single note; `supersedes` relates notes but replaces rather than composes; there is no `inputs`/`returns` contract to compose against (rejected, above); the spec's wikilink allowance is steps → *fact/feedback* notes (knowledge references), not runbook → runbook chaining. If two runbooks surface for one task, any interleaving happens ad hoc in the agent's context and is never recorded. Nearest adjacent work: **#728 (OPEN)** — the adapter/function-note microkernel spike — adds call-by-name *invocation* of a single procedure note, still not composition/merging.

### (c) What `engram resituate` actually does (`internal/cli/resituate.go`)

`engram resituate --note <ref> --situation <text>` (both required; no `--dry-run`; host-only — never served): under the vault flock, resolves the note ref (basename / wikilink / `.md` / bare Luhmann id), then rewrites the situation in **both places it lives** — the `situation:` frontmatter field and the body's opening prose formula (fact and feedback shapes; unknown types error) — writes atomically, **re-embeds** the note so the sidecar's `situation_vector` and `content_hash` track the new text, then re-runs vocab term assignment and the refit-trigger check. It exists because `engram learn` is create-only, so the two situation copies could previously drift (the INV-S2 divergence): it is the **re-keying** operation — changing *when a memory fires* without touching what it says. (The please skill's Step-7 audit is its main consumer: "if a note existed but did not surface… reword the note — that beats writing a second note.")

### (d) The full chunks→notes consolidation path and its trigger points

**Stage 1 — mechanical episodic capture (no LLM):** `engram ingest --auto` stats every known source, strips transcripts (`internal/context`), chunks (`internal/chunk`: heading-sections / `turn-N`), embeds each chunk, appends to per-source `.jsonl` indexes (append-only within source; cross-source hash dedup). *Triggers:* learn Step 1 (every cycle-opening and cycle-closing learn — the closing learn ALWAYS sweeps), recall Step 0.5 (unless already swept this session). Explicitly forbidden mid-task (mid-cycle learn fast path skips it).

**Stage 2 — the retrieval joint:** `engram query` ranks chunks and notes in one recency-biased cosine space, clusters the matched set (AutoK k-means), and emits per-cluster `candidate_l2s` — existing notes nominated as possible crystallizations of what the cluster's chunks evidence.

**Stage 3 — agent-judged crystallization (the actual episodic→semantic step; LLM does all abstraction):**
- **recall deep Step 2.5C** (per cluster, and per explore item): *covered* → `amend --activate --chunk-source …` (provenance-enrich only); *near* → `amend` re-synthesizing content from the recency-weighted members; *absent* → write-memory → `engram learn` (one fact-or-feedback note per cluster). Every path passes `--chunk-source <source#anchor>` so the semantic note back-links its episodic evidence. Glance mode skips this entirely.
- **recall Step 4**: a sound non-trivial synthesis conclusion → a derived note (`--source "synthesis (abduction)…"`, certainty by inference mode) + a QA pair when the synthesis cites wikilinks.
- **learn Step 2**: the four explicit lesson-moments (corrections, save-requests, reversals, confirmed approaches — shape-routed to runbook vs feedback) → write-memory → `engram learn`; mid-cycle correction fast path fires at the moment of correction.
- **learn Step 2.5**: ad-hoc QA capture (≥1 wikilink bar).
- **please Step 7 lessons audit**: mechanical sweep of STOPs / gate FAILs / CORRECTION-commits / escalations → unmapped items become reversal handoffs to the closing learn (also asks whether an existing note should be *resituated* instead of duplicated).
- **curate**: served pending offers judged covered/near/absent → fold/discard/accept.

**Stage 4 — post-consolidation maintenance:** vocab auto-assign at write + geometry refit; `activate` keeps used notes warm; `amend --supersedes` revises beliefs; `resituate` re-keys; `--reparent-luhmann` re-organizes the tree.

**Character of the pipeline:** provenance-preserving and judgment-gated, but entirely **pull-triggered** — consolidation happens only where a query lands or an explicit lesson-moment fires; there is no background pass over never-retrieved episodes.