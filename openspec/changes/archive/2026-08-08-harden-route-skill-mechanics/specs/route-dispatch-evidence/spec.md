## MODIFIED Requirements

### Requirement: Every route dispatch SHALL be recorded as a tagged fact note
After each dispatch resolves, the orchestrator SHALL hand off a structured fact note to
write-memory with three categorical tags (work-kind/<k>, tier/<cheap|mid|deep>,
outcome/<pass|fail>) and provenance in the note's situation/subject/predicate/object fields.
The evidence note and the aggregate update are written through two different paths: the
evidence note goes through write-memory (parents judge, worker writes), while the aggregate
amend-or-create is composed and executed directly by the route-executing agent — structurally
forced, because write-memory has no amend form. This is a deliberate exception to the
write-site doctrine, not an oversight, and the skill SHALL state it explicitly.

#### Scenario: Recording a passing dispatch
- **WHEN** a dispatch completes with a passing review verdict
- **THEN** the orchestrator creates a fact note tagged work-kind/<k>, tier/<t>, outcome/pass with the dispatch details in the object field

#### Scenario: Recording a failing dispatch
- **WHEN** a dispatch fails review and escalates
- **THEN** the orchestrator creates a fact note tagged work-kind/<k>, tier/<t>, outcome/fail with the escalation tier and outcome details

#### Scenario: Write-path split is stated, not implicit
- **WHEN** an orchestrator reads the aggregate-update procedure
- **THEN** the skill text states that the evidence note goes through write-memory while the aggregate amend-or-create is composed directly by the route-executing agent, and names write-memory's lack of an amend form as the reason

### Requirement: Each work-kind SHALL have one aggregate fact note
For each work-kind, an aggregate fact note (route-evidence-<work-kind>) SHALL maintain running
tier tallies (e.g., "cheap 14/16, mid 2/2") plus an append-only wikilink list of every evidence
note that informs it. Before creating a new aggregate on a no-match lookup result, the
orchestrator SHALL run a deterministic uniqueness check (a vault basename listing/glob for
`route-evidence-<work-kind>*`, since aggregates are untagged and cannot be enumerated via
`engram count`) at the same trigger moments as the count audit, to distinguish "the aggregate
truly doesn't exist" from "the lookup missed an existing aggregate" (an embedding miss, or the
ADR-0019 drowning case). When the uniqueness check finds more than one aggregate for the same
work-kind, the orchestrator SHALL merge them: union the evidence wikilinks, recompute the tally
via the documented count commands, and record the incident as evidence for the ADR-0019
drowning-remedy decision.

#### Scenario: Aggregate creation
- **WHEN** the first dispatch of a work-kind is recorded
- **THEN** orchestrator creates route-evidence-<work-kind> with the tally from that one dispatch and a wikilink to its evidence note

#### Scenario: Aggregate update
- **WHEN** a subsequent dispatch of the same work-kind is recorded
- **THEN** orchestrator amends route-evidence-<work-kind>, recomputes the tier tally, and appends the new evidence-note wikilink

#### Scenario: Uniqueness check before minting a new aggregate
- **WHEN** the aggregate lookup for a work-kind returns no match
- **THEN** the orchestrator runs the documented uniqueness check (vault basename listing for `route-evidence-<work-kind>*`) before creating a new aggregate, to rule out a missed lookup rather than a genuinely absent aggregate

#### Scenario: Duplicate aggregates found and merged
- **WHEN** the uniqueness check finds more than one `route-evidence-<work-kind>` note
- **THEN** the orchestrator unions their evidence wikilinks, recomputes the tally via the documented count commands, merges them into one aggregate, and records the incident as ADR-0019 drowning-remedy evidence

### Requirement: Count SHALL audit aggregate drift using tags
engram count with --group-by tags filters SHALL recompute ground-truth tier tallies from
evidence-note tags, never from aggregate prose, so drift is detected and aggregates can be
repaired. The count-audit section SHALL warn that `--group-by tier` is a namespace-collision
footgun: it runs without error but groups the wrong attribute — the pre-existing L1/L2/L3
note-tier frontmatter field, not the `tier/<cheap|mid|deep>` tag family used by route evidence.

#### Scenario: Count audit for a work-kind
- **WHEN** auditing tier routing parity for a work-kind
- **THEN** orchestrator runs `engram count --group-by tags --filter tags=work-kind/<k> --filter tags=tier/<t>` to recompute numerators from evidence-note tags

#### Scenario: Aggregate repair
- **WHEN** count output disagrees with an aggregate's prose tally
- **THEN** orchestrator amends the aggregate's object field to match the recomputed count result

#### Scenario: `--group-by tier` footgun is documented
- **WHEN** an orchestrator reads the count-audit section
- **THEN** the skill text warns that `--group-by tier` groups the pre-existing note-tier frontmatter attribute (L1/L2/L3), not the `tier/` evidence tag family, and that `--group-by tags --filter tags=tier/<t>` is the correct form
