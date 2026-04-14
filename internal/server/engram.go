package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"engram/internal/chat"
)

// EngramAgent manages the engram-agent's claude -p session.
type EngramAgent struct {
	config    EngramAgentConfig
	sessionID string
	refresh   *RefreshTracker
}

// NewEngramAgent creates an EngramAgent with the given config.
func NewEngramAgent(config EngramAgentConfig) *EngramAgent {
	return &EngramAgent{
		config:  config,
		refresh: NewRefreshTracker(SkillRefreshInterval),
	}
}

// Process invokes the engram-agent with the given message and routes the response.
func (e *EngramAgent) Process(ctx context.Context, msg chat.Message) error {
	newSession := e.sessionID == ""
	prompt := buildPrompt(msg.Text, newSession, e.refresh.ShouldRefresh())

	output, runErr := e.config.RunClaude(ctx, prompt, e.sessionID)
	if runErr != nil {
		return fmt.Errorf("engram-agent: invoking claude: %w", runErr)
	}

	resp, parseErr := ParseStreamResponse(strings.NewReader(output))
	if parseErr != nil {
		return fmt.Errorf("engram-agent: parsing response: %w", parseErr)
	}

	if resp.SessionID != "" {
		e.sessionID = resp.SessionID
	}

	return e.routeResponse(resp)
}

// ProcessWithRecovery invokes Process with the unified error recovery protocol.
// Retries on same session (3x) → reset → retries on fresh session (3x) → escalate.
func (e *EngramAgent) ProcessWithRecovery(ctx context.Context, msg chat.Message) error {
	for sessionAttempt := range maxSessionResets + 1 {
		for retry := range maxRetriesPerSession {
			err := e.Process(ctx, msg)
			if err == nil {
				return nil
			}

			e.config.Logger.Warn("engram-agent error",
				"err", err,
				"retry", retry+1,
				"session_attempt", sessionAttempt,
				"session_id", e.sessionID,
			)
		}

		if sessionAttempt < maxSessionResets {
			e.ResetSession()
		}
	}

	return e.escalate(msg)
}

// ResetSession clears the session ID, forcing a fresh session on next invocation.
func (e *EngramAgent) ResetSession() {
	e.config.Logger.Info("engram-agent session reset", "old_session_id", e.sessionID)
	e.sessionID = ""
}

// SessionID returns the current session ID (empty if not yet invoked).
func (e *EngramAgent) SessionID() string { return e.sessionID }

func (e *EngramAgent) escalate(msg chat.Message) error {
	totalAttempts := (maxSessionResets + 1) * maxRetriesPerSession

	errMsg := fmt.Sprintf(
		"engram-agent cannot produce valid output after %d attempts. "+
			"The skill/server contract needs manual intervention. Last input: %s",
		totalAttempts, msg.Text,
	)

	_, postErr := e.config.PostToChat(chat.Message{
		From: "engram-server", To: "lead", Text: errMsg,
	})
	if postErr != nil {
		e.config.Logger.Error("failed to post escalation", "err", postErr)
	}

	e.config.Logger.Error("engram-agent escalation", "msg", errMsg)

	return fmt.Errorf("engram-agent: %s: %w", errMsg, errEscalationTriggered)
}

// postAndLog posts a message to chat and logs the event.
func (e *EngramAgent) postAndLog(to, text, action string, logArgs ...any) error {
	_, postErr := e.config.PostToChat(chat.Message{
		From: "engram-agent", To: to, Text: text,
	})
	if postErr != nil {
		return fmt.Errorf("engram-agent: posting %s: %w", action, postErr)
	}

	logFields := append([]any{"action", action}, logArgs...)
	e.config.Logger.Info("engram-agent responded", logFields...)

	return nil
}

// routeResponse dispatches the parsed response to the appropriate chat destination.
func (e *EngramAgent) routeResponse(resp AgentResponse) error {
	switch resp.Action {
	case "surface":
		return e.postAndLog(resp.To, resp.Text, "surface", "to", resp.To, "text_len", len(resp.Text))
	case "log-only":
		return e.postAndLog("log", resp.Text, "log-only")
	case "learn":
		return e.postAndLog(
			resp.To, resp.Text, "learn",
			"saved", resp.Saved, "path", resp.Path,
		)
	default:
		e.config.Logger.Warn("engram-agent: unknown action", "action", resp.Action)

		return nil
	}
}

// EngramAgentConfig configures the engram-agent lifecycle manager.
type EngramAgentConfig struct {
	RunClaude  RunClaudeFunc
	PostToChat PostFunc
	Logger     *slog.Logger
}

// RunClaudeFunc invokes claude -p and returns the raw stdout as a string.
// prompt is the input text. sessionID is empty on first invocation, non-empty for --resume.
type RunClaudeFunc func(ctx context.Context, prompt, sessionID string) (string, error)

// unexported constants.
const (
	maxRetriesPerSession = 3
	maxSessionResets     = 1
	newSessionDirective  = "You are the engram-agent, a memory specialist. " +
		"Load /use-engram-chat-as and /engram-agent. " +
		"You MUST respond with a single JSON object on one line. Valid actions:\n" +
		`{"action":"surface","to":"<agent>","text":"<memory>"}` + "\n" +
		`{"action":"log-only","text":"<note>"}` + "\n" +
		`{"action":"learn","saved":true,"path":"<slug>","to":"<agent>","text":"<outcome>"}` + "\n" +
		`{"action":"learn","saved":false,"to":"<agent>","text":"<reason>"}` + "\n\n" +
		"Process this message:\n\n"
	skillRefreshDirective = "SKILL REFRESH: Reload /use-engram-chat-as and /engram-agent. " +
		"Respond with a single JSON object.\n\n"
)

// unexported variables.
var (
	errEscalationTriggered = errors.New("engram-agent escalation triggered")
)

// buildPrompt constructs the prompt, prepending skill-loading directives as needed.
// On a new session, the agent needs to load its skills before processing.
// On a refresh cycle, the agent reloads skills to pick up updates.
func buildPrompt(text string, newSession, shouldRefresh bool) string {
	if newSession {
		return newSessionDirective + text
	}

	if shouldRefresh {
		return skillRefreshDirective + text
	}

	return text
}
