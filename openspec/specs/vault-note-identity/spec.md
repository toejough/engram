## Purpose

Every note gets structured, auto-detected provenance — which repo, which person, and which vault instance produced it — so origin survives beyond the existing free-text `source:` description and can be relied on by downstream consumers (filtering, attribution, parent exchange: `xid`, vault-qualified `parent` links, `aliases`).
## Requirements
### Requirement: Repo field auto-detected at note creation
`engram learn` SHALL stamp the note's `repo:` frontmatter field from the git remote `origin` URL of the working directory, falling back to the git root directory's basename when no `origin` remote is configured, and omitting the field entirely when the working directory is not inside a git repository.

#### Scenario: Origin remote configured
- **WHEN** `engram learn` runs in a working directory inside a git repo with an `origin` remote configured
- **THEN** the note's `repo:` frontmatter field is set to the `origin` remote's URL

#### Scenario: No origin remote
- **WHEN** `engram learn` runs in a working directory inside a git repo with no `origin` remote configured
- **THEN** the note's `repo:` frontmatter field is set to the basename of the git repo's top-level directory

#### Scenario: Not inside a git repo
- **WHEN** `engram learn` runs in a working directory that is not inside any git repository
- **THEN** the note's frontmatter has no `repo:` field

### Requirement: User field auto-detected at note creation
`engram learn` SHALL stamp the note's `user:` frontmatter field from `git config user.email` resolved at the working directory, falling back to the machine's OS username when no `user.email` is configured.

#### Scenario: Git user.email configured
- **WHEN** `engram learn` runs where `git config user.email` resolves to a non-empty value (repo-local or global config)
- **THEN** the note's `user:` frontmatter field is set to that email address

#### Scenario: No git user.email configured
- **WHEN** `engram learn` runs where `git config user.email` resolves to nothing
- **THEN** the note's `user:` frontmatter field is set to the OS username of the process running `engram learn`

### Requirement: Vault field resolved from explicit configuration
`engram learn` SHALL stamp the note's `vault:` frontmatter field by resolving, in order: a `--vault-name` flag, then an `ENGRAM_VAULT_NAME` environment variable, then the default value `"personal"` — the same flag-then-env-then-default order the existing `--vault`/`ENGRAM_VAULT_PATH` resolution already uses.

#### Scenario: Vault name flag supplied
- **WHEN** `engram learn --vault-name <name>` is run
- **THEN** the note's `vault:` frontmatter field is set to `<name>`

#### Scenario: Vault name from environment
- **WHEN** `engram learn` runs with `ENGRAM_VAULT_NAME` set in the environment and no `--vault-name` flag supplied
- **THEN** the note's `vault:` frontmatter field is set to the environment variable's value

#### Scenario: Vault name defaulted
- **WHEN** `engram learn` runs with neither `--vault-name` nor `ENGRAM_VAULT_NAME` set
- **THEN** the note's `vault:` frontmatter field is set to `"personal"`

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

### Requirement: Repo detection prefers the note's own project field during backfill only
`engram update --backfill-identity` SHALL prefer the note's existing `project:` field over a fresh git-remote read when `project:` is non-empty. `engram amend` SHALL NOT use this fallback — its `repo:` re-stamp always uses fresh detection from the current working directory, same as `learn`, regardless of the note's `project:` value.

#### Scenario: Backfilling a note with a project field set
- **WHEN** `engram update --backfill-identity` stamps `repo:` on a note whose `project:` field is non-empty
- **THEN** the note's `repo:` field is set to the `project:` value, not the current working directory's git remote

#### Scenario: Backfilling a note with no project field
- **WHEN** `engram update --backfill-identity` stamps `repo:` on a note whose `project:` field is empty or absent
- **THEN** the note's `repo:` field is freshly detected from the current working directory, same as `learn`

#### Scenario: Amend ignores the project field
- **WHEN** `engram amend` re-stamps `repo:` on a note whose `project:` field differs from the current working directory's repo
- **THEN** the note's `repo:` field is set to the current working directory's freshly detected repo, not the `project:` value

### Requirement: Backfill for pre-existing notes missing identity fields
`engram update` SHALL detect notes missing `repo:`, `user:`, or `vault:` frontmatter fields and surface a notify-only notice naming the `--backfill-identity` flag, following the same detect-and-notify convention as the vocab-migration, Luhmann-branching, and chunk-pruning notices. `engram update --backfill-identity` SHALL rewrite each flagged note's `repo:`, `user:`, and `vault:` fields using the same `user:`/`vault:` detection as `learn`/`amend`, and the project-field-preferred `repo:` fallback described above, leaving all other note fields unchanged.

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

### Requirement: Skill notes SHALL carry a `skill_hash` field
A runbook note mirroring a skill SHALL carry `skill_hash: <sha256 of the SKILL.md bytes its body was copied from>` in its frontmatter. The field SHALL be preserved by `engram amend` (amend re-stamps repo/user/vault but SHALL NOT drop or alter `skill_hash`); only registration SHALL change it.

#### Scenario: Amend preserves the hash
- **WHEN** `engram amend` rewrites a note carrying `skill_hash`
- **THEN** the field is unchanged in the written note

#### Scenario: Non-skill notes have no such field
- **WHEN** a note is captured by `engram learn` outside registration
- **THEN** it carries no `skill_hash` field

### Requirement: Skill-note identity fields SHALL survive every frontmatter rewrite
A runbook note created, adopted, or refreshed by skill registration SHALL carry these fields:

- `skill_key`, its source-qualified key (capability `skill-runbook-registration`);
- `skill_source`, the home-relative resolved path it was last copied from.

`skill_hash`, `skill_key`, and `skill_source` SHALL be preserved unchanged by every path that rewrites a note's frontmatter without being registration: `engram amend` (including `--clear-pending`), `engram resituate`, Luhmann reparenting (`engram update --reparent-luhmann`), rename-and-rewrite of wikilinks, identity backfill, and vocab tag rewrites. Only registration SHALL change them.

A note captured by `engram learn` outside registration SHALL carry neither `skill_key` nor `skill_source`.

#### Scenario: Amend preserves skill_key and skill_source
- **WHEN** `engram amend` rewrites a note carrying `skill_key` and `skill_source`
- **THEN** both fields are unchanged in the written note

#### Scenario: Reparenting preserves the identity fields
- **WHEN** Luhmann reparenting renames a skill note carrying `skill_hash`, `skill_key`, and `skill_source`
- **THEN** all three fields are unchanged in the renamed note

#### Scenario: Wikilink rewrite preserves the identity fields
- **WHEN** a rename-and-rewrite pass rewrites a wikilink inside a skill note's frontmatter
- **THEN** its `skill_hash`, `skill_key`, and `skill_source` are unchanged

#### Scenario: Ordinary capture carries neither
- **WHEN** a note is captured by `engram learn` outside registration
- **THEN** it carries no `skill_key` and no `skill_source` field

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

### Requirement: Exchange fields SHALL NOT be settable over the served API
The served-request decoding SHALL be unable to populate `xid`, `parent`, `aliases`, `skill_hash`, `skill_key`, or `skill_source` on any note, whatever the key spelling in the request body.

#### Scenario: A remote client cannot forge a link
- **WHEN** a served `learn` request body carries `parent`, `aliases`, or `xid` values
- **THEN** the resulting pending note carries none of those values

