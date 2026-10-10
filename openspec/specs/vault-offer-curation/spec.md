## Purpose

Gives offered notes, whether arriving from a child over the API or pulled down from the parent, a curated acceptance step instead of an immediate commit: an offered note is evaluated — discarded, folded into an existing note, or accepted — using the same agent-judged covered/near/absent reasoning `recall` already performs, before it becomes a normal, live vault note.

The technical contracts of the pending-offer marker, the scan for pending offers, the judged-version check on bookkeeping amends, the fold's merge of parent links and the exchange-hash header of `engram show` are in `docs/changes/792_documentation-framework-adoption/792b_technical-spec.md#792b34-added-pending-offer-bookkeeping` (792b34c1 to 792b34c8). What remains here waits for curate (792c7u1).
## Requirements
### Requirement: Served writes land as pending offers, not immediate notes
A `learn` request handled by `engram serve` SHALL be written with a pending-offer marker rather than being immediately treated as a normal, live note. So SHALL a note pulled down from the parent (capability `vault-parent-pulldown`). Offers therefore reach curation from both directions: from a child to its parent, and from the parent to the child.

#### Scenario: A served learn request creates a pending offer
- **WHEN** a `learn` request is handled over the served API
- **THEN** the resulting note carries the pending-offer marker

#### Scenario: A pulled-down parent note is a pending offer
- **WHEN** a child activates a parent-only note and it is pulled down
- **THEN** the local copy carries the pending-offer marker and awaits local curation

### Requirement: activate commits directly and never becomes a pending offer
`engram activate` of a note that exists in the vault it runs against (locally or served) SHALL commit directly, bumping the target note's sidecar `LastUsed`, without creating a pending offer and without going through curation. Activating a ref that exists only in the parent is not an activation of a local note. It is a pull-down (capability `vault-parent-pulldown`), which creates a pending copy of the parent's note locally and does not mark any existing note pending.

#### Scenario: A served activate request commits immediately
- **WHEN** an `activate` request is handled over the served API
- **THEN** the named note's sidecar `LastUsed` is updated immediately, with no pending-offer marker created and no curation step involved

### Requirement: Pending offers are excluded from normal query results
`engram query` SHALL NOT return a pending-offer note as part of its normal candidate/result set.

#### Scenario: A pending offer does not surface in recall
- **WHEN** `engram query` runs while a pending-offer note exists in the vault
- **THEN** that note is absent from the returned results

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

### Requirement: Curation outcome is not reported back to the offering caller
The served API's response to a `learn` request that creates or updates a pending offer SHALL confirm only that the offer was received, naming the pending note (792a20d in `docs/changes/792_documentation-framework-adoption/792a_product-spec.md#792a20d-a-served-learn-lands-as-a-pending-offer-and-returns-a-receipt`). It SHALL NOT report the offer's eventual curation outcome. No later notification of the outcome SHALL be sent to the offering caller.

#### Scenario: Server responds before curation happens
- **WHEN** a served `learn` request creates a pending offer
- **THEN** the server's response is returned before curation has run, and no subsequent message reports the offer's disposition

### Requirement: Pending offers are surfaced at three points
`engram query` SHALL include a pending-offer-exists flag in its payload on every call, and when the flag is true SHALL also include a `pending_offers_hint` string carrying the same curate instruction the notices carry (omitted when the flag is false, so payloads without pending offers are unchanged). When present, `pending_offers` and `pending_offers_hint` SHALL be the first keys after `version` in the YAML payload, before `items` and every other large field, so a truncated preview of a large payload still shows them. The curate instruction SHALL state that reviewing pending offers is expected vault upkeep, not an extra, that the agent is to curate them after it finishes the user's request without asking, and SHALL carry the copy-pasteable command; the update notice, the write-path nudge and the payload hint SHALL each embed this same instruction from one shared definition, and the notice and nudge SHALL each remain a single line. `engram update` SHALL include a notify-only notice when pending offers exist, following the same detect-and-notify convention as its other vault-condition detectors. `engram learn`, `engram amend`, and `engram resituate` SHALL log a warning-level nudge, at the same point each already checks the `vocab refit` trigger, when pending offers exist. The update notice and the write-path nudge SHALL each carry a copy-pasteable `engram query` command whose `--text` contains the curate runbook's trigger words, so that following the notice surfaces the curate runbook without naming a skill. The command's flags SHALL use the `--flag="value"` form (the quote touching the `=`, no space), not a space-separated `--text "value"`, consistent with every other shipped `engram` command template (`fix-eval-tooling-defects` D14/D17) — the string is a fixed literal today, not a free-text value that could itself begin with `-`, but the template form stays consistent with the rest of the surface this requirement is part of.

#### Scenario: Query payload reflects current state
- **WHEN** `engram query` runs
- **THEN** its payload's pending-offer-exists flag matches whether any pending offer currently exists

#### Scenario: Query payload carries the hint when offers are pending
- **WHEN** `engram query` runs while pending offers exist
- **THEN** its payload has `pending_offers: true` and a `pending_offers_hint` whose text is the same instruction (the `engram query --text="curate pending offers" ...` command and "follow the curate runbook it returns") that the update notice and write-path nudge embed, taken from one shared definition

#### Scenario: Flag and hint lead the payload
- **WHEN** `engram query` renders a payload larger than 100 KB while pending offers exist
- **THEN** `pending_offers: true` and `pending_offers_hint` appear before `items:`, and the first 1500 bytes of the payload contain the hint

#### Scenario: The instruction says curation is expected upkeep
- **WHEN** the shared curate instruction is read on any of the three surfaces
- **THEN** it says reviewing pending offers is expected vault upkeep, not an extra, says to curate them after finishing the user's request without asking, and gives the `engram query --text="curate pending offers" ...` command

#### Scenario: Query payload has no hint without offers
- **WHEN** `engram query` runs while no pending offer exists
- **THEN** its payload contains neither `pending_offers` nor `pending_offers_hint`, and is byte-identical to a payload produced before the hint existed

#### Scenario: Merged query carries the hint
- **WHEN** a merged query (local plus parent) runs
- **THEN** the merged payload's `pending_offers` and hint reflect the LOCAL vault's pending offers only — a child cannot curate its parent, so a parent-side pending flag is ignored; with no local pending offer the merged payload has neither field

#### Scenario: Served query carries the hint
- **WHEN** `GET /query` is served while pending offers exist on the host
- **THEN** the response body is the same YAML a local query would print, including `pending_offers: true` and `pending_offers_hint`

#### Scenario: Update surfaces a notice
- **WHEN** `engram update` runs while pending offers exist
- **THEN** its report includes a notify-only notice naming the pending offers and the `engram query --text="curate pending offers" ...` command that surfaces the curate runbook

#### Scenario: Write-path nudge fires
- **WHEN** `engram learn`, `engram amend`, or `engram resituate` completes while pending offers exist
- **THEN** a warning-level log line notes their presence and carries the same trigger query

#### Scenario: The notice's command surfaces the runbook
- **WHEN** an agent runs the command a notice prints against a vault that contains the curate runbook
- **THEN** the curate runbook appears in the query result with `trigger` provenance

