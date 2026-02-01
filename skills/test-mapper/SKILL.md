---
name: test-mapper
description: Map tests to traceability IDs
user-invocable: false
---

# Test Mapper

Map existing tests to traceability IDs for codebase adoption.

## Quick Reference

| Aspect | Details |
|--------|---------|
| Input | Context TOML | project dir | tasks.md | architecture.md |
| Process | Discover tests | Parse existing comments | Correlate with tasks | Assign TEST-NNN | Update files |
| Chain | REQ → DES → ARCH → TASK → TEST |

## Traceability Target

Tests trace to **closest upstream**:
1. TASK (preferred - specific functionality)
2. ARCH (no task, but arch decision covers it)
3. DES (no task/arch, but design covers it)
4. REQ (fallback)

## Comment Format

```go
// TEST-001 traces: TASK-005
func TestSomething(t *testing.T) {...}
```

## Workflow

| Step | Action |
|------|--------|
| 1 | Find all *_test.go files |
| 2 | Parse existing TEST-NNN comments (preserve them) |
| 3 | Analyze test function names/descriptions |
| 4 | Match to TASK-NNN by functionality |
| 5 | Add traceability comments to untagged tests |

## Rules

- Preserve existing TEST-NNN IDs
- Match by semantic analysis, not string matching
- One test function = one TEST-NNN ID

## Result Format

`result.toml`: `[status]`, tests mapped, `[[decisions]]`

## Full Documentation

`projctl skills docs --skillname test-mapper` or see SKILL-full.md
