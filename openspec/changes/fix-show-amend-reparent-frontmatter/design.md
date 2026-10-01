## Context

This change fixes four open defects together, with Joe's approval of 2026-10-01. It also fixes the same-class `resituateTyped` key drop reported in a comment on #780. Joe ruled on every open question on 2026-10-01; see the decision record at the end. Each claim was checked against the code at `b48d6557` before any design work. The table gives the result for each.

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
| #780(b) follow-on | A CRLF *referrer* loses its frontmatter `supersedes:` rewrite | `rewriteNoteReferences` (`luhmann_reparent.go:318-330`) handles CRLF content as having no frontmatter. Only `[[...]]` wikilinks are rewritten; `rewriteSupersedesFrontmatterNotes` never runs. | **Found during verification**. It is fixed by the same conversion (D5). |
| #780(c) | Adopt and refresh drop unknown keys | `skillreg_accept.go:323-347`: plain `yaml.Unmarshal` into `runbookFrontmatterDoc` (`learn.go:~415-447`), then `marshalFrontmatter`. The docstrings at `:92` and `:172` claim that "every other frontmatter field" is preserved. | **Confirmed** |
| #780 comment | resituate drops unknown keys | `resituate.go:242-264`: `resituateTyped` decodes into the typed doc with plain `yaml.Unmarshal` and re-marshals with `marshalFrontmatter`. Keys the doc does not define are lost. | **Confirmed** |

The capabilities affected are `recall-runbook-surfacing`, `vault-offer-curation`, `vault-note-identity`, `update-reparent-luhmann-batch`, `update-flat-vault-luhmann-notice` and `skill-runbook-registration`.

## Goals / Non-Goals

**Goals:**
- `engram show` is a full-fidelity source of every red_flag. The query marker gives a count and a command that works.
- A re-stamping amend never blanks a non-empty `user:`.
- The reparent payload, the notice and the learn batch mode agree.
- No rename or rewrite writes undecodable frontmatter or leaves a stale `luhmann:`. Adopt, refresh and resituate keep keys they don't model. A CRLF note that a rename or rewrite writes comes out as LF, and its sidecar is rebuilt.

**Non-Goals:**
- General CRLF support in `embed.SplitFrontmatter`. It has more than 9 callers, including exchange-hash and vocab code. CRLF is converted only on notes that a rename or rewrite already writes.
- Converting CRLF notes that no write touches.
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

### D5 (#780 a, b): CRLF→LF conversion on written notes, and a pre-flight inside `RenameAndRewriteReferences`

**Conversion (Joe, 2026-10-01: convert, don't refuse).**

- **Scope.** `renameAndRewriteOneNote` first applies `toLF` to the raw bytes it reads: every `\r\n` becomes `\n`, and a lone `\r` is left as it is. All rewriting (references, `luhmann:`, aliases) then runs on the LF text. A note is written, and so converted, only when it would have been written anyway: it is renamed, or its references changed after conversion. A CRLF note the rename does not otherwise touch is never written and never converted.
- **Atomicity.** The converted bytes go in the same single `WriteFile` as the rewrite. `WriteFile` is wired to `FS.WriteFileAtomic` (`update.go:374-380`), so conversion adds no extra write and no window of its own. The existing order (rename the file, then write it) is unchanged.
- **CRLF referrers.** This also closes the follow-on gap where a CRLF referrer's `supersedes:` was never rewritten. After conversion the frontmatter splits and `rewriteSupersedesFrontmatterNotes` runs.

**Pre-flight.** Before the loop at `luhmann_reparent.go:100`, `preflightRenames(deps, vault, renameMap)` checks each source basename in the map:

1. Read the note and apply `toLF`.
2. Compute `appendAliasField(rewriteLuhmannIDField(rewriteNoteReferences(lf)), …)`.
3. Decode the result's frontmatter into a `yaml.Node`. Refuse with `errRenameUndecodable` on an error or a non-mapping (for example, a dangling anchor alias).
4. Decode `luhmann`, and refuse with `errRenameStaleLuhmann` if it is not the new id.

The pre-flight collects every refusal and returns one joined error before any `Rename` or `WriteFile`.

Reparent apply and adopt both go through this function (`luhmann_reparent_apply.go:144`, `skillreg_accept.go:420`), and reparent `--dry-run` runs the same pre-flight. Adopt also needs `toLF` earlier: `adoptRenderInput` applies it first, so that `checkAdoptRenders` and `applySkillNoteBody` see LF text. Adopt now converts a CRLF note instead of refusing it with `errSkillNoteNoFrontmatter`.

**Exchange hash.** Joe asked for the exchange hash to stay the same, citing ruling S6. Reading `exchangehash.go` shows that this is **only partly true**:

- `canonicalExchangeBody` (`exchangehash.go:48-62`) normalizes CRLF in the **body** only.
- The offered **fields** are read only when `embed.SplitFrontmatter` succeeds (`exchangehash.go:88-95`), and it never succeeds on a CRLF delimiter.

The effect depends on where the CRLF line endings are:

- **LF frontmatter, CRLF body:** conversion leaves the hash unchanged. A test pins `exchangeHash(pre) == exchangeHash(toLF(pre))`.
- **CRLF frontmatter:** before conversion, the hash is computed as if the note had no frontmatter. The fields are empty and the whole file is the body. Conversion therefore changes the hash. **This is a deviation from the stated requirement. It cannot be avoided without making `SplitFrontmatter` CRLF-tolerant, which is out of scope.** It is harmless, for two reasons:
  - Before conversion the note's `xid` cannot be read: `decodeExchangeFrontmatter` (`amend_fold.go:59-63`) returns `errAmendNoFrontmatter`. So no exchange path can have recorded a hash for the CRLF form.
  - The converted note hashes the same as its LF original, so a hash recorded before the file became CRLF matches again.

  Tests pin both: `decodeExchangeFrontmatter(crlf)` returns an error, and `exchangeHash(toLF(crlfNote)) == exchangeHash(lfNote)`.

**Sidecar freshness.** Conversion changes `embed.ContentHash` (`embed/hash.go:49-56`) for every converted note:

- **CRLF frontmatter:** `SituationText` was empty, and `BodyText` included the frontmatter.
- **CRLF body:** `BodyText` is not CRLF-normalized.

In both cases the stored vectors were built from the wrong inputs, or are now stale. So every converted note is added to the `rewritten` list that `RenameAndRewriteReferences` returns. Today a note that is only renamed is left out of that list (`luhmann_reparent.go:81-85`). The callers already pass the list to `RebuildNoteSidecars` (`luhmann_reparent_apply.go:149`, `skillreg_accept.go:548`), so converted notes are re-embedded in the same invocation. A test asserts that `embed.ComputeState` (`embed/state.go:25-58`) returns `StateOK` for every converted note after apply.

**Alternatives rejected:**

- Refusing CRLF. This was the earlier default, and Joe overruled it.
- Making `rewriteLuhmannIDField` anchor-aware, e.g. `luhmann: &a "new"`. This would silently change every alias's value.
- Making `embed.SplitFrontmatter` CRLF-tolerant. It is out of scope; see Non-Goals.

### D6 (#780 c): adopt and refresh edit the frontmatter as a `yaml.Node`

Rewrite `applySkillNoteBody` to:

1. parse the frontmatter into a `yaml.Node`;
2. decode a typed `runbookFrontmatterDoc` from it, read-only, for `Supersedes` and validation;
3. call `setMappingValue` for `skill_hash`, `skill_key` and `skill_source`;
4. call `setMappingValue` for `pending: true`, or `deleteMappingKeys` for `pending` (omitempty parity);
5. encode the mapping.

Every other key keeps its value, and unknown keys survive. `pulldown.go:377-490,709` already uses this pattern. Move `setMappingValue`, `deleteMappingKeys`, `encodeNode` and `cloneNode` into a shared `frontmatter_node.go` so that two call paths don't import pulldown-specific code. Narrow both docstrings to the guarantee: "preserves every frontmatter key it doesn't set; comments are best-effort (`yaml.v3` round-trip)".

The alternative is to narrow only the docstrings. That leaves the data loss in place, and the spec now requires unknown keys to be preserved.

### D6b (#780 comment): resituate edits the frontmatter as a `yaml.Node` (Joe, 2026-10-01: fold it in)

Rewrite `resituateTyped` (`resituate.go:242-264`):

1. Parse the frontmatter into a `yaml.Node`.
2. Decode the typed doc from it only to validate the note and to build the body opener. `parseCreated` keeps refusing malformed notes untouched.
3. Set `situation` with `setMappingValue`.
4. Encode the mapping.

Every other key survives, including keys the typed doc does not define. Update the function's docstring and the `vault-note-identity` requirement to match. resituate's re-embed and offer path is unchanged.

### D7: tests

All tests follow TDD, with the failing test written first, and use `t.Parallel()`. Each subtest builds its own fixture.

- Mocks for `RenameRewriteDeps`, `AmendDeps` and `ShowDeps` are imptest mocks where the tests drive the interaction. In-memory map fakes are used where only the final state is asserted.
- Assertions use gomega.
- The #780 property is `rapid.Check`. It generates a frontmatter mapping containing:
  - the required keys;
  - 0–3 unknown keys with scalar or list values;
  - optionally, an anchor on the `luhmann:` value with an alias from another key;
  - line endings that are all LF, all CRLF, CRLF only in the frontmatter, or CRLF only in the body (a separate CRLF-shape generator).
- It checks two properties:
  - **P1, rename:** either `RenameAndRewriteReferences` returns an error and the fake records zero renames and writes, or both of these hold:
    - every renamed note decodes, its `luhmann` equals the new id, and every other top-level key except `aliases` decodes to the same value as before;
    - every written note contains no `\r\n`.

    An anchored input always takes the error branch. A CRLF input never does on its own.
  - **P1-CRLF:**
    - Notes that are not written are byte-identical afterwards.
    - Every converted note appears in the returned `rewritten` list.
    - `exchangeHash(written) == exchangeHash(lfEquivalent)`.
    - For a note with an LF frontmatter and a CRLF body, `exchangeHash(written) == exchangeHash(original)`.
  - **P3, resituate:** for any generated note, every key except `situation` decodes to the same value afterwards, including unknown keys.
  - **P2, adopt/refresh:** for LF, anchor-free input, `applySkillNoteBody`'s output decodes, the skill fields are set, and every unknown key's decoded value equals the input's.

## Risks / Trade-offs

- [Risk] A large `engram show` output gets cut by the harness preview again, which is the risk from #763. → Mitigation: the harness keeps the full output in a persisted file, and show no longer removes entries itself. A future issue can reorder show to put red_flags earlier if it is measured to matter. This change does not reorder it.
- [Risk] The yaml.Node re-encode in adopt and refresh reformats quoting and whitespace for keys it doesn't touch. → Mitigation: the typed-struct path already re-marshalled the whole document, so formatting changes no more than it did before. P2 asserts decoded values, not bytes.
- [Risk] The pre-flight reads each renamed note twice, once in the pre-flight and once in the loop. → This is acceptable. Reparent and adopt are rare, one-shot operations.
- [Risk] Converting a CRLF note changes its exchange hash when its frontmatter was CRLF. → No recorded hash can name the CRLF form, since its xid is unreadable. The converted note matches the hash of its LF original. Both facts are pinned by tests (D5).
- [Risk] A converted note's vectors are rebuilt during apply, which adds embed time. → Only CRLF notes that are written are affected, and that is rare.

## Migration Plan

No vault migration is needed. After the merge, run `go install ./cmd/engram` and `engram update` from the merged main checkout. Rollback is `git revert`, because no on-disk format changes.

## Decision record (Joe, 2026-10-01)

1. **#776 `user:` fallback:** keep the prior non-empty `user:` and warn (D3).
2. **#772:** Option A. `show` never truncates. The query marker gives the count, the total and the real basename (D1).
3. **#770:** Option 1. The payload defers to learn's batch mode: preview with `--dry-run`, then apply (D4).
4. **#780(b) CRLF:** convert to LF, only on notes the rename or rewrite writes, inside the same atomic write. The sidecar is rebuilt. The exchange hash is unchanged for a CRLF body. It changes, harmlessly, for CRLF frontmatter (D5); this deviation is recorded above.
5. **`resituateTyped` drops unknown keys:** folded into this change with the same YAML-node fix (D6b).

No open questions remain.
