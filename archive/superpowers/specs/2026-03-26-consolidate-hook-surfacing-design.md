# Consolidate Hook Surfacing to Turn Boundaries

**Issue:** #384
**Date:** 2026-03-26
**Status:** Approved

## Problem

Memory surfacing fires on every tool invocation via PreToolUse and PostToolUse hooks, producing high volume with ~35% relevance. The same irrelevant memories recur across sessions regardless of tool context. Meanwhile, UserPromptSubmit and Stop hooks surface at turn boundaries with higher-fidelity semantic context (full user message and full agent transcript respectively).

Per-tool hooks add noise without meaningful signal over what turn-boundary hooks already provide.

## Decision

Remove all per-tool hooks. Consolidate memory surfacing to exactly two points:
- **UserPromptSubmit** — start of turn (user message as query)
- **Stop** — end of turn (agent transcript as query)

## Changes

### Delete (4 files)

| File | Reason |
|------|--------|
| `hooks/pre-tool-use.sh` | Memory surfacing redundant with turn-boundary hooks |
| `hooks/post-tool-use.sh` | Memory surfacing redundant; skill/command advisory is minor |
| `hooks/post-tool-use-failure.sh` | Static error advice is not engram's responsibility |
| `hooks/pre-compact.sh` | Already a no-op (comment says "#350") |

### Modify (1 file)

**`hooks/hooks.json`** — Remove entries for `PreToolUse`, `PostToolUse`, `PostToolUseFailure`, and `PreCompact`. Remaining hook types: `UserPromptSubmit`, `SessionStart`, `Stop`.

### No changes

| File | Reason |
|------|--------|
| `hooks/user-prompt-submit.sh` | Stays as-is: memory surfacing + inline correction + pending maintenance |
| `hooks/stop-surface.sh` | Stays as-is: memory surfacing from agent transcript |
| `hooks/stop.sh` | Stays as-is: async learning from transcript |
| `hooks/session-start.sh` | Stays as-is: background maintenance |

## Resulting hooks.json

```json
{
  "hooks": {
    "UserPromptSubmit": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "${CLAUDE_PLUGIN_ROOT}/hooks/user-prompt-submit.sh",
            "timeout": 30
          }
        ]
      }
    ],
    "SessionStart": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "${CLAUDE_PLUGIN_ROOT}/hooks/session-start.sh",
            "timeout": 10
          }
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "${CLAUDE_PLUGIN_ROOT}/hooks/stop-surface.sh",
            "timeout": 15
          },
          {
            "type": "command",
            "command": "${CLAUDE_PLUGIN_ROOT}/hooks/stop.sh",
            "timeout": 120,
            "async": true
          }
        ]
      }
    ]
  }
}
```

## Testing

- Verify plugin loads with reduced hooks.json (no errors on `/plugin`)
- Verify UserPromptSubmit still surfaces memories on user message
- Verify Stop hook still surfaces memories at turn end
- Verify no PreToolUse/PostToolUse system-reminder injections appear during tool use
- Confirm deleted files don't leave dangling references in Go code

## Risks

- **Loss of tool-context surfacing** — mitigated by Stop hook seeing agent's recent text which includes tool usage context
- **Loss of error advice** — mitigated by this being generic advice any model already knows; not engram's value proposition
