# Architecture

System architecture for UC-3 (Remember & Correct), UC-2 (Hook-Time Surfacing & Enforcement), and UC-15 (Automatic Outcome Signal). Each ARCH decision traces to L2 items.

---

## ARCH-1: Pipeline Architecture

**Decision:** Three-stage pipeline with fast-path bypass:

```go
type Corrector struct {
    Classifier Classifier    // fast-path check + LLM: message + context → ClassifiedMemory or nil
    Writer     MemoryWriter  // file I/O: ClassifiedMemory → file path
    Renderer   Renderer      // format: ClassifiedMemory + path → system reminder text
}

func (c *Corrector) Run(ctx context.Context, message, transcriptPath string) (string, error) {
    // 1. Classifier.Classify(ctx, message, transcriptPath): fast-path or LLM → ClassifiedMemory or nil
    //    - Fast-path: "remember"/"always"/"never" → tier A, skip LLM classifier
    //    - LLM: single call classifies (A/B/C/null) + enriches structured fields
    //    - null tier → return "" (no signal)
    // 2. Writer.Write(memory, dataDir): write TOML file, return file path
    // 3. Renderer.Render(memory, path): build system reminder text with tier
    // 4. Return system reminder text
}
```

Three stages, each independently testable via DI. The Classifier replaces the old Corpus+Enricher stages — classification and enrichment happen in a single LLM call.

**Traces to:** REQ-1 (detection/classification), REQ-2 (unified LLM call), REQ-3 (file writing), REQ-4 (feedback), REQ-6 (Go binary)

---

## ARCH-2: Unified Classifier (Fast-Path + LLM)

**Decision:** Two-stage detection: deterministic fast-path, then LLM classifier.

```go
type Classifier interface {
    Classify(ctx context.Context, message, transcriptContext string) (*ClassifiedMemory, error)
}

type ClassifiedMemory struct {
    Tier            string   // "A", "B", or "C"
    Title           string
    Content         string
    ObservationType string
    Concepts        []string
    Keywords        []string
    Principle       string
    AntiPattern     string   // tier-gated: required for A, optional for B, empty for C
    Rationale       string
    FilenameSummary string
    Confidence      string   // same as Tier
    CreatedAt       time.Time
    UpdatedAt       time.Time
}
```

Implementation:
1. **Fast-path check:** Case-insensitive check for keywords `remember`, `always`, `never` in the message. If found → tier A, proceed to enrichment LLM call (or inline the enrichment in the classifier).
2. **LLM classifier:** For non-fast-path messages, a single API call (claude-haiku-4-5-20251001) receives the message + transcript context and returns JSON with `tier` (A/B/C/null) plus all structured fields.
3. **Null → nil return:** If classifier returns null tier, return nil (no signal detected).
4. **Anti-pattern gating:** Per REQ-7, `anti_pattern` field is populated only for tier A (always) and tier B (when generalizable, LLM decides). Tier C → empty string.

**Traces to:** REQ-1 (fast-path + classifier), REQ-2 (unified LLM call), REQ-7 (tier criteria + anti-pattern gating)

---

## ARCH-3: Transcript Context Reader

**Decision:** Read recent transcript context from the session transcript file for the unified classifier.

```go
type TranscriptReader interface {
    ReadRecent(transcriptPath string, maxTokens int) (string, error)
}
```

Implementation:
1. Open the file at `transcriptPath` (provided via hook JSON input).
2. Read the file content.
3. If content exceeds `maxTokens` (~2000), take the tail (most recent portion).
4. Return the recent context string.
5. If file is missing or unreadable → return empty string (non-fatal, context is advisory).

The CLI layer reads `transcript_path` from the hook JSON stdin and passes it to the Corrector pipeline. The Classifier (ARCH-2) includes this context in its LLM prompt.

**Authentication:** OAuth token from `ENGRAM_API_TOKEN` env var, sent as `Authorization: Bearer` header with `Anthropic-Beta: oauth-2025-04-20`. The hook script reads the token from the Claude Code Keychain via `security find-generic-password`. Returns `ErrNoToken` if no token is configured; returns an error if the LLM response cannot be parsed.

**Traces to:** REQ-X (transcript context reading), REQ-2 (unified LLM call context)

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

**Decision:** Format the system reminder text per DES-1, including tier:

```go
type Renderer interface {
    Render(memory *ClassifiedMemory, filePath string) string
}
```

Format (DES-1): `[engram] Memory captured (tier A).` + Created/Type/File

Returns empty string if no memory was created (shouldn't happen if called after Writer).

**Traces to:** REQ-4 (feedback with tier), DES-1 (format with tier)

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
    // Read transcript_path from hook JSON stdin
    // Construct real implementations:
    //   classifier := classify.New(apiKey, httpClient)  // fast-path + LLM classifier
    //   reader := transcript.New()                       // transcript context reader
    //   writer := tomlwriter.New()                       // file writer
    //   renderer := render.New()                          // reminder formatter
    // Read transcript context:
    //   context := reader.ReadRecent(transcriptPath, 2000)
    // Wire pipeline:
    //   corrector := correct.New(classifier, writer, renderer)
    // Run:
    //   output, err := corrector.Run(ctx, message, transcriptPath)
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
| Classifier | Fast-path + LLM classify+enrich | HTTP client to Anthropic | Fake returning canned ClassifiedMemory |
| TranscriptReader | Read recent transcript context | File reader | Fake returning canned string |
| MemoryWriter | File I/O | TOML file writer | In-memory recorder |
| Renderer | Text formatting with tier | Template renderer | Fake returning canned string |

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
    Title             string
    Content           string
    Concepts          []string
    Keywords          []string
    AntiPattern       string // for PreToolUse enforcement
    Principle         string
    UpdatedAt         time.Time
    FilePath          string
    SurfacedCount     int       // instrumentation: total surfacing events
    LastSurfaced      time.Time // instrumentation: most recent surfacing
    SurfacingContexts []string  // instrumentation: recent context types (max 10)
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
engram surface --mode <session-start|prompt|tool> --data-dir <path> [--format json] [mode-specific flags]
```

Routing:
- `--mode session-start`: Read creation log (ARCH-21 LogReader.ReadAndClear) → emit creation report if entries exist. Then call MemoryRetriever.ListMemories, sort by UpdatedAt desc, take top 20, emit DES-5 format. Both sections combined in output.
- `--mode prompt --message <text>`: Call MemoryRetriever.ListMemories, KeywordMatcher on message, emit DES-6 format.
- `--mode tool --tool-name <name> --tool-input <json>`: Call MemoryRetriever.ListMemories, KeywordMatcher, emit DES-7 advisory format (no LLM judgment).

Output format:
- **Default (no --format):** Writes `<system-reminder>` XML directly to stdout (backward compatible).
- **`--format json`:** Returns a JSON object `{"summary": "<brief message>", "context": "<system-reminder XML>"}`. Summary is a human-readable one-liner (e.g., `"[engram] Loaded 20 memories."`). Context is the full XML block.
- **No matches:** Empty stdout regardless of format (not an empty JSON object).

Design choices:
- **Unified entry point:** One surface subcommand, mode-specific logic inside.
- **Hook scripts call surface with JSON format:** Hooks invoke `engram surface --mode ... --format json`, then reshape into hook-specific JSON with `systemMessage` (user-visible) and `additionalContext` (model context). PreToolUse nests `additionalContext` under `hookSpecificOutput` per Claude Code hook API.
- **Advisory only (no blocking):** PreToolUse tool mode surfaces matching memories as advisory via system-reminder. Agent exercises judgment with full session context. No LLM call, no blocking decision from the Go binary.
- **JSON tool input:** PreToolUse hook passes full tool call as JSON (tool name + argument struct).

**Traces to:** REQ-14 (surface subcommand), REQ-9/10/11 (mode implementations)

---

## ARCH-13: Hook Script Integration

**Decision:** Existing hooks (session-start.sh, user-prompt-submit.sh) are extended. New PreToolUse hook added.

Hook flow:
- **SessionStart:** After build step, call `engram surface --mode session-start --format json`. Reshape into `{systemMessage, additionalContext}`. Creation report (from creation log, if any) is included in `systemMessage`.
- **UserPromptSubmit:** Run `engram correct` and `engram surface --mode prompt --format json` independently. Combine into `{systemMessage: (<surface_summary> + "\n" + <creation_output>), additionalContext: <surface_context>}`. Creation feedback always goes in `systemMessage` (user-visible), never in `additionalContext`.
- **PreToolUse:** Hook script calls `engram surface --mode tool --format json`. Reshape into `{continue: true, hookSpecificOutput: {additionalContext}}`. Tool call is always allowed.

Design choices:
- **Creation always in systemMessage:** The user must see memory creation events. Whether creation happens alone or alongside surface matches, the creation output goes into `systemMessage`.
- **PreToolUse is advisory only:** Hook script receives system-reminder text from `engram surface --mode tool`. No blocking decision from binary.
- **Hook scripts are thin wrappers:** All logic in Go binary (MemoryRetriever, KeywordMatcher, CreationLog). Scripts just invoke and reshape output.

**Traces to:** DES-8 (hook wiring), DES-3 (UserPromptSubmit creation + surface combining), DES-5 (SessionStart creation report)

---

## ARCH-19: Surfacing Instrumentation — Tracking Logic and Recorder

**Decision:** Separate pure tracking logic from I/O-performing recorder, both in `internal/track/`.

**Pure tracking logic (`ComputeUpdate`):**

```go
const MaxContextEntries = 10

type SurfacingUpdate struct {
    SurfacedCount     int
    LastSurfaced      time.Time
    SurfacingContexts []string
}

func ComputeUpdate(current *memory.Stored, mode string, now time.Time) SurfacingUpdate
```

Business logic only: increment count, set timestamp, append mode with FIFO eviction at 10. No I/O.

**Recorder (`Recorder`):**

```go
type Recorder struct {
    readFile   func(string) ([]byte, error)
    createTemp func(dir, pattern string) (*os.File, error)
    rename     func(oldpath, newpath string) error
    remove     func(name string) error
    now        func() time.Time
}

func (r *Recorder) RecordSurfacing(ctx context.Context, memories []*memory.Stored, mode string) error
```

For each memory: read existing TOML → decode full record (all fields) → apply `ComputeUpdate` → re-encode → write atomically (temp + rename). Errors on individual memories are collected but don't stop processing others. Uses the same `tomlRecord` field set as `tomlwriter` plus the three tracking fields to ensure round-trip fidelity.

All I/O is injected via functional options (`WithReadFile`, `WithCreateTemp`, `WithRename`, `WithRemove`, `WithNow`). Default: real `os.*` functions and `time.Now`.

**Traces to:** REQ-21 (tracking fields), REQ-22 (in-place TOML update)

---

## ARCH-20: Surfacer ↔ Tracker Integration

**Decision:** Optional `MemoryTracker` interface injected into the Surfacer.

```go
// In surface package
type MemoryTracker interface {
    RecordSurfacing(ctx context.Context, memories []*memory.Stored, mode string) error
}
```

The Surfacer accepts an optional tracker via `WithTracker` functional option. After each mode handler determines matched memories, the surfacer calls `tracker.RecordSurfacing(ctx, matched, mode)`. If the tracker is nil, no tracking occurs (backward compatible).

**Refactor required:** The three mode handlers (`runSessionStart`, `runPrompt`, `runTool`) currently return `Result` directly. They need to also return the matched `[]*memory.Stored` so the surfacer can pass them to the tracker. Internal refactor only — no public API change.

**Error handling:** Tracker errors are logged to stderr and swallowed — they never propagate to the caller (ARCH-6 exit-0 contract). This keeps surfacing instrumentation fire-and-forget.

**CLI wiring** (in `internal/cli/cli.go`):
```go
recorder := track.NewRecorder()
surfacer := surface.New(retriever, surface.WithTracker(recorder))
```

**Traces to:** REQ-22 (in-place TOML update on surfacing), ARCH-6 (exit-0 contract), ARCH-7 (DI boundary interfaces)

---

## UC-1 Architecture

---

## ARCH-14: Session Learner Pipeline

**Decision:** Linear pipeline of injected stages, similar to ARCH-1 but for transcript extraction:

```go
type CreationLogger interface {
    Append(entry LogEntry, dataDir string) error
}

type Learner struct {
    Extractor      TranscriptExtractor  // LLM: transcript → []CandidateLearning
    Retriever      MemoryRetriever      // file I/O: data-dir → existing memories (ARCH-9 reuse)
    Deduplicator   Deduplicator         // keyword overlap: candidates × existing → filtered candidates
    Writer         MemoryWriter         // file I/O: CandidateLearning → file path (ARCH-4 reuse)
    CreationLogger CreationLogger       // optional: log creation events for deferred visibility (ARCH-21)
}

func (l *Learner) Run(ctx context.Context, transcript string, dataDir string) ([]string, error) {
    // 1. Extractor.Extract(ctx, transcript): LLM call → []CandidateLearning
    // 2. Retriever.ListMemories(ctx, dataDir): read existing memory files
    // 3. Deduplicator.Filter(candidates, existing): remove duplicates by keyword overlap
    // 4. For each surviving candidate:
    //    a. Writer.Write(candidate, dataDir) → file path
    //    b. CreationLogger.Append({timestamp, title, tier, filename}, dataDir) — fire-and-forget
    // 5. Return list of created file paths (for stderr feedback)
}
```

Five stages (four existing + optional creation logger), each independently testable via DI. Reuses MemoryRetriever (ARCH-9) and MemoryWriter (ARCH-4). CreationLogger is optional — if nil, no creation log is written (backward compatible).

**Traces to:** REQ-15 (extraction), REQ-17 (dedup), REQ-3 (file writing via ARCH-4), REQ-20 (CLI entry), REQ-25 (creation log write)

---

## ARCH-15: Transcript Extraction via LLM with Unified Tier Criteria

**Decision:** Single LLM call to extract multiple learnings from a session transcript, classifying each using the same A/B/C tier criteria as the real-time classifier (ARCH-2).

```go
type TranscriptExtractor interface {
    Extract(ctx context.Context, transcript string) ([]CandidateLearning, error)
}

type CandidateLearning struct {
    Tier            string   // "A", "B", or "C"
    Title           string
    Content         string
    ObservationType string
    Concepts        []string
    Keywords        []string
    Principle       string
    AntiPattern     string   // tier-gated: required for A, optional for B, empty for C
    Rationale       string
    FilenameSummary string
}
```

Implementation sends a single `messages` API call to `claude-haiku-4-5-20251001`. The system prompt:
1. Instructs the LLM to review the transcript and extract actionable learnings.
2. Defines the JSON array output format (each element has all CandidateLearning fields, including `tier`).
3. Includes the same A/B/C tier definitions as the real-time classifier: A = explicit instruction, B = teachable correction, C = contextual fact.
4. Embeds anti-pattern gating: A → always generate anti_pattern, B → when generalizable, C → never.
5. Embeds the quality gate (REQ-16): explicitly reject mechanical patterns, vague generalizations, and overly narrow observations.
6. Instructs extraction of: missed corrections, architectural decisions, discovered constraints, working solutions, implicit preferences.

Returns `ErrNoToken` if no token is configured (REQ-18 — fail loud). Returns an error if the LLM response cannot be parsed. Returns empty slice if LLM finds no learnings worth extracting.

**Traces to:** REQ-15 (LLM extraction with tier classification), REQ-16 (quality gate), REQ-7 (unified tier criteria + anti-pattern gating), REQ-18 (fail loud)

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

## ARCH-21: Creation Log — Deferred Visibility for UC-1

**Decision:** New `internal/creationlog/` package for JSONL creation log read/write with full DI.

**Writer (used by UC-1 Learner pipeline):**

```go
type LogWriter struct {
    readFile  func(string) ([]byte, error)
    writeFile func(string, []byte, os.FileMode) error
    now       func() time.Time
}

type LogEntry struct {
    Timestamp string `json:"timestamp"` // RFC 3339
    Title     string `json:"title"`
    Tier      string `json:"tier"`       // A/B/C
    Filename  string `json:"filename"`   // e.g. "use-targ-test.toml"
}

func (w *LogWriter) Append(entry LogEntry, dataDir string) error
```

Implementation:
1. Read existing `<data-dir>/creation-log.jsonl` (or empty if missing).
2. Append new JSON line.
3. Write atomically (write full content to temp file, rename).
4. Fire-and-forget: errors logged to stderr, never fail the caller (ARCH-6 exit-0 contract).

**Reader (used by SessionStart surfacing):**

```go
type LogReader struct {
    readFile   func(string) ([]byte, error)
    removeFile func(string) error
}

func (r *LogReader) ReadAndClear(dataDir string) ([]LogEntry, error)
```

Implementation:
1. Read `<data-dir>/creation-log.jsonl`.
2. Parse each line as JSON → `LogEntry`.
3. Delete the file after successful read.
4. Return entries (or empty slice if file missing).
5. Read/delete errors logged to stderr, non-fatal.

All I/O injected via functional options (`WithReadFile`, `WithWriteFile`, `WithRemoveFile`, `WithNow`). Default: real `os.*` functions and `time.Now`.

**Traces to:** REQ-23 (creation log format), REQ-24 (read and clear at SessionStart), REQ-25 (write during learn)

---

## ARCH-22: Surfacing Log Infrastructure

**Decision:** Write surfacing events to a session-scoped JSONL file during each surfacing event (SessionStart, UserPromptSubmit, PreToolUse). The evaluate pass reads this log to determine which memories were surfaced, then clears it.

```go
type SurfacingLogger interface {
    LogSurfacing(filePath string, mode string, timestamp time.Time) error
    ReadAndClear(filePath string) ([]SurfacingEvent, error)
}

type SurfacingEvent struct {
    MemoryPath string
    Mode       string    // "session-start", "prompt", "tool"
    SurfacedAt time.Time
}
```

Implementation:
1. File path: `<data-dir>/surfacing-log.jsonl`
2. Append mode: each surfacing event appends one line per matched memory
3. Format: JSONL with memory_path, mode, surfaced_at (RFC 3339)
4. Read-and-clear: evaluate reads all lines, parses, returns slice, then removes file
5. Append errors are fire-and-forget (ARCH-6 exit-0 contract)

**Traces to:** REQ-26 (surfacing log write/read), DES-11 (JSONL format)

---

## ARCH-23: Outcome Evaluation Pipeline

**Decision:** Three-stage pipeline for outcome evaluation: read surfacing log → send to LLM for classification → write evaluation results.

```go
type Evaluator struct {
    SurfacingLogger  SurfacingLogger  // read surfacing log
    MemoryRetriever  MemoryRetriever  // fetch surfaced memory details
    LLMEvaluator     LLMEvaluator     // classify outcomes
    EvaluationWriter EvaluationWriter // write evaluation log
}

func (e *Evaluator) Evaluate(ctx context.Context, transcript string, dataDir string) error {
    // 1. Read surfacing log
    // 2. Fetch each surfaced memory's TOML
    // 3. Send transcript + surfaced memories to LLM
    // 4. Classify each outcome (followed/contradicted/ignored)
    // 5. Write evaluation log file
}

type Outcome struct {
    MemoryPath string // file path to the memory
    Outcome    string // "followed", "contradicted", "ignored"
    Evidence   string // brief LLM explanation
    EvaluatedAt time.Time
}
```

Implementation:
1. LLM call uses claude-haiku-4-5-20251001
2. Input: full transcript + list of surfaced memories (title, principle, anti_pattern, content)
3. Output: JSON array with one entry per surfaced memory
4. Each memory gets exactly one outcome classification
5. Evidence field captures LLM's reasoning

**Traces to:** REQ-27 (LLM evaluation), REQ-28 (evaluation log write), DES-12 (LLM prompt design), DES-13 (evaluation log schema)

---

## ARCH-24: Effectiveness Aggregation (Read Path)

**Decision:** Compute effectiveness on-the-fly from evaluation logs when surfacing. Pure computation, no caching in TOML.

```go
type EffectivenessComputer interface {
    Aggregate(evalDir string) map[string]EffectivenessStat
}

type EffectivenessStat struct {
    FollowedCount      int
    ContradictedCount  int
    IgnoredCount       int
    EffectivenessScore float64 // followed / (followed + contradicted + ignored) * 100
}
```

Implementation:
1. Read all `.jsonl` files in `<data-dir>/evaluations/`
2. Parse each line and group by memory_path
3. Aggregate counts: followed, contradicted, ignored per memory
4. Compute effectiveness percentage: `followed / (followed + contradicted + ignored) * 100`
5. Return map of memory_path → EffectivenessStat
6. Missing evaluations directory → empty map (no error)
7. Malformed lines skipped

Usage: When UC-2 surfaces memories, call Aggregate, then append effectiveness annotation to each memory's output line.

**Traces to:** REQ-29 (effectiveness aggregation), REQ-30 (effectiveness annotations)

---

## ARCH-25: Hook Integration — evaluate Subcommand

**Decision:** Thin CLI wrapper for the Evaluator pipeline. Same pattern as correct/learn: reads transcript from stdin, invokes Evaluator, outputs summary.

```go
func runEvaluate(ctx context.Context, dataDir string, in io.Reader, out io.Writer) error {
    // 1. Read transcript from stdin
    // 2. Create Evaluator with DI
    // 3. Call Evaluator.Evaluate(ctx, transcript, dataDir)
    // 4. Format and output evaluation summary to stdout
    // 5. Exit 0 always — errors logged to stderr per ARCH-6
}
```

Implementation:
1. CLI: `engram evaluate --data-dir <path>`
2. Reads transcript from stdin (same as learn)
3. Outputs summary to stdout for hook to reshape into systemMessage
4. Requires API token (same mechanism as correct/learn; emit error if missing)
5. Exit code: always 0 (ARCH-6 contract)

**Traces to:** REQ-32 (evaluate CLI subcommand), DES-15 (hook wiring)

---

## Bidirectional Traceability

### ARCH → L2

| ARCH | L2 items |
|------|----------|
| ARCH-1 | REQ-1, REQ-2, REQ-3, REQ-4, REQ-6 |
| ARCH-2 | REQ-1, REQ-2, REQ-7 |
| ARCH-3 | REQ-X, REQ-2 |
| ARCH-4 | REQ-3 |
| ARCH-5 | REQ-4, DES-1 |
| ARCH-6 | REQ-6, REQ-8, DES-3, DES-4 |
| ARCH-7 | REQ-6 |
| ARCH-8 | REQ-8, DES-4 |
| ARCH-9 | REQ-9, REQ-10, REQ-11, REQ-21 |
| ARCH-10 | REQ-11 |
| ARCH-12 | REQ-14, REQ-9, REQ-10, REQ-11, REQ-24 |
| ARCH-13 | DES-8, DES-3, DES-5 |
| ARCH-14 | REQ-15, REQ-17, REQ-3, REQ-20, REQ-25 |
| ARCH-15 | REQ-15, REQ-16, REQ-7, REQ-18 |
| ARCH-16 | REQ-17, REQ-19 |
| ARCH-17 | REQ-20, DES-10 |
| ARCH-18 | DES-9, REQ-19 |
| ARCH-19 | REQ-21, REQ-22 |
| ARCH-20 | REQ-22, ARCH-6, ARCH-7 |
| ARCH-21 | REQ-23, REQ-24, REQ-25 |
| ARCH-22 | REQ-26, DES-11 |
| ARCH-23 | REQ-27, REQ-28, DES-12, DES-13 |
| ARCH-24 | REQ-29, REQ-30 |
| ARCH-25 | REQ-32, DES-15 |

### L2 → ARCH

| L2 item | ARCH coverage |
|---------|--------------|
| REQ-1 | ARCH-1, ARCH-2 |
| REQ-2 | ARCH-1, ARCH-2, ARCH-3 |
| REQ-X | ARCH-3 |
| REQ-3 | ARCH-1, ARCH-4, ARCH-14 |
| REQ-4 | ARCH-1, ARCH-5, ARCH-13 |
| REQ-6 | ARCH-1, ARCH-6, ARCH-7 |
| REQ-7 | ARCH-2, ARCH-15 |
| REQ-8 | ARCH-6, ARCH-8 |
| DES-1 | ARCH-5 |
| DES-3 | ARCH-6, ARCH-13 |
| DES-4 | ARCH-6, ARCH-8 |
| REQ-9 | ARCH-9, ARCH-12 |
| REQ-10 | ARCH-9, ARCH-12 |
| REQ-11 | ARCH-9, ARCH-10, ARCH-12 |
| REQ-14 | ARCH-12 |
| DES-5 | ARCH-9, ARCH-12, ARCH-13, ARCH-21 |
| DES-6 | ARCH-9, ARCH-12 |
| DES-7 | ARCH-12 |
| DES-8 | ARCH-13 |
| REQ-21 | ARCH-9, ARCH-19 |
| REQ-23 | ARCH-21 |
| REQ-24 | ARCH-12, ARCH-21 |
| REQ-25 | ARCH-14, ARCH-21 |
| REQ-22 | ARCH-19, ARCH-20 |
| REQ-15 | ARCH-14, ARCH-15 |
| REQ-16 | ARCH-15 |
| REQ-17 | ARCH-14, ARCH-16 |
| REQ-18 | ARCH-15 |
| REQ-19 | ARCH-16, ARCH-18 |
| REQ-20 | ARCH-14, ARCH-17 |
| DES-9 | ARCH-18 |
| DES-10 | ARCH-17 |
| REQ-26 | ARCH-22 |
| DES-11 | ARCH-22 |
| REQ-27 | ARCH-23 |
| DES-12 | ARCH-23 |
| REQ-28 | ARCH-23 |
| DES-13 | ARCH-23 |
| REQ-29 | ARCH-24 |
| REQ-30 | ARCH-24 |
| DES-14 | ARCH-24 |
| REQ-31 | ARCH-25 |
| DES-15 | ARCH-25 |
| REQ-32 | ARCH-25 |
| REQ-33 | ARCH-15 |
| REQ-34 | ARCH-16 |

---

# UC-6: Memory Effectiveness Review

---

## ARCH-26: Matrix Classifier

**Decision:** New `internal/review/` package with a pure classification function.

**Interface:**

```go
// ClassifiedMemory holds a memory's quadrant assignment and stats.
type ClassifiedMemory struct {
    Name               string
    Quadrant           Quadrant // Working, HiddenGem, Leech, Noise, InsufficientData
    SurfacedCount      int
    EffectivenessScore float64
    EvaluationCount    int
    Flagged            bool
}

// Quadrant represents a position in the 2x2 effectiveness matrix.
type Quadrant string

const (
    Working          Quadrant = "Working"
    HiddenGem        Quadrant = "Hidden Gem"
    Leech            Quadrant = "Leech"
    Noise            Quadrant = "Noise"
    InsufficientData Quadrant = "Insufficient Data"
)

// Classify takes effectiveness stats and tracking data, returns classified memories.
func Classify(
    effectiveness map[string]effectiveness.Stat,
    tracking map[string]TrackingData,
) []ClassifiedMemory
```

**Algorithm:**
1. Merge effectiveness + tracking maps by memory path (union of keys).
2. Compute median surfaced count across all memories with tracking data.
3. For each memory:
   - Total evaluations = Followed + Contradicted + Ignored
   - If total < 5 → InsufficientData, Flagged = false
   - Else: assign quadrant by (surfaced > median) × (effectiveness >= 50%)
   - Flagged = total >= 5 AND effectiveness < 40%
4. Sort by: Flagged desc, then EffectivenessScore asc.

**DI:** Pure function. No I/O. Receives pre-aggregated data from callers.

**Traces to:** REQ-35 (matrix classification), REQ-36 (threshold flagging)

---

## ARCH-27: Review CLI Wiring

**Decision:** New `review` subcommand in `internal/cli/cli.go`, following existing subcommand pattern.

**Wiring:**

```
engram review --data-dir <path>
    │
    ├── effectiveness.New(evalDir).Aggregate()  → map[string]Stat
    ├── retrieve.New(memDir).All()              → []memory.Stored (for tracking fields)
    │
    ├── review.Classify(stats, tracking)        → []ClassifiedMemory
    └── review.Render(classified, stdout)       → human-readable output per DES-16
```

**New dependencies:**
- `internal/review/` — Classify + Render functions
- Reuses existing `internal/effectiveness/` and `internal/retrieve/`

**retrieve.Retriever extension:** Need to expose tracking fields (SurfacedCount) from retrieved memories. The `memory.Stored` type already has `SurfacedCount`, `LastSurfaced`, `SurfacingContexts` fields (from issue #46, ARCH-20). The review CLI reads these via the existing retriever.

**DI table (new entries):**

| Interface/Function | Production impl | Test impl |
|---|---|---|
| `review.Classify` | Pure function | Direct call with test data |
| `review.Render` | Pure function → io.Writer | Write to bytes.Buffer |
| `effectiveness.Computer` | File-reading aggregator | Injected readDir/readFile |
| `retrieve.Retriever` | File-reading retriever | Injected readDir/readFile |

**Traces to:** REQ-38 (review CLI), REQ-39 (no-data behavior), DES-16 (output format)

---

## L2 → ARCH Traceability (UC-6)

| L2 Item | ARCH Coverage |
|---------|--------------|
| REQ-35 | ARCH-26 |
| REQ-36 | ARCH-26 |
| REQ-37 | ARCH-24 (existing — effectiveness annotations already wired) |
| DES-17 | ARCH-24 (existing — formatEffectivenessAnnotation already implemented) |
| REQ-38 | ARCH-27 |
| DES-16 | ARCH-27 |
| REQ-39 | ARCH-27 |

All UC-6 L2 items have ARCH coverage.
