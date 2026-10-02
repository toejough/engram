## MODIFIED Requirements

### Requirement: Curation judges pending offers the same way recall judges candidates
Curation of a pending offer SHALL use the same covered/near/absent judgment `recall`'s Step 2.5 already performs against query candidates, applied instead to the pending offer against the host vault's existing notes. The curation procedure SHALL be carried by the curate skill, whose runbook note in the vault (per capability `skill-runbook-registration`) lets `engram query` surface it by situation and trigger. A **covered** offer SHALL be discarded: the existing note is reinforced with `engram amend --activate`, then the offer is deleted with `engram amend --discard --into <existing>`. On a served offer, every bookkeeping step SHALL pass `--expect-hash` with the exchange hash that was judged. A **near** offer SHALL be folded into the existing note it overlaps with via `engram amend`, and the now-redundant offer SHALL then be discarded with `engram amend --discard --into <existing>`. `--discard --into` SHALL record the offer's basename and all of its `aliases` in `<existing>`'s `aliases`, and SHALL merge the offer's parent links into `<existing>`'s (capability `vault-note-identity`). A pulled-down offer that curation rejects outright is discarded with a bare `--discard`, which records a declined pull (capability `vault-parent-pulldown`). An offer carrying `offer.for` SHALL be judged against that named note first. Curation SHALL judge offers the same way whether they arrived from a child over the served API or were pulled down from the parent. An **absent** offer SHALL be accepted as a normal note by clearing its own pending-offer marker with `engram amend --clear-pending`. That keeps the offer's declared author, because bookkeeping amends do not re-stamp identity. On a vault that has its own parent, accepting a served offer SHALL also offer the note onward (capability `vault-parent-offers`). The pending-offer marker is cleared only on an offer that is kept (the absent case); a discarded offer carries no marker afterwards. A pending **skill runbook note** (a runbook note carrying `skill_hash`) SHALL NOT be judged covered/near/absent and SHALL NOT be discarded by curation: curation SHALL instead author its `situation`, `done_when`, `triggers`, and `red_flags` from its body when it has none, or re-check existing ones against its body after a refresh (amending any that no longer fit). The 1200-byte figure is `engram query`'s `red_flags` preview budget, measured in rendered YAML bytes (indent, list marker, quotes, escapes and newline included), not a cap on what a runbook may carry: `engram show` always returns every entry, so curation SHALL NOT drop or shorten a still-valid red flag merely to fit the budget, and SHALL measure the budget in rendered bytes when it reports it, and then clear its marker with `engram amend --clear-pending`. Curation SHALL run asynchronously, never synchronously within the HTTP request that created the offer.

#### Scenario: A covered offer is discarded
- **WHEN** curation judges a pending offer as covered by an existing note
- **THEN** the existing note is reinforced, the offer is discarded with `--into` naming the existing note (which gains the offer's basename and aliases), and no new or modified note content results from it

#### Scenario: A near offer is folded into an existing note
- **WHEN** curation judges a pending offer as near an existing note
- **THEN** the existing note is amended to incorporate the offer's additional claim, and the pending offer is discarded with `--into` naming the existing note, so no second live note remains

#### Scenario: An absent offer is accepted
- **WHEN** curation judges a pending offer as absent from the vault
- **THEN** the offer's pending-offer marker is cleared and it becomes a normal, live note whose `user:` is still the offer's declared author

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
- **WHEN** curation judges a pulled-down offer as covered by local note L
- **THEN** the offer is discarded with `--into L`, and L gains a link to the parent note, so later merged queries dedupe it and it is not pulled again unchanged

#### Scenario: A rejected pulled-down offer is remembered
- **WHEN** curation discards a pulled-down offer outright
- **THEN** the parent note is recorded as declined at that exchange hash

#### Scenario: An amend-offer is judged against its target
- **WHEN** a pending offer carries `offer.for: E`
- **THEN** curation judges it against E first, and a near judgment folds it into E


#### Scenario: Curation does not trim valid red flags to fit the preview budget
- **WHEN** curation re-checks a skill runbook note whose still-valid `red_flags` render to more than 1200 bytes
- **THEN** curation keeps every still-valid entry, and any budget figure it reports is the rendered-YAML byte count, not the sum of the plain strings
