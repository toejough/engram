package queue

import "sweeper/internal/config"

// Entry is one queued item awaiting reconciliation.
type Entry struct {
	ID string
}

// Queue is the sweeper's inbound work queue.
type Queue struct {
	dsn     string
	pending []Entry
}

// New returns a Queue connected to cfg's queue DSN.
func New(cfg config.Config) *Queue {
	return &Queue{dsn: cfg.QueueDSN}
}

// Drain returns and clears all pending entries.
func (q *Queue) Drain() []Entry {
	entries := q.pending
	q.pending = nil
	return entries
}

// Enqueue adds an entry awaiting reconciliation.
func (q *Queue) Enqueue(e Entry) {
	q.pending = append(q.pending, e)
}
