# Conveyor compute notes (scratch log)

Compute-minutes breakdown, sampled over the last two weeks of CI runs:

- unit test shards: a minority slice, stable week over week
- the integration suite: the single largest slice by compute-minutes, and it retries
  automatically on failure — averaging 2.6 attempts per run before it passes
- the flaky failures are inside the vendored conformance suite, which is pinned upstream —
  we can't patch it, so the retry multiplier is what it is
- the vendored runner is all-or-nothing: there is no per-test rerun API, so a failed run can
  only be re-executed whole; a quarantine lane for the flaky subset was sketched, but the
  certification report requires a fresh full-suite pass on every run, so partial reruns
  don't count
- the compliance policy requires a certified full-suite pass recorded for every merged
  change — running the suite nightly or sampling merges doesn't satisfy the policy (the
  policy doesn't require the run to block the merge itself)
- fixture build caching for the integration suite shipped in Q1 — the rebuild is now ~6% of
  the suite's runtime; the rest is the conformance tests themselves
- a proposal on the table: move the integration suite off the blocking lane to the
  preemptible background lane — preemptible capacity is billed at ~40% of the blocking
  lane's rate, and nothing downstream gates on the suite finishing
