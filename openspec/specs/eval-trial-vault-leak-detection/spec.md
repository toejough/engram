# eval-trial-vault-leak-detection

## Purpose

Guard eval harnesses that spawn isolated trials against a trial reaching the operator's real vault, using trial-side evidence captured automatically at env-build time rather than a whole-real-vault diff — so the orchestrating session's own legitimate vault writes during a run never register as a false leak, while a genuine leak through a registered isolation primitive is still caught and fails loud. The guard's detection limits are stated explicitly rather than implied as complete.

## Requirements

### Requirement: Vault-leak detection SHALL be trial-side, not a whole-real-vault diff, and SHALL NOT require a harness to thread trial evidence through its own call stack

Any harness that spawns isolated eval trials and guards against a trial reaching the operator's real vault SHALL detect leaks by checking trial-side evidence — the isolation state each trial's env actually carried — rather than by fingerprinting the whole real vault before and after the run and treating any change as a leak. A whole-vault diff cannot distinguish a trial's leak from the orchestrating session's own legitimate writes (route-evidence notes, recall-glance sidecars) made to the real vault while the run is in progress, and SHALL NOT be the sole detection mechanism. This trial-side evidence SHALL be captured automatically by the same primitive that builds a trial's isolated environment, at the point it builds it — a calling harness SHALL NOT be required to separately collect, return, or pass along each trial's environment or identifying markers itself, since trials are commonly built deep inside per-trial worker functions (e.g. a `ThreadPoolExecutor` worker) whose return value a harness's own `__main__` guard never sees.

#### Scenario: The orchestrating session's own vault writes do not trigger a false leak report

- **GIVEN** a trial run is in progress, using isolated per-trial environments built through the standard env-building primitive
- **WHEN** the orchestrating session itself writes a note (and its embedding sidecar) into the real vault — for example via a `/route` recall glance — while that run is in progress
- **THEN** the run completes without reporting a leak, because that note's content names no trial directory or session identifier the run's trials are known by

#### Scenario: A harness captures no trial evidence itself and still gets a correct guard

- **GIVEN** a harness builds each trial's isolated environment inside a per-trial worker function, and never returns or otherwise surfaces that environment to its own top-level run loop
- **WHEN** the harness's run completes and it invokes the leak guard
- **THEN** the guard correctly reflects every trial that ran, because each trial's environment was recorded automatically when it was built, not supplied by the harness

### Requirement: A genuine real-vault leak that goes through a registered isolation primitive SHALL still be detected and SHALL fail loud

Replacing whole-vault diffing with trial-side detection SHALL NOT weaken the guard's ability to catch an actual leak whose trial env was built through the standard isolation primitive. If a note newly present in the real vault after a run names a trial directory or session identifier belonging to a trial from that run, the guard SHALL raise, naming the offending trial and/or note so the leak is immediately diagnosable.

#### Scenario: A new real-vault note naming a trial is flagged

- **GIVEN** a run whose trials were built through the standard isolation primitive
- **WHEN** a note appears in the real vault after the run that did not exist before it, and that note's content names a trial directory or session identifier from the run
- **THEN** the guard raises, naming the note and the trial/session it references

#### Scenario: A new real-vault note naming only a distinctive trial basename is still flagged

- **GIVEN** a run whose trial marker's basename is itself distinctive (e.g. a `tempfile.mkdtemp`-style random suffix, unlikely to appear by chance in unrelated prose)
- **WHEN** a note appears in the real vault after the run whose content names only that distinctive basename, not the trial's full path
- **THEN** the guard still raises, naming the note and the matched basename

### Requirement: The leak guard's detection limits SHALL be stated, not implied as complete

The trial-side guard SHALL NOT be described or relied upon as an unconditional leak-proof. Its detection is scoped to leaks that are traceable to a registered trial marker in the leaked note's own content; it SHALL NOT be claimed to catch every possible leak path.

#### Scenario: A leak through an unmediated path is not caught

- **GIVEN** a code path that writes to the operator's real vault without ever building its environment through the standard isolation primitive (bypassing this guard's registration point entirely)
- **WHEN** that path leaks into the real vault
- **THEN** the guard does not detect it, and this is a documented limitation, not a silent gap

#### Scenario: A leak whose content does not self-identify is not caught

- **GIVEN** a trial's env was built through the standard isolation primitive, and that trial itself leaks a note into the real vault
- **WHEN** the leaked note's content does not name any registered trial directory or session identifier
- **THEN** the guard does not flag it as a leak, and this is a documented limitation, not a silent gap

#### Scenario: A same-filename in-place content mutation is not caught

- **GIVEN** the guard's fingerprint is based on the real vault's set of `.md` basenames, not on file modification times
- **WHEN** an existing real-vault note's content is mutated in place, with no change to the vault's set of note basenames
- **THEN** the guard does not flag it as a leak, because the basename-based fingerprint is identical before and after — this is a documented limitation (a capability the guard's prior mtime-based fingerprint had, dropped when the mechanism changed), accepted because a mutation of this shape is only reachable through the same unmediated code path already named above, never through a trial env built via the standard isolation primitive

#### Scenario: A leak naming only a non-distinctive basename is not caught

- **GIVEN** a trial's marker basename is short and deterministic (e.g. a fixed task/arm name plus a small index, not a random suffix) rather than distinctive
- **WHEN** a leaked note's content names only that non-distinctive basename, and never the trial's full path
- **THEN** the guard does not flag it as a leak — this is a documented limitation, accepted in exchange for not false-matching an orchestrator note that merely mentions a short, deterministic trial basename in passing (the dominant real leak shape, a note quoting the trial's full path, is unaffected)
