## Purpose

Skills shipped in `agent-instructions/skills/` remain the single shipping and authoring source of truth for a procedure (deployed by `engram update`). A skill that authors runbook metadata in its own frontmatter is *registered*: engram mechanically derives an idempotent runbook note in the vault from it, so memory can surface the procedure by situation and trigger, the shim's follow-frame can drive it, and it can flow across vault-graph edges — without the skill ever leaving its distribution mechanism. Why: vault note 1054 (reversing 1031); the four 2026-09 delete-conversions broke distribution for every install but Joe's.

## ADDED Requirements

### Requirement: A skill SHALL register by authoring runbook fields in its frontmatter
A skill's `SKILL.md` (or a `runbooks/<slug>.md` file inside the skill directory) whose frontmatter carries `situation` and `done_when` SHALL be treated as a registered runbook. `triggers` and `red_flags` are optional, with the same semantics and limits as on a runbook note (whole-word trigger matching; `red_flags` rendered ≤ 1200 bytes). A skill file carrying only `name` and `description` SHALL NOT be registered and SHALL deploy exactly as before.

#### Scenario: Skill with runbook fields is registered
- **WHEN** `agent-instructions/skills/curate/SKILL.md` carries `situation`, `done_when`, `triggers`, and `red_flags` and registration runs
- **THEN** one runbook note derived from it exists in the vault

#### Scenario: Skill without runbook fields is not registered
- **WHEN** a skill's frontmatter has only `name` and `description` and registration runs
- **THEN** no vault note is derived from it and its deployment is unchanged

### Requirement: Registration SHALL derive an idempotent runbook note keyed by a stable identity
Each registered runbook SHALL be derived into a vault note carrying `registered_from: <skill-name>/<runbook-slug>` (the top runbook's slug is `SKILL`). Registration SHALL locate an existing note by that field: when found, it SHALL amend it, replacing every derived field (situation, done_when, red_flags, triggers, body) and preserving `created`; when not found, it SHALL create a new runbook note through the normal capture path (fresh Luhmann id, standard basename, embed on write). Running registration twice with no skill change SHALL produce no change.

#### Scenario: First registration creates
- **WHEN** no note with `registered_from: curate/SKILL` exists and registration runs
- **THEN** a new runbook note is created with that field, a fresh Luhmann id, and a `.vec.json` sidecar

#### Scenario: Re-registration updates in place
- **WHEN** the skill's `situation` text changes and registration runs
- **THEN** the existing note (same basename, same `created`) has the new situation and a rebuilt sidecar; no second note is created

#### Scenario: Idempotent when unchanged
- **WHEN** registration runs twice with no intervening skill edit
- **THEN** the second run writes nothing

### Requirement: Registration SHALL be a sync
A derived note whose `registered_from` no longer matches any registered runbook in the source SHALL be removed by registration. A derived note SHALL carry a body preamble stating it is derived from a skill and that local amends to derived fields are overwritten on the next registration; `engram amend` targeting such a note SHALL print a warning naming the source skill.

#### Scenario: Unregistered skill's note is removed
- **WHEN** a skill's runbook fields are removed (or the skill directory is deleted) and registration runs
- **THEN** the note previously derived from it is removed from the vault

#### Scenario: Local amend is overwritten on next registration
- **WHEN** `engram amend` changes a derived note's `situation` and registration runs afterward
- **THEN** the situation is restored to the skill's authored value

### Requirement: Multi-runbook skills SHALL use one file per runbook with slug cross-references
A skill directory MAY contain `runbooks/<slug>.md` files, each a registered runbook. Cross-references between a skill's runbooks SHALL be authored as `[[<skill-name>/<slug>]]` wikilinks and SHALL be rewritten to the derived notes' real basenames at registration. An unresolvable skill-local slug SHALL fail registration for that skill with an error naming the slug; wikilinks that are not skill-local slugs SHALL be left unchanged.

#### Scenario: Sub-runbook references resolve
- **WHEN** `please/SKILL.md` contains `[[please/adversarial-review-gates]]` and `please/runbooks/adversarial-review-gates.md` is registered
- **THEN** the derived top note's body contains `[[<the sub note's real basename>]]` and `engram show` on it lists the sub note as an outbound link

#### Scenario: Dangling slug fails loudly
- **WHEN** a skill file references `[[please/no-such-runbook]]`
- **THEN** registration reports an error naming the slug and derives nothing for that skill

### Requirement: Existing promoted notes SHALL be adopted, not re-created
When a restored skill's runbook metadata equals, field for field, an existing vault runbook note that predates registration, first registration SHALL adopt that note by stamping `registered_from` onto it rather than creating a duplicate.

#### Scenario: Adopting a previously promoted note
- **WHEN** `curate/SKILL.md`'s metadata matches note `1049.2026-09-21.curate-review-pending-offers` and registration runs for the first time
- **THEN** note 1049 gains `registered_from: curate/SKILL` and no new curate note is created

### Requirement: Registration SHALL run on `engram update` and be invocable standalone
`engram update` SHALL run registration after deploying skills (in the post-update vault step). A standalone `engram register-skills` command with `--dry-run` SHALL perform or preview the same derivation.

#### Scenario: Update registers
- **WHEN** `engram update` completes deploying skills
- **THEN** every registered skill has a current derived note and every stale derived note is removed

#### Scenario: Dry run previews
- **WHEN** `engram register-skills --dry-run` runs
- **THEN** it prints each create/update/remove it would perform and writes nothing
