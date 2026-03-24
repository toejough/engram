# Suppress Memory Surfacing for Engram Operations — Design

**Issue:** [#369](https://github.com/toejough/engram/issues/369)
**Date:** 2026-03-23

## Problem

When the user invokes an engram skill (`/recall`, `/memory-triage`), the skill text triggers memory surfacing via the `user-prompt-submit` hook. The surfaced memories match on engram-internal keywords ("recall", "output", "verbosity") rather than the user's actual task. Similarly, Bash tool calls to `engram` subcommands during skill execution trigger pre/post-tool-use hooks, surfacing more irrelevant memories.

The hooks already skip `engram feedback` and `engram correct` Bash commands (#352), but the filter is too narrow.

## Design

Two changes, both in hook scripts. No Go code changes.

### 1. User-Prompt-Submit: Skip Engram Skill Invocations

Add an early exit in `hooks/user-prompt-submit.sh` after capturing `USER_MESSAGE` (line 29) and before the `correct` call (line 33). If the message starts with `/` followed by a name matching a directory under `$PLUGIN_ROOT/skills/`, exit immediately — no correction check, no surfacing.

```bash
# Skip surfacing for engram skill invocations (#369)
SKILL_CMD="${USER_MESSAGE%% *}"
if [[ "$SKILL_CMD" == /* ]]; then
    SKILL_NAME="${SKILL_CMD#/}"
    if [[ -d "$PLUGIN_ROOT/skills/$SKILL_NAME" ]]; then
        exit 0
    fi
fi
```

This auto-discovers skills from the directory structure. No maintenance required when skills are added or removed.

Both `correct` and `surface` are skipped because the entire message is a command to engram, not user content to learn from or surface against.

### 2. Pre/Post-Tool-Use: Widen Bash Filter to All Engram Commands

In both `hooks/pre-tool-use.sh` and `hooks/post-tool-use.sh`, replace the narrow filter:

```bash
if [[ "$BASH_CMD" == *"engram feedback"* || "$BASH_CMD" == *"engram correct"* ]]; then
```

With:

```bash
if [[ "$BASH_CMD" == *"engram "* ]]; then
```

This catches all `engram` subcommands (`surface`, `recall`, `feedback`, `correct`, `learn`, `show`, `migrate-scores`, etc.). There is no scenario where surfacing memories about an engram CLI invocation is useful — the memories would describe engram's own behavior, not the user's task.

### Unchanged

- `hooks/session-start.sh` — no tool or command context to filter on; surfacing at session start is about the project, not a specific invocation.
- Go surface pipeline — no changes; filtering happens at the hook boundary before `engram surface` is called.

## Testing

Manual verification:
1. `/recall` no longer surfaces engram-internal memories in the system reminder.
2. `/recall some query` also suppressed.
3. `/memory-triage` also suppressed.
4. Normal user messages (not starting with a skill name) still surface memories.
5. Bash calls to `engram show`, `engram learn`, etc. no longer trigger surfacing.
6. Bash calls to non-engram commands still trigger surfacing normally.
