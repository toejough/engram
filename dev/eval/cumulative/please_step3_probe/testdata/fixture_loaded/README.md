# Sweeper Service

A background job that drains the queue and reconciles stale entries into the
store. The sweep runs every 6 hours.

See `docs/architecture.md` for the full design, `docs/runbook.md` for the
on-call procedure, and `cmd/sweeper/` for the implementation.
