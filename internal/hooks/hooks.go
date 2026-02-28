// Package hooks provides shell scripts for Claude Code hook integration.
package hooks

// PreToolUseScript returns the bash hook for surfacing memories before tool use.
func PreToolUseScript() string {
	return `#!/usr/bin/env bash
set -euo pipefail

ENGRAM_BIN="${CLAUDE_PLUGIN_ROOT}/bin/engram"
ENGRAM_DATA="${CLAUDE_PLUGIN_ROOT}/data"

# UC-2: Surface relevant memories before tool use
"$ENGRAM_BIN" surface --hook pre-tool-use --tool-input "$CLAUDE_TOOL_INPUT" --data-dir "$ENGRAM_DATA"
`
}

// SessionStartScript returns the bash hook for surfacing memories at session start.
func SessionStartScript() string {
	return `#!/usr/bin/env bash
set -euo pipefail

ENGRAM_BIN="${CLAUDE_PLUGIN_ROOT}/bin/engram"
ENGRAM_DATA="${CLAUDE_PLUGIN_ROOT}/data"

# UC-2: Surface relevant memories at session start
"$ENGRAM_BIN" surface --hook session-start --project-dir "$CLAUDE_PROJECT_DIR" --data-dir "$ENGRAM_DATA"
`
}

// StopScript returns the bash hook for extracting learnings and catching up on session stop.
func StopScript() string {
	return `#!/usr/bin/env bash
set -euo pipefail

ENGRAM_BIN="${CLAUDE_PLUGIN_ROOT}/bin/engram"
ENGRAM_DATA="${CLAUDE_PLUGIN_ROOT}/data"

# UC-1: Extract learnings from session
"$ENGRAM_BIN" extract --session "$CLAUDE_SESSION_TRANSCRIPT" --data-dir "$ENGRAM_DATA"

# UC-3: Catch up missed corrections
"$ENGRAM_BIN" catchup --session "$CLAUDE_SESSION_TRANSCRIPT" --data-dir "$ENGRAM_DATA"
`
}

// UserPromptSubmitScript returns the bash hook for correction detection and surfacing on prompt submit.
func UserPromptSubmitScript() string {
	return `#!/usr/bin/env bash
set -euo pipefail

ENGRAM_BIN="${CLAUDE_PLUGIN_ROOT}/bin/engram"
ENGRAM_DATA="${CLAUDE_PLUGIN_ROOT}/data"

# UC-3: Check for inline correction
"$ENGRAM_BIN" correct --message "$CLAUDE_USER_MESSAGE" --data-dir "$ENGRAM_DATA"

# UC-2: Surface relevant memories (after correction per DES-4)
"$ENGRAM_BIN" surface --hook user-prompt --message "$CLAUDE_USER_MESSAGE" --data-dir "$ENGRAM_DATA"
`
}
