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

**Decision:** Single binary (`engram`) with subcommands, one per write-path operation:

```
engram extract --session <path>      # Stop hook: session-end learning extraction
engram correct --message <text>      # UserPromptSubmit hook: inline correction detection
engram catchup --session <path>      # Stop hook: missed correction catch-up
```

Read-path commands (`engram query`, `engram surface`) are out of scope until L2B reaches ARCH.

**Rationale:** Each hook invocation is a short-lived process — the binary starts, does its work, exits. No daemon, no long-running process. Subcommands map 1:1 to hook entry points, making the hook scripts trivial shell wrappers. Shared infrastructure (DB, reconciler, audit log) is initialized per invocation in `cmd/engram/main.go`.

**Alternatives considered:**
- Long-running daemon with socket API: Lower per-invocation latency (no startup cost) but adds process management complexity, crash recovery, and socket cleanup. Overkill for write path where latency budget is generous. Rejected for now — can revisit if hook latency becomes a problem on the read path.
- Separate binaries per operation: No code sharing, larger install footprint. Rejected.

**Output contract:** Stdout contains system reminder text (for hooks that surface to the agent) or is empty. Stderr is for fatal errors only. Audit log captures all operational detail. Exit 0 always — hook failures must not break Claude Code.

**Traces to:** REQ-1 (Stop hook invocation), REQ-6 (Go binary), REQ-13 (correction detection), DES-6 (extraction flow), DES-8 (catch-up flow)

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

**Decision:** All I/O through injected interfaces. Library code in `internal/` never imports `os`, `database/sql`, `net/http`, or any I/O package directly.

**Core interfaces:**

```go
// Storage
type MemoryStore interface {
    Get(ctx context.Context, id string) (*Memory, error)
    Create(ctx context.Context, m *Memory) error
    Update(ctx context.Context, m *Memory) error
    FindSimilar(ctx context.Context, query string, k int) ([]ScoredMemory, error)
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

**Wiring:** `cmd/engram/main.go` constructs real implementations:

```go
func main() {
    db := sqlite.Open(dataDir + "/engram.db")
    store := sqlite.NewMemoryStore(db)
    audit := file.NewAuditLog(dataDir + "/audit.log")
    corpus := file.NewPatternCorpus(dataDir + "/patterns.json")
    sessionLog := file.NewSessionLog(dataDir + "/session.log")
    llm := anthropic.NewClient(apiKey)
    clock := realclock.New()

    // subcommand dispatch...
}
```

`internal/` packages receive interfaces only. Tests use in-memory fakes.

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

## Bidirectional Traceability

### ARCH → L2 (every ARCH traces to at least one L2 item)

| ARCH | L2 items |
|------|----------|
| ARCH-1 | REQ-2, REQ-3, REQ-5, REQ-6 |
| ARCH-2 | REQ-1, REQ-6, REQ-13, DES-6, DES-8 |
| ARCH-3 | REQ-1, REQ-2, REQ-3, REQ-5, REQ-18, REQ-22, DES-6 |
| ARCH-4 | REQ-13, REQ-14, REQ-22, DES-3, DES-5 |
| ARCH-5 | REQ-5, REQ-14 |
| ARCH-6 | REQ-15, DES-8 |
| ARCH-7 | REQ-22, DES-7 |
| ARCH-8 | REQ-6 |
| ARCH-9 | REQ-1, REQ-13, DES-6, DES-8 |

### L2 → ARCH (every L2A item covered by at least one ARCH)

| L2 item | ARCH coverage |
|---------|--------------|
| REQ-1 | ARCH-2, ARCH-3, ARCH-9 |
| REQ-2 | ARCH-1, ARCH-3 |
| REQ-3 | ARCH-1, ARCH-3 |
| REQ-5 | ARCH-1, ARCH-3, ARCH-5 |
| REQ-6 | ARCH-1, ARCH-2, ARCH-8 |
| REQ-13 | ARCH-2, ARCH-4, ARCH-9 |
| REQ-14 | ARCH-4, ARCH-5 |
| REQ-15 | ARCH-6 |
| REQ-18 | ARCH-3 |
| REQ-22 | ARCH-3, ARCH-4, ARCH-7 |
| DES-3 | ARCH-4 |
| DES-5 | ARCH-4 |
| DES-6 | ARCH-2, ARCH-3, ARCH-9 |
| DES-7 | ARCH-7 |
| DES-8 | ARCH-2, ARCH-6, ARCH-9 |

All L2A items have ARCH coverage. No orphaned ARCH decisions.
