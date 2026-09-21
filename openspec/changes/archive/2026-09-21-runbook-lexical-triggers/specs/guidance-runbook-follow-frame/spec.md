## MODIFIED Requirements

### Requirement: The shim SHALL name the first action as a literal `engram query` on every user request

Before the first tool call on any user request, the shim SHALL instruct the agent to run `engram query` with at least two `--phrase` values (the request in the agent's own words; the kind of situation it is in) and a `--text` value carrying the user's message verbatim (first ~300 characters if long, never rewritten), as a bare imperative naming the command, so that the instruction is executable with no `recall` skill present. The trigger SHALL NOT depend on the agent judging the task multi-step, recurring, or previously done: that judgment is what the returned runbook supplies, so it cannot gate the lookup. Over-fire on one-shot requests is accepted (task-init fire-unit, measured 3.4x; per-fire cost is one query with no crystallization).

#### Scenario: Shim-only session runs the first query

- **WHEN** an agent with the shim in CLAUDE.md and no engram skills installed receives a user request
- **THEN** its first tool call is an `engram query` carrying the request phrase, the situation phrase, and `--text` with the user's message verbatim, before any other tool call

#### Scenario: A one-shot request still fires

- **WHEN** the request is a single-step action (a one-line answer, a single-file typo fix)
- **THEN** the query still runs first; the agent proceeds without a runbook when none matches

#### Scenario: The raw text is not paraphrased

- **WHEN** the user's message is "/please rename the tally CLI's --out flag to --output"
- **THEN** the `--text` value contains that string verbatim up to whitespace differences; the paraphrase goes only in `--phrase`
