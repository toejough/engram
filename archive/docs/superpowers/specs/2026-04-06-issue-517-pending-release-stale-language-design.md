# Design: Fix Stale PENDING-RELEASE Language in Lead Skill (Issue #517)

## Summary

Single-phrase replacement in `skills/engram-tmux-lead/SKILL.md` line 477. The PENDING-RELEASE state table row currently says "Monitor holds via background tasks" — Phase 1 language. Phase 2 replaced background hold monitoring with a manual `engram hold check` command triggered after each agent `done` event. The stale phrase causes a lead agent reading this row to attempt non-existent Phase 1 behavior.

## Scope

One line, one phrase. No architectural changes. No other files affected.

## Stale Reference Audit

All other `background task` mentions in `skills/engram-tmux-lead/SKILL.md` are Phase 2-correct:
- Chat monitor task hygiene (Section 6.4)
- READY-check loop drain rules (Section 1.5, 2.1)
- `(no background task)` annotation at line 796 (explicit Phase 2 note, correct)

No additional stale references found.

## Change

**File:** `skills/engram-tmux-lead/SKILL.md`
**Line:** 477 (PENDING-RELEASE row, action cell)

**Before:**
```
Do NOT kill pane. Agent remains alive and responsive. Monitor holds via background tasks. Silence threshold still applies — use PENDING-RELEASE-specific nudge text (see 3.2).
```

**After:**
```
Do NOT kill pane. Agent remains alive and responsive. Run engram hold check after each agent done event. Silence threshold still applies — use PENDING-RELEASE-specific nudge text (see 3.2).
```

## Testing

No Go code changes. Verification: confirm the replacement appears correctly and no other stale references remain.
