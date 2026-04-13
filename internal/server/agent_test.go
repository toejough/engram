package server_test

import (
	"context"
	"testing"
	"time"

	"engram/internal/chat"
	"engram/internal/server"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"
)

func TestAgentLoop_AlwaysDeliversMessagesAddressedToAgent(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		agentName := rapid.StringMatching(`[a-z]{3,10}`).Draw(rt, "agent")
		text := rapid.StringMatching(`[A-Za-z0-9 ]{5,50}`).Draw(rt, "text")

		// Fake chat data with one message addressed to our agent.
		msg := chat.Message{From: "sender", To: agentName, Text: text}
		messages := []chat.Message{msg}

		notify := make(chan struct{}, 1)
		delivered := make(chan chat.Message, 1)

		loop := server.NewAgentLoop(server.AgentLoopConfig{
			Name:     agentName,
			WatchAll: false,
			Notify:   notify,
			ReadMessages: func(_ int) ([]chat.Message, int, error) {
				return messages, 1, nil
			},
			OnMessage: func(m chat.Message) { delivered <- m },
		})

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		go loop.Run(ctx)

		// Trigger notification.
		notify <- struct{}{}

		// Should receive the message.
		g.Eventually(delivered).WithTimeout(time.Second).Should(Receive(Equal(msg)))
	})
}

func TestAgentLoop_CursorAdvancesBetweenNotifications(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	callCount := 0
	cursorsReceived := make([]int, 0, 2)

	notify := make(chan struct{}, 2)
	delivered := make(chan chat.Message, 10)

	loop := server.NewAgentLoop(server.AgentLoopConfig{
		Name:     "my-agent",
		WatchAll: true,
		Notify:   notify,
		ReadMessages: func(cursor int) ([]chat.Message, int, error) {
			cursorsReceived = append(cursorsReceived, cursor)
			callCount++

			if callCount == 1 {
				return []chat.Message{
					{From: "a", To: "all", Text: "first"},
				}, 5, nil
			}

			return []chat.Message{
				{From: "b", To: "all", Text: "second"},
			}, 10, nil
		},
		OnMessage: func(m chat.Message) { delivered <- m },
	})

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	go loop.Run(ctx)

	notify <- struct{}{}

	g.Eventually(delivered).WithTimeout(time.Second).Should(Receive())

	notify <- struct{}{}

	g.Eventually(delivered).WithTimeout(time.Second).Should(Receive())

	cancel()

	// First call should start at cursor 0, second at cursor 5.
	g.Expect(cursorsReceived).To(HaveLen(2))
	g.Expect(cursorsReceived[0]).To(Equal(0))
	g.Expect(cursorsReceived[1]).To(Equal(5))
}

func TestAgentLoop_NonMatchingMessagesNotDelivered(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	msg := chat.Message{From: "a", To: "other-agent", Text: "not for me"}

	notify := make(chan struct{}, 1)
	delivered := make(chan chat.Message, 1)

	loop := server.NewAgentLoop(server.AgentLoopConfig{
		Name:     "my-agent",
		WatchAll: false,
		Notify:   notify,
		ReadMessages: func(_ int) ([]chat.Message, int, error) {
			return []chat.Message{msg}, 1, nil
		},
		OnMessage: func(m chat.Message) { delivered <- m },
	})

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	go loop.Run(ctx)

	notify <- struct{}{}

	g.Consistently(delivered).
		WithTimeout(200 * time.Millisecond).ShouldNot(Receive())
}

func TestAgentLoop_WatchAllDeliversAllMessages(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	msg1 := chat.Message{From: "a", To: "other", Text: "first"}
	msg2 := chat.Message{From: "b", To: "another", Text: "second"}
	messages := []chat.Message{msg1, msg2}

	notify := make(chan struct{}, 1)
	delivered := make(chan chat.Message, 10)

	loop := server.NewAgentLoop(server.AgentLoopConfig{
		Name:     "engram-agent",
		WatchAll: true,
		Notify:   notify,
		ReadMessages: func(_ int) ([]chat.Message, int, error) {
			return messages, 2, nil
		},
		OnMessage: func(m chat.Message) { delivered <- m },
	})

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	go loop.Run(ctx)

	notify <- struct{}{}

	g.Eventually(delivered).WithTimeout(time.Second).Should(Receive(Equal(msg1)))
	g.Eventually(delivered).WithTimeout(time.Second).Should(Receive(Equal(msg2)))
}
