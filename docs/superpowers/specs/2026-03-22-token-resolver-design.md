# Token Resolver Design

**Issue:** #363 — Recall skill should not require OAuth token sourcing in bash

## Problem

Three hooks (`stop.sh`, `user-prompt-submit.sh`, `pre-tool-use.sh`) duplicate the same macOS Keychain lookup pattern to set `ENGRAM_API_TOKEN` before calling the Go binary. The binary just reads `os.Getenv`. This is fragile, duplicated, and means standalone CLI usage requires manual token setup.

## Design

### New package: `internal/tokenresolver/`

A `Resolver` interface:

```go
type Resolver interface {
    Resolve(ctx context.Context) (string, error)
}
```

Default implementation chains two strategies in order:
1. **Env var:** Read `ENGRAM_API_TOKEN` — fast path, backwards-compatible
2. **macOS Keychain (Darwin only):** Skip on non-macOS. On macOS: `security find-generic-password -s "Claude Code-credentials" -w` → parse JSON → extract `.claudeAiOauth.accessToken`
3. If both empty (including `accessToken` present but `""`), return `""` with no error (some commands work without a token)

Keychain errors (access denied, command not found) are treated as "no token available" — not propagated as errors. This preserves graceful degradation for commands that handle empty tokens (`recall`, `instruct`, `maintain`) and lets commands that require tokens fail at the API call point with a clear error.

DI: env reader (`func(string) string`) and command executor (`func(ctx, name, args) ([]byte, error)`) are injected so tests don't touch real Keychain or env.

Token is resolved **once per binary invocation** — multiple LLM calls within one invocation reuse the same token.

### Wiring in `cli.go`

Each `run*` function that currently does `os.Getenv("ENGRAM_API_TOKEN")` calls `resolver.Resolve(ctx)` instead. The resolved token string flows into `makeAnthropicCaller` as before.

Affected: `runRecall`, `runCorrect`, `runLearn`, `runMaintain`, `runInstruct`, `runFlush` (calls `RunLearn` internally — resolves once and passes token down).
Not affected: `runSurface`, `runFeedback`, `runShow`, `runReview`.

### Hook cleanup

Remove the Keychain lookup block from:
- `hooks/stop.sh`
- `hooks/user-prompt-submit.sh`
- `hooks/pre-tool-use.sh`

These hooks just call the binary — the binary handles its own credentials now.

### Not changing

- `makeAnthropicCaller` / `callAnthropicAPI` — still takes a token string
- Commands that don't need a token — they don't call the resolver
- The recall skill — already had OAuth block removed in the #361 fix

## Testing

- Unit tests for resolver: env var present → returns it; env var empty + mock Keychain → returns extracted token; both empty → returns ""
- Keychain JSON parsing: valid JSON, missing field, malformed JSON, `accessToken` present but empty string
- Keychain errors: command fails (access denied, not found) → returns "" not error
- Non-macOS: Keychain strategy skipped entirely → returns "" if env var also empty
- Integration: existing tests for commands that use tokens continue to pass (they inject tokens via env or mocks)
