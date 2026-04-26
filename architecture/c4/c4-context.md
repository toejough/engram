---
level: 4
name: context
parent: "c3-engram-cli-binary.md"
children: []
last_reviewed_commit: 6002fa69
---

# C4 — context (Property/Invariant Ledger)

> Component in focus: **E23 · context** (refines L3 c3-engram-cli-binary).
> Source files in scope:
> - [../../internal/context/context.go](../../internal/context/context.go)
> - [../../internal/context/strip.go](../../internal/context/strip.go)
> - [../../internal/context/stripconfig.go](../../internal/context/stripconfig.go)
> - [../../internal/context/toolsummary.go](../../internal/context/toolsummary.go)
> - [../../internal/context/delta.go](../../internal/context/delta.go)
> - [../../internal/context/context_test.go](../../internal/context/context_test.go)
> - [../../internal/context/stripconfig_test.go](../../internal/context/stripconfig_test.go)

## Context (from L3)

Scoped slice of [c3-engram-cli-binary.md](c3-engram-cli-binary.md): the L3 edge that
touches E23 is R8 (recall reads + strips session transcripts within budget). The DI
back-edge convention applies: E23's `DeltaReader` consumes a `FileReader` interface wired
by E21 cli with `os.ReadFile`.

![C4 context context diagram](svg/c4-context.svg)

> Diagram source: [svg/c4-context.mmd](svg/c4-context.mmd). Re-render with
> `npx @mermaid-js/mermaid-cli -i architecture/c4/svg/c4-context.mmd -o architecture/c4/svg/c4-context.svg`.
> Pre-rendered because GitHub's Mermaid lacks the ELK layout engine, which is needed to
> separate bidirectional R/D edges between the same node pair.

## Dependency Manifest

Each row is one injected dependency a focus type receives. Manifest expands the D6
back-edge into per-dep wiring rows. Reciprocal entries live in the wirer's L4 under
"DI Wires" — those two sections must stay in sync.

| Dep field | Type | Wired by | Concrete adapter | Properties |
|---|---|---|---|---|
| `DeltaReader.reader` | `context.FileReader` (`Read(path) ([]byte, error)`) | [E21 · cli](c3-engram-cli-binary.md#e21-cli) (L4: c4-cli.md — TBD) | `os.ReadFile`-backed adapter | P7, P8, P9 |

## Property Ledger

| ID | Property | Statement | Enforced at | Tested at | Notes |
|---|---|---|---|---|---|
| <a id="p1-strip-total"></a>P1 | Strip is total | For all input slices `lines []string`, `Strip(lines)` returns a `[]string` without panicking and without returning an error. | [internal/context/strip.go:16](../../internal/context/strip.go#L16) | [internal/context/context_test.go:266](../../internal/context/context_test.go#L266), [:286](../../internal/context/context_test.go#L286), [:309](../../internal/context/context_test.go#L309) | No error in the signature; malformed JSON, base64 blobs, oversized lines all handled. |
| <a id="p2-keeps-only-user-assistant"></a>P2 | Keeps only user/assistant entries | For all JSONL lines whose `type` (or legacy `role`) is not `"user"` or `"assistant"`, `Strip` produces no output for that line. | [internal/context/strip.go:154](../../internal/context/strip.go#L154) | [internal/context/context_test.go:91](../../internal/context/context_test.go#L91) | Progress / system / tool-call types are dropped at the kept-type filter. |
| <a id="p3-drops-system-reminders"></a>P3 | Drops system-reminder content | For all text blocks whose trimmed prefix is `"<system-reminder"`, `Strip` produces no output for that block. | [internal/context/strip.go:166](../../internal/context/strip.go#L166) | [internal/context/context_test.go:29](../../internal/context/context_test.go#L29) | Both string-content and array-content cases covered. |
| <a id="p4-base64-replaced"></a>P4 | Long base64 redacted | For all sequences of ≥100 chars matching `[A-Za-z0-9+/=]+`, `Strip` replaces the sequence with `[base64 removed]` before downstream parsing. | [internal/context/strip.go:193](../../internal/context/strip.go#L193) | [internal/context/context_test.go:286](../../internal/context/context_test.go#L286) | Threshold is `minBase64Len = 100`. |
| <a id="p5-line-budget"></a>P5 | Per-line length budget | For all extracted lines, the result is at most `maxContentBlockLen = 2000` chars + the `[truncated]` placeholder. | [internal/context/strip.go:198](../../internal/context/strip.go#L198) | [internal/context/context_test.go:309](../../internal/context/context_test.go#L309) | Truncation appends a literal placeholder, not a hard cut. |
| <a id="p6-strip-config-superset"></a>P6 | StripWithConfig superset | For all inputs `(lines, cfg)` where `cfg.KeepToolCalls == false && cfg.ToolSummaryMode == false`, `StripWithConfig(lines, cfg)` returns the same value as `Strip(lines)`. | [internal/context/stripconfig.go:49](../../internal/context/stripconfig.go#L49) | [internal/context/stripconfig_test.go:14](../../internal/context/stripconfig_test.go#L14) | Default config delegates straight to `Strip`. |
| <a id="p7-delta-totality"></a>P7 | Delta read totality (DI errors propagate) | For all `(path, offset)` pairs, `DeltaReader.Read` returns either the new lines plus updated offset on success, or `(nil, 0, err)` wrapping the injected `FileReader` error. No other failure modes. | [internal/context/delta.go:20](../../internal/context/delta.go#L20) | [internal/context/context_test.go:16](../../internal/context/context_test.go#L16) | The pure-logic layer never touches the OS directly. |
| <a id="p8-rotation-reset"></a>P8 | Rotation reset on shrunk file | For all `(path, offset)` pairs where the file is shorter than `offset` (e.g. log rotation), `DeltaReader.Read` resets to offset 0 and returns the entire content. | [internal/context/delta.go:33](../../internal/context/delta.go#L33) | [internal/context/context_test.go:218](../../internal/context/context_test.go#L218) | — |
| <a id="p9-empty-file-zero-offset"></a>P9 | Empty file returns zero offset | For all reads of an empty file, `DeltaReader.Read` returns `(nil, 0, nil)`. | [internal/context/delta.go:28](../../internal/context/delta.go#L28) | [internal/context/context_test.go:244](../../internal/context/context_test.go#L244) | Avoids advancing the persisted offset past file end. |
| <a id="p10-tool-summary-pairs"></a>P10 | Tool summary pairs by ID | For all `tool_use` blocks with id `X` followed by a `tool_result` block whose `tool_use_id` is `X`, `StripWithConfig` in `ToolSummaryMode` emits exactly one summary line per pair. Orphans (no matching id) are dropped silently. | [internal/context/toolsummary.go:218](../../internal/context/toolsummary.go#L218), [:143](../../internal/context/toolsummary.go#L143) | [internal/context/stripconfig_test.go:248](../../internal/context/stripconfig_test.go#L248), [:392](../../internal/context/stripconfig_test.go#L392) | Pending map is keyed by `tool_use_id`. |
| <a id="p11-summary-budget"></a>P11 | Summary args/output capped at 120 | For all summary lines, the args section is capped at `toolSummaryArgsCap = 120` chars and the first-output-line is capped at `toolSummaryOutputCap = 120` chars (each truncation appends `[truncated]`). | [internal/context/toolsummary.go:11](../../internal/context/toolsummary.go#L11), [:119](../../internal/context/toolsummary.go#L119), [:130](../../internal/context/toolsummary.go#L130) | [internal/context/stripconfig_test.go:223](../../internal/context/stripconfig_test.go#L223), [:414](../../internal/context/stripconfig_test.go#L414) | Fixed-cap budgets independent of `cfg.ToolArgsTruncate`. |
| <a id="p12-no-direct-io"></a>P12 | No direct I/O (DI-only) | For all package code, no symbol references `os.ReadFile`, `os.Open`, `net/http`, or any other process/OS facility directly; transcript bytes flow exclusively through the injected `FileReader` interface. | [internal/context/context.go:7](../../internal/context/context.go#L7) | **⚠ UNTESTED** | Architectural invariant from project DI rule (CLAUDE.md "DI everywhere"). |

## Cross-links

- Parent: [c3-engram-cli-binary.md](c3-engram-cli-binary.md) (refines **E23 · context**)
- Siblings:
  - [c4-main.md](c4-main.md)
  - [c4-memory.md](c4-memory.md)
  - [c4-tokenresolver.md](c4-tokenresolver.md)

See `skills/c4/references/property-ledger-format.md` for the full row format and untested-property
discipline.
