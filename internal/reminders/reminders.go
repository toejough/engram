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
	// SessionStart announces skill availability at the start of a session.
	SessionStart = "[engram] Memory skills available." +
		" Call /prepare before starting new work." +
		" Call /learn after completing work." +
		" Call /recall to load previous session context." +
		" Call /remember to save something explicitly."
	// System is appended to the system prompt on every LLM request.
	// Uses imperative language — these are requirements, not suggestions.
	System = "\n\n## Engram Memory Requirements\n" +
		"BEFORE starting any new work, switching tasks, or beginning implementation:" +
		" you MUST call /prepare to load relevant context." +
		" Do NOT skip this step even if you think you already know the context." +
		"\nAFTER completing any task, resolving a bug, or changing direction:" +
		" you MUST call /learn to capture lessons for future sessions." +
		" Lessons not captured are lost." +
		"\nThese are system requirements, not optional reminders."
	// UserPrompt nudges the agent on every user message turn.
	UserPrompt = "MEMORY REQUIREMENTS:" +
		" Before starting new work, call /prepare." +
		" After completing work, call /learn." +
		" These are not optional."
	// PostTool nudges the agent after every tool execution.
	PostTool = UserPrompt
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
