// Package register orchestrates registration of memory entries
// into the unified instruction registry.
package register

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"engram/internal/registry"
)

// Option configures a Registrar.
type Option func(*Registrar)

// Registrar orchestrates discovery, registration, pruning, and implicit
// surfacing of memory entries.
type Registrar struct {
	registry     Registry
	surfacingLog SurfacingLogger
	now          func() time.Time
	stderr       io.Writer
}

// NewRegistrar creates a Registrar with the given registry and surfacing
// logger, applying any functional options. Defaults use real os.* functions.
func NewRegistrar(reg Registry, logger SurfacingLogger, opts ...Option) *Registrar {
	r := &Registrar{
		registry:     reg,
		surfacingLog: logger,
		now:          time.Now,
		stderr:       os.Stderr,
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

// Run executes the 3-phase registration pipeline:
// 1. Register/update entries, 2. Prune stale, 3. Record surfacing.
func (r *Registrar) Run(entries []registry.InstructionEntry) error {
	// Phase 1: Register / update.
	r.registerEntries(entries)

	// Phase 2: Prune stale non-memory entries.
	r.pruneStale(entries)

	// Phase 3: Record implicit surfacing.
	r.recordSurfacing(entries)

	return nil
}

// logErrorf writes an error message to stderr without failing the pipeline.
func (r *Registrar) logErrorf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if !strings.HasSuffix(msg, "\n") {
		msg += "\n"
	}

	_, _ = fmt.Fprintf(r.stderr, "engram: register: %s", msg)
}

// pruneStale removes non-memory entries from the registry that are not in the
// discovered set.
func (r *Registrar) pruneStale(discovered []registry.InstructionEntry) {
	// Build set of discovered IDs.
	discoveredIDs := make(map[string]struct{}, len(discovered))
	for _, entry := range discovered {
		discoveredIDs[entry.ID] = struct{}{}
	}

	existing, err := r.registry.List()
	if err != nil {
		r.logErrorf("listing for prune: %v", err)

		return
	}

	for _, entry := range existing {
		// Never prune memory entries.
		if entry.SourceType == sourceTypeMemory {
			continue
		}

		if _, found := discoveredIDs[entry.ID]; found {
			continue
		}

		removeErr := r.registry.Remove(entry.ID)
		if removeErr != nil {
			r.logErrorf("pruning %s: %v", entry.ID, removeErr)
		}
	}
}

// recordSurfacing records implicit surfacing for all discovered (always-loaded)
// entries via both registry and surfacing logger.
func (r *Registrar) recordSurfacing(discovered []registry.InstructionEntry) {
	now := r.now()

	for _, entry := range discovered {
		surfErr := r.registry.RecordSurfacing(entry.ID)
		if surfErr != nil {
			r.logErrorf("recording surfacing %s: %v", entry.ID, surfErr)
		}

		logErr := r.surfacingLog.LogSurfacing(entry.ID, "session-start", now)
		if logErr != nil {
			r.logErrorf("logging surfacing %s: %v", entry.ID, logErr)
		}
	}
}

// registerEntries registers new entries and updates changed ones.
func (r *Registrar) registerEntries(discovered []registry.InstructionEntry) {
	for _, entry := range discovered {
		existing, err := r.registry.Get(entry.ID)
		if err != nil {
			if !errors.Is(err, registry.ErrNotFound) {
				r.logErrorf("getting entry %s: %v", entry.ID, err)

				continue
			}

			// New entry — register it.
			regErr := r.registry.Register(entry)
			if regErr != nil {
				r.logErrorf("registering %s: %v", entry.ID, regErr)
			}

			continue
		}

		// Entry exists — check if content changed.
		if existing.ContentHash == entry.ContentHash {
			continue // same content, skip
		}

		// Content changed — update: preserve counters, remove + re-register.
		entry.SurfacedCount = existing.SurfacedCount
		entry.LastSurfaced = existing.LastSurfaced
		entry.Evaluations = existing.Evaluations
		entry.Absorbed = existing.Absorbed
		entry.RegisteredAt = existing.RegisteredAt

		removeErr := r.registry.Remove(entry.ID)
		if removeErr != nil {
			r.logErrorf("removing for update %s: %v", entry.ID, removeErr)

			continue
		}

		regErr := r.registry.Register(entry)
		if regErr != nil {
			r.logErrorf("re-registering %s: %v", entry.ID, regErr)
		}
	}
}

// Registry is the subset of registry.Registry needed by Registrar.
type Registry interface {
	Register(entry registry.InstructionEntry) error
	RecordSurfacing(id string) error
	Remove(id string) error
	List() ([]registry.InstructionEntry, error)
	Get(id string) (*registry.InstructionEntry, error)
}

// SurfacingLogger logs surfacing events for the evaluate pipeline.
type SurfacingLogger interface {
	LogSurfacing(memoryPath, mode string, timestamp time.Time) error
}

// WithNow injects a time provider function.
func WithNow(fn func() time.Time) Option {
	return func(r *Registrar) { r.now = fn }
}

// WithStderr injects a writer for error logging.
func WithStderr(w io.Writer) Option {
	return func(r *Registrar) { r.stderr = w }
}

// unexported constants.
const (
	sourceTypeMemory = "memory"
)
