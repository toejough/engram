## ADDED Requirements

### Requirement: A shipped instruction template SHALL pass a free-text `engram` flag value in `--flag="value"` form, never space-separated

Any shipped instruction surface that shows an `engram` command with a free-text flag value — a skill's `SKILL.md`, a guidance doc, or a string the `engram` binary itself emits to an agent as an instruction — SHALL write that flag as `--flag="value"` (the value in quotes immediately after `=`), not `--flag "value"` (space-separated). `engram`'s CLI parser reads a value beginning with `-` as a new flag and exits non-zero with `flag needs an argument: --flag`; a free-text value (a user's message, an agent's own phrasing of a situation, subject, behavior, or claim) can begin with `-` with no control over its shape, and the space-separated form has no defense against that. This requirement applies to every free-text flag across `engram query` (`--phrase`, `--text`), `engram fact`/`feedback`/`runbook`/`qa`/`amend`/`resituate` (`--situation`, `--subject`, `--predicate`, `--object`, `--behavior`, `--impact`, `--action`, `--done-when`, `--body`, `--question`, `--answer`, `--source`, `--supersedes`, `--trigger`, `--red-flag`).

#### Scenario: A skill's example command uses the safe form

- **WHEN** a skill's `SKILL.md` shows an example `engram` command with a free-text flag value
- **THEN** the example writes it as `--flag="value"`, not `--flag "value"`

#### Scenario: A value beginning with a dash succeeds

- **WHEN** an agent follows a shipped template literally with a free-text value that begins with `-` (for example a situation phrase starting with "- " or a flag-like word)
- **THEN** the resulting `engram` command exits 0, because the value was passed in `--flag="value"` form

#### Scenario: A binary-emitted instruction string uses the safe form

- **WHEN** the `engram` binary itself emits a string instructing an agent to run an `engram` command with a free-text flag value (for example a pending-offer curation cue)
- **THEN** that emitted string also uses the `--flag="value"` form

### Requirement: A skill SHALL explain the safe form once, so an agent improvising its own command also uses it

A skill whose body shows one or more example `engram` commands with a free-text flag SHALL state, at least once, in its own words, that a free-text flag value must be passed as `--flag="value"` because a value beginning with `-` otherwise breaks the command — so an agent that improvises its own command (rather than copying an example verbatim) still uses the safe form. Guidance docs read on every turn (the shim) MAY state this more tersely than a skill body, since every token there recurs on every turn.

#### Scenario: A skill states the rule once

- **WHEN** a skill's body is read in full
- **THEN** at least one sentence states that a free-text flag value must use `--flag="value"` form, and names the reason (a value beginning with `-` is otherwise misread as a flag)

#### Scenario: An improvised command still uses the safe form

- **WHEN** an agent composes an `engram` command of its own wording, rather than copying a template's example verbatim, following a skill that states the rule
- **THEN** the agent's own command also uses the `--flag="value"` form for its free-text values
