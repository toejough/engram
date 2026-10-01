## 0. Decisions gate

- [x] 0.1 Joe answered design Open Questions 1-5 on 2026-10-01. The design "Decision record" holds the answers. Two answers differ from the original defaults: CRLF notes are converted to LF (D5), and `resituateTyped` is folded in (D6b). The design, the spec deltas and these tasks were revised to match, and `openspec validate --strict` is green.

## 1. #772: show returns every red_flag (design D1, D2)

- [ ] 1.1 RED: In `internal/cli/show_test.go`, add a test that runs `RunShow` on a runbook fixture whose red_flags render to more than 1200 B. Assert that every entry is in the output, in file order, and that no `OMITTED` marker appears. Run `targ test` and confirm the test fails against `show.go:60`.
- [ ] 1.2 RED: In `redflags_truncation_internal_test.go`, assert that the marker for a 15-entry fixture reads `[<N> EARLIER RED_FLAGS OMITTED — run engram show <real-basename> for all 15]`, with N equal to the number of dropped entries. Add the guard test: parse the marker's command, run `RunShow` with that basename against the same fixture, and assert that every entry, including the omitted ones, is present. Add a rapid property: for random entry lists, under the budget the output is byte-identical; over the budget, the result is at most 1200 B and keeps a contiguous run of entries ending at the newest one. Confirm all three fail.
- [ ] 1.3 GREEN:
  - Remove the cap from `show.go:60`.
  - Change `capRedFlagsForPreview` to take the basename.
  - Compute the counted marker and re-fit the kept set against the real marker length.
  - Update the three query call sites (`query.go:1652`, `query_triggers.go:61`, `merged_dedupe.go:233`).
  - Reword the constant's comment to "query preview budget (rendered YAML bytes)".
  - Update the existing `query_runbook_test.go` and `show_test.go` assertions that pinned the old behavior. Each such edit must be justified by the spec delta.
  - Run `targ test` and confirm it is green.
- [ ] 1.4 Curate SKILL.md (enumeration rows 1-2), under the `superpowers:writing-skills` skill:
  - RED: a headless `claude -p` arm curates a fixture skill note whose valid red_flags render to more than 1200 B, from a cwd outside the repo with a scratch vault. Record whether it drops an entry or measures plain bytes.
  - Apply rows 1-2 and run each row's new-present and old-absent greps.
  - GREEN: the same arm keeps every entry.
  - REFACTOR: run a pressure test.

## 2. #776: amend keeps the prior `user:` (design D3)

- [ ] 2.1 RED: In `amend_identity_test.go`, a re-stamping amend (`--object`) on a note with `user: alice`, with `DetectUser` mocked to return `""`. Assert that `user: alice` is kept and that `LogWarning` was called once with the note's basename. Use imptest for `AmendDeps`. Also add a case where the note has no prior `user:` and assert that the field is written as detected. Confirm both cases fail.
- [ ] 2.2 RED (guard for existing behavior; no code change is expected): drive the `amend` targ target through the `targets.go` wiring with no `--vault-name` and an empty `ENGRAM_VAULT_NAME`. Assert that the written note has `vault: personal`. This test is expected to PASS when first run. Record that result in the task as evidence for the refuted part of #776.
- [ ] 2.3 GREEN: Add the user-preserving stamp to the amend path only (`amendIdentity` / `applyTypedAmend` / `applyRunbookAmend`). Leave `identityStamp.stamp` unchanged for the other callers. Run `targ test` and confirm it is green.

## 3. #770: the reparent payload defers to learn's batch mode (design D4)

- [ ] 3.1 RED: In `luhmann_reparent_apply_test.go`, assert that the derive payload's `instruction` contains `learn`, `batch mode`, `--dry-run` and `distinct`, and does not contain `then re-run`. Assert that `luhmannBranchingNotice` names learn's batch mode. Confirm both fail.
- [ ] 3.2 GREEN: Rewrite `Instruction` at `luhmann_reparent_apply.go:537-545` and append the pointer to `update.go:58-59`. Leave `NextCommand` and `Fingerprint` unchanged. Run `targ test` and confirm it is green.
- [ ] 3.3 Run a consistency grep. Compare the new `instruction` text with `agent-instructions/skills/learn/SKILL.md:256-299` and with vault note 1067's red_flags (`engram show 1067`, read-only). Confirm that nothing tells the answering pass to run apply. Paste the grep output into the task.
- [ ] 3.4 Apply enumeration rows 8-9 to `docs/GLOSSARY.md` and run their greps.

## 4. #780 (a, b): rename pre-flight and CRLF→LF conversion (design D5)

- [ ] 4.1 RED, example tests in `luhmann_reparent_test.go`. Use in-memory `RenameRewriteDeps` fakes that record every `Rename` and `WriteFile`; each subtest builds its own vault map. Cover four cases:
  - **(a) Anchored luhmann.** `luhmann: &a "1050"` with `issue: *a` returns an error naming the note, with zero recorded renames or writes. A two-note map where only the second note is unsafe also writes nothing.
  - **(b) CRLF renamed note.** It is renamed and written once. The written bytes contain no `\r\n`, `luhmann:` is the new id, and the alias is recorded. Every other key decodes unchanged, and the path appears in the returned `rewritten` list.
  - **(c) CRLF referrer.** A referrer whose frontmatter `supersedes:` names a renamed note is written as LF, with `supersedes:` rewritten.
  - **(d) Untouched CRLF note.** A CRLF note that is neither renamed nor a referrer is never written.

  Confirm that all four cases fail today.
- [ ] 4.2 RED, exchange-hash and freshness pins (in `exchangehash_internal_test.go` and the reparent tests):
  - for a note with LF frontmatter and a CRLF body, `exchangeHash(pre) == exchangeHash(toLF(pre))`;
  - `decodeExchangeFrontmatter(crlfFrontmatterNote)` returns `errAmendNoFrontmatter`;
  - `exchangeHash(toLF(crlfNote)) == exchangeHash(lfAuthoredEquivalent)`;
  - after `applyReparentRenames`, `RebuildNoteSidecars` received every converted path. Use an imptest mock for the embedder, or assert `embed.ComputeState` is `StateOK` against an in-memory FS.

  The first three should pass when first run, because they pin existing hashing behavior. Record that. The last one must fail.
- [ ] 4.3 RED, rapid properties P1 and P1-CRLF (design D7). Use a CRLF-shape generator over four layouts: all LF, all CRLF, CRLF frontmatter only, and CRLF body only. Cross it with the anchor and unknown-key generators. The properties must fail on today's code.
- [ ] 4.4 GREEN:
  - Add `toLF`.
  - Apply `toLF` in `renameAndRewriteOneNote` and in `adoptRenderInput`, before any rewrite. Write only notes already being written, in the existing single atomic `WriteFile`.
  - Add converted notes to the returned `rewritten` list.
  - Implement `preflightRenames` with the sentinels `errRenameUndecodable` and `errRenameStaleLuhmann`, joined. Call it at the top of `RenameAndRewriteReferences` and from the reparent `--dry-run` path.
  - Apply enumeration row 20 and run its grep.
  - Run `targ test` and confirm it is green.

## 5. #780 (c): adopt and refresh preserve unknown keys (design D6)

- [ ] 5.1 RED:
  - Example tests: refresh and adopt each keep `luhmann_old: "12"` and a nested unknown map.
  - Rapid property P2: for LF, anchor-free input, every unknown key's decoded value survives and the skill fields are set.
  - Pending parity: adopt writes no `pending` key; refresh writes `pending: true`.
  - Confirm the tests fail.
- [ ] 5.2 GREEN: Move `setMappingValue`, `deleteMappingKeys`, `encodeNode` and `cloneNode` from `pulldown.go` into `internal/cli/frontmatter_node.go`, with no behavior change; pulldown's tests must stay green. Rewrite `applySkillNoteBody` as a node edit. Run `targ test` and confirm it is green.
- [ ] 5.3 Apply enumeration row 17 (docstrings at `skillreg_accept.go:92,172,317-322`) and run its greps.

## 6. #780 comment: resituate preserves unknown keys (design D6b)

- [ ] 6.1 RED: in `resituate_test.go`, `engram resituate` keeps `luhmann_old: "12"` and a nested unknown map on a fact note and on a feedback note. Add rapid property P3: for generated frontmatter, every key except `situation` decodes unchanged afterwards. Confirm both fail.
- [ ] 6.2 GREEN: rewrite `resituateTyped` (`resituate.go:242-264`) as a node edit with the shared `frontmatter_node.go` helpers. `parseCreated` must still refuse malformed notes untouched; the existing resituate tests must stay green. Run `targ test` and confirm it is green.
- [ ] 6.3 Apply enumeration row 19 and run its greps.

## 7. Verify

- [ ] 7.1 Run `targ check-full` and fix every finding in one pass. Do not suppress any lint finding; raise it with Joe instead.
- [ ] 7.2 Real-binary check on a SCRATCH vault. First run `go install ./cmd/engram` from this worktree. Then, from a cwd outside any repo:
  - `S=$(mktemp -d)`; every command runs with `XDG_DATA_HOME=$S/data` and `--vault $S/vault`, never the real vault.
  - **#772:** `engram learn runbook` with 15 red flags, then `engram show <basename>`. All 15 must be present with no marker. `engram query` for its situation must show a marker with a count and the real basename. Running that marker's command must print all 15.
  - **#776:** `env -u ENGRAM_VAULT_NAME engram amend --vault $S/vault --target <id> --object x` must stamp `vault: personal`. For the user case, hand-set `user: alice`, then run with `GIT_CONFIG_GLOBAL=/dev/null HOME=$S/home` so git `user.email` is empty. Record that the OS-username fallback still resolves on macOS, because it is not reachable from the binary. The unit test in 2.1 is the evidence for the empty path.
  - **#770:** create two near-duplicate top-level notes and run `engram update --reparent-luhmann --vault $S/vault`. Inspect the `instruction` field.
  - **#780:**
    - Hand-write an anchored candidate note. Apply with `--answers`. The command must refuse, naming the note, and `ls $S/vault` must be unchanged.
    - Then hand-write a CRLF candidate note (`printf` with `\r\n`) and a CRLF referrer, and apply.
      - The renamed note and the referrer are LF (`grep -c $'\r'` gives 0).
      - `luhmann:` matches the new basename.
      - `engram embed status --vault $S/vault` reports no stale sidecars.
      - An untouched CRLF note's `shasum` is unchanged. Hand-add `luhmann_old:` to a runbook note, run `engram register-skills --adopt <key>=<id> --vault $S/vault`, and confirm the key survives. Hand-add `luhmann_old:` to a fact note, run `engram resituate`, and confirm the key survives.
  - Paste the outputs into this task.
- [ ] 7.3 Fresh-context review (implementation review with argumentation). A reviewer reads the full diff against the spec deltas, design D1-D7 (including D6b) and `enumeration.md`. Every finding is fixed or rebutted until the reviewer ACKs. Keep the implementing agent alive until the ACK.
- [ ] 7.4 For every *no change* row in `enumeration.md` (rows 5, 6, 7, 10, 15, 18), re-read the cited lines against the final code and confirm that the reason still holds.

## 8. Close-out

- [ ] 8.1 Run the requirement-header collision sweep (vault note 744). Compare this change's `### Requirement:` headers against every active change in `openspec list` and against any archived change that has not been synced. Read every active sibling's tasks.md in full (vault note 757). Record the result.
- [ ] 8.2 Enumeration row 16: update `docs/ROADMAP.md` rows for #780, #772, #776 and #770, and run the greps.
- [ ] 8.3 Rebase on main and run `targ check-full` again. Merge with `git merge --ff-only`; if main moved, repeat the rebase loop. Push only after review.
- [ ] 8.4 After the merge and push, run `go install ./cmd/engram`, then `engram update`, from the merged main checkout. Confirm that the deployed curate SKILL.md is byte-identical to its source (`cmp`) at the path `engram update` names. Leave the skill-note refresh offers for Joe.
- [ ] 8.5 Close #772, #770 and #780 with `Fixes #N` in the merge commit. Close #776 with a comment that says the `vault:` part was already fixed at `targets.go:144` (`98e85e09`) and that the `user:` part is fixed here. The #780 closing note says the `resituateTyped` key drop raised in its comment was fixed here too (D6b).
- [ ] 8.6 Run `openspec archive fix-show-amend-reparent-frontmatter`. Confirm that the MODIFIED and ADDED requirements appear verbatim in the six `openspec/specs/*/spec.md` files.
