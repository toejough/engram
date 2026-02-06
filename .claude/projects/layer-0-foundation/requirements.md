# Layer 0: Foundation - Requirements

**Project:** layer-0-foundation
**Phase:** PM
**Created:** 2026-02-04

**Traces to:** ISSUE-45

---

## Overview

Implement core projctl infrastructure without agent spawning, completing Layer 0 as defined in docs/orchestration-system.md Section 13.3. This includes filling implementation gaps for memory extraction, centralized yield path generation, integration testing, and comprehensive documentation.

---

## Requirements

### REQ-1: Memory Extract Command

As a projctl orchestrator, I want to extract key insights from yield/result TOML files and store them in the semantic memory system, so that future agents can benefit from prior learnings and decisions.

**Acceptance Criteria:**
- [ ] `projctl memory extract` command exists
- [ ] Accepts `--result` flag with path to result.toml file
- [ ] Accepts `--yield` flag with path to yield.toml file
- [ ] Parses TOML structure to extract:
  - Decisions (from `[payload.decisions]` arrays)
  - Learnings/insights (from `[payload]` fields like `summary`, `findings`, etc.)
  - Key context (phase, subphase, status)
- [ ] Generates embeddings via ONNX e5-small model
- [ ] Stores embeddings in SQLite-vec database
- [ ] Returns success/failure with count of items extracted
- [ ] CLI usage: `projctl memory extract --result .claude/context/pm-result.toml`
- [ ] Integration: Called by orchestrator after QA approval

**Priority:** P0

**Traces to:** ISSUE-45

---

### REQ-2: Centralized Yield Path Generation

As a projctl orchestrator, I want a single method to generate unique yield/result file paths, so that parallel execution doesn't cause file conflicts.

**Acceptance Criteria:**
- [ ] Internal package function `GenerateYieldPath(projectDir, phase, taskID string) string`
- [ ] Generates unique path using pattern: `.claude/context/{phase}-{taskID}-{uuid}.toml`
- [ ] For sequential execution (no taskID): `.claude/context/{phase}-{uuid}.toml`
- [ ] UUID ensures uniqueness even for parallel invocations of same task
- [ ] `projctl context write` includes `output.yield_path` field in generated context
- [ ] `output.yield_path` uses absolute path (not relative)
- [ ] Skills read `output.yield_path` from context and write results there
- [ ] Documentation updated to show yield_path usage pattern

**Priority:** P0

**Traces to:** ISSUE-45

---

### REQ-3: Memory System Integration Tests

As a projctl maintainer, I want integration tests proving the memory system works end-to-end, so that semantic queries return relevant results.

**Acceptance Criteria:**
- [ ] Integration test: learn → query returns learned content
- [ ] Integration test: decide → query returns decision
- [ ] Integration test: extract from yield → query returns extracted insights
- [ ] Integration test: extract from result → query returns decisions
- [ ] Integration test: session-end → query returns summary
- [ ] Tests verify ONNX model downloads on first use
- [ ] Tests verify SQLite-vec database creation
- [ ] Tests verify embedding generation produces non-zero vectors
- [ ] Tests verify semantic similarity (related queries rank higher than unrelated)
- [ ] Tests run on macOS and Linux (document Windows as future work)

**Priority:** P1

**Traces to:** ISSUE-45

---

### REQ-4: Context Write Integration Tests

As a projctl maintainer, I want integration tests proving context write generates valid yield paths, so that parallel execution works correctly.

**Acceptance Criteria:**
- [ ] Integration test: `context write` generates context file with `output.yield_path`
- [ ] Test verifies yield_path is absolute path
- [ ] Test verifies yield_path includes UUID (unique per invocation)
- [ ] Test verifies sequential context (no taskID) gets unique path
- [ ] Test verifies parallel contexts (different taskIDs) get unique paths
- [ ] Test verifies parallel contexts (same taskID, different invocations) get unique paths via UUID
- [ ] Mock skill can read `output.yield_path` and write result there
- [ ] Result file at yield_path is readable by `context read`

**Priority:** P1

**Traces to:** ISSUE-45

---

### REQ-5: Trace Repair Documentation

As a projctl user, I want documentation explaining trace repair behavior, so that I understand when auto-fixes happen vs. when escalations are created.

**Acceptance Criteria:**
- [ ] docs/commands/trace.md (or equivalent) documents `trace repair`
- [ ] Explains duplicate ID auto-fix (renumbering)
- [ ] Explains dangling reference escalation (manual fix required)
- [ ] Shows example output for each case
- [ ] Shows example escalation file created
- [ ] References orchestration-system.md Layer 0 section
- [ ] User can understand repair behavior without reading source code

**Priority:** P1

**Traces to:** ISSUE-45

---

### REQ-6: Memory System Documentation

As a projctl user, I want comprehensive memory system documentation, so that I understand how to query, learn, and extract insights.

**Acceptance Criteria:**
- [ ] docs/commands/memory.md (or equivalent) documents all memory commands
- [ ] Explains semantic query vs. grep (when to use each)
- [ ] Shows example: `memory learn` with message
- [ ] Shows example: `memory decide` with context, choice, reason, alternatives
- [ ] Shows example: `memory extract` from result file
- [ ] Shows example: `memory query` with text and limit
- [ ] Shows example: `memory grep` with pattern
- [ ] Shows example: `memory session-end` for project
- [ ] Explains ONNX model auto-download on first use
- [ ] Explains SQLite-vec database location (~/.claude/memory)
- [ ] Documents embedding model (e5-small, 384 dimensions)
- [ ] References orchestration-system.md Layer 0 section

**Priority:** P1

**Traces to:** ISSUE-45

---

### REQ-7: Layer 0 Summary Documentation

As a projctl contributor, I want a summary document showing what was implemented for Layer 0, so that I can see the foundation before moving to Layer 1.

**Acceptance Criteria:**
- [ ] docs/layer-0-implementation.md (or equivalent) exists
- [ ] Lists all Layer 0 commands with status (existing vs. new)
- [ ] State commands: get, transition, next (existing)
- [ ] Context commands: write (with yield_path), read (existing + enhancement)
- [ ] ID commands: next (existing)
- [ ] Trace commands: validate, repair (existing)
- [ ] Territory commands: map, show (existing)
- [ ] Memory commands: query, learn, grep, extract (new), session-end (existing)
- [ ] Shows architecture: ONNX runtime + e5-small + SQLite-vec
- [ ] Shows dependency auto-download approach
- [ ] Shows yield_path generation pattern
- [ ] Links to command-specific docs (memory.md, trace.md)
- [ ] References orchestration-system.md Section 13.3

**Priority:** P2

**Traces to:** ISSUE-45

---

### REQ-8: Dependency Auto-Download Documentation

As a projctl user, I want to know what happens on first use of memory commands, so that I'm not surprised by model downloads.

**Acceptance Criteria:**
- [ ] README.md or docs/memory.md documents first-use behavior
- [ ] Explains e5-small model download (~130MB)
- [ ] Explains ONNX runtime download (platform-specific)
- [ ] Shows download location (e.g., ~/.claude/models/)
- [ ] Explains that subsequent commands skip download (cached)
- [ ] Documents supported platforms: macOS, Linux
- [ ] Documents future work: Windows support
- [ ] Shows expected first-run output (download progress)

**Priority:** P2

**Traces to:** ISSUE-45

---

### REQ-9: Integration Test Coverage for Trace Repair

As a projctl maintainer, I want integration tests proving trace repair works correctly, so that duplicate IDs get renumbered and dangling refs get escalated.

**Acceptance Criteria:**
- [ ] Integration test: duplicate ID in same file → auto-renumbered
- [ ] Integration test: duplicate ID across files → auto-renumbered
- [ ] Integration test: dangling reference → escalation created
- [ ] Test verifies renumbering uses next available ID
- [ ] Test verifies escalation file contains dangling ref details
- [ ] Test verifies no escalation for duplicate IDs (auto-fixed)
- [ ] Test verifies repair is idempotent (running twice produces same result)

**Priority:** P1

**Traces to:** ISSUE-45

---

## Success Metrics

1. **Functionality**: All Layer 0 commands work as documented
2. **Integration**: Memory extract → query returns expected results
3. **Parallel Safety**: Multiple context writes generate unique yield_paths
4. **Documentation**: User can implement Layer 1 using Layer 0 docs
5. **Testing**: Integration tests cover end-to-end memory and context flows

---

## Out of Scope

- Layer 1 implementation (leaf commands spawning Claude CLI)
- Layer 2+ implementation (pair loops, TDD, orchestration)
- Windows support for memory system (future work)
- GUI/TUI for projctl commands
- API server for projctl
- Performance optimization of semantic queries
- Memory system migration/export tools

---

## Dependencies

### External
- ONNX runtime (auto-downloaded on first use)
- e5-small model (~130MB, auto-downloaded on first use)
- SQLite-vec (already used in codebase)

### Internal
- Existing state management (projctl state)
- Existing context serialization (projctl context)
- Existing trace validation (projctl trace validate)
- Existing memory infrastructure (internal/memory package)

---

## Notes

- Trace repair is already correctly implemented; only documentation needed
- Memory infrastructure (embeddings, SQLite-vec) already exists
- Context write exists but needs yield_path generation enhancement
- Focus is on filling gaps, testing integration, and documenting what exists
