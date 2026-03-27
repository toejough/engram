# Architecture

## Purpose

Engram is self-correcting memory for LLM agents. It measures impact, not just frequency -- memories that don't improve outcomes get diagnosed and fixed.

## Pipeline Overview

```
Extract --> Deduplicate --> Write --> Surface --> Evaluate --> Maintain
```

1. **Extract**: Parse session transcripts for learnings (corrections, patterns, instructions).
2. **Deduplicate**: BM25 + TF-IDF similarity against existing memories to prevent redundancy.
3. **Write**: Persist as TOML files with structured fields, tier classification, and provenance.
4. **Surface**: Retrieve relevant memories at prompt submission, tool use, and session stop. Rank by keyword match, generalizability, quality score, and frecency.
5. **Evaluate**: Track whether surfaced memories were followed, contradicted, ignored, or irrelevant. Outcome signals are recorded directly in each memory's TOML file.
6. **Maintain**: Diagnose effectiveness quadrants (working/leech/hidden-gem/noise), propose fixes (rewrite, broaden keywords, escalate, remove, consolidate).

## Key Architecture Decisions

**DI everywhere.** No function in `internal/` calls `os.*`, `http.*`, `sql.Open`, or any I/O directly. All I/O flows through injected interfaces. Wiring happens at the CLI edges (`internal/cli`).

**Pure Go, no CGO.** TF-IDF and BM25 for text similarity instead of ONNX embeddings. External API calls for LLM classification only.

**Plugin form factor.** Engram is a Claude Code plugin. Shell hooks handle event dispatch; a Go binary handles computation; skills provide interactive workflows.

**Fire-and-forget error handling.** Hooks never fail the caller. All hook scripts exit 0 even on internal errors. Errors are logged to stderr or debug logs, never propagated to Claude Code.

**Filesystem as registry.** Memory ID = file path. One `rm` = complete removal. No database, no index files to sync. TOML files are the source of truth.

**Effectiveness as first-class data.** Outcome counters (followed, contradicted, ignored, irrelevant) are embedded in each memory file. Maintenance decisions are driven by these metrics, not by age or frequency alone.

## Package Map

### Core Pipeline

| Package | Purpose |
|---------|---------|
| `extract` | Parse correction signals from user messages |
| `learn` | Incremental learning from session transcripts, offset tracking, merge-on-write |
| `retrieve` | BM25-based memory retrieval with keyword matching |
| `surface` | Surfacing orchestration with budget, policy, generalizability factor, suppression |
| `recall` | Cross-session context recall with summarization |

### Memory and Data

| Package | Purpose |
|---------|---------|
| `memory` | Canonical types: MemoryRecord, Stored, Enriched, CandidateLearning, read-modify-write |
| `context` | Session context tracking with delta computation |
| `correct` | Correction signal detection and memory creation |
| `creationlog` | Append-only JSONL log of memory creation events |
| `surfacinglog` | JSONL log of surfacing events for outcome tracking |

### Signal and Maintenance

| Package | Purpose |
|---------|---------|
| `signal` | Consolidation detection (keyword overlap + TF-IDF), surface signal analysis, LLM confirmation |
| `adapt` | Adaptive policy analysis -- proposes system-level changes from feedback patterns |
| `maintain` | Maintenance proposals: quadrant diagnosis, apply actions, purge tier-C |
| `review` | Effectiveness review with budget tracking and threshold analysis |
| `effectiveness` | Aggregate effectiveness scoring from memory counters |

### Classification and Ranking

| Package | Purpose |
|---------|---------|
| `classify` | Unified LLM-based tier classification (A/B/C) with structured field extraction |
| `dedup` | BM25 + TF-IDF deduplication against existing corpus |
| `bm25` | BM25 scoring implementation |
| `tfidf` | TF-IDF vectorization and cosine similarity |
| `keyword` | Keyword extraction and matching |
| `frecency` | Quality-weighted scoring: effectiveness, frequency, tier boost |

### Policy and Config

| Package | Purpose |
|---------|---------|
| `policy` | Adaptive policy types, lifecycle (proposed/approved/active/retired), TOML persistence |
| `tomlwriter` | TOML serialization for memory files |
| `tokenresolver` | API token resolution for LLM calls |

### I/O and Wiring

| Package | Purpose |
|---------|---------|
| `cli` | CLI subcommands, flag parsing, I/O wiring via `targ` |
| `hooks` | Shell scripts for Claude Code hook events |
| `render` | Output formatting for surfaced memories |
| `track` | Surfacing outcome tracking (followed/contradicted/ignored) |
| `transcript` | Session transcript JSONL parsing |
| `crossref` | Cross-reference extraction from transcripts |
| `merge` | Memory merge operations for consolidation |
| `instruct` | Instruction quality audit |
| `contradict` | Contradiction detection between memories |

## Data Model

Memories are individual TOML files stored in `~/.claude/engram/data/memories/`. Each file contains content fields (title, content, concepts, keywords, principle, anti-pattern), tracking counters (surfaced, followed, contradicted, ignored, irrelevant), provenance (source type, content hash), and maintenance history. See [data-model.md](data-model.md) for the full schema.

## Plugin Integration

Engram registers three hook events in `hooks/hooks.json`:

| Event | Hook | Behavior |
|-------|------|----------|
| `UserPromptSubmit` | `user-prompt-submit.sh` | Surface relevant memories, detect corrections, consume pending maintenance |
| `SessionStart` | `session-start.sh` | Emit recall reminder (sync), run maintain + build in background (async) |
| `Stop` | `stop-surface.sh` | Surface memories based on agent output, block if relevant |
| `Stop` | `stop.sh` (async) | Run flush pipeline: learn from transcript, record outcomes |

All hooks auto-build the Go binary if missing or stale (source newer than binary).
