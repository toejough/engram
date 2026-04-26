---
level: 4
name: anthropic
parent: "c3-engram-cli-binary.md"
children: []
last_reviewed_commit: 6002fa69
---

# C4 — anthropic (Property/Invariant Ledger)

> Component in focus: **E26 · anthropic** (refines L3 c3-engram-cli-binary).
> Source files in scope:
> - [../../internal/anthropic/anthropic.go](../../internal/anthropic/anthropic.go)
> - [../../internal/anthropic/anthropic_test.go](../../internal/anthropic/anthropic_test.go)

## Context (from L3)

Scoped slice of [c3-engram-cli-binary.md](c3-engram-cli-binary.md): the L3 edges that touch
E26. The DI back-edge convention applies — E26 → E21 represents the category of calls E26
makes through the `HTTPDoer` wired by E21.

![C4 anthropic context diagram](svg/c4-anthropic.svg)

> Diagram source: [svg/c4-anthropic.mmd](svg/c4-anthropic.mmd). Re-render with
> `npx @mermaid-js/mermaid-cli -i architecture/c4/svg/c4-anthropic.mmd -o architecture/c4/svg/c4-anthropic.svg`.
> Pre-rendered because GitHub's Mermaid lacks the ELK layout engine, which is needed to
> separate bidirectional R/D edges between the same node pair.

## Dependency Manifest

| Dep field | Type | Wired by | Concrete adapter | Properties |
|---|---|---|---|---|
| `client` | `HTTPDoer` (`Do(req) (*http.Response, error)`) | [E21 · cli](c3-engram-cli-binary.md#e21-cli) ([c4-cli.md](c4-cli.md)) | `&http.Client{}` at [internal/cli/cli.go:131](../../internal/cli/cli.go#L131) | P3, P4, P5, P6, P7, P8 |
| `token` | `string` | [E21 · cli](c3-engram-cli-binary.md#e21-cli) ([c4-cli.md](c4-cli.md)) | resolved via `tokenresolver.Resolve` at [internal/cli/cli.go:165](../../internal/cli/cli.go#L165) | P1, P9 |
| `apiURL` | `string` (settable) | [E21 · cli](c3-engram-cli-binary.md#e21-cli) ([c4-cli.md](c4-cli.md)) | `cli.AnthropicAPIURL` global at [internal/cli/cli.go:23](../../internal/cli/cli.go#L23) via `SetAPIURL` | P9 |

## Property Ledger

| ID | Property | Statement | Enforced at | Tested at | Notes |
|---|---|---|---|---|---|
| <a id="p1-no-token"></a>P1 | Empty token returns ErrNoToken | For all `Call(...)` invocations on a `Client` whose `token` is `""`, the call returns `("", ErrNoToken)` without invoking the injected `HTTPDoer`. | [internal/anthropic/anthropic.go:55](../../internal/anthropic/anthropic.go#L55) | [internal/anthropic/anthropic_test.go:116](../../internal/anthropic/anthropic_test.go#L116) | Sentinel exported as `ErrNoToken`. |
| <a id="p2-haiku-pinned"></a>P2 | Haiku model pinned | For all callers using `HaikuModel`, the model identifier resolves to `"claude-haiku-4-5-20251001"`. | [internal/anthropic/anthropic.go:17](../../internal/anthropic/anthropic.go#L17) | **⚠ UNTESTED** | Architectural pin documented in L3 catalog row for E26. |
| <a id="p3-headers"></a>P3 | Required headers set | For all successful (non-empty-token) `Call(...)` invocations, the outgoing `*http.Request` carries `Authorization: Bearer <token>`, `Anthropic-Version: 2023-06-01`, `Anthropic-Beta: oauth-2025-04-20`, and `Content-Type: application/json`. | [internal/anthropic/anthropic.go:107](../../internal/anthropic/anthropic.go#L107) | [internal/anthropic/anthropic_test.go:145](../../internal/anthropic/anthropic_test.go#L145) | Beta/version constants are unexported. |
| <a id="p4-non2xx-wraps-errapi"></a>P4 | Non-2xx wraps ErrAPIError | For all responses with `StatusCode < 200` or `StatusCode >= 300`, `Call(...)` returns an error wrapping `ErrAPIError` with the status code. | [internal/anthropic/anthropic.go:128](../../internal/anthropic/anthropic.go#L128) | [internal/anthropic/anthropic_test.go:16](../../internal/anthropic/anthropic_test.go#L16), [:38](../../internal/anthropic/anthropic_test.go#L38), [:58](../../internal/anthropic/anthropic_test.go#L58) | Both JSON and non-JSON error bodies covered. |
| <a id="p5-nil-response"></a>P5 | Nil HTTP response → ErrNilResponse | For all `HTTPDoer.Do` returns where err is nil and resp is nil, `Call(...)` returns an error wrapping `ErrNilResponse`. | [internal/anthropic/anthropic.go:117](../../internal/anthropic/anthropic.go#L117) | [internal/anthropic/anthropic_test.go:105](../../internal/anthropic/anthropic_test.go#L105) | Defensive guard for fakes. |
| <a id="p6-empty-content"></a>P6 | Empty content → ErrNoContentBlocks | For all 2xx response bodies decoding to a `response{Content: []}`, `Call(...)` returns `("", err)` wrapping `ErrNoContentBlocks`. | [internal/anthropic/anthropic.go:239](../../internal/anthropic/anthropic.go#L239) | [internal/anthropic/anthropic_test.go:79](../../internal/anthropic/anthropic_test.go#L79) | Returns first content block's `Text` on success. |
| <a id="p7-success-returns-text"></a>P7 | Success returns first block text | For all 2xx response bodies of shape `{"content":[{"type":"text","text":<S>}, ...]}`, `Call(...)` returns `(S, nil)`. | [internal/anthropic/anthropic.go:243](../../internal/anthropic/anthropic.go#L243) | [internal/anthropic/anthropic_test.go:126](../../internal/anthropic/anthropic_test.go#L126) | Only first block consumed. |
| <a id="p8-caller-delegates"></a>P8 | Caller delegates to Call | For all `Caller(maxTokens)` invocations followed by the returned function being called with `(ctx, model, system, user)`, the underlying `Call` is invoked with those exact arguments and `maxTokens`. | [internal/anthropic/anthropic.go:68](../../internal/anthropic/anthropic.go#L68) | [internal/anthropic/anthropic_test.go:173](../../internal/anthropic/anthropic_test.go#L173) | Adapter producing `CallerFunc` for downstream consumers (recall.NewSummarizer). |
| <a id="p9-setapiurl"></a>P9 | SetAPIURL overrides endpoint | For all `Client` instances, after `SetAPIURL(url)` the next `Call(...)` posts to `url` instead of the default `https://api.anthropic.com/v1/messages`. | [internal/anthropic/anthropic.go:75](../../internal/anthropic/anthropic.go#L75) | [internal/anthropic/anthropic_test.go:199](../../internal/anthropic/anthropic_test.go#L199) | Used by cli for test overrides. |
| <a id="p10-strip-code-fences"></a>P10 | StripCodeFences extracts JSON | For all input strings, `StripCodeFences(s)` returns the substring inside the first \`\`\`json or \`\`\` fence if present; otherwise the substring from the first `{` or `[` to the matching last `}` or `]`; otherwise the trimmed input. | [internal/anthropic/anthropic.go:142](../../internal/anthropic/anthropic.go#L142) | [internal/anthropic/anthropic_test.go:226](../../internal/anthropic/anthropic_test.go#L226) | Pure helper used by callers post-decode. |
| <a id="p11-marshal-uses-snake"></a>P11 | Request uses snake_case JSON | For all outgoing requests, the body fields are serialized as `model`, `max_tokens`, `system`, `messages` per Anthropic API spec. | [internal/anthropic/anthropic.go:205](../../internal/anthropic/anthropic.go#L205) | **⚠ UNTESTED** | Documented via `//nolint:tagliatelle`; no test asserts the exact JSON body shape. |
| <a id="p12-context-propagated"></a>P12 | Context plumbed to HTTP | For all `Call(ctx, ...)` invocations, the resulting `*http.Request` is constructed via `http.NewRequestWithContext(ctx, ...)` so cancellation cancels the in-flight request. | [internal/anthropic/anthropic.go:97](../../internal/anthropic/anthropic.go#L97) | **⚠ UNTESTED** | Architectural rule from `.claude/rules/go.md`. |

## Cross-links

- Parent: [c3-engram-cli-binary.md](c3-engram-cli-binary.md) (refines **E26 · anthropic**)

See `skills/c4/references/property-ledger-format.md` for the full row format and untested-property
discipline.
