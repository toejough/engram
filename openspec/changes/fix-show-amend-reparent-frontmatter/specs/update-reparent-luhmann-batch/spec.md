## MODIFIED Requirements

### Requirement: Derive phase proposes candidates from existing embeddings
Given `--reparent-luhmann` with no answers file, the command SHALL compute, for each top-level
note, its nearest-neighbor top-level notes (top-3) by embedding-sidecar cosine similarity above
a similarity floor (0.75), and emit a JSON payload of candidate pairs plus a fingerprint of the
vault's current state. It SHALL NOT write any file in this phase. The payload SHALL include a
`next_command` field naming the literal follow-up command (`engram update --reparent-luhmann
--answers <path>`), so the acting agent does not need to infer the flag shape from
documentation elsewhere. The payload's `instruction` field SHALL agree with the learn skill's "Batch mode — Luhmann re-eval
answers" section. It SHALL name that skill mode as the procedure that holds the continuation/sibling/top
disposition test. It SHALL state the answers-file JSON shape and require exactly one entry per distinct
candidate `note`. It SHALL say that the finished answers file is handed back to the user or orchestrating
agent, who previews it with `next_command` plus `--dry-run` and then applies it without `--dry-run`. The
answering pass itself SHALL NOT be told to run apply.

#### Scenario: Candidates emitted for review
- **WHEN** `--reparent-luhmann` runs with no `--answers` file against a vault with top-level
  notes that have above-floor similarity neighbors
- **THEN** the command exits after printing a JSON payload of candidate pairs, a fingerprint,
  and a `next_command` field, without renaming or rewriting any file

#### Scenario: No candidates above the floor
- **WHEN** no top-level note pair has similarity above the floor
- **THEN** the command reports no candidates found and exits without writing

#### Scenario: Derive instruction routes to learn's batch mode
- **WHEN** `--reparent-luhmann` runs with no `--answers` file against a vault with above-floor candidates
- **THEN** the payload's `instruction` names the learn skill's batch mode, requires one entry per
  distinct candidate `note`, names `--dry-run` as the preview before applying, and does not tell the
  answering pass to run `engram update --reparent-luhmann --answers` itself

## ADDED Requirements

### Requirement: Rename and rewrite SHALL refuse unsafe frontmatter before any write
The shared rename and rewrite step that reparent apply and `register-skills --adopt` both use SHALL check
every note it is about to rename before it renames or writes any file. For each such note it SHALL compute
the rewritten content (new `luhmann:`, alias, and reference rewrites). The note SHALL be refused when:
- its rewritten frontmatter does not decode as a YAML mapping, for example because the old `luhmann:`
  value carried an anchor that another key aliases; or
- the decoded `luhmann:` value is not the new id; or
- the note's frontmatter opening delimiter uses CRLF line endings.
A refusal SHALL fail the whole invocation with an error naming each refused note and the reason, and no
note SHALL be renamed or written by that invocation. `--dry-run` SHALL report the same refusals without
writing.

#### Scenario: Anchored luhmann value is refused, not corrupted
- **WHEN** an apply would rename a note whose frontmatter has `luhmann: &a "1050"` and `issue: *a`
- **THEN** the invocation fails naming that note, and no note file or sidecar in the vault is renamed or rewritten

#### Scenario: CRLF note is refused, not left stale
- **WHEN** an apply would rename a note whose frontmatter lines end in `\r\n`
- **THEN** the invocation fails naming that note as CRLF, and no file is renamed, so no note ends up with a basename id that disagrees with its `luhmann:` field

#### Scenario: Safe notes rename with every other key intact
- **WHEN** an apply renames notes whose frontmatter is LF and anchor-free
- **THEN** each renamed note decodes, its `luhmann:` equals its new id, and every other top-level key other than `aliases` decodes to the same value as before
