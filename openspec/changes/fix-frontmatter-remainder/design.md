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

**Non-Goals:**
- General CRLF support in `embed.SplitFrontmatter`, or in `engram show`'s `# exchange_hash:` header. A hand-converted CRLF-frontmatter note still gets no header (see Risks).
- Converting a note amend does not write: a bare `--discard` target, a fold's offer, or a fold's existing note when the fold changes nothing.
- Unknown sub-keys inside a modeled list entry that amend replaces (unchanged from the archived change, V6 F4 step 3).
- Re-offering a note's new content while its entry needs attention. The note must take the old receipt first; then the hash-change check queues it again.

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

### D4: tests

Strict TDD, a failing test first for each defect, `t.Parallel()` everywhere, gomega assertions, a fresh fixture per subtest.

- `AmendDeps` and `IdentityDeps` are structs of closures, so the tests use closure fakes, as the archived design D7 recorded. The outbox tests use the existing `outboxEnv` / `memVaultFS` / `fakeParent` harness with a receipt applier that runs the real `applyReceiptToContent` on the in-memory note.
- Rapid properties:
  - **P1 (backfill):** for a generated fact or feedback note missing identity, with 0–3 unknown keys (scalars, lists or maps), optionally an anchor on an unedited key (`source`, aliased by another key), and a CRLF layout (none, all, frontmatter only, body only), backfill succeeds, writes no `\r\n`, every unknown key decodes to its input value, the anchored key and its alias decode to the same value, and `repo`/`user`/`vault` are set.
  - **P2 (amend CRLF):** for a generated note and amend kind, amend of the all-CRLF form equals, byte for byte, amend of the LF form, and contains no `\r\n`.
- Parity: `TestBackfillIdentity_ParityWithPreChangeBackfill` (14 goldens from `0f5d91b7`, generated before any code change). Amend's parity is pinned by the existing `TestRunAmend_ExchangeHashParityWithPreChangeAmend` (32 goldens); this change re-generated those 32 outputs with the code at `0f5d91b7` before any change (32/32 byte-identical to the committed goldens), and a binary built from `0f5d91b7` reproduced the 26 that need no chunk index byte for byte.

## Risks / Trade-offs

- [Risk] A CRLF-frontmatter note that carries `offer.origin` and `xid` gets no `# exchange_hash:` header from `engram show`, so a curator cannot pass `--expect-hash` to clear or discard it. → Such a note exists only if someone hand-converts a served offer to CRLF. Any amend that writes it (for example `--activate`) converts it, after which show prints the header. Recorded, not fixed (show is out of scope).
- [Risk] Converting a CRLF-frontmatter note changes its exchange hash. → As in the archived D5: before conversion its `xid` is unreadable, so no exchange path can have recorded a hash for the CRLF form, and the converted note hashes as its LF original.
- [Risk] A note whose entry needs attention is not re-offered when its content changes, until the anchor is removed. → The warning names the note and the reason, and the notice keeps listing it. Re-sending would fail to record again.
- [Risk] Backfill now fails the command when it refuses a note. → It still stamps every other note first, and the refusal names the note and key. Such anchors exist only in hand-written YAML.

## Migration Plan

No vault migration. After merge, run `go install ./cmd/engram` and `engram update` from the merged main checkout. Rollback is `git revert`; an outbox left with `attention` entries is read by the older binary as queued (D3, version skew).

## Open Questions

None. Each choice follows the archived change's patterns: conversion only on written notes with a sidecar rebuild (D5), the shared node edit with ruling V3's guards (D6), and collected refusals (D5 pre-flight).
