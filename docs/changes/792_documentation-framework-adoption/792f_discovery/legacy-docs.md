# Legacy Documentation Inventory → Capabilities & Components

Scope read: every file under `docs/deprecated/` (adr.md skimmed by ADR heading/status/decision; c1/c2/c3
by section), `docs/GLOSSARY.md`, `docs/ROADMAP.md`, all **53** `openspec/specs/*/spec.md` (purpose +
every requirement header), and `openspec/changes/archive/` directory names (42). Framework read:
`docs/AGENTS.md`, `docs/SPECIFICATION-LIFECYCLE-GUIDE.md`, `docs/TECHNICAL-SPEC-GUIDE.md` (Component
Qualification), `docs/PRODUCT-SPEC-GUIDE.md` (capability rules).

Note: the brief said 56 openspec specs; there are **53** (`ls openspec/specs | wc -l`). `openspec/changes/`
holds only `archive/` (no active change).

Users = Joe (operator) and the LLM agents running in a harness (Claude Code, Pi).

---

## 1. Proposed CAPABILITIES (`docs/specs/product/<name>.md`)

| # | Capability | One-line description | Legacy sources |
|---|---|---|---|
| P1 | `memory-recall` | An agent runs the recall procedure (glance / deep rungs) to surface relevant memory, judge each cluster's coverage (covered/near/absent), apply it to its plan, and mark used notes. | openspec `recall-glance-deep-dial`; `recall-payload-cuts` req "Agent reads chunk content before judging cluster relevance" (graduated from superpowers/plans/…733…); c1 "Flow: recall"; c2 "Recall-time lazy-L2 synthesis", "Flowchart: lazy-L2 coverage decision"; ADR-0001, ADR-0005 (superseded history); GLOSSARY §Recall (recall modes, Step 0…, honored-when-fired, surfaced notes) |
| P2 | `memory-query` | `engram query` (+ `show`, `show-chunk`, `query-chunks`) returns one two-channel payload: clustered relevance (notes+chunks, note floor, explore half, trigger hits) plus an un-clustered recency channel, with lazy chunk content, limits, timings and model provenance. | openspec `recall-two-channel-payload`, `recall-matched-note-floor`, `recall-centroid-sampling`, `recall-payload-cuts`, `recall-query-timings`, `vault-query-model-provenance`, `recall-runbook-surfacing`, `runbook-lexical-triggers` (query side); c1 "Flow: recall" (Channel 1/2 detail); c2 C2 catalog row, "Flowchart: recall query pipeline"; c3 K6, "Flow: engram query internals", "Data contracts" (K6 payload); ADR-0011 (superseded), ADR-0025; GLOSSARY §Recall (matched set, matched-note floor, relevance floor, recency channel, lazy-chunks, candidate_l2s, cluster, provenances, trigger, trigger hit), §Vocab "explore sampling" |
| P3 | `lesson-capture` | The learn skill decides which confirmed lessons from a session (corrections, save-requests, reversals, confirmed approaches; subagent `LESSONS:` lines) become notes, where they sit in the Luhmann hierarchy, and hands them to write-memory. | openspec `learn`, `learn-branching-disposition`, `learn-adhoc-qa-capture`, `write-memory-worker` req "Learn skill SHALL capture four kinds…"; c1 "Flow: learn", "engram engagement" flowchart; c2 "Flowchart: learn capture kinds"; ADR-0015; GLOSSARY §Learn (Step 2, reversal, confirmed approach) |
| P4 | `vault-note-writing` | Agents/operators create and modify vault notes via `engram learn fact|feedback|runbook`, `amend`, `resituate`, `activate` (supersession, tags, chunk-source provenance, use bumps), executed through the write-memory worker. | openspec `write-memory-worker`, `vault-embed-on-write` (product half), `vault-wikilink-resolution`, `vault-concurrency-safe-writes` (product guarantee); c2 C1→C2/C2→C4 rows; c3 K4, K4w, "Flow: engram learn write internals"; ADR-0003, ADR-0013, ADR-0015, ADR-0019; GLOSSARY §Learn flags (`--target/--position`, `--supersedes`, `--tag`, `--source`, `--project`, `--issue`), write-memory, handoff contract |
| P5 | `question-answer-memory` | `engram learn qa` captures a linked Q/A pair where the answer competes in retrieval and the question is excluded (D5′ asymmetry). | openspec `learn-qa-capture`, `learn-adhoc-qa-capture`; c3 "Flow: engram learn qa capture path"; ADR-0012; GLOSSARY qa-question, qa-answer, D5′, contributors |
| P6 | `runbook-memory` | Task-shaped "how to approach X" notes (`type: runbook`, steps/done_when/red_flags/triggers) are captured and surface in recall by situation similarity or literal trigger hit. | openspec `learn-runbook-capture`, `recall-runbook-surfacing`, `runbook-lexical-triggers`; ADR-0026; GLOSSARY runbook, trigger, trigger hit |
| P7 | `runbook-follow-frame` | Deployed shim guidance makes an agent query engram first on every request and follow a matched runbook (announce, restate, red_flags, done_when, stop-and-ask, wikilinked runbooks). | openspec `guidance-runbook-follow-frame`, `production-guidance-activation` (shim import); ADR-0026 (follow-frame portion); GLOSSARY guidance file |
| P8 | `skill-runbook-registration` | `engram update` / `engram register-skills` offer, refresh, adopt and remove one mirrored runbook note per skill/command/prompt found in every default harness skill folder, remembering declines. | openspec `skill-runbook-registration`, `update-deploy-sync` req "Update SHALL run registration offers"; c1 S3 catalog; c2 C1 catalog; ADR-0015 Amendment (2026-09-26); AGENTS.md (root) registration paragraph |
| P9 | `memory-firing-guidance` | Ambient guidance docs fire recall (`recall.md`) and learn (`learn.md`) at defined moments, incl. learn at subagent return. | openspec `guidance-recall-moments`, `guidance-learn-moments`, `production-guidance-activation`; ADR-0001 (cited "why"); GLOSSARY guidance file |
| P10 | `delegation-guidance` | Ambient guidance (`delegate.md`) makes the agent plan, route, review and report every unit of work instead of doing it solo. | openspec `guidance-delegate`, `production-guidance-activation` |
| P11 | `end-to-end-task-workflow` | The please skill runs a 7-step gated workflow (opening learn, orient, plan + gate A, TDD execute + gate B, document + gate C, complete + gate D, closing learn), incl. the doc-surface enumeration gate and LESSONS collection. | openspec `please`, `please-doc-enumeration-gate`; c1 "Flow: please", "/please seven-step gated workflow", "Companion: who reviews at gates A–D"; GLOSSARY lessons audit, surprise harvest, escalation provenance |
| P12 | `subagent-dispatch-routing` | The route skill picks agent type/model tier/effort from recalled evidence (cheapest-first, memory discount, spec-first retry), records every dispatch as tagged evidence + aggregates, and mandates the LESSONS line in each handoff. | openspec `route`, `route-evidence-rubric`, `route-dispatch-evidence`; ADR-0014, ADR-0017, ADR-0019 (tags as evidence categories); c1 "Out of scope at L1" (route note); GLOSSARY §Route evidence |
| P13 | `subagent-lessons-contract` | Every dispatched subagent's completion report ends with a `LESSONS:` line; orchestrators re-ask once if missing and pass all lines to the closing learn. | openspec `lessons-contract` (+ the LESSONS slices of `learn`, `please`, `route`, `guidance-learn-moments`) — see Overlaps |
| P14 | `pending-offer-curation` | The curate skill judges pending offers (served from children, pulled from parent, skill registrations) as discard / fold / accept, using recall's covered/near/absent reasoning. | openspec `vault-offer-curation`; c2 C1 catalog (curate); ADR-0029 D10 |
| P15 | `parent-vault-exchange` | With `ENGRAM_PARENT`, a local-first vault merges parent notes into queries, offers its writes upward via an outbox, and pulls parent notes down on activate. | openspec `vault-local-first`, `vault-merged-recall`, `vault-parent-offers`, `vault-parent-pulldown`, `vault-note-identity` (xid/parent/aliases reqs); c1 S7, R7; c2 C2→S7, C4 (`.engram-vault-id`, `.engram/`); c3 K14; ADR-0029 (D1–D12); GLOSSARY §Parent sync |
| P16 | `vault-serving` | `engram serve` exposes a fixed HTTP API (query, show, activate, learn) so child vaults can exchange with this one. | openspec `vault-serve-api`; c1 S7; c3 K14 (serve side); ADR-0029 D7 |
| P17 | `vault-identity-management` | Every vault has a stable vault ID (`engram vault-id`, `--regenerate`/`--claim`) and is created on first use; copied/cloned/self-parented vaults are refused. | openspec `vault-local-first`; ADR-0029 D2; GLOSSARY vault ID, ENGRAM_SERVER (removed); README.md (root) | 
| P18 | `note-provenance` | Each note carries auto-detected repo/user/vault identity fields (with backfill), re-stamped on amend. | openspec `vault-note-identity` (repo/user/vault/backfill/skill_hash reqs) |
| P19 | `session-ingestion` | `engram ingest --auto` sweeps session transcripts and markdown into a searchable chunk index (change-detected, deduped by content, skipping throwaway prefixes); `engram prune` detaches/cleans entries without losing memory. | openspec `ingest-auto-sweep`, `ingest-chunk-dedup`, `ingest-prune-detach`; c1 R4, "Flow: learn" (ingest step); c2 "Flowchart: ingest/chunking", C2→S5; c3 K1, K12; ADR-0008/0009/0010 (superseded history), ADR-0021; GLOSSARY §Transcript |
| P20 | `controlled-vocabulary` | Notes are auto-tagged with vocab terms on write; the vault flags when a refit is due and `engram vocab` (bootstrap/propose/stats/refit/tag-definitions) derives and applies a new vocabulary. | openspec `vault-vocab-lifecycle`; c2 "Flowchart: vocab lifecycle"; ADR-0011 (superseded), ADR-0019, ADR-0024; GLOSSARY §Vocab, vocab.centroids.json fields |
| P21 | `note-hierarchy-placement` | Notes are placed as Luhmann children/siblings; `engram update` flags an all-flat vault and `--reparent-luhmann` re-parents existing notes via derive → agent answers → apply. | openspec `learn-branching-disposition`, `update-flat-vault-luhmann-notice`, `update-reparent-luhmann-batch`; GLOSSARY Luhmann ID, `engram update` (reparent) |
| P22 | `engram-update` | `engram update` installs/upgrades the binary (local or remote clone, downgrade guard), re-execs it, syncs skills/commands/opt-in guidance into each harness, runs registration offers and maintenance notices, with dry-run. | openspec `update-self-reexec`, `update-local-install-safety`, `update-deploy-sync`, `ingest-chunk-dedup` req "update SHALL detect backlog"; c1 R5, R6, "Flow: update"; c2 C2→S6; c3 K9; ADR-0022, ADR-0023; invariant H/U1; GLOSSARY `engram update`, XDG paths |
| P23 | `vault-aggregation` | `engram count` gives read-only frontmatter membership counts (`--group-by/--filter`) and wikilink backlinks (`--backlinks-of`). | openspec `cli-count`, `route-dispatch-evidence` req "Count SHALL audit aggregate drift"; ADR-0018, ADR-0019; c1 S2 catalog; GLOSSARY `engram count` |
| P24 | `vault-integrity-check` | `engram check` reports vault-invariant violations (unresolved/dangling links, embedding presence, etc.). | memory-invariants.md §A–D + "What the checker must implement"; memory-system-rigor.md §3; c2 C2 catalog (`check`); c3 K7 (`UnresolvedTargets`); openspec `vault-wikilink-resolution` (G3 scenario). **No openspec spec** — must be derived from code |
| P25 | `embedding-migration` | `engram embed status/apply` reports and repairs missing/stale/off-model sidecars. | c3 K5b; c2 C3 catalog (M4); ADR-0002, ADR-0003; memory-invariants M4; openspec `vault-embed-on-write` (state classification). **No dedicated openspec spec** |
| P26 | `memory-effectiveness-evaluation` *(dev-facing; ambiguous — see §4)* | Joe measures whether memory helps: isolated eval trials (warm configs, vault-leak guard, fixture scoring), retrieval probes, and the memory-loop audit over real transcripts. | openspec `eval-warm-config-fidelity`, `eval-trial-vault-leak-detection`, `eval-phase2-fixture-scoring-fidelity`, `retrieval-probe-signal-fidelity`, `memory-loop-audit`; ADR-0027, ADR-0028; FEATURES.md "Validated goals"; dev/eval/LEDGER.md (not in scope, referenced everywhere) |
| P27 | `automatic-memory-agent` *(not shipped — in-flight)* | A parallel process recalls and captures memory automatically at moments, independent of the working agent's compliance. | design/parallel-memory-agent-requirements.md (REC-1..7, CAP-1..6, CTL-1..4, IND-1..3, NFR-1..3). Belongs in `docs/changes/<issue>_…/` as a change product spec, not canonical |

Candidates considered and folded (not separate capabilities): `engram-command-template-safety` → cross-cutting
MUST in the instruction-artifact component (C18) plus a product MUST in P4/P9 if desired; `vault-concurrency-safe-writes` →
product guarantee inside P4/P19, technical in C3; `recall-query-timings` → P2.

---

## 2. Proposed COMPONENTS (`docs/specs/technical/<name>.md`)

Each passes the five qualification criteria unless flagged. "Realization" is informational only.

| # | Component | Responsibility (contract) | Supports | Legacy sources |
|---|---|---|---|---|
| C1 | `vault-note-store` | Read/write/scan notes in the flat vault: filename shape `<luhmann>.<date>.<slug>.md`, frontmatter schema per kind (fact/feedback/runbook/qa), LF normalization, identity/exchange fields survive rewrites, pending marker. Realization: `internal/cli` note/frontmatter code, `vaultgraph.ScanVault`. | P2–P6, P14–P18, P21, P23, P24 | c2 C4 row; c1 S4; openspec `vault-note-identity`, `vault-offer-curation` (pending marker), `update-reparent-luhmann-batch` (CRLF req); invariants S2, M5; GLOSSARY §Vault structure |
| C2 | `luhmann-id-allocator` | Parse/order Luhmann IDs; allocate the next unique top-level/child/sibling ID; uniqueness. Realization: `internal/luhmann`, `cli/luhmann.go`. | P4, P21 | c3 K10; invariants M8, K1(concurrency); GLOSSARY Luhmann ID |
| C3 | `vault-write-lock` | Exclusive flock on `.luhmann.lock` (notes) and `.manifest.lock` (chunk index) held across read-modify-write; atomic temp-file+rename so readers never see torn files. | P4, P19, P15, P16 | openspec `vault-concurrency-safe-writes`; ADR-0013; c2 C2→C4; c3 K4/K12; invariant G/K1 |
| C4 | `embedding-sidecar` | Dual-vector `.vec.json` (situation + body, model id, dims, content_hash, last_used); stale/missing/incompatible classification excluding additive metadata; best-axis selection. Realization: `internal/embed`. | P2, P4, P25 | openspec `vault-embed-on-write`; ADR-0003, ADR-0006 (superseded); c3 K5, "Data contracts" K5; invariants E1, M4 |
| C5 | `sentence-embedder` | External-library boundary: deterministic 384-d MiniLM-L6 embeddings in pure Go (go:embed model, Hugot/GoMLX simplego). Qualifies as a meaningful library boundary. | P2, P4, P19, P20, P25 | ADR-0002; c1 "Out of scope at L1"; c2 C3; AGENTS.md design principles |
| C6 | `chunk-index` | Per-source chunk record files + `manifest.json` staleness (mtime/size/hash), merge-append, content-hash+class dedup with canonical precedence and record-subset eviction gate, prune detach/`--empty`/`--duplicates`; directory-scan discovery. | P19, P2 | openspec `ingest-chunk-dedup`, `ingest-prune-detach`, `ingest-auto-sweep`; ADR-0021; c2 ingest flowchart; c3 K1, K12 |
| C7 | `session-source-reader` | Find and read harness session sources (Claude Code / Pi JSONL, markdown, sweep roots incl. non-persistent prefix exclusion, `.claude/jobs` exclusion) and strip harness noise. Realization: `internal/transcript`, `internal/context`, `sweepspec.go`. | P19 | openspec `ingest-auto-sweep`; ADR-0010 (partially superseded); c1 S5/R4; c3 K1; GLOSSARY transcript, session, non-persistent workspace |
| C8 | `text-chunker` | Pure split of stripped transcripts/markdown into byte-budgeted chunks. Realization: `internal/chunk`. *Qualification borderline* — has an independent pure contract; could fold into C6/C7. | P19 | c2 ingest flowchart (chunk step); AGENTS.md dir map |
| C9 | `query-retrieval-pipeline` | Owns the query payload contract: per-phrase recency-biased scoring, note floor, relevance floor, matched-set cap, clustering call, candidate_l2s, explore half by centroid proximity, supersession ride-alongs, trigger hits first, recency channel, lazy chunks, `--limit`, timings, model_id, merged-payload budget/order. | P2, P1, P6, P15 | openspec all `recall-*` specs, `runbook-lexical-triggers`, `vault-query-model-provenance`, `vault-merged-recall` (ordering/floor reqs); c2 recall pipeline; c3 K6, "Flow: engram query internals", "Data contracts"; ADR-0025; invariants R1 |
| C10 | `vector-clustering` | Deterministic k-means + silhouette AutoK over bounded inputs. Realization: `internal/cluster`. | P2, P20, P21 | c3 K8; invariant C1; ADR-0024 (silhouette auto-K for refit) |
| C11 | `vocabulary-engine` | Write-time term assignment (cosine floor, top-3), growth-only refit trigger state in `vocab.centroids.json`, derivational refit (cluster → centroid match → naming requests → apply/retire), definition self-tags. | P20, P2 (explore centroids) | openspec `vault-vocab-lifecycle`; ADR-0024, ADR-0019, ADR-0011; c2 vocab flowchart; GLOSSARY §Vocab |
| C12 | `wikilink-graph` | Parse wikilinks, resolve by extension-less basename (tolerating `.md`), report unresolved targets, compute backlinks/in-degree. Realization: `internal/vaultgraph`. | P23, P24, P4 | openspec `vault-wikilink-resolution`; ADR-0007; c3 K7, "Dead/test-only surface" (stale — see §4); invariants G0, G2, G3 |
| C13 | `vault-invariant-checker` | Evaluate vault invariants and report violations (`engram check`). | P24 | memory-invariants.md "What the checker must implement"; memory-system-rigor.md §3 |
| C14 | `vault-aggregation-counter` | Frontmatter membership counts with AND filters; backlinks mode; mutual exclusivity. | P23 | openspec `cli-count`; ADR-0018 |
| C15 | `harness-deploy-sync` | Engram-owned root per harness with `.engram-owned` marker, sync to intended deploy set (add/update/delete), symlink surfaces, dangling cleanup, first-sync migration, manifest-mode fallback, guidance opt-in + compat symlinks, dry-run classification. Realization: `internal/update`. | P22, P7, P9, P10 | openspec `update-deploy-sync`; ADR-0022 (D1–D9); c1 R6; c2 C2→S6; c3 K9 |
| C16 | `self-installer` | Resolve local clone vs remote clone, `go install`, report revision, refuse provable downgrades, handoff report + loop-guarded re-exec with in-process fallback. Boundary: Go toolchain + git. | P22 | openspec `update-self-reexec`, `update-local-install-safety`; ADR-0023; c1 R5/S6; c3 K9; invariant U1 |
| C17 | `skill-registration-scanner` | Scan the default skill/command/prompt sources, derive source-qualified keys, collapse identical copies, report conflicts, compute offers (new/refresh/remove), persist declines (`skill-registrations.json` v2), terminal gating. | P8 | openspec `skill-runbook-registration`; c1 S3; ADR-0015 amendment |
| C18 | `agent-instruction-artifacts` *(ambiguous)* | Shipped skills + guidance markdown as a contract surface: command-template safety (`--flag="value"`), handoff contract, LESSONS format. May be better modeled as one component per skill (C22–C27) — see §4. | P1, P3, P7, P9–P13 | openspec `engram-command-template-safety`, `lessons-contract`; ADR-0001 |
| C19 | `parent-exchange-client` | Offer classification, exchange hash, payload translation, durable outbox, drain-without-lock, backoff, receipts, merged-query fetch/dedupe/origin tagging, pull-down on activate with decline memory and loop guard. | P15 | openspec `vault-parent-offers`, `vault-merged-recall`, `vault-parent-pulldown`; ADR-0029 D3–D9, D12; c3 K14 child side; GLOSSARY offer, outbox, exchange hash, parent link, pull-down, xid |
| C20 | `vault-http-api` | Served endpoints (`/query`, `/show`, `/activate`, `/learn`), explicit bind, caller-declared identity, size caps, in-place pending updates for same origin, field-setting refusals, loop refusal. | P16 | openspec `vault-serve-api`; ADR-0029 D7; c3 K14 serve side |
| C21 | `vault-identity` | Vault ID file, first-use vault creation, exchange-state dir outside tracked files, copy/clone detection, self-parent refusal, `ENGRAM_SERVER` hard error. Could fold into C19. | P17, P15 | openspec `vault-local-first`; ADR-0029 D1/D2; c2 C4 row |
| C22 | `pending-offer-lifecycle` | Pending marker semantics: served/pulled writes land pending, excluded from queries, stateless scan, surfaced at three points, judged-version verification, fold parent-link merge. (Curate judgment itself is skill-side.) | P14, P8, P15 | openspec `vault-offer-curation`; ADR-0029 D10 |
| C23 | `cli-command-router` *(qualification check)* | Owns caller-side routing of subcommands/flags to components, entry-point lock acquisition, stdout/stderr contracts. Qualifies only for its interaction requirements (ordering/gating); otherwise it is wiring. | all CLI capabilities | c3 diagram (process boxes), K13; ADR-0020 |
| C24 | `debug-log` *(borderline)* | Env-gated, tail-friendly debug sink across targets. Probably MAY-level; qualifies only if a product MUST depends on it. | operator diagnostics | c3 K11 |
| C25–C30 | Skill procedures: `recall-procedure`, `learn-procedure`, `write-memory-worker`, `please-procedure`, `route-procedure`, `curate-procedure` *(ambiguous — see §4)* | LLM-executed procedures with stateable contracts (inputs, handoffs, outputs), directly testable via headless skill evals. | P1, P3, P4, P11, P12, P14 | openspec `write-memory-worker`, `learn`, `please`, `route`, `recall-glance-deep-dial`; ADR-0001, ADR-0015; c2 "skills↔binary split" |
| C31 | `eval-harness` *(dev)* | Isolated trial env construction (warm configs from verified skill sources), vault-leak guard, fixture scoring, surfacing probes. | P26 | openspec `eval-*`, `retrieval-probe-signal-fidelity` |
| C32 | `memory-loop-auditor` *(dev)* | Semantic moment detection over transcripts, point-in-time existence check, credit-record emission (never consumed). | P26 | openspec `memory-loop-audit`; ADR-0028 D1–D8 |

**Explicitly does NOT qualify:** `cli.Primitives` / `cli.NewDeps` / `cmd/engram` edge primitives (c3 K13,
ADR-0020) — an injection mechanism and composition root; the "DI everywhere / thin cmd" rule is a repo
engineering convention (keep in AGENTS.md), not a component contract.

---

## 3. SOURCE MAP

### 3a. docs/deprecated top-level & misc

| Source | Destination |
|---|---|
| `docs/deprecated/README.md` (doc index) | Neither. Superseded by `docs/AGENTS.md` structure. Its "design/ charter" rule (graduate & delete) is superseded by the lifecycle guide's `docs/changes/` flow. Delete. |
| `docs/deprecated/FEATURES.md` — pointer paragraph | Neither; obsolete pointer to openspec. Delete. |
| `FEATURES.md` — "Validated goals (mission rollup)" | Neither capability nor component; mission/goal status → `docs/ROADMAP.md` (a "mission status" section) or root README. Evidence anchors stay in `dev/eval/LEDGER.md`. |
| `docs/deprecated/images/vault-graph.png` | Neither; still referenced by root `README.md:16` as `docs/images/vault-graph.png` (**now a broken link**). Move back to a non-deprecated assets path (e.g. `docs/images/`) or update README link. |
| `docs/deprecated/agents/domain.md` | Neither; agent-workflow governance. Rewrite into `docs/AGENTS.md`/root `AGENTS.md` "Domain docs" pointer (root AGENTS.md still links `docs/agents/domain.md` — stale). Content (read ADR/C4/openspec first) is obsolete under the new framework. |
| `docs/deprecated/agents/issue-tracker.md` | Neither; tracker conventions for skills (gh CLI, wayfinder ops). Keep as governance — move to `docs/` governance file or a repo-level `.claude`/skill config; root AGENTS.md links `docs/agents/issue-tracker.md` (stale). |
| `docs/deprecated/agents/triage-labels.md` | Neither; governance (label vocabulary). Same destination as issue-tracker.md; root AGENTS.md link stale. |
| `docs/deprecated/design/2026-07-01-engram-recall-subprocess-design.md` | Neither — **REJECTED 2026-07-08** design. Status already captured in ROADMAP Provenance ("Payload-prune production build"). Delete (git history keeps it). |
| `docs/deprecated/design/parallel-memory-agent-requirements.md` | In-flight change: → `docs/changes/<issue-key>_parallel-memory-agent/<issue-key>a_product-spec.md` (future capability P27 `automatic-memory-agent`). Needs an issue key; its REC/CAP/CTL/IND/NFR IDs must be re-rooted to Luhmann IDs under that key. Open Decisions §6 stay with the change. |
| `docs/deprecated/research/2026-08-30-memory-taxonomy-{human,llm-agents,composition,critic,engram-map,synthesis}.md` (6 files) | Neither — landscape research. Conclusions already extracted into ADR-0027/ADR-0028 and ROADMAP "Memory north-star track". Options: delete (git keeps), or relocate to a non-spec `docs/research/` if Joe wants it retained. engram-map.md is a stale inventory (cites "35 specs", ADR-0001..0026). |
| `docs/deprecated/superpowers/plans/2026-08-26-733-recall-chunk-only-cluster-read-gate.md` | Neither — completed plan. Its requirement already graduated to openspec `recall-payload-cuts` ("Agent reads chunk content before judging cluster relevance…") → migrates into P1/P2. Delete. |
| `docs/GLOSSARY.md` (stays) | Stays. Needs reference updates: cites `docs/architecture/adr.md` (×2), ADR-0019/0021, `openspec/specs/`, `README.md`. Retired-term entries (Permanent, MOC, vocab nomination, vocab term-note, vocab-index, `Vocab:` line, ENGRAM_SERVER) are history — keep or prune per Joe. Term definitions that are normative (e.g. matched set, note floor, lazy-chunks zero-note exception, offer/outbox) must not duplicate spec MUSTs — point to the canonical spec. |
| `docs/ROADMAP.md` (stays) | Stays. Stale refs: `docs/architecture/adr.md` (×5), `openspec/changes/archive/` (×8), `openspec/changes/fix-eval-tooling-defects/…` (now archived), `openspec/specs/<x>/spec` (×4), `docs/design/2026-07-0x-*.md` and `docs/superpowers/plans/2026-0x-*.md` (already-deleted files), `superpowers/sdd/tasks/progress.md`. ~35 `dev/eval/LEDGER.md#…` anchors (out of scope, fine). |

### 3b. architecture/c1-system-context.md

| Section | Destination |
|---|---|
| Intro + L1 diagram | Neither (diagram; lifecycle guide forbids diagrams in canonical specs except Mermaid sequences in technical specs). Context sentences → P-spec context paragraphs. |
| Element catalog S1 operator | Context for all product specs (user = operator). |
| S2 Engram (incl. prune, count) | P19 (prune), P23 (count). |
| S3 harness (skills, registration sources) | P8 / C17 (source list); P22 / C15. |
| S4 vault (flat layout, paths, retired Permanent/MOCs) | C1 `vault-note-store`; P17 (vault path/creation). |
| S5 session stores (Claude/Pi JSONL; OpenCode removed) | C7. |
| S6 Go toolchain | C16. |
| S7 parent vault | P15, P16, C19, C20. |
| Relationships R1–R7 | R1 context; R2 C23/C25–30; R3 C1/C3; R4 C6/C7; R5 C16; R6 C15; R7 C19. |
| Key flows intro | Neither. |
| Flow: recall | P1 (skill procedure), P2/C9 (Channel 1/2 mechanics). Sequence diagram → C9 technical spec as Mermaid. |
| Flow: learn | P3, P4, P19 (ingest-first step); retired `learn episode`/`transcript --mark` → §4. |
| Flow: please | P11 / C28. |
| Flow: update | P22 / C15, C16. |
| L1 decision flowchart: engagement lifecycle | P9 (firing guidance) context; non-normative. |
| `/please` seven-step flowchart | P11. |
| Companion: who reviews at gates A–D | P11 (gate staffing requirements) + P12 (route for reviewer staffing). |
| Out of scope at L1 (embedder not external; route not L1) | C5 (embedder boundary rationale → ADR-equivalent; not canonical); route note → P12 context. |
| Related | Delete. |

### 3c. architecture/c2-containers.md

| Section | Destination |
|---|---|
| Intro + L2 diagram | Neither (diagram). |
| Container catalog C1 Skills | C25–C30, P8. |
| C2 engram CLI (ingest/learn/query/vocab/embed/update/check/show/prune/count) | Split: C6/C7 (ingest), C1/C2/C3/C11 (note write), C9/C10 (query), C11 (vocab), C4 (embed), C15/C16 (update), C13 (check), C14 (count). |
| C3 Embedded model (M4 defect) | C5, C4 (model-mismatch policy). |
| C4 Vault (layout, lock, vault-id, `.engram/`) | C1, C3, C21. G0 defect → C12. |
| Relationships table | Agent→C1: C25–30; C1→C2: C23 + P1 (show/show-chunk/amend/learn usage); C2→C3: C5; C2→C4: C3/C1/QA→P5; C2→S5: C7 (non-persistent prefixes, `.engram/sweep.json`); C2→S6: C15/C16; C2→S7: C19. |
| Flowchart: ingest/chunking | P19 / C6 / C7 / C8. |
| Cross-level note (L1↔L2) | Neither; delete. |
| The skills↔binary split | Architectural principle → root AGENTS.md "Design Principles" (already there) + C23/C25 context. Not a canonical requirement. |
| Key flows (L2) intro | Neither. |
| Flow: recall (L2) | P1 + C9 (sequence). |
| Flowchart: recall query pipeline | C9. |
| Flow: learn (L2) | P3/P4, C25–C27. |
| Flowchart: learn capture kinds | P3. |
| Flowchart: vocab lifecycle | P20 / C11. |
| Recall-time lazy-L2 synthesis | P1 (coverage decision is skill behavior; `engram synthesize` does not exist). |
| Flowchart: lazy-L2 coverage decision | P1. |

### 3d. architecture/c3-components.md

| Section | Destination |
|---|---|
| Intro ("separate processes" correction) | C23 context. |
| L3 diagram | Neither (diagram); K-IDs map below. |
| K1 transcript+context (ingest) | C7, C6. |
| K4 learn + qa | C1, C2, C3; P4, P5. |
| K4w amend/resituate/activate | P4; C1/C3. |
| K5 embed | C4, C5. |
| K5b cli/embed | P25 / C4. |
| K6 query | C9. |
| K7 vaultgraph | C12 (+ C1 scan). |
| K8 cluster | C10. |
| K9 update | C15, C16. |
| K10 luhmann | C2. |
| K11 debuglog | C24. |
| K12 prune | C6 / P19. |
| K13 cmd/engram primitives | Does not qualify (see §2); AGENTS.md convention + ADR-0020 content. |
| K14 serve + exchange kernel | C19, C20, C21, C22. |
| Recurring defect shape (G0 / M4) | C12 (link form), C4 (model policy); defect history → neither (git). |
| Dead/test-only surface | **Stale** (see §4); drop. |
| Data contracts | C9 (payload contract), C4 (sidecar contract), C6/C25 (ingest→skill→learn). |
| Flow: `engram query` internals | C9 (Mermaid sequence). |
| Flow: `engram learn` write internals | C1/C2/C3/C4/C11 interaction reqs, owned by the learn write path (P4). |
| Flow: `engram learn qa` capture path | P5 / C1. |

### 3e. architecture/memory-invariants.md

| Section | Destination |
|---|---|
| Header / test-method legend | Neither (history). |
| A. G0 link-form, G2 orphans, G3 dangling (G1/G4/G5 RETIRED) | C12 + C13 / P24. Retired → drop. |
| B. T1a/b/c, T2 | All RETIRED → drop. |
| C. E1 sidecar presence (E3/E4/E5 RETIRED) | C4 / C13. |
| D. P1 | RETIRED → drop. |
| E. Marker M1–M3 | Effectively retired with ADR-0009 (markers subsumed into ingest manifest) but **not marked retired** here; M3-like "one source can't starve another" may map to C6 byte budget — verify vs code. |
| F. C1 determinism, R1 recall-mirror (L3-1, R2 RETIRED) | C10 (determinism), C9 (recall-mirror property). |
| G. K1 vault write-lock | C3 / C2. |
| H. U1 update idempotence | C15/C16. |
| Acceptance RT-1 | RETIRED → drop. |
| Phase-1 additions M4 model homogeneity, M5 situation presence, M6 learn idempotency, M7 marker monotonicity, M8 ID uniqueness | M4 → C4/C9; M5 → C1; M6 → P3 (open); M7 → retired with markers; M8 → C2. |
| Dispositions / severity model / checker robustness (UTF-8) | C13 (severity + robustness). |
| Phase-3 S1 (no unmediated vault writes), S2 (single representation) | S1 → C25–C27 / P1 (skills write only via CLI); S2 → C1 (verify whether duplicated `situation` still exists). |
| What the checker must implement | C13 / P24. |

### 3f. architecture/memory-system-rigor.md

Whole file = process history (2026-06-04 recovery plan, phases 0–8, antagonist loops). **Neither.** Surviving
substance: "executable vault-invariant checker as a targ gate" → C13; "verify-don't-assert" and "per-phase
adversarial review" → already embodied in the please skill / AGENTS.md process. §1 property list duplicates
memory-invariants.md (same mapping). Delete after extraction.

### 3g. architecture/adr.md (every ADR)

ADRs are rationale; under the new framework, rationale lives in change artifacts → git/issue history, not
canonical specs. Recommended: map each ADR's *still-normative decisions* into the listed specs; keep the
"why" either nowhere (git) or in a retained decision log if Joe wants one (open question, §4).

| ADR | Status | Destination |
|---|---|---|
| Preamble / status legend | — | Neither. |
| 0001 Skills + slim binary split | Accepted | Principle → root AGENTS.md; C23/C25 context. |
| 0002 Pure-Go embedded model | Accepted (M4) | C5; P25. |
| 0003 Embed-on-write sidecars | Accepted | C4; P4. |
| 0004 Three tiers; blended retrieval | Accepted but **largely historical** (tiers removed 2026-06-20) | Drop (history). |
| 0005 L3 ADRs synthesized at learn time | Superseded | Drop; lazy synthesis lives in P1. |
| 0006 Embed source by kind (episodes) | Superseded | Drop; residual: situation vector defaults (C4). |
| 0007 Wikilink graph authored/walked by binary | Accepted (G0) | C12 (graph walk for recall is gone; backlinks/check remain). |
| 0008 Per-arc episodes | Superseded | Drop. |
| 0009 Marker forward-progress | Superseded | Drop. |
| 0010 Session reader/finder interfaces, composite backends | Superseded (partial; OpenCode removed) | C7 (JSONL reader only). |
| 0011 Controlled-vocab tag nomination | Superseded by 0025 | C11 write-time assignment survives; nomination drop. |
| 0012 D5′ asymmetric QA | Accepted | P5 / C9 (exclusion of q-notes). |
| 0013 Vault flock + atomic rename | Accepted | C3. |
| 0014 Memory-backed tier discount | Accepted | P12. |
| 0015 Skill decomposition stops at write seam (+ 2026-09-26 amendment) | Accepted | C27 write-memory, P4; amendment → P8. |
| 0016 Hand-authored mermaid diagrams | Accepted but **superseded by new framework** | Neither; governance (TECHNICAL-SPEC-GUIDE now rules diagrams). Drop. |
| 0017 Evidence-based route rubric | Accepted | P12. |
| 0018 Counting surface distinct from recall | Accepted | P23 / C14. |
| 0019 Tags sole categorical representation | Accepted | C1 (tags field), P23, P12 (route evidence tags), C11. |
| 0020 Enforced internal/ purity | Accepted | Repo engineering convention → root AGENTS.md (already present). Not a component. |
| 0021 Chunk dedup + record-subset eviction gate; update notifies | Accepted | C6 (D1–D8, D10), P22 (D9 notify). |
| 0022 Deployment as sync | Accepted | C15 (D1–D9); note OpenCode root in D1 is stale. |
| 0023 Update re-execs fresh binary | Accepted | C16. |
| 0024 Derivational vocab refit | Accepted | C11 / P20. |
| 0025 Centroid-proximity explore sampling | Accepted | C9 / P2. |
| 0026 Runbook fourth content kind | Accepted | P6, P7, C1. |
| 0027 Human-memory taxonomy as coverage checklist | Accepted | Neither — strategy/north star → ROADMAP "Memory north-star track" (already referenced there). |
| 0028 Engram as experiential learning layer; memory-loop audit; judged-only credit | Accepted | P26 / C32 (audit D1–D3, D8); D4 experience records not built → ROADMAP; D5–D7 governance → ROADMAP/AGENTS. |
| 0029 Local-first parent sync | Accepted | P15/P16/P17, C19–C22 (D1–D12); "rejected alternatives" → history only. |
| "Decisions deliberately NOT made into ADRs" | — | Neither; drop (B2 capture-phrasing convention is skill-authoring guidance → learn skill). |

### 3h. openspec/specs (all 53)

| Spec | → Capability | → Component(s) |
|---|---|---|
| cli-count | P23 vault-aggregation | C14, C12 |
| engram-command-template-safety | (P4/P9 cross-cutting MUST) | C18 |
| eval-phase2-fixture-scoring-fidelity | P26 | C31 |
| eval-trial-vault-leak-detection | P26 | C31 |
| eval-warm-config-fidelity | P26 | C31 |
| guidance-delegate | P10 | C18 |
| guidance-learn-moments | P9 (+P13) | C18 |
| guidance-recall-moments | P9 | C18 |
| guidance-runbook-follow-frame | P7 | C18, C9 (query-first) |
| ingest-auto-sweep | P19 | C7, C6 |
| ingest-chunk-dedup | P19 (+P22 backlog notice) | C6 |
| ingest-prune-detach | P19 | C6 |
| learn | P3 (+P13) | C26 |
| learn-adhoc-qa-capture | P3 / P5 | C26 |
| learn-branching-disposition | P21 / P3 | C26, C2 |
| learn-qa-capture | P5 | C1, C4, C11 |
| learn-runbook-capture | P6 | C1, C27 |
| lessons-contract | P13 | C18 |
| memory-loop-audit | P26 | C32 |
| please | P11 (+P13) | C28 |
| please-doc-enumeration-gate | P11 | C28 |
| production-guidance-activation | P9 / P7 / P10 | C15 (guidance deploy) |
| recall-centroid-sampling | P2 | C9, C11 |
| recall-glance-deep-dial | P1 | C25 |
| recall-matched-note-floor | P2 | C9 |
| recall-payload-cuts | P2 (+P1 read-gate req) | C9, C25 |
| recall-query-timings | P2 | C9 |
| recall-runbook-surfacing | P6 / P2 | C9 |
| recall-two-channel-payload | P2 | C9 |
| retrieval-probe-signal-fidelity | P26 | C31 |
| route | P12 (+P13) | C29 |
| route-dispatch-evidence | P12 (+P23 drift audit) | C29, C14 |
| route-evidence-rubric | P12 | C29 |
| runbook-lexical-triggers | P6 | C9, C1 |
| skill-runbook-registration | P8 | C17, C22 |
| update-deploy-sync | P22 | C15 |
| update-flat-vault-luhmann-notice | P21 / P22 | C2 |
| update-local-install-safety | P22 | C16 |
| update-reparent-luhmann-batch | P21 | C2, C10, C1 |
| update-self-reexec | P22 | C16 |
| vault-concurrency-safe-writes | P4 / P19 (guarantee) | C3 |
| vault-embed-on-write | P4 | C4, C5 |
| vault-local-first | P17 / P15 | C21 |
| vault-merged-recall | P15 | C19, C9 |
| vault-note-identity | P18 (+P15 exchange fields) | C1, C19 |
| vault-offer-curation | P14 | C22, C30, C20 |
| vault-parent-offers | P15 | C19 |
| vault-parent-pulldown | P15 | C19, C22 |
| vault-query-model-provenance | P2 | C9, C4 |
| vault-serve-api | P16 | C20 |
| vault-vocab-lifecycle | P20 | C11 |
| vault-wikilink-resolution | P4 / P24 | C12 |
| write-memory-worker | P4 (+P3, P11 lessons-audit req) | C27 |

### 3i. openspec/changes/archive (names only) — rationale history, delete or leave as git history

Each archive is change history; per the lifecycle guide, no completed-change archive is retained. Capability touched:
vocab-* (×3, fix-vocab-refit-dangling-links) → P20; update-deploy-as-sync, update-reexec-fresh-binary,
gate-local-mode-downgrade, update-flags-flat-vault-luhmann-reeval, reparent-luhmann-autonomous-pipeline → P22/P21;
recall-centroid-sampling, recall-chunk-only-cluster-read-gate → P2/P1; batch-small-issues-714-672-683-652 → mixed;
harden-route-skill-mechanics, route-escalate-go-package-implementation-priors, route-skill-to-runbook → P12;
remove-opencode-and-shelley-support → P22/C7 (removal); reword-surface-unmanaged-label → P22;
restore-luhmann-branching-disposition → P21; note-origin-identity → P18; serve-client-declared-identity, serve-vault-api → P16;
runbook-note-kind, runbook-lexical-triggers, runbook-trigger-whole-word, runbook-redflags-truncation-safety, runbook-retrieval-probe → P6/P26;
vault-merged-recall, local-first-parent-sync → P15/P17; eval-warm-config-fidelity, fix-eval-tooling-defects → P26;
memory-loop-audit → P26; runbook-shim-follow-frame, activate-shim-guidance, shim-question-stop-genuine-ambiguity → P7;
curate-skill-to-runbook, please-skill-to-runbook, recall-learn-writememory-to-runbook → (reversed — skills stay skills) P8;
register-skills-as-runbooks, register-skills-all-skill-folders → P8; learn-rate-skill-only, learn-at-subagent-return → P3/P9/P13;
fix-frontmatter-remainder, fix-show-amend-reparent-frontmatter → C1.

---

## 4. Notable things

### Superseded / removed features still described in legacy docs
- **Tiers L1/L2/L3, `--tier`, `nearest_l3`, `hubs`, subgraph/hub recall, episode kind, `engram learn episode`,
  `engram transcript --mark`, learnmarker** — removed 2026-06-19/20 (recall-v2 deep clean). Still pervasive in
  ADR-0004/0005/0006/0008/0009, memory-invariants (marked RETIRED), memory-system-rigor.
- **Marker invariants M1–M3, M7** (memory-invariants §E / Phase-1) are **not marked retired** even though ADR-0009
  says markers were subsumed into the ingest manifest. Likely dead — verify before dropping.
- **OpenCode** — ADR-0022 D1 still lists `~/.config/opencode/engram/`; archive `remove-opencode-and-shelley-support`
  removed it. ADR-0010's composite reader is gone.
- **`ENGRAM_SERVER` thin client** — removed (now a hard error; ADR-0029 D1, `vault-local-first`). Keep only the MUST NOT.
- **Tag nomination at query time** (ADR-0011) superseded by explore sampling (ADR-0025); `vault-vocab-lifecycle` spec
  Purpose still says "tag-based term nomination" — wording drift (write-time *assignment* is what survives). Its
  "Untagged rate trigger"/"Hub concentration trigger" scenarios actually assert they do *not* trigger (diagnostics only).
- **Vocab plan flow** removed (ADR-0024); vocab index/term-notes/`Vocab:` line retired (GLOSSARY).
- **Permanent/ and MOCs/ folders** retired 2026-06-12 (flat vault); GLOSSARY still defines them.
- **`engram recall` subprocess design** — REJECTED; do not migrate.
- **Skills→runbooks direction reversed** (archives `*-skill-to-runbook`, then ADR-0015 amendment / register-skills):
  skills are the shipping form, runbook notes mirror them. Any legacy text saying a skill "is now a runbook" is stale.
- **ADR-0016** (mermaid diagram governance) is superseded by the new doc framework.
- **c3 "Dead/test-only surface"** claims `BuildGraph`/`InDegreeIn` have zero production consumers — **stale**:
  `internal/cli/count.go:217` calls `vaultgraph.BuildGraph` for `--backlinks-of`. `BFSWithCap` may still be dead (verify).
- **c3 K1** says "marker advance" — stale per ADR-0009.
- **c1 "Flow: recall"** says recall phrases exactly 10 query strings; glance uses ~3; verify current skill text before promoting numbers.
- **memory-engram-map.md** research inventory is a 2026-08-30 snapshot (35 specs, ADR ≤0026) — stale.

### Overlaps
- **LESSONS contract** is spread over 5 specs (`lessons-contract`, `learn`, `please`, `route`, `guidance-learn-moments`).
  The `learn`/`please`/`route` specs are **delta-shaped** (their whole content is the LESSONS slice), not full capability
  specs for those skills — the bulk of recall/learn/please/route/curate behavior lives only in SKILL.md files and must be
  derived from them (or Joe decides the SKILL.md is the canonical spec — see ambiguities).
- **write-memory-worker** spec mixes worker mechanics (P4) with learn's four capture kinds (P3) and please Step 7 lessons audit (P11).
- **vault-note-identity** mixes provenance (P18) with parent-exchange fields xid/parent/aliases (P15).
- **vault-local-first** mixes vault identity (P17) with exchange (P15).
- **recall-payload-cuts** mixes a CLI flag contract (P2) with a skill behavior requirement (P1).
- **production-guidance-activation** vs **update-deploy-sync** (guidance opt-in): one is "which imports a harness config
  carries", the other is deployment mechanics — keep split P9/P7 vs P22.
- **ingest-chunk-dedup** includes an `engram update` notification requirement (P22).
- **route-dispatch-evidence** includes a `count` drift-audit requirement (P23).
- c1/c2/c3 restate the same flows at three levels plus the README CLI reference and GLOSSARY — heavy duplication;
  pick one owner (the technical spec) per contract.
- memory-invariants.md and memory-system-rigor.md §1 duplicate each other.

### Gaps (shipped behavior with no openspec spec — derive from code/skills)
`engram check` (P24), `engram embed apply/status` (P25), `engram show` / `show-chunk` / `query-chunks` (P2),
`engram amend` / `resituate` / `activate` (P4; only incidental reqs elsewhere), `engram vault-id` (P17),
`engram vocab bootstrap/propose/stats/tag-definitions` (partly in P20), the **curate** skill procedure (P14 judgment
steps), most of the **recall/learn/please/route** procedures, debug logging.

### Ambiguities for Joe
1. **Skills as components?** Are recall/learn/please/route/curate/write-memory procedures technical components
   (C25–C30, tested by headless evals), product capabilities only, or is each SKILL.md itself the canonical spec?
2. **Guidance granularity:** one capability per guidance doc (P7, P9, P10 as proposed) or a single `ambient-guidance`?
3. **Eval/audit tooling (P26, C31, C32):** product capability (Joe is a user) or dev tooling excluded from `docs/specs/`?
   ADR-0028 D5 calls the audit "product"; the eval harness is dev-only.
4. **Where does "why" go?** The new framework has no ADR home. Drop ADR rationale to git, or keep a decision log?
5. **Parent exchange granularity:** P15/P16/P17 split vs one `vault-federation` capability; C21 vault-identity
   standalone vs folded into C19.
6. **LESSONS contract (P13)** as its own capability vs requirements inside P3/P11/P12.
7. **Borderline components:** C8 text-chunker, C23 cli-command-router, C24 debug-log.
8. **Mission rollup** (FEATURES.md "Validated goals") — keep somewhere (ROADMAP / README) or drop?
9. **Stale links from live files:** root `README.md` (docs/images/vault-graph.png, docs/architecture/*, docs/README.md),
   root `AGENTS.md` (docs/agents/*, docs/README.md, docs/architecture/c1-system-context.md, openspec/ as "primary"),
   GLOSSARY, ROADMAP — all need repointing once destinations are decided.
10. **Spec count:** brief said 56 openspec specs; repo has 53.
