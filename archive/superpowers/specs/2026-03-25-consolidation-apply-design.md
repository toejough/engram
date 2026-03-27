# Consolidation Apply Handler (#373, sub-feature 2 completion)

## Problem

`engram maintain` emits consolidation proposals and the triage hook renders them, but there's no `apply-proposal --action consolidate` handler. Users can see consolidation candidates but can't act on them through the CLI. The building blocks exist (LLM extractor, counter transfer, archiver) but aren't wired into the apply dispatcher.

## Design

### Store member paths at emit time

Currently `cli.go` stores members as `[]string` (titles) in the `consolidateDetails` struct. The apply handler needs **paths** to load memories. Change `Members` from `[]string` to `[]consolidateMember`:

```go
type consolidateMember struct {
    Path  string `json:"path"`
    Title string `json:"title"`
}

type consolidateDetails struct {
    Members        []consolidateMember `json:"members"`
    SharedKeywords []string            `json:"shared_keywords"`
    Confidence     float64             `json:"confidence"`
}
```

This is a JSON schema change (`members` goes from `["title1", "title2"]` to `[{"path":"...", "title":"..."}]`). The triage hook does NOT read `.details.members` — it reads `.memory_path` and `.diagnosis` from the top-level proposal object — so this change doesn't break rendering. No other consumers parse this field.

### Add consolidate action to Applier

Add to `signal/apply.go`:

- `actionConsolidate = "consolidate"` constant
- `case actionConsolidate:` in `Apply()` switch
- `applyConsolidate(ctx, action)` handler

The handler:
1. Parses `action.Fields["members"]` to get member paths
2. Loads each member as `*memory.MemoryRecord` via injected `loadRecord` function
3. Builds a `ConfirmedCluster{Members: loadedMembers}`
4. Calls `Extractor.ExtractPrinciple(ctx, cluster)` to synthesize generalized memory
5. Calls `TransferFields(consolidated, loadedMembers, time.Now())`
6. Writes consolidated memory to `action.Memory` path (the survivor path from the proposal)
7. Archives original member files (excluding the survivor, which gets overwritten)

### New Applier dependencies

The `Applier` struct needs three new injected dependencies for consolidation:

```go
type Applier struct {
    // ... existing fields ...
    extractor   Extractor                                    // LLM principle extraction
    archiver    Archiver                                     // file archival
    loadRecord  func(string) (*memory.MemoryRecord, error)   // load MemoryRecord from TOML path
}
```

With corresponding `With*` option functions. These are only needed when `action == "consolidate"` — nil is fine for other actions.

### Load MemoryRecord from path

No `memory.ReadRecord(path) (*MemoryRecord, error)` function exists. The existing `readStoredMemory` in `signal.go` returns `*memory.Stored`, which is the wrong type — `ConfirmedCluster.Members` requires `[]*memory.MemoryRecord`.

Add a thin `readRecord` helper (in `signal.go` or `memory/` package) that reads a TOML file into `*memory.MemoryRecord`. This is essentially the read half of `ReadModifyWrite` without the write-back. Always overwrite `record.SourcePath = path` after `toml.Decode` — the on-disk value may be stale or absent.

### Wire in CLI

In `runApplyProposal` (`signal.go`), when action is `"consolidate"`:

- Read API token from env (same pattern as `RunMaintain`'s `makeAnthropicCaller`)
- Construct a `LLMExtractor` via `NewLLMExtractor` from `llm_confirm.go` (needs the LLM caller)
- Construct a `FileArchiver` via `NewFileArchiver(archiveDir, os.Rename, os.MkdirAll)` — archive dir is `dataDir/archive/` per existing test convention
- Construct the `loadRecord` function
- Pass all three via `With*` options to `NewApplier`

### Apply() becomes context-aware

`applyConsolidate` needs `context.Context` for the LLM call. The current `Apply` signature already accepts `context.Context` but names it `_`. Rename to `ctx` and pass to `applyConsolidate`. Other `apply*` methods don't need ctx and keep their current signatures — only `applyConsolidate` receives it.

### Memory-triage skill update

Add `consolidate` to the documented CLI commands:

```
- **Consolidate**: `engram apply-proposal --action consolidate --memory <survivor-path> --fields '{"members":[{"path":"..."},...]}'`
```

## Files changed

| File | Change |
|------|--------|
| `internal/cli/cli.go` | Change `consolidateDetails.Members` from `[]string` to `[]consolidateMember` with path+title; update emit logic |
| `internal/signal/apply.go` | Add `actionConsolidate`, `applyConsolidate()`, new `With*` options, use ctx |
| `internal/cli/signal.go` | Add `readRecord` helper; wire extractor, archiver, loadRecord for consolidate action |
| `skills/memory-triage/memory-triage.md` | Document consolidate CLI command |

## Not in scope

- Rejection/suppression mechanism (not needed — proposals re-surface until data changes)
- Changes to `ExtractPrinciple`, `TransferFields`, or `FileArchiver` (existing code, used as-is)
- Changes to triage hook rendering (already renders consolidation proposals)
- Changes to `Plan()` or cluster detection
