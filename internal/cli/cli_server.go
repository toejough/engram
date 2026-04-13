package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"engram/internal/chat"
	"engram/internal/server"
)

// unexported constants.
const (
	serverCmd = "server"
)

// unexported variables.
var (
	errServerSubcmdRequired = errors.New("server: subcommand required (up)")
)

// buildServerConfig constructs a server.Config with real I/O wired to the chat file.
func buildServerConfig(addr, chatFilePath string, logger *slog.Logger) server.Config {
	poster := newFilePoster(chatFilePath)

	return server.Config{
		Addr:   addr,
		Logger: logger,
		PostFunc: func(msg chat.Message) (int, error) {
			return poster.Post(msg)
		},
		// WatchFunc: independent watcher per long-poll request.
		WatchFunc: func(
			watchCtx context.Context,
			_ string,
			toAgent string,
			afterCursor int,
		) (chat.Message, int, error) {
			return newFileWatcher(chatFilePath).Watch(watchCtx, toAgent, afterCursor, nil)
		},
		// SubscribeFunc: independent watcher per subscribe; wraps single-message Watch.
		SubscribeFunc: func(
			subCtx context.Context,
			agent string,
			afterCursor int,
		) ([]chat.Message, int, error) {
			msg, newCursor, watchErr := newFileWatcher(chatFilePath).Watch(subCtx, agent, afterCursor, nil)
			if watchErr != nil {
				return nil, 0, watchErr
			}

			return []chat.Message{msg}, newCursor, nil
		},
	}
}

// buildServerLogger creates an slog.Logger writing JSON to stderr.
// If logFilePath is non-empty, output is also written to that file.
func buildServerLogger(stderr io.Writer, logFilePath string) (*slog.Logger, error) {
	logWriter := stderr

	if logFilePath != "" {
		logFile, openErr := os.OpenFile( //nolint:gosec // caller-supplied path
			logFilePath,
			os.O_APPEND|os.O_CREATE|os.O_WRONLY,
			chatFileMode,
		)
		if openErr != nil {
			return nil, fmt.Errorf("server up: opening log file: %w", openErr)
		}

		defer logFile.Close() //nolint:errcheck

		logWriter = io.MultiWriter(stderr, logFile)
	}

	return slog.New(slog.NewJSONHandler(logWriter, nil)), nil
}

// runAPIWithSignal wraps runAPIDispatch with a signal-cancellable context.
// Called from Run() to keep the cyclomatic complexity of Run below the lint threshold.
func runAPIWithSignal(cmd string, args []string, stdout io.Writer) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return runAPIDispatch(ctx, cmd, args, stdout)
}

// runDispatchWithSignal wraps runDispatchDispatch with a signal-cancellable context.
// Called from Run() to keep the cyclomatic complexity of Run below the lint threshold.
func runDispatchWithSignal(args []string, stdout io.Writer) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return runDispatchDispatch(ctx, args, stdout)
}

// runServerDispatch dispatches the server subcommand (currently only "up").
func runServerDispatch(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errServerSubcmdRequired
	}

	switch args[0] {
	case "up":
		return runServerUp(ctx, args[1:], stdout, stderr)
	default:
		return fmt.Errorf("%w: server %s", errUnknownCommand, args[0])
	}
}

// runServerUp is the thin wiring layer for `engram server up`.
// Parses flags, constructs real I/O, starts the server, blocks until context cancelled.
func runServerUp(ctx context.Context, args []string, _ io.Writer, stderr io.Writer) error {
	fs := newFlagSet("server up")

	var chatFilePath, logFilePath, addr string

	fs.StringVar(&chatFilePath, "chat-file", "", "chat file path")
	fs.StringVar(&logFilePath, "log-file", "", "log file path (optional, in addition to stderr)")
	fs.StringVar(&addr, "addr", defaultAPIAddr, "listen address (e.g. localhost:7932)")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("server up: %w", parseErr)
	}

	if chatFilePath == "" {
		resolved, resolveErr := deriveChatFilePath("", os.UserHomeDir, os.Getwd)
		if resolveErr != nil {
			return fmt.Errorf("server up: %w", resolveErr)
		}

		chatFilePath = resolved
	}

	logger, logErr := buildServerLogger(stderr, logFilePath)
	if logErr != nil {
		return logErr
	}

	cfg := buildServerConfig(addr, chatFilePath, logger)

	_, startErr := server.Start(ctx, cfg)
	if startErr != nil {
		return fmt.Errorf("server up: %w", startErr)
	}

	// Block until context is cancelled (Ctrl+C or POST /shutdown).
	<-ctx.Done()

	return nil
}

// runServerWithSignal wraps runServerDispatch with a signal-cancellable context.
// Called from Run() to keep the cyclomatic complexity of Run below the lint threshold.
func runServerWithSignal(args []string, stdout, stderr io.Writer) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return runServerDispatch(ctx, args, stdout, stderr)
}
