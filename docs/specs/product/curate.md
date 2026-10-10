# Curate - Product Spec

An offer is a note that another vault proposes: a child's served `learn` ([vault-serving](./vault-serving.md)) or a parent note pulled down by
`engram activate` ([parent-sync](./parent-sync.md)). It lands as a **pending offer**: a real note file marked `pending: true` that is not yet
knowledge, because nobody has judged it against what the vault already holds. Curation is that judgment. It is carried by
the `curate` skill (`agent-instructions/skills/curate/SKILL.md`, which a vault runbook note also mirrors), which uses the
same covered, near and absent reasoning that `recall` applies to its candidates ([recall](./recall.md)), and settles each offer with
`engram amend` flags: `--clear-pending`, `--discard`, `--into` and `--expect-hash`. The binary supplies the flags and the
checks that make a curator's judgment safe; the judgment itself is the agent's. Curation runs on the host that holds the
offer, after the request that created it, and never inside it. How an offer is written, offered and answered is vault-serving
([vault-serving](./vault-serving.md)) and [parent-sync](./parent-sync.md).

Rationale: The curate skill's behavior overrode the older spec: a covered offer is discarded, and a near offer
is folded into the existing note and then discarded, because leaving it live creates a second redundant note.
The marker is cleared only on an offer that is kept (Joe, 2026-09-21).

## 792a22a Pending offers are left out of normal query results

Requirement: [792a22a Pending offers are left out of normal query results](#792a22a-pending-offers-are-left-out-of-normal-query-results) `engram query` MUST NOT return a pending offer as part of its normal results, whatever its kind and
whether or not it is a skill runbook note.

Rationale: an offer is unjudged, so recall must not act on it as if it were knowledge. The query can tell the agent that
offers exist ([792a22o Pending offers are surfaced at three points](#792a22o-pending-offers-are-surfaced-at-three-points)) and not which, and the agent finds them by scanning the vault.
A pending offer is a normal note carrying a marker and not a separate queue. This reuses frontmatter parsing,
`query` and `amend`, so only the marker's presence changes behavior.

### 792a22a1 UAT Scenarios

```gherkin
Scenario: 792a22a1a A pending offer does not surface in recall
  Given a vault that holds a pending offer
  When the user runs engram query
  Then the offer is absent from the returned results

Scenario: 792a22a1b A pending skill runbook note does not surface either
  Given a vault that holds a pending skill runbook note
  When the user runs engram query
  Then the note is absent from the returned results
```

## 792a22b A note becomes pending only by an exchange

Requirement: [792a22b A note becomes pending only by an exchange](#792a22b-a-note-becomes-pending-only-by-an-exchange) A local `engram learn` and a local `engram amend` MUST NOT write the pending-offer marker. A served `learn` ([792a20d A served learn lands as a pending offer and returns a receipt](./vault-serving.md#792a20d-a-served-learn-lands-as-a-pending-offer-and-returns-a-receipt)),
a pull-down ([792a21zd A pulled-down note is an exact-content local pending offer](./parent-sync.md#792a21zd-a-pulled-down-note-is-an-exact-content-local-pending-offer)), accepting a skill's registration offer
([792a12d Accepting a registration creates a pending note](./skill-runbook-registration.md#792a12d-accepting-a-registration-creates-a-pending-note)) and accepting its refresh offer
([792a12e Accepting a refresh replaces the body, keeps the fields, and marks the note pending](./skill-runbook-registration.md#792a12e-accepting-a-refresh-replaces-the-body-keeps-the-fields-and-marks-the-note-pending)) MUST make a note pending, and a skill note is then
authored by curation ([792a22k A pending skill runbook note is authored, not judged](#792a22k-a-pending-skill-runbook-note-is-authored-not-judged)). `engram amend --clear-pending` MUST be how a curator clears the marker of a
note any of them left behind.

Rationale: the marker means that the note came from another vault, or from a skill file, and has not been judged or authored, so a note the user
writes in their own vault is never held back from recall. Only the exchanges and the skill registration write it; `engram learn` has no flag for it, and the
only amend flag that touches the marker clears it.

### 792a22b1 UAT Scenarios

```gherkin
Scenario: 792a22b1a A local learn is never pending
  Given the user runs engram learn on the host
  When the note is written
  Then it carries no pending marker and engram query can return it

Scenario: 792a22b1b A local amend never sets the marker
  Given a live note
  When the user runs engram amend on it
  Then the note carries no pending marker afterwards
```

## 792a22c Activating a note commits directly

Requirement: [792a22c Activating a note commits directly](#792a22c-activating-a-note-commits-directly) `engram activate` of a note that exists in the vault it runs against, locally or served, MUST commit
directly: it MUST bump the note's sidecar `LastUsed` without creating a pending offer and without going through curation.
Activating a ref that exists only in the parent vault is not an activation of a local note: it is a pull-down ([792a21zd A pulled-down note is an exact-content local pending offer](./parent-sync.md#792a21zd-a-pulled-down-note-is-an-exact-content-local-pending-offer)),
which writes a pending copy and marks no existing note pending.

Rationale: activation never changes a note's content, so there is no new claim for curation to judge.

### 792a22c1 UAT Scenarios

```gherkin
Scenario: 792a22c1a A served activate commits immediately
  Given an activate request handled over the served API
  When it completes
  Then the named note's sidecar LastUsed is updated, no pending-offer marker is created and no curation step is involved

Scenario: 792a22c1b A local activate commits immediately
  Given a note in the local vault
  When the user runs engram activate on it
  Then the note's sidecar LastUsed is updated and the note carries no pending marker
```

## 792a22d Curation judges each offer by its content against the host vault

Requirement: [792a22d Curation judges each offer by its content against the host vault](#792a22d-curation-judges-each-offer-by-its-content-against-the-host-vault) Curation MUST judge each pending offer as covered, near or absent against the host vault's existing,
non-pending notes, by reading the offer and the candidates in full and judging their content, never by a similarity score
alone, and MUST find the offers by scanning the vault for `pending: true` and not by `engram query`. It MUST judge an
offer the same way whether it arrived from a child over the served API or was pulled down from the parent. It MUST run on
the host that holds the offer, off the request path that created it, and MUST NOT run synchronously inside a served
request. The vault MUST be checked with `engram check` after a curation pass.

Rationale: recall's own documented mistake is to apply a threshold, and a high score can still be an unrelated false
positive while a low one can still be the same claim in different words. A served write's only job is to stamp identity,
persist the pending marker and answer, so judgment happens later and on the host's own initiative, when one of the three
signals of [792a22o Pending offers are surfaced at three points](#792a22o-pending-offers-are-surfaced-at-three-points) appears or when asked.

### 792a22d1 UAT Scenarios

```gherkin
Scenario: 792a22d1a A served offer and a pulled-down offer are judged alike
  Given a pending offer that arrived over the served API and another pulled down from the parent, with the same content
  When curation judges each
  Then both receive the same judgment

Scenario: 792a22d1b Judgment reads the content, not a score
  Given a pending offer whose nearest existing note has a high similarity score but states a different claim
  When curation judges it
  Then the offer is not judged covered

Scenario: 792a22d1c The offers are found by scanning the vault
  Given a vault that holds pending offers
  When curation looks for them
  Then it scans the vault's notes for pending: true, because engram query excludes them

Scenario: 792a22d1d Curation does not run inside the request
  Given a served learn that creates a pending offer
  When the server answers it
  Then the answer is returned before any curation has run

Scenario: 792a22d1e The vault is checked afterwards
  Given a curation pass that settled offers
  When it finishes
  Then engram check passes and engram query no longer shows the settled offers as pending
```

## 792a22e An amend-offer is judged against its target first

Requirement: [792a22e An amend-offer is judged against its target first](#792a22e-an-amend-offer-is-judged-against-its-target-first) A pending offer that carries `offer.for` MUST be judged against the note it names before any query, and
a near judgment MUST fold it into that note.

Rationale: an amend-offer is a child's change to a note that the parent holds, so the parent note is the candidate
that matters, and it is usually near, since it adds a claim, or covered.

### 792a22e1 UAT Scenarios

```gherkin
Scenario: 792a22e1a An amend-offer is judged against its target
  Given a pending offer that carries offer.for E
  When curation judges it
  Then it is judged against E first, and a near judgment folds it into E
```

## 792a22f A covered offer is reinforced and folded away

Requirement: [792a22f A covered offer is reinforced and folded away](#792a22f-a-covered-offer-is-reinforced-and-folded-away) When curation judges an offer covered by an existing note, it MUST reinforce that note with `engram amend
--activate` and then delete the offer with `engram amend --discard --into <existing>`, so that the offer's basename and
all of its `aliases` join the existing note's `aliases` and its parent links join that note's. Curation MUST NOT use a bare
`--discard` for it, and MUST NOT clear the offer's marker. The result MUST be no new or modified note content.

Rationale: a pending offer, unlike a recall candidate, is already a file. Clearing its marker after its content was found
elsewhere would create a second, live, redundant note, which is the duplication curation exists to prevent. `--into` keeps
the offer's identity, so a merged query dedupes the parent's copy and an unchanged parent note is never pulled down again.

### 792a22f1 UAT Scenarios

```gherkin
Scenario: 792a22f1a A covered offer is discarded
  Given curation judges a pending offer as covered by an existing note
  When it settles the offer
  Then the existing note is reinforced, the offer is discarded with --into naming the existing note, which gains the offer's basename and aliases, and no new or modified note content results

Scenario: 792a22f1b A pulled-down offer covered by a local note
  Given curation judges a pulled-down offer as covered by local note L
  When it settles the offer
  Then the offer is discarded with --into L, L gains a link to the parent note, and later merged queries dedupe it and it is not pulled again unchanged
```

## 792a22g A near offer is folded away first, then its claim is folded in

Requirement: [792a22g A near offer is folded away first, then its claim is folded in](#792a22g-a-near-offer-is-folded-away-first-then-its-claim-is-folded-in) When curation judges an offer near an existing note, it MUST first delete the offer with `engram amend --discard
--into <existing>` and then amend the existing note to incorporate the offer's additional claim, in that order, so that no
second live note remains and the amend is offered to the parent as an amend of the note the offer came from.

Rationale: the content amend is offered to the parent at once, and only after the fold does the existing note carry a pulled
offer's parent link, so folding first makes the amend go up as an amend-offer and not as a new note ([792a21n A local write is offered to the parent](./parent-sync.md#792a21n-a-local-write-is-offered-to-the-parent)).

### 792a22g1 UAT Scenarios

```gherkin
Scenario: 792a22g1a A near offer is folded into an existing note
  Given curation judges a pending offer as near an existing note
  When it settles the offer
  Then the offer is discarded with --into naming the existing note, the existing note is amended to incorporate the offer's additional claim, and no second live note remains

Scenario: 792a22g1b The fold comes before the amend
  Given a pulled offer O judged near a live local note E that has no primary link
  When curation folds O into E and then amends E, and when it instead amends E first
  Then in the first order E carries O's pulled link when the amend is offered, so the amend goes up as an amend-offer, and in the second E is still unlinked, so the amend goes up as a new note
```

## 792a22h An absent offer is accepted with its declared author

Requirement: [792a22h An absent offer is accepted with its declared author](#792a22h-an-absent-offer-is-accepted-with-its-declared-author) When curation judges an offer absent from the vault, it MUST accept it by clearing its own pending-offer marker with
`engram amend --clear-pending`, which MUST keep the offer's declared `repo`, `user` and `vault`, and MUST NOT write a new note.
The marker is cleared only on an offer that is kept: a discarded offer carries no marker afterwards. On a vault that has its
own parent, accepting a served offer MUST also offer the note onward ([792a21n A local write is offered to the parent](./parent-sync.md#792a21n-a-local-write-is-offered-to-the-parent)).

Rationale: the note already exists as the offer, so clearing the marker is the entire action. The bookkeeping amends do not
re-stamp identity, so an accepted offer keeps its author, which is the person who learned it and not the curator.

### 792a22h1 UAT Scenarios

```gherkin
Scenario: 792a22h1a An absent offer is accepted
  Given curation judges a pending offer as absent from the vault
  When it settles the offer
  Then the marker is cleared and the offer is a normal, live note whose user: is still the offer's declared author

Scenario: 792a22h1b Accepting an offer keeps its author
  Given a pending offer whose user: is alice, and a curator whose detected user is bob
  When the curator runs engram amend --target O --clear-pending
  Then O's user: is still alice
```

## 792a22i A rejected offer is discarded outright

Requirement: [792a22i A rejected offer is discarded outright](#792a22i-a-rejected-offer-is-discarded-outright) When curation judges an offer wrong, because an existing note or a user correction contradicts it or because it is
advice the agent would never apply, it MUST delete it with a bare `engram amend --discard`, never `--into`. For a pulled-down
offer the discard MUST record a decline of the parent note at its exchange hash, so that the note is not pulled down again
unless the parent changes it.

Rationale: `--into` means covered by, so folding a wrong offer into the note that contradicts it would record the bad note
as covered by the good one ([792a21ze A pull-down is idempotent, remembers declines and cannot loop](./parent-sync.md#792a21ze-a-pull-down-is-idempotent-remembers-declines-and-cannot-loop)).

### 792a22i1 UAT Scenarios

```gherkin
Scenario: 792a22i1a A rejected pulled-down offer is remembered
  Given curation discards a pulled-down offer outright
  When the discard completes
  Then the parent note is recorded as declined at that exchange hash

Scenario: 792a22i1b A rejected offer is never folded in
  Given a pending offer that an existing note contradicts
  When curation settles it
  Then it is deleted with a bare --discard and not with --into
```

## 792a22j Curation edits existing notes and never composes a new learn

Requirement: [792a22j Curation edits existing notes and never composes a new learn](#792a22j-curation-edits-existing-notes-and-never-composes-a-new-learn) Every curation action MUST be an `engram amend` call on existing files, with no `engram learn` and no handoff to
`write-memory`, and a free-text flag value MUST be attached to its flag as `--flag="value"`.

Rationale: `write-memory` composes brand-new `engram learn` calls from scratch content fields, and an offer's content already
exists as a file, so even the absent case has nothing to compose.

### 792a22j1 UAT Scenarios

```gherkin
Scenario: 792a22j1a Curation never composes a new learn
  Given curation handling any offer
  When it acts
  Then every action is an engram amend call on existing files, with no engram learn and no handoff to write-memory

Scenario: 792a22j1b A free-text value is attached to its flag
  Given curation composing an amend with a free-text value that begins with a dash
  When it runs the command
  Then the value is attached as --flag="value" and the command succeeds
```

## 792a22k A pending skill runbook note is authored, not judged

Requirement: [792a22k A pending skill runbook note is authored, not judged](#792a22k-a-pending-skill-runbook-note-is-authored-not-judged) A pending skill runbook note (a runbook note that carries `skill_hash`) MUST NOT be judged covered, near or absent and
MUST NOT be discarded. Curation MUST author its `situation`, `done_when`, `triggers` and `red_flags` from its body when it has
none, or re-check the existing ones against its body after a refresh and amend any that no longer fit, and then clear its marker
with `engram amend --clear-pending`, with no hash to pass because the note has no `xid`. It MUST NOT drop or shorten a still-valid
red flag merely to fit the 1200-byte preview budget of `engram query`, which is measured in rendered YAML bytes, and when it
reports the budget it MUST measure it in rendered bytes.

Rationale: a skill note is a registration offer and not a served write, so another note covering similar ground is no reason to
lose it ([skill-runbook-registration](./skill-runbook-registration.md)). `engram show` always returns every red flag, so the budget limits only the preview.

### 792a22k1 UAT Scenarios

```gherkin
Scenario: 792a22k1a A new skill note gets its runbook fields
  Given a pending skill runbook note with no situation, done_when, triggers or red_flags
  When curation handles it
  Then it amends the note to add those fields, authored from the note's body, clears the marker, and does not discard the note

Scenario: 792a22k1b A refreshed skill note is re-checked, not discarded
  Given a pending skill runbook note that already has runbook fields
  When curation handles it
  Then it amends any field that no longer matches the body, clears the marker, and does not discard the note even if another note covers similar ground

Scenario: 792a22k1c Curation does not trim valid red flags to fit the preview budget
  Given a skill runbook note whose still-valid red_flags render to more than 1200 bytes
  When curation re-checks it
  Then curation keeps every still-valid entry, and any budget figure it reports is the rendered-YAML byte count and not the sum of the plain strings

Scenario: 792a22k1d Clearing a skill note's marker makes it live
  Given a pending skill runbook note
  When engram amend --target <note> --clear-pending runs
  Then the note is live and engram query can return it
```

## 792a22l A bookkeeping step names the version that was judged

Requirement: [792a22l A bookkeeping step names the version that was judged](#792a22l-a-bookkeeping-step-names-the-version-that-was-judged) Every bookkeeping step (`--clear-pending`, `--discard`, `--discard --into`) on a note that carries `offer.origin`, pending or live,
MUST pass `--expect-hash` with the exchange hash that `engram show` printed on its first line, and the binary MUST refuse the
step without it and when the note's current hash is not that hash, changing nothing. When the hash no longer matches, because
the offer was updated in place after it was read, curation MUST re-read the whole offer, judge it again from scratch and pass
the new hash, and MUST NOT copy the new hash from the error or from the first line of `engram show` without re-judging.

Rationale: an offer can be updated in place after a curator judged it, so acting on the old judgment would accept or discard
content nobody reviewed. A note without an `xid` has no such line and needs no hash.

### 792a22l1 UAT Scenarios

```gherkin
Scenario: 792a22l1a A step on a served offer without the hash is refused
  Given a pending offer that carries offer.origin
  When engram amend --target N --clear-pending runs without --expect-hash
  Then it fails telling the curator to pass the exchange hash that was judged, and N is unchanged

Scenario: 792a22l1b An offer updated after judgment is not accepted
  Given a curator judged pending offer N at hash H1 and the child then updated N in place to H2
  When engram amend --target N --clear-pending --expect-hash H1 runs
  Then it fails and N stays pending with the H2 content

Scenario: 792a22l1c A changed offer is judged again
  Given a failed --expect-hash because the offer changed
  When the curator proceeds
  Then it re-reads the whole offer, judges it again, redoes the actions of the row it lands on and passes the new hash

Scenario: 792a22l1d A note with no exchange header needs no hash
  Given a pending skill runbook note, which has no xid
  When engram amend --target <note> --clear-pending runs without --expect-hash
  Then it succeeds
```

## 792a22m Folding an offer keeps its identity and reads a CRLF note as LF

Requirement: [792a22m Folding an offer keeps its identity and reads a CRLF note as LF](#792a22m-folding-an-offer-keeps-its-identity-and-reads-a-crlf-note-as-lf) `engram amend --target O --discard --into E` MUST delete O and its sidecar after recording O's basename and aliases in E's
`aliases` and merging O's parent links into E's, MUST fail without changing anything for `--into` without `--discard`, for an
`--into` that names no note and for an `--into` that names O, and MUST NOT re-stamp E's identity, queue an offer or record a
decline. It MUST read E as its LF form and write it as LF only when the fold changes it, and the judged-version check MUST run on
the LF form, which for a note whose frontmatter is LF is the hash `engram show` prints.

Rationale: the fold is for an offer that curation judged covered or near, where E already holds the content, so only its names
and links grow; the links keep the parent note from being pulled down again and the aliases keep a later reference to O
resolving. A hand-edited or Windows-edited note with CRLF line endings must still be foldable.

### 792a22m1 UAT Scenarios

```gherkin
Scenario: 792a22m1a A fold unions aliases and links
  Given offer O with an alias and parent links, and an existing note E
  When engram amend --target O --discard --into E runs
  Then O and its sidecar are gone, and E's aliases include O's basename and O's alias and E holds O's links, keeping its own primary link when it had one

Scenario: 792a22m1b A bad --into is refused
  Given --into without --discard, an --into that names no note, and an --into that names the offer itself
  When each amend runs
  Then each fails and nothing changes

Scenario: 792a22m1c A fold does not re-stamp identity
  Given an offer folded into E
  When the fold completes
  Then E's repo, user and vault, xid and content are unchanged and no offer is queued

Scenario: 792a22m1d A fold converts a CRLF existing note
  Given an existing note E whose lines end in \r\n and a fold that changes E's aliases or parent links
  When engram amend --target O --discard --into E runs
  Then the fold succeeds, E is written with no \r\n, and O and its sidecar are gone

Scenario: 792a22m1e A fold that changes nothing leaves E alone
  Given an offer whose names and links E already holds
  When the offer is folded into E
  Then E is not written and O is deleted
```

## 792a22n The outcome of a curation is not reported back

Requirement: [792a22n The outcome of a curation is not reported back](#792a22n-the-outcome-of-a-curation-is-not-reported-back) The served API's response to a `learn` that creates or updates a pending offer MUST confirm only that the offer was received,
and MUST NOT report the offer's eventual curation outcome. No later notification of the outcome MUST be sent to the offering
caller.

Rationale: the response is a receipt ([792a20d A served learn lands as a pending offer and returns a receipt](./vault-serving.md#792a20d-a-served-learn-lands-as-a-pending-offer-and-returns-a-receipt)), and curation happens later on the host, so the caller learns what became of its
offer only by reading the parent's notes or by the next merged query.

### 792a22n1 UAT Scenarios

```gherkin
Scenario: 792a22n1a The server responds before curation happens
  Given a served learn that creates a pending offer
  When the server responds
  Then the response is returned before curation has run, and no subsequent message reports the offer's disposition
```

## 792a22o Pending offers are surfaced at three points

Requirement: [792a22o Pending offers are surfaced at three points](#792a22o-pending-offers-are-surfaced-at-three-points) `engram query` MUST include a pending-offer flag in its payload on every call, true exactly when a pending offer exists in
the vault it answers for, and when the flag is true MUST also include a `pending_offers_hint` carrying the curate instruction,
omitting both when no offer is pending, so that a payload without pending offers is unchanged. When present, `pending_offers`
and `pending_offers_hint` MUST be the first keys after `version`, before `items`, so that a truncated preview of a large payload
still shows them. `engram update` MUST include a notify-only notice when pending offers exist. `engram learn`, `engram amend`
and `engram resituate` MUST log a warning-level nudge when pending offers exist. The update notice, the write-path nudge and the
payload hint MUST each carry the same one-line instruction, which says that reviewing pending offers is expected vault upkeep
and not an extra, that the agent is to curate them after it finishes the user's request without asking, and gives the
copy-pasteable command `engram query --text="curate pending offers" --phrase="..."` in the attached flag form. Following that command
against a vault that holds the curate runbook MUST return it with `trigger` provenance. A merged or served query MUST report the
local vault's flag only.

Rationale: pending offers are lower-volume and costlier to leave stale than a vocabulary refit backlog, so there are three
signals, one on each place an agent already looks. An agent that met the cue in the middle of unrelated work declined it every
time as "you didn't ask for that", so the instruction says that curation is expected. A runbook has no description the
harness reads, so each cue carries the query that surfaces it.
`pending_offers` and its hint are the first keys of the payload because an unrequested-work cue at the tail of
a 94 to 112 KB first payload was never seen: the shell preview saved the rest to a file, and 0 of 3 agents saw
the hint. Shrinking the payload was rejected as a separate lever, and so was a general `notices:` list, which
would be a new shape for every consumer.

### 792a22o1 UAT Scenarios

```gherkin
Scenario: 792a22o1a The query payload reflects current state
  Given a vault with and without a pending offer
  When engram query runs on each
  Then the payload's pending-offer flag matches whether a pending offer exists

Scenario: 792a22o1b The payload carries the hint when offers are pending
  Given a vault that holds pending offers
  When engram query runs
  Then the payload has pending_offers true and a pending_offers_hint whose text is the same instruction the update notice and the write-path nudge carry

Scenario: 792a22o1c The flag and hint lead the payload
  Given a payload larger than 100 KB and pending offers
  When engram query renders it
  Then pending_offers true and pending_offers_hint appear before items, and the first 1500 bytes of the payload contain the hint

Scenario: 792a22o1d The instruction says curation is expected upkeep
  Given the curate instruction on any of the three surfaces
  When it is read
  Then it says reviewing pending offers is expected vault upkeep and not an extra, says to curate them after finishing the user's request without asking, and gives the engram query --text="curate pending offers" command

Scenario: 792a22o1e The payload has no hint without offers
  Given a vault with no pending offer
  When engram query runs
  Then the payload contains neither pending_offers nor pending_offers_hint and is byte-identical to a payload produced before the hint existed

Scenario: 792a22o1f A served query carries the hint
  Given a served query while pending offers exist on the host
  When it is answered
  Then the response body is the same YAML a local query would print, with pending_offers true and the hint

Scenario: 792a22o1g Update surfaces a notice
  Given a vault that holds pending offers
  When the user runs engram update
  Then its report includes a notify-only notice naming the pending offers and the engram query --text="curate pending offers" command

Scenario: 792a22o1h The write-path nudge fires
  Given a vault that holds pending offers
  When engram learn, engram amend or engram resituate completes
  Then a warning-level log line notes their presence and carries the same query

Scenario: 792a22o1i The notice's command surfaces the runbook
  Given a vault that contains the curate runbook
  When an agent runs the command a notice prints
  Then the curate runbook appears in the result with trigger provenance
```
