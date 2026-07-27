# Chunk-Index Dedup by Content Hash Specification

## Purpose

When the same content is reachable from multiple paths (a project file copied into a `.claude` ancestor dir, a session transcript duplicated by a worktree), engram groups sources by content hash and indexes only one canonical copy per group, selected by a fixed precedence order (explicit flags > repo root > ancestor `.claude`/`.pi` dirs > other), rather than creating duplicate index files and manifest entries that waste disk and let the same content surface multiple times in search results. Duplicate index files are removed only after verifying the retained canonical's index covers every chunk record the duplicate ever held (not merely that both sources have identical bytes today), since each per-source `.jsonl` index accumulates records from every past ingest of that path; byte-identical sources today can hold entirely disjoint historical records. `engram prune --duplicates` applies the same dedup retroactively to indexes that accumulated duplicates before this capability existed, and `engram update` detects a live duplicate backlog and notifies the user with the exact command when cleanup would remove something. Why: `docs/architecture/adr.md` ADR-0021. Validation: `dev/eval/LEDGER.md#chunk-index-dedup` (unit tests lock canonical selection, record-subset gate, and convergence; real-scale measurement on ~62K-source index: 1,960 of 8,572 duplicates removed −23%).

## Requirements

### Requirement: Sources SHALL be grouped by content hash AND chunking class

Sources SHALL be partitioned by both their content hash and their chunking class (`.jsonl` transcripts vs markdown documents), so a transcript and a markdown file with identical bytes are never treated as duplicates of each other, even though their chunk records differ.

#### Scenario: Transcript and markdown with same bytes are not duplicates
- **WHEN** a session transcript file and a markdown file have identical byte content (same hash)
- **THEN** they are placed in separate hash groups and both are indexed independently, not as duplicates

#### Scenario: Two sources with same content hash and same class are grouped together
- **WHEN** two markdown files or two transcript files have identical content
- **THEN** they are placed in the same hash group for canonical selection

### Requirement: Canonical SHALL be selected by fixed precedence order

Within a hash group, the canonical member SHALL be selected by precedence: explicit (--transcript/--markdown/--pi-sessions) > repo-markdown root > ancestor `.claude`/`.pi` dir (closest first) > session log or manual --sweep. When multiple candidates tie on origin rank, the closest ancestor wins; further tie-breaks go to fewest path separators, then lexicographic path order.

#### Scenario: Explicit flag outranks repo root
- **WHEN** a group contains an explicitly-named source and the same content found in the repo-markdown root
- **THEN** the explicit source is selected as canonical, and the repo root copy is marked a duplicate

#### Scenario: Repo root outranks ancestor dirs
- **WHEN** a group contains the same content in both the repo root and a `.claude` ancestor directory
- **THEN** the repo root copy is selected as canonical

#### Scenario: Closest ancestor outranks farther ancestors
- **WHEN** a group contains the same content in both a `.claude` dir at cwd level 1 and a `.claude` dir at cwd level 3
- **THEN** the closer ancestor (level 1) is selected as canonical

### Requirement: Duplicate eviction gate SHALL verify record-level subset coverage

Before removing a duplicate's index file, the system SHALL verify that every chunk record (identified by content hash) present in the duplicate's index is already present in the canonical's index. This covers the case where two sources are byte-identical today but their index files hold different historical chunk records from prior ingests, since mergeChunkRecords is append-only within a source. If the gate fails, eviction is refused and the duplicate is left for a later run once the canonical catches up.

#### Scenario: Duplicate with superset of records is refused
- **WHEN** canonical's index has 5 chunk records and duplicate's index has 7 chunk records (a superset)
- **THEN** the duplicate is refused eviction rather than deleted; the run reports "needs review" and suggests manual investigation

#### Scenario: Duplicate with subset is safely evicted
- **WHEN** canonical's index contains all chunk records present in duplicate's index
- **THEN** the duplicate's index file and manifest entry are removed

### Requirement: `prune --duplicates` SHALL retroactively deduplicate

`engram prune --duplicates` (+ `--dry-run`) SHALL apply the same record-level subset gate retroactively to a chunk index, grouping live manifest entries by (hash, chunking class), skipping entries already tagged DuplicateOf, and evicting non-canonical members one per hash group.

#### Scenario: Prune reconciles groups from live manifest
- **WHEN** `engram prune --duplicates` runs against a chunk index with multiple indexed members per hash group
- **THEN** it groups entries by hash and chunking class, skips pre-marked duplicates, selects canonical per group, and removes the rest

#### Scenario: Dry run shows what would be removed
- **WHEN** `engram prune --duplicates --dry-run` runs
- **THEN** it reports removals, refusals, and failed removals without modifying the manifest

### Requirement: `engram update` SHALL detect backlog and notify with exact command

`engram update` SHALL detect a live duplicate backlog (manifest entries with multiple canonical members per hash group) and notify the user with the exact `engram prune --duplicates` command when cleanup would remove something, rather than removing anything on its own.

#### Scenario: Update reports prune command when duplicates exist
- **WHEN** a chunk index has multiple canonical entries per hash group
- **THEN** `engram update` prints a notice with the exact `engram prune --duplicates` command and its expected impact
