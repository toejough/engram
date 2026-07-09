# Relay memory notes (scratch log)

Resident-memory breakdown, sampled over the last week of daemon uptime:

- each shard's sync-state buffer holds roughly 40-60MB resident, and buffer size grows slowly
  over a shard's lifetime (never released back to the OS once allocated, even after a burst
  subsides)
- all shards currently run inside the daemon's single process, sharing one address space
- peak RSS crosses the cgroup limit during the daily bulk-resync window, when every shard's
  buffer grows at once
- a design sketch on the whiteboard: giving each shard its own OS process would let the OS
  reclaim that shard's memory entirely whenever the process exits or is recycled, instead of
  the buffer sitting resident forever
- pooling/reusing a single shared buffer across shards instead of growing one per shard: not
  yet tried
- releasing a shard's buffer back down after the daily bulk-resync window ends, instead of
  leaving it grown: not yet tried
