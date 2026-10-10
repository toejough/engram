# Skill Registration Ledger - Technical Spec

## 792b31a Orientation

- Ownership: Repository component; terminal detection boundary.
- Responsibility: turns the keyed candidates of [skill-source-discovery](./skill-source-discovery.md) and the vault's skill notes into
  offers, remembers declined hashes, answers the offers from flags or a terminal, and carries an accepted
  offer out by writing, refreshing, adopting or removing the skill's runbook note, so that nothing is
  written without an answer.
- Current realization (informational): `internal/cli/skillreg.go`, `skillreg_run.go`,
  `skillreg_accept.go`, `skilloffers.go` and `skillanswers.go` (`RunSkillRegistration`,
  `CompareSkillOffers`, `AnswerSkillOffers`, `RegisterSkill`, `RefreshSkill`, `RemoveSkill`,
  `AdoptSkillNote`, `ReadSkillRegistrations`, `RecordSkillDeclined`) and `registerSkillsTargets` in
  `internal/cli/targets.go`, over the injected terminal gate, vault lock, filesystem and embedder.
- Qualification: responsibility, deciding and recording what to offer and carrying the answer out is stated
  without naming any function; ownership, it owns offers, declines and note writes, while discovery and keys
  belong to [skill-source-discovery](./skill-source-discovery.md) and the note format to [note-document-format](./note-document-format.md); independent verification,
  the comparison, the answers and each action run over an in-memory vault with injected faults and a fake
  terminal; stability, the contract survives a new source or a new prompt style; necessity, it is the only
  writer of skill notes.
- Supported capabilities: [skill-runbook-registration](../product/skill-runbook-registration.md).

Rationale: registration offers and never acts alone, because a skill note mirrors a file the user owns and a
write the user did not choose would be a silent change to their memory. Declines are remembered by hash so
that a refusal lasts until the skill changes, and a removal is offered only when its root was read, since
an unreadable folder looks the same as a deleted skill. Without a terminal nothing is prompted or written, so
an agent that runs `engram update` in a shell is told what is waiting instead of guessing.

## 792b31c Behavior Requirements

### 792b31c1 Skill note lookup

Requirement: [792b31c1 Skill note lookup](#792b31c1-skill-note-lookup) The skill-registration-ledger MUST find the note of a key as the note of type runbook whose basename slug begins `skill-`, that carries a
non-empty `skill_hash` and a recognized `skill_key` equal to the key ([792b30c17 Skill keys](./skill-source-discovery.md#792b30c17-skill-keys)), reading only `skill-`
slugs, and MUST give no note for zero matches and an error naming every match (`errDuplicateSkillNote`)
for more than one. A slug alone, a runbook without `skill_hash`, a note of another type, an unreadable
or unparseable file, and a note with an unrecognized key MUST NOT be a skill note.

Supports: [792a12b1a Note located by skill_key](../product/skill-runbook-registration.md#792a12b1-uat-scenarios), [792a12b1b Note not located by slug](../product/skill-runbook-registration.md#792a12b1-uat-scenarios), [792a12b1c Duplicate notes are an error](../product/skill-runbook-registration.md#792a12b1-uat-scenarios), [792a12b1d Unkeyed legacy notes are ignored until adopted](../product/skill-runbook-registration.md#792a12b1-uat-scenarios), [792a12b1e An unrecognized skill_key is never matched or removed](../product/skill-runbook-registration.md#792a12b1-uat-scenarios), [792a12b1f A skill-prefixed slug without skill_hash is not a skill note](../product/skill-runbook-registration.md#792a12b1-uat-scenarios), [792a12b1g A keyed note with another slug is not found](../product/skill-runbook-registration.md#792a12b1-uat-scenarios).

#### 792b31c1a Scenarios

```gherkin
Scenario: 792b31c1a1 A note is found by its skill key
  Given a runbook note with skill_hash and skill_key superpowers:brainstorming
  When the note of that key is looked up
  Then it is found

Scenario: 792b31c1a2 Only keyed runbook notes with a hash are skill notes
  Given notes of another type, a runbook without skill_hash, an unreadable file, an unrecognized key and a non-skill slug
  When a key is looked up
  Then none of them is found, and zero, one and duplicate matches give none, the note and an error

Scenario: 792b31c1a3 A lookup finds exactly the key's notes
  Given a generated vault of keyed, unkeyed and hash-less notes
  When any key is looked up
  Then exactly its keyed notes are found

Scenario: 792b31c1a4 A keyed note beside an unkeyed one is no duplicate
  Given a keyed note and an unkeyed legacy note with the same hash
  When the key is looked up
  Then the keyed note is found with no duplicate error

Scenario: 792b31c1a5 A slug alone locates nothing
  Given an unkeyed legacy note whose slug names the skill
  When the key is looked up
  Then no note is found

Scenario: 792b31c1a6 A duplicate qualified key is an error
  Given two notes carrying the same qualified key
  When the key is looked up
  Then the error names both
```

### 792b31c2 Offer comparison

Requirement: [792b31c2 Offer comparison](#792b31c2-offer-comparison) The skill-registration-ledger MUST compare the keyed candidates with the vault's skill notes and the declined hashes and produce the
offers: candidates in precedence order ([792b30c16 Source composition and precedence](./skill-source-discovery.md#792b30c16-source-composition-and-precedence)), entries resolving to one file collapsed, entries
sharing a key collapsed when byte-identical and reported as a key conflict naming every path (`engram:
skill key conflict: <key> at <paths>`) otherwise, a plugin conflict reported (`engram: plugin name
conflict: <plugin> in <marketplaces>`), a winner whose hash equals that of an earlier candidate or of
any skill note an alias that makes no offer, and otherwise a register offer when the key has no note and
a refresh offer when the note's hash differs, unless that hash is declined for the key. The offers MUST
be sorted by scope then key and carry the key, scope, source path, kind and hash, and the set MUST NOT
depend on the order the candidates, roots or vault listing arrive in. A duplicate skill note MUST abort
the comparison. It MUST print the warnings and then the conflict lines after every other offer is
handled and return a failure when there is any conflict, and MUST print nothing and succeed otherwise.

Supports: [792a12b1d Unkeyed legacy notes are ignored until adopted](../product/skill-runbook-registration.md#792a12b1-uat-scenarios), [792a12b1e An unrecognized skill_key is never matched or removed](../product/skill-runbook-registration.md#792a12b1-uat-scenarios), [792a12b1f A skill-prefixed slug without skill_hash is not a skill note](../product/skill-runbook-registration.md#792a12b1-uat-scenarios), [792a12b1g A keyed note with another slug is not found](../product/skill-runbook-registration.md#792a12b1-uat-scenarios), [792a12c1a A new skill is offered once](../product/skill-runbook-registration.md#792a12c1-uat-scenarios), [792a12c1b A changed skill is offered again after a decline](../product/skill-runbook-registration.md#792a12c1-uat-scenarios), [792a12c1c A matching hash makes no offer](../product/skill-runbook-registration.md#792a12c1-uat-scenarios), [792a12p1a A Pi copy of an engram skill makes no offer](../product/skill-runbook-registration.md#792a12p1-uat-scenarios), [792a12p1b Two Pi skills with one name and different content](../product/skill-runbook-registration.md#792a12p1-uat-scenarios), [792a12p1c The same project skill at two levels](../product/skill-runbook-registration.md#792a12p1-uat-scenarios), [792a12p1d A plugin name in two marketplaces](../product/skill-runbook-registration.md#792a12p1-uat-scenarios), [792a12p1e Diverged harness copies are separate keys](../product/skill-runbook-registration.md#792a12p1-uat-scenarios), [792a12p1f Two synced buckets hold the same skill](../product/skill-runbook-registration.md#792a12p1-uat-scenarios), [792a12p1g A diverged plugin skill of the same name is its own skill](../product/skill-runbook-registration.md#792a12p1-uat-scenarios), [792a12o1e Engram-installed skills are keyed by their folder](../product/skill-runbook-registration.md#792a12o1-uat-scenarios).

#### 792b31c2a Scenarios

```gherkin
Scenario: 792b31c2a1 Each offer kind
  Given a candidate with no note, a candidate with a stale note, a matching note and a note without a candidate
  When offers are compared
  Then register, refresh, no offer and remove are produced

Scenario: 792b31c2a2 A decline suppresses each kind
  Given a declined hash for a register, a refresh and a removal
  When offers are compared
  Then no offer is made for a declined hash

Scenario: 792b31c2a3 Declining everything then recomputing yields no offers
  Given a decline recorded for every offer
  When offers are compared again
  Then there are none

Scenario: 792b31c2a4 A content change after a decline offers exactly once
  Given a declined skill whose content then changes
  When offers are compared
  Then exactly one offer for the new hash

Scenario: 792b31c2a5 Offers carry key, scope, source and kind in order
  Given several offers
  When they are compared
  Then each carries key, scope, source path and kind, sorted by scope then key, and a removal names its note's source

Scenario: 792b31c2a6 Copies collapse and conflicts are named
  Given same-path entries, same-key identical entries and same-key different entries
  When offers are compared
  Then the first two collapse, the third is a conflict with no offer, for every source

Scenario: 792b31c2a7 An alias never offers
  Given any candidates and notes
  When offers are compared
  Then no register or refresh offer carries a hash some note already holds, and no candidate equal to an earlier one is offered

Scenario: 792b31c2a8 The offer set does not depend on scan order
  Given the candidates, roots, plugin facts and vault listing permuted
  When offers are compared
  Then the offer set is the same

Scenario: 792b31c2a9 Engram's installed skills are ordinary keys
  Given engram's skills reached through ~/.claude/skills and ~/.pi/agent/skills
  When offers are compared
  Then they are keyed by their folder and treated like any other entry

Scenario: 792b31c2a10 Six legacy notes yield six register offers
  Given six unkeyed legacy notes whose skill_hash equals the scanned bytes
  When offers are compared
  Then they are not skill notes and six register offers result

Scenario: 792b31c2a11 Unkeyed notes never change the offer set
  Given any vault to which unkeyed notes (skill_hash, no skill_key) are added or removed
  When offers are compared
  Then the offer set does not change

Scenario: 792b31c2a12 Problems are reported last
  Given warnings and conflict lines
  When they are reported
  Then warnings print first, then conflicts, any conflict is a failure, and with none nothing is printed

Scenario: 792b31c2a13 User and Pi sources compare end to end
  Given a fake home resolved through ResolveSkillSources
  When offers are compared
  Then the user-root and Pi scenarios produce their offers

Scenario: 792b31c2a14 Plugin, synced and Pi settings sources compare end to end
  Given plugins, synced buckets, Pi settings entries and packages resolved
  When offers are compared
  Then their scenarios produce their offers

Scenario: 792b31c2a15 Project sources compare end to end
  Given project sources with conflicts across chain levels and removals scoped to the exact directory
  When offers are compared
  Then their scenarios produce their offers

Scenario: 792b31c2a16 Claude keys parse to Claude forms
  Given keys claude:<n>, claude:cmd:… and a skill named cmd
  When they are parsed
  Then claude:<n> is a user skill (scope claude-user), claude:cmd:… with a command path a user command (scope claude-cmd)

Scenario: 792b31c2a17 Unrecognized keys are never matched or removed
  Given every source root read and a note whose key is unrecognized
  When offers are compared
  Then it is matched to no entry and offered no removal

Scenario: 792b31c2a18 An unrecognized key is never eligible for any state
  Given generated keys, root states, plugin facts and candidates
  When offers are compared
  Then a note with an unrecognized key is never removal-eligible, matched or an alias source
```

### 792b31c3 Removal eligibility

Requirement: [792b31c3 Removal eligibility](#792b31c3-removal-eligibility) The skill-registration-ledger MUST offer to remove a skill note only when no candidate at all holds its key (disabled, collapsed,
conflicted or alias ones included), its key form's own roots were read, and its `skill_hash` is not
declined, and MUST make no removal offer when removals are suppressed (`--skills-dir`). For `claude:`,
`claude:cmd:`, `pi:`, `agents:` and `pi-prompt:` keys the one fixed user root of the form must have been
recorded and every root of that form read. For source-rooted keys (`anthropic-skills:`, `pi-settings:`,
`pi-pkg:` and `project:`) the deepest read root of that form that contains the note's recorded
`skill_source` (expanded against the home as given, matched by resolved path or path) must have been
read and must vouch for the key by one of its key prefixes, a deeper root of another scope hiding a
shallower one, and equal-depth roots all having to be read. For a plugin key the plugin manifests must
have been read, the plugin must not be in conflict, and it must be scanned or absent from the manifest.
A note with no `skill_source`, one under no read root, and one whose key is unrecognized MUST NOT be
eligible. A root that could not be read MUST be given the real location it points into, as far as it
resolves, so that it hides the notes under that location from an outer read root.

Supports: [792a12q1a A project skill note from another directory](../product/skill-runbook-registration.md#792a12q1-uat-scenarios), [792a12q1b An update run without a project](../product/skill-runbook-registration.md#792a12q1-uat-scenarios), [792a12q1c A disabled plugin keeps its notes](../product/skill-runbook-registration.md#792a12q1-uat-scenarios), [792a12q1d A Pi-disabled skill keeps its note](../product/skill-runbook-registration.md#792a12q1-uat-scenarios), [792a12q1e An uninstalled plugin is offered for removal](../product/skill-runbook-registration.md#792a12q1-uat-scenarios), [792a12q1f An unreadable user skills directory](../product/skill-runbook-registration.md#792a12q1-uat-scenarios), [792a12q1g Another namespace's missing root does not block removal](../product/skill-runbook-registration.md#792a12q1-uat-scenarios), [792a12q1h An unkeyed legacy note is never offered for removal](../product/skill-runbook-registration.md#792a12q1-uat-scenarios), [792a12q1i A missing synced manifest](../product/skill-runbook-registration.md#792a12q1-uat-scenarios), [792a12q1j A nested-only project skill is safe from the top level](../product/skill-runbook-registration.md#792a12q1-uat-scenarios), [792a12q1k One failing synced bucket](../product/skill-runbook-registration.md#792a12q1-uat-scenarios), [792a12q1l A Pi settings note is tied to its own entry](../product/skill-runbook-registration.md#792a12q1-uat-scenarios), [792a12q1m A package dropped from settings leaves an orphan](../product/skill-runbook-registration.md#792a12q1-uat-scenarios), [792a12q1n An alias keeps its note](../product/skill-runbook-registration.md#792a12q1-uat-scenarios), [792a12q1o A machine without the harness](../product/skill-runbook-registration.md#792a12q1-uat-scenarios).

#### 792b31c3a Scenarios

```gherkin
Scenario: 792b31c3a1 Plugin notes follow the manifest rule
  Given plugin notes with the manifests read, unread, in conflict, disabled and uninstalled
  When removals are compared
  Then only a note of an enabled-and-read or uninstalled plugin without conflict is eligible

Scenario: 792b31c3a2 Project notes need their exact project directory
  Given project notes and project directories read and unread
  When removals are compared
  Then a note is eligible only when the exact directory holding its source was read and vouches for its key

Scenario: 792b31c3a3 Source-rooted notes need the root holding their source
  Given synced, pi-settings and pi-pkg notes
  When removals are compared
  Then a note is eligible only when the root holding its skill_source was read

Scenario: 792b31c3a4 User-root notes need their own fixed root
  Given user-form notes with aliases, disabled candidates, --skills-dir and declines
  When removals are compared
  Then the fixed root being read decides, aliases and disabled candidates keep their notes, --skills-dir makes none and a declined hash is not offered again

Scenario: 792b31c3a5 Removal only when the own root was read and the key is absent
  Given drawn notes with intended root states
  When removals are compared
  Then a removal is offered exactly when the own root was read and the key is absent

Scenario: 792b31c3a6 Another namespace's missing root does not block removal
  Given ~/.agents/skills missing, ~/.pi/agent/skills denied, ~/.claude/skills read and lacking c4, and a claude:c4 note
  When removals are compared
  Then removal of the claude:c4 note is offered

Scenario: 792b31c3a7 An unkeyed legacy note is never offered for removal
  Given an unkeyed legacy note and every root read
  When removals are compared
  Then none is offered

Scenario: 792b31c3a8 A fixed form follows only its own roots
  Given a single-root form note
  When other roots change state
  Then its eligibility follows only its own roots

Scenario: 792b31c3a9 Eligibility depends only on the note's own namespace
  Given any note
  When another namespace's roots are read or unread
  Then its eligibility does not change

Scenario: 792b31c3a10 Read roots carry their form
  Given every read root
  When sources are resolved
  Then each carries the removal-eligibility form of the notes it can prove absent

Scenario: 792b31c3a11 A failed root keeps its real location
  Given a root that cannot be resolved, or a chain of links ending at a missing path
  When it is resolved best-effort
  Then it gets the real location up to the failing component
```

### 792b31c4 Declines file

Requirement: [792b31c4 Declines file](#792b31c4-declines-file) The skill-registration-ledger MUST read the vault-root `skill-registrations.json`: a missing file MUST be an empty state, any other read
failure MUST be an error (a decline is intent, so an unreadable present file must not mean "nothing
declined"), an unrecognized field or undecodable body MUST be an error, and any `schema_version` other
than 2 (missing, below or above) MUST be the error `skill registrations: unsupported schema version`
naming the file and telling the user to delete it to reset the declines. It MUST record a decline by
setting the key's hash and leaving every other entry untouched, stamping `schema_version: 2` in the
unchanged `{schema_version, declined}` shape, writing atomically, and never writing when the existing
file is unreadable or of another version. A registration run MUST write nothing when the file is not
version 2.

Supports: [792a12s1a A version-1 file fails loudly](../product/skill-runbook-registration.md#792a12s1-uat-scenarios), [792a12s1b An unknown schema version fails loudly](../product/skill-runbook-registration.md#792a12s1-uat-scenarios), [792a12s1c A missing file is no declines](../product/skill-runbook-registration.md#792a12s1-uat-scenarios), [792a12c1a A new skill is offered once](../product/skill-runbook-registration.md#792a12c1-uat-scenarios), [792a12c1b A changed skill is offered again after a decline](../product/skill-runbook-registration.md#792a12c1-uat-scenarios).

#### 792b31c4a Scenarios

```gherkin
Scenario: 792b31c4a1 An absent file is the empty state
  Given no skill-registrations.json
  When it is read
  Then the state is empty

Scenario: 792b31c4a2 Other read errors surface
  Given a present file that cannot be read
  When it is read
  Then the error is returned

Scenario: 792b31c4a3 Unknown fields are rejected
  Given a file with an unrecognized key
  When it is read
  Then it is an error

Scenario: 792b31c4a4 Version 2 is read by key
  Given a version-2 file
  When it is read
  Then its declines are read keyed by skill key

Scenario: 792b31c4a5 Every other version is rejected
  Given any schema_version but 2, present or missing
  When it is read
  Then it is an error and version 2 reads back its declines

Scenario: 792b31c4a6 A decline sets the hash and keeps the rest
  Given existing declines
  When a decline is recorded
  Then the key's hash is set and every other entry is intact

Scenario: 792b31c4a7 The first decline writes the file
  Given no file
  When a decline is recorded
  Then the file is written with the one entry

Scenario: 792b31c4a8 A read failure stops a decline
  Given an unreadable existing file
  When a decline is recorded
  Then nothing is written

Scenario: 792b31c4a9 A write failure is returned
  Given a failing write
  When a decline is recorded
  Then the error is returned

Scenario: 792b31c4a10 An older reader decodes the version-2 file
  Given a recorded version-2 file
  When it is decoded with the old shape
  Then it decodes with the same fields

Scenario: 792b31c4a11 Recording keeps every entry and stamps version 2
  Given any version-2 state
  When a decline is recorded
  Then the file is version 2 and keeps every entry

Scenario: 792b31c4a12 A run on a non-version-2 file writes nothing
  Given a version-1 file
  When registration runs
  Then it errors and writes nothing
```

### 792b31c5 Answers and prompts

Requirement: [792b31c5 Answers and prompts](#792b31c5-answers-and-prompts) The skill-registration-ledger MUST parse `--accept` and `--decline` tokens, refusing a token named in both before anything is acted on
(`register-skills: answer named in both --accept and --decline`), and answer each offer by an exact key
first, then the longest matching `*` pattern, then a matching `@<scope-id>` selector, where a removal is
matched only by its exact key. It MUST answer scope by scope: explicit answers first; then, when
interactive, a scope with more than one remaining register or refresh offer by listing them (each with
its kind, key and source) and asking once ``@<scope> (<n>): [a]ccept all / [d]ecline all / [r]eview each
/ [s]kip for now`` where `a` or `accept all`, `d` or `decline all`, `r` or `review each` choose and
anything else, and end of input, skip and record nothing; a single such offer and every removal by its
own prompt (``Register skill `<key>` (<source>) as a vault runbook? [y/N]``, ``Skill `<key>` (<source>)
changed since its note was last synced. Update the note? [y/N]``, ``Skill `<key>` (<source>) is no
longer found in its source. Remove its runbook note? [y/N]``) where `y` or `yes`, case-insensitively,
accepts, another line declines and end of input records nothing; otherwise the offer stays outstanding.
It MUST then note each token that matched no offer, print the one-line summary of outstanding offers,
and report the problems ([792b31c2 Offer comparison](#792b31c2-offer-comparison)). An accept or decline action that fails MUST stop the run with its
error. `--dry-run` MUST print each scope as ``@<scope> (<n>)`` and each offer as `` would offer: <kind>
<key> (<source>)`` (``note <basename>`` for a removal with no source), then the problems.

Supports: [792a12r1a Decline a whole plugin](../product/skill-runbook-registration.md#792a12r1-uat-scenarios), [792a12r1b The longest pattern wins](../product/skill-runbook-registration.md#792a12r1-uat-scenarios), [792a12r1c An exact key beats a pattern](../product/skill-runbook-registration.md#792a12r1-uat-scenarios), [792a12r1d The Claude user scope is nameable](../product/skill-runbook-registration.md#792a12r1-uat-scenarios), [792a12r1e Removals are never bulk-accepted](../product/skill-runbook-registration.md#792a12r1-uat-scenarios), [792a12r1f Skip leaves the scope for next time](../product/skill-runbook-registration.md#792a12r1-uat-scenarios), [792a12r1g A scope with several offers is asked once](../product/skill-runbook-registration.md#792a12r1-uat-scenarios), [792a12r1h End of input at a prompt records nothing](../product/skill-runbook-registration.md#792a12r1-uat-scenarios), [792a12g1a An agent runs update through a shell](../product/skill-runbook-registration.md#792a12g1-uat-scenarios), [792a12h1a An agent relays the user's answer](../product/skill-runbook-registration.md#792a12h1-uat-scenarios), [792a12h1b A dry run](../product/skill-runbook-registration.md#792a12h1-uat-scenarios), [792a12h1c An answer matching nothing is noted](../product/skill-runbook-registration.md#792a12h1-uat-scenarios), [792a12h1f One answer in both flags is refused](../product/skill-runbook-registration.md#792a12h1-uat-scenarios).

#### 792b31c5a Scenarios

```gherkin
Scenario: 792b31c5a1 The same token in both flags is refused
  Given a token in --accept and --decline
  When answers are parsed
  Then parsing fails, exactly when a token appears in both

Scenario: 792b31c5a2 Exact, then longest pattern, then selector
  Given any offer and any disjoint accept and decline token sets
  When an offer is answered
  Then the exact key wins, then the longest pattern, then a selector

Scenario: 792b31c5a3 A token matching no offer is noted
  Given a token that answers no offer
  When offers are answered
  Then the no-pending-offer line is printed and the run continues

Scenario: 792b31c5a4 The outstanding offers are summarized
  Given offers left unanswered in two scopes, or one
  When offers are answered non-interactively
  Then one line gives the total, each scope with its count and the answering commands, singular for one

Scenario: 792b31c5a5 Accept all accepts every offer of the scope
  Given a scope with several register and refresh offers and the answer a
  When offers are answered
  Then every one is accepted

Scenario: 792b31c5a6 Decline all declines every offer of the scope
  Given the answer d
  When offers are answered
  Then each key is declined and recorded per key

Scenario: 792b31c5a7 Review each prompts per offer
  Given the answer r
  When offers are answered
  Then each offer then gets its own prompt

Scenario: 792b31c5a8 Skip records nothing and moves on
  Given the answer s
  When offers are answered
  Then nothing is recorded for that scope and the next scope is asked

Scenario: 792b31c5a9 End of input at the group prompt records nothing
  Given input ending at the group prompt
  When offers are answered
  Then nothing is recorded

Scenario: 792b31c5a10 A single offer uses the per-offer prompt
  Given a scope with one register offer
  When offers are answered
  Then the per-offer prompt is used with no group question

Scenario: 792b31c5a11 End of input at a per-offer prompt records nothing
  Given input ending at a per-offer prompt
  When offers are answered
  Then no note is written and no decline is recorded

Scenario: 792b31c5a12 Removals prompt individually
  Given a scope with register and removal offers
  When offers are answered interactively
  Then the removals each get their own prompt and are never bulk-accepted

Scenario: 792b31c5a13 Prompts name every source
  Given offers with sources
  When they are prompted
  Then each prompt shows the offer's source path

Scenario: 792b31c5a14 An action error stops the run
  Given an accept or decline action that fails
  When offers are answered
  Then the run stops with that error

Scenario: 792b31c5a15 A dry run lists offers under scope headers
  Given offers in two scopes
  When a dry run previews them
  Then each scope header and each would-offer line is printed

Scenario: 792b31c5a16 A dry run reports conflicts last
  Given a conflict
  When a dry run previews
  Then the conflict is printed after the offers with the failure status

Scenario: 792b31c5a17 The register prompt and a yes answer
  Given an interactive register offer and the answer yes
  When it is prompted
  Then the exact prompt is shown and the note is created

Scenario: 792b31c5a18 The register prompt and a no answer
  Given an interactive register offer and the answer n
  When it is prompted
  Then the hash is recorded as declined

Scenario: 792b31c5a19 The refresh prompt and a yes answer
  Given an interactive refresh offer and the answer y
  When it is prompted
  Then the note's body is replaced

Scenario: 792b31c5a20 The remove prompt and a yes answer
  Given an interactive removal offer and the answer y
  When it is prompted
  Then the note is deleted
```

### 792b31c6 The registration run

Requirement: [792b31c6 The registration run](#792b31c6-the-registration-run) The skill-registration-ledger MUST run registration as one operation for `engram register-skills` and for the update hook: resolve the
default source set ([792b30c16 Source composition and precedence](./skill-source-discovery.md#792b30c16-source-composition-and-precedence)) from the home and the working directory its dependency supplies (failing
when there is none, or when resolution fails), reject `--accept` and `--decline` named in both, read the
declines and, when there is an `--adopt`, compute the offers once as a preflight (a failure of any of
these reads aborting the run before anything is written), run each `--adopt`, then compute the offers
again against the vault listing and the declines, answer them ([792b31c5 Answers and prompts](#792b31c5-answers-and-prompts)), and print the scan and
key warnings on every path. With `--dry-run` it MUST only preview the offers and run no `--adopt`. With
`--skills-dir` it MUST refuse `--accept`, `--decline` and `--adopt` before scanning, scan each directory
as Claude Code user skills keyed `claude:<name>`, offer nothing for an entry without `SKILL.md`, make no
removal offer and write nothing. It MUST NOT change a source file. `--adopt` entries MUST be parsed as
`<key>=<note-ref>` with the key ending at the first `=` and both sides non-empty, a key named twice MUST
be refused, entries MUST be validated all before the first one writes and run in key order, and an entry
MUST be refused, writing nothing, when its key names no scanned skill (a skill that Pi's overrides disable
is kept for removal but is not one), its key's copies conflict, its
note reference is ambiguous, its note is keyed to another skill or claimed by another entry, its key is
registered to another note, its basename has no Luhmann ID and date, or its note does not render;
adopted notes MUST NOT also be offered, and linked notes MUST adopt in any order. An accepted offer
whose source cannot be found, or whose kind is unknown, MUST be an error that writes nothing.

Supports: [792a12h1a An agent relays the user's answer](../product/skill-runbook-registration.md#792a12h1-uat-scenarios), [792a12h1b A dry run](../product/skill-runbook-registration.md#792a12h1-uat-scenarios), [792a12h1d A skills-dir run is a preview](../product/skill-runbook-registration.md#792a12h1-uat-scenarios), [792a12h1e A skills-dir run refuses answers](../product/skill-runbook-registration.md#792a12h1-uat-scenarios), [792a12h1f One answer in both flags is refused](../product/skill-runbook-registration.md#792a12h1-uat-scenarios), [792a12a1a A deployed skill is byte-identical](../product/skill-runbook-registration.md#792a12a1-uat-scenarios), [792a12a1b Command and prompt files are untouched](../product/skill-runbook-registration.md#792a12a1-uat-scenarios), [792a12i1a Adopting a previously promoted note](../product/skill-runbook-registration.md#792a12i1-uat-scenarios), [792a12i1b Adopt keeps an unmodeled key](../product/skill-runbook-registration.md#792a12i1-uat-scenarios), [792a12i1c Adopt refuses an anchored key](../product/skill-runbook-registration.md#792a12i1-uat-scenarios), [792a12i1e Adopt refuses a note that is not a runbook](../product/skill-runbook-registration.md#792a12i1-uat-scenarios), [792a12i1f Adopt refuses a note keyed to another skill](../product/skill-runbook-registration.md#792a12i1-uat-scenarios), [792a12i1g Adopt refuses an unscanned key](../product/skill-runbook-registration.md#792a12i1-uat-scenarios), [792a12i1h Adopt refuses an ambiguous reference](../product/skill-runbook-registration.md#792a12i1-uat-scenarios), [792a12i1i A dry run adopts nothing](../product/skill-runbook-registration.md#792a12i1-uat-scenarios), [792a12i1j Adopt refuses a skill that Pi disables](../product/skill-runbook-registration.md#792a12i1-uat-scenarios), [792a12t1c A dry run previews the offers too](../product/skill-runbook-registration.md#792a12t1-uat-scenarios).

#### 792b31c6a Scenarios

```gherkin
Scenario: 792b31c6a1 The default set is resolved from the working directory
  Given a working directory supplied by the dependency
  When registration runs
  Then sources are resolved from it and the home

Scenario: 792b31c6a2 A resolution failure is returned
  Given a harness probe failing for a reason other than not-exist
  When registration runs
  Then the error is returned

Scenario: 792b31c6a3 A listing failure is returned
  Given a vault listing that fails
  When registration runs
  Then the error is returned and nothing is written

Scenario: 792b31c6a4 A declines read failure is returned
  Given an unreadable skill-registrations.json
  When registration runs
  Then the error is returned and nothing is written

Scenario: 792b31c6a5 A comparison failure is returned
  Given two notes carrying one key
  When registration runs
  Then the duplicate error is returned

Scenario: 792b31c6a6 Scan warnings print on every path
  Given a skill named with a colon
  When registration runs for real and under --dry-run
  Then the skip warning is printed in both

Scenario: 792b31c6a7 An explicit accept acts without prompting
  Given a non-interactive run with --accept
  When registration runs
  Then the offer is carried out with no prompt

Scenario: 792b31c6a8 An explicit decline records without prompting
  Given a non-interactive run with --decline
  When registration runs
  Then the hash is recorded with no prompt

Scenario: 792b31c6a9 A skills-dir entry without SKILL.md is ignored
  Given a --skills-dir entry without SKILL.md
  When registration runs
  Then it produces no offer

Scenario: 792b31c6a10 A skills-dir run is a read-only preview
  Given several --skills-dir directories
  When registration runs
  Then they replace the defaults, the offers are previewed and nothing is written

Scenario: 792b31c6a11 A skills-dir run refuses answers before scanning
  Given --skills-dir with --accept, --decline or --adopt
  When registration runs
  Then each is refused before scanning

Scenario: 792b31c6a12 A skills-dir run keys claude and writes nothing
  Given any preview directory and vault
  When registration runs
  Then every offer is keyed claude:<name> and nothing is written

Scenario: 792b31c6a13 A relative skills-dir needs a working directory
  Given a relative --skills-dir and no resolvable working directory
  When registration runs
  Then it is an error, not a scan

Scenario: 792b31c6a14 Accepting leaves source files byte-identical
  Given accepted offers over fixture sources
  When registration runs
  Then every source file is byte-identical afterward

Scenario: 792b31c6a15 Adopt runs before the offers
  Given an existing promoted note and its skill in the scanned set
  When --adopt runs
  Then the note is adopted and no duplicate register offer is made

Scenario: 792b31c6a16 Adopt takes the key of a scanned source
  Given a key naming a scanned candidate
  When --adopt runs
  Then the note mirrors that candidate

Scenario: 792b31c6a17 Adopting a conflicted key is an error
  Given a key whose copies differ
  When --adopt names it
  Then it errors and writes nothing

Scenario: 792b31c6a18 Adopting an unscanned skill is an error
  Given --adopt naming a key that is not scanned
  When it runs
  Then it errors

Scenario: 792b31c6a19 An ambiguous bare ID writes nothing
  Given a bare Luhmann ID matching more than one note
  When --adopt names it
  Then it is refused up front

Scenario: 792b31c6a20 A key registered elsewhere writes nothing
  Given a later entry adopting a key the vault already registers to another note
  When --adopt runs
  Then no entry writes

Scenario: 792b31c6a21 A batch that re-keys the key's note writes nothing
  Given an entry adopting key K onto note N while K's note is re-keyed by another entry
  When --adopt runs
  Then no entry writes

Scenario: 792b31c6a22 Linked notes adopt in any order
  Given several adopts where one note links to another
  When they run
  Then the result does not depend on the order

Scenario: 792b31c6a23 Multi-line Luhmann values are adopted
  Given notes whose luhmann value is a block scalar
  When --adopt runs
  Then they are adopted

Scenario: 792b31c6a24 One note to two keys is an error
  Given two --adopt entries naming one note
  When they run
  Then it errors and writes nothing

Scenario: 792b31c6a25 An unparseable basename later writes nothing
  Given a later entry whose note basename has no Luhmann ID and date
  When --adopt runs
  Then no entry writes

Scenario: 792b31c6a26 An unrenderable note later writes nothing
  Given a later entry whose frontmatter does not render
  When --adopt runs
  Then no entry writes

Scenario: 792b31c6a27 A repeated or malformed adopt flag is refused
  Given a key named twice, or a value not shaped <key>=<note-ref>
  When the flags are parsed
  Then they are refused before anything is written

Scenario: 792b31c6a28 An accepted offer with no source errors
  Given an accepted offer naming no scanned source
  When it is accepted
  Then it errors and writes nothing

Scenario: 792b31c6a29 An unknown offer kind errors
  Given an offer of an unknown kind
  When it is accepted
  Then it errors and writes nothing

Scenario: 792b31c6a30 Adopting takes a conflict-free qualified key
  Given a qualified key and an unkeyed note
  When --adopt runs
  Then the note is renamed to the key slug and stamped with key and source
```

### 792b31c7 Note actions

Requirement: [792b31c7 Note actions](#792b31c7-note-actions) The skill-registration-ledger MUST carry an accepted offer out on the vault under the vault lock. Registering MUST create a runbook note
through `engram learn`'s write path (a fresh top-level Luhmann ID, the slug derived from the key,
`skill_hash`, `skill_key`, `skill_source`, `pending: true`, the `source:` line, an embedded sidecar and
no `situation`, `done_when`, `triggers` or `red_flags`), whose body is the preamble naming the
home-relative source, a blank line and the file's bytes, with a source path recorded home-relative
against the home as given or its resolved form and kept absolute under no home. Refreshing MUST edit the
note's frontmatter as a YAML node: set the hash, key and source, set `pending: true`, replace the body,
keep every other key with its value, convert CRLF to LF in the same write, rebuild the sidecar, refuse
an anchored edited key and write only frontmatter that decodes. Removing MUST delete the note and its
sidecar, and MUST NOT fail for a missing sidecar. Adopting MUST resolve the reference as `engram amend
--target` does, require a runbook, refuse a note keyed to another skill, refuse a key registered to
another note, rename the note to the key's slug keeping its ID and date (a note already carrying the
slug staying unrenamed), rewrite inbound links and rebuild the sidecars of the notes rewritten, replace
the body, set the three fields, clear any `pending` marker, keep every other key, and render the note
before any rename so that a note that would not render is refused unrenamed. A failure to lock, read,
scan, list, write or embed MUST be returned.

Supports: [792a12d1a Accepted registration](../product/skill-runbook-registration.md#792a12d1-uat-scenarios), [792a12d1b A plugin skill's preamble names its real file](../product/skill-runbook-registration.md#792a12d1-uat-scenarios), [792a12d1c A command is registered from its file](../product/skill-runbook-registration.md#792a12d1-uat-scenarios), [792a12e1a Accepted refresh](../product/skill-runbook-registration.md#792a12e1-uat-scenarios), [792a12e1b Refresh follows a plugin version bump](../product/skill-runbook-registration.md#792a12e1-uat-scenarios), [792a12e1c Refresh keeps an unmodeled key](../product/skill-runbook-registration.md#792a12e1-uat-scenarios), [792a12e1d Refresh converts a CRLF note](../product/skill-runbook-registration.md#792a12e1-uat-scenarios), [792a12e1e Refresh refuses an anchored key](../product/skill-runbook-registration.md#792a12e1-uat-scenarios), [792a12e1f Authored fields survive a refresh](../product/skill-runbook-registration.md#792a12e1-uat-scenarios), [792a12f1a A skill no longer shipped](../product/skill-runbook-registration.md#792a12f1-uat-scenarios), [792a12i1a Adopting a previously promoted note](../product/skill-runbook-registration.md#792a12i1-uat-scenarios), [792a12i1b Adopt keeps an unmodeled key](../product/skill-runbook-registration.md#792a12i1-uat-scenarios), [792a12i1c Adopt refuses an anchored key](../product/skill-runbook-registration.md#792a12i1-uat-scenarios), [792a12i1d Adopt converts a CRLF note](../product/skill-runbook-registration.md#792a12i1-uat-scenarios), [792a12i1e Adopt refuses a note that is not a runbook](../product/skill-runbook-registration.md#792a12i1-uat-scenarios), [792a12i1f Adopt refuses a note keyed to another skill](../product/skill-runbook-registration.md#792a12i1-uat-scenarios), [792a12i1g Adopt refuses an unscanned key](../product/skill-runbook-registration.md#792a12i1-uat-scenarios), [792a12i1h Adopt refuses an ambiguous reference](../product/skill-runbook-registration.md#792a12i1-uat-scenarios), [792a12v1c A non-engram skill's preamble names the real file](../product/skill-runbook-registration.md#792a12v1-uat-scenarios), [792a12v1d An engram-installed skill's preamble names the deployed file](../product/skill-runbook-registration.md#792a12v1-uat-scenarios).

#### 792b31c7a Scenarios

```gherkin
Scenario: 792b31c7a1 Registering creates a pending note with hash and preamble
  Given an accepted registration of curate
  When it is registered
  Then the note has the hash, the preamble and pending: true

Scenario: 792b31c7a2 Registering sets no runbook fields
  Given an accepted registration
  When it is registered
  Then the note has no situation, done_when, triggers or red_flags

Scenario: 792b31c7a3 An engram-installed skill's preamble names the deployed file
  Given the six engram skills behind the real notes
  When they are registered
  Then each preamble names ~/.claude/engram/skills/<name>/SKILL.md

Scenario: 792b31c7a4 Other sources name their path and key slug
  Given a plugin skill, a project command and a Pi prompt
  When they are registered
  Then each note carries the key-derived slug and a preamble naming its path

Scenario: 792b31c7a5 A recorded source round-trips through removal eligibility
  Given a candidate under the home or its resolved form
  When it is registered
  Then its skill_source expands back to a path under a read root

Scenario: 792b31c7a6 A source is recorded home-relative
  Given sources under the home, its resolved form and no home
  When note sources are built
  Then each is ~/-relative or absolute under no home

Scenario: 792b31c7a7 Refresh replaces the body, keeps the fields and marks pending
  Given a stale note
  When it is refreshed
  Then the body and hash are new, authored fields and basename are kept and pending is true

Scenario: 792b31c7a8 Refresh sets pending again
  Given a note whose pending marker was cleared
  When it is refreshed
  Then pending is true

Scenario: 792b31c7a9 Red flags survive a refresh
  Given red_flags authored on the note
  When it is refreshed
  Then they are still present

Scenario: 792b31c7a10 Refresh follows a plugin version bump
  Given a plugin skill under a new installPath version
  When it is refreshed
  Then skill_source and the preamble name the new path

Scenario: 792b31c7a11 Refresh converts a CRLF note
  Given a note with CRLF lines
  When it is refreshed
  Then it is written as LF in one write

Scenario: 792b31c7a12 Refresh refuses an anchored key
  Given an anchored skill_key
  When it is refreshed
  Then it is refused untouched, naming the key

Scenario: 792b31c7a13 Refresh and adopt keep unmodeled keys and agree on pending
  Given a note with an unmodeled key and any unknown keys
  When it is refreshed or adopted
  Then the keys survive, refresh sets pending and adopt clears it

Scenario: 792b31c7a14 Refresh failures are returned
  Given a lock, read, write, embed or frontmatter failure
  When it is refreshed
  Then the error is returned

Scenario: 792b31c7a15 Removal deletes the note and its sidecar
  Given an accepted removal
  When it is carried out
  Then both files are gone

Scenario: 792b31c7a16 Removal tolerates a missing sidecar
  Given a note with no sidecar
  When it is removed
  Then there is no error

Scenario: 792b31c7a17 Adopt renames, rewrites links, keeps fields and clears pending
  Given note 1049 with no skill_key and an inbound link
  When it is adopted as claude:curate
  Then it is renamed, links rewritten, fields kept and pending cleared

Scenario: 792b31c7a18 Adopt is idempotent
  Given a note already carrying the key's slug
  When it is adopted again
  Then body and hash are refreshed and nothing else changes

Scenario: 792b31c7a19 Adopt clears a pending marker
  Given a note with pending: true
  When it is adopted
  Then pending is cleared

Scenario: 792b31c7a20 Adopt migrates an unkeyed legacy note
  Given an unkeyed legacy note 1049
  When --adopt claude:curate=1049 runs
  Then it is renamed to the key slug and stamped

Scenario: 792b31c7a21 Adopt rebuilds the referrers' sidecars
  Given a note linking the old basename
  When it is adopted
  Then embed status is clean afterward

Scenario: 792b31c7a22 Adopt refuses a note keyed to another skill
  Given a note keyed claude:route
  When claude:curate is adopted onto it
  Then it is refused

Scenario: 792b31c7a23 Adopt refuses a conflicting existing note
  Given a different note already registered for the key
  When it is adopted
  Then it errors

Scenario: 792b31c7a24 Adopt errors on a missing target
  Given a reference naming no note
  When it is adopted
  Then it errors

Scenario: 792b31c7a25 Adopt errors on a non-runbook target
  Given a fact note
  When it is adopted
  Then it errors

Scenario: 792b31c7a26 Adopt errors on an unparseable basename
  Given a runbook whose basename has no Luhmann ID and date
  When it is adopted
  Then it errors

Scenario: 792b31c7a27 A render failure leaves the note unrenamed
  Given a runbook with a string red_flags
  When it is adopted
  Then it errors and the note is not renamed

Scenario: 792b31c7a28 Adopt converts a CRLF note
  Given a CRLF runbook note, renamed or already slugged
  When it is adopted
  Then it is converted to LF with its fields and luhmann kept

Scenario: 792b31c7a29 Adopt refuses an anchored pending key
  Given a pending key carrying an anchor
  When it is adopted
  Then it is refused untouched

Scenario: 792b31c7a30 Adopt failures are returned
  Given an embed, lookup, listing, lock, scan or write failure
  When it is adopted
  Then the error is returned
```

### 792b31c8 The command wiring

Requirement: [792b31c8 The command wiring](#792b31c8-the-command-wiring) The skill-registration-ledger MUST wire `engram register-skills` to the registration run over the real filesystem: `--accept` MUST register
a real note, `--adopt` MUST adopt a real note, `--dry-run` MUST list a register offer, and a run whose
standard input is not a terminal MUST never prompt or write. The command MUST accept `--vault`,
`--vault-name`, `--skills-dir`, `--dry-run`, `--accept`, `--decline` and `--adopt` and describe itself
as offering to register, refresh or remove vault runbook notes mirroring the skills, commands and prompt
templates in the default folders.

Supports: [792a12g1a An agent runs update through a shell](../product/skill-runbook-registration.md#792a12g1-uat-scenarios), [792a12h1a An agent relays the user's answer](../product/skill-runbook-registration.md#792a12h1-uat-scenarios), [792a12h1b A dry run](../product/skill-runbook-registration.md#792a12h1-uat-scenarios), [792a12h1d A skills-dir run is a preview](../product/skill-runbook-registration.md#792a12h1-uat-scenarios), [792a12i1a Adopting a previously promoted note](../product/skill-runbook-registration.md#792a12i1-uat-scenarios).

#### 792b31c8a Scenarios

```gherkin
Scenario: 792b31c8a1 Accept registers a real note
  Given a real filesystem and a scanned skill
  When register-skills --accept runs
  Then a real note is written

Scenario: 792b31c8a2 Adopt adopts a real note
  Given a real filesystem and a runbook note
  When register-skills --adopt runs
  Then the note is adopted

Scenario: 792b31c8a3 A dry run lists a register offer
  Given a real filesystem and a scanned skill
  When register-skills --dry-run runs
  Then the register offer is listed

Scenario: 792b31c8a4 A non-terminal never prompts or writes
  Given the real built binary and stdin that is not a terminal
  When register-skills runs
  Then it never prompts and writes nothing
```
