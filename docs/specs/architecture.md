# Architecture

System architecture for the engram memory system. Each ARCH-N decision traces to L2 items (REQ and/or DES).

---

## ARCH-1: Memory Storage Model

**Decision:** SQLite with FTS5 for memory storage. Single database file at `<plugin-data-dir>/engram.db`. Pure Go driver (`modernc.org/sqlite`, no CGO).

**Rationale:** Single file, no server, zero-config. FTS5 provides BM25 ranking for free, covering local similarity needs without a separate TF-IDF index. Pure Go driver satisfies the no-CGO constraint. SQLite's WAL mode supports concurrent reads from hook invocations.

**Alternatives considered:**
- File-per-memory (JSON/TOML): Simple but no query capability, no FTS, O(n) similarity search. Rejected.
- Separate TF-IDF index + flat file store: More moving parts, index rebuild complexity. FTS5 subsumes this. Rejected.
- bbolt/badger (embedded KV): No built-in full-text search. Would need custom TF-IDF on top. More code for less capability. Rejected.

**Schema:**

```sql
CREATE TABLE memories (
    id TEXT PRIMARY KEY,              -- m_<8hex> (SHA-256 truncated)
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    observation_type TEXT,
    concepts TEXT,                     -- JSON array
    principle TEXT,
    anti_pattern TEXT,
    rationale TEXT,
    enriched_content TEXT,
    keywords TEXT,                     -- JSON array (LLM-generated)
    confidence TEXT NOT NULL,          -- A, B, C
    enrichment_count INTEGER DEFAULT 0,
    impact_score REAL DEFAULT 0.5,    -- neutral baseline for cold start
    created_at TEXT NOT NULL,          -- RFC 3339
    updated_at TEXT NOT NULL,
    last_surfaced_at TEXT,
    surfacing_count INTEGER DEFAULT 0
);

CREATE VIRTUAL TABLE memories_fts USING fts5(
    title, content, keywords, enriched_content,
    content='memories', content_rowid='rowid'
);
```

FTS5 `rank` function provides BM25 scores. Combined with keyword overlap scoring on the `keywords` JSON array, this covers the local similarity retrieval needed by reconciliation (REQ-5, REQ-14) and hook-time surfacing (future L2B scope).

**Traces to:** REQ-2 (structured metadata), REQ-3 (confidence field), REQ-5 (candidate retrieval via FTS5), REQ-6 (Go binary with no CGO)

---

## ARCH-2: Go Binary Command Structure

**Decision:** Single binary (`engram`) with subcommands:

```
# Write path (L2A/L3A)
engram extract --session <path>      # Stop hook: session-end learning extraction
engram correct --message <text>      # UserPromptSubmit hook: inline correction detection
engram catchup --session <path>      # Stop hook: missed correction catch-up

# Read path (L2B/L3B)
engram surface --hook <type> --query <text>  # All hooks: memory surfacing (ARCH-11)
```

**Rationale:** Each hook invocation is a short-lived process — the binary starts, does its work, exits. No daemon, no long-running process. Subcommands map 1:1 to hook entry points, making the hook scripts trivial shell wrappers. Shared infrastructure (DB, reconciler, audit log) is initialized per invocation in `cmd/engram/main.go`.

**Alternatives considered:**
- Long-running daemon with socket API: Lower per-invocation latency (no startup cost) but adds process management complexity, crash recovery, and socket cleanup. Overkill for write path where latency budget is generous. Rejected for now — can revisit if hook latency becomes a problem on the read path.
- Separate binaries per operation: No code sharing, larger install footprint. Rejected.

**Output contract:** Stdout contains system reminder text (for hooks that surface to the agent) or is empty. Stderr is for fatal errors only. Audit log captures all operational detail. Exit 0 always — hook failures must not break Claude Code.

**Traces to:** REQ-1 (Stop hook invocation), REQ-6 (Go binary), REQ-13 (correction detection), REQ-7/8/9 (surface subcommand), DES-6 (extraction flow), DES-8 (catch-up flow)

---

## ARCH-3: Extraction Pipeline Architecture

**Decision:** Pipeline of injected stages, each independently testable:

```go
type Extractor struct {
    Enricher    Enricher       // sonnet: transcript → []RawLearning (with metadata + keywords)
    QualityGate QualityGate    // filter: []RawLearning → []RawLearning
    Classifier  Classifier     // haiku: RawLearning → confidence tier (A/B/C)
    Reconciler  Reconciler     // local similarity + haiku: learning × store → create|enrich
    Store       MemoryStore    // DB read/write
    SessionLog  SessionLog     // mid-session correction IDs for dedup pre-filter
    AuditLog    AuditLog       // structured log writer
}

func (e *Extractor) Run(ctx context.Context, transcript []byte) error {
    // 1. Enricher: extract learnings with metadata + keywords
    // 2. QualityGate: reject vague/mechanical patterns
    // 3. For each surviving learning:
    //    a. SessionLog dedup pre-filter (REQ-18): skip if already captured mid-session
    //    b. Classifier: assign confidence tier
    //    c. Reconciler: candidate retrieval + overlap gate → create or enrich
    // 4. AuditLog: record all actions (created, enriched, rejected, skipped)
}
```

**Stage responsibilities:**

| Stage | Model tier | Input | Output |
|-------|-----------|-------|--------|
| Enricher | sonnet | transcript bytes | []RawLearning (6 metadata fields + keywords) |
| QualityGate | deterministic (initially) | []RawLearning | []RawLearning (filtered) |
| Classifier | haiku | RawLearning + transcript context | confidence tier (A/B/C) |
| Reconciler | local similarity + haiku | learning + store | ReconcileResult (created/enriched) |

**QualityGate** starts deterministic: reject if content < 10 tokens, reject if no concrete nouns/verbs (pure filler). Can upgrade to haiku if deterministic gate proves too coarse — but start cheap.

**Dedup pre-filter** runs before reconciliation: check if the learning's content matches any memory ID in the session's correction log. If yes, skip entirely — don't even call the reconciler. This is the REQ-18 optimization that avoids redundant haiku calls.

**Alternatives considered:**
- Single monolithic function: Untestable, violates DI principles. Rejected.
- Event-driven/channel pipeline: Over-engineered for sequential processing of typically 1-10 learnings per session. Rejected.

**Traces to:** REQ-1 (trigger), REQ-2 (metadata/keywords via Enricher), REQ-3 (confidence via Classifier), REQ-5 (reconciliation), REQ-18 (dedup pre-filter), REQ-22 (audit), DES-6 (extraction flow)

---

## ARCH-4: Correction Detection Architecture

**Decision:** Pattern corpus matching with reconciliation on match:

```go
type CorrectionDetector struct {
    Corpus     PatternCorpus      // persisted regex patterns, loaded per invocation
    Reconciler Reconciler         // shared with extraction (ARCH-5)
    Store      MemoryStore        // shared
    AuditLog   AuditLog           // shared
    Formatter  ReminderFormatter  // system reminder text builder
}

func (d *CorrectionDetector) Detect(ctx context.Context, message string) (string, error) {
    // 1. Corpus.Match(message): check all patterns, return first match (or none)
    // 2. If no match: return "" (empty stdout, no system reminder)
    // 3. If match:
    //    a. Build Learning from message + matched pattern context
    //    b. Reconciler.Reconcile: candidate retrieval + overlap gate → create or enrich
    //    c. SessionLog.Record: log this correction for dedup at session-end
    //    d. Formatter.CorrectionCaptured: build system reminder text (DES-3 format)
    //    e. AuditLog.Log: record the correction event
    //    f. Return system reminder text
}
```

**PatternCorpus** is a JSON file at `<plugin-data-dir>/patterns.json`:

```json
{
    "patterns": [
        {"regex": "^no,", "label": "direct-negation"},
        {"regex": "^wait", "label": "interruption"},
        ...
    ]
}
```

Loaded fresh each invocation (the binary is short-lived). New patterns are appended by the catch-up processor (ARCH-6). The file ships with the 15 initial patterns from REQ-13.

**Alternatives considered:**
- LLM-based correction detection at hook time: Adds latency and cost to every UserPromptSubmit. Violates model hierarchy (deterministic first). Rejected.
- Compiled regex set (Aho-Corasick): Premature optimization — 15-50 regexes are fast enough with sequential matching. Can optimize later if corpus grows large. Rejected for now.

**Traces to:** REQ-13 (pattern matching), REQ-14 (reconciliation on match), REQ-22 (audit), DES-3 (feedback format), DES-5 (false positive handling — capture and decay)

---

## ARCH-5: Reconciliation as Shared Component

**Decision:** Single reconciler implementation used by both extraction (ARCH-3) and correction detection (ARCH-4):

```go
type Reconciler interface {
    Reconcile(ctx context.Context, learning Learning) (ReconcileResult, error)
}

type reconciler struct {
    Store       MemoryStore    // FindSimilar for candidate retrieval
    OverlapGate OverlapGate    // haiku: candidate × learning → overlap decision
    K           int            // candidate count (default 3, user-configurable)
}

type ReconcileResult struct {
    Action   string   // "created" or "enriched"
    MemoryID string
    Title    string
    Keywords []string
    Overlap  float64  // similarity score of best match (0.0 if created)
}
```

**`MemoryStore.FindSimilar(query string, k int)`** uses FTS5 BM25 ranking to retrieve the top-K most similar existing memories. The query is the learning's content + keywords concatenated.

**`OverlapGate`** is a haiku LLM call per candidate: "Is this learning saying the same thing as this existing memory?" Returns yes/no + rationale. If yes for any candidate, enrich the best-scoring one. If no for all candidates, create new.

**K is a budget, not a threshold.** If K=3, haiku evaluates 3 candidates. If the best candidate has low FTS5 score, haiku still evaluates it — and will likely reject it. No similarity floor. Self-correction: bad merges produce memories that get surfaced but not followed, causing frecency decay.

**Alternatives considered:**
- Separate reconcilers for extraction vs correction: Code duplication, same logic. Rejected.
- Skip haiku, use similarity threshold only: FTS5 BM25 scores aren't calibrated for semantic overlap. Two memories about "git staging" might score high but be about different aspects. Haiku judgment is worth the cost (corrections are infrequent, extraction is background). Rejected.

**Traces to:** REQ-5 (extraction reconciliation), REQ-14 (correction reconciliation)

---

## ARCH-6: Session-End Catch-Up Architecture

**Decision:** Separate processor for finding corrections the inline detector missed:

```go
type CatchupProcessor struct {
    Evaluator  CatchupEvaluator  // haiku: transcript × correction_log → []MissedCorrection
    Reconciler Reconciler         // shared (ARCH-5)
    Corpus     PatternCorpus      // for appending new patterns
    SessionLog SessionLog         // mid-session corrections to compare against
    Store      MemoryStore
    AuditLog   AuditLog
}

func (p *CatchupProcessor) Run(ctx context.Context, transcript []byte) error {
    // 1. SessionLog.List(): get all mid-session corrections
    // 2. Evaluator: send transcript + correction list to haiku
    //    "Here are the corrections already captured. Are there other user
    //     corrections in this transcript that were missed?"
    // 3. For each missed correction:
    //    a. Reconciler.Reconcile: create or enrich memory
    //    b. Extract correction phrase for corpus candidate
    // 4. Corpus.AddCandidates: append new pattern candidates
    // 5. AuditLog: record findings
}
```

**CatchupEvaluator** receives the full transcript and the list of already-captured corrections. This context allows haiku to identify only genuinely missed corrections — not re-flag things already handled.

**Pattern corpus candidates** are not immediately active. New patterns extracted from missed corrections are added as candidates and validated by future occurrence. Implementation: patterns.json gains a `"candidate": true` field. Candidates are promoted to active after matching in N future sessions (N=2 default). This prevents one-off phrasings from polluting the corpus.

**Alternatives considered:**
- Run catch-up inline (at every UserPromptSubmit): Too expensive — haiku call on every message. Rejected.
- Skip catch-up entirely, rely on session-end extraction: Extraction (ARCH-3) catches learnings but doesn't specifically look for correction patterns. Catch-up specifically targets the gap between inline detection and full transcript analysis. Keep both. Rejected.

**Traces to:** REQ-15 (session-end catch-up), DES-8 (catch-up flow)

---

## ARCH-7: Audit Log Implementation

**Decision:** Structured key-value log, append-only file:

```go
type AuditLog interface {
    Log(entry AuditEntry) error
}

type AuditEntry struct {
    Timestamp time.Time
    Operation string            // extract, correct, catchup, surface, reclass
    Action    string            // created, enriched, rejected, skipped, returned, decreased
    Fields    map[string]string // operation-specific k/v pairs
}
```

**Format:** One line per entry, key=value pairs, as specified in DES-7:

```
2026-02-27T16:30:00Z extract created memory_id=m_7f3a title="Use targ build system" confidence=B quality_score=0.85
```

**Storage:** `<plugin-data-dir>/audit.log`. Append-only. Implementation is a file writer with a mutex (binary is short-lived, but extract + catchup run sequentially in the same Stop hook invocation).

**Rotation:** Not implemented at launch. The log grows ~1KB per session (typical: 5-10 entries). At 1000 sessions that's ~1MB — manageable. Revisit if growth becomes a problem.

**Alternatives considered:**
- SQLite audit table: Queryable but heavier. Audit data is primarily for human debugging, not programmatic queries. A grep-able log file is simpler. Rejected.
- JSON lines: More structured but harder to scan visually. Key-value is more readable for debugging. Rejected.

**Traces to:** REQ-22 (audit logging), DES-7 (format specification)

---

## ARCH-8: DI Interfaces and Wiring

**Decision:** All I/O through injected interfaces. Library code in `internal/` never imports `os`, `database/sql`, `net/http`, or any I/O package directly. **Exception:** `internal/cli/` is the composition root (wiring layer), not library code — it imports I/O packages to construct real implementations and dispatch subcommands.

**Core interfaces:**

```go
// Storage
type MemoryStore interface {
    Get(ctx context.Context, id string) (*Memory, error)
    Create(ctx context.Context, m *Memory) error
    Update(ctx context.Context, m *Memory) error
    FindSimilar(ctx context.Context, query string, k int) ([]ScoredMemory, error)
    Surface(ctx context.Context, query string, k int) ([]ScoredMemory, error)           // ARCH-11: frecency-ranked retrieval
    IncrementSurfacing(ctx context.Context, ids []string) error                          // ARCH-11: update surfacing metadata
}

// Audit
type AuditLog interface {
    Log(entry AuditEntry) error
}

// Correction patterns
type PatternCorpus interface {
    Match(message string) (*PatternMatch, error)        // check message against all active patterns
    AddCandidates(patterns []CandidatePattern) error     // append new candidates from catch-up
}

// Session tracking (mid-session corrections for dedup)
type SessionLog interface {
    Record(correction CorrectionEvent) error
    List() ([]CorrectionEvent, error)
    HasOverlap(memoryID string) bool  // dedup check
}

// LLM calls
type LLMClient interface {
    Enrich(ctx context.Context, transcript []byte) ([]RawLearning, error)
    Classify(ctx context.Context, learning RawLearning, transcript []byte) (string, error)
    OverlapGate(ctx context.Context, learning Learning, candidate Memory) (bool, string, error)
    FindMissedCorrections(ctx context.Context, transcript []byte, captured []CorrectionEvent) ([]MissedCorrection, error)
}

// Time
type Clock interface {
    Now() time.Time
}
```

**Wiring:** `internal/cli/cli.go` is the composition root. `cmd/engram/main.go` is a thin entry point that delegates to `cli.Run(os.Args)`:

```go
// cmd/engram/main.go — thin entry point
func main() {
    err := cli.Run(os.Args)
    if err != nil { fmt.Fprintln(os.Stderr, err) }
}

// internal/cli/cli.go — composition root
// Opens DB, audit log, pattern corpus per subcommand invocation.
// Constructs real implementations and wires them together.
```

`internal/` packages (except `internal/cli/`) receive interfaces only. Tests use in-memory fakes.

**Alternatives considered:**
- Fewer interfaces (combine Store + AuditLog + SessionLog): Different responsibilities, different backing stores. Keep separate for testability. Rejected.
- Global singletons: Violates DI. The single most pervasive failure in the old system. Rejected.

**Traces to:** REQ-6 (Go binary, pure Go), CLAUDE.md DI principles

---

## ARCH-9: Hook Shell Scripts

**Decision:** Thin shell wrappers that invoke the Go binary. All logic in the binary, not in scripts.

**Stop hook** (`hooks/stop.sh`):
```bash
#!/usr/bin/env bash
set -euo pipefail

ENGRAM_BIN="${CLAUDE_PLUGIN_ROOT}/bin/engram"
ENGRAM_DATA="${CLAUDE_PLUGIN_ROOT}/data"

# UC-1: Extract learnings from session
"$ENGRAM_BIN" extract --session "$CLAUDE_SESSION_TRANSCRIPT" --data-dir "$ENGRAM_DATA"

# UC-3: Catch up missed corrections
"$ENGRAM_BIN" catchup --session "$CLAUDE_SESSION_TRANSCRIPT" --data-dir "$ENGRAM_DATA"
```

**UserPromptSubmit hook** (`hooks/user-prompt-submit.sh`):
```bash
#!/usr/bin/env bash
set -euo pipefail

ENGRAM_BIN="${CLAUDE_PLUGIN_ROOT}/bin/engram"
ENGRAM_DATA="${CLAUDE_PLUGIN_ROOT}/data"

# UC-3: Check for inline correction
"$ENGRAM_BIN" correct --message "$CLAUDE_USER_MESSAGE" --data-dir "$ENGRAM_DATA"
```

**Environment variables:** Scripts use `CLAUDE_PLUGIN_ROOT` for binary and data paths. Hook-specific variables (`CLAUDE_SESSION_TRANSCRIPT`, `CLAUDE_USER_MESSAGE`) are provided by Claude Code.

**Open question:** The exact environment variables Claude Code passes to hooks need verification against the hook API. `CLAUDE_SESSION_TRANSCRIPT` and `CLAUDE_USER_MESSAGE` are assumed names — the actual names may differ. This will be resolved at implementation time (L5).

**Error handling:** The binary handles errors internally (logs to audit, returns empty stdout). The shell scripts use `set -euo pipefail` as a safety net but the binary should never exit non-zero in normal operation.

**Traces to:** REQ-1 (Stop hook trigger), REQ-13 (UserPromptSubmit hook trigger), DES-6 (extraction flow), DES-8 (catch-up flow)

---

## ARCH-10: Frecency Ranking Algorithm

**Decision:** Frecency is the harmonic mean of recency and impact, computed in SQL at query time:

```
frecency = 2.0 * recency * impact / (recency + impact)
```

**Recency signal:** Exponential decay from most recent activity:

```
recency = 1.0 / (1.0 + days_since_last_activity)
last_activity = MAX(updated_at, last_surfaced_at, created_at)
```

When `last_surfaced_at` is NULL (never surfaced), falls back to `updated_at` or `created_at`. A memory updated today has recency ≈ 1.0; a memory last touched 30 days ago has recency ≈ 0.032.

**Impact signal:** Stored as `impact_score` in the memories table (ARCH-1 schema). Defaults to 0.5 (neutral baseline). Updated by the evaluation pipeline (out of scope until L2C/L1B). During cold start, all memories share the same impact (0.5), so the harmonic mean simplifies to a function of recency alone — satisfying REQ-4 AC(2).

**Confidence tiebreaker:** When frecency scores are equal, ORDER BY confidence: A=3 > B=2 > C=1.

**SQL expression for surfacing queries:**

```sql
SELECT m.*,
    2.0 * (1.0 / (1.0 + julianday('now') - julianday(
        COALESCE(m.last_surfaced_at, m.updated_at, m.created_at)
    ))) * m.impact_score / (
        (1.0 / (1.0 + julianday('now') - julianday(
            COALESCE(m.last_surfaced_at, m.updated_at, m.created_at)
        ))) + m.impact_score
    ) AS frecency
FROM memories m
JOIN memories_fts ON memories_fts.rowid = m.rowid
WHERE memories_fts MATCH ?
ORDER BY frecency DESC,
    CASE m.confidence WHEN 'A' THEN 3 WHEN 'B' THEN 2 ELSE 1 END DESC
LIMIT ?
```

FTS5 MATCH acts as a relevance filter (only text-relevant memories), while ORDER BY does frecency ranking. This differs from `FindSimilar` (ARCH-5) which uses `ORDER BY rank` (pure BM25).

**Alternatives considered:**
- Pure BM25 ranking for surfacing: Ignores recency and impact. A high-quality memory from 3 months ago that's never followed would rank alongside a fresh, proven memory. Rejected.
- Weighted linear combination instead of harmonic mean: Harmonic mean naturally penalizes when either signal is low — a very recent but zero-impact memory doesn't dominate. Better behavior for our use case. Rejected linear.

**Traces to:** REQ-4 (ranking formula), REQ-10 (no LLM — pure SQL computation)

---

## ARCH-11: Surfacing Pipeline Architecture

**Decision:** `engram surface` subcommand with per-hook behavior, extending ARCH-2's command structure:

```
engram surface --hook <session-start|user-prompt|pre-tool-use> --query <text> --data-dir <path> [--budget K]
```

**Pipeline stages:**

```go
type SurfacePipeline struct {
    Store     MemoryStore     // Surface method with frecency ranking
    Formatter ReminderFormatter
    AuditLog  AuditLog
}

func (p *SurfacePipeline) Run(ctx context.Context, hookType string, query string, budget int) (string, error) {
    // 1. Store.Surface(ctx, query, budget): FTS5 MATCH + frecency ORDER BY + LIMIT
    // 2. If no results: return "" (empty stdout, no system reminder)
    // 3. Formatter.FormatSurfacing(memories, hookType): build system reminder text
    //    - session-start, user-prompt: full format (DES-1 numbered list)
    //    - pre-tool-use: compact single-line format (DES-1 variant)
    // 4. AuditLog.Log: record surfacing event (hook, count, query_tokens, latency_ms)
    // 5. Update surfacing metadata: increment surfacing_count, set last_surfaced_at
    // 6. Return system reminder text to stdout
}
```

**New store method:** `Surface(ctx, query, k)` distinct from `FindSimilar`:
- `FindSimilar` (ARCH-5): FTS5 BM25 ranking, for reconciliation candidate retrieval
- `Surface`: FTS5 match + frecency ranking (ARCH-10), for hook-time surfacing

```go
type MemoryStore interface {
    // existing methods from ARCH-8...
    Surface(ctx context.Context, query string, k int) ([]ScoredMemory, error)
    IncrementSurfacing(ctx context.Context, ids []string) error
}
```

**ReminderFormatter** builds DES-1 format:

```go
type ReminderFormatter interface {
    FormatSurfacing(memories []ScoredMemory, hookType string) string
}
```

- `session-start` / `user-prompt`: Full format with `<system-reminder source="engram">`, `[engram] N memories for this context:`, numbered entries with title, confidence, impact, body.
- `pre-tool-use`: Compact single-line format: `[engram] <title> (<confidence>, <impact>)`
- Returns empty string for zero results.

**Default budgets per hook type (user-configurable via --budget flag):**

| Hook type | Default K | Rationale |
|-----------|----------|-----------|
| session-start | 5 | Broad context, most room |
| user-prompt | 3 | Task-scoped |
| pre-tool-use | 1 | Latency-critical |

**Surfacing metadata update:** After formatting results, the pipeline updates `surfacing_count` and `last_surfaced_at` for each surfaced memory. This feeds the frecency algorithm — memories that are surfaced frequently have higher recency, and the feedback loop with evaluation data (future L1B scope) will adjust their impact.

**Alternatives considered:**
- Reuse `FindSimilar` and re-rank in Go code: Extra allocation and sorting for something SQL handles natively. FTS5 MATCH filtering + SQL ORDER BY is a single query. Rejected.
- Separate binary for surfacing: No code sharing with existing infrastructure (store, audit). The single-binary model (ARCH-2) already handles multiple subcommands. Rejected.

**Traces to:** REQ-7 (SessionStart), REQ-8 (UserPromptSubmit), REQ-9 (PreToolUse), REQ-10 (no LLM), REQ-12 (system reminders), DES-1 (format), DES-2 (scenarios)

---

## ARCH-12: Read-Path Hook Integration

**Decision:** Extend existing hook scripts (ARCH-9) with surfacing invocations. Add new hooks for SessionStart and PreToolUse.

**SessionStart hook** (`hooks/session-start.sh`):
```bash
#!/usr/bin/env bash
set -euo pipefail

ENGRAM_BIN="${CLAUDE_PLUGIN_ROOT}/bin/engram"
ENGRAM_DATA="${CLAUDE_PLUGIN_ROOT}/data"

# Build project context query from available sources
QUERY=""
[ -f "${CLAUDE_PROJECT_DIR:-}/CLAUDE.md" ] && QUERY="$(cat "$CLAUDE_PROJECT_DIR/CLAUDE.md") "
[ -f "${CLAUDE_PROJECT_DIR:-}/README.md" ] && QUERY="$QUERY$(cat "$CLAUDE_PROJECT_DIR/README.md") "
QUERY="$QUERY${CLAUDE_PROJECT_DIR##*/}"

# UC-2: Surface project-relevant memories
"$ENGRAM_BIN" surface --hook session-start --query "$QUERY" --data-dir "$ENGRAM_DATA"
```

**UserPromptSubmit hook** (extended from ARCH-9):
```bash
#!/usr/bin/env bash
set -euo pipefail

ENGRAM_BIN="${CLAUDE_PLUGIN_ROOT}/bin/engram"
ENGRAM_DATA="${CLAUDE_PLUGIN_ROOT}/data"

# UC-3: Check for inline correction (DES-4: correction first)
"$ENGRAM_BIN" correct --message "$CLAUDE_USER_MESSAGE" --data-dir "$ENGRAM_DATA"

# UC-2: Surface relevant memories (DES-4: surfacing second)
"$ENGRAM_BIN" surface --hook user-prompt --query "$CLAUDE_USER_MESSAGE" --data-dir "$ENGRAM_DATA"
```

Stdout from both commands concatenates naturally — correction feedback appears first, then surfaced memories, implementing DES-4's ordering without special logic.

**PreToolUse hook** (`hooks/pre-tool-use.sh`):
```bash
#!/usr/bin/env bash
set -euo pipefail

ENGRAM_BIN="${CLAUDE_PLUGIN_ROOT}/bin/engram"
ENGRAM_DATA="${CLAUDE_PLUGIN_ROOT}/data"

# UC-2: Surface most relevant memory (compact format, budget=1)
"$ENGRAM_BIN" surface --hook pre-tool-use --query "$CLAUDE_TOOL_INPUT" --data-dir "$ENGRAM_DATA"
```

**Environment variables:** `CLAUDE_PROJECT_DIR`, `CLAUDE_USER_MESSAGE`, `CLAUDE_TOOL_INPUT` are assumed names — exact variable names need verification against the Claude Code hook API (same caveat as ARCH-9).

**Dual-duty UserPromptSubmit:** The hook runs two separate binary invocations. This is intentional — each invocation is independent (correction detection doesn't need surfacing results, and vice versa). Two short-lived processes are simpler than a combined mode, and the latency overhead of a second process start (~10-20ms) is acceptable for UserPromptSubmit (not the latency-critical path).

**Alternatives considered:**
- Combined `engram prompt --message <text>` that does both correction + surfacing: More complex, couples two independent operations, harder to test. The simple concatenation of two invocations gets the same result. Rejected.
- Surfacing via stdin instead of --query: Less transparent for debugging. Explicit flags are easier to trace. Rejected.

**Traces to:** REQ-7 (SessionStart hook), REQ-8 (UserPromptSubmit hook), REQ-9 (PreToolUse hook), DES-2 (per-hook scenarios), ARCH-9 (existing hook scripts)

---

## Bidirectional Traceability

### ARCH → L2 (every ARCH traces to at least one L2 item)

| ARCH | L2 items |
|------|----------|
| ARCH-1 | REQ-2, REQ-3, REQ-5, REQ-6 |
| ARCH-2 | REQ-1, REQ-6, REQ-7, REQ-8, REQ-9, REQ-13, DES-6, DES-8 |
| ARCH-3 | REQ-1, REQ-2, REQ-3, REQ-5, REQ-18, REQ-22, DES-6 |
| ARCH-4 | REQ-13, REQ-14, REQ-17, REQ-22, DES-3, DES-5 |
| ARCH-5 | REQ-5, REQ-14 |
| ARCH-6 | REQ-15, DES-8 |
| ARCH-7 | REQ-22, DES-7 |
| ARCH-8 | REQ-6 |
| ARCH-9 | REQ-1, REQ-13, DES-6, DES-8 |
| ARCH-10 | REQ-4, REQ-10 |
| ARCH-11 | REQ-7, REQ-8, REQ-9, REQ-10, REQ-12, DES-1, DES-2 |
| ARCH-12 | REQ-7, REQ-8, REQ-9, DES-2, DES-4 |

### L2 → ARCH (every L2 item covered by at least one ARCH)

| L2 item | ARCH coverage |
|---------|--------------|
| REQ-1 | ARCH-2, ARCH-3, ARCH-9 |
| REQ-2 | ARCH-1, ARCH-3 |
| REQ-3 | ARCH-1, ARCH-3 |
| REQ-4 | ARCH-10 |
| REQ-5 | ARCH-1, ARCH-3, ARCH-5 |
| REQ-6 | ARCH-1, ARCH-2, ARCH-8 |
| REQ-7 | ARCH-2, ARCH-11, ARCH-12 |
| REQ-8 | ARCH-2, ARCH-11, ARCH-12 |
| REQ-9 | ARCH-2, ARCH-11, ARCH-12 |
| REQ-10 | ARCH-10, ARCH-11 |
| REQ-12 | ARCH-11 |
| REQ-13 | ARCH-2, ARCH-4, ARCH-9 |
| REQ-14 | ARCH-4, ARCH-5 |
| REQ-15 | ARCH-6 |
| REQ-18 | ARCH-3 |
| REQ-22 | ARCH-3, ARCH-4, ARCH-7 |
| DES-1 | ARCH-11 |
| DES-2 | ARCH-11, ARCH-12 |
| REQ-17 | ARCH-4 |
| DES-3 | ARCH-4 |
| DES-4 | ARCH-12 |
| DES-5 | ARCH-4 |
| DES-6 | ARCH-2, ARCH-3, ARCH-9 |
| DES-7 | ARCH-7 |
| DES-8 | ARCH-2, ARCH-6, ARCH-9 |

All L2A and L2B items have ARCH coverage (including REQ-17 and DES-4, absorbed from L2C). No orphaned ARCH decisions. L2C (REQ-16) and L2D (REQ-19..21) are pending — no ARCH coverage yet.
