# Stop Hook Memory Surfacing (#382)

## Problem

The Stop hook only runs `engram flush` (learning pipeline). It doesn't surface memories based on what the agent just said. This means engram can't react to agent claims like "this is pre-existing" with relevant memories reminding the agent that pre-existing issues are not acceptable.

## Design

### New surface mode: `stop`

Add `--mode stop` to `engram surface` that:
1. Reads transcript delta since last stop-hook offset
2. Strips JSONL to text, filters to assistant-only lines
3. Surfaces memories matching that text via the existing prompt-mode BM25 pipeline
4. Tracks its own offset (separate from learn-offset)

### Pipeline

```
transcript JSONL → delta reader (offset tracking) → Strip → filter ASSISTANT: lines → join as query → runPrompt (existing BM25 pipeline) → Result
```

### New CLI flags for stop mode

- `--transcript-path`: Path to session transcript JSONL
- `--session-id`: Current session ID (for offset reset on new session)

These already exist as concepts in the learn pipeline. The surface command adds them alongside the existing `--mode`, `--data-dir`, `--format` flags.

### Offset tracking

Reuse the existing `learn.OffsetStore` interface and `learn.Offset` struct. Store at `<dataDir>/stop-surface-offset.json` (separate from `learn-offset.json` so the two pipelines track independently).

### Assistant-only filtering

After `context.Strip()` returns `["USER: ...", "ASSISTANT: ...", ...]`, filter to only lines starting with `"ASSISTANT: "` and strip the prefix. This keeps the surface query focused on what the agent said, not what the user asked.

### Hook changes

Split the Stop hook into two entries in `hooks.json`:
1. **Sync entry** (new): Runs `engram surface --mode stop`, returns surfaced memories as `additionalContext`
2. **Async entry** (existing): Runs `engram flush` as before

The sync entry runs first (appears first in the array), surfaces memories, then the async entry runs fire-and-forget for learning.

### Integration with runSurface

In `internal/cli/cli.go`'s `runSurface`, add handling for `mode == "stop"`:
1. Read offset from `stop-surface-offset.json`
2. Read transcript delta
3. Strip and filter to assistant lines
4. If no assistant text, return empty result and update offset
5. Otherwise, call `surfacer.Run()` with `ModePrompt` and the assistant text as the message
6. Update offset

This reuses the entire existing prompt-mode surfacing pipeline (BM25, spreading activation, suppression, etc.) — the stop mode is just a different way of constructing the query text.

### Surface package changes

Add `ModeStop = "stop"` constant. In `Surfacer.Run()`, the stop mode is handled at the CLI layer (extracting transcript delta) and delegates to `runPrompt` internally, so no changes needed in `surface.go` itself. The CLI constructs the message and passes `ModePrompt` to the surfacer.

Actually, to keep it clean: add `ModeStop` to the surface package and handle it in `Run()` by calling `runPrompt` with the pre-constructed message. This way the CLI just passes options through and the surface package owns the mode dispatch.

Wait — the delta reading is I/O (file reads, offset tracking). Per the DI principle, that shouldn't happen inside the surface package. Better approach:

- **CLI layer** (`runSurface`): Reads transcript delta, strips, filters, joins → produces a `message` string
- **Surface layer**: Receives `ModeStop` but internally delegates to `runPrompt` with the message

This means `Options` needs a new field for the stop-mode message, or we reuse the existing `Message` field and just set `Mode = ModePrompt` after extracting. Simplest: CLI does the extraction, sets `opts.Message = assistantText` and `opts.Mode = ModePrompt`, and the surface package doesn't need to know about stop mode at all.

**Final approach**: No surface package changes. The CLI's `runSurface` handles `--mode stop` by:
1. Reading transcript delta and extracting assistant text
2. Setting mode to `prompt` and message to the extracted text
3. Calling the existing surfacer with prompt mode

This is the simplest possible design with maximum reuse.

## Files changed

| File | Change |
|------|--------|
| `internal/cli/cli.go` | Add stop-mode handling in `runSurface`: delta read, strip, filter, delegate to prompt mode |
| `internal/cli/targets.go` | Add `TranscriptPath` and `SessionID` flags to `SurfaceArgs` |
| `hooks/hooks.json` | Split Stop into sync (surface) + async (flush) entries |
| `hooks/stop.sh` | Extract surfacing logic into new `stop-surface.sh`; keep flush-only |
| `hooks/stop-surface.sh` | New sync hook: read transcript, call `engram surface --mode stop`, output memories |

## Not in scope

- New surface mode constant in `internal/surface/` (not needed — CLI translates to prompt mode)
- Changes to `context.Strip()` or `context.DeltaReader` (reused as-is)
- Changes to the BM25/surfacing pipeline (reused via prompt mode)
