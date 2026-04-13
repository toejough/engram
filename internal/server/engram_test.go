package server_test

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"engram/internal/chat"
	"engram/internal/server"

	. "github.com/onsi/gomega"
)

func TestEngramAgent_Process_CapturesSessionID(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	agent := server.NewEngramAgent(server.EngramAgentConfig{
		RunClaude: func(_ context.Context, _, _ string) (string, error) {
			return validSurfaceStreamOutput("sess-captured", "lead-1", "memory"), nil
		},
		PostToChat: func(_ chat.Message) (int, error) { return 1, nil },
		Logger:     slog.Default(),
	})

	g.Expect(agent.SessionID()).To(BeEmpty())

	processErr := agent.Process(t.Context(), chat.Message{Text: "test"})
	g.Expect(processErr).NotTo(HaveOccurred())
	g.Expect(agent.SessionID()).To(Equal("sess-captured"))
}

func TestEngramAgent_Process_LearnAction_PostsToOriginator(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var postedMsg chat.Message

	agent := server.NewEngramAgent(server.EngramAgentConfig{
		RunClaude: func(_ context.Context, _, _ string) (string, error) {
			return validLearnStreamOutput("sess-3", "lead-1", "Learned: DI rocks"), nil
		},
		PostToChat: func(msg chat.Message) (int, error) {
			postedMsg = msg

			return 1, nil
		},
		Logger: slog.Default(),
	})

	processErr := agent.Process(t.Context(), chat.Message{
		From: "lead-1", To: "engram-agent", Text: "learn this",
	})
	g.Expect(processErr).NotTo(HaveOccurred())
	g.Expect(postedMsg.To).To(Equal("lead-1"))
	g.Expect(postedMsg.From).To(Equal("engram-agent"))
}

func TestEngramAgent_Process_LogOnlyAction_PostsWithSentinelTo(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var postedMsg chat.Message

	agent := server.NewEngramAgent(server.EngramAgentConfig{
		RunClaude: func(_ context.Context, _, _ string) (string, error) {
			return validLogOnlyStreamOutput("sess-2", "Internal log note"), nil
		},
		PostToChat: func(msg chat.Message) (int, error) {
			postedMsg = msg

			return 1, nil
		},
		Logger: slog.Default(),
	})

	processErr := agent.Process(t.Context(), chat.Message{
		From: "lead-1", To: "engram-agent", Text: "log this",
	})
	g.Expect(processErr).NotTo(HaveOccurred())
	g.Expect(postedMsg.To).To(Equal("log"))
	g.Expect(postedMsg.From).To(Equal("engram-agent"))
}

func TestEngramAgent_Process_SkillRefreshEvery13Turns(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var receivedPrompts []string

	agent := server.NewEngramAgent(server.EngramAgentConfig{
		RunClaude: func(_ context.Context, prompt, _ string) (string, error) {
			receivedPrompts = append(receivedPrompts, prompt)

			return validSurfaceStreamOutput("sess-1", "lead", "mem"), nil
		},
		PostToChat: func(_ chat.Message) (int, error) { return 1, nil },
		Logger:     slog.Default(),
	})

	const totalTurns = 13

	const refreshTurn = 13

	for turn := range totalTurns {
		processErr := agent.Process(t.Context(), chat.Message{Text: "turn"})
		g.Expect(processErr).NotTo(HaveOccurred())

		_ = turn
	}

	g.Expect(receivedPrompts).To(HaveLen(totalTurns))

	for turnIdx, prompt := range receivedPrompts {
		if turnIdx == refreshTurn-1 {
			g.Expect(prompt).To(ContainSubstring("SKILL REFRESH"))
		} else {
			g.Expect(prompt).NotTo(ContainSubstring("SKILL REFRESH"))
		}
	}
}

func TestEngramAgent_Process_SurfaceAction_PostsToChat(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var postedMsg chat.Message

	agent := server.NewEngramAgent(server.EngramAgentConfig{
		RunClaude: func(_ context.Context, _, _ string) (string, error) {
			return validSurfaceStreamOutput("sess-1", "lead-1", "Memory: use DI"), nil
		},
		PostToChat: func(msg chat.Message) (int, error) {
			postedMsg = msg

			return 1, nil
		},
		Logger: slog.Default(),
	})

	processErr := agent.Process(t.Context(), chat.Message{
		From: "lead-1", To: "engram-agent", Text: "testing",
	})
	g.Expect(processErr).NotTo(HaveOccurred())
	g.Expect(postedMsg.To).To(Equal("lead-1"))
	g.Expect(postedMsg.From).To(Equal("engram-agent"))
	g.Expect(postedMsg.Text).To(ContainSubstring("Memory: use DI"))
}

func TestEngramAgent_Process_UnknownAction_LogsAndContinues(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	postCalled := false

	agent := server.NewEngramAgent(server.EngramAgentConfig{
		RunClaude: func(_ context.Context, _, _ string) (string, error) {
			return makeEngramStreamInput("sess-u", `{"action":"unknown-thing","text":"what"}`), nil
		},
		PostToChat: func(_ chat.Message) (int, error) {
			postCalled = true

			return 1, nil
		},
		Logger: slog.Default(),
	})

	processErr := agent.Process(t.Context(), chat.Message{Text: "test"})
	g.Expect(processErr).NotTo(HaveOccurred())
	g.Expect(postCalled).To(BeFalse())
}

func TestEngramAgent_ResetSession_ClearsSessionID(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	agent := server.NewEngramAgent(server.EngramAgentConfig{
		RunClaude: func(_ context.Context, _, _ string) (string, error) {
			return validSurfaceStreamOutput("sess-xyz", "lead", "mem"), nil
		},
		PostToChat: func(_ chat.Message) (int, error) { return 1, nil },
		Logger:     slog.Default(),
	})

	processErr := agent.Process(t.Context(), chat.Message{Text: "test"})
	g.Expect(processErr).NotTo(HaveOccurred())
	g.Expect(agent.SessionID()).To(Equal("sess-xyz"))

	agent.ResetSession()
	g.Expect(agent.SessionID()).To(BeEmpty())
}

func makeEngramStreamInput(sessionID, innerJSON string) string {
	systemLine := `{"type":"system","session_id":"` + sessionID + `"}`
	escapedInner := strings.ReplaceAll(innerJSON, `"`, `\"`)
	assistantLine := `{"type":"assistant","message":{"content":[` +
		`{"type":"text","text":"` + escapedInner + `"}]}}`

	return strings.Join([]string{systemLine, assistantLine}, "\n")
}

func validLearnStreamOutput(sessionID, to, text string) string {
	innerJSON := `{"action":"learn","to":"` + to + `","text":"` + text + `","saved":true}`

	return makeEngramStreamInput(sessionID, innerJSON)
}

func validLogOnlyStreamOutput(sessionID, text string) string {
	innerJSON := `{"action":"log-only","text":"` + text + `"}`

	return makeEngramStreamInput(sessionID, innerJSON)
}

// unexported functions.

func validSurfaceStreamOutput(sessionID, to, text string) string {
	innerJSON := `{"action":"surface","to":"` + to + `","text":"` + text + `"}`

	return makeEngramStreamInput(sessionID, innerJSON)
}
