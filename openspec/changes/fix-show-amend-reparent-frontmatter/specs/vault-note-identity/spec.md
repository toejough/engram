## MODIFIED Requirements

### Requirement: Amend re-stamps identity fields on every write
`engram amend` SHALL re-detect and overwrite a note's `repo:`, `user:`, and `vault:` frontmatter fields on every call that changes content or relations, using the same detection as `engram learn` (see the three requirements above), regardless of what the note previously held. One exception: when `user:` detection resolves to an empty string (both `git config user.email` and the OS username lookup fail), a re-stamping amend SHALL keep the note's existing non-empty `user:` value and SHALL print a warning to stderr naming the note and the failed detection, rather than writing `user: ""`. When the note also has no prior `user:`, the field is written as detected. `vault:` SHALL be resolved through the same flag, then `ENGRAM_VAULT_NAME`, then `"personal"` order as `engram learn`, so a re-stamping amend never writes `vault: ""`. The calls that change content or relations are any content flag, `--supersedes`, and `--chunk-source`. This is unlike `source:`, `project:`, `issue:`, and `tier:`, which continue to be preserved verbatim through amend. Bookkeeping amends SHALL NOT re-stamp these fields, so an accepted offer keeps its declared author. Those amends are:
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

#### Scenario: Empty user detection keeps the prior user
- **WHEN** a re-stamping `engram amend` runs on a note whose `user:` is alice, and user detection resolves to an empty string
- **THEN** the amended note's `user:` is still alice, a warning naming the note is printed to stderr, and the amend otherwise succeeds

#### Scenario: Amend with no vault name configured stamps the default
- **WHEN** a re-stamping `engram amend` runs with neither `--vault-name` nor `ENGRAM_VAULT_NAME` set
- **THEN** the amended note's `vault:` is `personal`, never an empty string

### Requirement: Exchange fields SHALL survive every frontmatter rewrite
`xid`, `parent`, `aliases`, and `offer` SHALL be preserved unchanged by every path that rewrites a note's frontmatter and is not one of the exchange paths named above. Those paths are:
- `engram amend` (every flag, including `--activate` and `--clear-pending`);
- `engram resituate`;
- identity backfill;
- Luhmann reparenting and rename-and-rewrite of wikilinks (which only append to `aliases`);
- vocab tag rewrites, clear, legacy cleanup, self-tag, and version stamp;
- scrubbing references to deleted notes;
- skill registration refresh and adopt.

`engram resituate` SHALL also preserve every other frontmatter key it does not change, with its value, changing only `situation` and the body opener. This includes `pending`, `sources`, `tags`, `supersedes`, `vocab_version`, `issue`, and `project`, and keys the typed note model does not define.

`engram amend` SHALL likewise preserve every frontmatter key it does not change, with its value — including keys the typed note model does not define, and YAML anchors on keys it does not edit — for every amend kind that writes frontmatter: content flags, `--supersedes`, `--chunk-source`, `--clear-pending`, and identity re-stamping. Amend edits only the keys whose value its edit changes (plus `created`, re-emitted in its quoted form), as YAML-node edits. When such a key — or its value, or anything beneath the value — carries a YAML anchor, amend SHALL refuse the note untouched and name the key (`errFrontmatterAnchoredKey`). Before writing, amend SHALL decode the rewritten frontmatter again and refuse to write it if it does not decode (`errFrontmatterUndecodable`). For a note without such keys, amend SHALL write the same bytes, and so the same exchange hash, as before this requirement.

The curation fold (`engram amend --discard --into`) and the offer receipt SHALL edit `aliases:` and `parent:` as YAML nodes. They SHALL keep unknown keys under `parent:` and inside every link they keep. When `aliases:` or `parent:` — or anything beneath it — carries a YAML anchor and the edit would change that key, they SHALL refuse untouched and name the key (`errFrontmatterAnchoredKey`). They SHALL decode the rewritten frontmatter again before writing (`errFrontmatterUndecodable`). The fold SHALL do both before it writes the existing note or deletes the offer, so a refusal leaves both files untouched. For a note without unknown keys or anchors, both SHALL write the same bytes as before this requirement.

`engram resituate` SHALL convert a CRLF note to LF inside the single write it already makes, as rename and adopt do; it SHALL NOT refuse the note for its line endings.

`engram resituate` edits `situation` and `created` as a YAML node. When `situation` or `created` — or that key's value, or anything beneath the value — carries a YAML anchor, resituate SHALL refuse the note untouched and name the key (`errFrontmatterAnchoredKey`), rather than drop the value and leave an alias to it dangling. Before writing, resituate SHALL decode the rewritten frontmatter again and refuse to write it if it does not decode (`errFrontmatterUndecodable`).

#### Scenario: Amend preserves the exchange fields
- **WHEN** `engram amend --target L --clear-pending` (or any content amend) rewrites a note carrying `xid`, `parent`, `aliases`, and `offer`
- **THEN** all four are unchanged in the written note

#### Scenario: Resituate preserves every untouched field
- **WHEN** `engram resituate` rewrites a pending note carrying `tags`, `sources`, `supersedes`, `parent`, and `aliases`
- **THEN** all of them, and the pending marker, are unchanged, and only `situation` and the body opener differ

#### Scenario: Identity backfill preserves the link
- **WHEN** `engram update --backfill-identity` rewrites a note carrying `parent`
- **THEN** `parent` is unchanged

#### Scenario: Resituate keeps an unmodeled key
- **WHEN** `engram resituate` rewrites a note whose frontmatter carries a key the typed note model does not define (e.g. `luhmann_old: "12"`)
- **THEN** the written note still carries that key with the same value

#### Scenario: Resituate refuses an anchored key
- **WHEN** `engram resituate` would replace `situation` or `created` on a note whose `situation` or `created` key carries a YAML anchor, or whose value does
- **THEN** the resituate is refused, naming the key, and the note is unchanged

#### Scenario: Amend keeps an unmodeled key
- **WHEN** `engram amend` with any content flag, `--supersedes`, `--chunk-source`, or `--clear-pending` rewrites a fact, feedback, or runbook note whose frontmatter carries a key the typed note model does not define (e.g. `luhmann_old: "12"`, or a nested map)
- **THEN** the written note still carries that key with the same value

#### Scenario: Amend keeps an anchor on a key it does not edit
- **WHEN** `engram amend --object new` rewrites a fact note whose `source: &src test` is aliased by another key
- **THEN** the amend succeeds, and `source` and the aliasing key both still decode to `test`

#### Scenario: Amend refuses an anchored edited key
- **WHEN** `engram amend --object new` targets a fact note whose `object:` value carries a YAML anchor that another key aliases
- **THEN** the amend fails naming `object`, and the note is not written

#### Scenario: Amend output is unchanged for notes without unknown keys
- **WHEN** any amend kind rewrites a note whose frontmatter holds only keys the typed note model defines and no anchors
- **THEN** the written note is byte-identical to what amend wrote before this requirement, so its exchange hash is unchanged

#### Scenario: Resituate converts a CRLF note
- **WHEN** `engram resituate` rewrites a fact or feedback note whose lines end in `\r\n`
- **THEN** the rewrite succeeds, the written note contains no `\r`, and only `situation` and the body opener changed

#### Scenario: Fold refuses an anchored parent
- **WHEN** `engram amend --discard --into E` folds an offer into a note E whose `parent:` carries an anchor that another key aliases, and the fold would change E's parent links
- **THEN** the command fails naming `parent`, E is byte-identical, and the offer is not deleted

#### Scenario: Fold keeps unknown keys under parent
- **WHEN** the fold rewrites E's `parent:`, and E's `parent:` and one of its links carry keys the exchange model does not define
- **THEN** the written E still carries both keys with their values, alongside the folded links

#### Scenario: Receipt refuses an anchored parent
- **WHEN** an offer receipt would rewrite a note whose `parent:` carries an anchor that another key aliases
- **THEN** the receipt is refused naming `parent`, and the note is not written

#### Scenario: Receipt keeps unknown keys under parent
- **WHEN** an offer receipt rewrites a note whose `parent:` and one of its kept links carry keys the exchange model does not define
- **THEN** the written note still carries both keys with their values
