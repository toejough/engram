---
level: 4
name: memory
parent: "c3-engram-cli-binary.md"
children: []
last_reviewed_commit: 6002fa69
---

# C4 — memory (Property/Invariant Ledger)

> Component in focus: **E24 · memory** (refines L3 c3-engram-cli-binary).
> Source files in scope:
> - [../../internal/memory/memory.go](../../internal/memory/memory.go)
> - [../../internal/memory/record.go](../../internal/memory/record.go)
> - [../../internal/memory/readmodifywrite.go](../../internal/memory/readmodifywrite.go)
> - [../../internal/memory/memory_test.go](../../internal/memory/memory_test.go)
> - [../../internal/memory/record_test.go](../../internal/memory/record_test.go)
> - [../../internal/memory/readmodifywrite_test.go](../../internal/memory/readmodifywrite_test.go)
> - [../../internal/memory/maintenance_test.go](../../internal/memory/maintenance_test.go)

## Context (from L3)

Scoped slice of [c3-engram-cli-binary.md](c3-engram-cli-binary.md): R4 (cli reads/writes
feedback + fact TOML through memory), R9 (recall lists memories during ranking), and R16
(memory reads existing TOML on disk). The DI back-edge convention applies: `Lister` and
`Modifier` consume readDir/readFile/AtomicWriter dependencies wired by E21 cli.

![C4 memory context diagram](svg/c4-memory.svg)

> Diagram source: [svg/c4-memory.mmd](svg/c4-memory.mmd). Re-render with
> `npx @mermaid-js/mermaid-cli -i architecture/c4/svg/c4-memory.mmd -o architecture/c4/svg/c4-memory.svg`.
> Pre-rendered because GitHub's Mermaid lacks the ELK layout engine, which is needed to
> separate bidirectional R/D edges between the same node pair.

## Dependency Manifest

| Dep field | Type | Wired by | Concrete adapter | Properties |
|---|---|---|---|---|
| `Lister.readDir` | `func(string) ([]os.DirEntry, error)` | [E21 · cli](c3-engram-cli-binary.md#e21-cli) (L4: c4-cli.md — TBD) | `os.ReadDir` (default in `NewLister`) | P5, P6, P7, P11 |
| `Lister.readFile` | `func(string) ([]byte, error)` | [E21 · cli](c3-engram-cli-binary.md#e21-cli) (L4: c4-cli.md — TBD) | `os.ReadFile` (default in `NewLister`) | P5, P6, P11 |
| `Modifier.readFile` | `func(string) ([]byte, error)` | [E21 · cli](c3-engram-cli-binary.md#e21-cli) (L4: c4-cli.md — TBD) | `os.ReadFile` (default in `NewModifier`) | P8, P9, P11 |
| `Modifier.writer` | `memory.AtomicWriter` (`AtomicWrite(path, record any) error`) | [E21 · cli](c3-engram-cli-binary.md#e21-cli) (L4: c4-cli.md — TBD) | `tomlwriter.Writer` (must be supplied via `WithModifierWriter`; nil panics on first use) | P8, P9, P10, P11 |

## Property Ledger

| ID | Property | Statement | Enforced at | Tested at | Notes |
|---|---|---|---|---|---|
| <a id="p1-facts-dir-path"></a>P1 | FactsDir path convention | For all `dataDir`, `FactsDir(dataDir) == filepath.Join(dataDir, "memory", "facts")`. | [internal/memory/memory.go:21](../../internal/memory/memory.go#L21) | [internal/memory/memory_test.go:11](../../internal/memory/memory_test.go#L11) | — |
| <a id="p2-feedback-dir-path"></a>P2 | FeedbackDir path convention | For all `dataDir`, `FeedbackDir(dataDir) == filepath.Join(dataDir, "memory", "feedback")`. | [internal/memory/memory.go:26](../../internal/memory/memory.go#L26) | [internal/memory/memory_test.go:17](../../internal/memory/memory_test.go#L17) | — |
| <a id="p3-resolve-precedence"></a>P3 | Resolve precedence: feedback → facts → legacy | For all `(dataDir, slug)` where the file exists in multiple candidate locations, `ResolveMemoryPath` returns feedback first, then facts, then legacy `memories/`. | [internal/memory/memory.go:44](../../internal/memory/memory.go#L44) | [internal/memory/memory_test.go:23](../../internal/memory/memory_test.go#L23), [:36](../../internal/memory/memory_test.go#L36), [:50](../../internal/memory/memory_test.go#L50) | — |
| <a id="p4-resolve-fallback-legacy"></a>P4 | Resolve fallback to legacy when missing | For all `(dataDir, slug)` where no candidate file exists, `ResolveMemoryPath` returns the legacy `memories/<slug>.toml` path so the caller gets a meaningful "not found" error downstream. | [internal/memory/memory.go:61](../../internal/memory/memory.go#L61) | [internal/memory/memory_test.go:63](../../internal/memory/memory_test.go#L63) | — |
| <a id="p5-listall-skips-non-toml"></a>P5 | ListAll skips non-TOML and subdirs | For all directory contents, `Lister.ListAll` skips entries that are subdirectories or whose names do not end in `.toml`, and silently skips files that fail to read or decode. | [internal/memory/readmodifywrite.go:48](../../internal/memory/readmodifywrite.go#L48), [:55](../../internal/memory/readmodifywrite.go#L55), [:62](../../internal/memory/readmodifywrite.go#L62) | [internal/memory/readmodifywrite_test.go:65](../../internal/memory/readmodifywrite_test.go#L65), [:95](../../internal/memory/readmodifywrite_test.go#L95) | Best-effort listing; lone bad files don't kill the read. |
| <a id="p6-liststored-sort-desc"></a>P6 | ListStored sorts by UpdatedAt desc | For all results, `Lister.ListStored` returns `Stored` records ordered by `UpdatedAt` descending. | [internal/memory/readmodifywrite.go:98](../../internal/memory/readmodifywrite.go#L98) | [internal/memory/readmodifywrite_test.go:346](../../internal/memory/readmodifywrite_test.go#L346) | Same ordering applied by `ListAllMemories` after merge. |
| <a id="p7-listallmemories-routing"></a>P7 | ListAllMemories layout routing | For all `dataDir`, `Lister.ListAllMemories` returns the union of `memory/feedback/` + `memory/facts/` when feedback contains any `.toml` file; otherwise returns the legacy `memories/` directory. | [internal/memory/readmodifywrite.go:76](../../internal/memory/readmodifywrite.go#L76), [:106](../../internal/memory/readmodifywrite.go#L106) | [internal/memory/readmodifywrite_test.go:135](../../internal/memory/readmodifywrite_test.go#L135), [:183](../../internal/memory/readmodifywrite_test.go#L183), [:229](../../internal/memory/readmodifywrite_test.go#L229) | "New layout" detection requires at least one `.toml` in feedback. |
| <a id="p8-readmodifywrite-atomic"></a>P8 | ReadModifyWrite always writes through AtomicWriter | For all successful decodes, `Modifier.ReadModifyWrite` calls `mutate(&record)` then exactly one `writer.AtomicWrite(path, record)`; on read or decode error, it returns wrapped error and does not call the writer. | [internal/memory/readmodifywrite.go:172](../../internal/memory/readmodifywrite.go#L172) | [internal/memory/readmodifywrite_test.go:649](../../internal/memory/readmodifywrite_test.go#L649), [:779](../../internal/memory/readmodifywrite_test.go#L779), [:720](../../internal/memory/readmodifywrite_test.go#L720) | Decode/read errors short-circuit before the write. |
| <a id="p9-rmw-preserves-fields"></a>P9 | ReadModifyWrite preserves all fields | For all TOML records on disk and all mutations that touch only a subset of fields, the bytes written by `Modifier` retain every other field present in the source record (including `schema_version`, `source`, content sub-fields, and timestamps). | [internal/memory/readmodifywrite.go:178](../../internal/memory/readmodifywrite.go#L178) | [internal/memory/readmodifywrite_test.go:794](../../internal/memory/readmodifywrite_test.go#L794) | Single canonical `MemoryRecord` struct prevents the field-loss class of bugs (#353). |
| <a id="p10-record-roundtrip"></a>P10 | MemoryRecord round-trips | For all valid `MemoryRecord` values, encoding to TOML and decoding back yields a structurally equal record (modulo zero-value omitempty fields). | [internal/memory/record.go:26](../../internal/memory/record.go#L26) | [internal/memory/record_test.go:13](../../internal/memory/record_test.go#L13), [:52](../../internal/memory/record_test.go#L52), [:97](../../internal/memory/record_test.go#L97), [:140](../../internal/memory/record_test.go#L140) | Covers fact content, feedback content, and `schema_version`. |
| <a id="p11-default-io-via-options"></a>P11 | I/O wired via constructor options only | For all callers, `NewLister` and `NewModifier` accept I/O dependencies exclusively via `ListerOption` / `ModifierOption` functional options; no exported field on `Lister` or `Modifier` is mutable from outside the package. | [internal/memory/readmodifywrite.go:26](../../internal/memory/readmodifywrite.go#L26), [:159](../../internal/memory/readmodifywrite.go#L159) | [internal/memory/readmodifywrite_test.go:388](../../internal/memory/readmodifywrite_test.go#L388), [:422](../../internal/memory/readmodifywrite_test.go#L422), [:528](../../internal/memory/readmodifywrite_test.go#L528) | Default constructors fall back to real `os` functions when no option is supplied; `Modifier.writer` defaults to nil and panics if used unwired. |
| <a id="p12-tostored-malformed-time"></a>P12 | ToStored tolerates malformed timestamps | For all `MemoryRecord` values whose `UpdatedAt` is a non-empty unparseable string, `ToStored` logs a warning to stderr and returns a `Stored` with zero-value `UpdatedAt`. | [internal/memory/record.go:40](../../internal/memory/record.go#L40) | [internal/memory/memory_test.go:73](../../internal/memory/memory_test.go#L73), [:88](../../internal/memory/memory_test.go#L88), [:103](../../internal/memory/memory_test.go#L103) | Allows recovery from a single malformed file in a directory of memories. |

## Cross-links

- Parent: [c3-engram-cli-binary.md](c3-engram-cli-binary.md) (refines **E24 · memory**)
- Siblings:
  - [c4-main.md](c4-main.md)
  - [c4-context.md](c4-context.md)
  - [c4-tokenresolver.md](c4-tokenresolver.md)

See `skills/c4/references/property-ledger-format.md` for the full row format and untested-property
discipline.
