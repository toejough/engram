# Please Step 3 Doc-surface Enumeration Grep + Gate A Verification Specification

## Purpose

The please skill's Step 3 (Plan) carries a non-waivable doc-surface enumeration grep that runs when a planned change alters a repeated invariant (payload shape, cadence, naming convention) echoed across docs, diagrams, or skills. Before any plan proceeds, Gate A's docs/diagrams-alignment reviewer independently verifies the grep disposition list and runs its own independent discovery pass. Why: issue #685 (no dedicated ADR). Validation: dev/eval/LEDGER.md#685-doc-enumeration-grep and new headless probe harness dev/eval/cumulative/please_step3_probe/.

## Requirements

### Requirement: Plan author SHALL run doc-surface enumeration grep for repeated invariants
When a planned change alters a repeated invariant (payload shape, sweep cadence, command set, count, naming convention echoed across docs/diagrams/skills), the plan author SHALL search the term, its synonyms, hyphenated forms, and the OLD text's echoes in labels and comments, then paste the per-file disposition list into the plan.

#### Scenario: Planning a change to a repeated invariant
- **WHEN** a plan alters something repeated across multiple files (a field name, a command, a convention)
- **THEN** the plan author runs the grep over the repo and includes the per-file disposition list (file → keep / update / rewrite / N/A, with one-line reason) in the plan text

#### Scenario: Small or seemingly obvious surface
- **WHEN** a surface appears small or obvious enough to skip the grep
- **THEN** the author still runs the grep; cost scales with surface size, so small surfaces produce cheap greps, never exemptions

### Requirement: Gate A docs/diagrams-alignment reviewer SHALL independently verify and discover
Gate A's docs/diagrams-alignment reviewer SHALL verify the plan author's enumeration-grep disposition list against the actual files AND still run its own independent discovery pass — the author's list is never the reviewer's source, and its presence never narrows the reviewer's scan.

#### Scenario: Reviewing enumeration grep during Gate A
- **WHEN** Gate A's docs/diagrams-alignment review begins
- **THEN** the reviewer verifies every listed disposition AND independently searches for any missed surfaces, treating the author's list as a checklist item ("did they find everything?"), not as the complete source

### Requirement: Disposition list shall name each file and reason
Each entry in the enumeration-grep disposition list SHALL name the file, the disposition (keep / update / rewrite / N/A), and a one-line reason.

#### Scenario: Disposition list format
- **WHEN** the plan author compiles the enumeration-grep results
- **THEN** each line names a file path, a disposition, and the reason (e.g., "docs/ROADMAP.md:3 → update: cites old FEATURES, retarget to openspec/specs/")

### Requirement: Grep is non-waivable regardless of plan size
The enumeration grep SHALL run for every plan that alters a repeated invariant, regardless of how large the surface looks or how small the change appears.

#### Scenario: Minimal change to a widely-referenced concept
- **WHEN** a plan touches a concept that appears in many files (a field name, a command, a procedure)
- **THEN** the author still runs the full grep; the cost/benefit does not exempt small changes
