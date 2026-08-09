## ADDED Requirements

### Requirement: Cold-start priors SHALL reflect evidence-backed escalations per work-kind
When recorded dispatch evidence for a work-kind clears the escalation thresholds (cheap-tier pass
rate below 70% AND mid-tier pass rate above 85%, both past the 5-dispatch evidence floor), the
route skill's Cold-start priors table SHALL carry a row for that work-kind starting at the
evidence-supported tier instead of the default cheapest tier, citing the evidence tally that
justified it.

#### Scenario: go-package-implementation clears the escalation threshold
- **WHEN** recorded dispatch evidence shows `work-kind/go-package-implementation` at 8/12 (67%)
  pass rate at the cheap tier and 10/10 (100%) pass rate at the mid tier
- **THEN** the Cold-start priors table carries a `work-kind/go-package-implementation` row with
  cold-start tier `mid`, citing the evidence tally

#### Scenario: A work-kind with high cheap-tier pass rate stays at the default
- **WHEN** recorded dispatch evidence shows a work-kind's cheap-tier pass rate at or above 90%
- **THEN** the Cold-start priors table does not add an escalated row for that work-kind — it
  remains covered by the default cheapest-tier posture
