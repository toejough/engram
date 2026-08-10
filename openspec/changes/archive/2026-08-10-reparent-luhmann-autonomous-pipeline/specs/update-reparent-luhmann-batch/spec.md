## MODIFIED Requirements

### Requirement: Derive phase proposes candidates from existing embeddings
Given `--reparent-luhmann` with no answers file, the command SHALL compute, for each top-level
note, its nearest-neighbor top-level notes by embedding-sidecar cosine similarity, and emit a
JSON payload of candidate pairs (each above a similarity floor) plus a fingerprint of the
vault's current state. It SHALL NOT write any file in this phase. The output SHALL include an
explicit, literal next-command instruction — naming `engram update --reparent-luhmann --answers
<path>` with the answers-file path as a fillable placeholder — so the acting agent does not need
to infer the flag shape from documentation elsewhere.

#### Scenario: Candidates emitted for review
- **WHEN** `--reparent-luhmann` runs with no `--answers` file against a vault with top-level
  notes that have above-floor similarity neighbors
- **THEN** the command exits after printing a JSON payload of candidate pairs, a fingerprint,
  and a literal next-command instruction, without renaming or rewriting any file

#### Scenario: No candidates above the floor
- **WHEN** no top-level note pair has similarity above the floor
- **THEN** the command reports no candidates found and exits without writing

### Requirement: Apply phase consumes agent-authored disposition answers
Given `--reparent-luhmann --answers <file>` (no `--dry-run`), the command SHALL compute new
Luhmann IDs, rename each affected note file and its `.vec.json` sidecar, update the note's
`luhmann:` frontmatter field, rewrite every incoming wikilink/`supersedes` reference vault-wide
to the new basename (unchanged from prior behavior), AND additionally, in the same invocation:
re-index the renamed notes (`RunIngest`) and detach the renamed notes' old-path chunk-manifest
entries (`RunPrune`, plain mode). No separate `engram ingest --auto` or `engram prune` command
SHALL be required for a complete, self-contained apply.

#### Scenario: Continuation answer renames, rewrites, and reconciles the chunk index
- **WHEN** an answers file marks note `12` as `continuation` of note `7`, and note `20`
  contains a `[[12.2026-05-01.old-slug]]` wikilink
- **THEN** note `12`'s file and sidecar are renamed to the new child ID under `7`, its
  `luhmann:` frontmatter field is updated, note `20`'s wikilink is rewritten to the new
  basename, the new path is indexed in the chunk manifest, and note `12`'s old-path manifest
  entry is detached — all within this single apply invocation

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
- **WHEN** the rename+rewrite step succeeds but the subsequent `RunIngest` or `RunPrune` step
  fails
- **THEN** the command reports which stage failed and that the vault rename itself is intact and
  authoritative, and that a manual `engram ingest --auto` / `engram prune` remains available as
  a safe fallback to finish reconciling the chunk index

### Requirement: Apply reports whether further candidates remain
After the mechanical pipeline completes successfully, apply SHALL check for remaining
above-floor candidates against the vault's now-current state (read-only; it does not emit a full
candidate payload) and report either that further candidates exist — with the literal
next-command instruction to re-run derive — or that no further candidates were found.

#### Scenario: Further candidates remain after this round
- **WHEN** a successful apply run's resulting vault state still has ≥1 above-floor top-level
  candidate pair
- **THEN** the apply's output reports the candidate count and the literal next command
  (`engram update --reparent-luhmann`) to continue the loop

#### Scenario: No further candidates
- **WHEN** a successful apply run's resulting vault state has zero above-floor candidate pairs
- **THEN** the apply's output reports the vault as fully evaluated, with no further next-command
  instruction

### Requirement: Dry-run shows intended renames and requires answers
`--reparent-luhmann --dry-run` SHALL require `--answers <file>` and SHALL print the intended
renames — the old→new basename map plus every reference that would be rewritten for each —
without renaming, rewriting, re-indexing, or pruning anything. The derive phase (no `--answers`)
never writes regardless of `--dry-run`, so `--dry-run` without `--answers` is rejected as a usage
error. `--dry-run` remains an optional inspection tool; it is not a required step before a normal
apply.

#### Scenario: Dry-run apply preview
- **WHEN** `--reparent-luhmann --answers <file> --dry-run` runs against valid, non-stale answers
- **THEN** the command prints the old→new basename map and the list of referencing notes that
  would be rewritten for each, and no file on disk is modified and no chunk-index change occurs

#### Scenario: Dry-run without answers is rejected
- **WHEN** the user runs `--reparent-luhmann --dry-run` without `--answers`
- **THEN** the command rejects the invocation with a usage error and writes nothing

## REMOVED Requirements

### Requirement: Chunk index requires a manual prune step after rename
**Reason**: Superseded — apply now performs the chunk-index reconciliation (re-index + stale-entry
detach) automatically as part of the same invocation (see "Apply phase runs the full mechanical
pipeline" above), so a manual follow-up `engram prune` is no longer required for the common case.
**Migration**: No action needed for new usage. An operator who ran the OLD `--reparent-luhmann`
apply (pre-this-change) and still has a stale manifest entry from that run can still resolve it
with a one-time manual `engram prune` — that fallback path continues to work, it's just no longer
the expected/required step for new applies.
