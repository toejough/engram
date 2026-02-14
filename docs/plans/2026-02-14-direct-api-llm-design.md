# ISSUE-211: Direct API LLM Client

## Problem

`ClaudeCLIExtractor` spawns a new `claude --print` subprocess for every LLM operation. Benchmark results show ~59s per call, of which ~58s is CLI/Node.js startup overhead and <1s is actual API round-trip.

During `projctl memory optimize`, 5-15+ sequential LLM calls make the pipeline take 5-15 minutes. Hook-based calls (session extraction, curation) similarly suffer.

## Benchmark Results

| Approach | Total (3 calls) | Per-call avg | vs Baseline |
|----------|-----------------|-------------|-------------|
| baseline (sequential CLI) | 2m56s | ~59s | -- |
| parallel CLI | 55s | ~55s | 3.2x wall-clock |
| interactive CLI (stream-json) | 2m14s | ~45s | 1.3x |
| **direct API** | **2.2s** | **~730ms** | **80x** |

Model comparison (single CLI call each): haiku ~60s, sonnet ~67s, opus ~62s. Model choice is irrelevant -- CLI overhead dominates.

## Solution

Replace `ClaudeCLIExtractor` with `DirectAPIExtractor` that calls `https://api.anthropic.com/v1/messages` directly from Go using `net/http`.

### Authentication

The Claude Code CLI stores an OAuth token in macOS Keychain:
- Service: `Claude Code-credentials`
- Account: `$USER`
- Format: JSON with `claudeAiOauth.accessToken` (type `sk-ant-oat01-...`)

The OAuth token works on `/v1/messages` with a required beta header:
```
Authorization: Bearer sk-ant-oat01-...
anthropic-beta: oauth-2025-04-20
anthropic-version: 2023-06-01
```

### Token Lifecycle

- `expiresAt` field in the credential JSON provides expiry timestamp
- If expired: fall back to `ClaudeCLIExtractor`
- Future: implement refresh via `refreshToken` + `/v1/oauth/token` endpoint

### Fallback

If any of these fail, transparently fall back to `ClaudeCLIExtractor`:
1. Keychain read fails (not macOS, no entry, permission denied)
2. Token expired
3. API returns 401 (token revoked, beta header changed)

### Components

1. **`internal/memory/auth.go`** -- Keychain token extraction, expiry check
2. **`internal/memory/llm_api.go`** -- `DirectAPIExtractor` implementing `LLMExtractor`, `SkillCompiler`, `SpecificityDetector`
3. **`cmd/projctl/memory_optimize.go`** -- Wire direct API as default with CLI fallback
4. **Tests** -- Mock HTTP server, keychain stub via `CommandRunner` pattern

### Interface

`DirectAPIExtractor` implements all three interfaces that `ClaudeCLIExtractor` already implements:
- `LLMExtractor` (Extract, Synthesize, Curate, Decide, Rewrite, AddRationale)
- `SkillCompiler` (CompileSkill, Synthesize)
- `SpecificityDetector` (IsNarrowLearning)

Same prompts, same JSON parsing, different transport.

## Non-Goals

- Token refresh automation (future work)
- Linux/Windows keychain support (future work -- plaintext fallback file exists at `~/.claude/.credentials.json`)
- Parallelizing optimize pipeline steps (separate issue, now less critical)
- Changing the LLM prompts or response handling

## Acceptance Criteria

- [ ] `projctl memory optimize` completes LLM calls at <2s per call (vs ~60s)
- [ ] Ctrl-C propagates via context cancellation
- [ ] Falls back to CLI if Keychain/API unavailable
- [ ] Existing tests pass unchanged
- [ ] New tests cover: auth extraction, API call, fallback behavior

## References

- Benchmark tool: `dev/llmbench/main.go`
- OAuth beta header: `anthropic-beta: oauth-2025-04-20`
- Keychain service: `Claude Code-credentials`
