# Engram discovery: history and memory

Read-only discovery for the docs migration (#792). Sources:
- git log: 3,898 commits, 2026-01-28 to 2026-10-09 (raw dump: `discovery/raw/gitlog.txt`).
- GitHub issues: 791 total, 46 open (raw: `discovery/raw/issues.json`).
- Native memory: `/Users/joe/.claude/projects/-Users-joe-repos-personal-engram/memory/` (50 files).
- Vault: 9 `engram query` runs (raw: `discovery/raw/q_*.yaml`), plus `engram show` of the key notes.
- Cross-checked against `docs/ROADMAP.md` (Parked / Deferred / Provenance bands), `docs/GLOSSARY.md` (retired markers), `docs/deprecated/architecture/adr.md` (ADR status lines), and `engram --help` (the live command surface).

---

## 1. Timeline of capability evolution

### Era 0: "projctl" (2026-01-28 to 2026-02-27). All of it is DEAD.
- `0dd3549e` 01-28 "initial projctl implementation": a traceability/orchestration tool with parser, trace graph, task, territory, escalation, state, worktree, retro, and breakdown-producer pieces.
- `9327fe36` 02-27 renamed projctl to engram.
- None of this survives. **Do not document** trace, parser, territory, escalation, state machine, or worktree orchestration.

### Era 1: hook-based memory (2026-03 to 2026-04-14). DEAD.
- From 03-04: hooks, surface, signal, maintain, registry, tomlwriter (TOML memories), classify/extract/correct/retrieve/evaluate/instruct/policy/adapt, plus spec(L1..L5) layers.
- Hooks were added and removed several times. `b3612136` (03-06) removed SessionEnd. `1099a138` (04-02) removed all engram hooks. A second round of hooks (04-14/15) used UserPromptSubmit, Stop, and PostToolUse nudges.
- Hooks were finally deleted with the vault pivot (`655988e6`). Roadmap: recall-at-decision-moments shipped "via CLAUDE.md guidance (not hooks)" (`bca7a9fc` 06-29). The blanket "recall before every tool call" hook is listed as a dead end.

### Era 2: multi-agent chat/server (2026-04-03 to 2026-04-13). DEAD.
- engram-agent, `/engram-up` and `/engram-down`, engram-tmux-lead/engram-lead, `engram chat`, dispatch, streamjson, apiclient, server, and an MCP server push POC.
- `b355e496` 04-13 reverted the branch. `1f825eb8` and `e5e25c03` 04-14 removed the packages, commands, and orchestration skills.
- **Do not confuse** this "server" with today's `engram serve`, which arrived in 2026-08.

### Era 3: "engram v3" skills + external-sources recall (2026-04-14 to 2026-05-13). Mostly DEAD.
- `dbf517cc` 04-14 "engram v3 — memory skills redesign" introduced the `/prepare` and `/learn` skills.
- Recall "mode B" pipeline (04-15/16) used a Haiku-ranked externalsources phase over CLAUDE.md, rules, auto-memory, and skills.
- c4 tooling era (04-25 to 05-03): `dev/c4` builders and the c4 skill, about 150 commits and issues #585 to #606. It later moved to a separate repo and was decided against for engram (ADR-0016; see section 3).
- `655988e6` 05-14 "vault-only memory, OpenCode support, paired recall/learn skills" deleted hooks, externalsources, tomlwriter, and related packages.

### Era 4: the zettelkasten vault (2026-05-14 to now). This is the current product.

| Date | Change | Status now |
|---|---|---|
| 05-14/15 | Vault-only memory. Vault defaults to `$XDG_DATA_HOME/engram/vault` and is bootstrapped on first learn (`1e0de9cb`). | Current |
| 05-14 | OpenCode harness support | **Removed**: OpenCode transcript backend deleted 06-20 (`0a2bf999`); OpenCode/Shelley harness support removed 08-08 (`968124c1`) |
| 05-22 | `please` skill added (`a943d5af`) | Current (see the skill flip-flop below) |
| 05-22 | tokenresolver/Keychain removed (`22c0265b`); all MEMORY.md references removed (`2c57efad`) | Removed |
| 05-25/26 | `engram learn moc` dropped (`d34b24d5`); MOCs/ dropped from bootstrap; legacy `engram recall` subcommand dropped (`8b4b97a2`) | Removed |
| 05-25 | `engram learn episode` (L1 episodes) and `engram transcript --mark/--segments` | **Removed** 06-19 (#649, `9bf02704`); ADR-0006/0008/0009 superseded |
| 05-26 | `engram query` repeatable `--phrase` (`32f3aa0c`); `--project`/`--issue` metadata (#636) | Current |
| 05-31 | L3 ADR synthesis at learn time (`aa44b6b3`) | **Removed**: ADR-0005 superseded by lazy-L2 06-19 (`cf1ee74a` removed the L1/L3 surface) |
| 06-04 | `engram check` (G0 graph resolution), `migrate-links`, `resituate` (`4f63eb67`) | check/resituate current; migrate-links absent from current help (verify) |
| 06-06 | `engram show` (`97422266`) | Current |
| 06-12 | Flat vault layout: `Permanent/` and `MOCs/` retired (`ec7da404`) | Current (flat) |
| 06-12 | Skills rewritten around chunk memory, "ingest-first, no L1/L3" (`f620bfaf`) | Current |
| 06-14 | `route` skill: delegate-everything agent routing (`7f5a0322`) | Current |
| 06-17 | STM: `engram activate`, LastUsed sidecar field, recency decay on note ranking (`280b09f9`) | activate is current. **Recency decay: open #774 says it "has never been active". Verify before documenting.** |
| 06-19/20 | Recall v2: `--synthesize-l2` became the sole path and was then removed as a flag; subgraph, hubs, `--tier`, and `--synthesis` dropped (`c07af643`, `b57ecc67`). `engram amend` and `learn --chunk-source` added (`1cb48bc8`). Recency channel added. | amend current. Subgraph/tier/synthesize-l2 **removed** |
| 06-20 | `engram prune` (`12e63f91`); ingest skips non-persistent workspaces | Current |
| 06-23 | Graph-expanded retrieval **reverted**, "0 marginal value" (`1cd5cf8f`) | Dead |
| 06-24 | Recall host/haiku split **reverted** (`3a4dc72e`) | Dead |
| 06-27 | `--lazy-chunks`, `show-chunk`, `--recent-fill`; per-source ingest byte cap removed | Current |
| 06-27 | async-learn dropped from the roadmap (`537f3d35`) | Rejected |
| 06-28 | Matched-note floor (`33821e64`); route memory-tier discount (`2bf959f4`, ADR-0014) | Current |
| 06-29 | Recall glance/deep dial (#662, `bdb8b0dc`) | Current |
| 07-01 | Question-anchored crystallization **PARKED** (`d66aa066`). Recall-firing guidance deployed via `update --with-guidance` (`120bb080`). | Parked / current |
| 07-02/03 | vocab (controlled-vocabulary tags) slices 1-2, centroid assigner, refit trigger. `--relation`/Step 2.6 removed in favor of `--supersedes`. QA pairs: `engram learn qa`, qa-question/qa-answer kinds (ADR-0012). | vocab current (reshaped twice, see below). QA current but **under reconsideration** (#727, #765) |
| 07-04 | write-memory worker skill (`bac6dce0`, ADR-0015); capture guards G1/G2/G6 | Current |
| 07-05 | Docs restructure: 106 historical docs deleted (`d425a153`); ADR-0016 (mermaid, no c4 skill) | Superseded by the 10-09 framework |
| 07-07 | `delegate.md` guidance (`470e9a6e`); multi-file guidance import | Current |
| 07-08 | `engram count` (ADR-0018) | Current |
| 07-10/11 | vocab moved to tags (#678): definition fact notes; hub notes, `Vocab:` lines, and vocab-index retired; `migrate-tags` retired (#681) | Tags current; the listed legacy forms are **retired** |
| 07-12/13 | Clusters-first payload (#684, #689): built, then **reverted and refuted** (`0ae98779`, `701ec870`) | Dead |
| 07-14 | prune `--empty`; empty-index notice (#694) | Current |
| 07-19 to 22 | #700: `internal/` purity; `cmd/engram` primitives; `cli.NewDeps` (ADR-0020) | Current architecture |
| 07-24 | Pi harness in `update`; Pi session ingest (#696) | Current |
| 07-25/26 | Chunk-index dedup + `prune --duplicates` (ADR-0021) | Current (residual: #771) |
| 07-27 | OpenSpec initialized (`c9f6ae88`); 07-27 backfill of specs. `update` deploys as a sync (#706, ADR-0022). README "Upgrading" section removed (#713). | OpenSpec is now being superseded by #792 |
| 07-28 | `update` self re-exec (ADR-0023); derivational vocab refit (ADR-0024) | Current |
| 07-29 | Explore (centroid-proximity) sampling **replaces tag nomination** (ADR-0025) | Current. Tag nomination is retired |
| 08-08/09 | OpenCode/Shelley removed; `update --reparent-luhmann`; Luhmann branching restored in learn (#701) | Current |
| 08-22 | `engram serve` + `ENGRAM_SERVER` thin client + offer curation + `curate` skill (`90f28a6e`, `61ae1ff5`); Cloudflare dependency dropped (`196db20c`) | serve current. **`ENGRAM_SERVER` thin client removed 09-27** (`1e372553`: now a hard error) |
| 08-24 | `runbook` note kind, the fourth kind (ADR-0026, `eea84e03`) | Current |
| 08-25 | Merged local+parent recall (`2d03233c`) | Current (reworked 09-28) |
| 08-30 | ADR-0027 (human-memory taxonomy as coverage checklist), ADR-0028 (experiential learning layer, memory-loop audit) | Current north star |
| 09-18 | Runbook `red_flags`; shim follow-frame guidance (`8c7829b5`) | Current (#790: shim refused as prompt injection in bare CLAUDE.md) |
| 09-19 to 22 | **Skills converted to runbooks and deleted**: route, please, curate, write-memory (`87e9a006`, `b79f390e`, `2c4c2637`, `a81f7edc`). Runbook lexical `triggers` + `query --text` (`00efc1c6`). | **Reversed** 09-25 (see next row); triggers remain current |
| 09-25/26 | Skills restored (`21d6c603`). `register-skills` plus update registration offers. Source-qualified skill keys (`claude:route`). All default skill folders scanned. (Vault note 1054; ADR-0026 amendment) | Current |
| 09-27/28 | Local-first parent sync (ADR-0029): vault ID, outbox, offers both ways, pull-down via activate, merged query dedupe keeping local, `amend --into` fold, `show` parent fallback | Current |
| 09-30 to 10-01 | learn-at-subagent-return: the LESSONS line is judged at the return; please drops captured-marking | Current |
| 10-09 | New docs framework adopted; legacy docs moved to `docs/deprecated/` (`61eb98d4`); root CLAUDE.md renamed to AGENTS.md (`c0e63825`) | In progress (#792) |

### The skill/runbook flip-flop: a high risk of documenting a dead state
- 2026-09-13, vault note 1031: the goal was "every skill becomes a runbook".
- 2026-09-19 to 22: four skills were deleted.
- 2026-09-23, note 1054 (refutes 1031): deleting skills was a distribution regression. Runbooks in a personal vault do not deploy to other installs.
- **Current state:** all six SKILL.md files ship. Each skill can optionally have one registered vault runbook note (`skill-<name>`, `skill_hash`). Docs `68dfdf29` 09-25 reversed the "retired skill" language.
- Stale residue remains in places: `openspec/specs/write-memory-worker` and `please-doc-enumeration-gate` still describe please "sub-runbooks" (#768). GLOSSARY lines 125 and 998 carry "retired … restored" history.
- **Rule for the migration:** please, route, curate, and write-memory are current skills. Their runbook notes are mirrors, not the source.

### Parked, refuted, and dead directions (from ROADMAP Provenance, Deferred, and Parked bands). Do NOT document these as features.
- **Retrieval and ranking:**
  - graph-expanded retrieval and graph traversal (PPR, spreading activation)
  - tag nomination (superseded by ADR-0025)
  - clusters-first payload, Variants A and B (#684, #689)
  - artifacts/file-keyed retrieval angle
  - question-anchored crystallization (parked)
  - link prediction (TransE/RotatE)
  - #693 vector-read cache (deferred)
  - #637 `--field` (deferred)
- **Cost:**
  - payload-prune production build (rejected 07-08)
  - recall host/haiku split, and whole-op haiku
  - async-learn and the between-session consolidation pass (rejected)
- **Process:**
  - #686 old-text echo detector (out of purview, note 293)
  - #687 surprise harvest (refuted)
  - #671 parallel-builders (deferred)
  - blanket recall hook (dead end)
- **Note model (removed):**
  - episodes / L1
  - L3 ADR synthesis
  - MOCs and the Permanent/ folder
  - the Fleeting/ scan
  - `--relation` and Step 2.6
  - `Vocab:` lines, the vocab-index, and vocab hub/term notes
  - `migrate-tags`
  - `engram promote` (renamed to `learn`, #611/#617)
- **Deployment and mode (removed):** OpenCode/Shelley harnesses; `ENGRAM_SERVER` thin client (now a hard error); the README Upgrading section.

### The live command surface today (`engram --help`)
`query`, `ingest`, `prune`, `query-chunks`, `activate`, `count`, `check`, `vault-id`, `show`, `show-chunk`, `update`, `resituate`, `amend`, `register-skills`, `serve`, `learn`, `embed`, `vocab`.

---

## 2. Proposed CAPABILITIES and COMPONENTS, in Joe's vocabulary

Joe's preferred terms come from GLOSSARY, commit scopes, and note titles:
- **vault** (short for "agent-memory vault")
- **note**, with kinds **fact**, **feedback**, **runbook**, and qa-question/qa-answer
- **chunk** / **chunk index** (not "episode")
- **sidecar** (the `.vec.json` embedding file)
- **Luhmann ID**
- **wikilink**
- **supersedes**
- **vocab** / **tags** / **vocab definition note**
- **explore sampling**
- **matched-note floor**
- **recency channel**
- **glance / deep** (the recall modes; "rung" is already taken by these, so avoid it elsewhere)
- **offer** / **outbox** / **pull-down** / **parent** / **vault ID**
- **curation** (verdicts are covered / near / absent)
- **guidance file** (recall.md / delegate.md / learn.md / shim.md)
- **follow-frame**
- **triggers**
- **LESSONS contract**
- **dispatch evidence** (route)
- **harness** (Claude Code, Pi)
- **deploy as sync**

Avoid coined jargon (note 291) and the retired names: episode, MOC, Permanent, L1/L3, nomination, promote, ENGRAM_SERVER, synthesize-l2.

### Proposed product capabilities (`docs/specs/product/<capability>`) and the openspec specs that would fold into each

| Capability slug (proposed) | User-facing surface | Existing openspec specs to fold in |
|---|---|---|
| `recall` | recall skill; `engram query`, `show`, `show-chunk`, `activate`, `query-chunks` | recall-centroid-sampling, recall-glance-deep-dial, recall-matched-note-floor, recall-payload-cuts, recall-query-timings, recall-runbook-surfacing, recall-two-channel-payload, runbook-lexical-triggers, vault-merged-recall, vault-query-model-provenance |
| `learn` | learn skill; `engram learn` (fact/feedback/runbook/qa); `amend`, `resituate` | learn, learn-adhoc-qa-capture, learn-branching-disposition, learn-qa-capture, learn-runbook-capture, lessons-contract, memory-loop-audit, write-memory-worker (or a separate `write-memory` capability, since it is a shipped skill) |
| `transcript-ingest` (Joe says "ingest" / "chunk index") | `engram ingest --auto`, `prune` | ingest-auto-sweep, ingest-chunk-dedup, ingest-prune-detach |
| `vault-maintenance` (or split into vocab and integrity) | `check`, `vault-id`, `count`, `vocab`, `embed`, `update --reparent-luhmann` | cli-count, vault-concurrency-safe-writes, vault-embed-on-write, vault-note-identity, vault-vocab-lifecycle, vault-wikilink-resolution, update-flat-vault-luhmann-notice, update-reparent-luhmann-batch |
| `parent-sync` (Joe's terms: "local-first parent sync", "vault exchange") | `ENGRAM_PARENT`, `engram serve`, offers, outbox, pull-down, curate skill | vault-local-first, vault-parent-offers, vault-parent-pulldown, vault-serve-api, vault-offer-curation |
| `install-and-update` | `engram update` (deploy sync, self re-exec, guidance activation, local install safety) | update-deploy-sync, update-self-reexec, update-local-install-safety, production-guidance-activation, engram-command-template-safety |
| `skill-registration` | `engram register-skills`, update offers | skill-runbook-registration |
| `guidance` (ambient firing docs) | recall.md, delegate.md, learn.md, shim.md | guidance-recall-moments, guidance-delegate, guidance-learn-moments, guidance-runbook-follow-frame |
| `please` | please skill | please, please-doc-enumeration-gate |
| `route` | route skill | route, route-dispatch-evidence, route-evidence-rubric |
| `curate` | curate skill | (inside vault-offer-curation; could join parent-sync) |
| (dev only, not product) eval harness | `dev/eval` | eval-phase2-fixture-scoring-fidelity, eval-trial-vault-leak-detection, eval-warm-config-fidelity, retrieval-probe-signal-fidelity. Decide whether this is a technical component or out of scope. |

### Proposed technical components (`docs/specs/technical/<component>`)
Based on package layout plus commit scopes.
- **cli-composition-root**: `cmd/engram` + `internal/cli/primitives.go` + targets (ADR-0001, ADR-0020; the `targ check-thin-api` invariant).
- **query-engine**: unified cosine ranking, clusters, explore sampling, recency channel, merged parent query. Lives in `internal/cli` and `internal/cluster`.
- **embedder-and-sidecars**: `internal/embed` (MiniLM-L6 `go:embed`, `.vec.json`, model_id stamps; ADR-0002, ADR-0003).
- **chunk-index**: `internal/chunk` + ingest/prune manifest + dedup (ADR-0021).
- **transcript-reader**: `internal/transcript` + `internal/context` (JSONL; Claude and Pi sessions; ADR-0010).
- **vault-note-store**: note frontmatter, Luhmann IDs (`internal/luhmann`), flock + atomic rename (ADR-0013), rename/rewrite, vault ID.
- **vault-graph**: `internal/vaultgraph` (wikilinks, `check`; ADR-0007).
- **vocab**: tags, definition notes, derivational refit (ADR-0019, ADR-0024).
- **exchange** (outbox / offers / serve HTTP API; ADR-0029).
- **deployer**: `internal/update` (sync roots, symlinks, self re-exec, guidance imports; ADR-0022, ADR-0023).
- **skill-registry**: commit scope `skillreg` (scanners, source-qualified keys, declines in `skill-registrations.json`).
- **agent-instructions**: skills + guidance as deployable artifacts (ADR-0015 write seam).
- **debuglog**: minor.
- **eval harness** (`dev/eval`, Python; #710/#711 plan a port to Go): optional.

---

## 3. Standing decisions and conventions relevant to the docs

### Documentation and spec system
- **OpenSpec was the primary behavior-spec system.**
  - Vault note `913.2026-09-06.engram-designs-go-through-openspec-not-superpowers-spec-files`: designs go through opsx changes, never `docs/superpowers/specs`.
  - OpenSpec was initialized 07-27 (`c9f6ae88`) with a backfill of specs.
  - It is now being superseded or reconciled by the `docs/AGENTS.md` framework. #792 asks to "decide and document how OpenSpec relates to `docs/specs/`."
  - **Note 913 will need amending** once that decision is made (vault write; not done here).
- **ADR-0016** (`docs/deprecated/architecture/adr.md:390`) and vault note `171.2026-07-04.engram-c4-diagrams-...`: architecture diagrams are hand-authored mermaid verified against code. The user-level c4 skill is decided-out for engram (decided 07-05, docs-restructure DP7).
- Vault note `572.2026-07-27.joes-spec-framework-landscape`: Joe also uses traced, spec-kit, and OpenSpec across repos. Reconcile with the repo's decided conventions; do not assume greenfield.
- Vault note `622.2026-07-28.requirement-quality-bar-shared-across-joes-spec-frameworks`: one shared requirement bar applies.
  - testable; no should/may
  - measurable and technology-agnostic success criteria
  - scenarios enumerated, edge cases covered, scope bounded
  - no conflicts
  - traceable
- Vault note `645.2026-07-28.specifying-current-behavior-converts-bugs-into-change-proposals`: in a descriptive spec tree, writing the spec converts divergences into change proposals. Re-frame findings accordingly.
- Doc-scrub notes (all relevant to migration completeness):
  - `566` (a presence-search cannot find an absence; diff against a canonical set)
  - `186` (enumerate the doc-scrub surface by grep, including key tokens)
  - `62` (stale docs cause design misses after removals)
  - `616` (fixing a claim in code does not fix it in docs; grep the claim repo-wide)
  - `589` and `925` (verify provenance pointers and note-number citations by opening them)
  - `545` (write status language last)
  - native memory `feedback_doc_memory_scrub_is_part_of_architectural_removal.md`
- Vault note `60` and native memory `feedback_git_is_the_fallback_no_deprecated_hoarding.md`: lean aggressive on cleanup. Consolidate, then delete; git is the recovery net; no "restore window". This supports deleting `docs/deprecated/` after migration, as #792 already expects.
- Vault note `188` (use the target system's own vocabulary) and note `291` (no coined jargon; check new labels do not collide, e.g. M1-M8 and "rung" are already taken).
- Vault note `293`: engram's purview is memory. Do not spec bespoke dev-tooling as product.
- `docs/AGENTS.md`:
  - The file list is the docs index; there is no `docs/README.md` (Joe 10-09, in #792).
  - Issue numbers are the Luhmann ID roots.
  - GLOSSARY and ROADMAP stay top-level.

### Architecture, code, and testing conventions
These belong in the technical specs.
- **DI everywhere.** `internal/` does no direct I/O. `cmd/engram` is wiring-only (ADR-0001, ADR-0020, #700; lint plus `targ check-thin-api`). Test hard-to-test code by refactoring for DI, not by writing integration tests around the whole thing (root `AGENTS.md`).
- **Test stack: imptest + rapid + gomega** (native memory `feedback_engram_test_stack.md`). The package's sibling tests decide the style (vault `604`). Property tests are required for production code only, not harness code (vault `556`).
- **Test helpers:** pin HOME and XDG via the DI env seam, not `t.Setenv` (vault `1078`). Use `targ test` / `targ check-full`, never `go test` directly. #792 requires `//go:build integration` and `//go:build network` tags plus spec-ID-to-test traceability; check whether targ passes those tags.
- **Skill edits** require `superpowers:writing-skills` TDD (root `AGENTS.md`; native memory `feedback_skill_edits_writing_skills.md`).
- **Deploys are syncs** (vault `442`, ADR-0022). Opt-in deploy flags are one-time (vault `152`). Upgrade actions need an active notice, not just a doc (vault `272`, #713).
- **Standing roadmap constraint:** recall/learn skill changes are gated by the trap harness (`dev/eval/traps/gate.py`) and the `recall_cost` meter. The "win-nucleus" must not be touched (`docs/ROADMAP.md`, "Standing constraint").

### Process conventions
- **Issue briefing format** (native memory `feedback_issue_briefing_format.md`), five parts:
  1. problem, history, and why now
  2. systems, artifacts, and current state
  3. solution
  4. after-state
  5. how it solves the problem

  Parts 3 to 5 repeat per option. #768 is an example.
- **Issue tracking:** GitHub Issues via `gh`, never a local issues.md (native `reference_engram_issue_tracking.md`; commit `9724c6f5` removed `docs/issues.md`). Use "Fixes #N" to autoclose (native `feedback_use_fixes_keyword_for_autoclose.md`).
- **Merging:** ff-only merges, rebase on main, review before merge (root `AGENTS.md`).
- **Changes:** open a change whenever stated guarantees change (vault `695`).

### Stale claims in Joe's own memory and docs (do not propagate)
- Global CLAUDE.md "Memory System" tiers (decay 0.9x, confidence pruning) diverge from the code (vault `565`). Do not document them.
- Recency decay: commit `280b09f9` says it was added, but open #774 says it was never active. Verify in the code.
- Native memory `project_*` notes cite `--synthesize-l2` and candidate_l2s as the mechanism (06-20). Treat them as historical.
- `docs/GLOSSARY.md` still carries retired entries (MOC, Permanent, vocab nomination, ENGRAM_SERVER (removed)) with banners. Decide whether the migrated glossary keeps a "retired" section or drops it. Joe's "git is the fallback" stance suggests dropping it.
- Tracked root strays to triage:
  - `check_removals.py`: an eval-fixture script, 09-12.
  - `index.html`: the tailnet landing page, 08-25.

---

## 4. Open issues related to docs, specs, or traceability

| # | Title (short) | Relevance |
|---|---|---|
| **792** | docs: reconcile docs to new `docs/` guides + spec-to-test traceability | **The migration itself.** Lists the broken refs (root `AGENTS.md`, `README.md`, `agent-instructions/skills/{learn,route}/SKILL.md`), requires build tags and requirement-ID-to-test links, and asks to decide OpenSpec vs `docs/specs` |
| 768 | openspec main specs still describe please sub-runbooks | Stale spec content. Fix it during the migration instead of via an openspec change |
| 773 | guidance: reconcile recall/delegate/learn.md with shim.md | Duplicate cues; shapes the `guidance` capability spec |
| 685 | please/recall: plan-write doc-surface enumeration mechanism | Doc-staleness process |
| 790 | Shim guidance refused as prompt injection | Affects how the guidance and follow-frame capability is described |
| 791 | `engram check` G3: 10 dangling wikilinks in skill runbook notes | Vault integrity spec |
| 727, 765 | Collapse or reconsider the qa note kind | Note-kind model may change. Spec qa as current but flag it |
| 774 | Recency decay never active | Spec accuracy for recall ranking |
| 775, 777, 778, 779, 781 | Skill registration limits, polish, refactor, Pi/plugin parity, orphan prune | skill-registration spec gaps |
| 782, 785, 786 | merged query: double read, no parent-version check, self-parent guard | parent-sync spec gaps |
| 783, 784 | write-memory and curate skill guidance gaps | Skill specs |
| 771 | Chunk-index duplicates re-accumulate | transcript-ingest spec |
| 725 | Vault graph follow-ups (scope field, multi-hop edge, embedding consistency) | Parked; ROADMAP Parked backlog |
| 710, 711 | Port the eval harness to Go | Affects whether the eval harness becomes a technical component |
| 735 to 742, 751 | Runbook validation chain, memory routing across delegation, lexical channel probe | Experiments and roadmap. Do not spec as current behavior |
| 718, 747, 748, 675 | Efficacy / outcome-record tracking | Future work; roadmap only |

Recently closed doc-adjacent issues, for history: #647 (README drift), #649 (transcript/episode retirement scrub), #692/#713 (upgrade docs), #698 (reword the features doc), #761 (openspec skills to runbooks, not planned).
