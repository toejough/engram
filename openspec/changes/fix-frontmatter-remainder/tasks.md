## 0. Goldens before any code change

- [x] 0.1 With the tree at `0f5d91b7` and no code change, generate the backfill parity goldens in-process (a temporary generator test run under `targ test`, then deleted) into `internal/cli/testdata/backfill_identity_parity/`. Inputs: fact and feedback notes missing identity, in the typed writer's form, with and without `project:`, with and without `tags`/`sources`/`supersedes`/`issue`, and with an unquoted `created:`.
- [x] 0.2 With the same generator, re-generate amend's 32 parity outputs at `0f5d91b7` and confirm they are byte-identical to `internal/cli/testdata/amend_hash_parity/`.
- [x] 0.3 Build a binary from `0f5d91b7` into a scratch dir (no `go install`). Run the 26 non-chunk amend cases with it and confirm the notes match the amend goldens. Try `engram update --backfill-identity` on a scratch vault holding the backfill inputs; if the binary cannot run the backfill in isolation, record why and rely on the in-process goldens. Record both in `verification.md`. (Done: amend 26/26 byte-identical; backfill is not reachable in isolation because update self-updates and re-execs first.)

## 1. Defect 1: amend converts CRLF (design D1)

- [x] 1.1 RED: in a new `amend_crlf_test.go`, example tests: a content amend (`--object`), a `--supersedes` amend and an `--activate` amend on an all-CRLF fact note each succeed, write no `\r\n`, and re-embed (the sidecar is written); a fold into a CRLF existing note E succeeds and writes E as LF; a fold that changes nothing leaves a CRLF E byte-identical; a bare `--discard` of a CRLF pulled note records its decline. Confirm they fail on today's code. (RED: every example failed for the expected reason — content/`--supersedes`/`--activate`/`--clear-pending` and both folds with `amend: note has no parseable frontmatter`, the judged-version case with a hash mismatch, and the bare discard recorded no decline.)
- [x] 1.2 RED: rapid property P2: amend of the all-CRLF form equals amend of the LF form, byte for byte, with no `\r\n`, across fact, feedback and runbook notes and every frontmatter-writing amend kind. Confirm it fails. (RED: failed on the first draw, `fact-exchange-object CRLF (all) amend refused: amend: note has no parseable frontmatter`.)
- [x] 1.3 GREEN: `toLF` in `readJudgedTarget` (after the judged-version check moves onto the LF bytes), in `foldInto` and in `declinePulledNote`; compare the fold's output with E's LF text; force the re-embed when the target was converted; re-embed a converted E the fold wrote. Run `targ test`.

## 2. Defect 2: identity backfill uses the node edit (design D2)

- [x] 2.1 RED: in a new `identity_backfill_node_test.go`, example tests: an unmodeled key (scalar and nested map) survives; an anchor on `source` aliased by another key survives; an anchored `user:` refuses that note untouched while a second note is still stamped, and the error names the note and `user`; `--dry-run` reports the same refusal and writes nothing. Confirm they fail. (RED: the anchor and unmodeled-key cases failed because backfill dropped `x_src`, `luhmann_old` and `provenance`; both anchored-`user:` cases failed with `Expected an error, got nil`.)
- [x] 2.2 RED: rapid property P1 (unknown keys and unedited-key anchors survive backfill). Confirm it fails. (RED: `failed after 0 tests: key x_src: before agent, after <nil>`.)
- [x] 2.3 RED: `TestBackfillIdentity_ParityWithPreChangeBackfill` against the 0.1 goldens. It passes on today's code; record that, as the guard for 2.4. (Passed, 14/14, in the same run where 2.1 and 2.2 failed.)
- [x] 2.4 GREEN: move amend's `amendFrontmatter` body into a shared `nodeEditFrontmatter` in `frontmatter_node.go`; rewrite the backfill stampers as one generic node-edit stamper; collect refusals in `backfillIdentity`. Run `targ test`.

## 3. Defect 3: a refused receipt needs attention (design D3)

- [ ] 3.1 RED: in `outbox_test.go`, with a receipt applier that runs the real `applyReceiptToContent` on the in-memory note: a refused receipt moves the entry to `attention` with the receipt, `sent_hash` and reason, leaves the note untouched, and warns once naming the note and `parent`; two further drains send nothing and print nothing; after the anchor is removed, the next drain records the receipt without sending and removes the entry; a note changed since it was sent goes back to `queued`; an `attention` entry with no kept receipt is sent; a write failure still keeps the entry queued. In `offer_wiring_test.go`: the update notice counts and lists an entry that needs attention, and prints as before when there is none. Confirm they fail.
- [ ] 3.2 GREEN: the `attention` state, `receipt` and `sent_hash` fields, `receiptRefused`, `errReceiptNoteUndecodable`, the `offerHeld` outcome, the merge-time warning, `queuedOfferCount` and the notice. Run `targ test`.

## 4. Docs

- [ ] 4.1 `docs/ROADMAP.md` row 9: mark #789 done on this branch, pending merge. `docs/GLOSSARY.md` `outbox`: add the `attention` state. `docs/architecture/adr.md` D6: one sentence on the `attention` state.

## 5. Verification

- [ ] 5.1 Real-binary check on a scratch vault: `go build` into a temp dir (no `go install`), `XDG_DATA_HOME` and `--vault` in the temp dir, cwd outside any repo; never the real vault or `~/.claude`. Cover CRLF amend (content and `--supersedes`), backfill keeping `luhmann_old:` and refusing an anchored `user:`, and a refused receipt against a scratch parent (`engram serve` on a second scratch vault): one warning, no re-send on the next drain, recovery after the anchor is removed. Record results in `verification.md`.
- [ ] 5.2 `targ test`, `targ check-full` (9/9 after commit) and `openspec validate --all --strict` are green. Every MODIFIED block carries every base scenario header verbatim.

## 6. Close-out (not done by the implementing agent)

- [ ] 6.1 Review, rebase on main, `git merge --ff-only`.
- [ ] 6.2 `go install ./cmd/engram` and `engram update` from the merged main checkout.
- [ ] 6.3 Archive the change.
- [ ] 6.4 Close #789.
