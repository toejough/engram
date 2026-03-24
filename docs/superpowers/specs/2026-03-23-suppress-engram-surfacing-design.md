# Suppress Memory Surfacing for Engram Operations — Design

**Issue:** [#369](https://github.com/toejough/engram/issues/369)
**Date:** 2026-03-23

## Problem

When the user invokes an engram skill (`/recall`, `/memory-triage`), the skill text triggers memory surfacing via the `user-prompt-submit` hook. The surfaced memories match on engram-internal keywords ("recall", "output", "verbosity") rather than the user's actual task. Similarly, Bash tool calls to `engram` subcommands during skill execution trigger pre/post-tool-use hooks, surfacing more irrelevant memories.

The hooks already skip `engram feedback` and `engram correct` Bash commands (#352), but the filter is too narrow.

## Design

Two changes, both in hook scripts. No Go code changes.

### 1. User-Prompt-Submit: Skip Engram Skill Invocations

Add an early exit in `hooks/user-prompt-submit.sh` after capturing `USER_MESSAGE` and before the `CORRECT_ARGS=` line. If the message starts with `/` followed by a name matching a directory under `$PLUGIN_ROOT/skills/`, exit immediately — no correction check, no surfacing.

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

Both `correct` and `surface` are skipped because the entire message is a command to engram, not user content to learn from or surface against. This is a design constraint: entries in the `skills/` directory are treated as pure commands with no user-content semantics.

Messages starting with `/` that don't match a skill directory (e.g., `/something-else`) pass through normally — the directory-existence check prevents false suppression.

Leading whitespace is not a concern: Claude Code trims user prompts before passing them to hooks.

### 2. Pre/Post-Tool-Use: Widen Bash Filter to All Engram Commands

In both `hooks/pre-tool-use.sh` and `hooks/post-tool-use.sh`, replace the narrow filter:

```bash
if [[ "$BASH_CMD" == *"engram feedback"* || "$BASH_CMD" == *"engram correct"* ]]; then
```

With a match against `$ENGRAM_BIN` (already in scope in both hooks):

```bash
if [[ "$BASH_CMD" == *"$ENGRAM_BIN"* ]]; then
```

This catches all engram CLI invocations by matching the full binary path (`~/.claude/engram/bin/engram`), avoiding false positives on commands that happen to contain the word "engram" (e.g., `grep "engram" some-file`). All hook-invoked engram calls use `$ENGRAM_BIN`, so the path match is reliable.

### Unchanged

- `hooks/session-start.sh` — no tool or command context to filter on; surfacing at session start is about the project, not a specific invocation.
- Go surface pipeline — no changes; filtering happens at the hook boundary before `engram surface` is called.

## Testing

Manual verification:

**Positive cases (suppressed):**
1. `/recall` no longer surfaces engram-internal memories in the system reminder.
2. `/recall some query` also suppressed.
3. `/memory-triage` also suppressed.
4. Bash calls to `engram show`, `engram learn`, etc. no longer trigger surfacing.

**Negative cases (still surface normally):**
5. Normal user messages (not starting with a skill name) still surface memories.
6. Bash calls to non-engram commands still trigger surfacing normally.
7. `/something-not-a-skill` still surfaces memories (no matching skill directory).
8. `grep "engram" hooks/pre-tool-use.sh` still surfaces memories (not an engram binary invocation).
9. Session-start surfacing is unaffected.
