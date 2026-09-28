---
name: curate
description: >
  Use when an engram vault holds pending offers awaiting review — engram query's payload shows
  pending_offers: true, an engram update notice names pending offers, a write-path warning nudge
  fires after engram learn/amend/resituate, or a child's offer or a pulled-down parent note is
  pending. Also use when explicitly asked to curate, review, or triage pending offers in a vault.
  Runs host-local only; never invoked from inside engram serve's request handling.
---

# Curate — Judge Pending Offers Against the Host Vault

An offer (a served `engram learn` from a child, or a parent note pulled down by `engram activate`)
lands as a **pending offer**: a real note file, marked `pending: true`, that hasn't yet been
reviewed against the vault's existing (non-pending) notes. Curation is that review — the same
agent-judged covered/near/absent reasoning `recall`'s Step 2.5 performs against query candidates,
applied instead to a pending offer against the host vault's existing notes.

**Host-local only, on your own initiative.** `engram serve`'s only job on an offer is stamp the
declared identity, persist the pending marker (or update the same origin's pending offer in place),
respond; a pull-down's only job is fetch and persist the pending copy — judgment happens later, off
the request path, never synchronously inside the HTTP request that created the offer. Invoke this
skill reactively, whenever you notice one of the three surfacing signals (query payload flag, update
notice, write-path nudge), or whenever explicitly asked to curate.

## Special Case — Pending Skill Runbook Notes

A pending runbook note that carries a `skill_hash` field is a skill's registration offer, not a
served write. Handle it separately from regular pending offers — never judge it covered/near/absent,
never discard it, and always keep it in the vault.

**For new notes (no runbook fields yet):** Author its `situation`, `done_when`, `triggers`, and
`red_flags` from the skill's body:
- `situation`: when would an agent reach for this skill (retrieval-shaped phrasing, e.g. "reviewing code for style violations")
- `done_when`: the observable end state (e.g. "all flagged issues fixed or documented")
- `triggers`: literal user phrasings that should fire it (e.g. "code review", "style check"; may be empty)
- `red_flags`: the skill's key failure modes, total rendered under 1200 bytes

Amend the note: `engram amend --target <basename> --situation "<text>" --done-when "<text>" --red-flag "<text>" --trigger "<phrase>"` (repeat `--trigger` and `--red-flag` for multiple entries). Then clear the marker: `engram amend --target <basename> --clear-pending` (a skill note has no xid, so
`engram show` prints no `# exchange_hash` line and there is no `--expect-hash` to pass).

**For refreshed notes (fields already present):** Re-check each field against the skill's current body.
Amend any that no longer fit, keeping `red_flags` under the 1200-byte rendered cap. Then clear the
marker: `engram amend --target <basename> --clear-pending`.

## Step 1 — Find pending offers

`engram query` excludes pending offers from its results by design — it can only tell you THAT some
exist (`pending_offers: true` in its payload), not which ones. There is no CLI listing for them
(`engram count --group-by pending` only counts). Scan the vault directly instead:

```bash
grep -l '^pending: true$' <vault>/*.md
```

Read each match in full with `engram show <offer>` — that's the offer's claim. Its first line,
`# exchange_hash: xh1:…`, is **the version you judged**: write it down with your judgment. (A note
without `xid` has no such line; then there is no hash to pass.)

## Step 2 — Judge each offer against the host vault's existing notes

**An offer carrying `offer.for: <note>` is an amend-offer for that note: judge it against that note
first** (`engram show <note>`), before any query. It is usually near (it adds a claim) or covered.

Otherwise, `engram query --phrase "<offer's situation>"` finds related existing notes the normal way
(it already excludes offers, so every result is a real candidate to judge against).

Every bookkeeping step (`--clear-pending`, `--discard`, `--discard --into`) on an offer whose
`engram show` printed an `# exchange_hash` line passes `--expect-hash <the version you judged>`. The
binary refuses it on a served offer without the hash.

| Outcome | Criterion | Action |
| --- | --- | --- |
| **Covered** | an existing note already states the offer's claim, no material omission | reinforce it: `engram amend --target <existing> --activate --chunk-source <ids>` (carry forward any chunk-source the offer already had) [`--supersedes ...` if it corrects a different, outdated note] — then `engram amend --target <offer> --discard --into <existing> --expect-hash <hash>` |
| **Near** | overlaps an existing note's topic but adds ≥1 substantive claim the existing note omits | fold the new claim in: `engram amend --target <existing> --chunk-source <ids> --subject/--predicate/--object` (or `--behavior/--impact/--action`) [`--supersedes ...` if correcting] — then `engram amend --target <offer> --discard --into <existing> --expect-hash <hash>` |
| **Absent** | no existing note addresses the offer's situation | accept as-is, clear its own marker: `engram amend --target <offer> --clear-pending --expect-hash <hash>`. On a vault that has its own parent, accepting a served offer offers it onward automatically — nothing more to do |
| **Rejected** | the offer is wrong: an existing note (or a user correction) contradicts it, or it is advice you would never apply | `engram amend --target <offer> --discard --expect-hash <hash>` — a bare `--discard`, **never `--into`**. For a pulled-down note this records a decline, so it is not pulled again unless the parent changes it. Folding it `--into` the note that contradicts it would record the bad note as covered by it |

**`--into` is what keeps the offer's identity.** It adds the offer's basename and aliases to the
existing note's `aliases` and moves its parent links onto that note, so a merged query dedupes the
parent's copy and an unchanged parent note is never pulled down again. A bare `--discard` of a
covered or near offer loses all of that.

**If `<existing>` is itself a pending offer**, judge and settle it first (its own row above, with
its own hash), then fold the second offer into it.

**When `--expect-hash` fails** ("the note changed since it was judged"), the offer was updated in
place after you read it — new content nobody has judged. Do not copy the new hash out of the error
or out of `engram show | head -1`. Re-read the whole offer with `engram show`, judge it again from
scratch (a new claim can turn covered into near, or add a claim you must fold in), redo that row's
actions, and pass the new hash.

Judge content, never a cosine/similarity score alone — recall's own documented mistake, and the
same trap here: a high score can still be an unrelated false positive, a low one can still be the
same claim in different words.

**Never hand off to `write-memory`.** Every curation action above is an `engram amend` call,
composed and executed directly. `write-memory`'s contract only composes brand-new `engram learn`
calls from scratch content fields — an offer's content already exists as a file, so there's
nothing to compose from scratch, not even in the absent case.

## Step 3 — Why covered and near both fold the offer away (`--discard --into`)

recall's Step 2.5 never leaves a leftover file behind: its "candidate" is an idea in the agent's
head at query time, not yet written anywhere — covered/near there just means "don't write it" or
"write it into the existing note instead." A pending offer is different: it's already a file.
Leaving it in place with `--clear-pending` after its content has been folded elsewhere (near) or
found already covered (covered) would create a second, live, redundant note — exactly the
duplication curation exists to prevent. `--clear-pending` is reserved for the one outcome where the
offer's content is genuinely new: absent.

## Step 4 — Verify

Run `engram check` after curating — confirms vault invariants (Luhmann structure, wikilink graph)
still hold. `engram query` should no longer show the curated offers as pending; an absent offer you
accepted should now surface in normal results.

## Red flags — STOP and re-read

| Sign you're off-script | What you should be doing |
| --- | --- |
| You rewrote the OFFER note's own content | Near enriches the EXISTING note; the offer itself is only ever discarded or marker-cleared, never content-amended |
| You cleared an offer's marker after folding or discarding its content elsewhere | Only absent clears the marker and keeps the note; covered/near both end in `--discard --into <existing>` |
| You ran a bare `--discard` on a covered or near offer | `--discard --into <existing>` — a bare discard loses the offer's basename, aliases and parent links |
| You folded a wrong or contradicted offer `--into` the note that contradicts it | Rejected is a bare `--discard`; `--into` means "covered by" |
| A bookkeeping amend without `--expect-hash` | Pass the `# exchange_hash` of the version you judged, every time |
| After a hash mismatch you retried with the new hash without re-judging | Re-read the whole offer and judge it again; the update may carry a claim you must fold in |
| You judged an `offer.for` offer by query alone | Judge it against the `offer.for` note first |
| You ran `engram learn` for the absent case | The note already exists as the offer — `--clear-pending` is the entire action, no new write |
| You handed a judgment off to `write-memory` | write-memory composes brand-new `engram learn` calls only; curation is self-contained `engram amend` |
| You used `engram query` to find pending offers | Query excludes them by design — scan the vault's `.md` files directly for `pending: true` |
| You applied a cosine threshold to decide covered/near/absent | Judge content — recall's own documented mistake, same trap here |
| You curated from inside a served HTTP request | Host-local only, invoked separately — never synchronous with `engram serve` |
