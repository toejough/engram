# Project: issue-028-auto-close

**Issue:** ISSUE-028 - Issue closure should be automatic when linked work completes
**Workflow:** task
**Status:** complete

## Summary

Updated `skills/project/SKILL-full.md` to replace vague prose in the "Issue Update" phase with explicit, deterministic bash commands for auto-closing linked issues.

Also closed ISSUE-030 (create issue-update-producer skill) as "won't do" - over-engineering for a one-liner.

## Changes

1. `skills/project/SKILL-full.md` - Added explicit bash snippet for auto-close
2. `docs/issues.md` - Closed ISSUE-028, ISSUE-030

## Commits

- `cb16eb0` - docs: make issue auto-close explicit in orchestrator
- `b134e5a` - fix: use macOS-compatible grep in issue auto-close
- `50b1c83` - chore: close ISSUE-028 and complete project
