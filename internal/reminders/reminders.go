// Package reminders provides canonical reminder strings shared by Claude Code
// shell hooks and the OpenCode plugin. It is the single source of truth for
// what the agent is told at each lifecycle boundary.
package reminders

import (
	"errors"
	"fmt"
)

// Exported constants.
const (
	// PostTool nudges the agent after every tool execution.
	PostTool = UserPrompt
	// SessionStart announces skill availability at the start of a session.
	SessionStart = "[engram] Memory skills available." +
		" Call /prepare before starting new work." +
		" Call /learn after completing work." +
		" Call /recall to load previous session context." +
		" Call /remember to save something explicitly."
	// System is appended to the system prompt on every LLM request.
	System = "\n\n## Engram Memory Reminder\n" +
		"Use /prepare before starting new work." +
		" Use /learn after completing work to capture lessons.\n" +
		"Use /recall to load previous session context." +
		" Use /remember to save something explicitly."
	// UserPrompt nudges the agent on every user message turn.
	UserPrompt = "Important reminders from the user:" +
		" remember to call /learn at completion boundaries" +
		" (task done, bug resolved, direction change, commit)" +
		" and /prepare when starting new work." +
		" These are CRITICAL memory boundaries." +
		" If you are at one or recently completed work without calling /learn," +
		" PAUSE and CALL IT NOW." +
		" If you are at one or recently started work without calling /prepare," +
		" PAUSE and CALL IT NOW."
)

// Exported variables.
var (
	ErrUnknownKind = errors.New("unknown reminder kind")
)

// Get returns the reminder string for the given kind.
// Valid kinds: "session-start", "user-prompt", "post-tool", "system".
func Get(kind string) (string, error) {
	switch kind {
	case "session-start":
		return SessionStart, nil
	case "user-prompt":
		return UserPrompt, nil
	case "post-tool":
		return PostTool, nil
	case "system":
		return System, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnknownKind, kind)
	}
}
