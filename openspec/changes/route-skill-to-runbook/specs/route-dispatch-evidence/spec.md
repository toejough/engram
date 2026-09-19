## MODIFIED Requirements

### Requirement: Every route dispatch SHALL be recorded as a tagged fact note
After each dispatch resolves, the orchestrator SHALL hand off a structured fact note to write-memory with three categorical tags (work-kind/<k>, tier/<cheap|mid|deep>, outcome/<pass|fail>) and provenance in the note's situation/subject/predicate/object fields.
The evidence note and the aggregate update are written through two different paths: the
evidence note goes through write-memory (parents judge, worker writes), while the aggregate
amend-or-create is composed and executed directly by the route-executing agent — structurally
forced, because write-memory has no amend form. This is a deliberate exception to the
write-site doctrine, not an oversight, and route's runbook SHALL state it explicitly.

#### Scenario: Recording a passing dispatch
- **WHEN** a dispatch completes with a passing review verdict
- **THEN** the orchestrator creates a fact note tagged work-kind/<k>, tier/<t>, outcome/pass with the dispatch details in the object field

#### Scenario: Recording a failing dispatch
- **WHEN** a dispatch fails review and escalates
- **THEN** the orchestrator creates a fact note tagged work-kind/<k>, tier/<t>, outcome/fail with the escalation tier and outcome details

#### Scenario: Write-path split is stated, not implicit
- **WHEN** an orchestrator reads the aggregate-update procedure
- **THEN** route's runbook text states that the evidence note goes through write-memory while the aggregate amend-or-create is composed directly by the route-executing agent, and names write-memory's lack of an amend form as the reason
