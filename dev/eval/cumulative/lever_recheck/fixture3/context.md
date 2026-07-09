# Conveyor compute notes (scratch log)

Compute-minutes breakdown, sampled over the last two weeks of CI runs:

- unit test shards: a minority slice, stable week over week
- the integration suite: the single largest slice by compute-minutes, and it retries
  automatically on failure — averaging 2.6 attempts per run before it passes
- a proposal on the table: move the flaky integration suite off the visible, blocking
  pipeline into a background lane, expecting a cheaper visible CI bill — it was piloted on
  one runner group for a week and the visible-lane numbers looked promising; the pilot has
  not been extended beyond that group
- build artifact caching: not yet tried for the integration suite (it currently rebuilds its
  fixtures from scratch on every attempt)
- reducing the suite's own retry count / fixing its underlying flakiness: not yet tried
