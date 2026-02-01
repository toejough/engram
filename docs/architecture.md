# projctl Architecture

Architecture decisions derived from [review-2025-01.md](./review-2025-01.md). Each ARCH item represents a phase of the implementation plan.

---

### ARCH-001: Structured Result Format

**Phase:** 0
**Priority:** High
**Timeline:** This Week

Foundation for skill communication. All skills return structured result.toml with status, outputs, decisions, and learnings.

**Traces to:** REQ-001, TASK-001, TASK-002, TASK-003, TASK-013

---

### ARCH-002: CLI Completeness

**Phase:** 1
**Priority:** High
**Timeline:** This Week

Complete CLI commands referenced by skills: context, escalation, conflict, integrate.

**Traces to:** REQ-001, TASK-004, TASK-005, TASK-006, TASK-007, TASK-008

---

### ARCH-003: Model Routing

**Phase:** 2
**Priority:** Medium
**Timeline:** This Month

Automatic model selection based on task complexity. Haiku for simple, Sonnet for medium, Opus for complex.

**Traces to:** REQ-001, TASK-014, TASK-015, TASK-016, TASK-017

---

### ARCH-004: Cost Visibility

**Phase:** 3
**Priority:** Medium
**Timeline:** Next Month

Token usage tracking and budget alerts for cost optimization.

**Traces to:** REQ-001, TASK-027, TASK-028, TASK-029

---

### ARCH-005: Learning Loop

**Phase:** 4
**Priority:** Lower
**Timeline:** Next Quarter

Correction tracking and pattern detection for automatic skill improvement proposals.

**Traces to:** REQ-001, TASK-040, TASK-041, TASK-042, TASK-043, TASK-044

---

### ARCH-006: Parallel Skill Dispatch

**Phase:** 5
**Priority:** Medium
**Timeline:** Next Month

Concurrent execution of independent skills for efficiency.

**Traces to:** REQ-001, TASK-030, TASK-031, TASK-032, TASK-033

---

### ARCH-007: Background Territory Mapping

**Phase:** 6
**Priority:** Medium
**Timeline:** Next Month

Pre-exploration of codebase structure to reduce repeated discovery.

**Traces to:** REQ-001, TASK-034, TASK-035, TASK-036

---

### ARCH-008: Graceful Degradation

**Phase:** 7
**Priority:** High
**Timeline:** This Month

Error recovery and continuation with unblocked tasks when failures occur.

**Traces to:** REQ-001, TASK-018, TASK-019, TASK-020

---

### ARCH-009: LSP Integration

**Phase:** 8
**Priority:** Lower
**Timeline:** Next Quarter

LSP-backed refactoring for deterministic symbol operations.

**Traces to:** REQ-001, TASK-045, TASK-046, TASK-047

---

### ARCH-010: CLAUDE.md Migration

**Phase:** 9
**Priority:** High
**Timeline:** This Month

Critical rules from skills moved to CLAUDE.md for passive context availability.

**Traces to:** REQ-001, TASK-021, TASK-022, TASK-023

---

### ARCH-011: Skill Compression

**Phase:** 10
**Priority:** Medium
**Timeline:** Next Month

Compress skills to < 500 tokens with full docs retrievable on demand.

**Traces to:** REQ-001, TASK-037, TASK-038, TASK-039

---

### ARCH-012: Deterministic Workflow Enforcement

**Phase:** 11
**Priority:** High
**Timeline:** This Week

State machine preconditions prevent skipping workflow steps.

**Traces to:** REQ-001, TASK-009, TASK-010

---

### ARCH-013: Relentless Continuation

**Phase:** 12
**Priority:** High
**Timeline:** This Week

Orchestrator continues autonomously until all tasks complete or blocked.

**Traces to:** REQ-001, TASK-011, TASK-012, TASK-059, TASK-060, TASK-061, TASK-062, TASK-063, TASK-064, TASK-065

---

### ARCH-014: Cross-Project Memory System

**Phase:** 13
**Priority:** Medium
**Timeline:** Next Quarter

Persistent memory across projects and sessions for learnings and decisions.

**Traces to:** REQ-001, TASK-048, TASK-049, TASK-050, TASK-051, TASK-052, TASK-053

---

### ARCH-015: Visual Acceptance Criteria

**Phase:** 14
**Priority:** Medium
**Timeline:** This Month

UI verification through screenshots and visual regression detection.

**Traces to:** REQ-001, TASK-024, TASK-025, TASK-026

---

### ARCH-016: Skill Version Control

**Phase:** 15
**Priority:** High
**Timeline:** This Week

Skills versioned in projctl repo with install/status/uninstall commands.

**Traces to:** REQ-001, TASK-055, TASK-056, TASK-057, TASK-058

---

### ARCH-017: Code Cleanup

**Phase:** Housekeeping
**Priority:** Lower
**Timeline:** Next Quarter

Remove stub code and consolidate implementations.

**Traces to:** REQ-001, TASK-054

---

