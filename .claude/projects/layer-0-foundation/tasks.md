# Layer 0: Foundation - Tasks

**Project:** layer-0-foundation
**Phase:** Breakdown
**Created:** 2026-02-04

**Traces to:** ISSUE-045

---

## Task Dependency Graph

```
TASK-1 (types/structs)
    |
    +---> TASK-2 (path generation) ---> TASK-6 (context write integration)
    |                                        |
    +---> TASK-3 (TOML parsing/validation)  |
              |                              |
              +---> TASK-4 (extract core) ---+---> TASK-7 (memory extract CLI)
                         |                   |
                         |                   +---> TASK-8 (memory integration tests)
                         |                   |
                         +------------------+---> TASK-9 (context integration tests)
                                             |
TASK-5 (trace repair tests) ----------------+
                                             |
                                             +---> TASK-10 (memory docs)
                                             |
                                             +---> TASK-11 (trace docs)
                                             |
                                             +---> TASK-12 (context docs)
                                             |
                                             +---> TASK-13 (layer-0 summary docs)
```

---

## Tasks

### TASK-1: Define core types and error structures

**Description:** Define Go types for memory extraction (ExtractOpts, ExtractResult), yield/result file schemas (YieldFile, ResultFile), and structured error types (SchemaValidationError). These types form the foundation for all memory and context operations.

**Status:** Complete

**Acceptance Criteria:**
- [x] `internal/memory/types.go` exists with ExtractOpts struct
- [x] ExtractOpts includes FilePath, MemoryRoot, ModelDir fields
- [x] ExtractOpts includes optional injection fields (ReadFile, WriteDB) for testing
- [x] ExtractResult struct includes status, file path, items_extracted count
- [x] ExtractResult includes slice of extracted items (type, context, content)
- [x] YieldFile and ResultFile structs match yield protocol schema
- [x] Structs use TOML struct tags for validation (`toml:"field_name"`)
- [x] SchemaValidationError struct includes Field, Expected, Actual, Line fields
- [x] SchemaValidationError implements error interface
- [x] All types have godoc comments
- [x] Unit tests verify struct tags map to TOML fields correctly

**Files:** `internal/memory/types.go`, `internal/memory/types_test.go`

**Dependencies:** None

**Traces to:** ARCH-001, ARCH-004, ARCH-006, ARCH-008

---

### TASK-2: Implement yield path generation function

**Description:** Implement GenerateYieldPath() function that creates unique file paths using timestamp + UUID pattern for parallel execution safety. Function generates project-level directory structure and ensures absolute paths.

**Status:** Complete

**Acceptance Criteria:**
- [x] `internal/context/yieldpath.go` exists with GenerateYieldPath function
- [x] Function signature: `GenerateYieldPath(projectDir, phase, taskID string) (string, error)`
- [x] Path pattern for parallel: `.claude/context/{date}-{project}-{projectUUID}/{datetime}-{phase}-{taskID}-{fileUUID}.toml`
- [x] Path pattern for sequential: `.claude/context/{date}-{project}-{projectUUID}/{datetime}-{phase}-{fileUUID}.toml`
- [x] Date format: YYYY-MM-DD (e.g., 2026-02-04)
- [x] Datetime format: YYYY-MM-DD.HH-mm-SS (e.g., 2026-02-04.12-45-30)
- [x] Project UUID retrieved from or stored in state.toml
- [x] File UUID generated per invocation using github.com/google/uuid
- [x] Returns absolute path (via filepath.Abs)
- [x] Creates parent directories with mode 0755 (via os.MkdirAll)
- [x] Returns error with context on permission failures
- [x] Unit tests verify path format matches pattern
- [x] Unit tests verify multiple invocations return unique paths
- [x] Unit tests verify absolute path returned
- [x] Property tests verify uniqueness across parallel invocations

**Files:** `internal/context/yieldpath.go`, `internal/context/yieldpath_test.go`

**Dependencies:** TASK-1

**Traces to:** REQ-2, ARCH-002, DES-003

---

### TASK-3: Implement TOML parsing and schema validation

**Description:** Create TOML parsing logic with strict schema validation that fails fast on invalid structure. Uses YieldFile/ResultFile structs from TASK-1 with BurntSushi/toml unmarshaling.

**Status:** Complete

**Acceptance Criteria:**
- [x] `internal/memory/parse.go` exists with ParseYieldFile and ParseResultFile functions
- [x] ParseYieldFile signature: `ParseYieldFile(data []byte) (*YieldFile, error)`
- [x] ParseResultFile signature: `ParseResultFile(data []byte) (*ResultFile, error)`
- [x] Uses toml.Unmarshal with struct tags for validation
- [x] Returns wrapped error with context on parse failure
- [x] Returns SchemaValidationError on missing required fields
- [x] Error includes field name, expected type, line number when available
- [x] Fails fast: first error returns immediately
- [x] Unit tests verify valid TOML parses successfully
- [x] Unit tests verify invalid TOML returns parse error
- [x] Unit tests verify missing required field returns SchemaValidationError
- [x] Unit tests verify error message includes field name and expected type
- [x] Property tests with random valid/invalid TOML structures

**Files:** `internal/memory/parse.go`, `internal/memory/parse_test.go`

**Dependencies:** TASK-1

**Traces to:** REQ-1, ARCH-001, DES-001, DES-009

---

### TASK-4: Implement memory extract core logic

**Description:** Implement Extract() function that parses yield/result TOML files, extracts decisions and learnings, generates embeddings via ONNX, and stores in SQLite-vec. Reuses existing embedding infrastructure.

**Status:** Complete

**Acceptance Criteria:**
- [x] `internal/memory/extract.go` exists with Extract function
- [x] Function signature: `Extract(opts ExtractOpts) (*ExtractResult, error)`
- [x] Uses ParseYieldFile/ParseResultFile from TASK-3
- [x] Extracts decisions from [payload.decisions] arrays
- [x] Extracts learnings from [payload] fields (summary, findings, etc.)
- [x] Extracts context (phase, subphase, status)
- [x] Calls existing generateEmbeddingONNX() for each item
- [x] Calls existing createEmbeddings() to store in SQLite-vec
- [x] Sets source field to "result:{filename}" or "yield:{filename}"
- [x] Returns ExtractResult with count and items
- [x] Uses injected ReadFile function if provided (for testing)
- [x] Returns wrapped error with context on any failure
- [x] Unit tests with mocked file reading verify extraction logic
- [x] Unit tests verify source field format
- [x] Unit tests verify error wrapping includes file path

**Files:** `internal/memory/extract.go`, `internal/memory/extract_test.go`

**Dependencies:** TASK-1, TASK-3

**Traces to:** REQ-1, ARCH-001, ARCH-003, ARCH-004

---

### TASK-5: Add integration tests for trace repair

**Description:** Integration tests proving trace repair renumbers duplicate IDs and escalates dangling references. Tests use real files and verify idempotency.

**Status:** Complete

**Acceptance Criteria:**
- [x] `internal/trace/trace_integration_test.go` exists
- [x] Test: duplicate ID in same file → auto-renumbered
- [x] Test: duplicate ID across files → auto-renumbered
- [x] Test: dangling reference → escalation file created
- [x] Test verifies renumbering uses next available ID
- [x] Test verifies escalation file contains ref details (source file, line, referenced ID)
- [x] Test verifies no escalation for duplicate IDs (auto-fixed)
- [x] Test verifies repair is idempotent (running twice produces same result)
- [x] Tests use t.TempDir() for isolation
- [x] Tests use gomega matchers for readable assertions

**Files:** `internal/trace/trace_integration_test.go`

**Dependencies:** None (tests existing code)

**Traces to:** REQ-9, ARCH-005, ARCH-009

---

### TASK-6: Integrate yield path generation with context write

**Description:** Enhance context.Write() to call GenerateYieldPath() and include output.yield_path field in generated context files. Skills read this field to know where to write results.

**Status:** Complete

**Acceptance Criteria:**
- [x] `internal/context/context.go` enhanced with yield path integration
- [x] WriteWithYieldPath function exists: `WriteWithYieldPath(dir, phase, taskID string, data map[string]interface{}) error`
- [x] Calls GenerateYieldPath from TASK-2
- [x] Adds output.yield_path field to context data
- [x] yield_path is absolute path
- [x] Returns error before writing context if path generation fails
- [x] Existing Write() function unchanged (backward compatibility)
- [x] Unit tests verify output.yield_path field present in context
- [x] Unit tests verify yield_path is absolute
- [x] Unit tests verify error returned on path generation failure

**Files:** `internal/context/context.go`, `internal/context/context_test.go`

**Dependencies:** TASK-2

**Traces to:** REQ-2, ARCH-002, DES-004

---

### TASK-7: [visual] Add memory extract CLI command

**Description:** Add `projctl memory extract` CLI command with --result and --yield flags. Formats output as TOML (machine-readable) and terminal summary (human-readable) per DES-001 and DES-002.

**Status:** Complete

**Acceptance Criteria:**
- [x] `cmd/projctl/memory_extract.go` exists
- [x] Cobra command registered: `memory extract`
- [x] Flag: `--result <path>` (mutually exclusive with --yield)
- [x] Flag: `--yield <path>` (mutually exclusive with --result)
- [x] Error if both flags or neither flag provided
- [x] Calls memory.Extract() from TASK-4
- [x] Terminal output shows success: "✓ Extracted N items from {file}"
- [x] Terminal output shows item breakdown: "- X decisions\n- Y learnings"
- [x] Terminal output shows storage location: "Stored in semantic memory (~/.claude/memory/embeddings.db)"
- [x] Terminal output shows errors per DES-009 format
- [x] TOML output written to stdout (for orchestrator consumption)
- [x] Integration test via subprocess verifies CLI works end-to-end
- [x] Integration test verifies --result flag works
- [x] Integration test verifies --yield flag works
- [x] Integration test verifies mutual exclusion error

**Files:** `cmd/projctl/memory_extract.go`, `cmd/projctl/memory_extract_test.go`

**Dependencies:** TASK-4

**Traces to:** REQ-1, ARCH-001, DES-001, DES-002, DES-009

---

### TASK-8: Add memory system integration tests

**Description:** Integration tests proving memory system works end-to-end: learn/decide/extract → query returns relevant results with semantic similarity. Tests verify ONNX model download and embedding generation.

**Status:** Complete

**Acceptance Criteria:**
- [ ] `internal/memory/memory_integration_test.go` exists
- [ ] Test: memory learn → query returns learned content
- [ ] Test: memory decide → query returns decision with alternatives
- [ ] Test: memory extract from yield → query returns insights
- [ ] Test: memory extract from result → query returns decisions
- [ ] Test: memory session-end → query returns summary
- [ ] Test: ONNX model downloads on first use (check file created)
- [ ] Test: SQLite-vec database created at expected location
- [ ] Test: embedding generation produces non-zero vectors
- [ ] Test: embedding vectors have correct dimensions (384 for e5-small)
- [ ] Test: semantic similarity works (related queries rank higher than unrelated)
- [ ] Example similarity test: "error handling" matches "exception management" better than "ui design"
- [ ] Tests use t.TempDir() for isolated test directories
- [ ] Tests use t.Setenv() for environment isolation
- [ ] Tests skip auto-download if model already present (CI caching)
- [ ] Tests use testing.Short() to skip slow tests in fast mode
- [ ] Tests run on macOS and Linux (document Windows as future work)

**Files:** `internal/memory/memory_integration_test.go`

**Dependencies:** TASK-4, TASK-7

**Traces to:** REQ-3, ARCH-003, ARCH-005

---

### TASK-9: Add context write integration tests

**Description:** Integration tests proving context write generates valid unique yield paths for sequential and parallel execution. Tests verify mock skills can read yield_path and write results.

**Status:** Complete

**Acceptance Criteria:**
- [x] `internal/context/context_integration_test.go` exists
- [x] Test: context write generates context file with output.yield_path
- [x] Test: yield_path is absolute path (starts with /)
- [x] Test: yield_path includes UUID (unique per invocation)
- [x] Test: yield_path matches expected pattern with timestamp and UUID
- [x] Test: sequential context (no taskID) gets unique path
- [x] Test: parallel contexts (different taskIDs) get unique paths
- [x] Test: parallel contexts (same taskID, different invocations) get unique paths via UUID
- [x] Test: mock skill can read output.yield_path from context
- [x] Test: mock skill can write result to yield_path location
- [x] Test: result file at yield_path is readable by context read
- [x] Tests use t.TempDir() for isolation
- [x] Tests use gomega matchers for readable assertions

**Files:** `internal/context/context_integration_test.go`

**Dependencies:** TASK-6

**Traces to:** REQ-4, ARCH-002, ARCH-005

---

### TASK-10: [visual] Create memory commands documentation

**Description:** Create comprehensive docs/commands/memory.md documenting all memory commands (query, learn, decide, extract, grep, session-end), architecture (ONNX + e5-small + SQLite-vec), and first-use auto-download behavior.

**Status:** Complete

**Acceptance Criteria:**
- [x] `docs/commands/memory.md` exists
- [ ] Overview section explains semantic memory vs. grep (when to use each)
- [ ] Commands section documents each memory subcommand
- [ ] memory query: semantic search examples with flags
- [ ] memory learn: store arbitrary insights example
- [ ] memory decide: store decisions with context, choice, reason, alternatives example
- [ ] memory extract: CLI interface, --result and --yield flags, output format, examples
- [ ] memory grep: pattern-based search examples
- [ ] memory session-end: end-of-session summary example
- [ ] Architecture section documents embedding model (e5-small, 384 dimensions)
- [ ] Architecture section documents storage (SQLite-vec location: ~/.claude/memory)
- [ ] Architecture section documents first use (auto-download behavior, platforms supported)
- [ ] Examples section shows real-world usage scenarios
- [ ] Documents supported platforms: macOS (arm64, x86_64), Linux (x86_64)
- [ ] Documents future work: Windows support
- [ ] Shows expected first-run output (download progress per DES-010)
- [ ] References orchestration-system.md Section 13.3

**Files:** `docs/commands/memory.md`

**Dependencies:** TASK-7, TASK-8

**Traces to:** REQ-6, REQ-8, ARCH-007, DES-006, DES-010

---

### TASK-11: [visual] Add trace repair documentation

**Description:** Enhance docs/commands/trace.md with trace repair section explaining auto-fix behavior (duplicate IDs renumbered) and escalation behavior (dangling references require manual fix). Include examples of output and escalation files.

**Status:** Complete

**Acceptance Criteria:**
- [x] `docs/commands/trace.md` exists or is created
- [x] "trace repair" section added
- [x] Explains automatic renumbering of duplicate IDs
- [x] Shows example: duplicate REQ-5 in two files → one renumbered to REQ-10
- [x] Explains escalation of dangling references
- [x] Shows example: ARCH-001 references missing DES-005 → escalation created
- [x] Shows example escalation file format
- [x] Escalation file includes: source file, line number, referenced ID, reason
- [x] Documents that duplicate IDs never create escalations (auto-fixed)
- [x] Documents that repair is idempotent (safe to run multiple times)
- [x] Shows terminal output examples for both cases
- [x] References orchestration-system.md Section 13.3

**Files:** `docs/commands/trace.md`

**Dependencies:** TASK-5

**Traces to:** REQ-5, ARCH-007, ARCH-009, DES-007

---

### TASK-12: [visual] Add context write yield_path documentation

**Description:** Enhance docs/commands/context.md (or create if missing) to document output.yield_path pattern. Explains how skills read yield_path from context and write results there, enabling parallel execution.

**Status:** Complete

**Acceptance Criteria:**
- [x] `docs/commands/context.md` exists or is created
- [x] Documents context write command
- [x] Explains output.yield_path field in generated context files
- [x] Shows example context file with output.yield_path
- [x] Shows yield_path format: `.claude/context/{date}-{project}-{uuid}/{datetime}-{phase}-{taskID}-{uuid}.toml`
- [x] Explains that yield_path is always absolute
- [x] Explains that UUID ensures uniqueness for parallel execution
- [x] Shows skill usage pattern: read context → extract yield_path → write result
- [x] Example: skill reads yield_path and writes result.toml there
- [x] Documents sequential vs. parallel path patterns
- [x] References orchestration-system.md Section 13.3

**Files:** `docs/commands/context.md`

**Dependencies:** TASK-9

**Traces to:** REQ-2, ARCH-002, ARCH-007, DES-003, DES-004

---

### TASK-13: [visual] Create Layer 0 implementation summary

**Description:** Create docs/layer-0-implementation.md as high-level summary showing all Layer 0 commands by category, marking status (existing/enhanced/new), architecture summary, key patterns, and links to detailed docs.

**Status:** Complete

**Acceptance Criteria:**
- [x] `docs/layer-0-implementation.md` exists
- [ ] Overview section references orchestration-system.md Section 13.3
- [ ] Command inventory section organized by category
- [ ] State commands listed: get, transition, next (mark as EXISTING)
- [ ] Context commands listed: write (mark as ENHANCED with yield_path), read (mark as EXISTING)
- [ ] ID commands listed: next (mark as EXISTING)
- [ ] Trace commands listed: validate (mark as EXISTING), repair (mark as EXISTING, DOCUMENTED)
- [ ] Territory commands listed: map, show (mark as EXISTING)
- [ ] Memory commands listed:
  - query (mark as NEW: semantic search)
  - learn (mark as NEW: arbitrary insights)
  - grep (mark as EXISTING)
  - extract (mark as NEW: from yield/result files)
  - session-end (mark as EXISTING)
- [ ] Architecture section summarizes: ONNX runtime + e5-small model + SQLite-vec
- [ ] Key patterns section documents: yield path generation, auto-download, parallel safety via UUID
- [ ] Documentation section links to: memory.md, trace.md, context.md
- [ ] References section links to orchestration-system.md Section 13.3

**Files:** `docs/layer-0-implementation.md`

**Dependencies:** TASK-10, TASK-11, TASK-12

**Traces to:** REQ-7, ARCH-007, DES-008

---

## Success Criteria

1. **Functionality:** `projctl memory extract --result` and `--yield` work end-to-end
2. **Uniqueness:** Context write generates unique yield_path for parallel execution
3. **Integration:** Memory extract → query returns expected results with semantic similarity
4. **Testing:** All integration tests pass (memory, context, trace)
5. **Documentation:** User can understand and use all Layer 0 commands from docs

---

## Task Summary

- **Total Tasks:** 13
- **Foundation (Pure):** 2 (TASK-1, TASK-2)
- **Core Logic:** 2 (TASK-3, TASK-4)
- **Integration:** 2 (TASK-6, TASK-7)
- **Testing:** 3 (TASK-5, TASK-8, TASK-9)
- **Documentation:** 4 (TASK-10, TASK-11, TASK-12, TASK-13)

---

## Parallel Execution Opportunities

**Can run in parallel:**
- TASK-1 (types) is foundation for multiple branches
- TASK-5 (trace tests) has no dependencies - can start immediately
- After TASK-1 completes: TASK-2 (path gen) and TASK-3 (parsing) can run in parallel
- After TASK-4 completes: TASK-7 (CLI) and TASK-8 (memory tests) can run in parallel
- After TASK-6 completes: TASK-9 (context tests) can run in parallel
- After all tests complete: TASK-10, TASK-11, TASK-12 can run in parallel
