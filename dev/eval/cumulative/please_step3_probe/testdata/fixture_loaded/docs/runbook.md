# Sweeper Runbook

## On-call procedure

The sweeper operates on the six-hourly reconciliation window: it wakes,
drains the queue, and reconciles stale entries into the store on a
hexahourly cadence. If a sweep is skipped, the next six-hourly wake picks up
the backlog — no manual replay is needed.

## Troubleshooting

If entries are piling up in the queue faster than the hexahourly cadence
drains them, check `internal/queue` for a stuck consumer before assuming the
cadence itself needs to shrink.

## Escalation

If three consecutive six-hourly wakes fail to drain the backlog, page the
on-call engineer.
