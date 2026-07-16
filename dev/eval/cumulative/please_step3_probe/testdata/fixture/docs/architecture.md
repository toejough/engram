# Sweeper Architecture

The sweeper polls the queue on a fixed cadence and reconciles entries older than the cadence window.

## Cadence

The sweep cadence is 6 hours: every 6h, the sweeper wakes, scans, and reconciles.

```
 [queue] --poll (6h)--> [sweeper] --reconcile--> [store]
 <!-- cadence: 6h -->
```
