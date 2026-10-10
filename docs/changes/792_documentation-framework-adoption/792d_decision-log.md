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

## 792d63 Which writer takes which lock is the writer's requirement

- **Choice:** 792b8 specifies the lock itself: exclusive hold of a lock file, its creation, waiting,
  release and failure reporting, and the vault's lock file `.luhmann.lock`. The per-command clauses
  of `vault-concurrency-safe-writes` stay in that spec for the callers that own them:
  `learn` and `learn qa` for note-authoring (792c7b8); `amend`, `resituate`, `activate` and the
  lock-free shared helpers for note-amendment (792c7c3); the manifest lock for chunk-index-store
  (792c7d5); and `vocab bootstrap` and `refit` for vocabulary-engine (792c7e3). Those nodes' sources
  now list them. ADR-0013 is reduced to a pointer to 792b3 and 792b8, and code comments citing it
  for the lock now cite 792b8.
  `TestInvariant_K1_ConcurrentLearnNeverCollides` takes product scenario 792a3m1a, because it drives
  the real locked write path and asserts the product outcome.
- **Rationale:** the technical guide puts an interaction requirement (what a caller does, and in
  what order, while holding a dependency) in the caller's component. Holding the lock across
  ID allocation and the write is note-authoring's ordering, not the lock's behavior.
- **Alternatives:** port every lock clause into 792b8 now. Rejected because most would support no
  existing product scenario and would describe callers' ordering.

## 792d64 `TARG_BASELINE_PATTERN` now matches both test-name forms

- **Choice:** node 792c7b5 renamed the first two `TestT<N>_` tests (`TestT10_…` and `TestT13_…`),
  dropping the old prefix as 792d38 says. `dev/targs.go` now defaults the pattern to
  `Test(T[0-9]+|[0-9]+[a-z][a-z0-9]*)_`, which matches the remaining `TestT138`–`TestT143` tests
  until transcript-reader (792c7d2) renames them, and every scenario-ID test name.
- **Rationale:** 792d38 asks the first such rename to keep the renamed tests in baseline selection.
  The scenario-ID form is the new spec-trace naming, so the pattern follows it. The final value is
  still 792e7's question.
- **Alternatives:** match only the eight original tests by their new names. Rejected because new
  spec-traced tests would never be selected.

## 792d65 The missing-model error names an unimplemented `ENGRAM_MODEL_PATH`

- **Choice:** 792b9c5 states only that a binary without the bundled model fails with an error saying
  the model is missing. That error also tells the user to set `ENGRAM_MODEL_PATH` to a directory
  holding `model.onnx`, but no code reads `ENGRAM_MODEL_PATH`. The divergence is recorded here, not
  specified.
- **Rationale:** 792d33 and 792d42. The advice is not current behavior.
- **Alternatives:** specify the variable. Rejected because setting it changes nothing.

## 792d66 Embedding-sidecar scope, and the partial-model-mismatch gap

- **Choice:**
  - 792b10 specifies the sidecar format and path, the embedding sources, the content hash,
    building, the learn-time write, state classification and cosine similarity. The `engram embed
    apply` and `status` selection waits for vault-health (792c7h1), and the `amend`, `resituate` and
    `activate` sidecar writes for note-revision (792c7c1); `vault-embed-on-write` keeps those
    residues, and both nodes' sources now list them. Their tests (`embed_test`, `TestSelectStates`)
    stay unrenamed; `TestKindFromContent` reads a note's kind for the query payload and is left to
    memory-query (792c7f1).
  - The known defect M4 is described, not specified: a query drops sidecars made by another model
    with only a stderr advisory, so a partial model change silently thins its results, and only a
    vault with no compatible sidecar at all fails. 792b10c7's rationale states this.
  - ADR-0006 (embed source by kind) is deleted. Its only surviving content, that every kind embeds a
    situation and a body, is in 792b10's rationale; the episode kind it routed is retired (792d6).
    ADR-0002 and ADR-0003 are reduced to pointers.
- **Rationale:** 792d51, 792d33 and 792d6.
- **Alternatives:** a **MUST** that all sidecars share one model. Rejected because nothing enforces
  it.

## 792d67 note-provenance covers detection and the first-write stamp

- **Choice:** 792b11 specifies repository and user detection and the first-write stamp, which
  note-writing's scenarios need. The amend re-stamp, which keeps a prior user when detection comes
  back empty, goes to note-revision (792c7c1); identity backfill and its `project:` preference go to
  vault-migration (792c7i1), which already owns them; and first-write stamping by a served learn and
  by a pull-down go to vault-serving (792c7s1) and parent-sync (792c7t1). `vault-note-identity`
  keeps those clauses, and the nodes' sources now list them. Their tests stay unrenamed.
  `TestDetectRepo` and `TestDetectUser` are tables whose cases cover every scenario of their
  requirement, so each takes the first scenario's ID and lists the rest as `split deferred`.
- **Rationale:** 792d51. Only `learn` has product scenarios so far.
- **Alternatives:** specify the amend and backfill stamping now, with no supported scenarios.

## 792d68 note-authoring scope and the tests it leaves

- **Choice:** 792b12 specifies argument validation, required content, the locked write, the
  mapping of arguments to fields, and the question-and-answer write. These stay for later nodes:
  the vocabulary refit trigger after a learn (`TestApplyVocabAssignmentAfterLearn_TriggerFires`) and
  `vocab_version` survival through amend (`TestFactAmend_VocabVersionRoundTrip`) for
  controlled-vocabulary (792c7e1) and note-revision (792c7c1); the rule that `learn` never writes the
  skill fields (`TestRunLearn_Runbook_NeverWritesSkillHash`,
  `TestRunLearn_Runbook_NeverWritesSkillKeyOrSource`) for skill-runbook-registration (792c7k1); and
  the question-note exclusions in query and `vocab stats` for memory-query (792c7f1, query-engine
  792c7f5) and controlled-vocabulary. The c3 QA flow's notes on those exclusions stay in
  `c3-components.md` as a residue, and those nodes' sources list it. The c2 learn flow keeps its
  skill side for learn (792c7n1). `TestLearnFact_BootstrapsMissingVault`, a dispatch test that
  `learn` creates a missing vault first, takes command-line-surface's 792b2c3a1 (792c4a step 6).
- **Rationale:** 792d51, and 792d52 for the bootstrap test.
- **Alternatives:** none considered.

## 792d69 Mixed sources of note-revision are split by owner

- **Choice:** node 792c7c1 ports only the note-revision parts of its sources (792d31, 792d57):
  - `vault-note-identity`: the amend re-stamp, its `project:` exclusion, the vault fill and the
    amend and resituate key-preservation, anchor and CRLF paragraphs are ported. The bookkeeping
    rule for `--clear-pending`, `--discard --into` and the offer receipt stays as a residue
    requirement for curate (792c7u1) and parent-sync (792c7t1), with the fold's CRLF and
    `--expect-hash` clause and the "Accepting an offer keeps its author" and fold scenarios. The
    clear-pending vault-fill scenario is replaced by an `--activate` one (792a4i1a), because the
    rule covers every amend that writes.
  - The note-reader CRLF requirement is split: its embedding clause is 792b10c9, and amend's and
    resituate's reads are 792a4m. Its query, check, count and vocabulary readers stay in place for
    memory-query (792c7f1), vault-health (792c7h1), vault-counting (792c7g1) and
    controlled-vocabulary (792c7e1), whose sources now list them; the `crlf_readers_internal_test`
    tests go with them.
  - `vault-embed-on-write`: the amend and resituate re-embedding and the lossless `last_used` bump
    are ported; the `embed apply` residue stays for 792c7h1.
  - `vault-wikilink-resolution`: the canonical-form requirement is ported (792a4d). The parser
    requirement and its two resolution scenarios wait for wikilink-graph (792c7c2), and the
    dangling-link scenario for 792c7h1.
  - c3 K4w is only a node of the L3 diagram, with no catalog row. It states no product behavior, so
    it is left whole for note-amendment (792c7c3).
- **Rationale:** each part has one owner, and a node removes only what it ports.
- **Alternatives:** port the CRLF reader requirement whole into 792a4. Rejected because its
  scenarios are query, check, count and vocabulary behavior, which other capabilities own.

## 792d70 `amend --repo`, `--user` and `--pending` are not specified

- **Choice:** 792a4 leaves out the three undocumented `amend` flags that `--help` lists without a
  description (the 792d42 divergence). `--repo` and `--user` are accepted and have no effect on a
  CLI amend; they carry a served write's identity across the wire. `--pending` fails with
  "unsupported value type: *bool" for any value, because the field is a pointer that targ cannot
  parse.
- **Rationale:** 792d33 and 792d42. None of them is observable behavior a user can rely on.
- **Alternatives:** specify `--repo` and `--user` as ignored (MUST NOT affect the note). Rejected as
  protecting an accident of how the argument struct is shared with the served path.

## 792d71 Rebuilding a sidecar drops its last-used date

- **Choice:** a content amend or a resituate rebuilds the note's sidecar from scratch, and the new
  sidecar carries no `last_used`, so a recorded use is lost. 792a4n specifies only that an
  activation keeps the rest of the sidecar; the loss on a rebuild is recorded here, not specified.
- **Rationale:** 792d33. Whether a rebuild should carry `last_used` forward is a change proposal.
- **Alternatives:** a **MUST** that a rebuild keeps `last_used`. Rejected because the code does not.

## 792d72 Note-revision requirements beyond the legacy text

- **Choice:** 792a4 adds requirements that the legacy sources don't state but the code, `--help` and
  tests show: the accepted reference forms and the not-found refusal (792a4a), field replacement by
  kind and the rebuilt body formula (792a4b), the refusal of other kinds (792a4c), the chunk-source
  merge (792a4e), amend's argument validation (792a4f), resituate's limit to fact and feedback and
  its failure when the sidecar cannot be rebuilt (792a4l), and activation (792a4n). 792a4o states
  for revisions the guarantee that `vault-concurrency-safe-writes` gives every vault writer.
  `activate --note` resolves a path or basename, not a bare Luhmann ID, unlike amend and resituate;
  792a4n describes this as it is.
- **Rationale:** 792d33, 792d49 and 792d58.
- **Alternatives:** port only the OpenSpec text.

## 792d73 One rule for the supersession link form

- **Choice:** 792a4d states the canonical `Supersedes: [[<target>]]` line for both `engram learn` and
  `engram amend`, as the `vault-wikilink-resolution` requirement does. 792a3h already shows the
  learn line for a target given without `.md`.
- **Rationale:** both commands write the line through one renderer, and a single owner keeps the rule
  from being restated in note-writing.
- **Alternatives:** add a learn-only requirement to 792a3. Rejected as duplicating the rule.

## 792d74 Where note-revision's technical contracts land

- **Choice:**
  - The YAML-node edits and the CRLF read are note-document-format's 792b6c7 and 792b6c8 (792d61),
    the re-stamp is note-provenance's 792b11c4 (792d67), and the CRLF embedding read is
    embedding-sidecar's 792b10c9.
  - The sidecar rebuild after a revision adds no embedding-sidecar requirement: it builds through
    792b10c5 and writes through 792b3c1, and warning or failing is the reviser's choice, specified by
    note-amendment (792c7c3). The last-used bump lives in `activate.go` and is note-amendment's too.
  - `TestIdentityStamp_Stamp_OverwritesAllThree` verifies the raw stamp that blanks an empty user,
    which no revision uses, so it is a no-scenario test. 792b11c4a1 is left to the re-stamp property
    test that note-amendment examines.
  - `TestTargets_Resituate` and `TestActivate_MissWithoutParentFails` dispatch whole commands and take
    the product IDs 792a4a1b and 792a4n1c (792d52).
- **Rationale:** 792d30 and 792d51: a migrated component gains the requirements that a later
  capability's scenarios need.
- **Alternatives:** a separate sidecar-rebuild requirement in 792b10. Rejected because building and
  writing are already specified.

## 792d75 wikilink-graph's link, graph and resolution contracts arrive with their scenarios

- **Choice:** 792b13 specifies only what existing product scenarios need: the vault scan (used by
  `amend`, `resituate`, `embed status` and `check`) and note-reference resolution. These wait, as in
  792d51 and 792d61:
  - link parsing (`.md` normalization, fences, dedupe), the graph build and in-degree for
    vault-counting (792c7g1), because `engram count` prints a note's in-degree, with ADR-0007's
    decision and c3 K7's graph clause;
  - the unresolved-link report, the G0 invariant, the recurring-defect shape and the whole
    `vault-wikilink-resolution` parser requirement with its three scenarios for vault-health
    (792c7h1), because `engram check` reports them.
  The parser, fence, graph and resolution tests stay unrenamed until then. `BFSWithCap`,
  `InDegreeIn` and `UndirectedNeighbors` have no production caller; their tests verify no scenario.
  ADR-0007's context, that navigation lives in authored links and the graph is derived fresh, is
  ported as 792b13a's rationale.
- **Rationale:** the technical guide requires every requirement to name the product scenarios it
  supports, and no 792a2–792a4 scenario depends on how links are parsed or counted.
- **Alternatives:** specify parsing and the graph now with no supported scenarios. Rejected as
  untraceable.
- **Divergence noted:** a link's target is the whole text between `[[` and `]]`, so
  `[[note|display]]` and `[[note#heading]]` do not resolve to `note` in the graph, although note
  references (792b13c2) strip a `|display` part. Recorded for vault-counting and vault-health to
  describe, not resolved here (792d33).

## 792d76 note-amendment scope and the tests it leaves

- **Choice:** 792b14 specifies the locked read-modify-write of amend, resituate and activate: the
  lock, amend's refusals, field replacement, supersessions and chunk sources, re-stamp gating and the
  vault fill, the sidecar rebuild, CRLF revision, resituation, the last-used bump and local
  activation refs. These stay for later nodes, with their tests unrenamed: the curation flags
  (`TestDiscardNote_*`, `TestRunAmend_CRLFJudgedVersionUsesTheLFForm`,
  `TestRunAmend_DiscardIntoConvertsCRLFExisting`, `TestRunAmend_ClearPendingKeepsOfferAuthor`) for
  curate (792c7u1); the pulled-note decline (`TestRunAmend_DiscardRecordsDeclineOfCRLFPulledNote`)
  for parent-sync (792c7t1); the vocabulary assignment after a revision
  (`TestApplyVocabAssignmentAfterAmend_TriggerFires`, `TestApplyVocabAssignmentAfterResituate_*`,
  `TestRunResituate_CallsVocabAssignment`) for controlled-vocabulary (792c7e1); the skill-hash
  survival tests (`TestRunAmend_Runbook_PreservesSkillHash_*`) for skill-runbook-registration
  (792c7k1); the supersession inverse used by query (`TestBuildSupersedesInverse`) for memory-query
  (792c7f1); and the chunk-ID set read from the index (`TestBuildChunkIDSet_ReturnsSourceAnchorKeys`)
  for chunk-index-store (792c7d5). The `amend` and `resituate` lock clauses of
  `vault-concurrency-safe-writes` and the vault-lock helpers of its lock-free helper requirement are
  ported; `writeManifestFile` stays for chunk-index-store.
- **Rationale:** 792d51 and 792d63: a caller's ordering under the lock is the caller's requirement.
- **Alternatives:** none considered.

## 792d77 Mixed sources of chunk-indexing are split by owner

- **Choice:** node 792c7d1 ports the product behavior of `ingest-auto-sweep`, `ingest-chunk-dedup`
  and `ingest-prune-detach` into 792a5 and leaves each file a residue for the node that owns the
  rest (792d31, 792d50):
  - `ingest-auto-sweep` keeps the sweep-root mechanics (name prefixes on the session-log root,
    path prefixes at root derivation, the `.claude` exclude list) for ingestion-source-discovery
    (792c7d4), and the zero-record index-file clause for chunk-index-store (792c7d5).
  - `ingest-chunk-dedup` keeps the dedup mechanics for 792c7d5 and the `engram update` backlog
    notice for install-and-update (792c7j1).
  - `ingest-prune-detach` keeps the directory-scan clause for 792c7d5.
  - The prune details of the c1 S2 row are ported (792a5p–792a5r) and the row now points at 792a5.
  - ADR-0021's measurements and its context for decisions 1–8 and 10 justify the product
    requirements 792a5d, 792a5l, 792a5m, 792a5n and 792a5r, so their `Rationale:` paragraphs carry
    them. ADR-0021 itself stays whole for chunk-index-store (792c7d5), its listed owner, which adds
    the technical rationale and removes the ADR.
  - `embedding-sidecar` (`792b10`) is added to the extended components: Rule B (792a5n) recognizes a
    vault-note copy by its sidecar's name, which is 792b10c2.
- **Rationale:** each part has one owner, and a node removes only what it ports.
- **Alternatives:** port the mechanics into 792a5. Rejected because product specs exclude
  implementation detail.

## 792d78 Chunk-indexing requirements beyond the legacy text

- **Choice:** 792a5 adds requirements that the OpenSpec specs don't state but the code, `--help`
  and tests show, each confirmed against the built binary: the index location (792a5a), the named,
  Pi and swept sources and their read-failure rules (792a5b, 792a5c), the automatic roots and
  exclude lists (792a5d), `.engram/sweep.json` (792a5e), transcript stripping (792a5g), chunk
  boundaries and anchors (792a5h), incremental re-indexing and its stdout line (792a5i), the
  append-only rule (792a5j), Rule B (792a5n), the prune messages (792a5p) and `prune --duplicates`'
  origin-blind choice, refusal lines and failure exit (792a5r). The time each chunk was first
  indexed is left to memory-query (792c7f1), because only the query's recency channel makes it
  observable.
- **Rationale:** 792d33, 792d49 and 792d58.
- **Alternatives:** port only the OpenSpec text.

## 792d79 Known divergences in chunk indexing

- **Choice:** 792a5 describes these as they are and does not specify the alternative (792d33,
  792d42):
  - A source that yields no chunks is read, re-chunked and reported
    (`ingest <path>: 0 chunks (0 reused, 0 embedded)`) on every run, because the cheap skip
    requires the source's index file to exist. `TestRunIngestSkipsEmptyIndexFile`'s comment says the
    source "is not re-ingested next sweep"; its fake filesystem has no stat, which makes the skip
    succeed.
  - A source is recorded under the path as given. `engram ingest --markdown src/d.md` records
    `src/d.md`, its chunk IDs start `src/d.md#`, and `engram prune` run from another directory
    detaches it as gone.
  - `prune --duplicates` reports a refusal whose kept copy has chunks that don't cover the removed
    copy as "canonical missing but sibling content still exists", although the kept copy's index
    file exists.
  - `RunIngest`'s doc comment says stale chunks of edited content disappear; they are kept
    (792a5j). Duplicates re-accumulate when copies appear across runs (#771); this is current
    behavior, not a requirement to fix.
- **Rationale:** a divergence becomes a later change proposal (vault note 645).
- **Alternatives:** specify the intended behavior with coverage gaps. Rejected because the code does
  not do it.

## 792d80 command-line-surface gains chunk index resolution

- **Choice:** chunk-indexing adds 792b2c6, the chunk index directory's precedence, as 792d51
  planned. `TestResolveChunksDirPrecedence` covers the flag, the environment and the home fallback,
  so it takes 792b2c6a1 with the others `split deferred`; no test resolves the directory with
  `XDG_DATA_HOME` set, so 792b2c6a3 is a coverage gap. `TestTargets_IngestAndQueryChunksEmpty` and
  `TestTargets_PruneEmpty` check only that a command with nothing to do is wired and writes no
  stderr, so they verify no scenario. durable-file-store, vault-write-lock, text-embedder and
  embedding-sidecar gain mappings only; their tests that touch the chunk index are chunk-index-store's
  and are examined in 792c7d5.
- **Rationale:** 792d30 and 792d51.
- **Alternatives:** none considered.

## 792d81 transcript-reader scope and the tests it leaves

- **Choice:**
  - 792b15 specifies the production read only: the whole file, user and assistant entries, harness
    noise, oversized content and tool-call summaries. `ReadFrom`'s start time and byte budget, which
    ingestion leaves unset, and `SegmentsFrom`, `Strip` and the `KeepToolCalls` mode, which have no
    production caller, are not specified. Their tests verify no scenario and keep their names,
    including `TestT138`–`TestT143`, which test `Strip`. 792d38 expected this node to rename those
    six; the catalog's expectation that `Strip` tests are no-scenario tests wins, so the
    `T[0-9]+` alternative of `TARG_BASELINE_PATTERN` (792d64) still matches them, and 792e7 is
    unchanged.
  - The last kept entry's timestamp only feeds the time a chunk was first indexed, which is left to
    memory-query (792d78), so its two tests wait for 792c7f1.
  - ADR-0010 is deleted: its surviving JSONL reader is 792b15a's rationale, and its OpenCode backend
    and composite dispatch are retired (792d6). The c1 S5 row and the reader part of c3 K1 point at
    792b15.
  - 792a5b1d becomes a Scenario Outline over `--markdown` and `--transcript`, so a transcript that
    cannot be read is covered too.
- **Divergence noted:** a `tool_result` whose content is a list of blocks, the form Claude Code
  records for some tools, matches no summary, so neither the call nor its result appears in the
  chunk. 792a5g and 792b15c5 state this as current behavior (792d33).
- **Rationale:** 792d51 and 792d75: requirements arrive with the product scenarios they support,
  and a test-only entry point is not a component's contract.
- **Alternatives:** count the `Strip` tests as verifying the shared cleaning helpers. Rejected
  because production reaches those helpers only through the summary mode, which has its own tests.

## 792d82 source-chunking and the stale byte budget

- **Choice:** 792b16 specifies section and turn boundaries, anchors, the line-packing split and the
  40-character floor. The c2 ingest flowchart's chunk step said chunking was "byte-budget capped";
  no budget applies, because ingestion reads every transcript whole (`ingestBudgetBytes` is 0), so
  the clause is dropped and the step points at 792b16. 792a5h gains scenario 792a5h1d, because its
  split branch had no UAT scenario. An oversized section is packed line by line, not paragraph by
  paragraph as `TestChunkMarkdownSplitsOversizedSectionOnParagraphs`' name says; blank lines are
  dropped from the pieces.
- **Rationale:** 792d33; the product guide requires a scenario for every mandatory branch.
- **Alternatives:** none considered.

## 792d83 ingestion-source-discovery scope and the tests it leaves

- **Choice:**
  - 792b17 specifies how sources are found: named files, Pi directories, `--sweep` roots, the
    `--auto` roots with their exclusions and depths, the throwaway-path drop, the sweep
    configuration, the session-log directory, the walk and the source filter. How a source's read
    failure is handled, Rule B, deduplication and everything after are chunk-index-store's
    (792c7d5), so the tests of those (the `TestSweep*` tests, the `TestIngest*` dedup tests and
    `TestPiSessionsReadErrorPropagates`) wait for 792c7d5.
  - The c2 C2→S5 row is ported whole and points at 792b17, 792b15 and 792a5i. Its "byte-capped with
    continuation signalling" clause is stale (792d82) and is dropped. `ingest-auto-sweep` keeps only
    its zero-record residue for 792c7d5.
  - `TestIngestPrefersPhysicallyCloserAncestorAcrossTypes` checks the ancestor depth through
    deduplication, so 792c7d5 gives it a chunk-index-store scenario with 792b17c3a6 `split
    deferred`. `TestPiSessionsRespectsBudget` tests no budget: it checks that one Pi session is
    indexed, which is 792b17c1a2.
- **Rationale:** 792d30, and one owner per requirement.
- **Alternatives:** put the read-failure policy here, since it depends on whether a source was
  named. Rejected because the read and its failure happen in the index's reconciliation.

## 792d84 chunk-index-store scope, legacy removal and the tests it leaves

- **Choice:**
  - 792b18 specifies the index layout, the locked reconciliation, change detection, read and write
    failures, the append-only rebuild, grouping and precedence, copy eviction and its coverage gate,
    Rule B, the three prune modes, index listing and the chunk-ID set that `amend --chunk-source`
    checks (792a4e1a, 792a4f1a). The time a chunk was first indexed and its backfill wait for
    memory-query (792d78), and so does `show-chunk`'s lookup, which only memory-query's scenarios
    use; their tests (`TestIngestMarkdownSetsIngestedAtFromNow`,
    `TestIngestTranscriptSetsIngestedAtFromPerRowTimestamp`, `TestLoadPriorRecordsPreservesIngestedAt`,
    `TestMergeAppendBackfillsIngestedAtFromManifestMtime`, `TestMergeAppendPreservesIngestedAtOnReIngest`,
    `chunk/ingestedat_test`, `show_chunk_test` and the four `TestChunkQuery*` tests in
    `ingest_integration_test`) stay unrenamed.
  - Legacy removal: ADR-0021 keeps only decision 9 for 792c7j1, behind a pointer; the c2 ingest
    flowchart and its source list are deleted; c1 R4, c3 K1, K12 and their diagram nodes point at
    792b15–792b18; the c3 ingest data contract keeps its skill half and points at 792a5i for the
    binary half; the staleness sentence of c2's cross-level note points at 792b18.
    `ingest-auto-sweep` and `ingest-prune-detach` are deleted; `ingest-chunk-dedup` keeps the
    `update` notice for 792c7j1; `vault-concurrency-safe-writes` keeps only the vocabulary lock
    requirement for 792c7e3. GLOSSARY and ROADMAP citations of the dedup parts of ADR-0021 point at
    792b18.
  - Invariants M1–M3 (section E) are deleted, not ported: no marker exists, and no byte budget
    applies to ingestion (792d82), so M3's "one source cannot starve another" has nothing to enforce.
  - `TestRawForRebuild` covers reuse, a fresh read, a tolerated vanished swept source and a failing
    named one, so it takes 792b18c4a1 with 792b18c7a6 `split deferred`;
    `TestIngestPrefersPhysicallyCloserAncestorAcrossTypes` takes 792b18c6a5 with 792b17c3a6 `split
    deferred` (792d83). `TestIngestDepsReadTranscriptReadsViaFS` checks only that the composed reader
    reads through the injected filesystem, so it verifies no scenario.
- **Rationale:** 792d30, 792d51 and 792d6.
- **Alternatives:** specify the first-indexed time here as a **MAY**. Rejected because memory-query's
  recency channel makes it a contract there.

## 792d85 Mixed sources of controlled-vocabulary are split by owner

- **Choice:** node 792c7e1 ports the product behavior of its sources into 792a6 (792d31, 792d50):
  - `vault-vocab-lifecycle` keeps a residue requirement with the mechanism clauses (definition
    discrimination, the assignment function, the centroid preference, the trigger function, the
    slug parse, the derivation and the centroids-file format) and their function-named scenarios for
    vocabulary-engine (792c7e3). Its supersession ride-along requirement is query behavior, not
    vocabulary behavior, so it is re-deferred to memory-query (792c7f1), whose sources now list it.
  - `vault-note-identity`'s CRLF reader clause for vocabulary tagging, removal and refit and its
    "Vocab tagging reads a CRLF note" scenario are ported (792a6f).
  - ADR-0012's `vocab stats` QA lines and the c3 QA capture residue's `vocab stats` bullet are ported
    (792a6t); both now point at it.
  - ADR-0019 is ported whole as 792a6c's rationale, including route's evidence and aggregate notes
    and the drowning remedies, and deleted. Its citations in GLOSSARY, ROADMAP, ADR-0011 and the
    `route-dispatch-evidence` spec point at 792a6c; route SKILL.md's three citations wait for
    792c7p1, as its catalog says (792d40).
  - The query payload's `refit_pending` field stays with memory-query, and acting on a pending
    refit with the learn skill (792d32).
- **Rationale:** each part has one owner, and a node removes only what it ports.
- **Alternatives:** port the ride-along into 792a6. Rejected because it describes how `engram query`
  orders its results.

## 792d86 Controlled-vocabulary requirements beyond the legacy text, and their components

- **Choice:**
  - 792a6 adds requirements that the legacy sources don't state but the code, `--help` and tests
    show, each confirmed against the built binary on a temporary vault: the definition and family
    note shapes and the version bumps (792a6a, 792a6b), the skip cases (792a6e), bootstrap's output,
    floor and failures (792a6g, 792a6h), propose's output and its missing-family-note warning
    (792a6i), the dry-run, naming-request and naming-answer formats and errors (792a6l–792a6n),
    the applied-refit line (792a6o), the refusal of `--plan` and `--emit-request` (792a6r), the
    baseline seeding of the trigger (792a6s), every `vocab stats` line (792a6t), the backfill's skip
    lines (792a6u), and `engram update`'s two vocabulary notices and `--regen-vocab` (792a6v,
    792a6w).
  - note-amendment (`792b14`) is added to the node's extended components: `amend` and `resituate`
    tag the note they write. note-authoring gains 792b12c6 and note-amendment 792b14c11, the
    caller's ordering of tagging and the refit check after a write (792d63). command-line-surface,
    note-document-format, text-embedder and embedding-sidecar gain mappings only.
  - The dispatch tests in `vocab_commands_test` that assert a 792a6 scenario end to end
    (`TestTargets_Vocab*`) are renamed by vocabulary-engine (792c7e3), which examines that file; the
    learn and amend tests that check the skip cases (`TestRunLearn_VocabAssignment_Skips*`,
    `TestRunAmend_VocabAssignment_SkipsWhenSidecarMissing`) go there too, because the skip is the
    tagger's behavior.
- **Rationale:** 792d33, 792d49 and 792d58; 792d30 for the extensions.
- **Alternatives:** port only the OpenSpec text.

## 792d87 Known divergences in the controlled vocabulary

- **Choice:** 792a6 describes these as they are and does not specify the alternative (792d33,
  792d42):
  - `engram vocab propose` does not check for an existing definition: proposing a term twice writes
    a second `vocab-<term>-definition` note, and only the first one listed is used as the term.
  - Retirement scrubs links from note bodies and `supersedes:` lists but not from other frontmatter
    fields: a fact whose `object:` links a retired definition keeps that link, and `engram check`
    reports it as dangling. 792a6p1b is stated for body links.
  - The refit check after `engram learn` lists the vault with the starter `README.md`, so its note
    count is one higher than the baseline bootstrap and refit record (a vault with one fact recorded
    a baseline of 2 notes).
  - A bootstrap rewrites `vocab.centroids.json` without each term's origin, so a proposed term loses
    its `proposed` origin and the next refit can retire it (792a6q holds only until then).
  - `engram vocab <subcommand> --help` prints the `vocab` group's help, not the subcommand's flags.
- **Rationale:** a divergence becomes a later change proposal (vault note 645).
- **Alternatives:** specify the intended behavior with coverage gaps. Rejected because the code does
  not do it.

## 792d88 similarity-clustering scope and the tests it leaves

- **Choice:** 792b19 specifies cosine distance, the seeded k-means partition, the silhouette score
  and the choice of the number of clusters, which refit's scenarios need. The per-point silhouette
  (`PointSilhouette`), used only by the query to pick cluster representatives, waits for memory-query
  (792c7f1) with its four tests. `TestInvariant_C1_ClusteringDeterminism` checks determinism through
  `RunQuery` with a phrase-derived seed, and `TestProperty_Cluster_NearDuplicateL2CosineAtLeast095`
  checks a query payload's cosine, so both wait for memory-query too. Invariant C1 is ported as
  792b19c2's determinism clause and rationale; the query's own seeding from the phrase is the query
  engine's (792c7f5). c3 K8 points at 792b19 and keeps the query's bound on clustering inputs as a
  residue for memory-query, whose K6 row states the same caps.
- **Rationale:** 792d51: requirements arrive with the product scenarios they support.
- **Alternatives:** specify the per-point silhouette now as a **MAY**. Rejected because the query
  makes it a contract there.

## 792d89 vocabulary-engine scope, legacy removal and the tests it leaves

- **Choice:**
  - 792b20 specifies definition recognition, term vectors, term selection, the tag rewrite, tagging a
    written note, the refit check, statistics, definition and family notes with the version,
    bootstrap and its two-pass re-tag, propose, derivation, matching, refit routing, naming
    requests and answers, applying, retirement, the self-tag backfill, the old-format migration and
    the two vault-state checks behind `engram update`'s notices. The question-note helpers in
    `qa.go` and the two detectors in `update.go` are its realization because they answer
    vocabulary questions; how `engram update` runs them and renders their result is
    install-and-update's (792c7j1), so `update_test`'s `TestWriteUpdateReport_VocabMigrationHint`,
    `TestWriteUpdateReport_VocabRegenReport`, `TestWriteUpdateReport_VocabSelfTagHint` and
    `TestRunUpdate_RegenVocabFlag_RunsRegenAndUpdatesReport` wait for it.
  - The dispatch tests `TestTargets_Vocab*` take the product IDs of the scenarios they assert
    (792a6h1a, 792a6i1b, 792a6j1b, 792a6t1a; 792d52).
  - Legacy removal: ADR-0024 is ported as the rationale of 792a6j–792a6s and 792b20, and deleted; its
    "retirement semantics" consequence (demote and record a supersession) is stale since retirement
    deletes (792a6p) and is dropped. ADR-0011 keeps its history, with its assignment clause pointing
    at 792b20. The c2 vocab lifecycle flowchart becomes 792b20b; its query-payload step is
    memory-query's (`refit_pending`, now in that node's shipped artifacts) and its learn-skill step
    is learn's (792d32). `vault-concurrency-safe-writes` loses its last requirement (bootstrap and
    refit hold the lock: 792b20c10, 792b20c15) and is deleted; `vault-vocab-lifecycle` keeps only the
    ride-along residue for 792c7f1 (792d85).
  - Remaining tests in its files verify no 792b20 scenario: `TestNewVocabDeps_ClosuresCalled`
    (composition wiring), `TestIdAndDateFromNoteFilename` (a filename helper used by reparenting,
    registration and update, left to vault-migration 792c7i1), and
    `TestVocabDefinitionNote_FloorPromoted` and `TestVocabDefinitionNote_ReturnedWhenOnlyItem` (query
    behavior, left to memory-query 792c7f1). The vocabulary tests in `query_vault_meta_test`,
    `skillfields_survival_test` and `exchangefields_survival_test` belong to memory-query,
    skill-runbook-registration and parent-sync.
- **Divergences noted (792d33):** `engram update --regen-vocab` rewrites and deletes vault files
  without taking the vault lock, unlike every other vocabulary writer; `engram vocab
  tag-definitions` lists the vault before taking the lock.
- **Rationale:** 792d30, 792d51 and 792d63.
- **Alternatives:** split the statistics or the migration into their own components. Rejected
  because each is one command over the same definition and tag contracts, with no contract of its own
  that a caller could use apart from the vocabulary.

## 792d90 Mixed sources of memory-query are split by owner

- **Choice:** node 792c7f1 ports the query behavior of its sources into 792a7 and leaves the rest
  where its owner will find it (792d31, 792d50):
  - `recall-two-channel-payload`, `recall-matched-note-floor`, `recall-centroid-sampling`,
    `recall-query-timings`, `recall-runbook-surfacing` and `vault-query-model-provenance` are ported
    whole and deleted. Their technical clauses were all behavior the product spec already states, so
    nothing was left for recency-ranking (792c7f2) or query-payload-renderer (792c7f4) to port; those
    components are written from the code and from their other listed sources.
  - `recall-payload-cuts` keeps only the requirement that the agent reads chunk content before judging
    a cluster, for recall (792c7m1). Its lazy-chunks, recent-fill and limit requirements are 792a7s,
    792a7o and 792a7q.
  - `runbook-lexical-triggers` keeps three residue requirements: the served `--text` round trip and
    its 2 KB truncation for vault-serving (792c7s1), the shim's verbatim `--text` instruction for
    ambient-guidance (792c7r1), and the authoring guidance for trigger cues for write-memory
    (792c7l1). Their sources now list them. The `triggers` field and its flags were already note-writing's
    (792a3d, 792a3g) and note-revision's (792a4b).
  - `vault-note-identity` loses the query clauses of its CRLF reader requirement (792a7zg); the
    check and count clauses remain, as a residue, for vault-health (792c7h1) and vault-counting
    (792c7g1).
  - `vault-vocab-lifecycle` is deleted: its supersession ride-along was its last requirement (792a7n).
  - ADR-0026's retrieval decision is 792a7v. Its lexical-trigger amendment stays for
    runbook-trigger-matching (792c7f3), and ADR-0025 for query-engine (792c7f5).
- **Rationale:** each part has one owner, and a node removes only what it ports.
- **Alternatives:** keep `recall-two-channel-payload` and `recall-query-timings` for the technical
  nodes. Rejected because the technical contract they imply (a recency channel with a limit, five
  timing stages) is the same behavior the product requirements already state, so a technical node would
  have nothing left to port from them.

## 792d91 Memory-query requirements beyond the legacy text, and the components extended

- **Choice:**
  - 792a7 adds requirements that the legacy sources do not state but the code, `--help` and the
    built binary show, each confirmed on a temporary vault: the query input rule (792a7a), the payload
    shape and budget fields (792a7b), the dropped-sidecar warnings and the no-embeddings failure
    (792a7c), the per-phrase ranking, floor and cap (792a7d), recency scaling (792a7f), clusters and
    candidate notes (792a7g, 792a7h), the content budget (792a7r), `engram show-chunk` (792a7t), the
    project filter (792a7u), `engram show` (792a7x), `query-chunks` (792a7zf), the read-only rule
    (792a7zh), and the first-indexed time of a chunk (792a7p).
  - Component sections extended by this node: command-line-surface gains 792b2c7 (note lookup) and
    792b2c8 (chunk lookup), the routing requirements 792d51 deferred to it; wikilink-graph gains 792b13c3
    (link targets), the parsing contract 792d75 deferred to it; chunk-index-store gains 792b18c14
    (first-indexed time); transcript-reader gains 792b15c6 (last timestamp), which was not on this
    node's list but is the other half of the first-indexed time; similarity-clustering gains 792b19c5
    (per-point silhouette, 792d88). text-embedder, embedding-sidecar and vocabulary-engine gain
    mappings only.
  - Tests renamed here: the `RunShow` and `RunShowChunk` tests, the `ParseWikilinks` tests, the
    first-indexed-time tests of `ingest_test` and `chunk/ingestedat_test`, the transcript timestamp
    tests and the `PointSilhouette` tests. The tests that exercise `RunQuery` and the helpers behind it
    wait for query-engine (792c7f5), which also takes `ingest_integration_test`'s `TestChunkQuery*`
    (they test `query-chunks`), `qa_test`'s `TestIsQueryExcludedKind`, `vocab_test`'s two
    `TestVocabDefinitionNote_*` tests, `supersedes_test`'s `TestBuildSupersedesInverse`,
    `embed_selection_test`'s `TestKindFromContent`, the six query reader tests of
    `crlf_readers_internal_test`, and the two query-level clustering tests (792d88).
- **Rationale:** 792d33, 792d49, 792d58 and 792d86; 792d30 for the extensions.
- **Alternatives:** port only the OpenSpec text.

## 792d92 Known divergences in memory-query

- **Choice:** 792a7 describes these as they are and does not specify the alternative (792d33,
  792d42):
  - `--recent-fill 0` means the default of 25, not "off". The OpenSpec scenario "Recent-fill zero
    disables channel" does not match the code; only a negative value turns the channel off, and the
    flag takes a negative number only in the form `--recent-fill=-1`.
  - Recency scaling is live. 792d42 listed it as inactive (#774). The built binary shows it scaling a
    chunk first indexed 120 days ago to a quarter of its cosine, and a note whose sidecar has
    `last_used` 120 days ago to a quarter. What #774 describes is narrower: a note that has never been
    used is not scaled, because `engram learn` quotes `created`, which the date parser rejects, so
    about 972 of the 1,006 notes in the real vault are unscaled until used. 792a7f states both.
  - `--project` filters only the matched note items. The explore, ride-along, chunk and recent items,
    the clusters, and their candidates are not filtered.
  - A runbook's `red_flags` are capped in the preview only when it is a direct match, a candidate note
    or a trigger hit. An explore or ride-along item carries the whole list, with no marker (verified on
    a runbook of 30 flags). The OpenSpec requirement says "in `items[]` or `candidate_l2s`".
  - `engram show` lists `[[target|display]]` as `target|display`; `engram show` resolves that form.
  - `--limit` counts every item that is neither a trigger hit nor a recent chunk, so the explore notes,
    ride-alongs and matched chunks use it as well as the matched notes. The OpenSpec calls these
    "Channel 1" items.
  - The C4 views say superseding notes are added to `candidate_l2s`; in the code they are inserted in
    `items` and `candidate_l2s` holds only the cluster's own notes.
- **Rationale:** a divergence becomes a later change proposal (vault note 645).
- **Alternatives:** specify the intended behavior with coverage gaps. Rejected because the code does
  not do it.

## 792d93 recency-ranking scope and the tests it leaves

- **Choice:**
  - 792b21 specifies the recency factor, a note's age, the creation date read from a note, turn
    positions and the newest chunks. How the factor is applied to note and chunk scores, and the age of
    a chunk, are the query-engine's (792c7f5), so `RunQuery`'s three recency tests wait there:
    `TestRunQuery_NoteRecencyDecayRanksFreshLastUsedFirst`,
    `TestRunQuery_RecencyLiftsRecentChunkOverStaleHighCosine` and
    `TestCluster_NewerChunkRanksAboveOlderWithSameVector`.
  - `fillRecencyBand`, `spliceRecent` and `mostRecentlyUsedNoteItems` have no caller outside tests, so
    they are not specified, as with the unwired parts of the transcript reader (792d81). Their tests,
    and the `TestSortScoredDescOrdersDescending` test of a sort that production code does not call, are
    listed as no-scenario tests. So is `TestDefaultRecencyParamsSaneDefaults`, which checks a `floor`
    value that only those helpers read, and `TestRecencyEvalDiscriminatingHalfLife`, an in-suite
    experiment that sweeps half-lives and logs the ranks.
  - The six query reader tests of `crlf_readers_internal_test` (792d91) go to the component each tests,
    not all to query-engine: `TestParseCreatedFromNote_CRLFNote` is 792b21c3a6;
    `TestTriggerIndex_CRLFRunbook` goes to runbook-trigger-matching (792c7f3) and
    `TestCapRedFlagsForPreview_CRLFNote` to query-payload-renderer (792c7f4).
  - 792a7 gains the scenarios 792a7f1e (a late turn is lifted) and 792a7zg1d to 792a7zg1g (the
    recency date, red-flag preview, kind and frontmatter lists of a CRLF note), which the component
    tests above verify.
- **Rationale:** 792d51 and 792d52: a test goes to the component whose contract it checks.
- **Alternatives:** rename the band helpers' tests to a MAY requirement. Rejected because no production
  path uses them.

## 792d94 runbook-trigger-matching scope and the tests it leaves

- **Choice:**
  - 792b22 specifies the trigger index, the whole-word occurrence rule, the list of hits, their
    placement, the trigger rank, the lookup by message alone and the absence of a message. Two
    interaction rules belong to the callers (792c3c): a hit's exemption from the relevance floor, the
    matched-set cap and the matched-note floor belongs to query-engine (792c7f5), which adds hits after
    matching, and its exemption from `--limit` to query-payload-renderer (792c7f4), which splits hits
    off before the cap.
  - The `RunQuery` tests that check the trigger path take technical IDs of 792b22 (792d52), because each
    asserts the contract of this component through the whole command; the product scenarios they
    support are in the mapping table. Three tests of `query_runbook_test` go elsewhere:
    `TestRunQuery_TriggerHitDoesNotConsumeLimit`, `TestRunQuery_OversizedRedFlagsListKeepsNewestEntry`
    and `TestRunQuery_RunbookItemContentIncludesRedFlags` to query-payload-renderer (792c7f4), and
    `TestRunQuery_NoPhrasesAndNoTextIsRejected` and `TestRunQuery_RunbookCompetesInMainMatchedSet` to
    query-engine (792c7f5).
  - ADR-0026's 2026-09-20 amendment is replaced by a pointer to 792b22 and 792a7za. The served `--text`
    clause stays in `runbook-lexical-triggers` for vault-serving (792d90).
- **Rationale:** 792c3c and 792d52.
- **Alternatives:** none apply.

## 792d95 query-payload-renderer scope and the tests it leaves

- **Choice:**
  - 792b23 specifies the document and its keys, item labels, the order and the limit, the chunk content
    budget, lazy chunks, the budget block, the stage timings and the red-flag preview. The contents of
    the clusters and of `candidate_l2s`, the default limit of 20 and the default recent-fill count are
    the query-engine's (792c7f5): it decides what is in the payload and resolves the flags before the
    renderer sees them. `TestRunQuery_DefaultLimitCapsLargeResultSet` and
    `TestResolveRecentFill_DefaultsAndOverrides` go there, although they sit in the files this node
    examined.
  - The keys that lead a payload with pending offers, and `from_parent` in a merged payload, are
    curate's and parent-sync's, so their tests wait for those nodes (792d51):
    `TestQueryPayload_PendingOffersHintOmittedWhenFalse`, `TestQueryPayload_PendingOffersHintPresentWhenTrue`,
    `TestQueryPayload_PendingOffersLeadPayload` and `TestPendingOfferCurateInstruction_IsExpectedUpkeep`
    to curate (792c7u1), and `TestQueryPayload_NoFromParentKeyWithoutMerge` to parent-sync (792c7t1).
  - Timings are split as the code splits them: the query-engine takes the readings that end the scan,
    embed, cluster and nominate stages and 792b23c7 owns the block and the final reading, so
    `TestRunQuery_TimingsFlag` is renamed here and exercises the whole run.
  - Tests that run `RunQuery` and assert a renderer requirement take technical IDs (792d52), as in 792d94.
  - The renderer takes `kindFromContent`, so `embed_selection_test`'s `TestKindFromContent` and the CRLF
    kind test are renamed here (792d91, 792d93).
- **Rationale:** 792c3c, 792d51 and 792d52.
- **Alternatives:** keep the cluster content in the renderer. Rejected because the clustering and the
  candidate ranking are the query-engine's choices, and the renderer only prints them.
