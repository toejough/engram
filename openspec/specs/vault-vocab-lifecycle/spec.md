# Vocabulary Lifecycle Specification

## Purpose

The controlled vocabulary's product behavior (definition and family notes, write-time tagging, bootstrap, propose, derivational refit, the refit trigger, `vocab stats`, the self-tag backfill and the old-format migration) is in `docs/changes/792_documentation-framework-adoption/792a_product-spec.md#792a6-added-controlled-vocabulary`. Its mechanism is in `docs/changes/792_documentation-framework-adoption/792b_technical-spec.md#792b20-added-vocabulary-engine`. What remains here waits for memory-query (792c7f1: the supersession ride-along).
## Requirements
### Requirement: Supersession ride-along (residue for memory-query, 792c7f1)
When a note carries a `supersedes:` frontmatter block listing older notes it replaces (via `type: updates|narrows|refutes`), the newer note SHALL be inserted directly after any delivered older note in the query results, regardless of whether the newer note itself matched the query. The inserted note MUST carry provenance `ride_along` so Gate-2 analysis can detect rank shifts from insertions.

#### Scenario: Superseder follows superseded in results
- **WHEN** query results include a delivered note with a recorded superseder in `SupersedesInverse`
- **THEN** `applySupersedesRideAlong()` inserts the superseding note immediately after the superseded note, marked with provenance `ride_along`

#### Scenario: Deduplication prevents double insertion
- **WHEN** a superseding note is already present in results (as a direct hit or prior ride-along)
- **THEN** it is not inserted again
