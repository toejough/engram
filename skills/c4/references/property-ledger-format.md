# L4 Property Ledger Format

L4 (Code) does NOT use UML. It uses a property/invariant ledger: a table of universally-quantified
guarantees the code provides, each linked to the code that enforces it and the test that validates it.

## Why properties, not classes

- UML class diagrams go stale fast and IDEs already show structure.
- The durable, decision-relevant thing is the **contract**: what does this code guarantee for ALL
  inputs? That contract survives refactors.
- Test gaps become visible when every untested property is explicitly flagged.

## Row Format

Every row in an L4 ledger has these columns:

| Column | Content |
|---|---|
| Property | A short name (≤6 words). |
| Statement | A universally-quantified claim ("for all X, …"). |
| Enforced at | Markdown link to `file:line` where the property is established. |
| Tested at | Markdown link to `test_file:line`, OR **⚠ UNTESTED** if no test exists. |
| Notes | Caveats, edge-case carve-outs, related properties. |

## Statement Style

- Begin with "For all …" — make the universal quantifier explicit.
- State the GUARANTEE, not the implementation. ("For all valid inputs, the function returns within
  500ms" is a guarantee. "The function uses a cache" is implementation.)
- One property per row. Don't smuggle two properties into one statement with "and".
- Negative properties are allowed: "For all inputs, the function never panics."
- Probabilistic guarantees include the bound: "For all inputs, the cache hit rate is ≥ 0.9 after
  warm-up."

## Examples

| Property | Statement | Enforced at | Tested at | Notes |
|---|---|---|---|---|
| Idempotent learn | For all transcripts T and offsets O, calling `RunIncremental(T, O)` twice consecutively produces identical state and zero additional LLM calls on the second call. | [internal/learn/incremental.go:42](../../internal/learn/incremental.go#L42) | [internal/learn/incremental_test.go:88](../../internal/learn/incremental_test.go#L88) | Relies on offset persistence after first call. |
| Bounded surface output | For all queries Q, `engram surface --format json` returns ≤ `--budget-bytes` bytes of context. | [internal/surface/budget.go:31](../../internal/surface/budget.go#L31) | **⚠ UNTESTED** | No test exercises the budget cap with real surfaces. |
| Strip is total | For all transcript bytes B, `Strip(B)` returns successfully (no panic, no error). | [internal/context/strip.go:17](../../internal/context/strip.go#L17) | [internal/context/strip_test.go:54](../../internal/context/strip_test.go#L54) | Includes malformed JSON, binary blobs. |

## Untested-Property Discipline

- Never invent a test pointer. If no test exists, the row is **⚠ UNTESTED**.
- Untested rows are a feature, not a bug — they make coverage gaps visible at the architecture
  layer where they get triaged.
- Do NOT remove an untested property to make the ledger "cleaner". The skill must surface it.

## Anti-Patterns

| Anti-pattern | Fix |
|---|---|
| "The function handles errors gracefully" | State which errors and what "gracefully" means: "For all errors returned by io.Read, the function wraps them with context and returns." |
| "Returns the right answer" | State the answer in terms of inputs: "For all sorted inputs S and target T, returns the index i such that S[i] == T or -1 if absent." |
| Mixing implementation with guarantee | Split: implementation goes in Notes; the row's Statement is the guarantee. |

## Dependency Manifest (consumer-side)

When the focus component receives DI dependencies (function values, interfaces) from a wirer,
add a `## Dependency Manifest` section listing each DI seam. The wiring diagram (companion
to the call diagram — see `mermaid-conventions.md`) is derived from this table by grouping
rows by `(wired_by_id, wrapped_entity_id)`.

| Column | Content |
|---|---|
| Field | The struct field or constructor parameter name (e.g., `finder`, `summarizer`). |
| Type | Go type signature or interface name (`Finder`, `func(string) string`). |
| Wired by | Markdown link to the wirer's L3 catalog row + the wirer's L4 ledger (or `TBD` if not yet drafted). |
| Wrapped entity | SNM ID of the diagram node (component or external) the seam ultimately drives behavior against. **Must match a node on the call diagram** — `targ c4-l4-build` rejects manifests that violate this. |
| Properties | Comma-separated P-IDs from this same ledger that depend on this seam. Use range notation for contiguous runs. |

Example (from c4-recall.md, where recall is `S2-N3-M3`):

| Field | Type | Wired by | Wrapped entity | Properties |
|---|---|---|---|---|
| `finder` | `Finder` | [M2 · cli](c3-engram-cli-binary.md#m2-cli) ([c4-cli.md](c4-cli.md)) | `S3` | S2-N3-M3-P1, S2-N3-M3-P2, S2-N3-M3-P9 |
| `summarizer` | `SummarizerI` | [M2 · cli](c3-engram-cli-binary.md#m2-cli) ([c4-cli.md](c4-cli.md)) | `S2-N3-M7` | S2-N3-M3-P5–P8, S2-N3-M3-P11–P16 |
| `memoryLister` | `MemoryLister` | [M2 · cli](c3-engram-cli-binary.md#m2-cli) ([c4-cli.md](c4-cli.md)) | `S2-N3-M5` | S2-N3-M3-P11–P13, S2-N3-M3-P15 |

The wrapped-entity column distinguishes "I depend on `Finder`" (the seam) from "behavior
ultimately runs against the OS filesystem / Claude Code" (the wrapped entity, `S3`). Two
seams sharing a wrapped entity collapse to one wiring edge.

## DI Wires (provider-side)

When the focus component **wires** dependencies into other components — a composition root or
adapter-shim owner — add a `## DI Wires` section listing every DI seam wired *for others*.
This is the reciprocal of consumer-side Dependency Manifests; the audit checks symmetry —
every Dependency Manifest row in a consumer must have a matching DI Wires row in the wirer.

| Column | Content |
|---|---|
| Field | The seam's field name in the consumer's struct (matches that consumer's Dependency Manifest row). |
| Type | Go type signature or interface name (matches the consumer's Dependency Manifest row). |
| Consumer | Markdown link to consumer's L3 catalog row + L4 ledger. |
| Wrapped entity | SNM ID of the diagram node behavior ultimately drives against — same value as the consumer's row. |

## Untested-Property Discipline (revisited under DI)

A DI dependency is part of the test seam. If a property is asserted by a unit test that injects
mocks for the deps, that property is **tested** — not "tested via real I/O" but tested at the
seam the property is about. Don't downgrade a unit-tested-via-DI property to UNTESTED just
because no integration test exists. If you want integration coverage as a separate concern,
add a separate property like "For all real-OS environments, the resolver…" and mark *that*
UNTESTED if no integration harness exists.
