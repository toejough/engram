# Architecture

System architecture for UC-3 (Remember & Correct). Each ARCH decision traces to L2 items.

---

## ARCH-1: Pipeline Architecture

**Decision:** Linear pipeline of injected stages:

```go
type Corrector struct {
    Corpus   PatternMatcher  // deterministic: message → match or nil
    Enricher Enricher        // LLM: message → EnrichedMemory (or degraded)
    Writer   MemoryWriter    // file I/O: EnrichedMemory → file path
    Renderer Renderer        // format: EnrichedMemory + path → system reminder text
}

func (c *Corrector) Run(ctx context.Context, message string) (string, error) {
    // 1. Corpus.Match(message): check patterns, return match or nil
    // 2. If no match: return "" (empty stdout)
    // 3. Enricher.Enrich(ctx, message, match): LLM call → EnrichedMemory
    //    (falls back to degraded memory if no API key)
    // 4. Writer.Write(memory): write TOML file, return file path
    // 5. Renderer.Render(memory, path): build system reminder text
    // 6. Return system reminder text
}
```

Four stages, each independently testable via DI. The pipeline is the composition root's responsibility to wire.

**Traces to:** REQ-1 (detection), REQ-2 (enrichment), REQ-3 (file writing), REQ-4 (feedback), REQ-6 (Go binary)

---

## ARCH-2: Pattern Matching

**Decision:** Compiled regex patterns, embedded in the binary:

```go
type PatternMatcher interface {
    Match(message string) *PatternMatch
}

type PatternMatch struct {
    Pattern    string // the regex that matched
    Label      string // human-readable label (e.g., "direct-negation")
    Confidence string // "A" for remember patterns, "B" for correction patterns
}
```

The 40 patterns from REQ-1 are compiled at init time. `Match` returns the first match (sequential scan) or nil. Pattern order doesn't matter for correctness — any match triggers enrichment.

Confidence assignment per REQ-7: `\bremember\s+(that|to)` and `\bfrom\s+now\s+on\b` → "A", all others → "B".

**Traces to:** REQ-1 (pattern matching), REQ-7 (confidence tiers)

---

## ARCH-3: LLM Enrichment via Anthropic API

**Decision:** Direct HTTP client to `api.anthropic.com/v1/messages`:

```go
type Enricher interface {
    Enrich(ctx context.Context, message string, match *PatternMatch) (*EnrichedMemory, error)
}

type EnrichedMemory struct {
    Title           string
    Content         string
    ObservationType string
    Concepts        []string
    Keywords        []string
    Principle       string
    AntiPattern     string
    Rationale       string
    FilenameSummary string // 3-5 words for slug
    Confidence      string
    CreatedAt       time.Time
    UpdatedAt       time.Time
}
```

Implementation sends a single `messages` API call to `claude-haiku-4-5-20251001` with a system prompt instructing JSON output of the structured fields. OAuth token from `ENGRAM_API_TOKEN` env var, sent as `Authorization: Bearer` header with `Anthropic-Beta: oauth-2025-04-20`. The hook script reads the token from the Claude Code Keychain via `security find-generic-password`. Returns `ErrNoToken` if no token is configured; returns an error if the LLM response cannot be parsed.

**Traces to:** REQ-2 (LLM enrichment), REQ-7 (confidence from match)

---

## ARCH-4: TOML File Writer

**Decision:** Write TOML to `<data-dir>/memories/<slug>.toml`:

```go
type MemoryWriter interface {
    Write(memory *EnrichedMemory, dataDir string) (string, error) // returns file path
}
```

Implementation:
1. Ensure `<data-dir>/memories/` directory exists.
2. Slugify `FilenameSummary`: lowercase, replace spaces/non-alphanumeric with hyphens, trim to 3-5 words, append `.toml`.
3. If file already exists, append a numeric suffix (`-2`, `-3`, etc.).
4. Marshal `EnrichedMemory` to TOML and write atomically (write to temp file, rename).

**Traces to:** REQ-3 (TOML file writing)

---

## ARCH-5: System Reminder Renderer

**Decision:** Format the system reminder text per DES-1:

```go
type Renderer interface {
    Render(memory *EnrichedMemory, filePath string) string
}
```

Format (DES-1): `[engram] Memory captured.` + Created/Type/File

Returns empty string if no memory was created (shouldn't happen if called after Writer).

**Traces to:** REQ-4 (feedback), DES-1 (normal format)

---

## ARCH-6: CLI Wiring and Entry Point

**Decision:** Single binary with `correct` subcommand. Composition root at `internal/cli/`:

```go
// cmd/engram/main.go — thin entry point
func main() {
    if err := cli.Run(os.Args[1:]); err != nil {
        fmt.Fprintln(os.Stderr, err)
    }
}

// internal/cli/cli.go — composition root
func Run(args []string) error {
    // Parse: engram correct --message <text> --data-dir <path>
    // Construct real implementations:
    //   corpus := corpus.New()          // compiled patterns
    //   enricher := enrich.New(apiKey)  // Anthropic client (or degraded)
    //   writer := tomlwriter.New()      // file writer
    //   renderer := render.New()        // reminder formatter
    // Wire pipeline:
    //   corrector := correct.New(corpus, enricher, writer, renderer)
    // Run:
    //   output, err := corrector.Run(ctx, message)
    //   fmt.Print(output)
}
```

`internal/cli/` is the only package that imports I/O packages. All other `internal/` packages receive interfaces.

**Data directory convention:** `${CLAUDE_PLUGIN_ROOT}/data` — the hook script sets this via the `--data-dir` flag. Memory TOML files are written to `<data-dir>/memories/`.

**Plugin manifest:** `plugin.json` at repo root registers two hooks:
1. `SessionStart` → `hooks/session-start.sh` (builds binary, see ARCH-8)
2. `UserPromptSubmit` → `hooks/user-prompt-submit.sh` (runs correction pipeline)

**Hook script token retrieval (cross-platform):**
```bash
# macOS: try Keychain first
if [[ "$(uname)" == "Darwin" ]]; then
    TOKEN=$(security find-generic-password -s "Claude Code-credentials" -w 2>/dev/null \
        | python3 -c "import sys,json; print(json.load(sys.stdin)['claudeAiOauth']['accessToken'])" 2>/dev/null) || true
fi
# Fallback: use ENGRAM_API_TOKEN env var if set, or Keychain result
export ENGRAM_API_TOKEN="${TOKEN:-${ENGRAM_API_TOKEN:-}}"
```

**Traces to:** REQ-6 (Go binary CLI), REQ-8 (build mechanism), DES-3 (hook wiring, cross-platform token), DES-4 (installation), ARCH-1 (pipeline)

---

## ARCH-7: DI Boundary Interfaces

**Decision:** All I/O through injected interfaces. This is a lateral standard from CLAUDE.md design principles.

Core DI interfaces (summary — defined in detail by ARCH-2 through ARCH-5):

| Interface | Responsibility | Real Implementation | Test Double |
|-----------|---------------|-------------------|-------------|
| PatternMatcher | Regex matching | Compiled patterns | Fake returning canned match |
| Enricher | LLM API call | HTTP client to Anthropic | Fake returning canned EnrichedMemory |
| MemoryWriter | File I/O | TOML file writer | In-memory recorder |
| Renderer | Text formatting | Template renderer | Fake returning canned string |

`internal/` packages (except `internal/cli/`) never import `os`, `net/http`, or any I/O package.

**Traces to:** REQ-6 (pure Go), CLAUDE.md DI principles

---

## ARCH-8: Build Automation via SessionStart Hook

**Decision:** A `SessionStart` hook script builds the Go binary on every session start:

```bash
#!/usr/bin/env bash
set -euo pipefail
cd "${CLAUDE_PLUGIN_ROOT}"
go build -o bin/engram ./cmd/engram/ 2>/dev/null || echo "[engram] Warning: build failed. Is Go installed?" >&2
```

Design choices:
- **Always build:** Go's build cache makes this a sub-second no-op when source is unchanged. Simpler than staleness checks.
- **Silent success:** No stdout on success (stdout from hooks becomes system reminders). Errors go to stderr.
- **Graceful failure:** Build failure logs a warning but exits 0 — a broken build must not break the Claude Code session. The `UserPromptSubmit` hook will fail separately with a clear error if the binary doesn't exist.
- **Binary location:** `${CLAUDE_PLUGIN_ROOT}/bin/engram` — matches the path referenced by `hooks/user-prompt-submit.sh`.
- **`.gitignore`:** `bin/` directory is gitignored. The binary is a build artifact, not committed.

**Traces to:** REQ-8 (build mechanism), DES-4 (installation UX — auto-build means no manual build step)

---

## Bidirectional Traceability

### ARCH → L2

| ARCH | L2 items |
|------|----------|
| ARCH-1 | REQ-1, REQ-2, REQ-3, REQ-4, REQ-6 |
| ARCH-2 | REQ-1, REQ-7 |
| ARCH-3 | REQ-2, REQ-7 |
| ARCH-4 | REQ-3 |
| ARCH-5 | REQ-4, DES-1 |
| ARCH-6 | REQ-6, REQ-8, DES-3, DES-4 |
| ARCH-7 | REQ-6 |
| ARCH-8 | REQ-8, DES-4 |

### L2 → ARCH

| L2 item | ARCH coverage |
|---------|--------------|
| REQ-1 | ARCH-1, ARCH-2 |
| REQ-2 | ARCH-1, ARCH-3 |
| REQ-3 | ARCH-1, ARCH-4 |
| REQ-4 | ARCH-1, ARCH-5 |
| REQ-6 | ARCH-1, ARCH-6, ARCH-7 |
| REQ-7 | ARCH-2, ARCH-3 |
| REQ-8 | ARCH-6, ARCH-8 |
| DES-1 | ARCH-5 |
| DES-3 | ARCH-6 |
| DES-4 | ARCH-6, ARCH-8 |

All L2 items have ARCH coverage.
