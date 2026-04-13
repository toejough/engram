package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"engram/internal/chat"
	"engram/internal/server"

	. "github.com/onsi/gomega"
)

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
