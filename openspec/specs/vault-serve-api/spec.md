## Purpose

Lets remote environments with no local checkout or binary path use the vault over HTTP, reusing the CLI's existing command behavior, locks, and code paths rather than a parallel implementation, with writes attributed to a server-verified caller identity instead of a client-supplied one.

## Requirements

### Requirement: Served command set is fixed and explicit
`engram serve` SHALL expose exactly `query`, `query-chunks`, `show`, `show-chunk`, `activate`, `learn`, and `amend` over HTTP. `engram serve` SHALL NOT expose `ingest`, `vocab refit`, `prune`, `check`, `update`, or `resituate` — no route SHALL exist for these host-only commands.

#### Scenario: A served command is reachable over the API
- **WHEN** a request for `query`, `query-chunks`, `show`, `show-chunk`, `activate`, `learn`, or `amend` is sent to a running `engram serve` instance
- **THEN** the request is handled and produces the same result the equivalent local CLI invocation would produce

#### Scenario: A host-only command has no route
- **WHEN** a request naming `ingest`, `vocab refit`, `prune`, `check`, `update`, or `resituate` is sent to a running `engram serve` instance
- **THEN** the server returns an error and performs no vault operation

### Requirement: Bind address must be explicit
`engram serve` SHALL require an explicit, operator-configured bind address and SHALL NOT default to `0.0.0.0`.

#### Scenario: No bind address supplied
- **WHEN** `engram serve` is started without an explicit bind address
- **THEN** the server refuses to start rather than defaulting to a wildcard address

### Requirement: Served commands reuse existing code paths and locks
Every served command SHALL execute through the same implementation and the same vault locks (ADR-0013) that the local CLI already uses for that command — not a separate implementation.

#### Scenario: Concurrent local and remote writers do not lose an update
- **WHEN** a local `engram learn`/`amend` invocation and a served `learn`/`amend` request race against the same vault
- **THEN** both writes are applied without either being silently lost or corrupting vault state

### Requirement: The CLI is a transparent HTTP client when ENGRAM_SERVER is set
When the `ENGRAM_SERVER` environment variable is set, the CLI SHALL issue HTTP requests to that server for the served command set instead of touching local files, with no change to command syntax or output shape versus local-file operation.

#### Scenario: Output parity between local and remote invocation
- **WHEN** the same `query` invocation is run once against a local vault and once with `ENGRAM_SERVER` pointed at a server serving the same vault content
- **THEN** the two invocations produce byte-identical output

### Requirement: Served writes are attributed to the server-verified caller identity
For a served `learn` or `amend` request, the server SHALL stamp the note's `user:` field from the caller's Cloudflare Access-authenticated identity on the inbound request. The server SHALL NOT use a client-supplied `user:` value for this field, regardless of what the calling instance's own local detection would have produced.

#### Scenario: A client cannot spoof its identity
- **WHEN** a served `learn` or `amend` request arrives from a caller authenticated by Cloudflare Access as identity A, carrying a self-detected `user:` value naming a different identity B
- **THEN** the resulting note's `user:` field is A, never B

### Requirement: repo: is not server-overridden on served writes
For a served `learn` or `amend` request, the server SHALL resolve `repo:` the same way local `learn`/`amend` already does (client/caller-detected) — SHALL NOT substitute a server-derived value for `repo:`.

#### Scenario: repo: reflects the caller's own detection
- **WHEN** a served `learn` request arrives with a self-detected `repo:` value
- **THEN** the resulting note's `repo:` field is that self-detected value, unchanged by the server
