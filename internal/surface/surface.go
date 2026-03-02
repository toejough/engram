package surface

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"engram/internal/audit"
	"engram/internal/store"
)

// AuditLog records audit entries for surface operations.
type AuditLog interface {
	Log(entry audit.Entry) error
}

// Config holds dependencies for the surface pipeline.
type Config struct {
	Store     Store
	Formatter Formatter
	Audit     AuditLog
}

// Formatter converts surfaced memories into formatted output.
type Formatter interface {
	FormatSurfacing(memories []store.ScoredMemory, hookType string) string
}

// Pipeline orchestrates the surface operation.
type Pipeline struct{}

// NewPipeline creates a new Pipeline with the given configuration.
func NewPipeline(cfg Config) (*Pipeline, error) {
	var missing []string
	if cfg.Store == nil {
		missing = append(missing, "Store")
	}

	if cfg.Formatter == nil {
		missing = append(missing, "Formatter")
	}

	if cfg.Audit == nil {
		missing = append(missing, "Audit")
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("%w: %s", errMissingDeps, strings.Join(missing, ", "))
	}

	return &Pipeline{}, nil
}

// Store retrieves and updates surfaced memories.
type Store interface {
	Surface(ctx context.Context, query string, k int) ([]store.ScoredMemory, error)
	IncrementSurfacing(ctx context.Context, ids []string) error
	RecordSurfacing(ctx context.Context, ids []string) error
	ClearSessionSurfacings(ctx context.Context) error
}

// Run surfaces memories for a query and returns formatted output.
func Run(
	ctx context.Context,
	st Store,
	formatter Formatter,
	auditLog AuditLog,
	hookType string,
	query string,
	budget int,
) (string, error) {
	if hookType == "session-start" {
		err := st.ClearSessionSurfacings(ctx)
		if err != nil {
			return "", fmt.Errorf("surface: clear session surfacings: %w", err)
		}
	}

	memories, err := st.Surface(ctx, query, budget)
	if err != nil {
		return "", fmt.Errorf("surface: query: %w", err)
	}

	if len(memories) == 0 {
		return "", nil
	}

	text := formatter.FormatSurfacing(memories, hookType)

	ids := make([]string, len(memories))
	for i, sm := range memories {
		ids[i] = sm.Memory.ID
	}

	err = st.IncrementSurfacing(ctx, ids)
	if err != nil {
		return "", fmt.Errorf("surface: increment: %w", err)
	}

	err = st.RecordSurfacing(ctx, ids)
	if err != nil {
		return "", fmt.Errorf("surface: record surfacing: %w", err)
	}

	err = auditLog.Log(audit.Entry{
		Operation: "surface",
		Action:    "returned",
		Fields: map[string]string{
			"hook":  hookType,
			"count": strconv.Itoa(len(memories)),
		},
	})
	if err != nil {
		return "", fmt.Errorf("surface: audit: %w", err)
	}

	return text, nil
}

// unexported variables.
var (
	errMissingDeps = errors.New("surface: missing dependencies")
)
