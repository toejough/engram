package debuglog

import (
	"context"
	"crypto/rand"
	"encoding/hex"
)

// CycleIDFromContext returns the cycle id set by WithCycleID, or
// "?" if none was set.
func CycleIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(cycleIDKey{}).(string)
	if id == "" {
		return "?"
	}

	return id
}

// LoggerFromContext returns the *Logger stored in ctx, or nil if none
// was set. The returned *Logger is safe to call methods on directly —
// Logger methods are nil-receiver safe.
func LoggerFromContext(ctx context.Context) *Logger {
	logger, _ := ctx.Value(loggerKey{}).(*Logger)

	return logger
}

// NewCycleID returns a short hex id suitable for tagging a single
// engram-cycle invocation. Falls back to a fixed sentinel if the
// system rng is unavailable.
func NewCycleID() string {
	buf := make([]byte, cycleIDBytes)

	_, err := rand.Read(buf)
	if err != nil {
		return "fixed"
	}

	return hex.EncodeToString(buf)
}

// PhaseFromContext returns the phase label set by WithPhase, or
// "<unspecified>" if none was set. Safe to call with any context.
func PhaseFromContext(ctx context.Context) string {
	phase, _ := ctx.Value(phaseKey{}).(string)
	if phase == "" {
		return "<unspecified>"
	}

	return phase
}

// WithCycleID returns a copy of ctx that carries a cycle-invocation
// id. Use NewCycleID to mint one at the entry point.
func WithCycleID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, cycleIDKey{}, id)
}

// WithLogger returns a copy of ctx that carries logger. Package-level
// Log and Timed read this value; if absent (or nil), they no-op.
func WithLogger(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

// WithPhase returns a copy of ctx that carries phase as a debug-log
// label (e.g. "cycle.learn", "recall.skills[brainstorming]"). The
// llmcmd Runner reads this to tag every LLM call with where it
// originated in the pipeline.
func WithPhase(ctx context.Context, phase string) context.Context {
	return context.WithValue(ctx, phaseKey{}, phase)
}

// unexported constants.
const (
	cycleIDBytes = 3
)

type cycleIDKey struct{}

type loggerKey struct{}

type phaseKey struct{}
