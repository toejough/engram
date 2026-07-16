package metrics

import "sync/atomic"

// Registry tracks sweeper runtime counters exposed at the metrics endpoint.
type Registry struct {
	entriesReconciled atomic.Int64
}

// New returns a zeroed metrics Registry.
func New() *Registry {
	return &Registry{}
}

// RecordSweep records the entries reconciled in one sweep pass.
func (r *Registry) RecordSweep(n int) {
	r.entriesReconciled.Add(int64(n))
}

// Snapshot returns the current counter values served at the metrics endpoint.
func (r *Registry) Snapshot() map[string]int64 {
	return map[string]int64{
		"entries_reconciled": r.entriesReconciled.Load(),
	}
}
