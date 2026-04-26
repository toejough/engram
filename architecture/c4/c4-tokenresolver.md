---
level: 4
name: tokenresolver
parent: "c3-engram-cli-binary.md"
children: []
last_reviewed_commit: cd55eab2
---

# C4 — tokenresolver (Property/Invariant Ledger)

> Component in focus: **E27 · tokenresolver** (refines L3 c3-engram-cli-binary.md).
> Source files in scope:
> - [internal/tokenresolver/tokenresolver.go](internal/tokenresolver/tokenresolver.go)
> - [internal/tokenresolver/tokenresolver_test.go](internal/tokenresolver/tokenresolver_test.go)

## Context (from L3)

tokenresolver is the API-token resolution component for the engram CLI. It is consumed by `cli` (E21) before any Anthropic LLM call (R7). The component itself has no outgoing R/D edges to other components — it only invokes its three injected dependencies (`getenv`, `execCmd`, `goos`) which are wired by `cli` at construction time (D1 back-edge). The component encodes one strict architectural invariant: Resolve never returns a non-nil error. All failure modes (env unset, non-darwin OS, Keychain process failure, malformed JSON, missing JSON field) collapse to `("", nil)` so callers can branch on empty-string alone.

![C4 tokenresolver context diagram](svg/c4-tokenresolver.svg)

> Diagram source: [svg/c4-tokenresolver.mmd](svg/c4-tokenresolver.mmd). Re-render with
> `npx @mermaid-js/mermaid-cli -i architecture/c4/svg/c4-tokenresolver.mmd -o architecture/c4/svg/c4-tokenresolver.svg`.
> Pre-rendered because GitHub's Mermaid lacks the ELK layout engine, which is needed to
> separate bidirectional R/D edges between the same node pair.

**Legend:**
- **E27 · tokenresolver** (focus, yellow) — this ledger.
- **E21 · cli** — the wirer; consumer-side R7 plus DI back-edge D1.
- **Process / Keychain** — external resources reached through injected `execCmd` (the `security` binary on darwin).

## Dependency Manifest

Each row is one injected dependency the focus component receives. Manifest expands the
Rdi back-edge into per-dep wiring rows. Reciprocal entries live in the wirer's L4 under
"DI Wires" — those two sections must stay in sync.

| Dep field | Type | Wired by | Concrete adapter | Properties |
|---|---|---|---|---|
| `getenv` | `func(string) string` | [E21 · cli](c3-engram-cli-binary.md#e21-cli) ([c4-cli.md](c4-cli.md)) | `os.Getenv` | P1, P2, P8 |
| `execCmd` | `func(ctx, name, args...) ([]byte, error)` | [E21 · cli](c3-engram-cli-binary.md#e21-cli) ([c4-cli.md](c4-cli.md)) | inline closure wrapping `exec.CommandContext(...).Output()` | P3–P8 |
| `goos` | `string` | [E21 · cli](c3-engram-cli-binary.md#e21-cli) ([c4-cli.md](c4-cli.md)) | `runtime.GOOS` | P3, P8 |

## Property Ledger

| ID | Property | Statement | Enforced at | Tested at | Notes |
|---|---|---|---|---|---|
| <a id="p1-env-precedence"></a>P1 | Env precedence | For all calls to Resolve, if `getenv("ENGRAM_API_TOKEN")` returns a non-empty string, Resolve returns that string and never invokes `execCmd`. | [internal/tokenresolver/tokenresolver.go:34](../../internal/tokenresolver/tokenresolver.go#L34) | [internal/tokenresolver/tokenresolver_test.go:13](../../internal/tokenresolver/tokenresolver_test.go#L13) | Asserts the executor stays uncalled — no Keychain side-effect when env is set. |
| <a id="p2-empty-env-not-used"></a>P2 | Empty env not used | For all calls to Resolve, if `getenv("ENGRAM_API_TOKEN")` returns the empty string, Resolve does not return that empty string as the token; it falls through to the OS-gated Keychain branch. | [internal/tokenresolver/tokenresolver.go:34](../../internal/tokenresolver/tokenresolver.go#L34) | [internal/tokenresolver/tokenresolver_test.go:90](../../internal/tokenresolver/tokenresolver_test.go#L90) | The `if token != ""` guard is what enables every keychain-branch test. |
| <a id="p3-non-darwin-skips-keychain"></a>P3 | Non-darwin skips Keychain | For all calls to Resolve where env is empty and `goos != "darwin"`, Resolve returns `("", nil)` and never invokes `execCmd`. | [internal/tokenresolver/tokenresolver.go:38](../../internal/tokenresolver/tokenresolver.go#L38) | [internal/tokenresolver/tokenresolver_test.go:156](../../internal/tokenresolver/tokenresolver_test.go#L156) | Guards Linux/Windows callers from spurious `security` invocations. |
| <a id="p4-keychain-happy-path"></a>P4 | Keychain happy path | For all calls to Resolve on darwin where env is empty and `execCmd` returns valid JSON containing `claudeAiOauth.accessToken = T`, Resolve returns `(T, nil)`. | [internal/tokenresolver/tokenresolver.go:58](../../internal/tokenresolver/tokenresolver.go#L58) | [internal/tokenresolver/tokenresolver_test.go:90](../../internal/tokenresolver/tokenresolver_test.go#L90) | The single nested-field path documented in `keychainPayload`. |
| <a id="p5-keychain-exec-error-swallowed"></a>P5 | Keychain exec error swallowed | For all calls to Resolve on darwin where `execCmd` returns a non-nil error, Resolve returns `("", nil)` — the error is not propagated. | [internal/tokenresolver/tokenresolver.go:47](../../internal/tokenresolver/tokenresolver.go#L47) | [internal/tokenresolver/tokenresolver_test.go:68](../../internal/tokenresolver/tokenresolver_test.go#L68) | `//nolint:nilerr` documents the intentional swallow. Keychain unavailable ≠ fatal. |
| <a id="p6-malformed-json-swallowed"></a>P6 | Malformed JSON swallowed | For all calls to Resolve on darwin where `execCmd` returns bytes that do not parse as JSON, Resolve returns `("", nil)`. | [internal/tokenresolver/tokenresolver.go:54](../../internal/tokenresolver/tokenresolver.go#L54) | [internal/tokenresolver/tokenresolver_test.go:112](../../internal/tokenresolver/tokenresolver_test.go#L112) | Second `//nolint:nilerr` site. Robustness against future Keychain output drift. |
| <a id="p7-missing-json-field-returns-empty"></a>P7 | Missing JSON field returns empty | For all calls to Resolve on darwin where `execCmd` returns valid JSON without `claudeAiOauth.accessToken`, Resolve returns `("", nil)`. | [internal/tokenresolver/tokenresolver.go:58](../../internal/tokenresolver/tokenresolver.go#L58) | [internal/tokenresolver/tokenresolver_test.go:134](../../internal/tokenresolver/tokenresolver_test.go#L134) | Zero-value of the typed payload field is the empty string. Also covers explicit empty-string accessToken (test at line 46). |
| <a id="p8-resolve-never-errors"></a>P8 | Resolve never errors | For all inputs and all DI configurations, Resolve returns a nil error. | [internal/tokenresolver/tokenresolver.go:33](../../internal/tokenresolver/tokenresolver.go#L33), [:47](../../internal/tokenresolver/tokenresolver.go#L47), [:54](../../internal/tokenresolver/tokenresolver.go#L54) | [internal/tokenresolver/tokenresolver_test.go:35](../../internal/tokenresolver/tokenresolver_test.go#L35), [:59](../../internal/tokenresolver/tokenresolver_test.go#L59), [:81](../../internal/tokenresolver/tokenresolver_test.go#L81), [:102](../../internal/tokenresolver/tokenresolver_test.go#L102), [:125](../../internal/tokenresolver/tokenresolver_test.go#L125), [:146](../../internal/tokenresolver/tokenresolver_test.go#L146), [:172](../../internal/tokenresolver/tokenresolver_test.go#L172) | Architectural invariant: the doc comment guarantees Resolve's signature degrades to ("", nil) on every failure path. Every test asserts `err NotTo HaveOccurred`. Callers branch on empty-string alone. |
| <a id="p9-no-direct-i-o"></a>P9 | No direct I/O | For all code paths in package tokenresolver, no function calls `os.*`, `exec.*`, `runtime.*`, or any other I/O package directly; all OS interaction is mediated by injected `getenv`, `execCmd`, and `goos`. | [internal/tokenresolver/tokenresolver.go:11](../../internal/tokenresolver/tokenresolver.go#L11) | **⚠ UNTESTED** | Architectural invariant aligned with the project's DI-everywhere rule. UNTESTED — there is no static-analysis check that the package's import set excludes `os`/`exec`/`runtime`. |
| <a id="p10-keychain-query-parameters-fixed"></a>P10 | Keychain query parameters fixed | For all calls into resolveFromKeychain, `execCmd` is invoked with arguments `("security", "find-generic-password", "-s", "Claude Code-credentials", "-w")`. | [internal/tokenresolver/tokenresolver.go:46](../../internal/tokenresolver/tokenresolver.go#L46) | **⚠ UNTESTED** | UNTESTED — existing tests ignore the args passed to execCmd. A drift in the service name would silently break Keychain lookup. |
| <a id="p11-context-propagation"></a>P11 | Context propagation | For all calls Resolve(ctx), the same ctx is forwarded into every `execCmd` invocation, so a cancelled context cancels the underlying `security` process. | [internal/tokenresolver/tokenresolver.go:46](../../internal/tokenresolver/tokenresolver.go#L46) | **⚠ UNTESTED** | UNTESTED — no test cancels ctx and asserts execCmd received the cancelled context. |

## Cross-links

- Parent: [c3-engram-cli-binary.md](c3-engram-cli-binary.md) (refines **E27 · tokenresolver**)
- Siblings:
  - [c4-anthropic.md](c4-anthropic.md)
  - [c4-cli.md](c4-cli.md)
  - [c4-context.md](c4-context.md)
  - [c4-externalsources.md](c4-externalsources.md)
  - [c4-main.md](c4-main.md)
  - [c4-memory.md](c4-memory.md)
  - [c4-recall.md](c4-recall.md)
  - [c4-tomlwriter.md](c4-tomlwriter.md)

See `skills/c4/references/property-ledger-format.md` for the full row format and untested-property
discipline.

