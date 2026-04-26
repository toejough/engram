---
level: 4
name: externalsources
parent: "c3-engram-cli-binary.md"
children: []
last_reviewed_commit: cd55eab2
---

# C4 — externalsources (Property/Invariant Ledger)

> Component in focus: **E25 · externalsources** (refines L3 c3-engram-cli-binary.md).
> Source files in scope:
> - [internal/externalsources/automemory.go](internal/externalsources/automemory.go)
> - [internal/externalsources/cache.go](internal/externalsources/cache.go)
> - [internal/externalsources/claudemd.go](internal/externalsources/claudemd.go)
> - [internal/externalsources/discover.go](internal/externalsources/discover.go)
> - [internal/externalsources/externalsources.go](internal/externalsources/externalsources.go)
> - [internal/externalsources/frontmatter.go](internal/externalsources/frontmatter.go)
> - [internal/externalsources/imports.go](internal/externalsources/imports.go)
> - [internal/externalsources/rules.go](internal/externalsources/rules.go)
> - [internal/externalsources/skills.go](internal/externalsources/skills.go)
> - [internal/externalsources/slug.go](internal/externalsources/slug.go)

## Context (from L3)

Scoped slice of [c3-engram-cli-binary.md](c3-engram-cli-binary.md): the L3 edges that touch E25. The DI back-edge (D4) decomposes into the per-dependency Dependency Manifest below — each injected function is wired by E21 · cli at the `externalsources.Discover` call site.

E25 reads four kinds of files (CLAUDE.md hierarchy, .claude/rules, auto memory, skill SKILL.md) plus transitive @import expansions. All filesystem effects flow through injected adapters; the package itself never calls `os.*`, `runtime.*`, or `filepath.WalkDir` directly.

![C4 externalsources context diagram](svg/c4-externalsources.svg)

> Diagram source: [svg/c4-externalsources.mmd](svg/c4-externalsources.mmd). Re-render with
> `npx @mermaid-js/mermaid-cli -i architecture/c4/svg/c4-externalsources.mmd -o architecture/c4/svg/c4-externalsources.svg`.
> Pre-rendered because GitHub's Mermaid lacks the ELK layout engine, which is needed to
> separate bidirectional R/D edges between the same node pair.

## Dependency Manifest

Each row is one injected dependency the focus component receives. Manifest expands the
Rdi back-edge into per-dep wiring rows. Reciprocal entries live in the wirer's L4 under
"DI Wires" — those two sections must stay in sync.

| Dep field | Type | Wired by | Concrete adapter | Properties |
|---|---|---|---|---|
| `StatFn` | `func(path string) (bool, error)` | [E21 · cli](c3-engram-cli-binary.md#e21-cli) ([c4-cli.md](c4-cli.md)) | `osStatExists` wrapping `os.Stat` | P1–P3 |
| `Reader` | `func(path string) ([]byte, error)` | [E21 · cli](c3-engram-cli-binary.md#e21-cli) ([c4-cli.md](c4-cli.md)) | `FileCache.Read` over `os.ReadFile` | P5–P8, P10 |
| `MdWalker` | `func(root string) []string` | [E21 · cli](c3-engram-cli-binary.md#e21-cli) ([c4-cli.md](c4-cli.md)) | `osWalkMd` wrapping `filepath.WalkDir` | P10 |
| `MatchAny` | `func(globs []string) bool` | [E21 · cli](c3-engram-cli-binary.md#e21-cli) ([c4-cli.md](c4-cli.md)) | `osMatchAny` wrapping `filepath.Glob` | P10 |
| `Settings` | `func() (dir string, found bool)` | [E21 · cli](c3-engram-cli-binary.md#e21-cli) ([c4-cli.md](c4-cli.md)) | `readAutoMemoryDirectorySetting` reading settings.json | P11 |
| `DirLister` | `func(dir string) ([]string, error)` | [E21 · cli](c3-engram-cli-binary.md#e21-cli) ([c4-cli.md](c4-cli.md)) | `osDirListMd` wrapping `os.ReadDir` | P11–P13 |
| `SkillFinder` | `func(root string) []string` | [E21 · cli](c3-engram-cli-binary.md#e21-cli) ([c4-cli.md](c4-cli.md)) | `osWalkSkills` wrapping `filepath.WalkDir` | P14 |
| `CWD / Home / GOOS` | `string` | [E21 · cli](c3-engram-cli-binary.md#e21-cli) ([c4-cli.md](c4-cli.md)) | `os.Getwd`, `$HOME`, `runtime.GOOS` | P3, P4, P14, P15 |

## Property Ledger

| ID | Property | Statement | Enforced at | Tested at | Notes |
|---|---|---|---|---|---|
| <a id="p1-claude-md-ancestor-walk"></a>P1 | CLAUDE.md ancestor walk | For all start directories `cwd`, `DiscoverClaudeMd` visits each ancestor from `cwd` to the filesystem root and includes every `CLAUDE.md` and `CLAUDE.local.md` for which `StatFn` reports existence. | [internal/externalsources/claudemd.go:79](../../internal/externalsources/claudemd.go#L79) | [internal/externalsources/claudemd_test.go:12](../../internal/externalsources/claudemd_test.go#L12) |   |
| <a id="p2-stat-errors-silently-skipped"></a>P2 | Stat errors silently skipped | For all stat results, `DiscoverClaudeMd` ignores any error returned by `StatFn` (treating it as `not exists`) and never propagates it to the caller. | [internal/externalsources/claudemd.go:87](../../internal/externalsources/claudemd.go#L87) | [internal/externalsources/claudemd_test.go:97](../../internal/externalsources/claudemd_test.go#L97) |   |
| <a id="p3-empty-home-skips-user-scope"></a>P3 | Empty home skips user scope | For all calls to `DiscoverClaudeMd` where `home == ""`, no `~/.claude/CLAUDE.md` candidate is stat-ed and no user-scope entry appears in the result. | [internal/externalsources/claudemd.go:65](../../internal/externalsources/claudemd.go#L65) | [internal/externalsources/claudemd_test.go:39](../../internal/externalsources/claudemd_test.go#L39) |   |
| <a id="p4-managed-policy-path-is-os-specific"></a>P4 | Managed-policy path is OS-specific | For `goos` in {`darwin`, `linux`, `windows`}, `ManagedPolicyPath(goos)` returns the documented system-wide CLAUDE.md path; for any other value it returns `""` and `DiscoverClaudeMd` adds no managed-policy entry. | [internal/externalsources/claudemd.go:29](../../internal/externalsources/claudemd.go#L29) | [internal/externalsources/claudemd_test.go:121](../../internal/externalsources/claudemd_test.go#L121), [:133](../../internal/externalsources/claudemd_test.go#L133), [:107](../../internal/externalsources/claudemd_test.go#L107) |   |
| <a id="p5-imports-depth-capped-at-5"></a>P5 | Imports depth-capped at 5 | For all import graphs, `ExpandImports` enqueues no node at depth ≥ 5; transitive `@`-imports beyond five hops from the start file are never returned. | [internal/externalsources/imports.go:23](../../internal/externalsources/imports.go#L23) | [internal/externalsources/imports_test.go:11](../../internal/externalsources/imports_test.go#L11) | `maxImportHops = 5` constant. |
| <a id="p6-import-cycles-broken"></a>P6 | Import cycles broken | For all import graphs containing a cycle, `ExpandImports` returns each distinct file at most once and terminates. | [internal/externalsources/imports.go:66](../../internal/externalsources/imports.go#L66) | [internal/externalsources/imports_test.go:48](../../internal/externalsources/imports_test.go#L48) | Visited map seeded with `startPath` so the start file is not re-emitted. |
| <a id="p7-relative-imports-resolved-against-containing-file"></a>P7 | Relative imports resolved against containing file | For all `@target` references where `target` does not start with `/` or `~`, the resolved path is `filepath.Join(filepath.Dir(containingFile), target)`; absolute and `~`-rooted paths are returned unchanged. | [internal/externalsources/imports.go:90](../../internal/externalsources/imports.go#L90) | [internal/externalsources/imports_test.go:68](../../internal/externalsources/imports_test.go#L68), [:87](../../internal/externalsources/imports_test.go#L87) |   |
| <a id="p8-reader-errors-skipped-silently"></a>P8 | Reader errors skipped silently | For all import-expansion `Reader` calls returning a non-nil error or nil body, `ExpandImports` skips that node without aborting; remaining queue items are still processed. | [internal/externalsources/imports.go:28](../../internal/externalsources/imports.go#L28) | [internal/externalsources/imports_test.go:36](../../internal/externalsources/imports_test.go#L36) |   |
| <a id="p9-imports-deduplicated-across-ancestors"></a>P9 | Imports deduplicated across ancestors | For all `Discover` invocations, every imported file path appears at most once in the returned slice even when multiple CLAUDE.md ancestors transitively import it. | [internal/externalsources/discover.go:46](../../internal/externalsources/discover.go#L46) | [internal/externalsources/discover_test.go:97](../../internal/externalsources/discover_test.go#L97) |   |
| <a id="p10-rules-without-paths-frontmatter-always-included"></a>P10 | Rules without paths-frontmatter always included | For all rule files where `ParseFrontmatter` yields an empty `Paths` slice, `DiscoverRules` includes the file unconditionally; rules with a non-empty `paths:` list are included only when `MatchAny` returns true for that list. | [internal/externalsources/rules.go:49](../../internal/externalsources/rules.go#L49) | [internal/externalsources/rules_test.go:12](../../internal/externalsources/rules_test.go#L12), [:43](../../internal/externalsources/rules_test.go#L43), [:63](../../internal/externalsources/rules_test.go#L63) |   |
| <a id="p11-auto-memory-resolution-precedence"></a>P11 | Auto-memory resolution precedence | For all `DiscoverAutoMemory` calls, the resolved directory is the first of (1) the `Settings`-supplied directory when found and non-empty AND it has files, (2) `cwdProjectDir` when it has files, (3) `mainProjectDir` when set, non-empty, and distinct from `cwdProjectDir`. | [internal/externalsources/automemory.go:25](../../internal/externalsources/automemory.go#L25) | [internal/externalsources/automemory_test.go:33](../../internal/externalsources/automemory_test.go#L33), [:12](../../internal/externalsources/automemory_test.go#L12), [:68](../../internal/externalsources/automemory_test.go#L68), [:94](../../internal/externalsources/automemory_test.go#L94) |   |
| <a id="p12-auto-memory-only-includes-md-files"></a>P12 | Auto-memory only includes *.md files | For all entries returned by `DirLister`, `DiscoverAutoMemory` includes only those with `filepath.Ext(path) == ".md"` as `KindAutoMemory` files. | [internal/externalsources/automemory.go:57](../../internal/externalsources/automemory.go#L57) | [internal/externalsources/automemory_test.go:132](../../internal/externalsources/automemory_test.go#L132) |   |
| <a id="p13-empty-result-is-normal"></a>P13 | Empty result is normal | For all `DiscoverAutoMemory` calls where no resolution step yields any files, the returned slice is empty (length 0) and no error is surfaced. | [internal/externalsources/automemory.go:41](../../internal/externalsources/automemory.go#L41) | [internal/externalsources/automemory_test.go:116](../../internal/externalsources/automemory_test.go#L116) |   |
| <a id="p14-skill-discovery-roots"></a>P14 | Skill discovery roots | For all `DiscoverSkills` calls, `SkillFinder` is invoked on `<cwd>/.claude/skills` always, and additionally on `<home>/.claude/skills` and `<home>/.claude/plugins/cache` only when `home != ""`. | [internal/externalsources/skills.go:15](../../internal/externalsources/skills.go#L15) | [internal/externalsources/skills_test.go:12](../../internal/externalsources/skills_test.go#L12), [:47](../../internal/externalsources/skills_test.go#L47) |   |
| <a id="p15-projectslug-substitution"></a>P15 | ProjectSlug substitution | For all absolute paths P, `ProjectSlug(P)` equals `strings.ReplaceAll(P, "/", "-")`; matches Claude Code's `~/.claude/projects/<slug>/` directory layout. | [internal/externalsources/slug.go:15](../../internal/externalsources/slug.go#L15) | [internal/externalsources/slug_test.go:11](../../internal/externalsources/slug_test.go#L11), [:21](../../internal/externalsources/slug_test.go#L21) |   |
| <a id="p16-filecache-memoizes-content-and-errors"></a>P16 | FileCache memoizes content and errors | For all paths P, `FileCache.Read(P)` invokes the underlying `ReaderFunc` at most once across the lifetime of a single `FileCache`; subsequent calls return the cached `(content, err)` pair, including cached errors. | [internal/externalsources/cache.go:22](../../internal/externalsources/cache.go#L22) | [internal/externalsources/cache_test.go:33](../../internal/externalsources/cache_test.go#L33), [:56](../../internal/externalsources/cache_test.go#L56), [:12](../../internal/externalsources/cache_test.go#L12) | Per-process; no cross-invocation persistence. |
| <a id="p17-frontmatter-parser-fence-requirement"></a>P17 | Frontmatter parser fence requirement | For all bodies that do not begin with `---\n` or that lack a closing `\n---\n` fence, `ParseFrontmatter` returns a zero-value `Frontmatter` and the original body bytes unchanged. | [internal/externalsources/frontmatter.go:24](../../internal/externalsources/frontmatter.go#L24) | [internal/externalsources/frontmatter_test.go:70](../../internal/externalsources/frontmatter_test.go#L70), [:118](../../internal/externalsources/frontmatter_test.go#L118) |   |
| <a id="p18-frontmatter-recognized-keys"></a>P18 | Frontmatter recognized keys | For all YAML blocks, `ParseFrontmatter` populates only the `name`, `description` (scalar or `>` folded), and `paths` (`- item` list) keys; all other keys are silently ignored. | [internal/externalsources/frontmatter.go:121](../../internal/externalsources/frontmatter.go#L121) | [internal/externalsources/frontmatter_test.go:51](../../internal/externalsources/frontmatter_test.go#L51), [:11](../../internal/externalsources/frontmatter_test.go#L11), [:83](../../internal/externalsources/frontmatter_test.go#L83) |   |
| <a id="p19-discover-ordering"></a>P19 | Discover ordering | For all `Discover` calls, the returned slice is the concatenation, in order, of CLAUDE.md ancestors → expanded imports → rules → auto memory → skills. | [internal/externalsources/discover.go:31](../../internal/externalsources/discover.go#L31) | [internal/externalsources/discover_test.go:12](../../internal/externalsources/discover_test.go#L12) | Ordering here does NOT determine recall phase priority — that is set by recall/orchestrate. |
| <a id="p20-kinds-are-exhaustively-stringified"></a>P20 | Kinds are exhaustively stringified | For all `Kind` values defined in this package (`KindClaudeMd`, `KindRules`, `KindAutoMemory`, `KindSkill`, `KindUnknown`), `Kind.String()` returns the documented lowercase identifier; any other int value returns `"invalid"`. | [internal/externalsources/externalsources.go:21](../../internal/externalsources/externalsources.go#L21) | [internal/externalsources/externalsources_test.go:25](../../internal/externalsources/externalsources_test.go#L25) |   |
| <a id="p21-no-direct-i-o-di-only"></a>P21 | No direct I/O (DI-only) | For all package code under `internal/externalsources/`, no symbol references `os.*`, `runtime.*`, `filepath.WalkDir`, `filepath.Glob`, or any other process/OS facility directly; all such effects flow through the injected `StatFn`, `Reader`, `MdWalker`, `MatchAny`, `Settings`, `DirLister`, `SkillFinder` adapters wired by E21 · cli. | [internal/externalsources/discover.go:6](../../internal/externalsources/discover.go#L6) | **⚠ UNTESTED** | Architectural invariant from project DI rule (CLAUDE.md "DI everywhere"). No automated guard; would need an import-scanner test. |

## Cross-links

- Parent: [c3-engram-cli-binary.md](c3-engram-cli-binary.md) (refines **E25 · externalsources**)
- Siblings:
  - [c4-anthropic.md](c4-anthropic.md)
  - [c4-cli.md](c4-cli.md)
  - [c4-context.md](c4-context.md)
  - [c4-main.md](c4-main.md)
  - [c4-memory.md](c4-memory.md)
  - [c4-recall.md](c4-recall.md)
  - [c4-tokenresolver.md](c4-tokenresolver.md)
  - [c4-tomlwriter.md](c4-tomlwriter.md)

See `skills/c4/references/property-ledger-format.md` for the full row format and untested-property
discipline.

