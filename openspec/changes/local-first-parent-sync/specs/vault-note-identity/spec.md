## ADDED Requirements

### Requirement: Exchanged notes SHALL carry a URL-qualified parent link
A note that has been exchanged with a parent vault SHALL carry a single nested `parent` frontmatter key. It SHALL have these fields:
- `url`: the configured `ENGRAM_PARENT`, normalized (lower-case scheme and host, no trailing slash);
- `note`: the counterpart's basename in that parent;
- `via`: `offered` (local to parent) or `pulled` (parent to local);
- `hash`: the content hash of the version last exchanged.

Only three things SHALL set or change it:
- an offer receipt (capability `vault-parent-offers`);
- a pull-down (capability `vault-parent-pulldown`);
- `engram amend --discard --into`, when the target has no link.

A link SHALL count for dedupe and offer targeting only while its `url` equals the currently configured, normalized `ENGRAM_PARENT`. A note never exchanged SHALL carry no `parent` key.

#### Scenario: Offered note is linked
- **WHEN** a learned note's offer receipt names parent basename B
- **THEN** the note carries `parent: {url: <configured>, note: B, via: offered, hash: <offered hash>}`

#### Scenario: Repointed parent ignores old links
- **WHEN** `ENGRAM_PARENT` changes to a different URL
- **THEN** notes whose `parent.url` names the old URL are treated as unlinked, for both dedupe and offer targeting

#### Scenario: Unexchanged notes carry no link
- **WHEN** a note is learned with `ENGRAM_PARENT` unset
- **THEN** it has no `parent` key

### Requirement: Curation folds SHALL record aliases
`engram amend --target <offer> --discard --into <existing>` SHALL delete the offer note and its sidecar, as a bare `--discard` does. It SHALL also append the offer's basename to `<existing>`'s `aliases` list (without duplicates), and SHALL copy the offer's `parent` link to `<existing>` when `<existing>` has none. It is a bookkeeping amend: it SHALL NOT re-embed `<existing>`, and SHALL NOT queue an offer.

#### Scenario: Fold records the alias
- **WHEN** `engram amend --target O --discard --into E` runs
- **THEN** O and its sidecar are gone, and E's `aliases` includes O's basename

#### Scenario: Fold transfers a pull-down link
- **WHEN** O carries a `parent` link and E carries none
- **THEN** E carries O's `parent` link afterwards

### Requirement: Parent links and aliases SHALL survive every frontmatter rewrite
`parent` and `aliases` SHALL be preserved unchanged by every path that rewrites a note's frontmatter without being one of the exchange paths named above:
- `engram amend` (every flag, including `--activate` and `--clear-pending`);
- `engram resituate`;
- identity backfill (`engram update --backfill-identity`);
- Luhmann reparenting and rename-and-rewrite of wikilinks;
- vocab tag rewrites and legacy vocab cleanup;
- scrubbing references to deleted notes;
- skill registration refresh and adopt.

#### Scenario: Amend preserves the link
- **WHEN** `engram amend --target L --clear-pending` (or any content amend) rewrites a note carrying `parent` and `aliases`
- **THEN** both are unchanged in the written note

#### Scenario: Resituate preserves the link
- **WHEN** `engram resituate` rewrites a note carrying `parent` and `aliases`
- **THEN** both are unchanged

#### Scenario: Reparenting preserves the link
- **WHEN** Luhmann reparenting renames a note carrying `parent` and `aliases`
- **THEN** both are unchanged in the renamed note

#### Scenario: Identity backfill preserves the link
- **WHEN** `engram update --backfill-identity` rewrites a note carrying `parent`
- **THEN** `parent` is unchanged

### Requirement: Origin fields SHALL NOT be settable over the served API
The served-request decoding SHALL be unable to populate `parent`, `aliases`, `skill_hash`, `skill_key`, or `skill_source` on any note, regardless of key spelling in the request body.

#### Scenario: A remote client cannot forge a link
- **WHEN** a served `learn` request body carries `parent` or `aliases` values
- **THEN** the resulting pending note carries neither
