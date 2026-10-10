# runbook-lexical-triggers Specification

## Purpose

Runbook triggers are in `docs/changes/792_documentation-framework-adoption/792a_product-spec.md#792a7-added-memory-query`: the `triggers` field and its write-time flags are 792a3d, 792a3g and 792a4b, the `--text` input is 792a7y, a trigger hit is 792a7za and whole-word matching is 792a7zb. The authoring guidance for trigger cues is 792a13l and 792a13m in `docs/changes/792_documentation-framework-adoption/792a_product-spec.md#792a13-added-write-memory`. What remains here waits for the node named below.
## Requirements
### Requirement: `engram query` SHALL accept the user's raw message via `--text` (served-path residue for vault-serving, 792c7s1)
The served query path SHALL round-trip `text` byte-identically up to 2 KB; longer text SHALL be silently truncated to 2048 bytes on a UTF-8 rune boundary (capability `vault-serve-api`).

#### Scenario: Served round-trip
- **WHEN** a client builds a served query with `--text` (2 KB or less) containing quotes, newlines, `&`, and non-ASCII characters
- **THEN** the server receives the identical string and produces the same payload as the local path
