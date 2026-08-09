# Evidence-based Route Rubric Specification

## Purpose

The route skill picks a subagent's model tier from recalled memory, not a fixed table: every unit starts at the cheapest tier, and only recalled evidence — or a failed review — raises it. Each dispatch is recorded as evidence per the route-dispatch-evidence spec. Those records become recallable memory, so the rubric improves over time without editing the skill. Why: docs/architecture/adr.md — ADR-0017 (extends ADR-0014). Validation: dev/eval/LEDGER.md#tier-routing-parity.

## Requirements

### Requirement: Routing SHALL default to the cheapest tier absent evidence
Every unit of work SHALL start at the cheapest / fastest available tier, regardless of how hard it appears (complex debugging, cross-cutting refactors, correctness reviews, greenfield design all start cheap).

#### Scenario: No prior evidence for a work-kind
- **WHEN** routing a unit of work with no recalled evidence on similar work-kinds
- **THEN** the orchestrator selects the cheapest tier, treating the felt difficulty as a prediction, not evidence

#### Scenario: Hard-looking work starts cheap
- **WHEN** a unit of work appears genuinely complex (cross-cutting refactor, new API design, race-condition debugging)
- **THEN** it still starts at the cheapest tier; complexity is a hunch, not recorded evidence

### Requirement: Recalled evidence SHALL be the sole tier escalator at routing time
Only recalled evidence of prior failure SHALL raise the starting tier at dispatch time, never prediction or forecasting of difficulty.

#### Scenario: Prior failures on same work-kind surface at routing
- **WHEN** recall surfaces evidence that this work-kind failed at the cheap tier before
- **THEN** the orchestrator escalates to the tier the evidence supports, not higher

### Requirement: Failed dispatch SHALL trigger spec-first retry before escalation
When a dispatch fails, the orchestrator SHALL first retry with an improved spec at the same tier before escalating to a higher tier.

#### Scenario: First failure after dispatch
- **WHEN** a subagent's dispatch fails review
- **THEN** the orchestrator rewrite the handoff spec (exact files, acceptance checks, do-NOT-touch bounds) and retries the same tier

#### Scenario: Second failure after spec retry
- **WHEN** the same unit fails a second time despite spec improvement
- **THEN** the orchestrator escalates one tier and retries

### Requirement: Memory-backed units SHALL discount one tier
A unit whose needed knowledge is recallable (a known convention, prior decision, crystallized diagnostic) SHALL drop one tier (floored at cheap) because the model applies recalled knowledge instead of deriving it.

#### Scenario: Memory-discount applies
- **WHEN** a unit requires knowledge that is already in the vault as a recallable note
- **THEN** the starting tier is one tier cheaper than the evidence alone would select (e.g., mid → cheap)

### Requirement: Cheap tier SHALL resolve to the cheapest available agent in the current environment
The cold-start roster (the concrete model names a tier resolves to) is an EXAMPLE, not a
prescription. When resolving the cheap tier at dispatch time, the orchestrator SHALL prefer
the cheapest agent genuinely available in the current environment — a free local model (e.g.
an LMStudio provider) counts as cheaper than any paid API model and SHALL be tried first for
cheap-tier work when one is available. Paid API models enter the roster only above free-local
options.

#### Scenario: Free local model available for cheap-tier work
- **WHEN** a free local model is registered and available in the current environment, and the orchestrator is routing cheap-tier work
- **THEN** the orchestrator dispatches to the free local model rather than a paid API model

#### Scenario: No free local model available
- **WHEN** no free local model is registered or available in the current environment
- **THEN** the orchestrator resolves cheap tier to the cheapest paid API model on the roster, as before

#### Scenario: Red flag on paid dispatch with a free option available
- **WHEN** the orchestrator dispatches cheap-tier work to a paid API model while a free local model was available and suitable
- **THEN** this is a documented red flag in the skill's red-flags table, naming "resolve the roster against the environment first" as the correction
