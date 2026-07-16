# Sweeper Architecture

The sweeper polls the queue on a fixed cadence and reconciles entries older
than the cadence window into the store.

## Cadence

The sweep cadence is 6 hours: every 6h, the sweeper wakes, drains the queue,
and reconciles.

```
 [queue] --poll (6h)--> [sweeper] --reconcile--> [store]
 <!-- cadence: 6h -->
```

## Components

- `internal/queue` — inbound work queue
- `internal/store` — durable reconciliation target
- `internal/metrics` — counters exposed at the metrics endpoint
- `internal/config` — environment-sourced runtime configuration
