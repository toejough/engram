# Project: issue-036-state-init-default

**Issue:** ISSUE-036 - projctl state init should default to .claude/projects/<name>/
**Workflow:** task
**Status:** complete

## Summary

Updated `projctl state init` to default `--dir` to `.claude/projects/<name>/` when not provided. The directory is created automatically if it doesn't exist.

This prevents the common mistake of initializing projects in the wrong directory (repo root instead of project directory).

## Changes

1. `cmd/projctl/state.go` - Made `--dir` optional, defaults to `.claude/projects/<name>/`
2. `cmd/projctl/state_test.go` - Added tests for default behavior
3. `skills/project/SKILL-full.md` - Updated initialization examples

## Related

- Closes ISSUE-034 (decision: projects always live in .claude/projects/<name>/)
- Closes ISSUE-036

## Commits

- `a444c46` - test: add failing tests for state init default directory
- `e459cfd` - feat: default state init dir to .claude/projects/<name>/
- `df4be61` - docs: update SKILL-full.md init examples for default dir
