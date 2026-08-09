## ADDED Requirements

### Requirement: Route mini-report SHALL convert tokens to a dollar estimate when a rate is known
The route mini-report SHALL consult a maintained per-model price table (cheap/mid/deep tiers, split input/output/cache rates where the provider exposes them) to convert a dispatch's recorded `subagent_tokens` into a dollar estimate. When a rate is known for the dispatch's model, the mini-report SHALL show the dollar estimate; when no rate is known, it SHALL show the raw unit-labeled token count instead of fabricating a dollar figure.

#### Scenario: Dollar estimate shown when a rate is known
- **WHEN** a dispatch's mini-report is rendered and the price table has a rate for that dispatch's model
- **THEN** the report shows a dollar estimate computed from `subagent_tokens` and the table's rate

#### Scenario: Raw tokens shown when no rate is known
- **WHEN** a dispatch's mini-report is rendered and the price table has no rate for that dispatch's model
- **THEN** the report shows the unit-labeled raw token count, not a fabricated dollar figure

### Requirement: Cost/duration reporting SHALL degrade explicitly for non-Claude-Code harnesses
For harnesses other than Claude Code that do not expose a per-subagent usage block, the route mini-report SHALL use a documented harness-specific cost/duration source when one exists, and SHALL otherwise show `n/a` with a stated reason rather than a fabricated or silently-omitted value.

#### Scenario: Documented alternate source used
- **WHEN** a dispatch runs on a non-Claude-Code harness that has a documented cost/duration source
- **THEN** the mini-report uses that source to populate cost and duration

#### Scenario: No signal available
- **WHEN** a dispatch runs on a harness with no per-subagent usage block and no documented alternate source
- **THEN** the mini-report shows `n/a` with the reason the signal is unavailable, and never fabricates a cost or duration number
