---
level: 4
name: context
parent: "c3-engram-cli-binary.md"
children: []
last_reviewed_commit: 95a5a92f
---

# C4 — context (Property/Invariant Ledger)

> Component in focus: **S2-N3-M4 · context**.
> Source files in scope:
> - [../../internal/context/context.go](../../internal/context/context.go)
> - [../../internal/context/delta.go](../../internal/context/delta.go)
> - [../../internal/context/strip.go](../../internal/context/strip.go)
> - [../../internal/context/stripconfig.go](../../internal/context/stripconfig.go)
> - [../../internal/context/toolsummary.go](../../internal/context/toolsummary.go)
> - [../../internal/context/context_test.go](../../internal/context/context_test.go)
> - [../../internal/context/stripconfig_test.go](../../internal/context/stripconfig_test.go)

## Context (from L3)

Scoped slice of [c3-engram-cli-binary.md](c3-engram-cli-binary.md): the L3 edges that touch E23. internal/context declares a `FileReader` interface as a public type but no function in the package consumes it — the recall package wires its own `TranscriptReader` adapter when invoking `context.StripSeqAndConvertHumanTurns`. context therefore has zero DI seams of its own and no Dependency Manifest under the new schema.

![C4 context context diagram](svg/c4-context.svg)

> Diagram source: [svg/c4-context.mmd](svg/c4-context.mmd). Re-render with
> `npx @mermaid-js/mermaid-cli -i architecture/c4/svg/c4-context.mmd -o architecture/c4/svg/c4-context.svg`.
> Pre-rendered because GitHub's Mermaid lacks the ELK layout engine, which is needed to
> separate bidirectional R-edges between the same node pair.

**Legend:**
- Yellow = focus component (S2-N3-M4 · context).
- Blue components = sibling components in c3-engram-cli-binary.md.
- R-edges carry inline property IDs `[P…]` linking to the Property Ledger.
- All edges traceable to a relationship in c3-engram-cli-binary.md.

## Property Ledger

| ID | Property | Statement | Enforced at | Tested at | Notes |
|---|---|---|---|---|---|
| <a id="s2-n3-m4-p1-delta-from-offset-0"></a>S2-N3-M4-P1 | Delta from offset 0 | For all transcript paths P with file content C of length N, calling `DeltaReader.Read(P, 0)` returns every non-empty line of C as the delta and N as the new offset. | [internal/context/delta.go:20](../../internal/context/delta.go#L20) | [internal/context/context_test.go:151](../../internal/context/context_test.go#L151) | Splits on `\n`; empty trailing line dropped. |
| <a id="s2-n3-m4-p2-mid-file-offset-delta"></a>S2-N3-M4-P2 | Mid-file offset delta | For all transcript paths P with content C and any offset O where 0 ≤ O ≤ len(C), `DeltaReader.Read(P, O)` returns only the lines in C[O:] and len(C) as the new offset. | [internal/context/delta.go:37](../../internal/context/delta.go#L37) | [internal/context/context_test.go:180](../../internal/context/context_test.go#L180) | Stable byte-offset semantics enable incremental reading. |
| <a id="s2-n3-m4-p3-rotation-reset"></a>S2-N3-M4-P3 | Rotation reset | For all transcript paths P with content of length N and any offset O > N, `DeltaReader.Read(P, O)` resets to offset 0 and returns the full file as the delta. | [internal/context/delta.go:33](../../internal/context/delta.go#L33) | [internal/context/context_test.go:218](../../internal/context/context_test.go#L218) | Handles log rotation / truncation gracefully. |
| <a id="s2-n3-m4-p4-empty-file-delta"></a>S2-N3-M4-P4 | Empty file delta | For all transcript paths P pointing to a zero-length file and any offset O, `DeltaReader.Read(P, O)` returns a nil/empty delta and offset 0. | [internal/context/delta.go:28](../../internal/context/delta.go#L28) | [internal/context/context_test.go:244](../../internal/context/context_test.go#L244) | Resets offset because there is nothing to read past. |
| <a id="s2-n3-m4-p5-filereader-error-propagated"></a>S2-N3-M4-P5 | FileReader error propagated | For all errors E returned by the injected `FileReader.Read`, `DeltaReader.Read` returns (nil, 0, E') where E' wraps or matches E. | [internal/context/delta.go:22](../../internal/context/delta.go#L22) | [internal/context/context_test.go:16](../../internal/context/context_test.go#L16) | I/O errors surface to caller; not silently swallowed. |
| <a id="s2-n3-m4-p6-strip-total"></a>S2-N3-M4-P6 | Strip total | For all input lines L (any string content, malformed JSON, binary garbage included), `Strip(L)` returns successfully without panic and without error. | [internal/context/strip.go:16](../../internal/context/strip.go#L16) | **⚠ UNTESTED** | Function has no error return; lack of panic asserted only indirectly by other tests. No targeted fuzz/property test for malformed bytes. |
| <a id="s2-n3-m4-p7-filter-to-user-assistant"></a>S2-N3-M4-P7 | Filter to user/assistant | For all input lines L, `Strip(L)` emits output only for lines whose JSONL `type` or message `role` equals `"user"` or `"assistant"`; all other entry types (progress, system, file-history-snapshot, etc.) are dropped. | [internal/context/strip.go:154](../../internal/context/strip.go#L154) | [internal/context/context_test.go:91](../../internal/context/context_test.go#L91) | Role-only legacy format also accepted (P8). |
| <a id="s2-n3-m4-p8-legacy-role-fallback"></a>S2-N3-M4-P8 | Legacy role fallback | For all input lines L lacking an outer `type` field but carrying `message.role` equal to `"user"` or `"assistant"`, `Strip(L)` still emits the corresponding `USER:` / `ASSISTANT:` line. | [internal/context/strip.go:182](../../internal/context/strip.go#L182) | [internal/context/context_test.go:130](../../internal/context/context_test.go#L130) | Backwards compatibility with older transcript schemas. |
| <a id="s2-n3-m4-p9-drops-tool-blocks-strip"></a>S2-N3-M4-P9 | Drops tool blocks (Strip) | For all input lines L, `Strip(L)` discards every `tool_use` and `tool_result` content block; only `text` blocks contribute to the output. | [internal/context/strip.go:107](../../internal/context/strip.go#L107) | [internal/context/context_test.go:59](../../internal/context/context_test.go#L59), [:359](../../internal/context/context_test.go#L359) | Mixed assistant messages (text + tool_use) keep only the text. |
| <a id="s2-n3-m4-p10-drops-system-reminders"></a>S2-N3-M4-P10 | Drops system-reminders | For all text content T whose trimmed prefix is `<system-reminder`, T is excluded from `Strip` output regardless of whether it appears as a string content or inside a content block. | [internal/context/strip.go:166](../../internal/context/strip.go#L166) | [internal/context/context_test.go:29](../../internal/context/context_test.go#L29), [:14](../../internal/context/stripconfig_test.go#L14) | Mixed blocks: only the system-reminder text is dropped; other text blocks survive. |
| <a id="s2-n3-m4-p11-base64-redaction"></a>S2-N3-M4-P11 | Base64 redaction | For all input lines L containing a contiguous `[A-Za-z0-9+/=]` run of length ≥ 100, `Strip(L)` replaces every such run with the literal `[base64 removed]` before extraction. | [internal/context/strip.go:193](../../internal/context/strip.go#L193) | [internal/context/context_test.go:286](../../internal/context/context_test.go#L286) | Acts on the raw line, so it also redacts base64 inside otherwise-stripped tool blocks. |
| <a id="s2-n3-m4-p12-per-line-truncation"></a>S2-N3-M4-P12 | Per-line truncation | For all extracted lines E produced by `Strip`, `len(E) ≤ 2000 + len("[truncated]")`; any longer extraction is cut to the first 2000 bytes and suffixed with `[truncated]`. | [internal/context/strip.go:198](../../internal/context/strip.go#L198) | [internal/context/context_test.go:309](../../internal/context/context_test.go#L309) | Constant `maxContentBlockLen = 2000`. |
| <a id="s2-n3-m4-p13-role-prefix-correctness"></a>S2-N3-M4-P13 | Role prefix correctness | For all surviving messages M with normalized role R, `Strip` emits exactly one line beginning with `"USER: "` if R = user, or `"ASSISTANT: "` if R = assistant. | [internal/context/strip.go:139](../../internal/context/strip.go#L139) | [internal/context/context_test.go:114](../../internal/context/context_test.go#L114), [:329](../../internal/context/context_test.go#L329), [:344](../../internal/context/context_test.go#L344) | Plain-string content and array-of-blocks content both accepted. |
| <a id="s2-n3-m4-p14-default-config-equals-strip"></a>S2-N3-M4-P14 | Default config equals Strip | For all input lines L, `StripWithConfig(L, StripConfig{})` returns a result equal to `Strip(L)`. | [internal/context/stripconfig.go:49](../../internal/context/stripconfig.go#L49) | [internal/context/stripconfig_test.go:96](../../internal/context/stripconfig_test.go#L96), [:133](../../internal/context/stripconfig_test.go#L133) | Zero-value `StripConfig` is the recall-pipeline default. |
| <a id="s2-n3-m4-p15-keeptoolcalls-preserves-tools"></a>S2-N3-M4-P15 | KeepToolCalls preserves tools | For all input lines L containing `tool_use` and `tool_result` blocks, `StripWithConfig(L, StripConfig{KeepToolCalls: true})` emits a `TOOL_USE [Name]: <args>` line for each tool_use and a `TOOL_RESULT [ok|error]: <content>` line for each tool_result, in source order. | [internal/context/stripconfig.go:62](../../internal/context/stripconfig.go#L62) | [internal/context/stripconfig_test.go:101](../../internal/context/stripconfig_test.go#L101), [:55](../../internal/context/stripconfig_test.go#L55) | `is_error` true → `[error]`; false → `[ok]`. |
| <a id="s2-n3-m4-p16-toolargs-resulttruncate-honored"></a>S2-N3-M4-P16 | ToolArgs/ResultTruncate honored | For all positive integer caps `A = ToolArgsTruncate` and `R = ToolResultTruncate`, args longer than A bytes and result content longer than R bytes are truncated to that length and suffixed with `[truncated]`; zero means no truncation. | [internal/context/stripconfig.go:212](../../internal/context/stripconfig.go#L212), [:230](../../internal/context/stripconfig.go#L230) | [internal/context/stripconfig_test.go:163](../../internal/context/stripconfig_test.go#L163), [:196](../../internal/context/stripconfig_test.go#L196) | Only effective when `KeepToolCalls` is true and `ToolSummaryMode` is false. |
| <a id="s2-n3-m4-p17-toolsummarymode-precedence"></a>S2-N3-M4-P17 | ToolSummaryMode precedence | For all `StripConfig` C where `C.ToolSummaryMode` is true, `StripWithConfig` produces tool-summary output regardless of `C.KeepToolCalls`, `C.ToolArgsTruncate`, or `C.ToolResultTruncate`. | [internal/context/stripconfig.go:45](../../internal/context/stripconfig.go#L45) | [internal/context/stripconfig_test.go:248](../../internal/context/stripconfig_test.go#L248) | Documented in the StripConfig comment. |
| <a id="s2-n3-m4-p18-tool-summary-pairing"></a>S2-N3-M4-P18 | Tool summary pairing | For all tool_use blocks U with id I followed (any distance) by a tool_result block with `tool_use_id` = I, ToolSummaryMode emits exactly one `[tool] <Name>(<args>) → exit <0|1> | <first non-empty line>` line; orphaned tool_use (no matching result) emits nothing. | [internal/context/toolsummary.go:143](../../internal/context/toolsummary.go#L143) | [internal/context/stripconfig_test.go:248](../../internal/context/stripconfig_test.go#L248), [:392](../../internal/context/stripconfig_test.go#L392), [:364](../../internal/context/stripconfig_test.go#L364) | exit 0 when `is_error == false`, exit 1 when true. |
| <a id="s2-n3-m4-p19-tool-summary-120-char-caps"></a>S2-N3-M4-P19 | Tool summary 120-char caps | For all tool summaries produced by ToolSummaryMode, the args portion inside the parentheses and the post-`|` first-line portion are each ≤ 120 + len("[truncated]") bytes. | [internal/context/toolsummary.go:119](../../internal/context/toolsummary.go#L119), [:130](../../internal/context/toolsummary.go#L130) | [internal/context/stripconfig_test.go:222](../../internal/context/stripconfig_test.go#L222), [:414](../../internal/context/stripconfig_test.go#L414) | Constants `toolSummaryArgsCap = 120`, `toolSummaryOutputCap = 120`. |
| <a id="s2-n3-m4-p20-tool-summary-args-determinism"></a>S2-N3-M4-P20 | Tool summary args determinism | For all `tool_use.input` JSON objects, `formatToolSummaryArgs` emits `key=value` pairs in lexicographic order of keys. | [internal/context/toolsummary.go:101](../../internal/context/toolsummary.go#L101) | **⚠ UNTESTED** | Sorted via `sort.Strings`. Determinism is asserted indirectly by `ContainSubstring`-style tests but no test pins the ordering across multiple keys. |
| <a id="s2-n3-m4-p21-tool-summary-first-line-only"></a>S2-N3-M4-P21 | Tool summary first-line only | For all tool_result content blocks C, the post-`|` portion of the emitted summary is the first non-blank line of C (after `strings.TrimSpace`); subsequent lines never appear. | [internal/context/toolsummary.go:73](../../internal/context/toolsummary.go#L73) | [internal/context/stripconfig_test.go:342](../../internal/context/stripconfig_test.go#L342) | Empty content omits the `| …` suffix entirely. |
| <a id="s2-n3-m4-p22-unknown-block-types-ignored"></a>S2-N3-M4-P22 | Unknown block types ignored | For all content blocks B whose `type` is not `text`, `tool_use`, or `tool_result` (e.g. `image`), B contributes nothing to `StripWithConfig` output in any mode. | [internal/context/stripconfig.go:103](../../internal/context/stripconfig.go#L103), [:184](../../internal/context/toolsummary.go#L184) | [internal/context/stripconfig_test.go:38](../../internal/context/stripconfig_test.go#L38) | Forward compatibility with new block types. |
| <a id="s2-n3-m4-p23-malformed-json-dropped"></a>S2-N3-M4-P23 | Malformed JSON dropped | For all input lines L that fail `json.Unmarshal` into `jsonlLine` (or have an empty/unrecognized role), `Strip` and `StripWithConfig` emit no output for L. | [internal/context/strip.go:129](../../internal/context/strip.go#L129), [:147](../../internal/context/stripconfig.go#L147) | [internal/context/stripconfig_test.go:273](../../internal/context/stripconfig_test.go#L273) | Total-function behavior — malformed lines are silently discarded, not errored. |
| <a id="s2-n3-m4-p24-no-direct-i-o-di-only"></a>S2-N3-M4-P24 | No direct I/O (DI-only) | For all package code in `internal/context`, no symbol references `os.*`, `io/ioutil`, `http.*`, `net.*`, or any other I/O primitive directly; the only file access is through the injected `FileReader` interface. | [internal/context/context.go:7](../../internal/context/context.go#L7) | **⚠ UNTESTED** | Architectural invariant from project DI rule (CLAUDE.md "DI everywhere"). No automated guard; would need an import-scanner test or `forbidigo` lint rule. |

## Cross-links

- Parent: [c3-engram-cli-binary.md](c3-engram-cli-binary.md) (refines **S2-N3-M4 · context**)
- Siblings:
  - [c4-anthropic.md](c4-anthropic.md)
  - [c4-cli.md](c4-cli.md)
  - [c4-externalsources.md](c4-externalsources.md)
  - [c4-main.md](c4-main.md)
  - [c4-memory.md](c4-memory.md)
  - [c4-recall.md](c4-recall.md)
  - [c4-tokenresolver.md](c4-tokenresolver.md)
  - [c4-tomlwriter.md](c4-tomlwriter.md)

See `skills/c4/references/property-ledger-format.md` for the full row format and untested-property
discipline.

