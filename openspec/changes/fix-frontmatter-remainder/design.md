## Context

#789 is the remainder of #780. The archived change `2026-10-02-fix-show-amend-reparent-frontmatter` built the shared machinery this change reuses: `toLF` (D5), the YAML-node edit in `frontmatter_node.go` with the anchor refusal `errFrontmatterAnchoredKey` and the decode-again guard `verifyFrontmatterDecodes` (D6, D6b, ruling V3), and amend's typed-to-node edit `applyTypedEdit` (ruling V6 F4). Its Non-Goals left out amend's CRLF refusal and identity backfill's typed re-render. Its final review noted the refused-receipt re-send loop (#789 comment).

Each claim was checked against the code at `0f5d91b7`:

| Defect | Claim | Verified at | Status |
|---|---|---|---|
| 1 | amend refuses a CRLF note | `amendContent` (`amend.go:207-213`) calls `splitFrontmatter`, which needs an LF `---\n` delimiter, and returns `errAmendNoFrontmatter`. The fold's `foldedContent` refuses a CRLF note the same way through `decodeExchangeFrontmatter` (`amend_fold.go:59-63`). | **Confirmed** |
| 1 | conversion changes the sidecar's content hash | `embed.ContentHash` (`embed/hash.go:49-56`) hashes `SituationText` and `BodyText`; neither normalizes CRLF. A `--supersedes`, `--chunk-source`, `--clear-pending` or `--activate` amend does not re-embed (`reEmbedAndActivate`, `amend.go:716`). | **Confirmed**: conversion must force a re-embed |
| 2 | backfill drops unmodeled keys | `backfillFactNote` / `backfillFeedbackNote` (`identity_backfill.go:58-117`) decode into the typed doc and write `marshalFrontmatter(doc)`. The spec already says "leaving all other note fields unchanged". | **Confirmed**, and a spec violation |
| 3 | a refused receipt keeps the entry queued | `applyAcceptedStep` (`outbox.go:150-168`): an `apply` error increments `attempts`, records `last_error` and leaves the entry `queued`; the next drain's `sendOutboxEntries` sends it again. `applyReceiptToContent` (`offer_receipt.go:35-59`) refuses an anchored `parent:` through `editExchangeBlocks`. | **Confirmed** |
| 3 | the exchange hash does not cover `parent:` | ADR D3: the hash covers the offered content fields only, never exchange bookkeeping. | **Confirmed**: removing the anchor does not change the hash, so the `rejected` state's hash-change re-arm cannot detect the fix |

## Goals / Non-Goals

**Goals:**
- amend edits CRLF notes and writes them as LF, only when it writes them anyway, with fresh sidecars.
- identity backfill keeps every key it does not set, with the same anchor and decode guards as every other node edit.
- a refused receipt is reported once, is never re-sent, stays visible, and resumes by itself.
- amend and backfill write the same bytes as before for notes without CRLF, unknown keys or anchors.
- identity backfill stamps CRLF notes too, converting them to LF only when it writes them, with fresh sidecars (coordinator follow-up, 2026-10-02: no exception for this pre-existing gap).

- every reader that gates an exchange operation reads a note as its LF form, so `engram show` prints the exchange hash of a CRLF note and `--expect-hash` works on it (Joe, 2026-10-02; D4).
- amend keeps unknown sub-keys inside a `supersedes:` entry that survives a replacement (Joe, 2026-10-02; D5).
- no write ever produces `user: ""`: an undetectable user is omitted and warned once, and backfill fills it in later (Joe, 2026-10-02; D6).

- every other reader of a note (embedding, query, vocab, check, count) reads a CRLF note as its LF form (ruling W2; D8).
- no offer is sent without a user identity; such an entry waits in `attention` (ruling W2; D9).
- no write produces `vault: ""` (ruling W2; D10).

**Non-Goals:**
- None. Every known-broken behaviour found in this area is fixed in this change or is intended behaviour (below).

**Intended behaviour (not deferrals; Joe, 2026-10-02):**
- A note amend does not write is never converted: a bare `--discard` target, a fold's offer, or a fold's existing note when the fold changes nothing. Conversion happens only inside a write a path already makes, as on every other path.
- While an outbox entry needs attention, the note's new content is not offered. The note takes the kept receipt first; then the hash-change check queues it again.

## Decisions

### D1 (defect 1): amend reads notes as LF and writes them only where it already does

Apply `toLF` at amend's three read sites:

1. `readJudgedTarget` (the target of every amend). The judged-version check runs on the converted bytes, then the converted bytes are returned. So a content, bookkeeping, `--discard` or `--into` amend all see LF text.
2. `foldInto`'s read of the existing note (E).
3. `declinePulledNote`'s read for a bare `--discard`, so a CRLF pulled note still records its decline.

Writes are unchanged:

- The content path writes the target once, as now, through the existing `deps.Write` (`WriteFileAtomic` in production). A converted target is therefore always written as LF, in that single write.
- The fold writes E only when the folded text differs from E's LF text (`folded != string(intoLF)`). A fold that changes nothing leaves a CRLF E byte-identical, as the rule "only notes amend writes" requires.
- A bare discard and a fold's offer delete the note, so nothing is converted.

**Re-embed.** Conversion changes `embed.ContentHash`, so the stored vector is stale even when amend's own edit would not re-embed. `runAmendLocked` passes `contentChanged || converted` to `reEmbedAndActivate`, and the fold re-embeds E with `writeAmendedSidecar` when it wrote a converted E (a warning on failure, as amend's re-embed does). Re-embedding is the archived D5 treatment of converted notes.

**Judged version on the LF form.** `--expect-hash` and the "required on `offer.origin`" rule now run on the converted bytes, the bytes amend edits. Before this change, a CRLF-frontmatter note's `offer.origin` could not be read, so the rule silently did not apply; but the amend then failed anyway. Running on the raw bytes now would let a `--clear-pending` succeed on such a note without a judged hash. For LF frontmatter (with any body), the LF hash equals the raw hash (archived ruling V2), so this is the hash `engram show` prints.

The alternative, refusing CRLF in amend, is what #789 rejects. Changing `splitFrontmatter` is out of scope (archived Non-Goal).

### D2 (defect 2): identity backfill uses the shared typed-to-node edit

Amend's `amendFrontmatter` is generalized into a shared `nodeEditFrontmatter(mapping, before, after, body)` in `frontmatter_node.go`: `applyTypedEdit(mapping, before, encodeNode(after), "created")`, render, then `verifyFrontmatterDecodes`. Amend keeps its error prefix by wrapping the result.

`backfillFactNote` / `backfillFeedbackNote` become one generic `backfillTypedNote[T]`:

1. `parseFrontmatterMapping` then decode the typed doc from the mapping (an unparseable note still self-silences, as before).
2. If identity is not missing, return; it is not flagged.
3. `before := encodeNode(doc)`; set `repo`/`user`/`vault`; `nodeEditFrontmatter(mapping, before, doc, body)`.
4. An `errFrontmatterAnchoredKey` or `errFrontmatterUndecodable` refusal is returned to `backfillIdentity`, which records it, leaves the note untouched, and continues. After the loop it returns the stamped count and `errors.Join` of the refusals, each naming the note. `--dry-run` computes the edit too, so it reports the same refusals.
5. Otherwise write once (`WriteFileAtomic`), as now.

**CRLF (follow-up, 2026-10-02).** Backfill used to skip a CRLF note as unparseable, and `notesMissingIdentityFields` did not count it, so such a note was never stamped and never reported. Now `backfillOneNote` reads the note through `toLF` before splitting the frontmatter, and `notesMissingIdentityFields` detects on the LF form. The stamped note is written as LF in the same single `WriteFileAtomic`. When the note was converted, `writeBackfilledNote` rebuilds its sidecar through a new `IdentityDeps.Embedder` (wired from `Deps.Embed`), because conversion changes `embed.ContentHash`; a rebuild failure fails the run like a write failure. An already-stamped, refused or dry-run note is not written, so it stays CRLF. `backfillTypedNote` now only computes the edit, and `backfillOneNote` owns the dry-run gate and the write.

Collecting refusals and continuing matches the archived reparent pre-flight, which reports every refusal at once. Stopping at the first refusal would leave a whole vault unstamped behind one hand-edited note.

The node edit writes the same bytes as `marshalFrontmatter(doc)` for a note in the typed writer's form, which is how every note that predates identity was written. `TestBackfillIdentity_ParityWithPreChangeBackfill` pins this against goldens generated in-process by the code at `0f5d91b7` before any change. A binary cross-check is not possible here: `engram update --backfill-identity` runs the backfill only after its self-update and re-exec, so a `0f5d91b7` binary never runs it in isolation (verification.md, task 0.3).

### D3 (defect 3): a refused receipt puts the entry in an `attention` state

The outbox has two states, `queued` and `rejected`. `rejected` re-arms when the note's exchange hash changes, but the hash does not cover `parent:`, so it cannot detect the fix. A new third state fits the existing shape best:

- `outboxEntry` gains `state: "attention"` and two optional fields: `receipt` (the parent's receipt, as returned) and `sent_hash` (the exchange hash that was sent). `last_error` holds the reason. Older outbox files decode unchanged.
- **Transition.** In `applyAcceptedStep`, an `apply` error that is a frontmatter refusal (`receiptRefused`: `errFrontmatterAnchoredKey`, `errFrontmatterUndecodable`, `errFrontmatterNotMapping`, `errNoteNoFrontmatter`, or the new `errReceiptNoteUndecodable` that wraps a `parent:` parse failure) sets `attention`, stores the receipt and `sent_hash`, and increments `attempts`. `mergeDrainSteps` prints one warning to stderr naming the note and the reason. It is not also returned as a drain error, so the command warns once, not twice. Any other `apply` error (a write failure) keeps today's behaviour.
- **No re-send.** `sendOutboxEntries` never sends an `attention` entry. Instead it emits a local step carrying the kept receipt and `sent_hash` with a new outcome, `offerHeld`. A gone or pending note still drops the entry, as for every state.
- **Resume.** `applyDrainStep` handles `offerHeld` by retrying `apply` with the kept receipt, under the lock and on the re-read note. If it is refused again, the entry is left as it is, silently. If it succeeds, `applyAcceptedStep`'s normal ending runs: the entry is removed, or queued again (receipt and `sent_hash` cleared) when the note's exchange hash changed since it was sent. The parent cache, the backoff and the `Sent` count are not touched, because the parent was not contacted.
- **Visibility.** `queuedOfferCount` counts only `queued` entries. `outboxNotice` adds ", N need attention" to its count line only when N > 0, so an outbox without such entries prints exactly as before, and lists each as `  needs attention: <note>: <reason>`.

Alternatives considered:

- *Back off per entry (retry the send later).* The parent already accepted the offer; re-sending it is the noise #789 reports, only less often.
- *Drop the entry and record it elsewhere.* A second file to keep in step with the outbox, and the recorded receipt would have to be re-associated with the note.
- *Reuse `rejected` with a sentinel hash.* `rejected` re-arms by re-sending, which is wrong here, and the notice would call an accepted offer rejected.

Retrying the local apply is safe because the receipt is idempotent: `applyReceiptToContent` writes nothing when the note already carries the link (`updated == note.Raw`).

**Version skew.** An older binary reading an `attention` entry treats it as `queued` (its sends skip only `rejected`) and re-sends it, which is the pre-change behaviour. When it saves the outbox it drops the `receipt` and `sent_hash` fields it does not know, but keeps `state: attention`. So the new binary treats an `attention` entry without a kept receipt as `queued`: it sends it again, and a refused receipt puts it back in `attention` with a fresh receipt (and one more warning).

### D4 (follow-up 1): exchange readers read a note as its LF form

`embed.SplitFrontmatter` stays LF-only. Instead each reader that gates an exchange operation on the frontmatter applies `toLF` before splitting:

| Reader | Function | Operations it gates |
|---|---|---|
| exchange hash | `exchangeHash` (`exchangehash.go`) | show's header, `--expect-hash`, outbox change checks, rejected re-arm, loop suppression, dedupe rule 2, `offer.key`, pull-down skip and hash check, served `dedupe-keys`, raw `show` envelope hash |
| exchange-field decode | `decodeExchangeFrontmatter` (`amend_fold.go`) | show's header (`xid`), the judged-version rule (`offer.origin`), the curation fold |
| pending marker | `noteHasPendingMarker` (`offer.go`) | query's pending-offer exclusion, pending-offer warnings and update notice, served raw `show` refusal |
| exchange note scan | `parseExchangeNote` (`serve_learn.go`), used by `scanExchangeNotes` | the outbox drain, `supersedes` translation targets, served-learn lookup, merged-query dedupe, pull-down session, served `dedupe-keys` aliases |
| offer classification | `parseOfferClassNote` (`offer_classify.go`) | whether a write is offered |
| offer payload | `buildOfferPayload` (`offer_payload.go`) | the fields and body sent to the parent |
| offer receipt | `applyReceiptToContent` (`offer_receipt.go`) | recording the parent link |
| pull-down | `parsePulledSource`, `pulledDecline` (`pulldown.go`) | parsing a fetched envelope; the decline record of a discarded pulled note |
| parent-link re-check | `appendParentLinks` (`activate.go`) | activate's re-check of linked parent notes |

`exchangeHash` hashing the LF form makes every hash comparison agree for CRLF and LF forms; it supersedes the archived D5 note that converting a CRLF-frontmatter note changes its hash (it no longer does). The archived test pinning that such a note's `xid` is unreadable is replaced by one pinning that it decodes as its LF form.

None of these readers writes a note just to convert it. The one that writes, the receipt, writes the LF result in its existing single write when it has something to record (it compares against the LF form, so an already-linked CRLF note is not rewritten), and then rebuilds the converted note's sidecar (`rebuildConvertedSidecar`, shared with backfill), since conversion changes `embed.ContentHash`.

### D5 (follow-up 2): amend keeps unknown sub-keys of surviving list entries

`applyTypedEdit` replaces a changed key's value whole. For a modeled list of mappings, `listEntryIdentity` names the entry key that identifies an entry: `supersedes` → `note`. `mergeListEntries` builds the new value: each entry of the new list whose identity (the `note` value as a basename, a trailing `.md` ignored, since older notes stored `x.md` and amend writes `x`) matches an unused entry of the old list is a clone of that old entry with the new entry's keys set in it (`setMappingValueOrdered`), so keys the typed entry does not define survive; any other entry is written as encoded. The other lists amend rewrites (`sources`, `red_flags`, `triggers`, `tags`) are lists of scalars, with no sub-keys to keep. The anchor refusal still covers the whole old value first.

For a note without unknown sub-keys every kept entry ends up with exactly the encoded keys and values, in the typed order, so the output is unchanged; the 32 amend goldens still match (see D6 for the one-line exception).

### D6 (follow-up 3): no write produces `user: ""`

`user:` gets `omitempty` on the typed fact, feedback and runbook docs. The first-write sites (`engram learn`, a served learn's in-place rewrite, a pull-down) build their identity with `firstWriteIdentity`, which prints one warning (`<command>: user detection resolved empty …; writing the note without user:`) when detection is empty. A pull-down sets `user:` on its node-edited copy only when detected and otherwise deletes the key, so the parent's own `user:` never stands in. A served learn cannot reach the empty case: its identity floor already rejects an empty declared user (`TestServeLearn_EmptyDeclaredIdentity_Rejected`), so that site is covered by the shared helper and that guard.

Amend never writes `user: ""` either: when detection is empty and there is no prior value, the key stays absent. That changes 3 of the 32 amend goldens (`{fact,feedback,runbook}-minimal-clear-pending`) by exactly one line: the pre-change bookkeeping amend inserted `user: ""` into a note with no `user:` because the typed field had no `omitempty`. The parity test adjusts those goldens before comparing (the golden files stay as written at `49cfc120`; D10 extends the adjustment to their `vault: ""` line); every other byte, and all 32 exchange hashes, match.

Readers of a missing `user:`: the exchange hash excludes identity; `engram show` prints the file; the offer payload falls back to detection (`TestBuildOfferPayload_DeclaresNotesOwnIdentity`); backfill now counts a note as missing identity when it has no `user:`, and for a note that has `vault:` it fills in only `user:` (and leaves the note alone while detection is still empty), so `repo:`/`vault:` from the first write are kept.

**Ruling W1 (2026-10-02): the parity deviation is accepted.** Removing `user: ""` is the requested fix and all 32 exchange hashes match, so the 3 goldens' one-line byte difference is intended. (D10 changes the same 3 goldens' `vault: ""` line; see there.)

### D8 (ruling W2, item 1): every note reader reads a CRLF note as its LF form

Two options were weighed:

1. **Make `embed.SplitFrontmatter` CRLF-aware.** It returns sub-slices of the raw bytes, and about a dozen writers splice those slices back into a file with LF delimiters (`fmStart`/`fmEnd`): the vocab tag writers, legacy vocab stripping, reference scrubbing, the Luhmann and alias rewrites, the receipt and xid inserts, `editExchangeBlocks`. A CRLF-aware split would hand them CRLF frontmatter lines, and they would write mixed-ending files; writers that today skip a CRLF note would start writing it. Each would need its own audit and conversion anyway, so the change is wide and its failure mode (silently corrupted line endings in written notes) is worse than the bug.
2. **Route the readers through `toLF`.** The conversion happens at the reader, before the split; writers that already work on the LF form are unaffected, and a writer converts only inside a write it makes.

**Chosen: 2.** It is the narrower change, it keeps `SplitFrontmatter`'s contract (LF-only, slices of the input) that every splicing writer depends on, and it matches how every exchange reader (D4) and every rewrite path (D1, D2, archived D5) already handle CRLF.

The readers converted:

| Family | Reader | Effect |
|---|---|---|
| embedding | `embed.ExtractBody`, `embed.SituationText` (so `BodyText`, `ContentHash`) | a CRLF note embeds from its real situation and body; its content hash equals its LF form's, so conversion no longer stales a sidecar |
| shared cli split | `splitFrontmatter` (`resituate.go`), `splitFrontmatterAndBody` (`vocab.go`) | every cli reader built on them: `engram check` (M5 situation presence), `engram count` (`readNoteAttrs`), vocab definition detection, legacy term parsing, the self-tag check, and the update notices |
| query | `itemMatchesProject` (`--project`), `parseCreatedFromNote` (recency), `parseNoteQueryFrontmatter` (vault metadata: tags, supersedes, triggers) | CRLF notes filter, age and carry metadata like LF notes |
| vocab writers | `WriteVocabAssignment`, tag removal, via `splitFrontmatterAndBody` | tags are written into a CRLF note, as LF; `vocabAssignmentUnchanged` compares against the LF form so an assignment that changes nothing never writes a note just to convert it |

LF notes are untouched: `toLF` is the identity on them. The amend (32), backfill (14) and fold/receipt (11) parity tests all still pass. A CRLF note's sidecar built before this change was built from the wrong text, so `engram embed status` reports it stale once, which is correct; LF notes' sidecars are unaffected.

### D9 (ruling W2, item 2): no offer without a user identity

`buildOfferPayload` refuses with `errOfferNoUserIdentity` ("cannot offer: no user identity detected; set git user.email") when neither the note's `user:` nor detection gives a user, instead of sending `user: ""` into the parent's 400. The sender turns it into an unsent step as for any payload that cannot be built; `applyNotSentStep` moves the entry to `attention` (no kept receipt) and returns the cause once, for the merge's one warning, staying silent while the entry is already in `attention`. Because an `attention` entry without a kept receipt is treated as queued (D3, version skew), each drain tries to build the payload again — locally, the parent is not contacted — so the entry is sent as soon as detection works.

### D10 (ruling W2, item 3): no write produces `vault: ""`

`vault:` gets `omitempty` on the typed docs. Amend fills an empty `vault:` with `args.VaultName` (`fillEmptyVault`), which the `amend` target has already resolved with `resolveVaultName` (flag, then `ENGRAM_VAULT_NAME`, then `personal`); a declared `vault:` is never changed, so an accepted offer keeps its author. If no name was resolved (a direct caller), the key is omitted and `warnIfNoVault` warns. First writes warn the same way through `firstWriteIdentity`, and pull-down deletes the parent's `vault:` rather than keep it or write it empty. Backfill already resolves a non-empty name.

Parity: the same 3 amend goldens (`{fact,feedback,runbook}-minimal-clear-pending`) were written by a bookkeeping amend that inserted `user: ""` and `vault: ""` into a note with no identity. They now get no `user:` and `vault: personal`; the parity test replaces those two golden lines with `vault: personal` before comparing. Every other byte, and all 32 exchange hashes, match. This is the same class of deviation ruling W1 accepted.

### D12: final-review fixes (2026-10-02)

1. **More hand-rolled readers.** `kindFromContent` (`query.go`) cut `type:` at `\n`, so a CRLF note's kind was `runbook\r`: its triggers never entered the trigger index, a CRLF qa-question was not excluded from the main set, and query items carried `fact\r`. `capRedFlagsForPreview` (`redflags_truncation.go`) searched for `\nred_flags:\n`, so a CRLF runbook's red flags were never capped. Both now read the LF form. A sweep of every `"\n<key>:"` marker, `Cut`/`Index`/`Split` on raw content and frontmatter regex found no other reader of raw note text: the rest operate on frontmatter or body already produced by the LF-normalized split helpers (D8), or on non-note text (transcripts, chunk sources, git output).
2. **Stampable notice.** `notesMissingIdentityFields` takes whether a user is detectable now (`userDetectable`) and counts a note only when backfill can stamp it: no `vault:` either, or a user to fill in. The notice no longer asks for a backfill that would stamp nothing.
3. **No-user entries return to queued.** `requeueBuiltOffer` moves a D9 `attention` entry (no kept receipt) back to `queued` before its step is merged, unless its step is again the no-identity one, so a network failure keeps it queued; `queuedOfferCount`, used by the update notice and the backoff and unreachable warnings, then counts it.
4. **Kept receipts before the gate.** `recordKeptReceipts` is the local half of the drain: under the lock, it turns each `attention` entry that keeps a receipt into an `offerHeld` step and merges it, with no `send` at all. `drainForCommand` runs it before `gateParentContact` and `ensureParentVaultID`, so a fixed note takes its receipt even while the parent is backed off or unreachable; the send phase skips these entries.
5. **Set-aside (a), served-learn identity: no defect.** The reviewer read `rewritePendingOffer`'s `deps.learn.DetectUser` as the parent host's detection. Its only caller (`serve.go:430-436`) overrides `DetectUser`/`DetectRepo` to return the offering child's declared `user`/`repo`, so the in-place rewrite keeps the child's declared identity, as local-first-parent-sync's design intends (an accepted served offer keeps the child's declared identity). `TestServeLearn_AmendBeforeCurationUpdatesInPlace` already pins it (`user: second-declared@example.com` after the rewrite).
6. **Set-aside (b), a kept receipt from another parent: fixed.** Links count only under the configured parent's vault ID (local-first ruling S16; a link recorded for another vault never stands in). `applyHeldStep` records a kept receipt only when the parent cache belongs to the configured URL and its vault ID equals the receipt's; otherwise it discards the receipt with one warning and queues the entry for the current parent.

### D11: tests

Strict TDD, a failing test first for each defect, `t.Parallel()` everywhere, gomega assertions, a fresh fixture per subtest.

- `AmendDeps` and `IdentityDeps` are structs of closures, so the tests use closure fakes, as the archived design D7 recorded. The outbox tests use the existing `outboxEnv` / `memVaultFS` / `fakeParent` harness with a receipt applier that runs the real `applyReceiptToContent` on the in-memory note.
- Rapid properties:
  - **P1 (backfill):** for a generated fact or feedback note missing identity, with 0–3 unknown keys (scalars, lists or maps), optionally an anchor on an unedited key (`source`, aliased by another key), and a CRLF layout (none, all, frontmatter only, body only), backfill succeeds, writes no `\r\n`, every unknown key decodes to its input value, the anchored key and its alias decode to the same value, and `repo`/`user`/`vault` are set.
  - **P2 (amend CRLF):** for a generated note and amend kind, amend of the all-CRLF form equals, byte for byte, amend of the LF form, and contains no `\r\n`.
  - **P3 (supersedes sub-keys):** for any old `supersedes:` list drawn from a pool of notes, each entry optionally carrying an unknown sub-key, and any new list from the same pool, amend writes the new list in order with its modeled values; an entry that survived keeps its unknown sub-key; any other entry carries exactly `note`, `type` and `claim`.
- Parity: `TestBackfillIdentity_ParityWithPreChangeBackfill` (14 goldens from `0f5d91b7`, generated before any code change). Amend's parity is pinned by the existing `TestRunAmend_ExchangeHashParityWithPreChangeAmend` (32 goldens); this change re-generated those 32 outputs with the code at `0f5d91b7` before any change (32/32 byte-identical to the committed goldens), and a binary built from `0f5d91b7` reproduced the 26 that need no chunk index byte for byte.

## Risks / Trade-offs

- [Risk] Hashing the LF form changes the exchange hash of a note with CRLF frontmatter compared with older binaries. → Before, such a note's `xid` was unreadable, so no exchange path can have recorded a hash for its CRLF form; it now hashes as its LF form, which is what every recorded hash for that note describes.
- [Risk] A note whose entry needs attention is not re-offered when its content changes, until the anchor is removed. → The warning names the note and the reason, and the notice keeps listing it. Re-sending would fail to record again.
- [Risk] Backfill now fails the command when it refuses a note. → It still stamps every other note first, and the refusal names the note and key. Such anchors exist only in hand-written YAML.

## Migration Plan

No vault migration. After merge, run `go install ./cmd/engram` and `engram update` from the merged main checkout. Rollback is `git revert`; an outbox left with `attention` entries is read by the older binary as queued (D3, version skew).

## Open Questions

None. Each choice follows the archived change's patterns: conversion only on written notes with a sidecar rebuild (D5), the shared node edit with ruling V3's guards (D6), and collected refusals (D5 pre-flight).
