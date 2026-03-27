# Implementation Plan: Fix project_slug Never Written to Memory TOML Files (#389)

## Root Cause

`runFlush` in `internal/cli/flush.go` never resolves the project slug:

1. The `flush` subcommand has no `--project-slug` flag definition.
2. There is no `applyProjectSlugDefault` call in `runFlush`.
3. `runFlush` delegates to `RunLearn` by building a `learnArgs` slice — it never includes `--project-slug` in that slice.
4. `RunLearn` only calls `learner.SetProjectSlug()` when the flag value is non-empty, so it receives an empty string and sets nothing.

`hooks/stop.sh` calls `engram flush` with no `--project-slug` flag, relying on the binary to derive the slug from the working directory (via `os.Getwd()`). The same defaulting pattern exists in `runRecall` (calls `applyProjectSlugDefault`) but is absent in `runFlush`.

The `runCorrect` path has the same gap: it accepts `--project-slug` but never calls `applyProjectSlugDefault` when the flag is omitted.

## Fix

### 1. `internal/cli/flush.go` — add flag + default + pass to RunLearn

**Red:** Write a test `TestFlush_PassesProjectSlugToLearn` that:
- Invokes `runFlush` with `--transcript-path`, `--session-id`, and no `--project-slug`.
- Asserts that the `learnArgs` passed downstream include a non-empty `--project-slug` matching `ProjectSlugFromPath(os.Getwd())`.

Because `RunLearn` is called internally, the simplest test strategy is to run `flush` end-to-end with a stub transcript and assert the written memory TOML has a non-empty `project_slug` field. Use the existing pattern from `flush_test.go`.

**Green:**

In `runFlush`:

```go
projectSlug := fs.String("project-slug", "", "originating project slug")
```

After `applyDataDirDefault`:

```go
slugErr := applyProjectSlugDefault(projectSlug)
if slugErr != nil {
    return fmt.Errorf("flush: %w", slugErr)
}
```

In `learnArgs`:

```go
learnArgs := []string{
    "--transcript-path", *transcriptPath,
    "--session-id", *sessionID,
    "--data-dir", *dataDir,
    "--project-slug", *projectSlug,
}
```

### 2. `internal/cli/targets.go` — add ProjectSlug to FlushArgs

Add `ProjectSlug` to `FlushArgs` so the targ CLI surface is consistent:

```go
type FlushArgs struct {
    DataDir        string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
    TranscriptPath string `targ:"flag,name=transcript-path,desc=path to session transcript"`
    SessionID      string `targ:"flag,name=session-id,desc=session identifier"`
    ProjectSlug    string `targ:"flag,name=project-slug,desc=originating project slug"`
}
```

Update `FlushFlags` to include the new field:

```go
func FlushFlags(a FlushArgs) []string {
    return BuildFlags(
        "--data-dir", a.DataDir,
        "--transcript-path", a.TranscriptPath,
        "--session-id", a.SessionID,
        "--project-slug", a.ProjectSlug,
    )
}
```

Update the existing `TestFlushFlags` test to cover the new field.

### 3. `internal/cli/cli.go` — fix runCorrect (same gap)

In `runCorrect`, after `applyDataDirDefault`:

```go
slugErr := applyProjectSlugDefault(projectSlug)
if slugErr != nil {
    return fmt.Errorf("correct: %w", slugErr)
}
```

Remove the `if *projectSlug != ""` guard and always call `corrector.SetProjectSlug(*projectSlug)` — the default is now always populated.

## Backfill Strategy

Existing memories have `project_slug = ""`. Add a new `backfill-slugs` flag to `maintain` (or a standalone `migrate-slugs` subcommand following the `migrate-scores` pattern).

### New subcommand: `engram migrate-slugs`

**Approach:** For each memory TOML with an empty `project_slug`, derive the slug from the memory's file path. Engram stores memories under `~/.claude/engram/data/memories/`. The data directory itself is scoped per project by convention, but in the current single-data-dir model all memories share one directory — so the slug must come from a different source.

**Source of truth:** The session transcript's working directory. The `creationlog` `LogEntry` records `Timestamp` and `Filename`. By correlating memory `created_at` timestamps against session transcript `cwd` fields (available in Claude's hook JSON), we could derive slugs. However, this is complex and error-prone.

**Pragmatic fallback:** Since all 2,803 memories were created in the same engram development project context, backfill them with the slug derived from the current PWD when the migrate command is run (i.e., `applyProjectSlugDefault`). Add a `--slug` override flag for cases where the caller knows the correct slug.

**Algorithm:**

```
for each .toml in memoriesDir:
    if record.ProjectSlug == "":
        memory.ReadModifyWrite(path, func(r *MemoryRecord) {
            r.ProjectSlug = slug
        })
```

**Red:** Test `TestMigrateSlugs_BackfillsEmptySlugs` — creates memories with empty slug, runs command, asserts all have the expected slug. Test `TestMigrateSlugs_SkipsPopulated` — verifies memories that already have a slug are unchanged.

**Green:** Add `migrate-slugs` to `internal/cli/` following the `migrate.go` pattern.

The subcommand signature:

```
engram migrate-slugs [--data-dir <dir>] [--slug <override>] [--dry-run]
```

Dry-run prints `[engram] would set project_slug=<slug> on <N> memories` without writing.
Apply mode writes and prints `[engram] backfilled project_slug on <N> memories`.

Add `MigrateSlugsArgs` to `targets.go` and `MigrateSlugsFlags`.

## Test Changes Needed

| File | Test | Purpose |
|---|---|---|
| `internal/cli/flush_test.go` | `TestFlush_WritesProjectSlug` | End-to-end: flush with stub transcript writes memory with non-empty `project_slug` |
| `internal/cli/targets_test.go` | Update `TestFlushFlags` | Cover new `ProjectSlug` field |
| `internal/cli/migrate_slugs_test.go` | `TestMigrateSlugs_BackfillsEmptySlugs` | Backfill writes slug to empty memories |
| `internal/cli/migrate_slugs_test.go` | `TestMigrateSlugs_SkipsPopulated` | Populated slugs are not overwritten |
| `internal/cli/migrate_slugs_test.go` | `TestMigrateSlugs_DryRun` | Dry-run prints count without writing |
| `internal/cli/migrate_slugs_test.go` | `TestMigrateSlugsFlags_BuildsCorrectArgs` | Flag wiring |

## TDD Order

1. **Red** `TestFlush_WritesProjectSlug` — fails because `runFlush` doesn't pass slug.
2. **Green** — add flag, `applyProjectSlugDefault`, pass `--project-slug` in `learnArgs`.
3. **Red** `TestFlushFlags` extension — add `ProjectSlug` assertion.
4. **Green** — update `FlushArgs` and `FlushFlags`.
5. **Red** `TestMigrateSlugs_*` — new file, fails because command doesn't exist.
6. **Green** — implement `migrate-slugs` command.
7. **Refactor** — fix `runCorrect` slug defaulting in the same pass.
8. Run `targ check-full` to clear lint.

## Files to Touch

- `internal/cli/flush.go` — primary fix
- `internal/cli/flush_test.go` — new test
- `internal/cli/targets.go` — `FlushArgs` + `FlushFlags`
- `internal/cli/targets_test.go` — extend `TestFlushFlags`
- `internal/cli/cli.go` — fix `runCorrect` slug defaulting
- `internal/cli/migrate_slugs.go` — new file
- `internal/cli/migrate_slugs_test.go` — new file
