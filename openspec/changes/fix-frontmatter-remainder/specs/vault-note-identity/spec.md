## MODIFIED Requirements

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

`engram amend` SHALL read every note it edits as LF, converting each `\r\n` to `\n` (a lone `\r` is kept), and SHALL NOT refuse a note for its line endings. This covers the target of every amend kind that writes it and the existing note of a fold (`--discard --into`). A note amend writes SHALL be written as LF in the single atomic write amend already makes, and SHALL be re-embedded when it was converted, because conversion changes its content hash. A note amend does not write (a bare `--discard` target, a fold's offer, or a fold's existing note when the fold changes nothing) SHALL NOT be converted. The judged-version check (`--expect-hash`) SHALL run on the LF form, the form amend edits. For a note whose frontmatter is LF this is the hash `engram show` prints.

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

#### Scenario: Amend converts a CRLF note
- **WHEN** `engram amend` with any content flag, `--supersedes`, `--chunk-source`, `--clear-pending` or `--activate` rewrites a note whose lines end in `\r\n`
- **THEN** the amend succeeds, the written note contains no `\r\n`, it is byte-identical to what the same amend writes for the LF form of the note, and its sidecar is fresh

#### Scenario: Fold converts a CRLF existing note
- **WHEN** `engram amend --target O --discard --into E` folds an offer into a note E whose lines end in `\r\n`, and the fold changes E's aliases or parent links
- **THEN** the fold succeeds, E is written with no `\r\n`, and O and its sidecar are gone

### Requirement: Backfill for pre-existing notes missing identity fields
`engram update` SHALL detect notes missing `repo:`, `user:`, or `vault:` frontmatter fields and surface a notify-only notice naming the `--backfill-identity` flag, following the same detect-and-notify convention as the vocab-migration, Luhmann-branching, and chunk-pruning notices. `engram update --backfill-identity` SHALL rewrite each flagged note's `repo:`, `user:`, and `vault:` fields using the same `user:`/`vault:` detection as `learn`/`amend`, and the project-field-preferred `repo:` fallback described above, leaving all other note fields unchanged.

Identity backfill SHALL make that edit on the parsed frontmatter as YAML-node edits: it sets `repo:`, `user:` and `vault:`, and re-emits `created:` in its quoted form when the file's text for it differs. Every other key SHALL keep its value, including keys the typed note model does not define and YAML anchors on keys it does not edit. When a key it would set or re-emit, or that key's value, or anything beneath the value, carries a YAML anchor, backfill SHALL leave that note untouched, continue with the other notes, and then fail naming each refused note and key (`errFrontmatterAnchoredKey`). Before writing, it SHALL decode the rewritten frontmatter again and refuse to write a note whose frontmatter does not decode (`errFrontmatterUndecodable`). `--dry-run` SHALL report the same refusals without writing. For a note without CRLF line endings, unknown keys or anchors, backfill SHALL write the same bytes as before this requirement.

Identity backfill, and `engram update`'s missing-identity detection, SHALL read every note as LF, converting each `\r\n` to `\n`, and SHALL NOT skip a note for its line endings. A note backfill stamps SHALL be written as LF in the single atomic write backfill already makes, and its sidecar SHALL be rebuilt when it was converted, because conversion changes its content hash. A note backfill does not write (already stamped, refused, or under `--dry-run`) SHALL NOT be converted.

#### Scenario: Update detects notes missing identity fields
- **WHEN** `engram update` runs and the vault contains one or more notes with no `repo:`, `user:`, or `vault:` frontmatter fields
- **THEN** the update report includes a notify-only notice naming `engram update --backfill-identity`

#### Scenario: No notice when nothing is missing
- **WHEN** `engram update` runs and every note already has `repo:`, `user:`, and `vault:` fields
- **THEN** no identity-backfill notice appears in the update report

#### Scenario: Backfill stamps current environment onto flagged notes
- **WHEN** `engram update --backfill-identity` runs
- **THEN** every note previously missing `repo:`/`user:`/`vault:` gains those fields, `user:`/`vault:` from the current environment's detection and `repo:` per the project-field-preferred fallback above

#### Scenario: Backfill is idempotent
- **WHEN** `engram update --backfill-identity` runs a second time with no newly-missing notes
- **THEN** no notes are modified

#### Scenario: Backfill keeps an unmodeled key
- **WHEN** `engram update --backfill-identity` stamps a fact or feedback note whose frontmatter carries a key the typed note model does not define (e.g. `luhmann_old: "12"`, or a nested map)
- **THEN** the written note carries `repo:`, `user:` and `vault:` and still carries that key with the same value

#### Scenario: Backfill keeps an anchor on a key it does not edit
- **WHEN** `engram update --backfill-identity` stamps a note whose `source: &src test` is aliased by another key
- **THEN** the note is stamped, and `source` and the aliasing key both still decode to `test`

#### Scenario: Backfill refuses an anchored identity key
- **WHEN** `engram update --backfill-identity` runs on a vault where one flagged note's `user:` key carries a YAML anchor that another key aliases, and a second flagged note has none
- **THEN** the second note is stamped, the first note is byte-identical, and the command fails naming the first note and `user`

#### Scenario: Backfill converts a CRLF note
- **WHEN** `engram update --backfill-identity` runs on a vault holding a fact or feedback note missing identity whose lines end in `\r\n`
- **THEN** the note is stamped and written with no `\r\n`, byte-identical to the backfill of its LF form, and its sidecar is fresh; `engram update` without the flag had counted it in the missing-identity notice

#### Scenario: Backfill output is unchanged for notes without unknown keys
- **WHEN** `engram update --backfill-identity` stamps a note whose frontmatter holds only keys the typed note model defines and no anchors
- **THEN** the written note is byte-identical to what backfill wrote before this requirement
