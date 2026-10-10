# runbook-lexical-triggers Specification

## Purpose

Runbook triggers are in `docs/changes/792_documentation-framework-adoption/792a_product-spec.md#792a7-added-memory-query`: the `triggers` field and its write-time flags are 792a3d, 792a3g and 792a4b, the `--text` input is 792a7y, a trigger hit is 792a7za and whole-word matching is 792a7zb. The authoring guidance for trigger cues is 792a13l and 792a13m in `docs/changes/792_documentation-framework-adoption/792a_product-spec.md#792a13-added-write-memory`. What remains here waits for the nodes named below.
## Requirements
### Requirement: `engram query` SHALL accept the user's raw message via `--text` (served-path residue for vault-serving, 792c7s1)
The served query path SHALL round-trip `text` byte-identically up to 2 KB; longer text SHALL be silently truncated to 2048 bytes on a UTF-8 rune boundary (capability `vault-serve-api`).

#### Scenario: Served round-trip
- **WHEN** a client builds a served query with `--text` (2 KB or less) containing quotes, newlines, `&`, and non-ASCII characters
- **THEN** the server receives the identical string and produces the same payload as the local path

### Requirement: The shim SHALL pass the user's verbatim message as `--text` (residue for ambient-guidance, 792c7r1)
The shim's first-action `engram query` block SHALL include `--text="<the user's message, word for word>"` alongside the two `--phrase` values, with the instruction that the text is pasted verbatim (first ~300 characters if long) and never rewritten, and that the flag uses the `--flag="value"` form (the quote touching the `=`, no space) rather than a space-separated `--text "value"` — a verbatim user message can begin with `-` with no control over its shape, and the space-separated form has no defense against that (`fix-eval-tooling-defects` D14/D17).

#### Scenario: Shim-only agent passes the raw message
- **WHEN** an agent with the updated shim receives the message "/please rename the tally CLI's --out flag to --output"
- **THEN** its first `engram query` carries `--text` whose value contains that message verbatim up to whitespace differences, plus the two phrases

