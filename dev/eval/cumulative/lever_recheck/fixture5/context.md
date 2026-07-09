# Relay memory notes (scratch log)

Resident-memory breakdown, sampled over the last week of daemon uptime:

- each shard's sync-state buffer holds roughly 40-60MB resident, and buffer size grows slowly
  over a shard's lifetime
- all shards currently run inside the daemon's single process, sharing one address space
- peak RSS crosses the cgroup limit during the daily bulk-resync window, when every shard's
  buffer grows at once
- an explicit post-resync buffer release shipped in v2.3 — it freed only ~10%: allocator
  fragmentation in the long-lived process keeps most freed pages resident anyway
- a shared cross-shard buffer pool was prototyped in v2.2 with the same result — fragmentation
  keeps the pages resident, and the win was negligible
- the whiteboard proposal: per-shard subprocess isolation — run each shard's sync in its own
  spawned process, and the OS reclaims everything at process exit; fragmentation stops
  mattering because the address space itself goes away
