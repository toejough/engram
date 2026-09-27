## ADDED Requirements

### Requirement: The pending-offer marker SHALL apply to every note type
A note of any type (`fact`, `feedback`, or `runbook`, with or without `skill_hash`) that carries the pending-offer marker SHALL be treated as a pending offer. It SHALL be excluded from normal query results, counted by pending-offer detection, and reported at every surfacing point.

#### Scenario: An offered runbook is pending
- **WHEN** a served `learn runbook` creates a note with the pending-offer marker and no `skill_hash`
- **THEN** `engram query` omits it, and the payload has `pending_offers: true`

## MODIFIED Requirements

### Requirement: Served writes land as pending offers, not immediate notes
A `learn` request handled by `engram serve` SHALL be written with a pending-offer marker rather than being immediately treated as a normal, live note. A note pulled down from the parent (capability `vault-parent-pulldown`) SHALL likewise be written with the pending-offer marker. Offers therefore reach curation from both directions: from a child to its parent, and from the parent to the child.

#### Scenario: A served learn request creates a pending offer
- **WHEN** a `learn` request is handled over the served API
- **THEN** the resulting note carries the pending-offer marker

#### Scenario: A pulled-down parent note is a pending offer
- **WHEN** a child activates a parent-only note and it is pulled down
- **THEN** the local copy carries the pending-offer marker and awaits local curation

### Requirement: activate commits directly and never becomes a pending offer
`engram activate` of a note that exists in the vault it runs against (locally, or served) SHALL commit directly (bump the target note's sidecar `LastUsed`) without creating a pending offer and without going through curation. Activating a ref that exists only in the parent is not an activation of a local note. It is a pull-down (capability `vault-parent-pulldown`), which creates a pending copy of the *parent's* note in the local vault and does not mark any existing note pending.

#### Scenario: A served activate request commits immediately
- **WHEN** an `activate` request is handled over the served API
- **THEN** the named note's sidecar `LastUsed` is updated immediately, with no pending-offer marker created and no curation step involved

### Requirement: Curation judges pending offers the same way recall judges candidates
Curation of a pending offer SHALL use the same covered/near/absent judgment `recall`'s Step 2.5 already performs against query candidates, applied instead to the pending offer against the host vault's existing notes. The curation procedure SHALL be carried by the curate skill, whose runbook note in the vault (per capability `skill-runbook-registration`) lets `engram query` surface it by situation and trigger. A **covered** offer SHALL be discarded (the existing note is reinforced with `engram amend --activate`, then the offer is deleted with `engram amend --discard --into <existing>`). A **near** offer SHALL be folded into the existing note it overlaps with via `engram amend`, and the now-redundant offer SHALL then be discarded with `engram amend --discard --into <existing>`. `--discard --into <existing>` SHALL record the discarded offer's basename in `<existing>`'s `aliases`, and SHALL move the offer's parent link to `<existing>` when `<existing>` has none (capability `vault-note-identity`). An offer carrying `offer.for` SHALL be judged against that named note first. Curation SHALL judge offers the same way whether they arrived from a child over the served API or were pulled down from the parent. An **absent** offer SHALL be accepted as a normal note by clearing its own pending-offer marker with `engram amend --clear-pending`. The pending-offer marker is cleared only on an offer that is kept (the absent case); a discarded offer carries no marker afterwards. A pending **skill runbook note** (a runbook note carrying `skill_hash`) SHALL NOT be judged covered/near/absent and SHALL NOT be discarded by curation: curation SHALL instead author its `situation`, `done_when`, `triggers`, and `red_flags` from its body when it has none, or re-check existing ones against its body after a refresh (amending any that no longer fit, with `red_flags` within the 1200-byte rendered cap), and then clear its marker with `engram amend --clear-pending`. Curation SHALL run asynchronously, never synchronously within the HTTP request that created the offer.

#### Scenario: A covered offer is discarded
- **WHEN** curation judges a pending offer as covered by an existing note
- **THEN** the existing note is reinforced, the offer is discarded with `--into` naming the existing note (which gains the offer's basename in `aliases`), and no new or modified note content results from it

#### Scenario: A near offer is folded into an existing note
- **WHEN** curation judges a pending offer as near an existing note
- **THEN** the existing note is amended to incorporate the offer's additional claim, and the pending offer is discarded with `--into` naming the existing note, so no second live note remains

#### Scenario: An absent offer is accepted
- **WHEN** curation judges a pending offer as absent from the vault
- **THEN** the offer's pending-offer marker is cleared and it becomes a normal, live note

#### Scenario: Curation never composes a new learn
- **WHEN** curation handles any offer
- **THEN** every action is an `engram amend` call on existing files, with no `engram learn` and no handoff to `write-memory`

#### Scenario: A new skill note gets its runbook fields
- **WHEN** curation handles a pending skill runbook note with no `situation`, `done_when`, `triggers`, or `red_flags`
- **THEN** it amends the note to add those fields, authored from the note's body, clears the marker, and does not discard the note

#### Scenario: A refreshed skill note is re-checked, not discarded
- **WHEN** curation handles a pending skill runbook note that already has runbook fields
- **THEN** it amends any field that no longer matches the body, clears the marker, and does not discard the note even if another note covers similar ground

#### Scenario: A pulled-down offer covered by a local note
- **WHEN** curation judges a pulled-down offer (parent link `via: pulled`) as covered by local note L, and L has no parent link
- **THEN** the offer is discarded with `--into L`, and L gains the offer's parent link, so later merged queries dedupe L against the parent note

#### Scenario: An amend-offer is judged against its target
- **WHEN** a pending offer carries `offer.for: E`
- **THEN** curation judges it against E first, and a near judgment folds it into E

### Requirement: Curation outcome is not reported back to the offering caller
The served API's response to a `learn` request that creates a pending offer SHALL confirm only that the offer was received, naming the pending note's basename (capability `vault-serve-api`), not its eventual curation outcome. No later notification of the outcome SHALL be sent to the offering caller.

#### Scenario: Server responds before curation happens
- **WHEN** a served `learn` request creates a pending offer
- **THEN** the server's response is returned before curation has run, and no subsequent message reports the offer's disposition

### Requirement: The pending-offer marker SHALL apply to skill runbook notes
A runbook note carrying `skill_hash` (capability `skill-runbook-registration`) and the pending-offer marker SHALL be treated as a pending offer exactly as a pending fact or feedback note is: excluded from normal query results, counted by pending-offer detection, and reported at every pending-offer surfacing point. A runbook note without `skill_hash` that carries the pending-offer marker is covered by the requirement "The pending-offer marker SHALL apply to every note type".

#### Scenario: A newly registered skill note is pending
- **WHEN** a skill's registration is accepted and its note carries the pending-offer marker
- **THEN** `engram query` omits the note from its results and its payload has `pending_offers: true`

#### Scenario: Clearing the marker makes the skill note live
- **WHEN** curation clears the marker on a skill note whose runbook fields are authored
- **THEN** the note surfaces in `engram query` results like any runbook, and it no longer counts as a pending offer
