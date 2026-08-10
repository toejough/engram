## Purpose

`engram update --reparent-luhmann` is a standalone, opt-in, one-shot migration that re-parents
existing top-level vault notes into `continuation`/`sibling` placement, giving the
`update-flat-vault-luhmann-notice` notice a real remedy. Modeled on `vocab refit`'s derive →
naming_requests → answers → apply flow, since "is note B a sub-point of note A" is a content
judgment the binary cannot make from embeddings alone — the same boundary
`restore-luhmann-branching-disposition` (#701) drew for per-capture disposition.

## Requirements

### Requirement: Reparent command is opt-in only
`engram update --reparent-luhmann` SHALL run only when the user explicitly passes the flag. It
SHALL NOT run automatically as part of a plain `engram update`.

#### Scenario: Plain update never reparents
- **WHEN** the user runs `engram update` without `--reparent-luhmann`
- **THEN** no note is renamed and no wikilink/supersedes reference is rewritten

### Requirement: Derive phase proposes candidates from existing embeddings
Given `--reparent-luhmann` with no answers file, the command SHALL compute, for each top-level
note, its nearest-neighbor top-level notes (top-3) by embedding-sidecar cosine similarity above
a similarity floor (0.75), and emit a JSON payload of candidate pairs plus a fingerprint of the
vault's current state. It SHALL NOT write any file in this phase.

#### Scenario: Candidates emitted for review
- **WHEN** `--reparent-luhmann` runs with no `--answers` file against a vault with top-level
  notes that have above-floor similarity neighbors
- **THEN** the command exits after printing a JSON payload of candidate pairs and a
  fingerprint, without renaming or rewriting any file

#### Scenario: No candidates above the floor
- **WHEN** no top-level note pair has similarity above the floor
- **THEN** the command reports no candidates found and exits without writing

### Requirement: Apply phase consumes agent-authored disposition answers
Given `--reparent-luhmann --answers <file>`, where the file supplies a `position`
(`top`/`continuation`/`sibling`) and `target` for each candidate note the agent judged, the
command SHALL compute new Luhmann IDs (via the existing `nextLuhmannID` machinery, processed in
ascending original-ID order so renumbering within one run cannot collide), rename each affected
note file and its `.vec.json` sidecar, update the note's `luhmann:` frontmatter field, and
rewrite every incoming wikilink/`supersedes` reference vault-wide to the new basename —
including cascading renames, where a note being renamed also references another note renamed in
the same run.

#### Scenario: Continuation answer renames and rewrites backlinks
- **WHEN** an answers file marks note `12` as `continuation` of note `7`, and note `20`
  contains a `[[12.2026-05-01.old-slug]]` wikilink
- **THEN** note `12`'s file and sidecar are renamed to the new child ID under `7`, its
  `luhmann:` frontmatter field is updated, and note `20`'s wikilink is rewritten to the new
  basename

#### Scenario: Top answer is a no-op
- **WHEN** an answers file marks a candidate note as `position: top`
- **THEN** that note is not renamed and no reference to it is rewritten

#### Scenario: Stale answers are rejected
- **WHEN** the vault's current state no longer matches the fingerprint recorded in the answers
  file (the vault changed between derive and apply)
- **THEN** the apply is rejected with a stale-answers error and no file is written; the derive
  phase must be re-run to produce a fresh payload

### Requirement: Dry-run shows intended renames and requires answers
`--reparent-luhmann --dry-run` SHALL require `--answers <file>` and SHALL print the intended
renames — the old→new basename map plus every reference that would be rewritten for each —
without renaming or rewriting any file. The derive phase (no `--answers`) never writes
regardless of `--dry-run`, so `--dry-run` without `--answers` is rejected as a usage error.

#### Scenario: Dry-run apply preview
- **WHEN** `--reparent-luhmann --answers <file> --dry-run` runs against valid, non-stale answers
- **THEN** the command prints the old→new basename map and the list of referencing notes that
  would be rewritten for each, and no file on disk is modified

#### Scenario: Dry-run without answers is rejected
- **WHEN** the user runs `--reparent-luhmann --dry-run` without `--answers`
- **THEN** the command rejects the invocation with a usage error and writes nothing

### Requirement: Chunk index requires a manual prune step after rename
`engram ingest --auto` run after a `--reparent-luhmann` apply SHALL index the renamed note
cleanly under its new path, but is NOT required to automatically detach the old path's manifest
entry — this is a known gap (issue #724). The old entry's chunks remain searchable until the
user runs a manual `engram prune` (not `--duplicates`, since a renamed note's frontmatter change
alters its content hash, so it is never a byte-identical duplicate).

#### Scenario: Old path lingers after ingest alone
- **WHEN** a note is renamed via `--reparent-luhmann` apply and `engram ingest --auto` is then
  run
- **THEN** the new path is indexed cleanly, and the old path's manifest entry and chunk index
  file remain present (no error, no automatic detach)

#### Scenario: Manual prune detaches the stale entry
- **WHEN** `engram prune` (without `--duplicates`) is run after the above
- **THEN** the old path's manifest entry is detached (its already-embedded chunks remain
  searchable, per prune's existing design)
