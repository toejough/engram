## Context

This change fixes four open defects together, with Joe's approval of 2026-10-01. Each claim was checked against the code at `b48d6557` before any design work. The table gives the result for each.

| Issue | Claim | Verified at | Status |
|---|---|---|---|
| #772 | show truncates red_flags | `internal/cli/show.go:60` calls `capRedFlagsForPreview` | **Confirmed** |
| #772 | The marker points back at show and gives no count | `internal/cli/redflags_truncation.go:11`. The marker is the fixed text `"[EARLIER RED_FLAGS OMITTED — run \`engram show <basename>\` for the full list]"` with no count. The basename stays the literal `<basename>`. | **Confirmed** |
| #772 | Query caps red_flags as well | `query.go:1652`, `query_triggers.go:61`, `merged_dedupe.go:233` | Confirmed. This is intended, and it stays. |
| #776 | amend writes `vault: ""` | `amend.go:239` passes `args.VaultName` through as is. But the only caller of `RunAmend` is `targets.go:144`, which already runs `a.VaultName = resolveVaultName(a.VaultName, deps.Getenv)` (commit `98e85e09`, 2026-08-20, before the issue was filed). A real-binary check on a scratch vault, with `ENGRAM_VAULT_NAME` unset and no flag, stamped `vault: personal`. | **Refuted, already fixed.** No code change. A regression test pins the behavior. |
| #776 | amend writes `user: ""` | `amend.go:236-240` and `identity.go:28-34`: `stamp` overwrites unconditionally. `detectUser` (`identity.go:64-90`) returns `""` when both git and the OS username lookup fail. `learn.go:298` (`yaml:"user"`, no omitempty) writes a literal `user: ""`. | **Confirmed in code.** In practice this needs both git `user.email` and `os/user.Current` to fail, so it is rare. |
| #770 | The payload says "re-run" apply, never names learn, and skips `--dry-run` | `luhmann_reparent_apply.go:537-545`. The notice at `update.go:58-59` doesn't name learn either. No test asserts on `Instruction`. | **Confirmed** |
| #780(a) | Anchor dropped on reparent, with no validation before the write | `luhmann_reparent.go:284-304` replaces the `luhmann:` lines as text, so `&a` is lost. `renameOneNote` (`:250-275`) renames and writes without decoding again. Adopt is protected only as a side effect of `checkAdoptRenders`. | **Confirmed** |
| #780(b) | A CRLF note silently skips the rewrite | `embed/hash.go:98-119`: the delimiter must be the LF-only `"---\n"`. `rewriteLuhmannIDField` (`:285-288`) and `appendAliasField` (`:134-137`) return the content unchanged, and `renameOneNote` still renames the file and its sidecar. Adopt already refuses CRLF (`applySkillNoteBody` → `errSkillNoteNoFrontmatter`). | **Confirmed** for reparent |
| #780(c) | Adopt and refresh drop unknown keys | `skillreg_accept.go:323-347`: plain `yaml.Unmarshal` into `runbookFrontmatterDoc` (`learn.go:~415-447`), then `marshalFrontmatter`. The docstrings at `:92` and `:172` claim that "every other frontmatter field" is preserved. | **Confirmed** |

The capabilities affected are `recall-runbook-surfacing`, `vault-offer-curation`, `vault-note-identity`, `update-reparent-luhmann-batch`, `update-flat-vault-luhmann-notice` and `skill-runbook-registration`.

## Goals / Non-Goals

**Goals:**
- `engram show` is a full-fidelity source of every red_flag. The query marker gives a count and a command that works.
- A re-stamping amend never blanks a non-empty `user:`.
- The reparent payload, the notice and the learn batch mode agree.
- No rename or rewrite writes undecodable frontmatter or leaves a stale `luhmann:`. Adopt and refresh keep keys they don't model.

**Non-Goals:**
- General CRLF support in `embed.SplitFrontmatter`. It has more than 9 callers, including exchange-hash and vocab code, and vault notes are written as LF by engram.
- `resituateTyped` dropping unknown keys. This is the same class of bug, reported in a comment on #780. See Open Questions.
- `user:` fallback on first-write paths (`learn.go:1046`, `serve_learn.go:257`, `pulldown.go:351`). They have no prior value to keep.
- Any edit to the learn SKILL.md or vault note 1067.
- Any change to the order or layout of show's output.

## Decisions

### D1 (#772): show stops truncating, and query's marker gives a count and a basename (issue Option A)

- Remove `capRedFlagsForPreview` from `show.go:60`.
- Change the signature to `capRedFlagsForPreview(content, basename string)`. The marker becomes `    - "[<N> EARLIER RED_FLAGS OMITTED — run engram show <basename> for all <M>]"\n`, with the real basename substituted. All three query call sites already hold the basename: `hit.note.Basename`, `basename` and `local`'s basename.
- Size the kept entries against `budget - len(marker)`. N and M change the marker's length, so compute the marker after choosing the kept set, then shrink the set again if the real marker does not fit. Inputs under the budget stay byte-identical.

Option B was rejected. It keeps show bounded and adds `--red-flags` or `--full`. But route's list alone renders to about 4 KB, so a second bounded command recreates the same loop. The #763 risk (the harness preview cutting a large show output) is real. However, a harness that persists a large output keeps the full text in the persisted file. Truncating inside engram is the only change that loses the entries completely. Shim rules 6 and 7 already describe show as the full-body source, so the shim needs no edit.

The guard test is part of the TDD for this decision. It parses the marker's command, runs `RunShow` with that basename against the same fixture, and asserts that every entry is present.

### D2 (#772): the 1200 bytes becomes a "preview budget"

- Rename the constant comment.
- Reword the `vault-offer-curation` requirement and `curate/SKILL.md:37,43` to say: "`engram query` preview budget, measured in rendered YAML bytes. Do not trim a valid red flag to fit it."
- Edit the SKILL.md under the writing-skills skill. RED is a headless arm curating a fixture skill note whose red_flags are over 1200 B. The arm either trims a valid entry or measures plain bytes. The edit is GREEN when the arm keeps them all.

### D3 (#776): when user detection is empty, keep the prior `user:` and warn (recommended; needs Joe's decision)

`amendIdentity` already returns an `*identityStamp`. Add a `stampPreservingUser` step:

- If the detected `User` is `""` and the note's current `user:` is non-empty, keep the current value.
- Emit `amend: user detection resolved empty; keeping user: <prior> on <basename>` through the existing `deps.LogWarning`.

This applies only on the amend path. `stamp` stays as it is for the other callers. `repo:` keeps its fresh-detection semantics, because an empty `repo:` is legitimate outside a git repo. For `vault:` there is no code change, only a test through the `amend` targ wiring (see Context).

Alternatives:

- (b) Refuse the amend with an error. This blocks content edits on a machine with broken identity.
- (c) Preserve silently. This hides a broken environment.

### D4 (#770): the payload defers to the skill (issue Option 1)

Replace `Instruction` with text that:

1. names "the learn skill's batch mode (Batch mode — Luhmann re-eval answers)" as the disposition procedure;
2. keeps the JSON shape verbatim;
3. says "exactly one entry per distinct candidate `note`";
4. says to hand the answers file back so the user or orchestrating agent runs `next_command` with `--dry-run` first, then without it, and that the answering pass does not run apply.

The text must be consistent with the spec's statement that dry-run is optional. It presents the preview as the recommended order and does not make it a gate. Append " (answers come from the learn skill's batch mode)" to `luhmannBranchingNotice`.

A new unit test pins these key phrases: `learn`, `batch mode`, `--dry-run`, `distinct`. It also asserts that the text does not say "then re-run". Leave `next_command`, the fingerprint and the gating unchanged.

Option 2 was rejected. It would edit the skill and remove the red flag in note 1067, which loses a human checkpoint before bulk renames. The GLOSSARY claim that the loop needs "no further manual step" (`docs/GLOSSARY.md:893-895`) is reworded to match.

### D5 (#780 a, b): pre-flight check inside `RenameAndRewriteReferences`

Before the loop at `luhmann_reparent.go:100`, add `preflightRenames(deps, vault, renameMap)`. For each source basename in the map it:

- reads the note;
- refuses it with `errRenameCRLF` if the note starts with `"---\r\n"`;
- otherwise computes `appendAliasField(rewriteLuhmannIDField(rewriteNoteReferences(raw)), …)`;
- decodes the result's frontmatter into a `yaml.Node`, and refuses with `errRenameUndecodable` on an error or a non-mapping;
- decodes `luhmann`, and refuses with `errRenameStaleLuhmann` if it is not the new id.

The pre-flight collects every refusal and returns one joined error before any `Rename` or `WriteFile`. Reparent apply and adopt both go through this function (`luhmann_reparent_apply.go:144`, `skillreg_accept.go:420`), so both are covered. Reparent `--dry-run` calls the same pre-flight and prints the refusals.

Alternatives:

- Making `rewriteLuhmannIDField` anchor-aware, for example rewriting to `luhmann: &a "new"`. That would silently change the value of every alias, so it is a semantic change.
- CRLF normalization (rewrite the file as LF). It changes bytes the user didn't ask to change. See Open Questions.
- Fixing `embed.SplitFrontmatter`. That is out of scope (Non-Goals).

The pre-flight is a pure function over injected `ReadFile`. Its rapid property is in D7.

### D6 (#780 c): adopt and refresh edit the frontmatter as a `yaml.Node`

Rewrite `applySkillNoteBody` to:

1. parse the frontmatter into a `yaml.Node`;
2. decode a typed `runbookFrontmatterDoc` from it, read-only, for `Supersedes` and validation;
3. call `setMappingValue` for `skill_hash`, `skill_key` and `skill_source`;
4. call `setMappingValue` for `pending: true`, or `deleteMappingKeys` for `pending` (omitempty parity);
5. encode the mapping.

Every other key keeps its value, and unknown keys survive. `pulldown.go:377-490,709` already uses this pattern. Move `setMappingValue`, `deleteMappingKeys`, `encodeNode` and `cloneNode` into a shared `frontmatter_node.go` so that two call paths don't import pulldown-specific code. Narrow both docstrings to the guarantee: "preserves every frontmatter key it doesn't set; comments are best-effort (`yaml.v3` round-trip)".

The alternative is to narrow only the docstrings. That leaves the data loss in place, and the spec now requires unknown keys to be preserved.

### D7: tests

All tests follow TDD, with the failing test written first, and use `t.Parallel()`. Each subtest builds its own fixture.

- Mocks for `RenameRewriteDeps`, `AmendDeps` and `ShowDeps` are imptest mocks where the tests drive the interaction. In-memory map fakes are used where only the final state is asserted.
- Assertions use gomega.
- The #780 property is `rapid.Check`. It generates a frontmatter mapping containing:
  - the required keys;
  - 0–3 unknown keys with scalar or list values;
  - optionally, an anchor on the `luhmann:` value with an alias from another key;
  - LF or CRLF line endings.
- It checks two properties:
  - **P1, rename:** either `RenameAndRewriteReferences` returns an error and the fake records zero renames and writes, or every renamed note decodes, its `luhmann` equals the new id, and every other top-level key except `aliases` decodes to the same value as before. An anchored or CRLF input always takes the error branch.
  - **P2, adopt/refresh:** for LF, anchor-free input, `applySkillNoteBody`'s output decodes, the skill fields are set, and every unknown key's decoded value equals the input's.

## Risks / Trade-offs

- [Risk] A large `engram show` output gets cut by the harness preview again, which is the risk from #763. → Mitigation: the harness keeps the full output in a persisted file, and show no longer removes entries itself. A future issue can reorder show to put red_flags earlier if it is measured to matter. This change does not reorder it.
- [Risk] The yaml.Node re-encode in adopt and refresh reformats quoting and whitespace for keys it doesn't touch. → Mitigation: the typed-struct path already re-marshalled the whole document, so formatting changes no more than it did before. P2 asserts decoded values, not bytes.
- [Risk] The pre-flight reads each renamed note twice, once in the pre-flight and once in the loop. → This is acceptable. Reparent and adopt are rare, one-shot operations.
- [Trade-off] A refused CRLF note blocks the whole reparent run. → The error names the note, so the user can convert it and run again.

## Migration Plan

No vault migration is needed. After the merge, run `go install ./cmd/engram` and `engram update` from the merged main checkout. Rollback is `git revert`, because no on-disk format changes.

## Open Questions

These need Joe's decision. Each has a recommended default, and the design above assumes it.

1. **#776 `user:` fallback.** The options are:
   - (a) keep the prior non-empty value and warn (**default**);
   - (b) refuse the amend with an error;
   - (c) keep the prior value silently.
2. **#772 option.** The options are:
   - A: show never truncates, and the query marker gives a count and a basename (**default**);
   - B: bounded show plus a `--red-flags` or `--full` retrieval flag.
3. **#770 canonical side.** The options are:
   - Option 1: the payload defers to learn batch mode, with `--dry-run` and the user applying (**default**);
   - Option 2: edit the skill and note 1067 so the agent may apply directly.
4. **#780(b) CRLF.** The options are:
   - refuse at the rename boundary (**default**);
   - normalize the renamed note to LF;
   - make `embed.SplitFrontmatter` CRLF-tolerant everywhere.
5. **`resituateTyped` drops unknown keys** (comment on #780). The options are:
   - leave it out of this change and file a follow-up issue (**default**);
   - fold it into D6, using the same node helpers.
