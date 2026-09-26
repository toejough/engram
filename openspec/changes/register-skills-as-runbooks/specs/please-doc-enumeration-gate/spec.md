## MODIFIED Requirements

### Requirement: Plan author SHALL run doc-surface enumeration grep for repeated invariants

When a planned change alters a repeated invariant (payload shape, sweep cadence, command set, count, naming convention echoed across docs/diagrams/skills), the plan author SHALL search the term, its synonyms, hyphenated forms, and the OLD text's echoes in labels and comments, then paste the per-file disposition list into the plan. This procedure SHALL be carried by the please skill (`agent-instructions/skills/please/SKILL.md`, with its optional vault runbook note mirroring the skill) rather than a separate runbook body.

#### Scenario: Planning a change to a repeated invariant
- **WHEN** a plan alters something repeated across multiple files (a field name, a command, a convention)
- **THEN** the plan author runs the grep over the repo and includes the per-file disposition list (file → keep / update / rewrite / N/A, with one-line reason) in the plan text

#### Scenario: Small or seemingly obvious surface
- **WHEN** a surface appears small or obvious enough to skip the grep
- **THEN** the author still runs the grep; cost scales with surface size, so small surfaces produce cheap greps, never exemptions

#### Scenario: Carrier is the runbook, not a skill
- **WHEN** an agent plans a change under the please skill
- **THEN** the please skill carries the doc-surface enumeration requirement in its Step 3, and the grep requirement applies unchanged; the please skill's optional vault runbook note (when registered) mirrors this same requirement
