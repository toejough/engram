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

## 2b. Defect 2 follow-up: backfill converts CRLF (design D2, coordinator 2026-10-02)

- [x] 2b.1 RED: `TestBackfillIdentity_ConvertsCRLFNote` (all, frontmatter-only and body-only CRLF layouts are stamped as LF, byte-identical to the LF backfill, sidecar rebuilt; an already-stamped CRLF note is not written; a dry run counts and writes nothing), a CRLF row in `TestNotesMissingIdentityFields`, and the CRLF layout draw in property P1. (RED: the property failed with `backfill: stamped 0, err <nil>` on an all-CRLF draw; the all- and frontmatter-CRLF examples stamped 0; the body-CRLF example wrote CRLF; the detector returned false for the CRLF note.)
- [x] 2b.2 GREEN: `toLF` in `backfillOneNote` and `notesMissingIdentityFields`; `writeBackfilledNote` with the sidecar rebuild; `IdentityDeps.Embedder` wired from `Deps.Embed`. The 14 parity goldens still pass. Run `targ test`.
- [x] 2b.3 Spec, design, proposal, `enumeration.md` and `verification.md` updated; the Non-Goal removed.

## 3. Defect 3: a refused receipt needs attention (design D3)

- [x] 3.1 RED: in a new `outbox_attention_test.go`, with a receipt applier that runs the real `applyReceiptToContent` on the in-memory note: a refused receipt moves the entry to `attention` with the receipt, `sent_hash` and reason, leaves the note untouched, and warns once naming the note and `parent`; two further drains send nothing and print nothing; after the anchor is removed, the next drain records the receipt without sending and removes the entry; a note changed since it was sent goes back to `queued`; an `attention` entry with no kept receipt is sent; a write failure still keeps the entry queued (the existing `TestDrainOutbox_ReceiptWriteFailureKeepsEntry`). Through the production wiring (`ExportUpdateExchange`): the update notice counts and lists an entry that needs attention, and prints as before when there is none. Confirm they fail. (RED: the refused-receipt and requeue tests failed with `outbox: recording the receipt on 1.2026-09-27.a: offer receipt: frontmatter key the edit replaces carries a YAML anchor: parent` returned as a drain error and the entry left queued; the wiring test's notice read `1 offer(s) queued, 0 rejected` with no attention count. The no-receipt `attention` entry test passed, since today's code sends any non-`rejected` entry; it is kept as the version-skew guard. The existing write-failure test stayed green.)
- [x] 3.2 GREEN: the `attention` state, `receipt` and `sent_hash` fields, `receiptRefused`, `errReceiptNoteUndecodable`, the `offerHeld` outcome, the merge-time warning, `queuedOfferCount` and the notice. Run `targ test`.

## 3b. Follow-ups (Joe, 2026-10-02; design D4–D6)

- [x] 3b.1 RED (exchange readers, D4): `TestShow_ExchangeHashHeaderForCRLFNote`, `TestExchangeHash_CRLFFrontmatterHashesAsLF`, `TestNoteHasPendingMarker_CRLF`, `TestDrainOutbox_FindsCRLFNote`, `TestActivate_PullsCRLFParentNote`. (RED: show printed `---` as its first line, no header; the CRLF hash differed from the LF hash; the pending marker read false; the drain sent nothing (`sentXIDs` empty), dropping the entry; activate failed with `the parent note has no frontmatter`.)
- [x] 3b.2 RED (supersedes sub-keys, D5): `TestRunAmend_SupersedesKeepsUnknownSubKeysOfKeptEntries` and property P3 `TestRunAmend_SupersedesSubKeysProperty`. (RED: the kept entry lost `x_reason`.)
- [x] 3b.3 RED (first-write user, D6): `TestLearn_OmitsEmptyUser`, `TestActivate_PullDownOmitsEmptyUser` (wiring, with git `user.email` and the OS username lookup both failing), `TestBackfillIdentity_FillsAnOmittedUser`; `TestLearn_StampsDetectedUser` guards the normal case; the served-learn site is guarded by the existing `TestServeLearn_EmptyDeclaredIdentity_Rejected`. (RED: learn wrote `user: ""` and no warning; the pulled copy kept a `user:` key; backfill neither flagged nor filled a note with `vault:` but no `user:`.)
- [x] 3b.4 GREEN: `toLF` in the nine readers of D4's table, receipt write converts and rebuilds the sidecar (`rebuildConvertedSidecar`); `mergeListEntries` in `applyTypedEdit`; `user,omitempty`, `firstWriteIdentity`, pull-down deletes an undetected `user:`, backfill flags by `user:` and fills only `user:` when `vault:` is present. Pinned tests updated with the spec delta: `TestRenderFrontmatter_{Feedback,Runbook}` (no `user: ""`), `TestRunAmend_EmptyUserDetectionNoPriorUserOmitsUser`, `TestDecodeExchangeFrontmatter_CRLFFrontmatterReadsAsLF`, and the amend parity test (3 goldens minus their `user: ""` line). Run `targ test`.
- [x] 3b.5 Spec deltas (vault-offer-curation show header; vault-parent-offers ADDED exchange-reader requirement; vault-note-identity user, amend re-stamp, exchange-fields and backfill blocks), design, proposal, enumeration and verification updated.

## 4. Docs

- [x] 4.1 `docs/ROADMAP.md` row 9: mark #789 done on this branch, pending merge. `docs/GLOSSARY.md` `outbox`: add the `attention` state. `docs/architecture/adr.md` D6: one sentence on the `attention` state.

## 5. Verification

- [x] 5.1 Real-binary check on a scratch vault: `go build` into a temp dir (no `go install`), `XDG_DATA_HOME` and `--vault` in the temp dir, cwd outside any repo; never the real vault or `~/.claude`. Cover CRLF amend (content and `--supersedes`), backfill keeping `luhmann_old:` and refusing an anchored `user:`, and a refused receipt against a scratch parent (`engram serve` on a second scratch vault): one warning, no re-send on the next drain, recovery after the anchor is removed. Record results in `verification.md`.
- [x] 5.2 `targ test`, `targ check-full` (9/9 after commit) and `openspec validate --all --strict` are green. Every MODIFIED block carries every base scenario header verbatim. (`targ test` exit 0; `targ check-full` PASS:9; `openspec validate --all --strict` 51 passed, 0 failed; scenario coverage: all 14 + 4 + 4 + 1 base scenario headers present in the four MODIFIED blocks.)

## 6. Close-out (not done by the implementing agent)

- [ ] 6.1 Review, rebase on main, `git merge --ff-only`.
- [ ] 6.2 `go install ./cmd/engram` and `engram update` from the merged main checkout.
- [ ] 6.3 Archive the change.
- [ ] 6.4 Close #789.
