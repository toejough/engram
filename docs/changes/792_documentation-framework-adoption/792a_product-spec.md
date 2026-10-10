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

## 792a8 Added: vault-counting

Canonical target: `docs/specs/product/vault-counting.md` (title `Vault Counting - Product Spec`).

`engram count` answers "how many?" over the whole vault: how many notes carry each value of a
frontmatter attribute, and how many notes link to a given note. It reads only the Markdown notes at
the vault's root and changes nothing. It runs against the vault located by 792a2a, and a vault that
does not exist is created first (792a2e). `engram query` answers a different question, "what is
relevant to this phrase", and returns a ranked, bounded sample of the vault (792a7). The `route`
skill uses `count` to audit the evidence tallies it reports.

### 792a8a Count notes by attribute value

Requirement: 792a8a `engram count --group-by <attr>` MUST print, for each distinct value of the
frontmatter attribute `<attr>` among the notes at the vault root, one line `<value>`, a tab and the
number of notes that carry the value, ordered by that number descending and then by value
ascending. A list attribute MUST contribute each of its distinct elements once, and a note that
repeats an element MUST count once for it.

Rationale: a question such as "how many notes carry this tag" needs an exact count over the whole
vault. A query returns a similarity-ranked, truncated sample with no sign that it is not the full
set, so counting does not ride on it.

#### 792a8a1 UAT Scenarios

```gherkin
Scenario: 792a8a1a A scalar attribute is counted per value
  Given a vault of 349 feedback notes, 192 fact notes and 8 qa-answer notes
  When the user runs engram count --group-by type
  Then stdout is "feedback", a tab and 349, then "fact", a tab and 192, then "qa-answer", a tab and 8, then the line "total: 549"

Scenario: 792a8a1b A list attribute counts each distinct element once per note
  Given the only note with vocab values is one whose frontmatter has vocab: [x, y, z]
  When the user runs engram count --group-by vocab
  Then each of x, y and z is printed with the count 1, followed by "total: 1"

Scenario: 792a8a1c A repeated element counts once
  Given a note whose frontmatter has vocab: [dup, dup, other]
  When the user runs engram count --group-by vocab
  Then dup and other are each printed with the count 1, not dup with 2, followed by "total: 1"

Scenario: 792a8a1d Equal counts are ordered by value
  Given two notes with type: fact and two notes with type: feedback
  When the user runs engram count --group-by type
  Then fact is printed before feedback

Scenario: 792a8a1e Only notes at the vault root are counted
  Given a vault with one note at its root and another note in a subdirectory, both with type: fact
  When the user runs engram count --group-by type
  Then fact is printed with the count 1
```

### 792a8b Absent attributes and the total

Requirement: 792a8b `engram count --group-by <attr>` MUST print, after the value lines and before
the total, the line `(<attr> absent): <N>` when `<N>` counted notes have no `<attr>` key. It MUST
end with the line `total: <T>`, where `<T>` is the number of notes counted, not the number of
values. A note whose `<attr>` is present but empty MUST count toward the total and appear in no
value line and not in the absent line. When no note is counted it MUST print nothing, not even a
total.

#### 792a8b1 UAT Scenarios

```gherkin
Scenario: 792a8b1a Notes without the attribute are reported
  Given one note with vocab: [x] and one note with no vocab key
  When the user runs engram count --group-by vocab
  Then stdout is "x", a tab and 1, then "(vocab absent): 1", then "total: 2"

Scenario: 792a8b1b No note has the attribute
  Given 557 notes and none has the attribute attr
  When the user runs engram count --group-by attr
  Then stdout is "(attr absent): 557" then "total: 557", with no value line

Scenario: 792a8b1c An empty vault prints nothing
  Given a vault with no notes
  When the user runs engram count --group-by type
  Then nothing is printed to stdout or stderr

Scenario: 792a8b1d An empty attribute counts only toward the total
  Given a note whose frontmatter has the key foo with no value
  When the user runs engram count --group-by foo
  Then stdout is "total: 1"
```

### 792a8c Filter the counted notes

Requirement: 792a8c `engram count --group-by <attr>` MUST count only the notes that satisfy every
`--filter <attr>=<value>` it is given; the flag MAY be repeated. A note satisfies a filter when its
attribute equals the value (a scalar) or lists the value among its elements (a list). The filter
text is split at its first `=`. The command MUST refuse a filter that has no `=`, failing with
`count: --filter must be attr=value: "<filter>"` and a non-zero exit.

#### 792a8c1 UAT Scenarios

```gherkin
Scenario: 792a8c1a A scalar filter matches by equality
  Given notes of type feedback and notes of type fact
  When the user runs engram count --group-by type --filter type=feedback
  Then only the feedback notes are counted, and no other type is printed

Scenario: 792a8c1b A list filter matches by membership
  Given a note with tags: [tier/haiku, vocab/x]
  When the user runs engram count --group-by tags --filter tags=tier/haiku
  Then the note is counted

Scenario: 792a8c1c Every filter must match
  Given a fact note with outcome: pass, a fact note with outcome: fail, and a feedback note with outcome: pass
  When the user runs engram count --group-by outcome --filter type=fact --filter outcome=pass
  Then only the first note is counted, and the output is "pass", a tab and 1, then "total: 1"

Scenario: 792a8c1d A note without the filter's attribute is excluded
  Given a note that has no outcome key
  When the user runs engram count --group-by type --filter outcome=pass
  Then the note is not counted

Scenario: 792a8c1e No note matches
  Given a vault with no feedback note
  When the user runs engram count --group-by type --filter type=feedback
  Then nothing is printed

Scenario: 792a8c1f A filter without = is refused
  Given a vault
  When the user runs engram count --group-by type --filter bad
  Then it fails with an error saying --filter must be attr=value and quoting "bad", and prints nothing to stdout
```

### 792a8d Notes that cannot be counted

Requirement: 792a8d `engram count --group-by` MUST skip a note that it cannot read, a note with no
frontmatter, and a note whose frontmatter is not valid YAML, counting it neither as a value, nor as
absent, nor in the total. It MUST count a note whose lines end in `\r\n` exactly as it counts the
same note with LF line endings.

Rationale: a note that cannot be parsed has no attribute to count, and one stray file must not
stop a count of the rest. Notes written on a CRLF platform are valid notes (792d69).

#### 792a8d1 UAT Scenarios

```gherkin
Scenario: 792a8d1a A note without usable frontmatter is skipped
  Given one well-formed note with type: feedback, one note with no frontmatter, and one note whose frontmatter is malformed YAML
  When the user runs engram count --group-by type
  Then stdout is "feedback", a tab and 1, then "total: 1"

Scenario: 792a8d1b An unreadable note is skipped
  Given a vault with a note with type: fact that the user cannot read, and a readable note with type: fact
  When the user runs engram count --group-by type
  Then fact is printed with the count 1, followed by "total: 1"

Scenario: 792a8d1c A CRLF note is counted
  Given a note with type: feedback whose lines end in \r\n
  When the user runs engram count --group-by type
  Then feedback is printed with the count 1, followed by "total: 1"
```

### 792a8e Count the notes that link to a note

Requirement: 792a8e `engram count --backlinks-of <basename>` MUST print `in-degree: <N>`, where
`<N>` is the number of other notes at the vault root whose text links to the note named
`<basename>`, followed by those notes' basenames one per line in ascending order. A note that links
more than once MUST count once. A note's link to itself, a link inside a fenced code block, and a
link to a note that is not at the vault root MUST NOT count. A link written with a trailing `.md`
MUST count as the link without it. `<basename>` is matched exactly as given. The command MUST fail
with a non-zero exit, naming the file, when any note at the vault root cannot be read.

Rationale: the wikilinks in a note's text are the vault's links, and the count agrees with the
note's backlinks panel in Obsidian. A link inside a fenced code block is not a link there, and a
link to a name with no note is not a backlink of any note.

#### 792a8e1 UAT Scenarios

```gherkin
Scenario: 792a8e1a A note with incoming links
  Given three notes that contain [[foo.alpha]]
  When the user runs engram count --backlinks-of foo.alpha
  Then stdout starts with "in-degree: 3", followed by the three linkers' basenames, one per line, in ascending order

Scenario: 792a8e1b A note with no incoming links
  Given no note links to unknown.note
  When the user runs engram count --backlinks-of unknown.note
  Then stdout is "in-degree: 0" and no linker lines

Scenario: 792a8e1c A basename that is not a note
  Given a vault in which the basename nosuchfile never appears
  When the user runs engram count --backlinks-of nosuchfile
  Then stdout is "in-degree: 0" and no linker lines

Scenario: 792a8e1d A repeated link counts once
  Given a note x that contains [[y]] twice
  When the user runs engram count --backlinks-of y
  Then stdout is "in-degree: 1" then "x"

Scenario: 792a8e1e A note's link to itself does not count
  Given a note x whose only link is [[x]]
  When the user runs engram count --backlinks-of x
  Then stdout is "in-degree: 0"

Scenario: 792a8e1f A link inside a fenced code block does not count
  Given a note x that contains [[z]] only inside a fenced code block
  When the user runs engram count --backlinks-of z
  Then stdout is "in-degree: 0"

Scenario: 792a8e1g A link ending in .md counts as the bare link
  Given a note x that contains [[y.md]]
  When the user runs engram count --backlinks-of y
  Then stdout is "in-degree: 1" then "x"

Scenario: 792a8e1h A link to a missing note counts for no note
  Given a note x that contains [[nowhere]], and no note named nowhere
  When the user runs engram count --backlinks-of nowhere
  Then stdout is "in-degree: 0"

Scenario: 792a8e1i Only notes at the vault root link
  Given a note in a subdirectory that contains [[x]], and a note x at the vault root
  When the user runs engram count --backlinks-of x
  Then stdout is "in-degree: 0"

Scenario: 792a8e1j An unreadable note fails the report
  Given a vault with a note the user cannot read
  When the user runs engram count --backlinks-of x
  Then it fails with an error naming the unreadable file, and prints nothing to stdout
```

### 792a8f One mode per invocation

Requirement: 792a8f `engram count` MUST fail with a non-zero exit, printing nothing to stdout, when
it is given both `--group-by` and `--backlinks-of` (`count: --group-by and --backlinks-of are
mutually exclusive`), and when it is given neither (`count: specify --group-by <attr> or
--backlinks-of <basename>`).

Rationale: the two modes measure different things (792a8g), so one invocation does not mix them.

#### 792a8f1 UAT Scenarios

```gherkin
Scenario: 792a8f1a Both flags are refused
  Given a vault
  When the user runs engram count --group-by type --backlinks-of note.name
  Then it fails with "count: --group-by and --backlinks-of are mutually exclusive"

Scenario: 792a8f1b Neither flag is refused
  Given a vault
  When the user runs engram count with no --group-by and no --backlinks-of
  Then it fails with "count: specify --group-by <attr> or --backlinks-of <basename>"

Scenario: 792a8f1c A filter alone is not a mode
  Given a vault
  When the user runs engram count --filter type=fact
  Then it fails with "count: specify --group-by <attr> or --backlinks-of <basename>"
```

### 792a8g The two modes measure different things

Requirement: 792a8g `engram count --backlinks-of` MUST count every note that links to the named
note, whether or not that note carries a given attribute value, and `engram count --group-by` MUST
count only the notes that carry the value, whether or not they link. For a note named for an
attribute value, the in-degree therefore equals the `--group-by` count of that value plus the
number of linking notes that do not carry the value.

Rationale: each mode can be checked by hand against its own Obsidian view (`--group-by` against a
property or tag filter, `--backlinks-of` against the backlinks panel), and the two are not equal to
each other. An index or map-of-content page links every note on a topic without carrying the topic
in its own frontmatter, so it counts toward `--backlinks-of` and not toward `--group-by`. Reporting
the two as "the same count" would make the modes look redundant when they are complementary
(ADR-0018). The relationship was checked on the live vault in July 2026: 33
members of `retrieval-design` and an in-degree of 34, the extra linker being a generated index page
that was retired later.

#### 792a8g1 UAT Scenarios

```gherkin
Scenario: 792a8g1a The modes agree when every linker is a member
  Given notes n1 and n2 with foo: [alpha] that each contain [[foo.alpha]], and the note foo.alpha
  When the user runs engram count --group-by foo and engram count --backlinks-of foo.alpha
  Then --group-by prints alpha with the count 2 and --backlinks-of prints "in-degree: 2"

Scenario: 792a8g1b The in-degree exceeds the count by the non-member linkers
  Given n1 with foo: [alpha] that contains [[foo.alpha]], idx with no foo attribute that contains [[foo.alpha]], and the note foo.alpha
  When the user runs engram count --group-by foo and engram count --backlinks-of foo.alpha
  Then --group-by prints alpha with the count 1 and --backlinks-of prints "in-degree: 2" then n1 and idx
```

### 792a8h A count changes nothing and is exact

Requirement: 792a8h `engram count` MUST NOT create, modify or delete any note or sidecar in the
vault, and MUST NOT need a note's sidecar or an embedding to count it. It MUST count every note at
the vault root, not a ranked or truncated subset.

Rationale: counting is a read-only aggregation, deliberately off the recall and similarity path. It
never embeds, scores cosine similarity or clusters (ADR-0018).

#### 792a8h1 UAT Scenarios

```gherkin
Scenario: 792a8h1a A count leaves the vault as it was
  Given an existing vault with notes
  When the user runs engram count --group-by type and then engram count --backlinks-of x
  Then every file in the vault is byte-identical afterwards, and no file was created

Scenario: 792a8h1b Every note is counted without sidecars
  Given a vault of 600 notes with type: fact and no sidecars
  When the user runs engram count --group-by type
  Then stdout is "fact", a tab and 600, then "total: 600"
```

## 792a9 Added: vault-health

Canonical target: `docs/specs/product/vault-health.md` (title `Vault Health - Product Spec`).

A vault is a directory of notes and a sidecar beside each note holding its embedding (792a3n). Two
commands keep it healthy. `engram check` reads the vault and reports where it breaks its
invariants: links that cannot resolve, notes recall cannot match, and sidecars that cannot be
read. `engram embed status` and `engram embed apply` count and repair the sidecars. All three read
only the Markdown notes at the vault's root, run against the vault located by 792a2a, and create a
vault that does not exist first (792a2e). The model change that makes an embed run necessary is the
embedding-sidecar's 792b10c7.

### 792a9a The check report

Requirement: 792a9a `engram check` MUST print one line for each invariant it checks, beginning
`PASS`, `WARN` or `FAIL` and naming the invariant, and under a violated invariant MUST list the
offending items, one per line, at most ten, followed by `… and <N> more` when there are more. It MUST
exit non-zero, with `check: FAIL-class invariant violations found` on stderr, when any line begins
`FAIL`, and MUST exit zero when the worst line is `WARN`. It MUST fail with `check: scan: <error>`
and a non-zero exit, printing no report, when a note at the vault root cannot be read. It MUST check
a note whose text is not valid UTF-8 like any other.

Rationale: a vault check is only useful if a violation cannot go unnoticed, so a correctness
violation is a `FAIL` and a non-zero exit, while a corpus-health finding is a `WARN` that reports
without blocking. The checker is Go code over bytes, not a `grep`: a `grep` that stops at an invalid
multibyte sequence silently skipped notes in an early census and produced a false count.

#### 792a9a1 UAT Scenarios

```gherkin
Scenario: 792a9a1a A clean vault passes
  Given a vault whose links resolve, whose fact notes all have a situation, and whose sidecars are current
  When the user runs engram check
  Then stdout has a PASS line for G0 graph-resolution, M5 situation-presence and S1 sidecar-schema, and the exit status is zero

Scenario: 792a9a1b A FAIL line fails the command
  Given a vault with a note whose link is a bare ID that should resolve
  When the user runs engram check
  Then a line beginning FAIL names G0 graph-resolution, stderr has "check: FAIL-class invariant violations found", and the exit status is non-zero

Scenario: 792a9a1c A WARN line does not fail the command
  Given a vault whose only finding is a link to a note that does not exist
  When the user runs engram check
  Then a line beginning WARN names G3 dangling, and the exit status is zero

Scenario: 792a9a1d Offenders are capped at ten
  Given a vault in which twelve links target no note
  When the user runs engram check
  Then ten of them are listed, followed by the line "… and 2 more"

Scenario: 792a9a1e An unreadable note stops the check
  Given a vault with a note the user cannot read
  When the user runs engram check
  Then it fails with an error beginning "check: scan:" and naming the file, and prints no PASS, WARN or FAIL line

Scenario: 792a9a1f An empty vault passes
  Given an existing vault with no notes
  When the user runs engram check
  Then every line begins PASS, the exit status is zero, and stderr is empty

Scenario: 792a9a1g A note that is not valid UTF-8 is still checked
  Given a note with invalid UTF-8 bytes in its body that links [[ghost]], and no such note
  When the user runs engram check
  Then the dangling link is reported, as it would be without the invalid bytes
```

### 792a9b Links that should resolve

Requirement: 792a9b `engram check` MUST report `FAIL  G0 graph-resolution: <N> bare-id links that
should resolve`, listing each as `<source> → [[<target>]]`, when a note's link target is a bare
Luhmann ID, digits followed by letters or digits, that is the leading ID of an existing note, and
MUST otherwise report `PASS  G0 graph-resolution: every authored link resolves by form`. A link to a
note's full basename, with or without a trailing `.md`, MUST count as resolving.

Rationale: a link is written in one form and resolved in another when `[[105]]` is written for a
note `105.2026-05-30.foo`. Neither engram nor Obsidian resolves a prefix: both resolve only a full
basename (or alias). Such a link is a lost edge, and in an early census of a live vault it dropped
155 of 183 links, leaving 138 of 171 notes without a link in or out. The invariant behind it is that
for every write and read of the same datum the read key must be a function of, or equal to, the write
key, and a mismatch must be loud, not silent. A machine-written supersession link used to carry a
`.md` suffix and so never resolved (#717). Machine writers now write the extension-less form, and the
reader accepts the suffixed form already in the vault (792b13c3).

#### 792a9b1 UAT Scenarios

```gherkin
Scenario: 792a9b1a A bare-ID link fails
  Given a note 1.2026-05-30.foo that links [[2]], and a note 2.2026-05-30.bar
  When the user runs engram check
  Then stdout has "FAIL  G0 graph-resolution: 1 bare-id links that should resolve" and the line "1.2026-05-30.foo → [[2]]"

Scenario: 792a9b1b A basename link passes
  Given a note A that links [[B]], and a note B
  When the user runs engram check
  Then stdout has "PASS  G0 graph-resolution: every authored link resolves by form"

Scenario: 792a9b1c A legacy suffixed link resolves
  Given a note body containing "Supersedes: [[B.md]]" and a note B
  When the user runs engram check
  Then the link is not reported, neither under G0 nor as dangling under G3
```

### 792a9c Links to nothing

Requirement: 792a9c `engram check` MUST warn, without failing, with `WARN  G3 dangling: <N>
authored links target no note`, listing each as `<source> → [[<target>]]`, for every distinct link
in a note whose target, after a trailing `.md` is dropped, is neither a note's basename nor a G0
bare ID of an existing note. It MUST NOT report a note's link to itself, and MUST NOT report a link
inside a fenced code block. It MUST NOT print the line when there are none.

Rationale: the graph drops a dangling link without a sound, so a dangling link is a lost intended
edge. A deleted note's name, or a name such as `vocab.X`, is a legitimate finding, but a note may
still stand alone, so this is corpus health and not correctness.

#### 792a9c1 UAT Scenarios

```gherkin
Scenario: 792a9c1a A link to a missing note warns
  Given a fact note that links [[999]] and no note has the ID 999
  When the user runs engram check
  Then stdout has a WARN G3 dangling line listing the link, G0 still passes, and the exit status is zero

Scenario: 792a9c1b A missing note is reported once however it is spelled
  Given a note that links [[vocab.X]] and [[vocab.X.md]], and no such note
  When the user runs engram check
  Then stdout has "WARN  G3 dangling: 1 authored links target no note"

Scenario: 792a9c1c A note's link to itself is not reported
  Given a note that links its own basename
  When the user runs engram check
  Then no G3 line is printed

Scenario: 792a9c1d A link inside a fenced code block is not reported
  Given a note whose only link to a missing note is inside a fenced code block
  When the user runs engram check
  Then no G3 line is printed

Scenario: 792a9c1e A vault with no dangling link prints no G3 line
  Given a vault whose links all resolve
  When the user runs engram check
  Then stdout has no line mentioning G3
```

### 792a9d Notes recall can match

Requirement: 792a9d `engram check` MUST report `FAIL  M5 situation-presence: <N> note(s) missing a
situation`, listing the notes, when a note of type `fact` or `feedback` has a `situation:` that is
absent, empty or only whitespace, and MUST otherwise report `PASS  M5 situation-presence: every
fact/feedback names a situation`. It MUST NOT judge a note of any other type, a note with no
frontmatter, or a note whose frontmatter is not valid YAML. It MUST judge a note whose lines end in
`\r\n` as it judges the same note with LF line endings.

Rationale: recall matches a query against each note's situation, and the embedding is shaped
around it (792a7d1h). Every write path refuses a blank one (792a3f), so this check finds a note that
reached the vault another way.

#### 792a9d1 UAT Scenarios

```gherkin
Scenario: 792a9d1a A fact without a situation fails
  Given a fact note whose frontmatter has no situation key
  When the user runs engram check
  Then stdout has "FAIL  M5 situation-presence: 1 note(s) missing a situation" naming the note, and the exit status is non-zero

Scenario: 792a9d1b A blank situation fails
  Given a feedback note whose situation is only spaces
  When the user runs engram check
  Then M5 fails naming the note

Scenario: 792a9d1c Every situation present passes
  Given fact and feedback notes that each name a situation
  When the user runs engram check
  Then stdout has "PASS  M5 situation-presence: every fact/feedback names a situation"

Scenario: 792a9d1d Other note types are not judged
  Given a note of type moc and a note of type runbook, neither with a situation
  When the user runs engram check
  Then M5 passes

Scenario: 792a9d1e A note without usable frontmatter is not judged
  Given a note with no frontmatter and a note whose frontmatter is malformed YAML
  When the user runs engram check
  Then M5 passes

Scenario: 792a9d1f A CRLF note is judged
  Given a fact note whose lines end in \r\n and whose situation is empty
  When the user runs engram check
  Then M5 fails naming the note
```

### 792a9e Sidecars that can be read

Requirement: 792a9e `engram check` MUST report `FAIL  S1 sidecar-schema: <N> sidecar(s) on an
old/invalid schema (run `engram embed apply --force`)`, listing the notes, when a note's sidecar
cannot be decoded under the current schema, whether it is the old single-vector schema, malformed
JSON, or a vector of the wrong length. It MUST report `WARN  S1 sidecar-schema: <N> note(s) missing
a sidecar (run `engram embed apply`)`, without failing, for each note that has no sidecar. It MUST
report `PASS  S1 sidecar-schema: every sidecar parses under the current schema` when no sidecar is
invalid.

Rationale: a sidecar that cannot be decoded silently drops its note from retrieval. A missing sidecar
is a gap that `engram embed apply` fills, and it is no corruption.

#### 792a9e1 UAT Scenarios

```gherkin
Scenario: 792a9e1a An old-schema sidecar fails
  Given a fact note whose sidecar has the old single-vector schema
  When the user runs engram check
  Then stdout has a FAIL line for S1 sidecar-schema naming the note and `engram embed apply --force`, and the exit status is non-zero

Scenario: 792a9e1b A malformed sidecar fails
  Given a note whose sidecar is not valid JSON
  When the user runs engram check
  Then S1 fails naming the note

Scenario: 792a9e1c A current sidecar passes
  Given a note with a current sidecar
  When the user runs engram check
  Then stdout has "PASS  S1 sidecar-schema: every sidecar parses under the current schema"

Scenario: 792a9e1d A missing sidecar warns
  Given notes with no sidecar and no invalid sidecar
  When the user runs engram check
  Then stdout has a WARN line for S1 sidecar-schema counting the notes and naming `engram embed apply`, and the exit status is zero
```

### 792a9f A check changes nothing

Requirement: 792a9f `engram check` MUST NOT create, modify or delete any note or sidecar.

#### 792a9f1 UAT Scenarios

```gherkin
Scenario: 792a9f1a A check leaves the vault as it was
  Given an existing vault with notes, some with sidecars, some violating invariants
  When the user runs engram check
  Then every file in the vault is byte-identical afterwards, and no file was created
```

### 792a9g The embedding census

Requirement: 792a9g `engram embed status` MUST print six lines: `total:`, `with-embeddings:`,
`without:`, `stale:`, `incompatible:` and `broken:`, each with the number of notes at the vault root in
that state, so that the five counts add up to the total. A note is with-embeddings when its sidecar
matches its content and the current model, without when it has no sidecar, stale when its content
changed after its sidecar was written, incompatible when its sidecar was made by another model or an
older schema, and broken when the note or its sidecar cannot be read or decoded. It MUST fail with
`embed status: scan: <error>` and a non-zero exit when a note at the vault root cannot be read.

#### 792a9g1 UAT Scenarios

```gherkin
Scenario: 792a9g1a Notes without sidecars are counted as without
  Given a vault of two notes and no sidecar
  When the user runs engram embed status
  Then stdout has "total:           2", "with-embeddings: 0" and "without:         2"

Scenario: 792a9g1b Every state is counted
  Given a vault of four notes: one embedded and current, one whose body changed after embedding, one whose sidecar names another model, and one whose sidecar is malformed
  When the user runs engram embed status
  Then total is 4, and with-embeddings, stale, incompatible and broken are each 1

Scenario: 792a9g1c An unreadable note stops the census
  Given a vault with a note the user cannot read
  When the user runs engram embed status
  Then it fails with an error beginning "embed status: scan:" and naming the file
```

### 792a9h Which notes get embedded

Requirement: 792a9h `engram embed apply` MUST embed, with no mode flag or with `--missing`, the notes
that have no sidecar; with `--stale`, the notes whose content changed after their sidecar; with
`--force`, the notes whose sidecar was made by another model or an older schema; and with `--all`,
every note. Mode flags MUST combine, each adding its notes, and `--all` MUST select every note whatever
else is given. A note whose sidecar is broken MUST be embedded under every selection.

Rationale: a re-embed under a new model also picks up any content change, so `--force` is the explicit
step that replaces another model's sidecars; the default and `--stale` leave them alone, which keeps a
routine run from silently rewriting a vault under a model it did not choose.

#### 792a9h1 UAT Scenarios

```gherkin
Scenario: 792a9h1a The default embeds only notes without a sidecar
  Given one note without a sidecar and one with a current sidecar
  When the user runs engram embed apply
  Then only the note without a sidecar is embedded

Scenario: 792a9h1b --stale embeds changed notes and not another model's sidecars
  Given a note whose body changed after its sidecar and a note whose sidecar names another model
  When the user runs engram embed apply --stale
  Then the changed note is re-embedded and the other model's sidecar is unchanged

Scenario: 792a9h1c --force replaces another model's sidecar
  Given a note whose sidecar names another model
  When the user runs engram embed apply --force
  Then its sidecar is rewritten under the current model

Scenario: 792a9h1d --all embeds every note
  Given a note with a current sidecar
  When the user runs engram embed apply --all
  Then it is embedded again

Scenario: 792a9h1e Mode flags combine
  Given a stale note, an incompatible note and a missing note
  When the user runs engram embed apply --stale --force
  Then the stale and the incompatible notes are embedded and the missing one is not

Scenario: 792a9h1f A broken sidecar is always replaced
  Given a note whose sidecar is malformed
  When the user runs engram embed apply, then with --stale, then with --force
  Then the note is embedded each time
```

### 792a9i Embedding the selected notes

Requirement: 792a9i `engram embed apply` MUST, for each selected note in ascending basename order,
write the note's sidecar beside it and print `embedded  <note>.md (<state>)`, where `<state>` is the
state the note was in. With `--dry-run` it MUST instead print `would-embed <note>.md (<state>)` and
write nothing. It MUST report a note it cannot read as `skip      <note>.md: read error: <error>`, an
embedding that fails as `fail      <note>.md: embed: <error>`, and a sidecar it cannot write as
`fail      <note>.md: write: <error>`, and MUST go on to the next note and exit zero. It MUST NOT
modify any note, and MUST fail with `embed apply: scan: <error>` and a non-zero exit when a note at the
vault root cannot be read.

Rationale: a sidecar is a derived file, not part of the note's content. It is recomputed from the
current note each time, so its content hash stays honest, and it is never stored in or versioned with
the note file.

#### 792a9i1 UAT Scenarios

```gherkin
Scenario: 792a9i1a A missing sidecar is written
  Given a note 1.foo.md with no sidecar
  When the user runs engram embed apply
  Then stdout has "embedded  1.foo.md (missing)" and the sidecar 1.foo.vec.json exists

Scenario: 792a9i1b A dry run writes nothing
  Given a note 1.foo.md with no sidecar
  When the user runs engram embed apply --dry-run
  Then stdout has "would-embed 1.foo.md (missing)" and no sidecar exists

Scenario: 792a9i1c Only sidecars are written
  Given notes needing embedding
  When the user runs engram embed apply --all
  Then every note file is byte-identical afterwards, none of them contains a vector or a model ID, and the only files created are sidecars

Scenario: 792a9i1d A failed write is reported and the run goes on
  Given two notes, where a directory stands where the first note's sidecar would be
  When the user runs engram embed apply
  Then stdout has a line beginning "fail      " for the first note and ending with the write error, the second note is embedded, and the exit status is zero

Scenario: 792a9i1e Notes are embedded in order
  Given three notes without sidecars
  When the user runs engram embed apply
  Then the "embedded" lines are in ascending basename order

Scenario: 792a9i1f An unreadable note stops the run
  Given a vault with a note the user cannot read
  When the user runs engram embed apply
  Then it fails with an error beginning "embed apply: scan:" and nothing is embedded
```

### 792a9j CRLF notes embed as LF

Requirement: 792a9j `engram embed status` and `engram embed apply` MUST treat a note whose lines end in
`\r\n` as the same note with LF line endings, so that its sidecar equals the one its LF form gets and
is not reported stale for its line endings.

#### 792a9j1 UAT Scenarios

```gherkin
Scenario: 792a9j1a A CRLF note is not stale
  Given a note embedded with LF line endings and then converted to CRLF
  When the user runs engram embed status
  Then the note is counted with-embeddings, and stale is 0

Scenario: 792a9j1b A CRLF note gets its LF form's sidecar
  Given a note with LF line endings and the same note with CRLF line endings
  When the user runs engram embed apply --all on each
  Then the two sidecars are byte-identical
```

## 792a10 Added: vault-migration

Canonical target: `docs/specs/product/vault-migration.md` (title `Vault Migration - Product Spec`).

Some vaults need a one-off change that no ordinary write makes. A vault that holds only top-level
notes has never branched its Luhmann IDs, and a note written before notes carried their origin has no
`repo:`, `user:` and `vault:`. `engram update` notices both and names the opt-in remedy, and the remedy
runs only when asked: `engram update --reparent-luhmann` re-parents top-level notes and
`engram update --backfill-identity` stamps the missing origin. Both act on the vault located by 792a2a.
The notices and the two flags belong to `engram update`, whose binary refresh and harness sync are
install-and-update's.

### 792a10a The flat-vault notice

Requirement: 792a10a `engram update` MUST print a one-line notice naming `engram update
--reparent-luhmann` and the learn skill's batch mode when the vault holds at least one note and the
Luhmann ID of every note at its root is top-level, one run of digits with no letter or digit branch.
It MUST NOT print the notice when any note has a deeper ID, when the vault has no note, or when the
vault cannot be read, and MUST NOT modify any note on account of it.

Rationale: a notice that only points at a mechanism affecting future notes, the learn skill's
disposition step for each capture, does nothing for a vault that has been flat since the skill stopped
placing notes. The notice gives it a real remedy to point at. An empty vault has nothing to
re-evaluate, and an ID that does not parse counts as not top-level, so a malformed vault is never
called flat.

#### 792a10a1 UAT Scenarios

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
  Then the notice names both engram update --reparent-luhmann and the learn skill's batch mode
```

### 792a10b Re-parenting only on request

Requirement: 792a10b `engram update` MUST NOT rename a note or rewrite a link unless it is given
`--reparent-luhmann`. With the flag it MUST run only the re-parenting and MUST NOT refresh the binary
or the harness skills.

#### 792a10b1 UAT Scenarios

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

### 792a10c Candidates to judge

Requirement: 792a10c `engram update --reparent-luhmann` without `--answers` MUST print a JSON payload
and write no file. For each top-level note that has a sidecar, the payload MUST list up to three other
top-level notes with a sidecar whose body vector has cosine similarity of at least 0.75 with the note's,
nearest first, as a candidate of the note, its target, a body excerpt of each of at most 200 bytes, and
the similarity. It MUST also carry a `fingerprint` of the notes it considered, a `next_command` field
naming the literal follow-up command `engram update --reparent-luhmann --answers <path-to-answers-file>`,
and an `instruction`. The instruction MUST name the learn skill's batch mode as the procedure that
holds the continuation, sibling and top disposition test, state the answers-file shape, require exactly
one entry per distinct candidate note, say that the finished answers file goes back to the user or the
orchestrating agent to preview with `--dry-run` and then apply, and not tell the answering pass to run
apply. When there are no candidates it MUST print `update --reparent-luhmann: no candidates found`
instead.

Rationale: whether one note is a sub-point of another is a judgment of content that an embedding cannot
make, so the binary proposes pairs and an agent decides. This is the boundary drawn for per-capture
disposition, and the flow follows `vocab refit`'s derive, names, apply.

#### 792a10c1 UAT Scenarios

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
  Then the instruction names the learn skill's batch mode, requires one entry per distinct candidate note, names --dry-run as the preview before applying, and does not tell the answering pass to run apply

Scenario: 792a10c1d Only top-level notes with sidecars are considered
  Given a branched note and a top-level note without a sidecar, each similar to a top-level note with a sidecar
  When the user runs engram update --reparent-luhmann
  Then neither appears as a candidate or a target
```

### 792a10d Answers that are refused

Requirement: 792a10d `engram update --reparent-luhmann --answers <file>` MUST refuse, writing nothing
and exiting non-zero, an answers file that cannot be read, one that is not valid JSON (`cannot parse
--answers JSON`), one whose fingerprint differs from the vault's current candidates (the vault changed
since they were derived), and one that names a note the vault does not hold (`--answers references an
unknown note`). The stale refusal MUST say to re-run `engram update --reparent-luhmann`.

#### 792a10d1 UAT Scenarios

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
  Given an answers file naming a note ID that no top-level note has
  When the user applies it
  Then it fails with "--answers references an unknown note" and the vault is unchanged
```

### 792a10e Renaming and rewriting links

Requirement: 792a10e `engram update --reparent-luhmann --answers <file>` MUST give each note the
answers place as a `continuation` or `sibling` of a target the next ID that placement allocates, processing the
answers in ascending order of the notes' original IDs. It MUST rename the note's file and its sidecar to
the new ID, with the same date and slug, set its `luhmann:` to the new ID, record the old basename in its
`aliases:`, and rewrite every link to the old basename anywhere in the vault to the new one. That covers
`[[old]]` and `[[old.md]]` in a body or any frontmatter string, a `Supersedes:` line, and the `note:` of a
`supersedes:` entry. It MUST resolve a note that is renamed and also links another renamed note to that
note's new basename. A note answered `top` MUST NOT change, and a note that neither is renamed nor links a
renamed note MUST NOT be written. It MUST rebuild the sidecar of every note whose content it rewrote, so
that no note is stale afterwards.

#### 792a10e1 UAT Scenarios

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

### 792a10f The chunk index follows the rename

Requirement: 792a10f When an apply renamed at least one note, `engram update --reparent-luhmann` MUST,
in the same invocation, re-index the renamed notes and detach the chunk-manifest entries of their old
paths, so no `engram ingest --auto` or `engram prune` is needed. A failure of that step MUST NOT undo
the rename. It MUST report which stage failed, that the vault rename is intact and authoritative, and
that `engram ingest --auto` and `engram prune` remain as a manual fallback. When it renamed no note it
MUST NOT touch the chunk index.

#### 792a10f1 UAT Scenarios

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

### 792a10g What is left to judge

Requirement: 792a10g After a successful apply, `engram update --reparent-luhmann` MUST report either the
number of further candidates found in the vault's resulting state and the literal command to continue, or
that the vault is fully evaluated, with no further instruction. It MUST print `renamed <N> note(s)` first.

#### 792a10g1 UAT Scenarios

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

### 792a10h Previewing an apply

Requirement: 792a10h `engram update --reparent-luhmann --answers <file> --dry-run` MUST print the
old-to-new basename of each rename and the notes in which a link would be rewritten, and MUST NOT rename,
rewrite, re-index or prune anything. It MUST report the same refusals as an apply (792a10i). The flag
`--dry-run` without `--answers` MUST be refused as a usage error, since deriving candidates already
writes nothing.

Rationale: a preview is an inspection tool and never a required step. An apply is gated by its own
fingerprint.

#### 792a10h1 UAT Scenarios

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

### 792a10i Notes a rename will not touch

Requirement: 792a10i A rename and rewrite MUST check, before it renames or writes any file, every note it
would rename, and MUST refuse the whole invocation, naming each refused note and its reason, when the
rewritten note's frontmatter does not decode as a mapping, or its `luhmann:` is not the new ID. An
anchored `luhmann:` value that another key aliases is one such note. The invocation MUST then rename and
write no note. The same step serves `register-skills --adopt`.

When it writes a note whose lines end in `\r\n`, whether renamed or a referrer, it MUST convert every
`\r\n` to `\n` in the same single atomic write and rebuild that note's sidecar in the same invocation.
A CRLF note that it does not otherwise write MUST stay byte-identical. A converted note's exchange hash
MUST equal the hash of the same note authored with LF line endings, and a note whose frontmatter was
already LF MUST keep its exchange hash.

#### 792a10i1 UAT Scenarios

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

### 792a10j The identity-backfill notice

Requirement: 792a10j `engram update` MUST print a one-line notice naming `engram update
--backfill-identity` when the vault holds a fact or feedback note that the backfill could stamp now. A
note counts when it has no `user:` and either has no `vault:` (it predates identity) or user detection
resolves now, and a CRLF note counts as its LF form. It MUST NOT print the notice when no note counts,
including when the only note without `user:` has `vault:` while detection still resolves empty, and
MUST NOT print it on a run that has just backfilled.

#### 792a10j1 UAT Scenarios

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

### 792a10k Stamping the missing origin

Requirement: 792a10k `engram update --backfill-identity` MUST stamp `repo:`, `user:` and `vault:` onto each
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

#### 792a10k1 UAT Scenarios

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

### 792a10l What a backfill keeps

Requirement: 792a10l A backfill MUST make its edit on the note's parsed frontmatter: it sets `repo:`,
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

#### 792a10l1 UAT Scenarios

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

## 792a11 Added: install-and-update

Canonical target: `docs/specs/product/install-and-update.md` (title `Install and Update - Product Spec`).

`engram update` is how an operator brings a machine to the current engram. It installs the engram
binary from a source checkout, then deploys engram's skills (and, on request, its guidance documents)
into each coding harness it finds, Claude Code and Pi. Engram keeps its own copy of what it deploys in
a root of its own for each harness and links the harness's discovery paths to that copy, so that a skill
removed from the source disappears from every harness on the next update, and a file engram does not
own is never deleted. The notices that `engram update` prints about the vault (the vocabulary, the
Luhmann branching, the missing note origin, the uncommitted vault ID, pending offers and the parent
outbox) belong to the capabilities whose state they report, and registering skills as runbook notes
after the deploy belongs to skill-runbook-registration. This capability is the install, the deploy and
the report around them.

### 792a11a Installing from a local clone or a fresh clone

Requirement: 792a11a `engram update` MUST install the engram binary from the nearest enclosing
directory of the working directory whose `go.mod` names the module `github.com/toejough/engram`, and
MUST install it from a fresh shallow clone of the engram repository when no enclosing directory has a
`go.mod`. It MUST install into `$GOBIN`, else `$GOPATH/bin`, else `~/go/bin`, and MUST fail naming the
missing program when `go` (or, for a clone, `git`) is not installed. It MUST stop at the first `go.mod`
it meets, so a working directory inside another Go module installs from a fresh clone.

Rationale: a clone is built rather than fetched with `go install …@latest`, because the Go module proxy
serves repository blobs raw and the embedded model is a git-lfs object, so a proxy build embeds a
133-byte pointer and every embedding call then fails (#645). Running `engram update` from inside a clone
installs what that clone holds, which lets a contributor test a branch.

#### 792a11a1 UAT Scenarios

```gherkin
Scenario: 792a11a1a A local clone is installed in place
  Given the working directory is inside a checkout whose go.mod names github.com/toejough/engram
  When the user runs engram update
  Then the binary is built from that checkout, and no clone is made

Scenario: 792a11a1b Outside a checkout the repository is cloned
  Given no directory from the working directory up holds a go.mod
  When the user runs engram update
  Then the repository is shallow-cloned, the binary is built from the clone, and the report names a remote source

Scenario: 792a11a1c The install location follows the Go toolchain
  Given GOBIN is unset, GOPATH is /gp and the home directory is /home/u
  When the user runs engram update
  Then the report names /gp/bin/engram as the install location, and with GOBIN also set to /gb it names /gb/engram

Scenario: 792a11a1d The first go.mod ends the walk
  Given the working directory is inside a checkout of a different Go module
  When the user runs engram update
  Then the binary is built from a fresh clone, not from that checkout

Scenario: 792a11a1e Go is missing
  Given go is not installed
  When the user runs engram update from a local clone
  Then it fails with an error that the go binary is not on PATH

Scenario: 792a11a1f Git is missing
  Given git is not installed and the working directory is outside a checkout
  When the user runs engram update
  Then it fails with an error that the git binary is not on PATH
```

### 792a11b A clone without the model is refused

Requirement: 792a11b `engram update` MUST fail without installing when the cloned embedding model file
is a git-lfs pointer or holds less than one mebibyte, and its error MUST name git-lfs and tell the user
to install it and rerun `engram update`.

Rationale: git-lfs absent on the machine means the clone's smudge filter never ran, and building from
the pointer would produce a binary whose embedder fails at its first use.

#### 792a11b1 UAT Scenarios

```gherkin
Scenario: 792a11b1a A pointer file stops the install
  Given the clone's model.onnx is a git-lfs pointer file
  When the user runs engram update outside a checkout
  Then it fails with an error naming git-lfs and the install command, and no binary is installed

Scenario: 792a11b1b A real model installs
  Given the clone's model.onnx holds the real model
  When the user runs engram update outside a checkout
  Then the binary is built from the clone

Scenario: 792a11b1c An unreadable model is an error
  Given the clone has no model.onnx
  When the user runs engram update outside a checkout
  Then it fails with an error that the cloned model could not be read
```

### 792a11c A provable downgrade is refused

Requirement: 792a11c When it installs from a local clone and a binary is already installed at the install
location, `engram update` MUST read that binary's embedded source revision and MUST refuse to install,
exiting non-zero and saying the install would downgrade the installed binary, when git proves that the
clone's current revision is not a descendant of (and not equal to) the installed revision. It MUST
install anyway when given `--allow-downgrade`. It MUST install without refusing when no binary is
installed, when the installed binary carries no readable revision, or when git cannot tell either way
(the installed revision is unknown to the clone, as in a shallow or unrelated repository).

Rationale: running `engram update` from a stale clone or an old worktree silently replaced a newer
installed binary with older code (#715). Only a provable regression blocks, because an unprovable case
must never stop an update.

#### 792a11c1 UAT Scenarios

```gherkin
Scenario: 792a11c1a A provable downgrade is refused
  Given the installed binary was built at a revision that the clone's current revision does not descend from
  When the user runs engram update from the clone
  Then the install is refused, the error says it would downgrade the installed binary and names --allow-downgrade, and the exit status is non-zero

Scenario: 792a11c1b The flag bypasses the check
  Given the same provable downgrade
  When the user runs engram update --allow-downgrade
  Then the binary is installed

Scenario: 792a11c1c A first install is not checked
  Given no binary exists at the install location
  When the user runs engram update from a clone
  Then the binary is installed without a downgrade check

Scenario: 792a11c1d An unknown installed revision does not block
  Given the installed binary's revision is not a commit known to the clone
  When the user runs engram update from the clone
  Then the binary is installed and the report does not claim a downgrade

Scenario: 792a11c1e An unreadable revision does not block
  Given the installed binary carries no revision, or one that does not parse
  When the user runs engram update from the clone
  Then the binary is installed
```

### 792a11d The report names the source and the install

Requirement: 792a11d `engram update` MUST begin its report with the line `engram update`, then a source
line naming `local clone at <path> (rev <short SHA>)` or `remote module github.com/toejough/engram <short
SHA>`, then a binary line naming the install command, and after a real install `... ok (engram <short SHA>
→ <install location>)`. It MUST show a path under the home directory as `~/…`, and MUST show the revision
as `unknown` when git cannot resolve one.

#### 792a11d1 UAT Scenarios

```gherkin
Scenario: 792a11d1a A local install reports its revision
  Given a local clone at ~/src/engram at commit abc1234
  When the user runs engram update
  Then the report's source line is "local clone at ~/src/engram (rev abc1234)"

Scenario: 792a11d1b A real install reports where the binary went
  Given the install succeeded to /home/u/go/bin/engram
  When the user runs engram update
  Then the report's binary line ends with "ok (engram <short SHA> → ~/go/bin/engram)"

Scenario: 792a11d1c A remote source reports its revision
  Given the binary was built from a fresh clone at commit abc1234
  When the user runs engram update
  Then the report's source line is "remote module github.com/toejough/engram abc1234"
```

### 792a11e The rest of the update runs in the new binary

Requirement: 792a11e After a successful install, `engram update` MUST print the install result once,
then hand the rest of the update (deploying to the harnesses and checking the vault and chunk index) to
the freshly installed binary at the install location, and MUST exit with that binary's exit status
without repeating those steps itself. The handed-off run MUST NOT install again and MUST NOT print the
install result a second time. When the installed binary cannot be started, `engram update` MUST complete
the rest of the update itself and MUST say in its report that the re-exec failed, with the reason, and
that pre-update logic completed the run. It MUST NOT hand off when `--dry-run` is given, because nothing
was installed.

Rationale: an update that installed a new release and then deployed with the old release's logic made
users one update behind on update behavior itself, so a new deployment rule took effect a run late
(ADR-0023). The bootstrap run, the first update after this behavior shipped, is the last one to run
stale. Local mode installs from the clone it runs in, not from a fresh pull, so a stale checkout still
downgrades on hand-off; the downgrade check covers it (792a11c).

#### 792a11e1 UAT Scenarios

```gherkin
Scenario: 792a11e1a The deploy runs in the new binary
  Given an update that installs successfully
  When the user runs engram update
  Then the install result is printed once, the deploy report is printed once by the installed binary, and the exit status is that binary's

Scenario: 792a11e1b The install result comes first
  Given an update that installs successfully
  When the user runs engram update
  Then the output begins with the "engram update", source and binary lines, followed by the deploy report with no second header

Scenario: 792a11e1c A failed hand-off falls back
  Given an install that succeeds but an installed binary that cannot be started
  When the user runs engram update
  Then the update still deploys to the harnesses, and its report says re-exec failed, completed with pre-update logic, and why

Scenario: 792a11e1d A handed-off run does not claim an install
  Given an update running as the handed-off binary
  When it prints its report
  Then the report does not say that go install ran

Scenario: 792a11e1e A dry run never hands off
  Given any machine
  When the user runs engram update --dry-run
  Then nothing is installed and no second process is started
```

### 792a11f Previewing an update

Requirement: 792a11f `engram update --dry-run` MUST install nothing, MUST create, change and delete no
file, directory or link in a harness, an engram-owned root or the vault (including a root's marker, and
including a sync that would create or delete), and MUST print the deploy plan, prefixing with `[dry-run] ` every report line that
describes something that was or would be done (the title, the `installed:` line, the root deletions, the
adoptions, the dangling-link cleanups and the guidance deploy lines) and leaving notice lines (a refused
deletion, an unattributable file, an unmanaged entry, a skill-row mapping) unprefixed.

Rationale: the same prefix discipline applies to every action line, so a reader can tell what happened
from what would happen without knowing which operation produced the line (#709). A dry run from outside a
checkout still clones the repository into its scratch directory, because the plan is read from the clone;
it builds nothing.

#### 792a11f1 UAT Scenarios

```gherkin
Scenario: 792a11f1a A dry run on a never-synced harness leaves no trace
  Given a machine whose ~/.claude holds no engram root
  When the user runs engram update --dry-run
  Then the report lists the planned sync and link operations with the [dry-run] prefix, and afterwards no engram root, marker, link or file exists

Scenario: 792a11f1b Action lines carry the prefix, notices do not
  Given a dry run that would delete a root file, adopt a copied skill, remove a dangling link and refresh guidance, and that finds an unmanaged entry
  When the user runs engram update --dry-run
  Then each action line begins with "[dry-run] " and the unmanaged line does not

Scenario: 792a11f1c A dry run installs nothing
  Given a local clone newer than the installed binary
  When the user runs engram update --dry-run
  Then the installed binary is unchanged and the binary line shows the planned install command without "ok"
```

### 792a11g The harnesses that are updated

Requirement: 792a11g `engram update` MUST update each of Claude Code (detected by `~/.claude`) and Pi
(detected by `~/.pi`) whose directory exists, MUST NOT create the directory of one that is absent, and
MUST fail with an error saying no supported harness was found at `~/.claude/` or `~/.pi/` when neither
exists, before installing anything. It MUST NOT deploy to any other harness. A failure in one harness MUST
NOT stop the other from being updated, MUST be reported under that harness as `error: <reason>`, MUST be
left out of the `installed:` line, and MUST make the command exit non-zero.

Rationale: OpenCode and Shelley support was removed (#721), so the harness list is Claude Code and Pi.

#### 792a11g1 UAT Scenarios

```gherkin
Scenario: 792a11g1a Only the harnesses present are updated
  Given ~/.claude exists and ~/.pi does not
  When the user runs engram update
  Then the report has a Claude Code section and no Pi section, and ~/.pi is not created

Scenario: 792a11g1b Both harnesses are updated in a stable order
  Given ~/.claude and ~/.pi both exist
  When the user runs engram update
  Then the report lists Claude Code then Pi, and the installed line names both

Scenario: 792a11g1c No harness is an error before any install
  Given neither ~/.claude nor ~/.pi exists
  When the user runs engram update
  Then it fails saying no supported harness was found at ~/.claude/ or ~/.pi/, and nothing was installed

Scenario: 792a11g1d One failing harness does not stop the other
  Given writing under ~/.claude fails and ~/.pi is healthy
  When the user runs engram update
  Then Pi is updated, the Claude Code section shows "error: <reason>", the installed line names only Pi, and the exit status is non-zero

Scenario: 792a11g1e A missing skills source is an error
  Given the source checkout has no agent-instructions/skills directory
  When the user runs engram update
  Then it fails with an error that the skills source directory is missing
```

### 792a11h Skills are deployed to an engram-owned root and linked

Requirement: 792a11h `engram update` MUST copy every file under `agent-instructions/skills/` of the
source into the `skills/` subtree of the harness's engram-owned root (`~/.claude/engram/` for Claude Code,
`~/.pi/agent/engram/` for Pi), creating a missing file and overwriting a different one, and MUST make
`~/.claude/skills/<skill>` and `~/.pi/agent/skills/<skill>` each a symbolic link to that skill's directory
in the root, one link per skill directory, never a copy. It MUST repoint a link that points anywhere else,
and MUST leave a correct link alone. Its report MUST list each skill as `agent-instructions/skills/<skill>/
→ <skills dir>/<skill>/  (<N> file[s])`, then `installed: <harnesses>`.

#### 792a11h1 UAT Scenarios

```gherkin
Scenario: 792a11h1a A skill deploys as a directory link
  Given the source holds skill recall and a symlink-mode harness whose skills dir is ~/.claude/skills
  When the user runs engram update
  Then ~/.claude/skills/recall is a symbolic link to the skills/recall directory of ~/.claude/engram, and its SKILL.md is readable through the link

Scenario: 792a11h1b Added and changed sources deploy
  Given a new skill and an edited file in an existing skill since the last update
  When the user runs engram update
  Then the new skill appears in the engram root with a link from the harness, and the edited file in the root matches the source

Scenario: 792a11h1c A link that points elsewhere is repointed
  Given ~/.claude/skills/recall is a symbolic link to a different directory
  When the user runs engram update
  Then it is a link to the engram root's skills/recall

Scenario: 792a11h1d The report lists each skill and its file count
  Given a source skill learn with 3 files and recall with 1
  When the user runs engram update
  Then the report lists "agent-instructions/skills/learn/ → ~/.claude/skills/learn/  (3 files)" and "... recall/ → ...  (1 file)"

Scenario: 792a11h1e Re-running changes nothing
  Given an update has completed against the same source
  When the user runs engram update again
  Then every root file and link is the same as after the first run
```

### 792a11i Removal propagates

Requirement: 792a11i `engram update` MUST delete from a harness's engram-owned root every file in a
managed subtree (`skills/`, `commands/` and, while guidance is managed, `guidance/`) that is not in the
deploy set it computes from the source (engram deploys no commands, so it deletes any file in `commands/`), MUST delete a directory in those subtrees that would be left holding nothing
the deploy set keeps, and MUST list each deletion in the report as `engram root: deleted <path>`. The
harness link for a skill whose directory was deleted MUST be gone by the end of the same run.

Rationale: an earlier update copied additively, so an artifact deleted from the source was never removed
from a harness, and a stray file in the source persisted until deleted by hand (ADR-0022). Removing a
skill is now a source change followed by an update.

#### 792a11i1 UAT Scenarios

```gherkin
Scenario: 792a11i1a A removed skill disappears
  Given a skill directory that was deployed earlier is deleted from agent-instructions/skills/
  When the user runs engram update
  Then the skill's subtree is deleted from the engram root, its link in the harness skills directory is gone, and the report lists the deletion

Scenario: 792a11i1b A removed file inside a kept skill disappears
  Given a file deleted from a skill that is still in the source
  When the user runs engram update
  Then the file and any directory it leaves empty are deleted from the engram root
```

### 792a11j Only what engram owns is deleted

Requirement: 792a11j `engram update` MUST NOT delete a file in an engram-owned root that lacks the
ownership marker `.engram-owned` at the start of the run, and MUST NOT delete a real (non-link) file or
directory in a harness's skills or guidance directory. A root that exists without the marker MUST get the
marker and its deploy set written, MUST NOT have anything deleted in it that run, and MUST be reported
with `engram root <path> exists without the .engram-owned marker — sync-deletion refused there` and one
`unattributable: <name>` line for each file at its top level that matches no deploy-set name. On every
run it MUST delete, from the skills directory and the guidance directory (or the engram root, for a
harness whose guidance directory is its root), exactly the symbolic links that point lexically into the
harness's engram root and whose target no longer exists, reporting each as `cleanup: removed dangling link
<path>`, and MUST NOT touch a link that points anywhere else.

Rationale: engram is the only writer inside its root, and the only deletions outside it are links that
provably point into it. The marker is what proves a directory named `engram/` is engram's.

#### 792a11j1 UAT Scenarios

```gherkin
Scenario: 792a11j1a An unmarked root is not purged
  Given the engram root path exists without the ownership marker and holds a file not in the deploy set
  When the user runs engram update
  Then the file is left in place and listed as unattributable, the marker is written, and no deletion happens in that directory

Scenario: 792a11j1b The marker enables deletion on the next run
  Given the previous update wrote the marker into a root that held an extra file
  When the user runs engram update again
  Then a file in the managed subtrees that is not in the deploy set is deleted

Scenario: 792a11j1c A dangling engram link is removed and a foreign link kept
  Given a skills directory holding a link into the engram root whose target was removed and a second link to a path outside the root
  When the user runs engram update
  Then the first link is deleted and the second is untouched

Scenario: 792a11j1d User-owned real files are never deleted
  Given a harness skills directory holds a real skill directory the user installed by other means
  When the user runs engram update
  Then it is untouched by the sync, the cleanup and the migration

Scenario: 792a11j1e A root that is a file is refused
  Given the engram root path is a regular file
  When the user runs engram update
  Then that harness reports an error that the engram root exists and is not a directory, and the file is untouched
```

### 792a11k The first update adopts earlier copies

Requirement: 792a11k `engram update` MUST replace each deploy-set skill that is a real directory in the
harness skills directory (a copy made by an earlier version) with a link to the root's copy, and MUST
report each as `adopted: <path>`. On the first update of a harness (its engram root absent or lacking the
marker) it MUST also create the root and its marker, and MUST list every other real file or directory in
the harness's skills or guidance directory as `unmanaged (left alone): <path>` without deleting or
changing it. Later runs MUST NOT list unmanaged entries again.

Rationale: the repository is the source of truth, so an earlier copy is replaced without comparing its
content. Engram cannot prove it wrote an entry it never recorded, so it does not call one engram's
debris (#716).

#### 792a11k1 UAT Scenarios

```gherkin
Scenario: 792a11k1a A copied skill becomes a link
  Given the first update of a harness whose skills directory holds recall as a real directory from an earlier copy-mode deploy
  When the user runs engram update
  Then recall's content lives in the engram root, the harness path is a link to it, the skill is still discoverable, and the report lists it as adopted

Scenario: 792a11k1b An unmanaged entry is reported, not deleted
  Given the first update finds a real directory in the skills directory that is not in the deploy set
  When the user runs engram update
  Then it is left in place and the report lists it as unmanaged, left alone

Scenario: 792a11k1c The unmanaged list is not repeated
  Given a later update of the same harness with the same unmanaged entry
  When the user runs engram update
  Then the report does not list it
```

### 792a11l Guidance is managed only after opt-in

Requirement: 792a11l `engram update` MUST include the guidance documents of `agent-instructions/guidance/`
(the `.md` files directly in it) in the deploy set of every detected harness only when `--with-guidance`
is given or the config file of at least one detected harness already imports a guidance file, and MUST
otherwise leave the root's `guidance/` subtree and any guidance link untouched, neither refreshing nor
deleting it. A config file imports a guidance file when a line outside a code fence is exactly
`@~/<guidance dir>/<name>.md` or the same path with the home directory spelled out (`~/.claude/CLAUDE.md`
with the `~/.claude/engram/` guidance dir for Claude Code, `~/.pi/agent/AGENTS.md` with
`~/.pi/agent/guidance/` for Pi). It MUST deploy a guidance document as `<engram root>/guidance/<name>.md`
and link it from the harness's guidance directory, and MUST delete from `guidance/`, when guidance is
managed, a document that has left the source.

Rationale: management is an opt-in once, so that after a user wires an import, a plain `engram update`
keeps their guidance current as it does skills. Not being in the deploy set means unmanaged, not removed.
The import lines, the hints that name them and the guidance documents themselves belong to
ambient-guidance.

#### 792a11l1 UAT Scenarios

```gherkin
Scenario: 792a11l1a The flag deploys guidance
  Given the source holds guidance recall.md and the user passes --with-guidance
  When the user runs engram update
  Then ~/.claude/engram/guidance/recall.md is written

Scenario: 792a11l1b An existing import keeps guidance current
  Given ~/.claude/CLAUDE.md imports @~/.claude/engram/recall.md and the user passes no flag
  When the user runs engram update
  Then guidance is refreshed

Scenario: 792a11l1c Without opt-in guidance is untouched
  Given guidance files were deployed earlier, no flag is passed, and the config file no longer imports them
  When the user runs engram update
  Then the guidance files are neither refreshed nor deleted

Scenario: 792a11l1d An import inside a code fence does not count
  Given the only line importing guidance is inside a fenced code block of the config file
  When the user runs engram update
  Then guidance is not managed

Scenario: 792a11l1e A guidance document removed from the source is deleted once managed
  Given guidance is managed and a deployed guidance file has left the source
  When the user runs engram update
  Then it is deleted from the engram root's guidance subtree and its link is gone

Scenario: 792a11l1f Pi guidance is linked into Pi's guidance directory
  Given the user passes --with-guidance and ~/.pi exists
  When the user runs engram update
  Then ~/.pi/agent/guidance/recall.md is a link to ~/.pi/agent/engram/guidance/recall.md

Scenario: 792a11l1g One harness's import manages guidance for every harness
  Given only ~/.claude/CLAUDE.md imports a guidance file, no flag is passed, and ~/.pi exists
  When the user runs engram update
  Then guidance is also deployed for Pi
```

### 792a11m Claude import paths keep working

Requirement: 792a11m For Claude Code, whose guidance directory is its engram root, `engram update` MUST
leave for each managed guidance document a symbolic link at `~/.claude/engram/<name>.md` pointing at
`~/.claude/engram/guidance/<name>.md`, replacing a real file there from an earlier version, so that an
`@~/.claude/engram/<name>.md` import in `CLAUDE.md` still resolves to the deployed content. It MUST delete
such a link when its document leaves the deploy set, and its report MUST give the canonical `guidance/`
path of each deployed Claude Code document.

Rationale: user-authored import lines name the flat path verbatim, so the canonical move into
`guidance/` could not break them. A later change can retire the links once imports are migrated.

#### 792a11m1 UAT Scenarios

```gherkin
Scenario: 792a11m1a An existing import still resolves
  Given the first sync moves recall.md into the root's guidance subtree and CLAUDE.md imports ~/.claude/engram/recall.md
  When the user runs engram update with guidance managed
  Then ~/.claude/engram/recall.md is a link to ~/.claude/engram/guidance/recall.md and the import resolves to the deployed content

Scenario: 792a11m1b A real flat file from an earlier version is adopted
  Given ~/.claude/engram/recall.md is a real file from an earlier version
  When the user runs engram update with guidance managed
  Then it becomes a link to the canonical copy and the report lists it as adopted

Scenario: 792a11m1c A link whose document left the source is removed
  Given a compat link for a guidance document that is no longer in the source
  When the user runs engram update
  Then the link is deleted

Scenario: 792a11m1d The report states the canonical path
  Given Claude Code guidance is deployed
  When the user runs engram update
  Then each guidance line for Claude Code ends with "(canonical: ~/.claude/engram/guidance/<name>)", and Pi's lines carry no such suffix
```

### 792a11n Backlog notices for the chunk index

Requirement: 792a11n `engram update` MUST print the line ``duplicate chunk-index files found — run `engram
prune --duplicates` to clear them (preview with `engram prune --duplicates --dry-run`)`` when running
`engram prune --duplicates` would remove at least one index file, and ``empty chunk-index files found —
run `engram prune --empty` to clear them (preview with `engram prune --empty --dry-run`)`` when the chunk
index holds a zero-byte index file. It MUST print neither when the index is clean, when its manifest or
directory is missing or unreadable, or, for the duplicates notice, when every duplicate group would be
refused by prune's coverage check. It MUST NOT remove anything itself.

Rationale: removing index files is a destructive one-shot migration over a user's memory store. Unlike an
empty file, which holds no records, a duplicate's removal is safe only while the record-subset check holds
at the moment it runs, so it runs when the user asks for it and not as a side effect of a routine
refresh (ADR-0021 decision 9). Detection keeps no sentinel: it reports the live backlog every run and
goes quiet once it is cleared. It runs prune's own dry-run check so the notice fires only when a removal
would happen, and a backlog whose every group would be refused stays silent because no command resolves
it (#713).

#### 792a11n1 UAT Scenarios

```gherkin
Scenario: 792a11n1a Update reports the prune command when duplicates exist
  Given a chunk index with several canonical entries per content-hash group whose removal prune would allow
  When the user runs engram update
  Then it prints the duplicates notice naming engram prune --duplicates and its --dry-run preview, and removes nothing

Scenario: 792a11n1b A refusal-only backlog is silent
  Given duplicate groups none of which prune --duplicates would remove
  When the user runs engram update
  Then it prints no duplicates notice

Scenario: 792a11n1c Empty index files are reported
  Given the chunk index holds a zero-byte .jsonl file
  When the user runs engram update
  Then it prints the empty-files notice naming engram prune --empty

Scenario: 792a11n1d A clean or missing index is silent
  Given a chunk index with no duplicates and no empty files, or no chunk index
  When the user runs engram update
  Then it prints neither notice

Scenario: 792a11n1e The notices coexist with the other upgrade notices
  Given both backlogs and a vault with old-format vocabulary files
  When the user runs engram update
  Then the report contains the empty-files notice, the duplicates notice and the vocabulary notice
```

### 792a11o The update options

Requirement: 792a11o `engram update --help` MUST describe the update as refreshing the engram binary and
harness skills, and MUST list `--dry-run`, `--with-guidance`, `--allow-downgrade`, `--regen-vocab`,
`--reparent-luhmann`, `--answers <string>` and `--backfill-identity`. `engram update` MUST pass
`--with-guidance`, `--regen-vocab` and `--backfill-identity` on to the handed-off run (792a11e), and MUST
NOT pass `--dry-run`, which never hands off. `--reparent-luhmann` MUST run only the re-parenting of
792a10, without installing or deploying anything (792a10b).

#### 792a11o1 UAT Scenarios

```gherkin
Scenario: 792a11o1a The help lists every option
  Given any machine
  When the user runs engram update --help
  Then it describes refreshing the binary and harness skills and lists --dry-run, --with-guidance, --allow-downgrade, --regen-vocab, --reparent-luhmann, --answers and --backfill-identity

Scenario: 792a11o1b Options reach the handed-off run
  Given an update that installs and hands off
  When the user runs engram update --with-guidance --regen-vocab --backfill-identity
  Then the installed binary is started as engram update with those three options and no others

Scenario: 792a11o1c Re-parenting skips the install and the deploy
  Given a machine with no harness and no Go toolchain
  When the user runs engram update --reparent-luhmann
  Then it derives candidates and installs and deploys nothing
```

## 792a12 Added: skill-runbook-registration

Canonical target: `docs/specs/product/skill-runbook-registration.md` (title `Skill Runbook Registration - Product Spec`).

Skills remain the shipping form of every procedure, and their files stay exactly as the harnesses that load them expect. Engram's own skills ship from `agent-instructions/skills/` through `engram update`. Registration lets memory surface a procedure by situation and trigger: each skill, command and Pi prompt template in the folders a harness loads may have one runbook note in the vault that mirrors its text and carries the runbook fields authored on the note, so the runbook shim's follow-frame can drive it and it can flow across vault-graph edges. Engram offers to create, refresh and remove these notes, the user decides, and a declined offer is not repeated. `engram update` runs the offers after deploying (792a12t), and `engram register-skills` runs them on their own. A note mirrors a skill and does not replace it: the direction that retired skills to runbook-only form was reversed (792d57). Curating the pending notes belongs to the curate capability, and the skills' own procedures to recall, learn, route, please, curate and write-memory.

### 792a12a Skill files are never changed

Requirement: 792a12a Registration MUST NOT read, require or write any frontmatter field of a skill's `SKILL.md`, a command's
`.md` file or a Pi prompt template, other than using the file's bytes as the note body and as the hash
input, and MUST NOT require or add any file beside them.

Rationale: the harness owns the schema of those files, and the runbook fields (`situation`, `triggers`,
`done_when`, `red_flags`) are authored on the note, never in the skill.

#### 792a12a1 UAT Scenarios

```gherkin
Scenario: 792a12a1a A deployed skill is byte-identical
  Given a skill whose SKILL.md frontmatter has only name and description, deployed and registered
  When the user runs engram register-skills
  Then the deployed SKILL.md is byte-identical to the source and no file is added to the skill directory

Scenario: 792a12a1b Command and prompt files are untouched
  Given a command .md file with description and argument-hint frontmatter, and a Pi prompt template, both registered
  When the user runs engram register-skills
  Then both files are byte-identical afterward, no file is added beside them, and their keys do not depend on any frontmatter field
```

### 792a12b One runbook note per skill, identified by its key

Requirement: 792a12b Each registered skill, command or prompt template MUST have at most one runbook note, which registration
MUST identify as the note of type runbook whose basename slug begins `skill-`, that carries a non-empty
`skill_hash`, and whose `skill_key` equals the key. A note's slug alone MUST NOT identify it. A runbook
note with `skill_hash` but no `skill_key`, one whose `skill_key` is unrecognized, and one without
`skill_hash` MUST NOT be matched to any key, MUST NOT make any entry an alias, and MUST NOT be offered
for removal. A key is unrecognized only when it has no `:`, has an empty segment, or begins with a
reserved source prefix (`claude`, `pi`, `agents`, `project`, `anthropic-skills`, `pi-settings`,
`pi-pkg`, `pi-prompt`) whose remainder lacks that source's shape; any other `<x>:<name>` is plugin
`<x>`'s key. More than one note for a key MUST be an error naming them.

#### 792a12b1 UAT Scenarios

```gherkin
Scenario: 792a12b1a Note located by skill_key
  Given a runbook note with skill_hash and skill_key superpowers:brainstorming
  When the user runs engram register-skills
  Then registration treats it as the note for key superpowers:brainstorming

Scenario: 792a12b1b Note not located by slug
  Given the vault holds 1049.2026-09-21.skill-curate.md of type runbook with a skill_hash field and no skill_key
  When the user runs engram register-skills
  Then it is not the note of claude:curate or of any other key

Scenario: 792a12b1c Duplicate notes are an error
  Given two runbook notes with skill_hash that both carry skill_key claude:curate
  When the user runs engram register-skills
  Then registration reports an error naming both and makes no change for claude:curate

Scenario: 792a12b1d Unkeyed legacy notes are ignored until adopted
  Given notes 1036, 1045, 1049, 1053, 1067 and 1068 with skill-<name> slugs, no skill_key, and skill_hash equal to the scanned skills' hashes
  When the user runs engram register-skills
  Then none is rewritten, no refresh or removal is offered for them, and registering claude:<name> is offered for each of the six skills

Scenario: 792a12b1e An unrecognized skill_key is never matched or removed
  Given a runbook note with skill_hash and skill_key route, and every source root read
  When the user runs engram register-skills
  Then it is matched to no entry and no removal of it is offered

Scenario: 792a12b1f A skill-prefixed slug without skill_hash is not a skill note
  Given a runbook note whose slug is skill-edits-validated-by-baseline-pressure-tests and that has no skill_hash
  When the user runs engram register-skills
  Then it is neither matched to a skill nor offered for removal

Scenario: 792a12b1g A keyed note with another slug is not found
  Given a runbook note with skill_hash and skill_key claude:curate whose slug does not begin skill-
  When the user runs engram register-skills
  Then it is not the note of claude:curate, so registering claude:curate is offered
```

### 792a12c Registration offers and remembers declines by hash

Requirement: 792a12c For each scanned skill, command or prompt template that is not an alias and has no key conflict,
registration MUST offer to register it when it has no note, and to refresh its note when the note's
`skill_hash` differs from the current hash, and MUST offer to remove a note when its root was read and
no scanned entry has its key (792a12q). It MUST make no offer when the hashes match or when the key's
current hash is recorded as declined. Declining an offer MUST record the current hash against the key in
the vault-root file `skill-registrations.json` and change nothing else; for a removal offer the recorded
hash is the note's `skill_hash`.

#### 792a12c1 UAT Scenarios

```gherkin
Scenario: 792a12c1a A new skill is offered once
  Given a scanned skill with no note that the user declined, unchanged since
  When the user runs engram register-skills again
  Then no offer is made for that skill

Scenario: 792a12c1b A changed skill is offered again after a decline
  Given the user declined a skill's refresh at hash A and its text now has hash B
  When the user runs engram register-skills
  Then the refresh is offered once for hash B

Scenario: 792a12c1c A matching hash makes no offer
  Given a skill whose note has skill_hash equal to the skill's current hash
  When the user runs engram register-skills
  Then no offer is made and nothing is written for that skill

Scenario: 792a12c1d A declined removal is not offered again
  Given a removal the user declined at the note's current skill_hash
  When the user runs engram register-skills
  Then no removal is offered for that note
```

### 792a12d Accepting a registration creates a pending note

Requirement: 792a12d Accepting a registration offer MUST create a runbook note through the normal capture path: a fresh
top-level Luhmann ID, the slug derived from the key, a sidecar embedded on write, the `source:` line
`skill registration: <skill_source>`, `skill_hash` (the SHA-256 of the source file's bytes),
`skill_key`, `skill_source` (the resolved source path, home-relative with `~/` when under the home
directory) and `pending: true`, and none of `situation`, `triggers`, `done_when` or `red_flags`. Its
body MUST be the one-line preamble ``> Mirrors skill `<skill_source>`.``, a blank line, and the source
file's bytes verbatim. The preamble MUST use the same wording for every source and MUST NOT say where to
edit the procedure.

Rationale: the runbook fields are authored on the note afterward, by curation, because only the skill's text is
known at registration.

#### 792a12d1 UAT Scenarios

```gherkin
Scenario: 792a12d1a Accepted registration
  Given the user accepts registration of claude:curate, whose resolved source is ~/.claude/engram/skills/curate/SKILL.md
  When the user runs engram register-skills
  Then a runbook note ending in .skill-claude-curate.md exists with the skill's text after a preamble naming ~/.claude/engram/skills/curate/SKILL.md, skill_hash, pending: true, no runbook fields, and a .vec.json sidecar

Scenario: 792a12d1b A plugin skill's preamble names its real file
  Given the user accepts registration of superpowers:brainstorming, whose resolved source is ~/.claude/plugins/cache/claude-plugins-official/superpowers/6.4.1/skills/brainstorming/SKILL.md
  When the user runs engram register-skills
  Then the note carries skill_key superpowers:brainstorming, that skill_source, and a preamble naming that path

Scenario: 792a12d1c A command is registered from its file
  Given the user accepts registration of project command project:github.com/toejough/engram:cmd:opsx:apply
  When the user runs engram register-skills
  Then the note's body is .claude/commands/opsx/apply.md after a preamble naming that file, and its basename ends in .skill-project-github-com-toejough-engram-cmd-opsx-apply.md
```

### 792a12e Accepting a refresh replaces the body, keeps the fields, and marks the note pending

Requirement: 792a12e Accepting a refresh offer MUST replace the note's body with the current source file after the preamble,
set `skill_hash` to the current hash, set `skill_source` to the current resolved source while keeping
`skill_key`, rebuild the sidecar, and set `pending: true` so that curation re-checks the fields against
the new text. It MUST preserve `situation`, `triggers`, `done_when`, `red_flags`, `created`, the
basename, and every other frontmatter key with its value, including keys the runbook note model does not
define. It MUST convert a CRLF note to LF in the same single write. It MUST refuse the note untouched,
naming the key, when a key it is about to set or delete (`skill_hash`, `skill_key`, `skill_source`,
`pending`), or that key's value, carries a YAML anchor, and MUST NOT write frontmatter that does not
decode.

#### 792a12e1 UAT Scenarios

```gherkin
Scenario: 792a12e1a Accepted refresh
  Given the user accepts a refresh of a note whose skill changed
  When the user runs engram register-skills
  Then the note has the new body and hash, its authored fields and basename are unchanged, and it carries pending: true

Scenario: 792a12e1b Refresh follows a plugin version bump
  Given a plugin skill's bytes changed under a new installPath version and the refresh is accepted
  When the user runs engram register-skills
  Then skill_source and the preamble name the new version's path

Scenario: 792a12e1c Refresh keeps an unmodeled key
  Given a refresh accepted for a note whose frontmatter carries a key the runbook note model does not define, such as luhmann_old: "12"
  When the user runs engram register-skills
  Then the refreshed note still carries that key with the same value

Scenario: 792a12e1d Refresh converts a CRLF note
  Given a refresh accepted for a skill note whose lines end in \r\n
  When the user runs engram register-skills
  Then it succeeds, the note is converted to LF inside its single write, and its authored fields are unchanged

Scenario: 792a12e1e Refresh refuses an anchored key
  Given an accepted refresh that would replace or delete a skill_hash, skill_key, skill_source or pending key that carries a YAML anchor, or whose value does
  When the user runs engram register-skills
  Then the refresh is refused naming the key and the note is unchanged

Scenario: 792a12e1f Authored fields survive a refresh
  Given red_flags changed on a skill note with engram amend, then a refresh accepted
  When the user runs engram register-skills
  Then the amended red_flags are still present
```

### 792a12f Accepting a removal deletes the note and its sidecar

Requirement: 792a12f Accepting a removal offer MUST delete the skill note and its `.vec.json` sidecar. Registration MUST NOT
delete any note without an accepted offer.

#### 792a12f1 UAT Scenarios

```gherkin
Scenario: 792a12f1a A skill no longer shipped
  Given a skill note for a skill absent from the scanned set, and the user accepts its removal
  When the user runs engram register-skills
  Then the note and its sidecar are gone
```

### 792a12g Registration never prompts or writes without a terminal

Requirement: 792a12g When stdin is not a terminal and no `--accept` or `--decline` answer covers an offer, registration MUST
NOT prompt, MUST NOT write a note, and MUST NOT record a decline for that offer. It MUST print one line
giving the total number of outstanding offers (`offer` for one, `offers` otherwise), each scope's
selector with its count, and the commands that answer them: ``engram: <N> skill runbook offers awaiting
an answer: @<scope> <count>, … — run `engram register-skills` in a terminal, or `engram register-skills
--accept <key|prefix*|@scope>` / `--decline <key|prefix*|@scope>` ``.

#### 792a12g1 UAT Scenarios

```gherkin
Scenario: 792a12g1a An agent runs update through a shell
  Given stdin is not a terminal and one scanned skill has no note
  When the user runs engram update
  Then nothing is written, skill-registrations.json is unchanged, and the output names the skill's scope selector with its count and the answering command
```

### 792a12h Registration can run on its own with explicit answers

Requirement: 792a12h `engram register-skills` MUST run the same comparison over the same default source set as `engram
update`. `--accept <key|pattern|@scope>` and `--decline <key|pattern|@scope>` (each repeatable) MUST
answer the matching offers without prompting, and an answer that matches no offer MUST print ``engram:
no pending offer for skill "<token>" — ignoring --accept`` (or `--decline`) and continue. `--dry-run`
MUST list every offer and write nothing, and run no `--adopt`. `--vault` and `--vault-name` MUST choose
the vault and the name stamped on a new note's `vault:` field. `--skills-dir <dir>` (repeatable,
relative to the working directory) MUST replace the default set with the listed directories, each
scanned as Claude Code user skills and keyed `claude:<name>`; that run MUST be a read-only preview that
behaves as `--dry-run`, refuses `--accept`, `--decline` and `--adopt` before scanning with
``register-skills: --skills-dir is a read-only preview; it cannot be combined with --accept, --decline
or --adopt``, and makes no removal offer. The same key, pattern or selector named in both `--accept` and
`--decline` MUST be refused before anything is acted on.

#### 792a12h1 UAT Scenarios

```gherkin
Scenario: 792a12h1a An agent relays the user's answer
  Given offers for claude:curate and claude:route
  When the user runs engram register-skills --accept claude:curate --decline claude:route non-interactively
  Then claude:curate's offer is carried out, claude:route's current hash is recorded as declined, and other offers are only reported

Scenario: 792a12h1b A dry run
  Given outstanding offers
  When the user runs engram register-skills --dry-run
  Then it prints each offer it would make under scope headers and writes nothing

Scenario: 792a12h1c An answer matching nothing is noted
  Given no offer for the key nosuch
  When the user runs engram register-skills --accept nosuch
  Then it prints the no-pending-offer line and carries on with the other offers

Scenario: 792a12h1d A skills-dir run is a preview
  Given a directory agent-instructions/skills holding skills
  When the user runs engram register-skills --skills-dir agent-instructions/skills
  Then it lists offers keyed claude:<name>, writes nothing, never rewrites a note's preamble, and lists no removal offer

Scenario: 792a12h1e A skills-dir run refuses answers
  Given any directory
  When the user runs engram register-skills --skills-dir agent-instructions/skills --accept claude:route
  Then it refuses with an error before scanning

Scenario: 792a12h1f One answer in both flags is refused
  Given an offer
  When the user runs engram register-skills --accept claude:route --decline claude:route
  Then it is refused before anything is acted on
```

### 792a12i An existing runbook note can be adopted as a skill's note

Requirement: 792a12i `engram register-skills --adopt <key>=<note-ref>` MUST, for a runbook note named by Luhmann ID or
basename, rename it to the slug derived from `<key>` keeping its Luhmann ID and date and recording its
old basename in `aliases`, rewrite every inbound wikilink and move its sidecar, replace its body with
the preamble and the current source file, set `skill_hash`, `skill_key` and `skill_source`, remove any
`pending` marker, and preserve its runbook fields and every other frontmatter key with its value,
including keys the runbook note model does not define. `<key>` MUST name a scanned, unconflicted skill,
command or prompt template. Adopting a note that already carries the key's slug MUST be a rename that
changes nothing and still refresh the body and fields. It MUST refuse, writing nothing, when the
reference names no note or more than one, when the note is not a runbook, when it is keyed to a
different skill, when the key is registered to a different note, when two `--adopt` entries name one key
or one note, when the note's basename has no Luhmann ID and date, and, before any rename, when a key it
is about to set or delete (or that key's value) carries a YAML anchor or the rewritten frontmatter would
not decode. All `--adopt` entries MUST run before the offers are computed.

#### 792a12i1 UAT Scenarios

```gherkin
Scenario: 792a12i1a Adopting a previously promoted note
  Given note 1049 is 1049.2026-09-21.skill-curate.md with no skill_key
  When the user runs engram register-skills --adopt claude:curate=1049
  Then 1049 is renamed to 1049.2026-09-21.skill-claude-curate.md, links to its old basename now point to the new one, its body is the curate skill after a preamble naming its skill_source, its runbook fields are unchanged, and it has skill_hash, skill_key claude:curate, skill_source and no pending marker

Scenario: 792a12i1b Adopt keeps an unmodeled key
  Given a runbook note whose frontmatter carries a key the runbook note model does not define
  When --adopt runs on it
  Then the adopted note still carries that key with the same value

Scenario: 792a12i1c Adopt refuses an anchored key
  Given a runbook note whose skill_hash, skill_key, skill_source or pending key carries a YAML anchor, or whose value does
  When --adopt targets it
  Then the adopt is refused naming the key and the note is neither renamed nor rewritten

Scenario: 792a12i1d Adopt converts a CRLF note
  Given a runbook note whose lines end in \r\n
  When --adopt runs on it
  Then the note is converted to LF and renamed, and its links are rewritten

Scenario: 792a12i1e Adopt refuses a note that is not a runbook
  Given a fact note
  When --adopt targets it
  Then it is refused and nothing changes

Scenario: 792a12i1f Adopt refuses a note keyed to another skill
  Given a note whose skill_key is claude:route
  When --adopt claude:curate targets it
  Then it is refused and the note is not re-keyed

Scenario: 792a12i1g Adopt refuses an unscanned key
  Given a key that no scanned entry has
  When --adopt names it
  Then it is refused and nothing is written

Scenario: 792a12i1h Adopt refuses an ambiguous reference
  Given a bare Luhmann ID shared by two notes
  When --adopt names it
  Then it is refused, asking for the full basename

Scenario: 792a12i1i A dry run adopts nothing
  Given a valid --adopt entry
  When the user runs engram register-skills --dry-run --adopt claude:curate=1049
  Then the note is neither renamed nor rewritten
```

### 792a12j Claude Code skills, commands and synced skills are found

Requirement: 792a12j Registration MUST find, for a detected Claude Code harness (`~/.claude`), its user skills as
`~/.claude/skills/<name>/SKILL.md` from immediate children that are directories or symlinks resolving to
directories, its user commands as `~/.claude/commands/**/*.md` named by their relative path with `/`
replaced by `:`, and its claude.ai-synced skills as `<bucket>/<name>/SKILL.md` for each directory bucket
under `~/.claude/skills/synced/` whose `manifest.json` parses. It MUST ignore root files, `synced`
itself as a skill, dangling symlinks and directories without `SKILL.md` among the user skills, and
non-directory entries among the buckets.

#### 792a12j1 UAT Scenarios

```gherkin
Scenario: 792a12j1a A symlinked user skill is found
  Given ~/.claude/skills/route is a symlink to ~/.claude/engram/skills/route containing SKILL.md
  When the user runs engram register-skills
  Then it is the scanned Claude user skill claude:route whose source is the resolved ~/.claude/engram/skills/route/SKILL.md

Scenario: 792a12j1b A dangling symlink is skipped
  Given ~/.claude/skills/qa is a symlink whose target does not exist
  When the user runs engram register-skills
  Then no offer is made for it and registration does not fail

Scenario: 792a12j1c A user command in a subdirectory
  Given ~/.claude/commands/opsx/apply.md exists
  When the user runs engram register-skills
  Then it is scanned as a command named opsx:apply

Scenario: 792a12j1d Synced skills come from manifest buckets
  Given ~/.claude/skills/synced/ holds one bucket directory with a parseable manifest.json and skill pdf/SKILL.md, plus a stray file .bucket-x
  When the user runs engram register-skills
  Then pdf is scanned as a synced skill and the stray file is ignored
```

### 792a12k Pi, agents and Pi-configured skills, commands and prompts are found

Requirement: 792a12k Registration MUST find, for a detected Pi harness (`~/.pi`), its user skills in `~/.pi/agent/skills/`
(recursively discovered skill directories, plus root `.md` files), the skills in `~/.agents/skills/`
(recursively, ignoring root `.md` files), its prompt templates in `~/.pi/agent/prompts/*.md` (flat,
named by file stem), and the `skills` and `prompts` paths and `packages` configured in Pi's global
`settings.json`. A package entry given as a string whose `package.json` has any `pi` key MUST load only
the manifest's `pi.skills` and `pi.prompts` entries, and the convention directories `skills/` and
`prompts/` MUST be used only when `package.json` has no `pi` key. For an object-form entry an explicit
`skills` or `prompts` pattern list MUST filter that type of the package (`[]` switches every file of
that type off), and an omitted key MUST fall back, for that type only, to the manifest's entry if
present, else the convention directory. A settings or package skill directory MUST be searched
recursively and include its own root `.md` files, and a prompt directory recursively. A file that a
settings entry, a `!pattern` or `-path` over a default folder, or an object-form package filter switches
off MUST be kept as disabled: it MUST NOT be offered, and it counts as present for removal.

#### 792a12k1 UAT Scenarios

```gherkin
Scenario: 792a12k1a A Pi prompt template
  Given ~/.pi/agent/prompts/review.md exists
  When the user runs engram register-skills
  Then it is scanned as a Pi prompt template named review

Scenario: 792a12k1b A Pi settings skill path
  Given Pi's global settings.json has "skills": ["~/extra-skills"] and ~/extra-skills/fmt/SKILL.md exists
  When the user runs engram register-skills
  Then fmt is scanned as a Pi settings skill

Scenario: 792a12k1c Package skills come from its manifest
  Given settings.json lists "npm:pi-intercom" and the package's package.json has "pi": {"skills": ["./skills"]} with skills/pi-intercom/SKILL.md
  When the user runs engram register-skills
  Then pi-intercom is scanned as a skill of package pi-intercom

Scenario: 792a12k1d A string-form package with a pi key ignores convention directories
  Given settings.json lists "npm:x", x's package.json has "pi": {"extensions": ["./index.ts"]}, and the package also has skills/fmt/SKILL.md
  When the user runs engram register-skills
  Then no skill of package x is scanned

Scenario: 792a12k1e An object-form omitted type falls back per type
  Given settings.json lists {"source": "npm:x", "extensions": []} for the same package
  When the user runs engram register-skills
  Then fmt is scanned from skills/, because the object form falls back to the convention directory for the omitted skills type

Scenario: 792a12k1f An object-form empty skills filter
  Given settings.json lists {"source": "npm:pi-intercom", "skills": []}
  When the user runs engram register-skills
  Then no skill of that package is offered, and its pi-intercom skill is kept as disabled, present for removal

Scenario: 792a12k1g A Pi root markdown skill
  Given ~/.pi/agent/skills/notes.md is a root file
  When the user runs engram register-skills
  Then it is scanned as Pi user skill notes

Scenario: 792a12k1h A configured package with a missing path contributes nothing
  Given Pi's settings.json lists a local package path that does not exist
  When the user runs engram register-skills
  Then no skill is scanned from it and registration does not fail
```

### 792a12l Installed Claude Code plugins' skills and commands are found

Requirement: 792a12l Registration MUST find the skills and commands of each installed Claude Code plugin that is enabled by
its `enabledPlugins` entry or, when the entry is absent, by its `plugin.json` `defaultEnabled` (default
true), whose scope is `user` or is project-scoped to the current repository, and whose `installPath`
exists. Its skills MUST come from `<installPath>/skills/<name>/SKILL.md` plus any `plugin.json` `skills`
paths, and its commands from `<installPath>/commands/**/*.md` or, when `plugin.json` `commands` is a
path or an array, from those paths instead. A declared path that is absolute, leaves the plugin
lexically or through a symlink, or does not exist MUST be refused with a warning and make that plugin
count as not scanned. No other cached version MUST be read.

#### 792a12l1 UAT Scenarios

```gherkin
Scenario: 792a12l1a plugin.json commands replace the commands directory
  Given an enabled plugin's plugin.json has "commands": ["./commands/setup.md"] and its commands/ directory also holds configure.md
  When the user runs engram register-skills
  Then only setup is scanned as that plugin's command

Scenario: 792a12l1b A disabled plugin contributes nothing
  Given a plugin in installed_plugins.json with enabledPlugins set to false
  When the user runs engram register-skills
  Then none of its skills or commands are scanned

Scenario: 792a12l1c A plugin without an enabledPlugins entry and without an install path
  Given a plugin in installed_plugins.json with no enabledPlugins entry and a missing installPath
  When the user runs engram register-skills
  Then none of its skills or commands are scanned

Scenario: 792a12l1d Only the installed version is read
  Given the plugin cache holds several versions of a plugin and installed_plugins.json names one installPath
  When the user runs engram register-skills
  Then only that installPath's skills and commands are scanned
```

### 792a12m Project skills and commands are found

Requirement: 792a12m Inside a git repository, registration MUST find the Claude Code project skills and commands
`.claude/skills/<name>/SKILL.md` and `.claude/commands/**/*.md` in the working directory and each
ancestor up to the repository top-level, ignoring root `.md` files in `.claude/skills`. For a project Pi
trusts it MUST also find, in this precedence order, `.pi/skills/`, `.agents/skills/` in the working
directory and its ancestors up to the top-level, the `.pi/settings.json` skill paths,
`.pi/prompts/*.md`, the `.pi/settings.json` prompt paths and the `.pi/settings.json` packages. A project
is trusted when the nearest saved decision in `~/.pi/agent/trust.json` for the folder or a parent says
so, or, with no saved decision, when the global `defaultProjectTrust` is `always`. Outside a git
repository registration MUST find no project entry.

#### 792a12m1 UAT Scenarios

```gherkin
Scenario: 792a12m1a Agents skills in an ancestor
  Given a trusted project has .agents/skills/lint/SKILL.md at its top-level
  When the user runs engram register-skills from a subdirectory of it
  Then lint is scanned as a project agents skill

Scenario: 792a12m1b Trust falls back to defaultProjectTrust always
  Given the working directory has .pi/skills/foo/SKILL.md, trust.json has no decision for it or any parent, and the global settings.json has "defaultProjectTrust": "always"
  When the user runs engram register-skills
  Then foo is scanned as a Pi project skill

Scenario: 792a12m1c An untrusted Pi project is not scanned
  Given the working directory has .pi/skills/foo/SKILL.md, trust.json has no trusting decision for it or any parent, and defaultProjectTrust is not always
  When the user runs engram register-skills
  Then no Pi project skill is scanned

Scenario: 792a12m1d Claude project sources from a subdirectory
  Given a repository whose top-level has .claude/skills/openspec-propose/SKILL.md and .claude/commands/commit.md
  When the user runs engram register-skills from internal/
  Then both are scanned, exactly as from the top-level

Scenario: 792a12m1e Root markdown in the project skills directory is ignored
  Given a repository's .claude/skills/ holds root files commit.md and engram-go-conventions.md and .claude/commands/commit.md exists
  When the user runs engram register-skills
  Then neither root file is scanned as a skill and .claude/commands/commit.md is scanned as a command

Scenario: 792a12m1f Outside a repository there are no project entries
  Given a working directory outside any git repository with a .claude/skills directory
  When the user runs engram register-skills
  Then no project skill is scanned
```

### 792a12n Sources are read only for detected harnesses, and names come from the file system

Requirement: 792a12n Registration MUST read the Claude Code sources only when `~/.claude` exists and the Pi and `~/.agents`
sources only when `~/.pi` exists. A name MUST be the directory name, file stem or command path, never a
frontmatter field. An entry whose name contains `:` (other than a command's namespace separators) MUST
be skipped with a warning, and an entry whose key is not recognized (792a12b) MUST be skipped with a
warning.

#### 792a12n1 UAT Scenarios

```gherkin
Scenario: 792a12n1a A machine without Pi reads no Pi sources
  Given ~/.claude exists, ~/.pi does not, and ~/.agents/skills/lint/SKILL.md exists
  When the user runs engram register-skills
  Then no Pi or agents entry is scanned

Scenario: 792a12n1b A name with a colon is skipped
  Given ~/.claude/skills/a:b/SKILL.md exists
  When the user runs engram register-skills
  Then it is skipped with a warning and no offer is made for it
```

### 792a12o Every scanned entry has a source-qualified key

Requirement: 792a12o Each scanned entry MUST have a key determined by the source folder it was found in, and no key MUST be
bare: Claude Code user skill `claude:<name>`; Claude Code user command `claude:cmd:<name>`;
claude.ai-synced skill `anthropic-skills:<name>`; Pi user skill `pi:<name>`; `~/.agents/skills` skill
`agents:<name>`; Pi settings skill `pi-settings:<name>`; Pi package skill `pi-pkg:<pkg-id>:<name>`; Pi
prompt template `pi-prompt:<name>` (global folder), `pi-settings:pi-prompt:<name>` or
`pi-pkg:<pkg-id>:pi-prompt:<name>`; plugin skill `<plugin>:<name>`; plugin command
`<plugin>:cmd:<name>`; Claude Code project skill `project:<r>:<name>` and command
`project:<r>:cmd:<name>`; and Pi and agents project entries `project:<r>:pi:<name>`,
`project:<r>:agents:<name>`, `project:<r>:pi-prompt:<name>`, `project:<r>:pi-settings:<name>` and
`project:<r>:pi-pkg:<pkg-id>:<name>`. `<plugin>` is the part of the manifest key before `@`. `<pkg-id>`
is the npm name, the git `<host>/<path>` or the `~`-relative local path. `<r>` is the origin remote's
host followed by its path, lowercased, without userinfo, port, trailing `.git` or `/`, or, without an
origin, `local/` followed by the basename of the parent of the absolute `git rev-parse
--git-common-dir`. A skill that engram installs MUST be keyed like any other entry of the folder it is
found in. An enabled plugin named `claude`, `pi`, `agents`, `project`, `anthropic-skills`,
`pi-settings`, `pi-pkg` or `pi-prompt` MUST be skipped with a warning, but a disabled one MUST
contribute nothing and print nothing. The note slug MUST be `skill-` followed by the key lowercased,
every run of characters outside `[a-z0-9]` replaced by `-`, and leading and trailing `-` trimmed.

#### 792a12o1 UAT Scenarios

```gherkin
Scenario: 792a12o1a A plugin skill's key and slug
  Given the enabled plugin superpowers@claude-plugins-official ships skills/brainstorming/SKILL.md
  When the user runs engram register-skills
  Then its key is superpowers:brainstorming and an accepted registration creates a note whose basename ends in .skill-superpowers-brainstorming.md

Scenario: 792a12o1b A plugin's skill and command never collide
  Given the enabled plugin commit@skills ships skills/commit/SKILL.md and commands/commit.md
  When the user runs engram register-skills
  Then their keys are commit:commit and commit:cmd:commit

Scenario: 792a12o1c A worktree resolves to owner and repository
  Given a linked worktree directory named runbook-vs-skill whose origin remote is ssh://git@github.com/toejough/engram.git
  When the user runs engram register-skills
  Then its project skills' keys begin with project:github.com/toejough/engram: and their slugs with skill-project-github-com-toejough-engram-

Scenario: 792a12o1d Two same-named projects stay distinct
  Given repositories with origins ssh://git@GitHub.com:22/toejough/engram.git and https://gitlab.com/toejough/engram, each with a project skill deploy
  When the user runs engram register-skills
  Then their keys are project:github.com/toejough/engram:deploy and project:gitlab.com/toejough/engram:deploy

Scenario: 792a12o1e Engram-installed skills are keyed by their folder
  Given ~/.claude/skills/route links to ~/.claude/engram/skills/route and ~/.pi/agent/skills/route links to ~/.pi/agent/engram/skills/route
  When the user runs engram register-skills
  Then the keys are claude:route and pi:route and no key is bare

Scenario: 792a12o1f A Claude user command's key
  Given ~/.claude/commands/audit.md is scanned
  When the user runs engram register-skills
  Then its key is claude:cmd:audit and its note slug is skill-claude-cmd-audit

Scenario: 792a12o1g A plugin named claude is reserved
  Given an enabled plugin claude@m is installed
  When the user runs engram register-skills
  Then it is skipped with a warning and no offer is made for any claude: key from it

Scenario: 792a12o1h A disabled reserved-name plugin is silent
  Given a plugin pi@m with enabledPlugins set to false
  When the user runs engram register-skills
  Then no offer is made for it and no warning is printed

Scenario: 792a12o1i A plugin named cmd is an ordinary plugin
  Given an enabled plugin cmd@m ships skills/x/SKILL.md
  When the user runs engram register-skills
  Then its key is cmd:x

Scenario: 792a12o1j A prompt template's key
  Given ~/.pi/agent/prompts/review.md is scanned
  When the user runs engram register-skills
  Then its key is pi-prompt:review
```

### 792a12p Copies of one skill collapse and key conflicts are reported

Requirement: 792a12p Scanned entries resolving to the same file MUST be one entry. Entries with the same key MUST collapse
when byte-identical, and when they differ MUST be reported as a key conflict naming every path, with no
offer for that key, and the run MUST exit with a failure status after handling every other offer. The
same name found at two levels of a project directory chain is such a conflict. A plugin name installed
from more than one marketplace MUST be reported as a plugin conflict, with no offer for any of its keys,
and its scope treated as not scanned. There MUST be no exception for any source, including skills that
engram installs. An entry whose SHA-256 equals that of an entry earlier in source precedence in the same
run, or equals the `skill_hash` of any existing skill note, MUST be an alias: it makes no offer, is
recorded nowhere, and counts as present for its key.

#### 792a12p1 UAT Scenarios

```gherkin
Scenario: 792a12p1a A Pi copy of an engram skill makes no offer
  Given ~/.pi/agent/skills/route and ~/.claude/skills/route are byte-identical and a note keyed claude:route carries that hash
  When the user runs engram register-skills
  Then no offer is made for claude:route or pi:route, and pi:route counts as present

Scenario: 792a12p1b Two Pi skills with one name and different content
  Given Pi recursion finds ~/.pi/agent/skills/a/foo/SKILL.md and ~/.pi/agent/skills/b/foo/SKILL.md with different bytes
  When the user runs engram register-skills
  Then a conflict is reported for pi:foo naming both paths, no offer is made for pi:foo, every other offer is handled, and the exit status is a failure

Scenario: 792a12p1c The same project skill at two levels
  Given a repository with .claude/skills/foo/SKILL.md at its top-level and a different sub/.claude/skills/foo/SKILL.md
  When the user runs engram register-skills from sub/
  Then a conflict is reported for project:<r>:foo naming both paths and no offer is made for it

Scenario: 792a12p1d A plugin name in two marketplaces
  Given plugins tools@alpha and tools@beta both installed and enabled
  When the user runs engram register-skills
  Then a plugin conflict is reported for tools, no offer is made for any tools: key, and no removal of existing tools: notes is offered

Scenario: 792a12p1e Diverged harness copies are separate keys
  Given ~/.claude/skills/route and ~/.pi/agent/skills/route differ in bytes, a note keyed claude:route carries the Claude copy's hash, and no note or decline covers the Pi copy's hash
  When the user runs engram register-skills
  Then no offer is made for claude:route, registering pi:route is offered, and no conflict or warning is reported

Scenario: 792a12p1f Two synced buckets hold the same skill
  Given two synced buckets both hold pdf/SKILL.md with identical bytes
  When the user runs engram register-skills
  Then one candidate anthropic-skills:pdf results and no conflict is reported

Scenario: 792a12p1g A diverged plugin skill of the same name is its own skill
  Given skill-creator@claude-plugins-official's skill-creator and the synced skill-creator differ in bytes
  When the user runs engram register-skills
  Then both skill-creator:skill-creator and anthropic-skills:skill-creator are offered
```

### 792a12q Removal is offered only for successfully read sources

Requirement: 792a12q A source root MUST count as read only when its read succeeded: a read error of any kind, including
not-exist, means not read, never empty. A skill note MUST be eligible for removal only as follows. For
`claude:`, `claude:cmd:`, `pi:`, `agents:` and `pi-prompt:` keys, the one fixed user root of that form
(`~/.claude/skills`, `~/.claude/commands`, `~/.pi/agent/skills`, `~/.agents/skills`,
`~/.pi/agent/prompts`) must have been read, and no other root matters. For `anthropic-skills:`,
`pi-settings:`, `pi-pkg:` and `project:` keys, the specific root that holds the note's recorded
`skill_source` must have been read in this run (the synced bucket whose `manifest.json` parsed, the
settings entry's path, the package root, or the exact project directory). For plugin keys,
`installed_plugins.json` and `settings.json` must both have parsed, the plugin must have no plugin
conflict, and it must be either enabled with its `installPath` read or absent from the manifest. A note
whose `skill_source` lies under no read root, and a runbook note with `skill_hash` but no `skill_key`,
MUST NOT be eligible. Registration MUST offer to remove a skill note only when it is eligible and no
scanned entry, alias or disabled or not, has its key, and MUST make no removal offer when `--skills-dir`
is given.

#### 792a12q1 UAT Scenarios

```gherkin
Scenario: 792a12q1a A project skill note from another directory
  Given a note with key project:github.com/toejough/engram:openspec-propose
  When the user runs engram register-skills from a directory outside that repository
  Then no removal offer is made for it

Scenario: 792a12q1b An update run without a project
  Given engram update runs from a directory that is not in a git repository and project skill notes exist
  When the user runs engram update
  Then no removal offer is made for any project skill note

Scenario: 792a12q1c A disabled plugin keeps its notes
  Given a note with key hookify:writing-rules and hookify installed but disabled
  When the user runs engram register-skills
  Then no removal offer is made for it

Scenario: 792a12q1d A Pi-disabled skill keeps its note
  Given a note with key pi:ping, ~/.pi/agent/skills read, and the global Pi settings.json skills holding "!ping"
  When the user runs engram register-skills
  Then no register, refresh or removal offer is made for it

Scenario: 792a12q1e An uninstalled plugin is offered for removal
  Given a note with key ralph-loop:cmd:help and ralph-loop absent from a readable installed_plugins.json
  When the user runs engram register-skills
  Then removal of that note is offered

Scenario: 792a12q1f An unreadable user skills directory
  Given listing ~/.claude/skills fails and claude: notes exist
  When the user runs engram register-skills
  Then no removal offer is made for them

Scenario: 792a12q1g Another namespace's missing root does not block removal
  Given ~/.agents/skills does not exist, ~/.pi/agent/skills fails with a permission error, ~/.claude/skills is read and lacks c4, and a note keyed claude:c4 exists
  When the user runs engram register-skills
  Then removal of the claude:c4 note is offered

Scenario: 792a12q1h An unkeyed legacy note is never offered for removal
  Given runbook note 1036.2026-09-18.skill-route.md carries skill_hash and no skill_key, and every source root was read
  When the user runs engram register-skills
  Then no removal offer is made for it

Scenario: 792a12q1i A missing synced manifest
  Given ~/.claude/skills/synced/ holds no bucket with a readable manifest.json and anthropic-skills: notes exist
  When the user runs engram register-skills
  Then no removal offer is made for them

Scenario: 792a12q1j A nested-only project skill is safe from the top level
  Given a note keyed project:github.com/toejough/engram:foo was registered from sub/ and its skill_source is under sub/.claude/skills
  When the user runs engram register-skills from the repository top-level
  Then no removal offer is made for it

Scenario: 792a12q1k One failing synced bucket
  Given two synced buckets, bucket A's manifest.json parses, bucket B's does not, and a note's skill_source lies in bucket B
  When the user runs engram register-skills
  Then no removal offer is made for that note

Scenario: 792a12q1l A Pi settings note is tied to its own entry
  Given settings.json has two skills entries, the first unreadable, and a pi-settings: note whose skill_source lies under the first
  When the user runs engram register-skills
  Then no removal offer is made for that note

Scenario: 792a12q1m A package dropped from settings leaves an orphan
  Given a pi-pkg:x: note exists and npm:x is no longer listed in settings.json
  When the user runs engram register-skills
  Then no removal offer is made for it

Scenario: 792a12q1n An alias keeps its note
  Given a note keyed pi:ping and the Pi ping skill now byte-identical to a higher-precedence skill
  When the user runs engram register-skills
  Then no removal offer is made for the pi:ping note

Scenario: 792a12q1o A machine without the harness
  Given ~/.pi does not exist and a note keyed pi:ping exists
  When the user runs engram register-skills
  Then no removal offer is made for it
```

### 792a12r Offers are grouped by scope and answered by key, pattern or selector

Requirement: 792a12r Each offer MUST belong to a scope: `claude-user` for `claude:` skill keys, `claude-cmd` for
`claude:cmd:` keys, `synced`, `pi-user`, `agents-user`, `pi-settings` and `pi-prompt` for those sources,
`pi-pkg:<pkg-id>` for a Pi package, `plugin:<plugin>` for a plugin and `project:<r>` for a project.
Offers MUST be ordered by scope then key, a scope's display label being its selector `@<scope-id>` with
its count. `--accept` and `--decline` MUST accept an exact key, a pattern ending in `*` that matches
keys by prefix, or a selector `@<scope-id>`; for each offer an exact key MUST take precedence, then the
longest matching pattern, then a selector. When prompting interactively, registration MUST ask once for
each scope holding more than one register or refresh offer, listing them and offering accept all,
decline all, review each and skip, where skip, any other answer and end of input record nothing; a
single register or refresh offer, and every removal, MUST get its own `[y/N]` prompt, where `y` or `yes`
accepts, any other answer declines and end of input records nothing. Removal offers MUST be answered
only by an exact key or an individual prompt, never by accept-all, a pattern or a selector. `--dry-run`
MUST list each offer as `would offer: <kind> <key> (<source>)` under its `@<scope-id> (<count>)` header,
naming the note instead of a source for a removal with no recorded source.

#### 792a12r1 UAT Scenarios

```gherkin
Scenario: 792a12r1a Decline a whole plugin
  Given 15 superpowers register offers outstanding
  When the user runs engram register-skills --decline @plugin:superpowers
  Then all 15 current hashes are recorded as declined under their keys and no note is written

Scenario: 792a12r1b The longest pattern wins
  Given superpowers register offers outstanding
  When the user runs engram register-skills --decline 'superpowers:*' --accept 'superpowers:writing-*'
  Then superpowers:writing-plans and superpowers:writing-skills are registered and the other superpowers offers are declined

Scenario: 792a12r1c An exact key beats a pattern
  Given superpowers register offers outstanding
  When the user runs engram register-skills --decline 'superpowers:*' --accept superpowers:brainstorming
  Then superpowers:brainstorming is registered and the other superpowers offers are declined

Scenario: 792a12r1d The Claude user scope is nameable
  Given register offers for claude:c4 and claude:dev
  When the user runs engram register-skills --decline @claude-user
  Then both are recorded as declined

Scenario: 792a12r1e Removals are never bulk-accepted
  Given one register offer and one removal offer outstanding
  When the user runs engram register-skills --accept '*'
  Then the register offer is carried out and the removal offer is only reported

Scenario: 792a12r1f Skip leaves the scope for next time
  Given the user answers skip for a scope at the interactive prompt
  When the user runs engram register-skills
  Then nothing is written or recorded for that scope's offers and the next run offers them again

Scenario: 792a12r1g A scope with several offers is asked once
  Given a terminal and a scope with three register offers
  When the user runs engram register-skills
  Then the three are listed and asked about once, and answering accept all registers them all

Scenario: 792a12r1h End of input at a prompt records nothing
  Given a terminal whose input ends at the first prompt
  When the user runs engram register-skills
  Then no offer is accepted or declined
```

### 792a12s Declines are keyed by skill key under schema version 2

Requirement: 792a12s `skill-registrations.json` MUST map keys to declined hashes under `schema_version: 2`. A file whose
`schema_version` is anything other than 2 (missing, below 2 or above 2), that has an unrecognized field,
or that cannot be decoded MUST be an error, not an empty decline state, and registration MUST write
nothing. A missing file MUST mean no declines.

#### 792a12s1 UAT Scenarios

```gherkin
Scenario: 792a12s1a A version-1 file fails loudly
  Given skill-registrations.json is {"schema_version":1,"declined":{"route":"<h>"}}
  When the user runs engram register-skills
  Then registration reports an error and writes nothing

Scenario: 792a12s1b An unknown schema version fails loudly
  Given skill-registrations.json has schema_version 3
  When the user runs engram register-skills
  Then registration reports an error and writes nothing

Scenario: 792a12s1c A missing file is no declines
  Given no skill-registrations.json
  When the user runs engram register-skills
  Then every scanned skill without a note is offered
```

### 792a12t Update runs registration after deploying the skills

Requirement: 792a12t After the engram-owned root sync completes, `engram update` MUST run skill registration against the
resolved vault over the same default source set that `engram register-skills` scans, resolved by the
same function from the same home and working directory and not from `agent-instructions/skills` of the
source, using its own `--dry-run` flag and no explicit answers. It MUST prompt for each outstanding
offer when stdin is a terminal, and otherwise write nothing and report the outstanding offers. A
registration failure MUST be reported as `skill registration error: <message>` and MUST NOT roll back
the deploy or fail the update.

#### 792a12t1 UAT Scenarios

```gherkin
Scenario: 792a12t1a An interactive update
  Given `engram update` runs in a terminal and a shipped skill has no note
  When the user runs engram update
  Then the skills are deployed to every harness and the user is offered registration of that skill

Scenario: 792a12t1b A non-interactive update
  Given `engram update` runs with stdin not a terminal and offers outstanding
  When the user runs engram update
  Then the skills are deployed, no vault note or decline record is written, and the report names the skills awaiting an answer

Scenario: 792a12t1c A dry run previews the offers too
  Given offers outstanding
  When the user runs engram update --dry-run
  Then the plan lists the registration offers alongside the sync operations and nothing is written

Scenario: 792a12t1d Update and register-skills see the same offers
  Given a completed sync
  When the user runs engram update --dry-run and engram register-skills --dry-run from the same directory against the same home and vault
  Then both list the same registration offers

Scenario: 792a12t1e A registration failure does not roll back the deploy
  Given registration fails after the skills are deployed
  When the user runs engram update
  Then the report holds a skill registration error line and the deployed skills remain
```

### 792a12u Skill identity fields belong to registration

Requirement: 792a12u A runbook note created, adopted or refreshed by registration MUST carry `skill_hash`, `skill_key` and
`skill_source`, and every path that rewrites a note's frontmatter without being registration MUST
preserve all three unchanged: `engram amend` (including `--clear-pending`), `engram resituate`, Luhmann
re-parenting, the rename-and-rewrite of wikilinks, the identity backfill and the vocabulary tag
rewrites. A note captured by `engram learn` outside registration MUST carry none of them, and `engram
learn` MUST NOT accept them as arguments.

#### 792a12u1 UAT Scenarios

```gherkin
Scenario: 792a12u1a Amend preserves the identity fields
  Given a note carrying skill_hash, skill_key and skill_source
  When the user runs engram amend on it
  Then all three are unchanged in the written note

Scenario: 792a12u1b Resituate preserves the identity fields
  Given a skill note
  When the user runs engram resituate on it
  Then all three are unchanged

Scenario: 792a12u1c Reparenting preserves the identity fields
  Given a skill note carrying all three fields
  When Luhmann re-parenting renames it
  Then all three are unchanged in the renamed note

Scenario: 792a12u1d A wikilink rewrite preserves the identity fields
  Given a rename-and-rewrite pass rewriting a wikilink inside a skill note's frontmatter
  When it runs
  Then its skill_hash, skill_key and skill_source are unchanged

Scenario: 792a12u1e The identity backfill preserves the identity fields
  Given a skill note missing repo, user and vault
  When the user runs engram update --backfill-identity
  Then the note is stamped and its three skill fields are unchanged

Scenario: 792a12u1f Vocabulary rewrites preserve the identity fields
  Given a skill note
  When vocabulary assignment, tag rewrites or definition tagging rewrite its tags
  Then its three skill fields are unchanged

Scenario: 792a12u1g Ordinary capture carries none
  Given a note captured by engram learn outside registration
  When it is written
  Then it carries no skill_hash, skill_key or skill_source, and no learn flag sets them
```

### 792a12v Skill notes behave like captured runbooks

Requirement: 792a12v A runbook note mirroring a skill MUST participate in the Luhmann hierarchy, embedding, vocabulary
tagging, retrieval, triggers and the follow-frame of the runbook shim exactly as a captured runbook
does, with its `situation`, `done_when`, `red_flags` and `triggers` authored on the note and edited with
`engram amend`. Until curated, a note registered or refreshed by registration is a pending offer. Its
body preamble MUST name the file it mirrors, which is its home-relative `skill_source`, for a skill,
command or prompt template alike.

#### 792a12v1 UAT Scenarios

```gherkin
Scenario: 792a12v1a A skill note surfaces like a captured one
  Given a non-pending skill note with a trigger
  When the user runs engram query --text containing that trigger
  Then the note surfaces first with trigger provenance, as a captured runbook with the same triggers would

Scenario: 792a12v1b The body preamble marks the source
  Given a skill runbook note
  When the user runs engram show on it
  Then the body begins with a line naming the skill file it mirrors

Scenario: 792a12v1c A non-engram skill's preamble names the real file
  Given a registered note mirroring the user skill ~/.claude/skills/c4/SKILL.md
  When the user runs engram show on it
  Then its preamble names ~/.claude/skills/c4/SKILL.md, not an agent-instructions/skills/ path

Scenario: 792a12v1d An engram-installed skill's preamble names the deployed file
  Given a registered note mirroring claude:route, whose resolved source is ~/.claude/engram/skills/route/SKILL.md
  When the user runs engram show on it
  Then its preamble names ~/.claude/engram/skills/route/SKILL.md, not agent-instructions/skills/route/SKILL.md

Scenario: 792a12v1e A pending skill note is a pending offer
  Given a newly registered skill note with pending: true
  When curation lists pending offers
  Then the note is among them
```

## 792a13 Added: write-memory

Canonical target: `docs/specs/product/write-memory.md` (title `Write Memory - Product Spec`).

`write-memory` is the skill that turns a decision to remember something into a vault note. A parent skill (recall or learn) decides what to capture and why, then hands the worker a structured request; the worker composes the matching `engram learn` command, runs it, checks the result and reports the written note path. The judgment stays with the parent and the mechanics stay with the worker. The shipped procedure is `agent-instructions/skills/write-memory/SKILL.md`; the vault runbook note that mirrors it is registered by the skill-runbook-registration capability (792a12). What each note kind contains and how the binary writes it are note-writing's (792a3); amending a note is note-revision's (792a4); what makes a good capture belongs to learn and recall.

Rationale: the atomic-skills exploration (2026-07-04) evaluated splitting the skills into shared behavioral atoms. It extracted exactly one, the write seam, as a worker invoked as the parent's next whole action, never a mid-procedure reference fetch. It did not extract a read-memory atom, because recall's read, judge and write pipeline is sequentially cohesive. An earlier reference-card form of the same atom was dropped, and its "0/27 mid-procedure dereference" measurement binds nothing, because the instrument was invalid. The worker form's fire rates were measured with a fixed, marker-gated harness: the handoff fired at its seams, no boundary was violated and no spurious fire occurred in six non-handoff trials (`dev/eval/LEDGER.md#write-memory-worker-fire-rates`, 2026-07-04). The skills-to-runbooks conversion of this worker was proposed and then reversed; the skill file is the shipped form (792d57).

### 792a13a The worker runs only on a handoff and never re-judges it

Requirement: 792a13a The write-memory worker MUST run only when a parent skill hands it a write, MUST execute that write without re-judging whether it deserves to be made, and MUST NOT write anything the parent did not hand off.

Rationale: the skill's `description:` names only parent-instructed invocation, so it never competes for autonomous firing. The worker has no trigger of its own and no user's words reach it directly.

#### 792a13a1 UAT Scenarios

```gherkin
Scenario: 792a13a1a A complete handoff is executed
  Given recall or learn hands write-memory a complete handoff of kind feedback
  When the worker runs
  Then it composes the engram learn feedback command from the handed-off fields, runs it and reports the written note path

Scenario: 792a13a1b The parent's decision is not re-litigated
  Given a handoff whose content the worker could consider minor or redundant
  When the worker runs
  Then it still composes and runs the write and does not decline, reword or merge it

Scenario: 792a13a1c No handoff, no fire
  Given a session in which something seems worth remembering but no parent skill has handed anything off
  When the session continues
  Then the worker is not invoked and writes nothing
```

### 792a13b Parent skills invoke the worker by name

Requirement: 792a13b A parent skill (recall or learn) MUST name write-memory as its next action at each write site and invoke it natively as a skill, and MUST NOT rely on `engram query` surfacing it by trigger or similarity.

Rationale: write-memory is reached only from a parent's own text, so a name in that text is the dependable route, as the please skill names route.

#### 792a13b1 UAT Scenarios

```gherkin
Scenario: 792a13b1a A write site names the worker
  Given recall or learn reaches a write site in its instructions
  When the parent proceeds
  Then its next action is to invoke the write-memory skill with the handoff

Scenario: 792a13b1b A session without the skill follows the runbook note
  Given a session that received only the write-memory runbook note over a vault-graph edge
  When a parent reaches a write site
  Then the note is followed through the runbook shim's follow-frame, and the same handoff contract applies
```

### 792a13c The handoff and its missing fields

Requirement: 792a13c The worker MUST accept a handoff of a kind (`fact`, `feedback`, `qa` or `runbook`), the content fields of that kind and a human-readable source, with optional chunk-sources, tags, supersedes, placement (position and target), red flags and triggers. When a required field is missing it MUST ask the in-session parent for it and MUST NOT invent content on the parent's behalf.

#### 792a13c1 UAT Scenarios

```gherkin
Scenario: 792a13c1a A missing field is requested
  Given a feedback handoff with no action field
  When the worker runs
  Then it asks the parent for the action and runs no command until it is supplied

Scenario: 792a13c1b Content is never invented
  Given a fact handoff with a situation, a subject and a predicate but no object
  When the worker runs
  Then it does not supply an object of its own
```

### 792a13d Fact and feedback notes

Requirement: 792a13d The worker MUST compose a feedback handoff as `engram learn feedback` with `--slug`, `--position`, `--source`, `--situation`, `--behavior`, `--impact` and `--action`, and a fact handoff as `engram learn fact` with `--slug`, `--position`, `--source`, `--situation`, `--subject`, `--predicate` and `--object`, each followed by one `--tag` per handed-off tag. The situation MUST be the retrieval-shaped phrase the parent supplied.

#### 792a13d1 UAT Scenarios

```gherkin
Scenario: 792a13d1a Feedback composition
  Given a feedback handoff with situation, behavior, impact, action and source
  When the worker composes the command
  Then it is engram learn feedback --slug <kebab-slug> --position top --source="<source>" --situation="<situation>" --behavior="<behavior>" --impact="<impact>" --action="<action>"

Scenario: 792a13d1b Fact composition
  Given a fact handoff with situation, subject, predicate, object and source
  When the worker composes the command
  Then it is engram learn fact --slug <kebab-slug> --position top --source="<source>" --situation="<situation>" --subject="<subject>" --predicate="<predicate>" --object="<object>"
```

### 792a13e Runbook notes and their red flags

Requirement: 792a13e The worker MUST compose a runbook handoff as `engram learn runbook` with `--slug`, `--position`, `--source`, `--situation`, `--done-when` and `--body`, with one `--red-flag` per handed-off red flag in the handed-off order, and with no `--red-flag` when the handoff carries none.

#### 792a13e1 UAT Scenarios

```gherkin
Scenario: 792a13e1a A runbook handoff is executed
  Given learn hands off a confirmed runbook with slug, situation, done-when, body, source, position and two red flags
  When the worker runs
  Then it runs engram learn runbook with one --red-flag for each, in order, and reports the written note path

Scenario: 792a13e1b No red flags, no flag
  Given a runbook handoff with no red flags
  When the worker composes the command
  Then the command has no --red-flag, and no error or empty flag results
```

### 792a13f Question-and-answer notes

Requirement: 792a13f The worker MUST compose a qa handoff as `engram learn qa` with `--slug`, `--question`, `--answer`, `--certainty` (`high`, `medium` or `low`) and `--source`, with one `--contributors` per handed-off contributor basename, copying the answer as handed off.

#### 792a13f1 UAT Scenarios

```gherkin
Scenario: 792a13f1a QA composition
  Given a qa handoff with slug, question, answer, contributors, certainty and source
  When the worker composes the command
  Then it is engram learn qa with --slug, --question, --answer, --contributors, --certainty and --source, the answer copied rather than re-derived
```

### 792a13g Placement

Requirement: 792a13g The worker MUST pass `--position top` when the handoff carries no placement, MUST pass the handed-off `--position` (`top`, `continuation` or `sibling`) when it carries one, and MUST pass `--target <id>` only when the position is `continuation` or `sibling`, never alongside `top`. Placement applies to fact, feedback and runbook handoffs.

#### 792a13g1 UAT Scenarios

```gherkin
Scenario: 792a13g1a No placement means top
  Given a fact handoff with no position or target
  When the worker composes the command
  Then it has --position top and no --target

Scenario: 792a13g1b A continuation carries its target
  Given a feedback handoff with position continuation and target 12.3
  When the worker composes the command
  Then it has --position continuation --target 12.3

Scenario: 792a13g1c Top never carries a target
  Given a handoff with position top and a stray target
  When the worker composes the command
  Then it has --position top and no --target
```

### 792a13h Free-text flags are written with the value touching the equals sign

Requirement: 792a13h The worker MUST write every free-text flag value (for example `--situation`, `--source`, `--subject`, `--behavior`) as `--flag="value"`, with the quote touching the `=` and no space.

Rationale: a value that begins with `-` is otherwise read as a new flag, and the command fails.

#### 792a13h1 UAT Scenarios

```gherkin
Scenario: 792a13h1a A value beginning with a dash survives
  Given a feedback handoff whose behavior begins with "-"
  When the worker composes the command
  Then the flag is written --behavior="-..." and the command runs without a misread flag
```

### 792a13i Provenance, tags, supersession and contributors

Requirement: 792a13i The worker MUST append one `--chunk-source <source#anchor>` per handed-off chunk ID, one `--tag <family>/<value>` per handed-off tag for fact, feedback and runbook handoffs, exactly as provided, and `--supersedes="<basename>|<type>|<claim>"` for each handed-off supersession (type `updates`, `narrows` or `refutes`). It MUST NOT invent tags and MUST NOT write the `vocab/` namespace.

Rationale: the binary assigns `vocab/` terms itself and writes the `Supersedes:` body line when `--supersedes` is passed. Handed-off categorical tags travel in the same `tags` list as vocabulary terms but are not vocabulary.

#### 792a13i1 UAT Scenarios

```gherkin
Scenario: 792a13i1a Chunk sources become flags
  Given a handoff with chunk-sources s#a1 and s#a2
  When the worker composes the command
  Then the command carries --chunk-source s#a1 --chunk-source s#a2

Scenario: 792a13i1b Tags are passed through exactly
  Given a feedback handoff with tags work-kind/rename and outcome/pass
  When the worker composes the command
  Then it carries --tag work-kind/rename --tag outcome/pass and no tag of the worker's own

Scenario: 792a13i1c A supersession is passed as given
  Given a handoff that supersedes note 12.3-old with type refutes and a claim
  When the worker composes the command
  Then it carries --supersedes="12.3-old|refutes|<claim>"
```

### 792a13j A qa note takes no tags

Requirement: 792a13j When a qa handoff carries tags, the worker MUST drop them and MUST append the exact line `tags dropped: qa takes no tag flags` to its output, even when the output is only the command.

#### 792a13j1 UAT Scenarios

```gherkin
Scenario: 792a13j1a QA tags are dropped and reported
  Given a qa handoff with the tag work-kind/review
  When the worker runs
  Then the command has no --tag and the output includes tags dropped: qa takes no tag flags
```

### 792a13k Kinds are never mixed, and the binary's own notation is not hand-authored

Requirement: 792a13k The worker MUST NOT mix fact flags (`--subject`, `--predicate`, `--object`), feedback flags (`--behavior`, `--impact`, `--action`) and runbook flags (`--done-when`, `--body`, `--red-flag`, `--trigger`) in one command, and MUST NOT hand-author a `vocab/<term>` tag or a `Supersedes:` backlink. An inline `[[basename]]` citation MAY be typed directly into a handed-off `--body` or `--object` when the handoff content calls for citing a note.

#### 792a13k1 UAT Scenarios

```gherkin
Scenario: 792a13k1a One kind's flags per command
  Given a fact handoff
  When the worker composes the command
  Then it has no feedback or runbook flag

Scenario: 792a13k1b A cited note is written inline
  Given a runbook handoff whose body cites [[12.3-old]]
  When the worker composes the command
  Then the --body carries [[12.3-old]] as typed
```

### 792a13l Runbook triggers are passed through

Requirement: 792a13l The worker MUST append one `--trigger="<cue>"` per handed-off trigger, in order, after any `--red-flag`, and MUST emit no `--trigger` when the handoff carries none.

#### 792a13l1 UAT Scenarios

```gherkin
Scenario: 792a13l1a Triggers become flags
  Given a runbook handoff with triggers /please and take this end-to-end
  When the worker composes the command
  Then the command ends with --trigger="/please" --trigger="take this end-to-end", after any --red-flag

Scenario: 792a13l1b No triggers, no flag
  Given a runbook handoff with no triggers
  When the worker composes the command
  Then the command has no --trigger
```

### 792a13m Trigger cues are specific, author-chosen strings

Requirement: 792a13m The worker MUST treat a trigger as a cue word or phrase that the user, or an engram notice, would literally write (a slash form, a multi-word phrase, or a single distinctive word of at least three characters), matched case-insensitively, with whitespace collapsed, and as a whole word at letter and digit edges. When a handed-off single word is common in ordinary prose, the worker MUST ask the parent for a distinctive cue instead of passing it through, unless the handoff states that over-firing is a deliberate, accepted choice, in which case the worker MUST pass it through.

Rationale: a trigger hit only makes the runbook a candidate, and the agent still judges it against the runbook's own `situation`, so over-firing is tolerable and triggers never replace a well-written situation. A text-only skill test (12 fresh runs per arm) found the old skill dropped triggers silently and never questioned a lone common word, while the revised skill passed the trigger scenarios 12 of 12 (`dev/eval/LEDGER.md#write-memory-triggers-contract`, 2026-09-20).

#### 792a13m1 UAT Scenarios

```gherkin
Scenario: 792a13m1a A distinctive bare word is composed without asking
  Given a runbook handoff with triggers curate, /curate and pending offers
  When the worker composes the command
  Then it carries --trigger="curate" --trigger="/curate" --trigger="pending offers" and asks for nothing

Scenario: 792a13m1b A very common bare word is questioned
  Given a runbook handoff with the trigger fix and no statement that over-firing is deliberate
  When the worker runs
  Then it asks the parent for a distinctive cue and does not emit --trigger="fix"

Scenario: 792a13m1c A deliberate over-fire is passed through
  Given a runbook handoff with the trigger please and a statement that over-firing is a deliberate, accepted choice
  When the worker composes the command
  Then it carries --trigger="please"
```

### 792a13n The write is verified and reported

Requirement: 792a13n The worker MUST run the composed command and, when it succeeds, MUST report the printed note path or paths to the parent in one line. When the command fails with a CLI error, the worker MUST read the error, correct exactly the problem it names, and retry at most twice. When the command still fails after the retries, it MUST report the exact command and the CLI error verbatim, and MUST NOT silently skip a handed-off write.

#### 792a13n1 UAT Scenarios

```gherkin
Scenario: 792a13n1a Success is reported in one line
  Given a handoff whose command succeeds
  When the worker finishes
  Then it reports the printed note path to the parent in one line

Scenario: 792a13n1b A named error is corrected and retried
  Given a command that fails because a flag is misspelled
  When the worker reads the error
  Then it fixes that flag and retries, at most twice

Scenario: 792a13n1c Persistent failure is reported verbatim
  Given a command that still fails after two retries
  When the worker stops
  Then it reports the exact command and the CLI error verbatim, and the write is not silently skipped
```

## 792a14 Added: recall

Canonical target: `docs/specs/product/recall.md` (title `Recall - Product Spec`).

`recall` is the skill an agent runs before work that takes more than a single tool call or quick thought. It states the agent's plan, retrieves raw conversation chunks and crystallized vault notes in one ranking, judges what the retrieved items say against the plan, and reports out loud whether memory changed the plan. At its deeper rung it also grows the vault: it refreshes or rewrites the notes that cover a cluster of evidence and writes a note where none does. The retrieval it runs is memory-query's (792a7), and the writes go through learn's worker (792a13) and note revision (792a4); this capability is what the agent does with them. The shipped procedure is `agent-instructions/skills/recall/SKILL.md`. Its steps are numbered in the skill (Step 0 to Step 4) and cited here by those numbers.

### 792a14a The plan is stated before anything is retrieved

Requirement: 792a14a Recall MUST print, before any `engram` call, three blocks in plain prose: the Ask (what is being asked, in the agent's own words, in one or two sentences), the Situation (the concrete context, such as the repository, the branch, the operation underway and what is loaded) and the Plan (the numbered actions the agent would take next absent any guidance from memory).

Rationale: an unstated plan cannot be tested against memory. The synthesis at the end walks this plan item by item, so skipping the step leaves the rest of the procedure with nothing to compare.

#### 792a14a1 UAT Scenarios

```gherkin
Scenario: 792a14a1a The three blocks come first
  Given a user request that needs more than one tool call
  When the agent runs recall
  Then it prints the Ask, the Situation and the Plan before it runs any engram command

Scenario: 792a14a1b The plan is the unguided plan
  Given the agent has not yet queried memory
  When it prints the Plan
  Then the plan lists the actions it would take with no help from memory, as a numbered list
```

### 792a14b The index is swept once per session

Requirement: 792a14b Recall MUST run `engram ingest --auto` before it queries, so that the index holds the latest sessions and document edits, MUST NOT run it again when a sweep already ran earlier in the same session, and MUST continue to query when the sweep fails, saying so.

Rationale: a sweep takes seconds when nothing changed. A re-sweep inside a session indexes only turns already in the agent's context, and a change made elsewhere is picked up by the next sweep, so suspicion of one is not a reason to sweep again. A slightly stale index still beats no retrieval.

#### 792a14b1 UAT Scenarios

```gherkin
Scenario: 792a14b1a The first recall of a session sweeps
  Given no sweep has run in this session
  When the agent runs recall
  Then it runs engram ingest --auto before engram query

Scenario: 792a14b1b A later recall does not sweep again
  Given a sweep already ran earlier in this session
  When the agent runs recall again, even suspecting a change from another terminal
  Then it does not run engram ingest --auto

Scenario: 792a14b1c A failed sweep does not stop recall
  Given engram ingest --auto exits with an error
  When the agent continues
  Then it says the sweep failed and still runs the query
```

### 792a14c Recall has two depths

Requirement: 792a14c Recall MUST run in one of two modes chosen by the caller: `deep`, which MUST be the mode when the caller names none, and `glance`, which MUST be used when the caller names it. Deep MUST run the whole procedure, applying memory to the decision and growing the vault. Glance MUST apply memory to the decision without writing vault knowledge: it MUST use about three query phrases instead of ten, MUST keep the reading of candidates (Step 2.5A), the recency weighting (Step 2.5B), the activation (Step 2.7), the closing synthesis (Step 3) and the re-entry query (Step 3.5), and MUST skip the coverage writes (Step 2.5C) and the persisting of a conclusion (Step 4).

Rationale: glance exists so that recall can fire often at everyday decision points, and deep for weighty or irreversible ones, or when recall should also learn. A real-scale vault (400+ notes) showed glance 2.23 times faster and 46% cheaper per fire than deep, while a tiny vault had shown only 1.2 times, which was an artifact of vault size (`dev/eval/LEDGER.md#glance-cost-realvault`). Glance delivered the convention, recency-supersession and abduction wins at their verified bars (`dev/eval/LEDGER.md#glance-delivers-c3-c4i-c6`). Its activation still bumps the used notes' recency metadata and pulls a used parent note down as a pending copy, which are kept; neither is a knowledge write.

#### 792a14c1 UAT Scenarios

```gherkin
Scenario: 792a14c1a No mode means deep
  Given recall is invoked with no mode word
  When it runs
  Then it follows the deep procedure, with ten phrases and the write side

Scenario: 792a14c1b Glance asks for about three phrases
  Given recall is invoked with the mode glance
  When it phrases its query
  Then it generates about three phrases, not ten

Scenario: 792a14c1c Glance writes no vault knowledge
  Given recall in glance mode surfaces a cluster and reads its candidates
  When it finishes
  Then no note was created or amended by coverage judgment or by persisting a conclusion

Scenario: 792a14c1d Glance still reads, weights, activates and synthesizes
  Given recall in glance mode
  When it runs
  Then it reads the candidates, applies the recency weight, activates the notes it used and closes with the plan synthesis
```

### 792a14d Glance escalates to deep for a recent standard

Requirement: 792a14d Recall in glance mode MUST switch to deep when its decision turns on honoring a standard that surfaced in the recent-activity channel (an item such as "use X going forward" or "the new convention is Y"), and MUST NOT escalate for applying conventions, for recency supersession within the matched set, or for abduction and synthesis.

Rationale: glance reliably surfaces a recent-channel item but does not elevate it to a requirement: it honored such a standard 0 of 5 times where deep honored it 4 of 5 (`dev/eval/LEDGER.md#glance-fails-c5-delivery`, 2026-06-29). Glance is validated as deep-equivalent only for the other three.

#### 792a14d1 UAT Scenarios

```gherkin
Scenario: 792a14d1a A recent convention that the decision turns on
  Given a glance recall surfaces a recent-activity item stating a new convention, and the decision requires applying it
  When the agent reaches the closing synthesis
  Then it has switched to deep first

Scenario: 792a14d1b Ordinary conventions do not escalate
  Given a glance recall surfaces conventions in the matched set, a newer note superseding an older one within it, or evidence that supports an abduction
  When the agent applies them
  Then it stays in glance and completes
```

### 792a14e Ten phrases from ten angles

Requirement: 792a14e Recall in deep mode MUST generate exactly ten short, specific query phrases, one from each of these angles: the situation or setting, the user's intent or goal, the current concrete action, the problem or blocker, a candidate solution or approach, the tooling or technology in play, prior related work, an adjacent technique, a failure mode to avoid and the domain or concept. It MUST NOT pre-filter them beyond dropping obvious dross such as a bare noun, and MUST phrase them by the task and not by fear.

Rationale: the agent cannot know what memory holds before it queries, so breadth is for finding crystallization candidates. "Implementing Claude Code hooks" retrieves the lessons for the task, while "common mistakes when writing hooks" retrieves only warnings. The binary caps each phrase to its top 30 matches. Glance's three phrases are the measured retrieval floor (#661).

#### 792a14e1 UAT Scenarios

```gherkin
Scenario: 792a14e1a Ten phrases, one per angle
  Given a deep recall about adding a flag to a CLI
  When the agent phrases its query
  Then it has exactly ten phrases, one for each of the ten angles

Scenario: 792a14e1b Phrases name the task, not the fear
  Given the angle "failure mode to avoid" for a task about writing hooks
  When the agent phrases it
  Then the phrase names a task or failure as it would occur in work, not a generic warning about hooks
```

### 792a14f One unified query

Requirement: 792a14f Recall MUST run ONE `engram query --lazy-chunks` that carries every phrase as a repeatable `--phrase="<phrase>"`, MUST NOT run one query per phrase, and MUST NOT pass `--vault` or `--chunks-dir`, because the binary resolves the vault and the chunk index itself. It MUST NOT dispatch subagents to synthesize clusters.

Rationale: the query clusters the matched notes and chunks together in one pass, which phrase-by-phrase calls would split. Cluster synthesis runs inline in the recall skill, and a per-cluster subagent was removed.

#### 792a14f1 UAT Scenarios

```gherkin
Scenario: 792a14f1a All phrases in one call
  Given ten phrases
  When the agent queries memory
  Then it runs one engram query with ten --phrase flags and no other query call for them

Scenario: 792a14f1b No vault flags
  Given the agent composes the query
  When it writes the command
  Then the command has neither --vault nor --chunks-dir

Scenario: 792a14f1c Values are attached to their flags
  Given a phrase that begins with a dash
  When the agent writes the flag
  Then it is --phrase="-..." with the quote touching the equals sign
```

### 792a14g The payload is read by kind and by channel

Requirement: 792a14g Recall MUST read the items of the matched channel by kind and MUST treat the recent-activity channel as situational context only. A chunk item is evidence from which the agent extracts the convention, decision or correction it shows, and MUST NOT be quoted wholesale. A fact, feedback or runbook note is a crystallized lesson that the agent applies directly from its inline content. An item with `provenance: explore` is judged with the notes the clusters nominate. Items with `provenance: recent` MUST NOT be used for coverage or synthesis judgment, are the agent's own recent activity and MUST NOT be re-derived or met with surprise. When the matched channel is empty, recall MUST say so in one sentence, skip the coverage step and proceed with its plan, and a non-empty recent block alone MUST NOT count as something surfacing.

#### 792a14g1 UAT Scenarios

```gherkin
Scenario: 792a14g1a Chunks are evidence, not quotations
  Given a matched chunk that records a reviewer's correction
  When the agent uses it
  Then it states the principle the chunk evidences and does not paste the chunk

Scenario: 792a14g1b Notes are applied as they stand
  Given a matched fact note with its content inline
  When the agent judges it
  Then it applies the note without running engram show

Scenario: 792a14g1c Recent items are read first and not judged
  Given a payload with a recent-activity block
  When the agent reads it
  Then it uses the block to re-immerse in recent work and does not count it for coverage

Scenario: 792a14g1d An empty matched set is reported
  Given the matched channel is empty and only recent items are present
  When the agent reads the payload
  Then it says in one sentence that nothing matched, skips the coverage step and proceeds with its plan
```

### 792a14h Chunk content is read before a cluster is judged

Requirement: 792a14h Recall MUST read the content of every chunk member of a cluster with `engram show-chunk <source#anchor>` before stating any relevance or coverage judgment about that cluster, including a cluster whose candidate list is empty. It MUST read a note candidate's content directly from the payload, MUST NOT run `engram show` for it, and MUST NOT judge a chunk from its path, title or anchor alone.

Rationale: under `--lazy-chunks` a chunk item carries no content, so a judgment from its path is a judgment without evidence. A measured miss (#733, the recency-standard axis at about 80% against an original 100%) traced to one trial that made no `show-chunk` call and called a load-bearing chunk unrelated seed content from its title, in a cluster with no note candidates. All four passing trials had read the chunk. A vault note could not fix this, because recall's coverage step consults only notes that match the current task's phrases and no step asks whether the step itself is being run correctly, so the fix belongs in the procedure's own text. Wording in skill text is expected to plateau below about 95% per-trial adherence, so a rare repeat of the miss is not evidence of a regression (vault notes 794 and 198).

#### 792a14h1 UAT Scenarios

```gherkin
Scenario: 792a14h1a A zero-note cluster still gets its chunks read
  Given a cluster whose candidate list is empty
  When the agent processes it
  Then it runs engram show-chunk on every chunk member before stating any judgment about the cluster

Scenario: 792a14h1b A title is not evidence
  Given a chunk member whose path suggests it is unrelated
  When the agent has not fetched its content
  Then it does not call the chunk unrelated or not applicable

Scenario: 792a14h1c A load-bearing chunk beside a distractor
  Given a cluster of two chunks and no note, one carrying a hard-requirement convention topically distant from the task
  When the agent processes the cluster
  Then it reads both chunks' content before judging either relevant or irrelevant

Scenario: 792a14h1d Candidate notes are read from the payload
  Given a cluster with candidate notes carrying inline content
  When the agent reads them
  Then it runs no engram show for them and reads their content before judging coverage
```

### 792a14i A newer claim wins only when it conflicts

Requirement: 792a14i Recall MUST treat evidence as conflicting only when a newer member explicitly negates or reverses an older claim (for example "no longer", "replaced by", "use X not Y", or the same subject and predicate with a different object in a newer item), and when it conflicts MUST let the newer win. When it does not conflict, recall MUST treat older and newer evidence as independently valid and MUST NOT demote a stable convention because it lacks a recent instance.

#### 792a14i1 UAT Scenarios

```gherkin
Scenario: 792a14i1a A reversal favors the newer note
  Given two notes on one convention, the newer saying "use X, not Y"
  When the agent judges the cluster
  Then it takes X as the standard

Scenario: 792a14i1b A stable convention is not demoted
  Given an old convention that no newer item contradicts
  When the agent judges the cluster
  Then it keeps the convention as valid

Scenario: 792a14i1c A candidate that only matches the superseded claim is near
  Given a candidate note that states only the superseded stance
  When the agent judges coverage against the recency-weighted view
  Then it judges the candidate near and not covered
```

### 792a14j Coverage is judged by the agent and written once per cluster

Requirement: 792a14j In deep mode, recall MUST judge, for each cluster and then for the explore items, whether a candidate note covers the cluster's principle, from the content read and not from cosine alone, and MUST act on the outcome. When a candidate states the principle with no material omission against the recency-weighted view, the outcome is covered and recall MUST run `engram amend --target <candidate> --activate --chunk-source <ids>` and MUST NOT rewrite its content. When a candidate addresses the same situation but omits at least one substantive claim the members evidence, the outcome is near and recall MUST run `engram amend` with the chunk sources and re-synthesized content flags from all members, recency-weighted. When no candidate addresses the situation, the outcome is absent and recall MUST hand off to the write-memory skill one note of kind fact or feedback with the cluster's chunk-source IDs and a descriptive source. When the write corrects, narrows or refutes a surfaced note, recall MUST also pass `--supersedes="<basename>|<type>|<claim>"` with type `updates`, `narrows` or `refutes`. Recall MUST write at most one note per cluster, the representative being a note and never a chunk, MUST wait for each write before the next cluster, and MUST NOT hand-author vocabulary tags.

Rationale: coverage is judged by the agent, not by a cosine threshold, because cosine only nominates the within-cluster top-five candidates. This reversed an earlier rule (D7, reversed by recall-v2 DECISION-2). An empty or absent candidate list is a reason to create, not to skip the cluster. A write is blocking because the note created or updated for one cluster may be a candidate for another. Updates use `engram amend` and `engram learn` creates only, so the amend path is the one that keeps both copies of a note in step.

#### 792a14j1 UAT Scenarios

```gherkin
Scenario: 792a14j1a Covered refreshes provenance and recency only
  Given a candidate that states the cluster's principle with no omission
  When the agent judges it covered
  Then it runs engram amend --target <candidate> --activate --chunk-source <new chunk IDs> with no content flags

Scenario: 792a14j1b Near re-synthesizes
  Given a candidate that omits one claim the members evidence
  When the agent judges it near
  Then it runs engram amend with the chunk sources and re-synthesized content flags

Scenario: 792a14j1c Absent creates through the worker
  Given a cluster with no candidate that addresses its situation
  When the agent judges it absent
  Then it hands one fact or feedback note, with the chunk sources and a source string, to the write-memory skill

Scenario: 792a14j1d A correction carries a supersession
  Given a write that corrects a surfaced note
  When the agent writes it
  Then the command or handoff carries a supersedes value with the note, the type and the claim

Scenario: 792a14j1e One note per cluster
  Given a cluster judged absent
  When the agent writes
  Then it writes one note of one kind and not both a fact and a feedback

Scenario: 792a14j1f High cosine does not decide
  Given a candidate at cosine 0.95 or above
  When the agent has not read its content
  Then it does not judge it covered

Scenario: 792a14j1g Writes are sequential
  Given two clusters to write
  When the agent processes them
  Then it waits for the first write to finish before judging the second

Scenario: 792a14j1h Updates never use learn
  Given an existing note that needs new content
  When the agent updates it
  Then it uses engram amend and not engram learn --target
```

### 792a14k A note from the parent vault is activated, never amended

Requirement: 792a14k Recall MUST NOT run `engram amend` on an item with `from_parent: true` or on its pending pulled copy, MUST NOT write a new note that restates it, and MUST activate it when it covers a cluster. This holds even when the user asks to enrich or record against that note: the copy becomes an amend target only after curation accepts it.

#### 792a14k1 UAT Scenarios

```gherkin
Scenario: 792a14k1a A parent note covers a cluster
  Given a from_parent item that states a cluster's principle
  When the agent judges coverage
  Then it activates the note and does not amend it or write a new note restating it

Scenario: 792a14k1b A pending copy is not amended
  Given a pulled copy of a parent note that curation has not yet accepted
  When the user asks the agent to enrich the note that states the rule
  Then the agent curates first and amends the copy only afterward
```

### 792a14l Only the notes that were used are activated

Requirement: 792a14l After processing the clusters, recall MUST run one `engram activate` with a `--note` for each note it actually drew on (a candidate judged covered or near, and any note cited in the closing synthesis), and MUST skip the call when it drew on no notes. It MUST NOT activate every returned note, MUST NOT activate a chunk or a recent-channel item, and MUST activate a used `from_parent` note the same way, by its path. Activation applies in both modes. Recall MUST NOT curate the pulled copies as part of the recall in either mode: it MUST leave them pending, finish the user's request, run the curate skill as a separate step afterward, and say in its reply that it did or that the copies await curation.

Rationale: activation is agent-driven (recall-v2): the binary no longer pre-judges use. A returned but unused note's last-used time goes stale and it fades by recency rank, which is the supersede-by-competition mechanism, so bumping every returned note would defeat it. Regularly useful memory stays fresh and never-retrieved memory decays and loses rank. The "vault holds pending offer(s) awaiting curation" warning that activation may print is expected, not an error.

#### 792a14l1 UAT Scenarios

```gherkin
Scenario: 792a14l1a Used notes are activated
  Given the agent judged one candidate covered and cited another in its synthesis
  When it activates
  Then one engram activate call carries a --note for each and for no other note

Scenario: 792a14l1b Unused returns are not activated
  Given a payload with ten notes of which the agent used two
  When it activates
  Then the other eight are not activated

Scenario: 792a14l1c Chunks and recent items are never activated
  Given a payload with chunks and a recent block
  When the agent activates
  Then no chunk or recent-channel item is among the notes

Scenario: 792a14l1d A used parent note is pulled down and left pending
  Given the agent used a from_parent note
  When it activates the note by its path
  Then a pending local copy appears, the agent does not amend it, and the reply says the copy awaits curation or that curate ran as a separate step

Scenario: 792a14l1e No notes drawn, no activation
  Given the payload was empty or the coverage step was skipped
  When the agent finishes
  Then it runs no engram activate
```

### 792a14m The closing synthesis says whether memory changed the plan

Requirement: 792a14m Recall MUST close with a synthesis shown to the user that opens with one sentence counting what the query surfaced and what was crystallized ("Query surfaced N items (K chunks, M notes); crystallized J lessons."), walks the Step 0 plan in order marking each numbered action confirmed, adjusted (and how), contradicted or silent, leads the load-bearing conventions with "Apply these as requirements:" drawn from lessons and chunk evidence alike, and surfaces contradictions inline where they affect the action. It MUST NOT dump the payload, MUST NOT paste whole chunks or list raw scores or paths, and MUST be one sentence when memory was silent on everything.

#### 792a14m1 UAT Scenarios

```gherkin
Scenario: 792a14m1a The count opens the synthesis
  Given a query that surfaced 12 items, 7 chunks and 5 notes, and one lesson crystallized
  When the agent writes its synthesis
  Then it begins "Query surfaced 12 items (7 chunks, 5 notes); crystallized 1 lessons."

Scenario: 792a14m1b The plan is walked in order
  Given a three-action plan
  When the agent synthesizes
  Then each action appears in order as confirmed, adjusted, contradicted or silent

Scenario: 792a14m1c Conventions are framed as requirements
  Given a load-bearing convention from a note and one from a chunk
  When the agent synthesizes
  Then both are listed under "Apply these as requirements:"

Scenario: 792a14m1d Silence is brief
  Given memory was silent on every plan item
  When the agent synthesizes
  Then the synthesis is one sentence

Scenario: 792a14m1e No payload dump
  Given a long payload
  When the agent synthesizes
  Then the reply contains no re-emitted payload, pasted chunk or list of raw scores or paths
```

### 792a14n A recommendation is reconciled against the asked task

Requirement: 792a14n Before recall states any option, lever or recommendation that the work generated, it MUST reconcile it against the asked task and the decisions already in context, MUST default to the asked task, and MAY deviate only by naming a genuinely new fact and stating that it is reversing direction. It MUST NOT substitute a prerequisite, a better test or a more rigorous foundation as the thing to do first, and MUST NOT treat reasoning already in context as new evidence.

Rationale: substituting a prerequisite feels like diligence but relitigates the settled task. Reconcile against prior outcomes, not mere mentions.

#### 792a14n1 UAT Scenarios

```gherkin
Scenario: 792a14n1a The asked task stays first
  Given a user asked for task T and the agent now sees a more rigorous prerequisite P
  When the agent writes its recommendation
  Then it does T first, and proposes P instead only on a new fact that it names as a reversal

Scenario: 792a14n1b Old reasoning is not new evidence
  Given the agent's own earlier reasoning in context favors P
  When it considers deviating
  Then it does not count that reasoning as the new fact
```

### 792a14o An emergent recommendation is checked against the vault

Requirement: 792a14o Before the synthesis is shown, when it is about to ship a recommendation, lever or approach, named as the thing to do, that does not appear in the Step 0 plan, recall MUST run one more `engram query --lazy-chunks` keyed to the recommendation itself with three phrases (the recommendation in its own words, the recommendation with "rolled back rejected not worth it superseded", and the recommendation with "tried measured outcome"), applying the recency weight to what returns. The synthesis MUST carry one `Re-entry:` line per emergent recommendation directly above the final recommendation statement, in one of three forms: clean (with the best-match basename or "no items"), CLOSED with the prior outcome and "drop", or CLOSED with the prior outcome and "revisit because" named new evidence. A recommendation MUST NOT ship without its line. A CLOSED recommendation marked drop MUST be withdrawn, with the synthesis saying what replaces it. A revisit MUST be valid only with new evidence that did not exist when the note was written, and re-weighing the note's own tradeoffs MUST NOT count. A note this query surfaced and the agent cites MUST be included in the activation, which MUST run again if it already ran. The step MUST run in both modes.

Rationale: this checks the vault for evidence outside the agent's context. The reconcile of 792a14n checks reasoning already in context. The line is the proof that the check ran.

#### 792a14o1 UAT Scenarios

```gherkin
Scenario: 792a14o1a A new recommendation triggers a re-entry query
  Given the synthesis is about to recommend an approach that is not in the Step 0 plan
  When the agent prepares to show it
  Then it first runs an engram query keyed to that approach with three phrases

Scenario: 792a14o1b The line sits above the recommendation
  Given a clean re-entry result
  When the agent writes its reply
  Then "Re-entry: <recommendation> — clean (top hit: <basename or no items>)" is directly above the recommendation statement

Scenario: 792a14o1c A closed recommendation is dropped
  Given the re-entry query returns a note recording that the approach was rolled back
  When the agent writes the Re-entry line
  Then it is CLOSED with the prior outcome and "drop", and the synthesis says what replaces it

Scenario: 792a14o1d A revisit needs new evidence
  Given a closed approach and no evidence newer than the note
  When the agent considers keeping it
  Then it does not use "revisit", because re-weighing the note's tradeoffs is not new evidence

Scenario: 792a14o1e A cited re-entry note is activated
  Given the re-entry query surfaced a note that the agent cites
  When the agent activates
  Then that note is included, in a second activation call if the first already ran

Scenario: 792a14o1f A recommendation without its line violates the step
  Given a reply with a "RECOMMENDATION:" line and no Re-entry line directly above it
  When it is checked
  Then it violates the step
```

### 792a14p A sound conclusion is persisted as a derived note

Requirement: 792a14p In deep mode, when the closing synthesis reaches a sound, non-trivial conclusion that no existing note states, recall MUST hand one synthesis note per conclusion to the write-memory skill as a fact or feedback, in which the note is the conclusion itself and not a restatement of an input. Recall MUST state a deduction as following necessarily and MUST mark an abduction or induction as probable, a best explanation or defeasible. It MUST mark the source as derived (for example `--source="synthesis (abduction) from recalled memory"`) and, when the conclusion corrects, narrows or refutes a surfaced note, include that note's basename, type and claim in the handoff. It MUST NOT persist a hunch, a conclusion it would hedge below probable, or a conclusion that merely re-aggregates one note, and MUST NOT hand-author a vocabulary tag or a supersession backlink. It MAY type an inline `[[full-basename]]` citation of an input note into the body.

Rationale: reasoning that is never written down evaporates; the note records the outcome and grows the web. A non-truth-preserving inference is a hypothesis, not a fact (vault note 69), and persisting only sound conclusions keeps the vault from rotting (vault note 68).

#### 792a14p1 UAT Scenarios

```gherkin
Scenario: 792a14p1a A sound new conclusion is handed off
  Given a synthesis that reaches a conclusion no note states
  When the agent judges it sound and non-trivial
  Then it hands one note to the write-memory skill with a derived source string

Scenario: 792a14p1b An abduction is marked probable
  Given a conclusion reached by abduction
  When the agent writes the note
  Then its text marks the conclusion probable, a best explanation or defeasible

Scenario: 792a14p1c A hunch is not persisted
  Given a conclusion the agent would hedge below probable
  When it considers persisting
  Then it writes no note

Scenario: 792a14p1d A glance recall persists nothing
  Given a glance recall that reaches a conclusion worth keeping
  When it finishes
  Then it writes no note

Scenario: 792a14p1e Citations are typed inline
  Given a synthesis note that cites an input note's claim
  When the agent writes the body
  Then it types [[full-basename]] itself and writes no vocab tag or Supersedes line
```

### 792a14q A cited synthesis is also captured as a question and answer

Requirement: 792a14q When a persisted synthesis body contains at least one `[[full-basename]]` wikilink, recall MUST also hand off to the write-memory skill a note of kind qa carrying the verbatim question, the synthesis conclusion as the answer, a certainty matching the synthesis note's label, as contributors the wikilink basenames found in the body, and the source "recall Step 4, session <date>". It MUST NOT ask the agent to list its inputs freely, and MUST skip the capture when the body has no wikilink.

Rationale: contributors are extracted from wikilinks the agent already wrote. A body with no citation has no provenance to record, so at least one citation is the bar.

#### 792a14q1 UAT Scenarios

```gherkin
Scenario: 792a14q1a A cited synthesis gets a qa note
  Given a persisted synthesis body that cites [[a]] and [[b]]
  When recall finishes persisting
  Then a qa handoff follows with contributors a and b, the question verbatim, the conclusion as the answer and a matching certainty

Scenario: 792a14q1b An uncited synthesis gets none
  Given a persisted synthesis body with no wikilink
  When recall finishes persisting
  Then no qa note is handed off
```

## 792a15 Added: learn

Canonical target: `docs/specs/product/learn.md` (title `Learn - Product Spec`).

`learn` is the skill an agent runs to keep memory true at the end of work, and at the moments a lesson is confirmed. It has two jobs, in order: sweep the chunk index so every conversation and document is searchable, and write a vault note for every explicit lesson of the session. The explicit lessons are user corrections, explicit save-requests, presented conclusions that were later overturned, and approaches confirmed good. It writes nothing else. The sweep is chunk-indexing's (792a5), the notes are written through write-memory (792a13) and note-writing (792a3), and the lessons that subagents return are lessons-contract's (792a16). The shipped procedure is `agent-instructions/skills/learn/SKILL.md`, cited here by the skill's step numbers.

### 792a15a Learn writes only explicit lessons

Requirement: 792a15a Learn MUST NOT write an episode, a session summary or a note for a fact that nobody asked to remember and that is not one of the four kinds of lesson (792a15e), and MUST NOT run `engram transcript` or `engram learn episode`, or write a tier or a decision-record note. A session with no correction, save-request, reversal or confirmed approach MUST end with the sweep and its report alone.

Rationale: raw event memory is automatic: `engram ingest` chunks and embeds session transcripts and markdown without summaries, episode notes or arc detection, so the agent's only writing job is crystallizing explicit lessons. The episode workflow was retired when the chunk layer replaced it.

#### 792a15a1 UAT Scenarios

```gherkin
Scenario: 792a15a1a A session with no lesson writes nothing
  Given a session with no correction, save-request, reversal or confirmed approach
  When the agent runs learn
  Then it runs the sweep, reports its tally and writes no note

Scenario: 792a15a1b No episode or summary is written
  Given a long session
  When the agent runs learn
  Then it writes no episode or session-summary note and runs neither engram transcript nor engram learn episode
```

### 792a15b The sweep comes first

Requirement: 792a15b Learn MUST begin with `engram ingest --auto` and report the one-line tally it prints (or "memory index up to date"), and MUST continue to the lessons when the sweep fails, surfacing the error. It MUST skip the sweep when one already ran earlier in the same session, unless this is the closing learn of a work cycle, which MUST always sweep.

Rationale: an unchanged corpus returns in seconds. The closing learn captures the session's tail for future sessions. A worry that something changed in another terminal is picked up by the next sweep, not by sweeping now on suspicion. Explicit lessons must not be lost to an ingest hiccup.

#### 792a15b1 UAT Scenarios

```gherkin
Scenario: 792a15b1a The sweep runs and is reported
  Given no sweep has run in this session
  When the agent runs learn
  Then it runs engram ingest --auto and reports the tally or "memory index up to date"

Scenario: 792a15b1b A failed sweep does not block the lessons
  Given engram ingest --auto fails
  When the agent continues
  Then it surfaces the error and still crystallizes the session's lessons

Scenario: 792a15b1c An earlier sweep is not repeated
  Given a sweep already ran earlier in this session and this is not the closing learn
  When the agent runs learn
  Then it does not sweep again

Scenario: 792a15b1d The closing learn always sweeps
  Given a sweep already ran earlier in this session and this is the closing learn of a work cycle
  When the agent runs learn
  Then it sweeps again
```

### 792a15c A correction moment gets a single-note capture

Requirement: 792a15c When learn fires at a correction moment in the middle of a task (a review, a failing check, or the user rejecting the approach or naming a different convention), it MUST skip the sweep and the vocabulary check, go straight to crystallizing the one confirmed correction, and hand it to write-memory as always. It MUST NOT run `engram ingest --auto` in the middle of a task.

Rationale: the capture is a focused single note and not the opening or closing of a cycle. A sweep is a cycle-boundary job: it is wasteful in the middle of a task, can block on a large corpus, and losing the capture to a hung sweep is the failure mode. The closing learn still sweeps. The same fast path serves a subagent return (792a16).

#### 792a15c1 UAT Scenarios

```gherkin
Scenario: 792a15c1a A rejected approach is captured at once
  Given a reviewer rejected the agent's approach in the middle of a task
  When learn fires
  Then it writes one note for that correction through write-memory with no sweep and no vocabulary check

Scenario: 792a15c1b No mid-task sweep
  Given learn fires at a correction moment
  When the agent proceeds
  Then it does not run engram ingest --auto
```

### 792a15d The vocabulary check and its autonomous refit

Requirement: 792a15d Learn MUST run `engram vocab stats` before it crystallizes lessons in a full cycle. On `verdict: OK` it MUST do nothing more. On `verdict: REFIT_PENDING (<reason>)` it MUST run the refit itself without deferring to the user: it MUST run `engram vocab refit`; when that prints `vocab refit: no structure found; vocabulary unchanged` it MUST report exactly that and not claim an applied refit; when it prints a JSON payload of naming requests and exits without writing it MUST name every new cluster exactly once from its exemplars, write an answers file that echoes the payload's `fingerprint` verbatim, and re-run `engram vocab refit --names <file>`; when that re-run fails with a stale-names error it MUST discard those requests, re-run the refit from scratch and answer the new ones; and when a refit is applied it MUST report "Vocab refit applied: <version bump>. Triggered by: <reason>." When the stats output carries `qa round-2 gate: READY (...)`, learn MUST tell the user that the round-2 validation is due and which gates to schedule, and MUST NOT run that validation itself.

Rationale: the vault's own write path marks a refit pending when it has grown enough (792a6s) and the learn skill acts on the verdict, so vocabulary drift is repaired without asking. `--dry-run` prints the derivation diff without writing, for a first refit over a long-drifted vault. The round-2 validation needs the user's oversight.

#### 792a15d1 UAT Scenarios

```gherkin
Scenario: 792a15d1a A healthy vocabulary needs nothing
  Given engram vocab stats prints verdict: OK
  When learn reads it
  Then it takes no vocabulary action

Scenario: 792a15d1b A pending refit is run autonomously
  Given engram vocab stats prints verdict: REFIT_PENDING (growth: 52 notes, 20 days)
  When learn reads it
  Then it runs engram vocab refit without asking the user

Scenario: 792a15d1c No structure found is reported as such
  Given engram vocab refit prints "vocab refit: no structure found; vocabulary unchanged"
  When learn reports
  Then it says the refit ran with no structure found and the vocabulary unchanged, and does not claim an applied refit

Scenario: 792a15d1d New clusters are named and the refit re-run
  Given engram vocab refit prints naming requests and a fingerprint
  When learn answers
  Then it names every new cluster exactly once, echoes the fingerprint verbatim in the answers file and re-runs the refit with --names

Scenario: 792a15d1e A stale answer restarts the refit
  Given the re-run fails with a stale-names error
  When learn recovers
  Then it re-runs engram vocab refit from scratch and answers the regenerated requests

Scenario: 792a15d1f An applied refit is reported loudly
  Given a refit is applied
  When learn reports
  Then it states "Vocab refit applied: <version bump>. Triggered by: <reason>."

Scenario: 792a15d1g The QA round-2 gate is reported and not run
  Given the stats output carries qa round-2 gate: READY (20>=20)
  When learn reads it
  Then it tells the user the round-2 validation is due and does not run it
```

### 792a15e The four kinds of explicit lesson

Requirement: 792a15e Learn MUST scan the session for exactly four kinds of moment and MUST hand each to write-memory. A correction (the user corrected the agent's approach or behavior) MUST be handed off as a feedback note carrying the situation, the behavior, its cost as the impact and what to do instead as the action. An explicit save-request ("remember this", "save that", "note for next time", "write this down") MUST be handed off immediately as a fact note carrying the situation, the subject, the predicate and the object. A reversal MUST be handed off as a feedback note whose situation says when the failure mode applies, whose behavior is what the original reasoning did wrong, whose impact is what the reversal cost and whose action is the guard that would have prevented it. A confirmed approach MUST be handed off as a feedback note, or as a runbook note when its shape calls for one (792a15g). Every handoff MUST carry a slug and a source of the form "session <date>, context: <one-line what-was-happening>", and the supersession details when the lesson corrects an existing vault note.

#### 792a15e1 UAT Scenarios

```gherkin
Scenario: 792a15e1a A correction becomes a feedback note
  Given the user said "don't suppress lint warnings, fix the underlying issue"
  When learn crystallizes it
  Then write-memory receives kind feedback with situation, behavior, impact and action

Scenario: 792a15e1b A save-request becomes a fact note at once
  Given the user said "remember that releases are tagged from main"
  When learn runs
  Then write-memory receives kind fact with situation, subject, predicate and object, before anything else is done

Scenario: 792a15e1c A reversal becomes a feedback note about the root cause
  Given a design that was presented and later overturned by a reviewer
  When learn crystallizes it
  Then write-memory receives kind feedback whose behavior is what the original reasoning did wrong and whose action is the guard against it

Scenario: 792a15e1d A confirmed approach becomes a feedback note
  Given a specific approach confirmed by the user or by an observable outcome
  When learn crystallizes it
  Then write-memory receives kind feedback with the situation when it would apply again, what worked, the confirming evidence and a keep-doing action

Scenario: 792a15e1e A correction of an existing note carries a supersession
  Given a lesson that corrects, narrows or refutes an existing vault note
  When learn hands it off
  Then the handoff includes that note's basename, the type and the claim
```

### 792a15f Reversals and repo-doc corrections

Requirement: 792a15f Learn MUST capture a reversal as a vault note even when nobody said the correction aloud (a self-discovered reversal by the agent, a reviewer or an instrument qualifies), and MUST NOT treat a CORRECTION section or postscript in a repository document as that capture. Learn MUST write the note for the reversal's root cause and not a narrative of the flip.

Rationale: a record of a correction is not the capture of its lesson.

#### 792a15f1 UAT Scenarios

```gherkin
Scenario: 792a15f1a A self-discovered reversal is captured
  Given the agent's presented design was overturned by a failing instrument with nobody saying so
  When learn runs
  Then it writes a reversal note for the root cause

Scenario: 792a15f1b A repo-doc correction does not count
  Given the agent added a CORRECTION section to a repository document
  When learn runs
  Then it still writes the vault note for the reversal
```

### 792a15g A confirmed approach needs a bet or specific praise, and takes the kind its shape calls for

Requirement: 792a15g Learn MUST capture a confirmed approach only when either the user explicitly praised or thanked a specific behavior that can be restated as a reusable tactic, or the agent made a genuine guess or was uncertain about a plan, acted on it, and a session-recorded outcome (a test passed, the user confirmed it, an artifact functioned, a blocker cleared) then confirmed it. It MUST NOT capture a bare pleasantry that names no behavior, a routine success with no bet behind it, or an unconfirmed guess. It MUST hand off a reusable, ordered, multi-step procedure for a recurring task as a runbook note (with `done_when` and a body of numbered steps, which may link fact or feedback notes) and a single behavioral tweak with no step structure as a feedback note. When it writes a runbook and the user also says to do that going forward, it MUST NOT additionally write a fact note for a save-request on the same content.

Rationale: a routing by shape was added when the runbook kind was introduced (ADR-0026): a runbook capture is itself the "remember to do it this way" note. The confirmed-approach kind is the positive mirror of a reversal and, in practice, the kind that fires least often, so the skill carries exemplars (a design assumption checked by a targeted code search, a suspected defect confirmed by a reproduction test, a vendor claim verified against evidence), but the bar does not move: a resolved uncertainty or an explicit specific confirmation, never "it worked".

#### 792a15g1 UAT Scenarios

```gherkin
Scenario: 792a15g1a Specific praise is captured
  Given the user said "thanks for attaching the file, it is easier to read from my phone"
  When learn runs
  Then it captures "attach deliverable files instead of inlining them" as a confirmed approach

Scenario: 792a15g1b A bare thanks is not
  Given the user said "thanks!" and named no behavior
  When learn runs
  Then it writes no note

Scenario: 792a15g1c A self-validated bet is captured
  Given the agent was unsure an assumption held, ran a targeted search to check it, and the search confirmed it
  When learn runs
  Then it captures the check-before-you-commit method as a confirmed approach

Scenario: 792a15g1d A routine success is not
  Given work that went well with no uncertainty resolved
  When learn runs
  Then it writes no confirmed-approach note

Scenario: 792a15g1e A procedure becomes a runbook
  Given a confirmed sequence of run tests, tag the release, push the tag and update the changelog
  When learn hands it off
  Then write-memory receives kind runbook with a situation, done_when and numbered steps

Scenario: 792a15g1f A single tweak stays feedback
  Given a confirmed behavior with no step structure
  When learn hands it off
  Then write-memory receives kind feedback

Scenario: 792a15g1g A runbook is not duplicated as a save-request
  Given a confirmed runbook and the user also said "do that going forward"
  When learn runs
  Then it writes the runbook and no additional fact note for the same content
```

### 792a15h Each note is placed by an ordered test

Requirement: 792a15h Before each hand-off, learn MUST decide the note's Luhmann placement, considering only notes written or recalled earlier in the same session, and MUST pass the resulting position and target to write-memory. It MUST apply these tests in order: when the note develops one specific sub-point raised inside an in-session note, the placement is `continuation` with that note's ID as the target; otherwise, when it continues or extends the same overall thought at the same level, the placement is `sibling` with that note's ID as the target; otherwise the placement is `top` with no target. When no in-session candidate exists, it MUST choose `top` and MUST NOT search the rest of the vault for a target.

Rationale: an ID's position encodes where a note sits in the thought tree, which follows Niklas Luhmann's own practice and Sönke Ahrens, *How to Take Smart Notes*. The disposition step had been removed from the skill on 2026-06-12, so for months every capture was `top` and all 447 notes in the vault carried flat integer IDs, leaving the branching the ID scheme was named for unused. Placement is a judgment of content, which the skill makes and the binary does not.

#### 792a15h1 UAT Scenarios

```gherkin
Scenario: 792a15h1a A sub-point is a continuation
  Given the new note develops one specific sub-point raised inside a note written earlier this session
  When learn decides placement
  Then the handoff carries position continuation and that note's ID as the target

Scenario: 792a15h1b The same thought at the same level is a sibling
  Given the new note extends the same overall thought as an in-session note without being a sub-point of it
  When learn decides placement
  Then the handoff carries position sibling and that note's ID as the target

Scenario: 792a15h1c An unrelated note is top-level
  Given the new note relates to no note written or recalled earlier this session
  When learn decides placement
  Then the handoff carries position top and no target

Scenario: 792a15h1d Continuation is checked before sibling
  Given a note that could be read as either a sub-point or a same-level continuation
  When learn decides placement
  Then it applies the continuation test first and takes the first test that matches

Scenario: 792a15h1e The vault is not searched
  Given no note was written or recalled earlier this session
  When learn decides placement
  Then it chooses top without searching the vault
```

### 792a15i A lesson states the general principle

Requirement: 792a15i Learn MUST state the general principle and not the session-specific instance, MUST phrase `situation` as the retrieval handle a future task would be described by (for example "releasing a Go module"), and MUST write one note per distinct principle.

Rationale: capture is generic and actionable, because a future session recalls by situation similarity and not by remembering this one. This is a skill-authoring convention, gated by baselines and evals, and not an architecture decision.

#### 792a15i1 UAT Scenarios

```gherkin
Scenario: 792a15i1a The principle, not the instance
  Given a correction made while fixing one specific test file
  When learn writes the note
  Then the note states the general rule and not the file's narrative

Scenario: 792a15i1b One note per principle
  Given a session with two distinct corrections
  When learn writes the notes
  Then there are two notes, one per principle
```

### 792a15j A write to a parent is reported as done

Requirement: 792a15j With `ENGRAM_PARENT` set, learn MUST treat the write as complete once the note is written locally, because the binary offers it upward without anything for the agent to run. When a write prints `engram: parent unreachable (retry after …); N offer(s) queued`, learn MUST report the note as saved and its offer as queued for the parent, MUST NOT re-run the write, MUST NOT call it failed, and MUST NOT set `ENGRAM_SERVER`.

Rationale: the note is in the local vault and the offer waits in the queue until a later engram command reaches the parent. `ENGRAM_SERVER`'s mode is gone and setting it is a hard error (792a21).

#### 792a15j1 UAT Scenarios

```gherkin
Scenario: 792a15j1a An unreachable parent is not a failure
  Given a write that printed "engram: parent unreachable (retry after 30s); 1 offer(s) queued"
  When learn reports
  Then it says the note is saved and the offer is queued, and does not retry the write

Scenario: 792a15j1b ENGRAM_SERVER is never set
  Given a write whose parent is unreachable
  When learn considers a workaround
  Then it does not set ENGRAM_SERVER
```

### 792a15k An answered question is captured once

Requirement: 792a15k Learn MUST scan the session for a substantive answered question, one whose answer body contains at least one `[[full-basename]]` wikilink or whose answer is a new vault note it crystallized, and for each such question not yet captured MUST hand off to write-memory a note of kind qa with a slug, the verbatim question, the answer body copied, contributor basenames, a certainty and the source "ad-hoc capture, learn session <date>". Contributors MUST come only from the `[[full-basename]]` wikilinks in the written answer, extracted verbatim and never free-listed, and learn MUST NOT pre-validate that they exist; when write-memory reports a contributor rejection, learn MUST surface it. Learn MUST skip a question answered by generic advice or without a wikilink and without a new note. It MUST NOT write a pair that was already written this session (for example by recall's Step 4), and MUST NOT write a pair for a question answered by a Step 2 note written this turn, because that note is the capture. It MUST still write a pair for a question answered by citing an existing earlier note when no new note was written for it. It MUST write one pair per distinct answered question.

Rationale: the traceability bar (a wikilink or a new note) is the observable test of a substantive answer. The self-answered exclusion fixes a production bug found by the write-memory eval (task 3.4b, trial R-0): the old wording treated the agent's own just-written note as a trigger for a second write, producing an unrequested duplicate `engram learn qa` write whenever the agent posed and answered a diagnostic question through its own note in the same turn (`dev/eval/cumulative/runbook_vs_skill/phase2/results/3.4b_write_memory_shim_only_retry2_sonnet5.md`).

#### 792a15k1 UAT Scenarios

```gherkin
Scenario: 792a15k1a A cited answer gets a pair
  Given a question answered with a body that cites [[a]] and [[b]]
  When learn captures it
  Then write-memory receives kind qa with the verbatim question, the copied answer and contributors a and b

Scenario: 792a15k1b Generic advice gets none
  Given a question answered with generic advice, no wikilink and no new note
  When learn scans
  Then no qa pair is written

Scenario: 792a15k1c A pair already written is not written again
  Given recall's Step 4 wrote the pair for this question earlier this session
  When learn scans
  Then it writes no second pair

Scenario: 792a15k1d The agent's own note is the capture
  Given a question posed and answered through a Step 2 note written this turn
  When learn scans
  Then it writes no qa pair for it, whatever wikilinks the note cites

Scenario: 792a15k1e A prior note still earns a pair
  Given a question answered by citing a note that existed before this turn, with no new Step 2 note for it
  When learn scans
  Then it writes the qa pair

Scenario: 792a15k1f A rejected contributor is surfaced
  Given write-memory reports that a contributor was rejected
  When learn receives the report
  Then it surfaces the rejection
```

### 792a15l The re-parent batch mode

Requirement: 792a15l When learn is handed the derive-phase candidate payload of `engram update --reparent-luhmann` (candidates with `note`, `note_excerpt`, `target`, `target_excerpt` and `similarity`, plus an `instruction` and a `fingerprint`) and asked for its answers file, it MUST judge each candidate by the content of its two excerpts and not by the similarity score alone, applying the placement test of 792a15h (sub-point as `continuation`, same thought as `sibling`, otherwise `top`). It MUST write exactly one entry for every distinct `note`, choosing the one `target` and `position` it finds most correct when a note has several candidates. It MUST write the answers as JSON of the shape `{"reparenting": [{"note", "position", "target"}], "fingerprint"}` with the fingerprint echoed verbatim, and tell the user to preview with `engram update --reparent-luhmann --answers <file> --dry-run` and then re-run the command without `--dry-run`. It MUST NOT run either command itself and MUST NOT run this mode as part of an ordinary capture.

Rationale: whether one note is a sub-point of another is a judgment of content that an embedding cannot make: two notes can share vocabulary and address different problems. A `top` verdict still needs a full entry, and there is no option to skip a note.

#### 792a15l1 UAT Scenarios

```gherkin
Scenario: 792a15l1a Every candidate note is answered once
  Given a payload in which one note is a candidate for two targets
  When learn writes the answers file
  Then the file has exactly one entry for that note, with the target it judged most correct

Scenario: 792a15l1b A high score does not decide
  Given a candidate with similarity 0.8 whose excerpts address different problems
  When learn judges it
  Then its entry is position top

Scenario: 792a15l1c The fingerprint is echoed and the next step is the user's
  Given a finished answers file
  When learn reports
  Then the fingerprint is the payload's verbatim, the user is told to run the command with --dry-run first, and learn runs neither command
```
