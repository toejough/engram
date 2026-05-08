package debuglog

import (
	"context"
	"crypto/rand"
	"encoding/hex"
)

type (
	phaseKey   struct{}
	cycleIDKey struct{}
)

const cycleIDBytes = 3

// WithPhase returns a copy of ctx that carries phase as a debug-log
// label (e.g. "cycle.learn", "recall.skills[brainstorming]"). The
// llmcmd Runner reads this to tag every LLM call with where it
// originated in the pipeline.
func WithPhase(ctx context.Context, phase string) context.Context {
	return context.WithValue(ctx, phaseKey{}, phase)
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

// CycleIDFromContext returns the cycle id set by WithCycleID, or
// "?" if none was set.
func CycleIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(cycleIDKey{}).(string)
	if id == "" {
		return "?"
	}

	return id
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
