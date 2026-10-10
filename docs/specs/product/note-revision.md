# Note Revision - Product Spec

`engram amend`, `engram resituate` and `engram activate` change notes that already exist: amend
replaces a note's fields, supersessions and chunk sources, resituate rewrites its situation, and
activate records that it was used. Writing new notes is the note-writing capability ([note-writing](./note-writing.md)). The curation
flags of `amend` (`--clear-pending`, `--discard`, `--into`, `--expect-hash`) belong to [curate](./curate.md), and
`activate --parent`, a pull-down on a local miss, and the offers a revision queues for a parent
belong to [parent-sync](./parent-sync.md). The vocabulary tagging that follows a revision belongs to
[controlled-vocabulary](./controlled-vocabulary.md).

## 792a4a Note references

Requirement: [792a4a Note references](#792a4a-note-references) `engram amend --target` and `engram resituate --note` MUST accept a note's full
basename, the basename followed by `.md`, a `[[wikilink]]` to it (with or without a `|display`
part), or its bare Luhmann ID, and MUST fail with an error saying the note was not found, naming
the reference as given, and write nothing when no note matches.

### 792a4a1 UAT Scenarios

```gherkin
Scenario Outline: 792a4a1a Every reference form finds the note
  Given a vault holding note 1.2026-10-09.a
  When the user runs engram amend --target <ref> --object new
  Then note 1.2026-10-09.a is amended and its path is printed on stdout

  Examples:
    | ref                     |
    | 1.2026-10-09.a          |
    | 1.2026-10-09.a.md       |
    | "[[1.2026-10-09.a]]"    |
    | "[[1.2026-10-09.a\|x]]" |
    | 1                       |

Scenario: 792a4a1b An unknown reference is refused
  Given a vault with no note 9
  When the user runs engram amend --target 9, or engram resituate --note 9 --situation s
  Then it fails saying note "9" was not found, and no note is written
```

## 792a4b Field replacement

Requirement: [792a4b Field replacement](#792a4b-field-replacement) `engram amend` MUST replace only the content fields it is given that belong to
the note's kind — `--situation`, `--subject`, `--predicate` and `--object` for a fact;
`--situation`, `--behavior`, `--impact` and `--action` for feedback; `--situation`, `--done-when`,
`--body`, and the whole `red_flags:` and `triggers:` lists from repeated `--red-flag` and
`--trigger` for a runbook — MUST rebuild a fact or feedback body formula from the updated fields,
and MUST print the note's path on stdout.

Rationale: a field that is not given keeps its value, so an agent can correct one part of a lesson
without restating the rest. The body formula repeats the situation and the lesson, so it is rebuilt
whenever they change.

### 792a4b1 UAT Scenarios

```gherkin
Scenario: 792a4b1a A fact field is replaced
  Given a fact note whose situation is "testing", subject s, predicate p and object o
  When the user runs engram amend --target <note> --object o2
  Then the note's object: is o2, its other fields are unchanged, its body reads "Information learned: when in testing, s p o2.", and its path is printed

Scenario: 792a4b1b A feedback field is replaced
  Given a feedback note whose situation is x and action a
  When the user runs engram amend --target <note> --action b
  Then the note's action: is b and its body reads "Lesson learned: when x, b."

Scenario: 792a4b1c A runbook's steps and red flags are replaced
  Given a runbook note with body "1. old" and red flags A and B
  When the user runs engram amend --target <note> --body "1. new" --red-flag C
  Then the note's body is "1. new" and its red_flags: lists only C
```

## 792a4c Other note kinds are refused

Requirement: [792a4c Other note kinds are refused](#792a4c-other-note-kinds-are-refused) `engram amend` MUST refuse, writing nothing, a note that is not a fact,
feedback or runbook note, or that has no frontmatter.

### 792a4c1 UAT Scenarios

```gherkin
Scenario: 792a4c1a A question-and-answer note is refused
  Given a vault holding an answer note qa.2026-10-09.why.a
  When the user runs engram amend --target qa.2026-10-09.why.a --situation s
  Then it fails saying the note type is unknown, and the note is unchanged
```

## 792a4d Supersession

Requirement: [792a4d Supersession](#792a4d-supersession) `engram amend --supersedes <note>|<type>|<claim>`, repeatable, MUST replace the
note's whole `supersedes:` list and all of its `Supersedes:` body lines with the given entries, and
`engram learn` and `engram amend` MUST write each body line as `Supersedes: [[<target>]] — <type>:
<claim>`, where `<target>` drops any trailing `.md` from the given note, while the entry's `note:`
field keeps the note exactly as given.

Rationale: the vault's link graph names each note by its basename without `.md`, as Obsidian does.
Supersession lines were once written with the `.md` kept, so they never resolved to the note they
named (#717). The frontmatter `note:` is a file reference rather than a graph link, so it keeps the
filename. Replacing the whole list, rather than appending to it, makes repeating an amend harmless.

### 792a4d1 UAT Scenarios

```gherkin
Scenario: 792a4d1a A supersession is written in canonical form
  Given a vault holding note 107.2026-06-27.validate-lazy-retrieval-by-measuring-fetch-behavior
  When the user runs engram amend --target <note> --supersedes "107.2026-06-27.validate-lazy-retrieval-by-measuring-fetch-behavior.md|refutes|claim"
  Then the body line is "Supersedes: [[107.2026-06-27.validate-lazy-retrieval-by-measuring-fetch-behavior]] — refutes: claim", and the supersedes: entry's note: is 107.2026-06-27.validate-lazy-retrieval-by-measuring-fetch-behavior.md

Scenario: 792a4d1b The list is replaced, not appended to
  Given a note whose supersedes: names A and B, with a Supersedes: line for each
  When the user runs engram amend --target <note> --supersedes "C|narrows|c"
  Then the note's supersedes: names only C, and its body has one Supersedes: line, for C
```

## 792a4e Chunk sources

Requirement: [792a4e Chunk sources](#792a4e-chunk-sources) `engram amend --chunk-source <source#anchor>`, repeatable, MUST add each chunk
ID to the note's `sources:` once, after the IDs already there, and MUST keep every ID already
there.

### 792a4e1 UAT Scenarios

```gherkin
Scenario: 792a4e1a Chunk sources are merged without duplicates
  Given a note whose sources: lists a#1, and a chunk index holding a#1 and b#2
  When the user runs engram amend --target <note> --chunk-source a#1 --chunk-source b#2
  Then the note's sources: lists a#1 then b#2
```

## 792a4f Amend argument validation

Requirement: [792a4f Amend argument validation](#792a4f-amend-argument-validation) `engram amend` MUST refuse, writing nothing, a `--supersedes` that is not
`<note>|<type>|<claim>` with a type of `updates`, `narrows` or `refutes` and a non-empty claim, a `--trigger` shorter than
three characters after trimming, and a `--chunk-source` ID that the chunk index does not hold.

Rationale: `engram learn` records chunk sources without checking them, because the chunks a note is
written from may not be indexed yet ([792a3h Optional metadata](./note-writing.md#792a3h-optional-metadata)). An amend runs later, so it can check them.

### 792a4f1 UAT Scenarios

```gherkin
Scenario Outline: 792a4f1a An invalid amend argument is refused before any write
  Given a vault holding fact note 1
  When the user runs engram amend --target 1 with <argument>
  Then it fails with an error saying <problem>, and the note is unchanged

  Examples:
    | argument                     | problem                                        |
    | --supersedes "a\|bogus\|c"   | type must be updates\|narrows\|refutes, got "bogus" |
    | --supersedes "a\|updates"    | format must be "<basename>\|<type>\|<claim>"    |
    | --supersedes "a\|updates\|"  | claim must be non-empty                        |
    | --trigger ab                 | trigger must be at least 3 characters after trimming |
    | --chunk-source nope#x        | unresolved chunk-source id "nope#x"            |
```

## 792a4g Re-embedding after an amend

Requirement: [792a4g Re-embedding after an amend](#792a4g-re-embedding-after-an-amend) When `engram amend` changes a content field of the note, it MUST rebuild the
note's `.vec.json` sidecar, and when it changes only supersessions, chunk sources or identity, it
MUST NOT rebuild the sidecar. When rebuilding fails, the note MUST still be written, and the
command MUST print a warning naming the note and succeed.

Rationale: the sidecar embeds the note's situation and body, and its content hash leaves out the
`Supersedes:` lines and all frontmatter but the situation, so a provenance-only change leaves the
sidecar current. A failed rebuild is recoverable later, because the note itself was written in one
atomic step.

### 792a4g1 UAT Scenarios

```gherkin
Scenario: 792a4g1a A content amend re-embeds
  Given a fact note with a current sidecar
  When the user runs engram amend --target <note> --object new
  Then the note's sidecar is rebuilt from the amended text, and engram embed status reports it ok

Scenario: 792a4g1b A provenance-only amend does not re-embed
  Given a fact note with a current sidecar
  When the user runs engram amend --target <note> with only --supersedes or --chunk-source
  Then the note's sidecar file is unchanged

Scenario: 792a4g1c A failed re-embed keeps the amend
  Given embedding fails for the note being amended
  When the user runs engram amend --target <note> --object new
  Then the note is amended, stderr carries a warning naming it, and the command succeeds
```

## 792a4h Identity re-stamp

Requirement: [792a4h Identity re-stamp](#792a4h-identity-re-stamp) An `engram amend` given any content flag, `--supersedes` or `--chunk-source`
MUST overwrite the note's `repo:`, `user:` and `vault:` with the values detected in its own
environment, as `engram learn` detects them ([792a3o Repository provenance](./note-writing.md#792a3o-repository-provenance), [792a3p User provenance](./note-writing.md#792a3p-user-provenance), [792a2b Vault name](./vault-identity.md#792a2b-vault-name)), whatever the note held
before and whatever its `project:` says. When user detection resolves empty, it MUST keep a
non-empty `user:` the note already has and print a warning naming the note, and MUST leave `user:`
absent when the note has none, never writing `user: ""`. An `engram amend --activate` with none of
those flags MUST keep `repo:`, `user:` and `vault:` as the note declares them. `source:`,
`project:`, `issue:` and `tier:` MUST be kept by every amend.

Rationale: a note's identity fields say which repository, person and vault last changed what the
note says, so a content or relation change re-stamps them. Bookkeeping is not authorship, so an
amend that only records a use keeps the declared author. When user detection fails, refusing the
amend would block content edits on a machine with broken identity, and keeping the old value
silently would hide the broken environment; keeping it with a warning does neither (#776).

### 792a4h1 UAT Scenarios

```gherkin
Scenario: 792a4h1a Amend from another environment re-stamps
  Given a fact note whose repo:, user: and vault: differ from the current environment's
  When the user runs engram amend --target <note> with a content flag, --supersedes or --chunk-source
  Then the note's repo:, user: and vault: are the current environment's detected values

Scenario: 792a4h1b Amend backfills missing identity
  Given a fact note with no repo:, user: or vault:
  When the user runs engram amend --target <note> --object new
  Then the note gains the current environment's repo:, user: and vault:

Scenario: 792a4h1c The project field does not stand in for the repo
  Given a note whose project: differs from the current working directory's repository
  When the user runs engram amend --target <note> --object new
  Then the note's repo: is the current working directory's repository, not the project: value

Scenario: 792a4h1d Empty user detection keeps the prior user
  Given a note whose user: is alice, and neither git config user.email nor the OS username resolves
  When the user runs engram amend --target <note> --object new
  Then the note's user: is still alice, stderr carries a warning naming the note, and the amend succeeds

Scenario: 792a4h1e Empty user detection never writes an empty user
  Given a note with no user:, and neither git config user.email nor the OS username resolves
  When the user runs engram amend --target <note> --object new
  Then the note has no user: field

Scenario: 792a4h1f Without a vault name the default is stamped
  Given neither --vault-name nor ENGRAM_VAULT_NAME is set
  When the user runs engram amend --target <note> --object new
  Then the note's vault: is personal, never an empty string

Scenario: 792a4h1g Recording a use keeps the declared identity
  Given a note whose repo:, user: and vault: differ from the current environment's
  When the user runs engram amend --target <note> --activate
  Then the note's repo:, user: and vault: are unchanged
```

## 792a4i Vault name on every amend

Requirement: [792a4i Vault name on every amend](#792a4i-vault-name-on-every-amend) Every `engram amend` that writes a note, one that keeps its declared identity
included, MUST write the resolved vault name ([792a2b Vault name](./vault-identity.md#792a2b-vault-name)) into a note whose `vault:` is absent or
empty, and MUST keep a `vault:` the note declares.

Rationale: a `vault: ""` would look like a real value to every later reader, and a note written
before identity existed would otherwise keep no vault at all.

### 792a4i1 UAT Scenarios

```gherkin
Scenario: 792a4i1a A use recorded on a note without vault fills it in
  Given a note with no user: and no vault:, and no vault name configured
  When the user runs engram amend --target <note> --activate
  Then the note gains vault: personal and gains no user: field
```

## 792a4j Untouched frontmatter is kept

Requirement: [792a4j Untouched frontmatter is kept](#792a4j-untouched-frontmatter-is-kept) `engram amend` and `engram resituate` MUST keep every frontmatter key they do
not change, with its value, including keys engram does not define and YAML anchors on keys they do
not edit. When an amend replaces `supersedes:`, an entry that names the same note as an existing
entry, compared without a trailing `.md`, MUST keep the existing entry's keys that engram does not
define.

Rationale: hand-edited, imported and Obsidian-edited notes carry keys that engram's note model does
not know. Revisions used to re-render the frontmatter from that model and silently dropped them
(#780).

### 792a4j1 UAT Scenarios

```gherkin
Scenario: 792a4j1a Amend keeps an unmodeled key
  Given a fact, feedback or runbook note whose frontmatter carries luhmann_old: "12" and a nested map that engram does not define
  When the user runs engram amend --target <note> with a content flag, --supersedes or --chunk-source
  Then the written note still carries luhmann_old: "12" and the nested map with the same values

Scenario: 792a4j1b Amend keeps an anchor on a key it does not edit
  Given a fact note whose source: &src test is aliased by another key
  When the user runs engram amend --target <note> --object new
  Then the amend succeeds, and source: and the aliasing key both still read test

Scenario: 792a4j1c Amend keeps the extra keys of a kept supersession
  Given a note whose supersedes: names A with an extra key x_reason: kept, and names B
  When the user runs engram amend --target <note> --supersedes "A|refutes|new" --supersedes "C|narrows|c"
  Then the A entry carries type: refutes, claim: new and x_reason: kept; B is gone; and the C entry carries only note, type and claim

Scenario: 792a4j1d Resituate keeps every untouched field
  Given a fact note carrying tags:, sources:, supersedes:, pending: true, parent:, aliases: and luhmann_old: "12"
  When the user runs engram resituate --note <note> --situation new
  Then all of those are unchanged, and only situation: and the body's first line differ
```

## 792a4k Anchored keys are refused

Requirement: [792a4k Anchored keys are refused](#792a4k-anchored-keys-are-refused) `engram amend` and `engram resituate` MUST refuse, leaving the note unchanged,
an edit to a frontmatter key whose key or value, or anything beneath the value, carries a YAML
anchor, with an error naming the key.

Rationale: replacing an anchored value would leave every alias to it dangling, so the frontmatter
would no longer read back.

### 792a4k1 UAT Scenarios

```gherkin
Scenario: 792a4k1a Amend refuses an anchored edited key
  Given a fact note whose object: value carries an anchor that another key aliases
  When the user runs engram amend --target <note> --object new
  Then it fails naming object, and the note is unchanged

Scenario: 792a4k1b Resituate refuses an anchored situation
  Given a fact note whose situation: or created: carries an anchor
  When the user runs engram resituate --note <note> --situation new
  Then it fails naming the key, and the note is unchanged
```

## 792a4l Resituation

Requirement: [792a4l Resituation](#792a4l-resituation) `engram resituate --note <ref> --situation <text>` MUST replace a fact or
feedback note's `situation:` with the text, rebuild the first line of its body around it, keep the
rest of the body, rebuild the note's sidecar, and print the note's path on stdout, and MUST refuse,
writing nothing, a note of any other kind. When rebuilding the sidecar fails, the command MUST fail
with an error naming the note, with the note already rewritten.

Rationale: a fact or feedback note states its situation twice, in `situation:` and in its body
formula, and `engram learn` only creates notes. Before resituate, the two copies could be kept in
step only by hand. Unlike an amend's warning, a failed sidecar rebuild fails the command, because a
resituate is an explicit request to change what the note is found by.

### 792a4l1 UAT Scenarios

```gherkin
Scenario: 792a4l1a A fact is resituated
  Given a fact note whose situation is "old" and whose body reads "Information learned: when in old, s p o." followed by a related-notes section
  When the user runs engram resituate --note <note> --situation new
  Then situation: is new, the body's first line reads "Information learned: when in new, s p o.", the related-notes section is unchanged, the sidecar is rebuilt, and the note's path is printed

Scenario: 792a4l1b A runbook cannot be resituated
  Given a vault holding runbook note 2
  When the user runs engram resituate --note 2 --situation new
  Then it fails saying the note type runbook is unknown, and the note is unchanged

Scenario: 792a4l1c A failed sidecar rebuild fails the command
  Given embedding fails for the note being resituated
  When the user runs engram resituate --note <note> --situation new
  Then the command fails naming the note, and the note's situation is already new
```

## 792a4m CRLF notes

Requirement: [792a4m CRLF notes](#792a4m-crlf-notes) `engram amend` and `engram resituate` MUST read a note whose lines end in
`\r\n` as its LF form, never refusing it for its line endings, MUST write it as LF in the single
write they make, byte-identical to what the same command writes for the note's LF form, and MUST
rebuild its sidecar.

Rationale: notes edited on Windows or by other tools can carry CRLF line endings, and amend used to
refuse them as having no frontmatter (#789). A note is converted only inside a write the command
makes anyway, so no command writes a note just to convert it. Conversion changes the note's text,
so its sidecar is rebuilt even when the amend itself would not re-embed.

### 792a4m1 UAT Scenarios

```gherkin
Scenario: 792a4m1a Amend converts a CRLF note
  Given a fact, feedback or runbook note whose lines end in \r\n
  When the user runs engram amend --target <note> with a content flag, --supersedes, --chunk-source or --activate
  Then the amend succeeds, the written note contains no \r\n, it is byte-identical to what the same amend writes for the note's LF form, and its sidecar is fresh

Scenario: 792a4m1b Resituate converts a CRLF note
  Given a fact or feedback note whose lines end in \r\n
  When the user runs engram resituate --note <note> --situation new
  Then the rewrite succeeds, the written note contains no \r, and only situation: and the body's first line changed
```

## 792a4n Activation

Requirement: [792a4n Activation](#792a4n-activation) `engram activate --note <ref>`, repeatable, where each ref is a note's path,
absolute or relative to the vault, with or without `.md`, MUST record today's date as the last-used
date in each named note's sidecar, changing nothing else in the sidecar and nothing in the note, and
MUST count a note that has no sidecar as activated. With no parent configured, a ref that names no
note MUST print a warning naming it, the other refs MUST still be activated, and the command MUST
fail saying how many refs failed. `engram amend --activate` MUST record the last-used date in the
same way, in the same command as its amend.

Rationale: activation is agent-driven. Query never marks a note used; the recall skill activates
only the notes it actually used. The last-used date is kept out of the sidecar's content hash, so
recording a use never makes the sidecar stale.

### 792a4n1 UAT Scenarios

```gherkin
Scenario: 792a4n1a A note's use is recorded
  Given a note with a current sidecar
  When the user runs engram activate --note <basename>
  Then the sidecar's last_used is today, its vectors and content hash are unchanged, the note file is unchanged, and the command succeeds

Scenario: 792a4n1b A note without a sidecar still counts
  Given a note that has no sidecar
  When the user runs engram activate --note <basename>
  Then the command succeeds and writes no sidecar

Scenario: 792a4n1c A missing note fails only its own ref
  Given no parent is configured, a note 3.2026-09-20.local, and no note 7.2026-09-01.gone
  When the user runs engram activate --note 3.2026-09-20.local --note 7.2026-09-01.gone.md
  Then 3.2026-09-20.local's last_used is today, stderr names 7.2026-09-01.gone.md as not found, and the command fails saying 1 of 2 refs failed

Scenario: 792a4n1d Amend records a use
  Given a note with a current sidecar
  When the user runs engram amend --target <note> --activate
  Then the sidecar's last_used is today
```

## 792a4o Concurrent revisions never lose an update

Requirement: [792a4o Concurrent revisions never lose an update](#792a4o-concurrent-revisions-never-lose-an-update) Two vault writers — `engram amend`, `resituate`, `activate` or `learn` —
running against one vault at the same time MUST NOT lose either one's change to a note, and MUST
NOT leave a note's sidecar holding the vectors of the note's earlier text.

Rationale: before every vault writer took the vault lock, two concurrent writers corrupted vault and
index state in production. Many sub-recalls can amend and activate the same vault in parallel.

### 792a4o1 UAT Scenarios

```gherkin
Scenario: 792a4o1a Concurrent amends both take effect
  Given a fact note
  When engram amend --target <note> --subject s2 and engram amend --target <note> --chunk-source a#1 start at the same time
  Then the note carries both subject: s2 and a#1 in sources:

Scenario: 792a4o1b An activation does not undo a re-embed
  Given a fact note with a current sidecar
  When engram amend --target <note> --object new and engram activate --note <note> start at the same time
  Then the sidecar holds the vectors of the amended text
```
