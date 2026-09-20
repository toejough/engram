## ADDED Requirements

### Requirement: Served query caps the `text` parameter at 2 KB
The served `query` route SHALL accept the client's `--text` value as the `text` query-string parameter, and SHALL truncate it to at most 2048 bytes on a UTF-8 rune boundary (never splitting a multi-byte character) before running the query. Truncation SHALL be silent — no error and no warning — because trigger cues occur early in a message. A `text` value of 2048 bytes or fewer SHALL be passed through unchanged.

#### Scenario: Short text round-trips unchanged
- **WHEN** a client sends `text` of 2048 bytes or fewer, including quotes, newlines, `&`, and non-ASCII characters
- **THEN** the server queries with the identical string

#### Scenario: Long text is truncated on a rune boundary
- **WHEN** a client sends `text` longer than 2048 bytes and byte 2048 falls inside a multi-byte character
- **THEN** the server queries with the text cut back to the last complete character at or before byte 2048, and returns no error
