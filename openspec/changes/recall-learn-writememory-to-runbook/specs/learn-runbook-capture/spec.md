## MODIFIED Requirements

### Requirement: `write-memory` SHALL compose and execute runbook-note writes

The write-memory runbook (a vault note of type `runbook`, reached by basename/wikilink from `learn`'s own instructions — not by `engram query`, since write-memory has no trigger of its own) SHALL compose the `engram learn runbook` command from fields handed off by `learn` (slug, situation, done_when, body/steps, source, target/position disposition, optional contributors, optional red_flags list), execute it, verify the result, and report the written note path — consistent with how it handles `fact` and `feedback` handoffs. Each handed-off red flag becomes one `--red-flag` argument.

#### Scenario: write-memory executes a runbook handoff

- **WHEN** `learn` hands off a confirmed runbook (slug, situation, done_when, body, source, position, optional target, optional red_flags) to the write-memory runbook
- **THEN** the write-memory runbook's steps compose and run the corresponding `engram learn runbook` command, with one `--red-flag` per handed-off entry, and report the written note path
