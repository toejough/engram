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
vault's current state. It SHALL NOT write any file in this phase. The payload SHALL include a
`next_command` field naming the literal follow-up command (`engram update --reparent-luhmann
--answers <path>`), so the acting agent does not need to infer the flag shape from
documentation elsewhere.

#### Scenario: Candidates emitted for review
- **WHEN** `--reparent-luhmann` runs with no `--answers` file against a vault with top-level
  notes that have above-floor similarity neighbors
- **THEN** the command exits after printing a JSON payload of candidate pairs, a fingerprint,
  and a `next_command` field, without renaming or rewriting any file

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
the same run. When the rename map is non-empty, apply SHALL then also re-index the renamed
notes and detach their old-path chunk-manifest entries, in-process, as part of this same
invocation — no separate `engram ingest --auto`/`engram prune` command is required. A failure in
that chunk-index reconciliation step SHALL NOT roll back the already-completed rename; it SHALL
be reported as which stage failed, that the vault rename remains intact/authoritative, and that
`engram ingest --auto`/`engram prune` remain available as a manual fallback. After a successful
apply (pipeline included, or skipped because nothing was renamed), the command SHALL report
whether further above-floor candidates exist in the vault's resulting state — with the literal
next command to continue — or that the vault is fully evaluated.

#### Scenario: Continuation answer renames, rewrites, and reconciles the chunk index
- **WHEN** an answers file marks note `12` as `continuation` of note `7`, and note `20`
  contains a `[[12.2026-05-01.old-slug]]` wikilink
- **THEN** note `12`'s file and sidecar are renamed to the new child ID under `7`, its
  `luhmann:` frontmatter field is updated, note `20`'s wikilink is rewritten to the new
  basename, the new path is indexed in the chunk manifest, and note `12`'s old-path manifest
  entry is detached — all within this single apply invocation, without a separate
  `engram ingest --auto`/`engram prune` step

#### Scenario: Top answer is a no-op
- **WHEN** an answers file marks a candidate note as `position: top`
- **THEN** that note is not renamed, no reference to it is rewritten, and no chunk-index change
  occurs for it

#### Scenario: Stale answers are rejected
- **WHEN** the vault's current state no longer matches the fingerprint recorded in the answers
  file (the vault changed between derive and apply)
- **THEN** the apply is rejected with a stale-answers error and no file is written and no
  chunk-index change occurs; the derive phase must be re-run to produce a fresh payload

#### Scenario: Chunk-index pipeline failure does not roll back the rename
- **WHEN** the rename+rewrite step succeeds but the subsequent re-index or manifest-detach step
  fails
- **THEN** the command reports which stage failed and that the vault rename itself is intact
  and authoritative, names `engram ingest --auto`/`engram prune` as a manual fallback, and does
  not revert the already-renamed note file(s)

#### Scenario: Apply reports further candidates or completion
- **WHEN** a successful apply run's resulting vault state still has ≥1 above-floor top-level
  candidate pair
- **THEN** the apply's output reports the candidate count and the literal next command
  (`engram update --reparent-luhmann`) to continue

#### Scenario: Apply reports the vault fully evaluated
- **WHEN** a successful apply run's resulting vault state has zero above-floor candidate pairs
- **THEN** the apply's output reports the vault as fully evaluated, with no further
  next-command instruction

### Requirement: Dry-run shows intended renames and requires answers
`--reparent-luhmann --dry-run` SHALL require `--answers <file>` and SHALL print the intended
renames — the old→new basename map plus every reference that would be rewritten for each —
without renaming, rewriting, re-indexing, or pruning anything. The derive phase (no `--answers`)
never writes regardless of `--dry-run`, so `--dry-run` without `--answers` is rejected as a
usage error. `--dry-run` is an optional inspection tool; a normal apply's own fingerprint-gating
is the safety net, so preview is never a required step before applying.

#### Scenario: Dry-run apply preview
- **WHEN** `--reparent-luhmann --answers <file> --dry-run` runs against valid, non-stale answers
- **THEN** the command prints the old→new basename map and the list of referencing notes that
  would be rewritten for each, and no file on disk is modified and no chunk-index change occurs

#### Scenario: Dry-run without answers is rejected
- **WHEN** the user runs `--reparent-luhmann --dry-run` without `--answers`
- **THEN** the command rejects the invocation with a usage error and writes nothing
