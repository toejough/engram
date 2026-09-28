## ADDED Requirements

### Requirement: Exchanged notes SHALL carry a stable exchange ID
A note SHALL gain an `xid` frontmatter field, a random identifier, lazily: the first time it takes part in exchange (queued as an offer, pulled down, or received as a served offer). No migration, backfill, or `update` step SHALL stamp `xid` on existing notes. An `xid` SHALL never change once written, including across renames. A note never involved in exchange SHALL carry no `xid`.

Every frontmatter field this capability adds (`xid`, `parent` and its members, `aliases`, `offer` and its members, including `offer.path`) SHALL be omitted when empty (`omitempty`). A note that never takes part in exchange SHALL therefore serialize exactly as it did before this capability.

#### Scenario: No backfill
- **WHEN** `engram update` runs on a vault whose notes have never taken part in exchange
- **THEN** no note gains an `xid`, and no note file changes

#### Scenario: The xid survives a rename
- **WHEN** a note carrying `xid` is renamed by Luhmann reparenting
- **THEN** the renamed note carries the same `xid`

### Requirement: Exchanged notes SHALL carry vault-qualified, multi-valued parent links
A note linked to parent notes SHALL carry a nested `parent` key with these fields:
- `vault`: the parent's vault ID (capability `vault-local-first`);
- `links`: a list of `{note, via, hash}` entries, where `note` is a parent basename, `via` is `offered`, `pulled`, or `covered`, and `hash` is the exchange hash last exchanged;
- `author`: for pulled notes only, the parent note's `repo`/`user`/`vault`.

At most one link SHALL have `via` `offered` or `pulled`. That link is the note's primary link. Links SHALL count only while `parent.vault` equals the vault ID that the configured parent reports, so a hostname or address change of the same parent keeps them valid. Only these SHALL set or change `parent`: an offer receipt, a pull-down, and `engram amend --discard --into`. A note never exchanged SHALL carry no `parent` key.

#### Scenario: Offered note is linked
- **WHEN** a learned note's offer receipt names parent basename B from vault `9a1e`
- **THEN** the note carries `parent.vault: 9a1e` and a primary link `{note: B, via: offered, hash: <offered hash>}`

#### Scenario: A changed parent address keeps links valid
- **WHEN** `ENGRAM_PARENT` changes to a new hostname for the same parent vault
- **THEN** existing links still count, because the reported vault ID is unchanged

#### Scenario: A different parent vault ignores old links
- **WHEN** `ENGRAM_PARENT` points at a parent reporting a different vault ID
- **THEN** notes linked to the old vault ID are treated as unlinked

### Requirement: Renames and curation folds SHALL record aliases
A note's `aliases` list SHALL record basenames the note answers to in its own vault:
- A rename by Luhmann reparenting, or by skill adoption, SHALL append the note's old basename to its `aliases`, in the same write.
- `engram amend --target O --discard --into E` SHALL delete O and its sidecar. It SHALL add O's basename and all of O's `aliases` to E's `aliases`, without duplicates.
- The same fold SHALL merge O's parent links into E. When E already has a primary link, O's primary link SHALL become a `covered` link on E; otherwise it SHALL become E's primary link.
- The fold SHALL NOT re-embed E, and SHALL NOT queue an offer.

#### Scenario: A rename records the old name
- **WHEN** reparenting renames `1100.2026-09-27.x` to `12a.2026-09-27.x`
- **THEN** the renamed note's `aliases` includes `1100.2026-09-27.x`

#### Scenario: A fold unions aliases
- **WHEN** O's `aliases` lists A, and `engram amend --target O --discard --into E` runs
- **THEN** O and its sidecar are gone, and E's `aliases` includes both O's basename and A

#### Scenario: A fold keeps E's primary link
- **WHEN** E has a primary link to P1, and O has a primary link `via: pulled` to P2
- **THEN** E keeps P1 as its primary link and gains `{note: P2, via: covered}`

### Requirement: Exchange fields SHALL survive every frontmatter rewrite
`xid`, `parent`, `aliases`, and `offer` SHALL be preserved unchanged by every path that rewrites a note's frontmatter and is not one of the exchange paths named above. Those paths are:
- `engram amend` (every flag, including `--activate` and `--clear-pending`);
- `engram resituate`;
- identity backfill;
- Luhmann reparenting and rename-and-rewrite of wikilinks (which only append to `aliases`);
- vocab tag rewrites, clear, legacy cleanup, self-tag, and version stamp;
- scrubbing references to deleted notes;
- skill registration refresh and adopt.

`engram resituate` SHALL also preserve every other field it does not change, namely `pending`, `sources`, `tags`, `supersedes`, `vocab_version`, `issue`, and `project`, changing only `situation` and the body opener.

#### Scenario: Amend preserves the exchange fields
- **WHEN** `engram amend --target L --clear-pending` (or any content amend) rewrites a note carrying `xid`, `parent`, `aliases`, and `offer`
- **THEN** all four are unchanged in the written note

#### Scenario: Resituate preserves every untouched field
- **WHEN** `engram resituate` rewrites a pending note carrying `tags`, `sources`, `supersedes`, `parent`, and `aliases`
- **THEN** all of them, and the pending marker, are unchanged, and only `situation` and the body opener differ

#### Scenario: Identity backfill preserves the link
- **WHEN** `engram update --backfill-identity` rewrites a note carrying `parent`
- **THEN** `parent` is unchanged

### Requirement: Exchange fields SHALL NOT be settable over the served API
The served-request decoding SHALL be unable to populate `xid`, `parent`, `aliases`, `skill_hash`, `skill_key`, or `skill_source` on any note, whatever the key spelling in the request body.

#### Scenario: A remote client cannot forge a link
- **WHEN** a served `learn` request body carries `parent`, `aliases`, or `xid` values
- **THEN** the resulting pending note carries none of those values

## MODIFIED Requirements

### Requirement: Amend re-stamps identity fields on every write
`engram amend` SHALL re-detect and overwrite a note's `repo:`, `user:`, and `vault:` frontmatter fields on every call that changes content or relations, using the same detection as `engram learn` (see the three requirements above), regardless of what the note previously held. The calls that change content or relations are any content flag, `--supersedes`, and `--chunk-source`. This is unlike `source:`, `project:`, `issue:`, and `tier:`, which continue to be preserved verbatim through amend. Bookkeeping amends SHALL NOT re-stamp these fields, so an accepted offer keeps its declared author. Those amends are:
- `--activate` alone;
- `--clear-pending`;
- `--discard --into` on the target note;
- the frontmatter-only link write that records an offer receipt.

#### Scenario: Amend from a different environment than the note's origin
- **WHEN** `engram amend` with a content, `--supersedes`, or `--chunk-source` flag rewrites a note whose `repo:`, `user:`, or `vault:` values differ from the environment `amend` is currently running in
- **THEN** the amended note's `repo:`, `user:`, and `vault:` values are overwritten with the current environment's freshly detected values

#### Scenario: Amend backfills missing identity fields
- **WHEN** `engram amend` with a content, `--supersedes`, or `--chunk-source` flag rewrites a note written before this capability existed (no `repo:`, `user:`, or `vault:` frontmatter fields present)
- **THEN** the amended note gains freshly detected `repo:`, `user:`, and `vault:` fields, same as any other amend

#### Scenario: Amend does not trigger re-embed for identity-only changes
- **WHEN** `engram amend` is invoked with only `--supersedes` or `--chunk-source` (no content-changing flag), and `repo:`/`user:`/`vault:` are the only identity fields that change
- **THEN** the note's vector sidecar is not re-embedded — identity re-stamping is a provenance-only change

#### Scenario: Accepting an offer keeps its author
- **WHEN** `engram amend --target O --clear-pending` runs on a pending offer whose `user:` is alice, from an environment whose detected user is bob
- **THEN** O's `user:` is still alice
