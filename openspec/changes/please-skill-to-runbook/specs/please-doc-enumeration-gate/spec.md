## MODIFIED Requirements

### Requirement: Plan author SHALL run doc-surface enumeration grep for repeated invariants
When a planned change alters a repeated invariant (payload shape, sweep cadence, command set, count, naming convention echoed across docs/diagrams/skills), the plan author SHALL search the term, its synonyms, hyphenated forms, and the OLD text's echoes in labels and comments, then paste the per-file disposition list into the plan. This procedure SHALL be carried by the please runbook set's Step 3 (the doc-surface enumeration sub-runbook) rather than a `please` skill body.

#### Scenario: Planning a change to a repeated invariant
- **WHEN** a plan alters something repeated across multiple files (a field name, a command, a convention)
- **THEN** the plan author runs the grep over the repo and includes the per-file disposition list (file → keep / update / rewrite / N/A, with one-line reason) in the plan text

#### Scenario: Small or seemingly obvious surface
- **WHEN** a surface appears small or obvious enough to skip the grep
- **THEN** the author still runs the grep; cost scales with surface size, so small surfaces produce cheap greps, never exemptions

#### Scenario: Carrier is the runbook, not a skill
- **WHEN** an agent plans a change under the please runbook
- **THEN** the top runbook's Step 3 body wikilinks (`[[basename]]`) the doc-surface enumeration sub-runbook, and the grep requirement above applies unchanged

### Requirement: Gate A docs/diagrams-alignment reviewer SHALL independently verify and discover
Gate A's docs/diagrams-alignment reviewer SHALL verify the plan author's enumeration-grep disposition list against the actual files AND still run its own independent discovery pass — the author's list is never the reviewer's source, and its presence never narrows the reviewer's scan. The reviewer charge SHALL be carried by the please runbook set's adversarial-review-gates sub-runbook.

#### Scenario: Reviewing enumeration grep during Gate A
- **WHEN** Gate A's docs/diagrams-alignment review begins
- **THEN** the reviewer verifies every listed disposition AND independently searches for any missed surfaces, treating the author's list as a checklist item ("did they find everything?"), not as the complete source
