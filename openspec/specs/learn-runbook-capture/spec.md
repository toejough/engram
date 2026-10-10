# Runbook Note Capture Specification

## Purpose

Residue awaiting later #792 nodes. Runbook capture itself (`engram learn runbook`, its schema,
Luhmann placement, red flags, triggers, and embed and vocabulary on write) is in
`docs/changes/792_documentation-framework-adoption/792a_product-spec.md#792a3-added-note-writing`.
The write-memory requirement below waits for node 792c7l1 (write-memory). The skill runbook note
requirement is in `docs/changes/792_documentation-framework-adoption/792a_product-spec.md#792a12v-skill-notes-behave-like-captured-runbooks`.

## Requirements
### Requirement: `write-memory` SHALL compose and execute runbook-note writes

The write-memory skill (`agent-instructions/skills/write-memory/SKILL.md`; its vault runbook is the registered, derived mirror of that skill per capability `skill-runbook-registration`) SHALL compose the `engram learn runbook` command from fields handed off by `learn` (slug, situation, done_when, body/steps, source, target/position disposition, optional contributors, optional red_flags list), execute it, verify the result, and report the written note path — consistent with how it handles `fact` and `feedback` handoffs. Each handed-off red flag becomes one `--red-flag` argument.

#### Scenario: write-memory executes a runbook handoff

- **WHEN** `learn` hands off a confirmed runbook (slug, situation, done_when, body, source, position, optional target, optional red_flags) to the write-memory skill
- **THEN** the write-memory skill's steps compose and run the corresponding `engram learn runbook` command, with one `--red-flag` per handed-off entry, and report the written note path
