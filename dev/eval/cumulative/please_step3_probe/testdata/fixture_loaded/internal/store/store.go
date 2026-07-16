package store

import (
	"sweeper/internal/config"
	"sweeper/internal/queue"
)

// Store is the durable backing store the sweeper reconciles queue entries into.
type Store struct {
	dsn string
}

// New returns a Store connected to cfg's store DSN.
func New(cfg config.Config) *Store {
	return &Store{dsn: cfg.StoreDSN}
}

// Reconcile persists one queue entry as reconciled.
func (s *Store) Reconcile(e queue.Entry) {
	_ = e // reconciliation write omitted in this fixture
}
