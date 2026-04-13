package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"engram/internal/chat"
)

// Config configures the API server.
type Config struct {
	// Addr is the listen address (e.g. "localhost:8080"). Use "localhost:0" for OS-assigned port.
	Addr string

	// Logger is used for structured event logging. If nil, slog.Default() is used.
	Logger *slog.Logger

	// PostFunc writes a message to the chat file and returns the new cursor.
	PostFunc PostFunc

	// WatchFunc blocks until a message matching from/to appears after the cursor.
	WatchFunc func(ctx context.Context, from, to string, afterCursor int) (chat.Message, int, error)

	// SubscribeFunc blocks until new messages for the agent appear after cursor.
	SubscribeFunc func(ctx context.Context, agent string, afterCursor int) ([]chat.Message, int, error)
}

// Server is the running engram API server.
type Server struct {
	listener net.Listener
	logger   *slog.Logger
}

// Addr returns the server's listen address (useful when port=0 to discover the assigned port).
func (s *Server) Addr() string {
	return s.listener.Addr().String()
}

// Start creates and starts the API server. Returns when the server is listening.
// The server shuts down when ctx is cancelled or POST /shutdown is called.
func Start(ctx context.Context, cfg Config) (*Server, error) {
	ctx, cancel := context.WithCancel(ctx)

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	httpServer := buildHTTPServer(cfg, logger, cancel)

	lc := &net.ListenConfig{}

	listener, listenErr := lc.Listen(ctx, "tcp", cfg.Addr)
	if listenErr != nil {
		cancel()

		return nil, fmt.Errorf("server: listen: %w", listenErr)
	}

	srv := &Server{listener: listener, logger: logger}

	go func() {
		<-ctx.Done()

		srv.logger.Info("server shutting down")

		_ = httpServer.Close()
	}()

	go func() {
		srv.logger.Info("server started", "addr", listener.Addr().String())

		serveErr := httpServer.Serve(listener)
		if !errors.Is(serveErr, http.ErrServerClosed) {
			srv.logger.Error("server error", "err", serveErr)
		}
	}()

	return srv, nil
}

// buildHTTPServer constructs the http.Server with all routes wired.
func buildHTTPServer(cfg Config, logger *slog.Logger, cancel context.CancelFunc) *http.Server {
	deps := &Deps{
		PostMessage:       cfg.PostFunc,
		WatchForMessage:   cfg.WatchFunc,
		SubscribeMessages: cfg.SubscribeFunc,
		Logger:            logger,
		ShutdownFn:        cancel,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /message", HandlePostMessage(deps))
	mux.HandleFunc("GET /wait-for-response", HandleWaitForResponse(deps))
	mux.HandleFunc("GET /subscribe", HandleSubscribe(deps))
	mux.HandleFunc("GET /status", HandleStatus(deps))
	mux.HandleFunc("POST /shutdown", HandleShutdown(deps))

	const readHeaderTimeout = 10 * time.Second

	return &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
	}
}
