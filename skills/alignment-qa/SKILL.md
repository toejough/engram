---
name: alignment-qa
description: Reviews traceability validation results for completeness and accuracy
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
role: qa
phase: alignment
---

# Alignment QA Skill

Reviews traceability validation results produced by alignment-producer for quality and completeness.

## Yield Protocol

See [YIELD.md](../shared/YIELD.md) for full protocol specification.

## Workflow: REVIEW -> RETURN

This skill follows the QA-TEMPLATE pattern.

### 1. REVIEW Phase

Validate the producer's traceability report:

1. Read producer's artifact from context (`.claude/context/alignment-report.toml`)
2. Load acceptance criteria for traceability validation
3. Check each criterion:
   - All artifact files were scanned?
   - ID extraction was complete?
   - Orphan detection is accurate?
   - Unlinked detection is accurate?
   - Chain coverage calculation is correct?
   - Recommendations are actionable?
4. Compile findings

#### Traceability Checklist

- [ ] All expected artifact files were analyzed (requirements, design, architecture, tasks)
- [ ] ID extraction used correct patterns (REQ-NNN, DES-NNN, ARCH-NNN, TASK-NNN)
- [ ] Orphan IDs are truly undefined (not just in a different file)
- [ ] Unlinked IDs have no upstream or downstream connections
- [ ] Chain direction is correct (downstream traces to upstream)
- [ ] Coverage metrics are mathematically accurate
- [ ] Domain boundary violations are correctly identified
- [ ] Suggested fixes are actionable and specific

### 2. RETURN Phase

Based on REVIEW findings, yield one of:

#### `approved`

All criteria pass. Traceability validation is complete and accurate.

```toml
[yield]
type = "approved"
timestamp = 2026-02-02T12:00:00Z

[payload]
reviewed_artifact = ".claude/context/alignment-report.toml"
checklist = [
    { item = "All artifact files analyzed", passed = true },
    { item = "ID extraction complete", passed = true },
    { item = "Orphan detection accurate", passed = true },
    { item = "Unlinked detection accurate", passed = true },
    { item = "Coverage metrics correct", passed = true }
]

[context]
phase = "alignment"
role = "qa"
iteration = 1
```

#### `improvement-request`

Issues found that the producer can fix.

```toml
[yield]
type = "improvement-request"
timestamp = 2026-02-02T12:05:00Z

[payload]
from_agent = "alignment-qa"
to_agent = "alignment-producer"
iteration = 2
issues = [
    "Missing scan of docs/issues.md for ISSUE-NNN prefixes",
    "REQ-015 incorrectly flagged as orphan - exists in requirements.md line 89",
    "Coverage calculation excludes TEST-NNN traces in source files"
]

[context]
phase = "alignment"
role = "qa"
iteration = 2
max_iterations = 3
```

#### `escalate-phase`

Problem discovered that requires changes to upstream artifacts. Used when:

- **error**: Producer made a mistake that violates constraints (e.g., wrong chain direction)
- **gap**: Missing traceability that should exist based on artifact content
- **conflict**: Contradictory traces across artifacts

```toml
[yield]
type = "escalate-phase"
timestamp = 2026-02-02T12:10:00Z

[payload.escalation]
from_phase = "alignment"
to_phase = "breakdown"  # or design, arch, depending on issue
reason = "gap"  # error | gap | conflict

[payload.issue]
summary = "TASK-023 missing traceability to ARCH-005"
context = "Task implements caching but has no trace to caching architecture decision"

[[payload.proposed_changes.tasks]]
action = "update"
id = "TASK-023"
file = "docs/tasks.md"
change = "Add **Traceability:** ARCH-005"

[context]
phase = "alignment"
role = "qa"
escalating = true
```

#### `escalate-user`

Cannot resolve issue without user input.

```toml
[yield]
type = "escalate-user"
timestamp = 2026-02-02T12:15:00Z

[payload]
reason = "Ambiguous traceability relationship"
context = "TASK-015 could trace to either ARCH-003 or ARCH-007"
question = "Which architecture decision does TASK-015 implement?"
options = ["ARCH-003 (caching layer)", "ARCH-007 (data persistence)", "Both"]

[context]
phase = "alignment"
role = "qa"
escalating = true
```

## Iteration Limits

QA tracks iterations to prevent infinite loops:

```toml
[context]
iteration = 2
max_iterations = 3
```

After max iterations:
1. Yield `escalate-user` if issues remain unresolved
2. Or yield `approved` with caveats noted in payload

## Quality Criteria

Traceability validation must be:

1. **Complete**: All artifact types scanned
2. **Accurate**: No false positives/negatives in orphan/unlinked detection
3. **Directional**: Chain direction correctly enforced (downstream traces to upstream)
4. **Actionable**: Issues include specific file locations and fix suggestions
5. **Measurable**: Coverage metrics are mathematically correct

## Common Issues to Check

| Issue | Symptom | Resolution |
|-------|---------|------------|
| False orphan | ID exists but not found | Check all artifact files, not just primary |
| False unlinked | ID has traces but not detected | Verify trace format (`**Traces to:**` vs `**Traceability:**`) |
| Wrong direction | DES traces to ARCH | Fix trace to point upstream (DES should trace to REQ) |
| Missing file | Artifact file not scanned | Ensure all docs/*.md files are included |
| TEST traces | Source file traces not found | Check `// traces:` comments in `*_test.go` files |
