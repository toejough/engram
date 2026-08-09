## ADDED Requirements

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
