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
