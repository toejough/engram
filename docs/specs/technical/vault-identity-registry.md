# Vault Identity Registry - Technical Spec

## 792b5a Orientation

- Ownership: Repository component.
- Responsibility: brings a vault into existence on first use and owns its identity: the tracked
  vault ID, the untracked `.engram/` state directory, and the location record that tells a copied,
  moved or cloned vault from the original. It stamps, validates, regenerates and claims the ID,
  checks and reports the location, refuses exchange with a parent that reports the vault's own ID,
  and detects an uncommitted ID file.
- Current realization (informational): `internal/cli/exchangestate.go` (`ensureVault`,
  `repairVaultID`, `stampVaultID`, `checkVaultLocation`, `warnVaultLocation`,
  `regenerateVaultID`, `claimVaultLocation`, `selfParentGuard`, `vaultIDUncommitted`),
  `internal/cli/vaultid.go` (`RunVaultID`), `internal/cli/vault_init.go` (`initializeVault`). The
  parent cache and backoff (`gateParentContact`, `recordParentFailure`, `recordParentSuccess`), the exchange-identifier
  checks (`isExchangeID`, `isExchangeBasename`) and the declined-pull record (`loadDeclined`, `recordDeclinedPull`,
  `declineMatches`) are [792b5c14 The parent cache and the contact backoff](#792b5c14-the-parent-cache-and-the-contact-backoff) to [792b5c16 The declined-pull record](#792b5c16-the-declined-pull-record).
- Qualification: responsibility, the identity contract is stated without naming any command;
  ownership, commands decide when to exchange, while this component decides whether the vault's
  identity allows it; independent verification, every requirement is tested directly over a
  temp-dir or scripted filesystem and an injected random source; stability, the ID file, state
  directory and location record are on-disk contracts that survive refactoring; necessity,
  vault-identity's ([vault-identity](../product/vault-identity.md)) creation, ID and refusal requirements all depend on it.
- Supported capabilities: [vault-identity](../product/vault-identity.md), [parent-sync](../product/parent-sync.md) ([792b5c14 The parent cache and the contact backoff](#792b5c14-the-parent-cache-and-the-contact-backoff) to [792b5c16 The declined-pull record](#792b5c16-the-declined-pull-record)).

Rationale: every environment has its own local vault, created by the first command
that resolves its path, and a stable random vault ID in the tracked `.engram-vault-id`. Exchange
links key on that ID, never on URL or hostname, so DNS or IP churn breaks nothing. The untracked
`.engram/home.json` location record catches a `cp -R` copy or a `git clone` before it exchanges.
Transient exchange state lives in `.engram/`, which ignores itself, so the vault's tracked root
`.gitignore` is never touched. Losing `.engram/`, for example through a fresh clone, pauses
child-side exchange until `--claim` or `--regenerate` runs; links and IDs live in tracked files, so
no note content is lost.

## 792b5c Behavior Requirements

### 792b5c1 First-use ensure

Requirement: [792b5c1 First-use ensure](#792b5c1-first-use-ensure) Given a vault path, the vault-identity-registry MUST, when nothing exists at
the path, write the starter files, create `.engram/`, stamp a vault ID with its location record,
and print exactly one stderr line `engram: created new vault at <path>`. It MUST leave an existing
directory untouched (except the repair in [792b5c2 Half-created vault repair](#792b5c2-half-created-vault-repair)), MUST fail when the path exists but is not a
directory, and MUST fail when the path can't be inspected or the random source fails.

Supports: [792a2e1a A read command creates the vault](../product/vault-identity.md#792a2e1-uat-scenarios), [792a2e1b amend and activate fail only on their own merits](../product/vault-identity.md#792a2e1-uat-scenarios), [792a2e1c A child host gets its own vault](../product/vault-identity.md#792a2e1-uat-scenarios), [792a2e1d A second run creates nothing](../product/vault-identity.md#792a2e1-uat-scenarios), [792a2f1a An existing vault is untouched by read-only use](../product/vault-identity.md#792a2f1-uat-scenarios).

#### 792b5c1a Scenarios

```gherkin
Scenario: 792b5c1a1 A missing vault is created with its ID and state directory
  Given nothing exists at the vault path
  When the vault is ensured with a fixed random source
  Then the starters, a 32-hex ID from that source, .engram/.gitignore holding *, and a location record of exactly the ID and the canonical path exist, and stderr has exactly one creation line

Scenario: 792b5c1a2 An existing vault without an ID is left alone
  Given an existing vault directory with no ID file and no .engram/
  When the vault is ensured
  Then nothing is created or changed and nothing is printed

Scenario: 792b5c1a3 A second ensure creates and prints nothing
  Given a vault that was just created
  When the vault is ensured again
  Then nothing is printed and the ID is unchanged

Scenario: 792b5c1a4 A regular file is not a vault
  Given a regular file at the vault path
  When the vault is ensured
  Then it fails saying the path is not a directory

Scenario: 792b5c1a5 An inspection failure is an error
  Given a vault path whose stat fails with an error other than not-exist
  When the vault is ensured
  Then that error is returned

Scenario: 792b5c1a6 A random-source failure is an error
  Given nothing exists at the vault path and a failing random source
  When the vault is ensured
  Then an error naming the random source is returned
```

### 792b5c2 Half-created vault repair

Requirement: [792b5c2 Half-created vault repair](#792b5c2-half-created-vault-repair) When an existing vault has `.engram/` but no `.engram-vault-id`, the
vault-identity-registry MUST stamp the vault ID and print exactly one stderr line
`engram: stamped the missing vault ID of <path>`.

Supports: [792a2g1a A half-created vault gets its ID](../product/vault-identity.md#792a2g1-uat-scenarios).

Rationale: first-use creation writes `.engram/` before the ID, so `.engram/` marks a vault engram
created, and a stamp that failed part-way is finished by the next run.

#### 792b5c2a Scenarios

```gherkin
Scenario: 792b5c2a1 A half-created vault is repaired
  Given an existing vault with .engram/ and no ID file
  When the vault is ensured
  Then the ID file and location record are written and stderr has exactly one repair line
```

### 792b5c3 Starter files

Requirement: [792b5c3 Starter files](#792b5c3-starter-files) The vault-identity-registry MUST initialize a vault by creating `.obsidian/`
and writing `.obsidian/app.json` (`{}`), a `.gitignore` listing `.luhmann.lock`,
`.obsidian/workspace*` and `.obsidian/cache`, and the starter `README.md`, writing each only when it
is missing, and MUST return any directory or write failure.

Supports: [792a2d1a A fresh vault has no notes](../product/vault-identity.md#792a2d1-uat-scenarios), [792a2e1a A read command creates the vault](../product/vault-identity.md#792a2e1-uat-scenarios).

Rationale: the minimal `.obsidian/app.json` lets Obsidian open the directory as a vault without
engram shipping an Obsidian configuration. The `.gitignore` keeps lock and workspace files out of
git for users who track their vault.

#### 792b5c3a Scenarios

```gherkin
Scenario: 792b5c3a1 The starter set is written
  Given an empty vault directory
  When the vault is initialized
  Then .obsidian/app.json, .gitignore and README.md exist with the starter content

Scenario: 792b5c3a2 Initialization is idempotent
  Given a vault whose starter files were edited
  When the vault is initialized again
  Then no existing file is changed

Scenario: 792b5c3a3 Directory and write failures are returned
  Given a filesystem whose directory creation or file write fails
  When the vault is initialized
  Then an error naming the failed step is returned
```

### 792b5c4 Vault ID stamping

Requirement: [792b5c4 Vault ID stamping](#792b5c4-vault-id-stamping) The vault-identity-registry MUST return the vault's existing ID when
`.engram-vault-id` exists, and otherwise MUST mint 16 bytes from the injected random source as 32
lowercase hex characters, create the file exclusively, re-read it, and use whatever ID the file
holds. Only the process whose exclusive create succeeded writes the location record. An existing
vault is stamped only by `engram serve` at startup, by a command with a parent configured that first
writes exchange state (parent contact, a queued offer or a recorded declined pull), and by
regenerating ([792b5c9 Regenerate](#792b5c9-regenerate)).

Supports: [792a2h1a serve stamps an existing vault](../product/vault-identity.md#792a2h1-uat-scenarios), [792a2h1b The first parent contact stamps an existing vault](../product/vault-identity.md#792a2h1-uat-scenarios), [792a2h1c Concurrent creation converges](../product/vault-identity.md#792a2h1-uat-scenarios).

Rationale: exclusive create followed by a re-read makes concurrent creators converge on one ID. The
random source is injected so tests are deterministic.

#### 792b5c4a Scenarios

```gherkin
Scenario: 792b5c4a1 Concurrent creators converge
  Given two processes stamping the same vault at once
  When both finish
  Then the file holds one ID and both processes returned it

Scenario: 792b5c4a2 The loser adopts the winner's ID
  Given the exclusive create reports the file already exists and its first read is empty
  When the vault ID is stamped
  Then the loser retries the read, returns the winner's ID, and writes no location record

Scenario: 792b5c4a3 The ID comes from the injected random source
  Given a fixed random source
  When a new vault ID is stamped
  Then the ID is the hex encoding of that source's bytes

Scenario: 792b5c4a4 An existing ID is reused
  Given a vault with a valid ID file
  When the vault ID is stamped
  Then the existing ID is returned and the file is unchanged
```

### 792b5c5 Vault ID validation

Requirement: [792b5c5 Vault ID validation](#792b5c5-vault-id-validation) The vault-identity-registry MUST accept the ID file's content, ignoring
trailing whitespace, only when it is 32 lowercase hex characters, and MUST otherwise fail with an
error saying the vault ID is invalid and naming `engram vault-id --regenerate`. A re-read after a
lost race retries an invalid read a bounded number of times before failing.

Supports: [792a2j1a serve starts despite a corrupt ID](../product/vault-identity.md#792a2j1-uat-scenarios), [792a2j1b A merged query warns and still runs](../product/vault-identity.md#792a2j1-uat-scenarios), [792a2j1c vault-id reports the corrupt file](../product/vault-identity.md#792a2j1-uat-scenarios).

#### 792b5c5a Scenarios

```gherkin
Scenario: 792b5c5a1 Invalid content is refused
  Given an ID file holding something other than 32 lowercase hex characters
  When the vault ID is read
  Then it fails with the invalid-vault-ID error naming engram vault-id --regenerate

Scenario: 792b5c5a2 A read failure is returned
  Given an ID file that cannot be read
  When the vault ID is read
  Then the read error is returned
```

### 792b5c6 State directory

Requirement: [792b5c6 State directory](#792b5c6-state-directory) The vault-identity-registry MUST create `<vault>/.engram/` with a `.gitignore`
holding only `*`, writing that `.gitignore` only when it is missing, and MUST NOT write the vault's
root `.gitignore`.

Supports: [792a2k1a State does not dirty the vault's git](../product/vault-identity.md#792a2k1-uat-scenarios), [792a2k1b The root .gitignore is untouched](../product/vault-identity.md#792a2k1-uat-scenarios).

#### 792b5c6a Scenarios

```gherkin
Scenario: 792b5c6a1 The root .gitignore is untouched
  Given an existing vault with its own root .gitignore and no ID
  When its ID is stamped
  Then .engram/.gitignore holds *, the root .gitignore is unchanged, and a second stamp reuses the ID

Scenario: 792b5c6a2 State-directory failures are returned
  Given a filesystem where .engram/ or its .gitignore cannot be created
  When the state directory is ensured
  Then the failure is returned
```

### 792b5c7 Location record and check

Requirement: [792b5c7 Location record and check](#792b5c7-location-record-and-check) The vault-identity-registry MUST record a vault's location in
`.engram/home.json` as exactly its vault ID and its canonical path (symlinks evaluated, made
absolute against the working directory, cleaned), with no hostname. Checking the location MUST
report `no vault id` when there is no ID, `missing` when there is no record, `mismatch` when the
record is unparseable or names another ID or path, and `ok` otherwise.

Supports: [792a2l1a A healthy vault reports its ID](../product/vault-identity.md#792a2l1-uat-scenarios), [792a2l1b A moved vault reports a mismatch with both remedies](../product/vault-identity.md#792a2l1-uat-scenarios), [792a2l1c A vault without an ID is reported without writing](../product/vault-identity.md#792a2l1-uat-scenarios), [792a2p1a A cloned child is caught before it offers](../product/vault-identity.md#792a2p1-uat-scenarios), [792a2p1b A cloned child's merged query still runs](../product/vault-identity.md#792a2p1-uat-scenarios), [792a2p1c serve starts despite a mismatched record](../product/vault-identity.md#792a2p1-uat-scenarios), [792a2p1d A symlinked path is not a false positive](../product/vault-identity.md#792a2p1-uat-scenarios), [792a2p1e A hostname change is not a false positive](../product/vault-identity.md#792a2p1-uat-scenarios).

Rationale: a hostname would give false positives, because DHCP, `.local` renames and containers
change it without the vault moving. Canonicalizing the path keeps a symlinked route to the same
vault from failing.

#### 792b5c7a Scenarios

```gherkin
Scenario: 792b5c7a1 A copy is a mismatch
  Given a vault copied to another path
  When the copy's location is checked
  Then the result is mismatch

Scenario: 792b5c7a2 A missing record is missing
  Given a vault with an ID and no location record
  When its location is checked
  Then the result is missing

Scenario: 792b5c7a3 No ID is reported
  Given a vault with no ID file
  When its location is checked
  Then the result is no vault id

Scenario: 792b5c7a4 A record for another ID is a mismatch
  Given a location record naming a different vault ID
  When the location is checked
  Then the result is mismatch

Scenario: 792b5c7a5 A symlinked path passes
  Given a vault reached through a symlink to its recorded path
  When its location is checked
  Then the result is ok
```

### 792b5c8 Location warning

Requirement: [792b5c8 Location warning](#792b5c8-location-warning) When the location check does not pass, or cannot run, the
vault-identity-registry MUST print exactly one warning naming the vault, the check result, and both
`engram vault-id --regenerate` and `engram vault-id --claim`, and report the failure to the caller;
a passing check prints nothing.

Supports: [792a2p1a A cloned child is caught before it offers](../product/vault-identity.md#792a2p1-uat-scenarios), [792a2p1b A cloned child's merged query still runs](../product/vault-identity.md#792a2p1-uat-scenarios), [792a2p1c serve starts despite a mismatched record](../product/vault-identity.md#792a2p1-uat-scenarios).

#### 792b5c8a Scenarios

```gherkin
Scenario: 792b5c8a1 A failed check warns once with both remedies
  Given a vault whose location check fails
  When the location warning runs
  Then exactly one warning naming both remedies is printed and the failure is reported, while a passing check prints nothing
```

### 792b5c9 Regenerate

Requirement: [792b5c9 Regenerate](#792b5c9-regenerate) Regenerating MUST mint a new vault ID, atomically rewrite `.engram-vault-id`
and the location record with it, and touch no note.

Supports: [792a2m1a Regenerate gives a copy its own identity](../product/vault-identity.md#792a2m1-uat-scenarios).

#### 792b5c9a Scenarios

```gherkin
Scenario: 792b5c9a1 A copy gets its own identity
  Given a copied vault with notes
  When its ID is regenerated
  Then the ID differs from the original's, the location check passes, and no note file changed
```

### 792b5c10 Claim

Requirement: [792b5c10 Claim](#792b5c10-claim) Claiming MUST rewrite only the location record for the vault's current ID and
path, and MUST fail with an error saying the vault has no vault ID when the ID file is absent.

Supports: [792a2n1a Claim accepts a moved vault](../product/vault-identity.md#792a2n1-uat-scenarios), [792a2n1b Claim needs an ID](../product/vault-identity.md#792a2n1-uat-scenarios).

#### 792b5c10a Scenarios

```gherkin
Scenario: 792b5c10a1 A moved vault keeps its ID and passes
  Given a vault moved to a new path
  When its location is claimed
  Then the ID is unchanged and the location check passes

Scenario: 792b5c10a2 Claim without an ID fails
  Given a vault with no ID file
  When its location is claimed
  Then it fails saying the vault has no vault ID
```

### 792b5c11 vault-id report

Requirement: [792b5c11 vault-id report](#792b5c11-vault-id-report) The `vault-id` command MUST refuse `--regenerate` and `--claim` together,
print `vault_id: <id> (regenerated)` or `vault_id: <id> (location claimed)` after those actions, and
with no flag print `vault_id:` (the ID or `none`) and `location:` (the check result), adding one
remedy line naming both commands when the result is `missing` or `mismatch`, without writing
anything.

Supports: [792a2l1a A healthy vault reports its ID](../product/vault-identity.md#792a2l1-uat-scenarios), [792a2l1b A moved vault reports a mismatch with both remedies](../product/vault-identity.md#792a2l1-uat-scenarios), [792a2l1c A vault without an ID is reported without writing](../product/vault-identity.md#792a2l1-uat-scenarios), [792a2o1a Both flags are refused](../product/vault-identity.md#792a2o1-uat-scenarios).

#### 792b5c11a Scenarios

```gherkin
Scenario: 792b5c11a1 The report prints the ID and the check
  Given a vault with an ID at its recorded path
  When the vault-id report runs with no flag
  Then it prints vault_id: with the ID and location: ok, and writes nothing

Scenario: 792b5c11a2 Both flags are refused
  Given both --regenerate and --claim
  When the vault-id command runs
  Then it fails as mutually exclusive and changes nothing
```

### 792b5c12 Self-parent guard

Requirement: [792b5c12 Self-parent guard](#792b5c12-self-parent-guard) Given the local vault ID and a parent's reported vault ID, the
vault-identity-registry MUST report a self-parent, and print exactly one warning naming
`engram vault-id --regenerate`, exactly when the local ID is non-empty and equal to the parent's,
and MUST print nothing otherwise.

Supports: [792a2q1a A copied vault served back to itself](../product/vault-identity.md#792a2q1-uat-scenarios), [792a2q1b A write in a vault that is its own parent queues nothing](../product/vault-identity.md#792a2q1-uat-scenarios).

#### 792b5c12a Scenarios

```gherkin
Scenario: 792b5c12a1 Only an equal, non-empty ID is a self-parent
  Given any local vault ID, possibly empty, and any parent vault ID
  When the self-parent guard runs
  Then it fires with one warning naming engram vault-id --regenerate exactly when the local ID is non-empty and equals the parent's
```

### 792b5c13 Uncommitted ID detection

Requirement: [792b5c13 Uncommitted ID detection](#792b5c13-uncommitted-id-detection) The vault-identity-registry MUST report the vault ID as uncommitted when
`.engram-vault-id` exists in a git work tree and is either not in the index (untracked or ignored)
or staged or modified relative to the last commit, and MUST report it as committed when the file is
absent, the vault is not a git work tree, the file is committed and unchanged, or git status fails.

Supports: [792a2i1a Uncommitted ID is flagged](../product/vault-identity.md#792a2i1-uat-scenarios), [792a2i1b An ignored ID is flagged](../product/vault-identity.md#792a2i1-uat-scenarios), [792a2i1c A committed ID is not flagged](../product/vault-identity.md#792a2i1-uat-scenarios), [792a2i1d A vault outside git is not flagged](../product/vault-identity.md#792a2i1-uat-scenarios).

Rationale: `git status` alone hides an ignored file, so index membership is checked first.

#### 792b5c13a Scenarios

```gherkin
Scenario Outline: 792b5c13a1 Uncommitted detection
  Given a vault where <case>
  When uncommitted detection runs
  Then it reports <result>

  Examples:
    | case                                  | result      |
    | the ID file is untracked              | uncommitted |
    | the ID file is ignored                | uncommitted |
    | the ID file is staged, not committed  | uncommitted |
    | the ID file is committed              | committed   |
    | git status fails                      | committed   |
    | the vault is not a git work tree      | committed   |
    | there is no ID file                   | committed   |

Scenario: 792b5c13a2 The update report carries the notice only when flagged
  Given the uncommitted flag is set or clear
  When the update report is written
  Then it names .engram-vault-id and asks for a commit only when the flag is set
```

### 792b5c14 The parent cache and the contact backoff

Requirement: [792b5c14 The parent cache and the contact backoff](#792b5c14-the-parent-cache-and-the-contact-backoff) The vault-identity-registry MUST keep, per vault, a record of the configured parent in
`<vault>/.engram/parent.json`: the parent's URL, the vault ID it last reported, the count of consecutive failures and the
time before which it is not to be contacted. It MUST ignore a record that names another URL. Recording a failure (a
transport error, a timeout or a 5xx) MUST add one to the count and set that time to now plus 30 seconds doubled for each
earlier failure and never more than 15 minutes. Recording any answer, a 4xx included, MUST set the count to zero, clear
the time and remember the vault ID the parent reported when it is well formed ([792b5c15 Exchange identifiers](#792b5c15-exchange-identifiers)), and MUST write nothing when there
is nothing to reset or learn. Asked whether a command may contact the parent, it MUST allow the contact without a message
when `update` asks, when the record names another URL, when no time is set or when that time has passed, and MUST
otherwise refuse it with exactly one warning that names the retry time and the number of queued offers. A record that
cannot be read or written MUST be reported to its caller as an error.

Supports: [792a21k1b A query inside the backoff window skips the parent](../product/parent-sync.md#792a21k1-uat-scenarios), [792a21s1a Commands inside the backoff window make no request](../product/parent-sync.md#792a21s1-uat-scenarios), [792a21s1b update always tries](../product/parent-sync.md#792a21s1-uat-scenarios), [792a21s1c The delay grows and is capped](../product/parent-sync.md#792a21s1-uat-scenarios), [792a21s1d A success resets the count](../product/parent-sync.md#792a21s1-uat-scenarios), [792a21s1e A different parent URL is not backed off](../product/parent-sync.md#792a21s1-uat-scenarios), [792a21s1f The window ends](../product/parent-sync.md#792a21s1-uat-scenarios).

Rationale: without a backoff every command would wait out the connect timeout while a parent is down, and one offline
host would slow every command the user runs. The record keys on the URL that failed, so reconfiguring the
parent tries the new one at once, and `update` always tries so that the end of an outage is noticed. The remembered vault
ID is what lets a link count only under the parent it was made for ([792a21zj An exchanged note carries vault-qualified parent links](../product/parent-sync.md#792a21zj-an-exchanged-note-carries-vault-qualified-parent-links)).

#### 792b5c14a Scenarios

```gherkin
Scenario: 792b5c14a1 The delay doubles from thirty seconds and is capped
  Given consecutive failure counts
  When the delay is computed for each
  Then it is 30 seconds for the first, doubles with each, never shrinks as the count grows, and never exceeds 15 minutes

Scenario: 792b5c14a2 Failures grow the window and a success resets it
  Given the parent fails twice and then answers
  When each outcome is recorded
  Then the window grows with the failures, and after the answer the next failure is 30 seconds again

Scenario: 792b5c14a3 An idle success writes nothing
  Given no failures recorded and a vault ID already remembered
  When a success is recorded
  Then parent.json is not rewritten

Scenario: 792b5c14a4 Inside the window the contact is refused once
  Given a backoff whose time has not passed
  When a command asks to contact the parent
  Then it is refused with exactly one warning naming the retry time and the queued-offer count

Scenario: 792b5c14a5 After the window the contact is allowed silently
  Given a backoff whose time has passed
  When a command asks to contact the parent
  Then it is allowed and prints nothing

Scenario: 792b5c14a6 update ignores the window
  Given a backoff whose time has not passed
  When update asks to contact the parent
  Then it is allowed

Scenario: 792b5c14a7 Another URL is not backed off
  Given a backoff recorded for one parent URL
  When a command asks to contact a different URL
  Then it is allowed
```

### 792b5c15 Exchange identifiers

Requirement: [792b5c15 Exchange identifiers](#792b5c15-exchange-identifiers) The vault-identity-registry MUST accept a vault ID or a note's `xid` as an exchange identifier only
when it is exactly 32 lowercase hexadecimal characters, and MUST accept a basename another vault hands this one only
when it is a Luhmann basename that contains none of `/`, `\` and `|`.

Supports: [792a21v1b A malformed receipt is not recorded](../product/parent-sync.md#792a21v1-uat-scenarios), [792a21zd1e A malformed envelope is refused](../product/parent-sync.md#792a21zd1-uat-scenarios).

Rationale: the parent is another machine's input. An accepted basename is stored in frontmatter, in the declined-pull
record and in offers, so a separator would become a path and a `|` would corrupt the `note|type|claim` encoding of a
supersession.

#### 792b5c15a Scenarios

```gherkin
Scenario: 792b5c15a1 Only a 32-character lowercase hexadecimal string is an identifier
  Given strings of the wrong length, with uppercase letters or with other characters
  When each is checked as an exchange identifier
  Then only a 32-character lowercase hexadecimal string is accepted

Scenario: 792b5c15a2 A basename with a separator is refused
  Given a Luhmann basename and the same basename containing /, \ or |
  When each is checked as an exchange basename
  Then only the one without a separator is accepted
```
### 792b5c16 The declined-pull record

Requirement: [792b5c16 The declined-pull record](#792b5c16-the-declined-pull-record) The vault-identity-registry MUST keep, in `<vault>/.engram/declined.json`, the parent notes whose
pulled copies were discarded outright, each entry holding the parent's vault ID, the basename and the exchange hash that
was declined. Recording a decline MUST create the state directory first (stamping the vault ID before it exists), MUST add
the entry only when an identical entry is not already there, and MUST write the file atomically. A decline MUST match a
parent note when its vault ID equals the note's vault ID, its basename is one of the names the note answers to (its
basename or an alias) and its hash is not changed from the note's hash, so that an unknown comparison counts as not
changed. A missing file MUST read as empty, and an unreadable or undecodable one MUST be an error.

Supports: [792a21ze1c A declined pull is remembered](../product/parent-sync.md#792a21ze1-uat-scenarios), [792a21ze1d A decline survives a parent rename](../product/parent-sync.md#792a21ze1-uat-scenarios), [792a21ze1g A decline under another parent does not apply](../product/parent-sync.md#792a21ze1-uat-scenarios).

Rationale: a decline keyed by the parent's vault ID never suppresses a note of another parent, naming the aliases keeps a
declined note declined after the parent renames it, and a changed hash is new information, so it pulls again.

#### 792b5c16a Scenarios

```gherkin
Scenario: 792b5c16a1 A decline is recorded once
  Given a declined-pull record
  When the same vault, basename and hash are recorded twice
  Then the file holds one entry, and it is written atomically

Scenario: 792b5c16a2 A decline matches by vault, name and hash
  Given a record of a decline under one parent vault
  When a note with the same vault ID, a listed name and the same hash is looked up, and when the vault ID, the name or the hash differs
  Then only the first matches

Scenario: 792b5c16a3 An unknown hash comparison still matches
  Given a decline whose recorded hash has an older version prefix
  When the note's current hash is looked up
  Then the decline matches, because unknown counts as not changed

Scenario: 792b5c16a4 A missing record is empty and a corrupt one is an error
  Given no declined.json, and a declined.json that is not JSON
  When each is read
  Then the first reads as empty and the second fails
```
