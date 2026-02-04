# Tasks: Documentation Testing Framework

**Project:** doc-testing-framework
**Created:** 2026-02-04

## TASK-001: Add documentation testing section to tdd-red-producer

Add "## Documentation Tests" section to `~/.claude/skills/tdd-red-producer/SKILL.md` per DES-001.

**Acceptance Criteria:**
- [x] Section "## Documentation Tests" exists in SKILL.md
- [x] Table shows 3 test types: word/phrase matching, semantic matching, structural
- [x] Example for word matching test (grep-based)
- [x] Example for semantic matching test (projctl memory query)
- [x] Example for structural test

**Traces to:** DES-001, REQ-001a

---

## TASK-002: Add documentation editing section to tdd-green-producer

Add "## Making Documentation Tests Pass" section to `~/.claude/skills/tdd-green-producer/SKILL.md` per DES-002.

**Acceptance Criteria:**
- [x] Section "## Making Documentation Tests Pass" exists in SKILL.md
- [x] At least 2 examples showing minimal doc edits
- [x] Principles for minimal changes documented

**Traces to:** DES-002, REQ-001b

---

## TASK-003: Add documentation refactoring section to tdd-refactor-producer

Add "## Refactoring Documentation" section to `~/.claude/skills/tdd-refactor-producer/SKILL.md` per DES-003.

**Acceptance Criteria:**
- [x] Section "## Refactoring Documentation" exists in SKILL.md
- [x] Lists 5 best practices: progressive disclosure, clarity/conciseness, consistent structure, remove redundancy, doc-type-specific
- [x] Emphasizes tests must still pass after refactoring

**Traces to:** DES-003, REQ-001c

---

## TASK-004: Add doc-focused task guidance to project orchestrator

Add guidance for documentation-focused tasks to `~/.claude/skills/project/SKILL-full.md` per DES-004.

**Acceptance Criteria:**
- [x] Section or subsection about documentation-focused tasks exists
- [x] Lists indicators of doc-focused work (issue mentions, AC targets .md files, etc.)
- [x] States that doc tasks get full TDD, not skipped
- [x] Clarifies when updates are incidental vs doc-focused

**Traces to:** DES-004, REQ-002a
