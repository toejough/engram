# Delta: route-dispatch-evidence — explore sampling replaces tag nomination

## MODIFIED Requirements

### Requirement: Route SHALL read dispatch evidence through plain recall, never special queries
When determining the starting tier for a dispatch, the orchestrator SHALL invoke `/recall` with standard phrases; evidence aggregates (route-evidence-<work-kind>) surface as normal memory items via vocab-tagged explore sampling (capability `recall-centroid-sampling`), not through special `engram count` queries on the read path.

#### Scenario: Routing with recalled evidence
- **WHEN** routing a unit and recall surfaces a route-evidence-<work-kind> aggregate
- **THEN** the orchestrator reads the aggregate's object prose to extract tier tallies and uses it to inform the tier choice
