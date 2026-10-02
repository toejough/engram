## MODIFIED Requirements

### Requirement: engram show SHALL print an exchanged note's exchange hash
For a note that carries `xid` (a note that has taken part in exchange), `engram show` SHALL print `# exchange_hash: <hash>` as its first output line, before the frontmatter. A note whose lines end in `\r\n` SHALL be read as its LF form for this: it gets the header when its LF form carries `xid`, and the hash SHALL be its LF form's, the hash amend's `--expect-hash` check compares; the note itself is printed verbatim. For a note without `xid`, its output SHALL be unchanged. On the local-miss parent fallback, the `# from_parent: true` label SHALL come first, followed by the parent's `show` output, which starts with the parent note's own exchange-hash line when that note carries `xid`. The served `show` route without `raw` SHALL return exactly the local `engram show` output, header included.

#### Scenario: Header on an exchanged note
- **WHEN** `engram show <basename>` runs on a pending offer carrying `xid`
- **THEN** the first line is `# exchange_hash: xh1:…`, and the note's frontmatter follows

#### Scenario: No header on an unexchanged note
- **WHEN** `engram show <basename>` runs on a note without `xid`
- **THEN** the output is byte-identical to its output before this capability

#### Scenario: Label order on the parent fallback
- **WHEN** `engram show <ref>` falls back to the parent for a note carrying `xid`
- **THEN** line 1 is `# from_parent: true` and line 2 is the parent note's `# exchange_hash:` line

#### Scenario: An offer updated after judgment is not silently accepted
- **WHEN** curation judges pending offer N at hash H1, the child's amend then updates N in place to H2, and curation runs `engram amend --target N --clear-pending --expect-hash H1`
- **THEN** the command fails, N stays pending with the H2 content, and nothing is lost

#### Scenario: The expected hash is required
- **WHEN** `engram amend --target N --discard --into E` runs on a served offer without `--expect-hash`
- **THEN** the command fails, and nothing changes

#### Scenario: Header on a CRLF note
- **WHEN** `engram show <basename>` runs on a pending offer carrying `xid` whose lines, frontmatter included, end in `\r\n`
- **THEN** the first line is `# exchange_hash: xh1:…`, the note follows byte for byte, and `engram amend --target <basename> --clear-pending --expect-hash` with that hash succeeds
