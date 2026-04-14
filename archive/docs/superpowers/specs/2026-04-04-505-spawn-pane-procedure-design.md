# Design: SPAWN-PANE Procedure for Two-Column Layout Enforcement (Issue #505)

## Problem

The `engram-tmux-lead` skill defines explicit two-column splitting rules in Section 1.3 but those rules are silently bypassed in four locations:

1. **Section 1.4** (spawn engram-agent): hardcodes `tmux split-window -h` + `tmux select-layout main-vertical`
2. **Section 2.1** (spawn template): hardcodes `tmux split-window -h` + `tmux select-layout main-vertical`
3. **Section 3.1 DONE state**: runs `tmux select-layout main-vertical` unconditionally on pane kill
4. **Section 3.3 Respawn**: runs `tmux select-layout main-vertical` unconditionally on kill

This causes 6+ agent panes to collapse into a single column at ~9 rows each.

Root cause: the splitting rules block in Section 1.3 is contextually scoped to "open chat tail pane setup" and is never named or referenced as a reusable procedure. Every subsequent section copy-pastes the simple single-column split instead of referencing back.

## Design

### Approach Selected: Named Procedures + Red Flag (Option C)

Two named procedures replace all inline `split-window` calls:

**`SPAWN-PANE`** — replaces every `tmux split-window` at every spawn site. Contains the full three-branch logic from Section 1.3 (check RIGHT_PANE_COUNT, split accordingly, update tracking variables). Framed with a HARD GATE: "NEVER call `tmux split-window` inline — always use SPAWN-PANE."

**`KILL-PANE`** — replaces every kill + rebalance sequence. Decrements RIGHT_PANE_COUNT, updates MIDDLE_COL_LAST_PANE or RIGHT_COL_LAST_PANE, and applies `select-layout main-vertical` only in single-column mode. Referenced by Section 3.1 DONE and Section 3.3 Respawn.

**Red Flags addition** — `tmux split-window` added to the top-of-skill Red Flags list alongside the existing "NEVER do implementation work" items.

### Alternatives Rejected

**Option B (inline reminder)**: keeps duplicated code at each call site, which can drift. Rejected.

**Option A (named procedure only, no kill fix)**: misses §3.1 and §3.3 kill paths where `select-layout main-vertical` runs unconditionally even in two-column mode. Rejected as incomplete.

## Sections to Change

| Section | Change |
|---------|--------|
| Top of skill (Red Flags) | Add: calling `tmux split-window` directly is a red flag — use SPAWN-PANE |
| Section 1.3 | Rename splitting rules block as `#### SPAWN-PANE procedure`; add HARD GATE |
| Section 1.3 | Add `#### KILL-PANE procedure` immediately after SPAWN-PANE |
| Section 1.4 | Replace inline `split-window -h` + `select-layout` with: "Use SPAWN-PANE from Section 1.3" |
| Section 2.1 | Replace inline `split-window -h` + `select-layout` with: "Use SPAWN-PANE from Section 1.3" |
| Section 3.1 DONE | Replace inline kill + `select-layout main-vertical` with: "Use KILL-PANE from Section 1.3" |
| Section 3.3 Respawn | Replace inline kill + `select-layout main-vertical` with: "Use KILL-PANE from Section 1.3" |

## SPAWN-PANE Procedure (canonical text)

```bash
#### SPAWN-PANE — use for EVERY pane creation
# HARD GATE: NEVER call tmux split-window directly elsewhere — always run these rules.
# Requires: RIGHT_PANE_COUNT, MIDDLE_COL_LAST_PANE, RIGHT_COL_LAST_PANE are set.

if [ "$RIGHT_PANE_COUNT" -lt 4 ]; then
  NEW_PANE=$(tmux split-window -h -d -P -F '#{pane_id}')
  MIDDLE_COL_LAST_PANE=$NEW_PANE
  tmux select-layout main-vertical
elif [ "$RIGHT_PANE_COUNT" -eq 4 ]; then
  NEW_PANE=$(tmux split-window -h -d -t "$MIDDLE_COL_LAST_PANE" -P -F '#{pane_id}')
  RIGHT_COL_LAST_PANE=$NEW_PANE
  # No main-vertical: manually managing multi-column layout
else
  NEW_PANE=$(tmux split-window -v -d -t "$RIGHT_COL_LAST_PANE" -P -F '#{pane_id}')
  RIGHT_COL_LAST_PANE=$NEW_PANE
fi
RIGHT_PANE_COUNT=$((RIGHT_PANE_COUNT + 1))
```

## KILL-PANE Procedure (canonical text)

```bash
#### KILL-PANE — use for EVERY pane removal
# HARD GATE: NEVER call tmux kill-pane + select-layout inline — always run these rules.
# Requires: PANE_ID of pane to kill; RIGHT_PANE_COUNT, MIDDLE_COL_LAST_PANE,
#           RIGHT_COL_LAST_PANE are set.

tmux kill-pane -t "$PANE_ID"
RIGHT_PANE_COUNT=$((RIGHT_PANE_COUNT - 1))
if [ -z "$RIGHT_COL_LAST_PANE" ]; then
  # Single-column mode: rebalance with main-vertical
  tmux select-layout main-vertical
else
  # Two-column mode: update column tracking
  # (If killed pane was in right column, update RIGHT_COL_LAST_PANE to the pane above it.
  #  If middle column, update MIDDLE_COL_LAST_PANE. Lead tracks this via its pane registry.)
  : # no rebalance — two-column layout is managed manually
fi
```

## Testing

Since this is a SKILL.md change, testing uses `superpowers:writing-skills` (mandatory):

1. **RED baseline**: write a behavior test asserting the current broken behavior — all spawns use `split-window -h` regardless of RIGHT_PANE_COUNT. Run the test to confirm it passes (RED = current behavior is confirmed broken).
2. **GREEN**: apply all SKILL.md edits (SPAWN-PANE, KILL-PANE, red flags, section replacements).
3. **Pressure test**: re-run the behavior test and confirm it now fails (behavioral change verified). Then write a positive pressure test asserting the new procedure is the only spawn path.

## Out of Scope

- Changes to the Go binary
- Changes to the chat protocol
- Changes to any other skill
