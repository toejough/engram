---
level: 4
name: cli
parent: "c3-engram-cli-binary.md"
children: []
last_reviewed_commit: cd55eab2
---

# C4 — cli (Property/Invariant Ledger)

> Component in focus: **E21 · cli** (refines L3 c3-engram-cli-binary.md).
> Source files in scope:
> - [internal/cli/cli.go](internal/cli/cli.go)
> - [internal/cli/targets.go](internal/cli/targets.go)
> - [internal/cli/show.go](internal/cli/show.go)
> - [internal/cli/list.go](internal/cli/list.go)
> - [internal/cli/learn.go](internal/cli/learn.go)
> - [internal/cli/update.go](internal/cli/update.go)
> - [internal/cli/signal.go](internal/cli/signal.go)
> - [internal/cli/externalsources_adapters.go](internal/cli/externalsources_adapters.go)

## Context (from L3)

cli is the composition root of the engram binary. It owns the targ Target definitions for every subcommand (recall, show, list, learn feedback/fact, update), holds the *Args structs, defines the thin I/O adapter shims (osDirLister, osFileReader, osStatExists, osWalkMd, osWalkSkills, osDirListMd, osMatchAny, makeAnthropicCaller, newTokenResolver, readAutoMemoryDirectorySetting), and wires those adapters into every pure-logic component (recall, memory, externalsources, anthropic, tokenresolver, tomlwriter). cli also embeds business-logic handlers for show, list, learn, and update — see the c3 Drift Note: those four are intended to live as peer packages alongside recall but currently live inside cli.

cli applies two universal CLI conventions before any subcommand runs business logic: it resolves an empty `--data-dir` to `$XDG_DATA_HOME/engram` (or `$HOME/.local/share/engram` when XDG is unset), and it resolves an empty `--project-slug` from cwd via `tr '/' '-'`. cli also enforces the SIGINT/SIGTERM force-exit-on-second-signal contract via SetupSignalHandling.

![C4 cli context diagram](svg/c4-cli.svg)

> Diagram source: [svg/c4-cli.mmd](svg/c4-cli.mmd). Re-render with
> `npx @mermaid-js/mermaid-cli -i architecture/c4/svg/c4-cli.mmd -o architecture/c4/svg/c4-cli.svg`.
> Pre-rendered because GitHub's Mermaid lacks the ELK layout engine, which is needed to
> separate bidirectional R/D edges between the same node pair.

**Legend:**
- Yellow = focus component (cli)
- Blue = peer components in the engram-cli-binary container
- Grey = external systems
- Solid R-edges = forward calls; dotted D-edges = DI back-edges (cli wires deps into peers)

## DI Wires

Each row is one adapter this component wires into a consumer. Reciprocal entries
live in the consumer's L4 under "Dependency Manifest".

| Wired adapter | Concrete value | Consumer | Consumer field |
|---|---|---|---|
| `os.Getenv` | stdlib `os.Getenv` | [E27 · tokenresolver](c3-engram-cli-binary.md#e27-tokenresolver) ([c4-tokenresolver.md](c4-tokenresolver.md)) | `getenv` |
| `execCmd` closure | inline closure wrapping `exec.CommandContext` ([cli.go:150](../../internal/cli/cli.go#L150)) | [E27 · tokenresolver](c3-engram-cli-binary.md#e27-tokenresolver) ([c4-tokenresolver.md](c4-tokenresolver.md)) | `execCmd` |
| `runtime.GOOS` | compile-time string from `runtime.GOOS` | [E27 · tokenresolver](c3-engram-cli-binary.md#e27-tokenresolver) ([c4-tokenresolver.md](c4-tokenresolver.md)) | `goos` |
| `*http.Client` | `&http.Client{}` ([cli.go:131](../../internal/cli/cli.go#L131)) | [E26 · anthropic](c3-engram-cli-binary.md#e26-anthropic) ([c4-anthropic.md](c4-anthropic.md)) | `httpClient` |
| API URL string | `AnthropicAPIURL` package var (test-overridable) | [E26 · anthropic](c3-engram-cli-binary.md#e26-anthropic) ([c4-anthropic.md](c4-anthropic.md)) | `apiURL` |
| API token string | result of `resolveToken(ctx)` (env or Keychain) | [E26 · anthropic](c3-engram-cli-binary.md#e26-anthropic) ([c4-anthropic.md](c4-anthropic.md)) | `token` |
| `osDirLister.ListJSONL` | `os.ReadDir`-backed adapter ([cli.go:46](../../internal/cli/cli.go#L46)) | [E22 · recall](c3-engram-cli-binary.md#e22-recall) ([c4-recall.md](c4-recall.md)) | `DirLister (SessionFinder)` |
| `osFileReader.Read` | `os.ReadFile`-backed adapter ([cli.go:84](../../internal/cli/cli.go#L84)) | [E22 · recall](c3-engram-cli-binary.md#e22-recall) ([c4-recall.md](c4-recall.md)) | `FileReader (TranscriptReader)` |
| `haikuCallerAdapter` | wraps `makeAnthropicCaller(token)` to satisfy `recall.HaikuCaller` ([cli.go:139](../../internal/cli/cli.go#L139)) | [E22 · recall](c3-engram-cli-binary.md#e22-recall) ([c4-recall.md](c4-recall.md)) | `Summarizer` |
| `memory.Lister` | `memory.NewLister()` ([cli.go:179](../../internal/cli/cli.go#L179)) | [E22 · recall](c3-engram-cli-binary.md#e22-recall) ([c4-recall.md](c4-recall.md)) | `MemoryLister` |
| `os.Stderr` | stdlib `os.Stderr` via `recall.WithStatusWriter` | [E22 · recall](c3-engram-cli-binary.md#e22-recall) ([c4-recall.md](c4-recall.md)) | `statusWriter` |
| external files + cache | `discoverExternalSources(ctx, home)` via `recall.WithExternalSources` | [E22 · recall](c3-engram-cli-binary.md#e22-recall) ([c4-recall.md](c4-recall.md)) | `externalFiles, externalCache` |
| `os.Getwd` | stdlib `os.Getwd` (passed into `runRecallSessions`) | [E22 · recall](c3-engram-cli-binary.md#e22-recall) ([c4-recall.md](c4-recall.md)) | `getwd` |
| `os.UserHomeDir` | stdlib `os.UserHomeDir` (passed into `runRecallSessions`) | [E22 · recall](c3-engram-cli-binary.md#e22-recall) ([c4-recall.md](c4-recall.md)) | `userHomeDir` |
| `os.Stat` adapter | `osStatExists` ([externalsources_adapters.go:128](../../internal/cli/externalsources_adapters.go#L128)) | [E25 · externalsources](c3-engram-cli-binary.md#e25-externalsources) ([c4-externalsources.md](c4-externalsources.md)) | `StatFn` |
| `FileCache.Read` | `externalsources.NewFileCache(os.ReadFile).Read` ([externalsources_adapters.go:61](../../internal/cli/externalsources_adapters.go#L61)) | [E25 · externalsources](c3-engram-cli-binary.md#e25-externalsources) ([c4-externalsources.md](c4-externalsources.md)) | `Reader` |
| `osWalkMd` | `filepath.WalkDir`-backed adapter for `*.md` ([externalsources_adapters.go:143](../../internal/cli/externalsources_adapters.go#L143)) | [E25 · externalsources](c3-engram-cli-binary.md#e25-externalsources) ([c4-externalsources.md](c4-externalsources.md)) | `MdWalker` |
| `osMatchAny(cwd)` | `filepath.Glob` closure over cwd ([externalsources_adapters.go:113](../../internal/cli/externalsources_adapters.go#L113)) | [E25 · externalsources](c3-engram-cli-binary.md#e25-externalsources) ([c4-externalsources.md](c4-externalsources.md)) | `MatchAny` |
| `readAutoMemoryDirectorySetting(home)` | reads `.claude/settings.local.json` then `~/.claude/settings.json` ([externalsources_adapters.go:183](../../internal/cli/externalsources_adapters.go#L183)) | [E25 · externalsources](c3-engram-cli-binary.md#e25-externalsources) ([c4-externalsources.md](c4-externalsources.md)) | `Settings` |
| `osDirListMd` | non-recursive `os.ReadDir` adapter ([externalsources_adapters.go:87](../../internal/cli/externalsources_adapters.go#L87)) | [E25 · externalsources](c3-engram-cli-binary.md#e25-externalsources) ([c4-externalsources.md](c4-externalsources.md)) | `DirLister` |
| `osWalkSkills` | `filepath.WalkDir`-backed adapter for `SKILL.md` ([externalsources_adapters.go:162](../../internal/cli/externalsources_adapters.go#L162)) | [E25 · externalsources](c3-engram-cli-binary.md#e25-externalsources) ([c4-externalsources.md](c4-externalsources.md)) | `SkillFinder` |
| cwd string | `os.Getwd()` (falls back to `/` on error) | [E25 · externalsources](c3-engram-cli-binary.md#e25-externalsources) ([c4-externalsources.md](c4-externalsources.md)) | `CWD` |
| home string | `os.UserHomeDir()` result threaded from caller | [E25 · externalsources](c3-engram-cli-binary.md#e25-externalsources) ([c4-externalsources.md](c4-externalsources.md)) | `Home` |
| GOOS string | `runtime.GOOS` | [E25 · externalsources](c3-engram-cli-binary.md#e25-externalsources) ([c4-externalsources.md](c4-externalsources.md)) | `GOOS` |
| main project dir string | `computeMainProjectDir(ctx, cwd, home)` via `git rev-parse --git-common-dir` ([externalsources_adapters.go:20](../../internal/cli/externalsources_adapters.go#L20)) | [E25 · externalsources](c3-engram-cli-binary.md#e25-externalsources) ([c4-externalsources.md](c4-externalsources.md)) | `MainProjectDir` |
| tomlwriter (defaults) | `tomlwriter.New()` accepts default `os.*` deps internally | [E28 · tomlwriter](c3-engram-cli-binary.md#e28-tomlwriter) ([c4-tomlwriter.md](c4-tomlwriter.md)) | `createTemp/rename/mkdirAll/stat/remove` |
| memory.Lister (defaults) | `memory.NewLister()` accepts default `os.ReadDir`/`os.ReadFile` internally | [E24 · memory](c3-engram-cli-binary.md#e24-memory) ([c4-memory.md](c4-memory.md)) | `readDir/readFile` |

## Property Ledger

| ID | Property | Statement | Enforced at | Tested at | Notes |
|---|---|---|---|---|---|
| <a id="p1-subcommand-surface-is-fixed"></a>P1 | Subcommand surface is fixed | For all invocations of `Targets`, the returned target slice exposes exactly five top-level subcommands (recall, show, list, learn, update) with `learn` as a group of `feedback` and `fact`. | [internal/cli/targets.go:100](../../internal/cli/targets.go#L100) | **⚠ UNTESTED** | Architectural invariant; no test asserts the set or shape of returned targets. |
| <a id="p2-empty-data-dir-resolves-to-xdg-home-default"></a>P2 | Empty data-dir resolves to XDG/HOME default | For all subcommand invocations with an empty `--data-dir`, `applyDataDirDefault` sets it to `$XDG_DATA_HOME/engram` if that env var is non-empty, otherwise to `$HOME/.local/share/engram`. | [internal/cli/cli.go:89](../../internal/cli/cli.go#L89), [:76](../../internal/cli/targets.go#L76) | [internal/cli/targets_test.go:18](../../internal/cli/targets_test.go#L18) | Tested via the pure helper `DataDirFromHome`; the wrapper `applyDataDirDefault` reads `os.Getenv`/`os.UserHomeDir` directly and is exercised end-to-end by every subcommand test. |
| <a id="p3-empty-project-slug-derives-from-cwd"></a>P3 | Empty project-slug derives from cwd | For all paths P, `ProjectSlugFromPath(P)` returns P with every `filepath.Separator` replaced by `-`, matching the shell `tr '/' '-'` convention. | [internal/cli/targets.go:86](../../internal/cli/targets.go#L86) | [internal/cli/targets_test.go:44](../../internal/cli/targets_test.go#L44) | applyProjectSlugDefault wraps this with an injected `getwd` so callers can stub cwd. |
| <a id="p4-source-flag-must-be-human-or-agent"></a>P4 | Source flag must be human or agent | For all `learn fact`, `learn feedback`, and `update --source=…` invocations, `validateSource` returns `errInvalidSource` unless source is exactly `"human"` or `"agent"`. | [internal/cli/learn.go:286](../../internal/cli/learn.go#L286) | [internal/cli/learn_test.go:687](../../internal/cli/learn_test.go#L687), [:695](../../internal/cli/learn_test.go#L695), [:703](../../internal/cli/learn_test.go#L703), [:711](../../internal/cli/learn_test.go#L711) | Empty source is rejected for learn but accepted for update (update only validates when --source is supplied). |
| <a id="p5-show-requires-a-slug"></a>P5 | Show requires a slug | For all `show` invocations with empty `--name`, `runShow` returns `errShowMissingSlug` without touching the filesystem. | [internal/cli/show.go:123](../../internal/cli/show.go#L123) | [internal/cli/show_test.go:176](../../internal/cli/show_test.go#L176) |   |
| <a id="p6-show-prints-only-non-empty-fields"></a>P6 | Show prints only non-empty fields | For all memory records M, `renderMemory(w, M)` writes a line for a field iff that field is a non-empty string on M. | [internal/cli/show.go:82](../../internal/cli/show.go#L82) | [internal/cli/show_test.go:206](../../internal/cli/show_test.go#L206) | Type, Source, CreatedAt, UpdatedAt and the type-specific content fields are independently gated. |
| <a id="p7-show-selects-renderer-by-type"></a>P7 | Show selects renderer by Type | For all memory records M, `renderMemoryContent` calls `renderFactContent` iff M.Type == "fact", otherwise `renderFeedbackContent` (so feedback fields render for any non-fact type, including empty). | [internal/cli/show.go:88](../../internal/cli/show.go#L88) | [internal/cli/show_test.go:38](../../internal/cli/show_test.go#L38), [:61](../../internal/cli/show_test.go#L61) |   |
| <a id="p8-list-swallows-missing-data-dir"></a>P8 | List swallows missing data dir | For all `list` invocations against a data dir where `ListAllMemories` returns `os.ErrNotExist`, `runList` returns nil and writes nothing. | [internal/cli/list.go:27](../../internal/cli/list.go#L27) | [internal/cli/list_test.go:11](../../internal/cli/list_test.go#L11) | First-run friendliness — an unprovisioned data dir is not an error. |
| <a id="p9-list-output-format-is-type-name-situation"></a>P9 | List output format is type | name | situation | For all stored memories M, `runList` writes one line per memory in the form `<type> | <name> | <situation>\n`. | [internal/cli/list.go:37](../../internal/cli/list.go#L37) | [internal/cli/list_test.go:22](../../internal/cli/list_test.go#L22), [:74](../../internal/cli/list_test.go#L74) |   |
| <a id="p10-learn-skips-dedup-when-no-token"></a>P10 | Learn skips dedup when no token | For all `learn fact`/`learn feedback` invocations where `resolveToken` returns the empty string, `makeConflictDeps` returns a nil caller and `checkForConflicts` short-circuits to (false, nil), so the write proceeds without an Anthropic call. | [internal/cli/learn.go:88](../../internal/cli/learn.go#L88), [:143](../../internal/cli/learn.go#L143) | [internal/cli/learn_test.go:117](../../internal/cli/learn_test.go#L117), [:865](../../internal/cli/learn_test.go#L865) |   |
| <a id="p11-dedup-api-failure-is-non-fatal"></a>P11 | Dedup API failure is non-fatal | For all `learn` invocations where the Haiku caller returns a non-nil error, `checkForConflicts` returns `(false, nil)` and the memory is still written. | [internal/cli/learn.go:109](../../internal/cli/learn.go#L109) | [internal/cli/learn_test.go:95](../../internal/cli/learn_test.go#L95) | Intentional: Anthropic outages must not block memory capture. |
| <a id="p12-dedup-conflict-aborts-write"></a>P12 | Dedup conflict aborts write | For all `learn` invocations where Haiku reports a `DUPLICATE` or `CONTRADICTION` line and `--no-dup-check` is not set, `writeMemory` returns nil before invoking `tomlwriter.Write`, leaving the data dir unchanged. | [internal/cli/learn.go:312](../../internal/cli/learn.go#L312) | [internal/cli/learn_test.go:764](../../internal/cli/learn_test.go#L764) |   |
| <a id="p13-dedup-recognizes-only-none-duplicate-contradiction"></a>P13 | Dedup recognizes only NONE/DUPLICATE/CONTRADICTION | For all Haiku responses R, `parseConflictResponse(R, ...)` returns true iff at least one trimmed line begins with `DUPLICATE:` or `CONTRADICTION:`; the literal `NONE` and any other text return false. | [internal/cli/learn.go:175](../../internal/cli/learn.go#L175) | [internal/cli/learn_test.go:504](../../internal/cli/learn_test.go#L504), [:542](../../internal/cli/learn_test.go#L542), [:595](../../internal/cli/learn_test.go#L595), [:606](../../internal/cli/learn_test.go#L606) |   |
| <a id="p14-learn-output-announces-created-name"></a>P14 | Learn output announces created name | For all successful `learn fact` / `learn feedback` writes, stdout receives a single trailing line `CREATED: <name>\n` where <name> is `memory.NameFromPath(filePath)`. | [internal/cli/learn.go:332](../../internal/cli/learn.go#L332) | [internal/cli/learn_test.go:429](../../internal/cli/learn_test.go#L429) |   |
| <a id="p15-update-preserves-unspecified-fields"></a>P15 | Update preserves unspecified fields | For all `update` invocations and field F, if the corresponding `UpdateArgs` field is empty, `applyUpdateArgs` does not modify M[F]. | [internal/cli/update.go:14](../../internal/cli/update.go#L14) | [internal/cli/update_test.go:262](../../internal/cli/update_test.go#L262) | Update is a sparse patch, not a full replacement. |
| <a id="p16-update-stamps-updatedat"></a>P16 | Update stamps UpdatedAt | For all successful `update` invocations, the persisted record's `UpdatedAt` field is set to `time.Now().UTC()` formatted as RFC3339. | [internal/cli/update.go:75](../../internal/cli/update.go#L75) | [internal/cli/update_test.go:218](../../internal/cli/update_test.go#L218) | Time source is not injected — tests assert format/non-empty rather than exact value. |
| <a id="p17-update-requires-name"></a>P17 | Update requires --name | For all `update` invocations missing `--name`, targ flag-parse returns an error and `runUpdate` is not called. | [internal/cli/targets.go:61](../../internal/cli/targets.go#L61) | [internal/cli/update_test.go:210](../../internal/cli/update_test.go#L210) | Enforced by `targ:"flag,...,required"` tag. |
| <a id="p18-second-signal-forces-exit"></a>P18 | Second signal forces exit | For all signal channels S, after `ForceExitOnRepeatedSignal` receives two values on S, it invokes `exitFn(ExitCodeSigInt)` (130). The first signal is allowed to flow through targ's context cancellation. | [internal/cli/signal.go:19](../../internal/cli/signal.go#L19) | [internal/cli/signal_test.go:16](../../internal/cli/signal_test.go#L16) | Buffered signal channel size is 10. |
| <a id="p19-setupsignalhandling-returns-targets"></a>P19 | SetupSignalHandling returns Targets | For all (stdout, stderr, stdin, exitFn) tuples, `SetupSignalHandling` returns `Targets(stdout, stderr, stdin)` after registering SIGINT/SIGTERM and starting the force-exit goroutine. | [internal/cli/signal.go:37](../../internal/cli/signal.go#L37) | [internal/cli/signal_test.go:65](../../internal/cli/signal_test.go#L65) | main.go relies on this to obtain the targ target slice. |
| <a id="p20-osdirlister-filters-to-jsonl-files"></a>P20 | osDirLister filters to .jsonl files | For all directories D, `osDirLister.ListJSONL(D)` returns one `recall.FileEntry` per non-directory entry whose name has suffix `.jsonl`, and only those entries. | [internal/cli/cli.go:46](../../internal/cli/cli.go#L46) | [internal/cli/adapters_test.go:53](../../internal/cli/adapters_test.go#L53), [:75](../../internal/cli/adapters_test.go#L75) | Stat failures are logged to stderr and silently skipped, not surfaced. |
| <a id="p21-osfilereader-is-a-transparent-os-readfile-shim"></a>P21 | osFileReader is a transparent os.ReadFile shim | For all paths P, `osFileReader.Read(P)` returns the same `(bytes, error)` pair as `os.ReadFile(P)` with no transformation. | [internal/cli/cli.go:84](../../internal/cli/cli.go#L84) | [internal/cli/adapters_test.go:84](../../internal/cli/adapters_test.go#L84), [:102](../../internal/cli/adapters_test.go#L102) |   |
| <a id="p22-osmatchany-ignores-glob-errors"></a>P22 | osMatchAny ignores Glob errors | For all glob slices G and cwd C, `osMatchAny(C)(G)` returns true iff at least one glob in G yields a non-empty `filepath.Glob` match under C; pattern errors are treated as no-match. | [internal/cli/externalsources_adapters.go:113](../../internal/cli/externalsources_adapters.go#L113) | [internal/cli/externalsources_adapters_test.go:56](../../internal/cli/externalsources_adapters_test.go#L56) | `**` patterns silently fail to match — known safe-failure mode. |
| <a id="p23-osdirlistmd-swallows-missing-dir"></a>P23 | osDirListMd swallows missing dir | For all directories D that do not exist or fail to read, `osDirListMd(D)` returns `(nil, nil)` so DiscoverAutoMemory treats it as no files. | [internal/cli/externalsources_adapters.go:87](../../internal/cli/externalsources_adapters.go#L87) | [internal/cli/externalsources_adapters_test.go:42](../../internal/cli/externalsources_adapters_test.go#L42) | Auto-memory dir is opt-in; absent dir is the common case. |
| <a id="p24-oswalkmd-is-recursive-and-lossy"></a>P24 | osWalkMd is recursive and lossy | For all roots R, `osWalkMd(R)` returns absolute paths to every file under R with extension `.md`; permission and read errors on subtrees are silently skipped, not propagated. | [internal/cli/externalsources_adapters.go:143](../../internal/cli/externalsources_adapters.go#L143) | [internal/cli/externalsources_adapters_test.go:101](../../internal/cli/externalsources_adapters_test.go#L101), [:118](../../internal/cli/externalsources_adapters_test.go#L118) |   |
| <a id="p25-oswalkskills-filters-to-skill-md-only"></a>P25 | osWalkSkills filters to SKILL.md only | For all roots R, `osWalkSkills(R)` returns absolute paths only to files named exactly `SKILL.md` under R. | [internal/cli/externalsources_adapters.go:162](../../internal/cli/externalsources_adapters.go#L162) | [internal/cli/externalsources_adapters_test.go:125](../../internal/cli/externalsources_adapters_test.go#L125) |   |
| <a id="p26-auto-memory-settings-precedence-is-local-then-user"></a>P26 | Auto-memory settings precedence is local-then-user | For all home directories H, `readAutoMemoryDirectorySetting(H)()` returns the first non-empty `autoMemoryDirectory` it finds in `./.claude/settings.local.json` then `H/.claude/settings.json`, otherwise `("", false)`. | [internal/cli/externalsources_adapters.go:183](../../internal/cli/externalsources_adapters.go#L183) | [internal/cli/externalsources_adapters_test.go:139](../../internal/cli/externalsources_adapters_test.go#L139), [:157](../../internal/cli/externalsources_adapters_test.go#L157) |   |
| <a id="p27-worktree-detection-falls-back-to-empty"></a>P27 | Worktree detection falls back to empty | For all (ctx, cwd, home), `computeMainProjectDir` returns the empty string when cwd is not inside a git worktree distinct from the main checkout, when git is unavailable, or when `git rev-parse --git-common-dir` errors. | [internal/cli/externalsources_adapters.go:20](../../internal/cli/externalsources_adapters.go#L20) | [internal/cli/externalsources_adapters_test.go:14](../../internal/cli/externalsources_adapters_test.go#L14) | Reciprocal-memory fallback for worktree-less repos. |
| <a id="p28-haikucalleradapter-pins-haiku-model"></a>P28 | haikuCallerAdapter pins Haiku model | For all (ctx, systemPrompt, userPrompt) tuples, `haikuCallerAdapter.Call` invokes the wrapped caller with model exactly `anthropic.HaikuModel` (claude-haiku-4-5-20251001). | [internal/cli/cli.go:36](../../internal/cli/cli.go#L36) | [internal/cli/adapters_test.go:15](../../internal/cli/adapters_test.go#L15), [:39](../../internal/cli/adapters_test.go#L39) | Centralizes the model pin so subcommands cannot accidentally choose a different one. |
| <a id="p29-newsummarizer-returns-nil-on-empty-token"></a>P29 | newSummarizer returns nil on empty token | For all empty token strings, `newSummarizer("")` returns nil, signalling the recall pipeline to skip LLM phases. | [internal/cli/cli.go:137](../../internal/cli/cli.go#L137) | [internal/cli/learn_test.go:467](../../internal/cli/learn_test.go#L467), [:475](../../internal/cli/learn_test.go#L475) |   |
| <a id="p30-discoverexternalsources-never-returns-nil-cache"></a>P30 | discoverExternalSources never returns nil cache | For all (ctx, home), `discoverExternalSources` returns a non-nil `*externalsources.FileCache` so downstream readers can always call `cache.Read` safely. | [internal/cli/externalsources_adapters.go:52](../../internal/cli/externalsources_adapters.go#L52) | **⚠ UNTESTED** | Architectural invariant; no direct unit test — exercised indirectly via runRecallSessions. |
| <a id="p31-all-cli-i-o-lives-at-this-seam"></a>P31 | All cli I/O lives at this seam | For all packages under `internal/` other than `internal/cli`, no production source calls `os.*`, `exec.*`, `runtime.GOOS`, or `http.*` directly — all such calls are wired through DI from cli. | [internal/cli/cli.go:1](../../internal/cli/cli.go#L1), [:1](../../internal/cli/externalsources_adapters.go#L1) | **⚠ UNTESTED** | Architectural invariant from the project's DI-everywhere principle (CLAUDE.md). No mechanical lint enforces it; review-time discipline. |

## Cross-links

- Parent: [c3-engram-cli-binary.md](c3-engram-cli-binary.md) (refines **E21 · cli**)
- Siblings:
  - [c4-anthropic.md](c4-anthropic.md)
  - [c4-context.md](c4-context.md)
  - [c4-externalsources.md](c4-externalsources.md)
  - [c4-main.md](c4-main.md)
  - [c4-memory.md](c4-memory.md)
  - [c4-recall.md](c4-recall.md)
  - [c4-tokenresolver.md](c4-tokenresolver.md)
  - [c4-tomlwriter.md](c4-tomlwriter.md)

See `skills/c4/references/property-ledger-format.md` for the full row format and untested-property
discipline.

