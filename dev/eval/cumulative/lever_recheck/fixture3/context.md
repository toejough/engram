# Conveyor compute notes (scratch log)

Compute-minutes breakdown, sampled over the last two weeks of CI runs:

- unit test shards: a minority slice, stable week over week
- the integration suite: the single largest slice by compute-minutes, and it retries
  automatically on failure — averaging 2.6 attempts per run before it passes
- the flaky failures are inside the vendored conformance suite, which is pinned upstream —
  we can't patch it, so the retry multiplier is what it is
- fixture build caching for the integration suite shipped in Q1 — the rebuild is now ~6% of
  the suite's runtime; the rest is the conformance tests themselves
- a proposal on the table: move the integration suite off the blocking lane to the
  preemptible background lane — preemptible capacity is billed at ~40% of the blocking
  lane's rate, and nothing downstream gates on the suite finishing
