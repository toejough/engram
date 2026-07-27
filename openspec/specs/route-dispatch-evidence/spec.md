# Route Dispatch Evidence + Aggregates (tags-based) Specification

## Purpose

Every route dispatch is recorded as an ordinary recallable fact note tagged with three categorical tags (work-kind, tier, outcome in frontmatter tags:), and each work-kind keeps one aggregate fact note (route-evidence-<work-kind>) whose object text holds running tier tallies plus wikilinks to every evidence note. Route reads evidence by plain recall — aggregates surface as normal memories; engram count recomputes tallies from tags as the drift audit. Why: docs/architecture/adr.md — ADR-0019 (the 2026-07-10 decision on #669); issue #674. Validation: internal/cli/learn_test.go (TestLearnFact_Tags_WrittenToFrontmatter, TestLearnFact_InvalidTag_RejectedBeforeWrite, TestRenderFactFrontmatter_TagsRoundtripFidelity) and internal/cli/amend_test.go (TestRunAmend_PreservesTagsFrontmatter); scratch-vault drowning gauge PASS at 20 sibling evidence notes + count recompute parity.

## Requirements

### Requirement: Every route dispatch SHALL be recorded as a tagged fact note
After each dispatch resolves, the orchestrator SHALL hand off a structured fact note to write-memory with three categorical tags (work-kind/<k>, tier/<cheap|mid|deep>, outcome/<pass|fail>) and provenance in the note's situation/subject/predicate/object fields.

#### Scenario: Recording a passing dispatch
- **WHEN** a dispatch completes with a passing review verdict
- **THEN** the orchestrator creates a fact note tagged work-kind/<k>, tier/<t>, outcome/pass with the dispatch details in the object field

#### Scenario: Recording a failing dispatch
- **WHEN** a dispatch fails review and escalates
- **THEN** the orchestrator creates a fact note tagged work-kind/<k>, tier/<t>, outcome/fail with the escalation tier and outcome details

### Requirement: Tags SHALL be three closed families: work-kind, tier, outcome
Tags SHALL use bare families (family only, no value) for definition notes, and family/value pairs (kebab-case segments) for evidence notes; only three tag families are valid: work-kind (open set, kebab-case), tier (closed: cheap, mid, deep), outcome (closed: pass, fail).

#### Scenario: Valid tag format
- **WHEN** recording dispatch evidence
- **THEN** tags are formatted as work-kind/<k>, tier/<t>, outcome/<o> with only alphanumeric and hyphen in each segment

#### Scenario: Invalid tag rejected
- **WHEN** an invalid tag (wrong family, underscore, uppercase) is passed to engram learn
- **THEN** internal/cli/learn.go's validateTags function rejects it before write (errTagInvalid)

### Requirement: Each work-kind SHALL have one aggregate fact note
For each work-kind, an aggregate fact note (route-evidence-<work-kind>) SHALL maintain running tier tallies (e.g., "cheap 14/16, mid 2/2") plus an append-only wikilink list of every evidence note that informs it.

#### Scenario: Aggregate creation
- **WHEN** the first dispatch of a work-kind is recorded
- **THEN** orchestrator creates route-evidence-<work-kind> with the tally from that one dispatch and a wikilink to its evidence note

#### Scenario: Aggregate update
- **WHEN** a subsequent dispatch of the same work-kind is recorded
- **THEN** orchestrator amends route-evidence-<work-kind>, recomputes the tier tally, and appends the new evidence-note wikilink

### Requirement: Count SHALL audit aggregate drift using tags
engram count with --group-by tags filters SHALL recompute ground-truth tier tallies from evidence-note tags, never from aggregate prose, so drift is detected and aggregates can be repaired.

#### Scenario: Count audit for a work-kind
- **WHEN** auditing tier routing parity for a work-kind
- **THEN** orchestrator runs `engram count --group-by tags --filter tags=work-kind/<k> --filter tags=tier/<t>` to recompute numerators from evidence-note tags

#### Scenario: Aggregate repair
- **WHEN** count output disagrees with an aggregate's prose tally
- **THEN** orchestrator amends the aggregate's object field to match the recomputed count result

### Requirement: Route SHALL read dispatch evidence through plain recall, never special queries
When determining the starting tier for a dispatch, the orchestrator SHALL invoke `/recall` with standard phrases; evidence aggregates (route-evidence-<work-kind>) surface as normal memory items through tag nomination, not through special `engram count` queries on the read path.

#### Scenario: Routing with recalled evidence
- **WHEN** routing a unit and recall surfaces a route-evidence-<work-kind> aggregate
- **THEN** the orchestrator reads the aggregate's object prose to extract tier tallies and uses it to inform the tier choice

### Requirement: Aggregates SHALL use wikilinks to evidence notes, tags for categorization only
Aggregates SHALL link evidence notes via wikilinks in the object field (for auditability and traversal) and SHALL NOT carry their own work-kind/tier/outcome tags — aggregates are prose summaries, not evidence rows.

#### Scenario: Wikilink trail in aggregate
- **WHEN** an aggregate is created or amended
- **THEN** its object field contains wikilinks to every evidence note it summarizes, e.g., "cheap 14/16, mid 2/2 as of <date> — evidence: [[ev-note-1]], [[ev-note-2]], ..."
