# Ingest Auto-Sweep with Non-Persistent-Workspace Skip Specification

## Purpose

`engram ingest --auto` keeps the chunk index current by mechanically sweeping every known session and markdown source, while excluding session logs and sweep roots whose paths sit under throwaway filesystem prefixes (`/tmp`, `/private/tmp`, `/var/folders`, `/private/var/folders`), so running evals or tests doesn't bloat the real vault's index. An ancestor `.claude` dir's sweep now also excludes its `jobs/` subdirectory (agent-harness scratch that can include whole snapshot copies of the vault), restoring parity with the `.pi` ancestor sweep. Sources that yield zero chunk records no longer leave 0-byte `.jsonl` index files behind, since the read path must open and enumerate each file on every query. `engram prune --empty` (+ `--dry-run`) separately cleans up any pre-existing empty files. Why: `docs/architecture/adr.md` ADR-0021 (decision 4 — `.pi`/`.claude` sweep parity, `jobs/` exclusion); the non-persistent-workspace skip and the 0-byte index-file guard have no dedicated ADR — they are locked by their unit tests (internal/cli/sweepspec_test.go, internal/cli/ingest_test.go). Validation: `internal/cli` ingest/sweep unit tests (TestRunIngestSkipsEmptyIndexFile, TestResolveSweepRootsDropsRootsUnderNonPersistentPaths, TestResolveSweepRootsClaudeAncestorExcludesJobsScratch); real-scale forensics: 1,397 of 1,960 duplicates removed by prune (71%) traced to the `.claude/jobs` tree, stopped at source by this exclusion.

## Requirements

### Requirement: SHALL skip session logs under non-persistent filesystem roots

During `--auto` sweep of session logs, any session log whose resolved directory path sits under a throwaway filesystem prefix (`/tmp`, `/private/tmp`, `/var/folders`, `/private/var/folders`) SHALL be skipped entirely, so eval and test runs don't permanently index transient session data. Explicit --transcript flags bypass this check.

#### Scenario: Session log under /tmp is skipped
- **WHEN** `engram ingest --auto` runs and session logs exist under `/tmp/eval-run-xyz`
- **THEN** no sessions under that tree are indexed or added to the manifest

#### Scenario: Explicit transcript under /tmp is indexed
- **WHEN** `engram ingest --transcript /tmp/explicit-session.jsonl` is run explicitly
- **THEN** the transcript is indexed despite the throwaway path, because explicit sources bypass the check

### Requirement: SHALL drop entire sweep roots whose path is under non-persistent prefixes

When resolving the repo-markdown root or ancestor `.claude`/`.pi` roots during `--auto`, if a resolved root's own path sits under a throwaway filesystem prefix, that root SHALL be dropped entirely before the walk begins (not just its subdirectories). Previously, only a session log's own subdirectories were checked, so a repo-markdown root resolving to `/tmp/eval-pool/warm0/` (no VCS marker) was swept in full and permanently indexed. This fix applies at root-derivation time via NonPersistentPathPrefixes configuration.

#### Scenario: Repo root under /tmp is not swept
- **WHEN** cwd is `/tmp/test-pool/run123` with no VCS marker, so the repo-markdown root resolves to that path
- **THEN** `engram ingest --auto` skips the entire root instead of indexing it

#### Scenario: Ancestor dir under non-persistent path is dropped
- **WHEN** walking the ancestor chain and finding a `.claude` dir at `/private/var/folders/xyz/...` (a macOS $TMPDIR location)
- **THEN** that root is dropped and not swept at all

### Requirement: SHALL exclude `.claude/jobs` from ancestor `.claude` sweeps

When sweeping ancestor `.claude` directories, the `jobs/` subdirectory SHALL be explicitly excluded, since it contains agent-harness scratch data including whole snapshot copies of the vault. This restores parity with the `.pi` ancestor sweep, which already excluded `jobs/`.

#### Scenario: .claude/jobs is not swept
- **WHEN** `engram ingest --auto` sweeps an ancestor `.claude` directory
- **THEN** the `jobs/` subdirectory within it is skipped and not indexed

#### Scenario: .claude/jobs exclusion prevents duplicate backlog
- **WHEN** the system sweeps a `.claude` ancestor without excluding `jobs/`
- **THEN** vault snapshots in `jobs/` are indexed as duplicates of the live vault, accumulating manifest entries that `prune --duplicates` later removes (measured: 71% of a large dedup run traced to this tree)

### Requirement: Sources yielding zero records SHALL NOT write 0-byte index files

When rebuildIndex processes a source that yields zero chunk records (fully deduplicated or non-embeddable content), it SHALL skip writing a `.jsonl` index file entirely. The manifest and dedup state are recorded independently, so nothing downstream requires the empty file. Previously, 0-byte files were written and the read path had to open and enumerate them on every query.

#### Scenario: Empty source does not create index file
- **WHEN** a source is ingested that produces no chunk records after deduplication
- **THEN** no `.jsonl` file is written for that source

#### Scenario: Prune --empty removes pre-existing empty files
- **WHEN** `engram prune --empty` runs against an index with pre-existing 0-byte `.jsonl` files
- **THEN** those files are removed; the cleanup is ranking-neutral (empty files contribute zero records to search)

#### Scenario: Dry run reports empty files that would be removed
- **WHEN** `engram prune --empty --dry-run` runs
- **THEN** it reports how many 0-byte files would be removed without actually deleting them
