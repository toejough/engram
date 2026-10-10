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
  archive for rationale, and node [792c8c OpenSpec archive and tooling removal](./792c_implementation-plan.md#792c8c-openspec-archive-and-tooling-removal) deletes it, because the lifecycle guide forbids keeping a
  completed-change archive. Removing the OpenSpec tooling is deferred to [792e3 Should the OpenSpec tooling be removed?](./792e_open-questions.md#792e3-should-the-openspec-tooling-be-removed). Vault note 913
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

- **Choice:** [792a14 recall](./792a_product-spec.md#792a14-added-recall), [792a15 learn](./792a_product-spec.md#792a15-added-learn), [792a13 write-memory](./792a_product-spec.md#792a13-added-write-memory), [792a18 please](./792a_product-spec.md#792a18-added-please), [792a17 route](./792a_product-spec.md#792a17-added-route) and [792a22 curate](./792a_product-spec.md#792a22-added-curate) get product capabilities but no
  technical component sections.
- **Rationale:** they meet the responsibility, ownership, stability and necessity criteria, but not
  independent verification. The technical guide requires a direct, deterministic contract test for
  every canonical technical requirement. Skill behavior is checked only by markdown baselines and
  headless evals, which aren't contract tests. The technical constraints they share live in
  [792b32 agent-instruction-bundle](./792b_technical-spec.md#792b32-added-agent-instruction-bundle) ([792d5 agent-instruction-bundle qualifies as a component](#792d5-agent-instruction-bundle-qualifies-as-a-component)).
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
  `ENGRAM_SERVER`, which the legacy docs explicitly reject, becomes a **MUST NOT** in [792a21 parent-sync](./792a_product-spec.md#792a21-added-parent-sync).
- **Rationale:** the lifecycle guide keeps only current behavior in canonical specs, and keeps
  absences only when they are enduring contracts. ROADMAP Provenance already records the dead ends.
- **Alternatives:** keep a "retired" section in each spec. Rejected because git is the fallback.

## 792d7 Rationale lives beside the requirement it justifies

- **Choice:** ADR, design and archive reasoning becomes a short `Rationale:` paragraph under the
  requirement or orientation it justifies.
- **Rationale:** Joe asked for this, and it keeps each decision next to the rule it explains.
- **Alternatives:** a separate decision document. Rejected because Joe asked for rationale in the
  relevant sections. The tension with the lifecycle guide's "concise context only" rule for canonical
  specs is left open as [792e4 Do rationale paragraphs survive promotion?](./792e_open-questions.md#792e4-do-rationale-paragraphs-survive-promotion), and doesn't block anything.

## 792d8 Per-component technical structure

- **Choice:** each component section has orientation, sequences, behavior requirements with Gherkin,
  and product-scenario mapping ([792c3b Technical Component Section](./792c_implementation-plan.md#792c3b-technical-component-section)). The option-analysis, impact and alternatives sections are
  omitted, and [792b1 Orientation](./792b_technical-spec.md#792b1-orientation) states why once.
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
  no matching issue, so the key is open question [792e2 Which issue key roots the parallel memory agent change?](./792e_open-questions.md#792e2-which-issue-key-roots-the-parallel-memory-agent-change). No issue was filed.
- **Rationale:** the lifecycle guide puts unbuilt behavior in an active change, never in canonical
  specs.
- **Alternatives:** root it under #792. Rejected because the feature is unrelated to this migration.

## 792d15 Capability granularity: 22 capabilities

- **Choice:**
  - Capabilities are grouped by what a user does or gets: one per shipped skill, and one per
    distinct CLI goal.
  - In plan order, the 22 are: [792a2 vault-identity](./792a_product-spec.md#792a2-added-vault-identity), [792a3 note-writing](./792a_product-spec.md#792a3-added-note-writing), [792a4 note-revision](./792a_product-spec.md#792a4-added-note-revision), [792a5 chunk-indexing](./792a_product-spec.md#792a5-added-chunk-indexing),
    [792a6 controlled-vocabulary](./792a_product-spec.md#792a6-added-controlled-vocabulary), [792a7 memory-query](./792a_product-spec.md#792a7-added-memory-query), [792a8 vault-counting](./792a_product-spec.md#792a8-added-vault-counting), [792a9 vault-health](./792a_product-spec.md#792a9-added-vault-health), [792a10 vault-migration](./792a_product-spec.md#792a10-added-vault-migration),
    [792a11 install-and-update](./792a_product-spec.md#792a11-added-install-and-update), [792a12 skill-runbook-registration](./792a_product-spec.md#792a12-added-skill-runbook-registration), [792a13 write-memory](./792a_product-spec.md#792a13-added-write-memory), [792a14 recall](./792a_product-spec.md#792a14-added-recall), [792a15 learn](./792a_product-spec.md#792a15-added-learn), [792a16 lessons-contract](./792a_product-spec.md#792a16-added-lessons-contract),
    [792a17 route](./792a_product-spec.md#792a17-added-route), [792a18 please](./792a_product-spec.md#792a18-added-please), [792a19 ambient-guidance](./792a_product-spec.md#792a19-added-ambient-guidance), [792a20 vault-serving](./792a_product-spec.md#792a20-added-vault-serving), [792a21 parent-sync](./792a_product-spec.md#792a21-added-parent-sync), [792a22 curate](./792a_product-spec.md#792a22-added-curate), and
    [792a23 memory-effectiveness-evaluation](./792a_product-spec.md#792a23-added-memory-effectiveness-evaluation).
- **Rationale:** the three discovery reports proposed 27, 12 and about 11 capabilities.
  - The 27-capability list split single surfaces apart, for example note provenance and QA from
    note writing.
  - The 12- and 11-capability lists merged unrelated goals, for example `check`, `count` and `embed`
    into one "vault-health", and please and route into one "orchestration".
  - Grouping by user goal keeps each file stable when the commands inside it change.
- **Alternatives:** the 27-, 12- and 11-capability inventories in `792f_discovery/`.

## 792d16 Capability names

- **Choice:** names use Joe's vocabulary from the GLOSSARY and commit scopes. Each skill capability
  is named after its skill. [792a7 memory-query](./792a_product-spec.md#792a7-added-memory-query) is the CLI retrieval surface and [792a14 recall](./792a_product-spec.md#792a14-added-recall) is the skill.
  [792a5 chunk-indexing](./792a_product-spec.md#792a5-added-chunk-indexing) is used over `transcript-ingest`, because ingest also covers markdown.
  [792a21 parent-sync](./792a_product-spec.md#792a21-added-parent-sync) follows "local-first parent sync". [792a11 install-and-update](./792a_product-spec.md#792a11-added-install-and-update) is used over `engram-update`.
- **Rationale:** avoid coined jargon (vault note 291) and the retired names (episode, MOC, nomination,
  promote, `ENGRAM_SERVER`).
- **Alternatives:** `memory-recall` for the query surface. Rejected because it collides with the
  recall skill. Also `session-ingestion` and `vault-sharing`.

## 792d17 Runbooks are spread across their surfaces

- **Choice:** there is no `runbook-memory` capability. Runbook capture is in [792a3 note-writing](./792a_product-spec.md#792a3-added-note-writing), trigger
  surfacing is in [792a7 memory-query](./792a_product-spec.md#792a7-added-memory-query), and the follow-frame is in [792a19 ambient-guidance](./792a_product-spec.md#792a19-added-ambient-guidance).
- **Rationale:** a runbook is a note kind. Each of these behaviors already belongs to a surface that
  owns the same behavior for the other kinds.
- **Alternatives:** a runbook-memory capability, as legacy P6. Rejected because it would cut across
  three surfaces.

## 792d18 The QA kind is documented as current

- **Choice:** `engram learn qa` capture belongs to [792a3 note-writing](./792a_product-spec.md#792a3-added-note-writing), and ad hoc QA capture to [792a15 learn](./792a_product-spec.md#792a15-added-learn). Both
  are documented as current, even though #727 and #765 are reconsidering the kind.
- **Rationale:** the kind ships today. A reconsideration is a future change, not current behavior.
- **Alternatives:** a separate `question-answer-memory` capability. Rejected as one surface split in
  two.

## 792d19 One ambient-guidance capability

- **Choice:** recall.md, learn.md, delegate.md and shim.md, together with their activation, form one
  [792a19 ambient-guidance](./792a_product-spec.md#792a19-added-ambient-guidance) capability.
- **Rationale:** they share one opt-in deployment (`--with-guidance`) and one activation mechanism.
  #773 is reconciling their overlapping cues, and separate files would freeze that duplication.
- **Alternatives:** four capabilities named after the AGENTS.md labels.

## 792d20 `lessons-contract` is its own capability

- **Choice:** the LESSONS line contract has one capability. The OpenSpec `learn`, `please` and
  `route` specs map to it, because their content is entirely the LESSONS slice.
- **Rationale:** a single owner prevents duplicating the rule across [792a15 learn](./792a_product-spec.md#792a15-added-learn), [792a18 please](./792a_product-spec.md#792a18-added-please), [792a17 route](./792a_product-spec.md#792a17-added-route) and the
  guidance capability.
- **Alternatives:** put the requirements in each skill's capability.

## 792d21 Vault identity comes first; `ENGRAM_SERVER` belongs to parent-sync

- **Choice:** [792a2 vault-identity](./792a_product-spec.md#792a2-added-vault-identity) is the first capability. It covers path resolution, first-use
  creation, the vault ID, and the copy/clone and self-parent refusals. The `ENGRAM_SERVER` **MUST
  NOT** sits in [792a21 parent-sync](./792a_product-spec.md#792a21-added-parent-sync).
- **Rationale:** every command depends on resolving the vault. `ENGRAM_SERVER`'s replacement is
  `ENGRAM_PARENT`.
- **Alternatives:** fold vault identity into [792a21 parent-sync](./792a_product-spec.md#792a21-added-parent-sync), as code-tests discovery proposed. Rejected
  because first-use creation affects every user, not only users of the exchange.

## 792d22 Counting is separate from health

- **Choice:** [792a8 vault-counting](./792a_product-spec.md#792a8-added-vault-counting) (`count`) is separate from [792a9 vault-health](./792a_product-spec.md#792a9-added-vault-health) (`check`, `embed`).
- **Rationale:** ADR-0018 makes counting a distinct surface, and route depends on it as an evidence
  audit.
- **Alternatives:** a merged [792a9 vault-health](./792a_product-spec.md#792a9-added-vault-health).

## 792d23 A vault-migration capability

- **Choice:** `update --reparent-luhmann/--answers`, `--backfill-identity`, and the flat-vault notice
  form [792a10 vault-migration](./792a_product-spec.md#792a10-added-vault-migration).
- **Rationale:** they are one-shot structural maintenance of an existing vault, and share neither
  trigger nor contract with installation.
- **Alternatives:** fold them into [792a11 install-and-update](./792a_product-spec.md#792a11-added-install-and-update).

## 792d24 The vault exchange is split three ways

- **Choice:** [792a20 vault-serving](./792a_product-spec.md#792a20-added-vault-serving) (`engram serve`), [792a21 parent-sync](./792a_product-spec.md#792a21-added-parent-sync) (child side), and [792a22 curate](./792a_product-spec.md#792a22-added-curate) (the pending-offer
  review skill).
- **Rationale:** each has a different user and entry point: the host operator, the child vault user,
  and the curating agent.
- **Alternatives:** one `vault-federation` capability.

## 792d25 Eval tooling is a provisional last capability

- **Choice:** [792a23 memory-effectiveness-evaluation](./792a_product-spec.md#792a23-added-memory-effectiveness-evaluation) and its components [792b38 eval-trial-harness](./792b_technical-spec.md#792b38-added-eval-trial-harness) and
  [792b39 memory-loop-auditor](./792b_technical-spec.md#792b39-added-memory-loop-auditor) come last, pending [792e1 Is eval and audit tooling a product capability?](./792e_open-questions.md#792e1-is-eval-and-audit-tooling-a-product-capability).
- **Rationale:** five OpenSpec specs and ADR-0028 must go somewhere for the deprecated docs to empty.
  Placing these last means a "no" from Joe removes the group without reordering anything.
- **Alternatives:** delete them as dev tooling. Vault note 293 says not to spec dev tooling as
  product, but ADR-0028 D5 calls the audit product.

## 792d26 Component inventory: 38 components

- **Choice:** the code-tests discovery list of 34 components is the base, because it is grounded in
  code and tests. It is adjusted as follows:
  - `installer` is split into [792b28 self-installer](./792b_technical-spec.md#792b28-added-self-installer) (ADR-0023) and [792b29 harness-deploy-sync](./792b_technical-spec.md#792b29-added-harness-deploy-sync) (ADR-0022).
  - `vault-bootstrap` is folded into [792b5 vault-identity-registry](./792b_technical-spec.md#792b5-added-vault-identity-registry).
  - `recall-usage-tracking` (activate) is folded into [792b14 note-amendment](./792b_technical-spec.md#792b14-added-note-amendment).
  - Note lookup (`show`) becomes caller routing in [792b2 command-line-surface](./792b_technical-spec.md#792b2-added-command-line-surface).
  - `vault-statistics` is kept as [792b25 vault-counter](./792b_technical-spec.md#792b25-added-vault-counter).
  - [792b32 agent-instruction-bundle](./792b_technical-spec.md#792b32-added-agent-instruction-bundle) is added ([792d5 agent-instruction-bundle qualifies as a component](#792d5-agent-instruction-bundle-qualifies-as-a-component)), and so are the two eval components ([792d25 Eval tooling is a provisional last capability](#792d25-eval-tooling-is-a-provisional-last-capability)).
- **Rationale:** each kept component meets all five qualification criteria. Each split follows an
  ADR boundary, and each fold removes a contract too thin to stand alone.
- **Alternatives:** the legacy report's 32 components, which include the skill procedures ([792d4 Skill procedures are not technical components](#792d4-skill-procedures-are-not-technical-components)),
  and the history report's 14 package-level groupings, which merge independent contracts.

## 792d27 Excluded component candidates

- **Choice:** these candidates are excluded:
  - `debug-log` fails necessity: no production code emits a log line.
  - `interrupt-handling` fails necessity: no product requirement mentions interruption.
  - The composition root (`cli.Primitives`, `cli.NewDeps`, `cmd/engram`) is a dependency-injection
    seam, which the guide says can't qualify on that basis alone.
  - Their tests are listed as no-scenario tests in the [792b2 command-line-surface](./792b_technical-spec.md#792b2-added-command-line-surface) node. `debug-log`'s
    legacy text (c3 K11) is deleted in node [792c8a Residual legacy sweep](./792c_implementation-plan.md#792c8a-residual-legacy-sweep).
- **Rationale:** the guide's necessity rule, and its rule that a mockable seam does not qualify
  solely because it can be injected.
- **Alternatives:** keep them as MAY-level components.

## 792d28 Component names

- **Choice:** names describe each contract, with no package or function names. For example,
  [792b24 query-engine](./792b_technical-spec.md#792b24-added-query-engine), not `semantic-retrieval-engine`; [792b23 query-payload-renderer](./792b_technical-spec.md#792b23-added-query-payload-renderer), not
  `recall-payload-renderer`, because recall is the skill; [792b20 vocabulary-engine](./792b_technical-spec.md#792b20-added-vocabulary-engine); [792b10 embedding-sidecar](./792b_technical-spec.md#792b10-added-embedding-sidecar);
  [792b11 note-provenance](./792b_technical-spec.md#792b11-added-note-provenance); [792b34 pending-offer-bookkeeping](./792b_technical-spec.md#792b34-added-pending-offer-bookkeeping).
- **Rationale:** the guide's stability criterion, plus Joe's vocabulary ([792d16 Capability names](#792d16-capability-names)).
- **Alternatives:** the discovery reports' names.

## 792d29 Order and topology

- **Choice:** the order is foundational first: vault, notes, chunks, vocabulary, query, maintenance,
  deployment, skills, guidance, exchange, evaluation. Within a capability, callees come before
  callers. The nodes run as a linear chain, except that [792c8b In-flight design relocation](./792c_implementation-plan.md#792c8b-in-flight-design-relocation) and [792c8c OpenSpec archive and tooling removal](./792c_implementation-plan.md#792c8c-openspec-archive-and-tooling-removal) may run in parallel.
- **Rationale:** each component is migrated in the first capability that needs it, so foundational
  components land first and later capabilities only extend them. Every node edits the same [792a Documentation Framework Adoption - Product Spec](./792a_product-spec.md#documentation-framework-adoption---product-spec),
  [792b Documentation Framework Adoption - Technical Spec](./792b_technical-spec.md#documentation-framework-adoption---technical-spec) and [792c Documentation Framework Adoption - Implementation Plan](./792c_implementation-plan.md#documentation-framework-adoption---implementation-plan) files, and Joe asked for capability-by-capability order. Both facts serialize the
  work.
- **Alternatives:** parallel worktrees per capability. Rejected because they would conflict on the
  shared change files.

## 792d30 Components are migrated once and extended later

- **Choice:** a component node writes the component's section in full. Later capability nodes add
  mapping rows and any newly sourced requirements to that section (step 6 of [792c4a Capability Node Workflow](./792c_implementation-plan.md#792c4a-capability-node-workflow)), and rename the
  tests that newly match.
- **Rationale:** this is Joe's instruction: migrate each component once, then add mappings.
- **Alternatives:** a separate node per capability-component pair. Rejected as 65+ extra nodes.

## 792d31 Each legacy item has one owning node

- **Choice:**
  - Every legacy file, section, ADR and OpenSpec spec is listed under exactly one node for each part
    it contains. Nodes remove only what they port, a file is deleted once it is empty, and node
    [792c8d Deprecated-directory removal and final stale-reference sweep](./792c_implementation-plan.md#792c8d-deprecated-directory-removal-and-final-stale-reference-sweep) removes whatever remains.
  - Coverage: 21 files under `docs/deprecated/`, 29 ADRs plus the "not made into ADRs" section, 53
    OpenSpec specs, and 42 archive entries. The discovery brief said there were 56 specs; there are
    53.
  - Archive entries are read-only rationale sources until node [792c8c OpenSpec archive and tooling removal](./792c_implementation-plan.md#792c8c-openspec-archive-and-tooling-removal) deletes them.
- **Rationale:** per-part ownership lets a mixed file (an ADR, a C4 view, a mixed spec) be emptied
  across several nodes without losing content.
- **Alternatives:** file-level ownership. Rejected because a node can't port content another node
  has already deleted.

## 792d32 Skill product specs are derived from the shipped SKILL.md

- **Choice:** where a legacy source doesn't fully describe a skill or guidance behavior, the product
  spec describes the shipped SKILL.md or guidance file.
- **Rationale:** most of the [792a14 recall](./792a_product-spec.md#792a14-added-recall), [792a15 learn](./792a_product-spec.md#792a15-added-learn), [792a18 please](./792a_product-spec.md#792a18-added-please), [792a17 route](./792a_product-spec.md#792a17-added-route) and [792a22 curate](./792a_product-spec.md#792a22-added-curate) behavior exists only in SKILL.md.
  The OpenSpec skill specs are delta-shaped slices (legacy report, Overlaps).
- **Alternatives:** document only what the legacy specs state. Rejected because it would leave the
  capabilities mostly empty.

## 792d33 Behavior is described, not validated

- **Choice:** nodes describe current behavior from the legacy docs, code and tests, without
  confirming each requirement with Joe. The product guide's per-requirement interview is replaced by
  this log and [792e Documentation Framework Adoption - Open Questions](./792e_open-questions.md#documentation-framework-adoption---open-questions).
- **Rationale:** Joe said "Do not attempt to validate correctness yet." Vault note 645: in a
  descriptive spec tree, a divergence found later becomes a change proposal.
- **Alternatives:** an interview for each requirement. Rejected because Joe deferred validation.

## 792d34 References are repointed to the change sections

- **Choice:** stale references are repointed to the [792a Documentation Framework Adoption - Product Spec](./792a_product-spec.md#documentation-framework-adoption---product-spec) and [792b Documentation Framework Adoption - Technical Spec](./792b_technical-spec.md#documentation-framework-adoption---technical-spec) section anchors, which exist
  now. At promotion, one grep for `792a_` and `792b_` moves them to `docs/specs/...`. The IDs are
  stable, so the anchors survive.
- **Rationale:** the canonical files won't exist until promotion ([792d44 Promotion is excluded from this plan](#792d44-promotion-is-excluded-from-this-plan)). Pointing at them now would
  break links in every commit until then.
- **Alternatives:** point at the future canonical paths.

## 792d35 Product scenarios are restated at their first technical mapping

- **Choice:** a product scenario is restated in full in [792b Documentation Framework Adoption - Technical Spec](./792b_technical-spec.md#documentation-framework-adoption---technical-spec) at the first place it is mapped, and
  referenced by ID afterwards.
- **Rationale:** the technical guide requires it. The rule against duplicating scenario text applies
  only to canonical technical specs, and promotion drops these restatements.
- **Alternatives:** IDs only. Rejected as a deviation from the guide.

## 792d36 ID templates and overflow

- **Choice:** the templates in [792c3a Product Capability Section](./792c_implementation-plan.md#792c3a-product-capability-section) and [792c3b Technical Component Section](./792c_implementation-plan.md#792c3b-technical-component-section) match the `docs/AGENTS.md` example. Capability
  sections are `## 792aN Added: <name>`, requirements `### 792aNx`, UAT groups `#### 792aNx1`, and
  scenarios `792aNx1a`. Components follow the same pattern with letter subsections a–d. More than 25
  letter siblings continue as `za`–`zy`. Section numbers are fixed in the catalog: [792a2 vault-identity](./792a_product-spec.md#792a2-added-vault-identity)–[792a23 memory-effectiveness-evaluation](./792a_product-spec.md#792a23-added-memory-effectiveness-evaluation) and
  [792b2 command-line-surface](./792b_technical-spec.md#792b2-added-command-line-surface)–[792b39 memory-loop-auditor](./792b_technical-spec.md#792b39-added-memory-loop-auditor).
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
  and the renames in [792c7b5 text-embedder tech spec](./792c_implementation-plan.md#792c7b5-text-embedder-tech-spec) and [792c7d2 transcript-reader tech spec](./792c_implementation-plan.md#792c7d2-transcript-reader-tech-spec) remove that prefix. The first node that renames a `TestT`
  test updates the pattern to match the new scenario-ID prefix, which keeps the old tests in
  baseline selection. Node [792c8e Build-tag classification (deferred)](./792c_implementation-plan.md#792c8e-build-tag-classification-deferred) confirms the final value.
- **Rationale:** without the update, those renames would silently empty the baseline selection. targ
  is a black box, so the final value is asked as [792e7 What should TARG_BASELINE_PATTERN become?](./792e_open-questions.md#792e7-what-should-targ_baseline_pattern-become).
- **Alternatives:** keep the `TestT` prefix on those tests. Rejected because it conflicts with
  [792d10 Test renaming](#792d10-test-renaming).

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
  repointed, in nodes [792c8c OpenSpec archive and tooling removal](./792c_implementation-plan.md#792c8c-openspec-archive-and-tooling-removal) and [792c8d Deprecated-directory removal and final stale-reference sweep](./792c_implementation-plan.md#792c8d-deprecated-directory-removal-and-final-stale-reference-sweep).
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
- **Rationale:** [792d6 Removed features are not documented; enduring absences become MUST NOT](#792d6-removed-features-are-not-documented-enduring-absences-become-must-not) and [792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated). A divergence becomes a later change proposal (vault note 645).
- **Alternatives:** fix the code during migration. Rejected as out of scope.

## 792d43 Build-tag classification is a separate, deferred node

- **Choice:** node [792c8e Build-tag classification (deferred)](./792c_implementation-plan.md#792c8e-build-tag-classification-deferred) classifies tests as `integration` or `network`. The migration nodes don't.
- **Rationale:** this was decided up front, and it keeps renames separate from build-tag edits.
  Discovery found about 70 candidate files, no `network` cases, and an existing `parity` tag.
- **Alternatives:** classify tests in each component node.

## 792d44 Promotion is excluded from this plan

- **Choice:** applying the deltas to `docs/specs/` and deleting the change directory happen after Joe
  validates this log and answers [792e Documentation Framework Adoption - Open Questions](./792e_open-questions.md#documentation-framework-adoption---open-questions).
- **Rationale:** Joe deferred validation of correctness, and promoting would make unvalidated text
  canonical.
- **Alternatives:** a promotion node at the end. Its timing is asked as [792e6 When are the specs promoted?](./792e_open-questions.md#792e6-when-are-the-specs-promoted).

## 792d45 `792f_discovery/` holds the working inventories

- **Choice:** `792f_discovery/{legacy-docs,code-tests,history-memory}.md` holds the three discovery
  reports that this plan reconciles. The directory is deleted with the change directory.
- **Rationale:** the session scratchpad that first held them was lost. Nodes cite their test maps and
  source maps.
- **Alternatives:** inline the reports into [792c Documentation Framework Adoption - Implementation Plan](./792c_implementation-plan.md#documentation-framework-adoption---implementation-plan). Rejected as duplicating working data in the plan.

## 792d46 First-use creation does not create `Permanent/`

- **Choice:** [792a2e First-use vault creation](./792a_product-spec.md#792a2e-first-use-vault-creation) describes the vault that first-use creation actually builds: `.obsidian/app.json`,
  `.gitignore`, the starter `README.md`, the vault ID and `.engram/`. The first known divergence in
  [792d42 Known divergences between docs and code](#792d42-known-divergences-between-docs-and-code) is corrected: no code creates `Permanent/`. Only the doc comment on `initializeVault`
  (`internal/cli/vault_init.go`) still names it. Root `README.md` "Vault location" also still says
  that only `engram learn` creates a missing vault and that other commands fail with "vault not
  found"; every vault-resolving command creates it now. Both stale texts are left for a later change
  ([792d41 Root README stays as orientation](#792d41-root-readme-stays-as-orientation), [792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated)).
- **Rationale:** discovery inferred the divergence from the comment. Reading `initializeVault` and
  `Test792b2c3a1_Targets_EveryVaultCommandCreatesMissingVault` shows the current behavior.
- **Alternatives:** fix the comment and README now. Rejected as outside a documentation-only node.

## 792d47 Exchange-state location belongs to vault-identity

- **Choice:** the OpenSpec requirement "Exchange state SHALL live outside the vault's tracked files"
  is ported to [792a2k Exchange state outside tracked files](./792a_product-spec.md#792a2k-exchange-state-outside-tracked-files), not left for parent-sync ([792c7t1 parent-sync product spec](./792c_implementation-plan.md#792c7t1-parent-sync-product-spec)).
- **Rationale:** `.engram/` is created with the vault and its ID, before any parent is configured,
  and [792b5 vault-identity-registry](./792b_technical-spec.md#792b5-added-vault-identity-registry) owns it. [792a21 parent-sync](./792a_product-spec.md#792a21-added-parent-sync) keeps what is written there (outbox, declines,
  parent cache).
- **Alternatives:** leave it with the exchange requirements for [792c7t1 parent-sync product spec](./792c_implementation-plan.md#792c7t1-parent-sync-product-spec).

## 792d48 The vault name is specified in vault-identity

- **Choice:** `--vault-name`/`ENGRAM_VAULT_NAME` and its default `personal` are requirement [792a2b Vault name](./792a_product-spec.md#792a2b-vault-name),
  as the [792c7a1 vault-identity product spec](./792c_implementation-plan.md#792c7a1-vault-identity-product-spec) catalog lists. [792a3 note-writing](./792a_product-spec.md#792a3-added-note-writing) and note-provenance ([792c7b1 note-writing product spec](./792c_implementation-plan.md#792c7b1-note-writing-product-spec), [792c7b7 note-provenance tech spec](./792c_implementation-plan.md#792c7b7-note-provenance-tech-spec)) map their
  `vault:` stamping to [792a2b Vault name](./792a_product-spec.md#792a2b-vault-name) rather than restating it.
- **Rationale:** the precedence mirrors the vault path's ([792a2a Vault location](./792a_product-spec.md#792a2a-vault-location)), and one owner avoids duplicating
  the rule.
- **Alternatives:** specify it only under [792a3 note-writing](./792a_product-spec.md#792a3-added-note-writing).

## 792d49 Product requirements beyond the OpenSpec text

- **Choice:** [792a2 vault-identity](./792a_product-spec.md#792a2-added-vault-identity) adds requirements that the legacy sources don't state but the code and tests
  show: the flat layout's exclusion of subdirectories ([792a2c Flat vault layout](./792a_product-spec.md#792a2c-flat-vault-layout)), the starter README ([792a2d Starter README](./792a_product-spec.md#792a2d-starter-readme)), repair
  of a half-created vault ([792a2g Interrupted creation is repaired](./792a_product-spec.md#792a2g-interrupted-creation-is-repaired)), corrupt-ID handling ([792a2j Corrupt vault ID](./792a_product-spec.md#792a2j-corrupt-vault-id)), the ignored, committed and non-git
  cases of the uncommitted-ID notice ([792a2i Uncommitted vault ID notice](./792a_product-spec.md#792a2i-uncommitted-vault-id-notice)), the `vault-id` outputs and flag errors
  ([792a2l Inspecting the vault ID](./792a_product-spec.md#792a2l-inspecting-the-vault-id)–[792a2o Regenerate and claim are exclusive](./792a_product-spec.md#792a2o-regenerate-and-claim-are-exclusive)), and the merged query's warning on a clone ([792a2p1b A cloned child's merged query still runs](./792a_product-spec.md#792a2p1-uat-scenarios)).
- **Rationale:** [792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated) describes current behavior from code and tests. Each addition is observable
  and already tested.
- **Alternatives:** port only the OpenSpec text.

## 792d50 Technical clauses of a mixed OpenSpec spec stay in place for their component node

- **Choice:** when a capability node ports a mixed OpenSpec spec, the technical clauses listed for a
  later component node stay in the spec file under a residue requirement that names the node. For
  `vault-local-first`, the exclusive-create-and-re-read, injected random source, location-record
  contents, and regenerate/claim file effects wait for [792c7a5 vault-identity-registry tech spec](./792c_implementation-plan.md#792c7a5-vault-identity-registry-tech-spec).
- **Rationale:** [792d31 Each legacy item has one owning node](#792d31-each-legacy-item-has-one-owning-node) per-part ownership. A node removes only what it ports.
- **Alternatives:** port the technical clauses early into [792a Documentation Framework Adoption - Product Spec](./792a_product-spec.md#documentation-framework-adoption---product-spec). Rejected because product specs
  exclude implementation detail.

## 792d51 command-line-surface routing requirements arrive with their scenarios

- **Choice:** [792b2 command-line-surface](./792b_technical-spec.md#792b2-added-command-line-surface) specifies only the routing that [792a2 vault-identity](./792a_product-spec.md#792a2-added-vault-identity)'s scenarios need: vault path and
  vault name resolution, first-use gating and result reporting. The `ENGRAM_SERVER` pre-dispatch
  guard, chunks-dir resolution and `show` local/parent routing are added to [792b2 command-line-surface](./792b_technical-spec.md#792b2-added-command-line-surface) by [792a21 parent-sync](./792a_product-spec.md#792a21-added-parent-sync)
  ([792c7t1 parent-sync product spec](./792c_implementation-plan.md#792c7t1-parent-sync-product-spec)), chunk-indexing ([792c7d1 chunk-indexing product spec](./792c_implementation-plan.md#792c7d1-chunk-indexing-product-spec)) and memory-query ([792c7f1 memory-query product spec](./792c_implementation-plan.md#792c7f1-memory-query-product-spec)), which introduce the product
  scenarios those requirements support. Their tests stay unrenamed until then and are listed as
  deferred no-scenario tests. The C2 container row's per-command detail is deleted rather than
  ported, because root `README.md` "Binary commands" and the later nodes' sources state the same
  facts; [792b2a Orientation](./792b_technical-spec.md#792b2a-orientation) keeps the command inventory.
- **Rationale:** the technical guide requires every requirement to name the product scenarios it
  supports, and those scenarios don't exist yet. [792d30 Components are migrated once and extended later](#792d30-components-are-migrated-once-and-extended-later) already has later capability nodes extend a
  migrated component.
- **Alternatives:** write the requirements now with no supported scenarios. Rejected as untraceable.

## 792d52 Test IDs: product scenario for end-to-end checks, technical scenario otherwise

- **Choice:** a test that runs a whole command through dispatch and asserts a product scenario's
  observable outcome takes the product scenario ID (for example
  `Test792a2e1a_Targets_QueryMissingVaultReturnsEmptyResult`). A test of one component's contract
  takes the technical scenario ID (for example `Test792b2c1a1_ResolveVault_FlagWins`).
- **Rationale:** [792c4b Component Node Workflow](./792c_implementation-plan.md#792c4b-component-node-workflow) step 4 renames tests that verify either spec's scenarios; this rule picks
  one ID deterministically when a test could be read as verifying both.
- **Alternatives:** always use the technical ID. Rejected because an end-to-end test is the direct
  evidence for its product scenario.

## 792d53 Test-rename reference updates skip historical records

- **Choice:** when a test is renamed, references are updated everywhere except the [792d37 Exemptions from the stale-reference sweep](#792d37-exemptions-from-the-stale-reference-sweep) exemption
  set (`dev/eval/raw/`, `dev/eval/audit/`, `dev/eval/LEDGER.md`, `dev/eval/cumulative/`),
  `openspec/changes/archive/`, and the `.superpowers/sdd/` briefs and progress notes, which are
  dated execution records of finished work.
- **Rationale:** rewriting a dated record falsifies what it recorded. The archive is deleted by
  [792c8c OpenSpec archive and tooling removal](./792c_implementation-plan.md#792c8c-openspec-archive-and-tooling-removal), and the `.superpowers/` plan material is handled under [792d12 Research, eval-process notes and plans are non-normative](#792d12-research-eval-process-notes-and-plans-are-non-normative).
- **Alternatives:** rewrite every occurrence. Rejected: the first such rename rewrote eval audit
  transcripts, which [792d37 Exemptions from the stale-reference sweep](#792d37-exemptions-from-the-stale-reference-sweep) exempts.

## 792d54 Node agents commit their own nodes

- **Choice:** the agent executing a capability group commits each node itself as the node
  completes (one commit per node, `docs(792): <node ID> <title>`), instead of the orchestrator
  committing (as [792c4b Component Node Workflow](./792c_implementation-plan.md#792c4b-component-node-workflow)'s last line originally said).
- **Rationale:** a group runs several nodes in sequence; per-node commits by the executor keep each
  status transition in the commit that establishes it (implementation-plan guide) without
  stalling between nodes.
- **Alternatives:** orchestrator commits after each node. Rejected: it forces one agent dispatch
  per node, roughly tripling the dispatches for no review benefit.

## 792d55 `learn --tier` is documented as a frontmatter field

- **Choice:** [792a3i Tier field](./792a_product-spec.md#792a3i-tier-field) specifies `--tier`: it writes `tier: L2` by default and writes `L1`, `L2` or
  `L3` as given, and [792a3g Argument validation](./792a_product-spec.md#792a3g-argument-validation) refuses any other value. This resolves the second known divergence in
  [792d42 Known divergences between docs and code](#792d42-known-divergences-between-docs-and-code): the flag survives the removal of tiers because it still has an observable effect.
- **Rationale:** [792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated) describes current behavior. The note's `tier:` field is visible to the user,
  even though no retrieval path reads it.
- **Alternatives:** leave `--tier` out as a retired feature ([792d6 Removed features are not documented; enduring absences become MUST NOT](#792d6-removed-features-are-not-documented-enduring-absences-become-must-not)). Rejected because the field is
  still written to every note.

## 792d56 Placement under a target that does not exist is permitted

- **Choice:** [792a3l Placement under an absent target](./792a_product-spec.md#792a3l-placement-under-an-absent-target) states, as **MAY**, that `engram learn --position continuation` or `sibling`
  can place a note under a `--target` that names no note: `--target 77` in a vault without note 77
  writes note `77a`. No scenario protects it.
- **Rationale:** the code does not check that the target exists, and no legacy source promises
  either outcome. Stating it as **MAY** describes the behavior without making it a contract. A
  change could later make it a refusal (vault note 645).
- **Alternatives:** a **MUST NOT** with a coverage gap. Rejected because it would describe behavior
  the code does not have.

## 792d57 Mixed sources of note-writing are split by owner

- **Choice:** node [792c7b1 note-writing product spec](./792c_implementation-plan.md#792c7b1-note-writing-product-spec) ports only the [792a3 note-writing](./792a_product-spec.md#792a3-added-note-writing) parts of its mixed sources, and leaves the
  rest in place for the node that owns it ([792d31 Each legacy item has one owning node](#792d31-each-legacy-item-has-one-owning-node), [792d50 Technical clauses of a mixed OpenSpec spec stay in place for their component node](#792d50-technical-clauses-of-a-mixed-openspec-spec-stay-in-place-for-their-component-node)):
  - `learn-qa-capture` is ported whole and deleted.
  - `learn-runbook-capture` keeps the [792a13 write-memory](./792a_product-spec.md#792a13-added-write-memory) requirement for [792c7l1 write-memory product spec](./792c_implementation-plan.md#792c7l1-write-memory-product-spec) and the skill runbook note
    requirement for [792c7k1 skill-runbook-registration product spec](./792c_implementation-plan.md#792c7k1-skill-runbook-registration-product-spec).
  - `vault-embed-on-write` keeps everything but the `learn` guarantee. The `amend` and `resituate`
    re-embedding requirement is added to the sources of note-revision ([792c7c1 note-revision product spec](./792c_implementation-plan.md#792c7c1-note-revision-product-spec)).
  - `vault-note-identity` keeps a residue requirement with the repo and user detection mechanism, the
    first-write user rule for served learns and pull-downs, and the empty-vault-name rules, for
    [792c7b7 note-provenance tech spec](./792c_implementation-plan.md#792c7b7-note-provenance-tech-spec).
  - `vault-concurrency-safe-writes` keeps all its text for [792c7b4 vault-write-lock tech spec](./792c_implementation-plan.md#792c7b4-vault-write-lock-tech-spec); only its Purpose gains a pointer.
  - ADR-0012 keeps its round-2 and round-3 gating caveat for [792c8a Residual legacy sweep](./792c_implementation-plan.md#792c8a-residual-legacy-sweep), the `vocab stats` QA lines for
    [792c7e1 controlled-vocabulary product spec](./792c_implementation-plan.md#792c7e1-controlled-vocabulary-product-spec), and the `stripMachineLines` QA markers for [792c7b6 embedding-sidecar tech spec](./792c_implementation-plan.md#792c7b6-embedding-sidecar-tech-spec).
  - ADR-0026 keeps its retrieval decision for [792c7f1 memory-query product spec](./792c_implementation-plan.md#792c7f1-memory-query-product-spec), its learn-routing decision for [792c7n1 learn product spec](./792c_implementation-plan.md#792c7n1-learn-product-spec), its
    historical consequences for [792c8a Residual legacy sweep](./792c_implementation-plan.md#792c8a-residual-legacy-sweep), and its two 2026-09-26 registration amendments for [792c7k1 skill-runbook-registration product spec](./792c_implementation-plan.md#792c7k1-skill-runbook-registration-product-spec).
  The catalog entries of those nodes now list these parts.
- **Rationale:** each part has one owner, and a node removes only what it ports.
- **Alternatives:** port the other parts into [792a3 note-writing](./792a_product-spec.md#792a3-added-note-writing) now. Rejected because they describe other
  capabilities' behavior.

## 792d58 Note-writing requirements beyond the legacy text

- **Choice:** [792a3 note-writing](./792a_product-spec.md#792a3-added-note-writing) adds requirements that the legacy sources don't state but the code, `--help` and
  tests show: the fact and feedback body formulas and the dropped leading `When` ([792a3a Fact notes](./792a_product-spec.md#792a3a-fact-notes), [792a3b Feedback notes](./792a_product-spec.md#792a3b-feedback-notes)),
  the argument validations ([792a3g Argument validation](./792a_product-spec.md#792a3g-argument-validation)), the optional metadata fields ([792a3h Optional metadata](./792a_product-spec.md#792a3h-optional-metadata)), the letter rollover and
  the placement refusals ([792a3j Luhmann placement](./792a_product-spec.md#792a3j-luhmann-placement), [792a3k Placement refusals](./792a_product-spec.md#792a3k-placement-refusals)), the empty-situation refusal ([792a3f Required content](./792a_product-spec.md#792a3f-required-content)), the QA validations
  and certainty default ([792a3r Question-and-answer validation](./792a_product-spec.md#792a3r-question-and-answer-validation)), and the warning when embedding fails ([792a3n Embed on write](./792a_product-spec.md#792a3n-embed-on-write)).
- **Rationale:** [792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated) and [792d49 Product requirements beyond the OpenSpec text](#792d49-product-requirements-beyond-the-openspec-text). Each addition is observable through `engram learn` and is
  already exercised by the shipped binary.
- **Alternatives:** port only the OpenSpec text.

## 792d59 QA retrieval and per-kind vocabulary tagging belong to note-writing

- **Choice:** [792a3t Question-and-answer retrieval](./792a_product-spec.md#792a3t-question-and-answer-retrieval) (query returns answers but never questions) and [792a3u Vocabulary tagging by kind](./792a_product-spec.md#792a3u-vocabulary-tagging-by-kind) (vocabulary tagging
  applies to every kind but questions) are [792a3 note-writing](./792a_product-spec.md#792a3-added-note-writing) requirements, although `engram query` and
  the vocabulary engine carry them out. The query-engine ([792c7f5 query-engine tech spec](./792c_implementation-plan.md#792c7f5-query-engine-tech-spec)) and vocabulary-engine ([792c7e3 vocabulary-engine tech spec](./792c_implementation-plan.md#792c7e3-vocabulary-engine-tech-spec))
  nodes map these scenarios to their requirements.
- **Rationale:** both are the defining contract of the QA kind (ADR-0012), which `learn-qa-capture`
  states and this node owns. Restating them in [792a7 memory-query](./792a_product-spec.md#792a7-added-memory-query) and [792a6 controlled-vocabulary](./792a_product-spec.md#792a6-added-controlled-vocabulary) would duplicate
  the rule.
- **Alternatives:** move [792a3t Question-and-answer retrieval](./792a_product-spec.md#792a3t-question-and-answer-retrieval) to [792a7 memory-query](./792a_product-spec.md#792a7-added-memory-query) and [792a3u Vocabulary tagging by kind](./792a_product-spec.md#792a3u-vocabulary-tagging-by-kind) to [792a6 controlled-vocabulary](./792a_product-spec.md#792a6-added-controlled-vocabulary).

## 792d60 command-line-surface refuses undefined and missing required flags

- **Choice:** [792a3 note-writing](./792a_product-spec.md#792a3-added-note-writing) adds requirement [792b2c5 Argument refusal](./792b_technical-spec.md#792b2c5-argument-refusal) to [792b2 command-line-surface](./792b_technical-spec.md#792b2-added-command-line-surface): an undefined flag or a
  missing required flag is refused before the command runs. `TestTargets_LearnRunbook_NoSkillHashFlag`
  and `TestTargets_NoSkillKeyOrSourceFlags` are renamed to its first scenario, [792b2c5a1 An undefined flag is refused](./792b_technical-spec.md#792b2c5a-scenarios); the
  [792a12 skill-runbook-registration](./792a_product-spec.md#792a12-added-skill-runbook-registration) node ([792c7k1 skill-runbook-registration product spec](./792c_implementation-plan.md#792c7k1-skill-runbook-registration-product-spec)) lists its own scenario for them as `split deferred`.
- **Rationale:** [792a3e1a A task type is refused](./792a_product-spec.md#792a3e1-uat-scenarios), [792a3f1a A missing situation is refused](./792a_product-spec.md#792a3f1-uat-scenarios) and parts of [792a3g1a An invalid argument is refused before any write](./792a_product-spec.md#792a3g1-uat-scenarios) and [792a3r1a An invalid pair is refused before any write](./792a_product-spec.md#792a3r1-uat-scenarios) are refused by argument
  parsing, which this component owns ([792b2a Orientation](./792b_technical-spec.md#792b2a-orientation)), before any `learn` code runs. Those two tests are the
  existing direct tests of that refusal.
- **Alternatives:** leave the tests for [792c7k1 skill-runbook-registration product spec](./792c_implementation-plan.md#792c7k1-skill-runbook-registration-product-spec). Rejected because [792b2c5a1 An undefined flag is refused](./792b_technical-spec.md#792b2c5a-scenarios) would then have no test.

## 792d61 note-document-format's edit, CRLF and field-survival contracts arrive with their scenarios

- **Choice:** [792b6 note-document-format](./792b_technical-spec.md#792b6-added-note-document-format) specifies only what [792a3 note-writing](./792a_product-spec.md#792a3-added-note-writing)'s scenarios need: filenames, the frontmatter
  block, each kind's keys, the body formulas, the QA documents and frontmatter splitting. These wait
  for the capabilities whose product scenarios they support, as [792d51 command-line-surface routing requirements arrive with their scenarios](#792d51-command-line-surface-routing-requirements-arrive-with-their-scenarios) does for [792b2 command-line-surface](./792b_technical-spec.md#792b2-added-command-line-surface):
  - YAML-node edits that keep unknown keys and anchors, refuse an anchored or undecodable rewrite,
    and read CRLF notes as LF go to note-revision ([792c7c1 note-revision product spec](./792c_implementation-plan.md#792c7c1-note-revision-product-spec)), whose sources now list the
    `vault-note-identity` note-reader requirement and the amend and resituate paragraphs of its
    field-survival requirement.
  - Survival of the exchange fields goes to parent-sync ([792c7t1 parent-sync product spec](./792c_implementation-plan.md#792c7t1-parent-sync-product-spec)), which already owns them.
  - The `skill_hash`, `skill_key` and `skill_source` fields go to [792a12 skill-runbook-registration](./792a_product-spec.md#792a12-added-skill-runbook-registration)
    ([792c7k1 skill-runbook-registration product spec](./792c_implementation-plan.md#792c7k1-skill-runbook-registration-product-spec)), whose sources now list those `vault-note-identity` requirements.
  - The "CRLF-safe reading requirement" this node listed from `update-reparent-luhmann-batch` is the
    rename-and-rewrite step's, which vault-migration ([792c7i1 vault-migration product spec](./792c_implementation-plan.md#792c7i1-vault-migration-product-spec), [792c7i2 note-reparenting tech spec](./792c_implementation-plan.md#792c7i2-note-reparenting-tech-spec)) already owns.
  Their tests stay unrenamed until then and are listed as deferred no-scenario tests.
- **Rationale:** the technical guide requires every requirement to name the product scenarios it
  supports, and none of these has a scenario yet.
- **Alternatives:** specify them now with no supported scenarios. Rejected as untraceable.

## 792d62 A fact or feedback situation is stored twice

- **Choice:** [792b6a Orientation](./792b_technical-spec.md#792b6a-orientation) describes the current form: a fact or feedback note holds its situation both in
  `situation:` and in its body formula. The legacy invariant S2 called this a defect and asked for
  one representation, and it said only the body was embedded and hashed, with `engram check` to
  compare the two copies. Today the situation vector is built from `situation:`, the content hash
  covers both the situation and the body, and `engram check` does not compare the copies.
- **Rationale:** [792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated). The S2 text is stale on the embedding and was never implemented on the
  check; the double copy is current behavior.
- **Alternatives:** a **MUST** that the copies match, enforced by `check`. Rejected because nothing
  enforces it.

## 792d63 Which writer takes which lock is the writer's requirement

- **Choice:** [792b8 vault-write-lock](./792b_technical-spec.md#792b8-added-vault-write-lock) specifies the lock itself: exclusive hold of a lock file, its creation, waiting,
  release and failure reporting, and the vault's lock file `.luhmann.lock`. The per-command clauses
  of `vault-concurrency-safe-writes` stay in that spec for the callers that own them:
  `learn` and `learn qa` for note-authoring ([792c7b8 note-authoring tech spec](./792c_implementation-plan.md#792c7b8-note-authoring-tech-spec)); `amend`, `resituate`, `activate` and the
  lock-free shared helpers for note-amendment ([792c7c3 note-amendment tech spec](./792c_implementation-plan.md#792c7c3-note-amendment-tech-spec)); the manifest lock for [792b18 chunk-index-store](./792b_technical-spec.md#792b18-added-chunk-index-store)
  ([792c7d5 chunk-index-store tech spec](./792c_implementation-plan.md#792c7d5-chunk-index-store-tech-spec)); and `vocab bootstrap` and `refit` for vocabulary-engine ([792c7e3 vocabulary-engine tech spec](./792c_implementation-plan.md#792c7e3-vocabulary-engine-tech-spec)). Those nodes' sources
  now list them. ADR-0013 is reduced to a pointer to [792b3 durable-file-store](./792b_technical-spec.md#792b3-added-durable-file-store) and [792b8 vault-write-lock](./792b_technical-spec.md#792b8-added-vault-write-lock), and code comments citing it
  for the lock now cite [792b8 vault-write-lock](./792b_technical-spec.md#792b8-added-vault-write-lock).
  `TestInvariant_K1_ConcurrentLearnNeverCollides` takes product scenario [792a3m1a Concurrent learns get distinct IDs](./792a_product-spec.md#792a3m1-uat-scenarios), because it drives
  the real locked write path and asserts the product outcome.
- **Rationale:** the technical guide puts an interaction requirement (what a caller does, and in
  what order, while holding a dependency) in the caller's component. Holding the lock across
  ID allocation and the write is [792b12 note-authoring](./792b_technical-spec.md#792b12-added-note-authoring)'s ordering, not the lock's behavior.
- **Alternatives:** port every lock clause into [792b8 vault-write-lock](./792b_technical-spec.md#792b8-added-vault-write-lock) now. Rejected because most would support no
  existing product scenario and would describe callers' ordering.

## 792d64 `TARG_BASELINE_PATTERN` now matches both test-name forms

- **Choice:** node [792c7b5 text-embedder tech spec](./792c_implementation-plan.md#792c7b5-text-embedder-tech-spec) renamed the first two `TestT<N>_` tests (`TestT10_…` and `TestT13_…`),
  dropping the old prefix as [792d38 TARG_BASELINE_PATTERN (provisional)](#792d38-targ_baseline_pattern-provisional) says. `dev/targs.go` now defaults the pattern to
  `Test(T[0-9]+|[0-9]+[a-z][a-z0-9]*)_`, which matches the remaining `TestT138`–`TestT143` tests
  until transcript-reader ([792c7d2 transcript-reader tech spec](./792c_implementation-plan.md#792c7d2-transcript-reader-tech-spec)) renames them, and every scenario-ID test name.
- **Rationale:** [792d38 TARG_BASELINE_PATTERN (provisional)](#792d38-targ_baseline_pattern-provisional) asks the first such rename to keep the renamed tests in baseline selection.
  The scenario-ID form is the new spec-trace naming, so the pattern follows it. The final value is
  still [792e7 What should TARG_BASELINE_PATTERN become?](./792e_open-questions.md#792e7-what-should-targ_baseline_pattern-become)'s question.
- **Alternatives:** match only the eight original tests by their new names. Rejected because new
  spec-traced tests would never be selected.

## 792d65 The missing-model error names an unimplemented `ENGRAM_MODEL_PATH`

- **Choice:** [792b9c5 Model cache](./792b_technical-spec.md#792b9c5-model-cache) states only that a binary without the bundled model fails with an error saying
  the model is missing. That error also tells the user to set `ENGRAM_MODEL_PATH` to a directory
  holding `model.onnx`, but no code reads `ENGRAM_MODEL_PATH`. The divergence is recorded here, not
  specified.
- **Rationale:** [792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated) and [792d42 Known divergences between docs and code](#792d42-known-divergences-between-docs-and-code). The advice is not current behavior.
- **Alternatives:** specify the variable. Rejected because setting it changes nothing.

## 792d66 Embedding-sidecar scope, and the partial-model-mismatch gap

- **Choice:**
  - [792b10 embedding-sidecar](./792b_technical-spec.md#792b10-added-embedding-sidecar) specifies the sidecar format and path, the embedding sources, the content hash,
    building, the learn-time write, state classification and cosine similarity. The `engram embed
    apply` and `status` selection waits for vault-health ([792c7h1 vault-health product spec](./792c_implementation-plan.md#792c7h1-vault-health-product-spec)), and the `amend`, `resituate` and
    `activate` sidecar writes for note-revision ([792c7c1 note-revision product spec](./792c_implementation-plan.md#792c7c1-note-revision-product-spec)); `vault-embed-on-write` keeps those
    residues, and both nodes' sources now list them. Their tests (`embed_test`, `TestSelectStates`)
    stay unrenamed; `TestKindFromContent` reads a note's kind for the query payload and is left to
    memory-query ([792c7f1 memory-query product spec](./792c_implementation-plan.md#792c7f1-memory-query-product-spec)).
  - The known defect M4 is described, not specified: a query drops sidecars made by another model
    with only a stderr advisory, so a partial model change silently thins its results, and only a
    vault with no compatible sidecar at all fails. [792b10c7 State classification](./792b_technical-spec.md#792b10c7-state-classification)'s rationale states this.
  - ADR-0006 (embed source by kind) is deleted. Its only surviving content, that every kind embeds a
    situation and a body, is in [792b10 embedding-sidecar](./792b_technical-spec.md#792b10-added-embedding-sidecar)'s rationale; the episode kind it routed is retired ([792d6 Removed features are not documented; enduring absences become MUST NOT](#792d6-removed-features-are-not-documented-enduring-absences-become-must-not)).
    ADR-0002 and ADR-0003 are reduced to pointers.
- **Rationale:** [792d51 command-line-surface routing requirements arrive with their scenarios](#792d51-command-line-surface-routing-requirements-arrive-with-their-scenarios), [792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated) and [792d6 Removed features are not documented; enduring absences become MUST NOT](#792d6-removed-features-are-not-documented-enduring-absences-become-must-not).
- **Alternatives:** a **MUST** that all sidecars share one model. Rejected because nothing enforces
  it.

## 792d67 note-provenance covers detection and the first-write stamp

- **Choice:** [792b11 note-provenance](./792b_technical-spec.md#792b11-added-note-provenance) specifies repository and user detection and the first-write stamp, which
  [792a3 note-writing](./792a_product-spec.md#792a3-added-note-writing)'s scenarios need. The amend re-stamp, which keeps a prior user when detection comes
  back empty, goes to note-revision ([792c7c1 note-revision product spec](./792c_implementation-plan.md#792c7c1-note-revision-product-spec)); identity backfill and its `project:` preference go to
  vault-migration ([792c7i1 vault-migration product spec](./792c_implementation-plan.md#792c7i1-vault-migration-product-spec)), which already owns them; and first-write stamping by a served learn and
  by a pull-down go to vault-serving ([792c7s1 vault-serving product spec](./792c_implementation-plan.md#792c7s1-vault-serving-product-spec)) and parent-sync ([792c7t1 parent-sync product spec](./792c_implementation-plan.md#792c7t1-parent-sync-product-spec)). `vault-note-identity`
  keeps those clauses, and the nodes' sources now list them. Their tests stay unrenamed.
  `TestDetectRepo` and `TestDetectUser` are tables whose cases cover every scenario of their
  requirement, so each takes the first scenario's ID and lists the rest as `split deferred`.
- **Rationale:** [792d51 command-line-surface routing requirements arrive with their scenarios](#792d51-command-line-surface-routing-requirements-arrive-with-their-scenarios). Only `learn` has product scenarios so far.
- **Alternatives:** specify the amend and backfill stamping now, with no supported scenarios.

## 792d68 note-authoring scope and the tests it leaves

- **Choice:** [792b12 note-authoring](./792b_technical-spec.md#792b12-added-note-authoring) specifies argument validation, required content, the locked write, the
  mapping of arguments to fields, and the question-and-answer write. These stay for later nodes:
  the vocabulary refit trigger after a learn (`TestApplyVocabAssignmentAfterLearn_TriggerFires`) and
  `vocab_version` survival through amend (`TestFactAmend_VocabVersionRoundTrip`) for
  controlled-vocabulary ([792c7e1 controlled-vocabulary product spec](./792c_implementation-plan.md#792c7e1-controlled-vocabulary-product-spec)) and note-revision ([792c7c1 note-revision product spec](./792c_implementation-plan.md#792c7c1-note-revision-product-spec)); the rule that `learn` never writes the
  skill fields (`TestRunLearn_Runbook_NeverWritesSkillHash`,
  `TestRunLearn_Runbook_NeverWritesSkillKeyOrSource`) for skill-runbook-registration ([792c7k1 skill-runbook-registration product spec](./792c_implementation-plan.md#792c7k1-skill-runbook-registration-product-spec)); and
  the question-note exclusions in query and `vocab stats` for memory-query ([792c7f1 memory-query product spec](./792c_implementation-plan.md#792c7f1-memory-query-product-spec), [792b24 query-engine](./792b_technical-spec.md#792b24-added-query-engine)
  [792c7f5 query-engine tech spec](./792c_implementation-plan.md#792c7f5-query-engine-tech-spec)) and [792a6 controlled-vocabulary](./792a_product-spec.md#792a6-added-controlled-vocabulary). The c3 QA flow's notes on those exclusions stay in
  `c3-components.md` as a residue, and those nodes' sources list it. The c2 learn flow keeps its
  skill side for learn ([792c7n1 learn product spec](./792c_implementation-plan.md#792c7n1-learn-product-spec)). `TestLearnFact_BootstrapsMissingVault`, a dispatch test that
  `learn` creates a missing vault first, takes [792b2 command-line-surface](./792b_technical-spec.md#792b2-added-command-line-surface)'s [792b2c3a1 Every vault-resolving command ensures the vault first](./792b_technical-spec.md#792b2c3a-scenarios) ([792c4a Capability Node Workflow](./792c_implementation-plan.md#792c4a-capability-node-workflow) step 6).
- **Rationale:** [792d51 command-line-surface routing requirements arrive with their scenarios](#792d51-command-line-surface-routing-requirements-arrive-with-their-scenarios), and [792d52 Test IDs: product scenario for end-to-end checks, technical scenario otherwise](#792d52-test-ids-product-scenario-for-end-to-end-checks-technical-scenario-otherwise) for the bootstrap test.
- **Alternatives:** none considered.

## 792d69 Mixed sources of note-revision are split by owner

- **Choice:** node [792c7c1 note-revision product spec](./792c_implementation-plan.md#792c7c1-note-revision-product-spec) ports only the [792a4 note-revision](./792a_product-spec.md#792a4-added-note-revision) parts of its sources ([792d31 Each legacy item has one owning node](#792d31-each-legacy-item-has-one-owning-node), [792d57 Mixed sources of note-writing are split by owner](#792d57-mixed-sources-of-note-writing-are-split-by-owner)):
  - `vault-note-identity`: the amend re-stamp, its `project:` exclusion, the vault fill and the
    amend and resituate key-preservation, anchor and CRLF paragraphs are ported. The bookkeeping
    rule for `--clear-pending`, `--discard --into` and the offer receipt stays as a residue
    requirement for curate ([792c7u1 curate product spec](./792c_implementation-plan.md#792c7u1-curate-product-spec)) and parent-sync ([792c7t1 parent-sync product spec](./792c_implementation-plan.md#792c7t1-parent-sync-product-spec)), with the fold's CRLF and
    `--expect-hash` clause and the "Accepting an offer keeps its author" and fold scenarios. The
    clear-pending vault-fill scenario is replaced by an `--activate` one ([792a4i1a A use recorded on a note without vault fills it in](./792a_product-spec.md#792a4i1-uat-scenarios)), because the
    rule covers every amend that writes.
  - The note-reader CRLF requirement is split: its embedding clause is [792b10c9 CRLF notes](./792b_technical-spec.md#792b10c9-crlf-notes), and amend's and
    resituate's reads are [792a4m CRLF notes](./792a_product-spec.md#792a4m-crlf-notes). Its query, check, count and vocabulary readers stay in place for
    memory-query ([792c7f1 memory-query product spec](./792c_implementation-plan.md#792c7f1-memory-query-product-spec)), vault-health ([792c7h1 vault-health product spec](./792c_implementation-plan.md#792c7h1-vault-health-product-spec)), vault-counting ([792c7g1 vault-counting product spec](./792c_implementation-plan.md#792c7g1-vault-counting-product-spec)) and
    controlled-vocabulary ([792c7e1 controlled-vocabulary product spec](./792c_implementation-plan.md#792c7e1-controlled-vocabulary-product-spec)), whose sources now list them; the `crlf_readers_internal_test`
    tests go with them.
  - `vault-embed-on-write`: the amend and resituate re-embedding and the lossless `last_used` bump
    are ported; the `embed apply` residue stays for [792c7h1 vault-health product spec](./792c_implementation-plan.md#792c7h1-vault-health-product-spec).
  - `vault-wikilink-resolution`: the canonical-form requirement is ported ([792a4d Supersession](./792a_product-spec.md#792a4d-supersession)). The parser
    requirement and its two resolution scenarios wait for wikilink-graph ([792c7c2 wikilink-graph tech spec](./792c_implementation-plan.md#792c7c2-wikilink-graph-tech-spec)), and the
    dangling-link scenario for [792c7h1 vault-health product spec](./792c_implementation-plan.md#792c7h1-vault-health-product-spec).
  - c3 K4w is only a node of the L3 diagram, with no catalog row. It states no product behavior, so
    it is left whole for note-amendment ([792c7c3 note-amendment tech spec](./792c_implementation-plan.md#792c7c3-note-amendment-tech-spec)).
- **Rationale:** each part has one owner, and a node removes only what it ports.
- **Alternatives:** port the CRLF reader requirement whole into [792a4 note-revision](./792a_product-spec.md#792a4-added-note-revision). Rejected because its
  scenarios are query, check, count and vocabulary behavior, which other capabilities own.

## 792d70 `amend --repo`, `--user` and `--pending` are not specified

- **Choice:** [792a4 note-revision](./792a_product-spec.md#792a4-added-note-revision) leaves out the three undocumented `amend` flags that `--help` lists without a
  description (the [792d42 Known divergences between docs and code](#792d42-known-divergences-between-docs-and-code) divergence). `--repo` and `--user` are accepted and have no effect on a
  CLI amend; they carry a served write's identity across the wire. `--pending` fails with
  "unsupported value type: *bool" for any value, because the field is a pointer that targ cannot
  parse.
- **Rationale:** [792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated) and [792d42 Known divergences between docs and code](#792d42-known-divergences-between-docs-and-code). None of them is observable behavior a user can rely on.
- **Alternatives:** specify `--repo` and `--user` as ignored (MUST NOT affect the note). Rejected as
  protecting an accident of how the argument struct is shared with the served path.

## 792d71 Rebuilding a sidecar drops its last-used date

- **Choice:** a content amend or a resituate rebuilds the note's sidecar from scratch, and the new
  sidecar carries no `last_used`, so a recorded use is lost. [792a4n Activation](./792a_product-spec.md#792a4n-activation) specifies only that an
  activation keeps the rest of the sidecar; the loss on a rebuild is recorded here, not specified.
- **Rationale:** [792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated). Whether a rebuild should carry `last_used` forward is a change proposal.
- **Alternatives:** a **MUST** that a rebuild keeps `last_used`. Rejected because the code does not.

## 792d72 Note-revision requirements beyond the legacy text

- **Choice:** [792a4 note-revision](./792a_product-spec.md#792a4-added-note-revision) adds requirements that the legacy sources don't state but the code, `--help` and
  tests show: the accepted reference forms and the not-found refusal ([792a4a Note references](./792a_product-spec.md#792a4a-note-references)), field replacement by
  kind and the rebuilt body formula ([792a4b Field replacement](./792a_product-spec.md#792a4b-field-replacement)), the refusal of other kinds ([792a4c Other note kinds are refused](./792a_product-spec.md#792a4c-other-note-kinds-are-refused)), the chunk-source
  merge ([792a4e Chunk sources](./792a_product-spec.md#792a4e-chunk-sources)), amend's argument validation ([792a4f Amend argument validation](./792a_product-spec.md#792a4f-amend-argument-validation)), resituate's limit to fact and feedback and
  its failure when the sidecar cannot be rebuilt ([792a4l Resituation](./792a_product-spec.md#792a4l-resituation)), and activation ([792a4n Activation](./792a_product-spec.md#792a4n-activation)). [792a4o Concurrent revisions never lose an update](./792a_product-spec.md#792a4o-concurrent-revisions-never-lose-an-update) states
  for revisions the guarantee that `vault-concurrency-safe-writes` gives every vault writer.
  `activate --note` resolves a path or basename, not a bare Luhmann ID, unlike amend and resituate;
  [792a4n Activation](./792a_product-spec.md#792a4n-activation) describes this as it is.
- **Rationale:** [792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated), [792d49 Product requirements beyond the OpenSpec text](#792d49-product-requirements-beyond-the-openspec-text) and [792d58 Note-writing requirements beyond the legacy text](#792d58-note-writing-requirements-beyond-the-legacy-text).
- **Alternatives:** port only the OpenSpec text.

## 792d73 One rule for the supersession link form

- **Choice:** [792a4d Supersession](./792a_product-spec.md#792a4d-supersession) states the canonical `Supersedes: [[<target>]]` line for both `engram learn` and
  `engram amend`, as the `vault-wikilink-resolution` requirement does. [792a3h Optional metadata](./792a_product-spec.md#792a3h-optional-metadata) already shows the
  learn line for a target given without `.md`.
- **Rationale:** both commands write the line through one renderer, and a single owner keeps the rule
  from being restated in [792a3 note-writing](./792a_product-spec.md#792a3-added-note-writing).
- **Alternatives:** add a learn-only requirement to [792a3 note-writing](./792a_product-spec.md#792a3-added-note-writing). Rejected as duplicating the rule.

## 792d74 Where note-revision's technical contracts land

- **Choice:**
  - The YAML-node edits and the CRLF read are [792b6 note-document-format](./792b_technical-spec.md#792b6-added-note-document-format)'s [792b6c7 Frontmatter node edits](./792b_technical-spec.md#792b6c7-frontmatter-node-edits) and [792b6c8 CRLF reading](./792b_technical-spec.md#792b6c8-crlf-reading) ([792d61 note-document-format's edit, CRLF and field-survival contracts arrive with their scenarios](#792d61-note-document-formats-edit-crlf-and-field-survival-contracts-arrive-with-their-scenarios)),
    the re-stamp is [792b11 note-provenance](./792b_technical-spec.md#792b11-added-note-provenance)'s [792b11c4 Re-stamp](./792b_technical-spec.md#792b11c4-re-stamp) ([792d67 note-provenance covers detection and the first-write stamp](#792d67-note-provenance-covers-detection-and-the-first-write-stamp)), and the CRLF embedding read is
    [792b10 embedding-sidecar](./792b_technical-spec.md#792b10-added-embedding-sidecar)'s [792b10c9 CRLF notes](./792b_technical-spec.md#792b10c9-crlf-notes).
  - The sidecar rebuild after a revision adds no [792b10 embedding-sidecar](./792b_technical-spec.md#792b10-added-embedding-sidecar) requirement: it builds through
    [792b10c5 Building a sidecar](./792b_technical-spec.md#792b10c5-building-a-sidecar) and writes through [792b3c1 Atomic replace](./792b_technical-spec.md#792b3c1-atomic-replace), and warning or failing is the reviser's choice, specified by
    note-amendment ([792c7c3 note-amendment tech spec](./792c_implementation-plan.md#792c7c3-note-amendment-tech-spec)). The last-used bump lives in `activate.go` and is [792b14 note-amendment](./792b_technical-spec.md#792b14-added-note-amendment)'s too.
  - `TestIdentityStamp_Stamp_OverwritesAllThree` verifies the raw stamp that blanks an empty user,
    which no revision uses, so it is a no-scenario test. [792b11c4a1 All three are overwritten](./792b_technical-spec.md#792b11c4a-scenarios) is left to the re-stamp property
    test that [792b14 note-amendment](./792b_technical-spec.md#792b14-added-note-amendment) examines.
  - `TestTargets_Resituate` and `TestActivate_MissWithoutParentFails` dispatch whole commands and take
    the product IDs [792a4a1b An unknown reference is refused](./792a_product-spec.md#792a4a1-uat-scenarios) and [792a4n1c A missing note fails only its own ref](./792a_product-spec.md#792a4n1-uat-scenarios) ([792d52 Test IDs: product scenario for end-to-end checks, technical scenario otherwise](#792d52-test-ids-product-scenario-for-end-to-end-checks-technical-scenario-otherwise)).
- **Rationale:** [792d30 Components are migrated once and extended later](#792d30-components-are-migrated-once-and-extended-later) and [792d51 command-line-surface routing requirements arrive with their scenarios](#792d51-command-line-surface-routing-requirements-arrive-with-their-scenarios): a migrated component gains the requirements that a later
  capability's scenarios need.
- **Alternatives:** a separate sidecar-rebuild requirement in [792b10 embedding-sidecar](./792b_technical-spec.md#792b10-added-embedding-sidecar). Rejected because building and
  writing are already specified.

## 792d75 wikilink-graph's link, graph and resolution contracts arrive with their scenarios

- **Choice:** [792b13 wikilink-graph](./792b_technical-spec.md#792b13-added-wikilink-graph) specifies only what existing product scenarios need: the vault scan (used by
  `amend`, `resituate`, `embed status` and `check`) and note-reference resolution. These wait, as in
  [792d51 command-line-surface routing requirements arrive with their scenarios](#792d51-command-line-surface-routing-requirements-arrive-with-their-scenarios) and [792d61 note-document-format's edit, CRLF and field-survival contracts arrive with their scenarios](#792d61-note-document-formats-edit-crlf-and-field-survival-contracts-arrive-with-their-scenarios):
  - link parsing (`.md` normalization, fences, dedupe), the graph build and in-degree for
    vault-counting ([792c7g1 vault-counting product spec](./792c_implementation-plan.md#792c7g1-vault-counting-product-spec)), because `engram count` prints a note's in-degree, with ADR-0007's
    decision and c3 K7's graph clause;
  - the unresolved-link report, the G0 invariant, the recurring-defect shape and the whole
    `vault-wikilink-resolution` parser requirement with its three scenarios for [792a9 vault-health](./792a_product-spec.md#792a9-added-vault-health)
    ([792c7h1 vault-health product spec](./792c_implementation-plan.md#792c7h1-vault-health-product-spec)), because `engram check` reports them.
  The parser, fence, graph and resolution tests stay unrenamed until then. `BFSWithCap`,
  `InDegreeIn` and `UndirectedNeighbors` have no production caller; their tests verify no scenario.
  ADR-0007's context, that navigation lives in authored links and the graph is derived fresh, is
  ported as [792b13a Orientation](./792b_technical-spec.md#792b13a-orientation)'s rationale.
- **Rationale:** the technical guide requires every requirement to name the product scenarios it
  supports, and no [792a2 vault-identity](./792a_product-spec.md#792a2-added-vault-identity)–[792a4 note-revision](./792a_product-spec.md#792a4-added-note-revision) scenario depends on how links are parsed or counted.
- **Alternatives:** specify parsing and the graph now with no supported scenarios. Rejected as
  untraceable.
- **Divergence noted:** a link's target is the whole text between `[[` and `]]`, so
  `[[note|display]]` and `[[note#heading]]` do not resolve to `note` in the graph, although note
  references ([792b13c2 Note references](./792b_technical-spec.md#792b13c2-note-references)) strip a `|display` part. Recorded for [792a8 vault-counting](./792a_product-spec.md#792a8-added-vault-counting) and [792a9 vault-health](./792a_product-spec.md#792a9-added-vault-health) to
  describe, not resolved here ([792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated)).

## 792d76 note-amendment scope and the tests it leaves

- **Choice:** [792b14 note-amendment](./792b_technical-spec.md#792b14-added-note-amendment) specifies the locked read-modify-write of amend, resituate and activate: the
  lock, amend's refusals, field replacement, supersessions and chunk sources, re-stamp gating and the
  vault fill, the sidecar rebuild, CRLF revision, resituation, the last-used bump and local
  activation refs. These stay for later nodes, with their tests unrenamed: the curation flags
  (`TestDiscardNote_*`, `TestRunAmend_CRLFJudgedVersionUsesTheLFForm`,
  `TestRunAmend_DiscardIntoConvertsCRLFExisting`, `TestRunAmend_ClearPendingKeepsOfferAuthor`) for
  curate ([792c7u1 curate product spec](./792c_implementation-plan.md#792c7u1-curate-product-spec)); the pulled-note decline (`TestRunAmend_DiscardRecordsDeclineOfCRLFPulledNote`)
  for parent-sync ([792c7t1 parent-sync product spec](./792c_implementation-plan.md#792c7t1-parent-sync-product-spec)); the vocabulary assignment after a revision
  (`TestApplyVocabAssignmentAfterAmend_TriggerFires`, `TestApplyVocabAssignmentAfterResituate_*`,
  `TestRunResituate_CallsVocabAssignment`) for controlled-vocabulary ([792c7e1 controlled-vocabulary product spec](./792c_implementation-plan.md#792c7e1-controlled-vocabulary-product-spec)); the skill-hash
  survival tests (`TestRunAmend_Runbook_PreservesSkillHash_*`) for [792a12 skill-runbook-registration](./792a_product-spec.md#792a12-added-skill-runbook-registration)
  ([792c7k1 skill-runbook-registration product spec](./792c_implementation-plan.md#792c7k1-skill-runbook-registration-product-spec)); the supersession inverse used by query (`TestBuildSupersedesInverse`) for [792a7 memory-query](./792a_product-spec.md#792a7-added-memory-query)
  ([792c7f1 memory-query product spec](./792c_implementation-plan.md#792c7f1-memory-query-product-spec)); and the chunk-ID set read from the index (`TestBuildChunkIDSet_ReturnsSourceAnchorKeys`)
  for chunk-index-store ([792c7d5 chunk-index-store tech spec](./792c_implementation-plan.md#792c7d5-chunk-index-store-tech-spec)). The `amend` and `resituate` lock clauses of
  `vault-concurrency-safe-writes` and the vault-lock helpers of its lock-free helper requirement are
  ported; `writeManifestFile` stays for [792b18 chunk-index-store](./792b_technical-spec.md#792b18-added-chunk-index-store).
- **Rationale:** [792d51 command-line-surface routing requirements arrive with their scenarios](#792d51-command-line-surface-routing-requirements-arrive-with-their-scenarios) and [792d63 Which writer takes which lock is the writer's requirement](#792d63-which-writer-takes-which-lock-is-the-writers-requirement): a caller's ordering under the lock is the caller's requirement.
- **Alternatives:** none considered.

## 792d77 Mixed sources of chunk-indexing are split by owner

- **Choice:** node [792c7d1 chunk-indexing product spec](./792c_implementation-plan.md#792c7d1-chunk-indexing-product-spec) ports the product behavior of `ingest-auto-sweep`, `ingest-chunk-dedup`
  and `ingest-prune-detach` into [792a5 chunk-indexing](./792a_product-spec.md#792a5-added-chunk-indexing) and leaves each file a residue for the node that owns the
  rest ([792d31 Each legacy item has one owning node](#792d31-each-legacy-item-has-one-owning-node), [792d50 Technical clauses of a mixed OpenSpec spec stay in place for their component node](#792d50-technical-clauses-of-a-mixed-openspec-spec-stay-in-place-for-their-component-node)):
  - `ingest-auto-sweep` keeps the sweep-root mechanics (name prefixes on the session-log root,
    path prefixes at root derivation, the `.claude` exclude list) for [792b17 ingestion-source-discovery](./792b_technical-spec.md#792b17-added-ingestion-source-discovery)
    ([792c7d4 ingestion-source-discovery tech spec](./792c_implementation-plan.md#792c7d4-ingestion-source-discovery-tech-spec)), and the zero-record index-file clause for chunk-index-store ([792c7d5 chunk-index-store tech spec](./792c_implementation-plan.md#792c7d5-chunk-index-store-tech-spec)).
  - `ingest-chunk-dedup` keeps the dedup mechanics for [792c7d5 chunk-index-store tech spec](./792c_implementation-plan.md#792c7d5-chunk-index-store-tech-spec) and the `engram update` backlog
    notice for install-and-update ([792c7j1 install-and-update product spec](./792c_implementation-plan.md#792c7j1-install-and-update-product-spec)).
  - `ingest-prune-detach` keeps the directory-scan clause for [792c7d5 chunk-index-store tech spec](./792c_implementation-plan.md#792c7d5-chunk-index-store-tech-spec).
  - The prune details of the c1 S2 row are ported ([792a5p Prune detaches vanished sources](./792a_product-spec.md#792a5p-prune-detaches-vanished-sources)–[792a5r Prune removes duplicate index files](./792a_product-spec.md#792a5r-prune-removes-duplicate-index-files)) and the row now points at [792a5 chunk-indexing](./792a_product-spec.md#792a5-added-chunk-indexing).
  - ADR-0021's measurements and its context for decisions 1–8 and 10 justify the product
    requirements [792a5d Automatic sweep roots](./792a_product-spec.md#792a5d-automatic-sweep-roots), [792a5l Identical content is indexed once](./792a_product-spec.md#792a5l-identical-content-is-indexed-once), [792a5m A later, preferred copy replaces an indexed one](./792a_product-spec.md#792a5m-a-later-preferred-copy-replaces-an-indexed-one), [792a5n Stray vault-note copies are not indexed](./792a_product-spec.md#792a5n-stray-vault-note-copies-are-not-indexed) and [792a5r Prune removes duplicate index files](./792a_product-spec.md#792a5r-prune-removes-duplicate-index-files), so their `Rationale:` paragraphs carry
    them. ADR-0021 itself stays whole for chunk-index-store ([792c7d5 chunk-index-store tech spec](./792c_implementation-plan.md#792c7d5-chunk-index-store-tech-spec)), its listed owner, which adds
    the technical rationale and removes the ADR.
  - `embedding-sidecar` ([792b10 embedding-sidecar](./792b_technical-spec.md#792b10-added-embedding-sidecar)) is added to the extended components: Rule B ([792a5n Stray vault-note copies are not indexed](./792a_product-spec.md#792a5n-stray-vault-note-copies-are-not-indexed)) recognizes a
    vault-note copy by its sidecar's name, which is [792b10c2 Sidecar path](./792b_technical-spec.md#792b10c2-sidecar-path).
- **Rationale:** each part has one owner, and a node removes only what it ports.
- **Alternatives:** port the mechanics into [792a5 chunk-indexing](./792a_product-spec.md#792a5-added-chunk-indexing). Rejected because product specs exclude
  implementation detail.

## 792d78 Chunk-indexing requirements beyond the legacy text

- **Choice:** [792a5 chunk-indexing](./792a_product-spec.md#792a5-added-chunk-indexing) adds requirements that the OpenSpec specs don't state but the code, `--help`
  and tests show, each confirmed against the built binary: the index location ([792a5a Chunk index location](./792a_product-spec.md#792a5a-chunk-index-location)), the named,
  Pi and swept sources and their read-failure rules ([792a5b Explicitly named sources](./792a_product-spec.md#792a5b-explicitly-named-sources), [792a5c Swept directories](./792a_product-spec.md#792a5c-swept-directories)), the automatic roots and
  exclude lists ([792a5d Automatic sweep roots](./792a_product-spec.md#792a5d-automatic-sweep-roots)), `.engram/sweep.json` ([792a5e Repository sweep configuration](./792a_product-spec.md#792a5e-repository-sweep-configuration)), transcript stripping ([792a5g Transcript content](./792a_product-spec.md#792a5g-transcript-content)), chunk
  boundaries and anchors ([792a5h Chunk boundaries and IDs](./792a_product-spec.md#792a5h-chunk-boundaries-and-ids)), incremental re-indexing and its stdout line ([792a5i Incremental re-indexing](./792a_product-spec.md#792a5i-incremental-re-indexing)), the
  append-only rule ([792a5j A source's chunks are never dropped](./792a_product-spec.md#792a5j-a-sources-chunks-are-never-dropped)), Rule B ([792a5n Stray vault-note copies are not indexed](./792a_product-spec.md#792a5n-stray-vault-note-copies-are-not-indexed)), the prune messages ([792a5p Prune detaches vanished sources](./792a_product-spec.md#792a5p-prune-detaches-vanished-sources)) and `prune --duplicates`'
  origin-blind choice, refusal lines and failure exit ([792a5r Prune removes duplicate index files](./792a_product-spec.md#792a5r-prune-removes-duplicate-index-files)). The time each chunk was first
  indexed is left to memory-query ([792c7f1 memory-query product spec](./792c_implementation-plan.md#792c7f1-memory-query-product-spec)), because only the query's recency channel makes it
  observable.
- **Rationale:** [792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated), [792d49 Product requirements beyond the OpenSpec text](#792d49-product-requirements-beyond-the-openspec-text) and [792d58 Note-writing requirements beyond the legacy text](#792d58-note-writing-requirements-beyond-the-legacy-text).
- **Alternatives:** port only the OpenSpec text.

## 792d79 Known divergences in chunk indexing

- **Choice:** [792a5 chunk-indexing](./792a_product-spec.md#792a5-added-chunk-indexing) describes these as they are and does not specify the alternative ([792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated),
  [792d42 Known divergences between docs and code](#792d42-known-divergences-between-docs-and-code)):
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
    ([792a5j A source's chunks are never dropped](./792a_product-spec.md#792a5j-a-sources-chunks-are-never-dropped)). Duplicates re-accumulate when copies appear across runs (#771); this is current
    behavior, not a requirement to fix.
- **Rationale:** a divergence becomes a later change proposal (vault note 645).
- **Alternatives:** specify the intended behavior with coverage gaps. Rejected because the code does
  not do it.

## 792d80 command-line-surface gains chunk index resolution

- **Choice:** [792a5 chunk-indexing](./792a_product-spec.md#792a5-added-chunk-indexing) adds [792b2c6 Chunk index directory resolution](./792b_technical-spec.md#792b2c6-chunk-index-directory-resolution), the chunk index directory's precedence, as [792d51 command-line-surface routing requirements arrive with their scenarios](#792d51-command-line-surface-routing-requirements-arrive-with-their-scenarios)
  planned. `TestResolveChunksDirPrecedence` covers the flag, the environment and the home fallback,
  so it takes [792b2c6a1 The flag wins](./792b_technical-spec.md#792b2c6a-scenarios) with the others `split deferred`; no test resolves the directory with
  `XDG_DATA_HOME` set, so [792b2c6a3 XDG_DATA_HOME sets the default](./792b_technical-spec.md#792b2c6a-scenarios) is a coverage gap. `TestTargets_IngestAndQueryChunksEmpty` and
  `TestTargets_PruneEmpty` check only that a command with nothing to do is wired and writes no
  stderr, so they verify no scenario. [792b3 durable-file-store](./792b_technical-spec.md#792b3-added-durable-file-store), [792b8 vault-write-lock](./792b_technical-spec.md#792b8-added-vault-write-lock), [792b9 text-embedder](./792b_technical-spec.md#792b9-added-text-embedder) and
  [792b10 embedding-sidecar](./792b_technical-spec.md#792b10-added-embedding-sidecar) gain mappings only; their tests that touch the chunk index are [792b18 chunk-index-store](./792b_technical-spec.md#792b18-added-chunk-index-store)'s
  and are examined in [792c7d5 chunk-index-store tech spec](./792c_implementation-plan.md#792c7d5-chunk-index-store-tech-spec).
- **Rationale:** [792d30 Components are migrated once and extended later](#792d30-components-are-migrated-once-and-extended-later) and [792d51 command-line-surface routing requirements arrive with their scenarios](#792d51-command-line-surface-routing-requirements-arrive-with-their-scenarios).
- **Alternatives:** none considered.

## 792d81 transcript-reader scope and the tests it leaves

- **Choice:**
  - [792b15 transcript-reader](./792b_technical-spec.md#792b15-added-transcript-reader) specifies the production read only: the whole file, user and assistant entries, harness
    noise, oversized content and tool-call summaries. `ReadFrom`'s start time and byte budget, which
    ingestion leaves unset, and `SegmentsFrom`, `Strip` and the `KeepToolCalls` mode, which have no
    production caller, are not specified. Their tests verify no scenario and keep their names,
    including `TestT138`–`TestT143`, which test `Strip`. [792d38 TARG_BASELINE_PATTERN (provisional)](#792d38-targ_baseline_pattern-provisional) expected this node to rename those
    six; the catalog's expectation that `Strip` tests are no-scenario tests wins, so the
    `T[0-9]+` alternative of `TARG_BASELINE_PATTERN` ([792d64 TARG_BASELINE_PATTERN now matches both test-name forms](#792d64-targ_baseline_pattern-now-matches-both-test-name-forms)) still matches them, and [792e7 What should TARG_BASELINE_PATTERN become?](./792e_open-questions.md#792e7-what-should-targ_baseline_pattern-become) is
    unchanged.
  - The last kept entry's timestamp only feeds the time a chunk was first indexed, which is left to
    [792a7 memory-query](./792a_product-spec.md#792a7-added-memory-query) ([792d78 Chunk-indexing requirements beyond the legacy text](#792d78-chunk-indexing-requirements-beyond-the-legacy-text)), so its two tests wait for [792c7f1 memory-query product spec](./792c_implementation-plan.md#792c7f1-memory-query-product-spec).
  - ADR-0010 is deleted: its surviving JSONL reader is [792b15a Orientation](./792b_technical-spec.md#792b15a-orientation)'s rationale, and its OpenCode backend
    and composite dispatch are retired ([792d6 Removed features are not documented; enduring absences become MUST NOT](#792d6-removed-features-are-not-documented-enduring-absences-become-must-not)). The c1 S5 row and the reader part of c3 K1 point at
    [792b15 transcript-reader](./792b_technical-spec.md#792b15-added-transcript-reader).
  - [792a5b1d An unreadable named file fails the command](./792a_product-spec.md#792a5b1-uat-scenarios) becomes a Scenario Outline over `--markdown` and `--transcript`, so a transcript that
    cannot be read is covered too.
- **Divergence noted:** a `tool_result` whose content is a list of blocks, the form Claude Code
  records for some tools, matches no summary, so neither the call nor its result appears in the
  chunk. [792a5g Transcript content](./792a_product-spec.md#792a5g-transcript-content) and [792b15c5 Tool-call summaries](./792b_technical-spec.md#792b15c5-tool-call-summaries) state this as current behavior ([792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated)).
- **Rationale:** [792d51 command-line-surface routing requirements arrive with their scenarios](#792d51-command-line-surface-routing-requirements-arrive-with-their-scenarios) and [792d75 wikilink-graph's link, graph and resolution contracts arrive with their scenarios](#792d75-wikilink-graphs-link-graph-and-resolution-contracts-arrive-with-their-scenarios): requirements arrive with the product scenarios they support,
  and a test-only entry point is not a component's contract.
- **Alternatives:** count the `Strip` tests as verifying the shared cleaning helpers. Rejected
  because production reaches those helpers only through the summary mode, which has its own tests.

## 792d82 source-chunking and the stale byte budget

- **Choice:** [792b16 source-chunking](./792b_technical-spec.md#792b16-added-source-chunking) specifies section and turn boundaries, anchors, the line-packing split and the
  40-character floor. The c2 ingest flowchart's chunk step said chunking was "byte-budget capped";
  no budget applies, because ingestion reads every transcript whole (`ingestBudgetBytes` is 0), so
  the clause is dropped and the step points at [792b16 source-chunking](./792b_technical-spec.md#792b16-added-source-chunking). [792a5h Chunk boundaries and IDs](./792a_product-spec.md#792a5h-chunk-boundaries-and-ids) gains scenario [792a5h1d A long section is split under one anchor](./792a_product-spec.md#792a5h1-uat-scenarios), because its
  split branch had no UAT scenario. An oversized section is packed line by line, not paragraph by
  paragraph as `TestChunkMarkdownSplitsOversizedSectionOnParagraphs`' name says; blank lines are
  dropped from the pieces.
- **Rationale:** [792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated); the product guide requires a scenario for every mandatory branch.
- **Alternatives:** none considered.

## 792d83 ingestion-source-discovery scope and the tests it leaves

- **Choice:**
  - [792b17 ingestion-source-discovery](./792b_technical-spec.md#792b17-added-ingestion-source-discovery) specifies how sources are found: named files, Pi directories, `--sweep` roots, the
    `--auto` roots with their exclusions and depths, the throwaway-path drop, the sweep
    configuration, the session-log directory, the walk and the source filter. How a source's read
    failure is handled, Rule B, deduplication and everything after are [792b18 chunk-index-store](./792b_technical-spec.md#792b18-added-chunk-index-store)'s
    ([792c7d5 chunk-index-store tech spec](./792c_implementation-plan.md#792c7d5-chunk-index-store-tech-spec)), so the tests of those (the `TestSweep*` tests, the `TestIngest*` dedup tests and
    `TestPiSessionsReadErrorPropagates`) wait for [792c7d5 chunk-index-store tech spec](./792c_implementation-plan.md#792c7d5-chunk-index-store-tech-spec).
  - The c2 C2→S5 row is ported whole and points at [792b17 ingestion-source-discovery](./792b_technical-spec.md#792b17-added-ingestion-source-discovery), [792b15 transcript-reader](./792b_technical-spec.md#792b15-added-transcript-reader) and [792a5i Incremental re-indexing](./792a_product-spec.md#792a5i-incremental-re-indexing). Its "byte-capped with
    continuation signalling" clause is stale ([792d82 source-chunking and the stale byte budget](#792d82-source-chunking-and-the-stale-byte-budget)) and is dropped. `ingest-auto-sweep` keeps only
    its zero-record residue for [792c7d5 chunk-index-store tech spec](./792c_implementation-plan.md#792c7d5-chunk-index-store-tech-spec).
  - `TestIngestPrefersPhysicallyCloserAncestorAcrossTypes` checks the ancestor depth through
    deduplication, so [792c7d5 chunk-index-store tech spec](./792c_implementation-plan.md#792c7d5-chunk-index-store-tech-spec) gives it a [792b18 chunk-index-store](./792b_technical-spec.md#792b18-added-chunk-index-store) scenario with [792b17c3a6 Ancestor depth is the physical distance](./792b_technical-spec.md#792b17c3a-scenarios) `split
    deferred`. `TestPiSessionsRespectsBudget` tests no budget: it checks that one Pi session is
    indexed, which is [792b17c1a2 Every session file of each Pi directory is a source](./792b_technical-spec.md#792b17c1a-scenarios).
- **Rationale:** [792d30 Components are migrated once and extended later](#792d30-components-are-migrated-once-and-extended-later), and one owner per requirement.
- **Alternatives:** put the read-failure policy here, since it depends on whether a source was
  named. Rejected because the read and its failure happen in the index's reconciliation.

## 792d84 chunk-index-store scope, legacy removal and the tests it leaves

- **Choice:**
  - [792b18 chunk-index-store](./792b_technical-spec.md#792b18-added-chunk-index-store) specifies the index layout, the locked reconciliation, change detection, read and write
    failures, the append-only rebuild, grouping and precedence, copy eviction and its coverage gate,
    Rule B, the three prune modes, index listing and the chunk-ID set that `amend --chunk-source`
    checks ([792a4e1a Chunk sources are merged without duplicates](./792a_product-spec.md#792a4e1-uat-scenarios), [792a4f1a An invalid amend argument is refused before any write](./792a_product-spec.md#792a4f1-uat-scenarios)). The time a chunk was first indexed and its backfill wait for
    [792a7 memory-query](./792a_product-spec.md#792a7-added-memory-query) ([792d78 Chunk-indexing requirements beyond the legacy text](#792d78-chunk-indexing-requirements-beyond-the-legacy-text)), and so does `show-chunk`'s lookup, which only [792a7 memory-query](./792a_product-spec.md#792a7-added-memory-query)'s scenarios
    use; their tests (`TestIngestMarkdownSetsIngestedAtFromNow`,
    `TestIngestTranscriptSetsIngestedAtFromPerRowTimestamp`, `TestLoadPriorRecordsPreservesIngestedAt`,
    `TestMergeAppendBackfillsIngestedAtFromManifestMtime`, `TestMergeAppendPreservesIngestedAtOnReIngest`,
    `chunk/ingestedat_test`, `show_chunk_test` and the four `TestChunkQuery*` tests in
    `ingest_integration_test`) stay unrenamed.
  - Legacy removal: ADR-0021 keeps only decision 9 for [792c7j1 install-and-update product spec](./792c_implementation-plan.md#792c7j1-install-and-update-product-spec), behind a pointer; the c2 ingest
    flowchart and its source list are deleted; c1 R4, c3 K1, K12 and their diagram nodes point at
    [792b15 transcript-reader](./792b_technical-spec.md#792b15-added-transcript-reader)–[792b18 chunk-index-store](./792b_technical-spec.md#792b18-added-chunk-index-store); the c3 ingest data contract keeps its skill half and points at [792a5i Incremental re-indexing](./792a_product-spec.md#792a5i-incremental-re-indexing) for the
    binary half; the staleness sentence of c2's cross-level note points at [792b18 chunk-index-store](./792b_technical-spec.md#792b18-added-chunk-index-store).
    `ingest-auto-sweep` and `ingest-prune-detach` are deleted; `ingest-chunk-dedup` keeps the
    `update` notice for [792c7j1 install-and-update product spec](./792c_implementation-plan.md#792c7j1-install-and-update-product-spec); `vault-concurrency-safe-writes` keeps only the vocabulary lock
    requirement for [792c7e3 vocabulary-engine tech spec](./792c_implementation-plan.md#792c7e3-vocabulary-engine-tech-spec). GLOSSARY and ROADMAP citations of the dedup parts of ADR-0021 point at
    [792b18 chunk-index-store](./792b_technical-spec.md#792b18-added-chunk-index-store).
  - Invariants M1–M3 (section E) are deleted, not ported: no marker exists, and no byte budget
    applies to ingestion ([792d82 source-chunking and the stale byte budget](#792d82-source-chunking-and-the-stale-byte-budget)), so M3's "one source cannot starve another" has nothing to enforce.
  - `TestRawForRebuild` covers reuse, a fresh read, a tolerated vanished swept source and a failing
    named one, so it takes [792b18c4a1 A vanished swept source is skipped and a named one fails](./792b_technical-spec.md#792b18c4a-scenarios) with [792b18c7a6 A copy is promoted when its preferred twin goes away](./792b_technical-spec.md#792b18c7a-scenarios) `split deferred`;
    `TestIngestPrefersPhysicallyCloserAncestorAcrossTypes` takes [792b18c6a5 A nearer .pi copy beats a farther .claude copy](./792b_technical-spec.md#792b18c6a-scenarios) with [792b17c3a6 Ancestor depth is the physical distance](./792b_technical-spec.md#792b17c3a-scenarios) `split
    deferred` ([792d83 ingestion-source-discovery scope and the tests it leaves](#792d83-ingestion-source-discovery-scope-and-the-tests-it-leaves)). `TestIngestDepsReadTranscriptReadsViaFS` checks only that the composed reader
    reads through the injected filesystem, so it verifies no scenario.
- **Rationale:** [792d30 Components are migrated once and extended later](#792d30-components-are-migrated-once-and-extended-later), [792d51 command-line-surface routing requirements arrive with their scenarios](#792d51-command-line-surface-routing-requirements-arrive-with-their-scenarios) and [792d6 Removed features are not documented; enduring absences become MUST NOT](#792d6-removed-features-are-not-documented-enduring-absences-become-must-not).
- **Alternatives:** specify the first-indexed time here as a **MAY**. Rejected because [792a7 memory-query](./792a_product-spec.md#792a7-added-memory-query)'s
  recency channel makes it a contract there.

## 792d85 Mixed sources of controlled-vocabulary are split by owner

- **Choice:** node [792c7e1 controlled-vocabulary product spec](./792c_implementation-plan.md#792c7e1-controlled-vocabulary-product-spec) ports the product behavior of its sources into [792a6 controlled-vocabulary](./792a_product-spec.md#792a6-added-controlled-vocabulary) ([792d31 Each legacy item has one owning node](#792d31-each-legacy-item-has-one-owning-node), [792d50 Technical clauses of a mixed OpenSpec spec stay in place for their component node](#792d50-technical-clauses-of-a-mixed-openspec-spec-stay-in-place-for-their-component-node)):
  - `vault-vocab-lifecycle` keeps a residue requirement with the mechanism clauses (definition
    discrimination, the assignment function, the centroid preference, the trigger function, the
    slug parse, the derivation and the centroids-file format) and their function-named scenarios for
    vocabulary-engine ([792c7e3 vocabulary-engine tech spec](./792c_implementation-plan.md#792c7e3-vocabulary-engine-tech-spec)). Its supersession ride-along requirement is query behavior, not
    vocabulary behavior, so it is re-deferred to memory-query ([792c7f1 memory-query product spec](./792c_implementation-plan.md#792c7f1-memory-query-product-spec)), whose sources now list it.
  - `vault-note-identity`'s CRLF reader clause for vocabulary tagging, removal and refit and its
    "Vocab tagging reads a CRLF note" scenario are ported ([792a6f CRLF notes](./792a_product-spec.md#792a6f-crlf-notes)).
  - ADR-0012's `vocab stats` QA lines and the c3 QA capture residue's `vocab stats` bullet are ported
    ([792a6t Vocabulary statistics](./792a_product-spec.md#792a6t-vocabulary-statistics)); both now point at it.
  - ADR-0019 is ported whole as [792a6c Categorical tags share one list](./792a_product-spec.md#792a6c-categorical-tags-share-one-list)'s rationale, including route's evidence and aggregate notes
    and the drowning remedies, and deleted. Its citations in GLOSSARY, ROADMAP, ADR-0011 and the
    `route-dispatch-evidence` spec point at [792a6c Categorical tags share one list](./792a_product-spec.md#792a6c-categorical-tags-share-one-list); route SKILL.md's three citations wait for
    [792c7p1 route product spec](./792c_implementation-plan.md#792c7p1-route-product-spec), as its catalog says ([792d40 SKILL.md reference edits follow the root rule](#792d40-skillmd-reference-edits-follow-the-root-rule)).
  - The query payload's `refit_pending` field stays with [792a7 memory-query](./792a_product-spec.md#792a7-added-memory-query), and acting on a pending
    refit with the learn skill ([792d32 Skill product specs are derived from the shipped SKILL.md](#792d32-skill-product-specs-are-derived-from-the-shipped-skillmd)).
- **Rationale:** each part has one owner, and a node removes only what it ports.
- **Alternatives:** port the ride-along into [792a6 controlled-vocabulary](./792a_product-spec.md#792a6-added-controlled-vocabulary). Rejected because it describes how `engram query`
  orders its results.

## 792d86 Controlled-vocabulary requirements beyond the legacy text, and their components

- **Choice:**
  - [792a6 controlled-vocabulary](./792a_product-spec.md#792a6-added-controlled-vocabulary) adds requirements that the legacy sources don't state but the code, `--help` and tests
    show, each confirmed against the built binary on a temporary vault: the definition and family
    note shapes and the version bumps ([792a6a Definition notes](./792a_product-spec.md#792a6a-definition-notes), [792a6b The family note](./792a_product-spec.md#792a6b-the-family-note)), the skip cases ([792a6e Tagging is skipped without a vocabulary or an embedding](./792a_product-spec.md#792a6e-tagging-is-skipped-without-a-vocabulary-or-an-embedding)), bootstrap's output,
    floor and failures ([792a6g Bootstrapping a vocabulary](./792a_product-spec.md#792a6g-bootstrapping-a-vocabulary), [792a6h Bootstrap failures](./792a_product-spec.md#792a6h-bootstrap-failures)), propose's output and its missing-family-note warning
    ([792a6i Proposing a term](./792a_product-spec.md#792a6i-proposing-a-term)), the dry-run, naming-request and naming-answer formats and errors ([792a6l Dry run](./792a_product-spec.md#792a6l-dry-run)–[792a6n Naming answers](./792a_product-spec.md#792a6n-naming-answers)),
    the applied-refit line ([792a6o Applying a refit](./792a_product-spec.md#792a6o-applying-a-refit)), the refusal of `--plan` and `--emit-request` ([792a6r Refit takes no term plan](./792a_product-spec.md#792a6r-refit-takes-no-term-plan)), the
    baseline seeding of the trigger ([792a6s Refit trigger](./792a_product-spec.md#792a6s-refit-trigger)), every `vocab stats` line ([792a6t Vocabulary statistics](./792a_product-spec.md#792a6t-vocabulary-statistics)), the backfill's skip
    lines ([792a6u Self-tag backfill](./792a_product-spec.md#792a6u-self-tag-backfill)), and `engram update`'s two vocabulary notices and `--regen-vocab` ([792a6v Update notices](./792a_product-spec.md#792a6v-update-notices),
    [792a6w Migrating an old-format vocabulary](./792a_product-spec.md#792a6w-migrating-an-old-format-vocabulary)).
  - note-amendment ([792b14 note-amendment](./792b_technical-spec.md#792b14-added-note-amendment)) is added to the node's extended components: `amend` and `resituate`
    tag the note they write. [792b12 note-authoring](./792b_technical-spec.md#792b12-added-note-authoring) gains [792b12c6 Vocabulary tagging after a write](./792b_technical-spec.md#792b12c6-vocabulary-tagging-after-a-write) and [792b14 note-amendment](./792b_technical-spec.md#792b14-added-note-amendment) [792b14c11 Vocabulary tagging after a revision](./792b_technical-spec.md#792b14c11-vocabulary-tagging-after-a-revision), the
    caller's ordering of tagging and the refit check after a write ([792d63 Which writer takes which lock is the writer's requirement](#792d63-which-writer-takes-which-lock-is-the-writers-requirement)). [792b2 command-line-surface](./792b_technical-spec.md#792b2-added-command-line-surface),
    [792b6 note-document-format](./792b_technical-spec.md#792b6-added-note-document-format), [792b9 text-embedder](./792b_technical-spec.md#792b9-added-text-embedder) and [792b10 embedding-sidecar](./792b_technical-spec.md#792b10-added-embedding-sidecar) gain mappings only.
  - The dispatch tests in `vocab_commands_test` that assert a [792a6 controlled-vocabulary](./792a_product-spec.md#792a6-added-controlled-vocabulary) scenario end to end
    (`TestTargets_Vocab*`) are renamed by vocabulary-engine ([792c7e3 vocabulary-engine tech spec](./792c_implementation-plan.md#792c7e3-vocabulary-engine-tech-spec)), which examines that file; the
    learn and amend tests that check the skip cases (`TestRunLearn_VocabAssignment_Skips*`,
    `TestRunAmend_VocabAssignment_SkipsWhenSidecarMissing`) go there too, because the skip is the
    tagger's behavior.
- **Rationale:** [792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated), [792d49 Product requirements beyond the OpenSpec text](#792d49-product-requirements-beyond-the-openspec-text) and [792d58 Note-writing requirements beyond the legacy text](#792d58-note-writing-requirements-beyond-the-legacy-text); [792d30 Components are migrated once and extended later](#792d30-components-are-migrated-once-and-extended-later) for the extensions.
- **Alternatives:** port only the OpenSpec text.

## 792d87 Known divergences in the controlled vocabulary

- **Choice:** [792a6 controlled-vocabulary](./792a_product-spec.md#792a6-added-controlled-vocabulary) describes these as they are and does not specify the alternative ([792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated),
  [792d42 Known divergences between docs and code](#792d42-known-divergences-between-docs-and-code)):
  - `engram vocab propose` does not check for an existing definition: proposing a term twice writes
    a second `vocab-<term>-definition` note, and only the first one listed is used as the term.
  - Retirement scrubs links from note bodies and `supersedes:` lists but not from other frontmatter
    fields: a fact whose `object:` links a retired definition keeps that link, and `engram check`
    reports it as dangling. [792a6p1b Retirement leaves no dangling body link](./792a_product-spec.md#792a6p1-uat-scenarios) is stated for body links.
  - The refit check after `engram learn` lists the vault with the starter `README.md`, so its note
    count is one higher than the baseline bootstrap and refit record (a vault with one fact recorded
    a baseline of 2 notes).
  - A bootstrap rewrites `vocab.centroids.json` without each term's origin, so a proposed term loses
    its `proposed` origin and the next refit can retire it ([792a6q Proposed terms survive derivation](./792a_product-spec.md#792a6q-proposed-terms-survive-derivation) holds only until then).
  - `engram vocab <subcommand> --help` prints the `vocab` group's help, not the subcommand's flags.
- **Rationale:** a divergence becomes a later change proposal (vault note 645).
- **Alternatives:** specify the intended behavior with coverage gaps. Rejected because the code does
  not do it.

## 792d88 similarity-clustering scope and the tests it leaves

- **Choice:** [792b19 similarity-clustering](./792b_technical-spec.md#792b19-added-similarity-clustering) specifies cosine distance, the seeded k-means partition, the silhouette score
  and the choice of the number of clusters, which refit's scenarios need. The per-point silhouette
  (`PointSilhouette`), used only by the query to pick cluster representatives, waits for [792a7 memory-query](./792a_product-spec.md#792a7-added-memory-query)
  ([792c7f1 memory-query product spec](./792c_implementation-plan.md#792c7f1-memory-query-product-spec)) with its four tests. `TestInvariant_C1_ClusteringDeterminism` checks determinism through
  `RunQuery` with a phrase-derived seed, and `TestProperty_Cluster_NearDuplicateL2CosineAtLeast095`
  checks a query payload's cosine, so both wait for [792a7 memory-query](./792a_product-spec.md#792a7-added-memory-query) too. Invariant C1 is ported as
  [792b19c2 K-means partition](./792b_technical-spec.md#792b19c2-k-means-partition)'s determinism clause and rationale; the query's own seeding from the phrase is the query
  engine's ([792c7f5 query-engine tech spec](./792c_implementation-plan.md#792c7f5-query-engine-tech-spec)). c3 K8 points at [792b19 similarity-clustering](./792b_technical-spec.md#792b19-added-similarity-clustering) and keeps the query's bound on clustering inputs as a
  residue for [792a7 memory-query](./792a_product-spec.md#792a7-added-memory-query), whose K6 row states the same caps.
- **Rationale:** [792d51 command-line-surface routing requirements arrive with their scenarios](#792d51-command-line-surface-routing-requirements-arrive-with-their-scenarios): requirements arrive with the product scenarios they support.
- **Alternatives:** specify the per-point silhouette now as a **MAY**. Rejected because the query
  makes it a contract there.

## 792d89 vocabulary-engine scope, legacy removal and the tests it leaves

- **Choice:**
  - [792b20 vocabulary-engine](./792b_technical-spec.md#792b20-added-vocabulary-engine) specifies definition recognition, term vectors, term selection, the tag rewrite, tagging a
    written note, the refit check, statistics, definition and family notes with the version,
    bootstrap and its two-pass re-tag, propose, derivation, matching, refit routing, naming
    requests and answers, applying, retirement, the self-tag backfill, the old-format migration and
    the two vault-state checks behind `engram update`'s notices. The question-note helpers in
    `qa.go` and the two detectors in `update.go` are its realization because they answer
    vocabulary questions; how `engram update` runs them and renders their result is
    [792a11 install-and-update](./792a_product-spec.md#792a11-added-install-and-update)'s ([792c7j1 install-and-update product spec](./792c_implementation-plan.md#792c7j1-install-and-update-product-spec)), so `update_test`'s `TestWriteUpdateReport_VocabMigrationHint`,
    `TestWriteUpdateReport_VocabRegenReport`, `TestWriteUpdateReport_VocabSelfTagHint` and
    `TestRunUpdate_RegenVocabFlag_RunsRegenAndUpdatesReport` wait for it.
  - The dispatch tests `TestTargets_Vocab*` take the product IDs of the scenarios they assert
    ([792a6h1a A bad seed is refused](./792a_product-spec.md#792a6h1-uat-scenarios), [792a6i1b Propose in a vault without a family note](./792a_product-spec.md#792a6i1-uat-scenarios), [792a6j1b A vault without structure keeps its vocabulary](./792a_product-spec.md#792a6j1-uat-scenarios), [792a6t1a Stats on an empty vault](./792a_product-spec.md#792a6t1-uat-scenarios); [792d52 Test IDs: product scenario for end-to-end checks, technical scenario otherwise](#792d52-test-ids-product-scenario-for-end-to-end-checks-technical-scenario-otherwise)).
  - Legacy removal: ADR-0024 is ported as the rationale of [792a6j Deriving the vocabulary](./792a_product-spec.md#792a6j-deriving-the-vocabulary)–[792a6s Refit trigger](./792a_product-spec.md#792a6s-refit-trigger) and [792b20 vocabulary-engine](./792b_technical-spec.md#792b20-added-vocabulary-engine), and deleted; its
    "retirement semantics" consequence (demote and record a supersession) is stale since retirement
    deletes ([792a6p Retiring a term](./792a_product-spec.md#792a6p-retiring-a-term)) and is dropped. ADR-0011 keeps its history, with its assignment clause pointing
    at [792b20 vocabulary-engine](./792b_technical-spec.md#792b20-added-vocabulary-engine). The c2 vocab lifecycle flowchart becomes [792b20b Sequences](./792b_technical-spec.md#792b20b-sequences); its query-payload step is
    [792a7 memory-query](./792a_product-spec.md#792a7-added-memory-query)'s (`refit_pending`, now in that node's shipped artifacts) and its learn-skill step
    is learn's ([792d32 Skill product specs are derived from the shipped SKILL.md](#792d32-skill-product-specs-are-derived-from-the-shipped-skillmd)). `vault-concurrency-safe-writes` loses its last requirement (bootstrap and
    refit hold the lock: [792b20c10 Bootstrap](./792b_technical-spec.md#792b20c10-bootstrap), [792b20c15 Refit routing](./792b_technical-spec.md#792b20c15-refit-routing)) and is deleted; `vault-vocab-lifecycle` keeps only the
    ride-along residue for [792c7f1 memory-query product spec](./792c_implementation-plan.md#792c7f1-memory-query-product-spec) ([792d85 Mixed sources of controlled-vocabulary are split by owner](#792d85-mixed-sources-of-controlled-vocabulary-are-split-by-owner)).
  - Remaining tests in its files verify no [792b20 vocabulary-engine](./792b_technical-spec.md#792b20-added-vocabulary-engine) scenario: `TestNewVocabDeps_ClosuresCalled`
    (composition wiring), `TestIdAndDateFromNoteFilename` (a filename helper used by reparenting,
    registration and update, left to [792a10 vault-migration](./792a_product-spec.md#792a10-added-vault-migration) [792c7i1 vault-migration product spec](./792c_implementation-plan.md#792c7i1-vault-migration-product-spec)), and
    `TestVocabDefinitionNote_FloorPromoted` and `TestVocabDefinitionNote_ReturnedWhenOnlyItem` (query
    behavior, left to [792a7 memory-query](./792a_product-spec.md#792a7-added-memory-query) [792c7f1 memory-query product spec](./792c_implementation-plan.md#792c7f1-memory-query-product-spec)). The vocabulary tests in `query_vault_meta_test`,
    `skillfields_survival_test` and `exchangefields_survival_test` belong to [792a7 memory-query](./792a_product-spec.md#792a7-added-memory-query),
    [792a12 skill-runbook-registration](./792a_product-spec.md#792a12-added-skill-runbook-registration) and [792a21 parent-sync](./792a_product-spec.md#792a21-added-parent-sync).
- **Divergences noted ([792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated)):** `engram update --regen-vocab` rewrites and deletes vault files
  without taking the vault lock, unlike every other vocabulary writer; `engram vocab
  tag-definitions` lists the vault before taking the lock.
- **Rationale:** [792d30 Components are migrated once and extended later](#792d30-components-are-migrated-once-and-extended-later), [792d51 command-line-surface routing requirements arrive with their scenarios](#792d51-command-line-surface-routing-requirements-arrive-with-their-scenarios) and [792d63 Which writer takes which lock is the writer's requirement](#792d63-which-writer-takes-which-lock-is-the-writers-requirement).
- **Alternatives:** split the statistics or the migration into their own components. Rejected
  because each is one command over the same definition and tag contracts, with no contract of its own
  that a caller could use apart from the vocabulary.

## 792d90 Mixed sources of memory-query are split by owner

- **Choice:** node [792c7f1 memory-query product spec](./792c_implementation-plan.md#792c7f1-memory-query-product-spec) ports the query behavior of its sources into [792a7 memory-query](./792a_product-spec.md#792a7-added-memory-query) and leaves the rest
  where its owner will find it ([792d31 Each legacy item has one owning node](#792d31-each-legacy-item-has-one-owning-node), [792d50 Technical clauses of a mixed OpenSpec spec stay in place for their component node](#792d50-technical-clauses-of-a-mixed-openspec-spec-stay-in-place-for-their-component-node)):
  - `recall-two-channel-payload`, `recall-matched-note-floor`, `recall-centroid-sampling`,
    `recall-query-timings`, `recall-runbook-surfacing` and `vault-query-model-provenance` are ported
    whole and deleted. Their technical clauses were all behavior the product spec already states, so
    nothing was left for recency-ranking ([792c7f2 recency-ranking tech spec](./792c_implementation-plan.md#792c7f2-recency-ranking-tech-spec)) or query-payload-renderer ([792c7f4 query-payload-renderer tech spec](./792c_implementation-plan.md#792c7f4-query-payload-renderer-tech-spec)) to port; those
    components are written from the code and from their other listed sources.
  - `recall-payload-cuts` keeps only the requirement that the agent reads chunk content before judging
    a cluster, for recall ([792c7m1 recall product spec](./792c_implementation-plan.md#792c7m1-recall-product-spec)). Its lazy-chunks, recent-fill and limit requirements are [792a7s Lazy chunks](./792a_product-spec.md#792a7s-lazy-chunks),
    [792a7o The newest chunks](./792a_product-spec.md#792a7o-the-newest-chunks) and [792a7q Limit](./792a_product-spec.md#792a7q-limit).
  - `runbook-lexical-triggers` keeps three residue requirements: the served `--text` round trip and
    its 2 KB truncation for vault-serving ([792c7s1 vault-serving product spec](./792c_implementation-plan.md#792c7s1-vault-serving-product-spec)), the shim's verbatim `--text` instruction for
    ambient-guidance ([792c7r1 ambient-guidance product spec](./792c_implementation-plan.md#792c7r1-ambient-guidance-product-spec)), and the authoring guidance for trigger cues for [792a13 write-memory](./792a_product-spec.md#792a13-added-write-memory)
    ([792c7l1 write-memory product spec](./792c_implementation-plan.md#792c7l1-write-memory-product-spec)). Their sources now list them. The `triggers` field and its flags were already [792a3 note-writing](./792a_product-spec.md#792a3-added-note-writing)'s
    ([792a3d Runbook red flags and triggers](./792a_product-spec.md#792a3d-runbook-red-flags-and-triggers), [792a3g Argument validation](./792a_product-spec.md#792a3g-argument-validation)) and [792a4 note-revision](./792a_product-spec.md#792a4-added-note-revision)'s ([792a4b Field replacement](./792a_product-spec.md#792a4b-field-replacement)).
  - `vault-note-identity` loses the query clauses of its CRLF reader requirement ([792a7zg CRLF notes](./792a_product-spec.md#792a7zg-crlf-notes)); the
    check and count clauses remain, as a residue, for vault-health ([792c7h1 vault-health product spec](./792c_implementation-plan.md#792c7h1-vault-health-product-spec)) and [792a8 vault-counting](./792a_product-spec.md#792a8-added-vault-counting)
    ([792c7g1 vault-counting product spec](./792c_implementation-plan.md#792c7g1-vault-counting-product-spec)).
  - `vault-vocab-lifecycle` is deleted: its supersession ride-along was its last requirement ([792a7n Superseding notes ride along](./792a_product-spec.md#792a7n-superseding-notes-ride-along)).
  - ADR-0026's retrieval decision is [792a7v Runbooks rank like other notes](./792a_product-spec.md#792a7v-runbooks-rank-like-other-notes). Its lexical-trigger amendment stays for
    runbook-trigger-matching ([792c7f3 runbook-trigger-matching tech spec](./792c_implementation-plan.md#792c7f3-runbook-trigger-matching-tech-spec)), and ADR-0025 for query-engine ([792c7f5 query-engine tech spec](./792c_implementation-plan.md#792c7f5-query-engine-tech-spec)).
- **Rationale:** each part has one owner, and a node removes only what it ports.
- **Alternatives:** keep `recall-two-channel-payload` and `recall-query-timings` for the technical
  nodes. Rejected because the technical contract they imply (a recency channel with a limit, five
  timing stages) is the same behavior the product requirements already state, so a technical node would
  have nothing left to port from them.

## 792d91 Memory-query requirements beyond the legacy text, and the components extended

- **Choice:**
  - [792a7 memory-query](./792a_product-spec.md#792a7-added-memory-query) adds requirements that the legacy sources do not state but the code, `--help` and the
    built binary show, each confirmed on a temporary vault: the query input rule ([792a7a Query input](./792a_product-spec.md#792a7a-query-input)), the payload
    shape and budget fields ([792a7b The payload](./792a_product-spec.md#792a7b-the-payload)), the dropped-sidecar warnings and the no-embeddings failure
    ([792a7c Embeddings of the current model](./792a_product-spec.md#792a7c-embeddings-of-the-current-model)), the per-phrase ranking, floor and cap ([792a7d Matched notes and chunks](./792a_product-spec.md#792a7d-matched-notes-and-chunks)), recency scaling ([792a7f Recency scaling](./792a_product-spec.md#792a7f-recency-scaling)), clusters and
    candidate notes ([792a7g Clusters](./792a_product-spec.md#792a7g-clusters), [792a7h Candidate notes](./792a_product-spec.md#792a7h-candidate-notes)), the content budget ([792a7r Chunk content budget](./792a_product-spec.md#792a7r-chunk-content-budget)), `engram show-chunk` ([792a7t Reading one chunk](./792a_product-spec.md#792a7t-reading-one-chunk)), the
    project filter ([792a7u Project filter](./792a_product-spec.md#792a7u-project-filter)), `engram show` ([792a7x Reading one note](./792a_product-spec.md#792a7x-reading-one-note)), `query-chunks` ([792a7zf Searching chunks alone](./792a_product-spec.md#792a7zf-searching-chunks-alone)), the read-only rule
    ([792a7zh A query changes nothing](./792a_product-spec.md#792a7zh-a-query-changes-nothing)), and the first-indexed time of a chunk ([792a7p When a chunk was first indexed](./792a_product-spec.md#792a7p-when-a-chunk-was-first-indexed)).
  - Component sections extended by this node: [792b2 command-line-surface](./792b_technical-spec.md#792b2-added-command-line-surface) gains [792b2c7 Note lookup](./792b_technical-spec.md#792b2c7-note-lookup) (note lookup) and
    [792b2c8 Chunk lookup](./792b_technical-spec.md#792b2c8-chunk-lookup) (chunk lookup), the routing requirements [792d51 command-line-surface routing requirements arrive with their scenarios](#792d51-command-line-surface-routing-requirements-arrive-with-their-scenarios) deferred to it; [792b13 wikilink-graph](./792b_technical-spec.md#792b13-added-wikilink-graph) gains [792b13c3 Link targets](./792b_technical-spec.md#792b13c3-link-targets)
    (link targets), the parsing contract [792d75 wikilink-graph's link, graph and resolution contracts arrive with their scenarios](#792d75-wikilink-graphs-link-graph-and-resolution-contracts-arrive-with-their-scenarios) deferred to it; [792b18 chunk-index-store](./792b_technical-spec.md#792b18-added-chunk-index-store) gains [792b18c14 First-indexed time](./792b_technical-spec.md#792b18c14-first-indexed-time)
    (first-indexed time); [792b15 transcript-reader](./792b_technical-spec.md#792b15-added-transcript-reader) gains [792b15c6 Last timestamp](./792b_technical-spec.md#792b15c6-last-timestamp) (last timestamp), which was not on this
    node's list but is the other half of the first-indexed time; [792b19 similarity-clustering](./792b_technical-spec.md#792b19-added-similarity-clustering) gains [792b19c5 Per-point silhouette](./792b_technical-spec.md#792b19c5-per-point-silhouette)
    (per-point silhouette, [792d88 similarity-clustering scope and the tests it leaves](#792d88-similarity-clustering-scope-and-the-tests-it-leaves)). [792b9 text-embedder](./792b_technical-spec.md#792b9-added-text-embedder), [792b10 embedding-sidecar](./792b_technical-spec.md#792b10-added-embedding-sidecar) and [792b20 vocabulary-engine](./792b_technical-spec.md#792b20-added-vocabulary-engine) gain
    mappings only.
  - Tests renamed here: the `RunShow` and `RunShowChunk` tests, the `ParseWikilinks` tests, the
    first-indexed-time tests of `ingest_test` and `chunk/ingestedat_test`, the transcript timestamp
    tests and the `PointSilhouette` tests. The tests that exercise `RunQuery` and the helpers behind it
    wait for query-engine ([792c7f5 query-engine tech spec](./792c_implementation-plan.md#792c7f5-query-engine-tech-spec)), which also takes `ingest_integration_test`'s `TestChunkQuery*`
    (they test `query-chunks`), `qa_test`'s `TestIsQueryExcludedKind`, `vocab_test`'s two
    `TestVocabDefinitionNote_*` tests, `supersedes_test`'s `TestBuildSupersedesInverse`,
    `embed_selection_test`'s `TestKindFromContent`, the six query reader tests of
    `crlf_readers_internal_test`, and the two query-level clustering tests ([792d88 similarity-clustering scope and the tests it leaves](#792d88-similarity-clustering-scope-and-the-tests-it-leaves)).
- **Rationale:** [792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated), [792d49 Product requirements beyond the OpenSpec text](#792d49-product-requirements-beyond-the-openspec-text), [792d58 Note-writing requirements beyond the legacy text](#792d58-note-writing-requirements-beyond-the-legacy-text) and [792d86 Controlled-vocabulary requirements beyond the legacy text, and their components](#792d86-controlled-vocabulary-requirements-beyond-the-legacy-text-and-their-components); [792d30 Components are migrated once and extended later](#792d30-components-are-migrated-once-and-extended-later) for the extensions.
- **Alternatives:** port only the OpenSpec text.

## 792d92 Known divergences in memory-query

- **Choice:** [792a7 memory-query](./792a_product-spec.md#792a7-added-memory-query) describes these as they are and does not specify the alternative ([792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated),
  [792d42 Known divergences between docs and code](#792d42-known-divergences-between-docs-and-code)):
  - `--recent-fill 0` means the default of 25, not "off". The OpenSpec scenario "Recent-fill zero
    disables channel" does not match the code; only a negative value turns the channel off, and the
    flag takes a negative number only in the form `--recent-fill=-1`.
  - Recency scaling is live. [792d42 Known divergences between docs and code](#792d42-known-divergences-between-docs-and-code) listed it as inactive (#774). The built binary shows it scaling a
    chunk first indexed 120 days ago to a quarter of its cosine, and a note whose sidecar has
    `last_used` 120 days ago to a quarter. What #774 describes is narrower: a note that has never been
    used is not scaled, because `engram learn` quotes `created`, which the date parser rejects, so
    about 972 of the 1,006 notes in the real vault are unscaled until used. [792a7f Recency scaling](./792a_product-spec.md#792a7f-recency-scaling) states both.
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
  - [792b21 recency-ranking](./792b_technical-spec.md#792b21-added-recency-ranking) specifies the recency factor, a note's age, the creation date read from a note, turn
    positions and the newest chunks. How the factor is applied to note and chunk scores, and the age of
    a chunk, are the [792b24 query-engine](./792b_technical-spec.md#792b24-added-query-engine)'s ([792c7f5 query-engine tech spec](./792c_implementation-plan.md#792c7f5-query-engine-tech-spec)), so `RunQuery`'s three recency tests wait there:
    `TestRunQuery_NoteRecencyDecayRanksFreshLastUsedFirst`,
    `TestRunQuery_RecencyLiftsRecentChunkOverStaleHighCosine` and
    `TestCluster_NewerChunkRanksAboveOlderWithSameVector`.
  - `fillRecencyBand`, `spliceRecent` and `mostRecentlyUsedNoteItems` have no caller outside tests, so
    they are not specified, as with the unwired parts of the transcript reader ([792d81 transcript-reader scope and the tests it leaves](#792d81-transcript-reader-scope-and-the-tests-it-leaves)). Their tests,
    and the `TestSortScoredDescOrdersDescending` test of a sort that production code does not call, are
    listed as no-scenario tests. So is `TestDefaultRecencyParamsSaneDefaults`, which checks a `floor`
    value that only those helpers read, and `TestRecencyEvalDiscriminatingHalfLife`, an in-suite
    experiment that sweeps half-lives and logs the ranks.
  - The six query reader tests of `crlf_readers_internal_test` ([792d91 Memory-query requirements beyond the legacy text, and the components extended](#792d91-memory-query-requirements-beyond-the-legacy-text-and-the-components-extended)) go to the component each tests,
    not all to [792b24 query-engine](./792b_technical-spec.md#792b24-added-query-engine): `TestParseCreatedFromNote_CRLFNote` is [792b21c3a6 A CRLF note is read as its LF form](./792b_technical-spec.md#792b21c3a-scenarios);
    `TestTriggerIndex_CRLFRunbook` goes to runbook-trigger-matching ([792c7f3 runbook-trigger-matching tech spec](./792c_implementation-plan.md#792c7f3-runbook-trigger-matching-tech-spec)) and
    `TestCapRedFlagsForPreview_CRLFNote` to query-payload-renderer ([792c7f4 query-payload-renderer tech spec](./792c_implementation-plan.md#792c7f4-query-payload-renderer-tech-spec)).
  - [792a7 memory-query](./792a_product-spec.md#792a7-added-memory-query) gains the scenarios [792a7f1e A late turn is lifted](./792a_product-spec.md#792a7f1-uat-scenarios) (a late turn is lifted) and [792a7zg1d The creation date of a CRLF note is read](./792a_product-spec.md#792a7zg1-uat-scenarios) to [792a7zg1g The lists in a CRLF note's frontmatter are read](./792a_product-spec.md#792a7zg1-uat-scenarios) (the
    recency date, red-flag preview, kind and frontmatter lists of a CRLF note), which the component
    tests above verify.
- **Rationale:** [792d51 command-line-surface routing requirements arrive with their scenarios](#792d51-command-line-surface-routing-requirements-arrive-with-their-scenarios) and [792d52 Test IDs: product scenario for end-to-end checks, technical scenario otherwise](#792d52-test-ids-product-scenario-for-end-to-end-checks-technical-scenario-otherwise): a test goes to the component whose contract it checks.
- **Alternatives:** rename the band helpers' tests to a MAY requirement. Rejected because no production
  path uses them.

## 792d94 runbook-trigger-matching scope and the tests it leaves

- **Choice:**
  - [792b22 runbook-trigger-matching](./792b_technical-spec.md#792b22-added-runbook-trigger-matching) specifies the trigger index, the whole-word occurrence rule, the list of hits, their
    placement, the trigger rank, the lookup by message alone and the absence of a message. Two
    interaction rules belong to the callers ([792c3c ID Rules](./792c_implementation-plan.md#792c3c-id-rules)): a hit's exemption from the relevance floor, the
    matched-set cap and the matched-note floor belongs to query-engine ([792c7f5 query-engine tech spec](./792c_implementation-plan.md#792c7f5-query-engine-tech-spec)), which adds hits after
    matching, and its exemption from `--limit` to query-payload-renderer ([792c7f4 query-payload-renderer tech spec](./792c_implementation-plan.md#792c7f4-query-payload-renderer-tech-spec)), which splits hits
    off before the cap.
  - The `RunQuery` tests that check the trigger path take technical IDs of [792b22 runbook-trigger-matching](./792b_technical-spec.md#792b22-added-runbook-trigger-matching) ([792d52 Test IDs: product scenario for end-to-end checks, technical scenario otherwise](#792d52-test-ids-product-scenario-for-end-to-end-checks-technical-scenario-otherwise)), because each
    asserts the contract of this component through the whole command; the product scenarios they
    support are in the mapping table. Three tests of `query_runbook_test` go elsewhere:
    `TestRunQuery_TriggerHitDoesNotConsumeLimit`, `TestRunQuery_OversizedRedFlagsListKeepsNewestEntry`
    and `TestRunQuery_RunbookItemContentIncludesRedFlags` to query-payload-renderer ([792c7f4 query-payload-renderer tech spec](./792c_implementation-plan.md#792c7f4-query-payload-renderer-tech-spec)), and
    `TestRunQuery_NoPhrasesAndNoTextIsRejected` and `TestRunQuery_RunbookCompetesInMainMatchedSet` to
    query-engine ([792c7f5 query-engine tech spec](./792c_implementation-plan.md#792c7f5-query-engine-tech-spec)).
  - ADR-0026's 2026-09-20 amendment is replaced by a pointer to [792b22 runbook-trigger-matching](./792b_technical-spec.md#792b22-added-runbook-trigger-matching) and [792a7za A trigger hit comes first](./792a_product-spec.md#792a7za-a-trigger-hit-comes-first). The served `--text`
    clause stays in `runbook-lexical-triggers` for [792a20 vault-serving](./792a_product-spec.md#792a20-added-vault-serving) ([792d90 Mixed sources of memory-query are split by owner](#792d90-mixed-sources-of-memory-query-are-split-by-owner)).
- **Rationale:** [792c3c ID Rules](./792c_implementation-plan.md#792c3c-id-rules) and [792d52 Test IDs: product scenario for end-to-end checks, technical scenario otherwise](#792d52-test-ids-product-scenario-for-end-to-end-checks-technical-scenario-otherwise).
- **Alternatives:** none apply.

## 792d95 query-payload-renderer scope and the tests it leaves

- **Choice:**
  - [792b23 query-payload-renderer](./792b_technical-spec.md#792b23-added-query-payload-renderer) specifies the document and its keys, item labels, the order and the limit, the chunk content
    budget, lazy chunks, the budget block, the stage timings and the red-flag preview. The contents of
    the clusters and of `candidate_l2s`, the default limit of 20 and the default recent-fill count are
    the [792b24 query-engine](./792b_technical-spec.md#792b24-added-query-engine)'s ([792c7f5 query-engine tech spec](./792c_implementation-plan.md#792c7f5-query-engine-tech-spec)): it decides what is in the payload and resolves the flags before the
    renderer sees them. `TestRunQuery_DefaultLimitCapsLargeResultSet` and
    `TestResolveRecentFill_DefaultsAndOverrides` go there, although they sit in the files this node
    examined.
  - The keys that lead a payload with pending offers, and `from_parent` in a merged payload, are
    [792a22 curate](./792a_product-spec.md#792a22-added-curate)'s and [792a21 parent-sync](./792a_product-spec.md#792a21-added-parent-sync)'s, so their tests wait for those nodes ([792d51 command-line-surface routing requirements arrive with their scenarios](#792d51-command-line-surface-routing-requirements-arrive-with-their-scenarios)):
    `TestQueryPayload_PendingOffersHintOmittedWhenFalse`, `TestQueryPayload_PendingOffersHintPresentWhenTrue`,
    `TestQueryPayload_PendingOffersLeadPayload` and `TestPendingOfferCurateInstruction_IsExpectedUpkeep`
    to curate ([792c7u1 curate product spec](./792c_implementation-plan.md#792c7u1-curate-product-spec)), and `TestQueryPayload_NoFromParentKeyWithoutMerge` to parent-sync ([792c7t1 parent-sync product spec](./792c_implementation-plan.md#792c7t1-parent-sync-product-spec)).
  - Timings are split as the code splits them: the [792b24 query-engine](./792b_technical-spec.md#792b24-added-query-engine) takes the readings that end the scan,
    embed, cluster and nominate stages and [792b23c7 Stage timings](./792b_technical-spec.md#792b23c7-stage-timings) owns the block and the final reading, so
    `TestRunQuery_TimingsFlag` is renamed here and exercises the whole run.
  - Tests that run `RunQuery` and assert a renderer requirement take technical IDs ([792d52 Test IDs: product scenario for end-to-end checks, technical scenario otherwise](#792d52-test-ids-product-scenario-for-end-to-end-checks-technical-scenario-otherwise)), as in [792d94 runbook-trigger-matching scope and the tests it leaves](#792d94-runbook-trigger-matching-scope-and-the-tests-it-leaves).
  - The renderer takes `kindFromContent`, so `embed_selection_test`'s `TestKindFromContent` and the CRLF
    kind test are renamed here ([792d91 Memory-query requirements beyond the legacy text, and the components extended](#792d91-memory-query-requirements-beyond-the-legacy-text-and-the-components-extended), [792d93 recency-ranking scope and the tests it leaves](#792d93-recency-ranking-scope-and-the-tests-it-leaves)).
- **Rationale:** [792c3c ID Rules](./792c_implementation-plan.md#792c3c-id-rules), [792d51 command-line-surface routing requirements arrive with their scenarios](#792d51-command-line-surface-routing-requirements-arrive-with-their-scenarios) and [792d52 Test IDs: product scenario for end-to-end checks, technical scenario otherwise](#792d52-test-ids-product-scenario-for-end-to-end-checks-technical-scenario-otherwise).
- **Alternatives:** keep the cluster content in the renderer. Rejected because the clustering and the
  candidate ranking are the [792b24 query-engine](./792b_technical-spec.md#792b24-added-query-engine)'s choices, and the renderer only prints them.

## 792d96 query-engine scope, legacy removal and the tests it leaves

- **Choice:**
  - [792b24 query-engine](./792b_technical-spec.md#792b24-added-query-engine) specifies 24 requirements: input, usable embeddings, note and chunk scoring, the matched set
    and its floor, question notes, chunks with notes, clusters, candidate notes, item order, the project
    filter, the explore half in four requirements, superseding notes, the metadata pass, the newest
    chunks, the read-only rule, the refit flag, the recall mirror, the chunk-only search and the stage
    readings. The pending-offer exclusion and its payload flag are curate's, and the merge with a parent
    vault is [792a21 parent-sync](./792a_product-spec.md#792a21-added-parent-sync)'s, so their tests wait for those nodes ([792d51 command-line-surface routing requirements arrive with their scenarios](#792d51-command-line-surface-routing-requirements-arrive-with-their-scenarios), [792d95 query-payload-renderer scope and the tests it leaves](#792d95-query-payload-renderer-scope-and-the-tests-it-leaves)):
    `TestRunQuery_ExcludesPendingSkillRunbookNote` joins the four pending-offer renderer tests in
    curate ([792c7u1 curate product spec](./792c_implementation-plan.md#792c7u1-curate-product-spec)).
  - The `RunQuery` tests take technical IDs ([792d52 Test IDs: product scenario for end-to-end checks, technical scenario otherwise](#792d52-test-ids-product-scenario-for-end-to-end-checks-technical-scenario-otherwise)) except `TestEngramQuery_F6F91_EndToEnd`, which
    builds the production binary, runs `engram embed apply` and `engram query` as subprocesses and
    asserts the observable payload, so it takes the product scenario [792a7g1a Matches form clusters with a representative](./792a_product-spec.md#792a7g1-uat-scenarios). The two
    `ListIndexes` tests of `query_chunks_test` verify the [792b18 chunk-index-store](./792b_technical-spec.md#792b18-added-chunk-index-store)'s listing contract and take
    [792b18c12a1 Index files are found by listing the directory](./792b_technical-spec.md#792b18c12a-scenarios).
  - Four tests are listed as no-scenario tests: `TestMergeClusterReps_*` in
    `query_synthesize_l2_helpers_test`. `mergeClusterReps` is called with an empty matched set, by
    design ("the representative is agent-decided, not binary-computed"), so no query gives an item the
    provenance `cluster_rep` and the tests verify a path no command reaches. The rank of `cluster_rep`
    is still specified in [792b24c11 Item order](./792b_technical-spec.md#792b24c11-item-order), because `provenanceRankFor` returns it.
  - The legacy removals are ADR-0025 (deleted; its decision and consequences are [792a7i Half of the notes are explored](./792a_product-spec.md#792a7i-half-of-the-notes-are-explored) to [792a7m Explore picks are reported](./792a_product-spec.md#792a7m-explore-picks-are-reported) and
    [792b24c13 Explore budget and allocation](./792b_technical-spec.md#792b24c13-explore-budget-and-allocation) to [792b24c16 Delivering the explore half](./792b_technical-spec.md#792b24c16-delivering-the-explore-half)), c1 Flow: recall's query mechanics (its skill steps stay for recall,
    [792c7m1 recall product spec](./792c_implementation-plan.md#792c7m1-recall-product-spec)), c2 Flow: recall and Flowchart: recall query pipeline, c3 K6, K8's residue, Flow: engram
    query internals, the query payload data contract and the `learn qa` capture residue, and invariant
    R1 ([792b24c22 A note is found by its situation](./792b_technical-spec.md#792b24c22-a-note-is-found-by-its-situation)). References to ADR-0025 in GLOSSARY, ROADMAP, ADR-0011, the taxonomy map and the
    retrieval-probe spec point at [792a7i Half of the notes are explored](./792a_product-spec.md#792a7i-half-of-the-notes-are-explored), [792b24 query-engine](./792b_technical-spec.md#792b24-added-query-engine) and [792b24c10 Candidate notes](./792b_technical-spec.md#792b24c10-candidate-notes).
  - [792a7 memory-query](./792a_product-spec.md#792a7-added-memory-query) gains scenario [792a7d1h A note is found by its own situation](./792a_product-spec.md#792a7d1-uat-scenarios) (a note is found by its situation) for invariant R1.
- **Divergences noted ([792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated)):** the OpenSpec said the c3 K6 candidate list holds "ride-alongs for
  superseded notes", but in the code they are inserted in `items`; `provenanceClusterRep` is unreachable;
  c3's "Dead/test-only surface" paragraph is stale and is left to [792c8a Residual legacy sweep](./792c_implementation-plan.md#792c8a-residual-legacy-sweep).
- **Rationale:** [792d30 Components are migrated once and extended later](#792d30-components-are-migrated-once-and-extended-later), [792d51 command-line-surface routing requirements arrive with their scenarios](#792d51-command-line-surface-routing-requirements-arrive-with-their-scenarios), [792d52 Test IDs: product scenario for end-to-end checks, technical scenario otherwise](#792d52-test-ids-product-scenario-for-end-to-end-checks-technical-scenario-otherwise) and [792d89 vocabulary-engine scope, legacy removal and the tests it leaves](#792d89-vocabulary-engine-scope-legacy-removal-and-the-tests-it-leaves).
- **Alternatives:** split the explore half into its own component. Rejected because its sampling,
  its dedupe and its delivery are steps of one query and share its inputs.

## 792d97 vault-counting scope, and the behaviors the code shows

- **Choice:**
  - [792a8 vault-counting](./792a_product-spec.md#792a8-added-vault-counting) has eight requirements ([792a8a Count notes by attribute value](./792a_product-spec.md#792a8a-count-notes-by-attribute-value) to [792a8h A count changes nothing and is exact](./792a_product-spec.md#792a8h-a-count-changes-nothing-and-is-exact)). The graph build and in-degree that [792d75 wikilink-graph's link, graph and resolution contracts arrive with their scenarios](#792d75-wikilink-graphs-link-graph-and-resolution-contracts-arrive-with-their-scenarios) deferred
    arrive as [792b13c4 Link graph](./792b_technical-spec.md#792b13c4-link-graph) and [792b13c5 In-degree](./792b_technical-spec.md#792b13c5-in-degree), with the link parsing of [792b13c3 Link targets](./792b_technical-spec.md#792b13c3-link-targets) (ported by [792c7f1 memory-query product spec](./792c_implementation-plan.md#792c7f1-memory-query-product-spec)) mapped for the
    fenced-link and `.md` scenarios. The unresolved-link report stays with vault-health ([792c7h1 vault-health product spec](./792c_implementation-plan.md#792c7h1-vault-health-product-spec)).
  - ADR-0018 is deleted. ADR-0007 keeps its status line and its G0 consequence for [792c7h1 vault-health product spec](./792c_implementation-plan.md#792c7h1-vault-health-product-spec); its
    decision is [792b13c4 Link graph](./792b_technical-spec.md#792b13c4-link-graph). The note-reader CRLF clause for `engram count` is out of `vault-note-identity`
    ([792a8d Notes that cannot be counted](./792a_product-spec.md#792a8d-notes-that-cannot-be-counted)), which keeps the `engram check` reader and its scenario for [792a9 vault-health](./792a_product-spec.md#792a9-added-vault-health).
  - [792b3c4 Vault Markdown listing](./792b_technical-spec.md#792b3c4-vault-markdown-listing) gains `Supports:` [792a8a1e Only notes at the vault root are counted](./792a_product-spec.md#792a8a1-uat-scenarios), [792a8b1c An empty vault prints nothing](./792a_product-spec.md#792a8b1-uat-scenarios) (the vault root listing), and [792b25d Product-Scenario Mapping](./792b_technical-spec.md#792b25d-product-scenario-mapping) maps those scenarios
    to it. The `RunCount` tests take technical IDs ([792d52 Test IDs: product scenario for end-to-end checks, technical scenario otherwise](#792d52-test-ids-product-scenario-for-end-to-end-checks-technical-scenario-otherwise)); `TestTargets_CountEmptyVault`, which runs the
    command through dispatch, takes [792a8b1c An empty vault prints nothing](./792a_product-spec.md#792a8b1-uat-scenarios). `TestRunCount_OsDepsReadRealVault` checks only the wiring of
    the production dependencies and is a no-scenario test.
- **Divergences noted ([792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated)):**
  - The OpenSpec example printed `total: 557` for 349, 192 and 8 notes, which does not add up. [792a8a1a A scalar attribute is counted per value](./792a_product-spec.md#792a8a1-uat-scenarios) uses
    `total: 549`.
  - `--backlinks-of` fails when any root note cannot be read, naming the file ([792a8e1j An unreadable note fails the report](./792a_product-spec.md#792a8e1-uat-scenarios)), while `--group-by`
    skips an unreadable note ([792a8d1b An unreadable note is skipped](./792a_product-spec.md#792a8d1-uat-scenarios)).
  - A link with a display part, `[[y|shown]]`, is a target of `y|shown` ([792b13c3a10 A display part stays in the target](./792b_technical-spec.md#792b13c3a-scenarios)), so it is a link to no
    note and does not count toward `y` in Obsidian's terms. `--backlinks-of` does not normalize its
    argument (`y.md`, `[[y]]` and `y|shown` all print `in-degree: 0`), unlike the references of
    `show` and `amend` ([792b13c2 Note references](./792b_technical-spec.md#792b13c2-note-references)).
  - A basename with no note has in-degree 0 whatever links to it ([792a8e1h A link to a missing note counts for no note](./792a_product-spec.md#792a8e1-uat-scenarios)), because the graph drops
    links to missing notes.
  - `--filter` is read only by `--group-by`: with `--backlinks-of` it is ignored, and alone it fails as
    "no mode". Neither is specified as a requirement beyond [792a8f1c A filter alone is not a mode](./792a_product-spec.md#792a8f1-uat-scenarios).
  - c3's "Dead/test-only surface" paragraph still says `BuildGraph` has no production consumer; `engram
    count` uses it. The paragraph is left to [792c8a Residual legacy sweep](./792c_implementation-plan.md#792c8a-residual-legacy-sweep), as in [792d96 query-engine scope, legacy removal and the tests it leaves](#792d96-query-engine-scope-legacy-removal-and-the-tests-it-leaves).
- **Rationale:** [792d30 Components are migrated once and extended later](#792d30-components-are-migrated-once-and-extended-later), [792d51 command-line-surface routing requirements arrive with their scenarios](#792d51-command-line-surface-routing-requirements-arrive-with-their-scenarios), [792d52 Test IDs: product scenario for end-to-end checks, technical scenario otherwise](#792d52-test-ids-product-scenario-for-end-to-end-checks-technical-scenario-otherwise), [792d69 Mixed sources of note-revision are split by owner](#792d69-mixed-sources-of-note-revision-are-split-by-owner) and [792d75 wikilink-graph's link, graph and resolution contracts arrive with their scenarios](#792d75-wikilink-graphs-link-graph-and-resolution-contracts-arrive-with-their-scenarios).
- **Alternatives:** port the link parsing again in [792b25 vault-counter](./792b_technical-spec.md#792b25-added-vault-counter). Rejected because [792b13c3 Link targets](./792b_technical-spec.md#792b13c3-link-targets) already specifies it.

## 792d98 vault-counter scope and the tests it leaves

- **Choice:**
  - [792b25 vault-counter](./792b_technical-spec.md#792b25-added-vault-counter) specifies eight requirements: mode selection, attribute membership counting, filters,
    uncountable notes, the attribute report, the incoming-link report, the independence of the two
    measures, and the absence of writes. The link graph, in-degree and link parsing are
    [792b13 wikilink-graph](./792b_technical-spec.md#792b13-added-wikilink-graph)'s ([792d97 vault-counting scope, and the behaviors the code shows](#792d97-vault-counting-scope-and-the-behaviors-the-code-shows)). The five requirements of the OpenSpec `cli-count` spec map
    to these, and the spec is deleted.
  - `TestRunCount_AbsentAttrBucket` takes [792b25c2a5 A note without the key is absent](./792b_technical-spec.md#792b25c2a-scenarios), the first scenario it verifies, and lists
    [792b25c5a2 The absent line precedes the total](./792b_technical-spec.md#792b25c5a-scenarios) as `split deferred` ([792d10 Test renaming](#792d10-test-renaming)). `TestRunCount_OsDepsReadRealVault` checks only the
    wiring of the production dependencies over a real directory and is a no-scenario test.
- **Rationale:** [792d10 Test renaming](#792d10-test-renaming), [792d30 Components are migrated once and extended later](#792d30-components-are-migrated-once-and-extended-later), [792d52 Test IDs: product scenario for end-to-end checks, technical scenario otherwise](#792d52-test-ids-product-scenario-for-end-to-end-checks-technical-scenario-otherwise) and [792d97 vault-counting scope, and the behaviors the code shows](#792d97-vault-counting-scope-and-the-behaviors-the-code-shows).
- **Alternatives:** a requirement for the production wiring. Rejected because wiring is an injection
  seam, which the technical guide does not let qualify (it is not a stable contract).

## 792d99 vault-health scope, and the behaviors the code shows

- **Choice:**
  - [792a9 vault-health](./792a_product-spec.md#792a9-added-vault-health) has ten requirements ([792a9a The check report](./792a_product-spec.md#792a9a-the-check-report) to [792a9j CRLF notes embed as LF](./792a_product-spec.md#792a9j-crlf-notes-embed-as-lf)). The checks are the ones `engram check` runs: G0, G3,
    M5 and the sidecar schema check, which the code calls S1. `engram embed status` and `engram embed apply`
    are specified in [792b10 embedding-sidecar](./792b_technical-spec.md#792b10-added-embedding-sidecar) as [792b10c10 Embedding census](./792b_technical-spec.md#792b10c10-embedding-census) to [792b10c12 Embedding run](./792b_technical-spec.md#792b10c12-embedding-run) ([792d66 Embedding-sidecar scope, and the partial-model-mismatch gap](#792d66-embedding-sidecar-scope-and-the-partial-model-mismatch-gap)), with their classification by
    [792b10c7 State classification](./792b_technical-spec.md#792b10c7-state-classification), and the unresolved-link list in [792b13 wikilink-graph](./792b_technical-spec.md#792b13-added-wikilink-graph) as [792b13c6 Unresolved links](./792b_technical-spec.md#792b13c6-unresolved-links) ([792d75 wikilink-graph's link, graph and resolution contracts arrive with their scenarios](#792d75-wikilink-graphs-link-graph-and-resolution-contracts-arrive-with-their-scenarios)). The checker's own
    contract is [792b26 vault-invariant-checker](./792b_technical-spec.md#792b26-added-vault-invariant-checker) ([792c7h2 vault-invariant-checker tech spec](./792c_implementation-plan.md#792c7h2-vault-invariant-checker-tech-spec)).
  - ADR-0007 is deleted; its decision was [792b13c4 Link graph](./792b_technical-spec.md#792b13c4-link-graph) ([792d97 vault-counting scope, and the behaviors the code shows](#792d97-vault-counting-scope-and-the-behaviors-the-code-shows)) and its consequence is the rationale of [792a9b Links that should resolve](./792a_product-spec.md#792a9b-links-that-should-resolve).
    The KEYSTONE note and the G0 row of `memory-invariants` and c3's recurring defect shape are ported to
    [792a9b Links that should resolve](./792a_product-spec.md#792a9b-links-that-should-resolve). The `vault-embed-on-write` and `vault-wikilink-resolution` specs are deleted, and so is the
    CRLF requirement of `vault-note-identity`, whose readers are now all ported ([792d69 Mixed sources of note-revision are split by owner](#792d69-mixed-sources-of-note-revision-are-split-by-owner)).
  - Embedding-run tests take technical IDs ([792d52 Test IDs: product scenario for end-to-end checks, technical scenario otherwise](#792d52-test-ids-product-scenario-for-end-to-end-checks-technical-scenario-otherwise)). The `Targets` tests of `check` take product IDs and
    are renamed with [792c7h2 vault-invariant-checker tech spec](./792c_implementation-plan.md#792c7h2-vault-invariant-checker-tech-spec).
- **Divergences noted ([792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated)):**
  - `engram check` runs four checks. The invariants document also lists G2 (no orphan notes), E1 (every
    note has a sidecar, a FAIL class there), M6 and M7, which the checker does not run: G2 is not
    reported at all, and a missing sidecar is a WARN under S1. The document's S1 (the skill layer does not
    edit the vault) is a different invariant from the checker's S1 (sidecar schema).
  - The rigor plan wired the checker as a `targ` gate; `dev/targs.go` does not run it, and `check-full`
    does not call it.
  - S1 prints its PASS line even when notes lack a sidecar and it has just printed the WARN line.
  - A link is a G0 violation only when its target is a bare ID equal to an existing note's leading ID; a
    bare ID matching no note is a G3 warning.
  - `engram embed apply` exits zero when notes fail to embed or write, and always embeds a note whose
    sidecar is broken, whatever the selection flag.
  - The 2026-06 census figures (155 of 183 links dropped, 138 of 171 notes without a link) describe the
    vault of that date; they are kept in [792a9b Links that should resolve](./792a_product-spec.md#792a9b-links-that-should-resolve)'s rationale as history and are not a claim about now.
- **Rationale:** [792d30 Components are migrated once and extended later](#792d30-components-are-migrated-once-and-extended-later), [792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated), [792d52 Test IDs: product scenario for end-to-end checks, technical scenario otherwise](#792d52-test-ids-product-scenario-for-end-to-end-checks-technical-scenario-otherwise), [792d66 Embedding-sidecar scope, and the partial-model-mismatch gap](#792d66-embedding-sidecar-scope-and-the-partial-model-mismatch-gap), [792d69 Mixed sources of note-revision are split by owner](#792d69-mixed-sources-of-note-revision-are-split-by-owner), [792d75 wikilink-graph's link, graph and resolution contracts arrive with their scenarios](#792d75-wikilink-graphs-link-graph-and-resolution-contracts-arrive-with-their-scenarios) and [792d97 vault-counting scope, and the behaviors the code shows](#792d97-vault-counting-scope-and-the-behaviors-the-code-shows).
- **Alternatives:** specify G2 and E1 as requirements of the checker. Rejected because the checker does
  not implement them ([792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated)); they are noted as divergences instead.

## 792d100 vault-invariant-checker scope and the tests it leaves

- **Choice:**
  - [792b26 vault-invariant-checker](./792b_technical-spec.md#792b26-added-vault-invariant-checker) specifies six requirements: running the checks, link resolution (G0 and G3), situation presence
    (M5), the sidecar schema (S1), the example lists, and the absence of writes. It specifies what
    `engram check` runs ([792d99 vault-health scope, and the behaviors the code shows](#792d99-vault-health-scope-and-the-behaviors-the-code-shows)): G2 (orphans) and the legacy E1 as a FAIL are not in it, because the code
    does not run them. `memory-invariants` points G2 at this entry.
  - `memory-system-rigor` is deleted. Its rationale for an executable checker is [792b26a Orientation](./792b_technical-spec.md#792b26a-orientation)'s rationale, with
    the `targ` gate recorded as unrealized ([792d99 vault-health scope, and the behaviors the code shows](#792d99-vault-health-scope-and-the-behaviors-the-code-shows)); the rest of the file is the process history of a
    2026-06 effort, with its retired invariants.
  - The invariants document keeps its other rows for [792c8a Residual legacy sweep](./792c_implementation-plan.md#792c8a-residual-legacy-sweep). The severity model, which named M2-segments,
    M4, M6, M7, M8, S1 and S2 beside the checker's own, is dropped; a `FAIL` and a `WARN` are defined in
    [792a9a The check report](./792a_product-spec.md#792a9a-the-check-report) and [792b26a Orientation](./792b_technical-spec.md#792b26a-orientation).
  - `TestTargets_CheckEmptyVault` runs the command through dispatch and takes [792a9a1f An empty vault passes](./792a_product-spec.md#792a9a1-uat-scenarios) ([792d52 Test IDs: product scenario for end-to-end checks, technical scenario otherwise](#792d52-test-ids-product-scenario-for-end-to-end-checks-technical-scenario-otherwise)). Each
    `RunCheck` test that asserts the error identity takes the [792b26c1 Running the checks](./792b_technical-spec.md#792b26c1-running-the-checks) scenario it also verifies, and lists
    the others as `split deferred` ([792d10 Test renaming](#792d10-test-renaming)).
- **Rationale:** [792d10 Test renaming](#792d10-test-renaming), [792d30 Components are migrated once and extended later](#792d30-components-are-migrated-once-and-extended-later), [792d52 Test IDs: product scenario for end-to-end checks, technical scenario otherwise](#792d52-test-ids-product-scenario-for-end-to-end-checks-technical-scenario-otherwise), [792d99 vault-health scope, and the behaviors the code shows](#792d99-vault-health-scope-and-the-behaviors-the-code-shows).
- **Alternatives:** specify G2 as a requirement. Rejected because no code or test supports it ([792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated)).

## 792d101 vault-migration scope, and the behaviors the code shows

- **Choice:**
  - [792a10 vault-migration](./792a_product-spec.md#792a10-added-vault-migration) has twelve requirements ([792a10a The flat-vault notice](./792a_product-spec.md#792a10a-the-flat-vault-notice) to [792a10l What a backfill keeps](./792a_product-spec.md#792a10l-what-a-backfill-keeps)): the flat-vault notice, opt-in re-parenting, the
    derive payload, refused answers, renaming and rewriting, the chunk-index step, what is left to judge, the
    preview, the notes a rename will not touch, the identity-backfill notice, the stamping and what the
    backfill keeps.
  - The three notice and routing scenarios [792a10a1d The notice names the answering procedure](./792a_product-spec.md#792a10a1-uat-scenarios), [792a10b1a Plain update never re-parents](./792a_product-spec.md#792a10b1-uat-scenarios) and [792a10b1b The flag runs nothing else](./792a_product-spec.md#792a10b1-uat-scenarios), and the notice output of
    [792a10a1a Only top-level notes](./792a_product-spec.md#792a10a1-uat-scenarios) to [792a10a1c An empty vault has no notice](./792a_product-spec.md#792a10a1-uat-scenarios), concern `engram update`'s report writer and its flag dispatch. They are left
    unmapped to wait for self-installer ([792c7j2 self-installer tech spec](./792c_implementation-plan.md#792c7j2-self-installer-tech-spec)), as [792d51 command-line-surface routing requirements arrive with their scenarios](#792d51-command-line-surface-routing-requirements-arrive-with-their-scenarios) and [792d61 note-document-format's edit, CRLF and field-survival contracts arrive with their scenarios](#792d61-note-document-formats-edit-crlf-and-field-survival-contracts-arrive-with-their-scenarios) did for other later owners. The
    flat-vault detection is [792b27 note-reparenting](./792b_technical-spec.md#792b27-added-note-reparenting)'s ([792b27c1 Flat-vault detection](./792b_technical-spec.md#792b27c1-flat-vault-detection)), and the missing-identity detection is
    [792b11 note-provenance](./792b_technical-spec.md#792b11-added-note-provenance)'s ([792b11c6 Notes a backfill could stamp](./792b_technical-spec.md#792b11c6-notes-a-backfill-could-stamp)).
  - The backfill is specified in note-provenance ([792b11c5 Repository for a backfill](./792b_technical-spec.md#792b11c5-repository-for-a-backfill) to [792b11c7 Identity backfill](./792b_technical-spec.md#792b11c7-identity-backfill)), as [792d67 note-provenance covers detection and the first-write stamp](#792d67-note-provenance-covers-detection-and-the-first-write-stamp) said, and the ID and date
    of a filename in luhmann-addressing ([792b7c6 ID and date from a filename](./792b_technical-spec.md#792b7c6-id-and-date-from-a-filename)). [792b19 similarity-clustering](./792b_technical-spec.md#792b19-added-similarity-clustering) is not extended: the derive step
    takes sidecar vectors and measures them with [792b10c8 Cosine similarity](./792b_technical-spec.md#792b10c8-cosine-similarity), and uses no clustering. The link rewrite is its own
    pattern over note text and is not [792b13 wikilink-graph](./792b_technical-spec.md#792b13-added-wikilink-graph)'s parser, so [792b13 wikilink-graph](./792b_technical-spec.md#792b13-added-wikilink-graph) gains no mapping.
  - `update-flat-vault-luhmann-notice` keeps its detection requirement, `update-reparent-luhmann-batch` its
    derive and apply parameters, for [792c7i2 note-reparenting tech spec](./792c_implementation-plan.md#792c7i2-note-reparenting-tech-spec). `vault-note-identity` loses its two backfill requirements.
- **Divergences noted ([792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated)):**
  - `engram update` cannot run without network access and a model in this environment (it clones the
    repository and builds, and the model is a git-lfs pointer in the clone), so the notices and the
    backfill flag were not run against the built binary. `--reparent-luhmann` was, on a temporary vault with
    a temporary HOME: derive, dry-run, apply, the stale refusal, malformed answers and a missing file.
  - `--reparent-luhmann` runs without a harness or a Go toolchain, because it returns before the install.
  - The derive payload's `next_command` is HTML-escaped by the JSON encoder (`\u003c`) in the output.
  - A note with no sidecar, or a vector it cannot read, is silently left out of the candidates.
- **Rationale:** [792d30 Components are migrated once and extended later](#792d30-components-are-migrated-once-and-extended-later), [792d51 command-line-surface routing requirements arrive with their scenarios](#792d51-command-line-surface-routing-requirements-arrive-with-their-scenarios), [792d61 note-document-format's edit, CRLF and field-survival contracts arrive with their scenarios](#792d61-note-document-formats-edit-crlf-and-field-survival-contracts-arrive-with-their-scenarios), [792d67 note-provenance covers detection and the first-write stamp](#792d67-note-provenance-covers-detection-and-the-first-write-stamp) and [792d69 Mixed sources of note-revision are split by owner](#792d69-mixed-sources-of-note-revision-are-split-by-owner).
- **Alternatives:** map the notice scenarios to the detection requirements and leave the report writer
  unspecified. Rejected because the notice is what the user sees, and its owner is the next node of
  [792a11 install-and-update](./792a_product-spec.md#792a11-added-install-and-update).

## 792d102 note-reparenting scope and the tests it leaves

- **Choice:**
  - [792b27 note-reparenting](./792b_technical-spec.md#792b27-added-note-reparenting) specifies ten requirements: flat-vault detection, candidate selection, the derive output,
    answers validation, the rename map, the rename and link rewrite, rename safety, the sidecars of rewritten
    notes, the apply flow and the preview. Its derive and apply parameters (top 3, floor 0.75, ingest then
    prune) are in [792b27c2 Candidate selection](./792b_technical-spec.md#792b27c2-candidate-selection) and [792b27c9 Apply flow](./792b_technical-spec.md#792b27c9-apply-flow), so the OpenSpec file that kept them is deleted.
  - `TestVaultHasOnlyTopLevelNotes` verifies the detection and takes [792b27c1a1 Every note is top-level](./792b_technical-spec.md#792b27c1a-scenarios). The notice itself,
    `TestWriteUpdateReport_LuhmannBranchingHint`, and the command's routing are [792a11 install-and-update](./792a_product-spec.md#792a11-added-install-and-update)'s.
  - Tests of the rename and rewrite step that run it for skill adoption (`TestAdoptSkillNote_ConvertsCRLFNote`)
    belong to [792a12 skill-runbook-registration](./792a_product-spec.md#792a12-added-skill-runbook-registration), which extends [792b27c6 Rename and link rewrite](./792b_technical-spec.md#792b27c6-rename-and-link-rewrite) and [792b27c7 Rename safety](./792b_technical-spec.md#792b27c7-rename-safety).
  - `TestRenameAndRewriteReferences_CRLFProperty` checks several rename-safety scenarios in one property and
    takes the first, [792b27c7a1 An anchored luhmann value is refused untouched](./792b_technical-spec.md#792b27c7a-scenarios).
- **Rationale:** [792d10 Test renaming](#792d10-test-renaming), [792d30 Components are migrated once and extended later](#792d30-components-are-migrated-once-and-extended-later), [792d52 Test IDs: product scenario for end-to-end checks, technical scenario otherwise](#792d52-test-ids-product-scenario-for-end-to-end-checks-technical-scenario-otherwise) and [792d101 vault-migration scope, and the behaviors the code shows](#792d101-vault-migration-scope-and-the-behaviors-the-code-shows).
- **Alternatives:** none beyond [792d101 vault-migration scope, and the behaviors the code shows](#792d101-vault-migration-scope-and-the-behaviors-the-code-shows).

## 792d103 install-and-update scope, ownership of the update run, and the behaviors the code shows

- **Choice:**
  - [792a11 install-and-update](./792a_product-spec.md#792a11-added-install-and-update) has fifteen requirements ([792a11a Installing from a local clone or a fresh clone](./792a_product-spec.md#792a11a-installing-from-a-local-clone-or-a-fresh-clone) to [792a11o The update options](./792a_product-spec.md#792a11o-the-update-options)). The notices that `engram update` prints about
    the vault belong to the capabilities whose state they report (vocabulary [792a6 controlled-vocabulary](./792a_product-spec.md#792a6-added-controlled-vocabulary), vault ID [792a2i Uncommitted vault ID notice](./792a_product-spec.md#792a2i-uncommitted-vault-id-notice),
    Luhmann branching and missing origin [792a10 vault-migration](./792a_product-spec.md#792a10-added-vault-migration)), the pending-offer notice to curate ([792c7u1 curate product spec](./792c_implementation-plan.md#792c7u1-curate-product-spec)), the
    outbox notice to parent-sync ([792c7t1 parent-sync product spec](./792c_implementation-plan.md#792c7t1-parent-sync-product-spec)) and the registration step and its error line to
    skill-runbook-registration ([792c7k1 skill-runbook-registration product spec](./792c_implementation-plan.md#792c7k1-skill-runbook-registration-product-spec)). The guidance activation hints (the `@import` line to add, the
    refreshed line, the nudge) belong to ambient-guidance ([792c7r1 ambient-guidance product spec](./792c_implementation-plan.md#792c7r1-ambient-guidance-product-spec)), so `update_test`'s
    `TestWriteUpdateReport_GuidanceActivationHint` waits for it; the canonical-path suffix on Claude
    guidance lines is part of the deploy ([792a11m1d The report states the canonical path](./792a_product-spec.md#792a11m1-uat-scenarios)).
  - Per [792d101 vault-migration scope, and the behaviors the code shows](#792d101-vault-migration-scope-and-the-behaviors-the-code-shows), self-installer ([792b28 self-installer](./792b_technical-spec.md#792b28-added-self-installer)) owns the update run: source acquisition and install, the downgrade
    gate, the hand-off, flag routing, the post-sync checks and the report's header, notices and exit status.
    harness-deploy-sync ([792b29 harness-deploy-sync](./792b_technical-spec.md#792b29-added-harness-deploy-sync)) owns harness detection, the deploy set, the engram-owned root, the links,
    the cleanup, the adoption, the guidance gating and the lines the report shows for each harness.
  - [792b4 subprocess-runner](./792b_technical-spec.md#792b4-added-subprocess-runner) gains [792b4c4 Run with the caller's streams](./792b_technical-spec.md#792b4c4-run-with-the-callers-streams) (the spawn that shares the caller's streams) and [792b18 chunk-index-store](./792b_technical-spec.md#792b18-added-chunk-index-store)
    gains [792b18c15 Reporting a prune backlog](./792b_technical-spec.md#792b18c15-reporting-a-prune-backlog) (the backlog check the two chunk-index notices use). [792b2 command-line-surface](./792b_technical-spec.md#792b2-added-command-line-surface) gains
    nothing: the `update` command's flags and routing are the update run's, and the generic result
    mapping it already specifies needs no new requirement.
  - A product scenario about the report's text maps to the component that renders it, and a test that
    runs the command to assert such a scenario takes the product ID ([792d52 Test IDs: product scenario for end-to-end checks, technical scenario otherwise](#792d52-test-ids-product-scenario-for-end-to-end-checks-technical-scenario-otherwise)): `TestRunCommand_EndToEnd`
    takes [792a11a1b Outside a checkout the repository is cloned](./792a_product-spec.md#792a11a1-uat-scenarios).
- **Divergences noted ([792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated)):**
  - `engram update` cannot run its install in this environment (the clone's model is a git-lfs pointer),
    so the install and the hand-off were not run against a built binary. With a temporary HOME, vault and
    XDG directories, `engram update --dry-run` was run from the repository, and the sync phase was run
    through the loop-guard sentinel (`ENGRAM_UPDATE_REEXEC=1 engram update`) over a Claude Code
    directory that held a copied skill and an unrelated skill: the copy was adopted, the unrelated skill
    was listed as unmanaged and left, the root and its marker were created, and the links were correct.
  - The walk from the working directory stops at the first `go.mod`, even one for another module, and then
    installs from a clone.
  - A dry run from outside a checkout still removes and re-creates the scratch clone directory and runs
    `git clone` there, then checks the model; only `go install` is skipped.
  - `Report.BinaryVersion` is the source revision in both modes, although its comment says local mode
    leaves it empty, so the binary line reads `engram <short SHA>`. The unit tests of the report use fixtures
    that predate the clone build (`go install …@latest`), which the code no longer prints.
  - The registration offers are printed before the report when the stdin is not a terminal or on a dry
    run, because registration writes to the output directly.
  - Manifest deploy mode is implemented and tested, but no shipped harness uses it: both Claude Code and Pi
    run in symlink mode. It is specified by [792b29 harness-deploy-sync](./792b_technical-spec.md#792b29-added-harness-deploy-sync) as an alternative route to [792a11i1a A removed skill disappears](./792a_product-spec.md#792a11i1-uat-scenarios).
- **Rationale:** [792d30 Components are migrated once and extended later](#792d30-components-are-migrated-once-and-extended-later), [792d51 command-line-surface routing requirements arrive with their scenarios](#792d51-command-line-surface-routing-requirements-arrive-with-their-scenarios), [792d52 Test IDs: product scenario for end-to-end checks, technical scenario otherwise](#792d52-test-ids-product-scenario-for-end-to-end-checks-technical-scenario-otherwise), [792d101 vault-migration scope, and the behaviors the code shows](#792d101-vault-migration-scope-and-the-behaviors-the-code-shows) and [792d102 note-reparenting scope and the tests it leaves](#792d102-note-reparenting-scope-and-the-tests-it-leaves).
- **Alternatives:** put the report's notices in the capabilities that detect them. Rejected because the
  report's order and silence rules are one contract, and the notice scenarios of [792a10 vault-migration](./792a_product-spec.md#792a10-added-vault-migration) already wait for
  the update run's owner.

## 792d104 self-installer scope, and the behaviors the code shows

- **Choice:**
  - [792b28 self-installer](./792b_technical-spec.md#792b28-added-self-installer) specifies fifteen requirements. It owns the report and its notices ([792d101 vault-migration scope, and the behaviors the code shows](#792d101-vault-migration-scope-and-the-behaviors-the-code-shows), [792d103 install-and-update scope, ownership of the update run, and the behaviors the code shows](#792d103-install-and-update-scope-ownership-of-the-update-run-and-the-behaviors-the-code-shows)): the text
    and detection of each notice stay with the capability that owns the state, while [792b28c13 Notices](./792b_technical-spec.md#792b28c13-notices) owns that
    each is printed exactly when its flag is set, in a fixed order. The vault-ID notice keeps its scenario in
    [792b5c13a2 The update report carries the notice only when flagged](./792b_technical-spec.md#792b5c13a-scenarios), and the pending-offer, outbox and registration-error lines keep theirs for [792a22 curate](./792a_product-spec.md#792a22-added-curate), [792a21 parent-sync](./792a_product-spec.md#792a21-added-parent-sync)
    and [792a12 skill-runbook-registration](./792a_product-spec.md#792a12-added-skill-runbook-registration).
  - The tests of the vocabulary, Luhmann, identity and chunk-index notices that [792d89 vocabulary-engine scope, legacy removal and the tests it leaves](#792d89-vocabulary-engine-scope-legacy-removal-and-the-tests-it-leaves) and [792d101 vault-migration scope, and the behaviors the code shows](#792d101-vault-migration-scope-and-the-behaviors-the-code-shows) deferred
    take the [792b28 self-installer](./792b_technical-spec.md#792b28-added-self-installer) IDs: they exercise the report writer. The product scenarios [792a6v1a An old-format vault is flagged](./792a_product-spec.md#792a6v1-uat-scenarios) to [792a6w1c Nothing to migrate](./792a_product-spec.md#792a6w1-uat-scenarios),
    [792a10a1a Only top-level notes](./792a_product-spec.md#792a10a1-uat-scenarios) to [792a10b1b The flag runs nothing else](./792a_product-spec.md#792a10b1-uat-scenarios), [792a10j1a Notes missing identity are noticed](./792a_product-spec.md#792a10j1-uat-scenarios), [792a10j1b No notice when nothing is missing](./792a_product-spec.md#792a10j1-uat-scenarios), [792a10k1a The current environment is stamped](./792a_product-spec.md#792a10k1-uat-scenarios) and [792a10k1f A dry run writes nothing](./792a_product-spec.md#792a10k1-uat-scenarios) are mapped here as well as to the
    detecting component.
  - `TestInvariant_U1_SkillsSrcMissingSentinel` takes [792b28c15a6 A missing skills source keeps its typed error](./792b_technical-spec.md#792b28c15a-scenarios) (the typed error survives the report tail);
    the same sentinel's source, the deploy-set planning, is [792b29 harness-deploy-sync](./792b_technical-spec.md#792b29-added-harness-deploy-sync)'s. `update_test`'s fixtures
    (`stubCommander`, `liveUpdateFS`) and the generated mock stay unrenamed helpers.
- **Divergences noted ([792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated)):**
  - The handed-off run resolves its source again. In remote mode it clones the repository a second time and
    checks the model again, because the child inherits the working directory and `resolveSource` runs for it;
    only `go install` is skipped. A local source costs one `git rev-parse`.
  - Any non-empty value of `ENGRAM_UPDATE_REEXEC` counts as the sentinel; the parent sets it to `1`.
  - `--allow-downgrade`, `--answers`, `--reparent-luhmann` and `--dry-run` are not forwarded to the child,
    which never installs, never re-parents and never runs after a dry run.
  - When `Updater.Run` fails, for example with no harness, no report is written, only the wrapped error.
  - The hand-off report is the three header lines; a dry run never writes it, so its `[dry-run] ` prefix is
    never printed there.
  - `engram update --dry-run`, a sentinel run, and a full update with a `go` shim on PATH that copies a built
    binary to `GOBIN` were run in this environment with a temporary HOME, vault and XDG directories. They
    showed the install result printed once before the child's report, the exit status 0, the hand-off, and,
    from a worktree two commits behind the installed binary's revision, the downgrade refusal with exit 1 and
    the install going ahead with `--allow-downgrade`. The real `go install` of a fresh clone was not run (the
    clone's model is a git-lfs pointer here), nor was the remote route.
- **Rationale:** [792d30 Components are migrated once and extended later](#792d30-components-are-migrated-once-and-extended-later), [792d51 command-line-surface routing requirements arrive with their scenarios](#792d51-command-line-surface-routing-requirements-arrive-with-their-scenarios), [792d52 Test IDs: product scenario for end-to-end checks, technical scenario otherwise](#792d52-test-ids-product-scenario-for-end-to-end-checks-technical-scenario-otherwise), [792d89 vocabulary-engine scope, legacy removal and the tests it leaves](#792d89-vocabulary-engine-scope-legacy-removal-and-the-tests-it-leaves), [792d101 vault-migration scope, and the behaviors the code shows](#792d101-vault-migration-scope-and-the-behaviors-the-code-shows) and [792d103 install-and-update scope, ownership of the update run, and the behaviors the code shows](#792d103-install-and-update-scope-ownership-of-the-update-run-and-the-behaviors-the-code-shows).
- **Alternatives:** a separate update-report component. Rejected because the inventory of components is
  fixed ([792d30 Components are migrated once and extended later](#792d30-components-are-migrated-once-and-extended-later)) and the report is the update run's own contract.

## 792d105 harness-deploy-sync scope, and the behaviors the code shows

- **Choice:**
  - [792b29 harness-deploy-sync](./792b_technical-spec.md#792b29-added-harness-deploy-sync) specifies seventeen requirements. It owns harness detection, the deploy set, the roots, the links,
    the cleanup, the first-sync findings, manifest mode and the harness sections of the report; the update
    run, the install and the report header and notices are [792b28 self-installer](./792b_technical-spec.md#792b28-added-self-installer)'s ([792d104 self-installer scope, and the behaviors the code shows](#792d104-self-installer-scope-and-the-behaviors-the-code-shows)). The import detection
    that decides whether guidance is managed is here ([792b29c4 Import detection](./792b_technical-spec.md#792b29c4-import-detection)); the hints that tell a user which `@import`
    line to add stay with ambient-guidance ([792c7r1 ambient-guidance product spec](./792c_implementation-plan.md#792c7r1-ambient-guidance-product-spec)), which also owns the ADR-0022 citations in
    `agent-instructions/guidance/*.md` and the root `AGENTS.md`.
  - Manifest mode has no product scenario of its own. It is specified as an alternative route to [792a11i1a A removed skill disappears](./792a_product-spec.md#792a11i1-uat-scenarios),
    removal that propagates, and carries that scenario in its `Supports`, because no shipped harness uses
    it and nothing in the product requires it ([792d103 install-and-update scope, ownership of the update run, and the behaviors the code shows](#792d103-install-and-update-scope-ownership-of-the-update-run-and-the-behaviors-the-code-shows)).
  - The cleanup, link and adoption tests that the catalog lists for the update package are split by the
    contract they assert: the link step ([792b29c7 Skill links](./792b_technical-spec.md#792b29c7-skill-links)), the adoption step ([792b29c8 Adopting earlier copies](./792b_technical-spec.md#792b29c8-adopting-earlier-copies)) and the scan ([792b29c10 Dangling-link cleanup](./792b_technical-spec.md#792b29c10-dangling-link-cleanup)).
    The three `TestLexicallyResolveSymlinkTarget_*` and four `TestPathWithinRoot_*` tests share the one
    scenario of their helper ([792b29c10a20 A link target is resolved lexically](./792b_technical-spec.md#792b29c10a-scenarios), [792b29c10a21 A path is within the root by whole segments](./792b_technical-spec.md#792b29c10a-scenarios)).
  - Where `update_test` lists a test of the report's harness lines (`TestWriteUpdateReport_EngramRootNotice*`,
    `TestPluralFile`, `TestWriteUpdateReport_GuidanceCanonicalPathHint*`), it takes a [792b29c17 Harness sections of the report](./792b_technical-spec.md#792b29c17-harness-sections-of-the-report) scenario.
- **Divergences noted ([792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated)):**
  - Guidance is managed for every harness when any one of them imports a guidance file or `--with-guidance`
    is given: `GuidanceImported` is true when any harness has an import, and it gates the guidance of all of
    them. The OpenSpec scenario read as per harness, and the first product draft followed it. Confirmed by
    running a sentinel update over a temporary home in which only `~/.claude/CLAUDE.md` imported a file: Pi's
    guidance was deployed and linked too.
  - `commands/` is still a managed subtree of the root, although engram deploys no commands since OpenCode
    support was removed (#721), so any file under `commands/` is deleted on a sync.
  - The report lists a deleted directory as `engram root: deleted <dir>` after the files in it.
  - `unmanaged (left alone)` entries come from the first sync only; the markerless-root run counts as first.
  - The scan for strays covers a harness's skills directory and, for Pi, its guidance directory; Claude
    Code's flat guidance files at its root are found by the unattributable-file scan instead.
  - `internal/update` still carries `applyGuidanceOps`, `applySkillOps` and `clearSkillDirOnce` for manifest
    mode, whose only users are the tests and a mode no harness selects.
  - A temporary-home update through the sentinel confirmed the unmarked-root run (marker written, the
    stray file listed as unattributable, the copied skill adopted, the unrelated skill listed as unmanaged),
    the next run (a stale file, a removed skill's directory and its dangling link deleted, a foreign dangling
    link kept), and the guidance deploy with its compat and Pi links.
- **Rationale:** [792d30 Components are migrated once and extended later](#792d30-components-are-migrated-once-and-extended-later), [792d51 command-line-surface routing requirements arrive with their scenarios](#792d51-command-line-surface-routing-requirements-arrive-with-their-scenarios), [792d52 Test IDs: product scenario for end-to-end checks, technical scenario otherwise](#792d52-test-ids-product-scenario-for-end-to-end-checks-technical-scenario-otherwise), [792d101 vault-migration scope, and the behaviors the code shows](#792d101-vault-migration-scope-and-the-behaviors-the-code-shows), [792d103 install-and-update scope, ownership of the update run, and the behaviors the code shows](#792d103-install-and-update-scope-ownership-of-the-update-run-and-the-behaviors-the-code-shows) and [792d104 self-installer scope, and the behaviors the code shows](#792d104-self-installer-scope-and-the-behaviors-the-code-shows).
- **Alternatives:** make manifest mode a requirement of its own with a product scenario. Rejected because the
  product never exposes it ([792d6 Removed features are not documented; enduring absences become MUST NOT](#792d6-removed-features-are-not-documented-enduring-absences-become-must-not)).

## 792d106 skill-runbook-registration scope, and the behaviors the code shows

- **Choice:**
  - [792a12 skill-runbook-registration](./792a_product-spec.md#792a12-added-skill-runbook-registration) has twenty-two requirements ([792a12a Skill files are never changed](./792a_product-spec.md#792a12a-skill-files-are-never-changed) to [792a12v Skill notes behave like captured runbooks](./792a_product-spec.md#792a12v-skill-notes-behave-like-captured-runbooks)). The source scans keep their user-visible
    scenarios in the product spec ([792a12j Claude Code skills, commands and synced skills are found](./792a_product-spec.md#792a12j-claude-code-skills-commands-and-synced-skills-are-found) to [792a12n Sources are read only for detected harnesses, and names come from the file system](./792a_product-spec.md#792a12n-sources-are-read-only-for-detected-harnesses-and-names-come-from-the-file-system)), as the OpenSpec file did; their mechanics, such as
    the package filters and the trust gate, are [792b30 skill-source-discovery](./792b_technical-spec.md#792b30-added-skill-source-discovery)'s ([792c7k2 skill-source-discovery tech spec](./792c_implementation-plan.md#792c7k2-skill-source-discovery-tech-spec)). The accept actions, the
    offer comparison, the answers and the declines file are [792b31 skill-registration-ledger](./792b_technical-spec.md#792b31-added-skill-registration-ledger)'s ([792c7k3 skill-registration-ledger tech spec](./792c_implementation-plan.md#792c7k3-skill-registration-ledger-tech-spec)), so the
    accept-action tests the catalog deferred to this node are renamed there.
  - The pending-offer warning that a write prints after an accepted registration (``warning: vault holds
    pending offer(s) awaiting curation …``) belongs to curate ([792c7u1 curate product spec](./792c_implementation-plan.md#792c7u1-curate-product-spec)) and is not specified here.
  - The registration step of `engram update` takes [792b28c16 The registration step](./792b_technical-spec.md#792b28c16-the-registration-step) in [792b28 self-installer](./792b_technical-spec.md#792b28-added-self-installer), with its six tests.
  - The tests of the three skill fields that the catalog deferred to this node take [792b6 note-document-format](./792b_technical-spec.md#792b6-added-note-document-format)
    scenarios ([792b6c9 Skill identity fields](./792b_technical-spec.md#792b6c9-skill-identity-fields), [792b6c10 Rewrites by other writers keep the skill identity fields](./792b_technical-spec.md#792b6c10-rewrites-by-other-writers-keep-the-skill-identity-fields)): the render tests and the 15 survival tests of the other writers; the
    amend tests take the amend scenario and the two learn tests take [792b12c7 Skill identity fields are not written by learn](./792b_technical-spec.md#792b12c7-skill-identity-fields-are-not-written-by-learn).
- **Divergences noted ([792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated)):**
  - A skill note is found only when its slug begins `skill-`: a runbook with `skill_hash` and a matching
    `skill_key` under another slug is not read, so registering its key is offered again. The OpenSpec file
    said only that a slug never identifies a note.
  - The Pi sources and `~/.agents/skills` are read only when `~/.pi` exists, and the Claude Code sources only
    when `~/.claude` exists, although the OpenSpec scenarios do not say so.
  - At the per-offer prompt any answer other than `y` or `yes` declines and records the hash, while end of
    input records nothing.
  - `--dry-run` does not run `--adopt`, and `--accept` with `--dry-run` answers nothing.
  - A source path in an offer line is the fully resolved path (for example `/private/tmp/…` on macOS), while
    the note's `skill_source` is home-relative against the resolved home.
  - `register-skills` against a temporary HOME, vault and XDG directories, without a terminal: the dry run
    grouped offers by scope, `--accept` registered a note with the preamble, `skill_hash`, `skill_key`,
    `skill_source`, `pending: true` and a sidecar, `--decline @pi-user` recorded a version-2 declines file, a
    refresh kept the body's new text and set `pending`, a removal was reported but not carried out by
    `--accept '*'` and was carried out by its exact key, two Pi skills of one name and different bytes made
    exit status 1 with a conflict line, `--skills-dir` refused `--accept`, and `--adopt` renamed a runbook note,
    recorded its old basename under `aliases` and kept its authored fields.
- **Rationale:** [792d30 Components are migrated once and extended later](#792d30-components-are-migrated-once-and-extended-later), [792d57 Mixed sources of note-writing are split by owner](#792d57-mixed-sources-of-note-writing-are-split-by-owner), [792d61 note-document-format's edit, CRLF and field-survival contracts arrive with their scenarios](#792d61-note-document-formats-edit-crlf-and-field-survival-contracts-arrive-with-their-scenarios), [792d68 note-authoring scope and the tests it leaves](#792d68-note-authoring-scope-and-the-tests-it-leaves) and [792d104 self-installer scope, and the behaviors the code shows](#792d104-self-installer-scope-and-the-behaviors-the-code-shows).
- **Alternatives:** port the scan mechanics into the product spec. Rejected because package filters and trust
  decisions are not behavior a user states.

## 792d107 skill-source-discovery scope, and the behaviors the code shows

- **Choice:**
  - [792b30 skill-source-discovery](./792b_technical-spec.md#792b30-added-skill-source-discovery) specifies seventeen requirements. The user-visible source scenarios stay in [792a12 skill-runbook-registration](./792a_product-spec.md#792a12-added-skill-runbook-registration) ([792d106 skill-runbook-registration scope, and the behaviors the code shows](#792d106-skill-runbook-registration-scope-and-the-behaviors-the-code-shows)); the
    package filters, the trust gate, the depth limits and the read-status rule are specified here. The
    generic read-status rule ([792b30c2 Root read status](./792b_technical-spec.md#792b30c2-root-read-status)) is shown through the Claude Code user-skills scanner, whose tests
    assert it, and the other scanners' missing-root and read-failure tests take their own scanner's scenario.
  - The comparison and note tests that share the key and scanner test files belong to
    skill-registration-ledger ([792c7k3 skill-registration-ledger tech spec](./792c_implementation-plan.md#792c7k3-skill-registration-ledger-tech-spec)), as does `TestUpdateSkillRegistration_RealRunPrintsScanWarnings`.
    `TestUpdateSkillRegistration_SeesTheSameOffersAsRegisterSkills` runs both commands over one set of
    production-composed dependencies and takes the product scenario [792a12t1d Update and register-skills see the same offers](./792a_product-spec.md#792a12t1-uat-scenarios) ([792d52 Test IDs: product scenario for end-to-end checks, technical scenario otherwise](#792d52-test-ids-product-scenario-for-end-to-end-checks-technical-scenario-otherwise)).
  - The two `register_skills_resolve_test` tests of the default set and of a symlinked home take composition
    scenarios [792b30c16a15 The default set resolves every folder](./792b_technical-spec.md#792b30c16a-scenarios) and [792b30c16a16 A symlinked home round-trips skill_source](./792b_technical-spec.md#792b30c16a-scenarios).
- **Divergences noted ([792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated)):**
  - Discovery reads the Pi and `~/.agents` sources only when `~/.pi` exists, and the Claude Code sources only
    when `~/.claude` exists. A skill in `~/.agents/skills` is invisible on a Claude-only machine.
  - A project-package entry overrides a global one of the same identity, so a package that is both global and
    project-scoped is scanned once, from the project.
  - Past the depth limit a recursive scan marks its root not scanned instead of truncating, so no removal is
    offered for notes under it.
  - A plugin whose `plugin.json` names the plugin root itself as a skill is reported and not scanned, because
    a skill needs a folder name to key on.
- **Rationale:** [792d30 Components are migrated once and extended later](#792d30-components-are-migrated-once-and-extended-later), [792d52 Test IDs: product scenario for end-to-end checks, technical scenario otherwise](#792d52-test-ids-product-scenario-for-end-to-end-checks-technical-scenario-otherwise), [792d106 skill-runbook-registration scope, and the behaviors the code shows](#792d106-skill-runbook-registration-scope-and-the-behaviors-the-code-shows).
- **Alternatives:** none beyond [792d106 skill-runbook-registration scope, and the behaviors the code shows](#792d106-skill-runbook-registration-scope-and-the-behaviors-the-code-shows).

## 792d108 skill-registration-ledger scope, and the behaviors the code shows

- **Choice:**
  - [792b31 skill-registration-ledger](./792b_technical-spec.md#792b31-added-skill-registration-ledger) specifies eight requirements. The accept actions (register, refresh, remove, adopt) are specified
    here, in the component that owns `skillreg_accept.go`, with the offers they answer, since the catalog
    named no other owner for them ([792d106 skill-runbook-registration scope, and the behaviors the code shows](#792d106-skill-runbook-registration-scope-and-the-behaviors-the-code-shows)).
  - The comparison, note and key tests that [792c7k2 skill-source-discovery tech spec](./792c_implementation-plan.md#792c7k2-skill-source-discovery-tech-spec) left (`skillkeys_qualified_test`, the skill tests in
    `skilloffers_test` that resolve real sources) take ledger scenarios, as do the accept-action tests that
    [792c7k1 skill-runbook-registration product spec](./792c_implementation-plan.md#792c7k1-skill-runbook-registration-product-spec) deferred (`TestRefreshSkill_ConvertsCRLFNote`, the two anchored-key tests and
    `TestAdoptSkillNote_ConvertsCRLFNote`).
  - `TestResolveSkillPathBestEffort*` test a function of [792b30 skill-source-discovery](./792b_technical-spec.md#792b30-added-skill-source-discovery)'s file, but its only purpose
    is the eligibility rule that a failed root hides the notes under it, so they take [792b31c3 Removal eligibility](./792b_technical-spec.md#792b31c3-removal-eligibility).
- **Divergences noted ([792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated)):**
  - A missing or unrenderable target of `--adopt` is detected for every entry before the first one writes, but
    an I/O failure of the rename, write or embed after that point cannot be, so a later entry can still fail
    after an earlier one was written.
  - An unanswered offer is not an error: a non-interactive run exits 0 with the summary line, and only a
    conflict gives a failure status.
  - The summary line is printed after the unmatched-token notes and before the conflicts.
- **Rationale:** [792d30 Components are migrated once and extended later](#792d30-components-are-migrated-once-and-extended-later), [792d52 Test IDs: product scenario for end-to-end checks, technical scenario otherwise](#792d52-test-ids-product-scenario-for-end-to-end-checks-technical-scenario-otherwise), [792d106 skill-runbook-registration scope, and the behaviors the code shows](#792d106-skill-runbook-registration-scope-and-the-behaviors-the-code-shows), [792d107 skill-source-discovery scope, and the behaviors the code shows](#792d107-skill-source-discovery-scope-and-the-behaviors-the-code-shows).
- **Alternatives:** none beyond [792d106 skill-runbook-registration scope, and the behaviors the code shows](#792d106-skill-runbook-registration-scope-and-the-behaviors-the-code-shows).

## 792d109 write-memory scope, and what the skill text alone verifies

- **Choice:**
  - [792a13 write-memory](./792a_product-spec.md#792a13-added-write-memory) describes the shipped `SKILL.md`: fourteen requirements ([792a13a The worker runs only on a handoff and never re-judges it](./792a_product-spec.md#792a13a-the-worker-runs-only-on-a-handoff-and-never-re-judges-it) to [792a13n The write is verified and reported](./792a_product-spec.md#792a13n-the-write-is-verified-and-reported)). The `write-memory-worker`
    requirements, the trigger-cue authoring guidance [792d90 Mixed sources of memory-query are split by owner](#792d90-mixed-sources-of-memory-query-are-split-by-owner) deferred to this node, the `learn-runbook-capture`
    [792a13 write-memory](./792a_product-spec.md#792a13-added-write-memory) requirement and ADR-0015 are its sources. The "four kinds of lessons" requirement is learn's
    ([792c7n1 learn product spec](./792c_implementation-plan.md#792c7n1-learn-product-spec)) and the Step 7 audit is please's ([792c7q1 please product spec](./792c_implementation-plan.md#792c7q1-please-product-spec)), so both stay in `write-memory-worker`.
  - The runbook compose block is specified once, from the shipped skill (`--done-when`, `--body`, `--red-flag`,
    `--trigger`). The old requirement's list of handoff fields (slug, situation, done_when, body/steps, source,
    position, optional contributors and red flags) is the handoff of [792a13c The handoff and its missing fields](./792a_product-spec.md#792a13c-the-handoff-and-its-missing-fields) and [792a13e Runbook notes and their red flags](./792a_product-spec.md#792a13e-runbook-notes-and-their-red-flags).
  - Scenarios whose outcome is the worker composing text stay product scenarios without a technical mapping. Nine
    scenarios whose outcome is a flag the binary accepts and writes are mapped to [792b12 note-authoring](./792b_technical-spec.md#792b12-added-note-authoring). Others, such as
    the default placement, the qa tag drop and the retry bound, have no deterministic realization: the worker is a
    markdown procedure ([792d4 Skill procedures are not technical components](#792d4-skill-procedures-are-not-technical-components)), and this skill has no baseline tests.
  - No Go or Python test is renamed. [792b14 note-amendment](./792b_technical-spec.md#792b14-added-note-amendment) gains no mapping, because the skill never hands off an amend.
  - SKILL.md is not edited.
- **Divergences noted ([792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated)):**
  - The old spec's runbook-capture requirement lists `contributors` among runbook handoff fields. The shipped
    skill takes `--contributors` only for qa.
  - The old spec said the worker handles three kinds. The shipped skill handles four, with runbook.
  - The old spec's `--trigger "<cue>"` form (space-separated) differs from the shipped `--trigger="<cue>"`; [792a13h Free-text flags are written with the value touching the equals sign](./792a_product-spec.md#792a13h-free-text-flags-are-written-with-the-value-touching-the-equals-sign)
    follows the shipped form.
  - Root `AGENTS.md`, `README.md` and the GLOSSARY describe the worker as taking `fact|feedback|qa`; they are not
    legacy documents and are left to the final sweeps.
- **Rationale:** [792d3 Skills and guidance are product behavior](#792d3-skills-and-guidance-are-product-behavior), [792d4 Skill procedures are not technical components](#792d4-skill-procedures-are-not-technical-components), [792d32 Skill product specs are derived from the shipped SKILL.md](#792d32-skill-product-specs-are-derived-from-the-shipped-skillmd), [792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated), [792d57 Mixed sources of note-writing are split by owner](#792d57-mixed-sources-of-note-writing-are-split-by-owner) and [792d90 Mixed sources of memory-query are split by owner](#792d90-mixed-sources-of-memory-query-are-split-by-owner).
- **Alternatives:** keep the three-kind wording of the old spec. Rejected because it contradicts the shipped skill.

## 792d110 agent-instruction-bundle scope, and requirements with no product scenario

- **Choice:**
  - [792b32 agent-instruction-bundle](./792b_technical-spec.md#792b32-added-agent-instruction-bundle) specifies four requirements. The metadata limit ([792b32c1 Skill metadata fits the harness limit](./792b_technical-spec.md#792b32c1-skill-metadata-fits-the-harness-limit)) is checked against the shipped files by
    `TestSkillDescriptionsFitMetadataLimit`. The attached-form rule is split in two ([792b32c2 Free-text flag values use the attached form](./792b_technical-spec.md#792b32c2-free-text-flag-values-use-the-attached-form) for every shipped
    template and binary-emitted string, [792b32c3 A skill states the attached form once](./792b_technical-spec.md#792b32c3-a-skill-states-the-attached-form-once) for the skill statement of the rule), because the legacy spec states
    them as two requirements. Invariant S1 and the ADR-0001 boundary (arguments in, standard output out, the vault on
    disk) are one requirement ([792b32c4 Skills reach the vault only through the command line](./792b_technical-spec.md#792b32c4-skills-reach-the-vault-only-through-the-command-line)), because both say the skill layer touches engram only through its command
    line.
  - The container view's per-process statement ("each subcommand is its own process") is [792b2 command-line-surface](./792b_technical-spec.md#792b2-added-command-line-surface)'s
    rationale ([792b2a Orientation](./792b_technical-spec.md#792b2a-orientation)) and is not repeated here.
  - [792b32c1 Skill metadata fits the harness limit](./792b_technical-spec.md#792b32c1-skill-metadata-fits-the-harness-limit) carries `Supports: none`. The limit is the harness's, and no product capability states it. Every other
    technical requirement in [792b Documentation Framework Adoption - Technical Spec](./792b_technical-spec.md#documentation-framework-adoption---technical-spec) supports a product scenario; this is the one exception, because a product scenario
    for "a skill loads" would repeat in every skill capability ([792d5 agent-instruction-bundle qualifies as a component](#792d5-agent-instruction-bundle-qualifies-as-a-component)).
  - ADR-0001's status line, which records the thin `cmd/engram` entry point and the composition root (#700,
    ADR-0020), is deleted with the ADR. Nothing in it is a skill-layer constraint; the root `AGENTS.md` carries the
    live statement.
- **Divergences noted ([792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated)):**
  - The legacy spec lists `engram fact`/`feedback`/`runbook`/`qa` as commands whose free-text flags are covered; the
    commands are `engram learn fact|feedback|runbook|qa`, and [792b32c2 Free-text flag values use the attached form](./792b_technical-spec.md#792b32c2-free-text-flag-values-use-the-attached-form) names `learn`.
  - Only the pending-offer curate cue is a binary-emitted instruction string that shows an `engram` command with
    free-text flags; its tests belong to later nodes.
  - Root `AGENTS.md` still cites `docs/architecture/adr.md` (ADR-0001..0020) as the authority; left to the final
    sweeps.
- **Rationale:** [792d3 Skills and guidance are product behavior](#792d3-skills-and-guidance-are-product-behavior), [792d4 Skill procedures are not technical components](#792d4-skill-procedures-are-not-technical-components), [792d5 agent-instruction-bundle qualifies as a component](#792d5-agent-instruction-bundle-qualifies-as-a-component), [792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated) and [792d109 write-memory scope, and what the skill text alone verifies](#792d109-write-memory-scope-and-what-the-skill-text-alone-verifies).
- **Alternatives:** a product requirement repeated in every skill capability. Rejected in [792d5 agent-instruction-bundle qualifies as a component](#792d5-agent-instruction-bundle-qualifies-as-a-component).

## 792d111 recall scope, the tests it leaves, and known divergences

- **Choice:**
  - [792a14 recall](./792a_product-spec.md#792a14-added-recall) describes the shipped `SKILL.md`: seventeen requirements ([792a14a The plan is stated before anything is retrieved](./792a_product-spec.md#792a14a-the-plan-is-stated-before-anything-is-retrieved) to [792a14q A cited synthesis is also captured as a question and answer](./792a_product-spec.md#792a14q-a-cited-synthesis-is-also-captured-as-a-question-and-answer)). The skill's own step
    numbers (Step 0 to Step 4) are cited in the capability text, so a reader can find the passage. The legacy
    specs supply the dial ([792a14c Recall has two depths](./792a_product-spec.md#792a14c-recall-has-two-depths), [792a14d Glance escalates to deep for a recent standard](./792a_product-spec.md#792a14d-glance-escalates-to-deep-for-a-recent-standard)), the read-before-judge gate ([792a14h Chunk content is read before a cluster is judged](./792a_product-spec.md#792a14h-chunk-content-is-read-before-a-cluster-is-judged)) and, with the context and
    container views, the agent-judged coverage decision ([792a14j Coverage is judged by the agent and written once per cluster](./792a_product-spec.md#792a14j-coverage-is-judged-by-the-agent-and-written-once-per-cluster)) and agent-driven activation ([792a14l Only the notes that were used are activated](./792a_product-spec.md#792a14l-only-the-notes-that-were-used-are-activated)).
  - What the binary returns (the matched set, clusters, explore half, superseding notes, newest chunks) is
    [792a7 memory-query](./792a_product-spec.md#792a7-added-memory-query)'s ([792a7 memory-query](./792a_product-spec.md#792a7-added-memory-query)). Recall's requirements say what the agent does with the payload.
  - The recall-time "lazy-L2 synthesis" the container view names is the coverage decision of [792a14j Coverage is judged by the agent and written once per cluster](./792a_product-spec.md#792a14j-coverage-is-judged-by-the-agent-and-written-once-per-cluster). The
    view's remark that there is no `engram synthesize` is not specified, because it describes an absence of an
    old surface ([792d6 Removed features are not documented; enduring absences become MUST NOT](#792d6-removed-features-are-not-documented-enduring-absences-become-must-not)).
  - The `q-space` channel for question notes is proposed and not built, so it is left out ([792d6 Removed features are not documented; enduring absences become MUST NOT](#792d6-removed-features-are-not-documented-enduring-absences-become-must-not)).
  - Glance's measured cost (2.23 times faster, 46% cheaper per fire) is rationale and not a requirement, because
    it is a measurement of a run and not behavior a user can observe.
  - No test is renamed. The recall baselines are markdown evidence ([792d10 Test renaming](#792d10-test-renaming)); the mapped technical scenarios were
    renamed by the nodes that own those components. The superpowers plan's value was its #733 root-cause evidence,
    which is the rationale of [792a14h Chunk content is read before a cluster is judged](./792a_product-spec.md#792a14h-chunk-content-is-read-before-a-cluster-is-judged); the rest of the plan is a task list and is deleted ([792d12 Research, eval-process notes and plans are non-normative](#792d12-research-eval-process-notes-and-plans-are-non-normative)).
- **Divergences noted ([792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated)):**
  - `agent-instructions/skills/recall/tests/baseline-multi-query.md` describes a model the skill has since left:
    five to fifteen phrases, `activated: true` items batched into one `engram activate`, and candidate notes read
    through `engram show`. The shipped skill says ten phrases (three in glance), agent-driven activation and
    candidate content inline. The baseline's file is not changed (a baseline edit is a skill-test edit).
  - The dial spec cites ADR-0004 as its "why". ADR-0004 is the dead tier decision that [792c8d Deprecated-directory removal and final stale-reference sweep](./792c_implementation-plan.md#792c8d-deprecated-directory-removal-and-final-stale-reference-sweep) deletes, so the
    dial's reasoning is the measured rows in `dev/eval/LEDGER.md` (cited in [792a14c Recall has two depths](./792a_product-spec.md#792a14c-recall-has-two-depths) and [792a14d Glance escalates to deep for a recent standard](./792a_product-spec.md#792a14d-glance-escalates-to-deep-for-a-recent-standard)).
- **Rationale:** [792d3 Skills and guidance are product behavior](#792d3-skills-and-guidance-are-product-behavior), [792d4 Skill procedures are not technical components](#792d4-skill-procedures-are-not-technical-components), [792d12 Research, eval-process notes and plans are non-normative](#792d12-research-eval-process-notes-and-plans-are-non-normative), [792d32 Skill product specs are derived from the shipped SKILL.md](#792d32-skill-product-specs-are-derived-from-the-shipped-skillmd), [792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated) and [792d90 Mixed sources of memory-query are split by owner](#792d90-mixed-sources-of-memory-query-are-split-by-owner).
- **Alternatives:** none beyond those stated.

## 792d112 learn scope, the tests it leaves, and known divergences

- **Choice:**
  - [792a15 learn](./792a_product-spec.md#792a15-added-learn) describes the shipped `SKILL.md`: twelve requirements ([792a15a Learn writes only explicit lessons](./792a_product-spec.md#792a15a-learn-writes-only-explicit-lessons) to [792a15l The re-parent batch mode](./792a_product-spec.md#792a15l-the-re-parent-batch-mode)), including the vocabulary
    check (Step 1.5) and the re-parent batch mode, which are in the skill and which the catalog's sources did not
    name. The binary half of both is described elsewhere ([792a6 controlled-vocabulary](./792a_product-spec.md#792a6-added-controlled-vocabulary), [792a10c Candidates to judge](./792a_product-spec.md#792a10c-candidates-to-judge)); the agent's half is here.
  - The `LESSONS:` handling (judging a returned line on the fast path, the closing backstop, the kind-4 completion
    anchor and the audit-derived exemplars) is [792a16 lessons-contract](./792a_product-spec.md#792a16-added-lessons-contract)'s ([792c7o1 lessons-contract product spec](./792c_implementation-plan.md#792c7o1-lessons-contract-product-spec)). [792a15c A correction moment gets a single-note capture](./792a_product-spec.md#792a15c-a-correction-moment-gets-a-single-note-capture) states only the correction
    fast path and points at [792a16 lessons-contract](./792a_product-spec.md#792a16-added-lessons-contract) for the subagent return. [792a15g A confirmed approach needs a bet or specific praise, and takes the kind its shape calls for](./792a_product-spec.md#792a15g-a-confirmed-approach-needs-a-bet-or-specific-praise-and-takes-the-kind-its-shape-calls-for) carries the kind-4 bar and the exemplars as
    the skill states them in Step 2, and [792c7o1 lessons-contract product spec](./792c_implementation-plan.md#792c7o1-lessons-contract-product-spec) states the completion-moment anchor.
  - The third requirement of `learn-branching-disposition` (the [792a13 write-memory](./792a_product-spec.md#792a13-added-write-memory) handoff carries placement fields) is
    [792a13g Placement](./792a_product-spec.md#792a13g-placement). It was not on this node's list but sits in a file this node deletes; it is not ported twice.
  - The B2 capture-phrasing convention ("capture abstraction = generic-actionable") is the rationale of [792a15i A lesson states the general principle](./792a_product-spec.md#792a15i-a-lesson-states-the-general-principle).
    The other bullet of "Decisions deliberately NOT made into ADRs" (B10, a reversed operational decision) is
    deleted with the section; ROADMAP Provenance records dead ends ([792d6 Removed features are not documented; enduring absences become MUST NOT](#792d6-removed-features-are-not-documented-enduring-absences-become-must-not)).
  - [792b20 vocabulary-engine](./792b_technical-spec.md#792b20-added-vocabulary-engine) is added to the components this node extends with mappings, because the refit procedure's
    scenarios ([792a15d The vocabulary check and its autonomous refit](./792a_product-spec.md#792a15d-the-vocabulary-check-and-its-autonomous-refit)) have their binary half in [792b20c7 Statistics](./792b_technical-spec.md#792b20c7-statistics) and [792b20c15 Refit routing](./792b_technical-spec.md#792b20c15-refit-routing) to [792b20c18 Applying a derivation](./792b_technical-spec.md#792b20c18-applying-a-derivation).
  - No test is renamed. The learn baselines are markdown evidence ([792d10 Test renaming](#792d10-test-renaming)).
- **Divergences noted ([792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated)):**
  - `agent-instructions/skills/learn/tests/baseline-clean-write.md` says the Luhmann-continuation and `--target`
    write path is removed from the skill. The restored disposition step (archive 2026-08-09, in [792a15h Each note is placed by an ordered test](./792a_product-spec.md#792a15h-each-note-is-placed-by-an-ordered-test)) reverses
    that; the baseline was not updated. It is a skill-test edit.
  - The mid-task fast path is stated in the skill as for a "correction moment" or a subagent return; the ambient
    learn guidance carries the firing cue ([792c7r1 ambient-guidance product spec](./792c_implementation-plan.md#792c7r1-ambient-guidance-product-spec)).
  - Step 2 never sets `--tier`; `learn --tier` remains a flag in the binary ([792d55 learn --tier is documented as a frontmatter field](#792d55-learn---tier-is-documented-as-a-frontmatter-field)).
- **Rationale:** [792d3 Skills and guidance are product behavior](#792d3-skills-and-guidance-are-product-behavior), [792d12 Research, eval-process notes and plans are non-normative](#792d12-research-eval-process-notes-and-plans-are-non-normative), [792d32 Skill product specs are derived from the shipped SKILL.md](#792d32-skill-product-specs-are-derived-from-the-shipped-skillmd), [792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated), [792d57 Mixed sources of note-writing are split by owner](#792d57-mixed-sources-of-note-writing-are-split-by-owner) and [792d111 recall scope, the tests it leaves, and known divergences](#792d111-recall-scope-the-tests-it-leaves-and-known-divergences).
- **Alternatives:** leave the vocabulary check and batch mode out of the [792a15 learn](./792a_product-spec.md#792a15-added-learn) capability. Rejected because the
  shipped skill carries them and no other node describes the agent's side.

## 792d113 lessons-contract scope, and where the sources differ from the skills

- **Choice:**
  - [792a16 lessons-contract](./792a_product-spec.md#792a16-added-lessons-contract) states the contract once, from the LESSONS sections of the shipped route, please and learn skills
    and the subagent-return cue of the learn guidance. Its seven requirements are in the order a line travels.
    The cue's firing (what makes the orchestrator judge at the return) is [792a19 ambient-guidance](./792a_product-spec.md#792a19-added-ambient-guidance)'s ([792c7r1 ambient-guidance product spec](./792c_implementation-plan.md#792c7r1-ambient-guidance-product-spec)); [792a16e Each lesson is judged when it returns](./792a_product-spec.md#792a16e-each-lesson-is-judged-when-it-returns)
    states what the orchestrator does.
  - The completion-moment anchor and exemplars of kind 4, which the `learn` OpenSpec spec held, are [792a16g A confirmed approach is looked for at the moment of confirmation](./792a_product-spec.md#792a16g-a-confirmed-approach-is-looked-for-at-the-moment-of-confirmation).
    The kind-4 bar itself is in [792a15g A confirmed approach needs a bet or specific praise, and takes the kind its shape calls for](./792a_product-spec.md#792a15g-a-confirmed-approach-needs-a-bet-or-specific-praise-and-takes-the-kind-its-shape-calls-for).
  - The `learn`, `please` and `route` OpenSpec specs held nothing but the contract's slices, so they are
    deleted whole, as the catalog's mapping in [792d20 lessons-contract is its own capability](#792d20-lessons-contract-is-its-own-capability) expected.
  - No test is renamed: the contract has no Go or Python test. The ledger rows and the confined harness are
    evidence for the eval node ([792c7v memory-effectiveness-evaluation](./792c_implementation-plan.md#792c7v-memory-effectiveness-evaluation)).
- **Divergences noted ([792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated)):**
  - The `route` spec says an unanswered re-ask leaves "an empty slot for that subagent" and the [792a16 lessons-contract](./792a_product-spec.md#792a16-added-lessons-contract)
    spec gives the same; the shipped route skill says to record `LESSONS: none` and move on. [792a16c A missing line is asked for once](./792a_product-spec.md#792a16c-a-missing-line-is-asked-for-once) follows
    the skill.
  - The `please` spec's scenario "Re-executed work after gate rejection" says lines of a rejected attempt are
    not carried forward (collect once per final dispatch). The shipped please skill says to record each unit's
    line as its result comes back, and says nothing of a rejected attempt. The scenario is not ported; [792a16d Every line is recorded as it returns](./792a_product-spec.md#792a16d-every-line-is-recorded-as-it-returns)
    states only the unconditional recording.
  - The [792a16 lessons-contract](./792a_product-spec.md#792a16-added-lessons-contract) spec says a `LESSONS` line may carry "a single lesson" or "up to three lessons,
    comma-separated". The skill says "1 to 3 comma-separated one-line lessons". Both are [792a16a A completion report ends with a LESSONS line](./792a_product-spec.md#792a16a-a-completion-report-ends-with-a-lessons-line).
  - The `please` spec says please hands the list to learn as an entry such as `lessons_from_session: [...]`; the
    skill says only that the list is handed over. [792a16f The closing learn gets the whole list](./792a_product-spec.md#792a16f-the-closing-learn-gets-the-whole-list) does not name a field.
- **Rationale:** [792d20 lessons-contract is its own capability](#792d20-lessons-contract-is-its-own-capability), [792d32 Skill product specs are derived from the shipped SKILL.md](#792d32-skill-product-specs-are-derived-from-the-shipped-skillmd), [792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated) and [792d111 recall scope, the tests it leaves, and known divergences](#792d111-recall-scope-the-tests-it-leaves-and-known-divergences).
- **Alternatives:** put the requirements in each skill's capability. Rejected in [792d20 lessons-contract is its own capability](#792d20-lessons-contract-is-its-own-capability).

## 792d114 route scope, the SKILL.md repoint, and where the sources differ from the skill

- **Choice:**
  - [792a17 route](./792a_product-spec.md#792a17-added-route) describes the shipped `SKILL.md` and `price-table.md`: fifteen requirements ([792a17a The orchestrator routes and delegates](./792a_product-spec.md#792a17a-the-orchestrator-routes-and-delegates) to [792a17o The loop improves the rubric](./792a_product-spec.md#792a17o-the-loop-improves-the-rubric)). The
    two OpenSpec specs supply the tier rubric and the evidence, aggregate, count and cost requirements; the skill
    supplies the delegation line ([792a17a The orchestrator routes and delegates](./792a_product-spec.md#792a17a-the-orchestrator-routes-and-delegates)), the handoff ([792a17g Every dispatch hands over an exact spec](./792a_product-spec.md#792a17g-every-dispatch-hands-over-an-exact-spec)), recall-first and decomposition ([792a17h Every subagent recalls first, and large units are decomposed](./792a_product-spec.md#792a17h-every-subagent-recalls-first-and-large-units-are-decomposed)) and the
    loop ([792a17o The loop improves the rubric](./792a_product-spec.md#792a17o-the-loop-improves-the-rubric)). The completion-report contract is [792a16b Every dispatch is told to report lessons](./792a_product-spec.md#792a16b-every-dispatch-is-told-to-report-lessons), and [792a17g Every dispatch hands over an exact spec](./792a_product-spec.md#792a17g-every-dispatch-hands-over-an-exact-spec) cites it.
  - Route's "is not an L1 element" paragraph is not a behavior. The orientation sentence says route is skill-level
    guidance for the orchestrator and adds no boundary.
  - ADR-0017's "Deferred" list (#669 resolved by tags-based evidence, and #670 periodic rubric refit, #671 parallel
    builders and #672 cost and duration telemetry) is not specified. #669 is shipped as [792a17k Each dispatch is also written to the vault as a tagged evidence note](./792a_product-spec.md#792a17k-each-dispatch-is-also-written-to-the-vault-as-a-tagged-evidence-note) to [792a17m Counting audits the tallies and never routes](./792a_product-spec.md#792a17m-counting-audits-the-tallies-and-never-routes), #672's
    telemetry is [792a17j Cost and duration are real or stated as unavailable](./792a_product-spec.md#792a17j-cost-and-duration-are-real-or-stated-as-unavailable), and the others are ROADMAP items.
  - The route-skill-to-runbook archive is a reversed direction ([792d57 Mixed sources of note-writing are split by owner](#792d57-mixed-sources-of-note-writing-are-split-by-owner)) and is not read as a requirement.
  - **SKILL.md repoint ([792d40 SKILL.md reference edits follow the root rule](#792d40-skillmd-reference-edits-follow-the-root-rule)).** Four citations in route's `SKILL.md` pointed at deleted documents:
    `docs/architecture/c1-system-context.md` S3, and ADR-0019 three times. They now name [792b30 skill-source-discovery](./792b_technical-spec.md#792b30-added-skill-source-discovery) (where the harness
    sources and the Pi location landed, [792c7k2 skill-source-discovery tech spec](./792c_implementation-plan.md#792c7k2-skill-source-discovery-tech-spec)) and [792a6c Categorical tags share one list](./792a_product-spec.md#792a6c-categorical-tags-share-one-list) (where ADR-0019 landed). Judgement: a repoint of a
    reference to a moved document changes no instruction, procedure, command, flag or wording of any rule, so it is
    not a behavioral edit and the writing-skills red, green and pressure cycle that the root `AGENTS.md` requires for
    behavioral edits was not run. The skill's `skill_hash` changes, so registration will offer a refresh of its
    runbook note. The cited sections are in a change directory that is promoted later ([792d34 References are repointed to the change sections](#792d34-references-are-repointed-to-the-change-sections)), so the repoint
    follows the promotion grep.
  - Tests that verify the mapped scenarios were renamed by earlier nodes. The extension of [792b12 note-authoring](./792b_technical-spec.md#792b12-added-note-authoring) and
    [792b14 note-amendment](./792b_technical-spec.md#792b14-added-note-amendment) is for the tag and amend scenarios that the legacy spec named; [792b25 vault-counter](./792b_technical-spec.md#792b25-added-vault-counter) is on the
    catalog's list.
- **Divergences noted ([792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated)):**
  - The 70% and 85% escalation thresholds and the five-dispatch evidence floor are in the OpenSpec text only. The
    shipped skill carries the resulting table row and the rule that evidence overwrites priors. [792a17f Cold-start priors carry only evidence-backed escalations](./792a_product-spec.md#792a17f-cold-start-priors-carry-only-evidence-backed-escalations) states the
    thresholds because the legacy requirement did, and this is the only place they are written.
  - The OpenSpec spec says the free-local-model red flag is in "route's runbook `red_flags` field"; it is the red
    flags table of the skill, and [792a17e1c A paid dispatch with a free option available is a red flag](./792a_product-spec.md#792a17e1-uat-scenarios) follows the skill.
  - Both specs gave the re-ask outcome as an empty slot; the skill records `LESSONS: none` ([792d113 lessons-contract scope, and where the sources differ from the skills](#792d113-lessons-contract-scope-and-where-the-sources-differ-from-the-skills)).
  - The skill's aggregate procedure runs a query and a basename listing and calls the listing "documented"; the
    skill gives no command for the listing, only the match rule. [792a17l Each work-kind keeps one aggregate note](./792a_product-spec.md#792a17l-each-work-kind-keeps-one-aggregate-note) states the rule and the listing without a
    command.
- **Rationale:** [792d3 Skills and guidance are product behavior](#792d3-skills-and-guidance-are-product-behavior), [792d32 Skill product specs are derived from the shipped SKILL.md](#792d32-skill-product-specs-are-derived-from-the-shipped-skillmd), [792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated), [792d34 References are repointed to the change sections](#792d34-references-are-repointed-to-the-change-sections), [792d40 SKILL.md reference edits follow the root rule](#792d40-skillmd-reference-edits-follow-the-root-rule) and [792d113 lessons-contract scope, and where the sources differ from the skills](#792d113-lessons-contract-scope-and-where-the-sources-differ-from-the-skills).
- **Alternatives:** run the writing-skills cycle for the repoint. Rejected because no behavior changes.

## 792d115 please scope, and where the sources differ from the skill

- **Choice:**
  - [792a18 please](./792a_product-spec.md#792a18-added-please) describes the shipped `SKILL.md`: nineteen requirements ([792a18a The skill fires on a handed-over multi-step ask](./792a_product-spec.md#792a18a-the-skill-fires-on-a-handed-over-multi-step-ask) to [792a18s Object-level work is delegated](./792a_product-spec.md#792a18s-object-level-work-is-delegated)). The skill is the only
    source of the workflow, the gates and the anti-sycophancy rules, as [792d32 Skill product specs are derived from the shipped SKILL.md](#792d32-skill-product-specs-are-derived-from-the-shipped-skillmd) expects. The enumeration-grep and
    lessons-audit requirements are ported from the OpenSpec specs and re-checked against the skill.
  - The gates are one requirement for their firing and staffing ([792a18l Four gates, each with a fresh reviewer per angle](./792a_product-spec.md#792a18l-four-gates-each-with-a-fresh-reviewer-per-angle)), one for the reviewer charges ([792a18m Each reviewer tries to refute the artifact](./792a_product-spec.md#792a18m-each-reviewer-tries-to-refute-the-artifact)),
    and one for the argument protocol ([792a18n Findings are argued to resolution](./792a_product-spec.md#792a18n-findings-are-argued-to-resolution)), because the skill states them as a table, a list of charges and a
    protocol.
  - The LESSONS slice of please (collection, hand-over, gates not filtering) is [792a16d Every line is recorded as it returns](./792a_product-spec.md#792a16d-every-line-is-recorded-as-it-returns) and [792a16f The closing learn gets the whole list](./792a_product-spec.md#792a16f-the-closing-learn-gets-the-whole-list). [792a18r The closing learn gets the lessons and the audit](./792a_product-spec.md#792a18r-the-closing-learn-gets-the-lessons-and-the-audit) only
    states that the closing learn gets both inputs.
  - The "terminal report" contents drawn in the context view (commits made, paths written, gate audit, follow-ups
    offered) are not in the skill, which says only that the audit list appears in the closing report; [792a18p The closing learn is preceded by a lessons audit](./792a_product-spec.md#792a18p-the-closing-learn-is-preceded-by-a-lessons-audit)
    states what the skill says.
  - The context view's remark that the former gates sub-runbook `1042` was merged is history of a reversed
    direction ([792d57 Mixed sources of note-writing are split by owner](#792d57-mixed-sources-of-note-writing-are-split-by-owner)) and is left out.
  - No test is renamed. There is no `tests/` directory for please.
- **Divergences noted ([792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated)):**
  - The `write-memory-worker` spec's Step 7 requirement says the audit is "carried by the lessons-audit
    sub-runbook, reached by `[[basename]]` wikilink from the top runbook body", and the `please-doc-enumeration-gate`
    spec says the gate-A reviewer charge is "carried by the please runbook set's adversarial-review-gates
    sub-runbook". Both are stale (#768, #758's conversion was merged back into one note). The shipped skill carries
    both in its own body, and [792a18 please](./792a_product-spec.md#792a18-added-please) follows it.
  - The same spec's scenario "Vault note citations use wikilink syntax" is a rule of the retired runbook form
    (a wikilink inside a structured field). The shipped skill says nothing about citation syntax in the audit, so it
    is not specified.
  - The `please-doc-enumeration-gate` spec's third scenario, "Carrier is the runbook, not a skill", is the opposite
    of the shipped carrier and is not ported.
  - The skill's description, which is also the trigger discrimination, is stated as behavior in [792a18a The skill fires on a handed-over multi-step ask](./792a_product-spec.md#792a18a-the-skill-fires-on-a-handed-over-multi-step-ask). The
    registered runbook note's `triggers` are in [792a12 skill-runbook-registration](./792a_product-spec.md#792a12-added-skill-runbook-registration) and [792a13m Trigger cues are specific, author-chosen strings](./792a_product-spec.md#792a13m-trigger-cues-are-specific-author-chosen-strings).
  - ROADMAP row #768 still records the stale-spec work as open. Its two specs are migrated and deleted here, and the
    row now says so.
- **Rationale:** [792d3 Skills and guidance are product behavior](#792d3-skills-and-guidance-are-product-behavior), [792d12 Research, eval-process notes and plans are non-normative](#792d12-research-eval-process-notes-and-plans-are-non-normative), [792d32 Skill product specs are derived from the shipped SKILL.md](#792d32-skill-product-specs-are-derived-from-the-shipped-skillmd), [792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated), [792d57 Mixed sources of note-writing are split by owner](#792d57-mixed-sources-of-note-writing-are-split-by-owner) and [792d113 lessons-contract scope, and where the sources differ from the skills](#792d113-lessons-contract-scope-and-where-the-sources-differ-from-the-skills).
- **Alternatives:** keep the stale sub-runbook text in the requirements. Rejected because it contradicts the shipped
  skill (the caution on this node).

## 792d116 ambient-guidance scope, the guidance edits, and where the sources differ from the documents

- **Choice:**
  - [792a19 ambient-guidance](./792a_product-spec.md#792a19-added-ambient-guidance) describes the four shipped documents in `agent-instructions/guidance/` and the activation hints of
    `engram update`: twenty-three requirements ([792a19a Recall fires at four decision moments](./792a_product-spec.md#792a19a-recall-fires-at-four-decision-moments) to [792a19w Engram is engaged around work that is more than a trivial lookup](./792a_product-spec.md#792a19w-engram-is-engaged-around-work-that-is-more-than-a-trivial-lookup)). The documents are the source of the
    cues; the five OpenSpec specs are the source of the scenarios and rationale, re-checked against the documents.
    The recall cues (a, b), the learn cues (c to f), the delegate cues (g to i) and the shim (j to t) each have a
    requirement per distinct obligation, because the documents state them as separate rules.
  - The served `--text` round trip stays in `runbook-lexical-triggers` for [792c7s1 vault-serving product spec](./792c_implementation-plan.md#792c7s1-vault-serving-product-spec). Only the shim's verbatim
    `--text` requirement was removed from that spec, and it is [792a19j The first action on every request is an engram query](./792a_product-spec.md#792a19j-the-first-action-on-every-request-is-an-engram-query). The spec's Purpose now says that a
    node remains.
  - ADR-0026, whose capture, retrieval and lexical decisions were ported earlier, held no follow-frame text. Its
    remainder (four kinds, no validation gate, #730 out of scope) is rationale for [792a19l Each returned kind is treated for what it is](./792a_product-spec.md#792a19l-each-returned-kind-is-treated-for-what-it-is), and the ADR becomes a
    pointer like the other ported ADRs.
  - The activation hints of `engram update` get a [792b29 harness-deploy-sync](./792b_technical-spec.md#792b29-added-harness-deploy-sync) requirement ([792b29c18 Guidance activation hints](./792b_technical-spec.md#792b29c18-guidance-activation-hints)), with the product
    behavior in [792a19v The update report says how to activate guidance](./792a_product-spec.md#792a19v-the-update-report-says-how-to-activate-guidance), per [792d103 install-and-update scope, ownership of the update run, and the behaviors the code shows](#792d103-install-and-update-scope-ownership-of-the-update-run-and-the-behaviors-the-code-shows) and [792d105 harness-deploy-sync scope, and the behaviors the code shows](#792d105-harness-deploy-sync-scope-and-the-behaviors-the-code-shows). `TestWriteUpdateReport_GuidanceActivationHint` is one table test
    over eight cases, so it takes [792b29c18a1 A Claude Code file not imported is deployed with its line to add](./792b_technical-spec.md#792b29c18a-scenarios) and lists [792b29c18a2 An imported Claude Code file is refreshed](./792b_technical-spec.md#792b29c18a-scenarios) to [792b29c18a8 Each file is judged on its own imports](./792b_technical-spec.md#792b29c18a-scenarios) as split deferred.
  - Two Python static guards in `dev/eval/test_no_space_form_free_text_flags.py` verify [792b32c2a1 A skill's example command uses the attached form](./792b_technical-spec.md#792b32c2a-scenarios) and [792b32c3a1 A skill states the rule and its reason](./792b_technical-spec.md#792b32c3a-scenarios)
    (the attached form in every shipped skill and guidance document, and the sentence in the four skills). The
    nodes that added those requirements did not list them, so they are renamed here, where the guidance documents
    they also scan are described. The scanner's own self-tests keep their names.
  - **Guidance edits ([792d40 SKILL.md reference edits follow the root rule](#792d40-skillmd-reference-edits-follow-the-root-rule)).** The `ADR-0022 D9` citation in the header comment of `recall.md`, `learn.md` and
    `delegate.md`, and in the root `AGENTS.md`, now names [792a11m Claude import paths keep working](./792a_product-spec.md#792a11m-claude-import-paths-keep-working) (the compat links) in
    `docs/changes/792_documentation-framework-adoption/792a_product-spec.md`. A repoint of a citation in an HTML
    comment changes no instruction the agent reads, so it is not a behavioral edit and the writing-skills red,
    green and pressure cycle was not run. The documents' text is deployed by copy, so they change for every user on
    their next `engram update`.
  - The context view's engagement flowchart is the lifecycle of [792a19w Engram is engaged around work that is more than a trivial lookup](./792a_product-spec.md#792a19w-engram-is-engaged-around-work-that-is-more-than-a-trivial-lookup). The flowchart's step that runs
    `engram ingest --auto` is learn's ([792a15 learn](./792a_product-spec.md#792a15-added-learn)), and the "print Step-0 plan" step is recall's ([792a14a The plan is stated before anything is retrieved](./792a_product-spec.md#792a14a-the-plan-is-stated-before-anything-is-retrieved)), so
    neither is restated.
  - #773 (duplicate cues) and #790 (the shim refused as a suspected prompt injection) are open, and [792a19 ambient-guidance](./792a_product-spec.md#792a19-added-ambient-guidance) states so
    in its context paragraph. The blanket recall hook is a dead end and is stated as one.
- **Divergences noted ([792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated)):**
  - The recall-moments spec says the glance runs "with phrases derived from the approach". The shipped document says
    only `/recall glance`; [792a19a1a A proposed approach is recalled before it is endorsed](./792a_product-spec.md#792a19a1-uat-scenarios) keeps the phrases because recall's own skill requires them ([792a14 recall](./792a_product-spec.md#792a14-added-recall)).
  - The delegate spec says route picks the tier "starting at cheapest unless recalled evidence raises it" and that
    the report carries "cost if available". The shipped document says only that route sets tier, handoff and
    evidence, and names the evidence table and please's gate verdicts. The starting tier and the cost are route's
    ([792a17 route](./792a_product-spec.md#792a17-added-route)), and [792a19g Every unit of work is planned and routed first](./792a_product-spec.md#792a19g-every-unit-of-work-is-planned-and-routed-first) and [792a19i Returned work is reviewed fresh and reported with its evidence](./792a_product-spec.md#792a19i-returned-work-is-reviewed-fresh-and-reported-with-its-evidence) follow the document.
  - The follow-frame spec has a requirement that a runbook named by a skill's own instructions gets the same frame,
    for a skill that says to "fetch and follow `[[<basename>]]` with this handoff" at a write site. The shipped
    shim has no such text, and recall and learn now invoke [792a13 write-memory](./792a_product-spec.md#792a13-added-write-memory) natively as a skill, not by basename
    (AGENTS.md). It is not ported, and the scenario 792a19q1b (retired; no longer defined) is retired with it. The shim states the frame for the
    kind, so a runbook from a parent vault or of a registered skill is covered by [792a19q Linked and truncated runbooks get the same frame](./792a_product-spec.md#792a19q-linked-and-truncated-runbooks-get-the-same-frame) as a consequence.
  - The `production-guidance-activation` spec treats the full replacement of the other three documents as pending
    on #760. The ROADMAP records #760 as shipped for all six skills, yet the activation policy is unchanged, so
    [792a19u The shim joins the other guidance additively](./792a_product-spec.md#792a19u-the-shim-joins-the-other-guidance-additively) keeps the spec's wording. This is not described as resolved.
- **Rationale:** [792d3 Skills and guidance are product behavior](#792d3-skills-and-guidance-are-product-behavior), [792d19 One ambient-guidance capability](#792d19-one-ambient-guidance-capability), [792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated), [792d40 SKILL.md reference edits follow the root rule](#792d40-skillmd-reference-edits-follow-the-root-rule), [792d52 Test IDs: product scenario for end-to-end checks, technical scenario otherwise](#792d52-test-ids-product-scenario-for-end-to-end-checks-technical-scenario-otherwise), [792d90 Mixed sources of memory-query are split by owner](#792d90-mixed-sources-of-memory-query-are-split-by-owner), [792d103 install-and-update scope, ownership of the update run, and the behaviors the code shows](#792d103-install-and-update-scope-ownership-of-the-update-run-and-the-behaviors-the-code-shows), [792d105 harness-deploy-sync scope, and the behaviors the code shows](#792d105-harness-deploy-sync-scope-and-the-behaviors-the-code-shows) and [792d113 lessons-contract scope, and where the sources differ from the skills](#792d113-lessons-contract-scope-and-where-the-sources-differ-from-the-skills).
- **Alternatives:** one capability per guidance document. Rejected by [792d19 One ambient-guidance capability](#792d19-one-ambient-guidance-capability). Run the writing-skills cycle for the
  citation repoint. Rejected because no instruction changes.

## 792d117 vault-serving scope, what stays for the technical node, and where the sources differ from the server

- **Choice:**
  - [792a20 vault-serving](./792a_product-spec.md#792a20-added-vault-serving) describes `engram serve` as the code, a built binary run against a scratch vault, and `engram serve --help`
    show it: fifteen requirements ([792a20a The served command set is fixed](./792a_product-spec.md#792a20a-the-served-command-set-is-fixed) to [792a20o The receiving vault tags a served offer](./792a_product-spec.md#792a20o-the-receiving-vault-tags-a-served-offer)). Each requirement of the `vault-serve-api` spec has its product
    side here. The spec's served-`--text` requirement came from `runbook-lexical-triggers` ([792a20m A served query takes the message and can return dedupe keys](./792a_product-spec.md#792a20m-a-served-query-takes-the-message-and-can-return-dedupe-keys)), and the
    `learn` identity and offer cases from `vault-note-identity` and ADR-0029 D7.
  - `vault-serve-api` is not trimmed by this node. Each of its requirements states a contract of the server as well as a
    behavior of the product, and [792c7s4 vault-http-server tech spec](./792c_implementation-plan.md#792c7s4-vault-http-server-tech-spec) reads the same requirements to write [792b35 vault-http-server](./792b_technical-spec.md#792b35-added-vault-http-server), so a half-trimmed file would
    lose the server's side. It is deleted by [792c7s4 vault-http-server tech spec](./792c_implementation-plan.md#792c7s4-vault-http-server-tech-spec), with the spec's other content.
  - The last requirement of `vault-note-identity` ("Exchange fields SHALL NOT be settable over the served API") says
    what [792a20j A caller cannot set link, identity or registration fields](./792a_product-spec.md#792a20j-a-caller-cannot-set-link-identity-or-registration-fields) says. It stays with that spec for parent-sync ([792c7t1 parent-sync product spec](./792c_implementation-plan.md#792c7t1-parent-sync-product-spec)), which owns its other exchange-field
    requirements, and is deleted there as a duplicate of [792a20j A caller cannot set link, identity or registration fields](./792a_product-spec.md#792a20j-a-caller-cannot-set-link-identity-or-registration-fields).
  - Per [792d67 note-provenance covers detection and the first-write stamp](#792d67-note-provenance-covers-detection-and-the-first-write-stamp) the served first-write stamp is [792a20f A served learn stamps the caller's repo, the served vault's name and the first-write identity](./792a_product-spec.md#792a20f-a-served-learn-stamps-the-callers-repo-the-served-vaults-name-and-the-first-write-identity), and its technical side is [792b11 note-provenance](./792b_technical-spec.md#792b11-added-note-provenance)'s [792b11c3 First-write stamp](./792b_technical-spec.md#792b11c3-first-write-stamp).
    [792b11 note-provenance](./792b_technical-spec.md#792b11-added-note-provenance) is therefore added to the components this node extends with mapping rows. [792b24 query-engine](./792b_technical-spec.md#792b24-added-query-engine) is on the
    catalog's list and gains nothing: the served query is the route's contract and not a property of the query
    pipeline ([792b35 vault-http-server](./792b_technical-spec.md#792b35-added-vault-http-server)).
  - The receipt's `for` field, the 409 body's `reason` and `vault_id`, the per-ref result of a served activate and its
    four statuses, the 400 for an empty `note` and the vocabulary tagging after a served write are in the code and not
    in the spec; [792a20 vault-serving](./792a_product-spec.md#792a20-added-vault-serving) states them.
  - The child's 16 MiB cap on a parent response is in the `vault-serve-api` spec and has no other owner, so [792a20n Request and response bodies are capped](./792a_product-spec.md#792a20n-request-and-response-bodies-are-capped)
    states it, with the exchange itself left to parent-sync ([792a21 parent-sync](./792a_product-spec.md#792a21-added-parent-sync)).
  - Two command-level tests are renamed to product scenarios ([792d52 Test IDs: product scenario for end-to-end checks, technical scenario otherwise](#792d52-test-ids-product-scenario-for-end-to-end-checks-technical-scenario-otherwise)); the handler-level tests belong to [792c7s4 vault-http-server tech spec](./792c_implementation-plan.md#792c7s4-vault-http-server-tech-spec).
  - No change to `engram serve --help` and no shipped artifact other than a code comment was edited.
- **Divergences noted ([792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated)):**
  - `engram serve` also writes the vault ID file and warns on a location mismatch at startup. Those are [792a2 vault-identity](./792a_product-spec.md#792a2-added-vault-identity)'s
    ([792a2h Stable vault ID](./792a_product-spec.md#792a2h-stable-vault-id), [792a2j Corrupt vault ID](./792a_product-spec.md#792a2j-corrupt-vault-id), [792a2p Copied and cloned vaults pause exchange](./792a_product-spec.md#792a2p-copied-and-cloned-vaults-pause-exchange)).
- **Rationale:** [792d3 Skills and guidance are product behavior](#792d3-skills-and-guidance-are-product-behavior), [792d30 Components are migrated once and extended later](#792d30-components-are-migrated-once-and-extended-later), [792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated), [792d52 Test IDs: product scenario for end-to-end checks, technical scenario otherwise](#792d52-test-ids-product-scenario-for-end-to-end-checks-technical-scenario-otherwise), [792d67 note-provenance covers detection and the first-write stamp](#792d67-note-provenance-covers-detection-and-the-first-write-stamp) and [792d103 install-and-update scope, ownership of the update run, and the behaviors the code shows](#792d103-install-and-update-scope-ownership-of-the-update-run-and-the-behaviors-the-code-shows).
- **Alternatives:** trim `vault-serve-api` per requirement now. Rejected because each requirement is both product and
  server contract.

## 792d118 exchange-versioning scope, requirements with no product scenario yet, and the tests it keeps apart

- **Choice:**
  - [792b33 exchange-versioning](./792b_technical-spec.md#792b33-added-exchange-versioning) specifies nine requirements ([792b33c1 The exchange hash](./792b_technical-spec.md#792b33c1-the-exchange-hash) to [792b33c9 Recording an offer receipt](./792b_technical-spec.md#792b33c9-recording-an-offer-receipt)). It owns what an exchange hash covers and how two
    compare, the optional exchange fields of a note, the lazy `xid`, the edit of the `aliases:` and `parent:` blocks
    as YAML nodes, and the recording of an offer receipt. When to hash, send, pull or fold is each exchange path's
    ([792b35 vault-http-server](./792b_technical-spec.md#792b35-added-vault-http-server), [792a21 parent-sync](./792a_product-spec.md#792a21-added-parent-sync), [792a22 curate](./792a_product-spec.md#792a22-added-curate)), and the YAML-node machinery it reuses is [792b6 note-document-format](./792b_technical-spec.md#792b6-added-note-document-format)'s ([792b6c7 Frontmatter node edits](./792b_technical-spec.md#792b6c7-frontmatter-node-edits)).
  - Only [792b33c1 The exchange hash](./792b_technical-spec.md#792b33c1-the-exchange-hash), [792b33c6 The exchange fields](./792b_technical-spec.md#792b33c6-the-exchange-fields) and [792b33c7 The lazy exchange ID](./792b_technical-spec.md#792b33c7-the-lazy-exchange-id) support a scenario of [792a20 vault-serving](./792a_product-spec.md#792a20-added-vault-serving) (a receipt's `stored_hash`, the raw
    envelope and header, the dedupe keys, and the fields a caller cannot set). The other requirements are consumed by
    [792a21 parent-sync](./792a_product-spec.md#792a21-added-parent-sync) and [792a22 curate](./792a_product-spec.md#792a22-added-curate), which are not yet migrated, so they say `Supports: none` and name the capability that will
    map them ([792d110 agent-instruction-bundle scope, and requirements with no product scenario](#792d110-agent-instruction-bundle-scope-and-requirements-with-no-product-scenario) for a requirement with no product scenario).
  - The fold's refusal and unknown-key scenarios ([792b33c8a3 An anchored edited key is refused by a fold](./792b_technical-spec.md#792b33c8a-scenarios), [792b33c8a4 A node edit either refuses or keeps every unknown key](./792b_technical-spec.md#792b33c8a-scenarios), [792b33c8a5 A fold keeps unknown keys under parent](./792b_technical-spec.md#792b33c8a-scenarios), [792b33c8a6 A note without unknown keys is written as before](./792b_technical-spec.md#792b33c8a-scenarios)) are the edit's
    contract as seen through the fold; the fold itself (the union of aliases and links, the judged-version check) is
    [792b34 pending-offer-bookkeeping](./792b_technical-spec.md#792b34-added-pending-offer-bookkeeping)'s ([792b34 pending-offer-bookkeeping](./792b_technical-spec.md#792b34-added-pending-offer-bookkeeping)) and [792a22 curate](./792a_product-spec.md#792a22-added-curate)'s.
  - `TestSetXIDField` also asserts that a receipt on a note with no frontmatter fails; it takes [792b33c7a1 The stamp goes before the exchange blocks](./792b_technical-spec.md#792b33c7a-scenarios) and lists
    [792b33c7a2 An existing xid is kept](./792b_technical-spec.md#792b33c7a-scenarios), [792b33c7a3 A note with no frontmatter cannot be stamped](./792b_technical-spec.md#792b33c7a-scenarios) and [792b33c9a6 A note with no frontmatter cannot take a receipt](./792b_technical-spec.md#792b33c9a-scenarios) as `split deferred`. `TestExchangeHash_UnknownPerConsumer` takes [792b33c5a2 Unknown is not a match and not a change](./792b_technical-spec.md#792b33c5a-scenarios)
    and lists [792b33c5a3 Equal matches and changed changes](./792b_technical-spec.md#792b33c5a-scenarios). `Test792b6c7a7_RunAmend_ExchangeHashParityWithPreChangeAmend` and `Test792b6c8a1_ToLF` were
    named by [792b6 note-document-format](./792b_technical-spec.md#792b6-added-note-document-format)'s node and are not renamed again.
  - Four CRLF tests in `exchange_crlf_test` verify other components' scenarios (the outbox drain, the pull-down, the
    pending marker, the header of `show`) and keep their names. `exchangefields_survival_test` stays with [792a21 parent-sync](./792a_product-spec.md#792a21-added-parent-sync)
    ([792d61 note-document-format's edit, CRLF and field-survival contracts arrive with their scenarios](#792d61-note-document-formats-edit-crlf-and-field-survival-contracts-arrive-with-their-scenarios)).
  - ADR-0029 keeps the consumers of the hash (loop suppression, `offer.key`, pull-down skip, decline matching, re-arm,
    dedupe rule 2, `--expect-hash`) and the survival, rename and dedupe sentences of D4; its D3 and D4 headings now
    point at [792b33 exchange-versioning](./792b_technical-spec.md#792b33-added-exchange-versioning).
- **Divergences noted ([792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated)):**
  - ADR-0029 D3 says the canonical body of "one layout" changed in place under `xh1:`. [792b33c3 The canonical body](./792b_technical-spec.md#792b33c3-the-canonical-body) states the current
    rule only, and keeps the version-skew effects as rationale.
  - ADR-0029 D3 lists the offered fields as "`type`, `situation`, the fact/feedback/runbook fields, `done_when`,
    `red_flags`, `triggers`, and the body"; the code hashes exactly eleven frontmatter keys plus the body, and [792b33c1 The exchange hash](./792b_technical-spec.md#792b33c1-the-exchange-hash)
    names them.
- **Rationale:** [792d3 Skills and guidance are product behavior](#792d3-skills-and-guidance-are-product-behavior), [792d10 Test renaming](#792d10-test-renaming), [792d30 Components are migrated once and extended later](#792d30-components-are-migrated-once-and-extended-later), [792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated), [792d52 Test IDs: product scenario for end-to-end checks, technical scenario otherwise](#792d52-test-ids-product-scenario-for-end-to-end-checks-technical-scenario-otherwise), [792d61 note-document-format's edit, CRLF and field-survival contracts arrive with their scenarios](#792d61-note-document-formats-edit-crlf-and-field-survival-contracts-arrive-with-their-scenarios) and [792d110 agent-instruction-bundle scope, and requirements with no product scenario](#792d110-agent-instruction-bundle-scope-and-requirements-with-no-product-scenario).
- **Alternatives:** map the hash to [792a21 parent-sync](./792a_product-spec.md#792a21-added-parent-sync) scenarios now. Rejected because those scenarios do not exist yet.

## 792d119 pending-offer-bookkeeping scope, the requirements taken out of vault-offer-curation, and the tests it takes

- **Choice:**
  - [792b34 pending-offer-bookkeeping](./792b_technical-spec.md#792b34-added-pending-offer-bookkeeping) specifies eight requirements ([792b34c1 The pending-offer marker](./792b_technical-spec.md#792b34c1-the-pending-offer-marker) to [792b34c8 The exchange-hash header of show](./792b_technical-spec.md#792b34c8-the-exchange-hash-header-of-show)). It takes six requirements out of the
    `vault-offer-curation` spec because each one is the contract of the marker, the scan, the check or the fold that
    `offer.go` and `amend_fold.go` realize: the marker on skill runbooks and on every type, the stateless scan, the
    judged-version check, the merge of parent links and the exchange-hash header of `engram show`. The product
    requirements of that spec stay for curate ([792c7u1 curate product spec](./792c_implementation-plan.md#792c7u1-curate-product-spec)): served writes land pending, `activate` commits directly,
    pending offers are excluded from query results, the judgment, the outcome not reported and the three surfacing
    points. [792c7u1 curate product spec](./792c_implementation-plan.md#792c7u1-curate-product-spec) maps its scenarios onto [792b34 pending-offer-bookkeeping](./792b_technical-spec.md#792b34-added-pending-offer-bookkeeping).
  - The spec's pointer in its Purpose names [792b34c1 The pending-offer marker](./792b_technical-spec.md#792b34c1-the-pending-offer-marker) to [792b34c8 The exchange-hash header of show](./792b_technical-spec.md#792b34c8-the-exchange-hash-header-of-show), as the earlier trimmed specs name their targets.
  - The three scenarios that trailed the `show` requirement of that spec belong to the judged-version check (two) and
    to the header (the CRLF note), and move with them.
  - The `engram show` header has no component of its own in the inventory, and
    it exists so that a curator can name the judged version, so it is a requirement of this component. Its served form
    ([792b35 vault-http-server](./792b_technical-spec.md#792b35-added-vault-http-server)) and its parent-fallback label order ([792a21 parent-sync](./792a_product-spec.md#792a21-added-parent-sync)) are theirs.
  - The query payload's flag and hint are rendered by [792b23 query-payload-renderer](./792b_technical-spec.md#792b23-added-query-payload-renderer); the two tests of the rendered hint are
    renamed to [792b34c4a2 The hint follows the flag](./792b_technical-spec.md#792b34c4a-scenarios) here, because the scenario is about the instruction following the flag. The tests of the
    update notice and of the lead position of the flag in the payload (`TestQueryPayload_PendingOffersLeadPayload`) are
    curate's and keep their names.
  - A bare `--discard` also records a decline of the parent note; that is pull-down's ([792a21 parent-sync](./792a_product-spec.md#792a21-added-parent-sync)), not specified here.
  - `TestVaultHasPendingOffers` and `TestWarnIfPendingOffers` are tables or sequences over several scenarios; each
    takes the first and lists the rest as `split deferred`. So does the property test of the fold, which also
    verifies [792b34c7a1 An existing link keeps its role and takes the offer's hash](./792b_technical-spec.md#792b34c7a-scenarios) and [792b34c7a2 A vault switch drops the links under the old vault](./792b_technical-spec.md#792b34c7a-scenarios).
- **Divergences noted ([792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated)):**
  - The spec says a hash that "cannot be computed at all (S21)" fails; the code returns the hashing error, and no test
    asserts it for a given expected hash (a coverage gap).
  - The spec says the fold's note-level effects cover "the offer's basename and all of its `aliases`"; the code also
    excludes E's own basename and duplicates.
- **Rationale:** [792d3 Skills and guidance are product behavior](#792d3-skills-and-guidance-are-product-behavior), [792d10 Test renaming](#792d10-test-renaming), [792d30 Components are migrated once and extended later](#792d30-components-are-migrated-once-and-extended-later), [792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated) and [792d118 exchange-versioning scope, requirements with no product scenario yet, and the tests it keeps apart](#792d118-exchange-versioning-scope-requirements-with-no-product-scenario-yet-and-the-tests-it-keeps-apart).
- **Alternatives:** leave the six requirements for curate. Rejected because they are not product behavior: no user
  observes the fold's link rules, only the result of curation.

## 792d120 vault-http-server scope, the fetch boundary, and what the built server showed

- **Choice:**
  - [792b35 vault-http-server](./792b_technical-spec.md#792b35-added-vault-http-server) specifies seventeen requirements ([792b35c1 The route table](./792b_technical-spec.md#792b35c1-the-route-table) to [792b35c17 The bounded fetch client](./792b_technical-spec.md#792b35c17-the-bounded-fetch-client)) and one sequence, the served `learn`, whose `alt`
    branches each have their own requirement ([792b35c9 Refusing an invalid offer](./792b_technical-spec.md#792b35c9-refusing-an-invalid-offer) to [792b35c14 A new pending offer](./792b_technical-spec.md#792b35c14-a-new-pending-offer)), as the guide asks. Its orientation names the
    component a repository-owned dependency-boundary interaction, as the catalog does. The `net/http` types stay in
    `cmd/engram` (ADR-0020), so every other requirement runs over primitives.
  - The bounded client a child fetches through is in `cmd/engram/serve.go` beside the server, and
    `serve_config_test` is listed for this node, so [792b35c17 The bounded fetch client](./792b_technical-spec.md#792b35c17-the-bounded-fetch-client) specifies its timeouts and read limit. The exchange that
    uses it (the outbox, the merged query, the pull-down, and the check that a response over 16 MiB is an error) is
    [792a21 parent-sync](./792a_product-spec.md#792a21-added-parent-sync)'s, which maps [792a20n1c An oversized parent response is refused](./792a_product-spec.md#792a20n1-uat-scenarios); [792b35c17 The bounded fetch client](./792b_technical-spec.md#792b35c17-the-bounded-fetch-client) says `Supports: none` for that reason.
  - `vault-serve-api` is deleted with this node, as [792d117 vault-serving scope, what stays for the technical node, and where the sources differ from the server](#792d117-vault-serving-scope-what-stays-for-the-technical-node-and-where-the-sources-differ-from-the-server) said. The component view's serve node is shortened to a
    pointer and the serve-side sentence of its K14 row is replaced by one, since the row's remaining text is [792a21 parent-sync](./792a_product-spec.md#792a21-added-parent-sync)'s.
    The comments in the code and the three OpenSpec specs that cited `vault-serve-api` now cite [792a20 vault-serving](./792a_product-spec.md#792a20-added-vault-serving) or [792b35 vault-http-server](./792b_technical-spec.md#792b35-added-vault-http-server).
  - Two scenarios have no technical owner: [792a20b1c The help states the address rule](./792a_product-spec.md#792a20b1-uat-scenarios) (the help text of `serve`) and [792a20l1f Local activate keeps paths](./792a_product-spec.md#792a20l1-uat-scenarios) (the local `activate` keeps
    absolute and vault-relative paths). The first is a string of the command definition, the second a behavior of
    `activate` that no component of the inventory specifies ([792b14 note-amendment](./792b_technical-spec.md#792b14-added-note-amendment)'s [792b14c9 Recording a use](./792b_technical-spec.md#792b14c9-recording-a-use) is the use bump only). Both are
    listed as coverage gaps.
  - Two real-mux tests that drive a whole route over TCP are named for the product scenario they assert ([792d52 Test IDs: product scenario for end-to-end checks, technical scenario otherwise](#792d52-test-ids-product-scenario-for-end-to-end-checks-technical-scenario-otherwise)):
    `Test792a20e1c_ServeAndFetch_LearnRoundTrip` and `Test792a20c1b_ServeAndFetch_ShowRoundTrip`. The tests that call a
    handler directly take technical IDs.
  - `TestServeBodyCaps` asserts three constants; it takes [792b35c3a3 The request cap is a few MiB](./792b_technical-spec.md#792b35c3a-scenarios) and lists [792b35c17a2 The read limit is one byte over the accepted maximum](./792b_technical-spec.md#792b35c17a-scenarios) as `split deferred`. Likewise
    `TestServeLearn_InvalidOfferIsRejected` ([792b35c9a4 A key without an origin is a 400](./792b_technical-spec.md#792b35c9a-scenarios)), `TestServeLearn_AcceptedOfferRetry` ([792b35c12a1 A new key from an accepted origin makes a new pending note](./792b_technical-spec.md#792b35c12a-scenarios)),
    `TestServeLearn_ForResolution` ([792b35c13a2 A renamed target resolves through aliases](./792b_technical-spec.md#792b35c13a-scenarios) to [792b35c13a4 An unresolvable target is dropped](./792b_technical-spec.md#792b35c13a-scenarios)) and the RunServe registration test ([792b35c2a2 A close after cancellation is not an error](./792b_technical-spec.md#792b35c2a-scenarios)).
  - [792b35c2a3 A corrupt ID or a mismatched location warns and serves](./792b_technical-spec.md#792b35c2a-scenarios) maps to the serve-startup tests that [792a2 vault-identity](./792a_product-spec.md#792a2-added-vault-identity) already named (`Test792a2h1a`, `Test792a2j1a`,
    `Test792a2p1c`).
- **Divergences noted ([792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated)):**
  - The `vault-serve-api` spec says a served activate with a ref that is "an absolute path, contains a path separator
    or `..`" is a 400; the code also treats a name that is a listed note with a different case (`.MD`) as not found (404).
  - The spec does not say that a served `show` with an empty `note` is a 400 or that an unreadable or unparseable listed
    note is a 500; the code does, and [792b35c6 The show route](./792b_technical-spec.md#792b35c6-the-show-route) states it.
  - The spec says a served `activate` response is 200/404/200/500 by outcome only implicitly (as "commits directly");
    [792b35c7 The activate route](./792b_technical-spec.md#792b35c7-the-activate-route) states the four statuses from the code.
  - A GET to `learn` is answered 405 and an unrouted path 404 by the mux; the spec only says there is "no route".
  - The 409 loop body carries `reason` and `vault_id`; the spec's served-API requirement does not say so.
  - The `Cf-Access-Authenticated-User-Email` header field in the request type's doc comment is the remnant of the
    dropped edge authentication; no code reads it ([792a20e A served learn is attributed to the identity the caller declares](./792a_product-spec.md#792a20e-a-served-learn-is-attributed-to-the-identity-the-caller-declares)).
- **Rationale:** [792d3 Skills and guidance are product behavior](#792d3-skills-and-guidance-are-product-behavior), [792d30 Components are migrated once and extended later](#792d30-components-are-migrated-once-and-extended-later), [792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated), [792d52 Test IDs: product scenario for end-to-end checks, technical scenario otherwise](#792d52-test-ids-product-scenario-for-end-to-end-checks-technical-scenario-otherwise), [792d117 vault-serving scope, what stays for the technical node, and where the sources differ from the server](#792d117-vault-serving-scope-what-stays-for-the-technical-node-and-where-the-sources-differ-from-the-server) and [792d118 exchange-versioning scope, requirements with no product scenario yet, and the tests it keeps apart](#792d118-exchange-versioning-scope-requirements-with-no-product-scenario-yet-and-the-tests-it-keeps-apart).
- **Alternatives:** leave the fetch client to [792a21 parent-sync](./792a_product-spec.md#792a21-added-parent-sync). Rejected because `cmd/engram/serve.go` realizes both and its
  connect-timeout test is listed for this node.

## 792d121 parent-sync scope, what stays whole, the components extended, and where the sources differ from the code

- **Choice:**
  - [792a21 parent-sync](./792a_product-spec.md#792a21-added-parent-sync) states thirty-seven requirements ([792a21a ENGRAM_SERVER is refused](./792a_product-spec.md#792a21a-engram_server-is-refused) to [792a21zl Exchange fields survive every other frontmatter rewrite](./792a_product-spec.md#792a21zl-exchange-fields-survive-every-other-frontmatter-rewrite)) and 178 scenarios. The legacy specs give most of
    their wording; the rest comes from their tests and from the code: a looping offer and a vault
    that is its own parent ([792a21y A looping offer is refused](./792a_product-spec.md#792a21y-a-looping-offer-is-refused)), the vault-ID probe before a drain ([792a21r The outbox drains after the parent was reached, without the lock](./792a_product-spec.md#792a21r-the-outbox-drains-after-the-parent-was-reached-without-the-lock)), an offer withdrawn at send time
    ([792a21r The outbox drains after the parent was reached, without the lock](./792a_product-spec.md#792a21r-the-outbox-drains-after-the-parent-was-reached-without-the-lock)), an offer with no author ([792a21x An offer with no author waits](./792a_product-spec.md#792a21x-an-offer-with-no-author-waits)), the `attention` state ([792a21w A receipt the note refuses needs attention](./792a_product-spec.md#792a21w-a-receipt-the-note-refuses-needs-attention)), the parent contact pauses of a
    pull-down and the hash check on an envelope ([792a21zd A pulled-down note is an exact-content local pending offer](./792a_product-spec.md#792a21zd-a-pulled-down-note-is-an-exact-content-local-pending-offer), [792a21zg A pull-down tells the parent the note was used](./792a_product-spec.md#792a21zg-a-pull-down-tells-the-parent-the-note-was-used)), the use signal ([792a21zg A pull-down tells the parent the note was used](./792a_product-spec.md#792a21zg-a-pull-down-tells-the-parent-the-note-was-used)) and the order of the
    label and the header of a fallback `show` ([792a21l A note can be shown from the parent](./792a_product-spec.md#792a21l-a-note-can-be-shown-from-the-parent)). `ENGRAM_SERVER` is a **MUST NOT**-style refusal ([792d6 Removed features are not documented; enduring absences become MUST NOT](#792d6-removed-features-are-not-documented-enduring-absences-become-must-not)).
  - `vault-merged-recall`, `vault-parent-offers` and `vault-parent-pulldown` stay whole. Each requirement of them is
    both a behavior of the product and a contract of the outbox or of the client, so they are deleted by the technical
    nodes that read them: `vault-parent-offers` by [792c7t2 offer-outbox tech spec](./792c_implementation-plan.md#792c7t2-offer-outbox-tech-spec) and the other two by [792c7t3 parent-vault-client tech spec](./792c_implementation-plan.md#792c7t3-parent-vault-client-tech-spec). Their hash requirement is
    already [792b33c1 The exchange hash](./792b_technical-spec.md#792b33c1-the-exchange-hash) to [792b33c5 Comparing two hashes](./792b_technical-spec.md#792b33c5-comparing-two-hashes) and is deleted with the spec. `vault-local-first` is deleted whole, since its only
    requirement is [792a21a ENGRAM_SERVER is refused](./792a_product-spec.md#792a21a-engram_server-is-refused). `vault-note-identity` keeps the two clauses that wait for [792a22 curate](./792a_product-spec.md#792a22-added-curate): the bookkeeping rule for
    `--clear-pending` and `--discard --into`, and the fold's LF read and the judged-version check on the LF form. Its
    last requirement, that exchange fields cannot be set over the served API, is deleted as the duplicate of [792a20j A caller cannot set link, identity or registration fields](./792a_product-spec.md#792a20j-a-caller-cannot-set-link-identity-or-registration-fields) that
    [792d117 vault-serving scope, what stays for the technical node, and where the sources differ from the server](#792d117-vault-serving-scope-what-stays-for-the-technical-node-and-where-the-sources-differ-from-the-server) said it was.
  - ADR-0029 keeps D3's `--expect-hash` consumer, D10 and the consequence and risks that concern curation, for [792c7u1 curate product spec](./792c_implementation-plan.md#792c7u1-curate-product-spec).
    D11, which records how the original change was tested (hermetic headless arms, a recording proxy, a sandbox
    allowlist), is not ported, since it describes a past verification effort and not a behavior; git keeps it. The S7 row
    of the system context is a pointer, and R7 and K14 wait for [792c7t2 offer-outbox tech spec](./792c_implementation-plan.md#792c7t2-offer-outbox-tech-spec) and [792c7t3 parent-vault-client tech spec](./792c_implementation-plan.md#792c7t3-parent-vault-client-tech-spec).
  - The catalog lists [792b24 query-engine](./792b_technical-spec.md#792b24-added-query-engine) for extension. It gains nothing: the merged query is a step that wraps the local
    pipeline, and its ordering, budgets and dedupe are the client's ([792b37 parent-vault-client](./792b_technical-spec.md#792b37-added-parent-vault-client), [792c7t3 parent-vault-client tech spec](./792c_implementation-plan.md#792c7t3-parent-vault-client-tech-spec)), so [792b24 query-engine](./792b_technical-spec.md#792b24-added-query-engine)'s contract is
    unchanged. `query-payload-renderer` ([792b23c9 The item tags of a merge](./792b_technical-spec.md#792b23c9-the-item-tags-of-a-merge), the `from_parent` tag), [792b27 note-reparenting](./792b_technical-spec.md#792b27-added-note-reparenting) (a mapping row for
    [792b27c6a7 The old basename is recorded as an alias](./792b_technical-spec.md#792b27c6a-scenarios)) and `self-installer` ([792b28c13a9 The outbox notice is written](./792b_technical-spec.md#792b28c13a-scenarios), the outbox notice) are extended instead, as the notes of [792c7f4 query-payload-renderer tech spec](./792c_implementation-plan.md#792c7f4-query-payload-renderer-tech-spec) and
    [792c7j2 self-installer tech spec](./792c_implementation-plan.md#792c7j2-self-installer-tech-spec) asked.
  - The parent cache and its backoff, the exchange-identifier checks and the declined-pull record are three
    requirements of vault-identity-registry ([792b5c14 The parent cache and the contact backoff](./792b_technical-spec.md#792b5c14-the-parent-cache-and-the-contact-backoff) to [792b5c16 The declined-pull record](./792b_technical-spec.md#792b5c16-the-declined-pull-record)), as its orientation said. The record's code is in
    `pulldown.go`; the component is named for what it keeps in `.engram/` and not for the file.
  - Renames: 105 tests. Every test that runs a command through `Targets` and asserts a product scenario takes the
    scenario's ID ([792d52 Test IDs: product scenario for end-to-end checks, technical scenario otherwise](#792d52-test-ids-product-scenario-for-end-to-end-checks-technical-scenario-otherwise)), which is why the activate, outbox-wiring and merged-query dispatch tests are renamed here and
    not by [792c7t2 offer-outbox tech spec](./792c_implementation-plan.md#792c7t2-offer-outbox-tech-spec) and [792c7t3 parent-vault-client tech spec](./792c_implementation-plan.md#792c7t3-parent-vault-client-tech-spec). `parent_backoff_test` is renamed here, to [792b5c14 The parent cache and the contact backoff](./792b_technical-spec.md#792b5c14-the-parent-cache-and-the-contact-backoff), though the catalog lists it for
    [792c7t3 parent-vault-client tech spec](./792c_implementation-plan.md#792c7t3-parent-vault-client-tech-spec). `TestDrainOutbox_LocationCheckFailureSendsNothing` takes [792a2p1a A cloned child is caught before it offers](./792a_product-spec.md#792a2p1-uat-scenarios) and
    `TestTargets_Query_Merged_SelfParentReturnsLocalOnly` takes [792a2q1a A copied vault served back to itself](./792a_product-spec.md#792a2q1-uat-scenarios), the two scenarios that [792c7a5 vault-identity-registry tech spec](./792c_implementation-plan.md#792c7a5-vault-identity-registry-tech-spec) left for
    this node. A test that covers several scenarios takes the first and lists the others as `split deferred`.
- **Divergences noted ([792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated)):**
  - The legacy spec says the fallback labels its output as parent-sourced and does not say what the explicit `--parent`
    route prints; it prints the parent's answer unlabeled ([792a21l A note can be shown from the parent](./792a_product-spec.md#792a21l-a-note-can-be-shown-from-the-parent), [792b2c10 Parent routing of note lookup](./792b_technical-spec.md#792b2c10-parent-routing-of-note-lookup)).
  - `--pending=false` clears a pending marker as `--clear-pending` does, so it is offered like one when the note has
    `offer.origin`; the spec names only `--clear-pending`.
  - The merged query does not check the version of the parent: an old parent's answer merges without dedupe keys and a
    note can show twice (#785). `scanDedupeNotes` reads each local note twice (#782), and the self-parent guard can miss
    on a transient vault-ID read failure (#786). These are known gaps and are not in the specs.
- **Rationale:** [792d3 Skills and guidance are product behavior](#792d3-skills-and-guidance-are-product-behavior), [792d6 Removed features are not documented; enduring absences become MUST NOT](#792d6-removed-features-are-not-documented-enduring-absences-become-must-not), [792d10 Test renaming](#792d10-test-renaming), [792d30 Components are migrated once and extended later](#792d30-components-are-migrated-once-and-extended-later), [792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated), [792d51 command-line-surface routing requirements arrive with their scenarios](#792d51-command-line-surface-routing-requirements-arrive-with-their-scenarios), [792d52 Test IDs: product scenario for end-to-end checks, technical scenario otherwise](#792d52-test-ids-product-scenario-for-end-to-end-checks-technical-scenario-otherwise), [792d61 note-document-format's edit, CRLF and field-survival contracts arrive with their scenarios](#792d61-note-document-formats-edit-crlf-and-field-survival-contracts-arrive-with-their-scenarios), [792d67 note-provenance covers detection and the first-write stamp](#792d67-note-provenance-covers-detection-and-the-first-write-stamp), [792d117 vault-serving scope, what stays for the technical node, and where the sources differ from the server](#792d117-vault-serving-scope-what-stays-for-the-technical-node-and-where-the-sources-differ-from-the-server), [792d118 exchange-versioning scope, requirements with no product scenario yet, and the tests it keeps apart](#792d118-exchange-versioning-scope-requirements-with-no-product-scenario-yet-and-the-tests-it-keeps-apart) and [792d119 pending-offer-bookkeeping scope, the requirements taken out of vault-offer-curation, and the tests it takes](#792d119-pending-offer-bookkeeping-scope-the-requirements-taken-out-of-vault-offer-curation-and-the-tests-it-takes).
- **Alternatives:** trim the three parent specs requirement by requirement now. Rejected because each requirement is
  both product and contract, as [792d117 vault-serving scope, what stays for the technical node, and where the sources differ from the server](#792d117-vault-serving-scope-what-stays-for-the-technical-node-and-where-the-sources-differ-from-the-server) found for `vault-serve-api`.

## 792d122 offer-outbox scope, what the spec leaves to other components, and where the sources differ from the code

- **Choice:**
  - [792b36 offer-outbox](./792b_technical-spec.md#792b36-added-offer-outbox) specifies sixteen requirements ([792b36c1 Which writes are offered](./792b_technical-spec.md#792b36c1-which-writes-are-offered) to [792b36c16 The update exchange and the outbox notice](./792b_technical-spec.md#792b36c16-the-update-exchange-and-the-outbox-notice)) and two sequences, a write that is offered and a drain.
    Every `alt` branch of the drain has its own requirement ([792b36c3 An offer withdrawn at send time](./792b_technical-spec.md#792b36c3-an-offer-withdrawn-at-send-time), c6, c8 to c13), as the guide asks. The outbox
    state that waits (`attention`) is one requirement ([792b36c15 An entry that waits](./792b_technical-spec.md#792b36c15-an-entry-that-waits)), because a refused receipt and a missing user identity
    share the state, the single warning at the transition and the silent retry on later drains.
  - The `vault-parent-offers` spec is deleted whole. Its hash requirement is [792b33c1 The exchange hash](./792b_technical-spec.md#792b33c1-the-exchange-hash) to [792b33c5 Comparing two hashes](./792b_technical-spec.md#792b33c5-comparing-two-hashes) ([792d118 exchange-versioning scope, requirements with no product scenario yet, and the tests it keeps apart](#792d118-exchange-versioning-scope-requirements-with-no-product-scenario-yet-and-the-tests-it-keeps-apart)), the product side
    of it is [792a21o Only a change to offered content is a change](./792a_product-spec.md#792a21o-only-a-change-to-offered-content-is-a-change), and the rest is [792a21n A local write is offered to the parent](./792a_product-spec.md#792a21n-a-local-write-is-offered-to-the-parent) to [792a21zb A note with CRLF line endings exchanges as its LF form](./792a_product-spec.md#792a21zb-a-note-with-crlf-line-endings-exchanges-as-its-lf-form) and [792b36 offer-outbox](./792b_technical-spec.md#792b36-added-offer-outbox). Its connect timeout of 3 seconds is the fetch
    client's ([792b35c17 The bounded fetch client](./792b_technical-spec.md#792b35c17-the-bounded-fetch-client)) and its backoff is [792b5 vault-identity-registry](./792b_technical-spec.md#792b5-added-vault-identity-registry)'s ([792b5c14 The parent cache and the contact backoff](./792b_technical-spec.md#792b5c14-the-parent-cache-and-the-contact-backoff)). The backoff gate sits in front of the
    drain, so the drain's own requirement ([792b36c5 Before a drain sends](./792b_technical-spec.md#792b36c5-before-a-drain-sends)) names it and does not restate it.
  - The receipt edit that the drain calls is [792b33 exchange-versioning](./792b_technical-spec.md#792b33-added-exchange-versioning)'s ([792b33c9 Recording an offer receipt](./792b_technical-spec.md#792b33c9-recording-an-offer-receipt)); the outbox owns when it is applied, what
    finishes the entry afterwards and the `attention` state when the note refuses it.
  - The vault-ID probe before a drain (`GET /query?dedupe-keys=1` with one item) is in the code and in no spec: it is
    [792b36c5 Before a drain sends](./792b_technical-spec.md#792b36c5-before-a-drain-sends), with a scenario for each way it fails.
  - `offer_wiring_test` and `identity_gaps_test` hold two tests of `update`'s exchange step through the production wiring.
    Each asserts a product scenario ([792a21x1a No user identity holds the offer](./792a_product-spec.md#792a21x1-uat-scenarios), [792a21za1b Update reports an entry that needs attention](./792a_product-spec.md#792a21za1-uat-scenarios)) and so takes the product ID ([792d52 Test IDs: product scenario for end-to-end checks, technical scenario otherwise](#792d52-test-ids-product-scenario-for-end-to-end-checks-technical-scenario-otherwise)); the other
    `TestUpdateExchange_*` tests were renamed by [792c7t1 parent-sync product spec](./792c_implementation-plan.md#792c7t1-parent-sync-product-spec). The three `TestDrainForCommand_*` and `TestRecordKeptReceipts_*`
    tests call internal drain steps and take technical IDs, listing the product scenario as split deferred.
  - Renames: 69 tests, of which two take product IDs. The scenarios that the tests renamed by [792c7t1 parent-sync product spec](./792c_implementation-plan.md#792c7t1-parent-sync-product-spec) verify are listed in
    the traceability table as references.
- **Divergences noted ([792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated)):**
  - The spec says a malformed receipt is handled as an undecodable reply, leaving the note unlinked and the entry queued. The
    code classes an undecodable or malformed 2xx body as an outage (`failed`), so the parent is also backed off for the
    next commands, though it answered.
  - The spec's reader list for a CRLF note does not name the vault-ID probe, which reads no note.
  - The spec does not say what `update --dry-run` does with the outbox; the code drains nothing and still prints the
    notice ([792b36c16 The update exchange and the outbox notice](./792b_technical-spec.md#792b36c16-the-update-exchange-and-the-outbox-notice)).
  - A rejected entry's error is the parent's message; the spec says only that it is shown. The code also stores an empty
    error for a rejection that carries no cause ([792b36c9 A rejected offer](./792b_technical-spec.md#792b36c9-a-rejected-offer)).
- **Rationale:** [792d3 Skills and guidance are product behavior](#792d3-skills-and-guidance-are-product-behavior), [792d10 Test renaming](#792d10-test-renaming), [792d30 Components are migrated once and extended later](#792d30-components-are-migrated-once-and-extended-later), [792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated), [792d52 Test IDs: product scenario for end-to-end checks, technical scenario otherwise](#792d52-test-ids-product-scenario-for-end-to-end-checks-technical-scenario-otherwise), [792d118 exchange-versioning scope, requirements with no product scenario yet, and the tests it keeps apart](#792d118-exchange-versioning-scope-requirements-with-no-product-scenario-yet-and-the-tests-it-keeps-apart), [792d119 pending-offer-bookkeeping scope, the requirements taken out of vault-offer-curation, and the tests it takes](#792d119-pending-offer-bookkeeping-scope-the-requirements-taken-out-of-vault-offer-curation-and-the-tests-it-takes) and [792d121 parent-sync scope, what stays whole, the components extended, and where the sources differ from the code](#792d121-parent-sync-scope-what-stays-whole-the-components-extended-and-where-the-sources-differ-from-the-code).
- **Alternatives:** one requirement per outcome of `drainOutbox` with no gate requirement. Rejected because the gate has
  its own failure paths (the probe, the location check, the cached self-parent) that no outcome covers.

## 792d123 parent-vault-client scope, the one component for two directions, and where the sources differ from the code

- **Choice:**
  - [792b37 parent-vault-client](./792b_technical-spec.md#792b37-added-parent-vault-client) specifies sixteen requirements ([792b37c1 The parent address](./792b_technical-spec.md#792b37c1-the-parent-address) to [792b37c16 Re-checking a linked note's parent copy](./792b_technical-spec.md#792b37c16-re-checking-a-linked-notes-parent-copy)) and two sequences, a merged query and a pull-down. It is
    one component for both directions of the child's reading, as the catalog has it: they share the request, the gate, the
    envelope and the vault-ID bookkeeping. The merge ([792b37c5 Dropping and tagging the parent's items](./792b_technical-spec.md#792b37c5-dropping-and-tagging-the-parents-items) to [792b37c9 The merged payload's own fields](./792b_technical-spec.md#792b37c9-the-merged-payloads-own-fields)) is pure functions over payloads and notes, and
    the pull-down ([792b37c11 A pull-down is a gated parent contact](./792b_technical-spec.md#792b37c11-a-pull-down-is-a-gated-parent-contact) to [792b37c16 Re-checking a linked note's parent copy](./792b_technical-spec.md#792b37c16-re-checking-a-linked-notes-parent-copy)) runs through an injected fetch capability.
  - The decision that `engram query` takes the merged path when `ENGRAM_PARENT` is set is made in `targets.go`, the caller.
    [792b37c4 The merged query](./792b_technical-spec.md#792b37c4-the-merged-query) states it with the path it selects and not as a requirement of [792b2 command-line-surface](./792b_technical-spec.md#792b2-added-command-line-surface), since a one-sentence routing requirement
    would add a component section for no contract of its own. The unset half has no test (a coverage gap).
  - The `vault-merged-recall` and `vault-parent-pulldown` specs are deleted. Their cap on the merged `--limit` by position
    and the note floor are [792b37c7 The order of the merged relevance items](./792b_technical-spec.md#792b37c7-the-order-of-the-merged-relevance-items); their use of the pull-down's skip rule by the re-check is [792b37c16 Re-checking a linked note's parent copy](./792b_technical-spec.md#792b37c16-re-checking-a-linked-notes-parent-copy). The old design
    notes of the merge (score-sort interleave, no score normalization) are superseded by the direct and explore order and
    are not kept.
  - The child side of K14, the edge and row of C2 to S7 and the edge of R7 are pointers to [792a21 parent-sync](./792a_product-spec.md#792a21-added-parent-sync), [792b35 vault-http-server](./792b_technical-spec.md#792b35-added-vault-http-server), [792b36 offer-outbox](./792b_technical-spec.md#792b36-added-offer-outbox) and [792b37 parent-vault-client](./792b_technical-spec.md#792b37-added-parent-vault-client).
  - Tests: 51 renamed, of which one (`TestActivate_PullDownOmitsEmptyUser`) is a command-level test of [792a21zh1a No user detectable on a pull-down](./792a_product-spec.md#792a21zh1-uat-scenarios) that
    [792c7t1 parent-sync product spec](./792c_implementation-plan.md#792c7t1-parent-sync-product-spec) did not list, because `first_write_user_test` was not among the files it examined. The pull-down and
    merged-query tests that run a command through `Targets` were renamed to product IDs by [792c7t1 parent-sync product spec](./792c_implementation-plan.md#792c7t1-parent-sync-product-spec) ([792d121 parent-sync scope, what stays whole, the components extended, and where the sources differ from the code](#792d121-parent-sync-scope-what-stays-whole-the-components-extended-and-where-the-sources-differ-from-the-code)), and
    the traceability table here lists them as the evidence for [792b37c4 The merged query](./792b_technical-spec.md#792b37c4-the-merged-query) and [792b37c11 A pull-down is a gated parent contact](./792b_technical-spec.md#792b37c11-a-pull-down-is-a-gated-parent-contact) to [792b37c16 Re-checking a linked note's parent copy](./792b_technical-spec.md#792b37c16-re-checking-a-linked-notes-parent-copy).
  - `TestShowChunkTarget_LocalDispatch`, `TestLocalAmend_*` and `TestLocalLearn_NeverSetsPendingMarker` stay unrenamed:
    the first verifies a local branch of `show-chunk` that no scenario states, and the others verify that a local write
    never sets the pending marker, which curate ([792c7u1 curate product spec](./792c_implementation-plan.md#792c7u1-curate-product-spec)) owns.
- **Divergences noted ([792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated)):**
  - The spec says a parent chunk is dropped; the code also drops any parent item whose provenances include `recent`,
    whatever its kind; a recency item is a chunk, so the second test selects the same items ([792b37c5 Dropping and tagging the parent's items](./792b_technical-spec.md#792b37c5-dropping-and-tagging-the-parents-items)).
  - The spec does not say that a failed local scan of the notes for dedupe is a warning and not an error; the code warns
    and merges without dedupe ([792b37c6 Deduping against the local notes](./792b_technical-spec.md#792b37c6-deduping-against-the-local-notes), #782 reads each note twice there).
- **Rationale:** [792d3 Skills and guidance are product behavior](#792d3-skills-and-guidance-are-product-behavior), [792d10 Test renaming](#792d10-test-renaming), [792d30 Components are migrated once and extended later](#792d30-components-are-migrated-once-and-extended-later), [792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated), [792d52 Test IDs: product scenario for end-to-end checks, technical scenario otherwise](#792d52-test-ids-product-scenario-for-end-to-end-checks-technical-scenario-otherwise), [792d121 parent-sync scope, what stays whole, the components extended, and where the sources differ from the code](#792d121-parent-sync-scope-what-stays-whole-the-components-extended-and-where-the-sources-differ-from-the-code) and [792d122 offer-outbox scope, what the spec leaves to other components, and where the sources differ from the code](#792d122-offer-outbox-scope-what-the-spec-leaves-to-other-components-and-where-the-sources-differ-from-the-code).
- **Alternatives:** two components, a merged-query client and a pull-down client. Rejected because both are one boundary
  with one gated contact and one vault-ID cache, and the catalog lists them as one.

## 792d124 curate scope, the scenarios only the skill text verifies, and where the sources differ from the skill and the code

- **Choice:**
  - [792a22 curate](./792a_product-spec.md#792a22-added-curate) states fifteen requirements ([792a22a Pending offers are left out of normal query results](./792a_product-spec.md#792a22a-pending-offers-are-left-out-of-normal-query-results) to [792a22o Pending offers are surfaced at three points](./792a_product-spec.md#792a22o-pending-offers-are-surfaced-at-three-points)) and 45 scenarios. The legacy `vault-offer-curation` spec gives
    the judgment, the outcome, the surfacing and the exclusion their wording; the skill text gives the order of the near
    outcome, the rejected outcome, the pending skill note and the re-judging after a failed hash; the code gives the rule
    that a local learn or amend never sets the marker ([792a22b A note becomes pending only by an exchange](./792a_product-spec.md#792a22b-a-note-becomes-pending-only-by-an-exchange)) and the order of the three outcomes. The judgment itself
    is the agent's, so [792a22d Curation judges each offer by its content against the host vault](./792a_product-spec.md#792a22d-curation-judges-each-offer-by-its-content-against-the-host-vault), [792a22e An amend-offer is judged against its target first](./792a_product-spec.md#792a22e-an-amend-offer-is-judged-against-its-target-first), [792a22i A rejected offer is discarded outright](./792a_product-spec.md#792a22i-a-rejected-offer-is-discarded-outright), [792a22j Curation edits existing notes and never composes a new learn](./792a_product-spec.md#792a22j-curation-edits-existing-notes-and-never-composes-a-new-learn) and [792a22k A pending skill runbook note is authored, not judged](./792a_product-spec.md#792a22k-a-pending-skill-runbook-note-is-authored-not-judged) are procedure scenarios with no test, as the skill groups
    before it ([792d114 route scope, the SKILL.md repoint, and where the sources differ from the skill](#792d114-route-scope-the-skillmd-repoint-and-where-the-sources-differ-from-the-skill), [792d115 please scope, and where the sources differ from the skill](#792d115-please-scope-and-where-the-sources-differ-from-the-skill)).
  - `vault-offer-curation`, `vault-note-identity` and ADR-0029 are deleted. The two clauses of `vault-note-identity` that waited
    for curate are [792a22h1b Accepting an offer keeps its author](./792a_product-spec.md#792a22h1-uat-scenarios) and [792a22m1d A fold converts a CRLF existing note](./792a_product-spec.md#792a22m1-uat-scenarios); ADR-0029's remaining decisions were D3's `--expect-hash` consumer ([792a22l A bookkeeping step names the version that was judged](./792a_product-spec.md#792a22l-a-bookkeeping-step-names-the-version-that-was-judged)), D10
    ([792a22h An absent offer is accepted with its declared author](./792a_product-spec.md#792a22h-an-absent-offer-is-accepted-with-its-declared-author), [792a22m Folding an offer keeps its identity and reads a CRLF note as LF](./792a_product-spec.md#792a22m-folding-an-offer-keeps-its-identity-and-reads-a-crlf-note-as-lf)) and the consequence and risks about curating ([792a22o Pending offers are surfaced at three points](./792a_product-spec.md#792a22o-pending-offers-are-surfaced-at-three-points)). D11, which records how the original change was
    verified, was dropped in [792d121 parent-sync scope, what stays whole, the components extended, and where the sources differ from the code](#792d121-parent-sync-scope-what-stays-whole-the-components-extended-and-where-the-sources-differ-from-the-code). Nothing else cites ADR-0029 except the rationale of [792a Documentation Framework Adoption - Product Spec](./792a_product-spec.md#documentation-framework-adoption---product-spec) and [792b Documentation Framework Adoption - Technical Spec](./792b_technical-spec.md#documentation-framework-adoption---technical-spec), which name it as a
    source and are left as the earlier nodes left their deleted ADRs.
  - [792c7u1d](./792c_implementation-plan.md#792c7u1-curate-product-spec) adds components to the catalog's list: [792b12 note-authoring](./792b_technical-spec.md#792b12-added-note-authoring) (a local learn never sets the marker, [792b12c8 The pending marker is not set by a local learn](./792b_technical-spec.md#792b12c8-the-pending-marker-is-not-set-by-a-local-learn)), [792b14 note-amendment](./792b_technical-spec.md#792b14-added-note-amendment) (settling a
    pending offer, [792b14c14 Settling a pending offer](./792b_technical-spec.md#792b14c14-settling-a-pending-offer), and a CRLF decline, [792b14c13a5 A CRLF pulled note's decline is recorded](./792b_technical-spec.md#792b14c13a-scenarios)), [792b23 query-payload-renderer](./792b_technical-spec.md#792b23-added-query-payload-renderer) (the lead of a payload with pending offers, [792b23c10 The lead of a payload with pending offers](./792b_technical-spec.md#792b23c10-the-lead-of-a-payload-with-pending-offers)),
    [792b28 self-installer](./792b_technical-spec.md#792b28-added-self-installer) (the pending-offer notice, [792b28c13a10 The pending-offer notice is written](./792b_technical-spec.md#792b28c13a-scenarios)), [792b22 runbook-trigger-matching](./792b_technical-spec.md#792b22-added-runbook-trigger-matching) and [792b35 vault-http-server](./792b_technical-spec.md#792b35-added-vault-http-server) (mapping rows) and [792b32 agent-instruction-bundle](./792b_technical-spec.md#792b32-added-agent-instruction-bundle) (one mapping row). The
    amend flags `--clear-pending` and `--discard` had no requirement in [792b14 note-amendment](./792b_technical-spec.md#792b14-added-note-amendment), which [792b14c14 Settling a pending offer](./792b_technical-spec.md#792b14c14-settling-a-pending-offer) now holds.
  - The `Supports:` lines of [792b34c1 The pending-offer marker](./792b_technical-spec.md#792b34c1-the-pending-offer-marker) to [792b34c8 The exchange-hash header of show](./792b_technical-spec.md#792b34c8-the-exchange-hash-header-of-show) now name the curate scenarios they realize; [792c7s3 pending-offer-bookkeeping tech spec](./792c_implementation-plan.md#792c7s3-pending-offer-bookkeeping-tech-spec) had left them `none`
    for this node ([792d119 pending-offer-bookkeeping scope, the requirements taken out of vault-offer-curation, and the tests it takes](#792d119-pending-offer-bookkeeping-scope-the-requirements-taken-out-of-vault-offer-curation-and-the-tests-it-takes)).
  - `SKILL.md` is not edited: it cites no removed document.
  - Renames: 15 tests. `TestRunAmend_ClearPendingKeepsOfferAuthor` takes the technical ID [792b14c14a8 An accepted offer keeps its author](./792b_technical-spec.md#792b14c14a-scenarios) and lists the product
    scenario, since it calls `RunAmend` and not a command. The tests that [792c7s3 pending-offer-bookkeeping tech spec](./792c_implementation-plan.md#792c7s3-pending-offer-bookkeeping-tech-spec) renamed for the marker, the exclusion, the
    scan, the instruction, the judged-version check, the fold and the header verify the mapped curate scenarios and are
    listed in the traceability prose and not repeated.
- **Divergences noted ([792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated)):**
  - The skill's scan, `grep -l '^pending: true$' <vault>/*.md`, does not match a note with CRLF line endings, which the
    binary treats as pending ([792b34c1a4 A CRLF offer is still pending](./792b_technical-spec.md#792b34c1a-scenarios), [792a21zb1c A CRLF pending offer stays pending](./792a_product-spec.md#792a21zb1-uat-scenarios)). A CRLF offer is therefore excluded from query and flagged, and the skill's
    own scan misses it.
  - The legacy spec says a bookkeeping step passes `--expect-hash` "on a served offer"; the binary requires it for any note
    that carries `offer.origin`, pending or live ([792b34c5 The judged-version check](./792b_technical-spec.md#792b34c5-the-judged-version-check)), and [792a22l A bookkeeping step names the version that was judged](./792a_product-spec.md#792a22l-a-bookkeeping-step-names-the-version-that-was-judged) states that.
  - The skill does not say what to do when the content amend of a near outcome fails after the fold has deleted the offer;
    the offer is gone and the claim is not folded in (#784, a known gap and not in the spec).
- **Rationale:** [792d3 Skills and guidance are product behavior](#792d3-skills-and-guidance-are-product-behavior), [792d5 agent-instruction-bundle qualifies as a component](#792d5-agent-instruction-bundle-qualifies-as-a-component), [792d30 Components are migrated once and extended later](#792d30-components-are-migrated-once-and-extended-later), [792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated), [792d52 Test IDs: product scenario for end-to-end checks, technical scenario otherwise](#792d52-test-ids-product-scenario-for-end-to-end-checks-technical-scenario-otherwise), [792d114 route scope, the SKILL.md repoint, and where the sources differ from the skill](#792d114-route-scope-the-skillmd-repoint-and-where-the-sources-differ-from-the-skill), [792d119 pending-offer-bookkeeping scope, the requirements taken out of vault-offer-curation, and the tests it takes](#792d119-pending-offer-bookkeeping-scope-the-requirements-taken-out-of-vault-offer-curation-and-the-tests-it-takes) and [792d121 parent-sync scope, what stays whole, the components extended, and where the sources differ from the code](#792d121-parent-sync-scope-what-stays-whole-the-components-extended-and-where-the-sources-differ-from-the-code).
- **Alternatives:** one requirement per outcome of the legacy spec's single large judgment requirement, with the scenarios
  attached to it. Rejected because the guide asks for separate requirements for materially different behaviors, and the five
  outcomes differ in their actions.

## 792d125 memory-effectiveness-evaluation scope, 792e1's default, and where ADR-0028's decisions went

- **Choice:**
  - [792e1 Is eval and audit tooling a product capability?](./792e_open-questions.md#792e1-is-eval-and-audit-tooling-a-product-capability)'s provisional default is applied: eval and audit tooling is the product capability [792a23 memory-effectiveness-evaluation](./792a_product-spec.md#792a23-added-memory-effectiveness-evaluation), with the components
    eval-trial-harness ([792b38 eval-trial-harness](./792b_technical-spec.md#792b38-added-eval-trial-harness)) and memory-loop-auditor ([792b39 memory-loop-auditor](./792b_technical-spec.md#792b39-added-memory-loop-auditor)). Its expected user is the maintainer, since the tools are dev
    tooling in this repository and not part of the shipped command line. If Joe answers option 2 or 3, [792a23 memory-effectiveness-evaluation](./792a_product-spec.md#792a23-added-memory-effectiveness-evaluation) moves under the
    technical specs or to `dev/eval` documentation and its rationale paragraphs stay as they are.
  - [792a23 memory-effectiveness-evaluation](./792a_product-spec.md#792a23-added-memory-effectiveness-evaluation) states seventeen requirements. The first ten come from the four eval and probe specs (warm configuration, the leak
    guard, the fixtures' scoring and the surfacing probe), and the last seven from `memory-loop-audit`: moments, the record, the
    vault as of the moment, the safe query, injected memories, credit records and the additive ledger. The `memory-loop-audit`
    requirement for the #739 Phase 2 go/no-go decision (a joint review of the real results, with no pre-set formula, Joe
    2026-09-01) is not ported: it was a one-time procedure for a change that is done (the ROADMAP records the decision and the
    audit report is §9 of `dev/eval/audit/REPORT-2026-09-02.md`), and a product spec states what the tools do now.
  - The four eval specs stay whole until [792c7v2 eval-trial-harness tech spec](./792c_implementation-plan.md#792c7v2-eval-trial-harness-tech-spec) and `memory-loop-audit` until [792c7v3 memory-loop-auditor tech spec](./792c_implementation-plan.md#792c7v3-memory-loop-auditor-tech-spec), because each requirement states both a
    promise and a contract ([792d117 vault-serving scope, what stays for the technical node, and where the sources differ from the server](#792d117-vault-serving-scope-what-stays-for-the-technical-node-and-where-the-sources-differ-from-the-server)). Their comments in code (`isolation.py`, `cumulative/test_matrix.py`) are repointed when
    they are deleted.
  - ADR-0028 D4 to D7 are in `docs/ROADMAP.md` as a block under the memory north-star track, whose D4 row already cites them (#718,
    #747, #748). D4, D5 and D6 are planned work (experience records, the `engram audit` subcommand, vault commit discipline) and
    D7 is done (the build-value harness is retired and the ledger has its coverage map). D1 to D3 and D8, with the context, the
    consequences and the risks, stay in the ADR for [792c7v3 memory-loop-auditor tech spec](./792c_implementation-plan.md#792c7v3-memory-loop-auditor-tech-spec).
  - This node renames no test.
- **Divergences noted ([792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated)):**
  - The warm-configuration requirement says any function that builds a warm configuration verifies each skill's source. The
    Go harness behind `targ eval` (`NewOSConfigBuilder`) installs skills with `cp -R` from the operator's `~/.claude/skills/<skill>`
    and not from the repository, and checks no `SKILL.md`: a missing source fails the copy, but an empty source directory is
    installed silently. The three Python builders share `isolation.install_skill`.
  - When the vault gained notes, the leak guard reads only the last ten note names in alphabetical order, and it does nothing
    when the count did not grow; a leak that sits among earlier names, or that replaces one note by another at the same count, is
    not detected, limits that [792a23f The leak guard's limits are stated](./792a_product-spec.md#792a23f-the-leak-guards-limits-are-stated)'s scenarios do not list.
- **Rationale:** [792d3 Skills and guidance are product behavior](#792d3-skills-and-guidance-are-product-behavior), [792d6 Removed features are not documented; enduring absences become MUST NOT](#792d6-removed-features-are-not-documented-enduring-absences-become-must-not), [792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated), [792d117 vault-serving scope, what stays for the technical node, and where the sources differ from the server](#792d117-vault-serving-scope-what-stays-for-the-technical-node-and-where-the-sources-differ-from-the-server) and [792d124 curate scope, the scenarios only the skill text verifies, and where the sources differ from the skill and the code](#792d124-curate-scope-the-scenarios-only-the-skill-text-verifies-and-where-the-sources-differ-from-the-skill-and-the-code).
- **Alternatives:** keep the four specs' technical requirements out of the product spec and write only components. This is [792e1 Is eval and audit tooling a product capability?](./792e_open-questions.md#792e1-is-eval-and-audit-tooling-a-product-capability)'s
  option 2, left for Joe.

## 792d126 eval-trial-harness scope, the Go harness left unspecified, and where the sources differ from the code

- **Choice:**
  - [792b38 eval-trial-harness](./792b_technical-spec.md#792b38-added-eval-trial-harness) specifies eleven requirements ([792b38c1 Installing a skill into a trial configuration](./792b_technical-spec.md#792b38c1-installing-a-skill-into-a-trial-configuration) to [792b38c11 Every spawn site builds its environment through the primitive](./792b_technical-spec.md#792b38c11-every-spawn-site-builds-its-environment-through-the-primitive)) and the trial-run sequence. Its first ten carry the technical
    sides of the four eval and probe specs; the eleventh ([792b38c11 Every spawn site builds its environment through the primitive](./792b_technical-spec.md#792b38c11-every-spawn-site-builds-its-environment-through-the-primitive), the spawn sites' use of the environment primitive) comes from
    the tier-A and tier-B site tests, which guard the property the leak and isolation specs presuppose, and supports
    [792a23d1b A harness captures no trial evidence itself and still gets a correct guard](./792a_product-spec.md#792a23d1-uat-scenarios). The isolated environment ([792b38c3 The isolated environment of a trial](./792b_technical-spec.md#792b38c3-the-isolated-environment-of-a-trial)) and the fingerprint ([792b38c4 The real-vault fingerprint](./792b_technical-spec.md#792b38c4-the-real-vault-fingerprint)) are not in any legacy spec; they are the
    primitives that the leak-detection spec refers to as "the standard isolation primitive", and their tests are many.
  - The component is a repository component of dev tooling whose realization spans three directories of Python and the fixtures.
    The probe of [792a23j A surfacing probe reads the rank of the item list](./792a_product-spec.md#792a23j-a-surfacing-probe-reads-the-rank-of-the-item-list) is in it because the catalog lists the retrieval-probe spec for it; its batched runbook probe
    ([792b38c10a3 A batched runbook probe reports success, miss and failure](./792b_technical-spec.md#792b38c10a-scenarios), [792b38c10a4 Escalation is per phrase and the corpus loads](./792b_technical-spec.md#792b38c10a-scenarios)) is described from its tests and is in no legacy spec.
  - The four specs are deleted. The comments that cited two of them (`isolation.py`, `cumulative/test_matrix.py`) cite [792a23 memory-effectiveness-evaluation](./792a_product-spec.md#792a23-added-memory-effectiveness-evaluation) and
    [792b38 eval-trial-harness](./792b_technical-spec.md#792b38-added-eval-trial-harness), and the dated records under `dev/eval/audit`, the ledger and `dev/eval/raw` that name the specs are left as they are
    ([792d53 Test-rename reference updates skip historical records](#792d53-test-rename-reference-updates-skip-historical-records)).
  - The Go harness behind `targ eval` (`dev/eval/*.go`: arms, scenarios, scoring, aggregation) is not specified, because no
    legacy source states its behavior and [792a23 memory-effectiveness-evaluation](./792a_product-spec.md#792a23-added-memory-effectiveness-evaluation) states only what the tools promise. Its tests are listed as no-scenario
    tests ([792e8 Should the Go eval harness behind targ eval be specified?](./792e_open-questions.md#792e8-should-the-go-eval-harness-behind-targ-eval-be-specified)).
  - Renames: 77 Python tests, as `test_<ID>_<previous_name>` in their own files. A test that is parametrized keeps one name.
    The tests that an earlier node already named ([792c7l2 agent-instruction-bundle tech spec](./792c_implementation-plan.md#792c7l2-agent-instruction-bundle-tech-spec)) keep their names. `pytest` passes for every renamed file (the changed files are
    not run by `targ`, which runs only the Go tests of `dev/eval` under `targ test-dev`).
- **Divergences noted ([792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated)):**
  - [792a23a A warm trial carries the skills it claims](./792a_product-spec.md#792a23a-a-warm-trial-carries-the-skills-it-claims) and [792a23b A missing or empty skill source stops the build](./792a_product-spec.md#792a23b-a-missing-or-empty-skill-source-stops-the-build) say that any function that builds a warm configuration verifies each skill's source. The Go
    harness does not ([792d125 memory-effectiveness-evaluation scope, 792e1's default, and where ADR-0028's decisions went](#792d125-memory-effectiveness-evaluation-scope-792e1s-default-and-where-adr-0028s-decisions-went)), and [792b38c2 The warm-configuration builders](./792b_technical-spec.md#792b38c2-the-warm-configuration-builders) names only the three Python builders.
  - The probe harness's configuration builder installs the skills it is given and no skills for an empty list, which the legacy
    spec's two-skill wording does not say ([792b38c2 The warm-configuration builders](./792b_technical-spec.md#792b38c2-the-warm-configuration-builders)).
  - [792b38c8 A question stop](./792b_technical-spec.md#792b38c8-a-question-stop) states the order-awareness of `question_stop` as the end-state check short-circuiting to false; the legacy spec
    says the ambiguity must precede completion, which the code decides only through that end-state result.
- **Rationale:** [792d3 Skills and guidance are product behavior](#792d3-skills-and-guidance-are-product-behavior), [792d10 Test renaming](#792d10-test-renaming), [792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated), [792d52 Test IDs: product scenario for end-to-end checks, technical scenario otherwise](#792d52-test-ids-product-scenario-for-end-to-end-checks-technical-scenario-otherwise), [792d53 Test-rename reference updates skip historical records](#792d53-test-rename-reference-updates-skip-historical-records) and [792d125 memory-effectiveness-evaluation scope, 792e1's default, and where ADR-0028's decisions went](#792d125-memory-effectiveness-evaluation-scope-792e1s-default-and-where-adr-0028s-decisions-went).
- **Alternatives:** write the Go harness's arms, scenarios and scoring as requirements now. Rejected for lack of a source and of
  a product statement to support them.

## 792d127 memory-loop-auditor scope, ADR-0028 deleted, and where the sources differ from the code

- **Choice:**
  - [792b39 memory-loop-auditor](./792b_technical-spec.md#792b39-added-memory-loop-auditor) specifies eight requirements ([792b39c1 Windows](./792b_technical-spec.md#792b39c1-windows) to [792b39c8 Outcome records](./792b_technical-spec.md#792b39c8-outcome-records)) and the audit sequence. The memory-loop-audit spec supplied the
    existence check, the query invocation, injected memories and credit records; ADR-0028 D1 to D3 supplied the orientation's rationale
    (the standing instrument, the loss-stage questions, judged-only credit); the rest (windows, detection, the record, the fork context
    and the outcome-record rules) is described from `audit_moments.py`, `generate_outcome_records.py` and their tests.
  - The `memory-loop-audit` spec and ADR-0028 are deleted, and `openspec/specs/` is empty. The ADR's D4 to D7 went to the ROADMAP
    ([792d125 memory-effectiveness-evaluation scope, 792e1's default, and where ADR-0028's decisions went](#792d125-memory-effectiveness-evaluation-scope-792e1s-default-and-where-adr-0028s-decisions-went)), D8 and the hard gate are in the orientation's rationale, and its context, consequences and risks are in the
    rationale of [792b39a Orientation](./792b_technical-spec.md#792b39a-orientation) and [792a23 memory-effectiveness-evaluation](./792a_product-spec.md#792a23-added-memory-effectiveness-evaluation). The extractor, the runner, the scorecard, the parent-capture script and the re-judging and
    merge scripts of `dev/eval/audit` are not specified: they produce and report the corpus the auditor reads (a one-time measurement
    that is complete), and the spec states only the instrument. Their tests are no-scenario tests.
  - Renames: 86 Python tests, as `test_<ID>_<previous_name>`, in `test_audit_moments.py` (85) and `test_run_window_sample.py` (1). All
    90 tests of those two files pass.
- **Divergences noted ([792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated)):**
  - The legacy spec says the audit measures the orchestrator-side gap with a structural check validated by a sampled model judgment of
    paraphrased content. `judge_moment` leaves `memories_orchestrator_had_but_left_out` null for every dispatch moment (deferred to the
    full-corpus runner, which cannot see the parent either), and no paraphrase check exists in `dev/eval/audit`; the gap in the
    ledger was computed outside the record ([792b39c7 A dispatch prompt's memories](./792b_technical-spec.md#792b39c7-a-dispatch-prompts-memories), [792a23o1b A dispatch moment measures the handoff on the orchestrator's side](./792a_product-spec.md#792a23o1-uat-scenarios), [792a23o1c Paraphrased content is told from missing content](./792a_product-spec.md#792a23o1-uat-scenarios)).
  - The spec says the existence check judges against the vault "as of" T by a checkout at the last commit at or before T; the code
    does so only for a commit that has the 2026-06-12 flat-vault migration commit as an ancestor, and otherwise uses the ESTIMATE
    branch ([792b39c4 Existence as of the moment's time](./792b_technical-spec.md#792b39c4-existence-as-of-the-moments-time)).
  - The legacy spec's go/no-go requirement (a joint review of the real results, with no pre-set formula) is not ported ([792d125 memory-effectiveness-evaluation scope, 792e1's default, and where ADR-0028's decisions went](#792d125-memory-effectiveness-evaluation-scope-792e1s-default-and-where-adr-0028s-decisions-went)).
- **Rationale:** [792d3 Skills and guidance are product behavior](#792d3-skills-and-guidance-are-product-behavior), [792d10 Test renaming](#792d10-test-renaming), [792d33 Behavior is described, not validated](#792d33-behavior-is-described-not-validated), [792d125 memory-effectiveness-evaluation scope, 792e1's default, and where ADR-0028's decisions went](#792d125-memory-effectiveness-evaluation-scope-792e1s-default-and-where-adr-0028s-decisions-went) and [792d126 eval-trial-harness scope, the Go harness left unspecified, and where the sources differ from the code](#792d126-eval-trial-harness-scope-the-go-harness-left-unspecified-and-where-the-sources-differ-from-the-code).
- **Alternatives:** specify the runner and the scorecard too. Rejected: they are the measurement of one finished experiment, and no
  legacy source states a contract for them.

## 792d128 Residual legacy sweep: what was ported where, and what was deleted

- **Choice:** node [792c8a Residual legacy sweep](./792c_implementation-plan.md#792c8a-residual-legacy-sweep) ported the following and deleted the rest of `docs/deprecated/` that is not [792c8b In-flight design relocation](./792c_implementation-plan.md#792c8b-in-flight-design-relocation)'s.
  - `FEATURES.md`: "Validated goals" is a new `## Validated goals` section of `docs/ROADMAP.md`, before Provenance. The pointer paragraph,
    `docs/deprecated/README.md` and the index tables are gone, because the file list in `docs/AGENTS.md` is the docs index.
  - `agents/{domain,issue-tracker,triage-labels}.md`: ported inline to the three "Agent skills" subsections of the root `AGENTS.md`.
    "Domain docs" now points at `docs/AGENTS.md`, the 792 behavior record and the glossary, and tells agents to surface a conflict with
    a requirement or its rationale. The wayfinding operations were kept, condensed.
  - ADR-0027 and the six memory-taxonomy research briefings: a new "Decisions carried from ADR-0027" block in the ROADMAP north-star
    track, beside the existing ADR-0028 block. It carries the taxonomy-as-checklist ruling, the gap map (firing, apply, scope probe, strength
    and update, interference, composition, lexical channel, multi-agent routing), the deferred items with their revisit conditions, and the risks.
    The `runbook` glossary entry gained the "declarative memory about a procedure" framing. The field survey citations, the human-memory
    literature and the engram-to-human mapping table are background and were deleted.
  - ADR-0020 is the expanded "DI everywhere" bullet of the root `AGENTS.md` Design Principles (the two-sided boundary, `cli.NewDeps`,
    the `Primitives` rule). That bullet also carries the `cmd/engram` primitives convention that c3's K13 held.
  - ADR-0004, 0005, 0008, 0009, 0012 and 0026: the four dead ADRs are one new Provenance row ("Retired memory-model machinery"), since the
    ROADMAP covered none of them. ADR-0012's round-2/3 gate was already in the GATED band, so its consequences were deleted. ADR-0026 was already a
    pointer. ADR-0016 (hand-authored mermaid) is superseded by the framework and deleted. The preamble and status legend are deleted.
  - The 2026-07-01 recall subprocess design had no Provenance row of its own; the payload-prune row names it, and a new Provenance row records the
    rejection (Joe, 2026-07-08) and the revisit condition.
  - C4 views: the L1, L2 and L3 diagrams became one system-context diagram and edge table, [792b1a System context](./792b_technical-spec.md#792b1a-system-context), in the technical spec. Every other catalog
    row already pointed at its [792b Documentation Framework Adoption - Technical Spec](./792b_technical-spec.md#documentation-framework-adoption---technical-spec) section. K11 debuglog stays excluded ([792d27 Excluded component candidates](#792d27-excluded-component-candidates)), K13 is in `AGENTS.md`, and "Dead/test-only surface" was
    stale (it described the removed graph-expansion path) and is deleted.
  - `memory-invariants.md`: the header, legend, retired items, B, D and RT-1 are deleted. M6 (learn idempotency) was never built and nothing
    in the ROADMAP or the issues asks for it, so it is deleted with the file rather than parked; M7 (marker monotonicity) is retired with the markers.
  - `docs/GLOSSARY.md`: the OpenSpec entry is dropped (retired, [792d39 GLOSSARY](#792d39-glossary)), the ADR-0012 reference points at [792a3q Question-and-answer pairs](./792a_product-spec.md#792a3q-question-and-answer-pairs), and the stale
    `fact|feedback|qa` wording of [792a13 write-memory](./792a_product-spec.md#792a13-added-write-memory) and `handoff contract` now includes `runbook`, matching `write-memory/SKILL.md`.
  - Root `AGENTS.md`: the `openspec/` line is gone, `docs/` is described per `docs/AGENTS.md`, Key Files name `docs/AGENTS.md`, the glossary,
    the roadmap and the behavior record, and the Design Principles authority line names the [792b Documentation Framework Adoption - Technical Spec](./792b_technical-spec.md#documentation-framework-adoption---technical-spec) sections and their rationale.
- **Rationale:** [792d31 Each legacy item has one owning node](#792d31-each-legacy-item-has-one-owning-node) (a legacy file is deleted once empty) and the standing rule that git is the fallback. The ROADMAP is the one
  home for decisions that still steer work, and the specs are the home for behavior and its rationale, so nothing was kept as an archive.
- **Alternatives:** keep a `docs/architecture/` overview. Rejected: it would be a second structure view next to the technical spec. Park M6 in
  the ROADMAP. Rejected: no issue or ruling asks for it.

## 792d129 OpenSpec archive mined, OpenSpec removed (792e3 applied), references repointed

- **Choice:**
  - **Mining.** The 42 archive entries were compared against the `Rationale:` text already in [792a Documentation Framework Adoption - Product Spec](./792a_product-spec.md#documentation-framework-adoption---product-spec), [792b Documentation Framework Adoption - Technical Spec](./792b_technical-spec.md#documentation-framework-adoption---technical-spec) and the ROADMAP. Most were covered
    by the capability nodes. The material decisions that were not are now rationale under the requirement they justify: [792a3d Runbook red flags and triggers](./792a_product-spec.md#792a3d-runbook-red-flags-and-triggers) (red flags are a
    field), [792a7h Candidate notes](./792a_product-spec.md#792a7h-candidate-notes) (a null recency result), [792a7x Reading one note](./792a_product-spec.md#792a7x-reading-one-note) (`show` never truncates), [792a7za A trigger hit comes first](./792a_product-spec.md#792a7za-a-trigger-hit-comes-first) (hits live in `items[]`), [792a10b Re-parenting only on request](./792a_product-spec.md#792a10b-re-parenting-only-on-request) and [792a10c Candidates to judge](./792a_product-spec.md#792a10c-candidates-to-judge) (re-parenting is opt-in and
    stops before apply), [792a11c A provable downgrade is refused](./792a_product-spec.md#792a11c-a-provable-downgrade-is-refused), [792a11e The rest of the update runs in the new binary](./792a_product-spec.md#792a11e-the-rest-of-the-update-runs-in-the-new-binary) and [792a11g The harnesses that are updated](./792a_product-spec.md#792a11g-the-harnesses-that-are-updated) (downgrade alternatives, the hand-off child, the harness addendum), [792a12b One runbook note per skill, identified by its key](./792a_product-spec.md#792a12b-one-runbook-note-per-skill-identified-by-its-key) to [792a12s Declines are keyed by skill key under schema version 2](./792a_product-spec.md#792a12s-declines-are-keyed-by-skill-key-under-schema-version-2)
    (one note per skill, pending accepts, declines by hash, conflicts, removals only for read sources, grouping, schema v2), [792a14h Chunk content is read before a cluster is judged](./792a_product-spec.md#792a14h-chunk-content-is-read-before-a-cluster-is-judged) (a fix in the
    skill's own text), [792a16 lessons-contract](./792a_product-spec.md#792a16-added-lessons-contract) (why subagents do not run learn), [792a17 route](./792a_product-spec.md#792a17-added-route) and [792a17j Cost and duration are real or stated as unavailable](./792a_product-spec.md#792a17j-cost-and-duration-are-real-or-stated-as-unavailable) (retiring route needs the shim activated; a price table and no
    cross-harness interface), [792a18a The skill fires on a handed-over multi-step ask](./792a_product-spec.md#792a18a-the-skill-fires-on-a-handed-over-multi-step-ask) (a positive-only situation), [792a19l Each returned kind is treated for what it is](./792a_product-spec.md#792a19l-each-returned-kind-is-treated-for-what-it-is) and [792a19m A matched runbook is announced and its steps are the plan](./792a_product-spec.md#792a19m-a-matched-runbook-is-announced-and-its-steps-are-the-plan) (the frame lives in the shim, the same body as a fact gets
    no frame), [792a20a The served command set is fixed](./792a_product-spec.md#792a20a-the-served-command-set-is-fixed) and [792a20e A served learn is attributed to the identity the caller declares](./792a_product-spec.md#792a20e-a-served-learn-is-attributed-to-the-identity-the-caller-declares) (the route split, declared identity), [792a22 curate](./792a_product-spec.md#792a22-added-curate) and [792a22a Pending offers are left out of normal query results](./792a_product-spec.md#792a22a-pending-offers-are-left-out-of-normal-query-results) and [792a22o Pending offers are surfaced at three points](./792a_product-spec.md#792a22o-pending-offers-are-surfaced-at-three-points) (curation overrides, a pending offer is
    a normal note, the flag comes first), [792b20c1 Definition recognition](./792b_technical-spec.md#792b20c1-definition-recognition) (the definition discriminator) and [792b27c9 Apply flow](./792b_technical-spec.md#792b27c9-apply-flow) (the apply order and its failure). The
    mixed-scope entry `2026-08-08-batch-small-issues-714-672-683-652` was mined only for [792a17j Cost and duration are real or stated as unavailable](./792a_product-spec.md#792a17j-cost-and-duration-are-real-or-stated-as-unavailable) and [792a7h Candidate notes](./792a_product-spec.md#792a7h-candidate-notes); its #714 and #683 items are
    implementation detail that no requirement states.
  - **Deletions.** `openspec/` is deleted entirely: `openspec/changes/archive/` (42 entries), `openspec/config.yaml`, and no `openspec/specs/`
    existed. All were tracked.
  - **Tooling ([792e3 Should the OpenSpec tooling be removed?](./792e_open-questions.md#792e3-should-the-openspec-tooling-be-removed), option 1 applied).** `.claude/skills/openspec-{apply-change,archive-change,explore,propose,sync-specs,update-change}/` and
    `.claude/commands/opsx/` (six commands) are deleted. All were tracked in this repository. Test fixtures that use `openspec-propose` or
    `opsx:apply` as skill names are data about foreign skills and stay.
  - **Repoints (11 edits outside the ROADMAP).** Root `README.md` (the archived-changes sentence, ADR-0013 and ADR-0002/0003 citations, the c2 pointer, the
    design-principles and documentation sections, six edits in all; the image link is repointed in the next node, [792c8d Deprecated-directory removal and final stale-reference sweep](./792c_implementation-plan.md#792c8d-deprecated-directory-removal-and-final-stale-reference-sweep)),
    `internal/cli/update_test.go` (a comment now cites [792b28c12 Report](./792b_technical-spec.md#792b28c12-report)), `internal/cli/skillkeys_qualified_test.go` and
    `internal/cli/skilloffers_namespace_test.go` (comments cite [792b30 skill-source-discovery](./792b_technical-spec.md#792b30-added-skill-source-discovery) and [792b31 skill-registration-ledger](./792b_technical-spec.md#792b31-added-skill-registration-ledger)), `.review/README.md` (the design document it cited,
    `add-annotation-feedback`, had never been in the archive; the sentence now says the rationale is in git history), and
    `agent-instructions/guidance/shim.md` line 1.
  - **`shim.md` line 1 is not a behavioral edit.** It is an HTML comment that named "openspec runbook-shim-follow-frame, D0" and now names [792a19 ambient-guidance](./792a_product-spec.md#792a19-added-ambient-guidance).
    No instruction the agent reads changes, so the writing-skills red, green and pressure cycle was not run (as in [792d116 ambient-guidance scope, the guidance edits, and where the sources differ from the documents](#792d116-ambient-guidance-scope-the-guidance-edits-and-where-the-sources-differ-from-the-documents)). The file is
    deployed by copy, so the comment changes for every user on their next `engram update --with-guidance`.
  - **ROADMAP.** 11 archive and change-directory citations became "the deleted archive entry `<name>`" or "the deleted `<name>` change"; the
    header line and the stale `openspec/specs/` mentions were rewritten.
  - **Vault note 913** ("designs go through openspec") is superseded; the orchestrator amends it, since vault writes are outside the repository.
- **Rationale:** [792d53 Test-rename reference updates skip historical records](#792d53-test-rename-reference-updates-skip-historical-records) (the archive is deleted by [792c8c OpenSpec archive and tooling removal](./792c_implementation-plan.md#792c8c-openspec-archive-and-tooling-removal)) and the lifecycle guide, which forbids keeping a completed-change archive. The
  mining kept only choices that explain a specified behavior, since task lists, test counts and process notes record work and not behavior.
- **Alternatives:** port each archive entry's design as an appendix. Rejected: it would recreate the archive inside the specs.

## 792d130 Final stale-reference sweep, the image move (792e5 applied), and what the directory removal waits on

- **Choice:**
  - **Image ([792e5 Where does vault-graph.png live?](./792e_open-questions.md#792e5-where-does-vault-graphpng-live), option 1 applied).** `docs/deprecated/images/vault-graph.png` moved to `docs/images/vault-graph.png`, so the root `README.md`
    link is correct again with no edit. `docs/AGENTS.md` now lists `docs/images/` in its Documentation Structure as an assets directory that holds no
    specification content.
  - **`docs/deprecated/` is not yet removed.** One file remains, `design/parallel-memory-agent-requirements.md`, which node [792c8b In-flight design relocation](./792c_implementation-plan.md#792c8b-in-flight-design-relocation) relocates once
    [792e2 Which issue key roots the parallel memory agent change?](./792e_open-questions.md#792e2-which-issue-key-roots-the-parallel-memory-agent-change) supplies an issue key (default: none). Everything else was deleted by [792c8a Residual legacy sweep](./792c_implementation-plan.md#792c8a-residual-legacy-sweep) or moved by this node. Node [792c8d Deprecated-directory removal and final stale-reference sweep](./792c_implementation-plan.md#792c8d-deprecated-directory-removal-and-final-stale-reference-sweep) is therefore In
    progress, and removing the directory waits only on that file.
  - **Sweep.** The sweep ran `git grep` over the whole repository, excluding `.claude/worktrees/`, for `docs/deprecated`, `docs/architecture`,
    `docs/agents`, `docs/design`, `docs/research`, `docs/superpowers`, `docs/images`, `docs/README`, `FEATURES.md`, `openspec/`, `ADR-00`, and the C4
    element IDs (`c1-system-context`, `c2-containers`, `c3-components`, `K`/`S`/`R` element IDs in prose). Live hits were repointed: ROADMAP (the
    ADR-0012 gate citation, five deleted-plan paths, the subprocess design path), eight comments in live tooling (`dev/eval/traps/*.py` and
    `dev/eval/vocab/trigger_replay.py`, per [792d37 Exemptions from the stale-reference sweep](#792d37-exemptions-from-the-stale-reference-sweep)), and the README lines recorded in [792d129 OpenSpec archive mined, OpenSpec removed (792e3 applied), references repointed](#792d129-openspec-archive-mined-openspec-removed-792e3-applied-references-repointed). No tracked file named a C4 element ID outside the
    exempt records.
  - **ADR citations are names, not links.** `ADR-00NN` in [792a Documentation Framework Adoption - Product Spec](./792a_product-spec.md#documentation-framework-adoption---product-spec) and [792b Documentation Framework Adoption - Technical Spec](./792b_technical-spec.md#documentation-framework-adoption---technical-spec) rationale, the ROADMAP and two Go comments (`internal/cli/offer.go`,
    `offer_test.go`) names a decision whose text was ported into the section that cites it. Rewriting each to a requirement ID would invent
    mappings the sources do not state, so they stay. The deleted record is recoverable with
    `git show a080d5a0^:docs/deprecated/architecture/adr.md`.
  - **Added to the exemption set ([792d37 Exemptions from the stale-reference sweep](#792d37-exemptions-from-the-stale-reference-sweep), [792d53 Test-rename reference updates skip historical records](#792d53-test-rename-reference-updates-skip-historical-records)).** Raw outputs and snapshots of past runs: every `raw/` and `results/` directory under
    `dev/eval/` (`dev/eval/atoms-build/raw/`, `dev/eval/learn-at-return/results/`), the skill-text snapshots the eval arms were run with
    (`dev/eval/atoms/sandbox-texts/`, `dev/eval/atoms-build/**/candidate/`, `dev/eval/guards/candidate/`), the fixture query
    `dev/eval/traps/runbook_corpus.json` (a user-message encoding that mentions an `openspec/` directory), and `.review/events.jsonl` (the append-only
    event log). Rewriting them would falsify what they recorded. The change's own [792a Documentation Framework Adoption - Product Spec](./792a_product-spec.md#documentation-framework-adoption---product-spec) problem statement and Gherkin example that name the legacy
    layout ([792a1 Problem Statement](./792a_product-spec.md#792a1-problem-statement), a path example in [792a23 memory-effectiveness-evaluation](./792a_product-spec.md#792a23-added-memory-effectiveness-evaluation)), and [792e Documentation Framework Adoption - Open Questions](./792e_open-questions.md#documentation-framework-adoption---open-questions), which quotes the old paths in its question contexts, are records of this change and also stay.
- **Rationale:** [792d37 Exemptions from the stale-reference sweep](#792d37-exemptions-from-the-stale-reference-sweep) and [792d53 Test-rename reference updates skip historical records](#792d53-test-rename-reference-updates-skip-historical-records) (a dated record is not rewritten; live comments are), [792d31 Each legacy item has one owning node](#792d31-each-legacy-item-has-one-owning-node) (a legacy file is deleted once empty).
- **Alternatives:** relocate `parallel-memory-agent-requirements.md` under #792 now. Rejected: [792d14 The in-flight design becomes its own change](#792d14-the-in-flight-design-becomes-its-own-change) and [792e2 Which issue key roots the parallel memory agent change?](./792e_open-questions.md#792e2-which-issue-key-roots-the-parallel-memory-agent-change). Delete it. Rejected: it is the only
  copy of an unbuilt feature's requirements, and [792e2 Which issue key roots the parallel memory agent change?](./792e_open-questions.md#792e2-which-issue-key-roots-the-parallel-memory-agent-change) is Joe's call.

## 792d131 The parallel memory agent design moves to change 793 (792e2 answered)

- **Choice:**
  - **Key.** Joe answered [792e2 Which issue key roots the parallel memory agent change?](./792e_open-questions.md#792e2-which-issue-key-roots-the-parallel-memory-agent-change) with option 1: the design is rooted at the new issue #793. It now lives in
    `docs/changes/793_parallel-memory-agent/` as `793a_product-spec.md`, `793b_decision-log.md` and `793c_open-questions.md`. The source file
    `docs/deprecated/design/parallel-memory-agent-requirements.md` was deleted by this node.
  - **Capability.** All 23 requirements are Added operations against one new canonical file, `docs/specs/product/automatic-memory.md`. The
    feature replaces the agent's own initiative to recall and learn, so it fits none of [792a14 recall](./792a_product-spec.md#792a14-added-recall), [792a15 learn](./792a_product-spec.md#792a15-added-learn) or [792a19 ambient-guidance](./792a_product-spec.md#792a19-added-ambient-guidance) ([793b1 One new capability, automatic-memory](../793_parallel-memory-agent/793b_decision-log.md#793b1-one-new-capability-automatic-memory)).
  - **IDs.** REC, CAP, CTL, IND and NFR requirements became [793a2a Recall at a human prompt](../793_parallel-memory-agent/793a_product-spec.md#793a2a-recall-at-a-human-prompt)–[793a2g Human-originated moments only](../793_parallel-memory-agent/793a_product-spec.md#793a2g-human-originated-moments-only), [793a3a Evaluate every completed turn](../793_parallel-memory-agent/793a_product-spec.md#793a3a-evaluate-every-completed-turn)–[793a3f Non-blocking capture](../793_parallel-memory-agent/793a_product-spec.md#793a3f-non-blocking-capture), [793a4a Operating modes](../793_parallel-memory-agent/793a_product-spec.md#793a4a-operating-modes)–[793a4d Sensitive content](../793_parallel-memory-agent/793a_product-spec.md#793a4d-sensitive-content) and [793a5a Independence from agent compliance](../793_parallel-memory-agent/793a_product-spec.md#793a5a-independence-from-agent-compliance)–[793a5f Effect on outcomes](../793_parallel-memory-agent/793a_product-spec.md#793a5f-effect-on-outcomes), each with
    a "(formerly X)" note ([793b2 ID mapping](../793_parallel-memory-agent/793b_decision-log.md#793b2-id-mapping)). The 31 draft scenarios keep their order; two scenarios were added to cover the prompt-time and tool-result bounds of
    the volume requirement, which the product spec guide requires ([793b3 Scenarios added for uncovered MUST branches](../793_parallel-memory-agent/793b_decision-log.md#793b3-scenarios-added-for-uncovered-must-branches)).
  - **Dropped sections.** The Terms became problem-statement context. Scope, the coverage limits and the evidence traceability table moved to
    `793b_decision-log.md` as context, because the guide allows a problem statement, requirements and UAT only ([793b4 Draft sections that are not requirements](../793_parallel-memory-agent/793b_decision-log.md#793b4-draft-sections-that-are-not-requirements), [793b5 Traceability from the draft (evidence to requirement)](../793_parallel-memory-agent/793b_decision-log.md#793b5-traceability-from-the-draft-evidence-to-requirement)).
  - **Open decisions.** The six Open Decisions became [793c1 What are the capture precision bar and capture-rate target?](../793_parallel-memory-agent/793c_open-questions.md#793c1-what-are-the-capture-precision-bar-and-capture-rate-target)–[793c6 Does capture also run inside subagent sessions?](../793_parallel-memory-agent/793c_open-questions.md#793c6-does-capture-also-run-inside-subagent-sessions). Values the draft marked [proposed] stay marked.
- **Rationale:** [792d14 The in-flight design becomes its own change](#792d14-the-in-flight-design-becomes-its-own-change) (the in-flight design becomes its own change), the product spec guide, and the lifecycle guide (change context is not
  canonical).
- **Alternatives:** split the requirements across the existing [792a14 recall](./792a_product-spec.md#792a14-added-recall), [792a15 learn](./792a_product-spec.md#792a15-added-learn) and [792a19 ambient-guidance](./792a_product-spec.md#792a19-added-ambient-guidance) capabilities. Rejected in [793b1 One new capability, automatic-memory](../793_parallel-memory-agent/793b_decision-log.md#793b1-one-new-capability-automatic-memory). Promote the
  requirements to `docs/specs/` now. Rejected: the feature is unbuilt and [792d44 Promotion is excluded from this plan](#792d44-promotion-is-excluded-from-this-plan) excludes promotion.

## 792d132 `docs/deprecated/` is gone; the last sweep found no live reference

- **Choice:**
  - **Removal.** The directory no longer exists. Its last file was deleted by node [792c8b In-flight design relocation](./792c_implementation-plan.md#792c8b-in-flight-design-relocation) ([792d131 The parallel memory agent design moves to change 793 (792e2 answered)](#792d131-the-parallel-memory-agent-design-moves-to-change-793-792e2-answered)), so git left no empty directory behind.
  - **Sweep.** `git grep` for `parallel-memory-agent-requirements`, `docs/deprecated` and `docs/design/parallel` (excluding `.claude/worktrees/`,
    `.review/events.jsonl` and the `dev/eval` `raw/` and `results/` directories) found no hit outside this change. Nothing needed repointing.
  - **Left as records.** The remaining hits are in this change: the [792c Documentation Framework Adoption - Implementation Plan](./792c_implementation-plan.md#documentation-framework-adoption---implementation-plan) node text that names its sources, [792d14 The in-flight design becomes its own change](#792d14-the-in-flight-design-becomes-its-own-change), [792d130 Final stale-reference sweep, the image move (792e5 applied), and what the directory removal waits on](#792d130-final-stale-reference-sweep-the-image-move-792e5-applied-and-what-the-directory-removal-waits-on) and [792d131 The parallel memory agent design moves to change 793 (792e2 answered)](#792d131-the-parallel-memory-agent-design-moves-to-change-793-792e2-answered), the context of
    [792e2 Which issue key roots the parallel memory agent change?](./792e_open-questions.md#792e2-which-issue-key-roots-the-parallel-memory-agent-change), and the `792f_discovery` findings (dated 2026-10-09, one cites the old design path and the branch name). They describe what was
    moved and why, so they stay ([792d37 Exemptions from the stale-reference sweep](#792d37-exemptions-from-the-stale-reference-sweep)).
- **Rationale:** [792d37 Exemptions from the stale-reference sweep](#792d37-exemptions-from-the-stale-reference-sweep) and [792d53 Test-rename reference updates skip historical records](#792d53-test-rename-reference-updates-skip-historical-records) (a dated record is not rewritten), [792d31 Each legacy item has one owning node](#792d31-each-legacy-item-has-one-owning-node) (a legacy file is deleted once empty).
- **Alternatives:** rewrite the 792f discovery rows to the 793 paths. Rejected: discovery records what was found before the move.
