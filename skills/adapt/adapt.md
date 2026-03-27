---
name: adapt
description: >
  Use when the user says "/adapt", "review adaptation proposals", "adjust
  adaptation settings", "toggle auto-apply", or wants to manage engram's
  adaptive policies. Also triggered by triage output suggesting "Run /adapt".
---

# Adapt — Manage Adaptive Policies

Review and manage engram's learned adaptation policies.

## Commands

| Action | Command |
|--------|---------|
| List all policies | `~/.claude/engram/bin/engram adapt --data-dir "$ENGRAM_DATA_DIR"` |
| Approve a proposal | `~/.claude/engram/bin/engram adapt --data-dir "$ENGRAM_DATA_DIR" --approve <id>` |
| Reject a proposal | `~/.claude/engram/bin/engram adapt --data-dir "$ENGRAM_DATA_DIR" --reject <id>` |
| Retire an active policy | `~/.claude/engram/bin/engram adapt --data-dir "$ENGRAM_DATA_DIR" --retire <id>` |

## Presentation

When the user asks to review proposals:

1. Run the status command to get all policies
2. Present pending proposals first, grouped by dimension, with rationale
3. Ask the user to approve or reject each pending proposal
4. After all pending proposals are handled, show active policy effectiveness if any have measured results
5. If any dimension has 3+ consecutive approvals, offer to toggle auto-apply

## Auto-Apply

Per-dimension auto-apply is configured in the engram config. When a user wants to toggle it, edit `$ENGRAM_DATA_DIR/../config.toml` to add or update:

```toml
[adaptation]
extraction_auto = false
surfacing_auto = true
maintenance_auto = false
```

Inform the user which dimensions are currently automatic and which are manual.
