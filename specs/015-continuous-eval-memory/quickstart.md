# Quickstart: Continuous Evaluation Memory Pipeline

**Branch**: `015-continuous-eval-memory`

## Build & Test

```bash
# Build with FTS5 support
targ install-projctl
# or directly:
go install -tags sqlite_fts5 ./cmd/projctl

# Run tests
go test -tags sqlite_fts5 ./internal/memory/...

# Run specific test file
go test -tags sqlite_fts5 -run TestFilter ./internal/memory/
```

## Key Files to Modify

| Component | File | What Changes |
|-----------|------|-------------|
| Schema | `internal/memory/embeddings.go` | Add surfacing_events table, new embeddings columns |
| LLM interface | `internal/memory/llm_api.go` | Add `Filter()` method to LLMExtractor interface + DirectAPIExtractor |
| Hook pipeline | `internal/memory/format.go` | New `TierFiltered` output tier using Filter() instead of Curate() |
| Hook input | `internal/memory/hook_input.go` | No changes — HookInput already has all needed fields |
| Hook definitions | `internal/memory/hooks.go` | Update Stop/PreCompact commands to include scoring |
| Core API | `internal/memory/memory.go` | Add ScoreSession, ClassifyQuadrants, DiagnoseLeech functions |
| CLI | `cmd/projctl/main.go` | Add `/memory helpful|wrong|unclear` subcommand |

## New Files to Create

| File | Purpose |
|------|---------|
| `internal/memory/surfacing.go` | SurfacingEvent CRUD operations |
| `internal/memory/surfacing_test.go` | Tests for surfacing operations |
| `internal/memory/scoring.go` | Impact scoring, quadrant classification, auto-tuning |
| `internal/memory/scoring_test.go` | Tests for scoring |
| `internal/memory/leech.go` | Leech diagnosis engine |
| `internal/memory/leech_test.go` | Tests for leech diagnosis |
| `internal/memory/claudemd_quality.go` | Quality gate, scoring, section parser |
| `internal/memory/claudemd_quality_test.go` | Tests for CLAUDE.md quality |
| `internal/memory/filter.go` | Haiku filter implementation |
| `internal/memory/filter_test.go` | Tests for filter |

## Implementation Order

```
Phase 1: Foundation (P1 + P6 partial)
├── surfacing_events table + schema migration
├── embeddings new columns
├── Filter() method on LLMExtractor
├── Surfacing event CRUD
└── Hook pipeline integration (replace Curate with Filter)

Phase 2: Scoring (P2 + P6)
├── Impact scoring (ScoreSession)
├── Quadrant classification
├── Auto-tune thresholds
├── End-of-session hook integration (Stop, PreCompact)
└── Spreading activation in Query

Phase 3: Diagnosis (P3)
├── Leech detection (GetLeeches)
├── Leech diagnosis engine
├── Proposal presentation
└── Action execution

Phase 4: Quality Gate (P4)
├── CLAUDE.md section parser
├── Quality gate checks
├── Promotion proposals
├── ScoreClaudeMD()
└── Budget enforcement

Phase 5: Feedback (P5)
├── /memory helpful|wrong|unclear CLI
├── Feedback recording
└── Impact score adjustment
```

## Testing Strategy

- **Unit tests**: Each new file gets a `_test.go` companion.
- **Integration tests**: `_integration_test.go` files for DB-dependent tests using `InitDBForTest()`.
- **Assertion framework**: `gomega` (existing convention).
- **Test data**: Synthetic surfacing events with known outcomes.
- **LLM mocking**: Use the existing pattern of interface-based dependency injection.

## Environment Requirements

- Go 1.25+ with `sqlite_fts5` build tag
- Anthropic API key (via keychain for DirectAPIExtractor)
- ONNX Runtime 1.23.2 (for E5 embeddings, already installed)
