# Layer -1: Skill Unification Design

**Project:** Layer -1 Skill Unification
**Issue:** ISSUE-008
**Created:** 2026-02-02
**Status:** Draft

**Traces to:** REQ-1, REQ-2, REQ-3, REQ-4, REQ-5, REQ-6, REQ-8

---

## Design Decisions

### DES-1: Skill Naming Convention

**User Experience:** Users invoke skills via `/skillname` or orchestrator dispatches by name.

**Convention:**
```
<phase>-<variant>-<role>
```

| Component | Values | Examples |
|-----------|--------|----------|
| phase | pm, design, arch, breakdown, doc, tdd-red, tdd-green, tdd-refactor, alignment, retro, summary | |
| variant | interview, infer | (optional, only for pm/design/arch) |
| role | producer, qa | |

**Examples:**
- `pm-interview-producer` - PM requirements via user Q&A
- `pm-infer-producer` - PM requirements from code analysis
- `pm-qa` - PM quality gate
- `tdd-red-producer` - Write failing tests
- `tdd-red-qa` - Verify tests cover ACs

**Standalone skills (no role suffix):**
- `intake-evaluator` - Classifies request type
- `next-steps` - Suggests follow-up work
- `commit` - Commits changes

**Traces to:** REQ-2, REQ-3, REQ-4

---

### DES-2: Yield Protocol Output Location

**User Experience:** Skills write yield to a location provided by orchestrator, enabling parallel execution.

**Context provides yield path:**
```toml
[output]
yield_path = ".claude/context/pm-interview-producer-<session-id>-yield.toml"
```

**Skill reads path from context and writes there.** Skills do NOT hardcode yield paths.

**Format:** Per Section 4 of orchestration-system.md:
```toml
[yield]
type = "complete"  # or need-user-input, blocked, approved, improvement-request, escalate-phase

[payload]
# Type-specific data

[context]
# State for resumption
```

**Parallelism implications:**
- Skills are parallelism-agnostic (just write to provided path)
- Orchestrator (Layer 0+) provides unique paths with session/task IDs
- Orchestrator tracks which yield corresponds to which invocation

**Traces to:** REQ-1, REQ-5

---

### DES-3: SKILL.md File Structure

**User Experience:** Developers read SKILL.md to understand skill behavior.

**Standard sections:**

```markdown
---
name: <skill-name>
description: <one-line description>
context: fork
model: sonnet
user-invocable: true|false
---

# <Skill Title>

<One-line purpose>

## Quick Reference

| Aspect | Details |
|--------|---------|
| Role | Producer or QA |
| Input | Context TOML | <specific inputs> |
| Output | Yield TOML | <artifacts produced> |
| Yield Types | <valid yield types for this skill> |

## Process

1. GATHER: <what to read/query>
2. SYNTHESIZE: <what to analyze>
3. PRODUCE: <what to output>  # or REVIEW/RETURN for QA

## Yield Format

<Skill-specific yield payload fields>

## Guidelines

<Skill-specific rules and hints>
```

**Traces to:** REQ-1, REQ-6

---

### DES-4: Producer vs QA Skill Distinction

**User Experience:** Clear separation of concerns in skill behavior.

**Producer skills:**
- GATHER → SYNTHESIZE → PRODUCE pattern
- Create or modify artifacts
- Valid yields: `complete`, `need-user-input`, `blocked`
- Commit after producing

**QA skills:**
- REVIEW → RETURN pattern
- Read artifacts, do not modify
- Valid yields: `approved`, `improvement-request`, `escalate-phase`
- No commit (reviewing, not producing)

**Traces to:** REQ-2, REQ-6

---

### DES-5: User Invocability

**User Experience:** Some skills are user-invocable, others are orchestrator-only.

| Skill Type | User Invocable | Rationale |
|------------|----------------|-----------|
| Interview producers | Yes | User may want to run standalone |
| Infer producers | Yes | User may want to run on existing code |
| QA skills | No | Only meaningful in pair loop context |
| Support skills | Mixed | intake-evaluator: No, next-steps: Yes |
| commit | Yes | Already user-invocable |

**Traces to:** REQ-8

---

### DES-6: /project Skill Dispatch Interface

**User Experience:** `/project` invokes skills and parses their yields.

**Dispatch pattern:**
```bash
claude --skill <skill-name> --context .claude/context/<skill-name>-context.toml
```

**Parse pattern:**
```bash
projctl yield parse .claude/context/<skill-name>-yield.toml
```

**Or directly read TOML from skill stdout if skill writes to stdout.**

**Decision:** Skills write yield to file (not stdout) for reliability and debugging.

**Traces to:** REQ-8

---

## Out of Scope

- Visual/UI design (no GUI components)
- API design (skills are CLI-invoked)
- Data model design (TOML formats defined in orchestration-system.md)
- Parallel execution orchestration (Layer 0+ concern - skills just support it via provided paths)

---

## Open Questions

None - design is straightforward for skill unification.
