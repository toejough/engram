# Project Summary: path-fixes

## Overview

Fixed hardcoded `docs/` path assumptions throughout the projctl codebase so that project artifacts are found at the project root directory.

## Changes Made

### Config (internal/config)
- Changed default `DocsDir` from `"docs"` to `""` (empty string)
- This is the central fix that affects all path resolution

### Checker (cmd/projctl/checker.go)
- Fixed `RequirementsExist`, `RequirementsHaveIDs`, `DesignExists`, `DesignHasIDs` to look for artifacts at project root

### Task Package (internal/task)
- Fixed `deps.go` and `validate.go` to look for `tasks.md` at project root
- Updated tests to create files at root instead of `docs/` subdirectory

### Trace Package (internal/trace)
- Fixed trace validation to look for artifacts at project root
- Updated `writeArtifact` test helper to write at project root

### Escalation (cmd/projctl/escalation.go)
- Fixed all escalation commands to look for `escalations.md` at project root

### Documentation (skills/project/SKILL-full.md)
- Added "Project Layout" section documenting the canonical directory structure
- Clarified distinction between project artifacts and repo-level docs

## Issues Resolved

- ISSUE-006: projctl preconditions look in wrong location
- ISSUE-029: Artifact path assumptions need fixing

## Testing

All affected tests pass. The fix is backwards-compatible - projects can still configure `docs_dir` in their config to use a subdirectory if desired.
