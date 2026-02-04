# Layer 0: Foundation - Architecture

**Project:** layer-0-foundation
**Phase:** Architecture
**Created:** 2026-02-04

**Traces to:** ISSUE-045

---

## Overview

This document defines the technical architecture for completing Layer 0 foundation components in projctl. The architecture builds on existing memory, context, state, and trace infrastructure while adding memory extraction capabilities, centralized yield path generation, and comprehensive testing strategies.

---

## Architecture Decisions

### ARCH-001: Memory Extract Command Implementation

Extract insights from yield/result TOML files and store in semantic memory using existing ONNX infrastructure.

**Command Integration:**
- New subcommand: `projctl memory extract`
- Reuses existing `internal/memory` package infrastructure
- Leverages existing ONNX runtime, e5-small model, and SQLite-vec storage

**Implementation Approach:**
```
cmd/projctl/memory_extract.go
└─> internal/memory.Extract(opts ExtractOpts) (*ExtractResult, error)
    ├─> Parse TOML file (BurntSushi/toml)
    ├─> Validate schema (fail fast on invalid structure)
    ├─> Extract decisions from [payload.decisions] arrays
    ├─> Extract learnings from [payload] fields (summary, findings)
    ├─> Extract context (phase, subphase, status)
    ├─> Generate embeddings via generateEmbeddingONNX()
    └─> Store in SQLite-vec via createEmbeddings()
```

**TOML Schema Validation:**
- Use `toml.Unmarshal()` with struct tags for strict validation
- Define `YieldFile` and `ResultFile` structs matching yield protocol
- Return detailed error messages with field names on validation failure
- Fail fast: any schema violation returns error immediately

**Output Format:**
- Terminal: Human-readable summary (see DES-002)
- Structured: TOML with extract status, counts, items (see DES-001)
- Both outputs generated from same `ExtractResult` struct

**Error Handling:**
- File not found → wrap with context, return immediately
- TOML parse error → include line number, return immediately
- Schema validation error → include expected field, return immediately
- Embedding generation error → wrap with context, return immediately
- Database error → wrap with context, return immediately

**Rationale:** Reuse existing memory infrastructure (ONNX, SQLite-vec, embeddings) rather than creating new storage mechanism. Strict validation ensures data quality.

**Alternatives Considered:**
- Best-effort extraction (rejected: could silently miss required fields)
- Custom TOML parser (rejected: BurntSushi/toml is standard)
- Separate storage for extracted content (rejected: unified memory system is simpler)

**Traces to:** REQ-1, DES-001, DES-002, DES-009

---

### ARCH-002: Yield Path Generation Strategy

Centralized path generation function ensures uniqueness for parallel execution using timestamp + UUID pattern.

**Function Signature:**
```go
package context

func GenerateYieldPath(projectDir, phase, taskID string) (string, error)
```

**Path Generation Algorithm:**
```
1. Get or create project UUID (stored in state.toml)
2. Generate file-level UUID (uuid.New())
3. Format creation date: YYYY-MM-DD
4. Format current datetime: YYYY-MM-DD.HH-mm-SS
5. If taskID provided:
   path = {projectDir}/.claude/context/{date}-{project}-{projectUUID}/{datetime}-{phase}-{taskID}-{fileUUID}.toml
6. Else (sequential):
   path = {projectDir}/.claude/context/{date}-{project}-{projectUUID}/{datetime}-{phase}-{fileUUID}.toml
7. Convert to absolute path via filepath.Abs()
8. Create parent directories via os.MkdirAll() (mode 0755)
9. Return absolute path
```

**UUID Source:**
- Use `github.com/google/uuid` package (already in go.mod)
- Project UUID: stable across invocations, stored in state
- File UUID: unique per invocation, never stored

**Directory Structure:**
```
.claude/context/
└── 2026-02-04-layer-0-abc123/        # Project-level directory
    ├── 2026-02-04.12-45-30-pm-def456.toml           # Sequential
    ├── 2026-02-04.12-46-15-design-xyz789.toml        # Sequential with phase
    └── 2026-02-04.12-47-00-impl-TASK-001-qrs012.toml # Parallel
```

**Integration with Context Write:**
```go
// internal/context/context.go enhancement

func WriteWithYieldPath(dir, phase, taskID string, data map[string]interface{}) error {
    // Generate unique yield path
    yieldPath, err := GenerateYieldPath(dir, phase, taskID)
    if err != nil {
        return fmt.Errorf("failed to generate yield path: %w", err)
    }

    // Add to output section
    if output, ok := data["output"].(map[string]interface{}); ok {
        output["yield_path"] = yieldPath
    }

    // Write context file
    // ... existing Write() logic
}
```

**Error Handling:**
- Project directory does not exist → return error with path
- Cannot create parent directories → return error with permission details
- Absolute path conversion fails → return error (unlikely on valid filesystem)

**Rationale:** Timestamp provides human navigability (chronological sorting), UUID provides machine uniqueness (parallel safety). Project-level directory reduces clutter at .claude/context/ root.

**Alternatives Considered:**
- Flat structure (rejected: too many files in single directory)
- Sequential counter (rejected: requires locking for parallel execution)
- Timestamp only (rejected: not unique for parallel invocations)
- UUID only (rejected: hard for humans to navigate chronologically)

**Traces to:** REQ-2, DES-003, DES-004

---

### ARCH-003: ONNX Runtime Integration Architecture

Reuse existing ONNX runtime infrastructure with auto-download on first use.

**Current Implementation:**
- Location: `internal/memory/embeddings.go`
- Runtime: `github.com/yalue/onnxruntime_go`
- Model: **e5-small** (384 dimensions, BERT-based)
  - Downloaded from HuggingFace: `intfloat/e5-small`
  - Note: Current embeddings.go uses all-MiniLM-L6-v2; implementation phase will update to e5-small
- Auto-download: Downloads ONNX Runtime library and model on first use

**Tokenization Approach:**
- **Simplified hash-based tokenization** for testing/prototyping (current implementation in embeddings.go:281-289)
- Uses word-level hashing modulo 30000 to simulate BERT vocabulary
- **Known limitation:** Not production-quality BERT tokenization
- Real BERT tokenization requires WordPiece tokenizer (future enhancement)
- Current approach sufficient for semantic similarity in development/testing

**Platform Support:**
- macOS: arm64 and x86_64 (dylib)
- Linux: x86_64 (so)
- Windows: x86_64 (dll) - extraction not implemented

**Download Flow:**
```
initializeONNXRuntime(modelDir)
├─> Check if libonnxruntime exists
├─> If not: downloadONNXRuntime()
│   ├─> Determine OS/arch
│   ├─> Download tar.gz from github.com/microsoft/onnxruntime
│   ├─> Extract library to modelDir
│   └─> Set executable permissions
├─> SetSharedLibraryPath(libPath)
└─> InitializeEnvironment()

Query/Extract calls generateEmbeddingONNX()
├─> Check if model exists
├─> If not: downloadModel()
│   ├─> Download from HuggingFace (intfloat/e5-small)
│   ├─> Save to {modelDir}/model.onnx
│   └─> Return path
├─> Create ONNX session with model
├─> Tokenize input (simplified hash-based for development)
├─> Run inference
├─> Mean pooling over sequence dimension
├─> Normalize to unit vector
└─> Return 384-dim float32 vector
```

**No Changes Required:**
Memory extract will use existing `generateEmbeddingONNX()` and `createEmbeddings()` functions without modification.

**Rationale:** Existing implementation already handles auto-download, platform detection, and embedding generation. Memory extract is just another consumer of this infrastructure. The simplified tokenization is sufficient for development and testing; production tokenization can be enhanced later without changing the architecture.

**Alternatives Considered:**
- Different embedding model (e.g., all-MiniLM-L6-v2, OpenAI Ada, Cohere) → Rejected: e5-small specified in requirements, works offline, comparable quality
- Custom embedding implementation → Rejected: ONNX Runtime already handles tensor operations efficiently
- Third-party embedding service (OpenAI, Cohere) → Rejected: Requires API keys, introduces latency, costs money, fails without internet
- Production BERT tokenizer now → Deferred: Current simplified approach sufficient for Layer 0 scope

**Traces to:** REQ-1, REQ-3, REQ-6, REQ-8, DES-006, DES-010

---

### ARCH-004: SQLite-vec Storage Schema

Reuse existing two-table schema for embeddings and metadata.

**Current Schema:**
```sql
-- Metadata table (standard SQLite)
CREATE TABLE IF NOT EXISTS embeddings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    content TEXT NOT NULL,
    source TEXT NOT NULL,
    embedding_id INTEGER  -- foreign key to vec_embeddings.rowid
);

-- Vector table (sqlite-vec virtual table)
CREATE VIRTUAL TABLE IF NOT EXISTS vec_embeddings USING vec0(
    embedding FLOAT[384]
);
```

**Data Flow for Memory Extract:**
```
Extract from result.toml
├─> Parse decisions: {context, choice, reason, alternatives}
├─> Parse learnings: {summary, findings}
├─> For each item:
│   ├─> Generate embedding via generateEmbeddingONNX()
│   ├─> Insert into vec_embeddings → get rowid
│   ├─> Insert into embeddings with embedding_id=rowid
│   └─> Set source field to distinguish origin:
│       ├─> "result:{filename}" for result files
│       └─> "yield:{filename}" for yield files
└─> Return count of items inserted
```

**Query Pattern (unchanged):**
```sql
SELECT e.content, e.source,
       (1 - vec_distance_cosine(v.embedding, ?)) as score
FROM vec_embeddings v
JOIN embeddings e ON e.embedding_id = v.rowid
ORDER BY score DESC
LIMIT ?
```

**Source Field Values:**
- Manual learning: `"memory"` (existing)
- Session end: `"session:{project}"` (existing)
- Memory extract: `"result:{filename}"` or `"yield:{filename}"` (new)

**Rationale:** Two-table design separates vector storage (optimized for similarity search) from metadata (standard SQL queries). Source field enables filtering by origin.

**Alternatives Considered:**
- Separate tables per source (rejected: complicates queries)
- Store filename in separate column (rejected: source field is sufficient)
- JSON metadata column (rejected: adds complexity without benefit)

**Traces to:** REQ-1, REQ-3

---

### ARCH-005: Integration Test Strategy

Integration tests prove end-to-end memory and context workflows using real ONNX inference.

**Test Organization:**
```
internal/memory/memory_integration_test.go
internal/context/context_integration_test.go
```

**Memory Integration Tests:**
```go
func TestMemoryLearnAndQuery(t *testing.T)
    // Learn → Query returns learned content

func TestMemoryDecideAndQuery(t *testing.T)
    // Decide → Query returns decision with alternatives

func TestMemoryExtractFromYield(t *testing.T)
    // Extract from yield.toml → Query returns insights

func TestMemoryExtractFromResult(t *testing.T)
    // Extract from result.toml → Query returns decisions

func TestMemorySessionEndAndQuery(t *testing.T)
    // Session-end → Query returns summary

func TestONNXModelDownload(t *testing.T)
    // First use downloads ONNX runtime and model

func TestEmbeddingGeneration(t *testing.T)
    // Embedding vectors are non-zero, correct dimensions

func TestSemanticSimilarity(t *testing.T)
    // Related queries rank higher than unrelated
    // Example: "error handling" should match "exception management"
    //          better than "ui design"
```

**Context Integration Tests:**
```go
func TestContextWriteGeneratesYieldPath(t *testing.T)
    // Context write includes output.yield_path

func TestYieldPathIsAbsolute(t *testing.T)
    // Yield path starts with /

func TestYieldPathIncludesUUID(t *testing.T)
    // Path contains UUID (unique per invocation)

func TestSequentialContextUniquePaths(t *testing.T)
    // Multiple context writes without taskID get unique paths

func TestParallelContextUniquePaths(t *testing.T)
    // Multiple context writes with different taskIDs get unique paths

func TestParallelContextSameTaskUniquePaths(t *testing.T)
    // Multiple invocations with same taskID get unique paths via UUID

func TestSkillCanReadYieldPath(t *testing.T)
    // Mock skill reads output.yield_path and writes result there

func TestResultFileReadable(t *testing.T)
    // Result file at yield_path is parseable by context read
```

**Test Infrastructure:**
- Use `t.TempDir()` for isolated test directories
- Use `t.Setenv()` for environment isolation
- Use `testing.Short()` to skip slow tests in fast mode
- Use `require` package for assertions (fail fast on error)
- Use gomega matchers for readable assertions

**Semantic Similarity Test Approach:**
```go
// Test that semantic similarity works correctly
func TestSemanticSimilarity(t *testing.T) {
    // Setup: Store related and unrelated content
    Learn("error handling with try-catch")
    Learn("exception management strategies")
    Learn("user interface design patterns")

    // Query: Search for related topic
    results := Query("handling errors")

    // Assert: Related content ranks higher than unrelated
    require.Len(results, 3)
    assert.Contains(results[0].Content, "error handling")
    assert.Contains(results[1].Content, "exception management")
    // UI design should rank lowest
}
```

**Test Execution:**
- Integration tests run on macOS and Linux (CI)
- Windows support documented as future work
- Tests skip auto-download if model already present (CI caching)

**Rationale:** Integration tests prove that components work together correctly. Unit tests prove individual functions work; integration tests prove the system works.

**Traces to:** REQ-3, REQ-4, REQ-9

---

### ARCH-006: Error Handling Strategy

Consistent error handling across Layer 0 commands using Go error wrapping and structured error types.

**Error Wrapping Pattern:**
```go
// All functions return wrapped errors with context
func Extract(opts ExtractOpts) (*ExtractResult, error) {
    // Read file
    data, err := os.ReadFile(opts.FilePath)
    if err != nil {
        return nil, fmt.Errorf("failed to read file %s: %w", opts.FilePath, err)
    }

    // Parse TOML
    var parsed YieldFile
    if err := toml.Unmarshal(data, &parsed); err != nil {
        return nil, fmt.Errorf("failed to parse TOML in %s: %w", opts.FilePath, err)
    }

    // ... rest of implementation
}
```

**Structured Error Types (when needed):**
```go
// For errors that need special handling
type SchemaValidationError struct {
    Field    string
    Expected string
    Actual   string
    Line     int
}

func (e *SchemaValidationError) Error() string {
    return fmt.Sprintf("schema validation failed: field %s expected %s, got %s (line %d)",
        e.Field, e.Expected, e.Actual, e.Line)
}
```

**Error Recovery:**
- No recovery: All errors bubble up to CLI layer
- CLI layer formats errors per DES-009 (human-readable with technical details)
- Fail fast: First error returns immediately (no error collection)

**Validation Strategy:**
- Use struct tags for TOML validation where possible
- Use explicit checks for business logic validation
- Return errors immediately on validation failure
- Include field name and expected value in error message

**Terminal Output Format (CLI layer):**
```go
// cmd/projctl/memory_extract.go
func runExtract(cmd *cobra.Command, args []string) error {
    result, err := memory.Extract(opts)
    if err != nil {
        // Format error per DES-009
        fmt.Fprintf(os.Stderr, "✗ memory extract failed: %s\n", briefReason(err))
        fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
        if suggestion := errorSuggestion(err); suggestion != "" {
            fmt.Fprintf(os.Stderr, "  %s\n", suggestion)
        }
        return err
    }

    // Success output...
}
```

**Rationale:** Go's error wrapping provides context without complexity. Structured error types only when special handling needed (rare). CLI layer owns formatting.

**Alternatives Considered:**
- Sentinel errors (`var ErrFileNotFound = errors.New(...)`) → Rejected: Less flexible than wrapped errors, requires global error definitions
- Error codes (integer constants) → Rejected: Not idiomatic Go, harder to read than error messages
- Panic/recover → Rejected: Only for truly unrecoverable errors, not for validation or I/O failures

**Traces to:** REQ-1, REQ-2, DES-009

---

### ARCH-007: Documentation Organization

Enhance existing command documentation files rather than creating new files.

**File Structure:**
```
docs/
├── commands/
│   ├── memory.md         # [NEW]: Create with all memory commands and architecture
│   ├── trace.md          # Enhanced: Add repair behavior documentation
│   └── context.md        # Enhanced: Add yield_path pattern
└── layer-0-implementation.md  # [NEW]: Summary/index for Layer 0
```

**Enhancement Strategy:**
- Check if `docs/commands/{command}.md` exists before creating
- If exists: add new sections under appropriate headers
- If not exists: create with full structure (see DES-006, DES-007)
- Use consistent section structure across all command docs
- **Status verified:** `docs/commands/memory.md` does not currently exist, will be created as new file

**memory.md Structure:**
```markdown
# Memory Commands

## Overview
[Semantic memory vs. grep, when to use each]

## Commands

### memory query
[Existing content...]

### memory learn
[Existing content...]

### memory decide
[Existing content...]

### memory extract  ← NEW SECTION
[CLI interface, flags, output format, examples]

### memory grep
[Existing content...]

### memory session-end
[Existing content...]

## Architecture  ← NEW SECTION

### Embedding Model
[ONNX e5-small, 384 dimensions]

### Storage
[SQLite-vec location: ~/.claude/memory]

### First Use
[Auto-download behavior, platforms supported]

## Examples
[Real-world usage scenarios]
```

**trace.md Enhancement:**
```markdown
# Trace Commands

[Existing content...]

## trace repair  ← NEW SECTION

Automatically fixes duplicate IDs and escalates dangling references.

### Auto-Fixed: Duplicate IDs
[Renumbering behavior, examples]

### Escalated: Dangling References
[Manual fix required, escalation file format, examples]

### Examples
[Output for each case, escalation file format]
```

**layer-0-implementation.md Structure:**
- Overview of Layer 0 (reference to orchestration-system.md Section 13.3)
- Command inventory by category (state, context, trace, memory, etc.)
- Mark each command as [EXISTING], [ENHANCED], or [NEW]
- Architecture summary (ONNX + SQLite-vec)
- Key patterns (yield path generation, auto-download, parallel safety)
- Links to detailed command docs

**Rationale:** Users prefer consolidated documentation per command group over fragmented per-subcommand files. Easier to maintain single source of truth.

**Alternatives Considered:**
- One file per subcommand (rejected: too fragmented)
- Everything in README (rejected: mixes usage with implementation)
- Separate architecture doc (rejected: duplicates command-specific details)

**Traces to:** REQ-5, REQ-6, REQ-7, REQ-8, DES-005, DES-006, DES-007, DES-008

---

### ARCH-008: Dependency Injection for Testing

Use constructor injection to enable testing without I/O side effects.

**Pattern (already used in codebase):**
```go
// Opts struct with injected dependencies
type ExtractOpts struct {
    FilePath   string
    MemoryRoot string
    ModelDir   string

    // Injected for testing (optional)
    ReadFile  func(string) ([]byte, error)  // default: os.ReadFile
    WriteDB   func(*sql.DB, string) error   // default: real DB insert
}

// Implementation uses injected functions
func Extract(opts ExtractOpts) (*ExtractResult, error) {
    readFile := opts.ReadFile
    if readFile == nil {
        readFile = os.ReadFile
    }

    data, err := readFile(opts.FilePath)
    // ... rest of implementation
}
```

**Test Usage:**
```go
func TestExtractParseError(t *testing.T) {
    opts := ExtractOpts{
        FilePath: "test.toml",
        ReadFile: func(string) ([]byte, error) {
            return []byte("invalid toml [[["), nil
        },
    }

    _, err := Extract(opts)
    require.Error(t, err)
    assert.Contains(t, err.Error(), "parse TOML")
}
```

**When to Use:**
- Command-level functions: Always accept Opts with optional injection
- Internal helpers: Use injection when I/O is involved
- Pure functions: No injection needed (deterministic)

**Already Used In:**
- `internal/memory` package (MemoryRoot, ModelDir injection)
- `internal/context` package (directory injection)
- Test packages use `t.TempDir()` for isolation

**Rationale:** Explicit is better than implicit. Opts structs make dependencies visible. Optional injection avoids test-specific code in production paths.

**Alternatives Considered:**
- Interface-based DI (dependency injection container) → Rejected: Overkill for CLI tool, adds boilerplate, harder to understand
- Context-based injection (`context.WithValue`) → Rejected: Not type-safe, requires casting, easy to misuse
- Global mocks (replace package-level functions) → Rejected: Not thread-safe, requires cleanup, pollutes production code

**Traces to:** REQ-3, REQ-4

---

### ARCH-009: Trace Repair Implementation (Documentation Only)

Document existing trace repair implementation without changes.

**Current Implementation:**
- Location: `internal/trace/trace.go`
- Behavior: Renumber duplicate IDs, escalate dangling references
- Already correct: No implementation changes needed

**Documentation Task:**
- Add section to `docs/commands/trace.md`
- Explain auto-fix behavior for duplicate IDs
- Explain escalation behavior for dangling references
- Show example output for each case
- Show example escalation file format

**No Code Changes:**
This ARCH entry exists to document that trace repair is complete and only needs documentation updates.

**Traces to:** REQ-5, REQ-9, DES-007

---

## Implementation Plan

### Phase 1: Core Functionality (P0)

1. **Memory Extract Command** (ARCH-001)
   - Add CLI command: `cmd/projctl/memory_extract.go`
   - Add internal function: `internal/memory.Extract()`
   - Define TOML structs for yield/result schema validation
   - Wire up existing embedding generation and storage
   - Add terminal output formatting

2. **Yield Path Generation** (ARCH-002)
   - Add function: `internal/context.GenerateYieldPath()`
   - Integrate with context write
   - Store project UUID in state.toml
   - Create directory structure on path generation

### Phase 2: Testing (P1)

3. **Memory Integration Tests** (ARCH-005)
   - Test: learn → query
   - Test: decide → query
   - Test: extract → query
   - Test: ONNX model download
   - Test: embedding generation
   - Test: semantic similarity

4. **Context Integration Tests** (ARCH-005)
   - Test: context write generates yield_path
   - Test: yield_path properties (absolute, unique, includes UUID)
   - Test: parallel execution generates unique paths
   - Test: mock skill can use yield_path

5. **Trace Repair Integration Tests** (REQ-9, ARCH-005)
   - Test: duplicate ID renumbering
   - Test: dangling reference escalation
   - Test: idempotency

### Phase 3: Documentation (P1-P2)

6. **Command Documentation** (ARCH-007)
   - Create: `docs/commands/memory.md`
   - Enhance: `docs/commands/trace.md`
   - Enhance: `docs/commands/context.md`
   - Create: `docs/layer-0-implementation.md`

---

## Success Criteria

1. **Functionality:** `projctl memory extract` works for both --result and --yield flags
2. **Uniqueness:** Multiple context writes generate unique yield_path values
3. **Integration:** Memory extract → query returns extracted content with correct semantic similarity
4. **Testing:** All integration tests pass on macOS and Linux
5. **Documentation:** User can understand and use all Layer 0 commands from docs

---

## Non-Functional Requirements

### Performance
- Memory extract should process typical result file (<10KB) in <500ms including embedding generation
- Yield path generation should complete in <10ms (just UUID generation + path join)
- First-use ONNX download acceptable to take 30-60 seconds (one-time cost)

### Platform Support
- macOS: arm64 and x86_64 (primary)
- Linux: x86_64 (CI)
- Windows: Document as future work (ONNX Runtime extraction not implemented)

### Security
- No credentials or secrets in memory system (validate at CLI level)
- File permissions: 0644 for files, 0755 for directories
- No arbitrary code execution (TOML parsing only)

### Maintainability
- Reuse existing infrastructure (no new dependencies)
- Consistent error handling (wrapped errors with context)
- Integration tests prevent regressions
- Documentation kept in sync with implementation

---

## References

- **Specification:** docs/orchestration-system.md Section 13.3 Layer 0: Foundation
- **Requirements:** .claude/projects/layer-0-foundation/requirements.md
- **Design:** .claude/projects/layer-0-foundation/design.md
- **Issue:** ISSUE-045
- **Existing Code:**
  - `internal/memory/memory.go` - Memory commands implementation
  - `internal/memory/embeddings.go` - ONNX Runtime integration
  - `internal/context/context.go` - Context management
  - `internal/trace/trace.go` - Trace validation and repair
  - `internal/state/state.go` - State management
