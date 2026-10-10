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
