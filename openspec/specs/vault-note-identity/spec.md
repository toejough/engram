## Purpose

The repo/user provenance stamped by `engram learn` is in `docs/changes/792_documentation-framework-adoption/792a_product-spec.md#792a3-added-note-writing`, and its detection in `792b_technical-spec.md#792b11-added-note-provenance`. Revision by `engram amend` and `engram resituate` (identity re-stamp, vault fill, key preservation, anchor refusal and CRLF conversion) is in `docs/changes/792_documentation-framework-adoption/792a_product-spec.md#792a4-added-note-revision`. Every note gets structured, auto-detected provenance — which repo, which person, and which vault instance produced it — so origin survives beyond the existing free-text `source:` description and can be relied on by downstream consumers (filtering, attribution, parent exchange: `xid`, vault-qualified `parent` links, `aliases`).
## Requirements
### Requirement: First-write identity residue
The product behavior of `repo:` and `user:` stamping by `engram learn` is in `docs/changes/792_documentation-framework-adoption/792a_product-spec.md#792a3-added-note-writing` (792a3o, 792a3p), and detection and the first-write stamp in `docs/changes/792_documentation-framework-adoption/792b_technical-spec.md#792b11-added-note-provenance`. These clauses wait for parent-sync (792c7t1) and vault-serving (792c7s1); the amend vault fill is in 792a4i:

- The same detection SHALL stamp every first write: a served learn's in-place rewrite of a pending offer, and a pull-down. A pull-down SHALL NOT keep the parent note's own `user:` when user detection resolves empty.

#### Scenario: No user detectable on a pull-down
- **WHEN** a pull-down runs where neither `git config user.email` nor the OS username lookup resolves
- **THEN** the note has no `user:` key, `vault:` is stamped as usual, and stderr carries one warning that user detection resolved empty

### Requirement: Curation and receipt bookkeeping keep the declared identity (residue)
The amend re-stamp and the `--activate` bookkeeping amend are in `docs/changes/792_documentation-framework-adoption/792a_product-spec.md#792a4-added-note-revision` (792a4h, 792a4i). These bookkeeping amends also SHALL NOT re-stamp `repo:`, `user:` and `vault:`, so an accepted offer keeps its declared author; they wait for curate (792c7u1) and parent-sync (792c7t1, the receipt):
- `--clear-pending`;
- `--discard --into` on the target note;
- the frontmatter-only link write that records an offer receipt.

#### Scenario: Accepting an offer keeps its author
- **WHEN** `engram amend --target O --clear-pending` runs on a pending offer whose `user:` is alice, from an environment whose detected user is bob
- **THEN** O's `user:` is still alice

### Requirement: Repo detection prefers the note's own project field during backfill only
`engram update --backfill-identity` SHALL prefer the note's existing `project:` field over a fresh git-remote read when `project:` is non-empty.

#### Scenario: Backfilling a note with a project field set
- **WHEN** `engram update --backfill-identity` stamps `repo:` on a note whose `project:` field is non-empty
- **THEN** the note's `repo:` field is set to the `project:` value, not the current working directory's git remote

#### Scenario: Backfilling a note with no project field
- **WHEN** `engram update --backfill-identity` stamps `repo:` on a note whose `project:` field is empty or absent
- **THEN** the note's `repo:` field is freshly detected from the current working directory, same as `learn`

### Requirement: Backfill for pre-existing notes missing identity fields
`engram update` SHALL detect notes missing `repo:`, `user:`, or `vault:` frontmatter fields and surface a notify-only notice naming the `--backfill-identity` flag, following the same detect-and-notify convention as the vocab-migration, Luhmann-branching, and chunk-pruning notices. `engram update --backfill-identity` SHALL rewrite each flagged note's `repo:`, `user:`, and `vault:` fields using the same `user:`/`vault:` detection as `learn`/`amend`, and the project-field-preferred `repo:` fallback described above, leaving all other note fields unchanged.

A note counts as missing identity when it has no `user:`. A note that also has no `vault:` predates identity and SHALL be stamped with all three fields, as described above. A note that has `vault:` but no `user:` was first written where user detection resolved empty; backfill SHALL fill in only `user:`, keeping its `repo:` and `vault:`, and SHALL leave it untouched while detection still resolves empty. `engram update`'s missing-identity notice SHALL count only notes backfill can stamp at that moment, so it never asks for a backfill that would stamp nothing.

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

#### Scenario: No backfill notice for a note backfill cannot stamp
- **WHEN** `engram update` runs where user detection resolves empty, and the only note without `user:` has `vault:`
- **THEN** no identity-backfill notice appears

#### Scenario: Backfill fills in an omitted user
- **WHEN** `engram update --backfill-identity` runs on a note carrying `repo:` and `vault: work` but no `user:`, where user detection resolves
- **THEN** the note gains the detected `user:`, and its `repo:` and `vault: work` are unchanged; where detection still resolves empty, the note is not written

#### Scenario: Backfill output is unchanged for notes without unknown keys
- **WHEN** `engram update --backfill-identity` stamps a note whose frontmatter holds only keys the typed note model defines and no anchors
- **THEN** the written note is byte-identical to what backfill wrote before this requirement

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

The curation fold (`engram amend --discard --into`) and the offer receipt SHALL edit `aliases:` and `parent:` as YAML nodes. They SHALL keep unknown keys under `parent:` and inside every link they keep. When `aliases:` or `parent:` — or anything beneath it — carries a YAML anchor and the edit would change that key, they SHALL refuse untouched and name the key (`errFrontmatterAnchoredKey`). They SHALL decode the rewritten frontmatter again before writing (`errFrontmatterUndecodable`). The fold SHALL do both before it writes the existing note or deletes the offer, so a refusal leaves both files untouched. For a note without unknown keys or anchors, both SHALL write the same bytes as before this requirement.

Amend's and resituate's key preservation, anchor refusal and CRLF conversion are in `docs/changes/792_documentation-framework-adoption/792a_product-spec.md#792a4-added-note-revision` (792a4j, 792a4k, 792a4m). Waiting for curate (792c7u1): a fold (`--discard --into`) SHALL read its existing note as LF and write it as LF only when the fold changes it, and the judged-version check (`--expect-hash`) SHALL run on the LF form, the form amend edits; for a note whose frontmatter is LF this is the hash `engram show` prints.

#### Scenario: Amend preserves the exchange fields
- **WHEN** `engram amend --target L --clear-pending` (or any content amend) rewrites a note carrying `xid`, `parent`, `aliases`, and `offer`
- **THEN** all four are unchanged in the written note

#### Scenario: Identity backfill preserves the link
- **WHEN** `engram update --backfill-identity` rewrites a note carrying `parent`
- **THEN** `parent` is unchanged

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

#### Scenario: Fold converts a CRLF existing note
- **WHEN** `engram amend --target O --discard --into E` folds an offer into a note E whose lines end in `\r\n`, and the fold changes E's aliases or parent links
- **THEN** the fold succeeds, E is written with no `\r\n`, and O and its sidecar are gone

### Requirement: Exchange fields SHALL NOT be settable over the served API
The served-request decoding SHALL be unable to populate `xid`, `parent`, `aliases`, `skill_hash`, `skill_key`, or `skill_source` on any note, whatever the key spelling in the request body.

#### Scenario: A remote client cannot forge a link
- **WHEN** a served `learn` request body carries `parent`, `aliases`, or `xid` values
- **THEN** the resulting pending note carries none of those values

### Requirement: Note readers SHALL read a CRLF note as its LF form
The embedding readers (situation, body and content hash) are in `docs/changes/792_documentation-framework-adoption/792b_technical-spec.md#792b10-added-embedding-sidecar` (792b10c9), and amend's and resituate's reads in 792a4m. The other readers wait for memory-query (792c7f1: the project filter, recency, vault metadata, note kind, trigger index and red-flags preview), vault-health (792c7h1: `engram check`, and a CRLF note's sidecar not reported stale), vault-counting (792c7g1: `engram count`) and controlled-vocabulary (792c7e1: vocab tagging, removal and refit).

Every reader of a note's frontmatter or body SHALL read the note as its LF form (each `\r\n` converted to `\n`), so a note with CRLF line endings is read exactly as its LF conversion: `engram query`'s project filter, recency, vault metadata (tags, supersedes, triggers), note kind (runbook trigger indexing, the qa-question exclusion and the kind label) and red-flags preview, vocab tagging, removal and refit, `engram check` and `engram count`. `embed.SplitFrontmatter` SHALL stay LF-only; the readers convert before splitting. No reader SHALL write a note only to convert it: a writer converts a CRLF note only inside a write it makes anyway, and an assignment that changes nothing writes nothing.

#### Scenario: A CRLF note embeds and hashes as its LF form
- **WHEN** a note's lines end in `\r\n`
- **THEN** its situation text, body text and content hash equal those of its LF form, so its sidecar is not reported stale for its line endings

#### Scenario: Query, check and count read a CRLF note
- **WHEN** `engram query --project <p>`, `engram check` and `engram count` run on a vault holding a CRLF note whose `project:` is `<p>`
- **THEN** query returns the note, check judges its `situation:`, and count includes it

#### Scenario: A CRLF runbook's trigger matches
- **WHEN** `engram query --text` contains a trigger of a runbook whose lines end in `\r\n`
- **THEN** the runbook is placed ahead as a trigger hit, labelled `kind: runbook`

#### Scenario: Vocab tagging reads a CRLF note
- **WHEN** vocab assignment tags a CRLF note with a new term
- **THEN** the tag is written and the note is written as LF; a CRLF note whose tags do not change is not written

