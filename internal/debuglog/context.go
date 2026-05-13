package debuglog

import (
	"context"
)

// LoggerFromContext returns the *Logger stored in ctx, or nil if none
// was set. The returned *Logger is safe to call methods on directly —
// Logger methods are nil-receiver safe.
func LoggerFromContext(ctx context.Context) *Logger {
	logger, _ := ctx.Value(loggerKey{}).(*Logger)

	return logger
}

// WithLogger returns a copy of ctx that carries logger. Package-level
// Log and Timed read this value; if absent (or nil), they no-op.
func WithLogger(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

type loggerKey struct{}
