# Agent Instruction Bundle - Technical Spec

## 792b32a Orientation

- Ownership: Repository component (the shipped skill and guidance markdown).
- Responsibility: constrains the instruction artifacts that engram ships to an agent harness: the metadata a harness
  reads to load a skill, the form of every `engram` command those artifacts show, and the rule that the agent
  reaches engram only through its command line and writes the vault only through a sync-preserving command.
- Current realization (informational): `agent-instructions/skills/*/SKILL.md` and
  `agent-instructions/guidance/*.md`, checked by `dev/skillsmeta/metadata_test.go`.
- Qualification: responsibility, the shape of what is shipped to an agent is stated without naming any file;
  ownership, no other component constrains the text of a skill, while [harness-deploy-sync](./harness-deploy-sync.md) only copies it and each
  skill capability owns its own procedure; independent verification, the metadata limit is checked directly
  against the shipped files; stability, the constraints survive a new skill or a new harness; necessity, a
  harness loads a skill only when its metadata fits, and a command template that breaks on a dash-led value fails
  silently at the agent.
- Supported capabilities: [write-memory](../product/write-memory.md). Later capabilities that extend it: [recall](../product/recall.md), [learn](../product/learn.md),
  [lessons-contract](../product/lessons-contract.md), [route](../product/route.md), [please](../product/please.md), [ambient-guidance](../product/ambient-guidance.md) and curate.

Rationale: behavior lives in markdown skills that the agent executes, and deterministic compute lives in the
binary. The two communicate only through the command line (arguments in, standard output out) and the vault on
disk, and each subcommand is its own process ([792b2a Orientation](./command-line-surface.md#792b2a-orientation)). That split keeps the judgment layer out of Go code and
leaves the invariant checker, which gates the binary, with only deterministic code to gate. The skills are gated by
acceptance tests and measured baselines instead of contract tests, so the constraints they share are written here
once and not repeated in every skill capability.

## 792b32c Behavior Requirements

### 792b32c1 Skill metadata fits the harness limit

Requirement: [792b32c1 Skill metadata fits the harness limit](#792b32c1-skill-metadata-fits-the-harness-limit) The agent-instruction-bundle MUST ship every skill as a `SKILL.md` that begins with YAML
frontmatter, whose parsed `description` is non-empty and shorter than 1024 characters (counted as characters, not
bytes).

Supports: none. The limit belongs to the harness that loads the skill, not to a product behavior.

Rationale: a harness loads a skill from its metadata, and a description past the limit is not loaded.

#### 792b32c1a Scenarios

```gherkin
Scenario: 792b32c1a1 Every shipped skill has frontmatter
  Given the SKILL.md of each skill under agent-instructions/skills
  When its text is split at the frontmatter delimiters
  Then it has a frontmatter block that parses as YAML

Scenario: 792b32c1a2 Every shipped skill has a description
  Given the parsed frontmatter of each shipped skill
  When its description is read
  Then it is not empty

Scenario: 792b32c1a3 A description stays under the limit
  Given the parsed description of each shipped skill
  When its characters are counted
  Then the count is fewer than 1024
```

### 792b32c2 Free-text flag values use the attached form

Requirement: [792b32c2 Free-text flag values use the attached form](#792b32c2-free-text-flag-values-use-the-attached-form) The agent-instruction-bundle MUST write every free-text `engram` flag value in a shipped
instruction, whether in a skill, a guidance document or a string the binary emits to an agent, as `--flag="value"`
with the quote touching the `=`, and MUST NOT write it as `--flag "value"`. The rule covers every free-text flag of
`query` (`--phrase`, `--text`), `learn`, `amend` and `resituate` (`--situation`, `--subject`, `--predicate`,
`--object`, `--behavior`, `--impact`, `--action`, `--done-when`, `--body`, `--question`, `--answer`, `--source`,
`--supersedes`, `--trigger`, `--red-flag`).

Supports: [792a13h1a A value beginning with a dash survives](../product/write-memory.md#792a13h1-uat-scenarios).

Rationale: the command-line parser reads a value that begins with `-` as a new flag and exits with
`flag needs an argument`. A free-text value, such as a user's message or an agent's phrasing of a situation, can
begin with `-` with no control over its shape, and the space-separated form has no defense against that.

#### 792b32c2a Scenarios

```gherkin
Scenario: 792b32c2a1 A skill's example command uses the attached form
  Given a shipped SKILL.md that shows an engram command with a free-text flag
  When the example is read
  Then the flag is written --flag="value", not --flag "value"

Scenario: 792b32c2a2 A value beginning with a dash succeeds
  Given a template followed literally with a free-text value that begins with "-"
  When the resulting engram command runs
  Then it exits 0, because the value was attached to its flag

Scenario: 792b32c2a3 A binary-emitted instruction uses the attached form
  Given the engram binary emits a string that tells an agent to run an engram command with a free-text value
  When the string is read
  Then the value is written --flag="value"
```

### 792b32c3 A skill states the attached form once

Requirement: [792b32c3 A skill states the attached form once](#792b32c3-a-skill-states-the-attached-form-once) The agent-instruction-bundle MUST have every skill whose body shows an `engram` command with a
free-text flag state, at least once and in its own words, that the value must be passed as `--flag="value"` because a
value that begins with `-` otherwise breaks the command. A guidance document read on every turn MAY state the rule
more tersely.

Supports: [792a13h1a A value beginning with a dash survives](../product/write-memory.md#792a13h1-uat-scenarios).

Rationale: an agent that improvises its own command, instead of copying an example, still uses the attached form.

#### 792b32c3a Scenarios

```gherkin
Scenario: 792b32c3a1 A skill states the rule and its reason
  Given the full body of a skill that shows engram commands with free-text flags
  When it is read
  Then at least one sentence says a free-text value must use --flag="value" and names the dash-led value as the reason

Scenario: 792b32c3a2 An improvised command uses the attached form
  Given an agent following a skill that states the rule composes an engram command in its own words
  When the command carries a free-text value
  Then the value is attached to its flag as --flag="value"
```

### 792b32c4 Skills reach the vault only through the command line

Requirement: [792b32c4 Skills reach the vault only through the command line](#792b32c4-skills-reach-the-vault-only-through-the-command-line) The agent-instruction-bundle MUST have every shipped skill reach engram's deterministic
compute only by running the `engram` command line and reading its standard output and the vault on disk, and MUST
NOT have a skill create, edit or delete a vault note file except through a sync-preserving command (`engram learn`,
`engram amend` or `engram resituate`). Reading a note's file to judge it MAY be done directly.

Supports: [792a13a1a A complete handoff is executed](../product/write-memory.md#792a13a1-uat-scenarios).

Rationale: a hand edit can desync a note from its sidecar and from the two copies of its `situation`, which the
invariant checker cannot see in the skill layer. Reading member files directly stays allowed because the agent
decides, per cluster, whether the abstract signal is worth investigating (operator ruling, 2026-06-04); that lazy,
judgment-based read is intended. A situation edit was a manual file poke until `engram resituate` replaced it.

#### 792b32c4a Scenarios

```gherkin
Scenario: 792b32c4a1 A new note is written by learn
  Given a skill that captures a lesson
  When it writes the note
  Then it runs engram learn and does not create a note file itself

Scenario: 792b32c4a2 An existing note is edited by amend or resituate
  Given a skill that changes a note's content or situation
  When it writes the change
  Then it runs engram amend or engram resituate and does not edit the note file
```
