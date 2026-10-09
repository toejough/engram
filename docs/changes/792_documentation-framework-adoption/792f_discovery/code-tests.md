# Engram Code and Test Discovery: Capabilities, Components, Test Map

Scope: read-only inventory of `cmd/engram`, `internal/**`, `agent-instructions/**`, `dev/targs.go`,
`dev/skillsmeta`, and the Go test files. Branch `docs/parallel-memory-agent-requirements`, 2026-10-09.
Criteria applied: `docs/TECHNICAL-SPEC-GUIDE.md` sections Component Qualification, Component and
Behavior Requirements, and Test-Suite Classification, plus the `docs/AGENTS.md` layout of one product
spec per capability and one technical spec per component.

**Test count correction.** The "~466 test files" figure counts a stale git worktree twice
(`.claude/worktrees/runbook-vs-skill/`, which is ignored but still on disk). The real repo has:

- **225** `*_test.go` files outside `dev/eval`, containing **2075** top-level `func Test*`. Subtests are
  not counted. 170 of the files are in `internal/cli`.
- **16** test files under `dev/eval`. They are all behind `//go:build targ` and run via `targ test-dev`.

The raw per-file data used below is in `../testcounts.txt` (count, path, R=rapid / I=mock marker,
build tag) and `../joined.txt` (component, count, path).

---

## 1. User-facing surface, grouped into proposed capabilities

### 1.0 CLI surface inventory (from `engram --help`, the subcommand help, and the struct tags in `internal/cli/*.go`)

Global environment variables: `ENGRAM_VAULT_PATH` (sets `--vault`), `ENGRAM_VAULT_NAME`,
`ENGRAM_CHUNKS_DIR`, `ENGRAM_PARENT` (parent-vault URL), `ENGRAM_SERVER` (**rejected**: hard error on
every command, telling you to use ENGRAM_PARENT), `ENGRAM_DEBUG_LOG`, `ENGRAM_MODEL_PATH`,
`ENGRAM_TRANSCRIPT_DIR`, `ENGRAM_UPDATE_REEXEC` (internal), `ENGRAM_CONTENT_BUDGET`,
`ENGRAM_RECENT_FILL`, `ENGRAM_LAZY_CHUNKS`. Defaults: `$XDG_DATA_HOME/engram/{vault,chunks}`.

| Command | Flags (summary) | Realization |
|---|---|---|
| `query` | `--phrase`(rep) `--text` `--vault` `--chunks-dir` `--limit`(20) `--project` `--content-budget` `--recent-fill` `--lazy-chunks` `--timings` | query.go, query_explore.go, query_triggers.go, recency.go, merged_query.go, offer.go |
| `query-chunks` | `--phrase`(rep) `--chunks-dir` `--limit` | query_chunks.go |
| `show <ref>` | `--vault` `--parent` | show.go, serve_client.go |
| `show-chunk <source#anchor>` | `--chunks-dir` | show_chunk.go |
| `activate` | `--note`(rep) `--vault` `--parent` `--vault-name` | activate.go, pulldown.go |
| `count` | `--group-by` `--filter`(rep) `--backlinks-of` `--vault` | count.go + vaultgraph |
| `check` | `--vault` | check.go |
| `vault-id` | `--regenerate` \| `--claim` `--vault` | vaultid.go, exchangestate.go |
| `learn fact` | common + `--situation` `--subject` `--predicate` `--object` | targets.go, learn.go |
| `learn feedback` | common + `--situation` `--behavior` `--impact` `--action` | same |
| `learn runbook` | common + `--situation` `--done-when` `--body` `--red-flag`(rep) `--trigger`(rep) | same |
| `learn qa` | `--slug` `--question` `--answer`\|`--answer-file` `--contributors`(rep) `--certainty` `--source` `--vault` | qa.go |
| (learn common) | `--slug` `--vault` `--vault-name` `--target` `--position top\|continuation\|sibling` `--source`(req) `--supersedes` `--project` `--issue` `--tier` `--chunk-source`(rep) `--tag`(rep) | targets.go CommonLearnArgs |
| `amend` | `--target`(req) `--supersedes` `--chunk-source` `--situation` `--subject/--predicate/--object` `--behavior/--impact/--action` `--done-when` `--body` `--red-flag` `--trigger` `--activate` `--clear-pending` `--discard` `--into` `--expect-hash` `--vault` `--vault-name` `--chunks-dir`; plus undocumented `--pending` `--repo` `--user` | amend.go, amend_fold.go |
| `resituate` | `--note`(req) `--situation`(req) `--vault` | resituate.go |
| `ingest` | `--transcript`(rep) `--markdown`(rep) `--pi-sessions`(rep) `--sweep`(rep) `--auto` `--chunks-dir` `--vault` | ingest.go, ingest_dedup.go, sweepspec.go |
| `prune` | `--empty` `--duplicates` `--dry-run` `--chunks-dir` | prune.go, prune_duplicates.go |
| `embed apply` | `--all` \| `--missing` \| `--stale`, `--force`, `--dry-run`, `--vault` | embed.go |
| `embed status` | `--vault` | embed.go |
| `vocab bootstrap` | `--seed`(req) `--floor` `--vault` | vocab_commands.go |
| `vocab propose` | `--term` `--description` `--vault` | vocab_commands.go |
| `vocab refit` | `--dry-run` `--names` `--vault` | vocab_derive.go, vocab_apply.go |
| `vocab stats` / `vocab tag-definitions` | `--vault` | vocab_commands.go |
| `update` | `--dry-run` `--with-guidance` `--allow-downgrade` `--regen-vocab` `--reparent-luhmann` `--answers` `--backfill-identity` | cli/update.go, internal/update |
| `register-skills` | `--accept` `--decline` `--adopt key=ref`(rep) `--skills-dir`(rep, preview) `--dry-run` `--vault` `--vault-name` | skillreg_run.go, skillsources*.go |
| `serve` | `--addr`(req, no 0.0.0.0 default) `--vault` `--vault-name` `--chunks-dir`. Routes: query/show(+raw)/activate/learn | serve.go, serve_learn.go, serve_exchange_read.go, cmd/engram/serve.go |

Help-output oddity: the root help puts `learn`, `embed` and `vocab` under "Source: (unknown)". Asking
for help on a nested subcommand (`engram learn fact --help`) prints only the `--completion` flag, not
that subcommand's own flags.

### 1.1 Agent-instruction surface

| Artifact | Purpose |
|---|---|
| `skills/recall/SKILL.md` (367 lines, tests/ has 5 baselines) | Recall procedure: phrases → one `engram query` → judge clusters → inline writes |
| `skills/learn/SKILL.md` (313, tests/ has 7 baselines) | Capture lessons (facts/feedback/runbooks), the correction-moment fast path |
| `skills/write-memory/SKILL.md` (142) | Worker that composes and runs the `engram learn/amend` command handed off by recall or learn |
| `skills/curate/SKILL.md` (136) | Review pending offers (served learns, pulled-down parent notes) via `amend --clear-pending/--discard/--into` |
| `skills/please/SKILL.md` (158) | End-to-end task orchestration workflow with gates and a LESSONS audit |
| `skills/route/SKILL.md` (288) + `price-table.md`, tests/ has a microtest JSON | Subagent type, model and effort routing |
| `guidance/recall.md`, `learn.md`, `delegate.md` | Ambient firing guidance, deployed by `update --with-guidance` and pulled in via `@import` |
| `guidance/shim.md` (132) | Query-first plus runbook follow-frame shim |

### 1.2 Proposed capabilities (one product spec each)

| Capability | Description | Surface |
|---|---|---|
| `memory-recall` | An agent retrieves relevant vault notes and transcript evidence for a situation in one ranked, budgeted payload. Runbooks whose triggers match the request surface first, and evidence can be fetched lazily. | `query`, `query-chunks`, `show`, `show-chunk`; recall skill; recall.md, shim.md (query step) |
| `memory-capture` | An agent records a typed lesson (fact, feedback, runbook, Q&A) as a Luhmann-addressed, embedded, provenance-stamped vault note. | `learn fact/feedback/runbook/qa`; learn and write-memory skills; learn.md |
| `memory-revision` | An agent corrects, narrows, supersedes, re-situates or marks used an existing note, keeping the note, its sidecar and its links consistent. | `amend` (content, supersedes, activate), `resituate`, `activate` (local) |
| `session-history-indexing` | Turn transcripts (Claude JSONL, Pi sessions) and markdown into a searchable chunk index without any LLM, with incremental sweep, dedup and prune. | `ingest`, `prune` |
| `vault-health` | Inspect and enforce vault invariants and statistics, including embedding coverage. | `check`, `count`, `embed status/apply` |
| `vocabulary-curation` | Keep a controlled tag vocabulary: seed it, propose terms, refit from clusters, report stats, migrate old formats. | `vocab *`, `update --regen-vocab` |
| `vault-structure-migration` | One-shot structural maintenance: batch Luhmann re-parenting and provenance backfill. | `update --reparent-luhmann/--answers`, `update --backfill-identity` |
| `vault-sharing` (parent/child federation) | Expose a vault over HTTP and let a child vault merge parent results into recall, pull parent notes down, and offer local notes upward. | `serve`, `ENGRAM_PARENT`, `query`/`show --parent`/`activate --parent`, outbox drain, `vault-id` |
| `offer-curation` | Review offers awaiting a decision (served learns, pulled-down notes): accept, discard, or fold. | curate skill; `amend --clear-pending/--discard/--into/--expect-hash`; `pending_offers` flag in query output |
| `skill-runbook-registration` | Mirror installed skills, commands and prompt templates from every harness source into vault runbook notes, with remembered accept/decline/adopt decisions. | `register-skills`; offers inside `update` |
| `engram-installation` | Install or refresh the binary and sync skills and guidance into each detected harness, with downgrade protection and self re-exec. | `update` (`--dry-run`, `--with-guidance`, `--allow-downgrade`) |
| `agent-task-orchestration` | Agent workflow guidance (not Go code): carry a task end to end, route subagents, delegate, follow matched runbooks. | please and route skills; delegate.md; shim.md follow-frame |

Notes: `vault-identity` (`vault-id`) could stand as its own capability. I have folded it into
`vault-sharing` because its only consumer is exchange. `vault-health` and `vault-structure-migration`
could merge if fewer specs are wanted.

---

## 2. Proposed components

Ownership legend: **R** = repository component. **B** = repository-owned dependency-boundary
interaction (wraps an external, OS or library contract).

### 2.1 Components that meet all five qualification criteria

| # | Component | Own. | Responsibility (contract) | Current realization | Capabilities | External boundary |
|---|---|---|---|---|---|---|
| 1 | `command-line-surface` | R | Parse argv and env into typed command requests. Resolve vault, chunks-dir and vault-name precedence. Refuse `ENGRAM_SERVER`. Create a missing vault on first use. Map results to exit codes and streams. | cli/targets.go, cli.go, chunksdir resolution, `Targets()`; `github.com/toejough/targ` | all CLI | process argv/env/stdio (targ library) |
| 2 | `durable-file-store` | B | Read, write, rename and walk files. Writes are atomic (temp file, then rename), exclusive-create is supported, permissions do not depend on umask, and errors stay `errors.Is`-compatible. | cli/edgefs.go (`primFS`, `EdgeFS`), vault_fs.go, primitives.go `FSPrims`, cmd/engram/main.go | all writers | OS filesystem |
| 3 | `vault-write-lock` | B | Exclusive advisory lock that serializes concurrent vault and chunk-index writers (see the K1 "concurrent learn never collides" invariant). | cli/locker.go (`primLocker`, `FileLocker`), `LockPrims` | capture, revision, ingest | OS flock syscalls |
| 4 | `subprocess-runner` | B | Run external commands (git, go install), collect output, translate "not found", and spawn with inherited stdio to report the child's exit code. | cli/commander.go, spawner.go, `ExecPrims`; update.Commander/Spawner interfaces | installation, provenance | OS processes |
| 5 | `interrupt-handling` | B | Turn repeated SIGINT/SIGTERM into a forced exit, while the first signal still allows a graceful finish. | cli/signal.go (`ForceExitOnRepeatedSignal`, `ForwardAsPulses`) | all long commands | OS signals |
| 6 | `text-embedder` | B | Map text to a fixed-dimension vector with a declared model ID: the bundled MiniLM-L6 384-dim model, extracted once to a cache, loaded lazily, with over-length input handled. | internal/embed hugot.go, runtime.go, cache.go, cachefs.go, assets/model; cmd/engram/hugot.go | capture, recall, ingest, vocab | hugot/GoMLX library, ONNX model asset, cache dir |
| 7 | `embedding-sidecar-store` | R | Per-note `.vec.json` sidecar: the body/situation hash, the content-hash, stale/missing/model-mismatch state classification, embed-on-write, and bulk re-embed selection. | internal/embed sidecar.go, hash.go, state.go, cosine.go, embedder.go `BuildSidecar`; cli/embed.go, auto-embed in learn | capture, revision, vault-health | (uses durable-file-store, text-embedder) |
| 8 | `note-document-format` | R | Typed note frontmatter and body: render and parse fact/feedback/runbook/qa notes; edit YAML nodes so unknown keys survive; read CRLF safely. | cli/frontmatter_node.go, exchangefields.go, learn.go render functions, embed/hash.go `SplitFrontmatter` | capture, revision, sharing, registration | YAML library (go.yaml.in/yaml/v3) |
| 9 | `luhmann-addressing` | R | Parse, compare and sort Luhmann IDs. Allocate the next top, continuation or sibling ID. Derive the ID from a basename. | internal/luhmann; cli/luhmann.go | capture, migration | — |
| 10 | `note-reparenting` | R | Batch re-parent top-level notes: plan the renames, rewrite every wikilink reference, rebuild sidecars, and apply answers from an agent-filled JSON. | cli/luhmann_reparent.go, luhmann_reparent_apply.go | vault-structure-migration | — |
| 11 | `wikilink-graph` | R | Scan vault notes, parse wikilinks (fence-aware), build the link graph, report unresolved targets and in-degree. Resolve note refs (basename, `[[link]]`, `.md`, bare ID). | internal/vaultgraph; cli/note_ref.go | recall, vault-health, revision | — |
| 12 | `vault-invariant-checker` | R | Run read-only invariant checks (graph, situation presence, sidecar schema) and report PASS/FAIL, exiting non-zero on any FAIL. | cli/check.go | vault-health | — |
| 13 | `note-authoring` | R | Write a new typed note: validate the arguments, assign an ID, stamp provenance, record supersession, write atomically under the lock, embed it, and queue an offer. The QA variant writes a linked pair. | cli/learn.go, qa.go, targets.go learn args, deps_compose learn deps | memory-capture | — |
| 14 | `note-amendment` | R | Edit an existing note in place (field replace, supersedes, chunk-source merge, activate) and resituate it (frontmatter, body and sidecar stay in sync). Maintain the supersession inverse. | cli/amend.go, resituate.go, supersedes.go | memory-revision | — |
| 15 | `provenance-identity` | B | Detect and stamp `repo:`/`user:`/`vault:` provenance from git (origin URL, toplevel, user). Keep the first writer's user. Backfill notes that lack it. | cli/identity.go, identity_backfill.go | capture, revision, migration | git via subprocess-runner |
| 16 | `semantic-retrieval-engine` | R | Rank notes and chunks for a set of phrases: per-phrase cosine scoring, merging chunk and vault space, clustering with nominated representatives and candidate L2s, a matched-note floor, `--project` filter, excluding pending offers, supersession ride-along. | cli/query.go, query_chunks.go, query_vault_meta.go, supersedes ride-along, offer.go exclusion | memory-recall | — |
| 17 | `similarity-clustering` | R | K-means over cosine distance with silhouette-based automatic K. Pure and deterministic. | internal/cluster | recall, vocab refit | — |
| 18 | `runbook-trigger-matching` | R | Find runbooks whose literal triggers appear in the user's `--text` and put them first, with `red_flags` capped. | cli/query_triggers.go | memory-recall, follow-frame | — |
| 19 | `recall-payload-renderer` | R | Shape the YAML payload: item limit, content budget with snippets, lazy chunks, red-flags truncation (keeps the tail, names `engram show`), `pending_offers` hint, timings block. | cli/query.go render functions, redflags_truncation.go, query cap functions | memory-recall | stdout YAML |
| 20 | `recency-ranking` | R | Time-decay multiplier on note and chunk relevance, plus the recency channel of the newest ingested chunks (eval-tuned half-life). | cli/recency.go | memory-recall | (wall clock injected) |
| 21 | `vocabulary-taxonomy` | R | Controlled vocabulary of `vocab/<term>` tags: assign by cosine floor, seed and propose terms, derive and refit terms from clustering (match/retire/name), growth-triggered refit, centroids file, explore-half sampling of term members at query time, migrate old formats. | cli/vocab*.go (vocab.go, vocab_apply, vocab_derive, vocab_centroids, vocab_trigger, vocab_regen, vocab_commands), query_explore.go | vocabulary-curation, memory-recall | — |
| 22 | `transcript-reader` | R | Read Claude Code JSONL (and Pi) transcripts and strip noise (tool results, base64, oversize content) into chronological text within a byte budget, with source-line indices. | internal/transcript, internal/context (strip, stripconfig, toolsummary) | session-history-indexing | transcript file format |
| 23 | `source-chunking` | R | Split stripped transcripts and markdown into embedding-sized chunks (merge turns to a target size, split oversized ones, preamble handling). Pure. | internal/chunk/chunk.go | session-history-indexing | — |
| 24 | `chunk-index-store` | R | Persist chunk records (JSONL per source plus a manifest with file hashes and ingested-at). Incremental, cheap-skip ingest. Content-hash dedup with a canonical member. Detach-prune of vanished sources. Look up a chunk by `source#anchor`. | internal/chunk/index.go; cli/ingest.go, ingest_dedup.go, prune.go, prune_duplicates.go, show_chunk.go | session-history-indexing, recall | — |
| 25 | `ingestion-source-discovery` | R | Resolve which sources to ingest: the declarative sweep spec (repo markdown, ancestor `.claude` dirs, session logs; `.engram/sweep.json` override), Pi session dirs, the default Claude projects dir. | cli/sweepspec.go, ingest source gathering | session-history-indexing | — |
| 26 | `vault-identity-registry` | R | The vault's identity and location record (`.engram/home.json`), the parent cache (`parent.json`), declined pulls. Stamp on first use, check location, regenerate or claim. | cli/exchangestate.go, vaultid.go | vault-sharing | — |
| 27 | `exchange-versioning` | R | Exchange hash over the offered fields (three-way compare), exchange frontmatter blocks (aliases, parent links), and applying offer receipts and folds to frontmatter. | cli/exchangehash.go, exchange_blocks.go, offer_receipt.go, exchangefields.go | vault-sharing, offer-curation | — |
| 28 | `offer-outbox` | R | Decide which local writes are offered upward. Queue them by xid. Drain to the parent with receipts, attention states and requeue-on-change. Build the payload with an identity check. | cli/offer_classify.go, offer_payload.go, outbox.go, offer_wiring.go | vault-sharing | (parent via parent-vault-client) |
| 29 | `parent-vault-client` | B | Talk to `ENGRAM_PARENT`: fetch query/show/raw, size-limit responses, back off after failures, merge parent and local payloads (dedupe by hash and link, note floor), pull notes down as pending local copies, re-check them. | cli/serve_client.go, merged_query.go, merged_dedupe.go, pulldown.go | vault-sharing, recall | HTTP client (net/http in cmd/engram/serve.go) |
| 30 | `vault-http-server` | B | Serve query/show/activate/learn over HTTP on an explicit bind address. A served learn becomes a pending offer that returns a receipt. Raw show returns a JSON envelope; query adds dedupe keys. | cli/serve.go, serve_learn.go, serve_exchange_read.go; cmd/engram/serve.go | vault-sharing | net/http server |
| 31 | `offer-curation-bookkeeping` | R | Judged-version check (`--expect-hash`), clearing the pending marker, discard, and fold into an existing note (aliases and parent links). | cli/amend_fold.go, offer.go pending marker functions | offer-curation | — |
| 32 | `skill-source-discovery` | B | Enumerate skill, command and prompt-template sources across Claude user/project/plugins/claude.ai-synced and Pi user/agents/config/packages/projects. Handle symlinks and trust. Assign source-qualified skill keys and slugs. | cli/skillsources*.go (5 files), skillkeys.go | skill-runbook-registration | harness config files (`~/.claude`, `~/.pi`, plugin manifests) |
| 33 | `skill-registration-ledger` | R | Compare sources against existing runbook notes (register, refresh, remove offers). Record decisions by key, pattern or scope in `skill-registrations.json`. Accept, decline, adopt. Prompt only when stdin is a terminal. | cli/skillreg*.go, skilloffers.go, skillanswers.go | skill-runbook-registration | terminal detection |
| 34 | `installer` | B | `go install` the binary (with downgrade guard and resolution of the binary path), self re-exec, detect harnesses, sync skills and guidance into harness dirs (symlinks, dangling-link cleanup, compat symlinks), migrations, the update report. | internal/update; cli/update.go | engram-installation | go toolchain, harness dirs, module cache |

### 2.2 Borderline candidates: fold into another component, or flag

| Candidate | Recommendation | Reason |
|---|---|---|
| `vault-bootstrap` (vault_init.go: Permanent/, `.obsidian/app.json`, starter README) | Fold into `command-line-surface` (first-use) or `vault-identity-registry` | Small, but it does own an idempotent contract (8 tests) |
| `vault-statistics` (count.go) | Fold into `wikilink-graph` (backlinks) plus a thin `count` caller, or keep it | It is a frontmatter group-by/filter plus in-degree; 20 tests |
| note lookup (show.go) | Caller of `wikilink-graph` (ref resolution) and `parent-vault-client`; not a component | Routing only |
| `recall-usage-tracking` (activate.go: bump LastUsed in sidecar) | Part of `embedding-sidecar-store` (LastUsed lives in the sidecar) or `note-amendment` | Thin |
| `debug-log` (internal/debuglog) | **Fails Necessity**: no production code emits a log line (see section 4) | — |
| `composition-root` (primitives.go, deps.go, deps_compose.go, cmd/engram/main.go) | Not a component: it is the DI seam and wiring. Its `targ check-thin-api` thin-main rule is a code-structure rule | Guide: "mockable seam MUST NOT qualify solely…" |
| Agent skills and guidance (`agent-skill-bundle`) | Product-level realization of capabilities. The only Go test is the description-length check in `dev/skillsmeta`. Behavior tests are markdown baselines plus external evals | No Go-verifiable contract beyond metadata |
| `recency-ranking`, `recall-payload-renderer`, `runbook-trigger-matching` | Each has independent pure tests, so each qualifies, but they could merge into `semantic-retrieval-engine` if fewer specs are wanted | Judgment call |

---

## 3. Test map

### 3.1 Conventions found

- **Assertions:** gomega in 212 of 225 files (dot-import `. "github.com/onsi/gomega"` with `g := NewWithT(t)`). No testify.
- **Property tests:** `pgregory.net/rapid` in **68** files (names often end in `Property` or `_property_test.go`; the `invariants_*_property_test.go` files encode named invariants C1, K1, R1, plus U1 which is not a property test).
- **imptest:** the import is in only **4** files. There are 4 generated mocks: `internal/cli/generated_MockCommander_test.go`, `generated_MockSkillSourceFS_test.go`, `internal/transcript/generated_MockReader_test.go`, `internal/vaultgraph/generated_MockVaultFS_test.go`. Most fakes are hand-written: about 10 embedder fakes (`fakeIngestEmbedder`, `fixedVectorEmbedder`, `countingEmbedder`, ...), `skillsources_fakefs_test.go`, and in-memory FS maps.
- **Black-box style:** most files are `package cli_test` and reach internals through `export_test.go` (`Export*` wrappers, about 100). `*_internal_test.go` files are white-box.
- **Naming:** names follow `Test<FuncOrType>_<Behavior>` (e.g. `TestRunAmend_…` ×57, `TestTargets_…` ×55, `TestRun…` ×50 in update, `TestDrainOutbox_…` ×33). Only **8** tests use the old spec-trace form `TestT<N>_` (`TestT10_MissingBundledModel_ClearError`, `TestT13_ParityWithPythonReference`, `TestT138..T143_Strip*`), yet `dev/targs.go` still sets `TARG_BASELINE_PATTERN='TestT[0-9]+_'`. File names often carry design-decision or spec tags (`offer_s16_test.go`, `merged_query_w9_dispatch_test.go`, `invariants_k1_…`), and comments cite design IDs (D5, D7, M12, H7, ...).
- **t.Parallel:** used everywhere except helper and generated files and `internal/cli/merged_query_dispatch_test.go`, which uses `t.Setenv`. That file is the only real test file without it.
- **Build tags:** there is **no** `//go:build integration` and **no** `//go:build network` anywhere. Existing tags:
  - `//go:build parity` on `internal/embed/parity_test.go` (`TestT13_ParityWithPythonReference`, compares against a Python reference).
  - `//go:build targ` on all of `dev/eval` and `dev/targs.go`.
  - The only `testing.Short()` skip is `cmd/engram/hugot_test.go` `TestBundledEmbedder_Smoke`.
  - Comments mention `targ test-integration`, but no tag separates such tests.

### 3.2 Per-component test files: `file(count of top-level Test funcs)` and dominant name prefixes

Paths without a directory prefix are in `internal/cli/`.

| Component | Files / tests | Files (count) | Dominant name patterns |
|---|---|---|---|
| command-line-surface | 4 / 25 | targets_test(18), cli_test(4), removed_server_test(2), chunksdir_test(1) | TestTargets_*, TestEngramServerSet_*, TestEngramLearn_*_EndToEnd |
| durable-file-store, vault-write-lock, subprocess-runner, interrupt-handling (currently mixed as "os-capability-adapters") | 13 / 79 | edgefs_test(15), primitives_integration_test(14), deps_compose_test(10), primitives_test(6), deps_compose_internal_test(6), vault_fs_test(6), commander_test(5), commander_integration_test(4), locker_test(3), spawner_test(3), signal_test(2), signal_integration_test(1), cmd/engram/main_test(4) | TestEdgeFS_*, TestRealEdgeFS_*, TestRealFlockLocker_*, TestCommanderIntegration_*, TestSpawnPrimitives_*, TestForceExit_*, TestNewDeps_* |
| text-embedder | 12 / 46 | embed/cache_test(9), cachefs_test(7), runtime_test(7), embedder_fake_test(7), buildembedder_test(5), cachefs_integration_test(3), hugot_test(2), overlength_test(2), embedder_test(1), parity_test(1, tag parity), cmd/engram/hugot_test(2) | TestExtractToCache_*, TestCacheFS_*, TestRuntimeBackend_*, TestBuildEmbedder_*, TestHugotEmbedder_*, TestLazyEmbedder_* |
| embedding-sidecar-store | 12 / 73 | embed/hash_test(25), state_test(9), sidecar_test(7), cosine_test(5), notexist_test(5), hash_property_test(4), build_test(2), crlf_test(1); cli auto_embed_test(5), embed_test(5), os_adapters_test(3), embed_selection_test(2) | TestContentHash_*, TestComputeState_*, TestExtractBody_*, TestAutoEmbedNote_* |
| note-document-format | 8 / 52 | exchangefields_survival_test(16), skillfields_survival_test(15), crlf_readers_internal_test(9), amend_unknown_keys_test(4), frontmatter_anchor_test(3), crlf_node_edits_test(2), frontmatter_node_internal_test(2), frontmatter_node_guard_internal_test(1) | TestExchangeFieldsSurvive_*, TestSkillIdentityFieldsSurvive_* (rapid) |
| luhmann-addressing | 2 / 26 | luhmann_test(16), internal/luhmann/luhmann_test(10) | TestNextLuhmannID_*, TestParseID_* |
| note-reparenting | 6 / 50 | luhmann_reparent_apply_test(18), luhmann_reparent_test(14), luhmann_reparent_crlf_test(9), _apply_more(5), _frontmatter(2), _internal(2) | TestRunReparentLuhmann_*, TestRenameAndRewriteReferences_* |
| wikilink-graph | 9 / 47 | vaultgraph/parser_test(18), bfs_test(9), graph_test(7), scanner_test(6), resolution_test(3), indegree_in_test(2), fence_property_test(1), scanner_root_test(1) | TestParseWikilinks_*, TestBFSWithCap_* (production code never calls BFSWithCap), TestBuildGraph_* |
| vault-invariant-checker | 1 / 10 | check_test(10) | TestRunCheck_* |
| vault-statistics (borderline) | 1 / 20 | count_test(20) | TestRunCount_* |
| note-authoring | 5 / 105 | learn_test(69), qa_test(23), learn_adapters_test(11), adapters_test(1), invariants_k1_property_test(1) | TestRunLearn_*, TestRenderRunbookFrontmatter_*, TestRunLearnQA_*, TestInvariant_K1_* |
| note-amendment | 8 / 70 | amend_test(32), supersedes_test(13), resituate_test(12), amend_crlf_test(5), resituate_preserve_test(3), amend_list_entries_test(2), resituate_internal_test(2), amend_targets_test(1) | TestRunAmend_*, TestRunResituate_*, TestParseSupersedesFlag_* |
| provenance-identity | 7 / 32 | identity_test(9), identity_backfill_node_test(7), amend_identity_test(5), identity_gaps_test(5), first_write_user_test(3), identity_internal_test(2), identity_backfill_parity_test(1) | TestBackfillIdentity_*, TestIdentityStamp_* |
| semantic-retrieval-engine | 15 / 90 | query_test(36), query_vault_meta_test(14), query_synthesize_l2_helpers_test(10), query_helpers_test(7), query_chunks_test(5), query_cluster_phase1/2_test(4+4), query_unified_test(3), query_note_floor_test(1), query_property_test(1), query_cluster_chunk_bound_property_test(1), cluster_property_test(1), invariants_c1(1), invariants_r1(1), query_integration_test(1) | TestQuery*, TestRunQuery_*, TestCluster*, TestInvariant_C1/R1 |
| similarity-clustering | 4 / 20 | cluster/silhouette_test(7), kmeans_test(5), autok_test(4), distance_test(4) | TestKMeans_*, TestAutoK_*, TestSilhouette_* |
| runbook-trigger-matching | 3 / 13 | query_runbook_test(10), query_trigger_property_test(2), query_triggers_test(1) | TestRunQuery_*Runbook*, TestMatchTriggers_* |
| recall-payload-renderer | 3 / 21 | query_cap_test(8), query_limit_test(7), redflags_truncation_internal_test(6) | TestCapChunkContent_*, TestCapRedFlagsForPreview_* |
| recency-ranking | 2 / 25 | recency_test(24), recency_eval_test(1) | TestParseCreatedFromNote_*, TestRecencyEvalDiscriminatingHalfLife |
| vocabulary-taxonomy | 8 / 183 | vocab_commands_test(72), vocab_test(26), vocab_apply_test(24), query_explore_test(16), vocab_derive_test(16), vocab_trigger_test(16), vocab_regen_test(7), query_explore_integration_test(6) | TestWriteVocabAssignment_*, TestRunVocabBootstrap_*, TestCheckAndPersistVocabRefitTrigger_*, TestDeriveVocabClusters_* |
| transcript-reader | 5 / 53 | transcript/transcript_test(19), context/stripconfig_test(18), context/context_test(16) | TestStrip*, TestJSONLReader_*, TestToolSummaryMode_*, TestT138..143 |
| source-chunking | 1 / 8 | chunk/chunk_test(8) | TestChunkTranscript*, TestChunkMarkdown* |
| chunk-index-store | 10 / 78 | ingest_test(15), prune_duplicates_test(14), ingest_dedup_test(12), ingest_family_deps_test(9), ingest_integration_test(8), prune_test(8), ingest_dedup_internal_test(5), show_chunk_test(3), chunk/ingestedat_test(3), prune_integration_test(1) | TestRunIngest*, TestRunPrune_*, TestSelectCanonical*, TestShowChunk* |
| ingestion-source-discovery | 4 / 29 | sweepspec_test(10), ingest_sweep_test(8), pi_sessions_test(8), ingest_auto_test(3) | TestResolveSweepRoots*, TestSweep* |
| vault-identity-registry | 2 / 36 | exchangestate_test(22), vault_first_use_test(14) | TestEnsureVault_*, TestCheckVaultLocation_*, TestStampVaultID_*, TestVaultIDUncommitted* |
| vault-bootstrap (borderline) | 1 / 8 | vault_init_test(8) | TestInitializeVault_*, TestResolveVault_* |
| exchange-versioning | 9 / 37 | exchangehash_test(11), exchange_crlf_test(5), fold_receipt_node_test(5), offer_receipt_test(5), exchangefields_test(4), exchangehash_internal_test(4), amend_hash_parity(1), exchange_blocks_internal(1), fold_receipt_parity(1) | TestExchangeHash_*, TestApplyReceiptToContent_* |
| offer-outbox | 7 / 91 | outbox_test(37), offer_wiring_test(17), offer_payload_test(11), outbox_attention_test(10), offer_s16_test(6), offer_classify_test(5), offer_errors_test(5) | TestDrainOutbox_*, TestBuildOfferPayload_*, TestClassifyOffer_* |
| parent-vault-client | 8 / 121 | pulldown_test(30), serve_client_test(27), merged_query_dedupe_internal_test(20), merged_query_internal_test(15), merged_query_w9_dispatch_test(11), parent_backoff_test(10), merged_query_dispatch_test(4), pulldown_recheck_test(4) | TestMergeQueryPayloads_*, TestActivate_*(parent), TestShowParent_*, TestFetchQueryPayload_* |
| vault-http-server | 6 / 44 | serve_learn_test(17), serve_test(16), serve_exchange_read_test(6), learn_wire_test(2), cmd/engram/serve_integration_test(2), cmd/engram/serve_config_test(1) | TestServeLearn_*, TestServeShow_*, TestServeQuery_*, TestServeAndFetch_*RoundTrip |
| offer-curation-bookkeeping | 2 / 19 | amend_fold_test(11), offer_test(8) | TestRunAmend_*(clear-pending/discard/into), TestExcludePendingOffers_* |
| recall-usage-tracking (borderline) | 1 / 10 | activate_test(10) | TestRunActivate* |
| note lookup (caller) | 1 / 9 | show_test(9) | TestRunShow_* |
| skill-source-discovery | 9 / 154 | skillsources_test(31), skillsources_piconfig_test(30), skillsources_pi_test(29), skillsources_project_test(15), skillsources_plugin_test(13), skillsources_resolve_test(11), skillkeys_test(11), skillkeys_qualified_test(10), register_skills_resolve_test(4) | TestScanPiConfiguredSources_*, TestScanClaudePlugins_*, TestResolveSkillSources_* |
| skill-registration-ledger | 14 / 150 | skillreg_run_test(28), skillreg_accept_test(28), skillanswers_test(18), skilloffers_test(16), skillreg_test(13), skillreg_notesource_test(12), register_skills_cli_test(9), skillreg_declines_test(6), skilloffers_namespace_test(5), update_skillreg_internal_test(5), skillnoteid_test(4), skillreg_unknown_keys_test(3), skillreg_accept_guard_test(2), skillreg_adoptflags_property_test(1) | TestRunSkillRegistration_*, TestCompareSkillOffers_*, TestAdoptSkillNote_*, TestAnswerSkillOffers_*, TestRegisterSkillsCLI_* |
| installer | 13 / 234 | update/symlink_test(56), update/update_test(37), update/migration_test(22), update/sync_test(22), update/runner_test(21), update/reexec_test(8), update/downgrade_test(5), update/resolve_binary_path_test(1); cli update_test(44), update_deps_test(10), reexec_test(4), invariants_u1_test(4) | TestRun_*, TestWriteUpdateReport_*, TestUpdater_*, TestCleanupDanglingLinksInDir_*, TestMaterializeSymlink_* |
| debug-log (fails Necessity) | 1 / 5 | debuglog/debuglog_test(5) | TestLog_*, TestTimed_* |
| agent-skill-bundle | 1 / 1 | dev/skillsmeta/metadata_test(1) (reads real `agent-instructions/skills/*/SKILL.md`) | TestSkillDescriptionsFitMetadataLimit |
| (helpers, 0 tests) | 7 | export_test, testbinary_test (TestMain), testdeps_internal_test, testhelpers_test, skillsources_fakefs_test, 2 generated mocks | — |

Caveat: the test-to-component assignment is by file name and leading test names. Many `internal/cli`
tests drive the `Run*` entry points and so also exercise callers (e.g. `TestTargets_*` spreads across
offer, vault-identity and CLI). The spec migration will need per-test review against "direct contract
test" ownership.

### 3.3 Impure tests that need classification (all currently run in the default suite)

**Subprocess or built binary (`integration`):**
- `internal/cli/cli_test.go`: builds the binary via `go build` (`sharedEngramBinary`) and execs `learn fact/feedback`, `--help`, and `update --dry-run` with a git shim on PATH.
- `query_integration_test.go`: builds the binary, then runs a real query. The binary uses the **real bundled embedder**.
- `register_skills_cli_test.go`: builds the binary and execs it.
- `testbinary_test.go`: TestMain plus the build helper.
- `register_skills_resolve_test.go`: runs `git`.
- `vault_first_use_test.go` `TestVaultIDUncommitted_RealGitIgnoredAndCommitted`: runs real git via the commander.
- `primitives_integration_test.go`, `commander_integration_test.go`, `testdeps_internal_test.go`: real exec.
- `cmd/engram/main_test.go`: `sh -c "exit 7"`.

**Signals:** `signal_integration_test.go` (`TestForceExit_RealSignalDeliveryThroughForwardAsPulses`).

**Real HTTP:** `cmd/engram/serve_integration_test.go` uses `httptest.NewServer`, a local loopback round trip. This is `integration`, not `network`. The `internal/cli/serve*_test.go` files inject fake `ListenAndServe`/`Fetch`, so they are pure apart from tempdirs. No test calls a live remote service, so **no `network` candidates exist**.

**Real model and runtime (local impure plus heavy):**
- `cmd/engram/hugot_test.go`: bundled model smoke test, skipped under `-short`.
- `internal/embed/hugot_test.go`, `cache_test.go`, `cachefs_integration_test.go`: real OS cache extraction.
- `buildembedder_test.go`, `overlength_test.go`, `embedder_fake_test.go`: reference the real Hugot constructors.
- `parity_test.go`: already behind the `parity` tag.

**Real filesystem through `t.TempDir()` (63 files in internal/cli plus embed and cmd).** Many use it only as a scratch vault and could move to in-memory FS fakes, which would make them pure. Files that use real OS deps on purpose:
- Through `ExportNewTestOsDeps` (production `NewDeps` over the real os): activate_test, amend_test, learn_test, resituate_test, vocab_commands_test, vocab_trigger_test.
- Explicit `RealEdgeFS` and flock: primitives_integration_test, locker_test, edgefs_test, vault_fs_test, os_adapters_test, prune_integration_test, ingest_integration_test, query_explore_integration_test.
- Heavy tempdir use (count in parentheses): serve_test(29), targets_test(25), vault_first_use_test(25), query_test(24), vocab_commands_test(18), exchangestate_test(17), serve_client_test(16), merged_query_w9_dispatch_test(15), register_skills_cli_test(15), primitives_integration_test(14), cli_test(12), learn_adapters_test(11), deps_compose_test(10), show_test(9), merged_query_dispatch_test(8), learn_test(7). Also the property tests invariants_c1/k1/r1, query_property, cluster_property and query_cluster_chunk_bound_property each use a tempdir.
- `dev/skillsmeta/metadata_test.go` reads repo files.

**Process environment (`t.Setenv`):** targets_test, merged_query_dispatch_test, serve_client_test, offer_wiring_test, vault_first_use_test, amend_fold_test.

**Wall clock (`time.Now()` or `time.Sleep`):** activate_test, serve_test, targets_test (time.Now); ingest_test (Sleep).

**Pure examples (in-memory or fakes):** internal/cluster, internal/chunk, internal/luhmann, internal/vaultgraph (MockVaultFS), internal/context, internal/transcript, most of internal/update (mock Filesystem/Commander/Env), skill-source tests (skillsources_fakefs), offer/outbox/exchangehash/merged_query internals.

---

## 4. Dead, unused, or surprising

1. **Duplicate worktree inflates counts.** `.claude/worktrees/runbook-vs-skill/` (ignored) holds a full copy of the source and tests, so `find . -name '*_test.go'` gives 466 instead of 225 (+16 dev/eval).
2. **`internal/debuglog` emits nothing.** `targets.go` builds a Logger from `ENGRAM_DEBUG_LOG` and places it in ctx, but no production code calls `debuglog.Log`, `debuglog.Timed`, or `LoggerFromContext`. The sink composition (debugsink.go) and its tests exist anyway. This fails the Necessity criterion.
3. **Dead exported functions:**
   - `vaultgraph.BFSWithCap`: only tests call it.
   - The vaultgraph package doc says the public entry point is `StartingPoints`, which does not exist.
   - `context.Strip` and `context.StripWithConfig`: only tests call them; production uses `StripWithConfigIndexed`.
   - `cluster.KMeans` is reached only through `AutoK`, which is fine.
4. **Stale test name:** `query_synthesize_l2_helpers_test.go` names a removed `--synthesize-l2` feature. It actually tests `mergeClusterReps`, `provenanceRankFor` and `resolvedItemLess`.
5. **Spec-trace naming vestige:** `TARG_BASELINE_PATTERN=TestT[0-9]+_` in dev/targs.go, but only 8 tests match. The new scenario-ID prefix scheme will replace this, so the env default needs updating.
6. **No integration/network tags,** although the guide requires them. About 70 files would need `//go:build integration` or a refactor to in-memory fakes. `go build` inside tests (cli_test, query_integration_test, register_skills_cli_test) is the heaviest of these, and the comments name a `targ test-integration` target that has no tag behind it.
7. **One giant package:** `internal/cli` has 82 source files (33.5k LOC; query.go alone is 2110 lines) and 170 test files. It holds about 25 of the proposed components, so package boundaries do not track component boundaries (expected under the new guide, but the realization column will mostly cite files).
8. **Undocumented flags:** `amend --pending/--repo/--user` have no help text (they look like served-learn wire fields that leak as CLI flags). Help for nested `learn`/`embed`/`vocab` subcommands lists no flags, and those groups appear under "Source: (unknown)".
9. **`ENGRAM_SERVER` exists only to be rejected** (`removed_server_test.go`). It is a deprecated-compat sniff; per Joe's "git is the fallback" rule it may be removable.
10. **Tracked root clutter:** `check_removals.py` (an eval-fixture mapping script) and `index.html` are committed at the repo root. `engram` (binary), `coverage.out` and `tmpvault/` are present but ignored.
11. **`dev/skillsmeta` is test-only.** It has no source file, just one test that reads the skill markdown.
12. **dev/eval (103 MB, `targ` tag):**
    - A Go eval harness: run.go, arms.go (nothing / skills-only / current-state arms), scenarios.go, score.go, aggregate.go, transcript.go, adapters.go (OS vault cloner, config builder, agent runner, JSONL results), 4 imptest mocks, 13 tests.
    - Python and shell drivers: run-layer-*.sh, run-layer-loop.py, run-accumulation-chain.py, isolation.py, and test_*.py files.
    - Experiment dirs: atoms, atoms-build, audit, cumulative (runbook_vs_skill phase2 fixtures), guards, learn-at-return, links, qa, recency-centroid-652 (imports internal/cluster and internal/embed), traps, vocab, plus LEDGER.md and a stray `__pycache__`.
    - It is invoked via `targ eval <arm> [trials]`, which needs `engram` on PATH and copies `~/.local/share/engram/vault`.
13. **Skill "tests" are markdown baselines** (recall: 5, learn: 7, route: microtest JSON) for writing-skills TDD, run by agents. Nothing executes them in the Go suite.
14. **`recency_eval_test.go`** is an in-suite eval (`TestRecencyEvalDiscriminatingHalfLife`) whose result is pinned in `defaultRecencyParams`. It is more an experiment than a contract test.
15. **Alignment with openspec:** the existing `openspec/specs/` holds 52 capability-ish specs (e.g. vault-merged-recall, vault-offer-curation, ingest-prune-detach, recall-matched-note-floor, eval-*). These map many-to-one onto the 12 proposed capabilities. The `eval-*`, `memory-loop-audit`, `retrieval-probe-signal-fidelity` and `lessons-contract` specs cover dev/eval or skill process, not shipped CLI behavior.
