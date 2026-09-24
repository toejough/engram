## MODIFIED Requirements

### Requirement: show and show-chunk can route a lookup to the parent
`engram show` and `engram show-chunk` SHALL accept a `--parent` flag. When
`--parent` is set and `ENGRAM_PARENT` is configured, the command SHALL
resolve the given ref against the parent vault's existing `/show` or
`/show-chunk` endpoint instead of the local vault. Additionally, when
`--parent` is NOT set, `ENGRAM_PARENT` is configured, `ENGRAM_SERVER` is not
set, and the ref is not found locally, `engram show` and `engram show-chunk`
SHALL fall back to resolving the ref against the parent and SHALL label the
output as parent-sourced.

#### Scenario: --parent routes to the configured parent
- **WHEN** `engram show <ref> --parent` (or `engram show-chunk <id>
  --parent`) runs with `ENGRAM_PARENT` set
- **THEN** the ref is resolved against the parent vault, not the local
  vault

#### Scenario: Local hit without --parent is unchanged
- **WHEN** `engram show` or `engram show-chunk` runs without `--parent` and the ref exists locally
- **THEN** the local note is returned; the parent is not contacted

#### Scenario: Local miss falls back to the parent
- **WHEN** `engram show <ref>` runs without `--parent`, `ENGRAM_PARENT` is set, `ENGRAM_SERVER` is not set, and the ref is not found locally
- **THEN** the ref is resolved against the parent and the output is labeled as parent-sourced

#### Scenario: Local miss with no parent configured is still an error
- **WHEN** `engram show <ref>` runs without `--parent`, `ENGRAM_PARENT` is not set, and the ref is not found locally
- **THEN** the command returns the same not-found error as before this capability existed

#### Scenario: --parent without ENGRAM_PARENT configured is an error
- **WHEN** `--parent` is passed but `ENGRAM_PARENT` is not set
- **THEN** the command returns an error and performs no lookup
