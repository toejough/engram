# vault-local-first Specification

## Purpose
Makes the removed `ENGRAM_SERVER` thin-client mode a hard error. The vault-identity behavior that was here is in `docs/changes/792_documentation-framework-adoption/792a_product-spec.md#792a2-added-vault-identity` and `docs/changes/792_documentation-framework-adoption/792b_technical-spec.md#792b5-added-vault-identity-registry`.

## Requirements
### Requirement: ENGRAM_SERVER SHALL be a hard error
When the `ENGRAM_SERVER` environment variable is set to a non-empty value, every `engram` subcommand, including `engram serve`, SHALL exit non-zero before reading or writing any vault, chunk index, or network endpoint. The error message SHALL name `ENGRAM_SERVER` as no longer supported and SHALL tell the user to set `ENGRAM_PARENT` to the same URL instead. No subcommand SHALL treat `ENGRAM_SERVER` as an alias for `ENGRAM_PARENT`, and no thin-client mode SHALL remain.

#### Scenario: A command run with ENGRAM_SERVER set fails fast
- **WHEN** `engram query --phrase x` runs with `ENGRAM_SERVER=http://host:8093`
- **THEN** it exits non-zero, its stderr says to set `ENGRAM_PARENT=http://host:8093` instead, and no vault file, chunk file, or HTTP request is touched

#### Scenario: Writes are refused too
- **WHEN** `engram learn fact ...` runs with `ENGRAM_SERVER` set
- **THEN** it exits non-zero with the same error, and no note is written locally or remotely

#### Scenario: serve is not exempt
- **WHEN** `engram serve --addr 127.0.0.1:8093` runs with `ENGRAM_SERVER` set
- **THEN** it exits non-zero with the same error, and it binds no port

#### Scenario: ENGRAM_SERVER is not honored as an alias
- **WHEN** `ENGRAM_SERVER` is set and `ENGRAM_PARENT` is not
- **THEN** no command contacts the `ENGRAM_SERVER` URL
