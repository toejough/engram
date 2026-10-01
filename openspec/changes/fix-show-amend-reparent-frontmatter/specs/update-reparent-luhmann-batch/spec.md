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

### Requirement: Rename and rewrite SHALL refuse undecodable frontmatter and convert written CRLF notes to LF
Reparent apply and `register-skills --adopt` share one rename and rewrite step. That step SHALL check every
note it is about to rename before it renames or writes any file. For each such note, it SHALL compute the
rewritten content: the new `luhmann:`, the alias, and the reference rewrites. It SHALL refuse the note when
the rewritten frontmatter does not decode as a YAML mapping. One way this happens is when the old
`luhmann:` value carried an anchor that another key aliases. It SHALL also refuse the note when the decoded
`luhmann:` value is not the new id. A refusal SHALL fail the whole invocation with an error that names
each refused note and its reason. That invocation SHALL NOT rename or write any note. `--dry-run` SHALL
report the same refusals without writing.

Before rewriting a note that uses CRLF line endings, the step SHALL convert every `\r\n` to `\n`. That
applies to the note's frontmatter and body, and to a renamed note as well as a referrer. The step SHALL
convert only notes it writes anyway. The conversion SHALL happen inside the same single atomic write as
that note's rewrite. A CRLF note that the step does not otherwise write SHALL remain byte-identical. Every
converted note SHALL have its embedding sidecar rebuilt in the same invocation, so that afterwards it is
not stale. A converted note's exchange hash SHALL equal the hash of the same note authored with LF line
endings. A note whose frontmatter was already LF SHALL keep its exchange hash unchanged.

#### Scenario: Anchored luhmann value is refused, not corrupted
- **WHEN** an apply would rename a note whose frontmatter has `luhmann: &a "1050"` and `issue: *a`
- **THEN** the invocation fails naming that note, and no note file or sidecar in the vault is renamed or rewritten

#### Scenario: CRLF renamed note is converted and rewritten
- **WHEN** an apply renames a note whose lines end in `\r\n`
- **THEN** the written note contains no `\r\n`, its `luhmann:` equals its new id, its alias is recorded, every other key decodes to its prior value, and its sidecar is fresh after the invocation

#### Scenario: CRLF referrer gets its supersedes rewritten
- **WHEN** a CRLF note's frontmatter `supersedes:` names a note the apply renames
- **THEN** the referrer is written as LF with its `supersedes:` note rewritten to the new basename

#### Scenario: Untouched CRLF notes stay byte-identical
- **WHEN** a CRLF note is neither renamed nor references a renamed note
- **THEN** its bytes are unchanged after the apply

#### Scenario: Exchange hash survives conversion
- **WHEN** a note with LF frontmatter and a CRLF body is converted by a rename
- **THEN** its exchange hash is unchanged, and a note converted from CRLF frontmatter hashes the same as its LF-authored equivalent

#### Scenario: Safe notes rename with every other key intact
- **WHEN** an apply renames notes whose frontmatter is anchor-free
- **THEN** each renamed note decodes, its `luhmann:` equals its new id, and every other top-level key other than `aliases` decodes to the same value as before
