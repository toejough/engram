package cli

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// signalContext returns a context that cancels on SIGINT or SIGTERM.
// Callers should defer the returned cancel function.
func signalContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}
