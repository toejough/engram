package server_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"engram/internal/chat"
	"engram/internal/server"

	. "github.com/onsi/gomega"
)

func TestServer_AgentLoopDeliversMessages(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Trigger channel: each send simulates a file change notification.
	trigger := make(chan struct{}, 1)
	chatData := []byte(`
[[message]]
from = "lead"
to = "engram-agent"
thread = ""
type = ""
ts = 2026-04-13T10:00:00Z
text = """
hello agent
"""
`)

	processed := make(chan chat.Message, 1)

	cfg := server.Config{
		Addr:         "localhost:0",
		ChatFilePath: "/fake/chat.toml",
		WaitForChange: func(ctx context.Context, _ string) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-trigger:
				return nil
			}
		},
		ReadFile: func(_ string) ([]byte, error) {
			return chatData, nil
		},
		AgentProcess: func(_ context.Context, msg chat.Message) error {
			processed <- msg
			return nil
		},
		PostFunc: func(_ chat.Message) (int, error) { return 0, nil },
		WatchFunc: func(_ context.Context, _, _ string, _ int) (chat.Message, int, error) {
			return chat.Message{}, 0, nil
		},
		SubscribeFunc: func(_ context.Context, _ string, _ int) ([]chat.Message, int, error) {
			return nil, 0, nil
		},
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	_, startErr := server.Start(ctx, cfg)
	g.Expect(startErr).NotTo(HaveOccurred())

	if startErr != nil {
		return
	}

	// Trigger a file change notification.
	trigger <- struct{}{}

	// The agent loop should deliver the message to AgentProcess.
	g.Eventually(processed).WithTimeout(2 * time.Second).Should(Receive(
		WithTransform(func(m chat.Message) string { return m.Text }, ContainSubstring("hello agent")),
	))
}

func TestServer_AgentLoopProcessError_DoesNotCrash(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	trigger := make(chan struct{}, 1)
	chatData := []byte(`
[[message]]
from = "lead"
to = "engram-agent"
thread = ""
type = ""
ts = 2026-04-13T10:00:00Z
text = """
hello
"""
`)

	processCount := make(chan struct{}, 1)

	cfg := server.Config{
		Addr:         "localhost:0",
		ChatFilePath: "/fake/chat.toml",
		WaitForChange: func(ctx context.Context, _ string) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-trigger:
				return nil
			}
		},
		ReadFile: func(_ string) ([]byte, error) {
			return chatData, nil
		},
		AgentProcess: func(_ context.Context, _ chat.Message) error {
			processCount <- struct{}{}
			return errors.New("simulated failure")
		},
		PostFunc: func(_ chat.Message) (int, error) { return 0, nil },
		WatchFunc: func(_ context.Context, _, _ string, _ int) (chat.Message, int, error) {
			return chat.Message{}, 0, nil
		},
		SubscribeFunc: func(_ context.Context, _ string, _ int) ([]chat.Message, int, error) {
			return nil, 0, nil
		},
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	_, startErr := server.Start(ctx, cfg)
	g.Expect(startErr).NotTo(HaveOccurred())

	if startErr != nil {
		return
	}

	// Trigger a file change — agent process will return an error.
	trigger <- struct{}{}

	// Should still process (error is logged, not fatal).
	g.Eventually(processCount).WithTimeout(2 * time.Second).Should(Receive())
}

func TestServer_AgentLoopReadError_DoesNotCrash(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	trigger := make(chan struct{}, 1)
	readCalled := make(chan struct{}, 1)

	cfg := server.Config{
		Addr:         "localhost:0",
		ChatFilePath: "/fake/chat.toml",
		WaitForChange: func(ctx context.Context, _ string) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-trigger:
				return nil
			}
		},
		ReadFile: func(_ string) ([]byte, error) {
			readCalled <- struct{}{}
			return nil, errors.New("disk error")
		},
		AgentProcess: func(_ context.Context, _ chat.Message) error {
			return nil
		},
		PostFunc: func(_ chat.Message) (int, error) { return 0, nil },
		WatchFunc: func(_ context.Context, _, _ string, _ int) (chat.Message, int, error) {
			return chat.Message{}, 0, nil
		},
		SubscribeFunc: func(_ context.Context, _ string, _ int) ([]chat.Message, int, error) {
			return nil, 0, nil
		},
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	_, startErr := server.Start(ctx, cfg)
	g.Expect(startErr).NotTo(HaveOccurred())

	if startErr != nil {
		return
	}

	// Trigger — readMessages will fail.
	trigger <- struct{}{}

	// ReadFile should still be called (error handled gracefully).
	g.Eventually(readCalled).WithTimeout(2 * time.Second).Should(Receive())
}

func TestServer_AgentLoopRegistersInStatus(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cfg := server.Config{
		Addr:         "localhost:0",
		ChatFilePath: "/fake/chat.toml",
		WaitForChange: func(ctx context.Context, _ string) error {
			<-ctx.Done()
			return ctx.Err()
		},
		ReadFile: func(_ string) ([]byte, error) {
			return nil, nil
		},
		AgentProcess: func(_ context.Context, _ chat.Message) error {
			return nil
		},
		PostFunc: func(_ chat.Message) (int, error) { return 0, nil },
		WatchFunc: func(_ context.Context, _, _ string, _ int) (chat.Message, int, error) {
			return chat.Message{}, 0, nil
		},
		SubscribeFunc: func(_ context.Context, _ string, _ int) ([]chat.Message, int, error) {
			return nil, 0, nil
		},
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	srv, startErr := server.Start(ctx, cfg)
	g.Expect(startErr).NotTo(HaveOccurred())

	if startErr != nil {
		return
	}

	// Status should report the engram-agent.
	statusReq, reqErr := http.NewRequestWithContext(
		t.Context(), http.MethodGet, "http://"+srv.Addr()+"/status", nil,
	)
	g.Expect(reqErr).NotTo(HaveOccurred())

	if reqErr != nil {
		return
	}

	statusResp, httpErr := http.DefaultClient.Do(statusReq)
	g.Expect(httpErr).NotTo(HaveOccurred())

	if httpErr != nil {
		return
	}

	if statusResp == nil {
		return
	}

	defer func() { _ = statusResp.Body.Close() }()

	var body map[string]any
	g.Expect(json.NewDecoder(statusResp.Body).Decode(&body)).To(Succeed())

	agents, ok := body["agents"].([]any)
	g.Expect(ok).To(BeTrue())
	g.Expect(agents).To(ContainElement("engram-agent"))
}

func TestServer_ShutdownEndpointStopsServer(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cfg := server.Config{
		Addr:     "localhost:0",
		PostFunc: func(_ chat.Message) (int, error) { return 0, nil },
		WatchFunc: func(_ context.Context, _, _ string, _ int) (chat.Message, int, error) {
			return chat.Message{}, 0, nil
		},
		SubscribeFunc: func(_ context.Context, _ string, _ int) ([]chat.Message, int, error) {
			return nil, 0, nil
		},
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	srv, startErr := server.Start(ctx, cfg)
	g.Expect(startErr).NotTo(HaveOccurred())

	if startErr != nil {
		return
	}

	addr := srv.Addr()

	shutdownReq, reqErr := http.NewRequestWithContext(t.Context(), http.MethodPost, "http://"+addr+"/shutdown", nil)
	g.Expect(reqErr).NotTo(HaveOccurred())

	if reqErr != nil {
		return
	}

	shutdownResp, shutdownErr := http.DefaultClient.Do(shutdownReq)
	g.Expect(shutdownErr).NotTo(HaveOccurred())

	if shutdownErr != nil {
		return
	}

	if shutdownResp == nil {
		return
	}

	_ = shutdownResp.Body.Close()

	// Server should stop accepting connections.
	g.Eventually(func() error {
		statusReq, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://"+addr+"/status", nil)
		if err != nil {
			return err
		}

		statusResp, err := http.DefaultClient.Do(statusReq)
		if err != nil {
			return err
		}

		_ = statusResp.Body.Close()

		return nil
	}).WithTimeout(2 * time.Second).Should(HaveOccurred())
}

func TestServer_StartFailsOnBadAddr(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cfg := server.Config{
		Addr:     "999.999.999.999:0", // invalid address — listen will fail
		PostFunc: func(_ chat.Message) (int, error) { return 0, nil },
		WatchFunc: func(_ context.Context, _, _ string, _ int) (chat.Message, int, error) {
			return chat.Message{}, 0, nil
		},
		SubscribeFunc: func(_ context.Context, _ string, _ int) ([]chat.Message, int, error) {
			return nil, 0, nil
		},
	}

	_, startErr := server.Start(t.Context(), cfg)
	g.Expect(startErr).To(HaveOccurred())
}

func TestServer_StartWithNilLogger(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cfg := server.Config{
		Addr:     "localhost:0",
		Logger:   nil, // should fall back to slog.Default()
		PostFunc: func(_ chat.Message) (int, error) { return 0, nil },
		WatchFunc: func(_ context.Context, _, _ string, _ int) (chat.Message, int, error) {
			return chat.Message{}, 0, nil
		},
		SubscribeFunc: func(_ context.Context, _ string, _ int) ([]chat.Message, int, error) {
			return nil, 0, nil
		},
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	srv, startErr := server.Start(ctx, cfg)
	g.Expect(startErr).NotTo(HaveOccurred())

	if startErr != nil {
		return
	}

	g.Expect(srv.Addr()).NotTo(BeEmpty())
}

func TestServer_StatusEndpointResponds(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cfg := server.Config{
		Addr:     "localhost:0",
		PostFunc: func(_ chat.Message) (int, error) { return 0, nil },
		WatchFunc: func(_ context.Context, _, _ string, _ int) (chat.Message, int, error) {
			return chat.Message{}, 0, nil
		},
		SubscribeFunc: func(_ context.Context, _ string, _ int) ([]chat.Message, int, error) {
			return nil, 0, nil
		},
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	srv, startErr := server.Start(ctx, cfg)
	g.Expect(startErr).NotTo(HaveOccurred())

	if startErr != nil {
		return
	}

	req, reqErr := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+srv.Addr()+"/status", nil)
	g.Expect(reqErr).NotTo(HaveOccurred())

	if reqErr != nil {
		return
	}

	resp, httpErr := http.DefaultClient.Do(req)
	g.Expect(httpErr).NotTo(HaveOccurred())

	if httpErr != nil {
		return
	}

	if resp == nil {
		return
	}

	defer func() { _ = resp.Body.Close() }()

	g.Expect(resp.StatusCode).To(Equal(http.StatusOK))

	var body map[string]any
	g.Expect(json.NewDecoder(resp.Body).Decode(&body)).To(Succeed())
	g.Expect(body["running"]).To(BeTrue())
}
