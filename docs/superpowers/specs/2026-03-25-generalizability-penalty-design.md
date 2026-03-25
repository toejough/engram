# Generalizability Penalty for Cross-Project Surfacing (#373, sub-feature 1)

## Problem

Narrow memories (generalizability 1–2) surface across all projects equally. A memory created for a specific project clutters surfacing in unrelated projects. Generalizability and project slug are already written to TOML at extraction time but not read back at surfacing time.

## Design

### Penalty placement

The penalty applies to the BM25 relevance component only. Spreading activation (evidence from actual usage) and quality (effectiveness, recency, frequency) are unaffected.

Current: `(bm25 + alpha×spreading) × (1 + quality)`
New: `(bm25 × genFactor + alpha×spreading) × (1 + quality)`

### Penalty curve

Same-project memories receive no penalty regardless of generalizability. Cross-project:

| Generalizability | Factor | Meaning |
|-----------------|--------|---------|
| 5 | 1.0 | Universal — no penalty |
| 4 | 0.8 | Similar projects — small penalty |
| 3 | 0.5 | Moderate |
| 2 | 0.2 | Narrow — large penalty |
| 1 | 0.05 | This-project-only — near-zero |
| 0 (unset) | 0.5 | Missing data — conservative default |

### Changes by file

**`internal/memory/memory.go`** — Add to `Stored` struct:
```go
Generalizability int
ProjectSlug      string
```

**`internal/retrieve/retrieve.go`** — Wire new fields in `parseMemoryFile()`:
```go
Generalizability: record.Generalizability,
ProjectSlug:      record.ProjectSlug,
```

**`internal/surface/surface.go`** — Add `CurrentProjectSlug string` to `Options` struct. Compute `genFactor` for each memory and pass it to `CombinedScore()`.

**`internal/frecency/frecency.go`** — Change `CombinedScore` signature to accept a generalizability factor:
```go
func (s *Scorer) CombinedScore(relevance, spreading float64, genFactor float64, input Input) float64 {
    return (relevance*genFactor + s.alpha*spreading) * (1.0 + s.Quality(input))
}
```

**`internal/surface/surface.go`** — New helper:
```go
func genFactor(gen int, memProject, currentProject string) float64
```
Returns 1.0 if same project or if either slug is empty. Otherwise looks up the penalty from the curve table.

**CLI wiring** (`internal/cli/cli.go`) — Derive `CurrentProjectSlug` from the data directory path. The standard data dir is `~/.claude/engram/data/<project-slug>/memories/`. Extract the project slug as the parent directory name of the `memories/` subdirectory, or the leaf directory name of `--data-dir`. No hook script changes needed — the Go binary derives it from the existing `--data-dir` flag.

**Hook scripts** — No changes. Hooks already pass `--data-dir` which encodes the project context.

**Sort functions** (`sortPromptMatchesByActivation`, `sortToolMatchesByActivation`) — Must compute `genFactor` per memory in the sort closure using `mem.Generalizability`, `mem.ProjectSlug`, and `opts.CurrentProjectSlug`.

### Edge cases

- **Missing generalizability (0):** Treat as moderate (0.5) — conservative, doesn't hide potentially useful memories.
- **Missing project slug (empty):** No penalty. Can't determine cross-project.
- **Same project:** Always 1.0, regardless of generalizability score.

## Not in scope

- Consolidation signal (sub-feature 2 of #373)
- Modifying the generalizability scoring at extraction time
- Adding generalizability to the `frecency.Input` struct (it's a scoring modifier, not a quality signal)
