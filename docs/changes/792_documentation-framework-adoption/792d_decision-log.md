# Documentation Framework Adoption - Decision Log

These are choices made during the #792 migration, recorded for Joe to validate afterwards. Questions
that need Joe's input are in [`792e_open-questions.md`](./792e_open-questions.md). Nodes append new
decisions as `792dN`, continuing the sequence.

## 792d1 Change directory and artifact set

- **Choice:** `docs/changes/792_documentation-framework-adoption/` holds `792a_product-spec.md`,
  `792b_technical-spec.md`, `792c_implementation-plan.md`, `792d_decision-log.md` and
  `792e_open-questions.md`.
- **Rationale:** #792 is the tracking issue, and `docs/AGENTS.md` roots change IDs at the issue key.
  Joe asked for one log of choices to validate and a separate document of questions needing his
  input.
- **Alternatives:** one change per capability. Rejected because Joe asked for a single change with a
  plan that tracks the whole migration.

## 792d2 OpenSpec is legacy documentation in scope

- **Choice:** `openspec/specs/**` and `openspec/changes/archive/**` are migrated like
  `docs/deprecated/`. Each spec file is deleted once its content is ported. Capability nodes mine the
  archive for rationale, and node 792c8c deletes it, because the lifecycle guide forbids keeping a
  completed-change archive. Removing the OpenSpec tooling is deferred to 792e3. Vault note 913
  ("designs go through openspec") is superseded; the orchestrator amends it, since vault writes are
  outside the repository.
- **Rationale:** the new framework replaces OpenSpec as the behavior record. #792 asks for the
  relationship between the two to be decided, and Joe's direction on 2026-10-09 was to adopt the
  framework.
- **Alternatives:** keep OpenSpec alongside the new specs. Rejected because two normative records
  would duplicate each other.

## 792d3 Skills and guidance are product behavior

- **Choice:** the procedures in `agent-instructions/skills/*` and `agent-instructions/guidance/*`
  are described in product specs. The SKILL.md and guidance files stay where they are as shipped
  product source; they are not documentation to delete.
- **Rationale:** what a skill does is observable by its user, Joe or an agent. The files themselves
  are what `engram update` deploys.
- **Alternatives:** treat each SKILL.md as its own canonical spec. Rejected because it would bypass
  the framework's ID and UAT rules.

## 792d4 Skill procedures are not technical components

- **Choice:** recall, learn, write-memory, please, route and curate get product capabilities but no
  technical component sections.
- **Rationale:** they meet the responsibility, ownership, stability and necessity criteria, but not
  independent verification. The technical guide requires a direct, deterministic contract test for
  every canonical technical requirement. Skill behavior is checked only by markdown baselines and
  headless evals, which aren't contract tests. The technical constraints they share live in
  `agent-instruction-bundle` (792d5).
- **Alternatives:** one component per skill, as in the legacy report's C25–C30. Rejected because
  every requirement would lack a contract test. A single procedure component was rejected for the
  same reason.

## 792d5 `agent-instruction-bundle` qualifies as a component

- **Choice:** one component owns the constraints on shipped instruction artifacts: harness metadata
  limits, command-template safety (`--flag="value"`), and the rule that skills write to the vault only
  through the CLI.
- **Rationale:** `TestSkillDescriptionsFitMetadataLimit` tests this contract directly, and a harness
  only loads a skill whose metadata fits. `engram-command-template-safety` and invariant S1 need an
  owner.
- **Alternatives:** make command-template safety a product requirement repeated in every skill
  capability. Rejected because it would duplicate one rule across capabilities.

## 792d6 Removed features are not documented; enduring absences become MUST NOT

- **Choice:** removed and parked features stay out of the specs. These include episodes, tiers,
  L3 synthesis, MOCs, the `Permanent/` folder, tag nomination, graph-expanded retrieval, the
  clusters-first payload, the OpenCode and Shelley harnesses, the recall subprocess, and hooks.
  `ENGRAM_SERVER`, which the legacy docs explicitly reject, becomes a **MUST NOT** in parent-sync.
- **Rationale:** the lifecycle guide keeps only current behavior in canonical specs, and keeps
  absences only when they are enduring contracts. ROADMAP Provenance already records the dead ends.
- **Alternatives:** keep a "retired" section in each spec. Rejected because git is the fallback.

## 792d7 Rationale lives beside the requirement it justifies

- **Choice:** ADR, design and archive reasoning becomes a short `Rationale:` paragraph under the
  requirement or orientation it justifies.
- **Rationale:** Joe asked for this, and it keeps each decision next to the rule it explains.
- **Alternatives:** a separate decision document. Rejected because Joe asked for rationale in the
  relevant sections. The tension with the lifecycle guide's "concise context only" rule for canonical
  specs is left open as 792e4, and doesn't block anything.

## 792d8 Per-component technical structure

- **Choice:** each component section has orientation, sequences, behavior requirements with Gherkin,
  and product-scenario mapping (792c3b). The option-analysis, impact and alternatives sections are
  omitted, and 792b1 states why once.
- **Rationale:** this change documents an existing system, so there are no options to compare.
- **Alternatives:** the guide's full eight-part structure. Rejected because its comparison sections
  would be empty.

## 792d9 Every operation is Added

- **Choice:** every capability and component section is an **Added** delta.
- **Rationale:** no canonical specs exist yet, so there is nothing to modify or remove.
- **Alternatives:** none apply.

## 792d10 Test renaming

- **Choice:**
  - Existing tests are renamed `Test<ScenarioID>_<PreviousDescriptiveName>`, for example
    `Test792a3b1a_RunLearn_WritesNote`. Python tests use `test_<ScenarioID>_<previous_name>`.
  - A test that verifies several scenarios takes the first scenario's ID and lists the others as
    `split deferred`.
  - References to each old name are grepped and updated before the rename.
  - Scenarios with no test are listed as coverage gaps; tests that verify no scenario keep their
    names and are listed.
  - No new tests are written.
- **Rationale:** the ID is all lowercase letters and digits, so it needs no character substitution.
  The underscore keeps the ID visually separate from the old descriptive name, which already uses
  underscores.
- **Alternatives:**
  - The guide's example form with no separator (`TestCAP_1434b1a1SelectPriorMonth`). Not used:
    the orchestrator chose the underscore form, and this deviation from the guide's example needs
    Joe's validation.
  - Splitting tests that cover several scenarios. Deferred, because splitting rewrites tests rather
    than renaming them.

## 792d11 Statuses

- **Choice:** work entries and rename entries use the non-test vocabulary (`Not started` →
  `In progress` → `Done`). Nodes use Blocked by, Ready, In progress, and Complete. The guide's Merged
  status is omitted.
- **Rationale:** no tests are written, so the test-driven status ladder doesn't apply. There are no
  per-node branches, so Merged would add nothing beyond Complete.
- **Alternatives:** the test-status ladder for renames. Rejected because a rename has no red or green
  phase.

## 792d12 Research, eval-process notes and plans are non-normative

- **Choice:** the durable knowledge in `docs/deprecated/research/*`, the superpowers plan, and the
  eval-process notes goes into rationale paragraphs, the GLOSSARY or the ROADMAP. The rest is deleted.
- **Rationale:** these documents never stated behavior. Their conclusions already appear in
  ADR-0027 and ADR-0028 and in the ROADMAP north-star track.
- **Alternatives:** keep a `docs/research/` directory. Rejected because `docs/AGENTS.md` defines no
  such directory.

## 792d13 Agent governance moves to root `AGENTS.md`

- **Choice:** `docs/deprecated/agents/*` (issue tracker, triage labels, domain docs) is merged into
  the root `AGENTS.md` sections that already point to those files.
- **Rationale:** these are workflow conventions for agents, not product or technical specifications.
- **Alternatives:** separate governance files under `docs/`. Rejected because `docs/AGENTS.md` lists
  a fixed set of governance files.

## 792d14 The in-flight design becomes its own change

- **Choice:** `parallel-memory-agent-requirements.md` describes a feature that isn't built yet. It
  moves to its own `docs/changes/<issue-key>_parallel-memory-agent/` directory. `gh issue list` found
  no matching issue, so the key is open question 792e2. No issue was filed.
- **Rationale:** the lifecycle guide puts unbuilt behavior in an active change, never in canonical
  specs.
- **Alternatives:** root it under #792. Rejected because the feature is unrelated to this migration.

## 792d15 Capability granularity: 22 capabilities

- **Choice:**
  - Capabilities are grouped by what a user does or gets: one per shipped skill, and one per
    distinct CLI goal.
  - In plan order, the 22 are: vault-identity, note-writing, note-revision, chunk-indexing,
    controlled-vocabulary, memory-query, vault-counting, vault-health, vault-migration,
    install-and-update, skill-runbook-registration, write-memory, recall, learn, lessons-contract,
    route, please, ambient-guidance, vault-serving, parent-sync, curate, and
    memory-effectiveness-evaluation.
- **Rationale:** the three discovery reports proposed 27, 12 and about 11 capabilities.
  - The 27-capability list split single surfaces apart, for example note provenance and QA from
    note writing.
  - The 12- and 11-capability lists merged unrelated goals, for example `check`, `count` and `embed`
    into one "vault-health", and please and route into one "orchestration".
  - Grouping by user goal keeps each file stable when the commands inside it change.
- **Alternatives:** the 27-, 12- and 11-capability inventories in `792f_discovery/`.

## 792d16 Capability names

- **Choice:** names use Joe's vocabulary from the GLOSSARY and commit scopes. Each skill capability
  is named after its skill. `memory-query` is the CLI retrieval surface and `recall` is the skill.
  `chunk-indexing` is used over `transcript-ingest`, because ingest also covers markdown.
  `parent-sync` follows "local-first parent sync". `install-and-update` is used over `engram-update`.
- **Rationale:** avoid coined jargon (vault note 291) and the retired names (episode, MOC, nomination,
  promote, `ENGRAM_SERVER`).
- **Alternatives:** `memory-recall` for the query surface. Rejected because it collides with the
  recall skill. Also `session-ingestion` and `vault-sharing`.

## 792d17 Runbooks are spread across their surfaces

- **Choice:** there is no `runbook-memory` capability. Runbook capture is in note-writing, trigger
  surfacing is in memory-query, and the follow-frame is in ambient-guidance.
- **Rationale:** a runbook is a note kind. Each of these behaviors already belongs to a surface that
  owns the same behavior for the other kinds.
- **Alternatives:** a runbook-memory capability, as legacy P6. Rejected because it would cut across
  three surfaces.

## 792d18 The QA kind is documented as current

- **Choice:** `engram learn qa` capture belongs to note-writing, and ad hoc QA capture to learn. Both
  are documented as current, even though #727 and #765 are reconsidering the kind.
- **Rationale:** the kind ships today. A reconsideration is a future change, not current behavior.
- **Alternatives:** a separate `question-answer-memory` capability. Rejected as one surface split in
  two.

## 792d19 One ambient-guidance capability

- **Choice:** recall.md, learn.md, delegate.md and shim.md, together with their activation, form one
  `ambient-guidance` capability.
- **Rationale:** they share one opt-in deployment (`--with-guidance`) and one activation mechanism.
  #773 is reconciling their overlapping cues, and separate files would freeze that duplication.
- **Alternatives:** four capabilities named after the AGENTS.md labels.

## 792d20 `lessons-contract` is its own capability

- **Choice:** the LESSONS line contract has one capability. The OpenSpec `learn`, `please` and
  `route` specs map to it, because their content is entirely the LESSONS slice.
- **Rationale:** a single owner prevents duplicating the rule across learn, please, route and the
  guidance capability.
- **Alternatives:** put the requirements in each skill's capability.

## 792d21 Vault identity comes first; `ENGRAM_SERVER` belongs to parent-sync

- **Choice:** `vault-identity` is the first capability. It covers path resolution, first-use
  creation, the vault ID, and the copy/clone and self-parent refusals. The `ENGRAM_SERVER` **MUST
  NOT** sits in parent-sync.
- **Rationale:** every command depends on resolving the vault. `ENGRAM_SERVER`'s replacement is
  `ENGRAM_PARENT`.
- **Alternatives:** fold vault identity into parent-sync, as code-tests discovery proposed. Rejected
  because first-use creation affects every user, not only users of the exchange.

## 792d22 Counting is separate from health

- **Choice:** `vault-counting` (`count`) is separate from `vault-health` (`check`, `embed`).
- **Rationale:** ADR-0018 makes counting a distinct surface, and route depends on it as an evidence
  audit.
- **Alternatives:** a merged `vault-health`.

## 792d23 A vault-migration capability

- **Choice:** `update --reparent-luhmann/--answers`, `--backfill-identity`, and the flat-vault notice
  form `vault-migration`.
- **Rationale:** they are one-shot structural maintenance of an existing vault, and share neither
  trigger nor contract with installation.
- **Alternatives:** fold them into install-and-update.

## 792d24 The vault exchange is split three ways

- **Choice:** vault-serving (`engram serve`), parent-sync (child side), and curate (the pending-offer
  review skill).
- **Rationale:** each has a different user and entry point: the host operator, the child vault user,
  and the curating agent.
- **Alternatives:** one `vault-federation` capability.

## 792d25 Eval tooling is a provisional last capability

- **Choice:** `memory-effectiveness-evaluation` and its components `eval-trial-harness` and
  `memory-loop-auditor` come last, pending 792e1.
- **Rationale:** five OpenSpec specs and ADR-0028 must go somewhere for the deprecated docs to empty.
  Placing these last means a "no" from Joe removes the group without reordering anything.
- **Alternatives:** delete them as dev tooling. Vault note 293 says not to spec dev tooling as
  product, but ADR-0028 D5 calls the audit product.

## 792d26 Component inventory: 38 components

- **Choice:** the code-tests discovery list of 34 components is the base, because it is grounded in
  code and tests. It is adjusted as follows:
  - `installer` is split into `self-installer` (ADR-0023) and `harness-deploy-sync` (ADR-0022).
  - `vault-bootstrap` is folded into `vault-identity-registry`.
  - `recall-usage-tracking` (activate) is folded into `note-amendment`.
  - Note lookup (`show`) becomes caller routing in `command-line-surface`.
  - `vault-statistics` is kept as `vault-counter`.
  - `agent-instruction-bundle` is added (792d5), and so are the two eval components (792d25).
- **Rationale:** each kept component meets all five qualification criteria. Each split follows an
  ADR boundary, and each fold removes a contract too thin to stand alone.
- **Alternatives:** the legacy report's 32 components, which include the skill procedures (792d4),
  and the history report's 14 package-level groupings, which merge independent contracts.

## 792d27 Excluded component candidates

- **Choice:** these candidates are excluded:
  - `debug-log` fails necessity: no production code emits a log line.
  - `interrupt-handling` fails necessity: no product requirement mentions interruption.
  - The composition root (`cli.Primitives`, `cli.NewDeps`, `cmd/engram`) is a dependency-injection
    seam, which the guide says can't qualify on that basis alone.
  - Their tests are listed as no-scenario tests in the `command-line-surface` node. `debug-log`'s
    legacy text (c3 K11) is deleted in node 792c8a.
- **Rationale:** the guide's necessity rule, and its rule that a mockable seam does not qualify
  solely because it can be injected.
- **Alternatives:** keep them as MAY-level components.

## 792d28 Component names

- **Choice:** names describe each contract, with no package or function names. For example,
  `query-engine`, not `semantic-retrieval-engine`; `query-payload-renderer`, not
  `recall-payload-renderer`, because recall is the skill; `vocabulary-engine`; `embedding-sidecar`;
  `note-provenance`; `pending-offer-bookkeeping`.
- **Rationale:** the guide's stability criterion, plus Joe's vocabulary (792d16).
- **Alternatives:** the discovery reports' names.

## 792d29 Order and topology

- **Choice:** the order is foundational first: vault, notes, chunks, vocabulary, query, maintenance,
  deployment, skills, guidance, exchange, evaluation. Within a capability, callees come before
  callers. The nodes run as a linear chain, except that 792c8b and 792c8c may run in parallel.
- **Rationale:** each component is migrated in the first capability that needs it, so foundational
  components land first and later capabilities only extend them. Every node edits the same 792a,
  792b and 792c files, and Joe asked for capability-by-capability order. Both facts serialize the
  work.
- **Alternatives:** parallel worktrees per capability. Rejected because they would conflict on the
  shared change files.

## 792d30 Components are migrated once and extended later

- **Choice:** a component node writes the component's section in full. Later capability nodes add
  mapping rows and any newly sourced requirements to that section (step 6 of 792c4a), and rename the
  tests that newly match.
- **Rationale:** this is Joe's instruction: migrate each component once, then add mappings.
- **Alternatives:** a separate node per capability-component pair. Rejected as 65+ extra nodes.

## 792d31 Each legacy item has one owning node

- **Choice:**
  - Every legacy file, section, ADR and OpenSpec spec is listed under exactly one node for each part
    it contains. Nodes remove only what they port, a file is deleted once it is empty, and node
    792c8d removes whatever remains.
  - Coverage: 21 files under `docs/deprecated/`, 29 ADRs plus the "not made into ADRs" section, 53
    OpenSpec specs, and 42 archive entries. The discovery brief said there were 56 specs; there are
    53.
  - Archive entries are read-only rationale sources until node 792c8c deletes them.
- **Rationale:** per-part ownership lets a mixed file (an ADR, a C4 view, a mixed spec) be emptied
  across several nodes without losing content.
- **Alternatives:** file-level ownership. Rejected because a node can't port content another node
  has already deleted.

## 792d32 Skill product specs are derived from the shipped SKILL.md

- **Choice:** where a legacy source doesn't fully describe a skill or guidance behavior, the product
  spec describes the shipped SKILL.md or guidance file.
- **Rationale:** most of the recall, learn, please, route and curate behavior exists only in SKILL.md.
  The OpenSpec skill specs are delta-shaped slices (legacy report, Overlaps).
- **Alternatives:** document only what the legacy specs state. Rejected because it would leave the
  capabilities mostly empty.

## 792d33 Behavior is described, not validated

- **Choice:** nodes describe current behavior from the legacy docs, code and tests, without
  confirming each requirement with Joe. The product guide's per-requirement interview is replaced by
  this log and 792e.
- **Rationale:** Joe said "Do not attempt to validate correctness yet." Vault note 645: in a
  descriptive spec tree, a divergence found later becomes a change proposal.
- **Alternatives:** an interview for each requirement. Rejected because Joe deferred validation.

## 792d34 References are repointed to the change sections

- **Choice:** stale references are repointed to the `792a` and `792b` section anchors, which exist
  now. At promotion, one grep for `792a_` and `792b_` moves them to `docs/specs/...`. The IDs are
  stable, so the anchors survive.
- **Rationale:** the canonical files won't exist until promotion (792d44). Pointing at them now would
  break links in every commit until then.
- **Alternatives:** point at the future canonical paths.

## 792d35 Product scenarios are restated at their first technical mapping

- **Choice:** a product scenario is restated in full in 792b at the first place it is mapped, and
  referenced by ID afterwards.
- **Rationale:** the technical guide requires it. The rule against duplicating scenario text applies
  only to canonical technical specs, and promotion drops these restatements.
- **Alternatives:** IDs only. Rejected as a deviation from the guide.

## 792d36 ID templates and overflow

- **Choice:** the templates in 792c3a and 792c3b match the `docs/AGENTS.md` example. Capability
  sections are `## 792aN Added: <name>`, requirements `### 792aNx`, UAT groups `#### 792aNx1`, and
  scenarios `792aNx1a`. Components follow the same pattern with letter subsections a–d. More than 25
  letter siblings continue as `za`–`zy`. Section numbers are fixed in the catalog: 792a2–792a23 and
  792b2–792b39.
- **Rationale:** fixed numbers stop later nodes from drifting. Reserving `z` as a prefix keeps
  letter IDs unambiguous.
- **Alternatives:** assign numbers as nodes run.

## 792d37 Exemptions from the stale-reference sweep

- **Choice:** the sweep exempts `dev/eval/raw/` (named by #792) and dated eval records, which are
  historical results: `dev/eval/LEDGER.md` rows, `dev/eval/audit/*` reports, `dev/eval/cumulative/**`
  analyses, and fixture encodings. Code comments in live tooling (`dev/eval/isolation.py`,
  `dev/eval/cumulative/test_matrix.py`) are repointed.
- **Rationale:** #792 exempts historical records. Live comments mislead the next reader.
- **Alternatives:** exempt only `dev/eval/raw/`. Rejected because it would mean rewriting records of
  past experiments.

## 792d38 `TARG_BASELINE_PATTERN` (provisional)

- **Choice:** `dev/targs.go` sets `TARG_BASELINE_PATTERN='TestT[0-9]+_'`. Only eight tests match it,
  and the renames in 792c7b5 and 792c7d2 remove that prefix. The first node that renames a `TestT`
  test updates the pattern to match the new scenario-ID prefix, which keeps the old tests in
  baseline selection. Node 792c8e confirms the final value.
- **Rationale:** without the update, those renames would silently empty the baseline selection. targ
  is a black box, so the final value is asked as 792e7.
- **Alternatives:** keep the `TestT` prefix on those tests. Rejected because it conflicts with
  792d10.

## 792d39 GLOSSARY

- **Choice:** retired-term entries are dropped: Permanent, MOC, vocab nomination, term-note,
  vocab-index, `Vocab:` line, and `ENGRAM_SERVER`, whose absence is now a spec MUST NOT. Normative
  definitions link to their spec section instead of restating the rule.
- **Rationale:** git is the fallback. Keeping one normative copy follows the lifecycle guide.
- **Alternatives:** keep a "retired" section.

## 792d40 SKILL.md reference edits follow the root rule

- **Choice:** repointing a path inside a SKILL.md, as in route SKILL.md, is a SKILL.md edit, and the
  root `AGENTS.md` skill-editing rule applies to it. This plan doesn't waive the rule or name a
  skill.
- **Rationale:** the root `AGENTS.md` says the rule has no exceptions, and the plan guide forbids
  prescribing skills.
- **Alternatives:** none.

## 792d41 Root README stays as orientation

- **Choice:** root `README.md` keeps its orientation content and CLI reference. Only its links are
  repointed, in nodes 792c8c and 792c8d.
- **Rationale:** it isn't under `docs/` and isn't legacy. It is the repository's front page.
- **Alternatives:** move the CLI reference into the product specs. Rejected as unrequested scope.

## 792d42 Known divergences between docs and code

- **Choice:** where the docs and the code clearly disagree about whether a feature is live, the
  node describes what the code does now and records the divergence here. Known cases so far:
  - First-use bootstrap creates `Permanent/` even though the docs call it retired.
  - `learn --tier` is still a flag after tiers were removed.
  - Recency decay is inactive (#774).
  - `amend` has undocumented `--pending/--repo/--user` flags.
  - c3's "Dead/test-only surface" section is stale, because `count` uses `BuildGraph`.
- **Rationale:** 792d6 and 792d33. A divergence becomes a later change proposal (vault note 645).
- **Alternatives:** fix the code during migration. Rejected as out of scope.

## 792d43 Build-tag classification is a separate, deferred node

- **Choice:** node 792c8e classifies tests as `integration` or `network`. The migration nodes don't.
- **Rationale:** this was decided up front, and it keeps renames separate from build-tag edits.
  Discovery found about 70 candidate files, no `network` cases, and an existing `parity` tag.
- **Alternatives:** classify tests in each component node.

## 792d44 Promotion is excluded from this plan

- **Choice:** applying the deltas to `docs/specs/` and deleting the change directory happen after Joe
  validates this log and answers 792e.
- **Rationale:** Joe deferred validation of correctness, and promoting would make unvalidated text
  canonical.
- **Alternatives:** a promotion node at the end. Its timing is asked as 792e6.

## 792d45 `792f_discovery/` holds the working inventories

- **Choice:** `792f_discovery/{legacy-docs,code-tests,history-memory}.md` holds the three discovery
  reports that this plan reconciles. The directory is deleted with the change directory.
- **Rationale:** the session scratchpad that first held them was lost. Nodes cite their test maps and
  source maps.
- **Alternatives:** inline the reports into 792c. Rejected as duplicating working data in the plan.

## 792d46 First-use creation does not create `Permanent/`

- **Choice:** 792a2e describes the vault that first-use creation actually builds: `.obsidian/app.json`,
  `.gitignore`, the starter `README.md`, the vault ID and `.engram/`. The first known divergence in
  792d42 is corrected: no code creates `Permanent/`. Only the doc comment on `initializeVault`
  (`internal/cli/vault_init.go`) still names it. Root `README.md` "Vault location" also still says
  that only `engram learn` creates a missing vault and that other commands fail with "vault not
  found"; every vault-resolving command creates it now. Both stale texts are left for a later change
  (792d41, 792d33).
- **Rationale:** discovery inferred the divergence from the comment. Reading `initializeVault` and
  `Test792b2c3a1_Targets_EveryVaultCommandCreatesMissingVault` shows the current behavior.
- **Alternatives:** fix the comment and README now. Rejected as outside a documentation-only node.

## 792d47 Exchange-state location belongs to vault-identity

- **Choice:** the OpenSpec requirement "Exchange state SHALL live outside the vault's tracked files"
  is ported to 792a2k, not left for parent-sync (792c7t1).
- **Rationale:** `.engram/` is created with the vault and its ID, before any parent is configured,
  and `vault-identity-registry` owns it. Parent-sync keeps what is written there (outbox, declines,
  parent cache).
- **Alternatives:** leave it with the exchange requirements for 792c7t1.

## 792d48 The vault name is specified in vault-identity

- **Choice:** `--vault-name`/`ENGRAM_VAULT_NAME` and its default `personal` are requirement 792a2b,
  as the 792c7a1 catalog lists. Note-writing and note-provenance (792c7b1, 792c7b7) map their
  `vault:` stamping to 792a2b rather than restating it.
- **Rationale:** the precedence mirrors the vault path's (792a2a), and one owner avoids duplicating
  the rule.
- **Alternatives:** specify it only under note-writing.

## 792d49 Product requirements beyond the OpenSpec text

- **Choice:** 792a2 adds requirements that the legacy sources don't state but the code and tests
  show: the flat layout's exclusion of subdirectories (792a2c), the starter README (792a2d), repair
  of a half-created vault (792a2g), corrupt-ID handling (792a2j), the ignored, committed and non-git
  cases of the uncommitted-ID notice (792a2i), the `vault-id` outputs and flag errors
  (792a2l–792a2o), and the merged query's warning on a clone (792a2p1b).
- **Rationale:** 792d33 describes current behavior from code and tests. Each addition is observable
  and already tested.
- **Alternatives:** port only the OpenSpec text.

## 792d50 Technical clauses of a mixed OpenSpec spec stay in place for their component node

- **Choice:** when a capability node ports a mixed OpenSpec spec, the technical clauses listed for a
  later component node stay in the spec file under a residue requirement that names the node. For
  `vault-local-first`, the exclusive-create-and-re-read, injected random source, location-record
  contents, and regenerate/claim file effects wait for 792c7a5.
- **Rationale:** 792d31 per-part ownership. A node removes only what it ports.
- **Alternatives:** port the technical clauses early into 792a. Rejected because product specs
  exclude implementation detail.

## 792d51 command-line-surface routing requirements arrive with their scenarios

- **Choice:** 792b2 specifies only the routing that vault-identity's scenarios need: vault path and
  vault name resolution, first-use gating and result reporting. The `ENGRAM_SERVER` pre-dispatch
  guard, chunks-dir resolution and `show` local/parent routing are added to 792b2 by parent-sync
  (792c7t1), chunk-indexing (792c7d1) and memory-query (792c7f1), which introduce the product
  scenarios those requirements support. Their tests stay unrenamed until then and are listed as
  deferred no-scenario tests. The C2 container row's per-command detail is deleted rather than
  ported, because root `README.md` "Binary commands" and the later nodes' sources state the same
  facts; 792b2a keeps the command inventory.
- **Rationale:** the technical guide requires every requirement to name the product scenarios it
  supports, and those scenarios don't exist yet. 792d30 already has later capability nodes extend a
  migrated component.
- **Alternatives:** write the requirements now with no supported scenarios. Rejected as untraceable.

## 792d52 Test IDs: product scenario for end-to-end checks, technical scenario otherwise

- **Choice:** a test that runs a whole command through dispatch and asserts a product scenario's
  observable outcome takes the product scenario ID (for example
  `Test792a2e1a_Targets_QueryMissingVaultReturnsEmptyResult`). A test of one component's contract
  takes the technical scenario ID (for example `Test792b2c1a1_ResolveVault_FlagWins`).
- **Rationale:** 792c4b step 4 renames tests that verify either spec's scenarios; this rule picks
  one ID deterministically when a test could be read as verifying both.
- **Alternatives:** always use the technical ID. Rejected because an end-to-end test is the direct
  evidence for its product scenario.

## 792d53 Test-rename reference updates skip historical records

- **Choice:** when a test is renamed, references are updated everywhere except the 792d37 exemption
  set (`dev/eval/raw/`, `dev/eval/audit/`, `dev/eval/LEDGER.md`, `dev/eval/cumulative/`),
  `openspec/changes/archive/`, and the `.superpowers/sdd/` briefs and progress notes, which are
  dated execution records of finished work.
- **Rationale:** rewriting a dated record falsifies what it recorded. The archive is deleted by
  792c8c, and the `.superpowers/` plan material is handled under 792d12.
- **Alternatives:** rewrite every occurrence. Rejected: the first such rename rewrote eval audit
  transcripts, which 792d37 exempts.

## 792d54 Node agents commit their own nodes

- **Choice:** the agent executing a capability group commits each node itself as the node
  completes (one commit per node, `docs(792): <node ID> <title>`), instead of the orchestrator
  committing (as 792c4b's last line originally said).
- **Rationale:** a group runs several nodes in sequence; per-node commits by the executor keep each
  status transition in the commit that establishes it (implementation-plan guide) without
  stalling between nodes.
- **Alternatives:** orchestrator commits after each node. Rejected: it forces one agent dispatch
  per node, roughly tripling the dispatches for no review benefit.

## 792d55 `learn --tier` is documented as a frontmatter field

- **Choice:** 792a3i specifies `--tier`: it writes `tier: L2` by default and writes `L1`, `L2` or
  `L3` as given, and 792a3g refuses any other value. This resolves the second known divergence in
  792d42: the flag survives the removal of tiers because it still has an observable effect.
- **Rationale:** 792d33 describes current behavior. The note's `tier:` field is visible to the user,
  even though no retrieval path reads it.
- **Alternatives:** leave `--tier` out as a retired feature (792d6). Rejected because the field is
  still written to every note.

## 792d56 Placement under a target that does not exist is permitted

- **Choice:** 792a3l states, as **MAY**, that `engram learn --position continuation` or `sibling`
  can place a note under a `--target` that names no note: `--target 77` in a vault without note 77
  writes note `77a`. No scenario protects it.
- **Rationale:** the code does not check that the target exists, and no legacy source promises
  either outcome. Stating it as **MAY** describes the behavior without making it a contract. A
  change could later make it a refusal (vault note 645).
- **Alternatives:** a **MUST NOT** with a coverage gap. Rejected because it would describe behavior
  the code does not have.

## 792d57 Mixed sources of note-writing are split by owner

- **Choice:** node 792c7b1 ports only the note-writing parts of its mixed sources, and leaves the
  rest in place for the node that owns it (792d31, 792d50):
  - `learn-qa-capture` is ported whole and deleted.
  - `learn-runbook-capture` keeps the write-memory requirement for 792c7l1 and the skill runbook note
    requirement for 792c7k1.
  - `vault-embed-on-write` keeps everything but the `learn` guarantee. The `amend` and `resituate`
    re-embedding requirement is added to the sources of note-revision (792c7c1).
  - `vault-note-identity` keeps a residue requirement with the repo and user detection mechanism, the
    first-write user rule for served learns and pull-downs, and the empty-vault-name rules, for
    792c7b7.
  - `vault-concurrency-safe-writes` keeps all its text for 792c7b4; only its Purpose gains a pointer.
  - ADR-0012 keeps its round-2 and round-3 gating caveat for 792c8a, the `vocab stats` QA lines for
    792c7e1, and the `stripMachineLines` QA markers for 792c7b6.
  - ADR-0026 keeps its retrieval decision for 792c7f1, its learn-routing decision for 792c7n1, its
    historical consequences for 792c8a, and its two 2026-09-26 registration amendments for 792c7k1.
  The catalog entries of those nodes now list these parts.
- **Rationale:** each part has one owner, and a node removes only what it ports.
- **Alternatives:** port the other parts into 792a3 now. Rejected because they describe other
  capabilities' behavior.

## 792d58 Note-writing requirements beyond the legacy text

- **Choice:** 792a3 adds requirements that the legacy sources don't state but the code, `--help` and
  tests show: the fact and feedback body formulas and the dropped leading `When` (792a3a, 792a3b),
  the argument validations (792a3g), the optional metadata fields (792a3h), the letter rollover and
  the placement refusals (792a3j, 792a3k), the empty-situation refusal (792a3f), the QA validations
  and certainty default (792a3r), and the warning when embedding fails (792a3n).
- **Rationale:** 792d33 and 792d49. Each addition is observable through `engram learn` and is
  already exercised by the shipped binary.
- **Alternatives:** port only the OpenSpec text.

## 792d59 QA retrieval and per-kind vocabulary tagging belong to note-writing

- **Choice:** 792a3t (query returns answers but never questions) and 792a3u (vocabulary tagging
  applies to every kind but questions) are note-writing requirements, although `engram query` and
  the vocabulary engine carry them out. The query-engine (792c7f5) and vocabulary-engine (792c7e3)
  nodes map these scenarios to their requirements.
- **Rationale:** both are the defining contract of the QA kind (ADR-0012), which `learn-qa-capture`
  states and this node owns. Restating them in memory-query and controlled-vocabulary would duplicate
  the rule.
- **Alternatives:** move 792a3t to memory-query and 792a3u to controlled-vocabulary.

## 792d60 command-line-surface refuses undefined and missing required flags

- **Choice:** note-writing adds requirement 792b2c5 to command-line-surface: an undefined flag or a
  missing required flag is refused before the command runs. `TestTargets_LearnRunbook_NoSkillHashFlag`
  and `TestTargets_NoSkillKeyOrSourceFlags` are renamed to its first scenario, 792b2c5a1; the
  skill-runbook-registration node (792c7k1) lists its own scenario for them as `split deferred`.
- **Rationale:** 792a3e1a, 792a3f1a and parts of 792a3g1a and 792a3r1a are refused by argument
  parsing, which this component owns (792b2a), before any `learn` code runs. Those two tests are the
  existing direct tests of that refusal.
- **Alternatives:** leave the tests for 792c7k1. Rejected because 792b2c5a1 would then have no test.

## 792d61 note-document-format's edit, CRLF and field-survival contracts arrive with their scenarios

- **Choice:** 792b6 specifies only what note-writing's scenarios need: filenames, the frontmatter
  block, each kind's keys, the body formulas, the QA documents and frontmatter splitting. These wait
  for the capabilities whose product scenarios they support, as 792d51 does for command-line-surface:
  - YAML-node edits that keep unknown keys and anchors, refuse an anchored or undecodable rewrite,
    and read CRLF notes as LF go to note-revision (792c7c1), whose sources now list the
    `vault-note-identity` note-reader requirement and the amend and resituate paragraphs of its
    field-survival requirement.
  - Survival of the exchange fields goes to parent-sync (792c7t1), which already owns them.
  - The `skill_hash`, `skill_key` and `skill_source` fields go to skill-runbook-registration
    (792c7k1), whose sources now list those `vault-note-identity` requirements.
  - The "CRLF-safe reading requirement" this node listed from `update-reparent-luhmann-batch` is the
    rename-and-rewrite step's, which vault-migration (792c7i1, 792c7i2) already owns.
  Their tests stay unrenamed until then and are listed as deferred no-scenario tests.
- **Rationale:** the technical guide requires every requirement to name the product scenarios it
  supports, and none of these has a scenario yet.
- **Alternatives:** specify them now with no supported scenarios. Rejected as untraceable.

## 792d62 A fact or feedback situation is stored twice

- **Choice:** 792b6a describes the current form: a fact or feedback note holds its situation both in
  `situation:` and in its body formula. The legacy invariant S2 called this a defect and asked for
  one representation, and it said only the body was embedded and hashed, with `engram check` to
  compare the two copies. Today the situation vector is built from `situation:`, the content hash
  covers both the situation and the body, and `engram check` does not compare the copies.
- **Rationale:** 792d33. The S2 text is stale on the embedding and was never implemented on the
  check; the double copy is current behavior.
- **Alternatives:** a **MUST** that the copies match, enforced by `check`. Rejected because nothing
  enforces it.
