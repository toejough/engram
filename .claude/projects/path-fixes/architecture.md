# Architecture: path-fixes

## Overview

This project involves simple path changes with no architectural impact. No new components, interfaces, or patterns are introduced.

## ARCH-001: No Architectural Changes

The fix is mechanical: replace `filepath.Join(dir, "docs", "file.md")` with `filepath.Join(dir, "file.md")` in specific locations.

**Rationale:** The path assumptions were never architecturally significant - they were just hardcoded values that happened to be wrong for the new project directory structure.

**Traces to:** DES-001

## ARCH-002: Preserve Separation of Concerns

Two distinct directory contexts exist:

1. **Project directory** (`.claude/projects/<name>/`) - working artifacts during project execution
2. **Repo docs directory** (`docs/`) - permanent documentation, issues, repo-level files

Commands must use the correct context:
- Preconditions, task validation, trace validation → project directory
- Issue tracking, documentation integration → repo docs

**Traces to:** DES-002, REQ-006

## Technology Decisions

None required. Existing Go filepath.Join patterns are sufficient.

## Risk Assessment

**Low risk:**
- Changes are isolated to path string construction
- No interface changes
- No behavioral changes beyond fixing the bug
- Tests will verify correct paths are used
