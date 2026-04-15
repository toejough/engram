# Migrate engram CLI to targ for arg/command parsing

## Problem

Engram has two parallel arg-parsing systems: manual `flag.FlagSet` in each command function, and targ struct-based arg definitions in `targets.go` that convert back to `[]string` and re-invoke `RunSafe()`. This double-parse bridge is unnecessary complexity.

## Decision Summary

| Question | Decision |
|----------|----------|
| Entry point | `targ.Main()` — targ handles dispatch, help, error formatting |
| Learn subcommands | `targ.Group("learn", feedback, fact)` |
| Build system integration | Keep `Targets()` wired to real functions, delete flag-string bridge |
| I/O wiring | Keep inside each command function, context from targ |

## Architecture

### Entry point

`main.go` calls `targ.Main(cli.Targets(os.Stdout, os.Stderr, os.Stdin)...)`. Targ handles subcommand routing, help text, and error-to-stderr formatting.

### Arg structs

Existing (keep):
- `RecallArgs` — data-dir, project-slug, query, memories-only, limit
- `ShowArgs` — name (becomes positional), data-dir
- `ListArgs` — data-dir

New:
- `CommonLearnArgs` — embedded struct: situation, source, data-dir, no-dup-check
- `LearnFeedbackArgs` — embeds CommonLearnArgs + behavior, impact, action
- `LearnFactArgs` — embeds CommonLearnArgs + subject, predicate, object
- `UpdateArgs` — name (required), data-dir, situation, behavior, impact, action, subject, predicate, object, source

### Command functions

Each changes from `func(args []string, stdout io.Writer) error` to accepting a context and typed args:

```go
func runRecall(ctx context.Context, args RecallArgs, stdout io.Writer) error
func runShow(ctx context.Context, args ShowArgs, stdout io.Writer) error
func runList(ctx context.Context, args ListArgs, stdout io.Writer) error
func runLearnFeedback(ctx context.Context, args LearnFeedbackArgs, stdout io.Writer) error
func runLearnFact(ctx context.Context, args LearnFactArgs, stdout io.Writer) error
func runUpdate(ctx context.Context, args UpdateArgs, stdout io.Writer) error
```

I/O wiring (token resolution, adapter creation, etc.) stays inside each function. Context comes from targ (signal-aware).

### Target wiring

`Targets(stdout, stderr io.Writer, stdin io.Reader) []any` builds closures:

```go
targ.Targ(func(ctx context.Context, a RecallArgs) error {
    return runRecall(ctx, a, stdout)
}).Name("recall").Description("Recall recent session context")
```

Learn is a group:

```go
targ.Group("learn",
    targ.Targ(func(ctx context.Context, a LearnFeedbackArgs) error {
        return runLearnFeedback(ctx, a, stdout)
    }).Name("feedback").Description("Learn from behavioral feedback"),
    targ.Targ(func(ctx context.Context, a LearnFactArgs) error {
        return runLearnFact(ctx, a, stdout)
    }).Name("fact").Description("Learn a factual statement"),
)
```

### Show slug handling

Currently show accepts slug as positional arg or `--name` flag with manual extraction logic (`extractSlug`, `resolveSlug`). Simplified to a single required positional arg in `ShowArgs`:

```go
type ShowArgs struct {
    Name    string `targ:"positional,required,desc=memory slug to display"`
    DataDir string `targ:"flag,name=data-dir,env=ENGRAM_DATA_DIR,desc=path to data directory"`
}
```

## Deletions

### Functions
- `Run()` — replaced by targ dispatch
- `RunSafe()` — replaced by `targ.Main()` error handling
- `newFlagSet()` — no more flag.FlagSet
- `BuildFlags()`, `AddBoolFlag()`, `AddIntFlag()` — bridge helpers
- `RecallFlags()`, `ShowFlags()`, `ListFlags()` — bridge converters
- `BuildTargets()` — replaced by direct-wired `Targets()`
- `registerCommonFlags()` — replaced by embedded targ struct
- `extractSlug()`, `resolveSlug()` — replaced by targ positional arg
- `signalContext()` — targ provides signal-aware context
- `parseAndValidate()` — targ handles parsing; source validation moves inline

### Types
- `learnCommonFlags` — replaced by `CommonLearnArgs` with targ tags
- `updateFlags` — replaced by `UpdateArgs` with targ tags

### Variables
- `errUsage`, `errUnknownCommand`, `errLearnUsage`, `errUnknownLearnSubcmd` — targ handles dispatch errors
- `minArgs` — targ handles arg count validation

### Files
- `signal.go` — targ provides context

## What stays

- All business logic (conflict detection, memory writing, recall orchestration, rendering)
- I/O adapters (`osDirLister`, `osFileReader`, `haikuCallerAdapter`)
- Helper functions (`applyDataDirDefault`, `applyProjectSlugDefault`, `resolveToken`, `makeAnthropicCaller`, `newAnthropicClient`, `newSummarizer`, `newTokenResolver`)
- Public utilities (`DataDirFromHome`, `ProjectSlugFromPath`)
- `fileExists`, `loadMemoryTOML`, all render functions
- `export_test.go` entries for business logic helpers (conflict detection, rendering, etc.)

## Testing

- Tests calling `cli.Run([]string{...})` switch to `targ.Execute([]string{...}, cli.Targets(...)...)`
- Flag-bridge tests deleted (TestBuildFlags, TestAddBoolFlag, TestAddIntFlag, TestRecallFlags, TestShowFlags, TestListFlags, TestBuildTargets)
- `ExportRunLearn` deleted — tests exercise through `targ.Execute`
- New tests for learn and update arg struct wiring through `targ.Execute`
- Existing integration tests (TestRunRecall, TestRunSafe) adapted to new entry point
