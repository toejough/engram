# Recall Output Fix Design

**Issue:** #365 (output format), #367 (wire memory surfacer)
**Date:** 2026-03-22

## Problem

1. `engram recall` outputs a single-line JSON blob up to 50KB. Agent tooling (Read, head) can't chunk single lines, so the agent can't consume the output without workarounds.
2. The `memories` field is always empty because `runRecall` wires `nil` for the `MemorySurfacer` interface. No real implementation exists.
3. The mode A budget (50KB) produces more content than an agent can reliably summarize.

## Design

### 1. Plain text output format

Replace `json.NewEncoder(stdout).Encode(result)` in `runRecall` with plain text:

```
<transcript line 1>
<transcript line 2>
...
=== MEMORIES ===
<memory 1>
<memory 2>
...
```

Transcript content comes first. If memories are non-empty, an `=== MEMORIES ===` separator on its own line precedes them. If no memories, the separator is omitted.

The `=== MEMORIES ===` separator avoids ambiguity with `---` which commonly appears in markdown and diff content within transcripts.

The `Result` struct is unchanged internally — only the CLI serialization changes. The JSON struct tags remain on `Result` since tests and potential future consumers may use JSON marshaling.

### 2. Mode A budget: 15KB

Add a new constant `DefaultModeABudget = 15 * 1024` for mode A (raw transcript). Update `recallModeA` to use `DefaultModeABudget` at both the `Read` call and the loop break condition (currently `DefaultStripBudget` in both places).

Mode B keeps using `DefaultStripBudget` (50KB) as its per-session read budget since Haiku filters it down to `DefaultExtractCap` (1500 bytes).

### 3. Wire memory surfacer

Build an adapter that implements `recall.MemorySurfacer`:

```go
type MemorySurfacer interface {
    Surface(query string) (string, error)
}
```

The adapter wraps the existing `surface.Surfacer` in prompt mode:
- Accepts a query string (transcript content for mode A, user query for mode B)
- Creates `surface.Options{Mode: "prompt", Message: query, DataDir: dataDir}`
- Calls `surfacer.Run(ctx, &buf, opts)` and returns the buffer contents
- DI: retriever, effectiveness computer, and other surface dependencies are injected at construction time in `runRecall`
- If query is empty (no transcript content accumulated), skip surfacing and return `""` — don't pass an empty message to the surface pipeline

**Context threading:** The `MemorySurfacer` interface takes no `context.Context`. The adapter closes over a `context.Context` provided at construction time. This is acceptable because the adapter is short-lived (created per `runRecall` invocation, not long-lived).

**Circular import avoidance:** The adapter in `internal/recall/` must not import `internal/surface/` directly. Define a `SurfaceRunner` interface in the adapter file:

```go
type SurfaceRunner interface {
    Run(ctx context.Context, w io.Writer, opts surface.Options) error
}
```

The CLI wires the concrete `surface.Surfacer` into this interface. However, `surface.Options` is a concrete type — if this creates an import, move the adapter to `internal/cli/` instead, keeping it at the wiring boundary where concrete imports are allowed.

Wire this adapter into `runRecall` instead of `nil`.

### 4. Skill update

Update `skills/recall/SKILL.md`:
- Output is plain text, not JSON
- Transcript content first, then `=== MEMORIES ===` separator, then memories
- Remove JSON parsing instructions

### Not changing

- `Result` struct fields or JSON tags
- Mode B extraction logic (`DefaultExtractCap` stays at 1500 bytes)
- `TranscriptReader` / `SessionFinder`
- `surfaceMemories()` method on Orchestrator (it already handles the nil check and string return)

## DI Boundaries

The adapter depends on `surface.Surfacer` via interface (`SurfaceRunner`). It does not import I/O packages directly. The CLI (`runRecall`) wires the real dependencies:
- `retrieve.New()` for memory retrieval
- `effectiveness.FromMemories()` for scoring
- `surface.New(retriever, opts...)` for the surfacer

Tests use a fake `MemorySurfacer` (already exists in `orchestrate_test.go`).

## Verification

After implementation, run `engram recall --data-dir ~/.claude/engram/data --project-slug <slug>` and confirm:
1. Output is multi-line plain text, not JSON
2. Transcript content appears first
3. When memories are present, `=== MEMORIES ===` separator appears followed by real memory content
4. When no memories match, no separator appears
5. Total output is ~15KB or less for mode A
6. Mode B still works with `--query` flag
