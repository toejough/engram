---
level: 4
name: tomlwriter
parent: "c3-engram-cli-binary.md"
children: []
last_reviewed_commit: cd55eab2
---

# C4 — tomlwriter (Property/Invariant Ledger)

> Component in focus: **E28 · tomlwriter** (refines L3 c3-engram-cli-binary.md).
> Source files in scope:
> - [internal/tomlwriter/tomlwriter.go](internal/tomlwriter/tomlwriter.go)
> - [internal/tomlwriter/tomlwriter_test.go](internal/tomlwriter/tomlwriter_test.go)

## Context (from L3)

tomlwriter is the TOML serialization component for new and updated feedback and fact memory files. The cli component (E21) calls Writer.Write on learn / remember / update flows; tomlwriter routes the record to the correct subdirectory by Type (fact / feedback / fallback memories), generates a slug-based filename with collision-avoiding numeric suffixes, stamps timestamps if missing, and writes the TOML atomically via temp-file + rename with cleanup on any failure. All filesystem syscalls (createTemp, rename, mkdirAll, stat, remove) are injected as function values defaulted to their os.* counterparts, so the package contains no direct os.* I/O calls outside the New constructor's defaults — wiring happens at the cli edge.

![C4 tomlwriter context diagram](svg/c4-tomlwriter.svg)

> Diagram source: [svg/c4-tomlwriter.mmd](svg/c4-tomlwriter.mmd). Re-render with
> `npx @mermaid-js/mermaid-cli -i architecture/c4/svg/c4-tomlwriter.mmd -o architecture/c4/svg/c4-tomlwriter.svg`.
> Pre-rendered because GitHub's Mermaid lacks the ELK layout engine, which is needed to
> separate bidirectional R/D edges between the same node pair.

**Legend:**
- Solid arrows: runtime calls (R-edges) from L3.
- Dotted arrow: DI back-edge (D3) — function-value adapters wired by cli at tomlwriter.New.

## Dependency Manifest

Each row is one injected dependency the focus component receives. Manifest expands the
Rdi back-edge into per-dep wiring rows. Reciprocal entries live in the wirer's L4 under
"DI Wires" — those two sections must stay in sync.

| Dep field | Type | Wired by | Concrete adapter | Properties |
|---|---|---|---|---|
| `createTemp` | `func(dir, pattern string) (*os.File, error)` | [E21 · cli](c3-engram-cli-binary.md#e21-cli) ([c4-cli.md](c4-cli.md)) | `os.CreateTemp` (default in `New`) | P1, P2, P5, P6 |
| `rename` | `func(oldpath, newpath string) error` | [E21 · cli](c3-engram-cli-binary.md#e21-cli) ([c4-cli.md](c4-cli.md)) | `os.Rename` (default in `New`) | P1, P2, P6 |
| `mkdirAll` | `func(path string, perm os.FileMode) error` | [E21 · cli](c3-engram-cli-binary.md#e21-cli) ([c4-cli.md](c4-cli.md)) | `os.MkdirAll` (default in `New`) | P3, P6 |
| `stat` | `func(name string) (os.FileInfo, error)` | [E21 · cli](c3-engram-cli-binary.md#e21-cli) ([c4-cli.md](c4-cli.md)) | `os.Stat` (default in `New`) | P4, P6 |
| `remove` | `func(name string) error` | [E21 · cli](c3-engram-cli-binary.md#e21-cli) ([c4-cli.md](c4-cli.md)) | `os.Remove` (default in `New`) | P2 |

## Property Ledger

| ID | Property | Statement | Enforced at | Tested at | Notes |
|---|---|---|---|---|---|
| <a id="p1-atomic-publish"></a>P1 | Atomic publish | For all records R and target paths P, a successful `AtomicWrite(P, R)` makes R's TOML appear at P in one rename step — readers never observe a partial file at P. | [internal/tomlwriter/tomlwriter.go:53](../../internal/tomlwriter/tomlwriter.go#L53), [:77](../../internal/tomlwriter/tomlwriter.go#L77) | [internal/tomlwriter/tomlwriter_test.go:41](../../internal/tomlwriter/tomlwriter_test.go#L41), [:350](../../internal/tomlwriter/tomlwriter_test.go#L350) | Temp file is created in the same directory as the target so rename is same-filesystem and atomic on POSIX. |
| <a id="p2-temp-cleanup-on-failure"></a>P2 | Temp cleanup on failure | For all records R and target paths P, if any step (encode, close, rename) of `AtomicWrite(P, R)` fails, the temp file is removed and no stray file is left in the target directory. | [internal/tomlwriter/tomlwriter.go:59](../../internal/tomlwriter/tomlwriter.go#L59), [:65](../../internal/tomlwriter/tomlwriter.go#L65), [:72](../../internal/tomlwriter/tomlwriter.go#L72), [:79](../../internal/tomlwriter/tomlwriter.go#L79) | [internal/tomlwriter/tomlwriter_test.go:19](../../internal/tomlwriter/tomlwriter_test.go#L19) | Encode-error path is exercised by injecting an unencodable value (chan int); close- and rename-error paths covered by DI mocks. |
| <a id="p3-target-directory-is-created"></a>P3 | Target directory is created | For all records R, slugs S, and data dirs D, `Write(R, S, D)` ensures the type-specific target directory (facts / feedback / memories) exists before writing, creating it with mode 0o750 if missing. | [internal/tomlwriter/tomlwriter.go:96](../../internal/tomlwriter/tomlwriter.go#L96), [:180](../../internal/tomlwriter/tomlwriter.go#L180) | [internal/tomlwriter/tomlwriter_test.go:69](../../internal/tomlwriter/tomlwriter_test.go#L69), [:390](../../internal/tomlwriter/tomlwriter_test.go#L390) | Routing by Type: "fact" → memory/facts, "feedback" → memory/feedback, otherwise → memories/. |
| <a id="p4-filename-collision-avoidance"></a>P4 | Filename collision avoidance | For all data dirs D and slugs S, repeated `Write` calls with the same slug produce distinct filenames: the first uses `<S>.toml`, subsequent calls use `<S>-2.toml`, `<S>-3.toml`, … picking the first numeric suffix not present in D. | [internal/tomlwriter/tomlwriter.go:130](../../internal/tomlwriter/tomlwriter.go#L130) | [internal/tomlwriter/tomlwriter_test.go:200](../../internal/tomlwriter/tomlwriter_test.go#L200), [:90](../../internal/tomlwriter/tomlwriter_test.go#L90) | Stat errors other than IsNotExist are propagated as `tomlwriter: stat …` rather than silently treated as free. |
| <a id="p5-slug-normalization"></a>P5 | Slug normalization | For all input summaries S drawn from `[A-Za-z0-9 \-_]*`, the resulting filename slug is non-empty, lowercase, hyphen-separated, and matches `^[a-z0-9]+(-[a-z0-9]+)*$`; an empty or all-separator input becomes the literal slug `memory`. | [internal/tomlwriter/tomlwriter.go:101](../../internal/tomlwriter/tomlwriter.go#L101), [:190](../../internal/tomlwriter/tomlwriter.go#L190) | [internal/tomlwriter/tomlwriter_test.go:246](../../internal/tomlwriter/tomlwriter_test.go#L246), [:270](../../internal/tomlwriter/tomlwriter_test.go#L270) | Property test exercises the alphabet `[A-Za-z0-9 \-_]`. Behaviour outside that alphabet (e.g., unicode) is not pinned by tests. |
| <a id="p6-timestamp-defaulting"></a>P6 | Timestamp defaulting | For all records R passed to `Write`, on return either R.CreatedAt and R.UpdatedAt are both already set (caller-supplied values preserved) or are populated with the current time formatted as RFC3339 UTC. | [internal/tomlwriter/tomlwriter.go:112](../../internal/tomlwriter/tomlwriter.go#L112), [:116](../../internal/tomlwriter/tomlwriter.go#L116) | [internal/tomlwriter/tomlwriter_test.go:153](../../internal/tomlwriter/tomlwriter_test.go#L153) | Caller-supplied timestamps are preserved (asserted in TestWrite_CreatesTomlFileWithAllFields). The default-population branch is observed indirectly via duplicate-suffix and atomic-write tests but not asserted as RFC3339-shaped. |
| <a id="p7-error-wrapping-with-context"></a>P7 | Error wrapping with context | For all I/O failure paths in `Write` and `AtomicWrite`, the returned error wraps the underlying error with a human-readable phase prefix (`creating temp file`, `encoding TOML`, `closing temp file`, `renaming temp file`, `tomlwriter: create memories dir`, `tomlwriter: stat …`). | [internal/tomlwriter/tomlwriter.go:55](../../internal/tomlwriter/tomlwriter.go#L55), [:67](../../internal/tomlwriter/tomlwriter.go#L67), [:74](../../internal/tomlwriter/tomlwriter.go#L74), [:81](../../internal/tomlwriter/tomlwriter.go#L81), [:98](../../internal/tomlwriter/tomlwriter.go#L98), [:140](../../internal/tomlwriter/tomlwriter.go#L140) | [internal/tomlwriter/tomlwriter_test.go:69](../../internal/tomlwriter/tomlwriter_test.go#L69), [:90](../../internal/tomlwriter/tomlwriter_test.go#L90), [:111](../../internal/tomlwriter/tomlwriter_test.go#L111), [:132](../../internal/tomlwriter/tomlwriter_test.go#L132) | Each error path is asserted via DI-injected failure and ContainSubstring on the wrapped message. |
| <a id="p8-all-i-o-is-injectable"></a>P8 | All I/O is injectable | For all functions in package tomlwriter, every filesystem syscall (CreateTemp, Rename, MkdirAll, Stat, Remove) is invoked through a Writer field, never via direct `os.*` call — defaults are wired only inside `New`. | [internal/tomlwriter/tomlwriter.go:22](../../internal/tomlwriter/tomlwriter.go#L22), [:31](../../internal/tomlwriter/tomlwriter.go#L31) | [internal/tomlwriter/tomlwriter_test.go:69](../../internal/tomlwriter/tomlwriter_test.go#L69), [:90](../../internal/tomlwriter/tomlwriter_test.go#L90), [:111](../../internal/tomlwriter/tomlwriter_test.go#L111), [:132](../../internal/tomlwriter/tomlwriter_test.go#L132) | Architectural property derived from project DI principle (CLAUDE.md). Validated indirectly: the With* options exercised by tests demonstrate every syscall is overridable. |
| <a id="p9-empty-slug-fallback"></a>P9 | Empty slug fallback | For all calls `Write(R, "", D)`, the empty slug is replaced by the literal `memory` before slug normalization, so the file becomes `memory.toml` (or `memory-N.toml` on collision). | [internal/tomlwriter/tomlwriter.go:101](../../internal/tomlwriter/tomlwriter.go#L101) | **⚠ UNTESTED** | Distinct from P5: P5 covers the slugify-empty-result fallback; P9 covers the caller-passed-empty-string short-circuit before slugify. |
| <a id="p10-returned-path-is-the-written-file"></a>P10 | Returned path is the written file | For all successful `Write(R, S, D)` calls, the returned path satisfies (a) `filepath.Base` ends with `.toml`, (b) the file exists at that path after return, and (c) the path is under the type-specific target directory. | [internal/tomlwriter/tomlwriter.go:107](../../internal/tomlwriter/tomlwriter.go#L107), [:125](../../internal/tomlwriter/tomlwriter.go#L125) | [internal/tomlwriter/tomlwriter_test.go:153](../../internal/tomlwriter/tomlwriter_test.go#L153), [:350](../../internal/tomlwriter/tomlwriter_test.go#L350) |   |
| <a id="p11-path-cleaning"></a>P11 | Path cleaning | For all target paths P passed to `AtomicWrite`, the rename destination and the temp directory are computed from `filepath.Clean(P)`, so trailing slashes, `.`, and `..` segments are normalized before any filesystem call. | [internal/tomlwriter/tomlwriter.go:50](../../internal/tomlwriter/tomlwriter.go#L50), [:51](../../internal/tomlwriter/tomlwriter.go#L51) | **⚠ UNTESTED** | Defensive normalization; no test asserts behaviour for unclean inputs. |

## Cross-links

- Parent: [c3-engram-cli-binary.md](c3-engram-cli-binary.md) (refines **E28 · tomlwriter**)
- Siblings:
  - [c4-anthropic.md](c4-anthropic.md)
  - [c4-cli.md](c4-cli.md)
  - [c4-context.md](c4-context.md)
  - [c4-externalsources.md](c4-externalsources.md)
  - [c4-main.md](c4-main.md)
  - [c4-memory.md](c4-memory.md)
  - [c4-recall.md](c4-recall.md)
  - [c4-tokenresolver.md](c4-tokenresolver.md)

See `skills/c4/references/property-ledger-format.md` for the full row format and untested-property
discipline.

