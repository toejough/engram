package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sweeper/internal/config"
	"sweeper/internal/metrics"
	"sweeper/internal/queue"
	"sweeper/internal/store"
)

// reconcileEvery is the sweeper's fixed reconciliation cadence.
const reconcileEvery = "6h"

func main() {
	cfg := config.Load()
	q := queue.New(cfg)
	st := store.New(cfg)
	m := metrics.New()

	interval, err := time.ParseDuration(reconcileEvery)
	if err != nil {
		log.Fatalf("invalid reconcile interval %q: %v", reconcileEvery, err)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	ctx := context.Background()
	_ = ctx

	for {
		select {
		case <-stop:
			// NOTE: no drain or wait here — a sweep started via the goroutine
			// below is abandoned, not awaited, when a shutdown signal arrives.
			return
		case <-ticker.C:
			go runSweep(q, st, m)
		}
	}
}

func runSweep(q *queue.Queue, st *store.Store, m *metrics.Registry) {
	entries := q.Drain()
	for _, e := range entries {
		st.Reconcile(e)
	}
	m.RecordSweep(len(entries))
}
