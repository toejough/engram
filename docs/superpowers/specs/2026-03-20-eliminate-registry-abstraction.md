# Eliminate Registry Abstraction Design Spec (#354)

## Goal

Remove `internal/registry/` and `internal/register/` packages. All memory metadata lives in TOML files via `memory.MemoryRecord` with a shared `ReadModifyWrite` helper. No separate abstraction layer.

## Problem

The `internal/registry/` package is a competing read/write interface over the same memory TOML files that every other package already reads/writes. It has its own `memoryRecord` struct (a 7th divergent definition after #353 unified the other 6), 8 adapter types across 6 consumers, and a `Registry` interface with 10 methods — all for what amounts to TOML field updates.

## Design

### 1. Expand `memory.MemoryRecord`

Add fields currently only accessible through the registry's `memoryRecord`:

```go
// Provenance.
SourceType  string `toml:"source_type,omitempty"`
SourcePath  string `toml:"source_path,omitempty"`
ContentHash string `toml:"content_hash,omitempty"`

// Enforcement escalation.
EnforcementLevel string             `toml:"enforcement_level,omitempty"`
Transitions      []TransitionRecord `toml:"transitions,omitempty"`

// Relationships.
Links    []LinkRecord     `toml:"links,omitempty"`
Absorbed []AbsorbedRecord `toml:"absorbed,omitempty"`
```

Supporting nested types in `internal/memory/`:

```go
type LinkRecord struct {
    Target           string  `toml:"target"`
    Weight           float64 `toml:"weight"`
    Basis            string  `toml:"basis"`
    CoSurfacingCount int     `toml:"co_surfacing_count,omitempty"`
}

type AbsorbedRecord struct {
    From          string             `toml:"from"`
    SurfacedCount int                `toml:"surfaced_count"`
    Evaluations   EvaluationCounters `toml:"evaluations"`
    ContentHash   string             `toml:"content_hash"`
    MergedAt      string             `toml:"merged_at"`
}

// EvaluationCounters preserves the nested [evaluations] table in absorbed records.
type EvaluationCounters struct {
    Followed     int `toml:"followed"`
    Contradicted int `toml:"contradicted"`
    Ignored      int `toml:"ignored"`
}

type TransitionRecord struct {
    From   string `toml:"from"`
    To     string `toml:"to"`
    At     string `toml:"at"`
    Reason string `toml:"reason"`
}
```

### 2. Shared `ReadModifyWrite` helper

```go
// ReadModifyWrite atomically reads a memory TOML, applies a mutation, and writes back.
// Uses temp file + rename for atomic writes. Preserves all fields through the round-trip.
func ReadModifyWrite(path string, mutate func(*MemoryRecord)) error
```

This replaces every Registry method. Each caller provides its mutation:

- `RecordSurfacing` → `func(r) { r.SurfacedCount++; r.LastSurfacedAt = now }`
- `RecordEvaluation` → `func(r) { r.FollowedCount++ }` (or Contradicted/Ignored)
- `SetEnforcementLevel` → `func(r) { r.EnforcementLevel = level; r.Transitions = append(...) }`
- `UpdateLinks` → `func(r) { r.Links = newLinks }`
- `Register` → `tomlwriter.Write` (already exists) + optional `ReadModifyWrite` for provenance

Additional helper:

```go
// ListAll reads all memory TOML files from a directory, returning parsed MemoryRecords with paths.
func ListAll(memoriesDir string) ([]StoredRecord, error)

type StoredRecord struct {
    Path   string
    Record MemoryRecord
}
```

### 3. Consumer rewiring

| Consumer | Registry method | Replacement |
|----------|----------------|-------------|
| Learn (register) | `Register(entry)` | `tomlwriter.Write` + `memory.ReadModifyWrite` for provenance |
| Learn (absorb) | `RegistryAbsorber.RecordAbsorbed(...)` | `memory.ReadModifyWrite(targetPath, func(r) { r.Absorbed = append(r.Absorbed, ...) })` |
| Surface | `RecordSurfacing(id)` | `memory.ReadModifyWrite(path, incrSurfaced)` |
| Surface (links) | `GetEntryLinks`/`SetEntryLinks` | `memory.ReadModifyWrite(path, setLinks)` |
| Evaluate | `RecordEvaluation(id, outcome)` | `memory.ReadModifyWrite(path, incrOutcome)` |
| Maintain | `Remove(id)` | `os.Remove(path)` |
| Signal | `Remove(id)`, `SetEnforcementLevel(...)`, `UpdateContentHash(...)` | `os.Remove(path)`, `memory.ReadModifyWrite(path, setEnforcement)`, `memory.ReadModifyWrite(path, setHash)` |
| Graph | `List()`, `UpdateLinks(id, links)` | `memory.ListAll(dir)`, `memory.ReadModifyWrite(path, setLinks)` |
| CrossRef | `Register(entry)` for CLAUDE.md/rules/skills | **Deleted** — external instruction tracking dropped |

### 4. DI boundaries

Each consumer's local interface changes:

- **Learn**: `RegistryRegistrar` / `RegistryAbsorber` → deleted. Learn calls `memory.ReadModifyWrite` via injected `func(string, func(*MemoryRecord)) error`.
- **Surface**: `RegistryRecorder` → `SurfacingRecorder func(path string) error`. `LinkReader`/`LinkUpdater` → read/write via `MemoryRecord`.
- **Evaluate**: `RegistryRecorder` → `EvaluationRecorder func(path, outcome string) error`.
- **Maintain**: `RegistryUpdater` → `FileRemover func(path string) error`.
- **Signal**: adapters replaced with injected `func(string, func(*MemoryRecord)) error` (same DI pattern as Learn — no direct `ReadModifyWrite` calls from `internal/signal/`).
- **Graph**: `RegistryLinker` → `MemoryLister` + `LinkWriter` interfaces backed by `memory.ListAll` + `ReadModifyWrite`.

### 5. Packages deleted

- `internal/registry/` — 5 files (entry.go, registry.go, signals.go, classify.go, toml_directory_store.go) + tests
- `internal/register/` — 1 file (register.go) + tests

### 6. Packages modified

- `internal/memory/` — add `ReadModifyWrite`, `ListAll`, expanded `MemoryRecord`, nested types
- `internal/cli/cli.go` — remove all registry adapter types (~10 adapters) and wiring
- `internal/cli/signal.go` — remove registry adapters
- `internal/learn/learn.go` — remove registry interfaces
- `internal/surface/surface.go` — replace registry interfaces with simpler DI
- `internal/evaluate/evaluator.go` — replace registry interface
- `internal/maintain/apply.go` — replace registry interface
- `internal/graph/recompute.go` — replace registry interface
- `internal/crossref/extract.go` — remove registry entry creation

### 7. What's preserved

- **Memory TOML file format** — unchanged, all fields already on disk
- **Surfacing pipeline behavior** — same logic, simpler plumbing
- **Link computation and spreading activation** — same algorithms, reads/writes via `MemoryRecord`
- **Co-surfacing link updates** — same behavior, different I/O path
- **Feedback counters** — same fields, reliably round-tripped (#353)
- **Enforcement escalation** — same levels and transitions, stored in TOML
- **Merge/absorption tracking** — same `absorbed` records, stored in TOML

### 8. What's dropped

- **External instruction tracking** — CLAUDE.md lines, rules, and skills no longer registered as entries. They're loaded every session anyway.
- **Registry.Merge() method** — already dead (CLI scaffolding removed in #354 first commit). Internal merge logic in signal consolidation operates directly on TOMLs.
- **Registry.Get() method** — never called by any consumer.
- **The entire Registry interface** — replaced by `ReadModifyWrite` + `ListAll`.

## Risks

- **Large surface area** — touches 8+ packages. Mitigated by doing it consumer-by-consumer, each independently testable.
- **Subtle behavior differences** — registry had in-memory caching via `sync.Map`. Direct TOML reads are disk I/O per call. Acceptable for hook-driven usage (not hot path).
- **Link format compatibility** — existing TOMLs already have links in the registry's format. `MemoryRecord`'s new `LinkRecord` type must match the existing TOML schema exactly.

## Success criteria

1. `internal/registry/` and `internal/register/` are deleted
2. All existing tests pass (or are updated to reflect the new plumbing)
3. `targ check-full` passes
4. Feedback counters still persist across surfacing cycles (the #353 fix is preserved)
5. `engram surface` still uses spreading activation via links
6. `engram show` displays all fields including enforcement level, links, absorbed
