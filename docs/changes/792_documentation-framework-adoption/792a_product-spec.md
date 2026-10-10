# Documentation Framework Adoption - Product Spec

## 792a1 Problem Statement

Joe and the LLM agents that work on engram can't find engram's current behavior in one
authoritative, traceable place. That behavior is described across legacy documents that predate the
`docs/` framework: `docs/deprecated/` (ADRs, C4 views, invariants, design and research notes),
`openspec/specs/`, and `openspec/changes/archive/`. These documents use incompatible formats and
ID schemes. They mix removed features with shipped ones, and live files still link to paths that no
longer exist. No test is tied to a specification requirement, so nobody can check which behavior is
verified. Each capability and component needs one specification in the framework's format, with
stable issue-rooted IDs. The tests need to be traceable to the scenarios they verify, and no legacy
documentation can be left behind.

Capability sections `792a2` through `792a23` are added by the nodes of
[`792c_implementation-plan.md`](./792c_implementation-plan.md), using the template in its section
`792c3`.

## 792a2 Added: vault-identity

Canonical target: `docs/specs/product/vault-identity.md` (title `Vault Identity - Product Spec`).

Every engram command that reads or writes notes works against one vault: a directory of Markdown
notes on the local filesystem. This capability covers where that vault is, how it comes into
existence, and the stable identity that keeps a copied, cloned or self-parented vault from
exchanging notes under another vault's name. The chunk-only commands `prune`, `query-chunks` and
`show-chunk` resolve no vault. Exchange with a parent vault itself is the parent-sync capability.

### 792a2a Vault location

Requirement: 792a2a Engram MUST use, as the vault path, the first of these that is set: the
`--vault` flag, the `ENGRAM_VAULT_PATH` environment variable, `$XDG_DATA_HOME/engram/vault`, and
`~/.local/share/engram/vault`.

#### 792a2a1 UAT Scenarios

```gherkin
Scenario: 792a2a1a The flag wins over the environment
  Given ENGRAM_VAULT_PATH is set to /env/vault
  When the user runs an engram command with --vault /flag/vault
  Then the command uses /flag/vault

Scenario: 792a2a1b The environment variable is used without a flag
  Given ENGRAM_VAULT_PATH is set to /env/vault
  When the user runs an engram command without --vault
  Then the command uses /env/vault

Scenario: 792a2a1c The XDG data directory is the default
  Given neither --vault nor ENGRAM_VAULT_PATH is set and XDG_DATA_HOME is /data
  When the user runs an engram command
  Then the command uses /data/engram/vault

Scenario: 792a2a1d The home directory is the last fallback
  Given neither --vault, ENGRAM_VAULT_PATH nor XDG_DATA_HOME is set
  When the user runs an engram command
  Then the command uses ~/.local/share/engram/vault
```

### 792a2b Vault name

Requirement: 792a2b Engram MUST record a vault name in the `vault:` field of every note it writes,
taken from the first of these that is set: the `--vault-name` flag, the `ENGRAM_VAULT_NAME`
environment variable, and the default `personal`.

#### 792a2b1 UAT Scenarios

```gherkin
Scenario: 792a2b1a The default vault name is personal
  Given neither --vault-name nor ENGRAM_VAULT_NAME is set
  When the user writes a note with engram learn
  Then the note's vault: field is personal

Scenario: 792a2b1b The environment names the vault
  Given ENGRAM_VAULT_NAME is set to team
  When the user writes a note with engram learn without --vault-name
  Then the note's vault: field is team

Scenario: 792a2b1c The flag wins over the environment
  Given ENGRAM_VAULT_NAME is set to team
  When the user writes a note with engram learn --vault-name org
  Then the note's vault: field is org
```

### 792a2c Flat vault layout

Requirement: 792a2c Engram MUST NOT treat a file inside a vault subdirectory as a note; notes live
at the vault root.

Rationale: the 2026-06-12 flat-vault migration retired the `Permanent/` and `MOCs/` tiers, so a
note's place in the vault is carried by its Luhmann ID, not by a folder.

#### 792a2c1 UAT Scenarios

```gherkin
Scenario: 792a2c1a A Markdown file in a subdirectory is not a note
  Given a vault with one note at its root and one Markdown file in a subdirectory
  When the user runs engram embed status on the vault
  Then it reports a total of 1 note
```

### 792a2d Starter README

Requirement: 792a2d Engram MUST NOT treat the untouched starter `README.md` it writes into a new
vault as a note, and MUST treat any other root `README.md` as a note.

Rationale: a freshly created vault must report itself clean; the starter README is recognized by
its exact content, so a user's own README, or an edited starter, stays a normal note.

#### 792a2d1 UAT Scenarios

```gherkin
Scenario: 792a2d1a A fresh vault has no notes
  Given a vault that engram has just created
  When the user runs engram embed status and engram check on it
  Then embed status reports a total of 0 notes and check reports no note missing a sidecar

Scenario: 792a2d1b A user's README is a note
  Given a vault whose README.md the user has replaced with their own text
  When the user runs engram embed status and engram check on it
  Then embed status reports a total of 1 note and check reports 1 note missing a sidecar
```

### 792a2e First-use vault creation

Requirement: 792a2e Every command that resolves a vault path MUST, when no vault exists at that
path, create one before the command runs, whether or not `ENGRAM_PARENT` is set, and print exactly
one stderr line, `engram: created new vault at <path>`. The new vault contains `.obsidian/app.json`,
a `.gitignore`, a starter `README.md`, a vault ID (792a2h) and the `.engram/` state directory
(792a2k).

Rationale: Joe chose creation by any command (#766) over creating only on `update` or warning
without creating. The single stderr line makes a mistyped `--vault` or `ENGRAM_VAULT_PATH` visible.
The `.obsidian/` starter lets Obsidian open the directory as a vault.

#### 792a2e1 UAT Scenarios

```gherkin
Scenario: 792a2e1a A read command creates the vault
  Given no vault exists at the resolved path
  When the user runs engram query --phrase x
  Then an initialized vault exists at that path, stderr carries one line naming it, and the query returns an empty result without error

Scenario: 792a2e1b amend and activate fail only on their own merits
  Given no vault exists at the resolved path
  When the user runs engram amend or engram activate on a target that does not exist
  Then the vault is created first, and the command fails because the target is not found, never on a vault lock error

Scenario: 792a2e1c A child host gets its own vault
  Given a host with ENGRAM_PARENT set and no vault at its resolved local path
  When the user runs engram query for the first time
  Then a local vault is created at the resolved local path, separate from the parent's vault

Scenario: 792a2e1d A second run creates nothing
  Given a vault that an engram command has just created
  When the user runs the same command again
  Then no creation line is printed and the vault ID is unchanged
```

### 792a2f Existing vaults are left alone

Requirement: 792a2f A command MUST NOT create or modify any file, or print the creation line, when it
runs read-only against an existing vault while no parent is configured.

Rationale: `.engram-vault-id` is tracked, so stamping it on read-only use would dirty a git-backed
vault on every host that merely reads it.

#### 792a2f1 UAT Scenarios

```gherkin
Scenario: 792a2f1a An existing vault is untouched by read-only use
  Given an existing vault with no vault ID and no parent configured
  When the user runs a read-only command against it
  Then no vault file is created or modified, and no creation line is printed
```

### 792a2g Interrupted creation is repaired

Requirement: 792a2g When a vault has the `.engram/` state directory but no vault ID, because its
first-use creation stopped part-way, the next command that resolves it MUST give it a vault ID and
print exactly one stderr line, `engram: stamped the missing vault ID of <path>`.

Rationale: creation writes `.engram/` before the ID, so `.engram/` marks a vault engram created,
and a pre-existing vault with neither stays untouched (792a2f).

#### 792a2g1 UAT Scenarios

```gherkin
Scenario: 792a2g1a A half-created vault gets its ID
  Given a vault with a .engram/ directory and no .engram-vault-id
  When the user runs any vault-resolving command
  Then .engram-vault-id is created and stderr carries one line saying the missing vault ID was stamped
```

### 792a2h Stable vault ID

Requirement: 792a2h Every vault MUST have a random vault ID of 32 lowercase hex characters in the
tracked file `<vault>/.engram-vault-id`. A new vault gets it when it is created. An existing vault
without one gets it only when `engram serve` starts on it or when a command first contacts a parent
from it. The ID changes only through `engram vault-id --regenerate`.

Rationale: links to a parent's notes are keyed by the parent's vault ID, never by URL, so a hostname
or IP change breaks nothing. Lazy stamping keeps read-only use of an existing vault from creating a
tracked file (792a2f).

#### 792a2h1 UAT Scenarios

```gherkin
Scenario: 792a2h1a serve stamps an existing vault
  Given an existing vault that has no .engram-vault-id
  When engram serve starts on it
  Then the file is created with a new ID, and later starts reuse it unchanged

Scenario: 792a2h1b The first parent contact stamps an existing vault
  Given an existing vault that has no .engram-vault-id and ENGRAM_PARENT set
  When the user runs engram query
  Then .engram-vault-id is created and no location warning is printed

Scenario: 792a2h1c Concurrent creation converges
  Given an existing vault that has no .engram-vault-id
  When two processes create the ID for it at the same moment
  Then exactly one file is written, and both processes use its ID
```

### 792a2i Uncommitted vault ID notice

Requirement: 792a2i `engram update` MUST print a notice asking for a one-time vault commit while
`.engram-vault-id` exists in a git-backed vault and is untracked, ignored, staged, or modified
relative to the last commit, and MUST print no such notice otherwise.

Rationale: the ID file's first creation in an existing git-backed vault is a deliberate one-time
commit (`vault: add vault id`).

#### 792a2i1 UAT Scenarios

```gherkin
Scenario: 792a2i1a Uncommitted ID is flagged
  Given a git-backed vault whose .engram-vault-id is untracked
  When the user runs engram update
  Then its report includes a notice asking for a one-time vault commit

Scenario: 792a2i1b An ignored ID is flagged
  Given a git-backed vault whose .engram-vault-id is ignored by its .gitignore
  When the user runs engram update
  Then its report includes the commit notice

Scenario: 792a2i1c A committed ID is not flagged
  Given a git-backed vault whose .engram-vault-id is committed and unmodified
  When the user runs engram update
  Then its report includes no commit notice

Scenario: 792a2i1d A vault outside git is not flagged
  Given a vault that is not a git work tree and has a .engram-vault-id
  When the user runs engram update
  Then its report includes no commit notice
```

### 792a2j Corrupt vault ID

Requirement: 792a2j When `.engram-vault-id` is empty or does not hold 32 lowercase hex characters,
engram MUST report it as invalid with a message naming `engram vault-id --regenerate`, and MUST
leave the file unchanged. `engram serve` still starts, and a merged query still returns its local
results.

Rationale: a partial write or a git conflict can corrupt the file; refusing to serve would turn it
into a host outage.

#### 792a2j1 UAT Scenarios

```gherkin
Scenario: 792a2j1a serve starts despite a corrupt ID
  Given a vault whose .engram-vault-id is empty or holds a git conflict marker
  When engram serve starts on it
  Then it warns naming engram vault-id --regenerate, serves, and leaves the file unchanged

Scenario: 792a2j1b A merged query warns and still runs
  Given a vault whose .engram-vault-id holds junk and ENGRAM_PARENT set
  When the user runs engram query
  Then stderr says the vault ID could not be stamped and names engram vault-id --regenerate, and the local results are returned

Scenario: 792a2j1c vault-id reports the corrupt file
  Given a vault whose .engram-vault-id holds junk
  When the user runs engram vault-id
  Then it fails with an error saying the vault ID is invalid
```

### 792a2k Exchange state outside tracked files

Requirement: 792a2k Engram MUST keep transient exchange state in `<vault>/.engram/`, whose own
`.gitignore` holds only the pattern `*`, and MUST NOT write exchange state to the vault root or
modify the vault's root `.gitignore`.

Rationale: putting the outbox in the vault root and appending it to the tracked `.gitignore` would
dirty the vault's repository on every host. Keeping state under `$XDG_DATA_HOME`, keyed by vault
path, would orphan it when the vault moves, and it would not travel with the vault.

#### 792a2k1 UAT Scenarios

```gherkin
Scenario: 792a2k1a State does not dirty the vault's git
  Given a git-backed vault
  When an offer is queued in it
  Then git status --porcelain in the vault shows no change caused by exchange state

Scenario: 792a2k1b The root .gitignore is untouched
  Given an existing vault with its own root .gitignore
  When engram stamps the vault's ID and creates .engram/
  Then the root .gitignore is byte-for-byte unchanged
```

### 792a2l Inspecting the vault ID

Requirement: 792a2l `engram vault-id` with no flag MUST print the vault ID, or `none`, and the
location check result (`ok`, `missing`, `mismatch` or `no vault id`), MUST also print a remedy line
naming `engram vault-id --regenerate` and `engram vault-id --claim` when the check is `missing` or
`mismatch`, and MUST write nothing.

#### 792a2l1 UAT Scenarios

```gherkin
Scenario: 792a2l1a A healthy vault reports its ID
  Given a vault that engram created at its current path
  When the user runs engram vault-id
  Then stdout shows its vault ID and location: ok, and stderr is empty

Scenario: 792a2l1b A moved vault reports a mismatch with both remedies
  Given a vault that was moved after engram created it
  When the user runs engram vault-id
  Then stdout shows location: mismatch and names both engram vault-id --claim and engram vault-id --regenerate

Scenario: 792a2l1c A vault without an ID is reported without writing
  Given an existing vault with no vault ID
  When the user runs engram vault-id
  Then stdout shows location: no vault id, and no file in the vault is created or changed
```

### 792a2m Regenerating the vault ID

Requirement: 792a2m `engram vault-id --regenerate` MUST give the vault a new vault ID, record the
vault's current location for it, and change no note.

Rationale: this is the remedy for a copy. Links point at parent vault IDs and stay valid, and
queued offers carry the new origin ID when they are sent.

#### 792a2m1 UAT Scenarios

```gherkin
Scenario: 792a2m1a Regenerate gives a copy its own identity
  Given a copied vault
  When the user runs engram vault-id --regenerate in it
  Then its ID differs from the original's, exchange resumes, and no note file changed
```

### 792a2n Claiming a moved vault

Requirement: 792a2n `engram vault-id --claim` MUST record the vault's current location for its
existing vault ID without changing the ID, and MUST fail when the vault has no vault ID.

Rationale: this is the remedy for the same vault moved or re-cloned.

#### 792a2n1 UAT Scenarios

```gherkin
Scenario: 792a2n1a Claim accepts a moved vault
  Given a vault that was moved to a new path
  When the user runs engram vault-id --claim
  Then the ID is unchanged, engram vault-id then reports location: ok, and exchange resumes

Scenario: 792a2n1b Claim needs an ID
  Given an existing vault with no vault ID
  When the user runs engram vault-id --claim
  Then it fails with an error saying the vault has no vault ID
```

### 792a2o Regenerate and claim are exclusive

Requirement: 792a2o `engram vault-id` MUST refuse `--regenerate` and `--claim` together and leave
the vault ID unchanged.

#### 792a2o1 UAT Scenarios

```gherkin
Scenario: 792a2o1a Both flags are refused
  Given a vault with a vault ID
  When the user runs engram vault-id --regenerate --claim
  Then it fails naming --regenerate, and the vault ID is unchanged
```

### 792a2p Copied and cloned vaults pause exchange

Requirement: 792a2p Before sending an offer or pulling a note down, a command MUST check that the
vault's location record matches its current vault ID and canonical path. When the record is missing
(a fresh `git clone`) or differs (a `cp -R` or a move), the command sends no offer and pulls nothing
down, still writes locally and queues offers, and prints exactly one warning naming both
`engram vault-id --regenerate` and `engram vault-id --claim`. A merged query still runs, with the
same warning, and `engram serve` prints the same warning and starts anyway.

Rationale: the check catches two children sharing an ID, one cloned from the other, at the clone's
first exchange, before any offer carries a shared origin. `.engram/` is untracked, so a clone has no
record, while a copy or a move has one for another path. `engram serve` warns rather than refusing,
because refusing would turn a false positive into a host outage: launchd restarts it, it refuses
again, children back off, and offers pile up. The record holds no hostname, because DHCP, `.local`
renames and containers change hostnames without the vault moving.

#### 792a2p1 UAT Scenarios

```gherkin
Scenario: 792a2p1a A cloned child is caught before it offers
  Given vault A is cloned with git to a new host with ENGRAM_PARENT set
  When the clone runs engram learn
  Then the note is written locally and queued, no offer is sent, and the warning names both engram vault-id remedies

Scenario: 792a2p1b A cloned child's merged query still runs
  Given a vault with ENGRAM_PARENT set whose location record is missing
  When the user runs engram query
  Then the query returns its results, and stderr carries one warning naming both engram vault-id remedies

Scenario: 792a2p1c serve starts despite a mismatched record
  Given a vault whose location record is missing or differs
  When engram serve starts on it
  Then it prints the warning and serves normally

Scenario: 792a2p1d A symlinked path is not a false positive
  Given a vault reached through a symlink to its recorded path
  When the location check runs
  Then the check passes

Scenario: 792a2p1e A hostname change is not a false positive
  Given a host whose name changed while the vault path did not
  When the location check runs
  Then the check passes
```

### 792a2q A vault does not exchange with itself

Requirement: 792a2q When a parent reports the same vault ID as the local vault, a command MUST NOT
merge, offer or pull down, and MUST behave as if no parent were configured, apart from printing
exactly one warning naming `engram vault-id --regenerate`.

Rationale: a child cloned from its parent whose location record was copied too, at a matching path,
passes the location check (792a2p); this guard catches it instead.

#### 792a2q1 UAT Scenarios

```gherkin
Scenario: 792a2q1a A copied vault served back to itself
  Given a copy of vault V is served and V's environment points ENGRAM_PARENT at it
  When the user runs engram query in V
  Then it returns local-only results with a warning naming engram vault-id --regenerate, and no offer is sent
```

## 792a3 Added: note-writing

Canonical target: `docs/specs/product/note-writing.md` (title `Note Writing - Product Spec`).

`engram learn` writes new notes into the vault: `fact`, `feedback` and `runbook` notes, each placed
in the vault's Luhmann hierarchy, and `qa` pairs of a question note and an answer note. Each note's
`vault:` field takes the vault name of 792a2b. Changing an existing note is the note-revision
capability; deciding which lessons to capture, and how to phrase them, belongs to the learn and
write-memory skills.

### 792a3a Fact notes

Requirement: 792a3a `engram learn fact` MUST write one note named
`<luhmann>.<YYYY-MM-DD>.<slug>.md` at the vault root, whose frontmatter carries `type: fact`, the
situation, subject, predicate and object, the Luhmann ID, the creation date and the source, and
whose body reads `Information learned: when in <situation>, <subject> <predicate> <object>.` with
any leading `When ` dropped from the situation, and MUST print the note's path on stdout.

#### 792a3a1 UAT Scenarios

```gherkin
Scenario: 792a3a1a A fact note is written
  Given an empty vault
  When the user runs engram learn fact --slug ctx-fact --position top --source "smoke test" --situation "concurrent Go code" --subject goroutines --predicate "leak when" --object "ctx is ignored"
  Then stdout prints the path of 1.<today>.ctx-fact.md at the vault root, whose frontmatter has type: fact and whose body reads "Information learned: when in concurrent Go code, goroutines leak when ctx is ignored."

Scenario: 792a3a1b A leading When is not doubled
  Given an empty vault
  When the user runs engram learn fact with --situation "When testing" --subject s --predicate p --object o
  Then the note's body reads "Information learned: when in testing, s p o."
```

### 792a3b Feedback notes

Requirement: 792a3b `engram learn feedback` MUST write one note with the same filename shape, whose
frontmatter carries `type: feedback`, the situation, behavior, impact and action, the Luhmann ID,
the creation date and the source, and whose body reads `Lesson learned: when <situation>, <action>.`
with any leading `When ` dropped from the situation, and MUST print the note's path on stdout.

#### 792a3b1 UAT Scenarios

```gherkin
Scenario: 792a3b1a A feedback note is written
  Given an empty vault
  When the user runs engram learn feedback --slug ctx-rule --position top --source "smoke test" --situation "writing concurrent Go code" --behavior "ignoring ctx" --impact "leaks goroutines" --action "check ctx.Done()"
  Then stdout prints the path of 1.<today>.ctx-rule.md, whose frontmatter has type: feedback and whose body reads "Lesson learned: when writing concurrent Go code, check ctx.Done()."

Scenario: 792a3b1b A leading When is not doubled
  Given an empty vault
  When the user runs engram learn feedback with --situation "When x" --action a
  Then the note's body reads "Lesson learned: when x, a."
```

### 792a3c Runbook notes

Requirement: 792a3c `engram learn runbook` MUST write one note with the same filename shape, whose
frontmatter carries `type: runbook`, the situation, `done_when`, the Luhmann ID, the creation date
and the source, and whose body is the `--body` steps verbatim, and MUST print the note's path on
stdout.

Rationale: confirmed approaches used to live in a feedback note's single `action` sentence, with no
way to represent an ordered procedure or an ending condition. A runbook answers exactly three
questions: when to use it (`situation`, the retrieval handle), what the steps are (the body, which
may link the fact and feedback notes to consider), and what should be true when done (`done_when`,
which also makes runbooks scorable for later efficacy tracking). Joe chose the name `runbook` over
`procedure`, `function` and `strategy` (vault note 789). Runbooks use the fact and feedback filename
and Luhmann placement rather than the `qa` scheme: a top-level position needs no target, so "no
related note yet" never required a separate scheme, and `qa` is the one kind built to skip the
hierarchy and be partly excluded from retrieval.

#### 792a3c1 UAT Scenarios

```gherkin
Scenario: 792a3c1a A runbook note is written
  Given an empty vault
  When the user runs engram learn runbook --slug rb --position top --source src --situation "releasing a build" --done-when "the tag is pushed" --body "1. Run the checks"
  Then stdout prints the path of 1.<today>.rb.md, whose frontmatter has type: runbook, situation and done_when, and whose body is "1. Run the checks"
```

### 792a3d Runbook red flags and triggers

Requirement: 792a3d `engram learn runbook` MUST write each `--red-flag` value, in order, to a
`red_flags:` list and each `--trigger` value, in order, to a `triggers:` list after it, and MUST
write neither field when its flag is absent.

Rationale: a red flag names a failure mode specific to this procedure that a general "follow the
steps" rule would not catch; the general floor is not restated on each runbook. How triggers
surface a runbook belongs to the memory-query capability.

#### 792a3d1 UAT Scenarios

```gherkin
Scenario: 792a3d1a Red flags are kept in order
  Given an empty vault
  When the user runs engram learn runbook with --red-flag "A" --red-flag "B"
  Then the note's red_flags: lists A then B

Scenario: 792a3d1b Triggers follow the red flags in order
  Given an empty vault
  When the user runs engram learn runbook with --red-flag "A" --trigger "cue one" --trigger "cue two"
  Then the note's triggers: lists "cue one" then "cue two", after red_flags:

Scenario: 792a3d1c Neither flag means neither field
  Given an empty vault
  When the user runs engram learn runbook with no --red-flag and no --trigger
  Then the note is written with no red_flags: and no triggers: field
```

### 792a3e No task-type or input fields

Requirement: 792a3e `engram learn runbook` MUST NOT accept a task-type or input-signature field.

Rationale: a `task_type` field with a query pre-filter was planned on the strength of benchmark gains
for a whole bundled system, none of which isolated the classification step, so it could not justify a
new ranking mechanism and its regression risk; the field was dropped with the filter. An input
signature was considered during the #728 adapter experiment and rejected.

#### 792a3e1 UAT Scenarios

```gherkin
Scenario: 792a3e1a A task type is refused
  Given an existing vault
  When the user runs engram learn runbook with --task-type foo
  Then it fails naming --task-type as an undefined flag, and no note is written
```

### 792a3f Required content

Requirement: 792a3f `engram learn fact`, `feedback` and `runbook` MUST refuse to write a note whose
`--situation` is missing or blank, and `engram learn runbook` MUST also refuse one whose
`--done-when` is missing or blank, with an error naming the missing flag.

Rationale: the situation is the note's retrieval handle. Recall finds a note by a query phrased as
its situation, so a note without one cannot be recalled that way.

#### 792a3f1 UAT Scenarios

```gherkin
Scenario: 792a3f1a A missing situation is refused
  Given an existing vault
  When the user runs engram learn fact without --situation
  Then it fails naming --situation, and no note is written

Scenario: 792a3f1b A blank situation is refused
  Given an existing vault
  When the user runs engram learn feedback with --situation "  "
  Then it fails saying --situation is required, and no note is written

Scenario: 792a3f1c A runbook without done-when is refused
  Given an existing vault
  When the user runs engram learn runbook without --done-when
  Then it fails naming --done-when, and no note is written
```

### 792a3g Argument validation

Requirement: 792a3g `engram learn fact`, `feedback` and `runbook` MUST require `--source` and a
`--slug` matching `[a-z0-9-]+`, and MUST refuse, before writing anything, a `--project` that is not
kebab-case, an `--issue` containing whitespace, a `--tag` that is not `<family>` or
`<family>/<value>` with kebab-case segments, a `--trigger` shorter than three characters after
trimming, a `--tier` other than `L1`, `L2` or `L3`, and a `--supersedes` that is not
`<note>|<type>|<claim>` with a type of `updates`, `narrows` or `refutes`.

#### 792a3g1 UAT Scenarios

```gherkin
Scenario Outline: 792a3g1a An invalid argument is refused before any write
  Given an existing vault
  When the user runs engram learn <kind> with <argument>
  Then it fails with an error saying <problem>, and no note is written

  Examples:
    | kind    | argument               | problem                                         |
    | fact    | no --source            | --source is a missing required flag             |
    | fact    | no --slug              | slug is required                                |
    | fact    | --slug X_y             | slug must match [a-z0-9-]+                      |
    | fact    | --project A            | project slug must match [a-z0-9-]+              |
    | fact    | --issue "a b"          | issue must be non-empty with no whitespace      |
    | fact    | --tag "A B"            | tag must be <family> or <family>/<value>        |
    | runbook | --trigger ab           | trigger must be at least 3 characters           |
    | fact    | --tier L9              | tier must be L2, or L1 or L3                    |
    | fact    | --supersedes "a\|bogus\|c" | supersedes type must be updates, narrows or refutes |
```

### 792a3h Optional metadata

Requirement: 792a3h `engram learn fact`, `feedback` and `runbook` MUST write a given `--project` to
`project:`, `--issue` to `issue:`, each `--tag` to `tags:`, each `--chunk-source` to `sources:`, and
each `--supersedes` both to a `supersedes:` entry and to a body line
`Supersedes: [[<note>]] — <type>: <claim>`, and MUST omit each of these fields when its flag is
absent.

Rationale: `learn` records `--chunk-source` IDs without checking the chunk index, because the chunks
a note is written from may not be indexed yet when it is written. `engram amend`, which runs later,
checks them.

#### 792a3h1 UAT Scenarios

```gherkin
Scenario: 792a3h1a Optional metadata is written
  Given an empty vault
  When the user runs engram learn runbook with --project proj --issue 12 --tag fam/val --chunk-source "a#b" --supersedes "1.x|updates|claim"
  Then the note carries project: proj, issue: "12", tags: [fam/val], sources: [a#b], a supersedes: entry for 1.x of type updates, and the body line "Supersedes: [[1.x]] — updates: claim"

Scenario: 792a3h1b Absent metadata writes no field
  Given an empty vault
  When the user runs engram learn fact with none of those flags
  Then the note has no project:, issue:, tags:, sources: or supersedes: field
```

### 792a3i Tier field

Requirement: 792a3i `engram learn fact`, `feedback` and `runbook` MUST write `tier: L2` when `--tier`
is absent, and the given value when `--tier` is `L1`, `L2` or `L3`.

Rationale: the three-tier model is retired; `L1` and `L3` are still accepted so that hand-edited notes
carrying them stay valid.

#### 792a3i1 UAT Scenarios

```gherkin
Scenario: 792a3i1a The tier defaults to L2
  Given an empty vault
  When the user runs engram learn fact or feedback without --tier
  Then the note carries tier: L2

Scenario: 792a3i1b A legacy tier is written as given
  Given an empty vault
  When the user runs engram learn fact with --tier L3
  Then the note carries tier: L3
```

### 792a3j Luhmann placement

Requirement: 792a3j `engram learn fact`, `feedback` and `runbook` MUST give the new note, for
`--position top`, the next unused top-level number; for `--position continuation`, the next unused
child ID of `--target`; and for `--position sibling`, the next unused child ID of the target's
parent. A child of an ID ending in a number ends in the next unused letter (`a` to `z`, then `aa`),
and a child of an ID ending in a letter ends in the next unused number.

Rationale: runbooks take part in the hierarchy exactly as facts and feedback do, with no
kind-specific opt-out.

#### 792a3j1 UAT Scenarios

```gherkin
Scenario: 792a3j1a Top takes the next top-level number
  Given a vault whose notes have IDs 1, 2 and 2a
  When the user runs engram learn fact --position top
  Then the new note's ID is 3

Scenario: 792a3j1b A continuation of a number takes the next letter
  Given a vault whose notes have IDs 1 and 1a
  When the user runs engram learn fact --position continuation --target 1
  Then the new note's ID is 1b

Scenario: 792a3j1c A continuation of a letter takes the next number
  Given a vault whose note has ID 1a
  When the user runs engram learn runbook --position continuation --target 1a
  Then the new note's ID is 1a1

Scenario: 792a3j1d A sibling takes the next child of the target's parent
  Given a vault whose notes have IDs 1, 1a and 1a1
  When the user runs engram learn feedback --position sibling --target 1a1
  Then the new note's ID is 1a2

Scenario: 792a3j1e Letters roll over from z to aa
  Given a vault whose notes have IDs 1 and 1a through 1z
  When the user runs engram learn fact --position continuation --target 1
  Then the new note's ID is 1aa
```

### 792a3k Placement refusals

Requirement: 792a3k `engram learn fact`, `feedback` and `runbook` MUST refuse, with no note written,
a missing or unknown `--position`, a `continuation` or `sibling` without `--target`, a `--target`
that is not a Luhmann ID, and a `sibling` of a top-level ID, which must use `--position top`
instead.

#### 792a3k1 UAT Scenarios

```gherkin
Scenario: 792a3k1a A missing position is refused
  Given an existing vault
  When the user runs engram learn fact without --position
  Then it fails saying the position must be top, continuation or sibling, and no note is written

Scenario: 792a3k1b A continuation without a target is refused
  Given an existing vault
  When the user runs engram learn fact --position continuation without --target
  Then it fails saying a target is required, and no note is written

Scenario: 792a3k1c A sibling of a top-level note is refused
  Given a vault whose note has ID 1
  When the user runs engram learn fact --position sibling --target 1
  Then it fails saying a sibling of a top-level note requires --position top, and no note is written

Scenario: 792a3k1d A malformed target is refused
  Given an existing vault
  When the user runs engram learn fact --position continuation --target abc
  Then it fails saying the ID must start with a digit, and no note is written
```

### 792a3l Placement under an absent target

Requirement: 792a3l `engram learn` MAY place a note under a `--target` that names no existing note.

### 792a3m Concurrent writes never collide

Requirement: 792a3m Two `engram learn` runs against one vault at the same time MUST give their notes
distinct Luhmann IDs and MUST NOT overwrite each other's note.

Rationale: before every vault writer took the vault lock, two concurrent writers corrupted vault and
index state in production. The guarantee matters most when many sub-recalls write the vault in
parallel.

#### 792a3m1 UAT Scenarios

```gherkin
Scenario: 792a3m1a Concurrent learns get distinct IDs
  Given an existing vault
  When two engram learn fact --position top runs start at the same time
  Then two notes exist with different Luhmann IDs, each holding its own content
```

### 792a3n Embed on write

Requirement: 792a3n Every note `engram learn` writes MUST have a sibling `.vec.json` embedding
sidecar, named after the note, by the time the command returns. When embedding the note fails, the
note MUST still be written, and the command MUST print a warning naming the note and succeed.

Rationale: each note carries its own sidecar, so there is no separate index to build or keep in
sync. A missing sidecar is recoverable later, because the note itself was written in one atomic
step.

#### 792a3n1 UAT Scenarios

```gherkin
Scenario: 792a3n1a A learned note has its sidecar
  Given an empty vault
  When the user runs engram learn fact, feedback, runbook or qa
  Then every note written has a .vec.json sidecar beside it, and engram check reports no note missing a sidecar

Scenario: 792a3n1b A failed embedding keeps the note
  Given embedding fails for the note being learned
  When the user runs engram learn fact
  Then the note is written, stderr carries a warning naming it, and the command succeeds
```

### 792a3o Repository provenance

Requirement: 792a3o `engram learn fact`, `feedback` and `runbook` MUST stamp `repo:` with the
working directory's git `origin` remote URL, or, when there is no `origin` remote, with the basename
of the working directory's git top-level directory, and MUST omit `repo:` outside a git repository.

Rationale: the free-text `source:` field says what a note came from, but not which repository,
person or vault produced it. Structured, auto-detected provenance lets filtering, attribution and
parent exchange rely on a note's origin. Detection is self-reported by the writer's environment,
not verified. Being outside a git repository is not an error, so the field is simply omitted.

#### 792a3o1 UAT Scenarios

```gherkin
Scenario: 792a3o1a The origin remote names the repo
  Given a working directory inside a git repository whose origin is git@example.com:o/r.git
  When the user runs engram learn fact
  Then the note's repo: is git@example.com:o/r.git

Scenario: 792a3o1b Without an origin the top-level directory names the repo
  Given a working directory inside a git repository named proj with no origin remote
  When the user runs engram learn fact
  Then the note's repo: is proj

Scenario: 792a3o1c Outside a git repository there is no repo
  Given a working directory outside any git repository
  When the user runs engram learn fact
  Then the note has no repo: field
```

### 792a3p User provenance

Requirement: 792a3p `engram learn fact`, `feedback` and `runbook` MUST stamp `user:` with
`git config user.email`, or, when that resolves to nothing, with the OS username. When neither
resolves, the note MUST be written without a `user:` field, never with an empty one, and the command
MUST print one warning that user detection resolved empty.

#### 792a3p1 UAT Scenarios

```gherkin
Scenario: 792a3p1a The git email names the user
  Given git config user.email resolves to a@example.com
  When the user runs engram learn fact
  Then the note's user: is a@example.com

Scenario: 792a3p1b The OS username is the fallback
  Given git config user.email resolves to nothing and the OS username is joe
  When the user runs engram learn fact
  Then the note's user: is joe

Scenario: 792a3p1c No detectable user
  Given neither git config user.email nor the OS username resolves
  When the user runs engram learn fact
  Then the note has no user: field, vault: is stamped as usual, and stderr carries one warning that user detection resolved empty
```

### 792a3q Question-and-answer pairs

Requirement: 792a3q `engram learn qa` MUST write two notes at the vault root,
`qa.<YYYY-MM-DD>.<slug>.q.md` and `qa.<YYYY-MM-DD>.<slug>.a.md`, and print both paths, the question's
first. The question note's frontmatter carries `type: qa-question`, `date`,
`answered_by: <answer basename>` and `source`, and its body is the question followed by the line
`Answered by: [[<answer basename>]]`. The answer note's frontmatter carries `type: qa-answer`,
`date`, `answers: <question basename>`, `certainty`, `contributors` when any are given, and
`source`, and its body is the answer, from `--answer` or the file named by `--answer-file`, followed
by the line `Answers: [[<question basename>]]` and, when contributors are given, the line
`Contributors: [[<first>]], [[<second>]]`.

Rationale: an answer is a reasoned conclusion with provenance, while a question is situational
wording that measurably loses retrieval against notes with content: re-anchoring notes to their
questions gave no delivery benefit and lost retrieval in 10 of 10 cases. So the pair is linked both
ways, the answer competes for retrieval, and the question is reachable only through its answer
(792a3t). An earlier design excluded both halves.

#### 792a3q1 UAT Scenarios

```gherkin
Scenario: 792a3q1a A pair is captured
  Given a vault holding note 1.2026-10-09.a-fact
  When the user runs engram learn qa --slug why-x --question "Why X?" --answer "Because." --source src --contributors 1.2026-10-09.a-fact
  Then stdout prints the paths of qa.<today>.why-x.q.md and then qa.<today>.why-x.a.md, the question note ends "Answered by: [[qa.<today>.why-x.a]]", and the answer note ends "Answers: [[qa.<today>.why-x.q]]" and "Contributors: [[1.2026-10-09.a-fact]]"

Scenario: 792a3q1b A pair without contributors
  Given an existing vault
  When the user runs engram learn qa without --contributors
  Then the answer note has no contributors: field and no Contributors: line

Scenario: 792a3q1c The answer can come from a file
  Given a file holding the answer text
  When the user runs engram learn qa with --answer-file naming it
  Then the answer note's body is that file's text
```

### 792a3r Question-and-answer validation

Requirement: 792a3r `engram learn qa` MUST refuse, before writing either note, a missing or invalid
`--slug`, a missing or blank `--question`, anything other than exactly one of `--answer` and
`--answer-file`, a missing `--source`, a `--certainty` other than `high`, `medium` or `low`, and a
`--contributors` value that is not the basename of a note in the vault. An absent `--certainty`
MUST be written as `medium`.

#### 792a3r1 UAT Scenarios

```gherkin
Scenario Outline: 792a3r1a An invalid pair is refused before any write
  Given an existing vault
  When the user runs engram learn qa with <argument>
  Then it fails with an error saying <problem>, and neither note is written

  Examples:
    | argument                       | problem                                      |
    | no --question                  | --question is required                       |
    | both --answer and --answer-file | exactly one of --answer or --answer-file is required |
    | neither --answer nor --answer-file | exactly one of --answer or --answer-file is required |
    | --certainty huge               | certainty must be high, medium, or low       |
    | --contributors nope            | contributor not found in vault               |
    | --slug X_y                     | slug must match [a-z0-9-]+                   |
    | no --source                    | --source is a missing required flag          |

Scenario: 792a3r1b Certainty defaults to medium
  Given an existing vault
  When the user runs engram learn qa without --certainty
  Then the answer note carries certainty: medium
```

### 792a3s Question-and-answer partial failure

Requirement: 792a3s When the answer note cannot be written, `engram learn qa` MUST remove the
question note it has just written and fail with an error saying the answer note failed.

#### 792a3s1 UAT Scenarios

```gherkin
Scenario: 792a3s1a A failed answer removes the question
  Given a vault that already holds a file named like today's answer note for slug why-x
  When the user runs engram learn qa --slug why-x
  Then it fails saying the answer note could not be written, and no question note for why-x remains
```

### 792a3t Question-and-answer retrieval

Requirement: 792a3t `engram query` MUST return answer notes as it returns any other note, and MUST
NOT return question notes.

#### 792a3t1 UAT Scenarios

```gherkin
Scenario: 792a3t1a The answer is found and the question is not
  Given a vault holding a pair whose question and answer both match a phrase
  When the user runs engram query with that phrase
  Then the answer note can appear in the results, and the question note never does
```

### 792a3u Vocabulary tagging by kind

Requirement: 792a3u The vault's write-time vocabulary tagging MUST apply to every fact, feedback,
runbook and answer note `engram learn` writes, and MUST NOT apply to question notes.

Rationale: the answer is the half that competes for retrieval, so it is tagged like any other note;
the question carries no vocabulary, so it stays reachable only through its answer.

#### 792a3u1 UAT Scenarios

```gherkin
Scenario: 792a3u1a A runbook is tagged
  Given a vault with a vocabulary term close to a runbook's text
  When the user runs engram learn runbook with that text
  Then the runbook carries the term's vocab tag

Scenario: 792a3u1b The answer is tagged and the question is not
  Given a vault with a vocabulary term close to both a question and its answer
  When the user runs engram learn qa with them
  Then the answer note carries the term's vocab tag, and the question note has no tags: field
```

## 792a4 Added: note-revision

Canonical target: `docs/specs/product/note-revision.md` (title `Note Revision - Product Spec`).

`engram amend`, `engram resituate` and `engram activate` change notes that already exist: amend
replaces a note's fields, supersessions and chunk sources, resituate rewrites its situation, and
activate records that it was used. Writing new notes is the note-writing capability. The curation
flags of `amend` (`--clear-pending`, `--discard`, `--into`, `--expect-hash`) belong to curate, and
`activate --parent`, a pull-down on a local miss, and the offers a revision queues for a parent
belong to parent-sync. The vocabulary tagging that follows a revision belongs to
controlled-vocabulary.

### 792a4a Note references

Requirement: 792a4a `engram amend --target` and `engram resituate --note` MUST accept a note's full
basename, the basename followed by `.md`, a `[[wikilink]]` to it (with or without a `|display`
part), or its bare Luhmann ID, and MUST fail with an error saying the note was not found, naming
the reference as given, and write nothing when no note matches.

#### 792a4a1 UAT Scenarios

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

### 792a4b Field replacement

Requirement: 792a4b `engram amend` MUST replace only the content fields it is given that belong to
the note's kind — `--situation`, `--subject`, `--predicate` and `--object` for a fact;
`--situation`, `--behavior`, `--impact` and `--action` for feedback; `--situation`, `--done-when`,
`--body`, and the whole `red_flags:` and `triggers:` lists from repeated `--red-flag` and
`--trigger` for a runbook — MUST rebuild a fact or feedback body formula from the updated fields,
and MUST print the note's path on stdout.

Rationale: a field that is not given keeps its value, so an agent can correct one part of a lesson
without restating the rest. The body formula repeats the situation and the lesson, so it is rebuilt
whenever they change.

#### 792a4b1 UAT Scenarios

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

### 792a4c Other note kinds are refused

Requirement: 792a4c `engram amend` MUST refuse, writing nothing, a note that is not a fact,
feedback or runbook note, or that has no frontmatter.

#### 792a4c1 UAT Scenarios

```gherkin
Scenario: 792a4c1a A question-and-answer note is refused
  Given a vault holding an answer note qa.2026-10-09.why.a
  When the user runs engram amend --target qa.2026-10-09.why.a --situation s
  Then it fails saying the note type is unknown, and the note is unchanged
```

### 792a4d Supersession

Requirement: 792a4d `engram amend --supersedes <note>|<type>|<claim>`, repeatable, MUST replace the
note's whole `supersedes:` list and all of its `Supersedes:` body lines with the given entries, and
`engram learn` and `engram amend` MUST write each body line as `Supersedes: [[<target>]] — <type>:
<claim>`, where `<target>` drops any trailing `.md` from the given note, while the entry's `note:`
field keeps the note exactly as given.

Rationale: the vault's link graph names each note by its basename without `.md`, as Obsidian does.
Supersession lines were once written with the `.md` kept, so they never resolved to the note they
named (#717). The frontmatter `note:` is a file reference rather than a graph link, so it keeps the
filename. Replacing the whole list, rather than appending to it, makes repeating an amend harmless.

#### 792a4d1 UAT Scenarios

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

### 792a4e Chunk sources

Requirement: 792a4e `engram amend --chunk-source <source#anchor>`, repeatable, MUST add each chunk
ID to the note's `sources:` once, after the IDs already there, and MUST keep every ID already
there.

#### 792a4e1 UAT Scenarios

```gherkin
Scenario: 792a4e1a Chunk sources are merged without duplicates
  Given a note whose sources: lists a#1, and a chunk index holding a#1 and b#2
  When the user runs engram amend --target <note> --chunk-source a#1 --chunk-source b#2
  Then the note's sources: lists a#1 then b#2
```

### 792a4f Amend argument validation

Requirement: 792a4f `engram amend` MUST refuse, writing nothing, a `--supersedes` that is not
`<note>|<type>|<claim>` with a type of `updates`, `narrows` or `refutes`, a `--trigger` shorter than
three characters after trimming, and a `--chunk-source` ID that the chunk index does not hold.

Rationale: `engram learn` records chunk sources without checking them, because the chunks a note is
written from may not be indexed yet (792a3h). An amend runs later, so it can check them.

#### 792a4f1 UAT Scenarios

```gherkin
Scenario Outline: 792a4f1a An invalid amend argument is refused before any write
  Given a vault holding fact note 1
  When the user runs engram amend --target 1 with <argument>
  Then it fails with an error saying <problem>, and the note is unchanged

  Examples:
    | argument                     | problem                                        |
    | --supersedes "a\|bogus\|c"   | supersedes type must be updates, narrows or refutes |
    | --supersedes "a\|updates"    | supersedes format must be <basename>\|<type>\|<claim> |
    | --trigger ab                 | trigger must be at least 3 characters          |
    | --chunk-source nope#x        | unresolved chunk-source id "nope#x"            |
```

### 792a4g Re-embedding after an amend

Requirement: 792a4g When `engram amend` changes a content field of the note, it MUST rebuild the
note's `.vec.json` sidecar, and when it changes only supersessions, chunk sources or identity, it
MUST NOT rebuild the sidecar. When rebuilding fails, the note MUST still be written, and the
command MUST print a warning naming the note and succeed.

Rationale: the sidecar embeds the note's situation and body, and its content hash leaves out the
`Supersedes:` lines and all frontmatter but the situation, so a provenance-only change leaves the
sidecar current. A failed rebuild is recoverable later, because the note itself was written in one
atomic step.

#### 792a4g1 UAT Scenarios

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

### 792a4h Identity re-stamp

Requirement: 792a4h An `engram amend` given any content flag, `--supersedes` or `--chunk-source`
MUST overwrite the note's `repo:`, `user:` and `vault:` with the values detected in its own
environment, as `engram learn` detects them (792a3o, 792a3p, 792a2b), whatever the note held
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

#### 792a4h1 UAT Scenarios

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

### 792a4i Vault name on every amend

Requirement: 792a4i Every `engram amend` that writes a note, one that keeps its declared identity
included, MUST write the resolved vault name (792a2b) into a note whose `vault:` is absent or
empty, and MUST keep a `vault:` the note declares.

Rationale: a `vault: ""` would look like a real value to every later reader, and a note written
before identity existed would otherwise keep no vault at all.

#### 792a4i1 UAT Scenarios

```gherkin
Scenario: 792a4i1a A use recorded on a note without vault fills it in
  Given a note with no user: and no vault:, and no vault name configured
  When the user runs engram amend --target <note> --activate
  Then the note gains vault: personal and gains no user: field
```

### 792a4j Untouched frontmatter is kept

Requirement: 792a4j `engram amend` and `engram resituate` MUST keep every frontmatter key they do
not change, with its value, including keys engram does not define and YAML anchors on keys they do
not edit. When an amend replaces `supersedes:`, an entry that names the same note as an existing
entry, compared without a trailing `.md`, MUST keep the existing entry's keys that engram does not
define.

Rationale: hand-edited, imported and Obsidian-edited notes carry keys that engram's note model does
not know. Revisions used to re-render the frontmatter from that model and silently dropped them
(#780).

#### 792a4j1 UAT Scenarios

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

### 792a4k Anchored keys are refused

Requirement: 792a4k `engram amend` and `engram resituate` MUST refuse, leaving the note unchanged,
an edit to a frontmatter key whose key or value, or anything beneath the value, carries a YAML
anchor, with an error naming the key.

Rationale: replacing an anchored value would leave every alias to it dangling, so the frontmatter
would no longer read back.

#### 792a4k1 UAT Scenarios

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

### 792a4l Resituation

Requirement: 792a4l `engram resituate --note <ref> --situation <text>` MUST replace a fact or
feedback note's `situation:` with the text, rebuild the first line of its body around it, keep the
rest of the body, rebuild the note's sidecar, and print the note's path on stdout, and MUST refuse,
writing nothing, a note of any other kind. When rebuilding the sidecar fails, the command MUST fail
with an error naming the note, with the note already rewritten.

Rationale: a fact or feedback note states its situation twice, in `situation:` and in its body
formula, and `engram learn` only creates notes. Before resituate, the two copies could be kept in
step only by hand. Unlike an amend's warning, a failed sidecar rebuild fails the command, because a
resituate is an explicit request to change what the note is found by.

#### 792a4l1 UAT Scenarios

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

### 792a4m CRLF notes

Requirement: 792a4m `engram amend` and `engram resituate` MUST read a note whose lines end in
`\r\n` as its LF form, never refusing it for its line endings, MUST write it as LF in the single
write they make, byte-identical to what the same command writes for the note's LF form, and MUST
rebuild its sidecar.

Rationale: notes edited on Windows or by other tools can carry CRLF line endings, and amend used to
refuse them as having no frontmatter (#789). A note is converted only inside a write the command
makes anyway, so no command writes a note just to convert it. Conversion changes the note's text,
so its sidecar is rebuilt even when the amend itself would not re-embed.

#### 792a4m1 UAT Scenarios

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

### 792a4n Activation

Requirement: 792a4n `engram activate --note <ref>`, repeatable, where each ref is a note's path,
absolute or relative to the vault, with or without `.md`, MUST record today's date as the last-used
date in each named note's sidecar, changing nothing else in the sidecar and nothing in the note, and
MUST count a note that has no sidecar as activated. With no parent configured, a ref that names no
note MUST print a warning naming it, the other refs MUST still be activated, and the command MUST
fail saying how many refs failed. `engram amend --activate` MUST record the last-used date in the
same way, in the same command as its amend.

Rationale: activation is agent-driven. Query never marks a note used; the recall skill activates
only the notes it actually used. The last-used date is kept out of the sidecar's content hash, so
recording a use never makes the sidecar stale.

#### 792a4n1 UAT Scenarios

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

### 792a4o Concurrent revisions never lose an update

Requirement: 792a4o Two vault writers — `engram amend`, `resituate`, `activate` or `learn` —
running against one vault at the same time MUST NOT lose either one's change to a note, and MUST
NOT leave a note's sidecar holding the vectors of the note's earlier text.

Rationale: before every vault writer took the vault lock, two concurrent writers corrupted vault and
index state in production. Many sub-recalls can amend and activate the same vault in parallel.

#### 792a4o1 UAT Scenarios

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

## 792a5 Added: chunk-indexing

Canonical target: `docs/specs/product/chunk-indexing.md` (title `Chunk Indexing - Product Spec`).

`engram ingest` turns session transcripts and Markdown documents into a chunk index: the text of
each source is cut into chunks, each chunk is embedded, and the chunks are stored outside the vault
so that `engram query` can search them as evidence. `engram prune` maintains that index. No LLM is
involved. Searching chunks and fetching one chunk's text (`engram query`, `query-chunks`,
`show-chunk`) is the memory-query capability, and the backlog notice that `engram update` prints for
duplicate index files belongs to install-and-update. A chunk is named by its chunk ID,
`<source path>#<anchor>`.

### 792a5a Chunk index location

Requirement: 792a5a Every engram command that reads or writes the chunk index MUST use the
directory given by `--chunks-dir` when it is non-empty, else `ENGRAM_CHUNKS_DIR` when it is
non-empty, else `$XDG_DATA_HOME/engram/chunks` when `XDG_DATA_HOME` is non-empty, else
`~/.local/share/engram/chunks`.

Rationale: the index follows the same precedence as the vault, so a test or eval run can point both
at an isolated location (`ENGRAM_CHUNKS_DIR` with `ENGRAM_VAULT_PATH`) without touching the real
index.

#### 792a5a1 UAT Scenarios

```gherkin
Scenario: 792a5a1a The flag wins over the environment
  Given ENGRAM_CHUNKS_DIR is set to /env/chunks
  When the user runs engram ingest --chunks-dir /flag/chunks --markdown notes.md
  Then notes.md is indexed under /flag/chunks

Scenario: 792a5a1b The environment variable is used without a flag
  Given ENGRAM_CHUNKS_DIR is set to /env/chunks
  When the user runs engram ingest --markdown notes.md
  Then notes.md is indexed under /env/chunks

Scenario: 792a5a1c The XDG data directory is the default
  Given neither --chunks-dir nor ENGRAM_CHUNKS_DIR is set and XDG_DATA_HOME is /data
  When the user runs engram ingest --markdown notes.md
  Then notes.md is indexed under /data/engram/chunks

Scenario: 792a5a1d The home directory is the last fallback
  Given neither --chunks-dir, ENGRAM_CHUNKS_DIR nor XDG_DATA_HOME is set
  When the user runs engram ingest --markdown notes.md
  Then notes.md is indexed under ~/.local/share/engram/chunks
```

### 792a5b Explicitly named sources

Requirement: 792a5b `engram ingest` MUST index each file named by a repeatable `--transcript`
(a session transcript) or `--markdown` (a Markdown document), and each `.md` and `.jsonl` file under
each directory named by a repeatable `--pi-sessions`, skipping that directory's `jobs/` and
`projects/` subdirectories and its hidden subdirectories. It MUST ignore a `--pi-sessions` value that
is not a directory, and MUST fail with an error naming the file when a file named by `--transcript`
or `--markdown` cannot be read.

#### 792a5b1 UAT Scenarios

```gherkin
Scenario: 792a5b1a A named transcript and document are indexed
  Given a session transcript s.jsonl and a Markdown document d.md
  When the user runs engram ingest --transcript s.jsonl --markdown d.md
  Then the chunk index holds chunks of both files, and stdout has one "ingest <path>: N chunks" line for each

Scenario: 792a5b1b A Pi sessions directory is indexed without its scratch
  Given a directory p holding a.jsonl, jobs/b.jsonl, projects/c.jsonl and .hidden/d.jsonl
  When the user runs engram ingest --pi-sessions p
  Then a.jsonl is indexed and none of the other three files is

Scenario: 792a5b1c A Pi sessions value that is not a directory is ignored
  Given a regular file f
  When the user runs engram ingest --pi-sessions f
  Then the command succeeds and indexes nothing

Scenario Outline: 792a5b1d An unreadable named file fails the command
  Given no file exists at <file>
  When the user runs engram ingest <flag> <file>
  Then the command fails with an error naming <file>

  Examples:
    | flag         | file          |
    | --markdown   | missing.md    |
    | --transcript | missing.jsonl |
```

### 792a5c Swept directories

Requirement: 792a5c `engram ingest --sweep <dir>`, repeatable, MUST index every `.md` and `.jsonl`
file under the directory, skipping hidden subdirectories, subdirectories named `node_modules`,
`vendor`, `.git`, `.hg`, `.jj`, `dist`, `build`, `target`, `.venv`, `venv`, `__pycache__`, `.next`,
`.cache` or `.idea`, and any file inside the chunk index directory. A swept file that cannot be read
MUST be skipped with a `skip <path>: ...` line on stdout while the rest of the run continues.

Rationale: a named file is a deliberate request, so failing to read it is an error. A swept file can
vanish between the walk and the read, for example a session that a harness deletes, and losing one
file must not stop the sweep. Skipping the index directory keeps a sweep root that contains the
index from indexing the index itself.

#### 792a5c1 UAT Scenarios

```gherkin
Scenario: 792a5c1a A swept directory's documents and transcripts are indexed
  Given a directory r holding a.md, sub/b.jsonl, node_modules/c.md, .git/d.md and notes.txt
  When the user runs engram ingest --sweep r
  Then a.md and sub/b.jsonl are indexed, and node_modules/c.md, .git/d.md and notes.txt are not

Scenario: 792a5c1b A swept file that vanishes is skipped
  Given a swept directory whose file gone.md cannot be read when the sweep reaches it
  When the user runs engram ingest --sweep <dir>
  Then stdout has a "skip <dir>/gone.md:" line, the directory's other files are indexed, and the command succeeds

Scenario: 792a5c1c The chunk index directory is never indexed
  Given a chunk index directory inside the swept directory r
  When the user runs engram ingest --sweep r --chunks-dir r/chunks
  Then no file under r/chunks is indexed
```

### 792a5d Automatic sweep roots

Requirement: 792a5d `engram ingest --auto` MUST sweep, as in 792a5c, the repository root (the
nearest ancestor of the working directory, itself included, that contains `.git`, `.hg` or `.jj`, or
the working directory when none does), every `.claude` and every `.pi` directory on the working
directory's ancestor chain up to the filesystem root, and the session-log directory
(`ENGRAM_TRANSCRIPT_DIR` when set, else `~/.claude/projects`) when it exists. Inside a `.claude`
directory it MUST also skip the subdirectories `projects`, `jobs`, `plugins`, `cache`, `todos`,
`shell-snapshots`, `file-history`, `history`, `ide`, `statsig`, `session-env`, `debug` and
`worktrees`, and inside a `.pi` directory the subdirectories `jobs` and `projects`.

Rationale: the automatic sweep covers what an agent's memory can learn from: the project's own
documents, project and user harness configuration, and every recorded conversation of every
project. Inside `.claude`, harness state and third-party plugin content are not user memory, and
`projects/` holds every project's transcripts, which arrive through the session-log root instead.
`jobs/` is agent-harness scratch that can hold whole snapshot copies of the vault. The `.pi` sweep
always excluded it; the `.claude` sweep did not, and on one real index 1,397 of the 1,960 duplicate
index files that `engram prune --duplicates` removed (71%) came from `~/.claude/jobs`.

#### 792a5d1 UAT Scenarios

```gherkin
Scenario: 792a5d1a The repository, its ancestors and the session logs are swept
  Given a working directory /r/sub inside a git repository /r, a /r/.claude directory, a /r/sub/.pi directory, and session logs under ~/.claude/projects
  When the user runs engram ingest --auto from /r/sub
  Then Markdown under /r, the files of /r/.claude and /r/sub/.pi, and the session transcripts are indexed

Scenario: 792a5d1b Without a repository marker the working directory is the root
  Given a working directory /w with no .git, .hg or .jj in it or any ancestor
  When the user runs engram ingest --auto from /w
  Then Markdown under /w is indexed

Scenario: 792a5d1c Harness scratch and state are not swept
  Given an ancestor .claude directory holding notes.md, jobs/snap.md, projects/p/s.jsonl and plugins/x.md
  When the user runs engram ingest --auto
  Then notes.md is indexed and jobs/snap.md, projects/p/s.jsonl and plugins/x.md are not

Scenario: 792a5d1d ENGRAM_TRANSCRIPT_DIR names the session logs
  Given ENGRAM_TRANSCRIPT_DIR is set to /logs, which holds a.jsonl
  When the user runs engram ingest --auto
  Then /logs/a.jsonl is indexed, and no transcript under ~/.claude/projects is
```

### 792a5e Repository sweep configuration

Requirement: 792a5e `engram ingest --auto` MUST read `.engram/sweep.json` at the repository root
when it exists, letting each key it sets replace that setting's default: `repo_markdown`,
`ancestor_claude_dirs`, `ancestor_pi_dirs` and `session_logs` (each root on or off),
`extra_roots` (more directories to sweep), `exclude_dirs`, `claude_exclude_dirs`,
`include_hidden_dirs`, `non_persistent_prefixes` and `non_persistent_path_prefixes`. A setting the
file omits MUST keep its default, and a file that is not valid JSON MUST fail the command with an
error naming the file.

Rationale: the sweep is declarative and inspectable. Its defaults are compiled in, and a repository
tunes them by editing data rather than code.

#### 792a5e1 UAT Scenarios

```gherkin
Scenario: 792a5e1a A repository adds a root and turns one off
  Given a repository whose .engram/sweep.json is {"session_logs": false, "extra_roots": ["/x"]}
  When the user runs engram ingest --auto inside it
  Then /x and the repository are swept, and no session log is indexed

Scenario: 792a5e1b A malformed sweep file is refused
  Given a repository whose .engram/sweep.json is not valid JSON
  When the user runs engram ingest --auto inside it
  Then the command fails with an error naming .engram/sweep.json, and nothing is indexed
```

### 792a5f Throwaway workspaces are not swept

Requirement: 792a5f `engram ingest --auto` MUST NOT index a session-log project directory whose name
starts with `-private-tmp-`, `-tmp-`, `-var-folders-` or `-private-var-folders-`, and MUST NOT sweep a
repository, `.claude`, `.pi` or session-log root whose own path is, or is under, `/tmp`,
`/private/tmp`, `/var/folders` or `/private/var/folders`. Files named by `--transcript`, `--markdown`
or `--pi-sessions`, directories named by `--sweep`, and `extra_roots` MUST be indexed wherever they
are.

Rationale: eval and test runs work in throwaway directories, and indexing them would bloat the real
index permanently. The project-directory names are the slugified working directories under which
Claude Code records sessions, so a session run in `/private/tmp/...` lands in a directory named
`-private-tmp-...`. A root whose own path is throwaway, such as a repository root that resolves to
`/tmp/eval-pool/warm0` because that directory has no repository marker, is dropped whole, because a
name check only reaches a root's subdirectories. Deliberate test ingestion uses an explicit source
or an isolated index (`ENGRAM_CHUNKS_DIR`, `ENGRAM_VAULT_PATH`).

#### 792a5f1 UAT Scenarios

```gherkin
Scenario: 792a5f1a Sessions recorded in a throwaway directory are skipped
  Given session logs under ~/.claude/projects/-private-tmp-eval-1/ and ~/.claude/projects/-Users-u-repo/
  When the user runs engram ingest --auto
  Then the -Users-u-repo sessions are indexed and the -private-tmp-eval-1 sessions are not

Scenario: 792a5f1b A throwaway working directory is not swept
  Given a working directory /tmp/test-pool/run123 with no repository marker
  When the user runs engram ingest --auto from it
  Then no file under /tmp/test-pool/run123 is indexed

Scenario: 792a5f1c An ancestor directory under the macOS temp root is dropped
  Given a working directory under /private/var/folders/xy/ with a .claude directory on its ancestor chain there
  When the user runs engram ingest --auto from it
  Then no file of that .claude directory is indexed

Scenario: 792a5f1d Explicit sources are indexed anywhere
  Given a transcript /tmp/explicit-session.jsonl and a directory /tmp/sweep-me holding a.md
  When the user runs engram ingest --transcript /tmp/explicit-session.jsonl --sweep /tmp/sweep-me
  Then both files are indexed
```

### 792a5g Transcript content

Requirement: 792a5g `engram ingest` MUST index a session transcript as its user and assistant text,
each turn's text prefixed `USER: ` or `ASSISTANT: `, with each tool call whose result is recorded as
plain text shown as one `[tool] <name>(<arguments>) → exit <0|1> | <first output line>` line, and
MUST leave out every other
transcript entry, system reminders, injected skill bodies, local-command output, task notifications,
and slash-command invocations other than their arguments.

Rationale: a transcript row carries far more harness bookkeeping than conversation. Chunks are
evidence an agent reads back, so they keep what was said and done, compressed to a line per tool
call, and drop what the harness injected.

#### 792a5g1 UAT Scenarios

```gherkin
Scenario: 792a5g1a A conversation is indexed as text with tool summaries
  Given a transcript where the user asks to run the tests, the assistant runs a Bash tool that prints "ok", and a system reminder is injected
  When the user runs engram ingest --transcript <file> and reads the chunk with engram show-chunk
  Then the chunk text has a USER: line, an ASSISTANT: line and a "[tool] Bash(command=...) → exit 0 | ok" line, and no system-reminder text
```

### 792a5h Chunk boundaries and IDs

Requirement: 792a5h `engram ingest` MUST cut a Markdown document into one chunk per heading section,
ignoring headings inside fenced code, with the heading's text as the anchor and `preamble` for text
before the first heading. It MUST cut a transcript into chunks of consecutive turns of about 500
characters, anchored `turn-<N>` by the first user turn they hold, or `preamble` for text before the
first user turn. It MUST split any section or turn group longer than 1500 characters on paragraph,
then line, boundaries into pieces that share its anchor, and MUST NOT index a chunk shorter than 40
characters.

#### 792a5h1 UAT Scenarios

```gherkin
Scenario: 792a5h1a Markdown is chunked by heading
  Given d.md holding text before any heading, a "## Setup" section and a "## Usage" section, each over 40 characters
  When the user runs engram ingest --markdown d.md
  Then the index holds the chunks d.md#preamble, d.md#Setup and d.md#Usage

Scenario: 792a5h1b A transcript is chunked by turns
  Given s.jsonl holding many short user and assistant turns
  When the user runs engram ingest --transcript s.jsonl
  Then each chunk holds consecutive turns, and their IDs are s.jsonl#turn-1, then the turn numbers at which later chunks start

Scenario: 792a5h1c Short sections are not indexed
  Given d.md whose only section is "## A" followed by "tiny"
  When the user runs engram ingest --markdown d.md
  Then no chunk of d.md is indexed

Scenario: 792a5h1d A long section is split under one anchor
  Given d.md whose "## Big" section is three paragraphs of about 940 characters each
  When the user runs engram ingest --markdown d.md
  Then the index holds several chunks d.md#Big, each at most 1500 characters long
```

### 792a5i Incremental re-indexing

Requirement: 792a5i `engram ingest` MUST NOT read or re-embed a source whose modification time and
size match its last ingest, MUST re-chunk a source whose content changed and embed only the chunks
whose text is not already in that source's index, and MUST print
`ingest <path>: <N> chunks (<R> reused, <E> embedded)` for each source it re-chunks, where `<N>` is
the number of chunks the source's index now holds, and nothing for a source it does not.

Rationale: the automatic sweep runs at the start of every learn, over every session and document
the user has, so an unchanged source must cost only a stat. Reusing the vectors of unchanged chunk
text means editing one section of a long document re-embeds only that section.

#### 792a5i1 UAT Scenarios

```gherkin
Scenario: 792a5i1a An unchanged source costs nothing
  Given a source that was ingested and has not changed since
  When the user runs engram ingest again with the same source
  Then the source is not read, nothing is embedded, and stdout has no line for it

Scenario: 792a5i1b A changed source embeds only its new text
  Given a Markdown document with sections A and B that was ingested, after which section B was edited
  When the user runs engram ingest again with the document
  Then stdout reads "ingest <path>: 3 chunks (1 reused, 1 embedded)", the third chunk being B's earlier text (792a5j)
```

### 792a5j A source's chunks are never dropped

Requirement: 792a5j When a source's content changes, `engram ingest` MUST keep every chunk the source
yielded on earlier ingests, alongside the chunks of its new content.

Rationale: an embedded chunk is the memory; its source is only where it came from. Keeping earlier
chunks means rewriting or truncating a document never erases what the index learned from it. The
cost is that a document edited many times keeps every earlier version of its edited sections.

#### 792a5j1 UAT Scenarios

```gherkin
Scenario: 792a5j1a Edited-away text stays searchable
  Given a document that was ingested with the paragraph "use flock for the manifest"
  When the paragraph is replaced by other text and the user runs engram ingest again
  Then the index holds both the old paragraph's chunk and the new text's chunk
```

### 792a5k Sources without chunks leave no index file

Requirement: 792a5k `engram ingest` MUST NOT write an index file for a source that yields no
chunks.

Rationale: every query opens every index file, so an empty file costs a read on every query and adds
nothing to any result.

#### 792a5k1 UAT Scenarios

```gherkin
Scenario: 792a5k1a An empty transcript writes no index file
  Given an empty transcript e.jsonl
  When the user runs engram ingest --transcript e.jsonl
  Then stdout reads "ingest e.jsonl: 0 chunks (0 reused, 0 embedded)", and no index file exists for e.jsonl
```

### 792a5l Identical content is indexed once

Requirement: 792a5l When sources of the same kind (transcript or Markdown) have byte-identical
content, `engram ingest` MUST index only one of them, chosen by precedence: a source named by
`--transcript`, `--markdown` or `--pi-sessions`; then one under the repository root; then one under
an ancestor `.claude` or `.pi` directory, nearest the working directory first; then any other; and
among equals the path with the fewest separators, then the lexicographically first path. It MUST
index a transcript and a Markdown document separately even when their bytes are identical.

Rationale: the same content is often reachable from several swept paths, such as a project file
copied into an ancestor `.claude` directory, a session copied into a worktree, or a vault note
snapshotted into a scratch directory. On a real index of about 62,000 sources, 79% were exact copies
of another source, and duplicated content made up 9.6% of a query's returned items on average and
28.6% at worst. A list of excluded paths could not reach most of it, because the copies sit in
legitimately swept roots. The precedence is a fixed function of the copies found, so the same
content always resolves to the same copy whatever order the sweep finds them in. A transcript is
chunked differently from a Markdown document with the same bytes, so treating them as copies would
lose one of the two chunkings.

#### 792a5l1 UAT Scenarios

```gherkin
Scenario: 792a5l1a A repository file outranks its copy in an ancestor .claude directory
  Given byte-identical files /r/doc.md in the repository and /r/.claude/doc.md
  When the user runs engram ingest --auto from /r
  Then /r/doc.md is indexed and /r/.claude/doc.md is not

Scenario: 792a5l1b A named file outranks the repository copy
  Given byte-identical files /x/doc.md and /r/doc.md in the repository
  When the user runs engram ingest --auto --markdown /x/doc.md from /r
  Then /x/doc.md is indexed and /r/doc.md is not

Scenario: 792a5l1c The nearer ancestor directory wins
  Given byte-identical files in /r/a/.pi and in /r/.claude, and the working directory /r/a
  When the user runs engram ingest --auto from /r/a
  Then the /r/a/.pi copy is indexed and the /r/.claude copy is not

Scenario: 792a5l1d A transcript and a document with the same bytes are both indexed
  Given a.jsonl and a.md with byte-identical content
  When the user runs engram ingest --transcript a.jsonl --markdown a.md
  Then both files have chunks in the index
```

### 792a5m A later, preferred copy replaces an indexed one

Requirement: 792a5m When an already-indexed source becomes a copy of a newly found source that takes
precedence over it (792a5l), `engram ingest` MUST index the preferred source and remove the other
source's chunks, but only when every chunk the other source holds is also among the preferred
source's chunks. Otherwise it MUST keep the other source's chunks and print
``ingest: refusing to evict <path>: canonical <path> does not verifiably cover its records (needs review — run `engram prune --duplicates` once resolved)``.
When the preferred copy stops being found, the remaining copy MUST be indexed again.

Rationale: byte-identical content today does not mean identical chunks, because a source keeps every
chunk of its earlier versions (792a5j). Two files equal now can hold different histories, and
removing one on the strength of the other would destroy chunks that exist nowhere else. On a real
index, 124 of 6,612 index files held chunks from more than one ingest day, which is exactly that
shape.

#### 792a5m1 UAT Scenarios

```gherkin
Scenario: 792a5m1a A copy indexed first gives way to the preferred one
  Given /r/.claude/doc.md was indexed while /r/doc.md did not exist
  When /r/doc.md is created with the same bytes and the user runs engram ingest --auto from /r
  Then /r/doc.md is indexed and /r/.claude/doc.md no longer has chunks in the index

Scenario: 792a5m1b A copy with a history of its own is kept
  Given /r/.claude/doc.md was indexed with an earlier version, then edited to equal a new /r/doc.md
  When the user runs engram ingest --auto from /r
  Then /r/.claude/doc.md's chunks are kept, and stdout has a "refusing to evict /r/.claude/doc.md" line

Scenario: 792a5m1c The remaining copy is indexed when the preferred one goes away
  Given /r/doc.md is indexed and /r/.claude/doc.md is recorded as its copy
  When /r/doc.md is deleted and the user runs engram ingest --auto from /r
  Then /r/.claude/doc.md is indexed
```

### 792a5n Stray vault-note copies are not indexed

Requirement: 792a5n `engram ingest` MUST NOT index a swept `.md` file that has a sibling `.vec.json`
file and lies outside the vault (792a2a), and MUST index such a file when it lies inside the vault or
is named by `--markdown`.

Rationale: a vault note always has a `.vec.json` sidecar beside it, so such a file outside the vault
is a copy of a vault note, for example in a snapshot or a backup. A copy of a note amended since it
was taken no longer has the same bytes as the live note, so identical-content deduplication cannot
catch it; on one machine, 50 of 256 stray vault-note copies had drifted from their original.

#### 792a5n1 UAT Scenarios

```gherkin
Scenario: 792a5n1a A vault-note copy outside the vault is skipped
  Given a swept directory holding 1.2026-10-09.x.md with 1.2026-10-09.x.vec.json beside it, and plain.md with no sidecar, outside the vault
  When the user runs engram ingest --sweep <dir>
  Then plain.md is indexed and 1.2026-10-09.x.md is not
```

### 792a5o Concurrent index writers never lose an update

Requirement: 792a5o `engram ingest` and `engram prune` runs against one chunk index at the same time
MUST NOT lose either one's change to the index's record of sources.

Rationale: concurrent ingest and prune runs once lost manifest entries in production (#660): each
read the record of sources, and the later write overwrote the earlier one's change.

#### 792a5o1 UAT Scenarios

```gherkin
Scenario: 792a5o1a An ingest and a prune at the same time both take effect
  Given a chunk index that records a source whose file is gone
  When engram ingest --transcript new.jsonl and engram prune start at the same time
  Then afterwards the index records new.jsonl and no longer records the gone source
```

### 792a5p Prune detaches vanished sources

Requirement: 792a5p `engram prune` MUST stop recording each indexed source whose file no longer
exists, MUST keep that source's chunks in the index and searchable, and MUST print
`prune: detached <N> source(s) — embedded chunks preserved (still searchable)`, or
`prune: no dead sources` when every recorded source exists, or `prune: no manifest, nothing to prune`
when the index records no sources. With `--dry-run` it MUST print the same detach line prefixed
`[dry-run] ` and change nothing.

Rationale: the embedded chunk is the asset and the source file only its provenance. A user can delete
ingested files, for example a directory of restored transcripts, to reclaim disk without losing what
was learned from them.

#### 792a5p1 UAT Scenarios

```gherkin
Scenario: 792a5p1a A deleted source is detached and stays searchable
  Given a transcript that was ingested and then deleted
  When the user runs engram prune and then engram query with a phrase matching it
  Then prune prints "prune: detached 1 source(s) — embedded chunks preserved (still searchable)", and the query returns its chunks with no re-indexing

Scenario: 792a5p1b A source that still exists is kept
  Given an ingested source whose file still exists
  When the user runs engram prune
  Then it prints "prune: no dead sources", and the source is still recorded

Scenario: 792a5p1c An empty index has nothing to prune
  Given a chunk index directory with nothing ingested
  When the user runs engram prune
  Then it prints "prune: no manifest, nothing to prune" and succeeds

Scenario: 792a5p1d A dry run changes nothing
  Given an ingested source that was then deleted
  When the user runs engram prune --dry-run
  Then it prints "[dry-run] prune: detached 1 source(s) — embedded chunks preserved (still searchable)", and the source is still recorded
```

### 792a5q Prune removes empty index files

Requirement: 792a5q `engram prune --empty` MUST remove every empty index file in the chunk index and
print `prune: removed <N> empty chunk-index file(s) of <M> scanned`. With `--dry-run` it MUST print
the same line prefixed `[dry-run] ` and remove nothing.

Rationale: indexes built before 792a5k left an empty file for each source with no chunks. Removing
them changes no search result, since they hold no chunks.

#### 792a5q1 UAT Scenarios

```gherkin
Scenario: 792a5q1a Empty index files are removed
  Given a chunk index with three index files, one of them empty
  When the user runs engram prune --empty
  Then the empty file is gone, the other two remain, and it prints "prune: removed 1 empty chunk-index file(s) of 3 scanned"

Scenario: 792a5q1b A dry run removes nothing
  Given a chunk index with one empty index file
  When the user runs engram prune --empty --dry-run
  Then the file remains, and it prints "[dry-run] prune: removed 1 empty chunk-index file(s) of 1 scanned"
```

### 792a5r Prune removes duplicate index files

Requirement: 792a5r `engram prune --duplicates` MUST group the recorded sources that are not already
recorded as copies by content and kind (792a5l), keep one source of each group, choosing the path
with the fewest separators and then the lexicographically first path, and remove every other
source's chunks and record, but only when the kept source has chunks in the index that include all
of the removed source's chunks. It MUST then print
`prune: removed <N> duplicate(s), retained <M> canonical(s)`, a
`prune: refused <K> removal(s): canonical has no index file (zero-chunk sources; nothing to lose)`
line when no source of a refused group has chunks, and a `refusing to remove <path>` line for each
refusal whose group does have chunks elsewhere. A removal that fails MUST print
`[FAIL] <path>: <error>` on stderr without stopping the others, and the command MUST then fail saying
how many of how many removals failed. With `--dry-run` it MUST report the same counts, each line
prefixed `[dry-run] `, and change nothing.

Rationale: ingest-time deduplication (792a5l) prevents new copies but does nothing for copies
indexed before it existed, so this mode cleans up that backlog. It runs only when asked, because it
removes index files from the user's memory store. It knows only the paths in the index's record, not
how each was found, so it cannot apply ingest's precedence; a later `engram ingest --auto` may then
replace the copy it kept with a preferred one (792a5m), once per group. On one real index it removed
1,960 of 8,572 index files (−23%).

#### 792a5r1 UAT Scenarios

```gherkin
Scenario: 792a5r1a A duplicate group is collapsed
  Given an index where /a/doc.md and /a/b/doc.md are both indexed with identical content and chunks
  When the user runs engram prune --duplicates
  Then /a/doc.md is kept, /a/b/doc.md no longer has chunks or a record, and it prints "prune: removed 1 duplicate(s), retained 1 canonical(s)"

Scenario: 792a5r1b A duplicate with chunks of its own is refused
  Given two indexed copies of one content, where the copy that would be removed holds a chunk the kept copy lacks
  When the user runs engram prune --duplicates
  Then both copies keep their chunks, and stdout has a "refusing to remove" line naming the copy

Scenario: 792a5r1c A dry run removes nothing
  Given an index with one removable duplicate
  When the user runs engram prune --duplicates --dry-run
  Then it prints "[dry-run] prune: removed 1 duplicate(s), retained 1 canonical(s)", and both copies remain

Scenario: 792a5r1d A second run removes nothing more
  Given an index on which engram prune --duplicates has just run
  When the user runs engram prune --duplicates again
  Then it prints "prune: removed 0 duplicate(s), retained 0 canonical(s)", and nothing changes

Scenario: 792a5r1e A failed removal is reported and fails the run
  Given an index with two removable duplicates, one of whose index files cannot be removed
  When the user runs engram prune --duplicates
  Then stderr has a "[FAIL]" line naming that file, the other duplicate is removed, and the command fails saying 1 of 2 removals failed
```

## 792a6 Added: controlled-vocabulary

Canonical target: `docs/specs/product/controlled-vocabulary.md` (title `Controlled Vocabulary - Product Spec`).

The vault keeps a controlled vocabulary: a small set of terms, each defined by a definition note.
Every note that engram writes is tagged `vocab/<term>` with the terms closest to it, and
`engram vocab refit` re-derives the term set from the vault's own notes as it grows. `engram vocab`
also seeds, extends, reports on and repairs the vocabulary, and `engram update` flags vaults that
need a vocabulary migration. Which kinds of note are tagged (every kind but questions) is
note-writing's 792a3u. Using the vocabulary at query time, including the query payload's
`refit_pending` field, is the memory-query capability; acting on a pending refit is the learn
skill's. Tagging the notes that a pull-down or a served learn writes belongs to parent-sync and
vault-serving.

### 792a6a Definition notes

Requirement: 792a6a Every term definition that `engram vocab bootstrap`, `engram vocab propose` or
`engram vocab refit` writes MUST be a top-level fact note with the slug `vocab-<term>-definition`,
`tags: [vocab, vocab/<term>]`, `tier: L2`, situation `recalling what the <term> vocab term covers, or
assigning vocab terms`, subject `the <term> vocab term`, predicate `covers`, the term's description as
its object, and source `vocab lifecycle (<command> v<version>)`. When the term comes with exemplars,
its body MUST end with an `Exemplars:` list holding each one. Every definition note MUST receive a
sidecar, or a warning naming it when embedding fails, and a definition note MUST NOT itself be
tagged with any term by assignment or refit.

Rationale: the definition's body is the term's embedding text, so exemplars from member notes pull
the term's vector toward its members; a description alone embeds far from them (recall at 5 of
45.5%, against 64.6% for the member centroid). The bare `vocab` tag marks a definition, so every
counting and tagging path can leave definitions out and a term's own description never skews its
centroid. The `vocab/<term>` self-tag was added so that a term's definition appears with its members
in Obsidian's tag pane and graph; it has no other effect.

#### 792a6a1 UAT Scenarios

```gherkin
Scenario: 792a6a1a A definition note carries its self-tag
  Given a vault and a seed file naming the term cooking with the description "recipes and kitchens" and one exemplar
  When the user runs engram vocab bootstrap --seed with that file
  Then the vault holds a note <id>.<date>.vocab-cooking-definition.md with tags [vocab, vocab/cooking], object "recipes and kitchens", an Exemplars: list holding the exemplar, and a sidecar

Scenario: 792a6a1b A definition is never assigned a term
  Given a vault whose vocabulary holds the term cooking
  When the user runs engram vocab refit or engram vocab bootstrap
  Then the cooking definition note's tags are still exactly [vocab, vocab/cooking]
```

### 792a6b The family note

Requirement: 792a6b The vault's vocabulary MUST be recorded by one family note, slug
`vocab-definition`, tagged only `vocab`, whose frontmatter carries the vault-wide `vocab_version`
and whose text describes the tagging convention without naming any term. `engram vocab bootstrap`
and `engram update --regen-vocab` MUST create it with version `1.0` when it is absent and MUST leave
an existing one alone. `engram vocab propose` MUST raise the version's minor part by one and
`engram vocab refit` its major part by one, setting the minor part to `0`.

Rationale: the vocabulary is emergent: the terms are the definition notes that exist, found by their
tags. A note that listed the terms would be the stale `vocab.index.md` again, so the family note
holds only the convention and the version.

#### 792a6b1 UAT Scenarios

```gherkin
Scenario: 792a6b1a Bootstrap creates the family note
  Given a vault with no vocabulary
  When the user runs engram vocab bootstrap with a seed file naming two terms
  Then the vault holds one note <id>.<date>.vocab-definition.md tagged only vocab, carrying vocab_version "1.0", and its text names neither term

Scenario: 792a6b1b The version follows propose and refit
  Given a vault whose family note carries vocab_version "2.0"
  When the user runs engram vocab propose and then an engram vocab refit that applies
  Then engram vocab stats reports version 2.1 after the propose and 3.0 after the refit
```

### 792a6c Categorical tags share one list

Requirement: 792a6c A note's categorical classifications MUST all live in its `tags:` list:
vocabulary tagging, refit and `engram update --regen-vocab` MUST change only the `vocab/<term>`
entries, keeping every other tag and its order.

Rationale: route's dispatch records needed low-cardinality categories (work kind, tier, outcome) that
recall, `engram count` and Obsidian can all read, without a separate store. Frontmatter `tags:`, a
plain list written by `engram learn --tag <family>[/<value>]`, is the only categorical mechanism: no
attribute nodes, no categorical wikilinks, no tables. Three kinds of note ride on it: route's
evidence notes (one per dispatch, tagged `work-kind/<k>`, `tier/<t>` and `outcome/<o>`, recallable
like any fact), its aggregate notes (one per work kind, slug `route-evidence-<work-kind>`, untagged,
holding the tallies and a link to each evidence note), and family definition notes (a bare family
tag defines the family, a `family/value` tag is a member). Route reads the aggregates through plain
recall; `engram count --group-by tags --filter tags=...` recomputes the true tallies as an audit,
because tallies an LLM maintains will drift, and it is never on the read path. Many near-identical
evidence notes could outrank their aggregate on the read query; a scratch-vault gauge (20 evidence
notes and one aggregate) passed, so no remedy was built. If drowning is measured on a real vault,
the remedies on record are a "summarizes" ride-along that inserts the aggregate when its evidence
surfaces, or demoting evidence notes to the chunk ranking tier, chosen with the measured case in
hand, because a new edge type must first show retrieval value. The vocabulary moved onto the same
convention: definitions became bare-`vocab`-tagged fact notes and members `vocab/<term>` tags.

#### 792a6c1 UAT Scenarios

```gherkin
Scenario: 792a6c1a A categorical tag survives tagging
  Given a vault whose vocabulary holds the term build-tooling
  When the user runs engram learn fact --tag work-kind/review with text close to build-tooling
  Then the note's tags are [work-kind/review, vocab/build-tooling], in that order

Scenario: 792a6c1b A categorical tag survives a refit
  Given a note tagged [work-kind/review, vocab/old-term]
  When the user runs an engram vocab refit that retires old-term
  Then the note still carries work-kind/review
```

### 792a6d Write-time vocabulary tagging

Requirement: 792a6d After `engram learn`, `engram amend` or `engram resituate` writes a note, the
note's `vocab/<term>` tags MUST be set to the terms whose vector has a cosine similarity of at least
0.35 with the note's body vector, at most three, highest first with ties in term-name order,
replacing any earlier vocabulary tags. A note with no qualifying term MUST lose its vocabulary tags,
and its `tags:` field MUST be removed when nothing else is left in it. A term's vector MUST be the
centroid stored by the last bootstrap or refit when one exists for the vault's embedding model, and
its definition note's embedding otherwise.

Rationale: tagging is pure cosine, so it costs no LLM call on the write path. The 0.35 floor and the
cap of three were chosen by a sweep over 48 known retrieval misses: two passes at 0.35 with three
terms recovered 56.2% of them at a median candidate pool of 31, passing the gate of at least 54.2%
within a pool of 40. Tags replace rather than accumulate, so a revised note carries only the terms it
is close to now.

#### 792a6d1 UAT Scenarios

```gherkin
Scenario: 792a6d1a A learned note is tagged with its closest terms
  Given a vault whose vocabulary holds the terms build-tooling and cooking
  When the user runs engram learn fact about running a linter before committing Go code
  Then the note carries vocab/build-tooling and not vocab/cooking

Scenario: 792a6d1b A note close to no term gets no tag
  Given a vault whose vocabulary holds only the term cooking
  When the user runs engram learn fact about configuring a TLS certificate
  Then the note has no vocab/ tag and no tags: field

Scenario: 792a6d1c At most three terms are kept
  Given a vault with four terms, each close to one note's text
  When the user writes that note with engram learn
  Then the note carries the three closest terms, closest first

Scenario: 792a6d1d An amend re-tags the note
  Given a note tagged vocab/cooking whose object is then rewritten to describe a linter
  When the user runs engram amend --object with the new text
  Then the note carries vocab/build-tooling and no longer vocab/cooking

Scenario: 792a6d1e A resituated note is re-tagged
  Given a fact note and a vault vocabulary
  When the user runs engram resituate on it
  Then its vocab/ tags are set from its rewritten text
```

### 792a6e Tagging is skipped without a vocabulary or an embedding

Requirement: 792a6e A note write MUST succeed and write no vocabulary tag when the vault has no
definition note with a sidecar, or when the written note has no sidecar.

#### 792a6e1 UAT Scenarios

```gherkin
Scenario: 792a6e1a A vault without a vocabulary writes plain notes
  Given a vault with no definition notes
  When the user runs engram learn fact
  Then the note is written with no vocab/ tag

Scenario: 792a6e1b A note whose embedding failed is not tagged
  Given a vault with a vocabulary, where embedding the new note fails
  When the user runs engram learn fact
  Then the note is written, stderr carries a warning naming it, and the note has no vocab/ tag
```

### 792a6f CRLF notes

Requirement: 792a6f Vocabulary tagging, term removal and refit MUST read a note with CRLF line
endings as its LF form. A tagging that changes a CRLF note's tags MUST write the note with LF
endings, and one that changes nothing MUST NOT write the note.

Rationale: a note's text must not change meaning with its line endings, and no reader rewrites a note
only to convert it: the conversion happens inside a write that is needed anyway.

#### 792a6f1 UAT Scenarios

```gherkin
Scenario: 792a6f1a Vocab tagging reads a CRLF note
  Given a note whose lines end in \r\n
  When vocabulary tagging adds a new term to it
  Then the tag is written and the note is written as LF; a CRLF note whose tags do not change is not written
```

### 792a6g Bootstrapping a vocabulary

Requirement: 792a6g `engram vocab bootstrap --seed <file>` MUST read a YAML list of entries with
`term`, `description` and optional `exemplars`, create the family note when absent (792a6b), create a
definition note (792a6a) for every term that has none, and leave terms that already have one
untouched. It MUST then tag every note that is neither a definition nor a question in two passes, first
against the definitions' embeddings and then against each term's centroid (the mean of the notes the
first pass gave it), MUST record those centroids and the vault's current note count and date as the
refit baseline, MUST clear any pending refit, and MUST print
`vocab bootstrap: <N> terms, <M> notes assigned`, where `<N>` is the number of seed entries and `<M>`
the number of term assignments the second pass made. `--floor` MAY replace the 0.35 similarity floor
of 792a6d for this run.

Rationale: the second pass replaces each description embedding with the centroid of its members, so
terms are placed where their notes actually are; centroid assignment is what the 0.35, three-term
setting of 792a6d was measured with.

#### 792a6g1 UAT Scenarios

```gherkin
Scenario: 792a6g1a Bootstrap seeds terms and tags existing notes
  Given a vault holding notes about Go builds and about cooking, and a seed file naming build-tooling
  When the user runs engram vocab bootstrap --seed with that file
  Then it prints "vocab bootstrap: 1 terms, <M> notes assigned", the build notes carry vocab/build-tooling, and engram vocab stats reports verdict: OK

Scenario: 792a6g1b A second bootstrap creates nothing new
  Given a vault already bootstrapped with a seed file
  When the user runs engram vocab bootstrap with the same seed file again
  Then no new definition note or family note is created
```

### 792a6h Bootstrap failures

Requirement: 792a6h `engram vocab bootstrap` MUST be refused when `--seed` is absent, and MUST fail
with an error when the seed file cannot be read or is not a YAML list of entries, in each case
before writing anything.

#### 792a6h1 UAT Scenarios

```gherkin
Scenario Outline: 792a6h1a A bad seed is refused
  Given a vault
  When the user runs engram vocab bootstrap <arguments>
  Then it fails with an error saying <problem>, and nothing is written

  Examples:
    | arguments                   | problem                              |
    | without --seed              | --seed is a missing required flag    |
    | --seed missing.yaml         | the seed file cannot be read         |
    | --seed with "term: [unclosed" | the seed YAML cannot be parsed     |
```

### 792a6i Proposing a term

Requirement: 792a6i `engram vocab propose --term <term> --description <text>` MUST require both
flags, create the term's definition note (792a6a), raise the vocabulary's minor version (792a6b),
record the term as proposed (792a6q), and print `vocab propose: created <term> (version → <v>)`. In a
vault without a family note it MUST still create the definition and warn that the family note was
not found.

Rationale: the terms the clustering cannot see, a concept scattered across the vault's geometry,
still need a way in. Proposing is that one judgment-based path; whether a term is worth adding
(no existing term covers it, and it would tag no more than a fifth of the vault) is decided by the
agent before it runs the command, which only writes.

#### 792a6i1 UAT Scenarios

```gherkin
Scenario: 792a6i1a A proposed term is created
  Given a bootstrapped vault at vocab_version "1.0"
  When the user runs engram vocab propose --term gardening --description "growing plants"
  Then it prints "vocab propose: created gardening (version → 1.1)" and the vault holds a vocab-gardening-definition note

Scenario: 792a6i1b Propose in a vault without a family note
  Given a vault with no vocabulary
  When the user runs engram vocab propose --term test-term --description "a test"
  Then it prints "vocab propose: created test-term", and stderr warns that the family definition note (vocab-definition) was not found

Scenario: 792a6i1c Both flags are required
  Given a vault
  When the user runs engram vocab propose --term x without --description
  Then it fails saying --description is a missing required flag, and nothing is written
```

### 792a6j Deriving the vocabulary

Requirement: 792a6j `engram vocab refit` MUST cluster the embeddings of every note that is not a
definition, choosing the number of clusters K between 2 and 40 that gives the highest silhouette
score, and keeping the previous derivation's K when its score is within 0.02 of the best. When no
clustering scores at least 0.02, or there are too few notes to cluster, it MUST print
`vocab refit: no structure found; vocabulary unchanged` and write nothing.

Rationale: the earlier refit edited the existing term list from an LLM-written plan, and could only
add terms: an untagged-rate trigger fired whenever new notes missed the 0.35 floor, and minting was
the only way to satisfy it, so one vault drifted to about 150 terms for 400 notes. Tagging is already
pure cosine geometry, so the vocabulary is now derived from that geometry directly, and the number of
terms is the number the vault's clusters support. The 0.02 floor only rejects input with no
structure: whole-vault MiniLM embeddings score between 0.05 and 0.15 at every K (the production vault
of 597 notes peaked at 0.0987 at K=9), so recall's 0.10 floor would have made every refit a no-op.
Keeping the previous K within 0.02 stops term names churning on near ties.

#### 792a6j1 UAT Scenarios

```gherkin
Scenario: 792a6j1a The term count follows the vault's clusters
  Given a vault of notes in two clear topics
  When the user runs engram vocab refit --dry-run
  Then it prints "vocab refit (dry-run): K=2 silhouette=<s>"

Scenario: 792a6j1b A vault without structure keeps its vocabulary
  Given an empty vault, or one whose notes are too few to cluster
  When the user runs engram vocab refit
  Then it prints "vocab refit: no structure found; vocabulary unchanged" and no file changes
```

### 792a6k Term names stay stable

Requirement: 792a6k A derived cluster MUST keep the name and definition note of the existing term
whose vector has a cosine similarity of at least 0.80 with the cluster's centroid, pairing clusters
and terms best match first, each at most once. A cluster that matches no term is new and needs a
name; an existing term that no cluster matches is retired (792a6p) unless it was proposed (792a6q).

Rationale: names users and notes already rely on survive re-derivation when the concept is still
there. Greedy best-first matching is auditable and stable at a few tens of terms.

#### 792a6k1 UAT Scenarios

```gherkin
Scenario: 792a6k1a A cluster keeps its term's name
  Given a vault whose term build-tooling sits at the centre of one of its clusters
  When the user runs engram vocab refit --dry-run
  Then it prints "matched: build-tooling (cluster <n>, similarity <s>)" with s at least 0.800

Scenario: 792a6k1b Unmatched clusters and terms are reported
  Given a vault with an unmatched cluster and an unmatched derived term astronomy
  When the user runs engram vocab refit --dry-run
  Then it prints "new: cluster <n> (<m> members, needs naming)" and "retired: astronomy"
```

### 792a6l Dry run

Requirement: 792a6l `engram vocab refit --dry-run` MUST print `vocab refit (dry-run): K=<k>
silhouette=<s>`, one `matched:` line per kept term, one `new:` line per new cluster and one
`retired:` line per retired term, and MUST write nothing.

Rationale: the first derivation on a vault can retire most of its terms (K=9 against 29 existing
terms on the production vault), so reviewing the diff before applying it matters.

#### 792a6l1 UAT Scenarios

```gherkin
Scenario: 792a6l1a Dry run reports the derivation diff without writing
  Given a vault whose derivation matches one term, finds one new cluster and retires one term
  When the user runs engram vocab refit --dry-run
  Then the K and silhouette line, the matched, new and retired lines are printed, and no vault file or centroids file is modified
```

### 792a6m Naming requests

Requirement: 792a6m When a refit finds new clusters and no `--names` file is given, `engram vocab
refit` MUST print a JSON document with a `naming_requests` list, one entry per new cluster holding its
`cluster` index, `member_count` and up to three `exemplars` (the members nearest its centroid, each
with its `note` basename, a `title` from its slug and a `snippet` of at most 200 bytes of its body),
an `instruction` for answering, and a `fingerprint` of the notes it clustered, and MUST write nothing.

Rationale: the agent's LLM only names clusters; it never decides the term set. Nearest members make
better naming evidence than random ones.

#### 792a6m1 UAT Scenarios

```gherkin
Scenario: 792a6m1a New clusters are emitted as naming requests
  Given a vault whose derivation finds one cluster that matches no term
  When the user runs engram vocab refit
  Then it prints a naming_requests document for that cluster with its member count, three exemplars and a fingerprint, and no vault file changes
```

### 792a6n Naming answers

Requirement: 792a6n `engram vocab refit --names <file>` MUST apply the derivation only when the file
is JSON `{"names":[{"cluster":N,"term":"...","description":"..."}],"fingerprint":"..."}` whose
fingerprint equals the current vault's, whose entries each name a new cluster exactly once with a
non-empty term and description, and which names every new cluster. Otherwise it MUST fail, writing
nothing, with an error saying which check failed: the file cannot be read or parsed, the vault changed
since the requests were emitted (naming both fingerprints), a cluster is unknown, named twice or
unnamed, or an entry is empty.

Rationale: a cluster index means something only for the vault state that produced it; if notes
changed between the two runs, a name could land on the wrong cluster, so the stale answer is rejected
and the requests must be regenerated.

#### 792a6n1 UAT Scenarios

```gherkin
Scenario Outline: 792a6n1a A bad naming answer is refused
  Given a vault whose refit emitted naming requests for cluster 1
  When the user runs engram vocab refit --names with <answer>
  Then it fails with an error saying <problem>, and nothing is written

  Examples:
    | answer                                      | problem                                                     |
    | a fingerprint that does not match           | the vault changed since the naming requests were emitted    |
    | no entry for cluster 1                      | --names must name every new cluster: cluster 1 unnamed      |
    | an entry for cluster 7                      | --names references an unknown cluster                       |
    | two entries for cluster 1                   | --names names the same cluster twice                        |
    | an entry with an empty description          | --names entry needs non-empty term and description          |
    | a file that is not JSON                     | cannot parse --names JSON                                   |
```

### 792a6o Applying a refit

Requirement: 792a6o A refit that has no new clusters, or whose `--names` answer is accepted, MUST
raise the major version (792a6b), retire the unmatched derived terms (792a6p), create a definition
note for each named cluster with its exemplars' snippets, re-tag every note that is neither a
definition nor a question against the derived centroids, record each term's centroid, member count and
origin with the derivation's K, silhouette and date and a new refit baseline, clear any pending refit,
and print `vocab refit applied: version → <v> (K=<k>, matched=<m>, new=<n>, retired=<r>)`.

#### 792a6o1 UAT Scenarios

```gherkin
Scenario: 792a6o1a A named cluster becomes a term
  Given a vault at version 1.0 whose refit emitted a naming request for cluster 1
  When the user runs engram vocab refit --names with cooking for cluster 1 and the emitted fingerprint
  Then it prints "vocab refit applied: version → 2.0 (K=2, matched=1, new=1, retired=0)", a vocab-cooking-definition note exists, and the cluster's notes carry vocab/cooking

Scenario: 792a6o1b A refit with nothing to name applies at once
  Given a vault whose derivation matches every cluster to a term
  When the user runs engram vocab refit
  Then it applies without emitting naming requests, and engram vocab stats reports the new major version and verdict: OK
```

### 792a6p Retiring a term

Requirement: 792a6p Retiring a term MUST delete its definition note and the note's sidecar, remove
its `vocab/<term>` tag from every note, and remove every reference to the deleted note from the other
notes' bodies and `supersedes:` lists: a `supersedes:` entry naming it is dropped (and the
`supersedes:` key with its last entry), a `Supersedes: [[...]]` line linking it is dropped, and any
other link to it becomes its plain basename. Retirement MUST NOT keep the definition in any form or
record a supersession for it.

Rationale: a dormant term keeps attracting members at tagging time and stops the vocabulary from
converging, so retirement is active. A definition note is the term's identity, not a fact worth
preserving, so it has no supersession story; an earlier refit that demoted retired definitions and
recorded them as superseded on the family note left ten demoted notes and their dangling links in the
vault.

#### 792a6p1 UAT Scenarios

```gherkin
Scenario: 792a6p1a A retired term is deleted with its references
  Given a vault whose derived term astronomy matches no cluster, whose definition another note's body links to and names in a supersedes: entry
  When the user runs engram vocab refit
  Then the astronomy definition note and its sidecar are gone, no note carries vocab/astronomy, the supersedes: entry and its Supersedes: line are gone, and the body link reads as the plain basename

Scenario: 792a6p1b Retirement leaves no dangling body link
  Given a vault whose family note links to a term's definition that refit retires
  When the user runs engram vocab refit and then engram check
  Then engram check reports no dangling link to the deleted basename from a note body
```

### 792a6q Proposed terms survive derivation

Requirement: 792a6q A term created by `engram vocab propose` MUST NOT be retired by a refit that
finds no cluster for it: its definition note, its recorded origin `proposed` and its members' tags
MUST remain, and it MUST keep taking part in tagging.

Rationale: proposed terms exist precisely to name concepts the clustering cannot see, so the clustering
must not remove them.

#### 792a6q1 UAT Scenarios

```gherkin
Scenario: 792a6q1a An unmatched proposed term is kept
  Given a vault where the term gardening was proposed and no derived cluster matches it
  When the user runs engram vocab refit
  Then the gardening definition note remains and engram vocab stats still lists gardening
```

### 792a6r Refit takes no term plan

Requirement: 792a6r `engram vocab refit` MUST NOT accept an externally authored term plan: the
`--plan` and `--emit-request` flags MUST be refused as undefined.

Rationale: an authored term list cannot coexist with derivation; keeping the plan flow would keep the
additive path that 792a6j removed.

#### 792a6r1 UAT Scenarios

```gherkin
Scenario Outline: 792a6r1a The plan flow is gone
  Given a vault
  When the user runs engram vocab refit <flag> x
  Then it fails saying the flag provided is not defined, and nothing is written

  Examples:
    | flag           |
    | --plan         |
    | --emit-request |
```

### 792a6s Refit trigger

Requirement: 792a6s After every `engram learn`, `engram amend` and `engram resituate` write, the vault
MUST check whether a refit is due, without running one: when no refit baseline is recorded it MUST
record the current number of notes that are neither definitions nor questions, and today's date, as
the baseline; otherwise, when that number has grown by at least 40 since the baseline and at least 14
days have passed, it MUST mark a refit pending with the reason `growth: <N> notes, <D> days`. A
pending refit MUST stay pending until a bootstrap or an applied refit records a new baseline. The
untagged rate and term concentration MUST NOT mark a refit pending.

Rationale: the check is token-free and runs in the write path, so the vault notices its own drift and
the learn skill acts on the verdict. Growth is the only trigger because, under derivation, a high
untagged rate or a hub term no longer means terms should be minted; keeping them as triggers would
keep the old ratchet's cadence without its mechanism.

#### 792a6s1 UAT Scenarios

```gherkin
Scenario: 792a6s1a Growth marks a refit pending
  Given a vault whose refit baseline is 100 notes 20 days ago, now holding 150 notes
  When the user runs engram learn fact
  Then engram vocab stats reports "verdict: REFIT_PENDING (growth: <N> notes, <D> days)"

Scenario: 792a6s1b Growth without enough time does not trigger
  Given a vault whose refit baseline is 100 notes 5 days ago, now holding 150 notes
  When the user runs engram learn fact
  Then engram vocab stats reports verdict: OK

Scenario: 792a6s1c A first write records the baseline
  Given a vault with no refit baseline
  When the user runs engram learn fact
  Then a baseline is recorded, and engram vocab stats reports verdict: OK

Scenario: 792a6s1d A high untagged rate alone does not trigger
  Given a vault where more than 8% of notes are untagged and growth is below 40 notes
  When the user writes a note
  Then engram vocab stats reports the untagged rate with [high] and verdict: OK
```

### 792a6t Vocabulary statistics

Requirement: 792a6t `engram vocab stats` MUST print, without writing anything:
`vocab stats (version: <v>)` (`1.0` when there is no family note); `terms: <T>  member-notes: <M>
untagged: <U>`; `untagged-rate: <r>%`, followed by ` [high]` when the rate is above 8%; for each term
in name order `  <term>: <n> members`, followed by ` [hub]` when the term tags more than a quarter
of the member notes and ` [orphan]` when it has fewer than two members; `verdict: REFIT_PENDING
(<reason>)` when a refit is pending (792a6s) and `verdict: OK` otherwise; `qa pairs: <P>`, the number
of question notes whose answer note exists; and `qa round-2 gate: READY (<P>>=20)` when `<P>` is at
least 20, else `qa round-2 gate: accumulating (<P>/20)`. Member notes MUST exclude definition notes,
question notes and files without frontmatter; answer notes MUST count like any other note.

Rationale: the counts describe the notes that tagging applies to, so definitions (whose self-tag is
display-only) and questions (which carry no vocabulary, 792a3u) are left out. The untagged rate and
hubs are diagnostics for a person reading the report. The QA lines track the question-and-answer
kind's second round, which waits for 20 real captured pairs.

#### 792a6t1 UAT Scenarios

```gherkin
Scenario: 792a6t1a Stats on an empty vault
  Given an empty vault
  When the user runs engram vocab stats
  Then it prints "terms: 0  member-notes: 0  untagged: 0" and nothing on stderr

Scenario: 792a6t1b Hubs and orphans are flagged
  Given a vault with term a tagging 3 of 4 member notes and term b tagging none
  When the user runs engram vocab stats
  Then it prints "  a: 3 members [hub]" and "  b: 0 members [orphan]"

Scenario: 792a6t1c Questions and definitions are not member notes
  Given a vault with one definition note, one fact note and a question-and-answer pair
  When the user runs engram vocab stats
  Then member-notes counts the fact and the answer only, and it prints "qa pairs: 1" and "qa round-2 gate: accumulating (1/20)"

Scenario: 792a6t1d A pending refit is the verdict
  Given a vault with a refit pending for "growth: 45 notes, 20 days"
  When the user runs engram vocab stats
  Then it prints "verdict: REFIT_PENDING (growth: 45 notes, 20 days)"
```

### 792a6u Self-tag backfill

Requirement: 792a6u `engram vocab tag-definitions` MUST add the missing `vocab/<term>` self-tag to
every definition note, taking the term from its slug, and print `<basename>: added` for each, MUST
leave a definition that already has it unchanged and print `<basename>: already present`, MUST skip
the family note printing `<basename>: family note (skipped)`, and MUST skip a bare-`vocab` note whose
slug names no term, printing `<basename>: malformed definition slug (skipped)`. A rewritten note's
sidecar MUST stay current. The backfill MUST run only when this command is run, never as a side
effect of `engram update` or any other command.

Rationale: vaults whose definitions predate the self-tag need it added once. Tags are not part of a
note's content hash (the hash covers the situation and body), so a tags-only rewrite needs no
re-embedding. Mutating the vault stays an explicit command.

#### 792a6u1 UAT Scenarios

```gherkin
Scenario: 792a6u1a Backfill adds missing self-tags
  Given a vault whose definition notes carry only the bare vocab tag
  When the user runs engram vocab tag-definitions
  Then each per-term definition's tags become [vocab, vocab/<term>], engram embed status reports none of them stale, and one "added" line is printed per note

Scenario: 792a6u1b Backfill is idempotent
  Given a vault on which engram vocab tag-definitions has run
  When the user runs it a second time
  Then no note changes and every definition is reported "already present", the family note "family note (skipped)"
```

### 792a6v Update notices

Requirement: 792a6v `engram update` MUST print
`old-format vocab files found — run `engram update --regen-vocab` to migrate them (preview with
`engram update --regen-vocab --dry-run`)` when the vault holds a file named `vocab.<name>.md`, and
`vocab definition notes missing their vocab/<term> self-tag found — run `engram vocab
tag-definitions` to backfill them` when a definition note lacks its self-tag. It MUST NOT change the
vault to resolve either.

Rationale: update detects and notifies, and leaves rewriting the vault to an explicit command.

#### 792a6v1 UAT Scenarios

```gherkin
Scenario: 792a6v1a An old-format vault is flagged
  Given a vault holding vocab.index.md
  When the user runs engram update
  Then it prints the old-format vocab notice and the file is still there

Scenario: 792a6v1b A missing self-tag is flagged
  Given a vault with a definition note tagged only vocab
  When the user runs engram update
  Then it prints the missing self-tag notice and the note is unchanged

Scenario: 792a6v1c A current vault gets no vocab notice
  Given a vault with no vocab.<name>.md file and fully self-tagged definitions
  When the user runs engram update
  Then neither vocab notice is printed
```

### 792a6w Migrating an old-format vocabulary

Requirement: 792a6w `engram update --regen-vocab` MUST migrate a vault that holds old-format
vocabulary files: it MUST create the family note when absent and a definition note for each
`vocab.<term>.md` term note from its `term`, `description` and `Exemplars:` list, remove the `vocab:`
frontmatter line and `Vocab:` body line from every note that has them, delete every `vocab.<name>.md`
file and its sidecar, and re-tag every note as bootstrap does (792a6g). It MUST then print
`update --regen-vocab: removed <F> old-format file(s), cleaned <C> member note(s), seeded <S> term(s),
reassigned <A> note(s)`, or `update --regen-vocab: nothing to regenerate` when there were no old files
and no legacy lines. With `--dry-run` it MUST change nothing and print the same line prefixed
`[dry-run] ` with `would remove` in place of `removed`. After a migration, the old-format notice of
792a6v MUST NOT be printed.

Rationale: before the vocabulary moved onto tags, terms were `vocab.<term>.md` notes, a generated
`vocab.index.md` listed them, and members carried a `Vocab:` line and a `vocab:` key. The migration
re-scores members from the regenerated centroids rather than copying the old assignment.

#### 792a6w1 UAT Scenarios

```gherkin
Scenario: 792a6w1a An old-format vocabulary is migrated
  Given a vault holding vocab.cooking.md, vocab.index.md and a note with a Vocab: line
  When the user runs engram update --regen-vocab
  Then a vocab-cooking-definition note exists, both old files are gone, the note has no Vocab: line, and the report says removed 2 old-format file(s) and cleaned 1 member note(s)

Scenario: 792a6w1b A dry run changes nothing
  Given the same vault
  When the user runs engram update --regen-vocab --dry-run
  Then it prints "[dry-run] update --regen-vocab: would remove 2 old-format file(s), ..." and every file is unchanged

Scenario: 792a6w1c Nothing to migrate
  Given a vault with no old-format vocabulary
  When the user runs engram update --regen-vocab
  Then it prints "update --regen-vocab: nothing to regenerate"
```

## 792a7 Added: memory-query

Canonical target: `docs/specs/product/memory-query.md` (title `Memory Query - Product Spec`).

`engram query` searches the vault's notes and the chunk index for what is relevant to a set of
phrases and prints one YAML payload: the matched notes and chunks, the clusters they fall into, the
newest chunks, and a budget of counts. `engram show` prints one note in full, `engram show-chunk`
prints one chunk's text, and `engram query-chunks` searches the chunk index alone. All four only
read. The `recall` skill is the main caller and decides what to do with the payload. A query on a
vault that does not exist creates it first and returns an empty payload (792a2e). The payload's
`pending_offers` flag and the exclusion of pending offers from results belong to curate, results
merged in from a parent vault belong to parent-sync, and the served form of a query belongs to
vault-serving. Which kinds of note a query returns, and that it never returns a question note, is
note-writing's 792a3t.

### 792a7a Query input

Requirement: 792a7a `engram query` MUST accept one or more `--phrase` values and an optional
`--text`, and MUST fail with `query: empty query string` and a non-zero exit when it is given no
phrase and a `--text` that is empty or only whitespace.

#### 792a7a1 UAT Scenarios

```gherkin
Scenario: 792a7a1a A query with nothing to search for fails
  Given any vault
  When the user runs engram query with no --phrase and no --text, or with --text "   "
  Then it fails with "query: empty query string" and exits non-zero

Scenario: 792a7a1b Several phrases are searched together
  Given a vault holding a note about Go tests and a chunk about baking
  When the user runs engram query --phrase "writing Go tests" --phrase "how to bake sourdough bread"
  Then the payload's phrases lists both phrases in order, and its items hold the note and the chunk
```

### 792a7b The payload

Requirement: 792a7b `engram query` MUST print one YAML document on stdout with `version: 1`, the
`phrases` it was given, `items`, `clusters`, a `budget` and the `model_id` (792a7zc), and MUST give
each item a `path`, a `kind`, a `score`, its `provenances` and, unless the item was reduced to path
and score (792a7s), its `content`. A note item's `path` MUST be the note's file name and its `kind`
the note's `type`, or `unknown` when it has none; a chunk item's `path` MUST be its chunk ID
`<source>#<anchor>` (792a5) and its `kind` `chunk`. A note's content MUST show each `[[target]]` as
`target` and each `[[target|display]]` as `display`. The `budget` MUST report `phrases_queried`,
`total_notes`, `with_embeddings`, `clusters_found`, `direct_hits_returned` (the items whose
provenances include `direct`), `items_with_full_content`, `limit`, `content_budget`,
`chunks_snippeted` and `explore_allocated`.

Rationale: one call returns both "what is relevant" and "what just happened", so an agent
re-orienting after losing its context reads one document instead of making separate calls. Inline wikilink syntax is noise in the payload, because the matched set is already
returned in `items`.

#### 792a7b1 UAT Scenarios

```gherkin
Scenario: 792a7b1a The payload has the documented shape
  Given a vault of six notes, each with a current sidecar, and no chunk index
  When the user runs engram query --phrase "writing Go tests"
  Then stdout is a YAML document with version 1, phrases ["writing Go tests"], items, clusters and a budget whose total_notes and with_embeddings are both 6, and a model_id

Scenario: 792a7b1b A note item is labelled by its type
  Given a vault holding a fact note
  When the user runs engram query with a phrase that matches it
  Then its item has kind fact, provenances [direct] and its full file text as content

Scenario: 792a7b1c Wikilinks are flattened in the content
  Given a matched note whose body contains [[1.target]] and [[2.other|the other note]]
  When the user runs engram query with a phrase that matches it
  Then its content contains 1.target and the other note, and no [[
```

### 792a7c Embeddings of the current model

Requirement: 792a7c `engram query` MUST ignore a note whose sidecar was made by another embedding
model, and MUST print one warning on stderr naming the number dropped, their model IDs and
`engram embed apply --all`. It MUST ignore a note whose sidecar is on an old schema and print one
warning naming the number skipped and `engram embed apply --force`. When the vault has notes but none
has a usable sidecar it MUST fail with `query: vault has notes but no current-model embeddings; run
`engram embed apply --all``.

Rationale: a silent model swap would otherwise empty the results without saying why, so the
warning names the cause and the remedy. The all-mismatch case fails because an empty payload would
read as "nothing relevant".

#### 792a7c1 UAT Scenarios

```gherkin
Scenario: 792a7c1a Notes of another model are dropped with a warning
  Given a vault of ten notes of which nine have sidecars made by another model
  When the user runs engram query --phrase pasta
  Then stderr says "dropped 9 sidecar(s)" and names the other model and the active one, and the one note with a current sidecar can still be returned

Scenario: 792a7c1b A vault with no usable embeddings is an error
  Given a vault of notes whose sidecars are all from another model, or missing
  When the user runs engram query --phrase pasta
  Then it fails with "query: vault has notes but no current-model embeddings; run `engram embed apply --all`"

Scenario: 792a7c1c Old-schema sidecars are skipped with their own warning
  Given a vault in which a sidecar is on an old schema version
  When the user runs engram query --phrase pasta
  Then stderr says "1 sidecar(s) on an old schema were skipped" and names engram embed apply --force
```

### 792a7d Matched notes and chunks

Requirement: 792a7d For each phrase `engram query` MUST score every note by the higher of its
situation and body cosine to the phrase, and every indexed chunk by its cosine, rank them together
by recency-scaled score (792a7f) with ties by path, and keep the 30 best. It MUST then merge the
phrases keeping each item's best score, drop every item whose unscaled cosine is below 0.25, and
keep at most the 300 highest-scored. Every item kept MUST carry the provenance `direct`. The
matched notes MUST come first in `items`, highest score first, and the matched chunks, highest score
first, MUST follow the notes and the other note items (792a7m, 792a7n).
Only question notes (792a3t) MUST be left out of the matched set: a vocabulary definition note
(792a6a) MUST be matched like any other note.

Rationale: the floor is applied to the unscaled cosine, so a topically perfect but old item still
qualifies (vault note 53). The cap bounds the cost of clustering the set, whose silhouette scoring
is quadratic in its size, at a few hundred items however large the vault and the chunk index grow.
Notes and chunks are ranked together so that a transcript fragment and the lesson distilled from it
compete on one scale.

#### 792a7d1 UAT Scenarios

```gherkin
Scenario: 792a7d1a Notes are ranked by their better axis
  Given a note whose situation is close to the phrase and whose body is not, and a note whose body is close and whose situation is not
  When the user runs engram query with that phrase
  Then each note's score is its higher cosine, and the notes are ordered by it, highest first

Scenario: 792a7d1b An item below the floor is dropped
  Given a note and a chunk whose cosine to the phrase is below 0.25
  When the user runs engram query with that phrase
  Then neither appears among the direct matches

Scenario: 792a7d1c Only the best 30 of a phrase are kept
  Given a phrase that 40 chunks match above the floor
  When the user runs engram query with it
  Then 30 of them are direct matches, the 30 with the highest scores

Scenario: 792a7d1d An item keeps its best score across phrases
  Given a note that scores 0.4 for one phrase and 0.7 for another
  When the user runs engram query with both phrases
  Then the note appears once, scored 0.7

Scenario: 792a7d1e The matched set is capped at 300
  Given twelve phrases that each match 30 different chunks above the floor
  When the user runs engram query with them
  Then 300 matched items are clustered, the 300 with the highest scores

Scenario: 792a7d1f Notes and chunks compete in one list
  Given a vault note and an indexed chunk that match the phrase
  When the user runs engram query with that phrase
  Then both appear, the note first

Scenario: 792a7d1g A definition note is an ordinary match
  Given a vault whose only note close to the phrase is a vocabulary definition note
  When the user runs engram query with that phrase
  Then the definition note is returned

Scenario: 792a7d1h A note is found by its own situation
  Given several notes whose body text is their situation
  When the user runs engram query with one note's situation as the phrase
  Then that note is among the items, with cosine 1
```

### 792a7e Matched-note floor

Requirement: 792a7e Before keeping a phrase's 30 best, `engram query` MUST reserve up to 5 of the
slots for notes whose unscaled cosine is at least 0.25, promoting the best such notes ranked below
the cut and evicting the lowest-scoring chunks to make room. A note whose unscaled cosine is below
0.25 MUST NOT be promoted.

Rationale: without the reservation, a flood of transcript chunks with higher cosine pushed a
crystallized lesson out of the matched set entirely: a note the embedder ranks in its top five when
notes are searched alone reached the top five of the combined ranking only 22% of the time, against
83% once the reservation existed (measured 2026-06-28, `dev/eval/LEDGER.md#matched-note-floor`). The
floor means a promoted note is relevant in its own right, so the reservation never surfaces an
irrelevant one.

#### 792a7e1 UAT Scenarios

```gherkin
Scenario: 792a7e1a A relevant note survives a flood of chunks
  Given a phrase that matches 30 or more chunks above a note that scores 0.4 for it
  When the user runs engram query with that phrase
  Then the note appears among the results, in place of the lowest-scoring chunk

Scenario: 792a7e1b A note below the floor is not promoted
  Given a phrase that matches 30 or more chunks, and a note that scores 0.2 for it
  When the user runs engram query with that phrase
  Then the note is not among the results

Scenario: 792a7e1c Several notes share the reserved slots
  Given a phrase that 40 chunks match above seven notes that each score above 0.25
  When the user runs engram query with that phrase
  Then five of the notes appear, the five with the highest scores

Scenario: 792a7e1d A definition note is promoted like any other
  Given a phrase that 30 chunks match above a vocabulary definition note that scores 0.45
  When the user runs engram query with that phrase
  Then the definition note appears among the results
```

### 792a7f Recency scaling

Requirement: 792a7f `engram query` MUST multiply an item's cosine by a recency factor of
2 to the power of minus its age in days over 60 before ranking, so that an item loses half its score
every 60 days. A chunk's age MUST be the time since it was first indexed (792a7p), and a chunk from
a transcript turn MUST also gain up to 20% in proportion to how late in its session the turn
falls. A note's age MUST be the time since the date in its sidecar's `last_used`, or, when it has
none, since its `created` date when that is written as a bare date; a note with neither MUST NOT be
scaled. An item with an age of zero, or with an unreadable date, MUST NOT be scaled.

Rationale: operators work in monthly cycles, so the decay is a soft tilt that favours recent
content without drowning relevance (2 weeks 0.85, 1 month 0.71, 2 months 0.50). The newest chunks
are guaranteed a place separately by the recency channel (792a7o), so the scaling need not carry
that guarantee. A recency-weighted cluster centroid was tried and did not ship (792a7h). Notes that
`engram learn` writes quote their `created` date, which cannot be read as a bare date, so such a
note is not scaled until it has been used (#774).

#### 792a7f1 UAT Scenarios

```gherkin
Scenario: 792a7f1a An old chunk is scaled down
  Given an indexed chunk first indexed 120 days ago, and the same chunk indexed today
  When the user runs engram query with a phrase that matches it
  Then the old chunk's score is a quarter of the fresh chunk's

Scenario: 792a7f1b A long-unused note is scaled down
  Given two notes with the same text, one whose sidecar says last_used today and one 120 days ago
  When the user runs engram query with a phrase that matches them
  Then the first scores four times the second and is listed first

Scenario: 792a7f1c The floor ignores the scaling
  Given a chunk first indexed 120 days ago whose cosine to the phrase is 0.53
  When the user runs engram query with that phrase
  Then the chunk is a direct match, scored about 0.13

Scenario: 792a7f1d A note that was never used is not scaled
  Given a note engram learn wrote 120 days ago, with no last_used in its sidecar
  When the user runs engram query with a phrase that matches it
  Then its score is its cosine

Scenario: 792a7f1e A late turn is lifted
  Given two chunks of one transcript with the same cosine to the phrase, one at its first turn and one at its last, both first indexed today
  When the user runs engram query with a phrase that matches them
  Then the last turn's score is 1.2 times the first turn's
```

### 792a7g Clusters

Requirement: 792a7g `engram query` MUST cluster the matched notes and chunks together, once for
all the phrases and not once per phrase, into `clusters`. Each cluster MUST have an `id`, an empty
`phrase`, a `size`, a `silhouette` (the mean silhouette score of its members, 0 for the one-cluster
case below), and `members` listing each member's `path` and `score`, highest
score first, with exactly one member marked `is_representative`: the member whose vector is closest
to the cluster's centroid, ties going to the higher score and then to the lower path. The number of
clusters MUST be between 2 and 7 when a split scores a silhouette of at least 0.10, and otherwise
the whole matched set MUST be one cluster with silhouette 0. Nothing matched MUST give no clusters.
The same vault and phrases MUST give the same clusters.

Rationale: the clusters group what was found by topic so that the reader judges each topic once.
Clustering the union once keeps the number of clusters small, and the seed is derived from the
phrases, so the result does not vary between runs.

#### 792a7g1 UAT Scenarios

```gherkin
Scenario: 792a7g1a Matches form clusters with a representative
  Given a vault whose matches fall into two topics
  When the user runs engram query with phrases for both
  Then clusters holds two clusters, each listing its members by score and marking exactly one representative

Scenario: 792a7g1b Matches with no structure form one cluster
  Given a vault whose matches are too alike to split
  When the user runs engram query with a phrase that matches them
  Then clusters holds one cluster holding every match, with silhouette 0

Scenario: 792a7g1c Nothing matched means no clusters
  Given a vault with no note above the floor
  When the user runs engram query with a phrase
  Then clusters is empty

Scenario: 792a7g1d The same query gives the same clusters
  Given a vault and a set of phrases
  When the user runs engram query with them twice
  Then both payloads hold the same clusters
```

### 792a7h Candidate notes

Requirement: 792a7h `engram query` MUST give each cluster that has note members a `candidate_l2s`
list of its own notes, at most five, ranked by the higher of each note's situation and body cosine
to the cluster's centroid, highest first with ties by path, each with its `path`, `cosine` and
`content`. A cluster with no note members MUST have no `candidate_l2s`. The list MUST hold nothing
but the cluster's own notes: not notes sampled for other topics (792a7i) and not superseding notes
(792a7n), which travel in `items`.

Rationale: the cosine only nominates; whether a candidate covers the situation is the reader's
judgement. The centroid is the unweighted mean of the cluster's vectors: a centroid weighted by each
member's recency was measured on the production vault, on the 16 same-cluster pairs where one note
supersedes another and the cluster has more than five notes, and it changed nothing, with 8 of 16
(50.0%) superseded notes over-surfaced either way and a mean top-5 relevance of 0.6480 for both
(`dev/eval/LEDGER.md#652-recency-weighted-centroid-null`), so it did not ship.

#### 792a7h1 UAT Scenarios

```gherkin
Scenario: 792a7h1a Candidates are the cluster's nearest notes
  Given a cluster of seven notes and three chunks
  When the user runs engram query with a phrase that matches them
  Then candidate_l2s lists five of the notes, ordered by cosine to the centroid, and no chunk

Scenario: 792a7h1b A cluster of chunks has no candidates
  Given a cluster whose members are all chunks
  When the user runs engram query with a phrase that matches them
  Then that cluster has no candidate_l2s

Scenario: 792a7h1c A small cluster lists all its notes
  Given a cluster of three notes
  When the user runs engram query with a phrase that matches them
  Then candidate_l2s lists all three
```

### 792a7i Half of the notes are explored

Requirement: 792a7i The notes `engram query` delivers MUST be an exploit half, the notes matched
above (792a7d), and an explore half sampled from the vocabulary's terms (792a6) by their
closeness to the query. The explore half's budget MUST be the number of distinct notes in the
exploit half, so the halves are equal in notes, and chunks MUST count towards neither. When the
exploit half holds no note the query MUST skip explore sampling. When the explore half cannot be
filled, because the vault has too few distinct notes or `vocab.centroids.json` is absent or
unreadable, it MUST be smaller, and the budget's `explore_allocated` MUST show the shortfall, as `{}`
when it is empty.

Rationale: matches alone only find what resembles the query, so global reach comes from sampling
near concepts. Tag nomination did this by joining any note that shared a term with the top three
notes of a cluster, whatever its cosine, which tied candidates to an accident of rank, gave every
term-sharing note equal footing, and needed a per-cluster cap; it was replaced on 2026-07-29.
The missing-centroids case fails loudly, as an empty allocation, rather than silently.

#### 792a7i1 UAT Scenarios

```gherkin
Scenario: 792a7i1a The halves are equal on a healthy vault
  Given a bootstrapped vault with enough notes
  When the user runs engram query with a phrase that matches two notes
  Then the payload holds those two notes and two explore notes

Scenario: 792a7i1b Missing centroids leave only the matches
  Given a vault whose vocab.centroids.json is absent or unreadable
  When the user runs engram query with a phrase that matches notes
  Then no item has provenance explore and the budget's explore_allocated is {}

Scenario: 792a7i1c No matched note means no explore half
  Given a vault whose only matches are chunks
  When the user runs engram query with a phrase
  Then no item has provenance explore and explore_allocated is {}

Scenario: 792a7i1d A small vault shortens the explore half
  Given a vault whose notes are all in the exploit half
  When the user runs engram query with a phrase
  Then the explore half is smaller than the exploit half and explore_allocated sums to the number delivered
```

### 792a7j Explore allocation follows the query

Requirement: 792a7j `engram query` MUST divide the explore budget among the vocabulary's terms in
proportion to the softmax of the cosine between the mean of the phrases' vectors and the term's
centroid, divided by a temperature of 0.106, so that every term with a centroid takes part, nearer
terms are allocated at least as much as farther ones, and the allocations total the budget. A term
that has at least one member in the exploit half MUST have 0.05 added to that cosine first, once, no
matter how many members it has.

Rationale: a hard radius is brittle in cosine space, whereas a softmax needs one temperature and
degrades continuously, so far terms approach an allocation of zero without a cutoff. The temperature
was calibrated on ten queries against the production vault (best centroid cosine 0.550 and median
0.307, giving (0.550 - 0.307) / ln 10 = 0.106; `dev/eval/LEDGER.md#recall-explore-tau-calibration`).
The bonus keeps the one thing tag nomination did well, reach to a term whose notes already matched,
and is a constant so it cannot compound.

#### 792a7j1 UAT Scenarios

```gherkin
Scenario: 792a7j1a Nearer terms sample more
  Given two terms, one with a centroid nearer the query
  When the explore budget is divided
  Then the nearer term's allocation is at least the other's, and together they equal the budget

Scenario: 792a7j1b Evidence lifts a distant term
  Given a distant term with one member in the exploit half and an equally distant term with none
  When the explore budget is divided
  Then the term with a member is allocated more

Scenario: 792a7j1c The bonus does not stack
  Given two terms of equal distance, one with one member in the exploit half and one with three
  When the explore budget is divided
  Then the two are allocated the same
```

### 792a7k Explore picks the core of a term

Requirement: 792a7k `engram query` MUST pick a term's explore notes from the notes tagged
`vocab/<term>`, leaving out definition notes (792a6a), question notes and notes without a vector,
in descending order of the cosine between each note's body vector and the term's centroid, ties by
path. A note with several terms MUST be a candidate under each.

Rationale: members near the query are already in the exploit half, so the explore half shows each
concept by its core, and the core excludes the fringe of notes that only just cleared the
assignment floor of 0.35.

#### 792a7k1 UAT Scenarios

```gherkin
Scenario: 792a7k1a The core outranks the fringe
  Given a term with five member notes and an allocation of two
  When the explore notes are picked
  Then they are the two members closest to the term's centroid

Scenario: 792a7k1b Definition notes are never picked
  Given a term whose definition note is the closest note to its centroid
  When the explore notes are picked
  Then the definition note is not among them
```

### 792a7l Explore picks are not repeated

Requirement: 792a7l `engram query` MUST drop an explore pick that is already in the exploit half or
that another term already picked, and MUST refill each freed slot from the terms in order of
allocation, until the explore budget is met or no candidates remain.

#### 792a7l1 UAT Scenarios

```gherkin
Scenario: 792a7l1a A duplicate is replaced
  Given an explore pick that is already in the exploit half, and other terms with unpicked members
  When the explore notes are picked
  Then the duplicate is dropped and its slot is filled from the next term in allocation order

Scenario: 792a7l1b A note under two terms is picked once
  Given a note that is the nearest member of two allocated terms
  When the explore notes are picked
  Then it appears once

Scenario: 792a7l1c Exhausted candidates leave the budget unmet
  Given fewer eligible notes than the explore budget
  When the explore notes are picked
  Then every eligible note is picked and the half is smaller than its budget
```

### 792a7m Explore picks are reported

Requirement: 792a7m `engram query` MUST give every explore note the provenance `explore` and a
`source_term` naming its term, the cosine to that term's centroid as its `score`, and its full
content, and MUST place the explore notes in `items` after the matched notes and before the matched
chunks. They MUST NOT be members of any cluster or appear in any `candidate_l2s`. The budget's
`explore_allocated` MUST map each contributing term to the number of notes delivered for it, after
duplicates are dropped, and MUST be present in every payload.

#### 792a7m1 UAT Scenarios

```gherkin
Scenario: 792a7m1a An explore note names its term
  Given a bootstrapped vault with a term go-tooling
  When the user runs engram query with a phrase about cooking
  Then an item has provenance explore, source_term go-tooling and a score equal to its cosine to that term's centroid

Scenario: 792a7m1b Explore notes are outside the clusters
  Given a payload with an explore note
  When its clusters are read
  Then the explore note is not a member of any cluster and not in any candidate_l2s

Scenario: 792a7m1c The budget counts what was delivered
  Given a payload whose explore half holds two notes of one term
  When its budget is read
  Then explore_allocated maps that term to 2
```

### 792a7n Superseding notes ride along

Requirement: 792a7n When a delivered matched note has been superseded, because a newer note's
`supersedes:` list names it with `updates`, `narrows` or `refutes`, `engram query` MUST insert the
newer note in `items` directly after it, with the provenance `ride_along` and a score of 0, whether
or not the newer note matched. When several notes supersede one note it MUST insert them in order,
and it MUST NOT insert a note that is already among the results, or one that is not in the vault
with a usable sidecar. Only matched notes MUST be examined, not chunks, recent chunks or notes
already inserted.

Rationale: a note that corrects another should reach the reader with it. The provenance is distinct
so that an analysis of rank shifts can tell an insertion from a match.

#### 792a7n1 UAT Scenarios

```gherkin
Scenario: 792a7n1a The newer note follows the older one
  Given a note P matched by the phrase, and a note N that supersedes P but does not match
  When the user runs engram query with that phrase
  Then N is the item directly after P, with provenance ride_along and score 0

Scenario: 792a7n1b A note already present is not inserted again
  Given both P and N matched
  When the user runs engram query with a phrase that matches them
  Then N appears once

Scenario: 792a7n1c Several superseders are all inserted
  Given two notes N1 and N2 that both supersede P
  When the user runs engram query with a phrase that matches P
  Then N1 and N2 follow P in order

Scenario: 792a7n1d A missing superseder is skipped
  Given a note P whose superseder has no usable sidecar
  When the user runs engram query with a phrase that matches P
  Then no item is inserted and the query succeeds
```

### 792a7o The newest chunks

Requirement: 792a7o `engram query` MUST append to `items`, after the matched items, the newest
chunks of the chunk index by the time they were first indexed (792a7p), newest first and with ties
going to the later turn, skipping chunks that are already matched, up to the `--recent-fill` count.
That count MUST be 25 by default, `ENGRAM_RECENT_FILL` MUST set it from the environment, a positive
`--recent-fill` MUST replace it, and a negative one MUST turn the channel off. Each chunk MUST carry
the provenance `recent` and a score of 0, MUST NOT be a member of any cluster, and MUST NOT count
towards `--limit` (792a7q). Fewer chunks than the count MUST give all of them. A chunk with no
first-indexed time MUST sort after every chunk that has one.

Rationale: the agent that has lost its context re-encounters its own recent narration and recovers
what it was doing from recency, with no separate provenance mechanism. The channel is
non-load-bearing for recall, so a small count shrinks the payload: a count of 25 cut a real-vault
payload by 28% (230 to 165 KB, `dev/eval/LEDGER.md#payload-cut-recent-fill`). A zero value stands for
the default, because an unset flag and an unset environment variable both read as zero; to turn the
channel off the value must be negative.

#### 792a7o1 UAT Scenarios

```gherkin
Scenario: 792a7o1a Recent chunks follow the matches
  Given a chunk index of three chunks, one of which matches the phrase
  When the user runs engram query with that phrase
  Then items ends with the two other chunks, newest first, each with provenance recent and score 0, and they are in no cluster

Scenario: 792a7o1b A matched chunk is not repeated
  Given a chunk that is both matched and among the newest
  When the user runs engram query with a phrase that matches it
  Then it appears once, with provenance direct

Scenario: 792a7o1c The count can be set
  Given a chunk index of three chunks and none matched
  When the user runs engram query --recent-fill 1, or sets ENGRAM_RECENT_FILL=1
  Then exactly one recent chunk is returned, the newest

Scenario: 792a7o1d A negative count turns the channel off
  Given a chunk index of three chunks
  When the user runs engram query --recent-fill=-1
  Then no item has provenance recent

Scenario: 792a7o1e Zero means the default
  Given a chunk index of three chunks
  When the user runs engram query --recent-fill 0
  Then all three chunks are returned as recent, as with no flag

Scenario: 792a7o1f A chunk without a time sorts last
  Given a chunk index holding a chunk with no first-indexed time and a chunk with one
  When the user runs engram query with a count of 1
  Then the chunk with a time is the one returned
```

### 792a7p When a chunk was first indexed

Requirement: 792a7p Every chunk `engram ingest` indexes MUST record the time it was first indexed:
for a Markdown source the time of the ingest, and for a transcript the timestamp of the last entry
the reader returned. Ingesting a source again MUST keep the recorded time of every chunk it already
holds. A chunk in an index written before times were recorded MUST be given its source's recorded
modification time the next time the source is indexed again, or no time when the manifest has none.

Rationale: recency used to be approximated from the source file's modification time, so one edit
made every chunk in a source look new. A per-chunk time orders chunks across and within sessions;
all chunks of one transcript share the session's last timestamp, which is a sufficient
approximation because each session is its own source.

#### 792a7p1 UAT Scenarios

```gherkin
Scenario: 792a7p1a A Markdown chunk takes the ingest time
  Given a Markdown document ingested at 2026-06-15 12:00
  When its index file is read
  Then each record's ingested_at is 2026-06-15 12:00

Scenario: 792a7p1b A transcript chunk takes the session's last timestamp
  Given a transcript whose last entry is stamped 2026-05-10 14:00, ingested on 2026-06-01
  When its index file is read
  Then each record's ingested_at is 2026-05-10 14:00

Scenario: 792a7p1c Ingesting again keeps the first time
  Given an ingested transcript whose chunk is unchanged on a second ingest
  When the second ingest runs
  Then the chunk's ingested_at is still the first one

Scenario: 792a7p1d An old chunk gets its source's modification time
  Given an index record with no ingested_at whose source the manifest records as modified 2026-01-20
  When the source is ingested again
  Then the record's ingested_at is 2026-01-20
```

### 792a7q Limit

Requirement: 792a7q `engram query` MUST keep at most `--limit` of the items that are neither trigger
hits (792a7za) nor recent chunks (792a7o), the highest-ranked first, applied after the recency
channel is assembled and before chunk content is reduced (792a7r). The limit MUST be 20 when the
flag is absent, MUST come from the flag alone, and MUST be reported as the budget's `limit`. Trigger
hits MUST NOT be counted against it, so `items` can hold more than `--limit` items of the first
kind by the number of hits, and recent chunks MUST NOT be counted against it, so a full set of
matches never displaces them. It MUST apply in the same way to a query merged with a parent vault.

Rationale: before this limit `--limit` only reported a number, and the item count was whatever
clustering and nomination produced, with no ceiling. Counting the recent chunks under the same limit
would displace the whole channel whenever the matches alone reached it, which in a vault of any size
is routine, and so defeat `--recent-fill`.

#### 792a7q1 UAT Scenarios

```gherkin
Scenario: 792a7q1a The default caps a large result
  Given a vault whose matches exceed 20 items
  When the user runs engram query with no --limit
  Then items holds exactly 20 of them, the highest ranked, and the budget's limit is 20

Scenario: 792a7q1b An explicit limit caps the result
  Given a vault whose matches exceed 5 items
  When the user runs engram query --limit 5
  Then items holds at most 5 of them

Scenario: 792a7q1c A small result is returned whole
  Given a vault whose matches total fewer than the limit
  When the user runs engram query
  Then every match is returned

Scenario: 792a7q1d Trigger hits do not use the limit
  Given a message that hits one runbook, and several matched notes
  When the user runs engram query --text "<message>" --limit 1 with a phrase
  Then items lists the hit first, then one matched item, and the hit is not counted against the limit

Scenario: 792a7q1e The recent chunks survive a full set of matches
  Given matches that alone reach the limit, and a chunk index with recent chunks
  When the user runs engram query --limit 2
  Then items holds two matched items followed by the recent chunks
```

### 792a7r Chunk content budget

Requirement: 792a7r `engram query` MUST give the first `--content-budget` chunk items of `items`, in
the order printed, their full content, and MUST reduce the content of every later chunk item to a
snippet: its whitespace collapsed to single spaces, trimmed, and cut to at most 160 characters,
the last being `…` when it was cut. The budget MUST be 15 by default, `ENGRAM_CONTENT_BUDGET` MUST set
it from the environment, a negative value MUST mean no limit, and zero MUST stand for the default. Note items MUST never be
reduced.
The budget MUST report the budget used as `content_budget`, the number of items still carrying full
content as `items_with_full_content`, and the number reduced as `chunks_snippeted`.

Rationale: the budget sits at the knee of the cost and quality curve with headroom for long-chunk
recalls that the eval harnesses do not probe (2026-06-24).

#### 792a7r1 UAT Scenarios

```gherkin
Scenario: 792a7r1a Later chunks become snippets
  Given a payload with three chunk items
  When the user runs engram query with a content budget of 1
  Then the first chunk keeps its content, the other two carry snippets, and chunks_snippeted is 2

Scenario: 792a7r1b A long chunk is cut with an ellipsis
  Given a chunk item of 300 characters beyond the budget
  When the user runs engram query with a content budget of 0 chunks kept in full
  Then its content is 160 characters ending in …

Scenario: 792a7r1c Notes are never reduced
  Given a payload with a note and chunks beyond the budget
  When the user runs engram query with a content budget of 1
  Then the note keeps its full content

Scenario: 792a7r1d A negative budget means none
  Given a payload with 20 chunk items
  When the user runs engram query --content-budget=-1
  Then every chunk keeps its full content
```

### 792a7s Lazy chunks

Requirement: 792a7s When `--lazy-chunks` is set, or `ENGRAM_LAZY_CHUNKS` is true, `engram query`
MUST print each chunk item with its `path` and `score` but no content, MUST leave note items
with their full content, and MUST report `lazy_chunks: true` in the budget. A payload without the
flag MUST NOT contain `lazy_chunks`.

Rationale: a recall pass should carry less to read without losing reach, so the evidence is fetched
on demand with `engram show-chunk` (792a7t) and not delivered up front. On the real vault the
payload shrank 33.7% (146 to 97 KB), and across 13 realistic recalls that were not told to fetch,
none fetched a chunk, while a chunk that was the only source of a needed fact was fetched when it
mattered (`dev/eval/LEDGER.md#payload-cut-lazy-chunks`).

#### 792a7s1 UAT Scenarios

```gherkin
Scenario: 792a7s1a Chunks carry no content
  Given a vault with a matching chunk
  When the user runs engram query --lazy-chunks
  Then the chunk item has a path and a score and no content, and the budget says lazy_chunks: true

Scenario: 792a7s1b Notes keep their content
  Given a vault with a matching note
  When the user runs engram query --lazy-chunks
  Then the note item keeps its full content

Scenario: 792a7s1c The environment can turn it on
  Given ENGRAM_LAZY_CHUNKS=true
  When the user runs engram query with a phrase that matches a chunk
  Then the chunk item has no content

Scenario: 792a7s1d Without the flag the field is absent
  Given a vault with a matching chunk
  When the user runs engram query with no --lazy-chunks
  Then the chunk item has its content and the budget has no lazy_chunks
```

### 792a7t Reading one chunk

Requirement: 792a7t `engram show-chunk <source>#<anchor>` MUST print the text of the chunk with
that chunk ID, followed by a newline when the text lacks one, and MUST NOT write anything. It MUST
fail with `chunk not found: <id>` when no chunk has that ID, and with `show-chunk: empty chunk ref`
when the ID is empty or blank.

#### 792a7t1 UAT Scenarios

```gherkin
Scenario: 792a7t1a A chunk's text is printed
  Given an ingested document with a "Baking" section
  When the user runs engram show-chunk "<document>#Baking"
  Then stdout is that section's text

Scenario: 792a7t1b An unknown chunk is an error
  Given a chunk index that holds no chunk "nosuch#x"
  When the user runs engram show-chunk "nosuch#x"
  Then it fails with "chunk not found: nosuch#x" and exits non-zero

Scenario: 792a7t1c An empty ID is an error
  Given any chunk index
  When the user runs engram show-chunk with a blank ID
  Then it fails with "show-chunk: empty chunk ref"
```

### 792a7u Project filter

Requirement: 792a7u `engram query --project <slug>` MUST drop each matched note whose frontmatter
`project:` line is not the slug, and each matched note without a readable frontmatter, and MUST NOT
take a mention of the slug in a note's body as a match. It MUST leave every other item, and the
clusters, unchanged, and without `--project` it MUST drop nothing.

#### 792a7u1 UAT Scenarios

```gherkin
Scenario: 792a7u1a Notes of other projects are dropped
  Given matched notes with project: alpha and project: beta
  When the user runs engram query --project alpha
  Then only the alpha note remains among the matched notes

Scenario: 792a7u1b A body mention does not match
  Given a matched note whose body says "project: alpha" and whose frontmatter has no project
  When the user runs engram query --project alpha
  Then the note is dropped

Scenario: 792a7u1c No filter drops nothing
  Given matched notes of several projects
  When the user runs engram query with no --project
  Then all of them are returned
```

### 792a7v Runbooks rank like other notes

Requirement: 792a7v A `runbook` note MUST be a candidate for `engram query` on the same terms as a
`fact` or `feedback` note, with the same floor, phrase limit and deduplication, and MUST appear in
`items` when it ranks above the floor. It MUST rank by the similarity of its situation alone, with
no query flag, pre-filter or boost for its kind; a trigger hit (792a7za) MUST be the only exception.
Its presence in `items` MUST be what counts as surfacing: a runbook missing from its cluster's
`candidate_l2s` has still surfaced.

Rationale: a `task_type` classification field with a pre-filter was considered and rejected: the
benchmark evidence for it measured a whole bundled system, not type classification on its own, which
was too little to justify a new ranking mechanism and its risk to existing retrieval. Only
`qa-question` notes are withheld from retrieval, so a runbook needs no exclusion (ADR-0026). A literal cue the author declares is the one exception, because the agent
paraphrases the user's request into its phrases and the paraphrase drops the cue: on a real `/please`
ask "please" and "end-to-end" vanished and the deliverable was named instead, and adding the user's
words as a third phrase changed the runbook's rank or score in none of 95 scratch-vault cells.

#### 792a7v1 UAT Scenarios

```gherkin
Scenario: 792a7v1a A runbook is returned like a fact
  Given a runbook whose situation is close to the phrase
  When the user runs engram query with that phrase
  Then it appears in items with kind runbook

Scenario: 792a7v1b Surfacing does not need a candidate slot
  Given a runbook in items whose cluster's candidate_l2s does not list it
  When the payload is read
  Then the runbook has surfaced

Scenario: 792a7v1c A runbook has no kind boost
  Given a runbook and a feedback note whose situations are equally close to the phrase, and no trigger hit
  When the user runs engram query with that phrase
  Then their order follows their scores alone
```

### 792a7w Red flags in the query preview

Requirement: 792a7w When a runbook is a direct match, a candidate note or a trigger hit, `engram query`
MUST render its `red_flags` in its content. When the list takes more than 1200 bytes of rendered
YAML, `engram query` MUST keep the newest entries, those listed last, that fit and MUST replace the
earlier ones with one marker entry
`[<n> EARLIER RED_FLAGS OMITTED — run engram show <basename> for all <total>]`, giving the number
omitted, the list's total and the note's own basename. A list
within 1200 bytes MUST be rendered unchanged, and a note without red flags MUST be unchanged.

Rationale: `engram learn runbook --red-flag` and `engram amend --red-flag` append, so the newest entry
is last. In #763 a runbook's fifteenth and newest entry, added after fourteen, sat past a truncation
boundary of about 2 KB applied outside engram and never reached the agent. Dropping from the front
keeps the entry most likely to matter now, and the marker tells the reader that it has not seen the
whole list and how to get it.

#### 792a7w1 UAT Scenarios

```gherkin
Scenario: 792a7w1a Red flags are in the payload
  Given a runbook with two short red flags
  When the user runs engram query with a phrase that matches it
  Then its item's content includes both entries

Scenario: 792a7w1b An oversized list keeps its newest entries
  Given a runbook with 30 red flags of about 100 characters each
  When the user runs engram query with a phrase that matches it
  Then its content holds a marker saying "20 EARLIER RED_FLAGS OMITTED" with the note's basename and "all 30", followed by the last ten entries

Scenario: 792a7w1c A short list is unchanged
  Given a runbook whose red flags take under 1200 bytes
  When the user runs engram query with a phrase that matches it
  Then its red_flags block is identical to the note file's

Scenario: 792a7w1d A trigger hit is truncated the same way
  Given the same oversized runbook and a message that hits one of its triggers
  When the user runs engram query --text "<message>"
  Then its content holds the same marker
```

### 792a7x Reading one note

Requirement: 792a7x `engram show <ref>` MUST print the content of the note the ref names,
frontmatter and body exactly as the file holds them, with every `red_flags` entry and no omission marker, then a line
`# outbound links (fetch with: engram show <basename>)` followed by the target of each wikilink in
the note's text, each once in the order it first appears, with a trailing `.md` dropped and a
`|display` part kept, one per line, or `(none)`. A link inside a fenced code block MUST NOT be
listed. The ref
MUST be a full basename, a basename in `[[ ]]` with or without `|display`, a basename with a
trailing `.md`, or a bare Luhmann ID, and MUST fail with `show: note not found: "<ref>"` when no
note matches and with `show: empty note reference` when blank. It MUST NOT change any note. A note that
has been exchanged is preceded by one `# exchange_hash:` line, which belongs to curate.

Rationale: `engram show` is the full-fidelity source that the guidance shim names, so an agent can
restate every step and check every red flag when the query payload truncated them (792a7w). It
therefore never applies the preview budget.

#### 792a7x1 UAT Scenarios

```gherkin
Scenario: 792a7x1a A note is printed with its links
  Given a note 1.hub that links [[2.alpha]] and [[3.beta]]
  When the user runs engram show 1.hub
  Then stdout is the note, then the outbound links line and 2.alpha and 3.beta

Scenario: 792a7x1b A note with no links says so
  Given a note that links to nothing
  When the user runs engram show <basename>
  Then stdout ends with the outbound links line and (none)

Scenario: 792a7x1c Every reference form resolves
  Given the note 1.2026-10-09.go-testing
  When the user runs engram show with 1.2026-10-09.go-testing, [[1.2026-10-09.go-testing]], 1.2026-10-09.go-testing.md or 1
  Then each prints the same note

Scenario: 792a7x1d Show never truncates red flags
  Given a runbook whose red flags take more than 1200 bytes
  When the user runs engram show for it
  Then stdout holds every entry in file order and no omission marker

Scenario: 792a7x1e A fenced link is not listed
  Given a note whose body has [[9.fenced]] inside a code fence and [[2.alpha]] outside it
  When the user runs engram show for it
  Then only 2.alpha is listed

Scenario: 792a7x1f An unknown note is an error
  Given a vault with no note nosuch
  When the user runs engram show nosuch
  Then it fails with show: note not found: "nosuch" and exits non-zero

Scenario: 792a7x1g A blank reference is an error
  Given any vault
  When the user runs engram show with a blank reference
  Then it fails with show: empty note reference

Scenario: 792a7x1h Each link is listed once
  Given a note whose body links [[2.alpha]], [[3.beta.md]], [[2.alpha]] and [[4.gamma|the gamma note]]
  When the user runs engram show for it
  Then the outbound links are 2.alpha, 3.beta and 4.gamma|the gamma note, in that order
```

### 792a7y Trigger text

Requirement: 792a7y `engram query` MUST accept the user's message verbatim as `--text`, and MUST
use it only to look for runbook triggers: it MUST NOT embed it or match it by similarity. With
`--text` and no `--phrase` it MUST answer a trigger lookup alone, embedding nothing, with `phrases`
empty, and no clusters. Without `--text` it MUST NOT look for triggers, and its payload MUST be the
same as it was before `--text` existed.

Rationale: matching the agent's paraphrase of a request against a runbook misses a literal cue like
`/please` too often for cue-triggered skills to rest on similarity alone, so the user's own words are
passed unchanged (ADR-0026). The served form of `--text` belongs to vault-serving.

#### 792a7y1 UAT Scenarios

```gherkin
Scenario: 792a7y1a Text alone is a trigger lookup
  Given a runbook with triggers ["/release"]
  When the user runs engram query --text "please /Release it now"
  Then items holds only that runbook, with provenances [trigger] and score 0, and phrases and clusters are empty

Scenario: 792a7y1b Without text nothing changes
  Given a vault whose runbooks carry triggers
  When the user runs engram query with phrases and no --text
  Then the payload is the same as before --text existed and no item has provenance trigger

Scenario: 792a7y1c Text is not matched by similarity
  Given a runbook whose situation is close to the text but whose triggers are not in it
  When the user runs engram query --text "<that text>"
  Then the runbook is not returned
```

### 792a7za A trigger hit comes first

Requirement: 792a7za When any entry of a runbook's `triggers` occurs in `--text` (792a7zb), the
runbook MUST be a trigger hit: it MUST appear in `items` before every other item, with `trigger`
among its provenances, exempt from the relevance floor, the matched-set cap, the matched-note floor
and `--limit`. A hit that also matched by similarity MUST keep its score and its other provenances,
and a hit that did not MUST have score 0. Only `runbook` notes with a usable sidecar MUST be candidates: a `triggers:`
list on any other kind MUST be ignored. With `--project`, a hit outside the project MUST NOT be surfaced.

Rationale: a literal cue the author declared is evidence that the runbook applies, whatever its
similarity to the agent's paraphrase. A hit is still only a candidate: the agent judges it against
the runbook's own applicability text.

#### 792a7za1 UAT Scenarios

```gherkin
Scenario: 792a7za1a A hit with no similarity match is first
  Given a runbook with triggers ["/please"] and no similarity match above the floor
  When the user runs engram query --text "/please fix the flaky login test" --phrase "fixing a flaky test"
  Then the runbook is items[0] with provenances [trigger] and kind runbook

Scenario: 792a7za1b A hit outranks a stronger match
  Given a feedback note that scores 0.9 and a runbook that scores 0.3 and is a trigger hit
  When the user runs engram query with the message and a phrase
  Then the runbook precedes the feedback note

Scenario: 792a7za1c Other kinds never hit
  Given a fact note whose frontmatter has a triggers list matching the text
  When the user runs engram query --text "<the text>"
  Then it has no trigger provenance and ranks by similarity only

Scenario: 792a7za1d A hit that matched keeps its score
  Given a runbook that both matches a phrase and is a trigger hit
  When the user runs engram query with the phrase and the message
  Then its provenances include direct and trigger and its score is its similarity score

Scenario: 792a7za1e A hit outside the project is not surfaced
  Given a runbook trigger hit whose project is beta
  When the user runs engram query --project alpha with the message
  Then the runbook is not returned
```

### 792a7zb Triggers match whole words

Requirement: 792a7zb A trigger MUST occur in `--text` as a whole word, compared without regard to
case after each run of whitespace in both is collapsed to one space. An occurrence is a whole word
when, on each side where the trigger's own edge character is a letter or a digit, the character next
to the occurrence, if there is one, is not a letter or a digit, where letters and digits are those of
Unicode, so an underscore, a hyphen, a slash and any other punctuation are boundaries, as are the
start and end of the text. On a side where the trigger's edge character is not a letter or a digit
there MUST be no boundary requirement. Every occurrence MUST be tried, so a later one can match when
an earlier one does not.

Rationale: a plain substring let a bare-word trigger such as `curate` fire inside `accurate`. The
bare words `curate` and `please` make such false hits certain, so the boundary is enforced on the
sides that need it. A trigger written with punctuation at its edge, such as `/please`, already
carries its own delimiter, so `do/please` still hits while `/pleased` does not. The rule is two rune
lookups and a hand-written scan, with no pattern compiled from author text. Inflected forms no
longer hit, because the author lists the forms that should: `curate` does not match `curated`.

#### 792a7zb1 UAT Scenarios

```gherkin
Scenario: 792a7zb1a Case and whitespace are ignored
  Given triggers ["take this end-to-end"]
  When --text is "PLEASE   take this end-to-end: rename X"
  Then the runbook is a trigger hit

Scenario: 792a7zb1b A bare word hits as a whole word
  Given triggers ["curate"]
  When --text is "please curate the offers", "Curate!" or "(curate)"
  Then the runbook is a trigger hit

Scenario: 792a7zb1c A bare word does not hit inside a longer word
  Given triggers ["curate"]
  When --text is "that is accurate", "inaccurate", "curated offers" or "curates"
  Then the runbook is not a trigger hit

Scenario: 792a7zb1d A slash form needs only a right boundary
  Given triggers ["/please"]
  When --text is "/please fix it" or "do /please", and when it is "/pleased"
  Then the first two hit and the last does not

Scenario: 792a7zb1e A slash before a bare word is a boundary
  Given triggers ["curate"]
  When --text is "/curate"
  Then the runbook is a trigger hit

Scenario: 792a7zb1f The edges of the text are boundaries
  Given triggers ["curate"]
  When --text is "curate"
  Then the runbook is a trigger hit

Scenario: 792a7zb1g A later occurrence can match
  Given triggers ["curate"]
  When --text is "accurate, curate now"
  Then the runbook is a trigger hit

Scenario: 792a7zb1h Unicode letters and digits are not boundaries
  Given triggers ["curate"]
  When --text is "écurate", "curateé" or "curate2 now"
  Then the runbook is not a trigger hit

Scenario: 792a7zb1i Hyphen and underscore are boundaries
  Given triggers ["curate"]
  When --text is "re-curate-now" or "do_curate_it"
  Then the runbook is a trigger hit
```

### 792a7zc The embedding model

Requirement: 792a7zc `engram query` MUST put in its payload the `model_id` of the embedding model that
produced its results.

Rationale: a consumer comparing or fusing results from more than one vault can then tell whether
their models match before treating the scores as comparable.

#### 792a7zc1 UAT Scenarios

```gherkin
Scenario: 792a7zc1a Every payload names its model
  Given any vault
  When the user runs engram query with a phrase
  Then the payload's model_id is minilm-l6-v2@384
```

### 792a7zd A refit is pending

Requirement: 792a7zd `engram query` MUST print `refit_pending: true` when the vault's
`vocab.centroids.json` records a pending vocabulary refit (792a6s), and MUST NOT print the field
otherwise, including when the file is absent or unreadable.

#### 792a7zd1 UAT Scenarios

```gherkin
Scenario: 792a7zd1a A pending refit is reported
  Given a vault whose vocab.centroids.json has refit_pending true
  When the user runs engram query with a phrase
  Then the payload has refit_pending: true

Scenario: 792a7zd1b Otherwise the field is absent
  Given a vault whose vocab.centroids.json has no pending refit, or no such file
  When the user runs engram query with a phrase
  Then the payload has no refit_pending
```

### 792a7ze Stage timings

Requirement: 792a7ze When `--timings` is set, `engram query` MUST add a `timings` block to the
payload with `scan_ms`, `embed_ms`, `cluster_ms`, `nominate_ms` and `render_ms`, the wall-clock
milliseconds of the stages that read the vault and the chunk index, embed and match the phrases,
cluster, assemble the explore half, ride-alongs and recent chunks, and prepare the payload for
printing. Without the flag the payload MUST NOT contain `timings`, and the flag MUST NOT change
anything else in it.

Rationale: a measurement-only diagnostic (#691), so that query speed can be decomposed before it is
optimized. On a vault with 41,367 chunk-index files (364 MB) the scan dominated at about 67% of the
binary's wall time (4,787 ms median against embed 2,009, cluster 212, nominate 60 and render 0
ms; `dev/eval/LEDGER.md#query-inflight-split`). The stage key `nominate` remains from the era of tag
nomination and now measures explore sampling and ride-along assembly.

#### 792a7ze1 UAT Scenarios

```gherkin
Scenario: 792a7ze1a Timings are reported on request
  Given a vault with notes
  When the user runs engram query --timings with a phrase
  Then the payload has a timings block with five keys: scan_ms, embed_ms, cluster_ms, nominate_ms and render_ms

Scenario: 792a7ze1b Timings are absent by default
  Given the same vault
  When the user runs engram query with a phrase and no --timings
  Then the payload has no timings block
```

### 792a7zf Searching chunks alone

Requirement: 792a7zf `engram query-chunks --phrase <p>` MUST score every chunk in the chunk index by
its best cosine over the phrases, without recency scaling, and print the best `--limit` (20 by
default) as `items` with `source`, `anchor`, `score` and `content`, highest score first, under
`version: 1` and `space: chunks`. When it returns at least six chunks it MUST also print `clusters`,
groups of the returned chunks found by the clustering of 792a7g, each with an `id`, a `size` and its
members' `source`, `anchor` and `score`. It MUST end with a `budget` of `total_chunks` and
`returned`. It MUST fail with `query chunks: at least one --phrase is required` without a phrase. An
empty or absent chunk index MUST give an empty payload without embedding the phrase.

#### 792a7zf1 UAT Scenarios

```gherkin
Scenario: 792a7zf1a Chunks are ranked by cosine
  Given a chunk index holding a baking chunk and a networking chunk
  When the user runs engram query-chunks --phrase "sourdough"
  Then items lists the baking chunk first, each with source, anchor, score and content, and the budget says total_chunks: 2 and returned: 2

Scenario: 792a7zf1b The limit is honoured
  Given a chunk index of five chunks
  When the user runs engram query-chunks --phrase p --limit 2
  Then two items are returned and the budget says returned: 2

Scenario: 792a7zf1c Many chunks are clustered
  Given a chunk index of twelve chunks about two topics
  When the user runs engram query-chunks with a phrase
  Then the payload has clusters, each with an id, a size and members

Scenario: 792a7zf1d A phrase is required
  Given any chunk index
  When the user runs engram query-chunks with no phrase
  Then it fails with "query chunks: at least one --phrase is required"

Scenario: 792a7zf1e An empty index is an empty payload
  Given a chunk index directory that does not exist
  When the user runs engram query-chunks --phrase p
  Then the payload has no items, and the budget says total_chunks: 0
```

### 792a7zg CRLF notes

Requirement: 792a7zg `engram query` MUST read a note whose lines end in `\r\n` as its LF form (each
`\r\n` read as `\n`), in its project filter, its recency date, the tags, supersessions and triggers it
reads from frontmatter, the note's kind label, and the preview of its red flags, and MUST NOT
write a note only to convert it.

#### 792a7zg1 UAT Scenarios

```gherkin
Scenario: 792a7zg1a The project filter reads a CRLF note
  Given a note with CRLF line endings and project: alpha
  When the user runs engram query --project alpha with a phrase that matches it
  Then the note is returned

Scenario: 792a7zg1b A CRLF runbook's trigger matches
  Given a runbook whose lines end in \r\n and whose triggers include "curate"
  When the user runs engram query --text "curate"
  Then it is returned as a trigger hit with kind runbook

Scenario: 792a7zg1c A CRLF note is not rewritten
  Given a note with CRLF line endings
  When the user runs engram query with a phrase that matches it
  Then the note's file is byte-identical afterwards

Scenario: 792a7zg1d The creation date of a CRLF note is read
  Given a CRLF note whose created line is the bare date 120 days ago and whose sidecar has no last_used
  When the user runs engram query with a phrase that matches it
  Then its score is scaled by that age, as for the same note with LF line endings

Scenario: 792a7zg1e A CRLF runbook's red flags are previewed
  Given a runbook whose lines end in \r\n and whose red flags take more than 1200 bytes
  When the user runs engram query with a phrase that matches it
  Then its content holds the omission marker and the newest entries, as for the same note with LF line endings

Scenario: 792a7zg1f A CRLF note keeps its kind
  Given a fact note whose lines end in \r\n
  When the user runs engram query with a phrase that matches it
  Then its item has kind fact

Scenario: 792a7zg1g The lists in a CRLF note's frontmatter are read
  Given a CRLF note tagged vocab/cooking, and a CRLF note that supersedes another
  When the user runs engram query with phrases that match the term's members, and the superseded note
  Then the tagged note can be an explore note, and the superseding note rides along
```

### 792a7zh A query changes nothing

Requirement: 792a7zh `engram query` MUST NOT write any note, sidecar or chunk-index file, and MUST NOT
mark a note as used: it MUST NOT change a sidecar's `last_used`, and its payload MUST NOT carry an
`activated` field. A note is marked as used only by the agent, through `engram activate` or `engram amend --activate`.

Rationale: activation is agent-driven. A note that was returned but not used goes stale and fades in
rank, which is the mechanism by which unhelpful memory loses its place; if every returned note were
bumped, nothing would ever fade.

#### 792a7zh1 UAT Scenarios

```gherkin
Scenario: 792a7zh1a A query leaves the vault as it was
  Given a vault with notes and sidecars
  When the user runs engram query with a phrase that matches them
  Then every note, sidecar and chunk-index file is byte-identical afterwards, and the payload has no activated field
```
