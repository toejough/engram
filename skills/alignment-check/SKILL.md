---
name: alignment-check
description: Verify traceability coverage and consistency across artifacts
context: fork
model: haiku
skills: ownership-rules
user-invocable: true
---

# Alignment Check

Verify artifacts are properly linked via traceability.

## Quick Reference

| Aspect | Details |
|--------|---------|
| Input | Context TOML via $ARGUMENTS | project dir | phase completed |
| Process | Scan artifacts | Run `projctl trace validate` | Interpret results | Auto-fix gaps |
| Format | Inline `**Traces to:**` fields only - NO traceability.toml |

## Artifacts to Scan

| File | IDs | Traces |
|------|-----|--------|
| docs/requirements.md | REQ-NNN | - |
| docs/design.md | DES-NNN | **Traces to:** REQ |
| docs/architecture.md | ARCH-NNN | **Traces to:** DES/REQ |
| docs/tasks.md | TASK-NNN | **Traceability:** ARCH/DES |
| *_test.go | TEST-NNN | // traces: TASK |

## Gap Fixes

| Issue | Action |
|-------|--------|
| Orphan ID | Add missing ID or fix reference |
| Unlinked ID | Add **Traces to:** field with upstream |
| Ambiguous | Add to escalation list |

## Failure Hints

| Symptom | Fix |
|---------|-----|
| No tests.md | TEST tracing is in source files, not tests.md |
| Missing traces | Edit artifact directly to add **Traces to:** field |

## Result Format

`result.toml`: `[status]` success=bool, `[[decisions]]`, `[[learnings]]`

## Full Documentation

`projctl skills docs --skillname alignment-check` or see SKILL-full.md
