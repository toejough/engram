# Architecture

System architecture for UC-3 (Remember & Correct) and UC-2 (Hook-Time Surfacing & Enforcement). Each ARCH decision traces to L2 items.

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

## UC-2 Architecture

---

## ARCH-9: Memory Storage and Retrieval

**Decision:** Memories are stored as individual TOML files in `<data-dir>/memories/`. Retrieval happens by scanning and parsing all files at query time (no database).

```go
type MemoryRetriever interface {
    ListMemories(ctx context.Context, dataDir string) ([]*Memory, error)
    FindByKeywords(keywords []string, memories []*Memory) []*Memory
}

type Memory struct {
    Title       string
    Content     string
    Concepts    []string
    Keywords    []string
    AntiPattern string // for PreToolUse enforcement
    Principle   string
    UpdatedAt   time.Time
    FilePath    string
}
```

Design choices:
- **No database:** Each TOML file is a Memory record. Scan all files in the memories directory on each retrieval. For small corpora (hundreds of memories), scanning is faster than database setup/teardown.
- **File-based discovery:** `ioutil.ReadDir(memdir)` + parse each `.toml` file. Errors on individual files are logged but don't block other memories.
- **Sorting:** `sort.Slice` on Memory structs (sort by UpdatedAt for SessionStart, no sorting for keyword matches).

**Traces to:** REQ-9 (SessionStart needs file listing), REQ-10 (UserPromptSubmit needs all memories for keyword match), REQ-11/12 (PreToolUse scans all memories)

---

## ARCH-10: Keyword Pre-Filter for PreToolUse

**Decision:** Fast, deterministic keyword matching on tool input before LLM judgment.

```go
type KeywordMatcher interface {
    MatchMemories(toolName, toolInput string, memories []*Memory) []*Memory
}
```

Implementation:
- For each memory with non-empty `anti_pattern`, extract its `keywords` array.
- For each keyword, check if it appears as a whole word in `toolName` or `toolInput` (case-insensitive).
- A memory matches if at least one of its keywords matches.
- Return all matching memories.

Design choices:
- **Whole-word matching:** Regex `\b<keyword>\b` (case-insensitive). Avoids false positives like "commit" matching "recommit".
- **No fuzzy matching:** Deterministic, predictable behavior. User learns that `keywords` drive pre-filter.
- **Fast:** Runs on every PreToolUse. Regex compilation cached per memory on first access.

**Traces to:** REQ-11 (keyword pre-filter for PreToolUse)

---

## ARCH-12: Surface Subcommand and Mode Routing

**Decision:** Single `surface` subcommand with mode flag routes to different behaviors.

```bash
engram surface --mode <session-start|prompt|tool> --data-dir <path> [mode-specific flags]
```

Routing:
- `--mode session-start`: Call MemoryRetriever.ListMemories, sort by UpdatedAt desc, take top 20, emit DES-5 format.
- `--mode prompt --message <text>`: Call MemoryRetriever.ListMemories, KeywordMatcher on message, emit DES-6 format.
- `--mode tool --tool-name <name> --tool-input <json>`: Call MemoryRetriever.ListMemories, KeywordMatcher, emit DES-7 advisory format (no LLM judgment).

Design choices:
- **Unified entry point:** One surface subcommand, mode-specific logic inside.
- **Hook scripts call surface:** SessionStart/UserPromptSubmit/PreToolUse hooks invoke `engram surface --mode ...`.
- **Advisory only (no blocking):** PreToolUse tool mode surfaces matching memories as advisory via system-reminder. Agent exercises judgment with full session context. No LLM call, no blocking decision from the Go binary.
- **JSON tool input:** PreToolUse hook passes full tool call as JSON (tool name + argument struct).

**Traces to:** REQ-14 (surface subcommand), REQ-9/10/11 (mode implementations)

---

## ARCH-13: Hook Script Integration

**Decision:** Existing hooks (session-start.sh, user-prompt-submit.sh) are extended. New PreToolUse hook added.

Hook flow:
- **SessionStart:** After build step, call `engram surface --mode session-start`. Stdout becomes system reminder.
- **UserPromptSubmit:** After `engram correct`, also call `engram surface --mode prompt --message "$PROMPT"`. Concatenate both outputs to stdout.
- **PreToolUse:** New hook script calls `engram surface --mode tool --tool-name <name> --tool-input <json>`. Output is a system-reminder advisory (or empty if no matches). Tool call is always allowed.

Design choices:
- **Extend, don't replace:** SessionStart and UserPromptSubmit keep their existing UC-3 behavior; surfacing is added.
- **PreToolUse is advisory only:** Hook script receives system-reminder text from `engram surface --mode tool`. No blocking decision from binary. Tool call always allowed; hook may emit advisory text to stdout.
- **Hook scripts are thin wrappers:** All logic in Go binary (MemoryRetriever, KeywordMatcher). Scripts just invoke and return output.

**Traces to:** DES-8 (hook wiring)

---

## UC-1 Architecture

---

## ARCH-14: Session Learner Pipeline

**Decision:** Linear pipeline of injected stages, similar to ARCH-1 but for transcript extraction:

```go
type Learner struct {
    Extractor   TranscriptExtractor  // LLM: transcript → []CandidateLearning
    Retriever   MemoryRetriever      // file I/O: data-dir → existing memories (ARCH-9 reuse)
    Deduplicator Deduplicator        // keyword overlap: candidates × existing → filtered candidates
    Writer      MemoryWriter         // file I/O: CandidateLearning → file path (ARCH-4 reuse)
}

func (l *Learner) Run(ctx context.Context, transcript string, dataDir string) ([]string, error) {
    // 1. Extractor.Extract(ctx, transcript): LLM call → []CandidateLearning
    // 2. Retriever.ListMemories(ctx, dataDir): read existing memory files
    // 3. Deduplicator.Filter(candidates, existing): remove duplicates by keyword overlap
    // 4. For each surviving candidate: Writer.Write(candidate, dataDir) → file path
    // 5. Return list of created file paths (for stderr feedback)
}
```

Four stages, each independently testable via DI. Reuses MemoryRetriever (ARCH-9) and MemoryWriter (ARCH-4).

**Traces to:** REQ-15 (extraction), REQ-17 (dedup), REQ-3 (file writing via ARCH-4), REQ-20 (CLI entry)

---

## ARCH-15: Transcript Extraction via LLM

**Decision:** Single LLM call to extract multiple learnings from a session transcript.

```go
type TranscriptExtractor interface {
    Extract(ctx context.Context, transcript string) ([]CandidateLearning, error)
}

type CandidateLearning struct {
    Title           string
    Content         string
    ObservationType string
    Concepts        []string
    Keywords        []string
    Principle       string
    AntiPattern     string
    Rationale       string
    FilenameSummary string
}
```

Implementation sends a single `messages` API call to `claude-haiku-4-5-20251001`. The system prompt:
1. Instructs the LLM to review the transcript and extract actionable learnings.
2. Defines the JSON array output format (each element has all CandidateLearning fields).
3. Embeds the quality gate (REQ-16): explicitly reject mechanical patterns, vague generalizations, and overly narrow observations.
4. Instructs extraction of: missed corrections, architectural decisions, discovered constraints, working solutions, implicit preferences.

Returns `ErrNoToken` if no token is configured (REQ-18 — fail loud). Returns an error if the LLM response cannot be parsed. Returns empty slice if LLM finds no learnings worth extracting.

**Traces to:** REQ-15 (LLM extraction), REQ-16 (quality gate), REQ-18 (fail loud)

---

## ARCH-16: Deduplication via Keyword Overlap

**Decision:** Compare candidate keywords against existing memory keywords. Skip candidates with >50% overlap.

```go
type Deduplicator interface {
    Filter(candidates []CandidateLearning, existing []*Memory) []CandidateLearning
}
```

Implementation:
1. For each candidate, compute its keyword set.
2. For each existing memory, compute its keyword set.
3. For each candidate-memory pair, compute `|intersection| / |candidate keywords|`.
4. If any existing memory has >50% overlap with the candidate, skip the candidate.
5. Return surviving candidates.

Design choices:
- **50% threshold:** Balances dedup aggressiveness. Too low → over-dedup (useful nuances lost). Too high → duplicates slip through.
- **Keyword-only:** No semantic similarity — keeps it deterministic and LLM-free at dedup time.
- **Candidate vs existing direction:** Overlap measured against the candidate's keywords, not the existing memory's. A candidate with 3 keywords where 2 match an existing memory with 20 keywords = 66% overlap → skipped.

**Traces to:** REQ-17 (deduplication), REQ-19 (idempotency — file-based dedup covers multi-trigger)

---

## ARCH-17: CLI Learn Subcommand

**Decision:** Extend the engram binary with a `learn` subcommand. Transcript read from stdin.

```bash
engram learn --data-dir <path> < transcript.txt
```

Routing in `internal/cli/`:
- Parse args for `learn` subcommand with `--data-dir` flag.
- Read transcript from stdin (buffered read to EOF).
- Construct Learner pipeline (ARCH-14): wire Extractor, Retriever, Deduplicator, Writer.
- Run pipeline, collect created file paths.
- Emit DES-10 format to stderr.

Design choices:
- **Stdin for transcript:** Command-line length limits make flags impractical for full transcripts.
- **Stderr for feedback:** Session may be ending (SessionEnd hook). Stdout is reserved for hook protocol.

**Traces to:** REQ-20 (CLI learn subcommand), DES-10 (feedback format)

---

## ARCH-18: Hook Script Integration for PreCompact and SessionEnd

**Decision:** Two new hook scripts invoke `engram learn`. Registered in `hooks/hooks.json`.

Hook flow:
- **PreCompact:** `hooks/pre-compact.sh` reads the transcript from stdin JSON, pipes to `engram learn --data-dir <path>`. Stderr feedback visible in logs.
- **SessionEnd:** `hooks/session-end.sh` reads the transcript from stdin JSON, pipes to `engram learn --data-dir <path>`. Stderr feedback visible in logs.

Both scripts:
1. Extract transcript from stdin JSON payload (field name per hook event).
2. Use same token retrieval as DES-3 (macOS Keychain fallback to env var).
3. Pipe transcript to `engram learn --data-dir "$ENGRAM_DATA"`.
4. Exit 0 always.

Design choices:
- **Two separate scripts:** Although identical in logic, separate scripts allow future divergence (e.g., PreCompact might pass only the about-to-be-compacted portion).
- **Same binary:** Both invoke the same `engram learn` subcommand.

**Traces to:** DES-9 (hook wiring), REQ-19 (idempotency — second invocation deduplicates against first)

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
| ARCH-9 | REQ-9, REQ-10, REQ-11 |
| ARCH-10 | REQ-11 |
| ARCH-12 | REQ-14, REQ-9, REQ-10, REQ-11 |
| ARCH-13 | DES-8 |
| ARCH-14 | REQ-15, REQ-17, REQ-3, REQ-20 |
| ARCH-15 | REQ-15, REQ-16, REQ-18 |
| ARCH-16 | REQ-17, REQ-19 |
| ARCH-17 | REQ-20, DES-10 |
| ARCH-18 | DES-9, REQ-19 |

### L2 → ARCH

| L2 item | ARCH coverage |
|---------|--------------|
| REQ-1 | ARCH-1, ARCH-2 |
| REQ-2 | ARCH-1, ARCH-3 |
| REQ-3 | ARCH-1, ARCH-4, ARCH-14 |
| REQ-4 | ARCH-1, ARCH-5 |
| REQ-6 | ARCH-1, ARCH-6, ARCH-7 |
| REQ-7 | ARCH-2, ARCH-3 |
| REQ-8 | ARCH-6, ARCH-8 |
| DES-1 | ARCH-5 |
| DES-3 | ARCH-6 |
| DES-4 | ARCH-6, ARCH-8 |
| REQ-9 | ARCH-9, ARCH-12 |
| REQ-10 | ARCH-9, ARCH-12 |
| REQ-11 | ARCH-9, ARCH-10, ARCH-12 |
| REQ-14 | ARCH-12 |
| DES-5 | ARCH-9, ARCH-12 |
| DES-6 | ARCH-9, ARCH-12 |
| DES-7 | ARCH-12 |
| DES-8 | ARCH-13 |
| REQ-15 | ARCH-14, ARCH-15 |
| REQ-16 | ARCH-15 |
| REQ-17 | ARCH-14, ARCH-16 |
| REQ-18 | ARCH-15 |
| REQ-19 | ARCH-16, ARCH-18 |
| REQ-20 | ARCH-14, ARCH-17 |
| DES-9 | ARCH-18 |
| DES-10 | ARCH-17 |

All L2 items have ARCH coverage.
