# Design: path-fixes

## Overview

This is a code-level fix with no UI changes. The design is straightforward: remove the `"docs"` path segment from project artifact lookups.

## DES-001: Path Resolution Pattern

**Current pattern (broken):**
```go
filepath.Join(dir, "docs", "requirements.md")
```

**New pattern:**
```go
filepath.Join(dir, "requirements.md")
```

All project artifact lookups follow this same fix pattern.

**Traces to:** REQ-001, REQ-002, REQ-003, REQ-004

## DES-002: No Fallback/Search

We will NOT implement fallback path searching (check root, then docs/). The fix is simple and direct:

- Project artifacts live at project root
- Repo docs live at `docs/`
- No ambiguity, no search

**Traces to:** REQ-001

## DES-003: Documentation Update

Add a "Project Layout" section to the project skill documentation showing the canonical structure.

**Traces to:** REQ-005

## Files to Modify

| File | Change |
|------|--------|
| `cmd/projctl/checker.go` | Remove `"docs"` from 4 paths |
| `internal/task/deps.go` | Remove `"docs"` from 1 path |
| `internal/task/validate.go` | Remove `"docs"` from 2 paths |
| `internal/trace/trace.go` | Remove `"docs"` from 3 `docsDir` usages |
| `cmd/projctl/escalation.go` | Remove `"docs"` from 4 paths |
| `skills/project/SKILL-full.md` | Add project layout documentation |

## Files NOT to Modify

| File | Reason |
|------|--------|
| `cmd/projctl/integrate.go` | Targets repo `docs/` intentionally |
| `internal/issue/issue.go` | Repo-level, not project-level |
| `internal/territory/territory.go` | Maps repo structure |
| `internal/parser/discovery.go` | Discovers repo docs |
| `internal/id/id.go` | Generates IDs from repo docs |

**Traces to:** REQ-006
