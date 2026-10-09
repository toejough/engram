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
a missing or unknown `--position`, a `continuation` or `sibling` without `--target`, and a `sibling`
of a top-level ID, which must use `--position top` instead.

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
