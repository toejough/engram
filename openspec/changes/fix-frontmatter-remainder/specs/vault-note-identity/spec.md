## MODIFIED Requirements

### Requirement: User field auto-detected at note creation
`engram learn` SHALL stamp the note's `user:` frontmatter field from `git config user.email` resolved at the working directory, falling back to the machine's OS username when no `user.email` is configured. The same detection SHALL stamp every first write: `engram learn`, a served learn's in-place rewrite of a pending offer, and a pull-down. When both sources resolve empty, the note SHALL be written with no `user:` key, never `user: ""`, and the command SHALL print one warning saying user detection resolved empty. A pull-down SHALL NOT keep the parent note's own `user:` in that case. Identity backfill, or a later re-stamping amend, fills the key in.

#### Scenario: Git user.email configured
- **WHEN** `engram learn` runs where `git config user.email` resolves to a non-empty value (repo-local or global config)
- **THEN** the note's `user:` frontmatter field is set to that email address

#### Scenario: No git user.email configured
- **WHEN** `engram learn` runs where `git config user.email` resolves to nothing
- **THEN** the note's `user:` frontmatter field is set to the OS username of the process running `engram learn`

#### Scenario: No user detectable
- **WHEN** `engram learn` or a pull-down runs where neither `git config user.email` nor the OS username lookup resolves
- **THEN** the note has no `user:` key, `vault:` is stamped as usual, and stderr carries one warning that user detection resolved empty

### Requirement: Amend re-stamps identity fields on every write
`engram amend` SHALL re-detect and overwrite a note's `repo:`, `user:`, and `vault:` frontmatter fields on every call that changes content or relations, using the same detection as `engram learn` (see the three requirements above), regardless of what the note previously held. One exception: when `user:` detection resolves to an empty string (both `git config user.email` and the OS username lookup fail), a re-stamping amend SHALL keep the note's existing non-empty `user:` value and SHALL print a warning to stderr naming the note and the failed detection, rather than writing `user: ""`. When the note also has no prior `user:`, the key stays absent: amend SHALL NOT write `user: ""`, on a re-stamping or a bookkeeping amend. `vault:` SHALL be resolved through the same flag, then `ENGRAM_VAULT_NAME`, then `"personal"` order as `engram learn`, so a re-stamping amend never writes `vault: ""`. The calls that change content or relations are any content flag, `--supersedes`, and `--chunk-source`. This is unlike `source:`, `project:`, `issue:`, and `tier:`, which continue to be preserved verbatim through amend. Bookkeeping amends SHALL NOT re-stamp these fields, so an accepted offer keeps its declared author. Those amends are:
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

`engram amend` SHALL likewise preserve every frontmatter key it does not change, with its value — including keys the typed note model does not define, and YAML anchors on keys it does not edit — for every amend kind that writes frontmatter: content flags, `--supersedes`, `--chunk-source`, `--clear-pending`, and identity re-stamping. Amend edits only the keys whose value its edit changes (plus `created`, re-emitted in its quoted form), as YAML-node edits. When such a key — or its value, or anything beneath the value — carries a YAML anchor, amend SHALL refuse the note untouched and name the key (`errFrontmatterAnchoredKey`). Before writing, amend SHALL decode the rewritten frontmatter again and refuse to write it if it does not decode (`errFrontmatterUndecodable`). When amend replaces a modeled list of mappings (`supersedes:`), an entry of the new list that names the same note as an entry of the old list (compared as basenames, a trailing `.md` ignored) SHALL keep the old entry's keys the typed entry does not define, with the new modeled values; only an entry naming a note the old list did not name SHALL be written fresh. For a note without such keys, amend SHALL write the same bytes, and so the same exchange hash, as before this requirement, except that it SHALL NOT write `user: ""` where the earlier amend did.

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

#### Scenario: Amend keeps unknown sub-keys of a kept supersedes entry
- **WHEN** `engram amend --supersedes "A|refutes|new" --supersedes "C|narrows|c"` rewrites a note whose `supersedes:` names A with an extra key `x_reason: kept`, and names B
- **THEN** the A entry carries `type: refutes`, `claim: new` and `x_reason: kept`; B is gone; and the C entry carries only `note`, `type` and `claim`

#### Scenario: Fold converts a CRLF existing note
- **WHEN** `engram amend --target O --discard --into E` folds an offer into a note E whose lines end in `\r\n`, and the fold changes E's aliases or parent links
- **THEN** the fold succeeds, E is written with no `\r\n`, and O and its sidecar are gone

### Requirement: Backfill for pre-existing notes missing identity fields
`engram update` SHALL detect notes missing `repo:`, `user:`, or `vault:` frontmatter fields and surface a notify-only notice naming the `--backfill-identity` flag, following the same detect-and-notify convention as the vocab-migration, Luhmann-branching, and chunk-pruning notices. `engram update --backfill-identity` SHALL rewrite each flagged note's `repo:`, `user:`, and `vault:` fields using the same `user:`/`vault:` detection as `learn`/`amend`, and the project-field-preferred `repo:` fallback described above, leaving all other note fields unchanged.

A note counts as missing identity when it has no `user:`. A note that also has no `vault:` predates identity and SHALL be stamped with all three fields, as described above. A note that has `vault:` but no `user:` was first written where user detection resolved empty; backfill SHALL fill in only `user:`, keeping its `repo:` and `vault:`, and SHALL leave it untouched while detection still resolves empty.

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

#### Scenario: Backfill fills in an omitted user
- **WHEN** `engram update --backfill-identity` runs on a note carrying `repo:` and `vault: work` but no `user:`, where user detection resolves
- **THEN** the note gains the detected `user:`, and its `repo:` and `vault: work` are unchanged; where detection still resolves empty, the note is not written

#### Scenario: Backfill output is unchanged for notes without unknown keys
- **WHEN** `engram update --backfill-identity` stamps a note whose frontmatter holds only keys the typed note model defines and no anchors
- **THEN** the written note is byte-identical to what backfill wrote before this requirement
