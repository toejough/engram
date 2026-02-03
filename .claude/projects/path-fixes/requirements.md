# Requirements: path-fixes

## Overview

Fix hardcoded `docs/` path assumptions so project artifacts are found at project directory root (`.claude/projects/<name>/`), while preserving repo-level `docs/` for documentation phase integration.

**Linked Issues:** ISSUE-006, ISSUE-029

## Requirements

### REQ-001: Precondition checker uses project root

The precondition checker (`cmd/projctl/checker.go`) must look for artifacts at the project directory root, not in a `docs/` subdirectory.

**Affected checks:**
- `RequirementsExist` - look for `requirements.md`
- `RequirementsHaveIDs` - read `requirements.md`
- `DesignExists` - look for `design.md`
- `DesignHasIDs` - read `design.md`

**Acceptance Criteria:**
- [ ] `projctl state transition --dir .claude/projects/foo --to pm-complete` succeeds when `requirements.md` is at project root
- [ ] `projctl state transition --dir .claude/projects/foo --to design-complete` succeeds when `design.md` is at project root

### REQ-002: Task commands use project root

Task-related commands must look for `tasks.md` at project directory root.

**Affected files:**
- `internal/task/deps.go` - `Parallel()` function
- `internal/task/validate.go` - `ValidateAcceptanceCriteria()` function

**Acceptance Criteria:**
- [ ] `projctl task` commands find `tasks.md` at project root
- [ ] Task validation works with project-root `tasks.md`

### REQ-003: Trace commands use project root

Trace commands must look for artifacts at project directory root when `--dir` points to a project.

**Affected files:**
- `internal/trace/trace.go` (3 locations using `docsDir`)

**Acceptance Criteria:**
- [ ] `projctl trace validate --dir .claude/projects/foo/` finds artifacts at project root
- [ ] `projctl trace repair --dir .claude/projects/foo/` works with project root artifacts

### REQ-004: Escalation commands use project root

Escalation commands must look for `escalations.md` at project directory root.

**Affected files:**
- `cmd/projctl/escalation.go` (4 locations)

**Acceptance Criteria:**
- [ ] `projctl escalation` commands find `escalations.md` at project root

### REQ-005: Document canonical project layout

Document the expected project directory structure.

**Canonical layout:**
```
.claude/projects/<name>/
├── state.toml
├── requirements.md
├── design.md
├── architecture.md
├── tasks.md
├── escalations.md
├── retro.md
└── summary.md
```

**Acceptance Criteria:**
- [ ] Project layout documented in skill docs or README

### REQ-006: Preserve documentation phase integration

The documentation phase copies/merges project artifacts TO the repo's `docs/` directory. This behavior must not change.

**Files that should NOT change:**
- `cmd/projctl/integrate.go` - merges TO `docs/`
- `internal/issue/issue.go` - issues.md is repo-level
- `internal/territory/territory.go` - maps repo structure
- `internal/parser/discovery.go` - discovers repo docs
- `internal/id/id.go` - ID generation from repo docs

**Acceptance Criteria:**
- [ ] Documentation phase still copies artifacts to repo `docs/`
- [ ] `projctl integrate` still targets repo `docs/`

## Out of Scope

- Changing where repo-level docs live (`docs/issues.md`, etc.)
- Making paths configurable (simple fix: just use project root)
- Backward compatibility with old `docs/` project structure
