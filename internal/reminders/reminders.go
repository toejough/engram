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
		" Call /learn at task completion or to save something explicitly" +
		" (\"remember this\", \"save that for later\")." +
		" Call /recall to load previous session context."
	// System is appended to the system prompt on every LLM request.
	// Frames memory as the agent's native reasoning cycle
	// (OBSERVE → PREPARE → PLAN → ACT → LEARN) rather than an external
	// obligation. Disambiguates "memories" from in-context understanding
	// by naming engram-sourced past-session memories explicitly. The
	// "could this be considered" phrasing raises the bar for dismissing
	// a situation as not memory-worthy.
	System = "\n\n## Engram Memory Cycle\n" +
		"The memory system is only as good as its user." +
		" Engram-sourced memories from before this session that you don't" +
		" recall are inert; lessons from this session that you don't store" +
		" as engram memories for future sessions are lost." +
		" To be truly the system's memory, you have to engage with it" +
		" — every turn, not just when reminded." +
		"\n\nYour native cycle:" +
		"\n1. OBSERVE — what the user wants, what the situation is," +
		" what your own intent is." +
		"\n2. PREPARE — if this could be considered a new task," +
		" RECALL engram-sourced memories from before this session by" +
		" calling /prepare." +
		"\n3. PLAN — use the information in those memories to update" +
		" your plan." +
		"\n4. ACT — do the work." +
		"\n5. LEARN — if this could be considered the end of a task," +
		" reflect on the facts and feedback worth keeping, and STORE" +
		" them as engram memories for future sessions by calling /learn." +
		"\n\nEvery turn, before acting:" +
		" could this be considered a new task? If yes, /prepare." +
		" Could this be considered the end of a task? If yes, /learn." +
		" Stop and consider — don't just react."
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
