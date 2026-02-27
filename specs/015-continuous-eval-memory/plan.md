# Implementation Plan: Continuous Evaluation Memory Pipeline

**Branch**: `015-continuous-eval-memory` | **Date**: 2026-02-20 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/015-continuous-eval-memory/spec.md`
**Design Doc**: [design](../../docs/plans/2026-02-20-continuous-evaluation-memory-design.md)

## Summary

Replace the memory system's mechanical count-based promotion pipeline with a continuous inline evaluation loop. The system will filter surfaced memories for relevance (Haiku), track whether surfaced memories actually help (impact scoring), diagnose systematically broken memories (leech detection), and gate all CLAUDE.md changes on measured effectiveness with user approval. Scoring runs automatically at session end rather than requiring manual `optimize` runs.

## Technical Context

**Language/Version**: Go 1.25+ (module: `github.com/toejough/projctl`)
**Primary Dependencies**: go-sqlite3, sqlite-vec, onnxruntime_go (E5-small-v2), gomega (testing)
**Storage**: SQLite (embeddings.db) — WAL mode, busy_timeout=5000ms
**Testing**: `go test -tags sqlite_fts5 ./internal/memory/...`, gomega assertions
**Target Platform**: macOS (Darwin), CLI tool
**Project Type**: Single Go project — `internal/memory/` package contains all memory system code
**Performance Goals**: Haiku filter <200ms per interaction; Sonnet synthesis <1s when triggered; end-of-session scoring <5s
**Constraints**: Daily API cost <$0.15 for ~100 interactions; no background processes (CLI sessions are ephemeral)
**Scale/Scope**: Single user, ~100 interactions/day, ~500 memories in DB

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Constitution is an unpopulated template — no project-specific gates defined. Proceeding with standard engineering practices: TDD, incremental delivery, no breaking changes to existing interfaces.

## Project Structure

### Documentation (this feature)

```text
specs/015-continuous-eval-memory/
├── plan.md              # This file
├── research.md          # Phase 0 output — 8 research decisions
├── data-model.md        # Phase 1 output — schema, entities, state transitions
├── quickstart.md        # Phase 1 output — build, test, file map
├── contracts/           # Phase 1 output — 5 contract files
│   ├── filter.md        # Haiku filter interface
│   ├── surfacing-events.md  # Surfacing event CRUD
│   ├── scoring.md       # Impact scoring & quadrants
│   ├── leech-diagnosis.md   # Leech root cause analysis
│   ├── claude-md-quality.md # Quality gate & scoring
│   └── user-feedback.md     # /memory feedback commands
└── tasks.md             # Phase 2 output (via /speckit.tasks)
```

### Source Code (repository root)

```text
internal/memory/
├── embeddings.go        # MODIFY: Add surfacing_events table + new columns
├── llm_api.go           # MODIFY: Add Filter() to LLMExtractor + DirectAPIExtractor
├── format.go            # MODIFY: New TierFiltered output path
├── hooks.go             # MODIFY: Update Stop/PreCompact to include scoring
├── memory.go            # MODIFY: Add scoring/diagnosis entry points
├── hook_input.go        # UNCHANGED
├── surfacing.go         # NEW: SurfacingEvent CRUD operations
├── surfacing_test.go    # NEW: Tests
├── filter.go            # NEW: Haiku filter implementation
├── filter_test.go       # NEW: Tests
├── scoring.go           # NEW: Impact scoring, quadrants, auto-tune
├── scoring_test.go      # NEW: Tests
├── leech.go             # NEW: Leech diagnosis engine
├── leech_test.go        # NEW: Tests
├── claudemd_quality.go  # NEW: Quality gate, scoring, section parser
└── claudemd_quality_test.go # NEW: Tests

cmd/projctl/main.go      # MODIFY: Add memory feedback subcommand
```

**Structure Decision**: All new code goes into the existing `internal/memory/` package. This feature extends the memory system — it doesn't warrant a new package. New functionality is split across focused files (surfacing, filter, scoring, leech, claudemd_quality) rather than appending to the already-large `memory.go` (1534 lines).

## Artifacts Generated

| Artifact | Status | Description |
|----------|--------|-------------|
| [research.md](research.md) | Complete | 8 research decisions covering integration points, data model, and algorithms |
| [data-model.md](data-model.md) | Complete | Schema for surfacing_events, embeddings columns, computed scores, entity definitions |
| [quickstart.md](quickstart.md) | Complete | Build commands, file map, implementation order, testing strategy |
| [contracts/filter.md](contracts/filter.md) | Complete | Filter() interface, error handling, degradation strategy |
| [contracts/surfacing-events.md](contracts/surfacing-events.md) | Complete | CRUD operations for surfacing event lifecycle |
| [contracts/scoring.md](contracts/scoring.md) | Complete | Impact scoring, quadrant classification, auto-tuning, sampling strategy |
| [contracts/leech-diagnosis.md](contracts/leech-diagnosis.md) | Complete | Root cause analysis with 4 diagnosis categories |
| [contracts/claude-md-quality.md](contracts/claude-md-quality.md) | Complete | Quality gate, scoring, section parser, budget enforcement |
| [contracts/user-feedback.md](contracts/user-feedback.md) | Complete | /memory feedback CLI commands |

## Key Design Decisions

1. **Filter() replaces Curate() in hook pipeline** — new method returns structured tags + scores needed for surfacing_events. Curate() remains for backwards compat. (Research R1, R6)

2. **Haiku predicts synthesis value per-candidate** — the filter prompt includes a `should_synthesize` field for each candidate (no blanket "always synthesize multiple memories" rule). The caller triggers Synthesize() when 2+ candidates are flagged. (Research R2)

3. **ALTER TABLE migration pattern** — same fire-and-forget pattern used for all 16+ existing column additions. No formal migration system. (Research R3)

4. **End-of-session scoring (Option C)** — runs in Stop/PreCompact hooks. No background workers. Bounded per-event evaluation. (Research R4)

5. **Additive activation formula** — `B_i + α × effectiveness`, not multiplicative. Avoids zeroing out activation for negative effectiveness. (Research R5)

6. **Distribution-tracking auto-tune** — exponential moving average, capped adjustments per run, prevents oscillation. (Research R7)

7. **Pattern-matching leech diagnosis** — 4 categories detected from surfacing_events data signatures, no additional API calls. (Research R8)
