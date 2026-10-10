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
