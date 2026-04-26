---
level: 4
name: tomlwriter
parent: "c3-engram-cli-binary.md"
children: []
last_reviewed_commit: 6002fa69
---

# C4 — tomlwriter (Property/Invariant Ledger)

> Component in focus: **E28 · tomlwriter** (refines L3 c3-engram-cli-binary).
> Source files in scope:
> - [../../internal/tomlwriter/tomlwriter.go](../../internal/tomlwriter/tomlwriter.go)
> - [../../internal/tomlwriter/tomlwriter_test.go](../../internal/tomlwriter/tomlwriter_test.go)

## Context (from L3)

Scoped slice of [c3-engram-cli-binary.md](c3-engram-cli-binary.md): the L3 edges that touch
E28. The DI back-edge convention applies — E28 → E21 represents the category of file-system
calls E28 makes through `Option`-injected functions. In production cli wires `tomlwriter.New()`
with no options (defaults to `os.*`); the option-based DI exists primarily for tests.

![C4 tomlwriter context diagram](svg/c4-tomlwriter.svg)

> Diagram source: [svg/c4-tomlwriter.mmd](svg/c4-tomlwriter.mmd). Re-render with
> `npx @mermaid-js/mermaid-cli -i architecture/c4/svg/c4-tomlwriter.mmd -o architecture/c4/svg/c4-tomlwriter.svg`.
> Pre-rendered because GitHub's Mermaid lacks the ELK layout engine, which is needed to
> separate bidirectional R/D edges between the same node pair.

## Dependency Manifest

| Dep field | Type | Wired by | Concrete adapter | Properties |
|---|---|---|---|---|
| `createTemp` | `func(dir, pattern string) (*os.File, error)` | self-default in `New` | `os.CreateTemp` (override via `WithCreateTemp`) | P3, P5 |
| `rename` | `func(oldpath, newpath string) error` | self-default in `New` | `os.Rename` (override via `WithRename`) | P3, P5 |
| `mkdirAll` | `func(path string, perm os.FileMode) error` | self-default in `New` | `os.MkdirAll` (override via `WithMkdirAll`) | P4 |
| `stat` | `func(name string) (os.FileInfo, error)` | self-default in `New` | `os.Stat` (override via `WithStat`) | P6, P7 |
| `remove` | `func(name string) error` | self-default in `New` | `os.Remove` (override via `WithRemove`) | P5 |

Note: cli calls `tomlwriter.New()` with no options at [internal/cli/learn.go:323](../../internal/cli/learn.go#L323) and
[internal/cli/update.go:77](../../internal/cli/update.go#L77), so production wiring is the `os.*`
defaults set inside `New`. Tests override via the `With*` options.

## Property Ledger

| ID | Property | Statement | Enforced at | Tested at | Notes |
|---|---|---|---|---|---|
| <a id="p1-slugify-lowercase"></a>P1 | Slug is lowercase hyphenated | For all input strings, `Slugify(s)` returns a string containing only `[a-z0-9-]`, with no leading or trailing `-`. | [internal/tomlwriter/tomlwriter.go:190](../../internal/tomlwriter/tomlwriter.go#L190) | [internal/tomlwriter/tomlwriter_test.go:246](../../internal/tomlwriter/tomlwriter_test.go#L246) | Runs of non-alphanumerics collapse to single `-`. |
| <a id="p2-slug-empty"></a>P2 | Empty slug fallback | For all inputs that would produce a zero-length slug after normalization, `Slugify` returns `"memory"`. | [internal/tomlwriter/tomlwriter.go:195](../../internal/tomlwriter/tomlwriter.go#L195) | **⚠ UNTESTED** | Default applied at [tomlwriter.go:101](../../internal/tomlwriter/tomlwriter.go#L101) too; no direct test. |
| <a id="p3-atomic-rename"></a>P3 | Atomic temp+rename | For all successful `AtomicWrite(path, record)` calls, the target file is created via `createTemp`-then-`rename` (never partial writes visible at `path`). | [internal/tomlwriter/tomlwriter.go:53](../../internal/tomlwriter/tomlwriter.go#L53) | [internal/tomlwriter/tomlwriter_test.go:41](../../internal/tomlwriter/tomlwriter_test.go#L41), [:350](../../internal/tomlwriter/tomlwriter_test.go#L350) | Tests assert the rename happened and bytes match. |
| <a id="p4-mkdir-target"></a>P4 | Target dir created | For all `Write(record, slug, dataDir)` calls, the target directory (`facts/`, `feedback/`, or `memories/`) is created via `mkdirAll` with mode `0o750` before any file is written. | [internal/tomlwriter/tomlwriter.go:96](../../internal/tomlwriter/tomlwriter.go#L96) | [internal/tomlwriter/tomlwriter_test.go:69](../../internal/tomlwriter/tomlwriter_test.go#L69), [:390](../../internal/tomlwriter/tomlwriter_test.go#L390) | Permission constant `memoriesDirPerm`. |
| <a id="p5-temp-cleanup-on-failure"></a>P5 | Temp file cleaned on failure | For all `AtomicWrite` calls where TOML encoding, file close, or rename fails, the temp file is removed via `remove` and an error wrapping the cause is returned. | [internal/tomlwriter/tomlwriter.go:65](../../internal/tomlwriter/tomlwriter.go#L65) | [internal/tomlwriter/tomlwriter_test.go:19](../../internal/tomlwriter/tomlwriter_test.go#L19), [:132](../../internal/tomlwriter/tomlwriter_test.go#L132) | Encode-error and rename-error paths covered. |
| <a id="p6-suffix-on-collision"></a>P6 | Numeric suffix on collision | For all `Write` calls where `<slug>.toml` already exists, the writer probes `<slug>-2.toml`, `<slug>-3.toml`, ..., returning the first that does not exist. | [internal/tomlwriter/tomlwriter.go:130](../../internal/tomlwriter/tomlwriter.go#L130) | [internal/tomlwriter/tomlwriter_test.go:200](../../internal/tomlwriter/tomlwriter_test.go#L200) | Loop terminates on first `os.IsNotExist`. |
| <a id="p7-stat-error-propagated"></a>P7 | Stat errors propagated | For all `availablePath` probes where `stat` returns a non-IsNotExist error, `Write` returns an error wrapping that stat error. | [internal/tomlwriter/tomlwriter.go:139](../../internal/tomlwriter/tomlwriter.go#L139) | [internal/tomlwriter/tomlwriter_test.go:90](../../internal/tomlwriter/tomlwriter_test.go#L90) | Distinguishes "doesn't exist" from "can't read directory". |
| <a id="p8-type-routing"></a>P8 | Type-based routing | For all `Write(record, slug, dataDir)` calls: `record.Type == "fact"` → `memory.FactsDir(dataDir)`; `"feedback"` → `memory.FeedbackDir`; otherwise → `memory.MemoriesDir`. | [internal/tomlwriter/tomlwriter.go:203](../../internal/tomlwriter/tomlwriter.go#L203) | [internal/tomlwriter/tomlwriter_test.go:153](../../internal/tomlwriter/tomlwriter_test.go#L153), [learn_test.go:279](../../internal/cli/learn_test.go#L279) | Verified via cli integration too. |
| <a id="p9-timestamps-defaulted"></a>P9 | CreatedAt/UpdatedAt defaulted | For all `Write` calls where `record.CreatedAt`/`record.UpdatedAt` are empty, the writer sets them to `time.Now().UTC()` formatted as RFC3339 before writing. | [internal/tomlwriter/tomlwriter.go:112](../../internal/tomlwriter/tomlwriter.go#L112) | [internal/tomlwriter/tomlwriter_test.go:153](../../internal/tomlwriter/tomlwriter_test.go#L153) | Pre-existing values are preserved. |
| <a id="p10-write-returns-final-path"></a>P10 | Write returns final absolute path | For all successful `Write` calls, the returned string equals the final file path (with any collision suffix applied). | [internal/tomlwriter/tomlwriter.go:125](../../internal/tomlwriter/tomlwriter.go#L125) | [internal/tomlwriter/tomlwriter_test.go:200](../../internal/tomlwriter/tomlwriter_test.go#L200) | Used by cli to print `CREATED: <name>`. |
| <a id="p11-empty-slug-default"></a>P11 | Empty slug → "memory" | For all `Write` calls with `slug == ""`, the slug used for the filename is `"memory"`. | [internal/tomlwriter/tomlwriter.go:101](../../internal/tomlwriter/tomlwriter.go#L101) | **⚠ UNTESTED** | Subset of P2 but applied at the `Write` boundary. |

## Cross-links

- Parent: [c3-engram-cli-binary.md](c3-engram-cli-binary.md) (refines **E28 · tomlwriter**)

See `skills/c4/references/property-ledger-format.md` for the full row format and untested-property
discipline.
