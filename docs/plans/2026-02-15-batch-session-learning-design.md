# Batch Session Learning

## Problem

Session extraction only runs on Stop/PreCompact hooks, meaning months of old sessions contain unextracted learnings. We need a way to retroactively process old sessions and iterate on extraction quality.

## CLI Interface

```
projctl memory learn-sessions [--days N] [--last N] [--min-size 8KB] [--dry-run]
projctl memory learn-sessions --reset-last N
```

| Flag | Default | Description |
|------|---------|-------------|
| `--days N` | 7 | Process sessions modified within last N days |
| `--last N` | - | Process N most recent unevaluated sessions (overrides --days) |
| `--min-size` | 8KB | Skip transcripts smaller than threshold |
| `--dry-run` | false | Show what would be processed without extracting |
| `--reset-last N` | - | Clear evaluation flags for last N processed sessions |

All projects under `~/.claude/projects/` are scanned. Project name is derived from directory path via `DeriveProjectName`.

## Session Discovery

1. Scan `~/.claude/projects/*/*.jsonl` recursively
2. Skip subagent transcripts (`*/subagents/*.jsonl`)
3. Sort by modification time (most recent first)
4. Filter by `--days` or `--last`
5. Check `processed_sessions` table, skip already-evaluated

## Evaluation Tracking

New SQLite table in `embeddings.db`:

```sql
CREATE TABLE IF NOT EXISTS processed_sessions (
    session_id   TEXT PRIMARY KEY,
    project      TEXT NOT NULL,
    processed_at TEXT NOT NULL,
    items_found  INTEGER NOT NULL DEFAULT 0,
    status       TEXT NOT NULL DEFAULT 'success'
)
```

Status values: `success`, `partial` (some items failed), `timeout` (60s limit hit).

Reset: `DELETE FROM processed_sessions ORDER BY processed_at DESC LIMIT N`

## Processing Flow

```
1. Discover sessions
2. Filter by recency
3. Exclude already-processed (via processed_sessions table)
4. Print: "Found 47 unevaluated sessions across 5 projects (~34MB)"
5. For each session (sequential, most recent first):
   a. "[3/47] Extracting abc123 (projctl)..."
   b. ExtractSession(opts) with 60s timeout
   c. Record in processed_sessions
   d. "  -> 4 learnings extracted"
6. "Processed 47 sessions, extracted 128 learnings"
```

## Error Handling

- **API failures**: Session marked `partial`, continue to next
- **Timeout**: Session marked `timeout`, continue to next
- **Ctrl-C**: Graceful — recorded progress is preserved
- **Retry**: Use `--reset-last N` then re-run

## Code Organization

| File | Purpose |
|------|---------|
| `cmd/projctl/memory_learn_sessions.go` | CLI command, discovery, progress |
| `internal/memory/embeddings.go` | Add `processed_sessions` table to schema |

Reuses: `ExtractSession()`, `Learn()`, `DeriveProjectName()`, `NewLLMExtractor()`, `NewMemoryStoreSemanticMatcher()`

## Existing Hook Behavior

The Stop/PreCompact hooks continue to work as before. `extract-session` will also record to `processed_sessions` so that `learn-sessions` skips sessions that were already processed by hooks.
