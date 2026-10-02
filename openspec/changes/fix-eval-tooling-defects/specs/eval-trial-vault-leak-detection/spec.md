## ADDED Requirements

### Requirement: Vault-leak detection SHALL be trial-side, not a whole-real-vault diff

Any harness that spawns isolated eval trials and guards against a trial reaching the operator's real vault SHALL detect leaks by checking trial-side evidence — each trial's recorded environment (its `ENGRAM_VAULT_PATH`, or equivalent isolation evidence the harness already captures, such as a transcript/env record) — rather than by fingerprinting the whole real vault before and after the run and treating any change as a leak. A whole-vault diff cannot distinguish a trial's leak from the orchestrating session's own legitimate writes (route-evidence notes, recall-glance sidecars) made to the real vault while the run is in progress, and SHALL NOT be the sole detection mechanism.

#### Scenario: The orchestrating session's own vault writes do not trigger a false leak report

- **WHEN** the orchestrating session itself writes a note (and its embedding sidecar) into the real vault — for example via a `/route` recall glance — while a trial run is in progress
- **THEN** the run completes without reporting a leak, because those writes carry no trial directory or session id and every trial's own recorded environment shows an isolated `ENGRAM_VAULT_PATH`

#### Scenario: Each trial's isolation is asserted directly

- **WHEN** a trial run completes
- **THEN** the guard confirms, for every trial, that its recorded `ENGRAM_VAULT_PATH` (or equivalent captured isolation evidence) was set and resolved to a path other than the operator's real vault

### Requirement: A genuine real-vault leak SHALL still be detected and SHALL fail loud

Replacing whole-vault diffing with trial-side detection SHALL NOT weaken the guard's ability to catch an actual leak. If any trial's recorded environment shows an unset or non-isolated `ENGRAM_VAULT_PATH`, or if a note newly present in the real vault after a run names a trial directory or session id from that run in its content, the guard SHALL raise, naming the offending trial and/or note so the leak is immediately diagnosable — the same failure posture the prior whole-vault-diff guards had for a real leak, preserved under the new detection method.

#### Scenario: A trial with a non-isolated environment is flagged

- **WHEN** a completed trial's recorded environment shows `ENGRAM_VAULT_PATH` unset or resolving to the operator's real vault path
- **THEN** the guard raises, identifying that trial as the source of a potential leak

#### Scenario: A new real-vault note naming a trial is flagged

- **WHEN** a note appears in the real vault after a run that did not exist before it, and that note's content names a trial directory or session id from the run
- **THEN** the guard raises, naming the note and the trial/session it references
