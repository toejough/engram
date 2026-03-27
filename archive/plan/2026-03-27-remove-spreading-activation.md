# Plan: Remove Spreading Activation (#392, closes #390)

## Context
Playground experiment showed alpha=0 is optimal — spreading activation adds no value. The graph link builder was never wired (#390), so spreading has been inert in production.

## Tasks

All tasks are interdependent (shared compilation unit) — execute sequentially in a single worktree.

### Task 1: Delete `internal/graph/` package
- Delete `internal/graph/graph.go`, `recompute.go`, `graph_test.go`, `recompute_test.go`

### Task 2: Remove spreading/link code from `internal/surface/`
- `surface.go`: Remove `LinkGraphLink` struct, `LinkReader` interface, `linkReader` field from `Surfacer`, `WithLinkReader()` option, `computeSpreading()` function, spreading activation block in `runPrompt()` (~lines 243-286), `spreadingScore` field from `promptMatch`
- Remove spreading score from `CombinedScore` call site (~line 721-722)
- `export_test.go`: Remove `ExportComputeSpreading` export
- Delete `spreading_test.go` and `p3_test.go` entirely

### Task 3: Remove alpha from `internal/frecency/`
- `frecency.go`: Remove `alpha` field, `defaultAlpha` constant, `Alpha()` getter, `WithAlpha()` option, `alpha*spreading` from `CombinedScore()` — simplify to `bm25 * genFactor * (1 + quality)`
- `frecency_test.go`: Remove `TestAlpha_DefaultIsZero`, `TestWithAlpha_SetsCustomValue`, update alpha comments in other tests

### Task 4: Remove link recomputer from `internal/learn/`
- `learn.go`: Remove `graph` import, `linkRecomputer` field, `SetLinkRecomputer()` method, `LinkRecomputer` interface, link recomputation block in `processMerge()` (~lines 345-359)
- Delete `link_recompute_test.go`

### Task 5: Remove link recomputer from `internal/signal/`
- `consolidate.go`: Remove `linkRecomputer` field, `LinkRecomputer` interface, `WithLinkRecomputer()` option

### Task 6: Remove `LinkRecord` from `internal/memory/`
- `record.go`: Remove `LinkRecord` struct, `Links []LinkRecord` field from `MemoryRecord`

### Task 7: Remove `[[links]]` from memory TOML files
- Strip `[[links]]` sections from all 27 memory TOML files

### Task 8: Verify
- `targ test` passes
- `targ check-full` passes (or only pre-existing failures)
- No references to `spreading`, `LinkReader`, `graph.Builder`, `alpha` (scoring context), or `[[links]]` remain in code

## Acceptance Criteria (from issue)
- No code references `spreading`, `LinkReader`, `graph.Builder`, or link-related types
- `CombinedScore` formula simplified to `BM25 * GenFactor * (1 + Quality)`
- `internal/graph/` deleted
- `[[links]]` removed from all memory TOML files
- All tests pass
