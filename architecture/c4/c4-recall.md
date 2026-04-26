---
level: 4
name: recall
parent: "c3-engram-cli-binary.md"
children: []
last_reviewed_commit: 6002fa69
---

# C4 — recall (Property/Invariant Ledger)

> Component in focus: **E22 · recall** (refines L3 c3-engram-cli-binary).
> Source files in scope:
> - [../../internal/recall/recall.go](../../internal/recall/recall.go)
> - [../../internal/recall/orchestrate.go](../../internal/recall/orchestrate.go)
> - [../../internal/recall/summarize.go](../../internal/recall/summarize.go)
> - [../../internal/recall/automemory_phase.go](../../internal/recall/automemory_phase.go)
> - [../../internal/recall/skill_phase.go](../../internal/recall/skill_phase.go)
> - [../../internal/recall/claudemd_phase.go](../../internal/recall/claudemd_phase.go)
> - test files: `*_test.go` siblings of the above.

## Context (from L3)

Scoped slice of [c3-engram-cli-binary.md](c3-engram-cli-binary.md): the L3 edges that touch
E22. The DI back-edge convention applies — E22 → E21 represents the categories of dependencies
(`Finder`, `Reader`, `SummarizerI`, `MemoryLister`, status writer, external files + cache) wired
by E21 at `recall.NewOrchestrator`.

![C4 recall context diagram](svg/c4-recall.svg)

> Diagram source: [svg/c4-recall.mmd](svg/c4-recall.mmd). Re-render with
> `npx @mermaid-js/mermaid-cli -i architecture/c4/svg/c4-recall.mmd -o architecture/c4/svg/c4-recall.svg`.
> Pre-rendered because GitHub's Mermaid lacks the ELK layout engine, which is needed to
> separate bidirectional R/D edges between the same node pair.

## Dependency Manifest

| Dep field | Type | Wired by | Concrete adapter | Properties |
|---|---|---|---|---|
| `finder` | `Finder` (`Find(projectDir) ([]FileEntry, error)`) | [E21 · cli](c3-engram-cli-binary.md#e21-cli) ([c4-cli.md](c4-cli.md)) | `recall.NewSessionFinder(&osDirLister{})` at [internal/cli/cli.go:238](../../internal/cli/cli.go#L238) | P1, P3 |
| `reader` | `Reader` (`Read(path, budget) (string, int, error)`) | E21 cli | `recall.NewTranscriptReader(&osFileReader{})` at [internal/cli/cli.go:239](../../internal/cli/cli.go#L239) | P2, P4 |
| `summarizer` | `SummarizerI` | E21 cli | `recall.NewSummarizer(&haikuCallerAdapter{caller: makeAnthropicCaller(token)})` via `newSummarizer(token)` at [internal/cli/cli.go:137](../../internal/cli/cli.go#L137) — nil when token empty | P5, P7, P8, P11 |
| `memoryLister` | `MemoryLister` (`ListAllMemories(dataDir)`) | E21 cli | `memory.NewLister()` at [internal/cli/cli.go:179](../../internal/cli/cli.go#L179) | P9, P10 |
| `dataDir` | `string` | E21 cli | `cli.DataDirFromHome(home, os.Getenv)` at [internal/cli/cli.go:99](../../internal/cli/cli.go#L99) | P9, P10 |
| `statusWriter` (option) | `io.Writer` | E21 cli | `os.Stderr` via `recall.WithStatusWriter` at [internal/cli/cli.go:244](../../internal/cli/cli.go#L244) | P14 |
| `externalFiles` (option) | `[]externalsources.ExternalFile` | E21 cli | `discoverExternalSources(ctx, home)` at [internal/cli/cli.go:241](../../internal/cli/cli.go#L241) via `recall.WithExternalSources` | P12 |
| `fileCache` (option) | `*externalsources.FileCache` | E21 cli | shared cache from `discoverExternalSources` | P12 |
| `Summarizer.caller` | `HaikuCaller` | E21 cli | `&haikuCallerAdapter{...}` at [internal/cli/cli.go:139](../../internal/cli/cli.go#L139) | P7, P8 |
| `TranscriptReader.reader` | `sessionctx.FileReader` | E21 cli | `&osFileReader{}` at [internal/cli/cli.go:239](../../internal/cli/cli.go#L239) | P4 |
| `SessionFinder.lister` | `DirLister` | E21 cli | `&osDirLister{}` at [internal/cli/cli.go:238](../../internal/cli/cli.go#L238) | P1, P3 |

## Property Ledger

| ID | Property | Statement | Enforced at | Tested at | Notes |
|---|---|---|---|---|---|
| <a id="p1-find-sorts-desc"></a>P1 | Find sorts mtime descending | For all `Find(projectDir)` calls, the returned `[]FileEntry` is sorted by `Mtime` descending (newest first). | [internal/recall/recall.go:44](../../internal/recall/recall.go#L44) | [internal/recall/recall_test.go:50](../../internal/recall/recall_test.go#L50) | Sort is stable across implementations. |
| <a id="p2-read-budget"></a>P2 | Read respects budget | For all `(path, budgetBytes)`, `TranscriptReader.Read` accumulates from the tail and stops once `bytesRead + lineLen > budgetBytes` (with at least one line if any). | [internal/recall/recall.go:88](../../internal/recall/recall.go#L88) | [internal/recall/recall_test.go:95](../../internal/recall/recall_test.go#L95), [:133](../../internal/recall/recall_test.go#L133) | Returns chronological order even though selected from tail. |
| <a id="p3-find-error-wrapped"></a>P3 | Find errors wrapped | For all `lister.ListJSONL` errors, `Find` returns `fmt.Errorf("listing sessions: %w", err)`. | [internal/recall/recall.go:41](../../internal/recall/recall.go#L41) | [internal/recall/recall_test.go:34](../../internal/recall/recall_test.go#L34) | — |
| <a id="p4-strip-tool-summary"></a>P4 | Strip removes tool noise | For all transcript bytes, `TranscriptReader.Read` invokes `sessionctx.StripWithConfig` with `ToolSummaryMode: true` before applying the budget. | [internal/recall/recall.go:80](../../internal/recall/recall.go#L80) | [internal/recall/recall_test.go:170](../../internal/recall/recall_test.go#L170), [:201](../../internal/recall/recall_test.go#L201) | Stripping is owned by `internal/context`. |
| <a id="p5-mode-selection"></a>P5 | Mode selection by query | For all `Recall(ctx, projectDir, query)` calls: empty query → mode A (raw transcript concatenation); non-empty query → mode B (LLM-extracted snippets). | [internal/recall/orchestrate.go:87](../../internal/recall/orchestrate.go#L87) | [internal/recall/orchestrate_test.go:365](../../internal/recall/orchestrate_test.go#L365), [:819](../../internal/recall/orchestrate_test.go#L819) | Mode B requires a non-nil summarizer. |
| <a id="p6-empty-sessions"></a>P6 | Empty sessions returns empty | For all `Recall` calls where `finder.Find` returns 0 sessions, the result is `&Result{}` and no further phases run. | [internal/recall/orchestrate.go:83](../../internal/recall/orchestrate.go#L83) | [internal/recall/orchestrate_test.go:365](../../internal/recall/orchestrate_test.go#L365) | — |
| <a id="p7-mode-b-budget"></a>P7 | Mode B respects extract cap | For all mode-B runs, accumulated bytes across phases (memories, auto memory, sessions, skills, claude.md) stop being added once `bytesUsed >= DefaultExtractCap` (10 KiB). | [internal/recall/orchestrate.go:132](../../internal/recall/orchestrate.go#L132) | [internal/recall/orchestrate_test.go:1075](../../internal/recall/orchestrate_test.go#L1075), [cost_test.go:17](../../internal/recall/cost_test.go#L17) | Each phase enforces its own check. |
| <a id="p8-summarizer-nil-caller"></a>P8 | Summarizer guards nil caller | For all `ExtractRelevant` / `SummarizeFindings` calls on a `Summarizer` whose `caller` is nil, the call returns `("", ErrNilCaller)`. | [internal/recall/summarize.go:31](../../internal/recall/summarize.go#L31) | [internal/recall/summarize_test.go:78](../../internal/recall/summarize_test.go#L78) | Sentinel exported. |
| <a id="p9-memories-only-default-limit"></a>P9 | Default memory limit | For all `RecallMemoriesOnly(ctx, query, limit)` calls where `limit <= 0`, the effective limit is `DefaultMemoryLimit` (10). | [internal/recall/orchestrate.go:103](../../internal/recall/orchestrate.go#L103) | [internal/recall/orchestrate_test.go:114](../../internal/recall/orchestrate_test.go#L114) | — |
| <a id="p10-memory-windowing"></a>P10 | Mode A memory windowing | For all mode-A runs with a `memoryLister` and non-empty `dataDir`, returned `Memories` contain only memories whose `UpdatedAt` falls within any session's time window (defined by adjacent mtimes; first/only session uses 24 h). | [internal/recall/orchestrate.go:158](../../internal/recall/orchestrate.go#L158) | [internal/recall/orchestrate_test.go:674](../../internal/recall/orchestrate_test.go#L674) | `defaultSessionWindow = 24h`. |
| <a id="p11-mode-b-summary-error-propagated"></a>P11 | Mode B summary errors propagate | For all mode-B runs where `summarizer.SummarizeFindings` returns an error, `Recall` returns `nil, fmt.Errorf("summarizing recall: %w", err)`. | [internal/recall/orchestrate.go:307](../../internal/recall/orchestrate.go#L307) | [internal/recall/orchestrate_test.go:1047](../../internal/recall/orchestrate_test.go#L1047) | — |
| <a id="p12-phase-order"></a>P12 | Mode B phase priority | For all mode-B runs, phases execute in this fixed order: (1) engram memories, (2) auto memory, (3) per-session extraction, (4) skills, (5) CLAUDE.md/rules, then (6) structured summarization. | [internal/recall/orchestrate.go:264](../../internal/recall/orchestrate.go#L264) | [internal/recall/orchestrate_test.go:1075](../../internal/recall/orchestrate_test.go#L1075) | Documented phase ordering. |
| <a id="p13-cancellation-partial"></a>P13 | Cancellation returns partial mode A | For all mode-A runs where `ctx` is cancelled mid-loop, `Recall` returns the bytes accumulated so far without an error (cancellation is signal, not failure). | [internal/recall/orchestrate.go:231](../../internal/recall/orchestrate.go#L231) | [internal/recall/orchestrate_test.go:494](../../internal/recall/orchestrate_test.go#L494) | `//nolint:nilerr` documents intent. |
| <a id="p14-status-optional"></a>P14 | Status writes are optional | For all `Orchestrator` instances with no `WithStatusWriter`, `writeStatusf` is a no-op (no panic, no allocation of a real writer). | [internal/recall/orchestrate.go:337](../../internal/recall/orchestrate.go#L337) | [internal/recall/orchestrate_test.go:1011](../../internal/recall/orchestrate_test.go#L1011) | — |
| <a id="p15-format-result"></a>P15 | FormatResult layout | For all `*Result`, `FormatResult(w, r)` writes `r.Summary` then, if `r.Memories` is non-empty, `"\n=== MEMORIES ===\n" + r.Memories`. | [internal/recall/orchestrate.go:365](../../internal/recall/orchestrate.go#L365) | [internal/recall/orchestrate_test.go:30](../../internal/recall/orchestrate_test.go#L30) | Used by cli `runRecall` for stdout. |
| <a id="p16-haiku-call-bounded"></a>P16 | Haiku call count bounded | For all mode-B runs over N sessions and external sources, the number of `summarizer.ExtractRelevant`+`SummarizeFindings` calls is bounded by `O(N + sources)` and capped per-phase by buffer-full early exit. | [internal/recall/orchestrate.go:127](../../internal/recall/orchestrate.go#L127) | [internal/recall/cost_test.go:17](../../internal/recall/cost_test.go#L17) | Cost guard. |
| <a id="p17-no-direct-io"></a>P17 | No direct I/O (DI-only) | For all package code, no symbol references `os.*`, `net/http`, or any other I/O directly; all such effects flow through `Finder`, `Reader`, `SummarizerI`, `MemoryLister`, and `externalsources.FileCache` injected by the caller. | [internal/recall/orchestrate.go:35](../../internal/recall/orchestrate.go#L35) | **⚠ UNTESTED** | Architectural invariant. No automated guard. |

## Cross-links

- Parent: [c3-engram-cli-binary.md](c3-engram-cli-binary.md) (refines **E22 · recall**)

See `skills/c4/references/property-ledger-format.md` for the full row format and untested-property
discipline.
