## Why

`fix-show-amend-reparent-frontmatter` made rename, adopt, refresh and resituate follow the shared frontmatter rules, and left three gaps as Non-Goals or review notes, now filed as #789:

- `engram amend` still refuses a CRLF note ("no parseable frontmatter"), while every other rewrite converts it to LF.
- Identity backfill (`engram update --backfill-identity`) still re-renders frontmatter through the typed model, so it drops keys the model does not define (for example `luhmann_old:`). Its own spec already says it leaves all other fields unchanged.
- When an offer receipt is refused because the local note's `parent:` carries a YAML anchor, the outbox entry stays queued, so the same offer is re-sent to the parent on every drain until someone removes the anchor.

Joe asked for all three to be finished as one change on 2026-10-02.

## What Changes

- **Amend converts CRLF.** Amend reads every note it touches through `toLF`, as rename, adopt, refresh and resituate do. A note amend already writes (the target of a content or bookkeeping amend, or the existing note of a fold) is written as LF in that same atomic write and re-embedded, because conversion changes its content hash. A note amend does not write is never converted. The `--expect-hash` check runs on the LF form, the form amend edits.
- **Identity backfill edits frontmatter as YAML nodes.** It sets `repo:`, `user:` and `vault:` (and re-emits `created:` in its quoted form, as before) through the shared node edit in `frontmatter_node.go`. Keys the typed model does not define, and anchors on keys it does not edit, survive. An anchored key it would edit refuses that note untouched (`errFrontmatterAnchoredKey`), the rewritten frontmatter is decoded again before the write (`errFrontmatterUndecodable`), and the run stamps every other note and then fails naming each refused note. A note without unknown keys or anchors is written byte-for-byte as before.
- **A refused receipt needs attention instead of being re-sent.** When the parent accepts an offer but its receipt cannot be recorded because the local note's frontmatter refuses the edit (an anchored `parent:`, or frontmatter that does not decode), the outbox entry moves to a new `attention` state. It keeps the receipt and the hash that was sent. The drain warns once, naming the note and the reason, and never re-sends that entry. Each later drain retries recording the kept receipt locally, without contacting the parent. Once the note decodes and is anchor-free, the receipt is recorded and the entry finishes as an accepted one would. The `engram update` outbox notice lists entries that need attention.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `vault-note-identity`: amend converts the CRLF notes it writes to LF instead of refusing them. Identity backfill keeps every key it does not set, including unmodeled keys, with the shared anchor refusal and decode-again guard.
- `vault-parent-offers`: a receipt the local note refuses puts the entry in a terminal `attention` state that is never re-sent, is reported, and resumes on its own once the note can take the receipt.

## Impact

- **Code (`internal/cli`):** `amend.go` and `amend_fold.go` (read sites, re-embed on conversion), `identity_backfill.go` (node edit), `frontmatter_node.go` (a shared typed-edit helper), `offer_receipt.go`, `outbox.go` and `offer_wiring.go` (the `attention` state, the warning and the notice).
- **Not changed:** `embed.SplitFrontmatter`, `engram show`, the exchange hash, the parent's `/learn` handling, and every outbox state other than the new one. An older binary reading an outbox with an `attention` entry treats it as queued and re-sends it, which is the pre-change behaviour.
- **Docs:** `docs/ROADMAP.md` row 9, the `outbox` entry in `docs/GLOSSARY.md`, and ADR D6 in `docs/architecture/adr.md`.
- **Vault:** no migration. Notes without CRLF, unknown keys or anchors are written byte-for-byte as before by amend and backfill.
