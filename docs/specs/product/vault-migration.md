# Vault Migration - Product Spec

Some vaults need a one-off change that no ordinary write makes. A vault that holds only top-level
notes has never branched its Luhmann IDs, and a note written before notes carried their origin has no
`repo:`, `user:` and `vault:`. `engram update` notices both and names the opt-in remedy, and the remedy
runs only when asked: `engram update --reparent-luhmann` re-parents top-level notes and
`engram update --backfill-identity` stamps the missing origin. Both act on the vault located by [792a2a Vault location](./vault-identity.md#792a2a-vault-location).
The notices and the two flags belong to `engram update`, whose binary refresh and harness sync are
install-and-update's ([install-and-update](./install-and-update.md)).

## 792a10a The flat-vault notice

Requirement: [792a10a The flat-vault notice](#792a10a-the-flat-vault-notice) `engram update` MUST print a one-line notice naming `engram update
--reparent-luhmann` and the learn skill's batch mode ([792a15l The re-parent batch mode](./learn.md#792a15l-the-re-parent-batch-mode)) when the vault holds at least one note and the
Luhmann ID of every note at its root is top-level, one run of digits with no letter or digit branch.
It MUST NOT print the notice when any note has a deeper ID, when the vault has no note, or when the
vault cannot be read, and MUST NOT modify any note on account of it.

Rationale: a notice that only points at a mechanism affecting future notes, the learn skill's
disposition step for each capture, does nothing for a vault that has been flat since the skill stopped
placing notes. The notice gives it a real remedy to point at. An empty vault has nothing to
re-evaluate, and an ID that does not parse counts as not top-level, so a malformed vault is never
called flat.

### 792a10a1 UAT Scenarios

```gherkin
Scenario: 792a10a1a Only top-level notes
  Given a vault whose every note has a depth-1 Luhmann ID
  When the user runs engram update
  Then its output includes a notice naming engram update --reparent-luhmann, and no note file is modified

Scenario: 792a10a1b A branched note silences the notice
  Given a vault in which at least one note has a Luhmann ID deeper than depth 1
  When the user runs engram update
  Then its output includes no Luhmann-branching notice

Scenario: 792a10a1c An empty vault has no notice
  Given a vault with no notes
  When the user runs engram update
  Then its output includes no Luhmann-branching notice

Scenario: 792a10a1d The notice names the answering procedure
  Given a vault whose every note has a depth-1 Luhmann ID
  When the user runs engram update
  Then the notice names both engram update --reparent-luhmann and the learn skill's batch mode ([792a15l The re-parent batch mode](#792a15l-the-re-parent-batch-mode))
```

## 792a10b Re-parenting only on request

Requirement: [792a10b Re-parenting only on request](#792a10b-re-parenting-only-on-request) `engram update` MUST NOT rename a note or rewrite a link unless it is given
`--reparent-luhmann`. With the flag it MUST run only the re-parenting and MUST NOT refresh the binary
or the harness skills.

Rationale: Renaming a note and chasing every reference to its basename changes the note's on-disk identity,
and a bug could orphan a link or collide two notes onto one filename, which is why it is opt-in with a dry-run
preview. There is no pre-run uncommitted-changes guard; the dry run plus git as the fallback is the safety
net.

### 792a10b1 UAT Scenarios

```gherkin
Scenario: 792a10b1a Plain update never re-parents
  Given a vault of top-level notes with similar content
  When the user runs engram update without --reparent-luhmann
  Then no note is renamed and no link or supersedes reference is rewritten

Scenario: 792a10b1b The flag runs nothing else
  Given a machine with no installed harness
  When the user runs engram update --reparent-luhmann
  Then it derives candidates or applies answers, and does not fail for lack of a harness or a Go toolchain
```

## 792a10c Candidates to judge

Requirement: [792a10c Candidates to judge](#792a10c-candidates-to-judge) `engram update --reparent-luhmann` without `--answers` MUST print a JSON payload
and write no file. For each top-level note that has a sidecar, the payload MUST list up to three other
top-level notes with a sidecar whose body vector has cosine similarity of at least 0.75 with the note's,
nearest first, as a candidate of the note, its target, a body excerpt of each of at most 200 bytes, and
the similarity. It MUST also carry a `fingerprint` of the notes it considered, a `next_command` field
naming the literal follow-up command `engram update --reparent-luhmann --answers <path-to-answers-file>`,
and an `instruction`. The instruction MUST name the learn skill's batch mode ([792a15l The re-parent batch mode](./learn.md#792a15l-the-re-parent-batch-mode)) as the procedure that
holds the continuation, sibling and top disposition test, state the answers-file shape, require exactly
one entry per distinct candidate note, say that the finished answers file goes back to the user or the
orchestrating agent to preview with `--dry-run` and then apply, and not tell the answering pass to run
apply. When there are no candidates it MUST print `update --reparent-luhmann: no candidates found`
instead.

Rationale: whether one note is a sub-point of another is a judgment of content that an embedding cannot
make, so the binary proposes pairs and an agent decides. This is the boundary drawn for per-capture
disposition, and the flow follows `vocab refit`'s derive, names, apply.
The answering payload defers to the learn skill's batch mode ([792a15l The re-parent batch mode](./learn.md#792a15l-the-re-parent-batch-mode)), and the apply step stays with the user or
orchestrator after a `--dry-run`. Editing the skill to drop its stop-before-apply red flag was rejected,
because it would lose the human checkpoint before bulk renames.

### 792a10c1 UAT Scenarios

```gherkin
Scenario: 792a10c1a Candidates are emitted for review
  Given a vault of top-level notes with sidecars, some pairs of which are similar
  When the user runs engram update --reparent-luhmann with no --answers
  Then it prints a JSON payload of candidate pairs, a fingerprint and a next_command field, and renames and rewrites nothing

Scenario: 792a10c1b No pair is similar enough
  Given top-level notes none of whose pairs has a similarity of 0.75
  When the user runs engram update --reparent-luhmann
  Then it prints "update --reparent-luhmann: no candidates found" and writes nothing

Scenario: 792a10c1c The instruction routes to the learn skill
  Given a vault with candidates
  When the user runs engram update --reparent-luhmann
  Then the instruction names the learn skill's batch mode ([792a15l The re-parent batch mode](#792a15l-the-re-parent-batch-mode)), requires one entry per distinct candidate note, names --dry-run as the preview before applying, and does not tell the answering pass to run apply

Scenario: 792a10c1d Only top-level notes with sidecars are considered
  Given a branched note and a top-level note without a sidecar, each similar to a top-level note with a sidecar
  When the user runs engram update --reparent-luhmann
  Then neither appears as a candidate or a target
```

## 792a10d Answers that are refused

Requirement: [792a10d Answers that are refused](#792a10d-answers-that-are-refused) `engram update --reparent-luhmann --answers <file>` MUST refuse, writing nothing
and exiting non-zero, an answers file that cannot be read, one that is not valid JSON (`cannot parse
--answers JSON`), one whose fingerprint differs from the vault's current candidates (the vault changed
since they were derived), and one that names a note the vault does not hold (`--answers references an
unknown note`). The stale refusal MUST say to re-run `engram update --reparent-luhmann`.

### 792a10d1 UAT Scenarios

```gherkin
Scenario: 792a10d1a A missing answers file
  Given no file at the --answers path
  When the user runs engram update --reparent-luhmann --answers <path>
  Then it fails with an error that the answers file cannot be read, and the vault is unchanged

Scenario: 792a10d1b Malformed answers
  Given an answers file that is not valid JSON
  When the user applies it
  Then it fails with "cannot parse --answers JSON" and the vault is unchanged

Scenario: 792a10d1c Stale answers
  Given answers derived before the vault changed
  When the user applies them
  Then they are refused with a stale-answers error telling the user to re-run engram update --reparent-luhmann, and no file is written and the chunk index is unchanged

Scenario: 792a10d1d An unknown note
  Given an answers file naming a note ID that no note in the vault holds
  When the user applies it
  Then it fails with "--answers references an unknown note" and the vault is unchanged
```

## 792a10e Renaming and rewriting links

Requirement: [792a10e Renaming and rewriting links](#792a10e-renaming-and-rewriting-links) `engram update --reparent-luhmann --answers <file>` MUST give each note the
answers place as a `continuation` or `sibling` of a target the next ID that placement allocates, processing the
answers in ascending order of the notes' original IDs. It MUST rename the note's file and its sidecar to
the new ID, with the same date and slug, set its `luhmann:` to the new ID, record the old basename in its
`aliases:`, and rewrite every link to the old basename anywhere in the vault to the new one. That covers
`[[old]]` and `[[old.md]]` in a body or any frontmatter string, a `Supersedes:` line, and the `note:` of a
`supersedes:` entry. It MUST resolve a note that is renamed and also links another renamed note to that
note's new basename. A note answered `top` MUST NOT change, and a note that neither is renamed nor links a
renamed note MUST NOT be written. It MUST rebuild the sidecar of every note whose content it rewrote, so
that no note is stale afterwards.

### 792a10e1 UAT Scenarios

```gherkin
Scenario: 792a10e1a A continuation answer renames the note and rewrites links
  Given note 12 answered continuation of note 7, and note 20 containing a [[12.2026-05-01.old-slug]] link
  When the user applies the answers
  Then note 12's file and sidecar take the new child ID under 7, its luhmann: field is updated, its old basename is in its aliases, and note 20's link names the new basename

Scenario: 792a10e1b Renames are ordered by original ID
  Given notes 2 and 3 both answered continuation of note 1
  When the user applies the answers
  Then note 2 becomes 1a and note 3 becomes 1b

Scenario: 792a10e1c Cascading renames resolve to the final names
  Given two notes renamed in one apply that link each other
  When the user applies the answers
  Then each note's link names the other's new basename

Scenario: 792a10e1d A top answer changes nothing
  Given an answers file marking a candidate note position top
  When the user applies it
  Then the note is not renamed, no link to it is rewritten, and the chunk index is unchanged for it

Scenario: 792a10e1e Only supersedes entries are rewritten in a frontmatter list
  Given a note whose supersedes: entry and whose parent link both name notes
  When the named note is renamed
  Then the supersedes: entry is rewritten and the parent link is not

Scenario: 792a10e1f Rewritten notes are not left stale
  Given a note that links a renamed note
  When the user applies the answers
  Then engram embed status reports it with-embeddings

Scenario: 792a10e1g An untouched note is not written
  Given a note that neither is renamed nor links a renamed note
  When the user applies the answers
  Then its file is byte-identical afterwards
```

## 792a10f The chunk index follows the rename

Requirement: [792a10f The chunk index follows the rename](#792a10f-the-chunk-index-follows-the-rename) When an apply renamed at least one note, `engram update --reparent-luhmann` MUST,
in the same invocation, re-index the renamed notes and detach the chunk-manifest entries of their old
paths, so no `engram ingest --auto` or `engram prune` is needed. A failure of that step MUST NOT undo
the rename. It MUST report which stage failed, that the vault rename is intact and authoritative, and
that `engram ingest --auto` and `engram prune` remain as a manual fallback. When it renamed no note it
MUST NOT touch the chunk index.

### 792a10f1 UAT Scenarios

```gherkin
Scenario: 792a10f1a The new path is indexed and the old entry detached
  Given a note with chunks in the index
  When the user applies an answer that renames it
  Then the new path is indexed and the old path's manifest entry is detached, in that invocation

Scenario: 792a10f1b A failed re-index leaves the rename
  Given a rename that succeeded and a chunk-index step that fails
  When the apply finishes
  Then the renamed files stay renamed, and the output names the failed stage and the manual fallback of engram ingest --auto and engram prune

Scenario: 792a10f1c Nothing renamed, nothing re-indexed
  Given answers that rename no note
  When the user applies them
  Then the chunk index is not touched
```

## 792a10g What is left to judge

Requirement: [792a10g What is left to judge](#792a10g-what-is-left-to-judge) After a successful apply, `engram update --reparent-luhmann` MUST report either the
number of further candidates found in the vault's resulting state and the literal command to continue, or
that the vault is fully evaluated, with no further instruction. It MUST print `renamed <N> note(s)` first.

### 792a10g1 UAT Scenarios

```gherkin
Scenario: 792a10g1a More candidates remain
  Given an apply whose resulting vault still has a similar pair of top-level notes
  When the apply finishes
  Then the output reports the candidate count and says to run engram update --reparent-luhmann again

Scenario: 792a10g1b The vault is fully evaluated
  Given an apply whose resulting vault has no similar pair of top-level notes
  When the apply finishes
  Then the output says the vault is fully evaluated and gives no further command
```

## 792a10h Previewing an apply

Requirement: [792a10h Previewing an apply](#792a10h-previewing-an-apply) `engram update --reparent-luhmann --answers <file> --dry-run` MUST print the
old-to-new basename of each rename and the notes in which a link would be rewritten, and MUST NOT rename,
rewrite, re-index or prune anything. It MUST report the same refusals as an apply ([792a10i Notes a rename will not touch](#792a10i-notes-a-rename-will-not-touch)). The flag
`--dry-run` without `--answers` MUST be refused as a usage error, since deriving candidates already
writes nothing.

Rationale: a preview is an inspection tool and never a required step. An apply is gated by its own
fingerprint.

### 792a10h1 UAT Scenarios

```gherkin
Scenario: 792a10h1a A preview changes nothing
  Given valid, current answers
  When the user runs engram update --reparent-luhmann --answers <file> --dry-run
  Then the output lists each old-to-new basename and the notes whose links would be rewritten, and no file is modified and the chunk index is unchanged

Scenario: 792a10h1b A preview without answers is refused
  Given a vault
  When the user runs engram update --reparent-luhmann --dry-run
  Then it fails with a usage error saying --dry-run requires --answers, and writes nothing

Scenario: 792a10h1c A preview reports a refusal
  Given answers that would rename a note whose luhmann: is anchored
  When the user runs the preview
  Then it fails reporting the same refusal as an apply, and writes nothing
```

## 792a10i Notes a rename will not touch

Requirement: [792a10i Notes a rename will not touch](#792a10i-notes-a-rename-will-not-touch) A rename and rewrite MUST check, before it renames or writes any file, every note it
would rename, and MUST refuse the whole invocation, naming each refused note and its reason, when the
rewritten note's frontmatter does not decode as a mapping, or its `luhmann:` is not the new ID. An
anchored `luhmann:` value that another key aliases is one such note. The invocation MUST then rename and
write no note. The same step serves `register-skills --adopt`.

When it writes a note whose lines end in `\r\n`, whether renamed or a referrer, it MUST convert every
`\r\n` to `\n` in the same single atomic write and rebuild that note's sidecar in the same invocation.
A CRLF note that it does not otherwise write MUST stay byte-identical. A converted note's exchange hash
MUST equal the hash of the same note authored with LF line endings, and a note whose frontmatter was
already LF MUST keep its exchange hash.

### 792a10i1 UAT Scenarios

```gherkin
Scenario: 792a10i1a An anchored luhmann value is refused
  Given a note whose frontmatter has luhmann: &a "1050" and issue: *a, to be renamed
  When the user applies answers that rename it
  Then the invocation fails naming the note, and no note file or sidecar is renamed or rewritten

Scenario: 792a10i1b Every refusal is named
  Given two renamed notes that would each be refused
  When the user applies the answers
  Then one error names both notes and their reasons

Scenario: 792a10i1c A CRLF note that is renamed becomes LF
  Given a renamed note whose lines end in \r\n
  When the user applies the answers
  Then the written note has no \r\n, its luhmann: is its new ID, its alias is recorded, every other key decodes to its prior value, and its sidecar is fresh

Scenario: 792a10i1d A CRLF referrer is rewritten
  Given a CRLF note whose supersedes: names a note the apply renames
  When the user applies the answers
  Then the referrer is written as LF with its supersedes: note rewritten to the new basename

Scenario: 792a10i1e An untouched CRLF note stays as it was
  Given a CRLF note that is neither renamed nor links a renamed note
  When the user applies the answers
  Then its bytes are unchanged

Scenario: 792a10i1f The exchange hash survives the conversion
  Given a note with LF frontmatter and a CRLF body, and a note with CRLF frontmatter
  When a rename converts them
  Then the first note's exchange hash is unchanged and the second hashes as its LF-authored equivalent

Scenario: 792a10i1g An anchor-free note renames with every other key intact
  Given renamed notes whose frontmatter has no anchor
  When the user applies the answers
  Then each decodes, its luhmann: is its new ID, and every top-level key other than aliases decodes to its earlier value
```

## 792a10j The identity-backfill notice

Requirement: [792a10j The identity-backfill notice](#792a10j-the-identity-backfill-notice) `engram update` MUST print a one-line notice naming `engram update
--backfill-identity` when the vault holds a fact or feedback note that the backfill could stamp now. A
note counts when it has no `user:` and either has no `vault:` (it predates identity) or user detection
resolves now, and a CRLF note counts as its LF form. It MUST NOT print the notice when no note counts,
including when the only note without `user:` has `vault:` while detection still resolves empty, and
MUST NOT print it on a run that has just backfilled.

### 792a10j1 UAT Scenarios

```gherkin
Scenario: 792a10j1a Notes missing identity are noticed
  Given a vault with a fact note that has no repo:, user: or vault:
  When the user runs engram update
  Then the report includes a notice naming engram update --backfill-identity

Scenario: 792a10j1b No notice when nothing is missing
  Given a vault in which every fact and feedback note has user:
  When the user runs engram update
  Then no identity-backfill notice appears

Scenario: 792a10j1c No notice for a note that cannot be stamped
  Given user detection resolves empty and the only note without user: has vault:
  When the user runs engram update
  Then no identity-backfill notice appears

Scenario: 792a10j1d A CRLF note is noticed
  Given a fact note missing identity whose lines end in \r\n
  When the user runs engram update
  Then the notice appears
```

## 792a10k Stamping the missing origin

Requirement: [792a10k Stamping the missing origin](#792a10k-stamping-the-missing-origin) `engram update --backfill-identity` MUST stamp `repo:`, `user:` and `vault:` onto each
fact or feedback note that has no `user:` and no `vault:`, taking `user:` and `vault:` from the same
detection as `learn`, and `repo:` from the note's own `project:` when that is non-empty, else from the
same detection as `learn` run in the working directory. For a note that has `vault:` but no `user:`,
because it was first written where user detection resolved empty, it MUST fill in only `user:` and keep
its `repo:` and `vault:`, and MUST leave the note unwritten while detection still resolves empty. It MUST
leave every note it does not stamp, and any note that is neither fact nor feedback, unchanged, so a second
run with nothing newly missing writes nothing. It MUST print `update --backfill-identity: stamped <N>
note(s) with repo:/user:/vault: provenance`, or `nothing to backfill`, and with `--dry-run` print `[dry-run]`
and `would stamp` and write nothing.

Rationale: one backfill run has one working directory but may stamp notes that came from several
repositories, so the note's own `project:` is the better per-note signal than the run's directory. Amend
always uses fresh detection.

### 792a10k1 UAT Scenarios

```gherkin
Scenario: 792a10k1a The current environment is stamped
  Given fact and feedback notes missing repo:, user: and vault:
  When the user runs engram update --backfill-identity
  Then each gains the three fields, user: and vault: from the current environment's detection and repo: as described

Scenario: 792a10k1b The project field is preferred
  Given a note whose project: is non-empty and no repo:
  When the user runs engram update --backfill-identity
  Then the note's repo: is the project: value, not the working directory's git remote

Scenario: 792a10k1c Without a project field the repo is detected
  Given a note whose project: is empty or absent
  When the user runs engram update --backfill-identity
  Then the note's repo: is freshly detected from the working directory, as learn does

Scenario: 792a10k1d A backfill twice stamps once
  Given a vault that has just been backfilled
  When the user runs engram update --backfill-identity again
  Then no note is modified, and the output says "nothing to backfill"

Scenario: 792a10k1e An omitted user is filled in
  Given a note with repo: and vault: work but no user:, and user detection resolves
  When the user runs engram update --backfill-identity
  Then the note gains the detected user:, and its repo: and vault: work are unchanged; where detection still resolves empty the note is not written

Scenario: 792a10k1f A dry run writes nothing
  Given notes missing identity
  When the user runs engram update --backfill-identity --dry-run
  Then the output has "[dry-run]" and "would stamp" with the note count, and no file is modified

Scenario: 792a10k1g Other note types are left alone
  Given a vocab definition note and a runbook note without user:
  When the user runs engram update --backfill-identity
  Then neither is modified
```

## 792a10l What a backfill keeps

Requirement: [792a10l What a backfill keeps](#792a10l-what-a-backfill-keeps) A backfill MUST make its edit on the note's parsed frontmatter: it sets `repo:`,
`user:` and `vault:` and re-emits `created:` in its quoted form when the file's text differs, and every other
key keeps its value, including keys the typed note model does not define and anchors on keys it does not
edit. When a key it would set or re-emit, its value, or anything beneath it carries a YAML anchor, it MUST
leave that note untouched, go on with the other notes, and then fail naming each refused note and key.
Before it writes, it MUST decode the rewritten frontmatter again and refuse a note whose frontmatter does not
decode. `--dry-run` MUST report the same refusals without writing. For a note without CRLF line endings,
unknown keys or anchors, it MUST write the same bytes as before node edits were used.

It MUST read every note as LF, converting each `\r\n` to `\n`, and MUST NOT skip a note for its line
endings. A note it stamps MUST be written as LF in its single atomic write and have its sidecar rebuilt,
since conversion changes its content hash. A note it does not write, whether already stamped, refused or
under `--dry-run`, MUST NOT be converted.

### 792a10l1 UAT Scenarios

```gherkin
Scenario: 792a10l1a An unmodeled key is kept
  Given a fact or feedback note missing identity whose frontmatter has a key the typed note model does not define, such as luhmann_old: "12", or a nested map
  When the user runs engram update --backfill-identity
  Then the written note carries repo:, user: and vault: and still carries that key with the same value

Scenario: 792a10l1b An anchor on an unedited key is kept
  Given a note missing identity whose source: &src test is aliased by another key
  When the user runs engram update --backfill-identity
  Then the note is stamped, and source and the aliasing key both still decode to test

Scenario: 792a10l1c An anchored identity key is refused
  Given one flagged note whose user: key carries a YAML anchor that another key aliases, and a second flagged note without one
  When the user runs engram update --backfill-identity
  Then the second note is stamped, the first is byte-identical, and the command fails naming the first note and user

Scenario: 792a10l1d A CRLF note is converted
  Given a fact or feedback note missing identity whose lines end in \r\n
  When the user runs engram update --backfill-identity
  Then the note is stamped and written with no \r\n, byte-identical to the backfill of its LF form, and its sidecar is fresh

Scenario: 792a10l1e An ordinary note is written as before
  Given a note whose frontmatter holds only keys the typed note model defines and no anchors
  When the user runs engram update --backfill-identity
  Then the written note is byte-identical to what backfill wrote before node edits were used
```
