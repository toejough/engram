---
name: curate
description: >
  Use when an engram vault holds pending offers awaiting review — engram query's payload shows
  pending_offers: true, an engram update notice names pending offers, a write-path warning nudge
  fires after engram learn/amend/resituate, or engram serve has just accepted a served write.
  Also use when explicitly asked to curate, review, or triage pending offers in a vault. Runs
  host-local only; never invoked from inside engram serve's request handling.
---

# Curate — Judge Pending Offers Against the Host Vault

A served `engram learn`/`engram amend` write lands as a **pending offer**: a real note file,
marked `pending: true`, that hasn't yet been reviewed against the vault's existing (non-pending)
notes. Curation is that review — the same agent-judged covered/near/absent reasoning `recall`'s
Step 2.5 performs against query candidates, applied instead to a pending offer against the host
vault's existing notes.

**Host-local only, on your own initiative.** `engram serve`'s only job on a write is authenticate,
stamp identity, persist the pending marker, respond — judgment happens later, off the request
path, never synchronously inside the HTTP request that created the offer. Invoke this skill
reactively, whenever you notice one of the three surfacing signals (query payload flag, update
notice, write-path nudge), or whenever explicitly asked to curate.

## Step 1 — Find pending offers

`engram query` excludes pending offers from its results by design — it can only tell you THAT some
exist (`pending_offers: true` in its payload), not which ones. There is no CLI listing for them
(`engram count --group-by pending` only counts). Scan the vault directly instead:

```bash
grep -l '^pending: true$' <vault>/*.md
```

Read each match in full (frontmatter + body) — that's the offer's claim.

## Step 2 — Judge each offer against the host vault's existing notes

`engram query --phrase "<offer's situation>"` finds related existing notes the normal way (it
already excludes offers, so every result is a real candidate to judge against). For each offer:

| Outcome | Criterion | Action |
| --- | --- | --- |
| **Covered** | an existing note already states the offer's claim, no material omission | reinforce it: `engram amend --target <existing> --activate --chunk-source <ids>` (carry forward any chunk-source the offer already had) [`--supersedes ...` if it corrects a different, outdated note] — then discard the offer: `engram amend --target <offer> --discard` |
| **Near** | overlaps an existing note's topic but adds ≥1 substantive claim the existing note omits | fold the new claim in: `engram amend --target <existing> --chunk-source <ids> --subject/--predicate/--object` (or `--behavior/--impact/--action`) [`--supersedes ...` if correcting] — then discard the now-redundant offer: `engram amend --target <offer> --discard` |
| **Absent** | no existing note addresses the offer's situation | accept as-is, clear its own marker: `engram amend --target <offer> --clear-pending` |

Judge content, never a cosine/similarity score alone — recall's own documented mistake, and the
same trap here: a high score can still be an unrelated false positive, a low one can still be the
same claim in different words.

**Never hand off to `write-memory`.** Every curation action above is an `engram amend` call,
composed and executed directly. `write-memory`'s contract only composes brand-new `engram learn`
calls from scratch content fields — an offer's content already exists as a file, so there's
nothing to compose from scratch, not even in the absent case.

## Step 3 — Why covered and near both discard the offer

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
| You cleared an offer's marker after folding or discarding its content elsewhere | Only absent clears the marker and keeps the note; covered/near both end in `--discard` |
| You ran `engram learn` for the absent case | The note already exists as the offer — `--clear-pending` is the entire action, no new write |
| You handed a judgment off to `write-memory` | write-memory composes brand-new `engram learn` calls only; curation is self-contained `engram amend` |
| You used `engram query` to find pending offers | Query excludes them by design — scan the vault's `.md` files directly for `pending: true` |
| You applied a cosine threshold to decide covered/near/absent | Judge content — recall's own documented mistake, same trap here |
| You curated from inside a served HTTP request | Host-local only, invoked separately — never synchronous with `engram serve` |
