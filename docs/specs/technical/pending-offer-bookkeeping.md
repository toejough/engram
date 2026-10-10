# Pending Offer Bookkeeping - Technical Spec

## 792b34a Orientation

- Ownership: Repository component.
- Responsibility: decides whether a note is a pending offer, finds the pending offers in a vault, leaves them out of a
  list of notes, defines the one instruction that tells an agent to curate them, and carries the bookkeeping that
  settles an offer: the check that the version a curator judged is the version still there, the fold that records an
  offer's names and parent links on an existing note and deletes the offer, and the exchange-hash header that
  `engram show` prints so that the judged version can be named.
- Current realization (informational): `internal/cli/offer.go` (`noteHasPendingMarker`, `excludePendingOffers`,
  `vaultHasPendingOffers`, `warnIfPendingOffers`, `pendingOfferCurateInstruction`) and `internal/cli/amend_fold.go`
  (`checkJudgedVersion`, `foldInto`, `foldParentLinks`, `foldAliases`), with the header written by `RunShow`.
- Qualification: responsibility, the marker, the scan, the check and the fold are stated without naming a function;
  ownership, it owns what makes a note pending and how an offer is settled, while [exchange-versioning](./exchange-versioning.md) owns the hash and
  the link edit it calls, served learn and pull-down own the writing of a pending note, and [curate](../product/curate.md) owns the judgment;
  independent verification, each runs over in-memory notes and a fake vault; stability, the contract survives a new
  note type or a new surfacing point; necessity, without it a pending offer would be served as a live note and an
  offer changed after it was judged could be accepted unseen.
- Supported capabilities: [vault-serving](../product/vault-serving.md), [parent-sync](../product/parent-sync.md), [curate](../product/curate.md).

Rationale: an offered note, whether sent by a child or pulled down from a parent, must be judged before it counts as
knowledge, so everything that reads the vault must tell it apart from a live note, and what settles it must not act on
a version nobody judged. Curation is expected upkeep, so the same instruction travels with every signal that offers are
waiting. The fold refuses to guess: it never re-embeds, re-stamps identity, queues an offer or records a decline, because
the existing note's content is unchanged and only its links and names grow.

## 792b34c Behavior Requirements

### 792b34c1 The pending-offer marker

Requirement: [792b34c1 The pending-offer marker](#792b34c1-the-pending-offer-marker) The pending-offer-bookkeeping MUST report a note as a pending offer exactly when it is a fact,
feedback or runbook note, with or without `skill_hash`, whose frontmatter carries `pending: true`, reading a note with
CRLF line endings as its LF form. It MUST NOT report a note of any other type (a vocabulary definition, for example),
a note without that marker, or content that has no frontmatter.

Supports: [792a20d1b The offered note is pending, not live](../product/vault-serving.md#792a20d1-uat-scenarios), [792a22a1b A pending skill runbook note does not surface either](../product/curate.md#792a22a1-uat-scenarios), [792a21zb1c A CRLF pending offer stays pending](../product/parent-sync.md#792a21zb1-uat-scenarios).

Rationale: a pending skill runbook note is treated exactly as a pending fact is, so a newly registered skill is not
served before curation authors its fields ([skill-runbook-registration](../product/skill-runbook-registration.md)). A CRLF offer is still pending, so query excludes it and the server
never hands it out as live.

#### 792b34c1a Scenarios

```gherkin
Scenario: 792b34c1a1 Pending fact and feedback notes are flagged
  Given a pending fact, a pending feedback, a normal fact, a vocabulary definition note and text with no frontmatter
  When each is checked for the marker
  Then only the pending fact and the pending feedback are pending offers

Scenario: 792b34c1a2 A pending runbook is flagged with or without skill_hash
  Given a pending skill runbook, a pending runbook with no skill_hash and a skill runbook that is not pending
  When each is checked
  Then the first two are pending offers and the third is not

Scenario: 792b34c1a3 The pending flag decides for every type
  Given a note of any of the three types, with or without skill_hash, with or without pending: true
  When it is checked
  Then it is a pending offer exactly when it carries pending: true

Scenario: 792b34c1a4 A CRLF offer is still pending
  Given a pending offer and a normal note, each with CRLF line endings
  When each is checked
  Then the pending offer is reported pending and the normal note is not
```

### 792b34c2 Leaving pending offers out of a list of notes

Requirement: [792b34c2 Leaving pending offers out of a list of notes](#792b34c2-leaving-pending-offers-out-of-a-list-of-notes) The pending-offer-bookkeeping MUST, given a list of notes, return the notes that are not pending
offers and whether at least one pending offer was found, reading each note's file at the vault path joined with the
note's basename and `.md` once. It MUST keep a note whose file cannot be read or parsed, and MUST drop a pending skill
runbook note exactly as it drops a pending fact.

Supports: [792a20d1b The offered note is pending, not live](../product/vault-serving.md#792a20d1-uat-scenarios), [792a22a1a A pending offer does not surface in recall](../product/curate.md#792a22a1-uat-scenarios), [792a22a1b A pending skill runbook note does not surface either](../product/curate.md#792a22a1-uat-scenarios).

Rationale: a read failure must not silently drop a note that query would otherwise return, so the filter fails open.
The one pass gives the query payload its `pending_offers` flag without a second scan.

#### 792b34c2a Scenarios

```gherkin
Scenario: 792b34c2a1 A pending note is dropped and an unreadable one is kept
  Given a pending note, a normal note and a note whose read fails
  When the notes are filtered
  Then the pending note is dropped, the other two are kept and a pending offer is reported

Scenario: 792b34c2a2 A list with no pending offer is kept whole
  Given a list of normal notes
  When it is filtered
  Then every note is kept, no pending offer is reported, and each note was read at <vault>/<basename>.md

Scenario: 792b34c2a3 A pending skill runbook is dropped and an ordinary runbook kept
  Given a pending skill runbook note and a normal runbook note
  When they are filtered
  Then the pending skill runbook is dropped and the ordinary runbook is kept
```

### 792b34c3 Finding pending offers

Requirement: [792b34c3 Finding pending offers](#792b34c3-finding-pending-offers) The pending-offer-bookkeeping MUST decide whether a vault holds a pending offer by scanning the
vault's current notes every time it is asked, and MUST NOT persist a flag or withhold the answer until a growth or time
threshold is met. It MUST report true for a vault that holds one pending note, false for one that holds none, and false,
without an error, for a vault directory it cannot read or a note it cannot parse. It MUST, when a write has completed
and the vault holds a pending offer, log one warning that carries the curate instruction ([792b34c4 The curate instruction](#792b34c4-the-curate-instruction)), and MUST log
nothing when the vault holds none or when any dependency of the check is missing.

Supports: [792a22o1a The query payload reflects current state](../product/curate.md#792a22o1-uat-scenarios), [792a22o1h The write-path nudge fires](../product/curate.md#792a22o1-uat-scenarios).

Rationale: pending offers are lower-volume and costlier to leave stale than the refit trigger's backlog, so detection is
stateless and unbatched. A detection failure must never fail `engram update`'s primary job, so an unreadable vault is
no signal. The write-path nudge runs at the call sites where the refit trigger is already checked, and persists nothing.

#### 792b34c3a Scenarios

```gherkin
Scenario: 792b34c3a1 A vault with a pending note is flagged
  Given a vault holding one pending note
  When the vault is scanned
  Then it is reported as holding a pending offer

Scenario: 792b34c3a2 A vault of normal notes is not flagged
  Given a vault holding only normal notes
  When the vault is scanned
  Then no pending offer is reported

Scenario: 792b34c3a3 A missing vault directory is silent
  Given a vault directory that does not exist
  When the vault is scanned
  Then no pending offer is reported and there is no error

Scenario: 792b34c3a4 The write-path nudge fires once for pending offers
  Given a vault holding a pending note and a write that has completed
  When the nudge check runs
  Then one warning is logged, and it holds the curate query command and the shared instruction and names no skill

Scenario: 792b34c3a5 The nudge is silent without pending offers
  Given a vault whose notes are all normal
  When the nudge check runs
  Then nothing is logged

Scenario: 792b34c3a6 A missing dependency makes the check a no-op
  Given the note lister is missing
  When the nudge check runs
  Then nothing is logged and nothing panics
```

### 792b34c4 The curate instruction

Requirement: [792b34c4 The curate instruction](#792b34c4-the-curate-instruction) The pending-offer-bookkeeping MUST define one curate instruction on a single line that says
reviewing pending offers is expected vault upkeep and not an extra, says to curate them after finishing the user's
request without asking, and carries the command
`engram query --text="curate pending offers" --phrase="reviewing pending offers in a vault and judging each against existing notes"`
in the attached flag form, and names no skill. The update notice, the write-path nudge and the query payload's
`pending_offers_hint` MUST each embed this one instruction, and the hint MUST be derived from the pending flag, so it is
the instruction when offers are pending and empty otherwise.

Supports: [792a22o1b The payload carries the hint when offers are pending](../product/curate.md#792a22o1-uat-scenarios), [792a22o1d The instruction says curation is expected upkeep](../product/curate.md#792a22o1-uat-scenarios), [792a22o1e The payload has no hint without offers](../product/curate.md#792a22o1-uat-scenarios).

Rationale: an agent that met the cue in the middle of unrelated work declined it every time as "you didn't ask for that",
so the instruction says that curation is expected. The hint is derived from the flag and never transported, so a merged
or served payload cannot carry a stale or drifted copy. A runbook has no description the harness reads, so each cue
carries the query that surfaces the curate runbook.

#### 792b34c4a Scenarios

```gherkin
Scenario: 792b34c4a1 The instruction says curation is expected upkeep
  Given the shared curate instruction
  When it is read
  Then it says it is expected vault upkeep, says to curate after finishing the user's request, says without asking, and holds no newline

Scenario: 792b34c4a2 The hint follows the flag
  Given a query payload rendered with pending offers and one rendered without
  When their hints are read
  Then the first holds the shared instruction and the second holds no hint
```

### 792b34c5 The judged-version check

Requirement: [792b34c5 The judged-version check](#792b34c5-the-judged-version-check) The pending-offer-bookkeeping MUST require `--expect-hash` for `engram amend` with
`--clear-pending`, `--discard` or `--discard --into` on a note that carries `offer.origin`, whether the note is pending
or live, and MUST verify a given `--expect-hash` on any note. It MUST proceed only when the note's current exchange hash
is equal to the expected one, and MUST otherwise fail, changing nothing and recording no decline: when the hashes differ,
when the comparison is unknown, and when the note's current hash cannot be computed. A note whose frontmatter does not
parse has no readable `offer.origin` and MUST need no hash for a bare discard.

Supports: [792a22l1a A step on a served offer without the hash is refused](../product/curate.md#792a22l1-uat-scenarios), [792a22l1b An offer updated after judgment is not accepted](../product/curate.md#792a22l1-uat-scenarios), [792a22l1c A changed offer is judged again](../product/curate.md#792a22l1-uat-scenarios), [792a22l1d A note with no exchange header needs no hash](../product/curate.md#792a22l1-uat-scenarios).

Rationale: an offer can be updated in place after a curator judged it, so acting on the old judgment would accept or
discard content nobody reviewed. A hash of another version is not the version that was judged, so unknown fails as
changed does and the curator re-reads and judges again, and since `offer` survives acceptance, a host-side discard of a
once-offered live note needs the hash too, because a same-origin retry could otherwise race it.

#### 792b34c5a Scenarios

```gherkin
Scenario: 792b34c5a1 The hash is checked across every bookkeeping mode
  Given each of clear-pending, discard, discard-into and activate, a note with or without offer.origin, pending or live, and an expected hash that is absent, correct, changed or of another version
  When the amend runs
  Then it succeeds exactly when the hash is correct, or absent on a note that needs none, and otherwise writes nothing and records no decline

Scenario: 792b34c5a2 An offer updated after judgment is not accepted
  Given a curator judged pending offer N at hash H1 and the child then updated N in place to H2
  When engram amend --target N --clear-pending --expect-hash H1 runs
  Then it fails, and N stays pending with the H2 content

Scenario: 792b34c5a3 An unparseable note needs no hash
  Given a note whose frontmatter does not parse
  When a bare discard runs without --expect-hash
  Then the note is discarded

Scenario: 792b34c5a4 A hash that cannot be computed fails
  Given an expected hash and a note whose current exchange hash cannot be computed
  When the amend runs
  Then it fails and changes nothing
```

### 792b34c6 The fold of an offer into an existing note

Requirement: [792b34c6 The fold of an offer into an existing note](#792b34c6-the-fold-of-an-offer-into-an-existing-note) The pending-offer-bookkeeping MUST, for `engram amend --target O --discard --into E`, record on E
the basename of O and every alias O answered to (each once, never E's own basename), merge O's parent links into E's
([792b34c7 Merging parent links](#792b34c7-merging-parent-links)), and then delete O and its sidecar. It MUST change only E's `aliases:` and `parent:` keys, MUST NOT re-embed
E (except to refresh the sidecar of an E whose CRLF line endings the write converted), MUST NOT re-stamp identity, stamp an
`xid`, queue an offer or record a decline, and MUST keep E's own `xid` and author. It MUST fail, changing nothing, for
`--into` without `--discard`, for an `--into` that names no note, and for an `--into` that names O itself.

Supports: [792a22f1a A covered offer is discarded](../product/curate.md#792a22f1-uat-scenarios), [792a22f1b A pulled-down offer covered by a local note](../product/curate.md#792a22f1-uat-scenarios), [792a22g1a A near offer is folded into an existing note](../product/curate.md#792a22g1-uat-scenarios), [792a22g1b The fold comes before the amend](../product/curate.md#792a22g1-uat-scenarios), [792a22m1a A fold unions aliases and links](../product/curate.md#792a22m1-uat-scenarios), [792a22m1b A bad --into is refused](../product/curate.md#792a22m1-uat-scenarios), [792a22m1c A fold does not re-stamp identity](../product/curate.md#792a22m1-uat-scenarios), [792a22m1d A fold converts a CRLF existing note](../product/curate.md#792a22m1-uat-scenarios), [792a22m1e A fold that changes nothing leaves E alone](../product/curate.md#792a22m1-uat-scenarios).

Rationale: the fold is for an offer that curation judged covered or near, where E already holds the content, so only
its names and links grow; recording O's names keeps a later reference to O resolving, and the links keep the parent note
from being pulled down again unchanged. E keeps its own author, and O's is not carried (the loss is documented, and the
basename survives in `aliases`).

#### 792b34c6a Scenarios

```gherkin
Scenario: 792b34c6a1 A pulled copy is folded into a note that has a primary
  Given a note E with its own xid, an alias and a primary link, and an offer O whose aliases include one name and whose links are a pulled primary and a covered link
  When O is folded into E
  Then O and its sidecar are gone, E gains O's basename and alias, O's primary becomes a covered link on E and O's covered link rides along, E keeps its xid, only E's exchange keys changed, and nothing was re-embedded, re-stamped or declined

Scenario: 792b34c6a2 The offer's primary stays primary when E has none
  Given a note E with no parent block and an offer whose link is a pulled primary
  When O is folded into E
  Then E adopts O's parent vault, holds the link as its pulled primary and has no xid

Scenario: 792b34c6a3 Aliases and links are unioned for any inputs
  Given random alias and link sets on E and on O
  When O is folded into E
  Then E holds every alias it had and O's basename and aliases without duplicates or its own basename, holds every link O held under O's parent vault, keeps its own primary link when it had one and never holds two primaries, O is gone, and nothing outside parent and aliases changed

Scenario: 792b34c6a4 A bad target is refused
  Given --into without --discard, an --into naming no note, and an --into naming the offer itself
  When each amend runs
  Then each fails and no file changes and no decline is recorded

Scenario: 792b34c6a5 A CRLF existing note is converted by the fold
  Given an existing note E whose lines end in CRLF and a fold that changes E's aliases or parent links
  When the offer is folded into E
  Then E is written with no CRLF, its sidecar is rebuilt, and the offer and its sidecar are gone
```

### 792b34c7 Merging parent links

Requirement: [792b34c7 Merging parent links](#792b34c7-merging-parent-links) When the pending-offer-bookkeeping folds O into E it MUST give E the links O held under O's parent
vault, with O's primary (`offered` or `pulled`) link staying primary only when E has none of its own and becoming
`covered` otherwise. A link E already holds for the same parent note MUST keep E's `via` role and take O's link's
`hash`, the version just judged. When O's `parent.vault` differs from E's, E's links under its previous vault MUST be
dropped, E's `parent.vault` MUST become O's, and only O's links MUST carry forward. When E has no primary link and holds
a `covered` link to the note that is O's primary, that link MUST be promoted to O's primary role with O's hash.

Supports: [792a22f1b A pulled-down offer covered by a local note](../product/curate.md#792a22f1-uat-scenarios), [792a22m1a A fold unions aliases and links](../product/curate.md#792a22m1-uat-scenarios).

Rationale: links count only under the current parent's vault, as a receipt's own replacement rule has it ([792b33c9 Recording an offer receipt](./exchange-versioning.md#792b33c9-recording-an-offer-receipt)). Taking
O's hash records the version curation judged, so the pull-down skip and the dedupe see the pair as already related.

#### 792b34c7a Scenarios

```gherkin
Scenario: 792b34c7a1 An existing link keeps its role and takes the offer's hash
  Given E with its own primary link to Q and a covered link to P, and O whose primary link is to P under the same vault
  When O is folded into E
  Then E's primary link is still to Q, and its link to P is still covered and carries O's hash for P

Scenario: 792b34c7a2 A vault switch drops the links under the old vault
  Given E with links under vault V1 and O whose links are all under vault V2
  When O is folded into E
  Then E's parent.vault is V2, none of its V1 links remain and its links are O's links

Scenario: 792b34c7a3 A held covered link is promoted to primary
  Given E with no primary link but a covered link to P, and O whose primary link is to P
  When O is folded into E
  Then E's link to P is O's pulled or offered primary with O's hash
```

### 792b34c8 The exchange-hash header of show

Requirement: [792b34c8 The exchange-hash header of show](#792b34c8-the-exchange-hash-header-of-show) The pending-offer-bookkeeping MUST make `engram show` print `# exchange_hash: <hash>` as its first
output line, before the note, for a note that carries `xid`, reading a note with CRLF line endings as its LF form for
the header and printing the note itself verbatim, and MUST print a note without `xid` exactly as before. The hash printed
for a CRLF note MUST be the one that `--expect-hash` accepts.

Supports: [792a20k1a Show without raw matches the local output](../product/vault-serving.md#792a20k1-uat-scenarios), [792a22l1c A changed offer is judged again](../product/curate.md#792a22l1-uat-scenarios).

Rationale: the header is how a curator learns the version to name in `--expect-hash` ([792b34c5 The judged-version check](#792b34c5-the-judged-version-check)), and a note that never
took part in exchange has no version to name, so its output does not change.

#### 792b34c8a Scenarios

```gherkin
Scenario: 792b34c8a1 The header comes first on an exchanged note
  Given a pending offer that carries xid
  When engram show runs on it
  Then the first line is "# exchange_hash: " followed by its exchange hash, and the note follows byte for byte

Scenario: 792b34c8a2 No header without an xid
  Given a note that carries no xid
  When engram show runs on it
  Then the output is the note exactly as before and holds no exchange_hash line

Scenario: 792b34c8a3 A CRLF note shows its LF hash
  Given a pending offer with xid whose lines, frontmatter included, end in CRLF
  When engram show runs on it and the printed hash is passed to engram amend --clear-pending --expect-hash
  Then the first line is the exchange-hash header, the note follows verbatim and the amend succeeds
```
