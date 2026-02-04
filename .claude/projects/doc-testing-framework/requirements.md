# Requirements: Documentation Testing Framework

**Project:** doc-testing-framework
**Issue:** ISSUE-002
**Created:** 2026-02-04

## Problem Statement

Documentation is treated as "not testable" by default, but this is false. Documentation has intent and purpose that can be validated through tests, just like code. The TDD cycle (Red-Green-Refactor) applies equally to documentation.

Currently:
- TDD skills don't know how to write tests for documentation
- Doc updates are sometimes skipped from the full TDD loop
- No guidance on what "testing documentation" means in practice

## REQ-001: TDD Skills Support Documentation Testing

The TDD phase skills must treat documentation as a first-class testable artifact.

### REQ-001a: tdd-red-producer writes doc tests

When the task involves documentation changes, `tdd-red-producer` must know how to write failing tests that validate documentation intent.

**Test types for documentation:**

| Type | Description | Example |
|------|-------------|---------|
| Word/phrase matching | Specific terms must appear | `grep -q "## Acceptance Criteria" SKILL.md` |
| Semantic matching | Concepts must be conveyed | `projctl memory query --text "concept"` against doc content |
| Structural | Required sections/format | Heading hierarchy, section presence |

**Existing tooling for semantic matching:**
- `projctl memory query --text "..." --limit N` - uses ONNX (all-MiniLM-L6-v2) for embedding similarity
- Cosine similarity score returned (0.0 to 1.0)
- Threshold for "pass": score >= 0.7 (adjustable per test)

**Acceptance Criteria:**
- [ ] tdd-red-producer SKILL.md documents how to write doc tests
- [ ] Examples show word matching tests
- [ ] Examples show semantic matching tests (ONNX)
- [ ] Examples show structural tests

**Traces to:** ISSUE-002

### REQ-001b: tdd-green-producer makes doc tests pass

When doc tests are failing, `tdd-green-producer` must know how to write/edit documentation to make them pass.

**Acceptance Criteria:**
- [ ] tdd-green-producer SKILL.md covers doc editing to pass tests
- [ ] Includes at least 2 examples showing minimal doc edits that make tests pass without unnecessary additions

**Traces to:** ISSUE-002

### REQ-001c: tdd-refactor-producer refactors documentation

After doc tests pass, `tdd-refactor-producer` must know documentation refactoring best practices.

**Documentation refactoring principles:**
- Progressive disclosure (most important info first)
- Clarity and conciseness
- Consistent structure
- Remove redundancy
- Follow doc-type-specific best practices (README vs SKILL.md vs API docs)

**Acceptance Criteria:**
- [ ] tdd-refactor-producer SKILL.md covers doc refactoring
- [ ] Lists at least these documentation best practices: progressive disclosure, clarity and conciseness, consistent structure, remove redundancy, doc-type-specific practices
- [ ] Emphasizes tests must still pass after refactor

**Traces to:** ISSUE-002

## REQ-002: Orchestrator Treats Doc Tasks Like Code Tasks

The orchestrator must not skip TDD phases for documentation-focused tasks.

### REQ-002a: Doc issues get full TDD loop

When a task or issue specifies documentation work (not just incidental doc updates), the orchestrator must run the full TDD cycle.

**Indicators of doc-focused work:**
- Issue title/body mentions documentation explicitly
- Task AC include documentation deliverables
- File targets are .md files

**Acceptance Criteria:**
- [ ] Orchestrator SKILL-full.md documents this rule (canonical orchestrator documentation)
- [ ] Clear guidance on when doc work gets full TDD vs when it's incidental

**Traces to:** ISSUE-002

## Out of Scope

- `doc-producer` skill changes (future work)
- New CLI commands like `projctl docs validate`
- ISSUE-023 (validate-spec) - closed as won't do

## Success Criteria

After this project:
1. A developer working on a documentation task knows how to write tests for it
2. The TDD cycle works the same for docs as for code
3. Orchestrator doesn't skip TDD for doc-focused issues
