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

**Decision:** Memories are stored as individual TOML files in `<data-dir>/memories/`. Retrieval uses BM25 ranking: scan and parse all files, build a BM25 index on demand per query, score the input text against each memory, and return ranked results.

```go
type MemoryRetriever interface {
    ListMemories(ctx context.Context, dataDir string) ([]*Memory, error)
}

type BM25Scorer interface {
    Score(query string, memories []*Memory) []ScoredMemory
}

type ScoredMemory struct {
    Memory *Memory
    Score  float64
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
- **No persistent database:** Each TOML file is a Memory record. Scan all files in memories/ on each query. For small corpora (hundreds of memories), scanning is faster than database setup.
- **BM25 indexing:** Per-query BM25 index built by concatenating searchable fields (title, content, principle, keywords, concepts, anti_pattern). No persistent index — rebuilt per call.
- **Ranking:** BM25 algorithm produces relevance scores; memories sorted by score descending. Results are top-N limited (top 10 for UserPromptSubmit, top 5 for PreToolUse).
- **File-based discovery:** `os.ReadDir(memdir)` + parse each `.toml` file. Errors on individual files don't block others.

**Traces to:** REQ-9 (SessionStart lists all), REQ-10 (UserPromptSubmit BM25 ranking), REQ-11 (PreToolUse BM25 ranking on anti-pattern candidates)

---

## ARCH-10: BM25 Candidate Pruning (ARCH-9 detail)

**Decision:** Pre-query filtering in PreToolUse: only memories with non-empty `anti_pattern` field are indexed and ranked (tier-aware: tier A always, tier B sometimes, tier C never per REQ-7). Same BM25 scoring as UserPromptSubmit, applied to the filtered candidate set.

Implementation:
- Candidate selection: Filter all memories to keep only those with non-empty `anti_pattern` (respecting tier-awareness per REQ-7).
- BM25 indexing: Build BM25 index from candidate memories' searchable fields (title, principle, anti_pattern, keywords).
- Query: Concatenate tool name and input.
- Scoring and ranking: Score query against each candidate, sort by relevance score descending.
- Result limit: Return top 5 ranked candidates (or all if fewer than 5 exist).
- Zero results: If no candidates exist or all scores are zero, return empty (zero overhead, no advisory).

Design choices:
- **BM25 ranking:** Same scoring algorithm as prompt mode, applied to the filtered candidate set. No regex matching.
- **Fast:** Runs on every PreToolUse. BM25 index built per call from small candidate set (anti-pattern memories only).

**Traces to:** REQ-11 (BM25 + frecency ranking for PreToolUse)

---

## ARCH-35: Frecency Activation Scorer

**Decision:** Pure function that computes ACT-R frecency activation scores for memories. Used in all three surfacing modes.

```go
type FrecencyScorer struct {
    now            time.Time
    effectiveness  map[string]EffectivenessStat
}

type ActivationInput struct {
    SurfacedCount     int
    LastSurfaced      time.Time
    UpdatedAt         time.Time // fallback for never-surfaced
    SurfacingContexts []string
    FilePath          string    // key for effectiveness lookup
}

func (f *FrecencyScorer) Activation(input ActivationInput) float64 {
    // Returns: frequency × recency × spread × effectiveness
}

func (f *FrecencyScorer) CombinedScore(bm25Score float64, input ActivationInput) float64 {
    // Returns: bm25Score × (1 + Activation(input))
}
```

Implementation:
- **Frequency:** `log(1 + surfacedCount)` — never-surfaced = log(1) = 0, capped at reasonable value
- **Recency:** `1 / (1 + hoursSinceLastSurfaced)` — uses LastSurfaced if set, else UpdatedAt
- **Spread:** `log(1 + len(surfacingContexts))` — diversity of contexts
- **Effectiveness:** `max(0.1, effectivenessScore/100)` — defaults to 0.5 when no data
- **Combined:** For BM25 modes: `bm25Score × (1 + activation)`. For SessionStart: pure activation.

Design choices:
- **Pure function:** No I/O, no side effects. Injected time and effectiveness data.
- **Logarithmic scaling:** Prevents high-frequency or high-spread memories from dominating.
- **Time decay:** Recent usage boosts ranking; old memories decay but never reach zero.
- **Effectiveness floor:** 0.1 prevents zero-multiplication from penalizing never-evaluated memories.
- **Default effectiveness 0.5:** Neutral — neither boosted nor penalized when no evaluation data exists.

**Traces to:** REQ-46 (frecency formula), REQ-9 (SessionStart frecency), REQ-10 (prompt BM25 + frecency), REQ-11 (tool BM25 + frecency)

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
    Extractor      TranscriptExtractor  // LLM: stripped delta → []CandidateLearning
    Retriever      MemoryRetriever      // file I/O: data-dir → existing memories (ARCH-9 reuse)
    Deduplicator   Deduplicator         // keyword overlap: candidates × existing → filtered candidates
    Writer         MemoryWriter         // file I/O: CandidateLearning → file path (ARCH-4 reuse)
    CreationLogger CreationLogger       // optional: log creation events for deferred visibility (ARCH-21)
}

func (l *Learner) Run(ctx context.Context, strippedDelta string, dataDir string) ([]string, error) {
    // 1. Extractor.Extract(ctx, strippedDelta): LLM call on delta (not full transcript) → []CandidateLearning
    // 2. Retriever.ListMemories(ctx, dataDir): read existing memory files
    // 3. Deduplicator.Filter(candidates, existing): remove duplicates by keyword overlap
    // 4. For each surviving candidate:
    //    a. Writer.Write(candidate, dataDir) → file path
    //    b. CreationLogger.Append({timestamp, title, tier, filename}, dataDir) — fire-and-forget
    // 5. Return list of created file paths (for stderr feedback)
}
```

Five stages (four existing + optional creation logger), each independently testable via DI. Reuses MemoryRetriever (ARCH-9) and MemoryWriter (ARCH-4). CreationLogger is optional — if nil, no creation log is written (backward compatible). Input is now stripped delta (not full transcript) per ARCH-31.

**Traces to:** REQ-15 (delta extraction), REQ-17 (dedup), REQ-3 (file writing via ARCH-4), REQ-20 (CLI entry), REQ-25 (creation log write)

---

## ARCH-15: Transcript Delta Extraction via LLM with Unified Tier Criteria

**Decision:** Single LLM call to extract multiple learnings from the stripped transcript delta (not full transcript), classifying each using the same A/B/C tier criteria as the real-time classifier (ARCH-2).

```go
type TranscriptExtractor interface {
    Extract(ctx context.Context, strippedDelta string) ([]CandidateLearning, error)
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
1. Instructs the LLM to review the **delta** (new content since last extraction) and extract actionable learnings.
2. Defines the JSON array output format (each element has all CandidateLearning fields, including `tier`).
3. Includes the same A/B/C tier definitions as the real-time classifier: A = explicit instruction, B = teachable correction, C = contextual fact.
4. Embeds anti-pattern gating: A → always generate anti_pattern, B → when generalizable, C → never.
5. Embeds the quality gate (REQ-16): explicitly reject mechanical patterns, vague generalizations, and overly narrow observations.
6. Instructs extraction of: missed corrections, architectural decisions, discovered constraints, working solutions, implicit preferences.

Input is the stripped transcript delta (low-value content removed per REQ-27). Returns `ErrNoToken` if no token is configured (REQ-18 — fail loud). Returns an error if the LLM response cannot be parsed. Returns empty slice if LLM finds no learnings worth extracting.

**Traces to:** REQ-15 (delta extraction with tier classification), REQ-16 (quality gate), REQ-7 (unified tier criteria + anti-pattern gating), REQ-18 (fail loud), REQ-27 (preprocessing)

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

## ARCH-17: CLI Learn Subcommand with Incremental Offset Tracking

**Decision:** Extend the engram binary with a `learn` subcommand that reads transcript from file and tracks extraction offset per session.

```bash
engram learn --data-dir <path> --transcript-path <file> --session-id <id>
```

Routing in `internal/cli/`:
- Parse args for `learn` subcommand with `--data-dir`, `--transcript-path`, `--session-id` flags.
- Construct IncrementalLearner (ARCH-39) which handles:
  1. Offset lookup and session ID validation
  2. Transcript delta reading from file
  3. Preprocessing (Strip)
  4. Learner pipeline execution (ARCH-14)
  5. Offset update
- Emit DES-10 format to stderr.

Design choices:
- **File path for transcript:** Enables offset tracking across multiple invocations.
- **Session ID:** Distinguishes separate sessions — new session ID resets offset to 0.
- **Stderr for feedback:** Session may be ending (Stop hook). Stdout is reserved for hook protocol.

**Traces to:** REQ-20 (CLI learn subcommand), REQ-26 (offset tracking), REQ-27 (preprocessing), DES-10 (feedback format)

---

## ARCH-18: Hook Script Integration for PreCompact and Stop

**Decision:** Two new hook scripts invoke `engram learn` with incremental offset tracking. Registered in `hooks/hooks.json`.

Hook flow:
- **PreCompact:** `hooks/pre-compact.sh` extracts transcript_path and session_id from stdin JSON, invokes `engram learn --data-dir <path> --transcript-path <file> --session-id <id>`. Stderr feedback visible in logs.
- **Stop:** `hooks/stop.sh` extracts transcript_path and session_id from stdin JSON, invokes `engram learn --data-dir <path> --transcript-path <file> --session-id <id>`. Stderr feedback visible in logs.

Both scripts:
1. Extract transcript_path and session_id from stdin JSON payload.
2. Use same token retrieval as DES-3 (macOS Keychain fallback to env var).
3. Invoke `engram learn --data-dir "$ENGRAM_DATA" --transcript-path "$TRANSCRIPT_PATH" --session-id "$SESSION_ID"`.
4. Exit 0 always (synchronous, fire-and-forget per ARCH-6).

Design choices:
- **Two separate scripts:** Allow future divergence if needed.
- **Synchronous execution:** Both hooks must complete before context compaction (PreCompact) or session termination (Stop).
- **Same binary:** Both invoke the same `engram learn` subcommand.

**Traces to:** DES-9 (hook wiring), REQ-19 (idempotency — multiple extractions in same session deduplicate), REQ-26 (offset tracking)

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
2. Input: stripped transcript + list of surfaced memories (title, principle, anti_pattern, content)
3. Output: JSON array with one entry per surfaced memory
4. Each memory gets exactly one outcome classification
5. Evidence field captures LLM's reasoning
6. CLI wiring applies `sessionctx.Strip` to transcript before passing to Evaluate (removes toolResult bodies, base64 data, truncates long lines — same preprocessing as learn and context-update pipelines)

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

---

## ARCH-36: Maintain Proposal Generator

**Decision:** New `internal/maintain/` package with a `Generator` struct that orchestrates proposal generation across all four quadrants.

**Component:** `maintain.Generator`

```go
type Generator struct {
    llmCaller func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error)
    now       func() time.Time
}

type Proposal struct {
    MemoryPath  string          `json:"memory_path"`
    Quadrant    string          `json:"quadrant"`
    Diagnosis   string          `json:"diagnosis"`
    Action      string          `json:"action"`
    Details     json.RawMessage `json:"details"`
}

func (g *Generator) Generate(
    ctx context.Context,
    classified []review.ClassifiedMemory,
    memories map[string]*memory.Stored,
) []Proposal
```

**Behavioral contracts:**
- Iterates classified memories, dispatches to quadrant-specific handlers
- Skips `InsufficientData` memories (REQ-47)
- Working: deterministic staleness check (REQ-48)
- Leech: LLM call per memory (REQ-49)
- Hidden gem: LLM call per memory (REQ-50)
- Noise: deterministic evidence assembly (REQ-51)
- LLM failures omit that proposal, don't block others (REQ-52)
- Returns slice of Proposals (may be empty)

**DI table:**

| Interface/Function | Production impl | Test impl |
|---|---|---|
| `llmCaller` | `makeAnthropicCaller(token)` | Fake returning canned JSON |
| `now` | `time.Now` | Fixed time |

**Traces to:** REQ-47, REQ-48, REQ-49, REQ-50, REQ-51, REQ-52, DES-24, DES-25

---

## ARCH-37: Maintain CLI Wiring

**Decision:** New `RunMaintain` function in `internal/cli/cli.go` following the established pattern (RunReview, RunEvaluate).

**Wiring flow:**
1. Parse `--data-dir` flag
2. Aggregate effectiveness: `effectiveness.New(evalDir).Aggregate()`
3. Build tracking map: `buildTrackingMap(dataDir)` (reuse from RunReview)
4. Classify: `review.Classify(stats, tracking)`
5. Build memory lookup: `listMemories → map[filePath]*memory.Stored`
6. Create generator: `maintain.New(opts...)` with LLM caller from API key
7. Generate proposals: `generator.Generate(ctx, classified, memoryMap)`
8. Encode JSON to stdout

**No API key behavior:** If `ANTHROPIC_API_KEY` is empty, create generator without LLM caller. Generator skips leech/hidden-gem proposals (REQ-53).

**Traces to:** REQ-53, REQ-54, DES-23

---

## ARCH-38: Learn Offset Tracking Infrastructure

**Decision:** New `internal/learn/offset.go` package for persistent offset tracking per session.

```go
type OffsetStore interface {
    ReadOffset(sessionID string) (int64, error)
    WriteOffset(sessionID string, offset int64) error
}

type LearnOffset struct {
    // maps session_id -> byte_offset
}
```

Implementation uses JSON file at `<data-dir>/learn-offset.json`:
```json
{
    "session-123": 5432,
    "session-456": 8901
}
```

Atomicity: write to temp file, then rename to final path. DI interfaces WithReadFile, WithWriteFile for testability.

**Design choices:**
- **Per-session tracking:** Different sessions can be processed independently.
- **JSON format:** Human-readable, easy to inspect for debugging.
- **Atomic writes:** Prevent corruption from concurrent writes.
- **New session detection:** If session_id not in map, treat as new session (offset = 0).

**Traces to:** REQ-26 (offset tracking), REQ-27 (delta computation), UC-1 incremental learning

---

## ARCH-39: IncrementalLearner Orchestrator

**Decision:** New `internal/learn/incremental.go` package that orchestrates incremental extraction: delta reading → stripping → learning → offset update.

```go
type IncrementalLearner struct {
    TranscriptPath  string
    SessionID       string
    DataDir         string
    OffsetStore     OffsetStore
    DeltaReader     DeltaReader       // from internal/context (ARCH-28 reuse for UC-14)
    Stripper        Stripper          // from internal/context (ARCH-29 reuse for UC-14)
    Learner         Learner           // ARCH-14 pipeline
}

func (il *IncrementalLearner) Run(ctx context.Context) ([]string, error) {
    // 1. il.OffsetStore.ReadOffset(il.SessionID) → currentOffset
    // 2. If currentOffset offset not found, set to 0 (new session)
    // 3. il.DeltaReader.ReadDelta(il.TranscriptPath, currentOffset) → delta
    // 4. If delta is empty, return [] (skip extraction)
    // 5. il.Stripper.Strip(delta) → strippedDelta
    // 6. il.Learner.Run(ctx, strippedDelta, il.DataDir) → createdPaths
    // 7. il.OffsetStore.WriteOffset(il.SessionID, newFileSize) — atomic update
    // 8. return createdPaths
}
```

**Reuses:**
- `DeltaReader` from `internal/context` (ARCH-28, UC-14)
- `Stripper` from `internal/context` (ARCH-29, UC-14)
- `Learner` from `internal/learn` (ARCH-14, UC-1)

**Design choices:**
- **Session boundary handling:** Detects new sessions by session_id. Resets offset to 0 on new session.
- **Empty delta optimization:** Skips API call if delta is empty (no cost for idle periods).
- **Atomic offset update:** Offset written after extraction completes successfully.
- **Fire-and-forget errors:** Offset write errors logged to stderr, don't fail learning operation (ARCH-6).

**Traces to:** REQ-26 (offset tracking), REQ-27 (preprocessing), UC-1 incremental extraction

---

## ARCH-40: Token Estimation and Budget Enforcement

**Decision:** Token estimation is a pure stateless function. Budget configuration is loaded at startup and stored in a Configuration struct. Budget enforcement happens in surface.go's matching functions.

```go
// Token estimation: pure function, called for each memory before adding to output
func estimateTokens(text string) int {
    return len(text) / 4
}

// Budget configuration: loaded from config.toml or env vars
type BudgetConfig struct {
    SessionStartBudget  int  // default 800
    UserPromptBudget    int  // default 300
    PreToolBudget       int  // default 200
    PostToolBudget      int  // default 100
    StopBudget          int  // default 500
}

// In matchPromptMemories and matchToolMemories:
// 1. Sort memories by (effectiveness × relevance)
// 2. Accumulate tokens for each memory: tokens += estimateTokens(memory.Content)
// 3. Stop adding when tokens + nextMemoryTokens > budget
// 4. Remaining memories are silently skipped
```

**Design choices:**
- **Token estimation:** Formula `len(text) / 4` is conservative and avoids external tokenizer dependencies.
- **Budget configuration:** Loaded from `<data-dir>/config.toml` with fallback to defaults. Config is immutable after loading.
- **Priority sorting:** `effectiveness × relevance` combines historical quality (effectiveness) with query relevance (BM25).
- **Budget is hard cap:** Once budget is exhausted, no more memories are added. No spilling over.
- **Silent cutoff:** Memories that don't fit are not logged to preserve performance.

**Traces to:** REQ-55 (token estimation), REQ-56 (budget caps), REQ-57 (priority allocation)

---

## ARCH-41: Budget Reporting and Warning Detection

**Decision:** Budget statistics are computed during `engram review` by analyzing surfacing logs for the session. Warnings are raised when cap hit rate exceeds 50%.

```go
// In engram review command:
// 1. Load surfacing logs for session: <data-dir>/logs/surfacing-<session-id>.json
// 2. For each hook, count: total_invocations, cap_hits (invocations where tokens > budget)
// 3. Compute cap_hit_rate = cap_hits / total_invocations
// 4. For each hook, output budget table row with utilization % and warning if cap_hit_rate > 0.5

type BudgetStats struct {
    Hook           string
    Budget         int
    TokensSurfaced int
    Utilization    float64  // percentage
    CapHitRate     float64  // percentage
    Warning        bool     // true if > 50%
}
```

**Design choices:**
- **Surfacing log structure:** Each invocation records {hook, budget_cap, tokens_surfaced, timestamp}.
- **Warning threshold:** 50% means the hook is hitting budget on more than half of invocations. This is a signal to review memory quality or increase budget.
- **Reporting format:** Human-readable table with hook names, caps, utilization %, and warning indicator.
- **No retroactive changes:** Reporting is advisory only; budgets aren't auto-adjusted.

**Traces to:** REQ-58 (budget reporting), REQ-59 (warning detection)

---

## ARCH-42: Stop Hook Audit Phase

**Decision:** Stop hook is expanded to a 4-phase pipeline with audit as the 3rd phase. New `engram audit` command is invoked between `engram evaluate` and `engram context-update`.

```bash
# hooks/stop.sh phase ordering:
engram learn --transcript-path "$TRANSCRIPT_PATH" --session-id "$SESSION_ID"  # Phase 1
engram evaluate --data-dir "$DATA_DIR"                                       # Phase 2
engram audit --data-dir "$DATA_DIR" --timestamp "$(date ...)"               # Phase 3 (NEW)
engram context-update --data-dir "$DATA_DIR"                                 # Phase 4

# Each phase runs; errors are logged but don't block next phase (fire-and-forget)
```

**Design choices:**
- **Audit timing:** After effectiveness evaluation (so audit can access updated scores) and before context-update (so audit results don't affect next session's initialization).
- **Fire-and-forget errors:** If audit fails (e.g., no API token), error is logged to stderr but other phases continue.
- **Single audit per session:** One `engram audit` call per Stop hook invocation.

**Traces to:** REQ-60 (stop hook timing)

---

## ARCH-43: Audit Command Structure

**Decision:** New CLI subcommand `engram audit --data-dir <path> --timestamp <iso8601>`. Reads session data (surfacing logs, effectiveness data, transcript), invokes Haiku for compliance assessment, writes audit report, and injects results into effectiveness pipeline.

```go
// cmd/engram/audit.go
func runAudit(ctx context.Context, dataDir, timestamp string) error {
    // 1. Parse surfacing logs: extract high-priority memories from session
    // 2. Load effectiveness data: get effectiveness scores for those memories
    // 3. Read transcript (passed via stdin or file path)
    // 4. Build audit scope JSON: [{memory_id, effectiveness_score, ...}, ...]
    // 5. Call Haiku with scope + transcript: get compliance assessments
    // 6. Write audit report: <data-dir>/audits/<timestamp>.json
    // 7. Inject results into effectiveness: call evaluate.InjectAuditResults(report)
    // 8. Return error if API fails; log to stderr, exit 1 (fire-and-forget in hook)
}
```

**Command signature:**
```bash
engram audit --data-dir <path> --timestamp <iso8601> [--transcript-path <path>|--transcript-stdin]
```

**Design choices:**
- **Audit scope:** Only high-priority memories (top 20% by effectiveness score) are included in scope.
- **LLM call:** Single Haiku call assesses compliance across all scope items.
- **Atomic report write:** Report is written to `audits/<timestamp>.json` after LLM assessment succeeds.
- **Effectiveness injection:** After report write, results are injected into effectiveness data as outcome signals.

**Traces to:** REQ-61 (scope), REQ-62 (LLM assessment), REQ-63 (report format)

---

## ARCH-44: Effectiveness Signal Injection from Audit Results

**Decision:** After audit report is written, compliance results are fed into the effectiveness evaluation pipeline as negative outcome signals for non-compliant memories.

```go
// internal/evaluate/evaluate.go
func InjectAuditResults(auditReport *AuditReport) error {
    // 1. Parse audit report JSON
    // 2. For each non_compliant result:
    //    - Look up memory ID in effectiveness registry
    //    - Add negative outcome: {outcome_type: "audit_non_compliance", timestamp: audit_timestamp, outcome_value: -1}
    //    - Save updated effectiveness data
    // 3. Return nil (errors logged to stderr, non-fatal)
}

type OutcomeSignal struct {
    OutcomeType string    // "audit_non_compliance"
    Timestamp   time.Time
    Value       int       // -1 for non-compliance
    Evidence    string    // from audit report
}
```

**Design choices:**
- **Negative signal:** Non-compliance lowers the memory's follow rate in future effectiveness aggregations.
- **Signal strength:** -1 per violation (proportional to frequency of violations in evaluations).
- **No lookup failure:** If memory ID doesn't exist in effectiveness data, signal is skipped (non-fatal).
- **Permanent record:** Audit results persist in effectiveness history for visibility into long-term patterns.

**Traces to:** REQ-64 (effectiveness integration)

---

## ARCH-45: Audit Error Handling (No Graceful Degradation)

**Decision:** If Haiku API call fails (missing/invalid token), audit phase emits error to stderr and skips the audit. No report is written. Other Stop hook phases continue.

```go
// In runAudit:
if token := os.Getenv("ANTHROPIC_API_KEY"); token == "" {
    fmt.Fprintf(os.Stderr, "audit: API token missing or invalid, skipping audit\n")
    return ErrMissingToken  // exit code 1
}

// Hook script ignores exit code (fire-and-forget pattern)
// Other phases continue even if audit fails
```

**Design choices:**
- **Fail-open:** If audit can't run, the system continues without degradation.
- **Error is visible:** stderr message indicates the audit was skipped.
- **No partial reports:** If LLM call fails mid-assessment, no report is written (atomic behavior).
- **Hook resilience:** Stop hook continues even if one phase fails.

**Traces to:** REQ-65 (error handling)

---

## L2 → ARCH Traceability (UC-16)

| L2 Item | ARCH Coverage |
|---------|--------------|
| REQ-47 | ARCH-36 |
| REQ-48 | ARCH-36 |
| REQ-49 | ARCH-36 |
| REQ-50 | ARCH-36 |
| REQ-51 | ARCH-36 |
| REQ-52 | ARCH-36 |
| DES-24 | ARCH-36 |
| DES-25 | ARCH-36 |
| REQ-53 | ARCH-37 |
| REQ-54 | ARCH-37 |
| DES-23 | ARCH-37 |

All UC-16 L2 items have ARCH coverage.

---

## L2 → ARCH Traceability (UC-17 & UC-19)

| L2 Item | ARCH Coverage |
|---------|--------------|
| REQ-55 | ARCH-40 |
| DES-16 | ARCH-40 |
| REQ-56 | ARCH-40 |
| DES-17 | ARCH-40 |
| REQ-57 | ARCH-40 |
| DES-18 | ARCH-40 |
| REQ-58 | ARCH-41 |
| DES-19 | ARCH-41 |
| REQ-59 | ARCH-41 |
| DES-20 | ARCH-41 |
| REQ-60 | ARCH-42 |
| DES-21 | ARCH-42 |
| REQ-61 | ARCH-43 |
| DES-22 | ARCH-43 |
| REQ-62 | ARCH-43 |
| DES-23 | ARCH-43 |
| REQ-63 | ARCH-43 |
| REQ-64 | ARCH-44 |
| DES-24 | ARCH-44 |
| REQ-65 | ARCH-45 |

All UC-17 & UC-19 L2 items have ARCH coverage.

---

## ARCH-46: PostToolUse Reminder Pipeline

**Decision:** New reminder pipeline triggered by PostToolUse hook. Reads tool call details from stdin, matches file path against pattern config, resolves instruction IDs, checks suppression, emits single capped reminder.

```go
// internal/remind/remind.go
type Reminder struct {
    ConfigReader  func(path string) ([]byte, error)  // reads reminders.toml
    MemoryLoader  func(dataDir string) ([]Memory, error)
    TranscriptReader func(path string, maxTokens int) (string, error)
    EstimateTokens func(text string) int  // reuse from surface
}

func (r *Reminder) Run(ctx context.Context, toolCall ToolCallInput) (string, error) {
    // 1. Load pattern config from reminders.toml
    // 2. Match toolCall.FilePath against glob patterns
    // 3. No match → return "" (no reminder)
    // 4. Resolve instruction IDs → load memories/rules/CLAUDE.md entries
    // 5. Select highest-effectiveness instruction
    // 6. Check suppression: read recent transcript, look for compliance evidence
    // 7. If suppressed → return ""
    // 8. Cap at 100 tokens, format as "[engram] Reminder: <text>"
    // 9. Log to surfacing log for effectiveness tracking
    // 10. Return formatted reminder
}
```

**Design choices:**
- **Single reminder per invocation:** Not a dump — select the highest-effectiveness match.
- **Suppression before emission:** Check transcript for compliance evidence using keyword matching (no LLM).
- **Budget from UC-17:** Reuse estimateTokens for the 100-token cap.
- **Effectiveness tracking:** Log to surfacing log (same infrastructure as UC-2/UC-15).

**Traces to:** REQ-66 (hook), REQ-67 (patterns), REQ-68 (sourcing), REQ-69 (budget), REQ-70 (suppression), REQ-71 (tracking)

---

## ARCH-47: Reminder Hook Wiring and Configuration

**Decision:** PostToolUse hook registered in hooks/hooks.json. Hook script reads stdin JSON, extracts file_path and tool_name, invokes `engram remind`. Configuration in reminders.toml is optional — missing config means no reminders.

```bash
# hooks/post-tool-use-remind.sh
INPUT=$(cat)
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // empty')
TOOL_NAME=$(echo "$INPUT" | jq -r '.tool_name // empty')

if [ -z "$FILE_PATH" ]; then exit 0; fi
if [ "$TOOL_NAME" != "Write" ] && [ "$TOOL_NAME" != "Edit" ]; then exit 0; fi

RESULT=$(echo "$INPUT" | engram remind --data-dir "$DATA_DIR" --file-path "$FILE_PATH")
if [ -n "$RESULT" ]; then
  echo "{\"hookSpecificOutput\":{\"additionalContext\":\"$RESULT\"}}"
fi
```

**Traces to:** REQ-66 (hook registration), DES-26 (config format), DES-27 (output format)

---

## ARCH-48: Cross-Source Instruction Scanner

**Decision:** New scanner that reads all instruction sources and produces a unified instruction registry. Each instruction is a structured item with source metadata, content, and effectiveness data.

```go
// internal/instruct/scanner.go
type InstructionItem struct {
    Source     string   // "memory", "claude_md", "rule", "skill"
    Path       string   // file path
    ID         string   // unique identifier
    Text       string   // instruction content
    Keywords   []string
    Effectiveness float64  // from evaluation data, 0 if unavailable
}

type Scanner struct {
    ReadFile    func(path string) ([]byte, error)
    GlobFiles   func(pattern string) ([]string, error)
    EffData     map[string]float64  // memory_id → effectiveness score
}

func (s *Scanner) ScanAll(dataDir, projectDir string) ([]InstructionItem, error) {
    // 1. Scan CLAUDE.md (project + global) → extract by section header
    // 2. Scan <data-dir>/memories/*.toml → extract title, principle, keywords
    // 3. Scan .claude/rules/*.md → extract by file
    // 4. Scan plugin skill directories → extract by file
    // 5. Join effectiveness data where available
    // 6. Return unified list
}
```

**Traces to:** REQ-72 (scanning), REQ-73 (dedup input), REQ-76 (gap analysis input)

---

## ARCH-49: Instruct Audit Pipeline

**Decision:** Pipeline for deduplication, quality diagnosis, refinement, gap analysis, and skill decomposition. LLM-dependent steps are skippable (no API token).

```go
// internal/instruct/audit.go
type Auditor struct {
    Scanner    *Scanner
    LLMCaller  func(ctx, prompt) (string, error)  // nil = skip LLM steps
    EvalData   map[string][]EvalRecord
}

func (a *Auditor) Run(ctx context.Context) (*AuditReport, error) {
    // 1. ScanAll → unified instruction list
    // 2. Deduplicate: pairwise keyword overlap >80% → duplicate pairs
    // 3. If LLMCaller != nil:
    //    a. Quality diagnosis: bottom 20% → Haiku diagnosis
    //    b. Refinement proposals: rewrite diagnosed instructions
    // 4. Gap analysis: contradicted evals without matching instruction
    // 5. Skill decomposition: per-line effectiveness for skill files
    // 6. Return AuditReport JSON
}
```

**Design choices:**
- **LLM-optional:** Dedup, gaps, skill decomposition work without API token.
- **Salience hierarchy:** CLAUDE.md > rules > memories for dedup recommendations.
- **Maintain-compatible proposals:** Memory proposals match UC-16 format.

**Traces to:** REQ-74 (diagnosis), REQ-75 (proposals), REQ-76 (gaps), REQ-77 (skills), REQ-78 (CLI), REQ-79 (error handling)

---

## L2 → ARCH Traceability (UC-18 & UC-20)

| L2 Item | ARCH Coverage |
|---------|--------------|
| REQ-66 | ARCH-46, ARCH-47 |
| DES-26 | ARCH-47 |
| REQ-67 | ARCH-46 |
| REQ-68 | ARCH-46 |
| REQ-69 | ARCH-46 |
| DES-27 | ARCH-47 |
| REQ-70 | ARCH-46 |
| DES-28 | ARCH-46 |
| REQ-71 | ARCH-46 |
| REQ-72 | ARCH-48 |
| REQ-73 | ARCH-48, ARCH-49 |
| REQ-74 | ARCH-49 |
| DES-29 | ARCH-49 |
| REQ-75 | ARCH-49 |
| REQ-76 | ARCH-49 |
| REQ-77 | ARCH-49 |
| REQ-78 | ARCH-49 |
| REQ-79 | ARCH-49 |

All UC-18 & UC-20 L2 items have ARCH coverage.

---

## ARCH-50: Escalation Engine

**Decision:** Extend `engram maintain` with an escalation engine that analyzes leech memories and proposes graduated enforcement changes within engram's advisory range. Integrates with UC-18 (PostToolUse reminders). Ladder is 3 levels: advisory → emphasized_advisory → reminder. Beyond `reminder`, a graduation signal is emitted (UC-28). Dimension routing and automation candidate routing removed in S2.

```go
// internal/maintain/escalation.go
type EscalationEngine struct {
    EffData  map[string][]float64  // effectiveness history per level
}

type EscalationProposal struct {
    MemoryPath      string
    ProposalType    string  // "escalate", "de_escalate"
    CurrentLevel    string
    ProposedLevel   string
    Rationale       string
    PredictedImpact string
}

func (e *EscalationEngine) Analyze(leeches []Memory) ([]EscalationProposal, error) {
    // 1. For each leech memory:
    //    a. Check escalation_history for de-escalation signal
    //    b. If post-escalation effectiveness < pre for ≥3 cycles → de-escalate
    //    c. Otherwise: propose escalation to next level (advisory→emphasized_advisory→reminder)
    //    d. If already at reminder: no proposal (graduation signal emitted via UC-28)
    //    e. Compute predicted impact from historical data
    // 2. Return proposals (user confirms separately)
}
```

**Design choices:**
- **Three levels only:** advisory, emphasized_advisory, reminder. No blocking hooks or automation candidates.
- **De-escalation threshold:** 3 cycles prevents noise from triggering premature de-escalation.
- **Predicted impact:** Based on average effectiveness delta for memories at the proposed level. "Unknown" if no historical data.
- **TOML schema:** escalation_level field + escalation_history array are optional (default = advisory).

**Traces to:** REQ-80 (levels), REQ-81 (proposals), REQ-82 (de-escalation), REQ-84 (TOML), REQ-85 (confirmation)

---

## ARCH-51: Automation Generator Pipeline *(removed — Phase A-1/S1)*

**Status:** Removed. UC-22 removed in S1. The `internal/automate/` package and `engram automate` CLI command were deleted. Graduation signals (UC-28/ARCH-53) replace the automation proposal concept.

**Traces to:** REQ-90 (removed), REQ-91 (removed)

---

## L2 → ARCH Traceability (UC-21 & UC-22)

| L2 Item | ARCH Coverage |
|---------|--------------|
| REQ-80 | ARCH-50 |
| DES-30 | ARCH-50 |
| REQ-81 | ARCH-50 |
| DES-31 | ARCH-50 |
| REQ-82 | ARCH-50 |
| REQ-83 | ARCH-50 |
| REQ-84 | ARCH-50 |
| REQ-85 | ARCH-50 |
| REQ-86 | ARCH-51 |
| REQ-87 | ARCH-51 |
| DES-32 | ARCH-51 |
| REQ-88 | ARCH-51 |
| REQ-89 | ARCH-51 |
| REQ-90 | ARCH-51 |
| REQ-91 | ARCH-51 |

All UC-21 & UC-22 L2 items have ARCH coverage.

---

## UC-23: Unified Instruction Registry (ARCH Items)

### ARCH-52: Registry Storage — JSONL with one line per instruction ID

Registry persists to `data/instruction-registry.jsonl` with one line per registered instruction. Each line is a complete JSON object with all current state (counts, timestamps, absorbed history). Updates rewrite the entire line atomically. File size is O(num_instructions), not O(events).

**Rationale:** Bounded growth required by REQ-55. JSONL structure allows efficient reading into memory for all typical registry operations (< 10K instructions). Atomicity via full-line rewrites prevents partial updates under concurrent hook calls.

**Traces to:** REQ-55 (bounded growth), REQ-61 (atomic surfacing update), REQ-62 (atomic eval update), REQ-64 (concurrent writes)

---

### ARCH-53: Instruction ID Format — source_type:source_path:entry_slug

Instruction IDs are deterministic, globally unique, and human-readable. Format: `claude-md:CLAUDE-md:use-targ-build`, `memory:always-use-targ-reminder.toml`, `rule:go.md`, `memory-md:MEMORY.md:di-pattern`, `skill:commit`, `hook:PreToolUse:surface`. ID uniqueness is enforced at registration time.

**Rationale:** Supports all 6 source types. Slug-based (not UUID) for debuggability. Hierarchical structure (source_type as prefix) enables salience-based filtering and source-specific operations.

**Traces to:** REQ-56 (6 source types)

---

### ARCH-54: Signal Computation — Effectiveness and Frecency on Read

Effectiveness is computed on read: `followed / (followed + contradicted + ignored)`, null if denominator < threshold (e.g., 3 evaluations). Frecency is computed on read: `surfaced_count * exp(-t / half_life)` where t is days since last_surfaced. Both are deterministic, idempotent, and require no auxiliary storage.

**Rationale:** Avoids storing derived quantities that can go stale. Computation is O(1) per instruction. Thresholds are configurable per deployment.

**Traces to:** REQ-57 (effectiveness ratio), REQ-58 (frecency blend)

---

### ARCH-55: Content Hash — SHA256 of Instruction Content

Each registered instruction stores content_hash (SHA256 of the instruction text). If content changes (e.g., memory TOML edited, CLAUDE.md entry rewritten), content_hash changes. Change detection happens during evaluation: if current content hash differs from stored hash, instruction is flagged for re-evaluation.

**Rationale:** Enables detection of manual edits to memory files and CLAUDE.md entries. Supports deduplication and content-based merging in future expansions.

**Traces to:** REQ-59 (content hash detection)

---

### ARCH-56: Absorbed History — Array of Merge Records with Counters

When `engram registry merge --source X --target Y` runs, all of X's state becomes one element in Y's `absorbed` array: `{ from: "X", surfaced_count, evaluations: {followed, contradicted, ignored}, merged_at, content_hash }`. Multiple absorbed entries allowed (multiple duplicates merged into same target). Absorbed records are immutable after creation.

**Rationale:** Preserves violation history of deleted duplicates. Feeds escalation engine (chronic leeches identified by large absorbed history + low current effectiveness). Supports attribution reporting.

**Traces to:** REQ-60 (merge history), REQ-63 (merge semantics)

---

### ARCH-57: Concurrent Write Safety — Read-All, Write-Full Strategy

Registry uses simple concurrency model: read entire file on load, apply changes in memory, write entire file on update. No lock files or atomic rename tricks needed. For up to ~10K instructions, this is safe: worst case is two concurrent writes where one loses a small frequency delta (acceptable loss for occasional duplication).

**Rationale:** Simplicity over perfect consistency. Engram is single-user, single-machine. Hooks are bursty (PreCompact, Stop) not sustained. Loss of a few frequency deltas is acceptable and self-heals over time.

**Traces to:** REQ-64 (concurrent writes)

---

### ARCH-58: Backfill Migration — Aggregate Old Stores, Map Retired Instructions

`engram registry init` command performs Phase 1 backfill:
1. Read surfacing-log.jsonl, aggregate per-memory: sum surfaced_count, take max last_surfaced
2. Read creation-log.jsonl: set registered_at per memory
3. Read evaluations/*.jsonl: sum followed/contradicted/ignored per memory
4. Read memory TOML files: extract metadata fields
5. For each retired memory (has retired_by): identify covering instruction via semantic or ID match. Append retired memory's counters as one absorbed entry in covering instruction.
6. Write instruction-registry.jsonl with all aggregated data.

**Rationale:** Centralized migration plan. Maps duplicates to their covering instructions via retired_by metadata, preserving violation history. Provides --dry-run flag for verification before committing.

**Traces to:** REQ-65 (backfill migration), REQ-66 (retirement mapping)

---

### ARCH-59: Quadrant Classification — Source-Aware, Salience-Respecting

Classification logic reads registry and assigns each instruction to one of four quadrants:
- **Working:** high surfacing + high effectiveness
- **Leech:** high surfacing + low effectiveness
- **HiddenGem:** low surfacing + high effectiveness
- **Noise:** low surfacing + low effectiveness
- **Always-loaded sources (claude-md, memory-md):** binary only (Working or Leech), since they have maximal surfacing by definition

Thresholds are configurable. Always-loaded sources that are Leech (e.g., a CLAUDE.md entry with contradicted > followed) trigger rewrite proposals, same as memories.

**Rationale:** Unified classification across all sources. Enables surfacing of leeches in high-salience sources (CLAUDE.md, rules) that were previously invisible. Foundation for next UC batch (UC-4..10).

**Traces to:** REQ-67 (cross-source classification)

---

### ARCH-60: DI Boundary — Registry Interface in internal/registry

Registry abstraction lives in `internal/registry/registry.go` as an interface with methods:
- `Register(id, title, source_type, content_hash, registered_at) error`
- `RecordSurfacing(id) error`
- `RecordEvaluation(id, outcome) error` where outcome ∈ {followed, contradicted, ignored}
- `Merge(source_id, target_id) error`
- `Remove(id) error`
- `List() ([]InstructionEntry, error)`
- `Get(id) (*InstructionEntry, error)`

Concrete JSONL implementation lives outside internal/ (e.g., cli.go). Tests inject mock Registry. No os.*, io.*, or file operations in internal/.

**Rationale:** DI everywhere principle (ARCH-7). Enables testing without I/O. Concrete implementation can be swapped (future: SQL backend, cloud persistence).

**Traces to:** REQ-68 (DI boundary)

---

### ARCH-61: CLI Integration — Three New Subcommands + Hook Auto-Integration

Three new CLI subcommands:
- `engram registry init [--dry-run]`: backfill from old stores to instruction-registry.jsonl
- `engram review [--format json|table]`: read registry, classify by quadrant, output grouped by source + quadrant
- `engram registry merge --source <id> --target <id>`: absorb counters, delete source

Hooks auto-integrate (no new flags needed): surfacing hook calls `Registry.RecordSurfacing(id)`, evaluate hook calls `Registry.RecordEvaluation(id, outcome)`, learn pipeline calls `Registry.Register(...)`.

Fire-and-forget error handling: registry write failures log but don't crash hooks (ARCH-6 — hooks never block instruction delivery).

**Rationale:** User-facing commands (DES-26, 27, 28) backed by Registry interface. Auto-integration keeps memory management transparent. Fire-and-forget aligns with hook failure semantics.

**Traces to:** DES-26 (registry init), DES-27 (review), DES-28 (merge), DES-29 (auto-integration)

---

## L2 → ARCH Traceability (UC-23)

| L2 Item | ARCH Coverage |
|---------|--------------|
| REQ-55 | ARCH-52 |
| REQ-56 | ARCH-53 |
| REQ-57 | ARCH-54 |
| REQ-58 | ARCH-54 |
| REQ-59 | ARCH-55 |
| REQ-60 | ARCH-56 |
| REQ-61 | ARCH-52 |
| REQ-62 | ARCH-52 |
| REQ-63 | ARCH-56 |
| REQ-64 | ARCH-57 |
| REQ-65 | ARCH-58 |
| REQ-66 | ARCH-58 |
| REQ-67 | ARCH-59 |
| REQ-68 | ARCH-60 |
| DES-26 | ARCH-61 |
| DES-27 | ARCH-59, ARCH-61 |
| DES-28 | ARCH-56, ARCH-61 |
| DES-29 | ARCH-60, ARCH-61 |

All UC-23 L2 items have ARCH coverage.

---

## ARCH-62: Promoter Pipeline (UC-4)

**Decision:** New `internal/promote/` package with a `Promoter` struct orchestrating memory→skill promotion.

```go
type Promoter struct {
    Registry     RegistryReader   // List + Get from registry
    Generator    SkillGenerator   // LLM: memory → skill content
    SkillWriter  SkillWriter      // Write skill file to plugin dir
    Merger       RegistryMerger   // Merge source→target in registry
    MemoryRemover MemoryRemover   // Delete source memory TOML
    Confirmer    Confirmer        // User confirmation (stdin/stdout)
}

type RegistryReader interface {
    List() ([]registry.InstructionEntry, error)
    Get(id string) (registry.InstructionEntry, error)
}

type SkillGenerator interface {
    Generate(ctx context.Context, memory MemoryContent) (string, error)
}

type SkillWriter interface {
    Write(name string, content string) (string, error)  // returns path
}

type RegistryMerger interface {
    Merge(sourceID, targetID string) error
}

type MemoryRemover interface {
    Remove(path string) error
}

type Confirmer interface {
    Confirm(preview string) (bool, error)
}
```

**Candidate detection:** `Promoter.Candidates(threshold int)` filters registry entries where `source_type == "memory"` and `surfaced_count >= threshold` and quadrant is not Insufficient. Returns sorted by surfaced_count descending.

**Promote flow:** `Promoter.Promote(ctx, candidateID)` → get memory content → generate skill → show preview → confirm → write skill → register skill in registry → merge memory→skill → delete memory TOML. Atomic: if any step after confirmation fails, partial work is cleaned up.

**Traces to:** REQ-92 (threshold), REQ-93 (generation), REQ-94 (plugin registration), REQ-95 (retirement), REQ-96 (confirmation), DES-33 (CLI), DES-34 (format)

---

## ARCH-63: Skill Template Engine (UC-4)

**Decision:** Skill generation uses LLM with a structured prompt template. The LLM receives memory fields (title, content, principle, anti_pattern, keywords, concepts) and returns a complete skill file in the DES-34 format. The template is a Go `text/template` embedded in the generator.

**Fallback:** If no API token, `SkillGenerator.Generate` returns an error. The Promoter reports "no token" and skips. Candidate detection still works without a token.

**Skill ID format:** `skill:<slugified-title>` — consistent with UC-23 registry's source_type taxonomy (ARCH-53).

**Traces to:** REQ-93 (LLM generation), DES-34 (skill format), REQ-96 (no-token behavior)

---

## ARCH-64: Tier Transition Engine (UC-5)

**Decision:** New `internal/promote/` extension (same package as ARCH-62) with `ClaudeMDPromoter` struct. Reuses `SkillGenerator` (ARCH-63) for demotion path. Adds `ClaudeMDEditor` for CLAUDE.md file manipulation.

```go
type ClaudeMDPromoter struct {
    Registry       RegistryReader
    EntryGenerator ClaudeMDEntryGenerator  // LLM: skill → CLAUDE.md entry
    SkillGenerator SkillGenerator          // Reuse from ARCH-63 for demotion
    Editor         ClaudeMDEditor          // Parse + edit CLAUDE.md file
    SkillWriter    SkillWriter
    Merger         RegistryMerger
    Confirmer      Confirmer
}

type ClaudeMDEntryGenerator interface {
    Generate(ctx context.Context, skill SkillContent, existingClaudeMD string) (string, error)
}

type ClaudeMDEditor interface {
    AddEntry(content string, entry string) (string, error)    // returns new content
    RemoveEntry(content string, entryID string) (string, error)
}
```

**Promotion candidates:** `source_type == "skill"`, Working quadrant, `surfaced_count >= threshold`.

**Demotion candidates:** `source_type == "claude-md"`, Leech quadrant (binary: always-loaded sources only have Working/Leech per ARCH-59).

**Traces to:** REQ-97 (promotion detection), REQ-98 (entry generation), REQ-99 (demotion detection), REQ-100 (demotion execution), REQ-101 (registry merge), REQ-102 (confirmation), DES-35 (CLI), DES-36 (diff preview)

---

## ARCH-65: CLAUDE.md File Editor (UC-5)

**Decision:** `ClaudeMDEditor` implementation uses section-level parsing (split on `## ` headings). AddEntry appends a new section. RemoveEntry removes the section matching the entry ID (matched by comment marker `<!-- promoted from ... -->` or heading text). File writes are atomic (temp + rename).

**DI boundary:** `ClaudeMDEditor` interface in `internal/promote/`, concrete implementation at CLI edge (reads/writes actual files).

**Traces to:** REQ-100 (CLAUDE.md editing), DES-36 (diff preview)

---

## ARCH-66: Proposal Executor (UC-24)

**Decision:** New `internal/maintain/apply.go` extending the existing `internal/maintain/` package. `Executor` struct routes proposals by quadrant to strategy handlers.

```go
type Executor struct {
    Rewriter    MemoryRewriter     // Atomic TOML rewrite
    Remover     MemoryRemover      // Delete memory file
    Registry    RegistryUpdater    // Update content_hash or remove entry
    LLMCaller   LLMCaller          // For Working/Leech/HiddenGem rewrites
    Confirmer   Confirmer          // Per-proposal user confirmation
}

type MemoryRewriter interface {
    Rewrite(path string, updates map[string]interface{}) error
}

type RegistryUpdater interface {
    Register(entry registry.InstructionEntry) error  // re-register with new hash
    Remove(id string) error
}

type Proposal struct {
    Quadrant      string  // Working, Leech, HiddenGem, Noise
    Action        string  // update_content, rewrite, broaden_keywords, remove
    TargetPath    string  // memory TOML path
    TargetID      string  // registry instruction ID
    ProposedChange string // LLM-generated or deterministic
    Evidence      Evidence
}
```

**Strategy routing:** Working→`applyStaleUpdate`, Leech→`applyRewrite`, HiddenGem→`applyBroadenKeywords`, Noise→`applyRemoval`. Each strategy returns a diff string for confirmation display.

**Traces to:** REQ-103 (ingestion), REQ-104 (Working), REQ-105 (Leech), REQ-106 (HiddenGem), REQ-107 (Noise), REQ-108 (registry update), REQ-109 (confirmation), DES-37 (CLI), DES-38 (display)

---

## ARCH-67: Memory TOML Rewriter (UC-24)

**Decision:** `MemoryRewriter` in `internal/maintain/` reads existing TOML, applies field updates (content, keywords, concepts, principle, anti_pattern), writes atomically (temp + rename). Preserves all fields not being updated. Uses BurntSushi/toml for parse and encode.

**DI boundary:** File I/O injected via `WithReadFile`, `WithWriteFile` options (same pattern as track.Recorder, creationlog.LogWriter).

**Traces to:** REQ-104 (Working rewrite), REQ-105 (Leech rewrite), REQ-106 (keyword broadening)

---

## ARCH-68: Evaluate Strip Integration (UC-25)

**Decision:** Extend `internal/evaluate/Evaluator` with `WithStripFunc(fn func([]string) []string)` option. When set, the evaluator splits transcript into lines, applies the strip function, and rejoins before sending to LLM. Default is no-op (backward compatible).

```go
type Option func(*Evaluator)

func WithStripFunc(fn func([]string) []string) Option {
    return func(e *Evaluator) { e.stripFunc = fn }
}
```

CLI wiring in `runEvaluate`: `evaluate.WithStripFunc(sessionctx.Strip)`.

**Empty handling:** If post-strip content is empty, `Evaluate` returns early with no error and no LLM call. Stderr message logged.

**Traces to:** REQ-110 (strip injection), REQ-111 (empty handling), DES-39 (DI pattern)

---

## L2 → ARCH Traceability (UC-4, UC-5, UC-24, UC-25)

| L2 Item | ARCH Coverage |
|---------|--------------|
| REQ-92 | ARCH-62 |
| REQ-93 | ARCH-62, ARCH-63 |
| REQ-94 | ARCH-62 |
| REQ-95 | ARCH-62 |
| REQ-96 | ARCH-62, ARCH-63 |
| DES-33 | ARCH-62 |
| DES-34 | ARCH-63 |
| REQ-97 | ARCH-64 |
| REQ-98 | ARCH-64 |
| REQ-99 | ARCH-64 |
| REQ-100 | ARCH-64, ARCH-65 |
| REQ-101 | ARCH-64 |
| REQ-102 | ARCH-64 |
| DES-35 | ARCH-64 |
| DES-36 | ARCH-64, ARCH-65 |
| REQ-103 | ARCH-66 |
| REQ-104 | ARCH-66, ARCH-67 |
| REQ-105 | ARCH-66, ARCH-67 |
| REQ-106 | ARCH-66, ARCH-67 |
| REQ-107 | ARCH-66 |
| REQ-108 | ARCH-66 |
| REQ-109 | ARCH-66 |
| DES-37 | ARCH-66 |
| DES-38 | ARCH-66 |
| REQ-110 | ARCH-68 |
| REQ-111 | ARCH-68 |
| DES-39 | ARCH-68 |

All L2 items have ARCH coverage.

---

## ARCH-69: SourceRegistrar Component (UC-26)

**Decision:** New `internal/register/` package with `Registrar` struct that orchestrates the 4-phase auto-registration flow from DES-40.

```go
type SourceConfig struct {
    ClaudeMDPaths []string // e.g., ["./CLAUDE.md", "~/.claude/CLAUDE.md"]
    MemoryMDPaths []string // e.g., ["~/.claude/projects/<slug>/memory/MEMORY.md"]
    RulesDir      string   // e.g., ".claude/rules/"
    SkillsDir     string   // e.g., "<plugin-root>/skills/"
}

type Registrar struct {
    registry       Registry           // Register, Remove, List, RecordSurfacing
    surfacingLog   SurfacingLogger    // LogSurfacing (for evaluate pipeline)
    readFile       func(string) ([]byte, error)
    readDir        func(string) ([]os.DirEntry, error)
    glob           func(string) ([]string, error)
    now            func() time.Time
}

func (r *Registrar) Run(config SourceConfig) error {
    // 1. Discover: scan all configured paths, extract entries via crossref extractors (ARCH-79)
    // 2. Register: for each discovered entry, register or update content hash
    // 3. Prune: list registry, remove non-memory entries not in discovered set
    // 4. Record surfacing: for each always-loaded entry, call registry.RecordSurfacing
    //    AND surfacingLog.LogSurfacing (so evaluate pipeline picks them up)
}
```

**DI boundary:** All I/O injected — readFile, readDir, glob for source discovery; Registry interface for state management; SurfacingLogger interface for evaluate integration. No direct `os.*` calls.

**Key design choice:** Implicit surfacing writes to BOTH the registry (`RecordSurfacing` for long-term tracking) and the surfacing log (`LogSurfacing` for per-session evaluate input). The evaluator reads the surfacing log to know what was active this session.

**Traces to:** REQ-112 (source discovery), REQ-113 (auto-registration), REQ-114 (stale pruning), REQ-115 (implicit surfacing), REQ-118 (idempotency), DES-40 (flow ordering), DES-41 (source paths)

---

## ARCH-70: Extended Always-Loaded Classification (UC-26) — REMOVED (S4)

**Status:** Removed by S4 simplification. The `alwaysLoadedSources` map and binary Working/Leech classification for non-memory sources have been deleted from `internal/registry/classify.go`. The registry now only persists memory entries; `Classify` uses the standard 4-quadrant model for all entries.

**Traces to:** REQ-116 (extended classification) — REQ-116 is unsatisfied; to be revisited if non-memory classification is needed in future.

---

## ARCH-71: CLI Integration and Hook Wiring (UC-26)

**Decision:** Wire `Registrar.Run` into the existing `runSurface` function in `internal/cli/cli.go`, specifically in the `session-start` mode path. The registration runs before memory surfacing.

```go
func runSurface(...) {
    // NEW: auto-registration phase (session-start only)
    if mode == "session-start" {
        registrar := register.New(registry, surfacingLogger, ...)
        _ = registrar.Run(sourceConfig) // fire-and-forget
    }
    // EXISTING: memory surfacing
    ...
}
```

**Source path resolution** at CLI edge:
- Project CLAUDE.md: `filepath.Join(workDir, "CLAUDE.md")`
- Global CLAUDE.md: `filepath.Join(homeDir, ".claude", "CLAUDE.md")`
- MEMORY.md: Claude Code convention path (injected or derived from workDir)
- Rules: `filepath.Join(workDir, ".claude", "rules")`
- Skills: `filepath.Join(pluginRoot, "skills")` (plugin root from env or convention)

The evaluate pipeline (Stop hook) requires no code changes — it already reads the surfacing log and evaluates all entries. Always-loaded sources now appear in the log via ARCH-69's implicit surfacing.

**Traces to:** REQ-117 (evaluate all — achieved by surfacing log inclusion), DES-40 (flow ordering), DES-41 (path resolution)

---

## L2 → ARCH Traceability (UC-26)

| L2 Item | ARCH Coverage |
|---------|--------------|
| REQ-112 | ARCH-69 |
| REQ-113 | ARCH-69 |
| REQ-114 | ARCH-69 |
| REQ-115 | ARCH-69 |
| REQ-116 | ARCH-70 |
| REQ-117 | ARCH-71 (via surfacing log inclusion from ARCH-69) |
| REQ-118 | ARCH-69 |
| DES-40  | ARCH-69, ARCH-71 |
| DES-41  | ARCH-69, ARCH-71 |

All L2 items have ARCH coverage.

---

## ARCH-72: SessionStart symlink step (UC-27)

**Component:** `hooks/session-start.sh`

**What changes:** After the existing binary build block (lines 11-23), add a symlink management block that:

1. Defines target: `SYMLINK_TARGET="$HOME/.local/bin/engram"`
2. `mkdir -p "$HOME/.local/bin"` (create target dir if missing)
3. Check existing target:
   - If symlink pointing to `$ENGRAM_BIN` → skip (idempotent)
   - If symlink pointing elsewhere or regular file → log warning to stderr, skip (no-clobber)
   - If nothing exists → `ln -s "$ENGRAM_BIN" "$SYMLINK_TARGET"`
4. All errors captured with `|| true` (fire-and-forget per ARCH-6)

**Why shell, not Go:** This is a 10-line shell block. Adding a Go subcommand (`engram install-symlink`) would be over-engineering — the symlink logic has no business rules, no DI boundaries, no testable pure functions. It's filesystem plumbing that belongs in the hook script.

**Traces to:** REQ-119 (create), REQ-120 (idempotent), REQ-121 (no-clobber), REQ-122 (fire-and-forget), DES-42 (flow ordering)

---

## L2 → ARCH Traceability (UC-27)

| L2 Item | ARCH Coverage |
|---------|--------------|
| REQ-119 | ARCH-72 |
| REQ-120 | ARCH-72 |
| REQ-121 | ARCH-72 |
| REQ-122 | ARCH-72 |
| DES-42  | ARCH-72 |

All L2 items have ARCH coverage.

---

## ARCH-73: Signal Detector component (UC-28)

**Component:** `internal/signal/detector.go`

**Decision:** A Detector aggregates maintenance and promotion signals using injected interfaces that wrap existing domain logic. No LLM calls.

```go
type Classifier interface {
    Classify(stats map[string]effectiveness.Stat, tracking map[string]review.TrackingData) []review.ClassifiedMemory
}

type PromotionScanner interface {
    Candidates(threshold int) ([]promote.Candidate, error)
}

type ClaudeMDScanner interface {
    PromotionCandidates(threshold int) ([]promote.Candidate, error)
    DemotionCandidates() ([]promote.Candidate, error)
}

type Detector struct {
    classifier  Classifier
    promoter    PromotionScanner
    claudeMD    ClaudeMDScanner
    now         func() time.Time
}

type Signal struct {
    Type       string    `json:"type"`
    SourceID   string    `json:"source_id"`
    SignalKind string    `json:"signal"`
    Quadrant   string    `json:"quadrant,omitempty"`
    Summary    string    `json:"summary"`
    DetectedAt time.Time `json:"detected_at"`
}

func (d *Detector) Detect(ctx context.Context) ([]Signal, error)
```

Detect() calls classifier for maintenance quadrants (Noise→`noise_removal`, Leech→`leech_rewrite`, HiddenGem→`hidden_gem_broadening`), then calls promotion/demotion scanners. Returns all detected signals.

**Traces to:** REQ-123 (maintenance detection), REQ-124 (promotion detection)

---

## ARCH-74: Proposal Queue Store (UC-28)

**Component:** `internal/signal/queue.go`

**Decision:** Follows `creationlog` pattern exactly — JSONL file with atomic writes via temp+rename, DI for all I/O.

```go
type QueueStore struct {
    readFile   func(string) ([]byte, error)
    createTemp func(dir, pattern string) (*os.File, error)
    rename     func(oldpath, newpath string) error
    remove     func(name string) error
    now        func() time.Time
}

func (q *QueueStore) Read(path string) ([]Signal, error)
func (q *QueueStore) Append(signals []Signal, path string) error
func (q *QueueStore) Prune(path string, existsCheck func(string) bool) error
func (q *QueueStore) ClearBySourceID(path, sourceID string) error
```

File: `<data-dir>/proposal-queue.jsonl`. Read skips malformed lines. Prune removes >30d entries, entries for deleted sources, and deduplicates by source_id+type.

**Traces to:** REQ-125 (queue file), REQ-126 (pruning)

---

## ARCH-75: CLI wiring for signal subcommands (UC-28)

**Component:** `internal/cli/cli.go`

**Decision:** Three new subcommands added to the Run dispatcher:

- `signal-detect`: Opens registry, aggregates effectiveness, builds tracking data, calls Detector.Detect, reads existing queue, prunes, appends new signals, writes queue.
- `signal-surface`: Reads queue, loads memory TOML for each signal, enriches with title/content/effectiveness stats, formats model-facing context with action instructions, outputs JSON.
- `apply-proposal`: Parses action/memory/fields flags, dispatches to Applier, returns JSON result.

All follow existing CLI wiring pattern: parse flags → compose dependencies → call domain logic → render output.

**Traces to:** REQ-127 (surfacing), REQ-128 (apply-proposal), DES-46 (subcommands)

---

## ARCH-76: Hook integration for signal detection (UC-28)

**Component:** `hooks/stop.sh`, `hooks/session-start.sh`

**Decision:**
- `hooks/stop.sh`: Append `"$ENGRAM_BIN" signal-detect --data-dir "$ENGRAM_DATA" 2>/dev/null || true` after the audit step. Last phase, fire-and-forget.
- `hooks/session-start.sh`: Call `signal-surface --data-dir "$ENGRAM_DATA" --format json`, merge output into existing `additionalContext`.

**Traces to:** DES-45 (hook ordering), DES-44 (surfacing format)

---

## ARCH-77: Apply-proposal component (UC-28)

**Component:** `internal/signal/apply.go`

**Decision:** An Applier dispatches actions to handlers, each performing: read TOML → modify → write atomically → update registry → clear queue entry.

```go
type Applier struct {
    readFile   func(string) ([]byte, error)
    writeFile  func(string, []byte) error
    removeFile func(string) error
    registry   RegistryUpdater
    queue      QueueClearer
}

type ApplyAction struct {
    Action   string            `json:"action"`
    Memory   string            `json:"memory"`
    Fields   map[string]any    `json:"fields"`
    Keywords []string          `json:"keywords"`
    Level    int               `json:"level"`
}

type ApplyResult struct {
    Success bool   `json:"success"`
    Action  string `json:"action"`
    Memory  string `json:"memory"`
    Error   string `json:"error,omitempty"`
}

func (a *Applier) Apply(ctx context.Context, action ApplyAction) (ApplyResult, error)
```

Action handlers:
- `remove`: removeFile + registry.Remove + queue.ClearBySourceID
- `rewrite`: read TOML → update fields → writeFile atomically → registry update content hash
- `broaden`: read TOML → append keywords → writeFile atomically
- `escalate`: read TOML → set escalation_level → writeFile atomically

**Traces to:** REQ-128 (apply-proposal)

---

## ARCH-78: Promote with external content (UC-28)

**Component:** `internal/promote/promote.go`, `internal/promote/claudemd.go`

**Decision:** Extend `Promoter.Promote()` and `ClaudeMDPromoter.Promote()` to accept optional pre-generated content. When content is provided, skip the `Generator.Generate()` LLM call. When `--yes` flag is set, skip the `Confirmer.Confirm()` call. Registry merge + file write still happen normally.

```go
type PromoteOpts struct {
    Content     string // if non-empty, skip Generator.Generate
    SkipConfirm bool   // if true, skip Confirmer.Confirm
}
```

Added as optional parameter to Promote methods. Existing callers pass zero-value opts (unchanged behavior).

**Traces to:** REQ-129 (promote with content)

---

## L2 → ARCH Traceability (UC-28)

| L2 Item | ARCH Coverage |
|---------|--------------|
| REQ-123 | ARCH-73 |
| REQ-124 | ARCH-73 |
| REQ-125 | ARCH-74 |
| REQ-126 | ARCH-74 |
| REQ-127 | ARCH-75 |
| REQ-128 | ARCH-75, ARCH-77 |
| REQ-129 | ARCH-78 |
| DES-43  | ARCH-74 |
| DES-44  | ARCH-75, ARCH-76 |
| DES-45  | ARCH-76 |
| DES-46  | ARCH-75 |

All L2 items have ARCH coverage.

---

## ARCH-79: CrossRef Extractor Package (UC-23, S3 Phase A-1)

**Decision:** Create `internal/crossref/` package containing bullet extraction and parsing logic for non-memory instruction sources. Move `ClaudeMDExtractor`, `MemoryMDExtractor`, `RuleExtractor`, `SkillExtractor`, and `InstructionExtractor` interface from `internal/registry/extract.go` to `internal/crossref/extract.go`.

**Rationale:** The registry package's responsibility is memory lifecycle tracking. Non-memory source parsing is a separate concern that belongs in crossref — the package that will host the cross-source scanner (P0c/UC-29). Establishing this boundary now prevents the registry from accumulating unrelated extraction logic.

**Package boundary:**
- `internal/crossref/` — parses CLAUDE.md, MEMORY.md, rule, and skill sources into `registry.InstructionEntry` values. Imports `internal/registry` for the entry type. No I/O — callers inject file content as strings.
- `internal/registry/` — handles only memory source type. Registry JSONL store, surfacing, evaluation, merge operations.

**Callers updated:**
- `internal/register/register.go` — imports `internal/crossref`, uses `crossref.ClaudeMDExtractor` etc.
- `internal/cli/cli.go` — `buildExtractor` function uses `crossref.InstructionExtractor` return type and all four `crossref.*Extractor` types.

**Traces to:** UC-23 (registry), UC-26 (first-class non-memory sources via ARCH-69)

---

## ARCH-P1-1: contradict package is pure logic, no I/O (P1)

`internal/contradict` contains only business logic. BM25 scoring uses the existing `internal/bm25` package. The LLM classifier is an injected interface. No file I/O, no HTTP calls, no `os.*` usage.

- Traces to: UC-P1-1, ARCH-7 (DI everywhere)
- Rationale: Keeps detection testable without mocks for I/O. LLM calls isolated behind Classifier interface.

---

## ARCH-P1-2: Surface package uses interface indirection for contradict (P1)

Surface package does NOT import `internal/contradict` directly. It defines a local `ContradictionDetector` interface. This avoids circular imports and keeps surface package independently testable.

- Traces to: UC-P1-1, ARCH-7
- Rationale: Interface-based wiring consistent with existing surface pattern (MemoryRetriever, MemoryTracker, etc.)
