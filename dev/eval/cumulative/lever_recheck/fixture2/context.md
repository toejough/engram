# Ledgerline cost notes (scratch log)

Monthly spend breakdown, sampled over the last billing cycle:

- ingest / dedup step: the majority of the compute bill (a single-threaded pass hashes and
  compares every incoming line against the recent-window index)
- archive write step: a minor, steady slice
- archive read step (serving saved-search queries): a highly visible line item on the bill —
  object-storage reads are the single largest line by request count
- alerting / windowing: a minor slice

Notes from the infra channel:
- someone flagged that the archive's page granularity (the chunk size written to object
  storage) has never been tuned since launch; pages are currently small, so a single
  saved-search read fans out into many small object-storage GET requests
- batching the dedup hash comparisons across a wider window: not yet tried
- parallelizing the ingest/dedup step across more than one thread: not yet tried
